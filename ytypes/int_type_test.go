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
	"fmt"
	"testing"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/util"
)

var (
	tooSmall       = make(map[yang.TypeKind]int64)
	tooLarge       = make(map[yang.TypeKind]int64)
	validIntSchema = typeAndRangeToIntSchema("uint8-schema", yang.Yint8, nil)
)

func init() {
	for k, v := range defaultIntegerRange {
		valMin, _ := v[0].Min.Int()
		valMax, _ := v[0].Max.Int()
		tooSmall[k] = valMin - 1
		tooLarge[k] = valMax + 1
	}
}

func typeAndRangeToIntSchema(schemaName string, t yang.TypeKind, r yang.YangRange) *yang.Entry {
	return &yang.Entry{
		Name: schemaName,
		Type: &yang.YangType{
			Kind:  t,
			Range: r,
		},
	}
}

func TestValidateIntSchema(t *testing.T) {
	tests := []struct {
		desc    string
		schema  *yang.Entry
		wantErr bool
	}{
		{
			desc:   "success",
			schema: validIntSchema,
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
		{
			desc: "bad range Min",
			schema: &yang.Entry{
				Name: "bad-range-schema",
				Type: &yang.YangType{
					Kind: yang.Yint8,
					Range: yang.YangRange{
						yang.YRange{
							Min: yang.Number{
								Kind: yang.MaxNumber,
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			desc: "bad range Max",
			schema: &yang.Entry{
				Name: "bad-range-schema",
				Type: &yang.YangType{
					Kind: yang.Yint8,
					Range: yang.YangRange{
						yang.YRange{
							Max: yang.Number{
								Kind: yang.MinNumber,
							},
						},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := validateIntSchema(tt.schema)
			if got, want := (err != nil), tt.wantErr; got != want {
				t.Errorf("%s: validateIntSchema(%v) got error: %v, want error? %v", tt.desc, tt.schema, err, tt.wantErr)
			}
			testErrLog(t, tt.desc, err)
		})
	}
}

func TestValidateIntSchemaRanges(t *testing.T) {
	tests := []struct {
		desc    string
		ranges  yang.YangRange
		wantErr bool
	}{
		{
			desc: "ranges [-,+], [+,+]",
			ranges: yang.YangRange{
				yang.YRange{Min: yang.FromInt(-10), Max: yang.FromInt(1)},
				yang.YRange{Min: yang.FromInt(5), Max: yang.FromInt(10)},
			},
			wantErr: true,
		},
		{
			desc: "ranges [+,+], [+,+]",
			ranges: yang.YangRange{
				yang.YRange{Min: yang.FromInt(1), Max: yang.FromInt(3)},
				yang.YRange{Min: yang.FromInt(5), Max: yang.FromInt(10)},
			},
		},
		{
			desc: "ranges [0,+], [+,+]",
			ranges: yang.YangRange{
				yang.YRange{Min: yang.FromInt(0), Max: yang.FromInt(3)},
				yang.YRange{Min: yang.FromInt(5), Max: yang.FromInt(10)},
			},
		},
	}

	if err := validateIntSchema(typeAndRangeToIntSchema("string-schema", yang.Ystring, nil)); err == nil {
		t.Errorf("validateIntSchema bad type (Ystring): got: nil, want: error")
	}

	yangIntTypes := []yang.TypeKind{yang.Yuint8, yang.Yuint16, yang.Yuint32, yang.Yuint64}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			for _, ty := range yangIntTypes {
				err := validateIntSchema(typeAndRangeToIntSchema(tt.desc+"-schema", ty, tt.ranges))
				if got, want := (err != nil), tt.wantErr; got != want {
					t.Errorf("%s: validateIntSchema(%v) for %v got error: %v, want error? %v",
						tt.desc, ty, tt.ranges, err, tt.wantErr)
				}
				testErrLog(t, tt.desc, err)
			}
		})
	}
}

func TestIntRangeOverflow(t *testing.T) {
	tests := []struct {
		desc    string
		ranges  func(yt yang.TypeKind) yang.YangRange
		wantErr bool
	}{
		{
			desc: "bad min",
			ranges: func(yt yang.TypeKind) yang.YangRange {
				return yang.YangRange{yang.YRange{Min: yang.FromInt(tooSmall[yt]), Max: util.YangMaxNumber}}
			},
			wantErr: true,
		},
		{
			desc: "bad max",
			ranges: func(yt yang.TypeKind) yang.YangRange {
				return yang.YangRange{yang.YRange{Min: util.YangMinNumber, Max: yang.FromInt(tooLarge[yt])}}
			},
			wantErr: true,
		},
	}

	yangIntTypes := []yang.TypeKind{yang.Yint8, yang.Yint16, yang.Yint32,
		yang.Yuint8, yang.Yuint16, yang.Yuint32}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			for _, ty := range yangIntTypes {
				err := validateIntSchema(typeAndRangeToIntSchema(tt.desc+"-schema", ty, tt.ranges(ty)))
				if err == nil {
					t.Errorf("%s: validateIntSchema(%v) for %v, got nil, want overflow error",
						tt.desc, ty, tt.ranges(ty))
				}
			}
		})
	}

}

func TestValidateInt(t *testing.T) {
	tests := []struct {
		desc      string
		ranges    yang.YangRange
		inValues  []int64
		outValues []int64
	}{
		{
			desc: "single val range -ve",
			ranges: yang.YangRange{
				yang.YRange{Min: yang.FromInt(-10), Max: yang.FromInt(-10)},
			},
			inValues:  []int64{-10},
			outValues: []int64{-11, -9},
		},
		{
			desc: "single val range 0",
			ranges: yang.YangRange{
				yang.YRange{Min: yang.FromInt(0), Max: yang.FromInt(0)},
			},
			inValues:  []int64{0},
			outValues: []int64{-1, 1},
		},
		{
			desc: "single val range +ve",
			ranges: yang.YangRange{
				yang.YRange{Min: yang.FromInt(10), Max: yang.FromInt(10)},
			},
			inValues:  []int64{10},
			outValues: []int64{9, 11},
		},
		{
			desc: "ranges [-,-], [-,-]",
			ranges: yang.YangRange{
				yang.YRange{Min: yang.FromInt(-10), Max: yang.FromInt(-5)},
				yang.YRange{Min: yang.FromInt(-3), Max: yang.FromInt(-1)},
			},
			inValues:  []int64{-10, -7, -5, -3, -2, -1},
			outValues: []int64{-11, -4, 0, 1},
		},
		{
			desc: "ranges [-,-], [-,+]",
			ranges: yang.YangRange{
				yang.YRange{Min: yang.FromInt(-10), Max: yang.FromInt(-5)},
				yang.YRange{Min: yang.FromInt(-3), Max: yang.FromInt(10)},
			},
			inValues:  []int64{-10, -7, -5, -3, -2, 0, 5, 10},
			outValues: []int64{-11, -4, 11},
		},
		{
			desc: "ranges [-,-], [+,+]",
			ranges: yang.YangRange{
				yang.YRange{Min: yang.FromInt(-10), Max: yang.FromInt(-5)},
				yang.YRange{Min: yang.FromInt(5), Max: yang.FromInt(10)},
			},
			inValues:  []int64{-10, -7, -5, 5, 7, 10},
			outValues: []int64{-11, -4, 0, 4, 11},
		},
		{
			desc: "ranges [-,+], [+,+]",
			ranges: yang.YangRange{
				yang.YRange{Min: yang.FromInt(-10), Max: yang.FromInt(1)},
				yang.YRange{Min: yang.FromInt(5), Max: yang.FromInt(10)},
			},
			inValues:  []int64{-10, 0, 1, 5, 7, 10},
			outValues: []int64{-11, 2, 4, 11},
		},
		{
			desc: "ranges [+,+], [+,+]",
			ranges: yang.YangRange{
				yang.YRange{Min: yang.FromInt(1), Max: yang.FromInt(3)},
				yang.YRange{Min: yang.FromInt(5), Max: yang.FromInt(10)},
			},
			inValues:  []int64{1, 2, 3, 5, 7, 10},
			outValues: []int64{-1, 0, 4, 11},
		},
		{
			desc: "ranges [-,0], [+,+]",
			ranges: yang.YangRange{
				yang.YRange{Min: yang.FromInt(-10), Max: yang.FromInt(0)},
				yang.YRange{Min: yang.FromInt(5), Max: yang.FromInt(10)},
			},
			inValues:  []int64{-10, -7, 0, 5, 7, 10},
			outValues: []int64{-11, 1, 4, 11},
		},
		{
			desc: "ranges [-,-], [0,+]",
			ranges: yang.YangRange{
				yang.YRange{Min: yang.FromInt(-10), Max: yang.FromInt(-5)},
				yang.YRange{Min: yang.FromInt(0), Max: yang.FromInt(10)},
			},
			inValues:  []int64{-10, -7, -5, 0, 5, 10},
			outValues: []int64{-11, -4, -1, 11},
		},
		{
			desc: "ranges [-inf,-], [0,+]",
			ranges: yang.YangRange{
				yang.YRange{Min: util.YangMinNumber, Max: yang.FromInt(-5)},
				yang.YRange{Min: yang.FromInt(0), Max: yang.FromInt(10)},
			},
			inValues:  []int64{-100, -7, -5, 0, 5, 10},
			outValues: []int64{-4, -1, 11},
		},
		{
			desc: "ranges [-,-], [0,+inf]",
			ranges: yang.YangRange{
				yang.YRange{Min: yang.FromInt(-10), Max: yang.FromInt(-5)},
				yang.YRange{Min: yang.FromInt(0), Max: util.YangMaxNumber},
			},
			inValues:  []int64{-10, -7, -5, 0, 5, 100},
			outValues: []int64{-11, -4, -1},
		},
	}

	yangIntTypes := []yang.TypeKind{yang.Yint8, yang.Yint16, yang.Yint32, yang.Yint64}

	// Bad schema type.
	if err := validateInt(typeAndRangeToIntSchema("bad-schema", yang.Ystring, nil), nil); err == nil {
		t.Errorf("TestvalidateInt bad schema type (Ystring): got: nil, want: error")
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			for _, ty := range yangIntTypes {
				for _, val := range tt.inValues {
					if err := validateInt(typeAndRangeToIntSchema(tt.desc+"-schema", ty, tt.ranges), toGoType(ty, val)); err != nil {
						t.Errorf("%s: Validate for %v: %v should be inside ranges %v",
							tt.desc, ty, val, tt.ranges)
					}
				}
				for _, val := range tt.outValues {
					if err := validateInt(typeAndRangeToIntSchema(tt.desc+"-schema", ty, tt.ranges), toGoType(ty, val)); err == nil {
						t.Errorf("%s: Validate for %v: %v should be outside ranges %v",
							tt.desc, ty, val, tt.ranges)
					}
				}
			}
		})
	}
}

