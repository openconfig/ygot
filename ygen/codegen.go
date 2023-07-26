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

// Package ygen contains a library and base configuration options that can be
// extended to generate language-specific structs from a YANG model.
// The Goyang parsing library is used to parse YANG. The output can consider
// OpenConfig-specific conventions such that the schema is compressed.
// The output of this library is an intermediate representation (IR) designed
// to reduce the need for working with the Goyang parsing library's AST.
package ygen

import (
	"errors"
	"fmt"
	"strings"

	log "github.com/golang/glog"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/genutil"
	"github.com/openconfig/ygot/util"
	"github.com/openconfig/ygot/yangschema"

	"github.com/openconfig/ygot/internal/igenutil"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
)

// ParseOpts contains parsing configuration for a given schema.
type ParseOpts struct {
	// IgnoreUnsupportedStatements ignores unsupported YANG statements when
	// parsing, such that they do not show up errors during IR generation.
	IgnoreUnsupportedStatements bool
	// ExcludeModules specifies any modules that are included within the set of
	// modules that should have code generated for them that should be ignored during
	// code generation. This is due to the fact that some schemas (e.g., OpenConfig
	// interfaces) currently result in overlapping entities (e.g., /interfaces).
	ExcludeModules []string
	// YANGParseOptions provides the options that should be handed to the
	// github.com/openconfig/goyang/pkg/yang library. These specify how the
	// input YANG files should be parsed.
	YANGParseOptions yang.Options
}

// TransformationOpts specifies transformations to the generated code with
// respect to the input schema.
type TransformationOpts struct {
	// CompressBehaviour specifies how the set of direct children of any
	// entry should determined. Specifically, whether compression is
	// enabled, and whether state fields in the schema should be excluded.
	CompressBehaviour genutil.CompressBehaviour
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
		entry := yang.ToEntry(mods[modName])
		if errs := entry.GetErrors(); len(errs) > 0 {
			return nil, util.Errors(errs)
		}
		entries = append(entries, entry)
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
	schematree *yangschema.Tree
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
//   - yangFiles: an input set of YANG schema files and the paths that
//   - includePaths: the set of paths that are to be searched for included or
//     imported YANG modules.
//   - opts: the current generator's configuration.
//
// It returns a mappedYANGDefinitions struct populated with the directory, enum
// entries in the input schemas as well as the calculated schema tree.
func mappedDefinitions(yangFiles, includePaths []string, opts IROptions) (*mappedYANGDefinitions, util.Errors) {
	modules, errs := processModules(yangFiles, includePaths, opts.ParseOptions.YANGParseOptions)
	if errs != nil {
		return nil, errs
	}

	// Build a map of excluded modules to simplify lookup.
	excluded := map[string]bool{}
	for _, e := range opts.ParseOptions.ExcludeModules {
		excluded[e] = true
	}

	// Extract the entities that are eligible to have code generated for
	// them from the modules that are provided as an argument.
	dirs := map[string]*yang.Entry{}
	enums := map[string]*yang.Entry{}
	var rootElems, treeElems []*yang.Entry
	for _, module := range modules {
		// Need to transform the AST based on compression behaviour.
		genutil.TransformEntry(module, opts.TransformationOptions.CompressBehaviour)

		errs = append(errs, findMappableEntities(module, dirs, enums, opts.ParseOptions.ExcludeModules, opts.TransformationOptions.CompressBehaviour.CompressEnabled(), opts.ParseOptions.IgnoreUnsupportedStatements, modules)...)
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
	st, err := yangschema.BuildTree(treeElems)
	if err != nil {
		return nil, []error{err}
	}

	// If we were asked to generate a fake root entity, then go and find the top-level entities that
	// we were asked for.
	if opts.TransformationOptions.GenerateFakeRoot {
		if err := createFakeRoot(dirs, rootElems, opts.TransformationOptions.FakeRootName, opts.TransformationOptions.CompressBehaviour.CompressEnabled()); err != nil {
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
func findMappableEntities(e *yang.Entry, dirs map[string]*yang.Entry, enums map[string]*yang.Entry, excludeModules []string, compressPaths, ignoreUnsupportedStatements bool, modules []*yang.Entry) util.Errors {
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
			errs = util.AppendErrs(errs, findMappableEntities(ch, dirs, enums, excludeModules, compressPaths, ignoreUnsupportedStatements, modules))
		case util.HasOnlyChild(ch) && util.Children(ch)[0].IsList() && compressPaths:
			// This is a surrounding container for a list, and we are compressing
			// paths, so we don't want to map it but again we do want to map its
			// children.
			errs = util.AppendErrs(errs, findMappableEntities(ch, dirs, enums, excludeModules, compressPaths, ignoreUnsupportedStatements, modules))
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
				errs = util.AppendErrs(errs, findMappableEntities(gch, dirs, enums, excludeModules, compressPaths, ignoreUnsupportedStatements, modules))
			}
		case ch.IsContainer(), ch.IsList():
			dirs[ch.Path()] = ch
			// Recurse down the tree.
			errs = util.AppendErrs(errs, findMappableEntities(ch, dirs, enums, excludeModules, compressPaths, ignoreUnsupportedStatements, modules))
		case ch.Kind == yang.AnyDataEntry:
			continue
		default:
			if ignoreUnsupportedStatements {
				log.Infof("Unsupported statement type (%v) ignored: %s", ch.Kind, ch.Path())
				continue
			}
			errs = util.AppendErr(errs, fmt.Errorf("unsupported statement type (%v) in findMappableEntities for %s", ch.Kind, ch.Path()))
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
