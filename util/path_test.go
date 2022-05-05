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

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/kylelemons/godebug/pretty"
	"github.com/openconfig/gnmi/errdiff"
	"github.com/openconfig/goyang/pkg/yang"
)

func TestRelativeSchemaPath(t *testing.T) {
	tests := []struct {
		desc                 string
		fieldName            string
		want                 []string
		wantPreferShadowPath []string
		wantErr              string
	}{
		{
			desc:                 "Good",
			fieldName:            "Good",
			want:                 []string{"config", "a"},
			wantPreferShadowPath: []string{"config", "a"},
		},
		{
			desc:                 "Single",
			fieldName:            "Single",
			want:                 []string{"a"},
			wantPreferShadowPath: []string{"a"},
		},
		{
			desc:                 "Both path and shadow-path",
			fieldName:            "Both",
			want:                 []string{"config", "a"},
			wantPreferShadowPath: []string{"state", "a"},
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
		testcase := func(preferShadowPath bool, want []string) {
			ft, ok := pct.FieldByName(tt.fieldName)
			if !ok {
				t.Fatal("could not find field A")
			}
			sp, err := relativeSchemaPath(ft, preferShadowPath)
			if got, want := errToString(err), tt.wantErr; got != want {
				t.Errorf("error: %s, want error: %s", got, want)
			}
			testErrLog(t, tt.desc, err)
			if err == nil {
				got, want := sp, want
				if diff := cmp.Diff(want, got); diff != "" {
					t.Errorf("(-want, +got):\n%s", diff)
				}
			}
		}
		t.Run(tt.desc, func(t *testing.T) {
			testcase(false, tt.want)
		})
		t.Run(tt.desc+"_shadowpath", func(t *testing.T) {
			testcase(true, tt.wantPreferShadowPath)
		})
	}
}

// PathContainerType is a container type for testing.
type PathContainerType struct {
	Good      *int32 `path:"a|config/a"`
	Single    *int32 `path:"a"`
	NoPath    *int32
	EmptyPath *int32 `path:""`
	Both      *int32 `path:"a|config/a" shadow-path:"a|state/a"`
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
				t.Errorf("got error: %s, want error: %s", got, want)
			}
			testErrLog(t, tt.desc, err)
			if err == nil {
				got, want := sp, tt.want
				if diff := cmp.Diff(want, got); diff != "" {
					t.Errorf("struct (-want, +got):\n%s", diff)
				}
			}
		})
	}
}

func TestSchemaTreePath(t *testing.T) {
	tests := []struct {
		name         string
		in           *yang.Entry
		want         string
		wantNoModule string
	}{{
		name: "simple entry test",
		in: &yang.Entry{
			Name: "leaf",
			Parent: &yang.Entry{
				Name: "container",
				Parent: &yang.Entry{
					Name: "module",
				},
			},
		},
		want:         "/module/container/leaf",
		wantNoModule: "/container/leaf",
	}, {
		name: "entry with a choice node",
		in: &yang.Entry{
			Name: "leaf",
			Parent: &yang.Entry{
				Name: "choice",
				Kind: yang.ChoiceEntry,
				Parent: &yang.Entry{
					Name: "container",
					Parent: &yang.Entry{
						Name: "module",
					},
				},
			},
		},
		want:         "/module/container/leaf",
		wantNoModule: "/container/leaf",
	}, {
		name: "entry with choice and case",
		in: &yang.Entry{
			Name: "leaf",
			Parent: &yang.Entry{
				Name: "case",
				Kind: yang.CaseEntry,
				Parent: &yang.Entry{
					Name: "choice",
					Kind: yang.ChoiceEntry,
					Parent: &yang.Entry{
						Name: "container",
						Parent: &yang.Entry{
							Name: "module",
						},
					},
				},
			},
		},
		want:         "/module/container/leaf",
		wantNoModule: "/container/leaf",
	}}

	for _, tt := range tests {
		got, want := SchemaTreePath(tt.in), tt.want
		if got != want {
			t.Errorf("%s: SchemaTreePath(%v): did not get expected path, got: %v, want: %v", tt.name, tt.in, got, want)
		}
		got, want = SchemaTreePathNoModule(tt.in), tt.wantNoModule
		if got != want {
			t.Errorf("%s: SchemaTreePathNoModule(%v): did not get expected path, got: %v, want: %v", tt.name, tt.in, got, want)
		}
	}
}

// addParents adds parent pointers for a schema tree.
func addParents(e *yang.Entry) {
	for _, c := range e.Dir {
		c.Parent = e
		addParents(c)
	}
}

