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
	"encoding/base64"
	"fmt"
	"reflect"
	"strconv"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/util"
	"github.com/openconfig/ygot/ygot"

	log "github.com/golang/glog"
)

// enumStringToValue returns the enum type value that enumerated string value
// of type fieldName maps to in the parent, which must be a struct ptr.
func enumStringToValue(parent interface{}, fieldName, value string) (interface{}, error) {
	util.DbgPrint("enumStringToValue with parent type %T, fieldName %s, value %s", parent, fieldName, value)
	v := reflect.ValueOf(parent)
	if !util.IsValueStructPtr(v) {
		return 0, fmt.Errorf("enumStringToIntValue: %T is not a struct ptr", parent)
	}
	field := v.Elem().FieldByName(fieldName)
	if !field.IsValid() {
		return 0, fmt.Errorf("%s is not a valid enum field name in %T", fieldName, parent)
	}

	ev, err := castToEnumValue(field.Type(), value)
	if err != nil {
		return nil, err
	}
	if ev == nil {
		return 0, fmt.Errorf("%s is not a valid value for enum field %s, type %s", value, fieldName, field.Type())
	}
	return ev, nil
}

// enumAndNonEnumTypesForUnion returns the list of enum and non-enum types for
// a given union leaf's schema, provided a parent context.
func enumAndNonEnumTypesForUnion(schema *yang.Entry, parentT reflect.Type) ([]reflect.Type, []yang.TypeKind, error) {
	// Possible enum types, as []reflect.Type
	ets, err := schemaToEnumTypes(schema, parentT)
	if err != nil {
		return nil, nil, err
	}
	// Possible YANG scalar types, as []yang.TypeKind. This discards any
	// yang.Type restrictions, since these are expected to be checked during
	// verification after unmarshal.
	sks, err := getUnionKindsNotEnums(schema)
	if err != nil {
		return nil, nil, err
	}

	util.DbgPrint("enumAndNonEnumTypesForUnion: possible union types are enums %v or scalars %v", ets, sks)
	return ets, sks, nil
}

// getLoneUnionType checks whether the provided union type was created from a
// union of a single type. If so, it returns that single YANG type from the
// provided enum and non-enum types lists (which is most conveniently extracted
// from enumAndNonEnumTypesForUnion). If the union contains multiple types, it
// returns the yang.Ynone type instead. If the lone type is an enum, then that
// is indicated with the boolean return value.
func getLoneUnionType(schema *yang.Entry, unionT reflect.Type, ets []reflect.Type, sks []yang.TypeKind) (yang.TypeKind, bool, error) {
	// Single type union -- GoStruct field is that type rather than a union Interface type.
	if !util.IsTypeInterface(unionT) && !util.IsTypeSliceOfInterface(unionT) {
		// Is not an interface, we must have exactly one type in the union.
		var yk yang.TypeKind
		var isEnum bool
		switch {
		// That one type is either an enum or not an enum.
		case len(sks) == 1 && len(ets) == 0:
			yk = sks[0]
		case len(sks) == 0 && len(ets) == 1:
			yk = schema.Type.Type[0].Kind
			isEnum = true
		default:
			return yang.Ynone, false, fmt.Errorf("got %v non-enum types and %v enum types for union schema %s for type %v, expect just one type in total", sks, ets, schema.Name, unionT)
		}
		return yk, isEnum, nil
	}
	return yang.Ynone, false, nil
}

// castToOneEnumValue loops through the given enum types in order in converting
// the string value, and returns upon success. If the string value can't be
// casted to any, nil is returned (without error).
func castToOneEnumValue(ets []reflect.Type, value string) (interface{}, error) {
	util.DbgPrint("castToOneEnumValue: %q", value)
	for _, et := range ets {
		util.DbgPrint("try to unmarshal into enum type %v", et)
		ev, err := castToEnumValue(et, value)
		if err != nil {
			return nil, err
		}
		if ev != nil {
			return ev, nil
		}
		util.DbgPrint("could not unmarshal %q into enum type, err: %v", value, err)
	}
	return nil, nil
}

