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

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/genutil"
	"github.com/openconfig/ygot/ygot"
)

// TODO(robjs): Split the contents of this file out into a separate package
// once refactoring of Go generation has been completed.

const (
	// goEnumPrefix is the prefix that is used for type names in the output
	// Go code, such that an enumeration's name is of the form
	//   <goEnumPrefix><EnumName>
	goEnumPrefix string = "E_"
	// goEnumerationUseUnderscores determines whether underscores are used
	// within the output generated code.
	goEnumerationUseUnderscores = false
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

// Ensure at compile time that the GoLangMapper implements the LangMapper interface.
var _ LangMapper = &GoLangMapper{}

type GoLangMapper struct {
	// enumSet stores the generated set of enumerations that are be produced
	// for the generated code. It is set internally by the ygen IR implementation
	// and is not publicly accessible outside of the ygen package. It is used to
	// do type lookups following the initial generation of enumerated values.
	enumSet *EnumSet

	// schematree stores the parsed YANG schema tree that has been produced
	// from the input YANG modules during the ygen IR production process. It
	// is private to the ygen package, and is used only to look up leafref
	// types during the code production process.
	schematree *SchemaTree

	// definedGlobals specifies the global Go names that have been used during
	// name generation to avoid name clashes. IT is a map keyed by the name of
	// the global that has been defined.
	definedGlobals map[string]bool

	// uniqueDirectoryNames is a map, keyed by the path of a YANG entity representing
	// a directory in the generated code whose value is the unique name that it
	// was mapped to. It is used to determine based on a particular YANG path
	// the name that was assigned. It is not used outside of the IR generation
	// process.
	uniqueDirectoryNames map[string]string
}

// NewGoLangMapper returns a new instance of the LangMapper for the Go language.
func NewGoLangMapper() *GoLangMapper {
	return &GoLangMapper{
		definedGlobals: map[string]bool{
			// Mark the name that is used for the binary type as a reserved name
			// within the output structs.
			ygot.BinaryTypeName: true,
			ygot.EmptyTypeName:  true,
		},
		uniqueDirectoryNames: map[string]string{},
	}
}

// SetEnumSet stores the supplied enumSet within the GoLangMapper instance.
func (g *GoLangMapper) SetEnumSet(e *EnumSet) { g.enumSet = e }

// SetSchemaTree stores the supplied schemaTree within the GoLangMapper instance.
func (g *GoLangMapper) SetSchemaTree(s *SchemaTree) { g.schemaTree = s }

// DirectoryName returns the name of a directory that should be generated for a particular
// input YANG entry.
func (g *GoLangMapper) DirectoryName(e *yang.Entry, cb genutil.CompressBehaviour) (string, error) {
	uniqName := genutil.MakeNameUnique(pathToCamelCaseName(e, cb.CompressEnabled()), g.definedGlobals)

	// Record the name of the struct that was unique such that it can be referenced
	// by path.
	g.uniqueDirectoryNames[e.Path()] = uniqName

	return uniqName, nil
}

// FieldName returns the name that is used for a field within a directory that corresponds
// to the yang.Entry supplied.
func (g *GoLangMapper) FieldName(e *yang.Entry) (string, error) {
	// The Go name mapping for a leaf cannot create a erroneous name, and hence
	// we never return an error here.
	return genutil.EntryCamelCaseName(e), nil
}

// KeyLeafType returns the Go type that should be used for a leaf that corresponds to the entry e, with
// the specified compression behaviour. The type returned is represented as a MappedType IR pointer.
func (g *GoLangMapper) KeyLeafType(e *yang.Entry, cb genutil.CompressBehaviour) (*MappedType, error) {
	// In Go, there is no difference between the type that is used for a leaf type and that which
	// is used for a key leaf.
	return g.LeafType(e, cb)
}

// LeafType returns the Go type that should be used for a leaf that corresponds to the entry e, with
// the specified compression behaviour. The type returned is represented as a MappedType IR pointer.
func (g *GoLangMapper) LeafType(e *yang.Entry, cb genutil.CompressBehaviour) (*MappedType, error) {
	return g.leafTypeInternal(genutil.ResolveTypeArgs{YangType: e.Type, ContextEntry: e}, cb.CompressEnabled())
}

func (g *GoLangMapper) leafTypeInternal(args genutil.ResolveTypeArgs, compressPaths bool) (*MappedType, error) {
	defVal := genutil.TypeDefaultValue(args.YangType)
	// Handle the case of a typedef which is actually an enumeration.
	mtype, err := g.enumSet.LookupTypedef(args.YangType, args.ContextEntry)
	if err != nil {
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
		return g.goUnionType(args, compressPaths)
	case yang.Yenum:
		// Enumeration types need to be resolved to a particular data path such
		// that a created enumered Go type can be used to set their value. Hand
		// the leaf to the enumName function to determine the name.
		if args.ContextEntry == nil {
			return nil, fmt.Errorf("cannot map enum without context")
		}
		mtype, err := g.enumSet.LookupEnum(args.YangType, args.ContextEntry)
		if err != nil {
			return nil, err
		}
		if defVal != nil {
			mtype.DefaultValue = enumDefaultValue(mtype.NativeType, *defVal, goEnumPrefix)
		}
		mtype.ZeroValue = "0"
		return mtype, nil
	case yang.Yidentityref:
		// Identityref leaves are mapped according to the base identity that they
		// refer to - this is stored in the IdentityBase field of the context leaf
		// which is determined by the identityrefBaseTypeFromLeaf.
		if args.contextEntry == nil {
			return nil, fmt.Errorf("cannot map identityref without context")
		}
		mtype, err := g.enumSet.LookupIdentity(args.ContextEntry.Type.IdentityBase)
		if err != nil {
			return nil, err
		}
		if defVal != nil {
			mtype.DefaultValue = enumDefaultValue(mtype.NativeType, *defVal, goEnumPrefix)
		}
		mtype.ZeroValue = "0"
		return mtype, nil
	case yang.Ydecimal64:
		return &MappedType{NativeType: "float64", ZeroValue: goZeroValues["float64"]}, nil
	case yang.Yleafref:
		// This is a leafref, so we check what the type of the leaf that it
		// references is by looking it up in the schematree.
		target, err := g.schematree.resolveLeafrefTarget(args.YangType.Path, args.ContextEntry)
		if err != nil {
			return nil, err
		}
		mtype, err = g.leafTypeInternal(genutil.ResolveTypeArgs{YangType: target.Type, ContextEntry: target}, compressPaths)
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
// The skipEnumDedup argument specifies whether the code generation should aim
// to use a common type for enumerations that are logically defined once in the schema
// but used in multiple places.
//
// goUnionType returns an error if mapping is not possible.
func (g *GoLangMapper) goUnionType(args genutil.ResolveTypeArgs, compressOCPaths bool) (*MappedType, error) {
	var errs []error
	unionMappedTypes := make(map[int]*MappedType)

	// Extract the subtypes that are defined into a map which is keyed on the
	// mapped type. A map is used such that other functions that rely checking
	// whether a particular type is valid when creating mapping code can easily
	// check, rather than iterating the slice of strings.
	unionTypes := make(map[string]int)
	for _, subtype := range args.YangType.Type {
		errs = append(errs, g.goUnionSubTypes(subtype, args.ContextEntry, unionTypes, unionMappedTypes, compressOCPaths)...)
	}

	if errs != nil {
		return nil, fmt.Errorf("errors mapping element: %v", errs)
	}

	resolvedType := &MappedType{
		NativeType: fmt.Sprintf("%s_Union", pathToCamelCaseName(args.ContextEntry, compressOCPaths)),
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
func (g *GoLangMapper) goUnionSubTypes(subtype *yang.YangType, ctx *yang.Entry, currentTypes map[string]int, unionMappedTypes map[int]*MappedType, compressOCPaths bool) []error {
	var errs []error
	// If subtype.Type is not empty then this means that this type is defined to
	// be a union itself.
	if subtype.Type != nil {
		for _, st := range subtype.Type {
			errs = append(errs, g.goUnionSubTypes(st, ctx, currentTypes, unionMappedTypes, compressOCPaths, skipEnumDedup, shortenEnumLeafNames, useDefiningModuleForTypedefEnumNames)...)
		}
		return errs
	}

	contextType := subtype

	var mtype *MappedType
	switch subtype.Kind {
	case yang.Yidentityref:
		// Handle the specific case that the context entry is now not the correct entry
		// to map enumerated types to their module. This occurs in the case that the subtype
		// is an identityref - in this case, the context entry that we are carrying is the
		// leaf that refers to the union, not the specific subtype that is now being examined.
		baseType, err := g.enumSet.identityrefBaseTypeFromIdentity(subtype.IdentityBase)
		if err != nil {
			return append(errs, err)
		}
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

		mtype, err = g.leafTypeInternal(genutil.ResolveTypeArgs{YangType: contextType, ContextEntry: ctx}, compressOCPaths, skipEnumDedup, shortenEnumLeafNames, useDefiningModuleForTypedefEnumNames)
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

// NewTemporaryGoLangMapper creates a new Golang mapper, and is implemented
// to allow backwards compatibility throughout the refactoring of the ygen
// library with existing test implementations.
func NewTemporaryGoLangMapper(e *EnumSet, s *SchemaTree) *GoLangMapper {
	g := NewGoLangMapper()
	g.SetEnumSet(e)
	g.SetSchemaTree(s)
	return g
}
