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
	"reflect"
	"testing"

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

// TestBuildGoStructDefinitions tests the struct definition builder to ensure that the relevant
// entities are extracted from the input YANG.
func TestBuildGoStructDefinitions(t *testing.T) {
	tests := []struct {
		name           string
		in             []*yang.Entry
		wantCompress   map[string]yangDirectory
		wantUncompress map[string]yangDirectory
	}{{
		name: "basic struct generation test",
		in: []*yang.Entry{
			{
				Name: "module",
				Dir: map[string]*yang.Entry{
					"s1": {
						Name: "s1",
						Dir: map[string]*yang.Entry{
							"config": {
								Name:   "config",
								Parent: &yang.Entry{Name: "s1"},
								Dir: map[string]*yang.Entry{
									"l1": {Name: "l1", Type: &yang.YangType{Kind: yang.Ystring}},
									"l2": {Name: "l2", Type: &yang.YangType{Kind: yang.Yint8}},
								},
							},
							"state": {
								Name:   "state",
								Parent: &yang.Entry{Name: "s1"},
								Dir: map[string]*yang.Entry{
									"l1": {Name: "l1", Type: &yang.YangType{Kind: yang.Yint8}}, // Deliberate type mismatch
									"l2": {Name: "l2", Type: &yang.YangType{Kind: yang.Yint8}},
									"l3": {Name: "l3", Type: &yang.YangType{Kind: yang.Yint32}},
								},
							},
						},
					},
				},
			},
		},
		wantCompress: map[string]yangDirectory{
			"/s1": {
				name: "s1",
				fields: map[string]*yang.Entry{
					"l1": {Name: "l1", Type: &yang.YangType{Kind: yang.Ystring}},
					"l2": {Name: "l2", Type: &yang.YangType{Kind: yang.Yint8}},
					"l3": {Name: "l3", Type: &yang.YangType{Kind: yang.Yint32}},
				},
				path: []string{"", "s1"},
			},
		},
		wantUncompress: map[string]yangDirectory{
			"/s1": {
				name: "s1",
				path: []string{"", "s1"},
			},
			"/s1/config": {
				name: "config",
				fields: map[string]*yang.Entry{
					"l1": {Name: "l1", Type: &yang.YangType{Kind: yang.Ystring}},
					"l2": {Name: "l2", Type: &yang.YangType{Kind: yang.Yint8}},
				},
				path: []string{"", "s1", "config"},
			},
			"/s1/state": {
				name: "state",
				fields: map[string]*yang.Entry{
					"l1": {Name: "l1", Type: &yang.YangType{Kind: yang.Yint8}},
					"l2": {Name: "l2", Type: &yang.YangType{Kind: yang.Yint8}},
					"l3": {Name: "l3", Type: &yang.YangType{Kind: yang.Yint32}},
				},
				path: []string{"", "s1", "state"},
			},
		},
	}, {
		name: "nested container struct generation test",
		in: []*yang.Entry{
			{
				Name: "module",
				Dir: map[string]*yang.Entry{
					"s1": {
						Name: "s1",
						Dir: map[string]*yang.Entry{
							"config": {
								Name:   "config",
								Parent: &yang.Entry{Name: "s1"},
								Dir: map[string]*yang.Entry{
									"l1": {Name: "l1", Type: &yang.YangType{Kind: yang.Ystring}},
									"l2": {Name: "l2", Type: &yang.YangType{Kind: yang.Yint8}},
								},
							},
							"state": {
								Name:   "state",
								Parent: &yang.Entry{Name: "s1"},
								Dir: map[string]*yang.Entry{
									"l1": {Name: "l1", Type: &yang.YangType{Kind: yang.Yint8}}, // Deliberate type mismatch
									"l2": {Name: "l2", Type: &yang.YangType{Kind: yang.Yint8}},
									"l3": {Name: "l3", Type: &yang.YangType{Kind: yang.Yint32}},
								},
							},
							"outer-container": {
								Name:   "outer-container",
								Parent: &yang.Entry{Name: "s1"},
								Dir: map[string]*yang.Entry{
									"inner-container": {
										Name:   "inner-container",
										Parent: &yang.Entry{Name: "outer-container", Parent: &yang.Entry{Name: "s1"}},
										Dir: map[string]*yang.Entry{
											"config": {
												Name: "config",
												Parent: &yang.Entry{
													Name:   "inner-container",
													Parent: &yang.Entry{Name: "outer-container", Parent: &yang.Entry{Name: "s1"}},
												},
												Dir: map[string]*yang.Entry{
													"inner-leaf": {Name: "inner-leaf", Type: &yang.YangType{Kind: yang.Ystring}},
												},
											},
											"state": {
												Name: "state",
												Parent: &yang.Entry{
													Name:   "inner-container",
													Parent: &yang.Entry{Name: "outer-container", Parent: &yang.Entry{Name: "s1"}},
												},
												Dir: map[string]*yang.Entry{
													"inner-leaf":       {Name: "inner-leaf", Type: &yang.YangType{Kind: yang.Ystring}},
													"inner-state-leaf": {Name: "inner-state-leaf", Type: &yang.YangType{Kind: yang.Yint8}},
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
		},
		wantCompress: map[string]yangDirectory{
			"/s1": {
				name: "s1",
				fields: map[string]*yang.Entry{
					"l1": {Name: "l1", Type: &yang.YangType{Kind: yang.Ystring}},
					"l2": {Name: "l2", Type: &yang.YangType{Kind: yang.Yint8}},
					"l3": {Name: "l3", Type: &yang.YangType{Kind: yang.Yint32}},
				},
				path: []string{"", "s1"},
			},
			"/s1/outer-container": {
				name: "outer-container",
				fields: map[string]*yang.Entry{
					"inner-container": {Name: "inner-container"},
				},
				path: []string{"", "s1", "outer-container"},
			},
			"/s1/outer-container/inner-container": {
				name: "inner-container",
				fields: map[string]*yang.Entry{
					"inner-leaf":       {Name: "inner-leaf", Type: &yang.YangType{Kind: yang.Ystring}},
					"inner-state-leaf": {Name: "inner-state-leaf", Type: &yang.YangType{Kind: yang.Yint8}},
				},
				path: []string{"", "s1", "outer-container", "inner-container"},
			},
		},
		wantUncompress: map[string]yangDirectory{
			"/s1": {
				name: "s1",
				fields: map[string]*yang.Entry{
					"config":          {Name: "config"},
					"state":           {Name: "state"},
					"outer-container": {Name: "outer-container"},
				},
				path: []string{"", "s1"},
			},
			"/s1/config": {
				name: "config",
				fields: map[string]*yang.Entry{
					"l1": {Name: "l1", Type: &yang.YangType{Kind: yang.Ystring}},
					"l2": {Name: "l2", Type: &yang.YangType{Kind: yang.Yint8}},
				},
				path: []string{"", "s1", "config"},
			},
			"/s1/state": {
				name: "state",
				fields: map[string]*yang.Entry{
					"l1": {Name: "l1", Type: &yang.YangType{Kind: yang.Yint8}},
					"l2": {Name: "l2", Type: &yang.YangType{Kind: yang.Yint8}},
					"l3": {Name: "l3", Type: &yang.YangType{Kind: yang.Yint32}},
				},
				path: []string{"", "s1", "state"},
			},
			"/s1/outer-container": {
				name:   "outer-container",
				fields: map[string]*yang.Entry{"inner-container": {Name: "inner-container"}},
				path:   []string{"", "s1", "outer-container"},
			},
			"/s1/outer-container/inner-container": {
				name: "inner-container",
				fields: map[string]*yang.Entry{
					"config": {Name: "config"},
					"state":  {Name: "state"},
				},
				path: []string{"", "s1", "outer-container", "inner-container"},
			},
			"/s1/outer-container/inner-container/config": {
				name: "config",
				fields: map[string]*yang.Entry{
					"inner-leaf": {Name: "inner-leaf", Type: &yang.YangType{Kind: yang.Ystring}},
				},
				path: []string{"", "s1", "outer-container", "inner-container", "config"},
			},
			"/s1/outer-container/inner-container/state": {
				name: "state",
				fields: map[string]*yang.Entry{
					"inner-state-leaf": {Name: "inner-state-leaf", Type: &yang.YangType{Kind: yang.Yint8}},
				},
				path: []string{"", "s1", "outer-container", "inner-container", "state"},
			},
		},
	}, {
		name: "container with choice around leaves",
		in: []*yang.Entry{
			{
				Name: "module",
				Dir: map[string]*yang.Entry{
					"top-container": {
						Name: "top-container",
						Dir: map[string]*yang.Entry{
							"config": {
								Name:   "config",
								Parent: &yang.Entry{Name: "top-container"},
								Dir: map[string]*yang.Entry{
									"choice-node": {
										Name: "choice-node",
										Kind: yang.ChoiceEntry,
										Dir: map[string]*yang.Entry{
											"case-one": {
												Name: "case-one",
												Kind: yang.CaseEntry,
												Parent: &yang.Entry{
													Name:   "choice-node",
													Kind:   yang.ChoiceEntry,
													Parent: &yang.Entry{Name: "config", Parent: &yang.Entry{Name: "top-container"}},
												},
												Dir: map[string]*yang.Entry{
													"leaf-one": {Name: "leaf-one", Type: &yang.YangType{Kind: yang.Yint8}},
												},
											},
											"case-two": {
												Name: "case-two",
												Kind: yang.CaseEntry,
												Parent: &yang.Entry{
													Name:   "choice-node",
													Kind:   yang.ChoiceEntry,
													Parent: &yang.Entry{Name: "config", Parent: &yang.Entry{Name: "top-container"}},
												},
												Dir: map[string]*yang.Entry{
													"leaf-two": {Name: "leaf-two", Type: &yang.YangType{Kind: yang.Yint8}},
												},
											},
										},
									},
								},
							},
							"state": {
								Name:   "state",
								Parent: &yang.Entry{Name: "top-container"},
								Dir: map[string]*yang.Entry{
									"choice-node": {
										Name: "choice-node",
										Kind: yang.ChoiceEntry,
										Dir: map[string]*yang.Entry{
											"case-one": {
												Name: "case-one",
												Kind: yang.CaseEntry,
												Parent: &yang.Entry{
													Name:   "choice-node",
													Kind:   yang.ChoiceEntry,
													Parent: &yang.Entry{Name: "state", Parent: &yang.Entry{Name: "top-container"}},
												},
												Dir: map[string]*yang.Entry{"leaf-one": {Name: "leaf-one"}},
											},
											"case-two": {
												Name: "case-two",
												Kind: yang.CaseEntry,
												Parent: &yang.Entry{
													Name:   "choice-node",
													Kind:   yang.ChoiceEntry,
													Parent: &yang.Entry{Name: "state", Parent: &yang.Entry{Name: "top-container"}},
												},
												Dir: map[string]*yang.Entry{"leaf-two": {Name: "leaf-two"}},
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
		wantCompress: map[string]yangDirectory{
			"/top-container": {
				name: "top-container",
				fields: map[string]*yang.Entry{
					"leaf-one": {Name: "leaf-one", Type: &yang.YangType{Kind: yang.Yint8}},
					"leaf-two": {Name: "leaf-two", Type: &yang.YangType{Kind: yang.Yint8}},
				},
				path: []string{"", "top-container"},
			},
		},
		wantUncompress: map[string]yangDirectory{
			"/top-container": {
				name: "top-container",
				fields: map[string]*yang.Entry{
					"config": {Name: "config"},
					"state":  {Name: "state"},
				},
				path: []string{"", "top-container"},
			},
			"/top-container/config": {
				name: "config",
				fields: map[string]*yang.Entry{
					"leaf-one": {Name: "leaf-one", Type: &yang.YangType{Kind: yang.Yint8}},
					"leaf-two": {Name: "leaf-two", Type: &yang.YangType{Kind: yang.Yint8}},
				},
				path: []string{"", "top-container", "config"},
			},
			"/top-container/state": {
				name: "state",
				fields: map[string]*yang.Entry{
					"leaf-one": {Name: "leaf-one", Type: &yang.YangType{Kind: yang.Yint8}},
					"leaf-two": {Name: "leaf-two", Type: &yang.YangType{Kind: yang.Yint8}},
				},
				path: []string{"", "top-container", "state"},
			},
		},
	}, {
		name: "schema with list",
		in: []*yang.Entry{
			{
				Name: "container",
				Dir: map[string]*yang.Entry{
					"list": {
						Name:     "list",
						Parent:   &yang.Entry{Name: "container", Parent: &yang.Entry{Name: "module"}},
						Key:      "key",
						ListAttr: &yang.ListAttr{},
						Dir: map[string]*yang.Entry{
							"key": {
								Name: "key",
								Type: &yang.YangType{Kind: yang.Yleafref, Path: "../config/key"},
								Parent: &yang.Entry{
									Name: "list",
									Parent: &yang.Entry{
										Name: "container",
										Parent: &yang.Entry{
											Name: "module",
										},
									},
								},
							},
							"config": {
								Name:   "config",
								Parent: &yang.Entry{Name: "list", Parent: &yang.Entry{Name: "container", Parent: &yang.Entry{Name: "module"}}},
								Dir: map[string]*yang.Entry{
									"key": {
										Name: "key",
										Type: &yang.YangType{Kind: yang.Ystring},
										Parent: &yang.Entry{
											Name: "config",
											Parent: &yang.Entry{
												Name: "list",
												Parent: &yang.Entry{
													Name: "container",
													Parent: &yang.Entry{
														Name: "module",
													},
												},
											},
										},
									},
								},
							},
							"state": {
								Name:   "state",
								Parent: &yang.Entry{Name: "list", Parent: &yang.Entry{Name: "container", Parent: &yang.Entry{Name: "module"}}},

								Dir: map[string]*yang.Entry{
									"key": {
										Name: "key",
										Type: &yang.YangType{Kind: yang.Ystring},
										Parent: &yang.Entry{
											Name: "config",
											Parent: &yang.Entry{
												Name: "list",
												Parent: &yang.Entry{
													Name: "container",
													Parent: &yang.Entry{
														Name: "module",
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
				Parent: &yang.Entry{Name: "module"},
			},
		},
		wantCompress: map[string]yangDirectory{
			"/module/container/list": {
				name: "list",
				fields: map[string]*yang.Entry{
					"key": {Name: "key", Type: &yang.YangType{Kind: yang.Ystring}},
				},
			},
		},
		wantUncompress: map[string]yangDirectory{
			"/module/container/list": {
				name: "list",
				fields: map[string]*yang.Entry{
					"key":    {Name: "key", Type: &yang.YangType{Kind: yang.Yleafref}},
					"config": {Name: "config"},
					"state":  {Name: "state"},
				},
			},
			"/module/container/list/config": {
				name: "config",
				fields: map[string]*yang.Entry{
					"key": {Name: "key", Type: &yang.YangType{Kind: yang.Ystring}},
				},
			},
			"/module/container/list/state": {
				name: "state",
				fields: map[string]*yang.Entry{
					"key": {Name: "key", Type: &yang.YangType{Kind: yang.Ystring}},
				},
			},
		},
	}, {
		name: "schema with choice around container",
		in: []*yang.Entry{
			{
				Name: "container",
				Dir: map[string]*yang.Entry{
					"choice-node": {
						Name:   "choice-node",
						Kind:   yang.ChoiceEntry,
						Parent: &yang.Entry{Name: "container"},
						Dir: map[string]*yang.Entry{
							"case-one": {
								Name:   "case-one",
								Kind:   yang.CaseEntry,
								Parent: &yang.Entry{Name: "choice-node", Parent: &yang.Entry{Name: "container"}},
								Dir: map[string]*yang.Entry{
									"second-container": {
										Name: "second-container",
										Parent: &yang.Entry{
											Name: "case-one",
											Parent: &yang.Entry{
												Name:   "choice-node",
												Parent: &yang.Entry{Name: "container"},
											},
										},
										Dir: map[string]*yang.Entry{
											"config": {
												Name: "config",
												Parent: &yang.Entry{
													Name: "second-container",
													Parent: &yang.Entry{
														Name:   "case-one",
														Parent: &yang.Entry{Name: "choice-node", Parent: &yang.Entry{Name: "container"}},
													},
												},
												Dir: map[string]*yang.Entry{"leaf-one": {Name: "leaf-one"}},
											},
										},
									},
								},
							},
							"case-two": {
								Name:   "case-two",
								Kind:   yang.CaseEntry,
								Parent: &yang.Entry{Name: "choice-node", Parent: &yang.Entry{Name: "container"}},
								Dir: map[string]*yang.Entry{
									"third-container": {
										Name: "third-container",
										Parent: &yang.Entry{
											Name:   "case-two",
											Parent: &yang.Entry{Name: "choice-node", Parent: &yang.Entry{Name: "container"}},
										},
										Dir: map[string]*yang.Entry{
											"config": {
												Name: "config",
												Parent: &yang.Entry{
													Name: "third-container",
													Parent: &yang.Entry{
														Name:   "case-two",
														Parent: &yang.Entry{Name: "choice-node", Parent: &yang.Entry{Name: "container"}},
													},
												},
												Dir: map[string]*yang.Entry{"leaf-two": {Name: "leaf-two"}},
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
		wantCompress: map[string]yangDirectory{
			"/container": {
				name: "container",
				fields: map[string]*yang.Entry{
					"second-container": {Name: "second-container"},
					"third-container":  {Name: "third-container"},
				},
			},
			// Since these are schema paths then we still have the choice node's name
			// here, we need to check that the processing recursed correctly into the
			// container.
			"/container/choice-node/case-one/second-container": {
				name:   "second-container",
				fields: map[string]*yang.Entry{"leaf-one": {Name: "leaf-one"}},
			},
			"/container/choice-node/case-two/third-container": {
				name:   "third-container",
				fields: map[string]*yang.Entry{"leaf-two": {Name: "leaf-two"}},
			},
		},
		wantUncompress: map[string]yangDirectory{
			"/container": {
				name: "container",
				fields: map[string]*yang.Entry{
					"second-container": {Name: "second-container"},
					"third-container":  {Name: "third-container"},
				},
			},
			"/container/choice-node/case-one/second-container": {
				name:   "second-container",
				fields: map[string]*yang.Entry{"config": {Name: "config"}},
			},
			"/container/choice-node/case-two/third-container": {
				name:   "third-container",
				fields: map[string]*yang.Entry{"config": {Name: "config"}},
			},
			"/container/choice-node/case-one/second-container/config": {
				name:   "config",
				fields: map[string]*yang.Entry{"leaf-one": {Name: "leaf-one"}},
			},
			"/container/choice-node/case-two/third-container/config": {
				name:   "config",
				fields: map[string]*yang.Entry{"leaf-two": {Name: "leaf-two"}},
			},
		},
	}}

	for _, tt := range tests {
		for compress, expected := range map[bool]map[string]yangDirectory{true: tt.wantCompress, false: tt.wantUncompress} {
			cg := NewYANGCodeGenerator(&GeneratorConfig{
				CompressOCPaths: compress,
			})

			st, err := buildSchemaTree(tt.in)
			if err != nil {
				t.Errorf("%s: buildSchemaTree(%v), got unexpected err: %v", tt.name, tt.in, err)
				continue
			}
			cg.state.schematree = st

			structs := make(map[string]*yang.Entry)
			enums := make(map[string]*yang.Entry)

			for _, inc := range tt.in {
				cg.findMappableEntities(inc, structs, enums)
			}

			structDefs, errs := cg.state.buildGoStructDefinitions(structs, cg.Config.CompressOCPaths, cg.Config.GenerateFakeRoot)
			if len(errs) > 0 {
				t.Errorf("%s buildGoStructDefinitions(CompressOCPaths: %v): could not build struct defs: %v", tt.name, compress, errs)
				continue
			}

			for name, gostruct := range structDefs {
				expstr, ok := expected[name]
				if !ok {
					t.Errorf("%s buildGoStructDefinitions(CompressOCPaths: %v): could not find expected struct %s, got: %v, want: %v",
						tt.name, compress, name, structDefs, expected)
					continue
				}

				for fieldk, fieldv := range expstr.fields {
					cmpfield, ok := gostruct.fields[fieldk]
					if !ok {
						t.Errorf("%s buildGoStructDefinitions(CompressOCPaths: %v): could not find expected field %s in %s, got: %v",
							tt.name, compress, fieldk, name, gostruct.fields)
						continue
					}

					if fieldv.Name != cmpfield.Name {
						t.Errorf("%s buildGoStructDefinitions(CompressOCPaths: %v): field %s of %s did not have expected name, got: %v, want: %v",
							tt.name, compress, fieldk, name, fieldv.Name, cmpfield.Name)
					}

					if fieldv.Type != nil && cmpfield.Type != nil {
						if fieldv.Type.Kind != cmpfield.Type.Kind {
							t.Errorf("%s buildGoStructDefinitions(CompressOCPaths: %v): field %s of %s did not have expected type got: %s, want: %s",
								tt.name, compress, fieldk, name, fieldv.Type.Kind, cmpfield.Type.Kind)
						}
					}

				}

				if len(expstr.path) > 0 && !reflect.DeepEqual(expstr.path, gostruct.path) {
					t.Errorf("%s (%v): %s did not have matching path, got: %v, want: %v", tt.name, compress, name, expstr.path, gostruct.path)
				}
			}
		}
	}
}
