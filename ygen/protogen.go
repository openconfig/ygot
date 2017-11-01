// Copyright 2017 Google Inc.
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
package ygen

import (
	"bytes"
	"fmt"
	"hash/fnv"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/openconfig/goyang/pkg/yang"
)

const (
	// protoEnumZeroName is the name given to the value 0 in each generated protobuf enum.
	protoEnumZeroName string = "UNSET"
	// protoAnyType is the name of the type to use for a google.protobuf.Any field.
	protoAnyType = "google.protobuf.Any"
	// protoAnyPackage is the name of the import to be used when a google.protobuf.Any field
	// is included in the output data.
	protoAnyPackage = "google/protobuf/any.proto"
	// protoListKeyMessageSuffix specifies the suffix that should be added to a list's name
	// to specify the repeated message that makes up the list's key. The repeated message is
	// called <ListNameInCamelCase><protoListKeyMessageSuffix>.
	protoListKeyMessageSuffix = "Key"
	// defaultBasePackageName defines the default base package that is
	// generated when generating proto3 code.
	DefaultBasePackageName = "openconfig"
	// defaultEnumPackageName defines the default package name that is
	// used for the package that defines enumerated types that are
	// used throughout the schema.
	DefaultEnumPackageName = "enums"
	// defaultYwrapperPath defines the default import path for the ywrapper.proto file,
	// excluding the filename.
	DefaultYwrapperPath = "github.com/openconfig/ygot/proto/ywrapper"
	// defaultYextPath defines the default import path for the yext.proto file, excluding
	// the filename.
	DefaultYextPath = "github.com/openconfig/ygot/proto/yext"
	// protoSchemaAnnotationOption specifies the name of the FieldOption used to annotate
	// schemapaths into a protobuf message.
	protoSchemaAnnotationOption = "(yext.schemapath)"
)

// protoMsgField describes a field of a protobuf message.
type protoMsgField struct {
	Tag         uint32           // Tag is the field number that should be used in the protobuf message.
	Name        string           // Name is the field's name.
	Type        string           // Type is the protobuf type for the field.
	IsRepeated  bool             // IsRepeated indicates whether the field is repeated.
	Options     []*protoOption   //Extensions is the set of field extensions that should be specified for the field.
	IsOneOf     bool             // IsOneOf indicates that the field is a oneof and hence consists of multiple subfields.
	OneOfFields []*protoMsgField // OneOfFields contains the set of fields within the oneof
}

// protoOption describes a protobuf (message or field) option.
type protoOption struct {
	// Name is the protobuf option's name.
	Name string
	// Value is the protobuf option's value.
	Value string
}

// protoMsg describes a protobuf message.
type protoMsg struct {
	Name     string                   // Name is the name of the protobuf message to be output.
	YANGPath string                   // YANGPath stores the path that the message corresponds to within the YANG schema.
	Fields   []*protoMsgField         // Fields is a slice of the fields that are within the message.
	Imports  []string                 // Imports is a slice of strings that contains the relative import paths that are required by this message.
	Enums    map[string]*protoMsgEnum // Enums lists the embedded enumerations within the message.
}

// protoMsgEnum represents an embedded enumeration within a protobuf message.
type protoMsgEnum struct {
	Values map[int64]protoEnumValue // Values that the enumerated type can take.
}

// protoEnumValue describes a value within a Protobuf enumeration.
type protoEnumValue struct {
	ProtoLabel string // ProtoLabel is the label that should be used for the value in the protobuf.
	YANGLabel  string // YANGLabel is the label that was originally specified in the YANG schema.
}

// protoEnum represents an enumeration that is defined at the root of a protobuf
// package.
type protoEnum struct {
	Name        string                   // Name is the enumeration's name within the protobuf package.
	Description string                   // Description is a string description of the enumerated type within the YANG schema, used in comments.
	Values      map[int64]protoEnumValue // Values contains the string names, keyed by enum value, that the enumerated type can take.
	ValuePrefix string                   // ValuePrefix contains the string prefix that should be prepended to each value within the enumerated type.
}

// proto3Header describes the header of a Protobuf3 package.
type proto3Header struct {
	PackageName            string   // PackageName is the name of the package that is to be output.
	Imports                []string // Imports is the set of packages that should be imported by the package whose header is being output.
	SourceYANGFiles        []string // SourceYANGFiles specifies the list of the input YANG files that the protobuf is being generated based on.
	SourceYANGIncludePaths []string // SourceYANGIncludePaths specifies the list of the paths that were used to search for YANG imports.
	CompressPaths          bool     // CompressPaths indicates whether path compression was enabled or disabled for this generated protobuf.
	CallerName             string   // CallerName indicates the name of the entity initiating code generation.
	YwrapperPath           string   // YwrapperPath is the path to the ywrapper.proto file, excluding the filename.
	YextPath               string   // YextPath is the path to the yext.proto file, excluding the filename.
}

