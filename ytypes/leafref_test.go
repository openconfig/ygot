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
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/openconfig/gnmi/errdiff"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/testutil"
	"github.com/openconfig/ygot/ygot"
)

// addParents adds parent pointers for a schema tree.
func addParents(e *yang.Entry) {
	for _, c := range e.Dir {
		c.Parent = e
		addParents(c)
	}
}

func TestValidateLeafRefData(t *testing.T) {
	containerWithLeafListSchema := &yang.Entry{
		Name: "container",
		Kind: yang.DirectoryEntry,
		Dir: map[string]*yang.Entry{
			"leaf-list": {
				Name:     "leaf-list",
				Kind:     yang.LeafEntry,
				Type:     &yang.YangType{Kind: yang.Yint32},
				ListAttr: yang.NewDefaultListAttr(),
			},
			"list": {
				Name:     "list",
				Kind:     yang.DirectoryEntry,
				ListAttr: yang.NewDefaultListAttr(),
				Key:      "key",
				Dir: map[string]*yang.Entry{
					"key": {
						Name: "key",
						Kind: yang.LeafEntry,
						Type: &yang.YangType{Kind: yang.Yint32},
					},
					"int32": {
						Name: "int32",
						Kind: yang.LeafEntry,
						Type: &yang.YangType{Kind: yang.Yint32},
					},
				},
			},
			"list-enum-keyed": {
				Name:     "list-enum-keyed",
				Kind:     yang.DirectoryEntry,
				ListAttr: yang.NewDefaultListAttr(),
				Key:      "key",
				Dir: map[string]*yang.Entry{
					"key": {
						Name: "key",
						Kind: yang.LeafEntry,
						Type: &yang.YangType{Kind: yang.Yenum},
					},
					"int32": {
						Name: "int32",
						Kind: yang.LeafEntry,
						Type: &yang.YangType{Kind: yang.Yint32},
					},
				},
			},
			"int32": {
				Name: "int32",
				Kind: yang.LeafEntry,
				Type: &yang.YangType{Kind: yang.Yint32},
			},
			"key": {
				Name: "key",
				Kind: yang.LeafEntry,
				Type: &yang.YangType{Kind: yang.Yint32},
			},
			"enum": {
				Name: "enum",
				Kind: yang.LeafEntry,
				Type: &yang.YangType{Kind: yang.Yenum},
			},
			"union": {
				Name: "union",
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
					},
				},
			},
			"container2": {
				Name: "container2",
				Kind: yang.DirectoryEntry,
				Dir: map[string]*yang.Entry{
					"int32-ref-to-leaf": {
						Name: "int32-ref-to-leaf",
						Kind: yang.LeafEntry,
						Type: &yang.YangType{
							Kind: yang.Yleafref,
							Path: "../../int32",
						},
					},
					"enum-ref-to-leaf": {
						Name: "enum-ref-to-leaf",
						Kind: yang.LeafEntry,
						Type: &yang.YangType{
							Kind: yang.Yleafref,
							Path: "../../enum",
						},
					},
					"int32-ref-to-leaf-list": {
						Name: "int32-ref-to-leaf-list",
						Kind: yang.LeafEntry,
						Type: &yang.YangType{
							Kind: yang.Yleafref,
							Path: "../../leaf-list",
						},
					},
					"leaf-list-ref-to-leaf-list": {
						Name: "leaf-list-ref-to-leaf-list",
						Kind: yang.LeafEntry,
						Type: &yang.YangType{
							Kind: yang.Yleafref,
							Path: "../../../leaf-list",
						},
						ListAttr: yang.NewDefaultListAttr(),
					},
					"int32-ref-to-list": {
						Name: "int32-ref-to-list",
						Kind: yang.LeafEntry,
						Type: &yang.YangType{
							Kind: yang.Yleafref,
							Path: "../../list[key = current()/../../key]/int32",
						},
					},
					"enum-ref-to-list": {
						Name: "int32-ref-to-list-enum-keyed",
						Kind: yang.LeafEntry,
						Type: &yang.YangType{
							Kind: yang.Yleafref,
							Path: "../../list-enum-keyed[key = current()/../../enum]/int32",
						},
					},
					"key": {
						Name: "key",
						Kind: yang.LeafEntry,
						Type: &yang.YangType{Kind: yang.Yint32},
					},
					"container3": {
						Name: "container3",
						Kind: yang.DirectoryEntry,
						Dir: map[string]*yang.Entry{
							"int32-ref-to-list": {
								Name: "int32-ref-to-list",
								Kind: yang.LeafEntry,
								Type: &yang.YangType{
									Kind: yang.Yleafref,
									Path: "../../../list[key = current()/../../key]/int32",
								},
							},
						},
					},
					"leaf-list-with-leafref": {
						Name: "leaf-list-with-leafref",
						Kind: yang.LeafEntry,
						Type: &yang.YangType{
							Kind: yang.Yleafref,
							Path: "../../int32",
						},
						ListAttr: yang.NewDefaultListAttr(),
					},
					"leaf-ref-to-union": {
						Name: "leaf-ref-to-union",
						Kind: yang.LeafEntry,
						Type: &yang.YangType{
							Kind: yang.Yleafref,
							Path: "../../union",
						},
					},
				},
			},
		},
	}

	type Container3 struct {
		LeafRefToList *int32 `path:"int32-ref-to-list"`
	}
	type Container2 struct {
		LeafRefToInt32         *int32             `path:"int32-ref-to-leaf"`
		LeafRefToEnum          EnumType           `path:"enum-ref-to-leaf"`
		LeafRefToLeafList      *int32             `path:"int32-ref-to-leaf-list"`
		LeafListRefToLeafList  []*int32           `path:"leaf-list-ref-to-leaf-list"`
		LeafRefToList          *int32             `path:"int32-ref-to-list"`
		LeafRefToListEnumKeyed *int32             `path:"int32-ref-to-list-enum-keyed"`
		Key                    *int32             `path:"key"`
		Container3             *Container3        `path:"container3"`
		LeafListLeafRefToInt32 []*int32           `path:"leaf-list-with-leafref"`
		LeafRefToUnion         testutil.TestUnion `path:"leaf-ref-to-union"`
	}
	type ListElement struct {
		Key   *int32 `path:"key"`
		Int32 *int32 `path:"int32"`
	}
	type ListElementEnumKeyed struct {
		Key   EnumType `path:"key"`
		Int32 *int32   `path:"int32"`
	}
	type Container struct {
		LeafList      []*int32                           `path:"leaf-list"`
		List          map[int32]*ListElement             `path:"list"`
		ListEnumKeyed map[EnumType]*ListElementEnumKeyed `path:"list-enum-keyed"`
		Int32         *int32                             `path:"int32"`
		Key           *int32                             `path:"key"`
		Enum          EnumType                           `path:"enum"`
		Container2    *Container2                        `path:"container2"`
		Union         testutil.TestUnion                 `path:"union"`
	}

	tests := []struct {
		desc    string
		in      interface{}
		opts    *LeafrefOptions
		wantErr string
	}{
		{
			desc: "nil",
			in:   nil,
		},
		{
			desc: "int32",
			in: &Container{
				Int32:      Int32(42),
				Container2: &Container2{LeafRefToInt32: Int32(42)},
			},
		},
		{
			desc: "int32 unequal",
			in: &Container{
				Int32:      Int32(42),
				Container2: &Container2{LeafRefToInt32: Int32(43)},
			},
			wantErr: `field name LeafRefToInt32 value 43 (int32 ptr) schema path /int32-ref-to-leaf has leafref path ../../int32 not equal to any target nodes`,
		},
		{
			desc: "int32 points to nil",
			in: &Container{
				Container2: &Container2{LeafRefToInt32: Int32(42)},
			},
			wantErr: `pointed-to value with path ../../int32 from field LeafRefToInt32 value 42 (int32 ptr) schema /int32-ref-to-leaf is empty set`,
		},
		{
			desc: "int32 points to nil with ignore missing data true",
			in: &Container{
				Container2: &Container2{LeafRefToInt32: Int32(42)},
			},
			opts: &LeafrefOptions{IgnoreMissingData: true},
		},
		{
			desc: "nil points to int32",
			in: &Container{
				Int32:      Int32(42),
				Container2: &Container2{},
			},
		},
		{
			desc: "enum",
			in: &Container{
				Enum:       EnumType(42),
				Container2: &Container2{LeafRefToEnum: EnumType(42)},
			},
		},
		{
			desc: "enum unequal",
			in: &Container{
				Enum:       42,
				Container2: &Container2{LeafRefToEnum: EnumType(43)},
			},
			wantErr: `field name LeafRefToEnum value out-of-range EnumType enum value: 43 (int64) schema path /enum-ref-to-leaf has leafref path ../../enum not equal to any target nodes`,
		},
		{
			desc: "leaf-list int32",
			in: &Container{
				LeafList:   []*int32{Int32(40), Int32(41), Int32(42)},
				Container2: &Container2{LeafRefToLeafList: Int32(42)},
			},
		},
		{
			desc: "leaf-list int32 missing",
			in: &Container{
				LeafList:   []*int32{Int32(40), Int32(41), Int32(42)},
				Container2: &Container2{LeafRefToLeafList: Int32(43)},
			},
			wantErr: `field name LeafRefToLeafList value 43 (int32 ptr) schema path /int32-ref-to-leaf-list has leafref path ../../leaf-list not equal to any target nodes`,
		},
		{
			desc: "leaf-list ref to leaf-list",
			in: &Container{
				LeafList:   []*int32{Int32(40), Int32(41), Int32(42)},
				Container2: &Container2{LeafListRefToLeafList: []*int32{Int32(41), Int32(42)}},
			},
		},
		{
			desc: "leaf-list ref to leaf-list not subset",
			in: &Container{
				LeafList:   []*int32{Int32(40), Int32(41), Int32(42)},
				Container2: &Container2{LeafListRefToLeafList: []*int32{Int32(41), Int32(42), Int32(43)}},
			},
			wantErr: `field name LeafListRefToLeafList value 43 (int32 ptr) schema path /leaf-list-ref-to-leaf-list has leafref path ../../../leaf-list not equal to any target nodes`,
		},
		{
			desc: "keyed list match",
			in: &Container{
				List: map[int32]*ListElement{
					1: {Int32(1), Int32(42)},
					2: {Int32(2), Int32(43)},
				},
				Key:        Int32(1),
				Container2: &Container2{LeafRefToList: Int32(42)},
			},
		},
		{
			desc: "keyed list unequal",
			in: &Container{
				List: map[int32]*ListElement{
					1: {Int32(1), Int32(42)},
					2: {Int32(2), Int32(43)},
				},
				Key:        Int32(1),
				Container2: &Container2{LeafRefToList: Int32(43)},
			},
			wantErr: `field name LeafRefToList value 43 (int32 ptr) schema path /int32-ref-to-list has leafref path ../../list[key = current()/../../key]/int32 not equal to any target nodes`,
		},
		{
			desc: "keyed list bad key value",
			in: &Container{
				List: map[int32]*ListElement{
					1: {Int32(1), Int32(42)},
					2: {Int32(2), Int32(43)},
				},
				Key:        Int32(3),
				Container2: &Container2{LeafRefToList: Int32(43)},
			},
			wantErr: `pointed-to value with path ../../list[key = current()/../../key]/int32 from field LeafRefToList value 43 (int32 ptr) schema /int32-ref-to-list is empty set`,
		},
		{
			// The idea for this test is that since "current()/../../key" depends on context,
			// the implementation should be getting distinct values for these correctly.
			desc: "different level keyed list, bad value on upper node",
			in: &Container{
				List: map[int32]*ListElement{
					1: {Int32(1), Int32(42)},
					2: {Int32(2), Int32(43)},
				},
				Key: Int32(3),
				Container2: &Container2{
					Key:           Int32(2),
					LeafRefToList: Int32(43),
					Container3: &Container3{
						LeafRefToList: Int32(43),
					},
				},
			},
			wantErr: `pointed-to value with path ../../list[key = current()/../../key]/int32 from field LeafRefToList value 43 (int32 ptr) schema /int32-ref-to-list is empty set`,
		},
		{
			// By swapping which of the upper/lower nodes is pointing to a bad value,
			// we make the testing more robust to implementation details, which may
			// allow one of these to pass.
			// e.g. it caches the results for "current()/../../key", but visits
			// the nodes in a certain order to make one of the tests pass.
			desc: "different level keyed list, bad value on lower node",
			in: &Container{
				List: map[int32]*ListElement{
					1: {Int32(1), Int32(42)},
					2: {Int32(2), Int32(43)},
				},
				Key: Int32(2),
				Container2: &Container2{
					Key:           Int32(3),
					LeafRefToList: Int32(43),
					Container3: &Container3{
						LeafRefToList: Int32(43),
					},
				},
			},
			wantErr: `pointed-to value with path ../../../list[key = current()/../../key]/int32 from field LeafRefToList value 43 (int32 ptr) schema /int32-ref-to-list is empty set`,
		},
		{
			desc: "enum keyed list match",
			in: &Container{
				ListEnumKeyed: map[EnumType]*ListElementEnumKeyed{
					EnumType(42): {Int32: Int32(1), Key: EnumType(42)},
					EnumType(43): {Int32: Int32(2), Key: EnumType(43)},
				},
				Enum:       EnumType(42),
				Container2: &Container2{LeafRefToListEnumKeyed: Int32(1)},
			},
		},
		{
			desc: "enum keyed list unequal",
			in: &Container{
				ListEnumKeyed: map[EnumType]*ListElementEnumKeyed{
					EnumType(42): {Int32: Int32(1), Key: EnumType(42)},
					EnumType(43): {Int32: Int32(2), Key: EnumType(43)},
				},
				Enum:       EnumType(42),
				Container2: &Container2{LeafRefToListEnumKeyed: Int32(2)},
			},
			wantErr: `field name LeafRefToListEnumKeyed value 2 (int32 ptr) schema path /int32-ref-to-list-enum-keyed has leafref path ../../list-enum-keyed[key = current()/../../enum]/int32 not equal to any target nodes`,
		},
		{
			desc: "enum keyed list bad key value",
			in: &Container{
				ListEnumKeyed: map[EnumType]*ListElementEnumKeyed{
					EnumType(43): {Int32: Int32(2), Key: EnumType(43)},
				},
				Enum:       EnumType(42),
				Container2: &Container2{LeafRefToListEnumKeyed: Int32(1)},
			},
			wantErr: `pointed-to value with path ../../list-enum-keyed[key = current()/../../enum]/int32 from field LeafRefToListEnumKeyed value 1 (int32 ptr) schema /int32-ref-to-list-enum-keyed is empty set`,
		},
		{
			// By swapping which of the upper/lower nodes is pointing to a bad value,
			// we make the testing more robust to implementation details, which may
			// allow one of these to pass.
			// e.g. it caches the results for "current()/../../key", but visits
			// the nodes in a certain order to make one of the tests pass.
			desc: "different level keyed list, bad value on lower node",
			in: &Container{
				List: map[int32]*ListElement{
					1: {Int32(1), Int32(42)},
					2: {Int32(2), Int32(43)},
				},
				Key: Int32(2),
				Container2: &Container2{
					Key:           Int32(3),
					LeafRefToList: Int32(43),
					Container3: &Container3{
						LeafRefToList: Int32(43),
					},
				},
			},
			wantErr: `pointed-to value with path ../../../list[key = current()/../../key]/int32 from field LeafRefToList value 43 (int32 ptr) schema /int32-ref-to-list is empty set`,
		},
		{
			desc: "union leafref - string",
			in: &Container{
				Union: testutil.UnionString("val"),
				Container2: &Container2{
					LeafRefToUnion: testutil.UnionString("val"),
				},
			},
		},
		{
			desc: "union leafref - integer",
			in: &Container{
				Union: testutil.UnionInt16(42),
				Container2: &Container2{
					LeafRefToUnion: testutil.UnionInt16(42),
				},
			},
		},
		{
			desc: "union leafref - failure",
			in: &Container{
				Union: testutil.UnionInt16(42),
				Container2: &Container2{
					LeafRefToUnion: testutil.UnionInt16(4444),
				},
			},
			wantErr: "field name LeafRefToUnion value 4444 (int16) schema path /leaf-ref-to-union has leafref path ../../union not equal to any target nodes",
		},
		{
			desc: "union (wrapped union) leafref - string",
			in: &Container{
				Union: &Union1String{"val"},
				Container2: &Container2{
					LeafRefToUnion: &Union1String{"val"},
				},
			},
		},
		{
			desc: "union (wrapped union) leafref - integer",
			in: &Container{
				Union: &Union1Int16{42},
				Container2: &Container2{
					LeafRefToUnion: &Union1Int16{42},
				},
			},
		},
		{
			desc: "union (wrapped union) leafref - failure",
			in: &Container{
				Union: &Union1Int16{42},
				Container2: &Container2{
					LeafRefToUnion: &Union1Int16{4444},
				},
			},
			wantErr: "field name LeafRefToUnion value { 4444 (int16 ptr) } schema path /leaf-ref-to-union has leafref path ../../union not equal to any target nodes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			errs := ValidateLeafRefData(containerWithLeafListSchema, tt.in, tt.opts)
			if got, want := errs.String(), tt.wantErr; got != want {
				t.Errorf("%s: got error: %s, want error: %s", tt.desc, got, want)
			}
			testErrLog(t, tt.desc, errs)
		})
	}
}

