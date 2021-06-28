// Copyright 2019 Google Inc.
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

package ypathgen

import (
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/openconfig/gnmi/errdiff"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/testutil"
	"github.com/openconfig/ygot/ygen"
)

const (
	// TestRoot is the root of the test directory such that this is not
	// repeated when referencing files.
	TestRoot string = ""
	// deflakeRuns specifies the number of runs of code generation that
	// should be performed to check for flakes.
	deflakeRuns int = 10
	// datapath is the path to common YANG test modules.
	datapath = "../testdata/modules"
)

func TestGeneratePathCode(t *testing.T) {
	tests := []struct {
		// Name is the identifier for the test.
		name string
		// inFiles is the set of inputFiles for the test.
		inFiles []string
		// inIncludePaths is the set of paths that should be searched for imports.
		inIncludePaths []string
		// inPreferOperationalState says whether to prefer operational state over intended config in the path-building methods.
		inPreferOperationalState bool
		// inExcludeState determines whether derived state leaves are excluded from the path-building methods.
		inExcludeState bool
		// inListBuilderKeyThreshold determines the minimum number of keys beyond which the builder API is used for building the paths.
		inListBuilderKeyThreshold uint
		// inShortenEnumLeafNames says whether the enum leaf names are shortened (i.e. module name removed) in the generated Go code corresponding to the generated path library.
		inShortenEnumLeafNames bool
		// inUseDefiningModuleForTypedefEnumNames uses the defining module name to prefix typedef enumerated types instead of the module where the typedef enumerated value is used.
		inUseDefiningModuleForTypedefEnumNames bool
		// inGenerateWildcardPaths determines whether wildcard paths are generated.
		inGenerateWildcardPaths bool
		inSchemaStructPkgPath   string
		inPathStructSuffix      string
		inSimplifyWildcardPaths bool
		// checkYANGPath says whether to check for the YANG path in the NodeDataMap.
		checkYANGPath bool
		// wantStructsCodeFile is the path of the generated Go code that the output of the test should be compared to.
		wantStructsCodeFile string
		// wantNodeDataMap is the expected NodeDataMap to be produced to accompany the path struct outputs.
		wantNodeDataMap NodeDataMap
		// wantErr specifies whether the test should expect an error.
		wantErr bool
	}{{
		name:                     "simple openconfig test",
		inFiles:                  []string{filepath.Join(datapath, "openconfig-simple.yang")},
		wantStructsCodeFile:      filepath.Join(TestRoot, "testdata/structs/openconfig-simple.path-txt"),
		inPreferOperationalState: true,
		inShortenEnumLeafNames:   true,
		inGenerateWildcardPaths:  true,
		inSchemaStructPkgPath:    "",
		inPathStructSuffix:       "Path",
		checkYANGPath:            true,
		wantNodeDataMap: NodeDataMap{
			"DevicePath": {
				GoTypeName:            "*Device",
				LocalGoTypeName:       "*Device",
				SubsumingGoStructName: "Device",
				YANGPath:              "/",
			},
			"ParentPath": {
				GoTypeName:            "*Parent",
				LocalGoTypeName:       "*Parent",
				GoFieldName:           "Parent",
				SubsumingGoStructName: "Parent",
				IsLeaf:                false,
				IsScalarField:         false,
				YANGPath:              "/openconfig-simple/parent",
			},
			"Parent_ChildPath": {
				GoTypeName:            "*Parent_Child",
				LocalGoTypeName:       "*Parent_Child",
				GoFieldName:           "Child",
				SubsumingGoStructName: "Parent_Child",
				IsLeaf:                false,
				IsScalarField:         false,
				YANGPath:              "/openconfig-simple/parent/child",
			},
			"Parent_Child_FourPath": {
				GoTypeName:            "Binary",
				LocalGoTypeName:       "Binary",
				GoFieldName:           "Four",
				SubsumingGoStructName: "Parent_Child",
				IsLeaf:                true,
				IsScalarField:         false,
				YANGTypeName:          "binary",
				YANGPath:              "/openconfig-simple/parent/child/state/four",
			},
			"Parent_Child_OnePath": {
				GoTypeName:            "string",
				LocalGoTypeName:       "string",
				GoFieldName:           "One",
				SubsumingGoStructName: "Parent_Child",
				IsLeaf:                true,
				IsScalarField:         true,
				YANGTypeName:          "string",
				YANGPath:              "/openconfig-simple/parent/child/state/one",
			},
			"Parent_Child_ThreePath": {
				GoTypeName:            "E_Child_Three",
				LocalGoTypeName:       "E_Child_Three",
				GoFieldName:           "Three",
				SubsumingGoStructName: "Parent_Child",
				IsLeaf:                true,
				IsScalarField:         false,
				YANGTypeName:          "enumeration",
				YANGPath:              "/openconfig-simple/parent/child/state/three",
			},
			"Parent_Child_TwoPath": {
				GoTypeName:            "string",
				LocalGoTypeName:       "string",
				GoFieldName:           "Two",
				SubsumingGoStructName: "Parent_Child",
				IsLeaf:                true,
				IsScalarField:         true,
				YANGTypeName:          "string",
				YANGPath:              "/openconfig-simple/parent/child/state/two",
			},
			"RemoteContainerPath": {
				GoTypeName:            "*RemoteContainer",
				LocalGoTypeName:       "*RemoteContainer",
				GoFieldName:           "RemoteContainer",
				SubsumingGoStructName: "RemoteContainer",
				IsLeaf:                false,
				IsScalarField:         false,
				YANGPath:              "/openconfig-simple/remote-container",
			},
			"RemoteContainer_ALeafPath": {
				GoTypeName:            "string",
				LocalGoTypeName:       "string",
				GoFieldName:           "ALeaf",
				SubsumingGoStructName: "RemoteContainer",
				IsLeaf:                true,
				IsScalarField:         true,
				YANGTypeName:          "string",
				YANGPath:              "/openconfig-simple/remote-container/state/a-leaf",
			}},
	}, {
		name:                                   "simple openconfig test with preferOperationalState=false",
		inFiles:                                []string{filepath.Join(datapath, "openconfig-simple.yang")},
		inShortenEnumLeafNames:                 true,
		inGenerateWildcardPaths:                true,
		inUseDefiningModuleForTypedefEnumNames: true,
		wantStructsCodeFile:                    filepath.Join(TestRoot, "testdata/structs/openconfig-simple-intendedconfig.path-txt"),
		inSchemaStructPkgPath:                  "",
		inPathStructSuffix:                     "Path",
		wantNodeDataMap: NodeDataMap{
			"DevicePath": {
				GoTypeName:            "*Device",
				LocalGoTypeName:       "*Device",
				SubsumingGoStructName: "Device",
				YANGPath:              "/",
			},
			"ParentPath": {
				GoTypeName:            "*Parent",
				LocalGoTypeName:       "*Parent",
				GoFieldName:           "Parent",
				SubsumingGoStructName: "Parent",
				IsLeaf:                false,
				IsScalarField:         false,
			},
			"Parent_ChildPath": {
				GoTypeName:            "*Parent_Child",
				LocalGoTypeName:       "*Parent_Child",
				GoFieldName:           "Child",
				SubsumingGoStructName: "Parent_Child",
				IsLeaf:                false,
				IsScalarField:         false,
			},
			"Parent_Child_FourPath": {
				GoTypeName:            "Binary",
				LocalGoTypeName:       "Binary",
				GoFieldName:           "Four",
				SubsumingGoStructName: "Parent_Child",
				IsLeaf:                true,
				IsScalarField:         false,
				YANGTypeName:          "binary",
			},
			"Parent_Child_OnePath": {
				GoTypeName:            "string",
				LocalGoTypeName:       "string",
				GoFieldName:           "One",
				SubsumingGoStructName: "Parent_Child",
				IsLeaf:                true,
				IsScalarField:         true,
				YANGTypeName:          "string",
			},
			"Parent_Child_ThreePath": {
				GoTypeName:            "E_Child_Three",
				LocalGoTypeName:       "E_Child_Three",
				GoFieldName:           "Three",
				SubsumingGoStructName: "Parent_Child",
				IsLeaf:                true,
				IsScalarField:         false,
				YANGTypeName:          "enumeration",
			},
			"Parent_Child_TwoPath": {
				GoTypeName:            "string",
				LocalGoTypeName:       "string",
				GoFieldName:           "Two",
				SubsumingGoStructName: "Parent_Child",
				IsLeaf:                true,
				IsScalarField:         true,
				YANGTypeName:          "string",
			},
			"RemoteContainerPath": {
				GoTypeName:            "*RemoteContainer",
				LocalGoTypeName:       "*RemoteContainer",
				GoFieldName:           "RemoteContainer",
				SubsumingGoStructName: "RemoteContainer",
				IsLeaf:                false,
				IsScalarField:         false,
			},
			"RemoteContainer_ALeafPath": {
				GoTypeName:            "string",
				LocalGoTypeName:       "string",
				GoFieldName:           "ALeaf",
				SubsumingGoStructName: "RemoteContainer",
				IsLeaf:                true,
				IsScalarField:         true,
				YANGTypeName:          "string",
			}},
	}, {
		name:                                   "simple openconfig test with excludeState=true",
		inFiles:                                []string{filepath.Join(datapath, "openconfig-simple.yang")},
		inExcludeState:                         true,
		inShortenEnumLeafNames:                 true,
		inUseDefiningModuleForTypedefEnumNames: true,
		inGenerateWildcardPaths:                true,
		wantStructsCodeFile:                    filepath.Join(TestRoot, "testdata/structs/openconfig-simple-excludestate.path-txt"),
		inSchemaStructPkgPath:                  "",
		inPathStructSuffix:                     "Path",
		wantNodeDataMap: NodeDataMap{
			"DevicePath": {
				GoTypeName:            "*Device",
				LocalGoTypeName:       "*Device",
				SubsumingGoStructName: "Device",
				YANGPath:              "/",
			},
			"ParentPath": {
				GoTypeName:            "*Parent",
				LocalGoTypeName:       "*Parent",
				GoFieldName:           "Parent",
				SubsumingGoStructName: "Parent",
				IsLeaf:                false,
				IsScalarField:         false,
			},
			"Parent_ChildPath": {
				GoTypeName:            "*Parent_Child",
				LocalGoTypeName:       "*Parent_Child",
				GoFieldName:           "Child",
				SubsumingGoStructName: "Parent_Child",
				IsLeaf:                false,
				IsScalarField:         false,
			},
			"Parent_Child_FourPath": {
				GoTypeName:            "Binary",
				LocalGoTypeName:       "Binary",
				GoFieldName:           "Four",
				SubsumingGoStructName: "Parent_Child",
				IsLeaf:                true,
				IsScalarField:         false,
				YANGTypeName:          "binary",
			},
			"Parent_Child_OnePath": {
				GoTypeName:            "string",
				LocalGoTypeName:       "string",
				GoFieldName:           "One",
				SubsumingGoStructName: "Parent_Child",
				IsLeaf:                true,
				IsScalarField:         true,
				YANGTypeName:          "string",
			},
			"Parent_Child_ThreePath": {
				GoTypeName:            "E_Child_Three",
				LocalGoTypeName:       "E_Child_Three",
				GoFieldName:           "Three",
				SubsumingGoStructName: "Parent_Child",
				IsLeaf:                true,
				IsScalarField:         false,
				YANGTypeName:          "enumeration",
			},
			"RemoteContainerPath": {
				GoTypeName:            "*RemoteContainer",
				LocalGoTypeName:       "*RemoteContainer",
				GoFieldName:           "RemoteContainer",
				SubsumingGoStructName: "RemoteContainer",
				IsLeaf:                false,
				IsScalarField:         false,
			},
			"RemoteContainer_ALeafPath": {
				GoTypeName:            "string",
				LocalGoTypeName:       "string",
				GoFieldName:           "ALeaf",
				SubsumingGoStructName: "RemoteContainer",
				IsLeaf:                true,
				IsScalarField:         true,
				YANGTypeName:          "string",
			}},
	}, {
		name:                                   "simple openconfig test with list",
		inFiles:                                []string{filepath.Join(datapath, "openconfig-withlist.yang")},
		inPreferOperationalState:               true,
		inShortenEnumLeafNames:                 true,
		inUseDefiningModuleForTypedefEnumNames: true,
		inGenerateWildcardPaths:                true,
		inSchemaStructPkgPath:                  "",
		inPathStructSuffix:                     "Path",
		wantStructsCodeFile:                    filepath.Join(TestRoot, "testdata/structs/openconfig-withlist.path-txt"),
	}, {
		name:                                   "simple openconfig test with list, and inSimplifyWildcardPaths=true",
		inFiles:                                []string{filepath.Join(datapath, "openconfig-withlist.yang")},
		inPreferOperationalState:               true,
		inShortenEnumLeafNames:                 true,
		inUseDefiningModuleForTypedefEnumNames: true,
		inGenerateWildcardPaths:                true,
		inSchemaStructPkgPath:                  "",
		inPathStructSuffix:                     "Path",
		inSimplifyWildcardPaths:                true,
		wantStructsCodeFile:                    filepath.Join(TestRoot, "testdata/structs/openconfig-withlist-simplifyallwc.path-txt"),
	}, {
		name:                                   "simple openconfig test with list without wildcard paths",
		inFiles:                                []string{filepath.Join(datapath, "openconfig-withlist.yang")},
		inPreferOperationalState:               true,
		inShortenEnumLeafNames:                 true,
		inUseDefiningModuleForTypedefEnumNames: true,
		inGenerateWildcardPaths:                false,
		inSchemaStructPkgPath:                  "",
		inPathStructSuffix:                     "Path",
		wantStructsCodeFile:                    filepath.Join(TestRoot, "testdata/structs/openconfig-withlist-nowildcard.path-txt"),
	}, {
		name:                                   "simple openconfig test with list in separate package",
		inFiles:                                []string{filepath.Join(datapath, "openconfig-withlist.yang")},
		inPreferOperationalState:               true,
		inShortenEnumLeafNames:                 true,
		inUseDefiningModuleForTypedefEnumNames: true,
		inGenerateWildcardPaths:                true,
		inSchemaStructPkgPath:                  "github.com/openconfig/ygot/ypathgen/testdata/exampleoc",
		inPathStructSuffix:                     "",
		wantStructsCodeFile:                    filepath.Join(TestRoot, "testdata/structs/openconfig-withlist-separate-package.path-txt"),
	}, {
		name:                                   "simple openconfig test with list in builder API",
		inFiles:                                []string{filepath.Join(datapath, "openconfig-withlist.yang")},
		inListBuilderKeyThreshold:              2,
		inPreferOperationalState:               true,
		inShortenEnumLeafNames:                 true,
		inUseDefiningModuleForTypedefEnumNames: true,
		inGenerateWildcardPaths:                true,
		inSchemaStructPkgPath:                  "",
		inPathStructSuffix:                     "Path",
		wantStructsCodeFile:                    filepath.Join(TestRoot, "testdata/structs/openconfig-withlist-builder.path-txt"),
	}, {
		name:                                   "simple openconfig test with union & typedef & identity & enum",
		inFiles:                                []string{filepath.Join(datapath, "openconfig-unione.yang")},
		inPreferOperationalState:               true,
		inShortenEnumLeafNames:                 true,
		inUseDefiningModuleForTypedefEnumNames: true,
		inGenerateWildcardPaths:                true,
		inSchemaStructPkgPath:                  "",
		inPathStructSuffix:                     "Path",
		wantStructsCodeFile:                    filepath.Join(TestRoot, "testdata/structs/openconfig-unione.path-txt"),
		wantNodeDataMap: NodeDataMap{
			"DevicePath": {
				GoTypeName:            "*Device",
				LocalGoTypeName:       "*Device",
				SubsumingGoStructName: "Device",
				YANGPath:              "/",
			},
			"DupEnumPath": {
				GoTypeName:            "*DupEnum",
				LocalGoTypeName:       "*DupEnum",
				GoFieldName:           "DupEnum",
				SubsumingGoStructName: "DupEnum",
				IsLeaf:                false,
				IsScalarField:         false,
			},
			"DupEnum_APath": {
				GoTypeName:            "E_DupEnum_A",
				LocalGoTypeName:       "E_DupEnum_A",
				GoFieldName:           "A",
				SubsumingGoStructName: "DupEnum",
				IsLeaf:                true,
				IsScalarField:         false,
				YANGTypeName:          "enumeration",
			},
			"DupEnum_BPath": {
				GoTypeName:            "E_DupEnum_B",
				LocalGoTypeName:       "E_DupEnum_B",
				GoFieldName:           "B",
				SubsumingGoStructName: "DupEnum",
				IsLeaf:                true,
				IsScalarField:         false,
				YANGTypeName:          "enumeration",
			},
			"PlatformPath": {
				GoTypeName:            "*Platform",
				LocalGoTypeName:       "*Platform",
				GoFieldName:           "Platform",
				SubsumingGoStructName: "Platform",
				IsLeaf:                false,
				IsScalarField:         false,
			},
			"Platform_ComponentPath": {
				GoTypeName:            "*Platform_Component",
				LocalGoTypeName:       "*Platform_Component",
				GoFieldName:           "Component",
				SubsumingGoStructName: "Platform_Component",
				IsLeaf:                false,
				IsScalarField:         false,
			},
			"Platform_Component_E1Path": {
				GoTypeName:            "Platform_Component_E1_Union",
				LocalGoTypeName:       "Platform_Component_E1_Union",
				GoFieldName:           "E1",
				SubsumingGoStructName: "Platform_Component",
				IsLeaf:                true,
				IsScalarField:         false,
				YANGTypeName:          "enumtypedef",
			},
			"Platform_Component_EnumeratedPath": {
				GoTypeName:            "Platform_Component_Enumerated_Union",
				LocalGoTypeName:       "Platform_Component_Enumerated_Union",
				GoFieldName:           "Enumerated",
				SubsumingGoStructName: "Platform_Component",
				IsLeaf:                true,
				IsScalarField:         false,
				YANGTypeName:          "enumerated-union-type",
			},
			"Platform_Component_PowerPath": {
				GoTypeName:            "Platform_Component_Power_Union",
				LocalGoTypeName:       "Platform_Component_Power_Union",
				GoFieldName:           "Power",
				SubsumingGoStructName: "Platform_Component",
				IsLeaf:                true,
				IsScalarField:         false,
				YANGTypeName:          "union",
			},
			"Platform_Component_R1Path": {
				GoTypeName:            "Platform_Component_E1_Union",
				LocalGoTypeName:       "Platform_Component_E1_Union",
				GoFieldName:           "R1",
				SubsumingGoStructName: "Platform_Component",
				IsLeaf:                true,
				IsScalarField:         false,
				YANGTypeName:          "leafref",
			},
			"Platform_Component_TypePath": {
				GoTypeName:            "Platform_Component_Type_Union",
				LocalGoTypeName:       "Platform_Component_Type_Union",
				GoFieldName:           "Type",
				SubsumingGoStructName: "Platform_Component",
				IsLeaf:                true,
				IsScalarField:         false,
				YANGTypeName:          "union",
			}},
	}, {
		name:                     "simple openconfig test with union & typedef & identity & enum, with enum names not shortened",
		inFiles:                  []string{filepath.Join(datapath, "openconfig-unione.yang")},
		inPreferOperationalState: true,
		inGenerateWildcardPaths:  true,
		inSchemaStructPkgPath:    "",
		inPathStructSuffix:       "Path",
		wantStructsCodeFile:      filepath.Join(TestRoot, "testdata/structs/openconfig-unione.path-txt"),
		wantNodeDataMap: NodeDataMap{
			"DevicePath": {
				GoTypeName:            "*Device",
				LocalGoTypeName:       "*Device",
				SubsumingGoStructName: "Device",
				YANGPath:              "/",
			},
			"DupEnumPath": {
				GoTypeName:            "*DupEnum",
				LocalGoTypeName:       "*DupEnum",
				GoFieldName:           "DupEnum",
				SubsumingGoStructName: "DupEnum",
				IsLeaf:                false,
				IsScalarField:         false,
			},
			"DupEnum_APath": {
				GoTypeName:            "E_OpenconfigUnione_DupEnum_A",
				LocalGoTypeName:       "E_OpenconfigUnione_DupEnum_A",
				GoFieldName:           "A",
				SubsumingGoStructName: "DupEnum",
				IsLeaf:                true,
				IsScalarField:         false,
				YANGTypeName:          "enumeration",
			},
			"DupEnum_BPath": {
				GoTypeName:            "E_OpenconfigUnione_DupEnum_B",
				LocalGoTypeName:       "E_OpenconfigUnione_DupEnum_B",
				GoFieldName:           "B",
				SubsumingGoStructName: "DupEnum",
				IsLeaf:                true,
				IsScalarField:         false,
				YANGTypeName:          "enumeration",
			},
			"PlatformPath": {
				GoTypeName:            "*Platform",
				LocalGoTypeName:       "*Platform",
				GoFieldName:           "Platform",
				SubsumingGoStructName: "Platform",
				IsLeaf:                false,
				IsScalarField:         false,
			},
			"Platform_ComponentPath": {
				GoTypeName:            "*Platform_Component",
				LocalGoTypeName:       "*Platform_Component",
				GoFieldName:           "Component",
				SubsumingGoStructName: "Platform_Component",
				IsLeaf:                false,
				IsScalarField:         false,
			},
			"Platform_Component_E1Path": {
				GoTypeName:            "Platform_Component_E1_Union",
				LocalGoTypeName:       "Platform_Component_E1_Union",
				GoFieldName:           "E1",
				SubsumingGoStructName: "Platform_Component",
				IsLeaf:                true,
				IsScalarField:         false,
				YANGTypeName:          "enumtypedef",
			},
			"Platform_Component_EnumeratedPath": {
				GoTypeName:            "Platform_Component_Enumerated_Union",
				LocalGoTypeName:       "Platform_Component_Enumerated_Union",
				GoFieldName:           "Enumerated",
				SubsumingGoStructName: "Platform_Component",
				IsLeaf:                true,
				IsScalarField:         false,
				YANGTypeName:          "enumerated-union-type",
			},
			"Platform_Component_PowerPath": {
				GoTypeName:            "Platform_Component_Power_Union",
				LocalGoTypeName:       "Platform_Component_Power_Union",
				GoFieldName:           "Power",
				SubsumingGoStructName: "Platform_Component",
				IsLeaf:                true,
				IsScalarField:         false,
				YANGTypeName:          "union",
			},
			"Platform_Component_R1Path": {
				GoTypeName:            "Platform_Component_E1_Union",
				LocalGoTypeName:       "Platform_Component_E1_Union",
				GoFieldName:           "R1",
				SubsumingGoStructName: "Platform_Component",
				IsLeaf:                true,
				IsScalarField:         false,
				YANGTypeName:          "leafref",
			},
			"Platform_Component_TypePath": {
				GoTypeName:            "Platform_Component_Type_Union",
				LocalGoTypeName:       "Platform_Component_Type_Union",
				GoFieldName:           "Type",
				SubsumingGoStructName: "Platform_Component",
				IsLeaf:                true,
				IsScalarField:         false,
				YANGTypeName:          "union",
			}},
	}, {
		name:                                   "simple openconfig test with submodule and union list key",
		inFiles:                                []string{filepath.Join(datapath, "enum-module.yang")},
		inPreferOperationalState:               true,
		inShortenEnumLeafNames:                 true,
		inUseDefiningModuleForTypedefEnumNames: true,
		inGenerateWildcardPaths:                true,
		inSchemaStructPkgPath:                  "",
		inPathStructSuffix:                     "Path",
		wantStructsCodeFile:                    filepath.Join(TestRoot, "testdata/structs/enum-module.path-txt"),
		wantNodeDataMap: NodeDataMap{
			"DevicePath": {
				GoTypeName:            "*Device",
				LocalGoTypeName:       "*Device",
				SubsumingGoStructName: "Device",
				YANGPath:              "/",
			},
			"AListPath": {
				GoTypeName:            "*AList",
				LocalGoTypeName:       "*AList",
				GoFieldName:           "AList",
				SubsumingGoStructName: "AList",
				IsLeaf:                false,
				IsScalarField:         false,
			},
			"AList_ValuePath": {
				GoTypeName:            "AList_Value_Union",
				LocalGoTypeName:       "AList_Value_Union",
				GoFieldName:           "Value",
				SubsumingGoStructName: "AList",
				IsLeaf:                true,
				IsScalarField:         false,
				YANGTypeName:          "td",
			},
			"BListPath": {
				GoTypeName:            "*BList",
				LocalGoTypeName:       "*BList",
				GoFieldName:           "BList",
				SubsumingGoStructName: "BList",
				IsLeaf:                false,
				IsScalarField:         false,
			},
			"BList_ValuePath": {
				GoTypeName:            "BList_Value_Union",
				LocalGoTypeName:       "BList_Value_Union",
				GoFieldName:           "Value",
				SubsumingGoStructName: "BList",
				IsLeaf:                true,
				IsScalarField:         false,
				YANGTypeName:          "td",
			},
			"CPath": {
				GoTypeName:            "*C",
				LocalGoTypeName:       "*C",
				GoFieldName:           "C",
				SubsumingGoStructName: "C",
				IsLeaf:                false,
				IsScalarField:         false,
			},
			"C_ClPath": {
				GoTypeName:            "E_EnumModule_Cl",
				LocalGoTypeName:       "E_EnumModule_Cl",
				GoFieldName:           "Cl",
				SubsumingGoStructName: "C",
				IsLeaf:                true,
				IsScalarField:         false,
				YANGTypeName:          "enumeration",
			},
			"ParentPath": {
				GoTypeName:            "*Parent",
				LocalGoTypeName:       "*Parent",
				GoFieldName:           "Parent",
				SubsumingGoStructName: "Parent",
				IsLeaf:                false,
				IsScalarField:         false,
			},
			"Parent_ChildPath": {
				GoTypeName:            "*Parent_Child",
				LocalGoTypeName:       "*Parent_Child",
				GoFieldName:           "Child",
				SubsumingGoStructName: "Parent_Child",
				IsLeaf:                false,
				IsScalarField:         false,
			},
			"Parent_Child_IdPath": {
				GoTypeName:            "E_EnumTypes_ID",
				LocalGoTypeName:       "E_EnumTypes_ID",
				GoFieldName:           "Id",
				SubsumingGoStructName: "Parent_Child",
				IsLeaf:                true,
				IsScalarField:         false,
				YANGTypeName:          "identityref",
			},
			"Parent_Child_Id2Path": {
				GoTypeName:            "E_EnumTypes_ID",
				LocalGoTypeName:       "E_EnumTypes_ID",
				GoFieldName:           "Id2",
				SubsumingGoStructName: "Parent_Child",
				IsLeaf:                true,
				IsScalarField:         false,
				YANGTypeName:          "identityref",
			},
			"Parent_Child_EnumPath": {
				GoTypeName:            "E_EnumTypes_TdEnum",
				LocalGoTypeName:       "E_EnumTypes_TdEnum",
				GoFieldName:           "Enum",
				SubsumingGoStructName: "Parent_Child",
				IsLeaf:                true,
				IsScalarField:         false,
				YANGTypeName:          "td-enum",
			},
			"Parent_Child_InlineEnumPath": {
				GoTypeName:            "E_Child_InlineEnum",
				LocalGoTypeName:       "E_Child_InlineEnum",
				GoFieldName:           "InlineEnum",
				SubsumingGoStructName: "Parent_Child",
				IsLeaf:                true,
				IsScalarField:         false,
				YANGTypeName:          "enumeration",
			}},
	}, {
		name:                                   "simple openconfig test with choice and cases",
		inFiles:                                []string{filepath.Join(datapath, "choice-case-example.yang")},
		inPreferOperationalState:               true,
		inShortenEnumLeafNames:                 true,
		inUseDefiningModuleForTypedefEnumNames: true,
		inGenerateWildcardPaths:                true,
		inSchemaStructPkgPath:                  "",
		inPathStructSuffix:                     "Path",
		wantStructsCodeFile:                    filepath.Join(TestRoot, "testdata/structs/choice-case-example.path-txt"),
	}, {
		name: "simple openconfig test with augmentations",
		inFiles: []string{
			filepath.Join(datapath, "openconfig-simple-target.yang"),
			filepath.Join(datapath, "openconfig-simple-augment.yang"),
		},
		inPreferOperationalState:               true,
		inShortenEnumLeafNames:                 true,
		inUseDefiningModuleForTypedefEnumNames: true,
		inGenerateWildcardPaths:                true,
		inSchemaStructPkgPath:                  "github.com/openconfig/ygot/ypathgen/testdata/exampleoc",
		inPathStructSuffix:                     "",
		wantStructsCodeFile:                    filepath.Join(TestRoot, "testdata/structs/openconfig-augmented.path-txt"),
		wantNodeDataMap: NodeDataMap{
			"Device": {
				GoTypeName:            "*oc.Device",
				LocalGoTypeName:       "*Device",
				SubsumingGoStructName: "Device",
				YANGPath:              "/",
			},
			"Native": {
				GoTypeName:            "*oc.Native",
				LocalGoTypeName:       "*Native",
				GoFieldName:           "Native",
				SubsumingGoStructName: "Native",
				IsLeaf:                false,
				IsScalarField:         false,
			},
			"Native_A": {
				GoTypeName:            "string",
				LocalGoTypeName:       "string",
				GoFieldName:           "A",
				SubsumingGoStructName: "Native",
				IsLeaf:                true,
				IsScalarField:         true,
				YANGTypeName:          "string",
			},
			"Target": {
				GoTypeName:            "*oc.Target",
				LocalGoTypeName:       "*Target",
				GoFieldName:           "Target",
				SubsumingGoStructName: "Target",
				IsLeaf:                false,
				IsScalarField:         false,
			},
			"Target_Foo": {
				GoTypeName:            "*oc.Target_Foo",
				LocalGoTypeName:       "*Target_Foo",
				GoFieldName:           "Foo",
				SubsumingGoStructName: "Target_Foo",
				IsLeaf:                false,
				IsScalarField:         false,
			},
			"Target_Foo_A": {
				GoTypeName:            "string",
				LocalGoTypeName:       "string",
				GoFieldName:           "A",
				SubsumingGoStructName: "Target_Foo",
				IsLeaf:                true,
				IsScalarField:         true,
				YANGTypeName:          "string",
			}},
	}, {
		name:                                   "simple openconfig test with camelcase-name extension",
		inFiles:                                []string{filepath.Join(datapath, "openconfig-enumcamelcase.yang")},
		inPreferOperationalState:               true,
		inShortenEnumLeafNames:                 true,
		inUseDefiningModuleForTypedefEnumNames: true,
		inGenerateWildcardPaths:                true,
		inSchemaStructPkgPath:                  "",
		inPathStructSuffix:                     "Path",
		wantStructsCodeFile:                    filepath.Join(TestRoot, "testdata/structs/openconfig-enumcamelcase.path-txt"),
	}, {
		name:                                   "simple openconfig test with camelcase-name extension in container and leaf",
		inFiles:                                []string{filepath.Join(datapath, "openconfig-camelcase.yang")},
		inPreferOperationalState:               true,
		inShortenEnumLeafNames:                 true,
		inUseDefiningModuleForTypedefEnumNames: true,
		inGenerateWildcardPaths:                true,
		inSchemaStructPkgPath:                  "",
		inPathStructSuffix:                     "Path",
		wantStructsCodeFile:                    filepath.Join(TestRoot, "testdata/structs/openconfig-camelcase.path-txt"),
	}}

	for _, tt := range tests {
		t.Run(tt.name+":"+strings.Join(tt.inFiles, ","), func(t *testing.T) {
			genCode := func() (string, NodeDataMap, *GenConfig) {
				cg := NewDefaultConfig(tt.inSchemaStructPkgPath)
				// Set the name of the caller explicitly to avoid issues when
				// the unit tests are called by external test entities.
				cg.GeneratingBinary = "pathgen-tests"
				cg.FakeRootName = "device"
				cg.PathStructSuffix = tt.inPathStructSuffix
				cg.PreferOperationalState = tt.inPreferOperationalState
				cg.ExcludeState = tt.inExcludeState
				cg.ListBuilderKeyThreshold = tt.inListBuilderKeyThreshold
				cg.ShortenEnumLeafNames = tt.inShortenEnumLeafNames
				cg.UseDefiningModuleForTypedefEnumNames = tt.inUseDefiningModuleForTypedefEnumNames
				cg.GenerateWildcardPaths = tt.inGenerateWildcardPaths
				cg.SimplifyWildcardPaths = tt.inSimplifyWildcardPaths

				gotCode, gotNodeDataMap, err := cg.GeneratePathCode(tt.inFiles, tt.inIncludePaths)
				if err != nil && !tt.wantErr {
					t.Fatalf("GeneratePathCode(%v, %v): Config: %v, got unexpected error: %v, want: nil", tt.inFiles, tt.inIncludePaths, cg, err)
				}

				return gotCode.String(), gotNodeDataMap, cg
			}

			gotCode, gotNodeDataMap, cg := genCode()

			if tt.wantNodeDataMap != nil {
				var cmpOpts []cmp.Option
				if !tt.checkYANGPath {
					cmpOpts = append(cmpOpts, cmpopts.IgnoreFields(NodeData{}, "YANGPath"))
				}
				if diff := cmp.Diff(tt.wantNodeDataMap, gotNodeDataMap, cmpOpts...); diff != "" {
					t.Errorf("(-wantNodeDataMap, +gotNodeDataMap):\n%s", diff)
				}
			}

			wantCodeBytes, rferr := ioutil.ReadFile(tt.wantStructsCodeFile)
			if rferr != nil {
				t.Fatalf("ioutil.ReadFile(%q) error: %v", tt.wantStructsCodeFile, rferr)
			}

			wantCode := string(wantCodeBytes)

			if gotCode != wantCode {
				// Use difflib to generate a unified diff between the
				// two code snippets such that this is simpler to debug
				// in the test output.
				diff, _ := testutil.GenerateUnifiedDiff(wantCode, gotCode)
				t.Errorf("GeneratePathCode(%v, %v), Config: %v, did not return correct code (file: %v), diff:\n%s",
					tt.inFiles, tt.inIncludePaths, cg, tt.wantStructsCodeFile, diff)
			}

			for i := 0; i < deflakeRuns; i++ {
				gotAttempt, _, _ := genCode()
				if gotAttempt != gotCode {
					diff, _ := testutil.GenerateUnifiedDiff(gotAttempt, gotCode)
					t.Fatalf("flaky code generation, diff:\n%s", diff)
				}
			}
		})
	}
}