func TestValidateUint(t *testing.T) {
	tests := []struct {
		desc      string
		ranges    yang.YangRange
		inValues  []int64
		outValues []int64
	}{
		{
			desc: "single val range 0",
			ranges: yang.YangRange{
				yang.YRange{Min: yang.FromInt(0), Max: yang.FromInt(0)},
			},
			inValues:  []int64{0},
			outValues: []int64{1},
		},
		{
			desc: "single val range +ve",
			ranges: yang.YangRange{
				yang.YRange{Min: yang.FromInt(10), Max: yang.FromInt(10)},
			},
			inValues:  []int64{10},
			outValues: []int64{0, 9, 11},
		},
		{
			desc: "ranges [0,+], [+,+]",
			ranges: yang.YangRange{
				yang.YRange{Min: yang.FromInt(0), Max: yang.FromInt(3)},
				yang.YRange{Min: yang.FromInt(5), Max: yang.FromInt(10)},
			},
			inValues:  []int64{1, 2, 3, 5, 7, 10},
			outValues: []int64{4, 11},
		},
		{
			desc: "ranges [0,+], [+,+inf]",
			ranges: yang.YangRange{
				yang.YRange{Min: yang.FromInt(0), Max: yang.FromInt(3)},
				yang.YRange{Min: yang.FromInt(5), Max: yang.FromInt(10)},
			},
			inValues:  []int64{0, 1, 2, 3, 5, 7, 10},
			outValues: []int64{4, 11},
		},
	}

	yangIntTypes := []yang.TypeKind{yang.Yuint8, yang.Yuint16, yang.Yuint32, yang.Yuint64}
	//yangIntTypes := []yang.TypeKind{yang.Yuint8}

	// Bad schema type.
	if err := validateInt(typeAndRangeToIntSchema("bad-schema", yang.Ystring, nil), nil); err == nil {
		t.Errorf("TestvalidateInt bad schema type (Ystring): got: nil, want: error")
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			for _, ty := range yangIntTypes {
				for _, val := range tt.inValues {
					if err := validateInt(typeAndRangeToIntSchema(tt.desc+"-schema", ty, tt.ranges), toGoType(ty, val)); err != nil {
						t.Errorf("%s: Validate for %v: %v should be inside ranges %v",
							tt.desc, ty, val, tt.ranges)
					}
				}
				for _, val := range tt.outValues {
					if err := validateInt(typeAndRangeToIntSchema(tt.desc+"-schema", ty, tt.ranges), toGoType(ty, val)); err == nil {
						t.Errorf("%s: Validate for %v: %v should be outside ranges %v",
							tt.desc, ty, val, tt.ranges)
					}
				}
			}
		})
	}
}

