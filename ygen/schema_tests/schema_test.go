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

// Package schematest is used for testing with the default OpenConfig generated
// structs.
package schematest

import (
	"reflect"
	"testing"

	"github.com/openconfig/ygot/exampleoc"
	"github.com/openconfig/ygot/ygot"
)

func TestSimpleListRename(t *testing.T) {
	in := &exampleoc.Device{}
	if _, err := in.NewInterface("eth0"); err != nil {
		t.Fatalf("could not create eth0 entry, got: %v, want: nil", err)
	}

	if err := in.RenameInterface("eth0", "eth1"); err != nil {
		t.Fatalf("could not rename eth0 entry, got: %v, want: nil", err)
	}

	if _, ok := in.Interface["eth0"]; ok {
		t.Fatalf("did not remove eth0 from list")
	}

	if _, ok := in.Interface["eth1"]; !ok {
		t.Fatalf("did not populate eth1 in list")
	}

	if !reflect.DeepEqual(in.Interface["eth1"].Name, ygot.String("eth1")) {
		t.Errorf("did not get correct name value, got: %v, want: eth1", *in.Interface["eth1"].Name)
	}

	if _, err := in.NewInterface("eth2"); err != nil {
		t.Fatalf("could not create eth2 entry, got: %v, want: nil", err)
	}

	if err := in.RenameInterface("eth2", "eth1"); err == nil {
		t.Fatalf("incorrectly overwrote eth1 entry, got: %v, want: error", err)
	}
}

func TestMultiKeyListRename(t *testing.T) {
	in := &exampleoc.Device{}
	ni, err := in.NewNetworkInstance("DEFAULT")
	if err != nil {
		t.Fatalf("could not create DEFAULT network instance, got: %v, want: nil", err)
	}

	if _, err := ni.NewProtocol(exampleoc.OpenconfigPolicyTypes_INSTALL_PROTOCOL_TYPE_BGP, "15169"); err != nil {
		t.Fatalf("could not create BGP protocol instance, got: %v, want: nil", err)
	}

	oldBGP := exampleoc.NetworkInstance_Protocol_Key{Identifier: exampleoc.OpenconfigPolicyTypes_INSTALL_PROTOCOL_TYPE_BGP, Name: "15169"}
	newBGP := exampleoc.NetworkInstance_Protocol_Key{Identifier: exampleoc.OpenconfigPolicyTypes_INSTALL_PROTOCOL_TYPE_BGP, Name: "36040"}
	if err := ni.RenameProtocol(oldBGP, newBGP); err != nil {
		t.Fatalf("could not rename BGP protocol instance, got: %v, want: nil", err)
	}

	if _, ok := ni.Protocol[oldBGP]; ok {
		t.Fatalf("did not remove old BGP protocol instance, got: %v, want: nil", err)
	}

	if _, ok := ni.Protocol[newBGP]; !ok {
		t.Fatalf("did not find new BGP protocol instance, got: %v, want: nil", err)
	}

	if ni.Protocol[newBGP].Identifier != exampleoc.OpenconfigPolicyTypes_INSTALL_PROTOCOL_TYPE_BGP {
		t.Errorf("did not have correct identifier in newBGP, got: %v, want: OpenconfigPolicyTypes_INSTALL_PROTOCOL_TYPE_BGP", ni.Protocol[newBGP].Identifier)
	}

	if !reflect.DeepEqual(ni.Protocol[newBGP].Name, ygot.String("36040")) {
		t.Errorf("did not have correct name in newBGP, got: %v, want: 36040", *ni.Protocol[newBGP].Name)
	}
}

func TestSimpleKeyAppend(t *testing.T) {
	in := &exampleoc.Device{}
	ni := &exampleoc.NetworkInstance{
		Name: ygot.String("DEFAULT"),
	}
	if err := in.AppendNetworkInstance(ni); err != nil {
		t.Errorf("AppendNetworkInstance(%v): did not get expected error, got: %v, want: nil", ni, err)
	}

	if _, ok := in.NetworkInstance["DEFAULT"]; !ok {
		t.Errorf("AppendNetworkInstance(%v): did not find element after append, got: %v, want: true", ni, ok)
	}
}

