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
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	log "github.com/golang/glog"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygot"
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

// yangStruct represents a struct that is to be generated. It stores the
// fields that are required to output the relevant code for the entity.
type yangStruct struct {
	name       string                 // name is the name of the struct to be generated.
	entry      *yang.Entry            // entry is the yang.Entry that corresponds to the schema element being converted to a struct.
	fields     map[string]*yang.Entry // fields is a map, keyed by the YANG node identifier, of the entries that are the struct fields.
	path       []string               // path is a slice of strings indicating the element's path.
	listAttr   *yangListAttr          // listAttr is used to store characteristics of structs that represent YANG lists.
	isFakeRoot bool                   // isFakeRoot indicates that the struct is a fake root struct, so specific mapping rules should be implemented.
}

// yangListAttr is used to store the additional elements for a Go struct that
// are required if the struct represents a YANG list. It stores the name of
// the key elements, and their associated types, along with pointers to those
// elements.
type yangListAttr struct {
	// keys is a map, keyed by the name of the key leaf, with values of the type
	// of the key of a YANG list.
	keys map[string]mappedType
	// keyElems is a slice containing the pointers to yang.Entry structs that
	// make up the list key.
	keyElems []*yang.Entry
}

// yangGoEnum represents an enumerated type in YANG that is to be output in the
// Go code. The enumerated type may be a YANG 'identity' or enumeration.
type yangGoEnum struct {
	name  string      // The name of the enumeration or identity.
	entry *yang.Entry // The yang.Entry corresponding to the enumerated value.
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
	Structs []string // Structs is the generated set of structs representing containers or lists in the input YANG models.
	Enums   []string // Enums is the generated set of enum definitions corresponding to identities and enumerations in the input YANG models.
	Header  string   // Header is the package-level header or the generated code.
	EnumMap string   // EnumMap is a Go map that allows the YANG string values of enumerated types to be resolved.
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
}

// YANGCodeGeneratorError is a type implementing error that is returned to the
// caller of the library when errors are encountered during code generation. It
// implements error such that idiomatic if err != nil testing can be used, and
// contains a list of errors associted with the code generation call.
type YANGCodeGeneratorError struct {
	Errors []error
}

// NewYANGCodeGeneratorError returns a new instance of the YANGCodeGeneratorError
// struct with the relevant fields initialised.
func NewYANGCodeGeneratorError() *YANGCodeGeneratorError {
	return &YANGCodeGeneratorError{}
}

