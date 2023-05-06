package ytypes

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygot"

	"github.com/openconfig/gnmi/proto/gnmi"
	gpb "github.com/openconfig/gnmi/proto/gnmi"
)

func TestUnmarshalSetRequest(t *testing.T) {
	tests := []struct {
		desc            string
		inSchema        *Schema
		inReq           *gpb.SetRequest
		inUnmarshalOpts []UnmarshalOpt
		want            ygot.GoStruct
		wantErr         bool
	}{{
		desc: "nil input",
		inSchema: &Schema{
			Root: &ListElemStruct1{},
			SchemaTree: map[string]*yang.Entry{
				"ListElemStruct1": simpleSchema(),
			},
		},
		want: &ListElemStruct1{},
	}, {
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
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "invalid"}},
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
			Key1: ygot.String("invalid"),
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
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "world"}},
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
			Key1: ygot.String("world"),
			Outer: &OuterContainerType1{
				Inner: &InnerContainerType1{
					Int32LeafName:     ygot.Int32(43),
					Int32LeafListName: []int32{42},
					StringLeafName:    ygot.String("bear"),
				},
			},
		},
	}, {
		desc: "updates of invalid paths to non-empty struct with IgnoreExtraFields",
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
		inUnmarshalOpts: []UnmarshalOpt{&IgnoreExtraFields{}},
		inReq: &gpb.SetRequest{
			Prefix: &gpb.Path{},
			Update: []*gpb.Update{{
				Path: mustPath("/invalidkey1"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "world"}},
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
				Path: mustPath("/outer/inner/config/int32-leaf-field"),
				Val: &gpb.TypedValue{Value: &gpb.TypedValue_IntVal{
					IntVal: 42,
				}},
			}},
		},
		want: &ListElemStruct1{
			Key1: ygot.String("world"),
			Outer: &OuterContainerType1{
				Inner: &InnerContainerType1{
					Int32LeafName: ygot.Int32(42),
				},
			},
		},
	}, {
		desc: "deletes and update to a non-empty struct with preferShadowPath (no effect)",
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
				mustPath("/outer/inner/config/int32-leaf-field"),
			},
		},
		inUnmarshalOpts: []UnmarshalOpt{&PreferShadowPath{}},
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
		desc: "deletes, replaces and update to a non-empty struct with preferShadowPath (no effect)",
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
				Path: mustPath("/key1"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "world"}},
			}, {
				Path: mustPath("/outer/inner/config/int32-leaf-field"),
				Val: &gpb.TypedValue{Value: &gpb.TypedValue_IntVal{
					IntVal: 42,
				}},
			}},
		},
		inUnmarshalOpts: []UnmarshalOpt{&PreferShadowPath{}},
		want: &ListElemStruct1{
			Key1: ygot.String("world"),
			Outer: &OuterContainerType1{
				Inner: &InnerContainerType1{
					Int32LeafName:     ygot.Int32(43),
					Int32LeafListName: []int32{42},
					StringLeafName:    ygot.String("bear"),
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
	}, {
		desc: "replaces to a non-existent path",
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
			Prefix: mustPath("/outer-planets"),
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
		wantErr: true,
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := UnmarshalSetRequest(tt.inSchema, tt.inReq, tt.inUnmarshalOpts...)
			if gotErr := err != nil; gotErr != tt.wantErr {
				t.Fatalf("got error: %v, want: %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				if diff := cmp.Diff(tt.inSchema.Root, tt.want); diff != "" {
					t.Errorf("(-got, +want):\n%s", diff)
				}
			}
		})
	}
}

