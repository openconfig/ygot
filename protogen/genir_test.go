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

package protogen

import (
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/openconfig/gnmi/errdiff"
	gpb "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/ygot/genutil"
	"github.com/openconfig/ygot/ygen"
	"github.com/openconfig/ygot/ygot"
	"google.golang.org/protobuf/testing/protocmp"
)

// datapath is the path to common YANG test modules.
const datapath = "../testdata/modules"

func protoIR(nestedDirectories bool) *ygen.IR {
	packageName := "model"
	if nestedDirectories {
		packageName = ""
	}

	return &ygen.IR{
		Directories: map[string]*ygen.ParsedDirectory{
			"/device": {
				Name:       "Device",
				Type:       ygen.Container,
				Path:       "/device",
				SchemaPath: "",
				Fields: map[string]*ygen.NodeDetails{
					"model": {
						Name: "model",
						YANGDetails: ygen.YANGNodeDetails{
							Name:              "model",
							Defaults:          nil,
							BelongingModule:   "openconfig-complex",
							RootElementModule: "openconfig-complex",
							DefiningModule:    "openconfig-complex",
							Path:              "/openconfig-complex/model",
							SchemaPath:        "/model",
							LeafrefTargetPath: "",
							Description:       "",
						},
						Type:                    ygen.ContainerNode,
						LangType:                nil,
						MappedPaths:             [][]string{{"", "model"}},
						MappedPathModules:       [][]string{{"", "openconfig-complex"}},
						ShadowMappedPaths:       nil,
						ShadowMappedPathModules: nil,
					},
					"example-presence": {
						Name: "example_presence",
						YANGDetails: ygen.YANGNodeDetails{
							Name:              "example-presence",
							Defaults:          nil,
							BelongingModule:   "openconfig-complex",
							RootElementModule: "openconfig-complex",
							DefiningModule:    "openconfig-complex",
							Path:              "/openconfig-complex/example-presence",
							SchemaPath:        "/example-presence",
							LeafrefTargetPath: "",
							PresenceStatement: ygot.String("This is an example presence container"),
							Description:       "",
						},
						Type:                    ygen.ContainerNode,
						LangType:                nil,
						MappedPaths:             [][]string{{"", "example-presence"}},
						MappedPathModules:       [][]string{{"", "openconfig-complex"}},
						ShadowMappedPaths:       nil,
						ShadowMappedPathModules: nil,
					},
				},
				IsFakeRoot:  true,
				PackageName: "",
			},
			"/openconfig-complex/example-presence": {
				Name:                      "ExamplePresence",
				Type:                      ygen.Container,
				Path:                      "/openconfig-complex/example-presence",
				SchemaPath:                "/example-presence",
				Fields:                    map[string]*ygen.NodeDetails{},
				PackageName:               "",
				ListKeys:                  nil,
				IsFakeRoot:                false,
				BelongingModule:           "openconfig-complex",
				RootElementModule:         "openconfig-complex",
				DefiningModule:            "openconfig-complex",
				TelemetryAtomic:           true,
				CompressedTelemetryAtomic: false,
				PresenceContainer:         true,
			},
			"/openconfig-complex/model": {
				Name:       "Model",
				Type:       ygen.Container,
				Path:       "/openconfig-complex/model",
				SchemaPath: "/model",
				Fields: map[string]*ygen.NodeDetails{
					"anydata-leaf": {
						Name: "anydata_leaf",
						YANGDetails: ygen.YANGNodeDetails{
							Name:              "anydata-leaf",
							Defaults:          nil,
							BelongingModule:   "openconfig-complex",
							RootElementModule: "openconfig-complex",
							DefiningModule:    "openconfig-complex",
							Path:              "/openconfig-complex/model/anydata-leaf",
							SchemaPath:        "/model/anydata-leaf",
							LeafrefTargetPath: "",
							Description:       "some anydata leaf",
						},
						Type:                    ygen.AnyDataNode,
						LangType:                nil,
						MappedPaths:             [][]string{{"", "model", "anydata-leaf"}},
						MappedPathModules:       [][]string{{"", "openconfig-complex", "openconfig-complex"}},
						ShadowMappedPaths:       nil,
						ShadowMappedPathModules: nil,
					},
					"dateref": {
						Name: "dateref",
						YANGDetails: ygen.YANGNodeDetails{
							Name:              "dateref",
							Defaults:          nil,
							BelongingModule:   "openconfig-complex",
							RootElementModule: "openconfig-complex",
							DefiningModule:    "openconfig-complex",
							Path:              "/openconfig-complex/model/dateref",
							SchemaPath:        "/model/dateref",
							LeafrefTargetPath: "/openconfig-complex/model/a/single-key/config/dates",
							Description:       "",
						},
						Type: ygen.LeafNode,
						LangType: &ygen.MappedType{
							NativeType:   "ywrapper.UintValue",
							ZeroValue:    "",
							DefaultValue: nil,
						},
						MappedPaths:             [][]string{{"", "model", "dateref"}},
						MappedPathModules:       [][]string{{"", "openconfig-complex", "openconfig-complex"}},
						ShadowMappedPaths:       nil,
						ShadowMappedPathModules: nil,
					},
					"multi-key": {
						Name: "multi_key",
						YANGDetails: ygen.YANGNodeDetails{
							Name:              "multi-key",
							Defaults:          nil,
							BelongingModule:   "openconfig-complex",
							RootElementModule: "openconfig-complex",
							DefiningModule:    "openconfig-complex",
							Path:              "/openconfig-complex/model/b/multi-key",
							SchemaPath:        "/model/b/multi-key",
							LeafrefTargetPath: "",
							Description:       "",
						},
						Type:                    ygen.ListNode,
						LangType:                nil,
						MappedPaths:             [][]string{{"", "model", "b", "multi-key"}},
						MappedPathModules:       [][]string{{"", "openconfig-complex", "openconfig-complex", "openconfig-complex"}},
						ShadowMappedPaths:       nil,
						ShadowMappedPathModules: nil,
					},
					"single-key": {
						Name: "single_key",
						YANGDetails: ygen.YANGNodeDetails{
							Name:              "single-key",
							Defaults:          nil,
							BelongingModule:   "openconfig-complex",
							RootElementModule: "openconfig-complex",
							DefiningModule:    "openconfig-complex",
							Path:              "/openconfig-complex/model/a/single-key",
							SchemaPath:        "/model/a/single-key",
							LeafrefTargetPath: "",
							Description:       "",
						},
						Type:                    ygen.ListNode,
						LangType:                nil,
						MappedPaths:             [][]string{{"", "model", "a", "single-key"}},
						MappedPathModules:       [][]string{{"", "openconfig-complex", "openconfig-complex", "openconfig-complex"}},
						ShadowMappedPaths:       nil,
						ShadowMappedPathModules: nil,
					},
					"unkeyed-list": {
						Name: "unkeyed_list",
						YANGDetails: ygen.YANGNodeDetails{
							Name:              "unkeyed-list",
							BelongingModule:   "openconfig-complex",
							RootElementModule: "openconfig-complex",
							DefiningModule:    "openconfig-complex",
							Path:              "/openconfig-complex/model/c/unkeyed-list",
							SchemaPath:        "/model/c/unkeyed-list",
							ConfigFalse:       true,
						},
						Type:              ygen.ListNode,
						MappedPaths:       [][]string{{"", "model", "c", "unkeyed-list"}},
						MappedPathModules: [][]string{{"", "openconfig-complex", "openconfig-complex", "openconfig-complex"}},
					},
				},
				ListKeys:          nil,
				PackageName:       "",
				BelongingModule:   "openconfig-complex",
				RootElementModule: "openconfig-complex",
				DefiningModule:    "openconfig-complex",
			},
			"/openconfig-complex/model/a/single-key": {
				Name:       "SingleKey",
				Type:       ygen.List,
				Path:       "/openconfig-complex/model/a/single-key",
				SchemaPath: "/model/a/single-key",
				Fields: map[string]*ygen.NodeDetails{
					"dates": {
						Name: "dates",
						YANGDetails: ygen.YANGNodeDetails{
							Name:              "dates",
							Defaults:          []string{"5"},
							BelongingModule:   "openconfig-complex",
							RootElementModule: "openconfig-complex",
							DefiningModule:    "openconfig-complex",
							Path:              "/openconfig-complex/model/a/single-key/config/dates",
							SchemaPath:        "/model/a/single-key/config/dates",
							ShadowSchemaPath:  "/model/a/single-key/state/dates",
							LeafrefTargetPath: "",
							Description:       "",
						},
						Type: ygen.LeafListNode,
						LangType: &ygen.MappedType{
							NativeType:   "ywrapper.UintValue",
							ZeroValue:    "",
							DefaultValue: nil,
						},
						MappedPaths:             [][]string{{"", "model", "a", "single-key", "config", "dates"}},
						MappedPathModules:       [][]string{{"", "openconfig-complex", "openconfig-complex", "openconfig-complex", "openconfig-complex", "openconfig-complex"}},
						ShadowMappedPaths:       [][]string{{"", "model", "a", "single-key", "state", "dates"}},
						ShadowMappedPathModules: [][]string{{"", "openconfig-complex", "openconfig-complex", "openconfig-complex", "openconfig-complex", "openconfig-complex"}},
					},
					"dates-with-defaults": {
						Name: "dates_with_defaults",
						YANGDetails: ygen.YANGNodeDetails{
							Name:              "dates-with-defaults",
							Defaults:          []string{"1", "2"},
							BelongingModule:   "openconfig-complex",
							RootElementModule: "openconfig-complex",
							DefiningModule:    "openconfig-complex",
							Path:              "/openconfig-complex/model/a/single-key/config/dates-with-defaults",
							SchemaPath:        "/model/a/single-key/config/dates-with-defaults",
							ShadowSchemaPath:  "/model/a/single-key/state/dates-with-defaults",
							LeafrefTargetPath: "",
							Description:       "",
						},
						Type: ygen.LeafListNode,
						LangType: &ygen.MappedType{
							NativeType:   "ywrapper.UintValue",
							ZeroValue:    "",
							DefaultValue: nil,
						},
						MappedPaths:             [][]string{{"", "model", "a", "single-key", "config", "dates-with-defaults"}},
						MappedPathModules:       [][]string{{"", "openconfig-complex", "openconfig-complex", "openconfig-complex", "openconfig-complex", "openconfig-complex"}},
						ShadowMappedPaths:       [][]string{{"", "model", "a", "single-key", "state", "dates-with-defaults"}},
						ShadowMappedPathModules: [][]string{{"", "openconfig-complex", "openconfig-complex", "openconfig-complex", "openconfig-complex", "openconfig-complex"}},
					},
					"iref": {
						Name: "iref",
						YANGDetails: ygen.YANGNodeDetails{
							Name:              "iref",
							Defaults:          nil,
							BelongingModule:   "openconfig-complex",
							RootElementModule: "openconfig-complex",
							DefiningModule:    "openconfig-complex",
							Path:              "/openconfig-complex/model/a/single-key/config/iref",
							SchemaPath:        "/model/a/single-key/config/iref",
							ShadowSchemaPath:  "/model/a/single-key/state/iref",
							LeafrefTargetPath: "",
							Description:       "",
						},
						Type: ygen.LeafNode,
						LangType: &ygen.MappedType{
							NativeType:            "openconfig.enums.ComplexSOFTWARE",
							UnionTypes:            nil,
							IsEnumeratedValue:     true,
							EnumeratedYANGTypeKey: "/openconfig-complex/SOFTWARE",
							ZeroValue:             "",
							DefaultValue:          nil,
						},
						MappedPaths: [][]string{
							{"", "model", "a", "single-key", "config", "iref"},
						},
						MappedPathModules: [][]string{
							{
								"", "openconfig-complex", "openconfig-complex", "openconfig-complex",
								"openconfig-complex", "openconfig-complex",
							},
						},
						ShadowMappedPaths: [][]string{
							{"", "model", "a", "single-key", "state", "iref"},
						},
						ShadowMappedPathModules: [][]string{
							{
								"", "openconfig-complex", "openconfig-complex", "openconfig-complex",
								"openconfig-complex", "openconfig-complex",
							},
						},
					},
					"iref2": {
						Name: "iref2",
						YANGDetails: ygen.YANGNodeDetails{
							Name:              "iref2",
							Defaults:          nil,
							BelongingModule:   "openconfig-complex",
							RootElementModule: "openconfig-complex",
							DefiningModule:    "openconfig-complex",
							Path:              "/openconfig-complex/model/a/single-key/config/iref2",
							SchemaPath:        "/model/a/single-key/config/iref2",
							ShadowSchemaPath:  "/model/a/single-key/state/iref2",
							LeafrefTargetPath: "",
							Description:       "",
						},
						Type: ygen.LeafNode,
						LangType: &ygen.MappedType{
							NativeType:            "openconfig.enums.ComplexProgram",
							UnionTypes:            nil,
							IsEnumeratedValue:     true,
							EnumeratedYANGTypeKey: "/openconfig-complex/program",
							ZeroValue:             "",
							DefaultValue:          nil,
						},
						MappedPaths: [][]string{
							{"", "model", "a", "single-key", "config", "iref2"},
						},
						MappedPathModules: [][]string{
							{
								"", "openconfig-complex", "openconfig-complex", "openconfig-complex",
								"openconfig-complex", "openconfig-complex",
							},
						},
						ShadowMappedPaths: [][]string{
							{"", "model", "a", "single-key", "state", "iref2"},
						},
						ShadowMappedPathModules: [][]string{
							{
								"", "openconfig-complex", "openconfig-complex", "openconfig-complex",
								"openconfig-complex", "openconfig-complex",
							},
						},
					},
					"key": {
						Name: "key",
						YANGDetails: ygen.YANGNodeDetails{
							Name:              "key",
							Defaults:          nil,
							BelongingModule:   "openconfig-complex",
							RootElementModule: "openconfig-complex",
							DefiningModule:    "openconfig-complex",
							Path:              "/openconfig-complex/model/a/single-key/config/key",
							SchemaPath:        "/model/a/single-key/config/key",
							ShadowSchemaPath:  "/model/a/single-key/state/key",
							LeafrefTargetPath: "",
							Description:       "",
						},
						Type: ygen.LeafNode,
						LangType: &ygen.MappedType{
							NativeType: "",
							UnionTypes: map[string]ygen.MappedUnionSubtype{
								"openconfig.enums.ComplexWeekendDays": {
									Index:                 0,
									EnumeratedYANGTypeKey: "/openconfig-complex/weekend-days",
								},
								"uint64": {
									Index:                 1,
									EnumeratedYANGTypeKey: "",
								},
							},
							IsEnumeratedValue:     false,
							EnumeratedYANGTypeKey: "",
							ZeroValue:             "",
							DefaultValue:          nil,
						},
						MappedPaths: [][]string{
							{"", "model", "a", "single-key", "config", "key"},
							{"", "model", "a", "single-key", "key"},
						},
						MappedPathModules: [][]string{
							{
								"", "openconfig-complex", "openconfig-complex", "openconfig-complex",
								"openconfig-complex", "openconfig-complex",
							},
							{
								"",
								"openconfig-complex",
								"openconfig-complex",
								"openconfig-complex",
								"openconfig-complex",
							},
						},
						ShadowMappedPaths: [][]string{
							{"", "model", "a", "single-key", "state", "key"},
							{"", "model", "a", "single-key", "key"},
						},
						ShadowMappedPathModules: [][]string{
							{
								"", "openconfig-complex", "openconfig-complex", "openconfig-complex",
								"openconfig-complex", "openconfig-complex",
							},
							{
								"",
								"openconfig-complex",
								"openconfig-complex",
								"openconfig-complex",
								"openconfig-complex",
							},
						},
					},
					"leaf-default-override": {
						Name: "leaf_default_override",
						YANGDetails: ygen.YANGNodeDetails{
							Name:              "leaf-default-override",
							Defaults:          []string{"3"},
							BelongingModule:   "openconfig-complex",
							RootElementModule: "openconfig-complex",
							DefiningModule:    "openconfig-complex",
							Path:              "/openconfig-complex/model/a/single-key/config/leaf-default-override",
							SchemaPath:        "/model/a/single-key/config/leaf-default-override",
							ShadowSchemaPath:  "/model/a/single-key/state/leaf-default-override",
							LeafrefTargetPath: "",
							Description:       "",
						},
						Type: ygen.LeafNode,
						LangType: &ygen.MappedType{
							NativeType: "",
							UnionTypes: map[string]ygen.MappedUnionSubtype{
								"openconfig.enums.ComplexCycloneScalesEnum": {
									Index:                 0,
									EnumeratedYANGTypeKey: "/openconfig-complex/cyclone-scales",
								},
								"uint64": {
									Index:                 1,
									EnumeratedYANGTypeKey: "",
								},
							},
							IsEnumeratedValue:     false,
							EnumeratedYANGTypeKey: "",
							ZeroValue:             "",
							DefaultValue:          nil,
						},
						MappedPaths: [][]string{
							{"", "model", "a", "single-key", "config", "leaf-default-override"},
						},
						MappedPathModules: [][]string{
							{
								"", "openconfig-complex", "openconfig-complex", "openconfig-complex",
								"openconfig-complex", "openconfig-complex",
							},
						},
						ShadowMappedPaths: [][]string{
							{"", "model", "a", "single-key", "state", "leaf-default-override"},
						},
						ShadowMappedPathModules: [][]string{
							{
								"", "openconfig-complex", "openconfig-complex", "openconfig-complex",
								"openconfig-complex", "openconfig-complex",
							},
						},
					},
					"simple-union-enum": {
						Name: "simple_union_enum",
						YANGDetails: ygen.YANGNodeDetails{
							Name:              "simple-union-enum",
							Defaults:          []string{"TWO"},
							BelongingModule:   "openconfig-complex",
							RootElementModule: "openconfig-complex",
							DefiningModule:    "openconfig-complex",
							Path:              "/openconfig-complex/model/a/single-key/config/simple-union-enum",
							SchemaPath:        "/model/a/single-key/config/simple-union-enum",
							ShadowSchemaPath:  "/model/a/single-key/state/simple-union-enum",
							LeafrefTargetPath: "",
							Description:       "",
						},
						Type: ygen.LeafNode,
						LangType: &ygen.MappedType{
							NativeType: "",
							UnionTypes: map[string]ygen.MappedUnionSubtype{
								"SimpleUnionEnumEnum": {
									Index:                 0,
									EnumeratedYANGTypeKey: "/openconfig-complex/single-key-config/simple-union-enum",
								},
								"uint64": {
									Index:                 1,
									EnumeratedYANGTypeKey: "",
								},
							},
							IsEnumeratedValue:     false,
							EnumeratedYANGTypeKey: "",
							ZeroValue:             "",
							DefaultValue:          nil,
						},
						MappedPaths: [][]string{
							{"", "model", "a", "single-key", "config", "simple-union-enum"},
						},
						MappedPathModules: [][]string{
							{
								"", "openconfig-complex", "openconfig-complex", "openconfig-complex",
								"openconfig-complex", "openconfig-complex",
							},
						},
						ShadowMappedPaths: [][]string{
							{"", "model", "a", "single-key", "state", "simple-union-enum"},
						},
						ShadowMappedPathModules: [][]string{
							{
								"", "openconfig-complex", "openconfig-complex", "openconfig-complex",
								"openconfig-complex", "openconfig-complex",
							},
						},
					},
					"singleton-union-enum": {
						Name: "singleton_union_enum",
						YANGDetails: ygen.YANGNodeDetails{
							Name:              "singleton-union-enum",
							Defaults:          []string{"DEUX"},
							BelongingModule:   "openconfig-complex",
							RootElementModule: "openconfig-complex",
							DefiningModule:    "openconfig-complex",
							Path:              "/openconfig-complex/model/a/single-key/config/singleton-union-enum",
							SchemaPath:        "/model/a/single-key/config/singleton-union-enum",
							ShadowSchemaPath:  "/model/a/single-key/state/singleton-union-enum",
							LeafrefTargetPath: "",
							Description:       "",
						},
						Type: ygen.LeafNode,
						LangType: &ygen.MappedType{
							NativeType:            "SingletonUnionEnumEnum",
							UnionTypes:            nil,
							IsEnumeratedValue:     true,
							EnumeratedYANGTypeKey: "/openconfig-complex/single-key-config/singleton-union-enum",
							ZeroValue:             "",
							DefaultValue:          nil,
						},
						MappedPaths: [][]string{
							{"", "model", "a", "single-key", "config", "singleton-union-enum"},
						},
						MappedPathModules: [][]string{
							{
								"", "openconfig-complex", "openconfig-complex", "openconfig-complex",
								"openconfig-complex", "openconfig-complex",
							},
						},
						ShadowMappedPaths: [][]string{
							{"", "model", "a", "single-key", "state", "singleton-union-enum"},
						},
						ShadowMappedPathModules: [][]string{
							{
								"", "openconfig-complex", "openconfig-complex", "openconfig-complex",
								"openconfig-complex", "openconfig-complex",
							},
						},
					},
					"typedef-enum": {
						Name: "typedef_enum",
						YANGDetails: ygen.YANGNodeDetails{
							Name:              "typedef-enum",
							Defaults:          []string{"SATURDAY"},
							BelongingModule:   "openconfig-complex",
							RootElementModule: "openconfig-complex",
							DefiningModule:    "openconfig-complex",
							Path:              "/openconfig-complex/model/a/single-key/config/typedef-enum",
							SchemaPath:        "/model/a/single-key/config/typedef-enum",
							ShadowSchemaPath:  "/model/a/single-key/state/typedef-enum",
							LeafrefTargetPath: "",
							Description:       "",
						},
						Type: ygen.LeafNode,
						LangType: &ygen.MappedType{
							NativeType:            "openconfig.enums.ComplexWeekendDays",
							UnionTypes:            nil,
							IsEnumeratedValue:     true,
							EnumeratedYANGTypeKey: "/openconfig-complex/weekend-days",
							ZeroValue:             "",
							DefaultValue:          nil,
						},
						MappedPaths: [][]string{
							{"", "model", "a", "single-key", "config", "typedef-enum"},
						},
						MappedPathModules: [][]string{
							{
								"", "openconfig-complex", "openconfig-complex", "openconfig-complex",
								"openconfig-complex", "openconfig-complex",
							},
						},
						ShadowMappedPaths: [][]string{
							{"", "model", "a", "single-key", "state", "typedef-enum"},
						},
						ShadowMappedPathModules: [][]string{
							{
								"", "openconfig-complex", "openconfig-complex", "openconfig-complex",
								"openconfig-complex", "openconfig-complex",
							},
						},
					},
					"typedef-union-enum": {
						Name: "typedef_union_enum",
						YANGDetails: ygen.YANGNodeDetails{
							Name:              "typedef-union-enum",
							Defaults:          []string{"SUPER"},
							BelongingModule:   "openconfig-complex",
							RootElementModule: "openconfig-complex",
							DefiningModule:    "openconfig-complex",
							Path:              "/openconfig-complex/model/a/single-key/config/typedef-union-enum",
							SchemaPath:        "/model/a/single-key/config/typedef-union-enum",
							ShadowSchemaPath:  "/model/a/single-key/state/typedef-union-enum",
							LeafrefTargetPath: "",
							Description:       "",
						},
						Type: ygen.LeafNode,
						LangType: &ygen.MappedType{
							NativeType: "",
							UnionTypes: map[string]ygen.MappedUnionSubtype{
								// protoLangMapper sorts by name instead of YANG order.
								"openconfig.enums.ComplexCycloneScalesEnum": {
									Index:                 0,
									EnumeratedYANGTypeKey: "/openconfig-complex/cyclone-scales",
								},
								"uint64": {
									Index:                 1,
									EnumeratedYANGTypeKey: "",
								},
							},
							IsEnumeratedValue:     false,
							EnumeratedYANGTypeKey: "",
							ZeroValue:             "",
							DefaultValue:          nil,
						},
						MappedPaths: [][]string{
							{"", "model", "a", "single-key", "config", "typedef-union-enum"},
						},
						MappedPathModules: [][]string{
							{
								"", "openconfig-complex", "openconfig-complex", "openconfig-complex",
								"openconfig-complex", "openconfig-complex",
							},
						},
						ShadowMappedPaths: [][]string{
							{"", "model", "a", "single-key", "state", "typedef-union-enum"},
						},
						ShadowMappedPathModules: [][]string{
							{
								"", "openconfig-complex", "openconfig-complex", "openconfig-complex",
								"openconfig-complex", "openconfig-complex",
							},
						},
					},
				},
				ListKeys: map[string]*ygen.ListKey{
					"key": {
						Name: "key",
						LangType: &ygen.MappedType{
							NativeType: "",
							UnionTypes: map[string]ygen.MappedUnionSubtype{
								"openconfig.enums.ComplexWeekendDays": {
									Index:                 0,
									EnumeratedYANGTypeKey: "/openconfig-complex/weekend-days",
								},
								"uint64": {
									Index:                 1,
									EnumeratedYANGTypeKey: "",
								},
							},
							ZeroValue: "",
						},
					},
				},
				ListKeyYANGNames:          []string{"key"},
				PackageName:               packageName,
				IsFakeRoot:                false,
				BelongingModule:           "openconfig-complex",
				RootElementModule:         "openconfig-complex",
				DefiningModule:            "openconfig-complex",
				TelemetryAtomic:           false,
				CompressedTelemetryAtomic: true,
			},
			"/openconfig-complex/model/b/multi-key": {
				Name:       "MultiKey",
				Type:       ygen.List,
				Path:       "/openconfig-complex/model/b/multi-key",
				SchemaPath: "/model/b/multi-key",
				Fields: map[string]*ygen.NodeDetails{
					"key1": {
						Name: "key1",
						YANGDetails: ygen.YANGNodeDetails{
							Name:              "key1",
							Defaults:          nil,
							BelongingModule:   "openconfig-complex",
							RootElementModule: "openconfig-complex",
							DefiningModule:    "openconfig-complex",
							Path:              "/openconfig-complex/model/b/multi-key/config/key1",
							SchemaPath:        "/model/b/multi-key/config/key1",
							ShadowSchemaPath:  "/model/b/multi-key/state/key1",
							LeafrefTargetPath: "",
							Description:       "",
						},
						Type:     ygen.LeafNode,
						LangType: &ygen.MappedType{NativeType: "ywrapper.UintValue"},
						MappedPaths: [][]string{
							{"", "model", "b", "multi-key", "config", "key1"},
							{"", "model", "b", "multi-key", "key1"},
						},
						MappedPathModules: [][]string{
							{
								"", "openconfig-complex", "openconfig-complex", "openconfig-complex",
								"openconfig-complex", "openconfig-complex",
							},
							{
								"",
								"openconfig-complex",
								"openconfig-complex",
								"openconfig-complex",
								"openconfig-complex",
							},
						},
						ShadowMappedPaths: [][]string{
							{"", "model", "b", "multi-key", "state", "key1"},
							{"", "model", "b", "multi-key", "key1"},
						},
						ShadowMappedPathModules: [][]string{
							{
								"", "openconfig-complex", "openconfig-complex", "openconfig-complex",
								"openconfig-complex", "openconfig-complex",
							},
							{
								"",
								"openconfig-complex",
								"openconfig-complex",
								"openconfig-complex",
								"openconfig-complex",
							},
						},
					},
					"key2": {
						Name: "key2",
						YANGDetails: ygen.YANGNodeDetails{
							Name:              "key2",
							Defaults:          nil,
							BelongingModule:   "openconfig-complex",
							RootElementModule: "openconfig-complex",
							DefiningModule:    "openconfig-complex",
							Path:              "/openconfig-complex/model/b/multi-key/config/key2",
							SchemaPath:        "/model/b/multi-key/config/key2",
							ShadowSchemaPath:  "/model/b/multi-key/state/key2",
							LeafrefTargetPath: "",
							Description:       "",
						},
						Type: ygen.LeafNode,
						LangType: &ygen.MappedType{
							NativeType:            "Key2",
							IsEnumeratedValue:     true,
							EnumeratedYANGTypeKey: "/openconfig-complex/multi-key-config/key2",
							ZeroValue:             "",
						},
						MappedPaths: [][]string{
							{"", "model", "b", "multi-key", "config", "key2"},
							{"", "model", "b", "multi-key", "key2"},
						},
						MappedPathModules: [][]string{
							{
								"", "openconfig-complex", "openconfig-complex", "openconfig-complex",
								"openconfig-complex", "openconfig-complex",
							},
							{
								"",
								"openconfig-complex",
								"openconfig-complex",
								"openconfig-complex",
								"openconfig-complex",
							},
						},
						ShadowMappedPaths: [][]string{
							{"", "model", "b", "multi-key", "state", "key2"},
							{"", "model", "b", "multi-key", "key2"},
						},
						ShadowMappedPathModules: [][]string{
							{
								"", "openconfig-complex", "openconfig-complex", "openconfig-complex",
								"openconfig-complex", "openconfig-complex",
							},
							{
								"",
								"openconfig-complex",
								"openconfig-complex",
								"openconfig-complex",
								"openconfig-complex",
							},
						},
					},
				},
				ListKeys: map[string]*ygen.ListKey{
					"key1": {
						Name:     "key1",
						LangType: &ygen.MappedType{NativeType: "uint64", ZeroValue: ""},
					},
					"key2": {
						Name: "key2",
						LangType: &ygen.MappedType{
							NativeType:            "Key2",
							IsEnumeratedValue:     true,
							EnumeratedYANGTypeKey: "/openconfig-complex/multi-key-config/key2",
							ZeroValue:             "",
						},
					},
				},
				ListKeyYANGNames:          []string{"key1", "key2"},
				PackageName:               packageName,
				IsFakeRoot:                false,
				BelongingModule:           "openconfig-complex",
				RootElementModule:         "openconfig-complex",
				DefiningModule:            "openconfig-complex",
				TelemetryAtomic:           true,
				CompressedTelemetryAtomic: false,
			},
			"/openconfig-complex/model/c/unkeyed-list": {
				Name:       "UnkeyedList",
				Type:       ygen.List,
				Path:       "/openconfig-complex/model/c/unkeyed-list",
				SchemaPath: "/model/c/unkeyed-list",
				Fields: map[string]*ygen.NodeDetails{
					"field": {
						Name: "field",
						YANGDetails: ygen.YANGNodeDetails{
							Name:              "field",
							BelongingModule:   "openconfig-complex",
							RootElementModule: "openconfig-complex",
							DefiningModule:    "openconfig-complex",
							Path:              "/openconfig-complex/model/c/unkeyed-list/field",
							SchemaPath:        "/model/c/unkeyed-list/field",
							ConfigFalse:       true,
						},
						Type:              ygen.LeafNode,
						LangType:          &ygen.MappedType{NativeType: "ywrapper.BytesValue"},
						MappedPaths:       [][]string{{"", "model", "c", "unkeyed-list", "field"}},
						MappedPathModules: [][]string{{"", "openconfig-complex", "openconfig-complex", "openconfig-complex", "openconfig-complex"}},
					},
				},
				PackageName:       packageName,
				BelongingModule:   "openconfig-complex",
				RootElementModule: "openconfig-complex",
				DefiningModule:    "openconfig-complex",
				ConfigFalse:       true,
			},
		},
		Enums: map[string]*ygen.EnumeratedYANGType{
			"/openconfig-complex/cyclone-scales": {
				Name:     "ComplexCycloneScalesEnum",
				Kind:     ygen.DerivedUnionEnumerationType,
				TypeName: "cyclone-scales",
				ValToYANGDetails: []ygot.EnumDefinition{
					{
						Name:           "NORMAL",
						DefiningModule: "",
						Value:          0,
					},
					{
						Name:           "SUPER",
						DefiningModule: "",
						Value:          1,
					},
				},
			},
			"/openconfig-complex/SOFTWARE": {
				Name:             "ComplexSOFTWARE",
				Kind:             ygen.IdentityType,
				IdentityBaseName: "SOFTWARE",
				TypeName:         "identityref",
				ValToYANGDetails: []ygot.EnumDefinition{
					{Name: "OS", DefiningModule: "openconfig-complex"},
				},
			},
			"/openconfig-complex/program": {
				Name:             "ComplexProgram",
				Kind:             ygen.IdentityType,
				IdentityBaseName: "SOFTWARE",
				TypeName:         "program",
				ValToYANGDetails: []ygot.EnumDefinition{
					{Name: "OS", DefiningModule: "openconfig-complex"},
				},
			},
			"/openconfig-complex/multi-key-config/key2": {
				Name:     "MultiKeyKey2",
				Kind:     ygen.SimpleEnumerationType,
				TypeName: "enumeration",
				ValToYANGDetails: []ygot.EnumDefinition{
					{
						Name:           "RED",
						DefiningModule: "",
						Value:          0,
					},
					{
						Name:           "BLUE",
						DefiningModule: "",
						Value:          1,
					},
				},
			},
			"/openconfig-complex/weekend-days": {
				Name:     "ComplexWeekendDays",
				Kind:     ygen.DerivedEnumerationType,
				TypeName: "weekend-days",
				ValToYANGDetails: []ygot.EnumDefinition{
					{
						Name:           "SATURDAY",
						DefiningModule: "",
						Value:          0,
					},
					{
						Name:           "SUNDAY",
						DefiningModule: "",
						Value:          1,
					},
				},
			},
			"/openconfig-complex/single-key-config/simple-union-enum": {
				Name:     "SingleKeySimpleUnionEnumEnum",
				Kind:     ygen.UnionEnumerationType,
				TypeName: "union",
				ValToYANGDetails: []ygot.EnumDefinition{
					{
						Name:           "ONE",
						DefiningModule: "",
						Value:          0,
					},
					{
						Name:           "TWO",
						DefiningModule: "",
						Value:          1,
					},
					{
						Name:           "THREE",
						DefiningModule: "",
						Value:          2,
					},
				},
			},
			"/openconfig-complex/single-key-config/singleton-union-enum": {
				Name:     "SingleKeySingletonUnionEnumEnum",
				Kind:     ygen.UnionEnumerationType,
				TypeName: "union",
				ValToYANGDetails: []ygot.EnumDefinition{
					{
						Name:           "UN",
						DefiningModule: "",
						Value:          0,
					},
					{
						Name:           "DEUX",
						DefiningModule: "",
						Value:          1,
					},
					{
						Name:           "TROIS",
						DefiningModule: "",
						Value:          2,
					},
				},
			},
		},
		ModelData: []*gpb.ModelData{
			{Name: "openconfig-complex"},
			{Name: "openconfig-extensions"},
		},
	}
}

