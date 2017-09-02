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
	"fmt"
	"reflect"
	"strings"

	log "github.com/golang/glog"
	"github.com/openconfig/goyang/pkg/yang"
)

// Refer to: https://tools.ietf.org/html/rfc6020#section-7.6.

// validateLeaf validates the value of a leaf struct against the given schema.
// This value is expected to be a Go basic type corresponding to the leaf
// schema type.
func validateLeaf(schema *yang.Entry, value interface{}) (errors []error) {
	// TODO(mostrowski): "mandatory" not implemented.
	if isNil(value) {
		return nil
	}

	dbgPrint("validateLeaf with value %s, schema name %s", valueStr(value), schema.Name)

	var rv interface{}
	ykind := schema.Type.Kind
	rkind := reflect.ValueOf(value).Kind()
	switch rkind {
	case reflect.Ptr:
		rv = reflect.ValueOf(value).Elem().Interface()
	case reflect.Slice:
		if ykind != yang.Ybinary {
			return appendErr(errors, fmt.Errorf("bad leaf type: expect []byte for binary type for schema %s, have type %v",
				schema.Name, ykind))
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
	default:
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
// that match the type of passed in value. Returns nil if value is successfully
// validated against any matching schema, or a list of errors found during
// validation against each matching schema otherwise.
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
			ybt = yangBuiltinTypeToGoPtrType(t.Kind)
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
		return fmt.Errorf("leaf schema Type is nil for schema %s", schema.Name)
	}
	if schema.Kind != yang.LeafEntry {
		return fmt.Errorf("case schema has wrong type %v for schema %s", schema.Kind, schema.Name)
	}
	return nil
}
