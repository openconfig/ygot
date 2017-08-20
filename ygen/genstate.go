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
	"sort"
	"strings"

	"github.com/openconfig/gnmi/ctree"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygot"
)

// genState is used to store the state that is created throughout the code
// generation and must be shared between multiple entities.
type genState struct {
	// definedGlobals specifies the global Go names used during code generation.
	definedGlobals map[string]bool
	// uniqueStructNames is a map keyed by the path of a YANG entity representing a
	// struct in the generated Go code, whose value is the unique name that it
	// was mapped to. This allows routines to determine, based on a particular YANG
	// entry, how to refer to it when generating code.
	uniqueStructNames map[string]string
	// uniqueIdentityNames is a map which is keyed by a string in the form of
	// definingModule/identityName which stores the Go anme of the enumerated Go type
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
	// is the name of the Go enuerated value to which it is mapped. The path based
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
	// schematree stores a ctree.Tree structure that represents the YANG
	// schema tree. This is used for lookups within the module set where
	// they are required, e.g., for leafrefs.
	schematree *ctree.Tree
}

// newGenState creates a new genState instance, initialised with the default state
// required for code generation.
func newGenState() *genState {
	return &genState{
		// Mark the name that is used for the binary type as a reserved name
		// within the output structs.
		definedGlobals:               map[string]bool{ygot.BinaryTypeName: true},
		uniqueStructNames:            make(map[string]string),
		uniqueEnumeratedTypedefNames: make(map[string]string),
		uniqueIdentityNames:          make(map[string]string),
		uniqueEnumeratedLeafNames:    make(map[string]string),
	}
}

// enumeratedUnionEntry takes an input YANG union yang.Entry and returns the set of enumerated
// values that should be generated for the entry. New yang.Entry instances are synthesised within
// the yangGoEnums returned such that enumerations can be generated directly from the output of
// this function in common with enumerations that are not within a union. The name of the enumerated
// value is calculated based on the original context, and whether path compression is enabled based
// on the compressOCPaths boolean.
func (s *genState) enumeratedUnionEntry(e *yang.Entry, compressOCPaths bool) ([]*yangGoEnum, error) {
	var es []*yangGoEnum

	for _, t := range enumeratedUnionTypes(e.Type.Type) {
		var en *yangGoEnum
		switch {
		case t.IdentityBase != nil:
			en = &yangGoEnum{
				name: s.identityrefBaseTypeFromIdentity(t.IdentityBase),
				entry: &yang.Entry{
					Name: e.Name,
					Type: &yang.YangType{
						IdentityBase: t.IdentityBase,
					},
				},
			}
		case t.Enum != nil:
			var enumName string
			if _, chBuiltin := yang.TypeKindFromName[t.Name]; chBuiltin {
				enumName = s.resolveEnumName(e, compressOCPaths)
			} else {
				var err error
				enumName, err = s.resolveTypedefEnumeratedName(e)
				if err != nil {
					return nil, err
				}
			}

			en = &yangGoEnum{
				name: enumName,
				entry: &yang.Entry{
					Name: e.Name,
					Type: &yang.YangType{
						Enum: t.Enum,
					},
				},
			}
		}

		es = append(es, en)
	}

	return es, nil
}

