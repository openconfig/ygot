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
)

func TestFlattenOCJSON(t *testing.T) {
	// TODO: increase coverage of error code paths.
	tests := []struct {
		desc   string
		inJSON string
		want   map[string]interface{}
	}{{
		desc: "basic",
		inJSON: `
{
  "openconfig-network-instance:config": {
    "description": "VRF RED",
    "enabled": true,
    "enabled-address-families": [
      "openconfig-types:IPV4",
      "openconfig-types:IPV6"
    ],
    "protocols": {
      "protocol": [
        {
	  "name": "STATIC",
	  "id": 1,
	  "config": {
	    "name": "STATIC",
	    "id": 1,
	    "enabled": true
	  }
        },
        {
	  "id": 2,
	  "name": "IS-IS",
	  "config": {
	    "id": 2,
	    "name": "IS-IS",
	    "enabled": false
	  }
        },
        {
	  "name": "BGP",
	  "id": 3,
	  "config": {
	    "id": 3,
	    "name": "BGP",
	    "enabled": false
	  }
        }
      ]
    },
    "name": "RED",
    "type": "openconfig-network-instance-types:L3VRF",
    "leaf-list": []
    },
  "openconfig-network-instance:name": "RED"
}
`,
		want: map[string]interface{}{
			"/openconfig-network-instance:config/description":                                          "VRF RED",
			"/openconfig-network-instance:config/enabled":                                              true,
			"/openconfig-network-instance:config/enabled-address-families":                             []interface{}{"openconfig-types:IPV4", "openconfig-types:IPV6"},
			"/openconfig-network-instance:config/name":                                                 "RED",
			"/openconfig-network-instance:config/protocols/protocol[id=1][name=STATIC]/config/enabled": true,
			"/openconfig-network-instance:config/protocols/protocol[id=1][name=STATIC]/config/id":      float64(1),
			"/openconfig-network-instance:config/protocols/protocol[id=1][name=STATIC]/config/name":    "STATIC",
			"/openconfig-network-instance:config/protocols/protocol[id=1][name=STATIC]/id":             float64(1),
			"/openconfig-network-instance:config/protocols/protocol[id=1][name=STATIC]/name":           "STATIC",
			"/openconfig-network-instance:config/protocols/protocol[id=2][name=IS-IS]/config/enabled":  false,
			"/openconfig-network-instance:config/protocols/protocol[id=2][name=IS-IS]/config/id":       float64(2),
			"/openconfig-network-instance:config/protocols/protocol[id=2][name=IS-IS]/config/name":     "IS-IS",
			"/openconfig-network-instance:config/protocols/protocol[id=2][name=IS-IS]/id":              float64(2),
			"/openconfig-network-instance:config/protocols/protocol[id=2][name=IS-IS]/name":            "IS-IS",
			"/openconfig-network-instance:config/protocols/protocol[id=3][name=BGP]/config/enabled":    false,
			"/openconfig-network-instance:config/protocols/protocol[id=3][name=BGP]/config/id":         float64(3),
			"/openconfig-network-instance:config/protocols/protocol[id=3][name=BGP]/config/name":       "BGP",
			"/openconfig-network-instance:config/protocols/protocol[id=3][name=BGP]/id":                float64(3),
			"/openconfig-network-instance:config/protocols/protocol[id=3][name=BGP]/name":              "BGP",
			"/openconfig-network-instance:config/type":                                                 "openconfig-network-instance-types:L3VRF",
			"/openconfig-network-instance:config/leaf-list":                                            []interface{}{},
			"/openconfig-network-instance:name":                                                        "RED",
		},
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got, err := flattenOCJSON([]byte(tt.inJSON))
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("flattenOCJSON: (-want, +got):\n%s", diff)
			}
		})
	}
}
