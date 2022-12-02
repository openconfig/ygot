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

// Package protogen contains code for generating proto code given YANG input.
package protogen

import (
	"bytes"
	"fmt"
	"hash/fnv"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/genutil"
	"github.com/openconfig/ygot/internal/igenutil"
	"github.com/openconfig/ygot/util"
	"github.com/openconfig/ygot/ygen"
)

// Constants defining the defaults for Protobuf package generation. These constants
// can be referred to by calling applications as defaults that are presented to
// a user.
const (
	// DefaultBasePackageName defines the default base package that is
	// generated when generating proto3 code.
	DefaultBasePackageName = "openconfig"
	// DefaultEnumPackageName defines the default package name that is
	// used for the package that defines enumerated types that are
	// used throughout the schema.
	DefaultEnumPackageName = "enums"
	// DefaultYwrapperPath defines the default import path for the ywrapper.proto file,
	// excluding the filename.
	DefaultYwrapperPath = "github.com/openconfig/ygot/proto/ywrapper"
	// DefaultYextPath defines the default import path for the yext.proto file, excluding
	// the filename.
	DefaultYextPath = "github.com/openconfig/ygot/proto/yext"
	// ywrapperAccessor is the package accessor to the ywrapper.proto
	// file's definitions.
	ywrapperAccessor = "ywrapper."
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
	// protoSchemaAnnotationOption specifies the name of the FieldOption used to annotate
	// schemapaths into a protobuf message.
	protoSchemaAnnotationOption = "(yext.schemapath)"
	// protoMatchingListNameKeySuffix defines the suffix that should be added to a list
	// key's name in the case that it matches the name of the list itself. This is required
	// since in the case that we have YANG whereby there is a list that has a key
	// with the same name as the list, i.e.,:
	//
	// list foo {
	//   key "foo";
	//   leaf foo { type string; }
	// }
	//
	// Then we need to ensure that we do not generate a message that has the
	// same field name used twice, i.e.:
	//
	// message FooParent {
	//   message Foo {
	//     ywrapper.StringValue foo = NN;
	//   }
	//   message FooKey {
	//     string foo = 1;
	//     Foo foo = 2;
	//   }
	//   repeated FooKey foo = NN;
	// }
	//
	// which may otherwise occur. In these cases, rather than rely on
	// genutil.MakeNameUnique which would append "_" to the name of the key we explicitly
	// append _ plus the string defined in protoMatchingListNameKeySuffix to the list name.
	protoMatchingListNameKeySuffix = "key"
	// protoLeafListAnnotationOption specifies the name of the FieldOption used to annotate
	// whether repeated fields are leaf-lists.
	protoLeafListAnnotationOption = "(yext.leaflist)"
	// protoLeafListUnionAnnotationOption specifies the name of the FieldOption used to annotate
	// whether repeated fields are leaf-lists of unions.
	protoLeafListUnionAnnotationOption = "(yext.leaflistunion)"
)

// protoMsgField describes a field of a protobuf message.
// Note, throughout this package private structs that have public fields are used
// in text/template which cannot refer to unexported fields.
type protoMsgField struct {
	Tag         uint32           // Tag is the field number that should be used in the protobuf message.
	Name        string           // Name is the field's name.
	Type        string           // Type is the protobuf type for the field.
	IsRepeated  bool             // IsRepeated indicates whether the field is repeated.
	Options     []*protoOption   // Extensions is the set of field extensions that should be specified for the field.
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
	Name        string                    // Name is the name of the protobuf message to be output.
	YANGPath    string                    // YANGPath stores the path that the message corresponds to within the YANG schema.
	Fields      []*protoMsgField          // Fields is a slice of the fields that are within the message.
	Imports     []string                  // Imports is a slice of strings that contains the relative import paths that are required by this message.
	Enums       map[string]*protoMsgEnum  // Enums lists the embedded enumerations within the message.
	ChildMsgs   []*generatedProto3Message // ChildMsgs is the set of messages that should be embedded within the message.
	PathComment bool                      // PathComment - when set - indicates that comments that specify the path to a message should be included in the output protobuf.
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
	GoPackageName          string   // GoPackageName is the contents of the go_package fileoption in the generated protobuf.
}

var disallowedInProtoIDRegexp = regexp.MustCompile(`[^a-zA-Z0-9_]`)

// mustMakeTemplate generates a template.Template for a particular named source
// template; with a common set of helper functions.
func mustMakeTemplate(name, src string) *template.Template {
	return template.Must(template.New(name).Funcs(igenutil.TemplateHelperFunctions).Parse(src))
}

