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
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/openconfig/gnmi/errdiff"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/testutil"
	"github.com/openconfig/ygot/ygot"
)

//lint:file-ignore U1000 Ignore all unused code, it represents generated code.

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

func (e testEnum) String() string {
	return ygot.EnumLogString(e, int64(e), "testEnum")
}

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
		{s: "hehehe", t: reflect.TypeOf("")},
		{s: "123", t: reflect.TypeOf(uint16(10))},
		{s: "123", t: reflect.TypeOf(uint32(20))},
		{s: "123", t: reflect.TypeOf(int16(-30))},
		{s: "true", t: reflect.TypeOf(true)},
		{s: "false", t: reflect.TypeOf(false)},
		{s: "yes", t: reflect.TypeOf(false), wantErr: true},
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

type allKeysListStruct struct {
	StringKey               *string             `path:"stringKey"`
	Int8Key                 *int8               `path:"int8Key"`
	Int16Key                *int16              `path:"int16Key"`
	Int32Key                *int32              `path:"int32Key"`
	Int64Key                *int64              `path:"int64Key"`
	Uint8Key                *uint8              `path:"uint8Key"`
	Uint16Key               *uint16             `path:"uint16Key"`
	Uint32Key               *uint32             `path:"uint32Key"`
	Uint64Key               *uint64             `path:"uint64Key"`
	Decimal64Key            *float64            `path:"decimal64Key"`
	BoolKey                 *bool               `path:"boolKey"`
	BinaryKey               Binary              `path:"binaryKey"`
	EnumKey                 EnumType            `path:"enumKey"`
	LeafrefKey              *uint64             `path:"leafrefKey"`
	LeafrefToLeafrefKey     *uint64             `path:"leafrefToLeafrefKey"`
	LeafrefToUnionKey       testutil.TestUnion  `path:"leafrefToUnionKey"`
	UnionKey                testutil.TestUnion  `path:"unionKey"`
	UnionLoneTypeKey        *uint32             `path:"unionLoneTypeKey"`
	LeafrefToUnionKeySimple testutil.TestUnion2 `path:"leafrefToUnionKeySimple"`
	UnionKeySimple          testutil.TestUnion2 `path:"unionKeySimple"`
}

func (t *allKeysListStruct) To_TestUnion(i interface{}) (testutil.TestUnion, error) {
	switch v := i.(type) {
	case EnumType:
		return &Union1EnumType{v}, nil
	case int16:
		return &Union1Int16{v}, nil
	case string:
		return &Union1String{v}, nil
	default:
		return nil, fmt.Errorf("cannot convert %v to testutil.TestUnion, unknown union type, got: %T", i, i)
	}
}

func (*allKeysListStruct) To_TestUnion2(i interface{}) (testutil.TestUnion2, error) {
	if v, ok := i.(testutil.TestUnion2); ok {
		return v, nil
	}
	switch v := i.(type) {
	case []byte:
		return testutil.Binary(v), nil
	case int16:
		return testutil.UnionInt16(v), nil
	case int64:
		return testutil.UnionInt64(v), nil
	}
	return nil, fmt.Errorf("cannot convert %v to testutil.TestUnion2, unknown union type, got: %T, want any of [EnumType, Binary, Int16, Int64]", i, i)
}

func (*allKeysListStruct) ΛEnumTypeMap() map[string][]reflect.Type {
	return map[string][]reflect.Type{
		"/struct-key-list/unionKey":       {reflect.TypeOf(EnumType(0))},
		"/struct-key-list/unionKeySimple": {reflect.TypeOf(EnumType(0))},
	}
}

