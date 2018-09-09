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

package uncompressed

import (
	"testing"

	"github.com/openconfig/ygot/integration_tests/uncompressed/cschema"
	"github.com/openconfig/ygot/integration_tests/uncompressed/uschema"
	"github.com/openconfig/ygot/ygot"
)

// TestLeafref tests referencing between two lists using relative paths
// in compressed and uncompressed schemas. Previously, a bug was found
// in this behaviour in https://github.com/openconfig/ygot/issues/185.
func TestLeafref(t *testing.T) {
	tests := []struct {
		name    string
		in      ygot.ValidatedGoStruct
		wantErr bool
	}{{
		name: "compressed - relative reference OK",
		in: func() ygot.ValidatedGoStruct {
			c := &cschema.Root{}
			_ = c.GetOrCreateRootContainer().GetOrCreateTargetList("entry-one")
			r := c.GetOrCreateRootContainer().GetOrCreateReferencingList("pointer-one")
			r.RelativeReference = ygot.String("entry-one")
			return c
		}(),
	}, {
		name: "compressed - relative reference fails",
		in: func() ygot.ValidatedGoStruct {
			c := &cschema.Root{}
			_ = c.GetOrCreateRootContainer().GetOrCreateTargetList("entry-one")
			r := c.GetOrCreateRootContainer().GetOrCreateReferencingList("pointer-one")
			r.RelativeReference = ygot.String("crash-bang-wallop")
			return c
		}(),
		wantErr: true,
	}, {
		name: "uncompressed - relative reference OK",
		in: func() ygot.ValidatedGoStruct {
			u := &uschema.Root{}
			tv := u.GetOrCreateRootContainer().GetOrCreateTargetLists().GetOrCreateTargetList("entry-one").GetOrCreateConfig()
			tv.Key = ygot.String("entry-one")
			r := u.GetOrCreateRootContainer().GetOrCreateReferencingLists().GetOrCreateReferencingList("pointer-one").GetOrCreateConfig()
			r.Key = ygot.String("pointer-one")
			r.RelativeReference = ygot.String("entry-one")
			return u
		}(),
	}, {
		name: "uncompressed - relative reference fails",
		in: func() ygot.ValidatedGoStruct {
			u := &uschema.Root{}
			tv := u.GetOrCreateRootContainer().GetOrCreateTargetLists().GetOrCreateTargetList("entry-one").GetOrCreateConfig()
			tv.Key = ygot.String("entry-one")
			r := u.GetOrCreateRootContainer().GetOrCreateReferencingLists().GetOrCreateReferencingList("pointer-one").GetOrCreateConfig()
			r.Key = ygot.String("pointer-one")
			r.RelativeReference = ygot.String("fizz-buzz")
			return u
		}(),
		wantErr: true,
	}, {
		name: "compressed - absolute reference OK",
		in: func() ygot.ValidatedGoStruct {
			c := &cschema.Root{}
			_ = c.GetOrCreateRootContainer().GetOrCreateTargetList("entry-one")
			r := c.GetOrCreateRootContainer().GetOrCreateReferencingList("pointer-one")
			r.AbsoluteReference = ygot.String("entry-one")
			return c
		}(),
	}, {
		name: "compressed - absolute reference fails",
		in: func() ygot.ValidatedGoStruct {
			c := &cschema.Root{}
			_ = c.GetOrCreateRootContainer().GetOrCreateTargetList("entry-one")
			r := c.GetOrCreateRootContainer().GetOrCreateReferencingList("pointer-one")
			r.AbsoluteReference = ygot.String("crash-bang-wallop")
			return c
		}(),
		wantErr: true,
	}, {
		name: "uncompressed - absolute reference OK",
		in: func() ygot.ValidatedGoStruct {
			u := &uschema.Root{}
			tv := u.GetOrCreateRootContainer().GetOrCreateTargetLists().GetOrCreateTargetList("entry-one").GetOrCreateConfig()
			tv.Key = ygot.String("entry-one")
			r := u.GetOrCreateRootContainer().GetOrCreateReferencingLists().GetOrCreateReferencingList("pointer-one").GetOrCreateConfig()
			r.Key = ygot.String("pointer-one")
			r.AbsoluteReference = ygot.String("entry-one")
			return u
		}(),
	}, {
		name: "uncompressed - absolute reference fails",
		in: func() ygot.ValidatedGoStruct {
			u := &uschema.Root{}
			tv := u.GetOrCreateRootContainer().GetOrCreateTargetLists().GetOrCreateTargetList("entry-one").GetOrCreateConfig()
			tv.Key = ygot.String("entry-one")
			r := u.GetOrCreateRootContainer().GetOrCreateReferencingLists().GetOrCreateReferencingList("pointer-one").GetOrCreateConfig()
			r.Key = ygot.String("pointer-one")
			r.AbsoluteReference = ygot.String("fizz-buzz")
			return u
		}(),
		wantErr: true,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.in.Validate(); (err != nil) != tt.wantErr {
				t.Fatalf("incorrect validation result, got: %v, wantErr? %v", err, tt.wantErr)
			}
		})
	}
}
