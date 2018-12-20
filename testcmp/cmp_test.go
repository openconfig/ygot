// Copyright 2018 Google Inc.
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
package testcmp

import (
	"fmt"
	"testing"

	"github.com/kylelemons/godebug/pretty"
	"github.com/openconfig/gnmi/errdiff"
	"github.com/openconfig/ygot/exampleoc"
	"github.com/openconfig/ygot/testutil"
	"github.com/openconfig/ygot/uexampleoc"
	"github.com/openconfig/ygot/ygot"
	"github.com/openconfig/ygot/ytypes"

	gnmipb "github.com/openconfig/gnmi/proto/gnmi"
)

func mustPath(s string) *gnmipb.Path {
	p, err := ygot.StringToStructuredPath(s)
	if err != nil {
		panic(fmt.Errorf("cannot converting string %s to path, got err: %v", s, err))
	}
	return p
}

func jsonIETF(s string) *gnmipb.TypedValue {
	return &gnmipb.TypedValue{
		Value: &gnmipb.TypedValue_JsonIetfVal{
			[]byte(s),
		},
	}
}

func TestGNMIUpdateComparer(t *testing.T) {
	commonSpec, err := uexampleoc.Schema()
	if err != nil {
		t.Fatalf("cannot get schema from package, %v", err)
	}

	tests := []struct {
		desc             string
		inA              *gnmipb.Update
		inB              *gnmipb.Update
		inSpec           *ytypes.Schema
		wantDiff         *gnmipb.Notification
		wantEqual        bool
		wantErrSubstring string
	}{{
		desc: "simple, updates equal",
		inA: &gnmipb.Update{
			Path: mustPath("/system/config"),
			Val:  jsonIETF(`{"hostname": "system-1"}`),
		},
		inB: &gnmipb.Update{
			Path: mustPath("/system/config"),
			Val:  jsonIETF(`{"hostname": "system-1"}`),
		},
		inSpec:    commonSpec,
		wantEqual: true,
	}, {
		desc: "simple, updates not equal",
		inA: &gnmipb.Update{
			Path: mustPath("/system/config"),
			Val:  jsonIETF(`{"hostname": "system-1"}`),
		},
		inB: &gnmipb.Update{
			Path: mustPath("/system/config"),
			Val:  jsonIETF(`{"hostname": "bus"}`),
		},
		inSpec:    commonSpec,
		wantEqual: false,
		wantDiff: &gnmipb.Notification{
			Update: []*gnmipb.Update{{
				Path: mustPath("/system/config/hostname"),
				Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{"bus"}},
			}},
		},
	}, {
		desc: "equal with ignored extra leaves",
		inA: &gnmipb.Update{
			Path: mustPath("/system/config"),
			Val:  jsonIETF(`{"hostname": "system-1"}`),
		},
		inB: &gnmipb.Update{
			Path: mustPath("/system/config"),
			Val:  jsonIETF(`{"hostname": "system-1", "vendor-ext": 42}`),
		},
		inSpec:    commonSpec,
		wantEqual: true,
	}, {
		desc: "error: bad JSON in A",
		inA: &gnmipb.Update{
			Path: mustPath("/system/config"),
			Val:  jsonIETF(`invalid`),
		},
		inB: &gnmipb.Update{
			Path: mustPath("/system/config"),
			Val:  jsonIETF(`{"hostname": "system-1"}`),
		},
		inSpec:           commonSpec,
		wantEqual:        false,
		wantErrSubstring: "cannot unmarshal JSON for struct A",
	}, {
		desc: "error: bad JSON in B",
		inA: &gnmipb.Update{
			Path: mustPath("/system/config"),
			Val:  jsonIETF(`{"hostname": "system-1"}`),
		},
		inB: &gnmipb.Update{
			Path: mustPath("/system/config"),
			Val:  jsonIETF(`zap`),
		},
		inSpec:           commonSpec,
		wantEqual:        false,
		wantErrSubstring: "cannot unmarshal JSON for struct B",
	}, {
		desc: "error: unmarshal leaf into A",
		inA: &gnmipb.Update{
			Path: mustPath("/system/config/hostname"),
			Val:  jsonIETF(`"fish"`),
		},
		inB: &gnmipb.Update{
			Path: mustPath("/system/config/hostname"),
			Val:  jsonIETF(`"fish"`),
		},
		inSpec:           commonSpec,
		wantEqual:        false,
		wantErrSubstring: "does not correspond to a struct",
	}, {
		desc: "error: unmarshal leaf into B",
		inA: &gnmipb.Update{
			Path: mustPath("/system/config"),
			Val:  jsonIETF(`{"hostname": "cheese"}`),
		},
		inB: &gnmipb.Update{
			Path: mustPath("/system/config/hostname"),
			Val:  jsonIETF(`"fish"`),
		},
		inSpec:           commonSpec,
		wantEqual:        false,
		wantErrSubstring: "does not correspond to a struct",
	}, {
		desc: "equal: not IETF JSON",
		inA: &gnmipb.Update{
			Path: mustPath("/system/config/hostname"),
			Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{"fish"}},
		},
		inB: &gnmipb.Update{
			Path: mustPath("/system/config/hostname"),
			Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{"fish"}},
		},
		inSpec:    commonSpec,
		wantEqual: true,
	}, {
		desc: "not equal: different paths",
		inA: &gnmipb.Update{
			Path: mustPath("/system/config/domain-name"),
			Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{"fish"}},
		},
		inB: &gnmipb.Update{
			Path: mustPath("/system/config/hostname"),
			Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{"fish"}},
		},
		inSpec:    commonSpec,
		wantEqual: false,
	}, {
		desc: "equal: nil values",
		inA: &gnmipb.Update{
			Path: mustPath("/system/config/domain-name"),
		},
		inB: &gnmipb.Update{
			Path: mustPath("/system/config/domain-name"),
		},
		inSpec:    commonSpec,
		wantEqual: true,
	}, {
		desc: "not equal: one value nil",
		inA: &gnmipb.Update{
			Path: mustPath("/system/config/domain-name"),
			Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{"fish"}},
		},
		inB: &gnmipb.Update{
			Path: mustPath("/system/config/domain-name"),
		},
		inSpec:    commonSpec,
		wantEqual: false,
	}, {
		desc: "not equal: different types",
		inA: &gnmipb.Update{
			Path: mustPath("/system/config/hostname"),
			Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_StringVal{"fish"}},
		},
		inB: &gnmipb.Update{
			Path: mustPath("/system/config/hostname"),
			Val:  &gnmipb.TypedValue{Value: &gnmipb.TypedValue_UintVal{42}},
		},
		inSpec:    commonSpec,
		wantEqual: false,
	}, {
		desc: "error: invalid path in A",
		inA: &gnmipb.Update{
			Path: mustPath("/system/config/fish"),
			Val:  jsonIETF(`"value"`),
		},
		inB: &gnmipb.Update{
			Path: mustPath("/system/config/hostname"),
			Val:  jsonIETF(`"value"`),
		},
		inSpec:           commonSpec,
		wantErrSubstring: `cannot retrieve struct for path elem:<name:"system" > elem:<name:"config" > elem:<name:"fish" >`,
	}, {
		desc: "error: invalid path in B",
		inA: &gnmipb.Update{
			Path: mustPath("/system/config"),
			Val:  jsonIETF(`{"hostname": "system-1"}`),
		},
		inB: &gnmipb.Update{
			Path: mustPath("/system/config/chips"),
			Val:  jsonIETF(`"value"`),
		},
		inSpec:           commonSpec,
		wantErrSubstring: `cannot retrieve struct for path elem:<name:"system" > elem:<name:"config" > elem:<name:"chips" >`,
	}, {
		desc:             "error: nil spec",
		inA:              &gnmipb.Update{},
		inB:              &gnmipb.Update{},
		inSpec:           nil,
		wantErrSubstring: "JSON specification is not valid",
	}, {
		desc:             "error: invalid spec",
		inA:              &gnmipb.Update{},
		inB:              &gnmipb.Update{},
		inSpec:           &ytypes.Schema{},
		wantErrSubstring: "JSON specification is not valid",
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			gotDiff, gotEqual, err := GNMIUpdateComparer(tt.inA, tt.inB, tt.inSpec)
			if diff := errdiff.Substring(err, tt.wantErrSubstring); diff != "" {
				t.Fatalf("did not get expected error, %s", diff)
			}

			if err != nil {
				return
			}

			if gotEqual != tt.wantEqual {
				t.Errorf("did not get expected equal status, got: %v, want: %v", gotEqual, tt.wantEqual)
			}

			if diff := pretty.Compare(gotDiff, tt.wantDiff); diff != "" {
				t.Errorf("did not get expected diff, diff(-got,+want):\n%s", diff)
			}
		})
	}
}

