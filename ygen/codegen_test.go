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
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/openconfig/goyang/pkg/yang"

	"github.com/pmezard/go-difflib/difflib"
)

const (
	// TestRoot is the root of the test directory such that this is not
	// repeated when referencing files.
	TestRoot string = ""
)

// TestFindMappableEntities tests the extraction of elements that are to be mapped
// into Go code from a YANG schema.
func TestFindMappableEntities(t *testing.T) {
	tests := []struct {
		name          string      // name is an identifier for the test.
		in            *yang.Entry // in is the yang.Entry corresponding to the YANG root element.
		inSkipModules []string    // inSkipModules is a slice of strings indicating modules to be skipped.
		// wantCompressed is a map keyed by the string "structs" or "enums" which contains a slice
		// of the YANG identifiers for the corresponding mappable entities that should be
		// found. wantCompressed is the set that are expected when CompressOCPaths is set
		// to true,
		wantCompressed map[string][]string
		// wantUncompressed is a map of the same form as wantCompressed. It is the expected
		// result when CompressOCPaths is set to false.
		wantUncompressed map[string][]string
	}{{
		name: "base-test",
		in: &yang.Entry{
			Name: "module",
			Dir: map[string]*yang.Entry{
				"base": {
					Name: "base",
					Dir: map[string]*yang.Entry{
						"config": {
							Name: "config",
							Dir:  map[string]*yang.Entry{},
						},
						"state": {
							Name: "state",
							Dir:  map[string]*yang.Entry{},
						},
					},
				},
			},
		},
		wantCompressed: map[string][]string{
			"structs": {"base"},
			"enums":   {},
		},
		wantUncompressed: map[string][]string{
			"structs": {"base", "config", "state"},
			"enums":   {},
		},
	}, {
		name: "enum-test",
		in: &yang.Entry{
			Name: "module",
			Dir: map[string]*yang.Entry{
				"base": {
					Name: "base",
					Dir: map[string]*yang.Entry{
						"config": {
							Name: "config",
							Dir: map[string]*yang.Entry{
								"enumleaf": {
									Name: "enumleaf",
									Type: &yang.YangType{
										Kind: yang.Yenum,
									},
								},
							},
						},
						"state": {
							Name: "state",
							Dir: map[string]*yang.Entry{
								"enumleaf": {
									Name: "enumleaf",
									Type: &yang.YangType{
										Kind: yang.Yenum,
									},
								},
							},
						},
					},
				},
			},
		},
		wantCompressed: map[string][]string{
			"structs": {"base"},
			"enums":   {"enumleaf"},
		},
		wantUncompressed: map[string][]string{
			"structs": {"base", "config", "state"},
			"enums":   {"enumleaf"},
		},
	}, {
		name: "skip module",
		in: &yang.Entry{
			Name: "module",
			Dir: map[string]*yang.Entry{
				"ignored-container": {
					Name: "ignored-container",
					Dir:  map[string]*yang.Entry{},
				},
			},
		},
		inSkipModules: []string{"module"},
		wantCompressed: map[string][]string{
			"structs": {},
			"enums":   {},
		},
		wantUncompressed: map[string][]string{
			"structs": {},
			"enums":   {},
		},
	}, {
		name: "surrounding container for list at root",
		in: &yang.Entry{
			Name: "module",
			Dir: map[string]*yang.Entry{
				"surrounding-container": {
					Name: "surrounding-container",
					Dir: map[string]*yang.Entry{
						"child-list": {
							Name:     "child-list",
							Dir:      map[string]*yang.Entry{},
							ListAttr: &yang.ListAttr{},
						},
					},
				},
			},
		},
		wantCompressed: map[string][]string{
			"structs": {"child-list"},
		},
		wantUncompressed: map[string][]string{
			"structs": {"surrounding-container", "child-list"},
		},
	}, {
		name: "choice/case at root",
		in: &yang.Entry{
			Name: "module",
			Dir: map[string]*yang.Entry{
				"choice": {
					Name: "choice",
					Kind: yang.ChoiceEntry,
					Dir: map[string]*yang.Entry{
						"case": {
							Name: "case",
							Kind: yang.CaseEntry,
							Dir: map[string]*yang.Entry{
								"container": {
									Name: "container",
									Dir:  map[string]*yang.Entry{},
								},
							},
						},
					},
				},
			},
		},
		wantCompressed: map[string][]string{
			"structs": {"container"},
		},
		wantUncompressed: map[string][]string{
			"structs": {"container"},
		},
	}, {
		name: "enumerated value within a union leaf",
		in: &yang.Entry{
			Name: "module",
			Dir: map[string]*yang.Entry{
				"leaf": {
					Name: "leaf",
					Type: &yang.YangType{
						Kind: yang.Yunion,
						Type: []*yang.YangType{
							{Kind: yang.Yenum},
						},
					},
				},
			},
		},
		wantCompressed:   map[string][]string{"enums": {"leaf"}},
		wantUncompressed: map[string][]string{"enums": {"leaf"}},
	}, {
		name: "identityref value within a union leaf",
		in: &yang.Entry{
			Name: "module",
			Dir: map[string]*yang.Entry{
				"leaf": {
					Name: "leaf",
					Type: &yang.YangType{
						Name: "union",
						Kind: yang.Yunion,
						Type: []*yang.YangType{
							{Kind: yang.Yidentityref},
							{Kind: yang.Yenum},
						},
					},
				},
			},
		},
		wantCompressed:   map[string][]string{"enums": {"leaf"}},
		wantUncompressed: map[string][]string{"enums": {"leaf"}},
	}, {
		name: "enumeration within a typedef which is a union",
		in: &yang.Entry{
			Name: "module",
			Dir: map[string]*yang.Entry{
				"leaf": {
					Name: "leaf",
					Type: &yang.YangType{
						Name: "newtype",
						Kind: yang.Yunion,
						Type: []*yang.YangType{
							{Kind: yang.Yenum},
							{Kind: yang.Yenum},
						},
					},
				},
			},
		},
		wantCompressed:   map[string][]string{"enums": {"leaf"}},
		wantUncompressed: map[string][]string{"enums": {"leaf"}},
	}, {
		name: "enumerated value within a choice that has a child",
		in: &yang.Entry{
			Name: "module",
			Dir: map[string]*yang.Entry{
				"choice": {
					Name: "choice",
					Kind: yang.ChoiceEntry,
					Dir: map[string]*yang.Entry{
						"case": {
							Name: "case",
							Kind: yang.CaseEntry,
							Dir: map[string]*yang.Entry{
								"container": {
									Name: "container",
									Dir: map[string]*yang.Entry{
										"choice-case-container-leaf": {
											Name: "choice-case-container-leaf",
											Type: &yang.YangType{Kind: yang.Yenum},
										},
									},
								},
							},
						},
						"case2": {
							Name: "case2",
							Kind: yang.CaseEntry,
							Dir: map[string]*yang.Entry{
								"choice-case2-leaf": {
									Name: "choice-case2-leaf",
									Type: &yang.YangType{Kind: yang.Yenum},
								},
							},
						},
						"direct": {
							Name: "direct",
							Type: &yang.YangType{Kind: yang.Yenum},
						},
					},
				},
			},
		},
		wantCompressed:   map[string][]string{"enums": {"choice-case-container-leaf", "choice-case2-leaf", "direct"}},
		wantUncompressed: map[string][]string{"enums": {"choice-case-container-leaf", "choice-case2-leaf", "direct"}},
	}}

	for _, tt := range tests {
		testSpec := map[bool]map[string][]string{
			true:  tt.wantCompressed,
			false: tt.wantUncompressed,
		}

		for compress, expected := range testSpec {
			structs := make(map[string]*yang.Entry)
			enums := make(map[string]*yang.Entry)

			cg := NewYANGCodeGenerator(&GeneratorConfig{
				CompressOCPaths: compress,
				ExcludeModules:  tt.inSkipModules,
			})

			cg.findMappableEntities(tt.in, structs, enums)

			structOut := make(map[string]bool)
			enumOut := make(map[string]bool)
			for _, o := range structs {
				structOut[o.Name] = true
			}
			for _, e := range enums {
				enumOut[e.Name] = true
			}

			for _, e := range expected["structs"] {
				if !structOut[e] {
					t.Errorf("%s findMappableEntities(CompressOCPaths: %v): struct %s was not found in %v\n", tt.name, compress, e, structOut)
				}
			}

			for _, e := range expected["enums"] {
				if !enumOut[e] {
					t.Errorf("%s findMappableEntities(CompressOCPaths: %v): enum %s was not found in %v\n", tt.name, compress, e, enumOut)
				}
			}
		}
	}
}

