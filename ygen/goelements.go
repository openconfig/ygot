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
	"strings"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygot"
)

const (
	// goEnumPrefix is the prefix that is used for type names in the output
	// Go code, such that an enumeration's name is of the form
	//   <goEnumPrefix><EnumName>
	goEnumPrefix string = "E_"
)

var (
	// validGoBuiltinTypes stores the valid types that the Go code generation
	// produces, such that resolved types can be checked as to whether they are
	// Go built in types.
	validGoBuiltinTypes = map[string]bool{
		"int8":              true,
		"int16":             true,
		"int32":             true,
		"int64":             true,
		"uint8":             true,
		"uint16":            true,
		"uint32":            true,
		"uint64":            true,
		"float64":           true,
		"string":            true,
		"bool":              true,
		"interface{}":       true,
		ygot.BinaryTypeName: true,
		ygot.EmptyTypeName:  true,
	}

	// goZeroValues stores the defined zero value for the Go types that can
	// be used within a generated struct. It is used when leaf getters are
	// generated to return a zero value rather than the set value.
	goZeroValues = map[string]string{
		"int8":              "0",
		"int16":             "0",
		"int32":             "0",
		"int64":             "0",
		"uint8":             "0",
		"uint16":            "0",
		"uint32":            "0",
		"uint64":            "0",
		"float64":           "0.0",
		"string":            `""`,
		"bool":              "false",
		"interface{}":       "nil",
		ygot.BinaryTypeName: "nil",
		ygot.EmptyTypeName:  "false",
	}
)

// goCodeElements contains a definition of the entities within the YANG model
// that will be written out to Go code.
type goCodeElements struct {
	// packageName is the name of the Go package to be generated
	packageName string
	// structs is a map of yangDirectory definitions that is keyed by the
	// path of the entity being mapped (container, list etc.) that is being
	// described by the yangDirectory. A yangDirectory is mapped into a Go
	// struct.
	structs map[string]*yangDirectory
	// enums is a map of the enumerated values that are to be written out
	// in the Go code from the YANG schema. Each is described by a yangEnum
	// struct, and the map is keyed by the enumerated value identifier. For
	// an in-line enumeration in the YANG, this identifier is the enumeration
	// leaf's path; for a typedef it is the name of the typedef (which
	// represents an enumeration or identityref); and for an identitref it
	// is the name of the base of the identityref.
	enums map[string]*yangEnum
}

// mappedType is used to store the Go type that a leaf entity in YANG is
// mapped to. The nativeType is always populated for any leaf. unionTypes is populated
// when the type may have subtypes (i.e., is a union). enumValues is populated
// when the type is an enumerated type.
//
// The code generation explicitly maps YANG types to corresponding Go types. In
// the case that an explicit mapping is not specified, a type will be mapped to
// an empty interface (interface{}). For an explicit list of types that are
// supported, see the yangTypeToGoType function in this file.
type mappedType struct {
	// nativeType is the type which is to be used for the mapped entity.
	nativeType string
	// unionTypes is a map, keyed by the Go type, of the types specified
	// as valid for a union. The value of the map indicates the order
	// of the type, since order is important for unions in YANG. Where
	// two types are mapped to the same Go type (e.g., string) then
	// only the order of the first is maintained. Since the generated
	// code from the structs maintains only type validation, this
	// is not currently a limitation.
	unionTypes map[string]int
	// isEnumeratedValue specifies whether the nativeType that is returned
	// is a generated enumerated value. Such entities are reflected as
	// derived types with constant values, and are hence not represented
	// as pointers in the output code.
	isEnumeratedValue bool
	// zeroValue stores the value that should be used for the type if
	// it is unset. This is used only in contexts where the nil pointer
	// cannot be used, such as leaf getters.
	zeroValue string
	// defaultValue stores the default value for the type if is specified.
	// It is represented as a string pointer to ensure that default values
	// of the empty string can be distinguished from unset defaults.
	defaultValue *string
}

// resolveTypeArgs is a structure used as an input argument to the yangTypeToGoType
// function which allows extra context to be handed on. This provides the ability
// to use not only the YangType but also the yang.Entry that the type was part of
// to resolve the possible type name.
type resolveTypeArgs struct {
	// yangType is a pointer to the yang.YangType that is to be mapped.
	yangType *yang.YangType
	// contextEntry is an optional yang.Entry which is supplied where a
	// type requires knowledge of the leaf that it is used within to be
	// mapped. For example, where a leaf is defined to have a type of a
	// user-defined type (typedef) that in turn has enumerated values - the
	// context of the yang.Entry is required such that the leaf's context
	// can be established.
	contextEntry *yang.Entry
}

