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

	"github.com/kylelemons/godebug/pretty"
	"github.com/openconfig/goyang/pkg/yang"
)

// TestYangHelperChecks tests a known set of input data against the helper
// functions that check the type of a particular element in yanghelpers.go.
func TestYangHelperChecks(t *testing.T) {
	tests := []struct {
		name                string
		inEntry             *yang.Entry
		wantDir             bool
		wantContainer       bool
		wantList            bool
		wantRoot            bool
		wantConfigState     bool
		wantCompressedValid bool
		wantChoiceOrCase    bool
		wantHasOnlyChild    bool
		wantLeaf            bool
		wantLeafList        bool
	}{{
		name: "valid directory node",
		inEntry: &yang.Entry{
			Name: "container",
			Kind: yang.DirectoryEntry,
			Dir: map[string]*yang.Entry{
				"child": {},
			},
			Parent: &yang.Entry{},
		},
		wantDir:             true,
		wantContainer:       true,
		wantCompressedValid: true,
		wantHasOnlyChild:    true,
	}, {
		name: "non-directory entry",
		inEntry: &yang.Entry{
			Name:   "container",
			Parent: &yang.Entry{},
		},
		wantCompressedValid: true,
		wantLeaf:            true,
	}, {
		name: "list entry",
		inEntry: &yang.Entry{
			Name:   "list-node",
			Parent: &yang.Entry{},
			Dir: map[string]*yang.Entry{
				"child": {},
			},
			ListAttr: &yang.ListAttr{},
		},
		wantDir:             true,
		wantList:            true,
		wantCompressedValid: true,
		wantHasOnlyChild:    true,
	}, {
		name: "root entry",
		inEntry: &yang.Entry{
			Name: "root-entry",
		},
		wantRoot: true,
		wantLeaf: true,
	}, {
		name: "config container",
		inEntry: &yang.Entry{
			Name:   "config",
			Parent: &yang.Entry{},
			Dir: map[string]*yang.Entry{
				"child": {},
			},
		},
		wantDir:          true,
		wantConfigState:  true,
		wantHasOnlyChild: true,
	}, {
		name: "surrounding-container",
		inEntry: &yang.Entry{
			Name:   "plural",
			Parent: &yang.Entry{},
			Dir: map[string]*yang.Entry{
				"singular": {
					Name:     "singular",
					ListAttr: &yang.ListAttr{},
					Dir:      map[string]*yang.Entry{},
				},
			},
		},
		wantDir:          true,
		wantHasOnlyChild: true,
	}, {
		name: "choice-node",
		inEntry: &yang.Entry{
			Name:   "choice",
			Kind:   yang.ChoiceEntry,
			Parent: &yang.Entry{},
		},
		wantChoiceOrCase:    true,
		wantCompressedValid: false,
	}, {
		name: "case-node",
		inEntry: &yang.Entry{
			Name:   "case",
			Kind:   yang.CaseEntry,
			Parent: &yang.Entry{},
		},
		wantChoiceOrCase:    true,
		wantCompressedValid: false,
	}, {
		name: "leaf",
		inEntry: &yang.Entry{
			Name:   "leaf",
			Kind:   yang.LeafEntry,
			Parent: &yang.Entry{},
		},
		wantLeaf:            true,
		wantCompressedValid: true,
	}, {
		name: "leaf-list",
		inEntry: &yang.Entry{
			Name:     "leaf-list",
			Kind:     yang.LeafEntry,
			ListAttr: &yang.ListAttr{},
			Parent:   &yang.Entry{},
		},
		wantLeafList:        true,
		wantCompressedValid: true,
	}}

	for _, tt := range tests {
		if tt.inEntry.IsDir() != tt.wantDir {
			t.Errorf("%s: .IsDir is not %v", tt.name, tt.wantDir)
		}
		if tt.inEntry.IsList() != tt.wantList {
			t.Errorf("%s: .IsList is not %v", tt.name, tt.wantList)
		}
		if isRoot(tt.inEntry) != tt.wantRoot {
			t.Errorf("%s: isRoot is not %v", tt.name, tt.wantRoot)
		}
		if isConfigState(tt.inEntry) != tt.wantConfigState {
			t.Errorf("%s: isConfigState is not %v", tt.name, tt.wantConfigState)
		}
		if isOCCompressedValidElement(tt.inEntry) != tt.wantCompressedValid {
			t.Errorf("%s: isCompressedValidElement is not %v", tt.name, tt.wantCompressedValid)
		}
		if isChoiceOrCase(tt.inEntry) != tt.wantChoiceOrCase {
			t.Errorf("%s: isChoiceOrCase is not %v", tt.name, tt.wantChoiceOrCase)
		}
		if hasOnlyChild(tt.inEntry) != tt.wantHasOnlyChild {
			t.Errorf("%s: hasOnlyChild is not %v", tt.name, tt.wantHasOnlyChild)
		}
		if tt.inEntry.IsLeaf() != tt.wantLeaf {
			t.Errorf("%s: .IsLeaf is not %v", tt.name, tt.wantLeaf)
		}
		if tt.inEntry.IsLeafList() != tt.wantLeafList {
			t.Errorf("%s: .IsLeafList is not %v", tt.name, tt.wantLeafList)
		}
	}
}

