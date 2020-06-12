// Copyright 2020 Google Inc.
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
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/openconfig/gnmi/errdiff"
	"github.com/openconfig/goyang/pkg/yang"
)

func TestResolveNameClashSet(t *testing.T) {
	tests := []struct {
		name                        string
		inDefinedEnums              map[string]bool
		inDefinedEnumsNoUnderscores map[string]bool
		inNameClashSets             map[string]map[string]*yang.Entry
		inShortenEnumLeafNames      bool
		// wantUncompressFailDueToClash means the uncompressed test run will fail in
		// deviation from the compressed case due to existence of a name clash, which can
		// only be resolved for compressed paths.
		wantUncompressFailDueToClash    bool
		wantUniqueNamesMap              map[string]string
		wantUniqueNamesMapNoUnderscores map[string]string
		wantErrSubstr                   string
	}{{
		name: "no name clash",
		inDefinedEnums: map[string]bool{
			"Baz": true,
		},
		inDefinedEnumsNoUnderscores: map[string]bool{
			"Baz": true,
		},
		inNameClashSets: map[string]map[string]*yang.Entry{
			"Foo": {
				"enum-a": &yang.Entry{
					Name: "enum-a",
				},
			},
			"Bar": {
				"enum-b": &yang.Entry{
					Name: "enum-b",
				},
			},
		},
		inShortenEnumLeafNames: true,
		wantUniqueNamesMap: map[string]string{
			"enum-a": "Foo",
			"enum-b": "Bar",
		},
		wantUniqueNamesMapNoUnderscores: map[string]string{
			"enum-a": "Foo",
			"enum-b": "Bar",
		},
	}, {
		name: "no name clash, names not shortened",
		inDefinedEnums: map[string]bool{
			"Mod_Baz": true,
		},
		inDefinedEnumsNoUnderscores: map[string]bool{
			"ModBaz": true,
		},
		inNameClashSets: map[string]map[string]*yang.Entry{
			"Mod_Foo": {
				"enum-a": &yang.Entry{
					Name: "enum-a",
				},
			},
			"Mod_Bar": {
				"enum-b": &yang.Entry{
					Name: "enum-b",
				},
			},
		},
		wantUniqueNamesMap: map[string]string{
			"enum-a": "Mod_Foo",
			"enum-b": "Mod_Bar",
		},
		wantUniqueNamesMapNoUnderscores: map[string]string{
			"enum-a": "ModFoo",
			"enum-b": "ModBar",
		},
	}, {
		name: "no name clash but name already exists in definedEnums due to an algorithm bug",
		inDefinedEnums: map[string]bool{
			"Bar": true,
		},
		inDefinedEnumsNoUnderscores: map[string]bool{
			"Bar": true,
		},
		inNameClashSets: map[string]map[string]*yang.Entry{
			"Foo": {
				"enum-a": &yang.Entry{
					Name: "enum-a",
				},
			},
			"Bar": {
				"enum-b": &yang.Entry{
					Name: "enum-b",
				},
			},
		},
		inShortenEnumLeafNames: true,
		wantErrSubstr:          `default name "Bar" has already been assigned`,
	}, {
		name: "resolving name clash at module name",
		inDefinedEnums: map[string]bool{
			"Baz": true,
			"Foo": true,
		},
		inDefinedEnumsNoUnderscores: map[string]bool{
			"Baz": true,
			"Foo": true,
		},
		inNameClashSets: map[string]map[string]*yang.Entry{
			"Foo": {
				"enum-a": &yang.Entry{
					Name: "enum-a",
					Node: &yang.Enum{
						Parent: &yang.Container{
							Name: "parent-a",
							Parent: &yang.Container{
								Name: "gran-gran-a",
								Parent: &yang.Module{
									Name: "base-module",
								},
							},
						},
					},
					Parent: &yang.Entry{
						Name: "parent-a",
						Parent: &yang.Entry{
							Name: "gran-gran-a",
							Parent: &yang.Entry{
								Name: "base-module",
							},
						},
					},
				},
				"enum-b": &yang.Entry{
					Name: "enum-b",
					Node: &yang.Enum{
						Parent: &yang.Container{
							Name: "parent-b",
							Parent: &yang.Container{
								Name: "gran-gran-b",
								Parent: &yang.Module{
									Name: "support-module",
								},
							},
						},
					},
					Parent: &yang.Entry{
						Name: "parent-b",
						Parent: &yang.Entry{
							Name: "gran-gran-b",
							Parent: &yang.Entry{
								Name: "support-module",
							},
						},
					},
				},
			},
		},
		inShortenEnumLeafNames:       true,
		wantUncompressFailDueToClash: true,
		wantUniqueNamesMap: map[string]string{
			"enum-a": "BaseModule_Foo",
			"enum-b": "SupportModule_Foo",
		},
		wantUniqueNamesMapNoUnderscores: map[string]string{
			"enum-a": "BaseModuleFoo",
			"enum-b": "SupportModuleFoo",
		},
	}, {
		name:                        "cannot resolve name clash due to camel-case lossiness and no parents to disambiguate",
		inDefinedEnums:              map[string]bool{},
		inDefinedEnumsNoUnderscores: map[string]bool{},
		inNameClashSets: map[string]map[string]*yang.Entry{
			"Foo": {
				"enum-a": &yang.Entry{
					Name: "enum-a",
					Node: &yang.Enum{
						Parent: &yang.Module{
							Name: "base-module",
						},
					},
					Parent: &yang.Entry{
						Name: "base-module",
					},
				},
				"enum-A": &yang.Entry{
					Name: "enum-A",
					Node: &yang.Enum{
						Parent: &yang.Module{
							Name: "base-module",
						},
					},
					Parent: &yang.Entry{
						Name: "base-module",
					},
				},
			},
		},
		inShortenEnumLeafNames:       true,
		wantUncompressFailDueToClash: true,
		wantErrSubstr:                "cannot resolve enumeration name clash",
	}, {
		name:                        "cannot resolve name clash due to camel-case lossiness and no parents to disambiguate, with names not shortened",
		inDefinedEnums:              map[string]bool{},
		inDefinedEnumsNoUnderscores: map[string]bool{},
		inNameClashSets: map[string]map[string]*yang.Entry{
			"BaseModule_Foo": {
				"enum-a": &yang.Entry{
					Name: "enum-a",
					Node: &yang.Enum{
						Parent: &yang.Module{
							Name: "base-module",
						},
					},
					Parent: &yang.Entry{
						Name: "base-module",
					},
				},
				"enum-A": &yang.Entry{
					Name: "enum-A",
					Node: &yang.Enum{
						Parent: &yang.Module{
							Name: "base-module",
						},
					},
					Parent: &yang.Entry{
						Name: "base-module",
					},
				},
			},
		},
		wantUncompressFailDueToClash: true,
		wantErrSubstr:                "cannot resolve enumeration name clash",
	}, {
		name: "resolving name clash at grandparent for enumeration leaves",
		inDefinedEnums: map[string]bool{
			"Baz": true,
		},
		inDefinedEnumsNoUnderscores: map[string]bool{
			"Baz": true,
		},
		inNameClashSets: map[string]map[string]*yang.Entry{
			"Foo": {
				"enum-a": &yang.Entry{
					Name: "enum-a",
					Node: &yang.Enum{
						Parent: &yang.Container{
							Name: "parent-a",
							Parent: &yang.Container{
								Name: "gran-gran-a",
								Parent: &yang.Module{
									Name: "base-module",
								},
							},
						},
					},
					Parent: &yang.Entry{
						Name: "parent-a",
						Parent: &yang.Entry{
							Name: "gran-gran-a",
							Parent: &yang.Entry{
								Name: "base-module",
							},
						},
					},
				},
				"enum-b": &yang.Entry{
					Name: "enum-b",
					Node: &yang.Enum{
						Parent: &yang.Container{
							Name: "parent-b",
							Parent: &yang.Container{
								Name: "gran-gran-b",
								Parent: &yang.Module{
									Name: "base-module",
								},
							},
						},
					},
					Parent: &yang.Entry{
						Name: "parent-b",
						Parent: &yang.Entry{
							Name: "gran-gran-b",
							Parent: &yang.Entry{
								Name: "base-module",
							},
						},
					},
				},
			},
		},
		inShortenEnumLeafNames:       true,
		wantUncompressFailDueToClash: true,
		wantUniqueNamesMap: map[string]string{
			"enum-a": "GranGranA_Foo",
			"enum-b": "GranGranB_Foo",
		},
		wantUniqueNamesMapNoUnderscores: map[string]string{
			"enum-a": "GranGranAFoo",
			"enum-b": "GranGranBFoo",
		},
	}, {
		name: "resolving name clash at grandparent for enumeration leaves with names not shortened",
		inDefinedEnums: map[string]bool{
			"BaseModule_Baz": true,
			"BaseModule_Foo": true,
		},
		inDefinedEnumsNoUnderscores: map[string]bool{
			"BaseModuleBaz": true,
			"BaseModuleFoo": true,
		},
		inNameClashSets: map[string]map[string]*yang.Entry{
			"BaseModule_Foo": {
				"enum-a": &yang.Entry{
					Name: "enum-a",
					Node: &yang.Enum{
						Parent: &yang.Container{
							Name: "parent-a",
							Parent: &yang.Container{
								Name: "gran-gran-a",
								Parent: &yang.Module{
									Name: "base-module",
								},
							},
						},
					},
					Parent: &yang.Entry{
						Name: "parent-a",
						Node: &yang.Container{
							Name: "parent-a",
							Parent: &yang.Container{
								Name: "gran-gran-a",
								Parent: &yang.Module{
									Name: "base-module",
								},
							},
						},
						Parent: &yang.Entry{
							Name: "gran-gran-a",
							Node: &yang.Container{
								Name: "gran-gran-a",
								Parent: &yang.Module{
									Name: "base-module",
								},
							},
							Parent: &yang.Entry{
								Name: "base-module",
								Node: &yang.Module{
									Name: "base-module",
								},
							},
						},
					},
				},
				"enum-b": &yang.Entry{
					Name: "enum-b",
					Node: &yang.Enum{
						Parent: &yang.Container{
							Name: "parent-b",
							Parent: &yang.Container{
								Name: "gran-gran-b",
								Parent: &yang.Module{
									Name: "base-module",
								},
							},
						},
					},
					Parent: &yang.Entry{
						Name: "parent-b",
						Node: &yang.Container{
							Name: "parent-b",
							Parent: &yang.Container{
								Name: "gran-gran-b",
								Parent: &yang.Module{
									Name: "base-module",
								},
							},
						},
						Parent: &yang.Entry{
							Name: "gran-gran-b",
							Node: &yang.Container{
								Name: "gran-gran-b",
								Parent: &yang.Module{
									Name: "base-module",
								},
							},
							Parent: &yang.Entry{
								Name: "base-module",
								Node: &yang.Module{
									Name: "base-module",
								},
							},
						},
					},
				},
			},
		},
		wantUncompressFailDueToClash: true,
		wantUniqueNamesMap: map[string]string{
			"enum-a": "BaseModule_GranGranA_Foo",
			"enum-b": "BaseModule_GranGranB_Foo",
		},
		wantUniqueNamesMapNoUnderscores: map[string]string{
			"enum-a": "BaseModuleGranGranAFoo",
			"enum-b": "BaseModuleGranGranBFoo",
		},
	}, {
		name: "resolving name clash at grandparent and due to no more parent container",
		inDefinedEnums: map[string]bool{
			"Baz": true,
		},
		inDefinedEnumsNoUnderscores: map[string]bool{
			"Baz": true,
		},
		inNameClashSets: map[string]map[string]*yang.Entry{
			"Foo": {
				"enum-a": &yang.Entry{
					Name: "enum-a",
					Node: &yang.Enum{
						Parent: &yang.Container{
							Name: "parent",
							Parent: &yang.Container{
								Name: "gran-gran",
								Parent: &yang.Module{
									Name: "base-module",
								},
							},
						},
					},
					Parent: &yang.Entry{
						Name: "parent",
						Parent: &yang.Entry{
							Name: "gran-gran",
							Parent: &yang.Entry{
								Name: "base-module",
							},
						},
					},
				},
				"enum-b": &yang.Entry{
					Name: "enum-b",
					Node: &yang.Enum{
						Parent: &yang.Container{
							Name: "gran-gran",
							Parent: &yang.Module{
								Name: "base-module",
							},
						},
					},
					Parent: &yang.Entry{
						Name: "gran-gran",
						Parent: &yang.Entry{
							Name: "base-module",
						},
					},
				},
			},
		},
		inShortenEnumLeafNames:       true,
		wantUncompressFailDueToClash: true,
		wantUniqueNamesMap: map[string]string{
			"enum-a": "GranGran_Foo",
			"enum-b": "Foo",
		},
		wantUniqueNamesMapNoUnderscores: map[string]string{
			"enum-a": "GranGranFoo",
			"enum-b": "Foo",
		},
	}, {
		name: "resolving name clash at grandparent and due to no more parent container with names not shortened",
		inDefinedEnums: map[string]bool{
			"BaseModule_Baz": true,
		},
		inDefinedEnumsNoUnderscores: map[string]bool{
			"BaseModuleBaz": true,
		},
		inNameClashSets: map[string]map[string]*yang.Entry{
			"BaseModule_Foo": {
				"enum-a": &yang.Entry{
					Name: "enum-a",
					Node: &yang.Enum{
						Parent: &yang.Container{
							Name: "parent",
							Parent: &yang.Container{
								Name: "gran-gran",
								Parent: &yang.Module{
									Name: "base-module",
								},
							},
						},
					},
					Parent: &yang.Entry{
						Name: "parent",
						Node: &yang.Container{
							Name: "parent",
							Parent: &yang.Container{
								Name: "gran-gran",
								Parent: &yang.Module{
									Name: "base-module",
								},
							},
						},
						Parent: &yang.Entry{
							Node: &yang.Container{
								Name: "gran-gran",
								Parent: &yang.Module{
									Name: "base-module",
								},
							},
							Name: "gran-gran",
							Parent: &yang.Entry{
								Node: &yang.Module{
									Name: "base-module",
								},
								Name: "base-module",
							},
						},
					},
				},
				"enum-b": &yang.Entry{
					Name: "enum-b",
					Node: &yang.Enum{
						Parent: &yang.Container{
							Name: "gran-gran",
							Parent: &yang.Module{
								Name: "base-module",
							},
						},
					},
					Parent: &yang.Entry{
						Name: "gran-gran",
						Node: &yang.Container{
							Name: "gran-gran",
							Parent: &yang.Module{
								Name: "base-module",
							},
						},
						Parent: &yang.Entry{
							Name: "base-module",
							Node: &yang.Module{
								Name: "base-module",
							},
						},
					},
				},
			},
		},
		wantUncompressFailDueToClash: true,
		wantUniqueNamesMap: map[string]string{
			"enum-a": "BaseModule_GranGran_Foo",
			"enum-b": "BaseModule_Foo",
		},
		wantUniqueNamesMapNoUnderscores: map[string]string{
			"enum-a": "BaseModuleGranGranFoo",
			"enum-b": "BaseModuleFoo",
		},
	}, {
		name: "resolving name clash at parent and due to no more parent container",
		inDefinedEnums: map[string]bool{
			"Baz": true,
		},
		inDefinedEnumsNoUnderscores: map[string]bool{
			"Baz": true,
		},
		inNameClashSets: map[string]map[string]*yang.Entry{
			"Foo": {
				"enum-a": &yang.Entry{
					Name: "enum-a",
					Node: &yang.Enum{
						Parent: &yang.Container{
							Name: "parent",
							Parent: &yang.Module{
								Name: "base-module",
							},
						},
					},
					Parent: &yang.Entry{
						Name: "parent",
						Parent: &yang.Entry{
							Name: "base-module",
						},
					},
				},
				"enum-b": &yang.Entry{
					Name: "enum-b",
					Node: &yang.Enum{
						Parent: &yang.Module{
							Name: "base-module",
						},
					},
					Parent: &yang.Entry{
						Name: "base-module",
					},
				},
			},
		},
		inShortenEnumLeafNames:       true,
		wantUncompressFailDueToClash: true,
		wantErrSubstr:                "cannot resolve enumeration name clash",
	}, {
		name: "resolving name clash at parent and due to no more parent container, with names not shortened",
		inDefinedEnums: map[string]bool{
			"BaseModule_Baz": true,
		},
		inDefinedEnumsNoUnderscores: map[string]bool{
			"BaseModuleBaz": true,
		},
		inNameClashSets: map[string]map[string]*yang.Entry{
			"BaseModule_Foo": {
				"enum-a": &yang.Entry{
					Name: "enum-a",
					Node: &yang.Enum{
						Parent: &yang.Container{
							Name: "parent",
							Parent: &yang.Module{
								Name: "base-module",
							},
						},
					},
					Parent: &yang.Entry{
						Name: "parent",
						Node: &yang.Container{
							Name: "parent",
							Parent: &yang.Module{
								Name: "base-module",
							},
						},
						Parent: &yang.Entry{
							Name: "base-module",
							Node: &yang.Module{
								Name: "base-module",
							},
						},
					},
				},
				"enum-b": &yang.Entry{
					Name: "enum-b",
					Node: &yang.Enum{
						Parent: &yang.Module{
							Name: "base-module",
						},
					},
					Parent: &yang.Entry{
						Name: "base-module",
						Node: &yang.Module{
							Name: "base-module",
						},
					},
				},
			},
		},
		wantUncompressFailDueToClash: true,
		wantErrSubstr:                "cannot resolve enumeration name clash",
	}, {
		name: "resolving name clash at grandparent due to name from module-level disambiguation already in definedEnums",
		inDefinedEnums: map[string]bool{
			"Baz":               true,
			"Foo":               true,
			"SupportModule_Foo": true,
		},
		inDefinedEnumsNoUnderscores: map[string]bool{
			"Baz":              true,
			"Foo":              true,
			"SupportModuleFoo": true,
		},
		inNameClashSets: map[string]map[string]*yang.Entry{
			"Foo": {
				"enum-a": &yang.Entry{
					Name: "enum-a",
					Node: &yang.Enum{
						Parent: &yang.Container{
							Name: "parent-a",
							Parent: &yang.Container{
								Name: "gran-gran-a",
								Parent: &yang.Module{
									Name: "base-module",
								},
							},
						},
					},
					Parent: &yang.Entry{
						Name: "parent-a",
						Parent: &yang.Entry{
							Name: "gran-gran-a",
							Parent: &yang.Entry{
								Name: "base-module",
							},
						},
					},
				},
				"enum-b": &yang.Entry{
					Name: "enum-b",
					Node: &yang.Enum{
						Parent: &yang.Container{
							Name: "parent-b",
							Parent: &yang.Container{
								Name: "gran-gran-b",
								Parent: &yang.Module{
									Name: "support-module",
								},
							},
						},
					},
					Parent: &yang.Entry{
						Name: "parent-b",
						Parent: &yang.Entry{
							Name: "gran-gran-b",
							Parent: &yang.Entry{
								Name: "support-module",
							},
						},
					},
				},
			},
		},
		inShortenEnumLeafNames:       true,
		wantUncompressFailDueToClash: true,
		wantUniqueNamesMap: map[string]string{
			"enum-a": "GranGranA_Foo",
			"enum-b": "GranGranB_Foo",
		},
		wantUniqueNamesMapNoUnderscores: map[string]string{
			"enum-a": "GranGranAFoo",
			"enum-b": "GranGranBFoo",
		},
	}, {
		name: "resolving name clash at great-grandparent",
		inDefinedEnums: map[string]bool{
			"Baz": true,
			"Foo": true,
		},
		inDefinedEnumsNoUnderscores: map[string]bool{
			"Baz": true,
			"Foo": true,
		},
		inNameClashSets: map[string]map[string]*yang.Entry{
			"Foo": {
				"enum-a": &yang.Entry{
					Name: "enum-a",
					Node: &yang.Enum{
						Parent: &yang.Container{
							Name: "parent",
							Parent: &yang.Container{
								Name: "gran-gran",
								Parent: &yang.Container{
									Name: "great-gran-gran-a",
									Parent: &yang.Module{
										Name: "base-module",
									},
								},
							},
						},
					},
					Parent: &yang.Entry{
						Name: "parent",
						Parent: &yang.Entry{
							Name: "gran-gran",
							Parent: &yang.Entry{
								Name: "great-gran-gran-a",
								Parent: &yang.Entry{
									Name: "base-module",
								},
							},
						},
					},
				},
				"enum-b": &yang.Entry{
					Name: "enum-b",
					Node: &yang.Enum{
						Parent: &yang.Container{
							Name: "parent",
							Parent: &yang.Container{
								Name: "gran-gran",
								Parent: &yang.Container{
									Name: "great-gran-gran-b",
									Parent: &yang.Module{
										Name: "base-module",
									},
								},
							},
						},
					},
					Parent: &yang.Entry{
						Name: "parent",
						Parent: &yang.Entry{
							Name: "gran-gran",
							Parent: &yang.Entry{
								Name: "great-gran-gran-b",
								Parent: &yang.Entry{
									Name: "base-module",
								},
							},
						},
					},
				},
			},
		},
		inShortenEnumLeafNames:       true,
		wantUncompressFailDueToClash: true,
		wantUniqueNamesMap: map[string]string{
			"enum-a": "GreatGranGranA_Foo",
			"enum-b": "GreatGranGranB_Foo",
		},
		wantUniqueNamesMapNoUnderscores: map[string]string{
			"enum-a": "GreatGranGranAFoo",
			"enum-b": "GreatGranGranBFoo",
		},
	}, {
		name: "resolving name clash at great-grandparent due to name from grandparent-level disambiguation already present in definedEnums",
		inDefinedEnums: map[string]bool{
			"Baz":                       true,
			"GranGranA_Foo":             true,
			"BaseModule_ParentB_Enum":   true,
			"BaseModule_GranGranA_Enum": true,
		},
		inDefinedEnumsNoUnderscores: map[string]bool{
			"Baz":                      true,
			"GranGranAFoo":             true,
			"BaseModuleParentB_Enum":   true,
			"BaseModuleGranGranA_Enum": true,
		},
		inNameClashSets: map[string]map[string]*yang.Entry{
			"Foo": {
				"enum-a": &yang.Entry{
					Name: "enum-a",
					Node: &yang.Enum{
						Parent: &yang.Container{
							Name: "parent-a",
							Parent: &yang.Container{
								Name: "gran-gran-a",
								Parent: &yang.Container{
									Name: "great-gran-gran-a",
									Parent: &yang.Module{
										Name: "base-module",
									},
								},
							},
						},
					},
					Parent: &yang.Entry{
						Name: "parent-a",
						Parent: &yang.Entry{
							Name: "gran-gran-a",
							Parent: &yang.Entry{
								Name: "great-gran-gran-a",
								Parent: &yang.Entry{
									Name: "base-module",
								},
							},
						},
					},
				},
				"enum-b": &yang.Entry{
					Name: "enum-b",
					Node: &yang.Enum{
						Parent: &yang.Container{
							Name: "parent-b",
							Parent: &yang.Container{
								Name: "gran-gran-b",
								Parent: &yang.Container{
									Name: "great-gran-gran-b",
									Parent: &yang.Module{
										Name: "base-module",
									},
								},
							},
						},
					},
					Parent: &yang.Entry{
						Name: "parent-b",
						Parent: &yang.Entry{
							Name: "gran-gran-b",
							Parent: &yang.Entry{
								Name: "great-gran-gran-b",
								Parent: &yang.Entry{
									Name: "base-module",
								},
							},
						},
					},
				},
			},
		},
		inShortenEnumLeafNames:       true,
		wantUncompressFailDueToClash: true,
		wantUniqueNamesMap: map[string]string{
			"enum-a": "GreatGranGranA_Foo",
			"enum-b": "GreatGranGranB_Foo",
		},
		wantUniqueNamesMapNoUnderscores: map[string]string{
			"enum-a": "GreatGranGranAFoo",
			"enum-b": "GreatGranGranBFoo",
		},
	}, {
		name: "cannot resolve name clash due to names from module-level and grandparent-level disambiguation already in definedEnums",
		inDefinedEnums: map[string]bool{
			"Baz":                       true,
			"Foo":                       true,
			"SupportModule_Foo":         true,
			"GranGranB_Foo":             true,
			"BaseModule_ParentA_Enum":   true,
			"BaseModule_GranGranB_Enum": true,
		},
		inDefinedEnumsNoUnderscores: map[string]bool{
			"Baz":                      true,
			"Foo":                      true,
			"SupportModuleFoo":         true,
			"GranGranBFoo":             true,
			"BaseModuleParentA_Enum":   true,
			"BaseModuleGranGranB_Enum": true,
		},
		inNameClashSets: map[string]map[string]*yang.Entry{
			"Foo": {
				"enum-a": &yang.Entry{
					Name: "enum-a",
					Node: &yang.Enum{
						Parent: &yang.Container{
							Name: "parent-a",
							Parent: &yang.Container{
								Name: "gran-gran-a",
								Parent: &yang.Module{
									Name: "base-module",
								},
							},
						},
					},
					Parent: &yang.Entry{
						Name: "parent-a",
						Parent: &yang.Entry{
							Name: "gran-gran-a",
							Parent: &yang.Entry{
								Name: "base-module",
							},
						},
					},
				},
				"enum-b": &yang.Entry{
					Name: "enum-b",
					Node: &yang.Enum{
						Parent: &yang.Container{
							Name: "parent-b",
							Parent: &yang.Container{
								Name: "gran-gran-b",
								Parent: &yang.Module{
									Name: "base-module",
								},
							},
						},
					},
					Parent: &yang.Entry{
						Name: "parent-b",
						Parent: &yang.Entry{
							Name: "gran-gran-b",
							Parent: &yang.Entry{
								Name: "base-module",
							},
						},
					},
				},
			},
		},
		inShortenEnumLeafNames:       true,
		wantUncompressFailDueToClash: true,
		wantErrSubstr:                "cannot resolve enumeration name clash",
	}, {
		name: "cannot resolve name clash due to names from module-level and grandparent-level disambiguation already in definedEnums, with names not shortened",
		inDefinedEnums: map[string]bool{
			"BaseModule_Baz":            true,
			"BaseModule_Foo":            true,
			"SupportModule_Foo":         true,
			"BaseModule_GranGranB_Foo":  true,
			"BaseModule_ParentA_Enum":   true,
			"BaseModule_GranGranB_Enum": true,
		},
		inDefinedEnumsNoUnderscores: map[string]bool{
			"BaseModuleBaz":           true,
			"BaseModuleFoo":           true,
			"SupportModuleFoo":        true,
			"BaseModuleGranGranBFoo":  true,
			"BaseModuleParentAEnum":   true,
			"BaseModuleGranGranBEnum": true,
		},
		inNameClashSets: map[string]map[string]*yang.Entry{
			"BaseModule_Foo": {
				"enum-a": &yang.Entry{
					Name: "enum-a",
					Node: &yang.Enum{
						Parent: &yang.Container{
							Name: "parent-a",
							Parent: &yang.Container{
								Name: "gran-gran-a",
								Parent: &yang.Module{
									Name: "base-module",
								},
							},
						},
					},
					Parent: &yang.Entry{
						Name: "parent-a",
						Node: &yang.Container{
							Name: "parent-a",
							Parent: &yang.Container{
								Name: "gran-gran-a",
								Parent: &yang.Module{
									Name: "base-module",
								},
							},
						},
						Parent: &yang.Entry{
							Name: "gran-gran-a",
							Node: &yang.Container{
								Name: "gran-gran-a",
								Parent: &yang.Module{
									Name: "base-module",
								},
							},
							Parent: &yang.Entry{
								Name: "base-module",
								Node: &yang.Module{
									Name: "base-module",
								},
							},
						},
					},
				},
				"enum-b": &yang.Entry{
					Name: "enum-b",
					Node: &yang.Enum{
						Parent: &yang.Container{
							Name: "parent-b",
							Parent: &yang.Container{
								Name: "gran-gran-b",
								Parent: &yang.Module{
									Name: "base-module",
								},
							},
						},
					},
					Parent: &yang.Entry{
						Name: "parent-b",
						Node: &yang.Container{
							Name: "parent-b",
							Parent: &yang.Container{
								Name: "gran-gran-b",
								Parent: &yang.Module{
									Name: "base-module",
								},
							},
						},
						Parent: &yang.Entry{
							Name: "gran-gran-b",
							Node: &yang.Container{
								Name: "gran-gran-b",
								Parent: &yang.Module{
									Name: "base-module",
								},
							},
							Parent: &yang.Entry{
								Name: "base-module",
								Node: &yang.Module{
									Name: "base-module",
								},
							},
						},
					},
				},
			},
		},
		wantUncompressFailDueToClash: true,
		wantErrSubstr:                "cannot resolve enumeration name clash",
	}, {
		name: "cannot resolve name clash at grandparent due to camel case lossiness",
		inDefinedEnums: map[string]bool{
			"Baz": true,
		},
		inDefinedEnumsNoUnderscores: map[string]bool{
			"Baz": true,
		},
		inNameClashSets: map[string]map[string]*yang.Entry{
			"Foo": {
				"enum-a": &yang.Entry{
					Name: "enum-a",
					Node: &yang.Enum{
						Parent: &yang.Container{
							Name: "parent",
							Parent: &yang.Container{
								Name: "gran-gran-a",
								Parent: &yang.Module{
									Name: "base-module",
								},
							},
						},
					},
					Parent: &yang.Entry{
						Name: "parent",
						Parent: &yang.Entry{
							Name: "gran-gran-a",
							Parent: &yang.Entry{
								Name: "base-module",
							},
						},
					},
				},
				"enum-b": &yang.Entry{
					Name: "enum-b",
					Node: &yang.Enum{
						Parent: &yang.Container{
							Name: "parent",
							Parent: &yang.Container{
								Name: "gran-granA",
								Parent: &yang.Module{
									Name: "base-module",
								},
							},
						},
					},
					Parent: &yang.Entry{
						Name: "parent",
						Parent: &yang.Entry{
							Name: "gran-granA",
							Parent: &yang.Entry{
								Name: "base-module",
							},
						},
					},
				},
			},
		},
		inShortenEnumLeafNames:       true,
		wantUncompressFailDueToClash: true,
		wantErrSubstr:                "cannot resolve enumeration name clash",
	}, {
		name: "error for invalid input when camel-case module name is not prefix of clash name",
		inDefinedEnums: map[string]bool{
			"BaseModule_Baz": true,
		},
		inDefinedEnumsNoUnderscores: map[string]bool{
			"BaseModuleBaz": true,
		},
		inNameClashSets: map[string]map[string]*yang.Entry{
			"BaseMod_Foo": {
				"enum-a": &yang.Entry{
					Name: "enum-a",
					Node: &yang.Enum{
						Parent: &yang.Container{
							Name: "parent",
							Parent: &yang.Module{
								Name: "base-module",
							},
						},
					},
					Parent: &yang.Entry{
						Name: "parent",
						Parent: &yang.Entry{
							Name: "base-module",
						},
					},
				},
				"enum-b": &yang.Entry{
					Name: "enum-b",
					Node: &yang.Enum{
						Parent: &yang.Container{
							Name: "parent",
							Parent: &yang.Module{
								Name: "base-module",
							},
						},
					},
					Parent: &yang.Entry{
						Name: "parent",
						Parent: &yang.Entry{
							Name: "base-module",
						},
					},
				},
			},
		},
		wantUncompressFailDueToClash: true,
		wantErrSubstr:                "provided clashName does not start with its defining module name",
	}, {
		name: "intersecting name clash sets",
		inDefinedEnums: map[string]bool{
			"Baz": true,
		},
		inDefinedEnumsNoUnderscores: map[string]bool{
			"Baz": true,
		},
		inNameClashSets: map[string]map[string]*yang.Entry{
			"Bar": {
				"enum-a": &yang.Entry{
					Name: "enum-a",
				},
			},
			"Foo": {
				"enum-a": &yang.Entry{
					Name: "enum-a",
				},
				"enum-b": &yang.Entry{
					Name: "enum-b",
				},
			},
		},
		inShortenEnumLeafNames: true,
		wantErrSubstr:          `enumKey "enum-a" has been given a second name`,
	}}

	for _, tt := range tests {
		for noUnderscores, wantUniqueNamesMap := range map[bool]map[string]string{
			false: tt.wantUniqueNamesMap,
			true:  tt.wantUniqueNamesMapNoUnderscores} {

			inDefinedEnums := tt.inDefinedEnums
			if noUnderscores {
				inDefinedEnums = tt.inDefinedEnumsNoUnderscores
			}

			for compressPaths := range map[bool]struct{}{false: {}, true: {}} {
				t.Run(tt.name+fmt.Sprintf("@compressPaths:%v,noUnderscores:%v,shortenEnumLeafNames:%v", compressPaths, noUnderscores, tt.inShortenEnumLeafNames), func(t *testing.T) {
					s := newEnumGenState()
					for k, v := range inDefinedEnums {
						// Copy the values as this map may be modified.
						s.definedEnums[k] = v
					}
					nameClashSets := map[string]map[string]*yang.Entry{}
					for k, v := range tt.inNameClashSets {
						if noUnderscores {
							k = strings.ReplaceAll(k, "_", "")
						}
						nameClashSets[k] = v
					}
					gotUniqueNamesMap, err := s.resolveNameClashSet(nameClashSets, compressPaths, noUnderscores, tt.inShortenEnumLeafNames)
					wantErrSubstr := tt.wantErrSubstr
					if !compressPaths && tt.wantUncompressFailDueToClash {
						wantErrSubstr = "clash in enumerated name occurred despite paths being uncompressed"
					}
					if diff := errdiff.Substring(err, wantErrSubstr); diff != "" {
						if err == nil {
							t.Errorf("gotUniqueNamesMap: %v", gotUniqueNamesMap)
						}
						t.Fatalf("did not get expected error:\n%s", diff)
					}
					if wantErrSubstr != "" {
						return
					}

					if diff := cmp.Diff(gotUniqueNamesMap, wantUniqueNamesMap); diff != "" {
						t.Errorf("TestResolveNameClashSet (-got, +want):\n%s", diff)
					}
				})
			}
		}
	}
}

