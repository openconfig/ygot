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
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/genutil"
)

func TestYangTypeToProtoType(t *testing.T) {
	tests := []struct {
		name                   string
		in                     []resolveTypeArgs
		inResolveProtoTypeArgs *resolveProtoTypeArgs
		inEntries              []*yang.Entry
		wantWrapper            *MappedType
		wantScalar             *MappedType
		wantSame               bool
		wantErr                bool
	}{{
		name: "integer types",
		in: []resolveTypeArgs{
			{yangType: &yang.YangType{Kind: yang.Yint8}},
			{yangType: &yang.YangType{Kind: yang.Yint16}},
			{yangType: &yang.YangType{Kind: yang.Yint32}},
			{yangType: &yang.YangType{Kind: yang.Yint64}},
		},
		wantWrapper: &MappedType{NativeType: "ywrapper.IntValue"},
		wantScalar:  &MappedType{NativeType: "sint64"},
	}, {
		name: "unsigned integer types",
		in: []resolveTypeArgs{
			{yangType: &yang.YangType{Kind: yang.Yuint8}},
			{yangType: &yang.YangType{Kind: yang.Yuint16}},
			{yangType: &yang.YangType{Kind: yang.Yuint32}},
			{yangType: &yang.YangType{Kind: yang.Yuint64}},
		},
		wantWrapper: &MappedType{NativeType: "ywrapper.UintValue"},
		wantScalar:  &MappedType{NativeType: "uint64"},
	}, {
		name: "bool types",
		in: []resolveTypeArgs{
			{yangType: &yang.YangType{Kind: yang.Ybool}},
			{yangType: &yang.YangType{Kind: yang.Yempty}},
		},
		wantWrapper: &MappedType{NativeType: "ywrapper.BoolValue"},
		wantScalar:  &MappedType{NativeType: "bool"},
	}, {
		name: "missing leafref path",
		in: []resolveTypeArgs{
			{yangType: &yang.YangType{Kind: yang.Yleafref}},
		},
		wantErr: true,
	}, {
		name: "identityref with no context",
		in: []resolveTypeArgs{
			{yangType: &yang.YangType{Kind: yang.Yidentityref}},
		},
		wantErr: true,
	}, {
		name: "missing leafref path in a union",
		in: []resolveTypeArgs{{
			yangType: &yang.YangType{
				Kind: yang.Yunion,
				Type: []*yang.YangType{{Kind: yang.Yleafref}},
			},
		}},
		wantErr: true,
	}, {
		name:        "string",
		in:          []resolveTypeArgs{{yangType: &yang.YangType{Kind: yang.Ystring}}},
		wantWrapper: &MappedType{NativeType: "ywrapper.StringValue"},
		wantScalar:  &MappedType{NativeType: "string"},
	}, {
		name:        "binary",
		in:          []resolveTypeArgs{{yangType: &yang.YangType{Kind: yang.Ybinary}}},
		wantWrapper: &MappedType{NativeType: "ywrapper.BytesValue"},
		wantScalar:  &MappedType{NativeType: "bytes"},
	}, {
		name:        "decimal64",
		in:          []resolveTypeArgs{{yangType: &yang.YangType{Kind: yang.Ydecimal64}}},
		wantWrapper: &MappedType{NativeType: "ywrapper.Decimal64Value"},
		wantSame:    true,
	}, {
		name: "unmapped types",
		in: []resolveTypeArgs{
			{yangType: &yang.YangType{Kind: yang.Ybits}},
		},
		wantErr: true,
	}, {
		name: "union of string, uint32",
		in: []resolveTypeArgs{
			{
				yangType: &yang.YangType{
					Kind: yang.Yunion,
					Type: []*yang.YangType{
						{Kind: yang.Ystring, Name: "string"},
						{Kind: yang.Yuint32, Name: "uint32"},
					},
				},
			},
		},
		wantWrapper: &MappedType{
			UnionTypes: map[string]MappedUnionSubtype{
				"string": {
					Index: 0,
				},
				"uint64": {
					Index: 1,
				},
			},
		},
		wantSame: true,
	}, {
		name: "union with only strings",
		in: []resolveTypeArgs{{
			yangType: &yang.YangType{
				Kind: yang.Yunion,
				Type: []*yang.YangType{
					{Kind: yang.Ystring, Name: "string"},
					{Kind: yang.Ystring, Name: "string"},
				},
			},
		}},
		wantWrapper: &MappedType{NativeType: "ywrapper.StringValue"},
		wantSame:    true,
	}, {
		name: "union of string, unsupported instance identifier",
		in: []resolveTypeArgs{
			{
				yangType: &yang.YangType{
					Kind: yang.Yunion,
					Type: []*yang.YangType{
						{Kind: yang.Ystring, Name: "string"},
						{Kind: yang.YinstanceIdentifier, Name: "inst-ident"},
					},
				},
			},
		},
		wantErr:  true,
		wantSame: true,
	}, {
		name: "enumeration in union as the lone type with default",
		in: []resolveTypeArgs{{
			contextEntry: &yang.Entry{
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
					Parent: &yang.Module{Name: "base-module"},
				},
			},
		}},
		wantWrapper: &MappedType{
			NativeType:        "UnionLeafEnum",
			IsEnumeratedValue: true,
		},
		wantSame: true,
	}, {
		name: "typedef enumeration in union as the lone type",
		in: []resolveTypeArgs{{
			contextEntry: &yang.Entry{
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
		}},
		wantWrapper: &MappedType{
			NativeType:        "UnionLeafEnum",
			IsEnumeratedValue: true,
		},
		wantSame: true,
	}, {
		name: "derived identityref",
		in: []resolveTypeArgs{{
			contextEntry: &yang.Entry{
				Type: &yang.YangType{
					Name: "derived-identityref",
					IdentityBase: &yang.Identity{
						Name:   "base-identity",
						Parent: &yang.Module{Name: "base-module"},
					},
					Kind: yang.Yidentityref,
					Base: &yang.Type{
						Name:   "base-identity",
						Parent: &yang.Module{Name: "base-module"},
					},
				},
				Node: &yang.Leaf{
					Parent: &yang.Module{Name: "base-module"},
				},
				Parent: &yang.Entry{Name: "base-module"},
			},
		}},
		wantWrapper: &MappedType{
			NativeType:        "basePackage.enumPackage.BaseModuleDerivedIdentityref",
			IsEnumeratedValue: true,
		},
		wantSame: true,
	}, {
		name: "identityref in union as the lone type with default",
		in: []resolveTypeArgs{{
			contextEntry: &yang.Entry{
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
						Base: &yang.Type{
							Name:   "base-identity",
							Parent: &yang.Module{Name: "base-module"},
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
		}},
		wantWrapper: &MappedType{
			NativeType:        "basePackage.enumPackage.BaseModuleBaseIdentity",
			IsEnumeratedValue: true,
		},
		wantSame: true,
	}, {
		name: "enumeration without context",
		in: []resolveTypeArgs{{
			yangType: &yang.YangType{
				Kind: yang.Yenum,
				Name: "enumeration",
			},
		}},
		wantErr: true,
	}, {
		name: "enumeration",
		in: []resolveTypeArgs{{
			contextEntry: &yang.Entry{
				Name: "enumeration-leaf",
				Type: &yang.YangType{
					Name: "enumeration",
					Enum: &yang.EnumType{},
					Kind: yang.Yenum,
				},
				Node: &yang.Enum{
					Parent: &yang.Module{
						Name: "base-module",
					},
				},
				Parent: &yang.Entry{Name: "base-module"},
			},
		}},
		wantWrapper: &MappedType{
			NativeType:        "EnumerationLeaf",
			IsEnumeratedValue: true,
		},
		wantSame: true,
	}, {
		name: "typedef enumeration",
		in: []resolveTypeArgs{{
			contextEntry: &yang.Entry{
				Name: "enumeration-leaf",
				Type: &yang.YangType{
					Name: "derived-enumeration",
					Enum: &yang.EnumType{},
					Kind: yang.Yenum,
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
				Parent: &yang.Entry{Name: "base-module"},
			},
		}},
		wantWrapper: &MappedType{NativeType: "basePackage.enumPackage.BaseModuleDerivedEnumeration", IsEnumeratedValue: true},
		wantSame:    true,
	}, {
		name: "identityref",
		in: []resolveTypeArgs{{
			contextEntry: &yang.Entry{
				Name: "identityref",
				Type: &yang.YangType{
					Name: "identityref",
					IdentityBase: &yang.Identity{
						Name: "base-identity",
						Parent: &yang.Module{
							Name: "test-module",
						},
					},
					Kind: yang.Yidentityref,
				},
				Node: &yang.Leaf{
					Parent: &yang.Module{
						Name: "test-module",
					},
				},
				Parent: &yang.Entry{Name: "test-module"},
			},
		}},
		wantWrapper: &MappedType{NativeType: "basePackage.enumPackage.TestModuleBaseIdentity", IsEnumeratedValue: true},
		wantSame:    true,
	}, {
		name: "identityref with underscore in identity name",
		in: []resolveTypeArgs{{
			contextEntry: &yang.Entry{
				Name: "identityref",
				Type: &yang.YangType{
					Name: "identityref",
					IdentityBase: &yang.Identity{
						Name: "BASE_IDENTITY",
						Parent: &yang.Module{
							Name: "test-module",
						},
					},
					Kind: yang.Yidentityref,
					Base: &yang.Type{
						Name:   "BASE_IDENTITY",
						Parent: &yang.Module{Name: "test-module"},
					},
				},
				Node: &yang.Leaf{
					Parent: &yang.Module{
						Name: "test-module",
					},
				},
				Parent: &yang.Entry{Name: "test-module"},
			},
		}},
		wantWrapper: &MappedType{NativeType: "basePackage.enumPackage.TestModuleBASEIDENTITY", IsEnumeratedValue: true},
		wantSame:    true,
	}, {
		name: "single type union with scalars requested",
		in: []resolveTypeArgs{{
			yangType: &yang.YangType{
				Kind: yang.Yunion,
				Type: []*yang.YangType{{
					Kind:         yang.Ystring,
					Pattern:      []string{"a.*"},
					POSIXPattern: []string{"^a.*$"},
				}, {
					Kind:         yang.Ystring,
					Pattern:      []string{"b.*"},
					POSIXPattern: []string{"^b.*$"},
				}},
			},
		}},
		inResolveProtoTypeArgs: &resolveProtoTypeArgs{
			basePackageName:             "basePackage",
			enumPackageName:             "enumPackage",
			scalarTypeInSingleTypeUnion: true,
		},
		wantWrapper: &MappedType{NativeType: "string"},
		wantSame:    true,
	}, {
		name: "leafref with bad path",
		in: []resolveTypeArgs{{
			contextEntry: &yang.Entry{
				Name: "leaf",
				Type: &yang.YangType{
					Kind: yang.Yleafref,
					Path: "/foo/bar",
				},
			},
		}},
		wantErr: true,
	}, {
		name: "leafref with valid path",
		in: []resolveTypeArgs{{
			contextEntry: &yang.Entry{
				Name: "leaf",
				Type: &yang.YangType{
					Kind: yang.Yleafref,
					Path: "/foo/bar",
				},
			},
		}},
		inEntries: []*yang.Entry{
			{
				Name: "foo",
				Parent: &yang.Entry{
					Name: "module",
				},
				Dir: map[string]*yang.Entry{
					"bar": {
						Name: "bar",
						Type: &yang.YangType{Kind: yang.Ystring},
						Parent: &yang.Entry{
							Name: "foo",
							Parent: &yang.Entry{
								Name: "module",
							},
						},
					},
				},
			},
		},
		wantWrapper: &MappedType{NativeType: "ywrapper.StringValue"},
		wantScalar:  &MappedType{NativeType: "string"},
	}, {
		name: "leafref to leafref",
		in: []resolveTypeArgs{{
			contextEntry: &yang.Entry{
				Name: "leaf",
				Type: &yang.YangType{
					Kind: yang.Yleafref,
					Path: "/foo/bar",
				},
			},
		}},
		inEntries: []*yang.Entry{
			{
				Name: "foo",
				Parent: &yang.Entry{
					Name: "module",
				},
				Dir: map[string]*yang.Entry{
					"bar": {
						Name: "bar",
						Type: &yang.YangType{
							Kind: yang.Yleafref,
							Path: "/foo/baz",
						},
						Parent: &yang.Entry{
							Name: "foo",
							Parent: &yang.Entry{
								Name: "module",
							},
						},
					},
					"baz": {
						Name: "baz",
						Type: &yang.YangType{
							Kind:         yang.Yidentityref,
							IdentityBase: &yang.Identity{Name: "IDENTITY"},
							Base: &yang.Type{
								Name:   "IDENTITY",
								Parent: &yang.Module{Name: "enum-module"},
							},
						},
						Parent: &yang.Entry{
							Name: "foo",
							Parent: &yang.Entry{
								Name: "module",
							},
						},
						Node: &yang.Leaf{
							Name: "baz",
							Parent: &yang.Module{
								Name: "enum-module",
							},
						},
					},
				},
			},
		},
		wantWrapper: &MappedType{NativeType: "basePackage.enumPackage.EnumModule", IsEnumeratedValue: true},
		wantSame:    true,
	}, {
		name: "leafref to union",
		in: []resolveTypeArgs{{
			contextEntry: &yang.Entry{
				Name: "leaf",
				Type: &yang.YangType{
					Kind: yang.Yleafref,
					Path: "/foo/bar",
				},
			},
		}},
		inEntries: []*yang.Entry{
			{
				Name: "foo",
				Parent: &yang.Entry{
					Name: "module",
				},
				Dir: map[string]*yang.Entry{
					"bar": {
						Name: "bar",
						Type: &yang.YangType{
							Kind: yang.Yleafref,
							Path: "/foo/baz",
						},
						Parent: &yang.Entry{
							Name: "foo",
							Parent: &yang.Entry{
								Name: "module",
							},
						},
					},
					"baz": {
						Name: "baz",
						Type: &yang.YangType{
							Kind: yang.Yunion,
							Type: []*yang.YangType{{
								Kind: yang.Ybool,
							}, {
								Kind: yang.Ystring,
							}},
						},
						Parent: &yang.Entry{
							Name: "foo",
							Parent: &yang.Entry{
								Name: "module",
							},
						},
						Node: &yang.Leaf{
							Name: "baz",
							Parent: &yang.Module{
								Name: "enum-module",
							},
						},
					},
				},
			},
		},
		wantWrapper: &MappedType{
			UnionTypes: map[string]MappedUnionSubtype{
				"bool": {
					Index: 0,
				},
				"string": {
					Index: 1,
				},
			},
		},
		wantSame: true,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i := range tt.in {
				st := &tt.in[i]
				// Populate the type from the entry's type when the
				// entry exists, as the code makes pointer comparisons.
				if st.contextEntry != nil {
					if st.yangType != nil {
						t.Fatalf("Test error: contextEntry and yangType both specified -- please only specify one of them, as yangType will be populated by contextEntry's Type field.")
					}
					st.yangType = st.contextEntry.Type
				}
			}

			rpt := resolveProtoTypeArgs{basePackageName: "basePackage", enumPackageName: "enumPackage"}
			if tt.inResolveProtoTypeArgs != nil {
				rpt = *tt.inResolveProtoTypeArgs
			}

			s := NewProtoLangMapper(DefaultBasePackageName, DefaultEnumPackageName)
			// Seed the schema tree with the injected entries, used to ensure leafrefs can
			// be resolved.
			if tt.inEntries != nil {
				if err := s.InjectSchemaTree(tt.inEntries); err != nil {
					t.Fatalf("%s: InjectSchemaTree(%v): got unexpected error, got: %v, want: nil", tt.name, tt.inEntries, err)
				}
			}
			// Seed the enumSet with the injected enum entries,
			// used to ensure that enum names can be resolved.
			enumMap := enumMapFromArgs(tt.in)
			for _, e := range enumMapFromEntries(tt.inEntries) {
				addEnumsToEnumMap(e, enumMap)
			}
			if err := s.InjectEnumSet(enumMap, false, true, false, true, true, true, nil); err != nil {
				if !tt.wantErr {
					t.Errorf("InjectEnumSet failed: %v", err)
				}
				return
			}

			for _, st := range tt.in {
				gotWrapper, err := s.yangTypeToProtoType(st, rpt, IROptions{
					TransformationOptions: TransformationOpts{
						CompressBehaviour:                    genutil.Uncompressed,
						IgnoreShadowSchemaPaths:              false,
						GenerateFakeRoot:                     true,
						ExcludeState:                         false,
						ShortenEnumLeafNames:                 false,
						EnumOrgPrefixesToTrim:                nil,
						UseDefiningModuleForTypedefEnumNames: true,
						EnumerationsUseUnderscores:           false,
					},
					NestedDirectories:                   true,
					AbsoluteMapPaths:                    true,
					AppendEnumSuffixForSimpleUnionEnums: true,
				})
				if (err != nil) != tt.wantErr {
					t.Errorf("%s: yangTypeToProtoType(%v): got unexpected error, got: %v, want error: %v", tt.name, tt.in, err, tt.wantErr)
					continue
				}

				// NOTE: We ignore testing "MappedType.EnumeratedYANGTypeKey" because it is a reference value,
				// and is best tested in an integration test where we can ensure that this value actually points to an enum value in the enum map.
				if diff := cmp.Diff(gotWrapper, tt.wantWrapper, cmpopts.IgnoreFields(MappedType{}, "EnumeratedYANGTypeKey")); diff != "" {
					t.Errorf("%s: yangTypeToProtoType(%v): did not get correct type, diff(-got,+want):\n%s", tt.name, tt.in, diff)
				}

				gotScalar, err := s.yangTypeToProtoScalarType(st, rpt, IROptions{
					TransformationOptions: TransformationOpts{
						CompressBehaviour:                    genutil.Uncompressed,
						IgnoreShadowSchemaPaths:              false,
						GenerateFakeRoot:                     true,
						ExcludeState:                         false,
						ShortenEnumLeafNames:                 false,
						EnumOrgPrefixesToTrim:                nil,
						UseDefiningModuleForTypedefEnumNames: true,
						EnumerationsUseUnderscores:           false,
					},
					NestedDirectories:                   false,
					AbsoluteMapPaths:                    true,
					AppendEnumSuffixForSimpleUnionEnums: true,
				})
				if (err != nil) != tt.wantErr {
					t.Errorf("%s: yangTypeToProtoScalarType(%v, basePackage, enumPackage): got unexpected error: %v", tt.name, tt.in, err)
				}

				wantScalar := tt.wantScalar
				if tt.wantSame {
					wantScalar = tt.wantWrapper
				}
				// NOTE: We ignore testing "MappedType.EnumeratedYANGTypeKey" because it is a reference value,
				// and is best tested in an integration test where we can ensure that this value actually points to an enum value in the enum map.
				if diff := cmp.Diff(gotScalar, wantScalar, cmpopts.IgnoreFields(MappedType{}, "EnumeratedYANGTypeKey")); diff != "" {
					t.Errorf("%s: yangTypeToProtoScalarType(%v): did not get correct type, diff(-got,+want):\n%s", tt.name, tt.in, diff)
				}
			}
		})
	}
}

func TestProtoMsgName(t *testing.T) {
	tests := []struct {
		name                   string
		inEntry                *yang.Entry
		inUniqueProtoMsgNames  map[string]map[string]bool
		inUniqueDirectoryNames map[string]string
		wantCompress           string
		wantUncompress         string
	}{{
		name: "simple message name",
		inEntry: &yang.Entry{
			Name: "msg",
			Parent: &yang.Entry{
				Name: "package",
				Parent: &yang.Entry{
					Name: "module",
				},
			},
		},
		wantCompress:   "Msg",
		wantUncompress: "Msg",
	}, {
		name: "simple message name with compression",
		inEntry: &yang.Entry{
			Name: "msg",
			Parent: &yang.Entry{
				Name: "config",
				Parent: &yang.Entry{
					Name: "container",
					Parent: &yang.Entry{
						Name: "module",
					},
				},
			},
		},
		wantCompress:   "Msg",
		wantUncompress: "Msg",
	}, {
		name: "simple message name with clash when compressing",
		inEntry: &yang.Entry{
			Name: "msg",
			Parent: &yang.Entry{
				Name: "config",
				Kind: yang.DirectoryEntry,
				Dir:  map[string]*yang.Entry{},
				Parent: &yang.Entry{
					Name: "container",
					Parent: &yang.Entry{
						Name: "module",
					},
				},
			},
		},
		inUniqueProtoMsgNames: map[string]map[string]bool{
			"container": {
				"Msg": true,
			},
		},
		wantCompress:   "Msg_",
		wantUncompress: "Msg",
	}, {
		name: "cached name",
		inEntry: &yang.Entry{
			Name: "leaf",
			Parent: &yang.Entry{
				Name: "config",
				Parent: &yang.Entry{
					Name: "container",
					Parent: &yang.Entry{
						Name: "module",
					},
				},
			},
		},
		inUniqueDirectoryNames: map[string]string{"/module/container/config/leaf": "OverriddenName"},
		wantCompress:           "OverriddenName",
		wantUncompress:         "OverriddenName",
	}}

	for _, tt := range tests {
		for compress, want := range map[bool]string{true: tt.wantCompress, false: tt.wantUncompress} {
			s := NewProtoLangMapper(DefaultBasePackageName, DefaultEnumPackageName)

			// Seed the proto message names with some known input.
			if tt.inUniqueProtoMsgNames != nil {
				s.uniqueProtoMsgNames = tt.inUniqueProtoMsgNames
			}

			if tt.inUniqueDirectoryNames != nil {
				s.uniqueDirectoryNames = tt.inUniqueDirectoryNames
			}

			if got := s.protoMsgName(tt.inEntry, compress); got != want {
				t.Errorf("%s: protoMsgName(%v, %v): did not get expected name, got: %v, want: %v", tt.name, tt.inEntry, compress, got, want)
			}
		}
	}
}

func TestProtoPackageName(t *testing.T) {
	tests := []struct {
		name                  string
		inEntry               *yang.Entry
		inDefinedGlobals      map[string]bool
		inUniqueProtoPackages map[string]string
		wantCompress          string
		wantUncompress        string
	}{{
		name: "simple package name",
		inEntry: &yang.Entry{
			Name: "leaf",
			Parent: &yang.Entry{
				Name: "child-container",
				Parent: &yang.Entry{
					Name: "parent-container",
					Kind: yang.DirectoryEntry,
					Dir:  map[string]*yang.Entry{},
					Parent: &yang.Entry{
						Name: "module",
					},
				},
			},
		},
		wantCompress:   "parent_container.child_container",
		wantUncompress: "module.parent_container.child_container",
	}, {
		name: "package name with choice and case",
		inEntry: &yang.Entry{
			Name: "leaf",
			Parent: &yang.Entry{
				Name: "child-container",
				Dir:  map[string]*yang.Entry{},
				Parent: &yang.Entry{
					Name: "case",
					Kind: yang.CaseEntry,
					Dir:  map[string]*yang.Entry{},
					Parent: &yang.Entry{
						Name: "choice",
						Kind: yang.ChoiceEntry,
						Dir:  map[string]*yang.Entry{},
						Parent: &yang.Entry{
							Name: "container",
							Dir:  map[string]*yang.Entry{},
							Parent: &yang.Entry{
								Name: "module",
							},
						},
					},
				},
			},
		},
		wantCompress:   "container.child_container",
		wantUncompress: "module.container.child_container",
	}, {
		name: "clashing names",
		inEntry: &yang.Entry{
			Name: "leaf",
			Parent: &yang.Entry{
				Name: "baz-bat",
				Parent: &yang.Entry{
					Name: "bar",
					Dir:  map[string]*yang.Entry{},
					Parent: &yang.Entry{
						Name: "foo",
						Dir:  map[string]*yang.Entry{},
					},
				},
			},
		},
		inDefinedGlobals: map[string]bool{
			"foo.bar.baz_bat": true, // Clash for uncompressed.
			"bar.baz_bat":     true, // Clash for compressed.
		},
		wantCompress:   "bar.baz_bat_",
		wantUncompress: "foo.bar.baz_bat_",
	}, {
		name: "previously defined parent name",
		inEntry: &yang.Entry{
			Name: "leaf",
			Parent: &yang.Entry{
				Name: "parent",
				Dir:  map[string]*yang.Entry{},
				Kind: yang.DirectoryEntry,
				Parent: &yang.Entry{
					Name: "module",
					Kind: yang.DirectoryEntry,
					Dir:  map[string]*yang.Entry{},
				},
			},
		},
		inUniqueProtoPackages: map[string]string{
			"/module/parent": "explicit.package.name",
		},
		wantCompress:   "explicit.package.name",
		wantUncompress: "explicit.package.name",
	}, {
		name: "list entry within surrounding container with path compression",
		inEntry: &yang.Entry{
			Name:     "list",
			Kind:     yang.DirectoryEntry,
			ListAttr: &yang.ListAttr{},
			Dir:      map[string]*yang.Entry{},
			Parent: &yang.Entry{
				Name: "surrounding-container",
				Kind: yang.DirectoryEntry,
				Parent: &yang.Entry{
					Name: "module",
					Kind: yang.DirectoryEntry,
				},
			},
		},
		wantCompress:   "",
		wantUncompress: "module.surrounding_container",
	}}

	for _, tt := range tests {
		for compress, want := range map[bool]string{true: tt.wantCompress, false: tt.wantUncompress} {
			s := NewProtoLangMapper(DefaultBasePackageName, DefaultEnumPackageName)
			if tt.inDefinedGlobals != nil {
				s.definedGlobals = tt.inDefinedGlobals
			}

			if tt.inUniqueProtoPackages != nil {
				s.uniqueProtoPackages = tt.inUniqueProtoPackages
			}

			if got := s.protobufPackage(tt.inEntry, compress); got != want {
				t.Errorf("%s: protobufPackage(%v, %v): did not get expected package name, got: %v, want: %v", tt.name, tt.inEntry, compress, got, want)
			}
		}
	}
}
