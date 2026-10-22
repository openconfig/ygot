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
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/openconfig/gnmi/errdiff"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/testutil"
	"github.com/openconfig/ygot/ygot"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

func (s *ListElemStruct1) ΛValidate(...ygot.ValidationOption) error {
	if s.Key1 != nil && *s.Key1 == "invalid" {
		return errors.New("invalid")
	}
	return nil
}

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
								EnumLeafName: EnumType(41),
							},
						},
					},
				},
			},
			inSchema: containerWithStringKey(),
			inPath:   mustPath("/config/simple-key-list[key1=forty-two]/outer/inner/enum-leaf-field"),
			want:     EnumType(41),
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
			inDesc:   "success getting a nil non-shadow value when preferShadowPath=true",
			inParent: &ContainerStruct1{},
			inSchema: containerWithStringKey(),
			inPath:   mustPath("/config/simple-key-list[key1=forty-two]/outer/inner/config/int32-leaf-field"),
			inOpts:   []GetOrCreateNodeOpt{&PreferShadowPath{}},
			want:     nil,
		},
		{
			inDesc:   "success getting an initialized shadow value with preferShadowPath=true",
			inParent: &ContainerStruct1{},
			inSchema: containerWithStringKey(),
			inPath:   mustPath("/config/simple-key-list[key1=forty-two]/outer/inner/state/int32-leaf-field"),
			inOpts:   []GetOrCreateNodeOpt{&PreferShadowPath{}},
			want:     ygot.Int32(0),
		},
		{
			inDesc: "success getting a shadow int32 leaf with an existing key with preferShadowPath=true",
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
			inOpts:   []GetOrCreateNodeOpt{&PreferShadowPath{}},
			want:     ygot.Int32(42),
		},
		{
			inDesc:           "fail getting a shadow value whose container doesn't exist with preferShadowPath=true",
			inParent:         &ContainerStruct1{},
			inSchema:         containerWithStringKey(),
			inPath:           mustPath("/config/simple-key-list[key1=forty-two]/outer/INVALID_CONTAINER/state/int32-leaf-field"),
			inOpts:           []GetOrCreateNodeOpt{&PreferShadowPath{}},
			wantErrSubstring: "no match found in *ytypes.OuterContainerType1",
		},
		{
			inDesc:           "fail getting a value that doesn't exist with preferShadowPath=true",
			inParent:         &ContainerStruct1{},
			inSchema:         containerWithStringKey(),
			inPath:           mustPath("/config/simple-key-list[key1=forty-two]/outer/inner/INVALID_LEAF"),
			inOpts:           []GetOrCreateNodeOpt{&PreferShadowPath{}},
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

func (l KeyStruct) ΛListKeyMap() (map[string]interface{}, error) {
	return map[string]interface{}{
		"key1": l.Key1,
		"key2": l.Key2,
		"key3": l.EnumKey,
	}, nil
}

func (l KeyStruct) IsYANGGoKeyStruct() {}

type ListElemStruct3 struct {
	Key1    *string                     `path:"key1"`
	Key2    *int32                      `path:"key2"`
	EnumKey EnumType                    `path:"key3"`
	Outer   *OuterContainerType1        `path:"outer"`
	KeyList map[uint32]*ListElemStruct4 `path:"key-list-4"`
}

func (*ListElemStruct3) IsYANGGoStruct() {}
func (l *ListElemStruct3) ΛListKeyMap() (map[string]interface{}, error) {
	if l.Key1 == nil {
		return nil, errors.New("nil value for key Key1")
	}
	if l.Key2 == nil {
		return nil, errors.New("nil value for key Key2")
	}
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
				"key-list-4": {
					Name:     "key-list-4",
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

// TestGetNode verifies GetNode traversal across containers, leaves, leaf-lists,
// keyed lists, and shadow paths.
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

	decSchema := decimalListSchema()
	uSchema := unionKeyListSchema()
	muSchema := multiUnionKeyListSchema()
	simpleKeySchemaWithType := func(kind yang.TypeKind) *yang.Entry {
		return &yang.Entry{
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
									Type: &yang.YangType{Kind: kind},
								},
							},
						},
					},
				},
			},
		}
	}
	uint32KeySchema := simpleKeySchemaWithType(yang.Yuint32)
	enumKeySchema := simpleKeySchemaWithType(yang.Yenum)

	tests := []struct {
		desc             string
		inSchema         *yang.Entry
		inData           interface{}
		inPath           *gpb.Path
		inArgs           []GetNodeOpt
		wantTreeNodes    []*TreeNode
		wantErrCode      codes.Code
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
		desc:     "simple get leaf with preferShadowPath=true where shadow-path doesn't exist",
		inSchema: rootSchema,
		inData: &rootStruct{
			Leaf: ygot.String("foo"),
		},
		inPath: mustPath("/leaf"),
		inArgs: []GetNodeOpt{&PreferShadowPath{}},
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
		desc:     "non-shadow path that traverses a non-leaf node with preferShadowPath=true",
		inSchema: rootSchema,
		inData: &rootStruct{
			Container: &childContainer{
				Container: &grandchildContainer{
					Val: ygot.String("forty-two"),
				},
			},
		},
		inPath:           mustPath("/container/grandchild/val"),
		inArgs:           []GetNodeOpt{&PreferShadowPath{}},
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
		desc:     "deeper list non-shadow leaf path with preferShadowPath=true",
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
		inArgs: []GetNodeOpt{&PreferShadowPath{}},
		wantTreeNodes: []*TreeNode{{
			Data:   nil,
			Schema: nil,
			Path:   mustPath("/state/childlist[key=one]/child-container/config/value"),
		}},
	}, {
		desc:     "deeper list shadow leaf path with preferShadowPath=true",
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
		inArgs: []GetNodeOpt{&PreferShadowPath{}},
		wantTreeNodes: []*TreeNode{{
			Data:   ygot.String("1"),
			Schema: rootSchema.Dir["state"].Dir["childlist"].Dir["child-container"].Dir["value"],
			Path:   mustPath("/state/childlist[key=one]/child-container/state/value"),
		}},
	}, {
		desc:     "deeper list leaf path not found with preferShadowPath=true",
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
		inArgs:           []GetNodeOpt{&PreferShadowPath{}},
		wantErrSubstring: "no match found in *ytypes.listChildContainer",
	}, {
		desc:     "empty initialised single-key map returns no nodes",
		inSchema: containerWithStringKey(),
		inData: &ContainerStruct1{
			StructKeyList: map[string]*ListElemStruct1{},
		},
		inPath: mustPath("/config/simple-key-list[key1=forty-two]"),
	}, {
		desc:     "empty initialised single-key map with wildcard returns no nodes",
		inSchema: containerWithStringKey(),
		inData: &ContainerStruct1{
			StructKeyList: map[string]*ListElemStruct1{},
		},
		inPath: mustPath("/config/simple-key-list[key1=*]"),
		inArgs: []GetNodeOpt{&GetHandleWildcards{}},
	}, {
		desc:     "empty initialised multi-key map with partial key returns no nodes",
		inSchema: containerWithMultiKeyedList,
		inData: &ContainerStruct3{
			StructKeyList: map[KeyStruct]*ListElemStruct3{},
		},
		inPath: mustPath("/struct-key-list[key1=forty-two]"),
		inArgs: []GetNodeOpt{&GetPartialKeyMatch{}},
	}, {
		desc:     "invalid uint32 key on populated list fails immediately with InvalidArgument",
		inSchema: uint32KeySchema,
		inData: &ContainerStruct2{
			StructKeyList: map[uint32]*ListElemStruct2{
				42: {Key1: ygot.Uint32(42)},
			},
		},
		inPath:           mustPath("/config/simple-key-list[key1=INVALID_KEY]"),
		wantErrCode:      codes.InvalidArgument,
		wantErrSubstring: `unable to convert "INVALID_KEY" to uint32`,
	}, {
		desc:     "invalid enum key on populated list fails immediately with InvalidArgument",
		inSchema: enumKeySchema,
		inData: &ContainerEnumKey{
			StructKeyList: map[EnumType]*ListElemEnumKey{
				42: {Key1: 42},
			},
		},
		inPath:           mustPath("/config/simple-key-list[key1=INVALID_ENUM]"),
		wantErrCode:      codes.InvalidArgument,
		wantErrSubstring: "no enum matching with INVALID_ENUM",
	}, {
		desc:     "invalid int32 key on populated multi-key list fails immediately with InvalidArgument",
		inSchema: containerWithMultiKeyedList,
		inData: &ContainerStruct3{
			StructKeyList: map[KeyStruct]*ListElemStruct3{
				{Key1: "forty-two", Key2: 42, EnumKey: 42}: {
					Key1:    ygot.String("forty-two"),
					Key2:    ygot.Int32(42),
					EnumKey: 42,
				},
			},
		},
		inPath:           mustPath("/struct-key-list[key1=forty-two][key2=not-an-int][key3=E_VALUE_FORTY_TWO]"),
		wantErrCode:      codes.InvalidArgument,
		wantErrSubstring: `unable to convert "not-an-int" to int32`,
	}, {
		desc:     "decimal64 concrete key lookup succeeds",
		inSchema: decSchema,
		inData: &containerDecimalKey{
			DecimalList: map[float64]*decimalListElem{
				3.14: {Key: ygot.Float64(3.14), Value: ygot.String("pi")},
				2.71: {Key: ygot.Float64(2.71), Value: ygot.String("e")},
			},
		},
		inPath: mustPath("/decimal-list[key1=3.14]/value"),
		wantTreeNodes: []*TreeNode{{
			Path:   mustPath("/decimal-list[key1=3.14]/value"),
			Schema: decSchema.Dir["decimal-list"].Dir["value"],
			Data:   ygot.String("pi"),
		}},
	}, {
		desc:     "decimal64 non-finite NaN key fails immediately with InvalidArgument",
		inSchema: decSchema,
		inData: &containerDecimalKey{
			DecimalList: map[float64]*decimalListElem{
				3.14: {Key: ygot.Float64(3.14), Value: ygot.String("pi")},
			},
		},
		inPath:           mustPath("/decimal-list[key1=NaN]/value"),
		wantErrCode:      codes.InvalidArgument,
		wantErrSubstring: "value must be finite",
	}, {
		desc:     "wrapper-union single-key concrete lookup hits matching entry",
		inSchema: uSchema,
		inData: &containerUnionKey{
			UnionList: map[testutil.TestUnion]*unionKeyListElem{
				&Union1String{String: "alpha"}: {Key: &Union1String{String: "alpha"}, Value: ygot.String("v-alpha")},
				&Union1Int16{Int16: 42}:        {Key: &Union1Int16{Int16: 42}, Value: ygot.String("v-42")},
			},
		},
		inPath: mustPath("/union-list[key1=42]/value"),
		wantTreeNodes: []*TreeNode{{
			Path:   mustPath("/union-list[key1=42]/value"),
			Schema: uSchema.Dir["union-list"].Dir["value"],
			Data:   ygot.String("v-42"),
		}},
	}, {
		desc:     "wrapper-union single-key concrete lookup misses absent entry",
		inSchema: uSchema,
		inData: &containerUnionKey{
			UnionList: map[testutil.TestUnion]*unionKeyListElem{
				&Union1String{String: "alpha"}: {Key: &Union1String{String: "alpha"}, Value: ygot.String("v-alpha")},
			},
		},
		inPath: mustPath("/union-list[key1=missing]"),
	}, {
		desc:     "wrapper-union single-key wildcard lookup returns all entries",
		inSchema: uSchema,
		inData: &containerUnionKey{
			UnionList: map[testutil.TestUnion]*unionKeyListElem{
				&Union1String{String: "alpha"}: {Key: &Union1String{String: "alpha"}, Value: ygot.String("v-alpha")},
				&Union1Int16{Int16: 42}:        {Key: &Union1Int16{Int16: 42}, Value: ygot.String("v-42")},
			},
		},
		inPath: mustPath("/union-list[key1=*]/value"),
		inArgs: []GetNodeOpt{&GetHandleWildcards{}},
		wantTreeNodes: []*TreeNode{{
			Path:   mustPath("/union-list[key1=alpha]/value"),
			Schema: uSchema.Dir["union-list"].Dir["value"],
			Data:   ygot.String("v-alpha"),
		}, {
			Path:   mustPath("/union-list[key1=42]/value"),
			Schema: uSchema.Dir["union-list"].Dir["value"],
			Data:   ygot.String("v-42"),
		}},
	}, {
		desc:     "multi-key struct with union field concrete lookup succeeds",
		inSchema: muSchema,
		inData: &containerMultiUnionKey{
			MultiUnionList: map[multiUnionKeyStruct]*multiUnionKeyListElem{
				{Key1: "k1", Key2: &Union1Int16{Int16: 7}}: {
					Key1:  ygot.String("k1"),
					Key2:  &Union1Int16{Int16: 7},
					Value: ygot.String("seven"),
				},
				{Key1: "k1", Key2: &Union1String{String: "eight"}}: {
					Key1:  ygot.String("k1"),
					Key2:  &Union1String{String: "eight"},
					Value: ygot.String("eight-val"),
				},
			},
		},
		inPath: mustPath("/multi-union-list[key1=k1][key2=7]/value"),
		wantTreeNodes: []*TreeNode{{
			Path:   mustPath("/multi-union-list[key1=k1][key2=7]/value"),
			Schema: muSchema.Dir["multi-union-list"].Dir["value"],
			Data:   ygot.String("seven"),
		}},
	}, {
		desc:     "concrete single-key with wildcard and partial-key options uses concrete lookup",
		inSchema: containerWithStringKey(),
		inData: &ContainerStruct1{
			StructKeyList: map[string]*ListElemStruct1{
				"one": {Key1: ygot.String("one")},
				"two": {Key1: ygot.String("two")},
			},
		},
		inPath: mustPath("/config/simple-key-list[key1=one]"),
		inArgs: []GetNodeOpt{&GetHandleWildcards{}, &GetPartialKeyMatch{}},
		wantTreeNodes: []*TreeNode{{
			Path:   mustPath("/config/simple-key-list[key1=one]"),
			Schema: containerWithStringKey().Dir["config"].Dir["simple-key-list"],
			Data:   &ListElemStruct1{Key1: ygot.String("one")},
		}},
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got, err := GetNode(tt.inSchema, tt.inData, tt.inPath, tt.inArgs...)
			if diff := errdiff.Substring(err, tt.wantErrSubstring); diff != "" {
				t.Fatalf("did not get expected error, %s", diff)
			}
			if tt.wantErrCode != codes.OK {
				if gotCode := status.Code(err); gotCode != tt.wantErrCode {
					t.Errorf("GetNode(%v) status code = %v, want %v (err: %v)", tt.inPath, gotCode, tt.wantErrCode, err)
				}
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

type ConfigStateContainer struct {
	Int32Leaf     *int32  `path:"state/int32-leaf" shadow-path:"config/int32-leaf"`
	Int32LeafList []int32 `path:"state/int32-leaflist" shadow-path:"config/int32-leaflist"`
}

type ConfigStateRoot struct {
	Child *ConfigStateContainer `path:"config-state"`
}

func configStateContainerParentSchema() *yang.Entry {
	sch := &yang.Entry{
		Name: "",
		Kind: yang.DirectoryEntry,
		Dir: map[string]*yang.Entry{
			"config-state": {
				Name: "config-state",
				Kind: yang.DirectoryEntry,
				Dir: map[string]*yang.Entry{
					"config": {
						Name: "config",
						Kind: yang.DirectoryEntry,
						Dir: map[string]*yang.Entry{
							"int32-leaf": {
								Name: "int32-leaf",
								Kind: yang.LeafEntry,
								Type: &yang.YangType{Kind: yang.Yint32},
							},
							"int32-leaflist": {
								Name:     "int32-leaflist",
								Kind:     yang.LeafEntry,
								ListAttr: yang.NewDefaultListAttr(),
								Type:     &yang.YangType{Kind: yang.Yint32},
							},
						},
					},
					"state": {
						Name: "state",
						Kind: yang.DirectoryEntry,
						Dir: map[string]*yang.Entry{
							"int32-leaf": {
								Name: "int32-leaf",
								Kind: yang.LeafEntry,
								Type: &yang.YangType{Kind: yang.Yint32},
							},
							"int32-leaflist": {
								Name:     "int32-leaflist",
								Kind:     yang.LeafEntry,
								ListAttr: yang.NewDefaultListAttr(),
								Type:     &yang.YangType{Kind: yang.Yint32},
							},
						},
					},
				},
			},
		},
	}
	addParents(sch)
	return sch
}

func TestSetNode(t *testing.T) {
	tests := []struct {
		inDesc           string
		inSchema         *yang.Entry
		inParentFn       func() interface{}
		inPath           *gpb.Path
		inVal            interface{}
		inValJSON        interface{}
		inOpts           []SetNodeOpt
		wantErrSubstring string
		wantLeaf         interface{}
		wantParent       interface{}
	}{
		{
			inDesc:     "success setting string field in top node",
			inSchema:   simpleSchema(),
			inParentFn: func() interface{} { return &ListElemStruct1{} },
			inPath:     mustPath("/key1"),
			inVal:      &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "hello"}},
			wantLeaf:   ygot.String("hello"),
			wantParent: &ListElemStruct1{Key1: ygot.String("hello")},
		},
		{
			inDesc:     "success setting string field in top node with preferShadowPath=true where shadow-path doesn't exist",
			inSchema:   simpleSchema(),
			inParentFn: func() interface{} { return &ListElemStruct1{} },
			inPath:     mustPath("/key1"),
			inVal:      &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "hello"}},
			inOpts:     []SetNodeOpt{&PreferShadowPath{}},
			wantLeaf:   ygot.String("hello"),
			wantParent: &ListElemStruct1{Key1: ygot.String("hello")},
		},
		{
			inDesc:           "failure setting uint field in top node with int value",
			inSchema:         listElemStruct4Schema,
			inParentFn:       func() interface{} { return &ListElemStruct4{} },
			inPath:           mustPath("/key1"),
			inVal:            &gpb.TypedValue{Value: &gpb.TypedValue_IntVal{IntVal: 42}},
			wantErrSubstring: "failed to unmarshal",
			wantParent:       &ListElemStruct4{},
		},
		{
			inDesc:           "failure setting uint field in top node with int value with InitMissingElements",
			inSchema:         listElemStruct4Schema,
			inParentFn:       func() interface{} { return &ListElemStruct4{} },
			inPath:           mustPath("/key1"),
			inVal:            &gpb.TypedValue{Value: &gpb.TypedValue_IntVal{IntVal: 42}},
			inOpts:           []SetNodeOpt{&InitMissingElements{}},
			wantErrSubstring: "failed to unmarshal",
			wantParent:       &ListElemStruct4{},
		},
		{
			inDesc:     "success setting uint field in uint node with positive int value with JSON tolerance is set",
			inSchema:   listElemStruct4Schema,
			inParentFn: func() interface{} { return &ListElemStruct4{} },
			inPath:     mustPath("/key1"),
			inVal:      &gpb.TypedValue{Value: &gpb.TypedValue_IntVal{IntVal: 42}},
			inOpts:     []SetNodeOpt{&TolerateJSONInconsistencies{}},
			wantLeaf:   ygot.Uint32(42),
			wantParent: &ListElemStruct4{Key1: ygot.Uint32(42)},
		},
		{
			inDesc:     "success setting uint field in uint node with 0 int value with JSON tolerance is set",
			inSchema:   listElemStruct4Schema,
			inParentFn: func() interface{} { return &ListElemStruct4{} },
			inPath:     mustPath("/key1"),
			inVal:      &gpb.TypedValue{Value: &gpb.TypedValue_IntVal{IntVal: 0}},
			inOpts:     []SetNodeOpt{&TolerateJSONInconsistencies{}},
			wantLeaf:   ygot.Uint32(0),
			wantParent: &ListElemStruct4{Key1: ygot.Uint32(0)},
		},
		{
			inDesc:           "failure setting uint field in uint node with negative int value with JSON tolerance is set",
			inSchema:         listElemStruct4Schema,
			inParentFn:       func() interface{} { return &ListElemStruct4{} },
			inPath:           mustPath("/key1"),
			inVal:            &gpb.TypedValue{Value: &gpb.TypedValue_IntVal{IntVal: -42}},
			inOpts:           []SetNodeOpt{&TolerateJSONInconsistencies{}},
			wantErrSubstring: "failed to unmarshal",
			wantParent:       &ListElemStruct4{},
		},
		{
			inDesc:           "fail setting value for node with non-leaf schema",
			inSchema:         simpleSchema(),
			inParentFn:       func() interface{} { return &ListElemStruct1{} },
			inPath:           mustPath("/outer"),
			inVal:            &gpb.TypedValue{},
			wantErrSubstring: `path ` + (&gpb.Path{Elem: []*gpb.PathElem{{Name: "outer"}}}).String() + ` points to a node with non-leaf schema`,
			wantParent:       &ListElemStruct1{},
		},
		{
			inDesc:     "success setting annotation in top node",
			inSchema:   simpleSchema(),
			inParentFn: func() interface{} { return &ListElemStruct1{} },
			inPath:     mustPath("/@annotation"),
			inVal:      &ExampleAnnotation{ConfigSource: "devicedemo"},
			wantLeaf:   []ygot.Annotation{&ExampleAnnotation{ConfigSource: "devicedemo"}},
			wantParent: &ListElemStruct1{
				Annotation: []ygot.Annotation{&ExampleAnnotation{ConfigSource: "devicedemo"}},
			},
		},
		{
			inDesc:     "success setting annotation in inner node",
			inSchema:   simpleSchema(),
			inParentFn: func() interface{} { return &ListElemStruct1{} },
			inPath:     mustPath("/outer/inner/@annotation"),
			inVal:      &ExampleAnnotation{ConfigSource: "devicedemo"},
			inOpts:     []SetNodeOpt{&InitMissingElements{}},
			wantLeaf:   []ygot.Annotation{&ExampleAnnotation{ConfigSource: "devicedemo"}},
			wantParent: &ListElemStruct1{
				Outer: &OuterContainerType1{
					Inner: &InnerContainerType1{
						Annotation: []ygot.Annotation{&ExampleAnnotation{ConfigSource: "devicedemo"}},
					},
				},
			},
		},
		{
			inDesc:     "success setting int32 field in inner node",
			inSchema:   simpleSchema(),
			inParentFn: func() interface{} { return &ListElemStruct1{} },
			inPath:     mustPath("/outer/inner/int32-leaf-field"),
			inVal:      &gpb.TypedValue{Value: &gpb.TypedValue_IntVal{IntVal: 42}},
			inOpts:     []SetNodeOpt{&InitMissingElements{}},
			wantLeaf:   ygot.Int32(42),
			wantParent: &ListElemStruct1{
				Outer: &OuterContainerType1{
					Inner: &InnerContainerType1{
						Int32LeafName: ygot.Int32(42),
					},
				},
			},
		},
		{
			inDesc:     "success setting leaf-list field",
			inSchema:   configStateContainerParentSchema(),
			inParentFn: func() interface{} { return &ConfigStateRoot{} },
			inPath:     mustPath("/config-state/state/int32-leaflist"),
			inVal: &gpb.TypedValue{Value: &gpb.TypedValue_LeaflistVal{
				LeaflistVal: &gpb.ScalarArray{
					Element: []*gpb.TypedValue{{
						Value: &gpb.TypedValue_IntVal{IntVal: 42},
					}, {
						Value: &gpb.TypedValue_IntVal{IntVal: 43},
					}},
				},
			}},
			inOpts:   []SetNodeOpt{&InitMissingElements{}},
			wantLeaf: []int32{42, 43},
			wantParent: &ConfigStateRoot{
				Child: &ConfigStateContainer{
					Int32LeafList: []int32{42, 43},
				},
			},
		},
		{
			inDesc:     "success setting leaf-list field, with prefer shadow paths (path is shadow path)",
			inSchema:   configStateContainerParentSchema(),
			inParentFn: func() interface{} { return &ConfigStateRoot{} },
			inPath:     mustPath("/config-state/config/int32-leaflist"),
			inVal: &gpb.TypedValue{Value: &gpb.TypedValue_LeaflistVal{
				LeaflistVal: &gpb.ScalarArray{
					Element: []*gpb.TypedValue{{
						Value: &gpb.TypedValue_IntVal{IntVal: 42},
					}, {
						Value: &gpb.TypedValue_IntVal{IntVal: 43},
					}},
				},
			}},
			inOpts:   []SetNodeOpt{&InitMissingElements{}, &PreferShadowPath{}},
			wantLeaf: []int32{42, 43},
			wantParent: &ConfigStateRoot{
				Child: &ConfigStateContainer{
					Int32LeafList: []int32{42, 43},
				},
			},
		},
		{
			inDesc:     "success setting leaf-list field (path is not shadow path)",
			inSchema:   configStateContainerParentSchema(),
			inParentFn: func() interface{} { return &ConfigStateRoot{} },
			inPath:     mustPath("/config-state/state/int32-leaflist"),
			inVal: &gpb.TypedValue{Value: &gpb.TypedValue_LeaflistVal{
				LeaflistVal: &gpb.ScalarArray{
					Element: []*gpb.TypedValue{{
						Value: &gpb.TypedValue_IntVal{IntVal: 42},
					}, {
						Value: &gpb.TypedValue_IntVal{IntVal: 43},
					}},
				},
			}},
			inOpts:   []SetNodeOpt{&InitMissingElements{}},
			wantLeaf: []int32{42, 43},
			wantParent: &ConfigStateRoot{
				Child: &ConfigStateContainer{
					Int32LeafList: []int32{42, 43},
				},
			},
		},
		{
			inDesc:     "success setting leaf-list field, without prefer shadow paths (path is shadow path)",
			inSchema:   configStateContainerParentSchema(),
			inParentFn: func() interface{} { return &ConfigStateRoot{} },
			inPath:     mustPath("/config-state/config/int32-leaflist"),
			inVal: &gpb.TypedValue{Value: &gpb.TypedValue_LeaflistVal{
				LeaflistVal: &gpb.ScalarArray{
					Element: []*gpb.TypedValue{{
						Value: &gpb.TypedValue_IntVal{IntVal: 42},
					}, {
						Value: &gpb.TypedValue_IntVal{IntVal: 43},
					}},
				},
			}},
			inOpts: []SetNodeOpt{&InitMissingElements{}},
			// In this case, we've said "please do not prefer shadow paths" - i.e., just use whatever the path
			// annotation tells you. We should have a no-op here -- since we were given a shadow path
			// that we didn't want to unmarshal.
			wantLeaf: nil,
			// But, hey, we said that we should initialise missing elements, so we do mutate the parent, just
			// not with the leaf-list value.
			wantParent: &ConfigStateRoot{
				Child: &ConfigStateContainer{},
			},
		},
		{
			inDesc:     "success setting int32 leaf list field",
			inSchema:   simpleSchema(),
			inParentFn: func() interface{} { return &ListElemStruct1{} },
			inPath:     mustPath("/outer/inner/int32-leaf-list"),
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
			inParentFn: func() interface{} {
				return &ListElemStruct1{
					Outer: &OuterContainerType1{
						Inner: &InnerContainerType1{
							Int32LeafListName: []int32{42},
						},
					},
				}
			},
			inPath: mustPath("/outer/inner/int32-leaf-list"),
			inVal: &gpb.TypedValue{Value: &gpb.TypedValue_LeaflistVal{
				LeaflistVal: &gpb.ScalarArray{
					Element: []*gpb.TypedValue{
						{Value: &gpb.TypedValue_IntVal{IntVal: 43}},
					}},
			}},
			inOpts:   []SetNodeOpt{&InitMissingElements{}},
			wantLeaf: []int32{43},
			wantParent: &ListElemStruct1{
				Outer: &OuterContainerType1{
					Inner: &InnerContainerType1{
						Int32LeafListName: []int32{43},
					},
				},
			},
		},
		{
			inDesc:   "success setting annotation in list element",
			inSchema: containerWithStringKey(),
			inParentFn: func() interface{} {
				return &ContainerStruct1{
					StructKeyList: map[string]*ListElemStruct1{
						"forty-two": {
							Key1:  ygot.String("forty-two"),
							Outer: &OuterContainerType1{},
						},
					},
				}
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
			inParentFn: func() interface{} {
				return &ContainerStruct1{
					StructKeyList: map[string]*ListElemStruct1{},
				}
			},
			inPath:           mustPath("/config/simple-key-list[key1=forty-two]/@annotation"),
			inVal:            &ExampleAnnotation{ConfigSource: "devicedemo"},
			wantErrSubstring: "unable to find any nodes for the given path",
			wantParent: &ContainerStruct1{
				StructKeyList: map[string]*ListElemStruct1{},
			},
		},
		{
			inDesc:           "failed to set annotation in uninitialized node without InitMissingElements in SetNodeOpt",
			inSchema:         simpleSchema(),
			inParentFn:       func() interface{} { return &ListElemStruct1{} },
			inPath:           mustPath("/outer/inner/@annotation"),
			inVal:            &ExampleAnnotation{ConfigSource: "devicedemo"},
			wantErrSubstring: "could not find children",
			wantParent:       &ListElemStruct1{},
		},
		{
			inDesc:           "failed to set value on invalid node",
			inSchema:         simpleSchema(),
			inParentFn:       func() interface{} { return &ListElemStruct1{} },
			inPath:           mustPath("/invalidkey"),
			inVal:            ygot.String("hello"),
			wantErrSubstring: "no match found in *ytypes.ListElemStruct1",
			wantParent:       &ListElemStruct1{},
		},
		{
			inDesc:     "set on invalid node OK when IgnoreExtraFields set",
			inSchema:   simpleSchema(),
			inParentFn: func() interface{} { return &ListElemStruct1{} },
			inPath:     mustPath("/invalidkey"),
			inVal:      ygot.String("hello"),
			inOpts:     []SetNodeOpt{&IgnoreExtraFields{}},
			wantLeaf:   nil,
			wantParent: &ListElemStruct1{},
		},
		{
			inDesc:           "failed to set value with invalid type",
			inSchema:         simpleSchema(),
			inParentFn:       func() interface{} { return &ListElemStruct1{} },
			inPath:           mustPath("/@annotation"),
			inVal:            struct{ field string }{"hello"},
			wantErrSubstring: "failed to update struct field Annotation",
			wantParent:       &ListElemStruct1{},
		},
		{
			inDesc:   "success setting already-set dual non-shadow and shadow leaf",
			inSchema: containerWithStringKey(),
			inParentFn: func() interface{} {
				return &ContainerStruct1{
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
				}
			},
			inPath:    mustPath("/config/simple-key-list[key1=forty-two]/outer/inner/int32-leaf-field"),
			inVal:     &gpb.TypedValue{Value: &gpb.TypedValue_IntVal{IntVal: 43}},
			inValJSON: &gpb.TypedValue{Value: &gpb.TypedValue_JsonIetfVal{JsonIetfVal: []byte("43")}},
			wantLeaf:  ygot.Int32(43),
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
			inDesc:   "fail setting with Json (non-ietf) value",
			inSchema: containerWithStringKey(),
			inParentFn: func() interface{} {
				return &ContainerStruct1{
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
				}
			},
			inPath:           mustPath("/config/simple-key-list[key1=forty-two]/outer/inner/int32-leaf-field"),
			inValJSON:        &gpb.TypedValue{Value: &gpb.TypedValue_JsonVal{JsonVal: []byte("43")}},
			wantErrSubstring: "json_val format is deprecated, please use json_ietf_val",
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
			inDesc:   "success setting already-set non-shadow leaf",
			inSchema: containerWithStringKey(),
			inParentFn: func() interface{} {
				return &ContainerStruct1{
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
				}
			},
			inPath:    mustPath("/config/simple-key-list[key1=forty-two]/outer/inner/config/int32-leaf-field"),
			inVal:     &gpb.TypedValue{Value: &gpb.TypedValue_IntVal{IntVal: 43}},
			inValJSON: &gpb.TypedValue{Value: &gpb.TypedValue_JsonIetfVal{JsonIetfVal: []byte("43")}},
			wantLeaf:  ygot.Int32(43),
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
			inParentFn: func() interface{} {
				return &ContainerStruct1{
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
				}
			},
			inPath:    mustPath("/config/simple-key-list[key1=forty-two]/outer/inner/state/int32-leaf-field"),
			inVal:     &gpb.TypedValue{Value: &gpb.TypedValue_IntVal{IntVal: 43}},
			inValJSON: &gpb.TypedValue{Value: &gpb.TypedValue_JsonIetfVal{JsonIetfVal: []byte("43")}},
			wantLeaf:  nil,
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
			inParentFn: func() interface{} {
				return &ContainerStruct1{
					StructKeyList: map[string]*ListElemStruct1{
						"forty-two": {
							Key1:  ygot.String("forty-two"),
							Outer: &OuterContainerType1{},
						},
					},
				}
			},
			inPath:    mustPath("/config/simple-key-list[key1=forty-two]/outer/inner/string-leaf-field"),
			inOpts:    []SetNodeOpt{&InitMissingElements{}},
			inVal:     &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "hello"}},
			inValJSON: &gpb.TypedValue{Value: &gpb.TypedValue_JsonIetfVal{JsonIetfVal: []byte(`"hello"`)}},
			wantLeaf:  ygot.String("hello"),
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
			inDesc:     "success ignore setting shadow leaf",
			inSchema:   containerWithStringKey(),
			inParentFn: func() interface{} { return &ContainerStruct1{} },
			inPath:     mustPath("/config/simple-key-list[key1=forty-two]/outer/inner/state/string-leaf-field"),
			inOpts:     []SetNodeOpt{&InitMissingElements{}},
			inVal:      &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "hello"}},
			inValJSON:  &gpb.TypedValue{Value: &gpb.TypedValue_JsonIetfVal{JsonIetfVal: []byte(`"hello"`)}},
			wantLeaf:   nil,
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
			inDesc:   "success setting already-set shadow leaf when preferShadowPath=true",
			inSchema: containerWithStringKey(),
			inParentFn: func() interface{} {
				return &ContainerStruct1{
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
				}
			},
			inPath:    mustPath("/config/simple-key-list[key1=forty-two]/outer/inner/state/int32-leaf-field"),
			inOpts:    []SetNodeOpt{&PreferShadowPath{}},
			inVal:     &gpb.TypedValue{Value: &gpb.TypedValue_IntVal{IntVal: 43}},
			inValJSON: &gpb.TypedValue{Value: &gpb.TypedValue_JsonIetfVal{JsonIetfVal: []byte("43")}},
			wantLeaf:  ygot.Int32(43),
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
			inDesc:     "success ignoring non-shadow leaf when preferShadowPath=true",
			inSchema:   containerWithStringKey(),
			inParentFn: func() interface{} { return &ContainerStruct1{} },
			inPath:     mustPath("/config/simple-key-list[key1=forty-two]/outer/inner/config/int32-leaf-field"),
			inOpts:     []SetNodeOpt{&InitMissingElements{}, &PreferShadowPath{}},
			inVal:      &gpb.TypedValue{Value: &gpb.TypedValue_IntVal{IntVal: 42}},
			inValJSON:  &gpb.TypedValue{Value: &gpb.TypedValue_JsonIetfVal{JsonIetfVal: []byte("42")}},
			wantLeaf:   nil,
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
			inDesc:     "success writing shadow leaf when preferShadowPath=true",
			inSchema:   containerWithStringKey(),
			inParentFn: func() interface{} { return &ContainerStruct1{} },
			inPath:     mustPath("/config/simple-key-list[key1=forty-two]/outer/inner/string-leaf-field"),
			inOpts:     []SetNodeOpt{&InitMissingElements{}, &PreferShadowPath{}},
			inVal:      &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "hello"}},
			inValJSON:  &gpb.TypedValue{Value: &gpb.TypedValue_JsonIetfVal{JsonIetfVal: []byte(`"hello"`)}},
			wantLeaf:   ygot.String("hello"),
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
			inDesc:   "fail setting leaf that doesn't exist when preferShadowPath=true",
			inSchema: containerWithStringKey(),
			inParentFn: func() interface{} {
				return &ContainerStruct1{
					StructKeyList: map[string]*ListElemStruct1{
						"forty-two": {
							Key1:  ygot.String("forty-two"),
							Outer: &OuterContainerType1{},
						},
					},
				}
			},
			inOpts:           []SetNodeOpt{&InitMissingElements{}, &PreferShadowPath{}},
			inPath:           mustPath("/config/simple-key-list[key1=forty-two]/outer/inner/INVALID-LEAF"),
			inVal:            &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "hello"}},
			inValJSON:        &gpb.TypedValue{Value: &gpb.TypedValue_JsonIetfVal{JsonIetfVal: []byte(`"hello"`)}},
			wantErrSubstring: "no match found in *ytypes.InnerContainerType1",
			wantParent: &ContainerStruct1{
				StructKeyList: map[string]*ListElemStruct1{
					"forty-two": {
						Key1: ygot.String("forty-two"),
						Outer: &OuterContainerType1{
							// TODO(wenovus): https://github.com/openconfig/ygot/issues/544
							// This should be deleted.
							Inner: &InnerContainerType1{},
						},
					},
				},
			},
		},
		{
			inDesc:   "success setting struct",
			inSchema: containerWithStringKey(),
			inParentFn: func() interface{} {
				return &ContainerStruct1{
					StructKeyList: map[string]*ListElemStruct1{
						"forty-two": {
							Key1:  ygot.String("forty-two"),
							Outer: &OuterContainerType1{},
						},
					},
				}
			},
			inPath:    mustPath("/config/simple-key-list[key1=forty-two]/outer"),
			inOpts:    []SetNodeOpt{&InitMissingElements{}},
			inValJSON: &gpb.TypedValue{Value: &gpb.TypedValue_JsonIetfVal{JsonIetfVal: []byte(`{ "config": { "inner": { "config": { "int32-leaf-field": 42 } } } }`)}},
			wantLeaf: &OuterContainerType1{
				Inner: &InnerContainerType1{
					Int32LeafName: ygot.Int32(42),
				},
			},
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
			inDesc:   "failure setting JSON struct with unknown field",
			inSchema: containerWithStringKey(),
			inParentFn: func() interface{} {
				return &ContainerStruct1{
					StructKeyList: map[string]*ListElemStruct1{
						"forty-two": {
							Key1:  ygot.String("forty-two"),
							Outer: &OuterContainerType1{},
						},
					},
				}
			},
			inPath:           mustPath("/config/simple-key-list[key1=forty-two]/outer"),
			inOpts:           []SetNodeOpt{&InitMissingElements{}},
			inValJSON:        &gpb.TypedValue{Value: &gpb.TypedValue_JsonIetfVal{JsonIetfVal: []byte(`{ "config": { "inner": { "config": { "int32-leaf-field": 42, "unknown-field": 42 } } } }`)}},
			wantErrSubstring: "JSON contains unexpected field unknown-field",
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
			inDesc:   "OK setting JSON struct with unknown field with IgnoreExtraFields",
			inSchema: containerWithStringKey(),
			inParentFn: func() interface{} {
				return &ContainerStruct1{
					StructKeyList: map[string]*ListElemStruct1{
						"forty-two": {
							Key1:  ygot.String("forty-two"),
							Outer: &OuterContainerType1{},
						},
					},
				}
			},
			inPath:    mustPath("/config/simple-key-list[key1=forty-two]/outer"),
			inOpts:    []SetNodeOpt{&InitMissingElements{}, &IgnoreExtraFields{}},
			inValJSON: &gpb.TypedValue{Value: &gpb.TypedValue_JsonIetfVal{JsonIetfVal: []byte(`{ "config": { "inner": { "config": { "int32-leaf-field": 42, "unknown-field": 42 } } } }`)}},
			wantLeaf: &OuterContainerType1{
				Inner: &InnerContainerType1{
					Int32LeafName: ygot.Int32(42),
				},
			},
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
			inDesc:   "success setting list element",
			inSchema: containerWithStringKey(),
			inParentFn: func() interface{} {
				return &ContainerStruct1{
					StructKeyList: map[string]*ListElemStruct1{
						"forty-two": {
							Key1:  ygot.String("forty-two"),
							Outer: &OuterContainerType1{},
						},
					},
				}
			},
			inPath:    mustPath("/config/simple-key-list[key1=forty-two]"),
			inOpts:    []SetNodeOpt{&InitMissingElements{}},
			inValJSON: &gpb.TypedValue{Value: &gpb.TypedValue_JsonIetfVal{JsonIetfVal: []byte(`{ "outer": { "config": { "inner": { "config": { "int32-leaf-field": 42 } } } } }`)}},
			wantLeaf: &ListElemStruct1{
				Key1: ygot.String("forty-two"),
				Outer: &OuterContainerType1{
					Inner: &InnerContainerType1{
						Int32LeafName: ygot.Int32(42),
					},
				},
			},
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
			inDesc:   "success setting struct with a list field ignoring shadow path",
			inSchema: containerWithStringKey(),
			inParentFn: func() interface{} {
				return &ContainerStruct1{
					StructKeyList: map[string]*ListElemStruct1{
						"forty-two": {
							Key1:  ygot.String("forty-two"),
							Outer: &OuterContainerType1{},
						},
					},
				}
			},
			inPath:    mustPath("/"),
			inOpts:    []SetNodeOpt{&InitMissingElements{}},
			inValJSON: &gpb.TypedValue{Value: &gpb.TypedValue_JsonIetfVal{JsonIetfVal: []byte(`{ "config": { "simple-key-list": [ { "key1": "forty-two", "outer": { "config": { "inner": { "config": { "int32-leaf-field": 42 }, "state": { "int32-leaf-field": 43 } } } } } ] } }`)}},
			wantLeaf: &ContainerStruct1{
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
			inDesc:   "success setting struct with a list field unmarshalling shadow path",
			inSchema: containerWithStringKey(),
			inParentFn: func() interface{} {
				return &ContainerStruct1{
					StructKeyList: map[string]*ListElemStruct1{
						"forty-two": {
							Key1:  ygot.String("forty-two"),
							Outer: &OuterContainerType1{},
						},
					},
				}
			},
			inPath:    mustPath("/"),
			inOpts:    []SetNodeOpt{&InitMissingElements{}, &PreferShadowPath{}},
			inValJSON: &gpb.TypedValue{Value: &gpb.TypedValue_JsonIetfVal{JsonIetfVal: []byte(`{ "config": { "simple-key-list": [ { "key1": "forty-two", "outer": { "config": { "inner": { "config": { "int32-leaf-field": 42 }, "state": { "int32-leaf-field": 43 } } } } } ] } }`)}},
			wantLeaf: &ContainerStruct1{
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
			inDesc:   "bug reproduction: avoid panic with invalid input type",
			inSchema: containerWithStringKey(),
			inParentFn: func() interface{} {
				return &ContainerStruct1{
					StructKeyList: map[string]*ListElemStruct1{
						"forty-two": {
							Key1:  ygot.String("forty-two"),
							Outer: &OuterContainerType1{},
						},
					},
				}
			},
			inPath: mustPath("/config/simple-key-list[key1=forty-two]/outer/inner/string-leaf-field"),
			inOpts: []SetNodeOpt{&InitMissingElements{}},
			inVal:  "hello",
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
			wantErrSubstring: "invalid input data",
		},
	}

	for _, tt := range tests {
		for typeDesc, inVal := range map[string]interface{}{"scalar": tt.inVal, "JSON": tt.inValJSON} {
			if inVal == nil {
				continue
			}
			t.Run(tt.inDesc+" "+typeDesc, func(t *testing.T) {
				parent := tt.inParentFn()
				err := SetNode(tt.inSchema, parent, tt.inPath, inVal, tt.inOpts...)
				if diff := errdiff.Substring(err, tt.wantErrSubstring); diff != "" {
					t.Fatalf("got %v\nwant %v", err, tt.wantErrSubstring)
				}
				if diff := cmp.Diff(tt.wantParent, parent); diff != "" {
					t.Errorf("(-wantParent, +got):\n%s", diff)
				}
				if err != nil {
					return
				}
				if tt.wantLeaf == nil && hasIgnoreExtraFieldsSetNode(tt.inOpts) {
					return
				}

				var getNodeOpts []GetNodeOpt
				if hasSetNodePreferShadowPath(tt.inOpts) {
					getNodeOpts = append(getNodeOpts, &PreferShadowPath{})
				}
				treeNode, err := GetNode(tt.inSchema, parent, tt.inPath, getNodeOpts...)
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
}

// TestDeleteNode verifies DeleteNode behaviour across leaves, containers,
// lists, and shadow paths.
func TestDeleteNode(t *testing.T) {
	keepUnionKey := &Union1String{String: "keep"}
	delUnionKey := &Union1Int16{Int16: 42}

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
		name:     "deleting a string leaf with preferShadowPath=true where shadow-path doesn't exist",
		inSchema: simpleSchema(),
		inRoot:   &ListElemStruct1{Key1: ygot.String("hello")},
		inPath:   mustPath("/key1"),
		inOpts:   []DelNodeOpt{&PreferShadowPath{}},
		want:     &ListElemStruct1{Key1: (*string)(nil)},
	}, {
		name:     "deleting a int32 leaf list field in inner node as the last populated field of a set of containers",
		inSchema: simpleSchema(),
		inRoot:   &ListElemStruct1{Key1: ygot.String("hello"), Outer: &OuterContainerType1{Inner: &InnerContainerType1{Int32LeafListName: []int32{42, 43, 44}}}},
		inPath:   mustPath("/outer/inner/int32-leaf-list"),
		want:     &ListElemStruct1{Key1: ygot.String("hello")},
	}, {
		name:     "deleting a enum field as the last populated field",
		inSchema: simpleSchema(),
		inRoot:   &ListElemStruct1{Outer: &OuterContainerType1{Inner: &InnerContainerType1{EnumLeafName: EnumType(42)}}},
		inPath:   mustPath("/outer/inner/enum-leaf-field"),
		want:     &ListElemStruct1{},
	}, {
		name:     "deleting a non-leaf",
		inSchema: simpleSchema(),
		inRoot:   &ListElemStruct1{Key1: ygot.String("hello"), Outer: &OuterContainerType1{Inner: &InnerContainerType1{Int32LeafListName: []int32{42, 43, 44}, Int32LeafName: ygot.Int32(5)}}},
		inPath:   mustPath("/outer"),
		want:     &ListElemStruct1{Key1: ygot.String("hello")},
	}, {
		name:     "deleting on root struct element",
		inSchema: simpleSchema(),
		inRoot:   &ListElemStruct1{Key1: ygot.String("hello"), Outer: &OuterContainerType1{Inner: &InnerContainerType1{Int32LeafListName: []int32{42, 43, 44}, Int32LeafName: ygot.Int32(5)}}},
		inPath:   mustPath("/"),
		want:     &ListElemStruct1{},
	}, {
		name:     "deleting on nil root struct element",
		inSchema: simpleSchema(),
		inRoot:   (*ListElemStruct1)(nil),
		inPath:   mustPath("/"),
		want:     (*ListElemStruct1)(nil),
	}, {
		name:             "deleting on non-deletable element",
		inRoot:           42,
		inPath:           mustPath("/"),
		wantErrSubstring: "cannot delete on unsettable element",
	}, {
		name:     "deleting on root leaf element",
		inSchema: simpleSchema(),
		inRoot:   &ListElemStruct1{Key1: ygot.String("hello"), Outer: &OuterContainerType1{Inner: &InnerContainerType1{Int32LeafListName: []int32{42, 43, 44}, Int32LeafName: ygot.Int32(5)}}},
		inPath:   mustPath("/"),
		want:     &ListElemStruct1{},
	}, {
		name:     "deleting int32 leaf in inner node",
		inSchema: simpleSchema(),
		inRoot:   &ListElemStruct1{Key1: ygot.String("world"), Outer: &OuterContainerType1{Inner: &InnerContainerType1{Int32LeafName: ygot.Int32(5), Int32LeafListName: []int32{42, 43, 44}}}},
		inPath:   mustPath("/outer/inner/int32-leaf-field"),
		want:     &ListElemStruct1{Key1: ygot.String("world"), Outer: &OuterContainerType1{Inner: &InnerContainerType1{Int32LeafListName: []int32{42, 43, 44}}}},
	}, {
		name:     "deleting a non-leaf in inner node",
		inSchema: simpleSchema(),
		inRoot:   &ListElemStruct1{Key1: ygot.String("world"), Outer: &OuterContainerType1{Inner: &InnerContainerType1{Int32LeafName: ygot.Int32(5), Int32LeafListName: []int32{42, 43, 44}}}},
		inPath:   mustPath("/outer/inner"),
		want:     &ListElemStruct1{Key1: ygot.String("world")},
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
		want:     &ListElemStruct1{Key1: ygot.String("42"), Annotation: []ygot.Annotation{&ExampleAnnotation{ConfigSource: "devicedemo"}}},
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
					Key1: ygot.String("forty-two"),
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
		name:     "deleting a list entry key field",
		inSchema: containerWithStringKey(),
		inRoot: &ContainerStruct1{
			StructKeyList: map[string]*ListElemStruct1{
				"forty-one": {
					Key1:  ygot.String("forty-one"),
					Outer: &OuterContainerType1{Inner: &InnerContainerType1{Int32LeafName: ygot.Int32(5)}},
				},
				"forty-two": {
					Key1:  ygot.String("forty-two"),
					Outer: &OuterContainerType1{Inner: &InnerContainerType1{Int32LeafName: ygot.Int32(5)}},
				},
			},
		},
		inPath: mustPath("/config/simple-key-list[key1=forty-one]/key1"),
		want: &ContainerStruct1{
			StructKeyList: map[string]*ListElemStruct1{
				"forty-one": {
					Outer: &OuterContainerType1{Inner: &InnerContainerType1{Int32LeafName: ygot.Int32(5)}},
				},
				"forty-two": {
					Key1:  ygot.String("forty-two"),
					Outer: &OuterContainerType1{Inner: &InnerContainerType1{Int32LeafName: ygot.Int32(5)}},
				},
			},
		},
	}, {
		name:     "deleting a list entry non-key field when the key field has been deleted -- this should trigger the entire list entry to be deleted since it's now empty",
		inSchema: containerWithStringKey(),
		inRoot: &ContainerStruct1{
			StructKeyList: map[string]*ListElemStruct1{
				"forty-one": {
					Outer: &OuterContainerType1{Inner: &InnerContainerType1{Int32LeafName: ygot.Int32(5)}},
				},
				"forty-two": {
					Key1:  ygot.String("forty-two"),
					Outer: &OuterContainerType1{Inner: &InnerContainerType1{Int32LeafName: ygot.Int32(5)}},
				},
			},
		},
		inPath: mustPath("/config/simple-key-list[key1=forty-one]/outer"),
		want: &ContainerStruct1{
			StructKeyList: map[string]*ListElemStruct1{
				"forty-two": {
					Key1:  ygot.String("forty-two"),
					Outer: &OuterContainerType1{Inner: &InnerContainerType1{Int32LeafName: ygot.Int32(5)}},
				},
			},
		},
	}, {
		name:     "deleting a list entry non-key field",
		inSchema: containerWithStringKey(),
		inRoot: &ContainerStruct1{
			StructKeyList: map[string]*ListElemStruct1{
				"forty-one": {
					Key1:  ygot.String("forty-one"),
					Outer: &OuterContainerType1{Inner: &InnerContainerType1{Int32LeafName: ygot.Int32(5)}},
				},
				"forty-two": {
					Key1:  ygot.String("forty-two"),
					Outer: &OuterContainerType1{Inner: &InnerContainerType1{Int32LeafName: ygot.Int32(5)}},
				},
			},
		},
		inPath: mustPath("/config/simple-key-list[key1=forty-one]/outer"),
		want: &ContainerStruct1{
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
		name:     "deleting a multi-keyed list key field",
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
		inPath: mustPath("/struct-key-list[key1=forty-two][key2=42][key3=E_VALUE_FORTY_TWO]/key2"),
		want: &ContainerStruct3{
			StructKeyList: map[KeyStruct]*ListElemStruct3{
				{"forty", 40, 40}: {
					Key1:    ygot.String("forty"),
					Key2:    ygot.Int32(40),
					EnumKey: EnumType(40),
					Outer:   &OuterContainerType1{Inner: &InnerContainerType1{Int32LeafName: ygot.Int32(4321)}},
				},
				{"forty-two", 42, 42}: {
					Key1:    ygot.String("forty-two"),
					EnumKey: EnumType(42),
					Outer:   &OuterContainerType1{Inner: &InnerContainerType1{Int32LeafName: ygot.Int32(1234)}},
				},
			},
		},
	}, {
		name:     "deleting a multi-keyed list entry non-key field when a key field has been deleted",
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
					EnumKey: EnumType(42),
					Outer:   &OuterContainerType1{Inner: &InnerContainerType1{Int32LeafName: ygot.Int32(1234)}},
				},
			},
		},
		inPath: mustPath("/struct-key-list[key1=forty-two][key2=42][key3=E_VALUE_FORTY_TWO]/outer"),
		want: &ContainerStruct3{
			StructKeyList: map[KeyStruct]*ListElemStruct3{
				{"forty", 40, 40}: {
					Key1:    ygot.String("forty"),
					Key2:    ygot.Int32(40),
					EnumKey: EnumType(40),
					Outer:   &OuterContainerType1{Inner: &InnerContainerType1{Int32LeafName: ygot.Int32(4321)}},
				},
				{"forty-two", 42, 42}: {
					Key1:    ygot.String("forty-two"),
					EnumKey: EnumType(42),
				},
			},
		},
	}, {
		name:     "deleting the last leaf field from a multi-keyed list should trigger deletion of the entire key element",
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
					EnumKey: EnumType(42),
				},
			},
		},
		inPath: mustPath("/struct-key-list[key1=forty-two][key2=42][key3=E_VALUE_FORTY_TWO]/key3"),
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
		name:     "deleting the last non-leaf container field from a multi-keyed list should trigger deletion of the entire key element",
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
					Outer: &OuterContainerType1{Inner: &InnerContainerType1{Int32LeafName: ygot.Int32(4321)}},
				},
			},
		},
		inPath: mustPath("/struct-key-list[key1=forty-two][key2=42][key3=E_VALUE_FORTY_TWO]/outer/inner/int32-leaf-field"),
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
		name:     "deleting the last non-leaf list element field from a multi-keyed list should trigger deletion of the entire key element",
		inSchema: containerWithMultiKeyedList,
		inRoot: &ContainerStruct3{
			StructKeyList: map[KeyStruct]*ListElemStruct3{
				{"forty", 40, 40}: {
					Key1:    ygot.String("forty"),
					Key2:    ygot.Int32(40),
					EnumKey: EnumType(40),
					KeyList: map[uint32]*ListElemStruct4{
						42: {
							Key1: ygot.Uint32(42),
						},
					},
				},
				{"forty-two", 42, 42}: {
					KeyList: map[uint32]*ListElemStruct4{
						42: {
							Key1: ygot.Uint32(42),
						},
					},
				},
			},
		},
		inPath: mustPath("/struct-key-list[key1=forty-two][key2=42][key3=E_VALUE_FORTY_TWO]/key-list-4[key1=42]/key1"),
		want: &ContainerStruct3{
			StructKeyList: map[KeyStruct]*ListElemStruct3{
				{"forty", 40, 40}: {
					Key1:    ygot.String("forty"),
					Key2:    ygot.Int32(40),
					EnumKey: EnumType(40),
					KeyList: map[uint32]*ListElemStruct4{
						42: {
							Key1: ygot.Uint32(42),
						},
					},
				},
			},
		},
	}, {
		name:     "deleting a leaf whose parent node is nil",
		inSchema: simpleSchema(),
		inRoot:   &ListElemStruct1{Key1: ygot.String("world"), Outer: &OuterContainerType1{}},
		inPath:   mustPath("/outer/inner/int32-leaf-field"),
		want:     &ListElemStruct1{Key1: ygot.String("world")},
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
		name:     "success deleting a list entry with preferShadowPath=true",
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
		inOpts: []DelNodeOpt{&PreferShadowPath{}},
		want: &ContainerStruct1{
			StructKeyList: map[string]*ListElemStruct1{
				"forty-one": {
					Key1: ygot.String("forty-one"),
				},
			},
		},
	}, {
		name:     "failure deleting a non-existing list inner node with preferShadowPath=true",
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
		inOpts:           []DelNodeOpt{&PreferShadowPath{}},
		wantErrSubstring: "no match found in *ytypes.OuterContainerType1",
	}, {
		name:     "deleting a union-keyed list entry non-key field when the key field has been deleted",
		inSchema: unionKeyListSchema(),
		inRoot: &containerUnionKey{
			UnionList: map[testutil.TestUnion]*unionKeyListElem{
				keepUnionKey: {Key: &Union1String{String: "keep"}, Value: ygot.String("stay")},
				// NB: Key field inside the struct has already been deleted (nil), leaving a transitory state.
				delUnionKey: {Value: ygot.String("to-delete")},
			},
		},
		inPath: mustPath("/union-list[key1=42]/value"),
		want: &containerUnionKey{
			UnionList: map[testutil.TestUnion]*unionKeyListElem{
				keepUnionKey: {Key: &Union1String{String: "keep"}, Value: ygot.String("stay")},
			},
		},
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

// decimalListElem represents a YANG list element keyed by a decimal64 leaf.
type decimalListElem struct {
	// Key is the decimal64 list key leaf.
	Key *float64 `path:"key1"`
	// Value is an arbitrary string leaf under the list element.
	Value *string `path:"value"`
}

// IsYANGGoStruct implements the ygot.GoStruct interface.
func (*decimalListElem) IsYANGGoStruct() {}

// ΛListKeyMap returns the map of YANG list key names to values for d. It
// returns an error if Key is nil.
func (d *decimalListElem) ΛListKeyMap() (map[string]any, error) {
	if d.Key == nil {
		return nil, errors.New("nil value for key Key")
	}
	return map[string]any{"key1": *d.Key}, nil
}

// containerDecimalKey represents a container holding a decimal64-keyed list.
type containerDecimalKey struct {
	// DecimalList stores list entries keyed by their float64 decimal64 value.
	DecimalList map[float64]*decimalListElem `path:"decimal-list"`
}

// IsYANGGoStruct implements the ygot.GoStruct interface.
func (*containerDecimalKey) IsYANGGoStruct() {}

// decimalListSchema builds the YANG schema entry for containerDecimalKey.
func decimalListSchema() *yang.Entry {
	root := &yang.Entry{
		Name: "container",
		Kind: yang.DirectoryEntry,
		Dir:  map[string]*yang.Entry{},
	}
	list := &yang.Entry{
		Name:     "decimal-list",
		Kind:     yang.DirectoryEntry,
		Parent:   root,
		ListAttr: yang.NewDefaultListAttr(),
		Key:      "key1",
		Config:   yang.TSTrue,
		Dir:      map[string]*yang.Entry{},
	}
	list.Dir["key1"] = &yang.Entry{
		Name:   "key1",
		Kind:   yang.LeafEntry,
		Parent: list,
		Type:   &yang.YangType{Kind: yang.Ydecimal64},
	}
	list.Dir["value"] = &yang.Entry{
		Name:   "value",
		Kind:   yang.LeafEntry,
		Parent: list,
		Type:   &yang.YangType{Kind: yang.Ystring},
	}
	root.Dir["decimal-list"] = list
	return root
}

// unionKeyListElem represents a single-keyed YANG list element whose key is a
// multi-type YANG union.
type unionKeyListElem struct {
	// Key stores the wrapper-union key value for the list element.
	Key testutil.TestUnion `path:"key1"`
	// Value stores a string leaf value on the list element.
	Value *string `path:"value"`
}

// IsYANGGoStruct implements the ygot.GoStruct interface.
func (*unionKeyListElem) IsYANGGoStruct() {}

// ΛListKeyMap returns the map of YANG list key names to values for u. It
// returns an error if Key is nil.
func (u *unionKeyListElem) ΛListKeyMap() (map[string]any, error) {
	if u.Key == nil {
		return nil, errors.New("nil value for key Key")
	}
	return map[string]any{"key1": u.Key}, nil
}

// To_TestUnion converts i into a concrete wrapper type implementing
// testutil.TestUnion. It returns an error if i does not match a union member
// type.
func (*unionKeyListElem) To_TestUnion(i any) (testutil.TestUnion, error) {
	switch v := i.(type) {
	case EnumType:
		return &Union1EnumType{v}, nil
	case int16:
		return &Union1Int16{v}, nil
	case string:
		return &Union1String{v}, nil
	default:
		return nil, fmt.Errorf("cannot convert %v (%T) to TestUnion", i, i)
	}
}

// ΛEnumTypeMap returns the map of schema paths to candidate GoEnum types for
// unionKeyListElem.
func (*unionKeyListElem) ΛEnumTypeMap() map[string][]reflect.Type {
	return map[string][]reflect.Type{
		"/union-list/key1": {reflect.TypeOf(EnumType(0))},
	}
}

// containerUnionKey represents a container holding a single-keyed union list.
type containerUnionKey struct {
	// UnionList maps wrapper-union keys to their corresponding list elements.
	UnionList map[testutil.TestUnion]*unionKeyListElem `path:"union-list"`
}

// IsYANGGoStruct implements the ygot.GoStruct interface.
func (*containerUnionKey) IsYANGGoStruct() {}

// unionKeyListSchema builds the YANG schema entry for containerUnionKey.
func unionKeyListSchema() *yang.Entry {
	root := &yang.Entry{
		Name: "container",
		Kind: yang.DirectoryEntry,
		Dir:  map[string]*yang.Entry{},
	}
	list := &yang.Entry{
		Name:     "union-list",
		Kind:     yang.DirectoryEntry,
		Parent:   root,
		ListAttr: yang.NewDefaultListAttr(),
		Key:      "key1",
		Config:   yang.TSTrue,
		Dir:      map[string]*yang.Entry{},
	}
	list.Dir["key1"] = &yang.Entry{
		Name:   "key1",
		Kind:   yang.LeafEntry,
		Parent: list,
		Type: &yang.YangType{
			Kind: yang.Yunion,
			Type: []*yang.YangType{
				{Name: "enum-type", Kind: yang.Yenum},
				{Name: "int16", Kind: yang.Yint16},
				{Name: "string", Kind: yang.Ystring},
			},
		},
	}
	list.Dir["value"] = &yang.Entry{
		Name:   "value",
		Kind:   yang.LeafEntry,
		Parent: list,
		Type:   &yang.YangType{Kind: yang.Ystring},
	}
	root.Dir["union-list"] = list
	return root
}

// multiUnionKeyStruct represents the composite Go map key for
// multiUnionKeyListElem.
type multiUnionKeyStruct struct {
	// Key1 is the concrete string key component.
	Key1 string `path:"key1"`
	// Key2 is the union interface key component.
	Key2 testutil.TestUnion `path:"key2"`
}

// IsYANGGoKeyStruct implements the ygot.KeyHelperGoStruct marker method.
func (multiUnionKeyStruct) IsYANGGoKeyStruct() {}

// ΛListKeyMap returns the map of YANG list key names to values for m.
func (m multiUnionKeyStruct) ΛListKeyMap() (map[string]any, error) {
	return map[string]any{
		"key1": m.Key1,
		"key2": m.Key2,
	}, nil
}

// multiUnionKeyListElem represents a multi-keyed YANG list element containing
// both a concrete scalar key and a union interface key.
type multiUnionKeyListElem struct {
	// Key1 is the string key leaf of the list element.
	Key1 *string `path:"key1"`
	// Key2 is the union key leaf of the list element.
	Key2 testutil.TestUnion `path:"key2"`
	// Value is an arbitrary string leaf under the list element.
	Value *string `path:"value"`
}

// IsYANGGoStruct implements the ygot.GoStruct interface.
func (*multiUnionKeyListElem) IsYANGGoStruct() {}

// ΛListKeyMap returns the map of YANG list key names to values for m. It
// returns an error if either key field is nil.
func (m *multiUnionKeyListElem) ΛListKeyMap() (map[string]any, error) {
	if m.Key1 == nil || m.Key2 == nil {
		return nil, errors.New("nil key field")
	}
	return map[string]any{
		"key1": *m.Key1,
		"key2": m.Key2,
	}, nil
}

// To_TestUnion converts i into a concrete wrapper type implementing
// testutil.TestUnion. It returns an error if i does not match a union member
// type.
func (*multiUnionKeyListElem) To_TestUnion(i any) (testutil.TestUnion, error) {
	switch v := i.(type) {
	case int16:
		return &Union1Int16{v}, nil
	case string:
		return &Union1String{v}, nil
	default:
		return nil, fmt.Errorf("cannot convert %v (%T) to TestUnion", i, i)
	}
}

// ΛEnumTypeMap returns the map of schema paths to candidate GoEnum types for
// multiUnionKeyListElem.
func (*multiUnionKeyListElem) ΛEnumTypeMap() map[string][]reflect.Type {
	return nil
}

// containerMultiUnionKey represents a container holding a multi-keyed list with
// a union key component.
type containerMultiUnionKey struct {
	// MultiUnionList maps composite key structs to their corresponding list elements.
	MultiUnionList map[multiUnionKeyStruct]*multiUnionKeyListElem `path:"multi-union-list"`
}

// IsYANGGoStruct implements the ygot.GoStruct interface.
func (*containerMultiUnionKey) IsYANGGoStruct() {}

// multiUnionKeyListSchema builds the YANG schema entry for
// containerMultiUnionKey.
func multiUnionKeyListSchema() *yang.Entry {
	root := &yang.Entry{
		Name: "container",
		Kind: yang.DirectoryEntry,
		Dir:  map[string]*yang.Entry{},
	}
	list := &yang.Entry{
		Name:     "multi-union-list",
		Kind:     yang.DirectoryEntry,
		Parent:   root,
		ListAttr: yang.NewDefaultListAttr(),
		Key:      "key1 key2",
		Config:   yang.TSTrue,
		Dir:      map[string]*yang.Entry{},
	}
	list.Dir["key1"] = &yang.Entry{
		Name:   "key1",
		Kind:   yang.LeafEntry,
		Parent: list,
		Type:   &yang.YangType{Kind: yang.Ystring},
	}
	list.Dir["key2"] = &yang.Entry{
		Name:   "key2",
		Kind:   yang.LeafEntry,
		Parent: list,
		Type: &yang.YangType{
			Kind: yang.Yunion,
			Type: []*yang.YangType{
				{Name: "int16", Kind: yang.Yint16},
				{Name: "string", Kind: yang.Ystring},
			},
		},
	}
	list.Dir["value"] = &yang.Entry{
		Name:   "value",
		Kind:   yang.LeafEntry,
		Parent: list,
		Type:   &yang.YangType{Kind: yang.Ystring},
	}
	root.Dir["multi-union-list"] = list
	return root
}

// TestGetOrCreateNodeUnionKeyList verifies GetOrCreateNode insertion into
// populated single-keyed and multi-keyed lists with union key types.
func TestGetOrCreateNodeUnionKeyList(t *testing.T) {
	uSchema := unionKeyListSchema()
	muSchema := multiUnionKeyListSchema()

	listEntriesByKey := func(t *testing.T, parent any) map[string]any {
		t.Helper()
		m := reflect.ValueOf(parent).Elem().Field(0)
		out := map[string]any{}
		for _, k := range m.MapKeys() {
			elem := m.MapIndex(k)
			keys, err := ygot.PathKeyFromStruct(elem)
			if err != nil {
				t.Fatalf("PathKeyFromStruct(%v) failed: %v", elem.Interface(), err)
			}
			out[fmt.Sprint(keys)] = elem.Interface()
		}
		return out
	}

	tests := []struct {
		desc        string
		inSchema    *yang.Entry
		inParent    any
		inPath      *gpb.Path
		want        any
		wantEntries map[string]any
		wantErr     bool
	}{{
		desc:     "inserts into non-empty single-key union list",
		inSchema: uSchema,
		inParent: &containerUnionKey{
			UnionList: map[testutil.TestUnion]*unionKeyListElem{
				&Union1String{String: "existing"}: {
					Key:   &Union1String{String: "existing"},
					Value: ygot.String("v1"),
				},
			},
		},
		inPath: mustPath("/union-list[key1=42]"),
		want:   &unionKeyListElem{Key: &Union1Int16{Int16: 42}},
		wantEntries: map[string]any{
			"map[key1:existing]": &unionKeyListElem{
				Key:   &Union1String{String: "existing"},
				Value: ygot.String("v1"),
			},
			"map[key1:42]": &unionKeyListElem{
				Key: &Union1Int16{Int16: 42},
			},
		},
	}, {
		desc:     "inserts into non-empty multi-key union list",
		inSchema: muSchema,
		inParent: &containerMultiUnionKey{
			MultiUnionList: map[multiUnionKeyStruct]*multiUnionKeyListElem{
				{Key1: "k1", Key2: &Union1Int16{Int16: 7}}: {
					Key1:  ygot.String("k1"),
					Key2:  &Union1Int16{Int16: 7},
					Value: ygot.String("seven"),
				},
			},
		},
		inPath: mustPath("/multi-union-list[key1=k1][key2=hello]"),
		want: &multiUnionKeyListElem{
			Key1: ygot.String("k1"),
			Key2: &Union1String{String: "hello"},
		},
		wantEntries: map[string]any{
			"map[key1:k1 key2:7]": &multiUnionKeyListElem{
				Key1:  ygot.String("k1"),
				Key2:  &Union1Int16{Int16: 7},
				Value: ygot.String("seven"),
			},
			"map[key1:k1 key2:hello]": &multiUnionKeyListElem{
				Key1: ygot.String("k1"),
				Key2: &Union1String{String: "hello"},
			},
		},
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got, _, err := GetOrCreateNode(tt.inSchema, tt.inParent, tt.inPath)
			if (err != nil) != tt.wantErr {
				t.Fatalf("GetOrCreateNode(%v): did not get expected error, got: %v, wantErr? %v", tt.inPath, err, tt.wantErr)
			}
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Fatalf("GetOrCreateNode(%v): did not get expected node, diff(-got,+want):\n%s", tt.inPath, diff)
			}
			// NB: Wrapper-union map keys are struct pointers, so compare entries
			// indexed by their canonical gNMI key map rather than pointer identity.
			gotEntries := listEntriesByKey(t, tt.inParent)
			if diff := cmp.Diff(gotEntries, tt.wantEntries); diff != "" {
				t.Fatalf("GetOrCreateNode(%v): did not get expected list entries, diff(-got,+want):\n%s", tt.inPath, diff)
			}
		})
	}
}

