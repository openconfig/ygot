// Copyright 2023 Google Inc.
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

// gnmidiff contains gNMI utilities for diffing SetRequests and GetResponses.
package gnmidiff

import (
	"encoding/base64"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/derekparker/trie"
	"github.com/openconfig/ygot/util"
	"github.com/openconfig/ygot/ygot"
	"github.com/openconfig/ygot/ytypes"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
)

// MismatchedUpdate represents two different update values for the same leaf
// node.
type MismatchedUpdate struct {
	// A is the update value in A.
	A interface{}
	// B is the update value in B.
	B interface{}
}

// SetRequestIntentDiff contains the intent difference between two SetRequests.
//
// - The key of the maps is the string representation of a gpb.Path constructed
// by ygot.PathToString.
// - The value of the update fields is the JSON_IETF representation of the
// value.
type SetRequestIntentDiff struct {
	AOnlyDeletes      map[string]struct{}
	BOnlyDeletes      map[string]struct{}
	CommonDeletes     map[string]struct{}
	AOnlyUpdates      map[string]interface{}
	BOnlyUpdates      map[string]interface{}
	CommonUpdates     map[string]interface{}
	MismatchedUpdates map[string]MismatchedUpdate
}

// Format is the string format of any gNMI diff utility in this package.
type Format struct {
	// Full indicates that common values are also output.
	Full bool
	// TODO: Implement IncludeList and ExcludeList.
	// IncludeList is a list of paths that will be included in the output.
	// wildcards are allowed.
	//
	// empty implies all paths are included.
	//IncludeList []string
	// ExcludeList is a list of paths that will be excluded from the output.
	// wildcards are allowed.
	//
	// empty implies no paths are excluded.
	//ExcludeList []string
}

func formatJSONValue(value interface{}) interface{} {
	if v, ok := value.(string); ok {
		return strconv.Quote(v)
	}
	return value
}

// Format outputs the SetRequestIntentDiff in human-readable format.
//
// NOTE: Do not depend on the output of this being stable.
func (diff SetRequestIntentDiff) Format(f Format) string {
	var b strings.Builder
	writeDeletes := func(deletePaths map[string]struct{}, symbol rune) {
		var paths []string
		for path := range deletePaths {
			paths = append(paths, path)

		}
		sort.Strings(paths)
		for _, path := range paths {
			b.WriteString(fmt.Sprintf("%c %s: deleted\n", symbol, path))
		}
	}

	writeUpdates := func(updates map[string]interface{}, symbol rune) {
		var paths []string
		for path := range updates {
			paths = append(paths, path)

		}
		sort.Strings(paths)
		for _, path := range paths {
			b.WriteString(fmt.Sprintf("%c %s: %v\n", symbol, path, formatJSONValue(updates[path])))
		}
	}

	b.WriteString("SetRequestIntentDiff(-A, +B):\n")
	b.WriteString("-------- deletes --------\n")
	if f.Full {
		writeDeletes(diff.CommonDeletes, ' ')
	}
	writeDeletes(diff.AOnlyDeletes, '-')
	writeDeletes(diff.BOnlyDeletes, '+')
	b.WriteString("-------- updates --------\n")
	if f.Full {
		writeUpdates(diff.CommonUpdates, ' ')
	}
	writeUpdates(diff.AOnlyUpdates, '-')
	writeUpdates(diff.BOnlyUpdates, '+')
	var paths []string
	for path := range diff.MismatchedUpdates {
		paths = append(paths, path)

	}
	sort.Strings(paths)
	for _, path := range paths {
		mismatch := diff.MismatchedUpdates[path]
		b.WriteString(fmt.Sprintf("m %s:\n  - %v\n  + %v\n", path, formatJSONValue(mismatch.A), formatJSONValue(mismatch.B)))
	}
	return b.String()
}

