package main

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/google/go-cmp/cmp"
	"github.com/openconfig/ygot/ygot"

	gnmipb "github.com/openconfig/gnmi/proto/gnmi"
)

func TestRenderToGNMINotifications(t *testing.T) {
	device, err := CreateAFTInstance()
	if err != nil {
		t.Fatalf("got unexpected error creating example AFT data: %v", err)
	}

	tests := []struct {
		name          string
		inStruct      ygot.GoStruct
		inTimestamp   int64
		inUsePathElem bool
		wantProtoFile string
		wantErr       bool
	}{{
		name:          "aft test, element paths",
		inStruct:      device,
		inTimestamp:   42,
		wantProtoFile: filepath.Join("testdata", "elem.pb.txt"),
	}, {
		name:          "aft test, pathelem paths",
		inStruct:      device,
		inTimestamp:   42,
		inUsePathElem: true,
		wantProtoFile: filepath.Join("testdata", "pathelem.pb.txt"),
	}}

	for _, tt := range tests {
		gotProtos, err := renderToGNMINotifications(tt.inStruct, tt.inTimestamp, tt.inUsePathElem)
		if (err != nil) != tt.wantErr {
			t.Errorf("%s: renderToGNMINotifications(%v, %v, %v): did not get expected error status, got: %v, want error: %v", tt.name, tt.inStruct, tt.inTimestamp, tt.inUsePathElem, err, tt.wantErr)
		}

		if len(gotProtos) != 1 {
			t.Errorf("%s: renderToGNMINotifications(%v, %v, %v): did not get expected number of returned protos, got: %d, want: 1", tt.name, tt.inStruct, tt.inTimestamp, tt.inUsePathElem, len(gotProtos))
      continue
		}

		wantData, err := ioutil.ReadFile(tt.wantProtoFile)
		if err != nil {
			t.Errorf("%s: ioutil.ReadFile(%v): could not read protobuf testdata file", tt.name, tt.wantProtoFile)
		}

		want := &gnmipb.Notification{}
		if err := proto.UnmarshalText(string(wantData), want); err != nil {
			t.Errorf("%s: proto.UnmarshalTextString(%v, &gnmipb.Notification{}): got unexpected error, got: %v, want: nil", tt.name, wantData, err)
		}

		if gotProtos[0].Timestamp != want.Timestamp {
			t.Errorf("%s: renderToGNMINotifications(%v, %v, %v): did not get expexted timestamp, got: %v, want: %v", tt.name, tt.inStruct, tt.inTimestamp, tt.inUsePathElem, gotProtos[0].Timestamp, want.Timestamp)
		}

		if !updateSetEqual(gotProtos[0].Update, want.Update) {
			diff := cmp.Diff(gotProtos[0], want)
			t.Errorf("%s: renderToGNMINotifications(%v, %v, %v): did not get expected output, diff(-got,+want):\n%s", tt.name, tt.inStruct, tt.inTimestamp, tt.inUsePathElem, diff)
		}
	}
}

// updateSetEqual is a helper to check whether two sets of gNMI updates are equal.
// TODO(robjs): Replace with cmp when it is able to deal with slices that are
// treated as sets in protos.
func updateSetEqual(a, b []*gnmipb.Update) bool {
	if len(a) != len(b) {
		return false
	}

	for _, aelem := range a {
		var m bool
		for _, belem := range b {
			if proto.Equal(aelem, belem) {
				m = true
				break
			}
		}

		if !m {
			return false
		}
	}

	return true
}
