// Copyright 2018 Google Inc.
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

package yangplugin

import (
	"bytes"
	"compress/gzip"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/generator"
	"github.com/kylelemons/godebug/pretty"

	dpb "github.com/golang/protobuf/protoc-gen-go/descriptor"
	pb "github.com/openconfig/ygot/protoyangplugin/pkg/testproto"
)

const TestRoot string = ""

// toDescriptorProto takes a pb.AMessage message and extracts its file descriptor
// protobuf message. Since the way that this is accessed is via the gzipped
// FileDescriptor in the generated code, we must un-gzip it and unmarshal the
// binary-marshalled proto that is returned.
func toDescriptorProto(msg *pb.AMessage) (*dpb.FileDescriptorProto, error) {
	gzB, _ := msg.Descriptor()
	var r *gzip.Reader
	var err error
	if r, err = gzip.NewReader(bytes.NewReader(gzB)); err != nil {
		return nil, err
	}

	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	desc := &dpb.FileDescriptorProto{}
	if err := proto.Unmarshal(b, desc); err != nil {
		return nil, err
	}
	return desc, nil
}

// descriptorProtoField extracts the specified field from the descriptor proto
// of msg. The order of the fields in the message is as per their definition within
// the original proto.
func descriptorProtoField(msg *pb.AMessage, field int) (*dpb.FieldDescriptorProto, error) {
	desc, err := toDescriptorProto(msg)
	if err != nil {
		return nil, err
	}
	return desc.MessageType[0].Field[field], nil
}

func TestGetSchemaPathAnnotation(t *testing.T) {
	tests := []struct {
		name     string
		in       int // Field number that should be used - 0 has extension set, 1 does not.
		wantPath string
		wantSet  bool
	}{{
		name:     "annotation set on field",
		in:       0,
		wantPath: "/b",
		wantSet:  true,
	}, {
		name:     "annotation not set on field",
		in:       1,
		wantPath: "",
		wantSet:  false,
	}}

	for _, tt := range tests {
		desc, err := descriptorProtoField(&pb.AMessage{}, tt.in)
		if err != nil {
			t.Errorf("%s: toDescriptorProto(%s): got unexpected error extracting descriptor, got: %v, want: nil", tt.name, proto.CompactTextString(desc), err)
		}
		if gotPath, gotSet := getSchemaPathAnnotation(desc); gotPath != tt.wantPath || gotSet != tt.wantSet {
			t.Errorf("%s: getSchemaPathAnnotation(%v), did not get expected return status, got: (path: %s, set: %v), want: (path: %s, set: %v)", tt.name, proto.CompactTextString(desc), gotPath, gotSet, tt.wantPath, tt.wantSet)
		}
	}
}

func TestBuildMap(t *testing.T) {
	g := generator.New()

	data, err := ioutil.ReadFile(filepath.Join(TestRoot, "testdata", "request.textproto"))
	if err != nil {
		t.Fatalf("could not read testdata/request.textproto, got err: %v", err)
	}

	if err := proto.UnmarshalText(string(data), g.Request); err != nil {
		t.Fatalf("cannot unmarshal input proto, got err: %v", err)
	}

	// Boilerplate requirements from main.go to setup the generator correctly.
	// We do this so that all of the internals of the implementation are setup
	// for the object type, and type name calls. This is preferred to implementing
	// a mock of generator.Generator since this test also determines when the
	// generator type in protoc-gen-go is changed.
	g.CommandLineParameters(g.Request.GetParameter())
	g.WrapTypes()
	g.SetPackageNames()
	g.BuildTypeNameMap()
	g.GenerateAllFiles()

	want := &yangProtoMap{
		MessagePathToGoType: map[string]string{
			"/b":     "BMessage",
			"/b/c":   "AMessage_CMessage",
			"/b/c/d": "AMessage_CMessage_DMessage",
		},
		MessageYANGFieldToProtoField: map[string]map[string]string{
			"AMessage": {
				"/b":   "Set",
				"/b/c": "Nested",
			},
			"AMessage_CMessage": {
				"/b/c/d": "Nested",
			},
			"AMessage_CMessage_DMessage": {
				"/b/c/d/field": "Field",
			},
			"BMessage": {},
		},
	}

	// Retrieve the output of the calculated map from the global variable. We use
	// this approach because Generate does not return the map, and the wrapped
	// file descriptors inside the generator code are private.
	got := generatedProtoMap
	if diff := pretty.Compare(got, want); diff != "" {
		t.Errorf("test.proto: did not get expected map output, diff(-got,+want):\n%s", diff)
	}
}
