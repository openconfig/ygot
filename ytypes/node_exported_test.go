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

package ytypes_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/openconfig/gnmi/errdiff"
	gpb "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/integration_tests/schemaops/ctestschema"
	"github.com/openconfig/ygot/integration_tests/schemaops/utestschema"
	"github.com/openconfig/ygot/internal/ytestutil"
	"github.com/openconfig/ygot/ygot"
	"github.com/openconfig/ygot/ytypes"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
)

func treeNodesEqual(got, want []*ytypes.TreeNode) error {
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

func mustPath(s string) *gpb.Path {
	p, err := ygot.StringToStructuredPath(s)
	if err != nil {
		panic(err)
	}
	return p
}

func TestGetNodeOrderedMap(t *testing.T) {
	tests := []struct {
		desc             string
		inSchema         *yang.Entry
		inParent         any
		inPath           *gpb.Path
		inArgs           []ytypes.GetNodeOpt
		wantTreeNodes    []*ytypes.TreeNode
		wantErrSubstring string
	}{{
		desc:     "single-keyed ordered list",
		inSchema: ctestschema.SchemaTree["Device"],
		inParent: &ctestschema.Device{
			OrderedList: ctestschema.GetOrderedMap(t),
		},
		inPath: mustPath("/ordered-lists/ordered-list[key=foo]"),
		wantTreeNodes: []*ytypes.TreeNode{{
			Data: &ctestschema.OrderedList{
				Key:   ygot.String("foo"),
				Value: ygot.String("foo-val"),
			},
			Schema: ctestschema.SchemaTree["OrderedList"],
			Path:   mustPath("/ordered-lists/ordered-list[key=foo]"),
		}},
	}, {
		desc:     "single-keyed ordered list uncompressed",
		inSchema: utestschema.SchemaTree["Device"],
		inParent: utestschema.GetDeviceWithOrderedMap(t),
		inPath:   mustPath("/ordered-lists/ordered-list[key=foo]"),
		wantTreeNodes: []*ytypes.TreeNode{{
			Data: &utestschema.Ctestschema_OrderedLists_OrderedList{
				Key: ygot.String("foo"),
				Config: &utestschema.Ctestschema_OrderedLists_OrderedList_Config{
					Value: ygot.String("foo-val"),
				},
			},
			Schema: utestschema.SchemaTree["Ctestschema_OrderedLists_OrderedList"],
			Path:   mustPath("/ordered-lists/ordered-list[key=foo]"),
		}},
	}, {
		desc:     "single-keyed ordered list that doesn't match anything",
		inSchema: ctestschema.SchemaTree["Device"],
		inParent: &ctestschema.Device{
			OrderedList: ctestschema.GetOrderedMap(t),
		},
		inPath:        mustPath("/ordered-lists/ordered-list[key=boo]"),
		wantTreeNodes: []*ytypes.TreeNode{},
	}, {
		desc:     "single-keyed ordered list match on second",
		inSchema: ctestschema.SchemaTree["Device"],
		inParent: &ctestschema.Device{
			OrderedList: ctestschema.GetOrderedMap(t),
		},
		inPath: mustPath("/ordered-lists/ordered-list[key=bar]"),
		wantTreeNodes: []*ytypes.TreeNode{{
			Data: &ctestschema.OrderedList{
				Key:   ygot.String("bar"),
				Value: ygot.String("bar-val"),
			},
			Schema: ctestschema.SchemaTree["OrderedList"],
			Path:   mustPath("/ordered-lists/ordered-list[key=bar]"),
		}},
	}, {
		desc:     "multi-keyed ordered list",
		inSchema: ctestschema.SchemaTree["Device"],
		inParent: &ctestschema.Device{
			OrderedMultikeyedList: ctestschema.GetOrderedMapMultikeyed(t),
		},
		inPath: mustPath("/ordered-multikeyed-lists/ordered-multikeyed-list[key1=foo][key2=42]"),
		wantTreeNodes: []*ytypes.TreeNode{{
			Data: &ctestschema.OrderedMultikeyedList{
				Key1:  ygot.String("foo"),
				Key2:  ygot.Uint64(42),
				Value: ygot.String("foo-val"),
			},
			Schema: ctestschema.SchemaTree["OrderedMultikeyedList"],
			Path:   mustPath("/ordered-multikeyed-lists/ordered-multikeyed-list[key1=foo][key2=42]"),
		}},
	}, {
		desc:     "multi-keyed ordered list match on third",
		inSchema: ctestschema.SchemaTree["Device"],
		inParent: &ctestschema.Device{
			OrderedMultikeyedList: ctestschema.GetOrderedMapMultikeyed(t),
		},
		inPath: mustPath("/ordered-multikeyed-lists/ordered-multikeyed-list[key1=baz][key2=84]"),
		wantTreeNodes: []*ytypes.TreeNode{{
			Data: &ctestschema.OrderedMultikeyedList{
				Key1:  ygot.String("baz"),
				Key2:  ygot.Uint64(84),
				Value: ygot.String("baz-val"),
			},
			Schema: ctestschema.SchemaTree["OrderedMultikeyedList"],
			Path:   mustPath("/ordered-multikeyed-lists/ordered-multikeyed-list[key1=baz][key2=84]"),
		}},
	}, {
		desc:     "nested ordered list",
		inSchema: ctestschema.SchemaTree["Device"],
		inParent: &ctestschema.Device{
			OrderedList: ctestschema.GetNestedOrderedMap(t),
		},
		inPath: mustPath("/ordered-lists/ordered-list[key=foo]/ordered-lists/ordered-list[key=foo]"),
		wantTreeNodes: []*ytypes.TreeNode{{
			Data: &ctestschema.OrderedList_OrderedList{
				Key:   ygot.String("foo"),
				Value: ygot.String("foo-val"),
			},
			Schema: ctestschema.SchemaTree["OrderedList_OrderedList"],
			Path:   mustPath("/ordered-lists/ordered-list[key=foo]/ordered-lists/ordered-list[key=foo]"),
		}},
	}, {
		desc:     "wildcard match on single-keyed ordered list",
		inSchema: ctestschema.SchemaTree["Device"],
		inParent: &ctestschema.Device{
			OrderedList: ctestschema.GetOrderedMap(t),
		},
		inPath: mustPath("/ordered-lists/ordered-list[key=*]"),
		inArgs: []ytypes.GetNodeOpt{&ytypes.GetHandleWildcards{}},
		wantTreeNodes: []*ytypes.TreeNode{{
			Data: &ctestschema.OrderedList{
				Key:   ygot.String("foo"),
				Value: ygot.String("foo-val"),
			},
			Schema: ctestschema.SchemaTree["OrderedList"],
			Path:   mustPath("/ordered-lists/ordered-list[key=foo]"),
		}, {
			Data: &ctestschema.OrderedList{
				Key:   ygot.String("bar"),
				Value: ygot.String("bar-val"),
			},
			Schema: ctestschema.SchemaTree["OrderedList"],
			Path:   mustPath("/ordered-lists/ordered-list[key=bar]"),
		}},
	}, {
		desc:     "partial match on single-keyed ordered list",
		inSchema: ctestschema.SchemaTree["Device"],
		inParent: &ctestschema.Device{
			OrderedList: ctestschema.GetOrderedMap(t),
		},
		inPath: mustPath("/ordered-lists/ordered-list"),
		inArgs: []ytypes.GetNodeOpt{&ytypes.GetPartialKeyMatch{}},
		wantTreeNodes: []*ytypes.TreeNode{{
			Data: &ctestschema.OrderedList{
				Key:   ygot.String("foo"),
				Value: ygot.String("foo-val"),
			},
			Schema: ctestschema.SchemaTree["OrderedList"],
			Path:   mustPath("/ordered-lists/ordered-list[key=foo]"),
		}, {
			Data: &ctestschema.OrderedList{
				Key:   ygot.String("bar"),
				Value: ygot.String("bar-val"),
			},
			Schema: ctestschema.SchemaTree["OrderedList"],
			Path:   mustPath("/ordered-lists/ordered-list[key=bar]"),
		}},
	}, {
		desc:     "wildcard match on multi-keyed ordered list",
		inSchema: ctestschema.SchemaTree["Device"],
		inParent: &ctestschema.Device{
			OrderedMultikeyedList: ctestschema.GetOrderedMapMultikeyed(t),
		},
		inPath: mustPath("/ordered-multikeyed-lists/ordered-multikeyed-list[key1=*][key2=42]"),
		inArgs: []ytypes.GetNodeOpt{&ytypes.GetHandleWildcards{}},
		wantTreeNodes: []*ytypes.TreeNode{{
			Data: &ctestschema.OrderedMultikeyedList{
				Key1:  ygot.String("foo"),
				Key2:  ygot.Uint64(42),
				Value: ygot.String("foo-val"),
			},
			Schema: ctestschema.SchemaTree["OrderedMultikeyedList"],
			Path:   mustPath("/ordered-multikeyed-lists/ordered-multikeyed-list[key1=foo][key2=42]"),
		}, {
			Data: &ctestschema.OrderedMultikeyedList{
				Key1:  ygot.String("bar"),
				Key2:  ygot.Uint64(42),
				Value: ygot.String("bar-val"),
			},
			Schema: ctestschema.SchemaTree["OrderedMultikeyedList"],
			Path:   mustPath("/ordered-multikeyed-lists/ordered-multikeyed-list[key1=bar][key2=42]"),
		}},
	}, {
		desc:     "partial match on multi-keyed ordered list",
		inSchema: ctestschema.SchemaTree["Device"],
		inParent: &ctestschema.Device{
			OrderedMultikeyedList: ctestschema.GetOrderedMapMultikeyed(t),
		},
		inPath: mustPath("/ordered-multikeyed-lists/ordered-multikeyed-list[key2=42]"),
		inArgs: []ytypes.GetNodeOpt{&ytypes.GetPartialKeyMatch{}},
		wantTreeNodes: []*ytypes.TreeNode{{
			Data: &ctestschema.OrderedMultikeyedList{
				Key1:  ygot.String("foo"),
				Key2:  ygot.Uint64(42),
				Value: ygot.String("foo-val"),
			},
			Schema: ctestschema.SchemaTree["OrderedMultikeyedList"],
			Path:   mustPath("/ordered-multikeyed-lists/ordered-multikeyed-list[key1=foo][key2=42]"),
		}, {
			Data: &ctestschema.OrderedMultikeyedList{
				Key1:  ygot.String("bar"),
				Key2:  ygot.Uint64(42),
				Value: ygot.String("bar-val"),
			},
			Schema: ctestschema.SchemaTree["OrderedMultikeyedList"],
			Path:   mustPath("/ordered-multikeyed-lists/ordered-multikeyed-list[key1=bar][key2=42]"),
		}},
	}, {
		desc:     "wildcard match on nested ordered list",
		inSchema: ctestschema.SchemaTree["Device"],
		inParent: &ctestschema.Device{
			OrderedList: ctestschema.GetNestedOrderedMap(t),
		},
		inPath: mustPath("/ordered-lists/ordered-list[key=foo]/ordered-lists/ordered-list[key=*]"),
		inArgs: []ytypes.GetNodeOpt{&ytypes.GetHandleWildcards{}},
		wantTreeNodes: []*ytypes.TreeNode{{
			Data: &ctestschema.OrderedList_OrderedList{
				Key:   ygot.String("foo"),
				Value: ygot.String("foo-val"),
			},
			Schema: ctestschema.SchemaTree["OrderedList_OrderedList"],
			Path:   mustPath("/ordered-lists/ordered-list[key=foo]/ordered-lists/ordered-list[key=foo]"),
		}, {
			Data: &ctestschema.OrderedList_OrderedList{
				Key:   ygot.String("bar"),
				Value: ygot.String("bar-val"),
			},
			Schema: ctestschema.SchemaTree["OrderedList_OrderedList"],
			Path:   mustPath("/ordered-lists/ordered-list[key=foo]/ordered-lists/ordered-list[key=bar]"),
		}},
	}, {
		desc:     "value not found through single-keyed ordered list",
		inSchema: ctestschema.SchemaTree["Device"],
		inParent: &ctestschema.Device{
			OrderedList: ctestschema.GetOrderedMap(t),
		},
		inPath:           mustPath("/ordered-lists/ordered-list[key=foo]/config/does-not-exist"),
		wantErrSubstring: "no match found",
	}, {
		desc:     "value through single-keyed ordered list",
		inSchema: ctestschema.SchemaTree["Device"],
		inParent: &ctestschema.Device{
			OrderedList: ctestschema.GetOrderedMap(t),
		},
		inPath: mustPath("/ordered-lists/ordered-list[key=foo]/config/value"),
		wantTreeNodes: []*ytypes.TreeNode{{
			Data:   ygot.String("foo-val"),
			Schema: ctestschema.SchemaTree["OrderedList"].Dir["config"].Dir["value"],
			Path:   mustPath("/ordered-lists/ordered-list[key=foo]/config/value"),
		}},
	}, {
		desc:     "value not preferred through single-keyed ordered list",
		inSchema: ctestschema.SchemaTree["Device"],
		inParent: &ctestschema.Device{
			OrderedList: ctestschema.GetOrderedMap(t),
		},
		inPath: mustPath("/ordered-lists/ordered-list[key=foo]/config/value"),
		inArgs: []ytypes.GetNodeOpt{&ytypes.PreferShadowPath{}},
		wantTreeNodes: []*ytypes.TreeNode{{
			Data:   nil,
			Schema: nil,
			Path:   mustPath("/ordered-lists/ordered-list[key=foo]/config/value"),
		}},
	}, {
		desc:     "shadow-path value through single-keyed ordered list",
		inSchema: ctestschema.SchemaTree["Device"],
		inParent: &ctestschema.Device{
			OrderedList: ctestschema.GetOrderedMap(t),
		},
		inPath: mustPath("/ordered-lists/ordered-list[key=foo]/state/value"),
		wantTreeNodes: []*ytypes.TreeNode{{
			Data:   nil,
			Schema: nil,
			Path:   mustPath("/ordered-lists/ordered-list[key=foo]/state/value"),
		}},
	}, {
		desc:     "shadow-path value preferred through single-keyed ordered list",
		inSchema: ctestschema.SchemaTree["Device"],
		inParent: &ctestschema.Device{
			OrderedList: ctestschema.GetOrderedMap(t),
		},
		inPath: mustPath("/ordered-lists/ordered-list[key=foo]/state/value"),
		inArgs: []ytypes.GetNodeOpt{&ytypes.PreferShadowPath{}},
		wantTreeNodes: []*ytypes.TreeNode{{
			Data:   ygot.String("foo-val"),
			Schema: ctestschema.SchemaTree["OrderedList"].Dir["state"].Dir["value"],
			Path:   mustPath("/ordered-lists/ordered-list[key=foo]/state/value"),
		}},
	}, {
		desc:     "value through multi-keyed ordered list",
		inSchema: ctestschema.SchemaTree["Device"],
		inParent: &ctestschema.Device{
			OrderedMultikeyedList: ctestschema.GetOrderedMapMultikeyed(t),
		},
		inPath: mustPath("/ordered-multikeyed-lists/ordered-multikeyed-list[key1=foo][key2=42]/config/value"),
		wantTreeNodes: []*ytypes.TreeNode{{
			Data:   ygot.String("foo-val"),
			Schema: ctestschema.SchemaTree["OrderedMultikeyedList"].Dir["config"].Dir["value"],
			Path:   mustPath("/ordered-multikeyed-lists/ordered-multikeyed-list[key1=foo][key2=42]/config/value"),
		}},
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got, err := ytypes.GetNode(tt.inSchema, tt.inParent, tt.inPath, tt.inArgs...)
			if diff := errdiff.Substring(err, tt.wantErrSubstring); diff != "" {
				t.Fatalf("did not get expected error, %s", diff)
			}
			if err != nil {
				return
			}

			if err := treeNodesEqual(got, tt.wantTreeNodes); err != nil {
				if len(got) > 0 {
					fmt.Println("------------")
					fmt.Printf("%T: %v\n", got[0].Data, got[0].Data)
					fmt.Println(got[0].Schema.Path())
					fmt.Println(tt.wantTreeNodes[0].Schema.Path())
				}
				t.Fatalf("did not get expected result, %v", err)
			}
		})
	}
}

