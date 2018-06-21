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
	}
	return reflect.ValueOf(nil), fmt.Errorf("no matching type to cast for %v", t)
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
