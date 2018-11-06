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

// Package ygen contains a library to generate Go structs from a YANG model.
// The Goyang parsing library is used to parse YANG. The output can consider
// OpenConfig-specific conventions such that the schema is compressed.
package ygen

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	log "github.com/golang/glog"

	"github.com/openconfig/gnmi/ctree"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/util"
	"github.com/openconfig/ygot/ygot"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
)

// YANGCodeGenerator is a structure that is used to pass arguments as to
// how the output Go code should be generated.
type YANGCodeGenerator struct {
	// Config stores the configuration parameters used for code generation.
	Config GeneratorConfig
	// genState is used internally to the ygen library to store state for
	// code generation.
	state *genState
}

// GeneratorConfig stores the configuration options used for code generation.
type GeneratorConfig struct {
	// CompressOCPaths indicates whether paths should be compressed in the output
	// of an OpenConfig schema.
	CompressOCPaths bool
	// ExcludeModules specifies any modules that are included within the set of
	// modules that should have code generated for them that should be ignored during
	// code generation. This is due to the fact that some schemas (e.g., OpenConfig
	// interfaces) currently result in overlapping entities (e.g., /interfaces).
	ExcludeModules []string
	// PackageName is the name that should be used for the generating package.
	PackageName string
	// Caller is the name of the binary calling the generator library, it is
	// included in the header of output files for debugging purposes. If a
	// string is not specified, the location of the library is utilised.
	Caller string
	// YANGParseOptions provides the options that should be handed to the
	// //third_party/golang/goyang/pkg/yang library. These specify how the
	// input YANG files should be parsed.
	YANGParseOptions yang.Options
	// GenerateFakeRoot specifies whether an entity that represents the
	// root of the YANG schema tree should be generated in the generated
	// code.
	GenerateFakeRoot bool
	// FakeRootName specifies the name of the struct that should be generated
	// representing the root.
	FakeRootName string
	// GenerateJSONSchema stores a boolean which defines whether to generate
	// the JSON corresponding to the YANG schema parsed to generate the
	// output code.
	GenerateJSONSchema bool
	// StoreRawSchema the raw JSON schema should be returned by the code
	// generation function, such that it can be handled by an external
	// library.
	StoreRawSchema bool
	// GoOptions stores a struct which stores Go code generation specific
	// options for the code generaton.
	GoOptions GoOpts
	// ProtoOptions stores a struct which contains Protobuf specific options.
	ProtoOptions ProtoOpts
	// ExcludeState specifies whether config false values should be
	// included in the generated code output. When set, all values that are
	// not writeable (i.e., config false) within the YANG schema and their
	// children are excluded from the generated code.
	ExcludeState bool
}

// GoOpts stores Go specific options for the code generation library.
type GoOpts struct {
	// SchemaVarName is the name for the variable which stores the compressed
	// JSON schema in the generated Go code. JSON schema output is only
	// produced if the GenerateJSONSchema YANGCodeGenerator field is set to
	// true.
	SchemaVarName string
	// GoyangImportPath specifies the path that should be used in the generated
	// code for importing the goyang/pkg/yang package.
	GoyangImportPath string
	// YgotImportPath specifies the path to the ygot library that should be used
	// in the generated code.
	YgotImportPath string
	// YtypesImportPath specifies the path to ytypes library that should be used
	// in the generated code.
	YtypesImportPath string
	// GenerateRenameMethod specifies whether methods for renaming list entries
	// should be generated in the output Go code.
	GenerateRenameMethod bool
	// AddAnnotationFields specifies whether annotation fields should be added to
	// the generated structs. When set to true, a metadata field is added for each
	// struct, and for each field of each struct. Metadata field's names are
	// prefixed by the string specified in the AnnotationPrefix argument.
	AddAnnotationFields bool
	// AnnotationPrefix specifies the string which is prefixed to the name of
	// annotation fields. It defaults to Λ.
	AnnotationPrefix string
	// GenerateGetters specifies whether GetOrCreate* methods should be created
	// for struct pointer (YANG container) and map (YANG list) fields of generated
	// structs.
	GenerateGetters bool
	// GenerateDeleteMethod specifies whether Delete* methods should be created for
	// map (YANG list) fields of generated structs.
	GenerateDeleteMethod bool
	// GenerateAppendList specifies whether Append* methods should be created for
	// list fields of a struct. These methods take an input list member type, extract
	// the key and append the supplied value to the list.
	GenerateAppendMethod bool
	// GenerateLeafGetters specifies whether Get* methods should be created for
	// leaf fields of a struct. Care should be taken with this option since a Get
	// method returns the *Go* zero value for a particular entity if the field is
	// unset. This means that it is not possible for a caller of method to know
	// whether a field has been explicitly set to the zero value (i.e., an integer
	// field is set to 0), or whether the field was actually unset.
	GenerateLeafGetters bool
	// GNMIProtoPath specifies the path to the generated gNMI protobuf, which
	// is used to store the catalogue entries for generated modules.
	GNMIProtoPath string
	// IncludeModelData specifies whether gNMI ModelData messages should be generated
	// in the output code.
	IncludeModelData bool
}

