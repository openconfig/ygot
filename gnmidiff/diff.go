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
	"sort"
	"strconv"
	"strings"
)

// MismatchedUpdate represents two different update values for the same leaf
// node.
type MismatchedUpdate struct {
	// A is the update value in A.
	A interface{}
	// B is the update value in B.
	B interface{}
}

// StructuredDiff contains a set of difference fields that can be used by
// SetRequests/Notifications.
//
// - The key of the maps is the string representation of a gpb.Path constructed
// by ygot.PathToString.
// - The value of the update fields is the JSON_IETF representation of the
// value. This is to facilitate comparing JSON_IETF-represented values whose
// real data type is obscured without knowledge of the YANG schema.
type StructuredDiff struct {
	DeleteDiff
	UpdateDiff
}

type DeleteDiff struct {
	MissingDeletes map[string]struct{}
	ExtraDeletes   map[string]struct{}
	CommonDeletes  map[string]struct{}
}

type UpdateDiff struct {
	// MissingUpdates (-) are updates specified in the first argument but
	// missing in the second argument.
	MissingUpdates map[string]interface{}
	// ExtraUpdates (+) are updates not specified in the first argument but
	// present in the second argument.
	ExtraUpdates      map[string]interface{}
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
	// title is an optional custom title of the diff.
	title string
	// aName is an optional custom name for A in the diff.
	aName string
	// bName is an optional custom name for B in the diff.
	bName string
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
func (diff StructuredDiff) Format(f Format) string {
	var b strings.Builder
	if f.title == "" {
		f.title = "StructuredDiff"
	}
	if f.aName == "" {
		f.aName = "A"
	}
	if f.bName == "" {
		f.bName = "B"
	}
	b.WriteString(fmt.Sprintf("%s(-%s, +%s):\n", f.title, f.aName, f.bName))

	deleteDiff := diff.DeleteDiff.format(f)
	if deleteDiff != "" {
		b.WriteString("-------- deletes --------\n")
		b.WriteString(deleteDiff)
		b.WriteString("-------- updates --------\n")
	}
	b.WriteString(diff.UpdateDiff.format(f))
	return b.String()
}

// format outputs the UpdateDiff in human-readable format.
//
// This is intended to aid StructuredDiff when building up its exported Format
// output.
func (diff UpdateDiff) format(f Format) string {
	var b strings.Builder
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

	if f.Full {
		writeUpdates(diff.CommonUpdates, ' ')
	}
	writeUpdates(diff.MissingUpdates, '-')
	writeUpdates(diff.ExtraUpdates, '+')
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

// format outputs the DeleteDiff in human-readable format.
//
// This is intended to aid StructuredDiff when building up its exported Format
// output.
func (diff DeleteDiff) format(f Format) string {
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

	if f.Full {
		writeDeletes(diff.CommonDeletes, ' ')
	}
	writeDeletes(diff.MissingDeletes, '-')
	writeDeletes(diff.ExtraDeletes, '+')
	return b.String()
}
