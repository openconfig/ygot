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

const (
	largestValidFloat         = float64(yang.MaxInt64)
	largestNegativeValidFloat = float64(yang.MinInt64)
	// Overflows to max
	tooLargeFloat = 123e123
	// Overflows to min
	tooLargeNegativeFloat = -123e123
	// Section 9.3.4 places a limit on decimal representation. Rounded to 0.
	tooSmallFloat = 1e-20
)

var validDecimalSchema = &yang.Entry{Name: "valid-decimal-schema", Type: &yang.YangType{Kind: yang.Ydecimal64}}

func rangeToDecimalSchema(schemaName string, r yang.YangRange) *yang.Entry {
	return &yang.Entry{
		Name: schemaName,
		Type: &yang.YangType{
			Kind:  yang.Ydecimal64,
			Range: r,
		},
	}
}

func TestValidateDecimalSchema(t *testing.T) {
	tests := []struct {
		desc    string
		schema  *yang.Entry
		wantErr bool
	}{
		{
			desc:   "success",
			schema: validDecimalSchema,
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
			err := validateDecimalSchema(tt.schema)
			if got, want := (err != nil), tt.wantErr; got != want {
				t.Errorf("%s: validDecimalSchema(%v) got error: %v, want error? %v", tt.desc, tt.schema, err, tt.wantErr)
			}
			testErrLog(t, tt.desc, err)
		})
	}
}

func TestValidateDecimalType(t *testing.T) {
	tests := []struct {
		desc    string
		schema  *yang.Entry
		val     interface{}
		wantErr bool
	}{
		{
			desc:   "success",
			schema: validDecimalSchema,
			val:    float64(4.4),
		},
		{
			desc:    "bad schema",
			schema:  nil,
			val:     float64(4.4),
			wantErr: true,
		},
		{
			desc:    "non float64 type",
			schema:  validDecimalSchema,
			val:     "",
			wantErr: true,
		},
		{
			desc:   "largest float",
			schema: validDecimalSchema,
			val:    tooLargeFloat,
		},
		{
			desc:   "largest -ve float",
			schema: validDecimalSchema,
			val:    tooLargeNegativeFloat,
		},
		{
			// This is ok, rounds to maxNumber (see yang.FromFloat)
			desc:   "too large",
			schema: validDecimalSchema,
			val:    tooLargeFloat,
		},
		{
			// This is ok, rounds to minNumber.
			desc:   "too large -ve",
			schema: validDecimalSchema,
			val:    tooLargeNegativeFloat,
		},
		{
			// This is ok, rounds to 0.
			desc:   "too small float",
			schema: validDecimalSchema,
			val:    tooSmallFloat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := validateDecimal(tt.schema, tt.val)
			if got, want := (err != nil), tt.wantErr; got != want {
				t.Errorf("%s: validateDecimal(%v) got error: %v, want error? %v", tt.desc, tt.val, err, tt.wantErr)
			}
			testErrLog(t, tt.desc, err)
		})
	}
}

