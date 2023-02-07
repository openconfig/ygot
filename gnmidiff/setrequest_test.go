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
	"testing"

	"github.com/google/go-cmp/cmp"
	gpb "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/ygot/ygot"
)

// must7951 calls Marshal7951 to create a JSON_IETF TypedValue.
func must7951(v interface{}, args ...ygot.Marshal7951Arg) *gpb.TypedValue {
	b, err := ygot.Marshal7951(v, args...)
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
	// FIXME: test with different leaf types, TypedValues, and nested structs or lists.
	// FIXME: test with namespaced JSON.
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
				Val:  must7951(&Interface{Name: ygot.String("eth0")}),
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
				Val:  must7951(&Interface{Name: ygot.String("eth0")}),
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
				Val:  must7951(&Interface{Name: ygot.String("eth0")}),
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
				Val:  must7951(&Interface{Name: ygot.String("eth0")}),
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
				Val:  must7951(&Interface{Name: ygot.String("eth0"), Description: ygot.String("hello")}),
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
				Val:  must7951(&Interface{Name: ygot.String("eth0"), Description: ygot.String("hello")}),
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
				Val:  must7951(&Interface{Name: ygot.String("eth0"), Description: ygot.String("hello")}),
			}},
		},
		wantErr: true,
	}, {
		desc: "non-conflicting nested update with previous replace",
		inSetRequest: &gpb.SetRequest{
			Replace: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&Interface{Name: ygot.String("eth0")}),
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
				Val:  must7951(&Interface{Name: ygot.String("eth0")}),
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
				Val:  must7951(&Interface{Name: ygot.String("eth0")}),
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
				Val:  must7951(&Interface{Name: ygot.String("eth0")}),
			}},
			Update: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&Interface{Description: ygot.String("I am an eth port")}),
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
		desc: "non-conflicting update with previous update with same parent update path",
		inSetRequest: &gpb.SetRequest{
			Update: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&Interface{Name: ygot.String("eth0")}),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&Interface{Description: ygot.String("I am an eth port")}),
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
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got, err := minimalSetRequestIntent(tt.inSetRequest)
			if (err != nil) != tt.wantErr {
				t.Fatalf("got error: %v, want error: %v", err, tt.wantErr)
			}
			if diff := cmp.Diff(tt.wantIntent, got); diff != "" {
				t.Errorf("minimalSetRequestIntent (-want, +got):\n%s", diff)
			}
		})
	}
}
