package ytypes

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygot"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
)

func TestUnmarshalSetRequest(t *testing.T) {
	tests := []struct {
		desc               string
		inSchema           *Schema
		inReq              *gpb.SetRequest
		inPreferShadowPath bool
		want               ygot.GoStruct
		wantErr            bool
	}{{
		desc: "updates to an empty struct",
		inSchema: &Schema{
			Root: &ListElemStruct1{},
			SchemaTree: map[string]*yang.Entry{
				"ListElemStruct1": simpleSchema(),
			},
		},
		inReq: &gpb.SetRequest{
			Prefix: &gpb.Path{},
			Update: []*gpb.Update{{
				Path: mustPath("/key1"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "hello"}},
			}, {
				Path: mustPath("/outer/inner"),
				Val: &gpb.TypedValue{Value: &gpb.TypedValue_JsonIetfVal{
					JsonIetfVal: []byte(`
{
	"int32-leaf-list": [42]
}
					`),
				}},
			}},
		},
		want: &ListElemStruct1{
			Key1: ygot.String("hello"),
			Outer: &OuterContainerType1{
				Inner: &InnerContainerType1{
					Int32LeafListName: []int32{42},
				},
			},
		},
	}, {
		desc: "updates to non-empty struct",
		inSchema: &Schema{
			Root: &ListElemStruct1{
				Key1: ygot.String("hello"),
				Outer: &OuterContainerType1{
					Inner: &InnerContainerType1{
						Int32LeafName:     ygot.Int32(43),
						Int32LeafListName: []int32{100},
						StringLeafName:    ygot.String("bear"),
					},
				},
			},
			SchemaTree: map[string]*yang.Entry{
				"ListElemStruct1": simpleSchema(),
			},
		},
		inReq: &gpb.SetRequest{
			Prefix: &gpb.Path{},
			Update: []*gpb.Update{{
				Path: mustPath("/key1"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "hello"}},
			}, {
				Path: mustPath("/outer/inner"),
				Val: &gpb.TypedValue{Value: &gpb.TypedValue_JsonIetfVal{
					JsonIetfVal: []byte(`
{
	"int32-leaf-list": [42]
}
					`),
				}},
			}},
		},
		want: &ListElemStruct1{
			Key1: ygot.String("hello"),
			Outer: &OuterContainerType1{
				Inner: &InnerContainerType1{
					Int32LeafName:     ygot.Int32(43),
					Int32LeafListName: []int32{42},
					StringLeafName:    ygot.String("bear"),
				},
			},
		},
	}, {
		desc: "replaces and update to a non-empty struct",
		inSchema: &Schema{
			Root: &ListElemStruct1{
				Key1: ygot.String("hello"),
				Outer: &OuterContainerType1{
					Inner: &InnerContainerType1{
						Int32LeafName:     ygot.Int32(43),
						Int32LeafListName: []int32{42},
						StringLeafName:    ygot.String("bear"),
					},
				},
			},
			SchemaTree: map[string]*yang.Entry{
				"ListElemStruct1": simpleSchema(),
			},
		},
		inReq: &gpb.SetRequest{
			Prefix: &gpb.Path{},
			Replace: []*gpb.Update{{
				Path: mustPath("/outer/inner"),
				Val: &gpb.TypedValue{Value: &gpb.TypedValue_JsonIetfVal{
					JsonIetfVal: []byte(`
{
	"int32-leaf-list": [42]
}
					`),
				}},
			}},
			Update: []*gpb.Update{{
				Path: mustPath("/outer/inner/string-leaf-field"),
				Val: &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{
					StringVal: "foo",
				}},
			}},
		},
		want: &ListElemStruct1{
			Key1: ygot.String("hello"),
			Outer: &OuterContainerType1{
				Inner: &InnerContainerType1{
					Int32LeafListName: []int32{42},
					StringLeafName:    ygot.String("foo"),
				},
			},
		},
	}, {
		desc: "deletes to a non-empty struct",
		inSchema: &Schema{
			Root: &ListElemStruct1{
				Key1: ygot.String("hello"),
				Outer: &OuterContainerType1{
					Inner: &InnerContainerType1{
						Int32LeafName:     ygot.Int32(43),
						Int32LeafListName: []int32{42},
						StringLeafName:    ygot.String("bear"),
					},
				},
			},
			SchemaTree: map[string]*yang.Entry{
				"ListElemStruct1": simpleSchema(),
			},
		},
		inReq: &gpb.SetRequest{
			Prefix: &gpb.Path{},
			Delete: []*gpb.Path{
				mustPath("/outer"),
			},
		},
		want: &ListElemStruct1{
			Key1: ygot.String("hello"),
		},
	}, {
		desc: "deletes, replaces and update to a non-empty struct",
		inSchema: &Schema{
			Root: &ListElemStruct1{
				Key1: ygot.String("hello"),
				Outer: &OuterContainerType1{
					Inner: &InnerContainerType1{
						Int32LeafName:     ygot.Int32(43),
						Int32LeafListName: []int32{42},
						StringLeafName:    ygot.String("bear"),
					},
				},
			},
			SchemaTree: map[string]*yang.Entry{
				"ListElemStruct1": simpleSchema(),
			},
		},
		inReq: &gpb.SetRequest{
			Prefix: &gpb.Path{},
			Delete: []*gpb.Path{
				mustPath("/outer/inner"),
			},
			Replace: []*gpb.Update{{
				Path: mustPath("/key1"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "world"}},
			}},
			Update: []*gpb.Update{{
				Path: mustPath("/outer/inner/string-leaf-field"),
				Val: &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{
					StringVal: "foo",
				}},
			}},
		},
		want: &ListElemStruct1{
			Key1: ygot.String("world"),
			Outer: &OuterContainerType1{
				Inner: &InnerContainerType1{
					StringLeafName: ygot.String("foo"),
				},
			},
		},
	}, {
		desc: "replaces to a non-empty struct with prefix",
		inSchema: &Schema{
			Root: &ListElemStruct1{
				Key1: ygot.String("hello"),
				Outer: &OuterContainerType1{
					Inner: &InnerContainerType1{
						Int32LeafName:     ygot.Int32(43),
						Int32LeafListName: []int32{42},
						StringLeafName:    ygot.String("bear"),
					},
				},
			},
			SchemaTree: map[string]*yang.Entry{
				"ListElemStruct1":     simpleSchema(),
				"OuterContainerType1": simpleSchema().Dir["outer"],
			},
		},
		inReq: &gpb.SetRequest{
			Prefix: mustPath("/outer"),
			Replace: []*gpb.Update{{
				Path: mustPath("inner"),
				Val: &gpb.TypedValue{Value: &gpb.TypedValue_JsonIetfVal{
					JsonIetfVal: []byte(`
{
	"int32-leaf-list": [42]
}
					`),
				}},
			}},
		},
		want: &ListElemStruct1{
			Key1: ygot.String("hello"),
			Outer: &OuterContainerType1{
				Inner: &InnerContainerType1{
					Int32LeafListName: []int32{42},
				},
			},
		},
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got, err := UnmarshalSetRequest(tt.inSchema, tt.inReq, tt.inPreferShadowPath)
			if gotErr := err != nil; gotErr != tt.wantErr {
				t.Fatalf("got error: %v, want: %v", err, tt.wantErr)
			}
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("(-got, +want):\n%s", diff)
			}
		})
	}
}

