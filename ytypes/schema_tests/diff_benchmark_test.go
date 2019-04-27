package validate

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	oc "github.com/openconfig/ygot/exampleoc"
	"github.com/openconfig/ygot/ygot"
)

func BenchmarkDiff(b *testing.B) {
	jsonFileA := "interfaceBenchmarkA.json"
	jsonFileB := "interfaceBenchmarkB.json"

	jsonA, err := ioutil.ReadFile(filepath.Join(testRoot, "testdata", jsonFileA))
	if err != nil {
		b.Errorf("ioutil.ReadFile(%s): could not open file: %v", jsonFileA, err)
		return
	}
	deviceA := &oc.Device{}
	if err := oc.Unmarshal(jsonA, deviceA); err != nil {
		b.Errorf("ioutil.ReadFile(%s): could unmarschal: %v", jsonFileA, err)
		return
	}

	jsonB, err := ioutil.ReadFile(filepath.Join(testRoot, "testdata", jsonFileB))
	if err != nil {
		b.Errorf("ioutil.ReadFile(%s): could not open file: %v", jsonFileB, err)
		return
	}
	deviceB := &oc.Device{}
	if err := oc.Unmarshal(jsonB, deviceB); err != nil {
		b.Errorf("ioutil.ReadFile(%s): could unmarschal: %v", jsonFileB, err)
		return
	}
	_, err = ygot.Diff(deviceA, deviceB)
	if err != nil {
		b.Errorf("Error in diff %v", err)
		return
	}
}
