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
	"testing"

	"github.com/kylelemons/godebug/pretty"
	"github.com/openconfig/gnmi/errdiff"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/internal/igenutil"
)

// TestFindMappableEntities tests the extraction of elements that are to be mapped
// into Go code from a YANG schema.
func TestFindMappableEntities(t *testing.T) {
	tests := []struct {
		name                          string        // name is an identifier for the test.
		in                            *yang.Entry   // in is the yang.Entry corresponding to the YANG root element.
		inSkipModules                 []string      // inSkipModules is a slice of strings indicating modules to be skipped.
		inModules                     []*yang.Entry // inModules is the set of modules that the code generation is for.
		inIgnoreUnsupportedStatements bool          // inIgnoreUnsupportedStatements determines whether unsupported statements should error out.
		// wantCompressed is a map keyed by the string "structs" or "enums" which contains a slice
		// of the YANG identifiers for the corresponding mappable entities that should be
		// found. wantCompressed is the set that are expected when compression is enabled.
		wantCompressed map[string][]string
		// wantUncompressed is a map of the same form as wantCompressed. It is the expected
		// result when compression is disabled.
		wantUncompressed map[string][]string
		wantErrSubstring string
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
		name: "unsupported-test",
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
							Dir: map[string]*yang.Entry{
								"leaf": {
									Name: "leaf",
									Kind: yang.NotificationEntry,
								},
							},
						},
					},
				},
			},
		},
		wantErrSubstring: "unsupported statement type (Notification)",
	}, {
		name: "ignore-unsupported-test",
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
							Dir: map[string]*yang.Entry{
								"leaf": {
									Name: "leaf",
									Kind: yang.NotificationEntry,
								},
							},
						},
					},
				},
			},
		},
		inIgnoreUnsupportedStatements: true,
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

			errs := findMappableEntities(tt.in, structs, enums, tt.inSkipModules, compress, tt.inIgnoreUnsupportedStatements, tt.inModules)

			var err error
			switch {
			case len(errs) == 1:
				err = errs[0]
			case len(errs) > 1:
				t.Errorf("%s: findMappableEntities(compressEnabled: %v): got too many errors, got: %v, want: %q", tt.name, compress, errs, tt.wantErrSubstring)
				continue
			}
			if diff := errdiff.Substring(err, tt.wantErrSubstring); diff != "" {
				t.Errorf("%s: findMappableEntities(compressEnabled: %v): did not get expected error:\n%s", tt.name, compress, diff)
			}
			if len(errs) > 0 {
				continue
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