func TestGeneratePathCodeSplitFiles(t *testing.T) {
	tests := []struct {
		name                  string   // Name is the identifier for the test.
		inFiles               []string // inFiles is the set of inputFiles for the test.
		inIncludePaths        []string // inIncludePaths is the set of paths that should be searched for imports.
		inFileNumber          int      // inFileNumber is the number of files into which to split the generated code.
		inSchemaStructPkgPath string
		wantStructsCodeFiles  []string // wantStructsCodeFiles is the paths of the generated Go code that the output of the test should be compared to.
		wantErr               bool     // whether an error is expected from the SplitFiles call
	}{{
		name:                  "fileNumber is higher than total number of structs",
		inFiles:               []string{filepath.Join(datapath, "openconfig-simple.yang")},
		inFileNumber:          5,
		inSchemaStructPkgPath: "",
		wantErr:               true,
	}, {
		name:                  "fileNumber is exactly the total number of structs",
		inFiles:               []string{filepath.Join(datapath, "openconfig-simple.yang")},
		inFileNumber:          4,
		inSchemaStructPkgPath: "github.com/openconfig/ygot/ypathgen/testdata/exampleoc",
		wantStructsCodeFiles:  []string{filepath.Join(TestRoot, "testdata/structs/openconfig-simple-40.path-txt"), filepath.Join(TestRoot, "testdata/structs/openconfig-simple-41.path-txt"), filepath.Join(TestRoot, "testdata/structs/openconfig-simple-42.path-txt"), filepath.Join(TestRoot, "testdata/structs/openconfig-simple-43.path-txt")},
	}, {
		name:                  "fileNumber is just under the total number of structs",
		inFiles:               []string{filepath.Join(datapath, "openconfig-simple.yang")},
		inFileNumber:          3,
		inSchemaStructPkgPath: "",
		wantStructsCodeFiles:  []string{filepath.Join(TestRoot, "testdata/structs/openconfig-simple-30.path-txt"), filepath.Join(TestRoot, "testdata/structs/openconfig-simple-31.path-txt"), filepath.Join(TestRoot, "testdata/structs/openconfig-simple-32.path-txt")},
	}, {
		name:                  "fileNumber is half the total number of structs",
		inFiles:               []string{filepath.Join(datapath, "openconfig-simple.yang")},
		inFileNumber:          2,
		inSchemaStructPkgPath: "github.com/openconfig/ygot/ypathgen/testdata/exampleoc",
		wantStructsCodeFiles:  []string{filepath.Join(TestRoot, "testdata/structs/openconfig-simple-0.path-txt"), filepath.Join(TestRoot, "testdata/structs/openconfig-simple-1.path-txt")},
	}, {
		name:                  "single file",
		inFiles:               []string{filepath.Join(datapath, "openconfig-simple.yang")},
		inFileNumber:          1,
		inSchemaStructPkgPath: "",
		wantStructsCodeFiles:  []string{filepath.Join(TestRoot, "testdata/structs/openconfig-simple.path-txt")},
	}, {
		name:         "fileNumber is 0",
		inFiles:      []string{filepath.Join(datapath, "openconfig-simple.yang")},
		inFileNumber: 0,
		wantErr:      true,
	}}

	for _, tt := range tests {
		t.Run(tt.name+":"+strings.Join(tt.inFiles, ","), func(t *testing.T) {
			genCode := func() ([]string, *GenConfig) {
				cg := NewDefaultConfig(tt.inSchemaStructPkgPath)
				// Set the name of the caller explicitly to avoid issues when
				// the unit tests are called by external test entities.
				cg.GeneratingBinary = "pathgen-tests"
				cg.FakeRootName = "device"
				if tt.inSchemaStructPkgPath == "" {
					cg.PathStructSuffix = "Path"
				} else {
					cg.PathStructSuffix = ""
				}
				cg.PreferOperationalState = true
				cg.GenerateWildcardPaths = true

				gotCode, _, err := cg.GeneratePathCode(tt.inFiles, tt.inIncludePaths)
				if err != nil {
					t.Fatalf("GeneratePathCode(%v, %v): Config: %v, got unexpected error: %v", tt.inFiles, tt.inIncludePaths, cg, err)
				}

				files, e := gotCode.SplitFiles(tt.inFileNumber)
				if e != nil && !tt.wantErr {
					t.Fatalf("SplitFiles(%v): got unexpected error: %v", tt.inFileNumber, e)
				} else if e == nil && tt.wantErr {
					t.Fatalf("SplitFiles(%v): did not get expected error", tt.inFileNumber)
				}

				return files, cg
			}

			gotCode, cg := genCode()

			var wantCode []string
			for _, codeFile := range tt.wantStructsCodeFiles {
				wantCodeBytes, rferr := ioutil.ReadFile(codeFile)
				if rferr != nil {
					t.Fatalf("ioutil.ReadFile(%q) error: %v", tt.wantStructsCodeFiles, rferr)
				}
				wantCode = append(wantCode, string(wantCodeBytes))
			}

			if len(gotCode) != len(wantCode) {
				t.Errorf("GeneratePathCode(%v, %v), Config: %v, did not return correct code via SplitFiles function (files: %v), (gotfiles: %d, wantfiles: %d), diff (-want, +got):\n%s",
					tt.inFiles, tt.inIncludePaths, cg, tt.wantStructsCodeFiles, len(gotCode), len(wantCode), cmp.Diff(wantCode, gotCode))
			} else {
				for i := range gotCode {
					if gotCode[i] != wantCode[i] {
						// Use difflib to generate a unified diff between the
						// two code snippets such that this is simpler to debug
						// in the test output.
						diff, _ := testutil.GenerateUnifiedDiff(wantCode[i], gotCode[i])
						t.Errorf("GeneratePathCode(%v, %v), Config: %v, did not return correct code via SplitFiles function (file: %v), diff:\n%s",
							tt.inFiles, tt.inIncludePaths, cg, tt.wantStructsCodeFiles[i], diff)
					}
				}
			}
		})
	}
}

