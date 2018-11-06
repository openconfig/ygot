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
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	log "github.com/golang/glog"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/util"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
)

// ValidateLeafRefData traverses the entire tree with root value and the given
// corresponding schema. For the referring node A, the leafref will point to a
// value set B which may be empty. For each element in B:
// - if the element is a leaf, it checks whether A == B
// - if the element is a leaf list C, it check whether A is equal to
//   any of the elements of C.
// It returns nil if at least one equality check passes or an error otherwise.
// It also returns an error if any leafref points to a value outside of the tree
// rooted at value; therefore it should only be called on the root node of the
// entire data tree. The supplied LeafrefOptions specify particular behaviours
// of the leafref validation such as ignoring missing pointed to elements.
func ValidateLeafRefData(schema *yang.Entry, value interface{}, opt *LeafrefOptions) util.Errors {
	// If the IgnoreMissingData flag is set, then we do not need to iterate through nodes,
	// so immediately return no error.
	if opt != nil && opt.IgnoreMissingData {
		return nil
	}

	// validateLefRefDataIterFunc is called on every node in the tree through
	// ForEachField below.
	validateLefRefDataIterFunc := func(ni *util.NodeInfo, in, out interface{}) util.Errors {
		if util.IsValueNil(ni) || util.IsNilOrInvalidValue(ni.FieldValue) {
			return nil
		}
		schema := ni.Schema
		if schema == nil {
			return util.NewErrs(fmt.Errorf("schema is nil for value %s, type %T", util.ValueStr(value), value))
		}
		if !util.IsLeafRef(schema) || schema.IsLeafList() {
			return nil
		}

		gNMIPath, err := leafRefToGNMIPath(ni, schema.Type.Path)
		if err != nil {
			return util.NewErrs(err)
		}
		matchNodes, err := dataNodesAtPath(ni, gNMIPath)
		if err != nil {
			return util.NewErrs(err)
		}

		pathStr := util.StripModulePrefixesStr(schema.Type.Path)
		util.DbgPrint("Verifying leafref at %s, matching nodes are: %v", pathStr, util.ValueStrDebug(matchNodes))

		match, err := matchesNodes(ni, matchNodes)
		if err != nil {
			return leafrefErrOrLog(util.NewErrs(err), opt)
		}
		if !match {
			e := fmt.Errorf("field name %s value %s schema path %s has leafref path %s not equal to any target nodes",
				ni.StructField.Name, util.ValueStr(ni.FieldValue.Interface()), ni.Schema.Path(), pathStr)
			util.DbgPrint("ERR: %s", e)
			return leafrefErrOrLog(util.NewErrs(e), opt)
		}

		return nil
	}

	return util.ForEachField(schema, value, nil, nil, validateLefRefDataIterFunc)
}

// leafrefErrOrLog returns an error if the global ValidationOptions specifies
// that missing data should cause an error to be thrown. If the missing data is to
// be ignored by leafrefs, it logs the error that would have been returned if the
// Log field of the LeafrefOptions is set to true.
func leafrefErrOrLog(e util.Errors, opt *LeafrefOptions) util.Errors {
	if opt == nil {
		return e
	}

	if opt.Log {
		log.Errorf("%v", e)
	}

	return nil
}

// leafRefToGNMIPath takes a leafref path string and transforms any leafref
// path references of the form a[k1 = ../path/to/val and k2 = ...] to a GNMI
// path where the key values are the values being referenced i.e.
// ../path/to/val above is replaced with the actual value at that path.
func leafRefToGNMIPath(root *util.NodeInfo, path string) (*gpb.Path, error) {
	pv := splitPath(path)
	out := &gpb.Path{}

	for _, p := range pv {
		prefix, k, v, err := extractKeyValue(p)
		if err != nil {
			return nil, err
		}
		gp := &gpb.PathElem{Name: prefix}

		switch {
		case k == "":
		// No kvs, path element is just the prefix.
		case isInQuotes(v):
			// Value should be treated as a literal, just strip off the quotes.
			gp.Key = map[string]string{k: v[1 : len(v)-1]}
		default:
			// The value is a path, need to replace it with the actual val at
			// the indicated path.
			// current() can only mean the unique current node in the subtree
			// branch of the node containing the leafref. It can be removed
			// since it is implicit.
			v = strings.TrimPrefix(v, "current()/")

			gp.Key = make(map[string]string)
			ns, err := dataNodesAtPath(root, pathNoKeysToGNMIPath(v))
			var j []byte
			switch len(ns) {
			case 0:
			case 1:
				actualValue := ns[0]
				if err != nil {
					return nil, err
				}
				j, err = json.Marshal(actualValue)
				if err != nil {
					return nil, err
				}
			default:
				return nil, fmt.Errorf("expect single node to match value at path %s, got %d", v, len(ns))
			}

			gp.Key = map[string]string{k: removeQuotes(string(j))}
		}
		out.Elem = append(out.Elem, gp)
	}

	return out, nil
}