func TestGetOrCreateNodeOrderedMap(t *testing.T) {
	tests := []struct {
		desc             string
		inSchema         *yang.Entry
		inParent         any
		inPath           *gpb.Path
		inOpts           []ytypes.GetOrCreateNodeOpt
		want             any
		wantParent       any
		wantErrSubstring string
	}{{
		desc:     "single-keyed ordered list",
		inSchema: ctestschema.SchemaTree["Device"],
		inParent: &ctestschema.Device{},
		inPath:   mustPath("/ordered-lists/ordered-list[key=foo]"),
		want: &ctestschema.OrderedList{
			Key: ygot.String("foo"),
		},
		wantParent: &ctestschema.Device{
			OrderedList: func() *ctestschema.OrderedList_OrderedMap {
				orderedMap := &ctestschema.OrderedList_OrderedMap{}
				_, err := orderedMap.AppendNew("foo")
				if err != nil {
					t.Fatal(err)
				}
				return orderedMap
			}(),
		},
	}, {
		desc:     "single-keyed ordered list uncompressed",
		inSchema: utestschema.SchemaTree["Device"],
		inParent: &utestschema.Device{},
		inPath:   mustPath("/ordered-lists/ordered-list[key=foo]"),
		want: &utestschema.Ctestschema_OrderedLists_OrderedList{
			Key: ygot.String("foo"),
		},
		wantParent: &utestschema.Device{
			OrderedLists: &utestschema.Ctestschema_OrderedLists{
				OrderedList: func() *utestschema.Ctestschema_OrderedLists_OrderedList_OrderedMap {
					orderedMap := &utestschema.Ctestschema_OrderedLists_OrderedList_OrderedMap{}
					_, err := orderedMap.AppendNew("foo")
					if err != nil {
						t.Fatal(err)
					}
					return orderedMap
				}(),
			},
		},
	}, {
		desc:     "single-keyed ordered list leaf",
		inSchema: ctestschema.SchemaTree["Device"],
		inParent: &ctestschema.Device{},
		inPath:   mustPath("/ordered-lists/ordered-list[key=foo]/config/value"),
		want:     ygot.String(""),
		wantParent: &ctestschema.Device{
			OrderedList: func() *ctestschema.OrderedList_OrderedMap {
				orderedMap := &ctestschema.OrderedList_OrderedMap{}
				v, err := orderedMap.AppendNew("foo")
				if err != nil {
					t.Fatal(err)
				}
				v.Value = ygot.String("")
				return orderedMap
			}(),
		},
	}, {
		desc:     "single-keyed ordered list leaf already exists",
		inSchema: ctestschema.SchemaTree["Device"],
		inParent: &ctestschema.Device{
			OrderedList: ctestschema.GetOrderedMap(t),
		},
		inPath: mustPath("/ordered-lists/ordered-list[key=foo]/config/value"),
		want:   ygot.String("foo-val"),
		wantParent: &ctestschema.Device{
			OrderedList: ctestschema.GetOrderedMap(t),
		},
	}, {
		desc:             "single-keyed ordered list leaf without enough keys",
		inSchema:         ctestschema.SchemaTree["Device"],
		inParent:         &ctestschema.Device{},
		inPath:           mustPath("/ordered-lists/ordered-list/config/value"),
		wantErrSubstring: "got 0 valid keys, expected 1",
	}, {
		// TODO(wenbli): This is a bug: traversal should remember what
		// list entries it created so it can delete it when the
		// ultimate target is a shadow value.
		desc:     "single-keyed ordered list leaf shadow value",
		inSchema: ctestschema.SchemaTree["Device"],
		inParent: &ctestschema.Device{},
		inPath:   mustPath("/ordered-lists/ordered-list[key=foo]/state/value"),
		want:     nil,
		wantParent: &ctestschema.Device{
			OrderedList: func() *ctestschema.OrderedList_OrderedMap {
				orderedMap := &ctestschema.OrderedList_OrderedMap{}
				_, err := orderedMap.AppendNew("foo")
				if err != nil {
					t.Fatal(err)
				}
				return orderedMap
			}(),
		},
	}, {
		desc:     "multi-keyed ordered list",
		inSchema: ctestschema.SchemaTree["Device"],
		inParent: &ctestschema.Device{},
		inPath:   mustPath("/ordered-multikeyed-lists/ordered-multikeyed-list[key1=foo][key2=42]"),
		want: &ctestschema.OrderedMultikeyedList{
			Key1: ygot.String("foo"),
			Key2: ygot.Uint64(42),
		},
		wantParent: &ctestschema.Device{
			OrderedMultikeyedList: func() *ctestschema.OrderedMultikeyedList_OrderedMap {
				orderedMap := &ctestschema.OrderedMultikeyedList_OrderedMap{}
				_, err := orderedMap.AppendNew("foo", 42)
				if err != nil {
					t.Fatal(err)
				}
				return orderedMap
			}(),
		},
	}, {
		desc:             "multi-keyed ordered list with bad key",
		inSchema:         ctestschema.SchemaTree["Device"],
		inParent:         &ctestschema.Device{},
		inPath:           mustPath("/ordered-multikeyed-lists/ordered-multikeyed-list[key1=foo][key2=foo]"),
		wantErrSubstring: `unable to convert "foo" to uint64`,
	}, {
		desc:     "multi-keyed ordered list leaf",
		inSchema: ctestschema.SchemaTree["Device"],
		inParent: &ctestschema.Device{},
		inPath:   mustPath("/ordered-multikeyed-lists/ordered-multikeyed-list[key1=foo][key2=42]/config/value"),
		want:     ygot.String(""),
		wantParent: &ctestschema.Device{
			OrderedMultikeyedList: func() *ctestschema.OrderedMultikeyedList_OrderedMap {
				orderedMap := &ctestschema.OrderedMultikeyedList_OrderedMap{}
				v, err := orderedMap.AppendNew("foo", 42)
				if err != nil {
					t.Fatal(err)
				}
				v.Value = ygot.String("")
				return orderedMap
			}(),
		},
	}, {
		desc:     "multi-keyed ordered list leaf already exists",
		inSchema: ctestschema.SchemaTree["Device"],
		inParent: &ctestschema.Device{
			OrderedMultikeyedList: ctestschema.GetOrderedMapMultikeyed(t),
		},
		inPath: mustPath("/ordered-multikeyed-lists/ordered-multikeyed-list[key1=foo][key2=42]/config/value"),
		want:   ygot.String("foo-val"),
		wantParent: &ctestschema.Device{
			OrderedMultikeyedList: ctestschema.GetOrderedMapMultikeyed(t),
		},
	}, {
		desc:             "multi-keyed ordered list leaf without enough keys",
		inSchema:         ctestschema.SchemaTree["Device"],
		inParent:         &ctestschema.Device{},
		inPath:           mustPath("/ordered-multikeyed-lists/ordered-multikeyed-list[key2=42]/config/value"),
		wantErrSubstring: "got 1 valid keys, expected 2",
	}, {
		desc:     "nested single-keyed ordered list",
		inSchema: ctestschema.SchemaTree["Device"],
		inParent: &ctestschema.Device{},
		inPath:   mustPath("/ordered-lists/ordered-list[key=foo]/ordered-lists/ordered-list[key=bar]"),
		want: &ctestschema.OrderedList_OrderedList{
			Key: ygot.String("bar"),
		},
		wantParent: &ctestschema.Device{
			OrderedList: func() *ctestschema.OrderedList_OrderedMap {
				orderedMap := &ctestschema.OrderedList_OrderedMap{}
				v, err := orderedMap.AppendNew("foo")
				if err != nil {
					t.Fatal(err)
				}
				v.AppendNewOrderedList("bar")
				return orderedMap
			}(),
		},
	}, {
		desc:             "nested single-keyed ordered list error",
		inSchema:         ctestschema.SchemaTree["Device"],
		inParent:         &ctestschema.Device{},
		inPath:           mustPath("/ordered-lists/ordered-list[key=foo]/ordered-lists/ordered-list"),
		wantErrSubstring: "(/device/ordered-lists/ordered-list/ordered-lists/ordered-list): got 0 valid keys, expected 1",
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got, _, err := ytypes.GetOrCreateNode(tt.inSchema, tt.inParent, tt.inPath, tt.inOpts...)
			if diff := errdiff.Substring(err, tt.wantErrSubstring); diff != "" {
				t.Fatalf("did not get expected error, %s", diff)
			}
			if err != nil {
				return
			}
			if diff := cmp.Diff(tt.want, got, ytestutil.OrderedMapCmpOptions...); diff != "" {
				t.Errorf("(-want, +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.wantParent, tt.inParent, ytestutil.OrderedMapCmpOptions...); diff != "" {
				t.Errorf("parent (-want, +got):\n%s", diff)
			}
		})
	}
}