// addParents adds parent pointers for a schema tree.
func addParents(e *yang.Entry) {
	for _, c := range e.Dir {
		c.Parent = e
		addParents(c)
	}
}

// getSchemaAndDirs is a helper returning a module tree to be tested, and its
// corresponding Directory map with relevant fields filled out that would be
// returned from ygen.GetDirectories().
func getSchemaAndDirs() (*yang.Entry, map[string]*ygen.Directory, map[string]map[string]*ygen.MappedType) {
	schema := &yang.Entry{
		Name: "root-module",
		Kind: yang.DirectoryEntry,
		Dir: map[string]*yang.Entry{
			"leaf": {
				Name: "leaf",
				Kind: yang.LeafEntry,
				// Name is given here to test setting the YANGTypeName field.
				Type: &yang.YangType{Name: "ieeefloat32", Kind: yang.Ybinary},
			},
			"container": {
				Name: "container",
				Kind: yang.DirectoryEntry,
				Dir: map[string]*yang.Entry{
					"leaf": {
						Name: "leaf",
						Kind: yang.LeafEntry,
						Type: &yang.YangType{Name: "int32", Kind: yang.Yint32},
					},
				},
			},
			"container-with-config": {
				Name: "container-with-config",
				Kind: yang.DirectoryEntry,
				Dir: map[string]*yang.Entry{
					"config": {
						Name: "config",
						Kind: yang.DirectoryEntry,
						Dir: map[string]*yang.Entry{
							"leaf": {
								Name: "leaf",
								Kind: yang.LeafEntry,
								Type: &yang.YangType{Kind: yang.Ybinary},
							},
						},
					},
					"state": {
						Name:   "state",
						Kind:   yang.DirectoryEntry,
						Config: yang.TSFalse,
						Dir: map[string]*yang.Entry{
							"leaf": {
								Name: "leaf",
								Kind: yang.LeafEntry,
								Type: &yang.YangType{Kind: yang.Ybinary},
							},
							"leaflist": {
								Name:     "leaflist",
								Kind:     yang.LeafEntry,
								ListAttr: &yang.ListAttr{},
								Type:     &yang.YangType{Kind: yang.Yuint32},
							},
							"leaflist2": {
								Name:     "leaflist2",
								Kind:     yang.LeafEntry,
								ListAttr: &yang.ListAttr{},
								Type:     &yang.YangType{Kind: yang.Ybinary},
							},
						},
					},
				},
			},
			"list-container": {
				Name: "list-container",
				Kind: yang.DirectoryEntry,
				Dir: map[string]*yang.Entry{
					"list": {
						Name:     "list",
						Kind:     yang.DirectoryEntry,
						ListAttr: &yang.ListAttr{},
						Dir: map[string]*yang.Entry{
							"key1": {
								Name: "key1",
								Kind: yang.LeafEntry,
								Type: &yang.YangType{Kind: yang.Ystring},
							},
							"key2": {
								Name: "key2",
								Kind: yang.LeafEntry,
								Type: &yang.YangType{Kind: yang.Ybinary},
							},
							"union-key": {
								Name: "union-key",
								Type: &yang.YangType{
									Kind: yang.Yunion,
									Type: []*yang.YangType{{
										Name: "enumeration",
										Kind: yang.Yenum,
										Enum: &yang.EnumType{},
									}, {
										Kind: yang.Yuint32,
									}},
								},
							},
						},
					},
				},
			},
			"list-container-with-state": {
				Name: "list-container-with-state",
				Kind: yang.DirectoryEntry,
				Dir: map[string]*yang.Entry{
					"list-with-state": {
						Name:     "list-with-state",
						Kind:     yang.DirectoryEntry,
						ListAttr: &yang.ListAttr{},
						Dir: map[string]*yang.Entry{
							"key": {
								Name: "key",
								Kind: yang.LeafEntry,
								Type: &yang.YangType{
									Kind: yang.Yleafref,
									Path: "../state/key",
								},
							},
							"state": {
								Name: "state",
								Kind: yang.DirectoryEntry,
								Dir: map[string]*yang.Entry{
									"key": {
										Name: "key",
										Kind: yang.LeafEntry,
										Type: &yang.YangType{Kind: yang.Ydecimal64},
									},
								},
							},
						},
					},
				},
			},
		},
		Annotation: map[string]interface{}{"isCompressedSchema": true},
	}
	addParents(schema)

	// Build fake root.
	fakeRoot := ygen.MakeFakeRoot("root")
	for k, v := range schema.Dir {
		fakeRoot.Dir[k] = v
	}

	directories := map[string]*ygen.Directory{
		"/root": {
			Name: "Root",
			Fields: map[string]*yang.Entry{
				"leaf":                  schema.Dir["leaf"],
				"container":             schema.Dir["container"],
				"container-with-config": schema.Dir["container-with-config"],
				"list":                  schema.Dir["list-container"].Dir["list"],
				"list-with-state":       schema.Dir["list-container-with-state"].Dir["list-with-state"],
			},
			Path:  []string{"", "root"},
			Entry: fakeRoot,
		},
		"/root-module/container": {
			Name: "Container",
			Fields: map[string]*yang.Entry{
				"leaf": schema.Dir["container"].Dir["leaf"],
			},
			Path:  []string{"", "root-module", "container"},
			Entry: schema.Dir["container"],
		},
		"/root-module/container-with-config": {
			Name: "ContainerWithConfig",
			Fields: map[string]*yang.Entry{
				"leaf":      schema.Dir["container-with-config"].Dir["state"].Dir["leaf"],
				"leaflist":  schema.Dir["container-with-config"].Dir["state"].Dir["leaflist"],
				"leaflist2": schema.Dir["container-with-config"].Dir["state"].Dir["leaflist2"],
			},
			Path:  []string{"", "root-module", "container-with-config"},
			Entry: schema.Dir["container-with-config"],
		},
		"/root-module/list-container/list": {
			Name: "List",
			ListAttr: &ygen.YangListAttr{
				Keys: map[string]*ygen.MappedType{
					"key1":      {NativeType: "string"},
					"key2":      {NativeType: "Binary"},
					"union-key": {NativeType: "RootModule_List_UnionKey_Union", UnionTypes: map[string]int{"string": 0, "Binary": 1}},
				},
				KeyElems: []*yang.Entry{{Name: "key1"}, {Name: "key2"}, {Name: "union-key"}},
			},
			Fields: map[string]*yang.Entry{
				"key1":      schema.Dir["list-container"].Dir["list"].Dir["key1"],
				"key2":      schema.Dir["list-container"].Dir["list"].Dir["key2"],
				"union-key": schema.Dir["list-container"].Dir["list"].Dir["union-key"],
			},
			Path:  []string{"", "root-module", "list-container", "list"},
			Entry: schema.Dir["list-container"],
		},
		"/root-module/list-container-with-state/list-with-state": {
			Name: "ListWithState",
			ListAttr: &ygen.YangListAttr{
				Keys: map[string]*ygen.MappedType{
					"key": {NativeType: "float64"},
				},
				KeyElems: []*yang.Entry{{Name: "key"}},
			},
			Fields: map[string]*yang.Entry{
				"key": schema.Dir["list-container-with-state"].Dir["list-with-state"].Dir["key"],
			},
			Path:  []string{"", "root-module", "list-container-with-state", "list-with-state"},
			Entry: schema.Dir["list-container-with-state"],
		},
	}

	leafTypeMap := map[string]map[string]*ygen.MappedType{
		"/root": {
			"leaf":                  {NativeType: "Binary"},
			"container":             nil,
			"container-with-config": nil,
			"list":                  nil,
			"list-with-state":       nil,
		},
		"/root-module/container": {
			"leaf": {NativeType: "int32"},
		},
		"/root-module/container-with-config": {
			"leaf":      {NativeType: "Binary"},
			"leaflist":  {NativeType: "uint32"},
			"leaflist2": {NativeType: "Binary"},
		},
		"/root-module/list-container/list": {
			"key1":      {NativeType: "string"},
			"key2":      {NativeType: "Binary"},
			"union-key": {NativeType: "RootModule_List_UnionKey_Union", UnionTypes: map[string]int{"string": 0, "Binary": 1}},
		},
		"/root-module/list-container-with-state/list-with-state": {
			"key": {NativeType: "float64"},
		},
	}

	return schema, directories, leafTypeMap
}

