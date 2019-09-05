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

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/kylelemons/godebug/pretty"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygot"
)

// TestUnionSubTypes extracts the types which make up a YANG union from a
// Goyang YangType struct.
func TestUnionSubTypes(t *testing.T) {
	tests := []struct {
		name       string
		in         *yang.YangType
		inCtxEntry *yang.Entry
		want       []string
		wantMtypes map[int]*MappedType
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
		wantMtypes: map[int]*MappedType{
			0: {"string", nil, false, goZeroValues["string"], nil},
		},
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
		wantMtypes: map[int]*MappedType{
			0: {"int8", nil, false, goZeroValues["int8"], nil},
			1: {"string", nil, false, goZeroValues["string"], nil},
		},
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
		wantMtypes: map[int]*MappedType{
			0: {"string", nil, false, goZeroValues["string"], nil},
			1: {"int32", nil, false, goZeroValues["int32"], nil},
			2: {"uint64", nil, false, goZeroValues["uint64"], nil},
			3: {"int16", nil, false, goZeroValues["int16"], nil},
		},
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
		wantMtypes: map[int]*MappedType{
			0: {"E_Basemod_Id", nil, true, "0", nil},
			1: {"E_Basemod2_Id2", nil, true, "0", nil},
		},
	}, {
		name: "union of single identityref",
		in: &yang.YangType{
			Name: "union",
			Kind: yang.Yunion,
			Type: []*yang.YangType{{
				Kind:    yang.Yidentityref,
				Name:    "identityref",
				Default: "prefix:CHIPS",
				IdentityBase: &yang.Identity{
					Name: "base-identity",
					Parent: &yang.Module{
						Name: "base-module",
					},
				},
			}},
		},
		inCtxEntry: &yang.Entry{
			Name: "union-leaf",
			Kind: yang.LeafEntry,
			Type: &yang.YangType{
				Name: "identityref",
				Kind: yang.Yunion,
				Type: []*yang.YangType{{
					Kind:    yang.Yidentityref,
					Name:    "identityref",
					Default: "prefix:CHIPS",
					IdentityBase: &yang.Identity{
						Name: "base-identity",
						Parent: &yang.Module{
							Name: "base-module",
						},
					},
				}},
			},
			Parent: &yang.Entry{Name: "base-module"},
			Node: &yang.Leaf{
				Parent: &yang.Module{
					Name: "base-module",
				},
			},
		},
		want: []string{"E_BaseModule_BaseIdentity"},
		wantMtypes: map[int]*MappedType{
			0: {
				NativeType:        "E_BaseModule_BaseIdentity",
				unionTypes:        nil,
				isEnumeratedValue: true,
				zeroValue:         "0",
				defaultValue:      ygot.String("BaseModule_BaseIdentity_CHIPS"),
			},
		},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newGenState()
			mtypes := make(map[int]*MappedType)
			ctypes := make(map[string]int)
			errs := s.goUnionSubTypes(tt.in, tt.inCtxEntry, ctypes, mtypes, false)
			if !tt.wantErr && errs != nil {
				t.Errorf("unexpected errors: %v", errs)
			}

			for i, wt := range tt.want {
				if unionidx, ok := ctypes[wt]; !ok {
					t.Errorf("could not find expected type in ctypes: %s", wt)
					continue
				} else if i != unionidx {
					t.Errorf("index of type %s was not as expected (%d != %d)", wt, i, unionidx)
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
					t.Errorf("found unexpected type %s", ct)
				}
			}

			if diff := cmp.Diff(mtypes, tt.wantMtypes, cmp.AllowUnexported(MappedType{}), cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("mtypes not as expected\n%s", diff)
			}
		})
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
		want         *MappedType
		wantErr      bool
	}{{
		name: "simple lookup resolution",
		in:   &yang.YangType{Kind: yang.Yint32, Name: "int32"},
		want: &MappedType{NativeType: "int32", zeroValue: "0"},
	}, {
		name: "int32 with default",
		in:   &yang.YangType{Kind: yang.Yint32, Name: "int32", Default: "42"},
		want: &MappedType{NativeType: "int32", zeroValue: "0", defaultValue: ygot.String("42")},
	}, {
		name: "decimal64",
		in:   &yang.YangType{Kind: yang.Ydecimal64, Name: "decimal64"},
		want: &MappedType{NativeType: "float64", zeroValue: "0.0"},
	}, {
		name: "binary lookup resolution",
		in:   &yang.YangType{Kind: yang.Ybinary, Name: "binary"},
		want: &MappedType{NativeType: "Binary", zeroValue: "nil"},
	}, {
		name: "unknown lookup resolution",
		in:   &yang.YangType{Kind: yang.YinstanceIdentifier, Name: "instanceIdentifier"},
		want: &MappedType{NativeType: "interface{}", zeroValue: "nil"},
	}, {
		name: "simple empty resolution",
		in:   &yang.YangType{Kind: yang.Yempty, Name: "empty"},
		want: &MappedType{NativeType: "YANGEmpty", zeroValue: "false"},
	}, {
		name: "simple boolean resolution",
		in:   &yang.YangType{Kind: yang.Ybool, Name: "bool"},
		want: &MappedType{NativeType: "bool", zeroValue: "false"},
	}, {
		name: "simple int64 resolution",
		in:   &yang.YangType{Kind: yang.Yint64, Name: "int64"},
		want: &MappedType{NativeType: "int64", zeroValue: "0"},
	}, {
		name: "simple uint8 resolution",
		in:   &yang.YangType{Kind: yang.Yuint8, Name: "uint8"},
		want: &MappedType{NativeType: "uint8", zeroValue: "0"},
	}, {
		name: "simple uint16 resolution",
		in:   &yang.YangType{Kind: yang.Yuint16, Name: "uint16"},
		want: &MappedType{NativeType: "uint16", zeroValue: "0"},
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
				{Kind: yang.Ystring, Name: "string"},
				{Kind: yang.Yint8, Name: "int8"},
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
		want: &MappedType{
			NativeType: "Module_Container_Leaf_Union",
			unionTypes: map[string]int{"string": 0, "int8": 1},
			zeroValue:  "nil",
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
		want: &MappedType{
			NativeType: "string",
			unionTypes: map[string]int{"string": 0},
			zeroValue:  `""`,
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
		want: &MappedType{
			NativeType:        "E_BaseModule_DerivedIdentityref",
			isEnumeratedValue: true,
			zeroValue:         "0",
		},
	}, {
		name: "derived identityref",
		in:   &yang.YangType{Kind: yang.Yidentityref, Name: "derived-identityref", Default: "AARDVARK"},
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
		want: &MappedType{
			NativeType:        "E_BaseModule_DerivedIdentityref",
			isEnumeratedValue: true,
			zeroValue:         "0",
			defaultValue:      ygot.String("BaseModule_DerivedIdentityref_AARDVARK"),
		},
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
		want: &MappedType{
			NativeType:        "E_BaseModule_EnumerationLeaf",
			isEnumeratedValue: true,
			zeroValue:         "0",
		},
	}, {
		name: "enumeration with default",
		in:   &yang.YangType{Kind: yang.Yenum, Name: "enumeration", Default: "prefix:BLUE"},
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
		want: &MappedType{
			NativeType:        "E_BaseModule_EnumerationLeaf",
			isEnumeratedValue: true,
			zeroValue:         "0",
			defaultValue:      ygot.String("BaseModule_EnumerationLeaf_BLUE"),
		},
	}, {
		name: "enumeration in union as the lone type with default",
		in: &yang.YangType{
			Name: "union",
			Kind: yang.Yunion,
			Type: []*yang.YangType{
				{Kind: yang.Yenum, Name: "enumeration", Default: "prefix:BLUE"},
			},
		},
		ctx: &yang.Entry{
			Name: "union-leaf",
			Kind: yang.LeafEntry,
			Type: &yang.YangType{
				Name: "union",
				Kind: yang.Yunion,
				Type: []*yang.YangType{
					{Kind: yang.Yenum, Name: "enumeration", Default: "prefix:BLUE"},
				},
			},
			Parent: &yang.Entry{Name: "base-module"},
			Node: &yang.Enum{
				Parent: &yang.Module{Name: "base-module"},
			},
		},
		want: &MappedType{
			NativeType:        "E_BaseModule_UnionLeaf",
			unionTypes:        map[string]int{"E_BaseModule_UnionLeaf": 0},
			isEnumeratedValue: true,
			zeroValue:         "0",
			defaultValue:      ygot.String("BaseModule_UnionLeaf_BLUE"),
		},
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
		want: &MappedType{
			NativeType:        "E_BaseModule_DerivedEnumeration",
			isEnumeratedValue: true,
			zeroValue:         "0",
		},
	}, {
		name: "typedef enumeration in union as the lone type",
		in: &yang.YangType{
			Name: "union",
			Kind: yang.Yunion,
			Type: []*yang.YangType{
				{Kind: yang.Yenum, Name: "enumeration"},
			},
		},
		ctx: &yang.Entry{
			Name: "union-leaf",
			Kind: yang.LeafEntry,
			Type: &yang.YangType{
				Name: "union",
				Kind: yang.Yunion,
				Type: []*yang.YangType{
					{Kind: yang.Yenum, Name: "enumeration"},
				},
			},
			Parent: &yang.Entry{Name: "base-module"},
			Node: &yang.Enum{
				Parent: &yang.Module{
					Name: "base-module",
				},
			},
		},
		want: &MappedType{
			NativeType:        "E_BaseModule_UnionLeaf",
			unionTypes:        map[string]int{"E_BaseModule_UnionLeaf": 0},
			isEnumeratedValue: true,
			zeroValue:         "0",
		},
	}, {
		name: "typedef enumeration with default",
		in:   &yang.YangType{Kind: yang.Yenum, Name: "derived-enumeration", Default: "FISH"},
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
		want: &MappedType{
			NativeType:        "E_BaseModule_DerivedEnumeration",
			isEnumeratedValue: true,
			zeroValue:         "0",
			defaultValue:      ygot.String("BaseModule_DerivedEnumeration_FISH"),
		},
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
		want: &MappedType{
			NativeType:        "E_TestModule_BaseIdentity",
			isEnumeratedValue: true,
			zeroValue:         "0",
		},
	}, {
		name: "identityref with default",
		in:   &yang.YangType{Kind: yang.Yidentityref, Name: "identityref", Default: "CHIPS"},
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
		want: &MappedType{
			NativeType:        "E_TestModule_BaseIdentity",
			isEnumeratedValue: true,
			zeroValue:         "0",
			defaultValue:      ygot.String("TestModule_BaseIdentity_CHIPS"),
		},
	}, {
		name: "identityref in union as the lone type with default",
		in: &yang.YangType{
			Name: "union",
			Kind: yang.Yunion,
			Type: []*yang.YangType{{
				Kind:    yang.Yidentityref,
				Name:    "identityref",
				Default: "prefix:CHIPS",
				IdentityBase: &yang.Identity{
					Name: "base-identity",
					Parent: &yang.Module{
						Name: "base-module",
					},
				},
			}},
		},
		ctx: &yang.Entry{
			Name: "union-leaf",
			Kind: yang.LeafEntry,
			Type: &yang.YangType{
				Name: "identityref",
				Kind: yang.Yunion,
				Type: []*yang.YangType{{
					Kind:    yang.Yidentityref,
					Name:    "identityref",
					Default: "prefix:CHIPS",
					IdentityBase: &yang.Identity{
						Name: "base-identity",
						Parent: &yang.Module{
							Name: "base-module",
						},
					},
				}},
			},
			Parent: &yang.Entry{Name: "base-module"},
			Node: &yang.Leaf{
				Parent: &yang.Module{
					Name: "base-module",
				},
			},
		},
		want: &MappedType{
			NativeType:        "E_BaseModule_BaseIdentity",
			unionTypes:        map[string]int{"E_BaseModule_BaseIdentity": 0},
			isEnumeratedValue: true,
			zeroValue:         "0",
			defaultValue:      ygot.String("BaseModule_BaseIdentity_CHIPS"),
		},
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
		want: &MappedType{
			NativeType:        "E_BaseModule_Container_Eleaf",
			isEnumeratedValue: true,
			zeroValue:         "0",
		},
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
		want:         &MappedType{NativeType: "E_BaseMod_Container_Eleaf", isEnumeratedValue: true, zeroValue: "0"},
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
		want: &MappedType{NativeType: "uint32", zeroValue: "0"},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newGenState()
			if tt.inEntries != nil {
				st, err := buildSchemaTree(tt.inEntries)
				if err != nil {
					t.Fatalf("buildSchemaTree(%v): could not build schema tree: %v", tt.inEntries, err)
				}
				s.schematree = st
			}

			args := resolveTypeArgs{
				yangType:     tt.in,
				contextEntry: tt.ctx,
			}

			got, err := s.yangTypeToGoType(args, tt.compressPath)
			if tt.wantErr && err == nil {
				t.Fatalf("did not get expected error (%v)", got)

			} else if !tt.wantErr && err != nil {
				t.Errorf("error returned when mapping type: %v", err)
			}

			if err != nil {
				return
			}

			if diff := pretty.Compare(got, tt.want); diff != "" {
				t.Fatalf("did not get expected result, diff(-got,+want):\n%s", diff)
			}
		})
	}
}

