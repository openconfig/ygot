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
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/kylelemons/godebug/pretty"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygot"
)

func typeToLeafSchema(name string, t yang.TypeKind) *yang.Entry {
	return &yang.Entry{
		Name: name,
		Kind: yang.LeafEntry,
		Type: &yang.YangType{
			Kind: t,
		},
	}
}

var (
	validLeafSchema = &yang.Entry{
		Name: "valid-leaf",
		Kind: yang.LeafEntry,
		Type: &yang.YangType{
			Kind: yang.Ystring,
		},
	}
	enumLeafSchema = &yang.Entry{
		Name: "enum-leaf",
		Kind: yang.LeafEntry,
		Type: &yang.YangType{
			Kind: yang.Yenum,
		},
	}
)

func TestValidateLeafSchema(t *testing.T) {
	tests := []struct {
		desc    string
		schema  *yang.Entry
		wantErr bool
	}{
		{
			desc:   "test success",
			schema: validLeafSchema,
		},
		{
			desc:    "test nil schema",
			schema:  nil,
			wantErr: true,
		},
		{
			desc:    "test nil schema type",
			schema:  &yang.Entry{Type: nil},
			wantErr: true,
		},
	}

	for _, test := range tests {
		err := validateLeafSchema(test.schema)
		if got, want := (err != nil), test.wantErr; got != want {
			t.Errorf("%s: validateLeafSchema(%v) got error: %v, wanted error? %v", test.desc, test.schema, err, test.wantErr)
		}
		testErrLog(t, test.desc, err)
	}
}

func TestValidateLeaf(t *testing.T) {
	tests := []struct {
		desc    string
		schema  *yang.Entry
		val     interface{}
		wantErr bool
	}{
		{
			desc:    "bad schema",
			schema:  nil,
			val:     ygot.String("value"),
			wantErr: true,
		},
		{
			desc:   "string success",
			schema: typeToLeafSchema("string", yang.Ystring),
			val:    ygot.String("value"),
		},
		{
			desc:   "string nil value success",
			schema: typeToLeafSchema("string", yang.Ystring),
			val:    nil,
		},
		{
			desc:    "string bad type",
			schema:  typeToLeafSchema("string", yang.Ystring),
			val:     ygot.Int32(1),
			wantErr: true,
		},
		{
			desc:    "string non ptr type",
			schema:  typeToLeafSchema("string", yang.Ystring),
			val:     int(1),
			wantErr: true,
		},
		{
			desc:   "int success",
			schema: typeToLeafSchema("int32", yang.Yint32),
			val:    ygot.Int32(1),
		},
		{
			desc:    "int bad type",
			schema:  typeToLeafSchema("int32", yang.Yint32),
			val:     ygot.String("value"),
			wantErr: true,
		},
		// TODO(mostrowski): restore when representation is decided.
		/*{
			desc:   "bitset success",
			schema: bitsetLeafSchema,
			val:    ygot.String("name1 name2"),
		}, {
			desc:    "bitset bad type",
			schema:  bitsetLeafSchema,
			val:     ygot.Int32(1),
			wantErr: true,
		}, */
		{
			desc:   "binary success",
			schema: typeToLeafSchema("binary", yang.Ybinary),
			val:    []byte("value"),
		},
		{
			desc:    "binary bad type",
			schema:  typeToLeafSchema("binary", yang.Ybinary),
			val:     ygot.Int32(1),
			wantErr: true,
		},
		{
			desc:   "bool success",
			schema: typeToLeafSchema("bool", yang.Ybool),
			val:    ygot.Bool(true),
		},
		{
			desc:    "bool bad type",
			schema:  typeToLeafSchema("bool", yang.Ybool),
			val:     ygot.Int32(1),
			wantErr: true,
		},
		{
			desc:   "decimal64 success",
			schema: typeToLeafSchema("decimal", yang.Ydecimal64),
			val:    ygot.Float64(42.42),
		},
		{
			desc:    "decimal64 bad type",
			schema:  typeToLeafSchema("decimal", yang.Ydecimal64),
			val:     ygot.String("four hundred and twenty two point eight"),
			wantErr: true,
		},
	}

	for _, test := range tests {
		err := Validate(test.schema, test.val)
		if got, want := (err != nil), test.wantErr; got != want {
			t.Errorf("%s: Validate(%v) got error: %v, wanted error? %v", test.desc, test.schema, err, test.wantErr)
		}
		testErrLog(t, test.desc, err)
	}
}

