package ygotutils

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/kylelemons/godebug/pretty"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygot"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
	scpb "google.golang.org/genproto/googleapis/rpc/code"
	spb "google.golang.org/genproto/googleapis/rpc/status"
)

var (
	// testErrOutput controls whether expect error test cases log the error
	// values.
	testErrOutput = false
)

// EnumType is used as an enum type.
type EnumType int64

func (EnumType) ΛMap() map[string]map[int64]ygot.EnumDefinition {
	m := map[string]map[int64]ygot.EnumDefinition{
		"EnumType": map[int64]ygot.EnumDefinition{
			42: {Name: "E_VALUE_FORTY_TWO"},
		},
	}
	return m
}

// testErrLog logs err to t if err != nil and global value testErrOutput is set.
func testErrLog(t *testing.T, desc string, err error) {
	if err != nil {
		if testErrOutput {
			t.Logf("%s: %v", desc, err)
		}
	}
}

// errToString returns the string representation of err and the empty string if
// err is nil.
func errToString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func isOK(status spb.Status) bool {
	return status.Code == int32(scpb.Code_OK)
}

type InnerContainerType1 struct {
	LeafName *int32 `path:"leaf-field"`
}
type OuterContainerType1 struct {
	Inner *InnerContainerType1 `path:"inner|config/inner"`
}
type ListElemStruct1 struct {
	Key1   *string              `path:"key1"`
	Outer  *OuterContainerType1 `path:"outer"`
	Outer2 *OuterContainerType1 `path:"outer2"`
}
type ContainerStruct1 struct {
	StructKeyList map[string]*ListElemStruct1 `path:"config/simple-key-list"`
}

func (*InnerContainerType1) IsYANGGoStruct() {}
func (*OuterContainerType1) IsYANGGoStruct() {}
func (*ListElemStruct1) IsYANGGoStruct()     {}
func (*ContainerStruct1) IsYANGGoStruct()    {}