// Error is a method of YANGCodeGeneratorError which returns a concatenated
// string of the errors that are stored within the YANGCodeGeneratorError
// struct. Its implementation allows a YANGCodeGeneratorError to implement the
// error interface.
func (yangErr *YANGCodeGeneratorError) Error() string {
	var buf bytes.Buffer
	buf.WriteString("errors encountered during code generation:\n")

	// Concatenate the errors that are stored within the receiver YANGCodeGeneratorError struct.
	for _, e := range yangErr.Errors {
		buf.WriteString(fmt.Sprintf("%v\n", e))
	}
	return buf.String()
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
func (cg *YANGCodeGenerator) GenerateGoCode(yangFiles, includePaths []string) (*GeneratedGoCode, *YANGCodeGeneratorError) {
	// Process the modules from yangFiles, with the includePaths
	modules, errs := processModules(yangFiles, includePaths, cg.Config.YANGParseOptions)
	if len(errs) > 0 {
		return nil, &YANGCodeGeneratorError{Errors: errs}
	}

	// Extract the entities that are eligible to have code generated for them from the modules that are
	// provided as an argument.
	structs := make(map[string]*yang.Entry)
	enums := make(map[string]*yang.Entry)
	var rootElems []*yang.Entry
	for _, module := range modules {
		cg.findMappableEntities(module, structs, enums)
		if module != nil && module.Dir != nil {
			for _, e := range module.Dir {
				rootElems = append(rootElems, e)
			}
		}
	}

	// Build the schematree for the modules provided.
	st, err := buildSchemaTree(rootElems)
	if err != nil {
		return nil, &YANGCodeGeneratorError{Errors: []error{err}}
	}
	cg.state.schematree = st

	// If we were asked to generate a fake root entity, then go and find the top-level entities that
	// we were asked for.
	if cg.Config.GenerateFakeRoot {
		if err := cg.createFakeRoot(structs); err != nil {
			return nil, &YANGCodeGeneratorError{Errors: []error{err}}
		}
	}

	goStructs, errs := cg.state.buildGoStructDefinitions(structs, cg.Config.CompressOCPaths, cg.Config.GenerateFakeRoot)
	if len(errs) > 0 {
		return nil, &YANGCodeGeneratorError{Errors: errs}
	}

	codeHeader, err := writeGoHeader(yangFiles, includePaths, cg.Config)
	if err != nil {
		return nil, &YANGCodeGeneratorError{Errors: []error{err}}
	}

	// orderedStructNames is used to store the structs that have been
	// identified in alphabetical order, such that they are returned in a
	// determinstic order to the calling application. This ensures that if
	// the slice is simply output in order, then the diffs generated are
	// minimised (i.e., diffs are not generated simply due to reordering of
	// the maps used).
	var orderedStructNames []string
	structNameMap := make(map[string]*yangStruct)
	for _, goStruct := range goStructs {
		orderedStructNames = append(orderedStructNames, goStruct.name)
		structNameMap[goStruct.name] = goStruct
	}
	sort.Strings(orderedStructNames)

	codegenErr := NewYANGCodeGeneratorError()
	var structSnippets []string
	for _, structName := range orderedStructNames {
		structOut, errs := writeGoStruct(structNameMap[structName], goStructs, cg.state, cg.Config.CompressOCPaths, cg.Config.GenerateJSONSchema)
		if len(errs) > 0 {
			codegenErr.Errors = append(codegenErr.Errors, errs...)
			continue
		}
		// Append the actual struct definitions that were returned.
		structSnippets = append(structSnippets, structOut.structDef)
		structSnippets = appendIfNotEmpty(structSnippets, structOut.listKeys)
		structSnippets = appendIfNotEmpty(structSnippets, structOut.methods)
		structSnippets = appendIfNotEmpty(structSnippets, structOut.interfaces)
	}

	goEnums, errs := cg.state.findEnumSet(enums, cg.Config.CompressOCPaths)
	if len(errs) > 0 {
		codegenErr.Errors = append(codegenErr.Errors, errs...)
		return nil, codegenErr
	}

	// orderedEnumNames is used to store the enumerated types that have been
	// identified in alphabetical order, such that they are returned in a
	// deterministic order to the calling application. This ensures that
	// the diffs are minimised, similarly to the use of orderedStructNames
	// above.
	var orderedEnumNames []string
	enumNameMap := make(map[string]*yangGoEnum)
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
			codegenErr.Errors = append(codegenErr.Errors, err)
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
		codegenErr.Errors = append(codegenErr.Errors, err)
	}

	var rawSchema []byte
	var jsonSchema string
	if cg.Config.GenerateJSONSchema {
		var err error
		if rawSchema, err = cg.serialiseStructDefinitions(goStructs); err != nil {
			codegenErr.Errors = append(codegenErr.Errors, err)
		}

		if rawSchema != nil {
			if jsonSchema, err = writeGoSchema(rawSchema, cg.Config.GoOptions.SchemaVarName); err != nil {
				codegenErr.Errors = append(codegenErr.Errors, err)
			}
		}
	}

	// Return any errors that were encountered during code generation.
	if len(codegenErr.Errors) != 0 {
		return nil, codegenErr
	}

	return &GeneratedGoCode{
		Header:         codeHeader,
		Structs:        structSnippets,
		Enums:          enumSnippets,
		EnumMap:        enumMap,
		JSONSchemaCode: jsonSchema,
		RawJSONSchema:  rawSchema,
	}, nil
}

// processModules takes a list of the filenames of YANG modules (yangFiles),
// and a list of paths in which included modules or submodules may be found,
// and returns a processed set of yang.Entry pointers which correspond to the
// generated code for the modules. If errors are returned during the Goyang
// processing of the modules, these errors are returned.
func processModules(yangFiles, includePaths []string, options yang.Options) ([]*yang.Entry, []error) {
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

	var processErr []error
	for _, name := range yangFiles {
		if err := moduleSet.Read(name); err != nil {
			processErr = append(processErr, err)
		}
	}

	if len(processErr) > 0 {
		return nil, processErr
	}

	if errs := moduleSet.Process(); len(errs) != 0 {
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
	entries := make([]*yang.Entry, len(modNames))
	for _, modName := range modNames {
		entries = append(entries, yang.ToEntry(mods[modName]))
	}
	return entries, nil
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

	types := []*yang.YangType{}
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
	if len(types) > 0 {
		return e
	}
	return nil
}

// findMappableEntities finds the descendants of a yang.Entry (e) that should be mapped
// into Go code. The descendants that are to be mapped to structs are added to the supplied
// structs map, and the enumerated values (identityref, enumeration, any typedef referencing
// an enumeration or identityref) are added to the supplied enums map.
func (cg *YANGCodeGenerator) findMappableEntities(e *yang.Entry, structs map[string]*yang.Entry, enums map[string]*yang.Entry) {
	pp := strings.Split(e.Path(), "/")
	if !strings.HasPrefix(e.Path(), "/") || len(pp) < 2 {
		// This is a malformed path, since the path is defined in the form
		// /module/container-one. If the length is <2, there was no leading
		// slash on the path.
		return
	}

	// Skip entities who are defined within a module that we have been instructed
	// not to generate code for.
	for _, s := range cg.Config.ExcludeModules {
		if s == pp[1] {
			return
		}
	}

	for _, ch := range children(e) {
		switch {
		case !ch.IsDir():
			// Leaves are not mapped as structs, so do not map them unless we find
			// something that will be an enumeration - so that we can deal with this
			// as a top-level code entity.
			if e := mappableLeaf(ch); e != nil {
				enums[ch.Path()] = e
			}
		case isConfigState(ch) && cg.Config.CompressOCPaths:
			// If this is a config or state container and we are compressing paths
			// then we do not want to map this container - but we do want to map its
			// children.
			cg.findMappableEntities(ch, structs, enums)
		case hasOnlyChild(ch) && children(ch)[0].IsList() && cg.Config.CompressOCPaths:
			// This is a surrounding container for a list, and we are compressing
			// paths, so we don't want to map it but again we do want to map its
			// children.
			cg.findMappableEntities(ch, structs, enums)
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

				if gch.IsDir() {
					structs[fmt.Sprintf("%s/%s", ch.Parent.Path(), gch.Name)] = gch
				}
				cg.findMappableEntities(gch, structs, enums)
			}
		default:
			structs[ch.Path()] = ch
			// Recurse down the tree.
			cg.findMappableEntities(ch, structs, enums)
		}
	}
}

