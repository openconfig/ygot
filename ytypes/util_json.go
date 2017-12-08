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

package ytypes

import (
	"fmt"
	"reflect"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/util"
)

// getJSONTreeValForField returns the JSON subtree of the provided tree that
// corresponds to any of the paths in struct field f.
// If no such JSON subtree exists, it returns nil, nil.
// If more than one path has a JSON subtree, the function returns an error if
// the two subtrees are unequal.
func getJSONTreeValForField(parentSchema, schema *yang.Entry, f reflect.StructField, tree interface{}) (interface{}, error) {
	ps, err := dataTreePaths(parentSchema, schema, f)
	if err != nil {
		return nil, err
	}
	var out interface{}
	var outPath []string
	for _, p := range ps {
		if jr, ok := getJSONTreeValForPath(tree, p); ok {
			if out != nil && !reflect.DeepEqual(out, jr) {
				return nil, fmt.Errorf("values at paths %v and %v are different: %v != %v", outPath, p, out, jr)
			}
			out = jr
			outPath = p
		}
	}

	return out, nil
}

// getJSONTreeValForPath returns a JSON subtree from tree at the given path from
// the root. If returns (nil, false) if no subtree is found at the given path.
func getJSONTreeValForPath(tree interface{}, path []string) (interface{}, bool) {
	if len(path) == 0 {
		return tree, true
	}

	t, ok := tree.(map[string]interface{})
	if !ok {
		return nil, false
	}

	for k, v := range t {
		if path[0] == util.StripModulePrefix(k) {
			if ret, ok := getJSONTreeValForPath(v, path[1:]); ok {
				return ret, true
			}
		}
	}
	return nil, false
}