func TestValidateLeafRefDataCompressedSchemaListOnly(t *testing.T) {
	// YANG Schema (details mostly matches):
	// container root {
	//   container examples {
	//     list example {
	//
	//       key "conf";
	//       description
	//         "top-level list for the example data";
	//
	//       leaf conf {
	//         type leafref {
	//           path "../config/conf";
	//         }
	//       }
	//
	//       container config {
	//         leaf conf {
	//           type uint32;
	//         }
	//         leaf conf-ref {
	//           type leafref {
	//             path "../conf";
	//           }
	//         }
	//         leaf conf2-ref {
	//           type leafref {
	//             path "../../../../conf2";
	//           }
	//         }
	//       }
	//     }
	//   }
	//   leaf conf2 {
	//     type string;
	//   }
	// }
	containerWithListSchema := &yang.Entry{
		Kind: yang.LeafEntry,
		Dir: map[string]*yang.Entry{
			"root": {
				Name: "root",
				Kind: yang.DirectoryEntry,
				Dir: map[string]*yang.Entry{
					"examples": {
						Name: "examples",
						Kind: yang.DirectoryEntry,
						Dir: map[string]*yang.Entry{
							"example": {
								Name:     "example",
								Kind:     yang.DirectoryEntry,
								ListAttr: &yang.ListAttr{},
								Dir: map[string]*yang.Entry{
									"conf": {
										Name: "conf",
										Kind: yang.LeafEntry,
										Type: &yang.YangType{
											Kind: yang.Yleafref,
											Path: "../config/conf",
										},
									},
									"config": {
										Name: "config",
										Kind: yang.DirectoryEntry,
										Dir: map[string]*yang.Entry{
											"conf": {
												Name: "conf",
												Kind: yang.LeafEntry,
												Type: &yang.YangType{Kind: yang.Yint32},
											},
											"conf-ref": {
												Name: "conf-ref",
												Kind: yang.LeafEntry,
												Type: &yang.YangType{
													Kind: yang.Yleafref,
													Path: "../conf",
												},
											},
											"conf2-ref": {
												Name: "conf2-ref",
												Kind: yang.LeafEntry,
												Type: &yang.YangType{
													Kind: yang.Yleafref,
													Path: "../../../../conf2",
												},
											},
										},
									},
								},
							},
						},
					},
					"conf2": {
						Name: "conf2",
						Kind: yang.LeafEntry,
						Type: &yang.YangType{Kind: yang.Ystring},
					},
				},
			},
		},
		Annotation: map[string]interface{}{"isCompressedSchema": true, "isFakeRoot": true},
	}
	// Set the parent pointers
	addParents(containerWithListSchema)

	rootSchema := containerWithListSchema.Dir["root"]

	type RootExample struct {
		Conf     *uint32 `path:"config/conf|conf"`
		ConfRef  *uint32 `path:"config/conf-ref"`
		Conf2Ref *string `path:"config/conf2-ref"`
	}

	type Root struct {
		Conf2   *string                 `path:"conf2"`
		Example map[uint32]*RootExample `path:"examples/example"`
	}

	tests := []struct {
		desc    string
		in      interface{}
		opts    *LeafrefOptions
		wantErr string
	}{
		{
			desc: "nil",
			in:   nil,
		},
		{
			desc: "list key leafref (conf)",
			in: &Root{
				Example: map[uint32]*RootExample{42: {Conf: Uint32(42)}},
			},
		},
		{
			desc: "ref to leaf outside of list (conf2)",
			in: &Root{
				Conf2:   String("hitchhiker"),
				Example: map[uint32]*RootExample{42: {Conf: Uint32(42), Conf2Ref: String("hitchhiker")}},
			},
		},
		{
			desc: "ref to leaf outside of list (conf2) unequal",
			in: &Root{
				Conf2:   String("hitchhiker"),
				Example: map[uint32]*RootExample{42: {Conf: Uint32(42), Conf2Ref: String("haichhicker")}},
			},
			wantErr: `field name Conf2Ref value haichhicker (string ptr) schema path //root/examples/example/config/conf2-ref has leafref path ../../../../conf2 not equal to any target nodes`,
		},
		{
			desc: "ref to leaf outside of list (conf2) points to nil",
			in: &Root{
				Example: map[uint32]*RootExample{42: {Conf: Uint32(42), Conf2Ref: String("hitchhiker")}},
			},
			wantErr: `pointed-to value with path ../../../../conf2 from field Conf2Ref value hitchhiker (string ptr) schema //root/examples/example/config/conf2-ref is empty set`,
		},
		{
			desc: "ref to leaf outside of list (conf2) points to nil with ignore missing data true",
			in: &Root{
				Example: map[uint32]*RootExample{42: {Conf: Uint32(42), Conf2Ref: String("hitchhiker")}},
			},
			opts: &LeafrefOptions{IgnoreMissingData: true},
		},
		{
			desc: "nil points to conf2",
			in: &Root{
				Conf2:   String("hitchhiker"),
				Example: map[uint32]*RootExample{},
			},
		},
		{
			desc: "conf-ref",
			in: &Root{
				Example: map[uint32]*RootExample{42: {Conf: Uint32(42), ConfRef: Uint32(42)}},
			},
		},
		{
			desc: "conf-ref unequal to conf",
			in: &Root{
				Example: map[uint32]*RootExample{42: {Conf: Uint32(42), ConfRef: Uint32(43)}},
			},
			wantErr: `field name ConfRef value 43 (uint32 ptr) schema path //root/examples/example/config/conf-ref has leafref path ../conf not equal to any target nodes`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			errs := ValidateLeafRefData(rootSchema, tt.in, tt.opts)
			if got, want := errs.String(), tt.wantErr; got != want {
				t.Errorf("%s: got error: %s, want error: %s", tt.desc, got, want)
			}
			testErrLog(t, tt.desc, errs)
		})
	}
}

