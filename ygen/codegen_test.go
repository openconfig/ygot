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

	"github.com/kylelemons/godebug/pretty"
	"github.com/openconfig/goyang/pkg/yang"
)

const (
	// TestRoot is the root of the test directory such that this is not
	// repeated when referencing files.
	TestRoot string = ""
)

func TestNewYANGCodeGeneratorError(t *testing.T) {
	e := NewYANGCodeGeneratorError()
	e.Errors = append(e.Errors, fmt.Errorf("test string"))
	e.Errors = append(e.Errors, []error{fmt.Errorf("test string two"), fmt.Errorf("string three")}...)
	want := "errors encountered during code generation:\ntest string\ntest string two\nstring three\n"

	if got := e.Error(); got != want {
		t.Errorf("NewYANGCodeGenerator did not concatenate errors correctly, got: %s, want: %s", got, want)
	}
}

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

			findMappableEntities(tt.in, structs, enums, tt.inSkipModules, compress)

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
	}, {
		name:    "module with entities at the root",
		inFiles: []string{filepath.Join(TestRoot, "testdata/structs/root-entities.yang")},
		inConfig: GeneratorConfig{
			Caller:           "testcase",
			FakeRootName:     "fakeroot",
			GenerateFakeRoot: true,
		},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata/structs/root-entities.formatted-txt"),
	}, {
		name:                "module with empty leaf",
		inFiles:             []string{filepath.Join(TestRoot, "testdata/structs/empty.yang")},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata/structs/empty.formatted-txt"),
	}, {
		name:    "module with excluded modules",
		inFiles: []string{filepath.Join(TestRoot, "testdata/structs/excluded-module.yang")},
		inConfig: GeneratorConfig{
			GenerateFakeRoot: true,
			FakeRootName:     "office",
			ExcludeModules:   []string{"excluded-module-two"},
		},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata/structs/excluded-module.formatted-txt"),
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
			fmt.Fprint(&gotCode, gotStruct)
		}

		for _, gotEnum := range gotGeneratedCode.Enums {
			fmt.Fprint(&gotCode, gotEnum)
		}

		// Write generated enumeration map out.
		fmt.Fprint(&gotCode, gotGeneratedCode.EnumMap)

		if tt.inConfig.GenerateJSONSchema {
			// Write the schema byte array out.
			fmt.Fprint(&gotCode, gotGeneratedCode.JSONSchemaCode)
			fmt.Fprint(&gotCode, gotGeneratedCode.EnumTypeMap)

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
				diff, _ := generateUnifiedDiff(string(gotGeneratedCode.RawJSONSchema), string(wantSchema))
				t.Errorf("%s: GenerateGoCode(%v, %v), Config: %v, did not return correct JSON (file: %v), diff: \n%s", tt.name, tt.inFiles, tt.inIncludePaths, tt.inConfig, tt.wantSchemaFile, diff)
			}
		}

		if gotCode.String() != string(wantCode) {
			// Use difflib to generate a unified diff between the
			// two code snippets such that this is simpler to debug
			// in the test output.
			diff, _ := generateUnifiedDiff(gotCode.String(), string(wantCode))
			t.Errorf("%s: GenerateGoCode(%v, %v), Config: %v, did not return correct code (file: %v), diff:\n%s",
				tt.name, tt.inFiles, tt.inIncludePaths, tt.inConfig, tt.wantStructsCodeFile, diff)
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
		for compress, wantChildren := range map[bool][]string{true: tt.wantCompressRootChildren, false: tt.wantUncompressRootChildren} {
			if err := createFakeRoot(tt.inStructs, tt.inRootElems, tt.inRootName, compress); err != nil {
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
			CompressOCPaths: true,
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
		name:     "yang schema with a list",
		inFiles:  []string{filepath.Join(TestRoot, "testdata", "proto", "proto-test-b.yang")},
		inConfig: GeneratorConfig{CompressOCPaths: true},
		wantOutputFiles: map[string]string{
			"openconfig":        filepath.Join(TestRoot, "testdata", "proto", "proto-test-b.compress.formatted-txt"),
			"openconfig.device": filepath.Join(TestRoot, "testdata", "proto", "proto-test-b.compress.device.formatted-txt"),
		},
	}, {
		name:    "yang schema with simple enumerations",
		inFiles: []string{filepath.Join(TestRoot, "testdata", "proto", "proto-test-c.yang")},
		wantOutputFiles: map[string]string{
			"openconfig.proto_test_c":              filepath.Join(TestRoot, "testdata", "proto", "proto-test-c.proto-test-c.formatted-txt"),
			"openconfig.proto_test_c.entity":       filepath.Join(TestRoot, "testdata", "proto", "proto-test-c.proto-test-c.entity.formatted-txt"),
			"openconfig.proto_test_c.elists":       filepath.Join(TestRoot, "testdata", "proto", "proto-test-c.proto-test-c.elists.formatted-txt"),
			"openconfig.proto_test_c.elists.elist": filepath.Join(TestRoot, "testdata", "proto", "proto-test-c.proto-test-c.elists.elist.formatted-txt"),
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
		name:    "yang schema with fake root, path compression and union list key",
		inFiles: []string{filepath.Join(TestRoot, "testdata", "proto", "proto-union-list-key.yang")},
		inConfig: GeneratorConfig{
			CompressOCPaths:  true,
			GenerateFakeRoot: true,
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
			GenerateFakeRoot: true,
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
		name:     "yang schema with various types of enums with underscores",
		inFiles:  []string{filepath.Join(TestRoot, "testdata", "proto", "proto-enums.yang")},
		inConfig: GeneratorConfig{},
		wantOutputFiles: map[string]string{
			"openconfig.enums":       filepath.Join(TestRoot, "testdata", "proto", "proto-enums.enums.formatted-txt"),
			"openconfig.proto_enums": filepath.Join(TestRoot, "testdata", "proto", "proto-enums.formatted-txt"),
		},
	}, {
		name: "yang schema with identity that adds to previous module",
		inFiles: []string{
			filepath.Join(TestRoot, "testdata", "proto", "proto-enums.yang"),
			filepath.Join(TestRoot, "testdata", "proto", "proto-enums-addid.yang"),
		},
		inConfig: GeneratorConfig{
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
			ProtoOptions: ProtoOpts{
				AnnotateEnumNames:   true,
				AnnotateSchemaPaths: true,
				NestedMessages:      true,
			},
			GenerateFakeRoot: true,
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
			ProtoOptions: ProtoOpts{
				AnnotateEnumNames:   true,
				AnnotateSchemaPaths: true,
				NestedMessages:      true,
			},
			CompressOCPaths:  true,
			GenerateFakeRoot: true,
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
			ProtoOptions: ProtoOpts{
				AnnotateEnumNames:   true,
				AnnotateSchemaPaths: true,
				NestedMessages:      true,
			},
			GenerateFakeRoot: true,
		},
		wantOutputFiles: map[string]string{
			"openconfig.enums":          filepath.Join(TestRoot, "testdata", "proto", "union-list-key.enums.formatted-txt"),
			"openconfig.union_list_key": filepath.Join(TestRoot, "testdata", "proto", "union-list-key.union_list_key.formatted-txt"),
			"openconfig":                filepath.Join(TestRoot, "testdata", "proto", "union-list-key.formatted-txt"),
		},
	}}

	for _, tt := range tests {
		if tt.inConfig.Caller == "" {
			// Override the caller if it is not set, to ensure that test
			// output is deterministic.
			tt.inConfig.Caller = "codegen-tests"
		}

		cg := NewYANGCodeGenerator(&tt.inConfig)
		gotProto, err := cg.GenerateProto3(tt.inFiles, tt.inIncludePaths)
		if (err != nil) != tt.wantErr {
			t.Errorf("%s: cg.GenerateProto3(%v, %v), config: %v: got unexpected error: %v", tt.name, tt.inFiles, tt.inIncludePaths, tt.inConfig, err)
			continue
		}

		if tt.wantErr || err != nil {
			continue
		}

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

		for pkg, wantFile := range tt.wantOutputFiles {
			wantCode, err := ioutil.ReadFile(wantFile)
			if err != nil {
				t.Errorf("%s: ioutil.ReadFile(%v): could not read file for package %s", tt.name, wantFile, pkg)
				continue
			}

			gotPkg, ok := gotProto.Packages[pkg]
			if !ok {
				t.Errorf("%s: cg.GenerateProto3(%v, %v): did not find expected package %s in output, got: %#v, want key: %v", tt.name, tt.inFiles, tt.inIncludePaths, pkg, protoPkgs(gotProto.Packages), pkg)
				continue
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

			if diff := pretty.Compare(gotCodeBuf.String(), string(wantCode)); diff != "" {
				if diffl, _ := generateUnifiedDiff(gotCodeBuf.String(), string(wantCode)); diffl != "" {
					diff = diffl
				}
				t.Errorf("%s: cg.GenerateProto3(%v, %v) for package %s, did not get expected code (code file: %v), diff(-got,+want):\n%s", tt.name, tt.inFiles, tt.inIncludePaths, pkg, wantFile, diff)
			}
		}

		for pkg, seen := range seenPkg {
			if !seen {
				t.Errorf("%s: cg.GenerateProto3(%v, %v) did not test received package %v", tt.name, tt.inFiles, tt.inIncludePaths, pkg)
			}
		}
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
			Name: defaultRootName,
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
				Name: rootElementNodeName,
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
	}
}
