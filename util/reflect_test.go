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

package util

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/kylelemons/godebug/pretty"
	"github.com/openconfig/goyang/pkg/yang"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
)

const (
	// wildcardStr is a wildcard string that matches any one word in a string.
	wildcardStr = "{{*}}"
	// testErrOutput controls whether expect error test cases log the error
	// values.
	testErrOutput = false
)

// errToString returns the string representation of err and the empty string if
// err is nil.
func errToString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// testErrLog logs err to t if err != nil and global value testErrOutput is set.
func testErrLog(t *testing.T, desc string, err error) {
	if err != nil {
		if testErrOutput {
			t.Logf("%s: %v", desc, err)
		}
	}
}

// areEqual compares a and b. If a and b are both pointers, it compares the
// values they are pointing to.
func areEqual(a, b interface{}) bool {
	if IsValueNil(a) && IsValueNil(b) {
		return true
	}
	va, vb := reflect.ValueOf(a), reflect.ValueOf(b)
	if va.Kind() == reflect.Ptr && vb.Kind() == reflect.Ptr {
		return reflect.DeepEqual(va.Elem().Interface(), vb.Elem().Interface())
	}

	return reflect.DeepEqual(a, b)
}

// areEqualWithWildcards compares s against pattern word by word, where any
// instances of wildcardStr in pattern are skipped in s.
func areEqualWithWildcards(s, pattern string) bool {
	pv, sv := strings.Split(pattern, " "), strings.Split(s, " ")
	if len(pv) != len(sv) {
		return false
	}
	for i, v := range pv {
		if v == wildcardStr {
			continue
		}
		if pv[i] != sv[i] {
			return false
		}
	}
	return true
}

// to ptr conversion utility functions
func toStringPtr(s string) *string { return &s }
func toInt8Ptr(i int8) *int8       { return &i }
func toInt32Ptr(i int32) *int32    { return &i }

func TestIsValueNil(t *testing.T) {
	if !IsValueNil(nil) {
		t.Error("got IsValueNil(nil) false, want true")
	}
	if !IsValueNil((*int)(nil)) {
		t.Error("got IsValueNil(ptr) false, want true")
	}
	if !IsValueNil((map[int]int)(nil)) {
		t.Error("got IsValueNil(map) false, want true")
	}
	if !IsValueNil(([]int)(nil)) {
		t.Error("got IsValueNil(slice) false, want true")
	}
	if !IsValueNil((interface{})(nil)) {
		t.Error("got IsValueNil(interface) false, want true")
	}

	if IsValueNil(toInt8Ptr(42)) {
		t.Error("got IsValueNil(ptr) true, want false")
	}
	if IsValueNil(map[int]int{42: 42}) {
		t.Error("got IsValueNil(map) true, want false")
	}
	if IsValueNil([]int{1, 2, 3}) {
		t.Error("got IsValueNil(slice) true, want false")
	}
	if IsValueNil((interface{})(42)) {
		t.Error("got IsValueNil(interface) true, want false")
	}
}

func TestIsValueNilOrDefault(t *testing.T) {
	if !IsValueNilOrDefault(nil) {
		t.Error("got IsValueNilOrDefault(nil) false, want true")
	}
	if !IsValueNilOrDefault((*int)(nil)) {
		t.Error("got IsValueNilOrDefault(ptr) false, want true")
	}
	if !IsValueNilOrDefault((map[int]int)(nil)) {
		t.Error("got IsValueNilOrDefault(map) false, want true")
	}
	if !IsValueNilOrDefault(([]int)(nil)) {
		t.Error("got IsValueNilOrDefault(slice) false, want true")
	}
	if !IsValueNilOrDefault((interface{})(nil)) {
		t.Error("got IsValueNilOrDefault(interface) false, want true")
	}
	if !IsValueNilOrDefault(int(0)) {
		t.Error("got IsValueNilOrDefault(int(0)) false, want true")
	}
	if !IsValueNilOrDefault("") {
		t.Error("got IsValueNilOrDefault(\"\") false, want true")
	}
	if !IsValueNilOrDefault(false) {
		t.Error("got IsValueNilOrDefault(false) false, want true")
	}
	i := 32
	ip := &i
	if IsValueNilOrDefault(&ip) {
		t.Error("got IsValueNilOrDefault(ptr to ptr) false, want true")
	}
}

func TestIsValueFuncs(t *testing.T) {
	testInt := int(42)
	testStruct := struct{}{}
	testSlice := []bool{}
	testMap := map[bool]bool{}
	var testNilSlice []bool
	var testNilMap map[bool]bool

	allValues := []interface{}{nil, testInt, &testInt, testStruct, &testStruct, testNilSlice, testSlice, &testSlice, testNilMap, testMap, &testMap}

	tests := []struct {
		desc     string
		function func(v reflect.Value) bool
		okValues []interface{}
	}{
		{
			desc:     "IsValuePtr",
			function: IsValuePtr,
			okValues: []interface{}{&testInt, &testStruct, &testSlice, &testMap},
		},
		{
			desc:     "IsValueStruct",
			function: IsValueStruct,
			okValues: []interface{}{testStruct},
		},
		{
			desc:     "IsValueInterface",
			function: IsValueInterface,
			okValues: []interface{}{},
		},
		{
			desc:     "IsValueStructPtr",
			function: IsValueStructPtr,
			okValues: []interface{}{&testStruct},
		},
		{
			desc:     "IsValueMap",
			function: IsValueMap,
			okValues: []interface{}{testNilMap, testMap},
		},
		{
			desc:     "IsValueSlice",
			function: IsValueSlice,
			okValues: []interface{}{testNilSlice, testSlice},
		},
		{
			desc:     "IsValueScalar",
			function: IsValueScalar,
			okValues: []interface{}{testInt, &testInt},
		},
	}

	for _, tt := range tests {
		for vidx, v := range allValues {
			if got, want := tt.function(reflect.ValueOf(v)), isInListOfInterface(tt.okValues, v); got != want {
				t.Errorf("%s with %s (#%d): got: %t, want: %t", tt.desc, reflect.TypeOf(v), vidx, got, want)
			}
		}
	}
}