// ProtoOpts stores Protobuf specific options for the code generation library.
type ProtoOpts struct {
	// BaseImportPath stores the root URL or path for imports that are
	// relative within the imported protobufs.
	BaseImportPath string
	// EnumPackageName stores the package name that should be used
	// for the package that defines enumerated types that are used
	// in multiple parts of the schema (identityrefs, and enumerations)
	// that fall within type definitions.
	EnumPackageName string
	// YwrapperPath is the path to the ywrapper.proto file that stores
	// the definition of the wrapper messages used to ensure that unset
	// fields can be distinguished from those that are set to their
	// default value. The path excluds the filename.
	YwrapperPath string
	// YextPath is the path to the yext.proto file that stores the
	// definition of the extension messages that are used to annotat the
	// generated protobuf messages.
	YextPath string
	// AnnotateSchemaPaths specifies whether the extensions defined in
	// yext.proto should be used to annotate schema paths into the output
	// protobuf file. See
	// https://github.com/openconfig/ygot/blob/master/docs/yang-to-protobuf-transformations-spec.md#annotation-of-schema-paths
	AnnotateSchemaPaths bool
	// AnnotateEnumNames specifies whether the extensions defined in
	// yext.proto should be used to annotate enum values with their
	// original YANG names in the output protobuf file.
	// See https://github.com/openconfig/ygot/blob/master/docs/yang-to-protobuf-transformations-spec.md#annotation-of-enums
	AnnotateEnumNames bool
	// NestedMessages indicates whether nested messages should be
	// output for the protobuf schema. If false, a separate package
	// is generated per package.
	NestedMessages bool
}

// NewYANGCodeGenerator returns a new instance of the YANGCodeGenerator
// struct to the calling function.
func NewYANGCodeGenerator(c *GeneratorConfig) *YANGCodeGenerator {
	cg := &YANGCodeGenerator{
		state: newGenState(),
	}

	if c != nil {
		cg.Config = *c
	}

	return cg
}

// yangDirectory represents a directory entry that code is to be generated for. It stores the
// fields that are required to output the relevant code for the entity.
type yangDirectory struct {
	name       string                 // name is the name of the struct to be generated.
	entry      *yang.Entry            // entry is the yang.Entry that corresponds to the schema element being converted to a struct.
	fields     map[string]*yang.Entry // fields is a map, keyed by the YANG node identifier, of the entries that are the struct fields.
	path       []string               // path is a slice of strings indicating the element's path.
	listAttr   *yangListAttr          // listAttr is used to store characteristics of structs that represent YANG lists.
	isFakeRoot bool                   // isFakeRoot indicates that the struct is a fake root struct, so specific mapping rules should be implemented.
}

// isList returns true if the yangDirectory describes a list.
func (y *yangDirectory) isList() bool {
	return y.listAttr != nil
}

// yangListAttr is used to store the additional elements for a Go struct that
// are required if the struct represents a YANG list. It stores the name of
// the key elements, and their associated types, along with pointers to those
// elements.
type yangListAttr struct {
	// keys is a map, keyed by the name of the key leaf, with values of the type
	// of the key of a YANG list.
	keys map[string]*mappedType
	// keyElems is a slice containing the pointers to yang.Entry structs that
	// make up the list key.
	keyElems []*yang.Entry
}

// yangEnum represents an enumerated type in YANG that is to be output in the
// Go code. The enumerated type may be a YANG 'identity' or enumeration.
type yangEnum struct {
	name  string      // name is the name of the enumeration or identity.
	entry *yang.Entry // entry is the yang.Entry corresponding to the enumerated value.
}