func TestSplitUnescaped(t *testing.T) {
	tests := []struct {
		desc string
		in   string
		want []string
	}{
		{
			desc: "simple",
			in:   "a/b/c",
			want: []string{"a", "b", "c"},
		},
		{
			desc: "blank",
			in:   "a//b",
			want: []string{"a", "", "b"},
		},
		{
			desc: "lead trail slash",
			in:   "/a/b/c/",
			want: []string{"", "a", "b", "c", ""},
		},
		{
			desc: "escape slash",
			in:   `a/\/b/c`,
			want: []string{"a", `\/b`, "c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got, want := splitUnescaped(tt.in, '/'), tt.want
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("%s: (-want, +got):\n%s", tt.desc, diff)
			}
		})
	}
}

func TestSplitUnquoted(t *testing.T) {
	tests := []struct {
		desc     string
		in       string
		splitStr string
		want     []string
	}{
		{
			desc:     "simple",
			in:       "a/b/c",
			splitStr: "/",
			want:     []string{"a", "b", "c"},
		},
		{
			desc:     "blank",
			in:       "a//b",
			splitStr: "/",
			want:     []string{"a", "", "b"},
		},
		{
			desc:     "lead trail slash",
			in:       "/a/b/c/",
			splitStr: "/",
			want:     []string{"", "a", "b", "c", ""},
		},
		{
			desc:     "quoted",
			in:       `a/"/"b"/"/c`,
			splitStr: "/",
			want:     []string{"a", `"/"b"/"`, "c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got, want := splitUnquoted(tt.in, tt.splitStr), tt.want
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("%s: (-want, +got):\n%s", tt.desc, diff)
			}
		})
	}
}

