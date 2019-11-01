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
	"testing"

	"github.com/google/go-cmp/cmp"
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
	name                string   // Name is the identifier for the test.
	inFiles             []string // inFiles is the set of inputFiles for the test.
	inIncludePaths      []string // inIncludePaths is the set of paths that should be searched for imports.
	wantStructsCodeFile string   // wantStructsCodeFile is the path of the generated Go code that the output of the test should be compared to.
	wantErr             bool     // wantErr specifies whether the test should expect an error.
}

func TestGeneratePathCode(t *testing.T) {
	tests := []yangTestCase{
		{
			name:                "simple openconfig test, with compression",
			inFiles:             []string{filepath.Join(datapath, "openconfig-simple.yang")},
			wantStructsCodeFile: filepath.Join(TestRoot, "testdata/structs/openconfig-simple.path-txt"),
		}, {
			name:                "simple openconfig test with list, with compression",
			inFiles:             []string{filepath.Join(datapath, "openconfig-withlist.yang")},
			wantStructsCodeFile: filepath.Join(TestRoot, "testdata/structs/openconfig-withlist.path-txt"),
		}, {
			name:                "simple openconfig test with union & typedef & identity & enum",
			inFiles:             []string{filepath.Join(datapath, "openconfig-unione.yang")},
			wantStructsCodeFile: filepath.Join(TestRoot, "testdata/structs/openconfig-unione.path-txt"),
		}, {
			name:                "simple openconfig test with submodule and union list key",
			inFiles:             []string{filepath.Join(datapath, "enum-module.yang")},
			wantStructsCodeFile: filepath.Join(TestRoot, "testdata/structs/enum-module.path-txt"),
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
			genCode := func() (string, *GenConfig) {
				cg := NewDefaultConfig("github.com/openconfig/ygot/ypathgen/testdata/exampleoc")
				// Set the name of the caller explicitly to avoid issues when
				// the unit tests are called by external test entities.
				cg.GeneratingBinary = "pathgen-tests"

				gotCode, err := cg.GeneratePathCode(tt.inFiles, tt.inIncludePaths)
				if err != nil && !tt.wantErr {
					t.Fatalf("GeneratePathCode(%v, %v): Config: %v, got unexpected error: %v, want: nil", tt.inFiles, tt.inIncludePaths, cg, err)
				}

				return gotCode.String(), cg
			}

			gotCode, cg := genCode()

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
				gotAttempt, _ := genCode()
				if gotAttempt != gotCode {
					diff, _ := testutil.GenerateUnifiedDiff(gotCode, gotAttempt)
					t.Fatalf("flaky code generation, diff:\n%s", diff)
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
func getSchemaAndDirs() (*yang.Entry, map[string]*ygen.Directory) {
	schema := &yang.Entry{
		Name: "root-module",
		Kind: yang.DirectoryEntry,
		Dir: map[string]*yang.Entry{
			"leaf": {
				Name: "leaf",
				Kind: yang.LeafEntry,
				Type: &yang.YangType{Kind: yang.Ybinary},
			},
			"container": {
				Name: "container",
				Kind: yang.DirectoryEntry,
				Dir: map[string]*yang.Entry{
					"leaf": {
						Name: "leaf",
						Kind: yang.LeafEntry,
						Type: &yang.YangType{Kind: yang.Yint32},
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
	fakeRoot := ygen.MakeFakeRoot("device")
	for k, v := range schema.Dir {
		fakeRoot.Dir[k] = v
	}

	directories := map[string]*ygen.Directory{
		"/device": {
			Name: "Device",
			Fields: map[string]*yang.Entry{
				"leaf":                  schema.Dir["leaf"],
				"container":             schema.Dir["container"],
				"container-with-config": schema.Dir["container-with-config"],
				"list":                  schema.Dir["list-container"].Dir["list"],
				"list-with-state":       schema.Dir["list-container-with-state"].Dir["list-with-state"],
			},
			Path:  []string{"", "device"},
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
				"leaf":     schema.Dir["container-with-config"].Dir["state"].Dir["leaf"],
				"leaflist": schema.Dir["container-with-config"].Dir["state"].Dir["leaflist"],
			},
			Path:  []string{"", "root-module", "container-with-config"},
			Entry: schema.Dir["container-with-config"],
		},
		"/root-module/list-container/list": {
			Name: "List",
			ListAttr: &ygen.YangListAttr{
				Keys: map[string]*ygen.MappedType{
					"key1":      &ygen.MappedType{NativeType: "string"},
					"key2":      &ygen.MappedType{NativeType: "Binary"},
					"union-key": &ygen.MappedType{NativeType: "RootModule_List_UnionKey_Union"},
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
					"key": &ygen.MappedType{NativeType: "float64"},
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

	return schema, directories
}

func TestGenerateDirectorySnippet(t *testing.T) {
	_, directories := getSchemaAndDirs()

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

// ContainerWithConfig_Leaf represents the /root-module/container-with-config/state/leaf YANG schema element.
type ContainerWithConfig_Leaf struct {
	ygot.NodePath
}

// ContainerWithConfig_Leaflist represents the /root-module/container-with-config/state/leaflist YANG schema element.
type ContainerWithConfig_Leaflist struct {
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
`,
		},
	}, {
		name:        "fakeroot",
		inDirectory: directories["/device"],
		want: GoPathStructCodeSnippet{
			PathStructName: "Device",
			StructBase: `
// Device represents the /device YANG schema element.
type Device struct {
	ygot.NodePath
	id string
}

func ForDevice(id string) *Device {
	return &Device{id: id}
}

// Leaf represents the /root-module/leaf YANG schema element.
type Leaf struct {
	ygot.NodePath
}
`,
			ChildConstructors: `
// Container returns from Device the path struct for its child "container".
func (n *Device) Container() *Container {
	return &Container{
		NodePath: ygot.NewNodePath(
			[]string{"container"},
			map[string]interface{}{},
			n,
		),
	}
}

// ContainerWithConfig returns from Device the path struct for its child "container-with-config".
func (n *Device) ContainerWithConfig() *ContainerWithConfig {
	return &ContainerWithConfig{
		NodePath: ygot.NewNodePath(
			[]string{"container-with-config"},
			map[string]interface{}{},
			n,
		),
	}
}

// Leaf returns from Device the path struct for its child "leaf".
func (n *Device) Leaf() *Leaf {
	return &Leaf{
		NodePath: ygot.NewNodePath(
			[]string{"leaf"},
			map[string]interface{}{},
			n,
		),
	}
}

// List returns from Device the path struct for its child "list".
func (n *Device) List(Key1 string, Key2 oc.Binary, UnionKey RootModule_List_UnionKey_Union) *List {
	return &List{
		NodePath: ygot.NewNodePath(
			[]string{"list-container", "list"},
			map[string]interface{}{"key1": Key1, "key2": Key2, "union-key": UnionKey},
			n,
		),
	}
}

// ListWithState returns from Device the path struct for its child "list-with-state".
func (n *Device) ListWithState(Key float64) *ListWithState {
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

// List_Key1 represents the /root-module/list-container/list/key1 YANG schema element.
type List_Key1 struct {
	ygot.NodePath
}

// List_Key2 represents the /root-module/list-container/list/key2 YANG schema element.
type List_Key2 struct {
	ygot.NodePath
}

// List_UnionKey represents the /root-module/list-container/list/union-key YANG schema element.
type List_UnionKey struct {
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
	_, directories := getSchemaAndDirs()

	tests := []struct {
		name              string
		inDirectory       *ygen.Directory
		inFieldName       string
		inUniqueFieldName string
		wantErrSubstr     string
		want              string
	}{{
		name:              "container method",
		inDirectory:       directories["/device"],
		inFieldName:       "container",
		inUniqueFieldName: "Container",
		want: `
// Container returns from Device the path struct for its child "container".
func (n *Device) Container() *Container {
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
`,
	}, {
		name:              "top-level leaf method",
		inDirectory:       directories["/device"],
		inFieldName:       "leaf",
		inUniqueFieldName: "Leaf",
		want: `
// Leaf returns from Device the path struct for its child "leaf".
func (n *Device) Leaf() *Leaf {
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
`,
	}, {
		name:              "list method",
		inDirectory:       directories["/device"],
		inFieldName:       "list",
		inUniqueFieldName: "List",
		want: `
// List returns from Device the path struct for its child "list".
func (n *Device) List(Key1 string, Key2 oc.Binary, UnionKey RootModule_List_UnionKey_Union) *List {
	return &List{
		NodePath: ygot.NewNodePath(
			[]string{"list-container", "list"},
			map[string]interface{}{"key1": Key1, "key2": Key2, "union-key": UnionKey},
			n,
		),
	}
}
`,
	}, {
		name:              "list with state method",
		inDirectory:       directories["/device"],
		inFieldName:       "list-with-state",
		inUniqueFieldName: "ListWithState",
		want: `
// ListWithState returns from Device the path struct for its child "list-with-state".
func (n *Device) ListWithState(Key float64) *ListWithState {
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
			var gotErr error
			gotErr = generateChildConstructor(&buf, tt.inDirectory, tt.inFieldName, tt.inUniqueFieldName, directories, "oc")
			if diff := errdiff.Check(gotErr, tt.wantErrSubstr); diff != "" {
				t.Fatalf("func generateChildConstructor, %v", diff)
			}

			if got, want := buf.String(), tt.want; got != want {
				diff, _ := testutil.GenerateUnifiedDiff(got, want)
				t.Errorf("func generateChildConstructor returned incorrect code, diff:\n%s", diff)
			}
		})
	}
}

func TestGenerateParamListStr(t *testing.T) {
	tests := []struct {
		name             string
		in               *ygen.YangListAttr
		want             string
		wantErrSubstring string
	}{{
		name:             "empty listattr",
		in:               &ygen.YangListAttr{},
		wantErrSubstring: "invalid list - has no key",
	}, {
		name: "simple string param",
		in: &ygen.YangListAttr{
			Keys:     map[string]*ygen.MappedType{"fluorine": &ygen.MappedType{NativeType: "string"}},
			KeyElems: []*yang.Entry{{Name: "fluorine"}},
		},
		want: "Fluorine string",
	}, {
		name: "simple int param, also testing camel-case",
		in: &ygen.YangListAttr{
			Keys:     map[string]*ygen.MappedType{"cl-cl": &ygen.MappedType{NativeType: "int"}},
			KeyElems: []*yang.Entry{{Name: "cl-cl"}},
		},
		want: "ClCl int",
	}, {
		name: "name uniquification",
		in: &ygen.YangListAttr{
			Keys: map[string]*ygen.MappedType{
				"cl-cl": &ygen.MappedType{NativeType: "int"},
				"clCl":  &ygen.MappedType{NativeType: "int"},
			},
			KeyElems: []*yang.Entry{{Name: "cl-cl"}, {Name: "clCl"}},
		},
		want: "ClCl int, ClCl_ int",
	}, {
		name: "unsupported type",
		in: &ygen.YangListAttr{
			Keys:     map[string]*ygen.MappedType{"fluorine": &ygen.MappedType{NativeType: "interface{}"}},
			KeyElems: []*yang.Entry{{Name: "fluorine"}},
		},
		want: "Fluorine string",
	}, {
		name: "keyElems doesn't match keys",
		in: &ygen.YangListAttr{
			Keys:     map[string]*ygen.MappedType{"neon": &ygen.MappedType{NativeType: "light"}},
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
				"bromine":  &ygen.MappedType{NativeType: "complex128"},
				"cl-cl":    &ygen.MappedType{NativeType: "int"},
				"fluorine": &ygen.MappedType{NativeType: "string"},
				"iodine":   &ygen.MappedType{NativeType: "float64"},
			},
			KeyElems: []*yang.Entry{{Name: "fluorine"}, {Name: "cl-cl"}, {Name: "bromine"}, {Name: "iodine"}},
		},
		want: "Fluorine string, ClCl int, Bromine complex128, Iodine float64",
	}, {
		name: "enumerated and union parameters",
		in: &ygen.YangListAttr{
			Keys: map[string]*ygen.MappedType{
				"astatine":   &ygen.MappedType{NativeType: "Halogen", IsEnumeratedValue: true},
				"tennessine": &ygen.MappedType{NativeType: "Ununseptium", UnionTypes: map[string]int{"int32": 1, "float64": 2}},
			},
			KeyElems: []*yang.Entry{{Name: "astatine"}, {Name: "tennessine"}},
		},
		want: "Astatine oc.Halogen, Tennessine oc.Ununseptium",
	}, {
		name: "Binary and Empty",
		in: &ygen.YangListAttr{
			Keys: map[string]*ygen.MappedType{
				"bromine": &ygen.MappedType{NativeType: "Binary"},
				"cl-cl":   &ygen.MappedType{NativeType: "YANGEmpty"},
			},
			KeyElems: []*yang.Entry{{Name: "cl-cl"}, {Name: "bromine"}},
		},
		want: "ClCl oc.YANGEmpty, Bromine oc.Binary",
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := makeParamListStr(tt.in, "oc")
			if got != tt.want {
				t.Errorf("func makeParamListStr\nwant: %s\ngot: %s", tt.want, got)
			}

			if diff := errdiff.Check(err, tt.wantErrSubstring); diff != "" {
				t.Errorf("func makeParamListStr, %v", diff)
			}
		})
	}
}
