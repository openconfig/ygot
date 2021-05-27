// Copyright 2020 Google Inc.
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

package schemaops

import (
	"testing"

	"github.com/openconfig/ygot/integration_tests/schemaops/testschema"
	"github.com/openconfig/ygot/ygot"
	"github.com/openconfig/ygot/ytypes"
)

func TestGetOrCreateNode(t *testing.T) {
	t.Run("list with absolute leafref key, #489", func(t *testing.T) {
		ysch, err := testschema.Schema()
		if err != nil {
			t.Fatalf("could not get schema from test package, %v", err)
		}

		p, err := ygot.StringToStructuredPath("/ref/reference[name=foo]/name")
		if err != nil {
			t.Fatalf("could not convert string to path, %v", err)
		}

		v, sch, err := ytypes.GetOrCreateNode(ysch.RootSchema(), ysch.Root, p)
		if err != nil {
			t.Fatalf("could not retrieve node, %v", err)
		}
		_, _ = v, sch
	})
}