// TODO(robjs): When adding support for other language outputs, we should restructure
// the code such that we do not have genState receivers here, but rather pass in the
// generated state as a parameter to the function that is being called.

// pathToCamelCaseName takes an input yang.Entry and outputs its name as a Go compatible
// name in the form PathElement1_PathElement2, performing schema compression
// if required. The name is not checked for uniqueness. The genFakeRoot boolean
// specifies whether the fake root exists within the schema such that it can be
// handled specifically in the path generation.
func (s *genState) pathToCamelCaseName(e *yang.Entry, compressOCPaths, genFakeRoot bool) string {
	var pathElements []*yang.Entry

	if genFakeRoot && e.Node != nil && e.Node.NName() == rootElementNodeName {
		// Handle the special case of the root element if it exists.
		pathElements = []*yang.Entry{e}
	} else {
		// Determine the set of elements that make up the path back to the root of
		// the element supplied.
		element := e
		for element != nil {
			// If the CompressOCPaths option is set to true, then only append the
			// element to the path if the element itself would have code generated
			// for it - this compresses out surrounding containers, config/state
			// containers and root modules.
			if compressOCPaths && isOCCompressedValidElement(element) || !compressOCPaths && !isChoiceOrCase(element) {
				pathElements = append(pathElements, element)
			}
			element = element.Parent
		}
	}

	// Iterate through the pathElements slice backwards to build up the name
	// of the form CamelCaseElementOne_CamelCaseElementTwo.
	var buf bytes.Buffer
	for i := range pathElements {
		idx := len(pathElements) - 1 - i
		buf.WriteString(entryCamelCaseName(pathElements[idx]))
		if idx != 0 {
			buf.WriteRune('_')
		}
	}

	return buf.String()
}

// goStructName generates the name to be used for a particular YANG schema element
// in the generated Go code. If the compressOCPaths boolean is set to true,
// schemapaths are compressed, otherwise the name is returned simply as camel
// case. The genFakeRoot boolean specifies whether the fake root is to be generated
// such that the struct name can consider the fake root entity specifically.
func (s *genState) goStructName(e *yang.Entry, compressOCPaths, genFakeRoot bool) string {
	uniqName := makeNameUnique(s.pathToCamelCaseName(e, compressOCPaths, genFakeRoot), s.definedGlobals)

	// Record the name of the struct that was unique such that it can be referenced
	// by path.
	s.uniqueDirectoryNames[e.Path()] = uniqName

	return uniqName
}

// makeNameUnique makes the name specified as an argument unique based on the names
// already defined within a particular context which are specified within the
// definedNames map. If the name has already been defined, an underscore is appended
// to the name until it is unique.
func makeNameUnique(name string, definedNames map[string]bool) string {
	for {
		if _, nameUsed := definedNames[name]; !nameUsed {
			definedNames[name] = true
			return name
		}
		name = fmt.Sprintf("%s_", name)
	}
}

// entryCamelCaseName returns the camel case version of the Entry Name field, or
// the CamelCase name that is specified by a "camelcase-name" extension on the
// field. The returned name is not guaranteed to be unique within any context.
func entryCamelCaseName(e *yang.Entry) string {
	if name, ok := camelCaseNameExt(e.Exts); ok {
		return name
	}
	return yang.CamelCase(e.Name)
}

// camelCaseNameExt returns the CamelCase name from the slice of extensions, if
// one of the extensions is named "camelcase-name". It returns the a string
// containing the name if the bool return argumnet is set to true; otherwise no
// such extension was specified.
func camelCaseNameExt(exts []*yang.Statement) (string, bool) {
	// Check the extensions to determine whethere an extension
	// exists that specifies the camelcase name of the entity. If so
	// use this as the name in the structs.
	// TODO(robjs): Add more robust parsing into goyang such that rather
	// than having a Statement here, we have some more concrete type to
	// parse within the Extras field. This would allow robust validation
	// of the module in which the extension is defined.
	var name string
	var ok bool
	r := strings.NewReplacer(`\n`, ``, `"`, ``)
	for _, s := range exts {
		if p := strings.Split(s.Keyword, ":"); len(p) < 2 || p[1] != "camelcase-name" || !s.HasArgument {
			continue
		}
		name = r.Replace(s.Argument)
		ok = true
		break
	}
	return name, ok
}

