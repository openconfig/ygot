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

package genutil

import (
	"bytes"
	"testing"

	"github.com/openconfig/goyang/pkg/yang"
)

func TestWriteIfNotEmpty(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{{
		name: "empty",
		in:   "",
		want: "",
	}, {
		name: "non-empty",
		in:   "hello world!",
		want: "hello world!",
	}}

	for _, tt := range tests {
		b := bytes.Buffer{}
		WriteIfNotEmpty(&b, tt.in)
		if got, want := b.String(), tt.want; got != want {
			t.Errorf("%s (WriteIfNotEmpty: %v): %v is not %s", tt.name, tt.in, got, want)
		}
	}
}

func TestTypeDefaultValue(t *testing.T) {
	strPtr := func(s string) *string { return &s }

	tests := []struct {
		name     string
		yangType *yang.YangType
		want     *string
	}{{
		name:     "nil",
		yangType: nil,
		want:     nil,
	}, {
		name:     "no default",
		yangType: &yang.YangType{},
		want:     nil,
	}, {
		name: "default",
		yangType: &yang.YangType{
			Default: "hello world!",
		},
		want: strPtr("hello world!"),
	}}

	for _, tt := range tests {
		got, want := TypeDefaultValue(tt.yangType), tt.want

		if got == nil && want == nil {
			continue
		} else if got == nil || want == nil || *got != *want {
			t.Errorf("%s (TypeDefaultValue: %v): %s is not %s", tt.name, tt.yangType, *got, *want)
		}
	}
}

