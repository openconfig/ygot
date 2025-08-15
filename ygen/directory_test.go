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
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/openconfig/gnmi/errdiff"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/genutil"
	"github.com/openconfig/ygot/yangschema"
)

// errToString returns the string representation of err and the empty string if
// err is nil.
func errToString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func TestGetOrderedFieldNames(t *testing.T) {
	tests := []struct {
		name string
		in   *Directory
		want []string
	}{{
		name: "nil directory",
		in:   nil,
		want: nil,
	}, {
		name: "empty directory",
		in: &Directory{
			Fields: map[string]*yang.Entry{},
		},
		want: []string{},
	}, {
		name: "directory with one field",
		in: &Directory{
			Fields: map[string]*yang.Entry{
				"a": {},
			},
		},
		want: []string{"a"},
	}, {
		name: "directory with multiple fields",
		in: &Directory{
			Fields: map[string]*yang.Entry{
				"a": {},
				"b": {},
				"c": {},
				"d": {},
				"e": {},
				"f": {},
				"g": {},
			},
		},
		want: []string{"a", "b", "c", "d", "e", "f", "g"},
	}, {
		name: "directory with multiple fields 2",
		in: &Directory{
			Fields: map[string]*yang.Entry{
				"the":   {},
				"quick": {},
				"brown": {},
				"fox":   {},
				"jumps": {},
				"over":  {},
				"the2":  {},
				"lazy":  {},
				"dog":   {},
			},
		},
		want: []string{"brown", "dog", "fox", "jumps", "lazy", "over", "quick", "the", "the2"},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, want := GetOrderedFieldNames(tt.in), tt.want; !cmp.Equal(want, got) {
				t.Errorf("got: %s\nwant %s", got, want)
			}
		})
	}
}

func TestGoFieldNameMap(t *testing.T) {
	tests := []struct {
		name string
		in   *ParsedDirectory
		want map[string]string
	}{{
		name: "nil directory",
		in:   nil,
		want: nil,
	}, {
		name: "empty directory",
		in: &ParsedDirectory{
			Fields: map[string]*NodeDetails{},
		},
		want: map[string]string{},
	}, {
		name: "directory with one field",
		in: &ParsedDirectory{
			Fields: map[string]*NodeDetails{
				"a": {Name: "A"},
			},
		},
		want: map[string]string{"a": "A"},
	}, {
		name: "directory with multiple fields",
		in: &ParsedDirectory{
			Fields: map[string]*NodeDetails{
				"a": {Name: "a"},
				"b": {Name: "b"},
				"c": {Name: "c"},
				"d": {Name: "d"},
				"e": {Name: "e"},
				"f": {Name: "f"},
				"g": {Name: "g"},
			},
		},
		want: map[string]string{
			"a": "a",
			"b": "b",
			"c": "c",
			"d": "d",
			"e": "e",
			"f": "f",
			"g": "g",
		},
	}, {
		name: "directory with multiple fields and longer names and a camel case collision",
		in: &ParsedDirectory{
			Fields: map[string]*NodeDetails{
				"th-e":  {Name: "ThE"},
				"quick": {Name: "Quick"},
				"thE":   {Name: "ThE"},
				"th-E":  {Name: "ThE"},
			},
		},
		want: map[string]string{
			"quick": "Quick",
			"th-e":  "ThE_",
			"thE":   "ThE__",
			"th-E":  "ThE",
		},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, want := GoFieldNameMap(tt.in), tt.want; !cmp.Equal(want, got) {
				t.Errorf("got: %v\nwant %s", got, want)
			}
		})
	}
}