func TestIsTypeFuncs(t *testing.T) {
	testInt := int(42)
	testStruct := struct{}{}
	testSlice := []bool{}
	testSliceOfInterface := []interface{}{}
	testMap := map[bool]bool{}
	var testNilSlice []bool
	var testNilMap map[bool]bool

	allTypes := []interface{}{nil, testInt, &testInt, testStruct, &testStruct, testNilSlice,
		testSlice, &testSlice, testSliceOfInterface, testNilMap, testMap, &testMap}

	tests := []struct {
		desc     string
		function func(v reflect.Type) bool
		okTypes  []interface{}
	}{
		{
			desc:     "IsTypeStructPtr",
			function: IsTypeStructPtr,
			okTypes:  []interface{}{&testStruct},
		},
		{
			desc:     "IsTypeSlicePtr",
			function: IsTypeSlicePtr,
			okTypes:  []interface{}{&testSlice},
		},
		{
			desc:     "IsTypeMap",
			function: IsTypeMap,
			okTypes:  []interface{}{testNilMap, testMap},
		},
		{
			desc:     "IsTypeInterface",
			function: IsTypeInterface,
			okTypes:  []interface{}{},
		},
		{
			desc:     "IsTypeSliceOfInterface",
			function: IsTypeSliceOfInterface,
			okTypes:  []interface{}{testSliceOfInterface},
		},
	}

	for _, tt := range tests {
		for vidx, v := range allTypes {
			if got, want := tt.function(reflect.TypeOf(v)), isInListOfInterface(tt.okTypes, v); got != want {
				t.Errorf("%s with %s (#%d): got: %t, want: %t", tt.desc, reflect.TypeOf(v), vidx, got, want)
			}
		}
	}

}

type interfaceContainer struct {
	I anInterface
}

type anInterface interface {
	IsU()
}

type implementsInterface struct {
	A string
}

func (*implementsInterface) IsU() {}

func TestIsValueInterface(t *testing.T) {
	intf := &interfaceContainer{
		I: &implementsInterface{
			A: "a",
		},
	}
	iField := reflect.ValueOf(intf).Elem().FieldByName("I")
	if !IsValueInterface(iField) {
		t.Errorf("IsValueInterface(): got false, want true")
	}
	if !IsValueInterfaceToStructPtr(iField) {
		t.Errorf("IsValueInterface(): got false, want true")
	}
}

func TestIsTypeInterface(t *testing.T) {
	intf := &interfaceContainer{
		I: &implementsInterface{
			A: "a",
		},
	}
	testIfField := reflect.ValueOf(intf).Elem().Field(0)

	if !IsTypeInterface(testIfField.Type()) {
		t.Errorf("IsTypeInterface(): got false, want true")
	}
}

func isInListOfInterface(lv []interface{}, v interface{}) bool {
	for _, vv := range lv {
		if reflect.DeepEqual(vv, v) {
			return true
		}
	}
	return false
}

func TestUpdateField(t *testing.T) {
	type BasicStruct struct {
		IntField       int
		StringField    string
		IntPtrField    *int8
		StringPtrField *string
	}

	type StructOfStructs struct {
		BasicStructField *BasicStruct
	}

	tests := []struct {
		desc         string
		parentStruct interface{}
		fieldName    string
		fieldValue   interface{}
		wantVal      interface{}
		wantErr      string
	}{
		{
			desc:         "int",
			parentStruct: &BasicStruct{},
			fieldName:    "IntField",
			fieldValue:   42,
			wantVal:      &BasicStruct{IntField: 42},
		},
		{
			desc:         "int with nil",
			parentStruct: &BasicStruct{},
			fieldName:    "IntField",
			fieldValue:   nil,
			wantErr:      "cannot assign value <nil> (type <nil>) to struct field IntField (type int) in struct *util.BasicStruct",
		},
		{
			desc:         "nil parent",
			parentStruct: nil,
			fieldName:    "IntField",
			fieldValue:   42,
			wantErr:      "parent is nil in UpdateField for field IntField",
		},
		{
			desc:         "bad parent type",
			parentStruct: struct{}{},
			wantErr:      "parent type struct {} must be a struct ptr",
		},
		{
			desc:         "string",
			parentStruct: &BasicStruct{},
			fieldName:    "StringField",
			fieldValue:   "forty two",
			wantVal:      &BasicStruct{StringField: "forty two"},
		},
		{
			desc:         "nil parent struct",
			parentStruct: nil,
			fieldName:    "IntField",
			fieldValue:   42,
			wantErr:      "parent is nil in UpdateField for field IntField",
		},
		{
			desc:         "string to int field error",
			parentStruct: &BasicStruct{},
			fieldName:    "IntField",
			fieldValue:   "forty two",
			wantErr:      "cannot assign value forty two (type string) to struct field IntField (type int) in struct *util.BasicStruct",
		},
		{
			desc:         "int ptr",
			parentStruct: &BasicStruct{},
			fieldName:    "IntPtrField",
			fieldValue:   toInt8Ptr(42),
			wantVal:      &BasicStruct{IntPtrField: toInt8Ptr(42)},
		},
		{
			desc:         "nil int ptr",
			parentStruct: &BasicStruct{IntPtrField: toInt8Ptr(42)},
			fieldName:    "IntPtrField",
			fieldValue:   nil,
			wantVal:      &BasicStruct{},
		},
		{
			desc:         "string ptr",
			parentStruct: &BasicStruct{},
			fieldName:    "StringPtrField",
			fieldValue:   toStringPtr("forty two"),
			wantVal:      &BasicStruct{StringPtrField: toStringPtr("forty two")},
		},
		{
			desc:         "bad field error",
			parentStruct: &BasicStruct{},
			fieldName:    "BadField",
			wantErr:      "parent type *util.BasicStruct does not have a field name BadField",
		},
		{
			desc:         "int to int ptr field error",
			parentStruct: &BasicStruct{},
			fieldName:    "IntPtrField",
			fieldValue:   42,
			wantErr:      "cannot assign value 42 (type int) to struct field IntPtrField (type *int8) in struct *util.BasicStruct",
		},
		{
			desc:         "int ptr to int field error",
			parentStruct: &BasicStruct{},
			fieldName:    "IntField",
			fieldValue:   toInt8Ptr(42),
			wantErr:      "cannot assign value " + wildcardStr + " (type *int8) to struct field IntField (type int) in struct *util.BasicStruct",
		},
		{
			desc:         "struct",
			parentStruct: &StructOfStructs{},
			fieldName:    "BasicStructField",
			fieldValue:   &BasicStruct{IntField: 42, StringField: "forty two"},
			wantVal:      &StructOfStructs{BasicStructField: &BasicStruct{IntField: 42, StringField: "forty two"}},
		},
		{
			desc:         "struct bad field name",
			parentStruct: &StructOfStructs{},
			fieldName:    "StructBadField",
			fieldValue:   &BasicStruct{IntField: 42, StringField: "forty two"},
			wantErr:      "parent type *util.StructOfStructs does not have a field name StructBadField",
		},
		{
			desc:         "struct bad field type",
			parentStruct: &StructOfStructs{},
			fieldName:    "BasicStructField",
			fieldValue:   42,
			wantErr:      "cannot assign value 42 (type int) to struct field BasicStructField (type *util.BasicStruct) in struct *util.StructOfStructs",
		},
	}

	for _, tt := range tests {
		err := UpdateField(tt.parentStruct, tt.fieldName, tt.fieldValue)
		if got, want := errToString(err), tt.wantErr; !areEqualWithWildcards(got, want) {
			t.Errorf("%s: got error: %s, want error: %s", tt.desc, got, want)
		}
		if err == nil {
			if got, want := tt.parentStruct, tt.wantVal; !areEqual(got, want) {
				t.Errorf("%s: got:\n%v\nwant:\n%v\n", tt.desc, pretty.Sprint(got), pretty.Sprint(want))
			}
		}
		testErrLog(t, tt.desc, err)
	}
}

