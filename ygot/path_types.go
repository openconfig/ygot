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

import (
	"fmt"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
)

const (
	// PathStructInterfaceName is the name for the interface implemented by all
	// generated path structs.
	PathStructInterfaceName = "PathStruct"
	// PathBaseTypeName is the type name of the common embedded struct
	// containing the path information for a path struct.
	PathBaseTypeName = "NodePath"
	// FakeRootBaseTypeName is the type name of the fake root struct which
	// should be embedded within the fake root path struct.
	FakeRootBaseTypeName = "DeviceRootBase"
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

// fakeRootPathStruct is an interface that is implemented by the fake root path
// struct type.
type fakeRootPathStruct interface {
	PathStruct
	Id() string
	CustomData() map[string]interface{}
}

func NewDeviceRootBase(id string) *DeviceRootBase {
	return &DeviceRootBase{NodePath: &NodePath{}, id: id, customData: map[string]interface{}{}}
}

// DeviceRootBase represents the fakeroot for all YANG schema elements.
type DeviceRootBase struct {
	*NodePath
	id string
	// customData is meant to store root-specific information that may be
	// useful to know when processing the resolved path. It is meant to be
	// accessible through a user-defined accessor.
	customData map[string]interface{}
}

// Id returns the device ID of the DeviceRootBase struct.
func (d *DeviceRootBase) Id() string {
	return d.id
}

// CustomData returns the customData field of the DeviceRootBase struct.
func (d *DeviceRootBase) CustomData() map[string]interface{} {
	return d.customData
}

// PutCustomData modifies an entry in the customData field of the DeviceRootBase struct.
func (d *DeviceRootBase) PutCustomData(key string, val interface{}) {
	d.customData[key] = val
}

// ResolvePath is a helper which returns the resolved *gpb.Path of a PathStruct
// node as well as the root node's customData.
func ResolvePath(n PathStruct) (*gpb.Path, map[string]interface{}, []error) {
	p := []*gpb.PathElem{}
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

	root, ok := n.(fakeRootPathStruct)
	if !ok {
		return nil, nil, append(errs, fmt.Errorf("ygot.ResolvePath(ygot.PathStruct): got unexpected root of (type, value) (%T, %v)", n, n))
	}
	return &gpb.Path{Target: root.Id(), Elem: p}, root.CustomData(), nil
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
