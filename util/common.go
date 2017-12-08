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

// Package util implements utlity functions used in ygot.
package util

import (
	"github.com/openconfig/goyang/pkg/yang"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
)

var (
	// YangMaxNumber represents the maximum value for any integer type.
	YangMaxNumber = yang.Number{Kind: yang.MaxNumber}
	// YangMinNumber represents the minimum value for any integer type.
	YangMinNumber = yang.Number{Kind: yang.MinNumber}
)

// stringMapKeys returns the keys for map m.
func stringMapKeys(m map[string]*yang.Entry) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}

// TODO(mostrowski): move below functions into path package.

// pathMatchesPrefix reports whether prefix is a prefix of path.
func pathMatchesPrefix(path *gpb.Path, prefix []string) bool {
	if len(path.GetElem()) < len(prefix) {
		return false
	}
	for len(prefix) != 0 && prefix[len(prefix)-1] == "" {
		prefix = prefix[:len(prefix)-1]
	}
	for i := range prefix {
		if prefix[i] != path.GetElem()[i].GetName() {
			return false
		}
	}

	return true
}

// trimGNMIPathPrefix returns path with the prefix trimmed. It returns the
// original path if the prefix does not fully match.
func trimGNMIPathPrefix(path *gpb.Path, prefix []string) *gpb.Path {
	for len(prefix) != 0 && prefix[len(prefix)-1] == "" {
		prefix = prefix[:len(prefix)-1]
	}
	if !pathMatchesPrefix(path, prefix) {
		return path
	}
	out := *path
	out.Elem = out.GetElem()[len(prefix):]
	return &out
}

// popGNMIPath returns the supplied GNMI path with the first path element
// removed. If the path is empty, it returns an empty path.
func popGNMIPath(path *gpb.Path) *gpb.Path {
	if len(path.GetElem()) == 0 {
		return path
	}
	return &gpb.Path{
		Elem: path.GetElem()[1:],
	}
}