// BenchmarkSetNodeSingleKeyList measures bulk SetNode insertion and lookup
// performance across single-keyed YANG lists of size n.
func BenchmarkSetNodeSingleKeyList(b *testing.B) {
	schema := containerWithStringKey()
	val := &gpb.TypedValue{Value: &gpb.TypedValue_IntVal{IntVal: 42}}

	for _, n := range []int{10, 100, 1000} {
		paths := make([]*gpb.Path, n)
		for i := range n {
			paths[i] = mustPath(fmt.Sprintf("/config/simple-key-list[key1=item-%d]/outer/inner/int32-leaf-field", i))
		}
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				root := &ContainerStruct1{}
				for _, p := range paths {
					if err := SetNode(schema, root, p, val, &InitMissingElements{}); err != nil {
						b.Fatalf("SetNode(%v) failed: %v", p, err)
					}
				}
			}
		})
	}
}

// BenchmarkSetNodeMultiKeyList measures bulk SetNode insertion and lookup
// performance across multi-keyed YANG lists of size n.
func BenchmarkSetNodeMultiKeyList(b *testing.B) {
	addParents(containerWithMultiKeyedList)
	val := &gpb.TypedValue{Value: &gpb.TypedValue_IntVal{IntVal: 42}}

	for _, n := range []int{10, 100, 1000} {
		paths := make([]*gpb.Path, n)
		for i := range n {
			paths[i] = mustPath(fmt.Sprintf("/struct-key-list[key1=item-%d][key2=%d][key3=E_VALUE_FORTY_TWO]/outer/inner/int32-leaf-field", i, i))
		}
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				root := &ContainerStruct3{}
				for _, p := range paths {
					if err := SetNode(containerWithMultiKeyedList, root, p, val, &InitMissingElements{}); err != nil {
						b.Fatalf("SetNode(%v) failed: %v", p, err)
					}
				}
			}
		})
	}
}

