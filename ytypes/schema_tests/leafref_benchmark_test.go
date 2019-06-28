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
	if err := d.Validate(); err != nil {
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