// TestUnmarshalSetRequestWithNodeCache verifies the behavior of UnmarshalSetRequest
// when node cache is used (optional).
//
// Since the basic tests for UnmarshalSetRequest are covered by TestUnmarshalSetRequest,
// this test function focuses on data changes in the cache.
func TestUnmarshalSetRequestWithNodeCache(t *testing.T) {
	inSchema := &Schema{
		Root: &ListElemStruct1{
			Key1: ygot.String("hello"),
			Outer: &OuterContainerType1{
				Inner: &InnerContainerType1{
					Int32LeafListName: []int32{43},
				},
			},
		},
		SchemaTree: map[string]*yang.Entry{
			"ListElemStruct1":     simpleSchema(),
			"OuterContainerType1": simpleSchema().Dir["outer"],
		},
	}

	tests := []struct {
		desc               string
		inSchema           *Schema
		inReq              *gpb.SetRequest
		inUnmarshalOpts    []UnmarshalOpt
		want               ygot.GoStruct
		wantNodeCacheStore map[string]*cachedNodeInfo // Only `key` and `nodes` (`Data` and `Path`) are compared.
		wantErr            bool
	}{{
		desc:     "updates to an empty struct",
		inSchema: inSchema,
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
		wantNodeCacheStore: map[string]*cachedNodeInfo{
			`{"name":"key1"}`: {
				nodes: []*TreeNode{{
					Data: func(s string) *string { return &s }("hello"),
					Path: &gnmi.Path{
						Elem: []*gnmi.PathElem{{Name: "key1"}},
					},
				}},
			},
			`{"name":"outer"},{"name":"inner"}`: {
				nodes: []*TreeNode{
					{
						Data: &InnerContainerType1{
							Int32LeafListName: []int32{42},
						},
						Path: &gnmi.Path{
							Elem: []*gnmi.PathElem{
								{Name: "outer"},
								{Name: "inner"},
							},
						},
					},
				},
			},
		},
	}, {
		desc:     "updates to non-empty struct",
		inSchema: inSchema,
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
	"int32-leaf-list": [43]
}
					`),
				}},
			}},
		},
		want: &ListElemStruct1{
			Key1: ygot.String("hello"),
			Outer: &OuterContainerType1{
				Inner: &InnerContainerType1{
					Int32LeafListName: []int32{43},
				},
			},
		},
		wantNodeCacheStore: map[string]*cachedNodeInfo{
			`{"name":"key1"}`: {
				nodes: []*TreeNode{{
					Data: func(s string) *string { return &s }("hello"),
					Path: &gnmi.Path{
						Elem: []*gnmi.PathElem{{Name: "key1"}},
					},
				}},
			},
			`{"name":"outer"},{"name":"inner"}`: {
				nodes: []*TreeNode{
					{
						Data: &InnerContainerType1{
							Int32LeafListName: []int32{43},
						},
						Path: &gnmi.Path{
							Elem: []*gnmi.PathElem{
								{Name: "outer"},
								{Name: "inner"},
							},
						},
					},
				},
			},
		},
	}, {
		desc:            "updates of invalid paths to non-empty struct with IgnoreExtraFields",
		inSchema:        inSchema,
		inUnmarshalOpts: []UnmarshalOpt{&IgnoreExtraFields{}},
		inReq: &gpb.SetRequest{
			Prefix: &gpb.Path{},
			Update: []*gpb.Update{{
				Path: mustPath("/invalidkey1"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "invalid"}},
			}, {
				Path: mustPath("/outer/inner"),
				Val: &gpb.TypedValue{Value: &gpb.TypedValue_JsonIetfVal{
					JsonIetfVal: []byte(`
{
	"int32-leaf-list": [41]
}
					`),
				}},
			}},
		},
		want: &ListElemStruct1{
			Key1: ygot.String("hello"),
			Outer: &OuterContainerType1{
				Inner: &InnerContainerType1{
					Int32LeafListName: []int32{41},
				},
			},
		},
		wantNodeCacheStore: map[string]*cachedNodeInfo{
			`{"name":"key1"}`: {
				nodes: []*TreeNode{{
					Data: func(s string) *string { return &s }("hello"),
					Path: &gnmi.Path{
						Elem: []*gnmi.PathElem{{Name: "key1"}},
					},
				}},
			},
			`{"name":"outer"},{"name":"inner"}`: {
				nodes: []*TreeNode{
					{
						Data: &InnerContainerType1{
							Int32LeafListName: []int32{41},
						},
						Path: &gnmi.Path{
							Elem: []*gnmi.PathElem{
								{Name: "outer"},
								{Name: "inner"},
							},
						},
					},
				},
			},
		},
	}, {
		desc:     "replaces and update to a non-empty struct",
		inSchema: inSchema,
		inReq: &gpb.SetRequest{
			Prefix: &gpb.Path{},
			Replace: []*gpb.Update{{
				Path: mustPath("/outer/inner"),
				Val: &gpb.TypedValue{Value: &gpb.TypedValue_JsonIetfVal{
					JsonIetfVal: []byte(`
{
	"int32-leaf-list": [40]
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
					Int32LeafListName: []int32{40},
					StringLeafName:    ygot.String("foo"),
				},
			},
		},
		wantNodeCacheStore: map[string]*cachedNodeInfo{
			`{"name":"key1"}`: {
				nodes: []*TreeNode{{
					Data: func(s string) *string { return &s }("hello"),
					Path: &gnmi.Path{
						Elem: []*gnmi.PathElem{{Name: "key1"}},
					},
				}},
			},
			`{"name":"outer"},{"name":"inner"}`: {
				nodes: []*TreeNode{
					{
						Data: &InnerContainerType1{
							Int32LeafListName: []int32{40},
							StringLeafName:    func(s string) *string { return &s }("foo"),
						},
						Path: &gnmi.Path{
							Elem: []*gnmi.PathElem{
								{Name: "outer"},
								{Name: "inner"},
							},
						},
					},
				},
			},
			`{"name":"outer"},{"name":"inner"},{"name":"string-leaf-field"}`: {
				nodes: []*TreeNode{
					{
						Data: func(s string) *string { return &s }("foo"),
						Path: &gnmi.Path{
							Elem: []*gnmi.PathElem{
								{Name: "outer"},
								{Name: "inner"},
								{Name: "string-leaf-field"},
							},
						},
					},
				},
			},
		},
	}, {
		desc:     "deletes to a non-empty struct",
		inSchema: inSchema,
		inReq: &gpb.SetRequest{
			Prefix: &gpb.Path{},
			Delete: []*gpb.Path{
				mustPath("/outer"),
			},
		},
		want: &ListElemStruct1{
			Key1: ygot.String("hello"),
		},
		wantNodeCacheStore: map[string]*cachedNodeInfo{
			`{"name":"key1"}`: {
				nodes: []*TreeNode{{
					Data: func(s string) *string { return &s }("hello"),
					Path: &gnmi.Path{
						Elem: []*gnmi.PathElem{{Name: "key1"}},
					},
				}},
			},
		},
	}, {
		desc:     "deletes, replaces and update to a non-empty struct",
		inSchema: inSchema,
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
				Path: mustPath("/outer/inner/config/int32-leaf-field"),
				Val: &gpb.TypedValue{Value: &gpb.TypedValue_IntVal{
					IntVal: 42,
				}},
			}},
		},
		want: &ListElemStruct1{
			Key1: ygot.String("world"),
			Outer: &OuterContainerType1{
				Inner: &InnerContainerType1{
					Int32LeafName: ygot.Int32(42),
				},
			},
		},
		wantNodeCacheStore: map[string]*cachedNodeInfo{
			`{"name":"key1"}`: {
				nodes: []*TreeNode{{
					Data: func(s string) *string { return &s }("world"),
					Path: &gnmi.Path{
						Elem: []*gnmi.PathElem{{Name: "key1"}},
					},
				}},
			},
			`{"name":"outer"},{"name":"inner"},{"name":"config"},{"name":"int32-leaf-field"}`: {
				nodes: []*TreeNode{
					{
						Data: func(val int32) *int32 { return &val }(42),
						Path: &gnmi.Path{
							Elem: []*gnmi.PathElem{
								{Name: "outer"},
								{Name: "inner"},
								{Name: "config"},
								{Name: "int32-leaf-field"},
							},
						},
					},
				},
			},
		},
	}, {
		desc:     "deletes and update to a non-empty struct with preferShadowPath (no effect)",
		inSchema: inSchema,
		inReq: &gpb.SetRequest{
			Prefix: &gpb.Path{},
			Delete: []*gpb.Path{
				mustPath("/outer/inner/config/int32-leaf-field"),
			},
		},
		inUnmarshalOpts: []UnmarshalOpt{&PreferShadowPath{}},
		want: &ListElemStruct1{
			Key1: ygot.String("world"),
			Outer: &OuterContainerType1{
				Inner: &InnerContainerType1{
					Int32LeafName: ygot.Int32(42),
				},
			},
		},
		wantNodeCacheStore: map[string]*cachedNodeInfo{
			`{"name":"key1"}`: {
				nodes: []*TreeNode{{
					Data: func(s string) *string { return &s }("world"),
					Path: &gnmi.Path{
						Elem: []*gnmi.PathElem{{Name: "key1"}},
					},
				}},
			},
			`{"name":"outer"},{"name":"inner"},{"name":"config"},{"name":"int32-leaf-field"}`: {
				nodes: []*TreeNode{
					{
						Data: func(val int32) *int32 { return &val }(42),
						Path: &gnmi.Path{
							Elem: []*gnmi.PathElem{
								{Name: "outer"},
								{Name: "inner"},
								{Name: "config"},
								{Name: "int32-leaf-field"},
							},
						},
					},
				},
			},
		},
	}, {
		desc:     "updates to a non-empty struct",
		inSchema: inSchema,
		inReq: &gpb.SetRequest{
			Prefix: &gpb.Path{},
			Update: []*gpb.Update{{
				Path: mustPath("/outer/inner"),
				Val: &gpb.TypedValue{Value: &gpb.TypedValue_JsonIetfVal{
					JsonIetfVal: []byte(`
{
	"int32-leaf-list": [40]
}
					`),
				}},
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
					Int32LeafName:     ygot.Int32(42),
					Int32LeafListName: []int32{40},
					StringLeafName:    ygot.String("foo"),
				},
			},
		},
		wantNodeCacheStore: map[string]*cachedNodeInfo{
			`{"name":"key1"}`: {
				nodes: []*TreeNode{{
					Data: func(s string) *string { return &s }("world"),
					Path: &gnmi.Path{
						Elem: []*gnmi.PathElem{{Name: "key1"}},
					},
				}},
			},
			`{"name":"outer"},{"name":"inner"}`: {
				nodes: []*TreeNode{
					{
						Data: &InnerContainerType1{
							Int32LeafName:     func(val int32) *int32 { return &val }(42),
							Int32LeafListName: []int32{40},
							StringLeafName:    func(s string) *string { return &s }("foo"),
						},
						Path: &gnmi.Path{
							Elem: []*gnmi.PathElem{
								{Name: "outer"},
								{Name: "inner"},
							},
						},
					},
				},
			},
			`{"name":"outer"},{"name":"inner"},{"name":"config"},{"name":"int32-leaf-field"}`: {
				nodes: []*TreeNode{
					{
						Data: func(val int32) *int32 { return &val }(42),
						Path: &gnmi.Path{
							Elem: []*gnmi.PathElem{
								{Name: "outer"},
								{Name: "inner"},
								{Name: "config"},
								{Name: "int32-leaf-field"},
							},
						},
					},
				},
			},
			`{"name":"outer"},{"name":"inner"},{"name":"string-leaf-field"}`: {
				nodes: []*TreeNode{
					{
						Data: func(s string) *string { return &s }("foo"),
						Path: &gnmi.Path{
							Elem: []*gnmi.PathElem{
								{Name: "outer"},
								{Name: "inner"},
								{Name: "string-leaf-field"},
							},
						},
					},
				},
			},
		},
	}, {
		desc:     "deletes from a non-empty struct",
		inSchema: inSchema,
		inReq: &gpb.SetRequest{
			Prefix: &gpb.Path{},
			Delete: []*gnmi.Path{mustPath("/outer/inner/config/int32-leaf-field")},
		},
		want: &ListElemStruct1{
			Key1: ygot.String("world"),
			Outer: &OuterContainerType1{
				Inner: &InnerContainerType1{
					Int32LeafListName: []int32{40},
					StringLeafName:    ygot.String("foo"),
				},
			},
		},
		wantNodeCacheStore: map[string]*cachedNodeInfo{
			`{"name":"key1"}`: {
				nodes: []*TreeNode{{
					Data: func(s string) *string { return &s }("world"),
					Path: &gnmi.Path{
						Elem: []*gnmi.PathElem{{Name: "key1"}},
					},
				}},
			},
			`{"name":"outer"},{"name":"inner"}`: {
				nodes: []*TreeNode{
					{
						Data: &InnerContainerType1{
							Int32LeafListName: []int32{40},
							StringLeafName:    func(s string) *string { return &s }("foo"),
						},
						Path: &gnmi.Path{
							Elem: []*gnmi.PathElem{
								{Name: "outer"},
								{Name: "inner"},
							},
						},
					},
				},
			},
			`{"name":"outer"},{"name":"inner"},{"name":"string-leaf-field"}`: {
				nodes: []*TreeNode{
					{
						Data: func(s string) *string { return &s }("foo"),
						Path: &gnmi.Path{
							Elem: []*gnmi.PathElem{
								{Name: "outer"},
								{Name: "inner"},
								{Name: "string-leaf-field"},
							},
						},
					},
				},
			},
		},
	}, {
		desc:     "replaces to a non-empty struct with prefix",
		inSchema: inSchema,
		inReq: &gpb.SetRequest{
			Prefix: mustPath("/outer"),
			Replace: []*gpb.Update{{
				Path: mustPath("inner/string-leaf-field"),
				Val: &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{
					StringVal: "bar",
				}},
			}},
		},
		want: &ListElemStruct1{
			Key1: ygot.String("world"),
			Outer: &OuterContainerType1{
				Inner: &InnerContainerType1{
					Int32LeafListName: []int32{40},
					StringLeafName:    ygot.String("bar"),
				},
			},
		},
		wantNodeCacheStore: map[string]*cachedNodeInfo{
			`{"name":"key1"}`: {
				nodes: []*TreeNode{{
					Data: func(s string) *string { return &s }("world"),
					Path: &gnmi.Path{
						Elem: []*gnmi.PathElem{{Name: "key1"}},
					},
				}},
			},
			`{"name":"outer"}`: {
				nodes: []*TreeNode{{
					Data: &OuterContainerType1{
						Inner: &InnerContainerType1{
							Int32LeafListName: []int32{40},
							StringLeafName:    func(s string) *string { return &s }("bar"),
						},
					},
					Path: &gnmi.Path{
						Elem: []*gnmi.PathElem{{Name: "outer"}},
					},
				}},
			},
			`{"name":"outer"},{"name":"inner"}`: {
				nodes: []*TreeNode{
					{
						Data: &InnerContainerType1{
							Int32LeafListName: []int32{40},
							StringLeafName:    func(s string) *string { return &s }("bar"),
						},
						Path: &gnmi.Path{
							Elem: []*gnmi.PathElem{
								{Name: "outer"},
								{Name: "inner"},
							},
						},
					},
				},
			},
			`{"name":"inner"},{"name":"string-leaf-field"}`: {
				nodes: []*TreeNode{
					{
						Data: func(s string) *string { return &s }("bar"),
						Path: &gnmi.Path{
							Elem: []*gnmi.PathElem{
								{Name: "inner"},
								{Name: "string-leaf-field"},
							},
						},
					},
				},
			},
		},
	}, {
		desc:     "replaces to a non-existent path",
		inSchema: inSchema,
		inReq: &gpb.SetRequest{
			Prefix: mustPath("/outer-planets"),
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
		wantErr: true,
	}, {
		desc:     "delete string-leaf-field",
		inSchema: inSchema,
		inReq: &gpb.SetRequest{
			Prefix: &gpb.Path{},
			Delete: []*gnmi.Path{mustPath("/outer/inner/string-leaf-field")},
		},
		want: &ListElemStruct1{
			Key1: ygot.String("world"),
			Outer: &OuterContainerType1{
				Inner: &InnerContainerType1{
					Int32LeafListName: []int32{40},
				},
			},
		},
		wantNodeCacheStore: map[string]*cachedNodeInfo{
			`{"name":"key1"}`: {
				nodes: []*TreeNode{{
					Data: func(s string) *string { return &s }("world"),
					Path: &gnmi.Path{
						Elem: []*gnmi.PathElem{{Name: "key1"}},
					},
				}},
			},
			`{"name":"outer"}`: {
				nodes: []*TreeNode{{
					Data: &OuterContainerType1{
						Inner: &InnerContainerType1{
							Int32LeafListName: []int32{40},
						},
					},
					Path: &gnmi.Path{
						Elem: []*gnmi.PathElem{{Name: "outer"}},
					},
				}},
			},
			`{"name":"outer"},{"name":"inner"}`: {
				nodes: []*TreeNode{
					{
						Data: &InnerContainerType1{
							Int32LeafListName: []int32{40},
						},
						Path: &gnmi.Path{
							Elem: []*gnmi.PathElem{
								{Name: "outer"},
								{Name: "inner"},
							},
						},
					},
				},
			},
		},
	}}

	// Instantiate node cache.
	nodeCache := NewNodeCache()

	// Note: these test cases should not be running in parallel because of sequential
	// dependencies on working with the same node cache.
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := UnmarshalSetRequest(
				tt.inSchema,
				tt.inReq,
				append(tt.inUnmarshalOpts, &NodeCacheOpt{NodeCache: nodeCache})...,
			)
			if gotErr := err != nil; gotErr != tt.wantErr {
				t.Fatalf("got error: %v, want: %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				if diff := cmp.Diff(tt.inSchema.Root, tt.want); diff != "" {
					t.Errorf("(-got, +want):\n%s", diff)
				}

				// Check the data in the node cache.
				if len(nodeCache.store) != len(tt.wantNodeCacheStore) {
					t.Errorf("wanted node cache store size %d (%v), got %d (%v)", len(tt.wantNodeCacheStore), tt.wantNodeCacheStore, len(nodeCache.store), nodeCache.store)
					return
				}

				for key, info := range tt.wantNodeCacheStore {
					if infoGot, ok := nodeCache.store[key]; !ok {
						t.Errorf("missing expected key `%s` in the node cache store (%v)", key, nodeCache.store)
						continue
					} else {
						for i := 0; i < len(info.nodes); i++ {
							if diff := cmp.Diff(infoGot.nodes[i].Data, info.nodes[i].Data); diff != "" {
								t.Errorf("key %s: (-got, +want):\n%s", key, diff)
							}

							if diff := cmp.Diff(infoGot.nodes[i].Path.String(), info.nodes[i].Path.String()); diff != "" {
								t.Errorf("key %s: (-got, +want):\n%s", key, diff)
							}
						}
					}
				}
			}
		})
	}
}