// UnionContainer and types below are defined outside function scope because
// type methods cannot be defined in function scope.
type UnionContainer struct {
	UnionField Union1 `path:"union1"`
}

func (*UnionContainer) IsYANGGoStruct() {}

type Union1 interface {
	IsUnion1()
}

type Union1String struct {
	String *string
}

func (Union1String) IsUnion1() {}

type Union1Int16 struct {
	Int16 *int16
}

func (Union1Int16) IsUnion1() {}

type Union1EnumType struct {
	EnumType EnumType
}

func (Union1EnumType) IsUnion1() {}

type Union1BadLeaf struct {
	BadLeaf *float32
}

func (Union1BadLeaf) IsUnion1() {}

type UnionContainerCompressed struct {
	UnionField *string `path:"union1"`
}

func (*UnionContainerCompressed) IsYANGGoStruct() {}

func TestValidateLeafUnion(t *testing.T) {
	unionContainerSchema := &yang.Entry{
		Name: "union1-container",
		Kind: yang.DirectoryEntry,
		Dir: map[string]*yang.Entry{
			"union1": {
				Name: "union1",
				Kind: yang.LeafEntry,
				Type: &yang.YangType{
					Name: "union1-type",
					Kind: yang.Yunion,
					Type: []*yang.YangType{
						{
							Name:    "string",
							Kind:    yang.Ystring,
							Pattern: []string{"a+"},
						},
						{
							Name: "int16",
							Kind: yang.Yint16,
						},
						{
							Name: "enum",
							Kind: yang.Yenum,
						},
					},
				},
			},
		},
	}

	// This schema has a data tree that does not define wrappers for the union
	// choices because the types are all the same.
	unionContainerSchemaNoWrappingStruct := &yang.Entry{
		Name: "union1-container",
		Kind: yang.DirectoryEntry,
		Dir: map[string]*yang.Entry{
			"union1": {
				Name: "union1",
				Kind: yang.LeafEntry,
				Type: &yang.YangType{
					Kind: yang.Yunion,
					Type: []*yang.YangType{
						{
							Name:    "string",
							Kind:    yang.Ystring,
							Pattern: []string{"a+"},
						},
						{
							Name:    "int16",
							Kind:    yang.Ystring,
							Pattern: []string{"b+"},
						},
					},
				},
			},
		},
	}

	unionContainerSchemaRecursive := &yang.Entry{
		Name: "union1-container",
		Kind: yang.DirectoryEntry,
		Dir: map[string]*yang.Entry{
			"union1": {
				Name: "union1",
				Kind: yang.LeafEntry,
				Type: &yang.YangType{
					Kind: yang.Yunion,
					Type: []*yang.YangType{
						{
							Name:    "string",
							Kind:    yang.Ystring,
							Pattern: []string{"a+"},
						},
						{
							Name:    "int16",
							Kind:    yang.Ystring,
							Pattern: []string{"b+"},
						},
						{
							Name: "bad-leaf",
							Kind: yang.Yunion,
							Type: []*yang.YangType{
								{
									Name:    "bad-leaf",
									Kind:    yang.Ystring,
									Pattern: []string{"c+"},
								},
							},
						},
					},
				},
			},
		},
	}

	tests := []struct {
		desc    string
		schema  *yang.Entry
		val     interface{}
		wantErr bool
	}{
		{
			desc:   "success string",
			schema: unionContainerSchema,
			val:    &UnionContainer{UnionField: &Union1String{String: ygot.String("aaa")}},
		},
		{
			desc:   "success int16",
			schema: unionContainerSchema,
			val:    &UnionContainer{UnionField: &Union1Int16{Int16: ygot.Int16(1)}},
		},
		{
			desc:   "success enum",
			schema: unionContainerSchema,
			val:    &UnionContainer{UnionField: &Union1EnumType{EnumType: 42}},
		},
		{
			desc:    "bad regex",
			schema:  unionContainerSchema,
			val:     &UnionContainer{UnionField: &Union1String{String: ygot.String("bbb")}},
			wantErr: true,
		},
		{
			desc:    "bad type",
			schema:  unionContainerSchema,
			val:     &UnionContainer{UnionField: &Union1BadLeaf{BadLeaf: ygot.Float32(0)}},
			wantErr: true,
		},
		{
			desc:   "success no wrapping struct string",
			schema: unionContainerSchemaNoWrappingStruct,
			val:    &UnionContainerCompressed{UnionField: ygot.String("aaa")},
		},
		{
			desc:   "success no wrapping struct int16",
			schema: unionContainerSchemaNoWrappingStruct,
			val:    &UnionContainerCompressed{UnionField: ygot.String("bbb")},
		},
		{
			desc:   "success no wrapping struct string",
			schema: unionContainerSchemaNoWrappingStruct,
			val:    &UnionContainerCompressed{UnionField: ygot.String("aaa")},
		},
		{
			desc:    "no wrapping struct no schemas match",
			schema:  unionContainerSchemaNoWrappingStruct,
			val:     &UnionContainerCompressed{UnionField: ygot.String("ccc")},
			wantErr: true,
		},
		{
			desc:   "recursive union success",
			schema: unionContainerSchemaRecursive,
			val:    &UnionContainerCompressed{UnionField: ygot.String("ccc")},
		},
		{
			desc:    "recursive union no match",
			schema:  unionContainerSchemaRecursive,
			val:     &UnionContainerCompressed{UnionField: ygot.String("ddd")},
			wantErr: true,
		},
	}

	for _, test := range tests {
		err := Validate(test.schema, test.val)
		if got, want := (err != nil), test.wantErr; got != want {
			t.Errorf("%s: got error: %v, wanted error? %v", test.desc, err, test.wantErr)
		}
		testErrLog(t, test.desc, err)
	}
}