var (
	// protoHeaderTemplate is populated and output at the top of the protobuf code output.
	protoHeaderTemplate = mustMakeTemplate("header", `
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
{{- if or .YwrapperPath .YextPath .Imports }}
{{ end -}}
{{ if .YwrapperPath }}
import "{{ .YwrapperPath }}/ywrapper.proto";
{{- end -}}
{{ if .YextPath }}
import "{{ .YextPath }}/yext.proto";
{{- end -}}
{{ range $importedProto := .Imports }}
import "{{ $importedProto }}";
{{- end -}}

{{- if .GoPackageName }}

option go_package = "{{ .GoPackageName }}";
{{- end }}
`)

	// protoMessageTemplate is populated for each entity that is mapped to a message
	// within the output protobuf.
	protoMessageTemplate = mustMakeTemplate("msg", `
{{ if .PathComment -}}
// {{ .Name }} represents the {{ .YANGPath }} YANG schema element.
{{ end -}}
message {{ .Name }} {
{{- range $idx, $msg := .ChildMsgs -}}
	{{- indentLines $msg.MessageCode -}}
{{- end -}}
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
    {{- $noOptions := len $field.Options -}}
    {{- range $ooField := .OneOfFields }}
    {{ $ooField.Type }} {{ $ooField.Name }} = {{ $ooField.Tag }}
    {{- if ne $noOptions 0 }} [
      {{- range $i, $opt := $field.Options -}}
        {{ $opt.Name }} = {{ $opt.Value -}}
        {{- if ne (inc $i) $noOptions -}}, {{- end }}
      {{- end -}}
      ]
      {{- end -}}
      ;
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
}`)

	// protoEnumTemplate is the template used to generate enumerations that are
	// not within a message. Such enums are used where there are referenced YANG
	// identity nodes, and where there are typedefs which include an enumeration.
	protoEnumTemplate = mustMakeTemplate("enum", `
// {{ .Name }} represents an enumerated type generated for the {{ .Description }}.
enum {{ .Name }} {
{{- range $i, $val := .Values }}
  {{ toUpper $.ValuePrefix }}_{{ $val.ProtoLabel }} = {{ $i }}
  {{- if ne $val.YANGLabel "" }} [(yext.yang_name) = "{{ $val.YANGLabel }}"]{{ end -}}
  ;
{{- end }}
}
`)
)

// writeProto3Header outputs the header for a proto3 generated file. It takes
// an input proto3Header struct specifying the input arguments describing the
// generated package, and returns a string containing the generated package's
// header.
func writeProto3Header(in proto3Header) (string, error) {
	if in.CallerName == "" {
		in.CallerName = genutil.CallerName()
	}

	// Sort the list of imports such that they are output in alphabetical
	// order, minimising diffs.
	sort.Strings(in.Imports)

	var b bytes.Buffer
	if err := protoHeaderTemplate.Execute(&b, in); err != nil {
		return "", err
	}

	return b.String(), nil
}

// generatedProto3Message contains the code for a proto3 message.
type generatedProto3Message struct {
	PackageName        string   // PackageName is the name of the package that the proto3 message is within.
	MessageCode        string   // MessageCode contains the proto3 definition of the message.
	RequiredImports    []string // RequiredImports contains the imports that are required by the generated message.
	UsesYwrapperImport bool     // UsesYwrapperImport indicates whether the ywrapper proto package is used by the generated message.
	UsesYextImport     bool     // UsesYextImport indicates whether the yext proto package is used by the generated message.
}

// protoMsgConfig defines the set of configuration options required to generate a Protobuf message.
type protoMsgConfig struct {
	compressPaths       bool   // compressPaths indicates whether path compression should be enabled.
	basePackageName     string // basePackageName specifies the package name that is the base for all child packages.
	enumPackageName     string // enumPackageName specifies the package in which global enum definitions are specified.
	baseImportPath      string // baseImportPath specifies the path that should be used for importing the generated files.
	annotateSchemaPaths bool   // annotateSchemaPaths uses the yext protobuf field extensions to annotate the paths from the schema into the output protobuf.
	annotateEnumNames   bool   // annotateEnumNames uses the yext protobuf enum value extensions to annoate the original YANG name for an enum into the output protobuf.
	nestedMessages      bool   // nestedMessages indicates whether nested messages should be output for the protobuf schema.
}

// writeProto3Message outputs the generated Protobuf3 code for a particular protobuf message. It takes:
//   - msg:               The Directory struct that describes a particular protobuf3 message.
//   - msgs:              The set of other Directory structs, keyed by schema path, that represent the other proto3
//     messages to be generated.
//   - protogen:             The current generator state.
//   - cfg:		 The configuration for the message creation as defined in a protoMsgConfig struct.
//     It returns a generatedProto3Message pointer which includes the definition of the proto3 message, particularly the
//     name of the package it is within, the code for the message, and any imports for packages that are referenced by
//     the message.
func writeProto3Msg(msg *ygen.ParsedDirectory, ir *ygen.IR, cfg *protoMsgConfig) (*generatedProto3Message, util.Errors) {
	if cfg.nestedMessages {
		if !outputNestedMessage(msg, cfg.compressPaths) {
			return nil, nil
		}
		return writeProto3MsgNested(msg, ir, cfg)
	}
	return writeProto3MsgSingleMsg(msg, ir, cfg)
}

// isChildOfModule determines whether the Directory represents a container
// or list member that is the direct child of a module entry.
func isChildOfModule(y *ygen.ParsedDirectory) bool {
	if y.IsFakeRoot || len(strings.Split(y.Path, "/")) == 3 {
		// If the message has a path length of 3, then it is a top-level entity
		// within a module, since the  path is in the format []{"", <module>, <element>}.
		return true
	}
	return false
}

// outputNestedMessage determines whether the message represented by the supplied
// Directory is a message that should be output when nested messages are being
// created. The compressPaths argument specifies whether path compression is enabled.
// Valid messages are those that are direct children of a module, or become a direct
// child when path compression is enabled (i.e., lists that have their parent
// surrounding container removed).
func outputNestedMessage(msg *ygen.ParsedDirectory, compressPaths bool) bool {
	// If path compression is enabled, and this entry is a list, then its top-level
	// parent will have been removed, therefore this is a valid message. The path
	// is 4 elements long since it is of the form
	// []string{"", module-name, surrounding-container, list-name}.
	if compressPaths && msg.Type == ygen.List && len(strings.Split(msg.Path, "/")) == 4 {
		return true
	}

	return isChildOfModule(msg)
}

