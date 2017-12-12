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
	"testing"

	"github.com/kylelemons/godebug/pretty"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/util"
	"github.com/openconfig/ygot/ygot"
)

var (
	// testErrOutput controls whether expect error test cases log the error
	// values.
	testErrOutput = false
	globalEnumMap = map[string]map[int64]ygot.EnumDefinition{
		"EnumType": {
			42: {Name: "E_VALUE_FORTY_TWO"},
		},
		"EnumType2": {
			43: {Name: "E_VALUE_FORTY_THREE"},
		},
	}
)

// EnumType is used as an enum type in various tests in the ytypes package.
type EnumType int64

func (EnumType) ΛMap() map[string]map[int64]ygot.EnumDefinition {
	return globalEnumMap
}

// EnumType2 is used as an enum type in various tests in the ytypes package.
type EnumType2 int64

func (EnumType2) ΛMap() map[string]map[int64]ygot.EnumDefinition {
	return globalEnumMap
}

// populateParentField recurses through schema and populates each Parent field
// with the parent schema node ptr.
func populateParentField(parent, schema *yang.Entry) {
	schema.Parent = parent
	for _, e := range schema.Dir {
		populateParentField(schema, e)
	}
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
	if util.IsValueNil(a) && util.IsValueNil(b) {
		return true
	}
	va, vb := reflect.ValueOf(a), reflect.ValueOf(b)
	if va.Kind() == reflect.Ptr && vb.Kind() == reflect.Ptr {
		return reflect.DeepEqual(va.Elem().Interface(), vb.Elem().Interface())
	}

	return reflect.DeepEqual(a, b)
}

func TestValidateListAttr(t *testing.T) {
	validLeafListSchemaMin1 := &yang.Entry{
		Name:     "min1",
		Kind:     yang.LeafEntry,
		Type:     &yang.YangType{Kind: yang.Ystring},
		ListAttr: &yang.ListAttr{MinElements: &yang.Value{Name: "1"}},
	}
	validLeafListSchemaMax3 := &yang.Entry{
		Name:     "max3",
		Kind:     yang.LeafEntry,
		Type:     &yang.YangType{Kind: yang.Ystring},
		ListAttr: &yang.ListAttr{MaxElements: &yang.Value{Name: "3"}},
	}
	validLeafListSchemaMin1Max3 := &yang.Entry{
		Name:     "min1max3",
		Kind:     yang.LeafEntry,
		Type:     &yang.YangType{Kind: yang.Ystring},
		ListAttr: &yang.ListAttr{MinElements: &yang.Value{Name: "1"}, MaxElements: &yang.Value{Name: "3"}},
	}
	invalidLeafListSchemaNoAttr := &yang.Entry{
		Name: "no_attr",
		Kind: yang.LeafEntry,
		Type: &yang.YangType{Kind: yang.Ystring},
	}
	invalidLeafListSchemaBadRange := &yang.Entry{
		Name:     "bad_range",
		Kind:     yang.LeafEntry,
		Type:     &yang.YangType{Kind: yang.Ystring},
		ListAttr: &yang.ListAttr{MinElements: &yang.Value{Name: "bad"}},
	}
	invalidLeafListSchemaNegativeMinRange := &yang.Entry{
		Name:     "negative_min_range",
		Kind:     yang.LeafEntry,
		Type:     &yang.YangType{Kind: yang.Ystring},
		ListAttr: &yang.ListAttr{MinElements: &yang.Value{Name: "-1"}},
	}
	invalidLeafListSchemaNegativeMaxRange := &yang.Entry{
		Name:     "negative_min_range",
		Kind:     yang.LeafEntry,
		Type:     &yang.YangType{Kind: yang.Ystring},
		ListAttr: &yang.ListAttr{MaxElements: &yang.Value{Name: "-1"}},
	}

	tests := []struct {
		desc    string
		schema  *yang.Entry
		value   interface{}
		wantErr bool
	}{
		{
			desc:    "nil schema",
			schema:  nil,
			wantErr: true,
		},
		{
			desc:    "missing ListAttr",
			schema:  invalidLeafListSchemaNoAttr,
			wantErr: true,
		},
		{
			desc:    "bad range value",
			schema:  invalidLeafListSchemaBadRange,
			wantErr: true,
		},
		{
			desc:    "negative min range value",
			schema:  invalidLeafListSchemaNegativeMinRange,
			wantErr: true,
		},
		{
			desc:    "negative max range value",
			schema:  invalidLeafListSchemaNegativeMaxRange,
			wantErr: true,
		},
		{
			desc:    "bad value type",
			schema:  validLeafListSchemaMin1,
			value:   int(1),
			wantErr: true,
		},
		{
			desc:   "min elements success",
			schema: validLeafListSchemaMin1,
			value:  []string{"a"},
		},
		{
			desc:    "min elements too few",
			schema:  validLeafListSchemaMin1,
			value:   []string{},
			wantErr: true,
		},
		{
			desc:    "min elements too few, nil value",
			schema:  validLeafListSchemaMin1,
			value:   nil,
			wantErr: true,
		},
		{
			desc:   "max elements success",
			schema: validLeafListSchemaMax3,
			value:  []string{"a"},
		},
		{
			desc:    "max elements too many",
			schema:  validLeafListSchemaMax3,
			value:   []string{"a", "b", "c", "d"},
			wantErr: true,
		},
		{
			desc:   "min/max elements success",
			schema: validLeafListSchemaMin1Max3,
			value:  []string{"a"},
		},
		{
			desc:    "min/max elements too few",
			schema:  validLeafListSchemaMin1Max3,
			value:   []string{},
			wantErr: true,
		},
		{
			desc:    "min/max elements too many",
			schema:  validLeafListSchemaMax3,
			value:   []string{"a", "b", "c", "d"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := validateListAttr(tt.schema, tt.value)
			// TODO(mostrowski): make consistent with rest of structs library.
			if got, want := (err != nil), tt.wantErr; got != want {
				t.Errorf("%s: TestValidateListAttr(%v) got error: %v, want error? %v", tt.desc, tt.schema, err, tt.wantErr)
			}
			if err != nil {
				if testErrOutput {
					t.Logf("%s: %v", tt.desc, err)
				}
			}
		})
	}
}