func TestGenerateIR(t *testing.T) {
	tests := []struct {
		desc             string
		inYANGFiles      []string
		inIncludePaths   []string
		inExcludeModules []string
		inLangMapper     ygen.LangMapper
		inOpts           ygen.IROptions
		wantIR           *ygen.IR
		wantErrSubstring string
	}{{
		desc:        "complex openconfig test with compression using ProtoLangMapper with nested directories",
		inYANGFiles: []string{filepath.Join(datapath, "openconfig-complex.yang")},
		inLangMapper: func() ygen.LangMapper {
			return NewProtoLangMapper(DefaultBasePackageName, DefaultEnumPackageName)
		}(),
		inOpts: ygen.IROptions{
			NestedDirectories: true,
			AbsoluteMapPaths:  true,
			TransformationOptions: ygen.TransformationOpts{
				CompressBehaviour:                    genutil.PreferIntendedConfig,
				ShortenEnumLeafNames:                 true,
				EnumOrgPrefixesToTrim:                []string{"openconfig"},
				UseDefiningModuleForTypedefEnumNames: true,
				EnumerationsUseUnderscores:           false,
				GenerateFakeRoot:                     true,
			},
			AppendEnumSuffixForSimpleUnionEnums: true,
		},
		wantIR: protoIR(true),
	}, {
		desc:        "complex openconfig test with compression using ProtoLangMapper",
		inYANGFiles: []string{filepath.Join(datapath, "openconfig-complex.yang")},
		inLangMapper: func() ygen.LangMapper {
			return NewProtoLangMapper(DefaultBasePackageName, DefaultEnumPackageName)
		}(),
		inOpts: ygen.IROptions{
			NestedDirectories: false,
			AbsoluteMapPaths:  true,
			TransformationOptions: ygen.TransformationOpts{
				CompressBehaviour:                    genutil.PreferIntendedConfig,
				ShortenEnumLeafNames:                 true,
				EnumOrgPrefixesToTrim:                []string{"openconfig"},
				UseDefiningModuleForTypedefEnumNames: true,
				EnumerationsUseUnderscores:           false,
				GenerateFakeRoot:                     true,
			},
			AppendEnumSuffixForSimpleUnionEnums: true,
		},
		wantIR: protoIR(false),
	}, {
		desc: "simple openconfig test without compression using ProtoLangMapper with nested directories",
		inYANGFiles: []string{
			filepath.Join(datapath, "openconfig-simple.yang"),
			filepath.Join(datapath, "openconfig-simple-augment2.yang"),
		},
		inLangMapper: func() ygen.LangMapper {
			return NewProtoLangMapper(DefaultBasePackageName, DefaultEnumPackageName)
		}(),
		inOpts: ygen.IROptions{
			TransformationOptions: ygen.TransformationOpts{
				CompressBehaviour:                    genutil.Uncompressed,
				ShortenEnumLeafNames:                 true,
				EnumOrgPrefixesToTrim:                []string{"openconfig"},
				UseDefiningModuleForTypedefEnumNames: true,
				EnumerationsUseUnderscores:           true,
				GenerateFakeRoot:                     true,
			},
			AppendEnumSuffixForSimpleUnionEnums: true,
		},
		wantIR: &ygen.IR{
			Directories: map[string]*ygen.ParsedDirectory{
				"/device": {
					Name:       "Device",
					Type:       ygen.Container,
					Path:       "/device",
					SchemaPath: "",
					Fields: map[string]*ygen.NodeDetails{
						"parent": {
							Name: "parent",
							YANGDetails: ygen.YANGNodeDetails{
								Name:              "parent",
								Defaults:          nil,
								BelongingModule:   "openconfig-simple",
								RootElementModule: "openconfig-simple",
								DefiningModule:    "openconfig-simple",
								Path:              "/openconfig-simple/parent",
								SchemaPath:        "/parent",
								LeafrefTargetPath: "",
								Description:       "I am a parent container\nthat has 4 children.",
							},
							Type:                    ygen.ContainerNode,
							MappedPaths:             [][]string{{"parent"}},
							MappedPathModules:       [][]string{{"openconfig-simple"}},
							ShadowMappedPaths:       nil,
							ShadowMappedPathModules: nil,
						},
						"remote-container": {
							Name: "remote_container",
							YANGDetails: ygen.YANGNodeDetails{
								Name:              "remote-container",
								Defaults:          nil,
								BelongingModule:   "openconfig-simple",
								RootElementModule: "openconfig-simple",
								DefiningModule:    "openconfig-remote",
								Path:              "/openconfig-simple/remote-container",
								SchemaPath:        "/remote-container",
								LeafrefTargetPath: "",
								Description:       "",
							},
							Type:                    ygen.ContainerNode,
							MappedPaths:             [][]string{{"remote-container"}},
							MappedPathModules:       [][]string{{"openconfig-simple"}},
							ShadowMappedPaths:       nil,
							ShadowMappedPathModules: nil,
						},
					},
					IsFakeRoot: true,
				},
				"/openconfig-simple/parent": {
					Name:       "Parent",
					Type:       ygen.Container,
					Path:       "/openconfig-simple/parent",
					SchemaPath: "/parent",
					Fields: map[string]*ygen.NodeDetails{
						"child": {
							Name: "child",
							YANGDetails: ygen.YANGNodeDetails{
								Name:              "child",
								Defaults:          nil,
								BelongingModule:   "openconfig-simple",
								RootElementModule: "openconfig-simple",
								DefiningModule:    "openconfig-simple",
								Path:              "/openconfig-simple/parent/child",
								SchemaPath:        "/parent/child",
								LeafrefTargetPath: "",
								Description:       "",
							},
							Type:                    ygen.ContainerNode,
							LangType:                nil,
							MappedPaths:             [][]string{{"child"}},
							MappedPathModules:       [][]string{{"openconfig-simple"}},
							ShadowMappedPaths:       nil,
							ShadowMappedPathModules: nil,
						},
					},
					PackageName:       "openconfig_simple",
					BelongingModule:   "openconfig-simple",
					RootElementModule: "openconfig-simple",
					DefiningModule:    "openconfig-simple",
				},
				"/openconfig-simple/parent/child": {
					Name:       "Child",
					Type:       ygen.Container,
					Path:       "/openconfig-simple/parent/child",
					SchemaPath: "/parent/child",
					Fields: map[string]*ygen.NodeDetails{
						"config": {
							Name: "config",
							YANGDetails: ygen.YANGNodeDetails{
								Name:              "config",
								BelongingModule:   "openconfig-simple",
								RootElementModule: "openconfig-simple",
								DefiningModule:    "openconfig-simple",
								Path:              "/openconfig-simple/parent/child/config",
								SchemaPath:        "/parent/child/config",
								LeafrefTargetPath: "",
								Description:       "",
							},
							Type:              1,
							MappedPaths:       [][]string{{"config"}},
							MappedPathModules: [][]string{{"openconfig-simple"}},
						},
						"state": {
							Name: "state",
							YANGDetails: ygen.YANGNodeDetails{
								Name:              "state",
								BelongingModule:   "openconfig-simple",
								RootElementModule: "openconfig-simple",
								DefiningModule:    "openconfig-simple",
								Path:              "/openconfig-simple/parent/child/state",
								SchemaPath:        "/parent/child/state",
								LeafrefTargetPath: "",
								Description:       "",
								ConfigFalse:       true,
							},
							Type:              1,
							MappedPaths:       [][]string{{"state"}},
							MappedPathModules: [][]string{{"openconfig-simple"}},
						},
					},
					ListKeys:          nil,
					PackageName:       "openconfig_simple.parent",
					BelongingModule:   "openconfig-simple",
					RootElementModule: "openconfig-simple",
					DefiningModule:    "openconfig-simple",
				},
				"/openconfig-simple/parent/child/config": {
					Name:       "Config",
					Type:       ygen.Container,
					Path:       "/openconfig-simple/parent/child/config",
					SchemaPath: "/parent/child/config",
					Fields: map[string]*ygen.NodeDetails{
						"four": {
							Name: "four",
							YANGDetails: ygen.YANGNodeDetails{
								Name:              "four",
								Defaults:          nil,
								BelongingModule:   "openconfig-simple",
								RootElementModule: "openconfig-simple",
								DefiningModule:    "openconfig-simple",
								Path:              "/openconfig-simple/parent/child/config/four",
								SchemaPath:        "/parent/child/config/four",
								LeafrefTargetPath: "",
								Description:       "",
							},
							Type: 3,
							LangType: &ygen.MappedType{
								NativeType:        "ywrapper.BytesValue",
								UnionTypes:        nil,
								IsEnumeratedValue: false,
								ZeroValue:         "",
								DefaultValue:      nil,
							},
							MappedPaths:             [][]string{{"four"}},
							MappedPathModules:       [][]string{{"openconfig-simple"}},
							ShadowMappedPaths:       nil,
							ShadowMappedPathModules: nil,
						},
						"one": {
							Name: "one",
							YANGDetails: ygen.YANGNodeDetails{
								Name:              "one",
								Defaults:          nil,
								BelongingModule:   "openconfig-simple",
								RootElementModule: "openconfig-simple",
								DefiningModule:    "openconfig-simple",
								Path:              "/openconfig-simple/parent/child/config/one",
								SchemaPath:        "/parent/child/config/one",
								LeafrefTargetPath: "",
								Description:       "",
							},
							Type: 3,
							LangType: &ygen.MappedType{
								NativeType:        "ywrapper.StringValue",
								UnionTypes:        nil,
								IsEnumeratedValue: false,
								ZeroValue:         ``,
								DefaultValue:      nil,
							},
							MappedPaths:             [][]string{{"one"}},
							MappedPathModules:       [][]string{{"openconfig-simple"}},
							ShadowMappedPaths:       nil,
							ShadowMappedPathModules: nil,
						},
						"three": {
							Name: "three",
							YANGDetails: ygen.YANGNodeDetails{
								Name:              "three",
								Defaults:          nil,
								BelongingModule:   "openconfig-simple",
								RootElementModule: "openconfig-simple",
								DefiningModule:    "openconfig-simple",
								Path:              "/openconfig-simple/parent/child/config/three",
								SchemaPath:        "/parent/child/config/three",
								LeafrefTargetPath: "",
								Description:       "",
							},
							Type: 3,
							LangType: &ygen.MappedType{
								NativeType:            "Three",
								UnionTypes:            nil,
								IsEnumeratedValue:     true,
								EnumeratedYANGTypeKey: "/openconfig-simple/parent-config/three",
								DefaultValue:          nil,
							},
							MappedPaths:             [][]string{{"three"}},
							MappedPathModules:       [][]string{{"openconfig-simple"}},
							ShadowMappedPaths:       nil,
							ShadowMappedPathModules: nil,
						},
					},
					ListKeys:          nil,
					PackageName:       "openconfig_simple.parent.child",
					BelongingModule:   "openconfig-simple",
					RootElementModule: "openconfig-simple",
					DefiningModule:    "openconfig-simple",
				},
				"/openconfig-simple/parent/child/state": {
					Name:       "State",
					Type:       ygen.Container,
					Path:       "/openconfig-simple/parent/child/state",
					SchemaPath: "/parent/child/state",
					Fields: map[string]*ygen.NodeDetails{
						"four": {
							Name: "four",
							YANGDetails: ygen.YANGNodeDetails{
								Name:              "four",
								Defaults:          nil,
								BelongingModule:   "openconfig-simple",
								RootElementModule: "openconfig-simple",
								DefiningModule:    "openconfig-simple",
								Path:              "/openconfig-simple/parent/child/state/four",
								SchemaPath:        "/parent/child/state/four",
								LeafrefTargetPath: "",
								Description:       "",
								ConfigFalse:       true,
							},
							Type: 3,
							LangType: &ygen.MappedType{
								NativeType:        "ywrapper.BytesValue",
								UnionTypes:        nil,
								IsEnumeratedValue: false,
								ZeroValue:         "",
								DefaultValue:      nil,
							},
							MappedPaths:             [][]string{{"four"}},
							MappedPathModules:       [][]string{{"openconfig-simple"}},
							ShadowMappedPaths:       nil,
							ShadowMappedPathModules: nil,
						},
						"one": {
							Name: "one",
							YANGDetails: ygen.YANGNodeDetails{
								Name:              "one",
								Defaults:          nil,
								BelongingModule:   "openconfig-simple",
								RootElementModule: "openconfig-simple",
								DefiningModule:    "openconfig-simple",
								Path:              "/openconfig-simple/parent/child/state/one",
								SchemaPath:        "/parent/child/state/one",
								LeafrefTargetPath: "",
								Description:       "",
								ConfigFalse:       true,
							},
							Type: 3,
							LangType: &ygen.MappedType{
								NativeType:        "ywrapper.StringValue",
								UnionTypes:        nil,
								IsEnumeratedValue: false,
								ZeroValue:         ``,
								DefaultValue:      nil,
							},
							MappedPaths:             [][]string{{"one"}},
							MappedPathModules:       [][]string{{"openconfig-simple"}},
							ShadowMappedPaths:       nil,
							ShadowMappedPathModules: nil,
						},
						"three": {
							Name: "three",
							YANGDetails: ygen.YANGNodeDetails{
								Name:              "three",
								Defaults:          nil,
								BelongingModule:   "openconfig-simple",
								RootElementModule: "openconfig-simple",
								DefiningModule:    "openconfig-simple",
								Path:              "/openconfig-simple/parent/child/state/three",
								SchemaPath:        "/parent/child/state/three",
								LeafrefTargetPath: "",
								Description:       "",
								ConfigFalse:       true,
							},
							Type: 3,
							LangType: &ygen.MappedType{
								NativeType:            "Three",
								UnionTypes:            nil,
								IsEnumeratedValue:     true,
								EnumeratedYANGTypeKey: "/openconfig-simple/parent-config/three",
								ZeroValue:             "",
								DefaultValue:          nil,
							},
							MappedPaths:             [][]string{{"three"}},
							MappedPathModules:       [][]string{{"openconfig-simple"}},
							ShadowMappedPaths:       nil,
							ShadowMappedPathModules: nil,
						},
						"two": {
							Name: "two",
							YANGDetails: ygen.YANGNodeDetails{
								Name:              "two",
								Defaults:          nil,
								BelongingModule:   "openconfig-simple",
								RootElementModule: "openconfig-simple",
								DefiningModule:    "openconfig-simple",
								Path:              "/openconfig-simple/parent/child/state/two",
								SchemaPath:        "/parent/child/state/two",
								LeafrefTargetPath: "",
								Description:       "",
								ConfigFalse:       true,
							},
							Type: 3,
							LangType: &ygen.MappedType{
								NativeType:        "ywrapper.StringValue",
								UnionTypes:        nil,
								IsEnumeratedValue: false,
								ZeroValue:         ``,
								DefaultValue:      nil,
							},
							MappedPaths:             [][]string{{"two"}},
							MappedPathModules:       [][]string{{"openconfig-simple"}},
							ShadowMappedPaths:       nil,
							ShadowMappedPathModules: nil,
						},
						"zero": {
							Name: "zero",
							YANGDetails: ygen.YANGNodeDetails{
								Name:              "zero",
								Defaults:          nil,
								BelongingModule:   "openconfig-simple-augment2",
								RootElementModule: "openconfig-simple",
								DefiningModule:    "openconfig-simple-grouping",
								Path:              "/openconfig-simple/parent/child/state/zero",
								SchemaPath:        "/parent/child/state/zero",
								LeafrefTargetPath: "",
								Description:       "",
								ConfigFalse:       true,
							},
							Type: 3,
							LangType: &ygen.MappedType{
								NativeType:        "ywrapper.StringValue",
								UnionTypes:        nil,
								IsEnumeratedValue: false,
								ZeroValue:         ``,
								DefaultValue:      nil,
							},
							MappedPaths:             [][]string{{"zero"}},
							MappedPathModules:       [][]string{{"openconfig-simple-augment2"}},
							ShadowMappedPaths:       nil,
							ShadowMappedPathModules: nil,
						},
					},
					ListKeys:          nil,
					PackageName:       "openconfig_simple.parent.child",
					BelongingModule:   "openconfig-simple",
					RootElementModule: "openconfig-simple",
					DefiningModule:    "openconfig-simple",
					ConfigFalse:       true,
				},
				"/openconfig-simple/remote-container": {
					Name:       "RemoteContainer",
					Type:       ygen.Container,
					Path:       "/openconfig-simple/remote-container",
					SchemaPath: "/remote-container",
					Fields: map[string]*ygen.NodeDetails{
						"config": {
							Name: "config",
							YANGDetails: ygen.YANGNodeDetails{
								Name:              "config",
								BelongingModule:   "openconfig-simple",
								RootElementModule: "openconfig-simple",
								DefiningModule:    "openconfig-remote",
								Path:              "/openconfig-simple/remote-container/config",
								SchemaPath:        "/remote-container/config",
								LeafrefTargetPath: "",
								Description:       "",
							},
							Type:              1,
							MappedPaths:       [][]string{{"config"}},
							MappedPathModules: [][]string{{"openconfig-simple"}},
						},
						"state": {
							Name: "state",
							YANGDetails: ygen.YANGNodeDetails{
								Name:              "state",
								BelongingModule:   "openconfig-simple",
								RootElementModule: "openconfig-simple",
								DefiningModule:    "openconfig-remote",
								Path:              "/openconfig-simple/remote-container/state",
								SchemaPath:        "/remote-container/state",
								LeafrefTargetPath: "",
								Description:       "",
								ConfigFalse:       true,
							},
							Type:              1,
							MappedPaths:       [][]string{{"state"}},
							MappedPathModules: [][]string{{"openconfig-simple"}},
						},
					},
					ListKeys:          nil,
					PackageName:       "openconfig_simple",
					BelongingModule:   "openconfig-simple",
					RootElementModule: "openconfig-simple",
					DefiningModule:    "openconfig-remote",
				},
				"/openconfig-simple/remote-container/config": {
					Name:       "Config",
					Type:       ygen.Container,
					Path:       "/openconfig-simple/remote-container/config",
					SchemaPath: "/remote-container/config",
					Fields: map[string]*ygen.NodeDetails{
						"a-leaf": {
							Name: "a_leaf",
							YANGDetails: ygen.YANGNodeDetails{
								Name:              "a-leaf",
								Defaults:          nil,
								BelongingModule:   "openconfig-simple",
								RootElementModule: "openconfig-simple",
								DefiningModule:    "openconfig-remote",
								Path:              "/openconfig-simple/remote-container/config/a-leaf",
								SchemaPath:        "/remote-container/config/a-leaf",
								LeafrefTargetPath: "",
								Description:       "",
							},
							Type: 3,
							LangType: &ygen.MappedType{
								NativeType:        "ywrapper.StringValue",
								UnionTypes:        nil,
								IsEnumeratedValue: false,
								ZeroValue:         ``,
								DefaultValue:      nil,
							},
							MappedPaths:             [][]string{{"a-leaf"}},
							MappedPathModules:       [][]string{{"openconfig-simple"}},
							ShadowMappedPaths:       nil,
							ShadowMappedPathModules: nil,
						},
					},
					PackageName:       "openconfig_simple.remote_container",
					BelongingModule:   "openconfig-simple",
					RootElementModule: "openconfig-simple",
					DefiningModule:    "openconfig-remote",
				},
				"/openconfig-simple/remote-container/state": {
					Name:       "State",
					Type:       ygen.Container,
					Path:       "/openconfig-simple/remote-container/state",
					SchemaPath: "/remote-container/state",
					Fields: map[string]*ygen.NodeDetails{
						"a-leaf": {
							Name: "a_leaf",
							YANGDetails: ygen.YANGNodeDetails{
								Name:              "a-leaf",
								Defaults:          nil,
								BelongingModule:   "openconfig-simple",
								RootElementModule: "openconfig-simple",
								DefiningModule:    "openconfig-remote",
								Path:              "/openconfig-simple/remote-container/state/a-leaf",
								SchemaPath:        "/remote-container/state/a-leaf",
								LeafrefTargetPath: "",
								Description:       "",
								ConfigFalse:       true,
							},
							Type: 3,
							LangType: &ygen.MappedType{
								NativeType:        "ywrapper.StringValue",
								UnionTypes:        nil,
								IsEnumeratedValue: false,
								ZeroValue:         ``,
								DefaultValue:      nil,
							},
							MappedPaths:             [][]string{{"a-leaf"}},
							MappedPathModules:       [][]string{{"openconfig-simple"}},
							ShadowMappedPaths:       nil,
							ShadowMappedPathModules: nil,
						},
					},
					PackageName:       "openconfig_simple.remote_container",
					BelongingModule:   "openconfig-simple",
					RootElementModule: "openconfig-simple",
					DefiningModule:    "openconfig-remote",
					ConfigFalse:       true,
				},
			},
			Enums: map[string]*ygen.EnumeratedYANGType{
				"/openconfig-simple/parent-config/three": {
					Name:     "Simple_Parent_Child_Config_Three",
					Kind:     1,
					TypeName: "enumeration",
					ValToYANGDetails: []ygot.EnumDefinition{{
						Name:  "ONE",
						Value: 0,
					}, {
						Name:  "TWO",
						Value: 1,
					}},
				},
			},
			ModelData: []*gpb.ModelData{{Name: "openconfig-remote"}, {Name: "openconfig-simple"}, {Name: "openconfig-simple-augment2"}, {Name: "openconfig-simple-grouping"}},
		},
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			tt.inOpts.ParseOptions.ExcludeModules = tt.inExcludeModules
			got, err := ygen.GenerateIR(tt.inYANGFiles, tt.inIncludePaths, tt.inLangMapper, tt.inOpts)
			if diff := errdiff.Substring(err, tt.wantErrSubstring); diff != "" {
				t.Fatalf("did not get expected error, %s", diff)
			}
			if diff := cmp.Diff(got, tt.wantIR, cmpopts.IgnoreUnexported(ygen.IR{}, ygen.ParsedDirectory{}, ygen.EnumeratedYANGType{}), protocmp.Transform()); diff != "" {
				t.Fatalf("did not get expected IR, diff(-got,+want):\n%s", diff)
			}
		})
	}
}
