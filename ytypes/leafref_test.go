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
	"reflect"
	"testing"

	"github.com/openconfig/goyang/pkg/yang"
)

func TestValidateLeafRefData(t *testing.T) {
	containerWithLeafListSchema := &yang.Entry{
		Name: "container",
		Kind: yang.DirectoryEntry,
		Dir: map[string]*yang.Entry{
			"leaf-list": {
				Name:     "leaf-list",
				Kind:     yang.LeafEntry,
				Type:     &yang.YangType{Kind: yang.Yint32},
				ListAttr: &yang.ListAttr{MinElements: &yang.Value{Name: "0"}},
			},
			"list": {
				Name:     "list",
				Kind:     yang.DirectoryEntry,
				ListAttr: &yang.ListAttr{MinElements: &yang.Value{Name: "0"}},
				Key:      "key",
				Dir: map[string]*yang.Entry{
					"key": {
						Name: "key",
						Kind: yang.LeafEntry,
						Type: &yang.YangType{Kind: yang.Yint32},
					},
					"int32": {
						Name: "int32",
						Kind: yang.LeafEntry,
						Type: &yang.YangType{Kind: yang.Yint32},
					},
				},
			},
			"int32": {
				Name: "int32",
				Kind: yang.LeafEntry,
				Type: &yang.YangType{Kind: yang.Yint32},
			},
			"key": {
				Name: "key",
				Kind: yang.LeafEntry,
				Type: &yang.YangType{Kind: yang.Yint32},
			},
			"enum": {
				Name: "enum",
				Kind: yang.LeafEntry,
				Type: &yang.YangType{Kind: yang.Yint64},
			},
			"container2": {
				Name: "container2",
				Kind: yang.DirectoryEntry,
				Dir: map[string]*yang.Entry{
					"int32-ref-to-leaf": {
						Name: "int32-ref-to-leaf",
						Kind: yang.LeafEntry,
						Type: &yang.YangType{
							Kind: yang.Yleafref,
							Path: "../../int32",
						},
					},
					"enum-ref-to-leaf": {
						Name: "enum-ref-to-leaf",
						Kind: yang.LeafEntry,
						Type: &yang.YangType{
							Kind: yang.Yleafref,
							Path: "../../enum",
						},
					},
					"int32-ref-to-leaf-list": {
						Name: "int32-ref-to-leaf-list",
						Kind: yang.LeafEntry,
						Type: &yang.YangType{
							Kind: yang.Yleafref,
							Path: "../../leaf-list",
						},
					},
					"leaf-list-ref-to-leaf-list": {
						Name: "leaf-list-ref-to-leaf-list",
						Kind: yang.LeafEntry,
						Type: &yang.YangType{
							Kind: yang.Yleafref,
							Path: "../../../leaf-list",
						},
						ListAttr: &yang.ListAttr{MinElements: &yang.Value{Name: "0"}},
					},
					"int32-ref-to-list": {
						Name: "int32-ref-to-list",
						Kind: yang.LeafEntry,
						Type: &yang.YangType{
							Kind: yang.Yleafref,
							Path: "../../list[key = current()/../../key]/int32",
						},
					},
					"leaf-list-with-leafref": {
						Name: "leaf-list-with-leafref",
						Kind: yang.LeafEntry,
						Type: &yang.YangType{
							Kind: yang.Yleafref,
							Path: "../../int32",
						},
						ListAttr: &yang.ListAttr{MinElements: &yang.Value{Name: "0"}},
					},
				},
			},
		},
	}

	type Container2 struct {
		LeafRefToInt32         *int32   `path:"int32-ref-to-leaf"`
		LeafRefToEnum          int64    `path:"enum-ref-to-leaf"`
		LeafRefToLeafList      *int32   `path:"int32-ref-to-leaf-list"`
		LeafListRefToLeafList  []*int32 `path:"leaf-list-ref-to-leaf-list"`
		LeafRefToList          *int32   `path:"int32-ref-to-list"`
		LeafListLeafRefToInt32 []*int32 `path:"leaf-list-with-leafref"`
	}
	type ListElement struct {
		Key   *int32 `path:"key"`
		Int32 *int32 `path:"int32"`
	}
	type Container struct {
		LeafList   []*int32               `path:"leaf-list"`
		List       map[int32]*ListElement `path:"list"`
		Int32      *int32                 `path:"int32"`
		Key        *int32                 `path:"key"`
		Int64      int64                  `path:"enum"`
		Container2 *Container2            `path:"container2"`
	}

	tests := []struct {
		desc    string
		in      interface{}
		opts    *LeafrefOptions
		wantErr string
	}{
		{
			desc: "nil",
			in:   nil,
		},
		{
			desc: "int32",
			in: &Container{
				Int32:      Int32(42),
				Container2: &Container2{LeafRefToInt32: Int32(42)},
			},
		},
		{
			desc: "int32 unequal",
			in: &Container{
				Int32:      Int32(42),
				Container2: &Container2{LeafRefToInt32: Int32(43)},
			},
			wantErr: `field name LeafRefToInt32 value 43 (int32 ptr) schema path /int32-ref-to-leaf has leafref path ../../int32 not equal to any target nodes`,
		},
		{
			desc: "int32 points to nil",
			in: &Container{
				Container2: &Container2{LeafRefToInt32: Int32(42)},
			},
			wantErr: `pointed-to value with path ../../int32 from field LeafRefToInt32 value 42 (int32 ptr) schema /int32-ref-to-leaf is empty set`,
		},
		{
			desc: "int32 points to nil with ignore missing data true",
			in: &Container{
				Container2: &Container2{LeafRefToInt32: Int32(42)},
			},
			opts: &LeafrefOptions{IgnoreMissingData: true},
		},
		{
			desc: "nil points to int32",
			in: &Container{
				Int32:      Int32(42),
				Container2: &Container2{},
			},
		},
		{
			desc: "enum",
			in: &Container{
				Int64:      42,
				Container2: &Container2{LeafRefToEnum: 42},
			},
		},
		{
			desc: "enum unequal",
			in: &Container{
				Int64:      42,
				Container2: &Container2{LeafRefToEnum: 43},
			},
			wantErr: `field name LeafRefToEnum value 43 (int64) schema path /enum-ref-to-leaf has leafref path ../../enum not equal to any target nodes`,
		},
		{
			desc: "leaf-list int32",
			in: &Container{
				LeafList:   []*int32{Int32(40), Int32(41), Int32(42)},
				Container2: &Container2{LeafRefToLeafList: Int32(42)},
			},
		},
		{
			desc: "leaf-list int32 missing",
			in: &Container{
				LeafList:   []*int32{Int32(40), Int32(41), Int32(42)},
				Container2: &Container2{LeafRefToLeafList: Int32(43)},
			},
			wantErr: `field name LeafRefToLeafList value 43 (int32 ptr) schema path /int32-ref-to-leaf-list has leafref path ../../leaf-list not equal to any target nodes`,
		},
		{
			desc: "leaf-list ref to leaf-list",
			in: &Container{
				LeafList:   []*int32{Int32(40), Int32(41), Int32(42)},
				Container2: &Container2{LeafListRefToLeafList: []*int32{Int32(41), Int32(42)}},
			},
		},
		{
			desc: "leaf-list ref to leaf-list not subset",
			in: &Container{
				LeafList:   []*int32{Int32(40), Int32(41), Int32(42)},
				Container2: &Container2{LeafListRefToLeafList: []*int32{Int32(41), Int32(42), Int32(43)}},
			},
			wantErr: `field name LeafListRefToLeafList value 43 (int32 ptr) schema path /leaf-list-ref-to-leaf-list has leafref path ../../../leaf-list not equal to any target nodes`,
		},
		{
			desc: "keyed list match",
			in: &Container{
				List: map[int32]*ListElement{
					1: {Int32(1), Int32(42)},
					2: {Int32(2), Int32(43)},
				},
				Key:        Int32(1),
				Container2: &Container2{LeafRefToList: Int32(42)},
			},
		},
		{
			desc: "keyed list unequal",
			in: &Container{
				List: map[int32]*ListElement{
					1: {Int32(1), Int32(42)},
					2: {Int32(2), Int32(43)},
				},
				Key:        Int32(1),
				Container2: &Container2{LeafRefToList: Int32(43)},
			},
			wantErr: `field name LeafRefToList value 43 (int32 ptr) schema path /int32-ref-to-list has leafref path ../../list[key = current()/../../key]/int32 not equal to any target nodes`,
		},
		{
			desc: "keyed list bad key value",
			in: &Container{
				List: map[int32]*ListElement{
					1: {Int32(1), Int32(42)},
					2: {Int32(2), Int32(43)},
				},
				Key:        Int32(3),
				Container2: &Container2{LeafRefToList: Int32(43)},
			},
			wantErr: `pointed-to value with path ../../list[key = current()/../../key]/int32 from field LeafRefToList value 43 (int32 ptr) schema /int32-ref-to-list is empty set`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			errs := ValidateLeafRefData(containerWithLeafListSchema, tt.in, tt.opts)
			if got, want := errs.String(), tt.wantErr; got != want {
				t.Errorf("%s: got error: %s, want error: %s", tt.desc, got, want)
			}
			testErrLog(t, tt.desc, errs)
		})
	}
}

