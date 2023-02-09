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

package gnmidiff

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/openconfig/ygot/exampleoc"
	"github.com/openconfig/ygot/ygot"
	"github.com/openconfig/ygot/ytypes"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
)

func TestSetRequestDiffFormat(t *testing.T) {
	tests := []struct {
		desc             string
		inSetRequestDiff SetRequestIntentDiff
		inFormat         Format
		want             string
	}{{
		desc: "compact output",
		inSetRequestDiff: SetRequestIntentDiff{
			AOnlyDeletes: map[string]struct{}{
				"/interfaces/interface[name=eth1]": {},
			},
			BOnlyDeletes: map[string]struct{}{
				"/interfaces/interface[name=eth2]": {},
			},
			CommonDeletes: map[string]struct{}{
				"/interfaces/interface[name=eth0]": {},
			},
			AOnlyUpdates: map[string]interface{}{
				"/interfaces/interface[name=eth1]/name":        "eth1",
				"/interfaces/interface[name=eth1]/config/name": "eth1",
			},
			BOnlyUpdates: map[string]interface{}{
				"/interfaces/interface[name=eth2]/name":              "eth2",
				"/interfaces/interface[name=eth2]/config/name":       "eth2",
				"/interfaces/interface[name=eth0]/state/transceiver": "FDM",
			},
			CommonUpdates: map[string]interface{}{
				"/interfaces/interface[name=eth0]/name":                                                  "eth0",
				"/interfaces/interface[name=eth0]/config/name":                                           "eth0",
				"/interfaces/interface[name=eth0]/config/description":                                    "I am an eth port",
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/config/index":      float64(0),
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/index":             float64(0),
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/state/oper-status": "TESTING",
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/config/enabled":    true,
				"/interfaces/interface[name=eth2]/state/transceiver":                                     "FDM",
			},
			MismatchedUpdates: map[string]MismatchedUpdate{
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/state/logical": {
					A: false,
					B: true,
				},
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/state/name": {
					A: "foo",
					B: "bar",
				},
			},
		},
		inFormat: Format{},
		want: `SetRequestIntentDiff(-A, +B):
-------- deletes --------
- /interfaces/interface[name=eth1]: deleted
+ /interfaces/interface[name=eth2]: deleted
-------- updates --------
- /interfaces/interface[name=eth1]/config/name: "eth1"
- /interfaces/interface[name=eth1]/name: "eth1"
+ /interfaces/interface[name=eth0]/state/transceiver: "FDM"
+ /interfaces/interface[name=eth2]/config/name: "eth2"
+ /interfaces/interface[name=eth2]/name: "eth2"
m /interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/state/logical:
  - false
  + true
m /interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/state/name:
  - "foo"
  + "bar"
`,
	}, {
		desc: "full output",
		inSetRequestDiff: SetRequestIntentDiff{
			AOnlyDeletes: map[string]struct{}{
				"/interfaces/interface[name=eth1]": {},
			},
			BOnlyDeletes: map[string]struct{}{
				"/interfaces/interface[name=eth2]": {},
			},
			CommonDeletes: map[string]struct{}{
				"/interfaces/interface[name=eth0]": {},
			},
			AOnlyUpdates: map[string]interface{}{
				"/interfaces/interface[name=eth1]/name":        "eth1",
				"/interfaces/interface[name=eth1]/config/name": "eth1",
			},
			BOnlyUpdates: map[string]interface{}{
				"/interfaces/interface[name=eth2]/name":              "eth2",
				"/interfaces/interface[name=eth2]/config/name":       "eth2",
				"/interfaces/interface[name=eth0]/state/transceiver": "FDM",
			},
			CommonUpdates: map[string]interface{}{
				"/interfaces/interface[name=eth0]/name":                                                  "eth0",
				"/interfaces/interface[name=eth0]/config/name":                                           "eth0",
				"/interfaces/interface[name=eth0]/config/description":                                    "I am an eth port",
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/config/index":      float64(0),
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/index":             float64(0),
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/state/oper-status": "TESTING",
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/config/enabled":    true,
				"/interfaces/interface[name=eth2]/state/transceiver":                                     "FDM",
			},
			MismatchedUpdates: map[string]MismatchedUpdate{
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/state/logical": {
					A: false,
					B: true,
				},
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/state/name": {
					A: "foo",
					B: "bar",
				},
			},
		},
		inFormat: Format{
			Full: true,
		},
		want: `SetRequestIntentDiff(-A, +B):
-------- deletes --------
  /interfaces/interface[name=eth0]: deleted
- /interfaces/interface[name=eth1]: deleted
+ /interfaces/interface[name=eth2]: deleted
-------- updates --------
  /interfaces/interface[name=eth0]/config/description: "I am an eth port"
  /interfaces/interface[name=eth0]/config/name: "eth0"
  /interfaces/interface[name=eth0]/name: "eth0"
  /interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/config/enabled: true
  /interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/config/index: 0
  /interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/index: 0
  /interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/state/oper-status: "TESTING"
  /interfaces/interface[name=eth2]/state/transceiver: "FDM"
- /interfaces/interface[name=eth1]/config/name: "eth1"
- /interfaces/interface[name=eth1]/name: "eth1"
+ /interfaces/interface[name=eth0]/state/transceiver: "FDM"
+ /interfaces/interface[name=eth2]/config/name: "eth2"
+ /interfaces/interface[name=eth2]/name: "eth2"
m /interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/state/logical:
  - false
  + true
m /interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/state/name:
  - "foo"
  + "bar"
`,
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got := tt.inSetRequestDiff.Format(tt.inFormat)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("SetRequestIntentDiff.Format (-want, +got):\n%s", diff)
			}
		})
	}
}

