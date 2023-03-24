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
	"fmt"
	"reflect"
	"strings"

	"github.com/derekparker/trie"
	"github.com/openconfig/ygot/ygot"
	"github.com/openconfig/ygot/ytypes"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
)

// SetRequestIntentDiff contains the intent difference between two SetRequests.
type SetRequestIntentDiff StructuredDiff

// Format outputs the SetRequestIntentDiff in human-readable format.
//
// NOTE: Do not depend on the output of this being stable.
func (diff SetRequestIntentDiff) Format(f Format) string {
	f.title = "SetRequestIntentDiff"
	f.aName = "A"
	f.bName = "B"
	f.deleteTitle = "deletes/replaces"
	f.deleteDesc = "deleted or replaced"
	return StructuredDiff(diff).Format(f)
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
// Currently, support is only for SetRequests whose delete, replace and updates
// that don't have conflicts. If a conflict exists, then an error will be
// returned.
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
		DeleteDiff: DeleteDiff{
			MissingDeletes: map[string]struct{}{},
			ExtraDeletes:   map[string]struct{}{},
			CommonDeletes:  map[string]struct{}{},
		},
		UpdateDiff: UpdateDiff{
			MissingUpdates:    map[string]interface{}{},
			ExtraUpdates:      map[string]interface{}{},
			CommonUpdates:     map[string]interface{}{},
			MismatchedUpdates: map[string]MismatchedUpdate{},
		},
	}
	for path := range intentA.Deletes {
		if _, ok := intentB.Deletes[path]; ok {
			delete(intentA.Deletes, path)
			delete(intentB.Deletes, path)
			diff.CommonDeletes[path] = struct{}{}
		}
	}
	diff.MissingDeletes = intentA.Deletes
	diff.ExtraDeletes = intentB.Deletes
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
	diff.MissingUpdates = intentA.Updates
	diff.ExtraUpdates = intentB.Updates
	return diff, nil
}

// minimalSetRequestIntent returns a unique and minimal intent for a SetRequest.
//
// TODO: Currently, support is only for SetRequests whose delete, replace and updates
// that don't have conflicts. If a conflict exists, then an error will be
// returned.
func minimalSetRequestIntent(req *gpb.SetRequest, schema *ytypes.Schema) (setRequestIntent, error) {
	if req == nil {
		req = &gpb.SetRequest{}
	}
	prefix, err := prefixStr(req.Prefix)
	if err != nil {
		return setRequestIntent{}, fmt.Errorf("gnmidiff: %v", err)
	}

	intent := setRequestIntent{
		Deletes: map[string]struct{}{},
		Updates: map[string]interface{}{},
	}
	// NOTE: This simple trie will not work if we intend to check conflicts with wildcard deletion paths.
	t := trie.New()
	for _, gPath := range req.Delete {
		path, err := fullPathStr(prefix, gPath)
		if err != nil {
			return setRequestIntent{}, err
		}
		if _, ok := intent.Deletes[path]; ok {
			return setRequestIntent{}, fmt.Errorf("gnmidiff: conflicting replaces in SetRequest: %v", path)
		}
		intent.Deletes[path] = struct{}{}
		t.Add(path, nil)
	}
	for _, upd := range req.Replace {
		path, err := fullPathStr(prefix, upd.Path)
		if err != nil {
			return setRequestIntent{}, err
		}
		if _, ok := intent.Deletes[path]; ok {
			return setRequestIntent{}, fmt.Errorf("gnmidiff: conflicting replaces in SetRequest: %v", path)
		}
		intent.Deletes[path] = struct{}{}
		t.Add(path, nil)

		if err := intent.populateUpdate(path, upd.GetVal(), schema, true); err != nil {
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
		path, err := fullPathStr(prefix, upd.Path)
		if err != nil {
			return setRequestIntent{}, err
		}

		if err := intent.populateUpdate(path, upd.GetVal(), schema, true); err != nil {
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

// prefixStr returns the path version of a prefix path, handling corner cases.
func prefixStr(prefix *gpb.Path) (string, error) {
	if prefix == nil {
		prefix = &gpb.Path{}
	}
	prefixStr, err := ygot.PathToString(prefix)
	if err != nil {
		return "", fmt.Errorf("gnmidiff/prefixStr: %w", err)
	}
	return prefixStr, nil
}

// fullPathStr returns the full string given a prefix and a proto path,
// handling corner cases.
func fullPathStr(prefix string, path *gpb.Path) (string, error) {
	pathStr, err := ygot.PathToString(path)
	if err != nil {
		return "", fmt.Errorf("gnmidiff/fullPathStr: %w", err)
	}
	prefix = strings.TrimSuffix(prefix, "/")
	pathStr = strings.TrimSuffix(pathStr, "/")
	return prefix + pathStr, nil
}
