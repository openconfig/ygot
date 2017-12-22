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
	"testing"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygot"
)

type Case1Leaf1ChoiceStruct struct {
	Case1Leaf1 *string `path:"case1-leaf1"`
}

func (*Case1Leaf1ChoiceStruct) IsYANGGoStruct() {}

type Leaf1ContainerStruct struct {
	Leaf1Name *string `path:"config/leaf1|leaf1"`
}

func (*Leaf1ContainerStruct) IsYANGGoStruct() {}

type EmptyContainerStruct struct {
}

func (*EmptyContainerStruct) IsYANGGoStruct() {}

type FakeRootStruct struct {
	LeafOne *string `path:"leaf-one"`
	LeafTwo *string `path:"leaf-two"`
}

func (*FakeRootStruct) IsYANGGoStruct() {}

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
		ListAttr: &yang.ListAttr{MinElements: &yang.Value{Name: "0"}},
		Type:     &yang.YangType{Kind: yang.Ystring},
		Name:     "leaf-list-schema",
	}

	listSchema := &yang.Entry{
		Name:     "list-schema",
		Kind:     yang.DirectoryEntry,
		ListAttr: &yang.ListAttr{MinElements: &yang.Value{Name: "0"}},
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
	}

	tests := []struct {
		desc    string
		val     interface{}
		schema  *yang.Entry
		opts    []ygot.ValidationOption
		wantErr string
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
			testErrLog(t, tt.desc, errs)
		})
	}

}
