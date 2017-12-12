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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/kylelemons/godebug/pretty"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygot"
)

var (
	base64testString        = "forty two"
	base64testStringEncoded = base64.StdEncoding.EncodeToString([]byte(base64testString))
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
		{
			desc: "test bad schema type",
			schema: &yang.Entry{
				Kind: yang.DirectoryEntry,
				Type: &yang.YangType{
					Kind: yang.Ystring,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := validateLeafSchema(tt.schema)
			if got, want := (err != nil), tt.wantErr; got != want {
				t.Errorf("%s: validateLeafSchema(%v) got error: %v, want error? %v", tt.desc, tt.schema, err, tt.wantErr)
			}
			testErrLog(t, tt.desc, err)
		})
	}
}

// YANGEmpty is a derived type which is used to represent the YANG
// empty type.
type YANGEmpty bool

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
		{
			desc:   "empty type",
			schema: typeToLeafSchema("empty", yang.Yempty),
			val:    YANGEmpty(true),
		},
		{
			desc:    "bad empty type",
			schema:  typeToLeafSchema("empty", yang.Yempty),
			val:     "string",
			wantErr: true,
		},
		{
			desc:    "slice in non-binary type",
			schema:  typeToLeafSchema("binary", yang.Ystring),
			val:     []byte{1, 2, 3},
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
		{
			desc:   "enum success",
			schema: typeToLeafSchema("enum", yang.Yenum),
			val:    int64(0),
		},
		{
			desc:    "enum bad type",
			schema:  typeToLeafSchema("enum", yang.Yenum),
			val:     int(0),
			wantErr: true,
		},
		{
			desc:   "identityref success",
			schema: typeToLeafSchema("identityref", yang.Yidentityref),
			val:    int64(0),
		},
		{
			desc:    "identityref bad type",
			schema:  typeToLeafSchema("identityref", yang.Yidentityref),
			val:     int(0),
			wantErr: true,
		},
		{
			desc:   "empty success",
			schema: typeToLeafSchema("empty", yang.Yempty),
			val:    YANGEmpty(true),
		},
		{
			desc:    "empty bad type",
			schema:  typeToLeafSchema("empty", yang.Yempty),
			val:     ygot.Int32(1),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			errs := Validate(tt.schema, tt.val)
			if got, want := (errs != nil), tt.wantErr; got != want {
				t.Errorf("%s: Validate(%v) got error: %v, want error? %v", tt.desc, tt.schema, errs, tt.wantErr)
			}
			testErrLog(t, tt.desc, errs)
		})
	}

	// Additional tests through private API.
	if err := validateLeaf(nil, nil); err != nil {
		t.Errorf("nil value: got error: %v, want error: nil", err)
	}
	if err := validateLeaf(nil, 42); err == nil {
		t.Errorf("nil schema: got error: nil, want nil schema error")
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

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			errs := Validate(tt.schema, tt.val)
			if got, want := (errs != nil), tt.wantErr; got != want {
				t.Errorf("%s: got error: %v, want error? %v", tt.desc, errs, tt.wantErr)
			}
			testErrLog(t, tt.desc, errs)
		})
	}

	// Additional tests through private API.
	if err := validateUnion(unionContainerSchema, nil); err != nil {
		t.Errorf("nil value: got error: %v, want error: nil", err)
	}
	if err := validateUnion(unionContainerSchema, 42); err == nil {
		t.Errorf("bad value type: got error: nil, want type error")
	}
}

type Leaf1Container struct {
	Leaf1 *string `path:"container1/leaf1"`
	Leaf2 *string `path:"container1/leaf2"`
	Leaf3 *string `path:"container1/leaf3"`
	Leaf4 *string `path:"container1/leaf4"`
	Leaf5 *string `path:"leaf5"`
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
		"container2": {
			Name: "container2",
			Kind: yang.DirectoryEntry,
			Dir: map[string]*yang.Entry{
				"leaf2": {
					Name: "leaf2",
					Type: &yang.YangType{
						Kind:    yang.Ystring,
						Pattern: []string{"b.*"},
					},
				},
			},
		},
		"leaf5": {
			Name:   "leaf5",
			Parent: fakeRootWithList,
			Type: &yang.YangType{
				Kind: yang.Yleafref,
				Path: "/list-compressed-out/container2/leaf2",
			},
		},
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

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			errs := Validate(tt.schema, tt.val)
			if got, want := (errs != nil), tt.wantErr; got != want {
				t.Errorf("%s: got error: %v, want error? %v", tt.desc, errs, tt.wantErr)
			}
			testErrLog(t, tt.desc, errs)
		})
	}
}

