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
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/genutil"
	"github.com/openconfig/ygot/util"
)

// enumSet contains generated enum names which can be queried.
// It should be constructed from findEnumSet().
// The name sets should not be changed (essentially treat it as immutable), as
// deduplication might have occurred, which depended on knowing all of the
// entries beforehand.
type enumSet struct {
	// uniqueIdentityNames is a map which is keyed by a string in the form of
	// definingModule/identityName which stores the Go name of the enumerated Go type
	// that has been created to represent the identity. This allows de-duplication
	// between identityref leaves that reference the same underlying identity. The
	// name used includes the defining module to avoid clashes between two identities
	// that are named the same within different modules.
	uniqueIdentityNames map[string]string
	// uniqueEnumeratedTypedefNames is a map, keyed by a synthesised path for the typedef,
	// generated in the form definingModule/typedefName, the value stores the Go name of
	// the enumeration which represents a typedef that includes an enumerated type.
	uniqueEnumeratedTypedefNames map[string]string
	// uniqueEnumeratedLeafNames is a map, keyed by a synthesised path to an
	// enumeration leaf. The path used reflects the data tree path of the leaf
	// within the module that it is defined. That is to say, if a module
	// example-module defines a hierarchy of global/config/a-leaf where a-leaf
	// is of type enumeration, then the path example-module/global/config/a-leaf
	// is used for a-leaf in the uniqueEnumeratedLeafNames. The value of the map
	// is the name of the Go enumerated value to which it is mapped. The path based
	// on the module is guaranteed to be unique, since we cannot have multiple
	// modules of the same name, or multiple identical data tree paths within
	// the same module. This path is used since a particular leaf may be re-used
	// in multiple places, such that if the entire data tree path is used then
	// the names that are generated require deduplication. This approach ensures
	// that we have the same enumerated value for a particular leaf in multiple
	// contexts.
	// At the time of writing, in OpenConfig schemas, this occurs where there is
	// a module such as openconfig-bgp which defines /bgp and is also used at
	// /network-instances/network-instance/protocols/protocol/bgp.
	uniqueEnumeratedLeafNames map[string]string
}

// newEnumSet initializes a new empty enumSet instance.
func newEnumSet() *enumSet {
	return &enumSet{
		uniqueIdentityNames:          map[string]string{},
		uniqueEnumeratedTypedefNames: map[string]string{},
		uniqueEnumeratedLeafNames:    map[string]string{},
	}
}

// enumeratedUnionEntry takes an input YANG union yang.Entry and returns the set of enumerated
// values that should be generated for the entry. New yang.Entry instances are synthesised within
// the yangEnums returned such that enumerations can be generated directly from the output of
// this function in common with enumerations that are not within a union. The name of the enumerated
// value is calculated based on the original context, whether path compression is enabled based
// on the compressPaths boolean, and whether the name should not include underscores, as per the
// noUnderscores boolean.
func (s *enumSet) enumeratedUnionEntry(e *yang.Entry, compressPaths, noUnderscores, skipEnumDedup bool) ([]*yangEnum, error) {
	var es []*yangEnum

	for _, t := range util.EnumeratedUnionTypes(e.Type.Type) {
		var en *yangEnum
		switch {
		case t.IdentityBase != nil:
			identityName, err := s.identityrefBaseTypeFromIdentity(t.IdentityBase)
			if err != nil {
				return nil, err
			}
			en = &yangEnum{
				name: identityName,
				entry: &yang.Entry{
					Name: e.Name,
					Type: &yang.YangType{
						Name:         e.Type.Name,
						Kind:         yang.Yidentityref,
						IdentityBase: t.IdentityBase,
					},
				},
			}
		case t.Enum != nil:
			var enumName string
			var err error
			if util.IsYANGBaseType(t) {
				enumName, err = s.enumName(e, compressPaths, noUnderscores, skipEnumDedup)
			} else {
				enumName, err = s.typedefEnumeratedName(e, noUnderscores)
			}
			if err != nil {
				return nil, err
			}

			en = &yangEnum{
				name: enumName,
				entry: &yang.Entry{
					Name: e.Name,
					Type: &yang.YangType{
						Name: e.Type.Name,
						Kind: yang.Yenum,
						Enum: t.Enum,
					},
					Annotation: map[string]interface{}{"valuePrefix": util.SchemaPathNoChoiceCase(e)},
				},
			}
		}

		es = append(es, en)
	}

	return es, nil
}