func TestDiffSetRequest(t *testing.T) {
	tests := []struct {
		desc               string
		inA                *gpb.SetRequest
		inB                *gpb.SetRequest
		wantSetRequestDiff SetRequestIntentDiff
		wantErr            bool
	}{{
		desc: "exactly the same",
		inA: &gpb.SetRequest{
			Replace: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Name: ygot.String("eth0")}),
			}},
			Update: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Subinterface: map[uint32]*exampleoc.Interface_Subinterface{0: {Index: ygot.Uint32(0), OperStatus: exampleoc.Interface_OperStatus_TESTING}}, Description: ygot.String("I am an eth port")}),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/config/enabled"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_BoolVal{BoolVal: true}},
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/state/transceiver"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "FDM"}},
			}},
		},
		inB: &gpb.SetRequest{
			Replace: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Name: ygot.String("eth0")}),
			}},
			Update: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Subinterface: map[uint32]*exampleoc.Interface_Subinterface{0: {Index: ygot.Uint32(0), OperStatus: exampleoc.Interface_OperStatus_TESTING}}, Description: ygot.String("I am an eth port")}),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/config/enabled"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_BoolVal{BoolVal: true}},
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/state/transceiver"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "FDM"}},
			}},
		},
		wantSetRequestDiff: SetRequestIntentDiff{
			AOnlyDeletes: map[string]struct{}{},
			BOnlyDeletes: map[string]struct{}{},
			CommonDeletes: map[string]struct{}{
				"/interfaces/interface[name=eth0]": {},
			},
			AOnlyUpdates: map[string]interface{}{},
			BOnlyUpdates: map[string]interface{}{},
			CommonUpdates: map[string]interface{}{
				"/interfaces/interface[name=eth0]/name":                                                  "eth0",
				"/interfaces/interface[name=eth0]/config/name":                                           "eth0",
				"/interfaces/interface[name=eth0]/config/description":                                    "I am an eth port",
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/config/index":      float64(0),
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/index":             float64(0),
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/state/oper-status": "TESTING",
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/config/enabled":    true,
				"/interfaces/interface[name=eth0]/state/transceiver":                                     "FDM",
			},
			MismatchedUpdates: map[string]MismatchedUpdate{},
		},
	}, {
		desc: "not same but same intent",
		inA: &gpb.SetRequest{
			Replace: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Name: ygot.String("eth0")}),
			}},
			Update: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Subinterface: map[uint32]*exampleoc.Interface_Subinterface{0: {Index: ygot.Uint32(0), OperStatus: exampleoc.Interface_OperStatus_TESTING}}, Description: ygot.String("I am an eth port")}),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/config/enabled"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_BoolVal{BoolVal: true}},
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/state/transceiver"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "FDM"}},
			}},
		},
		inB: &gpb.SetRequest{
			Replace: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Name: ygot.String("eth0"), Transceiver: ygot.String("FDM")}),
			}},
			Update: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Subinterface: map[uint32]*exampleoc.Interface_Subinterface{0: {Index: ygot.Uint32(0), OperStatus: exampleoc.Interface_OperStatus_TESTING, Enabled: ygot.Bool(true)}}}),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Description: ygot.String("I am an eth port")}),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/state/transceiver"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "FDM"}},
			}},
		},
		wantSetRequestDiff: SetRequestIntentDiff{
			AOnlyDeletes: map[string]struct{}{},
			BOnlyDeletes: map[string]struct{}{},
			CommonDeletes: map[string]struct{}{
				"/interfaces/interface[name=eth0]": {},
			},
			AOnlyUpdates: map[string]interface{}{},
			BOnlyUpdates: map[string]interface{}{},
			CommonUpdates: map[string]interface{}{
				"/interfaces/interface[name=eth0]/name":                                                  "eth0",
				"/interfaces/interface[name=eth0]/config/name":                                           "eth0",
				"/interfaces/interface[name=eth0]/config/description":                                    "I am an eth port",
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/config/index":      float64(0),
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/index":             float64(0),
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/state/oper-status": "TESTING",
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/config/enabled":    true,
				"/interfaces/interface[name=eth0]/state/transceiver":                                     "FDM",
			},
			MismatchedUpdates: map[string]MismatchedUpdate{},
		},
	}, {
		desc: "SetRequest B has conflicts",
		inA: &gpb.SetRequest{
			Replace: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Name: ygot.String("eth0")}),
			}},
			Update: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Subinterface: map[uint32]*exampleoc.Interface_Subinterface{0: {Index: ygot.Uint32(0), OperStatus: exampleoc.Interface_OperStatus_TESTING}}, Description: ygot.String("I am an eth port")}),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/config/enabled"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_BoolVal{BoolVal: true}},
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/state/transceiver"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "FDM"}},
			}},
		},
		inB: &gpb.SetRequest{
			Replace: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Name: ygot.String("eth0"), Transceiver: ygot.String("FDM")}),
			}},
			Update: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Subinterface: map[uint32]*exampleoc.Interface_Subinterface{0: {Index: ygot.Uint32(0), OperStatus: exampleoc.Interface_OperStatus_TESTING, Enabled: ygot.Bool(true)}}}),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Description: ygot.String("I am an eth port")}),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/config/enabled"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_BoolVal{BoolVal: true}},
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/state/transceiver"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "TDM"}},
			}},
		},
		wantErr: true,
	}, {
		desc: "only A",
		inA: &gpb.SetRequest{
			Replace: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Name: ygot.String("eth0")}),
			}},
			Update: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Subinterface: map[uint32]*exampleoc.Interface_Subinterface{0: {Index: ygot.Uint32(0), OperStatus: exampleoc.Interface_OperStatus_TESTING}}, Description: ygot.String("I am an eth port")}),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/config/enabled"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_BoolVal{BoolVal: true}},
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/state/transceiver"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "FDM"}},
			}},
		},
		inB: &gpb.SetRequest{},
		wantSetRequestDiff: SetRequestIntentDiff{
			AOnlyDeletes: map[string]struct{}{
				"/interfaces/interface[name=eth0]": {},
			},
			BOnlyDeletes:  map[string]struct{}{},
			CommonDeletes: map[string]struct{}{},
			AOnlyUpdates: map[string]interface{}{
				"/interfaces/interface[name=eth0]/name":                                                  "eth0",
				"/interfaces/interface[name=eth0]/config/name":                                           "eth0",
				"/interfaces/interface[name=eth0]/config/description":                                    "I am an eth port",
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/config/index":      float64(0),
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/index":             float64(0),
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/state/oper-status": "TESTING",
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/config/enabled":    true,
				"/interfaces/interface[name=eth0]/state/transceiver":                                     "FDM",
			},
			BOnlyUpdates:      map[string]interface{}{},
			CommonUpdates:     map[string]interface{}{},
			MismatchedUpdates: map[string]MismatchedUpdate{},
		},
	}, {
		desc: "only B",
		inB: &gpb.SetRequest{
			Replace: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Name: ygot.String("eth0"), Transceiver: ygot.String("FDM")}),
			}},
			Update: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Subinterface: map[uint32]*exampleoc.Interface_Subinterface{0: {Index: ygot.Uint32(0), OperStatus: exampleoc.Interface_OperStatus_TESTING, Enabled: ygot.Bool(true)}}}),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Description: ygot.String("I am an eth port")}),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/state/transceiver"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "FDM"}},
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/config/mtu"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_UintVal{UintVal: 1500}},
			}},
		},
		wantSetRequestDiff: SetRequestIntentDiff{
			AOnlyDeletes: map[string]struct{}{},
			BOnlyDeletes: map[string]struct{}{
				"/interfaces/interface[name=eth0]": {},
			},
			CommonDeletes: map[string]struct{}{},
			AOnlyUpdates:  map[string]interface{}{},
			BOnlyUpdates: map[string]interface{}{
				"/interfaces/interface[name=eth0]/name":                                                  "eth0",
				"/interfaces/interface[name=eth0]/config/name":                                           "eth0",
				"/interfaces/interface[name=eth0]/config/description":                                    "I am an eth port",
				"/interfaces/interface[name=eth0]/config/mtu":                                            float64(1500),
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/config/index":      float64(0),
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/index":             float64(0),
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/state/oper-status": "TESTING",
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/config/enabled":    true,
				"/interfaces/interface[name=eth0]/state/transceiver":                                     "FDM",
			},
			CommonUpdates:     map[string]interface{}{},
			MismatchedUpdates: map[string]MismatchedUpdate{},
		},
	}, {
		desc: "mismatch",
		inA: &gpb.SetRequest{
			Replace: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Name: ygot.String("eth0")}),
			}},
			Update: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Subinterface: map[uint32]*exampleoc.Interface_Subinterface{0: {Index: ygot.Uint32(0), OperStatus: exampleoc.Interface_OperStatus_DORMANT}}, Description: ygot.String("I am an ethernet port")}),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/config/enabled"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_BoolVal{BoolVal: false}},
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/state/transceiver"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "TDM"}},
			}},
		},
		inB: &gpb.SetRequest{
			Replace: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Name: ygot.String("eth0"), Transceiver: ygot.String("FDM")}),
			}},
			Update: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Subinterface: map[uint32]*exampleoc.Interface_Subinterface{0: {Index: ygot.Uint32(0), OperStatus: exampleoc.Interface_OperStatus_TESTING, Enabled: ygot.Bool(true)}}}),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Description: ygot.String("I am an eth port")}),
			}},
		},
		wantSetRequestDiff: SetRequestIntentDiff{
			AOnlyDeletes: map[string]struct{}{},
			BOnlyDeletes: map[string]struct{}{},
			CommonDeletes: map[string]struct{}{
				"/interfaces/interface[name=eth0]": {},
			},
			AOnlyUpdates: map[string]interface{}{},
			BOnlyUpdates: map[string]interface{}{},
			CommonUpdates: map[string]interface{}{
				"/interfaces/interface[name=eth0]/name":                                             "eth0",
				"/interfaces/interface[name=eth0]/config/name":                                      "eth0",
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/config/index": float64(0),
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/index":        float64(0),
			},
			MismatchedUpdates: map[string]MismatchedUpdate{
				"/interfaces/interface[name=eth0]/config/description": {
					A: "I am an ethernet port",
					B: "I am an eth port",
				},
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/state/oper-status": {
					A: "DORMANT",
					B: "TESTING",
				},
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/config/enabled": {
					A: false,
					B: true,
				},
				"/interfaces/interface[name=eth0]/state/transceiver": {
					A: "TDM",
					B: "FDM",
				},
			},
		},
	}, {
		desc: "not the same with every difference case",
		inA: &gpb.SetRequest{
			Replace: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth1]"),
				Val:  must7951(&exampleoc.Interface{Name: ygot.String("eth1")}),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Name: ygot.String("eth0")}),
			}},
			Update: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Subinterface: map[uint32]*exampleoc.Interface_Subinterface{0: {Index: ygot.Uint32(0), OperStatus: exampleoc.Interface_OperStatus_TESTING}}, Description: ygot.String("I am an eth port")}),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/config/enabled"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_BoolVal{BoolVal: true}},
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth2]/state/transceiver"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "FDM"}},
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/state/logical"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_BoolVal{BoolVal: false}},
			}},
		},
		inB: &gpb.SetRequest{
			Replace: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Name: ygot.String("eth0"), Transceiver: ygot.String("FDM")}),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth2]"),
				Val:  must7951(&exampleoc.Interface{Name: ygot.String("eth2"), Transceiver: ygot.String("FDM"), Mtu: ygot.Uint16(1500)}),
			}},
			Update: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Subinterface: map[uint32]*exampleoc.Interface_Subinterface{0: {Index: ygot.Uint32(0), OperStatus: exampleoc.Interface_OperStatus_TESTING, Enabled: ygot.Bool(true)}}}),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Description: ygot.String("I am an eth port")}),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/state/logical"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_BoolVal{BoolVal: true}},
			}},
		},
		wantSetRequestDiff: SetRequestIntentDiff{
			AOnlyDeletes: map[string]struct{}{
				"/interfaces/interface[name=eth1]": {},
			},
			BOnlyDeletes: map[string]struct{}{
				"/interfaces/interface[name=eth2]": {},
			},
			CommonDeletes: map[string]struct{}{
				"/interfaces/interface[name=eth0]": {},
			},
			AOnlyUpdates: map[string]interface{}{
				"/interfaces/interface[name=eth1]/name":        "eth1",
				"/interfaces/interface[name=eth1]/config/name": "eth1",
			},
			BOnlyUpdates: map[string]interface{}{
				"/interfaces/interface[name=eth2]/name":              "eth2",
				"/interfaces/interface[name=eth2]/config/name":       "eth2",
				"/interfaces/interface[name=eth2]/config/mtu":        float64(1500),
				"/interfaces/interface[name=eth0]/state/transceiver": "FDM",
			},
			CommonUpdates: map[string]interface{}{
				"/interfaces/interface[name=eth0]/name":                                                  "eth0",
				"/interfaces/interface[name=eth0]/config/name":                                           "eth0",
				"/interfaces/interface[name=eth0]/config/description":                                    "I am an eth port",
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/config/index":      float64(0),
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/index":             float64(0),
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/state/oper-status": "TESTING",
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/config/enabled":    true,
				"/interfaces/interface[name=eth2]/state/transceiver":                                     "FDM",
			},
			MismatchedUpdates: map[string]MismatchedUpdate{
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/state/logical": {
					A: false,
					B: true,
				},
			},
		},
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			for _, withNewSchemaFn := range []bool{false, true} {
				var inNewSchemaFn func() (*ytypes.Schema, error)
				if withNewSchemaFn {
					inNewSchemaFn = exampleoc.Schema
				}
				t.Run(fmt.Sprintf("withNewSchemaFn-%v", withNewSchemaFn), func(t *testing.T) {
					got, err := DiffSetRequest(tt.inA, tt.inB, inNewSchemaFn)
					if (err != nil) != tt.wantErr {
						t.Fatalf("got error: %v, want error: %v", err, tt.wantErr)
					}
					if diff := cmp.Diff(tt.wantSetRequestDiff, got); diff != "" {
						t.Errorf("DiffSetRequest (-want, +got):\n%s", diff)
					}
				})
			}
		})
	}
}