type Leaf1Container struct {
	Leaf1 *string `path:"container1/leaf1"`
	Leaf2 *string `path:"container1/leaf2"`
	Leaf3 *string `path:"container1/leaf3"`
	Leaf4 *string `path:"container1/leaf4"`
}

func (*Leaf1Container) IsYANGGoStruct() {}

type PredicateSchema struct {
	List      map[string]*PredicateSchemaList `path:"list"`
	Value     *string                         `path:"value"`
	Reference *string                         `path:"reference"`
}

func (*PredicateSchema) IsYANGGoStruct() {}

type PredicateSchemaList struct {
	Key *string `path:"key"`
}

func (*PredicateSchemaList) IsYANGGoStruct() {}

func TestValidateLeafRef(t *testing.T) {
	validDeviceSchema := &yang.Entry{
		Name: "device",
		Kind: yang.DirectoryEntry,
	}
	validContainerSchema := &yang.Entry{
		Name:   "container1",
		Parent: validDeviceSchema,
		Kind:   yang.DirectoryEntry,
	}
	validContainerSchema.Dir = map[string]*yang.Entry{
		"config": {
			Parent: validContainerSchema,
			Dir: map[string]*yang.Entry{
				"leaf-type": {
					Kind: yang.LeafEntry,
					Name: "leaf-type",
					Type: &yang.YangType{Kind: yang.Ystring, Pattern: []string{"a+"}},
				},
			},
		},
		"leaf1": {
			Parent: validContainerSchema,
			Kind:   yang.LeafEntry,
			Name:   "leaf1",
			Type: &yang.YangType{
				Kind: yang.Yleafref,
				Path: "../config/leaf-type",
			},
		},
		"leaf2": {
			Parent: validContainerSchema,
			Kind:   yang.LeafEntry,
			Name:   "leaf2",
			Type: &yang.YangType{
				Kind: yang.Yleafref,
				Path: "/container1/config/leaf-type",
			},
		},
		"leaf3": {
			Parent: validContainerSchema,
			Kind:   yang.LeafEntry,
			Name:   "leaf3",
			Type: &yang.YangType{
				Kind: yang.Yleafref,
				Path: "../pfx:config/pfx:leaf-type",
			},
		},
		"leaf4": {
			Parent: validContainerSchema,
			Kind:   yang.LeafEntry,
			Name:   "leaf4",
			Type: &yang.YangType{
				Kind: yang.Yleafref,
				Path: "/pfx:container1/pfx:config/pfx:leaf-type",
			},
		},
	}
	validDeviceSchema.Dir = map[string]*yang.Entry{
		"container1": validContainerSchema,
	}

	badContainerSchema := &yang.Entry{
		Name: "container1",
		Kind: yang.DirectoryEntry,
	}
	badContainerSchema.Dir = map[string]*yang.Entry{
		"config": {
			Parent: badContainerSchema,
			Dir: map[string]*yang.Entry{
				"leaf-type": {
					Kind: yang.LeafEntry,
					Name: "leaf-type",
					Type: &yang.YangType{Kind: yang.Ystring, Pattern: []string{"a+"}},
				},
			},
		},
		"leaf1": {
			Parent: badContainerSchema,
			Kind:   yang.LeafEntry,
			Name:   "leaf1",
			Type: &yang.YangType{
				Kind: yang.Yleafref,
				Path: "../config/bad-path",
			},
		},
	}

	missingParentContainerSchema := &yang.Entry{
		Name: "container1",
		Kind: yang.DirectoryEntry,
	}
	missingParentContainerSchema.Dir = map[string]*yang.Entry{
		"config": {
			Dir: map[string]*yang.Entry{
				"leaf-type": {
					Kind: yang.LeafEntry,
					Name: "leaf-type",
					Type: &yang.YangType{Kind: yang.Ystring, Pattern: []string{"a+"}},
				},
			},
		},
		"leaf1": {
			Kind: yang.LeafEntry,
			Name: "leaf1",
			Type: &yang.YangType{
				Kind: yang.Yleafref,
				Path: "../config/leaf-type",
			},
		},
	}

	invalidLeafrefPathContainerSchema := &yang.Entry{
		Name: "container1",
		Kind: yang.DirectoryEntry,
	}
	invalidLeafrefPathContainerSchema.Dir = map[string]*yang.Entry{
		"config": {
			Dir: map[string]*yang.Entry{
				"leaf-type": {
					Kind: yang.LeafEntry,
					Name: "leaf-type",
					Type: &yang.YangType{Kind: yang.Ystring},
				},
			},
		},
		"leaf1": {
			Kind:   yang.LeafEntry,
			Name:   "leaf1",
			Parent: validContainerSchema,
			Type: &yang.YangType{
				Kind: yang.Yleafref,
				Path: "../foo:bar:config/leaf-type",
			},
		},
	}

	fakeRootWithList := &yang.Entry{
		Name:       "device",
		Kind:       yang.DirectoryEntry,
		Annotation: map[string]interface{}{"isFakeRoot": true},
	}
	fakeRootContainerSchema := &yang.Entry{
		Name:   "container1",
		Kind:   yang.DirectoryEntry,
		Parent: fakeRootWithList,
	}
	fakeRootContainerSchema.Dir = map[string]*yang.Entry{
		"leaf1": {
			Name:   "leaf1",
			Parent: fakeRootContainerSchema,
			Type: &yang.YangType{
				Kind: yang.Yleafref,
				Path: "/container/list/key",
			},
		},
	}
	fakeRootWithList.Dir = map[string]*yang.Entry{
		"container": {
			Name: "container",
			Kind: yang.DirectoryEntry,
			Dir: map[string]*yang.Entry{
				"list": {
					Name: "list",
					Kind: yang.DirectoryEntry,
					Key:  "key",
					Dir: map[string]*yang.Entry{
						"key": {
							Name: "key",
							Type: &yang.YangType{
								Kind:    yang.Ystring,
								Pattern: []string{"b.*"},
							},
						},
					},
					ListAttr: &yang.ListAttr{},
				},
			},
		},
		"container1": fakeRootContainerSchema,
	}

	predicateSchema := &yang.Entry{
		Name: "container",
		Kind: yang.DirectoryEntry,
	}
	predicateSchema.Dir = map[string]*yang.Entry{
		"list": {
			Name: "list",
			Kind: yang.DirectoryEntry,
			Key:  "key",
			Dir: map[string]*yang.Entry{
				"key": {
					Name: "key",
					Type: &yang.YangType{
						Kind:    yang.Ystring,
						Pattern: []string{"b.*"},
					},
				},
			},
			Parent: predicateSchema,
		},
		"value": {
			Name:   "value",
			Type:   &yang.YangType{Kind: yang.Ystring},
			Parent: predicateSchema,
		},
		"reference": {
			Name: "value",
			Type: &yang.YangType{
				Kind: yang.Yleafref,
				Path: `/list[key="current()/../value"]/key`,
			},
			Parent: predicateSchema,
		},
	}

	tests := []struct {
		desc    string
		schema  *yang.Entry
		val     interface{}
		wantErr bool
	}{
		{
			desc:   "success relative leaf-ref",
			schema: validContainerSchema,
			val:    &Leaf1Container{Leaf1: ygot.String("aaa")},
		},
		{
			desc:   "success absolute leaf-ref",
			schema: validContainerSchema,
			val:    &Leaf1Container{Leaf2: ygot.String("aaa")},
		},
		{
			desc:   "success relative leafref with prefixed path",
			schema: validContainerSchema,
			val:    &Leaf1Container{Leaf3: ygot.String("aaa")},
		},
		{
			desc:   "success absolute leafref with prefixed path",
			schema: validContainerSchema,
			val:    &Leaf1Container{Leaf4: ygot.String("aaa")},
		},
		{
			desc:   "success absolute leafref with fakeroot",
			schema: fakeRootContainerSchema,
			val:    &Leaf1Container{Leaf1: ygot.String("bbb")},
		},
		{
			desc:    "bad value",
			schema:  validContainerSchema,
			val:     &Leaf1Container{Leaf1: ygot.String("bbb")},
			wantErr: true,
		},
		{
			desc:    "bad schema",
			schema:  badContainerSchema,
			val:     &Leaf1Container{Leaf1: ygot.String("aaa")},
			wantErr: true,
		},
		{
			desc:    "missing parent in schema",
			schema:  missingParentContainerSchema,
			val:     &Leaf1Container{Leaf1: ygot.String("aaa")},
			wantErr: true,
		},
		{
			desc:   "predicate in path",
			schema: predicateSchema,
			val: &PredicateSchema{
				Reference: ygot.String("baz"),
			},
		},
		{
			desc:   "predicate bad value",
			schema: predicateSchema,
			val: &PredicateSchema{
				Reference: ygot.String("aardvark"),
			},
			wantErr: true,
		},
	}

	for _, test := range tests {
		err := Validate(test.schema, test.val)
		if got, want := (err != nil), test.wantErr; got != want {
			t.Errorf("%s: got error: %v, wanted error? %v", test.desc, err, test.wantErr)
		}
		testErrLog(t, test.desc, err)
	}
}

