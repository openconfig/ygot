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
	"testing"

	"github.com/openconfig/gnmi/errdiff"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/testutil"
	"github.com/openconfig/ygot/ygen"
)

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
	fakeRoot := &yang.Entry{
		Name: "device",
		Kind: yang.DirectoryEntry,
		Dir:  map[string]*yang.Entry{},
	}
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
			gotErr = generateChildConstructor(&buf, tt.inDirectory, tt.inFieldName, tt.inUniqueFieldName, directories)
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
			got, err := makeParamListStr(tt.in)
			if got != tt.want {
				t.Errorf("func makeParamListStr\nwant: %s\ngot: %s", tt.want, got)
			}

			if diff := errdiff.Check(err, tt.wantErrSubstring); diff != "" {
				t.Errorf("func makeParamListStr, %v", diff)
			}
		})
	}
}