// findAllChildrenWithoutCompression finds the entries that are children of an
// entry e, when not compressing paths. It does not recurse into any child nodes
// other than those that do not represent data tree nodes (i.e., choice and
// case nodes). Choice and case nodes themselves are not appended to the children
// list. If the excludeState argument is set to true, children that are
// config false (i.e., read only) in the YANG schema are not returned.
func findAllChildrenWithoutCompression(e *yang.Entry, excludeState bool) (map[string]*yang.Entry, []error) {
	var errs []error
	directChildren := map[string]*yang.Entry{}
	for _, child := range children(e) {
		// Exclude children that are config false if requested.
		if excludeState && !isConfig(child) {
			continue
		}

		// For each child, if it is a case or choice, then find the set of nodes that
		// are not choice or case nodes and append them to the directChildren map,
		// so they are effectively skipped over.
		if isChoiceOrCase(child) {
			errs = addNonChoiceChildren(directChildren, child, errs)
		} else {
			errs = addNewChild(directChildren, child.Name, child, errs)
		}
	}
	return directChildren, errs
}

// findAllChildren finds the data tree elements that are children of a YANG entry e, which
// should have code generated for them. In general, this means data tree elements that are
// directly connected to a particular data tree element, however, when compression of the
// schema is enabled then recursion is required.
//
// For example, if we have a YANG tree:
//    /interface (list)
//    /interface/config (container)
//    /interface/config/admin-state (leaf)
//    /interface/state (container)
//    /interface/state/admin-state (leaf)
//    /interface/state/oper-state (leaf)
//    /interface/state/counters (container)
//    /interface/state/counters/in-pkts (leaf)
//    /interface/ethernet/config (container)
//    /interface/ethernet/config/mac-address (leaf)
//    /interface/ethernet/state (state)
//    /interface/ethernet/state/mac-address (leaf)
//    /interface/subinterfaces (container)
//    /interface/subinterfaces/subinterface (list)
//
// With compression disabled, then each directly connected child of a container should have
// code generated for it - so therefore we end up with:
//
//    /interface: config, state, ethernet, subinterfaces
//    /interface/config: admin-state
//    /interface/state: admin-state, oper-state, counters
//    /interface/state/counters: in-pkts
//    /interface/ethernet: config, state
//    /interface/ethernet/config: mac-address
//    /interface/ethernet/state: mac-address
//    /interface/subinterfaces: subinterface
//
// This is simply achieved by examining the directory provided by goyang (e.Dir)
// and extracting the direct children that exist. These are appended to the directChildren
// map (keyed on element name) and then returned.
//
// When CompressOCPaths in YANGCodeGenerator is set to true, then more complex logic is
// required based on the OpenConfig path rules. In this case, the following "look-aheads" are
// implemented:
//
//  1. The 'config' and 'state' containers under a directory are removed. This is because
//     OpenConfig duplicates nodes under config and state to represent intended versus
//     applied configuration. In the compressed schema then we do not care about the intended
//     configuration leaves (those leaves that are defined as the set under the 'state' container
//     that do not exist within the 'config' container). The logic implemented is to recurse into
//     the config container, and select these leaves as direct children of the original parent.
//     Any leaves that do not exist in the 'config' container but do within 'state' are operation
//     state leaves, and hence are also mapped.
//
//     Above, this means that /interfaces/interface has the admin-state and oper-state as direct
//     children.
//
//     Since containers can exist under the 'state' container, then these containers are also
//     considered as direct children of e.
//
//  2. Surrounding containers for lists are removed - that is to say, in an OpenConfig schema
//     a list (e.g. /interface/subinterfaces/subinterface) always has a container that surrounds
//     it. This is due to implementation requirements when it is supported on vendor devices.
//     However, to a developer this looks like stuttering, and hence we remove this - by checking
//     that for each directory that would be a child of e, if it has only one child, which is
//     a list, then we skip over it.
//
// Implementing these two rules means that the schema is simplified, such that the tree described
// becomes:
//
//	/interface: admin-state, oper-state, counters, ethernet, subinterface
//	/interface/counters: in-pkts
//	/interface/ethernet: mac-address
//
// As can be seen the advantage of this compression is that the set of entities for which code
// generation is done is smaller, with less levels of schema hierarchy. However, it depends upon
// a number of rules of the OpenConfig schema. If CompressOCPaths is set to true and the schema
// does not comply with the rules of OpenConfig schema, then errors may occur and be returned
// in the []error slice by findAllChildren.
//
// It should be noted that special handling is required for choice and case - because these are
// directories within the resulting schema, but they are not data tree nodes. So for example,
// we can have:
//	/container/choice/case-one/leaf-a
//	/container/choice/case-two/leaf-b
// In this tree, "choice" and "case-one" (which are choice and case nodes respectively) are not
// valid data tree elements, so we recurse down both of the branches of "choice" to return leaf-a
// and leaf-b. Since choices can be nested (/choice-a/choice-b/choice-c/case-a), and can have
// multiple data nodes per case, then the addNonChoiceChildren function will find the first
// children of the specified node that are not choice or case statements themselves (i.e., leaf-a
// and leaf-b in the above example).
//
// The excludeState argument further filters the returned set of children
// based on their YANG 'config' status. When excludeState is true, then
// any read-only (config false) node is excluded from the returned set of children.
// The 'config' status is inherited from a entry's parent if required, as per
// the rules in RFC6020.
func findAllChildren(e *yang.Entry, compressOCPaths, excludeState bool) (map[string]*yang.Entry, []error) {
	// If we are asked to exclude 'config false' leaves, and this node is
	// config false itself, then we can return an empty set of children since
	// config false is inherited from the parent by all children.
	if excludeState && !isConfig(e) {
		return nil, nil
	}

	// If compression is not required, then we do not need to recurse into as many
	// nodes, so return simply the first level direct children (other than choice or case).
	if !compressOCPaths {
		return findAllChildrenWithoutCompression(e, excludeState)
	}

	// orderedChildNames is used to provide an ordered list of the name of children
	// to check.
	var orderedChildNames []string

	// If this is a directory and it has a container named "config" underneath
	// it then we must process this first. This is due to the fact that in the
	// schema there are duplicated leaves under config/ and state/ - and we want
	// to provide the 'config' version of them to the mapping code. This is
	// important as we care about the path that is handed to code that subsequently
	// maps back to the uncompressed schema.
	//
	// To achieve this then we build an orderedChildNames slice which specifies the
	// order in which we should process the children of entry e.
	if e.IsContainer() || e.IsList() {
		if _, ok := e.Dir["config"]; ok {
			orderedChildNames = append(orderedChildNames, "config")
		}
	}

	// For all other entries in the directory, then append them after "config"
	// (appended above) to the orderedChildren list.
	for _, child := range children(e) {
		if child.Name != "config" {
			orderedChildNames = append(orderedChildNames, child.Name)
		}
	}

	// Errors encountered during the extraction of the elements that should
	// be direct children of the entity representing e.
	var errs []error
	// directChildren is used to store the nodes that will be mapped to be direct
	// children of the struct that represents the entry e being processed. It is
	// keyed by the name of the child YANG node ((yang.Entry).Name).
	directChildren := make(map[string]*yang.Entry)
	for _, currChild := range orderedChildNames {
		switch {
		// If config false values are being excluded, and this child is config
		// false, then simply skip it from being considered. This check is performed
		// first to avoid comparisons on this node which are irrelevant.
		case excludeState && !isConfig(e.Dir[currChild]):
			continue
		// Implement rule 1 from the function documentation - skip over config and state
		// containers.
		case isConfigState(e.Dir[currChild]):
			// Recurse into this directory so that we extract its children and
			// present them as being at a higher-layer. This allows the "config"
			// and "state" container to be removed from the schema.
			// For example, /foo/bar/config/{a,b,c} becomes /foo/bar/{a,b,c}.
			for _, configStateChild := range children(e.Dir[currChild]) {
				// If we get an error for the state container then we ignore it as we
				// expect that there are duplicates here for applied configuration leaves
				// (those that appear both in the "config" and "state" container).
				if e.Dir[currChild].Name == "state" {
					// Ensure that choice/case nodes that are in the state container only
					// do not get mapped. This is again specifically for the OpenConfig\
					// routing policy model. We must ignore the error that is returned
					// in this case, since if the choice/case is already defined in the
					// config container then it will be duplicate.
					if isChoiceOrCase(configStateChild) {
						_ = addNonChoiceChildren(directChildren, configStateChild, nil)
					} else {
						_ = addNewChild(directChildren, configStateChild.Name, configStateChild, nil)
					}
				} else {
					// Handle the specific case of having a choice underneath a config
					// or state container as this occurs in the routing policy model.
					if isChoiceOrCase(configStateChild) {
						errs = addNonChoiceChildren(directChildren, configStateChild, errs)
					} else {
						errs = addNewChild(directChildren, configStateChild.Name, configStateChild, errs)
					}
				}
			}
		case e.Dir[currChild].IsDir():
			// This is a directory that is not a config or state directory, so it is
			// either purely hierarchical or a surrounding container for a list.
			///
			// e.Dir[currChild] is the first level child of the container that we're looking at
			// which is any directory in the YANG schema that is not a "config" or
			// "state" container, as well as choice/case nodes, since these also
			// contain child nodes.
			//
			// eGrandChildren is a slice of the elements that are children of the
			// directory that was a child of e.
			eGrandChildren := children(e.Dir[currChild])
			switch {
			// Implement rule 2 - remove surrounding containers for lists and consider
			// the list under the surrounding container a direct child.
			case len(eGrandChildren) == 1 && eGrandChildren[0].IsList():
				if !isConfig(eGrandChildren[0]) && excludeState {
					// If the list child is read-only, then it is not a valid child.
					continue
				}
				errs = addNewChild(directChildren, eGrandChildren[0].Name, eGrandChildren[0], errs)
			// See note in function documentation about choice and case nodes - which are
			// not valid data tree elements. We therefore skip past any number of nested
			// choice/case statements and treat the first data tree elements as direct children.
			case isChoiceOrCase(e.Dir[currChild]):
				errs = addNonChoiceChildren(directChildren, e.Dir[currChild], errs)
			default:
				// This is simply a normal container so map it into the hierarchy
				// as a direct child.
				errs = addNewChild(directChildren, e.Dir[currChild].Name, e.Dir[currChild], errs)
			}
		default:
			// This is a leaf node - but we want to ignore leafref nodes that are
			// within a list because these are duplicated keys.
			if !(e.IsList() && e.Dir[currChild].Type.Kind == yang.Yleafref) {
				errs = addNewChild(directChildren, e.Dir[currChild].Name, e.Dir[currChild], errs)
			}
		}
	}
	return directChildren, errs
}

