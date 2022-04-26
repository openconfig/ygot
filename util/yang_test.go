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
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kylelemons/godebug/pretty"
	"github.com/openconfig/gnmi/errdiff"
	"github.com/openconfig/goyang/pkg/yang"
)

func TestSchemaTreeRoot(t *testing.T) {
	module := &yang.Entry{
		Name: "module",
	}
	tests := []struct {
		name    string
		want    *yang.Entry
		inChild *yang.Entry
	}{{
		name: "simple chained entries",
		want: module,
		inChild: &yang.Entry{
			Name: "child",
			Parent: &yang.Entry{
				Name:   "parent",
				Parent: module,
			},
		},
	}, {
		name: "single-leveled chain",
		want: module,
		inChild: &yang.Entry{
			Name:   "child",
			Parent: module,
		},
	}, {
		name:    "no parent",
		want:    module,
		inChild: module,
	}, {
		name:    "nil schema",
		want:    nil,
		inChild: nil,
	}}

	for _, tt := range tests {
		if got := SchemaTreeRoot(tt.inChild); got != tt.want {
			t.Errorf("%s: SchemaTreeRoot(%v): didn't determine root correctly, got: %v, want: %v", tt.name, tt.inChild, got, tt.want)
		}
	}
}

func TestSanitizedPattern(t *testing.T) {
	tests := []struct {
		desc        string
		in          *yang.YangType
		want        []string
		wantIsPOSIX bool
	}{{
		desc: "both are present",
		in: &yang.YangType{
			Pattern:      []string{`abc`},
			POSIXPattern: []string{`^def$`, `^ghi$`},
		},
		want:        []string{`^def$`, `^ghi$`},
		wantIsPOSIX: true,
	}, {
		desc: "POSIXPattern only present",
		in: &yang.YangType{
			POSIXPattern: []string{``, `^def$`},
		},
		want:        []string{``, `^def$`},
		wantIsPOSIX: true,
	}, {
		desc: "Pattern only present",
		in: &yang.YangType{
			Pattern: []string{`abc`},
		},
		want:        []string{`^(abc)$`},
		wantIsPOSIX: false,
	}, {
		desc: "Pattern only present, with different sanitization behaviours",
		in: &yang.YangType{
			Pattern: []string{``, `^abc`, `^abc$`, `abc$`, `a$b^c[^d]\\\ne`},
		},
		want:        []string{``, `^abc$`, `^abc$`, `^(abc)$`, `^(a\$b\^c[^d]\\\ne)$`},
		wantIsPOSIX: false,
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			gotPatterns, gotIsPOSIX := SanitizedPattern(tt.in)
			if diff := cmp.Diff(gotPatterns, tt.want); diff != "" {
				t.Errorf("(-got, +want):\n%s", diff)
			}
			if diff := cmp.Diff(gotIsPOSIX, tt.wantIsPOSIX); diff != "" {
				t.Errorf("(-gotIsPOSIX, +wantIsPOSIX):\n%s", diff)
			}
		})
	}
}

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
		if IsRoot(tt.inEntry) != tt.wantRoot {
			t.Errorf("%s: IsRoot is not %v", tt.name, tt.wantRoot)
		}
		if IsConfigState(tt.inEntry) != tt.wantConfigState {
			t.Errorf("%s: IsConfigState is not %v", tt.name, tt.wantConfigState)
		}
		if IsOCCompressedValidElement(tt.inEntry) != tt.wantCompressedValid {
			t.Errorf("%s: IsCompressedValidElement is not %v", tt.name, tt.wantCompressedValid)
		}
		if IsChoiceOrCase(tt.inEntry) != tt.wantChoiceOrCase {
			t.Errorf("%s: IsChoiceOrCase is not %v", tt.name, tt.wantChoiceOrCase)
		}
		if HasOnlyChild(tt.inEntry) != tt.wantHasOnlyChild {
			t.Errorf("%s: HasOnlyChild is not %v", tt.name, tt.wantHasOnlyChild)
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
		cset := Children(tt.inEntry)
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
		if IsConfig(tt.in) != tt.wantConfig {
			t.Errorf("%s: did not have expected config value", tt.name)
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
			Name: "parent-nl",
			Kind: yang.DirectoryEntry,
			Dir:  map[string]*yang.Entry{},
			Parent: &yang.Entry{
				Name: "module-nl",
			},
		},
		inChild: &yang.Entry{
			Name: "child",
			Kind: yang.DirectoryEntry,
			Dir:  map[string]*yang.Entry{},
			Parent: &yang.Entry{
				Name: "parent-nl",
				Parent: &yang.Entry{
					Name: "module-nl",
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
	}, {
		name: "compress paths on, container in state container",
		inParent: &yang.Entry{
			Name: "parent",
			Kind: yang.DirectoryEntry,
			Parent: &yang.Entry{
				Name: "module",
			},
		},
		inChild: &yang.Entry{
			Name: "counters",
			Kind: yang.DirectoryEntry,
			Parent: &yang.Entry{
				Name: "state",
				Kind: yang.DirectoryEntry,
				Dir:  map[string]*yang.Entry{},
				Parent: &yang.Entry{
					Name: "parent",
					Parent: &yang.Entry{
						Name: "module",
					},
				},
			},
		},
		inCompressPaths: true,
		want:            true,
	}}

	for _, tt := range tests {
		if got := IsDirectEntryChild(tt.inParent, tt.inChild, tt.inCompressPaths); got != tt.want {
			t.Errorf("%s: IsDirectEntryChild(%v, %v, %v): did determine child status correctly, got: %v, want: %v", tt.name, tt.inParent, tt.inChild, tt.inCompressPaths, got, tt.want)
		}
	}
}

func TestIsCompressedSchema(t *testing.T) {
	tests := []struct {
		name string
		in   *yang.Entry
		want bool
	}{{
		name: "simple entry - root",
		in: &yang.Entry{
			Annotation: map[string]interface{}{
				CompressedSchemaAnnotation: true,
			},
		},
		want: true,
	}, {
		name: "simple entry - not compressed - root",
		in:   &yang.Entry{},
	}, {
		name: "child entry - compressed",
		in: &yang.Entry{
			Parent: &yang.Entry{
				Parent: &yang.Entry{
					Parent: &yang.Entry{
						Parent: &yang.Entry{},
					},
				},
			},
			Annotation: map[string]interface{}{
				CompressedSchemaAnnotation: true,
			},
		},
	}, {
		name: "child entry - not compressed",
		in: &yang.Entry{
			Parent: &yang.Entry{
				Parent: &yang.Entry{},
			},
		},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsCompressedSchema(tt.in); got != tt.want {
				t.Fatalf("incorrect result, got: %v, want: %v", got, tt.want)
			}
		})
	}
}

