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
	"github.com/golang/protobuf/proto"
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

// PathMatchesPrefix reports whether prefix is a prefix of path.
func PathMatchesPrefix(path *gpb.Path, prefix []string) bool {
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

// PathMatchesPathElemPrefix checks whether prefix is a prefix of path. Both paths
// must use the gNMI >=0.4.0 PathElem path format.
func PathMatchesPathElemPrefix(path, prefix *gpb.Path) bool {
	if len(path.GetElem()) < len(prefix.GetElem()) || path.Origin != prefix.Origin {
		return false
	}
	for i, v := range prefix.Elem {
		if !proto.Equal(v, path.GetElem()[i]) {
			return false
		}
	}
	return true
}

// TrimGNMIPathPrefix returns path with the prefix trimmed. It returns the
// original path if the prefix does not fully match.
func TrimGNMIPathPrefix(path *gpb.Path, prefix []string) *gpb.Path {
	for len(prefix) != 0 && prefix[len(prefix)-1] == "" {
		prefix = prefix[:len(prefix)-1]
	}
	if !PathMatchesPrefix(path, prefix) {
		return path
	}
	out := *path
	out.Elem = out.GetElem()[len(prefix):]
	return &out
}

// TrimGNMIPathElemPrefix returns the path with the prefix trimmed. It returns
// the original path if the prefix does not match.
func TrimGNMIPathElemPrefix(path, prefix *gpb.Path) *gpb.Path {
	if prefix == nil {
		return path
	}
	if !PathMatchesPathElemPrefix(path, prefix) {
		return path
	}
	out := proto.Clone(path).(*gpb.Path)
	out.Elem = out.GetElem()[len(prefix.GetElem()):]
	return out
}

// FindPathElemPrefix finds the longest common prefix of the paths specified.
func FindPathElemPrefix(paths []*gpb.Path) *gpb.Path {
	var prefix *gpb.Path
	i := 0
	for {
		var elem *gpb.PathElem
		for _, e := range paths {
			switch {
			case i >= len(e.Elem):
				return prefix
			case elem == nil:
				// Only happens on the first iteration through the
				// loop, so we use this as the base element to
				// compare the other paths to.
				elem = e.Elem[i]
			case !proto.Equal(e.Elem[i], elem):
				return prefix
			}
		}
		if prefix == nil {
			prefix = &gpb.Path{}
		}
		prefix.Elem = append(prefix.Elem, elem)
		i++
	}
}

// PopGNMIPath returns the supplied GNMI path with the first path element
// removed. If the path is empty, it returns an empty path.
func PopGNMIPath(path *gpb.Path) *gpb.Path {
	if len(path.GetElem()) == 0 {
		return path
	}
	return &gpb.Path{
		Elem: path.GetElem()[1:],
	}
}
