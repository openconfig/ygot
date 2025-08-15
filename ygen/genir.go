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

package ygen

import (
	"fmt"
	"sort"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/genutil"
	"github.com/openconfig/ygot/util"
	"github.com/openconfig/ygot/ygot"
)

// IROptions contains options used to customize IR generation.
type IROptions struct {
	// ParseOptions specifies the options for how the YANG schema is
	// produced.
	ParseOptions ParseOpts

	// Transformation options specifies any transformations that should
	// be applied to the input YANG schema when producing the IR.
	TransformationOptions TransformationOpts

	// NestedDirectories specifies whether the generated directories should
	// be nested in the IR.
	NestedDirectories bool

	// AbsoluteMapPaths specifies whether the path annotation provided for
	// each field should be relative paths or absolute paths.
	AbsoluteMapPaths bool

	// AppendEnumSuffixForSimpleUnionEnums appends an "Enum" suffix to the
	// enumeration name for simple (i.e. non-typedef) leaves which are
	// unions with an enumeration inside. This makes all inlined
	// enumerations within unions, whether typedef or not, have this
	// suffix, achieving consistency. Since this flag is planned to be a
	// v1 compatibility flag along with
	// UseDefiningModuleForTypedefEnumNames, and will be removed in v1, it
	// only applies when useDefiningModuleForTypedefEnumNames is also set
	// to true.
	// NOTE: This flag will be removed by v1 release.
	AppendEnumSuffixForSimpleUnionEnums bool
}

// GenerateIR creates the ygen intermediate representation for a set of
// YANG modules. The YANG files to be parsed are read from the yangFiles
// argument, with any includes that they use searched for in the string
// slice of paths specified by includePaths. The supplier LangMapper interface
// is used to perform mapping of language-specific naming whilst creating
// the IR -- the full details of the implementation of LangMapper can be found
// in ygen/ir.go and docs/code-generation-design.md.
//
// The supplied IROptions controls the parsing and transformations that are
// applied to the schema whilst generating the IR.
//
// GenerateIR returns the complete ygen intermediate representation.
func GenerateIR(yangFiles, includePaths []string, langMapper LangMapper, opts IROptions) (*IR, error) {
	// Extract the entities to be mapped into structs and enumerations in the output
	// Go code. Extract the schematree from the modules provided such that it can be
	// used to reference entities within the tree.
	mdef, errs := mappedDefinitions(yangFiles, includePaths, opts)
	if errs != nil {
		return nil, errs
	}

	enumSet, genEnums, errs := findEnumSet(mdef.enumEntries, opts.TransformationOptions.CompressBehaviour.CompressEnabled(), !opts.TransformationOptions.EnumerationsUseUnderscores, opts.TransformationOptions.SkipEnumDeduplication, opts.TransformationOptions.ShortenEnumLeafNames, opts.TransformationOptions.UseDefiningModuleForTypedefEnumNames, opts.AppendEnumSuffixForSimpleUnionEnums, opts.TransformationOptions.EnumOrgPrefixesToTrim)
	if errs != nil {
		return nil, errs
	}

	langMapper.setEnumSet(enumSet)
	langMapper.setSchemaTree(mdef.schematree)

	directoryMap, errs := buildDirectoryDefinitions(langMapper, mdef.directoryEntries, opts)
	if errs != nil {
		return nil, errs
	}

	var rootEntry *yang.Entry
	for _, d := range directoryMap {
		if d.IsFakeRoot {
			rootEntry = d.Entry
		}
	}

	dirDets, err := getOrderedDirDetails(langMapper, directoryMap, mdef.schematree, opts)
	if err != nil {
		return nil, util.AppendErr(errs, err)
	}

	var enumDefinitionMap map[string]*EnumeratedYANGType
	if len(genEnums) != 0 {
		enumDefinitionMap = make(map[string]*EnumeratedYANGType, len(genEnums))
	}
	for _, enum := range genEnums {
		et := &EnumeratedYANGType{
			Name:     enum.name,
			Kind:     enum.kind,
			TypeName: enum.entry.Type.Name,
		}
		if _, ok := enumDefinitionMap[enum.id]; ok {
			return nil, util.AppendErr(errs, fmt.Errorf("Enumeration already created: "+et.Name))
		}

		if defaultValue, ok := enum.entry.SingleDefaultValue(); ok {
			et.TypeDefaultValue = defaultValue
		}

		switch {
		case len(enum.entry.Type.Type) != 0:
			errs = append(errs, fmt.Errorf("unimplemented: support for multiple enumerations within a union for %v", enum.name))
			continue
		case et.Kind == UnknownEnumerationType:
			errs = append(errs, fmt.Errorf("unknown type of enumerated value for %s, got: %v, type: %v", enum.name, enum, enum.entry.Type))
			continue
		}

		switch enum.kind {
		case IdentityType:
			et.IdentityBaseName = enum.entry.Type.IdentityBase.Name
			// enum corresponds to an identityref - hence the values are defined
			// based on the values that the identity has. Since there is no explicit ordering
			// in an identity, then we go through and put the values in alphabetical order in
			// order to avoid reordering during code generation of the same entity.
			valNames := []string{}
			valLookup := map[string]*yang.Identity{}
			for _, v := range enum.entry.Type.IdentityBase.Values {
				valNames = append(valNames, v.Name)
				valLookup[v.Name] = v
			}
			sort.Strings(valNames)

			for i, v := range valNames {
				et.ValToYANGDetails = append(et.ValToYANGDetails, ygot.EnumDefinition{
					Name:           v,
					DefiningModule: genutil.ParentModuleName(valLookup[v]),
					Value:          i,
				})
			}
		default:
			// The remaining enumerated types are all represented as an Enum type within the
			// Goyang entry construct. The values are accessed in a map keyed by an int64
			// and with a value of the name of the enumerated value - retrieved via ValueMap().
			var values []int
			valueMap := enum.entry.Type.Enum.ValueMap()
			for v := range valueMap {
				values = append(values, int(v))
			}
			sort.Ints(values)
			for _, v := range values {
				et.ValToYANGDetails = append(et.ValToYANGDetails, ygot.EnumDefinition{
					Name:  valueMap[int64(v)],
					Value: v,
				})
			}
		}

		et.Flags = langMapper.PopulateEnumFlags(*et, enum.entry.Type)

		enumDefinitionMap[enum.id] = et
	}

	if errs != nil {
		return nil, errs
	}

	return &IR{
		Directories:   dirDets,
		Enums:         enumDefinitionMap,
		ModelData:     mdef.modelData,
		opts:          opts,
		fakeroot:      rootEntry,
		parsedModules: mdef.modules,
	}, nil
}
