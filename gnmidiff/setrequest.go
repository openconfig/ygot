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

	"github.com/derekparker/trie"
	"github.com/openconfig/ygot/ygot"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
)

// SetRequestDiff contains the intent difference between two SetRequests.
//
// - The key of the maps is the string representation of a gpb.Path constructed
// by ygot.PathToString.
// - The value of the update fields is the JSON_IETF representation of the
// value.
type SetRequestDiff struct {
	AOnlyDeletes  map[string]struct{}
	BOnlyDeletes  map[string]struct{}
	CommonDeletes map[string]struct{}
	AOnlyUpdates  map[string]interface{}
	BOnlyUpdates  map[string]interface{}
	CommonUpdates map[string]interface{}
}

// DiffSetRequest returns a unique and minimal intent diff of two SetRequests.
//
// Currently, support is only for SetRequests without any delete paths, and
// replace and updates that don't have conflicting leaf values. If not
// supported, then an error will be returned.
func DiffSetRequest(a *gpb.SetRequest, b *gpb.SetRequest) (SetRequestDiff, error) {
	intentA, err := minimalSetRequestIntent(a)
	if err != nil {
		return SetRequestDiff{}, fmt.Errorf("DiffSetRequest on b: %v", err)
	}
	intentB, err := minimalSetRequestIntent(b)
	if err != nil {
		return SetRequestDiff{}, fmt.Errorf("DiffSetRequest on a: %v", err)
	}
	diff := SetRequestDiff{}
	// TODO
	for range intentA.Deletes {
	}
	for range intentB.Deletes {
	}
	for range intentA.Updates {
	}
	for range intentB.Updates {
	}
	return diff, nil
}

type setRequestIntent struct {
	Deletes map[string]struct{}
	Updates map[string]interface{}
}

// minimalSetRequestIntent returns a unique and minimal intent for a SetRequest.
//
// Currently, support is only for SetRequests without any delete paths, and
// replace and updates that don't have conflicting leaf values. If not
// supported, then an error will be returned.
func minimalSetRequestIntent(req *gpb.SetRequest) (setRequestIntent, error) {
	if len(req.Delete) > 0 {
		return setRequestIntent{}, fmt.Errorf("gnmidiff: delete paths are not supported.")
	}
	intent := setRequestIntent{
		Deletes: map[string]struct{}{},
		Updates: map[string]interface{}{},
	}
	// NOTE: This simple trie will not work if we intend to check conflicts with wildcard deletion paths.
	t := trie.New()
	// FIXME: error out if not openconfig origin. Also update doc-comments.
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

		if err := populateUpdates(&intent, path, upd.GetVal()); err != nil {
			return setRequestIntent{}, err
		}
	}

	// Do prefix match to check for conflicting replace paths.
	for _, path := range t.Keys() {
		if matches := t.PrefixSearch(path); len(matches) >= 2 {
			return setRequestIntent{}, fmt.Errorf("gnmidiff: conflicting replaces in SetRequest: %v", matches)
		}
	}

	for _, upd := range req.Update {
		path, err := ygot.PathToString(upd.Path)
		if err != nil {
			return setRequestIntent{}, fmt.Errorf("gnmidiff: %v", err)
		}

		if err := populateUpdates(&intent, path, upd.GetVal()); err != nil {
			return setRequestIntent{}, err
		}
	}

	t = trie.New()
	for path := range intent.Updates {
		t.Add(path, nil)
	}

	// Do prefix match to check for conflicting update paths.
	for _, path := range t.Keys() {
		if matches := t.PrefixSearch(path); len(matches) >= 2 {
			return setRequestIntent{}, fmt.Errorf("gnmidiff: bad SetRequest, there are leaf updates that have a prefix match: %v", matches)
		}
	}

	return intent, nil
}

// populateUpdates populates all leaf updates at the given path into the intent.
//
// Note: The input path must NOT end with "/".
//
// e.g. /a for b/c="foo" would introduce an update of /a/b/c="foo" into the intent.
func populateUpdates(intent *setRequestIntent, path string, tv *gpb.TypedValue) error {
	if len(path) > 0 && path[len(path)-1] == '/' {
		return fmt.Errorf("gnmidiff: invalid input path %q, must not end with \"/\"", path)
	}
	var leafVal interface{}
	isLeaf := true
	switch tv.GetValue().(type) {
	// TODO: Handle other scalar types.
	case *gpb.TypedValue_StringVal:
		leafVal = tv.GetStringVal()
	case *gpb.TypedValue_BoolVal:
		leafVal = tv.GetBoolVal()
	case *gpb.TypedValue_JsonIetfVal:
		updates, err := flattenOCJSON(tv.GetJsonIetfVal())
		if err != nil {
			return err
		}
		if val, ok := updates[""]; len(updates) == 1 && ok {
			leafVal = val
		} else {
			isLeaf = false
			for subpath, value := range updates {
				pathToLeaf := path + subpath
				if _, ok := intent.Updates[pathToLeaf]; ok {
					return fmt.Errorf("gnmidiff: leaf value set twice in SetRequest: %v", pathToLeaf)
				}
				intent.Updates[pathToLeaf] = value
			}
		}
	default:
		return fmt.Errorf("unsupported TypedValue type %T (only string and JSON_IETF are currently supported)", tv.GetValue())
	}

	if isLeaf {
		// leaf replace is the same as a leaf update, so remove
		// the deletion action to keep the intent minimal.
		delete(intent.Deletes, path)
		if _, ok := intent.Updates[path]; ok {
			return fmt.Errorf("gnmidiff: leaf value set twice in SetRequest: %v", path)
		}
		intent.Updates[path] = leafVal
	}
	return nil
}
