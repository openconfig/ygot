package gnmidiff

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	gpb "github.com/openconfig/gnmi/proto/gnmi"
)

func TestProtoLeafToJSON(t *testing.T) {
	tests := []struct {
		desc         string
		inTypedValue *gpb.TypedValue
		wantJSON     interface{}
		wantErr      bool
	}{{
		desc: "int",
		inTypedValue: &gpb.TypedValue{
			Value: &gpb.TypedValue_IntVal{IntVal: 42},
		},
		wantJSON: float64(42),
	}, {
		desc: "uint",
		inTypedValue: &gpb.TypedValue{
			Value: &gpb.TypedValue_UintVal{UintVal: 42},
		},
		wantJSON: float64(42),
	}, {
		desc: "string",
		inTypedValue: &gpb.TypedValue{
			Value: &gpb.TypedValue_StringVal{StringVal: "42"},
		},
		wantJSON: "42",
	}, {
		desc: "double",
		inTypedValue: &gpb.TypedValue{
			Value: &gpb.TypedValue_DoubleVal{DoubleVal: 42.42},
		},
		wantJSON: "42.42",
	}, {
		desc: "bool",
		inTypedValue: &gpb.TypedValue{
			Value: &gpb.TypedValue_BoolVal{BoolVal: true},
		},
		wantJSON: true,
	}, {
		desc: "leaf-list",
		inTypedValue: &gpb.TypedValue{
			Value: &gpb.TypedValue_LeaflistVal{LeaflistVal: &gpb.ScalarArray{Element: []*gpb.TypedValue{
				{
					Value: &gpb.TypedValue_BoolVal{BoolVal: true},
				},
			}}},
		},
		wantJSON: []interface{}{true},
	}, {
		desc: "bool",
		inTypedValue: &gpb.TypedValue{
			Value: &gpb.TypedValue_BytesVal{BytesVal: []byte("aabbcc")},
		},
		wantJSON: binaryBase64([]byte("aabbcc")),
	}, {
		desc: "JSON_IETF",
		inTypedValue: &gpb.TypedValue{
			Value: &gpb.TypedValue_JsonIetfVal{JsonIetfVal: []byte("aabbcc")},
		},
		wantErr: true,
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got, err := protoLeafToJSON(tt.inTypedValue)
			if gotErr := (err != nil); gotErr != tt.wantErr {
				t.Fatalf("gotErr: %v, wantErr: %v", gotErr, tt.wantErr)
			}
			if diff := cmp.Diff(tt.wantJSON, got); diff != "" {
				t.Errorf("(-want, +got):\n%s", diff)
			}
		})
	}
}
