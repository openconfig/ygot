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
	"encoding/json"
	"reflect"
	"testing"

	"github.com/kylelemons/godebug/pretty"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygot"
)

type ChoiceStruct struct {
	Case1Leaf1  *string `path:"case1-leaf1"`
	Case21Leaf1 *string `path:"case21-leaf"`
}

func (*ChoiceStruct) IsYANGGoStruct()                          {}
func (*ChoiceStruct) ΛValidate(...ygot.ValidationOption) error { return nil }
func (*ChoiceStruct) ΛEnumTypeMap() map[string][]reflect.Type  { return nil }
func (*ChoiceStruct) ΛBelongingModule() string                 { return "bar" }

type BadChoiceStruct struct {
	StringLeaf    *string `path:"string-leaf"`
	BadNoPathLeaf *int32  ``
}

func (*BadChoiceStruct) IsYANGGoStruct()                          {}
func (*BadChoiceStruct) ΛValidate(...ygot.ValidationOption) error { return nil }
func (*BadChoiceStruct) ΛEnumTypeMap() map[string][]reflect.Type  { return nil }
func (*BadChoiceStruct) ΛBelongingModule() string                 { return "bar" }

type ChoiceEmptyEnumStringStruct struct {
	EmptyLeaf  YANGEmpty `path:"empty-leaf"`
	EnumLeaf   EnumType  `path:"enum-leaf"`
	StringLeaf *string   `path:"string-leaf"`
}

func (*ChoiceEmptyEnumStringStruct) IsYANGGoStruct()                          {}
func (*ChoiceEmptyEnumStringStruct) ΛValidate(...ygot.ValidationOption) error { return nil }
func (*ChoiceEmptyEnumStringStruct) ΛEnumTypeMap() map[string][]reflect.Type  { return nil }
func (*ChoiceEmptyEnumStringStruct) ΛBelongingModule() string                 { return "bar" }

