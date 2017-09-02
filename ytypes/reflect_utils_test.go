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
	"strings"
	"testing"

	"github.com/kylelemons/godebug/pretty"
	"github.com/openconfig/ygot/ygot"
)

const (
	// wildcardStr is a wildcard string that matches any one word in a string.
	wildcardStr = "{{*}}"
)

// errToString returns the string representation of err and the empty string if
// err is nil.
func errToString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// areEqual compares a and b. If a and b are both pointers, it compares the
// values they are pointing to.
func areEqual(a, b interface{}) bool {
	if isNil(a) && isNil(b) {
		return true
	}
	va, vb := reflect.ValueOf(a), reflect.ValueOf(b)
	if va.Kind() == reflect.Ptr && vb.Kind() == reflect.Ptr {
		return va.Elem().Interface() == vb.Elem().Interface()
	}

	return a == b
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
			wantVal:      42,
		},
		{
			desc:         "int with nil",
			parentStruct: &BasicStruct{},
			fieldName:    "IntField",
			fieldValue:   nil,
			wantErr:      "cannot assign value <nil> (type <nil>) to field IntField (type int) in struct BasicStruct",
		},
		{
			desc:         "nil parent",
			parentStruct: nil,
			fieldName:    "IntField",
			fieldValue:   42,
			wantErr:      "parentStruct is nil in UpdateField for field IntField, value 42",
		},
		{
			desc:         "string",
			parentStruct: &BasicStruct{},
			fieldName:    "StringField",
			fieldValue:   "forty two",
			wantVal:      "forty two",
		},
		{
			desc:         "nil parent struct",
			parentStruct: nil,
			fieldName:    "IntField",
			fieldValue:   42,
			wantErr:      "parentStruct is nil in UpdateField for field IntField, value 42",
		},
		{
			desc:         "string to int field error",
			parentStruct: &BasicStruct{},
			fieldName:    "IntField",
			fieldValue:   "forty two",
			wantErr:      "cannot assign value forty two (type string) to field IntField (type int) in struct BasicStruct",
		},
		{
			desc:         "int ptr",
			parentStruct: &BasicStruct{},
			fieldName:    "IntPtrField",
			fieldValue:   ygot.Int8(42),
			wantVal:      ygot.Int8(42),
		},
		{
			desc:         "nil int ptr",
			parentStruct: &BasicStruct{},
			fieldName:    "IntPtrField",
			fieldValue:   nil,
			wantVal:      nil,
		},
		{
			desc:         "string ptr",
			parentStruct: &BasicStruct{},
			fieldName:    "StringPtrField",
			fieldValue:   ygot.String("forty two"),
			wantVal:      ygot.String("forty two"),
		},
		{
			desc:         "int to int ptr field error",
			parentStruct: &BasicStruct{},
			fieldName:    "IntPtrField",
			fieldValue:   42,
			wantErr:      "cannot assign value 42 (type int) to field IntPtrField (type ptr) in struct BasicStruct",
		},
		{
			desc:         "int ptr to int field error",
			parentStruct: &BasicStruct{},
			fieldName:    "IntField",
			fieldValue:   ygot.Int8(42),
			wantErr:      "cannot assign value " + wildcardStr + " (type *int8) to field IntField (type int) in struct BasicStruct",
		},
		{
			desc:         "struct",
			parentStruct: &StructOfStructs{},
			fieldName:    "BasicStructField",
			fieldValue:   &BasicStruct{IntField: 42, StringField: "forty two"},
			wantVal:      &BasicStruct{IntField: 42, StringField: "forty two"},
		},
		{
			desc:         "struct bad field name",
			parentStruct: &StructOfStructs{},
			fieldName:    "StructBadField",
			fieldValue:   &BasicStruct{IntField: 42, StringField: "forty two"},
			wantErr:      "no field named StructBadField in struct StructOfStructs",
		},
		{
			desc:         "struct bad field type",
			parentStruct: &StructOfStructs{},
			fieldName:    "BasicStructField",
			fieldValue:   42,
			wantErr:      "cannot assign value 42 (type int) to field BasicStructField (type ptr) in struct StructOfStructs",
		},
	}

	for _, tt := range tests {
		val, err := UpdateField(tt.parentStruct, tt.fieldName, tt.fieldValue)
		if got, want := errToString(err), tt.wantErr; !areEqualWithWildcards(got, want) {
			t.Errorf("%s: got error: %s, want error: %s", tt.desc, got, want)
		}
		if err == nil {
			if got, want := val, tt.wantVal; !areEqual(got, want) {
				t.Errorf("%s: got value: %v, want value: %v", tt.desc, pretty.Sprint(val), pretty.Sprint(tt.wantVal))
			}
		} else {
			if testErrOutput {
				t.Logf("%s: %v", tt.desc, err)
			}
		}
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

	printFieldsIterFunc := func(parentStruct interface{}, fieldType reflect.StructField, fieldValue reflect.Value, fieldKeys []reflect.Value, fieldKey reflect.Value, in, out interface{}) (errs []error) {
		// Only print basic scalar values, skip everything else.
		if !isValueScalar(fieldValue) || isNil(fieldKey) {
			return
		}
		outs := out.(*string)
		*outs += fmt.Sprintf("%v : %v, ", fieldType.Name, pretty.Sprint(fieldValue.Interface()))
		return
	}

	printMapKeysIterFunc := func(parentStruct interface{}, fieldType reflect.StructField, fieldValue reflect.Value, fieldKeys []reflect.Value, fieldKey reflect.Value, in, out interface{}) (errs []error) {
		// Only print basic scalar values, skip everything else.
		if !isValueScalar(fieldValue) || isNilOrInvalidValue(fieldKey) {
			return
		}
		outs := out.(*string)
		s := "nil"
		if !isNilOrInvalidValue(fieldValue) {
			s = pretty.Sprint(fieldValue.Interface())
		}
		*outs += fmt.Sprintf("%s/%s : %s, ", pretty.Sprint(fieldKey.Interface()), fieldType.Name, s)
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
		} else {
			if testErrOutput {
				t.Logf("%s: %s", tt.desc, errs)
			}
		}
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
	setFunc := func(parentStruct interface{}, fieldType reflect.StructField, fieldValue reflect.Value, fieldKeys []reflect.Value, fieldKey reflect.Value, in, out interface{}) (errs []error) {
		if fieldType.Name == "BasicStructField" {
			_, e := UpdateField(parentStruct, "BasicStructField", &basicStruct1)
			errs = appendErr(errs, e)
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
