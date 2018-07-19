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

type CCChild struct{}

func (*CCChild) IsYANGGoStruct() {}

type CCParentStruct struct {
	EnumerationLeaf int64     `path:"enumeration-leaf"`
	Leaf            string    `path:"leaf"`
	Container       *CCChild  `path:"container"`
	Empty           YANGEmpty `path:"empty"`
}

func (*CCParentStruct) IsYANGGoStruct() {}

type ChoiceCaseStruct struct {
	Parent *CCParentStruct `path:"parent"`
}

func (*ChoiceCaseStruct) IsYANGGoStruct() {}

type ParentStruct struct {
	Leaf *string `path:"leaf"`
}

func (*ParentStruct) IsYANGGoStruct() {}

func TestUnmarshal(t *testing.T) {

	validSchema := &yang.Entry{
		Name: "leaf",
		Kind: yang.LeafEntry,
		Type: &yang.YangType{
			Kind: yang.Ystring,
		},
	}
	choiceSchema := &yang.Entry{
		Name: "choice",
		Kind: yang.ChoiceEntry,
	}

	rootSchema := &yang.Entry{
		Name: "root",
		Kind: yang.DirectoryEntry,
		Dir:  map[string]*yang.Entry{},
	}

	parentSchema := &yang.Entry{
		Name:   "parent",
		Kind:   yang.DirectoryEntry,
		Dir:    map[string]*yang.Entry{},
		Parent: rootSchema,
	}
	rootSchema.Dir["parent"] = parentSchema

	enumType := yang.NewEnumType()
	enumType.Set("ONE", int64(1))
	enumLeafSchema := &yang.Entry{
		Name: "enumeration-leaf",
		Kind: yang.LeafEntry,
		Type: &yang.YangType{
			Kind: yang.Yenum,
			Enum: enumType,
		},
		Parent: parentSchema,
	}
	parentSchema.Dir["enumeration-leaf"] = enumLeafSchema

	childChoiceSchema := &yang.Entry{
		Name:   "choice",
		Kind:   yang.ChoiceEntry,
		Parent: parentSchema,
		Dir:    map[string]*yang.Entry{},
	}
	parentSchema.Dir["choice"] = childChoiceSchema

	childChoiceLeafSchema := &yang.Entry{
		Name: "leaf",
		Kind: yang.LeafEntry,
		Type: &yang.YangType{
			Kind: yang.Ystring,
		},
		Parent: childChoiceSchema,
	}
	childChoiceSchema.Dir["leaf"] = childChoiceLeafSchema

	childChoiceContainerSchema := &yang.Entry{
		Name:   "container",
		Kind:   yang.DirectoryEntry,
		Parent: childChoiceSchema,
		Dir:    map[string]*yang.Entry{},
	}
	childChoiceSchema.Dir["container"] = childChoiceContainerSchema

	childEmptySchema := &yang.Entry{
		Name: "empty",
		Kind: yang.LeafEntry,
		Type: &yang.YangType{
			Kind: yang.Yempty,
		},
		Parent: childChoiceSchema,
	}
	childChoiceSchema.Dir["empty"] = childEmptySchema

	tests := []struct {
		desc    string
		schema  *yang.Entry
		value   interface{}
		target  ygot.GoStruct
		opts    []UnmarshalOpt
		wantErr string
	}{
		{
			desc:   "success nil field",
			schema: validSchema,
			target: &ParentStruct{},
			value:  nil,
		},
		{
			desc:    "error nil schema",
			schema:  nil,
			target:  &ParentStruct{},
			value:   "{}",
			wantErr: `nil schema for parent type *ytypes.ParentStruct, value {} (string)`,
		},
		{
			desc:    "error choice schema",
			schema:  choiceSchema,
			target:  &ParentStruct{},
			value:   "{}",
			wantErr: `cannot pass choice schema choice to Unmarshal`,
		},
		{
			desc:   "passing options to Unmarshal",
			schema: validSchema,
			target: &ParentStruct{},
			value:  nil,
			opts:   []UnmarshalOpt{&IgnoreExtraFields{}},
		}, {
			desc:   "unmarshal with choice/case and enum",
			schema: rootSchema,
			target: &ChoiceCaseStruct{},
			value: map[string]interface{}{
				"parent": map[string]interface{}{
					"empty": []interface{}{nil},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := Unmarshal(tt.schema, tt.target, tt.value, tt.opts...)
			if got, want := errToString(err), tt.wantErr; got != want {
				t.Errorf("%s: got error: %v, want error: %v", tt.desc, got, want)
			}

		})
	}
}
