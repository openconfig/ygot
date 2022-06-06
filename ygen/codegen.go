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

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/genutil"
	"github.com/openconfig/ygot/util"

	"github.com/openconfig/ygot/internal/igenutil"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
)

// YANGCodeGenerator is a structure that is used to pass arguments as to
// how the output Go code should be generated.
type YANGCodeGenerator struct {
	// Config stores the configuration parameters used for code generation.
	Config GeneratorConfig
}

// GeneratorConfig stores the configuration options used for code generation.
type GeneratorConfig struct {
	// PackageName is the name that should be used for the generating package.
	PackageName string
	// Caller is the name of the binary calling the generator library, it is
	// included in the header of output files for debugging purposes. If a
	// string is not specified, the location of the library is utilised.
	Caller string
	// GenerateJSONSchema stores a boolean which defines whether to generate
	// the JSON corresponding to the YANG schema parsed to generate the
	// output code.
	GenerateJSONSchema bool
	// StoreRawSchema the raw JSON schema should be returned by the code
	// generation function, such that it can be handled by an external
	// library.
	StoreRawSchema bool
	// ParseOptions contains parsing options for a given set of schema files.
	ParseOptions ParseOpts
	// TransformationOptions contains options for how the generated code
	// may be transformed from a simple 1:1 mapping with respect to the
	// given YANG schema.
	TransformationOptions TransformationOpts
	// ProtoOptions stores a struct which contains Protobuf specific options.
	ProtoOptions ProtoOpts
	// IncludeDescriptions specifies that YANG entry descriptions are added
	// to the JSON schema. Is false by default, to reduce the size of generated schema
	IncludeDescriptions bool
}

// ParseOpts contains parsing configuration for a given schema.
type ParseOpts struct {
	// ExcludeModules specifies any modules that are included within the set of
	// modules that should have code generated for them that should be ignored during
	// code generation. This is due to the fact that some schemas (e.g., OpenConfig
	// interfaces) currently result in overlapping entities (e.g., /interfaces).
	ExcludeModules []string
	// YANGParseOptions provides the options that should be handed to the
	// github.com/openconfig/goyang/pkg/yang library. These specify how the
	// input YANG files should be parsed.
	YANGParseOptions yang.Options
	// SkipEnumDeduplication specifies whether leaves of type 'enumeration' that
	// are used in multiple places in the schema should share a common type within
	// the generated code that is output by ygen. By default (false), a common type
	// is used.
	//
	// This behaviour is useful in scenarios where there is no difference between
	// two types, and the leaf is mirrored in a logical way (e.g., the OpenConfig
	// config/state split). For example:
	//
	// grouping foo-config {
	//	leaf enabled {
	//		type enumeration {
	//			enum A;
	//			enum B;
	//			enum C;
	//		}
	//	 }
	// }
	//
	//  container config { uses foo-config; }
	//  container state { uses foo-config; }
	//
	// will result in a single enumeration type (ModuleName_Config_Enabled) being
	// output when de-duplication is enabled.
	//
	// When it is disabled, two different enumerations (ModuleName_(State|Config)_Enabled)
	// will be output in the generated code.
	SkipEnumDeduplication bool
}

