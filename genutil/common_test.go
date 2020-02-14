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
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/openconfig/goyang/pkg/yang"
)

func TestTranslateToCompressBehaviour(t *testing.T) {
	tests := []struct {
		inCompressPaths bool
		inExcludeState  bool
		inPreferState   bool
		want            CompressBehaviour
		wantErr         bool
	}{{
		want: Uncompressed,
	}, {
		inCompressPaths: true,
		want:            PreferIntendedConfig,
	}, {
		inExcludeState: true,
		want:           UncompressedExcludeDerivedState,
	}, {
		inPreferState: true,
		wantErr:       true,
	}, {
		inCompressPaths: true,
		inExcludeState:  true,
		want:            ExcludeDerivedState,
	}, {
		inCompressPaths: true,
		inPreferState:   true,
		want:            PreferOperationalState,
	}, {
		inExcludeState: true,
		inPreferState:  true,
		wantErr:        true,
	}, {
		inCompressPaths: true,
		inExcludeState:  true,
		inPreferState:   true,
		wantErr:         true,
	}}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("inCompressPaths %v, inExcludeState %v, inPreferState %v", tt.inCompressPaths, tt.inExcludeState, tt.inPreferState), func(t *testing.T) {
			got, err := TranslateToCompressBehaviour(tt.inCompressPaths, tt.inExcludeState, tt.inPreferState)
			if gotErr := err != nil; gotErr != tt.wantErr {
				t.Fatalf("gotErr: %v, wantErr: %v", err, tt.wantErr)
			}

			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

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

func TestGetOrderedEntryKeys(t *testing.T) {
	tests := []struct {
		name string
		in   map[string]*yang.Entry
		want []string
	}{{
		name: "nil map",
		in:   nil,
		want: nil,
	}, {
		name: "map with one entry",
		in: map[string]*yang.Entry{
			"a": {},
		},
		want: []string{"a"},
	}, {
		name: "multiple entries",
		in: map[string]*yang.Entry{
			"a": {},
			"b": {},
			"c": {},
			"d": {},
			"e": {},
			"f": {},
			"g": {},
		},
		want: []string{"a", "b", "c", "d", "e", "f", "g"},
	}, {
		name: "multiple entries 2 - non-alphabetical order",
		in: map[string]*yang.Entry{
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
		want: []string{"brown", "dog", "fox", "jumps", "lazy", "over", "quick", "the", "the2"},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if diff := cmp.Diff(tt.want, GetOrderedEntryKeys(tt.in)); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
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
		name      string
		inElement *yang.Entry
		want      map[CompressBehaviour][]yang.Entry
		// wantErr says whether a given compressBehaviour expects errors.
		wantErr map[CompressBehaviour]bool
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
		want: map[CompressBehaviour][]yang.Entry{
			PreferIntendedConfig: []yang.Entry{
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
			Uncompressed: []yang.Entry{
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
			ExcludeDerivedState: []yang.Entry{
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
			},
			UncompressedExcludeDerivedState: []yang.Entry{
				{
					Name:   "config",
					Config: yang.TSTrue,
					Type:   &yang.YangType{},
				},
				{
					Name:   "name",
					Config: yang.TSTrue,
					Type:   &yang.YangType{Kind: yang.Yleafref},
				},
			},
			PreferOperationalState: []yang.Entry{
				{
					Name:   "name",
					Config: yang.TSFalse,
					Type: &yang.YangType{
						Kind: yang.Ystring,
					},
				},
				{
					Name:   "type",
					Config: yang.TSFalse,
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
		}}, {
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
		want: map[CompressBehaviour][]yang.Entry{
			PreferIntendedConfig: []yang.Entry{
				{
					Name:   "singular",
					Config: yang.TSTrue,
					Type:   &yang.YangType{},
				},
			},
			PreferOperationalState: []yang.Entry{
				{
					Name:   "singular",
					Config: yang.TSTrue,
					Type:   &yang.YangType{},
				},
			},
			Uncompressed: []yang.Entry{
				{
					Name:   "plural",
					Config: yang.TSTrue,
					Type:   &yang.YangType{},
				},
			},
		}}, {
		name: "duplicate-elements-in-config",
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
		wantErr: map[CompressBehaviour]bool{
			ExcludeDerivedState:  true,
			PreferIntendedConfig: true,
		},
	}, {
		name: "duplicate-elements-in-state",
		inElement: &yang.Entry{
			Name:   "root",
			Config: yang.TSTrue,
			Type:   &yang.YangType{},
			Kind:   yang.DirectoryEntry,
			Dir: map[string]*yang.Entry{
				"name": {Name: "name"},
				"state": {
					Name:   "state",
					Config: yang.TSFalse,
					Type:   &yang.YangType{},
					Kind:   yang.DirectoryEntry,
					Dir: map[string]*yang.Entry{
						"name": {Name: "name"},
					},
				},
			},
		},
		wantErr: map[CompressBehaviour]bool{
			PreferIntendedConfig:   true,
			PreferOperationalState: true,
		},
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
		want: map[CompressBehaviour][]yang.Entry{
			PreferIntendedConfig: []yang.Entry{
				{
					Name: "option",
					Type: &yang.YangType{},
				},
			},
			PreferOperationalState: []yang.Entry{
				{
					Name: "option",
					Type: &yang.YangType{},
				},
			},
			Uncompressed: []yang.Entry{
				{
					Name: "option",
					Type: &yang.YangType{},
				},
			},
		}}, {
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
		want: map[CompressBehaviour][]yang.Entry{
			PreferIntendedConfig: []yang.Entry{{
				Name: "string",
			}},
			PreferOperationalState: []yang.Entry{{
				Name: "string",
			}},
			Uncompressed: []yang.Entry{{
				Name: "state",
			}},
		}}, {
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
		want: map[CompressBehaviour][]yang.Entry{
			PreferIntendedConfig: []yang.Entry{{
				Name: "string",
			}},
			PreferOperationalState: []yang.Entry{{
				Name: "string",
			}},
			Uncompressed: []yang.Entry{{
				Name: "config",
			}},
		}}, {
		name: "exclude container which is config false",
		inElement: &yang.Entry{
			Name:   "container",
			Kind:   yang.DirectoryEntry,
			Config: yang.TSFalse,
			Dir: map[string]*yang.Entry{
				"field": {Name: "field"},
			},
		},
		want: map[CompressBehaviour][]yang.Entry{
			ExcludeDerivedState:             []yang.Entry{},
			UncompressedExcludeDerivedState: []yang.Entry{},
		}}, {
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
		want: map[CompressBehaviour][]yang.Entry{
			ExcludeDerivedState:             []yang.Entry{{Name: "config-true"}},
			UncompressedExcludeDerivedState: []yang.Entry{{Name: "config-true"}},
		}}, {
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
		want: map[CompressBehaviour][]yang.Entry{
			ExcludeDerivedState: []yang.Entry{},
			UncompressedExcludeDerivedState: []yang.Entry{{
				Name:   "surrounding-container",
				Config: yang.TSTrue,
			}},
		}}}

	for _, tt := range tests {
		for _, c := range []CompressBehaviour{
			Uncompressed,
			UncompressedExcludeDerivedState,
			PreferIntendedConfig,
			PreferOperationalState,
			ExcludeDerivedState,
		} {
			// If this isn't a test case that has anything to test, we skip it.
			wantErr, ok := tt.wantErr[c]
			want := tt.want[c]
			if !ok && want == nil {
				// If not checking for either an error or output, then assume it's an uninteresting case.
				continue
			}

			t.Run(fmt.Sprintf("%s:(compressBehaviour:%v)", tt.name, c), func(t *testing.T) {
				elems, errs := FindAllChildren(tt.inElement, c)
				if !wantErr && errs != nil {
					t.Errorf("Unexpected errors %v for children of %s", errs, tt.inElement.Name)
				} else if wantErr && errs == nil {
					t.Error("Did not receive expected error")
				}

				retMap := map[string]*yang.Entry{}
				for _, elem := range elems {
					retMap[elem.Name] = elem
				}

				for _, expectEntry := range want {
					elem, ok := retMap[expectEntry.Name]
					if !ok {
						t.Errorf("Could not find expected child %s in %s", expectEntry.Name, tt.inElement.Name)
					}
					delete(retMap, expectEntry.Name)
					if elem.Config != expectEntry.Config {
						t.Errorf("Element %s had wrong config status %s", expectEntry.Name, elem.Config)
					}
					if elem.Type != nil && elem.Type.Kind != expectEntry.Type.Kind {
						t.Errorf("Element %s had wrong type %s", expectEntry.Name, elem.Type.Kind)
					}
				}

				if len(retMap) != 0 && want != nil {
					t.Errorf("Got unexpected entries, got: %v, want: nil", retMap)
				}
			})
		}
	}
}
