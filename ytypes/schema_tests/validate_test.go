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

package validate

import (
	"testing"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygot"

	oc "github.com/openconfig/ygot/exampleoc"
)

var (
	// testErrOutput controls whether expect error test cases log the error
	// values.
	testErrOutput = false
)

// testErrLog logs err to t if err != nil and global value testErrOutput is set.
func testErrLog(t *testing.T, desc string, err error) {
	if err != nil {
		if testErrOutput {
			t.Logf("%s: %v", desc, err)
		}
	}
}

// To debug a schema node subtree, any of the following can be used:
//
// 1. Print hierarchy without details (good for viewing large subtrees):
//      fmt.Println(schemaTreeString(oc.SchemaTree["LocalRoutes_Static"], ""))
//
// 2. Print in-memory structure representations. Replace is needed due to large
//    default indentations:
//      fmt.Println(strings.Replace(pretty.Sprint(oc.SchemaTree["LocalRoutes_Static"].Dir["next-hops"].Dir["next-hop"].Dir["config"].Dir["next-hop"])[0:], "              ", "  ", -1))
//
// 3. Detailed representation in JSON format:
//      j, _ := json.MarshalIndent(oc.SchemaTree["LocalRoutes_Static"].Dir["next-hops"].Dir["next-hop"].Dir["config"].Dir["next-hop"], "", "  ")
//      fmt.Println(string(j))

func TestValidateInterface(t *testing.T) {
	dev := &oc.Device{}
	eth0, err := dev.NewInterface("eth0")
	if err != nil {
		t.Errorf("eth0.NewInterface(): got %v, want nil", err)
	}

	eth0.Description = ygot.String("eth0 description")
	eth0.Type = oc.IETFInterfaces_InterfaceType_ethernetCsmacd

	// Validate the fake root device.
	if err := dev.Validate(); err != nil {
		t.Errorf("root success: got %s, want nil", err)
	}
	// Validate an element in the device subtree.
	if err := eth0.Validate(); err != nil {
		t.Errorf("eth0 success: got %s, want nil", err)
	}

	// Key in map != key field value in element. Key should be "eth0" here.
	dev.Interface["bad_key"] = eth0
	if err := dev.Validate(); err == nil {
		t.Errorf("bad key: got nil, want error")
	} else {
		testErrLog(t, "bad key", err)
	}

	vlan0, err := eth0.NewSubinterface(0)
	if err != nil {
		t.Errorf("eth0.NewSubinterface(): got %v, want nil", err)
	}

	// Device/interface/subinterfaces/subinterface/vlan
	vlan0.Vlan = &oc.Interface_Subinterface_Vlan{
		VlanId: &oc.Interface_Subinterface_Vlan_VlanId_Union_Uint16{
			Uint16: 1234,
		},
	}

	// Validate the vlan.
	if err := vlan0.Validate(); err != nil {
		t.Errorf("vlan0 success: got %s, want nil", err)
	}

	// Set vlan-id to be out of range (1-4094)
	vlan0.Vlan = &oc.Interface_Subinterface_Vlan{
		VlanId: &oc.Interface_Subinterface_Vlan_VlanId_Union_Uint16{
			Uint16: 4095,
		},
	}
	// Validate the vlan.
	if err := vlan0.Validate(); err == nil {
		t.Errorf("bad vlan-id value: got nil, want error")
	} else {
		testErrLog(t, "bad vlan-id value", err)
	}
}

func TestValidateSystemDns(t *testing.T) {
	dev := &oc.Device{
		System: &oc.System{
			Dns: &oc.System_Dns{
				Server: map[string]*oc.System_Dns_Server{
					"10.10.10.10": {
						Address: ygot.String("10.10.10.10"),
					},
				},
			},
		},
	}

	// Validate the fake root device.
	if err := dev.Validate(); err != nil {
		t.Errorf("root success: got %s, want nil", err)
	}
	// Validate an element in the device subtree.
	if err := dev.System.Validate(); err != nil {
		t.Errorf("system success: got %s, want nil", err)
	}

	// Key in map != key field value in element.
	dev.System.Dns.Server["bad_key"] = &oc.System_Dns_Server{Address: ygot.String("server1")}
	if err := dev.Validate(); err == nil {
		t.Errorf("bad key: got nil, want error")
	} else {
		testErrLog(t, "bad key", err)
	}
}

