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
	"github.com/openconfig/ygot/ygot"

	log "github.com/golang/glog"
)

// enumStringToIntValue returns the integer value that enumerated string value
// of type fieldName maps to in the parent, which must be a struct ptr.
func enumStringToIntValue(parent interface{}, fieldName, value string) (int64, error) {
	v := reflect.ValueOf(parent)
	if !IsValueStructPtr(v) {
		return 0, fmt.Errorf("enumStringToIntValue: %T is not a struct ptr", parent)
	}
	field := v.Elem().FieldByName(fieldName)
	if !field.IsValid() {
		return 0, fmt.Errorf("%s is not a valid enum field name in %T", fieldName, parent)
	}
	ft := field.Type()
	// leaf-list case
	if ft.Kind() == reflect.Slice {
		ft = ft.Elem()
	}

	mapMethod := reflect.New(ft).Elem().MethodByName("ΛMap")
	if !mapMethod.IsValid() {
		return 0, fmt.Errorf("%s in %T does not have a ΛMap function", fieldName, parent)
	}

	ec := mapMethod.Call(nil)
	if len(ec) == 0 {
		return 0, fmt.Errorf("%s in %T ΛMap function returns empty value", fieldName, parent)
	}
	ei := ec[0].Interface()
	enumMap, ok := ei.(map[string]map[int64]ygot.EnumDefinition)
	if !ok {
		return 0, fmt.Errorf("%s in %T ΛMap function returned wrong type %T, want map[string]map[int64]ygot.EnumDefinition",
			fieldName, parent, ei)
	}

	m, ok := enumMap[ft.Name()]
	if !ok {
		return 0, fmt.Errorf("%s is not a valid enum field name, has type %s", fieldName, ft.Name())
	}

	for k, v := range m {
		if stripModulePrefix(v.Name) == stripModulePrefix(value) {
			return k, nil
		}
	}

	return 0, fmt.Errorf("%s is not a valid value for enum field %s", value, fieldName)
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
