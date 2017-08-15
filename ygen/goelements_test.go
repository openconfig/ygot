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
	"testing"

	"github.com/openconfig/goyang/pkg/yang"
)

// TestFindChildren tests the findAllChildren function to ensure that the
// child nodes that are extracted from a YANG schema instance correctly. The
// test is run with the schema compression flag on and off - such that both
// a simplified and unsimplified schema can be tested.
func TestFindChildren(t *testing.T) {
	tests := []struct {
		name             string
		inElement        *yang.Entry
		wantCompressed   []yang.Entry
		wantUncompressed []yang.Entry
		// wantErr is a map keyed by the CompressOCPaths value of whether errors
		// are expected. i.e., wantErr[true] = false means that an error is not
		// expected when the test is run with CompressOCPaths == true.
		wantErr map[bool]bool
	}{{
		name: "interface",
		inElement: &yang.Entry{
			Name:     "interface",
			ListAttr: &yang.ListAttr{},
			Dir: map[string]*yang.Entry{
				"config": {
					Name:   "config",
					Type:   &yang.YangType{},
					Config: yang.TSTrue,
					Dir: map[string]*yang.Entry{
						"type": {
							Name:   "type",
							Config: yang.TSTrue,
							Type: &yang.YangType{
								Kind: yang.Ystring,
							},
						},
						"name": {
							Name:   "name",
							Config: yang.TSTrue,
							Type: &yang.YangType{
								Kind: yang.Ystring,
							},
						},
					},
				},
				"state": {
					Name:   "state",
					Type:   &yang.YangType{},
					Config: yang.TSFalse,
					Dir: map[string]*yang.Entry{
						"type": {
							Name:   "type",
							Config: yang.TSFalse,
							Type: &yang.YangType{
								Kind: yang.Ystring,
							},
						},
						"name": {
							Name:   "name",
							Config: yang.TSFalse,
							Type:   &yang.YangType{Kind: yang.Ystring},
						},
						"admin-status": {
							Name:   "admin-status",
							Config: yang.TSFalse,
							Type:   &yang.YangType{Kind: yang.Ystring},
						},
					},
				},
				"name": {
					Name:   "name",
					Config: yang.TSTrue,
					Type:   &yang.YangType{Kind: yang.Yleafref},
				},
			},
		},
		wantCompressed: []yang.Entry{
			{
				Name:   "name",
				Config: yang.TSTrue,
				Type: &yang.YangType{
					Kind: yang.Ystring,
				},
			},
			{
				Name:   "type",
				Config: yang.TSTrue,
				Type:   &yang.YangType{Kind: yang.Ystring},
			},
			{
				Name:   "admin-status",
				Config: yang.TSFalse,
				Type: &yang.YangType{
					Kind: yang.Ystring,
				},
			},
		},
		wantUncompressed: []yang.Entry{
			{
				Name:   "config",
				Config: yang.TSTrue,
				Type:   &yang.YangType{},
			},
			{
				Name:   "state",
				Config: yang.TSFalse,
				Type:   &yang.YangType{},
			},
			{
				Name:   "name",
				Config: yang.TSTrue,
				Type:   &yang.YangType{Kind: yang.Yleafref},
			},
		},
	}, {
		name: "surrounding-container",
		inElement: &yang.Entry{
			Name:   "root",
			Config: yang.TSTrue,
			Type:   &yang.YangType{},
			Dir: map[string]*yang.Entry{
				"plural": {
					Name:   "plural",
					Config: yang.TSTrue,
					Type:   &yang.YangType{},
					Dir: map[string]*yang.Entry{
						"singular": {
							Name:     "singular",
							Config:   yang.TSTrue,
							Dir:      map[string]*yang.Entry{},
							Type:     &yang.YangType{},
							ListAttr: &yang.ListAttr{},
						},
					},
				},
			},
		},
		wantCompressed: []yang.Entry{
			{
				Name:   "singular",
				Config: yang.TSTrue,
				Type:   &yang.YangType{},
			},
		},
		wantUncompressed: []yang.Entry{
			{
				Name:   "plural",
				Config: yang.TSTrue,
				Type:   &yang.YangType{},
			},
		},
	}, {
		name: "duplicate-elements",
		inElement: &yang.Entry{
			Name:   "root",
			Config: yang.TSTrue,
			Type:   &yang.YangType{},
			Dir: map[string]*yang.Entry{
				"name": {Name: "name"},
				"config": {
					Name:   "config",
					Config: yang.TSTrue,
					Type:   &yang.YangType{},
					Dir: map[string]*yang.Entry{
						"name": {Name: "name"},
					},
				},
			},
		},
		wantErr: map[bool]bool{true: true},
	}, {
		name: "choice entry",
		inElement: &yang.Entry{
			Name: "choice-node",
			Kind: yang.ChoiceEntry,
			Dir: map[string]*yang.Entry{
				"case-one": {
					Name: "case-one",
					Kind: yang.CaseEntry,
					Dir: map[string]*yang.Entry{
						"option": {
							Name: "option",
							Type: &yang.YangType{},
						},
					},
				},
			},
		},
		wantCompressed: []yang.Entry{
			{
				Name: "option",
				Type: &yang.YangType{},
			},
		},
		wantUncompressed: []yang.Entry{
			{
				Name: "option",
				Type: &yang.YangType{},
			},
		},
	}}

	for _, tt := range tests {
		for compress, expected := range map[bool][]yang.Entry{true: tt.wantCompressed, false: tt.wantUncompressed} {
			elems, errs := findAllChildren(tt.inElement, compress)
			if tt.wantErr == nil && len(errs) > 0 {
				t.Errorf("%s (compress: %v): errors %v for children of %s", tt.name, compress, errs, tt.inElement.Name)
			} else {
				if expErr, ok := tt.wantErr[compress]; ok {
					if (len(errs) > 0) != expErr {
						t.Errorf("%s (compress: %v): did not get expected error", tt.name, compress)
					}
				}
			}

			retMap := make(map[string]*yang.Entry)
			for _, elem := range elems {
				retMap[elem.Name] = elem
			}

			for _, expectEntry := range expected {
				if elem, ok := retMap[expectEntry.Name]; ok {
					if elem.Config != expectEntry.Config {
						t.Errorf("%s (compress: %v): element %s had wrong config status %s", tt.name, compress,
							expectEntry.Name, elem.Config)
					}
					if elem.Type.Kind != expectEntry.Type.Kind {
						t.Errorf("%s (compress: %v): element %s had wrong type %s", tt.name,
							compress, expectEntry.Name, elem.Type.Kind)
					}
				} else {
					t.Errorf("%s (compress: %v): could not find expected child %s in %s", tt.name, compress,
						expectEntry.Name, tt.inElement.Name)
				}
			}
		}
	}
}