func TestRemoveXPATHPredicates(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{{
		name: "simple predicate",
		in:   `/foo/bar[name="eth0"]`,
		want: "/foo/bar",
	}, {
		name: "predicate with path",
		in:   `/foo/bar[name="/foo/bar/baz"]/config/hat`,
		want: "/foo/bar/config/hat",
	}, {
		name: "predicate with function",
		in:   `/foo/bar[name="current()/../interface"]/config/baz`,
		want: "/foo/bar/config/baz",
	}, {
		name: "multiple predicates",
		in:   `/foo/bar[name="current()/../interface"]/container/list[key="42"]/config/foo`,
		want: "/foo/bar/container/list/config/foo",
	}, {
		name:    "] without [",
		in:      `/foo/bar]`,
		wantErr: true,
	}, {
		name:    "[ without closure",
		in:      `/foo/bar[`,
		wantErr: true,
	}, {
		name: "multiple predicates, end of string",
		in:   `/foo/bar/name[e="1"]/bar[j="2"]`,
		want: "/foo/bar/name/bar",
	}, {
		name:    "][ in incorrect order",
		in:      `/foo/bar][`,
		wantErr: true,
	}, {
		name: "empty string",
		in:   ``,
		want: ``,
	}, {
		name: "predicate directly",
		in:   `foo[bar="test"]`,
		want: `foo`,
	}}

	for _, tt := range tests {
		got, err := removeXPATHPredicates(tt.in)
		if (err != nil) != tt.wantErr {
			t.Errorf("%s: removeXPATHPredicates(%s): got unexpected error, got: %v", tt.name, tt.in, err)
		}

		if got != tt.want {
			t.Errorf("%s: removePredicate(%v): did not get expected value, got: %v, want: %v", tt.name, tt.in, got, tt.want)
		}
	}
}