// identityrefBaseTypeFromLeaf retrieves the mapped name of an identityref's
// base such that it can be used in generated code. The value that is returned
// is defining module name followed by the CamelCase-ified version of the
// base's name. This function wraps the identityrefBaseTypeFromIdentity
// function since it covers the common case that the caller is interested in
// determining the name from an identityref leaf, rather than directly from the
// identity.
func (s *enumSet) identityrefBaseTypeFromLeaf(idr *yang.Entry) (string, error) {
	return s.identityrefBaseTypeFromIdentity(idr.Type.IdentityBase)
}

// identityrefBaseTypeFromIdentity retrieves the generated type name of the
// input *yang.Identity. The value returned is based on the defining module
// followed by the CamelCase-ified version of the identity's name.
func (s *enumSet) identityrefBaseTypeFromIdentity(i *yang.Identity) (string, error) {
	definedName, ok := s.uniqueIdentityNames[s.identityBaseKey(i)]
	if !ok {
		return "", fmt.Errorf("enumSet: cannot retrieve type name for identity without a name generated (was findEnumSet called?): %+v", i)
	}
	return definedName, nil
}

// enumName retrieves the type name of the input enum *yang.Entry that will be
// used in the generated code.
func (s *enumSet) enumName(e *yang.Entry, compressPaths, noUnderscores, skipDedup bool) (string, error) {
	key, _ := s.enumLeafKey(e, compressPaths, noUnderscores, skipDedup)
	definedName, ok := s.uniqueEnumeratedLeafNames[key]
	if !ok {
		return "", fmt.Errorf("enumSet: cannot retrieve type name for enumerated leaf without a name generated (was findEnumSet called?): %v", e.Path())
	}
	return definedName, nil
}

// enumeratedTypedefTypeName retrieves the name of an enumerated typedef (i.e.,
// a typedef which is either an identityref or an enumeration). The resolved
// name is prefixed with the prefix supplied. If the type that was supplied
// within the resolveTypeArgs struct is not a type definition which includes an
// enumerated type, the MappedType returned is nil, otherwise it should be
// populated.
func (s *enumSet) enumeratedTypedefTypeName(args resolveTypeArgs, prefix string, noUnderscores bool) (*MappedType, error) {
	// If the type that is specified is not a built-in type (i.e., one of those
	// types which is defined in RFC6020/RFC7950) then we establish what the type
	// that we must actually perform the mapping for is. By default, start with
	// the type that is specified in the schema.
	if !util.IsYANGBaseType(args.yangType) {
		switch args.yangType.Kind {
		case yang.Yenum, yang.Yidentityref:
			// In the case of a typedef that specifies an enumeration or identityref
			// then generate a enumerated type in the Go code according to the contextEntry
			// which has been provided by the calling code.
			if args.contextEntry == nil {
				return nil, fmt.Errorf("error mapping node %s due to lack of context", args.yangType.Name)
			}

			tn, err := s.typedefEnumeratedName(args.contextEntry, noUnderscores)
			if err != nil {
				return nil, err
			}

			return &MappedType{
				NativeType:        fmt.Sprintf("%s%s", prefix, tn),
				IsEnumeratedValue: true,
			}, nil
		}
	}
	return nil, nil
}

// typedefEnumeratedName retrieves the generated name of the input *yang.Entry
// which represents a typedef that has an underlying enumerated type (e.g.,
// identityref or enumeration).
func (s *enumSet) typedefEnumeratedName(e *yang.Entry, noUnderscores bool) (string, error) {
	typedefKey, _, _, err := s.enumeratedTypedefKey(e, noUnderscores)
	if err != nil {
		return "", err
	}
	definedName, ok := s.uniqueEnumeratedTypedefNames[typedefKey]
	if !ok {
		return "", fmt.Errorf("enumSet: cannot retrieve type name for typedef enumeration without a name generated (was findEnumSet called?): %v", e.Path())
	}
	return definedName, nil
}

