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
			Kind:   yang.DirectoryEntry,
			Dir: map[string]*yang.Entry{
				"name": {Name: "name"},
				"config": {
					Name:   "config",
					Config: yang.TSTrue,
					Type:   &yang.YangType{},
					Kind:   yang.DirectoryEntry,
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
	}, {
		name: "choice entry within state",
		inElement: &yang.Entry{
			Name: "container",
			Kind: yang.DirectoryEntry,
			Dir: map[string]*yang.Entry{
				"state": {
					Name: "state",
					Kind: yang.DirectoryEntry,
					Dir: map[string]*yang.Entry{
						"choice": {
							Kind: yang.ChoiceEntry,
							Dir: map[string]*yang.Entry{
								"case": {
									Kind: yang.CaseEntry,
									Dir: map[string]*yang.Entry{
										"string": {
											Name: "string",
										},
									},
								},
							},
						},
					},
				},
			},
		},
		wantCompressed: []yang.Entry{{
			Name: "string",
		}},
		wantUncompressed: []yang.Entry{{
			Name: "state",
		}},
	}, {
		name: "choice entry within config",
		inElement: &yang.Entry{
			Name: "container",
			Kind: yang.DirectoryEntry,
			Dir: map[string]*yang.Entry{
				"config": {
					Name: "config",
					Kind: yang.DirectoryEntry,
					Dir: map[string]*yang.Entry{
						"choice": {
							Kind: yang.ChoiceEntry,
							Dir: map[string]*yang.Entry{
								"case": {
									Kind: yang.CaseEntry,
									Dir: map[string]*yang.Entry{
										"string": {
											Name: "string",
										},
									},
								},
							},
						},
					},
				},
			},
		},
		wantCompressed: []yang.Entry{{
			Name: "string",
		}},
		wantUncompressed: []yang.Entry{{
			Name: "config",
		}},
	}}

	for _, tt := range tests {
		for compress, expected := range map[bool][]yang.Entry{true: tt.wantCompressed, false: tt.wantUncompressed} {
			elems, errs := findAllChildren(tt.inElement, compress)
			if tt.wantErr == nil && errs != nil {
				t.Errorf("%s (compress: %v): errors %v for children of %s", tt.name, compress, errs, tt.inElement.Name)
			} else {
				if expErr, ok := tt.wantErr[compress]; ok {
					if (errs != nil) != expErr {
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
					delete(retMap, expectEntry.Name)
					if elem.Config != expectEntry.Config {
						t.Errorf("%s (compress: %v): element %s had wrong config status %s", tt.name, compress,
							expectEntry.Name, elem.Config)
					}
					if elem.Type != nil && elem.Type.Kind != expectEntry.Type.Kind {
						t.Errorf("%s (compress: %v): element %s had wrong type %s", tt.name,
							compress, expectEntry.Name, elem.Type.Kind)
					}
				} else {
					t.Errorf("%s (compress: %v): could not find expected child %s in %s", tt.name, compress,
						expectEntry.Name, tt.inElement.Name)
				}
			}

			if len(retMap) != 0 && expected != nil {
				t.Errorf("%s (compress: %v): got unexpected entries, got: %v, want: nil", tt.name, compress, retMap)
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
	}, {
		name: "non-camelcase extension",
		inEntry: &yang.Entry{
			Name: "little-creatures",
			Exts: []*yang.Statement{{
				Keyword:     "amod:other-ext",
				HasArgument: true,
				Argument:    "true\n",
			}},
		},
		wantName: "LittleCreatures",
	}}

	for _, tt := range tests {
		ctx := make(map[string]bool)
		for _, prevName := range tt.inPrevNames {
			_ = makeNameUnique(entryCamelCaseName(prevName), ctx)
		}

		if got := makeNameUnique(entryCamelCaseName(tt.inEntry), ctx); got != tt.wantName {
			t.Errorf("%s: did not get expected name for %v (after defining %v): %s",
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
		errs := s.goUnionSubTypes(tt.in, tt.inCtxEntry, ctypes, false)
		if !tt.wantErr && errs != nil {
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

// TestYangTypeToGoType tests the resolution of a particular YangType to the
// corresponding Go type.
func TestYangTypeToGoType(t *testing.T) {
	tests := []struct {
		name         string
		in           *yang.YangType
		ctx          *yang.Entry
		inEntries    []*yang.Entry
		compressPath bool
		want         *mappedType
		wantErr      bool
	}{{
		name: "simple lookup resolution",
		in:   &yang.YangType{Kind: yang.Yint32, Name: "int32"},
		want: &mappedType{nativeType: "int32"},
	}, {
		name: "decimal64",
		in:   &yang.YangType{Kind: yang.Ydecimal64, Name: "decimal64"},
		want: &mappedType{nativeType: "float64"},
	}, {
		name: "binary lookup resolution",
		in:   &yang.YangType{Kind: yang.Ybinary, Name: "binary"},
		want: &mappedType{nativeType: "Binary"},
	}, {
		name: "unknown lookup resolution",
		in:   &yang.YangType{Kind: yang.YinstanceIdentifier, Name: "instanceIdentifier"},
		want: &mappedType{nativeType: "interface{}"},
	}, {
		name: "simple empty resolution",
		in:   &yang.YangType{Kind: yang.Yempty, Name: "empty"},
		want: &mappedType{nativeType: "YANGEmpty"},
	}, {
		name: "simple boolean resolution",
		in:   &yang.YangType{Kind: yang.Ybool, Name: "bool"},
		want: &mappedType{nativeType: "bool"},
	}, {
		name: "simple int64 resolution",
		in:   &yang.YangType{Kind: yang.Yint64, Name: "int64"},
		want: &mappedType{nativeType: "int64"},
	}, {
		name: "simple uint8 resolution",
		in:   &yang.YangType{Kind: yang.Yuint8, Name: "uint8"},
		want: &mappedType{nativeType: "uint8"},
	}, {
		name: "simple uint16 resolution",
		in:   &yang.YangType{Kind: yang.Yuint16, Name: "uint16"},
		want: &mappedType{nativeType: "uint16"},
	}, {
		name:    "leafref without valid path",
		in:      &yang.YangType{Kind: yang.Yleafref, Name: "leafref"},
		wantErr: true,
	}, {
		name:    "enum without context",
		in:      &yang.YangType{Kind: yang.Yenum, Name: "enumeration"},
		wantErr: true,
	}, {
		name:    "identityref without context",
		in:      &yang.YangType{Kind: yang.Yidentityref, Name: "identityref"},
		wantErr: true,
	}, {
		name:    "typedef without context",
		in:      &yang.YangType{Kind: yang.Yenum, Name: "tdef"},
		wantErr: true,
	}, {
		name: "union with enum without context",
		in: &yang.YangType{
			Name: "union",
			Kind: yang.Yunion,
			Type: []*yang.YangType{
				{Kind: yang.Yenum, Name: "enumeration"},
			},
		},
		wantErr: true,
	}, {
		name: "union of string, int32",
		in: &yang.YangType{
			Kind: yang.Yunion,
			Type: []*yang.YangType{
				{Kind: yang.Yint8, Name: "int8"},
				{Kind: yang.Ystring, Name: "string"},
			},
		},
		ctx: &yang.Entry{
			Name: "leaf",
			Parent: &yang.Entry{
				Name: "container",
				Parent: &yang.Entry{
					Name: "module",
				},
			},
		},
		want: &mappedType{
			nativeType: "Module_Container_Leaf_Union",
			unionTypes: map[string]int{"string": 0, "int8": 1},
		},
	}, {
		name: "string-only union",
		in: &yang.YangType{
			Kind: yang.Yunion,
			Type: []*yang.YangType{
				{Kind: yang.Ystring, Name: "string"},
				{Kind: yang.Ystring, Name: "string"},
			},
		},
		want: &mappedType{
			nativeType: "string",
			unionTypes: map[string]int{"string": 0},
		},
	}, {
		name: "derived identityref",
		in:   &yang.YangType{Kind: yang.Yidentityref, Name: "derived-identityref"},
		ctx: &yang.Entry{
			Type: &yang.YangType{
				Name: "derived-identityref",
				IdentityBase: &yang.Identity{
					Name:   "base-identity",
					Parent: &yang.Module{Name: "base-module"},
				},
			},
			Node: &yang.Leaf{
				Parent: &yang.Module{Name: "base-module"},
			},
		},
		want: &mappedType{nativeType: "E_BaseModule_DerivedIdentityref", isEnumeratedValue: true},
	}, {
		name: "enumeration",
		in:   &yang.YangType{Kind: yang.Yenum, Name: "enumeration"},
		ctx: &yang.Entry{
			Name: "enumeration-leaf",
			Type: &yang.YangType{
				Name: "enumeration",
				Enum: &yang.EnumType{},
			},
			Parent: &yang.Entry{Name: "base-module"},
			Node: &yang.Enum{
				Parent: &yang.Module{Name: "base-module"},
			},
		},
		want: &mappedType{nativeType: "E_BaseModule_EnumerationLeaf", isEnumeratedValue: true},
	}, {
		name: "typedef enumeration",
		in:   &yang.YangType{Kind: yang.Yenum, Name: "derived-enumeration"},
		ctx: &yang.Entry{
			Name: "enumeration-leaf",
			Type: &yang.YangType{
				Name: "derived-enumeration",
				Enum: &yang.EnumType{},
			},
			Node: &yang.Enum{
				Parent: &yang.Module{
					Name: "base-module",
				},
			},
		},
		want: &mappedType{nativeType: "E_BaseModule_DerivedEnumeration", isEnumeratedValue: true},
	}, {
		name: "identityref",
		in:   &yang.YangType{Kind: yang.Yidentityref, Name: "identityref"},
		ctx: &yang.Entry{
			Name: "identityref",
			Type: &yang.YangType{
				Name: "identityref",
				IdentityBase: &yang.Identity{
					Name: "base-identity",
					Parent: &yang.Module{
						Name: "test-module",
					},
				},
			},
			Node: &yang.Leaf{
				Parent: &yang.Module{
					Name: "test-module",
				},
			},
		},
		want: &mappedType{nativeType: "E_TestModule_BaseIdentity", isEnumeratedValue: true},
	}, {
		name: "enumeration with compress paths",
		in:   &yang.YangType{Kind: yang.Yenum, Name: "enumeration"},
		ctx: &yang.Entry{
			Name: "eleaf",
			Type: &yang.YangType{
				Name: "enumeration",
				Enum: &yang.EnumType{},
			},
			Parent: &yang.Entry{
				Name: "config", Parent: &yang.Entry{Name: "container"}},
			Node: &yang.Enum{
				Parent: &yang.Module{Name: "base-module"},
			},
		},
		compressPath: true,
		want:         &mappedType{nativeType: "E_BaseModule_Container_Eleaf", isEnumeratedValue: true},
	}, {
		name: "enumeration in submodule",
		in:   &yang.YangType{Kind: yang.Yenum, Name: "enumeration"},
		ctx: &yang.Entry{
			Name:   "eleaf",
			Type:   &yang.YangType{Name: "enumeration", Enum: &yang.EnumType{}},
			Parent: &yang.Entry{Name: "config", Parent: &yang.Entry{Name: "container"}},
			Node: &yang.Enum{
				Parent: &yang.Module{Name: "submodule", BelongsTo: &yang.BelongsTo{Name: "base-mod"}},
			},
		},
		compressPath: true,
		want:         &mappedType{nativeType: "E_BaseMod_Container_Eleaf", isEnumeratedValue: true},
	}, {
		name: "leafref",
		in:   &yang.YangType{Kind: yang.Yleafref, Name: "leafref", Path: "../c"},
		ctx: &yang.Entry{
			Name: "d",
			Parent: &yang.Entry{
				Name: "b",
				Parent: &yang.Entry{
					Name:   "a",
					Parent: &yang.Entry{Name: "module"},
				},
			},
		},
		inEntries: []*yang.Entry{
			{
				Name: "a",
				Dir: map[string]*yang.Entry{
					"b": {
						Name: "b",
						Dir: map[string]*yang.Entry{
							"c": {
								Name: "c",
								Type: &yang.YangType{Kind: yang.Yuint32},
								Parent: &yang.Entry{
									Name: "b",
									Parent: &yang.Entry{
										Name:   "a",
										Parent: &yang.Entry{Name: "module"},
									},
								},
							},
						},
						Parent: &yang.Entry{
							Name:   "a",
							Parent: &yang.Entry{Name: "module"},
						},
					},
				},
				Parent: &yang.Entry{Name: "module"},
			},
		},
		want: &mappedType{nativeType: "uint32"},
	}}

	for _, tt := range tests {
		s := newGenState()
		if tt.inEntries != nil {
			st, err := buildSchemaTree(tt.inEntries)
			if err != nil {
				t.Errorf("%s: buildSchemaTree(%v): could not build schema tree: %v", tt.name, tt.inEntries, err)
				continue
			}
			s.schematree = st
		}

		args := resolveTypeArgs{
			yangType:     tt.in,
			contextEntry: tt.ctx,
		}

		mappedType, err := s.yangTypeToGoType(args, tt.compressPath)
		if tt.wantErr && err == nil {
			t.Errorf("%s: did not get expected error (%v)", tt.name, mappedType)
			continue
		} else if !tt.wantErr && err != nil {
			t.Errorf("%s: error returned when mapping type: %v", tt.name, err)
		}

		if err != nil {
			continue
		}

		if mappedType.nativeType != tt.want.nativeType {
			t.Errorf("%s: wrong type returned when mapping type: %s", tt.name, mappedType.nativeType)
		}

		if tt.want.unionTypes != nil {
			for k := range tt.want.unionTypes {
				if _, ok := mappedType.unionTypes[k]; !ok {
					t.Errorf("%s: union type did not include expected type: %s", tt.name, k)
				}
			}
		}

		if mappedType.isEnumeratedValue != tt.want.isEnumeratedValue {
			t.Errorf("%s: returned isEnumeratedValue was incorrect, got: %v, want: %v", tt.name, mappedType.isEnumeratedValue, tt.want.isEnumeratedValue)
		}
	}
}

// TestBuildListKey takes an input yang.Entry and ensures that the correct yangListAttr
// struct is returned representing the keys of the list e.
func TestBuildListKey(t *testing.T) {
	tests := []struct {
		name       string        // name is the test identifier.
		in         *yang.Entry   // in is the yang.Entry of the test list.
		inCompress bool          // inCompress is a boolean indicating whether CompressOCPaths should be true/false.
		inEntries  []*yang.Entry // inEntries is used to provide context entries in the schema, particularly where a leafref key is used.
		want       yangListAttr  // want is the expected yangListAttr output.
		wantErr    bool          // wantErr is a boolean indicating whether errors are expected from buildListKeys
	}{{
		name: "non-list",
		in: &yang.Entry{
			Name: "not-list",
			Dir: map[string]*yang.Entry{
				"keyleaf": {
					Name: "keyleaf",
					Type: &yang.YangType{Kind: yang.Ystring},
				},
			},
		},
		wantErr: true,
	}, {
		name: "no key in config true list",
		in: &yang.Entry{
			Name:     "list",
			ListAttr: &yang.ListAttr{},
			Dir: map[string]*yang.Entry{
				"keyleaf": {Type: &yang.YangType{Kind: yang.Ystring}},
			},
			Config: yang.TSTrue,
		},
		wantErr: true,
	}, {
		name: "invalid key in config true list",
		in: &yang.Entry{
			Name:     "list",
			ListAttr: &yang.ListAttr{},
			Dir: map[string]*yang.Entry{
				"keyleaf": {Type: &yang.YangType{Kind: yang.Yidentityref}},
			},
			Key: "keyleaf",
		},
		wantErr: true,
	}, {
		name: "basic list key test",
		in: &yang.Entry{
			Name:     "list",
			ListAttr: &yang.ListAttr{},
			Key:      "keyleaf",
			Dir: map[string]*yang.Entry{
				"keyleaf": {
					Name: "keyleaf",
					Type: &yang.YangType{Kind: yang.Ystring},
				},
			},
		},
		want: yangListAttr{
			keys: map[string]*mappedType{
				"keyleaf": {nativeType: "string"},
			},
			keyElems: []*yang.Entry{
				{
					Name: "keyleaf",
					Type: &yang.YangType{Kind: yang.Ystring},
				},
			},
		},
	}, {
		name: "missing key list",
		in: &yang.Entry{
			Name:     "list",
			ListAttr: &yang.ListAttr{},
			Key:      "null",
			Dir:      map[string]*yang.Entry{},
		},
		wantErr: true,
	}, {
		name: "keyless list test",
		in: &yang.Entry{
			Name:     "list",
			ListAttr: &yang.ListAttr{},
			Config:   yang.TSFalse,
			Dir:      map[string]*yang.Entry{},
		},
		want: yangListAttr{},
	}, {
		name: "list with invalid leafref path",
		in: &yang.Entry{
			Name:     "list",
			ListAttr: &yang.ListAttr{},
			Key:      "keyleaf",
			Dir: map[string]*yang.Entry{
				"keyleaf": {
					Name: "keyleaf",
					Type: &yang.YangType{
						Kind: yang.Yleafref,
						Path: "not-a-valid-path",
					},
				},
			},
		},
		inCompress: true,
		wantErr:    true,
	}, {
		name: "list with leafref in invalid container",
		in: &yang.Entry{
			Name:     "list",
			ListAttr: &yang.ListAttr{},
			Key:      "keyleaf",
			Dir: map[string]*yang.Entry{
				"keyleaf": {
					Name: "keyleaf",
					Type: &yang.YangType{
						Kind: yang.Yleafref,
						Path: "../config/keyleaf",
					},
				},
			},
		},
		inCompress: true,
		wantErr:    true,
	}, {
		name: "list with leafref that does not exist",
		in: &yang.Entry{
			Name:     "list",
			ListAttr: &yang.ListAttr{},
			Key:      "invalid",
			Dir: map[string]*yang.Entry{
				"invalid": {
					Name: "invalid",
					Type: &yang.YangType{
						Kind: yang.Yleafref,
						Path: "../config/invalid",
					},
				},
				"config": {
					Name: "config",
					Dir:  map[string]*yang.Entry{},
				},
			},
		},
		inCompress: true,
		wantErr:    true,
	}, {
		name: "single leafref key test",
		in: &yang.Entry{
			Name:     "list",
			ListAttr: &yang.ListAttr{},
			Key:      "keyleafref",
			Dir: map[string]*yang.Entry{
				"keyleafref": {
					Name: "keyleafref",
					Type: &yang.YangType{
						Kind: yang.Yleafref,
						Path: "../config/keyleafref",
					},
				},
				"config": {
					Name: "config",
					Dir: map[string]*yang.Entry{
						"keyleafref": {
							Name: "keyleafref",
							Type: &yang.YangType{Kind: yang.Ystring},
						},
					},
				},
			},
			Parent: &yang.Entry{
				Name: "container",
				Parent: &yang.Entry{
					Name: "module",
				},
			},
		},
		inCompress: true,
		want: yangListAttr{
			keys: map[string]*mappedType{
				"keyleafref": {nativeType: "string"},
			},
			keyElems: []*yang.Entry{
				{
					Name: "keyleafref",
					Type: &yang.YangType{Kind: yang.Ystring},
				},
			},
		},
	}, {
		name: "multiple key list test",
		in: &yang.Entry{
			Name:     "list",
			ListAttr: &yang.ListAttr{},
			Key:      "key1 key2",
			Dir: map[string]*yang.Entry{
				"key1": {
					Name: "key1",
					Type: &yang.YangType{Kind: yang.Ystring},
				},
				"key2": {
					Name: "key2",
					Type: &yang.YangType{Kind: yang.Ystring},
				},
			},
		},
		want: yangListAttr{
			keys: map[string]*mappedType{
				"key1": {nativeType: "string"},
				"key2": {nativeType: "string"},
			},
			keyElems: []*yang.Entry{
				{
					Name: "key1",
					Type: &yang.YangType{Kind: yang.Ystring},
				},
				{
					Name: "key2",
					Type: &yang.YangType{Kind: yang.Ystring},
				},
			},
		},
	}, {
		name: "multiple leafref key",
		in: &yang.Entry{
			Name:     "list",
			ListAttr: &yang.ListAttr{},
			Key:      "key1 key2",
			Dir: map[string]*yang.Entry{
				"key1": {
					Name: "key1",
					Type: &yang.YangType{
						Kind: yang.Yleafref,
						Path: "../state/key1",
					},
				},
				"key2": {
					Name: "key2",
					Type: &yang.YangType{
						Kind: yang.Yleafref,
						Path: "../state/key2",
					},
				},
				"state": {
					Name: "state",
					Dir: map[string]*yang.Entry{
						"key1": {
							Name: "key1",
							Type: &yang.YangType{Kind: yang.Ystring},
						},
						"key2": {
							Name: "key2",
							Type: &yang.YangType{Kind: yang.Yint8},
						},
					},
				},
			},
		},
		inCompress: true,
		want: yangListAttr{
			keys: map[string]*mappedType{
				"key1": {nativeType: "string"},
				"key2": {nativeType: "int8"},
			},
			keyElems: []*yang.Entry{
				{
					Name: "key2",
					Type: &yang.YangType{Kind: yang.Yint8},
				},
				{
					Name: "key1",
					Type: &yang.YangType{Kind: yang.Ystring},
				},
			},
		},
	}, {
		name: "single prefixed leafref key test",
		in: &yang.Entry{
			Name:     "list",
			ListAttr: &yang.ListAttr{},
			Key:      "keyleafref",
			Dir: map[string]*yang.Entry{
				"keyleafref": {
					Name: "keyleafref",
					Type: &yang.YangType{
						Kind: yang.Yleafref,
						Path: "../pfx:config/pfx:keyleafref",
					},
				},
				"config": {
					Name: "config",
					Dir: map[string]*yang.Entry{
						"keyleafref": {
							Name: "keyleafref",
							Type: &yang.YangType{Kind: yang.Ystring},
						},
					},
				},
			},
		},
		inCompress: true,
		want: yangListAttr{
			keys: map[string]*mappedType{
				"keyleafref": {nativeType: "string"},
			},
			keyElems: []*yang.Entry{
				{
					Name: "keyleafref",
					Type: &yang.YangType{Kind: yang.Ystring},
				},
			},
		},
	}, {
		name: "uncompressed leafref",
		in: &yang.Entry{
			Name:     "list",
			ListAttr: &yang.ListAttr{},
			Key:      "keyleafref",
			Dir: map[string]*yang.Entry{
				"keyleafref": {
					Name: "keyleafref",
					Type: &yang.YangType{
						Kind: yang.Yleafref,
						Path: "/a/b/c",
					},
				},
			},
		},
		inEntries: []*yang.Entry{
			{
				Name: "a",
				Dir: map[string]*yang.Entry{
					"b": {
						Name: "b",
						Dir: map[string]*yang.Entry{
							"c": {
								Name: "c",
								Type: &yang.YangType{
									Kind: yang.Ystring,
								},
								Parent: &yang.Entry{
									Name: "b",
									Parent: &yang.Entry{
										Name: "a",
										Parent: &yang.Entry{
											Name: "module",
										},
									},
								},
							},
						},
						Parent: &yang.Entry{
							Name: "a",
							Parent: &yang.Entry{
								Name: "module",
							},
						},
					},
				},
				Parent: &yang.Entry{Name: "module"},
			},
		},
		want: yangListAttr{
			keys: map[string]*mappedType{
				"keyleafref": {nativeType: "string"},
			},
			keyElems: []*yang.Entry{
				{
					Name: "keyleafref",
					Type: &yang.YangType{Kind: yang.Ystring},
				},
			},
		},
	}}

	for _, tt := range tests {
		s := newGenState()
		if tt.inEntries != nil {
			st, err := buildSchemaTree(tt.inEntries)
			if err != nil {
				t.Errorf("%s: buildSchemaTree(%v), could not build tree: %v", tt.name, tt.inEntries, err)
				continue
			}
			s.schematree = st
		}

		got, err := s.buildListKey(tt.in, tt.inCompress)
		if err != nil && !tt.wantErr {
			t.Errorf("%s: could not build list key successfully %v", tt.name, err)
		}

		if err == nil && tt.wantErr {
			t.Errorf("%s: did not get expected error", tt.name)
		}

		if tt.wantErr || got == nil {
			continue
		}

		for name, gtype := range got.keys {
			elem, ok := tt.want.keys[name]
			if !ok {
				t.Errorf("%s: could not find key %s", tt.name, name)
				continue
			}
			if elem.nativeType != gtype.nativeType {
				t.Errorf("%s: key %s had the wrong type %s", tt.name, name, gtype.nativeType)
			}
		}
	}
}

// TestTypeResolutionManyToOne tests cases where there can be many leaves that target the
// same underlying typedef or identity, ensuring that generated names are reused where required.
func TestTypeResolutionManyToOne(t *testing.T) {
	tests := []struct {
		name string // name is the test identifier.
		// inLeaves is the set of yang.Entry pointers that are to have types generated
		// for them.
		inLeaves []*yang.Entry
		// inCompressOCPaths enables or disables "CompressOCPaths" for the YANGCodeGenerator
		// instance used for the test.
		inCompressOCPaths bool
		// wantTypes is a map, keyed by the path of the yang.Entry within inLeaves and
		// describing the mappedType that is expected to be output.
		wantTypes map[string]*mappedType
	}{{
		name: "identity with multiple identityref leaves",
		inLeaves: []*yang.Entry{{
			Name: "leaf-one",
			Type: &yang.YangType{
				Name: "identityref",
				Kind: yang.Yidentityref,
				IdentityBase: &yang.Identity{
					Name: "base-identity",
					Parent: &yang.Module{
						Name: "test-module",
					},
				},
			},
			Parent: &yang.Entry{Name: "test-module"},
			Node: &yang.Leaf{
				Parent: &yang.Module{
					Name: "test-module",
				},
			},
		}, {
			Name: "leaf-two",
			Type: &yang.YangType{
				Name: "identityref",
				Kind: yang.Yidentityref,
				IdentityBase: &yang.Identity{
					Name: "base-identity",
					Parent: &yang.Module{
						Name: "test-module",
					},
				},
			},
			Parent: &yang.Entry{Name: "test-module"},
			Node: &yang.Leaf{
				Parent: &yang.Module{
					Name: "test-module",
				},
			},
		}},
		wantTypes: map[string]*mappedType{
			"/test-module/leaf-one": {nativeType: "E_TestModule_BaseIdentity", isEnumeratedValue: true},
			"/test-module/leaf-two": {nativeType: "E_TestModule_BaseIdentity", isEnumeratedValue: true},
		},
	}, {
		name: "typedef with multiple references",
		inLeaves: []*yang.Entry{{
			Name: "leaf-one",
			Parent: &yang.Entry{
				Name: "base-module",
			},
			Type: &yang.YangType{
				Name: "definedType",
				Kind: yang.Yenum,
				Enum: &yang.EnumType{},
			},
			Node: &yang.Enum{
				Parent: &yang.Module{
					Name: "base-module",
				},
			},
		}, {
			Name: "leaf-two",
			Parent: &yang.Entry{
				Name: "base-module",
			},
			Type: &yang.YangType{
				Name: "definedType",
				Kind: yang.Yenum,
				Enum: &yang.EnumType{},
			},
			Node: &yang.Enum{
				Parent: &yang.Module{
					Name: "base-module",
				},
			},
		}},
		wantTypes: map[string]*mappedType{
			"/base-module/leaf-one": {nativeType: "E_BaseModule_DefinedType", isEnumeratedValue: true},
			"/base-module/leaf-two": {nativeType: "E_BaseModule_DefinedType", isEnumeratedValue: true},
		},
	}}

	for _, tt := range tests {
		s := newGenState()
		gotTypes := make(map[string]*mappedType)
		for _, leaf := range tt.inLeaves {
			mtype, err := s.yangTypeToGoType(resolveTypeArgs{leaf.Type, leaf}, tt.inCompressOCPaths)
			if err != nil {
				t.Errorf("%s: yangTypeToGoType(%v, %v): got unexpected err: %v, want: nil",
					tt.name, leaf.Type, leaf, err)
				continue
			}
			gotTypes[leaf.Path()] = mtype
		}

		if diff := pretty.Compare(gotTypes, tt.wantTypes); diff != "" {
			t.Errorf("%s: yangTypesToGoTypes(...): incorrect output returned, diff (-got,+want):\n%s",
				tt.name, diff)
		}
	}
}