// TestFindEnumSet tests the findEnumSet function, ensuring that it performs
// deduplication of re-used identities, and re-used typedefs. For inline
// definitions, the enumerations should be duplicated. Tests are performed with
// compression set to both true and false.
func TestFindEnumSet(t *testing.T) {
	tests := []struct {
		name                    string
		in                      map[string]*yang.Entry
		inOmitUnderscores       bool
		inSkipEnumDeduplication bool
		inShortenEnumLeafNames  bool
		wantCompressed          map[string]*yangEnum
		wantUncompressed        map[string]*yangEnum
		// wantUseDefiningModuleForTypedefEnumNames should be specified whenever the output
		// is expected to be different when useDefiningModuleForTypedefEnumNames is set to
		// true. Its output should be compression independent since typedef enum names are
		// compression independent, so wantSame must be true when specified. When not
		// specified, useDefiningModuleForTypedefEnumNames is expected to not have an effect
		// on the output.
		wantUseDefiningModuleForTypedefEnumNames        map[string]*yangEnum
		wantEnumSetCompressed                           *enumSet
		wantEnumSetUncompressed                         *enumSet
		wantEnumSetUseDefiningModuleForTypedefEnumNames *enumSet
		// Whether to expect same compressed/uncompressed output.
		wantSame bool
		// wantUncompressFailDueToClash means the uncompressed test run will fail in
		// deviation from the compressed case due to existence of a name clash, which can
		// only be resolved for compressed paths.
		wantUncompressFailDueToClash bool
		wantErrSubstr                string
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
		inShortenEnumLeafNames: true,
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
		wantEnumSetCompressed: &enumSet{
			uniqueIdentityNames: map[string]string{
				"test-module/base-identity": "TestModule_BaseIdentity",
			},
		},
		wantSame: true,
	}, {
		name: "simple identityref that conflicts",
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
			"/container/config/identityref-leaf2": {
				Name: "identityref-leaf2",
				Type: &yang.YangType{
					Name: "identityref",
					IdentityBase: &yang.Identity{
						Name: "baseIdentity",
						Parent: &yang.Module{
							Name: "test-module",
						},
					},
				},
			},
		},
		inShortenEnumLeafNames: true,
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
		wantSame:      true,
		wantErrSubstr: "identity name conflict",
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
			"/container/state/enumeration-leaf": {
				Name: "enumeration-leaf",
				Type: &yang.YangType{
					Name: "enumeration",
					Enum: &yang.EnumType{},
				},
				Node: &yang.Enum{
					Name: "enumeration-leaf",
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
		inShortenEnumLeafNames: true,
		wantCompressed: map[string]*yangEnum{
			"Container_EnumerationLeaf": {
				name: "Container_EnumerationLeaf",
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
		wantEnumSetCompressed: &enumSet{
			uniqueEnumeratedLeafNames: map[string]string{
				"/base-module/container/config/enumeration-leaf": "Container_EnumerationLeaf",
			},
		},
		wantEnumSetUncompressed: &enumSet{
			uniqueEnumeratedLeafNames: map[string]string{
				"/base-module/container/config/enumeration-leaf": "BaseModule_Container_Config_EnumerationLeaf",
				"/base-module/container/state/enumeration-leaf":  "BaseModule_Container_State_EnumerationLeaf",
			},
		},
	}, {
		name: "simple enumeration without shortened names",
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
			"/container/state/enumeration-leaf": {
				Name: "enumeration-leaf",
				Type: &yang.YangType{
					Name: "enumeration",
					Enum: &yang.EnumType{},
				},
				Node: &yang.Enum{
					Name: "enumeration-leaf",
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
		wantEnumSetCompressed: &enumSet{
			uniqueEnumeratedLeafNames: map[string]string{
				"/base-module/container/config/enumeration-leaf": "BaseModule_Container_EnumerationLeaf",
			},
		},
		wantEnumSetUncompressed: &enumSet{
			uniqueEnumeratedLeafNames: map[string]string{
				"/base-module/container/config/enumeration-leaf": "BaseModule_Container_Config_EnumerationLeaf",
				"/base-module/container/state/enumeration-leaf":  "BaseModule_Container_State_EnumerationLeaf",
			},
		},
	}, {
		name: "simple enumeration unresolvable conflicting names due to camel-case lossiness",
		in: map[string]*yang.Entry{
			"/container/state/enumeration-leaf": {
				Name: "enumeration-leaf",
				Type: &yang.YangType{
					Name: "enumeration",
					Enum: &yang.EnumType{},
				},
				Node: &yang.Enum{
					Name: "enumeration-leaf",
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
			"/container/state/enumerationLeaf": {
				Name: "enumerationLeaf",
				Type: &yang.YangType{
					Name: "enumeration",
					Enum: &yang.EnumType{},
				},
				Node: &yang.Enum{
					Name: "enumerationLeaf",
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
		inShortenEnumLeafNames:       true,
		wantUncompressFailDueToClash: true,
		wantErrSubstr:                "cannot resolve enumeration name clash",
	}, {
		name: "simple enumeration with naming conflict due to same grandparent context",
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
			"/container/state/enumeration-leaf": {
				Name: "enumeration-leaf",
				Type: &yang.YangType{
					Name: "enumeration",
					Enum: &yang.EnumType{},
				},
				Node: &yang.Enum{
					Name: "enumeration-leaf",
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
			"/outer-container/container/config/enumeration-leaf": {
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
							Parent: &yang.Container{
								Name: "outer-container",
								Parent: &yang.Module{
									Name: "base-module",
								},
							},
						},
					},
				},
				Parent: &yang.Entry{
					Name: "config",
					Parent: &yang.Entry{
						Name: "container",
						Parent: &yang.Entry{
							Name:   "outer-container",
							Parent: &yang.Entry{Name: "base-module"},
						},
					},
				},
			},
		},
		inShortenEnumLeafNames: true,
		wantCompressed: map[string]*yangEnum{
			"Container_EnumerationLeaf": {
				name: "Container_EnumerationLeaf",
				entry: &yang.Entry{
					Name: "enumeration-leaf",
					Type: &yang.YangType{
						Enum: &yang.EnumType{},
					},
				},
			},
			"OuterContainer_Container_EnumerationLeaf": {
				name: "OuterContainer_Container_EnumerationLeaf",
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
			"BaseModule_OuterContainer_Container_Config_EnumerationLeaf": {
				name: "BaseModule_OuterContainer_Container_Config_EnumerationLeaf",
				entry: &yang.Entry{
					Name: "enumeration-leaf",
					Type: &yang.YangType{
						Enum: &yang.EnumType{},
					},
				},
			},
		},
		wantEnumSetCompressed: &enumSet{
			uniqueEnumeratedLeafNames: map[string]string{
				"/base-module/container/config/enumeration-leaf":                 "Container_EnumerationLeaf",
				"/base-module/outer-container/container/config/enumeration-leaf": "OuterContainer_Container_EnumerationLeaf",
			},
		},
		wantEnumSetUncompressed: &enumSet{
			uniqueEnumeratedLeafNames: map[string]string{
				"/base-module/container/config/enumeration-leaf":                 "BaseModule_Container_Config_EnumerationLeaf",
				"/base-module/container/state/enumeration-leaf":                  "BaseModule_Container_State_EnumerationLeaf",
				"/base-module/outer-container/container/config/enumeration-leaf": "BaseModule_OuterContainer_Container_Config_EnumerationLeaf",
			},
		},
	}, {
		name: "simple enumeration with naming conflict due to same grandparent context without shortened names",
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
					Node: &yang.Container{
						Name: "config",
						Parent: &yang.Container{
							Name: "container",
							Parent: &yang.Module{
								Name: "base-module",
							},
						},
					},
					Parent: &yang.Entry{
						Name: "container",
						Node: &yang.Container{
							Name: "container",
							Parent: &yang.Module{
								Name: "base-module",
							},
						},
						Parent: &yang.Entry{
							Node: &yang.Module{
								Name: "base-module",
							},
							Name: "base-module",
						},
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
					Name: "enumeration-leaf",
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
					Node: &yang.Container{
						Name: "state",
						Parent: &yang.Container{
							Name: "container",
							Parent: &yang.Module{
								Name: "base-module",
							},
						},
					},
					Parent: &yang.Entry{
						Name: "container",
						Node: &yang.Container{
							Name: "container",
							Parent: &yang.Module{
								Name: "base-module",
							},
						},
						Parent: &yang.Entry{
							Node: &yang.Module{
								Name: "base-module",
							},
							Name: "base-module",
						},
					},
				},
			},
			"/outer-container/container/config/enumeration-leaf": {
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
							Parent: &yang.Container{
								Name: "outer-container",
								Parent: &yang.Module{
									Name: "base-module",
								},
							},
						},
					},
				},
				Parent: &yang.Entry{
					Name: "config",
					Node: &yang.Container{
						Name: "config",
						Parent: &yang.Container{
							Name: "container",
							Parent: &yang.Container{
								Name: "outer-container",
								Parent: &yang.Module{
									Name: "base-module",
								},
							},
						},
					},
					Parent: &yang.Entry{
						Name: "container",
						Node: &yang.Container{
							Name: "container",
							Parent: &yang.Container{
								Name: "outer-container",
								Parent: &yang.Module{
									Name: "base-module",
								},
							},
						},
						Parent: &yang.Entry{
							Name: "outer-container",
							Node: &yang.Container{
								Name: "outer-container",
								Parent: &yang.Module{
									Name: "base-module",
								},
							},
							Parent: &yang.Entry{
								Node: &yang.Module{
									Name: "base-module",
								},
								Name: "base-module",
							},
						},
					},
				},
			},
		},
		wantCompressed: map[string]*yangEnum{
			"BaseModule_Container_EnumerationLeaf": {
				name: "Container_EnumerationLeaf",
				entry: &yang.Entry{
					Name: "enumeration-leaf",
					Type: &yang.YangType{
						Enum: &yang.EnumType{},
					},
				},
			},
			"BaseModule_OuterContainer_Container_EnumerationLeaf": {
				name: "OuterContainer_Container_EnumerationLeaf",
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
			"BaseModule_OuterContainer_Container_Config_EnumerationLeaf": {
				name: "BaseModule_OuterContainer_Container_Config_EnumerationLeaf",
				entry: &yang.Entry{
					Name: "enumeration-leaf",
					Type: &yang.YangType{
						Enum: &yang.EnumType{},
					},
				},
			},
		},
		wantEnumSetCompressed: &enumSet{
			uniqueEnumeratedLeafNames: map[string]string{
				"/base-module/container/config/enumeration-leaf":                 "BaseModule_Container_EnumerationLeaf",
				"/base-module/outer-container/container/config/enumeration-leaf": "BaseModule_OuterContainer_Container_EnumerationLeaf",
			},
		},
		wantEnumSetUncompressed: &enumSet{
			uniqueEnumeratedLeafNames: map[string]string{
				"/base-module/container/config/enumeration-leaf":                 "BaseModule_Container_Config_EnumerationLeaf",
				"/base-module/container/state/enumeration-leaf":                  "BaseModule_Container_State_EnumerationLeaf",
				"/base-module/outer-container/container/config/enumeration-leaf": "BaseModule_OuterContainer_Container_Config_EnumerationLeaf",
			},
		},
	}, {
		name: "simple enumeration with same grandparent context but are from different modules",
		in: map[string]*yang.Entry{
			"/base-module/container/config/enumeration-leaf": {
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
			"/base-module/container/state/enumeration-leaf": {
				Name: "enumeration-leaf",
				Type: &yang.YangType{
					Name: "enumeration",
					Enum: &yang.EnumType{},
				},
				Node: &yang.Enum{
					Name: "enumeration-leaf",
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
			"/base-module2/container/config/enumeration-leaf": {
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
								Name: "base-module2",
							},
						},
					},
				},
				Parent: &yang.Entry{
					Name: "config",
					Parent: &yang.Entry{
						Name:   "container",
						Parent: &yang.Entry{Name: "base-module2"},
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
			"BaseModule2_Container_EnumerationLeaf": {
				name: "BaseModule2_Container_EnumerationLeaf",
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
			"BaseModule2_Container_Config_EnumerationLeaf": {
				name: "BaseModule2_Container_State_EnumerationLeaf",
				entry: &yang.Entry{
					Name: "enumeration-leaf",
					Type: &yang.YangType{
						Enum: &yang.EnumType{},
					},
				},
			},
		},
		wantEnumSetCompressed: &enumSet{
			uniqueEnumeratedLeafNames: map[string]string{
				"/base-module/container/config/enumeration-leaf":  "BaseModule_Container_EnumerationLeaf",
				"/base-module2/container/config/enumeration-leaf": "BaseModule2_Container_EnumerationLeaf",
			},
		},
		wantEnumSetUncompressed: &enumSet{
			uniqueEnumeratedLeafNames: map[string]string{
				"/base-module/container/config/enumeration-leaf":  "BaseModule_Container_Config_EnumerationLeaf",
				"/base-module/container/state/enumeration-leaf":   "BaseModule_Container_State_EnumerationLeaf",
				"/base-module2/container/config/enumeration-leaf": "BaseModule2_Container_Config_EnumerationLeaf",
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
					Base: &yang.Type{
						Name: "enumeration",
						Parent: &yang.Typedef{
							Name: "derived-enumeration",
							Parent: &yang.Module{
								Name: "typedef-module",
							},
						},
					},
				},
				Node: &yang.Enum{
					Name: "derived-enumeration",
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
					Base: &yang.Type{
						Name: "enumeration",
						Parent: &yang.Typedef{
							Name: "derived-enumeration",
							Parent: &yang.Module{
								Name: "typedef-module",
							},
						},
					},
				},
				Node: &yang.Enum{
					Name: "derived-enumeration",
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
					Name: "state",
					Node: &yang.Container{Name: "state"}, Parent: &yang.Entry{
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
		inShortenEnumLeafNames: true,
		wantUseDefiningModuleForTypedefEnumNames: map[string]*yangEnum{
			"TypedefModule_DerivedEnumeration": {
				name: "TypedefModule_DerivedEnumeration",
				entry: &yang.Entry{
					Name: "enumeration-leaf",
					Type: &yang.YangType{
						Enum: &yang.EnumType{},
					},
				},
			},
		},
		wantEnumSetUseDefiningModuleForTypedefEnumNames: &enumSet{
			uniqueEnumeratedTypedefNames: map[string]string{
				"typedef-module/derived-enumeration": "TypedefModule_DerivedEnumeration",
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
		wantEnumSetCompressed: &enumSet{
			uniqueEnumeratedTypedefNames: map[string]string{
				"typedef-module/derived-enumeration": "BaseModule_DerivedEnumeration",
			},
		},
		wantSame: true,
	}, {
		name: "typedef which is an enumeration name conflict due to camelcase lossiness",
		in: map[string]*yang.Entry{
			"/container/config/enumeration-leaf": {
				Name: "enumeration-leaf",
				Type: &yang.YangType{
					Name: "derived-enumeration",
					Enum: &yang.EnumType{},
					Base: &yang.Type{
						Name: "enumeration",
						Parent: &yang.Typedef{
							Name: "derived-enumeration",
							Parent: &yang.Module{
								Name: "typedef-module",
							},
						},
					},
				},
				Node: &yang.Enum{
					Name: "derived-enumeration",
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
			"/super/container/state/enumeration-leaf": {
				Name: "enumeration-leaf",
				Type: &yang.YangType{
					Name: "derivedEnumeration",
					Enum: &yang.EnumType{},
					Base: &yang.Type{
						Name: "enumeration",
						Parent: &yang.Typedef{
							Name: "derivedEnumeration",
							Parent: &yang.Module{
								Name: "typedef-module",
							},
						},
					},
				},
				Node: &yang.Enum{
					Name: "derivedEnumeration",
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
							Name: "super",
							Node: &yang.Container{Name: "super"},
							Parent: &yang.Entry{
								Name: "base-module",
								Node: &yang.Module{Name: "base-module"},
							},
						},
					},
				},
			},
		},
		inShortenEnumLeafNames: true,
		wantSame:               true,
		wantErrSubstr:          "enumerated typedef name conflict",
	}, {
		name: "union which contains a typedef enumeration",
		in: map[string]*yang.Entry{
			"/container/config/e": {
				Name: "e",
				Type: &yang.YangType{
					Name: "union",
					Kind: yang.Yunion,
					Type: []*yang.YangType{
						{
							Name: "derived",
							Kind: yang.Yenum,
							Enum: &yang.EnumType{},
							Base: &yang.Type{
								Name: "enumeration",
								Parent: &yang.Typedef{
									Name: "derived",
									Parent: &yang.Module{
										Name: "typedef-module",
									},
								},
							},
						},
						{Kind: yang.Ystring},
					},
				},
				Node: &yang.Enum{
					Name: "e",
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
		inShortenEnumLeafNames: true,
		wantCompressed: map[string]*yangEnum{
			"BaseModule_Derived_Enum": {
				name: "BaseModule_Derived_Enum",
				entry: &yang.Entry{
					Name: "e",
					Type: &yang.YangType{
						Kind: yang.Yunion,
						Type: []*yang.YangType{
							{
								Name: "derived",
								Kind: yang.Yenum,
								Enum: &yang.EnumType{},
								Base: &yang.Type{
									Name: "enumeration",
									Parent: &yang.Typedef{
										Name: "enum-container",
										Parent: &yang.Module{
											Name: "typedef-module",
										},
									},
								},
							},
							{Kind: yang.Ystring},
						},
					},
				},
			},
		},
		wantEnumSetCompressed: &enumSet{
			uniqueEnumeratedTypedefNames: map[string]string{
				"typedef-module/derived": "BaseModule_Derived_Enum",
			},
		},
		wantUseDefiningModuleForTypedefEnumNames: map[string]*yangEnum{
			"TypedefModule_Derived_Enum": {
				name: "TypedefModule_Derived_Enum",
				entry: &yang.Entry{
					Name: "e",
					Type: &yang.YangType{
						Kind: yang.Yunion,
						Type: []*yang.YangType{
							{
								Name: "derived",
								Kind: yang.Yenum,
								Enum: &yang.EnumType{},
								Base: &yang.Type{
									Name: "enumeration",
									Parent: &yang.Typedef{
										Name: "enum-container",
										Parent: &yang.Module{
											Name: "typedef-module",
										},
									},
								},
							},
							{Kind: yang.Ystring},
						},
					},
				},
			},
		},
		wantEnumSetUseDefiningModuleForTypedefEnumNames: &enumSet{
			uniqueEnumeratedTypedefNames: map[string]string{
				"typedef-module/derived": "TypedefModule_Derived_Enum",
			},
		},
		wantSame: true,
	}, {
		name: "typedef of union that contains an enumeration (not typedef)",
		in: map[string]*yang.Entry{
			"/container/state/e": {
				Name: "e",
				Type: &yang.YangType{
					Name: "derived-type",
					Kind: yang.Yunion,
					Type: []*yang.YangType{{
						Name: "union",
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
				},
				Node: &yang.Enum{
					Name: "e",
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
		inShortenEnumLeafNames: true,
		wantCompressed: map[string]*yangEnum{
			"Container_E": {
				name: "Container_E",
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
		wantEnumSetCompressed: &enumSet{
			uniqueEnumeratedLeafNames: map[string]string{
				"container:/base-module/container/state/e": "Container_E",
			},
		},
		wantEnumSetUncompressed: &enumSet{
			uniqueEnumeratedLeafNames: map[string]string{
				"e:/base-module/container/state/e": "BaseModule_Container_State_E",
			},
		},
	}, {
		name: "typedef union with a typedef enumeration",
		in: map[string]*yang.Entry{
			"/container/config/enumeration-leaf": {
				Name: "enumeration-leaf",
				Type: &yang.YangType{
					Name: "derived-union-enum",
					Kind: yang.Yunion,
					Type: []*yang.YangType{
						{
							Name: "derived-enumeration",
							Kind: yang.Yenum,
							Enum: &yang.EnumType{},
							Base: &yang.Type{
								Name: "enumeration",
								Parent: &yang.Typedef{
									Name: "derived-enumeration",
									Parent: &yang.Module{
										Name: "typedef-module",
									},
								},
							},
						},
						{Kind: yang.Yuint32},
					},
				},
				Node: &yang.Enum{
					Name: "enumeration-leaf",
					Parent: &yang.Module{
						Name: "base-module",
					},
				},
			},
			"/container/state/enumeration-leaf": {
				Name: "enumeration-leaf",
				Type: &yang.YangType{
					Name: "derived-union-enum",
					Kind: yang.Yunion,
					Type: []*yang.YangType{
						{
							Name: "derived-enumeration",
							Kind: yang.Yenum,
							Enum: &yang.EnumType{},
							Base: &yang.Type{
								Name: "enumeration",
								Parent: &yang.Typedef{
									Name: "derived-enumeration",
									Parent: &yang.Module{
										Name: "typedef-module",
									},
								},
							},
						},
						{Kind: yang.Yuint32},
					},
				},
				Node: &yang.Enum{
					Name: "enumeration-leaf",
					Parent: &yang.Module{
						Name: "base-module",
					},
				},
			},
		},
		inShortenEnumLeafNames: true,
		wantCompressed: map[string]*yangEnum{
			"BaseModule_DerivedEnumeration_Enum": {
				name: "BaseModule_DerivedEnumeration_Enum",
				entry: &yang.Entry{
					Name: "enumeration-leaf",
					Type: &yang.YangType{
						Enum: &yang.EnumType{},
					},
				},
			},
		},
		wantEnumSetCompressed: &enumSet{
			uniqueEnumeratedTypedefNames: map[string]string{
				"typedef-module/derived-enumeration": "BaseModule_DerivedEnumeration_Enum",
			},
		},
		wantUseDefiningModuleForTypedefEnumNames: map[string]*yangEnum{
			"TypedefModule_DerivedEnumeration_Enum": {
				name: "TypedefModule_DerivedEnumeration_Enum",
				entry: &yang.Entry{
					Name: "enumeration-leaf",
					Type: &yang.YangType{
						Enum: &yang.EnumType{},
					},
				},
			},
		},
		wantEnumSetUseDefiningModuleForTypedefEnumNames: &enumSet{
			uniqueEnumeratedTypedefNames: map[string]string{
				"typedef-module/derived-enumeration": "TypedefModule_DerivedEnumeration_Enum",
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
					Base: &yang.Type{
						Name: "identityref",
						Parent: &yang.Typedef{
							Name: "derived-identityref",
							Parent: &yang.Container{
								Name: "identity-container",
								Parent: &yang.Module{
									Name: "identity-module",
								},
							},
						},
					},
					IdentityBase: &yang.Identity{
						Name: "base-identityref",
						Parent: &yang.Module{
							Name: "identity-module",
						},
					},
				},
				Node: &yang.Leaf{
					Name: "identityref-leaf",
					Parent: &yang.Module{
						Name: "base-module",
					},
				},
			},
			"/container/state/identityref-leaf": {
				Name: "identityref-leaf",
				Type: &yang.YangType{
					Name: "derived-identityref",
					Base: &yang.Type{
						Name: "identityref",
						Parent: &yang.Typedef{
							Name: "derived-identityref",
							Parent: &yang.Container{
								Name: "identity-container",
								Parent: &yang.Module{
									Name: "identity-module",
								},
							},
						},
					},
					IdentityBase: &yang.Identity{
						Name: "base-identityref",
						Parent: &yang.Module{
							Name: "identity-module",
						},
					},
				},
				Node: &yang.Leaf{
					Name: "identityref-leaf",
					Parent: &yang.Module{
						Name: "base-module",
					},
				},
			},
		},
		inShortenEnumLeafNames: true,
		wantCompressed: map[string]*yangEnum{
			"BaseModule_DerivedIdentityref": {
				name: "BaseModule_DerivedIdentityref",
				entry: &yang.Entry{
					Name: "identityref-leaf",
					Type: &yang.YangType{
						IdentityBase: &yang.Identity{
							Name: "base-identityref",
							Parent: &yang.Module{
								Name: "identity-module",
							},
						},
					},
				},
			},
		},
		wantEnumSetCompressed: &enumSet{
			uniqueEnumeratedTypedefNames: map[string]string{
				"identity-module/identity-container/derived-identityref": "BaseModule_DerivedIdentityref",
			},
		},
		wantUseDefiningModuleForTypedefEnumNames: map[string]*yangEnum{
			"IdentityModule_DerivedIdentityref": {
				name: "IdentityModule_DerivedIdentityref",
				entry: &yang.Entry{
					Name: "identityref-leaf",
					Type: &yang.YangType{
						IdentityBase: &yang.Identity{
							Name: "base-identityref",
							Parent: &yang.Module{
								Name: "identity-module",
							},
						},
					},
				},
			},
		},
		wantEnumSetUseDefiningModuleForTypedefEnumNames: &enumSet{
			uniqueEnumeratedTypedefNames: map[string]string{
				"identity-module/identity-container/derived-identityref": "IdentityModule_DerivedIdentityref",
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
					Name: "invalid-identityref-leaf",
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
		inShortenEnumLeafNames: true,
		wantErrSubstr:          "an identity with a nil base",
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
		inShortenEnumLeafNames: true,
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
		wantEnumSetCompressed: &enumSet{
			uniqueIdentityNames: map[string]string{
				"base-module/base-identity": "BaseModule_BaseIdentity",
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
					Name: "union-typedef-identityref",
					Parent: &yang.Module{
						Name: "test-module",
					},
				},
			},
		},
		inShortenEnumLeafNames: true,
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
		wantEnumSetCompressed: &enumSet{
			uniqueIdentityNames: map[string]string{
				"base-module/base-identity": "BaseModule_BaseIdentity",
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
					Name: "typedef-union-identityref",
					Parent: &yang.Module{
						Name: "test-module",
					},
				},
			},
		},
		inShortenEnumLeafNames: true,
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
		wantEnumSetCompressed: &enumSet{
			uniqueIdentityNames: map[string]string{
				"base-module/base-identity": "BaseModule_BaseIdentity",
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
		inShortenEnumLeafNames: true,
		wantErrSubstr:          "multiple enumerated types within a single enumeration not supported",
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
		inShortenEnumLeafNames: true,
		wantErrSubstr:          "enumerated type had an empty union within it",
	}, {
		name: "union of unions that contains an enumeration",
		in: map[string]*yang.Entry{
			"/container/state/e": {
				Name: "e",
				Type: &yang.YangType{
					Name: "union",
					Kind: yang.Yunion,
					Type: []*yang.YangType{{
						Name: "union",
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
					Name: "e",
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
		inShortenEnumLeafNames: true,
		wantCompressed: map[string]*yangEnum{
			"Container_E": {
				name: "Container_E",
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
		wantEnumSetCompressed: &enumSet{
			uniqueEnumeratedLeafNames: map[string]string{
				"/base-module/container/state/e": "Container_E",
			},
		},
		wantEnumSetUncompressed: &enumSet{
			uniqueEnumeratedLeafNames: map[string]string{
				"/base-module/container/state/e": "BaseModule_Container_State_E",
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
					Name: "enumeration-leaf",
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
		inShortenEnumLeafNames: true,
		wantCompressed: map[string]*yangEnum{
			"Container_EnumerationLeaf": {
				name: "Container_EnumerationLeaf",
				entry: &yang.Entry{
					Name: "enumeration-leaf",
					Type: &yang.YangType{
						Enum: &yang.EnumType{},
					},
				},
			},
			"Container_EnumerationLeafTwo": {
				name: "Container_EnumerationLeafTwo",
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
		wantEnumSetCompressed: &enumSet{
			uniqueEnumeratedLeafNames: map[string]string{
				"/base-module/container/config/enumeration-leaf":     "Container_EnumerationLeaf",
				"/base-module/container/config/enumeration-leaf-two": "Container_EnumerationLeafTwo",
			},
		},
		wantEnumSetUncompressed: &enumSet{
			uniqueEnumeratedLeafNames: map[string]string{
				"/base-module/container/config/enumeration-leaf":     "BaseModule_Container_Config_EnumerationLeaf",
				"/base-module/container/config/enumeration-leaf-two": "BaseModule_Container_Config_EnumerationLeafTwo",
				"/base-module/container/state/enumeration-leaf":      "BaseModule_Container_State_EnumerationLeaf",
				"/base-module/container/state/enumeration-leaf-two":  "BaseModule_Container_State_EnumerationLeafTwo",
			},
		},
	}, {
		name: "two enums with deduplication disabled, where duplication of enums is only happening for uncompressed due to compressed context being the same (i.e. config/state)",
		in: map[string]*yang.Entry{
			"/container/config/enumeration-leaf": {
				Name: "enumeration-leaf",
				Type: &yang.YangType{
					Name: "enumeration",
					Enum: &yang.EnumType{},
				},
				Node: &yang.Enum{
					Name: "enumeration-leaf",
					Parent: &yang.Grouping{
						Name: "foo",
						Parent: &yang.Module{
							Name: "base-module2",
						},
					},
				},
				Parent: &yang.Entry{
					Name: "config",
					Parent: &yang.Entry{
						Name:   "container",
						Parent: &yang.Entry{Name: "base-module2"},
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
					Name: "enumeration-leaf",
					Parent: &yang.Grouping{
						Name: "foo",
						Parent: &yang.Module{
							Name: "base-module2",
						},
					},
				},
				Parent: &yang.Entry{
					Name: "state",
					Parent: &yang.Entry{
						Name: "container",
						Parent: &yang.Entry{
							Name: "base-module2",
						},
					},
				},
			},
		},
		inShortenEnumLeafNames:  true,
		inSkipEnumDeduplication: true,
		wantCompressed: map[string]*yangEnum{
			"Container_EnumerationLeaf": {
				name: "Container_EnumerationLeaf",
				entry: &yang.Entry{
					Name: "enumeration-leaf",
					Type: &yang.YangType{
						Enum: &yang.EnumType{},
					},
				},
			},
		},
		wantUncompressed: map[string]*yangEnum{
			"BaseModule2_Container_State_EnumerationLeaf": {
				name: "BaseModule2_Container_State_EnumerationLeaf",
				entry: &yang.Entry{
					Name: "enumeration-leaf",
					Type: &yang.YangType{
						Enum: &yang.EnumType{},
					},
				},
			},
			"BaseModule2_Container_Config_EnumerationLeaf": {
				name: "BaseModule2_Container_Config_EnumerationLeaf",
				entry: &yang.Entry{
					Name: "enumeration-leaf",
					Type: &yang.YangType{
						Enum: &yang.EnumType{},
					},
				},
			},
		},
		wantEnumSetCompressed: &enumSet{
			uniqueEnumeratedLeafNames: map[string]string{
				"/base-module2/foo/enumeration-leaf:Container_EnumerationLeaf": "Container_EnumerationLeaf",
			},
		},
		wantEnumSetUncompressed: &enumSet{
			uniqueEnumeratedLeafNames: map[string]string{
				"/base-module2/container/config/enumeration-leaf": "BaseModule2_Container_Config_EnumerationLeaf",
				"/base-module2/container/state/enumeration-leaf":  "BaseModule2_Container_State_EnumerationLeaf",
			},
		},
	}, {
		name: "two enums with deduplication disabled, and where duplication occurs for both compressed and decompressed",
		in: map[string]*yang.Entry{
			"/container/apple/enumeration-leaf": {
				Name: "enumeration-leaf",
				Type: &yang.YangType{
					Name: "enumeration",
					Enum: &yang.EnumType{},
				},
				Node: &yang.Enum{
					Name: "enumeration-leaf",
					Parent: &yang.Grouping{
						Name: "foo",
						Parent: &yang.Module{
							Name: "base-module2",
						},
					},
				},
				Parent: &yang.Entry{
					Name: "apple",
					Parent: &yang.Entry{
						Name:   "cherry",
						Parent: &yang.Entry{Name: "base-module2"},
					},
				},
			},
			"/container/banana/enumeration-leaf": {
				Name: "enumeration-leaf",
				Type: &yang.YangType{
					Name: "enumeration",
					Enum: &yang.EnumType{},
				},
				Node: &yang.Enum{
					Name: "enumeration-leaf",
					Parent: &yang.Grouping{
						Name: "foo",
						Parent: &yang.Module{
							Name: "base-module2",
						},
					},
				},
				Parent: &yang.Entry{
					Name: "banana",
					Parent: &yang.Entry{
						Name: "donuts",
						Parent: &yang.Entry{
							Name: "base-module2",
						},
					},
				},
			},
		},
		inShortenEnumLeafNames:  true,
		inSkipEnumDeduplication: true,
		wantCompressed: map[string]*yangEnum{
			"Cherry_EnumerationLeaf": {
				name: "BaseModule2_Cherry_EnumerationLeaf",
				entry: &yang.Entry{
					Name: "enumeration-leaf",
					Type: &yang.YangType{
						Enum: &yang.EnumType{},
					},
				},
			},
			"Donuts_EnumerationLeaf": {
				name: "BaseModule2_Donuts_EnumerationLeaf",
				entry: &yang.Entry{
					Name: "enumeration-leaf",
					Type: &yang.YangType{
						Enum: &yang.EnumType{},
					},
				},
			},
		},
		wantUncompressed: map[string]*yangEnum{
			"BaseModule2_Cherry_Apple_EnumerationLeaf": {
				name: "BaseModule2_Cherry_Apple_EnumerationLeaf",
				entry: &yang.Entry{
					Name: "enumeration-leaf",
					Type: &yang.YangType{
						Enum: &yang.EnumType{},
					},
				},
			},
			"BaseModule2_Donuts_Banana_EnumerationLeaf": {
				name: "BaseModule2_Donuts_Banana_EnumerationLeaf",
				entry: &yang.Entry{
					Name: "enumeration-leaf",
					Type: &yang.YangType{
						Enum: &yang.EnumType{},
					},
				},
			},
		},
		wantEnumSetCompressed: &enumSet{
			uniqueEnumeratedLeafNames: map[string]string{
				"/base-module2/foo/enumeration-leaf:Cherry_EnumerationLeaf": "Cherry_EnumerationLeaf",
				"/base-module2/foo/enumeration-leaf:Donuts_EnumerationLeaf": "Donuts_EnumerationLeaf",
			},
		},
		wantEnumSetUncompressed: &enumSet{
			uniqueEnumeratedLeafNames: map[string]string{
				"/base-module2/cherry/apple/enumeration-leaf":  "BaseModule2_Cherry_Apple_EnumerationLeaf",
				"/base-module2/donuts/banana/enumeration-leaf": "BaseModule2_Donuts_Banana_EnumerationLeaf",
			},
		},
	}, {
		name: "two enums with deduplication disabled, and where duplication occurs for both compressed and decompressed but the enum contexts (grandparents) are the same",
		in: map[string]*yang.Entry{
			"/container/apple/enumeration-leaf": {
				Name: "enumeration-leaf",
				Type: &yang.YangType{
					Name: "enumeration",
					Enum: &yang.EnumType{},
				},
				Node: &yang.Enum{
					Name: "enumeration-leaf",
					Parent: &yang.Grouping{
						Name: "foo",
						Parent: &yang.Module{
							Name: "base-module2",
						},
					},
				},
				Parent: &yang.Entry{
					Name: "apple",
					Parent: &yang.Entry{
						Name:   "container",
						Parent: &yang.Entry{Name: "base-module2"},
					},
				},
			},
			"/container/banana/enumeration-leaf": {
				Name: "enumeration-leaf",
				Type: &yang.YangType{
					Name: "enumeration",
					Enum: &yang.EnumType{},
				},
				Node: &yang.Enum{
					Name: "enumeration-leaf",
					Parent: &yang.Grouping{
						Name: "foo",
						Parent: &yang.Module{
							Name: "base-module2",
						},
					},
				},
				Parent: &yang.Entry{
					Name: "banana",
					Parent: &yang.Entry{
						Name: "container",
						Parent: &yang.Entry{
							Name: "base-module2",
						},
					},
				},
			},
		},
		inShortenEnumLeafNames:  true,
		inSkipEnumDeduplication: true,
		wantCompressed: map[string]*yangEnum{
			"Container_EnumerationLeaf": {
				name: "Container_EnumerationLeaf",
				entry: &yang.Entry{
					Name: "enumeration-leaf",
					Type: &yang.YangType{
						Enum: &yang.EnumType{},
					},
				},
			},
		},
		wantUncompressed: map[string]*yangEnum{
			"BaseModule2_Container_Apple_EnumerationLeaf": {
				name: "BaseModule2_Container_Apple_EnumerationLeaf",
				entry: &yang.Entry{
					Name: "enumeration-leaf",
					Type: &yang.YangType{
						Enum: &yang.EnumType{},
					},
				},
			},
			"BaseModule2_Container_Banana_EnumerationLeaf": {
				name: "BaseModule2_Container_Banana_EnumerationLeaf",
				entry: &yang.Entry{
					Name: "enumeration-leaf",
					Type: &yang.YangType{
						Enum: &yang.EnumType{},
					},
				},
			},
		},
		wantEnumSetCompressed: &enumSet{
			uniqueEnumeratedLeafNames: map[string]string{
				"/base-module2/foo/enumeration-leaf:Container_EnumerationLeaf": "Container_EnumerationLeaf",
			},
		},
		wantEnumSetUncompressed: &enumSet{
			uniqueEnumeratedLeafNames: map[string]string{
				"/base-module2/container/apple/enumeration-leaf":  "BaseModule2_Container_Apple_EnumerationLeaf",
				"/base-module2/container/banana/enumeration-leaf": "BaseModule2_Container_Banana_EnumerationLeaf",
			},
		},
	}, {
		name: "two enums with deduplication enabled, and where duplication occurs for both compressed and decompressed",
		in: map[string]*yang.Entry{
			"/container/apple/enumeration-leaf": {
				Name: "enumeration-leaf",
				Type: &yang.YangType{
					Name: "enumeration",
					Enum: &yang.EnumType{},
				},
				Node: &yang.Enum{
					Name: "enumeration-leaf",
					Parent: &yang.Grouping{
						Name: "foo",
						Parent: &yang.Module{
							Name: "base-module2",
						},
					},
				},
				Parent: &yang.Entry{
					Name: "apple",
					Parent: &yang.Entry{
						Name:   "container",
						Parent: &yang.Entry{Name: "base-module2"},
					},
				},
			},
			"/container/banana/enumeration-leaf": {
				Name: "enumeration-leaf",
				Type: &yang.YangType{
					Name: "enumeration",
					Enum: &yang.EnumType{},
				},
				Node: &yang.Enum{
					Name: "enumeration-leaf",
					Parent: &yang.Grouping{
						Name: "foo",
						Parent: &yang.Module{
							Name: "base-module2",
						},
					},
				},
				Parent: &yang.Entry{
					Name: "banana",
					Parent: &yang.Entry{
						Name: "container",
						Parent: &yang.Entry{
							Name: "base-module2",
						},
					},
				},
			},
		},
		inShortenEnumLeafNames: true,
		wantCompressed: map[string]*yangEnum{
			"Container_EnumerationLeaf": {
				name: "Container_EnumerationLeaf",
				entry: &yang.Entry{
					Name: "enumeration-leaf",
					Type: &yang.YangType{
						Enum: &yang.EnumType{},
					},
				},
			},
		},
		wantUncompressed: map[string]*yangEnum{
			"BaseModule2_Container_Apple_EnumerationLeaf": {
				name: "BaseModule2_Container_Apple_EnumerationLeaf",
				entry: &yang.Entry{
					Name: "enumeration-leaf",
					Type: &yang.YangType{
						Enum: &yang.EnumType{},
					},
				},
			},
		},
		wantEnumSetCompressed: &enumSet{
			uniqueEnumeratedLeafNames: map[string]string{
				"/base-module2/foo/enumeration-leaf": "Container_EnumerationLeaf",
			},
		},
		wantEnumSetUncompressed: &enumSet{
			uniqueEnumeratedLeafNames: map[string]string{
				"/base-module2/foo/enumeration-leaf": "BaseModule2_Container_Apple_EnumerationLeaf",
			},
		},
	}}

	doChecks := func(t *testing.T, errs []error, wantErrSubstr string, gotEnumSet, wantEnumSet *enumSet, gotEntries, wantEntries map[string]*yangEnum) {
		t.Helper()
		if errs != nil {
			if diff := errdiff.Substring(errs[0], wantErrSubstr); diff != "" {
				t.Errorf("findEnumSet: did not get expected error when extracting enums, got: %v (len %d), wanted err: %v", errs, len(errs), wantErrSubstr)
			}
			if len(errs) > 1 {
				t.Errorf("findEnumSet: got too many errors, expecting length 1 only, (len %d), all errors: %v", len(errs), errs)
			}
			return
		}

		if diff := cmp.Diff(gotEnumSet, wantEnumSet, cmp.AllowUnexported(enumSet{}), cmpopts.EquateEmpty()); diff != "" {
			t.Errorf("enumSet (-got, +want):\n%s", diff)
		}

		// This checks just the keys of the output yangEnum map to ensure the entries match.
		if diff := cmp.Diff(gotEntries, wantEntries, cmpopts.IgnoreUnexported(yangEnum{}), cmpopts.EquateEmpty()); diff != "" {
			t.Errorf("(-got, +want):\n%s", diff)
		}

		for k, want := range wantEntries {
			got, ok := gotEntries[k]
			if !ok {
				t.Fatalf("could not find expected entry, got: %v, want: %s", gotEntries, k)
			}

			if want.entry.Name != got.entry.Name {
				t.Errorf("extracted entry has wrong name: got %s, want: %s (%+v)", got.entry.Name, want.entry.Name, got)
			}

			if want.entry.Type.IdentityBase != nil {
				// Check the identity's base if this was an identityref.
				if want.entry.Type.IdentityBase.Name != got.entry.Type.IdentityBase.Name {
					t.Errorf("found identity %s, has wrong base, got: %v, want: %v", want.entry.Name, want.entry.Type.IdentityBase.Name, got.entry.Type.IdentityBase.Name)
				}
			}
		}
	}

	for _, tt := range tests {
		hasDifferentTypedefTest := tt.wantUseDefiningModuleForTypedefEnumNames != nil
		if hasDifferentTypedefTest != (tt.wantEnumSetUseDefiningModuleForTypedefEnumNames != nil) {
			t.Fatalf("Test set-up error for %q: expected output for useDefiningModuleForTypedefEnumNames inconsistent", tt.name)
		}

		for _, compressed := range []bool{false, true} {
			wantEntries := tt.wantCompressed
			wantEnumSet := tt.wantEnumSetCompressed
			if !compressed && !tt.wantSame {
				wantEntries = tt.wantUncompressed
				wantEnumSet = tt.wantEnumSetUncompressed
			}
			for _, useDefiningModuleForTypedefEnumNames := range []bool{false, true} {
				if useDefiningModuleForTypedefEnumNames && hasDifferentTypedefTest {
					if !tt.wantSame {
						t.Fatalf("Test set-up error for %q: when testing typedef names, compression shouldn't have an effect on the output since typedef name generation is compression-independent", tt.name)
					}
					wantEntries = tt.wantUseDefiningModuleForTypedefEnumNames
					wantEnumSet = tt.wantEnumSetUseDefiningModuleForTypedefEnumNames
				}
				t.Run(fmt.Sprintf("%s findEnumSet(compress:%v,skipEnumDedup:%v,useDefiningModuleForTypedefEnumNames:%v)", tt.name, compressed, tt.inSkipEnumDeduplication, useDefiningModuleForTypedefEnumNames), func(t *testing.T) {
					gotEnumSet, gotEntries, errs := findEnumSet(tt.in, compressed, tt.inOmitUnderscores, tt.inSkipEnumDeduplication, tt.inShortenEnumLeafNames, useDefiningModuleForTypedefEnumNames)
					wantErrSubstr := tt.wantErrSubstr
					if !compressed && tt.wantUncompressFailDueToClash {
						wantErrSubstr = "clash in enumerated name occurred despite paths being uncompressed"
					}
					doChecks(t, errs, wantErrSubstr, gotEnumSet, wantEnumSet, gotEntries, wantEntries)
				})
			}
		}
	}
}