func TestStringToKeyType(t *testing.T) {
	listSchema := &yang.Entry{
		Name:     "struct-key-list",
		Kind:     yang.DirectoryEntry,
		ListAttr: yang.NewDefaultListAttr(),
		Key:      "every key, but irrelevant in this test",
		Config:   yang.TSTrue,
		Dir: map[string]*yang.Entry{
			"stringKey": {
				Kind: yang.LeafEntry,
				Name: "stringKey",
				Type: &yang.YangType{Kind: yang.Ystring},
			},
			"int8Key": {
				Kind: yang.LeafEntry,
				Name: "int8Key",
				Type: &yang.YangType{Kind: yang.Yint8},
			},
			"int16Key": {
				Kind: yang.LeafEntry,
				Name: "int16Key",
				Type: &yang.YangType{Kind: yang.Yint16},
			},
			"int32Key": {
				Kind: yang.LeafEntry,
				Name: "int32Key",
				Type: &yang.YangType{Kind: yang.Yint32},
			},
			"int64Key": {
				Kind: yang.LeafEntry,
				Name: "int64Key",
				Type: &yang.YangType{Kind: yang.Yint64},
			},
			"uint8Key": {
				Kind: yang.LeafEntry,
				Name: "uint8Key",
				Type: &yang.YangType{Kind: yang.Yuint8},
			},
			"uint16Key": {
				Kind: yang.LeafEntry,
				Name: "uint16Key",
				Type: &yang.YangType{Kind: yang.Yuint16},
			},
			"uint32Key": {
				Kind: yang.LeafEntry,
				Name: "uint32Key",
				Type: &yang.YangType{Kind: yang.Yuint32},
			},
			"uint64Key": {
				Kind: yang.LeafEntry,
				Name: "uint64Key",
				Type: &yang.YangType{Kind: yang.Yuint64},
			},
			"decimal64Key": {
				Kind: yang.LeafEntry,
				Name: "decimal64Key",
				Type: &yang.YangType{Kind: yang.Ydecimal64},
			},
			"boolKey": {
				Kind: yang.LeafEntry,
				Name: "boolKey",
				Type: &yang.YangType{Kind: yang.Ybool},
			},
			"binaryKey": {
				Kind: yang.LeafEntry,
				Name: "binaryKey",
				Type: &yang.YangType{Kind: yang.Ybinary},
			},
			"enumKey": {
				Kind: yang.LeafEntry,
				Name: "enumKey",
				Type: &yang.YangType{Kind: yang.Yenum},
			},
			"leafrefKey": {
				Kind: yang.LeafEntry,
				Name: "leafrefKey",
				Type: &yang.YangType{Kind: yang.Yleafref, Path: "../uint64Key"},
			},
			"leafrefToLeafrefKey": {
				Kind: yang.LeafEntry,
				Name: "leafrefToLeafrefKey",
				Type: &yang.YangType{Kind: yang.Yleafref, Path: "../leafrefKey"},
			},
			"leafrefToUnionKey": {
				Kind: yang.LeafEntry,
				Name: "leafrefToUnionKey",
				Type: &yang.YangType{Kind: yang.Yleafref, Path: "../unionKey"},
			},
			"unionKey": {
				Kind: yang.LeafEntry,
				Name: "unionKey",
				Type: &yang.YangType{
					Kind: yang.Yunion,
					Type: []*yang.YangType{
						{
							Name: "enum-type",
							Kind: yang.Yenum,
						},
						{
							Name: "string",
							Kind: yang.Ystring,
						},
						{
							Name: "int16",
							Kind: yang.Yint16,
						},
					},
				},
			},
			"unionLoneTypeKey": {
				Kind: yang.LeafEntry,
				Name: "unionLoneTypeKey",
				Type: &yang.YangType{
					Kind: yang.Yunion,
					Type: []*yang.YangType{
						{
							Name: "uint32",
							Kind: yang.Yuint32,
						},
					},
				},
			},
			"leafrefToUnionKeySimple": {
				Kind: yang.LeafEntry,
				Name: "leafrefToUnionKeySimple",
				Type: &yang.YangType{Kind: yang.Yleafref, Path: "../unionKeySimple"},
			},
			"unionKeySimple": {
				Kind: yang.LeafEntry,
				Name: "unionKeySimple",
				Type: &yang.YangType{
					Kind: yang.Yunion,
					Type: []*yang.YangType{
						{
							Name: "enum-type",
							Kind: yang.Yenum,
						},
						{
							Name: "int16",
							Kind: yang.Yint16,
						},
						{
							Name: "int64",
							Kind: yang.Yint64,
						},
						{
							Name: "binary",
							Kind: yang.Ybinary,
						},
					},
				},
			},
		},
	}
	addParents(listSchema)

	tests := []struct {
		name             string
		inSchema         *yang.Entry
		inParent         interface{}
		inFieldName      string
		in               string
		want             interface{}
		wantErrSubstring string
	}{{
		name:        "string",
		inSchema:    listSchema.Dir["stringKey"],
		inParent:    &allKeysListStruct{},
		inFieldName: "StringKey",
		in:          "hello, world!",
		want:        "hello, world!",
	}, {
		name:        "int8",
		inSchema:    listSchema.Dir["int8Key"],
		inParent:    &allKeysListStruct{},
		inFieldName: "Int8Key",
		in:          "-123",
		want:        int8(-123),
	}, {
		name:        "int16",
		inSchema:    listSchema.Dir["int16Key"],
		inParent:    &allKeysListStruct{},
		inFieldName: "Int16Key",
		in:          "-1234",
		want:        int16(-1234),
	}, {
		name:        "int32",
		inSchema:    listSchema.Dir["int32Key"],
		inParent:    &allKeysListStruct{},
		inFieldName: "Int32Key",
		in:          "-1234",
		want:        int32(-1234),
	}, {
		name:        "int64",
		inSchema:    listSchema.Dir["int64Key"],
		inParent:    &allKeysListStruct{},
		inFieldName: "Int64Key",
		in:          "-1234",
		want:        int64(-1234),
	}, {
		name:        "uint8",
		inSchema:    listSchema.Dir["uint8Key"],
		inParent:    &allKeysListStruct{},
		inFieldName: "Uint8Key",
		in:          "123",
		want:        uint8(123),
	}, {
		name:        "uint16",
		inSchema:    listSchema.Dir["uint16Key"],
		inParent:    &allKeysListStruct{},
		inFieldName: "Uint16Key",
		in:          "1234",
		want:        uint16(1234),
	}, {
		name:        "uint32",
		inSchema:    listSchema.Dir["uint32Key"],
		inParent:    &allKeysListStruct{},
		inFieldName: "Uint32Key",
		in:          "1234",
		want:        uint32(1234),
	}, {
		name:        "uint64",
		inSchema:    listSchema.Dir["uint64Key"],
		inParent:    &allKeysListStruct{},
		inFieldName: "Uint64Key",
		in:          "1234",
		want:        uint64(1234),
	}, {
		name:        "decimal64",
		inSchema:    listSchema.Dir["decimal64Key"],
		inParent:    &allKeysListStruct{},
		inFieldName: "Decimal64Key",
		in:          "2.718281828",
		want:        float64(2.718281828),
	}, {
		name:        "bool (true)",
		inSchema:    listSchema.Dir["boolKey"],
		inParent:    &allKeysListStruct{},
		inFieldName: "BoolKey",
		in:          "true",
		want:        true,
	}, {
		name:             "invalid bool",
		inSchema:         listSchema.Dir["boolKey"],
		inParent:         &allKeysListStruct{},
		inFieldName:      "BoolKey",
		in:               "yes",
		wantErrSubstring: `cannot convert "yes" to bool`,
	}, {
		name:        "bool (false)",
		inSchema:    listSchema.Dir["boolKey"],
		inParent:    &allKeysListStruct{},
		inFieldName: "BoolKey",
		in:          "false",
		want:        false,
	}, {
		name:        "binary",
		inSchema:    listSchema.Dir["binaryKey"],
		inParent:    &allKeysListStruct{},
		inFieldName: "BinaryKey",
		in:          "NDI=",
		want:        []byte("42"),
	}, {
		name:        "union lone type",
		inSchema:    listSchema.Dir["unionLoneTypeKey"],
		inParent:    &allKeysListStruct{},
		inFieldName: "UnionLoneTypeKey",
		in:          "42",
		want:        uint32(42),
	}, {
		name:        "enum",
		inSchema:    listSchema.Dir["enumKey"],
		inParent:    &allKeysListStruct{},
		inFieldName: "EnumKey",
		in:          "E_VALUE_FORTY_TWO",
		want:        EnumType(42),
	}, {
		name:             "enum",
		inSchema:         listSchema.Dir["enumKey"],
		inParent:         allKeysListStruct{},
		inFieldName:      "EnumKey",
		in:               "E_VALUE_FORTY_TWO",
		wantErrSubstring: "is not a struct ptr",
	}, {
		name:        "union/enum",
		inSchema:    listSchema.Dir["unionKeySimple"],
		inParent:    &allKeysListStruct{},
		inFieldName: "UnionKeySimple",
		in:          "E_VALUE_FORTY_TWO",
		want:        EnumType(42),
	}, {
		name:             "union/enum",
		inSchema:         listSchema.Dir["unionKeySimple"],
		inParent:         &allKeysListStruct{},
		inFieldName:      "UnionKeySimple",
		in:               "E_VALUE_FORTY_TWO_NEMATODES",
		wantErrSubstring: `could not find suitable union type to unmarshal value "E_VALUE_FORTY_TWO_NEMATODES"`,
	}, {
		name:        "union/binary",
		inSchema:    listSchema.Dir["unionKeySimple"],
		inParent:    &allKeysListStruct{},
		inFieldName: "UnionKeySimple",
		in:          base64testStringEncoded,
		want:        testBinary,
	}, {
		name:        "union/int16",
		inSchema:    listSchema.Dir["unionKeySimple"],
		inParent:    &allKeysListStruct{},
		inFieldName: "UnionKeySimple",
		in:          "1234",
		want:        testutil.UnionInt16(1234),
	}, {
		name:        "union/enum (wrapper union)",
		inSchema:    listSchema.Dir["unionKey"],
		inParent:    &allKeysListStruct{},
		inFieldName: "UnionKey",
		in:          "E_VALUE_FORTY_TWO",
		want:        &Union1EnumType{EnumType(42)},
	}, {
		// NOTE: it would be non-deterministic to test int16.
		name:        "union/string (wrapper union)",
		inSchema:    listSchema.Dir["unionKey"],
		inParent:    &allKeysListStruct{},
		inFieldName: "UnionKey",
		in:          "1234-1234",
		want:        &Union1String{"1234-1234"},
	}, {
		name:        "leafref",
		inSchema:    listSchema.Dir["leafrefKey"],
		inParent:    &allKeysListStruct{},
		inFieldName: "LeafrefKey",
		in:          "1234",
		want:        uint64(1234),
	}, {
		name:        "leafref to leafref",
		inSchema:    listSchema.Dir["leafrefToLeafrefKey"],
		inParent:    &allKeysListStruct{},
		inFieldName: "LeafrefToLeafrefKey",
		in:          "2345",
		want:        uint64(2345),
	}, {
		name:        "leafref to union",
		inSchema:    listSchema.Dir["leafrefToUnionKeySimple"],
		inParent:    &allKeysListStruct{},
		inFieldName: "LeafrefToUnionKeySimple",
		in:          "1234",
		want:        testutil.UnionInt16(1234),
	}, {
		name:             "invalid: field name not part of union type",
		inSchema:         listSchema.Dir["unionKey"],
		inParent:         &allKeysListStruct{},
		inFieldName:      "NEKey",
		in:               "E_VALUE_FORTY_TWO",
		wantErrSubstring: `field "NEKey" not found in parent type`,
	}, {
		name:        "leafref to union (wrapper union)",
		inSchema:    listSchema.Dir["leafrefToUnionKey"],
		inParent:    &allKeysListStruct{},
		inFieldName: "LeafrefToUnionKey",
		in:          "2345-2345",
		want:        &Union1String{"2345-2345"},
	}, {
		name:             "invalid: struct is not a ptr when unmarshalling union",
		inSchema:         listSchema.Dir["unionKey"],
		inParent:         allKeysListStruct{},
		inFieldName:      "UnionKey",
		in:               "E_VALUE_FORTY_TWO",
		wantErrSubstring: "not a struct ptr",
	}, {
		name:             "invalid: field name not part of union type",
		inSchema:         listSchema.Dir["unionKey"],
		inParent:         &allKeysListStruct{},
		inFieldName:      "NEKey",
		in:               "E_VALUE_FORTY_TWO",
		wantErrSubstring: `field "NEKey" not found in parent type`,
	}, {
		name:             "invalid: string for float",
		inSchema:         listSchema.Dir["decimal64Key"],
		inParent:         &allKeysListStruct{},
		inFieldName:      "Decimal64Key",
		in:               "I am a float?",
		wantErrSubstring: "unable to convert",
	}, {
		name:             "invalid: too big for int8",
		inSchema:         listSchema.Dir["int8Key"],
		inParent:         &allKeysListStruct{},
		inFieldName:      "Int8Key",
		in:               "-1234",
		wantErrSubstring: "unable to convert",
	}, {
		name:             "invalid: negative for uint64",
		inSchema:         listSchema.Dir["uint64Key"],
		inParent:         &allKeysListStruct{},
		inFieldName:      "Uint64Key",
		in:               "-1234",
		wantErrSubstring: "unable to convert",
	}, {
		name:             "invalid: non-base64",
		inSchema:         listSchema.Dir["binaryKey"],
		inParent:         &allKeysListStruct{},
		inFieldName:      "BinaryKey",
		in:               "%",
		wantErrSubstring: "error in DecodeString",
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := stringToKeyType(tt.inSchema, tt.inParent, tt.inFieldName, tt.in)
			if diff := errdiff.Substring(err, tt.wantErrSubstring); diff != "" {
				t.Fatalf("got %v, want error %v", err, tt.wantErrSubstring)
			}
			if err != nil {
				return
			}
			if diff := cmp.Diff(v.Interface(), tt.want); diff != "" {
				t.Errorf("(-got, +want):\n%s", diff)
			}
		})
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
