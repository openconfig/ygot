// Copyright 2017 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ytypes

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	log "github.com/golang/glog"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/util"
	"github.com/openconfig/ygot/ygot"
)

// Refer to: https://tools.ietf.org/html/rfc6020#section-7.6.

// validateLeaf validates the value of a leaf struct against the given schema.
// This value is expected to be a Go basic type corresponding to the leaf
// schema type.
func validateLeaf(inSchema *yang.Entry, value interface{}) util.Errors {
	// TODO(mostrowski): "mandatory" not implemented.
	if util.IsValueNil(value) {
		return nil
	}

	// Check that the schema itself is valid.
	if err := validateLeafSchema(inSchema); err != nil {
		return util.NewErrs(err)
	}

	util.DbgPrint("validateLeaf with value %s (%T), schema name %s (%s)", util.ValueStr(value), value, inSchema.Name, inSchema.Type.Kind)

	schema, err := resolveLeafRef(inSchema)
	if err != nil {
		return util.NewErrs(err)
	}

	var rv interface{}
	ykind := schema.Type.Kind
	rkind := reflect.ValueOf(value).Kind()
	switch rkind {
	case reflect.Ptr:
		rv = reflect.ValueOf(value).Elem().Interface()
	case reflect.Slice:
		if ykind != yang.Ybinary {
			return util.NewErrs(fmt.Errorf("bad leaf type: expect []byte for binary value %v for schema %s, have type %v", value, schema.Name, ykind))
		}
	case reflect.Int64:
		if ykind != yang.Yenum && ykind != yang.Yidentityref {
			return util.NewErrs(fmt.Errorf("bad leaf type: expect Int64 for enum type for schema %s, have type %v", schema.Name, ykind))
		}
	case reflect.Bool:
		if ykind != yang.Yempty {
			return util.NewErrs(fmt.Errorf("bad leaf type: expect Bool for empty type for schema %s, have type %v", schema.Name, ykind))
		}
		rv = value
	default:
		return util.NewErrs(fmt.Errorf("bad leaf value type %v, expect Ptr or Int64 for schema %s", rkind, schema.Name))
	}

	switch ykind {
	case yang.Ybinary:
		return util.NewErrs(validateBinary(schema, rv))
	case yang.Ybits:
		return nil
		// TODO(mostrowski): restore when representation is decided.
		//return util.NewErrs(validateBitset(schema, rv))
	case yang.Ybool:
		return util.NewErrs(validateBool(schema, rv))
	case yang.Yempty:
		return util.NewErrs(validateEmpty(schema, rv))
	case yang.Ystring:
		return util.NewErrs(validateString(schema, rv))
	case yang.Ydecimal64:
		return util.NewErrs(validateDecimal(schema, rv))
	case yang.Yenum, yang.Yidentityref:
		if rkind != reflect.Int64 && !isValueInterfacePtrToEnum(reflect.ValueOf(value)) {
			return util.NewErrs(fmt.Errorf("bad leaf value type %v, expect Int64 for schema %s, type %v", rkind, schema.Name, ykind))
		}
		return nil
	case yang.Yunion:
		return validateUnion(schema, value)
	}
	if isIntegerType(ykind) {
		return util.NewErrs(validateInt(schema, rv))
	}
	return util.NewErrs(fmt.Errorf("unknown leaf type %v for schema %s", ykind, schema.Name))
}

