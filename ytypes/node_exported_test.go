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
		inData           any
		inPath           *gpb.Path
		inArgs           []ytypes.GetNodeOpt
		wantTreeNodes    []*ytypes.TreeNode
		wantErrSubstring string
	}{{
		desc:     "single-keyed ordered list",
		inSchema: ctestschema.SchemaTree["Device"],
		inData: &ctestschema.Device{
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
		desc:     "single-keyed ordered list match on second",
		inSchema: ctestschema.SchemaTree["Device"],
		inData: &ctestschema.Device{
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
		inData: &ctestschema.Device{
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
		inData: &ctestschema.Device{
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
		inData: &ctestschema.Device{
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
		inData: &ctestschema.Device{
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
		inData: &ctestschema.Device{
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
		inData: &ctestschema.Device{
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
		inData: &ctestschema.Device{
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
		inData: &ctestschema.Device{
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
		inData: &ctestschema.Device{
			OrderedList: ctestschema.GetOrderedMap(t),
		},
		inPath:           mustPath("/ordered-lists/ordered-list[key=foo]/config/does-not-exist"),
		wantErrSubstring: "no match found",
	}, {
		desc:     "value through single-keyed ordered list",
		inSchema: ctestschema.SchemaTree["Device"],
		inData: &ctestschema.Device{
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
		inData: &ctestschema.Device{
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
		inData: &ctestschema.Device{
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
		inData: &ctestschema.Device{
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
		inData: &ctestschema.Device{
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
			got, err := ytypes.GetNode(tt.inSchema, tt.inData, tt.inPath, tt.inArgs...)
			if diff := errdiff.Substring(err, tt.wantErrSubstring); diff != "" {
				t.Fatalf("did not get expected error, %s", diff)
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
