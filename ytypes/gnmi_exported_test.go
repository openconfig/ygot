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

	"github.com/google/go-cmp/cmp"
	gpb "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/ygot/integration_tests/schemaops/ctestschema"
	"github.com/openconfig/ygot/integration_tests/schemaops/utestschema"
	"github.com/openconfig/ygot/internal/ytestutil"
	"github.com/openconfig/ygot/ygot"
	"github.com/openconfig/ygot/ytypes"
)

func TestUnmarshalNotificationsOrderedMap(t *testing.T) {
	tests := []struct {
		desc            string
		inSchema        *ytypes.Schema
		inNotifications []*gpb.Notification
		inUnmarshalOpts []ytypes.UnmarshalOpt
		want            ygot.GoStruct
		wantErr         bool
	}{{
		desc: "atomic update to a non-empty struct",
		inSchema: &ytypes.Schema{
			Root: &ctestschema.Device{
				OrderedList: ctestschema.GetOrderedMap(t),
			},
			SchemaTree: ctestschema.SchemaTree,
		},
		inNotifications: []*gpb.Notification{{
			Timestamp: 42,
			Atomic:    true,
			Prefix:    mustPath("/ordered-lists"),
			Update: []*gpb.Update{{
				Path: mustPath(`ordered-list[key=boo]/config/key`),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "boo"}},
			}, {
				Path: mustPath(`ordered-list[key=boo]/key`),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "boo"}},
			}, {
				Path: mustPath(`ordered-list[key=boo]/config/value`),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "boo-val"}},
			}, {
				Path: mustPath(`ordered-list[key=coo]/config/key`),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "coo"}},
			}, {
				Path: mustPath(`ordered-list[key=coo]/key`),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "coo"}},
			}, {
				Path: mustPath(`ordered-list[key=coo]/config/value`),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "coo-val"}},
			}},
		}},
		want: &ctestschema.Device{
			OrderedList: func() *ctestschema.OrderedList_OrderedMap {
				orderedMap := &ctestschema.OrderedList_OrderedMap{}
				v, err := orderedMap.AppendNew("boo")
				if err != nil {
					t.Error(err)
				}
				v.Value = ygot.String("boo-val")

				v, err = orderedMap.AppendNew("coo")
				if err != nil {
					t.Error(err)
				}
				v.Value = ygot.String("coo-val")
				return orderedMap
			}(),
		},
	}, {
		desc: "atomic update to a non-empty uncompressed struct",
		inSchema: &ytypes.Schema{
			Root:       utestschema.GetDeviceWithOrderedMap(t),
			SchemaTree: utestschema.SchemaTree,
		},
		inNotifications: []*gpb.Notification{{
			Timestamp: 42,
			Atomic:    true,
			Prefix:    mustPath("/ordered-lists"),
			Update: []*gpb.Update{{
				Path: mustPath(`ordered-list[key=boo]/config/key`),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "boo"}},
			}, {
				Path: mustPath(`ordered-list[key=boo]/key`),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "boo"}},
			}, {
				Path: mustPath(`ordered-list[key=boo]/config/value`),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "boo-val"}},
			}, {
				Path: mustPath(`ordered-list[key=coo]/config/key`),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "coo"}},
			}, {
				Path: mustPath(`ordered-list[key=coo]/key`),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "coo"}},
			}, {
				Path: mustPath(`ordered-list[key=coo]/state/value`),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "coo-val"}},
			}},
		}},
		want: &utestschema.Device{
			OrderedLists: &utestschema.Ctestschema_OrderedLists{
				OrderedList: func() *utestschema.Ctestschema_OrderedLists_OrderedList_OrderedMap {
					orderedMap := &utestschema.Ctestschema_OrderedLists_OrderedList_OrderedMap{}
					v, err := orderedMap.AppendNew("boo")
					if err != nil {
						t.Error(err)
					}
					v.GetOrCreateConfig().Key = ygot.String("boo")
					v.GetOrCreateConfig().Value = ygot.String("boo-val")
					v, err = orderedMap.AppendNew("coo")
					if err != nil {
						t.Error(err)
					}
					v.GetOrCreateConfig().Key = ygot.String("coo")
					v.GetOrCreateState().Value = ygot.String("coo-val")
					return orderedMap
				}(),
			},
		},
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := ytypes.UnmarshalNotifications(tt.inSchema, tt.inNotifications, tt.inUnmarshalOpts...)
			if gotErr := err != nil; gotErr != tt.wantErr {
				t.Fatalf("got error: %v, want: %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				if diff := cmp.Diff(tt.inSchema.Root, tt.want, ytestutil.OrderedMapCmpOptions...); diff != "" {
					t.Errorf("(-got, +want):\n%s", diff)
				}
			}
		})
	}
}
