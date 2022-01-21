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
	"reflect"
	"testing"

	"github.com/openconfig/goyang/pkg/yang"
)

var validBinarySchema = yrangeToBinarySchema("schema-with-range-2-to-10", yang.YRange{Min: yang.FromInt(2), Max: yang.FromInt(10)})

func yrangeToBinarySchema(schemaName string, yr yang.YRange) *yang.Entry {
	return &yang.Entry{
		Name: schemaName,
		Type: &yang.YangType{Kind: yang.Ybinary, Length: yang.YangRange{yr}}}
}

func TestValidateBinarySchema(t *testing.T) {
	tests := []struct {
		desc    string
		schema  *yang.Entry
		wantErr bool
	}{
		{
			desc:   "success",
			schema: validBinarySchema,
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
			schema:  &yang.Entry{Name: "empty-type-schema", Type: &yang.YangType{Kind: yang.Yempty}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := validateBinarySchema(tt.schema)
			if got, want := (err != nil), tt.wantErr; got != want {
				t.Errorf("%s: validateBinarySchema(%v) got error: %v, want error? %v", tt.desc, tt.schema, err, tt.wantErr)
			}
			testErrLog(t, tt.desc, err)
		})
	}
}

func TestValidateBinarySchemaRanges(t *testing.T) {
	tests := []struct {
		desc       string
		length     yang.YRange
		schemaName string
		wantErr    bool
	}{
		{
			desc:       "success",
			schemaName: "range-2-to-10",
			length:     yang.YRange{Min: yang.FromInt(2), Max: yang.FromInt(10)},
		},
		{
			desc:       "unset min length success",
			schemaName: "range-10-or-less",
			length:     yang.YRange{Min: yang.FromInt(0), Max: yang.FromInt(10)},
		},
		{
			desc:       "unset max length success",
			schemaName: "range-2-or-more",
			length:     yang.YRange{Min: yang.FromInt(2), Max: yang.FromUint(maxUint64)},
		}, {
			schemaName: "range-any",
			desc:       "unset min and max length success",
			length:     yang.Uint64Range[0],
		},
		{
			desc:       "bad length range",
			schemaName: "bad-range",
			length:     yang.YRange{Min: yang.FromInt(20), Max: yang.FromInt(10)},
			wantErr:    true,
		},
		{
			desc:       "negative min length",
			schemaName: "negative-min-length",
			length:     yang.YRange{Min: yang.FromInt(-1), Max: yang.FromUint(maxUint64)},
			wantErr:    true,
		},
		{
			desc:       "negative max length",
			schemaName: "negative-max-length",
			length:     yang.YRange{Min: yang.FromInt(0), Max: yang.FromInt(-1)},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := validateBinarySchema(yrangeToBinarySchema(tt.schemaName, tt.length))
			if got, want := (err != nil), tt.wantErr; got != want {
				t.Errorf("%s: validateBinarySchema(%v) got error: %v, want error? %v, ", tt.desc, tt.length, err, tt.wantErr)
			}
			testErrLog(t, tt.desc, err)
		})
	}
}

