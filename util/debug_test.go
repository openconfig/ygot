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

package util

import (
	"testing"

	"github.com/openconfig/goyang/pkg/yang"
)

var (
	debugLibraryOrig    = debugLibrary
	debugSchemaOrig     = debugSchema
	maxCharsPerLineOrig = maxCharsPerLine
	maxValueStrLenOrig  = maxValueStrLen
)

func saveDebugVals() {
	debugLibraryOrig = debugLibrary
	debugSchemaOrig = debugSchema
	maxCharsPerLineOrig = maxCharsPerLine
	maxValueStrLenOrig = maxValueStrLen
}

func restoreDebugVals() {
	debugLibrary = debugLibraryOrig
	debugSchema = debugSchemaOrig
	maxCharsPerLine = maxCharsPerLineOrig
	maxValueStrLen = maxValueStrLenOrig
}

func TestDbgPrint(t *testing.T) {
	saveDebugVals()
	defer restoreDebugVals()

	debugLibrary = true
	maxCharsPerLine = 10
	DbgPrint("test debug <this should be truncated>")
}

func TestDbgSchema(t *testing.T) {
	saveDebugVals()
	defer restoreDebugVals()

	debugSchema = true
	maxCharsPerLine = 10
	DbgSchema("test debug <this should not be truncated>\n")
}

func TestValueStr(t *testing.T) {
	toStrPtr := func(s string) *string { return &s }

	saveDebugVals()
	defer restoreDebugVals()

	testStruct := struct {
		A int
		B string
		C *string
		D []string
	}{
		A: 42,
		B: "forty two",
		C: toStrPtr("forty two"),
		D: []string{"a", "b", "c"},
	}

	wantStr := `{ 42 (int), forty two (string), forty two (string ptr), [ a (string), b (string), c (string) ] }`
	maxValueStrLen = len(wantStr) - 1
	want := wantStr[:len(wantStr)-1] + "..."

	if got := ValueStr(testStruct); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestSchemaTypeStr(t *testing.T) {
	tests := []struct {
		schema *yang.Entry
		want   string
	}{
		{
			schema: &yang.Entry{
				Kind: yang.ChoiceEntry,
			},
			want: "choice",
		},
		{
			schema: &yang.Entry{
				Kind: yang.CaseEntry,
			},
			want: "case",
		},
		{
			schema: &yang.Entry{
				Kind: yang.DirectoryEntry,
			},
			want: "container",
		},
		{
			schema: &yang.Entry{
				Kind:     yang.DirectoryEntry,
				Dir:      map[string]*yang.Entry{},
				ListAttr: &yang.ListAttr{MinElements: &yang.Value{Name: "0"}},
			},
			want: "list",
		},
		{
			schema: &yang.Entry{
				Kind: yang.LeafEntry,
			},
			want: "leaf",
		},
		{
			schema: &yang.Entry{
				Kind:     yang.LeafEntry,
				ListAttr: &yang.ListAttr{MinElements: &yang.Value{Name: "0"}},
			},
			want: "leaf-list",
		},
		{
			schema: &yang.Entry{
				Kind:     yang.DirectoryEntry,
				ListAttr: &yang.ListAttr{MinElements: &yang.Value{Name: "0"}},
			},
			want: "other",
		},
	}
	for _, tt := range tests {
		if got, want := SchemaTypeStr(tt.schema), tt.want; got != want {
			t.Errorf("got : %s, want: %s", got, want)
		}
	}
}

func TestYangTypeToDebugString(t *testing.T) {
	yangType := &yang.YangType{
		Kind:    yang.Ystring,
		Pattern: []string{"abc"},
		Range:   yang.YangRange{yang.YRange{Min: YangMinNumber, Max: YangMaxNumber}},
	}

	wantStr := `(TypeKind: string, Pattern: abc, Range: min..max)`
	if got, want := YangTypeToDebugString(yangType), wantStr; got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestSchemaTreeString(t *testing.T) {
	schema := &yang.Entry{
		Kind:     yang.DirectoryEntry,
		ListAttr: &yang.ListAttr{MinElements: &yang.Value{Name: "0"}},
		Key:      "key_field_name",
		Config:   yang.TSTrue,
		Dir: map[string]*yang.Entry{
			"key_field_name": {
				Kind: yang.LeafEntry,
				Name: "key_field_name",
				Type: &yang.YangType{Kind: yang.Ystring},
			},
		},
	}

	wantStr := ` (list)
  key_field_name (leaf)
`
	if got, want := SchemaTreeString(schema, ""), wantStr; got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

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
				Key:       String("keyval1"),
				LeafField: Int32(42),
			},
		},
	}

	got := DataSchemaTreesString(containerWithListSchema, container)
	want := `     [container (container)]
    KeyList [key-list (list)]
    keyval1
      Key : "keyval1" [key (leaf)]
      LeafField : 42 [leaf-field (leaf)]
`
	if got != want {
		t.Errorf("got:\n%swant:\n%s", got, want)
	}
}

func String(s string) *string { return &s }
func Int32(i int32) *int32    { return &i }
