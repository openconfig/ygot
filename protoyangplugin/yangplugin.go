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

// Package yangplugin is a plugin to protoc-gen-go which adds additional
// output to the generated .pb.go. Particularly, it provides a map between
// YANG schema paths in the yext.schemapath annotation and the corresponding
// Go type, and a map between Go type and the corresponding schemapath
// annotation.
package yangplugin

import (
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/generator"

	dpb "github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/openconfig/ygot/proto/yext"
)

// init registers the generator against the protoc-gen-go framework.
func init() {
	generator.RegisterPlugin(new(yang))
}

var (
	// generatedProtoMap is a global variable used to catch the output of the
	// built map by the plugin. It is used only in testing. This is required
	// because the output Go code is not provided back to the caller, but
	// rather written into the generator. We capture the output of the map
	// such that we can test its correctness.
	generatedProtoMap *yangProtoMap
)

// yang is the type used for the YANG<->Protobuf generator plugin.
type yang struct {
	*generator.Generator
}

// Name provides a string name for the plugin to protoc-gen-go's generator
// package. The name is used on the command-line as a plugin name.
func (y *yang) Name() string {
	return "yang"
}

// Init is called by the protoc-gen-go generator package to initialise the YANG
// plugin. The provided gen is the current Generator instance being used to
// process the protobuf files for which code is being generated.
func (y *yang) Init(gen *generator.Generator) {
	y.Generator = gen
}

// Generate is called by the plugin infrastructure of the protoc-gen-go generator
// implementation for each input file (represented by a wrapped protobuf
// FileDescriptor proto).
func (y *yang) Generate(file *generator.FileDescriptor) {
	// Generate the output and catch it in a global variable so that we can
	// test the output of buildMap even though the list of files in the
	// generator is internal.
	generatedProtoMap = y.buildMap(file)

	// Alias the generatedProtoMap for use in this function.
	ypm := generatedProtoMap

	y.P("var (")
	y.P("	YANGPathToProtoGoStruct = map[string]reflect.Type{")
	for path, msgType := range ypm.MessagePathToGoType {
		y.P(`		"`, path, `": reflect.TypeOf(`, msgType, `{}),`)
	}
	y.P("	}")
	y.P("")
	y.P("	ProtoGoStructPathToFieldName = map[string]map[string]string{")
	for goStructName, fields := range ypm.MessageYANGFieldToProtoField {
		y.P(`		"`, goStructName, `": map[string]string{`)
		for path, structFieldName := range fields {
			y.P(`			"`, path, `": "`, structFieldName, `",`)
		}
		y.P("		},")
	}
	y.P("	}")
	y.P(")")
}

// GenerateImports adds required imports to the output .pb.go file.
func (y *yang) GenerateImports(_ *generator.FileDescriptor) {
	y.P("import (")
	y.P(`	"reflect"`)
	y.P(")")
}

// yangProtoMap stores mappings between YANG schema paths and the corresponding
// protobuf entities.
type yangProtoMap struct {
	// MessagePathToGoType maps a message by YANG path to its corresponding
	// type in the generated .pb.go file.
	MessagePathToGoType map[string]string
	// MessageProtoFieldToYANGField maps a message by Go type name for the
	// generated Protobuf message to its fields. The map of fields is keyed
	// by YANG field name to Go field name.
	MessageYANGFieldToProtoField map[string]map[string]string
}

// newYANGProtoMap initialises a yangProtoMap struct.
func newYANGProtoMap() *yangProtoMap {
	return &yangProtoMap{
		MessagePathToGoType:          map[string]string{},
		MessageYANGFieldToProtoField: map[string]map[string]string{},
	}
}

// buildMap processes the messages within the input file descriptor and returns
// a yangProtoMap describing the mapping between the YANG schema paths within
// the file and the corresponding entities within the .pb.go that is being
// created by the generator (*yang)
func (y *yang) buildMap(file *generator.FileDescriptor) *yangProtoMap {
	ypm := newYANGProtoMap()
	for _, m := range file.MessageType {
		// The base message is called ".<package_name>.<message_name>" - this logic
		// is also implemented in the generator code within private functions.
		y.buildMapInternal(m, strings.Join([]string{"", *file.Package, m.GetName()}, "."), ypm)
	}
	return ypm
}

// buildMapInternal iteratively traverses the input descriptor proto (which describes
// a message), and calculates the schema path mappings within it. The messageType
// provided is appended to for child messages. The yangProtoMap provided is appended
// to as new mappings are extracted from the input message.
func (y *yang) buildMapInternal(m *dpb.DescriptorProto, messageType string, ypm *yangProtoMap) {
	goMessageType := y.TypeName(y.ObjectNamed(messageType))
	if _, ok := ypm.MessageYANGFieldToProtoField[goMessageType]; !ok {
		ypm.MessageYANGFieldToProtoField[y.TypeName(y.ObjectNamed(messageType))] = map[string]string{}
	}

	for _, f := range m.Field {
		if path, ok := getSchemaPathAnnotation(f); ok {
			if *f.Type == dpb.FieldDescriptorProto_TYPE_MESSAGE {
				// Resolve this message to its Go type.
				ypm.MessagePathToGoType[path] = y.TypeName(y.ObjectNamed(f.GetTypeName()))
			}
			ypm.MessageYANGFieldToProtoField[goMessageType][path] = generator.CamelCase(*f.Name)
		}
		for _, nm := range m.NestedType {
			y.buildMapInternal(nm, strings.Join([]string{messageType, nm.GetName()}, "."), ypm)
		}
	}
}

// getSchemaPathAnnotation extracts the yext.schemapath annotation from the
// field f. It returns a string containing the schema path's contents, and a bool
// which indicates whether the schema path was extracted. It is intended to be
// called like a type check - i.e., name, ok := getSchemaPathAnnotation(f) such
// that handling code skips the field if an error is encountered (such as the
// annotation not being present). This approach allows the same binary to be
// used against protobufs output by ygot if only the enum annotations are used,
// and the schema path annotation is not present.
func getSchemaPathAnnotation(f *dpb.FieldDescriptorProto) (string, bool) {
	if o := f.Options; o != nil {
		ext, err := proto.GetExtension(o, yext.E_Schemapath)
		if err == nil && ext != nil {
			return *ext.(*string), true
		}
	}
	// If the annotation is not present, or its contents are empty
	// we indicate this to the caller by setting the bool return value to false.
	return "", false
}
