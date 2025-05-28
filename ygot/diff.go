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
	"sort"
	"strings"

	"github.com/kylelemons/godebug/pretty"
	"github.com/openconfig/gnmi/errlist"
	"github.com/openconfig/ygot/internal/yreflect"
	"github.com/openconfig/ygot/util"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"

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
	p.Elem = append(p.Elem, child.Elem...)
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
			if proto.Equal(path, otherPath) {
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
//	type Foo struct {
//	 YANGList map[string]*Foo_Child `path:"yang-list"`
//	}
//
//	type Foo_Child struct {
//	 KeyValue *string `path:"key-value"`
//	}
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

// pathInfo contains the path-value information for a single gNMI leaf.
type pathInfo struct {
	val  interface{}
	path *gnmipb.Path
}

// toStringPathMap converts the pathSpec-value map to a simple path-value map.
func toStringPathMap(pathMap map[*pathSpec]interface{}) (map[string]*pathInfo, error) {
	strPathMap := map[string]*pathInfo{}
	for paths, val := range pathMap {
		for _, path := range paths.gNMIPaths {
			strPath, err := PathToString(path)
			if err != nil {
				return nil, err
			}
			strPathMap[strPath] = &pathInfo{
				val:  val,
				path: path,
			}
		}
	}
	return strPathMap, nil
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
//
// - orderedMapAsLeaf=true specifies that ordered maps (GoOrderedMap
// interface) will be treated as a leaf and will be returned as-is instead of
// being walked and its leaves populated.
func findSetLeaves(s GoStruct, orderedMapAsLeaf bool, opts ...DiffOpt) (map[*pathSpec]interface{}, error) {
	pathOpt := hasDiffPathOpt(opts)
	processedPaths := map[string]bool{}

	findSetIterFunc := func(ni *util.NodeInfo, in, out interface{}) (action util.IterationAction, errs util.Errors) {
		if reflect.DeepEqual(ni.StructField, reflect.StructField{}) {
			return
		}

		// Handle the case of having an annotated struct - in the diff case we
		// do not process schema annotations.
		if util.IsYgotAnnotation(ni.StructField) {
			return
		}

		var sp [][]string
		if pathOpt != nil && pathOpt.PreferShadowPath {
			// Try the shadow-path tag first to see if it exists.
			sp = util.ShadowSchemaPaths(ni.StructField)
		}
		if len(sp) == 0 {
			var err error
			if sp, err = util.SchemaPaths(ni.StructField); err != nil {
				errs = util.AppendErr(errs, err)
				return
			}
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
			errs = util.NewErrs(err)
			return
		}

		// Avoid processing twice if there is duplicate path.
		keys := make([]string, len(vp.gNMIPaths))
		for i, paths := range vp.gNMIPaths {
			s, err := PathToString(paths)
			if err != nil {
				errs = util.NewErrs(err)
				return
			}
			keys[i] = s
		}
		sort.Strings(keys)
		key := strings.Join(keys, "/")
		if _, ok := processedPaths[key]; ok {
			return
		}
		processedPaths[key] = true

		ni.Annotation = []interface{}{vp}

		ival := ni.FieldValue.Interface()

		orderedMap, isOrderedMap := ival.(GoOrderedMap)

		// Ignore non-data, or default data values.
		if util.IsNilOrInvalidValue(ni.FieldValue) || util.IsValueNilOrDefault(ni.FieldValue.Interface()) || util.IsValueMap(ni.FieldValue) {
			return
		}
		// Ignore structs unless it is an ordered map and we're
		// treating it as a leaf (since it is assumed to be
		// telemetry-atomic in order to preserve ordering of entries).
		if (!isOrderedMap || !orderedMapAsLeaf) && util.IsValueStructPtr(ni.FieldValue) {
			return
		}
		if isOrderedMap && orderedMap.Len() == 0 {
			return
		}

		// If this is an enumerated value in the output structs, then check whether
		// it is set. Only include values that are set to a non-zero value.
		if _, isEnum := ival.(GoEnum); isEnum {
			val := ni.FieldValue
			// If the value is a simple union enum, then extract
			// the underlying enum value from the interface.
			if val.Kind() == reflect.Interface {
				val = val.Elem()
			}
			if val.Int() == 0 {
				return
			}
		}

		outs := out.(map[*pathSpec]interface{})
		outs[vp] = ival

		if isOrderedMap && orderedMapAsLeaf {
			// We treat the ordered map as a leaf, so don't
			// traverse any descendant elements.
			action = util.DoNotIterateDescendants
		}

		return
	}

	out := map[*pathSpec]interface{}{}
	if errs := util.ForEachDataField2(s, nil, out, findSetIterFunc); errs != nil {
		return nil, fmt.Errorf("error from ForEachDataField iteration: %v", errs)
	}

	return out, nil
}

// hasDiffPathOpt extracts a DiffPathOpt from the opts slice provided. In
// the case that there are multiple DiffPathOpt structs within opts slice, the
// first is returned.
func hasDiffPathOpt(opts []DiffOpt) *DiffPathOpt {
	for _, o := range opts {
		switch v := o.(type) {
		case *DiffPathOpt:
			return v
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
// to the path and value supplied. path is the string version of the path in pathInfo.
func appendUpdate(n *gnmipb.Notification, path string, pathInfo *pathInfo) error {
	v, err := EncodeTypedValue(pathInfo.val, gnmipb.Encoding_PROTO)
	if err != nil {
		return fmt.Errorf("cannot represent field value %v as TypedValue for path %v: %v", pathInfo.val, path, err)
	}
	n.Update = append(n.Update, &gnmipb.Update{
		Path: pathInfo.path,
		Val:  v,
	})
	return nil
}

// DiffOpt is an interface that is implemented by the options to the Diff
// function. It allows user specified options to be propagated to the diff
// method.
type DiffOpt interface {
	// IsDiffOpt is a marker method for each DiffOpt.
	IsDiffOpt()
}

// IgnoreAdditions is a DiffOpt that indicates newly-added fields should be
// ignored. The returned Notification will only contain the updates and
// deletions from original to modified.
type IgnoreAdditions struct{}

func (*IgnoreAdditions) IsDiffOpt() {}

// hasIgnoreAdditions returns the first IgnoreAdditions from an opts slice, or
// nil if there isn't one.
func hasIgnoreAdditions(opts []DiffOpt) *IgnoreAdditions {
	for _, o := range opts {
		switch v := o.(type) {
		case *IgnoreAdditions:
			return v
		}
	}
	return nil
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
	// PreferShadowPath specifies whether the "shadow-path" struct tag
	// annotation should be used instead of the "path" struct tag when it
	// exists.
	//
	// This option is used when GoStructs are generated with the
	// -ignore_shadow_schema_paths flag, and therefore have the
	// "shadow-path" tag.
	PreferShadowPath bool
	// Explicitly delete and then update leaf-list, which is different and exists in both orig and modified.
	// For example, if the original path is x/y/[a, b] and the modified leaf-list is x/y/[b, c],
	// Then resulting notification will contain:
	//   - Delete x/y/[a, b]
	//   - Update x/y/[b, c]
	// If this is false, then the leaf-list will be updated directly to the new value
	// without the delete step.
	// This is required because if the leaf-list is updated directly,
	// the resulting notification will be:
	//   - Update x/y/[b, c]
	// This will cause the new value to be appended in the OC, but the original value
	// will still be present in the OC. Resultant OC in the device will be x/y/[a, b, c].
	DeleteWithUpdateLeafList bool
}

// IsDiffOpt marks DiffPathOpt as a diff option.
func (*DiffPathOpt) IsDiffOpt() {}

// Diff takes an original and modified GoStruct, which must be of the same type
// and returns a gNMI Notification that contains the diff between them. The original
// struct is considered as the "from" data, with the modified struct the "to" such that:
//
//   - The contents of the Update field of the notification indicate that the
//     field in modified was either not present in original, or had a different
//     field value.
//   - The paths within the Delete field of the notification indicate that the
//     field was not present in the modified struct, but was set in the original.
//   - NOTE: For `ordered-by user` nodes, which are represented by ordered maps
//     in the ygot-generated code, the output Notification cannot be directly
//     unmarshalling into original to arrive at modified since updates are
//     granular. For generating atomic:true Notifications, use
//     ygot.DiffWithAtomic instead.
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
	notifs, err := diff(original, modified, false, opts...)
	switch len(notifs) {
	case 0:
		return &gnmipb.Notification{}, err
	case 1:
		return notifs[0], err
	default:
		return nil, fmt.Errorf("internal error: Diff expected a single Notification but got multiple")
	}
}

// FormatDiff formats the output of ygot.Diff as a multiline string. This
// function is only intended for human consumption and ignores errors. Do not
// depend on the output being stable. It may change over time across different
// versions of the program.
func FormatDiff(n *gnmipb.Notification) string {
	if n == nil {
		return "<nil> Notification"
	}
	var build strings.Builder
	for _, d := range n.Delete {
		path, err := PathToString(d)
		if err != nil {
			path = prototext.Format(d)
		}
		if build.Len() != 0 {
			build.WriteRune('\n')
		}
		build.WriteString(fmt.Sprintf("deleted: %s", path))
	}
	for _, u := range n.Update {
		path, err := PathToString(u.GetPath())
		if err != nil {
			path = prototext.Format(u.GetPath())
		}
		if build.Len() != 0 {
			build.WriteRune('\n')
		}
		build.WriteString(fmt.Sprintf("new/updated %s: %v", path, u.GetVal()))
	}
	if build.Len() == 0 {
		return "no diff"
	}
	return build.String()
}

// DiffWithAtomic takes an original and modified GoStruct, which must be of the same
// type and returns a slice of gNMI Notifications that represents the diff
// between them in a way that can be used to update a local gNMI.Subscribe
// server with the Notifications to be served. When
// ytypes.UnmarshalNotifications is called, it can also provide the same
// GoStruct back.
//
// The last message in the slice (if it exists) contains a non-atomic
// Notification representing updates/deletes to non-atomic nodes and deletes to
// atomic nodes. All messages prior to the last one are atomic Notifications
// representing updates to atomic nodes.
//
// NOTE: Currently only YANG `ordered-by user lists` are supported as atomic
// nodes. Further, they're always treated as so since there is no way of
// representing order using TypedValue scalar types.
//
// The original struct is considered as the "from" data, with the
// modified struct the "to" such that:
//
//   - The contents of the Update field of the notification indicate that the
//     field in modified was either not present in original, or had a different
//     field value.
//   - The paths within the Delete field of the notification indicate that the
//     field was not present in the modified struct, but was set in the original.
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
func DiffWithAtomic(original, modified GoStruct, opts ...DiffOpt) ([]*gnmipb.Notification, error) {
	return diff(original, modified, true, opts...)
}

// orderedMapLeaves returns an ordered list of path-value pairs representing
// the leaves belonging to the input ordered map.
//
//   - parent is the gNMI path representing the absolute path to the ordered
//     list. This must not be empty.
//   - The second return value is the prefix path that must be used in the
//     atomic Notification representing the ordered list.
func orderedMapLeaves(orderedMap GoOrderedMap, parent *gnmiPath, preferShadowPath bool) ([]*pathval, *gnmiPath, error) {
	var errs errlist.List
	var atomicLeaves []*pathval

	if err := yreflect.RangeOrderedMap(orderedMap, func(k reflect.Value, v reflect.Value) bool {
		childPath, err := mapValuePath(k, v, parent)
		if err != nil {
			errs.Add(err)
			return true
		}

		goStruct, ok := v.Interface().(GoStruct)
		if !ok {
			errs.Add(fmt.Errorf("%v: was not a valid GoStruct", parent))
			return true
		}
		errs.Add(findUpdatedLeaves(&atomicLeaves, goStruct, childPath, preferShadowPath))
		return true
	}); err != nil {
		errs.Add(err)
		return nil, nil, errs.Err()
	}

	// TODO(wenbli): Make this more robust by potentially introducing another struct
	// tag to indicate which element in the compressed path should the prefix cutoff
	// be based on the placement of the atomic extension, although for ordered-maps
	// it should always be at the container level. Need more discussion on this.
	// The current use case is for BGP policy statements:
	// https://github.com/openconfig/public/pull/867
	// The reason for this subtreePath hack is that the atomic annotation
	// is at the container surrounding the ordered lists.
	subtreePath := parent.Copy()
	if err := subtreePath.Pop(); err != nil {
		errs.Add(err)
	}

	return atomicLeaves, subtreePath, errs.Err()
}

// orderedMapNotif returns an atomic Notification for the given ordered map.
//
//   - If empty, then nil is returned.
//   - parent is the gNMI path representing the absolute path to the ordered
//     list. This must not be empty.
func orderedMapNotif(orderedMap GoOrderedMap, parent *gnmiPath, ts int64, preferShadowPath bool) (*gnmipb.Notification, error) {
	atomicLeaves, subtreePath, err := orderedMapLeaves(orderedMap, parent, preferShadowPath)
	if err != nil {
		return nil, err
	}
	if len(atomicLeaves) == 0 {
		return nil, nil
	}

	return createAtomicNotif(atomicLeaves, ts, subtreePath)
}

func createAtomicNotif(atomicLeaves []*pathval, ts int64, subtreePfx *gnmiPath) (*gnmipb.Notification, error) {
	no := &gnmipb.Notification{
		Timestamp: ts,
		Atomic:    true,
	}
	p, err := subtreePfx.ToProto()
	if err != nil {
		return nil, err
	}
	no.Prefix = p

	for _, pv := range atomicLeaves {
		if err := addToNotification(pv.path, pv.val, no, subtreePfx); err != nil {
			return nil, err
		}
	}
	return no, nil
}

// isLeafList returns true if the value is a slice (leaf-list).
func isLeafList(val any) bool {
	if val == nil {
		return false
	}
	return reflect.TypeOf(val).Kind() == reflect.Slice
}

// diff produces a slice of notifications given two GoStructs.
//
// See documentation for Diff and DiffWithAtomic for more information.
//
//   - withAtomic indicates that atomic notifications should be generated
//     (currently this is only supported for `ordered-by user` lists)
func diff(original, modified GoStruct, withAtomic bool, opts ...DiffOpt) ([]*gnmipb.Notification, error) {
	if reflect.TypeOf(original) != reflect.TypeOf(modified) {
		return nil, fmt.Errorf("cannot diff structs of different types, original: %T, modified: %T", original, modified)
	}

	origLeaves, err := findSetLeaves(original, withAtomic, opts...)
	if err != nil {
		return nil, fmt.Errorf("could not extract set leaves from original struct: %v", err)
	}

	modLeaves, err := findSetLeaves(modified, withAtomic, opts...)
	if err != nil {
		return nil, fmt.Errorf("could not extract set leaves from modified struct: %v", err)
	}

	origLeavesStr, err := toStringPathMap(origLeaves)
	if err != nil {
		return nil, fmt.Errorf("could not convert leaf path map to string path map: %v", err)
	}
	modLeavesStr, err := toStringPathMap(modLeaves)
	if err != nil {
		return nil, fmt.Errorf("could not convert leaf path map to string path map: %v", err)
	}

	var atomicNotifs []*gnmipb.Notification
	n := &gnmipb.Notification{}
	processUpdate := func(path string, modVal *pathInfo, origVal *pathInfo) error {
		diffopts := hasDiffPathOpt(opts)
		if orderedMap, isOrderedMap := modVal.val.(GoOrderedMap); isOrderedMap {
			preferShadowPath := diffopts != nil && diffopts.PreferShadowPath
			notif, err := orderedMapNotif(orderedMap, newPathElemGNMIPath(modVal.path.GetElem()), 0, preferShadowPath)
			if err != nil {
				return err
			}
			atomicNotifs = append(atomicNotifs, notif)
		} else if diffopts != nil && diffopts.DeleteWithUpdateLeafList && isLeafList(modVal.val) && origVal != nil {
			// Handle the leaf-list replace.
			// Delete the original path before updating it.
			n.Delete = append(n.Delete, origVal.path)
			if err := appendUpdate(n, path, modVal); err != nil {
				return err
			}
		} else {
			// The contents of the value should indicate that value a has changed
			// to value b.
			if err := appendUpdate(n, path, modVal); err != nil {
				return err
			}
		}
		return nil
	}

	for origPath, origVal := range origLeavesStr {
		if modVal, ok := modLeavesStr[origPath]; ok {
			if !reflect.DeepEqual(origVal.val, modVal.val) {
				if err := processUpdate(origPath, modVal, origVal); err != nil {
					return nil, err
				}
			}
		} else if !ok {
			if orderedMap, isOrderedMap := origVal.val.(GoOrderedMap); isOrderedMap {
				pathLen := len(origVal.path.Elem)
				if pathLen == 0 {
					return nil, fmt.Errorf("deletion path on ordered list is empty, this is unexpected: %T", orderedMap)
				}
				origVal.path.Elem = origVal.path.Elem[:pathLen-1]
			}
			// This leaf was set in the original struct, but not in the modified
			// struct, therefore it has been deleted.
			n.Delete = append(n.Delete, origVal.path)
		}
	}

	if hasIgnoreAdditions(opts) == nil {
		// Check that all paths that are in the modified struct have been examined, if
		// not they are updates.
		for modPath, modVal := range modLeavesStr {
			if _, ok := origLeavesStr[modPath]; !ok {
				if err := processUpdate(modPath, modVal, nil); err != nil {
					return nil, err
				}
			}
		}
	}

	if len(n.Delete)+len(n.Update) == 0 {
		return atomicNotifs, nil
	}

	return append([]*gnmipb.Notification{n}, atomicNotifs...), nil
}