// DiffSetRequest returns a unique and minimal intent diff of two SetRequests.
//
// schema is intended to be provided via the function defined in generated
// ygot code (e.g. exampleoc.Schema).
// If schema is nil, then DiffSetRequest will make the following assumption:
// - Any JSON value in the input SetRequest MUST conform to the OpenConfig
// YANG style guidelines. See the following for checking compliance.
// * https://github.com/openconfig/oc-pyang
// * https://github.com/openconfig/public/blob/master/doc/openconfig_style_guide.md
//
// Currently, support is only for SetRequests without any delete paths, and
// replace and updates that don't have conflicting leaf values. If not
// supported, then an error will be returned.
func DiffSetRequest(a *gpb.SetRequest, b *gpb.SetRequest, schema *ytypes.Schema) (SetRequestIntentDiff, error) {
	intentA, err := minimalSetRequestIntent(a, schema)
	if err != nil {
		return SetRequestIntentDiff{}, fmt.Errorf("DiffSetRequest on a: %v", err)
	}
	intentB, err := minimalSetRequestIntent(b, schema)
	if err != nil {
		return SetRequestIntentDiff{}, fmt.Errorf("DiffSetRequest on b: %v", err)
	}
	diff := SetRequestIntentDiff{
		AOnlyDeletes:      map[string]struct{}{},
		BOnlyDeletes:      map[string]struct{}{},
		CommonDeletes:     map[string]struct{}{},
		AOnlyUpdates:      map[string]interface{}{},
		BOnlyUpdates:      map[string]interface{}{},
		CommonUpdates:     map[string]interface{}{},
		MismatchedUpdates: map[string]MismatchedUpdate{},
	}
	for path := range intentA.Deletes {
		if _, ok := intentB.Deletes[path]; ok {
			delete(intentA.Deletes, path)
			delete(intentB.Deletes, path)
			diff.CommonDeletes[path] = struct{}{}
		}
	}
	diff.AOnlyDeletes = intentA.Deletes
	diff.BOnlyDeletes = intentB.Deletes
	for path, vA := range intentA.Updates {
		vB, ok := intentB.Updates[path]
		if !ok {
			continue
		}
		delete(intentA.Updates, path)
		delete(intentB.Updates, path)
		if !reflect.DeepEqual(vA, vB) { // leaf-lists cannot be compared directly.
			diff.MismatchedUpdates[path] = MismatchedUpdate{A: vA, B: vB}
		} else {
			diff.CommonUpdates[path] = vA
		}
	}
	diff.AOnlyUpdates = intentA.Updates
	diff.BOnlyUpdates = intentB.Updates
	return diff, nil
}

type setRequestIntent struct {
	// Deletes are deletions to any node.
	Deletes map[string]struct{}
	// Updates are leaf updates only.
	Updates map[string]interface{}
}

// minimalSetRequestIntent returns a unique and minimal intent for a SetRequest.
//
// TODO: Currently, support is only for SetRequests without any delete paths,
// and replace and updates that don't have conflicting leaf values. If not
// supported, then an error will be returned.
func minimalSetRequestIntent(req *gpb.SetRequest, schema *ytypes.Schema) (setRequestIntent, error) {
	// TODO: Handle prefix in SetRequest.
	if req == nil {
		req = &gpb.SetRequest{}
	}
	if len(req.Delete) > 0 {
		return setRequestIntent{}, fmt.Errorf("gnmidiff: delete paths are not supported.")
	}
	intent := setRequestIntent{
		Deletes: map[string]struct{}{},
		Updates: map[string]interface{}{},
	}
	// NOTE: This simple trie will not work if we intend to check conflicts with wildcard deletion paths.
	t := trie.New()
	for _, upd := range req.Replace {
		path, err := ygot.PathToString(upd.Path)
		if err != nil {
			return setRequestIntent{}, fmt.Errorf("gnmidiff: %v", err)
		}
		if _, ok := intent.Deletes[path]; ok {
			return setRequestIntent{}, fmt.Errorf("gnmidiff: conflicting replaces in SetRequest: %v", path)
		}
		intent.Deletes[path] = struct{}{}
		t.Add(path, nil)

		if err := populateUpdate(&intent, path, upd.GetVal(), schema); err != nil {
			return setRequestIntent{}, err
		}
	}

	// Do prefix match to check for conflicting replace paths.
	for _, path := range t.Keys() {
		if matches := t.PrefixSearch(path + "/"); len(matches) >= 1 {
			return setRequestIntent{}, fmt.Errorf("gnmidiff: conflicting replaces in SetRequest: %v, %v", path, matches)
		}
	}

	for _, upd := range req.Update {
		path, err := ygot.PathToString(upd.Path)
		if err != nil {
			return setRequestIntent{}, fmt.Errorf("gnmidiff: %v", err)
		}

		if err := populateUpdate(&intent, path, upd.GetVal(), schema); err != nil {
			return setRequestIntent{}, err
		}
	}

	t = trie.New()
	for path := range intent.Updates {
		t.Add(path, nil)
	}

	// Do prefix match to check for conflicting update paths.
	for _, path := range t.Keys() {
		if matches := t.PrefixSearch(path + "/"); len(matches) >= 1 {
			return setRequestIntent{}, fmt.Errorf("gnmidiff: bad SetRequest, there are leaf updates that have a prefix match: %v, %v", path, matches)
		}
	}

	return intent, nil
}