var (
	// protoHeaderTemplate is populated and output at the top of the protobuf code output.
	protoHeaderTemplate = `
{{- /**/ -}}
// {{ .PackageName }} is generated by {{ .CallerName }} as a protobuf
// representation of a YANG schema.
//
// Input schema modules:
{{- range $inputFile := .SourceYANGFiles }}
//  - {{ $inputFile }}
{{- end }}
{{- if .SourceYANGIncludePaths }}
// Include paths:
{{- range $importPath := .SourceYANGIncludePaths }}
//   - {{ $importPath }}
{{- end -}}
{{- end }}
syntax = "proto3";

package {{ .PackageName }};

import "{{ .YwrapperPath }}/ywrapper.proto";
import "{{ .YextPath }}/yext.proto";
{{- range $importedProto := .Imports }}
import "{{ $importedProto }}";
{{- end }}
`

	// protoMessageTemplate is populated for each entity that is mapped to a message
	// within the output protobuf.
	protoMessageTemplate = `
// {{ .Name }} represents the {{ .YANGPath }} YANG schema element.
message {{ .Name }} {
{{- range $ename, $enum := .Enums }}
  enum {{ $ename }} {
    {{- range $i, $val := $enum.Values }}
    {{ toUpper $ename }}_{{ $val.ProtoLabel }} = {{ $i }}
    {{- if ne $val.YANGLabel "" }} [(yext.yang_name) = "{{ $val.YANGLabel }}"]{{ end -}}
    ;
    {{- end }}
  }
{{- end -}}
{{- range $idx, $field := .Fields }}
  {{ if $field.IsOneOf -}}
  oneof {{ $field.Name }} {
    {{- range $ooField := .OneOfFields }}
    {{ $ooField.Type }} {{ $ooField.Name }} = {{ $ooField.Tag }};
    {{- end }}
  }
  {{- else -}}
  {{ if $field.IsRepeated }}repeated {{ end -}}
  {{ $field.Type }} {{ $field.Name }} = {{ $field.Tag }}
  {{- $noOptions := len .Options -}}
  {{- if ne $noOptions 0 }} [
    {{- range $i, $opt := $field.Options -}}
      {{- $opt.Name }} = {{ $opt.Value -}}
      {{- if ne (inc $i) $noOptions -}}, {{- end }}
   {{- end -}}
  ]
  {{- end -}}
  ;
  {{- end -}}
{{- end }}
}
`

	// protoListKeyTemplate is generated as a wrapper around each list entry within
	// a YANG schema that has a key.
	protoListKeyTemplate = `
// {{ .Name }} represents the list element {{ .YANGPath }} of the YANG schema. It
// contains only the keys of the list, and an embedded message containing all entries
// below this entity in the schema.
message {{ .Name }} {
{{- range $ename, $enum := .Enums }}
  enum {{ $ename }} {
    {{- range $i, $val := $enum.Values }}
    {{ toUpper $ename }}_{{ $val.ProtoLabel }} = {{ $i }}
    {{- if ne $val.YANGLabel "" }} [(yext.yang_name) = "{{ $val.YANGLabel }}"]{{ end -}}
    ;
    {{- end }}
  }
{{- end -}}
{{- range $idx, $field := .Fields }}
  {{ if $field.IsOneOf -}}
  oneof {{ $field.Name }} {
    {{- range $ooField := .OneOfFields }}
    {{ $ooField.Type }} {{ $ooField.Name }} = {{ $ooField.Tag }};
    {{- end }}
  }
  {{- else -}}
  {{ $field.Type }} {{ $field.Name }} = {{ $field.Tag }}
  {{- $noOptions := len .Options -}}
  {{- if ne $noOptions 0 }} [
    {{- range $i, $opt := $field.Options -}}
      {{- $opt.Name }} = {{ $opt.Value -}}
      {{- if ne (inc $i) $noOptions -}}, {{- end }}
   {{- end -}}
  ]
  {{- end -}}
  ;
  {{- end }}
{{- end -}}
}
`

	// protoEnumTemplate is the template used to generate enumerations that are
	// not within a message. Such enums are used where there are referenced YANG
	// identity nodes, and where there are typedefs which include an enumeration.
	protoEnumTemplate = `
// {{ .Name }} represents an enumerated type generated for the {{ .Description }}.
enum {{ .Name }} {
{{- range $i, $val := .Values }}
  {{ toUpper $.ValuePrefix }}_{{ $val.ProtoLabel }} = {{ $i }}
  {{- if ne $val.YANGLabel "" }} [(yext.yang_name) = "{{ $val.YANGLabel }}"]{{ end -}}
  ;
{{- end }}
}
`

	// protoTemplates is the set of templates that are referenced during protbuf
	// code generation.
	protoTemplates = map[string]*template.Template{
		"header": makeTemplate("header", protoHeaderTemplate),
		"msg":    makeTemplate("msg", protoMessageTemplate),
		"list":   makeTemplate("list", protoListKeyTemplate),
		"enum":   makeTemplate("enum", protoEnumTemplate),
	}
)

// writeProto3Header outputs the header for a proto3 generated file. It takes
// an input proto3Header struct specifying the input arguments describing the
// generated package, and returns a string containing the generated package's
// header.
func writeProto3Header(in proto3Header) (string, error) {
	if in.CallerName == "" {
		in.CallerName = callerName()
	}

	// Sort the list of imports such that they are output in alphabetical
	// order, minimising diffs.
	sort.Strings(in.Imports)

	var b bytes.Buffer
	if err := protoTemplates["header"].Execute(&b, in); err != nil {
		return "", err
	}

	return b.String(), nil
}

// generatedProto3Message contains the code for a proto3 message.
type generatedProto3Message struct {
	packageName     string   // packageName is the name of the package that the proto3 message is within.
	messageCode     string   // messageCode contains the proto3 definition of the message.
	requiredImports []string // requiredImports contains the imports that are required by the generated message.
}