func TestGetOrderedDirectories(t *testing.T) {
	a := &Directory{Name: "a"}
	b := &Directory{Name: "b"}
	c := &Directory{Name: "c"}
	d := &Directory{Name: "d"}
	e := &Directory{Name: "e"}
	f := &Directory{Name: "f"}

	tests := []struct {
		name             string
		in               map[string]*Directory
		wantOrderedNames []string
		wantDirectoryMap map[string]*Directory
		wantErr          string
	}{{
		name:    "nil directory map",
		in:      nil,
		wantErr: "directory map null",
	}, {
		name:             "empty directory map",
		in:               map[string]*Directory{},
		wantOrderedNames: []string{},
		wantDirectoryMap: map[string]*Directory{},
	}, {
		name: "directory map with one directory",
		in: map[string]*Directory{
			"a/b/c": c,
		},
		wantOrderedNames: []string{"c"},
		wantDirectoryMap: map[string]*Directory{"c": c},
	}, {
		name: "directory map with multiple directories",
		in: map[string]*Directory{
			"a/b/d": d,
			"a/b/f": f,
			"a/b/c": c,
			"a/b/b": b,
			"a/b/a": a,
			"a/b/e": e,
		},
		wantOrderedNames: []string{"a", "b", "c", "d", "e", "f"},
		wantDirectoryMap: map[string]*Directory{
			"a": a,
			"b": b,
			"c": c,
			"d": d,
			"e": e,
			"f": f,
		},
	}, {
		name: "directory map with a conflict",
		in: map[string]*Directory{
			"a/b/d": d,
			"a/b/f": f,
			"a/b/c": c,
			"a/b/b": b,
			"a/b/a": a,
			"a/b/e": d,
		},
		wantErr: "directory name conflict(s) exist",
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOrderedNames, gotDirMap, err := GetOrderedDirectories(tt.in)
			if gotErr := errToString(err); gotErr != tt.wantErr {
				t.Fatalf("wantErr: %s\ngotErr: %s", tt.wantErr, gotErr)
			}
			if !cmp.Equal(gotOrderedNames, tt.wantOrderedNames) {
				t.Errorf("wantOrderedNames: %s\ngotOrderedNames: %s", tt.wantOrderedNames, gotOrderedNames)
			}
			if !cmp.Equal(gotDirMap, tt.wantDirectoryMap) {
				t.Errorf("wantDirMap: %v\ngotwantDirMap: %v", tt.wantDirectoryMap, gotDirMap)
			}
		})
	}
}

func compileModules(t *testing.T, inModules map[string]string) *yang.Modules {
	t.Helper()
	ms := yang.NewModules()
	for n, m := range inModules {
		if err := ms.Parse(m, n); err != nil {
			t.Fatalf("error parsing module %q: %v", n, err)
		}
	}
	if errs := ms.Process(); errs != nil {
		t.Fatalf("modules processing failed: %v", errs)
	}
	return ms

}

// findEntry gets the entry for the module given the path.
func findEntry(t *testing.T, ms *yang.Modules, moduleName, path string) *yang.Entry {
	t.Helper()
	module, errs := ms.GetModule(moduleName)
	if errs != nil {
		t.Fatalf("error getting module %q: %v", moduleName, errs)
	}
	if path == "" {
		return module
	}
	entry := module.Find(path)
	if entry == nil {
		t.Fatalf("error getting entry %q in module %q", path, moduleName)
	}
	return entry
}