type LeafContainerStruct struct {
	Int8Leaf   *int8         `path:"int8-leaf"`
	Uint8Leaf  *uint8        `path:"uint8-leaf"`
	Int16Leaf  *int16        `path:"int16-leaf"`
	Uint16Leaf *uint16       `path:"uint16-leaf"`
	Int32Leaf  *int32        `path:"int32-leaf"`
	Uint32Leaf *uint32       `path:"uint32-leaf"`
	Int64Leaf  *int64        `path:"int64-leaf"`
	Uint64Leaf *uint64       `path:"uint64-leaf"`
	StringLeaf *string       `path:"string-leaf"`
	BinaryLeaf []byte        `path:"binary-leaf"`
	BoolLeaf   *bool         `path:"bool-leaf"`
	EnumLeaf   EnumType      `path:"enum-leaf"`
	UnionLeaf  UnionLeafType `path:"union-leaf"`
}

type UnionLeafType interface {
	Is_UnionLeafType()
}

type UnionLeafType_String struct {
	String string
}

func (*UnionLeafType_String) Is_UnionLeafType() {}

type UnionLeafType_Uint32 struct {
	Uint32 uint32
}

func (*UnionLeafType_Uint32) Is_UnionLeafType() {}

type UnionLeafType_EnumType struct {
	EnumType EnumType
}

