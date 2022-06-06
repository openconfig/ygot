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
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sort"
	"testing"

	"github.com/kylelemons/godebug/pretty"
	"github.com/openconfig/gnmi/errdiff"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/genutil"
	"github.com/openconfig/ygot/internal/igenutil"
	"github.com/openconfig/ygot/testutil"
)

const (
	// TestRoot is the root of the test directory such that this is not
	// repeated when referencing files.
	TestRoot string = ""
	// deflakeRuns specifies the number of runs of code generation that
	// should be performed to check for flakes.
	deflakeRuns int = 10
)

// datapath is the path to common YANG test modules.
const datapath = "../testdata/modules"

// TestFindMappableEntities tests the extraction of elements that are to be mapped
// into Go code from a YANG schema.
func TestFindMappableEntities(t *testing.T) {
	tests := []struct {
		name          string        // name is an identifier for the test.
		in            *yang.Entry   // in is the yang.Entry corresponding to the YANG root element.
		inSkipModules []string      // inSkipModules is a slice of strings indicating modules to be skipped.
		inModules     []*yang.Entry // inModules is the set of modules that the code generation is for.
		// wantCompressed is a map keyed by the string "structs" or "enums" which contains a slice
		// of the YANG identifiers for the corresponding mappable entities that should be
		// found. wantCompressed is the set that are expected when compression is enabled.
		wantCompressed map[string][]string
		// wantUncompressed is a map of the same form as wantCompressed. It is the expected
		// result when compression is disabled.
		wantUncompressed map[string][]string
	}{{
		name: "base-test",
		in: &yang.Entry{
			Name: "module",
			Kind: yang.DirectoryEntry,
			Dir: map[string]*yang.Entry{
				"base": {
					Name: "base",
					Kind: yang.DirectoryEntry,
					Dir: map[string]*yang.Entry{
						"config": {
							Name: "config",
							Kind: yang.DirectoryEntry,
							Dir:  map[string]*yang.Entry{},
						},
						"state": {
							Name: "state",
							Kind: yang.DirectoryEntry,
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
					Kind: yang.DirectoryEntry,
					Dir: map[string]*yang.Entry{
						"config": {
							Name: "config",
							Kind: yang.DirectoryEntry,
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
							Kind: yang.DirectoryEntry,
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
			Kind: yang.DirectoryEntry,
			Dir: map[string]*yang.Entry{
				"ignored-container": {
					Name: "ignored-container",
					Kind: yang.DirectoryEntry,
					Dir:  map[string]*yang.Entry{},
					Node: &yang.Container{
						Name: "ignored-container",
						Parent: &yang.Module{
							Namespace: &yang.Value{
								Name: "module-namespace",
							},
						},
					},
				},
			},
			Node: &yang.Module{
				Namespace: &yang.Value{
					Name: "module-namespace",
				},
			},
		},
		inSkipModules: []string{"module"},
		inModules: []*yang.Entry{{
			Name: "module",
			Node: &yang.Module{
				Namespace: &yang.Value{
					Name: "module-namespace",
				},
			},
		}},
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
			Kind: yang.DirectoryEntry,
			Dir: map[string]*yang.Entry{
				"surrounding-container": {
					Name: "surrounding-container",
					Kind: yang.DirectoryEntry,
					Dir: map[string]*yang.Entry{
						"child-list": {
							Name:     "child-list",
							Kind:     yang.DirectoryEntry,
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
									Kind: yang.DirectoryEntry,
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
			Kind: yang.DirectoryEntry,
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
			Kind: yang.DirectoryEntry,
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
			Kind: yang.DirectoryEntry,
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
			Kind: yang.DirectoryEntry,
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
									Kind: yang.DirectoryEntry,
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
		wantCompressed: map[string][]string{
			"structs": {"container"},
			"enums":   {"choice-case-container-leaf", "choice-case2-leaf", "direct"}},
		wantUncompressed: map[string][]string{
			"structs": {"container"},
			"enums":   {"choice-case-container-leaf", "choice-case2-leaf", "direct"}},
	}}

	for _, tt := range tests {
		testSpec := map[bool]map[string][]string{
			true:  tt.wantCompressed,
			false: tt.wantUncompressed,
		}

		for compress, expected := range testSpec {
			structs := make(map[string]*yang.Entry)
			enums := make(map[string]*yang.Entry)

			errs := findMappableEntities(tt.in, structs, enums, tt.inSkipModules, compress, tt.inModules)
			if errs != nil {
				t.Errorf("%s: findMappableEntities(compressEnabled: %v): got unexpected error, got: %v, want: nil", tt.name, compress, errs)
			}

			entityNames := func(m map[string]bool) []string {
				o := []string{}
				for k := range m {
					o = append(o, k)
				}
				return o
			}

			structOut := make(map[string]bool)
			enumOut := make(map[string]bool)
			for _, o := range structs {
				structOut[o.Name] = true
			}
			for _, e := range enums {
				enumOut[e.Name] = true
			}

			if len(expected["structs"]) != len(structOut) {
				t.Errorf("%s: findMappableEntities(compressEnabled: %v): did not get expected number of structs, got: %v, want: %v", tt.name, compress, entityNames(structOut), expected["structs"])
			}

			for _, e := range expected["structs"] {
				if !structOut[e] {
					t.Errorf("%s: findMappableEntities(compressEnabled: %v): struct %s was not found in %v\n", tt.name, compress, e, structOut)
				}
			}

			if len(expected["enums"]) != len(enumOut) {
				t.Errorf("%s: findMappableEntities(compressEnabled: %v): did not get expected number of enums, got: %v, want: %v", tt.name, compress, entityNames(enumOut), expected["enums"])
			}

			for _, e := range expected["enums"] {
				if !enumOut[e] {
					t.Errorf("%s: findMappableEntities(compressEnabled: %v): enum %s was not found in %v\n", tt.name, compress, e, enumOut)
				}
			}
		}
	}
}

func TestGenerateErrs(t *testing.T) {
	tests := []struct {
		name                  string
		inFiles               []string
		inPath                []string
		inConfig              GeneratorConfig
		wantGoOK              bool
		wantGoErrSubstring    string
		wantProtoOK           bool
		wantProtoErrSubstring string
		wantSameErrSubstring  bool
	}{{
		name:                 "missing YANG file",
		inFiles:              []string{filepath.Join(TestRoot, "testdata", "errors", "doesnt-exist.yang")},
		wantGoErrSubstring:   "no such file",
		wantSameErrSubstring: true,
	}, {
		name:                 "bad YANG file",
		inFiles:              []string{filepath.Join(TestRoot, "testdata", "errors", "bad-module.yang")},
		wantGoErrSubstring:   "syntax error",
		wantSameErrSubstring: true,
	}, {
		name:                 "missing import due to path",
		inFiles:              []string{filepath.Join(TestRoot, "testdata", "errors", "missing-import.yang")},
		wantGoErrSubstring:   "no such module",
		wantSameErrSubstring: true,
	}, {
		name:        "import satisfied due to path",
		inFiles:     []string{filepath.Join(TestRoot, "testdata", "errors", "missing-import.yang")},
		inPath:      []string{filepath.Join(TestRoot, "testdata", "errors", "subdir")},
		wantGoOK:    true,
		wantProtoOK: true,
	}}

	for _, tt := range tests {
		cg := NewYANGCodeGenerator(&tt.inConfig)

		// TODO(wenbli): Move this to integration_tests.
		/*
			_, goErr := cg.GenerateGoCode(tt.inFiles, tt.inPath)
			switch {
			case tt.wantGoOK && goErr != nil:
				t.Errorf("%s: cg.GenerateGoCode(%v, %v): got unexpected error, got: %v, want: nil", tt.name, tt.inFiles, tt.inPath, goErr)
			case tt.wantGoOK:
			default:
				if diff := errdiff.Substring(goErr, tt.wantGoErrSubstring); diff != "" {
					t.Errorf("%s: cg.GenerateGoCode(%v, %v): %v", tt.name, tt.inFiles, tt.inPath, diff)
				}
			}
		*/

		if tt.wantSameErrSubstring {
			tt.wantProtoErrSubstring = tt.wantGoErrSubstring
		}

		_, protoErr := cg.GenerateProto3(tt.inFiles, tt.inPath)
		switch {
		case tt.wantProtoOK && protoErr != nil:
			t.Errorf("%s: cg.GenerateProto3(%v, %v): got unexpected error, got: %v, want: nil", tt.name, tt.inFiles, tt.inPath, protoErr)
		case tt.wantProtoOK:
		default:
			if diff := errdiff.Substring(protoErr, tt.wantProtoErrSubstring); diff != "" {
				t.Errorf("%s: cg.GenerateProto3(%v, %v): %v", tt.name, tt.inFiles, tt.inPath, diff)
			}
		}

	}
}

func TestFindRootEntries(t *testing.T) {
	tests := []struct {
		name                       string
		inStructs                  map[string]*yang.Entry
		inRootElems                []*yang.Entry
		inRootName                 string
		wantCompressRootChildren   []string
		wantUncompressRootChildren []string
	}{{
		name: "directory at root",
		inStructs: map[string]*yang.Entry{
			"/foo": {
				Name: "foo",
				Dir:  map[string]*yang.Entry{},
				Kind: yang.DirectoryEntry,
				Parent: &yang.Entry{
					Name: "module",
				},
			},
			"/foo/bar": {
				Name: "bar",
				Dir:  map[string]*yang.Entry{},
				Kind: yang.DirectoryEntry,
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
	}, {
		name: "directory and leaf at root",
		inStructs: map[string]*yang.Entry{
			"/foo": {
				Name: "foo",
				Dir:  map[string]*yang.Entry{},
				Kind: yang.DirectoryEntry,
				Parent: &yang.Entry{
					Name: "module",
				},
			},
		},
		inRootElems: []*yang.Entry{{
			Name: "foo",
			Dir:  map[string]*yang.Entry{},
			Kind: yang.DirectoryEntry,
			Parent: &yang.Entry{
				Name: "module",
			},
		}, {
			Name: "leaf",
			Type: &yang.YangType{
				Kind: yang.Ystring,
			},
			Parent: &yang.Entry{
				Name: "module",
			},
		}},
		inRootName:                 "fakeroot",
		wantCompressRootChildren:   []string{"foo", "leaf"},
		wantUncompressRootChildren: []string{"foo", "leaf"},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for compress, wantChildren := range map[bool][]string{true: tt.wantCompressRootChildren, false: tt.wantUncompressRootChildren} {
				if err := createFakeRoot(tt.inStructs, tt.inRootElems, tt.inRootName, compress); err != nil {
					t.Errorf("cg.createFakeRoot(%v), compressEnabled: %v, got unexpected error: %v", tt.inStructs, compress, err)
					continue
				}

				rootElem, ok := tt.inStructs["/"]
				if !ok {
					t.Errorf("cg.createFakeRoot(%v), compressEnabled: %v, could not find root element", tt.inStructs, compress)
					continue
				}

				gotChildren := map[string]bool{}
				for n := range rootElem.Dir {
					gotChildren[n] = true
				}

				for _, ch := range wantChildren {
					if _, ok := rootElem.Dir[ch]; !ok {
						t.Errorf("cg.createFakeRoot(%v), compressEnabled: %v, could not find child %v in %v", tt.inStructs, compress, ch, rootElem.Dir)
					}
					gotChildren[ch] = false
				}

				for ch, ok := range gotChildren {
					if ok == true {
						t.Errorf("cg.findRootentries(%v), compressEnabled: %v, did not expect child %v", tt.inStructs, compress, ch)
					}
				}
			}
		})
	}
}

func TestGenerateProto3(t *testing.T) {
	tests := []struct {
		name           string
		inFiles        []string
		inIncludePaths []string
		inConfig       GeneratorConfig
		// wantOutputFiles is a map keyed on protobuf package name with a path
		// to the file that is expected for each package.
		wantOutputFiles map[string]string
		wantErr         bool
	}{{
		name:    "simple protobuf test with compression",
		inFiles: []string{filepath.Join(TestRoot, "testdata", "proto", "proto-test-a.yang")},
		inConfig: GeneratorConfig{
			TransformationOptions: TransformationOpts{
				CompressBehaviour: genutil.PreferIntendedConfig,
			},
		},
		wantOutputFiles: map[string]string{
			"openconfig":        filepath.Join(TestRoot, "testdata", "proto", "proto-test-a.compress.parent.formatted-txt"),
			"openconfig.parent": filepath.Join(TestRoot, "testdata", "proto", "proto-test-a.compress.parent.child.formatted-txt"),
		},
	}, {
		name:    "simple protobuf test without compression",
		inFiles: []string{filepath.Join(TestRoot, "testdata", "proto", "proto-test-a.yang")},
		wantOutputFiles: map[string]string{
			"openconfig.proto_test_a":              filepath.Join(TestRoot, "testdata", "proto", "proto-test-a.nocompress.formatted-txt"),
			"openconfig.proto_test_a.parent":       filepath.Join(TestRoot, "testdata", "proto", "proto-test-a.nocompress.parent.formatted-txt"),
			"openconfig.proto_test_a.parent.child": filepath.Join(TestRoot, "testdata", "proto", "proto-test-a.nocompress.parent.child.formatted-txt"),
		},
	}, {
		name:    "enumeration under unions test with compression",
		inFiles: []string{filepath.Join(datapath, "enum-union.yang")},
		inConfig: GeneratorConfig{
			TransformationOptions: TransformationOpts{
				CompressBehaviour:                    genutil.PreferIntendedConfig,
				GenerateFakeRoot:                     true,
				UseDefiningModuleForTypedefEnumNames: true,
			},
			ProtoOptions: ProtoOpts{
				AnnotateEnumNames: true,
				NestedMessages:    true,
				GoPackageBase:     "github.com/foo/bar",
			},
		},
		wantOutputFiles: map[string]string{
			"openconfig":       filepath.Join(TestRoot, "testdata", "proto", "enum-union.compress.formatted-txt"),
			"openconfig.enums": filepath.Join(TestRoot, "testdata", "proto", "enum-union.compress.enums.formatted-txt"),
		},
	}, {
		name:     "yang schema with a list",
		inFiles:  []string{filepath.Join(TestRoot, "testdata", "proto", "proto-test-b.yang")},
		inConfig: GeneratorConfig{TransformationOptions: TransformationOpts{CompressBehaviour: genutil.PreferIntendedConfig}},
		wantOutputFiles: map[string]string{
			"openconfig":        filepath.Join(TestRoot, "testdata", "proto", "proto-test-b.compress.formatted-txt"),
			"openconfig.device": filepath.Join(TestRoot, "testdata", "proto", "proto-test-b.compress.device.formatted-txt"),
		},
	}, {
		name:    "yang schema with simple enumerations",
		inFiles: []string{filepath.Join(TestRoot, "testdata", "proto", "proto-test-c.yang")},
		inConfig: GeneratorConfig{
			ProtoOptions: ProtoOpts{
				GoPackageBase: "github.com/foo/baz",
			},
		},
		wantOutputFiles: map[string]string{
			"openconfig.proto_test_c":              filepath.Join(TestRoot, "testdata", "proto", "proto-test-c.proto-test-c.formatted-txt"),
			"openconfig.proto_test_c.entity":       filepath.Join(TestRoot, "testdata", "proto", "proto-test-c.proto-test-c.entity.formatted-txt"),
			"openconfig.proto_test_c.elists":       filepath.Join(TestRoot, "testdata", "proto", "proto-test-c.proto-test-c.elists.formatted-txt"),
			"openconfig.proto_test_c.elists.elist": filepath.Join(TestRoot, "testdata", "proto", "proto-test-c.proto-test-c.elists.elist.formatted-txt"),
			"openconfig.enums":                     filepath.Join(TestRoot, "testdata", "proto", "proto-test-c.enums.formatted-txt"),
		},
	}, {
		name:    "yang schema with identityref and enumerated typedef, compression off",
		inFiles: []string{filepath.Join(TestRoot, "testdata", "proto", "proto-test-d.yang")},
		wantOutputFiles: map[string]string{
			"openconfig.proto_test_d":      filepath.Join(TestRoot, "testdata", "proto", "proto-test-d.uncompressed.proto-test-d.formatted-txt"),
			"openconfig.proto_test_d.test": filepath.Join(TestRoot, "testdata", "proto", "proto-test-d.uncompressed.proto-test-d.test.formatted-txt"),
			"openconfig.enums":             filepath.Join(TestRoot, "testdata", "proto", "proto-test-d.uncompressed.enums.formatted-txt"),
		},
	}, {
		name:    "yang schema with unions",
		inFiles: []string{filepath.Join(TestRoot, "testdata", "proto", "proto-test-e.yang")},
		inConfig: GeneratorConfig{
			TransformationOptions: TransformationOpts{
				UseDefiningModuleForTypedefEnumNames: true,
			},
		},
		wantOutputFiles: map[string]string{
			"openconfig.proto_test_e":                filepath.Join(TestRoot, "testdata", "proto", "proto-test-e.uncompressed.proto-test-e.formatted-txt"),
			"openconfig.proto_test_e.test":           filepath.Join(TestRoot, "testdata", "proto", "proto-test-e.uncompressed.proto-test-e.test.formatted-txt"),
			"openconfig.proto_test_e.foos":           filepath.Join(TestRoot, "testdata", "proto", "proto-test-e.uncompressed.proto-test-e.foos.formatted-txt"),
			"openconfig.proto_test_e.foos.foo":       filepath.Join(TestRoot, "testdata", "proto", "proto-test-e.uncompressed.proto-test-e.foos.foo.formatted-txt"),
			"openconfig.proto_test_e.bars":           filepath.Join(TestRoot, "testdata", "proto", "proto-test-e.uncompressed.proto-test-e.bars.formatted-txt"),
			"openconfig.enums":                       filepath.Join(TestRoot, "testdata", "proto", "proto-test-e.uncompressed.enums.formatted-txt"),
			"openconfig.proto_test_e.animals":        filepath.Join(TestRoot, "testdata", "proto", "proto-test-e.uncompressed.proto-test-e.animals.formatted-txt"),
			"openconfig.proto_test_e.animals.animal": filepath.Join(TestRoot, "testdata", "proto", "proto-test-e.uncompressed.proto-test-e.animals.animal.formatted-txt"),
		},
	}, {
		name:    "yang schema with anydata",
		inFiles: []string{filepath.Join(TestRoot, "testdata", "proto", "proto-anydata-test.yang")},
		wantOutputFiles: map[string]string{
			"openconfig.proto_anydata_test":   filepath.Join(TestRoot, "testdata", "proto", "proto_anydata_test.formatted-txt"),
			"openconfig.proto_anydata_test.e": filepath.Join(TestRoot, "testdata", "proto", "proto_anydata_test.e.formatted-txt"),
		},
	}, {
		name:    "yang schema with path annotations",
		inFiles: []string{filepath.Join(TestRoot, "testdata", "proto", "proto-test-f.yang")},
		inConfig: GeneratorConfig{
			ProtoOptions: ProtoOpts{
				AnnotateSchemaPaths: true,
			},
		},
		wantOutputFiles: map[string]string{
			"openconfig.proto_test_f":     filepath.Join(TestRoot, "testdata", "proto", "proto_test_f.uncompressed.proto_test_f.formatted-txt"),
			"openconfig.proto_test_f.a":   filepath.Join(TestRoot, "testdata", "proto", "proto_test_f.uncompressed.proto_test_f.a.formatted-txt"),
			"openconfig.proto_test_f.a.c": filepath.Join(TestRoot, "testdata", "proto", "proto_test_f.uncompressed.proto_test_f.a.c.formatted-txt"),
		},
	}, {
		name:    "yang schema with leafrefs that point to the same path",
		inFiles: []string{filepath.Join(TestRoot, "testdata", "proto", "proto-test-g.yang")},
		inConfig: GeneratorConfig{
			ProtoOptions: ProtoOpts{
				GoPackageBase:  "github.com/foo/baz",
				NestedMessages: true,
			},
		},
		wantOutputFiles: map[string]string{
			"openconfig.proto_test_g": filepath.Join(TestRoot, "testdata", "proto", "proto-test-g.proto-test-g.formatted-txt"),
		},
	}, {
		name:    "yang schema with fake root, path compression and union list key",
		inFiles: []string{filepath.Join(TestRoot, "testdata", "proto", "proto-union-list-key.yang")},
		inConfig: GeneratorConfig{
			TransformationOptions: TransformationOpts{
				CompressBehaviour: genutil.PreferIntendedConfig,
				GenerateFakeRoot:  true,
			},
			ProtoOptions: ProtoOpts{
				AnnotateSchemaPaths: true,
			},
		},
		wantOutputFiles: map[string]string{
			"openconfig":                filepath.Join(TestRoot, "testdata", "proto", "proto-union-list-key.compressed.openconfig.formatted-txt"),
			"openconfig.routing_policy": filepath.Join(TestRoot, "testdata", "proto", "proto-union-list-key.compressed.openconfig.routing_policy.formatted-txt"),
		},
	}, {
		name:    "yang schema with fakeroot, and union list key",
		inFiles: []string{filepath.Join(TestRoot, "testdata", "proto", "proto-union-list-key.yang")},
		inConfig: GeneratorConfig{
			TransformationOptions: TransformationOpts{
				GenerateFakeRoot: true,
			},
			ProtoOptions: ProtoOpts{
				AnnotateSchemaPaths: true,
			},
		},
		wantOutputFiles: map[string]string{
			"openconfig":                                                     filepath.Join(TestRoot, "testdata", "proto", "proto-union-list_key.uncompressed.openconfig.formatted-txt"),
			"openconfig.proto_union_list_key":                                filepath.Join(TestRoot, "testdata", "proto", "proto-union-list-key.uncompressed.openconfig.proto_union_list_key.formatted-txt"),
			"openconfig.proto_union_list_key.routing_policy":                 filepath.Join(TestRoot, "testdata", "proto", "proto-union-list-key.uncompressed.openconfig.proto_union_list_key.routing_policy.formatted-txt"),
			"openconfig.proto_union_list_key.routing_policy.policies":        filepath.Join(TestRoot, "testdata", "proto", "proto-union-list-key.uncompressed.openconfig.proto_union_list_key.routing_policy.policies.formatted-txt"),
			"openconfig.proto_union_list_key.routing_policy.policies.policy": filepath.Join(TestRoot, "testdata", "proto", "proto-union-list-key.uncompressed.openconfig.proto_union_list_key.routing_policy.policies.policy.formatted-txt"),
			"openconfig.proto_union_list_key.routing_policy.sets":            filepath.Join(TestRoot, "testdata", "proto", "proto-union-list-key.uncompressed.openconfig.proto_union_list_key.routing_policy.sets.formatted-txt"),
		},
	}, {
		name:    "enums: yang schema with various types of enums with underscores",
		inFiles: []string{filepath.Join(TestRoot, "testdata", "proto", "proto-enums.yang")},
		inConfig: GeneratorConfig{
			TransformationOptions: TransformationOpts{
				UseDefiningModuleForTypedefEnumNames: true,
			},
		},
		wantOutputFiles: map[string]string{
			"openconfig.enums":       filepath.Join(TestRoot, "testdata", "proto", "proto-enums.enums.formatted-txt"),
			"openconfig.proto_enums": filepath.Join(TestRoot, "testdata", "proto", "proto-enums.formatted-txt"),
		},
	}, {
		name: "enums: yang schema with identity that adds to previous module",
		inFiles: []string{
			filepath.Join(TestRoot, "testdata", "proto", "proto-enums.yang"),
			filepath.Join(TestRoot, "testdata", "proto", "proto-enums-addid.yang"),
		},
		inConfig: GeneratorConfig{
			TransformationOptions: TransformationOpts{
				UseDefiningModuleForTypedefEnumNames: true,
			},
			ProtoOptions: ProtoOpts{
				AnnotateEnumNames: true,
			},
		},
		wantOutputFiles: map[string]string{
			"openconfig.enums":       filepath.Join(TestRoot, "testdata", "proto", "proto-enums-addid.enums.formatted-txt"),
			"openconfig.proto_enums": filepath.Join(TestRoot, "testdata", "proto", "proto-enums-addid.formatted-txt"),
		},
	}, {
		name: "yang schema with nested messages requested - uncompressed with fakeroot",
		inFiles: []string{
			filepath.Join(TestRoot, "testdata", "proto", "nested-messages.yang"),
		},
		inConfig: GeneratorConfig{
			TransformationOptions: TransformationOpts{
				GenerateFakeRoot:                     true,
				UseDefiningModuleForTypedefEnumNames: true,
			},
			ProtoOptions: ProtoOpts{
				AnnotateEnumNames:   true,
				AnnotateSchemaPaths: true,
				NestedMessages:      true,
			},
		},
		wantOutputFiles: map[string]string{
			"openconfig":                 filepath.Join(TestRoot, "testdata", "proto", "nested-messages.openconfig.formatted-txt"),
			"openconfig.enums":           filepath.Join(TestRoot, "testdata", "proto", "nested-messages.enums.formatted-txt"),
			"openconfig.nested_messages": filepath.Join(TestRoot, "testdata", "proto", "nested-messages.nested_messages.formatted-txt"),
		},
	}, {
		name: "yang schema with nested messages - compressed with fakeroot",
		inFiles: []string{
			filepath.Join(TestRoot, "testdata", "proto", "nested-messages.yang"),
		},
		inConfig: GeneratorConfig{
			TransformationOptions: TransformationOpts{
				CompressBehaviour:                    genutil.PreferIntendedConfig,
				IgnoreShadowSchemaPaths:              true,
				GenerateFakeRoot:                     true,
				UseDefiningModuleForTypedefEnumNames: true,
			},
			ProtoOptions: ProtoOpts{
				AnnotateEnumNames:   true,
				AnnotateSchemaPaths: true,
				NestedMessages:      true,
			},
		},
		wantOutputFiles: map[string]string{
			"openconfig.enums": filepath.Join(TestRoot, "testdata", "proto", "nested-messages.compressed.enums.formatted-txt"),
			"openconfig":       filepath.Join(TestRoot, "testdata", "proto", "nested-messages.compressed.nested_messages.formatted-txt"),
		},
	}, {
		name: "yang schema with a leafref key to a union with enumeration",
		inFiles: []string{
			filepath.Join(TestRoot, "testdata", "proto", "union-list-key.yang"),
		},
		inConfig: GeneratorConfig{
			TransformationOptions: TransformationOpts{
				GenerateFakeRoot:                     true,
				UseDefiningModuleForTypedefEnumNames: true,
			},
			ProtoOptions: ProtoOpts{
				AnnotateEnumNames:   true,
				AnnotateSchemaPaths: true,
				NestedMessages:      true,
			},
		},
		wantOutputFiles: map[string]string{
			"openconfig.enums":          filepath.Join(TestRoot, "testdata", "proto", "union-list-key.enums.formatted-txt"),
			"openconfig.union_list_key": filepath.Join(TestRoot, "testdata", "proto", "union-list-key.union_list_key.formatted-txt"),
			"openconfig":                filepath.Join(TestRoot, "testdata", "proto", "union-list-key.formatted-txt"),
		},
	}, {
		name: "protobuf generation with excluded read only fields - compressed",
		inFiles: []string{
			filepath.Join(datapath, "openconfig-config-false.yang"),
		},
		inConfig: GeneratorConfig{
			TransformationOptions: TransformationOpts{
				GenerateFakeRoot:  true,
				CompressBehaviour: genutil.UncompressedExcludeDerivedState,
			},
			ProtoOptions: ProtoOpts{
				AnnotateEnumNames:   true,
				AnnotateSchemaPaths: true,
				NestedMessages:      true,
			},
		},
		wantOutputFiles: map[string]string{
			"openconfig":                         filepath.Join(TestRoot, "testdata", "proto", "excluded-config-false.compressed.formatted-txt"),
			"openconfig.openconfig_config_false": filepath.Join(TestRoot, "testdata", "proto", "excluded-config-false.config_false.compressed.formatted-txt"),
		},
	}, {
		name: "protobuf generation with excluded read only fields - compressed",
		inFiles: []string{
			filepath.Join(datapath, "openconfig-config-false.yang"),
		},
		inConfig: GeneratorConfig{
			TransformationOptions: TransformationOpts{
				GenerateFakeRoot:  true,
				CompressBehaviour: genutil.ExcludeDerivedState,
			},
			ProtoOptions: ProtoOpts{
				AnnotateEnumNames:   true,
				AnnotateSchemaPaths: true,
				NestedMessages:      true,
				GoPackageBase:       "github.com/openconfig/a/package",
			},
		},
		wantOutputFiles: map[string]string{
			"openconfig": filepath.Join(TestRoot, "testdata", "proto", "excluded-config-false.uncompressed.formatted-txt"),
		},
	}, {
		name: "protobuf generation with leafref to a module excluded by the test",
		inFiles: []string{
			filepath.Join(TestRoot, "testdata", "proto", "cross-ref-target.yang"),
			filepath.Join(TestRoot, "testdata", "proto", "cross-ref-src.yang"),
		},
		inConfig: GeneratorConfig{
			ParseOptions: ParseOpts{
				ExcludeModules: []string{"cross-ref-target"},
			},
			ProtoOptions: ProtoOpts{
				NestedMessages: true,
			},
		},
		wantOutputFiles: map[string]string{
			"openconfig.cross_ref_src": filepath.Join(TestRoot, "testdata", "proto", "cross-ref-src.formatted-txt"),
		},
	}, {
		name: "multimod with fakeroot and nested",
		inFiles: []string{
			filepath.Join(TestRoot, "testdata", "proto", "fakeroot-multimod-one.yang"),
			filepath.Join(TestRoot, "testdata", "proto", "fakeroot-multimod-two.yang"),
		},
		inConfig: GeneratorConfig{
			TransformationOptions: TransformationOpts{
				GenerateFakeRoot:  true,
				CompressBehaviour: genutil.PreferIntendedConfig,
			},
			ProtoOptions: ProtoOpts{
				NestedMessages:      true,
				AnnotateEnumNames:   true,
				AnnotateSchemaPaths: true,
			},
		},
		wantOutputFiles: map[string]string{
			"openconfig": filepath.Join(TestRoot, "testdata", "proto", "fakeroot-multimod.formatted-txt"),
		},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			sortedPkgNames := func(pkgs map[string]string) []string {
				wantPkgs := []string{}
				for k := range tt.wantOutputFiles {
					wantPkgs = append(wantPkgs, k)
				}
				sort.Strings(wantPkgs)
				return wantPkgs
			}

			genCode := func() *GeneratedProto3 {
				if tt.inConfig.Caller == "" {
					// Override the caller if it is not set, to ensure that test
					// output is deterministic.
					tt.inConfig.Caller = "codegen-tests"
				}

				cg := NewYANGCodeGenerator(&tt.inConfig)
				gotProto, err := cg.GenerateProto3(tt.inFiles, tt.inIncludePaths)
				if (err != nil) != tt.wantErr {
					t.Fatalf("cg.GenerateProto3(%v, %v), config: %v: got unexpected error: %v", tt.inFiles, tt.inIncludePaths, tt.inConfig, err)
				}

				if tt.wantErr || err != nil {
					return nil
				}

				return gotProto
			}

			gotProto := genCode()

			allCode := bytes.Buffer{}

			seenPkg := map[string]bool{}
			for n := range gotProto.Packages {
				seenPkg[n] = false
			}

			protoPkgs := func(m map[string]Proto3Package) []string {
				a := []string{}
				for k := range m {
					a = append(a, k)
				}
				return a
			}

			wantPkgs := sortedPkgNames(tt.wantOutputFiles)
			for _, pkg := range wantPkgs {
				wantFile := tt.wantOutputFiles[pkg]
				wantCodeBytes, err := ioutil.ReadFile(wantFile)
				if err != nil {
					t.Errorf("%s: ioutil.ReadFile(%v): could not read file for package %s", tt.name, wantFile, pkg)
					return
				}

				gotPkg, ok := gotProto.Packages[pkg]
				if !ok {
					t.Fatalf("%s: cg.GenerateProto3(%v, %v): did not find expected package %s in output, got: %#v, want key: %v", tt.name, tt.inFiles, tt.inIncludePaths, pkg, protoPkgs(gotProto.Packages), pkg)
				}

				// Mark this package as having been seen.
				seenPkg[pkg] = true

				// Write the returned struct out to a buffer to compare with the
				// testdata file.
				var gotCodeBuf bytes.Buffer
				fmt.Fprintf(&gotCodeBuf, gotPkg.Header)

				for _, gotMsg := range gotPkg.Messages {
					fmt.Fprintf(&gotCodeBuf, "%s\n", gotMsg)
				}

				for _, gotEnum := range gotPkg.Enums {
					fmt.Fprintf(&gotCodeBuf, "%s", gotEnum)
				}

				wantCode := string(wantCodeBytes)

				allCode.WriteString(gotCodeBuf.String())

				if diff := pretty.Compare(gotCodeBuf.String(), wantCode); diff != "" {
					if diffl, _ := testutil.GenerateUnifiedDiff(wantCode, gotCodeBuf.String()); diffl != "" {
						diff = diffl
					}
					t.Errorf("%s: cg.GenerateProto3(%v, %v) for package %s, did not get expected code (code file: %v), diff(-want, +got):\n%s", tt.name, tt.inFiles, tt.inIncludePaths, pkg, wantFile, diff)
				}
			}

			for pkg, seen := range seenPkg {
				if !seen {
					t.Errorf("%s: cg.GenerateProto3(%v, %v) did not test received package %v", tt.name, tt.inFiles, tt.inIncludePaths, pkg)
				}
			}

			for i := 0; i < deflakeRuns; i++ {
				got := genCode()
				var gotCodeBuf bytes.Buffer

				wantPkgs := sortedPkgNames(tt.wantOutputFiles)
				for _, pkg := range wantPkgs {
					gotPkg, ok := got.Packages[pkg]
					if !ok {
						t.Fatalf("%s: cg.GenerateProto3(%v, %v): did not find expected package %s in output, got: %#v, want key: %v", tt.name, tt.inFiles, tt.inIncludePaths, pkg, protoPkgs(gotProto.Packages), pkg)
					}
					fmt.Fprintf(&gotCodeBuf, gotPkg.Header)
					for _, gotMsg := range gotPkg.Messages {
						fmt.Fprintf(&gotCodeBuf, "%s\n", gotMsg)
					}
					for _, gotEnum := range gotPkg.Enums {
						fmt.Fprintf(&gotCodeBuf, "%s", gotEnum)
					}
				}

				if diff := pretty.Compare(gotCodeBuf.String(), allCode.String()); diff != "" {
					diff, _ = testutil.GenerateUnifiedDiff(allCode.String(), gotCodeBuf.String())
					t.Fatalf("flaky code generation iter: %d, diff(-want, +got):\n%s", i, diff)
				}
			}
		})
	}
}

func TestMakeFakeRoot(t *testing.T) {
	tests := []struct {
		name       string
		inRootName string
		want       *yang.Entry
	}{{
		name:       "simple empty root named device",
		inRootName: "device",
		want: &yang.Entry{
			Name: "device",
			Kind: yang.DirectoryEntry,
			Dir:  map[string]*yang.Entry{},
			Node: &yang.Value{
				Name: igenutil.RootElementNodeName,
			},
		},
	}, {
		name:       "simple root named !@#$",
		inRootName: "!@#$",
		want: &yang.Entry{
			Name: "!@#$",
			Kind: yang.DirectoryEntry,
			Dir:  map[string]*yang.Entry{},
			Node: &yang.Value{
				Name: igenutil.RootElementNodeName,
			},
		},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MakeFakeRoot(tt.inRootName)
			if diff := pretty.Compare(tt.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
			if !igenutil.IsFakeRoot(got) {
				t.Errorf("IsFakeRoot returned false for entry %v", got)
			}
		})
	}
}

func TestCreateFakeRoot(t *testing.T) {
	tests := []struct {
		name            string
		inStructs       map[string]*yang.Entry
		inRootElems     []*yang.Entry
		inRootName      string
		inCompressPaths bool
		wantRoot        *yang.Entry
		wantErr         bool
	}{{
		name: "simple root",
		inStructs: map[string]*yang.Entry{
			"/module/foo": {
				Name: "foo",
				Kind: yang.DirectoryEntry,
				Parent: &yang.Entry{
					Name: "module",
				},
			},
		},
		inRootElems: []*yang.Entry{{
			Name: "foo",
			Kind: yang.DirectoryEntry,
			Parent: &yang.Entry{
				Name: "module",
			},
		}, {
			Name: "bar",
			Parent: &yang.Entry{
				Name: "module",
			},
			Type: &yang.YangType{Kind: yang.Ystring},
		}},
		inRootName:      "",
		inCompressPaths: false,
		wantRoot: &yang.Entry{
			Name: igenutil.DefaultRootName,
			Kind: yang.DirectoryEntry,
			Dir: map[string]*yang.Entry{
				"foo": {
					Name: "foo",
					Kind: yang.DirectoryEntry,
					Parent: &yang.Entry{
						Name: "module",
					},
				},
				"bar": {
					Name: "bar",
					Parent: &yang.Entry{
						Name: "module",
					},
					Type: &yang.YangType{Kind: yang.Ystring},
				},
			},
			Node: &yang.Value{
				Name: igenutil.RootElementNodeName,
			},
		},
	}, {
		name: "overlapping root entries",
		inStructs: map[string]*yang.Entry{
			"/module1/foo": {
				Name: "foo",
				Kind: yang.DirectoryEntry,
				Parent: &yang.Entry{
					Name: "module1",
				},
			},
			"/module2/foo": {
				Name: "foo",
				Kind: yang.DirectoryEntry,
				Parent: &yang.Entry{
					Name: "module2",
				},
			},
		},
		inRootName: "name",
		wantErr:    true,
	}}

	for _, tt := range tests {
		err := createFakeRoot(tt.inStructs, tt.inRootElems, tt.inRootName, tt.inCompressPaths)
		if (err != nil) != tt.wantErr {
			t.Errorf("%s: createFakeRoot(%v, %v, %s, %v): did not get expected error, got: %s, wantErr: %v", tt.name, tt.inStructs, tt.inRootElems, tt.inRootName, tt.inCompressPaths, err, tt.wantErr)
			continue
		}

		if err != nil {
			continue
		}

		if diff := pretty.Compare(tt.inStructs["/"], tt.wantRoot); diff != "" {
			t.Errorf("%s: createFakeRoot(%v, %v, %s, %v): did not get expected root struct, diff(-got,+want):\n%s", tt.name, tt.inStructs, tt.inRootElems, tt.inRootName, tt.inCompressPaths, diff)
		}

		if !igenutil.IsFakeRoot(tt.inStructs["/"]) {
			t.Errorf("IsFakeRoot returned false for entry %v", tt.inStructs["/"])
		}
	}
}
