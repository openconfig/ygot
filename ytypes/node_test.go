package ytypes

import (
	"reflect"
	"testing"

	"github.com/kylelemons/godebug/pretty"
	"github.com/openconfig/gnmi/errdiff"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygot"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
)

type InnerContainerType1 struct {
	Int32LeafName  *int32   `path:"int32-leaf-field"`
	StringLeafName *string  `path:"string-leaf-field"`
	EnumLeafName   EnumType `path:"enum-leaf-field"`
	Annotation     *string  `ygotAnnotation:"true"`
}
type OuterContainerType1 struct {
	Inner *InnerContainerType1 `path:"inner|config/inner"`
}
type ListElemStruct1 struct {
	Key1       *string              `path:"key1"`
	Outer      *OuterContainerType1 `path:"outer"`
	Annotation *string              `ygotAnnotation:"true"`
}
type ContainerStruct1 struct {
	StructKeyList map[string]*ListElemStruct1 `path:"config/simple-key-list"`
}
type ListElemStruct2 struct {
	Key1       *uint32              `path:"key1"`
	Outer      *OuterContainerType1 `path:"outer"`
	Annotation *string              `ygotAnnotation:"true"`
}
type ContainerStruct2 struct {
	StructKeyList map[uint32]*ListElemStruct2 `path:"config/simple-key-list"`
}

func (*InnerContainerType1) IsYANGGoStruct() {}
func (*OuterContainerType1) IsYANGGoStruct() {}
func (*ListElemStruct1) IsYANGGoStruct()     {}
func (*ContainerStruct1) IsYANGGoStruct()    {}

