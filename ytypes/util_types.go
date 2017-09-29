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
	"fmt"
	"reflect"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/util"
	"github.com/openconfig/ygot/ygen"
	"github.com/openconfig/ygot/ygot"

	log "github.com/golang/glog"
)

// enumStringToUnionStructValue returns a struct ptr with a field populated with
// the correct enum type and value, if a matching value can be found for the
// given value string.
func enumStringToUnionStructValue(schema *yang.Entry, parent interface{}, fieldName, value string) (interface{}, error) {
	util.DbgPrint("enumStringToUnionStructValue with schema %s, parent type %T, fieldName %s, value %s", schema.Name, parent, fieldName, value)
	v := reflect.ValueOf(parent)
	if !util.IsValueStructPtr(v) {
		return 0, fmt.Errorf("enumStringToIntValue: %T is not a struct ptr", parent)
	}

	field := v.Elem().FieldByName(fieldName)
	if !field.IsValid() {
		return 0, fmt.Errorf("%s is not a valid enum field name in %T", fieldName, parent)
	}

	if !util.IsTypeInterface(field.Type()) {
		return 0, fmt.Errorf("field %s is has type %s, expect interface", fieldName, field.Type())
	}

	fts, err := schemaToEnumTypes(schema, reflect.TypeOf(parent))
	if err != nil {
		return 0, err
	}
	if len(fts) == 0 {
		return 0, fmt.Errorf("enumStringToIntValue: schemaToEnumTypes returned null")
	}

	for _, ft := range fts {
		// Look for a match for the enum field, which is the only field in the
		// wrapping struct.
		ei, err := findMatchingEnumType(parent, ft.Field(0).Type, value)
		if err != nil {
			return 0, err
		}
		if ei != nil {
			// Return the struct ptr of the wrapping struct.
			nv := reflect.New(ft)
			nv.Elem().Field(0).Set(reflect.ValueOf(ei))
			return nv.Interface(), nil
		}
	}
	return 0, fmt.Errorf("%s is not a valid value for enum field %s", value, fieldName)
}

// enumStringToValue returns the enum type value that enumerated string value
// of type fieldName maps to in the parent, which must be a struct ptr.
func enumStringToValue(schema *yang.Entry, parent interface{}, fieldName, value string) (interface{}, error) {
	util.DbgPrint("enumStringToIntValue with schema %s, parent type %T, fieldName %s, value %s", schema.Name, parent, fieldName, value)
	v := reflect.ValueOf(parent)
	if !util.IsValueStructPtr(v) {
		return 0, fmt.Errorf("enumStringToIntValue: %T is not a struct ptr", parent)
	}
	field := v.Elem().FieldByName(fieldName)
	if !field.IsValid() {
		return 0, fmt.Errorf("%s is not a valid enum field name in %T", fieldName, parent)
	}

	return findMatchingEnumType(parent, field.Type(), value)
}

// findMatchingEnumType returns value as the given type ft, if value is one of
// the allowed values of ft, or an error otherwise.
func findMatchingEnumType(parent interface{}, ft reflect.Type, value string) (interface{}, error) {
	if ft.Kind() == reflect.Slice {
		// leaf-list case
		ft = ft.Elem()
	}

	util.DbgPrint("checking for matching enum value for type %s", ft)
	mapMethod := reflect.New(ft).MethodByName("ΛMap")
	if !mapMethod.IsValid() {
		return 0, fmt.Errorf("%s in %T does not have a ΛMap function", ft, parent)
	}

	ec := mapMethod.Call(nil)
	if len(ec) == 0 {
		return 0, fmt.Errorf("%s ΛMap function returns empty value", ft, parent)
	}
	ei := ec[0].Interface()
	enumMap, ok := ei.(map[string]map[int64]ygot.EnumDefinition)
	if !ok {
		return 0, fmt.Errorf("%s in %T ΛMap function returned wrong type %T, want map[string]map[int64]ygot.EnumDefinition", ft, parent, ei)
	}

	m, ok := enumMap[ft.Name()]
	if !ok {
		return 0, fmt.Errorf("%s is not a valid enum field name", ft.Name())
	}

	for k, v := range m {
		if stripModulePrefix(v.Name) == stripModulePrefix(value) {
			// Convert to destination enum type.
			return reflect.ValueOf(k).Convert(ft).Interface(), nil
		}
	}

	return 0, fmt.Errorf("%s is not a valid value for enum field %s", value, ft)
}