// identityBaseKey calculates a unique string key for the input identity.
func (s *enumSet) identityBaseKey(i *yang.Identity) string {
	definingModName := genutil.ParentModulePrettyName(i)
	// As per a typedef that includes an enumeration, there is a many to one
	// relationship between leaves and an identity value, therefore, we want to
	// reuse the existing name for the identity enumeration if one exists.
	return fmt.Sprintf("%s/%s", definingModName, i.Name)
}

// enumeratedTypedefKey calculates a unique string key for the input typedef
// *yang.Entry that has an underlying enumerated type (e.g., identityref or
// enumeration). It also returns the defining module and type name components
// of the identity for use in the name generation, if needed.
func (s *enumSet) enumeratedTypedefKey(e *yang.Entry, noUnderscores bool) (string, string, string, error) {
	typeName := e.Type.Name

	// Handle the case whereby we have been handed an enumeration that is within a
	// union. We need to synthesise the name of the type here such that it is based on
	// type name, plus the fact that it is an enumeration.
	if e.Type.Kind == yang.Yunion {
		enumTypes := util.EnumeratedUnionTypes(e.Type.Type)

		switch len(enumTypes) {
		case 1:
			// We specifically say that this is an enumeration within the leaf.
			if noUnderscores {
				typeName = fmt.Sprintf("%sEnum", enumTypes[0].Name)
			} else {
				typeName = fmt.Sprintf("%s_Enum", enumTypes[0].Name)
			}
		case 0:
			return "", "", "", fmt.Errorf("enumerated type had an empty union within it, path: %v, type: %v, enumerated: %v", e.Path(), e.Type, enumTypes)
		default:
			return "", "", "", fmt.Errorf("multiple enumerated types within a single enumeration not supported, path: %v, type: %v, enumerated: %v", e.Path(), e.Type, enumTypes)
		}
	}
	if e.Node == nil {
		return "", "", "", fmt.Errorf("nil Node in enum type %s", e.Name)
	}

	definingModName := genutil.ParentModulePrettyName(e.Node)
	// Since there can be many leaves that refer to the same typedef, then we do not generate
	// a name for each of them, but rather use a common name, we use the non-CamelCase lookup
	// as this is unique, whereas post-camelisation, we may have name clashes. Since a typedef
	// does not have a 'path' in Goyang, we synthesise one using the form
	// module-name/typedef-name.
	return fmt.Sprintf("%s/%s", definingModName, typeName), definingModName, typeName, nil
}

// enumLeafKey calculates a unique string key for the input leaf of type
// "enumeration" only. If compressPaths is true, it also returns the compressed
// version of the entry name for use in name generation, if needed.
func (s *enumSet) enumLeafKey(e *yang.Entry, compressPaths, noUnderscores, skipDedup bool) (string, string) {
	// uniqueIdentifier is the unique identifier used to determine whether to
	// define a new enum type for the input enum.
	// By default, using the entry's path ensures every enumeration
	// instance has its own name.
	uniqueIdentifier := e.Path()
	if !skipDedup || compressPaths {
		// However, if using compression or de-duplicating where
		// possible, then we do not use the entire path as the enum
		// name, and instead find the unique identifier that may de-dup
		// due to compression or multiple usages of a definition.
		uniqueIdentifier = enumIdentifier(e, compressPaths)
	}

	var compressName string
	if compressPaths {
		definingModName := genutil.ParentModulePrettyName(e.Node)
		// If we compress paths then the name of this enum is of the form
		// ModuleName_GrandParent_Leaf - we use GrandParent since Parent is
		// State or Config so would not be unique. The proposed name is
		// handed to genutil.MakeNameUnique to ensure that it does not clash with
		// other defined names.
		compressName = fmt.Sprintf("%s_%s_%s", yang.CamelCase(definingModName), yang.CamelCase(e.Parent.Parent.Name), yang.CamelCase(e.Name))
		if noUnderscores {
			compressName = strings.Replace(compressName, "_", "", -1)
		}

		if skipDedup {
			// If using compression and duplicating, then we add
			// compressName to the uniqueIdentifier, meaning every
			// enum instance in the compressed view of the schema
			// has its own definition.  The base enum identity is
			// still required to deal with collisions between
			// compressed enum names when they describe different
			// enums.
			uniqueIdentifier += compressName
		}
	}
	return uniqueIdentifier, compressName
}

