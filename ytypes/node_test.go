// Copyright 2020 Google Inc.
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

	"github.com/google/go-cmp/cmp"
	"github.com/openconfig/gnmi/errdiff"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygot"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
)

type InnerContainerType1 struct {
	Int32LeafName     *int32            `path:"config/int32-leaf-field|int32-leaf-field" shadow-path:"state/int32-leaf-field|int32-leaf-field"`
	Int32LeafListName []int32           `path:"int32-leaf-list"`
	StringLeafName    *string           `path:"string-leaf-field" shadow-path:"state/string-leaf-field|string-leaf-field"`
	EnumLeafName      EnumType          `path:"enum-leaf-field"`
	Annotation        []ygot.Annotation `path:"@annotation" ygotAnnotation:"true"`
}

func (*InnerContainerType1) IsYANGGoStruct() {}

type OuterContainerType1 struct {
	Inner *InnerContainerType1 `path:"inner|config/inner"`
}

func (*OuterContainerType1) IsYANGGoStruct() {}

type ListElemStruct1 struct {
	Key1       *string              `path:"key1"`
	Outer      *OuterContainerType1 `path:"outer"`
	Annotation []ygot.Annotation    `path:"@annotation" ygotAnnotation:"true"`
}

func (l *ListElemStruct1) ΛListKeyMap() (map[string]interface{}, error) {
	return map[string]interface{}{
		"key1":  l.Key1,
		"outer": l.Outer,
	}, nil
}
func (*ListElemStruct1) IsYANGGoStruct() {}

type ContainerStruct1 struct {
	StructKeyList map[string]*ListElemStruct1 `path:"config/simple-key-list"`
}

func (*ContainerStruct1) IsYANGGoStruct() {}

type ListElemStruct2 struct {
	Key1       *uint32              `path:"key1"`
	Outer      *OuterContainerType1 `path:"outer"`
	Annotation []ygot.Annotation    `path:"@annotation" ygotAnnotation:"true"`
}

func (l *ListElemStruct2) ΛListKeyMap() (map[string]interface{}, error) {
	return map[string]interface{}{
		"key1":  l.Key1,
		"outer": l.Outer,
	}, nil
}

type ContainerStruct2 struct {
	StructKeyList map[uint32]*ListElemStruct2 `path:"config/simple-key-list"`
}

type ListElemEnumKey struct {
	Key1          EnumType `path:"key1"`
	Int32LeafName *int32   `path:"int32-leaf-field"`
}

type ContainerEnumKey struct {
	StructKeyList map[EnumType]*ListElemEnumKey `path:"config/simple-key-list"`
}

type ListElemBoolKey struct {
	Key1          *bool  `path:"key1"`
	Int32LeafName *int32 `path:"int32-leaf-field"`
}

type ContainerBoolKey struct {
	StructKeyList map[bool]*ListElemBoolKey `path:"config/simple-key-list"`
}

type ListElemStruct4 struct {
	Key1 *uint32 `path:"key1"`
}

var listElemStruct4Schema = &yang.Entry{
	Name: "list-elem-struct4",
	Kind: yang.DirectoryEntry,
	Dir: map[string]*yang.Entry{
		"key1": {
			Name: "key1",
			Kind: yang.LeafEntry,
			Type: &yang.YangType{Kind: yang.Yuint32},
		},
	},
}

type SuperContainer struct {
	ContainerStruct1 *ContainerStruct1 `path:"container"`
}

func (*SuperContainer) IsYANGGoStruct() {}

var superContainerSchema = &yang.Entry{
	Name: "super-container",
	Kind: yang.DirectoryEntry,
	Dir: map[string]*yang.Entry{
		"container": containerWithStringKey(),
	},
}