// dataNodesAtPath returns all nodes that match the given path from the given
// node.
func dataNodesAtPath(ni *util.NodeInfo, path *gpb.Path) ([]interface{}, error) {
	util.DbgPrint("DataNodeAtPath got leafref with path %s from node path %s, field name %s", path, ni.Schema.Path(), ni.StructField.Name)
	if path == nil || len(path.GetElem()) == 0 {
		return []interface{}{ni}, nil
	}
	root := getDataTreeRoot(ni)
	if path.GetElem()[0].GetName() == "" {
		// absolute path
		path.Elem = path.GetElem()[1:]
	} else {
		// relative path, go up the data tree
		root = ni
		for len(path.GetElem()) != 0 && path.GetElem()[0].GetName() == ".." {
			if root.Parent == nil {
				return nil, fmt.Errorf("no parent for leafref path at %v, with remaining path %s", ni.Schema.Path(), path)
			}
			if !util.IsCompressedSchema(root.Schema) && root.Schema.IsList() && util.IsValueMap(root.FieldValue) {
				// If we are in an uncompressed schema, then we have one more level of the data tree than
				// the YANG expects, since our data tree layout is:
				// struct (parent container)
				//  --> map (the list)
				//  --> struct (the list member)
				//
				// In YANG, .. from the list member struct gets us to the parent container, but for us
				// we have only reached the map. This means that we end up over-consuming the ".."s. To
				// avoid this issue, if we are in an uncompressed schema, and in a list, and we find
				// that we're looking at the map, we consume another level of the data tree. This gets
				// us to the parent container with ".." as would be expected.
				//
				// This is NOT required for the compressed schema, because in this case, we have removed
				// a level of the data tree. So the parent container in the above example will have been
				// removed. This is enforced by ygen. In this case, we do want to consume the extra level
				// of ..s in the list case, such that we do not end up under-consuming them.
				root = root.Parent
				continue
			} else {
				path.Elem = removeParentDirPrefix(path.GetElem(), root.PathFromParent)
				util.DbgPrint("going up data tree from type %s to %s, schema path from parent is %v, remaining path %v",
					root.FieldValue.Type(), root.Parent.FieldValue.Type(), root.PathFromParent, path)
				root = root.Parent
			}
		}
	}

	util.DbgPrint("root element type %s with remaining path %s", root.FieldValue.Type(), path)
	nodes, _, err := util.GetNodes(root.Schema, root.FieldValue.Interface(), path)
	return nodes, err
}

// removeParentDirPrefix removes the leading .. from path and returns the
// remaining path from the parent node, restoring any compressed out path
// elements along the way.
func removeParentDirPrefix(path []*gpb.PathElem, pathFromParent []string) []*gpb.PathElem {
	plen := len(pathFromParent)
	out := path
	for len(out) > 0 && out[0].GetName() == ".." && plen > 0 {
		out = out[1:]
		plen--
	}
	// If we are inside a compressed node, restore the compressed out part
	// of the path when we go up to the parent.
	for i := 0; i < plen; i++ {
		out = append([]*gpb.PathElem{{Name: pathFromParent[i]}}, out...)
	}
	return out
}