// enumIdentifier takes in an enum entry and returns a unique identifier for
// that enum constructed using its path. This identifier would be the same for
// an enum that's used in two different places in the schema.
// This function can be called on a union entry that contains an enumeration type.
func enumIdentifier(e *yang.Entry, compressPaths bool) string {
	definingModName := genutil.ParentModulePrettyName(e.Node)
	// It is possible, given a particular enumerated leaf, for it to appear
	// multiple times in the schema. For example, through being defined in
	// a grouping which is instantiated in two places. In these cases, the
	// enumerated values must be the same since the path to the node - i.e.,
	// module/hierarchy/of/containers/leaf-name must be unique, since we
	// cannot have multiple modules of the same name, and paths within the
	// module must be unique. To this end, we check whether we are generating
	// an enumeration for exactly the same node, and if so, re-use the name
	// of the enumeration that has been generated. This improves usability
	// for the end user by avoiding multiple enumerated types.
	//
	// The path that is used for the enumeration is therefore taking the goyang
	// "Node" hierarchy - we walk back up the tree until such time as we find
	// a node that is not within the same module (ParentModulePrettyName(parent) !=
	// ParentModulePrettyName(currentNode)), and use this as the unique path.
	var identifierPathElem []string
	for elem := e.Node; elem.ParentNode() != nil && genutil.ParentModulePrettyName(elem) == definingModName; elem = elem.ParentNode() {
		identifierPathElem = append(identifierPathElem, elem.NName())
	}

	// Since the path elements are compiled from leaf back to root, then reverse them to
	// form the path, this is not strictly required, but aids debugging of the elements.
	var identifier string
	for i := len(identifierPathElem) - 1; i >= 0; i-- {
		identifier = fmt.Sprintf("%s/%s", identifier, identifierPathElem[i])
	}

	// For leaves that have an enumeration within a typedef that is within a union,
	// we do not want to just use the place in the schema definition for de-duplication,
	// since it becomes confusing for the user to have non-contextual names within
	// this context. We therefore rewrite the identifier path to have the context
	// that we are in. By default, we just use the name of the node, but in OpenConfig
	// schemas we rely on the grandparent name.
	if !util.IsYANGBaseType(e.Type) {
		idPfx := e.Name
		if compressPaths && e.Parent != nil && e.Parent.Parent != nil {
			idPfx = e.Parent.Parent.Name
		}
		identifier = fmt.Sprintf("%s%s", idPfx, identifier)
	}
	return identifier
}

