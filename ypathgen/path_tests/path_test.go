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

package pathtest

import (
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/google/go-cmp/cmp"
	oc "github.com/openconfig/ygot/exampleoc"
	ocp "github.com/openconfig/ygot/exampleocpath"
	"github.com/openconfig/ygot/ygot"
)

// The device ID used throughout this test file.
const deviceId = "dev"

// verifyPath checks the given path against expected.
func verifyPath(t *testing.T, p ygot.PathStruct, wantPathStr string) {
	t.Helper()
	gotPath, errs := ocp.Resolve(p)
	if errs != nil {
		t.Fatal(errs)
	}

	wantPath, err := ygot.StringToStructuredPath(wantPathStr)
	if err != nil {
		t.Fatal(err)
	}
	wantPath.Target = deviceId

	if diff := cmp.Diff(wantPath, gotPath, cmp.Comparer(proto.Equal)); diff != "" {
		t.Fatalf("verifyPath returned diff (-want +got):\n%s", diff)
	}
}

func TestPrefixing(t *testing.T) {
	root := ocp.ForDevice(deviceId)
	i := root.Interface("eth1")
	verifyPath(t, i, "/interfaces/interface[name=eth1]")
	s := i.Subinterface(1)
	verifyPath(t, s, "/interfaces/interface[name=eth1]/subinterfaces/subinterface[index=1]")
	ip := s.Ipv6()
	verifyPath(t, ip, "/interfaces/interface[name=eth1]/subinterfaces/subinterface[index=1]/ipv6")
	a := ip.Address("1:2:3:4::")
	verifyPath(t, a, "/interfaces/interface[name=eth1]/subinterfaces/subinterface[index=1]/ipv6/addresses/address[ip=1:2:3:4::]")
	v := a.VrrpGroup(2)
	verifyPath(t, v, "/interfaces/interface[name=eth1]/subinterfaces/subinterface[index=1]/ipv6/addresses/address[ip=1:2:3:4::]/vrrp/vrrp-group[virtual-router-id=2]")
	p := v.PreemptDelay()
	verifyPath(t, p, "/interfaces/interface[name=eth1]/subinterfaces/subinterface[index=1]/ipv6/addresses/address[ip=1:2:3:4::]/vrrp/vrrp-group[virtual-router-id=2]/state/preempt-delay")
}

// This test shows ways to reduce typing when creating similar paths.
func TestManualShortcuts(t *testing.T) {
	root := ocp.ForDevice(deviceId)
	preemptDelay := func(intf string, subintf uint32, ip string) ygot.PathStruct {
		return root.Interface(intf).Subinterface(subintf).Ipv6().Address(ip).VrrpGroup(1).PreemptDelay()
	}

	// defining short helpers
	verifyPath(t, preemptDelay("eth1", 1, "1::"), "/interfaces/interface[name=eth1]/subinterfaces/subinterface[index=1]/ipv6/addresses/address[ip=1::]/vrrp/vrrp-group[virtual-router-id=1]/state/preempt-delay")
	verifyPath(t, preemptDelay("eth1", 2, "2:2:2:2::"), "/interfaces/interface[name=eth1]/subinterfaces/subinterface[index=2]/ipv6/addresses/address[ip=2:2:2:2::]/vrrp/vrrp-group[virtual-router-id=1]/state/preempt-delay")
	verifyPath(t, preemptDelay("eth2", 2, "::"), "/interfaces/interface[name=eth2]/subinterfaces/subinterface[index=2]/ipv6/addresses/address[ip=::]/vrrp/vrrp-group[virtual-router-id=1]/state/preempt-delay")

	// re-using prefixes
	intf1 := root.Interface("eth1")
	verifyPath(t, intf1.Subinterface(3), "/interfaces/interface[name=eth1]/subinterfaces/subinterface[index=3]")
	verifyPath(t, intf1.Subinterface(4), "/interfaces/interface[name=eth1]/subinterfaces/subinterface[index=4]")
}