func (*UnionLeafType_EnumType) Is_UnionLeafType() {}

func (*UnionLeafType_EnumType) ΛMap() map[string]map[int64]ygot.EnumDefinition {
	return globalEnumMap
}

func (*LeafContainerStruct) ΛEnumTypeMap() map[string][]reflect.Type {
	return map[string][]reflect.Type{
		"/union-leaf": []reflect.Type{reflect.TypeOf(UnionLeafType_EnumType{})},
	}
}

func (*LeafContainerStruct) To_UnionLeafType(i interface{}) (UnionLeafType, error) {
	switch v := i.(type) {
	case string:
		return &UnionLeafType_String{v}, nil
	case uint32:
		return &UnionLeafType_Uint32{v}, nil
	case EnumType:
		return &UnionLeafType_EnumType{v}, nil
	default:
		return nil, fmt.Errorf("cannot convert %v to To_UnionLeafType, unknown union type, got: %T, want any of [string, uint32]", i, i)
	}
}

func TestUnmarshalLeaf(t *testing.T) {
	tests := []struct {
		desc    string
		schema  *yang.Entry
		json    string
		want    LeafContainerStruct
		wantErr string
	}{
		{
			desc: "int8 success",
			json: `{"int8-leaf" : -42}`,
			want: LeafContainerStruct{Int8Leaf: ygot.Int8(-42)},
		},
		{
			desc: "uint8 success",
			json: `{"uint8-leaf" : 42}`,
			want: LeafContainerStruct{Uint8Leaf: ygot.Uint8(42)},
		},
		{
			desc: "int16 success",
			json: `{"int16-leaf" : -42}`,
			want: LeafContainerStruct{Int16Leaf: ygot.Int16(-42)},
		},
		{
			desc: "uint16 success",
			json: `{"uint16-leaf" : 42}`,
			want: LeafContainerStruct{Uint16Leaf: ygot.Uint16(42)},
		},
		{
			desc: "int32 success",
			json: `{"int32-leaf" : -42}`,
			want: LeafContainerStruct{Int32Leaf: ygot.Int32(-42)},
		},
		{
			desc: "uint32 success",
			json: `{"uint32-leaf" : 42}`,
			want: LeafContainerStruct{Uint32Leaf: ygot.Uint32(42)},
		},
		{
			desc: "int64 success",
			json: `{"int64-leaf" : "-42"}`,
			want: LeafContainerStruct{Int64Leaf: ygot.Int64(-42)},
		},
		{
			desc: "uint64 success",
			json: `{"uint64-leaf" : "42"}`,
			want: LeafContainerStruct{Uint64Leaf: ygot.Uint64(42)},
		},
		{
			// TODO DEBUG 8
			desc: "enum success",
			json: `{"enum-leaf" : "E_VALUE_FORTY_TWO"}`,
			want: LeafContainerStruct{EnumLeaf: 42},
		},
		{
			desc: "union string success",
			json: `{"union-leaf" : "forty-two"}`,
			want: LeafContainerStruct{UnionLeaf: &UnionLeafType_String{String: "forty-two"}},
		},
		{
			desc: "union uint32 success",
			json: `{"union-leaf" : 42}`,
			want: LeafContainerStruct{UnionLeaf: &UnionLeafType_Uint32{Uint32: 42}},
		},
		{
			// TODO DEBUG 11
			desc: "union enum success",
			json: `{"union-leaf" : "E_VALUE_FORTY_TWO"}`,
			want: LeafContainerStruct{UnionLeaf: &UnionLeafType_EnumType{EnumType: 42}},
		},
		{
			desc:    "int32 bad type",
			json:    `{"int32-leaf" : "-42"}`,
			wantErr: `got string type for field int32-leaf, expect float64`,
		},
		{
			desc:    "uint32 bad type",
			json:    `{"uint32-leaf" : "42"}`,
			wantErr: `got string type for field uint32-leaf, expect float64`,
		},
		{
			desc:    "int64 bad type",
			json:    `{"int64-leaf" : -42}`,
			wantErr: `got float64 type for field int64-leaf, expect string`,
		},
		{
			desc:    "int8 out of range",
			json:    `{"int8-leaf" : -129}`,
			wantErr: `error parsing -129 for schema int8-leaf: value -129 falls outside the int range [-128, 127]`,
		},
		{
			desc:    "uint8 out of range",
			json:    `{"uint8-leaf" : -42}`,
			wantErr: `error parsing -42 for schema uint8-leaf: value -42 falls outside the int range [0, 255]`,
		},
		{
			desc:    "int16 out of range",
			json:    `{"int16-leaf" : -32769}`,
			wantErr: `error parsing -32769 for schema int16-leaf: value -32769 falls outside the int range [-32768, 32767]`,
		},
		{
			desc:    "uint16 out of range",
			json:    `{"uint16-leaf" : -42}`,
			wantErr: `error parsing -42 for schema uint16-leaf: value -42 falls outside the int range [0, 65535]`,
		},
		{
			desc:    "int32 out of range",
			json:    `{"int32-leaf" : -2147483649}`,
			wantErr: `error parsing -2.147483649e+09 for schema int32-leaf: value -2147483649 falls outside the int range [-2147483648, 2147483647]`,
		},
		{
			desc:    "uint32 out of range",
			json:    `{"uint32-leaf" : -42}`,
			wantErr: `error parsing -42 for schema uint32-leaf: value -42 falls outside the int range [0, 4294967295]`,
		},
		{
			desc:    "int64 out of range",
			json:    `{"int64-leaf" : "-9223372036854775809"}`,
			wantErr: `error parsing -9223372036854775809 for schema int64-leaf: strconv.ParseInt: parsing "-9223372036854775809": value out of range`,
		},
		{
			desc:    "uint64 out of range",
			json:    `{"uint64-leaf" : "-42"}`,
			wantErr: `error parsing -42 for schema uint64-leaf: strconv.ParseUint: parsing "-42": invalid syntax`,
		},
		{
			desc:    "enum bad value",
			json:    `{"enum-leaf" : "E_BAD_VALUE"}`,
			wantErr: `E_BAD_VALUE is not a valid value for enum field ytypes.EnumType`,
		},
		{
			desc:    "union bad type",
			json:    `{"union-leaf" : -42}`,
			wantErr: `could not find suitable union type to unmarshal value -42 type float64 into parent struct type *ytypes.LeafContainerStruct field UnionLeaf`,
		},
	}

	containerSchema := &yang.Entry{
		Name: "container-schema",
		Kind: yang.DirectoryEntry,
		Dir:  make(map[string]*yang.Entry),
	}

	unionSchema := &yang.Entry{
		Name: "union-leaf",
		Kind: yang.LeafEntry,
		Type: &yang.YangType{
			Kind: yang.Yunion,
			Type: []*yang.YangType{
				{
					Kind: yang.Ystring,
				},
				{
					Kind: yang.Yuint32,
				},
				{
					Kind: yang.Yidentityref,
					Path: "../enum-leaf",
				},
			},
		},
	}

	var leafSchemas = []*yang.Entry{
		typeToLeafSchema("int8-leaf", yang.Yint8),
		typeToLeafSchema("uint8-leaf", yang.Yuint8),
		typeToLeafSchema("int16-leaf", yang.Yint16),
		typeToLeafSchema("uint16-leaf", yang.Yuint16),
		typeToLeafSchema("int32-leaf", yang.Yint32),
		typeToLeafSchema("uint32-leaf", yang.Yuint32),
		typeToLeafSchema("int64-leaf", yang.Yint64),
		typeToLeafSchema("uint64-leaf", yang.Yuint64),
		typeToLeafSchema("string-leaf", yang.Ystring),
		typeToLeafSchema("binary-leaf", yang.Ybinary),
		typeToLeafSchema("bool-leaf", yang.Ybool),
		enumLeafSchema,
		unionSchema,
	}
	for _, s := range leafSchemas {
		s.Parent = containerSchema
		containerSchema.Dir[s.Name] = s
	}

	var jsonTree interface{}
	// TODO DEBUG REMOVE
	for _, test := range tests {
		var parent LeafContainerStruct

		if err := json.Unmarshal([]byte(test.json), &jsonTree); err != nil {
			t.Fatal(fmt.Sprintf("%s : %s", test.desc, err))
		}

		err := Unmarshal(containerSchema, &parent, jsonTree)
		if got, want := errToString(err), test.wantErr; got != want {
			t.Errorf("%s: Unmarshal got error: %v, wanted error? %v", test.desc, got, want)
		}
		testErrLog(t, test.desc, err)
		if err == nil {
			if got, want := parent, test.want; !reflect.DeepEqual(got, want) {
				t.Errorf("%s: Unmarshal got:\n%v\nwant:\n%v\n", test.desc, pretty.Sprint(got), pretty.Sprint(want))
			}
		}
	}
}

