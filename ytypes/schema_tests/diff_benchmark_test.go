package validate

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/idefixcert/ygot/ygot"
	oc "github.com/openconfig/ygot/exampleoc"
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
	err = oc.Unmarshal(jsonA, deviceA)
	if err != nil {
		b.Errorf("ioutil.ReadFile(%s): could unmarschal: %v", jsonFileA, err)
		return
	}

	jsonB, err := ioutil.ReadFile(filepath.Join(testRoot, "testdata", jsonFileB))
	if err != nil {
		b.Errorf("ioutil.ReadFile(%s): could not open file: %v", jsonFileB, err)
		return
	}
	deviceB := &oc.Device{}
	err = oc.Unmarshal(jsonB, deviceB)
	if err != nil {
		b.Errorf("ioutil.ReadFile(%s): could unmarschal: %v", jsonFileB, err)
		return
	}
	_, err = ygot.Diff(deviceA, deviceB)
	if err != nil {
		b.Errorf("Error in diff %v", err)
		return
	}
}