func TestValidateIntSlice(t *testing.T) {
	tests := []struct {
		desc    string
		val     []int64
		wantErr bool
	}{
		{
			desc: "valid slice",
			val:  []int64{2, 0, 1},
		},
		{
			desc:    "repeated value",
			val:     []int64{1, 0, 1},
			wantErr: true,
		},
	}

	// Bad schema type.
	want := `string is not an integer type for schema bad-schema`
	if got := errToString(validateIntSlice(typeAndRangeToIntSchema("bad-schema", yang.Ystring, nil), nil)); got != want {
		t.Errorf("TestvalidateIntSlice bad schema type (Ystring): got: %s, want: %s", got, want)
	}

	// Bad value type.
	want = `got type []int64 with value [1 2 3], want []int8 for schema uint8-schema`
	if got := errToString(validateIntSlice(typeAndRangeToIntSchema("uint8-schema", yang.Yint8, nil), []int64{1, 2, 3})); got != want {
		t.Errorf("TestValidateIntSlice bad value type: got: %s, want: %s", got, want)
	}

	yangIntTypes := []yang.TypeKind{yang.Yint8, yang.Yint16, yang.Yint32,
		yang.Yuint8, yang.Yuint16, yang.Yuint32}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			for _, ty := range yangIntTypes {
				err := validateIntSlice(typeAndRangeToIntSchema(tt.desc+"-schema", ty, nil), toGoSliceType(ty, tt.val))
				if got, want := (err != nil), tt.wantErr; got != want {
					t.Errorf("%s: validateIntSlice for %v: got: %v, want %v",
						tt.desc, ty, got, want)
				}
				testErrLog(t, tt.desc, err)
			}
		})
	}
}

