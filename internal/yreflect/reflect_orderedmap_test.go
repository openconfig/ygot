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

package yreflect_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/openconfig/gnmi/errdiff"
	"github.com/openconfig/ygot/integration_tests/schemaops/ctestschema"
	"github.com/openconfig/ygot/internal/yreflect"
	"github.com/openconfig/ygot/ygot"
)

func TestAppendIntoOrderedMap(t *testing.T) {
	om := ctestschema.GetOrderedMap(t)
	newKey := "new"
	for om.Get(newKey) != nil {
		newKey += "-repeat"
	}
	om2 := ctestschema.GetOrderedMap(t)
	newOrderedListElement, err := om2.AppendNew(newKey)
	if err != nil {
		t.Fatalf("om2.AppendNew(newKey) failed, this is unexpected for testing: %v", err)
	}
	var stringval = "val"
	newOrderedListElement.Value = &stringval

	tests := []struct {
		desc          string
		inMap         ygot.GoOrderedMap
		inValue       interface{}
		wantMap       ygot.GoOrderedMap
		wantErrSubstr string
	}{{
		desc:    "ordered map",
		inMap:   om,
		inValue: newOrderedListElement,
		wantMap: om2,
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := yreflect.AppendIntoOrderedMap(tt.inMap, tt.inValue)
			if diff := errdiff.Substring(err, tt.wantErrSubstr); diff != "" {
				t.Fatalf("InsertIntoMap: %s", diff)
			}
			if err != nil {
				return
			}

			if diff := cmp.Diff(tt.wantMap, tt.inMap, cmp.AllowUnexported(ctestschema.OrderedList_OrderedMap{})); diff != "" {
				t.Errorf("(-want, +got):\n%s", diff)
			}
		})
	}
}