/*
 validateUnion validates a union type and returns any validation errors.
 Unions have two types of possible representation in the data tree, which
 depends on the schema. The first case has alternatives with the same Go type,
 but different YANG types (possibly with different constraints):

 Name:        "address",
 Kind:        yang.Yleaf,
 Dir:         {},
 Type:        {
   Name:             "ip-address",
   Kind:             yang.Yunion,
   Type:             [
   {
           Name:             "ipv4-address",
           Kind:             yang.Ystring,
           Pattern:          [...pattern...],
   },
   {
           Name:             "ipv6-address",
           Kind:             yang.Ystring,
           Pattern:          [...pattern...],
           Type:             [],
   }]
 }

 In this case, the data tree will look like this:

 type System_Ntp_Server struct {
   Address *string `path:"address"`
 }

 The validation will check against all the schema nodes that match the YANG
 type corresponding to the Go type and return an error if none match.

 In the second case, where multiple Go types are present, the data tree has an
 additional struct layer. In this case, the struct field is compared against
 all YANG schemas that match the Go type of the selected wrapping struct e.g.

 Name:        "port",
 Kind:        yang.Yleafref,
 Dir:         {},
 Type:        {
   Name:             "port",
   Kind:             yang.Yunion,
   Type:             [
   {
           Name:             "port-string",
           Kind:             yang.Ystring,
           Pattern:          [...pattern...],
   },
   {
           Name:             "port-integer",
           Kind:             yang.Yuint16,
           Type:             [],
   }]
 }

 -- Corresponding structs data tree --

 type System_Ntp_Server struct {
   Port Port `path:"port"`
 }

 type Port interface {
   IsPort()
 }

 type Port_String struct {
   PortString *string
 }

 func (Port_String a) IsPort() {}

 type Port_Integer struct {
   PortInteger *uint16
 }

 func (Port_Integer a) IsPort() {}

 In this case, the appropriate schema is uniquely selected based on the struct
 path.

 A union may be nested. e.g. (shown as YANG schema for brevity)

 leaf foo {
   type union {
     type derived_string_type1;
     type union {
       type derived_string_type2;
       type derived_string_type3;
     }
   }
 }

 The data tree will look like this:

 type SomeContainer struct {
   Foo *string `path:"foo"`
 }

 In this case, the value for Foo would be recursively evaluated against any
 of the matching types in any contained unions.
 validateUnion supports any combination of nested union types and multiple
 choices with the same type that are not represented by a named wrapper struct.
*/
func validateUnion(schema *yang.Entry, value interface{}) util.Errors {
	if util.IsValueNil(value) {
		return nil
	}

	util.DbgPrint("validateUnion %s", schema.Name)
	// Must be a ptr - either a struct ptr or Go value ptr like *string.
	// Enum types are also represented as a struct for union where the field
	// has the enum type.
	if reflect.TypeOf(value).Kind() != reflect.Ptr {
		return util.NewErrs(fmt.Errorf("wrong value type for union %s: got: %T, expect ptr", schema.Name, value))
	}

	v := reflect.ValueOf(value).Elem()

	// Unions of enum types are passed as ptr to interface to struct ptr.
	// Normalize to a union struct.
	if util.IsValueInterface(v) {
		v = v.Elem()
		if util.IsValuePtr(v) {
			v = v.Elem()
		}
	}

	if v.Type().Kind() == reflect.Struct {
		if v.NumField() != 1 {
			return util.NewErrs(fmt.Errorf("union %s should only have one field, but has %d", schema.Name, v.NumField()))
		}
		return validateMatchingSchemas(schema, v.Field(0).Interface())
	}

	return validateMatchingSchemas(schema, value)
}

// validateMatchingSchemas validates against all schemas within the Type slice
// that match the type of passed in value. It returns nil if value is
// successfully validated against any matching schema, or a list of errors found
// during validation against each matching schema otherwise.
func validateMatchingSchemas(schema *yang.Entry, value interface{}) util.Errors {
	var errors []error
	ss := findMatchingSchemasInUnion(schema.Type, value)
	var kk []yang.TypeKind
	for _, s := range ss {
		kk = append(kk, s.Type.Kind)
	}
	util.DbgPrint("validateMatchingSchemas for value %v (%T) for schema %s with types %v", value, value, schema.Name, kk)
	if len(ss) == 0 {
		return util.NewErrs(fmt.Errorf("no types in schema %s match the type of value %v, which is %T", schema.Name, util.ValueStr(value), value))
	}
	for _, s := range ss {
		var errs []error
		if reflect.ValueOf(value).Kind() == reflect.Ptr {
			errs = validateLeaf(s, value)
		} else {
			// Unions with wrapping structs use non-ptr fields so here we need
			// to take the address of value to pass to validateLeaf, which
			// expects a ptr field.
			errs = validateLeaf(s, &value)
		}
		if errs == nil {
			return nil
		}
		errors = util.AppendErrs(errors, errs)
	}

	return errors
}

