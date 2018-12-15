// Copyright 2018 Google Inc.
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

package validate

import (
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/openconfig/gnmi/errdiff"
	"github.com/openconfig/ygot/exampleoc"
	"github.com/openconfig/ygot/uexampleoc"
	"github.com/openconfig/ygot/ygot"
	"github.com/openconfig/ygot/ytypes"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
)

func mustSchema(fn func() (*ytypes.Schema, error)) *ytypes.Schema {
	s, err := fn()
	if err != nil {
		panic(err)
	}
	return s
}

func TestSet(t *testing.T) {
	tests := []struct {
		desc             string
		inSchema         *ytypes.Schema
		inPath           *gpb.Path
		inValue          *gpb.TypedValue
		inOpts           []ytypes.SetNodeOpt
		wantErrSubstring string
		wantNode         *ytypes.TreeNode
	}{{
		desc:     "set leafref with mismatched name - compressed schema",
		inSchema: mustSchema(exampleoc.Schema),
		inPath: &gpb.Path{
			Elem: []*gpb.PathElem{{
				Name: "components",
			}, {
				Name: "component",
				Key: map[string]string{
					"name": "OCH-1-2",
				},
			}, {
				Name: "optical-channel",
			}, {
				Name: "config",
			}, {
				Name: "line-port",
			}},
		},
		inValue: &gpb.TypedValue{
			Value: &gpb.TypedValue_StringVal{"XCVR-1-2"},
		},
		inOpts: []ytypes.SetNodeOpt{&ytypes.InitMissingElements{}},
		wantNode: &ytypes.TreeNode{
			Path: &gpb.Path{
				Elem: []*gpb.PathElem{{
					Name: "components",
				}, {
					Name: "component",
					Key: map[string]string{
						"name": "OCH-1-2",
					},
				}, {
					Name: "optical-channel",
				}, {
					Name: "config",
				}, {
					Name: "line-port",
				}},
			},
			Data: ygot.String("XCVR-1-2"),
		},
	}, {
		desc:     "set leafref with mismatched name - uncompressed schema",
		inSchema: mustSchema(uexampleoc.Schema),
		inPath: &gpb.Path{
			Elem: []*gpb.PathElem{{
				Name: "components",
			}, {
				Name: "component",
				Key: map[string]string{
					"name": "OCH-1-2",
				},
			}, {
				Name: "optical-channel",
			}, {
				Name: "state",
			}, {
				Name: "line-port",
			}},
		},
		inValue: &gpb.TypedValue{
			Value: &gpb.TypedValue_StringVal{"XCVR-1-2"},
		},
		inOpts: []ytypes.SetNodeOpt{&ytypes.InitMissingElements{}},
		wantNode: &ytypes.TreeNode{
			Path: &gpb.Path{
				Elem: []*gpb.PathElem{{
					Name: "components",
				}, {
					Name: "component",
					Key: map[string]string{
						"name": "OCH-1-2",
					},
				}, {
					Name: "optical-channel",
				}, {
					Name: "state",
				}, {
					Name: "line-port",
				}},
			},
			Data: ygot.String("XCVR-1-2"),
		},
	}, {
		desc:     "bad path",
		inSchema: mustSchema(uexampleoc.Schema),
		inPath: &gpb.Path{
			Elem: []*gpb.PathElem{{
				Name: "doesnt-exist",
			}},
		},
		inValue: &gpb.TypedValue{
			Value: &gpb.TypedValue_IntVal{42},
		},
		wantErrSubstring: "no match found",
	}, {
		desc:     "wrong type",
		inSchema: mustSchema(uexampleoc.Schema),
		inPath: &gpb.Path{
			Elem: []*gpb.PathElem{{
				Name: "system",
			}, {
				Name: "config",
			}, {
				Name: "hostname",
			}},
		},
		inValue: &gpb.TypedValue{
			Value: &gpb.TypedValue_UintVal{42},
		},
		inOpts:           []ytypes.SetNodeOpt{&ytypes.InitMissingElements{}},
		wantErrSubstring: "failed to unmarshal &{42} into string",
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := ytypes.SetNode(tt.inSchema.RootSchema(), tt.inSchema.Root, tt.inPath, tt.inValue, tt.inOpts...)
			if diff := errdiff.Substring(err, tt.wantErrSubstring); diff != "" {
				t.Fatalf("did not get expected error, %s", diff)
			}

			if tt.wantNode == nil {
				return
			}

			got, err := ytypes.GetNode(tt.inSchema.RootSchema(), tt.inSchema.Root, tt.wantNode.Path)
			if err != nil {
				t.Fatalf("cannot perform get, %v", err)
			}
			if len(got) != 1 {
				t.Fatalf("unexpected number of nodes, want: 1, got: %d", len(got))
			}

			opts := []cmp.Option{
				cmpopts.IgnoreFields(ytypes.TreeNode{}, "Schema"),
				cmp.Comparer(proto.Equal),
			}

			if !cmp.Equal(got[0], tt.wantNode, opts...) {
				diff := cmp.Diff(got[0], tt.wantNode, opts...)
				t.Fatalf("did not get expected node, got: %v, want: %v, diff:\n%s", got[0], tt.wantNode, diff)
			}
		})
	}
}
