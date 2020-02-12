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
	"bytes"
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

// yangTestCase describes a test case for which code generation is performed
// through Goyang's API, it provides the input set of parameters in a way that
// can be reused across tests.
type yangTestCase struct {
	name                string      // Name is the identifier for the test.
	inFiles             []string    // inFiles is the set of inputFiles for the test.
	inIncludePaths      []string    // inIncludePaths is the set of paths that should be searched for imports.
	wantStructsCodeFile string      // wantStructsCodeFile is the path of the generated Go code that the output of the test should be compared to.
	wantNodeDataMap     NodeDataMap // wantNodeDataMap is the expected NodeDataMap to be produced to accompany the path struct outputs.
	wantErr             bool        // wantErr specifies whether the test should expect an error.
}

func TestGeneratePathCode(t *testing.T) {
	tests := []yangTestCase{
		{
			name:                "simple openconfig test",
			inFiles:             []string{filepath.Join(datapath, "openconfig-simple.yang")},
			wantStructsCodeFile: filepath.Join(TestRoot, "testdata/structs/openconfig-simple.path-txt"),
			wantNodeDataMap: NodeDataMap{
				"Parent": {
					GoTypeName:       "*oc.Parent",
					GoFieldName:      "Parent",
					ParentGoTypeName: "Device",
					IsLeaf:           false,
					IsScalarField:    false,
				},
				"Parent_Child": {
					GoTypeName:       "*oc.Parent_Child",
					GoFieldName:      "Child",
					ParentGoTypeName: "Parent",
					IsLeaf:           false,
					IsScalarField:    false,
				},
				"Parent_Child_Four": {
					GoTypeName:       "oc.Binary",
					GoFieldName:      "Four",
					ParentGoTypeName: "Parent_Child",
					IsLeaf:           true,
					IsScalarField:    false,
					YANGTypeName:     "binary",
				},
				"Parent_Child_One": {
					GoTypeName:       "string",
					GoFieldName:      "One",
					ParentGoTypeName: "Parent_Child",
					IsLeaf:           true,
					IsScalarField:    true,
					YANGTypeName:     "string",
				},
				"Parent_Child_Three": {
					GoTypeName:       "oc.E_OpenconfigSimple_Child_Three",
					GoFieldName:      "Three",
					ParentGoTypeName: "Parent_Child",
					IsLeaf:           true,
					IsScalarField:    false,
					YANGTypeName:     "enumeration",
				},
				"Parent_Child_Two": {
					GoTypeName:       "string",
					GoFieldName:      "Two",
					ParentGoTypeName: "Parent_Child",
					IsLeaf:           true,
					IsScalarField:    true,
					YANGTypeName:     "string",
				},
				"RemoteContainer": {
					GoTypeName:       "*oc.RemoteContainer",
					GoFieldName:      "RemoteContainer",
					ParentGoTypeName: "Device",
					IsLeaf:           false,
					IsScalarField:    false,
				},
				"RemoteContainer_ALeaf": {
					GoTypeName:       "string",
					GoFieldName:      "ALeaf",
					ParentGoTypeName: "RemoteContainer",
					IsLeaf:           true,
					IsScalarField:    true,
					YANGTypeName:     "string",
				}},
		}, {
			name:                "simple openconfig test with list",
			inFiles:             []string{filepath.Join(datapath, "openconfig-withlist.yang")},
			wantStructsCodeFile: filepath.Join(TestRoot, "testdata/structs/openconfig-withlist.path-txt"),
		}, {
			name:                "simple openconfig test with union & typedef & identity & enum",
			inFiles:             []string{filepath.Join(datapath, "openconfig-unione.yang")},
			wantStructsCodeFile: filepath.Join(TestRoot, "testdata/structs/openconfig-unione.path-txt"),
			wantNodeDataMap: NodeDataMap{
				"DupEnum": {
					GoTypeName:       "*oc.DupEnum",
					GoFieldName:      "DupEnum",
					ParentGoTypeName: "Device",
					IsLeaf:           false,
					IsScalarField:    false,
				},
				"DupEnum_A": {
					GoTypeName:       "oc.E_OpenconfigUnione_DupEnum_A",
					GoFieldName:      "A",
					ParentGoTypeName: "DupEnum",
					IsLeaf:           true,
					IsScalarField:    false,
					YANGTypeName:     "enumeration",
				},
				"DupEnum_B": {
					GoTypeName:       "oc.E_OpenconfigUnione_DupEnum_B",
					GoFieldName:      "B",
					ParentGoTypeName: "DupEnum",
					IsLeaf:           true,
					IsScalarField:    false,
					YANGTypeName:     "enumeration",
				},
				"Platform": {
					GoTypeName:       "*oc.Platform",
					GoFieldName:      "Platform",
					ParentGoTypeName: "Device",
					IsLeaf:           false,
					IsScalarField:    false,
				},
				"Platform_Component": {
					GoTypeName:       "*oc.Platform_Component",
					GoFieldName:      "Component",
					ParentGoTypeName: "Platform",
					IsLeaf:           false,
					IsScalarField:    false,
				},
				"Platform_Component_E1": {
					GoTypeName:       "oc.Platform_Component_E1_Union",
					GoFieldName:      "E1",
					ParentGoTypeName: "Platform_Component",
					IsLeaf:           true,
					IsScalarField:    false,
					YANGTypeName:     "enumtypedef",
				},
				"Platform_Component_Enumerated": {
					GoTypeName:       "oc.Platform_Component_Enumerated_Union",
					GoFieldName:      "Enumerated",
					ParentGoTypeName: "Platform_Component",
					IsLeaf:           true,
					IsScalarField:    false,
					YANGTypeName:     "enumerated-union-type",
				},
				"Platform_Component_Power": {
					GoTypeName:       "oc.Platform_Component_Power_Union",
					GoFieldName:      "Power",
					ParentGoTypeName: "Platform_Component",
					IsLeaf:           true,
					IsScalarField:    false,
					YANGTypeName:     "union",
				},
				"Platform_Component_R1": {
					GoTypeName:       "oc.Platform_Component_E1_Union",
					GoFieldName:      "R1",
					ParentGoTypeName: "Platform_Component",
					IsLeaf:           true,
					IsScalarField:    false,
					YANGTypeName:     "leafref",
				},
				"Platform_Component_Type": {
					GoTypeName:       "oc.Platform_Component_Type_Union",
					GoFieldName:      "Type",
					ParentGoTypeName: "Platform_Component",
					IsLeaf:           true,
					IsScalarField:    false,
					YANGTypeName:     "union",
				}},
		}, {
			name:                "simple openconfig test with submodule and union list key",
			inFiles:             []string{filepath.Join(datapath, "enum-module.yang")},
			wantStructsCodeFile: filepath.Join(TestRoot, "testdata/structs/enum-module.path-txt"),
			wantNodeDataMap: NodeDataMap{
				"AList": {
					GoTypeName:       "*oc.AList",
					GoFieldName:      "AList",
					ParentGoTypeName: "Device",
					IsLeaf:           false,
					IsScalarField:    false,
				},
				"AList_Value": {
					GoTypeName:       "oc.AList_Value_Union",
					GoFieldName:      "Value",
					ParentGoTypeName: "AList",
					IsLeaf:           true,
					IsScalarField:    false,
					YANGTypeName:     "td",
				},
				"BList": {
					GoTypeName:       "*oc.BList",
					GoFieldName:      "BList",
					ParentGoTypeName: "Device",
					IsLeaf:           false,
					IsScalarField:    false,
				},
				"BList_Value": {
					GoTypeName:       "oc.BList_Value_Union",
					GoFieldName:      "Value",
					ParentGoTypeName: "BList",
					IsLeaf:           true,
					IsScalarField:    false,
					YANGTypeName:     "td",
				},
				"C": {
					GoTypeName:       "*oc.C",
					GoFieldName:      "C",
					ParentGoTypeName: "Device",
					IsLeaf:           false,
					IsScalarField:    false,
				},
				"C_Cl": {
					GoTypeName:       "oc.E_EnumModule_EnumModule_Cl",
					GoFieldName:      "Cl",
					ParentGoTypeName: "C",
					IsLeaf:           true,
					IsScalarField:    false,
					YANGTypeName:     "enumeration",
				},
				"Parent": {
					GoTypeName:       "*oc.Parent",
					GoFieldName:      "Parent",
					ParentGoTypeName: "Device",
					IsLeaf:           false,
					IsScalarField:    false,
				},
				"Parent_Child": {
					GoTypeName:       "*oc.Parent_Child",
					GoFieldName:      "Child",
					ParentGoTypeName: "Parent",
					IsLeaf:           false,
					IsScalarField:    false,
				},
				"Parent_Child_Id": {
					GoTypeName:       "oc.E_EnumTypes_ID",
					GoFieldName:      "Id",
					ParentGoTypeName: "Parent_Child",
					IsLeaf:           true,
					IsScalarField:    false,
					YANGTypeName:     "identityref",
				}},
		}, {
			name:                "simple openconfig test with choice and cases",
			inFiles:             []string{filepath.Join(datapath, "choice-case-example.yang")},
			wantStructsCodeFile: filepath.Join(TestRoot, "testdata/structs/choice-case-example.path-txt"),
		}, {
			name: "simple openconfig test with augmentations",
			inFiles: []string{
				filepath.Join(datapath, "openconfig-simple-target.yang"),
				filepath.Join(datapath, "openconfig-simple-augment.yang"),
			},
			wantStructsCodeFile: filepath.Join(TestRoot, "testdata/structs/openconfig-augmented.path-txt"),
			wantNodeDataMap: NodeDataMap{
				"Native": {
					GoTypeName:       "*oc.Native",
					GoFieldName:      "Native",
					ParentGoTypeName: "Device",
					IsLeaf:           false,
					IsScalarField:    false,
				},
				"Native_A": {
					GoTypeName:       "string",
					GoFieldName:      "A",
					ParentGoTypeName: "Native",
					IsLeaf:           true,
					IsScalarField:    true,
					YANGTypeName:     "string",
				},
				"Target": {
					GoTypeName:       "*oc.Target",
					GoFieldName:      "Target",
					ParentGoTypeName: "Device",
					IsLeaf:           false,
					IsScalarField:    false,
				},
				"Target_Foo": {
					GoTypeName:       "*oc.Target_Foo",
					GoFieldName:      "Foo",
					ParentGoTypeName: "Target",
					IsLeaf:           false,
					IsScalarField:    false,
				},
				"Target_Foo_A": {
					GoTypeName:       "string",
					GoFieldName:      "A",
					ParentGoTypeName: "Target_Foo",
					IsLeaf:           true,
					IsScalarField:    true,
					YANGTypeName:     "string",
				}},
		}, {
			name:                "simple openconfig test with camelcase-name extension",
			inFiles:             []string{filepath.Join(datapath, "openconfig-enumcamelcase.yang")},
			wantStructsCodeFile: filepath.Join(TestRoot, "testdata/structs/openconfig-enumcamelcase.path-txt"),
		}, {
			name:                "simple openconfig test with camelcase-name extension in container and leaf",
			inFiles:             []string{filepath.Join(datapath, "openconfig-camelcase.yang")},
			wantStructsCodeFile: filepath.Join(TestRoot, "testdata/structs/openconfig-camelcase.path-txt"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			genCode := func() (string, NodeDataMap, *GenConfig) {
				cg := NewDefaultConfig("github.com/openconfig/ygot/ypathgen/testdata/exampleoc")
				// Set the name of the caller explicitly to avoid issues when
				// the unit tests are called by external test entities.
				cg.GeneratingBinary = "pathgen-tests"
				cg.FakeRootName = "device"

				gotCode, gotNodeDataMap, err := cg.GeneratePathCode(tt.inFiles, tt.inIncludePaths)
				if err != nil && !tt.wantErr {
					t.Fatalf("GeneratePathCode(%v, %v): Config: %v, got unexpected error: %v, want: nil", tt.inFiles, tt.inIncludePaths, cg, err)
				}

				return gotCode.String(), gotNodeDataMap, cg
			}

			gotCode, gotNodeDataMap, cg := genCode()

			if tt.wantNodeDataMap != nil {
				if diff := cmp.Diff(tt.wantNodeDataMap, gotNodeDataMap); diff != "" {
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
				diff, _ := testutil.GenerateUnifiedDiff(gotCode, wantCode)
				t.Errorf("GeneratePathCode(%v, %v), Config: %v, did not return correct code (file: %v), diff:\n%s",
					tt.inFiles, tt.inIncludePaths, cg, tt.wantStructsCodeFile, diff)
			}

			for i := 0; i < deflakeRuns; i++ {
				gotAttempt, _, _ := genCode()
				if gotAttempt != gotCode {
					diff, _ := testutil.GenerateUnifiedDiff(gotCode, gotAttempt)
					t.Fatalf("flaky code generation, diff:\n%s", diff)
				}
			}
		})
	}
}

func TestGeneratePathCodeSplitFiles(t *testing.T) {
	tests := []struct {
		name                 string   // Name is the identifier for the test.
		inFiles              []string // inFiles is the set of inputFiles for the test.
		inIncludePaths       []string // inIncludePaths is the set of paths that should be searched for imports.
		inFileNumber         int      // inFileNumber is the number of files into which to split the generated code.
		wantStructsCodeFiles []string // wantStructsCodeFiles is the paths of the generated Go code that the output of the test should be compared to.
		wantErr              bool     // whether an error is expected from the SplitFiles call
	}{{
		name:         "fileNumber is higher than total number of structs",
		inFiles:      []string{filepath.Join(datapath, "openconfig-simple.yang")},
		inFileNumber: 5,
		wantErr:      true,
	}, {
		name:                 "fileNumber is exactly the total number of structs",
		inFiles:              []string{filepath.Join(datapath, "openconfig-simple.yang")},
		inFileNumber:         4,
		wantStructsCodeFiles: []string{filepath.Join(TestRoot, "testdata/structs/openconfig-simple-40.path-txt"), filepath.Join(TestRoot, "testdata/structs/openconfig-simple-41.path-txt"), filepath.Join(TestRoot, "testdata/structs/openconfig-simple-42.path-txt"), filepath.Join(TestRoot, "testdata/structs/openconfig-simple-43.path-txt")},
	}, {
		name:                 "fileNumber is just under the total number of structs",
		inFiles:              []string{filepath.Join(datapath, "openconfig-simple.yang")},
		inFileNumber:         3,
		wantStructsCodeFiles: []string{filepath.Join(TestRoot, "testdata/structs/openconfig-simple-30.path-txt"), filepath.Join(TestRoot, "testdata/structs/openconfig-simple-31.path-txt"), filepath.Join(TestRoot, "testdata/structs/openconfig-simple-32.path-txt")},
	}, {
		name:                 "fileNumber is half the total number of structs",
		inFiles:              []string{filepath.Join(datapath, "openconfig-simple.yang")},
		inFileNumber:         2,
		wantStructsCodeFiles: []string{filepath.Join(TestRoot, "testdata/structs/openconfig-simple-0.path-txt"), filepath.Join(TestRoot, "testdata/structs/openconfig-simple-1.path-txt")},
	}, {
		name:                 "single file",
		inFiles:              []string{filepath.Join(datapath, "openconfig-simple.yang")},
		inFileNumber:         1,
		wantStructsCodeFiles: []string{filepath.Join(TestRoot, "testdata/structs/openconfig-simple.path-txt")},
	}, {
		name:         "fileNumber is 0",
		inFiles:      []string{filepath.Join(datapath, "openconfig-simple.yang")},
		inFileNumber: 0,
		wantErr:      true,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			genCode := func() ([]string, *GenConfig) {
				cg := NewDefaultConfig("github.com/openconfig/ygot/ypathgen/testdata/exampleoc")
				// Set the name of the caller explicitly to avoid issues when
				// the unit tests are called by external test entities.
				cg.GeneratingBinary = "pathgen-tests"
				cg.FakeRootName = "device"

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
						diff, _ := testutil.GenerateUnifiedDiff(gotCode[i], wantCode[i])
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

// wantListMethods is the expected child constructor methods for the list node.
const wantListMethods = `
// ListAny returns from Root the path struct for its child "list".
func (n *Root) ListAny() *ListAny {
	return &ListAny{
		NodePath: ygot.NewNodePath(
			[]string{"list-container", "list"},
			map[string]interface{}{"key1": "*", "key2": "*", "union-key": "*"},
			n,
		),
	}
}

// ListAnyKey2AnyUnionKey returns from Root the path struct for its child "list".
func (n *Root) ListAnyKey2AnyUnionKey(Key1 string) *ListAny {
	return &ListAny{
		NodePath: ygot.NewNodePath(
			[]string{"list-container", "list"},
			map[string]interface{}{"key1": Key1, "key2": "*", "union-key": "*"},
			n,
		),
	}
}

// ListAnyKey1AnyUnionKey returns from Root the path struct for its child "list".
func (n *Root) ListAnyKey1AnyUnionKey(Key2 oc.Binary) *ListAny {
	return &ListAny{
		NodePath: ygot.NewNodePath(
			[]string{"list-container", "list"},
			map[string]interface{}{"key1": "*", "key2": Key2, "union-key": "*"},
			n,
		),
	}
}

// ListAnyUnionKey returns from Root the path struct for its child "list".
func (n *Root) ListAnyUnionKey(Key1 string, Key2 oc.Binary) *ListAny {
	return &ListAny{
		NodePath: ygot.NewNodePath(
			[]string{"list-container", "list"},
			map[string]interface{}{"key1": Key1, "key2": Key2, "union-key": "*"},
			n,
		),
	}
}

// ListAnyKey1AnyKey2 returns from Root the path struct for its child "list".
func (n *Root) ListAnyKey1AnyKey2(UnionKey oc.RootModule_List_UnionKey_Union) *ListAny {
	return &ListAny{
		NodePath: ygot.NewNodePath(
			[]string{"list-container", "list"},
			map[string]interface{}{"key1": "*", "key2": "*", "union-key": UnionKey},
			n,
		),
	}
}

// ListAnyKey2 returns from Root the path struct for its child "list".
func (n *Root) ListAnyKey2(Key1 string, UnionKey oc.RootModule_List_UnionKey_Union) *ListAny {
	return &ListAny{
		NodePath: ygot.NewNodePath(
			[]string{"list-container", "list"},
			map[string]interface{}{"key1": Key1, "key2": "*", "union-key": UnionKey},
			n,
		),
	}
}

// ListAnyKey1 returns from Root the path struct for its child "list".
func (n *Root) ListAnyKey1(Key2 oc.Binary, UnionKey oc.RootModule_List_UnionKey_Union) *ListAny {
	return &ListAny{
		NodePath: ygot.NewNodePath(
			[]string{"list-container", "list"},
			map[string]interface{}{"key1": "*", "key2": Key2, "union-key": UnionKey},
			n,
		),
	}
}

// List returns from Root the path struct for its child "list".
func (n *Root) List(Key1 string, Key2 oc.Binary, UnionKey oc.RootModule_List_UnionKey_Union) *List {
	return &List{
		NodePath: ygot.NewNodePath(
			[]string{"list-container", "list"},
			map[string]interface{}{"key1": Key1, "key2": Key2, "union-key": UnionKey},
			n,
		),
	}
}
`

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
		name                   string
		inDirectories          map[string]*ygen.Directory
		inLeafTypeMap          map[string]map[string]*ygen.MappedType
		inSchemaStructPkgAlias string
		wantNodeDataMap        NodeDataMap
		wantSorted             []string
		wantErrSubstrings      []string
	}{{
		name:          "scalar leaf",
		inDirectories: map[string]*ygen.Directory{"/root-module/container": directories["/root-module/container"]},
		inLeafTypeMap: map[string]map[string]*ygen.MappedType{
			"/root-module/container": {
				"leaf": leafTypeMap["/root-module/container"]["leaf"],
			},
		},
		inSchemaStructPkgAlias: "struct",
		wantNodeDataMap: NodeDataMap{
			"Container_Leaf": {
				GoTypeName:       "int32",
				GoFieldName:      "Leaf",
				ParentGoTypeName: "Container",
				IsLeaf:           true,
				IsScalarField:    true,
				YANGTypeName:     "int32",
			},
		},
		wantSorted: []string{"Container_Leaf"},
	}, {
		name:                   "non-leaf and non-scalar leaf",
		inDirectories:          directoryWithBinaryLeaf,
		inLeafTypeMap:          leafTypeMap2,
		inSchemaStructPkgAlias: "struct",
		wantNodeDataMap: NodeDataMap{
			"Container": {
				GoTypeName:       "*struct.Container",
				GoFieldName:      "Container",
				ParentGoTypeName: "Root",
				IsLeaf:           false,
				IsScalarField:    false,
			},
			"Container_Leaf": {
				GoTypeName:       "struct.Binary",
				GoFieldName:      "Leaf",
				ParentGoTypeName: "Container",
				IsLeaf:           true,
				IsScalarField:    false,
			},
		},
		wantSorted: []string{"Container", "Container_Leaf"},
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
		inSchemaStructPkgAlias: "oc",
		wantErrSubstrings:      []string{`path "/root-module/container" does not exist`},
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
		inSchemaStructPkgAlias: "oc",
		wantErrSubstrings:      []string{`field name "leaf" does not exist`},
	}, {
		name:                   "big test with everything",
		inDirectories:          directories,
		inLeafTypeMap:          leafTypeMap,
		inSchemaStructPkgAlias: "oc",
		wantNodeDataMap: NodeDataMap{
			"Container": {
				GoTypeName:       "*oc.Container",
				GoFieldName:      "Container",
				ParentGoTypeName: "Root",
				IsLeaf:           false,
				IsScalarField:    false,
			},
			"ContainerWithConfig": {
				GoTypeName:       "*oc.ContainerWithConfig",
				GoFieldName:      "ContainerWithConfig",
				ParentGoTypeName: "Root",
				IsLeaf:           false,
				IsScalarField:    false,
			},
			"ContainerWithConfig_Leaf": {
				GoTypeName:       "oc.Binary",
				GoFieldName:      "Leaf",
				ParentGoTypeName: "ContainerWithConfig",
				IsLeaf:           true,
				IsScalarField:    false,
			},
			"ContainerWithConfig_Leaflist": {
				GoTypeName:       "[]uint32",
				GoFieldName:      "Leaflist",
				ParentGoTypeName: "ContainerWithConfig",
				IsLeaf:           true,
				IsScalarField:    false,
			},
			"ContainerWithConfig_Leaflist2": {
				GoTypeName:       "[]oc.Binary",
				GoFieldName:      "Leaflist2",
				ParentGoTypeName: "ContainerWithConfig",
				IsLeaf:           true,
				IsScalarField:    false,
			},
			"Container_Leaf": {
				GoTypeName:       "int32",
				GoFieldName:      "Leaf",
				ParentGoTypeName: "Container",
				IsLeaf:           true,
				IsScalarField:    true,
				YANGTypeName:     "int32",
			},
			"Leaf": {
				GoTypeName:       "oc.Binary",
				GoFieldName:      "Leaf",
				ParentGoTypeName: "Root",
				IsLeaf:           true,
				IsScalarField:    false,
				YANGTypeName:     "ieeefloat32",
			},
			"List": {
				GoTypeName:       "*oc.List",
				GoFieldName:      "List",
				ParentGoTypeName: "Root",
				IsLeaf:           false,
				IsScalarField:    false,
			},
			"ListWithState": {
				GoTypeName:       "*oc.ListWithState",
				GoFieldName:      "ListWithState",
				ParentGoTypeName: "Root",
				IsLeaf:           false,
				IsScalarField:    false,
			},
			"ListWithState_Key": {
				GoTypeName:       "float64",
				GoFieldName:      "Key",
				ParentGoTypeName: "ListWithState",
				IsLeaf:           true,
				IsScalarField:    true,
			},
			"List_Key1": {
				GoTypeName:       "string",
				GoFieldName:      "Key1",
				ParentGoTypeName: "List",
				IsLeaf:           true,
				IsScalarField:    true,
			},
			"List_Key2": {
				GoTypeName:       "oc.Binary",
				GoFieldName:      "Key2",
				ParentGoTypeName: "List",
				IsLeaf:           true,
				IsScalarField:    false,
			},
			"List_UnionKey": {
				GoTypeName:       "oc.RootModule_List_UnionKey_Union",
				GoFieldName:      "UnionKey",
				ParentGoTypeName: "List",
				IsLeaf:           true,
				IsScalarField:    false,
			}},
		wantSorted: []string{
			"Container",
			"ContainerWithConfig",
			"ContainerWithConfig_Leaf",
			"ContainerWithConfig_Leaflist",
			"ContainerWithConfig_Leaflist2",
			"Container_Leaf",
			"Leaf",
			"List",
			"ListWithState",
			"ListWithState_Key",
			"List_Key1",
			"List_Key2",
			"List_UnionKey",
		},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErrs := getNodeDataMap(tt.inDirectories, tt.inLeafTypeMap, tt.inSchemaStructPkgAlias)
			// TODO(wenbli): Enhance gNMI's errdiff with checking a slice of substrings and use here.
			var gotErrStrs []string
			for _, err := range gotErrs {
				gotErrStrs = append(gotErrStrs, err.Error())
			}
			if diff := cmp.Diff(tt.wantErrSubstrings, gotErrStrs, cmp.Comparer(func(x, y string) bool { return strings.Contains(x, y) || strings.Contains(y, x) })); diff != "" {
				t.Fatalf("Error substring check failed (-want, +got):\n%v", diff)
			}
			if diff := cmp.Diff(tt.wantNodeDataMap, got); diff != "" {
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
		name        string
		inDirectory *ygen.Directory
		want        GoPathStructCodeSnippet
	}{{
		name:        "container-with-config",
		inDirectory: directories["/root-module/container-with-config"],
		want: GoPathStructCodeSnippet{
			PathStructName: "ContainerWithConfig",
			StructBase: `
// ContainerWithConfig represents the /root-module/container-with-config YANG schema element.
type ContainerWithConfig struct {
	ygot.NodePath
}

// ContainerWithConfigAny represents the wildcard version of the /root-module/container-with-config YANG schema element.
type ContainerWithConfigAny struct {
	ygot.NodePath
}

// ContainerWithConfig_Leaf represents the /root-module/container-with-config/state/leaf YANG schema element.
type ContainerWithConfig_Leaf struct {
	ygot.NodePath
}

// ContainerWithConfig_LeafAny represents the wildcard version of the /root-module/container-with-config/state/leaf YANG schema element.
type ContainerWithConfig_LeafAny struct {
	ygot.NodePath
}

// ContainerWithConfig_Leaflist represents the /root-module/container-with-config/state/leaflist YANG schema element.
type ContainerWithConfig_Leaflist struct {
	ygot.NodePath
}

// ContainerWithConfig_LeaflistAny represents the wildcard version of the /root-module/container-with-config/state/leaflist YANG schema element.
type ContainerWithConfig_LeaflistAny struct {
	ygot.NodePath
}

// ContainerWithConfig_Leaflist2 represents the /root-module/container-with-config/state/leaflist2 YANG schema element.
type ContainerWithConfig_Leaflist2 struct {
	ygot.NodePath
}

// ContainerWithConfig_Leaflist2Any represents the wildcard version of the /root-module/container-with-config/state/leaflist2 YANG schema element.
type ContainerWithConfig_Leaflist2Any struct {
	ygot.NodePath
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
	}, {
		name:        "fakeroot",
		inDirectory: directories["/root"],
		want: GoPathStructCodeSnippet{
			PathStructName: "Root",
			StructBase: `
// Root represents the /root YANG schema element.
type Root struct {
	ygot.NodePath
	id string
}

func DeviceRoot(id string) *Root {
	return &Root{id: id}
}

// Leaf represents the /root-module/leaf YANG schema element.
type Leaf struct {
	ygot.NodePath
}

// LeafAny represents the wildcard version of the /root-module/leaf YANG schema element.
type LeafAny struct {
	ygot.NodePath
}
`,
			ChildConstructors: `
// Container returns from Root the path struct for its child "container".
func (n *Root) Container() *Container {
	return &Container{
		NodePath: ygot.NewNodePath(
			[]string{"container"},
			map[string]interface{}{},
			n,
		),
	}
}

// ContainerWithConfig returns from Root the path struct for its child "container-with-config".
func (n *Root) ContainerWithConfig() *ContainerWithConfig {
	return &ContainerWithConfig{
		NodePath: ygot.NewNodePath(
			[]string{"container-with-config"},
			map[string]interface{}{},
			n,
		),
	}
}

// Leaf returns from Root the path struct for its child "leaf".
func (n *Root) Leaf() *Leaf {
	return &Leaf{
		NodePath: ygot.NewNodePath(
			[]string{"leaf"},
			map[string]interface{}{},
			n,
		),
	}
}
` + wantListMethods + `
// ListWithStateAny returns from Root the path struct for its child "list-with-state".
func (n *Root) ListWithStateAny() *ListWithStateAny {
	return &ListWithStateAny{
		NodePath: ygot.NewNodePath(
			[]string{"list-container-with-state", "list-with-state"},
			map[string]interface{}{"key": "*"},
			n,
		),
	}
}

// ListWithState returns from Root the path struct for its child "list-with-state".
func (n *Root) ListWithState(Key float64) *ListWithState {
	return &ListWithState{
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
	ygot.NodePath
}

// ListAny represents the wildcard version of the /root-module/list-container/list YANG schema element.
type ListAny struct {
	ygot.NodePath
}

// List_Key1 represents the /root-module/list-container/list/key1 YANG schema element.
type List_Key1 struct {
	ygot.NodePath
}

// List_Key1Any represents the wildcard version of the /root-module/list-container/list/key1 YANG schema element.
type List_Key1Any struct {
	ygot.NodePath
}

// List_Key2 represents the /root-module/list-container/list/key2 YANG schema element.
type List_Key2 struct {
	ygot.NodePath
}

// List_Key2Any represents the wildcard version of the /root-module/list-container/list/key2 YANG schema element.
type List_Key2Any struct {
	ygot.NodePath
}

// List_UnionKey represents the /root-module/list-container/list/union-key YANG schema element.
type List_UnionKey struct {
	ygot.NodePath
}

// List_UnionKeyAny represents the wildcard version of the /root-module/list-container/list/union-key YANG schema element.
type List_UnionKeyAny struct {
	ygot.NodePath
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
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := generateDirectorySnippet(tt.inDirectory, directories, "oc")
			if gotErr != nil {
				t.Fatalf("func generateDirectorySnippet, unexpected error: %v", gotErr)
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
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
		name              string
		inDirectory       *ygen.Directory
		inDirectories     map[string]*ygen.Directory
		inFieldName       string
		inUniqueFieldName string
		want              string
	}{{
		name:              "container method",
		inDirectory:       directories["/root"],
		inDirectories:     directories,
		inFieldName:       "container",
		inUniqueFieldName: "Container",
		want: `
// Container returns from Root the path struct for its child "container".
func (n *Root) Container() *Container {
	return &Container{
		NodePath: ygot.NewNodePath(
			[]string{"container"},
			map[string]interface{}{},
			n,
		),
	}
}
`,
	}, {
		name:              "container leaf method",
		inDirectory:       directories["/root-module/container"],
		inDirectories:     directories,
		inFieldName:       "leaf",
		inUniqueFieldName: "Leaf",
		want: `
// Leaf returns from Container the path struct for its child "leaf".
func (n *Container) Leaf() *Container_Leaf {
	return &Container_Leaf{
		NodePath: ygot.NewNodePath(
			[]string{"leaf"},
			map[string]interface{}{},
			n,
		),
	}
}

// Leaf returns from ContainerAny the path struct for its child "leaf".
func (n *ContainerAny) Leaf() *Container_LeafAny {
	return &Container_LeafAny{
		NodePath: ygot.NewNodePath(
			[]string{"leaf"},
			map[string]interface{}{},
			n,
		),
	}
}
`,
	}, {
		name:              "top-level leaf method",
		inDirectory:       directories["/root"],
		inDirectories:     directories,
		inFieldName:       "leaf",
		inUniqueFieldName: "Leaf",
		want: `
// Leaf returns from Root the path struct for its child "leaf".
func (n *Root) Leaf() *Leaf {
	return &Leaf{
		NodePath: ygot.NewNodePath(
			[]string{"leaf"},
			map[string]interface{}{},
			n,
		),
	}
}
`,
	}, {
		name:              "container-with-config leaf method",
		inDirectory:       directories["/root-module/container-with-config"],
		inDirectories:     directories,
		inFieldName:       "leaf",
		inUniqueFieldName: "Leaf",
		want: `
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
`,
	}, {
		name:              "2nd-level list methods",
		inDirectory:       deepSchemaDirectories["/root-module/container"],
		inDirectories:     deepSchemaDirectories,
		inFieldName:       "list",
		inUniqueFieldName: "List",
		want: `
// ListAny returns from Container the path struct for its child "list".
func (n *Container) ListAny() *Container_ListAny {
	return &Container_ListAny{
		NodePath: ygot.NewNodePath(
			[]string{"list-container", "list"},
			map[string]interface{}{"key": "*"},
			n,
		),
	}
}

// ListAny returns from ContainerAny the path struct for its child "list".
func (n *ContainerAny) ListAny() *Container_ListAny {
	return &Container_ListAny{
		NodePath: ygot.NewNodePath(
			[]string{"list-container", "list"},
			map[string]interface{}{"key": "*"},
			n,
		),
	}
}

// List returns from Container the path struct for its child "list".
func (n *Container) List(Key string) *Container_List {
	return &Container_List{
		NodePath: ygot.NewNodePath(
			[]string{"list-container", "list"},
			map[string]interface{}{"key": Key},
			n,
		),
	}
}

// List returns from ContainerAny the path struct for its child "list".
func (n *ContainerAny) List(Key string) *Container_ListAny {
	return &Container_ListAny{
		NodePath: ygot.NewNodePath(
			[]string{"list-container", "list"},
			map[string]interface{}{"key": Key},
			n,
		),
	}
}
`,
	}, {
		name:              "inner container",
		inDirectory:       deepSchemaDirectories["/root-module/container"],
		inDirectories:     deepSchemaDirectories,
		inFieldName:       "inner-container",
		inUniqueFieldName: "InnerContainer",
		want: `
// InnerContainer returns from Container the path struct for its child "inner-container".
func (n *Container) InnerContainer() *Container_InnerContainer {
	return &Container_InnerContainer{
		NodePath: ygot.NewNodePath(
			[]string{"inner-container"},
			map[string]interface{}{},
			n,
		),
	}
}

// InnerContainer returns from ContainerAny the path struct for its child "inner-container".
func (n *ContainerAny) InnerContainer() *Container_InnerContainerAny {
	return &Container_InnerContainerAny{
		NodePath: ygot.NewNodePath(
			[]string{"inner-container"},
			map[string]interface{}{},
			n,
		),
	}
}
`,
	}, {
		name:              "list method",
		inDirectory:       directories["/root"],
		inDirectories:     directories,
		inFieldName:       "list",
		inUniqueFieldName: "List",
		want:              wantListMethods,
	}, {
		name:              "list with state method",
		inDirectory:       directories["/root"],
		inDirectories:     directories,
		inFieldName:       "list-with-state",
		inUniqueFieldName: "ListWithState",
		want: `
// ListWithStateAny returns from Root the path struct for its child "list-with-state".
func (n *Root) ListWithStateAny() *ListWithStateAny {
	return &ListWithStateAny{
		NodePath: ygot.NewNodePath(
			[]string{"list-container-with-state", "list-with-state"},
			map[string]interface{}{"key": "*"},
			n,
		),
	}
}

// ListWithState returns from Root the path struct for its child "list-with-state".
func (n *Root) ListWithState(Key float64) *ListWithState {
	return &ListWithState{
		NodePath: ygot.NewNodePath(
			[]string{"list-container-with-state", "list-with-state"},
			map[string]interface{}{"key": Key},
			n,
		),
	}
}
`,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			if errs := generateChildConstructors(&buf, tt.inDirectory, tt.inFieldName, tt.inUniqueFieldName, tt.inDirectories, "oc"); errs != nil {
				t.Fatal(errs)
			}

			if got, want := buf.String(), tt.want; got != want {
				diff, _ := testutil.GenerateUnifiedDiff(got, want)
				t.Errorf("func generateChildConstructors returned incorrect code, diff:\n%s", diff)
			}
		})
	}
}

func TestGenerateParamListStrs(t *testing.T) {
	tests := []struct {
		name             string
		in               *ygen.YangListAttr
		want             []string
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
		want: []string{"Fluorine string"},
	}, {
		name: "simple int param, also testing camel-case",
		in: &ygen.YangListAttr{
			Keys:     map[string]*ygen.MappedType{"cl-cl": {NativeType: "int"}},
			KeyElems: []*yang.Entry{{Name: "cl-cl"}},
		},
		want: []string{"ClCl int"},
	}, {
		name: "name uniquification",
		in: &ygen.YangListAttr{
			Keys: map[string]*ygen.MappedType{
				"cl-cl": {NativeType: "int"},
				"clCl":  {NativeType: "int"},
			},
			KeyElems: []*yang.Entry{{Name: "cl-cl"}, {Name: "clCl"}},
		},
		want: []string{"ClCl int", "ClCl_ int"},
	}, {
		name: "unsupported type",
		in: &ygen.YangListAttr{
			Keys:     map[string]*ygen.MappedType{"fluorine": {NativeType: "interface{}"}},
			KeyElems: []*yang.Entry{{Name: "fluorine"}},
		},
		want: []string{"Fluorine string"},
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
		want: []string{"Fluorine string", "ClCl int", "Bromine complex128", "Iodine float64"},
	}, {
		name: "enumerated and union parameters",
		in: &ygen.YangListAttr{
			Keys: map[string]*ygen.MappedType{
				"astatine":   {NativeType: "Halogen", IsEnumeratedValue: true},
				"tennessine": {NativeType: "Ununseptium", UnionTypes: map[string]int{"int32": 1, "float64": 2}},
			},
			KeyElems: []*yang.Entry{{Name: "astatine"}, {Name: "tennessine"}},
		},
		want: []string{"Astatine oc.Halogen", "Tennessine oc.Ununseptium"},
	}, {
		name: "Binary and Empty",
		in: &ygen.YangListAttr{
			Keys: map[string]*ygen.MappedType{
				"bromine": {NativeType: "Binary"},
				"cl-cl":   {NativeType: "YANGEmpty"},
			},
			KeyElems: []*yang.Entry{{Name: "cl-cl"}, {Name: "bromine"}},
		},
		want: []string{"ClCl oc.YANGEmpty", "Bromine oc.Binary"},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := makeParamListStrs(tt.in, "oc")
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("(-want, +got):\n%s", diff)
			}

			if diff := errdiff.Check(err, tt.wantErrSubstring); diff != "" {
				t.Errorf("func makeParamListStrs, %v", diff)
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
