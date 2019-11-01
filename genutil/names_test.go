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
	"testing"

	"github.com/openconfig/goyang/pkg/yang"
)

// TestCamelCase tests the functionality that is provided by MakeNameUnique and
// EntryCamelCaseName - ensuring
// that following being converted to CamelCase, a name is unique within the set of
// entities that have been generated already by the YANGCodeGenerator implementation.
func TestCamelCase(t *testing.T) {
	tests := []struct {
		name        string        // name is the test name.
		inPrevNames []*yang.Entry // inPrevNames is a set of names that have already been processed.
		inEntry     *yang.Entry   // inName is the name that we are testing.
		wantName    string        // wantName is the name that we expect for inName post conversion.
	}{{
		name:     "basic CamelCase test",
		inEntry:  &yang.Entry{Name: "leaf-one"},
		wantName: "LeafOne",
	}, {
		name:     "single word",
		inEntry:  &yang.Entry{Name: "leaf"},
		wantName: "Leaf",
	}, {
		name:     "already camelcase",
		inEntry:  &yang.Entry{Name: "AlreadyCamelCase"},
		wantName: "AlreadyCamelCase",
	}, {
		name:        "already defined",
		inPrevNames: []*yang.Entry{{Name: "interfaces"}},
		inEntry:     &yang.Entry{Name: "interfaces"},
		wantName:    "Interfaces_",
	}, {
		name:        "already defined twice",
		inPrevNames: []*yang.Entry{{Name: "interfaces"}, {Name: "interfaces"}},
		inEntry:     &yang.Entry{Name: "Interfaces"},
		wantName:    "Interfaces__",
	}, {
		name: "camelcase extension",
		inEntry: &yang.Entry{
			Name: "foobar",
			Exts: []*yang.Statement{{
				Keyword:     "some-module:camelcase-name",
				HasArgument: true,
				Argument:    "FooBar",
			}},
		},
		wantName: "FooBar",
	}, {
		name:        "camelcase extension with clashing name",
		inPrevNames: []*yang.Entry{{Name: "FishChips"}},
		inEntry: &yang.Entry{
			Name: "fish-chips",
			Exts: []*yang.Statement{{
				Keyword:     "anothermodule:camelcase-name",
				HasArgument: true,
				Argument:    `"FishChips\n"`,
			}},
		},
		wantName: "FishChips_",
	}, {
		name: "non-camelcase extension",
		inEntry: &yang.Entry{
			Name: "little-creatures",
			Exts: []*yang.Statement{{
				Keyword:     "amod:other-ext",
				HasArgument: true,
				Argument:    "true\n",
			}},
		},
		wantName: "LittleCreatures",
	}}

	for _, tt := range tests {
		ctx := make(map[string]bool)
		for _, prevName := range tt.inPrevNames {
			_ = MakeNameUnique(EntryCamelCaseName(prevName), ctx)
		}

		if got := MakeNameUnique(EntryCamelCaseName(tt.inEntry), ctx); got != tt.wantName {
			t.Errorf("%s: did not get expected name for %v (after defining %v): %s",
				tt.name, tt.inEntry, tt.inPrevNames, got)
		}
	}
}