func TestInsertIntoSliceStructField(t *testing.T) {
	type BasicStruct struct {
		IntSliceField    []int
		IntPtrSliceField []*int8
		NonSliceField    int
	}

	tests := []struct {
		desc         string
		parentStruct interface{}
		fieldName    string
		fieldValue   interface{}
		wantVal      interface{}
		wantErr      string
	}{
		{
			desc:         "slice of int",
			parentStruct: &BasicStruct{},
			fieldName:    "IntSliceField",
			fieldValue:   42,
			wantVal:      &BasicStruct{IntSliceField: []int{42}},
		},
		{
			desc:         "slice of int ptr",
			parentStruct: &BasicStruct{IntPtrSliceField: []*int8{toInt8Ptr(42)}},
			fieldName:    "IntPtrSliceField",
			fieldValue:   toInt8Ptr(43),
			wantVal:      &BasicStruct{IntPtrSliceField: []*int8{toInt8Ptr(42), toInt8Ptr(43)}},
		},
		{
			desc:         "slice of int ptr, nil value",
			parentStruct: &BasicStruct{},
			fieldName:    "IntPtrSliceField",
			fieldValue:   nil,
			wantVal:      &BasicStruct{IntPtrSliceField: []*int8{nil}},
		},
		{
			desc:         "missing field",
			parentStruct: &BasicStruct{},
			fieldName:    "MissingField",
			wantErr:      "parent type *util.BasicStruct does not have a field name MissingField",
		},
		{
			desc:         "bad parent type",
			parentStruct: struct{}{},
			wantErr:      "parent type struct {} must be a struct ptr",
		},
		{
			desc:         "bad field type",
			parentStruct: &BasicStruct{},
			fieldName:    "NonSliceField",
			fieldValue:   42,
			wantErr:      "parent type *util.BasicStruct, field name NonSliceField is type int, must be a slice",
		},
		{
			desc:         "slice of int, bad field type",
			parentStruct: &BasicStruct{},
			fieldName:    "IntSliceField",
			fieldValue:   "forty-two",
			wantErr:      "cannot assign value forty-two (type string) to struct field IntSliceField (type int) in struct *util.BasicStruct",
		},
	}

	for _, tt := range tests {
		err := InsertIntoSliceStructField(tt.parentStruct, tt.fieldName, tt.fieldValue)
		if got, want := errToString(err), tt.wantErr; !areEqualWithWildcards(got, want) {
			t.Errorf("%s: got error: %s, want error: %s", tt.desc, got, want)
		}
		if err == nil {
			if got, want := tt.parentStruct, tt.wantVal; !areEqual(got, want) {
				t.Errorf("%s: got:\n%v\nwant:\n%v\n", tt.desc, pretty.Sprint(got), pretty.Sprint(want))
			}
		}
		testErrLog(t, tt.desc, err)
	}
}

