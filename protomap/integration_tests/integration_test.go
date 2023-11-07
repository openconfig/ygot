package integration_tests

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/openconfig/gnmi/value"
	"github.com/openconfig/ygot/protomap"
	"github.com/openconfig/ygot/testutil"
	"github.com/openconfig/ygot/ygot"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
	wpb "github.com/openconfig/ygot/proto/ywrapper"
	aftpb "github.com/openconfig/ygot/protomap/integration_tests/testdata/gribi_aft"
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
		inProto: &aftpb.Afts{
			Ipv4Entry: []*aftpb.Afts_Ipv4EntryKey{{
				Prefix:    "1.0.0.0/24",
				Ipv4Entry: &aftpb.Afts_Ipv4Entry{},
			}},
		},
		wantPaths: map[*gpb.Path]interface{}{
			mustPath("afts/ipv4-unicast/ipv4-entry[prefix=1.0.0.0/24]/state/prefix"): "1.0.0.0/24",
			mustPath("afts/ipv4-unicast/ipv4-entry[prefix=1.0.0.0/24]/prefix"):       "1.0.0.0/24",
		},
	}, {
		desc: "IPv4 Entry with nil prefix",
		inProto: &aftpb.Afts{
			Ipv4Entry: []*aftpb.Afts_Ipv4EntryKey{{}},
		},
		wantErr: true,
	}, {
		desc: "IPv4 Entry with nil list member",
		inProto: &aftpb.Afts{
			Ipv4Entry: []*aftpb.Afts_Ipv4EntryKey{{
				Prefix: "2.2.2.2/32",
			}},
		},
		wantErr: true,
	}, {
		desc: "MPLS label entry - oneof key",
		inProto: &aftpb.Afts{
			LabelEntry: []*aftpb.Afts_LabelEntryKey{{
				Label: &aftpb.Afts_LabelEntryKey_LabelUint64{
					LabelUint64: 32,
				},
				LabelEntry: &aftpb.Afts_LabelEntry{},
			}},
		},
		wantPaths: map[*gpb.Path]interface{}{
			mustPath("afts/mpls/label-entry[label=32]/state/label"): uint64(32),
			mustPath("afts/mpls/label-entry[label=32]/label"):       uint64(32),
		},
	}, {
		desc: "NH entry with label stack",
		inProto: &aftpb.Afts{
			NextHop: []*aftpb.Afts_NextHopKey{{
				Index: 1,
				NextHop: &aftpb.Afts_NextHop{
					PushedMplsLabelStack: []*aftpb.Afts_NextHop_PushedMplsLabelStackUnion{{
						PushedMplsLabelStackUint64: 42,
					}, {
						PushedMplsLabelStackUint64: 84,
					}},
				},
			}},
		},
		wantPaths: map[*gpb.Path]interface{}{
			mustPath("afts/next-hops/next-hop[index=1]/state/pushed-mpls-label-stack"): []interface{}{uint64(42), uint64(84)},
			mustPath("afts/next-hops/next-hop[index=1]/index"):                         uint64(1),
			mustPath("afts/next-hops/next-hop[index=1]/state/index"):                   uint64(1),
		},
	}, {
		desc: "NHG entry",
		inProto: &aftpb.Afts{
			NextHopGroup: []*aftpb.Afts_NextHopGroupKey{{
				Id: 1,
				NextHopGroup: &aftpb.Afts_NextHopGroup{
					NextHop: []*aftpb.Afts_NextHopGroup_NextHopKey{{
						Index: 1,
						NextHop: &aftpb.Afts_NextHopGroup_NextHop{
							Weight: &wpb.UintValue{Value: 1},
						},
					}},
				},
			}},
		},
		wantPaths: map[*gpb.Path]interface{}{
			mustPath("afts/next-hop-groups/next-hop-group[id=1]/id"):                                       uint64(1),
			mustPath("afts/next-hop-groups/next-hop-group[id=1]/state/id"):                                 uint64(1),
			mustPath("afts/next-hop-groups/next-hop-group[id=1]/next-hops/next-hop[index=1]/index"):        uint64(1),
			mustPath("afts/next-hop-groups/next-hop-group[id=1]/next-hops/next-hop[index=1]/state/index"):  uint64(1),
			mustPath("afts/next-hop-groups/next-hop-group[id=1]/next-hops/next-hop[index=1]/state/weight"): uint64(1),
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

func mustValue(t *testing.T, v any) *gpb.TypedValue {
	tv, err := value.FromScalar(v)
	if err != nil {
		t.Fatalf("cannot create gNMI TypedValue from %v %T, err:  %v", v, v, err)
	}
	return tv
}

func TestGRIBIAFTToStruct(t *testing.T) {
	tests := []struct {
		desc      string
		inPaths   map[*gpb.Path]interface{}
		inProto   proto.Message
		inPrefix  *gpb.Path
		wantProto proto.Message
		wantErr   bool
	}{{
		desc: "ipv4 prefix",
		inPaths: map[*gpb.Path]interface{}{
			mustPath("state/entry-metadata"): mustValue(t, []byte{1, 2, 3}),
		},
		inProto:  &aftpb.Afts_Ipv4Entry{},
		inPrefix: mustPath("afts/ipv4-unicast/ipv4-entry"),
		wantProto: &aftpb.Afts_Ipv4Entry{
			EntryMetadata: &wpb.BytesValue{Value: []byte{1, 2, 3}},
		},
	}, {
		desc: "map next-hop-group",
		inPaths: map[*gpb.Path]interface{}{
			mustPath("next-hops/next-hop[index=1]/index"):        mustValue(t, uint64(1)),
			mustPath("next-hops/next-hop[index=1]/state/index"):  mustValue(t, uint64(1)),
			mustPath("next-hops/next-hop[index=1]/state/weight"): mustValue(t, uint64(1)),
		},
		inProto: &aftpb.Afts_NextHopGroup{},
		inPrefix: &gpb.Path{
			Elem: []*gpb.PathElem{{
				Name: "afts",
			}, {
				Name: "next-hop-groups",
			}, {
				Name: "next-hop-group",
			}},
		},
		wantProto: &aftpb.Afts_NextHopGroup{
			NextHop: []*aftpb.Afts_NextHopGroup_NextHopKey{{
				Index: 1,
				NextHop: &aftpb.Afts_NextHopGroup_NextHop{
					Weight: &wpb.UintValue{Value: 1},
				},
			}},
		},
	}, {
		desc: "multiple NHGs",
		inPaths: map[*gpb.Path]interface{}{
			mustPath("next-hops/next-hop[index=1]/index"):        mustValue(t, uint64(1)),
			mustPath("next-hops/next-hop[index=1]/state/index"):  mustValue(t, uint64(1)),
			mustPath("next-hops/next-hop[index=1]/state/weight"): mustValue(t, uint64(1)),
			mustPath("next-hops/next-hop[index=2]/index"):        mustValue(t, uint64(2)),
			mustPath("next-hops/next-hop[index=2]/state/index"):  mustValue(t, uint64(2)),
			mustPath("next-hops/next-hop[index=2]/state/weight"): mustValue(t, uint64(2)),
		},
		inProto: &aftpb.Afts_NextHopGroup{},
		inPrefix: &gpb.Path{
			Elem: []*gpb.PathElem{{
				Name: "afts",
			}, {
				Name: "next-hop-groups",
			}, {
				Name: "next-hop-group",
			}},
		},
		wantProto: &aftpb.Afts_NextHopGroup{
			NextHop: []*aftpb.Afts_NextHopGroup_NextHopKey{{
				Index: 1,
				NextHop: &aftpb.Afts_NextHopGroup_NextHop{
					Weight: &wpb.UintValue{Value: 1},
				},
			}, {
				Index: 2,
				NextHop: &aftpb.Afts_NextHopGroup_NextHop{
					Weight: &wpb.UintValue{Value: 2},
				},
			}},
		},
	}, {
		desc: "embedded field in next-hop",
		inPaths: map[*gpb.Path]interface{}{
			mustPath("ip-in-ip/state/src-ip"): mustValue(t, "1.1.1.1"),
		},
		inProto:  &aftpb.Afts_NextHop{},
		inPrefix: mustPath("afts/next-hops/next-hop"),
		wantProto: &aftpb.Afts_NextHop{
			IpInIp: &aftpb.Afts_NextHop_IpInIp{
				SrcIp: &wpb.StringValue{Value: "1.1.1.1"},
			},
		},
	}, {
		desc: "pushed mpls label stack",
		inPaths: map[*gpb.Path]interface{}{
			mustPath("state/pushed-mpls-label-stack"): &gpb.TypedValue{
				Value: &gpb.TypedValue_LeaflistVal{
					LeaflistVal: &gpb.ScalarArray{
						Element: []*gpb.TypedValue{
							mustValue(t, uint64(20)),
							mustValue(t, uint64(30)),
							mustValue(t, uint64(40)),
							mustValue(t, uint64(50)),
						},
					},
				},
			},
		},
		inProto:  &aftpb.Afts_NextHop{},
		inPrefix: mustPath("afts/next-hops/next-hop"),
		wantProto: &aftpb.Afts_NextHop{
			PushedMplsLabelStack: []*aftpb.Afts_NextHop_PushedMplsLabelStackUnion{{
				PushedMplsLabelStackUint64: 20,
			}, {
				PushedMplsLabelStackUint64: 30,
			}, {
				PushedMplsLabelStackUint64: 40,
			}, {
				PushedMplsLabelStackUint64: 50,
			}},
		},
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if err := protomap.ProtoFromPaths(tt.inProto, tt.inPaths,
				protomap.ProtobufMessagePrefix(tt.inPrefix),
				protomap.ValuePathPrefix(tt.inPrefix),
			); err != nil {
				if !tt.wantErr {
					t.Fatalf("cannot unmarshal paths, err: %v, wantErr? %v", err, tt.wantErr)
				}
				return
			}

			if diff := cmp.Diff(tt.inProto, tt.wantProto,
				protocmp.Transform(),
				protocmp.SortRepeatedFields(&aftpb.Afts_NextHopGroup{}, "next_hop"),
			); diff != "" {
				t.Fatalf("did not get expected protobuf, diff(-got,+want):\n%s", diff)
			}
		})
	}
}