func getContainerWithChoiceSchema() *yang.Entry {
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
	addParents(containerWithChoiceSchema)

	return containerWithChoiceSchema
}

func TestFirstChild(t *testing.T) {
	containerWithChoiceSchema := getContainerWithChoiceSchema()
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

	if got := FirstChild(containerWithChoiceSchema, nil); got != nil {
		t.Fatalf("nil path: got:\n%s\nwant: nil", pretty.Sprint(got))
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			cs := FirstChild(containerWithChoiceSchema, strings.Split(tt.path, "/"))
			if got, want := cs, tt.want; got != want {
				t.Errorf("got:\n%s\nwant:\n%s\n", pretty.Sprint(got), pretty.Sprint(want))
			}
		})
	}
}

func TestSchemaPathNoChoiceCase(t *testing.T) {
	containerWithChoiceSchema := getContainerWithChoiceSchema()

	tests := []struct {
		desc  string
		entry *yang.Entry
		want  []string
	}{
		{
			desc:  "nil entry",
			entry: nil,
			want:  []string{},
		},
		{
			desc:  "choice entry",
			entry: containerWithChoiceSchema.Dir["choice1"],
			want:  []string{"container-with-choice-schema"},
		},
		{
			desc:  "case entry",
			entry: containerWithChoiceSchema.Dir["choice1"].Dir["case1"],
			want:  []string{"container-with-choice-schema"},
		},
		{
			desc:  "case1-leaf1",
			entry: containerWithChoiceSchema.Dir["choice1"].Dir["case1"].Dir["case1-leaf1"],
			want:  []string{"container-with-choice-schema", "case1-leaf1"},
		},
		{
			desc:  "case21-leaf",
			entry: containerWithChoiceSchema.Dir["choice1"].Dir["case2"].Dir["case2_choice1"].Dir["case21"].Dir["case21-leaf"],
			want:  []string{"container-with-choice-schema", "case21-leaf"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if got, want := SchemaPathNoChoiceCase(tt.entry), tt.want; !cmp.Equal(got, want, cmpopts.EquateEmpty()) {
				t.Errorf("got:\n%s\nwant:\n%s\n", pretty.Sprint(got), pretty.Sprint(want))
			}
		})
	}
}

func TestSchemaEntryPathNoChoiceCase(t *testing.T) {
	containerWithChoiceSchema := getContainerWithChoiceSchema()

	tests := []struct {
		desc  string
		entry *yang.Entry
		want  []*yang.Entry
	}{
		{
			desc:  "nil entry",
			entry: nil,
			want:  nil,
		},
		{
			desc:  "choice entry",
			entry: containerWithChoiceSchema.Dir["choice1"],
			want:  []*yang.Entry{containerWithChoiceSchema},
		},
		{
			desc:  "case entry",
			entry: containerWithChoiceSchema.Dir["choice1"].Dir["case1"],
			want:  []*yang.Entry{containerWithChoiceSchema},
		},
		{
			desc:  "case1-leaf1",
			entry: containerWithChoiceSchema.Dir["choice1"].Dir["case1"].Dir["case1-leaf1"],
			want:  []*yang.Entry{containerWithChoiceSchema, containerWithChoiceSchema.Dir["choice1"].Dir["case1"].Dir["case1-leaf1"]},
		},
		{
			desc:  "case21-leaf",
			entry: containerWithChoiceSchema.Dir["choice1"].Dir["case2"].Dir["case2_choice1"].Dir["case21"].Dir["case21-leaf"],
			want:  []*yang.Entry{containerWithChoiceSchema, containerWithChoiceSchema.Dir["choice1"].Dir["case2"].Dir["case2_choice1"].Dir["case21"].Dir["case21-leaf"]},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if got, want := SchemaEntryPathNoChoiceCase(tt.entry), tt.want; !reflect.DeepEqual(got, want) {
				t.Errorf("got:\n%v\nwant:\n%v\n", got, want)
			}
		})
	}
}