// findMatchingSchemasInUnion returns all schemas in the given union type,
// including those within nested unions, that match the Go type of value.
// value must not be nil.
func findMatchingSchemasInUnion(ytype *yang.YangType, value interface{}) []*yang.Entry {
	var matches []*yang.Entry

	util.DbgPrint("findMatchingSchemasInUnion for type %T, kind %s", value, reflect.TypeOf(value).Kind())
	for _, t := range ytype.Type {
		if t.Kind == yang.Yunion {
			// Recursively check all union types within this union.
			matches = append(matches, findMatchingSchemasInUnion(t, value)...)
			continue
		}

		ybt := yangBuiltinTypeToGoType(t.Kind)
		if reflect.ValueOf(value).Kind() == reflect.Ptr {
			ybt = ygot.ToPtr(yangBuiltinTypeToGoType(t.Kind))
		}
		if ybt == nil {
			log.Warningf("no matching Go type for type %v in union value %s", t.Kind, util.ValueStr(value))
			continue
		}
		if reflect.TypeOf(ybt).Kind() == reflect.TypeOf(value).Kind() {
			matches = append(matches, yangTypeToLeafEntry(t))
		}
	}

	return matches
}

// stripPrefix removes the prefix from a YANG path element. For example, removing
// foo from "foo:bar". Such qualified paths are used in YANG modules where remote
// paths are referenced.
func stripPrefix(name string) (string, error) {
	ps := strings.Split(name, ":")
	switch len(ps) {
	case 1:
		return name, nil
	case 2:
		return ps[1], nil
	}
	return "", fmt.Errorf("path element did not form a valid name (name, prefix:name): %v", name)
}

// removeXPATHPredicates removes predicates from an XPath string. e.g.,
// removeXPATHPredicates(/foo/bar[name="foo"]/config/baz -> /foo/bar/config/baz.
func removeXPATHPredicates(s string) (string, error) {
	var b bytes.Buffer
	for i := 0; i < len(s); {
		ss := s[i:]
		si, ei := strings.Index(ss, "["), strings.Index(ss, "]")
		switch {
		case si == -1 && ei == -1:
			// This substring didn't contain a [] pair, therefore write it
			// to the buffer.
			b.WriteString(ss)
			// Move to the last character of the substring.
			i += len(ss)
		case si == -1 || ei == -1:
			// This substring contained a mismatched pair of []s.
			return "", fmt.Errorf("Mismatched brackets within substring %s of %s, [ pos: %d, ] pos: %d", ss, s, si, ei)
		case si > ei:
			// This substring contained a ] before a [.
			return "", fmt.Errorf("Incorrect ordering of [] within substring %s of %s, [ pos: %d, ] pos: %d", ss, s, si, ei)
		default:
			// This substring contained a matched set of []s.
			b.WriteString(ss[0:si])
			i += ei + 1
		}
	}

	return b.String(), nil
}

