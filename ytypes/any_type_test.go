// Copyright 2017 Google Inc.
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

	"github.com/openconfig/goyang/pkg/yang"
)

func TestValidateAny(t *testing.T) {
	tests := []struct {
		desc    string
		schema  *yang.Entry
		val     interface{}
		wantErr bool
	}{{
		desc:   "test success",
		schema: &yang.Entry{},
		val:    []interface{}{},
	}, {
		desc:   "test string success",
		schema: &yang.Entry{},
		val:    "xxx",
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := validateAny(tt.schema, tt.val)
			if got, want := (err != nil), tt.wantErr; got != want {
				t.Errorf("%s: validateAny(%v) got error: %v, want error? %v", tt.desc, tt.val, err, tt.wantErr)
			}
			testErrLog(t, tt.desc, err)
		})
	}
}

func TestValidateAnySlice(t *testing.T) {
	tests := []struct {
		desc    string
		schema  *yang.Entry
		val     interface{}
		wantErr bool
	}{{
		desc:   "test success",
		schema: &yang.Entry{},
		val:    []interface{}{},
	}, {
		desc:   "test string success",
		schema: &yang.Entry{},
		val:    "xxx",
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := validateAnySlice(tt.schema, tt.val)
			if got, want := (err != nil), tt.wantErr; got != want {
				t.Errorf("%s: validateAny(%v) got error: %v, want error: %v", tt.desc, tt.val, err, tt.wantErr)
			}
			testErrLog(t, tt.desc, err)
		})
	}
}