func TestRemoveXPATHPredicates(t *testing.T) {
	tests := []struct {
		desc    string
		in      string
		want    string
		wantErr bool
	}{{
		desc: "simple predicate",
		in:   `/foo/bar[name="eth0"]`,
		want: "/foo/bar",
	}, {
		desc: "predicate with path",
		in:   `/foo/bar[name="/foo/bar/baz"]/config/hat`,
		want: "/foo/bar/config/hat",
	}, {
		desc: "predicate with function",
		in:   `/foo/bar[name="current()/../interface"]/config/baz`,
		want: "/foo/bar/config/baz",
	}, {
		desc: "multiple predicates",
		in:   `/foo/bar[name="current()/../interface"]/container/list[key="42"]/config/foo`,
		want: "/foo/bar/container/list/config/foo",
	}, {
		desc: "a real example",
		in:   `/oc-if:interfaces/oc-if:interface[oc-if:name=current()/../interface]/oc-if:subinterfaces/oc-if:subinterface/oc-if:index`,
		want: "/oc-if:interfaces/oc-if:interface/oc-if:subinterfaces/oc-if:subinterface/oc-if:index",
	}, {
		desc:    "] without [",
		in:      `/foo/bar]`,
		wantErr: true,
	}, {
		desc:    "[ without closure",
		in:      `/foo/bar[`,
		wantErr: true,
	}, {
		desc: "multiple predicates, end of string",
		in:   `/foo/bar/name[e="1"]/bar[j="2"]`,
		want: "/foo/bar/name/bar",
	}, {
		desc:    "][ in incorrect order",
		in:      `/foo/bar][`,
		wantErr: true,
	}, {
		desc: "empty string",
		in:   ``,
		want: ``,
	}, {
		desc: "predicate directly",
		in:   `foo[bar="test"]`,
		want: `foo`,
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got, err := removeXPATHPredicates(tt.in)
			if (err != nil) != tt.wantErr {
				t.Errorf("removeXPATHPredicates(%s): got unexpected error, got: %v", tt.in, err)
			}

			if got != tt.want {
				t.Errorf("removePredicate(%v): did not get expected value, got: %v, want: %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestFindLeafRefSchema(t *testing.T) {
	choiceContainer := &yang.Entry{
		Name:   "container",
		Kind:   yang.DirectoryEntry,
		Parent: &yang.Entry{Name: "module"},
		Dir: map[string]*yang.Entry{
			"choice-node": {
				Name:   "choice-node",
				Kind:   yang.ChoiceEntry,
				Parent: &yang.Entry{Name: "container"},
				Dir: map[string]*yang.Entry{
					"case-one": {
						Name: "case-one",
						Kind: yang.CaseEntry,
						Parent: &yang.Entry{
							Name: "choice-node",
							Kind: yang.ChoiceEntry,
							Parent: &yang.Entry{
								Name: "container",
								Parent: &yang.Entry{
									Name: "module",
								},
							},
						},
						Dir: map[string]*yang.Entry{
							"second-container": {
								Name: "second-container",
								Kind: yang.DirectoryEntry,
								Parent: &yang.Entry{
									Name: "case-one",
									Kind: yang.CaseEntry,
									Parent: &yang.Entry{
										Name: "choice-node",
										Kind: yang.ChoiceEntry,
										Parent: &yang.Entry{
											Name: "container",
											Parent: &yang.Entry{
												Name: "module",
											},
										},
									},
								},
								Dir: map[string]*yang.Entry{
									"config": {
										Name: "config",
										Kind: yang.DirectoryEntry,
										Parent: &yang.Entry{
											Name: "second-container",
											Parent: &yang.Entry{
												Name: "case-one",
												Kind: yang.CaseEntry,
												Parent: &yang.Entry{
													Name: "choice-node",
													Kind: yang.ChoiceEntry,
													Parent: &yang.Entry{
														Name: "container",
														Parent: &yang.Entry{
															Name: "module",
														},
													},
												},
											},
										},
										Dir: map[string]*yang.Entry{"leaf-one": {Name: "leaf-one", Type: &yang.YangType{Kind: yang.Ystring}}},
									},
								},
							},
						},
					},
					"case-two": {
						Name: "case-two",
						Kind: yang.CaseEntry,
						Parent: &yang.Entry{
							Name: "choice-node",
							Kind: yang.ChoiceEntry,
							Parent: &yang.Entry{
								Name: "container",
								Parent: &yang.Entry{
									Name: "module",
								},
							},
						},
						Dir: map[string]*yang.Entry{
							"third-container": {
								Name: "third-container",
								Kind: yang.DirectoryEntry,
								Parent: &yang.Entry{
									Name: "case-two",
									Kind: yang.CaseEntry,
									Parent: &yang.Entry{
										Name: "choice-node",
										Kind: yang.ChoiceEntry,
										Parent: &yang.Entry{
											Name: "container",
											Parent: &yang.Entry{
												Name: "module",
											},
										},
									},
								},
								Dir: map[string]*yang.Entry{
									"config": {
										Name: "config",
										Kind: yang.DirectoryEntry,
										Parent: &yang.Entry{
											Name: "third-container",
											Parent: &yang.Entry{
												Name: "case-two",
												Kind: yang.CaseEntry,
												Parent: &yang.Entry{
													Name: "choice-node",
													Kind: yang.ChoiceEntry,
													Parent: &yang.Entry{
														Name: "container",
														Parent: &yang.Entry{
															Name: "module",
														},
													},
												},
											},
										},
										Dir: map[string]*yang.Entry{"leaf-two": {Name: "leaf-two", Type: &yang.YangType{Kind: yang.Yleafref}}},
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
		desc      string
		inSchema  *yang.Entry
		inPathStr string
		wantEntry *yang.Entry
		wantErr   string
	}{{
		desc: "simple reference",
		inSchema: &yang.Entry{
			Name: "referencing",
			Type: &yang.YangType{
				Kind: yang.Yleafref,
				Path: "../foo",
			},
			Parent: &yang.Entry{
				Name: "directory",
				Kind: yang.DirectoryEntry,
				Dir: map[string]*yang.Entry{
					"foo": {
						Name: "foo",
						Type: &yang.YangType{Kind: yang.Ystring},
					},
				},
			},
		},
		inPathStr: "../foo",
		wantEntry: &yang.Entry{
			Name: "foo",
			Type: &yang.YangType{Kind: yang.Ystring},
		},
	}, {
		desc: "forward reference with choice and case statements that should be skipped",
		inSchema: &yang.Entry{
			Name: "module",
			Dir: map[string]*yang.Entry{
				"container": choiceContainer,
			},
		},
		inPathStr: "/container/third-container/config/leaf-two",
		wantEntry: &yang.Entry{
			Name: "leaf-two",
			Type: &yang.YangType{Kind: yang.Yleafref},
		},
	}, {
		desc: "backwards and forwards reference within choice and case",
		inSchema: &yang.Entry{
			Name: "leaf-two",
			Type: &yang.YangType{
				Kind: yang.Yleafref,
				Path: "../../../second-container/config/leaf-one",
			},
			Parent: &yang.Entry{
				Name: "config",
				Parent: &yang.Entry{
					Name: "third-container",
					Parent: &yang.Entry{
						Name: "case-two",
						Kind: yang.CaseEntry,
						Parent: &yang.Entry{
							Name: "choice-node",
							Kind: yang.ChoiceEntry,
							Parent: &yang.Entry{
								Name: "container",
								Parent: &yang.Entry{
									Name: "module",
								},
								Dir: choiceContainer.Dir,
							},
						},
					},
				},
			},
		},
		inPathStr: "../../../second-container/config/leaf-one",
		wantEntry: &yang.Entry{
			Name: "leaf-one",
			Type: &yang.YangType{Kind: yang.Ystring},
		},
	}, {
		desc: "empty path",
		inSchema: &yang.Entry{
			Name: "referencing",
			Type: &yang.YangType{
				Kind: yang.Yleafref,
			},
		},
		wantErr: "leafref schema referencing has empty path",
	}, {
		desc: "bad xpath predicate, mismatched []s",
		inSchema: &yang.Entry{
			Name: "referencing",
			Type: &yang.YangType{
				Kind: yang.Yleafref,
				Path: "/interfaces/interface[name=foo/bar",
			},
		},
		inPathStr: "/interfaces/interface[name=foo/bar",
		wantErr:   "mismatched brackets within substring /interfaces/interface[name=foo/bar of /interfaces/interface[name=foo/bar, [ pos: 21, ] pos: -1",
	}, {
		desc: "with xpath predicate",
		inSchema: &yang.Entry{
			Name: "module",
			Kind: yang.DirectoryEntry,
			Dir: map[string]*yang.Entry{
				"interfaces": {
					Name: "interfaces",
					Kind: yang.DirectoryEntry,
					Dir: map[string]*yang.Entry{
						"interface": {
							Name:     "interface",
							Kind:     yang.DirectoryEntry,
							ListAttr: &yang.ListAttr{},
							Dir: map[string]*yang.Entry{
								"name": {
									Name: "name",
									Kind: yang.LeafEntry,
									Type: &yang.YangType{
										Kind: yang.Yleafref,
										Path: "../state/name",
									},
								},
								"state": {
									Name: "state",
									Kind: yang.DirectoryEntry,
									Dir: map[string]*yang.Entry{
										"name": {
											Name: "name",
											Kind: yang.LeafEntry,
											Type: &yang.YangType{Kind: yang.Ystring},
										},
									},
								},
								"subinterface": {
									Name: "subinterface",
									Kind: yang.LeafEntry,
									Type: &yang.YangType{Kind: yang.Ystring},
								},
							},
						},
					},
				},
			},
		},
		inPathStr: "/oc-if:interfaces/oc-if:interface[oc-if:name=current()/../interface]/oc-if:subinterface",
		wantEntry: &yang.Entry{
			Name: "subinterface",
			Kind: yang.LeafEntry,
			Type: &yang.YangType{Kind: yang.Ystring},
		},
	}, {
		desc: "strip prefix error in path",
		inSchema: &yang.Entry{
			Name: "referencing",
			Type: &yang.YangType{
				Kind: yang.Yleafref,
				Path: "/interface:foo:bar/baz",
			},
		},
		inPathStr: "/interface:foo:bar/baz",
		wantErr:   "leafref schema referencing path /interface:foo:bar/baz: path element did not form a valid name (name, prefix:name): interface:foo:bar",
	}, {
		desc: "nil reference",
		inSchema: &yang.Entry{
			Name: "referencing",
			Type: &yang.YangType{
				Kind: yang.Yleafref,
				Path: "/interfaces/interface/baz",
			},
		},
		inPathStr: "/interfaces/interface/baz",
		wantErr:   "schema node interfaces is nil for leafref schema referencing with path /interfaces/interface/baz",
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got, err := FindLeafRefSchema(tt.inSchema, tt.inPathStr)
			if err != nil && err.Error() != tt.wantErr {
				t.Errorf("FindLeafRefSchema(%v, %s): did not get expected error, got: %v, want: %v", tt.inSchema, tt.inPathStr, err, tt.wantErr)
			}

			if err != nil {
				return
			}

			if diff := pretty.Compare(got, tt.wantEntry); diff != "" {
				t.Errorf("FindLeafRefSchema(%v, %s): did not get expected entry, diff(-got,+want):\n%s", tt.inSchema, tt.inPathStr, diff)
			}
		})
	}
}

func TestStripModulePrefix(t *testing.T) {
	tests := []struct {
		desc     string
		inName   string
		wantName string
		wantErr  string
	}{{
		desc:     "valid with prefix",
		inName:   "one:two",
		wantName: "two",
	}, {
		desc:     "valid without prefix",
		inName:   "two",
		wantName: "two",
	}, {
		desc:    "invalid input",
		inName:  "foo:bar:foo",
		wantErr: "path element did not form a valid name (name, prefix:name): foo:bar:foo",
	}, {
		desc:     "empty string",
		inName:   "",
		wantName: "",
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got, err := stripModulePrefixWithCheck(tt.inName)
			if err != nil && err.Error() != tt.wantErr {
				t.Errorf("stripModulePrefixWithCheck(%v): did not get expected error, got: %v, want: %s", tt.inName, got, tt.wantErr)
			}

			if err != nil {
				return
			}

			if got != tt.wantName {
				t.Errorf("stripModulePrefixWithCheck(%v): did not get expected name, got: %s, want: %s", tt.inName, got, tt.wantName)
			}

			if got, want := StripModulePrefix(tt.inName), tt.wantName; got != want {
				t.Errorf("StripModulePrefix(%v): did not get expected name, got: %s, want: %s", tt.inName, got, want)
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
				t.Errorf("got: %v want: %v", got, want)
			}
		})
	}
}

func TestReplacePathSuffix(t *testing.T) {
	tests := []struct {
		desc             string
		inName           string
		inNewSuffix      string
		wantName         string
		wantErrSubstring string
	}{{
		desc:        "valid with prefix",
		inName:      "one:two",
		inNewSuffix: "three",
		wantName:    "one:three",
	}, {
		desc:        "valid without prefix",
		inName:      "two",
		inNewSuffix: "three",
		wantName:    "three",
	}, {
		desc:             "invalid input",
		inName:           "foo:bar:foo",
		inNewSuffix:      "three",
		wantErrSubstring: "path element did not form a valid name (name, prefix:name)",
	}, {
		desc:        "empty string",
		inName:      "",
		inNewSuffix: "foo",
		wantName:    "foo",
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got, err := ReplacePathSuffix(tt.inName, tt.inNewSuffix)
			if diff := errdiff.Substring(err, tt.wantErrSubstring); diff != "" {
				t.Errorf("ReplacePathSuffix(%v, %v): did not get expected error:%s", tt.inName, tt.inNewSuffix, diff)
			}

			if err != nil {
				return
			}

			if got != tt.wantName {
				t.Errorf("ReplacePathSuffix(%v, %v): did not get expected name, got: %s, want: %s", tt.inName, tt.inNewSuffix, got, tt.wantName)
			}

		})
	}
}

func TestSplitPath(t *testing.T) {
	tests := []struct {
		desc                      string
		in                        string
		want                      []string
		wantIgnoreLeadingTrailing []string
	}{
		{
			desc:                      "simple",
			in:                        "a/b/c",
			want:                      []string{"a", "b", "c"},
			wantIgnoreLeadingTrailing: []string{"a", "b", "c"},
		},
		{
			desc:                      "empty",
			in:                        "",
			want:                      nil,
			wantIgnoreLeadingTrailing: nil,
		},
		{
			desc:                      "one slash",
			in:                        "/",
			want:                      []string{""},
			wantIgnoreLeadingTrailing: nil,
		},
		{
			desc:                      "leading slash",
			in:                        "/a",
			want:                      []string{"", "a"},
			wantIgnoreLeadingTrailing: []string{"a"},
		},
		{
			desc:                      "trailing slash",
			in:                        "aa/",
			want:                      []string{"aa", ""},
			wantIgnoreLeadingTrailing: []string{"aa"},
		},
		{
			desc:                      "blank",
			in:                        "a//b",
			want:                      []string{"a", "", "b"},
			wantIgnoreLeadingTrailing: []string{"a", "", "b"},
		},
		{
			desc:                      "lead trail slash",
			in:                        "/a/b/c/",
			want:                      []string{"", "a", "b", "c", ""},
			wantIgnoreLeadingTrailing: []string{"a", "b", "c"},
		},
		{
			desc:                      "double lead trail slash",
			in:                        "//a/b/c//",
			want:                      []string{"", "", "a", "b", "c", "", ""},
			wantIgnoreLeadingTrailing: []string{"", "a", "b", "c", ""},
		},
		{
			desc:                      "escape slash",
			in:                        `a/\/b/c`,
			want:                      []string{"a", "/b", "c"},
			wantIgnoreLeadingTrailing: []string{"a", "/b", "c"},
		},
		{
			desc:                      "internal key slashes",
			in:                        `a/b[key1 = ../x/y key2 = "z"]/c`,
			want:                      []string{"a", `b[key1 = ../x/y key2 = "z"]`, "c"},
			wantIgnoreLeadingTrailing: []string{"a", `b[key1 = ../x/y key2 = "z"]`, "c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if diff := cmp.Diff(tt.want, SplitPath(tt.in), cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("SplitPath (-want, +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.wantIgnoreLeadingTrailing, PathStringToElements(tt.in), cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("PathStringToElements (-want, +got):\n%s", diff)
			}
		})
	}
}

// TestSlicePathToString tests functions related to YANG paths - these are strings of
// the form /a/b/c/d.
func TestSlicePathToString(t *testing.T) {
	tests := []struct {
		name             string
		inSplitPath      []string
		wantStringPath   string
		wantStrippedPath string
	}{{
		name:           "path without attributes",
		inSplitPath:    []string{"", "a", "b", "c", "d"},
		wantStringPath: "/a/b/c/d",
	}, {
		name:           "path with attributes",
		inSplitPath:    []string{"", "a", "b[key=1]", "c", "d"},
		wantStringPath: "/a/b[key=1]/c/d",
	}, {
		name:             "path with prefixes",
		inSplitPath:      []string{"", "pfx:a", "pfx:b", "pfx:c", "pfx:d"},
		wantStringPath:   "/pfx:a/pfx:b/pfx:c/pfx:d",
		wantStrippedPath: "/a/b/c/d",
	}}

	for _, tt := range tests {
		if got := SlicePathToString(tt.inSplitPath); got != tt.wantStringPath {
			t.Errorf("%s: SlicePathToString(%v) = %s, want %s", tt.name, tt.inSplitPath, got, tt.wantStringPath)
		}

		if tt.wantStrippedPath != "" {
			var s []string
			for _, p := range tt.inSplitPath {
				p := StripModulePrefix(p)
				s = append(s, p)
			}

			if got := SlicePathToString(s); got != tt.wantStrippedPath {
				t.Errorf("%s: StripModulePrefix(%v) = %s, want %s", tt.name, tt.inSplitPath, got, tt.wantStrippedPath)
			}
		}
	}
}