// addNonChoiceChildren recurses into a yang.entry e and finds the first
// nodes that are neither choice nor case nodes. It appends these to the map of
// yang.Entry nodes specified by m. If errors are encountered when adding an
// element, an error is appended to the errs slice, which is returned by the
// function.
func addNonChoiceChildren(m map[string]*yang.Entry, e *yang.Entry, errs []error) []error {
	nch := make(map[string]*yang.Entry)
	findFirstNonChoice(e, nch)
	for _, n := range nch {
		errs = addNewChild(m, n.Name, n, errs)
	}
	return errs
}

// addNewChild adds a new key (k) to a map with value v if k is not already
// defined in the map. When the key k is defined in the map an error is appended
// to errs, which is subsequently returned.
func addNewChild(m map[string]*yang.Entry, k string, v *yang.Entry, errs []error) []error {
	if _, ok := m[k]; !ok {
		m[k] = v
		return errs
	}
	errs = append(errs, fmt.Errorf("%s was duplicate", v.Path()))
	return errs
}

// yangTypeToGoType takes a yang.YangType (YANG type definition) and maps it
// to the type that should be used to represent it in the generated Go code.
// A resolveTypeArgs structure is used as the input argument which specifies a
// pointer to the YangType; and optionally context required to resolve the name
// of the type. The compressOCPaths argument specifies whether compression of
// OpenConfig paths is to be enabled.
func (s *genState) yangTypeToGoType(args resolveTypeArgs, compressOCPaths bool) (*mappedType, error) {
	defVal := typeDefaultValue(args.yangType)
	// Handle the case of a typedef which is actually an enumeration.
	mtype, err := s.enumeratedTypedefTypeName(args, goEnumPrefix, false)
	if err != nil {
		// err is non nil when this was a typedef which included
		// an invalid enumerated type.
		return nil, err
	}

	if mtype != nil {
		// mtype is set to non-nil when this was a valid enumeration
		// within a typedef. We explicitly set the zero and default values
		// here.
		mtype.zeroValue = "0"
		if defVal != nil {
			mtype.defaultValue = enumDefaultValue(mtype.nativeType, *defVal, goEnumPrefix)
		}

		return mtype, nil
	}

	// Perform the actual mapping of the type to the Go type.
	switch args.yangType.Kind {
	case yang.Yint8:
		return &mappedType{nativeType: "int8", zeroValue: goZeroValues["int8"], defaultValue: defVal}, nil
	case yang.Yint16:
		return &mappedType{nativeType: "int16", zeroValue: goZeroValues["int16"], defaultValue: defVal}, nil
	case yang.Yint32:
		return &mappedType{nativeType: "int32", zeroValue: goZeroValues["int32"], defaultValue: defVal}, nil
	case yang.Yint64:
		return &mappedType{nativeType: "int64", zeroValue: goZeroValues["int64"], defaultValue: defVal}, nil
	case yang.Yuint8:
		return &mappedType{nativeType: "uint8", zeroValue: goZeroValues["uint8"], defaultValue: defVal}, nil
	case yang.Yuint16:
		return &mappedType{nativeType: "uint16", zeroValue: goZeroValues["uint16"], defaultValue: defVal}, nil
	case yang.Yuint32:
		return &mappedType{nativeType: "uint32", zeroValue: goZeroValues["uint32"], defaultValue: defVal}, nil
	case yang.Yuint64:
		return &mappedType{nativeType: "uint64", zeroValue: goZeroValues["uint64"], defaultValue: defVal}, nil
	case yang.Ybool:
		return &mappedType{nativeType: "bool", zeroValue: goZeroValues["bool"], defaultValue: defVal}, nil
	case yang.Yempty:
		// Empty is a YANG type that either exists or doesn't, therefore
		// map it to a boolean to indicate its presence or not. The empty
		// type name uses a specific name in the generated code, such that
		// it can be identified for marshalling.
		return &mappedType{nativeType: ygot.EmptyTypeName, zeroValue: goZeroValues[ygot.EmptyTypeName]}, nil
	case yang.Ystring:
		return &mappedType{nativeType: "string", zeroValue: goZeroValues["string"], defaultValue: defVal}, nil
	case yang.Yunion:
		// A YANG Union is a leaf that can take multiple values - its subtypes need
		// to be extracted.
		return s.goUnionType(args, compressOCPaths)
	case yang.Yenum:
		// Enumeration types need to be resolved to a particular data path such
		// that a created enumered Go type can be used to set their value. Hand
		// the leaf to the resolveEnumName function to determine the name.
		if args.contextEntry == nil {
			return nil, fmt.Errorf("cannot map enum without context")
		}
		n := s.resolveEnumName(args.contextEntry, compressOCPaths, false)
		if defVal != nil {
			defVal = enumDefaultValue(n, *defVal, "")
		}
		return &mappedType{
			nativeType:        fmt.Sprintf("E_%s", n),
			isEnumeratedValue: true,
			zeroValue:         "0",
			defaultValue:      defVal,
		}, nil
	case yang.Yidentityref:
		// Identityref leaves are mapped according to the base identity that they
		// refer to - this is stored in the IdentityBase field of the context leaf
		// which is determined by the resolveIdentityRefBaseType.
		if args.contextEntry == nil {
			return nil, fmt.Errorf("cannot map identityref without context")
		}
		n := s.resolveIdentityRefBaseType(args.contextEntry, false)
		if defVal != nil {
			defVal = enumDefaultValue(n, *defVal, "")
		}
		return &mappedType{
			nativeType:        fmt.Sprintf("E_%s", n),
			isEnumeratedValue: true,
			zeroValue:         "0",
			defaultValue:      defVal,
		}, nil
	case yang.Ydecimal64:
		return &mappedType{nativeType: "float64", zeroValue: goZeroValues["float64"]}, nil
	case yang.Yleafref:
		// This is a leafref, so we check what the type of the leaf that it
		// references is by looking it up in the schematree.
		target, err := s.resolveLeafrefTarget(args.yangType.Path, args.contextEntry)
		if err != nil {
			return nil, err
		}
		return s.yangTypeToGoType(resolveTypeArgs{yangType: target.Type, contextEntry: target}, compressOCPaths)
	case yang.Ybinary:
		// Map binary fields to the Binary type defined in the output code,
		// this is used to ensure that we can distinguish a binary field from
		// a leaf-list of uint8s which is not possible if mapping to []byte.
		return &mappedType{nativeType: ygot.BinaryTypeName, zeroValue: goZeroValues[ygot.BinaryTypeName], defaultValue: defVal}, nil
	default:
		// Return an empty interface for the types that we do not currently
		// support. Back-end validation is required for these types.
		// TODO(robjs): Missing types currently bits. These
		// should be added.
		return &mappedType{nativeType: "interface{}", zeroValue: goZeroValues["interface{}"]}, nil
	}
}