// castToEnumValue returns value as the given type ft, if value is one of
// the allowed values of ft, or nil, nil otherwise.
func castToEnumValue(ft reflect.Type, value string) (interface{}, error) {
	if ft.Kind() == reflect.Slice {
		// leaf-list case
		ft = ft.Elem()
	}

	util.DbgPrint("checking for matching enum value for type %s", ft)
	mapMethod := reflect.New(ft).MethodByName("ΛMap")
	if !mapMethod.IsValid() {
		return 0, fmt.Errorf("%s does not have a ΛMap function", ft)
	}

	ec := mapMethod.Call(nil)
	if len(ec) == 0 {
		return 0, fmt.Errorf("%s ΛMap function returns empty value", ft)
	}
	ei := ec[0].Interface()
	enumMap, ok := ei.(map[string]map[int64]ygot.EnumDefinition)
	if !ok {
		return 0, fmt.Errorf("%s ΛMap function returned wrong type %T, want map[string]map[int64]ygot.EnumDefinition", ft, ei)
	}

	m, ok := enumMap[ft.Name()]
	if !ok {
		return 0, fmt.Errorf("%s is not a valid enum field name", ft.Name())
	}

	for k, v := range m {
		if util.StripModulePrefix(v.Name) == util.StripModulePrefix(value) {
			// Convert to destination enum type.
			return reflect.ValueOf(k).Convert(ft).Interface(), nil
		}
	}

	return nil, nil
}

func structFieldType(parent interface{}, fieldName string) reflect.Type {
	fv := reflect.ValueOf(parent).Elem().FieldByName(fieldName)
	ft := fv.Type()
	if util.IsValuePtr(fv) {
		ft = ft.Elem()
	}
	return ft
}

// StringToType converts given string to given type which can be one of
// the following;
// - int, int8, int16, int32, int64
// - uint, uint8, uint16, uint32, uint64
// - string
// - GoEnum type
// Function can be extended to support other types as well. If the given string
// carries an incompatible or overflowing value for the given type, function
// returns error.
// Note that castToEnumValue returns (nil, nil) if the given string is carrying
// an invalid enum string. Function checks not only error, but also value in this
// case.
func StringToType(t reflect.Type, s string) (reflect.Value, error) {
	if t.Implements(reflect.TypeOf((*ygot.GoEnum)(nil)).Elem()) {
		i, err := castToEnumValue(t, s)
		if err != nil || i == nil {
			return reflect.ValueOf(nil), fmt.Errorf("no enum matching with %s: %v", s, err)
		}
		return reflect.ValueOf(i), nil
	}

	switch t.Kind() {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		u, err := strconv.ParseUint(s, 10, int(t.Size())*8)
		if err != nil {
			return reflect.ValueOf(nil), fmt.Errorf("unable to convert %q to %v", s, t.Kind())
		}
		// Although Convert can panic, we know that the type is an unsigned integer type and
		// u must be a valid uint type of the same length -- it is therefore impossible that
		// Convert fails here.
		return reflect.ValueOf(u).Convert(t), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		u, err := strconv.ParseInt(s, 10, int(t.Size())*8)
		if err != nil {
			return reflect.ValueOf(nil), fmt.Errorf("unable to convert %q to %v", s, t.Kind())
		}
		// Although Convert can panic, we know that the type is an integer type and
		// u must be a valid int type of the same length -- it is therefore impossible that
		// Convert fails here.
		return reflect.ValueOf(u).Convert(t), nil
	case reflect.String:
		return reflect.ValueOf(s), nil
	case reflect.Bool:
		switch s {
		case "true":
			return reflect.ValueOf(true), nil
		case "false":
			return reflect.ValueOf(false), nil
		}
		return reflect.ValueOf(nil), fmt.Errorf("cannot cast to bool from %q", s)
	}
	return reflect.ValueOf(nil), fmt.Errorf("no matching type to cast for %v", t)
}