// findLeafRefSchema returns a schema Entry at the path pathStr relative to
// schema if it exists, or an error otherwise.
// pathStr has either:
//  - the relative form "../a/b/../b/c", where ".." indicates the parent of the
//    node, or
//  - the absolute form "/a/b/c", which indicates the absolute path from the
//    root of the schema tree.
func findLeafRefSchema(schema *yang.Entry, pathStr string) (*yang.Entry, error) {
	if pathStr == "" {
		return nil, fmt.Errorf("leafref schema %s has empty path", schema.Name)
	}

	refSchema := schema
	pathStr, err := removeXPATHPredicates(pathStr)
	if err != nil {
		return nil, err
	}
	path := strings.Split(pathStr, "/")

	// For absolute path, reset to root of the schema tree.
	if pathStr[0] == '/' {
		refSchema = schemaTreeRoot(schema)
		path = path[1:]
	}

	for i := 0; i < len(path); i++ {
		pe, err := stripPrefix(path[i])
		if err != nil {
			return nil, fmt.Errorf("leafref schema %s path %s: %v", schema.Name, pathStr, err)
		}

		if pe == ".." {
			if refSchema.Parent == nil {
				return nil, fmt.Errorf("parent of %s is nil for leafref schema %s with path %s", refSchema.Name, schema.Name, pathStr)
			}
			refSchema = refSchema.Parent
			continue
		}
		if refSchema.Dir[pe] == nil {
			return nil, fmt.Errorf("schema node %s is nil for leafref schema %s with path %s", pe, schema.Name, pathStr)
		}
		refSchema = refSchema.Dir[pe]
	}

	return refSchema, nil
}

// validateLeafSchema validates the given leaf type schema. This is a sanity
// check validation rather than a comprehensive validation against the RFC.
// It is assumed that such a validation is done when the schema is parsed from
// source YANG.
func validateLeafSchema(schema *yang.Entry) error {
	if schema == nil {
		return fmt.Errorf("leaf schema is nil")
	}
	if schema.Type == nil {
		return fmt.Errorf("leaf schema type is nil for schema %s", schema.Name)
	}
	if schema.Kind != yang.LeafEntry {
		return fmt.Errorf("case schema has wrong type %v for schema %s", schema.Kind, schema.Name)
	}
	return nil
}

// unmarshalLeaf unmarshals a scalar value (determined by json.Unmarshal) into
// the parent containing the leaf.
//   schema points to the schema for the leaf type.
func unmarshalLeaf(inSchema *yang.Entry, parent interface{}, value interface{}) error {
	if util.IsValueNil(value) {
		return nil
	}

	var err error
	if err := validateLeafSchema(inSchema); err != nil {
		return err
	}

	util.DbgPrint("unmarshalLeaf value %v, type %T, into parent type %T, schema name %s", util.ValueStr(value), value, parent, inSchema.Name)

	fieldName, _, err := schemaToStructFieldName(inSchema, parent)
	if err != nil {
		return err
	}

	schema, err := resolveLeafRef(inSchema)
	if err != nil {
		return err
	}

	ykind := schema.Type.Kind

	if ykind == yang.Yunion {
		return unmarshalUnion(schema, parent, fieldName, value)
	}

	if reflect.ValueOf(value).Type() != yangToJSONType(ykind) {
		return fmt.Errorf("got %T type for field %s, expect %v", value, schema.Name, yangToJSONType(ykind).Kind())
	}

	if ykind == yang.Ybits {
		// TODO(mostrowski)
		return nil
	}

	v, err := unmarshalScalar(parent, schema, fieldName, value)
	if err != nil {
		return err
	}
	if ykind == yang.Ybinary {
		// Binary is a slice field which is treated as a scalar.
		return util.InsertIntoStruct(parent, fieldName, v)
	}
	return util.UpdateField(parent, fieldName, v)
}

// unmarshalUnion unmarshals a union schema type with the given value into
// parent.
/*
for example, with structs schema:

type Bgp_Neighbor_RouteReflector struct {
	RouteReflectorClient    *bool                                                     `path:"config/route-reflector-client" module:"openconfig-bgp"`
	RouteReflectorClusterId Bgp_Neighbor_RouteReflector_RouteReflectorClusterId_Union `path:"config/route-reflector-cluster-id" module:"openconfig-bgp"`
}
type Bgp_Neighbor_RouteReflector_RouteReflectorClusterId_Union interface {
	Is_Bgp_Neighbor_RouteReflector_RouteReflectorClusterId_Union()
}
type Bgp_Neighbor_RouteReflector_RouteReflectorClusterId_Union_String struct {
	String string
}
func (t *Bgp_Neighbor_RouteReflector) To_Bgp_Neighbor_RouteReflector_RouteReflectorClusterId_Union(i interface{}) (Bgp_Neighbor_RouteReflector_RouteReflectorClusterId_Union, error) {

and input JSON:

{"config/route-reflector-cluster-id": "forty-two"}

the resulting Bgp_Neighbor_RouteReflector would have field
RouteReflectorClusterId set with the type Bgp_Neighbor_RouteReflector_RouteReflectorClusterId_Union_String,
with field String set to "forty-two".
*/