func TestFindSchemaPath(t *testing.T) {
	ms := compileModules(t, map[string]string{
		"module": `
			module module {
				prefix "m";
				namespace "urn:m";

				container foo {
					container bar {
						leaf baz {
							type string;
						}
					}
				}
			}
		`,
		"d-module": `
			module d-module {
				prefix "n";
				namespace "urn:n";

				container d-container {
					list d-list {
						key d-key;

						leaf d-key {
							type leafref {
								path "../config/d-key";
							}
						}

						container config {
							leaf d-key {
								type string;
							}
						}
					}
				}
			}
		`,
	})

	baz := findEntry(t, ms, "module", "foo/bar/baz")
	simpleDir := &Directory{
		Name: "Foo",
		Path: []string{"", "module", "foo"},
		Fields: map[string]*yang.Entry{
			"baz": baz,
		},
	}

	listDir := &Directory{
		Name: "DList",
		Path: []string{"", "d-module", "d-container", "d-list"},
		Fields: map[string]*yang.Entry{
			"d-key": findEntry(t, ms, "d-module", "d-container/d-list/config/d-key"),
		},
	}

	tests := []struct {
		name                  string
		inDirectory           *Directory
		inFieldName           string
		inAbsolutePaths       bool
		wantPath              []string
		wantModules           []string
		wantErrSubstr         string
		wantErrSubstrShadowed string
	}{{
		name:            "simple relative path",
		inDirectory:     simpleDir,
		inFieldName:     "baz",
		inAbsolutePaths: false,
		wantPath:        []string{"bar", "baz"},
	}, {
		name:            "simple absolute path",
		inDirectory:     simpleDir,
		inFieldName:     "baz",
		inAbsolutePaths: true,
		wantPath:        []string{"", "foo", "bar", "baz"},
	}, {
		name:            "field does not exist",
		inDirectory:     simpleDir,
		inFieldName:     "baazar",
		inAbsolutePaths: false,
		wantPath:        nil,
		wantErrSubstr:   "field name \"baazar\" does not exist in Directory",
		// wantErrSubstrShadowed is missing here: when shadowSchemaPaths is set, no error is returned when the field can't be found.
	}, {
		name: "directory has problematically-long path",
		inDirectory: &Directory{
			Name: "Foo",
			Path: []string{"", "module", "foo", "too", "long"},
			Fields: map[string]*yang.Entry{
				"baz": baz,
			},
		},
		inFieldName:           "baz",
		inAbsolutePaths:       false,
		wantErrSubstr:         "is not a valid child",
		wantErrSubstrShadowed: "is not a valid child",
	}, {
		name:            "list key relative path",
		inDirectory:     listDir,
		inFieldName:     "d-key",
		inAbsolutePaths: false,
		wantPath:        []string{"config", "d-key"},
	}, {
		name:            "list key absolute path",
		inDirectory:     listDir,
		inFieldName:     "d-key",
		inAbsolutePaths: true,
		wantPath:        []string{"", "d-container", "d-list", "config", "d-key"},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPath, err := FindSchemaPath(tt.inDirectory, tt.inFieldName, tt.inAbsolutePaths)
			if diff := errdiff.Check(err, tt.wantErrSubstr); diff != "" {
				t.Fatalf("FindSchemaPath, %v", diff)
			}
			if diff := cmp.Diff(gotPath, tt.wantPath); diff != "" {
				t.Fatalf("(-gotPath, want):\n%s", diff)
			}
		})
	}

	for _, tt := range tests {
		// Move over the shadowed fields to be the same as the direct fields (if not already done).
		if tt.inDirectory.ShadowedFields == nil {
			tt.inDirectory.ShadowedFields = tt.inDirectory.Fields
			tt.inDirectory.Fields = nil
		}

		t.Run(tt.name+" (ShadowedFields)", func(t *testing.T) {
			gotPath, _, err := findSchemaPath(tt.inDirectory, tt.inFieldName, true, tt.inAbsolutePaths)
			if diff := errdiff.Check(err, tt.wantErrSubstrShadowed); diff != "" {
				t.Fatalf("FindShadowedSchemaPath, %v", diff)
			}
			if diff := cmp.Diff(gotPath, tt.wantPath); diff != "" {
				t.Fatalf("(-gotPath, want):\n%s", diff)
			}
		})
	}
}

