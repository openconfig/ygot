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
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	oc "github.com/openconfig/ygot/exampleoc"
	"github.com/openconfig/ygot/ygot"
	"github.com/openconfig/ygot/ypathgen"
	"google.golang.org/protobuf/proto"
)

// The device ID used throughout this test file.
const deviceId = "dev"

// verifyPath checks the given path against expected.
func verifyPath(t *testing.T, p ygot.PathStruct, wantPathStr string) {
	t.Helper()
	gotPath, _, errs := ygot.ResolvePath(p)
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

// verifyTypesEqual checks that the target and wildcard path structs are of the
// expected types. Essentially, equal indicates whether target is expected to
// be the non-wildcard version of the path struct.
func verifyTypesEqual(t *testing.T, target ygot.PathStruct, wild ygot.PathStruct, equal bool) {
	t.Helper()
	targetPathProto, _, errs := ygot.ResolvePath(target)
	if errs != nil {
		t.Fatal(errs)
	}
	wildPathProto, _, errs := ygot.ResolvePath(wild)
	if errs != nil {
		t.Fatal(errs)
	}
	targetPath, err := ygot.PathToString(targetPathProto)
	if err != nil {
		t.Fatal(err)
	}
	wildPath, err := ygot.PathToString(wildPathProto)
	if err != nil {
		t.Fatal(err)
	}

	targetType := reflect.TypeOf(target)
	wildType := reflect.TypeOf(wild)
	if equal {
		if targetType != wildType {
			t.Errorf("target(%s) and wildcard(%s) have different types: target(%T), wildcard(%T)", targetPath, wildPath, target, wild)
		}
	} else {
		if targetType == wildType {
			t.Errorf("specified non-wildcard(%s) and wildcard(%s) expected to have different types; however, they're both %T", targetPath, wildPath, target)
		} else if wantWildName := targetType.String() + ypathgen.WildcardSuffix; wildType.String() != wantWildName {
			t.Errorf("got %q for wildcard type, want %q", wildType.String(), wantWildName)
		}
	}
}

func TestCustomData(t *testing.T) {
	root := oc.DeviceRoot(deviceId)
	p := root.WithName("foo").Interface("eth1").Ethernet().PortSpeed()
	verifyPath(t, p, "/interfaces/interface[name=eth1]/ethernet/config/port-speed")

	_, customData, _ := ygot.ResolvePath(p)
	if diff := cmp.Diff(map[string]interface{}{"name": "foo"}, customData); diff != "" {
		t.Errorf("resolved customData for root does not match (-want, +got):\n%s", diff)
	}
}

// This test shows ways to reduce typing when creating similar paths.
func TestManualShortcuts(t *testing.T) {
	root := oc.DeviceRoot(deviceId)
	preemptDelay := func(intf string, subintf uint32, ip string) ygot.PathStruct {
		return root.Interface(intf).Subinterface(subintf).Ipv6().Address(ip).VrrpGroup(1).PreemptDelay()
	}

	// defining short helpers
	verifyPath(t, preemptDelay("eth1", 1, "1::"), "/interfaces/interface[name=eth1]/subinterfaces/subinterface[index=1]/ipv6/addresses/address[ip=1::]/vrrp/vrrp-group[virtual-router-id=1]/config/preempt-delay")
	verifyPath(t, preemptDelay("eth1", 2, "2:2:2:2::"), "/interfaces/interface[name=eth1]/subinterfaces/subinterface[index=2]/ipv6/addresses/address[ip=2:2:2:2::]/vrrp/vrrp-group[virtual-router-id=1]/config/preempt-delay")
	verifyPath(t, preemptDelay("eth2", 2, "::"), "/interfaces/interface[name=eth2]/subinterfaces/subinterface[index=2]/ipv6/addresses/address[ip=::]/vrrp/vrrp-group[virtual-router-id=1]/config/preempt-delay")

	// re-using prefixes
	intf1 := root.InterfaceAny()
	verifyPath(t, intf1.Subinterface(3), "/interfaces/interface[name=*]/subinterfaces/subinterface[index=3]")
	verifyPath(t, intf1.Subinterface(4), "/interfaces/interface[name=*]/subinterfaces/subinterface[index=4]")
}

func TestPathCreation(t *testing.T) {
	tests := []struct {
		name     string
		makePath func(*oc.DevicePath) ygot.PathStruct
		wantPath string
	}{{
		name: "simple",
		makePath: func(root *oc.DevicePath) ygot.PathStruct {
			return root.Stp()
		},
		wantPath: "/stp",
	}, {
		name: "simple prefixing",
		makePath: func(root *oc.DevicePath) ygot.PathStruct {
			stp := root.Stp()
			return stp.Global()
		},
		wantPath: "/stp/global",
	}, {
		name: "simple chain",
		makePath: func(root *oc.DevicePath) ygot.PathStruct {
			return root.Stp().Global()
		},
		wantPath: "/stp/global",
	}, {
		name: "simple chain with leaf",
		makePath: func(root *oc.DevicePath) ygot.PathStruct {
			return root.Stp().Global().EnabledProtocol()
		},
		wantPath: "/stp/global/config/enabled-protocol",
	}, {
		name: "simple list",
		makePath: func(root *oc.DevicePath) ygot.PathStruct {
			return root.Interface("eth1").Ethernet().PortSpeed()
		},
		wantPath: "/interfaces/interface[name=eth1]/ethernet/config/port-speed",
	}, {
		name: "chain with multiple lists",
		makePath: func(root *oc.DevicePath) ygot.PathStruct {
			return root.Interface("eth1").Subinterface(1).Ipv6().Address("1:2:3:4::").VrrpGroup(2).PreemptDelay()
		},
		wantPath: "/interfaces/interface[name=eth1]/subinterfaces/subinterface[index=1]/ipv6/addresses/address[ip=1:2:3:4::]/vrrp/vrrp-group[virtual-router-id=2]/config/preempt-delay",
	}, {
		name: "fakeroot",
		makePath: func(root *oc.DevicePath) ygot.PathStruct {
			return root
		},
		wantPath: "/",
	}, {
		name: "identity ref key",
		makePath: func(root *oc.DevicePath) ygot.PathStruct {
			return root.NetworkInstance("DEFAULT").Protocol(oc.PolicyTypes_INSTALL_PROTOCOL_TYPE_BGP, "15169").Enabled()
		},
		wantPath: "/network-instances/network-instance[name=DEFAULT]/protocols/protocol[identifier=BGP][name=15169]/config/enabled",
	}, {
		name: "enumeration key",
		makePath: func(root *oc.DevicePath) ygot.PathStruct {
			return root.NetworkInstance("DEFAULT").Mpls().SignalingProtocols().Ldp().InterfaceAttributes().Interface("eth1").AddressFamily(oc.MplsLdp_MplsLdpAfi_IPV4).AfiName()
		},
		wantPath: "/network-instances/network-instance[name=DEFAULT]/mpls/signaling-protocols/ldp/interface-attributes/interfaces/interface[interface-id=eth1]/address-families/address-family[afi-name=IPV4]/config/afi-name",
	}, {
		name: "union key (uint32 value)",
		makePath: func(root *oc.DevicePath) ygot.PathStruct {
			return root.NetworkInstance("RED").Mpls().SignalingProtocols().SegmentRouting().Interface("eth1").SidCounter(oc.UnionUint32(100)).InOctets()
		},
		wantPath: "/network-instances/network-instance[name=RED]/mpls/signaling-protocols/segment-routing/interfaces/interface[interface-id=eth1]/sid-counters/sid-counter[mpls-label=100]/state/in-octets",
	}, {
		name: "union key (enum value)",
		makePath: func(root *oc.DevicePath) ygot.PathStruct {
			return root.NetworkInstance("RED").Mpls().SignalingProtocols().SegmentRouting().Interface("eth1").SidCounter(oc.MplsTypes_MplsLabel_Enum_IMPLICIT_NULL).InOctets()
		},
		wantPath: "/network-instances/network-instance[name=RED]/mpls/signaling-protocols/segment-routing/interfaces/interface[interface-id=eth1]/sid-counters/sid-counter[mpls-label=IMPLICIT_NULL]/state/in-octets",
	}, {
		name: "Builder API constructor",
		makePath: func(root *oc.DevicePath) ygot.PathStruct {
			return root.Lldp().Interface("foo").Neighbor("mars").TlvAny()
		},
		wantPath: "/lldp/interfaces/interface[name=foo]/neighbors/neighbor[id=mars]/custom-tlvs/tlv[type=*][oui=*][oui-subtype=*]",
	}, {
		name: "Builder API constructor going beyond the list",
		makePath: func(root *oc.DevicePath) ygot.PathStruct {
			return root.Lldp().Interface("foo").Neighbor("mars").TlvAny().Value()
		},
		wantPath: "/lldp/interfaces/interface[name=foo]/neighbors/neighbor[id=mars]/custom-tlvs/tlv[type=*][oui=*][oui-subtype=*]/state/value",
	}, {
		name: "Builder API builder for type",
		makePath: func(root *oc.DevicePath) ygot.PathStruct {
			return root.Lldp().Interface("foo").Neighbor("mars").TlvAny().WithType(3)
		},
		wantPath: "/lldp/interfaces/interface[name=foo]/neighbors/neighbor[id=mars]/custom-tlvs/tlv[type=3][oui=*][oui-subtype=*]",
	}, {
		name: "Builder API builder for oui",
		makePath: func(root *oc.DevicePath) ygot.PathStruct {
			return root.Lldp().Interface("foo").Neighbor("mars").TlvAny().WithOui("bar")
		},
		wantPath: "/lldp/interfaces/interface[name=foo]/neighbors/neighbor[id=mars]/custom-tlvs/tlv[type=*][oui=bar][oui-subtype=*]",
	}, {
		name: "Builder API builder for subtype",
		makePath: func(root *oc.DevicePath) ygot.PathStruct {
			return root.Lldp().Interface("foo").Neighbor("mars").TlvAny().WithOuiSubtype("baz").Value()
		},
		wantPath: "/lldp/interfaces/interface[name=foo]/neighbors/neighbor[id=mars]/custom-tlvs/tlv[type=*][oui=*][oui-subtype=baz]/state/value",
	}, {
		name: "Builder API builder for type and oui",
		makePath: func(root *oc.DevicePath) ygot.PathStruct {
			return root.Lldp().Interface("foo").Neighbor("mars").TlvAny().WithType(3).WithOui("bar")
		},
		wantPath: "/lldp/interfaces/interface[name=foo]/neighbors/neighbor[id=mars]/custom-tlvs/tlv[type=3][oui=bar][oui-subtype=*]",
	}, {
		name: "Builder API builder for type and oui and oui-subtype",
		makePath: func(root *oc.DevicePath) ygot.PathStruct {
			return root.Lldp().Interface("foo").Neighbor("mars").TlvAny().WithOui("bar").WithType(3).WithOuiSubtype("baz")
		},
		wantPath: "/lldp/interfaces/interface[name=foo]/neighbors/neighbor[id=mars]/custom-tlvs/tlv[type=3][oui=bar][oui-subtype=baz]",
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verifyPath(t, tt.makePath(oc.DeviceRoot(deviceId)), tt.wantPath)
		})
	}
}