// buildGoStructDefinitions extracts the yang.Entry instances from a map of
// entries that need struct definitions built for them. It resolves each
// yang.Entry to a yangStruct which contains the elements that are needed for
// subsequent code generation. The name of the struct is calculated based on
// whether path compression is enabled (via compressOCPaths), and the fake root
// entity is to be generated (genFakeRoot). If errors are encountered they are
// returned within an error slice.
func (s *genState) buildGoStructDefinitions(entries map[string]*yang.Entry, compressOCPaths, genFakeRoot bool) (map[string]*yangStruct, []error) {
	var errs []error
	mappedStructs := make(map[string]*yangStruct)

	for _, e := range entries {
		if e.IsList() || e.IsDir() || isRoot(e) {
			// This should be mapped to a struct in the generated code since it has
			// child elements in the YANG schema.
			elem := &yangStruct{
				entry: e,
			}

			// Encode the name of the element into a golang compatible struct name,
			elem.name = s.structName(e, compressOCPaths, genFakeRoot)

			// Find the elements that should be rooted on this particular entity.
			var fieldErr []error
			elem.fields, fieldErr = findAllChildren(e, compressOCPaths)
			if fieldErr != nil {
				errs = append(errs, fieldErr...)
				continue
			}

			// Determine the path of the element from the schema.
			elem.path = strings.Split(schemaTreePath(e), "/")

			// Mark this struct as the fake root if it is specified to be.
			if e.Node != nil && e.Node.NName() == rootElementNodeName {
				elem.isFakeRoot = true
			}

			// Handle structures that will represent the container which is duplicated
			// inside a list. This involves extracting the key elements of the list
			// and returning a yangListAttr structure that describes how they should
			// be represented.
			if e.IsList() {
				lattr, listErr := s.buildListKey(e, compressOCPaths)
				if listErr != nil {
					errs = append(errs, listErr...)
					continue
				}
				elem.listAttr = lattr
			}
			mappedStructs[e.Path()] = elem
		} else {
			errs = append(errs, fmt.Errorf("%s was not an element mapped to a struct", e.Path()))
		}
	}

	return mappedStructs, errs
}