func TestInsertIntoMapStructField(t *testing.T) {
	type KeyStruct struct {
		IntField int
	}

	type BasicStruct struct {
		StringToIntMapField    map[string]int
		StringToIntPtrMapField map[string]*int8
		StructToIntMapField    map[KeyStruct]int
		NonMapField            int
	}

	tests := []struct {
		desc         string
		parentStruct interface{}
		fieldName    string
		key          interface{}
		fieldValue   interface{}
		wantVal      interface{}
		wantErr      string
	}{
		{
			desc:         "string to int, create map",
			parentStruct: &BasicStruct{},
			fieldName:    "StringToIntMapField",
			key:          "forty-two",
			fieldValue:   42,
			wantVal:      &BasicStruct{StringToIntMapField: map[string]int{"forty-two": 42}},
		},
		{
			desc:         "string to int, map exists",
			parentStruct: &BasicStruct{StringToIntMapField: map[string]int{"forty-two": 42}},
			fieldName:    "StringToIntMapField",
			key:          "forty-three",
			fieldValue:   43,
			wantVal:      &BasicStruct{StringToIntMapField: map[string]int{"forty-two": 42, "forty-three": 43}},
		},
		{
			desc:         "string to int, update value",
			parentStruct: &BasicStruct{StringToIntMapField: map[string]int{"forty-two": 42}},
			fieldName:    "StringToIntMapField",
			key:          "forty-two",
			fieldValue:   43,
			wantVal:      &BasicStruct{StringToIntMapField: map[string]int{"forty-two": 43}},
		},
		{
			desc:         "string to int ptr",
			parentStruct: &BasicStruct{},
			fieldName:    "StringToIntPtrMapField",
			key:          "forty-two",
			fieldValue:   toInt8Ptr(42),
			wantVal:      &BasicStruct{StringToIntPtrMapField: map[string]*int8{"forty-two": toInt8Ptr(42)}},
		},
		{
			desc:         "string to int ptr, nil value",
			parentStruct: &BasicStruct{},
			fieldName:    "StringToIntPtrMapField",
			key:          "forty-two",
			fieldValue:   nil,
			wantVal:      &BasicStruct{StringToIntPtrMapField: map[string]*int8{"forty-two": nil}},
		},
		{
			desc:         "struct to int",
			parentStruct: &BasicStruct{},
			fieldName:    "StructToIntMapField",
			key:          KeyStruct{IntField: 42},
			fieldValue:   42,
			wantVal:      &BasicStruct{StructToIntMapField: map[KeyStruct]int{{IntField: 42}: 42}},
		},
		{
			desc:         "missing field",
			parentStruct: &BasicStruct{},
			fieldName:    "MissingField",
			key:          "forty-two",
			fieldValue:   42,
			wantErr:      "field MissingField not found in parent type *util.BasicStruct",
		},
		{
			desc:         "bad field type",
			parentStruct: &BasicStruct{},
			fieldName:    "NonMapField",
			wantErr:      "field NonMapField to insert into must be a map, type is int",
		},
		{
			desc:         "string to int, bad value",
			parentStruct: &BasicStruct{},
			fieldName:    "StringToIntMapField",
			key:          "forty-two",
			fieldValue:   "forty-two",
			wantErr:      "cannot assign value forty-two (type string) to field StringToIntMapField (type int) in struct BasicStruct",
		},
	}

	for _, tt := range tests {
		err := InsertIntoMapStructField(tt.parentStruct, tt.fieldName, tt.key, tt.fieldValue)
		if got, want := errToString(err), tt.wantErr; !areEqualWithWildcards(got, want) {
			t.Errorf("%s: got error: %s, want error: %s", tt.desc, got, want)
		}
		if err == nil {
			if got, want := tt.parentStruct, tt.wantVal; !areEqual(got, want) {
				t.Errorf("%s: got:\n%v\nwant:\n%v\n", tt.desc, pretty.Sprint(got), pretty.Sprint(want))
			}
		}
		testErrLog(t, tt.desc, err)
	}
}

func TestInsertIntoSlice(t *testing.T) {
	parentSlice := []int{42, 43}
	value := 44
	if err := InsertIntoSlice(&parentSlice, value); err != nil {
		t.Fatalf("got error: %s, want error: nil", err)
	}
	wantSlice := []int{42, 43, value}
	if got, want := parentSlice, wantSlice; !reflect.DeepEqual(got, want) {
		t.Errorf("got:\n%v\nwant:\n%v\n", got, want)
	}

	badParent := struct{}{}
	wantErr := `InsertIntoSlice parent type is *struct {}, must be slice ptr`
	if got, want := errToString(InsertIntoSlice(&badParent, value)), wantErr; got != want {
		t.Fatalf("got error: %s, want error: %s", got, want)
	}
}

func TestInsertIntoMap(t *testing.T) {
	parentMap := map[int]string{42: "forty two", 43: "forty three"}
	key := 44
	value := "forty four"
	if err := InsertIntoMap(parentMap, key, value); err != nil {
		t.Fatalf("got error: %s, want error: nil", err)
	}
	wantMap := map[int]string{42: "forty two", 43: "forty three", 44: "forty four"}
	if got, want := parentMap, wantMap; !reflect.DeepEqual(got, want) {
		t.Errorf("got:\n%v\nwant:\n%v\n", got, want)
	}

	badParent := struct{}{}
	wantErr := `InsertIntoMap parent type is *struct {}, must be map`
	if got, want := errToString(InsertIntoMap(&badParent, key, value)), wantErr; got != want {
		t.Fatalf("got error: %s, want error: %s", got, want)
	}
}

var (
	// forEachContainerSchema is a schema shared in tests below.
	forEachContainerSchema = &yang.Entry{
		Name: "container",
		Kind: yang.DirectoryEntry,
		Dir: map[string]*yang.Entry{
			"basic-struct": {
				Name: "basic-struct",
				Kind: yang.DirectoryEntry,
				Dir: map[string]*yang.Entry{
					"int32": {
						Kind: yang.LeafEntry,
						Name: "int32",
						Type: &yang.YangType{Kind: yang.Yint32},
					},
					"string": {
						Kind: yang.LeafEntry,
						Name: "string",
						Type: &yang.YangType{Kind: yang.Ystring},
					},
					"int32ptr": {
						Kind: yang.LeafEntry,
						Name: "int32ptr",
						Type: &yang.YangType{Kind: yang.Yint32},
					},
					"stringptr": {
						Kind: yang.LeafEntry,
						Name: "stringptr",
						Type: &yang.YangType{Kind: yang.Ystring},
					},
				},
			},
		},
	}
)

