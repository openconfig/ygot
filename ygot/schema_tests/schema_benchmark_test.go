package schematest

import (
	"testing"

	"github.com/openconfig/ygot/exampleoc"
)

func BenchmarkPopulateDefaults(b *testing.B) {
	for n := 0; n != b.N; n++ {
		d := &exampleoc.Device{}
		d.PopulateDefaults()
	}
}
