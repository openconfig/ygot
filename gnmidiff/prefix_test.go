// Copyright 2023 Google Inc.
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

package gnmidiff

import (
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestPrefixSearch(t *testing.T) {
	tests := []struct {
		name  string
		paths []string
		path  string
		want  []string
	}{
		{
			name:  "empty paths",
			paths: nil,
			path:  "/a",
			want:  nil,
		},
		{
			name:  "exact match only - no children",
			paths: []string{"/a"},
			path:  "/a",
			want:  nil,
		},
		{
			name:  "children matching prefix",
			paths: []string{"/a", "/a/b", "/a/b/c"},
			path:  "/a",
			want:  []string{"/a/b", "/a/b/c"},
		},
		{
			name:  "similar prefix with hyphen not matched",
			paths: []string{"/a", "/a-b", "/a/b", "/a/c", "/b"},
			path:  "/a",
			want:  []string{"/a/b", "/a/c"},
		},
		{
			name:  "key attributes containing hyphen",
			paths: []string{"/interfaces/interface[name=eth0]", "/interfaces/interface[name=eth0-0]", "/interfaces/interface[name=eth0]/config"},
			path:  "/interfaces/interface[name=eth0]",
			want:  []string{"/interfaces/interface[name=eth0]/config"},
		},
		{
			name:  "no matches in list",
			paths: []string{"/a", "/b", "/c"},
			path:  "/d",
			want:  nil,
		},
	}

	for _, tt := range tests {
		slices.Sort(tt.paths)
		t.Run(tt.name, func(t *testing.T) {
			got := prefixSearch(tt.paths, tt.path)
			if diff := cmp.Diff(tt.want, got, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("prefixSearch(%v, %q) returned diff (-want +got):\n%s", tt.paths, tt.path, diff)
			}
		})
	}
}