// TestYangChildren checks the helper functions from yanghelpers.go that extract
// the children of a particular YANG directory (container, list) node, along
// with those that extract only a particular subset of the children.
func TestYangChildren(t *testing.T) {
	tests := []struct {
		name           string
		inEntry        *yang.Entry
		wantChildNames map[string]bool
	}{{
		name: "basic test container",
		inEntry: &yang.Entry{
			Dir: map[string]*yang.Entry{
				"config": {
					Name: "config",
				},
				"state": {
					Name: "state",
				},
			},
		},
		wantChildNames: map[string]bool{
			"config":   true,
			"state":    true,
			"nonexist": false,
		},
	}, {
		name: "nested test container",
		inEntry: &yang.Entry{
			Dir: map[string]*yang.Entry{
				"config": {
					Name: "config",
					Dir: map[string]*yang.Entry{
						"config-leaf": {Name: "config-leaf"},
					},
				},
				"state": {
					Name: "state",
					Dir: map[string]*yang.Entry{
						"config-leaf": {Name: "config-leaf"},
						"state-leaf":  {Name: "state-leaf"},
					},
				},
			},
		},
		wantChildNames: map[string]bool{
			"config":      true,
			"state":       true,
			"config-leaf": false,
			"state-leaf":  false,
		},
	}, {
		name: "test container with RPC entry",
		inEntry: &yang.Entry{
			Dir: map[string]*yang.Entry{
				"rpc":    {RPC: &yang.RPCEntry{}},
				"config": {Name: "config"},
				"state":  {Name: "state"},
			},
		},
		wantChildNames: map[string]bool{
			"config": true,
			"state":  true,
			"rpc":    false,
		},
	}}

	for _, tt := range tests {
		cset := children(tt.inEntry)
		vmap := make(map[string]bool)

		for _, ch := range cset {
			vmap[ch.Name] = true
		}

		for k, v := range tt.wantChildNames {
			if _, ok := vmap[k]; ok != v {
				t.Errorf("%s: child did not have correct found status %v != %v", tt.name, ok, v)
			}
		}
	}
}

// TestYangPath tests functions related to YANG paths - these are strings of
// the form /a/b/c/d.
func TestYangPath(t *testing.T) {
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
		if got := joinPath(tt.inSplitPath); got != tt.wantStringPath {
			t.Errorf("%s: joinPath(%v) = %s, want %s", tt.name, tt.inSplitPath, got, tt.wantStringPath)
		}

		if tt.wantStrippedPath != "" {
			var s []string
			for _, p := range tt.inSplitPath {
				s = append(s, removePrefix(p))
			}

			if got := joinPath(s); got != tt.wantStrippedPath {
				t.Errorf("%s: removePrefix(%v) = %s, want %s", tt.name, tt.inSplitPath, got, tt.wantStrippedPath)
			}
		}
	}
}