// TestCamelCase tests the functionality that is provided by makeNameUnique and
// entryCamelCaseName- ensuring
// that following being converted to CamelCase, a name is unique within the set of
// entities that have been generated already by the YANGCodeGenerator implementation.
func TestCamelCase(t *testing.T) {
	tests := []struct {
		name        string        // name is the test name.
		inPrevNames []*yang.Entry // inPrevNames is a set of names that have already been processed.
		inEntry     *yang.Entry   // inName is the name that we are testing.
		wantName    string        // wantName is the name that we expect for inName post conversion.
	}{{
		name:     "basic CamelCase test",
		inEntry:  &yang.Entry{Name: "leaf-one"},
		wantName: "LeafOne",
	}, {
		name:     "single word",
		inEntry:  &yang.Entry{Name: "leaf"},
		wantName: "Leaf",
	}, {
		name:     "already camelcase",
		inEntry:  &yang.Entry{Name: "AlreadyCamelCase"},
		wantName: "AlreadyCamelCase",
	}, {
		name:        "already defined",
		inPrevNames: []*yang.Entry{{Name: "interfaces"}},
		inEntry:     &yang.Entry{Name: "interfaces"},
		wantName:    "Interfaces_",
	}, {
		name:        "already defined twice",
		inPrevNames: []*yang.Entry{{Name: "interfaces"}, {Name: "interfaces"}},
		inEntry:     &yang.Entry{Name: "Interfaces"},
		wantName:    "Interfaces__",
	}, {
		name: "camelcase extension",
		inEntry: &yang.Entry{
			Name: "foobar",
			Exts: []*yang.Statement{{
				Keyword:     "some-module:camelcase-name",
				HasArgument: true,
				Argument:    "FooBar",
			}},
		},
		wantName: "FooBar",
	}, {
		name:        "camelcase extension with clashing name",
		inPrevNames: []*yang.Entry{{Name: "FishChips"}},
		inEntry: &yang.Entry{
			Name: "fish-chips",
			Exts: []*yang.Statement{{
				Keyword:     "anothermodule:camelcase-name",
				HasArgument: true,
				Argument:    `"FishChips\n"`,
			}},
		},
		wantName: "FishChips_",
	}}

	for _, tt := range tests {
		ctx := make(map[string]bool)
		for _, prevName := range tt.inPrevNames {
			_ = makeNameUnique(entryCamelCaseName(prevName), ctx)
		}

		if got := makeNameUnique(entryCamelCaseName(tt.inEntry), ctx); got != tt.wantName {
			t.Errorf("%s: did not get expected name for %s (after defining %v): %s",
				tt.name, tt.inEntry, tt.inPrevNames, got)
		}
	}
}

