package yangplugin

import (
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/golang/protobuf/protoc-gen-go/generator"
	"github.com/openconfig/ygot/proto/yext"
)

func init() {
	generator.RegisterPlugin(new(yang))
}

var (
	// generatedProtoMap is a global variable used to catch the output of the
	// built map by the plugin. It is used only in testing.
	generatedProtoMap *yangProtoMap
)

// yang is the type used for the YANG<->Protobuf generator plugin.
type yang struct {
	gen *generator.Generator
}

// P wraps the generator.Generator P function which outputs code to the
// generated .pb.go file.
func (y *yang) P(args ...interface{}) { y.gen.P(args...) }

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

// Init is called by the protoc-gen-go generator package to initialise the YANG
// plugin. The provided gen is the current Generator instance being used to
// process the protobuf files for which code is being generated.
func (y *yang) Init(gen *generator.Generator) {
	y.gen = gen
}

// Name provides a string name for the plugin to protoc-gen-go's generator
// package. The name is used on the command-line as a plugin name.
func (y *yang) Name() string {
	return "yang"
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
// created by the generator (*yang).gen.
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
func (y *yang) buildMapInternal(m *descriptor.DescriptorProto, messageType string, ypm *yangProtoMap) {
	goMessageType := y.gen.TypeName(y.gen.ObjectNamed(messageType))
	if _, ok := ypm.MessageYANGFieldToProtoField[goMessageType]; !ok {
		ypm.MessageYANGFieldToProtoField[y.gen.TypeName(y.gen.ObjectNamed(messageType))] = map[string]string{}
	}

	for _, f := range m.Field {
		if path, ok := getSchemaPathAnnotation(f); ok {
			if *f.Type == descriptor.FieldDescriptorProto_TYPE_MESSAGE {
				// Resolve this message to its Go type.
				ypm.MessagePathToGoType[path] = y.gen.TypeName(y.gen.ObjectNamed(f.GetTypeName()))
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
func getSchemaPathAnnotation(f *descriptor.FieldDescriptorProto) (string, bool) {
	if o := f.Options; o != nil {
		ext, err := proto.GetExtension(o, yext.E_Schemapath)
		if err != nil {
			// We can safely ignore this, since in some cases, we don't have the
			// extension present. In this case, we just move on to the next field.
			return "", false
		}
		return *ext.(*string), true
	}
	return "", false
}