// wantListMethodsNonWildcard is the expected non-wildcard child constructor
// method for the test list node.
const wantListMethodsNonWildcard = `
// List returns from RootPath the path struct for its child "list".
// Key1: string
// Key2: oc.Binary
// UnionKey: [oc.UnionString, oc.Binary]
func (n *RootPath) List(Key1 string, Key2 oc.Binary, UnionKey oc.RootModule_List_UnionKey_Union) *ListPath {
	return &ListPath{
		NodePath: ygot.NewNodePath(
			[]string{"list-container", "list"},
			map[string]interface{}{"key1": Key1, "key2": Key2, "union-key": UnionKey},
			n,
		),
	}
}
`

const wantListMethodsWildcardCommon = `
// ListAnyKey2AnyUnionKey returns from RootPath the path struct for its child "list".
// Key1: string
func (n *RootPath) ListAnyKey2AnyUnionKey(Key1 string) *ListPathAny {
	return &ListPathAny{
		NodePath: ygot.NewNodePath(
			[]string{"list-container", "list"},
			map[string]interface{}{"key1": Key1, "key2": "*", "union-key": "*"},
			n,
		),
	}
}

// ListAnyKey1AnyUnionKey returns from RootPath the path struct for its child "list".
// Key2: oc.Binary
func (n *RootPath) ListAnyKey1AnyUnionKey(Key2 oc.Binary) *ListPathAny {
	return &ListPathAny{
		NodePath: ygot.NewNodePath(
			[]string{"list-container", "list"},
			map[string]interface{}{"key1": "*", "key2": Key2, "union-key": "*"},
			n,
		),
	}
}

// ListAnyUnionKey returns from RootPath the path struct for its child "list".
// Key1: string
// Key2: oc.Binary
func (n *RootPath) ListAnyUnionKey(Key1 string, Key2 oc.Binary) *ListPathAny {
	return &ListPathAny{
		NodePath: ygot.NewNodePath(
			[]string{"list-container", "list"},
			map[string]interface{}{"key1": Key1, "key2": Key2, "union-key": "*"},
			n,
		),
	}
}

// ListAnyKey1AnyKey2 returns from RootPath the path struct for its child "list".
// UnionKey: [oc.UnionString, oc.Binary]
func (n *RootPath) ListAnyKey1AnyKey2(UnionKey oc.RootModule_List_UnionKey_Union) *ListPathAny {
	return &ListPathAny{
		NodePath: ygot.NewNodePath(
			[]string{"list-container", "list"},
			map[string]interface{}{"key1": "*", "key2": "*", "union-key": UnionKey},
			n,
		),
	}
}

// ListAnyKey2 returns from RootPath the path struct for its child "list".
// Key1: string
// UnionKey: [oc.UnionString, oc.Binary]
func (n *RootPath) ListAnyKey2(Key1 string, UnionKey oc.RootModule_List_UnionKey_Union) *ListPathAny {
	return &ListPathAny{
		NodePath: ygot.NewNodePath(
			[]string{"list-container", "list"},
			map[string]interface{}{"key1": Key1, "key2": "*", "union-key": UnionKey},
			n,
		),
	}
}

// ListAnyKey1 returns from RootPath the path struct for its child "list".
// Key2: oc.Binary
// UnionKey: [oc.UnionString, oc.Binary]
func (n *RootPath) ListAnyKey1(Key2 oc.Binary, UnionKey oc.RootModule_List_UnionKey_Union) *ListPathAny {
	return &ListPathAny{
		NodePath: ygot.NewNodePath(
			[]string{"list-container", "list"},
			map[string]interface{}{"key1": "*", "key2": Key2, "union-key": UnionKey},
			n,
		),
	}
}
`