// stringToKeyType converts the given string to the type specified by the schema.
// This is in contrast to StringToType, which requires foreknowledge of the
// concrete type of the value. This is especially useful for converting the
// string to a union type, where the final concrete value is not known.
func stringToKeyType(schema *yang.Entry, parent interface{}, fieldName string, value string) (reflect.Value, error) {
	ykind := schema.Type.Kind
	switch ykind {
	// TODO(wenbli): case yang.Ybits:
	case yang.Yint64, yang.Yint32, yang.Yint16, yang.Yint8:
		bits, err := util.YangIntTypeBits(ykind)
		if err != nil {
			return reflect.ValueOf(nil), err
		}
		u, err := strconv.ParseInt(value, 10, bits)
		if err != nil {
			return reflect.ValueOf(nil), fmt.Errorf("unable to convert %q to %v", value, ykind)
		}
		// Although Convert can panic, we know that the type is an integer type and
		// u must be a valid int type of the same length -- it is therefore impossible that
		// Convert fails here.
		return reflect.ValueOf(u).Convert(reflect.TypeOf(yangBuiltinTypeToGoType(ykind))), nil
	case yang.Yuint64, yang.Yuint32, yang.Yuint16, yang.Yuint8:
		bits, err := util.YangIntTypeBits(ykind)
		if err != nil {
			return reflect.ValueOf(nil), err
		}
		u, err := strconv.ParseUint(value, 10, bits)
		if err != nil {
			return reflect.ValueOf(nil), fmt.Errorf("unable to convert %q to %v", value, ykind)
		}
		// Although Convert can panic, we know that the type is an integer type and
		// u must be a valid int type of the same length -- it is therefore impossible that
		// Convert fails here.
		return reflect.ValueOf(u).Convert(reflect.TypeOf(yangBuiltinTypeToGoType(ykind))), nil
	case yang.Ybinary:
		v, err := base64.StdEncoding.DecodeString(value)
		if err != nil {
			return reflect.ValueOf(nil), fmt.Errorf("error in DecodeString for \n%v\n for schema %s: %v", value, schema.Name, err)
		}
		return reflect.ValueOf([]byte(v)), nil
	case yang.Ystring:
		return reflect.ValueOf(value), nil
	case yang.Ybool:
		switch value {
		case "true":
			return reflect.ValueOf(true), nil
		case "false":
			return reflect.ValueOf(false), nil
		}
		return reflect.ValueOf(nil), fmt.Errorf("stringToKeyType: cannot convert %q to bool, schema.Type: %v", value, schema.Type)
	case yang.Ydecimal64:
		floatV, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return reflect.ValueOf(nil), fmt.Errorf("unable to convert %q to %v: %v", value, ykind, err)
		}
		return reflect.ValueOf(floatV), nil
	case yang.Yenum, yang.Yidentityref:
		enumVal, err := enumStringToValue(parent, fieldName, value)
		return reflect.ValueOf(enumVal), err
	case yang.Yunion:
		return stringToUnionType(schema, parent, fieldName, value)
	case yang.Yleafref:
		schema, err := util.FindLeafRefSchema(schema, schema.Type.Path)
		if err != nil {
			return reflect.ValueOf(nil), fmt.Errorf("stringToKeyType: unable to find target schema from leafref schema: %s", err)
		}
		return stringToKeyType(schema, parent, fieldName, value)
	}

	return reflect.ValueOf(nil), fmt.Errorf("stringToKeyType: unsupported type %v for conversion from string %q, schema.Type: %v", ykind, value, schema.Type)
}

// stringToUnionType converts a string value into a suitable union type
// determined by where it is located in the YANG tree.
func stringToUnionType(schema *yang.Entry, parent interface{}, fieldName string, value string) (reflect.Value, error) {
	util.DbgPrint("stringToUnionType value %v, into parent type %T field name %s, schema name %s", util.ValueStrDebug(value), parent, fieldName, schema.Name)
	if !util.IsTypeStructPtr(reflect.TypeOf(parent)) {
		return reflect.ValueOf(nil), fmt.Errorf("stringToKeyType: %T is not a struct ptr", parent)
	}
	parentT := reflect.TypeOf(parent)
	dft, found := parentT.Elem().FieldByName(fieldName)
	if !found {
		return reflect.ValueOf(nil), fmt.Errorf("stringToUnionType: field %q not found in parent type %T", fieldName, parent)
	}
	destUnionFieldElemT := dft.Type

	ets, sks, err := enumAndNonEnumTypesForUnion(schema, parentT)
	if err != nil {
		return reflect.ValueOf(nil), err
	}

	// Special case. If all possible union types map to a single go type, the
	// GoStruct field is that type rather than a union Interface type.
	loneType, _, err := getLoneUnionType(schema, destUnionFieldElemT, ets, sks)
	if err != nil {
		return reflect.ValueOf(nil), err
	}
	if loneType != yang.Ynone {
		return stringToKeyType(yangKindToLeafEntry(loneType), parent, fieldName, value)
	}

	fieldType := structFieldType(parent, fieldName)
	// For each possible union type, try to convert/unmarshal the value.
	// Note that values can resolve into more than one struct type depending on
	// the value and its range. In this case, no attempt is made to find the
	// most restrictive type.
	// Try to unmarshal to enum types first, since the case of union of string
	// and enum could unmarshal into either. Only string values can be enum
	// types.
	ev, err := castToOneEnumValue(ets, value)
	if err != nil {
		return reflect.ValueOf(nil), err
	}
	if ev != nil {
		return getUnionVal(reflect.TypeOf(parent), fieldType, ev)
	}

	for _, sk := range sks {
		util.DbgPrint("try to convert string %q into type %s", value, sk)
		gv, err := stringToKeyType(yangKindToLeafEntry(sk), parent, fieldName, value)
		if err == nil {
			return getUnionVal(reflect.TypeOf(parent), fieldType, gv.Interface())
		}
		util.DbgPrint("could not unmarshal %v into type %v: %v", value, sk, err)
	}

	return reflect.ValueOf(nil), fmt.Errorf("could not find suitable union type to unmarshal value %q into parent struct type %T field %s", value, parent, fieldName)
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
	case yang.Ybool:
		return reflect.TypeOf(bool(false))
	case yang.Yempty:
		return reflect.TypeOf([]interface{}{})
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
