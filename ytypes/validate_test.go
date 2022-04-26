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

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygot"
)

type Case1Leaf1ChoiceStruct struct {
	Case1Leaf1 *string `path:"case1-leaf1"`
}

func (*Case1Leaf1ChoiceStruct) IsYANGGoStruct()                          {}
func (*Case1Leaf1ChoiceStruct) ΛValidate(...ygot.ValidationOption) error { return nil }
func (*Case1Leaf1ChoiceStruct) ΛEnumTypeMap() map[string][]reflect.Type  { return nil }
func (*Case1Leaf1ChoiceStruct) ΛBelongingModule() string                 { return "bar" }

type Leaf1ContainerStruct struct {
	Leaf1Name *string `path:"config/leaf1|leaf1"`
}

func (*Leaf1ContainerStruct) IsYANGGoStruct()                          {}
func (*Leaf1ContainerStruct) ΛValidate(...ygot.ValidationOption) error { return nil }
func (*Leaf1ContainerStruct) ΛEnumTypeMap() map[string][]reflect.Type  { return nil }
func (*Leaf1ContainerStruct) ΛBelongingModule() string                 { return "bar" }

type EmptyContainerStruct struct {
}

func (*EmptyContainerStruct) IsYANGGoStruct()                          {}
func (*EmptyContainerStruct) ΛValidate(...ygot.ValidationOption) error { return nil }
func (*EmptyContainerStruct) ΛEnumTypeMap() map[string][]reflect.Type  { return nil }
func (*EmptyContainerStruct) ΛBelongingModule() string                 { return "bar" }

type FakeRootStruct struct {
	LeafOne   *string `path:"leaf-one"`
	LeafTwo   *string `path:"leaf-two"`
	LeafThree *string `path:"leaf-three"`
}

func (*FakeRootStruct) IsYANGGoStruct()                          {}
func (*FakeRootStruct) ΛValidate(...ygot.ValidationOption) error { return nil }
func (*FakeRootStruct) ΛEnumTypeMap() map[string][]reflect.Type  { return nil }
func (*FakeRootStruct) ΛBelongingModule() string                 { return "bar" }

