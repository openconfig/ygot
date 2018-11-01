// Copyright 2018 Google Inc.
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

// Package schematest is used for testing with the default OpenConfig generated
// structs.
package schematest

import (
	"io/ioutil"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/kylelemons/godebug/pretty"
	"github.com/openconfig/gnmi/errdiff"
	"github.com/openconfig/gnmi/value"
	"github.com/openconfig/ygot/exampleoc"
	"github.com/openconfig/ygot/testutil"
	"github.com/openconfig/ygot/ygot"

	gnmipb "github.com/openconfig/gnmi/proto/gnmi"
)

func TestBuildEmptyEthernet(t *testing.T) {
	got := &exampleoc.Interface_Ethernet{}
	ygot.BuildEmptyTree(got)

	wantEmpty := &exampleoc.Interface_Ethernet{
		SwitchedVlan: &exampleoc.Interface_Ethernet_SwitchedVlan{},
		Counters:     &exampleoc.Interface_Ethernet_Counters{},
	}

	if diff := pretty.Compare(got, wantEmpty); diff != "" {
		t.Fatalf("did not get expected output after BuildEmptyTree, diff(-got,+want):\n%s", diff)
	}

	got.AutoNegotiate = ygot.Bool(true)
	ygot.PruneEmptyBranches(got)

	wantPruned := &exampleoc.Interface_Ethernet{
		AutoNegotiate: ygot.Bool(true),
	}

	if diff := pretty.Compare(got, wantPruned); diff != "" {
		t.Fatalf("did not get expected output after PruneEmptyBranches, diff(-got,+want):\n%s", diff)
	}
}

func TestBuildEmptyDevice(t *testing.T) {
	got := &exampleoc.Device{}
	ygot.BuildEmptyTree(got)

	ni, err := got.NewNetworkInstance("DEFAULT")
	if err != nil {
		t.Fatalf("got unexpected error: %v", err)
	}
	ygot.BuildEmptyTree(ni)

	p, err := ni.NewProtocol(exampleoc.OpenconfigPolicyTypes_INSTALL_PROTOCOL_TYPE_BGP, "15169")
	if err != nil {
		t.Fatalf("got unexpected error: %v", err)
	}

	ygot.BuildEmptyTree(p)

	n, err := p.Bgp.NewNeighbor("192.0.2.1")
	if err != nil {
		t.Fatalf("got unexpected error: %v", err)
	}
	n.PeerAs = ygot.Uint32(42)
	n.SendCommunity = exampleoc.OpenconfigBgp_CommunityType_STANDARD

	p.Bgp.Global.As = ygot.Uint32(42)

	ygot.PruneEmptyBranches(got)

	want := &exampleoc.Device{
		NetworkInstance: map[string]*exampleoc.NetworkInstance{
			"DEFAULT": {
				Name: ygot.String("DEFAULT"),
				Protocol: map[exampleoc.NetworkInstance_Protocol_Key]*exampleoc.NetworkInstance_Protocol{
					{exampleoc.OpenconfigPolicyTypes_INSTALL_PROTOCOL_TYPE_BGP, "15169"}: {
						Identifier: exampleoc.OpenconfigPolicyTypes_INSTALL_PROTOCOL_TYPE_BGP,
						Name:       ygot.String("15169"),
						Bgp: &exampleoc.NetworkInstance_Protocol_Bgp{
							Global: &exampleoc.NetworkInstance_Protocol_Bgp_Global{
								As: ygot.Uint32(42),
							},
							Neighbor: map[string]*exampleoc.NetworkInstance_Protocol_Bgp_Neighbor{
								"192.0.2.1": {
									NeighborAddress: ygot.String("192.0.2.1"),
									PeerAs:          ygot.Uint32(42),
									SendCommunity:   exampleoc.OpenconfigBgp_CommunityType_STANDARD,
								},
							},
						},
					},
				},
			},
		},
	}

	if diff := pretty.Compare(got, want); diff != "" {
		t.Errorf("did not get expected device struct, diff(-got,+want):\n%s", diff)
	}
}

// mustPath returns a string as a gNMI path, causing a panic if the string
// is invalid.
func mustPath(s string) *gnmipb.Path {
	p, err := ygot.StringToStructuredPath(s)
	if err != nil {
		panic(err)
	}
	return p
}

// mustTypedValue returns a value (interface) supplied as a gNMI path, causing
// a panic if the interface{} is not a valid typed value.
func mustTypedValue(i interface{}) *gnmipb.TypedValue {
	v, err := value.FromScalar(i)
	if err != nil {
		panic(err)
	}
	return v
}