func toGoType(kind yang.TypeKind, val int64) interface{} {
	switch kind {
	case yang.Yint8:
		return int8(val)
	case yang.Yint16:
		return int16(val)
	case yang.Yint32:
		return int32(val)
	case yang.Yint64:
		return int64(val)
	case yang.Yuint8:
		return uint8(val)
	case yang.Yuint16:
		return uint16(val)
	case yang.Yuint32:
		return uint32(val)
	case yang.Yuint64:
		return uint64(val)
	default:
		panic(fmt.Sprintf("bad int type %v", kind))
	}
}

func toGoSliceType(kind yang.TypeKind, in []int64) interface{} {
	switch kind {
	case yang.Yint8:
		var out []int8
		for _, v := range in {
			out = append(out, int8(v))
		}
		return out
	case yang.Yint16:
		var out []int16
		for _, v := range in {
			out = append(out, int16(v))
		}
		return out
	case yang.Yint32:
		var out []int32
		for _, v := range in {
			out = append(out, int32(v))
		}
		return out
	case yang.Yint64:
		var out []int64
		for _, v := range in {
			out = append(out, int64(v))
		}
		return out
	case yang.Yuint8:
		var out []uint8
		for _, v := range in {
			out = append(out, uint8(v))
		}
		return out
	case yang.Yuint16:
		var out []uint16
		for _, v := range in {
			out = append(out, uint16(v))
		}
		return out
	case yang.Yuint32:
		var out []uint32
		for _, v := range in {
			out = append(out, uint32(v))
		}
		return out
	case yang.Yuint64:
		var out []uint64
		for _, v := range in {
			out = append(out, uint64(v))
		}
		return out
	default:
		panic(fmt.Sprintf("bad int type %v", kind))
	}
}