// protoMsgConfig defines the set of configuration options required to generate a Protobuf message.
type protoMsgConfig struct {
	compressPaths       bool   // compressPaths indicates whether path compression should be enabled.
	basePackageName     string // basePackageName specifies the package name that is the base for all child packages.
	enumPackageName     string // enumPackageName specifies the package in which global enum definitions are specified.
	baseImportPath      string // baseImportPath specifies the path that should be used for importing the generated files.
	annotateSchemaPaths bool   // annotateSchemaPaths uses the yext protobuf field extensions to annotate the paths from the schema into the output protobuf.
	annotateEnumNames   bool   // annotateEnumNames uses the yext protobuf enum value extensions to annoate the original YANG name for an enum into the output protobuf.
}

// writeProto3Message outputs the generated Protobuf3 code for a particular protobuf message. It takes:
//  - msg:               The yangDirectory struct that describes a particular protobuf3 message.
//  - msgs:              The set of other yangDirectory structs, keyed by schema path, that represent the other proto3
//                       messages to be generated.
//  - state:             The current generator state.
//  - cfg:		 The configuration for the message creation as defined in a protoMsgConfig struct.
//  It returns a generatedProto3Message pointer which includes the definition of the proto3 message, particularly the
//  name of the package it is within, the code for the message, and any imports for packages that are referenced by
//  the message.
func writeProto3Msg(msg *yangDirectory, msgs map[string]*yangDirectory, state *genState, cfg protoMsgConfig) (*generatedProto3Message, []error) {
	var pkg string
	switch {
	case msg.isFakeRoot:
		// In this case, we explicitly leave the package name as nil, which is interpeted
		// as meaning that the base package is used throughout the handling code.
	case msg.entry.Parent == nil:
		return nil, []error{fmt.Errorf("YANG schema element %s does not have a parent, protobuf messages are not generated for modules", msg.entry.Path())}
	default:
		// pkg is the name of the protobuf package, if the entry's parent has already
		// been seen in the schema, the same package name as for siblings of this
		// entry will be returned.
		pkg = state.protobufPackage(msg.entry, cfg.compressPaths)
	}

	msgDefs, errs := genProto3Msg(msg, msgs, state, cfg, pkg)
	if errs != nil {
		return nil, errs
	}

	var b bytes.Buffer
	imports := map[string]interface{}{}
	for _, msgDef := range msgDefs {
		if err := protoTemplates["msg"].Execute(&b, msgDef); err != nil {
			return nil, []error{err}
		}
		addNewKeys(imports, msgDef.Imports)
	}

	return &generatedProto3Message{
		packageName:     pkg,
		messageCode:     b.String(),
		requiredImports: stringKeys(imports),
	}, nil

}

// genProto3Msg takes an input yangDirectory which describes a container or list entry
// within the YANG schema and returns a protoMsg which can be mapped to the protobuf
// code representing it. It uses the set of messages that have been extracted and the
// current generator state to map to other messages and ensure uniqueness of names.
// The configuration parameters for the current code generation required are supplied
// as a protoMsgConfig struct. The parentPkg argument specifies the name of the parent
// package for the protobuf message(s) that are being generated, such that relative
// paths can be used in the messages.
// TODO(robjs): Split the logic of this function into multiple subfunctions.
func genProto3Msg(msg *yangDirectory, msgs map[string]*yangDirectory, state *genState, cfg protoMsgConfig, parentPkg string) ([]protoMsg, []error) {
	var errs []error

	var msgDefs []protoMsg

	msgDef := protoMsg{
		// msg.name is already specified to be CamelCase in the form we expect it
		// to be for the protobuf message name.
		Name:     msg.name,
		YANGPath: slicePathToString(msg.path),
		Enums:    make(map[string]*protoMsgEnum),
	}

	definedFieldNames := map[string]bool{}
	imports := map[string]interface{}{}

	fNames := []string{}
	for name := range msg.fields {
		fNames = append(fNames, name)
	}
	sort.Strings(fNames)

	skipFields := map[string]bool{}
	if isKeyedList(msg.entry) {
		skipFields = listKeyFieldsMap(msg.entry)
	}
	for _, name := range fNames {
		// Skip fields that we are explicitly not asked to include.
		if _, ok := skipFields[name]; ok {
			continue
		}

		field := msg.fields[name]

		fieldDef := &protoMsgField{
			Name: makeNameUnique(safeProtoIdentifierName(name), definedFieldNames),
		}

		t, err := protoTagForEntry(field)
		if err != nil {
			errs = append(errs, fmt.Errorf("proto: could not generate tag for field %s: %v", field.Name, err))
			continue
		}
		fieldDef.Tag = t

		switch {
		case field.IsList():
			listDef, keyMsg, err := protoListDefinition(protoDefinitionArgs{
				field:               field,
				definedDirectories:  msgs,
				state:               state,
				compressPaths:       cfg.compressPaths,
				basePackageName:     cfg.basePackageName,
				enumPackageName:     cfg.enumPackageName,
				baseImportPath:      cfg.baseImportPath,
				annotateSchemaPaths: cfg.annotateSchemaPaths,
				annotateEnumNames:   cfg.annotateEnumNames,
				parentPackage:       parentPkg,
			})

			if err != nil {
				errs = append(errs, fmt.Errorf("could not define list %s: %v", field.Path(), err))
				continue
			}

			if keyMsg != nil {
				msgDefs = append(msgDefs, *keyMsg)
			}

			fieldDef.Type = listDef.listType
			addNewKeys(imports, listDef.imports)

			// Lists are always repeated fields.
			fieldDef.IsRepeated = true
		case field.IsContainer():
			childmsg, ok := msgs[field.Path()]
			if !ok {
				err = fmt.Errorf("proto: could not resolve %s into a defined struct", field.Path())
			} else {
				var pfx string
				if cfg.compressPaths && msg.isFakeRoot {
					pfx = ""
				} else {
					childpkg := state.protobufPackage(childmsg.entry, cfg.compressPaths)
					// Add the import to the slice of imports if it is not already
					// there. This allows the message file to import the required
					// child packages.
					childpath := importPath(cfg.baseImportPath, cfg.basePackageName, childpkg)
					if _, ok := imports[childpath]; !ok {
						imports[childpath] = true
					}

					p, _ := stripPackagePrefix(parentPkg, childpkg)
					pfx = fmt.Sprintf("%s.", p)
				}
				fieldDef.Type = fmt.Sprintf("%s%s", pfx, childmsg.name)
			}
		case field.IsLeaf() || field.IsLeafList():
			d, err := protoLeafDefinition(fieldDef.Name, protoDefinitionArgs{
				field:             field,
				definedFieldNames: definedFieldNames,
				state:             state,
				basePackageName:   cfg.basePackageName,
				enumPackageName:   cfg.enumPackageName,
				annotateEnumNames: cfg.annotateEnumNames,
			})

			if err != nil {
				errs = append(errs, fmt.Errorf("could not define field %s: %v", field.Path(), err))
				continue
			}

			fieldDef.Type = d.protoType

			// For any enumerations that were within the field definition, glean them into the
			// message definition.
			for n, e := range d.enums {
				msgDef.Enums[n] = e
			}

			// For any oneof that is within the field definition, glean them into the message
			// definitions.
			if d.oneofs != nil {
				fieldDef.OneOfFields = append(fieldDef.OneOfFields, d.oneofs...)
				fieldDef.IsOneOf = true
			}

			if d.repeatedMsg != nil {
				msgDefs = append(msgDefs, *d.repeatedMsg)
			}

			// Add the global enumeration package if it is referenced by this field.
			if d.globalEnum {
				imports[importPath(cfg.baseImportPath, cfg.basePackageName, cfg.enumPackageName)] = true
			}

			if field.ListAttr != nil {
				fieldDef.IsRepeated = true
			}
		case isAnydata(field):
			fieldDef.Type = protoAnyType
			imports[protoAnyPackage] = true
		default:
			err = fmt.Errorf("proto: unknown field type in message %s, field %s", msg.name, field.Name)
		}

		if cfg.annotateSchemaPaths {
			o, err := protoSchemaPathAnnotation(msg, field, cfg.compressPaths)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			fieldDef.Options = append(fieldDef.Options, o)
		}

		if err != nil {
			errs = append(errs, err)
			continue
		}
		msgDef.Fields = append(msgDef.Fields, fieldDef)
	}

	msgDef.Imports = stringKeys(imports)

	return append(msgDefs, msgDef), errs
}