// BenchmarkGetNodeSingleKeyList measures point GetNode lookup latency across
// populated single-keyed YANG lists of size n.
func BenchmarkGetNodeSingleKeyList(b *testing.B) {
	schema := containerWithStringKey()
	val := &gpb.TypedValue{Value: &gpb.TypedValue_IntVal{IntVal: 42}}

	for _, n := range []int{10, 100, 1000} {
		root := &ContainerStruct1{}
		paths := make([]*gpb.Path, n)
		for i := range n {
			paths[i] = mustPath(fmt.Sprintf("/config/simple-key-list[key1=item-%d]/outer/inner/int32-leaf-field", i))
			if err := SetNode(schema, root, paths[i], val, &InitMissingElements{}); err != nil {
				b.Fatalf("SetNode(%v) setup failed: %v", paths[i], err)
			}
		}
		target := paths[n-1]
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := GetNode(schema, root, target); err != nil {
					b.Fatalf("GetNode(%v) failed: %v", target, err)
				}
			}
		})
	}
}

// BenchmarkGetNodeMultiKeyList measures point GetNode lookup latency across
// populated multi-keyed YANG lists of size n.
func BenchmarkGetNodeMultiKeyList(b *testing.B) {
	addParents(containerWithMultiKeyedList)
	val := &gpb.TypedValue{Value: &gpb.TypedValue_IntVal{IntVal: 42}}

	for _, n := range []int{10, 100, 1000} {
		root := &ContainerStruct3{}
		paths := make([]*gpb.Path, n)
		for i := range n {
			paths[i] = mustPath(fmt.Sprintf("/struct-key-list[key1=item-%d][key2=%d][key3=E_VALUE_FORTY_TWO]/outer/inner/int32-leaf-field", i, i))
			if err := SetNode(containerWithMultiKeyedList, root, paths[i], val, &InitMissingElements{}); err != nil {
				b.Fatalf("SetNode(%v) setup failed: %v", paths[i], err)
			}
		}
		target := paths[n-1]
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := GetNode(containerWithMultiKeyedList, root, target); err != nil {
					b.Fatalf("GetNode(%v) failed: %v", target, err)
				}
			}
		})
	}
}
