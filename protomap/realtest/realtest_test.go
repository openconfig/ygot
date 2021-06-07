package realtest

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/openconfig/gribi/v1/proto/gribi_aft"
	"github.com/openconfig/ygot/protomap"
	"google.golang.org/protobuf/proto"
)

func TestGRIBIAFT(t *testing.T) {
	tests := []struct {
		desc      string
		inProto   proto.Message
		wantPaths map[string]interface{}
		wantErr   bool
	}{{
		desc: "IPv4 Entry with key",
		inProto: &gribi_aft.Afts_Ipv4EntryKey{
			Prefix: "1.0.0.0/24",
		},
		wantPaths: map[string]interface{}{},
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got, err := protomap.PathsFromProto(tt.inProto)
			if (err != nil) != tt.wantErr {
				t.Fatalf("did not get expected error, got: %v, wantErr? %v", err, tt.wantErr)
			}
			if diff := cmp.Diff(got, tt.wantPaths); diff != "" {
				t.Fatalf("did not get expected paths, diff(-got,+want):\n%s", diff)
			}
		})
	}
}