// protoDefinitionArgs is used as the input argument when YANG is being mapped to protobuf.
type protoDefinitionArgs struct {
	field               *yang.Entry               // field is the yang.Entry for which the proto output is being defined, in the case that the definition is for an individual entry.
	directory           *yangDirectory            // directory is the yangDirectory for which the proto output is being defined, in the case that the definition is for an directory entry.
	definedDirectories  map[string]*yangDirectory // definedDirectories specifies the set of yangDirectories that have been defined in the current code generation context.
	definedFieldNames   map[string]bool           // definedFieldNames specifies the field names that have been defined in the context.
	state               *genState                 //state is the current generator state.
	basePackageName     string                    // basePackageName is the name of the base protobuf package being output.
	enumPackageName     string                    // enumPackageName is the name of the package that global enumerated types are defined in.
	baseImportPath      string                    // baseImportPath is the path to be used as the root for imports of generated packages.
	compressPaths       bool                      // compressPaths defines whether path compression is enabled for the current code generation context.
	annotateSchemaPaths bool                      // annotateSchemaPaths defines whether fields should have their schema path annotated to them.
	annotateEnumNames   bool                      // annotateEnumNames defines whether values within enumerations should be annotated with their original name in the YANG schema.
	parentPackage       string                    // parentPackage stores the name of the protobuf package that the field's parent is within.
}