// writeProto3MsgNested returns a nested set of protobuf messages for the message
// supplied, which is expected to be a top-level message that code generation is
// being performed for. It takes:
//   - msg: the top-level directory definition
//   - msgs: the set of message definitions (keyed by path) that are to be output
//   - protogen: the current code generation state.
//   - cfg: the configuration for the current code generation.
//
// It returns a generated protobuf3 message.
func writeProto3MsgNested(msg *ygen.ParsedDirectory, ir *ygen.IR, cfg *protoMsgConfig) (*generatedProto3Message, util.Errors) {
	var gerrs util.Errors
	var childMsgs []*generatedProto3Message
	if !msg.IsFakeRoot {
		// Except the fake root message, which should always be a
		// separate definition, find all the children of the current
		// message that should be output.
		childDirs, err := msg.ChildDirectories(ir)
		if err != nil {
			return nil, append(gerrs, err)
		}
		for _, n := range childDirs {
			cmsg, errs := writeProto3MsgNested(n, ir, cfg)
			if errs != nil {
				gerrs = append(gerrs, errs...)
				continue
			}
			childMsgs = append(childMsgs, cmsg)
		}
	}

	// Generate this message, and its associated messages.
	msgDefs, errs := genProto3Msg(msg, ir, cfg, msg.PackageName, childMsgs)
	if errs != nil {
		return nil, append(gerrs, errs...)
	}

	gmsg, errs := genProto3MsgCode(cfg, msg.PackageName, msgDefs, false)
	if errs != nil {
		return nil, append(gerrs, errs...)
	}

	if gerrs != nil {
		return nil, gerrs
	}

	// Inherit the set of imports that are required for this child. We
	// skip any that are relative imports as these are only needed for
	// the case that we have different files per hierarchy level and
	// are not nesting messages.
	var imports []string
	if msg.IsFakeRoot {
		imports = gmsg.RequiredImports
	} else {
		allImports := map[string]bool{}
		for _, ch := range childMsgs {
			for _, i := range ch.RequiredImports {
				allImports[i] = true
			}
			// Inherit yext and ywrapper imports.
			if ch.UsesYextImport {
				gmsg.UsesYextImport = true
			}
			if ch.UsesYwrapperImport {
				gmsg.UsesYwrapperImport = true
			}
		}
		for _, i := range gmsg.RequiredImports {
			allImports[i] = true
		}

		epk := filepath.Join(cfg.baseImportPath, cfg.basePackageName, cfg.enumPackageName, fmt.Sprintf("%s.proto", cfg.enumPackageName))
		for i := range allImports {
			if !strings.HasPrefix(i, cfg.baseImportPath) {
				imports = append(imports, i)
			}
			if allImports[epk] {
				imports = append(imports, epk)
			}
		}
	}
	gmsg.RequiredImports = imports

	return gmsg, nil
}

// writeProto3MsgSingleMsg generates a protobuf message definition. It takes the
// arguments of writeProto3Message, outputting an individual message that outputs
// a package definition and a single protobuf message.
func writeProto3MsgSingleMsg(msg *ygen.ParsedDirectory, ir *ygen.IR, cfg *protoMsgConfig) (*generatedProto3Message, util.Errors) {
	msgDefs, errs := genProto3Msg(msg, ir, cfg, msg.PackageName, nil)
	if errs != nil {
		return nil, errs
	}

	return genProto3MsgCode(cfg, msg.PackageName, msgDefs, true)
}

// genProto3MsgCode takes an input package name, and set of protobuf message
// definitions, and outputs the generated code for the messages. If the
// pathComment argument is setFunc, each message is output with a comment
// indicating its path in the YANG schema, otherwise it is included.
func genProto3MsgCode(cfg *protoMsgConfig, pkg string, msgDefs []*protoMsg, pathComment bool) (*generatedProto3Message, util.Errors) {
	var b bytes.Buffer
	var errs util.Errors
	imports := map[string]interface{}{}
	var usesYwrapperImport, usesYextImport bool
	for i, msgDef := range msgDefs {
		// Sort the child messages into a determinstic order. We cannot use the
		// package name as a key as it may be the same for multiple packages, therefore
		// use the code.
		cmsgs := map[string]*generatedProto3Message{}
		var cstrs []string
		for _, m := range msgDef.ChildMsgs {
			if m == nil {
				errs = append(errs, fmt.Errorf("received nil message in %s", pkg))
				continue
			}
			cmsgs[m.MessageCode] = m
			cstrs = append(cstrs, m.MessageCode)
		}
		sort.Strings(cstrs)
		var nm []*generatedProto3Message
		for _, c := range cstrs {
			nm = append(nm, cmsgs[c])
		}
		msgDef.ChildMsgs = nm
		msgDef.PathComment = pathComment

		// If one of the fields uses a definition from the ywrapper or
		// yext packages, then make sure to mark it for import.
		for _, field := range msgDef.Fields {
			if strings.HasPrefix(field.Type, ywrapperAccessor) {
				usesYwrapperImport = true
			}
			for _, f := range field.OneOfFields {
				if strings.HasPrefix(f.Type, ywrapperAccessor) {
					usesYwrapperImport = true
				}
			}
			for _, o := range field.Options {
				if o.Name == protoSchemaAnnotationOption {
					usesYextImport = true
				}
			}
		}
		// If there is any annotated enums, then make sure to mark the
		// yext package for import.
		if cfg.annotateEnumNames && len(msgDef.Enums) > 0 {
			usesYextImport = true
		}

		if err := protoMessageTemplate.Execute(&b, msgDef); err != nil {
			return nil, []error{err}
		}
		addNewKeys(imports, msgDef.Imports)
		if i != len(msgDefs)-1 {
			b.WriteRune('\n')
		}
	}

	if errs != nil {
		return nil, errs
	}

	return &generatedProto3Message{
		PackageName:        pkg,
		MessageCode:        b.String(),
		RequiredImports:    stringKeys(imports),
		UsesYwrapperImport: usesYwrapperImport,
		UsesYextImport:     usesYextImport,
	}, nil
}

