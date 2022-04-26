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
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/openconfig/gnmi/errdiff"
	gpb "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/ygot/genutil"
	"github.com/openconfig/ygot/ygot"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestGenerateIR(t *testing.T) {
	tests := []struct {
		desc             string
		inYANGFiles      []string
		inIncludePaths   []string
		inExcludeModules []string
		inLangMapperFn   NewLangMapperFn
		inOpts           IROptions
		wantIR           *IR
		wantErrSubstring string
	}{{
		desc: "simple openconfig test with compression",
		inYANGFiles: []string{
			filepath.Join(datapath, "openconfig-simple.yang"),
			filepath.Join(datapath, "openconfig-simple-augment2.yang"),
		},
		inLangMapperFn: func() LangMapper {
			return newGoGenState(nil, nil, true)
		},
		inOpts: IROptions{
			TransformationOptions: TransformationOpts{
				CompressBehaviour:                    genutil.PreferIntendedConfig,
				ShortenEnumLeafNames:                 true,
				EnumOrgPrefixesToTrim:                []string{"openconfig"},
				UseDefiningModuleForTypedefEnumNames: true,
				EnumerationsUseUnderscores:           true,
				GenerateFakeRoot:                     true,
			},
			AppendEnumSuffixForSimpleUnionEnums: true,
		},
		wantIR: &IR{
			Directories: map[string]*ParsedDirectory{
				"/device": {
					Name: "Device",
					Type: Container,
					Fields: map[string]*NodeDetails{
						"parent": {
							Name: "Parent",
							YANGDetails: YANGNodeDetails{
								Name:         "parent",
								Defaults:     nil,
								Module:       "openconfig-simple",
								Path:         "/openconfig-simple/parent",
								ResolvedPath: "",
							},
							Type:                    ContainerNode,
							MappedPaths:             [][]string{{"parent"}},
							MappedPathModules:       [][]string{{"openconfig-simple"}},
							ShadowMappedPaths:       nil,
							ShadowMappedPathModules: nil,
						},
						"remote-container": {
							Name: "RemoteContainer",
							YANGDetails: YANGNodeDetails{
								Name:         "remote-container",
								Defaults:     nil,
								Module:       "openconfig-simple",
								Path:         "/openconfig-simple/remote-container",
								ResolvedPath: "",
							},
							Type:                    ContainerNode,
							MappedPaths:             [][]string{{"remote-container"}},
							MappedPathModules:       [][]string{{"openconfig-simple"}},
							ShadowMappedPaths:       nil,
							ShadowMappedPathModules: nil,
						},
					},
					IsFakeRoot: true,
				},
				"/openconfig-simple/parent": {
					Name: "Parent",
					Type: Container,
					Fields: map[string]*NodeDetails{
						"child": {
							Name: "Child",
							YANGDetails: YANGNodeDetails{
								Name:         "child",
								Defaults:     nil,
								Module:       "openconfig-simple",
								Path:         "/openconfig-simple/parent/child",
								ResolvedPath: "",
							},
							Type:                    ContainerNode,
							LangType:                nil,
							MappedPaths:             [][]string{{"child"}},
							MappedPathModules:       [][]string{{"openconfig-simple"}},
							ShadowMappedPaths:       nil,
							ShadowMappedPathModules: nil,
						},
					},
				},
				"/openconfig-simple/parent/child": {
					Name: "Parent_Child",
					Type: Container,
					Fields: map[string]*NodeDetails{
						"zero": {
							Name: "Zero",
							YANGDetails: YANGNodeDetails{
								Name:   "zero",
								Module: "openconfig-simple-augment2",
								Path:   "/openconfig-simple/parent/child/state/zero",
							},
							Type:                    LeafNode,
							LangType:                &MappedType{NativeType: "string", ZeroValue: `""`},
							MappedPaths:             [][]string{{"state", "zero"}},
							MappedPathModules:       [][]string{{"openconfig-simple", "openconfig-simple-augment2"}},
							ShadowMappedPaths:       nil,
							ShadowMappedPathModules: nil,
						},
						"one": {
							Name: "One",
							YANGDetails: YANGNodeDetails{
								Name:         "one",
								Defaults:     nil,
								Module:       "openconfig-simple",
								Path:         "/openconfig-simple/parent/child/config/one",
								ResolvedPath: "",
							},
							Type: LeafNode,
							LangType: &MappedType{
								NativeType:        "string",
								UnionTypes:        nil,
								IsEnumeratedValue: false,
								ZeroValue:         `""`,
								DefaultValue:      nil,
							},
							MappedPaths:             [][]string{{"config", "one"}},
							MappedPathModules:       [][]string{{"openconfig-simple", "openconfig-simple"}},
							ShadowMappedPaths:       [][]string{{"state", "one"}},
							ShadowMappedPathModules: [][]string{{"openconfig-simple", "openconfig-simple"}},
						},
						"two": {
							Name: "Two",
							YANGDetails: YANGNodeDetails{
								Name:         "two",
								Defaults:     nil,
								Module:       "openconfig-simple",
								Path:         "/openconfig-simple/parent/child/state/two",
								ResolvedPath: "",
							},
							Type: LeafNode,
							LangType: &MappedType{
								NativeType:        "string",
								UnionTypes:        nil,
								IsEnumeratedValue: false,
								ZeroValue:         `""`,
								DefaultValue:      nil,
							},
							MappedPaths:             [][]string{{"state", "two"}},
							MappedPathModules:       [][]string{{"openconfig-simple", "openconfig-simple"}},
							ShadowMappedPaths:       nil,
							ShadowMappedPathModules: nil,
						},
						"three": {
							Name: "Three",
							YANGDetails: YANGNodeDetails{
								Name:         "three",
								Defaults:     nil,
								Module:       "openconfig-simple",
								Path:         "/openconfig-simple/parent/child/config/three",
								ResolvedPath: "",
							},
							Type: LeafNode,
							LangType: &MappedType{
								NativeType:        "E_Child_Three",
								UnionTypes:        nil,
								IsEnumeratedValue: true,
								ZeroValue:         "0",
								DefaultValue:      nil,
							},
							MappedPaths:             [][]string{{"config", "three"}},
							MappedPathModules:       [][]string{{"openconfig-simple", "openconfig-simple"}},
							ShadowMappedPaths:       [][]string{{"state", "three"}},
							ShadowMappedPathModules: [][]string{{"openconfig-simple", "openconfig-simple"}},
						},
						"four": {
							Name: "Four",
							YANGDetails: YANGNodeDetails{
								Name:         "four",
								Defaults:     nil,
								Module:       "openconfig-simple",
								Path:         "/openconfig-simple/parent/child/config/four",
								ResolvedPath: "",
							},
							Type: LeafNode,
							LangType: &MappedType{
								NativeType:        "Binary",
								UnionTypes:        nil,
								IsEnumeratedValue: false,
								ZeroValue:         "nil",
								DefaultValue:      nil,
							},
							MappedPaths:             [][]string{{"config", "four"}},
							MappedPathModules:       [][]string{{"openconfig-simple", "openconfig-simple"}},
							ShadowMappedPaths:       [][]string{{"state", "four"}},
							ShadowMappedPathModules: [][]string{{"openconfig-simple", "openconfig-simple"}},
						},
					},
				},
				"/openconfig-simple/remote-container": {
					Name: "RemoteContainer",
					Type: Container,
					Fields: map[string]*NodeDetails{
						"a-leaf": {
							Name: "ALeaf",
							YANGDetails: YANGNodeDetails{
								Name:         "a-leaf",
								Defaults:     nil,
								Module:       "openconfig-simple",
								Path:         "/openconfig-simple/remote-container/config/a-leaf",
								ResolvedPath: "",
							},
							Type: LeafNode,
							LangType: &MappedType{
								NativeType:        "string",
								UnionTypes:        nil,
								IsEnumeratedValue: false,
								ZeroValue:         `""`,
								DefaultValue:      nil,
							},
							MappedPaths:             [][]string{{"config", "a-leaf"}},
							MappedPathModules:       [][]string{{"openconfig-simple", "openconfig-simple"}},
							ShadowMappedPaths:       [][]string{{"state", "a-leaf"}},
							ShadowMappedPathModules: [][]string{{"openconfig-simple", "openconfig-simple"}},
						},
					},
				},
			},
			Enums: map[string]*EnumeratedYANGType{
				"Child_Three": {
					Name:     "Child_Three",
					Kind:     SimpleEnumerationType,
					TypeName: "enumeration",
					ValToYANGDetails: []*ygot.EnumDefinition{{
						Name: "ONE",
					}, {
						Name: "TWO",
					}},
				},
			},
			ModelData: []*gpb.ModelData{{Name: "openconfig-remote"}, {Name: "openconfig-simple"}, {Name: "openconfig-simple-augment2"}},
		},
	}, {
		desc: "simple openconfig test compression prefer state no underscores",
		inYANGFiles: []string{
			filepath.Join(datapath, "openconfig-simple.yang"),
			filepath.Join(datapath, "openconfig-simple-augment2.yang"),
		},
		inLangMapperFn: func() LangMapper {
			return newGoGenState(nil, nil, true)
		},
		inOpts: IROptions{
			TransformationOptions: TransformationOpts{
				CompressBehaviour:                    genutil.PreferOperationalState,
				ShortenEnumLeafNames:                 true,
				EnumOrgPrefixesToTrim:                []string{"openconfig"},
				UseDefiningModuleForTypedefEnumNames: true,
				EnumerationsUseUnderscores:           false,
				GenerateFakeRoot:                     true,
			},
			AppendEnumSuffixForSimpleUnionEnums: true,
		},
		wantIR: &IR{
			Directories: map[string]*ParsedDirectory{
				"/device": {
					Name: "Device",
					Type: Container,
					Fields: map[string]*NodeDetails{
						"parent": {
							Name: "Parent",
							YANGDetails: YANGNodeDetails{
								Name:         "parent",
								Defaults:     nil,
								Module:       "openconfig-simple",
								Path:         "/openconfig-simple/parent",
								ResolvedPath: "",
							},
							Type:                    ContainerNode,
							MappedPaths:             [][]string{{"parent"}},
							MappedPathModules:       [][]string{{"openconfig-simple"}},
							ShadowMappedPaths:       nil,
							ShadowMappedPathModules: nil,
						},
						"remote-container": {
							Name: "RemoteContainer",
							YANGDetails: YANGNodeDetails{
								Name:         "remote-container",
								Defaults:     nil,
								Module:       "openconfig-simple",
								Path:         "/openconfig-simple/remote-container",
								ResolvedPath: "",
							},
							Type:                    ContainerNode,
							MappedPaths:             [][]string{{"remote-container"}},
							MappedPathModules:       [][]string{{"openconfig-simple"}},
							ShadowMappedPaths:       nil,
							ShadowMappedPathModules: nil,
						},
					},
					IsFakeRoot: true,
				},
				"/openconfig-simple/parent": {
					Name: "Parent",
					Type: Container,
					Fields: map[string]*NodeDetails{
						"child": {
							Name: "Child",
							YANGDetails: YANGNodeDetails{
								Name:         "child",
								Defaults:     nil,
								Module:       "openconfig-simple",
								Path:         "/openconfig-simple/parent/child",
								ResolvedPath: "",
							},
							Type:                    ContainerNode,
							LangType:                nil,
							MappedPaths:             [][]string{{"child"}},
							MappedPathModules:       [][]string{{"openconfig-simple"}},
							ShadowMappedPaths:       nil,
							ShadowMappedPathModules: nil,
						},
					},
				},
				"/openconfig-simple/parent/child": {
					Name: "Parent_Child",
					Type: Container,
					Fields: map[string]*NodeDetails{
						"zero": {
							Name: "Zero",
							YANGDetails: YANGNodeDetails{
								Name:   "zero",
								Module: "openconfig-simple-augment2",
								Path:   "/openconfig-simple/parent/child/state/zero",
							},
							Type:                    LeafNode,
							LangType:                &MappedType{NativeType: "string", ZeroValue: `""`},
							MappedPaths:             [][]string{{"state", "zero"}},
							MappedPathModules:       [][]string{{"openconfig-simple", "openconfig-simple-augment2"}},
							ShadowMappedPaths:       nil,
							ShadowMappedPathModules: nil,
						},
						"one": {
							Name: "One",
							YANGDetails: YANGNodeDetails{
								Name:         "one",
								Defaults:     nil,
								Module:       "openconfig-simple",
								Path:         "/openconfig-simple/parent/child/state/one",
								ResolvedPath: "",
							},
							Type: LeafNode,
							LangType: &MappedType{
								NativeType:        "string",
								UnionTypes:        nil,
								IsEnumeratedValue: false,
								ZeroValue:         `""`,
								DefaultValue:      nil,
							},
							MappedPaths:             [][]string{{"state", "one"}},
							MappedPathModules:       [][]string{{"openconfig-simple", "openconfig-simple"}},
							ShadowMappedPaths:       [][]string{{"config", "one"}},
							ShadowMappedPathModules: [][]string{{"openconfig-simple", "openconfig-simple"}},
						},
						"two": {
							Name: "Two",
							YANGDetails: YANGNodeDetails{
								Name:         "two",
								Defaults:     nil,
								Module:       "openconfig-simple",
								Path:         "/openconfig-simple/parent/child/state/two",
								ResolvedPath: "",
							},
							Type: LeafNode,
							LangType: &MappedType{
								NativeType:        "string",
								UnionTypes:        nil,
								IsEnumeratedValue: false,
								ZeroValue:         `""`,
								DefaultValue:      nil,
							},
							MappedPaths:             [][]string{{"state", "two"}},
							MappedPathModules:       [][]string{{"openconfig-simple", "openconfig-simple"}},
							ShadowMappedPaths:       nil,
							ShadowMappedPathModules: nil,
						},
						"three": {
							Name: "Three",
							YANGDetails: YANGNodeDetails{
								Name:         "three",
								Defaults:     nil,
								Module:       "openconfig-simple",
								Path:         "/openconfig-simple/parent/child/state/three",
								ResolvedPath: "",
							},
							Type: LeafNode,
							LangType: &MappedType{
								NativeType:        "E_ChildThree",
								UnionTypes:        nil,
								IsEnumeratedValue: true,
								ZeroValue:         "0",
								DefaultValue:      nil,
							},
							MappedPaths:             [][]string{{"state", "three"}},
							MappedPathModules:       [][]string{{"openconfig-simple", "openconfig-simple"}},
							ShadowMappedPaths:       [][]string{{"config", "three"}},
							ShadowMappedPathModules: [][]string{{"openconfig-simple", "openconfig-simple"}},
						},
						"four": {
							Name: "Four",
							YANGDetails: YANGNodeDetails{
								Name:         "four",
								Defaults:     nil,
								Module:       "openconfig-simple",
								Path:         "/openconfig-simple/parent/child/state/four",
								ResolvedPath: "",
							},
							Type: LeafNode,
							LangType: &MappedType{
								NativeType:        "Binary",
								UnionTypes:        nil,
								IsEnumeratedValue: false,
								ZeroValue:         "nil",
								DefaultValue:      nil,
							},
							MappedPaths:             [][]string{{"state", "four"}},
							MappedPathModules:       [][]string{{"openconfig-simple", "openconfig-simple"}},
							ShadowMappedPaths:       [][]string{{"config", "four"}},
							ShadowMappedPathModules: [][]string{{"openconfig-simple", "openconfig-simple"}},
						},
					},
				},
				"/openconfig-simple/remote-container": {
					Name: "RemoteContainer",
					Type: Container,
					Fields: map[string]*NodeDetails{
						"a-leaf": {
							Name: "ALeaf",
							YANGDetails: YANGNodeDetails{
								Name:         "a-leaf",
								Defaults:     nil,
								Module:       "openconfig-simple",
								Path:         "/openconfig-simple/remote-container/state/a-leaf",
								ResolvedPath: "",
							},
							Type: LeafNode,
							LangType: &MappedType{
								NativeType:        "string",
								UnionTypes:        nil,
								IsEnumeratedValue: false,
								ZeroValue:         `""`,
								DefaultValue:      nil,
							},
							MappedPaths:             [][]string{{"state", "a-leaf"}},
							MappedPathModules:       [][]string{{"openconfig-simple", "openconfig-simple"}},
							ShadowMappedPaths:       [][]string{{"config", "a-leaf"}},
							ShadowMappedPathModules: [][]string{{"openconfig-simple", "openconfig-simple"}},
						},
					},
				},
			},
			Enums: map[string]*EnumeratedYANGType{
				"ChildThree": {
					Name:     "ChildThree",
					Kind:     SimpleEnumerationType,
					TypeName: "enumeration",
					ValToYANGDetails: []*ygot.EnumDefinition{{
						Name: "ONE",
					}, {
						Name: "TWO",
					}},
				},
			},
			ModelData: []*gpb.ModelData{{Name: "openconfig-remote"}, {Name: "openconfig-simple"}, {Name: "openconfig-simple-augment2"}},
		},
	}, {
		desc: "simple openconfig test without compression",
		inYANGFiles: []string{
			filepath.Join(datapath, "openconfig-simple.yang"),
			filepath.Join(datapath, "openconfig-simple-augment2.yang"),
		},
		inLangMapperFn: func() LangMapper {
			return newGoGenState(nil, nil, true)
		},
		inOpts: IROptions{
			TransformationOptions: TransformationOpts{
				CompressBehaviour:                    genutil.Uncompressed,
				ShortenEnumLeafNames:                 true,
				EnumOrgPrefixesToTrim:                []string{"openconfig"},
				UseDefiningModuleForTypedefEnumNames: true,
				EnumerationsUseUnderscores:           true,
				GenerateFakeRoot:                     true,
			},
			AppendEnumSuffixForSimpleUnionEnums: true,
		},
		wantIR: &IR{
			Directories: map[string]*ParsedDirectory{
				"/device": {
					Name: "Device",
					Type: Container,
					Fields: map[string]*NodeDetails{
						"parent": {
							Name: "Parent",
							YANGDetails: YANGNodeDetails{
								Name:         "parent",
								Defaults:     nil,
								Module:       "openconfig-simple",
								Path:         "/openconfig-simple/parent",
								ResolvedPath: "",
							},
							Type:                    ContainerNode,
							MappedPaths:             [][]string{{"parent"}},
							MappedPathModules:       [][]string{{"openconfig-simple"}},
							ShadowMappedPaths:       nil,
							ShadowMappedPathModules: nil,
						},
						"remote-container": {
							Name: "RemoteContainer",
							YANGDetails: YANGNodeDetails{
								Name:         "remote-container",
								Defaults:     nil,
								Module:       "openconfig-simple",
								Path:         "/openconfig-simple/remote-container",
								ResolvedPath: "",
							},
							Type:                    ContainerNode,
							MappedPaths:             [][]string{{"remote-container"}},
							MappedPathModules:       [][]string{{"openconfig-simple"}},
							ShadowMappedPaths:       nil,
							ShadowMappedPathModules: nil,
						},
					},
					IsFakeRoot: true,
				},
				"/openconfig-simple/parent": {
					Name: "OpenconfigSimple_Parent",
					Type: Container,
					Fields: map[string]*NodeDetails{
						"child": {
							Name: "Child",
							YANGDetails: YANGNodeDetails{
								Name:         "child",
								Defaults:     nil,
								Module:       "openconfig-simple",
								Path:         "/openconfig-simple/parent/child",
								ResolvedPath: "",
							},
							Type:                    ContainerNode,
							LangType:                nil,
							MappedPaths:             [][]string{{"child"}},
							MappedPathModules:       [][]string{{"openconfig-simple"}},
							ShadowMappedPaths:       nil,
							ShadowMappedPathModules: nil,
						},
					},
				},
				"/openconfig-simple/parent/child": {
					Name: "OpenconfigSimple_Parent_Child",
					Type: 1,
					Fields: map[string]*NodeDetails{
						"config": {
							Name: "Config",
							YANGDetails: YANGNodeDetails{
								Name:   "config",
								Module: "openconfig-simple",
								Path:   "/openconfig-simple/parent/child/config",
							},
							Type:              1,
							MappedPaths:       [][]string{{"config"}},
							MappedPathModules: [][]string{{"openconfig-simple"}},
						},
						"state": {
							Name: "State",
							YANGDetails: YANGNodeDetails{
								Name:   "state",
								Module: "openconfig-simple",
								Path:   "/openconfig-simple/parent/child/state",
							},
							Type:              1,
							MappedPaths:       [][]string{{"state"}},
							MappedPathModules: [][]string{{"openconfig-simple"}},
						},
					},
					ListKeys:    nil,
					PackageName: "",
				},
				"/openconfig-simple/parent/child/config": {
					Name: "OpenconfigSimple_Parent_Child_Config",
					Type: 1,
					Fields: map[string]*NodeDetails{
						"four": {
							Name: "Four",
							YANGDetails: YANGNodeDetails{
								Name:         "four",
								Defaults:     nil,
								Module:       "openconfig-simple",
								Path:         "/openconfig-simple/parent/child/config/four",
								ResolvedPath: "",
							},
							Type: 3,
							LangType: &MappedType{
								NativeType:        "Binary",
								UnionTypes:        nil,
								IsEnumeratedValue: false,
								ZeroValue:         "nil",
								DefaultValue:      nil,
							},
							MappedPaths:             [][]string{{"four"}},
							MappedPathModules:       [][]string{{"openconfig-simple"}},
							ShadowMappedPaths:       nil,
							ShadowMappedPathModules: nil,
						},
						"one": {
							Name: "One",
							YANGDetails: YANGNodeDetails{
								Name:         "one",
								Defaults:     nil,
								Module:       "openconfig-simple",
								Path:         "/openconfig-simple/parent/child/config/one",
								ResolvedPath: "",
							},
							Type: 3,
							LangType: &MappedType{
								NativeType:        "string",
								UnionTypes:        nil,
								IsEnumeratedValue: false,
								ZeroValue:         `""`,
								DefaultValue:      nil,
							},
							MappedPaths:             [][]string{{"one"}},
							MappedPathModules:       [][]string{{"openconfig-simple"}},
							ShadowMappedPaths:       nil,
							ShadowMappedPathModules: nil,
						},
						"three": {
							Name: "Three",
							YANGDetails: YANGNodeDetails{
								Name:         "three",
								Defaults:     nil,
								Module:       "openconfig-simple",
								Path:         "/openconfig-simple/parent/child/config/three",
								ResolvedPath: "",
							},
							Type: 3,
							LangType: &MappedType{
								NativeType:        "E_Simple_Parent_Child_Config_Three",
								UnionTypes:        nil,
								IsEnumeratedValue: true,
								ZeroValue:         "0",
								DefaultValue:      nil,
							},
							MappedPaths:             [][]string{{"three"}},
							MappedPathModules:       [][]string{{"openconfig-simple"}},
							ShadowMappedPaths:       nil,
							ShadowMappedPathModules: nil,
						},
					},
					ListKeys:    nil,
					PackageName: "",
				},
				"/openconfig-simple/parent/child/state": {
					Name: "OpenconfigSimple_Parent_Child_State",
					Type: 1,
					Fields: map[string]*NodeDetails{
						"four": {
							Name: "Four",
							YANGDetails: YANGNodeDetails{
								Name:         "four",
								Defaults:     nil,
								Module:       "openconfig-simple",
								Path:         "/openconfig-simple/parent/child/state/four",
								ResolvedPath: "",
							},
							Type: 3,
							LangType: &MappedType{
								NativeType:        "Binary",
								UnionTypes:        nil,
								IsEnumeratedValue: false,
								ZeroValue:         "nil",
								DefaultValue:      nil,
							},
							MappedPaths:             [][]string{{"four"}},
							MappedPathModules:       [][]string{{"openconfig-simple"}},
							ShadowMappedPaths:       nil,
							ShadowMappedPathModules: nil,
						},
						"one": {
							Name: "One",
							YANGDetails: YANGNodeDetails{
								Name:         "one",
								Defaults:     nil,
								Module:       "openconfig-simple",
								Path:         "/openconfig-simple/parent/child/state/one",
								ResolvedPath: "",
							},
							Type: 3,
							LangType: &MappedType{
								NativeType:        "string",
								UnionTypes:        nil,
								IsEnumeratedValue: false,
								ZeroValue:         `""`,
								DefaultValue:      nil,
							},
							MappedPaths:             [][]string{{"one"}},
							MappedPathModules:       [][]string{{"openconfig-simple"}},
							ShadowMappedPaths:       nil,
							ShadowMappedPathModules: nil,
						},
						"three": {
							Name: "Three",
							YANGDetails: YANGNodeDetails{
								Name:         "three",
								Defaults:     nil,
								Module:       "openconfig-simple",
								Path:         "/openconfig-simple/parent/child/state/three",
								ResolvedPath: "",
							},
							Type: 3,
							LangType: &MappedType{
								NativeType:        "E_Simple_Parent_Child_Config_Three",
								UnionTypes:        nil,
								IsEnumeratedValue: true,
								ZeroValue:         "0",
								DefaultValue:      nil,
							},
							MappedPaths:             [][]string{{"three"}},
							MappedPathModules:       [][]string{{"openconfig-simple"}},
							ShadowMappedPaths:       nil,
							ShadowMappedPathModules: nil,
						},
						"two": {
							Name: "Two",
							YANGDetails: YANGNodeDetails{
								Name:         "two",
								Defaults:     nil,
								Module:       "openconfig-simple",
								Path:         "/openconfig-simple/parent/child/state/two",
								ResolvedPath: "",
							},
							Type: 3,
							LangType: &MappedType{
								NativeType:        "string",
								UnionTypes:        nil,
								IsEnumeratedValue: false,
								ZeroValue:         `""`,
								DefaultValue:      nil,
							},
							MappedPaths:             [][]string{{"two"}},
							MappedPathModules:       [][]string{{"openconfig-simple"}},
							ShadowMappedPaths:       nil,
							ShadowMappedPathModules: nil,
						},
						"zero": {
							Name: "Zero",
							YANGDetails: YANGNodeDetails{
								Name:         "zero",
								Defaults:     nil,
								Module:       "openconfig-simple-augment2",
								Path:         "/openconfig-simple/parent/child/state/zero",
								ResolvedPath: "",
							},
							Type: 3,
							LangType: &MappedType{
								NativeType:        "string",
								UnionTypes:        nil,
								IsEnumeratedValue: false,
								ZeroValue:         `""`,
								DefaultValue:      nil,
							},
							MappedPaths:             [][]string{{"zero"}},
							MappedPathModules:       [][]string{{"openconfig-simple-augment2"}},
							ShadowMappedPaths:       nil,
							ShadowMappedPathModules: nil,
						},
					},
					ListKeys:    nil,
					PackageName: "",
				},
				"/openconfig-simple/remote-container": {
					Name: "OpenconfigSimple_RemoteContainer",
					Type: Container,
					Fields: map[string]*NodeDetails{
						"config": {
							Name: "Config",
							YANGDetails: YANGNodeDetails{
								Name:   "config",
								Module: "openconfig-simple",
								Path:   "/openconfig-simple/remote-container/config",
							},
							Type:              1,
							MappedPaths:       [][]string{{"config"}},
							MappedPathModules: [][]string{{"openconfig-simple"}},
						},
						"state": {
							Name: "State",
							YANGDetails: YANGNodeDetails{
								Name:   "state",
								Module: "openconfig-simple",
								Path:   "/openconfig-simple/remote-container/state",
							},
							Type:              1,
							MappedPaths:       [][]string{{"state"}},
							MappedPathModules: [][]string{{"openconfig-simple"}},
						},
					},
					ListKeys:    nil,
					PackageName: "",
				},
				"/openconfig-simple/remote-container/config": {
					Name: "OpenconfigSimple_RemoteContainer_Config",
					Type: 1,
					Fields: map[string]*NodeDetails{
						"a-leaf": {
							Name: "ALeaf",
							YANGDetails: YANGNodeDetails{
								Name:         "a-leaf",
								Defaults:     nil,
								Module:       "openconfig-simple",
								Path:         "/openconfig-simple/remote-container/config/a-leaf",
								ResolvedPath: "",
							},
							Type: 3,
							LangType: &MappedType{
								NativeType:        "string",
								UnionTypes:        nil,
								IsEnumeratedValue: false,
								ZeroValue:         `""`,
								DefaultValue:      nil,
							},
							MappedPaths:             [][]string{{"a-leaf"}},
							MappedPathModules:       [][]string{{"openconfig-simple"}},
							ShadowMappedPaths:       nil,
							ShadowMappedPathModules: nil,
						},
					},
				},
				"/openconfig-simple/remote-container/state": {
					Name: "OpenconfigSimple_RemoteContainer_State",
					Type: 1,
					Fields: map[string]*NodeDetails{
						"a-leaf": {
							Name: "ALeaf",
							YANGDetails: YANGNodeDetails{
								Name:         "a-leaf",
								Defaults:     nil,
								Module:       "openconfig-simple",
								Path:         "/openconfig-simple/remote-container/state/a-leaf",
								ResolvedPath: "",
							},
							Type: 3,
							LangType: &MappedType{
								NativeType:        "string",
								UnionTypes:        nil,
								IsEnumeratedValue: false,
								ZeroValue:         `""`,
								DefaultValue:      nil,
							},
							MappedPaths:             [][]string{{"a-leaf"}},
							MappedPathModules:       [][]string{{"openconfig-simple"}},
							ShadowMappedPaths:       nil,
							ShadowMappedPathModules: nil,
						},
					},
				},
			},
			Enums: map[string]*EnumeratedYANGType{
				"Simple_Parent_Child_Config_Three": {
					Name:             "Simple_Parent_Child_Config_Three",
					Kind:             1,
					TypeName:         "enumeration",
					ValToYANGDetails: []*ygot.EnumDefinition{{Name: "ONE"}, {Name: "TWO"}},
				},
			},
			ModelData: []*gpb.ModelData{{Name: "openconfig-remote"}, {Name: "openconfig-simple"}, {Name: "openconfig-simple-augment2"}},
		},
	}, {
		desc:             "exclude module test with compression",
		inYANGFiles:      []string{filepath.Join(datapath, "excluded-module-noimport.yang")},
		inExcludeModules: []string{"excluded-module-two"},
		inLangMapperFn: func() LangMapper {
			return newGoGenState(nil, nil, true)
		},
		inOpts: IROptions{
			TransformationOptions: TransformationOpts{
				CompressBehaviour:                    genutil.PreferIntendedConfig,
				ShortenEnumLeafNames:                 true,
				UseDefiningModuleForTypedefEnumNames: true,
				EnumerationsUseUnderscores:           true,
				GenerateFakeRoot:                     true,
			},
			AppendEnumSuffixForSimpleUnionEnums: true,
		},
		wantIR: &IR{
			Directories: map[string]*ParsedDirectory{
				"/device": {
					Name: "Device",
					Type: Container,
					Fields: map[string]*NodeDetails{
						"e1": {
							Name: "E1",
							YANGDetails: YANGNodeDetails{
								Name:         "e1",
								Defaults:     nil,
								Module:       "excluded-module-noimport",
								Path:         "/excluded-module-noimport/e1",
								ResolvedPath: "",
							},
							Type: LeafNode,
							LangType: &MappedType{
								NativeType:        "string",
								UnionTypes:        nil,
								IsEnumeratedValue: false,
								ZeroValue:         `""`,
								DefaultValue:      nil,
							},
							MappedPaths:             [][]string{{"e1"}},
							MappedPathModules:       [][]string{{"excluded-module-noimport"}},
							ShadowMappedPaths:       nil,
							ShadowMappedPathModules: nil,
						},
					},
					IsFakeRoot: true,
				},
			},
			// TODO(wenbli): Determine whether "excluded-module-two" should be here.
			ModelData: []*gpb.ModelData{{Name: "excluded-module-noimport"}, {Name: "excluded-module-two"}},
		},
	}, {
		desc:        "complex openconfig test with compression",
		inYANGFiles: []string{filepath.Join(datapath, "openconfig-complex.yang")},
		inLangMapperFn: func() LangMapper {
			return newGoGenState(nil, nil, true)
		},
		inOpts: IROptions{
			TransformationOptions: TransformationOpts{
				CompressBehaviour:                    genutil.PreferIntendedConfig,
				ShortenEnumLeafNames:                 true,
				EnumOrgPrefixesToTrim:                []string{"openconfig"},
				UseDefiningModuleForTypedefEnumNames: true,
				EnumerationsUseUnderscores:           true,
				GenerateFakeRoot:                     true,
			},
			AppendEnumSuffixForSimpleUnionEnums: true,
		},
		wantIR: &IR{
			Directories: map[string]*ParsedDirectory{
				"/device": {
					Name: "Device",
					Type: Container,
					Fields: map[string]*NodeDetails{
						"model": {
							Name: "Model",
							YANGDetails: YANGNodeDetails{
								Name:         "model",
								Defaults:     nil,
								Module:       "openconfig-complex",
								Path:         "/openconfig-complex/model",
								ResolvedPath: "",
							},
							Type:                    ContainerNode,
							LangType:                nil,
							MappedPaths:             [][]string{{"model"}},
							MappedPathModules:       [][]string{{"openconfig-complex"}},
							ShadowMappedPaths:       nil,
							ShadowMappedPathModules: nil,
						},
						"example-presence": {
							Name: "ExamplePresence",
							YANGDetails: YANGNodeDetails{
								Name:              "example-presence",
								Defaults:          nil,
								Module:            "openconfig-complex",
								Path:              "/openconfig-complex/example-presence",
								ResolvedPath:      "",
								PresenceStatement: ygot.String("This is an example presence container"),
							},
							Type:                    ContainerNode,
							LangType:                nil,
							MappedPaths:             [][]string{{"example-presence"}},
							MappedPathModules:       [][]string{{"openconfig-complex"}},
							ShadowMappedPaths:       nil,
							ShadowMappedPathModules: nil,
						},
					},
					IsFakeRoot: true,
				},
				"/openconfig-complex/example-presence": {
					Name:        "ExamplePresence",
					Type:        Container,
					Fields:      map[string]*NodeDetails{},
					PackageName: "",
					ListKeys:    nil,
					IsFakeRoot:  false,
				},
				"/openconfig-complex/model": {
					Name: "Model",
					Type: Container,
					Fields: map[string]*NodeDetails{
						"anydata-leaf": {
							Name: "AnydataLeaf",
							YANGDetails: YANGNodeDetails{
								Name:         "anydata-leaf",
								Defaults:     nil,
								Module:       "openconfig-complex",
								Path:         "/openconfig-complex/model/anydata-leaf",
								ResolvedPath: "",
							},
							Type:                    AnyDataNode,
							LangType:                nil,
							MappedPaths:             [][]string{{"anydata-leaf"}},
							MappedPathModules:       [][]string{{"openconfig-complex"}},
							ShadowMappedPaths:       nil,
							ShadowMappedPathModules: nil,
						},
						"dateref": {
							Name: "Dateref",
							YANGDetails: YANGNodeDetails{
								Name:         "dateref",
								Defaults:     nil,
								Module:       "openconfig-complex",
								Path:         "/openconfig-complex/model/dateref",
								ResolvedPath: "/openconfig-complex/model/a/single-key/config/dates",
							},
							Type: LeafNode,
							LangType: &MappedType{
								NativeType:   "uint8",
								ZeroValue:    "0",
								DefaultValue: ygot.String("5"),
							},
							MappedPaths:             [][]string{{"dateref"}},
							MappedPathModules:       [][]string{{"openconfig-complex"}},
							ShadowMappedPaths:       nil,
							ShadowMappedPathModules: nil,
						},
						"multi-key": {
							Name: "MultiKey",
							YANGDetails: YANGNodeDetails{
								Name:         "multi-key",
								Defaults:     nil,
								Module:       "openconfig-complex",
								Path:         "/openconfig-complex/model/b/multi-key",
								ResolvedPath: "",
							},
							Type:                    ListNode,
							LangType:                nil,
							MappedPaths:             [][]string{{"b", "multi-key"}},
							MappedPathModules:       [][]string{{"openconfig-complex", "openconfig-complex"}},
							ShadowMappedPaths:       nil,
							ShadowMappedPathModules: nil,
						},
						"single-key": {
							Name: "SingleKey",
							YANGDetails: YANGNodeDetails{
								Name:         "single-key",
								Defaults:     nil,
								Module:       "openconfig-complex",
								Path:         "/openconfig-complex/model/a/single-key",
								ResolvedPath: "",
							},
							Type:                    ListNode,
							LangType:                nil,
							MappedPaths:             [][]string{{"a", "single-key"}},
							MappedPathModules:       [][]string{{"openconfig-complex", "openconfig-complex"}},
							ShadowMappedPaths:       nil,
							ShadowMappedPathModules: nil,
						},
					},
					ListKeys:    nil,
					PackageName: "",
				},
				"/openconfig-complex/model/a/single-key": {
					Name: "Model_SingleKey",
					Type: List,
					Fields: map[string]*NodeDetails{
						"dates": {
							Name: "Dates",
							YANGDetails: YANGNodeDetails{
								Name:         "dates",
								Defaults:     []string{"5"},
								Module:       "openconfig-complex",
								Path:         "/openconfig-complex/model/a/single-key/config/dates",
								ResolvedPath: "",
							},
							Type:                    LeafListNode,
							LangType:                &MappedType{NativeType: "uint8", ZeroValue: "0", DefaultValue: ygot.String("5")},
							MappedPaths:             [][]string{{"config", "dates"}},
							MappedPathModules:       [][]string{{"openconfig-complex", "openconfig-complex"}},
							ShadowMappedPaths:       [][]string{{"state", "dates"}},
							ShadowMappedPathModules: [][]string{{"openconfig-complex", "openconfig-complex"}},
						},
						"dates-with-defaults": {
							Name: "DatesWithDefaults",
							YANGDetails: YANGNodeDetails{
								Name:         "dates-with-defaults",
								Defaults:     []string{"1", "2"},
								Module:       "openconfig-complex",
								Path:         "/openconfig-complex/model/a/single-key/config/dates-with-defaults",
								ResolvedPath: "",
							},
							Type:                    LeafListNode,
							LangType:                &MappedType{NativeType: "uint8", ZeroValue: "0", DefaultValue: ygot.String("5")},
							MappedPaths:             [][]string{{"config", "dates-with-defaults"}},
							MappedPathModules:       [][]string{{"openconfig-complex", "openconfig-complex"}},
							ShadowMappedPaths:       [][]string{{"state", "dates-with-defaults"}},
							ShadowMappedPathModules: [][]string{{"openconfig-complex", "openconfig-complex"}},
						},
						"iref": {
							Name: "Iref",
							YANGDetails: YANGNodeDetails{
								Name:         "iref",
								Defaults:     nil,
								Module:       "openconfig-complex",
								Path:         "/openconfig-complex/model/a/single-key/config/iref",
								ResolvedPath: "",
							},
							Type:                    LeafNode,
							LangType:                &MappedType{NativeType: "E_Complex_SOFTWARE", IsEnumeratedValue: true, ZeroValue: "0"},
							MappedPaths:             [][]string{{"config", "iref"}},
							MappedPathModules:       [][]string{{"openconfig-complex", "openconfig-complex"}},
							ShadowMappedPaths:       [][]string{{"state", "iref"}},
							ShadowMappedPathModules: [][]string{{"openconfig-complex", "openconfig-complex"}},
						},
						"key": {
							Name: "Key",
							YANGDetails: YANGNodeDetails{
								Name:         "key",
								Defaults:     nil,
								Module:       "openconfig-complex",
								Path:         "/openconfig-complex/model/a/single-key/config/key",
								ResolvedPath: "",
							},
							Type: LeafNode,
							LangType: &MappedType{
								NativeType: "Model_SingleKey_Key_Union",
								UnionTypes: map[string]int{"E_Complex_WeekendDays": 1, "uint8": 0},
								ZeroValue:  "nil",
							},
							MappedPaths:             [][]string{{"config", "key"}, {"key"}},
							MappedPathModules:       [][]string{{"openconfig-complex", "openconfig-complex"}, {"openconfig-complex"}},
							ShadowMappedPaths:       [][]string{{"state", "key"}, {"key"}},
							ShadowMappedPathModules: [][]string{{"openconfig-complex", "openconfig-complex"}, {"openconfig-complex"}},
						},
						"leaf-default-override": {
							Name: "LeafDefaultOverride",
							YANGDetails: YANGNodeDetails{
								Name:     "leaf-default-override",
								Defaults: []string{"3"},
								Module:   "openconfig-complex",
								Path:     "/openconfig-complex/model/a/single-key/config/leaf-default-override",
							},
							Type: LeafNode,
							LangType: &MappedType{
								NativeType:   "Model_SingleKey_LeafDefaultOverride_Union",
								UnionTypes:   map[string]int{"E_Complex_CycloneScales_Enum": 1, "uint8": 0},
								ZeroValue:    "nil",
								DefaultValue: ygot.String("SUPER"),
							},
							MappedPaths:             [][]string{{"config", "leaf-default-override"}},
							MappedPathModules:       [][]string{{"openconfig-complex", "openconfig-complex"}},
							ShadowMappedPaths:       [][]string{{"state", "leaf-default-override"}},
							ShadowMappedPathModules: [][]string{{"openconfig-complex", "openconfig-complex"}},
						},
						"simple-union-enum": {
							Name: "SimpleUnionEnum",
							YANGDetails: YANGNodeDetails{
								Name:     "simple-union-enum",
								Defaults: []string{"TWO"},
								Module:   "openconfig-complex",
								Path:     "/openconfig-complex/model/a/single-key/config/simple-union-enum",
							},
							Type: LeafNode,
							LangType: &MappedType{
								NativeType: "Model_SingleKey_SimpleUnionEnum_Union",
								UnionTypes: map[string]int{"E_SingleKey_SimpleUnionEnum_Enum": 1, "uint64": 0},
								ZeroValue:  "nil",
							},
							MappedPaths:             [][]string{{"config", "simple-union-enum"}},
							MappedPathModules:       [][]string{{"openconfig-complex", "openconfig-complex"}},
							ShadowMappedPaths:       [][]string{{"state", "simple-union-enum"}},
							ShadowMappedPathModules: [][]string{{"openconfig-complex", "openconfig-complex"}},
						},
						"singleton-union-enum": {
							Name: "SingletonUnionEnum",
							YANGDetails: YANGNodeDetails{
								Name:     "singleton-union-enum",
								Defaults: []string{"DEUX"},
								Module:   "openconfig-complex",
								Path:     "/openconfig-complex/model/a/single-key/config/singleton-union-enum",
							},
							Type: LeafNode,
							LangType: &MappedType{
								NativeType:        "E_SingleKey_SingletonUnionEnum_Enum",
								UnionTypes:        map[string]int{"E_SingleKey_SingletonUnionEnum_Enum": 0},
								IsEnumeratedValue: true,
								ZeroValue:         "0",
							},
							MappedPaths:             [][]string{{"config", "singleton-union-enum"}},
							MappedPathModules:       [][]string{{"openconfig-complex", "openconfig-complex"}},
							ShadowMappedPaths:       [][]string{{"state", "singleton-union-enum"}},
							ShadowMappedPathModules: [][]string{{"openconfig-complex", "openconfig-complex"}},
						},
						"typedef-enum": {
							Name: "TypedefEnum",
							YANGDetails: YANGNodeDetails{
								Name:     "typedef-enum",
								Defaults: []string{"SATURDAY"},
								Module:   "openconfig-complex",
								Path:     "/openconfig-complex/model/a/single-key/config/typedef-enum",
							},
							Type: LeafNode,
							LangType: &MappedType{
								NativeType:        "E_Complex_WeekendDays",
								IsEnumeratedValue: true,
								ZeroValue:         "0",
								DefaultValue:      ygot.String("SUNDAY"),
							},
							MappedPaths:             [][]string{{"config", "typedef-enum"}},
							MappedPathModules:       [][]string{{"openconfig-complex", "openconfig-complex"}},
							ShadowMappedPaths:       [][]string{{"state", "typedef-enum"}},
							ShadowMappedPathModules: [][]string{{"openconfig-complex", "openconfig-complex"}},
						},
						"typedef-union-enum": {
							Name: "TypedefUnionEnum",
							YANGDetails: YANGNodeDetails{
								Name:     "typedef-union-enum",
								Defaults: []string{"SUPER"},
								Module:   "openconfig-complex",
								Path:     "/openconfig-complex/model/a/single-key/config/typedef-union-enum",
							},
							Type: LeafNode,
							LangType: &MappedType{
								NativeType:   "Model_SingleKey_TypedefUnionEnum_Union",
								UnionTypes:   map[string]int{"E_Complex_CycloneScales_Enum": 1, "uint8": 0},
								ZeroValue:    "nil",
								DefaultValue: ygot.String("SUPER"),
							},
							MappedPaths:             [][]string{{"config", "typedef-union-enum"}},
							MappedPathModules:       [][]string{{"openconfig-complex", "openconfig-complex"}},
							ShadowMappedPaths:       [][]string{{"state", "typedef-union-enum"}},
							ShadowMappedPathModules: [][]string{{"openconfig-complex", "openconfig-complex"}},
						},
					},
					ListKeys: map[string]*ListKey{
						"key": {
							Name: "Key",
							LangType: &MappedType{
								NativeType: "Model_SingleKey_Key_Union",
								UnionTypes: map[string]int{"E_Complex_WeekendDays": 1, "uint8": 0},
								ZeroValue:  "nil",
							},
						},
					},
					PackageName: "",
					IsFakeRoot:  false,
				},
				"/openconfig-complex/model/b/multi-key": {
					Name: "Model_MultiKey",
					Type: List,
					Fields: map[string]*NodeDetails{
						"key1": {
							Name: "Key1",
							YANGDetails: YANGNodeDetails{
								Name:         "key1",
								Defaults:     nil,
								Module:       "openconfig-complex",
								Path:         "/openconfig-complex/model/b/multi-key/config/key1",
								ResolvedPath: "",
							},
							Type:                    LeafNode,
							LangType:                &MappedType{NativeType: "uint32", ZeroValue: "0"},
							MappedPaths:             [][]string{{"config", "key1"}, {"key1"}},
							MappedPathModules:       [][]string{{"openconfig-complex", "openconfig-complex"}, {"openconfig-complex"}},
							ShadowMappedPaths:       [][]string{{"state", "key1"}, {"key1"}},
							ShadowMappedPathModules: [][]string{{"openconfig-complex", "openconfig-complex"}, {"openconfig-complex"}},
						},
						"key2": {
							Name: "Key2",
							YANGDetails: YANGNodeDetails{
								Name:         "key2",
								Defaults:     nil,
								Module:       "openconfig-complex",
								Path:         "/openconfig-complex/model/b/multi-key/config/key2",
								ResolvedPath: "",
							},
							Type:                    LeafNode,
							LangType:                &MappedType{NativeType: "uint64", ZeroValue: "0"},
							MappedPaths:             [][]string{{"config", "key2"}, {"key2"}},
							MappedPathModules:       [][]string{{"openconfig-complex", "openconfig-complex"}, {"openconfig-complex"}},
							ShadowMappedPaths:       [][]string{{"state", "key2"}, {"key2"}},
							ShadowMappedPathModules: [][]string{{"openconfig-complex", "openconfig-complex"}, {"openconfig-complex"}},
						},
					},
					ListKeys: map[string]*ListKey{
						"key1": {
							Name:     "Key1",
							LangType: &MappedType{NativeType: "uint32", ZeroValue: "0"},
						},
						"key2": {
							Name:     "Key2",
							LangType: &MappedType{NativeType: "uint64", ZeroValue: "0"},
						},
					},
					PackageName: "",
					IsFakeRoot:  false,
				},
			},
			Enums: map[string]*EnumeratedYANGType{
				"Complex_CycloneScales_Enum": {
					Name: "Complex_CycloneScales_Enum",
					Kind: DerivedUnionEnumerationType,
					ValuePrefix: []string{
						"openconfig-complex",
						"model",
						"a",
						"single-key",
						"config",
						"leaf-default-override",
					},
					TypeName: "cyclone-scales",
					ValToYANGDetails: []*ygot.EnumDefinition{
						{
							Name:           "NORMAL",
							DefiningModule: "",
						},
						{
							Name:           "SUPER",
							DefiningModule: "",
						},
					},
				},
				"Complex_SOFTWARE": {
					Name:     "Complex_SOFTWARE",
					Kind:     IdentityType,
					TypeName: "identityref",
					ValToYANGDetails: []*ygot.EnumDefinition{
						{Name: "OS", DefiningModule: "openconfig-complex"},
					},
				},
				"Complex_WeekendDays": {
					Name: "Complex_WeekendDays",
					Kind: DerivedEnumerationType,
					ValuePrefix: []string{
						"openconfig-complex",
						"model",
						"a",
						"single-key",
						"config",
						"key",
					},
					TypeName: "days-of-week",
					ValToYANGDetails: []*ygot.EnumDefinition{
						{
							Name:           "SATURDAY",
							DefiningModule: "",
						},
						{
							Name:           "SUNDAY",
							DefiningModule: "",
						},
					},
				},
				"SingleKey_SimpleUnionEnum_Enum": {
					Name: "SingleKey_SimpleUnionEnum_Enum",
					Kind: UnionEnumerationType,
					ValuePrefix: []string{
						"openconfig-complex",
						"model",
						"a",
						"single-key",
						"config",
						"simple-union-enum",
					},
					TypeName: "union",
					ValToYANGDetails: []*ygot.EnumDefinition{
						{
							Name:           "ONE",
							DefiningModule: "",
						},
						{
							Name:           "TWO",
							DefiningModule: "",
						},
						{
							Name:           "THREE",
							DefiningModule: "",
						},
					},
				},
				"SingleKey_SingletonUnionEnum_Enum": {
					Name: "SingleKey_SingletonUnionEnum_Enum",
					Kind: UnionEnumerationType,
					ValuePrefix: []string{
						"openconfig-complex",
						"model",
						"a",
						"single-key",
						"config",
						"singleton-union-enum",
					},
					TypeName: "union",
					ValToYANGDetails: []*ygot.EnumDefinition{
						{
							Name:           "UN",
							DefiningModule: "",
						},
						{
							Name:           "DEUX",
							DefiningModule: "",
						},
						{
							Name:           "TROIS",
							DefiningModule: "",
						},
					},
				},
			},
			ModelData: []*gpb.ModelData{{Name: "openconfig-complex"}},
		},
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			tt.inOpts.ParseOptions.ExcludeModules = tt.inExcludeModules
			got, err := GenerateIR(tt.inYANGFiles, tt.inIncludePaths, tt.inLangMapperFn, tt.inOpts)
			if diff := errdiff.Substring(err, tt.wantErrSubstring); diff != "" {
				t.Fatalf("did not get expected error, %s", diff)
			}
			if diff := cmp.Diff(got, tt.wantIR, cmpopts.IgnoreUnexported(IR{}, ParsedDirectory{}), protocmp.Transform()); diff != "" {
				t.Fatalf("did not get expected IR, diff(-got,+want):\n%s", diff)
			}
		})
	}
}
