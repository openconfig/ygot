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
	"github.com/openconfig/ygot/ygot"
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
			fieldValue:   ygot.Int8(42),
			wantVal:      &BasicStruct{IntPtrField: ygot.Int8(42)},
		},
		{
			desc:         "nil int ptr",
			parentStruct: &BasicStruct{IntPtrField: ygot.Int8(42)},
			fieldName:    "IntPtrField",
			fieldValue:   nil,
			wantVal:      &BasicStruct{},
		},
		{
			desc:         "string ptr",
			parentStruct: &BasicStruct{},
			fieldName:    "StringPtrField",
			fieldValue:   ygot.String("forty two"),
			wantVal:      &BasicStruct{StringPtrField: ygot.String("forty two")},
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
			fieldValue:   ygot.Int8(42),
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
			parentStruct: &BasicStruct{IntPtrSliceField: []*int8{ygot.Int8(42)}},
			fieldName:    "IntPtrSliceField",
			fieldValue:   ygot.Int8(43),
			wantVal:      &BasicStruct{IntPtrSliceField: []*int8{ygot.Int8(42), ygot.Int8(43)}},
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
			fieldValue:   42,
			wantErr:      "parent type *util.BasicStruct does not have a field name MissingField",
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

func TestInsertIntoMapStructField(t *testing.T) {
	type KeyStruct struct {
		IntField int
	}

	type BasicStruct struct {
		StringToIntMapField    map[string]int
		StringToIntPtrMapField map[string]*int8
		StructToIntMapField    map[KeyStruct]int
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
			fieldValue:   ygot.Int8(42),
			wantVal:      &BasicStruct{StringToIntPtrMapField: map[string]*int8{"forty-two": ygot.Int8(42)}},
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
			wantVal:      &BasicStruct{StructToIntMapField: map[KeyStruct]int{KeyStruct{IntField: 42}: 42}},
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

func TestForEachField(t *testing.T) {
	type BasicStruct struct {
		Int32Field     int32
		StringField    string
		Int32PtrField  *int32
		StringPtrField *string
	}

	type StructOfStructs struct {
		BasicStructField    BasicStruct
		BasicStructPtrField *BasicStruct
	}

	type StructOfSliceOfStructs struct {
		BasicStructSliceField    []BasicStruct
		BasicStructPtrSliceField []*BasicStruct
	}

	type StructOfMapOfStructs struct {
		BasicStructMapField    map[string]BasicStruct
		BasicStructPtrMapField map[string]*BasicStruct
	}

	printFieldsIterFunc := func(ni *NodeInfo, in, out interface{}) (errs []error) {
		// Only print basic scalar values, skip everything else.
		if !IsValueScalar(ni.FieldValue) || IsValueNil(ni.FieldKey) {
			return
		}
		outs := out.(*string)
		*outs += fmt.Sprintf("%v : %v, ", ni.FieldType.Name, pretty.Sprint(ni.FieldValue.Interface()))
		return
	}

	printMapKeysIterFunc := func(ni *NodeInfo, in, out interface{}) (errs []error) {
		// Only print basic scalar values, skip everything else.
		if !IsValueScalar(ni.FieldValue) || IsNilOrInvalidValue(ni.FieldKey) {
			return
		}
		outs := out.(*string)
		s := "nil"
		if !IsNilOrInvalidValue(ni.FieldValue) {
			s = pretty.Sprint(ni.FieldValue.Interface())
		}
		*outs += fmt.Sprintf("%s/%s : %s, ", pretty.Sprint(ni.FieldKey.Interface()), ni.FieldType.Name, s)
		return
	}

	basicStruct1 := BasicStruct{Int32Field: int32(42), StringField: "forty two", Int32PtrField: ygot.Int32(4242), StringPtrField: ygot.String("forty two ptr")}
	basicStruct2 := BasicStruct{Int32Field: int32(43), StringField: "forty three", Int32PtrField: ygot.Int32(4343), StringPtrField: ygot.String("forty three ptr")}

	tests := []struct {
		desc         string
		parentStruct interface{}
		in           interface{}
		out          interface{}
		iterFunc     FieldIteratorFunc
		wantOut      string
		wantErr      string
	}{
		{
			desc:         "nil",
			parentStruct: nil,
			in:           nil,
			iterFunc:     printFieldsIterFunc,
			wantOut:      ``,
		},
		{
			desc:         "struct",
			parentStruct: &basicStruct1,
			in:           nil,
			iterFunc:     printFieldsIterFunc,
			wantOut:      `Int32Field : 42, StringField : "forty two", Int32PtrField : 4242, StringPtrField : "forty two ptr", `,
		},
		{
			desc:         "struct of struct",
			parentStruct: &StructOfStructs{BasicStructField: basicStruct1, BasicStructPtrField: &basicStruct2},
			in:           nil,
			iterFunc:     printFieldsIterFunc,
			wantOut: `Int32Field : 42, StringField : "forty two", Int32PtrField : 4242, StringPtrField : "forty two ptr", ` +
				`Int32Field : 43, StringField : "forty three", Int32PtrField : 4343, StringPtrField : "forty three ptr", `,
		},
		{
			desc:         "struct of slice of structs",
			parentStruct: &StructOfSliceOfStructs{BasicStructSliceField: []BasicStruct{basicStruct1}, BasicStructPtrSliceField: []*BasicStruct{&basicStruct2}},
			in:           nil,
			iterFunc:     printFieldsIterFunc,
			wantOut:      `Int32Field : 42, StringField : "forty two", Int32PtrField : 4242, StringPtrField : "forty two ptr", Int32Field : 43, StringField : "forty three", Int32PtrField : 4343, StringPtrField : "forty three ptr", `,
		},
		{
			desc:         "struct of map of structs",
			parentStruct: &StructOfMapOfStructs{BasicStructMapField: map[string]BasicStruct{"basicStruct1": basicStruct1}, BasicStructPtrMapField: map[string]*BasicStruct{"basicStruct2": &basicStruct2}},
			in:           nil,
			iterFunc:     printFieldsIterFunc,
			wantOut:      `Int32Field : 42, StringField : "forty two", Int32PtrField : 4242, StringPtrField : "forty two ptr", Int32Field : 43, StringField : "forty three", Int32PtrField : 4343, StringPtrField : "forty three ptr", `,
		},
		{
			desc:         "map keys",
			parentStruct: &StructOfMapOfStructs{BasicStructMapField: map[string]BasicStruct{"basicStruct1": basicStruct1}, BasicStructPtrMapField: map[string]*BasicStruct{"basicStruct2": &basicStruct2}},
			in:           nil,
			iterFunc:     printMapKeysIterFunc,
			wantOut: `"basicStruct1"/Int32Field : 42, "basicStruct1"/StringField : "forty two", "basicStruct1"/Int32PtrField : 4242, "basicStruct1"/StringPtrField : "forty two ptr", ` +
				`"basicStruct2"/Int32Field : 43, "basicStruct2"/StringField : "forty three", "basicStruct2"/Int32PtrField : 4343, "basicStruct2"/StringPtrField : "forty three ptr", `,
		},
	}

	for _, tt := range tests {
		outStr := ""
		var errs Errors
		errs = ForEachField(tt.parentStruct, tt.in, &outStr, tt.iterFunc)
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
		Int32Field     int32
		StringField    string
		Int32PtrField  *int32
		StringPtrField *string
	}

	type StructOfStructs struct {
		BasicStructField *BasicStruct
	}

	basicStruct1 := BasicStruct{Int32Field: int32(42), StringField: "forty two", Int32PtrField: ygot.Int32(4242), StringPtrField: ygot.String("forty two ptr")}

	// This doesn't work as a general insert because it won't create fields
	// that are nil, they must already exist. It only works as an update.
	setFunc := func(ni *NodeInfo, in, out interface{}) (errs []error) {
		if ni.FieldType.Name == "BasicStructField" {
			errs = AppendErr(errs, UpdateField(ni.ParentStruct, "BasicStructField", &basicStruct1))
		}
		return
	}

	a := StructOfStructs{BasicStructField: &BasicStruct{}}

	if errs := ForEachField(&a, nil, nil, setFunc); errs != nil {
		t.Fatalf("setFunc got unexpected error: %s", errs)
	}

	if got, want := *a.BasicStructField, basicStruct1; got != want {
		t.Errorf("set struct: got: %s, want: %s", pretty.Sprint(got), pretty.Sprint(want))
	}
}