func TestValidateLLDP(t *testing.T) {
	dev := &oc.Device{
		Lldp: &oc.Lldp{
			ChassisId: ygot.String("ch1"),
		},
	}

	intf, err := dev.Lldp.NewInterface("eth0")
	if err != nil {
		t.Fatalf("LLDP failure: could not create interface: %v", err)
	}

	neigh, err := intf.NewNeighbor("n1")
	if err != nil {
		t.Fatalf("LLDP failure: could not create neighbor: %v", err)
	}

	tlv, err := neigh.NewTlv(42, "oui", "oui-sub")
	if err != nil {
		t.Fatalf("LLDP failure: could not create TLV: %v", err)
	}

	tlv.Value = []byte{42, 42}

	// Validate the TLV
	if err := tlv.Validate(); err != nil {
		t.Errorf("LLDP failure: got TLV validation errors: %s", err)
	}

	// Validate the fake root device.
	if err := dev.Validate(); err != nil {
		t.Errorf("LLDP failure: got device validation errors: %s", err)
	}
}

func TestValidateSystemNtp(t *testing.T) {
	dev := &oc.Device{
		System: &oc.System{
			Ntp: &oc.System_Ntp{
				Server: map[string]*oc.System_Ntp_Server{
					"10.10.10.10": {
						Address: ygot.String("10.10.10.10"),
						Version: ygot.Uint8(1),
					},
				},
			},
		},
	}

	// Validate the fake root device.
	if err := dev.Validate(); err != nil {
		t.Errorf("root success: got %s, want nil", err)
	}

	// Key in map != key field value in element.
	dev.System.Ntp.Server["10.10.10.10"].Version = ygot.Uint8(5)
	if err := dev.Validate(); err == nil {
		t.Errorf("bad version: got nil, want error")
	} else {
		testErrLog(t, "bad version", err)
	}
}

func TestValidateNetworkInstance(t *testing.T) {
	// Struct key: schema Key is compound key "identifier name"
	instance1protocol1Key := oc.NetworkInstance_Protocol_Key{
		Identifier: oc.OpenconfigPolicyTypes_INSTALL_PROTOCOL_TYPE_BGP,
		Name:       "protocol1",
	}
	dev := &oc.Device{
		NetworkInstance: map[string]*oc.NetworkInstance{
			"instance1": {
				Name: ygot.String("instance1"),
				Protocol: map[oc.NetworkInstance_Protocol_Key]*oc.NetworkInstance_Protocol{
					instance1protocol1Key: {
						Identifier: oc.OpenconfigPolicyTypes_INSTALL_PROTOCOL_TYPE_BGP,
						Name:       ygot.String("protocol1"),
					},
				},
			},
		},
	}

	// Validate the fake root device.
	if err := dev.Validate(); err != nil {
		t.Errorf("root success: got %s, want nil", err)
	}

	// Key in map != key field value in element.
	dev.NetworkInstance["instance1"].Protocol[instance1protocol1Key].Name = ygot.String("bad_name")
	if err := dev.Validate(); err == nil {
		t.Errorf("bad element key field: got nil, want error")
	} else {
		testErrLog(t, "bad element key field", err)
	}
}

func TestValidateBGP(t *testing.T) {
	d := &oc.Device{
		Bgp: &oc.Bgp{
			Global: &oc.Bgp_Global{
				As: ygot.Uint32(15169),
				Confederation: &oc.Bgp_Global_Confederation{
					MemberAs: []uint32{65497, 65498},
				},
			},
		},
	}

	// Validate the fake root device.
	if err := d.Validate(); err != nil {
		t.Errorf("root success: got %s, want nil", err)
	}
}

func TestValidateLocalRoutes(t *testing.T) {
	// This schema element contains a union of union.
	lrs := &oc.LocalRoutes_Static{
		NextHop: map[string]*oc.LocalRoutes_Static_NextHop{
			"10.10.10.10": {
				Index: ygot.String("10.10.10.10"),
				NextHop: &oc.LocalRoutes_Static_NextHop_NextHop_Union_String{
					String: "10.10.10.1",
				},
			},
		},
	}

	// Validate the local static route.
	if err := lrs.Validate(); err != nil {
		t.Errorf("success: got %s, want nil", err)
	}
}

