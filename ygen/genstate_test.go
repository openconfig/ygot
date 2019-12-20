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
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/kylelemons/godebug/pretty"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/genutil"
)

// TestFindEnumSet tests the findEnumSet function, ensuring that it performs
// deduplication of re-used identities, and re-used typedefs. For inline
// definitions, the enumerations should be duplicated. Tests are performed with
// compression set to both true and false.
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
	}, {
		name: "two enums within the same directory, different definitions",
		in: map[string]*yang.Entry{
			"/container/config/enumeration-leaf": {
				Name: "enumeration-leaf",
				Type: &yang.YangType{
					Name: "enumeration",
					Enum: &yang.EnumType{},
				},
				Node: &yang.Enum{
					Name: "enumeration-leaf",
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
			"/container/config/enumeration-leaf-two": {
				Name: "enumeration-leaf-two",
				Type: &yang.YangType{
					Name: "enumeration",
					Enum: &yang.EnumType{},
				},
				Node: &yang.Enum{
					Name: "enumeration-leaf-two",
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
			"/container/state/enumeration-leaf-two": {
				Name: "enumeration-leaf-two",
				Type: &yang.YangType{
					Name: "enumeration",
					Enum: &yang.EnumType{},
				},
				Node: &yang.Enum{
					Name: "enumeration-leaf-two",
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
			"BaseModule_Container_EnumerationLeaf": {
				name: "BaseModule_Container_EnumerationLeaf",
				entry: &yang.Entry{
					Name: "enumeration-leaf",
					Type: &yang.YangType{
						Enum: &yang.EnumType{},
					},
				},
			},
			"BaseModule_Container_EnumerationLeafTwo": {
				name: "BaseModule_Container_EnumerationLeafTwo",
				entry: &yang.Entry{
					Name: "enumeration-leaf-two",
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
			"BaseModule_Container_State_EnumerationLeafTwo": {
				name: "BaseModule_Container_State_EnumerationLeafTwo",
				entry: &yang.Entry{
					Name: "enumeration-leaf-two",
					Type: &yang.YangType{
						Enum: &yang.EnumType{},
					},
				},
			},
			"BaseModule_Container_Config_EnumerationLeafTwo": {
				name: "BaseModule_Container_Config_EnumerationLeafTwo",
				entry: &yang.Entry{
					Name: "enumeration-leaf-two",
					Type: &yang.YangType{
						Enum: &yang.EnumType{},
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
			state := newEnumGenState()
			entries, errs := state.findEnumSet(tt.in, compressed, tt.inOmitUnderscores)

			if (errs != nil) != tt.wantErr {
				t.Errorf("%s findEnumSet(%v, %v): did not get expected error when extracting enums, got: %v (len %d), wanted err: %v", tt.name, tt.in, compressed, errs, len(errs), tt.wantErr)
				continue
			}

			for k, want := range wanted {
				got, ok := entries[k]
				if !ok {
					t.Errorf("%s findEnumSet(compressEnabled: %v): could not find expected entry, got: %v, want: %s", tt.name, compressed, entries, k)
					continue
				}

				if want.entry.Name != got.entry.Name {
					j, _ := json.Marshal(got)
					t.Errorf("%s findEnumSet(compressEnabled: %v): extracted entry has wrong name: got %s, want: %s (%s)", tt.name,
						compressed, got.entry.Name, want.entry.Name, string(j))
				}

				if want.entry.Type.IdentityBase != nil {
					// Check the identity's base if this was an identityref.
					if want.entry.Type.IdentityBase.Name != got.entry.Type.IdentityBase.Name {
						t.Errorf("%s findEnumSet(compressEnabled: %v): found identity %s, has wrong base, got: %v, want: %v", tt.name,
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
			s := newGoGenState(nil)
			if out := s.goStructName(tt.inElement, compress, false); out != expected {
				t.Errorf("%s (compress: %v): shortName output invalid - got: %s, want: %s", tt.name, compress, out, expected)
			}
		}
	}
}

func TestBuildDirectoryDefinitions(t *testing.T) {
	tests := []struct {
		name                                    string
		in                                      []*yang.Entry
		checkPath                               bool // checkPath says whether the Directories' Path field should be checked.
		wantGoCompress                          map[string]*Directory
		wantGoCompressPreferOperationalState    map[string]*Directory
		wantGoUncompress                        map[string]*Directory
		wantGoCompressStateExcluded             map[string]*Directory
		wantGoUncompressStateExcluded           map[string]*Directory
		wantProtoCompress                       map[string]*Directory
		wantProtoCompressPreferOperationalState map[string]*Directory
		wantProtoUncompress                     map[string]*Directory
		wantProtoCompressStateExcluded          map[string]*Directory
		wantProtoUncompressStateExcluded        map[string]*Directory
	}{{
		name: "basic struct generation test",
		in: []*yang.Entry{{
			Name: "module",
			Dir: map[string]*yang.Entry{
				"s1": {
					Name:   "s1",
					Parent: &yang.Entry{Name: "module"},
					Kind:   yang.DirectoryEntry,
					Dir: map[string]*yang.Entry{
						"config": {
							Name: "config",
							Kind: yang.DirectoryEntry,
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
							Name:   "state",
							Config: yang.TSFalse,
							Kind:   yang.DirectoryEntry,
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
		checkPath: true,
		wantGoCompress: map[string]*Directory{
			"/module/s1": {
				Name: "S1",
				Fields: map[string]*yang.Entry{
					"l1": {Name: "l1", Type: &yang.YangType{Kind: yang.Ystring}},
					"l2": {Name: "l2", Type: &yang.YangType{Kind: yang.Yint8}},
					"l3": {Name: "l3", Type: &yang.YangType{Kind: yang.Yint32}},
				},
				Path: []string{"", "module", "s1"},
			},
		},
		wantGoCompressPreferOperationalState: map[string]*Directory{
			"/module/s1": {
				Name: "S1",
				Fields: map[string]*yang.Entry{
					"l1": {Name: "l1", Type: &yang.YangType{Kind: yang.Yint8}},
					"l2": {Name: "l2", Type: &yang.YangType{Kind: yang.Yint8}},
					"l3": {Name: "l3", Type: &yang.YangType{Kind: yang.Yint32}},
				},
				Path: []string{"", "module", "s1"},
			},
		},
		wantGoCompressStateExcluded: map[string]*Directory{
			"/module/s1": {
				Name: "S1",
				Fields: map[string]*yang.Entry{
					"l1": {Name: "l1", Type: &yang.YangType{Kind: yang.Ystring}},
					"l2": {Name: "l2", Type: &yang.YangType{Kind: yang.Yint8}},
				},
				Path: []string{"", "module", "s1"},
			},
		},
		wantGoUncompress: map[string]*Directory{
			"/module/s1": {
				Name: "Module_S1",
				Fields: map[string]*yang.Entry{
					"config": {Name: "config"},
					"state":  {Name: "state"},
				},
				Path: []string{"", "module", "s1"},
			},
			"/module/s1/config": {
				Name: "Module_S1_Config",
				Fields: map[string]*yang.Entry{
					"l1": {Name: "l1", Type: &yang.YangType{Kind: yang.Ystring}},
					"l2": {Name: "l2", Type: &yang.YangType{Kind: yang.Yint8}},
				},
				Path: []string{"", "module", "s1", "config"},
			},
			"/module/s1/state": {
				Name: "Module_S1_State",
				Fields: map[string]*yang.Entry{
					"l1": {Name: "l1", Type: &yang.YangType{Kind: yang.Yint8}},
					"l2": {Name: "l2", Type: &yang.YangType{Kind: yang.Yint8}},
					"l3": {Name: "l3", Type: &yang.YangType{Kind: yang.Yint32}},
				},
				Path: []string{"", "module", "s1", "state"},
			},
		},
		wantGoUncompressStateExcluded: map[string]*Directory{
			"/module/s1": {
				Name: "Module_S1",
				Fields: map[string]*yang.Entry{
					"config": {Name: "config"},
				},
				Path: []string{"", "module", "s1"},
			},
			"/module/s1/config": {
				Name: "Module_S1_Config",
				Fields: map[string]*yang.Entry{
					"l1": {Name: "l1", Type: &yang.YangType{Kind: yang.Ystring}},
					"l2": {Name: "l2", Type: &yang.YangType{Kind: yang.Yint8}},
				},
				Path: []string{"", "module", "s1", "config"},
			},
		},
		wantProtoCompress: map[string]*Directory{
			"/module/s1": {
				Name: "S1",
				Fields: map[string]*yang.Entry{
					"l1": {Name: "l1", Type: &yang.YangType{Kind: yang.Ystring}},
					"l2": {Name: "l2", Type: &yang.YangType{Kind: yang.Yint8}},
					"l3": {Name: "l3", Type: &yang.YangType{Kind: yang.Yint32}},
				},
				Path: []string{"", "module", "s1"},
			},
		},
		wantProtoCompressPreferOperationalState: map[string]*Directory{
			"/module/s1": {
				Name: "S1",
				Fields: map[string]*yang.Entry{
					"l1": {Name: "l1", Type: &yang.YangType{Kind: yang.Yint8}},
					"l2": {Name: "l2", Type: &yang.YangType{Kind: yang.Yint8}},
					"l3": {Name: "l3", Type: &yang.YangType{Kind: yang.Yint32}},
				},
				Path: []string{"", "module", "s1"},
			},
		},
		wantProtoCompressStateExcluded: map[string]*Directory{
			"/module/s1": {
				Name: "S1",
				Fields: map[string]*yang.Entry{
					"l1": {Name: "l1", Type: &yang.YangType{Kind: yang.Ystring}},
					"l2": {Name: "l2", Type: &yang.YangType{Kind: yang.Yint8}},
				},
				Path: []string{"", "module", "s1"},
			},
		},
		wantProtoUncompress: map[string]*Directory{
			"/module/s1": {
				Name: "S1",
				Fields: map[string]*yang.Entry{
					"config": {Name: "config"},
					"state":  {Name: "state"},
				},
				Path: []string{"", "module", "s1"},
			},
			"/module/s1/config": {
				Name: "Config",
				Fields: map[string]*yang.Entry{
					"l1": {Name: "l1", Type: &yang.YangType{Kind: yang.Ystring}},
					"l2": {Name: "l2", Type: &yang.YangType{Kind: yang.Yint8}},
				},
				Path: []string{"", "module", "s1", "config"},
			},
			"/module/s1/state": {
				Name: "State",
				Fields: map[string]*yang.Entry{
					"l1": {Name: "l1", Type: &yang.YangType{Kind: yang.Yint8}},
					"l2": {Name: "l2", Type: &yang.YangType{Kind: yang.Yint8}},
					"l3": {Name: "l3", Type: &yang.YangType{Kind: yang.Yint32}},
				},
				Path: []string{"", "module", "s1", "state"},
			},
		},
		wantProtoUncompressStateExcluded: map[string]*Directory{
			"/module/s1": {
				Name: "S1",
				Fields: map[string]*yang.Entry{
					"config": {Name: "config"},
				},
				Path: []string{"", "module", "s1"},
			},
			"/module/s1/config": {
				Name: "Config",
				Fields: map[string]*yang.Entry{
					"l1": {Name: "l1", Type: &yang.YangType{Kind: yang.Ystring}},
					"l2": {Name: "l2", Type: &yang.YangType{Kind: yang.Yint8}},
				},
				Path: []string{"", "module", "s1", "config"},
			},
		},
	}, {
		name: "struct test with state only fields",
		in: []*yang.Entry{{
			Name: "module",
			Dir: map[string]*yang.Entry{
				"s1": {
					Name:   "s1",
					Parent: &yang.Entry{Name: "module"},
					Kind:   yang.DirectoryEntry,
					Dir: map[string]*yang.Entry{
						"read-only": {
							Name:   "read-only",
							Type:   &yang.YangType{Kind: yang.Ystring},
							Config: yang.TSFalse,
						},
						"read-write": {
							Name:   "read-write",
							Type:   &yang.YangType{Kind: yang.Ystring},
							Config: yang.TSTrue,
						},
					},
				},
			},
		}},
		wantGoUncompress: map[string]*Directory{
			"/module/s1": {
				Name: "Module_S1",
				Fields: map[string]*yang.Entry{
					"read-only":  {Name: "read-only", Type: &yang.YangType{Kind: yang.Ystring}},
					"read-write": {Name: "read-write", Type: &yang.YangType{Kind: yang.Ystring}},
				},
			},
		},
		wantGoUncompressStateExcluded: map[string]*Directory{
			"/module/s1": {
				Name: "Module_S1",
				Fields: map[string]*yang.Entry{
					"read-write": {Name: "read-write", Type: &yang.YangType{Kind: yang.Ystring}},
				},
			},
		},
		wantProtoUncompress: map[string]*Directory{
			"/module/s1": {
				Name: "S1",
				Fields: map[string]*yang.Entry{
					"read-only":  {Name: "read-only", Type: &yang.YangType{Kind: yang.Ystring}},
					"read-write": {Name: "read-write", Type: &yang.YangType{Kind: yang.Ystring}},
				},
			},
		},
		wantProtoUncompressStateExcluded: map[string]*Directory{
			"/module/s1": {
				Name: "S1",
				Fields: map[string]*yang.Entry{
					"read-write": {Name: "read-write", Type: &yang.YangType{Kind: yang.Ystring}},
				},
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
						Kind:   yang.DirectoryEntry,
						Parent: &yang.Entry{Name: "module"},
						Dir: map[string]*yang.Entry{
							"config": {
								Name: "config",
								Kind: yang.DirectoryEntry,
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
								Name:   "state",
								Kind:   yang.DirectoryEntry,
								Config: yang.TSFalse,
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
								Kind: yang.DirectoryEntry,
								Parent: &yang.Entry{
									Name: "s1",
									Parent: &yang.Entry{
										Name: "module",
									},
								},
								Dir: map[string]*yang.Entry{
									"inner-container": {
										Name: "inner-container",
										Kind: yang.DirectoryEntry,
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
												Kind: yang.DirectoryEntry,
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
												Name:   "state",
												Kind:   yang.DirectoryEntry,
												Config: yang.TSFalse,
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
		checkPath: true,
		wantGoCompress: map[string]*Directory{
			"/module/s1": {
				Name: "S1",
				Fields: map[string]*yang.Entry{
					"l1":              {Name: "l1", Type: &yang.YangType{Kind: yang.Ystring}},
					"l2":              {Name: "l2", Type: &yang.YangType{Kind: yang.Yint8}},
					"l3":              {Name: "l3", Type: &yang.YangType{Kind: yang.Yint32}},
					"outer-container": {Name: "outer-container"},
				},
				Path: []string{"", "module", "s1"},
			},
			"/module/s1/outer-container": {
				Name: "S1_OuterContainer",
				Fields: map[string]*yang.Entry{
					"inner-container": {Name: "inner-container"},
				},
				Path: []string{"", "module", "s1", "outer-container"},
			},
			"/module/s1/outer-container/inner-container": {
				Name: "S1_OuterContainer_InnerContainer",
				Fields: map[string]*yang.Entry{
					"inner-leaf":       {Name: "inner-leaf", Type: &yang.YangType{Kind: yang.Ystring}},
					"inner-state-leaf": {Name: "inner-state-leaf", Type: &yang.YangType{Kind: yang.Yint8}},
				},
				Path: []string{"", "module", "s1", "outer-container", "inner-container"},
			},
		},
		wantGoCompressPreferOperationalState: map[string]*Directory{
			"/module/s1": {
				Name: "S1",
				Fields: map[string]*yang.Entry{
					"l1":              {Name: "l1", Type: &yang.YangType{Kind: yang.Yint8}},
					"l2":              {Name: "l2", Type: &yang.YangType{Kind: yang.Yint8}},
					"l3":              {Name: "l3", Type: &yang.YangType{Kind: yang.Yint32}},
					"outer-container": {Name: "outer-container"},
				},
				Path: []string{"", "module", "s1"},
			},
			"/module/s1/outer-container": {
				Name: "S1_OuterContainer",
				Fields: map[string]*yang.Entry{
					"inner-container": {Name: "inner-container"},
				},
				Path: []string{"", "module", "s1", "outer-container"},
			},
			"/module/s1/outer-container/inner-container": {
				Name: "S1_OuterContainer_InnerContainer",
				Fields: map[string]*yang.Entry{
					"inner-leaf":       {Name: "inner-leaf", Type: &yang.YangType{Kind: yang.Ystring}},
					"inner-state-leaf": {Name: "inner-state-leaf", Type: &yang.YangType{Kind: yang.Yint8}},
				},
				Path: []string{"", "module", "s1", "outer-container", "inner-container"},
			},
		},
		wantGoUncompress: map[string]*Directory{
			"/module/s1": {
				Name: "Module_S1",
				Fields: map[string]*yang.Entry{
					"config":          {Name: "config"},
					"state":           {Name: "state"},
					"outer-container": {Name: "outer-container"},
				},
				Path: []string{"", "module", "s1"},
			},
			"/module/s1/config": {
				Name: "Module_S1_Config",
				Fields: map[string]*yang.Entry{
					"l1": {Name: "l1", Type: &yang.YangType{Kind: yang.Ystring}},
					"l2": {Name: "l2", Type: &yang.YangType{Kind: yang.Yint8}},
				},
				Path: []string{"", "module", "s1", "config"},
			},
			"/module/s1/state": {
				Name: "Module_S1_State",
				Fields: map[string]*yang.Entry{
					"l1": {Name: "l1", Type: &yang.YangType{Kind: yang.Yint8}},
					"l2": {Name: "l2", Type: &yang.YangType{Kind: yang.Yint8}},
					"l3": {Name: "l3", Type: &yang.YangType{Kind: yang.Yint32}},
				},
				Path: []string{"", "module", "s1", "state"},
			},
			"/module/s1/outer-container": {
				Name:   "Module_S1_OuterContainer",
				Fields: map[string]*yang.Entry{"inner-container": {Name: "inner-container"}},
				Path:   []string{"", "module", "s1", "outer-container"},
			},
			"/module/s1/outer-container/inner-container": {
				Name: "Module_S1_OuterContainer_InnerContainer",
				Fields: map[string]*yang.Entry{
					"config": {Name: "config"},
					"state":  {Name: "state"},
				},
				Path: []string{"", "module", "s1", "outer-container", "inner-container"},
			},
			"/module/s1/outer-container/inner-container/config": {
				Name: "Module_S1_OuterContainer_InnerContainer_Config",
				Fields: map[string]*yang.Entry{
					"inner-leaf": {Name: "inner-leaf", Type: &yang.YangType{Kind: yang.Ystring}},
				},
				Path: []string{"", "module", "s1", "outer-container", "inner-container", "config"},
			},
			"/module/s1/outer-container/inner-container/state": {
				Name: "Module_S1_OuterContainer_InnerContainer_State",
				Fields: map[string]*yang.Entry{
					"inner-leaf":       {Name: "inner-leaf", Type: &yang.YangType{Kind: yang.Ystring}},
					"inner-state-leaf": {Name: "inner-state-leaf", Type: &yang.YangType{Kind: yang.Yint8}},
				},
				Path: []string{"", "module", "s1", "outer-container", "inner-container", "state"},
			},
		},
		wantProtoCompress: map[string]*Directory{
			"/module/s1": {
				Name: "S1",
				Fields: map[string]*yang.Entry{
					"l1":              {Name: "l1", Type: &yang.YangType{Kind: yang.Ystring}},
					"l2":              {Name: "l2", Type: &yang.YangType{Kind: yang.Yint8}},
					"l3":              {Name: "l3", Type: &yang.YangType{Kind: yang.Yint32}},
					"outer-container": {Name: "outer-container"},
				},
				Path: []string{"", "module", "s1"},
			},
			"/module/s1/outer-container": {
				Name: "OuterContainer",
				Fields: map[string]*yang.Entry{
					"inner-container": {Name: "inner-container"},
				},
				Path: []string{"", "module", "s1", "outer-container"},
			},
			"/module/s1/outer-container/inner-container": {
				Name: "InnerContainer",
				Fields: map[string]*yang.Entry{
					"inner-leaf":       {Name: "inner-leaf", Type: &yang.YangType{Kind: yang.Ystring}},
					"inner-state-leaf": {Name: "inner-state-leaf", Type: &yang.YangType{Kind: yang.Yint8}},
				},
				Path: []string{"", "module", "s1", "outer-container", "inner-container"},
			},
		},
		wantProtoCompressPreferOperationalState: map[string]*Directory{
			"/module/s1": {
				Name: "S1",
				Fields: map[string]*yang.Entry{
					"l1":              {Name: "l1", Type: &yang.YangType{Kind: yang.Yint8}},
					"l2":              {Name: "l2", Type: &yang.YangType{Kind: yang.Yint8}},
					"l3":              {Name: "l3", Type: &yang.YangType{Kind: yang.Yint32}},
					"outer-container": {Name: "outer-container"},
				},
				Path: []string{"", "module", "s1"},
			},
			"/module/s1/outer-container": {
				Name: "OuterContainer",
				Fields: map[string]*yang.Entry{
					"inner-container": {Name: "inner-container"},
				},
				Path: []string{"", "module", "s1", "outer-container"},
			},
			"/module/s1/outer-container/inner-container": {
				Name: "InnerContainer",
				Fields: map[string]*yang.Entry{
					"inner-leaf":       {Name: "inner-leaf", Type: &yang.YangType{Kind: yang.Ystring}},
					"inner-state-leaf": {Name: "inner-state-leaf", Type: &yang.YangType{Kind: yang.Yint8}},
				},
				Path: []string{"", "module", "s1", "outer-container", "inner-container"},
			},
		},
		wantProtoUncompress: map[string]*Directory{
			"/module/s1": {
				Name: "S1",
				Fields: map[string]*yang.Entry{
					"config":          {Name: "config"},
					"state":           {Name: "state"},
					"outer-container": {Name: "outer-container"},
				},
				Path: []string{"", "module", "s1"},
			},
			"/module/s1/config": {
				Name: "Config",
				Fields: map[string]*yang.Entry{
					"l1": {Name: "l1", Type: &yang.YangType{Kind: yang.Ystring}},
					"l2": {Name: "l2", Type: &yang.YangType{Kind: yang.Yint8}},
				},
				Path: []string{"", "module", "s1", "config"},
			},
			"/module/s1/state": {
				Name: "State",
				Fields: map[string]*yang.Entry{
					"l1": {Name: "l1", Type: &yang.YangType{Kind: yang.Yint8}},
					"l2": {Name: "l2", Type: &yang.YangType{Kind: yang.Yint8}},
					"l3": {Name: "l3", Type: &yang.YangType{Kind: yang.Yint32}},
				},
				Path: []string{"", "module", "s1", "state"},
			},
			"/module/s1/outer-container": {
				Name:   "OuterContainer",
				Fields: map[string]*yang.Entry{"inner-container": {Name: "inner-container"}},
				Path:   []string{"", "module", "s1", "outer-container"},
			},
			"/module/s1/outer-container/inner-container": {
				Name: "InnerContainer",
				Fields: map[string]*yang.Entry{
					"config": {Name: "config"},
					"state":  {Name: "state"},
				},
				Path: []string{"", "module", "s1", "outer-container", "inner-container"},
			},
			"/module/s1/outer-container/inner-container/config": {
				Name: "Config",
				Fields: map[string]*yang.Entry{
					"inner-leaf": {Name: "inner-leaf", Type: &yang.YangType{Kind: yang.Ystring}},
				},
				Path: []string{"", "module", "s1", "outer-container", "inner-container", "config"},
			},
			"/module/s1/outer-container/inner-container/state": {
				Name: "State",
				Fields: map[string]*yang.Entry{
					"inner-leaf":       {Name: "inner-leaf", Type: &yang.YangType{Kind: yang.Ystring}},
					"inner-state-leaf": {Name: "inner-state-leaf", Type: &yang.YangType{Kind: yang.Yint8}},
				},
				Path: []string{"", "module", "s1", "outer-container", "inner-container", "state"},
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
						Kind:   yang.DirectoryEntry,
						Parent: &yang.Entry{Name: "module"},
						Dir: map[string]*yang.Entry{
							"config": {
								Name: "config",
								Kind: yang.DirectoryEntry,
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
								Kind: yang.DirectoryEntry,
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
		checkPath: true,
		wantGoCompress: map[string]*Directory{
			"/module/top-container": {
				Name: "TopContainer",
				Fields: map[string]*yang.Entry{
					"leaf-one": {Name: "leaf-one", Type: &yang.YangType{Kind: yang.Yint8}},
					"leaf-two": {Name: "leaf-two", Type: &yang.YangType{Kind: yang.Yint8}},
				},
				Path: []string{"", "module", "top-container"},
			},
		},
		wantGoCompressPreferOperationalState: map[string]*Directory{
			"/module/top-container": {
				Name: "TopContainer",
				Fields: map[string]*yang.Entry{
					"leaf-one": {Name: "leaf-one", Type: &yang.YangType{Kind: yang.Yint8}},
					"leaf-two": {Name: "leaf-two", Type: &yang.YangType{Kind: yang.Yint8}},
				},
				Path: []string{"", "module", "top-container"},
			},
		},
		wantGoUncompress: map[string]*Directory{
			"/module/top-container": {
				Name: "Module_TopContainer",
				Fields: map[string]*yang.Entry{
					"config": {Name: "config"},
					"state":  {Name: "state"},
				},
				Path: []string{"", "module", "top-container"},
			},
			"/module/top-container/config": {
				Name: "Module_TopContainer_Config",
				Fields: map[string]*yang.Entry{
					"leaf-one": {Name: "leaf-one", Type: &yang.YangType{Kind: yang.Yint8}},
					"leaf-two": {Name: "leaf-two", Type: &yang.YangType{Kind: yang.Yint8}},
				},
				Path: []string{"", "module", "top-container", "config"},
			},
			"/module/top-container/state": {
				Name: "Module_TopContainer_State",
				Fields: map[string]*yang.Entry{
					"leaf-one": {Name: "leaf-one", Type: &yang.YangType{Kind: yang.Yint8}},
					"leaf-two": {Name: "leaf-two", Type: &yang.YangType{Kind: yang.Yint8}},
				},
				Path: []string{"", "module", "top-container", "state"},
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
					Kind:     yang.DirectoryEntry,
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
							Kind:   yang.DirectoryEntry,
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
							Kind:   yang.DirectoryEntry,
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
		wantGoCompress: map[string]*Directory{
			"/module/container/list": {
				Name: "Container_List",
				Fields: map[string]*yang.Entry{
					"key": {Name: "key", Type: &yang.YangType{Kind: yang.Ystring}},
				},
			},
		},
		wantGoUncompress: map[string]*Directory{
			"/module/container/list": {
				Name: "Module_Container_List",
				Fields: map[string]*yang.Entry{
					"key":    {Name: "key", Type: &yang.YangType{Kind: yang.Yleafref}},
					"config": {Name: "config"},
					"state":  {Name: "state"},
				},
			},
			"/module/container/list/config": {
				Name: "Module_Container_List_Config",
				Fields: map[string]*yang.Entry{
					"key": {Name: "key", Type: &yang.YangType{Kind: yang.Ystring}},
				},
			},
			"/module/container/list/state": {
				Name: "Module_Container_List_State",
				Fields: map[string]*yang.Entry{
					"key": {Name: "key", Type: &yang.YangType{Kind: yang.Ystring}},
				},
			},
		},
	}, {
		name: "schema with choice around container",
		in: []*yang.Entry{
			{
				Name: "module",
				Dir: map[string]*yang.Entry{

					"container": {
						Name:   "container",
						Kind:   yang.DirectoryEntry,
						Parent: &yang.Entry{Name: "module"},
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
												Parent: &yang.Entry{
													Name: "module",
												},
											},
										},
										Dir: map[string]*yang.Entry{
											"second-container": {
												Name: "second-container",
												Kind: yang.DirectoryEntry,
												Parent: &yang.Entry{
													Name: "case-one",
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
														Kind: yang.DirectoryEntry,
														Parent: &yang.Entry{
															Name: "second-container",
															Parent: &yang.Entry{
																Name: "case-one",
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
												Parent: &yang.Entry{
													Name: "module",
												},
											},
										},
										Dir: map[string]*yang.Entry{
											"third-container": {
												Name: "third-container",
												Kind: yang.DirectoryEntry,
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
														Kind: yang.DirectoryEntry,
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
			},
		},
		wantGoCompress: map[string]*Directory{
			"/module/container": {
				Name: "Container",
				Fields: map[string]*yang.Entry{
					"second-container": {Name: "second-container"},
					"third-container":  {Name: "third-container"},
				},
			},
			// Since these are schema paths then we still have the choice node's name
			// here, we need to check that the processing recursed correctly into the
			// container.
			"/module/container/choice-node/case-one/second-container": {
				Name:   "Container_SecondContainer",
				Fields: map[string]*yang.Entry{"leaf-one": {Name: "leaf-one"}},
			},
			"/module/container/choice-node/case-two/third-container": {
				Name:   "Container_ThirdContainer",
				Fields: map[string]*yang.Entry{"leaf-two": {Name: "leaf-two"}},
			},
		},
		wantGoUncompress: map[string]*Directory{
			"/module/container": {
				Name: "Module_Container",
				Fields: map[string]*yang.Entry{
					"second-container": {Name: "second-container"},
					"third-container":  {Name: "third-container"},
				},
			},
			"/module/container/choice-node/case-one/second-container": {
				Name:   "Module_Container_SecondContainer",
				Fields: map[string]*yang.Entry{"config": {Name: "config"}},
			},
			"/module/container/choice-node/case-two/third-container": {
				Name:   "Module_Container_ThirdContainer",
				Fields: map[string]*yang.Entry{"config": {Name: "config"}},
			},
			"/module/container/choice-node/case-one/second-container/config": {
				Name:   "Module_Container_SecondContainer_Config",
				Fields: map[string]*yang.Entry{"leaf-one": {Name: "leaf-one"}},
			},
			"/module/container/choice-node/case-two/third-container/config": {
				Name:   "Module_Container_ThirdContainer_Config",
				Fields: map[string]*yang.Entry{"leaf-two": {Name: "leaf-two"}},
			},
		},
		wantProtoCompress: map[string]*Directory{
			"/module/container": {
				Name: "Container",
				Fields: map[string]*yang.Entry{
					"second-container": {Name: "second-container"},
					"third-container":  {Name: "third-container"},
				},
			},
			"/module/container/choice-node/case-one/second-container": {
				Name:   "SecondContainer",
				Fields: map[string]*yang.Entry{"leaf-one": {Name: "leaf-one"}},
			},
			"/module/container/choice-node/case-two/third-container": {
				Name:   "ThirdContainer",
				Fields: map[string]*yang.Entry{"leaf-two": {Name: "leaf-two"}},
			},
		},
		wantProtoUncompress: map[string]*Directory{
			"/module/container": {
				Name: "Container",
				Fields: map[string]*yang.Entry{
					"second-container": {Name: "second-container"},
					"third-container":  {Name: "third-container"},
				},
			},
			"/module/container/choice-node/case-one/second-container": {
				Name:   "SecondContainer",
				Fields: map[string]*yang.Entry{"config": {Name: "config"}},
			},
			"/module/container/choice-node/case-two/third-container": {
				Name:   "ThirdContainer",
				Fields: map[string]*yang.Entry{"config": {Name: "config"}},
			},
			"/module/container/choice-node/case-one/second-container/config": {
				Name:   "Config",
				Fields: map[string]*yang.Entry{"leaf-one": {Name: "leaf-one"}},
			},
			"/module/container/choice-node/case-two/third-container/config": {
				Name:   "Config",
				Fields: map[string]*yang.Entry{"leaf-two": {Name: "leaf-two"}},
			},
		},
	}}

	// Simple helper functions for error messages
	fieldNames := func(dir *Directory) []string {
		names := []string{}
		for k := range dir.Fields {
			names = append(names, k)
		}
		return names
	}

	langName := func(l generatedLanguage) string {
		languageName := map[generatedLanguage]string{
			golang:   "Go",
			protobuf: "Proto",
		}
		return languageName[l]
	}

	for _, tt := range tests {
		combinations := []struct {
			lang              generatedLanguage         // lang is the language to run the test for.
			compressBehaviour genutil.CompressBehaviour // compressBehaviour indicates whether path compression should be enabled and whether state fields should be excluded.
			excludeState      bool                      // excludeState indicates whether config false values should be excluded.
			want              map[string]*Directory     // want is the expected output of buildDirectoryDefinitions.
		}{{
			lang:              golang,
			compressBehaviour: genutil.PreferIntendedConfig,
			want:              tt.wantGoCompress,
		}, {
			lang:              golang,
			compressBehaviour: genutil.PreferOperationalState,
			want:              tt.wantGoCompressPreferOperationalState,
		}, {
			lang:              golang,
			compressBehaviour: genutil.Uncompressed,
			want:              tt.wantGoUncompress,
		}, {
			lang:              protobuf,
			compressBehaviour: genutil.PreferIntendedConfig,
			want:              tt.wantProtoCompress,
		}, {
			lang:              protobuf,
			compressBehaviour: genutil.PreferOperationalState,
			want:              tt.wantProtoCompressPreferOperationalState,
		}, {
			lang:              protobuf,
			compressBehaviour: genutil.Uncompressed,
			want:              tt.wantProtoUncompress,
		}, {
			lang:              golang,
			compressBehaviour: genutil.ExcludeDerivedState,
			want:              tt.wantGoCompressStateExcluded,
		}, {
			lang:              golang,
			compressBehaviour: genutil.UncompressedExcludeDerivedState,
			want:              tt.wantGoUncompressStateExcluded,
		}, {
			lang:              protobuf,
			compressBehaviour: genutil.ExcludeDerivedState,
			want:              tt.wantProtoCompressStateExcluded,
		}, {
			lang:              protobuf,
			compressBehaviour: genutil.UncompressedExcludeDerivedState,
			want:              tt.wantProtoUncompressStateExcluded,
		}}

		for _, c := range combinations {
			// If this isn't a test case that has been defined then we skip it.
			if c.want == nil {
				continue
			}

			t.Run(fmt.Sprintf("%s:buildDirectoryDefinitions(CompressBehaviour:%v,Language:%s,excludeState:%v)", tt.name, c.compressBehaviour, langName(c.lang), c.excludeState), func(t *testing.T) {
				st, err := buildSchemaTree(tt.in)
				if err != nil {
					t.Fatalf("buildSchemaTree(%v), got unexpected err: %v", tt.in, err)
				}
				gogen := newGoGenState(st)
				protogen := newProtoGenState(st)

				structs := make(map[string]*yang.Entry)
				enums := make(map[string]*yang.Entry)

				var errs []error
				for _, inc := range tt.in {
					// Always provide a nil set of modules to findMappableEntities since this
					// is only used to skip elements.
					errs = append(errs, findMappableEntities(inc, structs, enums, []string{}, c.compressBehaviour.CompressEnabled(), []*yang.Entry{})...)
				}
				if errs != nil {
					t.Fatalf("findMappableEntities(%v, %v, %v, nil, %v, nil): got unexpected error, want: nil, got: %v", tt.in, structs, enums, c.compressBehaviour.CompressEnabled(), err)
				}

				var got map[string]*Directory
				switch c.lang {
				case golang:
					got, errs = gogen.buildDirectoryDefinitions(structs, c.compressBehaviour, false)
				case protobuf:
					got, errs = protogen.buildDirectoryDefinitions(structs, c.compressBehaviour)
				}
				if errs != nil {
					t.Fatal(errs)
				}

				// This checks the "Name" and maybe "Path" attributes of the output Directories.
				ignoreFields := []string{"Entry", "Fields", "ListAttr", "IsFakeRoot"}
				if !tt.checkPath {
					ignoreFields = append(ignoreFields, "Path")
				}
				if diff := cmp.Diff(c.want, got, cmpopts.IgnoreFields(Directory{}, ignoreFields...)); diff != "" {
					t.Errorf("(-want +got):\n%s", diff)
				}

				// Verify certain fields of the "Fields" attribute -- there are too many fields to ignore to use cmp.Diff for comparison.
				for gotName, gotDir := range got {
					// Note that any missing or extra Directories would've been caught with the previous check.
					wantDir := c.want[gotName]
					if len(gotDir.Fields) != len(wantDir.Fields) {
						t.Fatalf("Did not get expected set of fields for %s, got: %v, want: %v", gotName, fieldNames(gotDir), fieldNames(wantDir))
					}
					for fieldk, fieldv := range wantDir.Fields {
						cmpfield, ok := gotDir.Fields[fieldk]
						if !ok {
							t.Errorf("Could not find expected field %s in %s, got: %v", fieldk, gotName, gotDir.Fields)
							continue // Fatal error for this field only.
						}

						if fieldv.Name != cmpfield.Name {
							t.Errorf("Field %s of %s did not have expected name, got: %v, want: %v", fieldk, gotName, cmpfield.Name, fieldv.Name)
						}

						if fieldv.Type != nil && cmpfield.Type != nil {
							if fieldv.Type.Kind != cmpfield.Type.Kind {
								t.Errorf("Field %s of %s did not have expected type got: %s, want: %s", fieldk, gotName, cmpfield.Type.Kind, fieldv.Type.Kind)
							}
						}
					}
				}
			})
		}
	}
}

func TestResolveLeafrefTargetType(t *testing.T) {
	tests := []struct {
		name           string
		inPath         string
		inContextEntry *yang.Entry
		inEntries      []*yang.Entry
		want           *yang.Entry
		wantErr        bool
	}{{
		name:   "simple test with leafref with absolute leafref",
		inPath: "/parent/child/a",
		inContextEntry: &yang.Entry{
			Name: "b",
			Type: &yang.YangType{
				Kind: yang.Yleafref,
				Path: "/parent/child/a",
			},
			Parent: &yang.Entry{
				Name: "child",
				Parent: &yang.Entry{
					Name:   "parent",
					Parent: &yang.Entry{Name: "module"},
				},
			},
		},
		inEntries: []*yang.Entry{
			{
				Name: "parent",
				Dir: map[string]*yang.Entry{
					"child": {
						Name: "child",
						Dir: map[string]*yang.Entry{
							"a": {
								Name: "a",
								Type: &yang.YangType{
									Kind: yang.Ystring,
								},
								Parent: &yang.Entry{
									Name: "child",
									Parent: &yang.Entry{
										Name:   "parent",
										Parent: &yang.Entry{Name: "module"},
									},
								},
							},
							"b": {
								Name: "b",
								Type: &yang.YangType{
									Kind: yang.Yleafref,
									Path: "/parent/child/a",
								},
								Parent: &yang.Entry{
									Name: "child",
									Parent: &yang.Entry{
										Name:   "parent",
										Parent: &yang.Entry{Name: "module"},
									},
								},
							},
						},
						Parent: &yang.Entry{
							Name:   "parent",
							Parent: &yang.Entry{Name: "module"},
						},
					},
				},
				Parent: &yang.Entry{Name: "module"},
			},
		},
		want: &yang.Entry{
			Name: "a",
			Type: &yang.YangType{
				Kind: yang.Ystring,
			},
			Parent: &yang.Entry{
				Name: "child",
				Parent: &yang.Entry{
					Name:   "parent",
					Parent: &yang.Entry{Name: "module"},
				},
			},
		},
	}}

	for _, tt := range tests {
		// Since we are outside of the build of a module, need to initialise
		// the schematree.
		st, err := buildSchemaTree(tt.inEntries)
		if err != nil {
			t.Errorf("%s: buildSchemaTree(%v): got unexpected error: %v", tt.name, tt.inEntries, err)
		}
		got, err := st.resolveLeafrefTarget(tt.inPath, tt.inContextEntry)
		if err != nil {
			if !tt.wantErr {
				t.Errorf("%s: resolveLeafrefTargetPath(%v, %v): got unexpected error: %v", tt.name, tt.inPath, tt.inContextEntry, err)
			}
			continue
		}

		if tt.wantErr {
			t.Errorf("%s: resolveLeafrefTargetPath(%v, %v): did not get expected error", tt.name, tt.inPath, tt.inContextEntry)
			continue
		}

		if diff := pretty.Compare(got, tt.want); diff != "" {
			t.Errorf("%s: resolveLeafrefTargetPath(%v, %v): did not get expected entry, diff(-got,+want):\n%s", tt.name, tt.inPath, tt.inContextEntry, diff)
		}
	}
}