// TestUnionSubTypes extracts the types which make up a YANG union from a
// Goyang YangType struct.
func TestUnionSubTypes(t *testing.T) {
	tests := []struct {
		name       string
		in         *yang.YangType
		inCtxEntry *yang.Entry
		want       []string
		wantErr    bool
	}{{
		name: "union of strings",
		in: &yang.YangType{
			Kind: yang.Yunion,
			Type: []*yang.YangType{
				{Kind: yang.Ystring},
				{Kind: yang.Ystring},
			},
		},
		want: []string{"string"},
	}, {
		name: "union of int8, string",
		in: &yang.YangType{
			Kind: yang.Yunion,
			Type: []*yang.YangType{
				{Kind: yang.Yint8},
				{Kind: yang.Ystring},
			},
		},
		want: []string{"int8", "string"},
	}, {
		name: "union of unions",
		in: &yang.YangType{
			Kind: yang.Yunion,
			Type: []*yang.YangType{
				{
					Kind: yang.Yunion,
					Type: []*yang.YangType{
						{Kind: yang.Ystring},
						{Kind: yang.Yint32},
					},
				},
				{
					Kind: yang.Yunion,
					Type: []*yang.YangType{
						{Kind: yang.Yuint64},
						{Kind: yang.Yint16},
					},
				},
			},
		},
		want: []string{"string", "int32", "uint64", "int16"},
	}, {
		name: "erroneous union without context",
		in: &yang.YangType{
			Name: "enumeration",
			Kind: yang.Yunion,
			Type: []*yang.YangType{
				{Kind: yang.Yenum, Name: "enumeration"},
			},
		},
		wantErr: true,
	}, {
		name: "union of identityrefs",
		in: &yang.YangType{
			Kind: yang.Yunion,
			Type: []*yang.YangType{{
				Kind: yang.Yidentityref,
				IdentityBase: &yang.Identity{
					Name:   "id",
					Parent: &yang.Module{Name: "basemod"},
				},
			}, {
				Kind: yang.Yidentityref,
				IdentityBase: &yang.Identity{
					Name:   "id2",
					Parent: &yang.Module{Name: "basemod2"},
				},
			}},
		},
		inCtxEntry: &yang.Entry{
			Name: "context-leaf",
			Type: &yang.YangType{
				Kind: yang.Yunion,
				Type: []*yang.YangType{{
					Kind: yang.Yidentityref,
					IdentityBase: &yang.Identity{
						Name:   "id",
						Parent: &yang.Module{Name: "basemod"},
					},
				}, {
					Kind: yang.Yidentityref,
					IdentityBase: &yang.Identity{
						Name:   "id2",
						Parent: &yang.Module{Name: "basemod2"},
					},
				}},
			},
			Node: &yang.Leaf{
				Name:   "context-leaf",
				Parent: &yang.Module{Name: "basemod"},
			},
		},
		want: []string{"E_Basemod_Id", "E_Basemod2_Id2"},
	}}

	for _, tt := range tests {
		s := newGenState()
		ctypes := make(map[string]int)
		errs := s.findUnionSubTypes(tt.in, tt.inCtxEntry, ctypes, false)
		if !tt.wantErr && len(errs) > 0 {
			t.Errorf("%s: unexpected errors: %v", tt.name, errs)
			continue
		}

		for i, wt := range tt.want {
			if unionidx, ok := ctypes[wt]; !ok {
				t.Errorf("%s: could not find expected type %s", tt.name, wt)
				continue
			} else if i != unionidx {
				t.Errorf("%s: index of type %s was not as expected (%d != %d)", tt.name, wt, i, unionidx)
			}
		}

		for ct := range ctypes {
			found := false
			for _, gt := range tt.want {
				if ct == gt {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("%s: found unexpected type %s", tt.name, ct)
			}
		}
	}
}