// findEnumSet walks the list of enumerated value leaves and determines whether
// code generation is required for each enum. When compressPaths is true, it
// removes duplication between config and state containers, and uses simplified
// (compressed) names for enumerated leaf names. It also de-dups references to
// the same identity base, and type definitions. If noUnderscores is set to
// true, underscores are omitted from the name returned such that the
// enumerated type name is compliant with language styles where underscores are
// not allowed in names. If skipEnumDedup is set to true, we do not attempt to
// deduplicate enumerated leaves that are used more than once in the schema
// into a common type.
// The returned enumSet can be used to query for enum/identity names.
// The returned map is the set of generated enums to be used for enum code generation.
func findEnumSet(entries map[string]*yang.Entry, compressPaths, noUnderscores, skipEnumDedup bool) (*enumSet, map[string]*yangEnum, []error) {
	validEnums := make(map[string]*yang.Entry)
	var enumPaths []string
	var errs []error

	if compressPaths {
		// Don't generate output for an element that exists both in the config and state containers,
		// i.e., /interfaces/interface/config/enum and /interfaces/interface/state/enum should not
		// both have code generated for them. Since there may be containers underneath state then
		// we cannot rely on state having a specific place in the tree, therefore, walk through the
		// path and swap 'state' for 'config' where it is found allowing us to check whether the
		// state leaf has a corresponding config leaf, and if so, to ignore it. Note that a schema
		// that is a valid OpenConfig schema has only a single instance of 'config' or 'state' in
		// the path, therefore the below algorithm replaces only one element.
		for path, e := range entries {
			parts := strings.Split(path, "/")

			var newPath []string
			for _, p := range parts {
				if p == "state" {
					p = "config"
				}
				newPath = append(newPath, p)
			}
			if path == util.SlicePathToString(newPath) {
				// If the path remains the same - i.e., we did not replace state with
				// config, then the enumeration is valid, such that code should have
				// code generated for it.
				validEnums[path] = e
				enumPaths = append(enumPaths, path)
			} else {
				// Else, if we changed the path, then we changed a state container for
				// a config container, and we should check whether the config leaf
				// exists. Only when it doesn't do we consider this enum.
				if _, ok := entries[util.SlicePathToString(newPath)]; !ok {
					validEnums[path] = e
					enumPaths = append(enumPaths, path)
				}
			}
		}
	} else {
		// No de-duplication occurs when path compression is disabled.
		validEnums = entries
		for p := range validEnums {
			enumPaths = append(enumPaths, p)
		}
	}

	// Sort the name of the enums such that we have deterministic ordering. This allows the
	// same entity to be used for code generation each time (avoiding flaky tests or scenarios
	// where there are erroneous config/state differences).
	sort.Strings(enumPaths)

	s := newEnumGenState()

	// This is the first of two passes over the input enum entries.
	// The purpose of this pass is to establish what the default name of
	// each of the enumerated leaf types that are to be output in the code is.
	// At this stage, we just try and calculate the complete set, rather
	// than resolving any clashes in names that might occur.
	// Since identity and enumerated typedef name clashes are not allowed,
	// their generated names are already final.
	for _, eP := range enumPaths {
		e := validEnums[eP]
		switch {
		case e.Type.Name == "union", len(e.Type.Type) > 0 && !util.IsYANGBaseType(e.Type):
			// Calculate any enumerated types that exist within a union, whether it
			// is a directly defined union, or a non-builtin typedef.
			if err := s.resolveEnumeratedUnionEntry(e, compressPaths, noUnderscores, skipEnumDedup); err != nil {
				errs = append(errs, err)
			}
		case e.Type.Name == "identityref":
			// This is an identityref - we do not want to generate code for an
			// identityref but rather for the base identity. This means that we reduce
			// duplication across different enum types. Re-map the "path" that is to
			// be used to the new identityref name.
			if e.Type.IdentityBase == nil {
				errs = append(errs, fmt.Errorf("entry %s was an identity with a nil base", e.Name))
				continue
			}
			if err := s.resolveIdentityRefBaseType(e, noUnderscores); err != nil {
				errs = append(errs, err)
			}
		case e.Type.Name == "enumeration":
			// Calculate generated name for enumeration leaf.
			s.resolveEnumName(e, compressPaths, noUnderscores, skipEnumDedup)
		default:
			// This is a type which is defined through a typedef.
			if err := s.resolveTypedefEnumeratedName(e, noUnderscores); err != nil {
				errs = append(errs, err)
			}
		}
	}
	if errs != nil {
		return nil, nil, errs
	}

	// Resolve any enumeration value name conflicts.
	// At this point, all enumerated value names are fully resolved.
	s.resolveNameClashSets()

	// This is the second and final pass over the input enum entries.
	// During this pass, the generated names are retrieved and packaged
	// into yangEnum entries.

	// Sort the list of enums such that we can ensure when there is deduplication then the same
	// source entity is used for code generation.
	genEnums := make(map[string]*yangEnum)
	for _, eP := range enumPaths {
		e := validEnums[eP]

		switch {
		case e.Type.Name == "union", len(e.Type.Type) > 0 && !util.IsYANGBaseType(e.Type):
			// Calculate any enumerated types that exist within a union, whether it
			// is a directly defined union, or a non-builtin typedef.
			es, err := s.enumSet.enumeratedUnionEntry(e, compressPaths, noUnderscores, skipEnumDedup)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			for _, en := range es {
				if _, ok := genEnums[en.name]; !ok {
					genEnums[en.name] = en
				}
			}
		case e.Type.Name == "identityref":
			// This is an identityref - we do not want to generate code for an
			// identityref but rather for the base identity. This means that we reduce
			// duplication across different enum types. Re-map the "path" that is to
			// be used to the new identityref name.
			if e.Type.IdentityBase == nil {
				errs = append(errs, fmt.Errorf("entry %s was an identity with a nil base", e.Name))
				continue
			}
			idBaseName, err := s.enumSet.identityrefBaseTypeFromLeaf(e)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			if _, ok := genEnums[idBaseName]; !ok {
				genEnums[idBaseName] = &yangEnum{
					name:  idBaseName,
					entry: e,
				}
			}
		case e.Type.Name == "enumeration":
			// We simply want to map this enumeration into a new name. Since we do
			// de-duplication of re-used enumerated leaves at different points in
			// the schema (e.g., if openconfig-bgp/container/enum-A can be instantiated
			// in two places, then we do not want to have multiple enumerated types
			// that represent this leaf), then we do not have errors if duplicates
			// occur, we simply perform de-duplication at this stage.
			enumName, err := s.enumSet.enumName(e, compressPaths, noUnderscores, skipEnumDedup)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			if _, ok := genEnums[enumName]; !ok {
				genEnums[enumName] = &yangEnum{
					name:  enumName,
					entry: e,
				}
			}
		default:
			// This is a type which is defined through a typedef.
			typeName, err := s.enumSet.typedefEnumeratedName(e, noUnderscores)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			if _, ok := genEnums[typeName]; !ok {
				genEnums[typeName] = &yangEnum{
					name:  typeName,
					entry: e,
				}
			}
		}
	}

	return s.enumSet, genEnums, errs
}

