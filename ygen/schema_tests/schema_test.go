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