func TestIsYangTypes(t *testing.T) {
	tests := []struct {
		desc           string
		schema         *yang.Entry
		wantLeafRef    bool
		wantUnion      bool
		wantEnumerated bool
		wantAnydata    bool
		wantSimpleEnum bool
	}{
		{
			desc:           "nil schema",
			schema:         nil,
			wantLeafRef:    false,
			wantUnion:      false,
			wantEnumerated: false,
			wantAnydata:    false,
			wantSimpleEnum: false,
		},
		{
			desc:           "nil Type",
			schema:         &yang.Entry{},
			wantLeafRef:    false,
			wantUnion:      false,
			wantEnumerated: false,
			wantAnydata:    false,
			wantSimpleEnum: false,
		},
		{
			desc: "int32 type",
			schema: &yang.Entry{
				Kind: yang.LeafEntry,
				Type: &yang.YangType{
					Kind: yang.Yint32,
				},
			},
			wantLeafRef:    false,
			wantUnion:      false,
			wantEnumerated: false,
			wantAnydata:    false,
			wantSimpleEnum: false,
		},
		{
			desc: "leafref type",
			schema: &yang.Entry{
				Kind: yang.LeafEntry,
				Type: &yang.YangType{
					Kind: yang.Yleafref,
				},
			},
			wantLeafRef:    true,
			wantUnion:      false,
			wantEnumerated: false,
			wantAnydata:    false,
			wantSimpleEnum: false,
		},
		{
			desc: "union type",
			schema: &yang.Entry{
				Kind: yang.LeafEntry,
				Type: &yang.YangType{
					Kind: yang.Yunion,
					Type: []*yang.YangType{{
						Name:         "string",
						Pattern:      []string{"a.*"},
						POSIXPattern: []string{"^a.*$"},
						Kind:         yang.Ystring,
						Length: yang.YangRange{{
							Min: yang.FromInt(10),
							Max: yang.FromInt(20),
						},
						},
					}, {
						Name:         "string",
						Pattern:      []string{"b.*"},
						POSIXPattern: []string{"^b.*$"},
						Kind:         yang.Ystring,
						Length: yang.YangRange{{
							Min: yang.FromInt(10),
							Max: yang.FromInt(20),
						},
						},
					},
					},
				}},
			wantLeafRef:    false,
			wantUnion:      true,
			wantEnumerated: false,
			wantAnydata:    false,
			wantSimpleEnum: false,
		},
		{
			desc: "union type with one type entry",
			schema: &yang.Entry{
				Kind: yang.LeafEntry,
				Type: &yang.YangType{
					Kind: yang.Yunion,
					Type: []*yang.YangType{{
						Name:         "string",
						Pattern:      []string{"a.*"},
						POSIXPattern: []string{"^a.*$"},
						Kind:         yang.Ystring,
						Length: yang.YangRange{{
							Min: yang.FromInt(10),
							Max: yang.FromInt(20),
						},
						},
					},
					},
				}},
			wantLeafRef:    false,
			wantUnion:      true,
			wantEnumerated: false,
			wantAnydata:    false,
			wantSimpleEnum: false,
		},
		{
			desc: "union type with no type entries (invalid)",
			schema: &yang.Entry{
				Kind: yang.LeafEntry,
				Type: &yang.YangType{
					Kind: yang.Yunion,
					Type: []*yang.YangType{},
				}},
			wantLeafRef:    false,
			wantUnion:      false,
			wantEnumerated: false,
			wantAnydata:    false,
			wantSimpleEnum: false,
		},
		{
			desc: "simple enum",
			schema: &yang.Entry{
				Kind: yang.LeafEntry,
				Type: &yang.YangType{
					Kind: yang.Yenum,
					Name: yang.TypeKindToName[yang.Yenum],
				},
			},
			wantLeafRef:    false,
			wantUnion:      false,
			wantEnumerated: true,
			wantAnydata:    false,
			wantSimpleEnum: false,
		},
		{
			desc: "identityref",
			schema: &yang.Entry{
				Kind: yang.LeafEntry,
				Type: &yang.YangType{
					Kind: yang.Yidentityref,
				},
			},
			wantLeafRef:    false,
			wantUnion:      false,
			wantEnumerated: true,
			wantAnydata:    false,
			wantSimpleEnum: false,
		},
		{
			desc: "anydata",
			schema: &yang.Entry{
				Kind: yang.AnyDataEntry,
			},
			wantLeafRef:    false,
			wantUnion:      false,
			wantEnumerated: false,
			wantAnydata:    true,
			wantSimpleEnum: false,
		},
		{
			desc: "non-simple enum",
			schema: &yang.Entry{
				Kind: yang.LeafEntry,
				Type: &yang.YangType{
					Kind: yang.Yenum,
					Name: "union-enum",
				},
			},
			wantLeafRef:    false,
			wantUnion:      false,
			wantEnumerated: true,
			wantAnydata:    false,
			wantSimpleEnum: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if got, want := IsLeafRef(tt.schema), tt.wantLeafRef; got != want {
				t.Errorf("IsLeafRef got: %v want: %v", got, want)
			}
			if got, want := IsAnydata(tt.schema), tt.wantAnydata; got != want {
				t.Errorf("IsAnydata got: %v want: %v", got, want)
			}
			if tt.schema != nil { // These functions take in type as the parameter.
				if got, want := IsUnionType(tt.schema.Type), tt.wantUnion; got != want {
					t.Errorf("IsUnionType got: %v want: %v", got, want)
				}
				if got, want := IsEnumeratedType(tt.schema.Type), tt.wantEnumerated; got != want {
					t.Errorf("IsEnumeratedType got: %v want: %v", got, want)
				}
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
				t.Errorf("got: %v want: %v", got, want)
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
				t.Errorf("got: %v want: %v", got, want)
			}
		})
	}
}