// GeneratedGoCode contains generated code snippets that can be processed by the calling
// application. The generated code is divided into two types of objects - both represented
// as a slice of strings: Structs contains a set of Go structures that have been generated,
// and Enums contains the code for generated enumerated types (corresponding to identities,
// or enumerated values within the YANG models for which code is being generated). Additionally
// the header with package comment of the generated code is returned in Header, along with the
// a slice of strings containing the packages that are required for the generated Go code to
// be compiled is returned.
//
// For schemas that contain enumerated types (identities, or enumerations), a code snippet is
// returned as the EnumMap field that allows the string values from the YANG schema to be resolved.
// The keys of the map are strings corresponding to the name of the generated type, with the
// map values being maps of the int64 identifier for each value of the enumeration to the name of
// the element, as used in the YANG schema.
type GeneratedGoCode struct {
	Structs      []GoStructCodeSnippet // Structs is the generated set of structs representing containers or lists in the input YANG models.
	Enums        []string              // Enums is the generated set of enum definitions corresponding to identities and enumerations in the input YANG models.
	CommonHeader string                // CommonHeader is the header that should be used for all output Go files.
	OneOffHeader string                // OneOffHeader defines the header that should be included in only one output Go file - such as package init statements.
	EnumMap      string                // EnumMap is a Go map that allows the YANG string values of enumerated types to be resolved.
	// JSONSchemaCode contains code defining a variable storing a serialised JSON schema for the
	// generated Go structs. When deserialised it consists of a map[string]*yang.Entry. The
	// entries are the root level yang.Entry definitions along with their corresponding
	// hierarchy (i.e., the yang.Entry for /foo contains /foo/... - all of foo's descendents).
	// Each yang.Entry which corresponds to a generated Go struct has two extra fields defined:
	//  - schemapath - the path to this entry within the schema. This is provided since the Path() method of
	//                 the deserialised yang.Entry does not return the path since the Parent pointer is not
	//                 populated.
	//  - structname - the name of the struct that was generated for the schema element.
	JSONSchemaCode string
	// RawJSONSchema stores the JSON document which is serialised and stored in JSONSchemaCode.
	// It is populated only if the StoreRawSchema YANGCodeGenerator boolean is set to true.
	RawJSONSchema []byte
	// EnumTypeMap is a Go map that allows YANG schemapaths to be mapped to reflect.Type values.
	EnumTypeMap string
}

// GeneratedProto3 stores a set of generated Protobuf packages.
type GeneratedProto3 struct {
	// Packages stores a map, keyed by the Protobuf package name, and containing the contents of the protobuf3
	// messages defined within the package. The calling application can write out the defined packages to the
	// files expected by the protoc tool.
	Packages map[string]Proto3Package
}

// Proto3Package stores the code for a generated protobuf3 package.
type Proto3Package struct {
	FilePath []string // FilePath is the path to the file that this package should be written to.
	Header   string   // Header is the header text to be used in the package.
	Messages []string // Messages is a slice of strings containing the set of messages that are within the generated package.
	Enums    []string // Enums is a slice of string containing the generated set of enumerations within the package.
}

const (
	// rootElementPath is the synthesised node name that is used for an
	// element that represents the root. Such an element is generated only
	// when the GenerateFakeRoot bool is set to true within the
	// YANGCodeGenerator instance used as a receiver.
	rootElementNodeName = "!fakeroot!"
	// defaultRootName is the default name for the root structure if GenerateFakeRoot is
	// set to true.
	defaultRootName = "device"
)

// generatedLanguage represents a language supported in this package.
type generatedLanguage int64

const (
	// golang indicates that Go code is being generated.
	golang generatedLanguage = iota
	// protobuf indicates that Protobuf messages are being generated.
	protobuf
)

