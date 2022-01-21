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

// TODO(mostrowski): create tests against an uncompressed schema.
package validate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/openconfig/ygot/testutil"
	"github.com/openconfig/ygot/util"
	"github.com/openconfig/ygot/ygot"
	"github.com/openconfig/ygot/ytypes"

	"github.com/openconfig/gnmi/errdiff"
	oc "github.com/openconfig/ygot/exampleoc"
	"github.com/openconfig/ygot/exampleoc/opstateoc"
	woc "github.com/openconfig/ygot/exampleoc/wrapperunionoc"
	uoc "github.com/openconfig/ygot/uexampleoc"
)

// To debug a schema node subtree, any of the following can be used:
//
// 1. Print hierarchy without details (good for viewing large subtrees):
//      fmt.Println(schemaTreeString(oc.SchemaTree["LocalRoutes_Static"], ""))
//
// 2. Print in-memory structure representations. Replace is needed due to large
//    default util.Indentations:
//      fmt.Println(strings.Replace(pretty.Sprint(oc.SchemaTree["LocalRoutes_Static"].Dir["next-hops"].Dir["next-hop"].Dir["config"].Dir["next-hop"])[0:], "              ", "  ", -1))
//
// 3. Detailed representation in JSON format:
//      j, _ := json.MarshalIndent(oc.SchemaTree["LocalRoutes_Static"].Dir["next-hops"].Dir["next-hop"].Dir["config"].Dir["next-hop"], "", "  ")
//      fmt.Println(string(j))
//
// 4. Combination of schema and data trees:
//       fmt.Println(ytypes.DataSchemaTreesString(oc.SchemaTree["Device"], dev))
//
// 5. Entire schema only in JSON format:
//       j, _ := json.MarshalIndent(oc.SchemaTree, "", "  ")
//	     fmt.Println(string(j))
//

const (
	// TestRoot is the path to the directory within which the test runs, appended
	// to any filename that is to be loaded.
	testRoot string = "."
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

// errToString returns the string representation of err and the empty string if
// err is nil.
func errToString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

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
		if diff := errdiff.Substring(err, "/device/interfaces/interface: key field Name: element key eth0 != map key bad_key"); diff != "" {
			t.Errorf("did not get expected vlan-id error, %s", diff)
		}
		testErrLog(t, "bad key", err)
	}

	vlan0, err := eth0.NewSubinterface(0)
	if err != nil {
		t.Errorf("eth0.NewSubinterface(): got %v, want nil", err)
	}

	// Device/interface/subinterfaces/subinterface/vlan
	vlan0.Vlan = &oc.Interface_Subinterface_Vlan{
		VlanId: oc.UnionUint16(1234),
	}

	// Validate the vlan.
	if err := vlan0.Validate(); err != nil {
		t.Errorf("vlan0 success: got %s, want nil", err)
	}

	// Set vlan-id to be out of range (1-4094)
	vlan0.Vlan = &oc.Interface_Subinterface_Vlan{
		VlanId: oc.UnionUint16(4095),
	}
	// Validate the vlan.
	err = vlan0.Validate()
	if diff := errdiff.Substring(err, `/device/interfaces/interface/subinterfaces/subinterface/vlan/config/vlan-id: schema "": unsigned integer value 4095 is outside specified ranges`); diff != "" {
		t.Errorf("did not get expected vlan-id error, %s", diff)
	}
	if err != nil {
		testErrLog(t, "bad vlan-id value", err)
	}

	// Validate that we get two errors.
	if errs := dev.Validate(); len(errs.(util.Errors)) != 2 {
		var b bytes.Buffer
		for _, err := range errs.(util.Errors) {
			b.WriteString(fmt.Sprintf("	[%s]\n", err))
		}
		t.Errorf("did not get expected errors when validating device, got:\n %s (len: %d), want 5 errors", b.String(), len(errs.(util.Errors)))
	}
}

