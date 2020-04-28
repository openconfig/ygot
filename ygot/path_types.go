// Copyright 2020 Google Inc.
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

package ygot

import gpb "github.com/openconfig/gnmi/proto/gnmi"

const (
	// PathStructInterfaceName is the name for the interface implemented by all
	// generated path structs.
	PathStructInterfaceName = "PathStruct"
	// PathBaseTypeName is the type name of the common embedded struct
	// containing the path information for a path struct.
	PathBaseTypeName = "NodePath"
)

// PathStruct is an interface that is implemented by any generated path struct
// type; it allows for generic handling of a path struct at any node.
type PathStruct interface {
	parent() PathStruct
	relPath() ([]*gpb.PathElem, []error)
}

// NewNodePath is the constructor for NodePath.
func NewNodePath(relSchemaPath []string, keys map[string]interface{}, p PathStruct) *NodePath {
	return &NodePath{relSchemaPath: relSchemaPath, keys: keys, p: p}
}

// NodePath is a common embedded type within all path structs. It
// keeps track of the necessary information to create the relative schema path
// as a []*gpb.PathElem during later processing using the Resolve() method,
// thereby delaying any errors being reported until that time.
type NodePath struct {
	relSchemaPath []string
	keys          map[string]interface{}
	p             PathStruct
}

// ResolvePath is a helper which returns the root PathStruct and absolute path
// of a PathStruct node.
func ResolvePath(n PathStruct) (PathStruct, []*gpb.PathElem, []error) {
	var p []*gpb.PathElem
	var errs []error
	for ; n.parent() != nil; n = n.parent() {
		rel, es := n.relPath()
		if es != nil {
			errs = append(errs, es...)
			continue
		}
		p = append(rel, p...)
	}
	if errs != nil {
		return nil, nil, errs
	}
	return n, p, nil
}

// ResolveRelPath returns the partial []*gpb.PathElem representing the
// PathStruct's relative path.
func ResolveRelPath(n PathStruct) ([]*gpb.PathElem, []error) {
	return n.relPath()
}

// ModifyKey updates a NodePath's key value.
func ModifyKey(n *NodePath, name string, value interface{}) {
	n.keys[name] = value
}

// relPath converts the information stored in NodePath into the partial
// []*gpb.PathElem representing the node's relative path.
func (n *NodePath) relPath() ([]*gpb.PathElem, []error) {
	var pathElems []*gpb.PathElem
	for _, name := range n.relSchemaPath {
		pathElems = append(pathElems, &gpb.PathElem{Name: name})
	}
	if len(n.keys) == 0 {
		return pathElems, nil
	}

	var errs []error
	keys := make(map[string]string)
	for name, val := range n.keys {
		var err error
		// TODO(wenbli): It is ideal to also implement leaf restriction validation.
		if keys[name], err = KeyValueAsString(val); err != nil {
			errs = append(errs, err)
		}
	}
	if errs != nil {
		return nil, errs
	}
	pathElems[len(pathElems)-1].Key = keys
	return pathElems, nil
}

func (n *NodePath) parent() PathStruct { return n.p }
