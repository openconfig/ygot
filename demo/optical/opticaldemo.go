// Copyright 2017 Google Inc.
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

// Binary devicedemo provides a demonstration application which uses the OpenConfig
// structs library to create a data instance of an entire device, and output it as
// JSON.
package main

import (
	"fmt"

	log "github.com/golang/glog"
	oc "github.com/openconfig/ygot/exampleoc"
	"github.com/openconfig/ygot/ygot"
)

func main() {
	i, err := CreateDemoOpticalInstance()
	if err != nil {
		log.Exitf("Error in OpenConfig optical demo instance creation: %v", err)
	}

	json, err := OutputJSON(i)
	if err != nil {
		log.Exitf("Error in OpenConfig optical demo: %v", err)
	}
	fmt.Println(json)

	ietfjson, err := OutputIETFJSON(i)
	if err != nil {
		log.Exitf("Error in IETF JSON demo: %v", err)
	}
	fmt.Println(ietfjson)
}

// apsModuleInput stores a high-level spec for an APS module.
type apsModuleInput struct {
	Name      string
	Hyst      []float64
	Thresh    []float64
	Revertive bool
}

// CreateDemoOpticalInstance creates an example optical device instance
// and returns the fakeroot Device struct which can be handled by the
// calling function. Returns an error if the device cannot be created.
func CreateDemoOpticalInstance() (*oc.Device, error) {
	d := &oc.Device{
		Aps: &oc.Aps{},
		Component:        map[string]*oc.Component{
			"mod-one": &oc.Component{Name: ygot.String("mod-one")},
			"mod-two": &oc.Component{Name: ygot.String("mod-two")},
		},
	}

	modules := []*apsModuleInput{{
		Name:      "mod-one",
		Hyst:      []float64{-42.42, -84.84},
		Thresh:    []float64{-96.96, -128.128},
		Revertive: false,
	}, {
		Name:      "mod-two",
		Hyst:      []float64{-42.42},
		Thresh:    []float64{-84.84},
		Revertive: true,
	}}

	for _, m := range modules {
		a, err := d.Aps.NewApsModule(m.Name)
		if err != nil {
			return nil, err
		}

		a.Revertive = ygot.Bool(m.Revertive)
		if len(m.Hyst) >= 1 {
			a.PrimarySwitchHysteresis = ygot.Float64(m.Hyst[0])
		}

		if len(m.Hyst) > 1 {
			a.SecondarySwitchHysteresis = ygot.Float64(m.Hyst[1])
		}

		if len(m.Thresh) >= 1 {
			a.PrimarySwitchThreshold = ygot.Float64(m.Thresh[0])
		}

		if len(m.Thresh) > 1 {
			a.SecondarySwitchThreshold = ygot.Float64(m.Thresh[1])
		}
	}
	return d, nil
}

// OutputJSON uses the legacy library to output JSON that is not RFC
// 7951 compliant.
func OutputJSON(d *oc.Device) (string, error) {
	j, err := ygot.EmitJSON(d, nil)
	if err != nil {
		return "", fmt.Errorf("got errors: %v", err)
	}
	return j, nil
}

// OutputIETFJSON uses the new validation approach to output JSON that
// is RFC7951 compliant.
func OutputIETFJSON(d *oc.Device) (string, error) {
	ietfj, err := ygot.EmitJSON(d, &ygot.EmitJSONConfig{
		Format: ygot.RFC7951,
		RFC7951Config: &ygot.RFC7951JSONConfig{
			AppendModuleName: true,
		},
	})

	if err != nil {
		return "", err
	}

	return ietfj, nil
}
