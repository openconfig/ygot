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

// Binary uncompressed is an example package showing the usage of ygot for
// an uncompressed schema.
package main

import (
	"fmt"

	yb "github.com/openconfig/ygot/demo/uncompressed/pkg/demo"
	"github.com/openconfig/ygot/ygot"
)

// Generate rule to create the example structs:
//go:generate go run ../../generator/generator.go -path=yang -output_file=pkg/demo/uncompressed.go -package_name=demo -generate_fakeroot -fakeroot_name=root yang/example.yang

func main() {
	e, err := BuildDemo()
	if err != nil {
		panic(err)
	}

	ij, err := DemoInternalJSON(e)
	if err != nil {
		panic(fmt.Sprintf("Internal error: %v", err))
	}
	fmt.Println(ij)

	rj, err := DemoRFC7951JSON(e)
	if err != nil {
		panic(fmt.Sprintf("RFC7951 error: %v", err))
	}
	fmt.Println(rj)
}

// BuildDemo populates a demo instance of the uncompressed GoStructs
// for the example.yang module.
func BuildDemo() (*yb.Root, error) {
	d := &yb.Root{
		Person: ygot.String("robjs"),
	}
	uk, err := d.NewCountry("United Kingdom")
	if err != nil {
		return nil, err
	}
	uk.CountryCode = ygot.String("GB")
	uk.DialCode = ygot.Uint32(44)

	c2, err := d.NewOperator(29636)
	if err != nil {
		return nil, err
	}
	c2.Name = ygot.String("Catalyst2")

	if err := d.Validate(); err != nil {
		return nil, err
	}

	return d, nil
}

// DemoInternalJSON returns internal format JSON for the input
// ucompressed root struct d.
func DemoInternalJSON(d *yb.Root) (string, error) {
	json, err := ygot.EmitJSON(d, &ygot.EmitJSONConfig{
		Format: ygot.Internal,
		Indent: "  ",
	})
	if err != nil {
		return "", err
	}
	return json, nil
}

// DemoRFC7951JSON returns RFC7951 JSON for the input uncompressed
// root struct d.
func DemoRFC7951JSON(d *yb.Root) (string, error) {
	json, err := ygot.EmitJSON(d, &ygot.EmitJSONConfig{
		Format: ygot.RFC7951,
		Indent: "  ",
		RFC7951Config: &ygot.RFC7951JSONConfig{
			AppendModuleName: true,
		},
	})
	if err != nil {
		return "", err
	}
	return json, nil

}
