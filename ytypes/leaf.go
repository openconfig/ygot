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
	"github.com/openconfig/ygot/ygot"
)

// Refer to: https://tools.ietf.org/html/rfc6020#section-7.6.

// validateLeaf validates the value of a leaf struct against the given schema.
// This value is expected to be a Go basic type corresponding to the leaf
// schema type.
func validateLeaf(inSchema *yang.Entry, value interface{}) (errors []error) {
	// TODO(mostrowski): "mandatory" not implemented.
	if isNil(value) {
		return nil
	}

	dbgPrint("validateLeaf with value %s, schema name %s", valueStr(value), inSchema.Name)

	schema, err := resolveLeafRef(inSchema)
	if err != nil {
		return appendErr(errors, err)
	}

	var rv interface{}
	ykind := schema.Type.Kind
	rkind := reflect.ValueOf(value).Kind()
	switch rkind {
	case reflect.Ptr:
		rv = reflect.ValueOf(value).Elem().Interface()
	case reflect.Slice:
		if ykind != yang.Ybinary {
			return appendErr(errors, fmt.Errorf("bad leaf type: expect []byte for binary value %v for schema %s, have type %v",
				value, schema.Name, ykind))
		}
	case reflect.Int64:
		if ykind != yang.Yenum && ykind != yang.Yidentityref {
			return appendErr(errors, fmt.Errorf("bad leaf type: expect Int64 for enum type for schema %s, have type %v",
				schema.Name, ykind))
		}
	default:
		return appendErr(errors, fmt.Errorf("bad leaf value type %v, expect Ptr or Int64 for schema %s", rkind, schema.Name))
	}

	switch ykind {
	case yang.Ybinary:
		return appendErr(errors, validateBinary(schema, rv))
	case yang.Ybits:
		return nil
		// TODO(mostrowski): restore when representation is decided.
		//return appendErr(errors, validateBitset(schema, rv))
	case yang.Ybool:
		return appendErr(errors, validateBool(schema, rv))
	case yang.Ystring:
		return appendErr(errors, validateString(schema, rv))
	case yang.Ydecimal64:
		return appendErr(errors, validateDecimal(schema, rv))
	case yang.Yenum, yang.Yidentityref:
		if rkind != reflect.Int64 {
			return appendErr(errors, fmt.Errorf("bad leaf value type %v, expect Int64 for schema %s, type %v", rkind, schema.Name, ykind))
		}
		return nil
	case yang.Yunion:
		return validateUnion(schema, value)
	case yang.Yleafref:
		return validateLeafRef(schema, value)
	}
	if isIntegerType(ykind) {
		return appendErr(errors, validateInt(schema, rv))
	}
	return appendErr(errors, fmt.Errorf("unknown leaf type %v for schema %s", ykind, schema.Name))
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
func validateUnion(schema *yang.Entry, value interface{}) (errors []error) {
	dbgPrint("validateUnion %s", schema.Name)
	if isNil(value) {
		return nil
	}

	// Must be a ptr - either a struct ptr or Go value ptr like *string.
	// Enum types are also represented as a struct for union where the field
	// has the enum type.
	if reflect.TypeOf(value).Kind() != reflect.Ptr {
		return appendErr(errors, fmt.Errorf("wrong value type for union %s: got: %T, expect ptr", schema.Name, value))
	}

	elem := reflect.ValueOf(value).Elem()

	if elem.Type().Kind() == reflect.Struct {
		structElems := reflect.ValueOf(value).Elem()
		if structElems.NumField() != 1 {
			return appendErr(errors, fmt.Errorf("union %s should only have one field, but has %d", schema.Name, structElems.NumField()))
		}

		return validateMatchingSchemas(schema, structElems.Field(0).Interface())
	}

	return validateMatchingSchemas(schema, value)
}

// validateMatchingSchemas validates against all schemas within the Type slice
// that match the type of passed in value. It returns nil if value is
// successfully validated against any matching schema, or a list of errors found
// during validation against each matching schema otherwise.
func validateMatchingSchemas(schema *yang.Entry, value interface{}) (errors []error) {
	ss := findMatchingSchemasInUnion(schema.Type, value)
	dbgPrint("validateMatchingSchemas for %s: %v", schema.Name, ss)
	if len(ss) == 0 {
		return []error{fmt.Errorf("no types in schema %s match the type of value %v, which is %T", schema.Name, value, value)}
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
		errors = appendErrs(errors, errs)
	}

	return
}

// findMatchingSchemasInUnion returns all schemas in the given union type,
// including those within nested unions, that match the Go type of value.
// value must not be nil.
func findMatchingSchemasInUnion(ytype *yang.YangType, value interface{}) []*yang.Entry {
	var matches []*yang.Entry

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
			log.Warningf("no matching Go type for type %v in union value %s", t.Kind, valueStr(value))
			continue
		}
		if reflect.ValueOf(ybt).Type() == reflect.ValueOf(value).Type() {
			matches = append(matches, yangTypeToLeafEntry(t))
		}
	}

	return matches
}