func TestIsOrNotKeyedList(t *testing.T) {
	tests := []struct {
		desc            string
		schema          *yang.Entry
		wantKeyedList   bool
		wantUnkeyedList bool
	}{
		{
			desc:            "nil schema",
			schema:          nil,
			wantKeyedList:   false,
			wantUnkeyedList: false,
		},
		{
			desc: "leaf type",
			schema: &yang.Entry{
				Kind: yang.LeafEntry,
				Type: &yang.YangType{
					Kind: yang.Yint32,
				},
			},
			wantKeyedList:   false,
			wantUnkeyedList: false,
		},
		{
			desc: "keyed list",
			schema: &yang.Entry{
				Kind:     yang.DirectoryEntry,
				ListAttr: yang.NewDefaultListAttr(),
				Key:      "key",
				Dir:      map[string]*yang.Entry{},
			},
			wantKeyedList:   true,
			wantUnkeyedList: false,
		},
		{
			desc: "unkeyed list",
			schema: &yang.Entry{
				Kind:     yang.DirectoryEntry,
				ListAttr: yang.NewDefaultListAttr(),
				Dir:      map[string]*yang.Entry{},
			},
			wantKeyedList:   false,
			wantUnkeyedList: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if got, want := IsKeyedList(tt.schema), tt.wantKeyedList; got != want {
				t.Errorf("got: %v wantKeyedList: %v", got, want)
			}
			if got, want := IsUnkeyedList(tt.schema), tt.wantUnkeyedList; got != want {
				t.Errorf("got: %v wantUnkeyedList: %v", got, want)
			}
		})
	}
}