// hasIgnoreExtraFieldsSetNode determines whether the supplied slice of SetNodeOpts contains
// the IgnoreExtraFields option.
func hasIgnoreExtraFieldsSetNode(opts []ytypes.SetNodeOpt) bool {
	for _, o := range opts {
		if _, ok := o.(*ytypes.IgnoreExtraFields); ok {
			return true
		}
	}
	return false
}

// hasSetNodePreferShadowPath determines whether there is an instance of
// PreferShadowPath within the supplied GetOrCreateNodeOpt slice. It is used to
// determine whether to use the "shadow-path" tags instead of the "path" tag
// when both are present while processing a GoStruct.
func hasSetNodePreferShadowPath(opts []ytypes.SetNodeOpt) bool {
	for _, o := range opts {
		if _, ok := o.(*ytypes.PreferShadowPath); ok {
			return true
		}
	}
	return false
}

func TestSetNodeOrderedMap(t *testing.T) {
	tests := []struct {
		desc     string
		inSchema *yang.Entry
		// inParentFn allows the same input to be tested more than once
		// even if the first usage involved a modification.
		inParentFn       func() any
		inPath           *gpb.Path
		inVal            interface{}
		inValJSON        interface{}
		inOpts           []ytypes.SetNodeOpt
		wantErrSubstring string
		want             any
		wantParent       interface{}
	}{{
		desc:     "success setting string field in ordered map",
		inSchema: ctestschema.SchemaTree["Device"],
		inParentFn: func() any {
			return &ctestschema.Device{
				OrderedList: func() *ctestschema.OrderedList_OrderedMap {
					orderedMap := &ctestschema.OrderedList_OrderedMap{}
					v, err := orderedMap.AppendNew("foo")
					if err != nil {
						t.Fatal(err)
					}
					v.Value = ygot.String("foo-value")
					return orderedMap
				}(),
			}
		},
		inPath:    mustPath("/ordered-lists/ordered-list[key=foo]/config/value"),
		inVal:     &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "hello"}},
		inValJSON: &gpb.TypedValue{Value: &gpb.TypedValue_JsonIetfVal{JsonIetfVal: []byte(`"hello"`)}},
		want:      ygot.String("hello"),
		wantParent: &ctestschema.Device{
			OrderedList: func() *ctestschema.OrderedList_OrderedMap {
				orderedMap := &ctestschema.OrderedList_OrderedMap{}
				v, err := orderedMap.AppendNew("foo")
				if err != nil {
					t.Fatal(err)
				}
				v.Value = ygot.String("hello")
				return orderedMap
			}(),
		},
	}, {
		desc:     "success setting string field in ordered map",
		inSchema: ctestschema.SchemaTree["Device"],
		inParentFn: func() any {
			return &ctestschema.Device{
				OrderedList: func() *ctestschema.OrderedList_OrderedMap {
					orderedMap := &ctestschema.OrderedList_OrderedMap{}
					v, err := orderedMap.AppendNew("foo")
					if err != nil {
						t.Fatal(err)
					}
					v.Value = ygot.String("foo-value")
					return orderedMap
				}(),
			}
		},
		inPath:    mustPath("/ordered-lists/ordered-list[key=foo]/config/value"),
		inVal:     &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "hello"}},
		inValJSON: &gpb.TypedValue{Value: &gpb.TypedValue_JsonIetfVal{JsonIetfVal: []byte(`"hello"`)}},
		want:      ygot.String("hello"),
		wantParent: &ctestschema.Device{
			OrderedList: func() *ctestschema.OrderedList_OrderedMap {
				orderedMap := &ctestschema.OrderedList_OrderedMap{}
				v, err := orderedMap.AppendNew("foo")
				if err != nil {
					t.Fatal(err)
				}
				v.Value = ygot.String("hello")
				return orderedMap
			}(),
		},
	}, {
		desc:     "success not setting shadow string field in ordered map",
		inSchema: ctestschema.SchemaTree["Device"],
		inParentFn: func() any {
			return &ctestschema.Device{
				OrderedList: func() *ctestschema.OrderedList_OrderedMap {
					orderedMap := &ctestschema.OrderedList_OrderedMap{}
					v, err := orderedMap.AppendNew("foo")
					if err != nil {
						t.Fatal(err)
					}
					v.Value = ygot.String("foo-value")
					return orderedMap
				}(),
			}
		},
		inPath:    mustPath("/ordered-lists/ordered-list[key=foo]/state/value"),
		inVal:     &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "hello"}},
		inValJSON: &gpb.TypedValue{Value: &gpb.TypedValue_JsonIetfVal{JsonIetfVal: []byte(`"hello"`)}},
		want:      nil,
		wantParent: &ctestschema.Device{
			OrderedList: func() *ctestschema.OrderedList_OrderedMap {
				orderedMap := &ctestschema.OrderedList_OrderedMap{}
				v, err := orderedMap.AppendNew("foo")
				if err != nil {
					t.Fatal(err)
				}
				v.Value = ygot.String("foo-value")
				return orderedMap
			}(),
		},
	}, {
		desc:     "success setting string field in ordered map and initializing new list element",
		inSchema: ctestschema.SchemaTree["Device"],
		inParentFn: func() any {
			return &ctestschema.Device{}
		},
		inOpts:    []ytypes.SetNodeOpt{&ytypes.InitMissingElements{}},
		inPath:    mustPath("/ordered-lists/ordered-list[key=foo]/config/value"),
		inVal:     &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "hello"}},
		inValJSON: &gpb.TypedValue{Value: &gpb.TypedValue_JsonIetfVal{JsonIetfVal: []byte(`"hello"`)}},
		want:      ygot.String("hello"),
		wantParent: &ctestschema.Device{
			OrderedList: func() *ctestschema.OrderedList_OrderedMap {
				orderedMap := &ctestschema.OrderedList_OrderedMap{}
				v, err := orderedMap.AppendNew("foo")
				if err != nil {
					t.Fatal(err)
				}
				v.Value = ygot.String("hello")
				return orderedMap
			}(),
		},
	}, {
		desc:     "failure setting string field in ordered map when initialization option not provided",
		inSchema: ctestschema.SchemaTree["Device"],
		inParentFn: func() any {
			return &ctestschema.Device{}
		},
		inPath:           mustPath("/ordered-lists/ordered-list[key=foo]/config/value"),
		inVal:            &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "hello"}},
		inValJSON:        &gpb.TypedValue{Value: &gpb.TypedValue_JsonIetfVal{JsonIetfVal: []byte(`"hello"`)}},
		wantParent:       &ctestschema.Device{},
		wantErrSubstring: "could not find children",
	}, {
		desc:     "success setting (appending) single-keyed ordered map element",
		inSchema: ctestschema.SchemaTree["Device"],
		inParentFn: func() any {
			return &ctestschema.Device{
				OrderedList: ctestschema.GetOrderedMap(t),
			}
		},
		inOpts:    []ytypes.SetNodeOpt{&ytypes.InitMissingElements{}},
		inPath:    mustPath("/ordered-lists/ordered-list[key=new-key]"),
		inValJSON: &gpb.TypedValue{Value: &gpb.TypedValue_JsonIetfVal{JsonIetfVal: []byte(`{ "key": "new-key", "config": { "key": "new-key", "value": "hello" } }`)}},
		want: &ctestschema.OrderedList{
			Key:   ygot.String("new-key"),
			Value: ygot.String("hello"),
		},
		wantParent: &ctestschema.Device{
			OrderedList: func() *ctestschema.OrderedList_OrderedMap {
				orderedMap := ctestschema.GetOrderedMap(t)
				v, err := orderedMap.AppendNew("new-key")
				if err != nil {
					t.Fatal(err)
				}
				v.Value = ygot.String("hello")
				return orderedMap
			}(),
		},
	}, {
		desc:     "success setting (appending) multi-keyed ordered map element",
		inSchema: ctestschema.SchemaTree["Device"],
		inParentFn: func() any {
			return &ctestschema.Device{
				OrderedMultikeyedList: ctestschema.GetOrderedMapMultikeyed(t),
			}
		},
		inOpts:    []ytypes.SetNodeOpt{&ytypes.InitMissingElements{}},
		inPath:    mustPath("/ordered-multikeyed-lists/ordered-multikeyed-list[key1=new-key][key2=1024]"),
		inValJSON: &gpb.TypedValue{Value: &gpb.TypedValue_JsonIetfVal{JsonIetfVal: []byte(`{ "key1": "new-key", "key2": "1024", "config": { "key1": "new-key", "key2": "1024", "value": "hello" } }`)}},
		want: &ctestschema.OrderedMultikeyedList{
			Key1:  ygot.String("new-key"),
			Key2:  ygot.Uint64(1024),
			Value: ygot.String("hello"),
		},
		wantParent: &ctestschema.Device{
			OrderedMultikeyedList: func() *ctestschema.OrderedMultikeyedList_OrderedMap {
				orderedMap := ctestschema.GetOrderedMapMultikeyed(t)
				v, err := orderedMap.AppendNew("new-key", 1024)
				if err != nil {
					t.Fatal(err)
				}
				v.Value = ygot.String("hello")
				return orderedMap
			}(),
		},
	}, {
		desc:     "success appending by setting at parent level",
		inSchema: ctestschema.SchemaTree["Device"],
		inParentFn: func() any {
			return &ctestschema.Device{
				OrderedList: ctestschema.GetOrderedMap(t),
			}
		},
		inOpts:    []ytypes.SetNodeOpt{&ytypes.InitMissingElements{}},
		inPath:    mustPath("/"),
		inValJSON: &gpb.TypedValue{Value: &gpb.TypedValue_JsonIetfVal{JsonIetfVal: []byte(`{ "ordered-lists": { "ordered-list": [{"key": "new-key", "config": { "key": "new-key", "value": "hello" } }] } }`)}},
		want: &ctestschema.Device{
			OrderedList: func() *ctestschema.OrderedList_OrderedMap {
				orderedMap := ctestschema.GetOrderedMap(t)
				v, err := orderedMap.AppendNew("new-key")
				if err != nil {
					t.Fatal(err)
				}
				v.Value = ygot.String("hello")
				return orderedMap
			}(),
		},
		wantParent: &ctestschema.Device{
			OrderedList: func() *ctestschema.OrderedList_OrderedMap {
				orderedMap := ctestschema.GetOrderedMap(t)
				v, err := orderedMap.AppendNew("new-key")
				if err != nil {
					t.Fatal(err)
				}
				v.Value = ygot.String("hello")
				return orderedMap
			}(),
		},
	}, {
		desc:     "success appending by setting at parent level uncompressed",
		inSchema: utestschema.SchemaTree["Device"],
		inParentFn: func() any {
			return utestschema.GetDeviceWithOrderedMap(t)
		},
		inOpts:    []ytypes.SetNodeOpt{&ytypes.InitMissingElements{}},
		inPath:    mustPath("/"),
		inValJSON: &gpb.TypedValue{Value: &gpb.TypedValue_JsonIetfVal{JsonIetfVal: []byte(`{ "ordered-lists": { "ordered-list": [{"key": "new-key", "config": { "key": "new-key", "value": "hello" } }] } }`)}},
		want: func() *utestschema.Device {
			d := utestschema.GetDeviceWithOrderedMap(t)
			v, err := d.GetOrderedLists().AppendNewOrderedList("new-key")
			if err != nil {
				t.Fatal(err)
			}
			v.GetOrCreateConfig().Key = ygot.String("new-key")
			v.GetOrCreateConfig().Value = ygot.String("hello")
			return d
		}(),
		wantParent: func() *utestschema.Device {
			d := utestschema.GetDeviceWithOrderedMap(t)
			v, err := d.GetOrderedLists().AppendNewOrderedList("new-key")
			if err != nil {
				t.Fatal(err)
			}
			v.GetOrCreateConfig().Key = ygot.String("new-key")
			v.GetOrCreateConfig().Value = ygot.String("hello")
			return d
		}(),
	}, {
		desc:     "success setting entire ordered map",
		inSchema: ctestschema.SchemaTree["Device"],
		inParentFn: func() any {
			return &ctestschema.Device{}
		},
		inOpts:    []ytypes.SetNodeOpt{&ytypes.InitMissingElements{}},
		inPath:    mustPath("/"),
		inValJSON: &gpb.TypedValue{Value: &gpb.TypedValue_JsonIetfVal{JsonIetfVal: []byte(`{ "ordered-lists": { "ordered-list": [{"key": "foo", "config": { "key": "foo", "value": "foo-val" } }, {"key": "bar", "config": { "key": "bar", "value": "bar-val" } }] } }`)}},
		want: &ctestschema.Device{
			OrderedList: ctestschema.GetOrderedMap(t),
		},
		wantParent: &ctestschema.Device{
			OrderedList: ctestschema.GetOrderedMap(t),
		},
	}}

	for _, tt := range tests {
		for typeDesc, inVal := range map[string]interface{}{"scalar": tt.inVal, "JSON": tt.inValJSON} {
			if inVal == nil {
				continue
			}
			t.Run(tt.desc+" "+typeDesc, func(t *testing.T) {
				parent := tt.inParentFn()
				err := ytypes.SetNode(tt.inSchema, parent, tt.inPath, inVal, tt.inOpts...)
				if diff := errdiff.Substring(err, tt.wantErrSubstring); diff != "" {
					t.Fatalf("got %v\nwant %v", err, tt.wantErrSubstring)
				}
				if diff := cmp.Diff(tt.wantParent, parent, ytestutil.OrderedMapCmpOptions...); diff != "" {
					t.Errorf("(-wantParent, +got):\n%s", diff)
				}
				if err != nil {
					return
				}
				if tt.want == nil && hasIgnoreExtraFieldsSetNode(tt.inOpts) {
					return
				}

				var getNodeOpts []ytypes.GetNodeOpt
				if hasSetNodePreferShadowPath(tt.inOpts) {
					getNodeOpts = append(getNodeOpts, &ytypes.PreferShadowPath{})
				}
				treeNode, err := ytypes.GetNode(tt.inSchema, parent, tt.inPath, getNodeOpts...)
				if err != nil {
					t.Fatalf("unexpected error returned from GetNode: %v", err)
				}
				switch {
				case len(treeNode) == 1:
					// Expected case for most tests.
					break
				case len(treeNode) == 0 && tt.want == nil:
					return
				default:
					t.Fatalf("did not get exactly one tree node: %v", treeNode)
				}
				got := treeNode[0].Data
				if diff := cmp.Diff(tt.want, got, ytestutil.OrderedMapCmpOptions...); diff != "" {
					t.Errorf("(-wantLeaf, +got):\n%s", diff)
				}
			})
		}
	}
}