func customValidation(val ygot.ValidatedGoStruct) error {
	fakeRoot, ok := val.(*FakeRootStruct)
	if !ok {
		return fmt.Errorf("not valid fakeroot")
	}
	if fakeRoot.LeafThree == nil || *fakeRoot.LeafThree != "kingfisher" {
		return fmt.Errorf("leafThree should be kingfisher")
	}
	return nil
}
func TestValidate(t *testing.T) {
	leafSchema := &yang.Entry{Name: "leaf-schema", Kind: yang.LeafEntry, Type: &yang.YangType{Kind: yang.Ystring}}

	containerSchema := &yang.Entry{
		Name: "container-schema",
		Kind: yang.DirectoryEntry,
		Dir: map[string]*yang.Entry{
			"config": {
				Dir: map[string]*yang.Entry{
					"leaf1": {
						Kind: yang.LeafEntry,
						Name: "Leaf1Name",
						Type: &yang.YangType{Kind: yang.Ystring},
					},
				},
			},
		},
	}

	emptyContainerSchema := &yang.Entry{
		Name: "empty-container-schema",
		Kind: yang.DirectoryEntry,
	}

	leafListSchema := &yang.Entry{
		Kind:     yang.LeafEntry,
		ListAttr: yang.NewDefaultListAttr(),
		Type:     &yang.YangType{Kind: yang.Ystring},
		Name:     "leaf-list-schema",
	}

	listSchema := &yang.Entry{
		Name:     "list-schema",
		Kind:     yang.DirectoryEntry,
		ListAttr: yang.NewDefaultListAttr(),
		Dir: map[string]*yang.Entry{
			"leaf-name": {
				Kind: yang.LeafEntry,
				Name: "LeafName",
				Type: &yang.YangType{Kind: yang.Ystring},
			},
		},
	}

	containerWithChoiceSchema := &yang.Entry{
		Name: "container-with-choice-schema",
		Kind: yang.DirectoryEntry,
		Dir: map[string]*yang.Entry{
			"Choice1Name": {
				Kind: yang.ChoiceEntry,
				Name: "Choice1Name",
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
				},
			},
		},
	}

	type StringListElemStruct struct {
		LeafName *string `path:"leaf-name"`
	}

	fakerootSchema := &yang.Entry{
		Name: "device",
		Kind: yang.DirectoryEntry,
		Annotation: map[string]interface{}{
			"isFakeRoot": true,
		},
	}
	fakerootSchema.Dir = map[string]*yang.Entry{
		"leaf-one": {
			Name: "leaf-one",
			Kind: yang.LeafEntry,
			Type: &yang.YangType{
				Kind: yang.Ystring,
			},
			Parent: fakerootSchema,
		},
		"leaf-two": {
			Name: "leaf-two",
			Kind: yang.LeafEntry,
			Type: &yang.YangType{
				Kind: yang.Yleafref,
				Path: "../leaf-one",
			},
			Parent: fakerootSchema,
		},
		"leaf-three": {
			Name: "leaf-three",
			Kind: yang.LeafEntry,
			Type: &yang.YangType{
				Kind:    yang.Ystring,
				Pattern: []string{"^a.*"},
			},
		},
	}

	tests := []struct {
		desc       string
		val        interface{}
		schema     *yang.Entry
		opts       []ygot.ValidationOption
		wantErr    string
		wantErrLen int
	}{
		{
			desc:   "leaf",
			schema: leafSchema,
			val:    ygot.String("value"),
		},
		{
			desc:   "container",
			schema: containerSchema,
			val: &Leaf1ContainerStruct{
				Leaf1Name: ygot.String("Leaf1Value"),
			},
		},
		{
			desc:   "fakeroot with leafref",
			schema: fakerootSchema,
			val: &FakeRootStruct{
				LeafOne: ygot.String("one"),
				LeafTwo: ygot.String("one"),
			},
		},
		{
			desc:   "fakeroot with leafref with missing data option",
			schema: fakerootSchema,
			val: &FakeRootStruct{
				LeafTwo: ygot.String("two"),
			},
			opts: []ygot.ValidationOption{&LeafrefOptions{IgnoreMissingData: true}},
		},
		{
			desc:   "fakeroot with custom validation",
			schema: fakerootSchema,
			val: &FakeRootStruct{
				LeafOne: ygot.String("one"),
				LeafTwo: ygot.String("one"),
			},
			opts:       []ygot.ValidationOption{&CustomValidationOptions{FakeRootCustomValidate: customValidation}},
			wantErr:    "leafThree should be kingfisher",
			wantErrLen: 1,
		},
		{
			desc:   "fakeroot with custom validation and ignore bad leafref",
			schema: fakerootSchema,
			val: &FakeRootStruct{
				LeafTwo: ygot.String("two"),
			},
			opts:       []ygot.ValidationOption{&LeafrefOptions{IgnoreMissingData: true}, &CustomValidationOptions{FakeRootCustomValidate: customValidation}},
			wantErr:    "leafThree should be kingfisher",
			wantErrLen: 1,
		},
		{
			desc:   "fakeroot with custom validation and bad leafref",
			schema: fakerootSchema,
			val: &FakeRootStruct{
				LeafTwo: ygot.String("two"),
			},
			opts:       []ygot.ValidationOption{&CustomValidationOptions{FakeRootCustomValidate: customValidation}},
			wantErr:    "pointed-to value with path ../leaf-one from field LeafTwo value two (string ptr) schema /device/leaf-two is empty set, leafThree should be kingfisher",
			wantErrLen: 2,
		},
		{
			desc:   "two errors",
			schema: fakerootSchema,
			val: &FakeRootStruct{
				LeafTwo:   ygot.String("two"),
				LeafThree: ygot.String("fish"),
			},
			wantErr:    `pointed-to value with path ../leaf-one from field LeafTwo value two (string ptr) schema /device/leaf-two is empty set, /leaf-three: schema "leaf-three": "fish" does not match regular expression pattern "^a.*$"`, // Check that there is an error
			wantErrLen: 2,
		},
		{
			desc:   "empty container",
			schema: emptyContainerSchema,
			val:    &EmptyContainerStruct{},
		},
		{
			desc:   "leaf-list",
			schema: leafListSchema,
			val:    []string{"test1", "test2"},
		},
		{
			desc:   "list",
			schema: listSchema,
			val:    []*StringListElemStruct{{LeafName: ygot.String("elem1_leaf_name")}},
		},
		{
			desc:   "choice",
			schema: containerWithChoiceSchema,
			val:    &Case1Leaf1ChoiceStruct{Case1Leaf1: ygot.String("Case1Leaf1Value")},
		},
		{
			desc:   "choice",
			schema: containerWithChoiceSchema,
			val:    &Case1Leaf1ChoiceStruct{Case1Leaf1: ygot.String("Case1Leaf1Value")},
		},
		{
			desc:    "choice schema not allowed",
			schema:  &yang.Entry{Kind: yang.ChoiceEntry, Name: "choice"},
			val:     &EmptyContainerStruct{},
			wantErr: `cannot pass choice schema choice to Validate`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			errs := Validate(tt.schema, tt.val, tt.opts...)
			if got, want := errs.String(), tt.wantErr; got != want {
				t.Errorf("%s: Validate got error: %s, want error: %s", tt.desc, got, want)
			}

			if tt.wantErrLen != 0 {
				if len(errs) != tt.wantErrLen {
					t.Errorf("%s: Validate did not get expected number of errors, got: %d, want: %d", tt.desc, len(errs), tt.wantErrLen)
				}
			}

			testErrLog(t, tt.desc, errs)
		})
	}

}
