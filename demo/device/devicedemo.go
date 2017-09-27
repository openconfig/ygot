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
	dev, err := CreateDemoDeviceInstance()
	if err != nil {
		log.Exitf("Error in OpenConfig device demo: %v", err)
	}

	json, err := EmitJSON(dev)
	if err != nil {
		log.Exitf("Error outputting device to JSON: %v", err)
	}
	fmt.Println(json)

	ietfjson, err := EmitRFC7951JSON(dev)
	if err != nil {
		log.Exitf("Error outtputing device to RFC7951 JSON: %v", err)
	}
	fmt.Println(ietfjson)
}

// CreateDemoDeviceInstance creates an example instance of the OpenConfig 'device'
// construct, demonstrating the population of fields along with the use of the fake
// root entity 'device' which does not exist in the YANG schema.
func CreateDemoDeviceInstance() (*oc.Device, error) {
	// Initialize a device.
	d := &oc.Device{
		System: &oc.System{
			Hostname: ygot.String("rtr02.pop44"),
		},
	}

	// Create a new interface under the device. In this case /interfaces/interface
	// is the list that is being populated, but due to schema compression the
	// 'interfaces' container is not created, making the 'interface' list a top-level
	// entity. The New... helper methods are therefore mapped to device.
	eth0, err := d.NewInterface("eth0")
	if err != nil {
		return nil, err
	}

	// Set some attributes of the interface.
	eth0.Description = ygot.String("Link to rtr01.pop44")
	eth0.Type = oc.IETFInterfaces_InterfaceType_ethernetCsmacd

	if err := addNetworkInstance(d); err != nil {
		return nil, err
	}

	// Add a component.
	c, err := d.NewComponent("os")
	if err != nil {
		return nil, err
	}
	c.Type = &oc.Component_Type_Union_E_OpenconfigPlatformTypes_OPENCONFIG_SOFTWARE_COMPONENT{oc.OpenconfigPlatformTypes_OPENCONFIG_SOFTWARE_COMPONENT_OPERATING_SYSTEM}

	return d, nil
}

// EmitJSON outputs the device instance specified as internal format JSON.
func EmitJSON(d *oc.Device) (string, error) {
	return ygot.EmitJSON(d, nil)
}

// EmitRFC7951JSON outputs the device instance specified as RFC7951 compliant
// JSON.
func EmitRFC7951JSON(d *oc.Device) (string, error) {
	return ygot.EmitJSON(d, &ygot.EmitJSONConfig{
		Format: ygot.RFC7951,
		RFC7951Config: &ygot.RFC7951JSONConfig{
			AppendModuleName: true,
		},
	})
}

// addNetworkInstance adds network instance content to a device.
func addNetworkInstance(d *oc.Device) error {
	netinst, err := d.NewNetworkInstance("DEFAULT")
	if err != nil {
		return err
	}

	p, err := netinst.NewProtocol(oc.OpenconfigPolicyTypes_INSTALL_PROTOCOL_TYPE_BGP, "15169")
	if err != nil {
		return err
	}

	ygot.BuildEmptyTree(p)
	p.Bgp.Global.As = ygot.Uint32(15169)

	return nil
}