// wantListMethods is the expected child constructor methods for the test list node.
const wantListMethods = `
// ListAny returns from RootPath the path struct for its child "list".
func (n *RootPath) ListAny() *ListPathAny {
	return &ListPathAny{
		NodePath: ygot.NewNodePath(
			[]string{"list-container", "list"},
			map[string]interface{}{"key1": "*", "key2": "*", "union-key": "*"},
			n,
		),
	}
}
` + wantListMethodsWildcardCommon + wantListMethodsNonWildcard

// wantListMethodsSimplified is the expected child constructor methods for
// the test list node when SimplifyWildcardPaths=true.
const wantListMethodsSimplified = `
// ListAny returns from RootPath the path struct for its child "list".
func (n *RootPath) ListAny() *ListPathAny {
	return &ListPathAny{
		NodePath: ygot.NewNodePath(
			[]string{"list-container", "list"},
			map[string]interface{}{},
			n,
		),
	}
}
` + wantListMethodsWildcardCommon + wantListMethodsNonWildcard

func TestGetNodeDataMap(t *testing.T) {
	_, directories, leafTypeMap := getSchemaAndDirs()

	schema2 := &yang.Entry{
		Name: "root-module",
		Kind: yang.DirectoryEntry,
		Dir: map[string]*yang.Entry{
			"container": {
				Name: "container",
				Kind: yang.DirectoryEntry,
				Dir: map[string]*yang.Entry{
					"leaf": {
						Name: "leaf",
						Kind: yang.LeafEntry,
						Type: &yang.YangType{Kind: yang.Ybinary},
					},
				},
			},
		},
	}
	addParents(schema2)
	binaryContainerEntry := schema2.Dir["container"]

	fakeRoot := ygen.MakeFakeRoot("root")
	fakeRoot.Dir["container"] = binaryContainerEntry

	directoryWithBinaryLeaf := map[string]*ygen.Directory{
		"/root": {
			Name: "Root",
			Fields: map[string]*yang.Entry{
				"container": binaryContainerEntry,
			},
			Path:  []string{"", "root"},
			Entry: fakeRoot,
		},
		"/root-module/container": {
			Name: "Container",
			Fields: map[string]*yang.Entry{
				"leaf": binaryContainerEntry.Dir["leaf"],
			},
			Path:  []string{"", "root-module", "container"},
			Entry: binaryContainerEntry,
		},
	}

	leafTypeMap2 := map[string]map[string]*ygen.MappedType{
		"/root": {
			"container": nil,
		},
		"/root-module/container": {
			"leaf": {NativeType: "Binary"},
		},
	}

	tests := []struct {
		name                      string
		inDirectories             map[string]*ygen.Directory
		inLeafTypeMap             map[string]map[string]*ygen.MappedType
		inSchemaStructPkgAccessor string
		inPathStructSuffix        string
		wantNodeDataMap           NodeDataMap
		wantSorted                []string
		wantErrSubstrings         []string
	}{{
		name:          "scalar leaf",
		inDirectories: map[string]*ygen.Directory{"/root-module/container": directories["/root-module/container"]},
		inLeafTypeMap: map[string]map[string]*ygen.MappedType{
			"/root-module/container": {
				"leaf": leafTypeMap["/root-module/container"]["leaf"],
			},
		},
		inSchemaStructPkgAccessor: "struct.",
		inPathStructSuffix:        "Path",
		wantNodeDataMap: NodeDataMap{
			"Container_LeafPath": {
				GoTypeName:            "int32",
				LocalGoTypeName:       "int32",
				GoFieldName:           "Leaf",
				SubsumingGoStructName: "Container",
				IsLeaf:                true,
				IsScalarField:         true,
				YANGTypeName:          "int32",
			},
		},
		wantSorted: []string{"Container_LeafPath"},
	}, {
		name:                      "non-leaf and non-scalar leaf",
		inDirectories:             directoryWithBinaryLeaf,
		inLeafTypeMap:             leafTypeMap2,
		inSchemaStructPkgAccessor: "struct.",
		inPathStructSuffix:        "_Path",
		wantNodeDataMap: NodeDataMap{
			"Container_Path": {
				GoTypeName:            "*struct.Container",
				LocalGoTypeName:       "*Container",
				GoFieldName:           "Container",
				SubsumingGoStructName: "Container",
				IsLeaf:                false,
				IsScalarField:         false,
			},
			"Container_Leaf_Path": {
				GoTypeName:            "struct.Binary",
				LocalGoTypeName:       "Binary",
				GoFieldName:           "Leaf",
				SubsumingGoStructName: "Container",
				IsLeaf:                true,
				IsScalarField:         false,
			},
		},
		wantSorted: []string{"Container_Leaf_Path", "Container_Path"},
	}, {
		name:          "non-existent path",
		inDirectories: map[string]*ygen.Directory{"/root-module/container": directories["/root-module/container"]},
		inLeafTypeMap: map[string]map[string]*ygen.MappedType{
			"/root": {
				"container": nil,
			},
			"/you can't find me": {
				"leaf": {NativeType: "Binary"},
			},
		},
		inSchemaStructPkgAccessor: "oc.",
		inPathStructSuffix:        "Path",
		wantErrSubstrings:         []string{`path "/root-module/container" does not exist`},
	}, {
		name:          "non-existent field",
		inDirectories: map[string]*ygen.Directory{"/root-module/container": directories["/root-module/container"]},
		inLeafTypeMap: map[string]map[string]*ygen.MappedType{
			"/root": {
				"container": nil,
			},
			"/root-module/container": {
				"laugh": leafTypeMap["/root-module/container"]["leaf"],
			},
		},
		inSchemaStructPkgAccessor: "oc.",
		inPathStructSuffix:        "Path",
		wantErrSubstrings:         []string{`field name "leaf" does not exist`},
	}, {
		name:               "big test with everything",
		inDirectories:      directories,
		inLeafTypeMap:      leafTypeMap,
		inPathStructSuffix: "Path",
		wantNodeDataMap: NodeDataMap{
			"ContainerPath": {
				GoTypeName:            "*Container",
				LocalGoTypeName:       "*Container",
				GoFieldName:           "Container",
				SubsumingGoStructName: "Container",
				IsLeaf:                false,
				IsScalarField:         false,
			},
			"ContainerWithConfigPath": {
				GoTypeName:            "*ContainerWithConfig",
				LocalGoTypeName:       "*ContainerWithConfig",
				GoFieldName:           "ContainerWithConfig",
				SubsumingGoStructName: "ContainerWithConfig",
				IsLeaf:                false,
				IsScalarField:         false,
			},
			"ContainerWithConfig_LeafPath": {
				GoTypeName:            "Binary",
				LocalGoTypeName:       "Binary",
				GoFieldName:           "Leaf",
				SubsumingGoStructName: "ContainerWithConfig",
				IsLeaf:                true,
				IsScalarField:         false,
			},
			"ContainerWithConfig_LeaflistPath": {
				GoTypeName:            "[]uint32",
				LocalGoTypeName:       "[]uint32",
				GoFieldName:           "Leaflist",
				SubsumingGoStructName: "ContainerWithConfig",
				IsLeaf:                true,
				IsScalarField:         false,
			},
			"ContainerWithConfig_Leaflist2Path": {
				GoTypeName:            "[]Binary",
				LocalGoTypeName:       "[]Binary",
				GoFieldName:           "Leaflist2",
				SubsumingGoStructName: "ContainerWithConfig",
				IsLeaf:                true,
				IsScalarField:         false,
			},
			"Container_LeafPath": {
				GoTypeName:            "int32",
				LocalGoTypeName:       "int32",
				GoFieldName:           "Leaf",
				SubsumingGoStructName: "Container",
				IsLeaf:                true,
				IsScalarField:         true,
				YANGTypeName:          "int32",
			},
			"LeafPath": {
				GoTypeName:            "Binary",
				LocalGoTypeName:       "Binary",
				GoFieldName:           "Leaf",
				SubsumingGoStructName: "Root",
				IsLeaf:                true,
				IsScalarField:         false,
				YANGTypeName:          "ieeefloat32",
			},
			"ListPath": {
				GoTypeName:            "*List",
				LocalGoTypeName:       "*List",
				GoFieldName:           "List",
				SubsumingGoStructName: "List",
				IsLeaf:                false,
				IsScalarField:         false,
			},
			"ListWithStatePath": {
				GoTypeName:            "*ListWithState",
				LocalGoTypeName:       "*ListWithState",
				GoFieldName:           "ListWithState",
				SubsumingGoStructName: "ListWithState",
				IsLeaf:                false,
				IsScalarField:         false,
			},
			"ListWithState_KeyPath": {
				GoTypeName:            "float64",
				LocalGoTypeName:       "float64",
				GoFieldName:           "Key",
				SubsumingGoStructName: "ListWithState",
				IsLeaf:                true,
				IsScalarField:         true,
			},
			"List_Key1Path": {
				GoTypeName:            "string",
				LocalGoTypeName:       "string",
				GoFieldName:           "Key1",
				SubsumingGoStructName: "List",
				IsLeaf:                true,
				IsScalarField:         true,
			},
			"List_Key2Path": {
				GoTypeName:            "Binary",
				LocalGoTypeName:       "Binary",
				GoFieldName:           "Key2",
				SubsumingGoStructName: "List",
				IsLeaf:                true,
				IsScalarField:         false,
			},
			"List_UnionKeyPath": {
				GoTypeName:            "RootModule_List_UnionKey_Union",
				LocalGoTypeName:       "RootModule_List_UnionKey_Union",
				GoFieldName:           "UnionKey",
				SubsumingGoStructName: "List",
				IsLeaf:                true,
				IsScalarField:         false,
			}},
		wantSorted: []string{
			"ContainerPath",
			"ContainerWithConfigPath",
			"ContainerWithConfig_LeafPath",
			"ContainerWithConfig_Leaflist2Path",
			"ContainerWithConfig_LeaflistPath",
			"Container_LeafPath",
			"LeafPath",
			"ListPath",
			"ListWithStatePath",
			"ListWithState_KeyPath",
			"List_Key1Path",
			"List_Key2Path",
			"List_UnionKeyPath",
		},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErrs := getNodeDataMap(tt.inDirectories, tt.inLeafTypeMap, tt.inSchemaStructPkgAccessor, tt.inPathStructSuffix)
			// TODO(wenbli): Enhance gNMI's errdiff with checking a slice of substrings and use here.
			var gotErrStrs []string
			for _, err := range gotErrs {
				gotErrStrs = append(gotErrStrs, err.Error())
			}
			if diff := cmp.Diff(tt.wantErrSubstrings, gotErrStrs, cmp.Comparer(func(x, y string) bool { return strings.Contains(x, y) || strings.Contains(y, x) })); diff != "" {
				t.Fatalf("Error substring check failed (-want, +got):\n%v", diff)
			}
			if diff := cmp.Diff(tt.wantNodeDataMap, got, cmpopts.IgnoreFields(NodeData{}, "YANGPath")); diff != "" {
				t.Errorf("(-want, +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.wantSorted, GetOrderedNodeDataNames(got), cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("(-want sorted names, +got sorted names):\n%s", diff)
			}
		})
	}
}