// GenerateGoCode takes a slice of strings containing the path to a set of YANG
// files which contain YANG modules, and a second slice of strings which
// specifies the set of paths that are to be searched for associated models (e.g.,
// modules that are included by the specified set of modules, or submodules of those
// modules). It extracts the set of modules that are to be generated, and returns
// a GeneratedGoCode struct which contains:
//	1. A struct definition for each container or list that is within the specified
//	    set of models.
//	2. Enumerated values which correspond to the set of enumerated entities (leaves
//	   of type enumeration, identities, typedefs that reference an enumeration)
//	   within the specified models.
// If errors are encountered during code generation, an error is returned.
func (cg *YANGCodeGenerator) GenerateGoCode(yangFiles, includePaths []string) (*GeneratedGoCode, util.Errors) {
	// Extract the entities to be mapped into structs and enumerations in the output
	// Go code. Extract the schematree from the modules provided such that it can be
	// used to reference entities within the tree.
	mdef, errs := mappedDefinitions(yangFiles, includePaths, &cg.Config)
	if errs != nil {
		return nil, errs
	}

	// Store the returned schematree within the state for this code generation.
	cg.state.schematree = mdef.schemaTree

	goStructs, errs := cg.state.buildDirectoryDefinitions(mdef.directoryEntries, cg.Config.CompressOCPaths, cg.Config.GenerateFakeRoot, golang, cg.Config.ExcludeState)
	if errs != nil {
		return nil, errs
	}

	var rootName string
	if rootName = resolveRootName(cg.Config.FakeRootName, defaultRootName, cg.Config.GenerateFakeRoot); rootName != "" {
		if r, ok := goStructs[fmt.Sprintf("/%s", rootName)]; ok {
			rootName = r.name
		}
	}

	commonHeader, oneoffHeader, err := writeGoHeader(yangFiles, includePaths, cg.Config, rootName, mdef.modelData)

	if err != nil {
		return nil, util.AppendErr(util.Errors{}, err)
	}

	// orderedStructNames is used to store the structs that have been
	// identified in alphabetical order, such that they are returned in a
	// deterministic order to the calling application. This ensures that if
	// the slice is simply output in order, then the diffs generated are
	// minimised (i.e., diffs are not generated simply due to reordering of
	// the maps used).
	var orderedStructNames []string
	structNameMap := make(map[string]*yangDirectory)
	for _, goStruct := range goStructs {
		orderedStructNames = append(orderedStructNames, goStruct.name)
		structNameMap[goStruct.name] = goStruct
	}
	sort.Strings(orderedStructNames)

	// enumTypeMap stores the map of the path to type.
	enumTypeMap := map[string][]string{}
	var codegenErr util.Errors
	var structSnippets []GoStructCodeSnippet
	for _, structName := range orderedStructNames {
		structOut, errs := writeGoStruct(structNameMap[structName], goStructs, cg.state,
			cg.Config.CompressOCPaths, cg.Config.GenerateJSONSchema, cg.Config.GoOptions)
		if errs != nil {
			codegenErr = util.AppendErrs(codegenErr, errs)
			continue
		}
		structSnippets = append(structSnippets, structOut)

		// Copy the contents of the enumTypeMap for the struct into the global
		// map.
		for p, t := range structOut.enumTypeMap {
			enumTypeMap[p] = t
		}
	}

	goEnums, errs := cg.state.findEnumSet(mdef.enumEntries, cg.Config.CompressOCPaths, false)
	if errs != nil {
		codegenErr = util.AppendErrs(codegenErr, errs)
		return nil, codegenErr
	}

	// orderedEnumNames is used to store the enumerated types that have been
	// identified in alphabetical order, such that they are returned in a
	// deterministic order to the calling application. This ensures that
	// the diffs are minimised, similarly to the use of orderedStructNames
	// above.
	var orderedEnumNames []string
	enumNameMap := make(map[string]*yangEnum)
	for _, goEnum := range goEnums {
		orderedEnumNames = append(orderedEnumNames, goEnum.name)
		enumNameMap[goEnum.name] = goEnum
	}
	sort.Strings(orderedEnumNames)

	var enumSnippets []string
	// enumValueMap is used to store a map of the different enumerations
	// that are included in the generated code. It is keyed by the name
	// of the generated enumeration type, with the values being a map,
	// keyed by value number to the string that is used in the YANG schema
	// for the enumeration. The value number is an int64 which is the value
	// of the constant that represents the enumeration type.
	enumValueMap := map[string]map[int64]ygot.EnumDefinition{}
	for _, enumName := range orderedEnumNames {
		enumOut, err := writeGoEnum(enumNameMap[enumName])
		if err != nil {
			util.AppendErr(codegenErr, err)
			continue
		}
		enumSnippets = append(enumSnippets, enumOut.constDef)
		enumValueMap[enumOut.name] = enumOut.valToString
	}

	// Generate the constant map which provides mappings between the
	// enums for which code was generated and their corresponding
	// string values.
	enumMap, err := generateEnumMap(enumValueMap)
	if err != nil {
		util.AppendErr(codegenErr, err)
	}

	var rawSchema []byte
	var jsonSchema string
	var enumTypeMapCode string
	if cg.Config.GenerateJSONSchema {
		var err error
		rawSchema, err = buildJSONTree(mdef.modules, cg.state.uniqueDirectoryNames, mdef.directoryEntries["/"], cg.Config.CompressOCPaths)
		if err != nil {
			util.AppendErr(codegenErr, fmt.Errorf("error marshalling JSON schema: %v", err))
		}

		if rawSchema != nil {
			if jsonSchema, err = writeGoSchema(rawSchema, cg.Config.GoOptions.SchemaVarName); err != nil {
				util.AppendErr(codegenErr, err)
			}
		}

		if enumTypeMapCode, err = generateEnumTypeMap(enumTypeMap); err != nil {
			util.AppendErr(codegenErr, err)
		}
	}

	// Return any errors that were encountered during code generation.
	if len(codegenErr) != 0 {
		return nil, codegenErr
	}

	return &GeneratedGoCode{
		CommonHeader:   commonHeader,
		OneOffHeader:   oneoffHeader,
		Structs:        structSnippets,
		Enums:          enumSnippets,
		EnumMap:        enumMap,
		JSONSchemaCode: jsonSchema,
		RawJSONSchema:  rawSchema,
		EnumTypeMap:    enumTypeMapCode,
	}, nil
}