func TestExtractKeyValue(t *testing.T) {
	tests := []struct {
		desc       string
		in         string
		wantErr    string
		wantPrefix string
		wantKey    string
		wantValue  string
	}{
		{
			desc:       "literal",
			in:         `b[key = "value"]`,
			wantPrefix: "b",
			wantKey:    "key",
			wantValue:  `"value"`,
		},
		{
			desc:       "spacing",
			in:         `b[key="value"]`,
			wantPrefix: "b",
			wantKey:    "key",
			wantValue:  `"value"`,
		},
		{
			desc:       "quotes",
			in:         `b[key="[=value=]"]`,
			wantPrefix: "b",
			wantKey:    "key",
			wantValue:  `"[=value=]"`,
		},
		{
			desc:       "path",
			in:         "b[key = current()/../a/b/c]",
			wantPrefix: "b",
			wantKey:    "key",
			wantValue:  "current()/../a/b/c",
		},
		{
			desc:       "path",
			in:         "b[key = ../a/b/c]",
			wantPrefix: "b",
			wantErr:    `bad kv string key = ../a/b/c: value must be in quotes or begin with current()/`,
		},
		{
			desc:       "escapes",
			in:         `b\[[\[key\]\" = "[a]"]`,
			wantPrefix: `b\[`,
			wantKey:    `\[key\]\"`,
			wantValue:  `"[a]"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			prefix, k, v, err := extractKeyValue(tt.in)
			if got, want := errToString(err), tt.wantErr; got != want {
				t.Errorf("%s: got error: %s, want error: %s", tt.desc, got, want)
			}
			if err != nil {
				return
			}
			if got, want := prefix, tt.wantPrefix; got != want {
				t.Errorf("%s prefix: got: %s, want: %s", tt.desc, got, want)
			}
			got, want := k, tt.wantKey
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("%s key: (-want, +got):\n%s", tt.desc, diff)
			}
			got, want = v, tt.wantValue
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("%s value: (-want, +got):\n%s", tt.desc, diff)
			}
		})
	}
}

func TestIsKeyValue(t *testing.T) {
	tests := []struct {
		name             string
		in               string
		want             bool
		wantErrSubstring string
	}{{
		name: "no quotes",
		in:   "foo[baz=bar]",
		want: true,
	}, {
		name: "quotes",
		in:   `foo[bar="baz"]`,
		want: true,
	}, {
		name:             "no key",
		in:               "",
		wantErrSubstring: "empty path element",
	}, {
		name:             "malformed",
		in:               "foo]",
		wantErrSubstring: "malformed path element",
	}, {
		name:             "trailing chars",
		in:               "foo[bar=baz]trailing",
		wantErrSubstring: "trailing chars after",
	}}

	for _, tt := range tests {
		got, err := isKeyValue(tt.in)
		if diff := errdiff.Substring(err, tt.wantErrSubstring); diff != "" {
			t.Errorf("%s: isKeyValue(%v): did not get expected error, %s", tt.name, tt.in, diff)
		}

		if err != nil {
			continue
		}

		if got != tt.want {
			t.Errorf("%s: isKeyValue(%v): did not get expected value, got: %v, want: %v", tt.name, tt.in, got, tt.want)
		}
	}
}

func TestPathMatchesPrefix(t *testing.T) {
	tests := []struct {
		name     string
		inPath   []string
		inPrefix []string
		want     bool
	}{{
		name:     "short path",
		inPath:   []string{"one"},
		inPrefix: []string{"two", "three"},
		want:     false,
	}, {
		name:     "path does not match",
		inPath:   []string{"one", "two", "three"},
		inPrefix: []string{"one", "four"},
		want:     false,
	}, {
		name:     "path matches prefix",
		inPath:   []string{"one", "two"},
		inPrefix: []string{"one"},
		want:     true,
	}}

	for _, tt := range tests {
		if got := pathMatchesPrefix(tt.inPath, tt.inPrefix); got != tt.want {
			t.Errorf("%s: pathMatchesPrefix(%v, %v): did not get expected output, got: %v, want: %v", tt.name, tt.inPath, tt.inPrefix, got, tt.want)
		}
	}
}

func Int32(i int32) *int32    { return &i }
func Uint32(i uint32) *uint32 { return &i }
func String(s string) *string { return &s }

type genericList struct {
	Key *uint32 `path:"key"`
	Val *uint32 `path:"val"`
}

type root struct {
	Target map[uint32]*genericList `path:"target"`
	Ref    map[uint32]*genericList `path:"ref"`
}

func TestLeafrefValidateCurrent(t *testing.T) {
	// This test checks against an uncompressed schema to determine whether we are able to validate references
	// correctly. It covers an ugly bit of logic concerning having two entries for lists in the data tree (one
	// which is the member of the list, and one which is the map.
	//
	// TODO(robjs): Seriously think about whether we should refactor here, this code is very complex to understand
	// and even harder to debug. :-( I think we made a mistake here.

	rootSchema := &yang.Entry{
		Name: "root",
		Kind: yang.DirectoryEntry,
		Dir:  map[string]*yang.Entry{},
		Annotation: map[string]interface{}{
			"isFakeRoot": true,
		},
	}
	targetListSchema := &yang.Entry{
		Name:     "target",
		Key:      "key",
		Kind:     yang.DirectoryEntry,
		ListAttr: &yang.ListAttr{},
		Parent:   rootSchema,
	}
	rootSchema.Dir["target"] = targetListSchema
	targetListSchema.Dir = map[string]*yang.Entry{
		"key": {
			Name:   "key",
			Kind:   yang.LeafEntry,
			Type:   &yang.YangType{Kind: yang.Yuint32},
			Parent: targetListSchema,
		},
		"val": {
			Name:   "val",
			Kind:   yang.LeafEntry,
			Type:   &yang.YangType{Kind: yang.Yuint32},
			Parent: targetListSchema,
		},
	}

	refListSchema := &yang.Entry{
		Name:     "ref",
		Kind:     yang.DirectoryEntry,
		Key:      "key",
		ListAttr: &yang.ListAttr{},
		Parent:   rootSchema,
	}
	rootSchema.Dir["ref"] = refListSchema
	refListSchema.Dir = map[string]*yang.Entry{
		"key": {
			Name:   "key",
			Kind:   yang.LeafEntry,
			Type:   &yang.YangType{Kind: yang.Yuint32},
			Parent: refListSchema,
		},
		"val": {
			Name: "val",
			Kind: yang.LeafEntry,
			Type: &yang.YangType{
				Kind: yang.Yleafref,
				Path: "../../target[key=current()/../key]/val",
			},
			Parent: refListSchema,
		},
	}

	tests := []struct {
		desc             string
		inSchema         *yang.Entry
		inValue          interface{}
		inOpts           *LeafrefOptions
		wantErrSubstring string
	}{{
		desc:     "succeeding relative reference",
		inSchema: rootSchema,
		inValue: &root{
			Target: map[uint32]*genericList{
				1: {ygot.Uint32(1), ygot.Uint32(42)},
				2: {ygot.Uint32(2), ygot.Uint32(422)},
			},
			Ref: map[uint32]*genericList{
				1: {ygot.Uint32(1), ygot.Uint32(42)},
				2: {ygot.Uint32(2), ygot.Uint32(422)},
			},
		},
	}, {
		desc:     "failing relative reference",
		inSchema: rootSchema,
		inValue: &root{
			Target: map[uint32]*genericList{
				1: {ygot.Uint32(1), ygot.Uint32(42)},
				2: {ygot.Uint32(2), ygot.Uint32(422)},
			},
			Ref: map[uint32]*genericList{
				1: {ygot.Uint32(1), ygot.Uint32(422)}, // this should fail -- since we're looking for /target[key=1]/value = 422 which isn't there in the data.
				2: {ygot.Uint32(2), ygot.Uint32(422)},
			},
		},
		wantErrSubstring: "not equal to any target nodes",
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			errs := ValidateLeafRefData(tt.inSchema, tt.inValue, tt.inOpts)
			switch {
			case errs == nil && tt.wantErrSubstring != "":
				t.Fatalf("unexpectedly got nil errors, want: %s", tt.wantErrSubstring)
			case tt.wantErrSubstring != "" && !strings.Contains(errs.String(), tt.wantErrSubstring):
				t.Fatalf("did not get expected error, got: %s, want error containing: %s", errs, tt.wantErrSubstring)
			}
		})
	}
}