// TestFindChildren tests the findAllChildren function to ensure that the
// child nodes that are extracted from a YANG schema instance correctly. The
// test is run with the schema compression flag on and off - such that both
// a simplified and unsimplified schema can be tested.
func TestFindChildren(t *testing.T) {
	tests := []struct {
		name             string
		inElement        *yang.Entry
		inExcludeState   bool
		wantCompressed   []yang.Entry
		wantUncompressed []yang.Entry
		// wantErr is a map keyed by the CompressOCPaths value of whether errors
		// are expected. i.e., wantErr[true] = false means that an error is not
		// expected when the test is run with CompressOCPaths == true.
		wantErr map[bool]bool
	}{{
		name: "interface",
		inElement: &yang.Entry{
			Name:     "interface",
			ListAttr: &yang.ListAttr{},
			Dir: map[string]*yang.Entry{
				"config": {
					Name:   "config",
					Type:   &yang.YangType{},
					Config: yang.TSTrue,
					Dir: map[string]*yang.Entry{
						"type": {
							Name:   "type",
							Config: yang.TSTrue,
							Type: &yang.YangType{
								Kind: yang.Ystring,
							},
						},
						"name": {
							Name:   "name",
							Config: yang.TSTrue,
							Type: &yang.YangType{
								Kind: yang.Ystring,
							},
						},
					},
				},
				"state": {
					Name:   "state",
					Type:   &yang.YangType{},
					Config: yang.TSFalse,
					Dir: map[string]*yang.Entry{
						"type": {
							Name:   "type",
							Config: yang.TSFalse,
							Type: &yang.YangType{
								Kind: yang.Ystring,
							},
						},
						"name": {
							Name:   "name",
							Config: yang.TSFalse,
							Type:   &yang.YangType{Kind: yang.Ystring},
						},
						"admin-status": {
							Name:   "admin-status",
							Config: yang.TSFalse,
							Type:   &yang.YangType{Kind: yang.Ystring},
						},
					},
				},
				"name": {
					Name:   "name",
					Config: yang.TSTrue,
					Type:   &yang.YangType{Kind: yang.Yleafref},
				},
			},
		},
		wantCompressed: []yang.Entry{
			{
				Name:   "name",
				Config: yang.TSTrue,
				Type: &yang.YangType{
					Kind: yang.Ystring,
				},
			},
			{
				Name:   "type",
				Config: yang.TSTrue,
				Type:   &yang.YangType{Kind: yang.Ystring},
			},
			{
				Name:   "admin-status",
				Config: yang.TSFalse,
				Type: &yang.YangType{
					Kind: yang.Ystring,
				},
			},
		},
		wantUncompressed: []yang.Entry{
			{
				Name:   "config",
				Config: yang.TSTrue,
				Type:   &yang.YangType{},
			},
			{
				Name:   "state",
				Config: yang.TSFalse,
				Type:   &yang.YangType{},
			},
			{
				Name:   "name",
				Config: yang.TSTrue,
				Type:   &yang.YangType{Kind: yang.Yleafref},
			},
		},
	}, {
		name: "surrounding-container",
		inElement: &yang.Entry{
			Name:   "root",
			Config: yang.TSTrue,
			Type:   &yang.YangType{},
			Dir: map[string]*yang.Entry{
				"plural": {
					Name:   "plural",
					Config: yang.TSTrue,
					Type:   &yang.YangType{},
					Dir: map[string]*yang.Entry{
						"singular": {
							Name:     "singular",
							Config:   yang.TSTrue,
							Dir:      map[string]*yang.Entry{},
							Type:     &yang.YangType{},
							ListAttr: &yang.ListAttr{},
						},
					},
				},
			},
		},
		wantCompressed: []yang.Entry{
			{
				Name:   "singular",
				Config: yang.TSTrue,
				Type:   &yang.YangType{},
			},
		},
		wantUncompressed: []yang.Entry{
			{
				Name:   "plural",
				Config: yang.TSTrue,
				Type:   &yang.YangType{},
			},
		},
	}, {
		name: "duplicate-elements",
		inElement: &yang.Entry{
			Name:   "root",
			Config: yang.TSTrue,
			Type:   &yang.YangType{},
			Kind:   yang.DirectoryEntry,
			Dir: map[string]*yang.Entry{
				"name": {Name: "name"},
				"config": {
					Name:   "config",
					Config: yang.TSTrue,
					Type:   &yang.YangType{},
					Kind:   yang.DirectoryEntry,
					Dir: map[string]*yang.Entry{
						"name": {Name: "name"},
					},
				},
			},
		},
		wantErr: map[bool]bool{true: true},
	}, {
		name: "choice entry",
		inElement: &yang.Entry{
			Name: "choice-node",
			Kind: yang.ChoiceEntry,
			Dir: map[string]*yang.Entry{
				"case-one": {
					Name: "case-one",
					Kind: yang.CaseEntry,
					Dir: map[string]*yang.Entry{
						"option": {
							Name: "option",
							Type: &yang.YangType{},
						},
					},
				},
			},
		},
		wantCompressed: []yang.Entry{
			{
				Name: "option",
				Type: &yang.YangType{},
			},
		},
		wantUncompressed: []yang.Entry{
			{
				Name: "option",
				Type: &yang.YangType{},
			},
		},
	}, {
		name: "choice entry within state",
		inElement: &yang.Entry{
			Name: "container",
			Kind: yang.DirectoryEntry,
			Dir: map[string]*yang.Entry{
				"state": {
					Name: "state",
					Kind: yang.DirectoryEntry,
					Dir: map[string]*yang.Entry{
						"choice": {
							Kind: yang.ChoiceEntry,
							Dir: map[string]*yang.Entry{
								"case": {
									Kind: yang.CaseEntry,
									Dir: map[string]*yang.Entry{
										"string": {
											Name: "string",
										},
									},
								},
							},
						},
					},
				},
			},
		},
		wantCompressed: []yang.Entry{{
			Name: "string",
		}},
		wantUncompressed: []yang.Entry{{
			Name: "state",
		}},
	}, {
		name: "choice entry within config",
		inElement: &yang.Entry{
			Name: "container",
			Kind: yang.DirectoryEntry,
			Dir: map[string]*yang.Entry{
				"config": {
					Name: "config",
					Kind: yang.DirectoryEntry,
					Dir: map[string]*yang.Entry{
						"choice": {
							Kind: yang.ChoiceEntry,
							Dir: map[string]*yang.Entry{
								"case": {
									Kind: yang.CaseEntry,
									Dir: map[string]*yang.Entry{
										"string": {
											Name: "string",
										},
									},
								},
							},
						},
					},
				},
			},
		},
		wantCompressed: []yang.Entry{{
			Name: "string",
		}},
		wantUncompressed: []yang.Entry{{
			Name: "config",
		}},
	}, {
		name: "exclude container which is config false",
		inElement: &yang.Entry{
			Name:   "container",
			Kind:   yang.DirectoryEntry,
			Config: yang.TSFalse,
			Dir: map[string]*yang.Entry{
				"field": {Name: "field"},
			},
		},
		inExcludeState:   true,
		wantCompressed:   []yang.Entry{},
		wantUncompressed: []yang.Entry{},
	}, {
		name: "exclude leaf which is config false",
		inElement: &yang.Entry{
			Name:   "container",
			Kind:   yang.DirectoryEntry,
			Config: yang.TSTrue,
			Dir: map[string]*yang.Entry{
				"config-true":  {Name: "config-true"},
				"config-false": {Name: "config-false", Config: yang.TSFalse},
			},
		},
		inExcludeState:   true,
		wantCompressed:   []yang.Entry{{Name: "config-true"}},
		wantUncompressed: []yang.Entry{{Name: "config-true"}},
	}, {
		name: "exclude read-only list within a container with compression",
		inElement: &yang.Entry{
			Name:   "container",
			Kind:   yang.DirectoryEntry,
			Config: yang.TSTrue,
			Dir: map[string]*yang.Entry{
				"surrounding-container": {
					Name:   "surrounding-container",
					Kind:   yang.DirectoryEntry,
					Config: yang.TSTrue,
					Dir: map[string]*yang.Entry{
						"list": {
							Name:     "list",
							Config:   yang.TSFalse,
							Kind:     yang.DirectoryEntry,
							ListAttr: &yang.ListAttr{},
							Dir:      map[string]*yang.Entry{},
						},
					},
				},
			},
		},
		inExcludeState: true,
		wantCompressed: []yang.Entry{},
		wantUncompressed: []yang.Entry{{
			Name:   "surrounding-container",
			Config: yang.TSTrue,
		}},
	}}

	for _, tt := range tests {
		for compress, expected := range map[bool][]yang.Entry{true: tt.wantCompressed, false: tt.wantUncompressed} {
			elems, errs := FindAllChildren(tt.inElement, compress, tt.inExcludeState)
			if tt.wantErr == nil && errs != nil {
				t.Errorf("%s (compress: %v): errors %v for children of %s", tt.name, compress, errs, tt.inElement.Name)
			} else {
				if expErr, ok := tt.wantErr[compress]; ok {
					if (errs != nil) != expErr {
						t.Errorf("%s (compress: %v): did not get expected error", tt.name, compress)
					}
				}
			}

			retMap := map[string]*yang.Entry{}
			for _, elem := range elems {
				retMap[elem.Name] = elem
			}

			for _, expectEntry := range expected {
				if elem, ok := retMap[expectEntry.Name]; ok {
					delete(retMap, expectEntry.Name)
					if elem.Config != expectEntry.Config {
						t.Errorf("%s (compress: %v): element %s had wrong config status %s", tt.name, compress,
							expectEntry.Name, elem.Config)
					}
					if elem.Type != nil && elem.Type.Kind != expectEntry.Type.Kind {
						t.Errorf("%s (compress: %v): element %s had wrong type %s", tt.name,
							compress, expectEntry.Name, elem.Type.Kind)
					}
				} else {
					t.Errorf("%s (compress: %v): could not find expected child %s in %s", tt.name, compress,
						expectEntry.Name, tt.inElement.Name)
				}
			}

			if len(retMap) != 0 && expected != nil {
				t.Errorf("%s (compress: %v): got unexpected entries, got: %v, want: nil", tt.name, compress, retMap)
			}
		}
	}
}
