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

// Binary interfaces is a getting started example for the ygot library.
package main

import (
	"fmt"

	oc "github.com/openconfig/ygot/demo/getting_started/pkg/ocdemo"
	"github.com/openconfig/ygot/ygot"
)

// The following generation rule uses the generator binary to create the
// pkg/ocdemo package, which generates the corresponding code for OpenConfig
// interfaces.
//
//go:generate go run ../../generator/generator.go -path=yang -output_file=pkg/ocdemo/oc.go -package_name=ocdemo -generate_fakeroot -fakeroot_name=device -compress_paths=true  -exclude_modules=ietf-interfaces yang/openconfig-interfaces.yang yang/openconfig-if-ip.yang

func main() {
	// Create a new device which is named according to the fake root specified above. To generate
	// the fakeroot then generate_fakeroot should be specified. This entity corresponds to the
	// root of the YANG schema tree. The fakeroot name is the CamelCase version of the name
	// supplied by the fakeroot_name argument.
	d := &oc.Device{}

	// To render the device (which is currently empty) to JSON in RFC7951 format, then we
	// simply call the ygot.EmitJSON method with the relevant arguments.
	j, err := ygot.EmitJSON(d, &ygot.EmitJSONConfig{
		Format: ygot.RFC7951,
		Indent: "  ",
		RFC7951Config: &ygot.RFC7951JSONConfig{
			AppendModuleName: true,
		},
	})

	// If an error was returned (which occurs if the struct's contents could not be validated
	// or an error occurred with rendering to JSON), then this should be handled by the
	// calling code.
	if err != nil {
		panic(err)
	}
	fmt.Printf("Empty JSON: %v\n", j)

	// Since compress_paths is set to true, then the /interfaces/interface list is compressed to
	// be Interface at the root of the tree, such that the root has a NewInterface method.
	i, err := d.NewInterface("eth0")

	// We can now work directly with the returned interface to specify some values.
	i.AdminStatus = oc.OpenconfigInterfaces_Interface_AdminStatus_UP
	i.Mtu = ygot.Uint16(1500)
	i.Description = ygot.String("An Interface")

	// We can then validate the contents of the interface that we created.
	if err := d.Interface["eth0"].Validate(); err != nil {
		panic(fmt.Sprintf("Interface validation failed: %v", err))
	}

	// We can also directly create an interface within the device.
	d.Interface["eth1"] = &oc.Interface{
		Name:        ygot.String("eth1"),
		Description: ygot.String("Another Interface"),
		Enabled:     ygot.Bool(false),
		Type:        oc.IETFInterfaces_InterfaceType_ethernetCsmacd,
	}

	s, err := d.Interface["eth1"].NewSubinterface(0)
	if err != nil {
		panic(fmt.Sprintf("Duplicate subinterface: %v", err))
	}

	// BuildEmptyTree initialises all subcontainers of a particular
	// point in the tree.
	ygot.BuildEmptyTree(s)

	// Loop through and add addresses on a particular interface.
	addresses := []struct {
		address string
		mask    uint8
	}{{
		address: "192.0.2.1",
		mask:    24,
	}, {
		address: "10.0.42.1",
		mask:    8,
	}}

	for _, addr := range addresses {
		a, err := s.Ipv4.NewAddress(addr.address)
		if err != nil {
			panic(err)
		}
		a.PrefixLength = ygot.Uint8(addr.mask)
	}

	// When we have invalid data (e.g., a non-matching IPv4 address) then .Validate()
	// returns an error.
	invalidIf := &oc.Interface{
		Name: ygot.String("eth42"),
	}
	subif, err := invalidIf.NewSubinterface(0)
	if err != nil {
		panic(err)
	}
	ygot.BuildEmptyTree(subif)
	_, err = subif.Ipv4.NewAddress("Not a valid address")
	if err := invalidIf.Validate(); err == nil {
		panic(fmt.Sprintf("Did not find invalid address, got nil err: %v", err))
	} else {
		fmt.Printf("Got expected error: %v\n", err)
	}

	// We can also validate the device overall.
	if err := d.Validate(); err != nil {
		panic(fmt.Sprintf("Device validation failed: %v", err))
	}

	// EmitJSON from the ygot library directly does .Validate() and outputs JSON in
	// the specified format.
	json, err := ygot.EmitJSON(d, &ygot.EmitJSONConfig{
		Format: ygot.RFC7951,
		Indent: "  ",
		RFC7951Config: &ygot.RFC7951JSONConfig{
			AppendModuleName: true,
		},
	})
	if err != nil {
		panic(fmt.Sprintf("JSON demo error: %v", err))
	}
	fmt.Println(json)

	// The generated code includes an Unmarshal function, which can be used to load
	// a data tree such as the one that we just created.
	loadd := &oc.Device{}
	if err := oc.Unmarshal([]byte(json), loadd); err != nil {
		panic(fmt.Sprintf("Can't unmarshal JSON: %v", err))
	}
}
