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

package gnmidiff

import (
	"fmt"
	"reflect"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
)

// DiffNotifications compares two slices of notifications and outputs a
// structured diff.
//
// Currently comparison on delete fields is not supported. Any delete fields
// will incur an error.
func DiffNotifications(a, b []*gpb.Notification) (StructuredDiff, error) {
	aUpdates, err := flattenNotifications(a...)
	if err != nil {
		return StructuredDiff{}, fmt.Errorf("cannot generate flat responses: %v", err)
	}
	bUpdates, err := flattenNotifications(b...)
	if err != nil {
		return StructuredDiff{}, fmt.Errorf("cannot generate flat responses: %v", err)
	}

	diff := UpdateDiff{
		MissingUpdates:    map[string]interface{}{},
		ExtraUpdates:      map[string]interface{}{},
		CommonUpdates:     map[string]interface{}{},
		MismatchedUpdates: map[string]MismatchedUpdate{},
	}

	for pathA, vA := range aUpdates {
		vB, ok := bUpdates[pathA]
		switch {
		case ok && !reflect.DeepEqual(vA, vB):
			diff.MismatchedUpdates[pathA] = MismatchedUpdate{A: vA, B: vB}
		case ok:
			diff.CommonUpdates[pathA] = vA
		default:
			diff.MissingUpdates[pathA] = vA
		}
		delete(bUpdates, pathA)
	}

	for pathB, vB := range bUpdates {
		diff.ExtraUpdates[pathB] = vB
	}

	return StructuredDiff{UpdateDiff: diff}, nil
}

// flattenNotifications returns a flattened set of notifications for a notification.
//
// This differs from gnmic's formatters.ResponsesFlat in that it is maximally
// tolerant to mixed JSON/scalar values, and provides output in JSON format
// rather than the raw format, which may be scalars or JSON.
func flattenNotifications(notifs ...*gpb.Notification) (map[string]interface{}, error) {
	updateIntent := setRequestIntent{
		Updates: map[string]interface{}{},
	}
	for _, notif := range notifs {
		// TODO: Handle deletes in notification.
		if len(notif.Delete) > 0 {
			return nil, fmt.Errorf("Deletes in notifications not currently supported.")
		}
		prefix, err := prefixStr(notif.Prefix)
		if err != nil {
			return nil, fmt.Errorf("gnmidiff: %v", err)
		}
		for _, upd := range notif.Update {
			path, err := fullPathStr(prefix, upd.Path)
			if err != nil {
				return nil, err
			}
			if err := updateIntent.populateUpdate(path, upd.Val, nil, false); err != nil {
				return nil, err
			}
		}
	}
	return updateIntent.Updates, nil
}