func TestWildcardPathCreation(t *testing.T) {
	tests := []struct {
		name            string
		makePath        func(*oc.DevicePath) ygot.PathStruct
		wantPath        string
		makeWildPath    func(*oc.DevicePath) ygot.PathStruct
		wantWildPath    string
		bothAreWildcard bool
	}{{
		name: "check interface wildcard type",
		makePath: func(root *oc.DevicePath) ygot.PathStruct {
			return root.Interface("eth1")
		},
		wantPath: "/interfaces/interface[name=eth1]",
		makeWildPath: func(root *oc.DevicePath) ygot.PathStruct {
			return root.InterfaceAny()
		},
		wantWildPath: "/interfaces/interface[name=*]",
	}, {
		name: "check 2nd-level wildcard type",
		makePath: func(root *oc.DevicePath) ygot.PathStruct {
			return root.Interface("eth1").Subinterface(1)
		},
		wantPath: "/interfaces/interface[name=eth1]/subinterfaces/subinterface[index=1]",
		makeWildPath: func(root *oc.DevicePath) ygot.PathStruct {
			return root.InterfaceAny().Subinterface(1)
		},
		wantWildPath: "/interfaces/interface[name=*]/subinterfaces/subinterface[index=1]",
	}, {
		name: "check 2nd-level wildcard type with different path",
		makePath: func(root *oc.DevicePath) ygot.PathStruct {
			return root.Interface("eth1").Subinterface(1)
		},
		wantPath: "/interfaces/interface[name=eth1]/subinterfaces/subinterface[index=1]",
		makeWildPath: func(root *oc.DevicePath) ygot.PathStruct {
			return root.Interface("eth1").SubinterfaceAny()
		},
		wantWildPath: "/interfaces/interface[name=eth1]/subinterfaces/subinterface[index=*]",
	}, {
		name: "check 2nd-level wildcard type with multiple wildcards",
		makePath: func(root *oc.DevicePath) ygot.PathStruct {
			return root.Interface("eth1").SubinterfaceAny()
		},
		wantPath: "/interfaces/interface[name=eth1]/subinterfaces/subinterface[index=*]",
		makeWildPath: func(root *oc.DevicePath) ygot.PathStruct {
			return root.InterfaceAny().SubinterfaceAny()
		},
		wantWildPath:    "/interfaces/interface[name=*]/subinterfaces/subinterface[index=*]",
		bothAreWildcard: true,
	}, {
		name: "check 3rd-level wildcard type",
		makePath: func(root *oc.DevicePath) ygot.PathStruct {
			return root.Interface("eth1").Subinterface(1).Ipv6()
		},
		wantPath: "/interfaces/interface[name=eth1]/subinterfaces/subinterface[index=1]/ipv6",
		makeWildPath: func(root *oc.DevicePath) ygot.PathStruct {
			return root.InterfaceAny().Subinterface(1).Ipv6()
		},
		wantWildPath: "/interfaces/interface[name=*]/subinterfaces/subinterface[index=1]/ipv6",
	}, {
		name: "check 4th-level wildcard type",
		makePath: func(root *oc.DevicePath) ygot.PathStruct {
			return root.Interface("eth1").Subinterface(1).Ipv6().Address("1:2:3:4::")
		},
		wantPath: "/interfaces/interface[name=eth1]/subinterfaces/subinterface[index=1]/ipv6/addresses/address[ip=1:2:3:4::]",
		makeWildPath: func(root *oc.DevicePath) ygot.PathStruct {
			return root.InterfaceAny().Subinterface(1).Ipv6().Address("1:2:3:4::")
		},
		wantWildPath: "/interfaces/interface[name=*]/subinterfaces/subinterface[index=1]/ipv6/addresses/address[ip=1:2:3:4::]",
	}, {
		name: "check 5th-level wildcard type",
		makePath: func(root *oc.DevicePath) ygot.PathStruct {
			return root.Interface("eth1").Subinterface(1).Ipv6().Address("1:2:3:4::").VrrpGroup(2)
		},
		wantPath: "/interfaces/interface[name=eth1]/subinterfaces/subinterface[index=1]/ipv6/addresses/address[ip=1:2:3:4::]/vrrp/vrrp-group[virtual-router-id=2]",
		makeWildPath: func(root *oc.DevicePath) ygot.PathStruct {
			return root.InterfaceAny().Subinterface(1).Ipv6().Address("1:2:3:4::").VrrpGroup(2)
		},
		wantWildPath: "/interfaces/interface[name=*]/subinterfaces/subinterface[index=1]/ipv6/addresses/address[ip=1:2:3:4::]/vrrp/vrrp-group[virtual-router-id=2]",
	}, {
		name: "check 6th-level leaf wildcard type",
		makePath: func(root *oc.DevicePath) ygot.PathStruct {
			return root.Interface("eth1").Subinterface(1).Ipv6().Address("1:2:3:4::").VrrpGroup(2).PreemptDelay()
		},
		wantPath: "/interfaces/interface[name=eth1]/subinterfaces/subinterface[index=1]/ipv6/addresses/address[ip=1:2:3:4::]/vrrp/vrrp-group[virtual-router-id=2]/config/preempt-delay",
		makeWildPath: func(root *oc.DevicePath) ygot.PathStruct {
			return root.InterfaceAny().Subinterface(1).Ipv6().Address("1:2:3:4::").VrrpGroup(2).PreemptDelay()
		},
		wantWildPath: "/interfaces/interface[name=*]/subinterfaces/subinterface[index=1]/ipv6/addresses/address[ip=1:2:3:4::]/vrrp/vrrp-group[virtual-router-id=2]/config/preempt-delay",
	}, {
		name: "check 6th-level leaf wildcard type in a different path",
		makePath: func(root *oc.DevicePath) ygot.PathStruct {
			return root.Interface("eth1").Subinterface(1).Ipv6().Address("1:2:3:4::").VrrpGroup(2).PreemptDelay()
		},
		wantPath: "/interfaces/interface[name=eth1]/subinterfaces/subinterface[index=1]/ipv6/addresses/address[ip=1:2:3:4::]/vrrp/vrrp-group[virtual-router-id=2]/config/preempt-delay",
		makeWildPath: func(root *oc.DevicePath) ygot.PathStruct {
			return root.Interface("eth1").Subinterface(1).Ipv6().AddressAny().VrrpGroup(2).PreemptDelay()
		},
		wantWildPath: "/interfaces/interface[name=eth1]/subinterfaces/subinterface[index=1]/ipv6/addresses/address[ip=*]/vrrp/vrrp-group[virtual-router-id=2]/config/preempt-delay",
	}, {
		name: "check 6th-level leaf wildcard types are same between different paths",
		makePath: func(root *oc.DevicePath) ygot.PathStruct {
			return root.InterfaceAny().Subinterface(1).Ipv6().Address("1:2:3:4::").VrrpGroup(2).PreemptDelay()
		},
		wantPath: "/interfaces/interface[name=*]/subinterfaces/subinterface[index=1]/ipv6/addresses/address[ip=1:2:3:4::]/vrrp/vrrp-group[virtual-router-id=2]/config/preempt-delay",
		makeWildPath: func(root *oc.DevicePath) ygot.PathStruct {
			return root.Interface("eth1").Subinterface(1).Ipv6().AddressAny().VrrpGroup(2).PreemptDelay()
		},
		wantWildPath:    "/interfaces/interface[name=eth1]/subinterfaces/subinterface[index=1]/ipv6/addresses/address[ip=*]/vrrp/vrrp-group[virtual-router-id=2]/config/preempt-delay",
		bothAreWildcard: true,
	}, {
		name: "check 6th-level leaf wildcard type for multiple wildcards",
		makePath: func(root *oc.DevicePath) ygot.PathStruct {
			return root.InterfaceAny().Subinterface(1).Ipv6().Address("1:2:3:4::").VrrpGroup(2).PreemptDelay()
		},
		wantPath: "/interfaces/interface[name=*]/subinterfaces/subinterface[index=1]/ipv6/addresses/address[ip=1:2:3:4::]/vrrp/vrrp-group[virtual-router-id=2]/config/preempt-delay",
		makeWildPath: func(root *oc.DevicePath) ygot.PathStruct {
			return root.InterfaceAny().SubinterfaceAny().Ipv6().AddressAny().VrrpGroupAny().PreemptDelay()
		},
		wantWildPath:    "/interfaces/interface[name=*]/subinterfaces/subinterface[index=*]/ipv6/addresses/address[ip=*]/vrrp/vrrp-group[virtual-router-id=*]/config/preempt-delay",
		bothAreWildcard: true,
	}, {
		name: "multi-keyed wildcarding",
		makePath: func(root *oc.DevicePath) ygot.PathStruct {
			return root.Acl().AclSet("foo", oc.Acl_ACL_TYPE_ACL_IPV4)
		},
		wantPath: "/acl/acl-sets/acl-set[name=foo][type=ACL_IPV4]",
		makeWildPath: func(root *oc.DevicePath) ygot.PathStruct {
			return root.Acl().AclSetAny()
		},
		wantWildPath: "/acl/acl-sets/acl-set[name=*][type=*]",
	}, {
		name: "multi-keyed wildcarding: AnyName",
		makePath: func(root *oc.DevicePath) ygot.PathStruct {
			return root.Acl().AclSetAny()
		},
		wantPath: "/acl/acl-sets/acl-set[name=*][type=*]",
		makeWildPath: func(root *oc.DevicePath) ygot.PathStruct {
			return root.Acl().AclSetAnyName(oc.Acl_ACL_TYPE_ACL_IPV4)
		},
		wantWildPath:    "/acl/acl-sets/acl-set[name=*][type=ACL_IPV4]",
		bothAreWildcard: true,
	}, {
		name: "multi-keyed wildcarding: AnyType",
		makePath: func(root *oc.DevicePath) ygot.PathStruct {
			return root.Acl().AclSetAny()
		},
		wantPath: "/acl/acl-sets/acl-set[name=*][type=*]",
		makeWildPath: func(root *oc.DevicePath) ygot.PathStruct {
			return root.Acl().AclSetAnyType("foo")
		},
		wantWildPath:    "/acl/acl-sets/acl-set[name=foo][type=*]",
		bothAreWildcard: true,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			device := oc.DeviceRoot(deviceId)

			target := tt.makePath(device)
			verifyPath(t, target, tt.wantPath)
			wild := tt.makeWildPath(device)
			verifyPath(t, wild, tt.wantWildPath)

			verifyTypesEqual(t, target, wild, tt.bothAreWildcard)
		})
	}
}