func TestGenerateDirectorySnippet(t *testing.T) {
	_, directories, _ := getSchemaAndDirs()

	tests := []struct {
		name                      string
		inDirectory               *ygen.Directory
		inListBuilderKeyThreshold uint
		inPathStructSuffix        string
		want                      GoPathStructCodeSnippet
		wantNoWildcard            GoPathStructCodeSnippet
	}{{
		name:        "container-with-config",
		inDirectory: directories["/root-module/container-with-config"],
		want: GoPathStructCodeSnippet{
			PathStructName: "ContainerWithConfig",
			StructBase: `
// ContainerWithConfig represents the /root-module/container-with-config YANG schema element.
type ContainerWithConfig struct {
	*ygot.NodePath
}

// ContainerWithConfigAny represents the wildcard version of the /root-module/container-with-config YANG schema element.
type ContainerWithConfigAny struct {
	*ygot.NodePath
}

// ContainerWithConfig_Leaf represents the /root-module/container-with-config/state/leaf YANG schema element.
type ContainerWithConfig_Leaf struct {
	*ygot.NodePath
}

// ContainerWithConfig_LeafAny represents the wildcard version of the /root-module/container-with-config/state/leaf YANG schema element.
type ContainerWithConfig_LeafAny struct {
	*ygot.NodePath
}

// ContainerWithConfig_Leaflist represents the /root-module/container-with-config/state/leaflist YANG schema element.
type ContainerWithConfig_Leaflist struct {
	*ygot.NodePath
}

// ContainerWithConfig_LeaflistAny represents the wildcard version of the /root-module/container-with-config/state/leaflist YANG schema element.
type ContainerWithConfig_LeaflistAny struct {
	*ygot.NodePath
}

// ContainerWithConfig_Leaflist2 represents the /root-module/container-with-config/state/leaflist2 YANG schema element.
type ContainerWithConfig_Leaflist2 struct {
	*ygot.NodePath
}

// ContainerWithConfig_Leaflist2Any represents the wildcard version of the /root-module/container-with-config/state/leaflist2 YANG schema element.
type ContainerWithConfig_Leaflist2Any struct {
	*ygot.NodePath
}
`,
			ChildConstructors: `
// Leaf returns from ContainerWithConfig the path struct for its child "leaf".
func (n *ContainerWithConfig) Leaf() *ContainerWithConfig_Leaf {
	return &ContainerWithConfig_Leaf{
		NodePath: ygot.NewNodePath(
			[]string{"state", "leaf"},
			map[string]interface{}{},
			n,
		),
	}
}

// Leaf returns from ContainerWithConfigAny the path struct for its child "leaf".
func (n *ContainerWithConfigAny) Leaf() *ContainerWithConfig_LeafAny {
	return &ContainerWithConfig_LeafAny{
		NodePath: ygot.NewNodePath(
			[]string{"state", "leaf"},
			map[string]interface{}{},
			n,
		),
	}
}

// Leaflist returns from ContainerWithConfig the path struct for its child "leaflist".
func (n *ContainerWithConfig) Leaflist() *ContainerWithConfig_Leaflist {
	return &ContainerWithConfig_Leaflist{
		NodePath: ygot.NewNodePath(
			[]string{"state", "leaflist"},
			map[string]interface{}{},
			n,
		),
	}
}

// Leaflist returns from ContainerWithConfigAny the path struct for its child "leaflist".
func (n *ContainerWithConfigAny) Leaflist() *ContainerWithConfig_LeaflistAny {
	return &ContainerWithConfig_LeaflistAny{
		NodePath: ygot.NewNodePath(
			[]string{"state", "leaflist"},
			map[string]interface{}{},
			n,
		),
	}
}

// Leaflist2 returns from ContainerWithConfig the path struct for its child "leaflist2".
func (n *ContainerWithConfig) Leaflist2() *ContainerWithConfig_Leaflist2 {
	return &ContainerWithConfig_Leaflist2{
		NodePath: ygot.NewNodePath(
			[]string{"state", "leaflist2"},
			map[string]interface{}{},
			n,
		),
	}
}

// Leaflist2 returns from ContainerWithConfigAny the path struct for its child "leaflist2".
func (n *ContainerWithConfigAny) Leaflist2() *ContainerWithConfig_Leaflist2Any {
	return &ContainerWithConfig_Leaflist2Any{
		NodePath: ygot.NewNodePath(
			[]string{"state", "leaflist2"},
			map[string]interface{}{},
			n,
		),
	}
}
`,
		},
		wantNoWildcard: GoPathStructCodeSnippet{
			PathStructName: "ContainerWithConfig",
			StructBase: `
// ContainerWithConfig represents the /root-module/container-with-config YANG schema element.
type ContainerWithConfig struct {
	*ygot.NodePath
}

// ContainerWithConfig_Leaf represents the /root-module/container-with-config/state/leaf YANG schema element.
type ContainerWithConfig_Leaf struct {
	*ygot.NodePath
}

// ContainerWithConfig_Leaflist represents the /root-module/container-with-config/state/leaflist YANG schema element.
type ContainerWithConfig_Leaflist struct {
	*ygot.NodePath
}

// ContainerWithConfig_Leaflist2 represents the /root-module/container-with-config/state/leaflist2 YANG schema element.
type ContainerWithConfig_Leaflist2 struct {
	*ygot.NodePath
}
`,
			ChildConstructors: `
// Leaf returns from ContainerWithConfig the path struct for its child "leaf".
func (n *ContainerWithConfig) Leaf() *ContainerWithConfig_Leaf {
	return &ContainerWithConfig_Leaf{
		NodePath: ygot.NewNodePath(
			[]string{"state", "leaf"},
			map[string]interface{}{},
			n,
		),
	}
}

// Leaflist returns from ContainerWithConfig the path struct for its child "leaflist".
func (n *ContainerWithConfig) Leaflist() *ContainerWithConfig_Leaflist {
	return &ContainerWithConfig_Leaflist{
		NodePath: ygot.NewNodePath(
			[]string{"state", "leaflist"},
			map[string]interface{}{},
			n,
		),
	}
}

// Leaflist2 returns from ContainerWithConfig the path struct for its child "leaflist2".
func (n *ContainerWithConfig) Leaflist2() *ContainerWithConfig_Leaflist2 {
	return &ContainerWithConfig_Leaflist2{
		NodePath: ygot.NewNodePath(
			[]string{"state", "leaflist2"},
			map[string]interface{}{},
			n,
		),
	}
}
`,
		},
	}, {
		name:               "fakeroot",
		inDirectory:        directories["/root"],
		inPathStructSuffix: "Path",
		want: GoPathStructCodeSnippet{
			PathStructName: "RootPath",
			StructBase: `
// RootPath represents the /root YANG schema element.
type RootPath struct {
	*ygot.DeviceRootBase
}

// DeviceRoot returns a new path object from which YANG paths can be constructed.
func DeviceRoot(id string) *RootPath {
	return &RootPath{ygot.NewDeviceRootBase(id)}
}

// LeafPath represents the /root-module/leaf YANG schema element.
type LeafPath struct {
	*ygot.NodePath
}

// LeafPathAny represents the wildcard version of the /root-module/leaf YANG schema element.
type LeafPathAny struct {
	*ygot.NodePath
}
`,
			ChildConstructors: `
// Container returns from RootPath the path struct for its child "container".
func (n *RootPath) Container() *ContainerPath {
	return &ContainerPath{
		NodePath: ygot.NewNodePath(
			[]string{"container"},
			map[string]interface{}{},
			n,
		),
	}
}

// ContainerWithConfig returns from RootPath the path struct for its child "container-with-config".
func (n *RootPath) ContainerWithConfig() *ContainerWithConfigPath {
	return &ContainerWithConfigPath{
		NodePath: ygot.NewNodePath(
			[]string{"container-with-config"},
			map[string]interface{}{},
			n,
		),
	}
}

// Leaf returns from RootPath the path struct for its child "leaf".
func (n *RootPath) Leaf() *LeafPath {
	return &LeafPath{
		NodePath: ygot.NewNodePath(
			[]string{"leaf"},
			map[string]interface{}{},
			n,
		),
	}
}
` + wantListMethods + `
// ListWithStateAny returns from RootPath the path struct for its child "list-with-state".
func (n *RootPath) ListWithStateAny() *ListWithStatePathAny {
	return &ListWithStatePathAny{
		NodePath: ygot.NewNodePath(
			[]string{"list-container-with-state", "list-with-state"},
			map[string]interface{}{"key": "*"},
			n,
		),
	}
}

// ListWithState returns from RootPath the path struct for its child "list-with-state".
// Key: float64
func (n *RootPath) ListWithState(Key float64) *ListWithStatePath {
	return &ListWithStatePath{
		NodePath: ygot.NewNodePath(
			[]string{"list-container-with-state", "list-with-state"},
			map[string]interface{}{"key": Key},
			n,
		),
	}
}
`,
		},
		wantNoWildcard: GoPathStructCodeSnippet{
			PathStructName: "RootPath",
			StructBase: `
// RootPath represents the /root YANG schema element.
type RootPath struct {
	*ygot.DeviceRootBase
}

// DeviceRoot returns a new path object from which YANG paths can be constructed.
func DeviceRoot(id string) *RootPath {
	return &RootPath{ygot.NewDeviceRootBase(id)}
}

// LeafPath represents the /root-module/leaf YANG schema element.
type LeafPath struct {
	*ygot.NodePath
}
`,
			ChildConstructors: `
// Container returns from RootPath the path struct for its child "container".
func (n *RootPath) Container() *ContainerPath {
	return &ContainerPath{
		NodePath: ygot.NewNodePath(
			[]string{"container"},
			map[string]interface{}{},
			n,
		),
	}
}

// ContainerWithConfig returns from RootPath the path struct for its child "container-with-config".
func (n *RootPath) ContainerWithConfig() *ContainerWithConfigPath {
	return &ContainerWithConfigPath{
		NodePath: ygot.NewNodePath(
			[]string{"container-with-config"},
			map[string]interface{}{},
			n,
		),
	}
}

// Leaf returns from RootPath the path struct for its child "leaf".
func (n *RootPath) Leaf() *LeafPath {
	return &LeafPath{
		NodePath: ygot.NewNodePath(
			[]string{"leaf"},
			map[string]interface{}{},
			n,
		),
	}
}
` + wantListMethodsNonWildcard + `
// ListWithState returns from RootPath the path struct for its child "list-with-state".
// Key: float64
func (n *RootPath) ListWithState(Key float64) *ListWithStatePath {
	return &ListWithStatePath{
		NodePath: ygot.NewNodePath(
			[]string{"list-container-with-state", "list-with-state"},
			map[string]interface{}{"key": Key},
			n,
		),
	}
}
`,
		},
	}, {
		name:        "list",
		inDirectory: directories["/root-module/list-container/list"],
		want: GoPathStructCodeSnippet{
			PathStructName: "List",
			StructBase: `
// List represents the /root-module/list-container/list YANG schema element.
type List struct {
	*ygot.NodePath
}

// ListAny represents the wildcard version of the /root-module/list-container/list YANG schema element.
type ListAny struct {
	*ygot.NodePath
}

// List_Key1 represents the /root-module/list-container/list/key1 YANG schema element.
type List_Key1 struct {
	*ygot.NodePath
}

// List_Key1Any represents the wildcard version of the /root-module/list-container/list/key1 YANG schema element.
type List_Key1Any struct {
	*ygot.NodePath
}

// List_Key2 represents the /root-module/list-container/list/key2 YANG schema element.
type List_Key2 struct {
	*ygot.NodePath
}

// List_Key2Any represents the wildcard version of the /root-module/list-container/list/key2 YANG schema element.
type List_Key2Any struct {
	*ygot.NodePath
}

// List_UnionKey represents the /root-module/list-container/list/union-key YANG schema element.
type List_UnionKey struct {
	*ygot.NodePath
}

// List_UnionKeyAny represents the wildcard version of the /root-module/list-container/list/union-key YANG schema element.
type List_UnionKeyAny struct {
	*ygot.NodePath
}
`,
			ChildConstructors: `
// Key1 returns from List the path struct for its child "key1".
func (n *List) Key1() *List_Key1 {
	return &List_Key1{
		NodePath: ygot.NewNodePath(
			[]string{"key1"},
			map[string]interface{}{},
			n,
		),
	}
}

// Key1 returns from ListAny the path struct for its child "key1".
func (n *ListAny) Key1() *List_Key1Any {
	return &List_Key1Any{
		NodePath: ygot.NewNodePath(
			[]string{"key1"},
			map[string]interface{}{},
			n,
		),
	}
}

// Key2 returns from List the path struct for its child "key2".
func (n *List) Key2() *List_Key2 {
	return &List_Key2{
		NodePath: ygot.NewNodePath(
			[]string{"key2"},
			map[string]interface{}{},
			n,
		),
	}
}

// Key2 returns from ListAny the path struct for its child "key2".
func (n *ListAny) Key2() *List_Key2Any {
	return &List_Key2Any{
		NodePath: ygot.NewNodePath(
			[]string{"key2"},
			map[string]interface{}{},
			n,
		),
	}
}

// UnionKey returns from List the path struct for its child "union-key".
func (n *List) UnionKey() *List_UnionKey {
	return &List_UnionKey{
		NodePath: ygot.NewNodePath(
			[]string{"union-key"},
			map[string]interface{}{},
			n,
		),
	}
}

// UnionKey returns from ListAny the path struct for its child "union-key".
func (n *ListAny) UnionKey() *List_UnionKeyAny {
	return &List_UnionKeyAny{
		NodePath: ygot.NewNodePath(
			[]string{"union-key"},
			map[string]interface{}{},
			n,
		),
	}
}
`,
		},
		wantNoWildcard: GoPathStructCodeSnippet{
			PathStructName: "List",
			StructBase: `
// List represents the /root-module/list-container/list YANG schema element.
type List struct {
	*ygot.NodePath
}

// List_Key1 represents the /root-module/list-container/list/key1 YANG schema element.
type List_Key1 struct {
	*ygot.NodePath
}

// List_Key2 represents the /root-module/list-container/list/key2 YANG schema element.
type List_Key2 struct {
	*ygot.NodePath
}

// List_UnionKey represents the /root-module/list-container/list/union-key YANG schema element.
type List_UnionKey struct {
	*ygot.NodePath
}
`,
			ChildConstructors: `
// Key1 returns from List the path struct for its child "key1".
func (n *List) Key1() *List_Key1 {
	return &List_Key1{
		NodePath: ygot.NewNodePath(
			[]string{"key1"},
			map[string]interface{}{},
			n,
		),
	}
}

// Key2 returns from List the path struct for its child "key2".
func (n *List) Key2() *List_Key2 {
	return &List_Key2{
		NodePath: ygot.NewNodePath(
			[]string{"key2"},
			map[string]interface{}{},
			n,
		),
	}
}

// UnionKey returns from List the path struct for its child "union-key".
func (n *List) UnionKey() *List_UnionKey {
	return &List_UnionKey{
		NodePath: ygot.NewNodePath(
			[]string{"union-key"},
			map[string]interface{}{},
			n,
		),
	}
}
`,
		},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := generateDirectorySnippet(tt.inDirectory, directories, "oc.", tt.inPathStructSuffix, tt.inListBuilderKeyThreshold, true, false)
			if gotErr != nil {
				t.Fatalf("func generateDirectorySnippet, unexpected error: %v", gotErr)
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("func generateDirectorySnippet mismatch (-want, +got):\n%s", diff)
			}
		})

		t.Run(tt.name+" no wildcard", func(t *testing.T) {
			got, gotErr := generateDirectorySnippet(tt.inDirectory, directories, "oc.", tt.inPathStructSuffix, tt.inListBuilderKeyThreshold, false, false)
			if gotErr != nil {
				t.Fatalf("func generateDirectorySnippet, unexpected error: %v", gotErr)
			}

			if diff := cmp.Diff(tt.wantNoWildcard, got); diff != "" {
				t.Errorf("func generateDirectorySnippet mismatch (-want, +got):\n%s", diff)
			}
		})
	}
}