func TestSplitPath(t *testing.T) {
	tests := []struct {
		desc string
		in   string
		want []string
	}{
		{
			desc: "simple",
			in:   "a/b/c",
			want: []string{"a", "b", "c"},
		},
		{
			desc: "blank",
			in:   "a//b",
			want: []string{"a", "", "b"},
		},
		{
			desc: "lead trail slash",
			in:   "/a/b/c/",
			want: []string{"", "a", "b", "c", ""},
		},
		{
			desc: "escape slash",
			in:   `a/\/b/c`,
			want: []string{"a", `\/b`, "c"},
		},
		{
			desc: "internal key slashes",
			in:   `a/b[key1 = ../x/y key2 = "z"]/c`,
			want: []string{"a", `b[key1 = ../x/y key2 = "z"]`, "c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if got, want := splitPath(tt.in), tt.want; !reflect.DeepEqual(got, want) {
				t.Errorf("%s: got: %v, want: %v", tt.desc, got, want)
			}
		})
	}
}

func TestSplitUnescaped(t *testing.T) {
	tests := []struct {
		desc string
		in   string
		want []string
	}{
		{
			desc: "simple",
			in:   "a/b/c",
			want: []string{"a", "b", "c"},
		},
		{
			desc: "blank",
			in:   "a//b",
			want: []string{"a", "", "b"},
		},
		{
			desc: "lead trail slash",
			in:   "/a/b/c/",
			want: []string{"", "a", "b", "c", ""},
		},
		{
			desc: "escape slash",
			in:   `a/\/b/c`,
			want: []string{"a", `\/b`, "c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if got, want := splitUnescaped(tt.in, '/'), tt.want; !reflect.DeepEqual(got, want) {
				t.Errorf("%s: got: %v, want: %v", tt.desc, got, want)
			}
		})
	}
}

