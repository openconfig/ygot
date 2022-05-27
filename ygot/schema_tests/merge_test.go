// Copyright 2022 Google Inc.
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

package schematest

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	oc "github.com/openconfig/ygot/exampleoc"
	"github.com/openconfig/ygot/ygot"
)

func TestMergeEmptyMap(t *testing.T) {
	want := &oc.Device{Interface: map[string]*oc.Interface{}}

	hasNil := &oc.Device{}
	hasEmpty := &oc.Device{Interface: map[string]*oc.Interface{}}
	got, err := ygot.MergeStructs(hasNil, hasEmpty, &ygot.MergeEmptyMaps{})
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("MergeStructs empty to nil (-got, +want):\n%s", diff)
	}

	hasNil = &oc.Device{}
	hasEmpty = &oc.Device{Interface: map[string]*oc.Interface{}}
	got, err = ygot.MergeStructs(hasEmpty, hasNil, &ygot.MergeEmptyMaps{})
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("MergeStructs nil to empty (-got, +want):\n%s", diff)
	}

	hasNil = &oc.Device{}
	hasEmpty = &oc.Device{Interface: map[string]*oc.Interface{}}
	ygot.MergeStructInto(hasNil, hasEmpty, &ygot.MergeEmptyMaps{})
	if diff := cmp.Diff(hasNil, want); diff != "" {
		t.Errorf("MergeStructInto empty to nil (-got, +want):\n%s", diff)
	}

	hasNil = &oc.Device{}
	hasEmpty = &oc.Device{Interface: map[string]*oc.Interface{}}
	ygot.MergeStructInto(hasEmpty, hasNil, &ygot.MergeEmptyMaps{})
	if diff := cmp.Diff(hasEmpty, want); diff != "" {
		t.Errorf("MergeStructInto nil to empty (-got, +want):\n%s", diff)
	}
}