// genProto3Msg takes an input Directory which describes a container or list entry
// within the YANG schema and returns a protoMsg which can be mapped to the protobuf
// code representing it. It uses the set of messages that have been extracted and the
// current generator state to map to other messages and ensure uniqueness of names.
// The configuration parameters for the current code generation required are supplied
// as a protoMsgConfig struct. The parentPkg argument specifies the name of the parent
// package for the protobuf message(s) that are being generated, such that relative
// paths can be used in the messages.
func genProto3Msg(msg *ygen.ParsedDirectory, ir *ygen.IR, cfg *protoMsgConfig, parentPkg string, childMsgs []*generatedProto3Message) ([]*protoMsg, util.Errors) {
	var errs util.Errors

	var msgDefs []*protoMsg

	msgDef := &protoMsg{
		// msg.name is already specified to be CamelCase in the form we expect it
		// to be for the protobuf message name.
		Name:      msg.Name,
		YANGPath:  msg.Path,
		Enums:     map[string]*protoMsgEnum{},
		ChildMsgs: childMsgs,
	}

	definedFieldNames := map[string]bool{}
	imports := map[string]interface{}{}

	var fNames []string
	for name := range msg.Fields {
		fNames = append(fNames, name)
	}
	sort.Strings(fNames)

	for _, name := range fNames {
		// Skip list key fields.
		if _, ok := msg.ListKeys[name]; ok {
			continue
		}

		field := msg.Fields[name]

		fieldDef := &protoMsgField{
			Name: genutil.MakeNameUnique(field.Name, definedFieldNames),
		}

		t, err := protoTagForEntry(field.YANGDetails)
		if err != nil {
			errs = append(errs, fmt.Errorf("proto: could not generate tag for field %s: %v", field.Name, err))
			continue
		}
		fieldDef.Tag = t

		defArgs := &protoDefinitionArgs{
			field:             field,
			directory:         msg,
			ir:                ir,
			definedFieldNames: definedFieldNames,
			cfg:               cfg,
			parentPkg:         parentPkg,
		}
		switch field.Type {
		case ygen.ListNode:
			keyMsg, listImports, listErrs := addProtoListField(fieldDef, msgDef, defArgs)
			if listErrs != nil {
				errs = append(errs, listErrs...)
				continue
			}
			addNewKeys(imports, listImports)
			if keyMsg != nil {
				msgDefs = append(msgDefs, keyMsg)
			}
		case ygen.ContainerNode:
			cImports, err := addProtoContainerField(fieldDef, defArgs)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			addNewKeys(imports, cImports)
		case ygen.LeafNode, ygen.LeafListNode:
			repeatedMsg, lImports, lErrs := addProtoLeafOrLeafListField(fieldDef, msgDef, defArgs)
			if lErrs != nil {
				errs = append(errs, lErrs...)
				continue
			}
			addNewKeys(imports, lImports)
			if repeatedMsg != nil {
				msgDefs = append(msgDefs, repeatedMsg)
			}
		case ygen.AnyDataNode:
			fieldDef.Type = protoAnyType
			imports[protoAnyPackage] = true
		default:
			err = fmt.Errorf("proto: unknown field type in message %s, field %s", msg.Name, field.Name)
		}

		if cfg.annotateSchemaPaths {
			o, err := protoSchemaPathAnnotation(msg, name, cfg.compressPaths)
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
	// field contains the node details for which the proto output is being
	// defined, in the case that the definition is for an individual entry.
	field *ygen.NodeDetails
	// directory is the ygen.ParsedDirectory for which the proto output is being
	// defined, in the case that the definition is for an directory entry.
	directory *ygen.ParsedDirectory
	// ir is the entirety of the IR as input to the code generation.
	ir *ygen.IR
	// definedFieldNames specifies the field names that have been defined in the context.
	definedFieldNames map[string]bool
	// cfg contains configuration options for proto generation.
	cfg *protoMsgConfig
	// parentPackage stores the name of the protobuf package that the field's parent is within.
	parentPkg string
}

// addProtoListField modifies the field definition in fieldDef (which must correspond to a list field of a
// YANG schema) to contain the definition of the field described by the args. In the case that the list is keyed
// and nested messages are being output, the generated protobuf message for the key is appended to the supplied
// message definition (msgDef). If nested messages are not being output, a definition of the key message is returned.
// Along with the optional key message, it returns a list of the imports being used for the list.
func addProtoListField(fieldDef *protoMsgField, msgDef *protoMsg, args *protoDefinitionArgs) (*protoMsg, []string, util.Errors) {
	listDef, keyMsg, err := protoListDefinition(args)
	if err != nil {
		return nil, nil, []error{fmt.Errorf("could not define list %s: %v", args.directory.Path, err)}
	}

	var nKeyMsg *protoMsg
	if keyMsg != nil {
		if args.cfg.nestedMessages {
			// If nested messages are being output, we must ensure that the
			// generated key message is output within the parent message - hence
			// it is generated directly here and appended to the child messages.
			kc, cerrs := genProto3MsgCode(args.cfg, args.directory.PackageName, []*protoMsg{keyMsg}, false)
			if cerrs != nil {
				return nil, nil, cerrs
			}
			msgDef.ChildMsgs = append(msgDef.ChildMsgs, kc)
		} else {
			nKeyMsg = keyMsg
		}
	}

	fieldDef.Type = listDef.listType

	// Lists are always repeated fields.
	fieldDef.IsRepeated = true
	return nKeyMsg, listDef.imports, nil
}

// addProtoContainerField modifies the field definition in fieldDef (which must correspond to a container field of
// a YANG schema) to contain the definition of the field described by the args. It returns a slice of strings containing
// the protobuf package imports that are required for the container definition.
func addProtoContainerField(fieldDef *protoMsgField, args *protoDefinitionArgs) ([]string, error) {
	childmsg, ok := args.ir.Directories[args.field.YANGDetails.Path]
	if !ok {
		return nil, fmt.Errorf("proto: could not resolve %s into a defined struct", args.field.YANGDetails.Path)
	}

	imports := map[string]interface{}{}

	var pfx string
	if !(args.cfg.compressPaths && args.directory.IsFakeRoot) {
		childpkg := childmsg.PackageName
		// Add the import to the slice of imports if it is not already
		// there. This allows the message file to import the required
		// child packages.
		childpath := importPath(args.cfg.baseImportPath, args.cfg.basePackageName, childpkg)
		if imports[childpath] == nil {
			if !args.cfg.nestedMessages || args.directory.IsFakeRoot {
				imports[childpath] = true
			}
		}

		p, _ := stripPackagePrefix(args.parentPkg, childpkg)
		if !args.cfg.nestedMessages || args.directory.IsFakeRoot {
			pfx = fmt.Sprintf("%s.", p)
		}
	}
	fieldDef.Type = fmt.Sprintf("%s%s", pfx, childmsg.Name)
	return stringKeys(imports), nil
}

// addProtoLeafOrLeafListField modifies the field definition in fieldDef to contain a definition of the field that is
// described in the args. If the field corresponds to a leaf-list of unions and hence requires another message to be
// generated for it, it is appended to the message definition supplied (msgDef) when nested messages are being output,
// otherwise it is returned. In addition, it returns a slice of strings describing the imports that are required for
// the message.
func addProtoLeafOrLeafListField(fieldDef *protoMsgField, msgDef *protoMsg, args *protoDefinitionArgs) (*protoMsg, []string, util.Errors) {
	var imports []string
	var repeatedMsg *protoMsg

	d, err := protoLeafDefinition(fieldDef.Name, args)
	if err != nil {
		return nil, nil, []error{fmt.Errorf("could not define field %s: %v", args.field.YANGDetails.Path, err)}
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
		if args.cfg.nestedMessages {
			gm, errs := genProto3MsgCode(args.cfg, args.parentPkg, []*protoMsg{d.repeatedMsg}, false)
			if err != nil {
				return nil, nil, errs
			}
			msgDef.ChildMsgs = append(msgDef.ChildMsgs, gm)
		} else {
			repeatedMsg = d.repeatedMsg
		}
	}

	// Add the global enumeration package if it is referenced by this field.
	if d.globalEnum {
		imports = append(imports, importPath(args.cfg.baseImportPath, args.cfg.basePackageName, args.cfg.enumPackageName))
	}

	if args.field.Type == ygen.LeafListNode {
		fieldDef.IsRepeated = true
		switch d.repeatedMsg {
		case nil:
			fieldDef.Options = append(fieldDef.Options, &protoOption{
				Name:  protoLeafListAnnotationOption,
				Value: "true",
			})
		default:
			fieldDef.Options = append(fieldDef.Options, &protoOption{
				Name:  protoLeafListUnionAnnotationOption,
				Value: "true",
			})
		}

	}
	return repeatedMsg, imports, nil
}

// writeProtoEnums takes a map of enumerated types within the YANG schema and
// returns the mapped Protobuf enum definition corresponding to each type. If
// the annotateEnumNames bool is set, then the original enum value label is
// stored in the definition. Since leaves that are of type enumeration are
// output directly within a Protobuf message, these are skipped.
func writeProtoEnums(enums map[string]*ygen.EnumeratedYANGType, annotateEnumNames bool) ([]string, error) {
	var errs util.Errors
	var genEnums []string
	for _, enum := range enums {
		// Make the name of the enum upper case to follow Protobuf enum convention.
		p := &protoEnum{Name: enum.Name}

		switch enum.Kind {
		case ygen.SimpleEnumerationType, ygen.UnionEnumerationType:
			// Skip simple enumerations and those within unions.
			continue
		case ygen.IdentityType:
			// For an identityref the values are based on
			// the name of the identities that correspond with the base, and the value
			// is gleaned from the YANG schema.
			values := map[int64]protoEnumValue{
				0: {ProtoLabel: protoEnumZeroName},
			}

			for _, enumDef := range enum.ValToYANGDetails {
				// Calculate a tag value for the identity values, since otherwise when another
				// module augments this module then the enum values may be subject to change.
				tag, err := fieldTag(fmt.Sprintf("%s%s", enum.IdentityBaseName, enumDef.Name))
				if err != nil {
					errs = append(errs, fmt.Errorf("cannot calculate tag for %s: %v", enumDef.Name, err))
				}

				// Names are converted to upper case to follow the protobuf style guide.
				values[int64(tag)] = toProtoEnumValue(safeProtoIdentifierName(enumDef.Name), enumDef.Name, annotateEnumNames)
			}
			p.Values = values
			p.ValuePrefix = strings.ToUpper(enum.Name)
			p.Description = fmt.Sprintf("YANG identity %s", enum.IdentityBaseName)
		case ygen.DerivedEnumerationType, ygen.DerivedUnionEnumerationType:
			ge, err := genProtoEnum(enum, annotateEnumNames, true)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			p.Values = ge.Values

			// Capitalize name per proto style.
			p.ValuePrefix = strings.ToUpper(enum.Name)
			p.Description = fmt.Sprintf("YANG enumerated type %s", enum.TypeName)
		default:
			errs = append(errs, fmt.Errorf("unknown type of enumerated value in writeProtoEnums for %s, got: %v, kind: %v", enum.Name, enum, enum.Kind))
		}

		var b bytes.Buffer
		if err := protoEnumTemplate.Execute(&b, p); err != nil {
			errs = append(errs, fmt.Errorf("cannot generate enumeration for %s: %v", enum.Name, err))
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
func genProtoEnum(enum *ygen.EnumeratedYANGType, annotateEnumNames, isLeafOrTypedef bool) (*protoMsgEnum, error) {
	eval := map[int64]protoEnumValue{}
	eval[0] = protoEnumValue{ProtoLabel: protoEnumZeroName}

	for _, enumDef := range enum.ValToYANGDetails {
		if isLeafOrTypedef && enumDef.Name == enum.TypeDefaultValue {
			// Can't happen if there was not a default, since "" is not
			// a valid enumeration name in YANG.
			eval[0] = toProtoEnumValue(safeProtoIdentifierName(enum.TypeDefaultValue), enum.TypeDefaultValue, annotateEnumNames)
			continue
		}
		// Names are converted to upper case to follow the protobuf style guide,
		// adding one to ensure that the 0 value can represent unused values.
		eval[int64(enumDef.Value)+1] = toProtoEnumValue(safeProtoIdentifierName(enumDef.Name), enumDef.Name, annotateEnumNames)
	}

	return &protoMsgEnum{Values: eval}, nil
}

// protoMsgListField describes a list field within a protobuf mesage.
type protoMsgListField struct {
	listType string   // listType is the name of the message that represents a list member.
	imports  []string // imports is the set of modules that are required by this list message.
}

// protoListDefinition takes an input field described by a yang.Entry, the generator context (the set of proto messages, and the generator
// state), along with whether path compression is enabled and generates the proto message definition for the list. It returns the definition
// of the field representing the list as a protoMsgListField and an optional message which stores the key of a keyed list.
func protoListDefinition(args *protoDefinitionArgs) (*protoMsgListField, *protoMsg, error) {
	listMsg, ok := args.ir.Directories[args.field.YANGDetails.Path]
	if !ok {
		return nil, nil, fmt.Errorf("proto: could not resolve list %s into a defined message", args.field.YANGDetails.Path)
	}

	listMsgName := listMsg.Name
	childPkg := listMsg.PackageName

	var listKeyMsg *protoMsg
	var listDef *protoMsgListField
	if len(listMsg.ListKeys) == 0 {
		// In proto3 we represent unkeyed lists as a
		// repeated field of the list message.
		listDef = &protoMsgListField{
			listType: listMsgName,
		}
		if !args.cfg.nestedMessages {
			p := fmt.Sprintf("%s.%s.%s", args.cfg.basePackageName, childPkg, listMsgName)
			p, _ = stripPackagePrefix(fmt.Sprintf("%s.%s", args.cfg.basePackageName, args.parentPkg), p)
			listDef = &protoMsgListField{
				listType: p,
			}
			listDef.imports = []string{importPath(args.cfg.baseImportPath, args.cfg.basePackageName, childPkg)}
		}
	} else {
		// YANG lists are mapped to a repeated message structure as described
		// in the YANG to Protobuf transformation specification.
		var err error
		listKeyMsg, err = genListKeyProto(childPkg, listMsgName, &protoDefinitionArgs{
			field:     args.field,
			directory: listMsg,
			ir:        args.ir,
			cfg:       args.cfg,
			parentPkg: args.parentPkg,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("proto: could not build mapping for list entry %s: %v", args.field.YANGDetails.Path, err)
		}
		// The type of this field is just the key message's name, since it
		// will be in the same package as the field's parent.
		listDef = &protoMsgListField{
			listType: listKeyMsg.Name,
		}
	}

	return listDef, listKeyMsg, nil
}

// protoDefinedLeaf defines a YANG leaf within a protobuf message.
type protoDefinedLeaf struct {
	protoType       string                   // protoType is the protobuf type that the leaf should be mapped to.
	globalEnum      bool                     // globalEnum indicates whether the leaf's type is a global scope enumeration (identityref, or typedef defining an enumeration)
	enums           map[string]*protoMsgEnum // enums defines the set of enumerated values that are required for this leaf within the parent message.
	oneofs          []*protoMsgField         // oneofs defines the set of types within the leaf, if the returned leaf type is a protobuf oneof.
	repeatedMsg     *protoMsg                // repeatedMsgs returns a message that should be repeated for this leaf, used in the case of a leaf-list of unions.
	isLeafList      bool
	isLeafListUnion bool
}

// protoLeafDefinition takes an input leafName, and a set of protoDefinitionArgs specifying the context
// for the leaf definition, and returns a protoDefinedLeaf describing how it is to be mapped within the
// protobuf parent message.
func protoLeafDefinition(leafName string, args *protoDefinitionArgs) (*protoDefinedLeaf, error) {
	protoType := args.field.LangType

	d := &protoDefinedLeaf{
		protoType: protoType.NativeType,
		enums:     map[string]*protoMsgEnum{},
	}

	var enum *ygen.EnumeratedYANGType
	if protoType.IsEnumeratedValue {
		enum = args.ir.Enums[protoType.EnumeratedYANGTypeKey]
	}

	switch {
	case protoType.IsEnumeratedValue && enum.Kind == ygen.SimpleEnumerationType:
		// For fields that are simple enumerations within a message, then we embed an enumeration
		// within the Protobuf message.
		e, err := genProtoEnum(enum, args.cfg.annotateEnumNames, args.field.Type == ygen.LeafNode)
		if err != nil {
			return nil, err
		}

		d.protoType = genutil.MakeNameUnique(protoType.NativeType, args.definedFieldNames)
		d.enums = map[string]*protoMsgEnum{}
		d.enums[d.protoType] = e
	case protoType.IsEnumeratedValue:
		d.globalEnum = true
	case protoType.UnionTypes != nil:
		u, err := unionFieldToOneOf(leafName, args.field, args.field.YANGDetails.Path, protoType, args.ir.Enums, args.cfg.annotateEnumNames, args.cfg.annotateSchemaPaths)
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
			d.isLeafListUnion = true
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
	// Therefore we need to replace all characters in the YANG identifier that are not a
	// letter, digit, or underscore.
	return disallowedInProtoIDRegexp.ReplaceAllLiteralString(name, "_")
}

// protoTagForEntry returns a protobuf tag value for the entry e.
func protoTagForEntry(n ygen.YANGNodeDetails) (uint32, error) {
	return fieldTag(n.Path)
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
// the key of a list for YANG lists. It takes a Directory pointer to the list being
// described, the name of the list, the package name that the list is within, and the
// current generator state. It returns the definition of the list key proto.
func genListKeyProto(listPackage string, listName string, args *protoDefinitionArgs) (*protoMsg, error) {
	n := fmt.Sprintf("%s%s", listName, protoListKeyMessageSuffix)
	km := &protoMsg{
		Name:     n,
		YANGPath: args.field.YANGDetails.Path,
		Enums:    map[string]*protoMsgEnum{},
	}

	if listPackage != "" {
		km.Imports = []string{importPath(args.cfg.baseImportPath, args.cfg.basePackageName, listPackage)}
	}

	definedFieldNames := map[string]bool{}
	ctag := uint32(1)
	// unionSubtypePaths keeps track of union keys such that if two keys point
	// to the same union entry, such a conflict when creating field tags
	// for them can be detected to avoid a tag collision.
	unionSubtypePaths := map[string]bool{}
	//for _, k := range strings.Fields(args.field.Key) {
	for _, k := range args.directory.OrderedListKeyNames() {
		scalarType := args.directory.ListKeys[k].LangType
		kf, ok := args.directory.Fields[k]
		if !ok {
			return nil, fmt.Errorf("list %s included a key %s that did not exist", args.field.YANGDetails.Path, k)
		}
		fieldName := kf.Name

		// Make the name of the key unique. We handle the case that the list name
		// matches the key field name by appending the protoMatchingListNameKeySuffix
		// to the field name, as described in the definition of protoMatchingListNameKeySuffix.
		fName := genutil.MakeNameUnique(fieldName, definedFieldNames)
		if args.field.Name == fieldName {
			fName = fmt.Sprintf("%s_%s", fName, protoMatchingListNameKeySuffix)
		}

		fd := &protoMsgField{
			Name: fName,
			Tag:  ctag,
		}

		var enum *ygen.EnumeratedYANGType
		if scalarType.IsEnumeratedValue {
			enum = args.ir.Enums[scalarType.EnumeratedYANGTypeKey]
		}
		switch {
		case scalarType.IsEnumeratedValue && enum.Kind == ygen.IdentityType:
			km.Imports = append(km.Imports, importPath(args.cfg.baseImportPath, args.cfg.basePackageName, args.cfg.enumPackageName))
			fd.Type = scalarType.NativeType
		case scalarType.IsEnumeratedValue:
			// list keys must be leafs and not leaf-lists.
			e, err := genProtoEnum(enum, args.cfg.annotateEnumNames, true)
			if err != nil {
				return nil, fmt.Errorf("error generating type for list %s key %s, type %v", args.field.YANGDetails.Path, k, enum.Kind)
			}
			tn := genutil.MakeNameUnique(scalarType.NativeType, definedFieldNames)
			fd.Type = tn
			km.Enums[tn] = e
		case scalarType.UnionTypes != nil:
			fd.IsOneOf = true
			path := kf.YANGDetails.LeafrefTargetPath
			if path != "" && !unionSubtypePaths[path] {
				unionSubtypePaths[path] = true
			} else {
				// It is possible for two keys to point to the same resolved unionEntry.
				// In this case, the path we use to generate the proto tag numbers needs
				// to be different to avoid a collision, and here we use the path of the
				// (leafref) key field. The reason the first instance uses the resolved
				// unionEntry is for backwards compatibility
				// (https://github.com/openconfig/ygot/pull/610#discussion_r781510037).
				path = kf.YANGDetails.Path
			}
			u, err := unionFieldToOneOf(fd.Name, kf, path, scalarType, args.ir.Enums, args.cfg.annotateEnumNames, args.cfg.annotateSchemaPaths)
			if err != nil {
				return nil, fmt.Errorf("error generating type for union list key %s in list %s", k, args.field.YANGDetails.Path)
			}
			fd.OneOfFields = append(fd.OneOfFields, u.oneOfFields...)
			for n, e := range u.enums {
				km.Enums[n] = e
			}
			if u.hadGlobalEnums {
				km.Imports = append(km.Imports, importPath(args.cfg.baseImportPath, args.cfg.basePackageName, args.cfg.enumPackageName))
			}
		default:
			fd.Type = scalarType.NativeType
		}

		if args.cfg.annotateSchemaPaths {
			o, err := protoSchemaPathAnnotation(args.directory, k, args.cfg.compressPaths)
			if err != nil {
				return nil, err
			}
			fd.Options = append(fd.Options, o)
		}

		km.Fields = append(km.Fields, fd)
		ctag++
	}

	// When using nested messages since the protobuf resolution rules mean that
	// the parent scope is searched, then we do not need to qualify the name of
	// the list message, even though it is in the parent's namespace.
	ltype := listName
	if !args.cfg.nestedMessages {
		p, _ := stripPackagePrefix(args.parentPkg, listPackage)
		ltype = fmt.Sprintf("%s.%s", p, listName)
		if listPackage == "" {
			// Handle the case that the context of the list is already the base package.
			ltype = listName
		}
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
func enumInProtoUnionField(name string, field *ygen.NodeDetails, Enums map[string]*ygen.EnumeratedYANGType, annotateEnumNames bool) (map[string]*protoMsgEnum, error) {
	enums := map[string]*protoMsgEnum{}
	for genName, subtype := range field.LangType.UnionTypes {
		if subtype.EnumeratedYANGTypeKey == "" {
			continue
		}
		enum, ok := Enums[subtype.EnumeratedYANGTypeKey]
		if !ok {
			return nil, fmt.Errorf("enumerated type within union %s not found in IR, field path: %s", genName, field.YANGDetails.Path)
		}
		switch enum.Kind {
		case ygen.SimpleEnumerationType, ygen.UnionEnumerationType:
			protoEnum, err := genProtoEnum(enum, annotateEnumNames, field.Type == ygen.LeafNode)
			if err != nil {
				return nil, err
			}
			enums[genName] = protoEnum
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

// unionFieldToOneOf takes an input name, a yang.Entry containing a field
// definition, a path argument used to compute the field tag numbers, and a ygen.MappedType
// containing the proto type that the entry has been mapped to, and returns a definition of a union
// field within the protobuf message. If the annotateEnumNames boolean is set, then any enumerated types
// within the union have their original names within the YANG schema appended.
func unionFieldToOneOf(fieldName string, field *ygen.NodeDetails, path string, mtype *ygen.MappedType, Enums map[string]*ygen.EnumeratedYANGType, annotateEnumNames, annotateSchemaPaths bool) (*protoUnionField, error) {
	enums, err := enumInProtoUnionField(fieldName, field, Enums, annotateEnumNames)
	if err != nil {
		return nil, err
	}

	var typeNames []string
	for tn := range mtype.UnionTypes {
		typeNames = append(typeNames, tn)
	}
	sort.Strings(typeNames)

	var importGlobalEnums bool
	var oofs []*protoMsgField
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
		ft, err := fieldTag(fmt.Sprintf("%s_%s", path, strings.ToLower(tn)))
		if err != nil {
			return nil, fmt.Errorf("could not calculate tag number for %s, type %s in oneof", field.YANGDetails.Path, tn)
		}
		st := &protoMsgField{
			Name: fmt.Sprintf("%s_%s", fieldName, strings.ToLower(tn)),
			Type: t,
			Tag:  ft,
		}

		if annotateSchemaPaths {
			st.Options = append(st.Options, protoFieldSchemaPathAnnotation(field.MappedPaths))
		}

		oofs = append(oofs, st)
	}

	if field.Type == ygen.LeafListNode {
		// In this case, we cannot return a oneof, since it is not possible to have a repeated
		// oneof, therefore we return a message that contains the protoMsgFields that are defined
		// above.
		p := &protoMsg{
			Name:     fmt.Sprintf("%sUnion", yang.CamelCase(fieldName)),
			YANGPath: fmt.Sprintf("%s union field %s", field.YANGDetails.Path, field.YANGDetails.Name),
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
func protoSchemaPathAnnotation(msg *ygen.ParsedDirectory, fieldName string, compressPaths bool) (*protoOption, error) {
	// protobuf paths are always absolute.
	return protoFieldSchemaPathAnnotation(msg.Fields[fieldName].MappedPaths), nil
}

// protoSchemaPathAnnotation takes a specific protobuf set of paths, and returns
// the protobuf field option definitions required to annotate it with its schema path(s).
func protoFieldSchemaPathAnnotation(smapp [][]string) *protoOption {
	var b bytes.Buffer
	b.WriteRune('"')
	for i, p := range smapp {
		b.WriteString(util.SlicePathToString(p))
		if i != len(smapp)-1 {
			b.WriteString("|")
		}
	}
	b.WriteRune('"')
	return &protoOption{Name: protoSchemaAnnotationOption, Value: b.String()}
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
