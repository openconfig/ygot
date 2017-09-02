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
	"github.com/openconfig/ygot/ygot"
)

var validLeafSchema = &yang.Entry{Name: "valid-leaf-schema", Kind: yang.LeafEntry, Type: &yang.YangType{Kind: yang.Ystring}}

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
	strLeafSchema := &yang.Entry{
		Name: "string-leaf-schema",
		Kind: yang.LeafEntry,
		Type: &yang.YangType{
			Kind: yang.Ystring,
		},
	}
	intLeafSchema := &yang.Entry{
		Name: "int-leaf-schema",
		Kind: yang.LeafEntry,
		Type: &yang.YangType{
			Kind: yang.Yint32,
		},
	}
	binaryLeafSchema := &yang.Entry{
		Name: "binary-leaf-schema",
		Kind: yang.LeafEntry,
		Type: &yang.YangType{
			Kind: yang.Ybinary,
		},
	}
	// TODO(mostrowski): restore when representation is decided.
	//bitsetLeafSchema := mapToBitsetSchema("bitset-leaf-schema", map[string]int64{"name1": 0, "name2": 1, "name3": 2})
	boolLeafSchema := &yang.Entry{
		Name: "bool-leaf-schema",
		Kind: yang.LeafEntry,
		Type: &yang.YangType{
			Kind: yang.Ybool,
		},
	}
	decimalLeafSchema := &yang.Entry{
		Name: "decimal-leaf-schema",
		Kind: yang.LeafEntry,
		Type: &yang.YangType{
			Kind: yang.Ydecimal64,
		},
	}

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
			schema: strLeafSchema,
			val:    ygot.String("value"),
		},
		{
			desc:   "string nil value success",
			schema: strLeafSchema,
			val:    nil,
		},
		{
			desc:    "string bad type",
			schema:  strLeafSchema,
			val:     ygot.Int32(1),
			wantErr: true,
		},
		{
			desc:    "string non ptr type",
			schema:  strLeafSchema,
			val:     int(1),
			wantErr: true,
		},
		{
			desc:   "int success",
			schema: intLeafSchema,
			val:    ygot.Int32(1),
		},
		{
			desc:    "int bad type",
			schema:  intLeafSchema,
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
			schema: binaryLeafSchema,
			val:    []byte("value"),
		},
		{
			desc:    "binary bad type",
			schema:  binaryLeafSchema,
			val:     ygot.Int32(1),
			wantErr: true,
		},
		{
			desc:   "bool success",
			schema: boolLeafSchema,
			val:    ygot.Bool(true),
		},
		{
			desc:    "bool bad type",
			schema:  boolLeafSchema,
			val:     ygot.Int32(1),
			wantErr: true,
		},
		{
			desc:   "decimal64 success",
			schema: decimalLeafSchema,
			val:    ygot.Float64(42.42),
		},
		{
			desc:    "decimal64 bad type",
			schema:  decimalLeafSchema,
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

type Union1Leaf1 struct {
	Leaf1 *string
}

func (Union1Leaf1) IsUnion1() {}

type Union1Leaf2 struct {
	Leaf2 *int16
}

func (Union1Leaf2) IsUnion1() {}

type Union1BadLeaf struct {
	Leaf3 *float32
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
					Kind: yang.Yunion,
					Type: []*yang.YangType{
						{
							Name:    "leaf1",
							Kind:    yang.Ystring,
							Pattern: []string{"a+"},
						},
						{
							Name: "leaf2",
							Kind: yang.Yint16,
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
							Name:    "leaf1",
							Kind:    yang.Ystring,
							Pattern: []string{"a+"},
						},
						{
							Name:    "leaf2",
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
							Name:    "leaf1",
							Kind:    yang.Ystring,
							Pattern: []string{"a+"},
						},
						{
							Name:    "leaf2",
							Kind:    yang.Ystring,
							Pattern: []string{"b+"},
						},
						{
							Name: "leaf3",
							Kind: yang.Yunion,
							Type: []*yang.YangType{
								{
									Name:    "leaf3",
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
			desc:   "success leaf1",
			schema: unionContainerSchema,
			val:    &UnionContainer{UnionField: &Union1Leaf1{Leaf1: ygot.String("aaa")}},
		},
		{
			desc:   "success leaf2",
			schema: unionContainerSchema,
			val:    &UnionContainer{UnionField: &Union1Leaf2{Leaf2: ygot.Int16(1)}},
		},
		{
			desc:    "bad regex",
			schema:  unionContainerSchema,
			val:     &UnionContainer{UnionField: &Union1Leaf1{Leaf1: ygot.String("bbb")}},
			wantErr: true,
		},
		{
			desc:    "bad type",
			schema:  unionContainerSchema,
			val:     &UnionContainer{UnionField: &Union1BadLeaf{Leaf3: ygot.Float32(0)}},
			wantErr: true,
		},
		{
			desc:   "success no wrapping struct leaf1",
			schema: unionContainerSchemaNoWrappingStruct,
			val:    &UnionContainerCompressed{UnionField: ygot.String("aaa")},
		},
		{
			desc:   "success no wrapping struct leaf2",
			schema: unionContainerSchemaNoWrappingStruct,
			val:    &UnionContainerCompressed{UnionField: ygot.String("bbb")},
		},
		{
			desc:   "success no wrapping struct leaf1",
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