func TestValidateBinary(t *testing.T) {
	tests := []struct {
		desc       string
		length     yang.YRange
		schemaName string
		val        interface{}
		wantErr    bool
	}{
		{
			desc:       "success",
			length:     yang.YRange{Min: yang.FromInt(2), Max: yang.FromInt(10)},
			schemaName: "range-2-to-10",
			val:        Binary("a09Z+/"),
		},
		{
			desc:       "success - nil binary",
			length:     yang.YRange{Min: yang.FromInt(2), Max: yang.FromInt(10)},
			schemaName: "range-2-to-10",
			val:        Binary(nil),
		},
		{
			desc:       "success - nil",
			length:     yang.YRange{Min: yang.FromInt(2), Max: yang.FromInt(10)},
			schemaName: "range-2-to-10",
			val:        nil,
		},
		{
			desc:       "bad schema",
			length:     yang.YRange{Min: yang.FromInt(20), Max: yang.FromInt(10)},
			schemaName: "bad-range",
			val:        Binary("aaa"),
			wantErr:    true,
		},
		{
			desc:       "non binary type",
			length:     yang.YRange{Min: yang.FromInt(2), Max: yang.FromInt(10)},
			schemaName: "range-2-to-10",
			val:        int64(1),
			wantErr:    true,
		},
		{
			desc:       "non binary type - base []byte",
			length:     yang.YRange{Min: yang.FromInt(2), Max: yang.FromInt(10)},
			schemaName: "range-2-to-10",
			val:        []byte("a09Z+/"),
			wantErr:    true,
		},
		{
			desc:       "too short",
			length:     yang.YRange{Min: yang.FromInt(2), Max: yang.FromInt(10)},
			schemaName: "range-2-to-10",
			val:        Binary("a"),
			wantErr:    true,
		},
		{
			desc:       "too long",
			length:     yang.YRange{Min: yang.FromInt(2), Max: yang.FromInt(4)},
			schemaName: "range-2-to-4",
			val:        Binary("aaaaaaaa"),
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if isBinaryType(reflect.TypeOf(tt.val)) {
				binaryVal := reflect.ValueOf(tt.val).Bytes()
				if binaryVal != nil {
					err := ValidateBinaryRestrictions(yrangeToBinarySchema(tt.schemaName, tt.length).Type, binaryVal)
					if got, want := (err != nil), tt.wantErr; got != want {
						t.Fatalf("%s: b.ValidateBinaryRestrictions (%v) got error: %v, want error? %v", tt.desc, tt.val, err, tt.wantErr)
					}
					testErrLog(t, tt.desc, err)
				}
			}

			err := validateBinary(yrangeToBinarySchema(tt.schemaName, tt.length), tt.val)
			if got, want := (err != nil), tt.wantErr; got != want {
				t.Errorf("%s: b.validateBinary(%v) got error: %v, want error? %v", tt.desc, tt.val, err, tt.wantErr)
			}
			testErrLog(t, tt.desc, err)
		})
	}
}

func TestValidateBinarySlice(t *testing.T) {
	tests := []struct {
		desc       string
		length     yang.YRange
		schemaName string
		val        interface{}
		wantErr    bool
	}{
		{
			desc:       "success",
			length:     yang.YRange{Min: yang.FromInt(2), Max: yang.FromInt(10)},
			schemaName: "range-2-to-10",
			val:        []Binary{Binary("a09Z+/"), Binary("ab++")},
		},
		{
			desc:       "bad schema",
			length:     yang.YRange{Min: yang.FromInt(20), Max: yang.FromInt(10)},
			schemaName: "bad-range",
			val:        []Binary{Binary("a09Z+/"), Binary("ab++")},
			wantErr:    true,
		},
		{
			desc:       "non []string",
			length:     yang.YRange{Min: yang.FromInt(2), Max: yang.FromInt(10)},
			schemaName: "range-2-to-10",
			val:        []int64{1, 2},
			wantErr:    true,
		},
		{
			desc:       "one element too short",
			length:     yang.YRange{Min: yang.FromInt(2), Max: yang.FromInt(10)},
			schemaName: "range-2-to-10",
			val:        []Binary{Binary("a09Z+/"), Binary("a")},
			wantErr:    true,
		},
		{
			desc:       "duplicate element",
			length:     yang.YRange{Min: yang.FromInt(2), Max: yang.FromInt(10)},
			schemaName: "range-2-to-10",
			val:        []Binary{Binary("a09Z+/"), Binary("ab++"), Binary("ab++")},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := validateBinarySlice(yrangeToBinarySchema(tt.schemaName, tt.length), tt.val)
			if got, want := (err != nil), tt.wantErr; got != want {
				t.Errorf("%s: b.validateBinarySlice(%v) got error: %v, want error? %v", tt.desc, tt.val, err, tt.wantErr)
			}
			testErrLog(t, tt.desc, err)
		})
	}
}
