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

package ygot_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/integration_tests/schemaops/ctestschema"
	"github.com/openconfig/ygot/integration_tests/schemaops/utestschema"
	"github.com/openconfig/ygot/internal/ytestutil"
	"github.com/openconfig/ygot/ygot"
)

func TestPruneConfigFalseOrderedMap(t *testing.T) {
	tests := []struct {
		desc     string
		inSchema *yang.Entry
		inStruct ygot.GoStruct
		want     ygot.GoStruct
		wantErr  bool
	}{{
		desc:     "prune through ordered map",
		inSchema: ctestschema.SchemaTree["Device"],
		inStruct: &ctestschema.Device{
			OrderedList: func() *ctestschema.OrderedList_OrderedMap {
				om := &ctestschema.OrderedList_OrderedMap{}
				ome, err := om.AppendNew("foo")
				if err != nil {
					t.Fatal(err)
				}
				ome.RoValue = ygot.String("ro-value")
				return om
			}(),
		},
		want: &ctestschema.Device{
			OrderedList: func() *ctestschema.OrderedList_OrderedMap {
				om := &ctestschema.OrderedList_OrderedMap{}
				_, err := om.AppendNew("foo")
				if err != nil {
					t.Fatal(err)
				}
				return om
			}(),
		},
	}, {
		desc:     "prune through uncompressed ordered map",
		inSchema: utestschema.SchemaTree["Device"],
		inStruct: &utestschema.Device{
			OrderedLists: &utestschema.Ctestschema_OrderedLists{
				OrderedList: func() *utestschema.Ctestschema_OrderedLists_OrderedList_OrderedMap {
					om := &utestschema.Ctestschema_OrderedLists_OrderedList_OrderedMap{}
					ome, err := om.AppendNew("foo")
					if err != nil {
						t.Fatal(err)
					}
					ome.GetOrCreateState().RoValue = ygot.String("ro-value")
					return om
				}(),
			},
		},
		want: &utestschema.Device{
			OrderedLists: &utestschema.Ctestschema_OrderedLists{
				OrderedList: func() *utestschema.Ctestschema_OrderedLists_OrderedList_OrderedMap {
					om := &utestschema.Ctestschema_OrderedLists_OrderedList_OrderedMap{}
					_, err := om.AppendNew("foo")
					if err != nil {
						t.Fatal(err)
					}
					return om
				}(),
			},
		},
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := ygot.PruneConfigFalse(tt.inSchema, tt.inStruct)
			if (err != nil) != tt.wantErr {
				t.Errorf("Got error %v, wantErr: %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if diff := cmp.Diff(tt.inStruct, tt.want, ytestutil.OrderedMapCmpOptions...); diff != "" {
				t.Errorf("diff(-got, +want):\n%s", diff)
			}
		})
	}
}