func unmarshalUnion(schema *yang.Entry, parent interface{}, fieldName string, value interface{}) error {
	util.DbgPrint("unmarshalUnion value %v, type %T, into parent type %T field name %s, schema name %s", util.ValueStr(value), value, parent, fieldName, schema.Name)
	parentV, parentT := reflect.ValueOf(parent), reflect.TypeOf(parent)
	if !util.IsTypeStructPtr(parentT) {
		return fmt.Errorf("%T is not a struct ptr in unmarshalUnion", parent)
	}

	// Get the value and type of the field to set, which may have slice or
	// interface types for leaf-list and union cases.
	destUnionFieldV := parentV.Elem().FieldByName(fieldName)
	if !destUnionFieldV.IsValid() {
		return fmt.Errorf("%s is not a valid field name in %T", fieldName, parent)
	}
	dft, _ := parentT.Elem().FieldByName(fieldName)
	destUnionFieldElemT := dft.Type

	// Possible enum types, as []reflect.Type
	ets, err := schemaToEnumTypes(schema, parentT)
	if err != nil {
		return err
	}
	// Possible YANG scalar types, as []yang.TypeKind. This discards any
	// yang.Type restrictions, since these are expected to be checked during
	// verification after unmarshal.
	sks, err := getUnionKindsNotEnums(schema)
	if err != nil {
		return err
	}

	util.DbgPrint("possible union types are enums %v or scalars %v", ets, sks)

	// Special case. If all possible union types map to a single go type, the
	// GoStruct field is that type rather than a union Interface type.
	if !util.IsTypeInterface(destUnionFieldElemT) && !util.IsTypeSliceOfInterface(destUnionFieldElemT) {
		// Is not an interface, we must have exactly one type in the union.
		if len(sks) != 1 {
			return fmt.Errorf("got %v types for union schema %s for type %T, expect just one type", sks, fieldName, parent)
		}
		yk := sks[0]
		goValue, err := unmarshalScalar(parent, yangKindToLeafEntry(yk), fieldName, value)
		if err != nil {
			return fmt.Errorf("could not unmarshal %v into type %s", value, yk)
		}
		destUnionFieldV.Set(reflect.ValueOf(ygot.ToPtr(goValue)))
		return nil
	}

	// For each possible union type, try to unmarshal the JSON value. If it can
	// unmarshaled, try to resolve the resulting type into a union struct type.
	// Note that values can resolve into more than one struct type depending on
	// the value and its range. In this case, no attempt is made to find the
	// most restrictive type.
	// Try to unmarshal to enum types first, since the case of union of string
	// and enum could unmarshal into either. Only string values can be enum
	// types.
	valueStr, ok := value.(string)
	if ok {
		for _, et := range ets {
			util.DbgPrint("try to unmarshal into enum type %s", et)
			ev, err := castToEnumValue(et, valueStr)
			if err != nil {
				return err
			}
			if ev != nil {
				return setFieldWithTypedValue(parentT, destUnionFieldV, destUnionFieldElemT, ev)
			}
			util.DbgPrint("could not unmarshal %v into enum type: %s", value, err)
		}
	}

	for _, sk := range sks {
		util.DbgPrint("try to unmarshal into type %s", sk)
		sch := yangKindToLeafEntry(sk)
		gv, err := unmarshalScalar(parent, sch, fieldName, value)
		if err == nil {
			return setFieldWithTypedValue(parentT, destUnionFieldV, destUnionFieldElemT, gv)
		}
		util.DbgPrint("could not unmarshal %v into type %s: %s", value, sk, err)
	}

	return fmt.Errorf("could not find suitable union type to unmarshal value %v type %T into parent struct type %T field %s", value, value, parent, fieldName)
}

