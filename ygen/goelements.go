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
	"github.com/openconfig/ygot/genutil"
	"github.com/openconfig/ygot/util"
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

// MappedType is used to store the Go type that a leaf entity in YANG is
// mapped to. The NativeType is always populated for any leaf. UnionTypes is populated
// when the type may have subtypes (i.e., is a union). enumValues is populated
// when the type is an enumerated type.
//
// The code generation explicitly maps YANG types to corresponding Go types. In
// the case that an explicit mapping is not specified, a type will be mapped to
// an empty interface (interface{}). For an explicit list of types that are
// supported, see the yangTypeToGoType function in this file.
type MappedType struct {
	// NativeType is the type which is to be used for the mapped entity.
	NativeType string
	// UnionTypes is a map, keyed by the Go type, of the types specified
	// as valid for a union. The value of the map indicates the order
	// of the type, since order is important for unions in YANG. Where
	// two types are mapped to the same Go type (e.g., string) then
	// only the order of the first is maintained. Since the generated
	// code from the structs maintains only type validation, this
	// is not currently a limitation.
	UnionTypes map[string]int
	// IsEnumeratedValue specifies whether the NativeType that is returned
	// is a generated enumerated value. Such entities are reflected as
	// derived types with constant values, and are hence not represented
	// as pointers in the output code.
	IsEnumeratedValue bool
	// ZeroValue stores the value that should be used for the type if
	// it is unset. This is used only in contexts where the nil pointer
	// cannot be used, such as leaf getters.
	ZeroValue string
	// DefaultValue stores the default value for the type if is specified.
	// It is represented as a string pointer to ensure that default values
	// of the empty string can be distinguished from unset defaults.
	DefaultValue *string
}

// IsYgenDefinedGoType returns true if the native type of a MappedType is a type that's
// defined by ygen's generated code.
func IsYgenDefinedGoType(t *MappedType) bool {
	return t.IsEnumeratedValue || len(t.UnionTypes) >= 2 || t.NativeType == ygot.BinaryTypeName || t.NativeType == ygot.EmptyTypeName
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

// pathToCamelCaseName takes an input yang.Entry and outputs its name as a Go
// compatible name in the form PathElement1_PathElement2, performing schema
// compression if required. The name is not checked for uniqueness. The
// genFakeRoot boolean specifies whether the fake root exists within the schema
// such that it can be handled specifically in the path generation.
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
			if compressOCPaths && util.IsOCCompressedValidElement(element) || !compressOCPaths && !util.IsChoiceOrCase(element) {
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
		buf.WriteString(genutil.EntryCamelCaseName(pathElements[idx]))
		if idx != 0 {
			buf.WriteRune('_')
		}
	}

	return buf.String()
}

// goStructName generates the name to be used for a particular YANG schema
// element in the generated Go code. If the compressOCPaths boolean is set to
// true, schemapaths are compressed, otherwise the name is returned simply as
// camel case. The genFakeRoot boolean specifies whether the fake root is to be
// generated such that the struct name can consider the fake root entity
// specifically.
func (s *genState) goStructName(e *yang.Entry, compressOCPaths, genFakeRoot bool) string {
	uniqName := genutil.MakeNameUnique(s.pathToCamelCaseName(e, compressOCPaths, genFakeRoot), s.definedGlobals)

	// Record the name of the struct that was unique such that it can be referenced
	// by path.
	s.uniqueDirectoryNames[e.Path()] = uniqName

	return uniqName
}

