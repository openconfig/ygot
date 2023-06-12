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
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/openconfig/ygot/integration_tests/schemaops/ctestschema"
	"github.com/openconfig/ygot/testutil"
	"github.com/openconfig/ygot/ygot"
	"google.golang.org/protobuf/testing/protocmp"

	gnmipb "github.com/openconfig/gnmi/proto/gnmi"
)

func mustPathElem(s string) []*gnmipb.PathElem {
	p, err := ygot.StringToStructuredPath(s)
	if err != nil {
		panic(err)
	}
	return p.Elem
}

func TestTogNMINotificationsOrderedMap(t *testing.T) {
	tests := []struct {
		name           string
		inTimestamp    int64
		inStruct       ygot.GoStruct
		inConfig       ygot.GNMINotificationsConfig
		wantAtomicMsgs int
		want           []*gnmipb.Notification
		wantErr        bool
	}{{
		name:        "struct with two ordered lists",
		inTimestamp: 42,
		inStruct: &ctestschema.Device{
			OrderedList:           ctestschema.GetOrderedMap(t),
			OrderedMultikeyedList: ctestschema.GetOrderedMapMultikeyed(t),
			OtherData: &ctestschema.OtherData{
				Motd: ygot.String("abc -> def"),
			},
		},
		inConfig: ygot.GNMINotificationsConfig{
			UsePathElem:    true,
			PathElemPrefix: mustPathElem("heart/of/gold"),
		},
		wantAtomicMsgs: 2,
		want: []*gnmipb.Notification{{
			Timestamp: 42,
			Prefix:    mustPath("heart/of/gold"),
			Update: []*gnmipb.Update{{
				Path: mustPath(`other-data/config/motd`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "abc -> def"}},
			}},
		}, {
			Timestamp: 42,
			Atomic:    true,
			Prefix:    mustPath("heart/of/gold/ordered-lists"),
			Update: []*gnmipb.Update{{
				Path: mustPath(`ordered-list[key=foo]/config/key`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "foo"}},
			}, {
				Path: mustPath(`ordered-list[key=foo]/key`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "foo"}},
			}, {
				Path: mustPath(`ordered-list[key=foo]/config/value`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "foo-val"}},
			}, {
				Path: mustPath(`ordered-list[key=bar]/config/key`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "bar"}},
			}, {
				Path: mustPath(`ordered-list[key=bar]/key`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "bar"}},
			}, {
				Path: mustPath(`ordered-list[key=bar]/config/value`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "bar-val"}},
			}},
		}, {
			Timestamp: 42,
			Atomic:    true,
			Prefix:    mustPath("heart/of/gold/ordered-multikeyed-lists"),
			Update: []*gnmipb.Update{{
				Path: mustPath(`ordered-multikeyed-list[key1=foo][key2=42]/config/key1`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "foo"}},
			}, {
				Path: mustPath(`ordered-multikeyed-list[key1=foo][key2=42]/key1`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "foo"}},
			}, {
				Path: mustPath(`ordered-multikeyed-list[key1=foo][key2=42]/config/key2`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_UintVal{UintVal: 42}},
			}, {
				Path: mustPath(`ordered-multikeyed-list[key1=foo][key2=42]/key2`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_UintVal{UintVal: 42}},
			}, {
				Path: mustPath(`ordered-multikeyed-list[key1=foo][key2=42]/config/value`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "foo-val"}},
			}, {
				Path: mustPath(`ordered-multikeyed-list[key1=bar][key2=42]/config/key1`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "bar"}},
			}, {
				Path: mustPath(`ordered-multikeyed-list[key1=bar][key2=42]/key1`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "bar"}},
			}, {
				Path: mustPath(`ordered-multikeyed-list[key1=bar][key2=42]/config/key2`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_UintVal{UintVal: 42}},
			}, {
				Path: mustPath(`ordered-multikeyed-list[key1=bar][key2=42]/key2`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_UintVal{UintVal: 42}},
			}, {
				Path: mustPath(`ordered-multikeyed-list[key1=bar][key2=42]/config/value`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "bar-val"}},
			}, {
				Path: mustPath(`ordered-multikeyed-list[key1=baz][key2=84]/config/key1`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "baz"}},
			}, {
				Path: mustPath(`ordered-multikeyed-list[key1=baz][key2=84]/key1`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "baz"}},
			}, {
				Path: mustPath(`ordered-multikeyed-list[key1=baz][key2=84]/config/key2`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_UintVal{UintVal: 84}},
			}, {
				Path: mustPath(`ordered-multikeyed-list[key1=baz][key2=84]/key2`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_UintVal{UintVal: 84}},
			}, {
				Path: mustPath(`ordered-multikeyed-list[key1=baz][key2=84]/config/value`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "baz-val"}},
			}},
		}},
	}, {
		name:        "struct with only an ordered list",
		inTimestamp: 42,
		inStruct: &ctestschema.Device{
			OrderedList: ctestschema.GetOrderedMap(t),
		},
		inConfig: ygot.GNMINotificationsConfig{
			UsePathElem:    true,
			PathElemPrefix: mustPathElem("heart/of/gold"),
		},
		wantAtomicMsgs: 1,
		want: []*gnmipb.Notification{{
			Timestamp: 42,
			Atomic:    true,
			Prefix:    mustPath("heart/of/gold/ordered-lists"),
			Update: []*gnmipb.Update{{
				Path: mustPath(`ordered-list[key=foo]/config/key`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "foo"}},
			}, {
				Path: mustPath(`ordered-list[key=foo]/key`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "foo"}},
			}, {
				Path: mustPath(`ordered-list[key=foo]/config/value`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "foo-val"}},
			}, {
				Path: mustPath(`ordered-list[key=bar]/config/key`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "bar"}},
			}, {
				Path: mustPath(`ordered-list[key=bar]/key`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "bar"}},
			}, {
				Path: mustPath(`ordered-list[key=bar]/config/value`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "bar-val"}},
			}},
		}},
	}, {
		name:        "ordered list string slice mode",
		inTimestamp: 42,
		inStruct: &ctestschema.Device{
			OrderedList: ctestschema.GetOrderedMap(t),
			OtherData: &ctestschema.OtherData{
				Motd: ygot.String("abc -> def"),
			},
		},
		inConfig: ygot.GNMINotificationsConfig{
			StringSlicePrefix: []string{"heart", "of", "gold"},
		},
		wantAtomicMsgs: 1,
		want: []*gnmipb.Notification{{
			Timestamp: 42,
			Prefix:    &gnmipb.Path{Element: []string{"heart", "of", "gold"}},
			Update: []*gnmipb.Update{{
				Path: &gnmipb.Path{Element: []string{"other-data", "config", "motd"}},
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "abc -> def"}},
			}},
		}, {
			Timestamp: 42,
			Atomic:    true,
			Prefix:    &gnmipb.Path{Element: []string{"heart", "of", "gold", "ordered-lists"}},
			Update: []*gnmipb.Update{{
				Path: &gnmipb.Path{Element: []string{"ordered-list", "foo", "config", "key"}},
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "foo"}},
			}, {
				Path: &gnmipb.Path{Element: []string{"ordered-list", "foo", "key"}},
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "foo"}},
			}, {
				Path: &gnmipb.Path{Element: []string{"ordered-list", "foo", "config", "value"}},
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "foo-val"}},
			}, {
				Path: &gnmipb.Path{Element: []string{"ordered-list", "bar", "config", "key"}},
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "bar"}},
			}, {
				Path: &gnmipb.Path{Element: []string{"ordered-list", "bar", "key"}},
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "bar"}},
			}, {
				Path: &gnmipb.Path{Element: []string{"ordered-list", "bar", "config", "value"}},
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "bar-val"}},
			}},
		}},
	}, {
		name:        "struct with nested ordered list",
		inTimestamp: 42,
		inStruct: &ctestschema.Device{
			OrderedList: ctestschema.GetNestedOrderedMap(t),
		},
		inConfig: ygot.GNMINotificationsConfig{
			UsePathElem:    true,
			PathElemPrefix: mustPathElem("heart/of/gold"),
		},
		wantAtomicMsgs: 1,
		wantErr:        true,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ygot.TogNMINotifications(tt.inStruct, tt.inTimestamp, tt.inConfig)
			if (err != nil) != tt.wantErr {
				t.Errorf("%s: TogNMINotifications(%v, %v, %v): got unexpected error: %v", tt.name, tt.inStruct, tt.inTimestamp, tt.inConfig, err)
			}
			if err != nil {
				return
			}

			if gotLen, wantLen := len(got), len(tt.want); gotLen != wantLen {
				t.Errorf("gotLen: %d, wantLen: %d", gotLen, wantLen)
				if diff := cmp.Diff(got, tt.want, cmpopts.SortSlices(testutil.NotificationLess), testutil.NotificationComparer()); diff != "" {
					t.Errorf("%s: telemetry-atomic values of TogNMINotifications(%v, %v): did not get expected Notification, diff(-got,+want):%s\n", tt.name, tt.inStruct, tt.inTimestamp, diff)
				}
				return
			}

			// Avoid test flakiness by ignoring the update ordering. Required because
			// there is no order to the map of fields that are returned by the struct
			// output.
			if diff := cmp.Diff(got[:len(got)-tt.wantAtomicMsgs], tt.want[:len(got)-tt.wantAtomicMsgs], cmpopts.SortSlices(testutil.NotificationLess), testutil.NotificationComparer()); diff != "" {
				t.Errorf("%s: non-telemetry-atomic values of TogNMINotifications(%v, %v): did not get expected Notification, diff(-got,+want):%s\n", tt.name, tt.inStruct, tt.inTimestamp, diff)
			}

			if diff := cmp.Diff(got[len(got)-tt.wantAtomicMsgs:], tt.want[len(got)-tt.wantAtomicMsgs:], cmpopts.SortSlices(testutil.NotificationLess), protocmp.Transform()); diff != "" {
				t.Errorf("%s: telemetry-atomic values of TogNMINotifications(%v, %v): did not get expected Notification, diff(-got,+want):%s\n", tt.name, tt.inStruct, tt.inTimestamp, diff)
			}
		})
	}
}