// TestChoiceFinder tests the functionality associated with extracting non-choice
// or case elements from a YANG structure.
func TestChoiceFinder(t *testing.T) {
	tests := []struct {
		name     string
		inEntry  *yang.Entry
		wantKeys []string
	}{{
		name: "choice with single case",
		inEntry: &yang.Entry{
			Name: "choice-node",
			Kind: yang.ChoiceEntry,
			Dir: map[string]*yang.Entry{
				"case-one": {
					Name: "case-one",
					Kind: yang.CaseEntry,
					Dir: map[string]*yang.Entry{
						"child-one": {
							Name:   "child-one",
							Parent: &yang.Entry{Name: "parent"},
						},
						"child-two": {
							Name:   "child-two",
							Parent: &yang.Entry{Name: "parent"},
						},
					},
				},
			},
		},
		wantKeys: []string{"/parent/child-one", "/parent/child-two"},
	}, {
		name: "choice with multiple cases",
		inEntry: &yang.Entry{
			Name: "choice-node",
			Kind: yang.ChoiceEntry,
			Dir: map[string]*yang.Entry{
				"case-one": {
					Name: "case-one",
					Kind: yang.CaseEntry,
					Dir: map[string]*yang.Entry{
						"child-one": {
							Name:   "child-one",
							Parent: &yang.Entry{Name: "parent"},
						},
					},
				},
				"case-two": {
					Name: "case-two",
					Kind: yang.CaseEntry,
					Dir: map[string]*yang.Entry{
						"child-two": {
							Name:   "child-two",
							Parent: &yang.Entry{Name: "parent"},
						},
					},
				},
			},
		},
		wantKeys: []string{"/parent/child-one", "/parent/child-two"},
	}}

	for _, tt := range tests {
		exp := make(map[string]bool)
		for _, k := range tt.wantKeys {
			exp[k] = false
		}

		found := make(map[string]*yang.Entry)
		findFirstNonChoice(tt.inEntry, found)

		// Check whether the expected paths were found.
		for k := range found {
			if _, ok := found[k]; ok {
				exp[k] = true
			} else {
				t.Errorf("%s: could not find expected node %s", tt.name, k)
			}
		}

		// Check that all expected paths were found.
		for k, v := range exp {
			if v == false {
				t.Errorf("%s: did not find expected node %s", tt.name, k)
			}
		}
	}
}

// TestIsConfig tests the isConfig function to ensure that the config parameter is correctly
// determined.
func TestIsConfig(t *testing.T) {
	tests := []struct {
		name       string
		in         *yang.Entry
		wantConfig bool
	}{{
		name: "simple element - config true",
		in: &yang.Entry{
			Name: "elem",
			Parent: &yang.Entry{
				Config: yang.TSTrue,
			},
			Config: yang.TSTrue,
		},
		wantConfig: true,
	}, {
		name: "simple element - config false",
		in: &yang.Entry{
			Name:   "elem",
			Config: yang.TSFalse,
			Parent: &yang.Entry{
				Config: yang.TSTrue,
			},
		},
		wantConfig: false,
	}, {
		name: "parent determines config",
		in: &yang.Entry{
			Name: "elem",
			Parent: &yang.Entry{
				Config: yang.TSFalse,
			},
		},
		wantConfig: false,
	}, {
		name: "root determines config",
		in: &yang.Entry{
			Name: "elem",
			Parent: &yang.Entry{
				Parent: &yang.Entry{Name: "parent"},
			},
		},
		wantConfig: true,
	}}

	for _, tt := range tests {
		if isConfig(tt.in) != tt.wantConfig {
			t.Errorf("%s: did not have expected config value", tt.name)
		}
	}
}

