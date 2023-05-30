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

package util_test

import (
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/openconfig/gnmi/errdiff"
	"github.com/openconfig/ygot/integration_tests/schemaops/ctestschema"
	"github.com/openconfig/ygot/util"
)

// to ptr conversion utility functions
func toInt8Ptr(i int8) *int8 { return &i }

func TestIsValueNil(t *testing.T) {
	if !util.IsValueNil(nil) {
		t.Error("got util.IsValueNil(nil) false, want true")
	}
	if !util.IsValueNil((*int)(nil)) {
		t.Error("got util.IsValueNil(ptr) false, want true")
	}
	if !util.IsValueNil((map[int]int)(nil)) {
		t.Error("got util.IsValueNil(map) false, want true")
	}
	if !util.IsValueNil(([]int)(nil)) {
		t.Error("got util.IsValueNil(slice) false, want true")
	}
	if !util.IsValueNil((interface{})(nil)) {
		t.Error("got util.IsValueNil(interface) false, want true")
	}
	if !util.IsValueNil((*ctestschema.OrderedList_OrderedMap)(nil)) {
		t.Error("got util.IsValueNil(interface) false, want true")
	}

	if util.IsValueNil(toInt8Ptr(42)) {
		t.Error("got util.IsValueNil(ptr) true, want false")
	}
	if util.IsValueNil(map[int]int{42: 42}) {
		t.Error("got util.IsValueNil(map) true, want false")
	}
	if util.IsValueNil([]int{1, 2, 3}) {
		t.Error("got util.IsValueNil(slice) true, want false")
	}
	if util.IsValueNil((interface{})(42)) {
		t.Error("got util.IsValueNil(interface) true, want false")
	}
	if util.IsValueNil(ctestschema.GetOrderedMap(t)) {
		t.Error("got util.IsValueNil(interface) true, want false")
	}
}

func TestIsValueNilOrDefault(t *testing.T) {
	// want true tests
	if !util.IsValueNilOrDefault(nil) {
		t.Error("got util.IsValueNilOrDefault(nil) false, want true")
	}
	if !util.IsValueNilOrDefault((*int)(nil)) {
		t.Error("got util.IsValueNilOrDefault(ptr) false, want true")
	}
	if !util.IsValueNilOrDefault((map[int]int)(nil)) {
		t.Error("got util.IsValueNilOrDefault(map) false, want true")
	}
	if !util.IsValueNilOrDefault(([]int)(nil)) {
		t.Error("got util.IsValueNilOrDefault(slice) false, want true")
	}
	if !util.IsValueNilOrDefault((interface{})(nil)) {
		t.Error("got util.IsValueNilOrDefault(interface) false, want true")
	}
	if !util.IsValueNilOrDefault(int(0)) {
		t.Error("got util.IsValueNilOrDefault(int(0)) false, want true")
	}
	if !util.IsValueNilOrDefault("") {
		t.Error("got util.IsValueNilOrDefault(\"\") false, want true")
	}
	if !util.IsValueNilOrDefault(false) {
		t.Error("got util.IsValueNilOrDefault(false) false, want true")
	}

	// want false tests
	i := 32
	ip := &i
	if util.IsValueNilOrDefault(&ip) {
		t.Error("got util.IsValueNilOrDefault(ptr to ptr) true, want false")
	}
	if util.IsValueNilOrDefault([]int{}) {
		t.Error("got util.IsValueNilOrDefault([]int{}) true, want false")
	}
	if util.IsValueNilOrDefault(ctestschema.GetOrderedMap(t)) {
		t.Error("got util.IsValueNilOrDefault(false) false, want true")
	}
}

func TestInsertIntoMap(t *testing.T) {
	tests := []struct {
		desc          string
		inMap         interface{}
		inKey         interface{}
		inValue       interface{}
		wantMap       interface{}
		wantErrSubstr string
	}{{
		desc:    "regular map",
		inMap:   map[int]string{42: "forty two", 43: "forty three"},
		inKey:   44,
		inValue: "forty four",
		wantMap: map[int]string{42: "forty two", 43: "forty three", 44: "forty four"},
	}, {
		desc:          "bad map",
		inMap:         &struct{}{},
		inKey:         44,
		inValue:       "forty four",
		wantErrSubstr: `InsertIntoMap parent type is *struct {}, must be map`,
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := util.InsertIntoMap(tt.inMap, tt.inKey, tt.inValue)
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

func TestInitializeStructField(t *testing.T) {
	type testStruct struct {
		// Following two fields exist to exercise
		// initializing pointer fields
		IPtr      *int
		SPtr      *string
		StructPtr *struct {
			IPtr *int
			SPtr *string
		}
		OMPtr *ctestschema.OrderedList_OrderedMap
		// Following field exists to exercise
		// initializing composite fields
		MPtr map[string]int
		// Following fields exist to exercise
		// skipping initializing a slice and
		// non pointer field
		SlPtr []string
		I     int
	}

	tests := []struct {
		f          string
		skip       bool
		isLeafType bool
	}{
		{f: "IPtr", isLeafType: true},
		{f: "SPtr", isLeafType: true},
		{f: "StructPtr"},
		{f: "OMPtr"},
		{f: "MPtr"},
		{f: "SlPtr", skip: true},
		{f: "I", skip: true},
	}

	for _, initLeaf := range []bool{false, true} {
		for _, tt := range tests {
			i := &testStruct{}
			v := reflect.ValueOf(i)
			if util.IsValuePtr(v) {
				v = v.Elem()
			}
			fv := v.FieldByName(tt.f)
			err := util.InitializeStructField(i, tt.f, initLeaf)
			if err != nil {
				t.Errorf("got %v, want no error", err)
			}
			skip := tt.skip || (!initLeaf && tt.isLeafType)
			switch {
			case !skip && fv.IsNil():
				t.Errorf("got nil, want initialized field value: %q", tt.f)
			case skip && !util.IsValuePtr(fv) && !fv.IsZero():
				t.Errorf("got initialized non-pointer field value %q, want zero value", tt.f)
			case skip && util.IsValuePtr(fv) && !fv.IsNil():
				t.Errorf("got initialized field value %q, want nil", tt.f)
			}
		}
	}
}

func TestInsertIntoSlice(t *testing.T) {
	tests := []struct {
		desc          string
		inSlice       any
		inValue       any
		wantSlice     any
		wantErrSubstr string
	}{{
		desc:      "basic",
		inSlice:   &[]int{42, 43},
		inValue:   44,
		wantSlice: &[]int{42, 43, 44},
	}, {
		desc:          "bad input slice",
		inSlice:       &struct{}{},
		inValue:       44,
		wantErrSubstr: `InsertIntoSlice parent type is *struct {}, must be slice ptr`,
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := util.InsertIntoSlice(tt.inSlice, tt.inValue)
			if diff := errdiff.Substring(err, tt.wantErrSubstr); diff != "" {
				t.Fatalf("InsertIntoMap: %s", diff)
			}
			if err != nil {
				return
			}

			if diff := cmp.Diff(tt.wantSlice, tt.inSlice); diff != "" {
				t.Errorf("(-want, +got):\n%s", diff)
			}
		})
	}
}
