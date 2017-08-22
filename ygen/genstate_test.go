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