func TestRemoveXPATHPredicates(t *testing.T) {
	tests := []struct {
		desc    string
		in      string
		want    string
		wantErr bool
	}{{
		desc: "simple predicate",
		in:   `/foo/bar[name="eth0"]`,
		want: "/foo/bar",
	}, {
		desc: "predicate with path",
		in:   `/foo/bar[name="/foo/bar/baz"]/config/hat`,
		want: "/foo/bar/config/hat",
	}, {
		desc: "predicate with function",
		in:   `/foo/bar[name="current()/../interface"]/config/baz`,
		want: "/foo/bar/config/baz",
	}, {
		desc: "multiple predicates",
		in:   `/foo/bar[name="current()/../interface"]/container/list[key="42"]/config/foo`,
		want: "/foo/bar/container/list/config/foo",
	}, {
		desc:    "] without [",
		in:      `/foo/bar]`,
		wantErr: true,
	}, {
		desc:    "[ without closure",
		in:      `/foo/bar[`,
		wantErr: true,
	}, {
		desc: "multiple predicates, end of string",
		in:   `/foo/bar/name[e="1"]/bar[j="2"]`,
		want: "/foo/bar/name/bar",
	}, {
		desc:    "][ in incorrect order",
		in:      `/foo/bar][`,
		wantErr: true,
	}, {
		desc: "empty string",
		in:   ``,
		want: ``,
	}, {
		desc: "predicate directly",
		in:   `foo[bar="test"]`,
		want: `foo`,
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got, err := removeXPATHPredicates(tt.in)
			if (err != nil) != tt.wantErr {
				t.Errorf("%s: removeXPATHPredicates(%s): got unexpected error, got: %v", tt.desc, tt.in, err)
			}

			if got != tt.want {
				t.Errorf("%s: removePredicate(%v): did not get expected value, got: %v, want: %v", tt.desc, tt.in, got, tt.want)
			}
		})
	}
}