// GenerateProto3 generates Protobuf 3 code for the input set of YANG files.
// The YANG schemas for which protobufs are to be created is supplied as the
// yangFiles argument, with included modules being searched for in includePaths.
// It returns a GeneratedProto3 struct containing the messages that are to be
// output, along with any associated values (e.g., enumerations).
func (cg *YANGCodeGenerator) GenerateProto3(yangFiles, includePaths []string) (*GeneratedProto3, util.Errors) {
	mdef, errs := mappedDefinitions(yangFiles, includePaths, &cg.Config)
	if errs != nil {
		return nil, errs
	}

	cg.state.schematree = mdef.schemaTree

	penums, errs := cg.state.findEnumSet(mdef.enumEntries, cg.Config.CompressOCPaths, true)
	if errs != nil {
		return nil, errs
	}
	protoEnums, errs := writeProtoEnums(penums, cg.Config.ProtoOptions.AnnotateEnumNames)
	if errs != nil {
		return nil, errs
	}

	protoMsgs, errs := cg.state.buildDirectoryDefinitions(mdef.directoryEntries, cg.Config.CompressOCPaths, cg.Config.GenerateFakeRoot, protobuf, cg.Config.ExcludeState)
	if errs != nil {
		return nil, errs
	}

	genProto := &GeneratedProto3{
		Packages: map[string]Proto3Package{},
	}

	// yerr stores errors encountered during code generation.
	var yerr util.Errors

	// pkgImports lists the imports that are required for the package that is being
	// written out.
	pkgImports := map[string]map[string]interface{}{}

	// Ensure that the slice of messages returned is in a deterministic order by
	// sorting the message paths. We use the path rather than the name as the
	// proto message name may not be unique.
	msgPaths := []string{}
	msgMap := map[string]*yangDirectory{}
	for _, m := range protoMsgs {
		k := strings.Join(m.path, "/")
		msgPaths = append(msgPaths, k)
		msgMap[k] = m
	}
	sort.Strings(msgPaths)

	basePackageName := cg.Config.PackageName
	if basePackageName == "" {
		basePackageName = DefaultBasePackageName
	}
	enumPackageName := cg.Config.ProtoOptions.EnumPackageName
	if enumPackageName == "" {
		enumPackageName = DefaultEnumPackageName
	}
	ywrapperPath := cg.Config.ProtoOptions.YwrapperPath
	if ywrapperPath == "" {
		ywrapperPath = DefaultYwrapperPath
	}
	yextPath := cg.Config.ProtoOptions.YextPath
	if yextPath == "" {
		yextPath = DefaultYextPath
	}

	// Only create the enums package if there are enums that are within the schema.
	if len(protoEnums) > 0 {
		// Sort the set of enumerations so that they are deterministically output.
		sort.Strings(protoEnums)
		fp := []string{basePackageName, enumPackageName, fmt.Sprintf("%s.proto", enumPackageName)}
		genProto.Packages[fmt.Sprintf("%s.%s", basePackageName, enumPackageName)] = Proto3Package{
			FilePath: fp,
			Enums:    protoEnums,
		}
	}

	for _, n := range msgPaths {
		m := msgMap[n]

		genMsg, errs := writeProto3Msg(m, protoMsgs, cg.state, &protoMsgConfig{
			compressPaths:       cg.Config.CompressOCPaths,
			basePackageName:     basePackageName,
			enumPackageName:     enumPackageName,
			baseImportPath:      cg.Config.ProtoOptions.BaseImportPath,
			annotateSchemaPaths: cg.Config.ProtoOptions.AnnotateSchemaPaths,
			annotateEnumNames:   cg.Config.ProtoOptions.AnnotateEnumNames,
			nestedMessages:      cg.Config.ProtoOptions.NestedMessages,
		})

		if errs != nil {
			yerr = util.AppendErrs(yerr, errs)
			continue
		}

		// Check whether any messages were required for this schema element, writeProto3Msg can
		// return nil if nested messages were being produced, and the message was encapsulated
		// in another message.
		if genMsg == nil {
			continue
		}

		if genMsg.PackageName == "" {
			genMsg.PackageName = basePackageName
		} else {
			genMsg.PackageName = fmt.Sprintf("%s.%s", basePackageName, genMsg.PackageName)
		}

		if pkgImports[genMsg.PackageName] == nil {
			pkgImports[genMsg.PackageName] = map[string]interface{}{}
		}
		addNewKeys(pkgImports[genMsg.PackageName], genMsg.RequiredImports)

		// If the package does not already exist within the generated proto3
		// output, then create it within the package map. This allows different
		// entries in the msgNames set to fall within the same package.
		tp, ok := genProto.Packages[genMsg.PackageName]
		if !ok {
			genProto.Packages[genMsg.PackageName] = Proto3Package{
				FilePath: protoPackageToFilePath(genMsg.PackageName),
				Messages: []string{},
			}
			tp = genProto.Packages[genMsg.PackageName]
		}
		tp.Messages = append(tp.Messages, genMsg.MessageCode)
		genProto.Packages[genMsg.PackageName] = tp
	}

	for n, pkg := range genProto.Packages {
		h, err := writeProto3Header(proto3Header{
			PackageName:            n,
			Imports:                stringKeys(pkgImports[n]),
			SourceYANGFiles:        yangFiles,
			SourceYANGIncludePaths: includePaths,
			CompressPaths:          cg.Config.CompressOCPaths,
			CallerName:             cg.Config.Caller,
			YwrapperPath:           ywrapperPath,
			YextPath:               yextPath,
		})
		if err != nil {
			yerr = util.AppendErrs(yerr, errs)
			continue
		}
		pkg.Header = h
		genProto.Packages[n] = pkg
	}

	if yerr != nil {
		return nil, yerr
	}

	return genProto, nil
}