// binaryBase64 takes an input byte slice and returns it as a base64
// encoded string.
func binaryBase64(i []byte) string {
	return base64.StdEncoding.EncodeToString(i)
}

// protoLeafToJSON converts a gNMI proto scalar to the equivalent RFC7951 JSON
// representation.
func protoLeafToJSON(tv *gpb.TypedValue) (interface{}, error) {
	switch tv.GetValue().(type) {
	case *gpb.TypedValue_StringVal:
		return tv.GetStringVal(), nil
	case *gpb.TypedValue_BoolVal:
		return tv.GetBoolVal(), nil
	case *gpb.TypedValue_IntVal:
		return float64(tv.GetIntVal()), nil
	case *gpb.TypedValue_UintVal:
		return float64(tv.GetUintVal()), nil
	case *gpb.TypedValue_DoubleVal:
		return float64(tv.GetDoubleVal()), nil
	case *gpb.TypedValue_LeaflistVal:
		elems := tv.GetLeaflistVal().GetElement()
		ss := make([]interface{}, len(elems))
		for x, e := range elems {
			var err error
			if ss[x], err = protoLeafToJSON(e); err != nil {
				return nil, err
			}
		}
		return ss, nil
	case *gpb.TypedValue_BytesVal:
		return binaryBase64(tv.GetBytesVal()), nil
	default:
		return nil, fmt.Errorf("gnmidiff: TypedValue type %T is not a scalar type", tv.GetValue())
	}
}

