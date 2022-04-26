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

package validate

import (
	"fmt"
	"testing"

	oc "github.com/openconfig/ygot/exampleoc"
	"github.com/openconfig/ygot/ygot"
)

func benchmarkIntsSubints(b *testing.B, ints, subints int) {
	d := &oc.Device{}
	for i := 0; i < ints; i++ {
		for j := 0; j < subints; j++ {
			d.GetOrCreateInterface(fmt.Sprintf("eth%d", i)).GetOrCreateSubinterface(uint32(j))
		}
	}

	// Create a reference just to ensure that we're validating leafrefs.
	d.GetOrCreateLldp().GetOrCreateInterface(fmt.Sprintf("eth%d", ints-1))

	r := d.GetOrCreateNetworkInstance("DEFAULT").GetOrCreateInterface("customerA")
	r.Interface = ygot.String(fmt.Sprintf("eth%d", ints-1))
	r.Subinterface = ygot.Uint32(uint32(subints) - 1)

	b.ResetTimer()
	if err := d.ΛValidate(); err != nil {
		b.FailNow()
	}
}

// Each of the following benchmarks has roughly the same number of subinterfaces.
func BenchmarkInterfaceLeafrefs1(b *testing.B) {
	benchmarkIntsSubints(b, 1000, 3)
}

func BenchmarkInterfaceLeafrefs2(b *testing.B) {
	benchmarkIntsSubints(b, 55, 55)
}

func BenchmarkInterfaceLeafrefs3(b *testing.B) {
	benchmarkIntsSubints(b, 3, 1000)
}
