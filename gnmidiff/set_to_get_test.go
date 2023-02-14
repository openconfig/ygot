package gnmidiff

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	gpb "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/ygot/exampleoc"
	"github.com/openconfig/ygot/ygot"
	"github.com/openconfig/ygot/ytypes"
)

func TestSetRequestToNotifications(t *testing.T) {
	// TODO: test that deletes in notifs.
	tests := []struct {
		desc                string
		dontCheckWithSchema bool
		inSetRequest        *gpb.SetRequest
		inNotifications     []*gpb.Notification
		wantSetToNotifsDiff SetToNotifsDiff
		wantErr             bool
	}{{
		desc: "exactly the same",
		inSetRequest: &gpb.SetRequest{
			Replace: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Name: ygot.String("eth0")}),
			}},
			Update: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Subinterface: map[uint32]*exampleoc.Interface_Subinterface{0: {Index: ygot.Uint32(0), OperStatus: exampleoc.Interface_OperStatus_TESTING}}, Description: ygot.String("I am an eth port")}),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/config/enabled"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_BoolVal{BoolVal: true}},
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/state/transceiver"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "FDM"}},
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/state/physical-channel"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_LeaflistVal{LeaflistVal: &gpb.ScalarArray{Element: []*gpb.TypedValue{{Value: &gpb.TypedValue_UintVal{UintVal: 42}}, {Value: &gpb.TypedValue_UintVal{UintVal: 84}}}}}},
			}},
		},
		inNotifications: []*gpb.Notification{{
			Update: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Name: ygot.String("eth0")}),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Subinterface: map[uint32]*exampleoc.Interface_Subinterface{0: {Index: ygot.Uint32(0), OperStatus: exampleoc.Interface_OperStatus_TESTING}}, Description: ygot.String("I am an eth port")}),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/config/enabled"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_BoolVal{BoolVal: true}},
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/state/transceiver"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "FDM"}},
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/state/physical-channel"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_LeaflistVal{LeaflistVal: &gpb.ScalarArray{Element: []*gpb.TypedValue{{Value: &gpb.TypedValue_UintVal{UintVal: 42}}, {Value: &gpb.TypedValue_UintVal{UintVal: 84}}}}}},
			}},
		}},
		wantSetToNotifsDiff: SetToNotifsDiff{
			MissingUpdates: map[string]interface{}{},
			ExtraUpdates:   map[string]interface{}{},
			CommonUpdates: map[string]interface{}{
				"/interfaces/interface[name=eth0]/name":                                                  "eth0",
				"/interfaces/interface[name=eth0]/config/name":                                           "eth0",
				"/interfaces/interface[name=eth0]/config/description":                                    "I am an eth port",
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/config/index":      float64(0),
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/index":             float64(0),
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/state/oper-status": "TESTING",
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/config/enabled":    true,
				"/interfaces/interface[name=eth0]/state/transceiver":                                     "FDM",
				"/interfaces/interface[name=eth0]/state/physical-channel":                                []interface{}{float64(42), float64(84)},
			},
			MismatchedUpdates: map[string]MismatchedUpdate{},
		},
	}, {
		desc: "exactly the same with prefix usage",
		inSetRequest: &gpb.SetRequest{
			Prefix: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
			Replace: []*gpb.Update{{
				Path: ygot.MustStringToPath(""),
				Val:  must7951(&exampleoc.Interface{Name: ygot.String("eth0")}),
			}},
			Update: []*gpb.Update{{
				Path: ygot.MustStringToPath(""),
				Val:  must7951(&exampleoc.Interface{Subinterface: map[uint32]*exampleoc.Interface_Subinterface{0: {Index: ygot.Uint32(0), OperStatus: exampleoc.Interface_OperStatus_TESTING}}, Description: ygot.String("I am an eth port")}),
			}, {
				Path: ygot.MustStringToPath("subinterfaces/subinterface[index=0]/config/enabled"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_BoolVal{BoolVal: true}},
			}, {
				Path: ygot.MustStringToPath("state/transceiver"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "FDM"}},
			}, {
				Path: ygot.MustStringToPath("state/physical-channel"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_LeaflistVal{LeaflistVal: &gpb.ScalarArray{Element: []*gpb.TypedValue{{Value: &gpb.TypedValue_UintVal{UintVal: 42}}, {Value: &gpb.TypedValue_UintVal{UintVal: 84}}}}}},
			}},
		},
		inNotifications: []*gpb.Notification{{
			Update: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Name: ygot.String("eth0")}),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Subinterface: map[uint32]*exampleoc.Interface_Subinterface{0: {Index: ygot.Uint32(0), OperStatus: exampleoc.Interface_OperStatus_TESTING}}, Description: ygot.String("I am an eth port")}),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/config/enabled"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_BoolVal{BoolVal: true}},
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/state/transceiver"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "FDM"}},
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/state/physical-channel"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_LeaflistVal{LeaflistVal: &gpb.ScalarArray{Element: []*gpb.TypedValue{{Value: &gpb.TypedValue_UintVal{UintVal: 42}}, {Value: &gpb.TypedValue_UintVal{UintVal: 84}}}}}},
			}},
		}},
		wantSetToNotifsDiff: SetToNotifsDiff{
			MissingUpdates: map[string]interface{}{},
			ExtraUpdates:   map[string]interface{}{},
			CommonUpdates: map[string]interface{}{
				"/interfaces/interface[name=eth0]/name":                                                  "eth0",
				"/interfaces/interface[name=eth0]/config/name":                                           "eth0",
				"/interfaces/interface[name=eth0]/config/description":                                    "I am an eth port",
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/config/index":      float64(0),
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/index":             float64(0),
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/state/oper-status": "TESTING",
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/config/enabled":    true,
				"/interfaces/interface[name=eth0]/state/transceiver":                                     "FDM",
				"/interfaces/interface[name=eth0]/state/physical-channel":                                []interface{}{float64(42), float64(84)},
			},
			MismatchedUpdates: map[string]MismatchedUpdate{},
		},
	}, {
		desc: "exactly the same but with some dont care paths in notifications",
		inSetRequest: &gpb.SetRequest{
			Replace: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Name: ygot.String("eth0")}),
			}},
			Update: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Subinterface: map[uint32]*exampleoc.Interface_Subinterface{0: {Index: ygot.Uint32(0), OperStatus: exampleoc.Interface_OperStatus_TESTING}}, Description: ygot.String("I am an eth port")}),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/config/enabled"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_BoolVal{BoolVal: true}},
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/state/transceiver"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "FDM"}},
			}},
		},
		inNotifications: []*gpb.Notification{{
			Update: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Name: ygot.String("eth0")}),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Subinterface: map[uint32]*exampleoc.Interface_Subinterface{0: {Index: ygot.Uint32(0), OperStatus: exampleoc.Interface_OperStatus_TESTING}}, Description: ygot.String("I am an eth port")}),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/config/enabled"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_BoolVal{BoolVal: true}},
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/state/transceiver"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "FDM"}},
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth2]"),
				Val:  must7951(&exampleoc.Interface{Name: ygot.String("eth2")}),
			}},
		}},
		wantSetToNotifsDiff: SetToNotifsDiff{
			MissingUpdates: map[string]interface{}{},
			ExtraUpdates:   map[string]interface{}{},
			CommonUpdates: map[string]interface{}{
				"/interfaces/interface[name=eth0]/name":                                                  "eth0",
				"/interfaces/interface[name=eth0]/config/name":                                           "eth0",
				"/interfaces/interface[name=eth0]/config/description":                                    "I am an eth port",
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/config/index":      float64(0),
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/index":             float64(0),
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/state/oper-status": "TESTING",
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/config/enabled":    true,
				"/interfaces/interface[name=eth0]/state/transceiver":                                     "FDM",
			},
			MismatchedUpdates: map[string]MismatchedUpdate{},
		},
	}, {
		desc: "Notification has some overwriting",
		inSetRequest: &gpb.SetRequest{
			Replace: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Name: ygot.String("eth0")}),
			}},
			Update: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Subinterface: map[uint32]*exampleoc.Interface_Subinterface{0: {Index: ygot.Uint32(0), OperStatus: exampleoc.Interface_OperStatus_TESTING}}, Description: ygot.String("I am an eth port")}),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/config/enabled"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_BoolVal{BoolVal: true}},
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/state/transceiver"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "FDM"}},
			}},
		},
		inNotifications: []*gpb.Notification{{
			Update: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Name: ygot.String("eth0"), Transceiver: ygot.String("TDM")}),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Subinterface: map[uint32]*exampleoc.Interface_Subinterface{0: {Index: ygot.Uint32(0), OperStatus: exampleoc.Interface_OperStatus_TESTING, Enabled: ygot.Bool(true)}}}),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Description: ygot.String("I am an eth port")}),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/state/transceiver"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "FDM"}},
			}},
		}},
		wantSetToNotifsDiff: SetToNotifsDiff{
			MissingUpdates: map[string]interface{}{},
			ExtraUpdates:   map[string]interface{}{},
			CommonUpdates: map[string]interface{}{
				"/interfaces/interface[name=eth0]/name":                                                  "eth0",
				"/interfaces/interface[name=eth0]/config/name":                                           "eth0",
				"/interfaces/interface[name=eth0]/config/description":                                    "I am an eth port",
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/config/index":      float64(0),
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/index":             float64(0),
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/state/oper-status": "TESTING",
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/config/enabled":    true,
				"/interfaces/interface[name=eth0]/state/transceiver":                                     "FDM",
			},
			MismatchedUpdates: map[string]MismatchedUpdate{},
		},
	}, {
		desc: "delete in SetRequest",
		inSetRequest: &gpb.SetRequest{
			Delete: []*gpb.Path{
				ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
			},
		},
		inNotifications: []*gpb.Notification{{
			Update: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Name: ygot.String("eth0")}),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Subinterface: map[uint32]*exampleoc.Interface_Subinterface{0: {Index: ygot.Uint32(0), OperStatus: exampleoc.Interface_OperStatus_TESTING}}, Description: ygot.String("I am an eth port")}),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/config/enabled"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_BoolVal{BoolVal: true}},
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/state/transceiver"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "FDM"}},
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/state/physical-channel"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_LeaflistVal{LeaflistVal: &gpb.ScalarArray{Element: []*gpb.TypedValue{{Value: &gpb.TypedValue_UintVal{UintVal: 42}}, {Value: &gpb.TypedValue_UintVal{UintVal: 84}}}}}},
			}},
		}},
		wantSetToNotifsDiff: SetToNotifsDiff{
			MissingUpdates: map[string]interface{}{},
			ExtraUpdates: map[string]interface{}{
				"/interfaces/interface[name=eth0]/name":                                                  "eth0",
				"/interfaces/interface[name=eth0]/config/name":                                           "eth0",
				"/interfaces/interface[name=eth0]/config/description":                                    "I am an eth port",
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/config/index":      float64(0),
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/index":             float64(0),
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/state/oper-status": "TESTING",
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/config/enabled":    true,
				"/interfaces/interface[name=eth0]/state/transceiver":                                     "FDM",
				"/interfaces/interface[name=eth0]/state/physical-channel":                                []interface{}{float64(42), float64(84)},
			},
			CommonUpdates:     map[string]interface{}{},
			MismatchedUpdates: map[string]MismatchedUpdate{},
		},
	}, {
		desc: "SetRequest has conflicts",
		inSetRequest: &gpb.SetRequest{
			Replace: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Name: ygot.String("eth0"), Transceiver: ygot.String("FDM")}),
			}},
			Update: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Subinterface: map[uint32]*exampleoc.Interface_Subinterface{0: {Index: ygot.Uint32(0), OperStatus: exampleoc.Interface_OperStatus_TESTING, Enabled: ygot.Bool(true)}}}),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Description: ygot.String("I am an eth port")}),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/config/enabled"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_BoolVal{BoolVal: true}},
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/state/transceiver"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "TDM"}},
			}},
		},
		inNotifications: []*gpb.Notification{{
			Update: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Name: ygot.String("eth0")}),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Subinterface: map[uint32]*exampleoc.Interface_Subinterface{0: {Index: ygot.Uint32(0), OperStatus: exampleoc.Interface_OperStatus_TESTING}}, Description: ygot.String("I am an eth port")}),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/config/enabled"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_BoolVal{BoolVal: true}},
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/state/transceiver"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "FDM"}},
			}},
		}},
		wantErr: true,
	}, {
		desc: "No notifications",
		inSetRequest: &gpb.SetRequest{
			Replace: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Name: ygot.String("eth0")}),
			}},
			Update: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Subinterface: map[uint32]*exampleoc.Interface_Subinterface{0: {Index: ygot.Uint32(0), OperStatus: exampleoc.Interface_OperStatus_TESTING}}, Description: ygot.String("I am an eth port")}),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/config/enabled"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_BoolVal{BoolVal: true}},
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/state/transceiver"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "FDM"}},
			}},
		},
		inNotifications: nil,
		wantSetToNotifsDiff: SetToNotifsDiff{
			MissingUpdates: map[string]interface{}{
				"/interfaces/interface[name=eth0]/name":                                                  "eth0",
				"/interfaces/interface[name=eth0]/config/name":                                           "eth0",
				"/interfaces/interface[name=eth0]/config/description":                                    "I am an eth port",
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/config/index":      float64(0),
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/index":             float64(0),
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/state/oper-status": "TESTING",
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/config/enabled":    true,
				"/interfaces/interface[name=eth0]/state/transceiver":                                     "FDM",
			},
			ExtraUpdates:      map[string]interface{}{},
			CommonUpdates:     map[string]interface{}{},
			MismatchedUpdates: map[string]MismatchedUpdate{},
		},
	}, {
		desc: "Notification has extra updates",
		inSetRequest: &gpb.SetRequest{
			Replace: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Name: ygot.String("eth0")}),
			}},
		},
		inNotifications: []*gpb.Notification{{
			Update: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Name: ygot.String("eth0"), Transceiver: ygot.String("FDM")}),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Subinterface: map[uint32]*exampleoc.Interface_Subinterface{0: {Index: ygot.Uint32(0), OperStatus: exampleoc.Interface_OperStatus_TESTING, Enabled: ygot.Bool(true)}}}),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Description: ygot.String("I am an eth port")}),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/state/transceiver"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "FDM"}},
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/config/mtu"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_UintVal{UintVal: 1500}},
			}},
		}},
		wantSetToNotifsDiff: SetToNotifsDiff{
			MissingUpdates: map[string]interface{}{},
			ExtraUpdates: map[string]interface{}{
				"/interfaces/interface[name=eth0]/config/description":                                    "I am an eth port",
				"/interfaces/interface[name=eth0]/config/mtu":                                            float64(1500),
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/config/index":      float64(0),
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/index":             float64(0),
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/state/oper-status": "TESTING",
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/config/enabled":    true,
				"/interfaces/interface[name=eth0]/state/transceiver":                                     "FDM",
			},
			CommonUpdates: map[string]interface{}{
				"/interfaces/interface[name=eth0]/name":        "eth0",
				"/interfaces/interface[name=eth0]/config/name": "eth0",
			},
			MismatchedUpdates: map[string]MismatchedUpdate{},
		},
	}, {
		desc:         "Nil SetRequest: entire notification is dont care",
		inSetRequest: nil,
		inNotifications: []*gpb.Notification{{
			Update: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Name: ygot.String("eth0"), Transceiver: ygot.String("FDM")}),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Subinterface: map[uint32]*exampleoc.Interface_Subinterface{0: {Index: ygot.Uint32(0), OperStatus: exampleoc.Interface_OperStatus_TESTING, Enabled: ygot.Bool(true)}}}),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Description: ygot.String("I am an eth port")}),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/state/transceiver"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "FDM"}},
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/config/mtu"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_UintVal{UintVal: 1500}},
			}},
		}},
		wantSetToNotifsDiff: SetToNotifsDiff{
			MissingUpdates:    map[string]interface{}{},
			ExtraUpdates:      map[string]interface{}{},
			CommonUpdates:     map[string]interface{}{},
			MismatchedUpdates: map[string]MismatchedUpdate{},
		},
	}, {
		desc: "mismatching values",
		inSetRequest: &gpb.SetRequest{
			Replace: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Name: ygot.String("eth0")}),
			}},
			Update: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Subinterface: map[uint32]*exampleoc.Interface_Subinterface{0: {Index: ygot.Uint32(0), OperStatus: exampleoc.Interface_OperStatus_DORMANT}}, Description: ygot.String("I am an ethernet port")}),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/config/enabled"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_BoolVal{BoolVal: false}},
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/state/transceiver"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "TDM"}},
			}},
		},
		inNotifications: []*gpb.Notification{{
			Update: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Name: ygot.String("eth0"), Transceiver: ygot.String("FDM")}),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Subinterface: map[uint32]*exampleoc.Interface_Subinterface{0: {Index: ygot.Uint32(0), OperStatus: exampleoc.Interface_OperStatus_TESTING, Enabled: ygot.Bool(true)}}}),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Description: ygot.String("I am an eth port")}),
			}},
		}},
		wantSetToNotifsDiff: SetToNotifsDiff{
			MissingUpdates: map[string]interface{}{},
			ExtraUpdates:   map[string]interface{}{},
			CommonUpdates: map[string]interface{}{
				"/interfaces/interface[name=eth0]/name":                                             "eth0",
				"/interfaces/interface[name=eth0]/config/name":                                      "eth0",
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/config/index": float64(0),
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/index":        float64(0),
			},
			MismatchedUpdates: map[string]MismatchedUpdate{
				"/interfaces/interface[name=eth0]/config/description": {
					A: "I am an ethernet port",
					B: "I am an eth port",
				},
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/state/oper-status": {
					A: "DORMANT",
					B: "TESTING",
				},
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/config/enabled": {
					A: false,
					B: true,
				},
				"/interfaces/interface[name=eth0]/state/transceiver": {
					A: "TDM",
					B: "FDM",
				},
			},
		},
	}, {
		desc: "not the same with every difference case -- dont care, replace, replace subtree extras, update subtree dont care",
		inSetRequest: &gpb.SetRequest{
			Replace: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth1]"),
				Val:  must7951(&exampleoc.Interface{Name: ygot.String("eth1")}),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Name: ygot.String("eth0")}),
			}},
			Update: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Subinterface: map[uint32]*exampleoc.Interface_Subinterface{0: {Index: ygot.Uint32(0), OperStatus: exampleoc.Interface_OperStatus_TESTING}}, Description: ygot.String("I am an eth port")}),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth2]"),
				Val:  must7951(&exampleoc.Interface{Subinterface: map[uint32]*exampleoc.Interface_Subinterface{0: {Index: ygot.Uint32(0), OperStatus: exampleoc.Interface_OperStatus_LOWER_LAYER_DOWN}}, Description: ygot.String("I am an eth port")}),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/config/enabled"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_BoolVal{BoolVal: true}},
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth2]/state/transceiver"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_StringVal{StringVal: "FDM"}},
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/state/logical"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_BoolVal{BoolVal: false}},
			}},
		},
		inNotifications: []*gpb.Notification{{
			Update: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Name: ygot.String("eth0"), Transceiver: ygot.String("FDM")}),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth2]"),
				Val:  must7951(&exampleoc.Interface{Name: ygot.String("eth2"), Transceiver: ygot.String("FDM"), Mtu: ygot.Uint16(1500)}),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Subinterface: map[uint32]*exampleoc.Interface_Subinterface{0: {Index: ygot.Uint32(0), OperStatus: exampleoc.Interface_OperStatus_TESTING, Enabled: ygot.Bool(true)}}}),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&exampleoc.Interface{Description: ygot.String("I am an eth port")}),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/state/logical"),
				Val:  &gpb.TypedValue{Value: &gpb.TypedValue_BoolVal{BoolVal: true}},
			}},
		}},
		wantSetToNotifsDiff: SetToNotifsDiff{
			MissingUpdates: map[string]interface{}{
				"/interfaces/interface[name=eth1]/name":                                                  "eth1",
				"/interfaces/interface[name=eth1]/config/name":                                           "eth1",
				"/interfaces/interface[name=eth2]/config/description":                                    "I am an eth port",
				"/interfaces/interface[name=eth2]/subinterfaces/subinterface[index=0]/config/index":      float64(0),
				"/interfaces/interface[name=eth2]/subinterfaces/subinterface[index=0]/index":             float64(0),
				"/interfaces/interface[name=eth2]/subinterfaces/subinterface[index=0]/state/oper-status": "LOWER_LAYER_DOWN",
			},
			ExtraUpdates: map[string]interface{}{
				"/interfaces/interface[name=eth0]/state/transceiver": "FDM",
			},
			CommonUpdates: map[string]interface{}{
				"/interfaces/interface[name=eth0]/name":                                                  "eth0",
				"/interfaces/interface[name=eth0]/config/name":                                           "eth0",
				"/interfaces/interface[name=eth0]/config/description":                                    "I am an eth port",
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/config/index":      float64(0),
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/index":             float64(0),
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/state/oper-status": "TESTING",
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/config/enabled":    true,
				"/interfaces/interface[name=eth2]/state/transceiver":                                     "FDM",
			},
			MismatchedUpdates: map[string]MismatchedUpdate{
				"/interfaces/interface[name=eth0]/subinterfaces/subinterface[index=0]/state/logical": {
					A: false,
					B: true,
				},
			},
		},
	}, {
		desc:                "not the same with every difference case using prefix-matching name",
		dontCheckWithSchema: true,
		inSetRequest: &gpb.SetRequest{
			Delete: []*gpb.Path{
				ygot.MustStringToPath("/interfaces/interface[name=eth4]"),
			},
			Replace: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth1]"),
				Val:  must7951(&Interface{Name: ygot.String("eth1")}), // missing
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&Interface{Name: ygot.String("eth0")}),
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth3]/config/n"), // Make sure Mallory is still dontcare.
				Val:  must7951(&Interface{}),
			}},
			Update: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&Interface{Namer: ygot.String("Alice")}), // missing
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth2]"),
				Val:  must7951(&Interface{Namer: ygot.String("Carol")}), // missing
			}},
		},
		inNotifications: []*gpb.Notification{{
			Update: []*gpb.Update{{
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth0]"),
				Val:  must7951(&Interface{Name: ygot.String("eth0")}), // common
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth1]"),
				Val:  must7951(&Interface{Namer: ygot.String("Bob")}), // extra
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth1]"),
				Val:  must7951(&Interface{Name: ygot.String("eth0")}), // mismatched
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth3]"),
				Val:  must7951(&Interface{Namer: ygot.String("Mallory")}), // dontcare
			}, {
				Path: ygot.MustStringToPath("/interfaces/interface[name=eth4]"),
				Val:  must7951(&Interface{Namer: ygot.String("Charlie")}), // extra
			}},
		}},
		wantSetToNotifsDiff: SetToNotifsDiff{
			MissingUpdates: map[string]interface{}{
				"/interfaces/interface[name=eth0]/config/namer": "Alice",
				"/interfaces/interface[name=eth2]/config/namer": "Carol",
			},
			ExtraUpdates: map[string]interface{}{
				"/interfaces/interface[name=eth1]/config/namer": "Bob",
				"/interfaces/interface[name=eth4]/config/namer": "Charlie",
			},
			CommonUpdates: map[string]interface{}{
				"/interfaces/interface[name=eth0]/name":        "eth0",
				"/interfaces/interface[name=eth0]/config/name": "eth0",
			},
			MismatchedUpdates: map[string]MismatchedUpdate{
				"/interfaces/interface[name=eth1]/name": {
					A: "eth1",
					B: "eth0",
				},
				"/interfaces/interface[name=eth1]/config/name": {
					A: "eth1",
					B: "eth0",
				},
			},
		},
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			withNewSchema := []bool{false}
			if !tt.dontCheckWithSchema {
				withNewSchema = append(withNewSchema, true)
			}
			for _, withSchema := range withNewSchema {
				var inSchema *ytypes.Schema
				if withSchema {
					var err error
					if inSchema, err = exampleoc.Schema(); err != nil {
						t.Fatalf("schema has error: %v", err)
					}
				}
				t.Run(fmt.Sprintf("withSchema-%v", withSchema), func(t *testing.T) {
					got, err := DiffSetRequestToNotifications(tt.inSetRequest, tt.inNotifications, inSchema)
					if (err != nil) != tt.wantErr {
						t.Fatalf("got error: %v, want error: %v", err, tt.wantErr)
					}
					if diff := cmp.Diff(tt.wantSetToNotifsDiff, got); diff != "" {
						t.Errorf("DiffSetRequest (-want, +got):\n%s", diff)
					}
				})
			}
		})
	}
}
