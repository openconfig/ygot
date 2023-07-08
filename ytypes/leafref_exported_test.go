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

package ytypes_test

import (
	"testing"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/integration_tests/schemaops/ctestschema"
	"github.com/openconfig/ygot/integration_tests/schemaops/utestschema"
	"github.com/openconfig/ygot/ygot"
	"github.com/openconfig/ygot/ytypes"
)

func TestValidateLeafRefDataOrderedMap(t *testing.T) {
	tests := []struct {
		desc     string
		inSchema *yang.Entry
		in       ygot.GoStruct
		opts     *ytypes.LeafrefOptions
		wantErr  string
	}{{
		desc:     "checking compressed list key leafref",
		inSchema: ctestschema.SchemaTree["Device"],
		in: &ctestschema.Device{
			OrderedList: ctestschema.GetOrderedMap(t),
		},
	}, {
		desc:     "checking uncompressed list key leafref success",
		inSchema: utestschema.SchemaTree["Device"],
		in: &utestschema.Device{
			OrderedLists: &utestschema.Ctestschema_OrderedLists{
				OrderedList: func() *utestschema.Ctestschema_OrderedLists_OrderedList_OrderedMap {
					orderedMap := &utestschema.Ctestschema_OrderedLists_OrderedList_OrderedMap{}
					v, err := orderedMap.AppendNew("foo")
					if err != nil {
						t.Error(err)
					}
					// Config value
					v.GetOrCreateConfig().Value = ygot.String("foo-val")
					v, err = orderedMap.AppendNew("bar")
					if err != nil {
						t.Error(err)
					}
					// State value
					v.GetOrCreateState().Value = ygot.String("bar-val")
					return orderedMap
				}(),
			},
		},
		wantErr: `pointed-to value with path ../config/key from field Key value foo (string ptr) schema /device/ordered-lists/ordered-list/key is empty set
pointed-to value with path ../config/key from field Key value bar (string ptr) schema /device/ordered-lists/ordered-list/key is empty set`,
	}, {
		desc:     "checking uncompressed list key leafref fail",
		inSchema: utestschema.SchemaTree["Device"],
		in:       utestschema.GetDeviceWithOrderedMap(t),
		wantErr: `pointed-to value with path ../config/key from field Key value foo (string ptr) schema /device/ordered-lists/ordered-list/key is empty set
pointed-to value with path ../config/key from field Key value bar (string ptr) schema /device/ordered-lists/ordered-list/key is empty set`,
	}, {
		desc:     "checking short leafref",
		inSchema: ctestschema.SchemaTree["Device"],
		in: &ctestschema.Device{
			OrderedList: func() *ctestschema.OrderedList_OrderedMap {
				om := &ctestschema.OrderedList_OrderedMap{}
				ome, err := om.AppendNew("bar")
				if err != nil {
					t.Fatal(err)
				}
				ome.ParentKey = ygot.String("foo")
				_, err = om.AppendNew("foo")
				if err != nil {
					t.Fatal(err)
				}
				return om
			}(),
		},
	}, {
		desc:     "checking long leafref",
		inSchema: ctestschema.SchemaTree["Device"],
		in: &ctestschema.Device{
			OrderedList: func() *ctestschema.OrderedList_OrderedMap {
				om := &ctestschema.OrderedList_OrderedMap{}
				_, err := om.AppendNew("bar")
				if err != nil {
					t.Fatal(err)
				}
				ome, err := om.AppendNew("foo")
				if err != nil {
					t.Fatal(err)
				}
				ch, err := ome.AppendNewOrderedList("foo-child")
				if err != nil {
					t.Fatal(err)
				}
				ch.ParentKey = ygot.String("foo")
				return om
			}(),
		},
	}, {
		desc:     "checking longer leafref doesn't match",
		inSchema: ctestschema.SchemaTree["Device"],
		in: &ctestschema.Device{
			OrderedList: func() *ctestschema.OrderedList_OrderedMap {
				om := &ctestschema.OrderedList_OrderedMap{}
				_, err := om.AppendNew("bar")
				if err != nil {
					t.Fatal(err)
				}
				ome, err := om.AppendNew("foo")
				if err != nil {
					t.Fatal(err)
				}
				ch, err := ome.AppendNewOrderedList("foo-child")
				if err != nil {
					t.Fatal(err)
				}
				ch.ParentKey = ygot.String("bar")
				return om
			}(),
		},
		wantErr: `field name ParentKey value bar (string ptr) schema path /device/ordered-lists/ordered-list/ordered-lists/ordered-list/state/parent-key has leafref path ../../../../config/key not equal to any target nodes`,
	}, {
		desc:     "checking longer leafref doesn't exist",
		inSchema: ctestschema.SchemaTree["Device"],
		in: &ctestschema.Device{
			OrderedList: func() *ctestschema.OrderedList_OrderedMap {
				om := &ctestschema.OrderedList_OrderedMap{}
				_, err := om.AppendNew("bar")
				if err != nil {
					t.Fatal(err)
				}
				ome, err := om.AppendNew("foo")
				if err != nil {
					t.Fatal(err)
				}
				ch, err := ome.AppendNewOrderedList("foo-child")
				if err != nil {
					t.Fatal(err)
				}
				ch.ParentKey = ygot.String("baz")
				return om
			}(),
		},
		wantErr: `field name ParentKey value baz (string ptr) schema path /device/ordered-lists/ordered-list/ordered-lists/ordered-list/state/parent-key has leafref path ../../../../config/key not equal to any target nodes`,
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			errs := ytypes.ValidateLeafRefData(tt.inSchema, tt.in, tt.opts)
			if got, want := errs.String(), tt.wantErr; got != want {
				t.Errorf("%s: got error: %s, want error: %s", tt.desc, got, want)
			}
		})
	}
}