// processModules takes a list of the filenames of YANG modules (yangFiles),
// and a list of paths in which included modules or submodules may be found,
// and returns a processed set of yang.Entry pointers which correspond to the
// generated code for the modules. If errors are returned during the Goyang
// processing of the modules, these errors are returned.
func processModules(yangFiles, includePaths []string, options yang.Options) ([]*yang.Entry, util.Errors) {
	// Append the includePaths to the Goyang path variable, this ensures
	// that where a YANG module uses an 'include' statement to reference
	// another module, then Goyang can find this module to process.
	for _, path := range includePaths {
		yang.AddPath(path)
	}

	// Propagate the options for the YANG library through to the parsing
	// code - this allows the calling binary to specify characteristics
	// of the YANG in a manner that we are transparent to.
	yang.ParseOptions = options

	// Initialise the set of YANG modules within the Goyang parsing package.
	moduleSet := yang.NewModules()

	var errs util.Errors
	for _, name := range yangFiles {
		errs = util.AppendErr(errs, moduleSet.Read(name))
	}

	if errs != nil {
		return nil, errs
	}

	if errs := moduleSet.Process(); errs != nil {
		return nil, errs
	}

	// Build the unique set of modules that are to be processed.
	var modNames []string
	mods := make(map[string]*yang.Module)
	for _, m := range moduleSet.Modules {
		if mods[m.Name] == nil {
			mods[m.Name] = m
			modNames = append(modNames, m.Name)
		}
	}

	// Process the ASTs that have been generated for the modules using the Goyang ToEntry
	// routines.
	entries := []*yang.Entry{}
	for _, modName := range modNames {
		entries = append(entries, yang.ToEntry(mods[modName]))
	}
	return entries, nil
}

// mappedYANGDefinitions stores the entities extracted from a YANG schema that are to be mapped to
// entities within the generated code, or can be used to look up entities within the YANG schema.
type mappedYANGDefinitions struct {
	// directoryEntries is the set of entities that are to be mapped to directories (e.g.,
	// Go structs, proto messages) in the generated code. The map is keyed by the string
	// path to the directory in the YANG schema.
	directoryEntries map[string]*yang.Entry
	// enumEntries is the set of entities that contain an enumerated type within the input
	// YANG and are to be mapped to enumerated types in the output code. This consists of
	// leaves that are of type enumeration, identityref, or unions that contain either of
	// these types. The map is keyed by the string path to the entry in the YANG schema.
	enumEntries map[string]*yang.Entry
	// schemaTree is a ctree.Tree that stores a copy of the YANG schema tree, containing
	// only leaf entries, such that schema paths can be referenced.
	schemaTree *ctree.Tree
	// modules is the set of parsed YANG modules that are being processed as part of the
	// code generatio, expressed as a slice of yang.Entry pointers.
	modules []*yang.Entry
	// modelData stores the details of the set of modules that were parsed to produce
	// the code. It is optionally returned in the generated code.
	modelData []*gpb.ModelData
}