// findRootEntries finds the entities that are at the root of the YANG schema tree,
// and returns them.
func (cg *YANGCodeGenerator) findRootEntries(structs map[string]*yang.Entry) map[string]*yang.Entry {
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
			if s.IsDir() {
				rootEntries[n] = s
			}
		case 4:
			// If schema path compression is enabled then we need
			// to find entities that might be one level deeper in the
			// tree, for example, the /interfaces/interface list.
			// Since we never expect a top-level 'state' or 'config'
			// container, then it is only such lists that must be
			// identified.
			if cg.Config.CompressOCPaths && s.IsList() {
				rootEntries[n] = s
			}
		}
	}
	return rootEntries
}

// createFakeRoot extracts the entities that are at the root of the YANG schema tree,
// which otherwise would have no parent in the generated structs, and appends them to
// a synthesised root element. In the case that the code generation is compressing the
// YANG schema, list entries that are two levels deep (e.g., /interfaces/interface) are
// also appended to the synthesised root entity (i.e., in this case the root element
// has a map entry named 'Interface', and the corresponding NewInterface() method.
func (cg *YANGCodeGenerator) createFakeRoot(structs map[string]*yang.Entry) error {
	rootName := defaultRootName
	if cg.Config.FakeRootName != "" {
		rootName = cg.Config.FakeRootName
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

	for _, s := range cg.findRootEntries(structs) {
		if e, ok := fakeRoot.Dir[s.Name]; ok {
			return fmt.Errorf("duplicate entry %s at the root: exists: %v, new: %v", s.Name, e.Path(), s.Path())
		}
		fakeRoot.Dir[s.Name] = s
	}

	// Append the synthesised root entry to the list of structs for which
	// code should be generated.
	structs["/"] = fakeRoot
	return nil
}

// serialiseStructDefinitions takes an input set of structs - expressed as a map of yangStructs
// and outputs a byte slice which corresponds to the serialised JSON representation of the schema.
// The output JSON contains only the root level entities of the schema - such that there is no
// repetition of definitions of entries. The entries have aditional information appended to the yang.Entry
// Annotation field - particularly, the name of the struct that was generated for a particular schema element,
// and the corresponding path within the schema. Both of these elements cannot be reconstructed from
// the deserialised yang.Entry contents.
func (cg *YANGCodeGenerator) serialiseStructDefinitions(structs map[string]*yangStruct) ([]byte, error) {
	entries := map[string]*yang.Entry{}
	for _, e := range structs {
		entries[e.name] = e.entry
		entries[e.name].Annotation = map[string]interface{}{
			"schemapath": e.entry.Path(),
			"structname": e.name,
		}
		if e.isFakeRoot {
			entries[e.name].Annotation["isFakeRoot"] = true
		}
	}

	schema := map[string]*yang.Entry{}
	if cg.Config.GenerateFakeRoot {
		rootName := yang.CamelCase(defaultRootName)
		if cg.Config.FakeRootName != "" {
			rootName = yang.CamelCase(cg.Config.FakeRootName)
		}
		if e, ok := entries[rootName]; ok {
			schema[rootName] = e
		}
	} else {
		schema = cg.findRootEntries(entries)
	}

	json, err := json.MarshalIndent(schema, "", "    ")
	if err != nil {
		return nil, err
	}
	return json, nil
}