func TestForEachField(t *testing.T) {
	type BasicStruct struct {
		Int32Field     int32   `path:"int32"`
		StringField    string  `path:"string"`
		Int32PtrField  *int32  `path:"int32ptr"`
		StringPtrField *string `path:"stringptr"`
	}

	type StructOfStructs struct {
		BasicStructField    BasicStruct  `path:"basic-struct"`
		BasicStructPtrField *BasicStruct `path:"basic-struct"`
	}

	type StructOfSliceOfStructs struct {
		BasicStructSliceField    []BasicStruct  `path:"basic-struct"`
		BasicStructPtrSliceField []*BasicStruct `path:"basic-struct"`
	}

	type StructOfMapOfStructs struct {
		BasicStructMapField    map[string]BasicStruct  `path:"basic-struct"`
		BasicStructPtrMapField map[string]*BasicStruct `path:"basic-struct"`
	}

	printFieldsIterFunc := func(ni *NodeInfo, in, out interface{}) (errs Errors) {
		// Only print basic scalar values, skip everything else.
		if !IsValueScalar(ni.FieldValue) || IsValueNil(ni.FieldKey) {
			return
		}
		outs := out.(*string)
		*outs += fmt.Sprintf("%v : %v, ", ni.StructField.Name, pretty.Sprint(ni.FieldValue.Interface()))
		return
	}

	printMapKeysIterFunc := func(ni *NodeInfo, in, out interface{}) (errs Errors) {
		if IsNilOrInvalidValue(ni.FieldKey) {
			return
		}
		outs := out.(*string)
		s := "nil"
		if !IsNilOrInvalidValue(ni.FieldValue) {
			s = pretty.Sprint(ni.FieldValue.Interface())
		}
		*outs += fmt.Sprintf("%s/%s : \n%s\n, ", ValueStr(ni.FieldKey.Interface()), ni.StructField.Name, ValueStr(s))
		return
	}

	basicStruct1 := BasicStruct{Int32Field: int32(42), StringField: "forty two", Int32PtrField: toInt32Ptr(4242), StringPtrField: toStringPtr("forty two ptr")}
	basicStruct2 := BasicStruct{Int32Field: int32(43), StringField: "forty three", Int32PtrField: toInt32Ptr(4343), StringPtrField: toStringPtr("forty three ptr")}

	tests := []struct {
		desc         string
		schema       *yang.Entry
		parentStruct interface{}
		in           interface{}
		out          interface{}
		iterFunc     FieldIteratorFunc
		wantOut      string
		wantErr      string
	}{
		{
			desc:         "nil",
			schema:       nil,
			parentStruct: nil,
			in:           nil,
			iterFunc:     printFieldsIterFunc,
			wantOut:      ``,
		},
		{
			desc:         "struct",
			schema:       forEachContainerSchema.Dir["basic-struct"],
			parentStruct: &basicStruct1,
			in:           nil,
			iterFunc:     printFieldsIterFunc,
			wantOut:      `Int32Field : 42, StringField : "forty two", Int32PtrField : 4242, StringPtrField : "forty two ptr", `,
		},
		{
			desc:         "struct of struct",
			schema:       forEachContainerSchema,
			parentStruct: &StructOfStructs{BasicStructField: basicStruct1, BasicStructPtrField: &basicStruct2},
			in:           nil,
			iterFunc:     printFieldsIterFunc,
			wantOut: `Int32Field : 42, StringField : "forty two", Int32PtrField : 4242, StringPtrField : "forty two ptr", ` +
				`Int32Field : 43, StringField : "forty three", Int32PtrField : 4343, StringPtrField : "forty three ptr", `,
		},
		{
			desc:         "struct of slice of structs",
			schema:       forEachContainerSchema,
			parentStruct: &StructOfSliceOfStructs{BasicStructSliceField: []BasicStruct{basicStruct1}, BasicStructPtrSliceField: []*BasicStruct{&basicStruct2}},
			in:           nil,
			iterFunc:     printFieldsIterFunc,
			wantOut:      `Int32Field : 42, StringField : "forty two", Int32PtrField : 4242, StringPtrField : "forty two ptr", Int32Field : 43, StringField : "forty three", Int32PtrField : 4343, StringPtrField : "forty three ptr", `,
		},
		{
			desc:         "struct of map of structs",
			schema:       forEachContainerSchema,
			parentStruct: &StructOfMapOfStructs{BasicStructMapField: map[string]BasicStruct{"basicStruct1": basicStruct1}, BasicStructPtrMapField: map[string]*BasicStruct{"basicStruct2": &basicStruct2}},
			in:           nil,
			iterFunc:     printFieldsIterFunc,
			wantOut:      `Int32Field : 42, StringField : "forty two", Int32PtrField : 4242, StringPtrField : "forty two ptr", Int32Field : 43, StringField : "forty three", Int32PtrField : 4343, StringPtrField : "forty three ptr", `,
		},
		{
			desc:         "map keys",
			schema:       forEachContainerSchema,
			parentStruct: &StructOfMapOfStructs{BasicStructMapField: map[string]BasicStruct{"basicStruct1": basicStruct1}, BasicStructPtrMapField: map[string]*BasicStruct{"basicStruct2": &basicStruct2}},
			in:           nil,
			iterFunc:     printMapKeysIterFunc,
			wantOut: `basicStruct1 (string)/BasicStructMapField : 
{Int32Field:     42,
 StringField:    "forty two",
 Int32PtrField:  4242,
 StringPtrField: "forty two ptr"} (string)
, basicStruct2 (string)/BasicStructPtrMapField : 
{Int32Field:     43,
 StringField:    "forty three",
 Int32PtrField:  4343,
 StringPtrField: "forty three ptr"} (string)
, `,
		},
	}

	for _, tt := range tests {
		outStr := ""
		var errs Errors
		errs = ForEachField(tt.schema, tt.parentStruct, tt.in, &outStr, tt.iterFunc)
		if got, want := errs.String(), tt.wantErr; got != want {
			t.Errorf("%s: got error: %s, want error: %s", tt.desc, got, want)
		}
		if errs == nil {
			if got, want := outStr, tt.wantOut; got != want {
				t.Errorf("%s:\ngot:\n(%v)\nwant:\n(%v)", tt.desc, got, want)
			}
		}
		testErrLog(t, tt.desc, errs)
	}
}

func TestUpdateFieldUsingForEachField(t *testing.T) {
	type BasicStruct struct {
		Int32Field     int32   `path:"int32"`
		StringField    string  `path:"string"`
		Int32PtrField  *int32  `path:"int32ptr"`
		StringPtrField *string `path:"stringptr"`
	}

	type StructOfStructs struct {
		BasicStructField *BasicStruct `path:"basic-struct"`
	}

	basicStruct1 := BasicStruct{Int32Field: int32(42), StringField: "forty two", Int32PtrField: toInt32Ptr(4242), StringPtrField: toStringPtr("forty two ptr")}

	// This doesn't work as a general insert because it won't create fields
	// that are nil, they must already exist. It only works as an update.
	setFunc := func(ni *NodeInfo, in, out interface{}) (errs Errors) {
		if ni.StructField.Name == "BasicStructField" {
			errs = AppendErr(errs, UpdateField(ni.Parent.FieldValue.Interface(), "BasicStructField", &basicStruct1))
		}
		return
	}

	a := StructOfStructs{BasicStructField: &BasicStruct{}}

	if errs := ForEachField(forEachContainerSchema, &a, nil, nil, setFunc); errs != nil {
		t.Fatalf("setFunc got unexpected error: %s", errs)
	}

	if got, want := *a.BasicStructField, basicStruct1; got != want {
		t.Errorf("set struct: got: %s, want: %s", pretty.Sprint(got), pretty.Sprint(want))
	}
}