func TestGenerateChildConstructor(t *testing.T) {
	_, directories, _ := getSchemaAndDirs()

	deepSchema := &yang.Entry{
		Name: "root-module",
		Kind: yang.DirectoryEntry,
		Dir: map[string]*yang.Entry{
			"container": {
				Name: "container",
				Kind: yang.DirectoryEntry,
				Dir: map[string]*yang.Entry{
					"leaf": {
						Name: "leaf",
						Kind: yang.LeafEntry,
						Type: &yang.YangType{Kind: yang.Yint32},
					},
					"list-container": {
						Name: "list-container",
						Kind: yang.DirectoryEntry,
						Dir: map[string]*yang.Entry{
							"list": {
								Name:     "list",
								Kind:     yang.DirectoryEntry,
								ListAttr: &yang.ListAttr{},
								Dir: map[string]*yang.Entry{
									"key": {
										Name: "key",
										Kind: yang.LeafEntry,
										Type: &yang.YangType{Kind: yang.Ystring},
									},
								},
							},
						},
					},
					"inner-container": {
						Name: "inner-container",
						Kind: yang.DirectoryEntry,
						Dir: map[string]*yang.Entry{
							"inner-leaf": {
								Name: "inner-leaf",
								Kind: yang.LeafEntry,
								Type: &yang.YangType{Kind: yang.Yint32},
							},
						},
					},
				},
			},
		},
		Annotation: map[string]interface{}{"isCompressedSchema": true},
	}
	addParents(deepSchema)

	// Build fake root.
	fakeRoot := ygen.MakeFakeRoot("root")
	for k, v := range deepSchema.Dir {
		fakeRoot.Dir[k] = v
	}

	deepSchemaDirectories := map[string]*ygen.Directory{
		"/root": {
			Name: "Root",
			Fields: map[string]*yang.Entry{
				"container": deepSchema.Dir["container"],
			},
			Path:  []string{"", "root"},
			Entry: fakeRoot,
		},
		"/root-module/container": {
			Name: "Container",
			Fields: map[string]*yang.Entry{
				"list":            deepSchema.Dir["container"].Dir["list-container"].Dir["list"],
				"inner-container": deepSchema.Dir["container"].Dir["inner-container"],
			},
			Path:  []string{"", "root-module", "container"},
			Entry: deepSchema.Dir["container"],
		},
		"/root-module/container/list-container/list": {
			Name: "Container_List",
			ListAttr: &ygen.YangListAttr{
				Keys: map[string]*ygen.MappedType{
					"key": {NativeType: "string"},
				},
				KeyElems: []*yang.Entry{{Name: "key"}},
			},
			Fields: map[string]*yang.Entry{
				"key": deepSchema.Dir["container"].Dir["list-container"].Dir["list"].Dir["key"],
			},
			Path:  []string{"", "root-module", "container", "list-container", "list"},
			Entry: deepSchema.Dir["container"].Dir["list-container"],
		},
		"/root-module/container/inner-container": {
			Name: "Container_InnerContainer",
			Fields: map[string]*yang.Entry{
				"leaf": deepSchema.Dir["container"].Dir["inner-container"].Dir["leaf"],
			},
			Path:  []string{"", "root-module", "container", "inner-container"},
			Entry: deepSchema.Dir["container"].Dir["inner-container"],
		},
	}

	tests := []struct {
		name                      string
		inDirectory               *ygen.Directory
		inDirectories             map[string]*ygen.Directory
		inFieldName               string
		inUniqueFieldName         string
		inListBuilderKeyThreshold uint
		inPathStructSuffix        string
		inGenerateWildcardPaths   bool
		inSimplifyWildcardPaths   bool
		want                      string
	}{{
		name:                    "container method",
		inDirectory:             directories["/root"],
		inDirectories:           directories,
		inFieldName:             "container",
		inUniqueFieldName:       "Container",
		inPathStructSuffix:      "Path",
		inGenerateWildcardPaths: true,
		want: `
// Container returns from RootPath the path struct for its child "container".
func (n *RootPath) Container() *ContainerPath {
	return &ContainerPath{
		NodePath: ygot.NewNodePath(
			[]string{"container"},
			map[string]interface{}{},
			n,
		),
	}
}
`,
	}, {
		name:                    "container leaf method",
		inDirectory:             directories["/root-module/container"],
		inDirectories:           directories,
		inFieldName:             "leaf",
		inUniqueFieldName:       "Leaf",
		inPathStructSuffix:      "Path",
		inGenerateWildcardPaths: true,
		want: `
// Leaf returns from ContainerPath the path struct for its child "leaf".
func (n *ContainerPath) Leaf() *Container_LeafPath {
	return &Container_LeafPath{
		NodePath: ygot.NewNodePath(
			[]string{"leaf"},
			map[string]interface{}{},
			n,
		),
	}
}

// Leaf returns from ContainerPathAny the path struct for its child "leaf".
func (n *ContainerPathAny) Leaf() *Container_LeafPathAny {
	return &Container_LeafPathAny{
		NodePath: ygot.NewNodePath(
			[]string{"leaf"},
			map[string]interface{}{},
			n,
		),
	}
}
`,
	}, {
		name:                    "container leaf method without wildcard paths",
		inDirectory:             directories["/root-module/container"],
		inDirectories:           directories,
		inFieldName:             "leaf",
		inUniqueFieldName:       "Leaf",
		inPathStructSuffix:      "Path",
		inGenerateWildcardPaths: false,
		want: `
// Leaf returns from ContainerPath the path struct for its child "leaf".
func (n *ContainerPath) Leaf() *Container_LeafPath {
	return &Container_LeafPath{
		NodePath: ygot.NewNodePath(
			[]string{"leaf"},
			map[string]interface{}{},
			n,
		),
	}
}
`,
	}, {
		name:                    "top-level leaf method",
		inDirectory:             directories["/root"],
		inDirectories:           directories,
		inFieldName:             "leaf",
		inUniqueFieldName:       "Leaf",
		inPathStructSuffix:      "Path",
		inGenerateWildcardPaths: true,
		want: `
// Leaf returns from RootPath the path struct for its child "leaf".
func (n *RootPath) Leaf() *LeafPath {
	return &LeafPath{
		NodePath: ygot.NewNodePath(
			[]string{"leaf"},
			map[string]interface{}{},
			n,
		),
	}
}
`,
	}, {
		name:                    "container-with-config leaf method",
		inDirectory:             directories["/root-module/container-with-config"],
		inDirectories:           directories,
		inFieldName:             "leaf",
		inUniqueFieldName:       "Leaf",
		inPathStructSuffix:      "Path",
		inGenerateWildcardPaths: true,
		want: `
// Leaf returns from ContainerWithConfigPath the path struct for its child "leaf".
func (n *ContainerWithConfigPath) Leaf() *ContainerWithConfig_LeafPath {
	return &ContainerWithConfig_LeafPath{
		NodePath: ygot.NewNodePath(
			[]string{"state", "leaf"},
			map[string]interface{}{},
			n,
		),
	}
}

// Leaf returns from ContainerWithConfigPathAny the path struct for its child "leaf".
func (n *ContainerWithConfigPathAny) Leaf() *ContainerWithConfig_LeafPathAny {
	return &ContainerWithConfig_LeafPathAny{
		NodePath: ygot.NewNodePath(
			[]string{"state", "leaf"},
			map[string]interface{}{},
			n,
		),
	}
}
`,
	}, {
		name:                    "2nd-level list methods",
		inDirectory:             deepSchemaDirectories["/root-module/container"],
		inDirectories:           deepSchemaDirectories,
		inFieldName:             "list",
		inUniqueFieldName:       "List",
		inPathStructSuffix:      "Path",
		inGenerateWildcardPaths: true,
		want: `
// ListAny returns from ContainerPath the path struct for its child "list".
func (n *ContainerPath) ListAny() *Container_ListPathAny {
	return &Container_ListPathAny{
		NodePath: ygot.NewNodePath(
			[]string{"list-container", "list"},
			map[string]interface{}{"key": "*"},
			n,
		),
	}
}

// ListAny returns from ContainerPathAny the path struct for its child "list".
func (n *ContainerPathAny) ListAny() *Container_ListPathAny {
	return &Container_ListPathAny{
		NodePath: ygot.NewNodePath(
			[]string{"list-container", "list"},
			map[string]interface{}{"key": "*"},
			n,
		),
	}
}

// List returns from ContainerPath the path struct for its child "list".
// Key: string
func (n *ContainerPath) List(Key string) *Container_ListPath {
	return &Container_ListPath{
		NodePath: ygot.NewNodePath(
			[]string{"list-container", "list"},
			map[string]interface{}{"key": Key},
			n,
		),
	}
}

// List returns from ContainerPathAny the path struct for its child "list".
// Key: string
func (n *ContainerPathAny) List(Key string) *Container_ListPathAny {
	return &Container_ListPathAny{
		NodePath: ygot.NewNodePath(
			[]string{"list-container", "list"},
			map[string]interface{}{"key": Key},
			n,
		),
	}
}
`,
	}, {
		name:                      "2nd-level list methods -- Builder API",
		inDirectory:               deepSchemaDirectories["/root-module/container"],
		inDirectories:             deepSchemaDirectories,
		inFieldName:               "list",
		inUniqueFieldName:         "List",
		inListBuilderKeyThreshold: 1,
		inPathStructSuffix:        "Path",
		inGenerateWildcardPaths:   true,
		want: `
// ListAny returns from ContainerPath the path struct for its child "list".
func (n *ContainerPath) ListAny() *Container_ListPathAny {
	return &Container_ListPathAny{
		NodePath: ygot.NewNodePath(
			[]string{"list-container", "list"},
			map[string]interface{}{"key": "*"},
			n,
		),
	}
}

// ListAny returns from ContainerPathAny the path struct for its child "list".
func (n *ContainerPathAny) ListAny() *Container_ListPathAny {
	return &Container_ListPathAny{
		NodePath: ygot.NewNodePath(
			[]string{"list-container", "list"},
			map[string]interface{}{"key": "*"},
			n,
		),
	}
}

// WithKey sets Container_ListPathAny's key "key" to the specified value.
// Key: string
func (n *Container_ListPathAny) WithKey(Key string) *Container_ListPathAny {
	ygot.ModifyKey(n.NodePath, "key", Key)
	return n
}
`,
	}, {
		name:                    "inner container",
		inDirectory:             deepSchemaDirectories["/root-module/container"],
		inDirectories:           deepSchemaDirectories,
		inFieldName:             "inner-container",
		inUniqueFieldName:       "InnerContainer",
		inPathStructSuffix:      "Path",
		inGenerateWildcardPaths: true,
		want: `
// InnerContainer returns from ContainerPath the path struct for its child "inner-container".
func (n *ContainerPath) InnerContainer() *Container_InnerContainerPath {
	return &Container_InnerContainerPath{
		NodePath: ygot.NewNodePath(
			[]string{"inner-container"},
			map[string]interface{}{},
			n,
		),
	}
}

// InnerContainer returns from ContainerPathAny the path struct for its child "inner-container".
func (n *ContainerPathAny) InnerContainer() *Container_InnerContainerPathAny {
	return &Container_InnerContainerPathAny{
		NodePath: ygot.NewNodePath(
			[]string{"inner-container"},
			map[string]interface{}{},
			n,
		),
	}
}
`,
	}, {
		name:                    "list with state method",
		inDirectory:             directories["/root"],
		inDirectories:           directories,
		inFieldName:             "list-with-state",
		inUniqueFieldName:       "ListWithState",
		inPathStructSuffix:      "Path",
		inGenerateWildcardPaths: true,
		want: `
// ListWithStateAny returns from RootPath the path struct for its child "list-with-state".
func (n *RootPath) ListWithStateAny() *ListWithStatePathAny {
	return &ListWithStatePathAny{
		NodePath: ygot.NewNodePath(
			[]string{"list-container-with-state", "list-with-state"},
			map[string]interface{}{"key": "*"},
			n,
		),
	}
}

// ListWithState returns from RootPath the path struct for its child "list-with-state".
// Key: float64
func (n *RootPath) ListWithState(Key float64) *ListWithStatePath {
	return &ListWithStatePath{
		NodePath: ygot.NewNodePath(
			[]string{"list-container-with-state", "list-with-state"},
			map[string]interface{}{"key": Key},
			n,
		),
	}
}
`,
	}, {
		name:                    "root-level list methods",
		inDirectory:             directories["/root"],
		inDirectories:           directories,
		inFieldName:             "list",
		inUniqueFieldName:       "List",
		inPathStructSuffix:      "Path",
		inGenerateWildcardPaths: true,
		want:                    wantListMethods,
	}, {
		name:                      "root-level list methods with builder API threshold over the number of keys",
		inDirectory:               directories["/root"],
		inDirectories:             directories,
		inFieldName:               "list",
		inUniqueFieldName:         "List",
		inListBuilderKeyThreshold: 4,
		inPathStructSuffix:        "Path",
		inGenerateWildcardPaths:   true,
		want:                      wantListMethods,
	}, {
		name:                      "root-level list methods with builder API threshold over the number of keys, inSimplifyWildcardPaths=true",
		inDirectory:               directories["/root"],
		inDirectories:             directories,
		inFieldName:               "list",
		inUniqueFieldName:         "List",
		inListBuilderKeyThreshold: 4,
		inPathStructSuffix:        "Path",
		inGenerateWildcardPaths:   true,
		inSimplifyWildcardPaths:   true,
		want:                      wantListMethodsSimplified,
	}, {
		name:                      "root-level list methods over key threshold -- should use builder API",
		inDirectory:               directories["/root"],
		inDirectories:             directories,
		inFieldName:               "list",
		inUniqueFieldName:         "List",
		inListBuilderKeyThreshold: 3,
		inPathStructSuffix:        "Path",
		inGenerateWildcardPaths:   true,
		want: `
// ListAny returns from RootPath the path struct for its child "list".
func (n *RootPath) ListAny() *ListPathAny {
	return &ListPathAny{
		NodePath: ygot.NewNodePath(
			[]string{"list-container", "list"},
			map[string]interface{}{"key1": "*", "key2": "*", "union-key": "*"},
			n,
		),
	}
}

// WithKey1 sets ListPathAny's key "key1" to the specified value.
// Key1: string
func (n *ListPathAny) WithKey1(Key1 string) *ListPathAny {
	ygot.ModifyKey(n.NodePath, "key1", Key1)
	return n
}

// WithKey2 sets ListPathAny's key "key2" to the specified value.
// Key2: oc.Binary
func (n *ListPathAny) WithKey2(Key2 oc.Binary) *ListPathAny {
	ygot.ModifyKey(n.NodePath, "key2", Key2)
	return n
}

// WithUnionKey sets ListPathAny's key "union-key" to the specified value.
// UnionKey: [oc.UnionString, oc.Binary]
func (n *ListPathAny) WithUnionKey(UnionKey oc.RootModule_List_UnionKey_Union) *ListPathAny {
	ygot.ModifyKey(n.NodePath, "union-key", UnionKey)
	return n
}
`,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf strings.Builder
			if errs := generateChildConstructors(&buf, tt.inDirectory, tt.inFieldName, tt.inUniqueFieldName, tt.inDirectories, "oc.", tt.inPathStructSuffix, tt.inListBuilderKeyThreshold, tt.inGenerateWildcardPaths, tt.inSimplifyWildcardPaths); errs != nil {
				t.Fatal(errs)
			}

			if got, want := buf.String(), tt.want; got != want {
				diff, _ := testutil.GenerateUnifiedDiff(want, got)
				t.Errorf("func generateChildConstructors returned incorrect code, diff:\n%s", diff)
			}
		})
	}
}