// TestBuildStructDefinitions tests the struct definition builder to ensure that the relevant
// entities are extracted from the input YANG.
func TestBuildStructDefinitions(t *testing.T) {
	tests := []struct {
		name           string
		in             []*yang.Entry
		wantCompress   map[string]yangStruct
		wantUncompress map[string]yangStruct
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
		wantCompress: map[string]yangStruct{
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
		wantUncompress: map[string]yangStruct{
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
		wantCompress: map[string]yangStruct{
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
		wantUncompress: map[string]yangStruct{
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
		wantCompress: map[string]yangStruct{
			"/top-container": {
				name: "top-container",
				fields: map[string]*yang.Entry{
					"leaf-one": {Name: "leaf-one", Type: &yang.YangType{Kind: yang.Yint8}},
					"leaf-two": {Name: "leaf-two", Type: &yang.YangType{Kind: yang.Yint8}},
				},
				path: []string{"", "top-container"},
			},
		},
		wantUncompress: map[string]yangStruct{
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
		wantCompress: map[string]yangStruct{
			"/module/container/list": {
				name: "list",
				fields: map[string]*yang.Entry{
					"key": {Name: "key", Type: &yang.YangType{Kind: yang.Ystring}},
				},
			},
		},
		wantUncompress: map[string]yangStruct{
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
		wantCompress: map[string]yangStruct{
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
		wantUncompress: map[string]yangStruct{
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
		for compress, expected := range map[bool]map[string]yangStruct{true: tt.wantCompress, false: tt.wantUncompress} {
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
				t.Errorf("%s buildStructDefinitions(CompressOCPaths: %v): could not build struct defs: %v", tt.name, compress, errs)
				continue
			}

			for name, gostruct := range structDefs {
				expstr, ok := expected[name]
				if !ok {
					t.Errorf("%s buildStructDefinitions(CompressOCPaths: %v): could not find expected struct %s, got: %v, want: %v",
						tt.name, compress, name, structDefs, expected)
					continue
				}

				for fieldk, fieldv := range expstr.fields {
					cmpfield, ok := gostruct.fields[fieldk]
					if !ok {
						t.Errorf("%s buildStructDefinitions(CompressOCPaths: %v): could not find expected field %s in %s, got: %v",
							tt.name, compress, fieldk, name, gostruct.fields)
						continue
					}

					if fieldv.Name != cmpfield.Name {
						t.Errorf("%s buildStructDefinitions(CompressOCPaths: %v): field %s of %s did not have expected name, got: %v, want: %v",
							tt.name, compress, fieldk, name, fieldv.Name, cmpfield.Name)
					}

					if fieldv.Type != nil && cmpfield.Type != nil {
						if fieldv.Type.Kind != cmpfield.Type.Kind {
							t.Errorf("%s buildStructDefinitions(CompressOCPaths: %v): field %s of %s did not have expected type got: %s, want: %s",
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

// yangTestCase describs a test case for which code generation is performed
// through Goyang's API, it provides the input set of parameters in a way that
// can be reused across tests.
type yangTestCase struct {
	name                string          // Name is the identifier for the test.
	inFiles             []string        // inFiles is the set of inputFiles for the test.
	inIncludePaths      []string        // inIncludePaths is the set of paths that should be searched for imports.
	inExcludeModules    []string        // inExcludeModules is the set of modules that should be excluded from code generation.
	inConfig            GeneratorConfig // inConfig specifies the configuration that should be used for the generator test case.
	wantStructsCodeFile string          // wantsStructsCodeFile is the path of the generated Go code that the output of the test should be compared to.
	wantErr             bool            // wantErr specifies whether the test should expect an error.
	wantSchemaFile      string          // wantSchemaFile is the path to the schema JSON that the output of the test should be compared to.
}

// TestSimpleStructs tests the processModules, GenerateGoCode and writeGoCode
// functions. It takes the set of YANG modules described in the slice of
// yangTestCases and generates the struct code for them, comparing the output
// to the wantStructsCodeFile.  In order to simplify the files that are used,
// the GenerateGoCode structs are concatenated before comparison with the
// expected output. If the generated code matches the expected output, it is
// run against the Go parser to ensure that the code is valid Go - this is
// expected, but it ensures that the input file does not contain Go which is
// invalid.
func TestSimpleStructs(t *testing.T) {
	tests := []yangTestCase{{
		name:                "simple openconfig test, with compression",
		inFiles:             []string{filepath.Join(TestRoot, "testdata/structs/openconfig-simple.yang")},
		inConfig:            GeneratorConfig{CompressOCPaths: true},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata/structs/openconfig-simple.formatted-txt"),
	}, {
		name:                "simple openconfig test, with no compression",
		inFiles:             []string{filepath.Join(TestRoot, "testdata/structs/openconfig-simple.yang")},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata/structs/openconfig-simple-no-compress.formatted-txt"),
	}, {
		name:                "simple openconfig test, with a list",
		inFiles:             []string{filepath.Join(TestRoot, "testdata/structs/openconfig-withlist.yang")},
		inConfig:            GeneratorConfig{CompressOCPaths: true},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata/structs/openconfig-withlist.formatted-txt"),
	}, {
		name:                "simple openconfig test, with a list that has an enumeration key",
		inFiles:             []string{filepath.Join(TestRoot, "testdata/structs/openconfig-list-enum-key.yang")},
		inConfig:            GeneratorConfig{CompressOCPaths: true},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata/structs/openconfig-list-enum-key.formatted-txt"),
	}, {
		name:                "openconfig test with a identityref union",
		inFiles:             []string{filepath.Join(TestRoot, "testdata/structs/openconfig-unione.yang")},
		inConfig:            GeneratorConfig{CompressOCPaths: true},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata/structs/openconfig-unione.formatted-txt"),
	}, {
		name:    "openconfig tests with fakeroot",
		inFiles: []string{filepath.Join(TestRoot, "testdata/structs/openconfig-fakeroot.yang")},
		inConfig: GeneratorConfig{
			CompressOCPaths:  true,
			GenerateFakeRoot: true,
		},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata/structs/openconfig-fakeroot.formatted-txt"),
	}, {
		name:    "openconfig noncompressed tests with fakeroot",
		inFiles: []string{filepath.Join(TestRoot, "testdata/structs/openconfig-fakeroot.yang")},
		inConfig: GeneratorConfig{
			GenerateFakeRoot: true,
		},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata/structs/openconfig-fakeroot-nc.formatted-txt"),
	}, {
		name:    "schema test with compression",
		inFiles: []string{filepath.Join(TestRoot, "testdata/schema/openconfig-options.yang")},
		inConfig: GeneratorConfig{
			CompressOCPaths:    true,
			GenerateJSONSchema: true,
		},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata/schema/openconfig-options-compress.formatted-txt"),
		wantSchemaFile:      filepath.Join(TestRoot, "testdata/schema/openconfig-options-compress-schema.json"),
	}, {
		name:    "schema test without compression",
		inFiles: []string{filepath.Join(TestRoot, "testdata/schema/openconfig-options.yang")},
		inConfig: GeneratorConfig{
			GenerateJSONSchema: true,
		},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata/schema/openconfig-options-nocompress.formatted-txt"),
		wantSchemaFile:      filepath.Join(TestRoot, "testdata/schema/openconfig-options-nocompress-schema.json"),
	}, {
		name:    "schema test with fakeroot",
		inFiles: []string{filepath.Join(TestRoot, "testdata/schema/openconfig-options.yang")},
		inConfig: GeneratorConfig{
			CompressOCPaths:    true,
			GenerateFakeRoot:   true,
			GenerateJSONSchema: true,
		},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata/schema/openconfig-options-compress-fakeroot.formatted-txt"),
		wantSchemaFile:      filepath.Join(TestRoot, "testdata/schema/openconfig-options-compress-fakeroot-schema.json"),
	}, {
		name:    "schema test with fakeroot and no compression",
		inFiles: []string{filepath.Join(TestRoot, "testdata/schema/openconfig-options.yang")},
		inConfig: GeneratorConfig{
			GenerateFakeRoot:   true,
			GenerateJSONSchema: true,
		},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata/schema/openconfig-options-nocompress-fakeroot.formatted-txt"),
		wantSchemaFile:      filepath.Join(TestRoot, "testdata/schema/openconfig-options-nocompress-fakeroot-schema.json"),
	}, {
		name:    "schema test with camelcase annotations",
		inFiles: []string{filepath.Join(TestRoot, "testdata/structs/openconfig-camelcase.yang")},
		inConfig: GeneratorConfig{
			CompressOCPaths:  true,
			GenerateFakeRoot: true,
		},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata/structs/openconfig-camelcase-compress.formatted-txt"),
	}, {
		name:    "structs test with camelcase annotations",
		inFiles: []string{filepath.Join(TestRoot, "testdata/structs/openconfig-enumcamelcase.yang")},
		inConfig: GeneratorConfig{
			CompressOCPaths: true,
		},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata/structs/openconfig-enumcamelcase-compress.formatted-txt"),
	}, {
		name:                "structs test with choices and cases",
		inFiles:             []string{filepath.Join(TestRoot, "testdata/structs/choice-case-example.yang")},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata/structs/choice-case-example.formatted-txt"),
	}, {
		name: "module with augments",
		inFiles: []string{
			filepath.Join(TestRoot, "testdata/structs/openconfig-simple-target.yang"),
			filepath.Join(TestRoot, "testdata/structs/openconfig-simple-augment.yang"),
		},
		inConfig: GeneratorConfig{
			CompressOCPaths:  true,
			GenerateFakeRoot: true,
		},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata/structs/openconfig-augmented.formatted-txt"),
	}, {
		name:    "variable and import explicitly specified",
		inFiles: []string{filepath.Join(TestRoot, "testdata/structs/openconfig-simple.yang")},
		inConfig: GeneratorConfig{
			CompressOCPaths:    true,
			GenerateFakeRoot:   true,
			Caller:             "testcase",
			FakeRootName:       "fakeroot",
			StoreRawSchema:     true,
			GenerateJSONSchema: true,
			GoOptions: GoOpts{
				SchemaVarName:    "YANGSchema",
				GoyangImportPath: "foo/goyang",
				YgotImportPath:   "bar/ygot",
				YtypesImportPath: "baz/ytypes",
			},
		},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata/schema/openconfig-options-explicit.formatted-txt"),
		wantSchemaFile:      filepath.Join(TestRoot, "testdata/schema/openconfig-options-explicit-schema.json"),
	}}

	for _, tt := range tests {
		// Set defaults within the supplied configuration for these tests.
		if tt.inConfig.Caller == "" {
			// Set the name of the caller explicitly to avoid issues when
			// the unit tests are called by external test entities.
			tt.inConfig.Caller = "codegen-tests"
		}
		tt.inConfig.StoreRawSchema = true

		cg := NewYANGCodeGenerator(&tt.inConfig)

		gotGeneratedCode, err := cg.GenerateGoCode(tt.inFiles, tt.inIncludePaths)
		if err != nil && !tt.wantErr {
			t.Errorf("%s: cg.GenerateCode(%v, %v): Config: %v, got unexpected error: %v, want: nil",
				tt.name, tt.inFiles, tt.inIncludePaths, tt.inConfig, err)
			continue
		}

		wantCode, rferr := ioutil.ReadFile(tt.wantStructsCodeFile)
		if rferr != nil {
			t.Errorf("%s: ioutil.ReadFile(%q) error: %v", tt.name, tt.wantStructsCodeFile, rferr)
			continue
		}

		// Write all the received structs into a single file such that
		// it can be compared to the received file.
		var gotCode bytes.Buffer
		fmt.Fprint(&gotCode, gotGeneratedCode.Header)
		for _, gotStruct := range gotGeneratedCode.Structs {
			fmt.Fprintf(&gotCode, gotStruct)
		}

		for _, gotEnum := range gotGeneratedCode.Enums {
			fmt.Fprintf(&gotCode, gotEnum)
		}

		// Write generated enumeration map out.
		fmt.Fprintf(&gotCode, gotGeneratedCode.EnumMap)

		if tt.inConfig.GenerateJSONSchema {
			// Write the schema byte array out.
			fmt.Fprintf(&gotCode, gotGeneratedCode.JSONSchemaCode)

			wantSchema, rferr := ioutil.ReadFile(tt.wantSchemaFile)
			if rferr != nil {
				t.Errorf("%s: ioutil.ReadFile(%q) error: %v", tt.name, tt.wantSchemaFile, err)
				continue
			}

			var gotJSON map[string]interface{}
			if err := json.Unmarshal(gotGeneratedCode.RawJSONSchema, &gotJSON); err != nil {
				t.Errorf("%s: json.Unmarshal(..., %v), could not unmarshal received JSON: %v", tt.name, gotGeneratedCode.RawJSONSchema, err)
				continue
			}

			var wantJSON map[string]interface{}
			if err := json.Unmarshal(wantSchema, &wantJSON); err != nil {
				t.Errorf("%s: json.Unmarshal(..., [contents of %s]), could not unmarshal golden JSON file: %v", tt.name, tt.wantSchemaFile, err)
				continue
			}

			if !reflect.DeepEqual(gotJSON, wantJSON) {
				diff := difflib.UnifiedDiff{
					A:        difflib.SplitLines(string(gotGeneratedCode.RawJSONSchema)),
					B:        difflib.SplitLines(string(wantSchema)),
					FromFile: "got",
					ToFile:   "want",
					Context:  3,
					Eol:      "\n",
				}
				diffr, _ := difflib.GetUnifiedDiffString(diff)
				t.Errorf("%s: GenerateGoCode(%v, %v), Config: %v, did not return correct JSON (file: %v), diff: \n%s", tt.name, tt.inFiles, tt.inIncludePaths, tt.inConfig, tt.wantSchemaFile, diffr)
			}
		}

		if gotCode.String() != string(wantCode) {
			// Use difflib to generate a unified diff between the
			// two code snippets such that this is simpler to debug
			// in the test output.
			diff := difflib.UnifiedDiff{
				A:        difflib.SplitLines(gotCode.String()),
				B:        difflib.SplitLines(string(wantCode)),
				FromFile: "got",
				ToFile:   "want",
				Context:  3,
				Eol:      "\n",
			}
			diffr, _ := difflib.GetUnifiedDiffString(diff)
			t.Errorf("%s: GenerateGoCode(%v, %v), Config: %v, did not return correct code (file: %v), diff:\n%s",
				tt.name, tt.inFiles, tt.inIncludePaths, tt.inConfig, tt.wantStructsCodeFile, diffr)
		}
	}
}

func TestFindRootEntries(t *testing.T) {
	tests := []struct {
		name                       string
		inStructs                  map[string]*yang.Entry
		inRootName                 string
		wantCompressRootChildren   []string
		wantUncompressRootChildren []string
	}{{
		name: "directory at root, compress paths on",
		inStructs: map[string]*yang.Entry{
			"/foo": {
				Name: "foo",
				Dir:  map[string]*yang.Entry{},
				Parent: &yang.Entry{
					Name: "module",
				},
			},
			"/foo/bar": {
				Name: "bar",
				Dir:  map[string]*yang.Entry{},
				Parent: &yang.Entry{
					Name: "foo",
					Parent: &yang.Entry{
						Name: "module",
					},
				},
			},
		},
		inRootName:                 "fakeroot",
		wantCompressRootChildren:   []string{"foo"},
		wantUncompressRootChildren: []string{"foo"},
	}}

	for _, tt := range tests {
		for compress, wantChildren := range map[bool][]string{true: tt.wantCompressRootChildren, false: tt.wantUncompressRootChildren} {
			cg := NewYANGCodeGenerator(&GeneratorConfig{
				CompressOCPaths: compress,
				FakeRootName:    tt.inRootName,
			})

			if err := cg.createFakeRoot(tt.inStructs); err != nil {
				t.Errorf("%s: cg.createFakeRoot(%v), CompressOCPaths: %v, got unexpected error: %v", tt.name, tt.inStructs, compress, err)
				continue
			}

			rootElem, ok := tt.inStructs["/"]
			if !ok {
				t.Errorf("%s: cg.createFakeRoot(%v), CompressOCPaths: %v, could not find root element", tt.name, tt.inStructs, compress)
				continue
			}

			gotChildren := map[string]bool{}
			for n := range rootElem.Dir {
				gotChildren[n] = true
			}

			for _, ch := range wantChildren {
				if _, ok := rootElem.Dir[ch]; !ok {
					t.Errorf("%s: cg.createFakeRoot(%v), CompressOCPaths: %v, could not find child %v in %v", tt.name, tt.inStructs, compress, ch, rootElem.Dir)
				}
				gotChildren[ch] = false
			}

			for ch, ok := range gotChildren {
				if ok == true {
					t.Errorf("%s: cg.findRootentries(%v), CompressOCPaths: %v, did not expect child %v", tt.name, tt.inStructs, compress, ch)
				}
			}
		}
	}
}