func TestIsYgotAnnotation(t *testing.T) {
	type testStruct struct {
		Yes *string `ygotAnnotation:"true"`
		No  *string
	}

	tests := []struct {
		name string
		in   reflect.StructField
		want bool
	}{{
		name: "annotated field",
		in:   reflect.TypeOf(testStruct{}).Field(0),
		want: true,
	}, {
		name: "standard field",
		in:   reflect.TypeOf(testStruct{}).Field(1),
		want: false,
	}}

	for _, tt := range tests {
		if got := IsYgotAnnotation(tt.in); got != tt.want {
			t.Errorf("%s: IsYgotAnnotation(%#v): did not get expected result, got: %v, want: %v", tt.name, tt.in, got, tt.want)
		}
	}
}

func TestIsYangPresence(t *testing.T) {
	type testStruct struct {
		Yes *string `yangPresence:"true"`
		No  *string
	}
	structFieldYes, ok := reflect.TypeOf(testStruct{}).FieldByName("Yes")
	if !ok {
		t.Fatalf("Cannot find field Yes in testStruct")
	}

	structFieldNo, ok := reflect.TypeOf(testStruct{}).FieldByName("No")
	if !ok {
		t.Fatalf("Cannot find field No in testStruct")
	}
	tests := []struct {
		name string
		in   reflect.StructField
		want bool
	}{{
		name: "yangPresence container/field",
		in:   structFieldYes,
		want: true,
	}, {
		name: "standard field",
		in:   structFieldNo,
		want: false,
	}}

	for _, tt := range tests {
		if got := IsYangPresence(tt.in); got != tt.want {
			t.Errorf("%s: IsYangPresence(%#v): did not get expected result, got: %v, want: %v", tt.name, tt.in, got, tt.want)
		}
	}
}

// complexUnionTypeName is the name used to refer to the name of the union
// type containing the slice of input types to the functions.
const complexUnionTypeName = "complexUnionTypeName"

var complexUnionType *yang.YangType = &yang.YangType{
	Name: complexUnionTypeName,
	Kind: yang.Yunion,
	Type: []*yang.YangType{{
		Name:         "string",
		Pattern:      []string{"a.*"},
		POSIXPattern: []string{"^a.*$"},
		Kind:         yang.Ystring,
		Length: yang.YangRange{{
			Min: yang.FromInt(10),
			Max: yang.FromInt(20),
		}},
	}, {
		Name: "iref",
		Kind: yang.Yidentityref,
	}, {
		Name: "enumeration",
		Kind: yang.Yenum,
	}, {
		Name: "derived-enum",
		Kind: yang.Yenum,
	}, {
		Name: "union",
		Kind: yang.Yunion,
		Type: []*yang.YangType{{
			Name:         "string",
			Pattern:      []string{"b.*"},
			POSIXPattern: []string{"^b.*$"},
			Kind:         yang.Ystring,
			Length: yang.YangRange{{
				Min: yang.FromInt(10),
				Max: yang.FromInt(20),
			}},
		}, {
			Name: "inner-iref",
			Kind: yang.Yidentityref,
		}, {
			Name: "enumeration",
			Kind: yang.Yenum,
		}, {
			Name: "inner-derived-enum",
			Kind: yang.Yenum,
		}, {
			Name: "inner-typedef-union",
			Kind: yang.Yunion,
			Type: []*yang.YangType{{
				Name:         "string",
				Pattern:      []string{"c.*"},
				POSIXPattern: []string{"^c.*$"},
				Kind:         yang.Ystring,
				Length: yang.YangRange{{
					Min: yang.FromInt(10),
					Max: yang.FromInt(20),
				}},
			}, {
				Name: "inner-inner-iref",
				Kind: yang.Yidentityref,
			}, {
				Name: "enumeration",
				Kind: yang.Yenum,
			}, {
				Name: "inner-inner-derived-enum",
				Kind: yang.Yenum,
			}},
		}},
	}, {
		Name: "typedef-union",
		Kind: yang.Yunion,
		Type: []*yang.YangType{{
			Name:         "string",
			Pattern:      []string{"d.*"},
			POSIXPattern: []string{"^d.*$"},
			Kind:         yang.Ystring,
			Length: yang.YangRange{{
				Min: yang.FromInt(10),
				Max: yang.FromInt(20),
			}},
		}, {
			Name: "typedef-inner-int",
			Kind: yang.Yint32,
		}, {
			Name: "typedef-inner-iref",
			Kind: yang.Yidentityref,
		}, {
			Name: "enumeration",
			Kind: yang.Yenum,
		}, {
			Name: "nested-typedef-union",
			Kind: yang.Yunion,
			Type: []*yang.YangType{{
				Name:         "string",
				Pattern:      []string{"e.*"},
				POSIXPattern: []string{"^e.*$"},
				Kind:         yang.Ystring,
				Length: yang.YangRange{{
					Min: yang.FromInt(10),
					Max: yang.FromInt(20),
				}},
			}, {
				Name: "nested-inner-iref",
				Kind: yang.Yidentityref,
			}, {
				Name: "enumeration",
				Kind: yang.Yenum,
			}, {
				Name: "nested-derived-enum",
				Kind: yang.Yenum,
			}},
		}},
	}, {
		Name: "typedef-union2",
		Kind: yang.Yunion,
		Type: []*yang.YangType{{
			Name:         "string",
			Pattern:      []string{"f.*"},
			POSIXPattern: []string{"^f.*$"},
			Kind:         yang.Ystring,
			Length: yang.YangRange{{
				Min: yang.FromInt(10),
				Max: yang.FromInt(20),
			}},
		}, {
			Name: "typedef-inner-int2",
			Kind: yang.Yint32,
		}, {
			Name: "typedef-inner-iref2",
			Kind: yang.Yidentityref,
		}, {
			Name: "enumeration",
			Kind: yang.Yenum,
		}, {
			Name: "nested-typedef-union2",
			Kind: yang.Yunion,
			Type: []*yang.YangType{{
				Name:         "string",
				Pattern:      []string{"g.*"},
				POSIXPattern: []string{"^g.*$"},
				Kind:         yang.Ystring,
				Length: yang.YangRange{{
					Min: yang.FromInt(10),
					Max: yang.FromInt(20),
				}},
			}, {
				Name: "nested-typedef-int",
				Kind: yang.Yint32,
			}, {
				Name: "nested-inner-iref2",
				Kind: yang.Yidentityref,
			}, {
				Name: "enumeration",
				Kind: yang.Yenum,
			}, {
				Name: "nested-derived-enum2",
				Kind: yang.Yenum,
			}},
		}},
	}},
}

