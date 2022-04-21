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
	"encoding/base64"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/kylelemons/godebug/pretty"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/genutil"
	"github.com/openconfig/ygot/ygot"
)

var (
	base64testString        = "forty two"
	base64testStringEncoded = base64.StdEncoding.EncodeToString([]byte(base64testString))
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
				DefaultValue:      ygot.String("prefix:CHIPS"),
			},
		},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enumSet, _, errs := findEnumSet(enumMapFromEntry(tt.inCtxEntry), false, false, false, true, true, true, true, nil)
			if errs != nil {
				t.Fatal(errs)
			}
			s := NewGoLangMapper(true)
			s.SetEnumSet(enumSet)

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
		in:   &yang.YangType{Kind: yang.Ydecimal64, Name: "decimal64", Default: "4.2"},
		want: &MappedType{NativeType: "float64", ZeroValue: "0.0", DefaultValue: ygot.String("4.2")},
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
				Default: "42",
			},
		},
		want: &MappedType{
			NativeType:   "Module_Container_Leaf_Union",
			UnionTypes:   map[string]int{"string": 0, "int8": 1},
			ZeroValue:    "nil",
			DefaultValue: ygot.String("42"),
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
			DefaultValue:      ygot.String("AARDVARK"),
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
			DefaultValue:      ygot.String("prefix:BLUE"),
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
			DefaultValue:      ygot.String("prefix:BLUE"),
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
			DefaultValue:      ygot.String("FISH"),
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
			DefaultValue:      ygot.String("CHIPS"),
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
			DefaultValue:      ygot.String("prefix:CHIPS"),
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
			enumSet, _, errs := findEnumSet(enumMap, tt.inCompressPath, false, tt.inSkipEnumDedup, true, true, true, true, nil)
			if errs != nil {
				if !tt.wantErr {
					t.Errorf("findEnumSet failed: %v", errs)
				}
				return
			}
			s := NewGoLangMapper(true)
			s.SetEnumSet(enumSet)

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
		for compress, expected := range map[genutil.CompressBehaviour]string{genutil.Uncompressed: tt.wantUncompressed, genutil.PreferIntendedConfig: tt.wantCompressed} {
			s := NewGoLangMapper(true)
			if out, err := s.DirectoryName(tt.inElement, compress); err != nil {
				t.Errorf("%s (compress: %v): got unexpected error: %v", tt.name, compress, err)
			} else if out != expected {
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
			enumSet, _, errs := findEnumSet(enumMapFromEntries(tt.inLeaves), tt.inCompressOCPaths, false, tt.inSkipEnumDedup, true, true, true, true, nil)
			if errs != nil {
				t.Fatalf("findEnumSet failed: %v", errs)
			}
			s := NewGoLangMapper(true)
			s.SetEnumSet(enumSet)

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

// TestYangDefaultValueToGo tests the resolution of a particular
// YANG default value to the corresponding representation in Go.
func TestYangDefaultValueToGo(t *testing.T) {
	testEnumType := yang.NewEnumType()
	enumValues := []string{"RED", "BLUE", "RED-BLUE"}
	for _, v := range enumValues {
		testEnumType.SetNext(v)
		if !testEnumType.IsDefined(v) {
			t.Fatalf("%q wasn't added to the test enum type", v)
		}
	}

	tests := []struct {
		name      string
		inType    *yang.YangType
		inValue   string
		inCtx     *yang.Entry
		inEntries []*yang.Entry
		// inEnumEntries is used to add more state for findEnumSet to test enum name generation.
		inEnumEntries   []*yang.Entry
		inSkipEnumDedup bool
		inCompressPath  bool
		want            string
		// wantUnionName is specified for testing the same type wrapped
		// within a union with a de-prioritized string type.
		wantUnionName string
		wantKind      yang.TypeKind
		wantErr       bool
	}{{
		name:          "int8",
		inType:        &yang.YangType{Kind: yang.Yint8},
		inValue:       "-128",
		want:          "-128",
		wantUnionName: "UnionInt8(-128)",
		wantKind:      yang.Yint8,
	}, {
		name:    "int8",
		inType:  &yang.YangType{Kind: yang.Yint8},
		inValue: "-129",
		wantErr: true,
	}, {
		name:          "int16",
		inType:        &yang.YangType{Kind: yang.Yint16},
		inValue:       "-129",
		want:          "-129",
		wantKind:      yang.Yint16,
		wantUnionName: "UnionInt16(-129)",
	}, {
		name:          "int32",
		inType:        &yang.YangType{Kind: yang.Yint32},
		inValue:       "8",
		want:          "8",
		wantKind:      yang.Yint32,
		wantUnionName: "UnionInt32(8)",
	}, {
		name:          "int64",
		inType:        &yang.YangType{Kind: yang.Yint64},
		inValue:       "-8",
		want:          "-8",
		wantKind:      yang.Yint64,
		wantUnionName: "UnionInt64(-8)",
	}, {
		name:          "uint8",
		inType:        &yang.YangType{Kind: yang.Yuint8},
		inValue:       "8",
		want:          "8",
		wantKind:      yang.Yuint8,
		wantUnionName: "UnionUint8(8)",
	}, {
		name:          "uint16",
		inType:        &yang.YangType{Kind: yang.Yuint16},
		inValue:       "8",
		want:          "8",
		wantKind:      yang.Yuint16,
		wantUnionName: "UnionUint16(8)",
	}, {
		name:          "uint32",
		inType:        &yang.YangType{Kind: yang.Yuint32},
		inValue:       "8",
		want:          "8",
		wantKind:      yang.Yuint32,
		wantUnionName: "UnionUint32(8)",
	}, {
		name:          "uint64",
		inType:        &yang.YangType{Kind: yang.Yuint64},
		inValue:       "8",
		want:          "8",
		wantKind:      yang.Yuint64,
		wantUnionName: "UnionUint64(8)",
	}, {
		name:          "decimal64",
		inType:        &yang.YangType{Kind: yang.Ydecimal64},
		inValue:       "3.14",
		want:          "3.14",
		wantKind:      yang.Ydecimal64,
		wantUnionName: "UnionFloat64(3.14)",
	}, {
		name:    "decimal64",
		inType:  &yang.YangType{Kind: yang.Ydecimal64},
		inValue: "21.02.04",
		wantErr: true,
	}, {
		name:          "binary",
		inType:        &yang.YangType{Kind: yang.Ybinary},
		inValue:       base64testStringEncoded,
		want:          `Binary("` + base64testStringEncoded + `")`,
		wantKind:      yang.Ybinary,
		wantUnionName: `Binary("` + base64testStringEncoded + `")`,
	}, {
		name:    "invalid binary",
		inType:  &yang.YangType{Kind: yang.Ybinary},
		inValue: "~~~",
		wantErr: true,
	}, {
		name:          "string",
		inType:        &yang.YangType{Kind: yang.Ystring},
		inValue:       "foo",
		want:          `"foo"`,
		wantKind:      yang.Ystring,
		wantUnionName: `UnionString("foo")`,
	}, {
		name:     "unknown lookup resolution",
		inType:   &yang.YangType{Kind: yang.YinstanceIdentifier},
		inValue:  "foo",
		wantErr:  true,
		wantKind: yang.Ynone,
	}, {
		name:     "empty is not allowed to have a default value",
		inType:   &yang.YangType{Kind: yang.Yempty},
		inValue:  "true",
		wantErr:  true,
		wantKind: yang.Ynone,
	}, {
		name:          "boolean false",
		inType:        &yang.YangType{Kind: yang.Ybool},
		inValue:       "false",
		want:          "false",
		wantKind:      yang.Ybool,
		wantUnionName: "UnionBool(false)",
	}, {
		name:          "boolean true",
		inType:        &yang.YangType{Kind: yang.Ybool},
		inValue:       "true",
		want:          "true",
		wantKind:      yang.Ybool,
		wantUnionName: "UnionBool(true)",
	}, {
		name:    "boolean unknown",
		inType:  &yang.YangType{Kind: yang.Ybool},
		inValue: "yes",
		wantErr: true,
	}, {
		name:    "leafref without valid path",
		inType:  &yang.YangType{Kind: yang.Yleafref},
		inValue: "foo",
		wantErr: true,
	}, {
		name:    "enum without context",
		inType:  &yang.YangType{Kind: yang.Yenum},
		inValue: "foo",
		wantErr: true,
	}, {
		name:    "identityref without context",
		inType:  &yang.YangType{Kind: yang.Yidentityref},
		inValue: "foo",
		wantErr: true,
	}, {
		name: "union with enum without context",
		inType: &yang.YangType{
			Name: "union",
			Kind: yang.Yunion,
			Type: []*yang.YangType{
				{Kind: yang.Yenum, Name: "enumeration"},
			},
		},
		inValue: "foo",
		wantErr: true,
	}, {
		name: "union of string, int32, given an int-compatible value",
		inCtx: &yang.Entry{
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
		inValue:  "42",
		want:     `UnionString("42")`,
		wantKind: yang.Ystring,
	}, {
		name: "union of int32, string, given an int-compatible value",
		inCtx: &yang.Entry{
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
					{Kind: yang.Yint8, Name: "int8"},
					{Kind: yang.Ystring, Name: "string"},
				},
			},
		},
		inValue:  "42",
		want:     "UnionInt8(42)",
		wantKind: yang.Yint8,
	}, {
		name: "derived identityref, with default as the derived value",
		inCtx: &yang.Entry{
			Name: "derived-identityref",
			Type: &yang.YangType{
				Name: "derived-identityref",
				Kind: yang.Yidentityref,
				IdentityBase: &yang.Identity{
					Name:   "base-identity",
					Parent: &yang.Module{Name: "base-module"},
					Values: []*yang.Identity{
						{Name: "DERIVED"},
						{Name: "BASE"},
					},
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
		inValue:  "DERIVED",
		want:     "BaseModule_DerivedIdentityref_DERIVED",
		wantKind: yang.Yidentityref,
	}, {
		name: "derived identityref, with value not found",
		inCtx: &yang.Entry{
			Name: "derived-identityref",
			Type: &yang.YangType{
				Name: "derived-identityref",
				Kind: yang.Yidentityref,
				IdentityBase: &yang.Identity{
					Name:   "base-identity",
					Parent: &yang.Module{Name: "base-module"},
					Values: []*yang.Identity{
						{Name: "FOO"},
						{Name: "BAR"},
					},
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
		inValue:  "BASE",
		wantErr:  true,
		wantKind: yang.Ynone,
	}, {
		name: "derived identityref, with inValue to be sanitised",
		inCtx: &yang.Entry{
			Name: "derived-identityref",
			Type: &yang.YangType{
				Name: "derived-identityref",
				Kind: yang.Yidentityref,
				IdentityBase: &yang.Identity{
					Name:   "base-identity",
					Parent: &yang.Module{Name: "base-module"},
					Values: []*yang.Identity{
						{Name: "FOO-BAR"},
						{Name: "DERIVED-VALUE"},
					},
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
		inValue:  "DERIVED-VALUE",
		want:     "BaseModule_DerivedIdentityref_DERIVED_VALUE",
		wantKind: yang.Yidentityref,
	}, {
		name: "identityref in union with restricted string, with prefix",
		inCtx: &yang.Entry{
			Name: "union-leaf",
			Kind: yang.LeafEntry,
			Type: &yang.YangType{
				Name: "union",
				Kind: yang.Yunion,
				Type: []*yang.YangType{{
					Name:         "Imaginary number",
					Kind:         yang.Ystring,
					POSIXPattern: []string{"^[1-9i]+$"},
				}, {
					Kind:    yang.Yidentityref,
					Name:    "identityref",
					Default: "prefix:CHIPS",
					IdentityBase: &yang.Identity{
						Name: "base-identity",
						Parent: &yang.Module{
							Name: "base-module",
						},
						Values: []*yang.Identity{
							{Name: "FOO"},
							{Name: "BAR"},
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
		inValue:  "oc:BAR",
		want:     "BaseModule_BaseIdentity_BAR",
		wantKind: yang.Yidentityref,
	}, {
		name: "identityref in union with string and binary, but resolves to binary",
		inCtx: &yang.Entry{
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
						Values: []*yang.Identity{
							{Name: "FOO"},
							{Name: "BAR"},
						},
					},
				}, {
					Kind: yang.Ybinary,
				}, {
					Kind: yang.Ystring,
				}},
			},
			Parent: &yang.Entry{Name: "base-module"},
			Node: &yang.Leaf{
				Parent: &yang.Module{
					Name: "base-module",
				},
			},
		},
		inValue:       base64testStringEncoded,
		wantKind:      yang.Ybinary,
		want:          `Binary("` + base64testStringEncoded + `")`,
		wantUnionName: `Binary("` + base64testStringEncoded + `")`,
	}, {
		name: "identityref in union with string and binary, but resolves to string due to restrictions",
		inCtx: &yang.Entry{
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
						Values: []*yang.Identity{
							{Name: "FOO"},
							{Name: "BAR"},
						},
					},
				}, {
					Kind:   yang.Ybinary,
					Length: yang.YangRange{yang.YRange{Min: yang.FromInt(2), Max: yang.FromInt(5)}},
				}, {
					Kind: yang.Ystring,
				}},
			},
			Parent: &yang.Entry{Name: "base-module"},
			Node: &yang.Leaf{
				Parent: &yang.Module{
					Name: "base-module",
				},
			},
		},
		inValue:       base64testStringEncoded,
		wantKind:      yang.Ystring,
		want:          `UnionString("` + base64testStringEncoded + `")`,
		wantUnionName: `UnionString("` + base64testStringEncoded + `")`,
	}, {
		name: "enumeration",
		inCtx: &yang.Entry{
			Name: "enumeration-leaf",
			Type: &yang.YangType{
				Name: "enumeration",
				Kind: yang.Yenum,
				Enum: testEnumType,
			},
			Parent: &yang.Entry{Name: "base-module"},
			Node: &yang.Identity{
				Parent: &yang.Module{Name: "base-module"},
			},
		},
		inValue:  "BLUE",
		want:     "BaseModule_EnumerationLeaf_BLUE",
		wantKind: yang.Yenum,
	}, {
		name: "enumeration not found",
		inCtx: &yang.Entry{
			Name: "enumeration-leaf",
			Type: &yang.YangType{
				Name: "enumeration",
				Kind: yang.Yenum,
				Enum: testEnumType,
			},
			Parent: &yang.Entry{Name: "base-module"},
			Node: &yang.Identity{
				Parent: &yang.Module{Name: "base-module"},
			},
		},
		inValue:  "GREEN",
		wantErr:  true,
		wantKind: yang.Ynone,
	}, {
		name: "enumeration, with inValue to be sanitised",
		inCtx: &yang.Entry{
			Name: "enumeration-leaf",
			Type: &yang.YangType{
				Name: "enumeration",
				Kind: yang.Yenum,
				Enum: testEnumType,
			},
			Parent: &yang.Entry{Name: "base-module"},
			Node: &yang.Identity{
				Parent: &yang.Module{Name: "base-module"},
			},
		},
		inValue:  "RED-BLUE",
		want:     "BaseModule_EnumerationLeaf_RED_BLUE",
		wantKind: yang.Yenum,
	}, {
		name: "enumeration in union with string as the second union type",
		inCtx: &yang.Entry{
			Name: "union-leaf",
			Kind: yang.LeafEntry,
			Type: &yang.YangType{
				Name: "union",
				Kind: yang.Yunion,
				Type: []*yang.YangType{{
					Name: "enumeration",
					Kind: yang.Yenum,
					Enum: testEnumType,
				}, {
					Kind: yang.Ystring,
				}},
			},
			Parent: &yang.Entry{Name: "base-module"},
			Node: &yang.Enum{
				Name:   "enum",
				Parent: &yang.Module{Name: "base-module"},
			},
		},
		inValue:  "BLUE",
		want:     "BaseModule_UnionLeaf_Enum_BLUE",
		wantKind: yang.Yenum,
	}, {
		name: "enumeration in union with string as the second union type, with inValue to be sanitised",
		inCtx: &yang.Entry{
			Name: "union-leaf",
			Kind: yang.LeafEntry,
			Type: &yang.YangType{
				Name: "union",
				Kind: yang.Yunion,
				Type: []*yang.YangType{{
					Name: "enumeration",
					Kind: yang.Yenum,
					Enum: testEnumType,
				}, {
					Kind: yang.Ystring,
				}},
			},
			Parent: &yang.Entry{Name: "base-module"},
			Node: &yang.Enum{
				Name:   "enum",
				Parent: &yang.Module{Name: "base-module"},
			},
		},
		inValue:  "RED-BLUE",
		want:     "BaseModule_UnionLeaf_Enum_RED_BLUE",
		wantKind: yang.Yenum,
	}, {
		name: "enumeration in union with string as the first union type, input matches string restrictions",
		inCtx: &yang.Entry{
			Name: "union-leaf",
			Kind: yang.LeafEntry,
			Type: &yang.YangType{
				Name: "union",
				Kind: yang.Yunion,
				Type: []*yang.YangType{{
					Kind:         yang.Ystring,
					POSIXPattern: []string{"^[A-Z]+$"},
					Length:       yang.YangRange{yang.YRange{Min: yang.FromInt(2), Max: yang.FromInt(10)}},
				}, {
					Name: "enumeration",
					Kind: yang.Yenum,
					Enum: testEnumType,
				}},
			},
			Parent: &yang.Entry{Name: "base-module"},
			Node: &yang.Enum{
				Name:   "enum",
				Parent: &yang.Module{Name: "base-module"},
			},
		},
		inValue:  "BLUE",
		want:     `UnionString("BLUE")`,
		wantKind: yang.Ystring,
	}, {
		name: "enumeration in union with string as the first union type, input doesn't match pattern restriction",
		inCtx: &yang.Entry{
			Name: "union-leaf",
			Kind: yang.LeafEntry,
			Type: &yang.YangType{
				Name: "union",
				Kind: yang.Yunion,
				Type: []*yang.YangType{{
					Kind:         yang.Ystring,
					POSIXPattern: []string{"^[a-z]+$"},
					Length:       yang.YangRange{yang.YRange{Min: yang.FromInt(2), Max: yang.FromInt(10)}},
				}, {
					Name: "enumeration",
					Kind: yang.Yenum,
					Enum: testEnumType,
				}},
			},
			Parent: &yang.Entry{Name: "base-module"},
			Node: &yang.Enum{
				Name:   "enum",
				Parent: &yang.Module{Name: "base-module"},
			},
		},
		inValue:  "BLUE",
		want:     "BaseModule_UnionLeaf_Enum_BLUE",
		wantKind: yang.Yenum,
	}, {
		name: "enumeration in union with string as the first union type, input doesn't match length restriction",
		inCtx: &yang.Entry{
			Name: "union-leaf",
			Kind: yang.LeafEntry,
			Type: &yang.YangType{
				Name: "union",
				Kind: yang.Yunion,
				Type: []*yang.YangType{{
					Kind:         yang.Ystring,
					POSIXPattern: []string{"^[A-Z]+$"},
					Length:       yang.YangRange{yang.YRange{Min: yang.FromInt(10), Max: yang.FromInt(20)}},
				}, {
					Name: "enumeration",
					Kind: yang.Yenum,
					Enum: testEnumType,
				}},
			},
			Parent: &yang.Entry{Name: "base-module"},
			Node: &yang.Enum{
				Name:   "enum",
				Parent: &yang.Module{Name: "base-module"},
			},
		},
		inValue:  "BLUE",
		want:     "BaseModule_UnionLeaf_Enum_BLUE",
		wantKind: yang.Yenum,
	}, {
		name: "typedef enumeration",
		inCtx: &yang.Entry{
			Name: "enumeration-leaf",
			Type: &yang.YangType{
				Name: "derived-enumeration",
				Kind: yang.Yenum,
				Enum: testEnumType,
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
		inValue:  "RED",
		want:     "BaseModule_DerivedEnumeration_RED",
		wantKind: yang.Yenum,
	}, {
		name: "typedef enumeration not found",
		inCtx: &yang.Entry{
			Name: "enumeration-leaf",
			Type: &yang.YangType{
				Name: "derived-enumeration",
				Kind: yang.Yenum,
				Enum: testEnumType,
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
		inValue: "YELLOW",
		wantErr: true,
	}, {
		name: "union with decimal, int, uint, and string, resolving to string due to restrictions",
		inCtx: &yang.Entry{
			Name: "union-leaf",
			Kind: yang.LeafEntry,
			Type: &yang.YangType{
				Name: "union",
				Kind: yang.Yunion,
				Type: []*yang.YangType{{
					Kind: yang.Ydecimal64,
					Range: yang.YangRange{
						yang.YRange{Min: yang.FromFloat(-10.1), Max: yang.FromFloat(-5.1)},
						yang.YRange{Min: yang.FromFloat(-3.1), Max: yang.FromFloat(-1.1)},
					},
				}, {
					Kind: yang.Yint64,
					Range: yang.YangRange{
						yang.YRange{Min: yang.FromInt(-10), Max: yang.FromFloat(5)},
						yang.YRange{Min: yang.FromFloat(10), Max: yang.FromFloat(15)},
					},
				}, {
					Kind: yang.Yuint32,
					Range: yang.YangRange{
						yang.YRange{Min: yang.FromInt(7), Max: yang.FromFloat(8)},
						yang.YRange{Min: yang.FromFloat(10), Max: yang.FromFloat(15)},
					},
				}, {
					Kind: yang.Ystring,
				}},
			},
			Parent: &yang.Entry{Name: "base-module"},
			Node: &yang.Leaf{
				Parent: &yang.Module{
					Name: "base-module",
				},
			},
		},
		inValue:       "6",
		wantKind:      yang.Ystring,
		want:          `UnionString("6")`,
		wantUnionName: `UnionString("6")`,
	}, {
		name: "union with decimal, int, uint, and string, resolving to uint due to restrictions",
		inCtx: &yang.Entry{
			Name: "union-leaf",
			Kind: yang.LeafEntry,
			Type: &yang.YangType{
				Name: "union",
				Kind: yang.Yunion,
				Type: []*yang.YangType{{
					Kind: yang.Ydecimal64,
					Range: yang.YangRange{
						yang.YRange{Min: yang.FromFloat(-10.1), Max: yang.FromFloat(-5.1)},
						yang.YRange{Min: yang.FromFloat(-3.1), Max: yang.FromFloat(-1.1)},
					},
				}, {
					Kind: yang.Yint64,
					Range: yang.YangRange{
						yang.YRange{Min: yang.FromInt(-10), Max: yang.FromFloat(5)},
						yang.YRange{Min: yang.FromFloat(10), Max: yang.FromFloat(15)},
					},
				}, {
					Kind: yang.Yuint32,
					Range: yang.YangRange{
						yang.YRange{Min: yang.FromInt(7), Max: yang.FromFloat(8)},
						yang.YRange{Min: yang.FromFloat(10), Max: yang.FromFloat(15)},
					},
				}, {
					Kind: yang.Ystring,
				}},
			},
			Parent: &yang.Entry{Name: "base-module"},
			Node: &yang.Leaf{
				Parent: &yang.Module{
					Name: "base-module",
				},
			},
		},
		inValue:       "7",
		wantKind:      yang.Yuint32,
		want:          `UnionUint32(7)`,
		wantUnionName: `UnionUint32(7)`,
	}, {
		name: "union with decimal, int, uint, and string, resolving to int due to restrictions",
		inCtx: &yang.Entry{
			Name: "union-leaf",
			Kind: yang.LeafEntry,
			Type: &yang.YangType{
				Name: "union",
				Kind: yang.Yunion,
				Type: []*yang.YangType{{
					Kind: yang.Ydecimal64,
					Range: yang.YangRange{
						yang.YRange{Min: yang.FromFloat(-10.1), Max: yang.FromFloat(-5.1)},
						yang.YRange{Min: yang.FromFloat(-3.1), Max: yang.FromFloat(-1.1)},
					},
				}, {
					Kind: yang.Yint64,
					Range: yang.YangRange{
						yang.YRange{Min: yang.FromInt(-10), Max: yang.FromFloat(5)},
						yang.YRange{Min: yang.FromFloat(10), Max: yang.FromFloat(15)},
					},
				}, {
					Kind: yang.Yuint32,
					Range: yang.YangRange{
						yang.YRange{Min: yang.FromInt(7), Max: yang.FromFloat(8)},
						yang.YRange{Min: yang.FromFloat(10), Max: yang.FromFloat(15)},
					},
				}, {
					Kind: yang.Ystring,
				}},
			},
			Parent: &yang.Entry{Name: "base-module"},
			Node: &yang.Leaf{
				Parent: &yang.Module{
					Name: "base-module",
				},
			},
		},
		inValue:       "12",
		wantKind:      yang.Yint64,
		want:          `UnionInt64(12)`,
		wantUnionName: `UnionInt64(12)`,
	}, {
		name: "union with decimal, int, uint, and string, resolving to decimal",
		inCtx: &yang.Entry{
			Name: "union-leaf",
			Kind: yang.LeafEntry,
			Type: &yang.YangType{
				Name: "union",
				Kind: yang.Yunion,
				Type: []*yang.YangType{{
					Kind: yang.Ydecimal64,
					Range: yang.YangRange{
						yang.YRange{Min: yang.FromFloat(-10.1), Max: yang.FromFloat(-5.1)},
						yang.YRange{Min: yang.FromFloat(-3.1), Max: yang.FromFloat(-1.1)},
					},
				}, {
					Kind: yang.Yint64,
					Range: yang.YangRange{
						yang.YRange{Min: yang.FromInt(-10), Max: yang.FromFloat(5)},
						yang.YRange{Min: yang.FromFloat(10), Max: yang.FromFloat(15)},
					},
				}, {
					Kind: yang.Yuint32,
					Range: yang.YangRange{
						yang.YRange{Min: yang.FromInt(7), Max: yang.FromFloat(8)},
						yang.YRange{Min: yang.FromFloat(10), Max: yang.FromFloat(15)},
					},
				}, {
					Kind: yang.Ystring,
				}},
			},
			Parent: &yang.Entry{Name: "base-module"},
			Node: &yang.Leaf{
				Parent: &yang.Module{
					Name: "base-module",
				},
			},
		},
		inValue:       "-6",
		wantKind:      yang.Ydecimal64,
		want:          `UnionFloat64(-6)`,
		wantUnionName: `UnionFloat64(-6)`,
	}, {
		name: "enumeration with compress paths",
		inCtx: &yang.Entry{
			Name: "eleaf",
			Type: &yang.YangType{
				Name: "enumeration",
				Kind: yang.Yenum,
				Enum: testEnumType,
			},
			Parent: &yang.Entry{
				Name: "config", Parent: &yang.Entry{Name: "container"}},
			Node: &yang.Enum{
				Parent: &yang.Module{Name: "base-module"},
			},
		},
		inCompressPath: true,
		inValue:        "RED",
		want:           "Container_Eleaf_RED",
		wantKind:       yang.Yenum,
	}, {
		name: "enumeration with compress paths and inValue to be sanitised",
		inCtx: &yang.Entry{
			Name: "eleaf",
			Type: &yang.YangType{
				Name: "enumeration",
				Kind: yang.Yenum,
				Enum: testEnumType,
			},
			Parent: &yang.Entry{
				Name: "config", Parent: &yang.Entry{Name: "container"}},
			Node: &yang.Enum{
				Parent: &yang.Module{Name: "base-module"},
			},
		},
		inCompressPath: true,
		inValue:        "RED-BLUE",
		want:           "Container_Eleaf_RED_BLUE",
		wantKind:       yang.Yenum,
	}, {
		name: "leafref",
		inCtx: &yang.Entry{
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
		inValue:       "42",
		want:          "42",
		wantKind:      yang.Yuint32,
		wantUnionName: "UnionUint32(42)",
	}, {
		name: "enumeration from grouping used in multiple places - skip deduplication",
		inCtx: &yang.Entry{
			Name: "leaf",
			Type: &yang.YangType{Kind: yang.Yenum, Name: "enumeration", Enum: testEnumType},
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
				Enum: testEnumType,
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
		inValue:         "BLUE",
		want:            "Bar_Leaf_BLUE",
		wantKind:        yang.Yenum,
	}}

	for _, tt := range tests {
		for _, unionRun := range []bool{false, true} {
			// --- Setup ---
			if unionRun && tt.wantUnionName == "" {
				continue
			}
			if unionRun {
				tt.name += "_unionRun"
			}
			// Populate the type from the entry's type when the
			// entry exists, as the code might make pointer comparisons.
			if tt.inCtx != nil {
				if tt.inType != nil {
					t.Fatalf("Test error: contextEntry and yangType both specified -- please only specify one of them, as yangType will be populated by contextEntry's Type field.")
				}
				tt.inType = tt.inCtx.Type
			}
			if unionRun {
				// Wrap type inside a union type.
				tt.inType = &yang.YangType{
					Kind: yang.Yunion,
					Type: []*yang.YangType{
						tt.inType,
						{Kind: yang.Ystring},
					},
				}
				if tt.inCtx != nil {
					tt.inCtx.Type = tt.inType
				}
				tt.want = tt.wantUnionName
			}

			// --- Test ---
			t.Run(tt.name, func(t *testing.T) {
				enumMap := enumMapFromEntries(tt.inEnumEntries)
				addEnumsToEnumMap(tt.inCtx, enumMap)
				enumSet, _, errs := findEnumSet(enumMap, tt.inCompressPath, false, tt.inSkipEnumDedup, true, true, true, true, nil)
				if errs != nil {
					if !tt.wantErr {
						t.Errorf("findEnumSet failed: %v", errs)
					}
					return
				}
				s := NewGoLangMapper(true)
				s.SetEnumSet(enumSet)

				if tt.inEntries != nil {
					st, err := buildSchemaTree(tt.inEntries)
					if err != nil {
						t.Fatalf("buildSchemaTree(%v): could not build schema tree: %v", tt.inEntries, err)
					}
					s.schematree = st
				}

				args := resolveTypeArgs{
					yangType:     tt.inType,
					contextEntry: tt.inCtx,
				}

				got, gotKind, err := s.yangDefaultValueToGo(tt.inValue, args, false, tt.inCompressPath, tt.inSkipEnumDedup, true, true, nil)
				if tt.wantErr && err == nil {
					t.Fatalf("did not get expected error (%v)", got)
				} else if !tt.wantErr && err != nil {
					t.Fatalf("error returned when mapping default value: %v", err)
				}

				if diff := cmp.Diff(got, tt.want); diff != "" {
					t.Errorf("did not get expected default value, diff(-got,+want):\n%s", diff)
				}

				if gotKind != tt.wantKind {
					t.Errorf("got kind %v, want %v", gotKind, tt.wantKind)
				}
			})

			// --- Teardown ---
			if tt.inCtx != nil {
				tt.inType = nil
			}
		}
	}

	// singletonUnionTests tests default value generation for singleton
	// unions, that is, a union containing only a single type. An example
	// might be multiple strings with different pattern restrictions being
	// unioned. Since singleton unions are reduced to the singleton type,
	// it means that the output shouldn't use the union wrapper types.
	singletonUnionTests := []struct {
		name      string
		inType    *yang.YangType
		inValue   string
		inCtx     *yang.Entry
		inEntries []*yang.Entry
		// inEnumEntries is used to add more state for findEnumSet to test enum name generation.
		inEnumEntries   []*yang.Entry
		inSkipEnumDedup bool
		inCompressPath  bool
		want            string
		wantKind        yang.TypeKind
		wantErr         bool
	}{{
		name:     "int8",
		inType:   &yang.YangType{Kind: yang.Yint8},
		inValue:  "-128",
		want:     "-128",
		wantKind: yang.Yint8,
	}, {
		name:    "int8",
		inType:  &yang.YangType{Kind: yang.Yint8},
		inValue: "-129",
		wantErr: true,
	}, {
		name:     "int16",
		inType:   &yang.YangType{Kind: yang.Yint16},
		inValue:  "-129",
		want:     "-129",
		wantKind: yang.Yint16,
	}, {
		name:     "int32",
		inType:   &yang.YangType{Kind: yang.Yint32},
		inValue:  "8",
		want:     "8",
		wantKind: yang.Yint32,
	}, {
		name:     "int64",
		inType:   &yang.YangType{Kind: yang.Yint64},
		inValue:  "-8",
		want:     "-8",
		wantKind: yang.Yint64,
	}, {
		name:     "uint8",
		inType:   &yang.YangType{Kind: yang.Yuint8},
		inValue:  "8",
		want:     "8",
		wantKind: yang.Yuint8,
	}, {
		name:     "uint16",
		inType:   &yang.YangType{Kind: yang.Yuint16},
		inValue:  "8",
		want:     "8",
		wantKind: yang.Yuint16,
	}, {
		name:     "uint32",
		inType:   &yang.YangType{Kind: yang.Yuint32},
		inValue:  "8",
		want:     "8",
		wantKind: yang.Yuint32,
	}, {
		name:     "uint64",
		inType:   &yang.YangType{Kind: yang.Yuint64},
		inValue:  "8",
		want:     "8",
		wantKind: yang.Yuint64,
	}, {
		name:     "decimal64",
		inType:   &yang.YangType{Kind: yang.Ydecimal64},
		inValue:  "3.14",
		want:     "3.14",
		wantKind: yang.Ydecimal64,
	}, {
		name:    "decimal64",
		inType:  &yang.YangType{Kind: yang.Ydecimal64},
		inValue: "21.02.04",
		wantErr: true,
	}, {
		name:     "binary",
		inType:   &yang.YangType{Kind: yang.Ybinary},
		inValue:  base64testStringEncoded,
		want:     `Binary("` + base64testStringEncoded + `")`,
		wantKind: yang.Ybinary,
	}, {
		name:    "invalid binary",
		inType:  &yang.YangType{Kind: yang.Ybinary},
		inValue: "~~~",
		wantErr: true,
	}, {
		name:     "string",
		inType:   &yang.YangType{Kind: yang.Ystring},
		inValue:  "foo",
		want:     `"foo"`,
		wantKind: yang.Ystring,
	}, {
		name:     "unknown lookup resolution",
		inType:   &yang.YangType{Kind: yang.YinstanceIdentifier},
		inValue:  "foo",
		wantErr:  true,
		wantKind: yang.Ynone,
	}, {
		name:     "empty is not allowed to have a default value",
		inType:   &yang.YangType{Kind: yang.Yempty},
		inValue:  "true",
		wantErr:  true,
		wantKind: yang.Ynone,
	}, {
		name:     "boolean false",
		inType:   &yang.YangType{Kind: yang.Ybool},
		inValue:  "false",
		want:     "false",
		wantKind: yang.Ybool,
	}, {
		name:     "boolean true",
		inType:   &yang.YangType{Kind: yang.Ybool},
		inValue:  "true",
		want:     "true",
		wantKind: yang.Ybool,
	}, {
		name:    "boolean unknown",
		inType:  &yang.YangType{Kind: yang.Ybool},
		inValue: "yes",
		wantErr: true,
	}, {
		name:    "leafref without valid path",
		inType:  &yang.YangType{Kind: yang.Yleafref},
		inValue: "foo",
		wantErr: true,
	}, {
		name:    "enum without context",
		inType:  &yang.YangType{Kind: yang.Yenum},
		inValue: "foo",
		wantErr: true,
	}, {
		name:    "identityref without context",
		inType:  &yang.YangType{Kind: yang.Yidentityref},
		inValue: "foo",
		wantErr: true,
	}, {
		name: "string-only union",
		inType: &yang.YangType{
			Kind: yang.Yunion,
			Type: []*yang.YangType{
				{Kind: yang.Ystring, Name: "string"},
				{Kind: yang.Ystring, Name: "string"},
			},
		},
		inValue:  "42",
		want:     `"42"`,
		wantKind: yang.Ystring,
	}, {
		name: "identityref in union as the lone type with default",
		inCtx: &yang.Entry{
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
						Values: []*yang.Identity{
							{Name: "FOO"},
							{Name: "BAR"},
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
		inValue:  "BAR",
		want:     "BaseModule_BaseIdentity_BAR",
		wantKind: yang.Yidentityref,
	}, {
		name: "enumeration in union as the lone type, with prefix",
		inCtx: &yang.Entry{
			Name: "union-leaf",
			Kind: yang.LeafEntry,
			Type: &yang.YangType{
				Name: "union",
				Kind: yang.Yunion,
				Type: []*yang.YangType{
					{
						Name: "enumeration",
						Kind: yang.Yenum,
						Enum: testEnumType,
					},
				},
			},
			Parent: &yang.Entry{Name: "base-module"},
			Node: &yang.Enum{
				Name:   "enum",
				Parent: &yang.Module{Name: "base-module"},
			},
		},
		inValue:  "oc:BLUE",
		want:     "BaseModule_UnionLeaf_Enum_BLUE",
		wantKind: yang.Yenum,
	}, {
		name: "typedef union with enumeration as the lone type, with prefix",
		inCtx: &yang.Entry{
			Name: "union-leaf",
			Kind: yang.LeafEntry,
			Type: &yang.YangType{
				Name: "union",
				Kind: yang.Yunion,
				Type: []*yang.YangType{
					{
						Name: "enumeration",
						Kind: yang.Yenum,
						Enum: testEnumType,
					},
				},
			},
			Parent: &yang.Entry{Name: "base-module"},
			Node: &yang.Enum{
				Parent: &yang.Module{
					Name: "base-module",
				},
			},
		},
		inValue:  "oc:RED",
		want:     "BaseModule_UnionLeaf_Enum_RED",
		wantKind: yang.Yenum,
	}, {
		name: "leafref",
		inCtx: &yang.Entry{
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
		inValue:  "42",
		want:     "42",
		wantKind: yang.Yuint32,
	}}

	for _, tt := range singletonUnionTests {
		// --- Setup ---
		// Populate the type from the entry's type when the
		// entry exists, as the code might make pointer comparisons.
		if tt.inCtx != nil {
			if tt.inType != nil {
				t.Fatalf("Test error: contextEntry and yangType both specified -- please only specify one of them, as yangType will be populated by contextEntry's Type field.")
			}
			tt.inType = tt.inCtx.Type
		}

		// Wrap type inside a union type as a singleton if it's not
		// already a union.
		if tt.inType.Kind != yang.Yunion {
			tt.inType = &yang.YangType{
				Kind: yang.Yunion,
				Type: []*yang.YangType{
					tt.inType,
				},
			}
			if tt.inCtx != nil {
				tt.inCtx.Type = tt.inType
			}
		}

		// --- Test ---
		t.Run("singleton union "+tt.name, func(t *testing.T) {
			enumMap := enumMapFromEntries(tt.inEnumEntries)
			addEnumsToEnumMap(tt.inCtx, enumMap)
			enumSet, _, errs := findEnumSet(enumMap, tt.inCompressPath, false, tt.inSkipEnumDedup, true, true, true, true, nil)
			if errs != nil {
				if !tt.wantErr {
					t.Errorf("findEnumSet failed: %v", errs)
				}
				return
			}
			s := NewGoLangMapper(true)
			s.SetEnumSet(enumSet)

			if tt.inEntries != nil {
				st, err := buildSchemaTree(tt.inEntries)
				if err != nil {
					t.Fatalf("buildSchemaTree(%v): could not build schema tree: %v", tt.inEntries, err)
				}
				s.schematree = st
			}

			args := resolveTypeArgs{
				yangType:     tt.inType,
				contextEntry: tt.inCtx,
			}

			got, gotKind, err := s.yangDefaultValueToGo(tt.inValue, args, true, tt.inCompressPath, tt.inSkipEnumDedup, true, true, nil)
			if tt.wantErr && err == nil {
				t.Fatalf("did not get expected error (%v)", got)
			} else if !tt.wantErr && err != nil {
				t.Fatalf("error returned when mapping default value: %v", err)
			}

			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("did not get expected default value, diff(-got,+want):\n%s", diff)
			}

			if gotKind != tt.wantKind {
				t.Errorf("got kind %v, want %v", gotKind, tt.wantKind)
			}
		})

		// --- Teardown ---
		if tt.inCtx != nil {
			tt.inType = nil
		}
	}
}