func TestGetOrCreateNodeSimpleKey(t *testing.T) {
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
						ListAttr: &yang.ListAttr{MinElements: &yang.Value{Name: "0"}},
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
														Name: "leaf-field",
														Kind: yang.LeafEntry,
														Type: &yang.YangType{Kind: yang.Yint32},
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
						ListAttr: &yang.ListAttr{MinElements: &yang.Value{Name: "0"}},
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

	tests := []struct {
		inDesc           string
		inParent         interface{}
		inSchema         *yang.Entry
		inPath           *gpb.Path
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
			inSchema: containerWithStringKey,
			inPath:   mustPath("/config/simple-key-list[key1=forty-two]/outer/inner/int32-leaf-field"),
			want:     ygot.Int32(42),
		},
		{
			inDesc:   "success get int32 leaf with a new key",
			inParent: &ContainerStruct1{},
			inSchema: containerWithStringKey,
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
			inSchema: containerWithStringKey,
			inPath:   mustPath("/config/simple-key-list[key1=forty-two]/outer/inner/string-leaf-field"),
			want:     ygot.String("forty_two"),
		},
		{
			inDesc:   "success get string leaf with a new key",
			inParent: &ContainerStruct1{},
			inSchema: containerWithStringKey,
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
			inSchema: containerWithStringKey,
			inPath:   mustPath("/config/simple-key-list[key1=forty-two]/outer/inner/enum-leaf-field"),
			want:     EnumType(43),
		},
		{
			inDesc:   "success get enum leaf with a new key",
			inParent: &ContainerStruct1{},
			inSchema: containerWithStringKey,
			inPath:   mustPath("/config/simple-key-list[key1=forty-two]/outer/inner/enum-leaf-field"),
			want:     EnumType(0),
		},
		{
			inDesc:           "fail get enum leaf incorrect container schema",
			inParent:         &ContainerStruct1{},
			inSchema:         containerWithStringKey,
			inPath:           mustPath("/config/simple-key-list[key1=forty-two]/INVAID_CONTAINER/inner/enum-leaf-field"),
			wantErrSubstring: "no match found in *ytypes.ListElemStruct1",
		},
		{
			inDesc:           "fail get enum leaf incorrect leaf schema",
			inParent:         &ContainerStruct1{},
			inSchema:         containerWithStringKey,
			inPath:           mustPath("/config/simple-key-list[key1=forty-two]/outer/inner/INVALID_LEAF"),
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
	}

	for i, tt := range tests {
		t.Run(tt.inDesc, func(t *testing.T) {
			got, _, err := GetOrCreateNode(tt.inSchema, tt.inParent, tt.inPath)
			if diff := errdiff.Substring(err, tt.wantErrSubstring); diff != "" {
				t.Fatalf("#%d: %s\ngot %v\nwant %v\n", i, tt.inDesc, err, tt.wantErrSubstring)
			}
			if err != nil {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("%s:\ngot: %v\nwant: %v\n", tt.inDesc, pretty.Sprint(got), pretty.Sprint(tt.want))
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
type ContainerStruct3 struct {
	StructKeyList map[KeyStruct]*ListElemStruct3 `path:"struct-key-list"`
}

func (*ListElemStruct3) IsYANGGoStruct()  {}
func (*ContainerStruct3) IsYANGGoStruct() {}

func TestGetOrCreateNodeStructKeyedList(t *testing.T) {
	containerWithLeafListSchema := &yang.Entry{
		Name: "container",
		Kind: yang.DirectoryEntry,
		Dir: map[string]*yang.Entry{
			"struct-key-list": {
				Name:     "struct-key-list",
				Kind:     yang.DirectoryEntry,
				ListAttr: &yang.ListAttr{MinElements: &yang.Value{Name: "0"}},
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
			inSchema: containerWithLeafListSchema,
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
			inSchema: containerWithLeafListSchema,
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
			inSchema: containerWithLeafListSchema,
			inParent: &ContainerStruct3{},
			inPath:   mustPath("/struct-key-list[key1=forty-two][key2=42][key3=E_VALUE_FORTY_TWO]/outer/inner/string-leaf-field"),
			want:     ygot.String(""),
		},
		{
			inDesc:           "fail get string leaf from a struct keyed list due to invalid key",
			inSchema:         containerWithLeafListSchema,
			inParent:         &ContainerStruct3{},
			inPath:           mustPath("/struct-key-list[key1=forty-two][key2=42][key3=INVALID_ENUM]/outer/inner/string-leaf-field"),
			wantErrSubstring: "no enum matching with INVALID_ENUM: <nil>",
		},
		{
			inDesc:           "fail get due to partial key for struct keyed list",
			inSchema:         containerWithLeafListSchema,
			inParent:         &ContainerStruct3{},
			inPath:           mustPath("/struct-key-list[key1=forty-two][key2=42]/outer"),
			wantErrSubstring: "missing key3 key in map",
		},
	}

	for i, tt := range tests {
		t.Run(tt.inDesc, func(t *testing.T) {
			got, _, err := GetOrCreateNode(tt.inSchema, tt.inParent, tt.inPath)
			if diff := errdiff.Substring(err, tt.wantErrSubstring); diff != "" {
				t.Fatalf("#%d: %s\ngot %v\nwant %v\n", i, tt.inDesc, err, tt.wantErrSubstring)
			}
			if err != nil {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("%s:\ngot: %v\nwant: %v\n", tt.inDesc, pretty.Sprint(got), pretty.Sprint(tt.want))
			}
		})
	}
}

func TestGetOrCreateNodeWithSimpleSchema(t *testing.T) {
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
			inSchema: simpleSchema,
			inParent: &ListElemStruct1{},
			inPath:   mustPath("/outer/inner"),
			want:     &InnerContainerType1{},
		},
		{
			inDesc:   "success retrieving container with indirect descendant schema",
			inSchema: simpleSchema,
			inParent: &ListElemStruct1{},
			inPath:   mustPath("/outer/config/inner"),
			want:     &InnerContainerType1{},
		},
		{
			inDesc:   "success retrieving int32 leaf with direct descendant schema",
			inSchema: simpleSchema,
			inParent: &ListElemStruct1{},
			inPath:   mustPath("/outer/inner/int32-leaf-field"),
			want:     ygot.Int32(0),
		},
		{
			inDesc:   "success retrieving int32 leaf with indirect descendant schema",
			inSchema: simpleSchema,
			inParent: &ListElemStruct1{},
			inPath:   mustPath("/outer/config/inner/int32-leaf-field"),
			want:     ygot.Int32(0),
		},
		{
			inDesc:   "success retrieving enum leaf with direct descendant schema",
			inSchema: simpleSchema,
			inParent: &ListElemStruct1{},
			inPath:   mustPath("/outer/inner/enum-leaf-field"),
			want:     EnumType(0),
		},
		{
			inDesc:   "success retrieving enum leaf from existing container",
			inSchema: simpleSchema,
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
			inSchema: simpleSchema,
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
				t.Fatalf("#%d: %s\ngot %v\nwant %v\n", i, tt.inDesc, err, tt.wantErrSubstring)
			}
			if err != nil {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("%s:\ngot: %v\nwant: %v\n", tt.inDesc, pretty.Sprint(got), pretty.Sprint(tt.want))
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