func typeNamesList(types []*yang.YangType) []string {
	var list []string
	for _, t := range types {
		list = append(list, t.Name)
	}
	return list
}

func TestFlattenedTypes(t *testing.T) {
	tests := []struct {
		desc    string
		inTypes []*yang.YangType
		want    []*yang.YangType
	}{{
		desc: "empty union type",
	}, {
		desc:    "complex union type",
		inTypes: complexUnionType.Type,
		want: []*yang.YangType{{
			Name: "string",
			Kind: yang.Ystring,
		}, {
			Name: "iref",
			Kind: yang.Yidentityref,
		}, {
			Name: "enumeration",
			Kind: yang.Yenum,
		}, {
			Name: "derived-enum",
			Kind: yang.Yenum,
		}, {
			Name: "string",
			Kind: yang.Ystring,
		}, {
			Name: "inner-iref",
			Kind: yang.Yidentityref,
		}, {
			Name: "enumeration",
			Kind: yang.Yenum,
		}, {
			Name: "inner-derived-enum",
			Kind: yang.Yenum,
		}, {
			Name: "string",
			Kind: yang.Ystring,
		}, {
			Name: "inner-inner-iref",
			Kind: yang.Yidentityref,
		}, {
			Name: "enumeration",
			Kind: yang.Yenum,
		}, {
			Name: "inner-inner-derived-enum",
			Kind: yang.Yenum,
		}, {
			Name: "string",
			Kind: yang.Ystring,
		}, {
			Name: "typedef-inner-int",
			Kind: yang.Yint32,
		}, {
			Name: "typedef-inner-iref",
			Kind: yang.Yidentityref,
		}, {
			Name: "enumeration",
			Kind: yang.Yenum,
		}, {
			Name: "string",
			Kind: yang.Ystring,
		}, {
			Name: "nested-inner-iref",
			Kind: yang.Yidentityref,
		}, {
			Name: "enumeration",
			Kind: yang.Yenum,
		}, {
			Name: "nested-derived-enum",
			Kind: yang.Yenum,
		}, {
			Name: "string",
			Kind: yang.Ystring,
		}, {
			Name: "typedef-inner-int2",
			Kind: yang.Yint32,
		}, {
			Name: "typedef-inner-iref2",
			Kind: yang.Yidentityref,
		}, {
			Name: "enumeration",
			Kind: yang.Yenum,
		}, {
			Name: "string",
			Kind: yang.Ystring,
		}, {
			Name: "nested-typedef-int",
			Kind: yang.Yint32,
		}, {
			Name: "nested-inner-iref2",
			Kind: yang.Yidentityref,
		}, {
			Name: "enumeration",
			Kind: yang.Yenum,
		}, {
			Name: "nested-derived-enum2",
			Kind: yang.Yenum,
		}},
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if diff := cmp.Diff(typeNamesList(FlattenedTypes(tt.inTypes)), typeNamesList(tt.want)); diff != "" {
				t.Errorf("FlattenedTypes (-got,+want):\n%s", diff)
			}
		})
	}
}