func TestStructValueHasNFields(t *testing.T) {
	type one struct {
		One string
	}

	type two struct {
		One string
		Two string
	}

	tests := []struct {
		name     string
		inStruct reflect.Value
		inNumber int
		want     bool
	}{{
		name:     "one",
		inStruct: reflect.ValueOf(one{}),
		inNumber: 1,
		want:     true,
	}, {
		name:     "one != two",
		inStruct: reflect.ValueOf(one{}),
		inNumber: 2,
		want:     false,
	}, {
		name:     "two",
		inStruct: reflect.ValueOf(two{}),
		inNumber: 2,
		want:     true,
	}, {
		name:     "non-struct type",
		inStruct: reflect.ValueOf("check"),
		inNumber: 42,
		want:     false,
	}}

	for _, tt := range tests {
		if got := IsStructValueWithNFields(tt.inStruct, tt.inNumber); got != tt.want {
			t.Errorf("%s: StructValueHasNFields(%#v, %d): did not get expected return, got: %v, want: %v", tt.name, tt.inStruct, tt.inNumber, got, tt.want)
		}
	}
}

// Types below are public to follow ygot generator output. Fields are public
// for reflect/serialization.

// InnerContainerType1 is a container type for testing.
type InnerContainerType1 struct {
	LeafName *int32 `path:"leaf-field"`
}

// IsYANGGoStruct implements the GoStruct interface method.
func (*InnerContainerType1) IsYANGGoStruct() {}

// OuterContainerType1 is a container type for testing.
type OuterContainerType1 struct {
	Inner        *InnerContainerType1 `path:"inner|config/inner"`
	InnerAbsPath *InnerContainerType1 `path:"inner-abs-path|config/inner-abs-path"`
}

// IsYANGGoStruct implements the GoStruct interface method.
func (*OuterContainerType1) IsYANGGoStruct() {}

// ContainerStruct1 is a list type for testing.
type ListElemStruct1 struct {
	Key1   *string              `path:"key1"`
	Outer  *OuterContainerType1 `path:"outer"`
	Outer2 *OuterContainerType1 `path:"outer2"`
}

// IsYANGGoStruct implements the GoStruct interface method.
func (*ListElemStruct1) IsYANGGoStruct() {}

// ContainerStruct1 is a container type for testing.
type ContainerStruct1 struct {
	StructKeyList map[string]*ListElemStruct1 `path:"config/simple-key-list"`
}

// IsYANGGoStruct implements the GoStruct interface method.
func (*ContainerStruct1) IsYANGGoStruct() {}