func TestValidateDecimalValue(t *testing.T) {
	tests := []struct {
		desc      string
		ranges    yang.YangRange
		inValues  []float64
		outValues []float64
	}{
		{
			desc: "single val range -ve",
			ranges: yang.YangRange{
				yang.YRange{Min: yang.FromFloat(-10.1), Max: yang.FromFloat(-10.1)},
			},
			inValues:  []float64{-10.1},
			outValues: []float64{tooLargeNegativeFloat, largestNegativeValidFloat, -10.15, -9, largestValidFloat, tooLargeFloat},
		},
		{
			desc: "single val range 0",
			ranges: yang.YangRange{
				yang.YRange{Min: yang.FromFloat(0), Max: yang.FromFloat(0)},
			},
			inValues:  []float64{0, tooSmallFloat},
			outValues: []float64{tooLargeNegativeFloat, largestNegativeValidFloat, -1e15, 1e15, largestValidFloat, tooLargeFloat},
		},
		{
			desc: "single val range +ve",
			ranges: yang.YangRange{
				yang.YRange{Min: yang.FromFloat(10.1), Max: yang.FromFloat(10.1)},
			},
			inValues:  []float64{10.1},
			outValues: []float64{tooLargeNegativeFloat, largestNegativeValidFloat, 10.05, 10.15, largestValidFloat, tooLargeFloat},
		},
		{
			desc: "ranges [-,-], [-,-]",
			ranges: yang.YangRange{
				yang.YRange{Min: yang.FromFloat(-10.1), Max: yang.FromFloat(-5.1)},
				yang.YRange{Min: yang.FromFloat(-3.1), Max: yang.FromFloat(-1.1)},
			},
			inValues:  []float64{-10.1, -10.05, -7, -5.1, -3.1, -3.05, -1.1},
			outValues: []float64{tooLargeNegativeFloat, largestNegativeValidFloat, -10.15, -5.05, 0, -1.05, largestValidFloat, tooLargeFloat},
		},
		{
			desc: "ranges [-,-], [-,+]",
			ranges: yang.YangRange{
				yang.YRange{Min: yang.FromFloat(-10.1), Max: yang.FromFloat(-5.1)},
				yang.YRange{Min: yang.FromFloat(-3.1), Max: yang.FromFloat(10.1)},
			},
			inValues:  []float64{-10.1, -10.05, -5.1, -3.1, -3.05, 0, tooSmallFloat, 5, 10.05, 10.1},
			outValues: []float64{tooLargeNegativeFloat, largestNegativeValidFloat, -10.15, -5.05, 10.15, largestValidFloat, tooLargeFloat},
		},
		{
			desc: "ranges [-,-], [+,+]",
			ranges: yang.YangRange{
				yang.YRange{Min: yang.FromFloat(-10.1), Max: yang.FromFloat(-5.1)},
				yang.YRange{Min: yang.FromFloat(5.1), Max: yang.FromFloat(10.1)},
			},
			inValues:  []float64{-10.1, -10.05, -5.15, -5.1, 5.1, 5.15, 10.05, 10.1},
			outValues: []float64{tooLargeNegativeFloat, largestNegativeValidFloat, -10.15, -5.05, 0, 5.05, 10.15, largestValidFloat, tooLargeFloat},
		},
		{
			desc: "ranges [-,+], [+,+]",
			ranges: yang.YangRange{
				yang.YRange{Min: yang.FromFloat(-10.1), Max: yang.FromFloat(1.1)},
				yang.YRange{Min: yang.FromFloat(5.1), Max: yang.FromFloat(10.1)},
			},
			inValues:  []float64{-10.1, -10.05, 0, tooSmallFloat, 1.05, 1.1, 5.1, 5.15, 7, 10.05, 10.1},
			outValues: []float64{tooLargeNegativeFloat, largestNegativeValidFloat, -10.15, 1.15, 5.05, 10.15, largestValidFloat, tooLargeFloat},
		},
		{
			desc: "ranges [+,+], [+,+]",
			ranges: yang.YangRange{
				yang.YRange{Min: yang.FromFloat(1.1), Max: yang.FromFloat(3.1)},
				yang.YRange{Min: yang.FromFloat(5.1), Max: yang.FromFloat(10.1)},
			},
			inValues:  []float64{1.1, 1.15, 3.05, 3.1, 5.1, 5.15, 7, 10.05, 10.1},
			outValues: []float64{tooLargeNegativeFloat, largestNegativeValidFloat, -1, 0, 1.05, 3.15, 5.05, 10.15, largestValidFloat, tooLargeFloat},
		},
		{
			desc: "ranges [-,0], [+,+]",
			ranges: yang.YangRange{
				yang.YRange{Min: yang.FromFloat(-10.1), Max: yang.FromFloat(0)},
				yang.YRange{Min: yang.FromFloat(5.1), Max: yang.FromFloat(10.1)},
			},
			inValues:  []float64{-10, -7, 0, tooSmallFloat, 5.1, 7, 10},
			outValues: []float64{tooLargeNegativeFloat, largestNegativeValidFloat, -10.15, 0.01, 5.05, 10.105, largestValidFloat, tooLargeFloat},
		},
		{
			desc: "ranges [-,-], [0,+]",
			ranges: yang.YangRange{
				yang.YRange{Min: yang.FromFloat(-10.1), Max: yang.FromFloat(-5.1)},
				yang.YRange{Min: yang.FromFloat(0), Max: yang.FromFloat(10.1)},
			},
			inValues:  []float64{-10.1, -10.0005, -7, -5.1, 0, tooSmallFloat, 5, 10.005, 10.1},
			outValues: []float64{tooLargeNegativeFloat, largestNegativeValidFloat, -10.1005, -5.005, -0.001, 10.15, largestValidFloat, tooLargeFloat},
		},
		{
			desc: "ranges [-inf,-], [0,+]",
			ranges: yang.YangRange{
				yang.YRange{Min: util.YangMinNumber, Max: yang.FromFloat(-5.1)},
				yang.YRange{Min: yang.FromFloat(0), Max: yang.FromFloat(10.1)},
			},
			inValues:  []float64{tooLargeNegativeFloat, largestNegativeValidFloat, -100, -7, -5.1, 0, tooSmallFloat, 5, 10},
			outValues: []float64{-5.05, -0.001, 10.10001, largestValidFloat, tooLargeFloat},
		},
		{
			desc: "ranges [-,-], [0,+inf]",
			ranges: yang.YangRange{
				yang.YRange{Min: yang.FromFloat(-10.1), Max: yang.FromFloat(-5.1)},
				yang.YRange{Min: yang.FromFloat(0), Max: util.YangMaxNumber},
			},
			inValues:  []float64{-10.1, -10.05, -7, -5.100001, -5.1, 0, tooSmallFloat, 5, 100, largestValidFloat, tooLargeFloat},
			outValues: []float64{tooLargeNegativeFloat, largestNegativeValidFloat, -10.15, -5.0009, -0.0001},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			for _, val := range tt.inValues {
				if err := validateDecimal(rangeToDecimalSchema(tt.desc+"-schema", tt.ranges), val); err != nil {
					t.Errorf("%s: %v should be inside ranges %v", tt.desc, val, tt.ranges)
				}
			}
			for _, val := range tt.outValues {
				if err := validateDecimal(rangeToDecimalSchema(tt.desc+"-schema", tt.ranges), val); err == nil {
					t.Errorf("%s: %v should be outside ranges %v", tt.desc, val, tt.ranges)
				}
			}
		})
	}
}

func TestValidateDecimalSlice(t *testing.T) {
	tests := []struct {
		desc     string
		schema   *yang.Entry
		val      interface{}
		wantErr  bool
		sliceLen int32
	}{
		{
			desc:   "success",
			schema: validDecimalSchema,
			val:    []float64{4.4, 5.0},
		},
		{
			desc:    "bad schema",
			schema:  nil,
			val:     []float64{4.4, 5.0},
			wantErr: true,
		},
		{
			desc:    "non []float64",
			schema:  validDecimalSchema,
			val:     []int{1, 2, 3},
			wantErr: true,
		},
		{
			desc:    "duplicate element",
			schema:  validDecimalSchema,
			val:     []float64{4.4, 5.0, 5.0},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := validateDecimalSlice(tt.schema, tt.val)
			if got, want := (err != nil), tt.wantErr; got != want {
				t.Errorf("%s: validateDecimal(%v) got error: %v, want error? %v", tt.desc, tt.val, err, tt.wantErr)
			}
			testErrLog(t, tt.desc, err)
		})
	}
}