func TestEnumeratedUnionTypes(t *testing.T) {
	tests := []struct {
		desc    string
		inTypes []*yang.YangType
		want    []*yang.YangType
	}{{
		desc: "single-level with no enumerated types",
		inTypes: []*yang.YangType{{
			Name:         "string",
			Pattern:      []string{"a.*"},
			POSIXPattern: []string{"^a.*$"},
			Kind:         yang.Ystring,
			Length: yang.YangRange{{
				Min: yang.FromInt(10),
				Max: yang.FromInt(20),
			}},
		}, {
			Name: "int",
			Kind: yang.Yint32,
		}},
		want: []*yang.YangType{},
	}, {
		desc: "single-level with mixed types",
		inTypes: []*yang.YangType{{
			Name:         "string",
			Pattern:      []string{"a.*"},
			POSIXPattern: []string{"^a.*$"},
			Kind:         yang.Ystring,
			Length: yang.YangRange{{
				Min: yang.FromInt(10),
				Max: yang.FromInt(20),
			}},
		}, {
			Name: "int",
			Kind: yang.Yint32,
		}, {
			Name: "identityref",
			Kind: yang.Yidentityref,
		}, {
			Name: "union-enum",
			Kind: yang.Yenum,
		}},
		want: []*yang.YangType{{
			Name: "identityref",
			Kind: yang.Yidentityref,
		}, {
			Name: "union-enum",
			Kind: yang.Yenum,
		}},
	}, {
		desc:    "empty",
		inTypes: []*yang.YangType{},
		want:    []*yang.YangType{},
	}, {
		desc: "multi-level with mixed types",
		inTypes: []*yang.YangType{{
			Name:         "string",
			Pattern:      []string{"a.*"},
			POSIXPattern: []string{"^a.*$"},
			Kind:         yang.Ystring,
			Length: yang.YangRange{{
				Min: yang.FromInt(10),
				Max: yang.FromInt(20),
			}},
		}, {
			Name: "int",
			Kind: yang.Yint32,
		}, {
			Name: "identityref",
			Kind: yang.Yidentityref,
		}, {
			Name: "union",
			Kind: yang.Yunion,
			Type: []*yang.YangType{{
				Name:         "string",
				Pattern:      []string{"a.*"},
				POSIXPattern: []string{"^a.*$"},
				Kind:         yang.Ystring,
				Length: yang.YangRange{{
					Min: yang.FromInt(10),
					Max: yang.FromInt(20),
				}},
			}, {
				Name: "inner-int",
				Kind: yang.Yint32,
			}, {
				Name: "inner-iref",
				Kind: yang.Yidentityref,
			}, {
				Name: "enumeration",
				Kind: yang.Yenum,
			}},
		}},
		want: []*yang.YangType{{
			Name: "identityref",
			Kind: yang.Yidentityref,
		}, {
			Name: "inner-iref",
			Kind: yang.Yidentityref,
		}, {
			Name: "enumeration",
			Kind: yang.Yenum,
		}},
	}, {
		desc:    "multi-level with typedef union",
		inTypes: complexUnionType.Type,
		want: []*yang.YangType{{
			Name: "iref",
			Kind: yang.Yidentityref,
		}, {
			Name: "enumeration",
			Kind: yang.Yenum,
		}, {
			Name: "derived-enum",
			Kind: yang.Yenum,
		}, {
			Name: "inner-iref",
			Kind: yang.Yidentityref,
		}, {
			Name: "enumeration",
			Kind: yang.Yenum,
		}, {
			Name: "inner-derived-enum",
			Kind: yang.Yenum,
		}, {
			Name: "inner-inner-iref",
			Kind: yang.Yidentityref,
		}, {
			Name: "enumeration",
			Kind: yang.Yenum,
		}, {
			Name: "inner-inner-derived-enum",
			Kind: yang.Yenum,
		}, {
			Name: "typedef-inner-iref",
			Kind: yang.Yidentityref,
		}, {
			Name: "enumeration",
			Kind: yang.Yenum,
		}, {
			Name: "nested-inner-iref",
			Kind: yang.Yidentityref,
		}, {
			Name: "enumeration",
			Kind: yang.Yenum,
		}, {
			Name: "nested-derived-enum",
			Kind: yang.Yenum,
		}, {
			Name: "typedef-inner-iref2",
			Kind: yang.Yidentityref,
		}, {
			Name: "enumeration",
			Kind: yang.Yenum,
		}, {
			Name: "nested-inner-iref2",
			Kind: yang.Yidentityref,
		}, {
			Name: "enumeration",
			Kind: yang.Yenum,
		}, {
			Name: "nested-derived-enum2",
			Kind: yang.Yenum,
		}},
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if diff := cmp.Diff(typeNamesList(EnumeratedUnionTypes(tt.inTypes)), typeNamesList(tt.want)); diff != "" {
				t.Errorf("EnumeratedUnionTypes (-got,+want):\n%s", diff)
			}
		})
	}
}