// TransformationOpts specifies transformations to the generated code with
// respect to the input schema.
type TransformationOpts struct {
	// CompressBehaviour specifies how the set of direct children of any
	// entry should determined. Specifically, whether compression is
	// enabled, and whether state fields in the schema should be excluded.
	CompressBehaviour genutil.CompressBehaviour
	// IgnoreShadowSchemaPaths indicates whether when OpenConfig path
	// compression is enabled, that the shadowed paths are to be ignored
	// while while unmarshalling.
	IgnoreShadowSchemaPaths bool
	// GenerateFakeRoot specifies whether an entity that represents the
	// root of the YANG schema tree should be generated in the generated
	// code.
	GenerateFakeRoot bool
	// FakeRootName specifies the name of the struct that should be generated
	// representing the root.
	FakeRootName string
	// ExcludeState specifies whether config false values should be
	// included in the generated code output. When set, all values that are
	// not writeable (i.e., config false) within the YANG schema and their
	// children are excluded from the generated code.
	ExcludeState bool
	// ShortenEnumLeafNames removes the module name from the name of
	// enumeration leaves.
	ShortenEnumLeafNames bool
	// EnumOrgPrefixesToTrim trims the organization name from the module
	// part of the name of enumeration leaves if there is a match.
	EnumOrgPrefixesToTrim []string
	// UseDefiningModuleForTypedefEnumNames uses the defining module name
	// to prefix typedef enumerated types instead of the module where the
	// typedef enumerated value is used.
	UseDefiningModuleForTypedefEnumNames bool
	// EnumerationsUseUnderscores specifies whether enumeration names
	// should use underscores between path segments.
	EnumerationsUseUnderscores bool
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
	// GoPackageBase specifies the base of the names that are used in
	// the go_package file option for generated protobufs. Additional
	// package identifiers are appended to the go_package - such that
	// the format <base>/<path>/<to>/<package> is used.
	GoPackageBase string
}

// NewYANGCodeGenerator returns a new instance of the YANGCodeGenerator
// struct to the calling function.
func NewYANGCodeGenerator(c *GeneratorConfig) *YANGCodeGenerator {
	cg := &YANGCodeGenerator{}

	if c != nil {
		cg.Config = *c
	}

	return cg
}

// yangEnum represents an enumerated type in YANG that is to be output in the
// Go code. The enumerated type may be a YANG 'identity' or enumeration.
type yangEnum struct {
	// name is the name of the enumeration or identity.
	name string
	// entry is the yang.Entry corresponding to the enumerated value.
	entry *yang.Entry
	// kind indicates the type of the enumeration.
	kind EnumeratedValueType
	// id is a unique synthesized key for the enumerated type.
	id string
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
	FilePath           []string // FilePath is the path to the file that this package should be written to.
	Header             string   // Header is the header text to be used in the package.
	Messages           []string // Messages is a slice of strings containing the set of messages that are within the generated package.
	Enums              []string // Enums is a slice of string containing the generated set of enumerations within the package.
	UsesYwrapperImport bool     // UsesYwrapperImport indicates whether the ywrapper proto package is used within the generated package.
	UsesYextImport     bool     // UsesYextImport indicates whether the yext proto package is used within the generated package.
}

