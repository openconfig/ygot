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
	"encoding/json"
	"testing"

	"github.com/kylelemons/godebug/pretty"
	"github.com/openconfig/goyang/pkg/yang"
)

// TestFindEnumSet tests the findEnumSet function, ensuring that it performs
// deduplication of re-used identities, and re-used typedefs. For inline
// definitions, the enumerations should be duplicated. Tests are performed with
// CompressOCPaths set to both true and false.
func TestFindEnumSet(t *testing.T) {
	tests := []struct {
		name             string
		in               map[string]*yang.Entry
		wantCompressed   map[string]*yangGoEnum
		wantUncompressed map[string]*yangGoEnum
		wantSame         bool // Whether to expect same compressed/uncompressed output
		wantErr          bool
	}{{
		name: "simple identityref",
		in: map[string]*yang.Entry{
			"/container/config/identityref-leaf": {
				Name: "identityref-leaf",
				Type: &yang.YangType{
					Name: "identityref",
					IdentityBase: &yang.Identity{
						Name: "base-identity",
						Parent: &yang.Module{
							Name: "test-module",
						},
					},
				},
			},
			"/container/state/identityref-leaf": {
				Name: "identityref-leaf",
				Type: &yang.YangType{
					Name: "identityref",
					IdentityBase: &yang.Identity{
						Name: "base-identity",
						Parent: &yang.Module{
							Name: "test-module",
						},
					},
				},
			},
		},
		wantCompressed: map[string]*yangGoEnum{
			"TestModule_BaseIdentity": {
				name: "TestModule_BaseIdentity",
				entry: &yang.Entry{
					Name: "identityref-leaf",
					Type: &yang.YangType{
						IdentityBase: &yang.Identity{
							Name: "base-identity",
							Parent: &yang.Module{
								Name: "test-module",
							},
						},
					},
				},
			},
		},
		wantSame: true,
	}, {
		name: "simple enumeration",
		in: map[string]*yang.Entry{
			"/container/config/enumeration-leaf": {
				Name: "enumeration-leaf",
				Type: &yang.YangType{
					Name: "enumeration",
					Enum: &yang.EnumType{},
				},
				Node: &yang.Enum{
					Parent: &yang.Container{
						Name: "config",
						Parent: &yang.Container{
							Name: "container",
							Parent: &yang.Module{
								Name: "base-module",
							},
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
			},
			"/container/state/enumeration-leaf": {
				Name: "enumeration-leaf",
				Type: &yang.YangType{
					Name: "enumeration",
					Enum: &yang.EnumType{},
				},
				Node: &yang.Enum{
					Parent: &yang.Container{
						Name: "state",
						Parent: &yang.Container{
							Name: "container",
							Parent: &yang.Module{
								Name: "base-module",
							},
						},
					},
				},
				Parent: &yang.Entry{
					Name: "state",
					Parent: &yang.Entry{
						Name: "container",
						Parent: &yang.Entry{
							Name: "base-module",
						},
					},
				},
			},
		},
		wantCompressed: map[string]*yangGoEnum{
			"BaseModule_Container_EnumerationLeaf": {
				name: "BaseModule_Container_EnumerationLeaf",
				entry: &yang.Entry{
					Name: "enumeration-leaf",
					Type: &yang.YangType{
						Enum: &yang.EnumType{},
					},
				},
			},
		},
		wantUncompressed: map[string]*yangGoEnum{
			"BaseModule_Container_State_EnumerationLeaf": {
				name: "BaseModule_Container_State_EnumerationLeaf",
				entry: &yang.Entry{
					Name: "enumeration-leaf",
					Type: &yang.YangType{
						Enum: &yang.EnumType{},
					},
				},
			},
			"BaseModule_Container_Config_EnumerationLeaf": {
				name: "BaseModule_Container_Config_EnumerationLeaf",
				entry: &yang.Entry{
					Name: "enumeration-leaf",
					Type: &yang.YangType{
						Enum: &yang.EnumType{},
					},
				},
			},
		},
	}, {
		name: "typedef which is an enumeration",
		in: map[string]*yang.Entry{
			"/container/config/enumeration-leaf": {
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
				Parent: &yang.Entry{
					Name: "config",
					Parent: &yang.Entry{
						Name: "container",
						Parent: &yang.Entry{
							Name: "base-module",
						},
					},
				},
			},
			"/container/state/enumeration-leaf": {
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
				Parent: &yang.Entry{
					Name: "state",
					Node: &yang.Container{Name: "state"},
					Parent: &yang.Entry{
						Name: "container",
						Node: &yang.Container{Name: "container"},
						Parent: &yang.Entry{
							Name: "base-module",
							Node: &yang.Module{Name: "base-module"},
						},
					},
				},
			},
		},
		wantCompressed: map[string]*yangGoEnum{
			"BaseModule_DerivedEnumeration": {
				name: "BaseModule_DerivedEnumeration",
				entry: &yang.Entry{
					Name: "enumeration-leaf",
					Type: &yang.YangType{
						Enum: &yang.EnumType{},
					},
				},
			},
		},
		wantSame: true,
	}, {
		name: "union which contains typedef with an enumeration",
		in: map[string]*yang.Entry{
			"/container/config/e": {
				Name: "e",
				Type: &yang.YangType{
					Kind: yang.Yunion,
					Type: []*yang.YangType{
						{Name: "derived", Kind: yang.Yenum, Enum: &yang.EnumType{}},
						{Kind: yang.Ystring},
					},
				},
				Node: &yang.Enum{
					Parent: &yang.Module{
						Name: "base-module",
					},
				},
				Parent: &yang.Entry{
					Name: "state",
					Node: &yang.Container{Name: "state"},
					Parent: &yang.Entry{
						Name: "base-module",
						Node: &yang.Module{Name: "base-module"},
					},
				},
			},
		},
		wantCompressed: map[string]*yangGoEnum{
			"BaseModule_Derived_Enum": {
				name: "BaseModule_Derived_Enum",
				entry: &yang.Entry{
					Name: "e",
					Type: &yang.YangType{
						Kind: yang.Yunion,
						Type: []*yang.YangType{
							{Name: "derived", Kind: yang.Yenum, Enum: &yang.EnumType{}},
							{Kind: yang.Ystring},
						},
					},
				},
			},
		},
		wantSame: true,
	}, {
		name: "typedef union with an enumeration",
		in: map[string]*yang.Entry{
			"/container/config/enumeration-leaf": {
				Name: "enumeration-leaf",
				Type: &yang.YangType{
					Name: "derived-union-enum",
					Type: []*yang.YangType{
						{Kind: yang.Yenum, Enum: &yang.EnumType{}},
						{Kind: yang.Yuint32},
					},
				},
				Node: &yang.Enum{
					Parent: &yang.Module{
						Name: "base-module",
					},
				},
			},
			"/container/state/enumeration-leaf": {
				Name: "enumeration-leaf",
				Type: &yang.YangType{
					Name: "derived-union-enum",
					Type: []*yang.YangType{
						{Kind: yang.Yenum, Enum: &yang.EnumType{}},
						{Kind: yang.Yuint32},
					},
				},
				Node: &yang.Enum{
					Parent: &yang.Module{
						Name: "base-module",
					},
				},
			},
		},
		wantCompressed: map[string]*yangGoEnum{
			"BaseModule_DerivedUnionEnum": {
				name: "BaseModule_DerivedUnionEnum",
				entry: &yang.Entry{
					Name: "enumeration-leaf",
					Type: &yang.YangType{
						Enum: &yang.EnumType{},
					},
				},
			},
		},
		wantSame: true,
	}, {
		name: "derived identityref",
		in: map[string]*yang.Entry{
			"/container/config/identityref-leaf": {
				Name: "identityref-leaf",
				Type: &yang.YangType{
					Name: "derived-identityref",
					IdentityBase: &yang.Identity{
						Name: "base-identity",
						Parent: &yang.Module{
							Name: "base-module",
						},
					},
				},
				Node: &yang.Leaf{
					Parent: &yang.Module{
						Name: "base-module",
					},
				},
			},
			"/container/state/identityref-leaf": {
				Name: "identityref-leaf",
				Type: &yang.YangType{
					Name: "derived-identityref",
					IdentityBase: &yang.Identity{
						Name: "base-identity",
						Parent: &yang.Module{
							Name: "base-module",
						},
					},
				},
				Node: &yang.Leaf{
					Parent: &yang.Module{
						Name: "base-module",
					},
				},
			},
		},
		wantCompressed: map[string]*yangGoEnum{
			"BaseModule_DerivedIdentityref": {
				name: "BaseModule_DerivedIdentityref",
				entry: &yang.Entry{
					Name: "identityref-leaf",
					Type: &yang.YangType{
						IdentityBase: &yang.Identity{
							Name: "base-identity",
							Parent: &yang.Module{
								Name: "test-module",
							},
						},
					},
				},
			},
		},
		wantSame: true,
	}, {
		name: "erroneous identityref",
		in: map[string]*yang.Entry{
			"/container/config/identityref-leaf": {
				Name: "invalid-identityref-leaf",
				Type: &yang.YangType{},
				Node: &yang.Leaf{
					Parent: &yang.Module{
						Name: "test-module",
					},
				},
			},
		},
		wantErr: true,
	}, {
		name: "union containing an identityref",
		in: map[string]*yang.Entry{
			"/container/state/union-identityref": {
				Name: "union-identityref",
				Type: &yang.YangType{
					Kind: yang.Yunion,
					Type: []*yang.YangType{{
						Kind: yang.Yidentityref,
						IdentityBase: &yang.Identity{
							Name: "base-identity",
							Parent: &yang.Module{
								Name: "base-module",
							},
						},
					}, {
						Kind: yang.Ystring,
					}},
				},
			},
		},
		wantCompressed: map[string]*yangGoEnum{
			"BaseModule_BaseIdentity": {
				name: "BaseModule_BaseIdentity",
				entry: &yang.Entry{
					Name: "union-identityref",
					Type: &yang.YangType{
						Type: []*yang.YangType{{
							Kind: yang.Yidentityref,
							IdentityBase: &yang.Identity{
								Name: "base-identity",
								Parent: &yang.Module{
									Name: "base-module",
								},
							},
						}, {
							Kind: yang.Ystring,
						}},
					},
				},
			},
		},
		wantSame: true,
	}, {
		name: "union containing a typedef identityref",
		in: map[string]*yang.Entry{
			"/container/state/union-typedef-identityref": {
				Name: "union-typedef-identityref",
				Type: &yang.YangType{
					Kind: yang.Yunion,
					Type: []*yang.YangType{{
						Name: "derived-identityref",
						Kind: yang.Yidentityref,
						IdentityBase: &yang.Identity{
							Name: "base-identity",
							Parent: &yang.Module{
								Name: "base-module",
							},
						},
					}, {
						Kind: yang.Ystring,
					}},
				},
				Node: &yang.Leaf{
					Parent: &yang.Module{
						Name: "test-module",
					},
				},
			},
		},
		wantCompressed: map[string]*yangGoEnum{
			"BaseModule_BaseIdentity": {
				name: "BaseModule_BaseIdentity",
				entry: &yang.Entry{
					Name: "union-typedef-identityref",
					Type: &yang.YangType{
						Kind: yang.Yunion,
						Type: []*yang.YangType{{
							Name: "derived-identityref",
							Kind: yang.Yidentityref,
							IdentityBase: &yang.Identity{
								Name: "base-identity",
								Parent: &yang.Module{
									Name: "base-module",
								},
							},
						}, {
							Kind: yang.Ystring,
						}},
					},
					Node: &yang.Leaf{
						Parent: &yang.Module{
							Name: "test-module",
						},
					},
				},
			},
		},
		wantSame: true,
	}, {
		name: "typedef union containing an identityref",
		in: map[string]*yang.Entry{
			"/container/state/typedef-union-identityref": {
				Name: "typedef-union-identityref",
				Type: &yang.YangType{
					Name: "derived",
					Kind: yang.Yunion,
					Type: []*yang.YangType{{
						Kind: yang.Yidentityref,
						IdentityBase: &yang.Identity{
							Name: "base-identity",
							Parent: &yang.Module{
								Name: "base-module",
							},
						},
					}, {
						Kind: yang.Ystring,
					}},
				},
				Node: &yang.Leaf{
					Parent: &yang.Module{
						Name: "test-module",
					},
				},
			},
		},
		wantCompressed: map[string]*yangGoEnum{
			"BaseModule_BaseIdentity": {
				name: "BaseModule_BaseIdentity",
				entry: &yang.Entry{
					Name: "typedef-union-identityref",
					Type: &yang.YangType{
						Name: "derived",
						Kind: yang.Yunion,
						Type: []*yang.YangType{{
							Kind: yang.Yidentityref,
							IdentityBase: &yang.Identity{
								Name: "base-identity",
								Parent: &yang.Module{
									Name: "base-module",
								},
							},
						}, {
							Kind: yang.Ystring,
						}},
					},
					Node: &yang.Leaf{
						Parent: &yang.Module{
							Name: "test-module",
						},
					},
				},
			},
		},
		wantSame: true,
	}, {
		name: "typedef of union that contains multiple enumerations",
		in: map[string]*yang.Entry{
			"err": {
				Name: "err",
				Type: &yang.YangType{
					Name: "derived",
					Kind: yang.Yunion,
					Type: []*yang.YangType{{
						Kind: yang.Yenum,
						Enum: &yang.EnumType{},
					}, {
						Kind: yang.Yenum,
						Enum: &yang.EnumType{},
					}},
				},
				Parent: &yang.Entry{
					Name: "config",
					Parent: &yang.Entry{
						Name: "test-container",
					},
				},
				Node: &yang.Leaf{
					Name: "err",
					Parent: &yang.Container{
						Name: "config",
						Parent: &yang.Container{
							Name: "test-container",
							Parent: &yang.Module{
								Name: "test-module",
							},
						},
					},
				},
			},
		},
		wantErr: true,
	}, {
		name: "typedef of union that contains an empty union",
		in: map[string]*yang.Entry{
			"err": {
				Name: "err",
				Type: &yang.YangType{
					Name: "derived",
					Kind: yang.Yunion,
					Type: []*yang.YangType{},
				},
				Parent: &yang.Entry{
					Name: "config",
					Parent: &yang.Entry{
						Name: "test-container",
					},
				},
				Node: &yang.Leaf{
					Name: "err",
					Parent: &yang.Container{
						Name: "config",
						Parent: &yang.Container{
							Name: "test-container",
							Parent: &yang.Module{
								Name: "test-module",
							},
						},
					},
				},
			},
		},
		wantErr: true,
	}, {
		name: "union of unions that contains an enumeration",
		in: map[string]*yang.Entry{
			"/container/state/e": {
				Name: "e",
				Type: &yang.YangType{
					Kind: yang.Yunion,
					Type: []*yang.YangType{{
						Kind: yang.Yunion,
						Type: []*yang.YangType{{
							Name: "enumeration",
							Kind: yang.Yenum,
							Enum: &yang.EnumType{},
						}, {
							Kind: yang.Ystring,
						}},
					}, {
						Kind: yang.Yint8,
					}},
					Enum: &yang.EnumType{},
				},
				Node: &yang.Enum{
					Parent: &yang.Container{
						Name: "state",
						Parent: &yang.Container{
							Name: "container",
							Parent: &yang.Module{
								Name: "base-module",
							},
						},
					},
				},
				Parent: &yang.Entry{
					Name: "state",
					Parent: &yang.Entry{
						Name:   "container",
						Parent: &yang.Entry{Name: "base-module"},
					},
				},
			},
		},
		wantCompressed: map[string]*yangGoEnum{
			"BaseModule_Container_E": {
				name: "BaseModule_Container_E",
				entry: &yang.Entry{
					Name: "e",
					Type: &yang.YangType{
						Kind: yang.Yunion,
						Type: []*yang.YangType{{
							Kind: yang.Yunion,
							Type: []*yang.YangType{{
								Name: "enumeration",
								Kind: yang.Yenum,
								Enum: &yang.EnumType{},
							}, {
								Kind: yang.Ystring,
							}},
						}, {
							Kind: yang.Yint8,
						}},
						Enum: &yang.EnumType{},
					},
					Node: &yang.Enum{
						Parent: &yang.Container{
							Name: "state",
							Parent: &yang.Container{
								Name: "container",
								Parent: &yang.Module{
									Name: "base-module",
								},
							},
						},
					},
					Parent: &yang.Entry{
						Name: "state",
						Parent: &yang.Entry{
							Name:   "container",
							Parent: &yang.Entry{Name: "base-module"},
						},
					},
				},
			},
		},
		wantUncompressed: map[string]*yangGoEnum{
			"BaseModule_Container_State_E": {
				name: "BaseModule_Container_State_E",
				entry: &yang.Entry{
					Name: "e",
					Type: &yang.YangType{
						Kind: yang.Yunion,
						Type: []*yang.YangType{{
							Kind: yang.Yunion,
							Type: []*yang.YangType{{
								Kind: yang.Yenum,
								Enum: &yang.EnumType{},
							}, {
								Kind: yang.Ystring,
							}},
						}, {
							Kind: yang.Yint8,
						}},
						Enum: &yang.EnumType{},
					},
					Node: &yang.Enum{
						Parent: &yang.Container{
							Name: "state",
							Parent: &yang.Container{
								Name: "container",
								Parent: &yang.Module{
									Name: "base-module",
								},
							},
						},
					},
					Parent: &yang.Entry{
						Name: "state",
						Parent: &yang.Entry{
							Name:   "container",
							Parent: &yang.Entry{Name: "base-module"},
						},
					},
				},
			},
		},
	}}

	for _, tt := range tests {
		var wantUncompressed map[string]*yangGoEnum
		if tt.wantSame {
			wantUncompressed = tt.wantCompressed
		} else {
			wantUncompressed = tt.wantUncompressed
		}
		for compressed, wanted := range map[bool]map[string]*yangGoEnum{true: tt.wantCompressed, false: wantUncompressed} {
			cg := NewYANGCodeGenerator(&GeneratorConfig{
				CompressOCPaths: compressed,
			})
			entries, errs := cg.state.findEnumSet(tt.in, cg.Config.CompressOCPaths)

			if len(errs) > 0 && !tt.wantErr {
				t.Errorf("%s (%v): encountered errors when extracting enums: %v",
					tt.name, compressed, errs)
				continue
			}

			for k, want := range wanted {
				got, ok := entries[k]
				if !ok {
					t.Errorf("%s findEnumSet(CompressOCPaths: %v): could not find expected entry, got: %v, want: %s", tt.name, compressed, entries, k)
					continue
				}

				if want.entry.Name != got.entry.Name {
					j, _ := json.Marshal(got)
					t.Errorf("%s findEnumSet(CompressOCPaths: %v): extracted entry has wrong name: got %s, want: %s (%s)", tt.name,
						compressed, got.entry.Name, want.entry.Name, string(j))
				}

				if want.entry.Type.IdentityBase != nil {
					// Check the identity's base if this was an identityref.
					if want.entry.Type.IdentityBase.Name != got.entry.Type.IdentityBase.Name {
						t.Errorf("%s findEnumSet(CompressOCPaths: %v): found identity %s, has wrong base, got: %v, want: %v", tt.name,
							compressed, want.entry.Name, want.entry.Type.IdentityBase.Name, got.entry.Type.IdentityBase.Name)
					}
				}
			}
		}
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
			s := newGenState()
			if out := s.structName(tt.inElement, compress, false); out != expected {
				t.Errorf("%s (compress: %v): shortName output invalid - got: %s, want: %s", tt.name, compress, out, expected)
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
		want         mappedType
		wantErr      bool
	}{{
		name: "simple lookup resolution",
		in:   &yang.YangType{Kind: yang.Yint32, Name: "int32"},
		want: mappedType{nativeType: "int32"},
	}, {
		name: "binary lookup resolution",
		in:   &yang.YangType{Kind: yang.Ybinary, Name: "binary"},
		want: mappedType{nativeType: "Binary"},
	}, {
		name: "unknown lookup resolution",
		in:   &yang.YangType{Kind: yang.YinstanceIdentifier, Name: "instanceIdentifier"},
		want: mappedType{nativeType: "interface{}"},
	}, {
		name: "simple empty resolution",
		in:   &yang.YangType{Kind: yang.Yempty, Name: "empty"},
		want: mappedType{nativeType: "bool"},
	}, {
		name: "simple boolean resolution",
		in:   &yang.YangType{Kind: yang.Ybool, Name: "bool"},
		want: mappedType{nativeType: "bool"},
	}, {
		name: "simple int64 resolution",
		in:   &yang.YangType{Kind: yang.Yint64, Name: "int64"},
		want: mappedType{nativeType: "int64"},
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
		want: mappedType{
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
		want: mappedType{
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
		want: mappedType{nativeType: "E_BaseModule_DerivedIdentityref", isEnumeratedValue: true},
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
		want: mappedType{nativeType: "E_BaseModule_EnumerationLeaf", isEnumeratedValue: true},
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
		want: mappedType{nativeType: "E_BaseModule_DerivedEnumeration", isEnumeratedValue: true},
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
		want: mappedType{nativeType: "E_TestModule_BaseIdentity", isEnumeratedValue: true},
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
		want:         mappedType{nativeType: "E_BaseModule_Container_Eleaf", isEnumeratedValue: true},
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
		want:         mappedType{nativeType: "E_BaseMod_Container_Eleaf", isEnumeratedValue: true},
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
		want: mappedType{nativeType: "uint32"},
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
			continue
		}

		if mappedType.nativeType != tt.want.nativeType {
			t.Errorf("%s: wrong type returned when mapping type: %s", tt.name, mappedType.nativeType)
		}

		if len(tt.want.unionTypes) > 0 {
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
			keys: map[string]mappedType{
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
			keys: map[string]mappedType{
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
			keys: map[string]mappedType{
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
			keys: map[string]mappedType{
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
			keys: map[string]mappedType{
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
			keys: map[string]mappedType{
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
		} else if err == nil && tt.wantErr {
			t.Errorf("%s: did not get expected error", tt.name)
		}

		if got == nil {
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
		wantTypes map[string]mappedType
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
		wantTypes: map[string]mappedType{
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
		wantTypes: map[string]mappedType{
			"/base-module/leaf-one": {nativeType: "E_BaseModule_DefinedType", isEnumeratedValue: true},
			"/base-module/leaf-two": {nativeType: "E_BaseModule_DefinedType", isEnumeratedValue: true},
		},
	}}

	for _, tt := range tests {
		s := newGenState()
		gotTypes := make(map[string]mappedType)
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
