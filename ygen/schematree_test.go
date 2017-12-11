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

package ygen

import (
	"reflect"
	"testing"

	"github.com/kylelemons/godebug/pretty"
	"github.com/openconfig/goyang/pkg/yang"
)

// wantTreeEntry describes an entry that is expected within a tree
// when testing the schematree.
type wantTreeEntry struct {
	path  []string
	value *yang.Entry
}

func TestBuildSchemaTree(t *testing.T) {
	tests := []struct {
		name         string
		inEntries    []*yang.Entry
		wantElements []wantTreeEntry
	}{{
		name: "simple single branch tree",
		inEntries: []*yang.Entry{
			{
				Name: "root-entity",
				Parent: &yang.Entry{
					Name: "module",
				},
				Dir: map[string]*yang.Entry{
					"child-one": {
						Name: "child-one",
						Parent: &yang.Entry{
							Name: "root-entity",
							Parent: &yang.Entry{
								Name: "module",
							},
						},
					},
					"child-two": {
						Name: "child-two",
						Parent: &yang.Entry{
							Name: "root-entity",
							Parent: &yang.Entry{
								Name: "module",
							},
						},
					},
					"child-dir": {
						Name: "grandchild-a",
						Parent: &yang.Entry{
							Name: "child-dir",
							Parent: &yang.Entry{
								Name: "root-entity",
								Parent: &yang.Entry{
									Name: "module",
								},
							},
						},
					},
				},
			},
		},
		wantElements: []wantTreeEntry{
			{
				path: []string{"root-entity", "child-one"},
				value: &yang.Entry{
					Name: "child-one",
					Parent: &yang.Entry{
						Name: "root-entity",
						Parent: &yang.Entry{
							Name: "module",
						},
					},
				},
			}, {
				path: []string{"root-entity", "child-two"},
				value: &yang.Entry{
					Name: "child-two",
					Parent: &yang.Entry{
						Name: "root-entity",
						Parent: &yang.Entry{
							Name: "module",
						},
					},
				},
			}, {
				path: []string{"root-entity", "child-dir", "grandchild-a"},
				value: &yang.Entry{
					Name: "grandchild-a",
					Parent: &yang.Entry{
						Name: "child-dir",
						Parent: &yang.Entry{
							Name: "root-entity",
							Parent: &yang.Entry{
								Name: "module",
							},
						},
					},
				},
			},
		},
	}}

	for _, tt := range tests {
		got, err := buildSchemaTree(tt.inEntries)
		if err != nil {
			t.Errorf("%s: buildSchemaTree(%v): got unexpected error building tree: %v", tt.name, tt.inEntries, err)
			continue
		}

		for _, want := range tt.wantElements {
			gotElement := got.GetLeafValue(want.path)
			if diff := pretty.Compare(gotElement, want.value); diff != "" {
				t.Errorf("%s: buildSchemaTree(%v): got incorrect value for element %v, diff(-got,+want)\n:%s", tt.name, tt.inEntries, want.path, diff)
				continue
			}
		}
	}
}

func TestFixSchemaTreePath(t *testing.T) {
	tests := []struct {
		name      string
		inPath    string
		inContext *yang.Entry
		wantParts []string
		wantErr   bool
	}{{
		name:      "simple path that does not need to be adjusted",
		inPath:    "/system/config/hostname",
		wantParts: []string{"system", "config", "hostname"},
	}, {
		name:      "path with keys in that should be removed",
		inPath:    "/interfaces/interface[name=current()/../config/name]/config/admin-status",
		wantParts: []string{"interfaces", "interface", "config", "admin-status"},
	}, {
		name:      "path with namespaces to be removed",
		inPath:    "/oc-if:interfaces/oc-if:interface/oc-if:config/name",
		wantParts: []string{"interfaces", "interface", "config", "name"},
	}, {
		name:    "relative path requiring a context entry, none supplied",
		inPath:  "../../../../fish/chips",
		wantErr: true,
	}, {
		name:   "relative path",
		inPath: "../../aardvark/anteater",
		inContext: &yang.Entry{
			Name: "cage",
			Parent: &yang.Entry{
				Name: "row",
				Parent: &yang.Entry{
					Name: "zoo",
					Parent: &yang.Entry{
						Name: "module",
					},
				},
			},
		},
		wantParts: []string{"zoo", "aardvark", "anteater"},
	}, {
		name:   "relative path with too many parts",
		inPath: "../../../../../../foo",
		inContext: &yang.Entry{
			Name:   "root",
			Parent: &yang.Entry{Name: "module"},
		},
		wantErr: true,
	}, {
		name:   "relative path that goes to the root",
		inPath: "../../foo",
		inContext: &yang.Entry{
			Name: "son",
			Parent: &yang.Entry{
				Name:   "parent",
				Parent: &yang.Entry{Name: "module"},
			},
		},
		wantParts: []string{"foo"},
	}, {
		name:   "relative path that goes above the root",
		inPath: "../../../foo",
		inContext: &yang.Entry{
			Name: "son",
			Parent: &yang.Entry{
				Name:   "parent",
				Parent: &yang.Entry{Name: "module"},
			},
		},
		wantErr: true,
	}}

	for _, tt := range tests {
		got, err := fixSchemaTreePath(tt.inPath, tt.inContext)
		if err != nil {
			if !tt.wantErr {
				t.Errorf("%s: fixedSchemaTreePath(%v,%v): got unexpected error: %v", tt.name, tt.inPath, tt.inContext, err)
			}
			continue
		}

		if tt.wantErr {
			t.Errorf("%s: fixedSchemaTreePath(%v, %v): did not get expected error", tt.name, tt.inPath, tt.inContext)
			continue
		}

		if !reflect.DeepEqual(got, tt.wantParts) {
			t.Errorf("%s: fixedSchemaTreePath(%v, %v): did not get expected parts, got: %v, want: %v", tt.name, tt.inPath, tt.inContext, got, tt.wantParts)
		}
	}
}

func TestSchemaTreePath(t *testing.T) {
	tests := []struct {
		name string
		in   *yang.Entry
		want string
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
		want: "/module/container/leaf",
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
		want: "/module/container/leaf",
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
		want: "/module/container/leaf",
	}}

	for _, tt := range tests {
		got := schemaTreePath(tt.in)
		if got != tt.want {
			t.Errorf("%s: schemaTreePath(%v): did not get expected path, got: %v, want: %v", tt.name, tt.in, got, tt.want)
		}
	}
}
