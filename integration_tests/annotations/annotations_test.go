// Copyright 2020 Google Inc.
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

package annotations

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	oc "github.com/nokia/ygot/exampleoc"
	"github.com/nokia/ygot/ygot"
	"github.com/openconfig/gnmi/errdiff"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/nokia/ygot/integration_tests/annotations/apb"
	"github.com/nokia/ygot/integration_tests/annotations/proto2apb"
	gpb "github.com/openconfig/gnmi/proto/gnmi"
)

type IncludedProto3 struct {
	a *apb.Annotation
}

func (i *IncludedProto3) MarshalJSON() ([]byte, error) {
	return protojson.Marshal(i.a)
}

func (i *IncludedProto3) UnmarshalJSON(b []byte) error {
	return protojson.Unmarshal(b, i.a)
}

type EmbeddedProto3 struct {
	*apb.Annotation
}

func (e *EmbeddedProto3) MarshalJSON() ([]byte, error) {
	return protojson.Marshal(e)
}

func (e *EmbeddedProto3) UnmarshalJSON(b []byte) error {
	return protojson.Unmarshal(b, e)
}

func NewEmbeddedProto3(s string) *EmbeddedProto3 {
	return &EmbeddedProto3{
		Annotation: &apb.Annotation{Comment: s},
	}
}

type IncludedProto2 struct {
	a *proto2apb.Annotation
}

func (i *IncludedProto2) MarshalJSON() ([]byte, error) {
	return protojson.Marshal(i.a)
}

func (i *IncludedProto2) UnmarshalJSON(b []byte) error {
	return protojson.Unmarshal(b, i.a)
}

type EmbeddedProto2 struct {
	*proto2apb.Annotation
}

func (e *EmbeddedProto2) MarshalJSON() ([]byte, error) {
	return protojson.Marshal(e)
}

func (e *EmbeddedProto2) UnmarshalJSON(b []byte) error {
	return protojson.Unmarshal(b, e)
}

func NewEmbeddedProto2(s string) *EmbeddedProto2 {
	return &EmbeddedProto2{
		Annotation: &proto2apb.Annotation{Comment: proto.String(s)},
	}
}

func TestProtoAnnotation(t *testing.T) {
	tests := []struct {
		desc                      string
		inAnnotation              ygot.Annotation
		wantJSON                  string
		wantMarshalErrSubstring   string
		wantUnmarshalErrSubstring string
	}{{
		desc:         "included proto3 annotation",
		inAnnotation: &IncludedProto3{a: &apb.Annotation{Comment: "hello"}},
		wantJSON: `
		{
			"@": [
			   {
				  "comment": "hello"
			   }
			]
		 }`,
	}, {
		desc:         "embedded proto3 annotation",
		inAnnotation: NewEmbeddedProto3("hello world"),
		wantJSON: `
		{
			"@": [
			   {
				  "comment": "hello world"
			   }
			]
		 }`,
	}, {
		desc:         "included proto2 annotation",
		inAnnotation: &IncludedProto2{a: &proto2apb.Annotation{Comment: proto.String("hello")}},
		wantJSON: `
		{
			"@": [
			   {
				  "comment": "hello"
			   }
			]
		 }`,
	}, {
		desc:         "embedded proto2 annotation",
		inAnnotation: NewEmbeddedProto2("hello world"),
		wantJSON: `
		{
			"@": [
			   {
				  "comment": "hello world"
			   }
			]
		 }
		`,
	}}

	isEmptyDiff := func(d *gpb.Notification) bool {
		return len(d.Update) == 0 && len(d.Delete) == 0
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			d := &oc.Device{}
			d.ΛMetadata = append(d.ΛMetadata, tt.inAnnotation)

			got, err := ygot.EmitJSON(d, &ygot.EmitJSONConfig{
				Format: ygot.RFC7951,
			})
			if diff := errdiff.Substring(err, tt.wantMarshalErrSubstring); diff != "" {
				t.Fatalf("did not get expected marshalling error, %s", diff)
			}

			wantJ := map[string]interface{}{}
			if err := json.Unmarshal([]byte(tt.wantJSON), &wantJ); err != nil {
				t.Fatalf("cannot unmarshal expected JSON, err: %v", err)
			}
			gotJ := map[string]interface{}{}
			if err := json.Unmarshal([]byte(got), &gotJ); err != nil {
				t.Fatalf("cannot unmarshal received JSON, err: %v", err)
			}

			if diff := cmp.Diff(gotJ, wantJ); diff != "" {
				t.Fatalf("did not get expected marshalled JSON, diff(-got,+want):\n%s", diff)
			}

			nd := &oc.Device{}
			err = oc.Unmarshal([]byte(got), nd)
			if diff := errdiff.Substring(err, tt.wantUnmarshalErrSubstring); diff != "" {
				t.Fatalf("did not get expected unmarshalling error, %s", diff)
			}

			diff, err := ygot.Diff(nd, d)
			if err != nil {
				t.Fatalf("error diffing expected and received unmarshalled content, %v", err)
			}
			if !isEmptyDiff(diff) {
				t.Fatalf("did not get expected unmarshalled output, not equal to input, diff: %s", diff)
			}
		})
	}
}