// schemaToEnumTypes returns the actual enum types (rather than the interface
// type) for a given schema, which must be for an enum type. 
func schemaToEnumTypes(schema *yang.Entry, ft reflect.Type) ([]reflect.Type, error) {
	enumTypesMethod := reflect.New(ft).Elem().MethodByName("ΛEnumTypeMap")
	if !enumTypesMethod.IsValid() {
		return nil, fmt.Errorf("type %s does not have a ΛEnumTypesMap function", ft)
	}

	ec := enumTypesMethod.Call(nil)
	if len(ec) == 0 {
		return nil, fmt.Errorf("%s ΛEnumTypes function returns empty value", ft)
	}
	ei := ec[0].Interface()
	enumTypesMap, ok := ei.(map[string][]reflect.Type)
	if !ok {
		return nil, fmt.Errorf("%s ΛEnumTypes function returned wrong type %T, want map[string][]reflect.Type", ft, ei)
	}
	util.DbgPrint("path is %s for schema %s", ygen.EntrySchemaPath(schema), schema.Name)

	return enumTypesMap[ygen.EntrySchemaPath(schema)], nil
}

// yangBuiltinTypeToGoType returns a pointer to the Go built-in value with
// the type corresponding to the provided YANG type. It returns nil for any type
// which is not an integer, float, string, boolean, or binary kind.
func yangBuiltinTypeToGoType(t yang.TypeKind) interface{} {
	switch t {
	case yang.Yint8:
		return int8(0)
	case yang.Yint16:
		return int16(0)
	case yang.Yint32:
		return int32(0)
	case yang.Yint64:
		return int64(0)
	case yang.Yuint8:
		return uint8(0)
	case yang.Yuint16:
		return uint16(0)
	case yang.Yuint32:
		return uint32(0)
	case yang.Yuint64:
		return uint64(0)
	case yang.Ybool, yang.Yempty:
		return bool(false)
	case yang.Ystring:
		return string("")
	case yang.Ydecimal64:
		return float64(0)
	case yang.Ybinary:
		return []byte(nil)
	case yang.Yenum, yang.Yidentityref:
		return int64(0)
	default:
		// TODO(mostrowski): handle bitset.
		log.Errorf("unexpected type %v in yangBuiltinTypeToGoPtrType", t)
	}
	return nil
}

// yangToJSONType returns the Go type that json.Unmarshal would render a value
// into for the given YANG leaf node schema type.
func yangToJSONType(t yang.TypeKind) reflect.Type {
	switch t {
	case yang.Yint8, yang.Yint16, yang.Yint32,
		yang.Yuint8, yang.Yuint16, yang.Yuint32:
		return reflect.TypeOf(float64(0))
	case yang.Ybinary, yang.Ydecimal64, yang.Yenum, yang.Yidentityref, yang.Yint64, yang.Yuint64, yang.Ystring:
		return reflect.TypeOf(string(""))
	case yang.Ybool, yang.Yempty:
		return reflect.TypeOf(bool(false))
	case yang.Yunion:
		return reflect.TypeOf(nil)
	default:
		// TODO(mostrowski): handle bitset.
		log.Errorf("unexpected type %v in yangToJSONType", t)
	}
	return reflect.TypeOf(nil)
}

// yangFloatIntToGoType the appropriate int type for YANG type t, set with value
// v, which must be within the allowed range for the type.
func yangFloatIntToGoType(t yang.TypeKind, v float64) (interface{}, error) {
	if err := checkJSONFloat64Range(t, v); err != nil {
		return nil, err
	}

	switch t {
	case yang.Yint8:
		return int8(v), nil
	case yang.Yint16:
		return int16(v), nil
	case yang.Yint32:
		return int32(v), nil
	case yang.Yuint8:
		return uint8(v), nil
	case yang.Yuint16:
		return uint16(v), nil
	case yang.Yuint32:
		return uint32(v), nil
	}
	return nil, fmt.Errorf("unexpected YANG type %v", t)
}

// checkJSONFloat64Range checks whether f is in range for the given YANG type.
func checkJSONFloat64Range(t yang.TypeKind, f float64) error {
	minMax := map[yang.TypeKind]struct {
		min int64
		max int64
	}{
		yang.Yint8:   {-128, 127},
		yang.Yint16:  {-32768, 32767},
		yang.Yint32:  {-2147483648, 2147483647},
		yang.Yuint8:  {0, 255},
		yang.Yuint16: {0, 65535},
		yang.Yuint32: {0, 4294967295},
	}

	if _, ok := minMax[t]; !ok {
		return fmt.Errorf("checkJSONFloat64Range bad YANG type %v", t)
	}
	if int64(f) < minMax[t].min || int64(f) > minMax[t].max {
		return fmt.Errorf("value %d falls outside the int range [%d, %d]", int64(f), minMax[t].min, minMax[t].max)
	}
	return nil
}

// yangTypeToLeafEntry returns a leaf Entry with Type set to t.
func yangTypeToLeafEntry(t *yang.YangType) *yang.Entry {
	return &yang.Entry{
		Kind: yang.LeafEntry,
		Type: t,
	}
}

// yangKindToLeafEntry returns a leaf Entry with Type Kind set to k.
func yangKindToLeafEntry(k yang.TypeKind) *yang.Entry {
	return &yang.Entry{
		Kind: yang.LeafEntry,
		Type: &yang.YangType{
			Kind: k,
		},
	}
}