// TestSafeGoEnumeratedValueName tests the safeGoEnumeratedValue function to ensure
// that enumeraton value names are correctly transformed to safe Go names.
func TestSafeGoEnumeratedValueName(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"SPEED_2.5G", "SPEED_2_5G"},
		{"IPV4-UNICAST", "IPV4_UNICAST"},
		{"frameRelay", "frameRelay"},
		{"coffee", "coffee"},
		{"ethernetCsmacd", "ethernetCsmacd"},
		{"SFP+", "SFP_PLUS"},
		{"LEVEL1/2", "LEVEL1_2"},
		{"DAYS1-3", "DAYS1_3"},
		{"FISH CHIPS", "FISH_CHIPS"},
		{"FOO*", "FOO_ASTERISK"},
	}

	for _, tt := range tests {
		got := safeGoEnumeratedValueName(tt.in)
		if got != tt.want {
			t.Errorf("safeGoEnumeratedValueName(%s): got: %s, want: %s", tt.in, got, tt.want)
		}
	}
}

func TestResolveLeafrefTargetType(t *testing.T) {
	tests := []struct {
		name           string
		inPath         string
		inContextEntry *yang.Entry
		inEntries      []*yang.Entry
		want           *yang.Entry
		wantErr        bool
	}{{
		name:   "simple test with leafref with absolute leafref",
		inPath: "/parent/child/a",
		inContextEntry: &yang.Entry{
			Name: "b",
			Type: &yang.YangType{
				Kind: yang.Yleafref,
				Path: "/parent/child/a",
			},
			Parent: &yang.Entry{
				Name: "child",
				Parent: &yang.Entry{
					Name:   "parent",
					Parent: &yang.Entry{Name: "module"},
				},
			},
		},
		inEntries: []*yang.Entry{
			{
				Name: "parent",
				Dir: map[string]*yang.Entry{
					"child": {
						Name: "child",
						Dir: map[string]*yang.Entry{
							"a": {
								Name: "a",
								Type: &yang.YangType{
									Kind: yang.Ystring,
								},
								Parent: &yang.Entry{
									Name: "child",
									Parent: &yang.Entry{
										Name:   "parent",
										Parent: &yang.Entry{Name: "module"},
									},
								},
							},
							"b": {
								Name: "b",
								Type: &yang.YangType{
									Kind: yang.Yleafref,
									Path: "/parent/child/a",
								},
								Parent: &yang.Entry{
									Name: "child",
									Parent: &yang.Entry{
										Name:   "parent",
										Parent: &yang.Entry{Name: "module"},
									},
								},
							},
						},
						Parent: &yang.Entry{
							Name:   "parent",
							Parent: &yang.Entry{Name: "module"},
						},
					},
				},
				Parent: &yang.Entry{Name: "module"},
			},
		},
		want: &yang.Entry{
			Name: "a",
			Type: &yang.YangType{
				Kind: yang.Ystring,
			},
			Parent: &yang.Entry{
				Name: "child",
				Parent: &yang.Entry{
					Name:   "parent",
					Parent: &yang.Entry{Name: "module"},
				},
			},
		},
	}}

	for _, tt := range tests {
		// Since we are outside of the build of a module, need to initialise
		// the schematree.
		st, err := buildSchemaTree(tt.inEntries)
		if err != nil {
			t.Errorf("%s: buildSchemaTree(%v): got unexpected error: %v", tt.name, tt.inEntries, err)
		}
		s := newGenState()
		s.schematree = st
		got, err := s.resolveLeafrefTarget(tt.inPath, tt.inContextEntry)
		if err != nil {
			if !tt.wantErr {
				t.Errorf("%s: resolveLeafrefTargetPath(%v, %v): got unexpected error: %v", tt.name, tt.inPath, tt.inContextEntry, err)
			}
			continue
		}

		if tt.wantErr {
			t.Errorf("%s: resolveLeafrefTargetPath(%v, %v): did not get expected error", tt.name, tt.inPath, tt.inContextEntry)
			continue
		}

		if diff := pretty.Compare(got, tt.want); diff != "" {
			t.Errorf("%s: resolveLeafrefTargetPath(%v, %v): did not get expected entry, diff(-got,+want):\n%s", tt.name, tt.inPath, tt.inContextEntry, diff)
		}
	}
}