// TestFindMapPaths ensures that the schema paths that an entity should be
// mapped to are properly extracted from a schema element.
func TestFindMapPaths(t *testing.T) {
	ms := compileModules(t, map[string]string{
		"a-module": `
			module a-module {
				prefix "m";
				namespace "urn:m";

				container a-container {
					leaf field-a {
						type string;
					}
				}

				container b-container {
					container config {
						leaf field-b {
							type string;
						}
					}
					container state {
						leaf field-b {
							type string;
						}
					}

					container c-container {
						leaf field-d {
							type string;
						}
					}
				}
			}
		`,
		"d-module": `
			module d-module {
				prefix "n";
				namespace "urn:n";

				import a-module { prefix "a"; }

				augment "/a:b-container/config" {
					leaf field-c { type string; }
				}

				augment "/a:b-container/state" {
					leaf field-c { type string; }
				}

				container d-container {
					list d-list {
						key d-key;

						leaf d-key {
							type leafref {
								path "../config/d-key";
							}
						}

						container config {
							leaf d-key {
								type string;
							}
						}

						container state {
							leaf d-key {
								type string;
							}
						}
					}
				}
			}
		`,
	})

	tests := []struct {
		name                string
		inStruct            *Directory
		inField             string
		inCompressPaths     bool
		inShadowSchemaPaths bool
		inAbsolutePaths     bool
		wantPaths           [][]string
		wantModules         [][]string
		wantErr             bool
	}{{
		name: "first-level container with path compression off",
		inStruct: &Directory{
			Name: "AContainer",
			Path: []string{"", "a-module", "a-container"},
			Fields: map[string]*yang.Entry{
				"field-a": findEntry(t, ms, "a-module", "a-container/field-a"),
			},
		},
		inField:     "field-a",
		wantPaths:   [][]string{{"field-a"}},
		wantModules: [][]string{{"a-module"}},
	}, {
		name: "invalid parent path - shorter than directory path",
		inStruct: &Directory{
			Name: "AContainer",
			Path: []string{"", "a-module", "a-container"},
			Fields: map[string]*yang.Entry{
				"field-a": findEntry(t, ms, "a-module", "a-container"),
			},
		},
		inField: "field-a",
		wantErr: true,
	}, {
		name: "first-level container with path compression on",
		inStruct: &Directory{
			Name: "BContainer",
			Path: []string{"", "a-module", "b-container"},
			Fields: map[string]*yang.Entry{
				"field-b": findEntry(t, ms, "a-module", "b-container/config/field-b"),
			},
			ShadowedFields: map[string]*yang.Entry{
				"field-b": findEntry(t, ms, "a-module", "b-container/state/field-b"),
			},
		},
		inField:         "field-b",
		inCompressPaths: true,
		wantPaths:       [][]string{{"config", "field-b"}},
		wantModules:     [][]string{{"a-module", "a-module"}},
	}, {
		name: "first-level container with path compression on and ignoreShadowSchemaPaths on",
		inStruct: &Directory{
			Name: "BContainer",
			Path: []string{"", "a-module", "b-container"},
			Fields: map[string]*yang.Entry{
				"field-b": findEntry(t, ms, "a-module", "b-container/config/field-b"),
			},
			ShadowedFields: map[string]*yang.Entry{
				"field-b": findEntry(t, ms, "a-module", "b-container/state/field-b"),
			},
		},
		inField:             "field-b",
		inCompressPaths:     true,
		inShadowSchemaPaths: true,
		wantPaths:           [][]string{{"state", "field-b"}},
		wantModules:         [][]string{{"a-module", "a-module"}},
	}, {
		name: "augmented first-level container with path compression on",
		inStruct: &Directory{
			Name: "BContainer",
			Path: []string{"", "a-module", "b-container"},
			Fields: map[string]*yang.Entry{
				"field-c": findEntry(t, ms, "a-module", "b-container/config/field-c"),
			},
			ShadowedFields: map[string]*yang.Entry{
				"field-c": findEntry(t, ms, "a-module", "b-container/state/field-c"),
			},
		},
		inField:         "field-c",
		inCompressPaths: true,
		wantPaths:       [][]string{{"config", "field-c"}},
		wantModules:     [][]string{{"a-module", "d-module"}},
	}, {
		name: "augmented first-level container with inShadowSchemaPaths=true",
		inStruct: &Directory{
			Name: "BContainer",
			Path: []string{"", "a-module", "b-container"},
			Fields: map[string]*yang.Entry{
				"field-c": findEntry(t, ms, "a-module", "b-container/config/field-c"),
			},
			ShadowedFields: map[string]*yang.Entry{
				"field-c": findEntry(t, ms, "a-module", "b-container/state/field-c"),
			},
		},
		inField:             "field-c",
		inCompressPaths:     true,
		inShadowSchemaPaths: true,
		wantPaths:           [][]string{{"state", "field-c"}},
		wantModules:         [][]string{{"a-module", "d-module"}},
	}, {
		name: "container with absolute paths on",
		inStruct: &Directory{
			Name: "BContainer",
			Path: []string{"", "a-module", "b-container", "c-container"},
			Fields: map[string]*yang.Entry{
				"field-d": findEntry(t, ms, "a-module", "b-container/c-container/field-d"),
			},
		},
		inField:         "field-d",
		inAbsolutePaths: true,
		wantPaths:       [][]string{{"", "b-container", "c-container", "field-d"}},
		wantModules:     [][]string{{"", "a-module", "a-module", "a-module"}},
	}, {
		name: "top-level module",
		inStruct: &Directory{
			Name: "CContainer",
			Path: []string{""},
			Fields: map[string]*yang.Entry{
				"top": findEntry(t, ms, "a-module", ""),
			},
		},
		inField:     "top",
		wantPaths:   [][]string{{"a-module"}},
		wantModules: [][]string{{"a-module"}},
	}, {
		name: "list with leafref key",
		inStruct: &Directory{
			Name: "DList",
			Path: []string{"", "d-module", "d-container", "d-list"},
			ListAttr: &YangListAttr{
				KeyElems: []*yang.Entry{
					findEntry(t, ms, "d-module", "d-container/d-list/config/d-key"),
				},
			},
			Fields: map[string]*yang.Entry{
				"d-key": findEntry(t, ms, "d-module", "d-container/d-list/config/d-key"),
			},
			ShadowedFields: map[string]*yang.Entry{
				"d-key": findEntry(t, ms, "d-module", "d-container/d-list/state/d-key"),
			},
		},
		inField:         "d-key",
		inCompressPaths: true,
		wantPaths: [][]string{
			{"config", "d-key"},
			{"d-key"},
		},
		wantModules: [][]string{
			{"d-module", "d-module"},
			{"d-module"},
		},
	}, {
		name: "list with leafref key with shadowSchemaPaths=true",
		inStruct: &Directory{
			Name: "DList",
			Path: []string{"", "d-module", "d-container", "d-list"},
			ListAttr: &YangListAttr{
				KeyElems: []*yang.Entry{
					findEntry(t, ms, "d-module", "d-container/d-list/config/d-key"),
				},
			},
			Fields: map[string]*yang.Entry{
				"d-key": findEntry(t, ms, "d-module", "d-container/d-list/config/d-key"),
			},
			ShadowedFields: map[string]*yang.Entry{
				"d-key": findEntry(t, ms, "d-module", "d-container/d-list/state/d-key"),
			},
		},
		inField:             "d-key",
		inCompressPaths:     true,
		inShadowSchemaPaths: true,
		wantPaths: [][]string{
			{"state", "d-key"},
			{"d-key"},
		},
		wantModules: [][]string{
			{"d-module", "d-module"},
			{"d-module"},
		},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPaths, gotModules, err := findMapPaths(tt.inStruct, tt.inField, tt.inCompressPaths, tt.inShadowSchemaPaths, tt.inAbsolutePaths)
			if (err != nil) != tt.wantErr {
				t.Errorf("%s: YANGCodeGenerator.findMapPaths(%v, %v): compress: %v, shadowSchemaPaths: %v, wantErr: %v, gotPaths error: %v",
					tt.name, tt.inStruct, tt.inField, tt.inCompressPaths, tt.inShadowSchemaPaths, tt.wantErr, err)
			}
			if tt.wantErr {
				return
			}

			if diff := cmp.Diff(tt.wantPaths, gotPaths); diff != "" {
				t.Errorf("%s: YANGCodeGenerator.findMapPaths(%v, %v): compress: %v, shadowSchemaPaths: %v, (-want, +gotPaths):\n%s", tt.name, tt.inStruct, tt.inField, tt.inCompressPaths, tt.inShadowSchemaPaths, diff)
			}

			if diff := cmp.Diff(tt.wantModules, gotModules); diff != "" {
				t.Errorf("%s: YANGCodeGenerator.findMapPaths(%v, %v): compress: %v, shadowSchemaPaths: %v, (-want, +gotModules):\n%s", tt.name, tt.inStruct, tt.inField, tt.inCompressPaths, tt.inShadowSchemaPaths, diff)
			}
		})
	}
}

