package integration_tests

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/openconfig/gribi/v1/proto/gribi_aft"
	"github.com/openconfig/ygot/protomap"
	"github.com/openconfig/ygot/testutil"
	"github.com/openconfig/ygot/ygot"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
)

func mustPath(p string) *gpb.Path {
	sp, err := ygot.StringToStructuredPath(p)
	if err != nil {
		panic(fmt.Sprintf("cannot parse path %s to proto, %v", p, err))
	}
	return sp
}

func TestGRIBIAFT(t *testing.T) {
	tests := []struct {
		desc      string
		inProto   proto.Message
		wantPaths map[*gpb.Path]interface{}
		wantErr   bool
	}{{
		desc: "IPv4 Entry with key",
		inProto: &gribi_aft.Afts{
			Ipv4Entry: []*gribi_aft.Afts_Ipv4EntryKey{{
				Prefix:    "1.0.0.0/24",
				Ipv4Entry: &gribi_aft.Afts_Ipv4Entry{},
			}},
		},
		wantPaths: map[*gpb.Path]interface{}{
			mustPath("afts/ipv4-unicast/ipv4-entry[prefix=1.0.0.0/24]/state/prefix"): "1.0.0.0/24",
			mustPath("afts/ipv4-unicast/ipv4-entry[prefix=1.0.0.0/24]/prefix"):       "1.0.0.0/24",
		},
	}, {
		desc: "IPv4 Entry with nil prefix",
		inProto: &gribi_aft.Afts{
			Ipv4Entry: []*gribi_aft.Afts_Ipv4EntryKey{{}},
		},
		wantErr: true,
	}, {
		desc: "IPv4 Entry with nil list member",
		inProto: &gribi_aft.Afts{
			Ipv4Entry: []*gribi_aft.Afts_Ipv4EntryKey{{
				Prefix: "2.2.2.2/32",
			}},
		},
		wantErr: true,
	}, {
		desc: "MPLS label entry - oneof key",
		inProto: &gribi_aft.Afts{
			LabelEntry: []*gribi_aft.Afts_LabelEntryKey{{
				Label: &gribi_aft.Afts_LabelEntryKey_LabelUint64{
					LabelUint64: 32,
				},
			}},
		},
		wantPaths: map[*gpb.Path]interface{}{
			mustPath("afts/label-entries/label-entry[label=32]/state/label"): 32,
		},
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got, err := protomap.PathsFromProto(tt.inProto)
			if (err != nil) != tt.wantErr {
				t.Fatalf("did not get expected error, got: %v, wantErr? %v", err, tt.wantErr)
			}
			if diff := cmp.Diff(got, tt.wantPaths, protocmp.Transform(), cmpopts.EquateEmpty(), cmpopts.SortMaps(testutil.PathLess)); diff != "" {
				t.Fatalf("did not get expected results, diff(-got,+want):\n%s", diff)
			}
		})
	}
}