type LeafContainerStruct struct {
	Int8Leaf    *int8         `path:"int8-leaf"`
	Uint8Leaf   *uint8        `path:"uint8-leaf"`
	Int16Leaf   *int16        `path:"int16-leaf"`
	Uint16Leaf  *uint16       `path:"uint16-leaf"`
	Int32Leaf   *int32        `path:"int32-leaf"`
	Uint32Leaf  *uint32       `path:"uint32-leaf"`
	Int64Leaf   *int64        `path:"int64-leaf"`
	Uint64Leaf  *uint64       `path:"uint64-leaf"`
	StringLeaf  *string       `path:"string-leaf"`
	BinaryLeaf  []byte        `path:"binary-leaf"`
	BoolLeaf    *bool         `path:"bool-leaf"`
	DecimalLeaf *float64      `path:"decimal-leaf"`
	EnumLeaf    EnumType      `path:"enum-leaf"`
	UnionLeaf   UnionLeafType `path:"union-leaf"`
	UnionLeaf2  *string       `path:"union-leaf2"`
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

type UnionLeafType_EnumType2 struct {
	EnumType2 EnumType2
}

func (*UnionLeafType_EnumType2) Is_UnionLeafType() {}

func (*UnionLeafType_EnumType2) ΛMap() map[string]map[int64]ygot.EnumDefinition {
	return globalEnumMap
}

func (*LeafContainerStruct) ΛEnumTypeMap() map[string][]reflect.Type {
	return map[string][]reflect.Type{
		"/container-schema/union-leaf": {reflect.TypeOf(EnumType(0)), reflect.TypeOf(EnumType2(0))},
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
	case EnumType2:
		return &UnionLeafType_EnumType2{v}, nil
	default:
		return nil, fmt.Errorf("cannot convert %v to To_UnionLeafType, unknown union type, got: %T, want any of [string, uint32]", i, i)
	}
}

func TestUnmarshalLeaf(t *testing.T) {
	tests := []struct {
		desc    string
		json    string
		want    LeafContainerStruct
		wantErr string
	}{
		{
			desc: "nil success",
			json: `{}`,
			want: LeafContainerStruct{},
		},
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
			desc: "enum success",
			json: `{"enum-leaf" : "E_VALUE_FORTY_TWO"}`,
			want: LeafContainerStruct{EnumLeaf: 42},
		},
		{
			desc: "binary success",
			json: `{"binary-leaf" : "` + base64testStringEncoded + `"}`,
			want: LeafContainerStruct{BinaryLeaf: []byte(base64testString)},
		},
		{
			desc: "bool success",
			json: `{"bool-leaf" : true}`,
			want: LeafContainerStruct{BoolLeaf: ygot.Bool(true)},
		},
		{
			desc: "decimal success",
			json: `{"decimal-leaf" : "42.42"}`,
			want: LeafContainerStruct{DecimalLeaf: ygot.Float64(42.42)},
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
			desc: "union enum success",
			json: `{"union-leaf" : "E_VALUE_FORTY_TWO"}`,
			want: LeafContainerStruct{UnionLeaf: &UnionLeafType_EnumType{EnumType: 42}},
		},
		{
			desc: "union enum2 success",
			json: `{"union-leaf" : "E_VALUE_FORTY_THREE"}`,
			want: LeafContainerStruct{UnionLeaf: &UnionLeafType_EnumType2{EnumType2: 43}},
		},
		{
			desc: "union no struct success, correct type, value unvalidated",
			json: `{"union-leaf2" : "ccc"}`,
			want: LeafContainerStruct{UnionLeaf2: ygot.String("ccc")},
		},
		{
			desc:    "bad field",
			json:    `{"bad-field" : "42"}`,
			wantErr: `parent container container-schema (type *ytypes.LeafContainerStruct): JSON contains unexpected field bad-field`,
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
			wantErr: `E_BAD_VALUE is not a valid value for enum field EnumLeaf, type ytypes.EnumType`,
		},
		{
			desc:    "union bad type",
			json:    `{"union-leaf" : -42}`,
			wantErr: `could not find suitable union type to unmarshal value -42 type float64 into parent struct type *ytypes.LeafContainerStruct field UnionLeaf`,
		},
		{
			desc:    "binary bad type",
			json:    `{"binary-leaf" : 42}`,
			wantErr: `got float64 type for field binary-leaf, expect string`,
		},
		{
			desc:    "bool bad type",
			json:    `{"bool-leaf" : "true"}`,
			wantErr: `got string type for field bool-leaf, expect bool`,
		},
		{
			desc:    "decimal bad type",
			json:    `{"decimal-leaf" : 42.42}`,
			wantErr: `got float64 type for field decimal-leaf, expect string`,
		},
	}

	containerSchema := &yang.Entry{
		Name: "container-schema",
		Kind: yang.DirectoryEntry,
		Dir: map[string]*yang.Entry{
			"leaf": {
				Name: "leaf",
				Kind: yang.LeafEntry,
				Type: &yang.YangType{
					Kind:    yang.Ystring,
					Pattern: []string{"b+"},
				},
			},
		},
	}

	unionSchema := &yang.Entry{
		Name: "union-leaf",
		Kind: yang.LeafEntry,
		Type: &yang.YangType{
			Kind: yang.Yunion,
			Type: []*yang.YangType{
				{
					Kind:    yang.Ystring,
					Pattern: []string{"a+"},
				},
				{
					Kind: yang.Yuint32,
				},
				{
					Kind: yang.Yenum,
				},
				{
					Kind: yang.Yidentityref,
				},
				{
					Kind: yang.Yleafref,
					Path: "../leaf",
				},
			},
		},
	}

	unionNoStructSchema := &yang.Entry{
		Name: "union-leaf2",
		Kind: yang.LeafEntry,
		Type: &yang.YangType{
			Kind: yang.Yunion,
			Type: []*yang.YangType{
				{
					// Note that Validate is not called as part of Unmarshal,
					// therefore any string pattern will actually match.
					Kind:    yang.Ystring,
					Pattern: []string{"a+"},
				},
				{
					Kind:    yang.Ystring,
					Pattern: []string{"b+"},
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
		typeToLeafSchema("decimal-leaf", yang.Ydecimal64),
		enumLeafSchema,
		unionSchema,
		unionNoStructSchema,
	}

	for _, s := range leafSchemas {
		s.Parent = containerSchema
		containerSchema.Dir[s.Name] = s
	}

	var jsonTree interface{}
	for idx, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			var parent LeafContainerStruct

			if err := json.Unmarshal([]byte(tt.json), &jsonTree); err != nil {
				t.Fatal(fmt.Sprintf("%s : %s", tt.desc, err))
			}

			err := Unmarshal(containerSchema, &parent, jsonTree)
			if got, want := errToString(err), tt.wantErr; got != want {
				t.Errorf("%s (#%d): Unmarshal got error: %v, want error: %v", tt.desc, idx, got, want)
			}
			testErrLog(t, tt.desc, err)
			if err == nil {
				if got, want := parent, tt.want; !reflect.DeepEqual(got, want) {
					t.Errorf("%s (#%d): Unmarshal got:\n%v\nwant:\n%v\n", tt.desc, idx, pretty.Sprint(got), pretty.Sprint(want))
				}
			}
		})
	}

	// nil schema
	err := Unmarshal(nil, &LeafContainerStruct{}, map[string]interface{}{})
	wantErr := `nil schema for parent type *ytypes.LeafContainerStruct, value map[] (map[string]interface {})`
	if got, want := errToString(err), wantErr; got != want {
		t.Errorf("nil schema: Unmarshal got error: %v, want error: %v", got, want)
	}
	// Additional tests through private API.
	// bad parent type
	err = unmarshalUnion(containerSchema, LeafContainerStruct{}, "int8-leaf", 42)
	wantErr = `ytypes.LeafContainerStruct is not a struct ptr in unmarshalUnion`
	if got, want := errToString(err), wantErr; got != want {
		t.Errorf("bad parent type: Unmarshal got error: %v, want error: %v", got, want)
	}
	if err := unmarshalLeaf(nil, nil, nil); err != nil {
		t.Errorf("nil value: got error: %v, want error: nil", err)
	}
	if err := unmarshalLeaf(nil, nil, map[string]interface{}{}); err == nil {
		t.Errorf("nil schema: got error: nil, want nil schema error")
	}
	if err := unmarshalLeaf(enumLeafSchema, LeafContainerStruct{}, map[string]interface{}{}); err == nil {
		t.Errorf("bad schema: got error: nil, want nil schema error")
	}
	if _, err := unmarshalScalar(nil, nil, "", nil); err != nil {
		t.Errorf("nil value: got error: %v, want error: nil", err)
	}
	if _, err := unmarshalScalar(nil, nil, "", 42); err == nil {
		t.Errorf("nil schema: got error: nil, want nil schema error")
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
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			var parent ContainerStruct

			if err := json.Unmarshal([]byte(tt.json), &jsonTree); err != nil {
				t.Fatal(fmt.Sprintf("%s : %s", tt.desc, err))
			}

			err := Unmarshal(containerSchema, &parent, jsonTree)
			if got, want := errToString(err), tt.wantErr; got != want {
				t.Errorf("%s: Unmarshal got error: %v, want error: %v", tt.desc, got, want)
			}
			testErrLog(t, tt.desc, err)
			if err == nil {
				if got, want := parent, tt.want; !reflect.DeepEqual(got, want) {
					t.Errorf("%s: Unmarshal got:\n%v\nwant:\n%v\n", tt.desc, pretty.Sprint(got), pretty.Sprint(want))
				}
			}
		})
	}
}