// matchesNodes reports whether ni matches any of the elements in matchNodes.
// matchNodes may contain one or more leaf-lists, in which case ni is compared
// against each value in the leaf-list.
func matchesNodes(ni *util.NodeInfo, matchNodes []interface{}) (bool, error) {
	// Handle source or destination being empty.
	pathStr := util.StripModulePrefixesStr(ni.Schema.Type.Path)
	if util.IsNilOrInvalidValue(ni.FieldValue) || util.IsValueNilOrDefault(ni.FieldValue.Interface()) {
		if len(matchNodes) == 0 {
			util.DbgPrint("OK: source value is nil, dest is empty or list")
			return true, nil
		}
		other := matchNodes[0]
		if util.IsValueNilOrDefault(other) {
			util.DbgPrint("OK: both values are nil for leafref")
			return true, nil
		}
		return true, nil
	}
	// ni is known not to be empty at this point.
	nii := ni.FieldValue.Interface()
	if len(matchNodes) == 0 {
		return false, util.NewErrs(util.DbgErr(fmt.Errorf("pointed-to value with path %s from field %s value %s schema %s is empty set",
			pathStr, ni.StructField.Name, util.ValueStr(nii), ni.Schema.Path())))
	}

	// Check if any of the matching data nodes is equal to the referring
	// node value. In the case that the referring node is a list, check that
	// each node in the list is also in the target list.
	sourceNodes := []interface{}{nii}
	if ni.FieldValue.Type().Kind() == reflect.Slice {
		sourceNodes = ni.FieldValue.Elem().Interface().([]interface{})
	}

	for _, sourceNode := range sourceNodes {
		match := false
		for _, other := range matchNodes {
			if util.IsValueNilOrDefault(other) {
				continue
			}
			ov := reflect.ValueOf(other)
			switch {
			case util.IsValueScalar(ov):
				util.DbgPrint("comparing leafref values %s vs %s", util.ValueStrDebug(sourceNode), util.ValueStrDebug(other))
				if util.DeepEqualDerefPtrs(sourceNode, other) {
					util.DbgPrint("values are equal")
					match = true
					break
				}
			case util.IsValueSlice(ov):
				sourceNode := ni.FieldValue.Interface()
				util.DbgPrint("checking whether value %s is leafref leaf-list %v", util.ValueStrDebug(sourceNode), util.ValueStrDebug(other))
				for i := 0; i < ov.Len(); i++ {
					if util.DeepEqualDerefPtrs(sourceNode, ov.Index(i).Interface()) {
						util.DbgPrint("value exists in list")
						match = true
						break
					}
				}
			case util.IsValueStructPtr(ov):
				// TODO(robjs): clean this up.
				// This is an interface value, which is represented as a struct pointer.
				ovv := ov.Elem().FieldByIndex([]int{0})
				svv := ni.FieldValue.Elem().Elem().FieldByIndex([]int{0})
				if reflect.DeepEqual(ovv.Interface(), svv.Interface()) {
					match = true
					break
				}
			}
		}
		if !match {
			return false, nil
		}
	}

	return true, nil
}

// getDataTreeRoot returns the root NodeInfo element for the current node.
func getDataTreeRoot(ni *util.NodeInfo) *util.NodeInfo {
	if ni == nil {
		return nil
	}
	cur := ni.Parent
	for cur.Parent != nil {
		cur = cur.Parent
	}
	return cur
}

// extractKeyValue parses a string containing a key-value of the form:
// prefix[key1 = current()/path] or prefix[key1 = "literal value"]
// It returns the prefix, key and value from the string, if they are present
// or empty strings otherwise.
func extractKeyValue(p string) (prefix string, k, v string, err error) {
	if p == "" {
		return "", "", "", nil
	}
	isKV, err := isKeyValue(p)
	if err != nil {
		return
	}
	if !isKV {
		return util.StripModulePrefix(p), "", "", nil
	}

	p1 := splitUnescapedUnquoted(p, '[')
	p2 := splitUnescapedUnquoted(p1[1], ']')
	kv := splitUnescapedUnquoted(p2[0], '=')
	if len(kv) != 2 {
		return "", "", "", fmt.Errorf("bad kv string %s", kv)
	}
	k = strings.TrimSpace(kv[0])
	v = strings.TrimSpace(kv[1])

	if !strings.HasPrefix(v, "current()/") && !isInQuotes(v) {
		return "", "", "", fmt.Errorf("bad kv string %s: value must be in quotes or begin with current()/", p2[0])
	}

	return util.StripModulePrefix(p1[0]), util.StripModulePrefix(k), util.StripModulePrefix(v), nil
}

// isKeyValue reports whether p contains a valid key-value leafref path element.
func isKeyValue(p string) (bool, error) {
	p1 := splitUnescapedUnquoted(p, '[')
	l1 := len(p1)
	pv := splitUnescapedUnquoted(p, ']')
	l2 := len(pv)

	switch {
	case l1 == 0 || l2 == 0:
		return false, fmt.Errorf("empty path element (%s)", p)
	case l1 == 1 && l2 == 1:
		return false, nil
	case l1 > 2 || l2 > 2 || l1 == 1 && l2 > 1 || l2 == 1 && l1 > 1:
		return false, fmt.Errorf("malformed path element %s ", p1)
	case pv[1] != "":
		return false, fmt.Errorf("trailing chars after [...]: %s", p)
	}

	return true, nil
}

