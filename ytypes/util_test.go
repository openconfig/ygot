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
	"github.com/openconfig/ygot/ygot"
)

func TestYangBuiltinTypeToGoType(t *testing.T) {
	tests := []struct {
		desc  string
		ykind yang.TypeKind
		want  reflect.Kind
	}{
		{
			desc:  "int8",
			ykind: yang.Yint8,
			want:  reflect.Int8,
		},
		{
			desc:  "uint8",
			ykind: yang.Yuint8,
			want:  reflect.Uint8,
		},
		{
			desc:  "int16",
			ykind: yang.Yint16,
			want:  reflect.Int16,
		},
		{
			desc:  "uint16",
			ykind: yang.Yuint16,
			want:  reflect.Uint16,
		},
		{
			desc:  "int32",
			ykind: yang.Yint32,
			want:  reflect.Int32,
		},
		{
			desc:  "uint32",
			ykind: yang.Yuint32,
			want:  reflect.Uint32,
		},
		{
			desc:  "int64",
			ykind: yang.Yint64,
			want:  reflect.Int64,
		},
		{
			desc:  "uint64",
			ykind: yang.Yuint64,
			want:  reflect.Uint64,
		},
		{
			desc:  "bool",
			ykind: yang.Ybool,
			want:  reflect.Bool,
		},
		{
			desc:  "empty",
			ykind: yang.Yempty,
			want:  reflect.Bool,
		},
		{
			desc:  "string",
			ykind: yang.Ystring,
			want:  reflect.String,
		},
		{
			desc:  "decimal",
			ykind: yang.Ydecimal64,
			want:  reflect.Float64,
		},
		{
			desc:  "binary",
			ykind: yang.Ybinary,
			want:  reflect.Slice,
		},
		{
			desc:  "enum",
			ykind: yang.Yenum,
			want:  reflect.Int64,
		},
		{
			desc:  "identityref",
			ykind: yang.Yidentityref,
			want:  reflect.Int64,
		},
	}

	for _, tt := range tests {
		if got, want := reflect.TypeOf(yangBuiltinTypeToGoType(tt.ykind)).Kind(), tt.want; got != want {
			t.Errorf("%s: got : %s, want: %s", tt.desc, got, want)
		}
	}

	// TODO(mostrowski): bitset not implemented
	if got := yangBuiltinTypeToGoType(yang.Ybits); got != nil {
		t.Errorf("bitset: got : %s, want: nil", got)
	}
}

func TestYangToJSONType(t *testing.T) {
	tests := []struct {
		desc   string
		ykinds []yang.TypeKind
		want   reflect.Kind
	}{
		{
			desc: "to float",
			ykinds: []yang.TypeKind{
				yang.Yint8, yang.Yuint8,
				yang.Yint16, yang.Yuint16,
				yang.Yint32, yang.Yuint32,
			},
			want: reflect.Float64,
		},
		{
			desc: "to string",
			ykinds: []yang.TypeKind{
				yang.Yint64, yang.Yuint64,
				yang.Ydecimal64, yang.Yuint64,
				yang.Yenum, yang.Yidentityref, yang.Ystring,
			},
			want: reflect.String,
		},
		{
			desc: "to bool",
			ykinds: []yang.TypeKind{
				yang.Ybool,
			},
			want: reflect.Bool,
		},
		{
			desc: "to []interface{}",
			ykinds: []yang.TypeKind{
				yang.Yempty,
			},
			want: reflect.Slice,
		},
	}

	for _, tt := range tests {
		for _, yk := range tt.ykinds {
			if got, want := yangToJSONType(yk).Kind(), tt.want; got != want {
				t.Errorf("%s from %s: got : %s, want: %s", tt.desc, yk, got, want)
			}
		}
	}

	if got := yangToJSONType(yang.Yunion); got != nil {
		t.Errorf("got: %v, want: nil", got)
	}
}

type testEnum int64

const (
	Enum1 testEnum = 1
	Enum2 testEnum = 2
	Enum3 testEnum = 3
)

func (testEnum) IsYANGGoEnum() {}

func (testEnum) ΛMap() map[string]map[int64]ygot.EnumDefinition { return ΛEnum }

var ΛEnum = map[string]map[int64]ygot.EnumDefinition{
	"testEnum": {
		1: {Name: "test_enum1"},
		2: {Name: "test_enum2"},
		3: {Name: "test_enum3"},
	},
}

type testStruct struct {
	Test testEnum
}

func TestStringToType(t *testing.T) {
	ts := testStruct{}
	tests := []struct {
		s       string
		t       reflect.Type
		wantErr bool
	}{
		{s: "123", t: reflect.TypeOf(uint16(10))},
		{s: "123", t: reflect.TypeOf(uint32(20))},
		{s: "123", t: reflect.TypeOf(int16(-30))},
		// invalid value for the type
		{s: "fortytwo", t: reflect.TypeOf(uint16(0)), wantErr: true},
		// overflowing value for the type
		{s: "257", t: reflect.TypeOf(uint8(0)), wantErr: true},
		{s: "test_enum3", t: reflect.TypeOf(ts.Test)},
		// invalid enum for the enum type
		{s: "fortytwo", t: reflect.TypeOf(ts.Test), wantErr: true},
	}

	for i, tt := range tests {
		v, e := StringToType(tt.t, tt.s)
		if (e != nil) != tt.wantErr {
			t.Errorf("#%d got %v, want error %v", i+1, e, tt.wantErr)
			continue
		}
		if e != nil {
			continue
		}
		if v.Type() != tt.t {
			t.Errorf("#%d got %v, want %v type", i+1, v.Type(), tt.t)
		}
	}
}

func TestDirectDescendantSchema(t *testing.T) {
	tests := []struct {
		desc    string
		s       interface{}
		w       string
		wantErr bool
	}{
		{
			desc: "simple schema tag",
			s: struct {
				f string `path:"key"`
			}{},
			w: "key",
		},
		{
			desc: "multiple schema tag",
			s: struct {
				f string `path:"config/key|key"`
			}{},
			w: "key",
		},
		{
			desc: "in the middle direct descendant",
			s: struct {
				f string `path:"config/key|key|state/key"`
			}{},
			w: "key",
		},
		{
			desc: "missing schema tag",
			s: struct {
				f string
			}{},
			wantErr: true,
		},
	}

	for i, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			k, e := directDescendantSchema(reflect.TypeOf(tt.s).Field(0))
			if (e != nil) != tt.wantErr {
				t.Fatalf("#%d got %v, want error %v", i, e, tt.wantErr)
			}
			if e != nil {
				return
			}
			if tt.w != k {
				t.Errorf("#%d got %v, want %v", i, k, tt.w)
			}
		})
	}
}
