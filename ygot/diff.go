// Copyright 2018 Google Inc.
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
	"reflect"

	"github.com/golang/protobuf/proto"
	"github.com/openconfig/ygot/util"

	gnmipb "github.com/openconfig/gnmi/proto/gnmi"
)

// schemaPathTogNMIPath takes an input schema path represented as a slice of
// strings, and returns a gNMI Path using the v0.4.0 path format Elem field
// containing the elements. A schema path cannot specify any keys, and hence
// only the path element's name is populated.
func schemaPathTogNMIPath(path []string) *gnmipb.Path {
	p := &gnmipb.Path{}
	for _, pe := range path {
		p.Elem = append(p.Elem, &gnmipb.PathElem{Name: pe})
	}
	return p
}

// joingNMIPaths takes a parent and child gNMI path, and concatenates the
// child path to the parent path, returning the combined path. It only populates
// the v0.4.0 Elem field.
func joingNMIPaths(parent *gnmipb.Path, child *gnmipb.Path) *gnmipb.Path {
	p := proto.Clone(parent).(*gnmipb.Path)
	for _, e := range child.Elem {
		p.Elem = append(p.Elem, e)
	}
	return p
}

// pathSpec is a wrapper type used to store a gNMI path for use as a map key.
type pathSpec struct {
	// gNMIPaths is the set of gNMI paths that the path represents.
	gNMIPaths []*gnmipb.Path
}

// nodeValuePath takes an input util.NodeInfo struct describing an element within
// a GoStruct tree (be it a leaf, leaf-list, container or list) and returns the
// set of paths that the value represents as a pathSpec pointer.
func nodeValuePath(ni *util.NodeInfo, schemaPaths [][]string) (*pathSpec, error) {
	if ni.Parent == nil || ni.Parent.Annotation == nil {
		return nodeRootPath(schemaPaths), nil
	}

	cp, err := getPathSpec(ni.Parent)
	if err != nil {
		return nil, err
	}

	if l, ok := ni.FieldValue.Interface().(KeyHelperGoStruct); ok {
		return nodeMapPath(l, cp)
	}

	return nodeChildPath(cp, schemaPaths)
}

// nodeMapPath takes an input list entry (which is a value of a Go map in the
// generated code) and the path of the YANG list node's parent (which stores the
// name of the YANG list) and returns the data tree path of the map. For a
// struct of the form:
//
//  type Foo struct {
//   YANGList map[string]*Foo_Child `path:"yang-list"`
//  }
//
//  type Foo_Child struct {
//   KeyValue *string `path:"key-value"`
//  }
//
// The parentPath handed to this function is "/yang-list" since this is the
// path of the YANGList struct field. The full data tree path of the list entry
// is formed by appending the key of the Foo_Child struct. In the generated Go
// code, Foo_Child implements the ListKeyHelper interface, and hence the keys are
// found by calling the ΛListKeyMap helper function.
func nodeMapPath(list KeyHelperGoStruct, parentPath *pathSpec) (*pathSpec, error) {
	keys, err := list.ΛListKeyMap()
	if err != nil {
		return nil, err
	}

	// Convert the keys into a string.
	strkeys, err := keyMapAsStrings(keys)
	if err != nil {
		return nil, fmt.Errorf("cannot convert keys to map[string]string: %v", err)
	}

	if parentPath == nil || parentPath.gNMIPaths == nil {
		// we cannot have a list member that does not have a list parent.
		return nil, fmt.Errorf("invalid list member with no parent")
	}

	gPaths := []*gnmipb.Path{}
	for _, p := range parentPath.gNMIPaths {
		np := proto.Clone(p).(*gnmipb.Path)
		np.Elem[len(p.Elem)-1].Key = strkeys
		gPaths = append(gPaths, np)
	}
	return &pathSpec{
		gNMIPaths: gPaths,
	}, nil
}

// nodeRootPath returns the gNMI path of a node at the root of a GoStruct tree -
// since such nodes do not have a parent, then the path returned is entirely
// gleaned from the schema path supplied.
func nodeRootPath(schemaPaths [][]string) *pathSpec {
	gPaths := []*gnmipb.Path{}
	for _, sp := range schemaPaths {
		gPaths = append(gPaths, schemaPathTogNMIPath(sp))
	}

	return &pathSpec{
		gNMIPaths: gPaths,
	}
}

// nodeChildPath returns the gNMI path of a node that is a child of the supplied
// parentPath, the schemaPaths of the child are appended to each entry in the
// parent's path.
func nodeChildPath(parentPath *pathSpec, schemaPaths [][]string) (*pathSpec, error) {
	if parentPath == nil || parentPath.gNMIPaths == nil {
		return nil, fmt.Errorf("could not find annotation for complete path")
	}

	gPaths := []*gnmipb.Path{}
	for _, p := range parentPath.gNMIPaths {
		for _, s := range schemaPaths {
			gPaths = append(gPaths, joingNMIPaths(p, schemaPathTogNMIPath(s)))
		}
	}

	return &pathSpec{
		gNMIPaths: gPaths,
	}, nil
}

// getPathSpec extracts the pathSpec pointer from the supplied NodeInfo's annotations.
func getPathSpec(ni *util.NodeInfo) (*pathSpec, error) {
	for _, a := range ni.Annotation {
		if p, ok := a.(*pathSpec); ok {
			return p, nil
		}
	}
	return nil, fmt.Errorf("could not find path specification annotation")
}

// findSetLeaves iteratively walks the fields of the supplied GoStruct, s, and
// returns a map, keyed by the path of the leaves that are set, with a the value
// that the leaf is set to. YANG lists (Go maps), and containers (Go structs) are
// not included within the returned map, such that only leaf or leaf-list values
// that are set are returned.
//
// The ForEachDataField helper of the util library is used to perform the iterative
// walk of the struct - using the out argument to store the set of changed leaves.
// A specific Annotation is used to store the absolute path of the entity during
// the walk.
func findSetLeaves(s GoStruct) (map[*pathSpec]interface{}, error) {
	findSetIterFunc := func(ni *util.NodeInfo, in, out interface{}) (errs util.Errors) {
		if reflect.DeepEqual(ni.StructField, reflect.StructField{}) {
			return
		}

		sp, err := util.SchemaPaths(ni.StructField)
		if err != nil {
			errs = util.AppendErr(errs, err)
			return
		}
		if len(sp) == 0 {
			errs = util.AppendErr(errs, fmt.Errorf("invalid schema path for %s", ni.StructField.Name))
			return
		}

		vp, err := nodeValuePath(ni, sp)
		if err != nil {
			return util.NewErrs(err)
		}
		ni.Annotation = []interface{}{vp}

		if util.IsNilOrInvalidValue(ni.FieldValue) || util.IsValueStructPtr(ni.FieldValue) || util.IsValueMap(ni.FieldValue) {
			return
		}

		outs := out.(map[*pathSpec]interface{})
		outs[vp] = ni.FieldValue.Interface()

		return
	}

	out := map[*pathSpec]interface{}{}
	if errs := util.ForEachDataField(s, nil, out, findSetIterFunc); errs != nil {
		return nil, fmt.Errorf("error from ForEachDataField iteration: %v", errs)
	}

	return out, nil
}