// validateLeafRef validates a leaf-ref type. This type contains a path pointing
// to the actual type definition.
// TODO(mostrowski): In leaf-list case, handle checking that value exists in the
// referenced data tree node.
func validateLeafRef(schema *yang.Entry, value interface{}) (errors []error) {
	dbgPrint("validateLeafRef %s\n", schema.Name)
	refSchema, err := findLeafRefSchema(schema, schema.Type.Path)
	if err == nil {
		return validateLeaf(refSchema, value)
	}
	return appendErr(errors, err)
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
			if isFakeRoot(refSchema) {
				// In the fake root, if we have something at the root of the form /list/container and
				// schema compression is enabled, then we actually have only 'container' at the fake
				// root. So we need to check whether there is a child of the name of the subsequent
				// entry in the path element.
				pech, err := stripPrefix(path[i+1])
				if err != nil {
					return nil, err
				}
				if refSchema.Dir[pech] != nil {
					refSchema = refSchema.Dir[pech]
					// Skip this element.
					i++
					continue
				}
			}
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
	if isNil(value) {
		return nil
	}

	var err error
	dbgPrint("unmarshalLeaf value %v, type %T, into parent type %T, schema name %s", valueStr(value), value, parent, inSchema.Name)

	if err := validateLeafSchema(inSchema); err != nil {
		return err
	}

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

	return UpdateField(parent, fieldName, v)
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
	dbgPrint("unmarshalUnion value %v, type %T, into parent type %T field name %s, schema name %s", valueStr(value), value, parent, fieldName, schema.Name)
	v, t := reflect.ValueOf(parent), reflect.TypeOf(parent)
	if !IsTypeStructPtr(t) {
		return fmt.Errorf("%T is not a struct ptr in unmarshalUnion", parent)
	}

	vf := v.Elem().FieldByName(fieldName)
	if !vf.IsValid() {
		return fmt.Errorf("%s is not a valid field name in %T", fieldName, parent)
	}
	ft, err := getFieldElemType(parent, fieldName)
	if err != nil {
		return err
	}

	yks := getUnionKinds(schema.Type)
	dbgPrint("possible union types are %v", yks)

	// This can either be a interface, where multiple types are involved, of
	// just the type itself, if the alternatives span only one type.
	if !IsTypeInterface(ft) {
		// Is not an interface, we must have exactly one type in the union.
		if len(yks) != 1 {
			return fmt.Errorf("got %v types for union schema %s for type %T, expect just one type", yks, fieldName, parent)
		}
		yk := yks[0]
		goValue, err := unmarshalScalar(parent, yangKindToLeafEntry(yk), fieldName, value)
		if err != nil {
			return fmt.Errorf("could not unmarshal %v into type %s", value, yk)
		}
		vf.Set(reflect.ValueOf(ygot.ToPtr(goValue)))
		return nil
	}

	// The "to union" conversion method is called To_<field type name>
	mn := "To_" + ft.Name()
	mapMethod := reflect.New(t).Elem().MethodByName(mn)
	if !mapMethod.IsValid() {
		return fmt.Errorf("%s in %T does not have a %s function", fieldName, parent, mn)
	}

	// For each possible union type, try to unmarshal the JSON value. If it can
	// unmarshaled, try to resolve the resulting type into a union struct type.
	// Note that values can resolve into more than one struct type depending on
	// the value and its range. In this case, no attempt is made to find the
	// most restrictive type.
	for _, yk := range yks {
		goValue, err := unmarshalScalar(parent, yangKindToLeafEntry(yk), fieldName, value)
		if err != nil {
			dbgPrint("could not unmarshal %v into type %s", value, yk)
			continue
		}

		ec := mapMethod.Call([]reflect.Value{reflect.ValueOf(goValue)})
		if len(ec) != 2 {
			return fmt.Errorf("%s in %T %s function returns %d params", fieldName, parent, mn, len(ec))
		}
		ei := ec[0].Interface()
		ee := ec[1].Interface()
		if ee != nil {
			dbgPrint("unmarshaled %v type %T does not have a union type", goValue, goValue)
			continue
		}

		vf.Set(reflect.ValueOf(ei))
		return nil
	}

	return fmt.Errorf("could not find suitable union type to unmarshal value %v type %T into parent struct type %T field %s", value, value, parent, fieldName)
}

func getUnionKinds(t *yang.YangType) []yang.TypeKind {
	var out []yang.TypeKind
	m := make(map[yang.TypeKind]interface{})
	yts := getUnionTypes(t)
	for _, yt := range yts {
		m[yt.Kind] = nil
	}
	for k := range m {
		out = append(out, k)
	}
	return out
}

func getUnionTypes(t *yang.YangType) []*yang.YangType {
	var out []*yang.YangType
	if t.Kind != yang.Yunion {
		return []*yang.YangType{t}
	}
	for _, t := range t.Type {
		out = append(out, getUnionTypes(t)...)
	}
	return out
}

func getFieldElemType(parent interface{}, fieldName string) (reflect.Type, error) {
	t := reflect.TypeOf(parent)
	ft, ok := t.Elem().FieldByName(fieldName)
	if !ok {
		return reflect.TypeOf(nil), fmt.Errorf("%s is not a valid field name in %T", fieldName, parent)
	}
	switch {
	case IsTypeStructPtr(t):
		return ft.Type, nil
	case IsTypeSlicePtr(t):
		// Dereference slice ptr, then Elem() gives slice element type.
		return ft.Type.Elem().Elem(), nil
	}

	return reflect.TypeOf(nil), fmt.Errorf("%T is not a valid parent type", parent)
}

// unmarshalScalar unmarshals value, which is the Go type from json.Unmarshal,
// to the corresponding value used in gostructs.
//   parent is the parent struct containing the field being unmarshaled.
//     Required if the unmarshaled type is an enum.
//   fieldName is the name of the field being unmarshaled.
//     Required if the unmarshaled type is an enum.
func unmarshalScalar(parent interface{}, schema *yang.Entry, fieldName string, value interface{}) (interface{}, error) {
	if isNil(value) {
		return nil, nil
	}

	if err := validateLeafSchema(schema); err != nil {
		return nil, err
	}

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
		intV, err := enumStringToIntValue(parent, fieldName, value.(string))
		if err != nil {
			return nil, err
		}
		// Convert to destination enum type.
		v := reflect.ValueOf(intV)
		t, err := GetFieldType(parent, fieldName)
		if err != nil {
			return nil, err
		}
		return v.Convert(t).Interface(), nil

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
