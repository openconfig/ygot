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

var validEmptySchema = &yang.Entry{Name: "empty-schema", Type: &yang.YangType{Kind: yang.Yempty}}

func TestValidateEmptySchema(t *testing.T) {
	tests := []struct {
		desc    string
		schema  *yang.Entry
		wantErr bool
	}{
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
		{
			desc:   "empty schema",
			schema: validEmptySchema,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := validateEmptySchema(tt.schema)
			if got, want := (err != nil), tt.wantErr; got != want {
				t.Errorf("%s: validateEmptySchema(%v) got error: %v, want error? %v", tt.desc, tt.schema, err, tt.wantErr)
			}
			testErrLog(t, tt.desc, err)
		})
	}
}

func TestValidateEmpty(t *testing.T) {
	tests := []struct {
		desc    string
		schema  *yang.Entry
		val     interface{}
		wantErr bool
	}{
		{
			desc:   "success",
			schema: validEmptySchema,
			val:    YANGEmpty(false),
		},
		{
			desc:    "bad schema",
			schema:  nil,
			val:     true,
			wantErr: true,
		},
		{
			desc:    "non empty type",
			schema:  validEmptySchema,
			val:     "",
			wantErr: true,
		},
		{
			desc:   "valid empty",
			schema: validEmptySchema,
			val:    YANGEmpty(true),
		},
		{
			desc:    "invalid empty",
			schema:  validEmptySchema,
			val:     true,
			wantErr: true,
		},
		{
			desc:    "invalid empty - wrong type",
			schema:  validEmptySchema,
			val:     "fish",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := validateEmpty(tt.schema, tt.val)
			if got, want := (err != nil), tt.wantErr; got != want {
				t.Errorf("%s: validateEmpty(%v) got error: %v, want error? %v", tt.desc, tt.val, err, tt.wantErr)
			}
			testErrLog(t, tt.desc, err)
		})
	}
}
