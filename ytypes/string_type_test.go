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
	"github.com/openconfig/ygot/util"
)

var validStringSchema = yrangeAndPatternToStringSchema("valid-string-schema", yang.YRange{Min: yang.FromInt(2), Max: yang.FromInt(10)}, nil)

func yrangeAndPatternToStringSchema(schemaName string, yr yang.YRange, rePattern []string) *yang.Entry {
	return &yang.Entry{Name: schemaName, Type: &yang.YangType{Kind: yang.Ystring, Length: yang.YangRange{yr}, Pattern: rePattern}}
}

func TestValidateStringSchema(t *testing.T) {
	tests := []struct {
		desc    string
		schema  *yang.Entry
		wantErr bool
	}{
		{
			desc:   "success",
			schema: validStringSchema,
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
			err := validateStringSchema(tt.schema)
			if got, want := (err != nil), tt.wantErr; got != want {
				t.Errorf("%s: validateStringSchema(%v) got error: %v, want error? %v", tt.desc, tt.schema, err, tt.wantErr)
			}
			testErrLog(t, tt.desc, err)
		})
	}
}

func TestValidateStringSchemaRanges(t *testing.T) {
	tests := []struct {
		desc       string
		length     yang.YRange
		schemaName string
		re         []string
		wantErr    bool
	}{
		{
			desc:       "success",
			length:     yang.YRange{Min: yang.FromInt(2), Max: yang.FromInt(10)},
			schemaName: "range-2-to-10",
		},
		{
			desc:       "unset min success",
			length:     yang.YRange{Min: util.YangMinNumber, Max: yang.FromInt(10)},
			schemaName: "range-10-or-less",
			re:         []string{`ab.`, `.*bc`},
		},
		{
			desc:       "unset max success",
			length:     yang.YRange{Min: yang.FromInt(2), Max: util.YangMaxNumber},
			schemaName: "range-2-or-more",
			re:         []string{`ab.`, `.*bc`},
		},
		{
			desc:       "unset min and max success",
			length:     yang.YRange{Min: util.YangMinNumber, Max: util.YangMaxNumber},
			schemaName: "range-any",
			re:         []string{`ab.`, `.*bc`},
		},
		{
			desc:       "bad length range",
			length:     yang.YRange{Min: yang.FromInt(20), Max: yang.FromInt(10)},
			schemaName: "bad-range",
			wantErr:    true,
		},
		{
			desc:       "negative min length",
			length:     yang.YRange{Min: yang.FromInt(-1), Max: util.YangMaxNumber},
			schemaName: "bad-range-negative-min",
			wantErr:    true,
		},
		{
			desc:       "negative max length",
			length:     yang.YRange{Min: util.YangMinNumber, Max: yang.FromInt(-1)},
			schemaName: "bad-range-negative-max",
			wantErr:    true,
		},
		{
			desc:       "bad pattern",
			length:     yang.YRange{Min: yang.FromInt(2), Max: yang.FromInt(10)},
			schemaName: "bad-pattern",
			re:         []string{"(^(.*)"},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := validateStringSchema(yrangeAndPatternToStringSchema(tt.schemaName, tt.length, tt.re))
			if got, want := (err != nil), tt.wantErr; got != want {
				t.Errorf("%s: validateStringSchema got error: %v, want error? %v", tt.desc, err, tt.wantErr)
			}
			testErrLog(t, tt.desc, err)
		})
	}
}

