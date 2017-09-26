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
	"fmt"
	"testing"

	"github.com/kylelemons/godebug/pretty"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygot"
)

// errToString returns the string representation of err and the empty string if
// err is nil.
func errToString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

type ContainerStruct struct {
	Leaf1Name *string `path:"config/leaf1|leaf1"`
	Leaf2Name *string `path:"leaf2"`
}

func (c *ContainerStruct) IsYANGGoStruct() {}

func TestValidateContainerSchema(t *testing.T) {
	validContainerSchema := &yang.Entry{
		Name: "valid-container-schema",
		Kind: yang.DirectoryEntry,
		Dir: map[string]*yang.Entry{
			"config": {
				Dir: map[string]*yang.Entry{
					"leaf1": {
						Kind: yang.LeafEntry,
						Name: "leaf1",
						Type: &yang.YangType{Kind: yang.Ystring},
					},
				},
			},
			"leaf2": {
				Kind: yang.LeafEntry,
				Name: "leaf2",
				Type: &yang.YangType{Kind: yang.Ystring},
			},
		},
	}
	tests := []struct {
		desc    string
		schema  *yang.Entry
		wantErr bool
	}{
		{
			desc:   "success",
			schema: validContainerSchema,
		},
		{
			desc:   "empty container",
			schema: &yang.Entry{Kind: yang.DirectoryEntry},
		},
		{
			desc:    "nil schema",
			schema:  nil,
			wantErr: true,
		},
		{
			desc:    "nil schema type",
			schema:  &yang.Entry{},
			wantErr: true,
		},
	}

	for _, test := range tests {
		err := validateContainerSchema(test.schema)
		if got, want := (err != nil), test.wantErr; got != want {
			t.Errorf("%s: validateContainerSchema(%v) got error: %v, wanted error? %v", test.desc, test.schema, err, test.wantErr)
		}
		testErrLog(t, test.desc, err)
	}
}

func TestValidateContainer(t *testing.T) {
	containerSchema := &yang.Entry{
		Name: "container-schema",
		Kind: yang.DirectoryEntry,
		Dir: map[string]*yang.Entry{
			"config": {
				Dir: map[string]*yang.Entry{
					"leaf1": {
						Kind: yang.LeafEntry,
						Name: "leaf1",
						Type: &yang.YangType{Kind: yang.Ystring},
					},
				},
			},
			"leaf2": {
				Kind: yang.LeafEntry,
				Name: "leaf2",
				Type: &yang.YangType{Kind: yang.Ystring},
			},
		},
	}

	type BadStruct struct {
		UnknownName *string `path:"unknown"`
	}

	tests := []struct {
		desc    string
		schema  *yang.Entry
		val     interface{}
		wantErr bool
	}{
		{
			desc: "success",
			val: &ContainerStruct{
				Leaf1Name: ygot.String("Leaf1Value"),
				Leaf2Name: ygot.String("Leaf2Value"),
			},
		},
		{
			desc:    "bad value type",
			schema:  containerSchema,
			val:     int(1),
			wantErr: true,
		},
		{
			desc:    "bad schema",
			schema:  nil,
			val:     int(1),
			wantErr: true,
		},
		{
			desc:    "missing key",
			schema:  containerSchema,
			val:     &BadStruct{UnknownName: ygot.String("Unknown")},
			wantErr: true,
		},
	}

	for _, test := range tests {
		err := Validate(containerSchema, test.val)
		if got, want := (err != nil), test.wantErr; got != want {
			t.Errorf("%s: Validate got error: %v, wanted error? %v", test.desc, err, test.wantErr)
		}
		testErrLog(t, test.desc, err)
	}
}

func TestUnmarshalContainer(t *testing.T) {
	innerContainerSchema := &yang.Entry{
		Name: "container-field",
		Kind: yang.DirectoryEntry,
		Dir: map[string]*yang.Entry{
			"config": {
				Name: "config",
				Kind: yang.DirectoryEntry,
				Dir: map[string]*yang.Entry{
					"leaf1-field": {
						Kind: yang.LeafEntry,
						Name: "leaf1-field",
						Type: &yang.YangType{Kind: yang.Yint32},
					},
				},
			},
			"state": {
				Name: "state",
				Kind: yang.DirectoryEntry,
				Dir: map[string]*yang.Entry{
					"leaf1-field": {
						Kind: yang.LeafEntry,
						Name: "leaf1-field",
						Type: &yang.YangType{Kind: yang.Yint32},
					},
				},
			},
			"leaf2-field": {
				Kind: yang.LeafEntry,
				Name: "leaf2-field",
				Type: &yang.YangType{Kind: yang.Yint32},
			},
		},
	}
	containerSchema := &yang.Entry{
		Name: "parent-field",
		Kind: yang.DirectoryEntry,
		Dir: map[string]*yang.Entry{
			"container-field": innerContainerSchema,
		},
	}

	populateParentField(nil, containerSchema)

	type ContainerStruct struct {
		ConfigLeaf1Field *int32 `path:"config/leaf1-field"`
		StateLeaf1Field  *int32 `path:"state/leaf1-field"`
		Leaf2Field       *int32 `path:"leaf2-field"`
	}

	type ParentContainerStruct struct {
		ContainerField *ContainerStruct `path:"container-field"`
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
			json:   `{ "container-field": { "leaf2-field": 43, "config": { "leaf1-field": 41 } , "state": { "leaf1-field": 42 } } }`,
			want:   &ParentContainerStruct{ContainerField: &ContainerStruct{ConfigLeaf1Field: ygot.Int32(41), StateLeaf1Field: ygot.Int32(42), Leaf2Field: ygot.Int32(43)}},
		},
		{
			desc:    "bad field name",
			schema:  containerSchema,
			json:    `{ "container-field": { "bad-field": 42 } }`,
			wantErr: `parent container container-field (type *ytypes.ContainerStruct): JSON contains unexpected field bad-field`,
		},
		{
			desc:    "bad field type",
			schema:  containerSchema,
			json:    `{ "container-field": { "leaf2-field":  "forty-two"} }`,
			wantErr: `got string type for field leaf2-field, expect float64`,
		},
	}

	var jsonTree interface{}
	for _, test := range tests {
		var parent ParentContainerStruct

		if err := json.Unmarshal([]byte(test.json), &jsonTree); err != nil {
			t.Fatal(fmt.Sprintf("json unmarshal (%s) : %s", test.desc, err))
		}

		err := Unmarshal(test.schema, &parent, jsonTree)
		if got, want := errToString(err), test.wantErr; got != want {
			t.Errorf("%s: got error: %v, wanted error? %v", test.desc, got, want)
		}
		testErrLog(t, test.desc, err)
		if err == nil {
			if got, want := &parent, test.want; !areEqual(got, want) {
				t.Errorf("%s: got:\n%v\nwant:\n%v\n", test.desc, pretty.Sprint(got), pretty.Sprint(want))
			}
		}
	}
}
