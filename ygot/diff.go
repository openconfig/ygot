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
	"github.com/kylelemons/godebug/pretty"
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

// pathSpec is a wrapper type used to store a set of gNMI paths to which
// a value within a GoStruct corresponds to.
type pathSpec struct {
	// gNMIPaths is the set of gNMI paths that the path represents.
	gNMIPaths []*gnmipb.Path
}

// Equal compares two pathSpecs, returning true if all paths within the pathSpec
// are matched.
func (p *pathSpec) Equal(o *pathSpec) bool {
	if p == nil || o == nil {
		return p == o
	}

	for _, path := range p.gNMIPaths {
		var found bool
		for _, otherPath := range o.gNMIPaths {
			if reflect.DeepEqual(path, otherPath) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// String returns a string representation of the pathSpec p.
func (p *pathSpec) String() string {
	s := pretty.Sprint(p)
	if len(p.gNMIPaths) != 0 {
		if ps, err := PathToString(p.gNMIPaths[0]); err == nil {
			s = ps
		}
	}
	return s
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
func findSetLeaves(s GoStruct, opts ...DiffOpt) (map[*pathSpec]interface{}, error) {
	pathOpt := hasDiffPathOpt(opts)

	findSetIterFunc := func(ni *util.NodeInfo, in, out interface{}) (errs util.Errors) {
		if reflect.DeepEqual(ni.StructField, reflect.StructField{}) {
			return
		}

		// Handle the case of having an annotated struct - in the diff case we
		// do not process schema annotations.
		if util.IsYgotAnnotation(ni.StructField) {
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

		// If the path options specify that each value should only be mapped to
		// a single path, then choose the most specific path.
		if pathOpt != nil && pathOpt.MapToSinglePath {
			sp = [][]string{leastSpecificPath(sp)}
		}

		vp, err := nodeValuePath(ni, sp)
		if err != nil {
			return util.NewErrs(err)
		}
		ni.Annotation = []interface{}{vp}

		if util.IsNilOrInvalidValue(ni.FieldValue) || util.IsValueStructPtr(ni.FieldValue) || util.IsValueMap(ni.FieldValue) {
			return
		}

		ival := ni.FieldValue.Interface()

		// If this is an enumerated value in the output structs, then check whether
		// it is set. Only include values that are set to a non-zero value.
		if _, isEnum := ival.(GoEnum); isEnum {
			if ni.FieldValue.Int() == 0 {
				return
			}
		}

		outs := out.(map[*pathSpec]interface{})
		outs[vp] = ival

		return
	}

	out := map[*pathSpec]interface{}{}
	if errs := util.ForEachDataField(s, nil, out, findSetIterFunc); errs != nil {
		return nil, fmt.Errorf("error from ForEachDataField iteration: %v", errs)
	}

	uOut := map[*pathSpec]interface{}{}
	// Deduplicate the list, since the iteration function will be called
	// multiple times for path tags that have >1 element.
	for ok, ov := range out {
		var skip bool
		for uk := range uOut {
			if ok.Equal(uk) {
				// This is a duplicate path, so we do not need to append it to the list.
				skip = true
			}
		}
		if !skip {
			uOut[ok] = ov
		}
	}

	return uOut, nil
}

// hasDiffPathOpt extracts a DiffPathOpt from the opts slice provided. In
// the case that there are multiple DiffPathOpt structs within opts slice, the
// first is returned.
func hasDiffPathOpt(opts []DiffOpt) *DiffPathOpt {
	for _, o := range opts {
		switch o.(type) {
		case *DiffPathOpt:
			return o.(*DiffPathOpt)
		}
	}
	return nil
}

// leastSpecificPath returns the path with the shortest length from the supplied
// paths slice. If the slice contains two paths that are equal in length, the
// first one encountered in the slice is returned.
func leastSpecificPath(paths [][]string) []string {
	var shortPath []string
	for _, p := range paths {
		if shortPath == nil {
			shortPath = p
		}

		if len(p) < len(shortPath) {
			shortPath = p
		}
	}

	return shortPath
}

// appendUpdate adds an update to the supplied gNMI Notification message corresponding
// to the path and value supplied.
func appendUpdate(n *gnmipb.Notification, path *pathSpec, val interface{}) error {
	v, err := EncodeTypedValue(val, gnmipb.Encoding_PROTO)
	if err != nil {
		return fmt.Errorf("cannot represent field value %v as TypedValue for path %v: %v", val, path, err)
	}
	for _, p := range path.gNMIPaths {
		n.Update = append(n.Update, &gnmipb.Update{
			Path: p,
			Val:  v,
		})
	}
	return nil
}

// DiffOpt is an interface that is implemented by the options to the Diff
// function. It allows user specified options to be propagated to the diff
// method.
type DiffOpt interface {
	// IsDiffOpt is a marker method for each DiffOpt.
	IsDiffOpt()
}

// DiffPathOpt is a DiffOpt that allows control of the path behaviour of the
// Diff function.
type DiffPathOpt struct {
	// MapToSinglePath specifies whether a single ygot.GoStruct field should
	// be mapped to more than one value. If set to true, when a struct tag
	// annotation specifies more than one path (e.g., `path:"foo|config/foo"`)
	// only the shortest path is mapped to.
	//
	// This option is primarily used where path compression has been used in the
	// generated structs, which can result in duplication of list key leaves in
	// the diff output.
	MapToSinglePath bool
}

// IsDiffOpt marks DiffPathOpt as a diff option.
func (*DiffPathOpt) IsDiffOpt() {}

// Diff takes an original and modified GoStruct, which must be of the same type
// and returns a gNMI Notification that contains the diff between them. The original
// struct is considered as the "from" data, with the modified struct the "to" such that:
//
//  - The contents of the Update field of the notification indicate that the
//    field in modified was either not present in original, or had a different
//    field value.
//  - The paths within the Delete field of the notification indicate that the
//    field was not present in the modified struct, but was set in the original.
//
// Annotation fields that are contained within the supplied original or modified
// GoStruct are skipped.
//
// A set of options for diff's behaviour, as specified by the supplied DiffOpts
// can be used to modify the behaviour of the Diff function per the individual
// option's specification.
//
// The returned gNMI Notification cannot be put on the wire unmodified, since
// it does not specify a timestamp - and may not contain the absolute paths
// to the fields specified if a GoStruct that does not represent the root of
// a YANG schema tree is not supplied as original and modified.
func Diff(original, modified GoStruct, opts ...DiffOpt) (*gnmipb.Notification, error) {

	if reflect.TypeOf(original) != reflect.TypeOf(modified) {
		return nil, fmt.Errorf("cannot diff structs of different types, original: %T, modified: %T", original, modified)
	}

	origLeaves, err := findSetLeaves(original, opts...)
	if err != nil {
		return nil, fmt.Errorf("could not extract set leaves from original struct: %v", err)
	}

	modLeaves, err := findSetLeaves(modified, opts...)
	if err != nil {
		return nil, fmt.Errorf("could not extract set leaves from modified struct: %v", err)
	}

	matched := map[*pathSpec]bool{}
	n := &gnmipb.Notification{}
	for origPath, origVal := range origLeaves {
		var origMatched bool
		for modPath, modVal := range modLeaves {
			if origPath.Equal(modPath) {
				// This path is set in both of the structs, so check whether the value
				// is equal.
				matched[modPath] = true
				origMatched = true
				if !reflect.DeepEqual(origVal, modVal) {
					// The contents of the value should indicate that value a has changed
					// to value b.
					if err := appendUpdate(n, origPath, modVal); err != nil {
						return nil, err
					}
				}
			}
		}
		if !origMatched {
			// This leaf was set in the original struct, but not in the modified
			// struct, therefore it has been deleted.
			for _, p := range origPath.gNMIPaths {
				n.Delete = append(n.Delete, p)
			}
		}
	}

	// Check that all paths that are in the modified struct have been examined, if
	// not they are updates.
	for modPath, modVal := range modLeaves {
		if !matched[modPath] {
			if err := appendUpdate(n, modPath, modVal); err != nil {
				return nil, err
			}
		}
	}

	return n, nil
}
