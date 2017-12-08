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
	"reflect"
	"strings"
	"testing"

	"github.com/kylelemons/godebug/pretty"
	"github.com/openconfig/goyang/pkg/yang"
)

// populateParentField recurses through schema and populates each Parent field
// with the parent schema node ptr.
func populateParentField(parent, schema *yang.Entry) {
	schema.Parent = parent
	for _, e := range schema.Dir {
		populateParentField(schema, e)
	}
}

func TestIsLeafRef(t *testing.T) {
	tests := []struct {
		desc   string
		schema *yang.Entry
		want   bool
	}{
		{
			desc:   "nil schema",
			schema: nil,
			want:   false,
		},
		{
			desc:   "nil Type",
			schema: &yang.Entry{},
			want:   false,
		},
		{
			desc: "int32 type",
			schema: &yang.Entry{
				Kind: yang.LeafEntry,
				Type: &yang.YangType{
					Kind: yang.Yint32,
				},
			},
			want: false,
		},
		{
			desc: "leafref type",
			schema: &yang.Entry{
				Kind: yang.LeafEntry,
				Type: &yang.YangType{
					Kind: yang.Yleafref,
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if got, want := IsLeafRef(tt.schema), tt.want; got != want {
				t.Errorf("%s: got: %v want: %v", tt.desc, got, want)
			}
		})
	}
}

func TestIsChoiceOrCase(t *testing.T) {
	tests := []struct {
		desc   string
		schema *yang.Entry
		want   bool
	}{
		{
			desc:   "nil schema",
			schema: nil,
			want:   false,
		},
		{
			desc: "leaf type",
			schema: &yang.Entry{
				Kind: yang.LeafEntry,
				Type: &yang.YangType{
					Kind: yang.Yint32,
				},
			},
			want: false,
		},
		{
			desc: "choice type",
			schema: &yang.Entry{
				Kind: yang.ChoiceEntry,
			},
			want: true,
		},
		{
			desc: "case type",
			schema: &yang.Entry{
				Kind: yang.CaseEntry,
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if got, want := IsChoiceOrCase(tt.schema), tt.want; got != want {
				t.Errorf("%s: got: %v want: %v", tt.desc, got, want)
			}
		})
	}
}

func TestIsFakeRoot(t *testing.T) {
	tests := []struct {
		desc   string
		schema *yang.Entry
		want   bool
	}{
		{
			desc:   "nil schema",
			schema: nil,
			want:   false,
		},
		{
			desc: "not fakeroot",
			schema: &yang.Entry{
				Kind:       yang.DirectoryEntry,
				Annotation: map[string]interface{}{},
			},
			want: false,
		},
		{
			desc: "fakeroot",
			schema: &yang.Entry{
				Kind:       yang.DirectoryEntry,
				Annotation: map[string]interface{}{"isFakeRoot": nil},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if got, want := IsFakeRoot(tt.schema), tt.want; got != want {
				t.Errorf("%s: got: %v want: %v", tt.desc, got, want)
			}
		})
	}
}

func TestIsUnkeyedList(t *testing.T) {
	tests := []struct {
		desc   string
		schema *yang.Entry
		want   bool
	}{
		{
			desc:   "nil schema",
			schema: nil,
			want:   false,
		},
		{
			desc: "leaf type",
			schema: &yang.Entry{
				Kind: yang.LeafEntry,
				Type: &yang.YangType{
					Kind: yang.Yint32,
				},
			},
			want: false,
		},
		{
			desc: "keyed list",
			schema: &yang.Entry{
				Kind:     yang.DirectoryEntry,
				ListAttr: &yang.ListAttr{MinElements: &yang.Value{Name: "0"}},
				Key:      "key",
				Dir:      map[string]*yang.Entry{},
			},
			want: false,
		},
		{
			desc: "unkeyed list",
			schema: &yang.Entry{
				Kind:     yang.DirectoryEntry,
				ListAttr: &yang.ListAttr{MinElements: &yang.Value{Name: "0"}},
				Dir:      map[string]*yang.Entry{},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if got, want := IsUnkeyedList(tt.schema), tt.want; got != want {
				t.Errorf("%s: got: %v want: %v", tt.desc, got, want)
			}
		})
	}
}

// PathContainerType is a container type for testing.
type PathContainerType struct {
	Good      *int32 `path:"a|config/a"`
	NoPath    *int32
	EmptyPath *int32 `path:""`
}

// IsYANGGoStruct implements the GoStruct interface method.
func (*PathContainerType) IsYANGGoStruct() {}

func TestSchemaPaths(t *testing.T) {
	tests := []struct {
		desc      string
		fieldName string
		want      [][]string
		wantErr   string
	}{
		{
			desc:      "Good",
			fieldName: "Good",
			want:      [][]string{{"a"}, {"config", "a"}},
		},
		{
			desc:      "NoPath",
			fieldName: "NoPath",
			wantErr:   `field NoPath did not specify a path`,
		},
		{
			desc:      "EmptyPath",
			fieldName: "EmptyPath",
			wantErr:   `field EmptyPath did not specify a path`,
		},
	}

	pct := reflect.TypeOf(PathContainerType{})

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			ft, ok := pct.FieldByName(tt.fieldName)
			if !ok {
				t.Fatal("could not find field A")
			}
			sp, err := SchemaPaths(ft)
			if got, want := errToString(err), tt.wantErr; got != want {
				t.Errorf("%s: got error: %s, want error: %s", tt.desc, got, want)
			}
			testErrLog(t, tt.desc, err)
			if err == nil {
				if got, want := sp, tt.want; !reflect.DeepEqual(got, want) {
					t.Errorf("%s: struct got:%v want: %v", tt.desc, pretty.Sprint(got), pretty.Sprint(want))
				}
			}
		})
	}
}