func TestValidateInterfaceWrapperUnion(t *testing.T) {
	dev := &woc.Device{}
	eth0, err := dev.NewInterface("eth0")
	if err != nil {
		t.Errorf("dev.NewInterface(): got %v, want nil", err)
	}

	eth0.Description = ygot.String("eth0 description")
	eth0.Type = woc.IETFInterfaces_InterfaceType_ethernetCsmacd

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
	err = dev.Validate()
	if diff := errdiff.Substring(err, "/device/interfaces/interface: key field Name: element key eth0 != map key bad_key"); diff != "" {
		t.Errorf("did not get expected vlan-id error, %s", diff)
	}
	if err != nil {
		testErrLog(t, "bad key", err)
	}

	vlan0, err := eth0.NewSubinterface(0)
	if err != nil {
		t.Errorf("eth0.NewSubinterface(): got %v, want nil", err)
	}

	// Device/interface/subinterfaces/subinterface/vlan
	vlan0.Vlan = &woc.Interface_Subinterface_Vlan{
		VlanId: &woc.Interface_Subinterface_Vlan_VlanId_Union_Uint16{
			Uint16: 1234,
		},
	}

	// Validate the vlan.
	if err := vlan0.Validate(); err != nil {
		t.Errorf("vlan0 success: got %s, want nil", err)
	}

	// Set vlan-id to be out of range (1-4094)
	vlan0.Vlan = &woc.Interface_Subinterface_Vlan{
		VlanId: &woc.Interface_Subinterface_Vlan_VlanId_Union_Uint16{
			Uint16: 4095,
		},
	}
	// Validate the vlan.
	if err := vlan0.Validate(); err == nil {
		t.Errorf("bad vlan-id value: got nil, want error")
	} else {
		if diff := errdiff.Substring(err, `/device/interfaces/interface/subinterfaces/subinterface/vlan/config/vlan-id: schema "": unsigned integer value 4095 is outside specified ranges`); diff != "" {
			t.Errorf("did not get expected vlan-id error, %s", diff)
		}
		testErrLog(t, "bad vlan-id value", err)
	}

	// Validate that we get two errors.
	if errs := dev.Validate(); len(errs.(util.Errors)) != 2 {
		var b bytes.Buffer
		for _, err := range errs.(util.Errors) {
			b.WriteString(fmt.Sprintf("	[%s]\n", err))
		}
		t.Errorf("did not get expected errors when validating device, got:\n %s (len: %d), want 5 errors", b.String(), len(errs.(util.Errors)))
	}
}

