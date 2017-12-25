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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/openconfig/ygot/experimental/ygotutils"
	"github.com/openconfig/ygot/util"
	"github.com/openconfig/ygot/ygot"
	"github.com/openconfig/ygot/ytypes"
	"github.com/pmezard/go-difflib/difflib"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
	oc "github.com/openconfig/ygot/exampleoc"
	scpb "google.golang.org/genproto/googleapis/rpc/code"
	spb "google.golang.org/genproto/googleapis/rpc/status"
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

func TestValidateSystemAaa(t *testing.T) {
	dev := &oc.Device{
		System: &oc.System{
			Aaa: &oc.System_Aaa{
				Authentication: &oc.System_Aaa_Authentication{
					AuthenticationMethod: []oc.System_Aaa_Authentication_AuthenticationMethod_Union{
						&oc.System_Aaa_Authentication_AuthenticationMethod_Union_E_OpenconfigAaaTypes_AAA_METHOD_TYPE{
							E_OpenconfigAaaTypes_AAA_METHOD_TYPE: oc.OpenconfigAaaTypes_AAA_METHOD_TYPE_LOCAL,
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
	dev.RoutingPolicy.DefinedSets.PrefixSet["prefix1"].Prefix[prefixKey1].MasklengthRange = ygot.String("bad_element_key")
	if err := dev.Validate(); err == nil {
		t.Errorf("bad regex: got nil, want error")
	} else {
		testErrLog(t, "bad regex", err)
	}
}

func TestUnmarshal(t *testing.T) {
	tests := []struct {
		desc              string
		jsonFilePath      string
		parent            ygot.ValidatedGoStruct
		wantValidationErr string
		wantErr           string
	}{
		{
			desc:         "basic",
			jsonFilePath: "basic.json",
			parent:       &oc.Device{},
		},
		{
			desc:         "bgp",
			jsonFilePath: "bgp-example.json",
			parent:       &oc.Device{},
		},
		{
			desc:              "interfaces",
			jsonFilePath:      "interfaces-example.json",
			parent:            &oc.Device{},
			wantValidationErr: `validation err: field name AggregateId value Bundle-Ether22 (string ptr) schema path /device/interfaces/interface/ethernet/config/aggregate-id has leafref path /interfaces/interface/name not equal to any target nodes`,
		},
		{
			desc:         "local-routing",
			jsonFilePath: "local-routing-example.json",
			parent:       &oc.Device{},
		},
		{
			desc:         "policy",
			jsonFilePath: "policy-example.json",
			parent:       &oc.Device{},
		},
	}

	emitJSONConfig := &ygot.EmitJSONConfig{
		Format: ygot.RFC7951,
		RFC7951Config: &ygot.RFC7951JSONConfig{
			AppendModuleName: true,
		}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			j, err := ioutil.ReadFile(filepath.Join(testRoot, "testdata", tt.jsonFilePath))
			if err != nil {
				t.Errorf("%s: ioutil.ReadFile(%s): could not open file: %v", tt.desc, tt.jsonFilePath, err)
				return
			}

			err = oc.Unmarshal(j, tt.parent)
			if got, want := errToString(err), tt.wantErr; got != want {
				t.Errorf("%s: got error: %v, want error: %v ", tt.desc, got, want)
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
				d, err := diffJSON(j, []byte(jo))
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

/*
type Device struct {
	Aps              *Aps                        `path:"" rootname:"aps" module:"openconfig-transport-line-protection"`
	Bgp              *Bgp                        `path:"" rootname:"bgp" module:"openconfig-bgp"`
	Component        map[string]*Component       `path:"components/component" rootname:"component" module:"openconfig-platform"`
	Interface        map[string]*Interface       `path:"interfaces/interface" rootname:"interface" module:"openconfig-interfaces"`
	Lacp             *Lacp                       `path:"" rootname:"lacp" module:"openconfig-lacp"`
	Lldp             *Lldp                       `path:"" rootname:"lldp" module:"openconfig-lldp"`
	LocalRoutes      *LocalRoutes                `path:"" rootname:"local-routes" module:"openconfig-local-routing"`
	Mpls             *Mpls                       `path:"" rootname:"mpls" module:"openconfig-mpls"`
	NetworkInstance  map[string]*NetworkInstance `path:"network-instances/network-instance" rootname:"network-instance" module:"openconfig-network-instance"`
	OpticalAmplifier *OpticalAmplifier           `path:"" rootname:"optical-amplifier" module:"openconfig-optical-amplifier"`
	RoutingPolicy    *RoutingPolicy              `path:"" rootname:"routing-policy" module:"openconfig-routing-policy"`
	Stp              *Stp                        `path:"" rootname:"stp" module:"openconfig-spanning-tree"`
	System           *System                     `path:"" rootname:"system" module:"openconfig-system"`
	TerminalDevice   *TerminalDevice             `path:"" rootname:"terminal-device" module:"openconfig-terminal-device"`
}

type Bgp struct {
	Global    *Bgp_Global               `path:"/bgp/global" module:"openconfig-bgp"`
	Neighbor  map[string]*Bgp_Neighbor  `path:"/bgp/neighbors/neighbor" module:"openconfig-bgp"`
	PeerGroup map[string]*Bgp_PeerGroup `path:"/bgp/peer-groups/peer-group" module:"openconfig-bgp"`
}

type Bgp_Global struct {
	AfiSafi               map[E_OpenconfigBgpTypes_AFI_SAFI_TYPE]*Bgp_Global_AfiSafi `path:"afi-safis/afi-safi" module:"openconfig-bgp"`
	As                    *uint32                                                    `path:"config/as" module:"openconfig-bgp"`
	Confederation         *Bgp_Global_Confederation                                  `path:"confederation" module:"openconfig-bgp"`
	...
}

// Bgp_Neighbor represents the /openconfig-bgp/bgp/neighbors/neighbor YANG schema element.
type Bgp_Neighbor struct {
	AfiSafi                map[E_OpenconfigBgpTypes_AFI_SAFI_TYPE]*Bgp_Neighbor_AfiSafi `path:"afi-safis/afi-safi" module:"openconfig-bgp"`
	ApplyPolicy            *Bgp_Neighbor_ApplyPolicy                                    `path:"apply-policy" module:"openconfig-bgp"`
	AsPathOptions          *Bgp_Neighbor_AsPathOptions                                  `path:"as-path-options" module:"openconfig-bgp"`
	...
	NeighborAddress        *string                                                      `path:"config/neighbor-address|neighbor-address" module:"openconfig-bgp"`
	...
}

// Bgp_Neighbor_AfiSafi represents the /openconfig-bgp/bgp/neighbors/neighbor/afi-safis/afi-safi YANG schema element.
type Bgp_Neighbor_AfiSafi struct {
	Active             *bool                                    `path:"state/active" module:"openconfig-bgp"`
	AddPaths           *Bgp_Neighbor_AfiSafi_AddPaths           `path:"add-paths" module:"openconfig-bgp"`
	AfiSafiName        E_OpenconfigBgpTypes_AFI_SAFI_TYPE       `path:"config/afi-safi-name|afi-safi-name" module:"openconfig-bgp"`
	ApplyPolicy        *Bgp_Neighbor_AfiSafi_ApplyPolicy        `path:"apply-policy" module:"openconfig-bgp"`
	...
}

*/

func TestNewNode(t *testing.T) {
	tests := []struct {
		desc       string
		rootType   interface{}
		gnmiPath   *gpb.Path
		want       interface{}
		wantStatus spb.Status
	}{
		{
			desc:       "empty path",
			rootType:   &oc.Device{},
			gnmiPath:   toGNMIPath(nil),
			want:       &oc.Device{},
			wantStatus: statusOK,
		},
		{
			desc:       "bgp/global/confederation",
			rootType:   &oc.Device{},
			gnmiPath:   toGNMIPath([]string{"bgp", "global", "confederation"}),
			want:       &oc.Bgp_Global_Confederation{},
			wantStatus: statusOK,
		},
		{
			desc:     "path with keys bgp/neighbors/neighbor...",
			rootType: &oc.Device{},
			gnmiPath: &gpb.Path{
				Elem: []*gpb.PathElem{
					{
						Name: "bgp",
					},
					{
						Name: "neighbors",
					},
					{
						Name: "neighbor",
						Key: map[string]string{
							"some-key1": "some-key-value1",
						},
					},
					{
						Name: "afi-safis",
					},
					{
						Name: "afi-safi",
						Key: map[string]string{
							"some-key2": "some-key-value2",
						},
					},
					{
						Name: "config",
					},
					{
						Name: "afi-safi-name",
					},
				},
			},
			want:       oc.E_OpenconfigBgpTypes_AFI_SAFI_TYPE(0),
			wantStatus: statusOK,
		},
		{
			desc:     "bad path",
			rootType: &oc.Device{},
			gnmiPath: toGNMIPath([]string{"bad", "path"}),
			wantStatus: spb.Status{
				Code:    int32(scpb.Code_NOT_FOUND),
				Message: `could not find path in tree beyond type *exampleoc.Device, remaining path elem:<name:"bad" > elem:<name:"path" > `,
			},
		},
	}

	for _, tt := range tests {
		n, status := ygotutils.NewNode(reflect.TypeOf(tt.rootType), tt.gnmiPath)
		if got, want := status, tt.wantStatus; got.GetMessage() != want.GetMessage() {
			t.Errorf("%s: got status: %v, want status: %v ", tt.desc, got, want)
		}
		testErrLog(t, tt.desc, fmt.Errorf(status.GetMessage()))
		if isOK(status) {
			if got, want := n, tt.want; !reflect.DeepEqual(got, want) {
				t.Errorf("%s: got: %v, want: %v ", tt.desc, util.ValueStr(got), util.ValueStr(want))
			}
		}
	}
}

func TestGetNode(t *testing.T) {
	testDevice := &oc.Device{
		Bgp: &oc.Bgp{
			Global: &oc.Bgp_Global{
				Confederation: &oc.Bgp_Global_Confederation{},
			},
			Neighbor: map[string]*oc.Bgp_Neighbor{
				"address1": {
					ApplyPolicy:     &oc.Bgp_Neighbor_ApplyPolicy{},
					NeighborAddress: ygot.String("address1"),
				},
			},
		},
	}

	tests := []struct {
		desc       string
		gnmiPath   *gpb.Path
		want       interface{}
		wantStatus spb.Status
	}{
		{
			desc:       "empty path",
			gnmiPath:   toGNMIPath(nil),
			want:       testDevice,
			wantStatus: statusOK,
		},
		{
			desc:       "bgp/global/confederation",
			gnmiPath:   toGNMIPath([]string{"bgp", "global", "confederation"}),
			want:       testDevice.Bgp.Global.Confederation,
			wantStatus: statusOK,
		},
		{
			desc: "path with keys bgp/neighbors/neighbor...",
			gnmiPath: &gpb.Path{
				Elem: []*gpb.PathElem{
					{
						Name: "bgp",
					},
					{
						Name: "neighbors",
					},
					{
						Name: "neighbor",
						Key: map[string]string{
							"neighbor-address": "address1",
						},
					},
					{
						Name: "apply-policy",
					},
				},
			},
			want:       testDevice.Bgp.Neighbor["address1"].ApplyPolicy,
			wantStatus: statusOK,
		},
		{
			desc: "bad key field",
			gnmiPath: &gpb.Path{
				Elem: []*gpb.PathElem{
					{
						Name: "bgp",
					},
					{
						Name: "neighbors",
					},
					{
						Name: "neighbor",
						Key: map[string]string{
							"bad-key-field": "address1",
						},
					},
					{
						Name: "apply-policy",
					},
				},
			},
			wantStatus: spb.Status{
				Code:    int32(scpb.Code_INVALID_ARGUMENT),
				Message: `gnmi path elem:<name:"neighbor" key:<key:"bad-key-field" value:"address1" > > elem:<name:"apply-policy" >  does not contain a map entry for the schema key field name neighbor-address, parent type map[string]*exampleoc.Bgp_Neighbor`,
			},
		},
		{
			desc: "bad key value",
			gnmiPath: &gpb.Path{
				Elem: []*gpb.PathElem{
					{
						Name: "bgp",
					},
					{
						Name: "neighbors",
					},
					{
						Name: "neighbor",
						Key: map[string]string{
							"neighbor-address": "bad key value",
						},
					},
					{
						Name: "apply-policy",
					},
				},
			},
			wantStatus: spb.Status{
				Code:    int32(scpb.Code_NOT_FOUND),
				Message: `could not find path in tree beyond schema node neighbor, (type map[string]*exampleoc.Bgp_Neighbor), remaining path elem:<name:"neighbor" key:<key:"neighbor-address" value:"bad key value" > > elem:<name:"apply-policy" > `,
			},
		},
		{
			desc:     "bad path",
			gnmiPath: toGNMIPath([]string{"bad", "path"}),
			wantStatus: spb.Status{
				Code:    int32(scpb.Code_NOT_FOUND),
				Message: `could not find path in tree beyond schema node device, (type *exampleoc.Device), remaining path elem:<name:"bad" > elem:<name:"path" > `,
			},
		},
	}

	for _, tt := range tests {
		n, status := ygotutils.GetNode(oc.SchemaTree["Device"], testDevice, tt.gnmiPath)
		if got, want := status, tt.wantStatus; !reflect.DeepEqual(got, want) {
			t.Errorf("%s: got status: %v, want status: %v ", tt.desc, got, want)
		}
		testErrLog(t, tt.desc, fmt.Errorf(status.GetMessage()))
		if isOK(status) {
			if got, want := n, tt.want; !reflect.DeepEqual(got, want) {
				t.Errorf("%s: got: %v, want: %v ", tt.desc, util.ValueStr(got), util.ValueStr(want))
			}
		}
	}
}

/* TestLeafrefCurrent validates that the current() function works when
   leafrefs are validated in a real schema.
   It uses the following struct as the input:
   type Mpls_Global_Interface struct {
        InterfaceId  *string                             `path:"config/interface-id|interface-id" module:"openconfig-mpls"`
        InterfaceRef *Mpls_Global_Interface_InterfaceRef `path:"interface-ref" module:"openconfig-mpls"`
        MplsEnabled  *bool                               `path:"config/mpls-enabled" module:"openconfig-mpls"`
   }
   where the InterfaceRef container references an interface/subinterface
   in the /interfaces/interface list.
*/
func TestLeafrefCurrent(t *testing.T) {
	dev := &oc.Device{}
	i, err := dev.NewInterface("eth0")
	if err != nil {
		t.Fatalf("TestLeafrefCurrent: could not create new interface, got: %v, want error: nil", err)
	}
	if _, err := i.NewSubinterface(0); err != nil {
		t.Fatalf("TestLeafrefCurrent: could not create subinterface, got: %v, want error: nil", err)
	}

	ygot.BuildEmptyTree(dev)
	mi, err := dev.Mpls.Global.NewInterface("eth0.0")
	if err != nil {
		t.Fatalf("TestLeafrefCurrent: could not add new MPLS interface, got: %v, want error: nil", err)
	}
	mi.InterfaceRef = &oc.Mpls_Global_Interface_InterfaceRef{
		Interface:    ygot.String("eth0"),
		Subinterface: ygot.Uint32(0),
	}

	if err := dev.Validate(); err != nil {
		t.Fatalf("TestLeafrefCurrent: could not validate populated interfaces, got: %v, want: nil", err)
	}

	dev.Mpls.Global.Interface["eth0.0"].InterfaceRef.Subinterface = ygot.Uint32(1)
	if err := dev.Validate(); err == nil {
		t.Fatal("TestLeafrefCurrent: did not get expected error for non-existent subinterface, got: nil, want: error")
	}

	if err := dev.Validate(&ytypes.LeafrefOptions{IgnoreMissingData: true}); err != nil {
		t.Fatalf("TestLeafrefCurrent: did not get nil error when disabling leafref data validation, got: %v, want: nil", err)
	}
}

func toGNMIPath(path []string) *gpb.Path {
	out := &gpb.Path{}
	for _, p := range path {
		out.Elem = append(out.GetElem(), &gpb.PathElem{Name: p})
	}
	return out
}

// statusOK indicates an OK Status.
var statusOK = spb.Status{Code: int32(scpb.Code_OK)}

func isOK(status spb.Status) bool {
	return status.GetCode() == int32(scpb.Code_OK)
}

// generateUnifiedDiff takes two strings and generates a diff that can be
// shown to the user in a test error message.
func generateUnifiedDiff(want, got string) (string, error) {
	diffl := difflib.UnifiedDiff{
		A:        difflib.SplitLines(got),
		B:        difflib.SplitLines(want),
		FromFile: "got",
		ToFile:   "want",
		Context:  3,
		Eol:      "\n",
	}
	return difflib.GetUnifiedDiffString(diffl)
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

	return generateUnifiedDiff(strings.Join(asv, "\n"), strings.Join(bsv, "\n"))
}