// goUnionType maps a YANG union to a set of Go types that should be used to
// represent it. In the simple case that the union contains only one
// subtype - e.g., is a union of string, string then the single type that
// is contained is returned as the type to be used in the generated Go code.
// This situation is common in cases that there are two strings that have
// different patterns (e.g., inet:ip-address defines two strings one matching
// the IPv4 address regexp, and the other IPv6).
//
// In the more complex case that the union consists of multiple types (e.g.,
// string, int8) then the type that is returned corresponds to a new type
// which directly relates to the path of the element. This type is intended to
// be mapped to an interface which can be implemented for each sub-type.
//
// For example:
//	container bar {
//		leaf foo {
//			type union {
//				type string;
//				type int8;
//			}
//		}
//	}
//
// Is returned with a goType of Bar_Foo_Union (where Bar_Foo is the schema
// path to an element). The unionTypes are specified to be string and int8.
//
// The compressOCPaths argument specifies whether OpenConfig path compression
// is enabled such that the name of enumerated types can be calculated correctly.
//
// goUnionType returns an error if mapping is not possible.
func (s *genState) goUnionType(args resolveTypeArgs, compressOCPaths bool) (*mappedType, error) {
	var errs []error
	unionTypes := make(map[string]int)

	// Extract the subtypes that are defined into a map which is keyed on the
	// mapped type. A map is used such that other functions that rely checking
	// whether a particular type is valid when creating mapping code can easily
	// check, rather than iterating the slice of strings.
	for _, subtype := range args.yangType.Type {
		errs = append(errs, s.goUnionSubTypes(subtype, args.contextEntry, unionTypes, compressOCPaths)...)
	}

	if errs != nil {
		return nil, fmt.Errorf("errors mapping element: %v", errs)
	}

	// Zero value is set to nil, other than in cases where there is a single type in
	// the union.
	zeroValue := "nil"

	nativeType := fmt.Sprintf("%s_Union", s.pathToCamelCaseName(args.contextEntry, compressOCPaths, false))
	if len(unionTypes) == 1 {
		for mappedType := range unionTypes {
			nativeType = mappedType
		}
		if zv, ok := goZeroValues[nativeType]; ok {
			zeroValue = zv
		}

	}

	return &mappedType{
		nativeType: nativeType,
		unionTypes: unionTypes,
		zeroValue:  zeroValue,
	}, nil
}