func TestChildSchema(t *testing.T) {
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
								Name: "case1-leaf1",
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

	tests := []struct {
		desc string
		path string
		want *yang.Entry
	}{
		{
			desc: "empty path",
			path: "",
			want: nil,
		},
		{
			desc: "bad path",
			path: "bad-path",
			want: nil,
		},
		{
			desc: "non-leaf node",
			path: "case1",
			want: nil,
		},
		{
			desc: "case1-leaf1",
			path: "case1-leaf1",
			want: containerWithChoiceSchema.Dir["choice1"].Dir["case1"].Dir["case1-leaf1"],
		},
		{
			desc: "case21-leaf",
			path: "case21-leaf",
			want: containerWithChoiceSchema.Dir["choice1"].Dir["case2"].Dir["case2_choice1"].Dir["case21"].Dir["case21-leaf"],
		},
	}

	if got := ChildSchema(containerWithChoiceSchema, nil); got != nil {
		t.Fatalf("nil path: got:\n%s\nwant: nil", pretty.Sprint(got))
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			cs := ChildSchema(containerWithChoiceSchema, strings.Split(tt.path, "/"))
			if got, want := cs, tt.want; got != want {
				t.Errorf("%s: got:\n%s\nwant:\n%s\n", tt.desc, pretty.Sprint(got), pretty.Sprint(want))
			}
		})
	}
}

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
					"leaf-list-with-leafref": {
						Name: "leaf-list-with-leafref",
						Kind: yang.LeafEntry,
						Type: &yang.YangType{
							Kind: yang.Yleafref,
							Path: "../../int32",
						},
						ListAttr: &yang.ListAttr{MinElements: &yang.Value{Name: "0"}},
					},
					"absolute-to-int32": {
						Name: "absolute-to-int32",
						Kind: yang.LeafEntry,
						Type: &yang.YangType{
							Kind: yang.Yleafref,
							Path: "/int32",
						},
						ListAttr: &yang.ListAttr{MinElements: &yang.Value{Name: "0"}},
					},
					"recursive": {
						Name: "recursive",
						Kind: yang.LeafEntry,
						Type: &yang.YangType{
							Kind: yang.Yleafref,
							Path: "../leaf-list-with-leafref",
						},
						ListAttr: &yang.ListAttr{MinElements: &yang.Value{Name: "0"}},
					},
					"bad-path": {
						Name: "bad-path",
						Kind: yang.LeafEntry,
						Type: &yang.YangType{
							Kind: yang.Yleafref,
							Path: "../../missing",
						},
						ListAttr: &yang.ListAttr{MinElements: &yang.Value{Name: "0"}},
					},
					"missing-path": {
						Name: "missing-path",
						Kind: yang.LeafEntry,
						Type: &yang.YangType{
							Kind: yang.Yleafref,
						},
						ListAttr: &yang.ListAttr{MinElements: &yang.Value{Name: "0"}},
					},
				},
			},
		},
	}

	emptySchema := &yang.Entry{}

	tests := []struct {
		desc    string
		in      *yang.Entry
		want    *yang.Entry
		wantErr string
	}{
		{
			desc: "nil",
			in:   nil,
			want: nil,
		},
		{
			desc: "nil Type",
			in:   emptySchema,
			want: emptySchema,
		},
		{
			desc: "leaf-list",
			in:   containerWithLeafListSchema.Dir["leaf-list"],
			want: containerWithLeafListSchema.Dir["leaf-list"],
		},
		{
			desc: "list/int32",
			in:   containerWithLeafListSchema.Dir["list"].Dir["int32"],
			want: containerWithLeafListSchema.Dir["list"].Dir["int32"],
		},
		{
			desc: "int32",
			in:   containerWithLeafListSchema.Dir["int32"],
			want: containerWithLeafListSchema.Dir["int32"],
		},
		{
			desc: "enum",
			in:   containerWithLeafListSchema.Dir["enum"],
			want: containerWithLeafListSchema.Dir["enum"],
		},
		{
			desc: "container2/int32-ref-to-leaf",
			in:   containerWithLeafListSchema.Dir["container2"].Dir["int32-ref-to-leaf"],
			want: containerWithLeafListSchema.Dir["int32"],
		},
		{
			desc: "container2/enum-ref-to-leaf",
			in:   containerWithLeafListSchema.Dir["container2"].Dir["enum-ref-to-leaf"],
			want: containerWithLeafListSchema.Dir["enum"],
		},
		{
			desc: "container2/int32-ref-to-leaf-list",
			in:   containerWithLeafListSchema.Dir["container2"].Dir["int32-ref-to-leaf-list"],
			want: containerWithLeafListSchema.Dir["leaf-list"],
		},
		{
			desc: "container2/leaf-list-with-leafref",
			in:   containerWithLeafListSchema.Dir["container2"].Dir["leaf-list-with-leafref"],
			want: containerWithLeafListSchema.Dir["int32"],
		},
		{
			desc: "container2/recursive",
			in:   containerWithLeafListSchema.Dir["container2"].Dir["absolute-to-int32"],
			want: containerWithLeafListSchema.Dir["int32"],
		},
		{
			desc: "container2/absolute-to-int32",
			in:   containerWithLeafListSchema.Dir["container2"].Dir["absolute-to-int32"],
			want: containerWithLeafListSchema.Dir["int32"],
		},
		{
			desc:    "container2/bad-path",
			in:      containerWithLeafListSchema.Dir["container2"].Dir["bad-path"],
			wantErr: `schema node missing is nil for leafref schema bad-path with path ../../missing`,
		},
		{
			desc:    "container2/missing-path",
			in:      containerWithLeafListSchema.Dir["container2"].Dir["missing-path"],
			wantErr: `leafref schema missing-path has empty path`,
		},
	}

	populateParentField(nil, containerWithLeafListSchema)

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			s, err := ResolveIfLeafRef(tt.in)
			if got, want := errToString(err), tt.wantErr; got != want {
				t.Errorf("%s: got error: %s, want error: %s", tt.desc, got, want)
			}
			testErrLog(t, tt.desc, err)
			if err == nil {
				if got, want := s, tt.want; got != want {
					t.Errorf("%s: struct got:\n%v\n want:\n%v\n", tt.desc, pretty.Sprint(got), pretty.Sprint(want))
				}
			}
		})
	}
}

func TestStripModulePrefixesStr(t *testing.T) {
	tests := []struct {
		desc string
		path string
		want string
	}{
		{
			desc: "empty",
			path: "",
			want: "",
		},
		{
			desc: "root",
			path: "/",
			want: "/",
		},
		{
			desc: "one element",
			path: "module-a:element",
			want: "element",
		},
		{
			desc: "relative",
			path: "../../module-a:element",
			want: "../../element",
		},
		{
			desc: "one element, trailing slash",
			path: "module-a:element/",
			want: "element/",
		},
		{
			desc: "multi element",
			path: "/module-a:element-a/module-b:element-b/element-c",
			want: "/element-a/element-b/element-c",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if got, want := StripModulePrefixesStr(tt.path), tt.want; got != want {
				t.Errorf("%s: got: %v want: %v", tt.desc, got, want)
			}
		})
	}
}