func TestDefiningType(t *testing.T) {
	strType := &yang.YangType{
		Name:         "string",
		Pattern:      []string{"a.*"},
		POSIXPattern: []string{"^a.*$"},
		Kind:         yang.Ystring,
		Length: yang.YangRange{{
			Min: yang.FromInt(10),
			Max: yang.FromInt(20),
		}},
	}

	tests := []struct {
		desc              string
		inLeafType        *yang.YangType
		inSubtypes        []*yang.YangType
		wantDefiningTypes []*yang.YangType
		wantErrSubstr     string
	}{{
		desc:              "trivial case -- subtype is itself",
		inLeafType:        strType,
		inSubtypes:        []*yang.YangType{strType},
		wantDefiningTypes: []*yang.YangType{strType},
	}, {
		desc:          "subtype cannot be found",
		inLeafType:    complexUnionType,
		inSubtypes:    []*yang.YangType{strType},
		wantErrSubstr: "not found within provided containing type",
	}, {
		desc:       "multi-level with typedef union",
		inLeafType: complexUnionType,
		inSubtypes: FlattenedTypes(complexUnionType.Type),
		wantDefiningTypes: []*yang.YangType{{
			Name: complexUnionTypeName,
			Kind: yang.Yunion,
		}, {
			Name: "iref",
			Kind: yang.Yidentityref,
		}, {
			Name: complexUnionTypeName,
			Kind: yang.Yunion,
		}, {
			Name: "derived-enum",
			Kind: yang.Yenum,
		}, {
			Name: complexUnionTypeName,
			Kind: yang.Yunion,
		}, {
			Name: "inner-iref",
			Kind: yang.Yidentityref,
		}, {
			Name: complexUnionTypeName,
			Kind: yang.Yunion,
		}, {
			Name: "inner-derived-enum",
			Kind: yang.Yenum,
		}, {
			Name: "inner-typedef-union",
			Kind: yang.Yunion,
		}, {
			Name: "inner-inner-iref",
			Kind: yang.Yidentityref,
		}, {
			Name: "inner-typedef-union",
			Kind: yang.Yunion,
		}, {
			Name: "inner-inner-derived-enum",
			Kind: yang.Yenum,
		}, {
			Name: "typedef-union",
			Kind: yang.Yunion,
		}, {
			Name: "typedef-inner-int",
			Kind: yang.Yint32,
		}, {
			Name: "typedef-inner-iref",
			Kind: yang.Yidentityref,
		}, {
			Name: "typedef-union",
			Kind: yang.Yunion,
		}, {
			Name: "nested-typedef-union",
			Kind: yang.Yunion,
		}, {
			Name: "nested-inner-iref",
			Kind: yang.Yidentityref,
		}, {
			Name: "nested-typedef-union",
			Kind: yang.Yunion,
		}, {
			Name: "nested-derived-enum",
			Kind: yang.Yenum,
		}, {
			Name: "typedef-union2",
			Kind: yang.Yunion,
		}, {
			Name: "typedef-inner-int2",
			Kind: yang.Yint32,
		}, {
			Name: "typedef-inner-iref2",
			Kind: yang.Yidentityref,
		}, {
			Name: "typedef-union2",
			Kind: yang.Yunion,
		}, {
			Name: "nested-typedef-union2",
			Kind: yang.Yunion,
		}, {
			Name: "nested-typedef-int",
			Kind: yang.Yint32,
		}, {
			Name: "nested-inner-iref2",
			Kind: yang.Yidentityref,
		}, {
			Name: "nested-typedef-union2",
			Kind: yang.Yunion,
		}, {
			Name: "nested-derived-enum2",
			Kind: yang.Yenum,
		}},
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			var gotTypes []string
			for i, subtype := range tt.inSubtypes {
				defType, err := DefiningType(subtype, tt.inLeafType)
				if diff := errdiff.Check(err, tt.wantErrSubstr); diff != "" {
					t.Fatalf("did not get expected error:\n%s", diff)
				}
				if err != nil {
					continue
				}

				if defType == nil && tt.wantDefiningTypes[i] != nil {
					t.Errorf("subtype not found in union: %v", subtype)
				}
				if defType != nil {
					gotTypes = append(gotTypes, defType.Name)
				}
			}
			if diff := cmp.Diff(gotTypes, typeNamesList(tt.wantDefiningTypes)); diff != "" {
				t.Errorf("definingType: (-got, +want):\n%s", diff)
			}
		})
	}
}

