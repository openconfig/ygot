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
		inCtxEntry *yang.Entry
		// inNoContext means to only pass in the type of the context
		// entry as a parameter to goUnionSubTypes without the context entry.
		inNoContext bool
		want        []string
		wantMtypes  map[int]*MappedType
		wantErr     bool
	}{{
		name: "union of strings",
		inCtxEntry: &yang.Entry{
			Name: "union-leaf",
			Kind: yang.LeafEntry,
			Type: &yang.YangType{
				Kind: yang.Yunion,
				Type: []*yang.YangType{
					{Kind: yang.Ystring},
					{Kind: yang.Ystring},
				},
			},
		},
		want: []string{"string"},
		wantMtypes: map[int]*MappedType{
			0: {"string", nil, false, goZeroValues["string"], nil},
		},
	}, {
		name: "union of int8, string",
		inCtxEntry: &yang.Entry{
			Name: "union-leaf",
			Kind: yang.LeafEntry,
			Type: &yang.YangType{
				Kind: yang.Yunion,
				Type: []*yang.YangType{
					{Kind: yang.Yint8},
					{Kind: yang.Ystring},
				},
			},
		},
		want: []string{"int8", "string"},
		wantMtypes: map[int]*MappedType{
			0: {"int8", nil, false, goZeroValues["int8"], nil},
			1: {"string", nil, false, goZeroValues["string"], nil},
		},
	}, {
		name: "union of unions",
		inCtxEntry: &yang.Entry{
			Name: "union-leaf",
			Kind: yang.LeafEntry,
			Type: &yang.YangType{
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
		inCtxEntry: &yang.Entry{
			Name: "union-leaf",
			Kind: yang.LeafEntry,
			Type: &yang.YangType{
				Name: "enumeration-union",
				Kind: yang.Yunion,
				Type: []*yang.YangType{{
					Kind: yang.Yenum,
					Name: "enumeration",
					Enum: &yang.EnumType{},
				}},
				Base: &yang.Type{
					Name: "union",
					Parent: &yang.Typedef{
						Name: "enumeration-union",
						Parent: &yang.Module{
							Name: "typedef-module",
						},
					},
				},
			},
			Parent: &yang.Entry{Name: "base-module"},
			Node: &yang.Leaf{
				Parent: &yang.Module{
					Name: "base-module",
				},
			},
		},
		inNoContext: true,
		wantErr:     true,
	}, {
		name: "typedef enum within a union",
		inCtxEntry: &yang.Entry{
			Name: "union-leaf",
			Kind: yang.LeafEntry,
			Type: &yang.YangType{
				Name: "union",
				Kind: yang.Yunion,
				Type: []*yang.YangType{{
					Kind: yang.Yenum,
					Name: "derived-enum",
					Enum: &yang.EnumType{},
					Base: &yang.Type{
						Name: "enumeration",
						Parent: &yang.Typedef{
							Name: "derived-enum",
							Parent: &yang.Module{
								Name: "typedef-module",
							},
						},
					},
				}, {
					Name: "int16",
					Kind: yang.Yint16,
				}},
			},
			Parent: &yang.Entry{Name: "base-module"},
			Node: &yang.Leaf{
				Parent: &yang.Module{
					Name: "base-module",
				},
			},
		},
		want: []string{"E_TypedefModule_DerivedEnum", "int16"},
		wantMtypes: map[int]*MappedType{
			0: {
				NativeType:        "E_TypedefModule_DerivedEnum",
				IsEnumeratedValue: true,
				ZeroValue:         "0",
			},
			1: {
				NativeType: "int16",
				ZeroValue:  "0",
			},
		},
	}, {
		name: "enum within a typedef union",
		inCtxEntry: &yang.Entry{
			Name: "union-leaf",
			Kind: yang.LeafEntry,
			Type: &yang.YangType{
				Name: "derived-union",
				Kind: yang.Yunion,
				Type: []*yang.YangType{{
					Kind: yang.Yenum,
					Name: "enumeration",
					Enum: &yang.EnumType{},
				}, {
					Name: "int16",
					Kind: yang.Yint16,
				}},
				Base: &yang.Type{
					Name: "union",
					Parent: &yang.Typedef{
						Name: "derived-union",
						Parent: &yang.Module{
							Name: "union-module",
						},
					},
				},
			},
			Parent: &yang.Entry{Name: "base-module"},
			Node: &yang.Leaf{
				Parent: &yang.Module{
					Name: "base-module",
				},
			},
		},
		want: []string{"E_UnionModule_DerivedUnion_Enum", "int16"},
		wantMtypes: map[int]*MappedType{
			0: {
				NativeType:        "E_UnionModule_DerivedUnion_Enum",
				IsEnumeratedValue: true,
				ZeroValue:         "0",
			},
			1: {
				NativeType: "int16",
				ZeroValue:  "0",
			},
		},
	}, {
		name: "typedef enum within a typedef union",
		inCtxEntry: &yang.Entry{
			Name: "union-leaf",
			Kind: yang.LeafEntry,
			Type: &yang.YangType{
				Name: "derived-union",
				Kind: yang.Yunion,
				Type: []*yang.YangType{{
					Kind: yang.Yenum,
					Name: "derived-enum",
					Enum: &yang.EnumType{},
					Base: &yang.Type{
						Name: "enumeration",
						Parent: &yang.Typedef{
							Name: "derived-enum",
							Parent: &yang.Module{
								Name: "typedef-module",
							},
						},
					},
				}, {
					Name: "int16",
					Kind: yang.Yint16,
				}},
				Base: &yang.Type{
					Name: "union",
					Parent: &yang.Typedef{
						Name: "derived-union",
						Parent: &yang.Module{
							Name: "union-module",
						},
					},
				},
			},
			Parent: &yang.Entry{Name: "base-module"},
			Node: &yang.Leaf{
				Parent: &yang.Module{
					Name: "base-module",
				},
			},
		},
		want: []string{"E_TypedefModule_DerivedEnum", "int16"},
		wantMtypes: map[int]*MappedType{
			0: {
				NativeType:        "E_TypedefModule_DerivedEnum",
				IsEnumeratedValue: true,
				ZeroValue:         "0",
			},
			1: {
				NativeType: "int16",
				ZeroValue:  "0",
			},
		},
	}, {
		name: "union of a single enum",
		inCtxEntry: &yang.Entry{
			Name: "union-leaf",
			Kind: yang.LeafEntry,
			Type: &yang.YangType{
				Name: "union",
				Kind: yang.Yunion,
				Type: []*yang.YangType{{
					Kind: yang.Yenum,
					Name: "enumeration",
					Enum: &yang.EnumType{},
				}},
			},
			Parent: &yang.Entry{Name: "base-module"},
			Node: &yang.Leaf{
				Parent: &yang.Module{
					Name: "base-module",
				},
			},
		},
		want: []string{"E_BaseModule_UnionLeaf_Enum"},
		wantMtypes: map[int]*MappedType{
			0: {
				NativeType:        "E_BaseModule_UnionLeaf_Enum",
				IsEnumeratedValue: true,
				ZeroValue:         "0",
			},
		},
	}, {
		name: "union of identityrefs",
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
		inCtxEntry: &yang.Entry{
			Name: "union-leaf",
			Kind: yang.LeafEntry,
			Type: &yang.YangType{
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
				UnionTypes:        nil,
				IsEnumeratedValue: true,
				ZeroValue:         "0",
				DefaultValue:      ygot.String("BaseModule_BaseIdentity_CHIPS"),
			},
		},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enumSet, _, errs := findEnumSet(enumMapFromEntry(tt.inCtxEntry), false, false, false, true, true, true, nil)
			if errs != nil {
				t.Fatal(errs)
			}
			s := newGoGenState(nil, enumSet)

			mtypes := make(map[int]*MappedType)
			ctypes := make(map[string]int)
			ctxEntry := tt.inCtxEntry
			if tt.inNoContext {
				ctxEntry = nil
			}
			if errs := s.goUnionSubTypes(tt.inCtxEntry.Type, ctxEntry, ctypes, mtypes, false, false, true, true, nil); !tt.wantErr && errs != nil {
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

			if diff := cmp.Diff(tt.wantMtypes, mtypes, cmp.AllowUnexported(MappedType{}), cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("mtypes not as expected (-want, +got):\n%s", diff)
			}
		})
	}
}

// TestYangTypeToGoType tests the resolution of a particular YangType to the
// corresponding Go type.
func TestYangTypeToGoType(t *testing.T) {
	tests := []struct {
		name            string
		in              *yang.YangType
		ctx             *yang.Entry
		inEntries       []*yang.Entry
		inEnumEntries   []*yang.Entry // inEnumEntries is used to add more state for findEnumSet to test enum name generation.
		inSkipEnumDedup bool
		inCompressPath  bool
		want            *MappedType
		wantErr         bool
	}{{
		name: "simple lookup resolution",
		in:   &yang.YangType{Kind: yang.Yint32, Name: "int32"},
		want: &MappedType{NativeType: "int32", ZeroValue: "0"},
	}, {
		name: "int32 with default",
		in:   &yang.YangType{Kind: yang.Yint32, Name: "int32", Default: "42"},
		want: &MappedType{NativeType: "int32", ZeroValue: "0", DefaultValue: ygot.String("42")},
	}, {
		name: "decimal64",
		in:   &yang.YangType{Kind: yang.Ydecimal64, Name: "decimal64"},
		want: &MappedType{NativeType: "float64", ZeroValue: "0.0"},
	}, {
		name: "binary lookup resolution",
		in:   &yang.YangType{Kind: yang.Ybinary, Name: "binary"},
		want: &MappedType{NativeType: "Binary", ZeroValue: "nil"},
	}, {
		name: "unknown lookup resolution",
		in:   &yang.YangType{Kind: yang.YinstanceIdentifier, Name: "instanceIdentifier"},
		want: &MappedType{NativeType: "interface{}", ZeroValue: "nil"},
	}, {
		name: "simple empty resolution",
		in:   &yang.YangType{Kind: yang.Yempty, Name: "empty"},
		want: &MappedType{NativeType: "YANGEmpty", ZeroValue: "false"},
	}, {
		name: "simple boolean resolution",
		in:   &yang.YangType{Kind: yang.Ybool, Name: "bool"},
		want: &MappedType{NativeType: "bool", ZeroValue: "false"},
	}, {
		name: "simple int64 resolution",
		in:   &yang.YangType{Kind: yang.Yint64, Name: "int64"},
		want: &MappedType{NativeType: "int64", ZeroValue: "0"},
	}, {
		name: "simple uint8 resolution",
		in:   &yang.YangType{Kind: yang.Yuint8, Name: "uint8"},
		want: &MappedType{NativeType: "uint8", ZeroValue: "0"},
	}, {
		name: "simple uint16 resolution",
		in:   &yang.YangType{Kind: yang.Yuint16, Name: "uint16"},
		want: &MappedType{NativeType: "uint16", ZeroValue: "0"},
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
		ctx: &yang.Entry{
			Name: "leaf",
			Parent: &yang.Entry{
				Name: "container",
				Parent: &yang.Entry{
					Name: "module",
				},
			},
			Type: &yang.YangType{
				Kind: yang.Yunion,
				Type: []*yang.YangType{
					{Kind: yang.Ystring, Name: "string"},
					{Kind: yang.Yint8, Name: "int8"},
				},
			},
		},
		want: &MappedType{
			NativeType: "Module_Container_Leaf_Union",
			UnionTypes: map[string]int{"string": 0, "int8": 1},
			ZeroValue:  "nil",
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
			UnionTypes: map[string]int{"string": 0},
			ZeroValue:  `""`,
		},
	}, {
		name: "derived identityref",
		ctx: &yang.Entry{
			Name: "derived-identityref",
			Type: &yang.YangType{
				Name: "derived-identityref",
				Kind: yang.Yidentityref,
				IdentityBase: &yang.Identity{
					Name:   "base-identity",
					Parent: &yang.Module{Name: "base-module"},
				},
				Base: &yang.Type{
					Name:   "base-identity",
					Parent: &yang.Module{Name: "base-module"},
				},
			},
			Parent: &yang.Entry{Name: "base-module"},
			Node: &yang.Leaf{
				Parent: &yang.Module{Name: "base-module"},
			},
		},
		want: &MappedType{
			NativeType:        "E_BaseModule_DerivedIdentityref",
			IsEnumeratedValue: true,
			ZeroValue:         "0",
		},
	}, {
		name: "derived identityref with default value",
		ctx: &yang.Entry{
			Name: "derived-identityref",
			Type: &yang.YangType{
				Name:    "derived-identityref",
				Kind:    yang.Yidentityref,
				Default: "AARDVARK",
				IdentityBase: &yang.Identity{
					Name:   "base-identity",
					Parent: &yang.Module{Name: "base-module"},
				},
				Base: &yang.Type{
					Name:   "base-identity",
					Parent: &yang.Module{Name: "base-module"},
				},
			},
			Parent: &yang.Entry{Name: "base-module"},
			Node: &yang.Identity{
				Parent: &yang.Module{Name: "base-module"},
			},
		},
		want: &MappedType{
			NativeType:        "E_BaseModule_DerivedIdentityref",
			IsEnumeratedValue: true,
			ZeroValue:         "0",
			DefaultValue:      ygot.String("BaseModule_DerivedIdentityref_AARDVARK"),
		},
	}, {
		name: "enumeration",
		ctx: &yang.Entry{
			Name: "enumeration-leaf",
			Type: &yang.YangType{
				Name: "enumeration",
				Kind: yang.Yenum,
				Enum: &yang.EnumType{},
			},
			Parent: &yang.Entry{Name: "base-module"},
			Node: &yang.Identity{
				Parent: &yang.Module{Name: "base-module"},
			},
		},
		want: &MappedType{
			NativeType:        "E_BaseModule_EnumerationLeaf",
			IsEnumeratedValue: true,
			ZeroValue:         "0",
		},
	}, {
		name: "enumeration with default",
		ctx: &yang.Entry{
			Name: "enumeration-leaf",
			Type: &yang.YangType{
				Name:    "enumeration",
				Kind:    yang.Yenum,
				Enum:    &yang.EnumType{},
				Default: "prefix:BLUE",
			},
			Parent: &yang.Entry{Name: "base-module"},
			Node: &yang.Enum{
				Parent: &yang.Module{Name: "base-module"},
			},
		},
		want: &MappedType{
			NativeType:        "E_BaseModule_EnumerationLeaf",
			IsEnumeratedValue: true,
			ZeroValue:         "0",
			DefaultValue:      ygot.String("BaseModule_EnumerationLeaf_BLUE"),
		},
	}, {
		name: "enumeration in union as the lone type with default",
		ctx: &yang.Entry{
			Name: "union-leaf",
			Kind: yang.LeafEntry,
			Type: &yang.YangType{
				Name: "union",
				Kind: yang.Yunion,
				Type: []*yang.YangType{
					{Kind: yang.Yenum, Enum: &yang.EnumType{}, Name: "enumeration", Default: "prefix:BLUE"},
				},
			},
			Parent: &yang.Entry{Name: "base-module"},
			Node: &yang.Enum{
				Name:   "enum",
				Parent: &yang.Module{Name: "base-module"},
			},
		},
		want: &MappedType{
			NativeType:        "E_BaseModule_UnionLeaf_Enum",
			UnionTypes:        map[string]int{"E_BaseModule_UnionLeaf_Enum": 0},
			IsEnumeratedValue: true,
			ZeroValue:         "0",
			DefaultValue:      ygot.String("BaseModule_UnionLeaf_Enum_BLUE"),
		},
	}, {
		name: "typedef enumeration",
		ctx: &yang.Entry{
			Name: "enumeration-leaf",
			Type: &yang.YangType{
				Name: "derived-enumeration",
				Kind: yang.Yenum,
				Enum: &yang.EnumType{},
				Base: &yang.Type{
					Name:   "enumeration",
					Parent: &yang.Module{Name: "base-module"},
				},
			},
			Node: &yang.Enum{
				Parent: &yang.Module{
					Name: "base-module",
				},
			},
		},
		want: &MappedType{
			NativeType:        "E_BaseModule_DerivedEnumeration",
			IsEnumeratedValue: true,
			ZeroValue:         "0",
		},
	}, {
		name: "typedef union with enumeration as the lone type",
		ctx: &yang.Entry{
			Name: "union-leaf",
			Kind: yang.LeafEntry,
			Type: &yang.YangType{
				Name: "union",
				Kind: yang.Yunion,
				Type: []*yang.YangType{
					{Kind: yang.Yenum, Enum: &yang.EnumType{}, Name: "enumeration"},
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
			NativeType:        "E_BaseModule_UnionLeaf_Enum",
			UnionTypes:        map[string]int{"E_BaseModule_UnionLeaf_Enum": 0},
			IsEnumeratedValue: true,
			ZeroValue:         "0",
		},
	}, {
		name: "typedef enumeration with default",
		ctx: &yang.Entry{
			Name: "enumeration-leaf",
			Type: &yang.YangType{
				Name:    "derived-enumeration",
				Kind:    yang.Yenum,
				Enum:    &yang.EnumType{},
				Default: "FISH",
				Base: &yang.Type{
					Name:   "enumeration",
					Parent: &yang.Module{Name: "base-module"},
				},
			},
			Node: &yang.Enum{
				Parent: &yang.Module{
					Name: "base-module",
				},
			},
		},
		want: &MappedType{
			NativeType:        "E_BaseModule_DerivedEnumeration",
			IsEnumeratedValue: true,
			ZeroValue:         "0",
			DefaultValue:      ygot.String("BaseModule_DerivedEnumeration_FISH"),
		},
	}, {
		name: "identityref",
		ctx: &yang.Entry{
			Name: "identityref",
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
			Node: &yang.Leaf{
				Parent: &yang.Module{
					Name: "test-module",
				},
			},
		},
		want: &MappedType{
			NativeType:        "E_TestModule_BaseIdentity",
			IsEnumeratedValue: true,
			ZeroValue:         "0",
		},
	}, {
		name: "identityref with default",
		ctx: &yang.Entry{
			Name: "identityref",
			Type: &yang.YangType{
				Name: "identityref",
				Kind: yang.Yidentityref,
				IdentityBase: &yang.Identity{
					Name: "base-identity",
					Parent: &yang.Module{
						Name: "test-module",
					},
				},
				Default: "CHIPS",
			},
			Node: &yang.Leaf{
				Parent: &yang.Module{
					Name: "test-module",
				},
			},
		},
		want: &MappedType{
			NativeType:        "E_TestModule_BaseIdentity",
			IsEnumeratedValue: true,
			ZeroValue:         "0",
			DefaultValue:      ygot.String("TestModule_BaseIdentity_CHIPS"),
		},
	}, {
		name: "identityref in union as the lone type with default",
		ctx: &yang.Entry{
			Name: "union-leaf",
			Kind: yang.LeafEntry,
			Type: &yang.YangType{
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
			Parent: &yang.Entry{Name: "base-module"},
			Node: &yang.Leaf{
				Parent: &yang.Module{
					Name: "base-module",
				},
			},
		},
		want: &MappedType{
			NativeType:        "E_BaseModule_BaseIdentity",
			UnionTypes:        map[string]int{"E_BaseModule_BaseIdentity": 0},
			IsEnumeratedValue: true,
			ZeroValue:         "0",
			DefaultValue:      ygot.String("BaseModule_BaseIdentity_CHIPS"),
		},
	}, {
		name: "enumeration with compress paths",
		ctx: &yang.Entry{
			Name: "eleaf",
			Type: &yang.YangType{
				Name: "enumeration",
				Kind: yang.Yenum,
				Enum: &yang.EnumType{},
			},
			Parent: &yang.Entry{
				Name: "config", Parent: &yang.Entry{Name: "container"}},
			Node: &yang.Enum{
				Parent: &yang.Module{Name: "base-module"},
			},
		},
		inCompressPath: true,
		want: &MappedType{
			NativeType:        "E_Container_Eleaf",
			IsEnumeratedValue: true,
			ZeroValue:         "0",
		},
	}, {
		name: "enumeration in submodule",
		ctx: &yang.Entry{
			Name:   "eleaf",
			Type:   &yang.YangType{Name: "enumeration", Kind: yang.Yenum, Enum: &yang.EnumType{}},
			Parent: &yang.Entry{Name: "config", Parent: &yang.Entry{Name: "container"}},
			Node: &yang.Enum{
				Parent: &yang.Module{Name: "submodule", BelongsTo: &yang.BelongsTo{Name: "base-mod"}},
			},
		},
		inCompressPath: true,
		want:           &MappedType{NativeType: "E_Container_Eleaf", IsEnumeratedValue: true, ZeroValue: "0"},
	}, {
		name: "leafref",
		ctx: &yang.Entry{
			Name: "d",
			Parent: &yang.Entry{
				Name: "b",
				Parent: &yang.Entry{
					Name:   "a",
					Parent: &yang.Entry{Name: "module"},
				},
			},
			Type: &yang.YangType{Kind: yang.Yleafref, Name: "leafref", Path: "../c"},
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
		want: &MappedType{NativeType: "uint32", ZeroValue: "0"},
	}, {
		name: "enumeration from grouping used in multiple places - skip deduplication",
		ctx: &yang.Entry{
			Name: "leaf",
			Type: &yang.YangType{Kind: yang.Yenum, Name: "enumeration", Enum: &yang.EnumType{}},
			Parent: &yang.Entry{
				Name: "config",
				Parent: &yang.Entry{
					Name:   "bar",
					Parent: &yang.Entry{Name: "foo-mod"},
				},
			},
			Node: &yang.Leaf{
				Name: "leaf",
				Parent: &yang.Grouping{
					Name: "group",
					Parent: &yang.Module{
						Name: "mod",
					},
				},
			},
		},
		inEnumEntries: []*yang.Entry{{
			Name: "enum-leaf",
			Type: &yang.YangType{
				Name: "enumeration",
				Enum: &yang.EnumType{},
				Kind: yang.Yenum,
			},
			Node: &yang.Leaf{
				Name: "leaf",
				Parent: &yang.Grouping{
					Name: "group",
					Parent: &yang.Module{
						Name: "mod",
					},
				},
			},
			Parent: &yang.Entry{
				Name: "config",
				Parent: &yang.Entry{
					Name:   "container",
					Parent: &yang.Entry{Name: "base-module"},
				},
			},
		}},
		inCompressPath:  true,
		inSkipEnumDedup: true,
		want:            &MappedType{NativeType: "E_Bar_Leaf", IsEnumeratedValue: true, ZeroValue: "0"},
	}, {
		name: "enumeration from grouping used in multiple places - with deduplication",
		ctx: &yang.Entry{
			Name: "leaf",
			Type: &yang.YangType{Kind: yang.Yenum, Name: "enumeration", Enum: &yang.EnumType{}},
			Parent: &yang.Entry{
				Name: "config",
				Parent: &yang.Entry{
					Name:   "bar",
					Parent: &yang.Entry{Name: "foo-mod"},
				},
			},
			Node: &yang.Leaf{
				Name: "leaf",
				Parent: &yang.Grouping{
					Name:   "group",
					Parent: &yang.Module{Name: "mod"},
				},
			},
		},
		inEnumEntries: []*yang.Entry{{
			Name: "enum-leaf",
			Type: &yang.YangType{
				Name: "enumeration",
				Enum: &yang.EnumType{},
				Kind: yang.Yenum,
			},
			Node: &yang.Leaf{
				Name: "leaf",
				Parent: &yang.Grouping{
					Name: "group",
					Parent: &yang.Module{
						Name: "mod",
					},
				},
			},
			Parent: &yang.Entry{
				Name: "config",
				Parent: &yang.Entry{
					Name:   "a-container-lexicographically-earlier",
					Parent: &yang.Entry{Name: "base-module"},
				},
			},
		}},
		inCompressPath: true,
		want:           &MappedType{NativeType: "E_AContainerLexicographicallyEarlier_EnumLeaf", IsEnumeratedValue: true, ZeroValue: "0"},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Populate the type from the entry's type when the
			// entry exists, as the code might make pointer comparisons.
			if tt.ctx != nil {
				if tt.in != nil {
					t.Fatalf("Test error: contextEntry and yangType both specified -- please only specify one of them, as yangType will be populated by contextEntry's Type field.")
				}
				tt.in = tt.ctx.Type
			}

			enumMap := enumMapFromEntries(tt.inEnumEntries)
			addEnumsToEnumMap(tt.ctx, enumMap)
			enumSet, _, errs := findEnumSet(enumMap, tt.inCompressPath, false, tt.inSkipEnumDedup, true, true, true, nil)
			if errs != nil {
				if !tt.wantErr {
					t.Errorf("findEnumSet failed: %v", errs)
				}
				return
			}
			s := newGoGenState(nil, enumSet)

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

			got, err := s.yangTypeToGoType(args, tt.inCompressPath, tt.inSkipEnumDedup, true, true, nil)
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

// TestStructName tests the generation of an element name from a parsed YANG
// hierarchy. It tests both OpenConfig path compression and generation of a
// structure name without such compression.
func TestStructName(t *testing.T) {
	tests := []struct {
		name             string      // name is the name of the test.
		inElement        *yang.Entry // inElement is a mock YANG Entry representing the struct.
		wantCompressed   string      // wantCompressed is the expected name with compression enabled.
		wantUncompressed string      // wantUncompressed is the expected name with compression disabled.
	}{{
		name: "/interfaces/interface/config/description",
		inElement: &yang.Entry{
			Name: "description",
			Parent: &yang.Entry{
				Name: "config",
				Dir:  map[string]*yang.Entry{},
				Parent: &yang.Entry{
					Name:     "interface",
					Dir:      map[string]*yang.Entry{},
					ListAttr: &yang.ListAttr{},
					Parent: &yang.Entry{
						Name: "interfaces",
						Dir: map[string]*yang.Entry{
							"interface": {
								Dir:      map[string]*yang.Entry{},
								ListAttr: &yang.ListAttr{},
							},
						},
						Parent: &yang.Entry{
							Name: "openconfig-interfaces",
							Dir:  map[string]*yang.Entry{},
						},
					},
				},
			},
		},
		wantCompressed:   "Interface_Description",
		wantUncompressed: "OpenconfigInterfaces_Interfaces_Interface_Config_Description",
	}, {
		name: "/interfaces/interface/hold-time/config/up",
		inElement: &yang.Entry{
			Name: "up",
			Parent: &yang.Entry{
				Name: "config",
				Dir:  map[string]*yang.Entry{},
				Parent: &yang.Entry{
					Name: "hold-time",
					Dir:  map[string]*yang.Entry{},
					Parent: &yang.Entry{
						Name:     "interface",
						Dir:      map[string]*yang.Entry{},
						ListAttr: &yang.ListAttr{},
						Parent: &yang.Entry{
							Name: "interfaces",
							Dir: map[string]*yang.Entry{
								"interface": {
									Dir:      map[string]*yang.Entry{},
									ListAttr: &yang.ListAttr{},
								},
							},
							Parent: &yang.Entry{
								Name: "openconfig-interfaces",
								Dir:  map[string]*yang.Entry{},
							},
						},
					},
				},
			},
		},
		wantCompressed:   "Interface_HoldTime_Up",
		wantUncompressed: "OpenconfigInterfaces_Interfaces_Interface_HoldTime_Config_Up",
	}, {
		name: "/interfaces/interface/subinterfaces/subinterface/ipv4/addresses/address/config/ip",
		inElement: &yang.Entry{
			Name: "ip",
			Parent: &yang.Entry{
				Name: "config",
				Dir:  map[string]*yang.Entry{},
				Parent: &yang.Entry{
					Name:     "address",
					ListAttr: &yang.ListAttr{},
					Dir:      map[string]*yang.Entry{},
					Parent: &yang.Entry{
						Name: "addresses",
						Dir: map[string]*yang.Entry{
							"address": {
								Dir:      map[string]*yang.Entry{},
								ListAttr: &yang.ListAttr{},
							},
						},
						Parent: &yang.Entry{
							Name: "ipv4",
							Dir:  map[string]*yang.Entry{},
							Parent: &yang.Entry{
								Name:     "subinterface",
								ListAttr: &yang.ListAttr{},
								Dir:      map[string]*yang.Entry{},
								Parent: &yang.Entry{
									Name: "subinterfaces",
									Dir: map[string]*yang.Entry{
										"subinterface": {
											Name:     "subinterface",
											ListAttr: &yang.ListAttr{},
											Dir:      map[string]*yang.Entry{},
										},
									},
									Parent: &yang.Entry{
										Name:     "interface",
										Dir:      map[string]*yang.Entry{},
										ListAttr: &yang.ListAttr{},
										Parent: &yang.Entry{
											Name: "interfaces",
											Dir: map[string]*yang.Entry{
												"interface": {
													Dir:      map[string]*yang.Entry{},
													ListAttr: &yang.ListAttr{},
												},
											},
											Parent: &yang.Entry{
												Name: "openconfig-interfaces",
												Dir:  map[string]*yang.Entry{},
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
		wantCompressed:   "Interface_Subinterface_Ipv4_Address_Ip",
		wantUncompressed: "OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface_Ipv4_Addresses_Address_Config_Ip",
	}}

	for _, tt := range tests {
		for compress, expected := range map[bool]string{false: tt.wantUncompressed, true: tt.wantCompressed} {
			s := newGoGenState(nil, nil)
			if out := s.goStructName(tt.inElement, compress, false); out != expected {
				t.Errorf("%s (compress: %v): shortName output invalid - got: %s, want: %s", tt.name, compress, out, expected)
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
		inSkipEnumDedup   bool
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
				Base: &yang.Type{
					Name:   "base-identity",
					Parent: &yang.Module{Name: "test-module"},
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
				Base: &yang.Type{
					Name:   "base-identity",
					Parent: &yang.Module{Name: "test-module"},
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
			"/test-module/leaf-one": {NativeType: "E_TestModule_BaseIdentity", IsEnumeratedValue: true, ZeroValue: "0"},
			"/test-module/leaf-two": {NativeType: "E_TestModule_BaseIdentity", IsEnumeratedValue: true, ZeroValue: "0"},
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
				Base: &yang.Type{
					Name:   "enumeration",
					Parent: &yang.Module{Name: "base-module"},
				},
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
				Base: &yang.Type{
					Name:   "enumeration",
					Parent: &yang.Module{Name: "base-module"},
				},
			},
			Node: &yang.Enum{
				Parent: &yang.Module{
					Name: "base-module",
				},
			},
		}},
		wantTypes: map[string]*MappedType{
			"/base-module/leaf-one": {NativeType: "E_BaseModule_DefinedType", IsEnumeratedValue: true, ZeroValue: "0"},
			"/base-module/leaf-two": {NativeType: "E_BaseModule_DefinedType", IsEnumeratedValue: true, ZeroValue: "0"},
		},
	}, {
		name: "enumeration defined in grouping used in multiple places - deduplication enabled",
		inLeaves: []*yang.Entry{{
			Name: "leaf-one",
			Parent: &yang.Entry{
				Name: "base-module",
			},
			Type: &yang.YangType{
				Name: "enumeration",
				Kind: yang.Yenum,
				Enum: &yang.EnumType{},
			},
			Node: &yang.Leaf{
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
				Name: "enumeration",
				Kind: yang.Yenum,
				Enum: &yang.EnumType{},
			},
			Node: &yang.Leaf{
				Parent: &yang.Module{
					Name: "base-module",
				},
			},
		}},
		wantTypes: map[string]*MappedType{
			"/base-module/leaf-one": {NativeType: "E_BaseModule_LeafOne", IsEnumeratedValue: true, ZeroValue: "0"},
			"/base-module/leaf-two": {NativeType: "E_BaseModule_LeafOne", IsEnumeratedValue: true, ZeroValue: "0"},
		},
	}, {
		name: "enumeration defined in grouping used in multiple places - deduplication disabled",
		inLeaves: []*yang.Entry{{
			Name: "leaf-one",
			Parent: &yang.Entry{
				Name: "base-module",
			},
			Type: &yang.YangType{
				Name: "enumeration",
				Kind: yang.Yenum,
				Enum: &yang.EnumType{},
			},
			Node: &yang.Leaf{
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
				Name: "enumeration",
				Kind: yang.Yenum,
				Enum: &yang.EnumType{},
			},
			Node: &yang.Leaf{
				Parent: &yang.Module{
					Name: "base-module",
				},
			},
		}},
		inSkipEnumDedup: true,
		wantTypes: map[string]*MappedType{
			"/base-module/leaf-one": {NativeType: "E_BaseModule_LeafOne", IsEnumeratedValue: true, ZeroValue: "0"},
			"/base-module/leaf-two": {NativeType: "E_BaseModule_LeafTwo", IsEnumeratedValue: true, ZeroValue: "0"},
		},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enumSet, _, errs := findEnumSet(enumMapFromEntries(tt.inLeaves), tt.inCompressOCPaths, false, tt.inSkipEnumDedup, true, true, true, nil)
			if errs != nil {
				t.Fatalf("findEnumSet failed: %v", errs)
			}
			s := newGoGenState(nil, enumSet)

			gotTypes := make(map[string]*MappedType)
			for _, leaf := range tt.inLeaves {
				mtype, err := s.yangTypeToGoType(resolveTypeArgs{yangType: leaf.Type, contextEntry: leaf}, tt.inCompressOCPaths, tt.inSkipEnumDedup, true, true, nil)
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
		})
	}
}