// mappedDefinitions find the set of directory and enumeration entities
// that are mapped to objects within output code in a language agnostic manner.
// It takes:
//	- yangFiles: an input set of YANG schema files and the paths that
//	- includePaths: the set of paths that are to be searched for included or
//	  imported YANG modules.
//	- cfg: the current generator's configuration.
// It returns a mappedYANGDefinitions struct populated with the directory and enum
// entries in the input schemas, along with the calculated schema tree.
func mappedDefinitions(yangFiles, includePaths []string, cfg *GeneratorConfig) (*mappedYANGDefinitions, util.Errors) {
	modules, errs := processModules(yangFiles, includePaths, cfg.YANGParseOptions)
	if errs != nil {
		return nil, errs
	}

	// Build a map of excluded modules to simplify lookup.
	excluded := map[string]bool{}
	for _, e := range cfg.ExcludeModules {
		excluded[e] = true
	}

	// Extract the entities that are eligible to have code generated for
	// them from the modules that are provided as an argument.
	dirs := map[string]*yang.Entry{}
	enums := map[string]*yang.Entry{}
	var rootElems, treeElems []*yang.Entry
	for _, module := range modules {
		errs = append(errs, findMappableEntities(module, dirs, enums, cfg.ExcludeModules, cfg.CompressOCPaths, modules)...)
		if module == nil {
			errs = append(errs, errors.New("found a nil module in the returned module set"))
			continue
		}

		for _, e := range module.Dir {
			if !excluded[module.Name] {
				rootElems = append(rootElems, e)
			}
			treeElems = append(treeElems, e)
		}
	}
	if errs != nil {
		return nil, errs
	}

	// Build the schematree for the modules provided - we build for all of the
	// root elements, since we might need to reference a part of the schema that
	// we are not outputting for leafref lookups.
	st, err := buildSchemaTree(treeElems)
	if err != nil {
		return nil, []error{err}
	}

	// If we were asked to generate a fake root entity, then go and find the top-level entities that
	// we were asked for.
	if cfg.GenerateFakeRoot {
		if err := createFakeRoot(dirs, rootElems, cfg.FakeRootName, cfg.CompressOCPaths); err != nil {
			return nil, []error{err}
		}
	}

	// For all non-excluded modules, we store these to be
	// used as the schema tree.
	ms := []*yang.Entry{}
	for _, m := range modules {
		if _, ok := excluded[m.Name]; !ok {
			ms = append(ms, m)
		}
	}

	modelData, err := findModelData(modules)
	if err != nil {
		return nil, util.NewErrs(fmt.Errorf("cannot extract model data, %v", err))
	}

	return &mappedYANGDefinitions{
		directoryEntries: dirs,
		enumEntries:      enums,
		schemaTree:       st,
		modules:          ms,
		modelData:        modelData,
	}, nil
}

// mappableLeaf determines whether the yang.Entry e is leaf with an
// enumerated value, such that the referenced enumerated type (enumeration or
// identity) should have code generated for it. If it is an enumerated type
// the leaf is returned.
func mappableLeaf(e *yang.Entry) *yang.Entry {
	if e.Type == nil {
		// If the type of the leaf is nil, then this is not a valid
		// leaf within the schema - since goyang must populate the
		// entry Type.
		// TODO(robjs): Add this as an error case that can be handled by
		// the caller directly.
		log.Warningf("got unexpected nil value type for leaf %s (%s), entry: %v", e.Name, e.Path(), e)
		return nil
	}

	var types []*yang.YangType
	switch {
	case isEnumType(e.Type):
		// Handle the case that this leaf is an enumeration or identityref itself.
		// This also handles cases where the leaf is a typedef that is an enumeration
		// or identityref, since the isEnumType check does not use the name of the
		// type.
		types = append(types, e.Type)
	case isUnionType(e.Type):
		// Check for leaves that include a union that itself
		// includes an identityref or enumerated value.
		types = append(types, enumeratedUnionTypes(e.Type.Type)...)
	}

	if types != nil {
		return e
	}
	return nil
}

