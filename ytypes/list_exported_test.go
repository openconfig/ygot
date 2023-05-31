// Copyright 2023 Google Inc.
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

package ytypes_test

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/h-fam/errdiff"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/integration_tests/schemaops/ctestschema"
	"github.com/openconfig/ygot/ygot"
	"github.com/openconfig/ygot/ytypes"
)

// addParents adds parent pointers for a schema tree.
func addParents(e *yang.Entry) {
	for _, c := range e.Dir {
		c.Parent = e
		addParents(c)
	}
}

func TestUnmarshalKeyedList(t *testing.T) {
	keyListSchema := func() *yang.Entry {
		return &yang.Entry{
			Name:     "key-list",
			Kind:     yang.DirectoryEntry,
			ListAttr: yang.NewDefaultListAttr(),
			Key:      "key",
			Config:   yang.TSTrue,
			Dir: map[string]*yang.Entry{
				"key": {
					Kind: yang.LeafEntry,
					Name: "key",
					Type: &yang.YangType{Kind: yang.Ystring},
				},
				"leaf-field": {
					Kind: yang.LeafEntry,
					Name: "leaf-field",
					Type: &yang.YangType{Kind: yang.Yint32},
				},
				"leaf-field2": {
					Kind: yang.LeafEntry,
					Name: "leaf-field2",
					Type: &yang.YangType{Kind: yang.Yint32},
				},
			},
		}
	}

	containerWithLeafListSchema := &yang.Entry{
		Name: "container",
		Kind: yang.DirectoryEntry,
		Dir: map[string]*yang.Entry{
			"key-list": keyListSchema(),
		},
	}
	addParents(containerWithLeafListSchema)

	containerWithPreferConfigSchema := &yang.Entry{
		Name: "container",
		Kind: yang.DirectoryEntry,
		Dir: map[string]*yang.Entry{
			"config": {
				Name: "config",
				Kind: yang.DirectoryEntry,
				Dir: map[string]*yang.Entry{
					"key-list": keyListSchema(),
				},
			},
			"state": {
				Name: "state",
				Kind: yang.DirectoryEntry,
				Dir: map[string]*yang.Entry{
					"key-list": keyListSchema(),
				},
			},
		},
	}
	addParents(containerWithPreferConfigSchema)

	type ListElemStruct struct {
		Key        *string `path:"key"`
		LeafField  *int32  `path:"leaf-field"`
		LeafField2 *int32  `path:"leaf-field2"`
	}
	type ContainerStruct struct {
		KeyList map[string]*ListElemStruct `path:"key-list"`
	}

	type ContainerStructPreferConfig struct {
		KeyList map[string]*ListElemStruct `path:"config/key-list" shadow-path:"state/key-list"`
	}

	tests := []struct {
		desc    string
		json    string
		schema  *yang.Entry
		parent  interface{}
		want    interface{}
		opts    []ytypes.UnmarshalOpt
		wantErr string
	}{
		{
			desc:   "success",
			json:   `{ "key-list" : [ { "key" : "forty-two", "leaf-field" : 42} ] }`,
			schema: containerWithLeafListSchema,
			parent: &ContainerStruct{},
			want: &ContainerStruct{
				KeyList: map[string]*ListElemStruct{
					"forty-two": {
						Key:       ygot.String("forty-two"),
						LeafField: ygot.Int32(42),
					},
				},
			},
		},
		{
			desc:   "success with config path",
			json:   `{ "config": { "key-list" : [ { "key" : "forty-two", "leaf-field" : 42} ] } }`,
			schema: containerWithPreferConfigSchema,
			parent: &ContainerStructPreferConfig{},
			want: &ContainerStructPreferConfig{
				KeyList: map[string]*ListElemStruct{
					"forty-two": {
						Key:       ygot.String("forty-two"),
						LeafField: ygot.Int32(42),
					},
				},
			},
		},
		{
			desc:   "success with already-instantiated list element",
			json:   `{ "key-list" : [ { "key" : "forty-two", "leaf-field" : 42} ] }`,
			schema: containerWithLeafListSchema,
			parent: &ContainerStruct{
				KeyList: map[string]*ListElemStruct{
					"forty-two": {
						Key:        ygot.String("forty-two"),
						LeafField2: ygot.Int32(43),
					},
				},
			},
			want: &ContainerStruct{
				KeyList: map[string]*ListElemStruct{
					"forty-two": {
						Key:        ygot.String("forty-two"),
						LeafField:  ygot.Int32(42),
						LeafField2: ygot.Int32(43),
					},
				},
			},
		},
		{
			desc:   "success ignoring shadowed state path",
			json:   `{ "state": { "key-list" : [ { "key" : "forty-two", "leaf-field" : 42} ] } }`,
			schema: containerWithPreferConfigSchema,
			parent: &ContainerStructPreferConfig{},
			want:   &ContainerStructPreferConfig{},
		},
		{
			desc:   "success ignoring path with preferShadowPath",
			json:   `{ "config": { "key-list" : [ { "key" : "forty-two", "leaf-field" : 42} ] } }`,
			opts:   []ytypes.UnmarshalOpt{&ytypes.PreferShadowPath{}},
			schema: containerWithPreferConfigSchema,
			parent: &ContainerStructPreferConfig{},
			want:   &ContainerStructPreferConfig{},
		},
		{
			desc:   "success unmarshalling shadow path",
			json:   `{ "state": { "key-list" : [ { "key" : "forty-two", "leaf-field" : 42} ] } }`,
			opts:   []ytypes.UnmarshalOpt{&ytypes.PreferShadowPath{}},
			schema: containerWithPreferConfigSchema,
			parent: &ContainerStructPreferConfig{},
			want: &ContainerStructPreferConfig{
				KeyList: map[string]*ListElemStruct{
					"forty-two": {
						Key:       ygot.String("forty-two"),
						LeafField: ygot.Int32(42),
					},
				},
			},
		},
		{
			desc:    "bad field",
			json:    `{ "key-list" : [ { "key" : "forty-two", "bad-field" : 42} ] }`,
			schema:  containerWithLeafListSchema,
			parent:  &ContainerStruct{},
			wantErr: `parent container key-list (type *ytypes_test.ListElemStruct): JSON contains unexpected field bad-field`,
		},
		{
			desc:   "ignore unknown field",
			json:   `{ "key-list" : [ { "key" : "forty-two", "bad-field" : 42} ] }`,
			opts:   []ytypes.UnmarshalOpt{&ytypes.IgnoreExtraFields{}},
			schema: containerWithLeafListSchema,
			parent: &ContainerStruct{},
			want: &ContainerStruct{
				KeyList: map[string]*ListElemStruct{
					"forty-two": {
						Key: ygot.String("forty-two"),
					},
				},
			},
		},
		{
			desc:   "success with ordered map",
			json:   `{ "ordered-lists": { "ordered-list" : [ { "key" : "foo", "config": { "value" : "foo-val" } }, { "key" : "bar", "config": { "value" : "bar-val" } } ] } }`,
			schema: ctestschema.SchemaTree["Device"],
			parent: &ctestschema.Device{},
			want: &ctestschema.Device{
				OrderedList: ctestschema.GetOrderedMap(t),
			},
		},
		{
			desc:   "success at ordered map level",
			json:   `[ { "key" : "foo", "config": { "value" : "foo-val" } }, { "key" : "bar", "config": { "value" : "bar-val" } } ]`,
			schema: ctestschema.SchemaTree["OrderedList"],
			parent: &ctestschema.OrderedList_OrderedMap{},
			want:   ctestschema.GetOrderedMap(t),
		},
		{
			desc:   "success with nested ordered map",
			json:   `{ "ordered-lists": { "ordered-list" : [ { "key" : "foo", "config": { "value" : "foo-val" }, "ordered-lists": { "ordered-list" : [ { "key" : "foo", "config": { "value" : "foo-val" } }, { "key" : "bar", "config": { "value" : "bar-val" } } ] } }, { "key" : "bar", "config": { "value" : "bar-val" } } ] } }`,
			schema: ctestschema.SchemaTree["Device"],
			parent: &ctestschema.Device{},
			want: &ctestschema.Device{
				OrderedList: ctestschema.GetNestedOrderedMap(t),
			},
		},
	}

	var jsonTree interface{}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if err := json.Unmarshal([]byte(tt.json), &jsonTree); err != nil {
				t.Fatalf("%s : %s", tt.desc, err)
			}

			err := ytypes.Unmarshal(tt.schema, tt.parent, jsonTree, tt.opts...)
			if diff := errdiff.Text(err, tt.wantErr); diff != "" {
				t.Fatalf("%s: Unmarshal error not expected:\n%s", tt.desc, diff)
			}
			if err == nil {
				got, want := tt.parent, tt.want
				if diff := cmp.Diff(want, got, cmp.AllowUnexported(ctestschema.OrderedList_OrderedMap{}, ctestschema.OrderedList_OrderedList_OrderedMap{})); diff != "" {
					t.Errorf("%s: Unmarshal (-want, +got):\n%s", tt.desc, diff)
				}
			}
		})
	}
}