func TestStripPrefix(t *testing.T) {
	tests := []struct {
		desc     string
		inName   string
		wantName string
		wantErr  string
	}{{
		desc:     "valid with prefix",
		inName:   "one:two",
		wantName: "two",
	}, {
		desc:     "valid without prefix",
		inName:   "two",
		wantName: "two",
	}, {
		desc:    "invalid input",
		inName:  "foo:bar:foo",
		wantErr: "path element did not form a valid name (name, prefix:name): foo:bar:foo",
	}, {
		desc:     "empty string",
		inName:   "",
		wantName: "",
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got, err := stripPrefix(tt.inName)
			if err != nil && err.Error() != tt.wantErr {
				t.Errorf("%s: stripPrefix(%v): did not get expected error, got: %v, want: %s", tt.desc, tt.inName, got, tt.wantErr)
			}

			if err != nil {
				return
			}

			if got != tt.wantName {
				t.Errorf("%s: stripPrefix(%v): did not get expected name, got: %s, want: %s", tt.desc, tt.inName, got, tt.wantName)
			}
		})
	}
}

func TestFindLeafRefSchema(t *testing.T) {
	tests := []struct {
		desc      string
		inSchema  *yang.Entry
		inPathStr string
		wantEntry *yang.Entry
		wantErr   string
	}{{
		desc: "simple reference",
		inSchema: &yang.Entry{
			Name: "referencing",
			Type: &yang.YangType{
				Kind: yang.Yleafref,
				Path: "../foo",
			},
			Parent: &yang.Entry{
				Name: "directory",
				Kind: yang.DirectoryEntry,
				Dir: map[string]*yang.Entry{
					"foo": {
						Name: "foo",
						Type: &yang.YangType{Kind: yang.Ystring},
					},
				},
			},
		},
		inPathStr: "../foo",
		wantEntry: &yang.Entry{
			Name: "foo",
			Type: &yang.YangType{Kind: yang.Ystring},
		},
	}, {
		desc: "empty path",
		inSchema: &yang.Entry{
			Name: "referencing",
			Type: &yang.YangType{
				Kind: yang.Yleafref,
			},
		},
		wantErr: "leafref schema referencing has empty path",
	}, {
		desc: "bad xpath predicate, mismatched []s",
		inSchema: &yang.Entry{
			Name: "referencing",
			Type: &yang.YangType{
				Kind: yang.Yleafref,
				Path: "/interfaces/interface[name=foo/bar",
			},
		},
		inPathStr: "/interfaces/interface[name=foo/bar",
		wantErr:   "Mismatched brackets within substring /interfaces/interface[name=foo/bar of /interfaces/interface[name=foo/bar, [ pos: 21, ] pos: -1",
	}, {
		desc: "strip prefix error in path",
		inSchema: &yang.Entry{
			Name: "referencing",
			Type: &yang.YangType{
				Kind: yang.Yleafref,
				Path: "/interface:foo:bar/baz",
			},
		},
		inPathStr: "/interface:foo:bar/baz",
		wantErr:   "leafref schema referencing path /interface:foo:bar/baz: path element did not form a valid name (name, prefix:name): interface:foo:bar",
	}, {
		desc: "nil reference",
		inSchema: &yang.Entry{
			Name: "referencing",
			Type: &yang.YangType{
				Kind: yang.Yleafref,
				Path: "/interfaces/interface/baz",
			},
		},
		inPathStr: "/interfaces/interface/baz",
		wantErr:   "schema node interfaces is nil for leafref schema referencing with path /interfaces/interface/baz",
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got, err := findLeafRefSchema(tt.inSchema, tt.inPathStr)
			if err != nil && err.Error() != tt.wantErr {
				t.Errorf("%s: findLeafRefSchema(%v, %s): did not get expected error, got: %v, want: %v", tt.desc, tt.inSchema, tt.inPathStr, err, tt.wantErr)
			}

			if err != nil {
				return
			}

			if diff := pretty.Compare(got, tt.wantEntry); diff != "" {
				t.Errorf("%s: findLeafRefSchema(%v, %s): did not get expected entry, diff(-got,+want):\n%s", tt.desc, tt.inSchema, tt.inPathStr, diff)
			}
		})
	}
}
