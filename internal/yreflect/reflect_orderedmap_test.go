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
	"fmt"
	"reflect"
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
		inValue       any
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

type invalidOrderedMapType struct {
	ygot.GoOrderedMap
}

func TestGetOrderedMapElement(t *testing.T) {
	tests := []struct {
		desc          string
		inMap         ygot.GoOrderedMap
		inKey         any
		wantVal       any
		wantOk        bool
		wantErrSubstr string
	}{{
		desc:  "has",
		inMap: ctestschema.GetOrderedMap(t),
		inKey: "foo",
		wantVal: &ctestschema.OrderedList{
			Key:   ygot.String("foo"),
			Value: ygot.String("foo-val"),
		},
		wantOk: true,
	}, {
		desc:    "has-not",
		inMap:   ctestschema.GetOrderedMap(t),
		inKey:   "fooo",
		wantVal: (*ctestschema.OrderedList)(nil),
		wantOk:  false,
	}, {
		desc:          "invalid",
		inMap:         &invalidOrderedMapType{},
		wantErrSubstr: "did not find Get() method on type",
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got, gotOk, err := yreflect.GetOrderedMapElement(tt.inMap, reflect.ValueOf(tt.inKey))
			if diff := errdiff.Substring(err, tt.wantErrSubstr); diff != "" {
				t.Fatalf("InsertIntoMap: %s", diff)
			}
			if err != nil {
				return
			}

			if diff := cmp.Diff(tt.wantVal, got.Interface()); diff != "" {
				t.Errorf("(-want, +got):\n%s", diff)
			}
			if gotOk != tt.wantOk {
				t.Errorf("gotOk: %v, wantOk: %v", gotOk, tt.wantOk)
			}
		})
	}
}

func TestRangeOrderedMap(t *testing.T) {
	tests := []struct {
		desc          string
		inMap         ygot.GoOrderedMap
		inStopAfter   int
		wantKeys      []string
		wantValues    []string
		wantErrSubstr string
	}{{
		desc:       "full iteration",
		inMap:      ctestschema.GetOrderedMap(t),
		wantKeys:   []string{"foo", "bar"},
		wantValues: []string{"foo-val", "bar-val"},
	}, {
		desc:        "early termination after first element",
		inMap:       ctestschema.GetOrderedMap(t),
		inStopAfter: 1,
		wantKeys:    []string{"foo"},
		wantValues:  []string{"foo-val"},
	}, {
		desc:  "empty ordered map",
		inMap: &ctestschema.OrderedList_OrderedMap{},
	}, {
		desc:  "typed nil ordered map",
		inMap: (*ctestschema.OrderedList_OrderedMap)(nil),
	}, {
		desc:          "invalid ordered map",
		inMap:         &invalidOrderedMapType{},
		wantErrSubstr: "did not find Get() method on type",
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			var gotKeys, gotValues []string
			err := yreflect.RangeOrderedMap(tt.inMap, func(k, v reflect.Value) bool {
				gotKeys = append(gotKeys, k.Interface().(string))
				elem := v.Interface().(*ctestschema.OrderedList)
				gotValues = append(gotValues, *elem.Value)
				if tt.inStopAfter > 0 && len(gotKeys) >= tt.inStopAfter {
					return false
				}
				return true
			})
			if diff := errdiff.Substring(err, tt.wantErrSubstr); diff != "" {
				t.Fatalf("RangeOrderedMap(%v): %s", tt.inMap, diff)
			}
			if err != nil {
				return
			}
			if diff := cmp.Diff(gotKeys, tt.wantKeys); diff != "" {
				t.Errorf("RangeOrderedMap(%v) keys diff(-got,+want):\n%s", tt.inMap, diff)
			}
			if diff := cmp.Diff(gotValues, tt.wantValues); diff != "" {
				t.Errorf("RangeOrderedMap(%v) values diff(-got,+want):\n%s", tt.inMap, diff)
			}
		})
	}
}

func TestRangeOrderedMapPerformance(t *testing.T) {
	om := &ctestschema.OrderedList_OrderedMap{}
	const numEntries = 100
	for i := 0; i < numEntries; i++ {
		k := fmt.Sprintf("key-%03d", i)
		el, err := om.AppendNew(k)
		if err != nil {
			t.Fatalf("om.AppendNew(%q): got unexpected error: %v", k, err)
		}
		el.Value = ygot.String(fmt.Sprintf("val-%03d", i))
	}

	gotAllocs := testing.AllocsPerRun(50, func() {
		count := 0
		if err := yreflect.RangeOrderedMap(om, func(k, v reflect.Value) bool {
			if k.IsValid() && v.IsValid() {
				count++
			}
			return true
		}); err != nil || count != numEntries {
			t.Fatalf("RangeOrderedMap: got (%d, %v), want (%d, nil)", count, err, numEntries)
		}
	})

	legacyRange := func() error {
		getMethod, err := yreflect.MethodByName(reflect.ValueOf(om), "Get")
		if err != nil {
			return err
		}
		keys, err := yreflect.OrderedMapKeys(om)
		if err != nil {
			return err
		}
		for _, k := range keys {
			ret := getMethod.Call([]reflect.Value{k})
			_ = ret[0]
		}
		return nil
	}
	baselineAllocs := testing.AllocsPerRun(50, func() {
		if err := legacyRange(); err != nil {
			t.Fatalf("legacyRange: got unexpected error: %v", err)
		}
	})

	maxAllowedAllocs := baselineAllocs * 0.95
	if gotAllocs > maxAllowedAllocs {
		t.Fatalf("RangeOrderedMap allocated %.1f objects/run, want <= %.1f (at least 5%% fewer than legacy baseline %.1f)", gotAllocs, maxAllowedAllocs, baselineAllocs)
	}
}