func TestValidateString(t *testing.T) {
	tests := []struct {
		desc       string
		length     yang.YRange
		schemaName string
		re         []string
		val        interface{}
		wantErr    bool
	}{
		{
			desc:       "success",
			length:     yang.YRange{Min: yang.FromInt(2), Max: yang.FromInt(10)},
			schemaName: "range-2-to-10",
			re:         []string{`ab.`, `.*bc`},
			val:        "abc",
		},
		{
			desc:       "logical expression",
			length:     yang.YRange{Min: yang.FromInt(2), Max: yang.FromInt(10)},
			schemaName: "range-2-to-10",
			re:         []string{`a+|\.`},
			val:        "aaa aaa",
			wantErr:    true,
		},
		{
			desc:       "bad schema",
			length:     yang.YRange{Min: yang.FromInt(20), Max: yang.FromInt(10)},
			schemaName: "bad-range",
			re:         []string{`ab.`, `.*bc`},
			val:        "abc",
			wantErr:    true,
		},
		{
			desc:       "regex failure",
			length:     yang.YRange{Min: yang.FromInt(2), Max: yang.FromInt(10)},
			schemaName: "range-2-to-10",
			re:         []string{`ab.`, `.*bc`},
			val:        "acbc",
			wantErr:    true,
		},
		{
			desc:       "repeating regex success",
			length:     yang.YRange{Min: yang.FromInt(2), Max: yang.FromInt(10)},
			schemaName: "range-2-to-10",
			re:         []string{`[a|b]*`},
			val:        "abbbab",
		},
		{
			desc:       "repeating regex failure",
			length:     yang.YRange{Min: yang.FromInt(2), Max: yang.FromInt(10)},
			schemaName: "range-2-to-10",
			re:         []string{`[a|b]*`},
			val:        "abbcbab",
			wantErr:    true,
		},
		{
			desc:       "regex failure with derived type name",
			length:     yang.YRange{Min: yang.FromInt(2), Max: yang.FromInt(10)},
			schemaName: "range-2-to-10",
			re:         []string{`ab.`, `.*bc`},
			val:        "acbc",
			wantErr:    true,
		},
		{
			desc:       "non string type",
			length:     yang.YRange{Min: util.YangMinNumber, Max: util.YangMaxNumber},
			schemaName: "range-any",
			val:        int64(123),
			wantErr:    true,
		},
		{
			desc:       "long string",
			length:     yang.YRange{Min: yang.FromInt(2), Max: yang.FromInt(4)},
			schemaName: "range-2-to-4",
			val:        "long_value",
			wantErr:    true,
		},
		{
			desc:       "short string",
			length:     yang.YRange{Min: yang.FromInt(20), Max: util.YangMaxNumber},
			schemaName: "range-20-or-more",
			val:        "short_value",
			wantErr:    true,
		},
		{
			desc:       "regular expression matching with no anchors OK",
			length:     yang.YRange{Min: util.YangMinNumber, Max: util.YangMaxNumber},
			schemaName: "range-any",
			re:         []string{`[ab]{2}([cd])?`},
			val:        "abc",
		},
		{
			desc:       "regular expression matching with no anchors failure",
			length:     yang.YRange{Min: util.YangMinNumber, Max: util.YangMaxNumber},
			schemaName: "range-any",
			re:         []string{`[ab]{2}([cd])?`},
			val:        "cdb",
			wantErr:    true,
		},
		{
			desc:       "unanchored regular expression does not match",
			length:     yang.YRange{Min: util.YangMinNumber, Max: util.YangMaxNumber},
			schemaName: "range-any",
			re:         []string{`[0-9]+`},
			val:        "abcd999",
			wantErr:    true,
		},
		{
			desc:       "regular expression matching with anchors",
			length:     yang.YRange{Min: util.YangMinNumber, Max: util.YangMaxNumber},
			schemaName: "range-any",
			re:         []string{`^[ab]{2}([cd])?$`},
			val:        "aad",
		},
		{
			desc:       "regular expression matching with embedded $",
			length:     yang.YRange{Min: util.YangMinNumber, Max: util.YangMaxNumber},
			schemaName: "range-any",
			re:         []string{`$[0-9]+`},
			val:        "$100",
		},
		{
			desc:       "regular expression matching with embedded ^",
			length:     yang.YRange{Min: util.YangMinNumber, Max: util.YangMaxNumber},
			schemaName: "range-any",
			re:         []string{`[a-z]+^`},
			val:        "caret^",
		},
		{
			desc:       "regular expression matching with escape chars",
			length:     yang.YRange{Min: util.YangMinNumber, Max: util.YangMaxNumber},
			schemaName: "range-any",
			re:         []string{`[0-9]+\.[0-9]+`},
			val:        "10.10",
		},
		{
			desc:       "regular expression with escaped escapes",
			length:     yang.YRange{Min: util.YangMinNumber, Max: util.YangMaxNumber},
			schemaName: "range-any",
			re:         []string{`foo\\^bar`},
			val:        `foo\^bar`,
		},
		{
			desc:       "regular expression with set negation, valid",
			length:     yang.YRange{Min: util.YangMinNumber, Max: util.YangMaxNumber},
			schemaName: "range-any",
			re:         []string{`[^:][0-9a-fA-F]+`},
			val:        ":FFFF",
			wantErr:    true,
		},
		{
			desc:       "regular expression with set negation, invalid",
			length:     yang.YRange{Min: util.YangMinNumber, Max: util.YangMaxNumber},
			schemaName: "range-any",
			re:         []string{`[^:][0-9a-fA-F]+`},
			val:        "CAFE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := validateString(yrangeAndPatternToStringSchema(tt.schemaName, tt.length, tt.re), tt.val)
			if got, want := (err != nil), tt.wantErr; got != want {
				t.Errorf("%s: s.validateString(%v) got error: %v, want error? %t", tt.desc, tt.val, err, tt.wantErr)
			}
			testErrLog(t, tt.desc, err)
		})
	}
}

func TestValidateStringSlice(t *testing.T) {
	tests := []struct {
		desc    string
		schema  *yang.Entry
		val     interface{}
		wantErr bool
	}{
		{
			desc:   "success",
			schema: validStringSchema,
			val:    []string{"aaa", "bbb", "ccc"},
		},
		{
			desc:    "bad schema",
			schema:  nil,
			val:     []string{"aaa"},
			wantErr: true,
		},
		{
			desc:    "non []string",
			schema:  validStringSchema,
			val:     []int32{1, 2},
			wantErr: true,
		},
		{
			desc:    "invalid element",
			schema:  validStringSchema,
			val:     []string{"aaa", "bbb", "this element is too long"},
			wantErr: true,
		},
		{
			desc:    "duplicate element",
			schema:  validStringSchema,
			val:     []string{"aaa", "bbb", "aaa"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := validateStringSlice(tt.schema, tt.val)
			if got, want := (err != nil), tt.wantErr; got != want {
				t.Errorf("%s: s.validateStringSlice(%v) got error: %v, want error? %t", tt.desc, tt.val, err, tt.wantErr)
			}
			testErrLog(t, tt.desc, err)
		})
	}
}