func TestIsFakeRoot(t *testing.T) {
	tests := []struct {
		desc string
		in   *yang.Entry
		want bool
	}{
		{
			desc: "explicitly true",
			in: &yang.Entry{
				Name: "entry",
				Annotation: map[string]interface{}{
					"isFakeRoot": true,
				},
			},
			want: true,
		},
		{
			desc: "unspecified",
			in: &yang.Entry{
				Name: "entry",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if got := util.IsFakeRoot(tt.in); got != tt.want {
				t.Errorf("%v: isFakeRoot(%v): did not get expected return value, got: %v, want: %v", tt.desc, tt.in, got, tt.want)
			}
		})
	}
}

type StringListElemStruct struct {
	LeafName *string `path:"string"`
}

func (c *StringListElemStruct) IsYANGGoStruct() {}

type ComplexStruct struct {
	List1       []*StringListElemStruct `path:"list1"`
	Case1Leaf1  *string                 `path:"case1-leaf1"`
	Case21Leaf1 *string                 `path:"case21-leaf"`
}

func (c *ComplexStruct) IsYANGGoStruct() {}

func TestForEachSchemaNode(t *testing.T) {
	complexSchema := &yang.Entry{
		Name: "complex-schema",
		Kind: yang.DirectoryEntry,
		Dir: map[string]*yang.Entry{
			"list1": {
				Kind:     yang.DirectoryEntry,
				Name:     "list1",
				ListAttr: &yang.ListAttr{MinElements: &yang.Value{Name: "0"}},
				Dir: map[string]*yang.Entry{
					"string": {
						Kind: yang.LeafEntry,
						Name: "string",
						Type: &yang.YangType{Kind: yang.Ystring},
					},
				},
			},
			"choice1": {
				Kind: yang.ChoiceEntry,
				Name: "choice1",
				Dir: map[string]*yang.Entry{
					"case1": {
						Kind: yang.CaseEntry,
						Name: "case1",
						Dir: map[string]*yang.Entry{
							"case1-leaf1": {
								Kind: yang.LeafEntry,
								Name: "case1-leaf1",
								Type: &yang.YangType{Kind: yang.Ystring},
							},
						},
					},
					"case2": {
						Kind: yang.CaseEntry,
						Name: "case2",
						Dir: map[string]*yang.Entry{
							"case2_choice1": {
								Kind: yang.ChoiceEntry,
								Name: "case2_choice1",
								Dir: map[string]*yang.Entry{
									"case21": {
										Kind: yang.CaseEntry,
										Name: "case21",
										Dir: map[string]*yang.Entry{
											"case21-leaf": {
												Kind: yang.LeafEntry,
												Name: "case21-leaf",
												Type: &yang.YangType{Kind: yang.Ystring},
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

	printFieldsIterFunc := func(ni *util.NodeInfo, in, out interface{}) (errs util.Errors) {
		// Only print basic scalar values, skip everything else.
		outs := out.(*string)
		*outs += fmt.Sprintf("%v : %v\n", ni.StructField.Name, pretty.Sprint(ni.FieldValue.Interface()))
		return
	}

	val := &ComplexStruct{
		List1:      []*StringListElemStruct{{LeafName: ygot.String("elem1_leaf_name")}},
		Case1Leaf1: ygot.String("Case1Leaf1Value"),
	}

	var outStr string
	var errs util.Errors = util.ForEachField(complexSchema, val, nil, &outStr, printFieldsIterFunc)
	if errs != nil {
		t.Errorf("ForEachSchemaNode: got error: %s, want error nil", errs)
	}
	testErrLog(t, "ForEachSchemaNode", errs)
	wantStr := ` : {List1:       [{LeafName: "elem1_leaf_name"}],
 Case1Leaf1:  "Case1Leaf1Value",
 Case21Leaf1: nil}
List1 : [{LeafName: "elem1_leaf_name"}]
List1 : {LeafName: "elem1_leaf_name"}

`
	if outStr == wantStr {
		t.Errorf("ForEachSchemaNode: got\n%s\nwant\n%s\n", outStr, wantStr)
	}
}