// splitPath splits path across unescaped /.
// Any / inside square brackets are ignored.
func splitPath(path string) []string {
	var prev rune
	var out []string
	var w bytes.Buffer
	skip := false

	for _, r := range path {
		switch {
		case r == '[' && prev != '\\':
			skip = true

		case r == ']' && prev != '\\':
			skip = false
		}

		if !skip && r == '/' && prev != '\\' {
			out = append(out, w.String())
			w.Reset()
		} else {
			w.WriteRune(r)
		}
		prev = r
	}

	if w.Len() != 0 {
		out = append(out, w.String())
	}
	if prev == '/' {
		out = append(out, "")
	}

	return out
}

// splitUnescaped splits source across splitCh. If splitCh is immedaitely
// preceded by \ it is skipped.
func splitUnescaped(source string, splitCh rune) []string {
	var prev rune
	var out []string
	var w bytes.Buffer
	doCompare := true

	for _, r := range source {
		if doCompare && r == splitCh && prev != '\\' {
			out = append(out, w.String())
			w.Reset()
		} else {
			w.WriteRune(r)
		}
		prev = r
	}

	if w.Len() != 0 {
		out = append(out, w.String())
	}
	// If split on last char, add trailing empty value.
	if prev == splitCh {
		out = append(out, "")
	}

	return out
}

// splitUnescapedUnquoted splits source across splitCh. If splitCh is
// immediately preceded by \ or inside unescaped quotes, it is skipped.
func splitUnescapedUnquoted(source string, splitCh rune) []string {
	var prev rune
	var out []string
	var w bytes.Buffer
	doCompare := true

	for _, r := range source {
		if r == '"' && prev != '\\' {
			doCompare = !doCompare
		}
		if doCompare && r == splitCh && prev != '\\' {
			out = append(out, w.String())
			w.Reset()
		} else {
			w.WriteRune(r)
		}
		prev = r
	}

	if w.Len() != 0 {
		out = append(out, w.String())
	}
	// If split on last char, add trailing empty value.
	if prev == splitCh {
		out = append(out, "")
	}

	return out
}

// splitUnquoted splits source source across splitStr. Any instance of splitStr
// inside quotes is ignored.
func splitUnquoted(source, splitStr string) []string {
	var prev rune
	var out []string
	var w bytes.Buffer
	doCompare := true

	for i, r := range source {
		// don't compare anything inside unquoted ".
		if r == '"' && prev != '\\' {
			doCompare = !doCompare
		}

		if doCompare && prev != '\\' && strings.HasPrefix(source[i:], splitStr) {
			// splitStr is included in all elements except the first, trim it.
			out = append(out, strings.TrimPrefix(w.String(), splitStr))
			w.Reset()
		} else {
			w.WriteRune(r)
		}
		prev = r
	}

	if w.Len() != 0 {
		out = append(out, w.String())
	}
	if strings.HasSuffix(source, splitStr) {
		out = append(out, "")
	}

	return out
}

// isInQuotes reports whether s starts and ends with the quote character.
func isInQuotes(s string) bool {
	return strings.HasPrefix(s, "\"") && strings.HasSuffix(s, "\"")
}

// removeQuotes removes quotes around s if they are present.
func removeQuotes(s string) string {
	out := strings.TrimPrefix(s, "\"")
	return strings.TrimSuffix(out, "\"")
}

// pathNoKeysToGNMIPath conerts the supplied path, which may not contain any
// keys, into a GNMI path.
func pathNoKeysToGNMIPath(path string) *gpb.Path {
	out := &gpb.Path{}
	for _, p := range strings.Split(path, "/") {
		out.Elem = append(out.Elem, &gpb.PathElem{Name: p})
	}
	return out
}

// pathMatchesPrefix reports whether prefix is a prefix of path.
func pathMatchesPrefix(path []string, prefix []string) bool {
	if len(path) < len(prefix) {
		return false
	}
	for i := range prefix {
		if prefix[i] != path[i] {
			return false
		}
	}

	return true
}
