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

// addNewKeys appends entries from the newKeys string slice to the
// existing map if the entry is not an existing key. The existing
// map is modified in place.
func addNewKeys(existing map[string]interface{}, newKeys []string) {
	for _, n := range newKeys {
		if _, ok := existing[n]; !ok {
			existing[n] = true
		}
	}
}

// stringKeys returns the keys of the supplied map as a slice of strings.
func stringKeys(m map[string]interface{}) []string {
	var ss []string
	for k := range m {
		ss = append(ss, k)
	}
	return ss
}

// resolveRootName resolves the name of the fakeroot by taking configuration
// and the default values, along with a boolean indicating whether the fake
// root is to be generated. It returns an empty string if the root is not
// to be generated.
func resolveRootName(name, defName string, generateRoot bool) string {
	if !generateRoot {
		return ""
	}

	if name == "" {
		return defName
	}

	return name
}