func TestValidateInterfaceOpState(t *testing.T) {
	dev := &opstateoc.Device{}
	eth0, err := dev.NewInterface("eth0")
	if err != nil {
		t.Errorf("eth0.NewInterface(): got %v, want nil", err)
	}

	eth0.Description = ygot.String("eth0 description")
	eth0.Type = opstateoc.IETFInterfaces_InterfaceType_ethernetCsmacd

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
		if diff := errdiff.Substring(err, "/device/interfaces/interface: key field Name: element key eth0 != map key bad_key"); diff != "" {
			t.Errorf("did not get expected vlan-id error, %s", diff)
		}
		testErrLog(t, "bad key", err)
	}

	vlan0, err := eth0.NewSubinterface(0)
	if err != nil {
		t.Errorf("eth0.NewSubinterface(): got %v, want nil", err)
	}

	// Device/interface/subinterfaces/subinterface/vlan
	vlan0.Vlan = &opstateoc.Interface_Subinterface_Vlan{
		VlanId: opstateoc.UnionUint16(1234),
	}

	// Validate the vlan.
	if err := vlan0.Validate(); err != nil {
		t.Errorf("vlan0 success: got %s, want nil", err)
	}

	// Set vlan-id to be out of range (1-4094)
	vlan0.Vlan = &opstateoc.Interface_Subinterface_Vlan{
		VlanId: opstateoc.UnionUint16(4095),
	}
	// Validate the vlan.
	if err := vlan0.Validate(); err == nil {
		t.Errorf("bad vlan-id value: got nil, want error")
	} else {
		if diff := errdiff.Substring(err, `/device/interfaces/interface/subinterfaces/subinterface/vlan/state/vlan-id: schema "": unsigned integer value 4095 is outside specified ranges`); diff != "" {
			t.Errorf("did not get expected vlan-id error, %s", diff)
		}
		testErrLog(t, "bad vlan-id value", err)
	}

	// Validate that we get two errors.
	if errs := dev.Validate(); len(errs.(util.Errors)) != 2 {
		var b bytes.Buffer
		for _, err := range errs.(util.Errors) {
			b.WriteString(fmt.Sprintf("	[%s]\n", err))
		}
		t.Errorf("did not get expected errors when validating device, got:\n %s (len: %d), want 5 errors", b.String(), len(errs.(util.Errors)))
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

func TestValidateSystemAaa(t *testing.T) {
	dev := &oc.Device{
		System: &oc.System{
			Aaa: &oc.System_Aaa{
				Authentication: &oc.System_Aaa_Authentication{
					AuthenticationMethod: []oc.System_Aaa_Authentication_AuthenticationMethod_Union{
						oc.AaaTypes_AAA_METHOD_TYPE_LOCAL,
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
}

func TestValidateSystemAaaWrapperUnion(t *testing.T) {
	dev := &woc.Device{
		System: &woc.System{
			Aaa: &woc.System_Aaa{
				Authentication: &woc.System_Aaa_Authentication{
					AuthenticationMethod: []woc.System_Aaa_Authentication_AuthenticationMethod_Union{
						&woc.System_Aaa_Authentication_AuthenticationMethod_Union_E_AaaTypes_AAA_METHOD_TYPE{
							E_AaaTypes_AAA_METHOD_TYPE: woc.AaaTypes_AAA_METHOD_TYPE_LOCAL,
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
	// Validate an element in the device subtree.
	if err := dev.System.Validate(); err != nil {
		t.Errorf("system success: got %s, want nil", err)
	}
}

func TestValidateLLDP(t *testing.T) {
	dev := &oc.Device{
		Lldp: &oc.Lldp{
			ChassisId: ygot.String("ch1"),
		},
	}
	_, err := dev.NewInterface("eth0")
	if err != nil {
		t.Errorf("eth0.NewInterface(): got %v, want nil", err)
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
		Identifier: oc.PolicyTypes_INSTALL_PROTOCOL_TYPE_BGP,
		Name:       "protocol1",
	}
	dev := &oc.Device{
		NetworkInstance: map[string]*oc.NetworkInstance{
			"instance1": {
				Name: ygot.String("instance1"),
				Protocol: map[oc.NetworkInstance_Protocol_Key]*oc.NetworkInstance_Protocol{
					instance1protocol1Key: {
						Identifier: oc.PolicyTypes_INSTALL_PROTOCOL_TYPE_BGP,
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
		NetworkInstance: map[string]*oc.NetworkInstance{
			"DEFAULT": {
				Name: ygot.String("DEFAULT"),
				Protocol: map[oc.NetworkInstance_Protocol_Key]*oc.NetworkInstance_Protocol{
					{
						Name:       "BGP",
						Identifier: oc.PolicyTypes_INSTALL_PROTOCOL_TYPE_BGP,
					}: {
						Name:       ygot.String("BGP"),
						Identifier: oc.PolicyTypes_INSTALL_PROTOCOL_TYPE_BGP,
						Bgp: &oc.NetworkInstance_Protocol_Bgp{
							Global: &oc.NetworkInstance_Protocol_Bgp_Global{
								As: ygot.Uint32(15169),
								Confederation: &oc.NetworkInstance_Protocol_Bgp_Global_Confederation{
									MemberAs: []uint32{65497, 65498},
								},
							},
						},
					},
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
				Index:   ygot.String("10.10.10.10"),
				NextHop: oc.UnionString("10.10.10.1"),
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
	   	Name *string                                                                                        `path:"config/prefix-set-name|prefix-set-name"`
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
						Name: ygot.String("prefix1"),
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
	badMaskLengthRange := "bad_element_key"
	prefixKey1.MasklengthRange = badMaskLengthRange
	dev.RoutingPolicy.DefinedSets.PrefixSet["prefix1"].Prefix[prefixKey1] = &oc.RoutingPolicy_DefinedSets_PrefixSet_Prefix{
		IpPrefix:        ygot.String("255.255.255.0/20"),
		MasklengthRange: ygot.String(badMaskLengthRange),
	}
	err := dev.Validate()
	if diff := errdiff.Substring(err, "does not match regular expression pattern"); diff != "" {
		t.Errorf("did not get expected bad regex error, %s", diff)
	} else {
		testErrLog(t, "bad regex", err)
	}
}

func TestUnmarshal(t *testing.T) {
	tests := []struct {
		desc              string
		jsonFilePath      string
		parent            ygot.ValidatedGoStruct
		opts              []ytypes.UnmarshalOpt
		unmarshalFn       ytypes.UnmarshalFunc
		wantValidationErr string
		wantErrSubstring  string
		outjsonFilePath   string // outjsonFilePath is the output JSON expected, when not specified it is assumed input == output.
	}{
		{
			desc:         "basic",
			jsonFilePath: "basic.json",
			parent:       &oc.Device{},
			unmarshalFn:  oc.Unmarshal,
		},
		{
			desc:         "bgp",
			jsonFilePath: "bgp-example.json",
			parent:       &oc.Device{},
			unmarshalFn:  oc.Unmarshal,
		},
		{
			desc:             "bgp, given shadow path but for schema that doesn't ignore shadow paths",
			jsonFilePath:     "bgp-example-opstate-with-shadow.json",
			parent:           &oc.Device{},
			unmarshalFn:      oc.Unmarshal,
			wantErrSubstring: "JSON contains unexpected field state",
		},
		{
			desc:            "bgp with prefer_operational_state, with schema ignoring shadow paths",
			jsonFilePath:    "bgp-example-opstate-with-shadow.json",
			parent:          &opstateoc.Device{},
			unmarshalFn:     opstateoc.Unmarshal,
			outjsonFilePath: "bgp-example-opstate.json",
		},
		{
			desc:              "interfaces",
			jsonFilePath:      "interfaces-example.json",
			parent:            &oc.Device{},
			unmarshalFn:       oc.Unmarshal,
			wantValidationErr: `validation err: field name AggregateId value Bundle-Ether22 (string ptr) schema path /device/interfaces/interface/ethernet/config/aggregate-id has leafref path /interfaces/interface/name not equal to any target nodes`,
		},
		{
			desc:         "local-routing",
			jsonFilePath: "local-routing-example.json",
			parent:       &oc.Device{},
			unmarshalFn:  oc.Unmarshal,
		},
		{
			desc:         "policy",
			jsonFilePath: "policy-example.json",
			parent:       &oc.Device{},
			unmarshalFn:  oc.Unmarshal,
		},
		{
			desc:            "basic with extra fields - ignored",
			jsonFilePath:    "basic-extra.json",
			parent:          &oc.Device{},
			unmarshalFn:     oc.Unmarshal,
			opts:            []ytypes.UnmarshalOpt{&ytypes.IgnoreExtraFields{}},
			outjsonFilePath: "basic.json",
		},
		{
			desc:             "basic with extra fields - not ignored",
			jsonFilePath:     "basic-extra.json",
			parent:           &oc.Device{},
			unmarshalFn:      oc.Unmarshal,
			wantErrSubstring: "JSON contains unexpected field",
		},
		{
			desc:             "extra leaf within a config subtree",
			jsonFilePath:     "basic-extra-config.json",
			parent:           &oc.Device{},
			unmarshalFn:      oc.Unmarshal,
			wantErrSubstring: "JSON contains unexpected field",
		},
		{
			desc:             "basic with extra fields - lower in tree",
			jsonFilePath:     "unexpected-ntp-invalid-leaf-when.json",
			parent:           &oc.Device{},
			unmarshalFn:      oc.Unmarshal,
			wantErrSubstring: "JSON contains unexpected field when",
		},
		{
			desc:            "relay agent leaf-list of single type union",
			jsonFilePath:    "relay-agent.json",
			parent:          &oc.Device{},
			unmarshalFn:     oc.Unmarshal,
			outjsonFilePath: "relay-agent.json",
		},
		{
			desc:            "unmarshal list with union key",
			jsonFilePath:    "system-cpu.json",
			parent:          &oc.Device{},
			unmarshalFn:     oc.Unmarshal,
			outjsonFilePath: "system-cpu.json",
		},
		{
			desc:            "unmarshal list with union key - uncompressed",
			jsonFilePath:    "system-cpu.json",
			parent:          &uoc.Device{},
			unmarshalFn:     uoc.Unmarshal,
			outjsonFilePath: "system-cpu.json",
		},
		{
			desc:            "relay agent leaf-list of single type union (wrapper union)",
			jsonFilePath:    "relay-agent.json",
			parent:          &woc.Device{},
			unmarshalFn:     woc.Unmarshal,
			outjsonFilePath: "relay-agent.json",
		},
		{
			desc:            "unmarshal list with union key (wrapper union)",
			jsonFilePath:    "system-cpu.json",
			parent:          &woc.Device{},
			unmarshalFn:     woc.Unmarshal,
			outjsonFilePath: "system-cpu.json",
		},
	}

	emitJSONConfig := &ygot.EmitJSONConfig{
		Format: ygot.RFC7951,
		RFC7951Config: &ygot.RFC7951JSONConfig{
			AppendModuleName: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {

			j, err := ioutil.ReadFile(filepath.Join(testRoot, "testdata", tt.jsonFilePath))
			if err != nil {
				t.Errorf("%s: ioutil.ReadFile(%s): could not open file: %v", tt.desc, tt.jsonFilePath, err)
				return
			}

			wantj := j
			if tt.outjsonFilePath != "" {
				rj, err := ioutil.ReadFile(filepath.Join(testRoot, "testdata", tt.outjsonFilePath))
				if err != nil {
					t.Errorf("%s: ioutil.ReadFile(%s): could not open file: %v", tt.desc, tt.outjsonFilePath, err)
				}
				wantj = rj
			}

			err = tt.unmarshalFn(j, tt.parent, tt.opts...)
			if diff := errdiff.Substring(err, tt.wantErrSubstring); diff != "" {
				t.Errorf("%s: did not get expected error: %s", tt.desc, diff)
			}
			testErrLog(t, tt.desc, err)
			if err == nil {
				jo, err := ygot.EmitJSON(tt.parent, emitJSONConfig)
				if got, want := errToString(err), tt.wantValidationErr; got != want {
					t.Errorf("%s: got error: %v, want error: %v ", tt.desc, got, want)
					return
				}
				if err != nil {
					return
				}
				d, err := diffJSON(wantj, []byte(jo))
				if err != nil {
					t.Fatal(err)
				}
				if d != "" {
					t.Errorf("%s: diff(-got,+want):\n%s", tt.desc, d)
				}
			}
		})
	}
}

/* TestLeafrefCurrent validates that the current() function works when
   leafrefs are validated in a real schema.
   It uses the following struct as the input:
   type Mpls_Global_Interface struct {
        InterfaceId  *string                             `path:"config/interface-id|interface-id" module:"openconfig-mpls/openconfig-mpls"`
        InterfaceRef *Mpls_Global_Interface_InterfaceRef `path:"interface-ref" module:"openconfig-mpls"`
        MplsEnabled  *bool                               `path:"config/mpls-enabled" module:"openconfig-mpls/openconfig-mpls"`
   }
   where the InterfaceRef container references an interface/subinterface
   in the /interfaces/interface list.
*/
func TestLeafrefCurrent(t *testing.T) {
	dev := &oc.Device{}
	ni := dev.GetOrCreateNetworkInstance("DEFAULT")

	i, err := dev.NewInterface("eth0")
	if err != nil {
		t.Fatalf("TestLeafrefCurrent: could not create new interface, got: %v, want error: nil", err)
	}
	if _, err := i.NewSubinterface(0); err != nil {
		t.Fatalf("TestLeafrefCurrent: could not create subinterface, got: %v, want error: nil", err)
	}

	ygot.BuildEmptyTree(ni)
	mi, err := ni.Mpls.Global.NewInterface("eth0.0")
	if err != nil {
		t.Fatalf("TestLeafrefCurrent: could not add new MPLS interface, got: %v, want error: nil", err)
	}
	mi.InterfaceRef = &oc.NetworkInstance_Mpls_Global_Interface_InterfaceRef{
		Interface:    ygot.String("eth0"),
		Subinterface: ygot.Uint32(0),
	}

	if err := dev.Validate(); err != nil {
		t.Fatalf("TestLeafrefCurrent: could not validate populated interfaces, got: %v, want: nil", err)
	}

	ni.Mpls.Global.Interface["eth0.0"].InterfaceRef.Subinterface = ygot.Uint32(1)
	if err := dev.Validate(); err == nil {
		t.Fatal("TestLeafrefCurrent: did not get expected error for non-existent subinterface, got: nil, want: error")
	}

	if err := dev.Validate(&ytypes.LeafrefOptions{IgnoreMissingData: true}); err != nil {
		t.Fatalf("TestLeafrefCurrent: did not get nil error when disabling leafref data validation, got: %v, want: nil", err)
	}
}

func diffJSON(a, b []byte) (string, error) {
	var aj, bj map[string]interface{}
	if err := json.Unmarshal(a, &aj); err != nil {
		return "", err
	}
	if err := json.Unmarshal(b, &bj); err != nil {
		return "", err
	}
	as, err := json.MarshalIndent(aj, "", "  ")
	if err != nil {
		return "", err
	}
	bs, err := json.MarshalIndent(bj, "", "  ")
	if err != nil {
		return "", err
	}

	asv, bsv := strings.Split(string(as), "\n"), strings.Split(string(bs), "\n")
	sort.Strings(asv)
	sort.Strings(bsv)

	return testutil.GenerateUnifiedDiff(strings.Join(asv, "\n"), strings.Join(bsv, "\n"))
}