func TestValidateChoice(t *testing.T) {
	containerWithChoiceSchema := &yang.Entry{
		Name: "container-with-choice-schema",
		Kind: yang.DirectoryEntry,
		Dir: map[string]*yang.Entry{
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
								Name: "Case1Leaf1",
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

	containerWithBadChoiceSchema := &yang.Entry{
		Name: "container-with-bad-choice-schema",
		Kind: yang.DirectoryEntry,
		Dir: map[string]*yang.Entry{
			"choice1": {
				Name: "choice1",
				Kind: yang.ChoiceEntry,
				Dir: map[string]*yang.Entry{
					"case1": {
						Name: "case1",
						Kind: yang.CaseEntry,
						Dir: map[string]*yang.Entry{
							"string-leaf": {
								Name: "string-leaf",
								Kind: yang.LeafEntry,
								Type: &yang.YangType{Kind: yang.Ystring},
							},
						},
					},
					"case2": {
						Name: "case2",
						Kind: yang.CaseEntry,
						Dir: map[string]*yang.Entry{
							"bad-no-path-leaf": {
								Name: "bad-no-path-leaf",
								Kind: yang.LeafEntry,
								Type: &yang.YangType{Kind: yang.Yint32},
							},
						},
					},
				},
			},
		},
	}

	containerWithChoiceEmptyEnumStringSchema := &yang.Entry{
		Name: "container-with-choice-empty-enum-string-schema",
		Kind: yang.DirectoryEntry,
		Dir: map[string]*yang.Entry{
			"choice1": {
				Name: "choice1",
				Kind: yang.ChoiceEntry,
				Dir: map[string]*yang.Entry{
					"case1": {
						Name: "case1",
						Kind: yang.CaseEntry,
						Dir: map[string]*yang.Entry{
							"empty-leaf": {
								Name: "empty-leaf",
								Kind: yang.LeafEntry,
								Type: &yang.YangType{Kind: yang.Yempty},
							},
						},
					},
					"case2": {
						Name: "case2",
						Kind: yang.CaseEntry,
						Dir: map[string]*yang.Entry{
							"enum-leaf": {
								Name: "enum-leaf",
								Kind: yang.LeafEntry,
								Type: &yang.YangType{Kind: yang.Yenum},
							},
						},
					},
					"case3": {
						Name: "case3",
						Kind: yang.CaseEntry,
						Dir: map[string]*yang.Entry{
							"string-leaf": {
								Name: "string-leaf",
								Kind: yang.LeafEntry,
								Type: &yang.YangType{Kind: yang.Ystring},
							},
						},
					},
				},
			},
		},
	}

	tests := []struct {
		desc    string
		val     interface{}
		schema  *yang.Entry
		wantErr bool
	}{
		{
			desc:   "success",
			schema: containerWithChoiceSchema,
			val:    &ChoiceStruct{Case1Leaf1: ygot.String("Case1Leaf1Value")},
		},
		{
			desc:    "multiple cases selected",
			schema:  containerWithChoiceSchema,
			val:     &ChoiceStruct{Case1Leaf1: ygot.String("Case1Leaf1Value"), Case21Leaf1: ygot.String("Case21Leaf1Value")},
			wantErr: true,
		},
		{
			desc:   "success - #189 - empty",
			schema: containerWithChoiceEmptyEnumStringSchema,
			val:    &ChoiceEmptyEnumStringStruct{EmptyLeaf: YANGEmpty(true)},
		},
		{
			desc:   "success - #189 - enum",
			schema: containerWithChoiceEmptyEnumStringSchema,
			val:    &ChoiceEmptyEnumStringStruct{EnumLeaf: EnumType(42)},
		},
		{
			desc:   "success - #189 - string & unset enum/empty",
			schema: containerWithChoiceEmptyEnumStringSchema,
			val:    &ChoiceEmptyEnumStringStruct{EmptyLeaf: YANGEmpty(false), EnumLeaf: EnumType(0), StringLeaf: ygot.String("")},
		},
		{
			desc:    "multiple cases selected - #189",
			schema:  containerWithChoiceEmptyEnumStringSchema,
			val:     &ChoiceEmptyEnumStringStruct{EmptyLeaf: YANGEmpty(true), EnumLeaf: EnumType(0), StringLeaf: ygot.String("")},
			wantErr: true,
		},
		{
			desc:    "bad struct missing path tag",
			schema:  containerWithBadChoiceSchema,
			val:     &BadChoiceStruct{BadNoPathLeaf: ygot.Int32(42)},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			errs := Validate(tt.schema, tt.val)
			if got, want := (errs != nil), tt.wantErr; got != want {
				t.Errorf("%s: Validate got error: %s, want error? %v", tt.desc, errs, tt.wantErr)
			}
			testErrLog(t, tt.desc, errs)
		})
	}
}

func TestUnmarshalChoice(t *testing.T) {
	containerWithChoiceSchema := &yang.Entry{
		Name: "container-with-choice",
		Kind: yang.DirectoryEntry,
		Dir: map[string]*yang.Entry{
			"choice1": {
				Kind: yang.ChoiceEntry,
				Name: "choice1",
				Dir: map[string]*yang.Entry{
					"case1": {
						Kind: yang.CaseEntry,
						Name: "case1",
						Dir: map[string]*yang.Entry{
							"leaf11-field": {
								Kind: yang.LeafEntry,
								Name: "leaf11-field",
								Type: &yang.YangType{Kind: yang.Yint32},
							},
						},
					},
					"case2": {
						Kind: yang.CaseEntry,
						Name: "case2",
						Dir: map[string]*yang.Entry{
							"choice1": {
								Kind: yang.ChoiceEntry,
								Name: "choice1",
								Dir: map[string]*yang.Entry{
									"case1": {
										Kind: yang.CaseEntry,
										Name: "case1",
										Dir: map[string]*yang.Entry{
											"leaf1211-field": {
												Kind: yang.LeafEntry,
												Name: "leaf1211-field",
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

	containerSchema := &yang.Entry{
		Name: "parent-field",
		Kind: yang.DirectoryEntry,
		Dir: map[string]*yang.Entry{
			"container-with-choice": containerWithChoiceSchema,
		},
	}

	populateParentField(nil, containerSchema)

	type ContainerWithChoiceStruct struct {
		Leaf11Field   *int32 `path:"choice1/case1/leaf11-field"`
		Leaf1211Field *int32 `path:"choice1/case2/choice1/case1/leaf1211-field"`
	}

	type ParentContainerStruct struct {
		ContainerField *ContainerWithChoiceStruct `path:"container-with-choice"`
	}

	tests := []struct {
		desc    string
		schema  *yang.Entry
		json    string
		want    interface{}
		wantErr string
	}{
		{
			desc:   "success",
			schema: containerSchema,
			json:   `{ "container-with-choice": { "m2:leaf11-field": 42, "m1:leaf1211-field": 43 } }`,
			want:   &ParentContainerStruct{ContainerField: &ContainerWithChoiceStruct{Leaf11Field: ygot.Int32(42), Leaf1211Field: ygot.Int32(43)}},
		},
		{
			desc:    "bad field name",
			schema:  containerSchema,
			json:    `{ "container-with-choice": { "bad-field": 42 } }`,
			wantErr: `parent container container-with-choice (type *ytypes.ContainerWithChoiceStruct): JSON contains unexpected field bad-field`,
		},
		{
			desc:    "bad field type",
			schema:  containerSchema,
			json:    `{ "container-with-choice": { "m2:leaf11-field":  "forty-two"} }`,
			wantErr: `got string type for field leaf11-field, expect float64`,
		},
	}

	var jsonTree interface{}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			var parent ParentContainerStruct

			if err := json.Unmarshal([]byte(tt.json), &jsonTree); err != nil {
				t.Fatalf("json unmarshal (%s) : %s", tt.desc, err)
			}

			err := Unmarshal(tt.schema, &parent, jsonTree)
			if got, want := errToString(err), tt.wantErr; got != want {
				t.Errorf("%s: got error: %v, want error: %v", tt.desc, got, want)
			}
			testErrLog(t, tt.desc, err)
			if err == nil {
				if got, want := &parent, tt.want; !areEqual(got, want) {
					t.Errorf("%s: got:\n%v\nwant:\n%v\n", tt.desc, pretty.Sprint(got), pretty.Sprint(want))
				}
			}
		})
	}
}
