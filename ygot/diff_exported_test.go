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
	"github.com/openconfig/gnmi/errdiff"
	"github.com/openconfig/ygot/integration_tests/schemaops/ctestschema"
	"github.com/openconfig/ygot/internal/ytestutil"
	"github.com/openconfig/ygot/testutil"
	"github.com/openconfig/ygot/ygot"
	"github.com/openconfig/ygot/ytypes"
	"google.golang.org/protobuf/testing/protocmp"

	gnmipb "github.com/openconfig/gnmi/proto/gnmi"
)

// mustPath returns a string as a gNMI path, causing a panic if the string
// is invalid.
func mustPath(s string) *gnmipb.Path {
	p, err := ygot.StringToStructuredPath(s)
	if err != nil {
		panic(err)
	}
	return p
}

func TestDiffOrderedMap(t *testing.T) {
	// TODO: Test for uncompressed structs.
	tests := []struct {
		name          string
		inOrig, inMod *ctestschema.Device
		inOpts        []ygot.DiffOpt
		// skipTestUnmarshal determines whether the unmarshal test is skipped.
		skipTestUnmarshal bool
		// want is the expected output for Diff.
		want *gnmipb.Notification
		// wantAtomic and wantAtomic are the expected output for DiffWithAtomic.
		wantAtomic          []*gnmipb.Notification
		wantNonAtomic       *gnmipb.Notification
		wantErrSubstrAtomic string
	}{{
		name: "no change",
		inOrig: &ctestschema.Device{
			OrderedList: ctestschema.GetOrderedMap(t),
		},
		inMod: &ctestschema.Device{
			OrderedList: ctestschema.GetOrderedMap(t),
		},
		want:       &gnmipb.Notification{},
		wantAtomic: nil,
	}, {
		name:   "empty",
		inOrig: &ctestschema.Device{},
		inMod:  &ctestschema.Device{},
	}, {
		name: "empty-and-nil",
		inOrig: &ctestschema.Device{
			OrderedList: &ctestschema.OrderedList_OrderedMap{},
		},
		inMod:             &ctestschema.Device{},
		skipTestUnmarshal: true,
	}, {
		name:   "nested-ordered-map",
		inOrig: &ctestschema.Device{},
		inMod: &ctestschema.Device{
			OrderedList: ctestschema.GetNestedOrderedMap(t),
		},
		want: &gnmipb.Notification{
			Update: []*gnmipb.Update{{
				Path: mustPath(`/ordered-lists/ordered-list[key=foo]/config/key`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "foo"}},
			}, {
				Path: mustPath(`/ordered-lists/ordered-list[key=foo]/key`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "foo"}},
			}, {
				Path: mustPath(`/ordered-lists/ordered-list[key=foo]/config/value`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "foo-val"}},
			}, {
				Path: mustPath(`/ordered-lists/ordered-list[key=bar]/config/key`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "bar"}},
			}, {
				Path: mustPath(`/ordered-lists/ordered-list[key=bar]/key`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "bar"}},
			}, {
				Path: mustPath(`/ordered-lists/ordered-list[key=bar]/config/value`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "bar-val"}},
			}, {
				Path: mustPath(`/ordered-lists/ordered-list[key=foo]/ordered-lists/ordered-list[key=foo]/config/key`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "foo"}},
			}, {
				Path: mustPath(`/ordered-lists/ordered-list[key=foo]/ordered-lists/ordered-list[key=foo]/key`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "foo"}},
			}, {
				Path: mustPath(`/ordered-lists/ordered-list[key=foo]/ordered-lists/ordered-list[key=foo]/config/value`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "foo-val"}},
			}, {
				Path: mustPath(`/ordered-lists/ordered-list[key=foo]/ordered-lists/ordered-list[key=bar]/config/key`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "bar"}},
			}, {
				Path: mustPath(`/ordered-lists/ordered-list[key=foo]/ordered-lists/ordered-list[key=bar]/key`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "bar"}},
			}, {
				Path: mustPath(`/ordered-lists/ordered-list[key=foo]/ordered-lists/ordered-list[key=bar]/config/value`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "bar-val"}},
			}},
		},
		wantErrSubstrAtomic: "detected nested `ordered-by user` list, this is not supported",
	}, {
		name:   "empty-original-two-ordered-maps",
		inOrig: &ctestschema.Device{},
		inMod: &ctestschema.Device{
			OrderedList:           ctestschema.GetOrderedMap(t),
			OrderedMultikeyedList: ctestschema.GetOrderedMapMultikeyed(t),
		},
		want: &gnmipb.Notification{
			Update: []*gnmipb.Update{{
				Path: mustPath(`/ordered-lists/ordered-list[key=foo]/config/key`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "foo"}},
			}, {
				Path: mustPath(`/ordered-lists/ordered-list[key=foo]/key`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "foo"}},
			}, {
				Path: mustPath(`/ordered-lists/ordered-list[key=foo]/config/value`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "foo-val"}},
			}, {
				Path: mustPath(`/ordered-lists/ordered-list[key=bar]/config/key`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "bar"}},
			}, {
				Path: mustPath(`/ordered-lists/ordered-list[key=bar]/key`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "bar"}},
			}, {
				Path: mustPath(`/ordered-lists/ordered-list[key=bar]/config/value`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "bar-val"}},
			}, {
				Path: mustPath(`/ordered-multikeyed-lists/ordered-multikeyed-list[key1=foo][key2=42]/config/key1`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "foo"}},
			}, {
				Path: mustPath(`/ordered-multikeyed-lists/ordered-multikeyed-list[key1=foo][key2=42]/key1`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "foo"}},
			}, {
				Path: mustPath(`/ordered-multikeyed-lists/ordered-multikeyed-list[key1=foo][key2=42]/config/key2`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_UintVal{UintVal: 42}},
			}, {
				Path: mustPath(`/ordered-multikeyed-lists/ordered-multikeyed-list[key1=foo][key2=42]/key2`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_UintVal{UintVal: 42}},
			}, {
				Path: mustPath(`/ordered-multikeyed-lists/ordered-multikeyed-list[key1=foo][key2=42]/config/value`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "foo-val"}},
			}, {
				Path: mustPath(`/ordered-multikeyed-lists/ordered-multikeyed-list[key1=bar][key2=42]/config/key1`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "bar"}},
			}, {
				Path: mustPath(`/ordered-multikeyed-lists/ordered-multikeyed-list[key1=bar][key2=42]/key1`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "bar"}},
			}, {
				Path: mustPath(`/ordered-multikeyed-lists/ordered-multikeyed-list[key1=bar][key2=42]/config/key2`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_UintVal{UintVal: 42}},
			}, {
				Path: mustPath(`/ordered-multikeyed-lists/ordered-multikeyed-list[key1=bar][key2=42]/key2`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_UintVal{UintVal: 42}},
			}, {
				Path: mustPath(`/ordered-multikeyed-lists/ordered-multikeyed-list[key1=bar][key2=42]/config/value`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "bar-val"}},
			}, {
				Path: mustPath(`/ordered-multikeyed-lists/ordered-multikeyed-list[key1=baz][key2=84]/config/key1`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "baz"}},
			}, {
				Path: mustPath(`/ordered-multikeyed-lists/ordered-multikeyed-list[key1=baz][key2=84]/key1`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "baz"}},
			}, {
				Path: mustPath(`/ordered-multikeyed-lists/ordered-multikeyed-list[key1=baz][key2=84]/config/key2`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_UintVal{UintVal: 84}},
			}, {
				Path: mustPath(`/ordered-multikeyed-lists/ordered-multikeyed-list[key1=baz][key2=84]/key2`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_UintVal{UintVal: 84}},
			}, {
				Path: mustPath(`/ordered-multikeyed-lists/ordered-multikeyed-list[key1=baz][key2=84]/config/value`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "baz-val"}},
			}},
		},
		wantAtomic: []*gnmipb.Notification{{
			Prefix: mustPath(`ordered-lists`),
			Atomic: true,
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
			Prefix: mustPath(`ordered-multikeyed-lists`),
			Atomic: true,
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
		name: "empty-modified",
		inOrig: &ctestschema.Device{
			OrderedList: ctestschema.GetOrderedMap2(t),
		},
		inMod: &ctestschema.Device{},
		want: &gnmipb.Notification{
			Delete: []*gnmipb.Path{
				mustPath(`/ordered-lists/ordered-list[key=wee]/config/key`),
				mustPath(`/ordered-lists/ordered-list[key=wee]/key`),
				mustPath(`/ordered-lists/ordered-list[key=wee]/config/value`),
				mustPath(`/ordered-lists/ordered-list[key=woo]/config/key`),
				mustPath(`/ordered-lists/ordered-list[key=woo]/key`),
				mustPath(`/ordered-lists/ordered-list[key=woo]/config/value`),
			},
		},
		wantNonAtomic: &gnmipb.Notification{
			Delete: []*gnmipb.Path{
				mustPath(`/ordered-lists`),
			},
		},
	}, {
		name: "empty-modified-with-other-data-in-orig",
		inOrig: &ctestschema.Device{
			OrderedList: ctestschema.GetOrderedMap2(t),
			OtherData: &ctestschema.OtherData{
				Motd: ygot.String("venus-is-hazy-today"),
			},
		},
		inMod: &ctestschema.Device{},
		want: &gnmipb.Notification{
			Delete: []*gnmipb.Path{
				mustPath(`/other-data/config/motd`),
				mustPath(`/ordered-lists/ordered-list[key=wee]/config/key`),
				mustPath(`/ordered-lists/ordered-list[key=wee]/key`),
				mustPath(`/ordered-lists/ordered-list[key=wee]/config/value`),
				mustPath(`/ordered-lists/ordered-list[key=woo]/config/key`),
				mustPath(`/ordered-lists/ordered-list[key=woo]/key`),
				mustPath(`/ordered-lists/ordered-list[key=woo]/config/value`),
			},
		},
		wantNonAtomic: &gnmipb.Notification{
			Delete: []*gnmipb.Path{
				mustPath(`/other-data/config/motd`),
				mustPath(`/ordered-lists`),
			},
		},
	}, {
		name: "empty-modified-with-other-data",
		inOrig: &ctestschema.Device{
			OrderedList: ctestschema.GetOrderedMap2(t),
		},
		inMod: &ctestschema.Device{
			OtherData: &ctestschema.OtherData{
				Motd: ygot.String("venus-is-hazy-today"),
			},
		},
		want: &gnmipb.Notification{
			Update: []*gnmipb.Update{{
				Path: mustPath(`/other-data/config/motd`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "venus-is-hazy-today"}},
			}},
			Delete: []*gnmipb.Path{
				mustPath(`/ordered-lists/ordered-list[key=wee]/config/key`),
				mustPath(`/ordered-lists/ordered-list[key=wee]/key`),
				mustPath(`/ordered-lists/ordered-list[key=wee]/config/value`),
				mustPath(`/ordered-lists/ordered-list[key=woo]/config/key`),
				mustPath(`/ordered-lists/ordered-list[key=woo]/key`),
				mustPath(`/ordered-lists/ordered-list[key=woo]/config/value`),
			},
		},
		wantNonAtomic: &gnmipb.Notification{
			Update: []*gnmipb.Update{{
				Path: mustPath(`/other-data/config/motd`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "venus-is-hazy-today"}},
			}},
			Delete: []*gnmipb.Path{
				mustPath(`/ordered-lists`),
			},
		},
	}, {
		name: "disjoint-ordered-lists",
		inOrig: &ctestschema.Device{
			OrderedList: ctestschema.GetOrderedMap2(t),
		},
		inMod: &ctestschema.Device{
			OrderedList: ctestschema.GetOrderedMap(t),
		},
		want: &gnmipb.Notification{
			Update: []*gnmipb.Update{{
				Path: mustPath(`/ordered-lists/ordered-list[key=foo]/config/key`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "foo"}},
			}, {
				Path: mustPath(`/ordered-lists/ordered-list[key=foo]/key`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "foo"}},
			}, {
				Path: mustPath(`/ordered-lists/ordered-list[key=foo]/config/value`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "foo-val"}},
			}, {
				Path: mustPath(`/ordered-lists/ordered-list[key=bar]/config/key`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "bar"}},
			}, {
				Path: mustPath(`/ordered-lists/ordered-list[key=bar]/key`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "bar"}},
			}, {
				Path: mustPath(`/ordered-lists/ordered-list[key=bar]/config/value`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "bar-val"}},
			}},
			Delete: []*gnmipb.Path{
				mustPath(`/ordered-lists/ordered-list[key=wee]/config/key`),
				mustPath(`/ordered-lists/ordered-list[key=wee]/key`),
				mustPath(`/ordered-lists/ordered-list[key=wee]/config/value`),
				mustPath(`/ordered-lists/ordered-list[key=woo]/config/key`),
				mustPath(`/ordered-lists/ordered-list[key=woo]/key`),
				mustPath(`/ordered-lists/ordered-list[key=woo]/config/value`),
			},
		},
		wantAtomic: []*gnmipb.Notification{{
			Prefix: mustPath(`ordered-lists`),
			Atomic: true,
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
		name: "modified-is-subset-of-original",
		inOrig: &ctestschema.Device{
			OrderedList: ctestschema.GetOrderedMap2(t),
		},
		inMod: &ctestschema.Device{
			OrderedList: func() *ctestschema.OrderedList_OrderedMap {
				om := ctestschema.GetOrderedMap2(t)
				om.Delete("wee")
				return om
			}(),
		},
		want: &gnmipb.Notification{
			Delete: []*gnmipb.Path{
				mustPath(`/ordered-lists/ordered-list[key=wee]/config/key`),
				mustPath(`/ordered-lists/ordered-list[key=wee]/key`),
				mustPath(`/ordered-lists/ordered-list[key=wee]/config/value`),
			},
		},
		wantAtomic: []*gnmipb.Notification{{
			Prefix: mustPath(`ordered-lists`),
			Atomic: true,
			Update: []*gnmipb.Update{{
				Path: mustPath(`ordered-list[key=woo]/config/key`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "woo"}},
			}, {
				Path: mustPath(`ordered-list[key=woo]/key`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "woo"}},
			}, {
				Path: mustPath(`ordered-list[key=woo]/config/value`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "woo-val"}},
			}},
		}},
	}, {
		name: "modified-is-superset-of-original",
		inOrig: &ctestschema.Device{
			OrderedList: func() *ctestschema.OrderedList_OrderedMap {
				om := ctestschema.GetOrderedMap2(t)
				om.Delete("wee")
				return om
			}(),
		},
		inMod: &ctestschema.Device{
			OrderedList: ctestschema.GetOrderedMap2(t),
		},
		want: &gnmipb.Notification{
			Update: []*gnmipb.Update{{
				Path: mustPath(`/ordered-lists/ordered-list[key=wee]/config/key`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "wee"}},
			}, {
				Path: mustPath(`/ordered-lists/ordered-list[key=wee]/key`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "wee"}},
			}, {
				Path: mustPath(`/ordered-lists/ordered-list[key=wee]/config/value`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "wee-val"}},
			}},
		},
		wantAtomic: []*gnmipb.Notification{{
			Prefix: mustPath(`ordered-lists`),
			Atomic: true,
			Update: []*gnmipb.Update{{
				Path: mustPath(`ordered-list[key=wee]/config/key`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "wee"}},
			}, {
				Path: mustPath(`ordered-list[key=wee]/key`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "wee"}},
			}, {
				Path: mustPath(`ordered-list[key=wee]/config/value`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "wee-val"}},
			}, {
				Path: mustPath(`ordered-list[key=woo]/config/key`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "woo"}},
			}, {
				Path: mustPath(`ordered-list[key=woo]/key`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "woo"}},
			}, {
				Path: mustPath(`ordered-list[key=woo]/config/value`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "woo-val"}},
			}},
		}},
	}, {
		name: "modified-overlaps-original",
		inOrig: &ctestschema.Device{
			OrderedList: func() *ctestschema.OrderedList_OrderedMap {
				om := ctestschema.GetOrderedMapLonger(t)
				om.Delete("baz")
				return om
			}(),
		},
		inMod: &ctestschema.Device{
			OrderedList: func() *ctestschema.OrderedList_OrderedMap {
				om := ctestschema.GetOrderedMapLonger(t)
				om.Delete("bar")
				return om
			}(),
		},
		want: &gnmipb.Notification{
			Update: []*gnmipb.Update{{
				Path: mustPath(`/ordered-lists/ordered-list[key=baz]/config/key`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "baz"}},
			}, {
				Path: mustPath(`/ordered-lists/ordered-list[key=baz]/key`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "baz"}},
			}, {
				Path: mustPath(`/ordered-lists/ordered-list[key=baz]/config/value`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "baz-val"}},
			}},
			Delete: []*gnmipb.Path{
				mustPath(`/ordered-lists/ordered-list[key=bar]/config/key`),
				mustPath(`/ordered-lists/ordered-list[key=bar]/key`),
				mustPath(`/ordered-lists/ordered-list[key=bar]/config/value`),
			},
		},
		wantAtomic: []*gnmipb.Notification{{
			Prefix: mustPath(`ordered-lists`),
			Atomic: true,
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
				Path: mustPath(`ordered-list[key=baz]/config/key`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "baz"}},
			}, {
				Path: mustPath(`ordered-list[key=baz]/key`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "baz"}},
			}, {
				Path: mustPath(`ordered-list[key=baz]/config/value`),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{StringVal: "baz-val"}},
			}},
		}},
	}}

	for _, tt := range tests {
		t.Run("Diff/"+tt.name, func(t *testing.T) {
			got, err := ygot.Diff(tt.inOrig, tt.inMod, tt.inOpts...)
			if err != nil {
				t.Errorf("Diff: got unexpected error: %v", err)
				return
			}

			if err != nil {
				return
			}
			// To re-use the NotificationSetEqual helper, we put the want and got into
			// a slice.
			if !testutil.NotificationSetEqual([]*gnmipb.Notification{got}, []*gnmipb.Notification{tt.want}) {
				diff := cmp.Diff(got, tt.want, protocmp.Transform())
				t.Errorf("Diff: did not get expected Notification, diff(-got,+want):\n%s", diff)
			}
		})

		t.Run("DiffWithAtomic/"+tt.name, func(t *testing.T) {
			got, err := ygot.DiffWithAtomic(tt.inOrig, tt.inMod, tt.inOpts...)
			if diff := errdiff.Substring(err, tt.wantErrSubstrAtomic); diff != "" {
				t.Errorf("DiffWithAtomic: did not get expected error status, got: %s, want: %s", err, tt.wantErrSubstrAtomic)
				return
			}

			if tt.wantErrSubstrAtomic != "" {
				return
			}

			var gotNonAtomic *gnmipb.Notification
			gotAtomic := got
			if tt.wantNonAtomic != nil && len(got) > 0 {
				gotNonAtomic = got[len(got)-1]
				gotAtomic = got[:len(got)-1]
				if len(gotAtomic) == 0 {
					gotAtomic = nil
				}
			}

			if diff := cmp.Diff(gotAtomic, tt.wantAtomic, cmpopts.SortSlices(testutil.NotificationLess), protocmp.Transform()); diff != "" {
				t.Errorf("telemetry-atomic values of DiffWithAtomic: did not get expected Notification, diff(-got,+want):%s\n", diff)
			}
			// Avoid test flakiness by ignoring the update ordering. Required because
			// there is no order to the map of fields that are returned by the struct
			// output.
			if diff := cmp.Diff(gotNonAtomic, tt.wantNonAtomic, testutil.NotificationComparer()); diff != "" {
				t.Errorf("non-telemetry-atomic values of DiffWithAtomic: did not get expected Notification, diff(-got,+want):%s\n", diff)
			}

			if tt.skipTestUnmarshal {
				return
			}
			// Test that unmarshalling into original gets back to modified.
			schema, err := ctestschema.Schema()
			if err != nil {
				t.Fatal(err)
			}
			schema.Root = tt.inOrig
			if err := ytypes.UnmarshalNotifications(schema, got); err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tt.inOrig, tt.inMod, ytestutil.OrderedMapCmpOptions...); diff != "" {
				t.Errorf("Unmarshal diff into orig (-got, +want):\n%s", diff)
			}
		})
	}
}