// TestBuildListKey takes an input yang.Entry and ensures that the correct YangListAttr
// struct is returned representing the keys of the list e.
func TestBuildListKey(t *testing.T) {
	tests := []struct {
		name       string        // name is the test identifier.
		in         *yang.Entry   // in is the yang.Entry of the test list.
		inCompress bool          // inCompress is a boolean indicating whether CompressOCPaths should be true/false.
		inEntries  []*yang.Entry // inEntries is used to provide context entries in the schema, particularly where a leafref key is used.
		want       YangListAttr  // want is the expected YangListAttr output.
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
		want: YangListAttr{
			Keys: map[string]*MappedType{
				"keyleaf": {NativeType: "string"},
			},
			KeyElems: []*yang.Entry{
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
		want: YangListAttr{},
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
		want: YangListAttr{
			Keys: map[string]*MappedType{
				"keyleafref": {NativeType: "string"},
			},
			KeyElems: []*yang.Entry{
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
		want: YangListAttr{
			Keys: map[string]*MappedType{
				"key1": {NativeType: "string"},
				"key2": {NativeType: "string"},
			},
			KeyElems: []*yang.Entry{
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
		want: YangListAttr{
			Keys: map[string]*MappedType{
				"key1": {NativeType: "string"},
				"key2": {NativeType: "int8"},
			},
			KeyElems: []*yang.Entry{
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
		want: YangListAttr{
			Keys: map[string]*MappedType{
				"keyleafref": {NativeType: "string"},
			},
			KeyElems: []*yang.Entry{
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
		want: YangListAttr{
			Keys: map[string]*MappedType{
				"keyleafref": {NativeType: "string"},
			},
			KeyElems: []*yang.Entry{
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

		for name, gtype := range got.Keys {
			elem, ok := tt.want.Keys[name]
			if !ok {
				t.Errorf("%s: could not find key %s", tt.name, name)
				continue
			}
			if elem.NativeType != gtype.NativeType {
				t.Errorf("%s: key %s had the wrong type %s", tt.name, name, gtype.NativeType)
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
		// describing the MappedType that is expected to be output.
		wantTypes map[string]*MappedType
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
		wantTypes: map[string]*MappedType{
			"/test-module/leaf-one": {NativeType: "E_TestModule_BaseIdentity", isEnumeratedValue: true, zeroValue: "0"},
			"/test-module/leaf-two": {NativeType: "E_TestModule_BaseIdentity", isEnumeratedValue: true, zeroValue: "0"},
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
		wantTypes: map[string]*MappedType{
			"/base-module/leaf-one": {NativeType: "E_BaseModule_DefinedType", isEnumeratedValue: true, zeroValue: "0"},
			"/base-module/leaf-two": {NativeType: "E_BaseModule_DefinedType", isEnumeratedValue: true, zeroValue: "0"},
		},
	}}

	for _, tt := range tests {
		s := newGenState()
		gotTypes := make(map[string]*MappedType)
		for _, leaf := range tt.inLeaves {
			mtype, err := s.yangTypeToGoType(resolveTypeArgs{yangType: leaf.Type, contextEntry: leaf}, tt.inCompressOCPaths)
			if err != nil {
				t.Errorf("%s: yangTypeToGoType(%v, %v): got unexpected err: %v, want: nil", tt.name, leaf.Type, leaf, err)
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
