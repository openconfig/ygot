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

var validBitsetSchema = mapToBitsetSchema("valid-bitset-schema", map[string]int64{"name1": 0, "name2": 1, "name3": 2})

func mapToBitsetSchema(schemaName string, bm map[string]int64) *yang.Entry {
	b := yang.NewBitfield()
	for k, v := range bm {
		b.Set(k, v)
	}
	return &yang.Entry{
		Name: schemaName,
		Type: &yang.YangType{
			Kind: yang.Ybits,
			Bit:  b,
		},
	}
}

func TestValidateBitsetSchema(t *testing.T) {
	tests := []struct {
		desc    string
		schema  *yang.Entry
		wantErr bool
	}{
		{
			desc:   "success",
			schema: validBitsetSchema,
		},
		{
			desc:    "nil schema",
			schema:  nil,
			wantErr: true,
		},
		{
			desc:    "nil schema type",
			schema:  &yang.Entry{Name: "empty schema", Type: nil},
			wantErr: true,
		},
		{
			desc:    "bad schema type",
			schema:  &yang.Entry{Name: "empty schema", Type: &yang.YangType{Kind: yang.Yempty}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := validateBitsetSchema(tt.schema)
			if got, want := (err != nil), tt.wantErr; got != want {
				t.Errorf("%s: validateBitsetSchema(%v) got error: %v, want error? %v", tt.desc, tt.schema, err, tt.wantErr)
			}
			testErrLog(t, tt.desc, err)
		})
	}
}

func TestValidateBitset(t *testing.T) {
	tests := []struct {
		desc    string
		schema  *yang.Entry
		val     interface{}
		wantErr bool
	}{
		{
			desc:   "success",
			schema: validBitsetSchema,
			val:    "name1 name2",
		},
		{
			desc:    "bad schema",
			schema:  nil,
			val:     "",
			wantErr: true,
		},
		{
			desc:    "non bitset type",
			schema:  validBitsetSchema,
			val:     "",
			wantErr: true,
		},
		{
			desc:    "nonexistent bit name",
			schema:  validBitsetSchema,
			val:     "name0 name2",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := validateBitset(tt.schema, tt.val)
			if got, want := (err != nil), tt.wantErr; got != want {
				t.Errorf("%s: validateBitset(%v) got error: %v, want error? %v", tt.desc, tt.val, err, tt.wantErr)
			}
			testErrLog(t, tt.desc, err)
		})
	}
}

func TestValidateBitsetSlice(t *testing.T) {
	tests := []struct {
		desc    string
		schema  *yang.Entry
		val     interface{}
		wantErr bool
	}{
		{
			desc:   "success",
			schema: validBitsetSchema,
			val:    []string{"name1 name2", "name1"},
		},
		{
			desc:    "non []string",
			schema:  validBitsetSchema,
			val:     []int32{1, 2},
			wantErr: true,
		},
		{
			desc:    "invalid element",
			schema:  validBitsetSchema,
			val:     []string{"name0 name2", "name1"},
			wantErr: true,
		},
		{
			desc:    "duplicate element",
			schema:  validBitsetSchema,
			val:     []string{"name1 name2", "name1", "name1"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := validateBitsetSlice(tt.schema, tt.val)
			if got, want := (err != nil), tt.wantErr; got != want {
				t.Errorf("%s: validateBitset(%v) got error: %v, want error? %v", tt.desc, tt.val, err, tt.wantErr)
			}
			testErrLog(t, tt.desc, err)
		})
	}
}