// findEnumSet walks the list of enumerated value leaves and determines whether
// code generation is required for each enum. Particularly, it removes
// duplication between config and state containers when compressOCPaths is true.
// It also de-dups references to the same identity base, and type definitions.
func (s *genState) findEnumSet(entries map[string]*yang.Entry, compressOCPaths bool) (map[string]*yangGoEnum, []error) {
	validEnums := make(map[string]*yang.Entry)
	var enumNames []string
	var errs []error

	if compressOCPaths {
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
			if path == joinPath(newPath) {
				// If the path remains the same - i.e., we did not replace state with
				// config, then the enumeration is valid, such that code should have
				// code generated for it.
				validEnums[path] = e
				enumNames = append(enumNames, path)
			} else {
				// Else, if we changed the path, then we changed a state container for
				// a config container, and we should check whether the config leaf
				// exists. Only when it doesn't do we consider this enum.
				if _, ok := entries[joinPath(newPath)]; !ok {
					validEnums[path] = e
					enumNames = append(enumNames, path)
				}
			}
		}
	} else {
		// No de-duplication occurs when path compression is disabled.
		validEnums = entries
		for n := range validEnums {
			enumNames = append(enumNames, n)
		}
	}

	// Sort the name of the enums such that we have deterministic ordering. This allows the
	// same entity to be used for code generation each time (avoiding flaky tests or scenarios
	// where there are erroneous config/state differences).
	sort.Strings(enumNames)

	// Sort the list of enums such that we can ensure when there is deduplication then the same
	// source entity is used for code generation.
	genEnums := make(map[string]*yangGoEnum)
	for _, eN := range enumNames {
		e := validEnums[eN]
		_, builtin := yang.TypeKindFromName[e.Type.Name]
		switch {
		case e.Type.Name == "union", len(e.Type.Type) > 0 && !builtin:
			// Calculate any enumerated types that exist within a union, whether it
			// is a directly defined union, or a non-builtin typedef.
			es, err := s.enumeratedUnionEntry(e, compressOCPaths)
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
			idBaseName := s.resolveIdentityRefBaseType(e)
			if _, ok := genEnums[idBaseName]; !ok {
				genEnums[idBaseName] = &yangGoEnum{
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
			enumName := s.resolveEnumName(e, compressOCPaths)
			if _, ok := genEnums[enumName]; !ok {
				genEnums[enumName] = &yangGoEnum{
					name:  enumName,
					entry: e,
				}
			}
		default:
			// This is a type which is defined through a typedef.
			typeName, err := s.resolveTypedefEnumeratedName(e)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			if _, ok := genEnums[typeName]; !ok {
				genEnums[typeName] = &yangGoEnum{
					name:  typeName,
					entry: e,
				}
			}
		}
	}
	return genEnums, errs
}

// resolveIdentityRefBaseType calculates the mapped name of an identityref's
// base such that it can be used in generated code. The value that is returned
// is defining module name followed by the CamelCase-ified version of the
// base's name. This function wraps the identityrefBaseTypeFromIdentity
// function since it covers the common case that the caller is interested in
// determining the name from an identityref leaf, rather than directly from the
// identity.
func (s *genState) resolveIdentityRefBaseType(idr *yang.Entry) string {
	return s.identityrefBaseTypeFromIdentity(idr.Type.IdentityBase)
}

// identityrefBaseTypeFromIdentity takes an input yang.Identity pointer and
// determines the name of the identity used within the generated code for it. The value
// returned is based on the defining module followed by the CamelCase-ified version
// of the identity's name.
func (s *genState) identityrefBaseTypeFromIdentity(i *yang.Identity) string {
	definingModName := parentModuleName(i)

	// As per a typedef that includes an enumeration, there is a many to one
	// relationship between leaves and an identity value, therefore, we want to
	// reuse the existing name for the identity enumeration if one exists.
	identityKey := fmt.Sprintf("%s/%s", definingModName, i.Name)
	if definedName, ok := s.uniqueIdentityNames[identityKey]; ok {
		return definedName
	}
	name := fmt.Sprintf("%s_%s", yang.CamelCase(definingModName), yang.CamelCase(i.Name))
	// The name of an identityref base type must be unique within the entire generated
	// code, so the context of name generation is global.
	uniqueName := makeNameUnique(name, s.definedGlobals)
	s.uniqueIdentityNames[identityKey] = uniqueName
	return uniqueName
}

// resolveEnumName takes a yang.Entry and resolves its name into the type name
// that will be used in the generated code. Whilst a leaf may only be used
// in a single context (i.e., at its own path), resolveEnumName may be called
// multiple times, and hence de-duplication of unique name generation is required.
func (s *genState) resolveEnumName(e *yang.Entry, compressOCPaths bool) string {
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
	// a node that is not within the same module (parentModuleName(parent) !=
	// parentModuleName(currentNode)), and use this as the unique path.
	definingModName := parentModuleName(e.Node)
	var identifierPathElem []string
	for elem := e.Node; elem.ParentNode() != nil && parentModuleName(elem) == definingModName; elem = elem.ParentNode() {
		identifierPathElem = append(identifierPathElem, elem.NName())
	}

	// Since the path elements are compiled from leaf back to root, then reverse them to
	// form the path, this is not strictly required, but aids debugging of the elements.
	var identifierPath string
	for i := len(identifierPathElem) - 1; i > 0; i-- {
		identifierPath = fmt.Sprintf("%s/%s", identifierPath, identifierPathElem[i])
	}

	// If the leaf had already been encountered, then return the previously generated
	// name, rather than generating a new name.
	if definedName, ok := s.uniqueEnumeratedLeafNames[identifierPath]; ok {
		return definedName
	}

	if compressOCPaths {
		// If we compress paths then the name of this enum is of the form
		// ModuleName_GrandParent_Leaf - we use GrandParent since Parent is
		// State or Config so would not be unique. The proposed name is
		// handed to makeNameUnique to ensure that it does not clash with
		// other defined names.
		name := fmt.Sprintf("%s_%s_%s", yang.CamelCase(definingModName),
			yang.CamelCase(e.Parent.Parent.Name), yang.CamelCase(e.Name))
		uniqueName := makeNameUnique(name, s.definedGlobals)
		s.uniqueEnumeratedLeafNames[identifierPath] = uniqueName
		return uniqueName
	}

	// If this was we don't compress the paths, then we write out the entire path.
	var nbuf bytes.Buffer
	for i, p := range traverseElementSchemaPath(e) {
		if i != 0 {
			nbuf.WriteRune('_')
		}
		nbuf.WriteString(yang.CamelCase(p))
	}
	uniqueName := makeNameUnique(nbuf.String(), s.definedGlobals)
	s.uniqueEnumeratedLeafNames[identifierPath] = uniqueName
	return uniqueName
}

// resolveTypedefEnumeratedName takes a yang.Entry which represents a typedef
// that has an underlying enumerated type (e.g., identityref or enumeration),
// and resolves the name of the enum that will be generated in the corresponding
// Go code.
func (s *genState) resolveTypedefEnumeratedName(e *yang.Entry) (string, error) {
	typeName := e.Type.Name

	// Handle the case whereby we have been handed an enumeration that is within a
	// union. We need to synthesise the name of the type here such that it is based on
	// type name, plus the fact that it is an enumeration.
	if e.Type.Kind == yang.Yunion {
		enumTypes := enumeratedUnionTypes(e.Type.Type)

		switch len(enumTypes) {
		case 1:
			// We specifically say that this is an enumeration within the leaf.
			typeName = fmt.Sprintf("%s_Enum", enumTypes[0].Name)
		case 0:
			return "", fmt.Errorf("enumerated type had an empty union within it, path: %v, type: %v, enumerated: %v", e.Path(), e.Type, enumTypes)
		default:
			return "", fmt.Errorf("multiple enumerated types within a single enumeration not supported, path: %v, type: %v, enumerated: %v", e.Path(), e.Type, enumTypes)
		}
	}

	definingModName := parentModuleName(e.Node)
	// Since there can be many leaves that refer to the same typedef, then we do not generate
	// a name for each of them, but rather use a common name, we use the non-CamelCase lookup
	// as this is unique, whereas post-camelisation, we may have name clashes. Since a typedef
	// does not have a 'path' in Goyang, so we synthesise one using the form
	// module-name/typedef-name.
	typedefKey := fmt.Sprintf("%s/%s", definingModName, typeName)
	if definedName, ok := s.uniqueEnumeratedTypedefNames[typedefKey]; ok {
		return definedName, nil
	}
	// The module/typedefName was not already defined with a CamelCase name, so generate one
	// here, and store it to be re-used later.
	name := fmt.Sprintf("%s_%s", yang.CamelCase(definingModName), yang.CamelCase(typeName))
	uniqueName := makeNameUnique(name, s.definedGlobals)
	s.uniqueEnumeratedTypedefNames[typedefKey] = uniqueName
	return uniqueName, nil
}

// enumeratedTypedefTypeName resolves the name of an enumerated typedef (i.e., a typedef which is either
// an identityref or an enumeration). If the type that was supplied within the resolveTypeArgs struct
// is not a type definition which includes an enumerated type, the mappedType returned is nil, otherwise
// it is populated. Returns an error if the type does include an enumerated typedef, but this typedef is
// invalid.
func (s *genState) enumeratedTypedefTypeName(args resolveTypeArgs) (*mappedType, error) {
	// If the type that is specified is not a built-in type (i.e., one of those
	// types which is defined in RFC6020/RFC7950) then we establish what the type
	// that we must actually perform the mapping for is. By default, start with
	// the type that is specified in the schema.
	if _, builtin := yang.TypeKindFromName[args.yangType.Name]; !builtin {
		switch args.yangType.Kind {
		case yang.Yenum, yang.Yidentityref:
			// In the case of a typedef that specifies an enumeration or identityref
			// then generate a enumerated type in the Go code according to the contextEntry
			// which has been provided by the calling code.
			if args.contextEntry == nil {
				return nil, fmt.Errorf("error mapping node %s due to lack of context", args.yangType.Name)
			}

			tn, err := s.resolveTypedefEnumeratedName(args.contextEntry)
			if err != nil {
				return nil, err
			}

			return &mappedType{
				nativeType:        fmt.Sprintf("E_%s", tn),
				isEnumeratedValue: true,
			}, nil
		}
	}
	return nil, nil
}

// resolveLeafrefTarget takes an input path and context entry and
// determines the type of the leaf that is referred to by the path, such that
// it can be mapped to a native language type. Returns the yang.YangType that
// is associated with the target, and the target yang.Entry, such that the
// caller can map this to the relevant language type.
func (s *genState) resolveLeafrefTarget(path string, contextEntry *yang.Entry) (*yang.Entry, error) {
	if s.schematree == nil {
		// This should not be possible if the calling code generation is
		// well structured and builds the schematree during parsing of YANG
		// files.
		return nil, fmt.Errorf("could not map leafref path: %v, from contextEntry: %v", path, contextEntry)
	}

	fixedPath, err := fixSchemaTreePath(path, contextEntry)
	if err != nil {
		return nil, err
	}

	e := s.schematree.Get(fixedPath)
	if e == nil {
		return nil, fmt.Errorf("could not resolve leafref path: %v from %v, tree: %v", fixedPath, contextEntry, s.schematree)
	}

	target, ok := e.(*yang.Entry)
	if !ok {
		return nil, fmt.Errorf("invalid element returned from schema tree, must be a yang.Entry for path %v from %v", path, contextEntry)
	}

	return target, nil
}
