// Copyright 2020 Google Inc.
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
package ytypes

import (
	"testing"

	"github.com/openconfig/gnmi/errdiff"
)

func TestCheckDataTreeAgainstPaths(t *testing.T) {
	tests := []struct {
		desc             string
		inJSONTree       map[string]interface{}
		inDataPaths      [][]string
		wantErrSubstring string
	}{{
		desc: "no missing keys",
		inJSONTree: map[string]interface{}{
			"hello": "world",
		},
		inDataPaths: [][]string{{"hello"}},
	}, {
		desc:        "unpopulated fields",
		inJSONTree:  map[string]interface{}{},
		inDataPaths: [][]string{{"hello"}},
	}, {
		desc: "missing keys",
		inJSONTree: map[string]interface{}{
			"hello": "world",
		},
		wantErrSubstring: "JSON contains unexpected field hello",
	}, {
		desc: "missing multiple keys",
		inJSONTree: map[string]interface{}{
			"hello":   "world",
			"bonjour": "la-mode",
		},
		wantErrSubstring: `JSON contains unexpected field [bonjour hello]`,
	}, {
		desc: "hierarchical fields, populated",
		inJSONTree: map[string]interface{}{
			"config": map[string]interface{}{
				"description": "hello-world",
			},
		},
		inDataPaths: [][]string{
			{"config", "description"},
		},
	}, {
		desc: "JSON path longer than datapath",
		inJSONTree: map[string]interface{}{
			"config": map[string]interface{}{
				"description": "cracking-cheese-gromit",
			},
		},
		inDataPaths: [][]string{
			{"config"},
		},
	}, {
		desc: "JSON path shorter than the specified datapath",
		inJSONTree: map[string]interface{}{
			"config": map[string]interface{}{
				"name": "feathers-mcgraw",
			},
		},
		inDataPaths: [][]string{
			{"config", "name", "penguin"},
		},
		wantErrSubstring: `JSON contains unexpected field name`,
	}, {
		desc: "hierarchical fields, not populated",
		inJSONTree: map[string]interface{}{
			"config": map[string]interface{}{
				"duplex": "full",
			},
		},
		inDataPaths: [][]string{
			{"config", "fish"},
			{"fish"},
		},
		wantErrSubstring: "JSON contains unexpected field duplex",
	}, {
		desc: "nil inputs",
	}, {
		desc: "nil JSON",
		inDataPaths: [][]string{
			{"mtu"},
		},
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := checkDataTreeAgainstPaths(tt.inJSONTree, tt.inDataPaths)
			if diff := errdiff.Substring(err, tt.wantErrSubstring); diff != "" {
				t.Fatalf("did not get expected error, %s", diff)
			}

		})
	}
}
