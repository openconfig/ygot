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
	"reflect"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/google/go-cmp/cmp"
	"github.com/kylelemons/godebug/pretty"
	"github.com/openconfig/gnmi/errdiff"
	"github.com/openconfig/gnmi/value"
	"github.com/openconfig/ygot/exampleoc"
	"github.com/openconfig/ygot/exampleoc/opstateoc"
	"github.com/openconfig/ygot/exampleoc/wrapperunionoc"
	"github.com/openconfig/ygot/testutil"
	"github.com/openconfig/ygot/uexampleoc"
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

	p, err := ni.NewProtocol(exampleoc.PolicyTypes_INSTALL_PROTOCOL_TYPE_BGP, "15169")
	if err != nil {
		t.Fatalf("got unexpected error: %v", err)
	}

	ygot.BuildEmptyTree(p)

	n, err := p.Bgp.NewNeighbor("192.0.2.1")
	if err != nil {
		t.Fatalf("got unexpected error: %v", err)
	}
	n.PeerAs = ygot.Uint32(42)
	n.SendCommunity = exampleoc.BgpTypes_CommunityType_STANDARD

	p.Bgp.Global.As = ygot.Uint32(42)

	ygot.PruneEmptyBranches(got)

	want := &exampleoc.Device{
		NetworkInstance: map[string]*exampleoc.NetworkInstance{
			"DEFAULT": {
				Name: ygot.String("DEFAULT"),
				Protocol: map[exampleoc.NetworkInstance_Protocol_Key]*exampleoc.NetworkInstance_Protocol{
					{
						Identifier: exampleoc.PolicyTypes_INSTALL_PROTOCOL_TYPE_BGP,
						Name:       "15169",
					}: {
						Identifier: exampleoc.PolicyTypes_INSTALL_PROTOCOL_TYPE_BGP,
						Name:       ygot.String("15169"),
						Bgp: &exampleoc.NetworkInstance_Protocol_Bgp{
							Global: &exampleoc.NetworkInstance_Protocol_Bgp_Global{
								As: ygot.Uint32(42),
							},
							Neighbor: map[string]*exampleoc.NetworkInstance_Protocol_Bgp_Neighbor{
								"192.0.2.1": {
									NeighborAddress: ygot.String("192.0.2.1"),
									PeerAs:          ygot.Uint32(42),
									SendCommunity:   exampleoc.BgpTypes_CommunityType_STANDARD,
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

func TestPopulateDefaults(t *testing.T) {
	// 1. recursively populate
	// 2. populates lists
	// 3. doesn't overwrite set fields.
	setAndPopulate := func() *exampleoc.Device {
		d := &exampleoc.Device{}
		i := d.GetOrCreateInterface("eth0")
		vrrpGroup := i.GetOrCreateSubinterface(1).GetOrCreateIpv4().GetOrCreateAddress("1.1.1.1").GetOrCreateVrrpGroup(1)
		vrrpGroup.AdvertisementInterval = ygot.Uint16(84)
		i.PopulateDefaults()
		return d
	}

	populateAndSet := func() *exampleoc.Device {
		d := &exampleoc.Device{}
		i := d.GetOrCreateInterface("eth0")
		vrrpGroup := i.GetOrCreateSubinterface(1).GetOrCreateIpv4().GetOrCreateAddress("1.1.1.1").GetOrCreateVrrpGroup(1)
		i.PopulateDefaults()
		setVal := ygot.Uint16(84)
		if reflect.DeepEqual(vrrpGroup.AdvertisementInterval, setVal) {
			t.Fatalf("expected default value to be populated that's different than the test val %v", *setVal)
		}
		vrrpGroup.AdvertisementInterval = setVal
		return d
	}

	got, want := setAndPopulate(), populateAndSet()
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("(-got, +want):\n%s", diff)
	}
	if got, want := len(got.Interface), 1; got != want {
		t.Errorf("got %v interfaces populated in struct, expected %v.", got, want)
	}
}

func TestPruneConfigFalse(t *testing.T) {
	configAndState := func() *exampleoc.Device {
		d := &exampleoc.Device{}
		b := d.GetOrCreateNetworkInstance("DEFAULT").GetOrCreateProtocol(exampleoc.PolicyTypes_INSTALL_PROTOCOL_TYPE_BGP, "15169").GetOrCreateBgp()
		n := b.GetOrCreateNeighbor("192.0.2.1")
		n.PeerAs = ygot.Uint32(29636)
		n.PeerType = exampleoc.BgpTypes_PeerType_EXTERNAL
		n.SessionState = exampleoc.Bgp_Neighbor_SessionState_ESTABLISHED

		i := d.GetOrCreateInterface("eth0")
		i.Description = ygot.String("foo")
		i.Mtu = ygot.Uint16(1500)
		i.OperStatus = exampleoc.Interface_OperStatus_UP
		i.Logical = ygot.Bool(false)
		return d
	}

	configOnly := func() *exampleoc.Device {
		d := &exampleoc.Device{}
		b := d.GetOrCreateNetworkInstance("DEFAULT").GetOrCreateProtocol(exampleoc.PolicyTypes_INSTALL_PROTOCOL_TYPE_BGP, "15169").GetOrCreateBgp()
		n := b.GetOrCreateNeighbor("192.0.2.1")
		n.PeerAs = ygot.Uint32(29636)
		n.PeerType = exampleoc.BgpTypes_PeerType_EXTERNAL

		i := d.GetOrCreateInterface("eth0")
		i.Description = ygot.String("foo")
		i.Mtu = ygot.Uint16(1500)
		return d
	}

	got, want := configAndState(), configOnly()
	if err := ygot.PruneConfigFalse(exampleoc.SchemaTree["Device"], got); err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("(-got, +want):\n%s", diff)
	}
}

func TestPruneConfigFalseOpState(t *testing.T) {
	configAndState := func() *opstateoc.Device {
		d := &opstateoc.Device{}
		b := d.GetOrCreateNetworkInstance("DEFAULT").GetOrCreateProtocol(opstateoc.OpenconfigPolicyTypes_INSTALL_PROTOCOL_TYPE_BGP, "15169").GetOrCreateBgp()
		n := b.GetOrCreateNeighbor("192.0.2.1")
		n.PeerAs = ygot.Uint32(29636)
		n.PeerType = opstateoc.OpenconfigBgpTypes_PeerType_EXTERNAL
		n.SessionState = opstateoc.OpenconfigBgp_Neighbor_SessionState_ESTABLISHED

		i := d.GetOrCreateInterface("eth0")
		i.Description = ygot.String("foo")
		i.Mtu = ygot.Uint16(1500)
		i.OperStatus = opstateoc.Interface_OperStatus_UP
		i.Logical = ygot.Bool(false)
		return d
	}

	configOnly := func() *opstateoc.Device {
		d := &opstateoc.Device{}
		b := d.GetOrCreateNetworkInstance("DEFAULT").GetOrCreateProtocol(opstateoc.OpenconfigPolicyTypes_INSTALL_PROTOCOL_TYPE_BGP, "15169").GetOrCreateBgp()
		n := b.GetOrCreateNeighbor("192.0.2.1")
		n.PeerAs = ygot.Uint32(29636)
		n.PeerType = opstateoc.OpenconfigBgpTypes_PeerType_EXTERNAL

		i := d.GetOrCreateInterface("eth0")
		i.Description = ygot.String("foo")
		i.Mtu = ygot.Uint16(1500)
		return d
	}

	got, want := configAndState(), configOnly()
	if err := ygot.PruneConfigFalse(opstateoc.SchemaTree["Device"], got); err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("(-got, +want):\n%s", diff)
	}
}

func TestPruneUncompressed(t *testing.T) {
	configAndState := func() *uexampleoc.Device {
		d := &uexampleoc.Device{}

		i := d.GetOrCreateInterfaces().GetOrCreateInterface("eth0")
		c := i.GetOrCreateConfig()
		c.Description = ygot.String("foo")
		c.Mtu = ygot.Uint16(1500)
		s := i.GetOrCreateState()
		s.Mtu = ygot.Uint16(1500)
		s.OperStatus = uexampleoc.OpenconfigInterfaces_Interfaces_Interface_State_OperStatus_UP
		s.Logical = ygot.Bool(false)
		return d
	}

	configOnly := func() *uexampleoc.Device {
		d := &uexampleoc.Device{}

		i := d.GetOrCreateInterfaces().GetOrCreateInterface("eth0")
		c := i.GetOrCreateConfig()
		c.Description = ygot.String("foo")
		c.Mtu = ygot.Uint16(1500)
		return d
	}

	got, want := configAndState(), configOnly()
	if err := ygot.PruneConfigFalse(uexampleoc.SchemaTree["Device"], got); err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("(-got, +want):\n%s", diff)
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
			b := d.GetOrCreateNetworkInstance("DEFAULT").GetOrCreateProtocol(exampleoc.PolicyTypes_INSTALL_PROTOCOL_TYPE_BGP, "15169").GetOrCreateBgp()
			n := b.GetOrCreateNeighbor("192.0.2.1")
			n.PeerAs = ygot.Uint32(29636)
			n.PeerType = exampleoc.BgpTypes_PeerType_EXTERNAL
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
		desc:   "diff BGP neighbour using prefer_operational_state",
		inOrig: &opstateoc.NetworkInstance_Protocol_Bgp{},
		inMod: func() *opstateoc.NetworkInstance_Protocol_Bgp {
			d := &opstateoc.Device{}
			b := d.GetOrCreateNetworkInstance("DEFAULT").GetOrCreateProtocol(opstateoc.OpenconfigPolicyTypes_INSTALL_PROTOCOL_TYPE_BGP, "15169").GetOrCreateBgp()
			n := b.GetOrCreateNeighbor("192.0.2.1")
			n.PeerAs = ygot.Uint32(29636)
			n.PeerType = opstateoc.OpenconfigBgpTypes_PeerType_EXTERNAL
			return b
		}(),
		want: &gnmipb.Notification{
			Update: []*gnmipb.Update{{
				Path: mustPath("neighbors/neighbor[neighbor-address=192.0.2.1]/neighbor-address"),
				Val:  mustTypedValue("192.0.2.1"),
			}, {
				Path: mustPath("neighbors/neighbor[neighbor-address=192.0.2.1]/state/neighbor-address"),
				Val:  mustTypedValue("192.0.2.1"),
			}, {
				Path: mustPath("neighbors/neighbor[neighbor-address=192.0.2.1]/state/peer-as"),
				Val:  mustTypedValue(uint32(29636)),
			}, {
				Path: mustPath("neighbors/neighbor[neighbor-address=192.0.2.1]/state/peer-type"),
				Val:  mustTypedValue("EXTERNAL"),
			}},
		},
	}, {
		desc:   "diff STP",
		inOrig: &exampleoc.Device{},
		inMod: func() *exampleoc.Device {
			d := &exampleoc.Device{}
			e := d.GetOrCreateStp().GetOrCreateGlobal()
			e.EnabledProtocol = []exampleoc.E_SpanningTreeTypes_STP_PROTOCOL{
				exampleoc.SpanningTreeTypes_STP_PROTOCOL_MSTP,
				exampleoc.SpanningTreeTypes_STP_PROTOCOL_RSTP,
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
		in       ygot.ValidatedGoStruct
		wantFile string
	}{{
		name: "unset enumeration",
		in: func() *exampleoc.Device {
			d := &exampleoc.Device{}
			acl := d.GetOrCreateAcl()
			set := acl.GetOrCreateAclSet("set", exampleoc.Acl_ACL_TYPE_ACL_IPV6)
			entry := set.GetOrCreateAclEntry(100)
			entry.GetOrCreateIpv6().Protocol = exampleoc.PacketMatchTypes_IP_PROTOCOL_UNSET
			return d
		}(),
		wantFile: "testdata/unsetenum.json",
	}, {
		name: "unset enumeration using wrapper union generated code",
		in: func() *wrapperunionoc.Device {
			d := &wrapperunionoc.Device{}
			acl := d.GetOrCreateAcl()
			set := acl.GetOrCreateAclSet("set", wrapperunionoc.Acl_ACL_TYPE_ACL_IPV6)
			entry := set.GetOrCreateAclEntry(100)
			entry.GetOrCreateIpv6().Protocol = &wrapperunionoc.Acl_AclSet_AclEntry_Ipv6_Protocol_Union_E_PacketMatchTypes_IP_PROTOCOL{
				wrapperunionoc.PacketMatchTypes_IP_PROTOCOL_UNSET,
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

func TestNotificationOutput(t *testing.T) {
	tests := []struct {
		name       string
		in         ygot.ValidatedGoStruct
		wantTextpb string
	}{{
		name: "int64 from root",
		in: func() *exampleoc.Device {
			d := &exampleoc.Device{}
			d.GetOrCreateInterface("eth0")
			is := d.GetOrCreateLldp().GetOrCreateInterface("eth0").GetOrCreateNeighbor("neighbor")
			is.LastUpdate = ygot.Int64(42)
			return d
		}(),
		wantTextpb: "testdata/notification_int64.txtpb",
	}, {
		name: "int64 uncompressed",
		in: func() *uexampleoc.OpenconfigLldp_Lldp_Interfaces_Interface_Neighbors_Neighbor_State {
			s := &uexampleoc.OpenconfigLldp_Lldp_Interfaces_Interface_Neighbors_Neighbor_State{}
			s.LastUpdate = ygot.Int64(42)
			return s
		}(),
		wantTextpb: "testdata/uncompressed_notification_int64.txtpb",
	}, {
		name: "int64 union",
		in: func() *exampleoc.Device {
			d := &exampleoc.Device{}
			t := d.GetOrCreateComponent("p1").GetOrCreateProperty("temperature")
			t.Value = exampleoc.UnionInt64(42)
			return d
		}(),
		wantTextpb: "testdata/notification_union_int64.txtpb",
	}, {
		name: "int64 union using wrapper union generated code",
		in: func() *wrapperunionoc.Device {
			d := &wrapperunionoc.Device{}
			t := d.GetOrCreateComponent("p1").GetOrCreateProperty("temperature")
			v, err := t.To_Component_Property_Value_Union(int64(42))
			if err != nil {
				panic(err)
			}
			t.Value = v
			return d
		}(),
		wantTextpb: "testdata/notification_union_int64.txtpb",
	}, {
		name: "int64 union using operational state",
		in: func() *opstateoc.Device {
			d := &opstateoc.Device{}
			t := d.GetOrCreateComponent("p1").GetOrCreateProperty("temperature")
			t.Value = opstateoc.UnionInt64(42)
			return d
		}(),
		wantTextpb: "testdata/notification_union_int64_opstate.txtpb",
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantTxtpb, err := ioutil.ReadFile(tt.wantTextpb)
			if err != nil {
				t.Fatalf("could not read textproto, %v", err)
			}

			wantNoti := &gnmipb.Notification{}
			if err := proto.UnmarshalText(string(wantTxtpb), wantNoti); err != nil {
				t.Fatalf("cannot unmarshal wanted textproto, %v", err)
			}

			gotSet, err := ygot.TogNMINotifications(tt.in, 0, ygot.GNMINotificationsConfig{
				UsePathElem: true,
			})
			if err != nil {
				t.Fatalf("cannot marshal input to gNMI Notifications, %v", err)
			}

			if !testutil.NotificationSetEqual(gotSet, []*gnmipb.Notification{wantNoti}) {
				diff, err := testutil.GenerateUnifiedDiff(proto.MarshalTextString(wantNoti), proto.MarshalTextString(gotSet[0]))
				if err != nil {
					t.Errorf("cannot diff generated protobufs, %v", err)
				}
				t.Fatalf("did not get unexpected Notifications, diff(-want,+got):\n%s", diff)
			}
		})
	}
}

func getBenchmarkDevice() *exampleoc.Device {
	intfs := []string{"eth0", "eth1", "eth2", "eth3"}
	d := &exampleoc.Device{}
	for _, intf := range intfs {
		d.GetOrCreateInterface(intf)
		is := d.GetOrCreateLldp().GetOrCreateInterface(intf).GetOrCreateNeighbor("neighbor")
		is.LastUpdate = ygot.Int64(42)
	}
	return d
}

func BenchmarkNotificationOutput(b *testing.B) {
	d := getBenchmarkDevice()
	b.ResetTimer()
	for i := 0; i != b.N; i++ {
		if _, err := ygot.TogNMINotifications(d, 0, ygot.GNMINotificationsConfig{
			UsePathElem: true,
		}); err != nil {
			b.Fatalf("cannot marshal input to gNMI Notifications, %v", err)
		}
	}
}

func BenchmarkNotificationOutputElement(b *testing.B) {
	d := getBenchmarkDevice()
	b.ResetTimer()
	for i := 0; i != b.N; i++ {
		if _, err := ygot.TogNMINotifications(d, 0, ygot.GNMINotificationsConfig{
			UsePathElem: false,
		}); err != nil {
			b.Fatalf("cannot marshal input to gNMI Notifications, %v", err)
		}
	}
}