func containerWithStringKey() *yang.Entry {
	containerWithStringKey := &yang.Entry{
		Name: "container",
		Kind: yang.DirectoryEntry,
		Dir: map[string]*yang.Entry{
			"config": {
				Name: "config",
				Kind: yang.DirectoryEntry,
				Dir: map[string]*yang.Entry{
					"simple-key-list": {
						Name:     "simple-key-list",
						Kind:     yang.DirectoryEntry,
						ListAttr: yang.NewDefaultListAttr(),
						Key:      "key1",
						Config:   yang.TSTrue,
						Dir: map[string]*yang.Entry{
							"key1": {
								Name: "key1",
								Kind: yang.LeafEntry,
								Type: &yang.YangType{Kind: yang.Ystring},
							},
							"outer": {
								Name: "outer",
								Kind: yang.DirectoryEntry,
								Dir: map[string]*yang.Entry{
									"config": {
										Name: "config",
										Kind: yang.DirectoryEntry,
										Dir: map[string]*yang.Entry{
											"inner": {
												Name: "inner",
												Kind: yang.DirectoryEntry,
												Dir: map[string]*yang.Entry{
													"int32-leaf-field": {
														Name: "int32-leaf-field",
														Kind: yang.LeafEntry,
														Type: &yang.YangType{Kind: yang.Yint32},
													},
													"config": {
														Name: "config",
														Kind: yang.DirectoryEntry,
														Dir: map[string]*yang.Entry{
															"int32-leaf-field": {
																Name: "int32-leaf-field",
																Kind: yang.LeafEntry,
																Type: &yang.YangType{Kind: yang.Yint32},
															},
														},
													},
													"state": {
														Name: "state",
														Kind: yang.DirectoryEntry,
														Dir: map[string]*yang.Entry{
															"int32-leaf-field": {
																Name: "int32-leaf-field",
																Kind: yang.LeafEntry,
																Type: &yang.YangType{Kind: yang.Yint32},
															},
															"string-leaf-field": {
																Name: "string-leaf-field",
																Kind: yang.LeafEntry,
																Type: &yang.YangType{Kind: yang.Ystring},
															},
														},
													},
													"int32-leaf-list": {
														Name:     "int32-leaf-list",
														Kind:     yang.LeafEntry,
														ListAttr: yang.NewDefaultListAttr(),
														Type:     &yang.YangType{Kind: yang.Yint32},
													},
													"string-leaf-field": {
														Name: "string-leaf-field",
														Kind: yang.LeafEntry,
														Type: &yang.YangType{Kind: yang.Ystring},
													},
													"enum-leaf-field": {
														Name: "enum-leaf-field",
														Kind: yang.LeafEntry,
														Type: &yang.YangType{Kind: yang.Yenum},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	addParents(containerWithStringKey)
	return containerWithStringKey
}

func TestGetOrCreateNodeSimpleKey(t *testing.T) {
	containerWithUInt32Key := &yang.Entry{
		Name: "container",
		Kind: yang.DirectoryEntry,
		Dir: map[string]*yang.Entry{
			"config": {
				Name: "config",
				Kind: yang.DirectoryEntry,
				Dir: map[string]*yang.Entry{
					"simple-key-list": {
						Name:     "simple-key-list",
						Kind:     yang.DirectoryEntry,
						ListAttr: yang.NewDefaultListAttr(),
						Key:      "key1",
						Config:   yang.TSTrue,
						Dir: map[string]*yang.Entry{
							"key1": {
								Name: "key1",
								Kind: yang.LeafEntry,
								Type: &yang.YangType{Kind: yang.Yuint32},
							},
							"outer": {
								Name: "outer",
								Kind: yang.DirectoryEntry,
								Dir: map[string]*yang.Entry{
									"config": {
										Name: "config",
										Kind: yang.DirectoryEntry,
										Dir: map[string]*yang.Entry{
											"inner": {
												Name: "inner",
												Kind: yang.DirectoryEntry,
												Dir: map[string]*yang.Entry{
													"int32-leaf-field": {
														Name: "leaf-field",
														Kind: yang.LeafEntry,
														Type: &yang.YangType{Kind: yang.Yint32},
													},
													"config": {
														Name: "config",
														Kind: yang.DirectoryEntry,
														Dir: map[string]*yang.Entry{
															"int32-leaf-field": {
																Name: "int32-leaf-field",
																Kind: yang.LeafEntry,
																Type: &yang.YangType{Kind: yang.Yint32},
															},
														},
													},
													"state": {
														Name: "state",
														Kind: yang.DirectoryEntry,
														Dir: map[string]*yang.Entry{
															"int32-leaf-field": {
																Name: "int32-leaf-field",
																Kind: yang.LeafEntry,
																Type: &yang.YangType{Kind: yang.Yint32},
															},
														},
													},
													"string-leaf-field": {
														Name: "leaf-field",
														Kind: yang.LeafEntry,
														Type: &yang.YangType{Kind: yang.Ystring},
													},
													"enum-leaf-field": {
														Name: "leaf-field",
														Kind: yang.LeafEntry,
														Type: &yang.YangType{Kind: yang.Yenum},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	containerWithEnumKey := &yang.Entry{
		Name: "container",
		Kind: yang.DirectoryEntry,
		Dir: map[string]*yang.Entry{
			"config": {
				Name: "config",
				Kind: yang.DirectoryEntry,
				Dir: map[string]*yang.Entry{
					"simple-key-list": {
						Name:     "simple-key-list",
						Kind:     yang.DirectoryEntry,
						ListAttr: yang.NewDefaultListAttr(),
						Key:      "key1",
						Config:   yang.TSTrue,
						Dir: map[string]*yang.Entry{
							"key1": {
								Name: "key1",
								Kind: yang.LeafEntry,
								Type: &yang.YangType{Kind: yang.Yenum},
							},
						},
					},
				},
			},
		},
	}

	containerWithBoolKey := &yang.Entry{
		Name: "container",
		Kind: yang.DirectoryEntry,
		Dir: map[string]*yang.Entry{
			"config": {
				Name: "config",
				Kind: yang.DirectoryEntry,
				Dir: map[string]*yang.Entry{
					"simple-key-list": {
						Name:     "simple-key-list",
						Kind:     yang.DirectoryEntry,
						ListAttr: yang.NewDefaultListAttr(),
						Key:      "key1",
						Config:   yang.TSTrue,
						Dir: map[string]*yang.Entry{
							"key1": {
								Name: "key1",
								Kind: yang.LeafEntry,
								Type: &yang.YangType{Kind: yang.Ybool},
							},
						},
					},
				},
			},
		},
	}

	tests := []struct {
		inDesc           string
		inParent         interface{}
		inSchema         *yang.Entry
		inPath           *gpb.Path
		inOpts           []GetOrCreateNodeOpt
		want             interface{}
		wantErrSubstring string
	}{
		{
			inDesc: "success get int32 leaf with an existing key",
			inParent: &ContainerStruct1{
				StructKeyList: map[string]*ListElemStruct1{
					"forty-two": {
						Key1: ygot.String("forty-two"),
						Outer: &OuterContainerType1{
							Inner: &InnerContainerType1{
								Int32LeafName: ygot.Int32(42),
							},
						},
					},
				},
			},
			inSchema: containerWithStringKey(),
			inPath:   mustPath("/config/simple-key-list[key1=forty-two]/outer/inner/int32-leaf-field"),
			want:     ygot.Int32(42),
		},
		{
			inDesc:   "success get int32 leaf with a new key",
			inParent: &ContainerStruct1{},
			inSchema: containerWithStringKey(),
			inPath:   mustPath("/config/simple-key-list[key1=forty-two]/outer/inner/int32-leaf-field"),
			want:     ygot.Int32(0),
		},
		{
			inDesc: "success get string leaf with an existing key",
			inParent: &ContainerStruct1{
				StructKeyList: map[string]*ListElemStruct1{
					"forty-two": {
						Key1: ygot.String("forty-two"),
						Outer: &OuterContainerType1{
							Inner: &InnerContainerType1{
								StringLeafName: ygot.String("forty_two"),
							},
						},
					},
				},
			},
			inSchema: containerWithStringKey(),
			inPath:   mustPath("/config/simple-key-list[key1=forty-two]/outer/inner/string-leaf-field"),
			want:     ygot.String("forty_two"),
		},
		{
			inDesc:   "success get string leaf with a new key",
			inParent: &ContainerStruct1{},
			inSchema: containerWithStringKey(),
			inPath:   mustPath("/config/simple-key-list[key1=forty-two]/outer/inner/string-leaf-field"),
			want:     ygot.String(""),
		},
		{
			inDesc: "success get enum leaf with an existing key",
			inParent: &ContainerStruct1{
				StructKeyList: map[string]*ListElemStruct1{
					"forty-two": {
						Key1: ygot.String("forty-two"),
						Outer: &OuterContainerType1{
							Inner: &InnerContainerType1{
								EnumLeafName: EnumType(43),
							},
						},
					},
				},
			},
			inSchema: containerWithStringKey(),
			inPath:   mustPath("/config/simple-key-list[key1=forty-two]/outer/inner/enum-leaf-field"),
			want:     EnumType(43),
		},
		{
			inDesc:   "success get enum leaf with a new key",
			inParent: &ContainerStruct1{},
			inSchema: containerWithStringKey(),
			inPath:   mustPath("/config/simple-key-list[key1=forty-two]/outer/inner/enum-leaf-field"),
			want:     EnumType(0),
		},
		{
			inDesc:           "fail get enum leaf incorrect container schema",
			inParent:         &ContainerStruct1{},
			inSchema:         containerWithStringKey(),
			inPath:           mustPath("/config/simple-key-list[key1=forty-two]/INVAID_CONTAINER/inner/enum-leaf-field"),
			wantErrSubstring: "no match found in *ytypes.ListElemStruct1",
		},
		{
			inDesc:           "fail get enum leaf incorrect leaf schema",
			inParent:         &ContainerStruct1{},
			inSchema:         containerWithStringKey(),
			inPath:           mustPath("/config/simple-key-list[key1=forty-two]/outer/inner/INVALID_LEAF"),
			wantErrSubstring: "no match found in *ytypes.InnerContainerType1",
		},
		{
			inDesc:   "success getting a nil shadow value",
			inParent: &ContainerStruct1{},
			inSchema: containerWithStringKey(),
			inPath:   mustPath("/config/simple-key-list[key1=forty-two]/outer/inner/state/int32-leaf-field"),
			want:     nil,
		},
		{
			inDesc:   "success getting an initialized non-shadow value",
			inParent: &ContainerStruct1{},
			inSchema: containerWithStringKey(),
			inPath:   mustPath("/config/simple-key-list[key1=forty-two]/outer/inner/config/int32-leaf-field"),
			want:     ygot.Int32(0),
		},
		{
			inDesc: "success getting nil shadow int32 leaf with an existing key",
			inParent: &ContainerStruct1{
				StructKeyList: map[string]*ListElemStruct1{
					"forty-two": {
						Key1: ygot.String("forty-two"),
						Outer: &OuterContainerType1{
							Inner: &InnerContainerType1{
								Int32LeafName: ygot.Int32(42),
							},
						},
					},
				},
			},
			inSchema: containerWithStringKey(),
			inPath:   mustPath("/config/simple-key-list[key1=forty-two]/outer/inner/state/int32-leaf-field"),
			want:     nil,
		},
		{
			inDesc:           "fail getting a shadow value whose container doesn't exist",
			inParent:         &ContainerStruct1{},
			inSchema:         containerWithStringKey(),
			inPath:           mustPath("/config/simple-key-list[key1=forty-two]/outer/INVALID_CONTAINER/state/int32-leaf-field"),
			wantErrSubstring: "no match found in *ytypes.OuterContainerType1",
		},
		{
			inDesc:   "success getting a nil non-shadow value when reverseShadowPath=true",
			inParent: &ContainerStruct1{},
			inSchema: containerWithStringKey(),
			inPath:   mustPath("/config/simple-key-list[key1=forty-two]/outer/inner/config/int32-leaf-field"),
			inOpts:   []GetOrCreateNodeOpt{&ReverseShadowPaths{}},
			want:     nil,
		},
		{
			inDesc:   "success getting an initialized shadow value with reverseShadowPath=true",
			inParent: &ContainerStruct1{},
			inSchema: containerWithStringKey(),
			inPath:   mustPath("/config/simple-key-list[key1=forty-two]/outer/inner/state/int32-leaf-field"),
			inOpts:   []GetOrCreateNodeOpt{&ReverseShadowPaths{}},
			want:     ygot.Int32(0),
		},
		{
			inDesc: "success getting a shadow int32 leaf with an existing key with reverseShadowPath=true",
			inParent: &ContainerStruct1{
				StructKeyList: map[string]*ListElemStruct1{
					"forty-two": {
						Key1: ygot.String("forty-two"),
						Outer: &OuterContainerType1{
							Inner: &InnerContainerType1{
								Int32LeafName: ygot.Int32(42),
							},
						},
					},
				},
			},
			inSchema: containerWithStringKey(),
			inPath:   mustPath("/config/simple-key-list[key1=forty-two]/outer/inner/state/int32-leaf-field"),
			inOpts:   []GetOrCreateNodeOpt{&ReverseShadowPaths{}},
			want:     ygot.Int32(42),
		},
		{
			inDesc:           "fail getting a shadow value whose container doesn't exist with reverseShadowPath=true",
			inParent:         &ContainerStruct1{},
			inSchema:         containerWithStringKey(),
			inPath:           mustPath("/config/simple-key-list[key1=forty-two]/outer/INVALID_CONTAINER/state/int32-leaf-field"),
			inOpts:           []GetOrCreateNodeOpt{&ReverseShadowPaths{}},
			wantErrSubstring: "no match found in *ytypes.OuterContainerType1",
		},
		{
			inDesc:           "fail getting a value that doesn't exist with reverseShadowPath=true",
			inParent:         &ContainerStruct1{},
			inSchema:         containerWithStringKey(),
			inPath:           mustPath("/config/simple-key-list[key1=forty-two]/outer/inner/INVALID_LEAF"),
			inOpts:           []GetOrCreateNodeOpt{&ReverseShadowPaths{}},
			wantErrSubstring: "no match found in *ytypes.InnerContainerType1",
		},
		{
			inDesc: "success get int32 leaf from the map with key type uint32",
			inParent: &ContainerStruct2{
				StructKeyList: map[uint32]*ListElemStruct2{
					42: {
						Key1: ygot.Uint32(42),
						Outer: &OuterContainerType1{
							Inner: &InnerContainerType1{
								Int32LeafName: ygot.Int32(42),
							},
						},
					},
				},
			},
			inSchema: containerWithUInt32Key,
			inPath:   mustPath("/config/simple-key-list[key1=42]/outer/inner/int32-leaf-field"),
			want:     ygot.Int32(42),
		},
		{
			inDesc:           "fail get enum leaf with incorrect map key",
			inParent:         &ContainerStruct2{},
			inSchema:         containerWithUInt32Key,
			inPath:           mustPath("/config/simple-key-list[key1=INVALID_KEY]/outer/inner/enum-leaf-field"),
			wantErrSubstring: `unable to convert "INVALID_KEY" to uint32`,
		},
		{
			inDesc:   "success get a new InnerContainerType1 node",
			inParent: &ContainerStruct2{},
			inSchema: containerWithUInt32Key,
			inPath:   mustPath("/config/simple-key-list[key1=42]/outer/inner"),
			want:     &InnerContainerType1{},
		},
		{
			inDesc: "success get an existing InnerContainerType1 node",
			inParent: &ContainerStruct2{
				StructKeyList: map[uint32]*ListElemStruct2{
					42: {
						Key1: ygot.Uint32(42),
						Outer: &OuterContainerType1{
							Inner: &InnerContainerType1{
								Int32LeafName: ygot.Int32(42),
							},
						},
					},
				},
			},
			inSchema: containerWithUInt32Key,
			inPath:   mustPath("/config/simple-key-list[key1=42]/outer/inner"),
			want:     &InnerContainerType1{Int32LeafName: ygot.Int32(42)},
		},
		{
			inDesc:           "fail finding with incorrect enum key",
			inSchema:         containerWithEnumKey,
			inParent:         &ContainerEnumKey{},
			inPath:           mustPath("/config/simple-key-list[key1=42]"),
			wantErrSubstring: "42 is not a valid value for enum field",
		},
		{
			inDesc:   "success finding enum key",
			inSchema: containerWithEnumKey,
			inParent: &ContainerEnumKey{},
			inPath:   mustPath("/config/simple-key-list[key1=E_VALUE_FORTY_TWO]"),
			want:     &ListElemEnumKey{Key1: 42},
		},
		{
			inDesc:   "success finding existing enum key",
			inSchema: containerWithEnumKey,
			inParent: &ContainerEnumKey{
				StructKeyList: map[EnumType]*ListElemEnumKey{
					42: {Key1: 42, Int32LeafName: ygot.Int32(99)},
				},
			},
			inPath: mustPath("/config/simple-key-list[key1=E_VALUE_FORTY_TWO]"),
			want:   &ListElemEnumKey{Key1: 42, Int32LeafName: ygot.Int32(99)},
		},
		{
			inDesc:   "success finding bool key",
			inSchema: containerWithBoolKey,
			inParent: &ContainerBoolKey{},
			inPath:   mustPath("/config/simple-key-list[key1=true]"),
			want:     &ListElemBoolKey{Key1: ygot.Bool(true)},
		},
	}

	for i, tt := range tests {
		t.Run(tt.inDesc, func(t *testing.T) {
			got, _, err := GetOrCreateNode(tt.inSchema, tt.inParent, tt.inPath, tt.inOpts...)
			if diff := errdiff.Substring(err, tt.wantErrSubstring); diff != "" {
				t.Fatalf("#%d: %s\ngot %v\nwant %v", i, tt.inDesc, err, tt.wantErrSubstring)
			}
			if err != nil {
				return
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("%s:\n(-want, +got):\n%s", tt.inDesc, diff)
			}
		})
	}
}

type KeyStruct struct {
	Key1    string   `path:"key1"`
	Key2    int32    `path:"key2"`
	EnumKey EnumType `path:"key3"`
}

type ListElemStruct3 struct {
	Key1    *string              `path:"key1"`
	Key2    *int32               `path:"key2"`
	EnumKey EnumType             `path:"key3"`
	Outer   *OuterContainerType1 `path:"outer"`
}

func (*ListElemStruct3) IsYANGGoStruct() {}
func (l *ListElemStruct3) ΛListKeyMap() (map[string]interface{}, error) {
	return map[string]interface{}{
		"key1": *l.Key1,
		"key2": *l.Key2,
		"key3": l.EnumKey,
	}, nil
}

type ContainerStruct3 struct {
	StructKeyList map[KeyStruct]*ListElemStruct3 `path:"struct-key-list"`
}

func (*ContainerStruct3) IsYANGGoStruct() {}

var containerWithMultiKeyedList *yang.Entry = &yang.Entry{
	Name: "container",
	Kind: yang.DirectoryEntry,
	Dir: map[string]*yang.Entry{
		"struct-key-list": {
			Name:     "struct-key-list",
			Kind:     yang.DirectoryEntry,
			ListAttr: yang.NewDefaultListAttr(),
			Key:      "key1 key2 key3",
			Config:   yang.TSTrue,
			Dir: map[string]*yang.Entry{
				"key1": {
					Name: "key1",
					Kind: yang.LeafEntry,
					Type: &yang.YangType{Kind: yang.Ystring},
				},
				"key2": {
					Name: "key2",
					Kind: yang.LeafEntry,
					Type: &yang.YangType{Kind: yang.Yint32},
				},
				"key3": {
					Name: "key3",
					Kind: yang.LeafEntry,
					Type: &yang.YangType{Kind: yang.Yenum},
				},
				"outer": {
					Name: "outer",
					Kind: yang.DirectoryEntry,
					Dir: map[string]*yang.Entry{
						"config": {
							Name: "config",
							Kind: yang.DirectoryEntry,
							Dir: map[string]*yang.Entry{
								"inner": {
									Name: "inner",
									Kind: yang.DirectoryEntry,
									Dir: map[string]*yang.Entry{
										"int32-leaf-field": {
											Name: "leaf-field",
											Kind: yang.LeafEntry,
											Type: &yang.YangType{Kind: yang.Yint32},
										},
										"config": {
											Name: "config",
											Kind: yang.DirectoryEntry,
											Dir: map[string]*yang.Entry{
												"int32-leaf-field": {
													Name: "int32-leaf-field",
													Kind: yang.LeafEntry,
													Type: &yang.YangType{Kind: yang.Yint32},
												},
											},
										},
										"state": {
											Name: "state",
											Kind: yang.DirectoryEntry,
											Dir: map[string]*yang.Entry{
												"int32-leaf-field": {
													Name: "int32-leaf-field",
													Kind: yang.LeafEntry,
													Type: &yang.YangType{Kind: yang.Yint32},
												},
											},
										},
										"int32-leaf-list": {
											Name:     "int32-leaf-list",
											Kind:     yang.LeafEntry,
											ListAttr: yang.NewDefaultListAttr(),
											Type:     &yang.YangType{Kind: yang.Yint32},
										},
										"string-leaf-field": {
											Name: "leaf-field",
											Kind: yang.LeafEntry,
											Type: &yang.YangType{Kind: yang.Ystring},
										},
										"enum-leaf-field": {
											Name: "leaf-field",
											Kind: yang.LeafEntry,
											Type: &yang.YangType{Kind: yang.Yenum},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	},
}

func TestGetOrCreateNodeStructKeyedList(t *testing.T) {
	tests := []struct {
		inDesc           string
		inParent         interface{}
		inSchema         *yang.Entry
		inPath           *gpb.Path
		want             interface{}
		wantErrSubstring string
	}{
		{
			inDesc:   "success get int32 leaf from a struct keyed list",
			inSchema: containerWithMultiKeyedList,
			inParent: &ContainerStruct3{
				StructKeyList: map[KeyStruct]*ListElemStruct3{
					{"forty-two", 42, 42}: {
						Key1:    ygot.String("forty-two"),
						Key2:    ygot.Int32(42),
						EnumKey: EnumType(42),
						Outer:   &OuterContainerType1{Inner: &InnerContainerType1{Int32LeafName: ygot.Int32(1234)}},
					},
				},
			},
			inPath: mustPath("/struct-key-list[key1=forty-two][key2=42][key3=E_VALUE_FORTY_TWO]/outer/inner/int32-leaf-field"),
			want:   ygot.Int32(1234),
		},
		{
			inDesc:   "success get InnerContainerType1 from a struct keyed list",
			inSchema: containerWithMultiKeyedList,
			inParent: &ContainerStruct3{
				StructKeyList: map[KeyStruct]*ListElemStruct3{
					{"forty-two", 42, EnumType(42)}: {
						Key1:    ygot.String("forty-two"),
						Key2:    ygot.Int32(42),
						EnumKey: EnumType(42),
						Outer:   &OuterContainerType1{Inner: &InnerContainerType1{Int32LeafName: ygot.Int32(1234)}},
					},
				},
			},
			inPath: mustPath("/struct-key-list[key1=forty-two][key2=42][key3=E_VALUE_FORTY_TWO]/outer/inner"),
			want:   &InnerContainerType1{Int32LeafName: ygot.Int32(1234)},
		},
		{
			inDesc:   "success get string leaf from a struct keyed list with a new key",
			inSchema: containerWithMultiKeyedList,
			inParent: &ContainerStruct3{},
			inPath:   mustPath("/struct-key-list[key1=forty-two][key2=42][key3=E_VALUE_FORTY_TWO]/outer/inner/string-leaf-field"),
			want:     ygot.String(""),
		},
		{
			inDesc:           "fail get string leaf from a struct keyed list due to invalid key",
			inSchema:         containerWithMultiKeyedList,
			inParent:         &ContainerStruct3{},
			inPath:           mustPath("/struct-key-list[key1=forty-two][key2=42][key3=INVALID_ENUM]/outer/inner/string-leaf-field"),
			wantErrSubstring: "INVALID_ENUM is not a valid value for enum field",
		},
		{
			inDesc:           "fail get due to partial key for struct keyed list",
			inSchema:         containerWithMultiKeyedList,
			inParent:         &ContainerStruct3{},
			inPath:           mustPath("/struct-key-list[key1=forty-two][key2=42]/outer"),
			wantErrSubstring: `missing "key3" key in map`,
		},
	}

	for i, tt := range tests {
		t.Run(tt.inDesc, func(t *testing.T) {
			got, _, err := GetOrCreateNode(tt.inSchema, tt.inParent, tt.inPath)
			if diff := errdiff.Substring(err, tt.wantErrSubstring); diff != "" {
				t.Fatalf("#%d: %s\ngot %v\nwant %v", i, tt.inDesc, err, tt.wantErrSubstring)
			}
			if err != nil {
				return
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("%s:\n(-want, +got):\n%s", tt.inDesc, diff)
			}
		})
	}
}

func simpleSchema() *yang.Entry {
	simpleSchema := &yang.Entry{
		Name: "list-elem-struct1",
		Kind: yang.DirectoryEntry,
		Dir: map[string]*yang.Entry{
			"key1": {
				Name: "key1",
				Kind: yang.LeafEntry,
				Type: &yang.YangType{Kind: yang.Ystring},
			},
			"outer": {
				Name: "outer",
				Kind: yang.DirectoryEntry,
				Dir: map[string]*yang.Entry{
					"config": {
						Name: "config",
						Kind: yang.DirectoryEntry,
						Dir: map[string]*yang.Entry{
							"inner": {
								Name: "inner",
								Kind: yang.DirectoryEntry,
								Dir: map[string]*yang.Entry{
									"int32-leaf-field": {
										Name: "int32-leaf-field",
										Kind: yang.LeafEntry,
										Type: &yang.YangType{Kind: yang.Yint32},
									},
									"config": {
										Name: "config",
										Kind: yang.DirectoryEntry,
										Dir: map[string]*yang.Entry{
											"int32-leaf-field": {
												Name: "int32-leaf-field",
												Kind: yang.LeafEntry,
												Type: &yang.YangType{Kind: yang.Yint32},
											},
										},
									},
									"state": {
										Name: "state",
										Kind: yang.DirectoryEntry,
										Dir: map[string]*yang.Entry{
											"int32-leaf-field": {
												Name: "int32-leaf-field",
												Kind: yang.LeafEntry,
												Type: &yang.YangType{Kind: yang.Yint32},
											},
										},
									},
									"int32-leaf-list": {
										Name:     "int32-leaf-list",
										Kind:     yang.LeafEntry,
										ListAttr: yang.NewDefaultListAttr(),
										Type:     &yang.YangType{Kind: yang.Yint32},
									},
									"string-leaf-field": {
										Name: "string-leaf-field",
										Kind: yang.LeafEntry,
										Type: &yang.YangType{Kind: yang.Ystring},
									},
									"enum-leaf-field": {
										Name: "enum-leaf-field",
										Kind: yang.LeafEntry,
										Type: &yang.YangType{Kind: yang.Yenum},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	addParents(simpleSchema)
	return simpleSchema
}

func TestGetOrCreateNodeWithSimpleSchema(t *testing.T) {
	tests := []struct {
		inDesc           string
		inSchema         *yang.Entry
		inParent         interface{}
		inPath           *gpb.Path
		wantErrSubstring string
		want             interface{}
	}{
		{
			inDesc:   "success retrieving container with direct descendant schema",
			inSchema: simpleSchema(),
			inParent: &ListElemStruct1{},
			inPath:   mustPath("/outer/inner"),
			want:     &InnerContainerType1{},
		},
		{
			inDesc:   "success retrieving container with indirect descendant schema",
			inSchema: simpleSchema(),
			inParent: &ListElemStruct1{},
			inPath:   mustPath("/outer/config/inner"),
			want:     &InnerContainerType1{},
		},
		{
			inDesc:   "success retrieving int32 leaf with direct descendant schema",
			inSchema: simpleSchema(),
			inParent: &ListElemStruct1{},
			inPath:   mustPath("/outer/inner/int32-leaf-field"),
			want:     ygot.Int32(0),
		},
		{
			inDesc:   "success retrieving int32 leaf with indirect descendant schema",
			inSchema: simpleSchema(),
			inParent: &ListElemStruct1{},
			inPath:   mustPath("/outer/config/inner/int32-leaf-field"),
			want:     ygot.Int32(0),
		},
		{
			inDesc:   "success retrieving enum leaf with direct descendant schema",
			inSchema: simpleSchema(),
			inParent: &ListElemStruct1{},
			inPath:   mustPath("/outer/inner/enum-leaf-field"),
			want:     EnumType(0),
		},
		{
			inDesc:   "success retrieving enum leaf from existing container",
			inSchema: simpleSchema(),
			inParent: &ListElemStruct1{
				Outer: &OuterContainerType1{
					Inner: &InnerContainerType1{
						EnumLeafName: EnumType(42),
					},
				},
			},
			inPath: mustPath("/outer/inner/enum-leaf-field"),
			want:   EnumType(42),
		},
		{
			inDesc:   "success retrieving container from existing container",
			inSchema: simpleSchema(),
			inParent: &ListElemStruct1{
				Outer: &OuterContainerType1{
					Inner: &InnerContainerType1{
						EnumLeafName: EnumType(42),
					},
				},
			},
			inPath: mustPath("/outer/inner"),
			want: &InnerContainerType1{
				EnumLeafName: EnumType(42),
			},
		},
	}
	for i, tt := range tests {
		t.Run(tt.inDesc, func(t *testing.T) {
			got, _, err := GetOrCreateNode(tt.inSchema, tt.inParent, tt.inPath)
			if diff := errdiff.Substring(err, tt.wantErrSubstring); diff != "" {
				t.Fatalf("#%d: %s\ngot %v\nwant %v", i, tt.inDesc, err, tt.wantErrSubstring)
			}
			if err != nil {
				return
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("%s:\n(-want, +got):\n%s", tt.inDesc, diff)
			}
		})
	}
}

func mustPath(s string) *gpb.Path {
	p, err := ygot.StringToStructuredPath(s)
	if err != nil {
		panic(err)
	}
	return p
}

func treeNodesEqual(got, want []*TreeNode) error {
	if len(got) != len(want) {
		return fmt.Errorf("mismatched lengths of nodes, got: %d, want: %d", len(got), len(want))
	}

	for _, w := range want {
		match := false
		for _, g := range got {
			// Use reflect.DeepEqual on schema comparison to avoid stack overflow (maybe due to circular references).
			if cmp.Equal(g.Data, w.Data) && reflect.DeepEqual(g.Schema, w.Schema) && proto.Equal(g.Path, w.Path) {
				match = true
				break
			}
		}
		if !match {
			paths := []string{}
			for _, g := range got {
				paths = append(paths, fmt.Sprintf("< %s | %#v >", prototext.MarshalOptions{Multiline: false}.Format(g.Path), g))
			}
			return fmt.Errorf("no match for %#v (path: %s) in %v", w, prototext.MarshalOptions{Multiline: false}.Format(w.Path), paths)
		}
	}
	return nil
}

type listEntry struct {
	Key *string `path:"key"`
}

func (l *listEntry) ΛListKeyMap() (map[string]interface{}, error) {
	return map[string]interface{}{"key": *l.Key}, nil
}

func (*listEntry) IsYANGGoStruct() {}

type multiListEntry struct {
	Keyone *uint32 `path:"keyone"`
	Keytwo *uint32 `path:"keytwo"`
}

func (l *multiListEntry) ΛListKeyMap() (map[string]interface{}, error) {
	return map[string]interface{}{"keyone": *l.Keyone, "keytwo": *l.Keytwo}, nil
}

func (*multiListEntry) IsYANGGoStruct() {}

type multiListKey struct {
	Keyone uint32 `path:"keyone"`
	Keytwo uint32 `path:"keytwo"`
}

type listChildContainer struct {
	Value *string `path:"value|config/value" shadow-path:"value|state/value"`
}

type childList struct {
	Key            *string             `path:"key"`
	ChildContainer *listChildContainer `path:"child-container"`
}

type childContainer struct {
	Container *grandchildContainer `path:"grandchild"`
}

type grandchildContainer struct {
	Val *string `path:"val"`
}

type rootStruct struct {
	Leaf      *string                          `path:"leaf"`
	LeafList  []int32                          `path:"int32-leaf-list"`
	Container *childContainer                  `path:"container" shadow-path:"shadow-container"`
	List      map[string]*listEntry            `path:"list"`
	Multilist map[multiListKey]*multiListEntry `path:"multilist"`
	ChildList map[string]*childList            `path:"state/childlist"`
}

func TestGetNode(t *testing.T) {
	rootSchema := &yang.Entry{
		Name: "root",
		Kind: yang.DirectoryEntry,
		Dir:  map[string]*yang.Entry{},
	}

	leafSchema := &yang.Entry{
		Name: "leaf",
		Kind: yang.LeafEntry,
		Type: &yang.YangType{
			Kind: yang.Ystring,
		},
		Parent: rootSchema,
	}
	rootSchema.Dir["leaf"] = leafSchema

	leafListSchema := &yang.Entry{
		Name:     "int32-leaf-list",
		Kind:     yang.LeafEntry,
		ListAttr: yang.NewDefaultListAttr(),
		Type:     &yang.YangType{Kind: yang.Yint32},
	}
	rootSchema.Dir["int32-leaf-list"] = leafListSchema

	childContainerSchema := &yang.Entry{
		Name:   "container",
		Kind:   yang.DirectoryEntry,
		Parent: rootSchema,
		Dir:    map[string]*yang.Entry{},
	}
	rootSchema.Dir["container"] = childContainerSchema
	childShadowContainerSchema := &yang.Entry{
		Name:   "shadow-container",
		Kind:   yang.DirectoryEntry,
		Parent: rootSchema,
		Dir:    map[string]*yang.Entry{},
	}
	rootSchema.Dir["shadow-container"] = childShadowContainerSchema

	grandchildContainerSchema := &yang.Entry{
		Name:   "grandchild",
		Kind:   yang.DirectoryEntry,
		Parent: childContainerSchema,
		Dir:    map[string]*yang.Entry{},
	}
	childContainerSchema.Dir["grandchild"] = grandchildContainerSchema

	valSchema := &yang.Entry{
		Name: "val",
		Kind: yang.LeafEntry,
		Type: &yang.YangType{
			Kind: yang.Ystring,
		},
	}
	grandchildContainerSchema.Dir["val"] = valSchema

	simpleListSchema := &yang.Entry{
		Name:     "list",
		Kind:     yang.DirectoryEntry,
		Parent:   rootSchema,
		Key:      "key",
		ListAttr: &yang.ListAttr{},
		Dir:      map[string]*yang.Entry{},
	}
	rootSchema.Dir["list"] = simpleListSchema

	keyLeafSchema := &yang.Entry{
		Name:   "key",
		Kind:   yang.LeafEntry,
		Parent: simpleListSchema,
	}
	simpleListSchema.Dir["key"] = keyLeafSchema

	multiKeyListSchema := &yang.Entry{
		Name:     "multilist",
		Kind:     yang.DirectoryEntry,
		Parent:   rootSchema,
		Key:      "keyone keytwo",
		ListAttr: &yang.ListAttr{},
		Dir:      map[string]*yang.Entry{},
	}
	rootSchema.Dir["multilist"] = multiKeyListSchema

	keyOneListSchema := &yang.Entry{
		Name:   "keyone",
		Kind:   yang.LeafEntry,
		Type:   &yang.YangType{Kind: yang.Yuint32},
		Parent: multiKeyListSchema,
	}
	multiKeyListSchema.Dir["keyone"] = keyOneListSchema

	keyTwoListSchema := &yang.Entry{
		Name:   "keytwo",
		Kind:   yang.LeafEntry,
		Type:   &yang.YangType{Kind: yang.Yuint32},
		Parent: multiKeyListSchema,
	}
	multiKeyListSchema.Dir["keytwo"] = keyTwoListSchema

	newChildListSchema := func(configStateName string) *yang.Entry {
		configStateEntry := &yang.Entry{
			Name:   configStateName,
			Kind:   yang.DirectoryEntry,
			Parent: rootSchema,
			Dir:    map[string]*yang.Entry{},
		}

		childListSchema := &yang.Entry{
			Name:     "childlist",
			Kind:     yang.DirectoryEntry,
			Parent:   configStateEntry,
			Key:      "key",
			ListAttr: &yang.ListAttr{},
			Dir:      map[string]*yang.Entry{},
		}
		configStateEntry.Dir["childlist"] = childListSchema

		childListKeySchema := &yang.Entry{
			Name:   "key",
			Kind:   yang.DirectoryEntry,
			Parent: childListSchema,
			Type:   &yang.YangType{Kind: yang.Ystring},
		}
		childListSchema.Dir["key"] = childListKeySchema

		childListContainerSchema := &yang.Entry{
			Name:   "child-container",
			Kind:   yang.DirectoryEntry,
			Parent: childListSchema,
			Dir:    map[string]*yang.Entry{},
		}
		childListSchema.Dir["child-container"] = childListContainerSchema

		childListContainerValueSchema := &yang.Entry{
			Name:   "value",
			Kind:   yang.LeafEntry,
			Parent: childListContainerSchema,
			Type:   &yang.YangType{Kind: yang.Ystring},
		}
		childListContainerSchema.Dir["value"] = childListContainerValueSchema

		configSchema := &yang.Entry{
			Name:   "config",
			Kind:   yang.DirectoryEntry,
			Parent: childListSchema,
			Dir:    map[string]*yang.Entry{},
		}
		childListContainerSchema.Dir["config"] = configSchema

		stateSchema := &yang.Entry{
			Name:   "state",
			Kind:   yang.DirectoryEntry,
			Parent: childListSchema,
			Dir:    map[string]*yang.Entry{},
		}
		childListContainerSchema.Dir["state"] = stateSchema

		configValueSchema := &yang.Entry{
			Name:   "value",
			Kind:   yang.LeafEntry,
			Parent: childListContainerSchema,
			Type:   &yang.YangType{Kind: yang.Ystring},
		}
		configSchema.Dir["config"] = configValueSchema

		stateValueSchema := &yang.Entry{
			Name:   "value",
			Kind:   yang.LeafEntry,
			Parent: childListContainerSchema,
			Type:   &yang.YangType{Kind: yang.Ystring},
		}
		stateSchema.Dir["state"] = stateValueSchema

		return configStateEntry
	}
	rootSchema.Dir["state"] = newChildListSchema("state")

	tests := []struct {
		desc             string
		inSchema         *yang.Entry
		inData           interface{}
		inPath           *gpb.Path
		inArgs           []GetNodeOpt
		wantTreeNodes    []*TreeNode
		wantErrSubstring string
	}{{
		desc:     "simple get leaf",
		inSchema: rootSchema,
		inData: &rootStruct{
			Leaf: ygot.String("foo"),
		},
		inPath: mustPath("/leaf"),
		wantTreeNodes: []*TreeNode{{
			Data:   ygot.String("foo"),
			Schema: leafSchema,
			Path:   mustPath("/leaf"),
		}},
	}, {
		desc:     "simple get leaf with reverseShadowPath=true where shadow-path doesn't exist",
		inSchema: rootSchema,
		inData: &rootStruct{
			Leaf: ygot.String("foo"),
		},
		inPath: mustPath("/leaf"),
		inArgs: []GetNodeOpt{&ReverseShadowPaths{}},
		wantTreeNodes: []*TreeNode{{
			Data:   ygot.String("foo"),
			Schema: leafSchema,
			Path:   mustPath("/leaf"),
		}},
	}, {
		desc:     "simple get leaf with no results",
		inSchema: rootSchema,
		inData:   &rootStruct{},
		inPath:   mustPath("/leaf"),
		wantTreeNodes: []*TreeNode{{
			Schema: leafSchema,
			Data:   (*string)(nil),
			Path:   mustPath("/leaf"),
		}},
	}, {
		desc:     "simple get container with no results",
		inSchema: rootSchema,
		inData:   &rootStruct{},
		inPath:   mustPath("/container"),
		wantTreeNodes: []*TreeNode{{
			Data:   (*childContainer)(nil),
			Schema: childContainerSchema,
			Path:   mustPath("/container"),
		}},
	}, {
		desc:     "simple get leaf list",
		inSchema: rootSchema,
		inData: &rootStruct{
			LeafList: []int32{42, 43},
		},
		inPath: mustPath("/int32-leaf-list"),
		wantTreeNodes: []*TreeNode{{
			Data:   []int32{42, 43},
			Schema: leafListSchema,
			Path:   mustPath("/int32-leaf-list"),
		}},
	}, {
		desc:     "simple get container",
		inSchema: rootSchema,
		inData: &rootStruct{
			Container: &childContainer{},
		},
		inPath: mustPath("/container"),
		wantTreeNodes: []*TreeNode{{
			Data:   &childContainer{},
			Schema: childContainerSchema,
			Path:   mustPath("/container"),
		}},
	}, {
		desc:     "simple get nested container",
		inSchema: rootSchema,
		inData: &rootStruct{
			Container: &childContainer{
				Container: &grandchildContainer{
					Val: ygot.String("forty-two"),
				},
			},
		},
		inPath: mustPath("/container/grandchild/val"),
		wantTreeNodes: []*TreeNode{{
			Data:   ygot.String("forty-two"),
			Schema: valSchema,
			Path:   mustPath("/container/grandchild/val"),
		}},
	}, {
		desc:     "simple list",
		inSchema: rootSchema,
		inData: &rootStruct{
			List: map[string]*listEntry{
				"one": {Key: ygot.String("one")},
				"two": {Key: ygot.String("two")},
			},
		},
		inPath: mustPath("/list[key=one]"),
		wantTreeNodes: []*TreeNode{{
			Data: &listEntry{
				Key: ygot.String("one"),
			},
			Schema: simpleListSchema,
			Path:   mustPath("/list[key=one]"),
		}},
	}, {
		desc:     "incorrectly spelled key name, * match",
		inSchema: rootSchema,
		inData: &rootStruct{
			List: map[string]*listEntry{
				"one": {Key: ygot.String("one")},
				"two": {Key: ygot.String("two")},
			},
		},
		inPath:           mustPath("/list[keyfalse=*]"),
		wantErrSubstring: "schema key key is not found in gNMI path",
	}, {
		desc:     "incorrectly spelled key name simple, * match (wildcard match)",
		inSchema: rootSchema,
		inData: &rootStruct{
			List: map[string]*listEntry{
				"one": {Key: ygot.String("one")},
				"two": {Key: ygot.String("two")},
			},
		},
		inPath:           mustPath("/list[keyfalse=*]"),
		inArgs:           []GetNodeOpt{&GetHandleWildcards{}},
		wantErrSubstring: "schema key key is not found in gNMI path",
	}, {
		desc:     "simple list, * match",
		inSchema: rootSchema,
		inData: &rootStruct{
			List: map[string]*listEntry{
				"one": {Key: ygot.String("one")},
				"two": {Key: ygot.String("two")},
			},
		},
		inPath: mustPath("/list[key=*]"),
		inArgs: []GetNodeOpt{&GetHandleWildcards{}},
		wantTreeNodes: []*TreeNode{{
			Data: &listEntry{
				Key: ygot.String("one"),
			},
			Schema: simpleListSchema,
			Path:   mustPath("/list[key=one]"),
		}, {
			Data: &listEntry{
				Key: ygot.String("two"),
			},
			Schema: simpleListSchema,
			Path:   mustPath("/list[key=two]"),
		}},
	}, {
		desc:     "simple list, unspecified key, no partial match",
		inSchema: rootSchema,
		inData: &rootStruct{
			List: map[string]*listEntry{
				"one": {Key: ygot.String("one")},
			},
		},
		inPath:           mustPath("/list"),
		wantErrSubstring: "schema key key is not found in gNMI path",
	}, {
		desc:     "simple list, unspecified key, partial match",
		inSchema: rootSchema,
		inData: &rootStruct{
			List: map[string]*listEntry{
				"one": {Key: ygot.String("one")},
			},
		},
		inPath: mustPath("/list"),
		inArgs: []GetNodeOpt{&GetPartialKeyMatch{}},
		wantTreeNodes: []*TreeNode{{
			Data:   &listEntry{Key: ygot.String("one")},
			Schema: simpleListSchema,
			Path:   mustPath("/list[key=one]"),
		}},
	}, {
		desc:     "simple list, all entries",
		inSchema: rootSchema,
		inData: &rootStruct{
			List: map[string]*listEntry{
				"one": {Key: ygot.String("one")},
				"two": {Key: ygot.String("two")},
			},
		},
		inPath: mustPath("/list"),
		inArgs: []GetNodeOpt{&GetPartialKeyMatch{}},
		wantTreeNodes: []*TreeNode{{
			Data:   &listEntry{Key: ygot.String("one")},
			Schema: simpleListSchema,
			Path:   mustPath("/list[key=one]"),
		}, {
			Data:   &listEntry{Key: ygot.String("two")},
			Schema: simpleListSchema,
			Path:   mustPath("/list[key=two]"),
		}},
	}, {
		desc:     "multiple key list",
		inSchema: rootSchema,
		inData: &rootStruct{
			Multilist: map[multiListKey]*multiListEntry{
				{Keyone: 1, Keytwo: 2}: {Keyone: ygot.Uint32(1), Keytwo: ygot.Uint32(2)},
			},
		},
		inPath: mustPath("/multilist[keyone=1][keytwo=2]"),
		wantTreeNodes: []*TreeNode{{
			Data:   &multiListEntry{Keyone: ygot.Uint32(1), Keytwo: ygot.Uint32(2)},
			Schema: multiKeyListSchema,
			Path:   mustPath("/multilist[keyone=1][keytwo=2]"),
		}},
	}, {
		desc:     "multiple key list (handle wildcards)",
		inSchema: rootSchema,
		inData: &rootStruct{
			Multilist: map[multiListKey]*multiListEntry{
				{Keyone: 1, Keytwo: 2}: {Keyone: ygot.Uint32(1), Keytwo: ygot.Uint32(2)},
			},
		},
		inPath: mustPath("/multilist[keyone=1][keytwo=2]"),
		inArgs: []GetNodeOpt{&GetHandleWildcards{}},
		wantTreeNodes: []*TreeNode{{
			Data:   &multiListEntry{Keyone: ygot.Uint32(1), Keytwo: ygot.Uint32(2)},
			Schema: multiKeyListSchema,
			Path:   mustPath("/multilist[keyone=1][keytwo=2]"),
		}},
	}, {
		desc:     "multiple key list, *,* match",
		inSchema: rootSchema,
		inData: &rootStruct{
			Multilist: map[multiListKey]*multiListEntry{
				{Keyone: 1, Keytwo: 2}: {Keyone: ygot.Uint32(1), Keytwo: ygot.Uint32(2)},
				{Keyone: 3, Keytwo: 4}: {Keyone: ygot.Uint32(3), Keytwo: ygot.Uint32(4)},
			},
		},
		inPath: mustPath("/multilist[keyone=*][keytwo=*]"),
		inArgs: []GetNodeOpt{&GetHandleWildcards{}},
		wantTreeNodes: []*TreeNode{{
			Data:   &multiListEntry{Keyone: ygot.Uint32(1), Keytwo: ygot.Uint32(2)},
			Schema: multiKeyListSchema,
			Path:   mustPath("/multilist[keyone=1][keytwo=2]"),
		}, {
			Data:   &multiListEntry{Keyone: ygot.Uint32(3), Keytwo: ygot.Uint32(4)},
			Schema: multiKeyListSchema,
			Path:   mustPath("/multilist[keyone=3][keytwo=4]"),
		}},
	}, {
		desc:     "incorrectly spelled key name *,* match",
		inSchema: rootSchema,
		inData: &rootStruct{
			Multilist: map[multiListKey]*multiListEntry{
				{Keyone: 1, Keytwo: 2}: {Keyone: ygot.Uint32(1), Keytwo: ygot.Uint32(2)},
				{Keyone: 3, Keytwo: 4}: {Keyone: ygot.Uint32(3), Keytwo: ygot.Uint32(4)},
			},
		},
		inPath:           mustPath("/multilist[keythree=*][keytwo=*]"),
		wantErrSubstring: "does not contain a map entry for schema keyone",
	}, {
		desc:     "incorrectly spelled key name *,* match (wildcard match)",
		inSchema: rootSchema,
		inData: &rootStruct{
			Multilist: map[multiListKey]*multiListEntry{
				{Keyone: 1, Keytwo: 2}: {Keyone: ygot.Uint32(1), Keytwo: ygot.Uint32(2)},
				{Keyone: 3, Keytwo: 4}: {Keyone: ygot.Uint32(3), Keytwo: ygot.Uint32(4)},
			},
		},
		inPath:           mustPath("/multilist[keythree=*][keytwo=*]"),
		inArgs:           []GetNodeOpt{&GetHandleWildcards{}},
		wantErrSubstring: "does not contain a map entry for schema keyone",
	}, {
		desc:     "multiple key list, *,2 match",
		inSchema: rootSchema,
		inData: &rootStruct{
			Multilist: map[multiListKey]*multiListEntry{
				{Keyone: 1, Keytwo: 2}: {Keyone: ygot.Uint32(1), Keytwo: ygot.Uint32(2)},
				{Keyone: 3, Keytwo: 4}: {Keyone: ygot.Uint32(3), Keytwo: ygot.Uint32(4)},
			},
		},
		inPath: mustPath("/multilist[keyone=*][keytwo=2]"),
		inArgs: []GetNodeOpt{&GetHandleWildcards{}},
		wantTreeNodes: []*TreeNode{{
			Data:   &multiListEntry{Keyone: ygot.Uint32(1), Keytwo: ygot.Uint32(2)},
			Schema: multiKeyListSchema,
			Path:   mustPath("/multilist[keyone=1][keytwo=2]"),
		}},
	}, {
		desc:     "multiple key list, *,2 match",
		inSchema: rootSchema,
		inData: &rootStruct{
			Multilist: map[multiListKey]*multiListEntry{
				{Keyone: 1, Keytwo: 2}: {Keyone: ygot.Uint32(1), Keytwo: ygot.Uint32(2)},
				{Keyone: 3, Keytwo: 2}: {Keyone: ygot.Uint32(3), Keytwo: ygot.Uint32(2)},
			},
		},
		inPath: mustPath("/multilist[keyone=*][keytwo=2]"),
		inArgs: []GetNodeOpt{&GetHandleWildcards{}},
		wantTreeNodes: []*TreeNode{{
			Data:   &multiListEntry{Keyone: ygot.Uint32(1), Keytwo: ygot.Uint32(2)},
			Schema: multiKeyListSchema,
			Path:   mustPath("/multilist[keyone=1][keytwo=2]"),
		}, {
			Data:   &multiListEntry{Keyone: ygot.Uint32(3), Keytwo: ygot.Uint32(2)},
			Schema: multiKeyListSchema,
			Path:   mustPath("/multilist[keyone=3][keytwo=2]"),
		}},
	}, {
		desc:     "multiple key list with >1 element",
		inSchema: rootSchema,
		inData: &rootStruct{
			Multilist: map[multiListKey]*multiListEntry{
				{Keyone: 1, Keytwo: 2}:     {Keyone: ygot.Uint32(1), Keytwo: ygot.Uint32(2)},
				{Keyone: 10, Keytwo: 20}:   {Keyone: ygot.Uint32(10), Keytwo: ygot.Uint32(20)},
				{Keyone: 100, Keytwo: 200}: {Keyone: ygot.Uint32(100), Keytwo: ygot.Uint32(200)},
			},
		},
		inPath: mustPath("/multilist[keyone=1][keytwo=2]"),
		wantTreeNodes: []*TreeNode{{
			Data:   &multiListEntry{Keyone: ygot.Uint32(1), Keytwo: ygot.Uint32(2)},
			Schema: multiKeyListSchema,
			Path:   mustPath("/multilist[keyone=1][keytwo=2]"),
		}},
	}, {
		desc:     "multiple key list, partial match not allowed",
		inSchema: rootSchema,
		inData: &rootStruct{
			Multilist: map[multiListKey]*multiListEntry{
				{Keyone: 1, Keytwo: 2}: {Keyone: ygot.Uint32(1), Keytwo: ygot.Uint32(2)},
				{Keyone: 1, Keytwo: 3}: {Keyone: ygot.Uint32(3), Keytwo: ygot.Uint32(4)},
			},
		},
		inPath:           mustPath("/multilist[keyone=1]"),
		wantErrSubstring: "does not contain a map entry for schema keytwo",
	}, {
		desc:     "multiple key list, partial match allowed",
		inSchema: rootSchema,
		inData: &rootStruct{
			Multilist: map[multiListKey]*multiListEntry{
				{Keyone: 1, Keytwo: 2}: {Keyone: ygot.Uint32(1), Keytwo: ygot.Uint32(2)},
				{Keyone: 1, Keytwo: 3}: {Keyone: ygot.Uint32(1), Keytwo: ygot.Uint32(3)},
			},
		},
		inPath: mustPath("/multilist[keyone=1]"),
		inArgs: []GetNodeOpt{&GetPartialKeyMatch{}},
		wantTreeNodes: []*TreeNode{{
			Data:   &multiListEntry{Keyone: ygot.Uint32(1), Keytwo: ygot.Uint32(2)},
			Schema: multiKeyListSchema,
			Path:   mustPath("/multilist[keyone=1][keytwo=2]"),
		}, {
			Data:   &multiListEntry{Keyone: ygot.Uint32(1), Keytwo: ygot.Uint32(3)},
			Schema: multiKeyListSchema,
			Path:   mustPath("/multilist[keyone=1][keytwo=3]"),
		}},
	}, {
		desc:     "shadow path that traverses a non-leaf node",
		inSchema: rootSchema,
		inData: &rootStruct{
			Container: &childContainer{
				Container: &grandchildContainer{
					Val: ygot.String("forty-two"),
				},
			},
		},
		inPath:           mustPath("/shadow-container/grandchild/val"),
		wantErrSubstring: "shadow path traverses a non-leaf node, this is not allowed",
	}, {
		desc:     "non-shadow path that traverses a non-leaf node with reverseShadowPath=true",
		inSchema: rootSchema,
		inData: &rootStruct{
			Container: &childContainer{
				Container: &grandchildContainer{
					Val: ygot.String("forty-two"),
				},
			},
		},
		inPath:           mustPath("/container/grandchild/val"),
		inArgs:           []GetNodeOpt{&ReverseShadowPaths{}},
		wantErrSubstring: "shadow path traverses a non-leaf node, this is not allowed",
	}, {
		desc:     "deeper list dual shadow/non-shadow leaf path",
		inSchema: rootSchema,
		inData: &rootStruct{
			ChildList: map[string]*childList{
				"one": {
					Key:            ygot.String("one"),
					ChildContainer: &listChildContainer{Value: ygot.String("1")},
				},
				"two": {
					Key:            ygot.String("two"),
					ChildContainer: &listChildContainer{Value: ygot.String("2")},
				},
			},
		},
		inPath: mustPath("/state/childlist[key=one]/child-container/value"),
		wantTreeNodes: []*TreeNode{{
			Data:   ygot.String("1"),
			Schema: rootSchema.Dir["state"].Dir["childlist"].Dir["child-container"].Dir["value"],
			Path:   mustPath("/state/childlist[key=one]/child-container/value"),
		}},
	}, {
		desc:     "deeper list non-shadow leaf path",
		inSchema: rootSchema,
		inData: &rootStruct{
			ChildList: map[string]*childList{
				"one": {
					Key:            ygot.String("one"),
					ChildContainer: &listChildContainer{Value: ygot.String("1")},
				},
				"two": {
					Key:            ygot.String("two"),
					ChildContainer: &listChildContainer{Value: ygot.String("2")},
				},
			},
		},
		inPath: mustPath("/state/childlist[key=one]/child-container/value"),
		wantTreeNodes: []*TreeNode{{
			Data:   ygot.String("1"),
			Schema: rootSchema.Dir["state"].Dir["childlist"].Dir["child-container"].Dir["value"],
			Path:   mustPath("/state/childlist[key=one]/child-container/value"),
		}},
	}, {
		desc:     "deeper list shadow leaf path",
		inSchema: rootSchema,
		inData: &rootStruct{
			ChildList: map[string]*childList{
				"one": {
					Key:            ygot.String("one"),
					ChildContainer: &listChildContainer{Value: ygot.String("1")},
				},
				"two": {
					Key:            ygot.String("two"),
					ChildContainer: &listChildContainer{Value: ygot.String("2")},
				},
			},
		},
		inPath: mustPath("/state/childlist[key=one]/child-container/state/value"),
		wantTreeNodes: []*TreeNode{{
			Data:   nil,
			Schema: nil,
			Path:   mustPath("/state/childlist[key=one]/child-container/state/value"),
		}},
	}, {
		desc:     "deeper list leaf path not found",
		inSchema: rootSchema,
		inData: &rootStruct{
			ChildList: map[string]*childList{
				"one": {
					Key:            ygot.String("one"),
					ChildContainer: &listChildContainer{Value: ygot.String("1")},
				},
				"two": {
					Key:            ygot.String("two"),
					ChildContainer: &listChildContainer{Value: ygot.String("2")},
				},
			},
		},
		inPath:           mustPath("/state/childlist[key=one]/child-container/valeur"),
		wantErrSubstring: "no match found in *ytypes.listChildContainer",
	}, {
		desc:     "deeper list non-shadow leaf path with reverseShadowPath=true",
		inSchema: rootSchema,
		inData: &rootStruct{
			ChildList: map[string]*childList{
				"one": {
					Key:            ygot.String("one"),
					ChildContainer: &listChildContainer{Value: ygot.String("1")},
				},
				"two": {
					Key:            ygot.String("two"),
					ChildContainer: &listChildContainer{Value: ygot.String("2")},
				},
			},
		},
		inPath: mustPath("/state/childlist[key=one]/child-container/config/value"),
		inArgs: []GetNodeOpt{&ReverseShadowPaths{}},
		wantTreeNodes: []*TreeNode{{
			Data:   nil,
			Schema: nil,
			Path:   mustPath("/state/childlist[key=one]/child-container/config/value"),
		}},
	}, {
		desc:     "deeper list shadow leaf path with reverseShadowPath=true",
		inSchema: rootSchema,
		inData: &rootStruct{
			ChildList: map[string]*childList{
				"one": {
					Key:            ygot.String("one"),
					ChildContainer: &listChildContainer{Value: ygot.String("1")},
				},
				"two": {
					Key:            ygot.String("two"),
					ChildContainer: &listChildContainer{Value: ygot.String("2")},
				},
			},
		},
		inPath: mustPath("/state/childlist[key=one]/child-container/state/value"),
		inArgs: []GetNodeOpt{&ReverseShadowPaths{}},
		wantTreeNodes: []*TreeNode{{
			Data:   ygot.String("1"),
			Schema: rootSchema.Dir["state"].Dir["childlist"].Dir["child-container"].Dir["value"],
			Path:   mustPath("/state/childlist[key=one]/child-container/state/value"),
		}},
	}, {
		desc:     "deeper list leaf path not found with reverseShadowPath=true",
		inSchema: rootSchema,
		inData: &rootStruct{
			ChildList: map[string]*childList{
				"one": {
					Key:            ygot.String("one"),
					ChildContainer: &listChildContainer{Value: ygot.String("1")},
				},
				"two": {
					Key:            ygot.String("two"),
					ChildContainer: &listChildContainer{Value: ygot.String("2")},
				},
			},
		},
		inPath:           mustPath("/state/childlist[key=one]/child-container/valeur"),
		inArgs:           []GetNodeOpt{&ReverseShadowPaths{}},
		wantErrSubstring: "no match found in *ytypes.listChildContainer",
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got, err := GetNode(tt.inSchema, tt.inData, tt.inPath, tt.inArgs...)
			if diff := errdiff.Substring(err, tt.wantErrSubstring); diff != "" {
				t.Fatalf("did not get expected error, %s", diff)
			}

			if err := treeNodesEqual(got, tt.wantTreeNodes); err != nil {
				fmt.Println(got[0].Schema)
				fmt.Println(tt.wantTreeNodes[0].Schema)
				t.Fatalf("did not get expected result, %v", err)
			}
		})
	}
}

// ExampleAnnotation is used to test SetNode on Annotation nodes.
type ExampleAnnotation struct {
	ConfigSource string `json:"cfg-source"`
}

// MarshalJSON marshals the ExampleAnnotation receiver to JSON.
func (e *ExampleAnnotation) MarshalJSON() ([]byte, error) {
	return json.Marshal(*e)
}

// UnmarshalJSON ensures that ExampleAnnotation implements the ygot.Annotation
// interface. It is stubbed out and unimplemented.
func (e *ExampleAnnotation) UnmarshalJSON([]byte) error {
	return fmt.Errorf("unimplemented")
}

func TestSetNode(t *testing.T) {
	tests := []struct {
		inDesc           string
		inSchema         *yang.Entry
		inParent         interface{}
		inPath           *gpb.Path
		inVal            interface{}
		inOpts           []SetNodeOpt
		wantErrSubstring string
		wantLeaf         interface{}
		wantParent       interface{}
	}{
		{
			inDesc:     "success setting string field in top node",
			inSchema:   simpleSchema(),
			inParent:   &ListElemStruct1{},
			inPath:     mustPath("/key1"),
			inVal:      &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "hello"}},
			wantLeaf:   ygot.String("hello"),
			wantParent: &ListElemStruct1{Key1: ygot.String("hello")},
		},
		{
			inDesc:     "success setting string field in top node with reverseShadowPath=true where shadow-path doesn't exist",
			inSchema:   simpleSchema(),
			inParent:   &ListElemStruct1{},
			inPath:     mustPath("/key1"),
			inVal:      &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "hello"}},
			inOpts:     []SetNodeOpt{&ReverseShadowPaths{}},
			wantLeaf:   ygot.String("hello"),
			wantParent: &ListElemStruct1{Key1: ygot.String("hello")},
		},
		{
			inDesc:           "failure setting uint field in top node with int value",
			inSchema:         listElemStruct4Schema,
			inParent:         &ListElemStruct4{},
			inPath:           mustPath("/key1"),
			inVal:            &gpb.TypedValue{Value: &gpb.TypedValue_IntVal{IntVal: 42}},
			wantErrSubstring: "failed to unmarshal",
		},
		{
			inDesc:     "success setting uint field in uint node with positive int value with JSON tolerance is set",
			inSchema:   listElemStruct4Schema,
			inParent:   &ListElemStruct4{},
			inPath:     mustPath("/key1"),
			inVal:      &gpb.TypedValue{Value: &gpb.TypedValue_IntVal{IntVal: 42}},
			inOpts:     []SetNodeOpt{&TolerateJSONInconsistencies{}},
			wantLeaf:   ygot.Uint32(42),
			wantParent: &ListElemStruct4{Key1: ygot.Uint32(42)},
		},
		{
			inDesc:     "success setting uint field in uint node with 0 int value with JSON tolerance is set",
			inSchema:   listElemStruct4Schema,
			inParent:   &ListElemStruct4{},
			inPath:     mustPath("/key1"),
			inVal:      &gpb.TypedValue{Value: &gpb.TypedValue_IntVal{IntVal: 0}},
			inOpts:     []SetNodeOpt{&TolerateJSONInconsistencies{}},
			wantLeaf:   ygot.Uint32(0),
			wantParent: &ListElemStruct4{Key1: ygot.Uint32(0)},
		},
		{
			inDesc:           "failure setting uint field in uint node with negative int value with JSON tolerance is set",
			inSchema:         listElemStruct4Schema,
			inParent:         &ListElemStruct4{},
			inPath:           mustPath("/key1"),
			inVal:            &gpb.TypedValue{Value: &gpb.TypedValue_IntVal{IntVal: -42}},
			inOpts:           []SetNodeOpt{&TolerateJSONInconsistencies{}},
			wantErrSubstring: "failed to unmarshal",
		},
		{
			inDesc:           "fail setting value for node with non-leaf schema",
			inSchema:         simpleSchema(),
			inParent:         &ListElemStruct1{},
			inPath:           mustPath("/outer"),
			inVal:            &gpb.TypedValue{},
			wantErrSubstring: `path ` + (&gpb.Path{Elem: []*gpb.PathElem{{Name: "outer"}}}).String() + ` points to a node with non-leaf schema`,
		},
		{
			inDesc:   "success setting annotation in top node",
			inSchema: simpleSchema(),
			inParent: &ListElemStruct1{},
			inPath:   mustPath("/@annotation"),
			inVal:    &ExampleAnnotation{ConfigSource: "devicedemo"},
			wantLeaf: []ygot.Annotation{&ExampleAnnotation{ConfigSource: "devicedemo"}},
			wantParent: &ListElemStruct1{
				Annotation: []ygot.Annotation{&ExampleAnnotation{ConfigSource: "devicedemo"}},
			},
		},
		{
			inDesc:   "success setting annotation in inner node",
			inSchema: simpleSchema(),
			inParent: &ListElemStruct1{},
			inPath:   mustPath("/outer/inner/@annotation"),
			inVal:    &ExampleAnnotation{ConfigSource: "devicedemo"},
			inOpts:   []SetNodeOpt{&InitMissingElements{}},
			wantLeaf: []ygot.Annotation{&ExampleAnnotation{ConfigSource: "devicedemo"}},
			wantParent: &ListElemStruct1{
				Outer: &OuterContainerType1{
					Inner: &InnerContainerType1{
						Annotation: []ygot.Annotation{&ExampleAnnotation{ConfigSource: "devicedemo"}},
					},
				},
			},
		},
		{
			inDesc:   "success setting int32 field in inner node",
			inSchema: simpleSchema(),
			inParent: &ListElemStruct1{},
			inPath:   mustPath("/outer/inner/int32-leaf-field"),
			inVal:    &gpb.TypedValue{Value: &gpb.TypedValue_IntVal{IntVal: 42}},
			inOpts:   []SetNodeOpt{&InitMissingElements{}},
			wantLeaf: ygot.Int32(42),
			wantParent: &ListElemStruct1{
				Outer: &OuterContainerType1{
					Inner: &InnerContainerType1{
						Int32LeafName: ygot.Int32(42),
					},
				},
			},
		},
		{
			inDesc:   "success setting int32 leaf list field",
			inSchema: simpleSchema(),
			inParent: &ListElemStruct1{},
			inPath:   mustPath("/outer/inner/int32-leaf-list"),
			inVal: &gpb.TypedValue{Value: &gpb.TypedValue_LeaflistVal{
				LeaflistVal: &gpb.ScalarArray{
					Element: []*gpb.TypedValue{
						{Value: &gpb.TypedValue_IntVal{IntVal: 42}},
					}},
			}},
			inOpts:   []SetNodeOpt{&InitMissingElements{}},
			wantLeaf: []int32{42},
			wantParent: &ListElemStruct1{
				Outer: &OuterContainerType1{
					Inner: &InnerContainerType1{
						Int32LeafListName: []int32{42},
					},
				},
			},
		},
		{
			inDesc:   "success setting int32 leaf list field for an existing leaf list",
			inSchema: simpleSchema(),
			inParent: &ListElemStruct1{
				Outer: &OuterContainerType1{
					Inner: &InnerContainerType1{
						Int32LeafListName: []int32{42},
					},
				},
			},
			inPath: mustPath("/outer/inner/int32-leaf-list"),
			inVal: &gpb.TypedValue{Value: &gpb.TypedValue_LeaflistVal{
				LeaflistVal: &gpb.ScalarArray{
					Element: []*gpb.TypedValue{
						{Value: &gpb.TypedValue_IntVal{IntVal: 43}},
					}},
			}},
			inOpts:   []SetNodeOpt{&InitMissingElements{}},
			wantLeaf: []int32{42, 43},
			wantParent: &ListElemStruct1{
				Outer: &OuterContainerType1{
					Inner: &InnerContainerType1{
						Int32LeafListName: []int32{42, 43},
					},
				},
			},
		},
		{
			inDesc:   "success setting annotation in list element",
			inSchema: containerWithStringKey(),
			inParent: &ContainerStruct1{
				StructKeyList: map[string]*ListElemStruct1{
					"forty-two": {
						Key1:  ygot.String("forty-two"),
						Outer: &OuterContainerType1{},
					},
				},
			},
			inPath:   mustPath("/config/simple-key-list[key1=forty-two]/@annotation"),
			inVal:    &ExampleAnnotation{ConfigSource: "devicedemo"},
			wantLeaf: []ygot.Annotation{&ExampleAnnotation{ConfigSource: "devicedemo"}},
			wantParent: &ContainerStruct1{
				StructKeyList: map[string]*ListElemStruct1{
					"forty-two": {
						Key1:       ygot.String("forty-two"),
						Outer:      &OuterContainerType1{},
						Annotation: []ygot.Annotation{&ExampleAnnotation{ConfigSource: "devicedemo"}},
					},
				},
			},
		},
		{
			inDesc:   "failed to set annotation in invalid list element",
			inSchema: containerWithStringKey(),
			inParent: &ContainerStruct1{
				StructKeyList: map[string]*ListElemStruct1{},
			},
			inPath:           mustPath("/config/simple-key-list[key1=forty-two]/@annotation"),
			inVal:            &ExampleAnnotation{ConfigSource: "devicedemo"},
			wantErrSubstring: "unable to find any nodes for the given path",
		},
		{
			inDesc:           "failed to set annotation in uninitialized node without InitMissingElements in SetNodeOpt",
			inSchema:         simpleSchema(),
			inParent:         &ListElemStruct1{},
			inPath:           mustPath("/outer/inner/@annotation"),
			inVal:            &ExampleAnnotation{ConfigSource: "devicedemo"},
			wantErrSubstring: "could not find children",
		},
		{
			inDesc:           "failed to set value on invalid node",
			inSchema:         simpleSchema(),
			inParent:         &ListElemStruct1{},
			inPath:           mustPath("/invalidkey"),
			inVal:            ygot.String("hello"),
			wantErrSubstring: "no match found in *ytypes.ListElemStruct1",
		},
		{
			inDesc:           "failed to set value with invalid type",
			inSchema:         simpleSchema(),
			inParent:         &ListElemStruct1{},
			inPath:           mustPath("/@annotation"),
			inVal:            struct{ field string }{"hello"},
			wantErrSubstring: "failed to update struct field Annotation",
		},
		{
			inDesc:   "success setting already-set dual non-shadow and shadow leaf",
			inSchema: containerWithStringKey(),
			inParent: &ContainerStruct1{
				StructKeyList: map[string]*ListElemStruct1{
					"forty-two": {
						Key1: ygot.String("forty-two"),
						Outer: &OuterContainerType1{
							Inner: &InnerContainerType1{
								Int32LeafName: ygot.Int32(42),
							},
						},
					},
				},
			},
			inPath:   mustPath("/config/simple-key-list[key1=forty-two]/outer/inner/int32-leaf-field"),
			inVal:    &gpb.TypedValue{Value: &gpb.TypedValue_IntVal{IntVal: 43}},
			wantLeaf: ygot.Int32(43),
			wantParent: &ContainerStruct1{
				StructKeyList: map[string]*ListElemStruct1{
					"forty-two": {
						Key1: ygot.String("forty-two"),
						Outer: &OuterContainerType1{
							Inner: &InnerContainerType1{
								Int32LeafName: ygot.Int32(43),
							},
						},
					},
				},
			},
		},
		{
			inDesc:   "success setting already-set non-shadow leaf",
			inSchema: containerWithStringKey(),
			inParent: &ContainerStruct1{
				StructKeyList: map[string]*ListElemStruct1{
					"forty-two": {
						Key1: ygot.String("forty-two"),
						Outer: &OuterContainerType1{
							Inner: &InnerContainerType1{
								Int32LeafName: ygot.Int32(42),
							},
						},
					},
				},
			},
			inPath:   mustPath("/config/simple-key-list[key1=forty-two]/outer/inner/config/int32-leaf-field"),
			inVal:    &gpb.TypedValue{Value: &gpb.TypedValue_IntVal{IntVal: 43}},
			wantLeaf: ygot.Int32(43),
			wantParent: &ContainerStruct1{
				StructKeyList: map[string]*ListElemStruct1{
					"forty-two": {
						Key1: ygot.String("forty-two"),
						Outer: &OuterContainerType1{
							Inner: &InnerContainerType1{
								Int32LeafName: ygot.Int32(43),
							},
						},
					},
				},
			},
		},
		{
			inDesc:   "success ignoring already-set shadow leaf",
			inSchema: containerWithStringKey(),
			inParent: &ContainerStruct1{
				StructKeyList: map[string]*ListElemStruct1{
					"forty-two": {
						Key1: ygot.String("forty-two"),
						Outer: &OuterContainerType1{
							Inner: &InnerContainerType1{
								Int32LeafName: ygot.Int32(42),
							},
						},
					},
				},
			},
			inPath:   mustPath("/config/simple-key-list[key1=forty-two]/outer/inner/state/int32-leaf-field"),
			inVal:    &gpb.TypedValue{Value: &gpb.TypedValue_IntVal{IntVal: 43}},
			wantLeaf: nil,
			wantParent: &ContainerStruct1{
				StructKeyList: map[string]*ListElemStruct1{
					"forty-two": {
						Key1: ygot.String("forty-two"),
						Outer: &OuterContainerType1{
							Inner: &InnerContainerType1{
								Int32LeafName: ygot.Int32(42),
							},
						},
					},
				},
			},
		},
		{
			inDesc:   "success setting non-shadow leaf",
			inSchema: containerWithStringKey(),
			inParent: &ContainerStruct1{
				StructKeyList: map[string]*ListElemStruct1{
					"forty-two": {
						Key1:  ygot.String("forty-two"),
						Outer: &OuterContainerType1{},
					},
				},
			},
			inPath:   mustPath("/config/simple-key-list[key1=forty-two]/outer/inner/string-leaf-field"),
			inOpts:   []SetNodeOpt{&InitMissingElements{}},
			inVal:    &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "hello"}},
			wantLeaf: ygot.String("hello"),
			wantParent: &ContainerStruct1{
				StructKeyList: map[string]*ListElemStruct1{
					"forty-two": {
						Key1: ygot.String("forty-two"),
						Outer: &OuterContainerType1{
							Inner: &InnerContainerType1{
								StringLeafName: ygot.String("hello"),
							},
						},
					},
				},
			},
		},
		{
			inDesc:   "success ignore setting shadow leaf",
			inSchema: containerWithStringKey(),
			inParent: &ContainerStruct1{},
			inPath:   mustPath("/config/simple-key-list[key1=forty-two]/outer/inner/state/string-leaf-field"),
			inOpts:   []SetNodeOpt{&InitMissingElements{}},
			inVal:    &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "hello"}},
			wantLeaf: nil,
			wantParent: &ContainerStruct1{
				StructKeyList: map[string]*ListElemStruct1{
					"forty-two": {
						Key1: ygot.String("forty-two"),
						Outer: &OuterContainerType1{
							Inner: &InnerContainerType1{},
						},
					},
				},
			},
		},
		{
			inDesc:   "success setting already-set shadow leaf when reverseShadowPath=true",
			inSchema: containerWithStringKey(),
			inParent: &ContainerStruct1{
				StructKeyList: map[string]*ListElemStruct1{
					"forty-two": {
						Key1: ygot.String("forty-two"),
						Outer: &OuterContainerType1{
							Inner: &InnerContainerType1{
								Int32LeafName: ygot.Int32(42),
							},
						},
					},
				},
			},
			inPath:   mustPath("/config/simple-key-list[key1=forty-two]/outer/inner/state/int32-leaf-field"),
			inOpts:   []SetNodeOpt{&ReverseShadowPaths{}},
			inVal:    &gpb.TypedValue{Value: &gpb.TypedValue_IntVal{IntVal: 43}},
			wantLeaf: ygot.Int32(43),
			wantParent: &ContainerStruct1{
				StructKeyList: map[string]*ListElemStruct1{
					"forty-two": {
						Key1: ygot.String("forty-two"),
						Outer: &OuterContainerType1{
							Inner: &InnerContainerType1{
								Int32LeafName: ygot.Int32(43),
							},
						},
					},
				},
			},
		},
		{
			inDesc:   "success ignoring non-shadow leaf when reverseShadowPath=true",
			inSchema: containerWithStringKey(),
			inParent: &ContainerStruct1{},
			inPath:   mustPath("/config/simple-key-list[key1=forty-two]/outer/inner/config/int32-leaf-field"),
			inOpts:   []SetNodeOpt{&InitMissingElements{}, &ReverseShadowPaths{}},
			inVal:    &gpb.TypedValue{Value: &gpb.TypedValue_IntVal{IntVal: 42}},
			wantLeaf: nil,
			wantParent: &ContainerStruct1{
				StructKeyList: map[string]*ListElemStruct1{
					"forty-two": {
						Key1: ygot.String("forty-two"),
						Outer: &OuterContainerType1{
							Inner: &InnerContainerType1{},
						},
					},
				},
			},
		},
		{
			inDesc:   "success writing dual shadow/non-shadow leaf when reverseShadowPath=true",
			inSchema: containerWithStringKey(),
			inParent: &ContainerStruct1{},
			inPath:   mustPath("/config/simple-key-list[key1=forty-two]/outer/inner/string-leaf-field"),
			inOpts:   []SetNodeOpt{&InitMissingElements{}, &ReverseShadowPaths{}},
			inVal:    &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "hello"}},
			wantLeaf: ygot.String("hello"),
			wantParent: &ContainerStruct1{
				StructKeyList: map[string]*ListElemStruct1{
					"forty-two": {
						Key1: ygot.String("forty-two"),
						Outer: &OuterContainerType1{
							Inner: &InnerContainerType1{
								StringLeafName: ygot.String("hello"),
							},
						},
					},
				},
			},
		},
		{
			inDesc:   "fail setting leaf that doesn't exist when reverseShadowPath=true",
			inSchema: containerWithStringKey(),
			inParent: &ContainerStruct1{
				StructKeyList: map[string]*ListElemStruct1{
					"forty-two": {
						Key1:  ygot.String("forty-two"),
						Outer: &OuterContainerType1{},
					},
				},
			},
			inOpts:           []SetNodeOpt{&InitMissingElements{}, &ReverseShadowPaths{}},
			inPath:           mustPath("/config/simple-key-list[key1=forty-two]/outer/inner/INVALID-LEAF"),
			inVal:            &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "hello"}},
			wantErrSubstring: "no match found in *ytypes.InnerContainerType1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.inDesc, func(t *testing.T) {
			err := SetNode(tt.inSchema, tt.inParent, tt.inPath, tt.inVal, tt.inOpts...)
			if diff := errdiff.Substring(err, tt.wantErrSubstring); diff != "" {
				t.Fatalf("got %v\nwant %v", err, tt.wantErrSubstring)
			}
			if err != nil {
				return
			}
			if diff := cmp.Diff(tt.wantParent, tt.inParent); diff != "" {
				t.Errorf("(-wantParent, +got):\n%s", diff)
			}

			var getNodeOpts []GetNodeOpt
			if hasSetNodeReverseShadowPaths(tt.inOpts) {
				getNodeOpts = append(getNodeOpts, &ReverseShadowPaths{})
			}
			treeNode, err := GetNode(tt.inSchema, tt.inParent, tt.inPath, getNodeOpts...)
			if err != nil {
				t.Fatalf("unexpected error returned from GetNode: %v", err)
			}
			switch {
			case len(treeNode) == 1:
				// Expected case for most tests.
				break
			case len(treeNode) == 0 && tt.wantLeaf == nil:
				return
			default:
				t.Fatalf("did not get exactly one tree node: %v", treeNode)
			}
			got := treeNode[0].Data
			if diff := cmp.Diff(tt.wantLeaf, got); diff != "" {
				t.Errorf("(-wantLeaf, +got):\n%s", diff)
			}
		})
	}
}

func TestDeleteNode(t *testing.T) {
	tests := []struct {
		name             string
		inSchema         *yang.Entry
		inRoot           interface{}
		inPath           *gpb.Path
		inOpts           []DelNodeOpt
		want             interface{}
		wantErrSubstring string
	}{{
		name:     "deleting a string leaf",
		inSchema: simpleSchema(),
		inRoot:   &ListElemStruct1{Key1: ygot.String("hello")},
		inPath:   mustPath("/key1"),
		want:     &ListElemStruct1{Key1: (*string)(nil)},
	}, {
		name:     "deleting a string leaf with reverseShadowPath=true where shadow-path doesn't exist",
		inSchema: simpleSchema(),
		inRoot:   &ListElemStruct1{Key1: ygot.String("hello")},
		inPath:   mustPath("/key1"),
		inOpts:   []DelNodeOpt{&ReverseShadowPaths{}},
		want:     &ListElemStruct1{Key1: (*string)(nil)},
	}, {
		name:     "deleting a int32 leaf list field",
		inSchema: simpleSchema(),
		inRoot:   &ListElemStruct1{Key1: ygot.String("hello"), Outer: &OuterContainerType1{Inner: &InnerContainerType1{Int32LeafListName: []int32{42, 43, 44}}}},
		inPath:   mustPath("/outer/inner/int32-leaf-list"),
		want:     &ListElemStruct1{Key1: ygot.String("hello"), Outer: &OuterContainerType1{Inner: &InnerContainerType1{Int32LeafListName: nil}}},
	}, {
		name:     "deleting a enum field",
		inSchema: simpleSchema(),
		inRoot:   &ListElemStruct1{Outer: &OuterContainerType1{Inner: &InnerContainerType1{EnumLeafName: EnumType(42)}}},
		inPath:   mustPath("/outer/inner/enum-leaf-field"),
		want:     &ListElemStruct1{Outer: &OuterContainerType1{Inner: &InnerContainerType1{}}},
	}, {
		name:     "deleting a non-leaf",
		inSchema: simpleSchema(),
		inRoot:   &ListElemStruct1{Key1: ygot.String("hello"), Outer: &OuterContainerType1{Inner: &InnerContainerType1{Int32LeafName: ygot.Int32(5)}}},
		inPath:   mustPath("/outer"),
		want:     &ListElemStruct1{Key1: ygot.String("hello")},
	}, {
		name:     "deleting int32 leaf in inner node",
		inSchema: simpleSchema(),
		inRoot:   &ListElemStruct1{Key1: ygot.String("world"), Outer: &OuterContainerType1{Inner: &InnerContainerType1{Int32LeafName: ygot.Int32(5)}}},
		inPath:   mustPath("/outer/inner/int32-leaf-field"),
		want:     &ListElemStruct1{Key1: ygot.String("world"), Outer: &OuterContainerType1{Inner: &InnerContainerType1{}}},
	}, {
		name:     "deleting a non-leaf in inner node",
		inSchema: simpleSchema(),
		inRoot:   &ListElemStruct1{Key1: ygot.String("world"), Outer: &OuterContainerType1{Inner: &InnerContainerType1{Int32LeafName: ygot.Int32(5)}}},
		inPath:   mustPath("/outer/inner"),
		want:     &ListElemStruct1{Key1: ygot.String("world"), Outer: &OuterContainerType1{}},
	}, {
		name:     "deleting an annotation in top node",
		inSchema: simpleSchema(),
		inRoot:   &ListElemStruct1{Annotation: []ygot.Annotation{&ExampleAnnotation{ConfigSource: "devicedemo"}}},
		inPath:   mustPath("/@annotation"),
		want:     &ListElemStruct1{},
	}, {
		name:     "deleting an annotation in inner node",
		inSchema: simpleSchema(),
		inRoot:   &ListElemStruct1{Key1: ygot.String("42"), Annotation: []ygot.Annotation{&ExampleAnnotation{ConfigSource: "devicedemo"}}, Outer: &OuterContainerType1{Inner: &InnerContainerType1{Annotation: []ygot.Annotation{&ExampleAnnotation{ConfigSource: "devicedemo"}}}}},
		inPath:   mustPath("/outer/inner/@annotation"),
		want:     &ListElemStruct1{Key1: ygot.String("42"), Annotation: []ygot.Annotation{&ExampleAnnotation{ConfigSource: "devicedemo"}}, Outer: &OuterContainerType1{Inner: &InnerContainerType1{}}},
	}, {
		name:     "deleting an inner node in list",
		inSchema: containerWithStringKey(),
		inRoot: &ContainerStruct1{
			StructKeyList: map[string]*ListElemStruct1{
				"forty-one": {
					Key1: ygot.String("forty-one"),
				},
				"forty-two": {
					Key1:  ygot.String("forty-two"),
					Outer: &OuterContainerType1{Inner: &InnerContainerType1{Int32LeafName: ygot.Int32(5)}},
				},
			},
		},
		inPath: mustPath("/config/simple-key-list[key1=forty-two]/outer/inner"),
		want: &ContainerStruct1{
			StructKeyList: map[string]*ListElemStruct1{
				"forty-one": {
					Key1: ygot.String("forty-one"),
				},
				"forty-two": {
					Key1:  ygot.String("forty-two"),
					Outer: &OuterContainerType1{},
				},
			},
		},
	}, {
		name:     "deleting a list entry",
		inSchema: containerWithStringKey(),
		inRoot: &ContainerStruct1{
			StructKeyList: map[string]*ListElemStruct1{
				"forty-one": {
					Key1: ygot.String("forty-one"),
				},
				"forty-two": {
					Key1:  ygot.String("forty-two"),
					Outer: &OuterContainerType1{Inner: &InnerContainerType1{Int32LeafName: ygot.Int32(5)}},
				},
			},
		},
		inPath: mustPath("/config/simple-key-list[key1=forty-two]"),
		want: &ContainerStruct1{
			StructKeyList: map[string]*ListElemStruct1{
				"forty-one": {
					Key1: ygot.String("forty-one"),
				},
			},
		},
	}, {
		name:     "deleting an inner node from a multi-keyed list",
		inSchema: containerWithMultiKeyedList,
		inRoot: &ContainerStruct3{
			StructKeyList: map[KeyStruct]*ListElemStruct3{
				{"forty-two", 42, 42}: {
					Key1:    ygot.String("forty-two"),
					Key2:    ygot.Int32(42),
					EnumKey: EnumType(42),
					Outer:   &OuterContainerType1{Inner: &InnerContainerType1{Int32LeafName: ygot.Int32(1234)}},
				},
			},
		},
		inPath: mustPath("/struct-key-list[key1=forty-two][key2=42][key3=E_VALUE_FORTY_TWO]/outer/inner/int32-leaf-field"),
		want: &ContainerStruct3{
			StructKeyList: map[KeyStruct]*ListElemStruct3{
				{"forty-two", 42, 42}: {
					Key1:    ygot.String("forty-two"),
					Key2:    ygot.Int32(42),
					EnumKey: EnumType(42),
					Outer:   &OuterContainerType1{Inner: &InnerContainerType1{Int32LeafName: (*int32)(nil)}},
				},
			},
		},
	}, {
		name:     "deleting a list entry from a multi-keyed list",
		inSchema: containerWithMultiKeyedList,
		inRoot: &ContainerStruct3{
			StructKeyList: map[KeyStruct]*ListElemStruct3{
				{"forty", 40, 40}: {
					Key1:    ygot.String("forty"),
					Key2:    ygot.Int32(40),
					EnumKey: EnumType(40),
					Outer:   &OuterContainerType1{Inner: &InnerContainerType1{Int32LeafName: ygot.Int32(4321)}},
				},
				{"forty-two", 42, 42}: {
					Key1:    ygot.String("forty-two"),
					Key2:    ygot.Int32(42),
					EnumKey: EnumType(42),
					Outer:   &OuterContainerType1{Inner: &InnerContainerType1{Int32LeafName: ygot.Int32(1234)}},
				},
			},
		},
		inPath: mustPath("/struct-key-list[key1=forty-two][key2=42][key3=E_VALUE_FORTY_TWO]"),
		want: &ContainerStruct3{
			StructKeyList: map[KeyStruct]*ListElemStruct3{
				{"forty", 40, 40}: {
					Key1:    ygot.String("forty"),
					Key2:    ygot.Int32(40),
					EnumKey: EnumType(40),
					Outer:   &OuterContainerType1{Inner: &InnerContainerType1{Int32LeafName: ygot.Int32(4321)}},
				},
			},
		},
	}, {
		name:     "deleting a leaf whose parent node is nil",
		inSchema: simpleSchema(),
		inRoot:   &ListElemStruct1{Key1: ygot.String("world"), Outer: &OuterContainerType1{}},
		inPath:   mustPath("/outer/inner/int32-leaf-field"),
		want:     &ListElemStruct1{Key1: ygot.String("world"), Outer: &OuterContainerType1{}},
	}, {
		name:     "deleting a non-leaf whose parent node is nil",
		inSchema: simpleSchema(),
		inRoot:   &ListElemStruct1{Key1: ygot.String("world")},
		inPath:   mustPath("/outer/inner"),
		want:     &ListElemStruct1{Key1: ygot.String("world")},
	}, {
		name:     "deleting an inner node from a list entry whose list is nil",
		inSchema: superContainerSchema,
		inRoot:   &SuperContainer{},
		inPath:   mustPath("/container/config/simple-key-list[key1=forty-two]/outer/inner"),
		want:     &SuperContainer{},
	}, {
		name:             "fail to set value on node whose field doesn't exist in the struct definition",
		inSchema:         containerWithStringKey(),
		inRoot:           &ContainerStruct1{},
		inPath:           mustPath("/invalidkey"),
		wantErrSubstring: "no match found in *ytypes.ContainerStruct1",
	}, {
		name:             "fail to set value on list whose field doesn't exist in the struct definition",
		inSchema:         simpleSchema(),
		inRoot:           &ListElemStruct1{},
		inPath:           mustPath("/invalid-list[key1=whatkey]"),
		wantErrSubstring: "no match found in *ytypes.ListElemStruct1",
	}, {
		name:     "deleting a list entry that doesn't exist",
		inSchema: containerWithStringKey(),
		inRoot: &ContainerStruct1{
			StructKeyList: map[string]*ListElemStruct1{
				"forty-two": {
					Key1:  ygot.String("forty-two"),
					Outer: &OuterContainerType1{Inner: &InnerContainerType1{Int32LeafName: ygot.Int32(5)}},
				},
			},
		},
		inPath: mustPath("/config/simple-key-list[key1=forty-one]"),
		want: &ContainerStruct1{
			StructKeyList: map[string]*ListElemStruct1{
				"forty-two": {
					Key1:  ygot.String("forty-two"),
					Outer: &OuterContainerType1{Inner: &InnerContainerType1{Int32LeafName: ygot.Int32(5)}},
				},
			},
		},
	}, {
		name:     "deleting a list entry from a multi-keyed list that doesn't exist",
		inSchema: containerWithMultiKeyedList,
		inRoot: &ContainerStruct3{
			StructKeyList: map[KeyStruct]*ListElemStruct3{
				{"forty-two", 42, 42}: {
					Key1:    ygot.String("forty-two"),
					Key2:    ygot.Int32(42),
					EnumKey: EnumType(42),
					Outer:   &OuterContainerType1{Inner: &InnerContainerType1{Int32LeafName: ygot.Int32(1234)}},
				},
			},
		},
		inPath: mustPath("/struct-key-list[key1=forty-two][key2=41][key3=E_VALUE_FORTY_TWO]"),
		want: &ContainerStruct3{
			StructKeyList: map[KeyStruct]*ListElemStruct3{
				{"forty-two", 42, 42}: {
					Key1:    ygot.String("forty-two"),
					Key2:    ygot.Int32(42),
					EnumKey: EnumType(42),
					Outer:   &OuterContainerType1{Inner: &InnerContainerType1{Int32LeafName: ygot.Int32(1234)}},
				},
			},
		},
	}, {
		name:     "success deleting a list entry with reverseShadowPath=true",
		inSchema: containerWithStringKey(),
		inRoot: &ContainerStruct1{
			StructKeyList: map[string]*ListElemStruct1{
				"forty-one": {
					Key1: ygot.String("forty-one"),
				},
				"forty-two": {
					Key1:  ygot.String("forty-two"),
					Outer: &OuterContainerType1{Inner: &InnerContainerType1{Int32LeafName: ygot.Int32(5)}},
				},
			},
		},
		inPath: mustPath("/config/simple-key-list[key1=forty-two]"),
		inOpts: []DelNodeOpt{&ReverseShadowPaths{}},
		want: &ContainerStruct1{
			StructKeyList: map[string]*ListElemStruct1{
				"forty-one": {
					Key1: ygot.String("forty-one"),
				},
			},
		},
	}, {
		name:     "failure deleting a non-existing list inner node with reverseShadowPath=true",
		inSchema: containerWithStringKey(),
		inRoot: &ContainerStruct1{
			StructKeyList: map[string]*ListElemStruct1{
				"forty-one": {
					Key1: ygot.String("forty-one"),
				},
				"forty-two": {
					Key1:  ygot.String("forty-two"),
					Outer: &OuterContainerType1{Inner: &InnerContainerType1{Int32LeafName: ygot.Int32(5)}},
				},
			},
		},
		inPath:           mustPath("/config/simple-key-list[key1=forty-two]/outer/INVALID"),
		inOpts:           []DelNodeOpt{&ReverseShadowPaths{}},
		wantErrSubstring: "no match found in *ytypes.OuterContainerType1",
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := DeleteNode(tt.inSchema, tt.inRoot, tt.inPath, tt.inOpts...)
			if diff := errdiff.Substring(err, tt.wantErrSubstring); diff != "" {
				t.Fatalf("got error %v\nwant error substr: %s", err, tt.wantErrSubstring)
			}
			if err != nil {
				return
			}

			if diff := cmp.Diff(tt.want, tt.inRoot); diff != "" {
				t.Errorf("TestDeleteNode (-want, +got):\n%s", diff)
			}
		})
	}
}

func TestRetrieveNodeError(t *testing.T) {
	tests := []struct {
		desc             string
		inSchema         *yang.Entry
		inRoot           interface{}
		inPath           *gpb.Path
		inArgs           retrieveNodeArgs
		wantErrSubstring string
	}{{
		desc:             "nil schema",
		inSchema:         nil,
		inRoot:           "test",
		inPath:           &gpb.Path{Elem: []*gpb.PathElem{{Name: "foo"}}},
		wantErrSubstring: "schema is nil",
	}, {
		desc:             "nil root",
		inSchema:         &yang.Entry{Name: "root"},
		inRoot:           nil,
		inPath:           &gpb.Path{Elem: []*gpb.PathElem{{Name: "foo"}}},
		wantErrSubstring: "could not find children",
	}, {
		desc:             "non-container parent",
		inSchema:         &yang.Entry{Name: "root"},
		inRoot:           "fakeroot",
		inPath:           &gpb.Path{Elem: []*gpb.PathElem{{Name: "foo"}}},
		wantErrSubstring: "can not use a parent that is not a container",
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			_, err := retrieveNode(tt.inSchema, tt.inRoot, tt.inPath, nil, tt.inArgs)
			if diff := errdiff.Substring(err, tt.wantErrSubstring); diff != "" {
				t.Fatalf("did not get expected error, %s", diff)
			}
		})
	}
}

func TestRetrieveContainerListError(t *testing.T) {
	rootSchema := &yang.Entry{
		Name: "",
		Kind: yang.DirectoryEntry,
		Dir: map[string]*yang.Entry{
			"ok": {
				Name: "ok",
				Type: &yang.YangType{Kind: yang.Ystring},
			},
		},
	}

	type NoTagRoot struct {
		Ok    *string `path:"ok"`
		NoTag *string
	}

	type BadSchemaRoot struct {
		Ok    *string `path:"ok"`
		Field *string `path:"field"`
	}

	lrSchema := &yang.Entry{
		Name: "",
		Kind: yang.DirectoryEntry,
		Dir: map[string]*yang.Entry{
			"field": {
				Name: "field",
				Type: &yang.YangType{
					Kind: yang.Yleafref,
					Path: "../fish",
				},
			},
		},
	}

	type UnresolvedLeafRef struct {
		Field *string `path:"field"`
	}

	tests := []struct {
		desc             string
		inSchema         *yang.Entry
		inRoot           interface{}
		inPath           *gpb.Path
		inArgs           retrieveNodeArgs
		inTestFunc       func(*yang.Entry, interface{}, *gpb.Path, *gpb.Path, retrieveNodeArgs) ([]*TreeNode, error)
		wantErrSubstring string
	}{{
		desc:             "non-struct ptr root",
		inSchema:         &yang.Entry{},
		inRoot:           "fish",
		inPath:           &gpb.Path{Elem: []*gpb.PathElem{{Name: "ok"}}},
		inTestFunc:       retrieveNodeContainer,
		wantErrSubstring: "want struct ptr root",
	}, {
		desc:             "no annotation on field",
		inSchema:         rootSchema,
		inRoot:           &NoTagRoot{Ok: ygot.String("mackerel")},
		inPath:           &gpb.Path{Elem: []*gpb.PathElem{{Name: "no-tag"}}},
		inTestFunc:       retrieveNodeContainer,
		wantErrSubstring: "failed to get child schema",
	}, {
		desc:             "no schema entry",
		inSchema:         rootSchema,
		inRoot:           &BadSchemaRoot{Ok: ygot.String("haddock")},
		inPath:           &gpb.Path{Elem: []*gpb.PathElem{{Name: "field"}}},
		inTestFunc:       retrieveNodeContainer,
		wantErrSubstring: "could not find schema",
	}, {
		desc:             "error case - leafref unresolved",
		inSchema:         lrSchema,
		inRoot:           &UnresolvedLeafRef{},
		inPath:           &gpb.Path{Elem: []*gpb.PathElem{{Name: "leaf"}}},
		inTestFunc:       retrieveNodeContainer,
		wantErrSubstring: "no match found",
	}, {
		desc:             "no list key",
		inSchema:         &yang.Entry{Name: "foo"},
		inTestFunc:       retrieveNodeList,
		wantErrSubstring: "unkeyed list can't be traversed",
	}, {
		desc:             "nil path",
		inSchema:         &yang.Entry{Name: "foo", Key: "bar"},
		inTestFunc:       retrieveNodeList,
		wantErrSubstring: "path length is 0",
	}, {
		desc:             "root is not a map",
		inSchema:         &yang.Entry{Name: "ant", Key: "bear"},
		inPath:           &gpb.Path{Elem: []*gpb.PathElem{{Name: "cat", Key: map[string]string{"dog": "woof"}}}},
		inRoot:           "menagerie",
		inTestFunc:       retrieveNodeList,
		wantErrSubstring: "root has type string, expect map",
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			_, err := tt.inTestFunc(tt.inSchema, tt.inRoot, tt.inPath, nil, tt.inArgs)
			if diff := errdiff.Substring(err, tt.wantErrSubstring); diff != "" {
				t.Fatalf("did not get expected error, %s", diff)
			}
		})
	}
}