func TestGetNodeSimpleKeyedList(t *testing.T) {
	containerWithLeafListSchema := &yang.Entry{
		Name: "container",
		Kind: yang.DirectoryEntry,
		Dir: map[string]*yang.Entry{
			"config": &yang.Entry{
				Name: "config",
				Kind: yang.DirectoryEntry,
				Dir: map[string]*yang.Entry{
					"simple-key-list": &yang.Entry{
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
									"inner": &yang.Entry{
										Name: "inner",
										Kind: yang.DirectoryEntry,
										Dir: map[string]*yang.Entry{
											"leaf-field": &yang.Entry{
												Name: "leaf-field",
												Kind: yang.LeafEntry,
												Type: &yang.YangType{
													Kind: yang.Yleafref,
													Path: "../../config/inner/leaf-field",
												},
											},
										},
									},
									"config": &yang.Entry{
										Name: "config",
										Kind: yang.DirectoryEntry,
										Dir: map[string]*yang.Entry{
											"inner": &yang.Entry{
												Name: "inner",
												Kind: yang.DirectoryEntry,
												Dir: map[string]*yang.Entry{
													"leaf-field": &yang.Entry{
														Name: "leaf-field",
														Kind: yang.LeafEntry,
														Type: &yang.YangType{Kind: yang.Yint32},
													},
												},
											},
										},
									},
								},
							},
							"outer2": {
								Name: "outer2",
								Kind: yang.DirectoryEntry,
								Dir: map[string]*yang.Entry{
									"inner": &yang.Entry{
										Name: "inner",
										Kind: yang.DirectoryEntry,
										Dir: map[string]*yang.Entry{
											"leaf-field": &yang.Entry{
												Name: "leaf-field",
												Kind: yang.LeafEntry,
												Type: &yang.YangType{Kind: yang.Yint32},
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

	c1 := &ContainerStruct1{
		StructKeyList: map[string]*ListElemStruct1{
			"forty-two": &ListElemStruct1{
				Key1:  ygot.String("forty-two"),
				Outer: &OuterContainerType1{Inner: &InnerContainerType1{LeafName: ygot.Int32(1234)}},
			},
		},
	}

	tests := []struct {
		desc       string
		rootStruct ygot.GoStruct
		path       *gpb.Path
		want       interface{}
		wantStatus spb.Status
	}{
		{
			desc:       "success leaf-ref",
			rootStruct: c1,
			path: &gpb.Path{
				Elem: []*gpb.PathElem{
					{
						Name: "config",
					},
					{
						Name: "simple-key-list",
						Key: map[string]string{
							"key1": "forty-two",
						},
					},
					{
						Name: "outer",
					},
					{
						Name: "inner",
					},
					{
						Name: "leaf-field",
					},
				},
			},
			want:       c1.StructKeyList["forty-two"].Outer.Inner.LeafName,
			wantStatus: statusOK,
		},
		{
			desc:       "success leaf full path",
			rootStruct: c1,
			path: &gpb.Path{
				Elem: []*gpb.PathElem{
					{
						Name: "config",
					},
					{
						Name: "simple-key-list",
						Key: map[string]string{
							"key1": "forty-two",
						},
					},
					{
						Name: "outer",
					},
					{
						Name: "config",
					},
					{
						Name: "inner",
					},
					{
						Name: "leaf-field",
					},
				},
			},
			want:       c1.StructKeyList["forty-two"].Outer.Inner.LeafName,
			wantStatus: statusOK,
		},
		{
			desc:       "bad path",
			rootStruct: c1,
			path: &gpb.Path{
				Elem: []*gpb.PathElem{
					{
						Name: "config",
					},
					{
						Name: "simple-key-list",
						Key: map[string]string{
							"key1": "forty-two",
						},
					},
					{
						Name: "bad-element",
					},
					{
						Name: "inner",
					},
					{
						Name: "leaf-field",
					},
				},
			},
			want:       nil,
			wantStatus: toStatus(scpb.Code_NOT_FOUND, `could not find path in tree beyond schema node simple-key-list, (type *ygotutils.ListElemStruct1), remaining path elem:<name:"bad-element" > elem:<name:"inner" > elem:<name:"leaf-field" > `),
		},
		{
			desc:       "nil field",
			rootStruct: c1,
			path: &gpb.Path{
				Elem: []*gpb.PathElem{
					{
						Name: "config",
					},
					{
						Name: "simple-key-list",
						Key: map[string]string{
							"key1": "forty-two",
						},
					},
					{
						Name: "outer2",
					},
					{
						Name: "inner",
					},
					{
						Name: "leaf-field",
					},
				},
			},
			want:       nil,
			wantStatus: toStatus(scpb.Code_INVALID_ARGUMENT, `nil data element type *ygotutils.OuterContainerType1, remaining path elem:<name:"inner" > elem:<name:"leaf-field" > `),
		},
	}

	for _, tt := range tests {
		val, status := GetNode(containerWithLeafListSchema, tt.rootStruct, tt.path)
		if got, want := status, tt.wantStatus; !reflect.DeepEqual(got, want) {
			t.Errorf("%s: got error: %v, wanted error? %v", tt.desc, got, want)
		}
		testErrLog(t, tt.desc, fmt.Errorf(status.GetMessage()))
		if isOK(status) {
			if got, want := val, tt.want; !reflect.DeepEqual(got, want) {
				t.Errorf("%s: struct got:\n%v\nwant:\n%v\n", tt.desc, pretty.Sprint(got), pretty.Sprint(want))
			}
		}
	}
}

type InnerContainerType2 struct {
	LeafName *int32 `path:"leaf-field"`
}
type OuterContainerType2 struct {
	Inner *InnerContainerType2 `path:"inner"`
}
type KeyStruct2 struct {
	Key1    string
	Key2    int32
	EnumKey EnumType
}
type ListElemStruct2 struct {
	Key1    *string              `path:"key1"`
	Key2    *int32               `path:"key2"`
	EnumKey EnumType             `path:"key3"`
	Outer   *OuterContainerType2 `path:"outer"`
}
type ContainerStruct2 struct {
	StructKeyList map[KeyStruct2]*ListElemStruct2 `path:"struct-key-list"`
}

func (*InnerContainerType2) IsYANGGoStruct() {}
func (*OuterContainerType2) IsYANGGoStruct() {}
func (*ListElemStruct2) IsYANGGoStruct()     {}
func (*ContainerStruct2) IsYANGGoStruct()    {}

func TestGetNodeStructKeyedList(t *testing.T) {
	containerWithLeafListSchema := &yang.Entry{
		Name: "container",
		Kind: yang.DirectoryEntry,
		Dir: map[string]*yang.Entry{
			"struct-key-list": &yang.Entry{
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
							"inner": &yang.Entry{
								Name: "inner",
								Kind: yang.DirectoryEntry,
								Dir: map[string]*yang.Entry{
									"leaf-field": &yang.Entry{
										Name: "leaf-field",
										Kind: yang.LeafEntry,
										Type: &yang.YangType{Kind: yang.Yint32},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	c1 := &ContainerStruct2{
		StructKeyList: map[KeyStruct2]*ListElemStruct2{
			{"forty-two", 42, 43}: &ListElemStruct2{
				Key1:    ygot.String("forty-two"),
				Key2:    ygot.Int32(42),
				EnumKey: 43,
				Outer:   &OuterContainerType2{Inner: &InnerContainerType2{LeafName: ygot.Int32(1234)}},
			},
		},
	}

	tests := []struct {
		desc       string
		rootStruct ygot.GoStruct
		path       *gpb.Path
		want       interface{}
		wantStatus spb.Status
	}{
		{
			desc:       "success leaf",
			rootStruct: c1,
			path: &gpb.Path{
				Elem: []*gpb.PathElem{
					{
						Name: "struct-key-list",
						Key: map[string]string{
							"key1": "forty-two",
							"key2": "42",
							"key3": "43",
						},
					},
					{
						Name: "outer",
					},
					{
						Name: "inner",
					},
					{
						Name: "leaf-field",
					},
				},
			},
			want:       c1.StructKeyList[KeyStruct2{"forty-two", 42, 43}].Outer.Inner.LeafName,
			wantStatus: statusOK,
		},
		{
			desc:       "success container",
			rootStruct: c1,
			path: &gpb.Path{
				Elem: []*gpb.PathElem{
					{
						Name: "struct-key-list",
						Key: map[string]string{
							"key1": "forty-two",
							"key2": "42",
							"key3": "43",
						},
					},
					{
						Name: "outer",
					},
					{
						Name: "inner",
					},
				},
			},
			want:       c1.StructKeyList[KeyStruct2{"forty-two", 42, 43}].Outer.Inner,
			wantStatus: statusOK,
		},
	}

	for _, tt := range tests {
		val, status := GetNode(containerWithLeafListSchema, tt.rootStruct, tt.path)
		if got, want := status, tt.wantStatus; !reflect.DeepEqual(got, want) {
			t.Errorf("%s: got error: %v, wanted error? %v", tt.desc, got, want)
		}
		testErrLog(t, tt.desc, fmt.Errorf(status.GetMessage()))
		if isOK(status) {
			if got, want := val, tt.want; !reflect.DeepEqual(got, want) {
				t.Errorf("%s: struct got:\n%v\nwant:\n%v\n", tt.desc, pretty.Sprint(got), pretty.Sprint(want))
			}
		}
	}
}

type InnerContainerType3 struct {
	LeafName *int32 `path:"leaf-field"`
}
type OuterContainerType3 struct {
	Inner *InnerContainerType3 `path:"inner"`
}
type ListElemStruct3 struct {
	Key1   *string              `path:"key1"`
	Outer  *OuterContainerType3 `path:"outer|config/outer"`
	Outer2 *OuterContainerType3 `path:"outer2"`
}
type ContainerStruct3 struct {
	StructKeyList map[string]*ListElemStruct3 `path:"simple-key-list"`
}

func (*InnerContainerType3) IsYANGGoStruct() {}
func (*OuterContainerType3) IsYANGGoStruct() {}
func (*ListElemStruct3) IsYANGGoStruct()     {}
func (*ContainerStruct3) IsYANGGoStruct()    {}

func TestNewNodeSimpleKeyedList(t *testing.T) {

	var c1 *ContainerStruct3

	tests := []struct {
		desc       string
		rootStruct ygot.GoStruct
		path       *gpb.Path
		want       interface{}
		wantStatus spb.Status
	}{
		{
			desc:       "success leaf",
			rootStruct: c1,
			path: &gpb.Path{
				Elem: []*gpb.PathElem{
					{
						Name: "simple-key-list",
						Key: map[string]string{
							"key1": "forty-two",
						},
					},
					{
						Name: "outer",
					},
					{
						Name: "inner",
					},
					{
						Name: "leaf-field",
					},
				},
			},
			want:       ygot.Int32(0),
			wantStatus: statusOK,
		},
		{
			desc:       "success leaf compressed path",
			rootStruct: c1,
			path: &gpb.Path{
				Elem: []*gpb.PathElem{
					{
						Name: "simple-key-list",
						Key: map[string]string{
							"key1": "forty-two",
						},
					},
					{
						Name: "config",
					},
					{
						Name: "outer",
					},
					{
						Name: "inner",
					},
					{
						Name: "leaf-field",
					},
				},
			},
			want:       ygot.Int32(0),
			wantStatus: statusOK,
		},
		{
			desc:       "success container",
			rootStruct: c1,
			path: &gpb.Path{
				Elem: []*gpb.PathElem{
					{
						Name: "simple-key-list",
						Key: map[string]string{
							"key1": "forty-two",
						},
					},
					{
						Name: "outer",
					},
				},
			},
			want:       &OuterContainerType3{},
			wantStatus: statusOK,
		},
		{
			desc:       "bad path",
			rootStruct: c1,
			path: &gpb.Path{
				Elem: []*gpb.PathElem{
					{
						Name: "simple-key-list",
						Key: map[string]string{
							"key1": "forty-two",
						},
					},
					{
						Name: "bad-element",
					},
					{
						Name: "inner",
					},
					{
						Name: "leaf-field",
					},
				},
			},
			want:       nil,
			wantStatus: toStatus(scpb.Code_NOT_FOUND, `could not find path in tree beyond type *ygotutils.ListElemStruct3, remaining path elem:<name:"bad-element" > elem:<name:"inner" > elem:<name:"leaf-field" > `),
		},
	}

	for _, tt := range tests {
		val, status := NewNode(reflect.TypeOf(tt.rootStruct), tt.path)
		if got, want := status, tt.wantStatus; !reflect.DeepEqual(got, want) {
			t.Errorf("%s: got error: %v, wanted error? %v", tt.desc, got, want)
		}
		testErrLog(t, tt.desc, fmt.Errorf(status.GetMessage()))
		if isOK(status) {
			if got, want := reflect.TypeOf(val), reflect.TypeOf(tt.want); got != want {
				t.Errorf("%s: got: %s, want: %s", tt.desc, got, want)
			}
		}
	}
}

type InnerContainerType4 struct {
	LeafName *int32 `path:"leaf-field"`
}
type OuterContainerType4 struct {
	Inner *InnerContainerType4 `path:"inner"`
}
type KeyStruct4 struct {
	Key1    string
	Key2    int32
	EnumKey EnumType
}
type ListElemStruct4 struct {
	Key1    *string              `path:"key1"`
	Key2    *int32               `path:"key2"`
	EnumKey EnumType             `path:"key3"`
	Outer   *OuterContainerType4 `path:"outer"`
}
type ContainerStruct4 struct {
	StructKeyList map[KeyStruct4]*ListElemStruct4 `path:"struct-key-list"`
}

func (*InnerContainerType4) IsYANGGoStruct() {}
func (*OuterContainerType4) IsYANGGoStruct() {}
func (*ListElemStruct4) IsYANGGoStruct()     {}
func (*ContainerStruct4) IsYANGGoStruct()    {}

func TestNewNodeStructKeyedList(t *testing.T) {
	c1 := &ContainerStruct4{
		StructKeyList: map[KeyStruct4]*ListElemStruct4{
			{"forty-two", 42, 43}: &ListElemStruct4{
				Key1:    ygot.String("forty-two"),
				Key2:    ygot.Int32(42),
				EnumKey: 43,
				Outer:   &OuterContainerType4{Inner: &InnerContainerType4{LeafName: ygot.Int32(1234)}},
			},
		},
	}

	tests := []struct {
		desc       string
		rootStruct ygot.GoStruct
		path       *gpb.Path
		want       interface{}
		wantStatus spb.Status
	}{
		{
			desc:       "success leaf",
			rootStruct: c1,
			path: &gpb.Path{
				Elem: []*gpb.PathElem{
					{
						Name: "struct-key-list",
						Key: map[string]string{
							"key1": "forty-two",
							"key2": "42",
							"key3": "43",
						},
					},
					{
						Name: "outer",
					},
					{
						Name: "inner",
					},
					{
						Name: "leaf-field",
					},
				},
			},
			want:       ygot.Int32(0),
			wantStatus: statusOK,
		},
		{
			desc:       "success container",
			rootStruct: c1,
			path: &gpb.Path{
				Elem: []*gpb.PathElem{
					{
						Name: "struct-key-list",
						Key: map[string]string{
							"key1": "forty-two",
							"key2": "42",
							"key3": "43",
						},
					},
					{
						Name: "outer",
					},
				},
			},
			want:       &OuterContainerType4{},
			wantStatus: statusOK,
		},
		{
			desc:       "bad path",
			rootStruct: c1,
			path: &gpb.Path{
				Elem: []*gpb.PathElem{
					{
						Name: "struct-key-list",
						Key: map[string]string{
							"key1": "forty-two",
						},
					},
					{
						Name: "bad-element",
					},
					{
						Name: "inner",
					},
					{
						Name: "leaf-field",
					},
				},
			},
			want:       nil,
			wantStatus: toStatus(scpb.Code_NOT_FOUND, `could not find path in tree beyond type *ygotutils.ListElemStruct4, remaining path elem:<name:"bad-element" > elem:<name:"inner" > elem:<name:"leaf-field" > `),
		},
	}

	for _, tt := range tests {
		val, status := NewNode(reflect.TypeOf(tt.rootStruct), tt.path)
		if got, want := status, tt.wantStatus; !reflect.DeepEqual(got, want) {
			t.Errorf("%s: got error: %v, wanted error? %v", tt.desc, got, want)
		}
		testErrLog(t, tt.desc, fmt.Errorf(status.GetMessage()))
		if isOK(status) {
			if got, want := val, tt.want; !reflect.DeepEqual(got, want) {
				t.Errorf("%s: got: %s, want: %s", tt.desc, got, want)
			}
		}
	}
}