// goUnionSubTypes extracts all the possible subtypes of a YANG union leaf,
// and returns map keyed by the mapped type along with any errors that occur. A
// map is returned in preference to a slice such that it is easier for calling
// functions to check whether a particular type is a valid type for a leaf. Since
// a union itself may contain unions, the supplied union is recursed. The
// compressOCPaths argument specifies whether OpenConfig path compression is enabled
// such that the name of enumerated types can be correctly calculated.
func (s *genState) goUnionSubTypes(subtype *yang.YangType, ctx *yang.Entry, currentTypes map[string]int, compressOCPaths bool) []error {
	var errs []error
	// If subtype.Type is not empty then this means that this type is defined to
	// be a union itself.
	if subtype.Type != nil {
		for _, st := range subtype.Type {
			errs = append(errs, s.goUnionSubTypes(st, ctx, currentTypes, compressOCPaths)...)
		}
		return errs
	}

	var mtype *mappedType
	switch subtype.Kind {
	case yang.Yidentityref:
		// Handle the specific case that the context entry is now not the correct entry
		// to map enumerated types to their module. This occurs in the case that the subtype
		// is an identityref - in this case, the context entry that we are carrying is the
		// leaf that refers to the union, not the specific subtype that is now being examined.
		mtype = &mappedType{
			nativeType: fmt.Sprintf("E_%s", s.identityrefBaseTypeFromIdentity(subtype.IdentityBase, false)),
			zeroValue:  "0",
		}
	default:
		var err error

		mtype, err = s.yangTypeToGoType(resolveTypeArgs{yangType: subtype, contextEntry: ctx}, compressOCPaths)
		if err != nil {
			errs = append(errs, err)
			return errs
		}
	}

	// Only append the type if it not one that is currently in the
	// list. To map the structure we don't care if there are two
	// typedefs that are strings underneath, as the Go code will
	// simply represent this as one string.
	if _, ok := currentTypes[mtype.nativeType]; !ok {
		currentTypes[mtype.nativeType] = len(currentTypes)
	}
	return errs
}