func TestValidateRoutingPolicy(t *testing.T) {
	/* Device subtree built from the following subtree in generated structs:

	   // RoutingPolicy represents the /openconfig-routing-policy/routing-policy YANG schema element.
	   type RoutingPolicy struct {
	   	DefinedSets      *RoutingPolicy_DefinedSets                 `path:"routing-policy/defined-sets"`
	   	PolicyDefinition map[string]*RoutingPolicy_PolicyDefinition `path:"routing-policy/policy-definitions/policy-definition"`
	   }

	   // RoutingPolicy_DefinedSets represents the /openconfig-routing-policy/routing-policy/defined-sets YANG schema element.
	   type RoutingPolicy_DefinedSets struct {
	   	NeighborSet map[string]*RoutingPolicy_DefinedSets_NeighborSet `path:"neighbor-sets/neighbor-set"`
	   	PrefixSet   map[string]*RoutingPolicy_DefinedSets_PrefixSet   `path:"prefix-sets/prefix-set"`
	   	TagSet      map[string]*RoutingPolicy_DefinedSets_TagSet      `path:"tag-sets/tag-set"`
	   }

	   type RoutingPolicy_DefinedSets_PrefixSet_Prefix_Key struct {
	   	IpPrefix        string
	   	MasklengthRange string
	   }

	   // RoutingPolicy_DefinedSets_PrefixSet represents the /openconfig-routing-policy/routing-policy/defined-sets/prefix-sets/prefix-set YANG schema element.
	   type RoutingPolicy_DefinedSets_PrefixSet struct {
	   	Prefix        map[RoutingPolicy_DefinedSets_PrefixSet_Prefix_Key]*RoutingPolicy_DefinedSets_PrefixSet_Prefix `path:"prefixes/prefix"`
	   	PrefixSetName *string                                                                                        `path:"config/prefix-set-name|prefix-set-name"`
	   }
	*/

	prefixKey1 := oc.RoutingPolicy_DefinedSets_PrefixSet_Prefix_Key{
		IpPrefix:        "255.255.255.0/20",
		MasklengthRange: "20..24",
	}
	dev := &oc.Device{
		RoutingPolicy: &oc.RoutingPolicy{
			DefinedSets: &oc.RoutingPolicy_DefinedSets{
				PrefixSet: map[string]*oc.RoutingPolicy_DefinedSets_PrefixSet{
					"prefix1": {
						PrefixSetName: ygot.String("prefix1"),
						Prefix: map[oc.RoutingPolicy_DefinedSets_PrefixSet_Prefix_Key]*oc.RoutingPolicy_DefinedSets_PrefixSet_Prefix{
							prefixKey1: {
								IpPrefix:        ygot.String("255.255.255.0/20"),
								MasklengthRange: ygot.String("20..24"),
							},
						},
					},
				},
			},
		},
	}

	// Validate the fake root device.
	if err := dev.Validate(); err != nil {
		t.Errorf("root success: got %s, want nil", err)
	}

	// MasklengthRange is a regex:
	// leaf masklength-range {
	// type string {
	//    pattern '^([0-9]+\.\.[0-9]+)|exact$';
	// }
	dev.RoutingPolicy.DefinedSets.PrefixSet["prefix1"].Prefix[prefixKey1].MasklengthRange = ygot.String("bad_element_key")
	if err := dev.Validate(); err == nil {
		t.Errorf("bad regex: got nil, want error")
	} else {
		testErrLog(t, "bad regex", err)
	}
}

// TODO(mostrowski): move below funtions into a helper package, or from common
// library when one is created.

// schemaTreeString returns the schema hierarchy tree as a string with node
// names and types only e.g.
// clock (container)
//   timezone (choice)
//     timezone-name (case)
//       timezone-name (leaf)
//     timezone-utc-offset (case)
//       timezone-utc-offset (leaf)
func schemaTreeString(schema *yang.Entry, prefix string) string {
	out := prefix + schema.Name + " (" + schemaTypeStr(schema) + ")" + "\n"
	for _, ch := range schema.Dir {
		out += schemaTreeString(ch, prefix+"  ")
	}
	return out
}

// schemaTypeStr returns a string representation of the type of element schema
// represents e.g. "container", "choice" etc.
func schemaTypeStr(schema *yang.Entry) string {
	switch {
	case schema.IsChoice():
		return "choice"
	case schema.IsContainer():
		return "container"
	case schema.IsCase():
		return "case"
	case schema.IsList():
		return "list"
	case schema.IsLeaf():
		return "leaf"
	case schema.IsLeafList():
		return "leaf-list"
	default:
	}
	return "other"
}