// writeProtoEnums takes a map of enumerated types within the YANG schema and
// returns the mapped Protobuf enum definition corresponding to each type. If
// the annotateEnumNames bool is set, then the original enum value label is
// stored in the definition. Since leaves that are of type enumeration are
// output directly within a Protobuf message, these are skipped.
func writeProtoEnums(enums map[string]*yangEnum, annotateEnumNames bool) ([]string, []error) {
	var errs []error
	var genEnums []string
	for _, enum := range enums {
		if isSimpleEnumerationType(enum.entry.Type) || enum.entry.Type.Kind == yang.Yunion {
			// Skip simple enumerations and those within unions.
			continue
		}

		// Make the name of the enum upper case to follow Protobuf enum convention.
		p := &protoEnum{Name: enum.name}
		switch {
		case isIdentityrefLeaf(enum.entry):
			// For an identityref the values are based on
			// the name of the identities that correspond with the base, and the value
			// is gleaned from the YANG schema.
			values := map[int64]protoEnumValue{
				0: {ProtoLabel: protoEnumZeroName},
			}

			// Ensure that we output the identity values in a determinstic order.
			nameMap := map[string]*yang.Identity{}
			names := []string{}
			for _, v := range enum.entry.Type.IdentityBase.Values {
				names = append(names, v.Name)
				nameMap[v.Name] = v
			}

			for _, n := range names {
				v := nameMap[n]
				// Calculate a tag value for the identity values, since otherwise when another
				// module augments this module then the enum values may be subject to change.
				tag, err := fieldTag(fmt.Sprintf("%s%s", enum.entry.Type.IdentityBase.Name, v.Name))
				if err != nil {
					errs = append(errs, fmt.Errorf("cannot calculate tag for %s: %v", v.Name, err))
				}

				values[int64(tag)] = toProtoEnumValue(strings.ToUpper(safeProtoIdentifierName(v.Name)), v.Name, annotateEnumNames)
			}
			p.Values = values
			p.ValuePrefix = strings.ToUpper(enum.name)
			p.Description = fmt.Sprintf("YANG identity %s", enum.entry.Type.IdentityBase.Name)
		case enum.entry.Type.Kind == yang.Yenum:
			ge, err := genProtoEnum(enum.entry, annotateEnumNames)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			p.Values = ge.Values

			// If the supplied enum entry has the valuePrefix annotation then use it to
			// calculate the enum value names.
			p.ValuePrefix = strings.ToUpper(enum.name)
			if e, ok := enum.entry.Annotation["valuePrefix"]; ok {
				t, ok := e.([]string)
				if ok {
					pp := []string{}
					for _, pe := range t {
						pp = append(pp, strings.ToUpper(safeProtoIdentifierName(yang.CamelCase(pe))))
					}
					p.ValuePrefix = strings.Join(pp, "_")
				}
			}

			p.Description = fmt.Sprintf("YANG enumerated type %s", enum.entry.Type.Name)
		case len(enum.entry.Type.Type) != 0:
			errs = append(errs, fmt.Errorf("unimplemented: support for multiple enumerations within a union for %v", enum.name))
			continue
		default:
			errs = append(errs, fmt.Errorf("unknown type of enumerated value in writeProtoEnums for %s, got: %v, type: %v", enum.name, enum, enum.entry.Type))
		}

		var b bytes.Buffer
		if err := protoTemplates["enum"].Execute(&b, p); err != nil {
			errs = append(errs, fmt.Errorf("cannot generate enumeration for %s: %v", enum.name, err))
			continue
		}
		genEnums = append(genEnums, b.String())
	}

	if len(errs) != 0 {
		return nil, errs
	}
	return genEnums, nil
}

// genProtoEnum takes an input yang.Entry that contains an enumerated type
// and returns a protoMsgEnum that contains its definition within the proto
// schema. If the annotateEnumNames bool is set, then the original YANG name
// is stored with each enum value.
func genProtoEnum(field *yang.Entry, annotateEnumNames bool) (*protoMsgEnum, error) {
	eval := map[int64]protoEnumValue{}
	names := field.Type.Enum.NameMap()
	eval[0] = protoEnumValue{ProtoLabel: protoEnumZeroName}

	if d := field.DefaultValue(); d != "" {
		if _, ok := names[d]; !ok {
			return nil, fmt.Errorf("enumeration %s specified a default - %s - that was not a valid value", field.Path(), d)
		}

		eval[0] = toProtoEnumValue(safeProtoIdentifierName(d), d, annotateEnumNames)
	}

	for n := range names {
		if n == field.DefaultValue() {
			// Can't happen if there was not a default, since "" is not
			// a valid enumeration name in YANG.
			continue
		}
		// Names are converted to upper case to follow the protobuf style guide,
		// adding one to ensure that the 0 value can represent unused values.
		eval[field.Type.Enum.Value(n)+1] = toProtoEnumValue(safeProtoIdentifierName(n), n, annotateEnumNames)
	}

	return &protoMsgEnum{Values: eval}, nil
}

type protoMsgList struct {
	listType string
	imports  []string
}