// enumGenState contains the state and functionality for generating enum names
// that seeks to be compatible in all supported languages. It assumes that
// enums are all in the same output namespace (within the generated code), a
// conservative assumption guaranteeing that all enum names are unique in any
// language.
type enumGenState struct {
	// definedEnums keeps track of generated enum names to avoid conflicts.
	definedEnums map[string]bool
	// enumSet contains the final collision-free enumerated value names for
	// the generated code.
	enumSet *enumSet
	// enumeratedLeafNameClashSets stores the unique string keys representing
	// enumeration leaves that were mapped to the same default generated
	// name using the default name generation rules.
	enumeratedLeafNameClashSets map[string]map[string]bool
	// uniqueEnumeratedLeafEntries keeps track of which enums have already had
	// a name generated to avoid a second name from being generated for the
	// same entry.
	uniqueEnumeratedLeafEntries map[string]bool
}

// newEnumGenState creates a new enumGenState instance initialised with the
// default state required for code generation.
func newEnumGenState() *enumGenState {
	return &enumGenState{
		definedEnums:                map[string]bool{},
		enumSet:                     newEnumSet(),
		enumeratedLeafNameClashSets: map[string]map[string]bool{},
		uniqueEnumeratedLeafEntries: map[string]bool{},
	}
}

// resolveNameClashSets scans through all of the name clash sets, and carries out
// name collision resolution between different enum keys that mapped to the same default
// generated name. Then it stores these final names in their appropriate name
// maps within enumSet.
func (s *enumGenState) resolveNameClashSets() {
	s.enumSet.uniqueEnumeratedLeafNames = s.resolveNameClashSet(s.enumeratedLeafNameClashSets)
}

