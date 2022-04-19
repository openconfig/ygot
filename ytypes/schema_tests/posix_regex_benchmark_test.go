// Copyright 2021 Google Inc.
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
	"fmt"
	"testing"

	oc "github.com/openconfig/ygot/exampleoc"
)

func BenchmarkPOSIXPattern(b *testing.B) {
	d := &oc.Device{}
	prefixSet := d.GetOrCreateRoutingPolicy().GetOrCreateDefinedSets().GetOrCreatePrefixSet("foo")
	for i := 0; i != 256; i++ {
		prefixSet.NewPrefix(fmt.Sprintf("%d.%d.%d.0/24", i, i, i), "exact")
	}
	for i := 0; i != 256; i++ {
		prefixSet.NewPrefix(fmt.Sprintf("FFFF:%d:EEEE:AAAA::/64", i), "60..64")
	}

	b.ResetTimer()
	for i := 0; i != b.N; i++ {
		if err := d.ΛValidate(); err != nil {
			b.Fatalf("d.Validate() failed: %v", err)
		}
	}
}