// findMappableEntities finds the descendants of a yang.Entry (e) that should be mapped in
// the generated code. The descendants that represent directories are appended to the dirs
// map (keyed by the schema path). Those that represent enumerated types (identityref, enumeration,
// unions containing these types, or typedefs containing these types) are appended to the
// enums map, which is again keyed by schema path. If any child of the entry is in a module
// defined in excludeModules, it is skipped. If compressPaths is set to true, then names are
// mapped with path compression enabled. The set of modules that the current code generation
// is processing is specified by the modules slice. This function returns slice of errors
// encountered during processing.
func findMappableEntities(e *yang.Entry, dirs map[string]*yang.Entry, enums map[string]*yang.Entry, excludeModules []string, compressPaths bool, modules []*yang.Entry) util.Errors {
	// Skip entities who are defined within a module that we have been instructed
	// not to generate code for.
	for _, s := range excludeModules {
		for _, m := range modules {
			if m.Name == s && m.Namespace().Name == e.Namespace().Name {
				return nil
			}
		}
	}

	var errs util.Errors
	for _, ch := range children(e) {
		switch {
		case ch.IsLeaf(), ch.IsLeafList():
			// Leaves are not mapped as directories so do not map them unless we find
			// something that will be an enumeration - so that we can deal with this
			// as a top-level code entity.
			if e := mappableLeaf(ch); e != nil {
				enums[ch.Path()] = e
			}
		case isConfigState(ch) && compressPaths:
			// If this is a config or state container and we are compressing paths
			// then we do not want to map this container - but we do want to map its
			// children.
			errs = util.AppendErrs(errs, findMappableEntities(ch, dirs, enums, excludeModules, compressPaths, modules))
		case hasOnlyChild(ch) && children(ch)[0].IsList() && compressPaths:
			// This is a surrounding container for a list, and we are compressing
			// paths, so we don't want to map it but again we do want to map its
			// children.
			errs = util.AppendErrs(errs, findMappableEntities(ch, dirs, enums, excludeModules, compressPaths, modules))
		case isChoiceOrCase(ch):
			// Don't map for a choice or case node itself, and rather skip over it.
			// However, we must walk each branch to find the first container that
			// exists there (if one does) to provide a mapping.
			nonchoice := map[string]*yang.Entry{}
			findFirstNonChoice(ch, nonchoice)
			for _, gch := range nonchoice {
				// The first entry that is not a choice or case could be a leaf
				// so we need to check whether it is an enumerated leaf that
				// should have code generated for it.
				if gch.IsLeaf() || gch.IsLeafList() {
					if e := mappableLeaf(gch); e != nil {
						enums[e.Path()] = e
					}
					continue
				}

				if gch.IsContainer() || gch.IsList() {
					dirs[fmt.Sprintf("%s/%s", ch.Parent.Path(), gch.Name)] = gch
				}
				errs = util.AppendErrs(errs, findMappableEntities(gch, dirs, enums, excludeModules, compressPaths, modules))
			}
		case ch.IsContainer(), ch.IsList():
			dirs[ch.Path()] = ch
			// Recurse down the tree.
			errs = util.AppendErrs(errs, findMappableEntities(ch, dirs, enums, excludeModules, compressPaths, modules))
		case ch.Kind == yang.AnyDataEntry:
			continue
		default:
			errs = util.AppendErr(errs, fmt.Errorf("unknown type of entry %v in findMappableEntities for %s", e.Kind, e.Path()))
		}
	}
	return errs
}

// findRootEntries finds the entities that are at the root of the YANG schema tree,
// and returns them.
func findRootEntries(structs map[string]*yang.Entry, compressPaths bool) map[string]*yang.Entry {
	rootEntries := map[string]*yang.Entry{}
	for n, s := range structs {
		pp := strings.Split(s.Path(), "/")
		switch len(pp) {
		case 3:
			// Find all containers and lists that have a path of
			// the form /module/entity-name regardless of whether
			// cg.CompressOCPaths is enabled. In the case that we
			// are compressing, then all invalid elements have
			// already been compressed out of the schema by this
			// stage.
			if s.IsContainer() || s.IsList() {
				rootEntries[n] = s
			}
		case 4:
			// If schema path compression is enabled then we need
			// to find entities that might be one level deeper in the
			// tree, for example, the /interfaces/interface list.
			// Since we never expect a top-level 'state' or 'config'
			// container, then it is only such lists that must be
			// identified.
			if compressPaths && s.IsList() {
				rootEntries[n] = s
			}
		}
	}
	return rootEntries
}

// createFakeRoot extracts the entities that are at the root of the YANG schema tree,
// which otherwise would have no parent in the generated structs, and appends them to
// a synthesised root element. Such entries are extracted from the supplied structs
// if they are lists or containers, or from the rootElems supplied if they are leaves
// or leaf-lists. In the case that the code generation is compressing the
// YANG schema, list entries that are two levels deep (e.g., /interfaces/interface) are
// also appended to the synthesised root entity (i.e., in this case the root element
// has a map entry named 'Interface', and the corresponding NewInterface() method.
// Takes the directories that are identified at the root (dirs), the elements found
// at the root (rootElems, such that non-directories can be mapped), and a string
// indicating the root name.
func createFakeRoot(structs map[string]*yang.Entry, rootElems []*yang.Entry, rootName string, compressPaths bool) error {
	if rootName == "" {
		rootName = defaultRootName
	}

	fakeRoot := &yang.Entry{
		Name: rootName,
		Kind: yang.DirectoryEntry,
		Dir:  map[string]*yang.Entry{},
		// Create a fake node that corresponds to the fake root, this
		// ensures that we can match the element elsewhere.
		Node: &yang.Value{
			Name: rootElementNodeName,
		},
	}

	for _, s := range findRootEntries(structs, compressPaths) {
		if e, ok := fakeRoot.Dir[s.Name]; ok {
			return fmt.Errorf("duplicate entry %s at the root: exists: %v, new: %v", s.Name, e.Path(), s.Path())
		}
		fakeRoot.Dir[s.Name] = s
	}

	for _, l := range rootElems {
		if l.IsLeaf() || l.IsLeafList() {
			fakeRoot.Dir[l.Name] = l
		}
	}

	// Append the synthesised root entry to the list of structs for which
	// code should be generated.
	structs["/"] = fakeRoot
	return nil
}