func TestUnmarshalNotifications(t *testing.T) {
	tests := []struct {
		desc            string
		inSchema        *Schema
		inNotifications []*gpb.Notification
		inUnmarshalOpts []UnmarshalOpt
		want            ygot.GoStruct
		wantErr         bool
	}{{
		desc: "updates to an empty struct",
		inSchema: &Schema{
			Root: &ListElemStruct1{},
			SchemaTree: map[string]*yang.Entry{
				"ListElemStruct1": simpleSchema(),
			},
		},
		inNotifications: []*gpb.Notification{{
			Prefix: &gpb.Path{},
			Update: []*gpb.Update{{
				Path: mustPath("/key1"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "invalid"}},
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
		}},
		want: &ListElemStruct1{
			Key1: ygot.String("invalid"),
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
		inNotifications: []*gpb.Notification{{
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
		}},
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
		desc: "fail: update to invalid field",
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
		inNotifications: []*gpb.Notification{{
			Prefix: &gpb.Path{},
			Update: []*gpb.Update{{
				Path: mustPath("/non-existent"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "hello"}},
			}},
		}},
		wantErr: true,
	}, {
		desc: "OK: update to invalid field with IgnoreExtraFields",
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
		inNotifications: []*gpb.Notification{{
			Prefix: &gpb.Path{},
			Update: []*gpb.Update{{
				Path: mustPath("/non-existent"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "hello"}},
			}},
		}},
		inUnmarshalOpts: []UnmarshalOpt{&IgnoreExtraFields{}},
		want: &ListElemStruct1{
			Key1: ygot.String("hello"),
			Outer: &OuterContainerType1{
				Inner: &InnerContainerType1{
					Int32LeafName:     ygot.Int32(43),
					Int32LeafListName: []int32{100},
					StringLeafName:    ygot.String("bear"),
				},
			},
		},
	}, {
		desc: "delete to a non-empty struct",
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
		inNotifications: []*gpb.Notification{{
			Prefix: &gpb.Path{},
			Delete: []*gpb.Path{
				mustPath("/outer/inner/config/int32-leaf-field"),
			},
		}},
		want: &ListElemStruct1{
			Key1: ygot.String("hello"),
			Outer: &OuterContainerType1{
				Inner: &InnerContainerType1{
					Int32LeafListName: []int32{42},
					StringLeafName:    ygot.String("bear"),
				},
			},
		},
	}, {
		desc: "delete to a non-empty struct with preferShadowPath (no effect)",
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
		inNotifications: []*gpb.Notification{{
			Prefix: &gpb.Path{},
			Delete: []*gpb.Path{
				mustPath("/outer/inner/config/int32-leaf-field"),
			},
		}},
		inUnmarshalOpts: []UnmarshalOpt{&PreferShadowPath{}},
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
		desc: "deletes and updates to a non-empty struct in multiple notifications",
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
				"InnerContainerType1": simpleSchema().Dir["outer"].Dir["config"].Dir["inner"],
			},
		},
		inNotifications: []*gpb.Notification{{
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
		}, {
			Prefix: mustPath("/outer/inner"),
			Delete: []*gpb.Path{
				mustPath("string-leaf-field"),
			},
			Update: []*gpb.Update{{
				Path: mustPath("config/int32-leaf-field"),
				Val: &gpb.TypedValue{Value: &gpb.TypedValue_IntVal{
					IntVal: 42,
				}},
			}},
		}},
		want: &ListElemStruct1{
			Key1: ygot.String("world"),
			Outer: &OuterContainerType1{
				Inner: &InnerContainerType1{
					Int32LeafName: ygot.Int32(42),
				},
			},
		},
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := UnmarshalNotifications(tt.inSchema, tt.inNotifications, tt.inUnmarshalOpts...)
			if gotErr := err != nil; gotErr != tt.wantErr {
				t.Fatalf("got error: %v, want: %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				if diff := cmp.Diff(tt.inSchema.Root, tt.want); diff != "" {
					t.Errorf("(-got, +want):\n%s", diff)
				}
			}
		})
	}
}