func TestDiff(t *testing.T) {
	tests := []struct {
		desc             string
		inOrig           ygot.GoStruct
		inMod            ygot.GoStruct
		want             *gnmipb.Notification
		wantErrSubstring string
	}{{
		desc:   "diff BGP neighbour",
		inOrig: &exampleoc.NetworkInstance_Protocol_Bgp{},
		inMod: func() *exampleoc.NetworkInstance_Protocol_Bgp {
			d := &exampleoc.Device{}
			b := d.GetOrCreateNetworkInstance("DEFAULT").GetOrCreateProtocol(exampleoc.OpenconfigPolicyTypes_INSTALL_PROTOCOL_TYPE_BGP, "15169").GetOrCreateBgp()
			n := b.GetOrCreateNeighbor("192.0.2.1")
			n.PeerAs = ygot.Uint32(29636)
			n.PeerType = exampleoc.OpenconfigBgp_PeerType_EXTERNAL
			return b
		}(),
		want: &gnmipb.Notification{
			Update: []*gnmipb.Update{{
				Path: mustPath("neighbors/neighbor[neighbor-address=192.0.2.1]/neighbor-address"),
				Val:  mustTypedValue("192.0.2.1"),
			}, {
				Path: mustPath("neighbors/neighbor[neighbor-address=192.0.2.1]/config/neighbor-address"),
				Val:  mustTypedValue("192.0.2.1"),
			}, {
				Path: mustPath("neighbors/neighbor[neighbor-address=192.0.2.1]/config/peer-as"),
				Val:  mustTypedValue(uint32(29636)),
			}, {
				Path: mustPath("neighbors/neighbor[neighbor-address=192.0.2.1]/config/peer-type"),
				Val:  mustTypedValue("EXTERNAL"),
			}},
		},
	}, {
		desc:   "diff STP",
		inOrig: &exampleoc.Device{},
		inMod: func() *exampleoc.Device {
			d := &exampleoc.Device{}
			e := d.GetOrCreateStp().GetOrCreateGlobal()
			e.EnabledProtocol = []exampleoc.E_OpenconfigSpanningTreeTypes_STP_PROTOCOL{
				exampleoc.OpenconfigSpanningTreeTypes_STP_PROTOCOL_MSTP,
				exampleoc.OpenconfigSpanningTreeTypes_STP_PROTOCOL_RSTP,
			}
			return d
		}(),
		want: &gnmipb.Notification{
			Update: []*gnmipb.Update{{
				Path: mustPath("/stp/global/config/enabled-protocol"),
				Val:  mustTypedValue([]string{"MSTP", "RSTP"}),
			}},
		},
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got, err := ygot.Diff(tt.inOrig, tt.inMod)
			if diff := errdiff.Substring(err, tt.wantErrSubstring); diff != "" {
				t.Fatalf("ygot.Diff(%#v, %#v): did not get expected error, %s", tt.inOrig, tt.inMod, diff)
			}

			if !testutil.NotificationSetEqual([]*gnmipb.Notification{got}, []*gnmipb.Notification{tt.want}) {
				diff, err := testutil.GenerateUnifiedDiff(proto.MarshalTextString(got), proto.MarshalTextString(tt.want))
				if err != nil {
					diff = "unable to produce diff"
				}

				t.Fatalf("ygot.Diff(%#v, %#v); did not get expected diff output, diff(-got,+want):\n%s", tt.inOrig, tt.inMod, diff)
			}
		})
	}
}

func TestJSONOutput(t *testing.T) {
	tests := []struct {
		name     string
		in       *exampleoc.Device
		wantFile string
	}{{
		name: "unset enumeration",
		in: func() *exampleoc.Device {
			d := &exampleoc.Device{}
			acl := d.GetOrCreateAcl()
			set := acl.GetOrCreateAclSet("set", exampleoc.OpenconfigAcl_ACL_TYPE_ACL_IPV6)
			entry := set.GetOrCreateAclEntry(100)
			entry.GetOrCreateIpv6().Protocol = &exampleoc.Acl_AclSet_AclEntry_Ipv6_Protocol_Union_E_OpenconfigPacketMatchTypes_IP_PROTOCOL{
				exampleoc.OpenconfigPacketMatchTypes_IP_PROTOCOL_UNSET,
			}
			return d
		}(),
		wantFile: "testdata/unsetenum.json",
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want, err := ioutil.ReadFile(tt.wantFile)
			if err != nil {
				t.Fatalf("cannot read wantfile, %v", err)
			}

			got, err := ygot.EmitJSON(tt.in, &ygot.EmitJSONConfig{Format: ygot.RFC7951})
			if err != nil {
				t.Fatalf("got unexpected error, %v", err)
			}

			if diff := pretty.Compare(string(got), string(want)); diff != "" {
				if diffl, err := testutil.GenerateUnifiedDiff(string(got), string(want)); err == nil {
					diff = diffl
				}
				t.Fatalf("did not get expected output, diff(-got,+want):\n%s", diff)
			}
		})
	}
}
