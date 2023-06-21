// Copyright 2023 Google Inc.
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
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygot"

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
		numErrs int
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
		desc: "updates to a struct containing a non-empty list",
		inSchema: &Schema{
			Root: &ContainerStruct1{
				StructKeyList: map[string]*ListElemStruct1{
					"forty-two": {
						Key1: ygot.String("forty-two"),
						Outer: &OuterContainerType1{
							Inner: &InnerContainerType1{
								Int32LeafName:     ygot.Int32(43),
								Int32LeafListName: []int32{100},
								StringLeafName:    ygot.String("bear"),
							},
						},
					},
					"forty-three": {
						Key1: ygot.String("forty-three"),
						Outer: &OuterContainerType1{
							Inner: &InnerContainerType1{
								Int32LeafName:     ygot.Int32(43),
								Int32LeafListName: []int32{100},
								StringLeafName:    ygot.String("bear"),
							},
						},
					},
				},
			},
			SchemaTree: map[string]*yang.Entry{
				"ContainerStruct1": containerWithStringKey(),
			},
		},
		inReq: &gpb.SetRequest{
			Prefix: &gpb.Path{},
			Update: []*gpb.Update{{
				Path: mustPath("/"),
				Val: &gpb.TypedValue{Value: &gpb.TypedValue_JsonIetfVal{
					JsonIetfVal: []byte(`
{
	"config": {
		"simple-key-list": [
			{
				"key1": "forty-two",
				"outer": {
					"inner": {
						"int32-leaf-list": [42]
					}
				}
			}
		]
	}
}
					`),
				}},
			}},
		},
		want: &ContainerStruct1{
			StructKeyList: map[string]*ListElemStruct1{
				"forty-two": {
					Key1: ygot.String("forty-two"),
					Outer: &OuterContainerType1{
						Inner: &InnerContainerType1{
							Int32LeafName:     ygot.Int32(43),
							Int32LeafListName: []int32{42},
							StringLeafName:    ygot.String("bear"),
						},
					},
				},
				"forty-three": {
					Key1: ygot.String("forty-three"),
					Outer: &OuterContainerType1{
						Inner: &InnerContainerType1{
							Int32LeafName:     ygot.Int32(43),
							Int32LeafListName: []int32{100},
							StringLeafName:    ygot.String("bear"),
						},
					},
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
	}, {
		desc: "mix of an error and a non-error update with best-effort flag",
		inSchema: &Schema{
			Root: &ListElemStruct1{
				Key1: ygot.String("mixedupdate"),
				Outer: &OuterContainerType1{
					Inner: &InnerContainerType1{
						Int32LeafName: ygot.Int32(43),
						Int32LeafListName: []int32{100},
						StringLeafName: ygot.String("bear"),
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
				Val: &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "non-error"}},
			}, {
				Path: mustPath("/outer/error"),
				Val: &gpb.TypedValue{Value: &gpb.TypedValue_JsonIetfVal{
					JsonIetfVal: []byte(`
{
	"int32-leaf-list": [42]
}
					`),
				}},
			}, {
				Path: mustPath("/outer/error2"),
				Val: &gpb.TypedValue{Value: &gpb.TypedValue_JsonIetfVal{
					JsonIetfVal: []byte(`
{
	"int32-leaf-list": [42]
}
					`),
				}},
			}},
		},
		inUnmarshalOpts: []UnmarshalOpt{&BestEffortUnmarshal{}},
		want: &ListElemStruct1{
			Key1: ygot.String("non-error"),
			Outer: &OuterContainerType1{
				Inner: &InnerContainerType1{
					Int32LeafName: ygot.Int32(43),
					Int32LeafListName: []int32{100},
					StringLeafName: ygot.String("bear"),
				},
			},
		},
		wantErr: true,
		numErrs: 2,
	}, {
		desc: "mix of an error and a non-error replace with best-effort flag",
		inSchema: &Schema{
			Root: &ListElemStruct1{
				Key1: ygot.String("mixedreplace"),
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
			Replace: []*gpb.Update{{
				Path: mustPath("/key1"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "success"}},
			}, {
				Path: mustPath("/key2"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "failure"}},
			}, {
				Path: mustPath("/key3"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "failure"}},
			}},
		},
		inUnmarshalOpts: []UnmarshalOpt{&BestEffortUnmarshal{}},
		want: &ListElemStruct1{
			Key1: ygot.String("success"),
			Outer: &OuterContainerType1{
				Inner: &InnerContainerType1{
					Int32LeafName:     ygot.Int32(43),
					Int32LeafListName: []int32{100},
					StringLeafName:    ygot.String("bear"),
				},
			},
		},
		wantErr: true,
		numErrs: 2,
	}, {
		desc: "mix of an error and a non-error delete with best-effort flag",
		inSchema: &Schema{
			Root: &ListElemStruct1{
				Key1: ygot.String("mixeddelete"),
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
			Delete: []*gpb.Path{
				mustPath("/outer/inner/config/int32-leaf-field"),
				mustPath("/outer/error"),
				mustPath("/outer/error2"),
			},
		},
		inUnmarshalOpts: []UnmarshalOpt{&BestEffortUnmarshal{}},
		want: &ListElemStruct1{
			Key1: ygot.String("mixeddelete"),
			Outer: &OuterContainerType1{
				Inner: &InnerContainerType1{
					Int32LeafListName: []int32{100},
					StringLeafName:    ygot.String("bear"),
				},
			},
		},
		wantErr: true,
		numErrs: 2,
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := UnmarshalSetRequest(tt.inSchema, tt.inReq, tt.inUnmarshalOpts...)
			if gotErr := err != nil; gotErr != tt.wantErr {
				t.Fatalf("got error: %v, want: %v", err, tt.wantErr)
			}
			if gotErr := err != nil; gotErr && hasBestEffortUnmarshal(tt.inUnmarshalOpts) {
				var ce *ComplianceErrors
				if errors.As(err, &ce) {
					if len(ce.Errors) != tt.numErrs {
						t.Fatalf("Got the incorrect number of errors: want %v, got %v", tt.numErrs, len(ce.Errors))
					}
				} else {
					t.Fatalf("Error casting BestEffortUnmarshal result to compliance errors struct")
				}
			}
			if !tt.wantErr {
				if diff := cmp.Diff(tt.inSchema.Root, tt.want); diff != "" {
					t.Errorf("(-got, +want):\n%s", diff)
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
