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

package util

import (
	"fmt"
	"sort"
	"strings"

	"github.com/openconfig/goyang/pkg/yang"
	"google.golang.org/protobuf/proto"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
)

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

// PathElemsEqual replaces the proto.Equal() check for PathElems.
// This significantly improves comparison speed.
func PathElemsEqual(a, b *gpb.PathElem) bool {
	// This check allows avoiding to deal with any null PathElems later on.
	if a == nil || b == nil {
		return a == nil && b == nil
	}

	if a.Name != b.Name {
		return false
	}
	if len(a.Key) != len(b.Key) {
		return false
	}

	for k, v := range a.Key {
		if vo, ok := b.Key[k]; !ok || vo != v {
			return false
		}
	}
	return true
}

// PathElemSlicesEqual compares whether two PathElem slices are equal.
func PathElemSlicesEqual(a, b []*gpb.PathElem) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !PathElemsEqual(a[i], b[i]) {
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
		if !PathElemsEqual(v, path.GetElem()[i]) {
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
	out := proto.Clone(path).(*gpb.Path)
	out.Elem = out.GetElem()[len(prefix):]
	return out
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
			case !PathElemsEqual(e.Elem[i], elem):
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

type modelDataProto []*gpb.ModelData

func (m modelDataProto) Less(a, b int) bool { return m[a].Name < m[b].Name }
func (m modelDataProto) Len() int           { return len(m) }
func (m modelDataProto) Swap(a, b int)      { m[a], m[b] = m[b], m[a] }

// FindModelData takes an input slice of yang.Entry pointers, which are assumed to
// represent YANG modules, and returns the gNMI ModelData that corresponds with each
// of the input modules.
func FindModelData(mods []*yang.Entry) ([]*gpb.ModelData, error) {
	modelData := modelDataProto{}
	for _, mod := range mods {
		mNode, ok := mod.Node.(*yang.Module)
		if !ok || mNode == nil {
			return nil, fmt.Errorf("nil node, or not a module for node %s", mod.Name)
		}
		md := &gpb.ModelData{
			Name: mod.Name,
		}

		if mNode.Organization != nil {
			md.Organization = mNode.Organization.Statement().Argument
		}

		for _, e := range mNode.Exts() {
			if p := strings.Split(e.Keyword, ":"); len(p) == 2 && p[1] == "openconfig-version" {
				md.Version = e.Argument
				break
			}
		}

		modelData = append(modelData, md)
	}

	sort.Sort(modelData)

	return modelData, nil
}