func TestUnmarshalLeafRef(t *testing.T) {
	containerSchema := &yang.Entry{
		Name: "container",
		Kind: yang.DirectoryEntry,
	}
	containerSchema.Dir = map[string]*yang.Entry{
		"config": {
			Parent: containerSchema,
			Dir: map[string]*yang.Entry{
				"leaf-type": {
					Kind: yang.LeafEntry,
					Name: "leaf-type",
					Type: &yang.YangType{Kind: yang.Yint32},
				},
			},
		},
		"leaf1": {
			Parent: containerSchema,
			Kind:   yang.LeafEntry,
			Name:   "leaf1",
			Type: &yang.YangType{
				Kind: yang.Yleafref,
				Path: "../config/leaf-type",
			},
		},
	}

	type ContainerStruct struct {
		Leaf1 *int32 `path:"leaf1"`
	}

	tests := []struct {
		desc    string
		json    string
		want    ContainerStruct
		wantErr string
	}{
		{
			desc: "success",
			json: `{ "leaf1" : 42}`,
			want: ContainerStruct{Leaf1: ygot.Int32(42)},
		},
		{
			desc:    "bad field name",
			json:    `{ "bad-field" : 42}`,
			wantErr: `parent container container (type *ytypes.ContainerStruct): JSON contains unexpected field bad-field`,
		},
	}

	var jsonTree interface{}
	for _, test := range tests {
		var parent ContainerStruct

		if err := json.Unmarshal([]byte(test.json), &jsonTree); err != nil {
			t.Fatal(fmt.Sprintf("%s : %s", test.desc, err))
		}

		err := Unmarshal(containerSchema, &parent, jsonTree)
		if got, want := errToString(err), test.wantErr; got != want {
			t.Errorf("%s: Unmarshal got error: %v, wanted error? %v", test.desc, got, want)
		}
		testErrLog(t, test.desc, err)
		if err == nil {
			if got, want := parent, test.want; !reflect.DeepEqual(got, want) {
				t.Errorf("%s: Unmarshal got:\n%v\nwant:\n%v\n", test.desc, pretty.Sprint(got), pretty.Sprint(want))
			}
		}
	}
}
