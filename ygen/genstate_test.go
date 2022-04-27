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
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/genutil"
)

func TestOrderedUnionTypes(t *testing.T) {
	tests := []struct {
		desc string
		in   *MappedType
		want []string
	}{{
		desc: "union type with 2 elements",
		in: &MappedType{
			NativeType: "A_Union",
			UnionTypes: map[string]int{
				"Binary":  1,
				"float64": 2,
			},
		},
		want: []string{
			"Binary",
			"float64",
		},
	}, {
		desc: "union type with 3 elements",
		in: &MappedType{
			NativeType: "A_Union",
			UnionTypes: map[string]int{
				"uint64":  3,
				"float64": 2,
				"Binary":  1,
			},
		},
		want: []string{
			"Binary",
			"float64",
			"uint64",
		},
	}, {
		desc: "non-union type",
		in: &MappedType{
			NativeType: "string",
		},
		want: nil,
	}, {
		desc: "union type with a single element",
		in: &MappedType{
			NativeType: "string",
			UnionTypes: map[string]int{
				"string": 0,
			},
		},
		want: []string{
			"string",
		},
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if diff := cmp.Diff(tt.in.OrderedUnionTypes(), tt.want); diff != "" {
				t.Errorf("(-got, +want):\n%s", diff)
			}
		})
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
				ShadowedFields: map[string]*yang.Entry{
					"l1": {Name: "l1", Type: &yang.YangType{Kind: yang.Yint8}},
					"l2": {Name: "l2", Type: &yang.YangType{Kind: yang.Yint8}},
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
				ShadowedFields: map[string]*yang.Entry{
					"l1": {Name: "l1", Type: &yang.YangType{Kind: yang.Ystring}},
					"l2": {Name: "l2", Type: &yang.YangType{Kind: yang.Yint8}},
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
				ShadowedFields: map[string]*yang.Entry{
					"l1": {Name: "l1", Type: &yang.YangType{Kind: yang.Yint8}},
					"l2": {Name: "l2", Type: &yang.YangType{Kind: yang.Yint8}},
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
				ShadowedFields: map[string]*yang.Entry{
					"l1": {Name: "l1", Type: &yang.YangType{Kind: yang.Ystring}},
					"l2": {Name: "l2", Type: &yang.YangType{Kind: yang.Yint8}},
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
				ShadowedFields: map[string]*yang.Entry{
					"l1": {Name: "l1", Type: &yang.YangType{Kind: yang.Yint8}},
					"l2": {Name: "l2", Type: &yang.YangType{Kind: yang.Yint8}},
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
				ShadowedFields: map[string]*yang.Entry{
					"inner-leaf": {Name: "inner-leaf", Type: &yang.YangType{Kind: yang.Ystring}},
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
				ShadowedFields: map[string]*yang.Entry{
					"l1": {Name: "l1", Type: &yang.YangType{Kind: yang.Ystring}},
					"l2": {Name: "l2", Type: &yang.YangType{Kind: yang.Yint8}},
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
				ShadowedFields: map[string]*yang.Entry{
					"inner-leaf": {Name: "inner-leaf", Type: &yang.YangType{Kind: yang.Ystring}},
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
				ShadowedFields: map[string]*yang.Entry{
					"l1": {Name: "l1", Type: &yang.YangType{Kind: yang.Yint8}},
					"l2": {Name: "l2", Type: &yang.YangType{Kind: yang.Yint8}},
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
				ShadowedFields: map[string]*yang.Entry{
					"inner-leaf": {Name: "inner-leaf", Type: &yang.YangType{Kind: yang.Ystring}},
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
				ShadowedFields: map[string]*yang.Entry{
					"l1": {Name: "l1", Type: &yang.YangType{Kind: yang.Ystring}},
					"l2": {Name: "l2", Type: &yang.YangType{Kind: yang.Yint8}},
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
				ShadowedFields: map[string]*yang.Entry{
					"inner-leaf": {Name: "inner-leaf", Type: &yang.YangType{Kind: yang.Ystring}},
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
				ShadowedFields: map[string]*yang.Entry{
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
				ShadowedFields: map[string]*yang.Entry{
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
										Name: "state",
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
				ShadowedFields: map[string]*yang.Entry{
					"key": {Name: "key", Type: &yang.YangType{Kind: yang.Ystring}},
				},
				ListAttr: &YangListAttr{
					Keys: map[string]*ListKey{
						"key": {
							Name: "Key",
							LangType: &MappedType{
								NativeType: "string",
								ZeroValue:  `""`,
							},
						},
					},
					ListKeyYANGNames: []string{"key"},
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
				ListAttr: &YangListAttr{
					Keys: map[string]*ListKey{
						"key": {
							Name: "Key",
							LangType: &MappedType{
								NativeType: "string",
								ZeroValue:  `""`,
							},
						},
					},
					ListKeyYANGNames: []string{"key"},
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
	shadowedFieldNames := func(dir *Directory) []string {
		names := []string{}
		for k := range dir.ShadowedFields {
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
				gogen := NewGoLangMapper(true)
				gogen.SetSchemaTree(st)
				protogen := newProtoGenState(st, nil)

				structs := make(map[string]*yang.Entry)
				enums := make(map[string]*yang.Entry)

				var errs []error
				for _, inc := range tt.in {
					// Always provide a nil set of modules to findMappableEntities since this
					// is only used to skip elements.
					errs = append(errs, findMappableEntities(inc, structs, enums, []string{}, c.compressBehaviour.CompressEnabled(), []*yang.Entry{})...)
				}
				if errs != nil {
					t.Fatalf("findMappableEntities(%v, %v, %v, nil, %v, nil): got unexpected error, want: nil, got: %v", tt.in, structs, enums, c.compressBehaviour.CompressEnabled(), errs)
				}

				var got map[string]*Directory
				switch c.lang {
				case golang:
					got, errs = buildDirectoryDefinitions(gogen, structs, IROptions{
						ParseOptions: ParseOpts{
							SkipEnumDeduplication: false,
						},
						TransformationOptions: TransformationOpts{
							CompressBehaviour:                    c.compressBehaviour,
							GenerateFakeRoot:                     false,
							ShortenEnumLeafNames:                 true,
							UseDefiningModuleForTypedefEnumNames: true,
							EnumOrgPrefixesToTrim:                nil,
						},
						NestedDirectories:                    false,
						AbsoluteMapPaths:                     false,
						AppendEnumSuffixForSimpleUnionEnums:  true,
						UseConsistentNamesForProtoUnionEnums: false,
					})
				case protobuf:
					got, errs = buildDirectoryDefinitions(protogen, structs, IROptions{
						ParseOptions: ParseOpts{
							SkipEnumDeduplication: false,
						},
						TransformationOptions: TransformationOpts{
							CompressBehaviour:                    c.compressBehaviour,
							GenerateFakeRoot:                     false,
							ShortenEnumLeafNames:                 true,
							UseDefiningModuleForTypedefEnumNames: true,
							EnumOrgPrefixesToTrim:                nil,
						},
						NestedDirectories:                    true,
						AbsoluteMapPaths:                     true,
						AppendEnumSuffixForSimpleUnionEnums:  true,
						UseConsistentNamesForProtoUnionEnums: true,
					})
				}
				if errs != nil {
					t.Fatal(errs)
				}

				// This checks the "Name" and maybe "Path" attributes of the output Directories.
				ignoreFields := []string{"Entry", "Fields", "ShadowedFields", "IsFakeRoot"}
				if !tt.checkPath {
					ignoreFields = append(ignoreFields, "Path")
				}
				if diff := cmp.Diff(c.want, got, cmpopts.IgnoreFields(Directory{}, ignoreFields...), cmpopts.IgnoreFields(YangListAttr{}, "KeyElems"), cmpopts.EquateEmpty()); diff != "" {
					t.Errorf("(-want +got):\n%s", diff)
				}

				// Verify certain fields of the "Fields" attribute -- there are too many fields to ignore to use cmp.Diff for comparison.
				for gotName, gotDir := range got {
					// Note that any missing or extra Directories would've been caught with the previous check.
					wantDir, ok := c.want[gotName]
					if !ok {
						t.Errorf("got directory keyed at %q, did not expect this", gotName)
						continue
					}
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
					if len(gotDir.ShadowedFields) != len(wantDir.ShadowedFields) {
						t.Fatalf("Did not get expected set of shadowed fields for %s, got: %v, want: %v", gotName, shadowedFieldNames(gotDir), shadowedFieldNames(wantDir))
					}
					for fieldk, fieldv := range wantDir.ShadowedFields {
						cmpfield, ok := gotDir.ShadowedFields[fieldk]
						if !ok {
							t.Errorf("Could not find expected shadowed field %s in %s, got: %v", fieldk, gotName, gotDir.Fields)
							continue // Fatal error for this field only.
						}

						if fieldv.Name != cmpfield.Name {
							t.Errorf("Shadowed field %s of %s did not have expected name, got: %v, want: %v", fieldk, gotName, cmpfield.Name, fieldv.Name)
						}

						if fieldv.Type != nil && cmpfield.Type != nil {
							if fieldv.Type.Kind != cmpfield.Type.Kind {
								t.Errorf("Shadowed field %s of %s did not have expected type got: %s, want: %s", fieldk, gotName, cmpfield.Type.Kind, fieldv.Type.Kind)
							}
						}
					}
				}
			})
		}
	}
}

// enumMapFromEntries recursively finds enumerated values from a slice of
// entries and returns an enumMap. The input enumMap is intended for
// findEnumSet.
func enumMapFromEntries(entries []*yang.Entry) map[string]*yang.Entry {
	enumMap := map[string]*yang.Entry{}
	for _, e := range entries {
		addEnumsToEnumMap(e, enumMap)
	}
	return enumMap
}

// enumMapFromEntries recursively finds enumerated values from a slice of
// resolveTypeArgs and returns an enumMap. The input enumMap is intended for
// findEnumSet.
func enumMapFromArgs(args []resolveTypeArgs) map[string]*yang.Entry {
	enumMap := map[string]*yang.Entry{}
	for _, a := range args {
		addEnumsToEnumMap(a.contextEntry, enumMap)
	}
	return enumMap
}

// enumMapFromEntries recursively finds enumerated values from an entry and
// returns an enumMap. The input enumMap is intended for findEnumSet.
func enumMapFromEntry(entry *yang.Entry) map[string]*yang.Entry {
	enumMap := map[string]*yang.Entry{}
	addEnumsToEnumMap(entry, enumMap)
	return enumMap
}

// enumMapFromEntries recursively finds enumerated values from a directory and
// returns an enumMap. The input enumMap is intended for findEnumSet.
func enumMapFromDirectory(dir *Directory) map[string]*yang.Entry {
	enumMap := map[string]*yang.Entry{}
	addEnumsToEnumMap(dir.Entry, enumMap)
	for _, e := range dir.Fields {
		addEnumsToEnumMap(e, enumMap)
	}
	return enumMap
}

// addEnumsToEnumMap recursively finds enumerated values and adds them to the
// input enumMap. The input enumMap is intended for findEnumSet, so that tests
// that need generated enumerated names have an easy time generating them, and
// subsequently adding them to their generated state during setup.
func addEnumsToEnumMap(entry *yang.Entry, enumMap map[string]*yang.Entry) {
	if entry == nil {
		return
	}
	if e := mappableLeaf(entry); e != nil {
		enumMap[entry.Path()] = e
	}
	for _, e := range entry.Dir {
		addEnumsToEnumMap(e, enumMap)
	}
}

// TestBuildListKey takes an input yang.Entry and ensures that the correct YangListAttr
// struct is returned representing the keys of the list e.
func TestBuildListKey(t *testing.T) {
	tests := []struct {
		name                    string        // name is the test identifier.
		in                      *yang.Entry   // in is the yang.Entry of the test list.
		inCompress              bool          // inCompress is a boolean indicating whether CompressOCPaths should be true/false.
		inEntries               []*yang.Entry // inEntries is used to provide context entries in the schema, particularly where a leafref key is used.
		inEnumEntries           []*yang.Entry // inEnumEntries is used to add more state for findEnumSet to test enum name generation.
		inSkipEnumDedup         bool          // inSkipEnumDedup says whether to dedup identical enums encountered in the models.
		inResolveKeyNameFuncNil bool          // inResolveKeyNameFuncNil specifies whether the key name function is not provided.
		want                    YangListAttr  // want is the expected YangListAttr output.
		wantErr                 bool          // wantErr is a boolean indicating whether errors are expected from buildListKeys
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
				"keyleaf": {
					Type: &yang.YangType{Kind: yang.Yidentityref},
				},
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
			Keys: map[string]*ListKey{
				"keyleaf": {
					Name: "Keyleaf",
					LangType: &MappedType{
						NativeType: "string",
					},
				},
			},
			KeyElems: []*yang.Entry{
				{
					Name: "keyleaf",
				},
			},
		},
	}, {
		name: "basic list enum key",
		in: &yang.Entry{
			Name:     "list",
			ListAttr: &yang.ListAttr{},
			Key:      "keyleaf",
			Dir: map[string]*yang.Entry{
				"keyleaf": {
					Name: "keyleaf",
					Type: &yang.YangType{
						Name: "enumeration",
						Enum: &yang.EnumType{},
						Kind: yang.Yenum,
					},
					Node: &yang.Enum{
						Name: "enumeration",
						Parent: &yang.Grouping{
							Name: "foo",
							Parent: &yang.Module{
								Name: "base-module",
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
			},
		},
		inCompress: true,
		want: YangListAttr{
			Keys: map[string]*ListKey{
				"keyleaf": {
					Name: "Keyleaf",
					LangType: &MappedType{
						NativeType: "E_Container_Keyleaf",
					},
				},
			},
			KeyElems: []*yang.Entry{
				{
					Name: "keyleaf",
					Type: &yang.YangType{
						Name: "enumeration",
						Enum: &yang.EnumType{},
						Kind: yang.Yenum,
					},
				},
			},
		},
	}, {
		name: "multiple list keys",
		in: &yang.Entry{
			Name:     "list",
			ListAttr: &yang.ListAttr{},
			Key:      "k1 k2",
			Dir: map[string]*yang.Entry{
				"k1": {Name: "k1", Type: &yang.YangType{Kind: yang.Ystring}},
				"k2": {Name: "k2", Type: &yang.YangType{Kind: yang.Ystring}},
			},
		},
		want: YangListAttr{
			Keys: map[string]*ListKey{
				"k1": {
					Name: "K1",
					LangType: &MappedType{
						NativeType: "string",
					},
				},
				"k2": {
					Name: "K2",
					LangType: &MappedType{
						NativeType: "string",
					},
				},
			},
			KeyElems: []*yang.Entry{
				{Name: "k1", Type: &yang.YangType{Kind: yang.Ystring}},
				{Name: "k2", Type: &yang.YangType{Kind: yang.Ystring}},
			},
		},
	}, {
		name: "multiple list keys - double spacing",
		in: &yang.Entry{
			Name:     "list",
			ListAttr: &yang.ListAttr{},
			Key:      "k1  k2",
			Dir: map[string]*yang.Entry{
				"k1": {Name: "k1", Type: &yang.YangType{Kind: yang.Ystring}},
				"k2": {Name: "k2", Type: &yang.YangType{Kind: yang.Ystring}},
			},
		},
		want: YangListAttr{
			Keys: map[string]*ListKey{
				"k1": {
					Name: "K1",
					LangType: &MappedType{
						NativeType: "string",
					},
				},
				"k2": {
					Name: "K2",
					LangType: &MappedType{
						NativeType: "string",
					},
				},
			},
			KeyElems: []*yang.Entry{
				{Name: "k1", Type: &yang.YangType{Kind: yang.Ystring}},
				{Name: "k2", Type: &yang.YangType{Kind: yang.Ystring}},
			},
		},
	}, {
		name: "multiple list keys - newlines",
		in: &yang.Entry{
			Name:     "list",
			ListAttr: &yang.ListAttr{},
			Key:      "k1  \nk2",
			Dir: map[string]*yang.Entry{
				"k1": {Name: "k1", Type: &yang.YangType{Kind: yang.Ystring}},
				"k2": {Name: "k2", Type: &yang.YangType{Kind: yang.Ystring}},
			},
		},
		want: YangListAttr{
			Keys: map[string]*ListKey{
				"k1": {
					Name: "K1",
					LangType: &MappedType{
						NativeType: "string",
					},
				},
				"k2": {
					Name: "K2",
					LangType: &MappedType{
						NativeType: "string",
					},
				},
			},
			KeyElems: []*yang.Entry{
				{Name: "k1", Type: &yang.YangType{Kind: yang.Ystring}},
				{Name: "k2", Type: &yang.YangType{Kind: yang.Ystring}},
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
			Keys: map[string]*ListKey{
				"keyleafref": {
					Name: "Keyleafref",
					LangType: &MappedType{
						NativeType: "string",
					},
				},
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
			Keys: map[string]*ListKey{
				"key1": {
					Name: "Key1",
					LangType: &MappedType{
						NativeType: "string",
					},
				},
				"key2": {
					Name: "Key2",
					LangType: &MappedType{
						NativeType: "string",
					},
				},
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
			Keys: map[string]*ListKey{
				"key1": {
					Name: "Key1",
					LangType: &MappedType{
						NativeType: "string",
					},
				},
				"key2": {
					Name: "Key2",
					LangType: &MappedType{
						NativeType: "int8",
					},
				},
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
			Keys: map[string]*ListKey{
				"keyleafref": {
					Name: "Keyleafref",
					LangType: &MappedType{
						NativeType: "string",
					},
				},
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
			Keys: map[string]*ListKey{
				"keyleafref": {
					Name: "Keyleafref",
					LangType: &MappedType{
						NativeType: "string",
					},
				},
			},
			KeyElems: []*yang.Entry{
				{
					Name: "keyleafref",
					Type: &yang.YangType{Kind: yang.Ystring},
				},
			},
		},
	}, {
		name: "list enum key -- already seen",
		in: &yang.Entry{
			Name:     "list",
			ListAttr: &yang.ListAttr{},
			Key:      "keyleaf",
			Dir: map[string]*yang.Entry{
				"keyleaf": {
					Name: "keyleaf",
					Type: &yang.YangType{
						Name: "enumeration",
						Enum: &yang.EnumType{},
						Kind: yang.Yenum,
					},
					Node: &yang.Enum{
						Name: "enumeration",
						Parent: &yang.Grouping{
							Name: "foo",
							Parent: &yang.Module{
								Name: "base-module",
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
			},
		},
		inEnumEntries: []*yang.Entry{{
			Name: "enum-leaf-lexicographically-earlier",
			Type: &yang.YangType{
				Name: "enumeration",
				Enum: &yang.EnumType{},
				Kind: yang.Yenum,
			},
			Node: &yang.Enum{
				Name: "enumeration",
				Parent: &yang.Grouping{
					Name: "foo",
					Parent: &yang.Module{
						Name: "base-module",
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
		inCompress: true,
		want: YangListAttr{
			Keys: map[string]*ListKey{
				"keyleaf": {
					Name: "Keyleaf",
					LangType: &MappedType{
						NativeType: "E_Container_EnumLeafLexicographicallyEarlier",
					},
				},
			},
			KeyElems: []*yang.Entry{
				{
					Name: "keyleaf",
					Type: &yang.YangType{
						Name: "enumeration",
						Enum: &yang.EnumType{},
						Kind: yang.Yenum,
					},
				},
			},
		},
	}, {
		name: "list enum key -- already seen but skip enum dedup",
		in: &yang.Entry{
			Name:     "list",
			ListAttr: &yang.ListAttr{},
			Key:      "keyleaf",
			Dir: map[string]*yang.Entry{
				"keyleaf": {
					Name: "keyleaf",
					Type: &yang.YangType{
						Name: "enumeration",
						Enum: &yang.EnumType{},
						Kind: yang.Yenum,
					},
					Node: &yang.Enum{
						Name: "enumeration",
						Parent: &yang.Grouping{
							Name: "foo",
							Parent: &yang.Module{
								Name: "base-module",
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
			},
		},
		inEnumEntries: []*yang.Entry{{
			Name: "enum-leaf-lexicographically-earlier",
			Type: &yang.YangType{
				Name: "enumeration",
				Enum: &yang.EnumType{},
				Kind: yang.Yenum,
			},
			Node: &yang.Enum{
				Name: "enumeration",
				Parent: &yang.Grouping{
					Name: "foo",
					Parent: &yang.Module{
						Name: "base-module",
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
		inCompress:      true,
		inSkipEnumDedup: true,
		want: YangListAttr{
			Keys: map[string]*ListKey{
				"keyleaf": {
					Name: "Keyleaf",
					LangType: &MappedType{
						NativeType: "E_Container_Keyleaf",
					},
				},
			},
			KeyElems: []*yang.Entry{
				{
					Name: "keyleaf",
					Type: &yang.YangType{
						Name: "enumeration",
						Enum: &yang.EnumType{},
						Kind: yang.Yenum,
					},
				},
			},
		},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var st *schemaTree
			if tt.inEntries != nil {
				var err error
				if st, err = buildSchemaTree(tt.inEntries); err != nil {
					t.Fatalf("%s: buildSchemaTree(%v), could not build tree: %v", tt.name, tt.inEntries, err)
				}
			}
			enumMap := enumMapFromEntries(tt.inEnumEntries)
			addEnumsToEnumMap(tt.in, enumMap)
			enumSet, _, errs := findEnumSet(enumMap, tt.inCompress, false, tt.inSkipEnumDedup, true, true, true, true, nil)
			if errs != nil {
				if !tt.wantErr {
					t.Errorf("findEnumSet failed: %v", errs)
				}
				return
			}
			s := NewGoLangMapper(true)
			s.SetEnumSet(enumSet)
			s.SetSchemaTree(st)

			compressBehaviour := genutil.Uncompressed
			if tt.inCompress {
				compressBehaviour = genutil.PreferIntendedConfig
			}

			got, err := buildListKey(tt.in, s, IROptions{
				ParseOptions: ParseOpts{
					SkipEnumDeduplication: tt.inSkipEnumDedup,
				},
				TransformationOptions: TransformationOpts{
					CompressBehaviour:                    compressBehaviour,
					GenerateFakeRoot:                     true,
					ShortenEnumLeafNames:                 true,
					UseDefiningModuleForTypedefEnumNames: true,
					EnumOrgPrefixesToTrim:                nil,
					EnumerationsUseUnderscores:           true,
				},
				NestedDirectories:                    false,
				AbsoluteMapPaths:                     false,
				AppendEnumSuffixForSimpleUnionEnums:  true,
				UseConsistentNamesForProtoUnionEnums: false,
			})
			if err != nil && !tt.wantErr {
				t.Errorf("%s: could not build list key successfully %v", tt.name, err)
			}

			if err == nil && tt.wantErr {
				t.Errorf("%s: did not get expected error", tt.name)
			}

			if tt.wantErr || got == nil {
				return
			}

			for name, gtype := range got.Keys {
				elem, ok := tt.want.Keys[name]
				if !ok {
					t.Errorf("%s: could not find key %s", tt.name, name)
					continue
				}
				if gtype == nil {
					t.Errorf("%s: key %s is nil", tt.name, name)
					continue
				}
				if elem.LangType.NativeType != gtype.LangType.NativeType {
					t.Errorf("%s: key %s had the wrong type %s, want %s", tt.name, name, gtype.LangType.NativeType, elem.LangType.NativeType)
				}
			}
		})
	}
}