// setFieldWithTypedValue sets the field destV that has type ft and the given
// parent type with v, which must be a compatible enum type.
func setFieldWithTypedValue(parentT reflect.Type, destV reflect.Value, destElemT reflect.Type, v interface{}) error {
	util.DbgPrint("setFieldWithTypedValue value %v into type %s", util.ValueStr(v), destElemT)
	if destElemT.Kind() == reflect.Slice {
		// leaf-list case
		destElemT = destElemT.Elem()
	}
	mn := "To_" + destElemT.Name()
	mapMethod := reflect.New(parentT).Elem().MethodByName(mn)
	if !mapMethod.IsValid() {
		return fmt.Errorf("%s does not have a %s function", destElemT.Name(), mn)
	}
	ec := mapMethod.Call([]reflect.Value{reflect.ValueOf(v)})
	if len(ec) != 2 {
		return fmt.Errorf("%s %s function returns %d params", destElemT.Name(), mn, len(ec))
	}
	ei := ec[0].Interface()
	ee := ec[1].Interface()
	if ee != nil {
		return fmt.Errorf("unmarshaled %v type %T does not have a union type: %v", v, v, ee)
	}

	util.DbgPrint("unmarshaling %v into type %s", v, reflect.TypeOf(ei))

	eiv := reflect.ValueOf(ei)
	if destV.Type().Kind() == reflect.Slice {
		destV.Set(reflect.Append(destV, eiv))
	} else {
		destV.Set(eiv)
	}

	return nil
}

// getUnionKindsNotEnums returns all the YANG kinds under the given schema node,
// dereferencing any refs.
func getUnionKindsNotEnums(schema *yang.Entry) ([]yang.TypeKind, error) {
	var uks []yang.TypeKind
	m := make(map[yang.TypeKind]interface{})
	uts, err := getUnionTypesNotEnums(schema, schema.Type)
	if err != nil {
		return nil, err
	}
	for _, yt := range uts {
		m[yt.Kind] = nil
	}
	for k := range m {
		uks = append(uks, k)
	}
	return uks, nil
}

// getUnionTypesNotEnums returns all the non-enum YANG types under the given
// schema node, dereferencing any refs.
func getUnionTypesNotEnums(schema *yang.Entry, yt *yang.YangType) ([]*yang.YangType, error) {
	var uts []*yang.YangType
	switch yt.Kind {
	case yang.Yenum, yang.Yidentityref:
		// Enum types handled separately.
		return nil, nil
	case yang.Yleafref:
		ns, err := findLeafRefSchema(schema, yt.Path)
		if err != nil {
			return nil, err
		}
		return getUnionTypesNotEnums(ns, ns.Type)
	case yang.Yunion:
		for _, t := range yt.Type {
			nt, err := getUnionTypesNotEnums(schema, t)
			if err != nil {
				return nil, err
			}
			uts = append(uts, nt...)
			if err != nil {
				return nil, err
			}
		}
	default:
		uts = []*yang.YangType{yt}
	}

	return uts, nil
}

// schemaToEnumTypes returns the actual enum types (rather than the interface
// type) for a given schema, which must be for an enum type. t is the type of
// the containing parent struct.
func schemaToEnumTypes(schema *yang.Entry, t reflect.Type) ([]reflect.Type, error) {
	enumTypesMethod := reflect.New(t).Elem().MethodByName("ΛEnumTypeMap")
	if !enumTypesMethod.IsValid() {
		return nil, fmt.Errorf("type %s does not have a ΛEnumTypesMap function", t)
	}

	ec := enumTypesMethod.Call(nil)
	if len(ec) == 0 {
		return nil, fmt.Errorf("%s ΛEnumTypes function returns empty value", t)
	}
	ei := ec[0].Interface()
	enumTypesMap, ok := ei.(map[string][]reflect.Type)
	if !ok {
		return nil, fmt.Errorf("%s ΛEnumTypes function returned wrong type %T, want map[string][]reflect.Type", t, ei)
	}

	util.DbgPrint("path is %s for schema %s", absoluteSchemaDataPath(schema), schema.Name)

	return enumTypesMap[absoluteSchemaDataPath(schema)], nil
}