func TestMakeKeyParams(t *testing.T) {
	tests := []struct {
		name             string
		in               *ygen.YangListAttr
		wantKeyParams    []keyParam
		wantErrSubstring string
	}{{
		name:             "empty listattr",
		in:               &ygen.YangListAttr{},
		wantErrSubstring: "invalid list - has no key",
	}, {
		name: "simple string param",
		in: &ygen.YangListAttr{
			Keys:     map[string]*ygen.MappedType{"fluorine": {NativeType: "string"}},
			KeyElems: []*yang.Entry{{Name: "fluorine"}},
		},
		wantKeyParams: []keyParam{{name: "fluorine", varName: "Fluorine", typeName: "string", typeDocString: "Fluorine: string"}},
	}, {
		name: "simple int param, also testing camel-case",
		in: &ygen.YangListAttr{
			Keys:     map[string]*ygen.MappedType{"cl-cl": {NativeType: "int"}},
			KeyElems: []*yang.Entry{{Name: "cl-cl"}},
		},
		wantKeyParams: []keyParam{{name: "cl-cl", varName: "ClCl", typeName: "int", typeDocString: "ClCl: int"}},
	}, {
		name: "name uniquification",
		in: &ygen.YangListAttr{
			Keys: map[string]*ygen.MappedType{
				"cl-cl": {NativeType: "int"},
				"clCl":  {NativeType: "int"},
			},
			KeyElems: []*yang.Entry{{Name: "cl-cl"}, {Name: "clCl"}},
		},
		wantKeyParams: []keyParam{
			{name: "cl-cl", varName: "ClCl", typeName: "int", typeDocString: "ClCl: int"},
			{name: "clCl", varName: "ClCl_", typeName: "int", typeDocString: "ClCl_: int"},
		},
	}, {
		name: "unsupported type",
		in: &ygen.YangListAttr{
			Keys:     map[string]*ygen.MappedType{"fluorine": {NativeType: "interface{}"}},
			KeyElems: []*yang.Entry{{Name: "fluorine"}},
		},
		wantKeyParams: []keyParam{{name: "fluorine", varName: "Fluorine", typeName: "string", typeDocString: "Fluorine: string"}},
	}, {
		name: "keyElems doesn't match keys",
		in: &ygen.YangListAttr{
			Keys:     map[string]*ygen.MappedType{"neon": {NativeType: "light"}},
			KeyElems: []*yang.Entry{{Name: "cl-cl"}},
		},
		wantErrSubstring: "key doesn't have a mappedType: cl-cl",
	}, {
		name: "mappedType is nil",
		in: &ygen.YangListAttr{
			Keys:     map[string]*ygen.MappedType{"cl-cl": nil},
			KeyElems: []*yang.Entry{{Name: "cl-cl"}},
		},
		wantErrSubstring: "mappedType for key is nil: cl-cl",
	}, {
		name: "multiple parameters",
		in: &ygen.YangListAttr{
			Keys: map[string]*ygen.MappedType{
				"bromine":  {NativeType: "complex128"},
				"cl-cl":    {NativeType: "int"},
				"fluorine": {NativeType: "string"},
				"iodine":   {NativeType: "float64"},
			},
			KeyElems: []*yang.Entry{{Name: "fluorine"}, {Name: "cl-cl"}, {Name: "bromine"}, {Name: "iodine"}},
		},
		wantKeyParams: []keyParam{
			{name: "fluorine", varName: "Fluorine", typeName: "string", typeDocString: "Fluorine: string"},
			{name: "cl-cl", varName: "ClCl", typeName: "int", typeDocString: "ClCl: int"},
			{name: "bromine", varName: "Bromine", typeName: "complex128", typeDocString: "Bromine: complex128"},
			{name: "iodine", varName: "Iodine", typeName: "float64", typeDocString: "Iodine: float64"},
		},
	}, {
		name: "enumerated and union parameters",
		in: &ygen.YangListAttr{
			Keys: map[string]*ygen.MappedType{
				"astatine":   {NativeType: "Halogen", IsEnumeratedValue: true},
				"tennessine": {NativeType: "Ununseptium", UnionTypes: map[string]int{"int32": 1, "float64": 2, "interface{}": 3}},
			},
			KeyElems: []*yang.Entry{{Name: "astatine"}, {Name: "tennessine"}},
		},
		wantKeyParams: []keyParam{
			{name: "astatine", varName: "Astatine", typeName: "oc.Halogen", typeDocString: "Astatine: oc.Halogen"},
			{name: "tennessine", varName: "Tennessine", typeName: "oc.Ununseptium", typeDocString: "Tennessine: [oc.UnionInt32, oc.UnionFloat64, *oc.UnionUnsupported]"},
		},
	}, {
		name: "Binary and Empty",
		in: &ygen.YangListAttr{
			Keys: map[string]*ygen.MappedType{
				"bromine": {NativeType: "Binary"},
				"cl-cl":   {NativeType: "YANGEmpty"},
			},
			KeyElems: []*yang.Entry{{Name: "cl-cl"}, {Name: "bromine"}},
		},
		wantKeyParams: []keyParam{
			{name: "cl-cl", varName: "ClCl", typeName: "oc.YANGEmpty", typeDocString: "ClCl: oc.YANGEmpty"},
			{name: "bromine", varName: "Bromine", typeName: "oc.Binary", typeDocString: "Bromine: oc.Binary"},
		},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotKeyParams, err := makeKeyParams(tt.in, "oc.")
			if diff := cmp.Diff(tt.wantKeyParams, gotKeyParams, cmp.AllowUnexported(keyParam{})); diff != "" {
				t.Errorf("(-want, +got):\n%s", diff)
			}

			if diff := errdiff.Check(err, tt.wantErrSubstring); diff != "" {
				t.Errorf("func makeKeyParams, %v", diff)
			}
		})
	}
}

func TestCombinations(t *testing.T) {
	tests := []struct {
		name string
		in   int
		want [][]int
	}{{
		name: "n = 0",
		in:   0,
		want: [][]int{{}},
	}, {
		name: "n = 1",
		in:   1,
		want: [][]int{{}, {0}},
	}, {
		name: "n = 2",
		in:   2,
		want: [][]int{{}, {0}, {1}, {0, 1}},
	}, {
		name: "n = 3",
		in:   3,
		want: [][]int{{}, {0}, {1}, {0, 1}, {2}, {0, 2}, {1, 2}, {0, 1, 2}},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := combinations(tt.in)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("(-want, +got):\n%s", diff)
			}
		})
	}
}