func TestGetNodesSimpleKeyedList(t *testing.T) {
	containerWithLeafListSchema := &yang.Entry{
		Name: "container",
		Kind: yang.DirectoryEntry,
		Dir: map[string]*yang.Entry{
			"config": {
				Name: "config",
				Kind: yang.DirectoryEntry,
				Dir: map[string]*yang.Entry{
					"simple-key-list": {
						Name:     "simple-key-list",
						Kind:     yang.DirectoryEntry,
						ListAttr: &yang.ListAttr{MinElements: &yang.Value{Name: "0"}},
						Key:      "key1",
						Config:   yang.TSTrue,
						Dir: map[string]*yang.Entry{
							"key1": {
								Name: "key1",
								Kind: yang.LeafEntry,
								Type: &yang.YangType{Kind: yang.Ystring},
							},
							"outer": {
								Name: "outer",
								Kind: yang.DirectoryEntry,
								Dir: map[string]*yang.Entry{
									"inner": {
										Name: "inner",
										Kind: yang.DirectoryEntry,
										Dir: map[string]*yang.Entry{
											"leaf-field": {
												Name: "leaf-field",
												Kind: yang.LeafEntry,
												Type: &yang.YangType{
													Kind: yang.Yleafref,
													Path: "../../config/inner/leaf-field",
												},
											},
										},
									},
									"inner-abs-path": {
										Name: "inner-abs-path",
										Kind: yang.DirectoryEntry,
										Dir: map[string]*yang.Entry{
											"leaf-field": {
												Name: "leaf-field",
												Kind: yang.LeafEntry,
												Type: &yang.YangType{
													Kind: yang.Yleafref,
													Path: "/config/inner/leaf-field",
												},
											},
										},
									},
									"config": {
										Name: "config",
										Kind: yang.DirectoryEntry,
										Dir: map[string]*yang.Entry{
											"inner": {
												Name: "inner",
												Kind: yang.DirectoryEntry,
												Dir: map[string]*yang.Entry{
													"leaf-field": {
														Name: "leaf-field",
														Kind: yang.LeafEntry,
														Type: &yang.YangType{Kind: yang.Yint32},
													},
												},
											},
										},
									},
								},
							},
							"outer2": {
								Name: "outer2",
								Kind: yang.DirectoryEntry,
								Dir: map[string]*yang.Entry{
									"inner": {
										Name: "inner",
										Kind: yang.DirectoryEntry,
										Dir: map[string]*yang.Entry{
											"leaf-field": {
												Name: "leaf-field",
												Kind: yang.LeafEntry,
												Type: &yang.YangType{Kind: yang.Yint32},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	c1 := &ContainerStruct1{
		StructKeyList: map[string]*ListElemStruct1{
			"forty-two": {
				Key1: String("forty-two"),
				Outer: &OuterContainerType1{
					Inner:        &InnerContainerType1{LeafName: Int32(1234)},
					InnerAbsPath: &InnerContainerType1{LeafName: Int32(4321)},
				},
			},
		},
	}

	tests := []struct {
		desc       string
		rootStruct interface{}
		path       *gpb.Path
		want       interface{}
		wantErr    string
	}{
		{
			desc:       "success leaf-ref",
			rootStruct: c1,
			path: &gpb.Path{
				Elem: []*gpb.PathElem{
					{
						Name: "config",
					},
					{
						Name: "simple-key-list",
						Key: map[string]string{
							"key1": "forty-two",
						},
					},
					{
						Name: "outer",
					},
					{
						Name: "inner",
					},
					{
						Name: "leaf-field",
					},
				},
			},
			want: []interface{}{c1.StructKeyList["forty-two"].Outer.Inner.LeafName},
		},
		{
			desc:       "success absolute leaf-ref",
			rootStruct: c1,
			path: &gpb.Path{
				Elem: []*gpb.PathElem{
					{
						Name: "config",
					},
					{
						Name: "simple-key-list",
						Key: map[string]string{
							"key1": "forty-two",
						},
					},
					{
						Name: "outer",
					},
					{
						Name: "inner-abs-path",
					},
					{
						Name: "leaf-field",
					},
				},
			},
			want: []interface{}{c1.StructKeyList["forty-two"].Outer.InnerAbsPath.LeafName},
		},
		{
			desc:       "success leaf full path",
			rootStruct: c1,
			path: &gpb.Path{
				Elem: []*gpb.PathElem{
					{
						Name: "config",
					},
					{
						Name: "simple-key-list",
						Key: map[string]string{
							"key1": "forty-two",
						},
					},
					{
						Name: "outer",
					},
					{
						Name: "config",
					},
					{
						Name: "inner",
					},
					{
						Name: "leaf-field",
					},
				},
			},
			want: []interface{}{c1.StructKeyList["forty-two"].Outer.Inner.LeafName},
		},
		{
			desc:       "bad path",
			rootStruct: c1,
			path: &gpb.Path{
				Elem: []*gpb.PathElem{
					{
						Name: "config",
					},
					{
						Name: "simple-key-list",
						Key: map[string]string{
							"key1": "forty-two",
						},
					},
					{
						Name: "bad-element",
					},
					{
						Name: "inner",
					},
					{
						Name: "leaf-field",
					},
				},
			},
			want:    nil,
			wantErr: `could not find path in tree beyond schema node simple-key-list, (type *util.ListElemStruct1), remaining path elem:<name:"bad-element" > elem:<name:"inner" > elem:<name:"leaf-field" > `,
		},
		{
			desc:       "nil source field",
			rootStruct: c1,
			path: &gpb.Path{
				Elem: []*gpb.PathElem{
					{
						Name: "config",
					},
					{
						Name: "simple-key-list",
						Key: map[string]string{
							"key1": "forty-two",
						},
					},
					{
						Name: "outer2",
					},
					{
						Name: "inner",
					},
					{
						Name: "leaf-field",
					},
				},
			},
			want: []interface{}(nil),
		},
		{
			desc:       "missing key name",
			rootStruct: c1,
			path: &gpb.Path{
				Elem: []*gpb.PathElem{
					{
						Name: "config",
					},
					{
						Name: "simple-key-list",
						Key: map[string]string{
							"bad-key": "forty-two",
						},
					},
					{
						Name: "outer2",
					},
					{
						Name: "inner",
					},
					{
						Name: "leaf-field",
					},
				},
			},
			want:    []interface{}(nil),
			wantErr: `gnmi path elem:<name:"simple-key-list" key:<key:"bad-key" value:"forty-two" > > elem:<name:"outer2" > elem:<name:"inner" > elem:<name:"leaf-field" >  does not contain a map entry for the schema key field name key1, parent type map[string]*util.ListElemStruct1`,
		},
		{
			desc:       "missing key value",
			rootStruct: c1,
			path: &gpb.Path{
				Elem: []*gpb.PathElem{
					{
						Name: "config",
					},
					{
						Name: "simple-key-list",
						Key: map[string]string{
							"key1": "bad-value",
						},
					},
					{
						Name: "outer2",
					},
					{
						Name: "inner",
					},
					{
						Name: "leaf-field",
					},
				},
			},
			want: []interface{}(nil),
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			val, _, err := GetNodes(containerWithLeafListSchema, tt.rootStruct, tt.path)
			if got, want := errToString(err), tt.wantErr; got != want {
				t.Errorf("%s: got error: %s, want error: %s", tt.desc, got, want)
			}
			testErrLog(t, tt.desc, err)
			if err == nil {
				if got, want := val, tt.want; !reflect.DeepEqual(got, want) {
					t.Errorf("%s: struct got:\n%v\nwant:\n%v\n", tt.desc, pretty.Sprint(got), pretty.Sprint(want))
				}
			}
		})
	}
}

// InnerContainerType2 is a container type for testing.
type InnerContainerType2 struct {
	LeafName *int32 `path:"leaf-field"`
}

// IsYANGGoStruct implements the GoStruct interface method.
func (*InnerContainerType2) IsYANGGoStruct() {}

// OuterContainerType2 is a container type for testing.
type OuterContainerType2 struct {
	Inner *InnerContainerType2 `path:"inner"`
}

// IsYANGGoStruct implements the GoStruct interface method.
func (*OuterContainerType2) IsYANGGoStruct() {}

// KeyStruct2 is a key type for testing.
type KeyStruct2 struct {
	Key1 string
	Key2 int32
}

// ListElemStruct2 is a list type for testing.
type ListElemStruct2 struct {
	Key1  *string              `path:"key1"`
	Key2  *int32               `path:"key2"`
	Outer *OuterContainerType2 `path:"outer"`
}

// IsYANGGoStruct implements the GoStruct interface method.
func (*ListElemStruct2) IsYANGGoStruct() {}

// ContainerStruct2 is a container type for testing.
type ContainerStruct2 struct {
	StructKeyList map[KeyStruct2]*ListElemStruct2 `path:"struct-key-list"`
}

// IsYANGGoStruct implements the GoStruct interface method.
func (*ContainerStruct2) IsYANGGoStruct() {}

func TestGetNodesStructKeyedList(t *testing.T) {
	containerWithLeafListSchema := &yang.Entry{
		Name: "container",
		Kind: yang.DirectoryEntry,
		Dir: map[string]*yang.Entry{
			"struct-key-list": {
				Name:     "struct-key-list",
				Kind:     yang.DirectoryEntry,
				ListAttr: &yang.ListAttr{MinElements: &yang.Value{Name: "0"}},
				Key:      "key1 key2",
				Config:   yang.TSTrue,
				Dir: map[string]*yang.Entry{
					"key1": {
						Name: "key1",
						Kind: yang.LeafEntry,
						Type: &yang.YangType{Kind: yang.Ystring},
					},
					"key2": {
						Name: "key2",
						Kind: yang.LeafEntry,
						Type: &yang.YangType{Kind: yang.Yint32},
					},
					"outer": {
						Name: "outer",
						Kind: yang.DirectoryEntry,
						Dir: map[string]*yang.Entry{
							"inner": {
								Name: "inner",
								Kind: yang.DirectoryEntry,
								Dir: map[string]*yang.Entry{
									"leaf-field": {
										Name: "leaf-field",
										Kind: yang.LeafEntry,
										Type: &yang.YangType{Kind: yang.Yint32},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	c1 := &ContainerStruct2{
		StructKeyList: map[KeyStruct2]*ListElemStruct2{
			{"forty-two", 42}: {
				Key1:  String("forty-two"),
				Key2:  Int32(42),
				Outer: &OuterContainerType2{Inner: &InnerContainerType2{LeafName: Int32(1234)}},
			},
			{"forty-three", 43}: {
				Key1:  String("forty-three"),
				Key2:  Int32(43),
				Outer: &OuterContainerType2{Inner: &InnerContainerType2{LeafName: Int32(4321)}},
			},
		},
	}

	// Note that error cases exercise the same logic as simple key test above,
	// hence they are omitted here.
	tests := []struct {
		desc       string
		rootStruct interface{}
		path       *gpb.Path
		want       []interface{}
		wantErr    string
	}{
		{
			desc:       "success leaf",
			rootStruct: c1,
			path: &gpb.Path{
				Elem: []*gpb.PathElem{
					{
						Name: "struct-key-list",
						Key: map[string]string{
							"key1": "forty-two",
							"key2": "42",
						},
					},
					{
						Name: "outer",
					},
					{
						Name: "inner",
					},
					{
						Name: "leaf-field",
					},
				},
			},
			want: []interface{}{c1.StructKeyList[KeyStruct2{"forty-two", 42}].Outer.Inner.LeafName},
		},
		{
			desc:       "success container",
			rootStruct: c1,
			path: &gpb.Path{
				Elem: []*gpb.PathElem{
					{
						Name: "struct-key-list",
						Key: map[string]string{
							"key1": "forty-two",
							"key2": "42",
						},
					},
					{
						Name: "outer",
					},
					{
						Name: "inner",
					},
				},
			},
			want: []interface{}{c1.StructKeyList[KeyStruct2{"forty-two", 42}].Outer.Inner},
		},
		{
			desc:       "empty key value",
			rootStruct: c1,
			path: &gpb.Path{
				Elem: []*gpb.PathElem{
					{
						Name: "struct-key-list",
					},
					{
						Name: "outer",
					},
					{
						Name: "inner",
					},
					{
						Name: "leaf-field",
					},
				},
			},
			want: []interface{}{
				c1.StructKeyList[KeyStruct2{"forty-two", 42}].Outer.Inner.LeafName,
				c1.StructKeyList[KeyStruct2{"forty-three", 43}].Outer.Inner.LeafName,
			},
		},
		{
			desc:       "partial key value",
			rootStruct: c1,
			path: &gpb.Path{
				Elem: []*gpb.PathElem{
					{
						Name: "struct-key-list",
						Key: map[string]string{
							"key2": "42",
						},
					},
					{
						Name: "outer",
					},
					{
						Name: "inner",
					},
					{
						Name: "leaf-field",
					},
				},
			},
			want: []interface{}{c1.StructKeyList[KeyStruct2{"forty-two", 42}].Outer.Inner.LeafName},
		},
		{
			desc:       "bad key value",
			rootStruct: c1,
			path: &gpb.Path{
				Elem: []*gpb.PathElem{
					{
						Name: "struct-key-list",
						Key: map[string]string{
							"key1": "bad-value",
							"key2": "42",
						},
					},
					{
						Name: "outer",
					},
					{
						Name: "inner",
					},
					{
						Name: "leaf-field",
					},
				},
			},
			want: []interface{}{},
		},
		{
			desc:       "bad path element",
			rootStruct: c1,
			path: &gpb.Path{
				Elem: []*gpb.PathElem{
					{
						Name: "struct-key-list",
						Key: map[string]string{
							"key1": "forty-two",
							"key2": "42",
						},
					},
					{
						Name: "bad-path-element",
					},
					{
						Name: "inner",
					},
					{
						Name: "leaf-field",
					},
				},
			},
			wantErr: `could not find path in tree beyond schema node struct-key-list, (type *util.ListElemStruct2), remaining path elem:<name:"bad-path-element" > elem:<name:"inner" > elem:<name:"leaf-field" > `,
		},
	}

	for _, tt := range tests {
		val, _, err := GetNodes(containerWithLeafListSchema, tt.rootStruct, tt.path)
		if got, want := errToString(err), tt.wantErr; got != want {
			t.Errorf("%s: got error: %s, want error: %s", tt.desc, got, want)
		}
		testErrLog(t, tt.desc, err)
		if err == nil {
			if got, want := sliceToMap(val), sliceToMap(tt.want); (len(want) != 0 || len(got) != 0) && !reflect.DeepEqual(got, want) {
				t.Errorf("%s: struct got:\n%v\nwant:\n%v\n", tt.desc, pretty.Sprint(got), pretty.Sprint(want))
			}
		}
	}
}

func TestDeepEqualDerefPtrs(t *testing.T) {
	a, b := 42, 42
	if !DeepEqualDerefPtrs(&a, &b) {
		t.Fatalf("DeepEqualDerefPtrs: expect that %v == %v", a, b)
	}
}

func sliceToMap(s []interface{}) map[string]int {
	m := make(map[string]int)
	for _, v := range s {
		vs := fmt.Sprint(v)
		m[vs] = m[vs] + 1
	}
	return m
}