func TestUnmarshalNotification(t *testing.T) {
	tests := []struct {
		desc               string
		inSchema           *Schema
		inReq              *gpb.Notification
		inPreferShadowPath bool
		want               ygot.GoStruct
		wantErr            bool
	}{{
		desc: "updates to an empty struct",
		inSchema: &Schema{
			Root: &ListElemStruct1{},
			SchemaTree: map[string]*yang.Entry{
				"ListElemStruct1": simpleSchema(),
			},
		},
		inReq: &gpb.Notification{
			Prefix: &gpb.Path{},
			Update: []*gpb.Update{{
				Path: mustPath("/key1"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "hello"}},
			}, {
				Path: mustPath("/outer/inner"),
				Val: &gpb.TypedValue{Value: &gpb.TypedValue_JsonIetfVal{
					JsonIetfVal: []byte(`
{
	"int32-leaf-list": [42]
}
					`),
				}},
			}},
		},
		want: &ListElemStruct1{
			Key1: ygot.String("hello"),
			Outer: &OuterContainerType1{
				Inner: &InnerContainerType1{
					Int32LeafListName: []int32{42},
				},
			},
		},
	}, {
		desc: "updates to non-empty struct",
		inSchema: &Schema{
			Root: &ListElemStruct1{
				Key1: ygot.String("hello"),
				Outer: &OuterContainerType1{
					Inner: &InnerContainerType1{
						Int32LeafName:     ygot.Int32(43),
						Int32LeafListName: []int32{100},
						StringLeafName:    ygot.String("bear"),
					},
				},
			},
			SchemaTree: map[string]*yang.Entry{
				"ListElemStruct1": simpleSchema(),
			},
		},
		inReq: &gpb.Notification{
			Prefix: &gpb.Path{},
			Update: []*gpb.Update{{
				Path: mustPath("/key1"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "hello"}},
			}, {
				Path: mustPath("/outer/inner"),
				Val: &gpb.TypedValue{Value: &gpb.TypedValue_JsonIetfVal{
					JsonIetfVal: []byte(`
{
	"int32-leaf-list": [42]
}
					`),
				}},
			}},
		},
		want: &ListElemStruct1{
			Key1: ygot.String("hello"),
			Outer: &OuterContainerType1{
				Inner: &InnerContainerType1{
					Int32LeafName:     ygot.Int32(43),
					Int32LeafListName: []int32{42},
					StringLeafName:    ygot.String("bear"),
				},
			},
		},
	}, {
		desc: "deletes to a non-empty struct",
		inSchema: &Schema{
			Root: &ListElemStruct1{
				Key1: ygot.String("hello"),
				Outer: &OuterContainerType1{
					Inner: &InnerContainerType1{
						Int32LeafName:     ygot.Int32(43),
						Int32LeafListName: []int32{42},
						StringLeafName:    ygot.String("bear"),
					},
				},
			},
			SchemaTree: map[string]*yang.Entry{
				"ListElemStruct1": simpleSchema(),
			},
		},
		inReq: &gpb.Notification{
			Prefix: &gpb.Path{},
			Delete: []*gpb.Path{
				mustPath("/outer"),
			},
		},
		want: &ListElemStruct1{
			Key1: ygot.String("hello"),
		},
	}, {
		desc: "deletes and updates to a non-empty struct",
		inSchema: &Schema{
			Root: &ListElemStruct1{
				Key1: ygot.String("hello"),
				Outer: &OuterContainerType1{
					Inner: &InnerContainerType1{
						Int32LeafName:     ygot.Int32(43),
						Int32LeafListName: []int32{42},
						StringLeafName:    ygot.String("bear"),
					},
				},
			},
			SchemaTree: map[string]*yang.Entry{
				"ListElemStruct1": simpleSchema(),
			},
		},
		inReq: &gpb.Notification{
			Prefix: &gpb.Path{},
			Delete: []*gpb.Path{
				mustPath("/outer/inner"),
			},
			Update: []*gpb.Update{{
				Path: mustPath("/key1"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "world"}},
			}, {
				Path: mustPath("/outer/inner/string-leaf-field"),
				Val: &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{
					StringVal: "foo",
				}},
			}},
		},
		want: &ListElemStruct1{
			Key1: ygot.String("world"),
			Outer: &OuterContainerType1{
				Inner: &InnerContainerType1{
					StringLeafName: ygot.String("foo"),
				},
			},
		},
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got, err := UnmarshalNotification(tt.inSchema, tt.inReq, tt.inPreferShadowPath)
			if gotErr := err != nil; gotErr != tt.wantErr {
				t.Fatalf("got error: %v, want: %v", err, tt.wantErr)
			}
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("(-got, +want):\n%s", diff)
			}
		})
	}
}