// buildListKey takes a yang.Entry, e, corresponding to a list and extracts the definition
// of the list key, returning a yangListAttr struct describing the key element(s). If
// errors are encountered during the extraction, they are returned as a slice of errors.
// The yangListAttr that is returned consists of a map, keyed by the key leaf's YANG
// identifier, with a value of a mappedType struct which indicates how that key leaf
// is to be represented in Go. The key elements themselves are returned in the keyElems
// slice.
func (s *genState) buildListKey(e *yang.Entry, compressOCPaths bool) (*yangListAttr, []error) {
	if !e.IsList() {
		return nil, []error{fmt.Errorf("%s is not a list", e.Name)}
	}

	if e.Key == "" {
		// A null key is not valid if we have a config true list, so return an error
		if isConfig(e) {
			return nil, []error{fmt.Errorf("No key specified for a config true list: %s", e.Name)}
		}
		// This is a keyless list so return an empty yangListAttr but no error, downstream
		// mapping code should consider this to mean that this should be mapped into a
		// keyless structure (i.e., a slice).
		return nil, nil
	}

	listattr := &yangListAttr{
		keys: make(map[string]*mappedType),
	}

	var errs []error
	keys := strings.Split(e.Key, " ")
	for _, k := range keys {
		// Extract the key leaf itself from the Dir of the list element. Dir is populated
		// by goyang, and is a map keyed by leaf identifier with values of a *yang.Entry
		// corresponding to the leaf.
		keyleaf, ok := e.Dir[k]
		if !ok {
			return nil, []error{fmt.Errorf("Key %s did not exist for %s", k, e.Name)}
		}

		if keyleaf.Type != nil {
			switch keyleaf.Type.Kind {
			case yang.Yleafref:
				// In the case that the key leaf is a YANG leafref, then in OpenConfig
				// this means that the key is a pointer to an element under 'config' or
				// 'state' under the list itself. In the case that this is not an OpenConfig
				// compliant schema, then it may be a leafref to some other element in the
				// schema. Therefore, when the key is a leafref for the OC case, then
				// find the actual leaf that it points to, for other schemas, then ignore
				// this lookup.
				if compressOCPaths {
					// keyleaf.Type.Path specifies the (goyang validated) path to the
					// leaf that is the target of the reference when the keyleaf is a
					// leafref.
					refparts := strings.Split(keyleaf.Type.Path, "/")
					if len(refparts) < 2 {
						return nil, []error{fmt.Errorf("Key %s had an invalid path %s", k, keyleaf.Path())}
					}
					// In the case of OpenConfig, the list key is specified to be under
					// the 'config' or 'state' container of the list element (e). To this
					// end, we extract the name of the config/state container. However, in
					// some cases, it can be prefixed, so we need to remove the prefixes
					// from the path.
					dir := removePrefix(refparts[len(refparts)-2])
					d, ok := e.Dir[dir]
					if !ok {
						return nil, []error{
							fmt.Errorf("Key %s had a leafref key (%s) in dir %s that did not exist (%v)",
								k, keyleaf.Path(), dir, refparts),
						}
					}
					targetLeaf := removePrefix(refparts[len(refparts)-1])
					if _, ok := d.Dir[targetLeaf]; !ok {
						return nil, []error{
							fmt.Errorf("Key %s had leafref key (%s) that did not exist at (%v)", k, keyleaf.Path(), refparts),
						}
					}
					keyleaf = d.Dir[targetLeaf]
				}
			}
		}

		listattr.keyElems = append(listattr.keyElems, keyleaf)
		keyType, err := s.yangTypeToGoType(resolveTypeArgs{yangType: keyleaf.Type, contextEntry: keyleaf}, compressOCPaths)
		if err != nil {
			errs = append(errs, err)
		}
		listattr.keys[keyleaf.Name] = keyType
	}

	return listattr, errs
}