func TestMultiKeyAppend(t *testing.T) {
	in := &exampleoc.Device{}
	if _, err := in.NewNetworkInstance("DEFAULT"); err != nil {
		t.Errorf("NewNetworkInstance('DEFAULT'): did not get expected error status, got: %v, want: nil", err)
	}

	p := &exampleoc.NetworkInstance_Protocol{
		Identifier: exampleoc.OpenconfigPolicyTypes_INSTALL_PROTOCOL_TYPE_BGP,
		Name:       ygot.String("15169"),
	}

	if err := in.NetworkInstance["DEFAULT"].AppendProtocol(p); err != nil {
		t.Errorf("AppendProtocol(%v): did not get expected error status, got: %v, want: nil", p, err)
	}

	wantKey := exampleoc.NetworkInstance_Protocol_Key{Identifier: exampleoc.OpenconfigPolicyTypes_INSTALL_PROTOCOL_TYPE_BGP, Name: "15169"}
	if _, ok := in.NetworkInstance["DEFAULT"].Protocol[wantKey]; !ok {
		t.Errorf("AppendProtocol(%v): did not find element after append, got: %v, want: true", p, ok)
	}
}

func TestGetOrCreateSimpleElement(t *testing.T) {
	d := &exampleoc.Device{}
	v := d.GetOrCreateSystem().GetOrCreateDns()
	v.Search = []string{"rob.sh", "google.com"}

	if got, want := d.System.Dns.Search, []string{"rob.sh", "google.com"}; !reflect.DeepEqual(got, want) {
		t.Errorf("GetOrCreateSystem().GetOrCreateDns(): got incorrect return value, got: %v, want: %v", got, want)
	}
}

func TestGetOrCreateSimpleList(t *testing.T) {
	d := &exampleoc.Device{}
	d.GetOrCreateInterface("eth0").GetOrCreateHoldTime().Up = ygot.Uint32(42)

	if got, want := *d.Interface["eth0"].HoldTime.Up, uint32(42); got != want {
		t.Errorf("GetOrCreateInterface('eth0').GetOrCreateHoldTime().Up: got incorrect return value, got: %v, want: %v", got, want)
	}
}

func TestGetOrCreateMultiKeyList(t *testing.T) {
	d := &exampleoc.Device{}
	d.GetOrCreateNetworkInstance("DEFAULT").GetOrCreateProtocol(exampleoc.OpenconfigPolicyTypes_INSTALL_PROTOCOL_TYPE_ISIS, "0").GetOrCreateIsis().GetOrCreateGlobal().MaxEcmpPaths = ygot.Uint8(42)

	if got, want := *d.NetworkInstance["DEFAULT"].Protocol[exampleoc.NetworkInstance_Protocol_Key{Identifier: exampleoc.OpenconfigPolicyTypes_INSTALL_PROTOCOL_TYPE_ISIS, Name: "0"}].Isis.Global.MaxEcmpPaths, uint8(42); got != want {
		t.Errorf("GetOrCreateNetworkInstance('DEFAULT').GetOrCreateProtocol(ISIS, '0').GetOrCreateGlobal().MaxEcmpPaths: got incorrect return value, got: %v, want: %v", got, want)
	}
}

func TestGetterChaining(t *testing.T) {
	d := &exampleoc.Device{}
	if got := d.GetSystem().GetAaa(); got != nil {
		t.Errorf("chained getters: GetSystem().GetAaa() did not return nil, got: %v, want: nil", got)
	}

	want := "eth0"
	_ = d.GetOrCreateInterface(want)
	if got := d.GetInterface(want).Name; got == nil || *got != want {
		t.Errorf("get list: GetInterface(%s), did not get expected result, got: %v, want: %v", want, got, want)
	}

	if got := d.GetInterface("does-not-exist").GetCounters(); got != nil {
		t.Errorf(`get list with missing key: GetInterface("does-not-exist"), did not get expected result, got: %v, want: nil`, got)
	}
}
