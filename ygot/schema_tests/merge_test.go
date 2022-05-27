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
	src := &oc.Device{Interface: map[string]*oc.Interface{}}

	dst := &oc.Device{}
	got, err := ygot.MergeStructs(dst, src, &ygot.MergeEmptyMaps{})
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(got, src); diff != "" {
		t.Errorf("MergeStructs (-got, +want):\n%s", diff)
	}

	dst = &oc.Device{}
	ygot.MergeStructInto(dst, src, &ygot.MergeEmptyMaps{})
	if diff := cmp.Diff(dst, src); diff != "" {
		t.Errorf("MergeStructs (-got, +want):\n%s", diff)
	}
}
