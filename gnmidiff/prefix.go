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
	"strings"
)

// prefixSearch returns the subsequence of paths (a sorted list)
// consisting of elements that have path + "/" as a prefix.
func prefixSearch(paths []string, path string) []string {
	// BinarySearchFunc finds the first index in paths where element is >= target + "/".
	start, _ := slices.BinarySearchFunc(paths, path, func(element, target string) int {
		L := len(target)
		if len(element) < L {
			return strings.Compare(element, target)
		}
		if sub := strings.Compare(element[:L], target); sub != 0 {
			return sub
		}
		if len(element) == L || element[L] < '/' {
			return -1
		}
		return 1
	})

	end := start
	for end < len(paths) &&
		len(paths[end]) > len(path) &&
		paths[end][len(path)] == '/' &&
		strings.HasPrefix(paths[end], path) {
		end++
	}
	return paths[start:end]
}