// must7951 calls Marshal7951 to create a JSON_IETF TypedValue.
func must7951(v interface{}) *gpb.TypedValue {
	b, err := ygot.Marshal7951(v, &ygot.RFC7951JSONConfig{AppendModuleName: true})
	if err != nil {
		panic(err)
	}
	return &gpb.TypedValue{
		Value: &gpb.TypedValue_JsonIetfVal{
			JsonIetfVal: b,
		},
	}
}

func TestMinimalSetRequestIntent(t *testing.T) {
	tests := []struct {
		desc         string
		inSetRequest *gpb.SetRequest
		wantIntent   setRequestIntent
		wantErr      bool
	}{{
		desc: "delete",
		inSetRequest: &gpb.SetRequest{
			Delete: []*gpb.Path{
				ygot.MustStringToPath("/interfaces/interface[name=eth0]/config/description"),
			},
		},
		wantErr: true,
	}, {
		desc:         "empty",
		inSetRequest: &gpb.SetRequest{},
		wantIntent: setRequestIntent{
			Deletes: map[string]struct{}{},
			Updates: map[string]interface{}{},
		},
	}, {
		desc: "conflicting leaf replace",
		inSetRequest: &gpb.SetRequest{
			Replace: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/config/description"),
				Val:  must7951(ygot.String("I am an eth port")),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/config/description"),
				Val:  must7951(ygot.String("I am an eth port again")),
			}},
		},
		wantErr: true,
	}, {
		desc: "conflicting leaf replace and update",
		inSetRequest: &gpb.SetRequest{
			Replace: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/config/description"),
				Val:  must7951(ygot.String("I am an eth port")),
			}},
			Update: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/config/description"),
				Val:  must7951(ygot.String("I am an eth port again")),
			}},
		},
		wantErr: true,
	}, {
		desc: "conflicting leaf update",
		inSetRequest: &gpb.SetRequest{
			Update: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/config/description"),
				Val:  must7951(ygot.String("I am an eth port")),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/config/description"),
				Val:  must7951(ygot.String("I am an eth port again")),
			}},
		},
		wantErr: true,
	}, {
		desc: "conflicting leaf update due to prefix match",
		inSetRequest: &gpb.SetRequest{
			Update: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/config/description"),
				Val:  must7951(ygot.String("I am an eth port")),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/config/description/desc"),
				Val:  must7951(ygot.String("I am an eth port again")),
			}},
		},
		wantErr: true,
	}, {
		desc: "conflicting replaces due to common path prefix",
		inSetRequest: &gpb.SetRequest{
			Replace: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Name: ygot.String("eth0")}),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/config/description"),
				Val:  must7951(ygot.String("I am an eth port")),
			}},
		},
		wantErr: true,
	}, {
		desc: "conflicting replaces due to common path prefix different order",
		inSetRequest: &gpb.SetRequest{
			Replace: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/config/description"),
				Val:  must7951(ygot.String("I am an eth port")),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Name: ygot.String("eth0")}),
			}},
		},
		wantErr: true,
	}, {
		desc: "leaf replace",
		inSetRequest: &gpb.SetRequest{
			Replace: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/config/description"),
				Val:  must7951(ygot.String("I am an eth port")),
			}},
		},
		wantIntent: setRequestIntent{
			Deletes: map[string]struct{}{},
			Updates: map[string]interface{}{
				"/interfaces/interface[name=eth0]/config/description": "I am an eth port",
			},
		},
	}, {
		desc: "list container replace",
		inSetRequest: &gpb.SetRequest{
			Replace: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Name: ygot.String("eth0")}),
			}},
		},
		wantIntent: setRequestIntent{
			Deletes: map[string]struct{}{
				"/interfaces/interface[name=eth0]": {},
			},
			Updates: map[string]interface{}{
				"/interfaces/interface[name=eth0]/name":        "eth0",
				"/interfaces/interface[name=eth0]/config/name": "eth0",
			},
		},
	}, {
		desc: "leaf update",
		inSetRequest: &gpb.SetRequest{
			Update: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/config/description"),
				Val:  must7951(ygot.String("I am an eth port")),
			}},
		},
		wantIntent: setRequestIntent{
			Deletes: map[string]struct{}{},
			Updates: map[string]interface{}{
				"/interfaces/interface[name=eth0]/config/description": "I am an eth port",
			},
		},
	}, {
		desc: "list container update",
		inSetRequest: &gpb.SetRequest{
			Update: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Name: ygot.String("eth0")}),
			}},
		},
		wantIntent: setRequestIntent{
			Deletes: map[string]struct{}{},
			Updates: map[string]interface{}{
				"/interfaces/interface[name=eth0]/name":        "eth0",
				"/interfaces/interface[name=eth0]/config/name": "eth0",
			},
		},
	}, {
		desc: "conflicting nested update with previous replace",
		inSetRequest: &gpb.SetRequest{
			Replace: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Name: ygot.String("eth0"), Description: ygot.String("hello")}),
			}},
			Update: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/config/description"),
				Val:  must7951(ygot.String("I am an eth port")),
			}},
		},
		wantErr: true,
	}, {
		desc: "conflicting nested update with previous update",
		inSetRequest: &gpb.SetRequest{
			Update: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Name: ygot.String("eth0"), Description: ygot.String("hello")}),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/config/description"),
				Val:  must7951(ygot.String("I am an eth port")),
			}},
		},
		wantErr: true,
	}, {
		desc: "conflicting nested update with previous update in different order",
		inSetRequest: &gpb.SetRequest{
			Update: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/config/description"),
				Val:  must7951(ygot.String("I am an eth port")),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Name: ygot.String("eth0"), Description: ygot.String("hello")}),
			}},
		},
		wantErr: true,
	}, {
		desc: "non-conflicting nested update with previous replace",
		inSetRequest: &gpb.SetRequest{
			Replace: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Name: ygot.String("eth0")}),
			}},
			Update: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/config/description"),
				Val:  must7951(ygot.String("I am an eth port")),
			}},
		},
		wantIntent: setRequestIntent{
			Deletes: map[string]struct{}{
				"/interfaces/interface[name=eth0]": {},
			},
			Updates: map[string]interface{}{
				"/interfaces/interface[name=eth0]/name":               "eth0",
				"/interfaces/interface[name=eth0]/config/name":        "eth0",
				"/interfaces/interface[name=eth0]/config/description": "I am an eth port",
			},
		},
	}, {
		desc: "non-conflicting nested update with previous update",
		inSetRequest: &gpb.SetRequest{
			Update: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Name: ygot.String("eth0")}),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/config/description"),
				Val:  must7951(ygot.String("I am an eth port")),
			}},
		},
		wantIntent: setRequestIntent{
			Deletes: map[string]struct{}{},
			Updates: map[string]interface{}{
				"/interfaces/interface[name=eth0]/name":               "eth0",
				"/interfaces/interface[name=eth0]/config/name":        "eth0",
				"/interfaces/interface[name=eth0]/config/description": "I am an eth port",
			},
		},
	}, {
		desc: "non-conflicting nested update with previous update in different order",
		inSetRequest: &gpb.SetRequest{
			Update: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/config/description"),
				Val:  must7951(ygot.String("I am an eth port")),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Name: ygot.String("eth0")}),
			}},
		},
		wantIntent: setRequestIntent{
			Deletes: map[string]struct{}{},
			Updates: map[string]interface{}{
				"/interfaces/interface[name=eth0]/name":               "eth0",
				"/interfaces/interface[name=eth0]/config/name":        "eth0",
				"/interfaces/interface[name=eth0]/config/description": "I am an eth port",
			},
		},
	}, {
		desc: "non-conflicting update with previous replace with same parent update path",
		inSetRequest: &gpb.SetRequest{
			Replace: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Mtu: ygot.Uint16(1500)}),
			}},
			Update: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Description: ygot.String("I am an eth port")}),
			}},
		},
		wantIntent: setRequestIntent{
			Deletes: map[string]struct{}{
				"/interfaces/interface[name=eth0]": {},
			},
			Updates: map[string]interface{}{
				"/interfaces/interface[name=eth0]/config/mtu":         float64(1500),
				"/interfaces/interface[name=eth0]/config/description": "I am an eth port",
			},
		},
	}, {
		desc: "non-conflicting update with previous update with same parent update path",
		inSetRequest: &gpb.SetRequest{
			Update: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Name: ygot.String("eth0")}),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Description: ygot.String("I am an eth port")}),
			}},
		},
		wantIntent: setRequestIntent{
			Deletes: map[string]struct{}{},
			Updates: map[string]interface{}{
				"/interfaces/interface[name=eth0]/name":               "eth0",
				"/interfaces/interface[name=eth0]/config/name":        "eth0",
				"/interfaces/interface[name=eth0]/config/description": "I am an eth port",
			},
		},
	}, {
		desc: "nested list",
		inSetRequest: &gpb.SetRequest{
			Replace: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Name: ygot.String("eth0")}),
			}},
			Update: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Subinterface: map[uint32]*exampleoc.Interface_Subinterface{0: {Index: ygot.Uint32(0), OperStatus: exampleoc.Interface_OperStatus_TESTING}}, Description: ygot.String("I am an eth port")}),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/config/enabled"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_BoolVal{BoolVal: true}},
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/state/transceiver"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "FDM"}},
			}},
		},
		wantIntent: setRequestIntent{
			Deletes: map[string]struct{}{
				"/interfaces/interface[name=eth0]": {},
			},
			Updates: map[string]interface{}{
				"/interfaces/interface[name=eth0]/name":                                                  "eth0",
				"/interfaces/interface[name=eth0]/config/name":                                           "eth0",
				"/interfaces/interface[name=eth0]/config/description":                                    "I am an eth port",
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/config/index":      float64(0),
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/index":             float64(0),
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/state/oper-status": "TESTING",
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/config/enabled":    true,
				"/interfaces/interface[name=eth0]/state/transceiver":                                     "FDM",
			},
		},
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			for _, withNewSchemaFn := range []bool{false, true} {
				var inNewSchemaFn func() (*ytypes.Schema, error)
				if withNewSchemaFn {
					inNewSchemaFn = exampleoc.Schema
				}
				t.Run(fmt.Sprintf("withNewSchemaFn-%v", withNewSchemaFn), func(t *testing.T) {
					got, err := minimalSetRequestIntent(tt.inSetRequest, inNewSchemaFn)
					if (err != nil) != tt.wantErr {
						t.Fatalf("got error: %v, want error: %v", err, tt.wantErr)
					}
					if diff := cmp.Diff(tt.wantIntent, got); diff != "" {
						t.Errorf("minimalSetRequestIntent (-want, +got):\n%s", diff)
					}
				})
			}
		})
	}
}
