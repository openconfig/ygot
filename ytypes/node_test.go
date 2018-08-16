package ytypes

import (
	"fmt"
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

func treeNodesEqual(got, want []*TreeNode) error {
	if len(got) != len(want) {
		return fmt.Errorf("mismatched lengths of nodes, got: %d, want: %d", len(got), len(want))
	}

	for _, w := range want {
		match := false
		for _, g := range got {
			if reflect.DeepEqual(g.Data, w.Data) && reflect.DeepEqual(g.Schema, w.Schema) {
				match = true
				break
			}
		}
		if !match {
			return fmt.Errorf("no match for %#v in %#v", w, got)
		}
	}
	return nil
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

	childContainerSchema := &yang.Entry{
		Name:   "container",
		Kind:   yang.DirectoryEntry,
		Parent: rootSchema,
	}
	rootSchema.Dir["container"] = childContainerSchema

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

	childListSchema := &yang.Entry{
		Name:     "childlist",
		Kind:     yang.DirectoryEntry,
		Parent:   rootSchema,
		Key:      "key",
		ListAttr: &yang.ListAttr{},
		Dir:      map[string]*yang.Entry{},
	}
	rootSchema.Dir["childlist"] = childListSchema

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

	type ChildContainer struct{}

	type ListEntry struct {
		Key *string `path:"key"`
	}

	type MultiListEntry struct {
		Keyone *uint32 `path:"keyone"`
		Keytwo *uint32 `path:"keytwo"`
	}

	type MultiListKey struct {
		Keyone uint32 `path:"keyone"`
		Keytwo uint32 `path:"keytwo"`
	}

	type ListChildContainer struct {
		Value *string `path:"value"`
	}

	type ChildList struct {
		Key            *string             `path:"key"`
		ChildContainer *ListChildContainer `path:"child-container"`
	}

	type RootStruct struct {
		Leaf      *string                          `path:"leaf"`
		Container *ChildContainer                  `path:"container"`
		List      map[string]*ListEntry            `path:"list"`
		Multilist map[MultiListKey]*MultiListEntry `path:"multilist"`
		ChildList map[string]*ChildList            `path:"childlist"`
	}

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
		inData: &RootStruct{
			Leaf: ygot.String("foo"),
		},
		inPath: mustPath("/leaf"),
		wantTreeNodes: []*TreeNode{{
			Data:   ygot.String("foo"),
			Schema: leafSchema,
		}},
	}, {
		desc:     "simple get container",
		inSchema: rootSchema,
		inData: &RootStruct{
			Container: &ChildContainer{},
		},
		inPath: mustPath("/container"),
		wantTreeNodes: []*TreeNode{{
			Data:   &ChildContainer{},
			Schema: childContainerSchema,
		}},
	}, {
		desc:     "simple list",
		inSchema: rootSchema,
		inData: &RootStruct{
			List: map[string]*ListEntry{
				"one": {Key: ygot.String("one")},
				"two": {Key: ygot.String("two")},
			},
		},
		inPath: mustPath("/list[key=one]"),
		wantTreeNodes: []*TreeNode{{
			Data: &ListEntry{
				Key: ygot.String("one"),
			},
			Schema: simpleListSchema,
		}},
	}, {
		desc:     "simple list, unspecified key, no partial match",
		inSchema: rootSchema,
		inData: &RootStruct{
			List: map[string]*ListEntry{
				"one": {Key: ygot.String("one")},
			},
		},
		inPath:           mustPath("/list"),
		wantErrSubstring: "schema key key is not found in gNMI path",
	}, {
		desc:     "simple list, all entries",
		inSchema: rootSchema,
		inData: &RootStruct{
			List: map[string]*ListEntry{
				"one": {Key: ygot.String("one")},
				"two": {Key: ygot.String("two")},
			},
		},
		inPath: mustPath("/list"),
		inArgs: []GetNodeOpt{&GetPartialKeyMatch{}},
		wantTreeNodes: []*TreeNode{{
			Data:   &ListEntry{Key: ygot.String("one")},
			Schema: simpleListSchema,
		}, {
			Data:   &ListEntry{Key: ygot.String("two")},
			Schema: simpleListSchema,
		}},
	}, {
		desc:     "multiple key list",
		inSchema: rootSchema,
		inData: &RootStruct{
			Multilist: map[MultiListKey]*MultiListEntry{
				{Keyone: 1, Keytwo: 2}: {Keyone: ygot.Uint32(1), Keytwo: ygot.Uint32(2)},
			},
		},
		inPath: mustPath("/multilist[keyone=1][keytwo=2]"),
		wantTreeNodes: []*TreeNode{{
			Data:   &MultiListEntry{Keyone: ygot.Uint32(1), Keytwo: ygot.Uint32(2)},
			Schema: multiKeyListSchema,
		}},
	}, {
		desc:     "multiple key list, partial match not allowed",
		inSchema: rootSchema,
		inData: &RootStruct{
			Multilist: map[MultiListKey]*MultiListEntry{
				{Keyone: 1, Keytwo: 2}: {Keyone: ygot.Uint32(1), Keytwo: ygot.Uint32(2)},
				{Keyone: 1, Keytwo: 3}: {Keyone: ygot.Uint32(3), Keytwo: ygot.Uint32(4)},
			},
		},
		inPath:           mustPath("/multilist[keyone=1]"),
		wantErrSubstring: "does not contain a map entry for schema keytwo",
	}, {
		desc:     "multiple key list, partial match allowed",
		inSchema: rootSchema,
		inData: &RootStruct{
			Multilist: map[MultiListKey]*MultiListEntry{
				{Keyone: 1, Keytwo: 2}: {Keyone: ygot.Uint32(1), Keytwo: ygot.Uint32(2)},
				{Keyone: 1, Keytwo: 3}: {Keyone: ygot.Uint32(1), Keytwo: ygot.Uint32(3)},
			},
		},
		inPath: mustPath("/multilist[keyone=1]"),
		inArgs: []GetNodeOpt{&GetPartialKeyMatch{}},
		wantTreeNodes: []*TreeNode{{
			Data:   &MultiListEntry{Keyone: ygot.Uint32(1), Keytwo: ygot.Uint32(2)},
			Schema: multiKeyListSchema,
		}, {
			Data:   &MultiListEntry{Keyone: ygot.Uint32(1), Keytwo: ygot.Uint32(3)},
			Schema: multiKeyListSchema,
		}},
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got, err := GetNode(tt.inSchema, tt.inData, tt.inPath, tt.inArgs...)
			if diff := errdiff.Substring(err, tt.wantErrSubstring); diff != "" {
				t.Fatalf("did not get expected error, %s", diff)
			}

			if err := treeNodesEqual(got, tt.wantTreeNodes); err != nil {
				t.Fatalf("did not get expected result, %v", err)
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
		wantErrSubstring: "root is nil",
	}, {
		desc:             "non-container parent",
		inSchema:         &yang.Entry{Name: "root"},
		inRoot:           "fakeroot",
		inPath:           &gpb.Path{Elem: []*gpb.PathElem{{Name: "foo"}}},
		wantErrSubstring: "can not use a parent that is not a container",
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			_, _, err := retrieveNode(tt.inSchema, tt.inRoot, tt.inPath, tt.inArgs)
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

	type Root struct {
		Ok *string `path:"ok"`
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
		inTestFunc       func(*yang.Entry, interface{}, *gpb.Path, retrieveNodeArgs) ([]interface{}, []*yang.Entry, error)
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
		desc:             "error case of setting a leaf - unimplemented",
		inSchema:         rootSchema,
		inRoot:           &Root{},
		inPath:           &gpb.Path{Elem: []*gpb.PathElem{{Name: "ok"}}},
		inArgs:           retrieveNodeArgs{val: "flounder"},
		inTestFunc:       retrieveNodeContainer,
		wantErrSubstring: "setting leaf/leaflist node is unimplemented",
	}, {
		desc:             "error case - leafref unresolved",
		inSchema:         lrSchema,
		inRoot:           &UnresolvedLeafRef{},
		inPath:           &gpb.Path{Elem: []*gpb.PathElem{{Name: "leaf"}}},
		inTestFunc:       retrieveNodeContainer,
		wantErrSubstring: "failed to resolve schema",
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
		desc:             "no key",
		inSchema:         &yang.Entry{Name: "baz", Key: "bap"},
		inPath:           &gpb.Path{Elem: []*gpb.PathElem{{Name: "bat"}}},
		inTestFunc:       retrieveNodeList,
		wantErrSubstring: "points to a list without a key element",
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
			_, _, err := tt.inTestFunc(tt.inSchema, tt.inRoot, tt.inPath, tt.inArgs)
			if diff := errdiff.Substring(err, tt.wantErrSubstring); diff != "" {
				t.Fatalf("did not get expected error, %s", diff)
			}
		})
	}
}