// GenerateProto3 generates Protobuf 3 code for the input set of YANG files.
// The YANG schemas for which protobufs are to be created is supplied as the
// yangFiles argument, with included modules being searched for in includePaths.
// It returns a GeneratedProto3 struct containing the messages that are to be
// output, along with any associated values (e.g., enumerations).
func (cg *YANGCodeGenerator) GenerateProto3(yangFiles, includePaths []string) (*GeneratedProto3, util.Errors) {
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

	// This flag is always true for proto generation.
	cg.Config.TransformationOptions.UseDefiningModuleForTypedefEnumNames = true
	opts := IROptions{
		ParseOptions:                        cg.Config.ParseOptions,
		TransformationOptions:               cg.Config.TransformationOptions,
		NestedDirectories:                   cg.Config.ProtoOptions.NestedMessages,
		AbsoluteMapPaths:                    true,
		AppendEnumSuffixForSimpleUnionEnums: true,
	}

	ir, err := GenerateIR(yangFiles, includePaths, NewProtoLangMapper(basePackageName, enumPackageName), opts)
	if err != nil {
		return nil, util.NewErrs(err)
	}

	protoEnums, err := writeProtoEnums(ir.Enums, cg.Config.ProtoOptions.AnnotateEnumNames)
	if err != nil {
		return nil, util.NewErrs(err)
	}

	genProto := &GeneratedProto3{
		Packages: map[string]Proto3Package{},
	}

	// yerr stores errors encountered during code generation.
	var yerr util.Errors

	// pkgImports lists the imports that are required for the package that is being
	// written out.
	pkgImports := map[string]map[string]interface{}{}

	// Only create the enums package if there are enums that are within the schema.
	if len(protoEnums) > 0 {
		// Sort the set of enumerations so that they are deterministically output.
		sort.Strings(protoEnums)
		fp := []string{basePackageName, enumPackageName, fmt.Sprintf("%s.proto", enumPackageName)}
		genProto.Packages[fmt.Sprintf("%s.%s", basePackageName, enumPackageName)] = Proto3Package{
			FilePath:       fp,
			Enums:          protoEnums,
			UsesYextImport: cg.Config.ProtoOptions.AnnotateEnumNames,
		}
	}

	// Ensure that the slice of messages returned is in a deterministic order by
	// sorting the message paths. We use the path rather than the name as the
	// proto message name may not be unique.
	for _, directoryPath := range ir.OrderedDirectoryPaths() {
		m := ir.Directories[directoryPath]

		genMsg, errs := writeProto3Msg(m, ir, &protoMsgConfig{
			compressPaths:       cg.Config.TransformationOptions.CompressBehaviour.CompressEnabled(),
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
		if genMsg.UsesYwrapperImport {
			tp.UsesYwrapperImport = true
		}
		if genMsg.UsesYextImport {
			tp.UsesYextImport = true
		}
		genProto.Packages[genMsg.PackageName] = tp
	}

	for n, pkg := range genProto.Packages {
		var gpn string
		if cg.Config.ProtoOptions.GoPackageBase != "" {
			gpn = fmt.Sprintf("%s/%s", cg.Config.ProtoOptions.GoPackageBase, strings.ReplaceAll(n, ".", "/"))
		}
		ywrapperPath := ywrapperPath
		if !pkg.UsesYwrapperImport {
			ywrapperPath = ""
		}
		yextPath := yextPath
		if !pkg.UsesYextImport {
			yextPath = ""
		}
		h, err := writeProto3Header(proto3Header{
			PackageName:            n,
			Imports:                stringKeys(pkgImports[n]),
			SourceYANGFiles:        yangFiles,
			SourceYANGIncludePaths: includePaths,
			CompressPaths:          cg.Config.TransformationOptions.CompressBehaviour.CompressEnabled(),
			CallerName:             cg.Config.Caller,
			YwrapperPath:           ywrapperPath,
			YextPath:               yextPath,
			GoPackageName:          gpn,
		})
		if err != nil {
			yerr = util.AppendErrs(yerr, []error{err})
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
	// Initialise the set of YANG modules within the Goyang parsing package.
	moduleSet := yang.NewModules()
	// Propagate the options for the YANG library through to the parsing
	// code - this allows the calling binary to specify characteristics
	// of the YANG in a manner that we are transparent to.
	moduleSet.ParseOptions = options
	// Append the includePaths to the Goyang path variable, this ensures
	// that where a YANG module uses an 'include' statement to reference
	// another module, then Goyang can find this module to process.
	for _, path := range includePaths {
		moduleSet.AddPath(path)
	}

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

	// Deduplicate the modules that are to be processed.
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
	// schematree is a copy of the YANG schema tree, containing only leaf
	// entries, such that schema paths can be referenced.
	schematree *schemaTree
	// modules is the set of parsed YANG modules that are being processed as part of the
	// code generatio, expressed as a slice of yang.Entry pointers.
	modules []*yang.Entry
	// modelData stores the details of the set of modules that were parsed to produce
	// the code. It is optionally returned in the generated code.
	modelData []*gpb.ModelData
}

// mappedDefinitions finds the set of directory and enumeration entities
// that are mapped to objects within output code in a language agnostic manner.
// It takes:
//	- yangFiles: an input set of YANG schema files and the paths that
//	- includePaths: the set of paths that are to be searched for included or
//	  imported YANG modules.
//	- cfg: the current generator's configuration.
// It returns a mappedYANGDefinitions struct populated with the directory, enum
// entries in the input schemas as well as the calculated schema tree.
func mappedDefinitions(yangFiles, includePaths []string, cfg *GeneratorConfig) (*mappedYANGDefinitions, util.Errors) {
	modules, errs := processModules(yangFiles, includePaths, cfg.ParseOptions.YANGParseOptions)
	if errs != nil {
		return nil, errs
	}

	// Build a map of excluded modules to simplify lookup.
	excluded := map[string]bool{}
	for _, e := range cfg.ParseOptions.ExcludeModules {
		excluded[e] = true
	}

	// Extract the entities that are eligible to have code generated for
	// them from the modules that are provided as an argument.
	dirs := map[string]*yang.Entry{}
	enums := map[string]*yang.Entry{}
	var rootElems, treeElems []*yang.Entry
	for _, module := range modules {
		// Need to transform the AST based on compression behaviour.
		genutil.TransformEntry(module, cfg.TransformationOptions.CompressBehaviour)

		errs = append(errs, findMappableEntities(module, dirs, enums, cfg.ParseOptions.ExcludeModules, cfg.TransformationOptions.CompressBehaviour.CompressEnabled(), modules)...)
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
	if cfg.TransformationOptions.GenerateFakeRoot {
		if err := createFakeRoot(dirs, rootElems, cfg.TransformationOptions.FakeRootName, cfg.TransformationOptions.CompressBehaviour.CompressEnabled()); err != nil {
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

	modelData, err := util.FindModelData(modules)
	if err != nil {
		return nil, util.NewErrs(fmt.Errorf("cannot extract model data, %v", err))
	}

	return &mappedYANGDefinitions{
		directoryEntries: dirs,
		enumEntries:      enums,
		schematree:       st,
		modules:          ms,
		modelData:        modelData,
	}, nil
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
	for _, ch := range util.Children(e) {
		switch {
		case ch.IsLeaf(), ch.IsLeafList():
			// Leaves are not mapped as directories so do not map them unless we find
			// something that will be an enumeration - so that we can deal with this
			// as a top-level code entity.
			if e := igenutil.MappableLeaf(ch); e != nil {
				enums[ch.Path()] = e
			}
		case util.IsConfigState(ch) && compressPaths:
			// If this is a config or state container and we are compressing paths
			// then we do not want to map this container - but we do want to map its
			// children.
			errs = util.AppendErrs(errs, findMappableEntities(ch, dirs, enums, excludeModules, compressPaths, modules))
		case util.HasOnlyChild(ch) && util.Children(ch)[0].IsList() && compressPaths:
			// This is a surrounding container for a list, and we are compressing
			// paths, so we don't want to map it but again we do want to map its
			// children.
			errs = util.AppendErrs(errs, findMappableEntities(ch, dirs, enums, excludeModules, compressPaths, modules))
		case util.IsChoiceOrCase(ch):
			// Don't map for a choice or case node itself, and rather skip over it.
			// However, we must walk each branch to find the first container that
			// exists there (if one does) to provide a mapping.
			nonchoice := util.FindFirstNonChoiceOrCase(ch)
			for _, gch := range nonchoice {
				// The first entry that is not a choice or case could be a leaf
				// so we need to check whether it is an enumerated leaf that
				// should have code generated for it.
				if gch.IsLeaf() || gch.IsLeafList() {
					if e := igenutil.MappableLeaf(gch); e != nil {
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
			errs = util.AppendErr(errs, fmt.Errorf("unknown type of entry %v in findMappableEntities for %s", ch.Kind, ch.Path()))
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
			// when compression is enabled. In the case that we
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

// MakeFakeRoot creates and returns a fakeroot *yang.Entry with rootName as its
// name. It has an empty, but initialized Dir.
func MakeFakeRoot(rootName string) *yang.Entry {
	return &yang.Entry{
		Name: rootName,
		Kind: yang.DirectoryEntry,
		Dir:  map[string]*yang.Entry{},
		// Create a fake node that corresponds to the fake root, this
		// ensures that we can match the element elsewhere.
		Node: &yang.Value{
			Name: igenutil.RootElementNodeName,
		},
	}
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
		rootName = igenutil.DefaultRootName
	}

	fakeRoot := MakeFakeRoot(rootName)

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
