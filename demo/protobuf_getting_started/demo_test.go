package main

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/kylelemons/godebug/pretty"

	ocpb "github.com/openconfig/ygot/demo/protobuf_getting_started/ribproto/openconfig"
)

func TestProtoGenerate(t *testing.T) {
	tests := []struct {
		name          string
		inTestFunc    func() *ocpb.Device
		wantTextProto string
	}{{
		name:          "simple route entry test",
		inTestFunc:    buildRouteProto,
		wantTextProto: "route_entry.txtpb",
	}}

	for _, tt := range tests {
		got := tt.inTestFunc()

		want := &ocpb.Device{}

		wantStr, err := ioutil.ReadFile(filepath.Join("testdata", tt.wantTextProto))
		if err != nil {
			t.Errorf("%s: ioutil.ReadFile(testdata/%s): could not read file, got: %v, want: nil", tt.name, tt.wantTextProto, err)
		}

		if err := proto.UnmarshalText(string(wantStr), want); err != nil {
			t.Errorf("%s: proto.UnmarshalText(file: %s): could not unmarshal test proto, got: %v, want: nil", tt.name, tt.wantTextProto, err)
		}

		if !proto.Equal(got, want) {
			t.Errorf("%s: %T: did not get expected return proto, diff(-got,+want):\n%s", tt.name, tt.inTestFunc, pretty.Compare(got, want))
		}
	}
}
