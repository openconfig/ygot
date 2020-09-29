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

	"github.com/google/go-cmp/cmp"
	"github.com/openconfig/gnmi/errdiff"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/testutil"
	"github.com/openconfig/ygot/ygot"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
)

var (
	base64testString        = "forty two"
	base64testStringEncoded = base64.StdEncoding.EncodeToString([]byte(base64testString))
	testBinary              = testutil.Binary(base64testString)
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

func yrangeToLeafSchema(name string, yr yang.YRange) *yang.Entry {
	return &yang.Entry{
		Name: name,
		Kind: yang.LeafEntry,
		Type: &yang.YangType{
			Kind:   yang.Ybinary,
			Length: yang.YangRange{yr},
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
			desc:    "int bad type - enum",
			schema:  typeToLeafSchema("int64", yang.Yint64),
			val:     int64(0),
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
			val:     Binary([]byte{1, 2, 3}),
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
			val:    Binary("value"),
		},
		{
			desc:   "binary success with length",
			schema: yrangeToLeafSchema("binary", yang.YRange{Min: yang.FromInt(2), Max: yang.FromInt(4)}),
			val:    Binary("aaa"),
		},
		{
			desc:    "binary bad type",
			schema:  typeToLeafSchema("binary", yang.Ybinary),
			val:     ygot.Int32(1),
			wantErr: true,
		},
		{
			desc:    "binary too short",
			schema:  yrangeToLeafSchema("binary", yang.YRange{Min: yang.FromInt(2), Max: yang.FromInt(4)}),
			val:     Binary("a"),
			wantErr: true,
		},
		{
			desc:    "binary too long",
			schema:  yrangeToLeafSchema("binary", yang.YRange{Min: yang.FromInt(2), Max: yang.FromInt(4)}),
			val:     Binary("aaaaaaaa"),
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
			desc:    "bool bad type - empty",
			schema:  typeToLeafSchema("bool", yang.Ybool),
			val:     YANGEmpty(true),
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
	UnionField testutil.TestUnion `path:"union1"`
}

func (*UnionContainer) IsYANGGoStruct() {}

// IsTestUnion ensures EnumType satisfies the testutil.TestUnion interface.
func (EnumType) IsTestUnion() {}

type Union1String struct {
	String string
}

func (*Union1String) IsTestUnion() {}

type Union1Int16 struct {
	Int16 int16
}

func (*Union1Int16) IsTestUnion() {}

type Union1EnumType struct {
	EnumType EnumType
}

func (*Union1EnumType) IsTestUnion() {}

type Union1BadLeaf struct {
	BadLeaf *float32
}

func (*Union1BadLeaf) IsTestUnion() {}

type UnionContainerCompressed struct {
	UnionField *string `path:"union1"`
}

func (*UnionContainerCompressed) IsYANGGoStruct() {}

type UnionContainerSingleEnum struct {
	UnionField EnumType `path:"union1"`
}

func (*UnionContainerSingleEnum) IsYANGGoStruct() {}

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
							Name:         "string",
							Kind:         yang.Ystring,
							Pattern:      []string{"a+"},
							POSIXPattern: []string{"^a+$"},
						},
						{
							Name: "int16",
							Kind: yang.Yint16,
						},
						{
							Name: "enum",
							Kind: yang.Yenum,
						},
						{
							Name: "bin",
							Kind: yang.Ybinary,
						},
					},
				},
			},
		},
	}

	unionContainerSingleEnumSchema := &yang.Entry{
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
							Name:         "string",
							Kind:         yang.Ystring,
							Pattern:      []string{"a+"},
							POSIXPattern: []string{"^a+$"},
						},
						{
							Name:         "string2",
							Kind:         yang.Ystring,
							Pattern:      []string{"b+"},
							POSIXPattern: []string{"^b+$"},
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
							Name:         "string",
							Kind:         yang.Ystring,
							Pattern:      []string{"a+"},
							POSIXPattern: []string{"^a+$"},
						},
						{
							Name:         "string2",
							Kind:         yang.Ystring,
							Pattern:      []string{"b+"},
							POSIXPattern: []string{"^b+$"},
						},
						{
							Name: "bad-leaf",
							Kind: yang.Yunion,
							Type: []*yang.YangType{
								{
									Name:         "bad-leaf",
									Kind:         yang.Ystring,
									Pattern:      []string{"c+"},
									POSIXPattern: []string{"^c+$"},
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
			val:    &UnionContainer{UnionField: testutil.UnionString("aaa")},
		},
		{
			desc:   "success int16",
			schema: unionContainerSchema,
			val:    &UnionContainer{UnionField: testutil.UnionInt16(42)},
		},
		{
			desc:   "success int64",
			schema: unionContainerSchema,
			val:    &UnionContainer{UnionField: testutil.UnionInt64(42)},
		},
		{
			desc:   "success enum",
			schema: unionContainerSchema,
			val:    &UnionContainer{UnionField: EnumType(42)},
		},
		{
			desc:    "bad regex",
			schema:  unionContainerSchema,
			val:     &UnionContainer{UnionField: testutil.UnionString("bbb")},
			wantErr: true,
		},
		{
			desc:   "success binary",
			schema: unionContainerSchema,
			val:    &UnionContainer{UnionField: testutil.Binary("abc")},
		},
		{
			desc:   "success string (wrapper union type)",
			schema: unionContainerSchema,
			val:    &UnionContainer{UnionField: &Union1String{"aaa"}},
		},
		{
			desc:   "success int16 (wrapper union type)",
			schema: unionContainerSchema,
			val:    &UnionContainer{UnionField: &Union1Int16{1}},
		},
		{
			desc:   "success enum (wrapper union type)",
			schema: unionContainerSchema,
			val:    &UnionContainer{UnionField: &Union1EnumType{EnumType: 42}},
		},
		{
			desc:    "bad regex (wrapper union type)",
			schema:  unionContainerSchema,
			val:     &UnionContainer{UnionField: &Union1String{"bbb"}},
			wantErr: true,
		},
		{
			desc:    "bad type (wrapper union type)",
			schema:  unionContainerSchema,
			val:     &UnionContainer{UnionField: &Union1BadLeaf{BadLeaf: ygot.Float32(0)}},
			wantErr: true,
		},
		{
			desc:   "success single-valued union: enum",
			schema: unionContainerSingleEnumSchema,
			val:    &UnionContainerSingleEnum{UnionField: EnumType(42)},
		},
		{
			desc:   "success single-valued union: string",
			schema: unionContainerSchemaNoWrappingStruct,
			val:    &UnionContainerCompressed{UnionField: ygot.String("aaa")},
		},
		{
			desc:   "success single-valued union: int16",
			schema: unionContainerSchemaNoWrappingStruct,
			val:    &UnionContainerCompressed{UnionField: ygot.String("bbb")},
		},
		{
			desc:   "success single-valued union: string",
			schema: unionContainerSchemaNoWrappingStruct,
			val:    &UnionContainerCompressed{UnionField: ygot.String("aaa")},
		},
		{
			desc:    "single-valued union: no schemas match",
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
	if err := validateUnion(unionContainerSchema.Dir["union1"], nil); err != nil {
		t.Errorf("nil value: got error: %v, want error: nil", err)
	}
	if err := validateUnion(unionContainerSchema.Dir["union1"], 42); err == nil {
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
					Type: &yang.YangType{
						Kind:         yang.Ystring,
						Pattern:      []string{"a+"},
						POSIXPattern: []string{"^a+$"},
					},
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
					Type: &yang.YangType{
						Kind:         yang.Ystring,
						Pattern:      []string{"a+"},
						POSIXPattern: []string{"^a+$"},
					},
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
					Type: &yang.YangType{
						Kind:         yang.Ystring,
						Pattern:      []string{"a+"},
						POSIXPattern: []string{"^a+$"},
					},
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
								Kind:         yang.Ystring,
								Pattern:      []string{"b.*"},
								POSIXPattern: []string{"^b.*$"},
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
						Kind:         yang.Ystring,
						Pattern:      []string{"b.*"},
						POSIXPattern: []string{"^b.*$"},
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
						Kind:         yang.Ystring,
						Pattern:      []string{"b.*"},
						POSIXPattern: []string{"^b.*$"},
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

type LeafContainerStruct struct {
	Int8Leaf             *int8                 `path:"int8-leaf"`
	Int8LeafList         []int8                `path:"int8-leaflist"`
	Uint8Leaf            *uint8                `path:"uint8-leaf"`
	Int16Leaf            *int16                `path:"int16-leaf"`
	Uint16Leaf           *uint16               `path:"uint16-leaf"`
	Int32Leaf            *int32                `path:"int32-leaf"`
	Uint32Leaf           *uint32               `path:"uint32-leaf"`
	Int64Leaf            *int64                `path:"int64-leaf"`
	Uint64Leaf           *uint64               `path:"uint64-leaf"`
	StringLeaf           *string               `path:"string-leaf"`
	BinaryLeaf           Binary                `path:"binary-leaf"`
	BoolLeaf             *bool                 `path:"bool-leaf"`
	DecimalLeaf          *float64              `path:"decimal-leaf"`
	EnumLeaf             EnumType              `path:"enum-leaf"`
	UnionEnumLeaf        EnumType              `path:"union-enum-leaf"`
	UnionLeaf            UnionLeafType         `path:"union-leaf"`
	UnionLeaf2           *string               `path:"union-leaf2"`
	EmptyLeaf            YANGEmpty             `path:"empty-leaf"`
	UnionLeafSlice       []UnionLeafType       `path:"union-leaflist"`
	UnionLeafSingleType  []string              `path:"union-stleaflist"`
	UnionEnumLeaflist    []EnumType            `path:"union-enum-leaflist"`
	UnionLeafSimple      UnionLeafTypeSimple   `path:"union-leaf-simple"`
	UnionLeafSliceSimple []UnionLeafTypeSimple `path:"union-leaflist-simple"`
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
		"/container-schema/union-leaf":            {reflect.TypeOf(EnumType(0)), reflect.TypeOf(EnumType2(0))},
		"/container-schema/union-enum-leaf":       {reflect.TypeOf(EnumType(0))},
		"/container-schema/union-leaflist":        {reflect.TypeOf(EnumType(0)), reflect.TypeOf(EnumType2(0))},
		"/container-schema/union-enum-leaflist":   {reflect.TypeOf(EnumType(0))},
		"/container-schema/union-leaf-simple":     {reflect.TypeOf(EnumType(0)), reflect.TypeOf(EnumType2(0))},
		"/container-schema/union-leaflist-simple": {reflect.TypeOf(EnumType(0)), reflect.TypeOf(EnumType2(0))},
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

type UnionLeafTypeSimple interface {
	Is_UnionLeafTypeSimple()
}

func (EnumType) Is_UnionLeafTypeSimple() {}

func (EnumType2) Is_UnionLeafTypeSimple() {}

func (*LeafContainerStruct) To_UnionLeafTypeSimple(i interface{}) (UnionLeafTypeSimple, error) {
	if v, ok := i.(UnionLeafTypeSimple); ok {
		return v, nil
	}
	switch v := i.(type) {
	case []byte:
		return testutil.Binary(v), nil
	case string:
		return testutil.UnionString(v), nil
	case uint32:
		return testutil.UnionUint32(v), nil
	}
	return nil, fmt.Errorf("cannot convert %v to UnionLeafTypeSimple, unknown union type, got: %T, want any of [string, uint32, EnumType, EnumType2, Binary]", i, i)
}

func TestUnmarshalLeafJSONEncoding(t *testing.T) {
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
			want: LeafContainerStruct{BinaryLeaf: Binary(base64testString)},
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
			json: `{"union-leaf-simple" : "forty-two"}`,
			want: LeafContainerStruct{UnionLeafSimple: testutil.UnionString("forty-two")},
		},
		{
			desc: "union uint32 success",
			json: `{"union-leaf-simple" : 42}`,
			want: LeafContainerStruct{UnionLeafSimple: testutil.UnionUint32(42)},
		},
		{
			desc: "union binary success",
			json: `{"union-leaf-simple" : "` + base64testStringEncoded + `"}`,
			want: LeafContainerStruct{UnionLeafSimple: testutil.Binary(base64testString)},
		},
		{
			desc: "union enum success",
			json: `{"union-leaf-simple" : "E_VALUE_FORTY_TWO"}`,
			want: LeafContainerStruct{UnionLeafSimple: EnumType(42)},
		},
		{
			desc: "union enum2 success",
			json: `{"union-leaf-simple" : "E_VALUE_FORTY_THREE"}`,
			want: LeafContainerStruct{UnionLeafSimple: EnumType2(43)},
		},
		{
			desc: "leaf-list of union success, single value",
			json: `{"union-leaflist-simple": ["E_VALUE_FORTY_THREE"]}`,
			want: LeafContainerStruct{UnionLeafSliceSimple: []UnionLeafTypeSimple{EnumType2(43)}},
		},
		{
			desc: "leaf-list of union success, multi-value",
			json: `{"union-leaflist-simple": ["E_VALUE_FORTY_THREE", 40]}`,
			want: LeafContainerStruct{
				UnionLeafSliceSimple: []UnionLeafTypeSimple{
					EnumType2(43),
					testutil.UnionUint32(40),
				},
			},
		},
		{
			desc: "union string success (wrapper union)",
			json: `{"union-leaf" : "forty-two"}`,
			want: LeafContainerStruct{UnionLeaf: &UnionLeafType_String{String: "forty-two"}},
		},
		{
			desc: "union uint32 success (wrapper union)",
			json: `{"union-leaf" : 42}`,
			want: LeafContainerStruct{UnionLeaf: &UnionLeafType_Uint32{Uint32: 42}},
		},
		{
			desc: "union enum success (wrapper union)",
			json: `{"union-leaf" : "E_VALUE_FORTY_TWO"}`,
			want: LeafContainerStruct{UnionLeaf: &UnionLeafType_EnumType{EnumType: 42}},
		},
		{
			desc: "union enum2 success (wrapper union)",
			json: `{"union-leaf" : "E_VALUE_FORTY_THREE"}`,
			want: LeafContainerStruct{UnionLeaf: &UnionLeafType_EnumType2{EnumType2: 43}},
		},
		{
			desc: "union no struct success, correct type, value unvalidated (wrapper union)",
			json: `{"union-leaf2" : "ccc"}`,
			want: LeafContainerStruct{UnionLeaf2: ygot.String("ccc")},
		},
		{
			desc: "leaf-list of single type union success, single value (wrapper union)",
			json: `{"union-stleaflist": ["ccc"]}`,
			want: LeafContainerStruct{UnionLeafSingleType: []string{"ccc"}},
		},
		{
			desc: "leaf-list of single type union success, multi-value (wrapper union)",
			json: `{"union-stleaflist": ["ccc", "ddd"]}`,
			want: LeafContainerStruct{UnionLeafSingleType: []string{"ccc", "ddd"}},
		},
		{
			desc: "leaf-list of union success, single value (wrapper union)",
			json: `{"union-leaflist": ["E_VALUE_FORTY_THREE"]}`,
			want: LeafContainerStruct{UnionLeafSlice: []UnionLeafType{&UnionLeafType_EnumType2{EnumType2: 43}}},
		},
		{
			desc: "leaf-list of union success, multi-value (wrapper union)",
			json: `{"union-leaflist": ["E_VALUE_FORTY_THREE", "eeee"]}`,
			want: LeafContainerStruct{
				UnionLeafSlice: []UnionLeafType{
					&UnionLeafType_EnumType2{EnumType2: 43},
					&UnionLeafType_String{"eeee"},
				},
			},
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
			desc:    "union bad type (wrapper union)",
			json:    `{"union-leaf" : -42}`,
			wantErr: `could not find suitable union type to unmarshal value -42 type float64 into parent struct type *ytypes.LeafContainerStruct field UnionLeaf`,
		},
		{
			desc:    "union bad type",
			json:    `{"union-leaf-simple" : -42}`,
			wantErr: `could not find suitable union type to unmarshal value -42 type float64 into parent struct type *ytypes.LeafContainerStruct field UnionLeafSimple`,
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
		{
			desc: "empty valid type",
			json: `{"empty-leaf": [null]}`,
			want: LeafContainerStruct{EmptyLeaf: true},
		},
		{
			desc:    "empty bad type",
			json:    `{"empty-leaf": "fish"}`,
			wantErr: "got string type for field empty-leaf, expect slice",
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
					Kind:         yang.Ystring,
					Pattern:      []string{"b+"},
					POSIXPattern: []string{"^b+$"},
				},
			},
		},
	}

	unionSchemaSimple := &yang.Entry{
		Name: "union-leaf-simple",
		Kind: yang.LeafEntry,
		Type: &yang.YangType{
			Kind: yang.Yunion,
			Type: []*yang.YangType{
				{
					Kind: yang.Yuint32,
				},
				{
					Kind: yang.Yenum,
				},
				{
					Kind: yang.Ybinary,
				},
				{
					Kind: yang.Yidentityref,
				},
				{
					Kind: yang.Yleafref,
					Path: "../leaf",
				},
				{
					Kind:    yang.Ystring,
					Pattern: []string{"a+"},
				},
			},
		},
	}

	unionLeafListSchemaSimple := &yang.Entry{
		Name:     "union-leaflist-simple",
		Kind:     yang.LeafEntry,
		ListAttr: &yang.ListAttr{},
		Type: &yang.YangType{
			Kind: yang.Yunion,
			Type: []*yang.YangType{
				{
					Kind: yang.Yuint32,
				},
				{
					Kind: yang.Yenum,
				},
				{
					Kind: yang.Ybinary,
				},
				{
					Kind: yang.Yidentityref,
				},
				{
					Kind:    yang.Ystring,
					Pattern: []string{"a+"},
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
					Kind:         yang.Ystring,
					Pattern:      []string{"a+"},
					POSIXPattern: []string{"^a+$"},
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

	unionLeafListSchema := &yang.Entry{
		Name:     "union-leaflist",
		Kind:     yang.LeafEntry,
		ListAttr: &yang.ListAttr{},
		Type: &yang.YangType{
			Kind: yang.Yunion,
			Type: []*yang.YangType{
				{
					Kind:         yang.Ystring,
					Pattern:      []string{"a+"},
					POSIXPattern: []string{"^a+$"},
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
					Kind:         yang.Ystring,
					Pattern:      []string{"a+"},
					POSIXPattern: []string{"^a+$"},
				},
				{
					Kind:         yang.Ystring,
					Pattern:      []string{"b+"},
					POSIXPattern: []string{"^b+$"},
				},
			},
		},
	}

	unionSTLeafListSchema := &yang.Entry{
		Name:     "union-stleaflist",
		Kind:     yang.LeafEntry,
		ListAttr: &yang.ListAttr{},
		Type: &yang.YangType{
			Kind: yang.Yunion,
			Type: []*yang.YangType{
				{
					// Note that Validate is not called as part of Unmarshal,
					// therefore any string pattern will actually match.
					Kind:         yang.Ystring,
					Pattern:      []string{"a+"},
					POSIXPattern: []string{"^a+$"},
				},
				{
					Kind:         yang.Ystring,
					Pattern:      []string{"b+"},
					POSIXPattern: []string{"^b+$"},
				},
			},
		},
	}

	leafListSchema := &yang.Entry{
		Name:     "int8-leaflist",
		Kind:     yang.LeafEntry,
		Type:     &yang.YangType{Kind: yang.Yint8},
		ListAttr: &yang.ListAttr{MinElements: &yang.Value{Name: "0"}},
	}

	unionSingleEnumSchema := &yang.Entry{
		Name: "union-enum-leaf",
		Kind: yang.LeafEntry,
		Type: &yang.YangType{
			Kind: yang.Yunion,
			Type: []*yang.YangType{
				{
					Kind: yang.Yenum,
				},
			},
		},
	}
	unionSingleEnumLeafListSchema := &yang.Entry{
		Name:     "union-enum-leaflist",
		Kind:     yang.LeafEntry,
		ListAttr: &yang.ListAttr{},
		Type: &yang.YangType{
			Kind: yang.Yunion,
			Type: []*yang.YangType{
				{
					Kind: yang.Yenum,
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
		typeToLeafSchema("empty-leaf", yang.Yempty),
		enumLeafSchema,
		unionSchemaSimple,
		unionLeafListSchemaSimple,
		unionSchema,
		unionNoStructSchema,
		unionLeafListSchema,
		unionSTLeafListSchema,
		leafListSchema,
		unionSingleEnumSchema,
		unionSingleEnumLeafListSchema,
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
				got, want := parent, tt.want
				if diff := cmp.Diff(want, got); diff != "" {
					t.Errorf("%s (#%d): Unmarshal (-want, +got):\n%s", tt.desc, idx, diff)
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
	err = unmarshalUnion(containerSchema, LeafContainerStruct{}, "int8-leaf", 42, JSONEncoding)
	wantErr = `ytypes.LeafContainerStruct is not a struct ptr in unmarshalUnion`
	if got, want := errToString(err), wantErr; got != want {
		t.Errorf("bad parent type: Unmarshal got error: %v, want error: %v", got, want)
	}
	err = unmarshalUnion(containerSchema, &LeafContainerStruct{}, "i-dont-exist", 42, JSONEncoding)
	wantErr = `i-dont-exist is not a valid field name in *ytypes.LeafContainerStruct`
	if got, want := errToString(err), wantErr; got != want {
		t.Errorf("bad parent type: Unmarshal got error: %v, want error: %v", got, want)
	}
	if err := unmarshalLeaf(nil, nil, nil, JSONEncoding); err != nil {
		t.Errorf("nil value: got error: %v, want error: nil", err)
	}
	if err := unmarshalLeaf(nil, nil, map[string]interface{}{}, JSONEncoding); err == nil {
		t.Errorf("nil schema: got error: nil, want nil schema error")
	}
	if err := unmarshalLeaf(enumLeafSchema, LeafContainerStruct{}, map[string]interface{}{}, JSONEncoding); err == nil {
		t.Errorf("bad schema: got error: nil, want nil schema error")
	}
	if _, err := unmarshalScalar(nil, nil, "", nil, JSONEncoding); err != nil {
		t.Errorf("nil value: got error: %v, want error: nil", err)
	}
	if _, err := unmarshalScalar(nil, nil, "", 42, JSONEncoding); err == nil {
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
				got, want := parent, tt.want
				if diff := cmp.Diff(want, got); diff != "" {
					t.Errorf("%s: Unmarshal (-want, +got):\n%s", tt.desc, diff)
				}
			}
		})
	}
}

func TestUnmarshalLeafGNMIEncoding(t *testing.T) {
	containerSchema := &yang.Entry{
		Name: "container-schema",
		Kind: yang.DirectoryEntry,
		Dir: map[string]*yang.Entry{
			"leaf": {
				Name: "leaf",
				Kind: yang.LeafEntry,
				Type: &yang.YangType{
					Kind:         yang.Ystring,
					Pattern:      []string{"b+"},
					POSIXPattern: []string{"^b+$"},
				},
			},
		},
	}
	unionSchemaSimple := &yang.Entry{
		Parent: containerSchema,
		Name:   "union-leaf-simple",
		Kind:   yang.LeafEntry,
		Type: &yang.YangType{
			Kind: yang.Yunion,
			Type: []*yang.YangType{
				{
					Kind: yang.Yuint32,
				},
				{
					Kind: yang.Yenum,
				},
				{
					Kind: yang.Ybinary,
				},
				{
					Kind: yang.Yidentityref,
				},
				{
					Kind: yang.Yleafref,
					Path: "../leaf",
				},
				{
					Kind:    yang.Ystring,
					Pattern: []string{"a+"},
				},
			},
		},
	}
	unionSchema := &yang.Entry{
		Parent: containerSchema,
		Name:   "union-leaf",
		Kind:   yang.LeafEntry,
		Type: &yang.YangType{
			Kind: yang.Yunion,
			Type: []*yang.YangType{
				{
					Kind:         yang.Ystring,
					Pattern:      []string{"a+"},
					POSIXPattern: []string{"^a+$"},
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
	unionSingleStringSchema := &yang.Entry{
		Parent: containerSchema,
		Name:   "union-leaf2",
		Kind:   yang.LeafEntry,
		Type: &yang.YangType{
			Kind: yang.Yunion,
			Type: []*yang.YangType{
				{
					Kind: yang.Ystring,
				},
			},
		},
	}
	unionSingleEnumSchema := &yang.Entry{
		Parent: containerSchema,
		Name:   "union-enum-leaf",
		Kind:   yang.LeafEntry,
		Type: &yang.YangType{
			Kind: yang.Yunion,
			Type: []*yang.YangType{
				{
					Kind: yang.Yenum,
				},
			},
		},
	}
	unionSingleEnumLeafListSchema := &yang.Entry{
		Parent:   containerSchema,
		Name:     "union-enum-leaflist",
		Kind:     yang.LeafEntry,
		ListAttr: &yang.ListAttr{},
		Type: &yang.YangType{
			Kind: yang.Yunion,
			Type: []*yang.YangType{
				{
					Kind: yang.Yenum,
				},
			},
		},
	}

	unionSTLeafListSchema := &yang.Entry{
		Parent:   containerSchema,
		Name:     "union-stleaflist",
		Kind:     yang.LeafEntry,
		ListAttr: &yang.ListAttr{},
		Type: &yang.YangType{
			Kind: yang.Yunion,
			Type: []*yang.YangType{
				{
					// Note that Validate is not called as part of Unmarshal,
					// therefore any string pattern will actually match.
					Kind:         yang.Ystring,
					Pattern:      []string{"a+"},
					POSIXPattern: []string{"^a+$"},
				},
				{
					Kind:         yang.Ystring,
					Pattern:      []string{"b+"},
					POSIXPattern: []string{"^b+$"},
				},
			},
		},
	}

	leafListSchema := &yang.Entry{
		Parent:   containerSchema,
		Name:     "int8-leaflist",
		Kind:     yang.LeafEntry,
		Type:     &yang.YangType{Kind: yang.Yint8},
		ListAttr: &yang.ListAttr{MinElements: &yang.Value{Name: "0"}},
	}

	tests := []struct {
		desc     string
		inSchema *yang.Entry
		inVal    interface{}
		wantVal  interface{}
		wantErr  string
	}{
		{
			desc:     "success gNMI BoolVal to Ybool",
			inSchema: typeToLeafSchema("bool-leaf", yang.Ybool),
			inVal: &gpb.TypedValue{
				Value: &gpb.TypedValue_BoolVal{
					BoolVal: true,
				},
			},
			wantVal: &LeafContainerStruct{BoolLeaf: ygot.Bool(true)},
		},
		{
			desc:     "success gNMI StringVal to Ystring",
			inSchema: typeToLeafSchema("string-leaf", yang.Ystring),
			inVal: &gpb.TypedValue{
				Value: &gpb.TypedValue_StringVal{
					StringVal: "forty two",
				},
			},
			wantVal: &LeafContainerStruct{StringLeaf: ygot.String("forty two")},
		},
		{
			desc:     "success gNMI StringVal to Yenum",
			inSchema: typeToLeafSchema("enum-leaf", yang.Yenum),
			inVal: &gpb.TypedValue{
				Value: &gpb.TypedValue_StringVal{
					StringVal: "E_VALUE_FORTY_TWO",
				},
			},
			wantVal: &LeafContainerStruct{EnumLeaf: EnumType(42)},
		},
		{
			desc:     "fail gNMI StringVal to Ystring due to missing StringVal in TypedValue",
			inSchema: typeToLeafSchema("string-leaf", yang.Ystring),
			inVal: &gpb.TypedValue{
				Value: &gpb.TypedValue_IntVal{
					IntVal: 42,
				},
			},
			wantErr: "failed to unmarshal &{42} into string",
		},
		{
			desc:     "success gNMI IntVal to Yint8",
			inSchema: typeToLeafSchema("int8-leaf", yang.Yint8),
			inVal: &gpb.TypedValue{
				Value: &gpb.TypedValue_IntVal{
					IntVal: 42,
				},
			},
			wantVal: &LeafContainerStruct{Int8Leaf: ygot.Int8(42)},
		},
		{
			desc:     "success gNMI IntVal to Yint64",
			inSchema: typeToLeafSchema("int64-leaf", yang.Yint64),
			inVal: &gpb.TypedValue{
				Value: &gpb.TypedValue_IntVal{
					IntVal: 4242,
				},
			},
			wantVal: &LeafContainerStruct{Int64Leaf: ygot.Int64(4242)},
		},
		{
			desc:     "fail gNMI IntVal to Yint8 due to overflow",
			inSchema: typeToLeafSchema("int8-leaf", yang.Yint8),
			inVal: &gpb.TypedValue{
				Value: &gpb.TypedValue_IntVal{
					IntVal: 4242,
				},
			},
			wantErr: `StringToType("4242", int8) failed; unable to convert "4242" to int8`,
		},
		{
			desc:     "failure gNMI nil value",
			inSchema: typeToLeafSchema("decimal-leaf", yang.Ydecimal64),
			inVal:    nil,
			wantErr:  "nil value to unmarshal",
		},
		{
			desc:     "failure gNMI nil TypedValue",
			inSchema: typeToLeafSchema("decimal-leaf", yang.Ydecimal64),
			inVal:    (*gpb.TypedValue)(nil),
			wantErr:  "nil value to unmarshal",
		},
		{
			desc:     "success gNMI IntVal to Yuint8",
			inSchema: typeToLeafSchema("uint8-leaf", yang.Yuint8),
			inVal: &gpb.TypedValue{
				Value: &gpb.TypedValue_UintVal{
					UintVal: 42,
				},
			},
			wantVal: &LeafContainerStruct{Uint8Leaf: ygot.Uint8(42)},
		},
		{
			desc:     "success gNMI IntVal to Yuint64",
			inSchema: typeToLeafSchema("uint64-leaf", yang.Yuint64),
			inVal: &gpb.TypedValue{
				Value: &gpb.TypedValue_UintVal{
					UintVal: 42,
				},
			},
			wantVal: &LeafContainerStruct{Uint64Leaf: ygot.Uint64(42)},
		},
		{
			desc:     "fail gNMI UintVal to Yuint8 due to overflow",
			inSchema: typeToLeafSchema("uint8-leaf", yang.Yuint8),
			inVal: &gpb.TypedValue{
				Value: &gpb.TypedValue_UintVal{
					UintVal: 4242,
				},
			},
			wantErr: `StringToType("4242", uint8) failed; unable to convert "4242" to uint8`,
		},
		{
			desc:     "fail gNMI TypedValue with nil Value field",
			inSchema: typeToLeafSchema("uint8-leaf", yang.Yuint8),
			inVal:    &gpb.TypedValue{},
			wantErr:  `failed to unmarshal`,
		},
		{
			desc:     "success gNMI FloatVal to Ydecimal64",
			inSchema: typeToLeafSchema("decimal-leaf", yang.Ydecimal64),
			inVal: &gpb.TypedValue{
				Value: &gpb.TypedValue_FloatVal{
					FloatVal: 42.42,
				},
			},
			// FloatVal above is casted to a float64 in sanitizeGNMI which changes the
			// precision. In wantVal, same operation is done to match the value.
			wantVal: &LeafContainerStruct{DecimalLeaf: ygot.Float64(float64(float32(42.42)))},
		},
		{
			desc:     "success gNMI Decimal64 to Ydecimal64",
			inSchema: typeToLeafSchema("decimal-leaf", yang.Ydecimal64),
			inVal: &gpb.TypedValue{
				Value: &gpb.TypedValue_DecimalVal{
					DecimalVal: &gpb.Decimal64{Digits: 42, Precision: 2},
				},
			},
			wantVal: &LeafContainerStruct{DecimalLeaf: ygot.Float64(0.42)},
		},
		{
			desc:     "fail gNMI nil Decimal64 value",
			inSchema: typeToLeafSchema("decimal-leaf", yang.Ydecimal64),
			inVal: &gpb.TypedValue{
				Value: &gpb.TypedValue_DecimalVal{
					DecimalVal: nil,
				},
			},
			wantErr: "DecimalVal is nil",
		},
		{
			desc:     "success gNMI BytesVal to Ybinary",
			inSchema: typeToLeafSchema("binary-leaf", yang.Ybinary),
			inVal: &gpb.TypedValue{
				Value: &gpb.TypedValue_BytesVal{
					BytesVal: []byte("value"),
				},
			},
			wantVal: &LeafContainerStruct{BinaryLeaf: Binary([]byte("value"))},
		},
		{
			desc:     "fail gNMI BytesVal is nil",
			inSchema: typeToLeafSchema("binary-leaf", yang.Ybinary),
			inVal: &gpb.TypedValue{
				Value: &gpb.TypedValue_BytesVal{
					BytesVal: nil,
				},
			},
			wantErr: "BytesVal is nil",
		},
		{
			desc:     "success unmarshalling union leaf string field",
			inSchema: unionSchemaSimple,
			inVal: &gpb.TypedValue{
				Value: &gpb.TypedValue_StringVal{
					StringVal: "forty two",
				},
			},
			wantVal: &LeafContainerStruct{UnionLeafSimple: testutil.UnionString("forty two")},
		},
		{
			desc:     "fail unmarshalling nil for union leaf field",
			inSchema: unionSchemaSimple,
			inVal:    nil,
			wantErr:  "nil value to unmarshal",
		},
		{
			desc:     "success unmarshalling union leaf enum field",
			inSchema: unionSchemaSimple,
			inVal: &gpb.TypedValue{
				Value: &gpb.TypedValue_StringVal{
					StringVal: "E_VALUE_FORTY_TWO",
				},
			},
			wantVal: &LeafContainerStruct{UnionLeafSimple: EnumType(42)},
		},
		{
			desc:     "success unmarshalling union leaf binary field",
			inSchema: unionSchemaSimple,
			inVal: &gpb.TypedValue{
				Value: &gpb.TypedValue_BytesVal{
					BytesVal: []byte(base64testString),
				},
			},
			wantVal: &LeafContainerStruct{UnionLeafSimple: testBinary},
		},
		{
			desc:     "success unmarshalling union (wrapper union) leaf string field",
			inSchema: unionSchema,
			inVal: &gpb.TypedValue{
				Value: &gpb.TypedValue_StringVal{
					StringVal: "forty two",
				},
			},
			wantVal: &LeafContainerStruct{UnionLeaf: &UnionLeafType_String{String: "forty two"}},
		},
		{
			desc:     "fail unmarshalling nil for union (wrapper union) leaf string field",
			inSchema: unionSchema,
			inVal:    nil,
			wantErr:  "nil value to unmarshal",
		},
		{
			desc:     "success unmarshalling union (wrapper union) leaf enum field",
			inSchema: unionSchema,
			inVal: &gpb.TypedValue{
				Value: &gpb.TypedValue_StringVal{
					StringVal: "E_VALUE_FORTY_TWO",
				},
			},
			wantVal: &LeafContainerStruct{UnionLeaf: &UnionLeafType_EnumType{EnumType: 42}},
		},
		{
			desc:     "success unmarshalling union with single string leaf field",
			inSchema: unionSingleStringSchema,
			inVal: &gpb.TypedValue{
				Value: &gpb.TypedValue_StringVal{
					StringVal: "forty two",
				},
			},
			wantVal: &LeafContainerStruct{UnionLeaf2: ygot.String("forty two")},
		},
		{
			desc:     "success unmarshalling union with single enum leaf field",
			inSchema: unionSingleEnumSchema,
			inVal: &gpb.TypedValue{
				Value: &gpb.TypedValue_StringVal{
					StringVal: "E_VALUE_FORTY_TWO",
				},
			},
			wantVal: &LeafContainerStruct{UnionEnumLeaf: EnumType(42)},
		},
		{
			desc:     "success unmarshalling leaflist of unions with single string leaf field",
			inSchema: unionSTLeafListSchema,
			inVal: &gpb.TypedValue{
				Value: &gpb.TypedValue_LeaflistVal{
					LeaflistVal: &gpb.ScalarArray{
						Element: []*gpb.TypedValue{
							{Value: &gpb.TypedValue_StringVal{StringVal: "forty two"}},
							{Value: &gpb.TypedValue_StringVal{StringVal: "forty three"}},
						},
					},
				},
			},
			wantVal: &LeafContainerStruct{UnionLeafSingleType: []string{"forty two", "forty three"}},
		},
		{
			desc:     "success unmarshalling leaflist of unions with single enum leaf field",
			inSchema: unionSingleEnumLeafListSchema,
			inVal: &gpb.TypedValue{
				Value: &gpb.TypedValue_LeaflistVal{
					LeaflistVal: &gpb.ScalarArray{
						Element: []*gpb.TypedValue{
							{Value: &gpb.TypedValue_StringVal{StringVal: "E_VALUE_FORTY_TWO"}},
						},
					},
				},
			},
			wantVal: &LeafContainerStruct{UnionEnumLeaflist: []EnumType{42}},
		},
		{
			desc:     "success unmarshalling int8 leaf list field with TypedValue_LeaflistVal",
			inSchema: leafListSchema,
			inVal: &gpb.TypedValue{
				Value: &gpb.TypedValue_LeaflistVal{
					LeaflistVal: &gpb.ScalarArray{
						Element: []*gpb.TypedValue{
							{Value: &gpb.TypedValue_IntVal{IntVal: 42}},
							{Value: &gpb.TypedValue_IntVal{IntVal: 43}},
						},
					},
				},
			},
			wantVal: &LeafContainerStruct{Int8LeafList: []int8{42, 43}},
		},
		{
			desc:     "fail unmarshalling int8 leaf list field with incorrect type",
			inSchema: leafListSchema,
			inVal:    "forty two",
			wantErr:  "got type string, expect *gpb.TypedValue",
		},
		{
			desc:     "fail unmarshalling int8 leaf list field with TypedValue_IntVal",
			inSchema: leafListSchema,
			inVal: &gpb.TypedValue{
				Value: &gpb.TypedValue_IntVal{
					IntVal: 42,
				},
			},
			wantErr: "expect *gpb.TypedValue_LeaflistVal set in *gpb.TypedValue",
		},
		{
			desc:     "fail unmarshalling int8 leaf list field with TypedValue_StringVal",
			inSchema: leafListSchema,
			inVal: &gpb.TypedValue{
				Value: &gpb.TypedValue_StringVal{
					StringVal: "forty two",
				},
			},
			wantErr: "expect *gpb.TypedValue_LeaflistVal set in *gpb.TypedValue",
		},
	}
	for _, tt := range tests {
		inParent := &LeafContainerStruct{}
		err := unmarshalGeneric(tt.inSchema, inParent, tt.inVal, GNMIEncoding)
		if diff := errdiff.Substring(err, tt.wantErr); diff != "" {
			t.Errorf("%s: unmarshalLeaf(%v, %v, %v, GNMIEncoding): %v", tt.desc, tt.inSchema, inParent, tt.inVal, diff)
		}
		if err != nil {
			continue
		}
		if diff := cmp.Diff(tt.wantVal, inParent); diff != "" {
			t.Errorf("%s: unmarshalLeaf(%v, %v, %v, GNMIEncoding): (-want, +got):\n%s", tt.desc, tt.inSchema, inParent, tt.inVal, diff)
		}
	}
}