func TestDeleteNodeOrderedMap(t *testing.T) {
	tests := []struct {
		desc             string
		inSchema         *yang.Entry
		inParent         any
		inPath           *gpb.Path
		inOpts           []ytypes.DelNodeOpt
		wantParent       any
		wantErrSubstring string
	}{{
		desc:     "success deleting string field in ordered map",
		inSchema: ctestschema.SchemaTree["Device"],
		inParent: &ctestschema.Device{
			OrderedList: ctestschema.GetOrderedMap(t),
		},
		inPath: mustPath("/ordered-lists/ordered-list[key=foo]/config/value"),
		wantParent: &ctestschema.Device{
			OrderedList: func() *ctestschema.OrderedList_OrderedMap {
				orderedMap := ctestschema.GetOrderedMap(t)
				v := orderedMap.Get("foo")
				if v == nil {
					t.Fatalf("key foo doesn't exist in ordered map")
				}
				v.Value = nil
				return orderedMap
			}(),
		},
	}, {
		desc:     "success not deleting shadow string field in ordered map",
		inSchema: ctestschema.SchemaTree["Device"],
		inParent: &ctestschema.Device{
			OrderedList: ctestschema.GetOrderedMap(t),
		},
		inPath: mustPath("/ordered-lists/ordered-list[key=foo]/state/value"),
		wantParent: &ctestschema.Device{
			OrderedList: ctestschema.GetOrderedMap(t),
		},
	}, {
		desc:     "success deleting an ordered map element",
		inSchema: ctestschema.SchemaTree["Device"],
		inParent: &ctestschema.Device{
			OrderedList: ctestschema.GetOrderedMap(t),
		},
		inPath: mustPath("/ordered-lists/ordered-list[key=foo]"),
		wantParent: &ctestschema.Device{
			OrderedList: func() *ctestschema.OrderedList_OrderedMap {
				orderedMap := ctestschema.GetOrderedMap(t)
				if deleted := orderedMap.Delete("foo"); !deleted {
					t.Fatalf("key foo was not deleted")
				}
				return orderedMap
			}(),
		},
	}, {
		desc:     "success deleting entire single-keyed ordered map at container level",
		inSchema: ctestschema.SchemaTree["Device"],
		inParent: &ctestschema.Device{
			OrderedList: ctestschema.GetOrderedMap(t),
		},
		inPath:     mustPath("/ordered-lists"),
		wantParent: &ctestschema.Device{},
	}, {
		desc:     "success deleting an ordered map element's key field",
		inSchema: ctestschema.SchemaTree["Device"],
		inParent: &ctestschema.Device{
			OrderedList: ctestschema.GetOrderedMap(t),
		},
		inPath: mustPath("/ordered-lists/ordered-list[key=foo]/config/key"),
		wantParent: &ctestschema.Device{
			OrderedList: func() *ctestschema.OrderedList_OrderedMap {
				orderedMap := ctestschema.GetOrderedMap(t)
				v := orderedMap.Get("foo")
				if v == nil {
					t.Fatalf("key foo doesn't exist in ordered map")
				}
				v.Key = nil
				return orderedMap
			}(),
		},
	}, {
		desc:     "deleting an ordered map element non-key field when the key field has been deleted -- this should trigger the entire list entry to be deleted since it's now empty",
		inSchema: ctestschema.SchemaTree["Device"],
		inParent: &ctestschema.Device{
			OrderedList: func() *ctestschema.OrderedList_OrderedMap {
				orderedMap := ctestschema.GetOrderedMap(t)
				v := orderedMap.Get("foo")
				if v == nil {
					t.Fatalf("key foo doesn't exist in ordered map")
				}
				v.Key = nil
				return orderedMap
			}(),
		},
		inPath: mustPath("/ordered-lists/ordered-list[key=foo]/config/value"),
		wantParent: &ctestschema.Device{
			OrderedList: func() *ctestschema.OrderedList_OrderedMap {
				orderedMap := ctestschema.GetOrderedMap(t)
				if deleted := orderedMap.Delete("foo"); !deleted {
					t.Fatalf("key foo was not deleted")
				}
				return orderedMap
			}(),
		},
	}, {
		desc:     "deleting an ordered map element key field when the non-key field has been deleted -- this should trigger the entire list entry to be deleted since it's now empty",
		inSchema: ctestschema.SchemaTree["Device"],
		inParent: &ctestschema.Device{
			OrderedList: func() *ctestschema.OrderedList_OrderedMap {
				orderedMap := ctestschema.GetOrderedMap(t)
				v := orderedMap.Get("foo")
				if v == nil {
					t.Fatalf("key foo doesn't exist in ordered map")
				}
				v.Value = nil
				return orderedMap
			}(),
		},
		inPath: mustPath("/ordered-lists/ordered-list[key=foo]/config/key"),
		wantParent: &ctestschema.Device{
			OrderedList: func() *ctestschema.OrderedList_OrderedMap {
				orderedMap := ctestschema.GetOrderedMap(t)
				if deleted := orderedMap.Delete("foo"); !deleted {
					t.Fatalf("key foo was not deleted")
				}
				return orderedMap
			}(),
		},
	}, {
		desc:     "success deleting string field in multi-keyed ordered map",
		inSchema: ctestschema.SchemaTree["Device"],
		inParent: &ctestschema.Device{
			OrderedMultikeyedList: ctestschema.GetOrderedMapMultikeyed(t),
		},
		inPath: mustPath("/ordered-multikeyed-lists/ordered-multikeyed-list[key1=foo][key2=42]/config/value"),
		wantParent: &ctestschema.Device{
			OrderedMultikeyedList: func() *ctestschema.OrderedMultikeyedList_OrderedMap {
				orderedMap := ctestschema.GetOrderedMapMultikeyed(t)
				v := orderedMap.Get(ctestschema.OrderedMultikeyedList_Key{
					Key1: "foo",
					Key2: 42,
				})
				if v == nil {
					t.Fatalf("key doesn't exist in ordered map")
				}
				v.Value = nil
				return orderedMap
			}(),
		},
	}, {
		desc:     "success deleting multi-keyed ordered map element",
		inSchema: ctestschema.SchemaTree["Device"],
		inParent: &ctestschema.Device{
			OrderedMultikeyedList: ctestschema.GetOrderedMapMultikeyed(t),
		},
		inPath: mustPath("/ordered-multikeyed-lists/ordered-multikeyed-list[key1=foo][key2=42]"),
		wantParent: &ctestschema.Device{
			OrderedMultikeyedList: func() *ctestschema.OrderedMultikeyedList_OrderedMap {
				orderedMap := ctestschema.GetOrderedMapMultikeyed(t)
				if deleted := orderedMap.Delete(ctestschema.OrderedMultikeyedList_Key{
					Key1: "foo",
					Key2: 42,
				}); !deleted {
					t.Fatalf("key was not deleted")
				}
				return orderedMap
			}(),
		},
	}, {
		desc:     "success deleting entire multi-keyed ordered map at container level",
		inSchema: ctestschema.SchemaTree["Device"],
		inParent: &ctestschema.Device{
			OrderedMultikeyedList: ctestschema.GetOrderedMapMultikeyed(t),
		},
		inPath:     mustPath("/ordered-multikeyed-lists"),
		wantParent: &ctestschema.Device{},
	}, {
		desc:     "success deleting entire multi-keyed ordered map at container level when shadowpath option is turned on",
		inSchema: ctestschema.SchemaTree["Device"],
		inParent: &ctestschema.Device{
			OrderedMultikeyedList: ctestschema.GetOrderedMapMultikeyed(t),
		},
		inOpts:     []ytypes.DelNodeOpt{&ytypes.PreferShadowPath{}},
		inPath:     mustPath("/ordered-multikeyed-lists"),
		wantParent: &ctestschema.Device{},
	}, {
		desc:     "success deleting last key field in multi-keyed ordered map which triggers deletion of element",
		inSchema: ctestschema.SchemaTree["Device"],
		inParent: &ctestschema.Device{
			OrderedMultikeyedList: func() *ctestschema.OrderedMultikeyedList_OrderedMap {
				orderedMap := ctestschema.GetOrderedMapMultikeyed(t)
				v := orderedMap.Get(ctestschema.OrderedMultikeyedList_Key{
					Key1: "foo",
					Key2: 42,
				})
				if v == nil {
					t.Fatalf("key doesn't exist in ordered map")
				}
				v.Key1 = nil
				v.Value = nil
				return orderedMap
			}(),
		},
		inPath: mustPath("/ordered-multikeyed-lists/ordered-multikeyed-list[key1=foo][key2=42]/key2"),
		wantParent: &ctestschema.Device{
			OrderedMultikeyedList: func() *ctestschema.OrderedMultikeyedList_OrderedMap {
				orderedMap := ctestschema.GetOrderedMapMultikeyed(t)
				if deleted := orderedMap.Delete(ctestschema.OrderedMultikeyedList_Key{
					Key1: "foo",
					Key2: 42,
				}); !deleted {
					t.Fatalf("key was not deleted")
				}
				return orderedMap
			}(),
		},
	}, {
		desc:     "success deleting non-existent multi-keyed ordered map element",
		inSchema: ctestschema.SchemaTree["Device"],
		inParent: &ctestschema.Device{
			OrderedMultikeyedList: ctestschema.GetOrderedMapMultikeyed(t),
		},
		inPath: mustPath("/ordered-multikeyed-lists/ordered-multikeyed-list[key1=dne][key2=999]"),
		wantParent: &ctestschema.Device{
			OrderedMultikeyedList: ctestschema.GetOrderedMapMultikeyed(t),
		},
	}, {
		desc:     "error deleting multi-keyed ordered map element path that doesn't exist",
		inSchema: ctestschema.SchemaTree["Device"],
		inParent: &ctestschema.Device{
			OrderedMultikeyedList: ctestschema.GetOrderedMapMultikeyed(t),
		},
		inPath:           mustPath("/ordered-multikeyed-lists/ordered-multikeyed-list[key1=foo][key2=42]/config/dne"),
		wantErrSubstring: "no match found",
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := ytypes.DeleteNode(tt.inSchema, tt.inParent, tt.inPath, tt.inOpts...)
			if diff := errdiff.Substring(err, tt.wantErrSubstring); diff != "" {
				t.Fatalf("got error %v\nwant error substr: %s", err, tt.wantErrSubstring)
			}
			if err != nil {
				return
			}

			if diff := cmp.Diff(tt.wantParent, tt.inParent, ytestutil.OrderedMapCmpOptions...); diff != "" {
				t.Errorf("TestDeleteNode (-want, +got):\n%s", diff)
			}
		})
	}
}
