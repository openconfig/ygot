// Copyright 2018 Google Inc.
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

package pathtranslate

import (
	"reflect"
	"testing"

	"github.com/openconfig/gnmi/errdiff"
	"github.com/openconfig/goyang/pkg/yang"

	gnmipb "github.com/openconfig/gnmi/proto/gnmi"
)

func TestInstantiationOfTranslator(t *testing.T) {
	simpleSchema := &yang.Entry{
		Name: "simpleKeyedList",
		Key:  "k1",
		Parent: &yang.Entry{
			Name: "simpleKeyedLists",
			Parent: &yang.Entry{
				Name: "b",
				Parent: &yang.Entry{
					Name:   "a",
					Parent: &yang.Entry{Name: "root"},
				},
			},
		},
	}

	structKeyedSchema := &yang.Entry{
		Name: "structKeyedList",
		Key:  "k1 k2 k3",
		Parent: &yang.Entry{Name: "structKeyedLists",
			Parent: &yang.Entry{
				Name: "simpleKeyedList",
				Key:  "k1",
				Parent: &yang.Entry{
					Name: "simpleKeyedLists",
					Parent: &yang.Entry{
						Name: "b",
						Parent: &yang.Entry{
							Name:   "a",
							Parent: &yang.Entry{Name: "root"},
						},
					},
				},
			},
		},
	}

	tests := []struct {
		inDesc           string
		inSchemas        []*yang.Entry
		wantRules        map[string][]string
		wantErrSubstring string
	}{
		{
			inDesc:    "success with unique schema paths for keyed lists",
			inSchemas: []*yang.Entry{simpleSchema},
			wantRules: map[string][]string{
				"/a/b/simpleKeyedLists/simpleKeyedList": {"k1"},
			},
		},
		{
			inDesc:    "success with struct keyed schema",
			inSchemas: []*yang.Entry{simpleSchema, structKeyedSchema},
			wantRules: map[string][]string{
				"/a/b/simpleKeyedLists/simpleKeyedList":                                  {"k1"},
				"/a/b/simpleKeyedLists/simpleKeyedList/structKeyedLists/structKeyedList": {"k1", "k2", "k3"},
			},
		},
		{
			inDesc:           "fail with similar schema paths for keyed lists",
			inSchemas:        []*yang.Entry{simpleSchema, simpleSchema},
			wantErrSubstring: "got /a/b/simpleKeyedLists/simpleKeyedList path multiple times",
		},
	}

	for _, tt := range tests {
		t.Run(tt.inDesc, func(t *testing.T) {
			r, err := NewPathTranslator(tt.inSchemas)
			if diff := errdiff.Substring(err, tt.wantErrSubstring); diff != "" {
				t.Fatalf("diff: %v", diff)
			}
			if err != nil {
				return
			}
			if !reflect.DeepEqual(r.rules, tt.wantRules) {
				t.Errorf("got %v, want %v", r.rules, tt.wantRules)
			}
		})
	}
}

