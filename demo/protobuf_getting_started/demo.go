package main

import (
	"fmt"

	log "github.com/golang/glog"
	"github.com/golang/protobuf/proto"

	ocpb "github.com/openconfig/ygot/demo/protobuf_getting_started/ribproto/openconfig"
	ocenums "github.com/openconfig/ygot/demo/protobuf_getting_started/ribproto/openconfig/enums"
	ocrpb "github.com/openconfig/ygot/demo/protobuf_getting_started/ribproto/openconfig/openconfig_rib_bgp/bgp_rib"
	ocrapb "github.com/openconfig/ygot/demo/protobuf_getting_started/ribproto/openconfig/openconfig_rib_bgp/bgp_rib/afi_safis"
	ocraapb "github.com/openconfig/ygot/demo/protobuf_getting_started/ribproto/openconfig/openconfig_rib_bgp/bgp_rib/afi_safis/afi_safi"
	ocraas4pb "github.com/openconfig/ygot/demo/protobuf_getting_started/ribproto/openconfig/openconfig_rib_bgp/bgp_rib/afi_safis/afi_safi/ipv4_unicast"
	ocraas4lpb "github.com/openconfig/ygot/demo/protobuf_getting_started/ribproto/openconfig/openconfig_rib_bgp/bgp_rib/afi_safis/afi_safi/ipv4_unicast/loc_rib"
	ocraas4lrpb "github.com/openconfig/ygot/demo/protobuf_getting_started/ribproto/openconfig/openconfig_rib_bgp/bgp_rib/afi_safis/afi_safi/ipv4_unicast/loc_rib/routes"
	ocraas4lrrpb "github.com/openconfig/ygot/demo/protobuf_getting_started/ribproto/openconfig/openconfig_rib_bgp/bgp_rib/afi_safis/afi_safi/ipv4_unicast/loc_rib/routes/route"
	ocraas4lrrspb "github.com/openconfig/ygot/demo/protobuf_getting_started/ribproto/openconfig/openconfig_rib_bgp/bgp_rib/afi_safis/afi_safi/ipv4_unicast/loc_rib/routes/route/state"
	ocraaspb "github.com/openconfig/ygot/demo/protobuf_getting_started/ribproto/openconfig/openconfig_rib_bgp/bgp_rib/afi_safis/afi_safi/state"
	ocratpb "github.com/openconfig/ygot/demo/protobuf_getting_started/ribproto/openconfig/openconfig_rib_bgp/bgp_rib/attr_sets"
	ocratapb "github.com/openconfig/ygot/demo/protobuf_getting_started/ribproto/openconfig/openconfig_rib_bgp/bgp_rib/attr_sets/attr_set"
	ocrataaggpb "github.com/openconfig/ygot/demo/protobuf_getting_started/ribproto/openconfig/openconfig_rib_bgp/bgp_rib/attr_sets/attr_set/aggregator"
	ocrataaggspb "github.com/openconfig/ygot/demo/protobuf_getting_started/ribproto/openconfig/openconfig_rib_bgp/bgp_rib/attr_sets/attr_set/aggregator/state"
	ocrataasppb "github.com/openconfig/ygot/demo/protobuf_getting_started/ribproto/openconfig/openconfig_rib_bgp/bgp_rib/attr_sets/attr_set/as_path"
	ocrataaspspb "github.com/openconfig/ygot/demo/protobuf_getting_started/ribproto/openconfig/openconfig_rib_bgp/bgp_rib/attr_sets/attr_set/as_path/segment"
	ocrataaspsspb "github.com/openconfig/ygot/demo/protobuf_getting_started/ribproto/openconfig/openconfig_rib_bgp/bgp_rib/attr_sets/attr_set/as_path/segment/state"
	ocrataspb "github.com/openconfig/ygot/demo/protobuf_getting_started/ribproto/openconfig/openconfig_rib_bgp/bgp_rib/attr_sets/attr_set/state"
	ywpb "github.com/openconfig/ygot/proto/ywrapper"
)