func TestIntegration(t *testing.T) {
	// This test checks that the GNMIUpdateComparer can be used within the testutil
	// framework.

	ocComparer, err := UpdateComparer(exampleoc.Schema)
	if err != nil {
		t.Fatalf("cannot get exampleoc comparer, %v", err)
	}

	uocComparer, err := UpdateComparer(uexampleoc.Schema)
	if err != nil {
		t.Fatalf("cannot get uexampleoc comparer, %v", err)
	}

	_, _ = ocComparer, uocComparer

	compare := func(a, b interface{}, opts ...testutil.ComparerOpt) bool {
		switch am := a.(type) {
		case []*gnmipb.Notification:
			bm, ok := b.([]*gnmipb.Notification)
			if !ok {
				t.Fatalf("invalid input %T != %T", a, b)
			}
			return testutil.NotificationSetEqual(am, bm, opts...)
		case *gnmipb.Notification:
			bm, ok := b.(*gnmipb.Notification)
			if !ok {
				t.Fatalf("invalid input %T != %T", a, b)
			}
			return testutil.NotificationSetEqual([]*gnmipb.Notification{am}, []*gnmipb.Notification{bm}, opts...)
		case *gnmipb.GetResponse:
			bm, ok := b.(*gnmipb.GetResponse)
			if !ok {
				t.Fatalf("invalid input %T != %T", a, b)
			}
			return testutil.GetResponseEqual(am, bm, opts...)
		}
		t.Fatalf("unhandled type %T", a)
		return false
	}

	tests := []struct {
		desc   string
		inA    interface{}
		inB    interface{}
		inOpts []testutil.ComparerOpt
		want   bool
	}{{
		desc: "simple, no options",
		inA: &gnmipb.Notification{
			Timestamp: 42,
		},
		inB: &gnmipb.Notification{
			Timestamp: 42,
		},
		want: true,
	}, {
		desc: "simple option",
		inA: &gnmipb.Notification{
			Timestamp: 42,
		},
		inB: &gnmipb.Notification{
			Timestamp: 84,
		},
		inOpts: []testutil.ComparerOpt{testutil.IgnoreTimestamp{}},
		want:   true,
	}, {
		desc: "Notification Set with OC comparer",
		inA: []*gnmipb.Notification{{
			Timestamp: 42,
			Update: []*gnmipb.Update{{
				Path: mustPath("/system"),
				Val: jsonIETF(`{
					"config": {
						"hostname": "box42.pop42"
					}
				}`),
			}},
		}},
		inB: []*gnmipb.Notification{{
			Timestamp: 42,
			Update: []*gnmipb.Update{{
				Path: mustPath("/system"),
				Val: jsonIETF(`{
					"config": {
						"hostname": "box42.pop42",
						"extra-field": "IGNORE"
					}
				}`),
			}},
		}},
		inOpts: []testutil.ComparerOpt{
			ocComparer,
		},
		want: true,
	}, {
		desc: "GetResponse equal with uoc comparer",
		inA: &gnmipb.GetResponse{
			Notification: []*gnmipb.Notification{{
				Timestamp: 0,
				Update: []*gnmipb.Update{{
					Path: mustPath("/system/config"),
					Val:  jsonIETF(`{"hostname": "dev1"}`),
				}},
			}},
		},
		inB: &gnmipb.GetResponse{
			Notification: []*gnmipb.Notification{{
				Timestamp: 0,
				Update: []*gnmipb.Update{{
					Path: mustPath("/system"),
					Val: jsonIETF(`{
						"config": {
							"hostname": "dev1",
							"ignored-val": "dev2"
						}
					}`),
				}},
			}},
		},
		inOpts: []testutil.ComparerOpt{
			uocComparer,
		},
		want: true,
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if got := compare(tt.inA, tt.inB, tt.inOpts...); got != tt.want {
				t.Fatalf("did not get expected equality status, got: %v, want: %v", got, tt.want)
			}
		})
	}
}