func TestPathElem(t *testing.T) {
	schemas := []*yang.Entry{
		{Name: "root"},
		{
			Name: "simpleKeyedList",
			Key:  "k1",
			Parent: &yang.Entry{
				Name: "simpleKeyedLists",
				Parent: &yang.Entry{
					Name: "b",
					Parent: &yang.Entry{
						Name:   "a",
						Parent: &yang.Entry{Name: "root"},
					},
				},
			},
		},
		{
			Name: "structKeyedList",
			Key:  "k1 k2 k3",
			Parent: &yang.Entry{Name: "structKeyedLists",
				Parent: &yang.Entry{
					Name: "simpleKeyedList",
					Key:  "k1",
					Parent: &yang.Entry{
						Name: "simpleKeyedLists",
						Parent: &yang.Entry{
							Name: "b",
							Parent: &yang.Entry{
								Name:   "a",
								Parent: &yang.Entry{Name: "root"},
							},
						},
					},
				},
			},
		},
	}

	tests := []struct {
		inDesc           string
		inPath           []string
		wantErrSubstring string
		wantPath         []*gnmipb.PathElem
	}{
		{
			inDesc: "success empty path",
			inPath: []string{},
		},
		{
			inDesc: "success path with no keyed list(note, it doesn't exist in schema)",
			inPath: []string{"a", "b"},
			wantPath: []*gnmipb.PathElem{
				{Name: "a"},
				{Name: "b"},
			},
		},
		{
			inDesc: "success path with keyed list at the end",
			inPath: []string{"a", "b", "simpleKeyedLists", "simpleKeyedList", "key1"},
			wantPath: []*gnmipb.PathElem{
				{Name: "a"},
				{Name: "b"},
				{Name: "simpleKeyedLists"},
				{Name: "simpleKeyedList", Key: map[string]string{"k1": "key1"}},
			},
		},
		{
			inDesc: "success path with keyed list followed by arbitrary elements",
			inPath: []string{"a", "b", "simpleKeyedLists", "simpleKeyedList", "key1", "arbitrary1", "arbitrary2"},
			wantPath: []*gnmipb.PathElem{
				{Name: "a"},
				{Name: "b"},
				{Name: "simpleKeyedLists"},
				{Name: "simpleKeyedList", Key: map[string]string{"k1": "key1"}},
				{Name: "arbitrary1"},
				{Name: "arbitrary2"},
			},
		},
		{
			inDesc: "success, but keys aren't treated as key for a path with keyed list after arbitrary elements",
			inPath: []string{"random1", "random2", "simpleKeyedLists", "simpleKeyedList", "NOT_TREATED_AS_KEY"},
			wantPath: []*gnmipb.PathElem{
				{Name: "random1"},
				{Name: "random2"},
				{Name: "simpleKeyedLists"},
				{Name: "simpleKeyedList"},
				{Name: "NOT_TREATED_AS_KEY"},
			},
		},
		{
			inDesc: "success path with keyed list in the middle",
			inPath: []string{"a", "b", "simpleKeyedLists", "simpleKeyedList", "key1", "arbitrary"},
			wantPath: []*gnmipb.PathElem{
				{Name: "a"},
				{Name: "b"},
				{Name: "simpleKeyedLists"},
				{Name: "simpleKeyedList", Key: map[string]string{"k1": "key1"}},
				{Name: "arbitrary"},
			},
		},
		{
			inDesc: "success path with struct keyed list",
			inPath: []string{"a", "b", "simpleKeyedLists", "simpleKeyedList", "key1", "structKeyedLists", "structKeyedList", "key1", "key2", "key3"},
			wantPath: []*gnmipb.PathElem{
				{Name: "a"},
				{Name: "b"},
				{Name: "simpleKeyedLists"},
				{Name: "simpleKeyedList", Key: map[string]string{"k1": "key1"}},
				{Name: "structKeyedLists"},
				{Name: "structKeyedList", Key: map[string]string{"k1": "key1", "k2": "key2", "k3": "key3"}},
			},
		},
		{
			inDesc:           "fail path due to insuffucient keys to fill the key struct",
			inPath:           []string{"a", "b", "simpleKeyedLists", "simpleKeyedList", "key1", "structKeyedLists", "structKeyedList", "key1", "key2"},
			wantErrSubstring: "got 2, want 3 keys for /a/b/simpleKeyedLists/simpleKeyedList/structKeyedLists/structKeyedList",
		},
	}
	r, err := NewPathTranslator(schemas)
	if err != nil {
		t.Errorf("failed to create path translator; %v", r)
	}
	for _, tt := range tests {
		t.Run(tt.inDesc, func(t *testing.T) {
			gotPath, err := r.PathElem(tt.inPath)
			if diff := errdiff.Substring(err, tt.wantErrSubstring); diff != "" {
				t.Errorf("diff: %v", diff)
				return
			}
			if err != nil {
				return
			}
			if !reflect.DeepEqual(gotPath, tt.wantPath) {
				t.Errorf("got %v, want %v", gotPath, tt.wantPath)
			}
		})
	}
}