type mockLangMapper struct{}

func (*mockLangMapper) FieldName(_ *yang.Entry) (string, error) { return "", nil }

func (*mockLangMapper) DirectoryName(_ *yang.Entry, _ genutil.CompressBehaviour) (string, error) {
	return "", nil
}

func (*mockLangMapper) KeyLeafType(_ *yang.Entry, _ IROptions) (*MappedType, error) {
	return nil, nil
}

func (*mockLangMapper) LeafType(_ *yang.Entry, _ IROptions) (*MappedType, error) { return nil, nil }

func (*mockLangMapper) PackageName(_ *yang.Entry, _ genutil.CompressBehaviour, _ bool) (string, error) {
	return "", nil
}

func (*mockLangMapper) InjectEnumSet(_ map[string]*yang.Entry, _, _, _, _, _, _ bool, _ []string) error {
	return nil
}

func (*mockLangMapper) InjectSchemaTree(_ []*yang.Entry) error { return nil }

func (*mockLangMapper) PopulateEnumFlags(EnumeratedYANGType, *yang.YangType) map[string]string {
	return nil
}

func (*mockLangMapper) PopulateFieldFlags(NodeDetails, *yang.Entry) map[string]string { return nil }

func (*mockLangMapper) setEnumSet(*enumSet) {}