// protoListDefinition takes an input field described by a yang.Entry, the generator context (the set of proto messages, and the generator
// state), along with whether path compression is enabled and generates the proto message definition for the list. It returns the type
// that the field within the parent should be mapped to, and an optional key proto definition (in the case of keyed lists).
func protoListDefinition(args protoDefinitionArgs) (*protoMsgList, *protoMsg, error) {
	listMsg, ok := args.definedDirectories[args.field.Path()]
	if !ok {
		return nil, nil, fmt.Errorf("proto: could not resolve list %s into a defined message", args.field.Path())
	}

	listMsgName, ok := args.state.uniqueDirectoryNames[args.field.Path()]
	if !ok {
		return nil, nil, fmt.Errorf("proto: could not find unique message name for %s", args.field.Path())
	}

	childPkg := args.state.protobufPackage(listMsg.entry, args.compressPaths)

	var listKeyMsg *protoMsg
	var listDef *protoMsgList
	if !isKeyedList(listMsg.entry) {
		// In proto3 we represent unkeyed lists as a
		// repeated field of the parent message.
		p := fmt.Sprintf("%s.%s.%s", args.basePackageName, childPkg, listMsgName)
		p, _ = stripPackagePrefix(fmt.Sprintf("%s.%s", args.basePackageName, args.parentPackage), p)
		listDef = &protoMsgList{
			listType: p,
			imports:  []string{importPath(args.baseImportPath, args.basePackageName, childPkg)},
		}
	} else {
		// YANG lists are mapped to a repeated message structure as described
		// in the YANG to Protobuf transformation specification.
		// TODO(robjs): Link to the published transformation specification.
		var err error
		listKeyMsg, err = genListKeyProto(childPkg, listMsgName, protoDefinitionArgs{
			field:               args.field,
			directory:           listMsg,
			state:               args.state,
			basePackageName:     args.basePackageName,
			enumPackageName:     args.enumPackageName,
			baseImportPath:      args.baseImportPath,
			annotateSchemaPaths: args.annotateSchemaPaths,
			annotateEnumNames:   args.annotateEnumNames,
			parentPackage:       args.parentPackage,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("proto: could not build mapping for list entry %s: %v", args.field.Path(), err)
		}
		// The type of this field is just the key message's name, since it
		// will be in the same package as the field's parent.
		listDef = &protoMsgList{
			listType: listKeyMsg.Name,
		}
	}

	return listDef, listKeyMsg, nil
}

// protoDefinedLeaf defines a YANG leaf within a protobuf message.
type protoDefinedLeaf struct {
	protoType   string                   // protoType is the protobuf type that the leaf should be mapped to.
	globalEnum  bool                     // globalEnum indicates whether the leaf's type is a global scope enumeration (identityref, or typedef defining an enumeration)
	enums       map[string]*protoMsgEnum // enums defines the set of enumerated values that are required for this leaf within the parent message.
	oneofs      []*protoMsgField         // oneofs defines the set of types within the leaf, if the returned leaf type is a protobuf oneof.
	repeatedMsg *protoMsg                // repeatedMsgs returns a message that should be repeated for this leaf, used in the case of a leaf-list of unions.
}

// protoLeafDefinition takes an input leafName, and a set of protoDefinitionArgs specifying the context
// for the leaf definition, and returns a protoDefinedLeaf describing how it is to be mapped within the
// protobuf parent message.
func protoLeafDefinition(leafName string, args protoDefinitionArgs) (*protoDefinedLeaf, error) {
	protoType, err := args.state.yangTypeToProtoType(resolveTypeArgs{
		yangType:     args.field.Type,
		contextEntry: args.field,
	}, resolveProtoTypeArgs{
		basePackageName: args.basePackageName,
		enumPackageName: args.enumPackageName,
	})
	if err != nil {
		return nil, err
	}

	d := &protoDefinedLeaf{
		protoType: protoType.nativeType,
		enums:     map[string]*protoMsgEnum{},
	}

	switch {
	case isSimpleEnumerationType(args.field.Type):
		// For fields that are simple enumerations within a message, then we embed an enumeration
		// within the Protobuf message.
		e, err := genProtoEnum(args.field, args.annotateEnumNames)
		if err != nil {
			return nil, err
		}

		d.protoType = makeNameUnique(protoType.nativeType, args.definedFieldNames)
		d.enums = map[string]*protoMsgEnum{}
		d.enums[d.protoType] = e
	case isEnumType(args.field.Type):
		d.globalEnum = true
	case isUnionType(args.field.Type) && protoType.unionTypes != nil:
		u, err := unionFieldToOneOf(leafName, args.field, protoType, args.annotateEnumNames)
		if err != nil {
			return nil, err
		}

		// Append any enumerations that are within the union.
		for n, e := range u.enums {
			d.enums[n] = e
		}

		d.globalEnum = u.hadGlobalEnums

		// Append the oneof that was in the union.
		d.oneofs = append(d.oneofs, u.oneOfFields...)

		if u.repeatedMsg != nil {
			d.repeatedMsg = u.repeatedMsg
			d.protoType = u.repeatedMsg.Name
		}
	}

	return d, nil
}

// toProtoEnumValue takes an input enum definition - with a protobuf and YANG label, and returns
// a protoEnumValue. The YANGLabel is only stored if annotateEnumValues is set.
func toProtoEnumValue(protoName, yangName string, annotateEnumValues bool) protoEnumValue {
	ev := protoEnumValue{
		ProtoLabel: protoName,
	}
	if annotateEnumValues {
		ev.YANGLabel = yangName
	}
	return ev
}

// safeProtoIdentifierName takes an input string which represents the name of a YANG schema
// element and sanitises for use as a protobuf field name.
func safeProtoIdentifierName(name string) string {
	// YANG identifiers must match the definition:
	//    ;; An identifier MUST NOT start with (('X'|'x') ('M'|'m') ('L'|'l'))
	//       identifier          = (ALPHA / "_")
	//                                *(ALPHA / DIGIT / "_" / "-" / ".")
	// For Protobuf they must match:
	//	ident = letter { letter | decimalDigit | "_" }
	//
	// Therefore we need to ensure that the "-", and "." characters that are allowed
	// in the YANG are replaced.
	replacer := strings.NewReplacer(
		".", "_",
		"-", "_",
	)
	return replacer.Replace(name)
}

// protoTagForEntry returns a protobuf tag value for the entry e.
func protoTagForEntry(e *yang.Entry) (uint32, error) {
	return fieldTag(e.Path())
}

// fieldTag takes an input string and calculates a FNV hash for the value. If the
// hash is in the range 19,000-19,999 or 1-1,000, the input string has _ appended to
// it and the hash is calculated.
func fieldTag(s string) (uint32, error) {
	h := fnv.New32()
	if _, err := h.Write([]byte(s)); err != nil {
		return 0, fmt.Errorf("could not write field path to hash: %v", err)
	}

	v := h.Sum32() & 0x1fffffff // 2^29-1
	if (v >= 19000 && v <= 19999) || (v >= 1 && v <= 1000) {
		return fieldTag(fmt.Sprintf("%s_", s))
	}
	return v, nil
}

// genListKeyProto generates a protoMsg that describes the proto3 message that represents
// the key of a list for YANG lists. It takes a yangDirectory pointer to the list being
// described, the name of the list, the package name that the list is within, and the
// current generator state. It returns the definition of the list key proto.
func genListKeyProto(listPackage string, listName string, args protoDefinitionArgs) (*protoMsg, error) {
	n := fmt.Sprintf("%s%s", listName, protoListKeyMessageSuffix)
	km := &protoMsg{
		Name:     n,
		YANGPath: args.field.Path(),
		Enums:    map[string]*protoMsgEnum{},
	}

	if listPackage != "" {
		km.Imports = []string{importPath(args.baseImportPath, args.basePackageName, listPackage)}
	}

	definedFieldNames := map[string]bool{}
	ctag := uint32(1)
	for _, k := range strings.Split(args.field.Key, " ") {
		kf, ok := args.directory.fields[k]
		if !ok {
			return nil, fmt.Errorf("list %s included a key %s did that did not exist", args.field.Path(), k)
		}

		scalarType, err := args.state.yangTypeToProtoScalarType(resolveTypeArgs{
			yangType:     kf.Type,
			contextEntry: kf,
		}, resolveProtoTypeArgs{
			basePackageName: args.basePackageName,
			enumPackageName: args.enumPackageName,
			// When there is a union within a list key that has a single type within it
			// e.g.,:
			// list foo {
			//   key "bar";
			//   leaf bar {
			//     type union {
			//       type string { pattern "a.*" }
			//			 type string { pattern "b.*" }
			//     }
			//   }
			// }
			// Then we want to use the scalar type rather than the wrapper type in
			// this message since all keys must be set. We therefore signal this in
			// the call to the type resolution.
			scalarTypeInSingleTypeUnion: true,
		})
		if err != nil {
			return nil, fmt.Errorf("list %s included a key %s that did not have a valid proto type: %v", args.field.Path(), k, kf.Type)
		}

		var enumEntry *yang.Entry
		var unionEntry *yang.Entry
		switch {
		case kf.Type.Kind == yang.Yleafref:
			target, err := args.state.resolveLeafrefTarget(kf.Type.Path, kf)
			if err != nil {
				return nil, fmt.Errorf("error generating type for list %s key %s: type %v", args.field.Path(), k, kf.Type)
			}

			if isSimpleEnumerationType(target.Type) {
				enumEntry = target
			}

			if isUnionType(target.Type) && scalarType.unionTypes != nil {
				unionEntry = target
			}

			if isIdentityrefLeaf(target) {
				km.Imports = append(km.Imports, importPath(args.baseImportPath, args.basePackageName, args.enumPackageName))
			}
		case isSimpleEnumerationType(kf.Type):
			enumEntry = kf
		case isUnionType(kf.Type) && scalarType.unionTypes != nil:
			unionEntry = kf
		}

		fd := &protoMsgField{
			Name: makeNameUnique(safeProtoIdentifierName(k), definedFieldNames),
			Tag:  ctag,
		}
		switch {
		case enumEntry != nil:
			enum, err := genProtoEnum(enumEntry, args.annotateEnumNames)
			if err != nil {
				return nil, fmt.Errorf("error generating type for list %s key %s, type %v", args.field.Path(), k, enumEntry.Type)
			}
			tn := makeNameUnique(scalarType.nativeType, definedFieldNames)
			fd.Type = tn
			km.Enums[tn] = enum
		case unionEntry != nil:
			fd.IsOneOf = true
			u, err := unionFieldToOneOf(fd.Name, kf, scalarType, args.annotateEnumNames)
			if err != nil {
				return nil, fmt.Errorf("error generating type for union list key %s in list %s", k, args.field.Path())
			}
			fd.OneOfFields = append(fd.OneOfFields, u.oneOfFields...)
			for n, e := range u.enums {
				km.Enums[n] = e
			}
			if u.hadGlobalEnums {
				km.Imports = append(km.Imports, importPath(args.baseImportPath, args.basePackageName, args.enumPackageName))
			}
		default:
			fd.Type = scalarType.nativeType
		}

		if args.annotateSchemaPaths {
			o, err := protoSchemaPathAnnotation(args.directory, kf, args.compressPaths)
			if err != nil {
				return nil, err
			}
			fd.Options = append(fd.Options, o)
		}

		km.Fields = append(km.Fields, fd)
		ctag++
	}

	p, _ := stripPackagePrefix(args.parentPackage, listPackage)
	ltype := fmt.Sprintf("%s.%s", p, listName)
	if listPackage == "" {
		// Handle the case that the context of the list is already the base package.
		ltype = listName
	}

	km.Fields = append(km.Fields, &protoMsgField{
		Name: safeProtoIdentifierName(args.field.Name),
		Type: ltype,
		Tag:  ctag,
	})

	return km, nil
}

// enumInProtoUnionField parses an enum that is within a union and returns the generated
// enumeration that should be included within a protobuf message for it. If annotateEnumNames
// is set to true, the enumerated value's original names are stored.
func enumInProtoUnionField(name string, types []*yang.YangType, annotateEnumNames bool) (map[string]*protoMsgEnum, error) {
	enums := map[string]*protoMsgEnum{}
	for _, t := range types {
		if isSimpleEnumerationType(t) {
			n := fmt.Sprintf("%s", yang.CamelCase(name))
			enum, err := genProtoEnum(&yang.Entry{
				Name: n,
				Type: t,
			}, annotateEnumNames)
			if err != nil {
				return nil, err
			}
			enums[n] = enum
		}
	}

	return enums, nil
}

// protoUnionField stores information relating to a oneof field within a protobuf
// message.
type protoUnionField struct {
	oneOfFields    []*protoMsgField         // oneOfFields contains a set of fields that are within a oneof.
	enums          map[string]*protoMsgEnum // enums stores a definition of any simple enumeration types within the YANG union.
	repeatedMsg    *protoMsg                // repeatedMsg stores a message that contains fields that should be repeated, and is used to store a YANG leaf-list of union leaves.
	hadGlobalEnums bool                     // hadGlobalEnums determines whether there was a global scope enum (typedef, identityref) in the message.
}

// unionFieldToOneOf takes an input name, a yang.Entry containing a field definition and a mappedType
// containing the proto type that the entry has been mapped to, and returns a definition of a union
// field within the protobuf message. If the annotateEnumNames boolean is set, then any enumerated types
// within the union have their original names within the YANG schema appended.
func unionFieldToOneOf(fieldName string, e *yang.Entry, mtype *mappedType, annotateEnumNames bool) (*protoUnionField, error) {
	enums, err := enumInProtoUnionField(fieldName, e.Type.Type, annotateEnumNames)
	if err != nil {
		return nil, err
	}

	typeNames := []string{}
	for tn := range mtype.unionTypes {
		typeNames = append(typeNames, tn)
	}
	sort.Strings(typeNames)

	var importGlobalEnums bool
	oofs := []*protoMsgField{}
	for _, t := range typeNames {
		// Split the type name on "." to ensure that we don't have oneof options
		// that reference some other package in the type name. If there was a "."
		// in the field name, then this means that we had a global enumeration
		// present and hence should import this path.
		tp := strings.Split(t, ".")
		if len(tp) > 1 {
			importGlobalEnums = true
		}
		tn := tp[len(tp)-1]
		// Calculate the tag by having the path, with the type name appended to it
		// such that we have unique inputs for each option. We make the name lower-case
		// as it is conventional that protobuf field names are lowercase separated by
		// underscores.
		ft, err := fieldTag(fmt.Sprintf("%s_%s", e.Path(), strings.ToLower(tn)))
		if err != nil {
			return nil, fmt.Errorf("could not calculate tag number for %s, type %s in oneof", e.Path(), tn)
		}
		st := &protoMsgField{
			Name: fmt.Sprintf("%s_%s", fieldName, strings.ToLower(tn)),
			Type: t,
			Tag:  ft,
		}
		oofs = append(oofs, st)
	}

	if e.IsLeafList() {
		// In this case, we cannot return a oneof, since it is not possible to have a repeated
		// oneof, therefore we return a message that contains the protoMsgFields that are defined
		// above.
		p := &protoMsg{
			Name:     fmt.Sprintf("%s%sUnion", yang.CamelCase(e.Parent.Name), yang.CamelCase(fieldName)),
			YANGPath: fmt.Sprintf("%s union field %s", e.Path(), e.Name),
			Fields:   oofs,
		}

		return &protoUnionField{
			enums:          enums,
			repeatedMsg:    p,
			hadGlobalEnums: importGlobalEnums,
		}, nil
	}

	return &protoUnionField{
		oneOfFields:    oofs,
		enums:          enums,
		hadGlobalEnums: importGlobalEnums,
	}, nil
}

// protoPackageToFilePath takes an input string containing a period separated protobuf package
// name in the form parent.child and returns a path to the file that it should be written to
// assuming a hierarchical directory structure is used. If the package supplied is
// openconfig.interfaces.interface, it is returned as []string{"openconfig", "interfaces",
// "interface.proto"} such that filepath.Join can create the relevant file system path
// for the input package.
func protoPackageToFilePath(pkg string) []string {
	pp := strings.Split(pkg, ".")
	return append(pp, fmt.Sprintf("%s.proto", pp[len(pp)-1]))
}

// protoSchemaPathAnnotation takes a protobuf message and field, and returns the protobuf
// field option definitions required to annotate it with its schema path(s).
func protoSchemaPathAnnotation(msg *yangDirectory, field *yang.Entry, compressPaths bool) (*protoOption, error) {
	// protobuf paths are always absolute.
	smapp, err := findMapPaths(msg, field, compressPaths, true)
	if err != nil {
		return nil, err
	}
	var b bytes.Buffer
	b.WriteRune('"')
	for i, p := range smapp {
		b.WriteString(slicePathToString(p))
		if i != len(smapp)-1 {
			b.WriteString("|")
		}
	}
	b.WriteRune('"')
	return &protoOption{Name: protoSchemaAnnotationOption, Value: b.String()}, nil
}

// stripPackagePrefix removes the prefix of pfx from the path supplied. If pfx
// is not a prefix of path the entire path is returned. If the prefix was
// stripped, the returned bool is set.
func stripPackagePrefix(pfx, path string) (string, bool) {
	pfxP := strings.Split(pfx, ".")
	pathP := strings.Split(path, ".")

	var i int
	for i = range pfxP {
		if pfxP[i] != pathP[i] {
			return path, false
		}
	}

	return strings.Join(pathP[i+1:], "."), true
}

// importPath returns a string indicating the import path for a particular
// child package - considering the base import path, and base package name
// for the generated set of protobuf messages.
func importPath(baseImportPath, basePkgName, childPkg string) string {
	return filepath.Join(append([]string{baseImportPath}, protoPackageToFilePath(fmt.Sprintf("%s.%s", basePkgName, childPkg))...)...)
}