// resolveNameClashSet carries out name collision resolution on the input name
// clash set to generate the final names, and stores those names in the given
// unique map.
func (s *enumGenState) resolveNameClashSet(nameClashSets map[string]map[string]bool) map[string]string {
	var defaultNames []string
	for defaultName := range nameClashSets {
		defaultNames = append(defaultNames, defaultName)
	}
	// Make collision resolution deterministic.
	sort.Strings(defaultNames)
	uniqueNamesMap := map[string]string{}
	for _, name := range defaultNames {
		nameClashSet := nameClashSets[name]
		var enumKeys []string
		for enumKey := range nameClashSet {
			enumKeys = append(enumKeys, enumKey)
		}
		// Make collision deterministic.
		sort.Strings(enumKeys)
		for _, enumKey := range enumKeys {
			if _, ok := uniqueNamesMap[enumKey]; ok {
				// TODO(wenbli), next PR: Return an error here
				// as each enumKey should not be given
				// different names.
				continue
			}
			uniqueName := genutil.MakeNameUnique(name, s.definedEnums)
			uniqueNamesMap[enumKey] = uniqueName
		}
	}

	return uniqueNamesMap
}

// resolveEnumeratedUnionEntry takes an input YANG union yang.Entry and computes the enumerated
// values that should be generated for the entry. The name of the enumerated
// value is calculated based on the original context, whether path compression is enabled based
// on the compressPaths boolean, and whether the name should not include underscores, as per the
// noUnderscores boolean.
func (s *enumGenState) resolveEnumeratedUnionEntry(e *yang.Entry, compressPaths, noUnderscores, skipEnumDedup bool) error {
	for _, t := range util.EnumeratedUnionTypes(e.Type.Type) {
		switch {
		case t.IdentityBase != nil:
			if err := s.resolveIdentityrefBaseTypeFromIdentity(t.IdentityBase, noUnderscores); err != nil {
				return err
			}
		case t.Enum != nil:
			if util.IsYANGBaseType(t) {
				s.resolveEnumName(e, compressPaths, noUnderscores, skipEnumDedup)
			} else {
				if err := s.resolveTypedefEnumeratedName(e, noUnderscores); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// resolveIdentityRefBaseType calculates the mapped name of an identityref's
// base such that it can be used in generated code. The value that is returned
// is defining module name followed by the CamelCase-ified version of the
// base's name. This function wraps the resolveIdentityrefBaseTypeFromIdentity
// function since it covers the common case that the caller is interested in
// determining the name from an identityref leaf, rather than directly from the
// identity. If the noUnderscores bool is set to true, underscores are omitted
// from the name returned such that the enumerated type name is compliant
// with language styles where underscores are not allowed in names.
// No conflicts in generated identity names are allowed. Any conflict would result in an error.
func (s *enumGenState) resolveIdentityRefBaseType(idr *yang.Entry, noUnderscores bool) error {
	return s.resolveIdentityrefBaseTypeFromIdentity(idr.Type.IdentityBase, noUnderscores)
}

// resolveIdentityrefBaseTypeFromIdentity takes an input yang.Identity pointer and
// determines the name of the identity used within the generated code for it. The value
// returned is based on the defining module followed by the CamelCase-ified version
// of the identity's name. If noUnderscores is set to false, underscores are omitted
// from the name returned such that the enumerated type name is compliant with
// language styles where underscores are not allowed in names.
// No conflicts in generated identity names are allowed. Any conflict would result in an error.
func (s *enumGenState) resolveIdentityrefBaseTypeFromIdentity(i *yang.Identity, noUnderscores bool) error {
	definingModName := genutil.ParentModulePrettyName(i)

	// As per a typedef that includes an enumeration, there is a many to one
	// relationship between leaves and an identity value, therefore, we want to
	// reuse the existing name for the identity enumeration if one exists.
	identityKey := s.enumSet.identityBaseKey(i)
	if _, ok := s.enumSet.uniqueIdentityNames[identityKey]; ok {
		return nil
	}
	var name string
	if noUnderscores {
		name = fmt.Sprintf("%s%s", yang.CamelCase(definingModName), strings.Replace(yang.CamelCase(i.Name), "_", "", -1))
	} else {
		name = fmt.Sprintf("%s_%s", yang.CamelCase(definingModName), yang.CamelCase(i.Name))
	}
	// The name of an identityref base type must be unique within the entire generated
	// code, so the context of name generation is global.
	if s.definedEnums[name] {
		return fmt.Errorf("enumgen.go: identity name conflict %q for identity %+v", name, i)
	}
	s.enumSet.uniqueIdentityNames[identityKey] = name
	s.definedEnums[name] = true
	return nil
}

// resolveEnumName takes a yang.Entry and resolves its name into the type name
// that will be used in the generated code. Whilst a leaf may only be used
// in a single context (i.e., at its own path), resolveEnumName may be called
// multiple times, and hence de-duplication of unique name generation is required.
// If noUnderscores is set to true, then underscores are omitted from the
// output name.
// If the skipDedup argument is set to true, where a single enumeration is defined
// once in the input YANG, but instantiated multiple times (e.g., a grouping is
// used multiple times that contains an enumeration), then we do not attempt to
// use a single output type in the generated code for such enumerations. This allows
// the user to control whether this behaviour is useful to them -- for OpenConfig,
// it tends to be due to the state/config split - which would otherwise result in
// multiple enumerated types being produced. For other schemas, it can result in
// somewhat difficult to understand enumerated types being produced - since the first
// leaf that is processed will define the name of the enumeration.
func (s *enumGenState) resolveEnumName(e *yang.Entry, compressPaths, noUnderscores, skipDedup bool) {
	// uniqueIdentifier is the unique identifier used to determine whether to
	// define a new enum type for the input enum.
	uniqueIdentifier, compressName := s.enumSet.enumLeafKey(e, compressPaths, noUnderscores, skipDedup)

	name := compressName
	if !compressPaths {
		// If we are not compressing the paths, then we write out the entire path.
		var nbuf bytes.Buffer
		for i, p := range util.SchemaPathNoChoiceCase(e) {
			if i != 0 && !noUnderscores {
				nbuf.WriteRune('_')
			}
			nbuf.WriteString(yang.CamelCase(p))
		}
		name = nbuf.String()
	}

	if !s.uniqueEnumeratedLeafEntries[uniqueIdentifier] {
		// Each enum should only get their name generated once.
		if _, ok := s.enumeratedLeafNameClashSets[name]; !ok {
			s.enumeratedLeafNameClashSets[name] = map[string]bool{}
		}
		s.enumeratedLeafNameClashSets[name][uniqueIdentifier] = true
		s.uniqueEnumeratedLeafEntries[uniqueIdentifier] = true
	}
}

// resolveTypedefEnumeratedName takes a yang.Entry which represents a typedef
// that has an underlying enumerated type (e.g., identityref or enumeration),
// and computes the default name of the enum in the generated code.
// If its name clashes with any other identity or enumerated name, an error
// would be returned.
func (s *enumGenState) resolveTypedefEnumeratedName(e *yang.Entry, noUnderscores bool) error {
	// Since there can be many leaves that refer to the same typedef, then we do not generate
	// a name for each of them, but rather use a common name, we use the non-CamelCase lookup
	// as this is unique, whereas post-camelisation, we may have name clashes. Since a typedef
	// does not have a 'path' in Goyang, so we synthesise one using the form
	// module-name/typedef-name.
	typedefKey, definingModName, typeName, err := s.enumSet.enumeratedTypedefKey(e, noUnderscores)
	if err != nil {
		return err
	}
	if _, ok := s.enumSet.uniqueEnumeratedTypedefNames[typedefKey]; ok {
		return nil
	}
	// The module/typedefName was not already defined with a CamelCase name, so generate one
	// here, and store it to be re-used later.
	name := fmt.Sprintf("%s_%s", yang.CamelCase(definingModName), yang.CamelCase(typeName))
	if noUnderscores {
		name = strings.Replace(name, "_", "", -1)
	}

	// The name of an enumerated typedef must be unique within the entire generated
	// code, so the context of name generation is global.
	if s.definedEnums[name] {
		return fmt.Errorf("enumgen.go: enumerated typedef name conflict %q for entry %+v", name, e)
	}
	s.enumSet.uniqueEnumeratedTypedefNames[typedefKey] = name
	s.definedEnums[name] = true
	return nil
}