func TestDirectEntryChild(t *testing.T) {
	tests := []struct {
		name            string
		inParent        *yang.Entry
		inChild         *yang.Entry
		inCompressPaths bool
		want            bool
	}{{
		name: "simple entry with no path compression",
		inParent: &yang.Entry{
			Name: "parent",
			Parent: &yang.Entry{
				Name: "module",
			},
		},
		inChild: &yang.Entry{
			Name: "child",
			Parent: &yang.Entry{
				Name: "parent",
				Parent: &yang.Entry{
					Name: "module",
				},
			},
		},
		want: true,
	}, {
		name: "non-child entry with path compression",
		inParent: &yang.Entry{
			Name: "item-one",
			Parent: &yang.Entry{
				Name: "module",
			},
		},
		inChild: &yang.Entry{
			Name: "item-two",
			Parent: &yang.Entry{
				Name: "module",
			},
		},
	}, {
		name: "child path length less than parent length",
		inParent: &yang.Entry{
			Name: "item-one",
			Parent: &yang.Entry{
				Name: "container",
				Parent: &yang.Entry{
					Name: "module",
				},
			},
		},
		inChild: &yang.Entry{
			Name: "item-two",
			Parent: &yang.Entry{
				Name: "module",
			},
		},
	}, {
		name: "compress paths on, child path length too short",
		inParent: &yang.Entry{
			Name: "item-one",
			Parent: &yang.Entry{
				Name: "container",
				Parent: &yang.Entry{
					Name: "module",
				},
			},
		},
		inChild: &yang.Entry{
			Name:     "item-two",
			Dir:      map[string]*yang.Entry{},
			ListAttr: &yang.ListAttr{},
			Parent: &yang.Entry{
				Name: "module",
			},
		},
		inCompressPaths: true,
	}, {
		name: "compress paths on, child path length is too long",
		inParent: &yang.Entry{
			Name: "item-one",
			Parent: &yang.Entry{
				Name: "module",
			},
		},
		inChild: &yang.Entry{
			Name:     "item-three",
			Dir:      map[string]*yang.Entry{},
			ListAttr: &yang.ListAttr{},
			Parent: &yang.Entry{
				Name: "item-two",
				Parent: &yang.Entry{
					Name: "item-one",
					Parent: &yang.Entry{
						Name: "module",
					},
				},
			},
		},
		inCompressPaths: true,
	}, {
		name: "compress paths on, child is not a list",
		inParent: &yang.Entry{
			Name: "parent",
			Kind: yang.DirectoryEntry,
			Dir:  map[string]*yang.Entry{},
			Parent: &yang.Entry{
				Name: "module",
			},
		},
		inChild: &yang.Entry{
			Name: "child",
			Kind: yang.DirectoryEntry,
			Dir:  map[string]*yang.Entry{},
			Parent: &yang.Entry{
				Name: "parent",
				Parent: &yang.Entry{
					Name: "module",
				},
			},
		},
		inCompressPaths: true,
		want:            true,
	}, {
		name: "compress paths on, parent does not have an only child",
		inParent: &yang.Entry{
			Name: "parent",
			Kind: yang.DirectoryEntry,
			Dir: map[string]*yang.Entry{
				"surrounding-container": {Name: "child"},
				"childtwo":              {Name: "childtwo"},
			},
			Parent: &yang.Entry{
				Name: "module",
			},
		},
		inChild: &yang.Entry{
			Name:     "child",
			Kind:     yang.DirectoryEntry,
			Dir:      map[string]*yang.Entry{},
			ListAttr: &yang.ListAttr{},
			Parent: &yang.Entry{
				Name: "surrounding-container",
				Parent: &yang.Entry{
					Name: "parent",
					Parent: &yang.Entry{
						Name: "module",
					},
				},
			},
		},
		inCompressPaths: true,
	}}

	for _, tt := range tests {
		if got := isDirectEntryChild(tt.inParent, tt.inChild, tt.inCompressPaths); got != tt.want {
			t.Errorf("%s: isDirectEntryChild(%v, %v, %v): did determine child status correctly, got: %v, want: %v", tt.name, tt.inParent, tt.inChild, tt.inCompressPaths, got, tt.want)
		}
	}
}