// yangTypeToGoType takes a yang.YangType (YANG type definition) and maps it
// to the type that should be used to represent it in the generated Go code.
// A resolveTypeArgs structure is used as the input argument which specifies a
// pointer to the YangType; and optionally context required to resolve the name
// of the type. The compressOCPaths argument specifies whether compression of
// OpenConfig paths is to be enabled.
func (s *genState) yangTypeToGoType(args resolveTypeArgs, compressOCPaths bool) (*MappedType, error) {
	defVal := genutil.TypeDefaultValue(args.yangType)
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
		mtype.ZeroValue = "0"
		if defVal != nil {
			mtype.DefaultValue = enumDefaultValue(mtype.NativeType, *defVal, goEnumPrefix)
		}

		return mtype, nil
	}

	// Perform the actual mapping of the type to the Go type.
	switch args.yangType.Kind {
	case yang.Yint8:
		return &MappedType{NativeType: "int8", ZeroValue: goZeroValues["int8"], DefaultValue: defVal}, nil
	case yang.Yint16:
		return &MappedType{NativeType: "int16", ZeroValue: goZeroValues["int16"], DefaultValue: defVal}, nil
	case yang.Yint32:
		return &MappedType{NativeType: "int32", ZeroValue: goZeroValues["int32"], DefaultValue: defVal}, nil
	case yang.Yint64:
		return &MappedType{NativeType: "int64", ZeroValue: goZeroValues["int64"], DefaultValue: defVal}, nil
	case yang.Yuint8:
		return &MappedType{NativeType: "uint8", ZeroValue: goZeroValues["uint8"], DefaultValue: defVal}, nil
	case yang.Yuint16:
		return &MappedType{NativeType: "uint16", ZeroValue: goZeroValues["uint16"], DefaultValue: defVal}, nil
	case yang.Yuint32:
		return &MappedType{NativeType: "uint32", ZeroValue: goZeroValues["uint32"], DefaultValue: defVal}, nil
	case yang.Yuint64:
		return &MappedType{NativeType: "uint64", ZeroValue: goZeroValues["uint64"], DefaultValue: defVal}, nil
	case yang.Ybool:
		return &MappedType{NativeType: "bool", ZeroValue: goZeroValues["bool"], DefaultValue: defVal}, nil
	case yang.Yempty:
		// Empty is a YANG type that either exists or doesn't, therefore
		// map it to a boolean to indicate its presence or not. The empty
		// type name uses a specific name in the generated code, such that
		// it can be identified for marshalling.
		return &MappedType{NativeType: ygot.EmptyTypeName, ZeroValue: goZeroValues[ygot.EmptyTypeName]}, nil
	case yang.Ystring:
		return &MappedType{NativeType: "string", ZeroValue: goZeroValues["string"], DefaultValue: defVal}, nil
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
		return &MappedType{
			NativeType:        fmt.Sprintf("E_%s", n),
			IsEnumeratedValue: true,
			ZeroValue:         "0",
			DefaultValue:      defVal,
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
		return &MappedType{
			NativeType:        fmt.Sprintf("E_%s", n),
			IsEnumeratedValue: true,
			ZeroValue:         "0",
			DefaultValue:      defVal,
		}, nil
	case yang.Ydecimal64:
		return &MappedType{NativeType: "float64", ZeroValue: goZeroValues["float64"]}, nil
	case yang.Yleafref:
		// This is a leafref, so we check what the type of the leaf that it
		// references is by looking it up in the schematree.
		target, err := s.resolveLeafrefTarget(args.yangType.Path, args.contextEntry)
		if err != nil {
			return nil, err
		}
		mtype, err = s.yangTypeToGoType(resolveTypeArgs{yangType: target.Type, contextEntry: target}, compressOCPaths)
		if err != nil {
			return nil, err
		}
		return mtype, nil
	case yang.Ybinary:
		// Map binary fields to the Binary type defined in the output code,
		// this is used to ensure that we can distinguish a binary field from
		// a leaf-list of uint8s which is not possible if mapping to []byte.
		return &MappedType{NativeType: ygot.BinaryTypeName, ZeroValue: goZeroValues[ygot.BinaryTypeName], DefaultValue: defVal}, nil
	default:
		// Return an empty interface for the types that we do not currently
		// support. Back-end validation is required for these types.
		// TODO(robjs): Missing types currently bits. These
		// should be added.
		return &MappedType{NativeType: "interface{}", ZeroValue: goZeroValues["interface{}"]}, nil
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
func (s *genState) goUnionType(args resolveTypeArgs, compressOCPaths bool) (*MappedType, error) {
	var errs []error
	unionMappedTypes := make(map[int]*MappedType)

	// Extract the subtypes that are defined into a map which is keyed on the
	// mapped type. A map is used such that other functions that rely checking
	// whether a particular type is valid when creating mapping code can easily
	// check, rather than iterating the slice of strings.
	unionTypes := make(map[string]int)
	for _, subtype := range args.yangType.Type {
		errs = append(errs, s.goUnionSubTypes(subtype, args.contextEntry, unionTypes, unionMappedTypes, compressOCPaths)...)
	}

	if errs != nil {
		return nil, fmt.Errorf("errors mapping element: %v", errs)
	}

	resolvedType := &MappedType{
		NativeType: fmt.Sprintf("%s_Union", s.pathToCamelCaseName(args.contextEntry, compressOCPaths, false)),
		// Zero value is set to nil, other than in cases where there is
		// a single type in the union.
		ZeroValue: "nil",
	}
	// If there is only one type inside the union, then promote it to replace the union type.
	if len(unionMappedTypes) == 1 {
		resolvedType = unionMappedTypes[0]
	}

	resolvedType.UnionTypes = unionTypes

	return resolvedType, nil
}

// goUnionSubTypes extracts all the possible subtypes of a YANG union leaf,
// returning any errors that occur. In case of nested unions, the entire union
// is flattened, and identical types are de-duped. currentTypes keeps track of
// this unique set of types, along with the order they're seen, and
// unionMappedTypes records the entire type information for each. The
// compressOCPaths argument specifies whether OpenConfig path compression is
// enabled such that the name of enumerated types can be correctly calculated.
func (s *genState) goUnionSubTypes(subtype *yang.YangType, ctx *yang.Entry, currentTypes map[string]int, unionMappedTypes map[int]*MappedType, compressOCPaths bool) []error {
	var errs []error
	// If subtype.Type is not empty then this means that this type is defined to
	// be a union itself.
	if subtype.Type != nil {
		for _, st := range subtype.Type {
			errs = append(errs, s.goUnionSubTypes(st, ctx, currentTypes, unionMappedTypes, compressOCPaths)...)
		}
		return errs
	}

	var mtype *MappedType
	switch subtype.Kind {
	case yang.Yidentityref:
		// Handle the specific case that the context entry is now not the correct entry
		// to map enumerated types to their module. This occurs in the case that the subtype
		// is an identityref - in this case, the context entry that we are carrying is the
		// leaf that refers to the union, not the specific subtype that is now being examined.
		baseType := s.identityrefBaseTypeFromIdentity(subtype.IdentityBase, false)
		defVal := genutil.TypeDefaultValue(subtype)
		if defVal != nil {
			defVal = enumDefaultValue(baseType, *defVal, "")
		}
		mtype = &MappedType{
			NativeType:        fmt.Sprintf("E_%s", baseType),
			IsEnumeratedValue: true,
			ZeroValue:         "0",
			DefaultValue:      defVal,
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
	if _, ok := currentTypes[mtype.NativeType]; !ok {
		index := len(currentTypes)
		currentTypes[mtype.NativeType] = index
		unionMappedTypes[index] = mtype
	}
	return errs
}

// buildListKey takes a yang.Entry, e, corresponding to a list and extracts the definition
// of the list key, returning a YangListAttr struct describing the key element(s). If
// errors are encountered during the extraction, they are returned as a slice of errors.
// The YangListAttr that is returned consists of a map, keyed by the key leaf's YANG
// identifier, with a value of a MappedType struct which indicates how that key leaf
// is to be represented in Go. The key elements themselves are returned in the keyElems
// slice.
func (s *genState) buildListKey(e *yang.Entry, compressOCPaths bool) (*YangListAttr, []error) {
	if !e.IsList() {
		return nil, []error{fmt.Errorf("%s is not a list", e.Name)}
	}

	if e.Key == "" {
		// A null key is not valid if we have a config true list, so return an error
		if util.IsConfig(e) {
			return nil, []error{fmt.Errorf("No key specified for a config true list: %s", e.Name)}
		}
		// This is a keyless list so return an empty YangListAttr but no error, downstream
		// mapping code should consider this to mean that this should be mapped into a
		// keyless structure (i.e., a slice).
		return nil, nil
	}

	listattr := &YangListAttr{
		Keys: make(map[string]*MappedType),
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
					dir := util.StripModulePrefix(refparts[len(refparts)-2])
					d, ok := e.Dir[dir]
					if !ok {
						return nil, []error{
							fmt.Errorf("Key %s had a leafref key (%s) in dir %s that did not exist (%v)",
								k, keyleaf.Path(), dir, refparts),
						}
					}
					targetLeaf := util.StripModulePrefix(refparts[len(refparts)-1])
					if _, ok := d.Dir[targetLeaf]; !ok {
						return nil, []error{
							fmt.Errorf("Key %s had leafref key (%s) that did not exist at (%v)", k, keyleaf.Path(), refparts),
						}
					}
					keyleaf = d.Dir[targetLeaf]
				}
			}
		}

		listattr.KeyElems = append(listattr.KeyElems, keyleaf)
		keyType, err := s.yangTypeToGoType(resolveTypeArgs{yangType: keyleaf.Type, contextEntry: keyleaf}, compressOCPaths)
		if err != nil {
			errs = append(errs, err)
		}
		listattr.Keys[keyleaf.Name] = keyType
	}

	return listattr, errs
}