func main() {
	routeEntry := &ocpb.Device{
		BgpRib: &ocrpb.BgpRib{
			AttrSets: &ocratpb.AttrSets{
				AttrSet: []*ocratpb.AttrSetKey{{
					Index: 1,
					AttrSet: &ocratapb.AttrSet{
						State: &ocrataspb.State{
							AtomicAggregate: &ywpb.BoolValue{true},
							Index:           &ywpb.UintValue{1},
							LocalPref:       &ywpb.UintValue{100},
							Med:             &ywpb.UintValue{10},
							NextHop:         &ywpb.StringValue{"10.0.0.1"},
							Origin:          ocenums.OpenconfigRibBgpBgpOriginAttrType_OPENCONFIGRIBBGPBGPORIGINATTRTYPE_IGP,
							OriginatorId:    &ywpb.StringValue{"192.0.2.42"},
						},
						Aggregator: &ocrataaggpb.Aggregator{
							State: &ocrataaggspb.State{
								Address: &ywpb.StringValue{"10.1.42.1"},
								As:      &ywpb.UintValue{15169},
							},
						},
						AsPath: &ocrataasppb.AsPath{
							Segment: []*ocrataaspspb.Segment{{
								State: &ocrataaspsspb.State{
									Member: []*ywpb.UintValue{{15169}, {6643}, {5400}, {2856}, {4445}, {1273}, {5413}, {29636}},
									Type:   ocenums.OpenconfigRibBgpAsPathSegmentType_OPENCONFIGRIBBGPASPATHSEGMENTTYPE_AS_SET,
								},
							}},
						},
					},
				}},
			},
			AfiSafis: &ocrapb.AfiSafis{
				AfiSafi: []*ocrapb.AfiSafiKey{{
					AfiSafiName: ocenums.OpenconfigBgpTypesAFISAFITYPE_OPENCONFIGBGPTYPESAFISAFITYPE_IPV4_UNICAST,
					AfiSafi: &ocraapb.AfiSafi{
						State: &ocraaspb.State{
							AfiSafiName: ocenums.OpenconfigBgpTypesAFISAFITYPE_OPENCONFIGBGPTYPESAFISAFITYPE_IPV4_UNICAST,
						},
						Ipv4Unicast: &ocraas4pb.Ipv4Unicast{
							LocRib: &ocraas4lpb.LocRib{
								Routes: &ocraas4lrpb.Routes{
									Route: []*ocraas4lrpb.RouteKey{{
										Prefix: "192.0.2.0/24",
										Origin: &ocraas4lrpb.RouteKey_OriginOpenconfigpolicytypesinstallprotocoltype{ocenums.OpenconfigPolicyTypesINSTALLPROTOCOLTYPE_OPENCONFIGPOLICYTYPESINSTALLPROTOCOLTYPE_BGP},
										PathId: 1,
										Route: &ocraas4lrrpb.Route{
											State: &ocraas4lrrspb.State{
												PathId:    &ywpb.UintValue{1},
												Prefix:    &ywpb.StringValue{"192.0.2.0/24"},
												Origin:    &ocraas4lrrspb.State_OriginOpenconfigpolicytypesinstallprotocoltype{ocenums.OpenconfigPolicyTypesINSTALLPROTOCOLTYPE_OPENCONFIGPOLICYTYPESINSTALLPROTOCOLTYPE_BGP},
												AttrIndex: &ywpb.UintValue{1},
											},
										},
									}},
								},
							},
						},
					},
				}},
			},
		},
	}

	b, err := proto.Marshal(routeEntry)
	if err != nil {
		log.Exitf("Error when marshalling proto: %v", err)
	}

	fmt.Printf("%s\n", proto.MarshalTextString(routeEntry))
	fmt.Printf("size in bytes: %d", len(b))
}
