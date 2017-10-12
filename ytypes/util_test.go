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

func TestDataSchemaTreesString(t *testing.T) {
	containerWithListSchema := &yang.Entry{
		Name: "container",
		Kind: yang.DirectoryEntry,
		Dir: map[string]*yang.Entry{
			"key-list": {
				Name:     "key-list",
				Kind:     yang.DirectoryEntry,
				ListAttr: &yang.ListAttr{MinElements: &yang.Value{Name: "0"}},
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
					"leaf2-field": {
						Kind: yang.LeafEntry,
						Name: "leaf2-field",
						Type: &yang.YangType{Kind: yang.Yint32},
					},
				},
			},
		},
	}

	type ListElemStruct struct {
		Key        *string `path:"key"`
		LeafField  *int32  `path:"leaf-field"`
		Leaf2Field *int32  `path:"leaf2-field"`
	}
	type ContainerStruct struct {
		KeyList map[string]*ListElemStruct `path:"key-list"`
	}

	container := &ContainerStruct{
		KeyList: map[string]*ListElemStruct{
			"keyval1": {
				Key:       ygot.String("keyval1"),
				LeafField: ygot.Int32(42),
			},
		},
	}

	got := DataSchemaTreesString(containerWithListSchema, container)
	want := ` [container (container)]
  KeyList [key-list (list)]
  keyval1
    Key : "keyval1" [key (leaf)]
    LeafField : 42 [leaf-field (leaf)]
`
	if got != want {
		t.Errorf("got:\n%swant:\n%s", got, want)
	}
}