func TestUnmarshalSingleListElement(t *testing.T) {
	listSchema := &yang.Entry{
		Name:     "struct-list",
		Kind:     yang.DirectoryEntry,
		ListAttr: yang.NewDefaultListAttr(),
		Dir: map[string]*yang.Entry{
			"leaf-field": {
				Kind: yang.LeafEntry,
				Name: "leaf-field",
				Type: &yang.YangType{Kind: yang.Yint32},
			},
			"enum-leaf-field": {
				Kind: yang.LeafEntry,
				Name: "enum-leaf-field",
				Type: &yang.YangType{Kind: yang.Yenum},
			},
			"leaf2-field": {
				Kind: yang.LeafEntry,
				Name: "leaf2-field",
				Type: &yang.YangType{Kind: yang.Yint64},
			},
		},
	}

	type ListElemStruct struct {
		LeafName  *int32          `path:"leaf-field"`
		EnumLeaf  ytypes.EnumType `path:"enum-leaf-field"`
		Leaf2Name *int64          `path:"leaf2-field"`
	}

	tests := []struct {
		desc    string
		schema  *yang.Entry
		json    string
		parent  any
		want    any
		wantErr string
	}{
		{
			desc:   "success",
			schema: listSchema,
			json:   `{ "leaf-field" : 42, "enum-leaf-field" : "E_VALUE_FORTY_TWO"}`,
			parent: &ListElemStruct{
				Leaf2Name: ygot.Int64(42),
			},
			want: &ListElemStruct{
				LeafName:  ygot.Int32(42),
				Leaf2Name: ygot.Int64(42),
				EnumLeaf:  42,
			},
		},
		{
			desc:    "bad field",
			schema:  listSchema,
			json:    `{ "leaf-field" : 42, "bad-field" : "E_VALUE_FORTY_TWO"}`,
			parent:  &ListElemStruct{},
			wantErr: `parent container struct-list (type *ytypes_test.ListElemStruct): JSON contains unexpected field bad-field`,
		},
		{
			desc:   "success with ordered map -- this should be the same as a regular map object",
			json:   `{ "key" : "foo", "config": { "value" : "foo-val"} }`,
			schema: ctestschema.SchemaTree["OrderedList"],
			parent: &ctestschema.OrderedList{},
			want: &ctestschema.OrderedList{
				Key:   ygot.String("foo"),
				Value: ygot.String("foo-val"),
			},
		},
	}

	var jsonTree interface{}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if err := json.Unmarshal([]byte(tt.json), &jsonTree); err != nil {
				t.Fatalf("%s : %s", tt.desc, err)
			}

			err := ytypes.Unmarshal(tt.schema, tt.parent, jsonTree)
			if diff := errdiff.Text(err, tt.wantErr); diff != "" {
				t.Fatalf("%s: Unmarshal error not expected:\n%s", tt.desc, diff)
			}
			if err == nil {
				got, want := tt.parent, tt.want
				if diff := cmp.Diff(want, got); diff != "" {
					t.Errorf("%s: Unmarshal (-want, +got):\n%s", tt.desc, diff)
				}
			}
		})
	}
}
