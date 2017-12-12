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

var validBoolSchema = &yang.Entry{Name: "valid-bool-schema", Type: &yang.YangType{Kind: yang.Ybool}}

func TestValidateBoolSchema(t *testing.T) {
	tests := []struct {
		desc    string
		schema  *yang.Entry
		wantErr bool
	}{
		{
			desc:   "success",
			schema: validBoolSchema,
		},
		{
			desc:    "nil schema",
			schema:  nil,
			wantErr: true,
		},
		{
			desc:    "nil schema type",
			schema:  &yang.Entry{Name: "nil-type-schema", Type: nil},
			wantErr: true,
		},
		{
			desc:    "bad schema type",
			schema:  &yang.Entry{Name: "string-type-schema", Type: &yang.YangType{Kind: yang.Ystring}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := validateBoolSchema(tt.schema)
			if got, want := (err != nil), tt.wantErr; got != want {
				t.Errorf("%s: validateBoolSchema(%v) got error: %v, want error? %v", tt.desc, tt.schema, err, tt.wantErr)
			}
			testErrLog(t, tt.desc, err)
		})
	}
}

func TestValidateBool(t *testing.T) {
	tests := []struct {
		desc    string
		schema  *yang.Entry
		val     interface{}
		wantErr bool
	}{
		{
			desc:   "success",
			schema: validBoolSchema,
			val:    true,
		},
		{
			desc:    "bad schema",
			schema:  nil,
			val:     true,
			wantErr: true,
		},
		{
			desc:    "non bool type",
			schema:  validBoolSchema,
			val:     "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := validateBool(tt.schema, tt.val)
			if got, want := (err != nil), tt.wantErr; got != want {
				t.Errorf("%s: validateBool(%v) got error: %v, want error? %v", tt.desc, tt.val, err, tt.wantErr)
			}
			testErrLog(t, tt.desc, err)
		})
	}
}

func TestValidateSliceBoolType(t *testing.T) {
	tests := []struct {
		desc    string
		schema  *yang.Entry
		val     interface{}
		wantErr bool
	}{
		{
			desc:   "success",
			schema: validBoolSchema,
			val:    []bool{true, false},
		},
		{
			desc:    "bad schema",
			schema:  nil,
			val:     []bool{true},
			wantErr: true,
		},
		{
			desc:    "non []bool",
			schema:  validBoolSchema,
			val:     []string{"abc", "def"},
			wantErr: true,
		},
		{
			desc:    "duplicate element",
			schema:  validBoolSchema,
			val:     []bool{true, true},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := validateBoolSlice(tt.schema, tt.val)
			if got, want := (err != nil), tt.wantErr; got != want {
				t.Errorf("%s: validateBool(%v) got error: %v, want error? %v", tt.desc, tt.val, err, tt.wantErr)
			}
			testErrLog(t, tt.desc, err)
		})
	}
}
