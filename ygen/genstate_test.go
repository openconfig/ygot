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
		name              string
		in                map[string]*yang.Entry
		inOmitUnderscores bool
		wantCompressed    map[string]*yangEnum
		wantUncompressed  map[string]*yangEnum
		wantSame          bool // Whether to expect same compressed/uncompressed output
		wantErr           bool
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
		wantCompressed: map[string]*yangEnum{
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
		wantCompressed: map[string]*yangEnum{
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
		wantUncompressed: map[string]*yangEnum{
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
		wantCompressed: map[string]*yangEnum{
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
		wantCompressed: map[string]*yangEnum{
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
		wantCompressed: map[string]*yangEnum{
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
		wantCompressed: map[string]*yangEnum{
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
				Type: &yang.YangType{
					Name: "identityref",
				},
				Node: &yang.Leaf{
					Parent: &yang.Container{
						Name: "config",
						Parent: &yang.Container{
							Name: "container",
							Parent: &yang.Module{
								Name: "module",
							},
						},
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
		wantCompressed: map[string]*yangEnum{
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
		wantCompressed: map[string]*yangEnum{
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
		wantCompressed: map[string]*yangEnum{
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
		wantCompressed: map[string]*yangEnum{
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
		wantUncompressed: map[string]*yangEnum{
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
		var wantUncompressed map[string]*yangEnum
		if tt.wantSame {
			wantUncompressed = tt.wantCompressed
		} else {
			wantUncompressed = tt.wantUncompressed
		}
		for compressed, wanted := range map[bool]map[string]*yangEnum{true: tt.wantCompressed, false: wantUncompressed} {
			cg := NewYANGCodeGenerator(&GeneratorConfig{
				CompressOCPaths: compressed,
			})
			entries, errs := cg.state.findEnumSet(tt.in, cg.Config.CompressOCPaths, tt.inOmitUnderscores)

			if (errs != nil) != tt.wantErr {
				t.Errorf("%s findEnumSet(%v, %v): did not get expected error when extracting enums, got: %v (len %d), wanted err: %v", tt.name, tt.in, cg.Config.CompressOCPaths, errs, len(errs), tt.wantErr)
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
			if out := s.goStructName(tt.inElement, compress, false); out != expected {
				t.Errorf("%s (compress: %v): shortName output invalid - got: %s, want: %s", tt.name, compress, out, expected)
			}
		}
	}
}

func TestBuildDirectoryDefinitions(t *testing.T) {
	tests := []struct {
		name                string
		in                  []*yang.Entry
		wantGoCompress      map[string]yangDirectory
		wantGoUncompress    map[string]yangDirectory
		wantProtoCompress   map[string]yangDirectory
		wantProtoUncompress map[string]yangDirectory
	}{{
		name: "basic struct generation test",
		in: []*yang.Entry{{
			Name: "module",
			Dir: map[string]*yang.Entry{
				"s1": {
					Name:   "s1",
					Parent: &yang.Entry{Name: "module"},
					Dir: map[string]*yang.Entry{
						"config": {
							Name: "config",
							Parent: &yang.Entry{
								Name: "s1",
								Parent: &yang.Entry{
									Name: "module",
								},
							},
							Dir: map[string]*yang.Entry{
								"l1": {Name: "l1", Type: &yang.YangType{Kind: yang.Ystring}},
								"l2": {Name: "l2", Type: &yang.YangType{Kind: yang.Yint8}},
							},
						},
						"state": {
							Name: "state",
							Parent: &yang.Entry{
								Name: "s1",
								Parent: &yang.Entry{
									Name: "module",
								},
							},
							Dir: map[string]*yang.Entry{
								"l1": {Name: "l1", Type: &yang.YangType{Kind: yang.Yint8}}, // Deliberate type mismatch
								"l2": {Name: "l2", Type: &yang.YangType{Kind: yang.Yint8}},
								"l3": {Name: "l3", Type: &yang.YangType{Kind: yang.Yint32}},
							},
						},
					},
				},
			},
		}},
		wantGoCompress: map[string]yangDirectory{
			"/module/s1": {
				name: "S1",
				fields: map[string]*yang.Entry{
					"l1": {Name: "l1", Type: &yang.YangType{Kind: yang.Ystring}},
					"l2": {Name: "l2", Type: &yang.YangType{Kind: yang.Yint8}},
					"l3": {Name: "l3", Type: &yang.YangType{Kind: yang.Yint32}},
				},
				path: []string{"", "module", "s1"},
			},
		},
		wantGoUncompress: map[string]yangDirectory{
			"/module/s1": {
				name: "Module_S1",
				path: []string{"", "module", "s1"},
			},
			"/module/s1/config": {
				name: "Module_S1_Config",
				fields: map[string]*yang.Entry{
					"l1": {Name: "l1", Type: &yang.YangType{Kind: yang.Ystring}},
					"l2": {Name: "l2", Type: &yang.YangType{Kind: yang.Yint8}},
				},
				path: []string{"", "module", "s1", "config"},
			},
			"/module/s1/state": {
				name: "Module_S1_State",
				fields: map[string]*yang.Entry{
					"l1": {Name: "l1", Type: &yang.YangType{Kind: yang.Yint8}},
					"l2": {Name: "l2", Type: &yang.YangType{Kind: yang.Yint8}},
					"l3": {Name: "l3", Type: &yang.YangType{Kind: yang.Yint32}},
				},
				path: []string{"", "module", "s1", "state"},
			},
		},
		wantProtoCompress: map[string]yangDirectory{
			"/module/s1": {
				name: "S1",
				fields: map[string]*yang.Entry{
					"l1": {Name: "l1", Type: &yang.YangType{Kind: yang.Ystring}},
					"l2": {Name: "l2", Type: &yang.YangType{Kind: yang.Yint8}},
					"l3": {Name: "l3", Type: &yang.YangType{Kind: yang.Yint32}},
				},
				path: []string{"", "module", "s1"},
			},
		},
		wantProtoUncompress: map[string]yangDirectory{
			"/module/s1": {
				name: "S1",
				path: []string{"", "module", "s1"},
			},
			"/module/s1/config": {
				name: "Config",
				fields: map[string]*yang.Entry{
					"l1": {Name: "l1", Type: &yang.YangType{Kind: yang.Ystring}},
					"l2": {Name: "l2", Type: &yang.YangType{Kind: yang.Yint8}},
				},
				path: []string{"", "module", "s1", "config"},
			},
			"/module/s1/state": {
				name: "State",
				fields: map[string]*yang.Entry{
					"l1": {Name: "l1", Type: &yang.YangType{Kind: yang.Yint8}},
					"l2": {Name: "l2", Type: &yang.YangType{Kind: yang.Yint8}},
					"l3": {Name: "l3", Type: &yang.YangType{Kind: yang.Yint32}},
				},
				path: []string{"", "module", "s1", "state"},
			},
		},
	}, {
		name: "nested container struct generation test",
		in: []*yang.Entry{
			{
				Name: "module",
				Dir: map[string]*yang.Entry{
					"s1": {
						Name:   "s1",
						Parent: &yang.Entry{Name: "module"},
						Dir: map[string]*yang.Entry{
							"config": {
								Name: "config",
								Parent: &yang.Entry{
									Name: "s1",
									Parent: &yang.Entry{
										Name: "module",
									},
								},
								Dir: map[string]*yang.Entry{
									"l1": {Name: "l1", Type: &yang.YangType{Kind: yang.Ystring}},
									"l2": {Name: "l2", Type: &yang.YangType{Kind: yang.Yint8}},
								},
							},
							"state": {
								Name: "state",
								Parent: &yang.Entry{
									Name: "s1",
									Parent: &yang.Entry{
										Name: "module",
									},
								},
								Dir: map[string]*yang.Entry{
									"l1": {Name: "l1", Type: &yang.YangType{Kind: yang.Yint8}}, // Deliberate type mismatch
									"l2": {Name: "l2", Type: &yang.YangType{Kind: yang.Yint8}},
									"l3": {Name: "l3", Type: &yang.YangType{Kind: yang.Yint32}},
								},
							},
							"outer-container": {
								Name: "outer-container",
								Parent: &yang.Entry{
									Name: "s1",
									Parent: &yang.Entry{
										Name: "module",
									},
								},
								Dir: map[string]*yang.Entry{
									"inner-container": {
										Name: "inner-container",
										Parent: &yang.Entry{
											Name: "outer-container",
											Parent: &yang.Entry{
												Name: "s1",
												Parent: &yang.Entry{
													Name: "module",
												},
											},
										},
										Dir: map[string]*yang.Entry{
											"config": {
												Name: "config",
												Parent: &yang.Entry{
													Name: "inner-container",
													Parent: &yang.Entry{
														Name: "outer-container",
														Parent: &yang.Entry{
															Name: "s1",
															Parent: &yang.Entry{
																Name: "module",
															},
														},
													},
												},
												Dir: map[string]*yang.Entry{
													"inner-leaf": {Name: "inner-leaf", Type: &yang.YangType{Kind: yang.Ystring}},
												},
											},
											"state": {
												Name: "state",
												Parent: &yang.Entry{
													Name: "inner-container",
													Parent: &yang.Entry{
														Name: "outer-container",
														Parent: &yang.Entry{
															Name: "s1",
															Parent: &yang.Entry{
																Name: "module",
															},
														},
													},
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
		wantGoCompress: map[string]yangDirectory{
			"/module/s1": {
				name: "S1",
				fields: map[string]*yang.Entry{
					"l1": {Name: "l1", Type: &yang.YangType{Kind: yang.Ystring}},
					"l2": {Name: "l2", Type: &yang.YangType{Kind: yang.Yint8}},
					"l3": {Name: "l3", Type: &yang.YangType{Kind: yang.Yint32}},
				},
				path: []string{"", "module", "s1"},
			},
			"/module/s1/outer-container": {
				name: "S1_OuterContainer",
				fields: map[string]*yang.Entry{
					"inner-container": {Name: "inner-container"},
				},
				path: []string{"", "module", "s1", "outer-container"},
			},
			"/module/s1/outer-container/inner-container": {
				name: "S1_OuterContainer_InnerContainer",
				fields: map[string]*yang.Entry{
					"inner-leaf":       {Name: "inner-leaf", Type: &yang.YangType{Kind: yang.Ystring}},
					"inner-state-leaf": {Name: "inner-state-leaf", Type: &yang.YangType{Kind: yang.Yint8}},
				},
				path: []string{"", "module", "s1", "outer-container", "inner-container"},
			},
		},
		wantGoUncompress: map[string]yangDirectory{
			"/module/s1": {
				name: "Module_S1",
				fields: map[string]*yang.Entry{
					"config":          {Name: "config"},
					"state":           {Name: "state"},
					"outer-container": {Name: "outer-container"},
				},
				path: []string{"", "module", "s1"},
			},
			"/module/s1/config": {
				name: "Module_S1_Config",
				fields: map[string]*yang.Entry{
					"l1": {Name: "l1", Type: &yang.YangType{Kind: yang.Ystring}},
					"l2": {Name: "l2", Type: &yang.YangType{Kind: yang.Yint8}},
				},
				path: []string{"", "module", "s1", "config"},
			},
			"/module/s1/state": {
				name: "Module_S1_State",
				fields: map[string]*yang.Entry{
					"l1": {Name: "l1", Type: &yang.YangType{Kind: yang.Yint8}},
					"l2": {Name: "l2", Type: &yang.YangType{Kind: yang.Yint8}},
					"l3": {Name: "l3", Type: &yang.YangType{Kind: yang.Yint32}},
				},
				path: []string{"", "module", "s1", "state"},
			},
			"/module/s1/outer-container": {
				name:   "Module_S1_OuterContainer",
				fields: map[string]*yang.Entry{"inner-container": {Name: "inner-container"}},
				path:   []string{"", "module", "s1", "outer-container"},
			},
			"/module/s1/outer-container/inner-container": {
				name: "Module_S1_OuterContainer_InnerContainer",
				fields: map[string]*yang.Entry{
					"config": {Name: "config"},
					"state":  {Name: "state"},
				},
				path: []string{"", "module", "s1", "outer-container", "inner-container"},
			},
			"/module/s1/outer-container/inner-container/config": {
				name: "Module_S1_OuterContainer_InnerContainer_Config",
				fields: map[string]*yang.Entry{
					"inner-leaf": {Name: "inner-leaf", Type: &yang.YangType{Kind: yang.Ystring}},
				},
				path: []string{"", "module", "s1", "outer-container", "inner-container", "config"},
			},
			"/module/s1/outer-container/inner-container/state": {
				name: "Module_S1_OuterContainer_InnerContainer_State",
				fields: map[string]*yang.Entry{
					"inner-state-leaf": {Name: "inner-state-leaf", Type: &yang.YangType{Kind: yang.Yint8}},
				},
				path: []string{"", "module", "s1", "outer-container", "inner-container", "state"},
			},
		},
		wantProtoCompress: map[string]yangDirectory{
			"/module/s1": {
				name: "S1",
				fields: map[string]*yang.Entry{
					"l1": {Name: "l1", Type: &yang.YangType{Kind: yang.Ystring}},
					"l2": {Name: "l2", Type: &yang.YangType{Kind: yang.Yint8}},
					"l3": {Name: "l3", Type: &yang.YangType{Kind: yang.Yint32}},
				},
				path: []string{"", "module", "s1"},
			},
			"/module/s1/outer-container": {
				name: "OuterContainer",
				fields: map[string]*yang.Entry{
					"inner-container": {Name: "inner-container"},
				},
				path: []string{"", "module", "s1", "outer-container"},
			},
			"/module/s1/outer-container/inner-container": {
				name: "InnerContainer",
				fields: map[string]*yang.Entry{
					"inner-leaf":       {Name: "inner-leaf", Type: &yang.YangType{Kind: yang.Ystring}},
					"inner-state-leaf": {Name: "inner-state-leaf", Type: &yang.YangType{Kind: yang.Yint8}},
				},
				path: []string{"", "module", "s1", "outer-container", "inner-container"},
			},
		},
		wantProtoUncompress: map[string]yangDirectory{
			"/module/s1": {
				name: "S1",
				fields: map[string]*yang.Entry{
					"config":          {Name: "config"},
					"state":           {Name: "state"},
					"outer-container": {Name: "outer-container"},
				},
				path: []string{"", "module", "s1"},
			},
			"/module/s1/config": {
				name: "Config",
				fields: map[string]*yang.Entry{
					"l1": {Name: "l1", Type: &yang.YangType{Kind: yang.Ystring}},
					"l2": {Name: "l2", Type: &yang.YangType{Kind: yang.Yint8}},
				},
				path: []string{"", "module", "s1", "config"},
			},
			"/module/s1/state": {
				name: "State",
				fields: map[string]*yang.Entry{
					"l1": {Name: "l1", Type: &yang.YangType{Kind: yang.Yint8}},
					"l2": {Name: "l2", Type: &yang.YangType{Kind: yang.Yint8}},
					"l3": {Name: "l3", Type: &yang.YangType{Kind: yang.Yint32}},
				},
				path: []string{"", "module", "s1", "state"},
			},
			"/module/s1/outer-container": {
				name:   "OuterContainer",
				fields: map[string]*yang.Entry{"inner-container": {Name: "inner-container"}},
				path:   []string{"", "module", "s1", "outer-container"},
			},
			"/module/s1/outer-container/inner-container": {
				name: "InnerContainer",
				fields: map[string]*yang.Entry{
					"config": {Name: "config"},
					"state":  {Name: "state"},
				},
				path: []string{"", "module", "s1", "outer-container", "inner-container"},
			},
			"/module/s1/outer-container/inner-container/config": {
				name: "Config",
				fields: map[string]*yang.Entry{
					"inner-leaf": {Name: "inner-leaf", Type: &yang.YangType{Kind: yang.Ystring}},
				},
				path: []string{"", "module", "s1", "outer-container", "inner-container", "config"},
			},
			"/module/s1/outer-container/inner-container/state": {
				name: "State",
				fields: map[string]*yang.Entry{
					"inner-state-leaf": {Name: "inner-state-leaf", Type: &yang.YangType{Kind: yang.Yint8}},
				},
				path: []string{"", "module", "s1", "outer-container", "inner-container", "state"},
			},
		},
	}, {
		name: "container with choice around leaves",
		in: []*yang.Entry{
			{
				Name: "module",
				Dir: map[string]*yang.Entry{
					"top-container": {
						Name:   "top-container",
						Parent: &yang.Entry{Name: "module"},
						Dir: map[string]*yang.Entry{
							"config": {
								Name: "config",
								Parent: &yang.Entry{
									Name: "top-container",
									Parent: &yang.Entry{
										Name: "module",
									},
								},
								Dir: map[string]*yang.Entry{
									"choice-node": {
										Name: "choice-node",
										Kind: yang.ChoiceEntry,
										Dir: map[string]*yang.Entry{
											"case-one": {
												Name: "case-one",
												Kind: yang.CaseEntry,
												Parent: &yang.Entry{
													Name: "choice-node",
													Kind: yang.ChoiceEntry,
													Parent: &yang.Entry{
														Name: "config",
														Parent: &yang.Entry{
															Name: "top-container",
															Parent: &yang.Entry{
																Name: "module",
															},
														},
													},
												},
												Dir: map[string]*yang.Entry{
													"leaf-one": {Name: "leaf-one", Type: &yang.YangType{Kind: yang.Yint8}},
												},
											},
											"case-two": {
												Name: "case-two",
												Kind: yang.CaseEntry,
												Parent: &yang.Entry{
													Name: "choice-node",
													Kind: yang.ChoiceEntry,
													Parent: &yang.Entry{
														Name: "config",
														Parent: &yang.Entry{
															Name: "top-container",
															Parent: &yang.Entry{
																Name: "module",
															},
														},
													},
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
								Name: "state",
								Parent: &yang.Entry{
									Name: "top-container",
									Parent: &yang.Entry{
										Name: "module",
									},
								},
								Dir: map[string]*yang.Entry{
									"choice-node": {
										Name: "choice-node",
										Kind: yang.ChoiceEntry,
										Dir: map[string]*yang.Entry{
											"case-one": {
												Name: "case-one",
												Kind: yang.CaseEntry,
												Parent: &yang.Entry{
													Name: "choice-node",
													Kind: yang.ChoiceEntry,
													Parent: &yang.Entry{
														Name: "state",
														Parent: &yang.Entry{
															Name: "top-container",
															Parent: &yang.Entry{
																Name: "module",
															},
														},
													},
												},
												Dir: map[string]*yang.Entry{"leaf-one": {Name: "leaf-one"}},
											},
											"case-two": {
												Name: "case-two",
												Kind: yang.CaseEntry,
												Parent: &yang.Entry{
													Name: "choice-node",
													Kind: yang.ChoiceEntry,
													Parent: &yang.Entry{
														Name: "state",
														Parent: &yang.Entry{
															Name: "top-container",
															Parent: &yang.Entry{
																Name: "module",
															},
														},
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
		wantGoCompress: map[string]yangDirectory{
			"/module/top-container": {
				name: "TopContainer",
				fields: map[string]*yang.Entry{
					"leaf-one": {Name: "leaf-one", Type: &yang.YangType{Kind: yang.Yint8}},
					"leaf-two": {Name: "leaf-two", Type: &yang.YangType{Kind: yang.Yint8}},
				},
				path: []string{"", "module", "top-container"},
			},
		},
		wantGoUncompress: map[string]yangDirectory{
			"/module/top-container": {
				name: "Module_TopContainer",
				fields: map[string]*yang.Entry{
					"config": {Name: "config"},
					"state":  {Name: "state"},
				},
				path: []string{"", "module", "top-container"},
			},
			"/module/top-container/config": {
				name: "Module_TopContainer_Config",
				fields: map[string]*yang.Entry{
					"leaf-one": {Name: "leaf-one", Type: &yang.YangType{Kind: yang.Yint8}},
					"leaf-two": {Name: "leaf-two", Type: &yang.YangType{Kind: yang.Yint8}},
				},
				path: []string{"", "module", "top-container", "config"},
			},
			"/module/top-container/state": {
				name: "Module_TopContainer_State",
				fields: map[string]*yang.Entry{
					"leaf-one": {Name: "leaf-one", Type: &yang.YangType{Kind: yang.Yint8}},
					"leaf-two": {Name: "leaf-two", Type: &yang.YangType{Kind: yang.Yint8}},
				},
				path: []string{"", "module", "top-container", "state"},
			},
		},
	}, {
		name: "schema with list",
		in: []*yang.Entry{{
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
		}},
		wantGoCompress: map[string]yangDirectory{
			"/module/container/list": {
				name: "Container_List",
				fields: map[string]*yang.Entry{
					"key": {Name: "key", Type: &yang.YangType{Kind: yang.Ystring}},
				},
			},
		},
		wantGoUncompress: map[string]yangDirectory{
			"/module/container/list": {
				name: "Module_Container_List",
				fields: map[string]*yang.Entry{
					"key":    {Name: "key", Type: &yang.YangType{Kind: yang.Yleafref}},
					"config": {Name: "config"},
					"state":  {Name: "state"},
				},
			},
			"/module/container/list/config": {
				name: "Module_Container_List_Config",
				fields: map[string]*yang.Entry{
					"key": {Name: "key", Type: &yang.YangType{Kind: yang.Ystring}},
				},
			},
			"/module/container/list/state": {
				name: "Module_Container_List_State",
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
								Name: "case-one",
								Kind: yang.CaseEntry,
								Parent: &yang.Entry{
									Name: "choice-node",
									Kind: yang.ChoiceEntry,
									Parent: &yang.Entry{
										Name: "container",
									},
								},
								Dir: map[string]*yang.Entry{
									"second-container": {
										Name: "second-container",
										Parent: &yang.Entry{
											Name: "case-one",
											Kind: yang.CaseEntry,
											Parent: &yang.Entry{
												Name: "choice-node",
												Kind: yang.ChoiceEntry,
												Parent: &yang.Entry{
													Name:   "container",
													Parent: &yang.Entry{Name: "module"},
												},
											},
										},
										Dir: map[string]*yang.Entry{
											"config": {
												Name: "config",
												Parent: &yang.Entry{
													Name: "second-container",
													Parent: &yang.Entry{
														Name: "case-one",
														Kind: yang.CaseEntry,
														Parent: &yang.Entry{
															Name: "choice-node",
															Kind: yang.ChoiceEntry,
															Parent: &yang.Entry{
																Name:   "container",
																Parent: &yang.Entry{Name: "module"},
															},
														},
													},
												},
												Dir: map[string]*yang.Entry{"leaf-one": {Name: "leaf-one"}},
											},
										},
									},
								},
							},
							"case-two": {
								Name: "case-two",
								Kind: yang.CaseEntry,
								Parent: &yang.Entry{
									Name: "choice-node",
									Kind: yang.ChoiceEntry,
									Parent: &yang.Entry{
										Name: "container",
									},
								},
								Dir: map[string]*yang.Entry{
									"third-container": {
										Name: "third-container",
										Parent: &yang.Entry{
											Name: "case-two",
											Kind: yang.CaseEntry,
											Parent: &yang.Entry{
												Name: "choice-node",
												Kind: yang.ChoiceEntry,
												Parent: &yang.Entry{
													Name: "container",
													Parent: &yang.Entry{
														Name: "module",
													},
												},
											},
										},
										Dir: map[string]*yang.Entry{
											"config": {
												Name: "config",
												Parent: &yang.Entry{
													Name: "third-container",
													Parent: &yang.Entry{
														Name: "case-two",
														Kind: yang.CaseEntry,
														Parent: &yang.Entry{
															Name: "choice-node",
															Kind: yang.ChoiceEntry,
															Parent: &yang.Entry{
																Name: "container",
																Parent: &yang.Entry{
																	Name: "module",
																},
															},
														},
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
		wantGoCompress: map[string]yangDirectory{
			"/module/container": {
				name: "Container",
				fields: map[string]*yang.Entry{
					"second-container": {Name: "second-container"},
					"third-container":  {Name: "third-container"},
				},
			},
			// Since these are schema paths then we still have the choice node's name
			// here, we need to check that the processing recursed correctly into the
			// container.
			"/module/container/choice-node/case-one/second-container": {
				name:   "Container_SecondContainer",
				fields: map[string]*yang.Entry{"leaf-one": {Name: "leaf-one"}},
			},
			"/module/container/choice-node/case-two/third-container": {
				name:   "Container_ThirdContainer",
				fields: map[string]*yang.Entry{"leaf-two": {Name: "leaf-two"}},
			},
		},
		wantGoUncompress: map[string]yangDirectory{
			"/module/container": {
				name: "Module_Container",
				fields: map[string]*yang.Entry{
					"second-container": {Name: "second-container"},
					"third-container":  {Name: "third-container"},
				},
			},
			"/module/container/choice-node/case-one/second-container": {
				name:   "Module_Container_SecondContainer",
				fields: map[string]*yang.Entry{"config": {Name: "config"}},
			},
			"/module/container/choice-node/case-two/third-container": {
				name:   "Module_Container_ThirdContainer",
				fields: map[string]*yang.Entry{"config": {Name: "config"}},
			},
			"/module/container/choice-node/case-one/second-container/config": {
				name:   "Module_Container_SecondContainer_Config",
				fields: map[string]*yang.Entry{"leaf-one": {Name: "leaf-one"}},
			},
			"/module/container/choice-node/case-two/third-container/config": {
				name:   "Module_Container_ThirdContainer_Config",
				fields: map[string]*yang.Entry{"leaf-two": {Name: "leaf-two"}},
			},
		},
		wantProtoCompress: map[string]yangDirectory{
			"/module/container": {
				name: "Container",
				fields: map[string]*yang.Entry{
					"second-container": {Name: "second-container"},
					"third-container":  {Name: "third-container"},
				},
			},
			"/module/container/choice-node/case-one/second-container": {
				name:   "SecondContainer",
				fields: map[string]*yang.Entry{"leaf-one": {Name: "leaf-one"}},
			},
			"/module/container/choice-node/case-two/third-container": {
				name:   "ThirdContainer",
				fields: map[string]*yang.Entry{"leaf-two": {Name: "leaf-two"}},
			},
		},
		wantProtoUncompress: map[string]yangDirectory{
			"/module/container": {
				name: "Container",
				fields: map[string]*yang.Entry{
					"second-container": {Name: "second-container"},
					"third-container":  {Name: "third-container"},
				},
			},
			"/module/container/choice-node/case-one/second-container": {
				name:   "SecondContainer",
				fields: map[string]*yang.Entry{"config": {Name: "config"}},
			},
			"/module/container/choice-node/case-two/third-container": {
				name:   "ThirdContainer",
				fields: map[string]*yang.Entry{"config": {Name: "config"}},
			},
			"/module/container/choice-node/case-one/second-container/config": {
				name:   "Module_Container_SecondContainer_Config",
				fields: map[string]*yang.Entry{"leaf-one": {Name: "leaf-one"}},
			},
			"/module/container/choice-node/case-two/third-container/config": {
				name:   "Module_Container_ThirdContainer_Config",
				fields: map[string]*yang.Entry{"leaf-two": {Name: "leaf-two"}},
			},
		},
	}}

	for _, tt := range tests {
		combinations := []struct {
			lang     generatedLanguage        // Language to run the test for.
			compress bool                     // Whether path compression should be enabled.
			want     map[string]yangDirectory // Expected output of buildDirectoryDefinitions.
		}{{
			lang:     golang,
			compress: true,
			want:     tt.wantGoCompress,
		}, {
			lang:     golang,
			compress: false,
			want:     tt.wantGoUncompress,
		}, {
			lang:     protobuf,
			compress: true,
			want:     tt.wantProtoCompress,
		}, {
			lang:     protobuf,
			compress: false,
			want:     tt.wantProtoUncompress,
		}}

		for _, c := range combinations {
			// If this isn't a test case that has been defined then we skip it.
			if c.want == nil {
				continue
			}

			cg := NewYANGCodeGenerator(&GeneratorConfig{
				CompressOCPaths: c.compress,
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
				findMappableEntities(inc, structs, enums, []string{}, c.compress)
			}

			got, errs := cg.state.buildDirectoryDefinitions(structs, cg.Config.CompressOCPaths, cg.Config.GenerateFakeRoot, c.lang)
			if errs != nil {
				t.Errorf("%s: buildDirectoryDefinitions(CompressOCPaths: %v): could not build struct defs: %v", tt.name, c.compress, errs)
				continue
			}

			for gotName, gotDir := range got {
				wantDir, ok := c.want[gotName]
				if !ok {
					t.Errorf("%s: buildDirectoryDefinitions(CompressOCPaths: %v): could not find expected struct %s, got: %v, want: %v",
						tt.name, c.compress, gotName, got, c.want)
					continue
				}

				for fieldk, fieldv := range wantDir.fields {
					cmpfield, ok := gotDir.fields[fieldk]
					if !ok {
						t.Errorf("%s: buildDirectoryDefinitions(CompressOCPaths: %v): could not find expected field %s in %s, got: %v",
							tt.name, c.compress, fieldk, gotName, gotDir.fields)
						continue
					}

					if fieldv.Name != cmpfield.Name {
						t.Errorf("%s: buildDirectoryDefinitions(CompressOCPaths: %v): field %s of %s did not have expected name, got: %v, want: %v",
							tt.name, c.compress, fieldk, gotName, fieldv.Name, cmpfield.Name)
					}

					if fieldv.Type != nil && cmpfield.Type != nil {
						if fieldv.Type.Kind != cmpfield.Type.Kind {
							t.Errorf("%s: buildDirectoryDefinitions(CompressOCPaths: %v): field %s of %s did not have expected type got: %s, want: %s",
								tt.name, c.compress, fieldk, gotName, fieldv.Type.Kind, cmpfield.Type.Kind)
						}
					}

				}

				if wantDir.path != nil && !reflect.DeepEqual(wantDir.path, gotDir.path) {
					t.Errorf("%s (%v): %s did not have matching path, got: %v, want: %v", tt.name, c.compress, gotName, gotDir.path, wantDir.path)
				}

				if wantDir.name != wantDir.name {
					t.Errorf("%s (%v): %s did not have matching name, got: %v, want: %v", tt.name, c.compress, gotDir.path, gotDir.name, wantDir.name)
				}
			}
		}
	}
}