func TestPathCreation(t *testing.T) {
	tests := []struct {
		name     string
		makePath func(*ocp.Device) ygot.PathStruct
		want     string
	}{{
		name: "simple",
		makePath: func(root *ocp.Device) ygot.PathStruct {
			return root.Stp()
		},
		want: "/stp",
	}, {
		name: "simple prefixing",
		makePath: func(root *ocp.Device) ygot.PathStruct {
			stp := root.Stp()
			return stp.Global()
		},
		want: "/stp/global",
	}, {
		name: "simple chain",
		makePath: func(root *ocp.Device) ygot.PathStruct {
			return root.Stp().Global()
		},
		want: "/stp/global",
	}, {
		name: "simple chain with leaf",
		makePath: func(root *ocp.Device) ygot.PathStruct {
			return root.Stp().Global().EnabledProtocol()
		},
		want: "/stp/global/state/enabled-protocol",
	}, {
		name: "simple list",
		makePath: func(root *ocp.Device) ygot.PathStruct {
			return root.Interface("eth1").Ethernet().PortSpeed()
		},
		want: "/interfaces/interface[name=eth1]/ethernet/state/port-speed",
	}, {
		name: "chain with multiple lists",
		makePath: func(root *ocp.Device) ygot.PathStruct {
			return root.Interface("eth1").Subinterface(1).Ipv6().Address("1:2:3:4::").VrrpGroup(2).PreemptDelay()
		},
		want: "/interfaces/interface[name=eth1]/subinterfaces/subinterface[index=1]/ipv6/addresses/address[ip=1:2:3:4::]/vrrp/vrrp-group[virtual-router-id=2]/state/preempt-delay",
	}, {
		name: "fakeroot",
		makePath: func(root *ocp.Device) ygot.PathStruct {
			return root
		},
		want: "/",
	}, {
		name: "identity ref key",
		makePath: func(root *ocp.Device) ygot.PathStruct {
			return root.NetworkInstance("DEFAULT").Protocol(oc.OpenconfigPolicyTypes_INSTALL_PROTOCOL_TYPE_BGP, "15169").Enabled()
		},
		want: "/network-instances/network-instance[name=DEFAULT]/protocols/protocol[identifier=BGP][name=15169]/state/enabled",
	}, {
		name: "enumeration key",
		makePath: func(root *ocp.Device) ygot.PathStruct {
			return root.NetworkInstance("DEFAULT").Mpls().SignalingProtocols().Ldp().InterfaceAttributes().Interface("eth1").AddressFamily(oc.OpenconfigMplsLdp_MplsLdpAfi_IPV4).AfiName()
		},
		want: "/network-instances/network-instance[name=DEFAULT]/mpls/signaling-protocols/ldp/interface-attributes/interfaces/interface[interface-id=eth1]/address-families/address-family[afi-name=IPV4]/state/afi-name",
	}, {
		name: "union key (uint32 value)",
		makePath: func(root *ocp.Device) ygot.PathStruct {
			label100 := &oc.NetworkInstance_Mpls_SignalingProtocols_SegmentRouting_Interface_SidCounter_MplsLabel_Union_Uint32{100}
			return root.NetworkInstance("RED").Mpls().SignalingProtocols().SegmentRouting().Interface("eth1").SidCounter(label100).InOctets()
		},
		want: "/network-instances/network-instance[name=RED]/mpls/signaling-protocols/segment-routing/interfaces/interface[interface-id=eth1]/sid-counters/sid-counter[mpls-label=100]/state/in-octets",
	}, {
		name: "union key (enum value)",
		makePath: func(root *ocp.Device) ygot.PathStruct {
			implicitNull := oc.OpenconfigSegmentRouting_SidCounter_MplsLabel_IMPLICIT_NULL
			iNullInUnion := &oc.NetworkInstance_Mpls_SignalingProtocols_SegmentRouting_Interface_SidCounter_MplsLabel_Union_E_OpenconfigSegmentRouting_SidCounter_MplsLabel{implicitNull}
			return root.NetworkInstance("RED").Mpls().SignalingProtocols().SegmentRouting().Interface("eth1").SidCounter(iNullInUnion).InOctets()
		},
		want: "/network-instances/network-instance[name=RED]/mpls/signaling-protocols/segment-routing/interfaces/interface[interface-id=eth1]/sid-counters/sid-counter[mpls-label=IMPLICIT_NULL]/state/in-octets",
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verifyPath(t, tt.makePath(ocp.ForDevice(deviceId)), tt.want)
		})
	}
}