// unmarshalScalar unmarshals value, which is the Go type from json.Unmarshal,
// to the corresponding value used in gostructs.
//   parent is the parent struct containing the field being unmarshaled.
//     Required if the unmarshaled type is an enum.
//   fieldName is the name of the field being unmarshaled.
//     Required if the unmarshaled type is an enum.
func unmarshalScalar(parent interface{}, schema *yang.Entry, fieldName string, value interface{}) (interface{}, error) {
	if util.IsValueNil(value) {
		return nil, nil
	}

	if err := validateLeafSchema(schema); err != nil {
		return nil, err
	}

	util.DbgPrint("unmarshalScalar value %v, type %T, into parent type %T field %s", value, value, parent, fieldName)

	ykind := schema.Type.Kind

	if ykind != yang.Yunion && reflect.ValueOf(value).Type() != yangToJSONType(ykind) {
		return nil, fmt.Errorf("unmarshalScalar got %T type for field %s, expect %T", value, schema.Name, yangToJSONType(ykind))
	}

	switch ykind {
	case yang.Ybinary:
		v, err := base64.StdEncoding.DecodeString(value.(string))
		if err != nil {
			return nil, fmt.Errorf("error in DecodeString for \n%v\n for schema %s: %v", value, schema.Name, err)
		}
		return []byte(v), nil

	case yang.Ybits:
		// TODO(mostrowski)
		return nil, nil

	case yang.Ybool:
		return value.(bool), nil

	case yang.Ystring:
		return value.(string), nil

	case yang.Ydecimal64:
		floatV, err := strconv.ParseFloat(value.(string), 64)
		if err != nil {
			return nil, fmt.Errorf("error parsing %v for schema %s: %v", value, schema.Name, err)
		}

		return floatV, nil

	case yang.Yenum, yang.Yidentityref:
		return enumStringToValue(parent, fieldName, value.(string))

	case yang.Yint64:
		// TODO(b/64812268): value types are different for internal style JSON.
		intV, err := strconv.ParseInt(value.(string), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("error parsing %v for schema %s: %v", value, schema.Name, err)
		}
		return intV, nil

	case yang.Yuint64:
		uintV, err := strconv.ParseUint(value.(string), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("error parsing %v for schema %s: %v", value, schema.Name, err)
		}
		return uintV, nil

	case yang.Yint8, yang.Yint16, yang.Yint32:
		pv, err := yangFloatIntToGoType(ykind, value.(float64))
		if err != nil {
			return nil, fmt.Errorf("error parsing %v for schema %s: %v", value, schema.Name, err)
		}
		return pv, nil

	case yang.Yuint8, yang.Yuint16, yang.Yuint32:
		pv, err := yangFloatIntToGoType(ykind, value.(float64))
		if err != nil {
			return nil, fmt.Errorf("error parsing %v for schema %s: %v", value, schema.Name, err)
		}
		return pv, nil

	case yang.Yunion:
		return value, nil

	}

	return nil, fmt.Errorf("unmarshalScalar: unsupported type %v in schema node %s", ykind, schema.Name)
}

// isValueInterfacePtrToEnum reports whether v is an interface ptr to enum type.
func isValueInterfacePtrToEnum(v reflect.Value) bool {
	if v.Kind() != reflect.Ptr {
		return false
	}
	v = v.Elem()
	if v.Kind() != reflect.Interface {
		return false
	}
	v = v.Elem()

	return v.Kind() == reflect.Int64
}
