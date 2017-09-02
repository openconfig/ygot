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
