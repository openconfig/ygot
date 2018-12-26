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
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/openconfig/gnmi/errdiff"
	"github.com/openconfig/goyang/pkg/yang"
	oc "github.com/openconfig/ygot/exampleoc"
	"github.com/openconfig/ygot/ygot"
	"github.com/openconfig/ygot/ytypes"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
)

func mustPath(s string) *gpb.Path {
	p, err := ygot.StringToStructuredPath(s)
	if err != nil {
		panic(err)
	}
	return p
}

func TestGetNodeFull(t *testing.T) {
	rootSchema := oc.SchemaTree[reflect.TypeOf(oc.Device{}).Name()]
	tests := []struct {
		name             string
		inRoot           *oc.Device
		inSchema         *yang.Entry
		inPath           *gpb.Path
		inOpts           []ytypes.GetNodeOpt
		wantNodes        []*ytypes.TreeNode
		wantErrSubstring string
	}{{
		name: "simple interface get",
		inRoot: func() *oc.Device {
			d := &oc.Device{}
			d.GetOrCreateInterface("eth0").Description = ygot.String("an interface")
			return d
		}(),
		inSchema: rootSchema,
		inPath:   mustPath("/interfaces/interface[name=eth0]"),
		wantNodes: []*ytypes.TreeNode{{
			Path: mustPath("/interfaces/interface[name=eth0]"),
			Data: &oc.Interface{
				Name:        ygot.String("eth0"),
				Description: ygot.String("an interface"),
			},
		}},
	}, {
		name: "interface leaf get",
		inRoot: func() *oc.Device {
			d := &oc.Device{}
			d.GetOrCreateInterface("eth0").Description = ygot.String("foo")
			return d
		}(),
		inSchema: rootSchema,
		inPath:   mustPath("/interfaces/interface[name=eth0]/config/description"),
		wantNodes: []*ytypes.TreeNode{{
			Path: mustPath("/interfaces/interface[name=eth0]/config/description"),
			Data: ygot.String("foo"),
		}},
	}, {
		name: "bad path",
		inRoot: func() *oc.Device {
			d := &oc.Device{}
			d.GetOrCreateSystem().Hostname = ygot.String("a value")
			return d
		}(),
		inSchema:         rootSchema,
		inPath:           mustPath("/does-not-exist"),
		wantErrSubstring: "no match found",
	}, {
		name:             "uninitialised path",
		inRoot:           &oc.Device{},
		inSchema:         rootSchema,
		inPath:           mustPath("/interfaces/interface[name=eth1]"),
		wantNodes:        []*ytypes.TreeNode{},
		wantErrSubstring: "could not find children",
	}, {
		name: "multiple leaves",
		inRoot: func() *oc.Device {
			d := &oc.Device{}
			d.GetOrCreateInterface("eth0").Description = ygot.String("eth0")
			d.GetOrCreateInterface("eth1").Description = ygot.String("eth1")
			return d
		}(),
		inSchema: rootSchema,
		inPath:   mustPath("/interfaces/interface/config/description"),
		inOpts:   []ytypes.GetNodeOpt{&ytypes.GetPartialKeyMatch{}},
		wantNodes: []*ytypes.TreeNode{{
			Path: mustPath("/interfaces/interface[name=eth0]/config/description"),
			Data: ygot.String("eth0"),
		}, {
			Path: mustPath("/interfaces/interface[name=eth1]/config/description"),
			Data: ygot.String("eth1"),
		}},
	}, {
		name: "multiple containers",
		inRoot: func() *oc.Device {
			d := &oc.Device{}
			d.GetOrCreateInterface("eth0").Description = ygot.String("eth0")
			d.GetOrCreateInterface("eth1").Description = ygot.String("eth1")
			return d
		}(),
		inSchema: rootSchema,
		inPath:   mustPath("/interfaces/interface"),
		inOpts:   []ytypes.GetNodeOpt{&ytypes.GetPartialKeyMatch{}},
		wantNodes: []*ytypes.TreeNode{{
			Path: mustPath("/interfaces/interface[name=eth0]"),
			Data: &oc.Interface{
				Name:        ygot.String("eth0"),
				Description: ygot.String("eth0"),
			},
		}, {
			Path: mustPath("/interfaces/interface[name=eth1]"),
			Data: &oc.Interface{
				Name:        ygot.String("eth1"),
				Description: ygot.String("eth1"),
			},
		}},
	}, {
		name: "nil interfaces",
		inRoot: func() *oc.Device {
			d := &oc.Device{}
			return d
		}(),
		inSchema:         rootSchema,
		inPath:           mustPath("/interfaces/interface[name=eth0]"),
		wantErrSubstring: "NotFound",
	}}

	ignoreSchema := cmpopts.IgnoreFields(ytypes.TreeNode{}, "Schema")
	sortNodes := cmpopts.SortSlices(func(a, b *ytypes.TreeNode) bool {
		an, err := ygot.PathToString(a.Path)
		if err != nil {
			panic(err)
		}
		bn, err := ygot.PathToString(b.Path)
		if err != nil {
			panic(err)
		}
		return an < bn
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ytypes.GetNode(tt.inSchema, tt.inRoot, tt.inPath, tt.inOpts...)
			if diff := errdiff.Substring(err, tt.wantErrSubstring); diff != "" {
				t.Fatalf("did not get expected error, %v", diff)
			}

			if err != nil {
				return
			}

			if diff := cmp.Diff(got, tt.wantNodes, ignoreSchema, cmpopts.EquateEmpty(), sortNodes); diff != "" {
				t.Fatalf("did not get expected result, diff(-got,+want):\n%s", diff)
			}
		})
	}
}