// TestFindFirstNonChoiceOrCase tests the functionality associated with extracting non-choice
// or case elements from a YANG structure.
func TestFindFirstNonChoiceOrCase(t *testing.T) {
	tests := []struct {
		name        string
		inEntry     *yang.Entry
		wantPaths   []string
		wantEntries []string
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
		wantPaths:   []string{"/parent/child-one", "/parent/child-two"},
		wantEntries: []string{"child-one", "child-two"},
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
		wantPaths:   []string{"/parent/child-one", "/parent/child-two"},
		wantEntries: []string{"child-one", "child-two"},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			check := func(testName string, found map[string]*yang.Entry, exp map[string]bool) {
				// Check whether the expected paths were found.
				for k := range found {
					if _, ok := found[k]; ok {
						exp[k] = true
					} else {
						t.Errorf("%s() %s: could not find expected node %s", testName, tt.name, k)
					}
				}

				// Check that all expected paths were found.
				for k, v := range exp {
					if v == false {
						t.Errorf("%s() %s: did not find expected node %s", testName, tt.name, k)
					}
				}
			}

			exp := make(map[string]bool)
			for _, k := range tt.wantPaths {
				exp[k] = false
			}

			found := FindFirstNonChoiceOrCase(tt.inEntry)
			check("FindFirstNonChoiceOrCase", found, exp)

			exp = make(map[string]bool)
			for _, k := range tt.wantEntries {
				exp[k] = false
			}

			found, err := findFirstNonChoiceOrCaseEntry(tt.inEntry)
			if err != nil {
				t.Fatal(err)
			}
			check("findFirstNonChoiceOrCaseEntry", found, exp)
		})
	}
}

// populateParentField recurses through schema and populates each Parent field
// with the parent schema node ptr.
func populateParentField(parent, schema *yang.Entry) {
	schema.Parent = parent
	for _, e := range schema.Dir {
		populateParentField(schema, e)
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
				ListAttr: yang.NewDefaultListAttr(),
			},
			"list": {
				Name:     "list",
				Kind:     yang.DirectoryEntry,
				ListAttr: yang.NewDefaultListAttr(),
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
					"int32-ref": {
						Name: "int32-ref",
						Kind: yang.LeafEntry,
						Type: &yang.YangType{
							Kind: yang.Yleafref,
							Path: "../int32",
						},
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
					"int32-ref-to-ref": {
						Name: "int32-ref-to-ref",
						Kind: yang.LeafEntry,
						Type: &yang.YangType{
							Kind: yang.Yleafref,
							Path: "../../oc:list[key=current()/../int32-ref-to-leaf]/oc:int32-ref",
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
						ListAttr: yang.NewDefaultListAttr(),
					},
					"absolute-to-int32": {
						Name: "absolute-to-int32",
						Kind: yang.LeafEntry,
						Type: &yang.YangType{
							Kind: yang.Yleafref,
							Path: "/int32",
						},
						ListAttr: yang.NewDefaultListAttr(),
					},
					"recursive": {
						Name: "recursive",
						Kind: yang.LeafEntry,
						Type: &yang.YangType{
							Kind: yang.Yleafref,
							Path: "../leaf-list-with-leafref",
						},
						ListAttr: yang.NewDefaultListAttr(),
					},
					"bad-path": {
						Name: "bad-path",
						Kind: yang.LeafEntry,
						Type: &yang.YangType{
							Kind: yang.Yleafref,
							Path: "../../missing",
						},
						ListAttr: yang.NewDefaultListAttr(),
					},
					"missing-path": {
						Name: "missing-path",
						Kind: yang.LeafEntry,
						Type: &yang.YangType{
							Kind: yang.Yleafref,
						},
						ListAttr: yang.NewDefaultListAttr(),
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
			desc: "container2/int32-ref-to-ref",
			in:   containerWithLeafListSchema.Dir["container2"].Dir["int32-ref-to-ref"],
			want: containerWithLeafListSchema.Dir["list"].Dir["int32"],
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
				t.Errorf("got error: %s, want error: %s", got, want)
			}
			testErrLog(t, tt.desc, err)
			if err == nil {
				if got, want := s, tt.want; got != want {
					t.Errorf("struct got:\n%v\n want:\n%v\n", pretty.Sprint(got), pretty.Sprint(want))
				}
			}
		})
	}
}

func TestListKeyFieldsMap(t *testing.T) {
	tests := []struct {
		desc  string
		entry *yang.Entry
		want  map[string]bool
	}{{
		desc: "empty",
		entry: &yang.Entry{
			Key: "",
		},
		want: map[string]bool{},
	}, {
		desc: "one one-letter key",
		entry: &yang.Entry{
			Key: "a",
		},
		want: map[string]bool{"a": true},
	}, {
		desc: "two one-letter keys",
		entry: &yang.Entry{
			Key: "a b",
		},
		want: map[string]bool{"a": true, "b": true},
	}, {
		desc: "one multi-letter key",
		entry: &yang.Entry{
			Key: "abc",
		},
		want: map[string]bool{"abc": true},
	}, {
		desc: "three variable letter keys",
		entry: &yang.Entry{
			Key: "ab a abc",
		},
		want: map[string]bool{"a": true, "ab": true, "abc": true},
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got, want := ListKeyFieldsMap(tt.entry), tt.want
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("ListKeyFieldsMap(%v): did not get expected map, (-want, +got):\n%s", tt.entry, diff)
			}
		})
	}
}