func (*mockLangMapper) setSchemaTree(*yangschema.Tree) {}

func TestGetOrderedDirDetailsPathOrigin(t *testing.T) {
	ms := compileModules(t, map[string]string{
		"a-module": `
			module a-module {
				prefix "m";
				namespace "urn:m";

				container a-container {
					leaf field-a {
						type string;
					}
				}

				container b-container {
					container config {
						leaf field-b {
							type string;
						}
					}
					container state {
						leaf field-b {
							type string;
						}
					}

					container c-container {
						leaf field-d {
							type string;
						}
					}
				}
			}
		`,
	})

	tests := []struct {
		name                string
		inDirectory         map[string]*Directory
		inSchemaTree        *yangschema.Tree
		inOpts              IROptions
		wantPathOriginName  map[string]string
		wantErr             bool
		wantErrorSubstrings []string
	}{{
		name: "PathOriginName is set",
		inDirectory: map[string]*Directory{
			"/a-module/a-container": {
				Name:  "AContainer",
				Entry: findEntry(t, ms, "a-module", "a-container/field-a"),
				Fields: map[string]*yang.Entry{
					"field-a": findEntry(t, ms, "a-module", "a-container/field-a"),
				},
				Path: []string{"", "a-module", "a-container"},
			},
		},
		inSchemaTree: &yangschema.Tree{},
		inOpts: IROptions{
			PathOriginName: "explicit-origin",
		},
		wantPathOriginName: map[string]string{
			"/a-module/a-container/field-a": "explicit-origin",
		},
	}, {
		name: "UseModuleNameAsPathOrigin is true",
		inDirectory: map[string]*Directory{
			"/a-module/a-container": {
				Name:  "AContainer",
				Entry: findEntry(t, ms, "a-module", "a-container/field-a"),
				Fields: map[string]*yang.Entry{
					"field-a": findEntry(t, ms, "a-module", "a-container/field-a"),
				},
				Path: []string{"", "a-module", "a-container"},
			},
		},
		inSchemaTree: &yangschema.Tree{},
		inOpts: IROptions{
			UseModuleNameAsPathOrigin: true,
		},
		wantPathOriginName: map[string]string{
			"/a-module/a-container/field-a": "a-module",
		},
	}, {
		name: "PathOriginName and UseModuleNameAsPathOrigin are not set",
		inDirectory: map[string]*Directory{
			"/a-module/a-container": {
				Name:  "AContainer",
				Entry: findEntry(t, ms, "a-module", "a-container/field-a"),
				Fields: map[string]*yang.Entry{
					"field-a": findEntry(t, ms, "a-module", "a-container/field-a"),
				},
				Path: []string{"", "a-module", "a-container"},
			},
		},
		inSchemaTree: &yangschema.Tree{},
		inOpts:       IROptions{},
		wantPathOriginName: map[string]string{
			"/a-module/a-container/field-a": "",
		},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			langMapper := &mockLangMapper{}
			got, err := getOrderedDirDetails(langMapper, tt.inDirectory, tt.inSchemaTree, tt.inOpts)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("getOrderedDirDetails() got no error, want error containing: %v", tt.wantErrorSubstrings)
				}
				for _, want := range tt.wantErrorSubstrings {
					if !strings.Contains(err.Error(), want) {
						t.Errorf("getOrderedDirDetails() got error: %v, want error containing: %q", err, want)
					}
				}
				return
			}

			if err != nil {
				t.Fatalf("getOrderedDirDetails() unexpected error: %v", err)
			}

			if diff := cmp.Diff(tt.wantPathOriginName, getPathOriginNames(got)); diff != "" {
				t.Errorf("getOrderedDirDetails() PathOriginName mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func getPathOriginNames(dirs map[string]*ParsedDirectory) map[string]string {
	origins := make(map[string]string)
	for path, dir := range dirs {
		for _, field := range dir.Fields {
			origins[path] = field.YANGDetails.Origin
			break // The PathOriginName of the first field in each ParsedDirectory is verified.
		}
	}
	return origins
}

func TestGetOrderedDirDetailsStatusFiltering(t *testing.T) {
	ms := compileModules(t, map[string]string{
		"status-module": `
			module status-module {
				prefix "s";
				namespace "urn:s";

				container test-container {
					leaf normal-field {
						type string;
					}
					leaf deprecated-field {
						status deprecated;
						type string;
					}
					leaf obsolete-field {
						status obsolete;
						type string;
					}
				}
			}
		`,
	})

	containerEntry := findEntry(t, ms, "status-module", "test-container")
	normalField := findEntry(t, ms, "status-module", "test-container/normal-field")
	deprecatedField := findEntry(t, ms, "status-module", "test-container/deprecated-field")
	obsoleteField := findEntry(t, ms, "status-module", "test-container/obsolete-field")

	tests := []struct {
		name              string
		inDirectory       map[string]*Directory
		inOpts            IROptions
		wantFieldNames    []string
		wantMissingFields []string
	}{{
		name: "no status filtering",
		inDirectory: map[string]*Directory{
			"/status-module/test-container": {
				Name:  "TestContainer",
				Entry: containerEntry,
				Fields: map[string]*yang.Entry{
					"normal-field":     normalField,
					"deprecated-field": deprecatedField,
					"obsolete-field":   obsoleteField,
				},
				Path: []string{"", "status-module", "test-container"},
			},
		},
		inOpts: IROptions{
			TransformationOptions: TransformationOpts{},
		},
		wantFieldNames:    []string{"normal-field", "deprecated-field", "obsolete-field"},
		wantMissingFields: []string{},
	}, {
		name: "skip deprecated fields",
		inDirectory: map[string]*Directory{
			"/status-module/test-container": {
				Name:  "TestContainer",
				Entry: containerEntry,
				Fields: map[string]*yang.Entry{
					"normal-field":     normalField,
					"deprecated-field": deprecatedField,
					"obsolete-field":   obsoleteField,
				},
				Path: []string{"", "status-module", "test-container"},
			},
		},
		inOpts: IROptions{
			TransformationOptions: TransformationOpts{SkipDeprecated: true},
		},
		wantFieldNames:    []string{"normal-field", "obsolete-field"},
		wantMissingFields: []string{"deprecated-field"},
	}, {
		name: "skip obsolete fields",
		inDirectory: map[string]*Directory{
			"/status-module/test-container": {
				Name:  "TestContainer",
				Entry: containerEntry,
				Fields: map[string]*yang.Entry{
					"normal-field":     normalField,
					"deprecated-field": deprecatedField,
					"obsolete-field":   obsoleteField,
				},
				Path: []string{"", "status-module", "test-container"},
			},
		},
		inOpts: IROptions{
			TransformationOptions: TransformationOpts{SkipObsolete: true},
		},
		wantFieldNames:    []string{"normal-field", "deprecated-field"},
		wantMissingFields: []string{"obsolete-field"},
	}, {
		name: "skip both deprecated and obsolete fields",
		inDirectory: map[string]*Directory{
			"/status-module/test-container": {
				Name:  "TestContainer",
				Entry: containerEntry,
				Fields: map[string]*yang.Entry{
					"normal-field":     normalField,
					"deprecated-field": deprecatedField,
					"obsolete-field":   obsoleteField,
				},
				Path: []string{"", "status-module", "test-container"},
			},
		},
		inOpts: IROptions{
			TransformationOptions: TransformationOpts{SkipDeprecated: true, SkipObsolete: true},
		},
		wantFieldNames:    []string{"normal-field"},
		wantMissingFields: []string{"deprecated-field", "obsolete-field"},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			langMapper := &mockLangMapper{}
			got, err := getOrderedDirDetails(langMapper, tt.inDirectory, &yangschema.Tree{}, tt.inOpts)
			if err != nil {
				t.Fatalf("getOrderedDirDetails() unexpected error: %v", err)
			}

			// Check that the expected fields are present
			for _, dirPath := range []string{"/status-module/test-container"} {
				dir, ok := got[dirPath]
				if !ok {
					t.Fatalf("getOrderedDirDetails() missing directory: %s", dirPath)
				}

				// Verify expected fields are present
				for _, fieldName := range tt.wantFieldNames {
					if _, ok := dir.Fields[fieldName]; !ok {
						t.Errorf("getOrderedDirDetails() missing expected field: %s", fieldName)
					}
				}

				// Verify unwanted fields are not present
				for _, fieldName := range tt.wantMissingFields {
					if _, ok := dir.Fields[fieldName]; ok {
						t.Errorf("getOrderedDirDetails() contains field that should be filtered: %s", fieldName)
					}
				}

				// Verify total field count matches expectations
				expectedCount := len(tt.wantFieldNames)
				if len(dir.Fields) != expectedCount {
					var actualFields []string
					for name := range dir.Fields {
						actualFields = append(actualFields, name)
					}
					t.Errorf("getOrderedDirDetails() field count mismatch: got %d fields %v, want %d fields %v",
						len(dir.Fields), actualFields, expectedCount, tt.wantFieldNames)
				}
			}
		})
	}
}