// populateUpdate populates all leaf updates at the given path into the intent.
//
// - schema is intended to be provided via the function defined in generated
// ygot code (e.g. exampleoc.Schema).
// If schema is nil, then it is assumed any JSON value in the input
// SetRequest MUST conform to the OpenConfig YANG style guidelines. Otherwise
// it is impossible to infer the list keys of a leaf update.
//
// Note: The input path must NOT end with "/".
//
// e.g. /a for b/c="foo" would introduce an update of /a/b/c="foo" into the intent.
func populateUpdate(intent *setRequestIntent, path string, tv *gpb.TypedValue, schema *ytypes.Schema) error {
	// A function schema is used as input instead of just
	// ytypes.Schema in order to start with a clean root object each time
	// unmarshalling happens. Otherwise there needs to be some `reflect`
	// code that resets the root object each time to avoid previous
	// unmarshalling done in previous invocations of `populateUpdates`
	// polluting the results.
	if len(path) > 0 && path[len(path)-1] == '/' {
		return fmt.Errorf("gnmidiff: invalid input path %q, must not end with \"/\"", path)
	}

	// Populate updates when the schema is unknown.
	if schema == nil {
		return populateUpdateNoSchema(intent, path, tv)
	}

	gpath, err := ygot.StringToStructuredPath(path)
	if err != nil {
		return fmt.Errorf("gnmidiff: %v", err)
	}

	// The code below uses the schema to populate leaves specified in the
	// input TypedValue into the generated ygot-GoStruct, and then
	// marshals the flattened leaf updates using ygot.TogNMINotifications.
	//
	// The procedure for populating the leaves differs by whether the input
	// path is a leaf or a non-leaf.
	//
	// - For leaves, the procedure is straightforward: simply unmarshal and
	// then marshal.
	// - For non-leaves, the procedure is more complicated due to the
	// implicit creation of list keys when a leaf is unmarshalled into a
	// GoStruct: when the input path points to a list entry, but the input
	// TypedValue does not contain list keys, then ygot.TogNMINotifications
	// will marshal these list keys. Although this is technically not
	// different in intent, for consistency with the non-schema-aware
	// result, these list keys should not be part of the returned intent.
	//
	// To solve this problem, the unmarshal target is set to be the
	// non-leaf node itself, which avoids the implicit creation of these
	// list keys.
	if !schema.IsValid() {
		return fmt.Errorf("gnmidiff: input schema is not valid: %+v", schema)
	}
	rootSchema := schema.RootSchema()
	targetSchema, err := util.FindLeafRefSchema(rootSchema, path)
	if err != nil {
		return fmt.Errorf("gnmidiff: error finding target schema: %v", err)
	}
	setNodeTargetSchema := rootSchema
	// Create a new empty root since we don't want previous updates to
	// count as the current update.
	setNodeTarget, ok := reflect.New(reflect.TypeOf(schema.Root).Elem()).Interface().(ygot.GoStruct)
	if !ok {
		return fmt.Errorf("schema root is a non-GoStruct, this is not allowed: %T, %v", schema.Root, schema.Root)
	}
	setNodePath := &gpb.Path{}
	if targetSchema.IsLeaf() || targetSchema.IsLeafList() {
		// leaf replace is the same as a leaf update, so remove
		// the deletion action to keep the intent minimal.
		delete(intent.Deletes, path)
	} else {
		// For a non-leaf update, we want to call SetNode on
		// the non-leaf node itself to avoid creating key
		// leaves as a side-effect of unmarshalling a path that
		// ends at a list entry.
		setNodePath = gpath
		gpath = &gpb.Path{}

		nodeI, _, err := ytypes.GetOrCreateNode(rootSchema, schema.Root, setNodePath)
		if err != nil {
			return fmt.Errorf("failed to GetOrCreate the prefix node: %v", err)
		}
		targetType := reflect.TypeOf(nodeI).Elem()
		var ok bool
		if setNodeTarget, ok = reflect.New(targetType).Interface().(ygot.GoStruct); !ok {
			return fmt.Errorf("prefix path points to a non-GoStruct, this is not allowed: %T, %v", nodeI, nodeI)
		}
		setNodeTargetSchema = schema.SchemaTree[targetType.Name()]
	}

	if err := ytypes.SetNode(setNodeTargetSchema, setNodeTarget, gpath, tv, &ytypes.InitMissingElements{}); err != nil {
		return fmt.Errorf("gnmidiff: error unmarshalling update: %v", err)
	}

	// Marshal flattened leaf paths and convert to JSON types for populating the intent.
	notifs, err := ygot.TogNMINotifications(setNodeTarget, 0, ygot.GNMINotificationsConfig{UsePathElem: true})
	if err != nil {
		return fmt.Errorf("gnmidiff: error marshalling GoStruct: %v", err)
	}
	prefixPathStr := ""
	if len(setNodePath.Elem) > 0 {
		if prefixPathStr, err = ygot.PathToString(setNodePath); err != nil {
			return fmt.Errorf("gnmidiff: error creating prefix path string: %v", err)
		}
	}
	for _, n := range notifs {
		for _, upd := range n.Update {
			pathToLeaf, err := ygot.PathToString(upd.Path)
			if err != nil {
				return fmt.Errorf("gnmidiff: %v", err)
			}
			pathToLeaf = prefixPathStr + pathToLeaf
			val, err := protoLeafToJSON(upd.GetVal())
			if err != nil {
				return err
			}
			if !strings.HasPrefix(pathToLeaf, path) {
				// This was a list key that was created by SetNode, so drop it.
				continue
			}
			if prevVal, ok := intent.Updates[pathToLeaf]; ok && val != prevVal {
				return fmt.Errorf("gnmidiff: leaf value set twice with different values in SetRequest: %v", pathToLeaf)
			}
			intent.Updates[pathToLeaf] = val
		}
	}
	return nil
}

// populateUpdateNoSchema populates all leaf updates at the given path into the intent.
//
// e.g. /a for b/c="foo" would introduce an update of /a/b/c="foo" into the intent.
func populateUpdateNoSchema(intent *setRequestIntent, path string, tv *gpb.TypedValue) error {
	var leafVal interface{}
	isLeaf := true
	switch tv.GetValue().(type) {
	case *gpb.TypedValue_JsonIetfVal:
		updates, err := flattenOCJSON(tv.GetJsonIetfVal(), false)
		if err != nil {
			return err
		}
		if val, ok := updates[""]; len(updates) == 1 && ok {
			leafVal = val
		} else {
			isLeaf = false
			for subpath, value := range updates {
				pathToLeaf := path + subpath
				if prevVal, ok := intent.Updates[pathToLeaf]; ok && val != prevVal {
					return fmt.Errorf("gnmidiff: leaf value set twice in SetRequest: %v", pathToLeaf)
				}
				intent.Updates[pathToLeaf] = value
			}
		}
	default:
		var err error
		leafVal, err = protoLeafToJSON(tv)
		if err != nil {
			return err
		}
	}

	if isLeaf {
		// leaf replace is the same as a leaf update, so remove
		// the deletion action to keep the intent minimal.
		delete(intent.Deletes, path)
		if prevVal, ok := intent.Updates[path]; ok && leafVal != prevVal {
			return fmt.Errorf("gnmidiff: leaf value set twice in SetRequest: %v", path)
		}
		intent.Updates[path] = leafVal
	}
	return nil
}