func TestSplitUnquoted(t *testing.T) {
	tests := []struct {
		desc     string
		in       string
		splitStr string
		want     []string
	}{
		{
			desc:     "simple",
			in:       "a/b/c",
			splitStr: "/",
			want:     []string{"a", "b", "c"},
		},
		{
			desc:     "blank",
			in:       "a//b",
			splitStr: "/",
			want:     []string{"a", "", "b"},
		},
		{
			desc:     "lead trail slash",
			in:       "/a/b/c/",
			splitStr: "/",
			want:     []string{"", "a", "b", "c", ""},
		},
		{
			desc:     "quoted",
			in:       `a/"/"b"/"/c`,
			splitStr: "/",
			want:     []string{"a", `"/"b"/"`, "c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if got, want := splitUnquoted(tt.in, tt.splitStr), tt.want; !reflect.DeepEqual(got, want) {
				t.Errorf("%s: got: %v, want: %v", tt.desc, got, want)
			}
		})
	}
}

func TestExtractKeyValue(t *testing.T) {
	tests := []struct {
		desc       string
		in         string
		wantErr    string
		wantPrefix string
		wantKey    string
		wantValue  string
	}{
		{
			desc:       "literal",
			in:         `b[key = "value"]`,
			wantPrefix: "b",
			wantKey:    "key",
			wantValue:  `"value"`,
		},
		{
			desc:       "spacing",
			in:         `b[key="value"]`,
			wantPrefix: "b",
			wantKey:    "key",
			wantValue:  `"value"`,
		},
		{
			desc:       "quotes",
			in:         `b[key="[=value=]"]`,
			wantPrefix: "b",
			wantKey:    "key",
			wantValue:  `"[=value=]"`,
		},
		{
			desc:       "path",
			in:         "b[key = current()/../a/b/c]",
			wantPrefix: "b",
			wantKey:    "key",
			wantValue:  "current()/../a/b/c",
		},
		{
			desc:       "path",
			in:         "b[key = ../a/b/c]",
			wantPrefix: "b",
			wantErr:    `bad kv string key = ../a/b/c: value must be in quotes or begin with current()/`,
		},
		{
			desc:       "escapes",
			in:         `b\[[\[key\]\" = "[a]"]`,
			wantPrefix: `b\[`,
			wantKey:    `\[key\]\"`,
			wantValue:  `"[a]"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			prefix, k, v, err := extractKeyValue(tt.in)
			if got, want := errToString(err), tt.wantErr; got != want {
				t.Errorf("%s: got error: %s, want error: %s", tt.desc, got, want)
			}
			if err != nil {
				return
			}
			if got, want := prefix, tt.wantPrefix; got != want {
				t.Errorf("%s prefix: got: %s, want: %s", tt.desc, got, want)
			}
			if got, want := k, tt.wantKey; !reflect.DeepEqual(got, want) {
				t.Errorf("%s key: got: %v, want: %v", tt.desc, got, want)
			}
			if got, want := v, tt.wantValue; !reflect.DeepEqual(got, want) {
				t.Errorf("%s value: got: %v, want: %v", tt.desc, got, want)
			}
		})
	}
}

func Int32(i int32) *int32 { return &i }
