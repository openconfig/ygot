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
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/openconfig/gnmi/errdiff"
	"github.com/openconfig/goyang/pkg/yang"
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
		in   *Directory
		want map[string]string
	}{{
		name: "nil directory",
		in:   nil,
		want: nil,
	}, {
		name: "empty directory",
		in: &Directory{
			Fields: map[string]*yang.Entry{},
		},
		want: map[string]string{},
	}, {
		name: "directory with one field",
		in: &Directory{
			Fields: map[string]*yang.Entry{
				"a": {Name: "a"},
			},
		},
		want: map[string]string{"a": "A"},
	}, {
		name: "directory with multiple fields",
		in: &Directory{
			Fields: map[string]*yang.Entry{
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
			"a": "A",
			"b": "B",
			"c": "C",
			"d": "D",
			"e": "E",
			"f": "F",
			"g": "G",
		},
	}, {
		name: "directory with multiple fields and longer names and a camel case collision",
		in: &Directory{
			Fields: map[string]*yang.Entry{
				"th-e":  {Name: "th-e"},
				"quick": {Name: "quick"},
				"brown": {Name: "brown"},
				"fox":   {Name: "fox"},
				"jumps": {Name: "jumps"},
				"over":  {Name: "over"},
				"thE":   {Name: "thE"},
				"lazy":  {Name: "lazy"},
				"dog":   {Name: "dog"},
			},
		},
		want: map[string]string{
			"brown": "Brown",
			"dog":   "Dog",
			"fox":   "Fox",
			"jumps": "Jumps",
			"lazy":  "Lazy",
			"over":  "Over",
			"quick": "Quick",
			"th-e":  "ThE",
			"thE":   "ThE_",
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

func TestFindSchemaPath(t *testing.T) {
	simpleDir := &Directory{
		Name: "Foo",
		Path: []string{"", "module", "foo"},
		Fields: map[string]*yang.Entry{
			"baz": {
				Name: "baz",
				Parent: &yang.Entry{
					Name: "bar",
					Parent: &yang.Entry{
						Name:   "foo",
						Parent: &yang.Entry{Name: "module"},
					},
				},
			},
		},
	}

	listDir := &Directory{
		Name: "DList",
		Path: []string{"", "d-module", "d-container", "d-list"},
		Fields: map[string]*yang.Entry{
			"d-key": {
				Name: "d-key",
				Type: &yang.YangType{
					Kind: yang.Yleafref,
				},
				Parent: &yang.Entry{
					Name: "config",
					Parent: &yang.Entry{
						Name: "d-list",
						Kind: yang.DirectoryEntry,
						Dir: map[string]*yang.Entry{
							"d-key": {
								Name: "d-key",
								Type: &yang.YangType{Kind: yang.Yleafref},
							},
						},
						Parent: &yang.Entry{
							Name: "d-container",
							Parent: &yang.Entry{
								Name: "d-module",
							},
						},
					},
				},
			},
		},
	}

	tests := []struct {
		name            string
		inDirectory     *Directory
		inFieldName     string
		inAbsolutePaths bool
		want            []string
		wantErrSubstr   string
	}{{
		name:            "simple relative path",
		inDirectory:     simpleDir,
		inFieldName:     "baz",
		inAbsolutePaths: false,
		want:            []string{"bar", "baz"},
	}, {
		name:            "simple absolute path",
		inDirectory:     simpleDir,
		inFieldName:     "baz",
		inAbsolutePaths: true,
		want:            []string{"", "foo", "bar", "baz"},
	}, {
		name:            "field does not exist",
		inDirectory:     simpleDir,
		inFieldName:     "baazar",
		inAbsolutePaths: false,
		wantErrSubstr:   "field name \"baazar\" does not exist in Directory",
	}, {
		name: "directory has problematically-long path",
		inDirectory: &Directory{
			Name: "Foo",
			Path: []string{"", "module", "foo", "too", "long"},
			Fields: map[string]*yang.Entry{
				"baz": {
					Name: "baz",
					Parent: &yang.Entry{
						Name: "bar",
						Parent: &yang.Entry{
							Name:   "foo",
							Parent: &yang.Entry{Name: "module"},
						},
					},
				},
			},
		},
		inFieldName:     "baz",
		inAbsolutePaths: false,
		wantErrSubstr:   "is not a valid child",
	}, {
		name:            "list key relative path",
		inDirectory:     listDir,
		inFieldName:     "d-key",
		inAbsolutePaths: false,
		want:            []string{"config", "d-key"},
	}, {
		name:            "list key absolute path",
		inDirectory:     listDir,
		inFieldName:     "d-key",
		inAbsolutePaths: true,
		want:            []string{"", "d-container", "d-list", "config", "d-key"},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FindSchemaPath(tt.inDirectory, tt.inFieldName, tt.inAbsolutePaths)
			if diff := errdiff.Check(err, tt.wantErrSubstr); diff != "" {
				t.Fatalf("FindSchemaPath, %v", diff)
			}
			if !cmp.Equal(got, tt.want) {
				t.Fatalf("want: %s\ngot: %s", tt.want, got)
			}
		})
	}
}
