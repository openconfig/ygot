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

func TestGetNodesSimpleKeyedList(t *testing.T) {
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
		want       []interface{}
		wantStatus []spb.Status
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
							"key1": "*",
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
			want:       []interface{}{c1.StructKeyList["forty-two"].Outer.Inner.LeafName},
			wantStatus: nil,
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
			want:       []interface{}{c1.StructKeyList["forty-two"].Outer.Inner.LeafName},
			wantStatus: nil,
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
			wantStatus: []spb.Status{toStatus(scpb.Code_NOT_FOUND, `could not find path in tree beyond schema node simple-key-list, (type *ygotutils.ListElemStruct1), remaining path elem:<name:"bad-element" > elem:<name:"inner" > elem:<name:"leaf-field" > `)},
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
			wantStatus: []spb.Status{toStatus(scpb.Code_INVALID_ARGUMENT, `nil data element type *ygotutils.OuterContainerType1, remaining path elem:<name:"inner" > elem:<name:"leaf-field" > `)},
		},
	}

	for _, tt := range tests {
		var val []interface{}
		var status []spb.Status
		GetNodes(containerWithLeafListSchema, tt.rootStruct, tt.path, func(g interface{}) {
			switch v := g.(type) {
			case spb.Status:
				// fmt.Println("got status", v)
				status = append(status, v)
			case interface{}:
				// fmt.Println("got value", v)
				val = append(val, v)
			default:
				fmt.Println("got unknown element", g)
			}
		})
		if got, want := status, tt.wantStatus; !reflect.DeepEqual(got, want) {
			fmt.Println(isNil(got), isNil(want))
			t.Errorf("%s: got error: %v, wanted error? %v", tt.desc, got, want)
		}
		//testErrLog(t, tt.desc, fmt.Errorf(status.GetMessage()))
		if len(status) == 0 {
			if got, want := val, tt.want; !reflect.DeepEqual(got, want) {
				t.Errorf("%s: struct got:\n%v\nwant:\n%v\n", tt.desc, pretty.Sprint(got), pretty.Sprint(want))
			}
		}
	}
}
func TestGetNodesStructKeyedList(t *testing.T) {
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
		want       []interface{}
		wantStatus []spb.Status
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
			want:       []interface{}{c1.StructKeyList[KeyStruct2{"forty-two", 42, 43}].Outer.Inner.LeafName},
			wantStatus: nil,
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
			want:       []interface{}{c1.StructKeyList[KeyStruct2{"forty-two", 42, 43}].Outer.Inner},
			wantStatus: nil,
		}, {
			desc:       "success container",
			rootStruct: c1,
			path: &gpb.Path{
				Elem: []*gpb.PathElem{
					{
						Name: "struct-key-list",
						Key: map[string]string{
							"key1": "forty-two",
							"key2": "42",
							"key3": "*",
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
			want:       []interface{}{c1.StructKeyList[KeyStruct2{"forty-two", 42, 43}].Outer.Inner},
			wantStatus: nil,
		},
	}

	for _, tt := range tests {
		var val []interface{}
		var status []spb.Status
		GetNodes(containerWithLeafListSchema, tt.rootStruct, tt.path, func(g interface{}) {
			switch v := g.(type) {
			case spb.Status:
				// fmt.Println("got status", v)
				status = append(status, v)
			case interface{}:
				// fmt.Println("got value", v)
				val = append(val, v)
			default:
				fmt.Println("got unknown element", g)
			}
		})
		if got, want := status, tt.wantStatus; !reflect.DeepEqual(got, want) {
			t.Errorf("%s: got error: %v, wanted error? %v", tt.desc, got, want)
		}
		//testErrLog(t, tt.desc, fmt.Errorf(status.GetMessage()))
		if len(status) > 0 {
			if got, want := val, tt.want; !reflect.DeepEqual(got, want) {
				t.Errorf("%s: struct got:\n%v\nwant:\n%v\n", tt.desc, pretty.Sprint(got), pretty.Sprint(want))
			}
		}
	}
}
