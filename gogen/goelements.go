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

package gogen

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/genutil"
	"github.com/openconfig/ygot/internal/igenutil"
	"github.com/openconfig/ygot/util"
	"github.com/openconfig/ygot/ygen"
	"github.com/openconfig/ygot/ygot"
	"github.com/openconfig/ygot/ytypes"
)

const (
	// goEnumPrefix is the prefix that is used for type names in the output
	// Go code, such that an enumeration's name is of the form
	//   <goEnumPrefix><EnumName>
	goEnumPrefix string = "E_"
)

// unionConversionSpec stores snippets that convert primitive Go types to
// union typedef types.
type unionConversionSpec struct {
	// PrimitiveType is the primitive Go type from which to convert to the
	// union type.
	PrimitiveType string
	// ConversionSnippet is the code snippet that converts the primitive
	// type to the union type.
	ConversionSnippet string
}

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

	// simpleUnionConversionsFromKind stores the simple union conversion
	// types in Go given a yang.TypeKind.
	simpleUnionConversionsFromKind = map[yang.TypeKind]string{
		yang.Yint8:      "UnionInt8",
		yang.Yint16:     "UnionInt16",
		yang.Yint32:     "UnionInt32",
		yang.Yint64:     "UnionInt64",
		yang.Yuint8:     "UnionUint8",
		yang.Yuint16:    "UnionUint16",
		yang.Yuint32:    "UnionUint32",
		yang.Yuint64:    "UnionUint64",
		yang.Ydecimal64: "UnionFloat64",
		yang.Ystring:    "UnionString",
		yang.Ybool:      "UnionBool",
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

	// unionConversionSnippets stores the valid primitive types that the Go
	// code generation produces that can be used as a union subtype, and
	// information on how to convert it to a union-satisfying type.
	unionConversionSnippets = map[string]*unionConversionSpec{
		"int8":              {PrimitiveType: "int8", ConversionSnippet: ygot.SimpleUnionBuiltinGoTypes["int8"] + "(v)"},
		"int16":             {PrimitiveType: "int16", ConversionSnippet: ygot.SimpleUnionBuiltinGoTypes["int16"] + "(v)"},
		"int32":             {PrimitiveType: "int32", ConversionSnippet: ygot.SimpleUnionBuiltinGoTypes["int32"] + "(v)"},
		"int64":             {PrimitiveType: "int64", ConversionSnippet: ygot.SimpleUnionBuiltinGoTypes["int64"] + "(v)"},
		"uint8":             {PrimitiveType: "uint8", ConversionSnippet: ygot.SimpleUnionBuiltinGoTypes["uint8"] + "(v)"},
		"uint16":            {PrimitiveType: "uint16", ConversionSnippet: ygot.SimpleUnionBuiltinGoTypes["uint16"] + "(v)"},
		"uint32":            {PrimitiveType: "uint32", ConversionSnippet: ygot.SimpleUnionBuiltinGoTypes["uint32"] + "(v)"},
		"uint64":            {PrimitiveType: "uint64", ConversionSnippet: ygot.SimpleUnionBuiltinGoTypes["uint64"] + "(v)"},
		"float64":           {PrimitiveType: "float64", ConversionSnippet: ygot.SimpleUnionBuiltinGoTypes["float64"] + "(v)"},
		"string":            {PrimitiveType: "string", ConversionSnippet: ygot.SimpleUnionBuiltinGoTypes["string"] + "(v)"},
		"bool":              {PrimitiveType: "bool", ConversionSnippet: ygot.SimpleUnionBuiltinGoTypes["bool"] + "(v)"},
		"interface{}":       {PrimitiveType: "interface{}", ConversionSnippet: "&UnionUnsupported{v}"},
		ygot.BinaryTypeName: {PrimitiveType: "[]byte", ConversionSnippet: ygot.BinaryTypeName + "(v)"},
		ygot.EmptyTypeName:  {PrimitiveType: "bool", ConversionSnippet: ygot.EmptyTypeName + "(v)"},
	}
)

// Ensure at compile time that the GoLangMapper implements the LangMapper interface.
var _ ygen.LangMapper = &GoLangMapper{}

// GoLangMapper contains the functionality and state for generating Go names for
// the generated code.
type GoLangMapper struct {
	// LangMapperBase being embedded is a requirement for GoLangMapper to
	// implement the LangMapper interface, and also gives it access to
	// built-in methods.
	ygen.LangMapperBase

	// definedGlobals specifies the global Go names used during code
	// generation to avoid conflicts.
	definedGlobals map[string]bool

	// uniqueDirectoryNames is a map keyed by the path of a YANG entity representing a
	// directory in the generated code whose value is the unique name that it
	// was mapped to. This allows routines to determine, based on a particular YANG
	// entry, how to refer to it when generating code.
	uniqueDirectoryNames map[string]string

	// simpleUnions specifies whether simple typedefs are used to represent
	// union subtypes in the generated code instead of using wrapper types.
	// NOTE: This flag will be removed as part of ygot's v1 release.
	simpleUnions bool

	// UnimplementedLangMapperExt ensures GoLangMapper implements the
	// LangMapperExt interface for forwards compatibility.
	ygen.UnimplementedLangMapperExt
}

// NewGoLangMapper creates a new GoLangMapper instance, initialised with the
// default state required for code generation.
func NewGoLangMapper(simpleUnions bool) *GoLangMapper {
	return &GoLangMapper{
		definedGlobals: map[string]bool{
			// Mark the name that is used for the binary type as a reserved name
			// within the output structs.
			ygot.BinaryTypeName: true,
			ygot.EmptyTypeName:  true,
		},
		uniqueDirectoryNames: map[string]string{},
		simpleUnions:         simpleUnions,
	}
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

// pathToCamelCaseName takes an input yang.Entry and outputs its name as a Go
// compatible name in the form PathElement1_PathElement2, performing schema
// compression if required. The name is not checked for uniqueness.
func pathToCamelCaseName(e *yang.Entry, compressOCPaths bool) string {
	var pathElements []*yang.Entry

	if igenutil.IsFakeRoot(e) {
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

// DirectoryName generates the final name to be used for a particular YANG
// schema element in the generated Go code. If path compressing is active,
// schemapaths are compressed, otherwise the name is returned simply as camel
// case.
// Although name conversion is lossy, name uniquification occurs at this stage
// since all generated struct names reside in the package namespace.
func (s *GoLangMapper) DirectoryName(e *yang.Entry, compressBehaviour genutil.CompressBehaviour) (string, error) {
	// TODO(wenbli): Do not uniquify at this step -- rather do this in a
	// later pass to avoid non-idempotent behaviour in GoLangMapper.
	uniqName := genutil.MakeNameUnique(pathToCamelCaseName(e, compressBehaviour.CompressEnabled()), s.definedGlobals)

	// Record the name of the struct that was unique such that it can be referenced
	// by path.
	s.uniqueDirectoryNames[e.Path()] = uniqName

	return uniqName, nil
}

// FieldName maps the input entry's name to what the Go name of the field would be.
// Since this conversion is lossy, a later step should resolve any naming
// conflicts between different fields.
func (s *GoLangMapper) FieldName(e *yang.Entry) (string, error) {
	return genutil.EntryCamelCaseName(e), nil
}

// LeafType maps the input leaf entry to a MappedType object containing the
// type information about the field.
func (s *GoLangMapper) LeafType(e *yang.Entry, opts ygen.IROptions) (*ygen.MappedType, error) {
	mtype, err := s.yangTypeToGoType(resolveTypeArgs{yangType: e.Type, contextEntry: e}, opts.TransformationOptions.CompressBehaviour.CompressEnabled(), opts.TransformationOptions.SkipEnumDeduplication, opts.TransformationOptions.ShortenEnumLeafNames, opts.TransformationOptions.UseDefiningModuleForTypedefEnumNames, opts.TransformationOptions.EnumOrgPrefixesToTrim)
	if err != nil {
		return nil, err
	}

	defaultValue, err := generateGoDefaultValue(e, mtype, s, opts.TransformationOptions.CompressBehaviour.CompressEnabled(), opts.TransformationOptions.SkipEnumDeduplication, opts.TransformationOptions.ShortenEnumLeafNames, opts.TransformationOptions.UseDefiningModuleForTypedefEnumNames, opts.TransformationOptions.EnumOrgPrefixesToTrim, s.simpleUnions)
	if err != nil {
		return nil, err
	}

	mtype.DefaultValue = defaultValue
	return mtype, nil
}

// LeafType maps the input list key entry to a MappedType object containing the
// type information about the key field.
func (s *GoLangMapper) KeyLeafType(e *yang.Entry, opts ygen.IROptions) (*ygen.MappedType, error) {
	return s.yangTypeToGoType(resolveTypeArgs{yangType: e.Type, contextEntry: e}, opts.TransformationOptions.CompressBehaviour.CompressEnabled(), opts.TransformationOptions.SkipEnumDeduplication, opts.TransformationOptions.ShortenEnumLeafNames, opts.TransformationOptions.UseDefiningModuleForTypedefEnumNames, opts.TransformationOptions.EnumOrgPrefixesToTrim)
}

// PackageName is not used by Go generation.
func (s *GoLangMapper) PackageName(*yang.Entry, genutil.CompressBehaviour, bool) (string, error) {
	return "", nil
}

// yangTypeToGoType takes a yang.YangType (YANG type definition) and maps it
// to the type that should be used to represent it in the generated Go code.
// A resolveTypeArgs structure is used as the input argument which specifies a
// pointer to the YangType; and optionally context required to resolve the name
// of the type. The compressOCPaths argument specifies whether compression of
// OpenConfig paths is to be enabled. The skipEnumDedup argument specifies whether
// the current schema is set to deduplicate enumerations that are logically defined
// once in the YANG schema, but instantiated in multiple places.
// The skipEnumDedup argument specifies whether leaves of type enumeration that are
// used more than once in the schema should share a common type. By default, a single
// type for each leaf is created.
func (s *GoLangMapper) yangTypeToGoType(args resolveTypeArgs, compressOCPaths, skipEnumDedup, shortenEnumLeafNames, useDefiningModuleForTypedefEnumNames bool, enumOrgPrefixesToTrim []string) (*ygen.MappedType, error) {
	defVal := genutil.TypeDefaultValue(args.yangType)
	// Handle the case of a typedef which is actually an enumeration.
	typedefName, _, isTypedef, err := s.EnumeratedTypedefTypeName(args.yangType, args.contextEntry, goEnumPrefix, false, useDefiningModuleForTypedefEnumNames)
	if err != nil {
		// err is non nil when this was a typedef which included
		// an invalid enumerated type.
		return nil, err
	}

	if isTypedef {
		return &ygen.MappedType{
			NativeType:        typedefName,
			IsEnumeratedValue: true,
			// mtype is set to non-nil when this was a valid enumeration
			// within a typedef. We explicitly set the zero and default values
			// here.
			ZeroValue:    "0",
			DefaultValue: defVal,
		}, nil
	}

	// Perform the actual mapping of the type to the Go type.
	switch args.yangType.Kind {
	case yang.Yint8:
		return &ygen.MappedType{NativeType: "int8", ZeroValue: goZeroValues["int8"], DefaultValue: defVal}, nil
	case yang.Yint16:
		return &ygen.MappedType{NativeType: "int16", ZeroValue: goZeroValues["int16"], DefaultValue: defVal}, nil
	case yang.Yint32:
		return &ygen.MappedType{NativeType: "int32", ZeroValue: goZeroValues["int32"], DefaultValue: defVal}, nil
	case yang.Yint64:
		return &ygen.MappedType{NativeType: "int64", ZeroValue: goZeroValues["int64"], DefaultValue: defVal}, nil
	case yang.Yuint8:
		return &ygen.MappedType{NativeType: "uint8", ZeroValue: goZeroValues["uint8"], DefaultValue: defVal}, nil
	case yang.Yuint16:
		return &ygen.MappedType{NativeType: "uint16", ZeroValue: goZeroValues["uint16"], DefaultValue: defVal}, nil
	case yang.Yuint32:
		return &ygen.MappedType{NativeType: "uint32", ZeroValue: goZeroValues["uint32"], DefaultValue: defVal}, nil
	case yang.Yuint64:
		return &ygen.MappedType{NativeType: "uint64", ZeroValue: goZeroValues["uint64"], DefaultValue: defVal}, nil
	case yang.Ybool:
		return &ygen.MappedType{NativeType: "bool", ZeroValue: goZeroValues["bool"], DefaultValue: defVal}, nil
	case yang.Yempty:
		// Empty is a YANG type that either exists or doesn't, therefore
		// map it to a boolean to indicate its presence or not. The empty
		// type name uses a specific name in the generated code, such that
		// it can be identified for marshalling.
		return &ygen.MappedType{NativeType: ygot.EmptyTypeName, ZeroValue: goZeroValues[ygot.EmptyTypeName]}, nil
	case yang.Ystring:
		return &ygen.MappedType{NativeType: "string", ZeroValue: goZeroValues["string"], DefaultValue: defVal}, nil
	case yang.Yunion:
		// A YANG Union is a leaf that can take multiple values - its subtypes need
		// to be extracted.
		return s.goUnionType(args, compressOCPaths, skipEnumDedup, shortenEnumLeafNames, useDefiningModuleForTypedefEnumNames, enumOrgPrefixesToTrim)
	case yang.Yenum:
		// Enumeration types need to be resolved to a particular data path such
		// that a created enumerated Go type can be used to set their value. Hand
		// the leaf to the enumName function to determine the name.
		if args.contextEntry == nil {
			return nil, fmt.Errorf("cannot map enum without context")
		}
		n, _, err := s.EnumName(args.contextEntry, compressOCPaths, false, skipEnumDedup, shortenEnumLeafNames, false, enumOrgPrefixesToTrim)
		if err != nil {
			return nil, err
		}
		return &ygen.MappedType{
			NativeType:        fmt.Sprintf("%s%s", goEnumPrefix, n),
			IsEnumeratedValue: true,
			ZeroValue:         "0",
			DefaultValue:      defVal,
		}, nil
	case yang.Yidentityref:
		// Identityref leaves are mapped according to the base identity that they
		// refer to - this is stored in the IdentityBase field of the context leaf
		// which is determined by the IdentityrefBaseTypeFromLeaf.
		if args.contextEntry == nil {
			return nil, fmt.Errorf("cannot map identityref without context")
		}
		n, _, err := s.IdentityrefBaseTypeFromLeaf(args.contextEntry)
		if err != nil {
			return nil, err
		}
		return &ygen.MappedType{
			NativeType:        fmt.Sprintf("%s%s", goEnumPrefix, n),
			IsEnumeratedValue: true,
			ZeroValue:         "0",
			DefaultValue:      defVal,
		}, nil
	case yang.Ydecimal64:
		return &ygen.MappedType{NativeType: "float64", ZeroValue: goZeroValues["float64"], DefaultValue: defVal}, nil
	case yang.Yleafref:
		// This is a leafref, so we check what the type of the leaf that it
		// references is by looking it up.
		target, err := s.ResolveLeafrefTarget(args.yangType.Path, args.contextEntry)
		if err != nil {
			return nil, err
		}
		mtype, err := s.yangTypeToGoType(resolveTypeArgs{yangType: target.Type, contextEntry: target}, compressOCPaths, skipEnumDedup, shortenEnumLeafNames, useDefiningModuleForTypedefEnumNames, enumOrgPrefixesToTrim)
		if err != nil {
			return nil, err
		}
		return mtype, nil
	case yang.Ybinary:
		// Map binary fields to the Binary type defined in the output code,
		// this is used to ensure that we can distinguish a binary field from
		// a leaf-list of uint8s which is not possible if mapping to []byte.
		return &ygen.MappedType{NativeType: ygot.BinaryTypeName, ZeroValue: goZeroValues[ygot.BinaryTypeName], DefaultValue: defVal}, nil
	default:
		// Return an empty interface for the types that we do not currently
		// support. Back-end validation is required for these types.
		// TODO(robjs): Missing types currently bits. These
		// should be added.
		return &ygen.MappedType{NativeType: "interface{}", ZeroValue: goZeroValues["interface{}"]}, nil
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
//
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
func (s *GoLangMapper) goUnionType(args resolveTypeArgs, compressOCPaths, skipEnumDedup, shortenEnumLeafNames, useDefiningModuleForTypedefEnumNames bool, enumOrgPrefixesToTrim []string) (*ygen.MappedType, error) {
	var errs []error
	unionMappedTypes := make(map[int]*ygen.MappedType)

	// Extract the subtypes that are defined into a map which is keyed on the
	// mapped type. A map is used such that other functions that rely checking
	// whether a particular type is valid when creating mapping code can easily
	// check, rather than iterating the slice of strings.
	unionTypes := make(map[string]ygen.MappedUnionSubtype)
	for _, subtype := range args.yangType.Type {
		errs = append(errs, s.goUnionSubTypes(subtype, args.contextEntry, unionTypes, unionMappedTypes, compressOCPaths, skipEnumDedup, shortenEnumLeafNames, useDefiningModuleForTypedefEnumNames, enumOrgPrefixesToTrim)...)
	}

	if errs != nil {
		return nil, fmt.Errorf("errors mapping element: %v", errs)
	}

	resolvedType := &ygen.MappedType{
		NativeType: fmt.Sprintf("%s_Union", pathToCamelCaseName(args.contextEntry, compressOCPaths)),
		// Zero value is set to nil, other than in cases where there is
		// a single type in the union.
		ZeroValue:    "nil",
		DefaultValue: genutil.TypeDefaultValue(args.yangType),
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
// The skipEnumDedup argument specifies whether the current code generation is
// de-duplicating enumerations where they are used in more than one place in
// the schema.
func (s *GoLangMapper) goUnionSubTypes(subtype *yang.YangType, ctx *yang.Entry, currentTypes map[string]ygen.MappedUnionSubtype, unionMappedTypes map[int]*ygen.MappedType, compressOCPaths, skipEnumDedup, shortenEnumLeafNames, useDefiningModuleForTypedefEnumNames bool, enumOrgPrefixesToTrim []string) []error {
	var errs []error
	// If subtype.Type is not empty then this means that this type is defined to
	// be a union itself.
	if subtype.Type != nil {
		for _, st := range subtype.Type {
			errs = append(errs, s.goUnionSubTypes(st, ctx, currentTypes, unionMappedTypes, compressOCPaths, skipEnumDedup, shortenEnumLeafNames, useDefiningModuleForTypedefEnumNames, enumOrgPrefixesToTrim)...)
		}
		return errs
	}

	contextType := subtype

	var mtype *ygen.MappedType
	switch subtype.Kind {
	case yang.Yidentityref:
		// Handle the specific case that the context entry is now not the correct entry
		// to map enumerated types to their module. This occurs in the case that the subtype
		// is an identityref - in this case, the context entry that we are carrying is the
		// leaf that refers to the union, not the specific subtype that is now being examined.
		baseType, _, err := s.IdentityrefBaseTypeFromIdentity(subtype.IdentityBase)
		if err != nil {
			return append(errs, err)
		}
		defVal := genutil.TypeDefaultValue(subtype)
		mtype = &ygen.MappedType{
			NativeType:        fmt.Sprintf("%s%s", goEnumPrefix, baseType),
			IsEnumeratedValue: true,
			ZeroValue:         "0",
			DefaultValue:      defVal,
		}
	default:
		var err error

		mtype, err = s.yangTypeToGoType(resolveTypeArgs{yangType: contextType, contextEntry: ctx}, compressOCPaths, skipEnumDedup, shortenEnumLeafNames, useDefiningModuleForTypedefEnumNames, enumOrgPrefixesToTrim)
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
		currentTypes[mtype.NativeType] = ygen.MappedUnionSubtype{
			Index: index,
		}
		unionMappedTypes[index] = mtype
	}
	return errs
}

// generateGoDefaultValue returns a pointer to a Go literal that represents the
// default value for the entry. If there is no default value for the field, nil
// is returned.
func generateGoDefaultValue(field *yang.Entry, mtype *ygen.MappedType, gogen *GoLangMapper, compressPaths, skipEnumDedup, shortenEnumLeafNames, useDefiningModuleForTypedefEnumNames bool, enumOrgPrefixesToTrim []string, simpleUnions bool) (*string, error) {
	// Set the default type to the mapped Go type.
	defaultValues := field.DefaultValues()
	if len(defaultValues) == 0 && mtype.DefaultValue != nil {
		defaultValues = []string{*mtype.DefaultValue}
	}
	for i, defVal := range defaultValues {
		var err error
		if defaultValues[i], _, err = gogen.yangDefaultValueToGo(defVal, resolveTypeArgs{yangType: field.Type, contextEntry: field}, len(mtype.UnionTypes) == 1, compressPaths, skipEnumDedup, shortenEnumLeafNames, useDefiningModuleForTypedefEnumNames, enumOrgPrefixesToTrim); err != nil {
			return nil, err
		}
	}
	// TODO(wenbli): In ygot v1, we should no longer
	// support the wrapper union generated code, so this if
	// block would be obsolete.
	if !simpleUnions {
		defaultValues = goLeafDefaults(field, mtype)
		if len(defaultValues) != 0 && len(mtype.UnionTypes) > 1 {
			// If the default value is applied to a union type, we will generate
			// non-compilable code when generating wrapper unions, so error out and inform
			// the user instead of having the user find out that the code doesn't compile.
			return nil, fmt.Errorf("path %q: default value not supported for wrapper union values, please generate using simplified union leaves", field.Path())
		}
	}

	var defaultValue *string
	if len(defaultValues) > 0 {
		switch {
		case field.ListAttr != nil: // field is a leaf-list
			dv := fmt.Sprintf("[]%s{%s}", mtype.NativeType, strings.Join(defaultValues, ", "))
			defaultValue = &dv
		case len(defaultValues) == 1:
			dv := defaultValues[0]
			defaultValue = &dv
		default:
			return nil, fmt.Errorf("path: %q, unexpected multiple default values %v for a non-leaf-list field", field.Path(), defaultValues)
		}
	}
	return defaultValue, nil
}

// yangDefaultValueToGo takes a default value, and its associated type, schema
// entry, whether it is a union with a single type, and other generation flags,
// and maps it to a Go snippet reference that would represent the value in the
// generated Go code.
// If it is unable to convert the default value according to the given type and
// context schema entry, an error is returned.
// NOTE: This function currently ONLY supports generating default union value
// snippets for simple unions.
//
// The yang.TypeKind return value specifies a non-Yunion, non-Yleafref TypeKind
// that the default value is converted to.
//
// A resolveTypeArgs structure is used as the input argument which specifies a
// pointer to the YangType; and optionally context required to resolve the name
// of the type. The compressOCPaths argument specifies whether compression of
// OpenConfig paths is to be enabled. The skipEnumDedup argument specifies whether
// the current schema is set to deduplicate enumerations that are logically defined
// once in the YANG schema, but instantiated in multiple places.
// The skipEnumDedup argument specifies whether leaves of type enumeration that are
// used more than once in the schema should share a common type. By default, a single
// type for each leaf is created.
func (s *GoLangMapper) yangDefaultValueToGo(value string, args resolveTypeArgs, isSingletonUnion, compressOCPaths, skipEnumDedup, shortenEnumLeafNames, useDefiningModuleForTypedefEnumNames bool, enumOrgPrefixesToTrim []string) (string, yang.TypeKind, error) {
	// Handle the case of a typedef which is actually an enumeration.
	typedefName, _, isTypedef, err := s.EnumeratedTypedefTypeName(args.yangType, args.contextEntry, goEnumPrefix, false, useDefiningModuleForTypedefEnumNames)
	if err != nil {
		// err is non nil when this was a typedef which included
		// an invalid enumerated type.
		return "", yang.Ynone, err
	}
	if isTypedef {
		if strings.Contains(value, ":") {
			value = strings.Split(value, ":")[1]
		}
		switch args.yangType.Kind {
		case yang.Yenum:
			if !args.yangType.Enum.IsDefined(value) {
				return "", yang.Ynone, fmt.Errorf("default value conversion: typedef enum value %q not found in enum with type name %q", value, args.yangType.Name)
			}
		case yang.Yidentityref:
			if !args.yangType.IdentityBase.IsDefined(value) {
				return "", yang.Ynone, fmt.Errorf("default value conversion: typedef identity value %q not found in enum with type name %q", value, args.yangType.Name)
			}
		}
		return enumDefaultValue(typedefName, value, goEnumPrefix), args.yangType.Kind, nil
	}

	signed := false
	// Perform mapping of the default value to the Go snippet.
	switch ykind := args.yangType.Kind; ykind {
	case yang.Yint64, yang.Yint32, yang.Yint16, yang.Yint8:
		signed = true
		fallthrough
	case yang.Yuint64, yang.Yuint32, yang.Yuint16, yang.Yuint8:
		bits, err := util.YangIntTypeBits(ykind)
		if err != nil {
			return "", yang.Ynone, err
		}

		// Check if the value is an empty string.
		signStripedValue := strings.TrimLeft(strings.TrimLeft(value, "-"), "+")
		if len(signStripedValue) == 0 {
			return "", yang.Ynone, fmt.Errorf("default value conversion: empty value")
		}

		// When the default value of int/uint is not base-10, check if the base should be supported according to
		// https://datatracker.ietf.org/doc/html/rfc7950#section-9.2.1
		//
		// When the first character in the value is `0`, the value could be hexadecimal, octal or binary. While hexadecimal
		// and octal values are supported (with an optional sign prefix), binary is not allowed by the RFC. The format support
		// of `strconv.ParseInt/ParseUint` functions should be further constrained.
		if signStripedValue[0] == '0' {
			if len(signStripedValue) > 1 && (signStripedValue[1] == 'b' || signStripedValue[1] == 'B') {
				return "", yang.Ynone, fmt.Errorf("default value conversion: base 2 value `%s` is not allowed", value)
			}
		}

		// Underscores are not allowed in the value, although `strconv` allows them in certain formats.
		if strings.Contains(value, "_") {
			return "", yang.Ynone, fmt.Errorf("default value conversion: `_` is not allowed in the value %s", value)
		}

		if signed {
			val, err := strconv.ParseInt(value, 0, bits)
			if err != nil {
				return "", yang.Ynone, fmt.Errorf("default value conversion: unable to convert default value %q to %v: %v", value, ykind, err)
			}
			if err := ytypes.ValidateIntRestrictions(args.yangType, val); err != nil {
				return "", yang.Ynone, fmt.Errorf("default value conversion: %q doesn't match int restrictions: %v", value, err)
			}
		} else {
			val, err := strconv.ParseUint(value, 0, bits)
			if err != nil {
				return "", yang.Ynone, fmt.Errorf("default value conversion: unable to convert default value %q to %v: %v", value, ykind, err)
			}
			if err := ytypes.ValidateUintRestrictions(args.yangType, val); err != nil {
				return "", yang.Ynone, fmt.Errorf("default value conversion: %q doesn't match int restrictions: %v", value, err)
			}
		}
		return value, ykind, nil
	case yang.Ydecimal64:
		val, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return "", yang.Ynone, fmt.Errorf("default value conversion: unable to convert default value %q to %v: %v", value, ykind, err)
		}
		if err := ytypes.ValidateDecimalRestrictions(args.yangType, val); err != nil {
			return "", yang.Ynone, fmt.Errorf("default value conversion: %q doesn't match int restrictions: %v", value, err)
		}
		return value, ykind, nil
	case yang.Ybinary:
		bytes, err := base64.StdEncoding.DecodeString(value)
		if err != nil {
			return "", yang.Ynone, fmt.Errorf("default value conversion: error in DecodeString for \n%v\n for type name %q: %q", value, args.yangType.Name, err)
		}
		if err := ytypes.ValidateBinaryRestrictions(args.yangType, bytes); err != nil {
			return "", yang.Ynone, fmt.Errorf("default value conversion: %q doesn't match binary restrictions: %v", value, err)
		}
		value := fmt.Sprintf(ygot.BinaryTypeName+"(%q)", value)
		return value, ykind, nil
	case yang.Ystring:
		if err := ytypes.ValidateStringRestrictions(args.yangType, value); err != nil {
			return "", yang.Ynone, fmt.Errorf("default value conversion: %q doesn't match string restrictions: %v", value, err)
		}
		value := fmt.Sprintf("%q", value)
		return value, ykind, nil
	case yang.Ybool:
		switch value {
		case "true", "false":
			return value, ykind, nil
		}
		return "", yang.Ynone, fmt.Errorf("default value conversion: cannot convert default value %q to bool, type name: %q", value, args.yangType.Name)
	case yang.Yempty:
		return "", yang.Ynone, fmt.Errorf("default value conversion: received default value %q, but an empty type cannot have a default value", value)
	case yang.Yenum:
		// Enumeration types need to be resolved to a particular data path such
		// that a created enumerated Go type can be used to set their value. Hand
		// the leaf to the enumName function to determine the name.
		if args.contextEntry == nil {
			return "", yang.Ynone, fmt.Errorf("default value conversion: cannot map enum without context")
		}
		if strings.Contains(value, ":") {
			value = strings.Split(value, ":")[1]
		}
		if !args.yangType.Enum.IsDefined(value) {
			return "", yang.Ynone, fmt.Errorf("default value conversion: enum value %q not found in enum with type name %q", value, args.yangType.Name)
		}
		n, _, err := s.EnumName(args.contextEntry, compressOCPaths, false, skipEnumDedup, shortenEnumLeafNames, false, enumOrgPrefixesToTrim)
		if err != nil {
			return "", yang.Ynone, err
		}
		return enumDefaultValue(n, value, ""), ykind, nil
	case yang.Yidentityref:
		// Identityref leaves are mapped according to the base identity that they
		// refer to - this is stored in the IdentityBase field of the context leaf
		// which is determined by the identityrefBaseTypeFromLeaf.
		if args.contextEntry == nil {
			return "", yang.Ynone, fmt.Errorf("default value conversion: cannot map identityref without context")
		}
		if strings.Contains(value, ":") {
			value = strings.Split(value, ":")[1]
		}
		if !args.yangType.IdentityBase.IsDefined(value) {
			return "", yang.Ynone, fmt.Errorf("default value conversion: identity value %q not found in enum with type name %q", value, args.yangType.Name)
		}
		n, _, err := s.IdentityrefBaseTypeFromIdentity(args.yangType.IdentityBase)
		if err != nil {
			return "", yang.Ynone, err
		}
		return enumDefaultValue(n, value, ""), ykind, nil
	case yang.Yleafref:
		// This is a leafref, so we check what the type of the leaf that it
		// references is by looking it up.
		target, err := s.ResolveLeafrefTarget(args.yangType.Path, args.contextEntry)
		if err != nil {
			return "", yang.Ynone, err
		}
		return s.yangDefaultValueToGo(value, resolveTypeArgs{yangType: target.Type, contextEntry: target}, isSingletonUnion, compressOCPaths, skipEnumDedup, shortenEnumLeafNames, useDefiningModuleForTypedefEnumNames, enumOrgPrefixesToTrim)
	case yang.Yunion:
		// Try to convert to each type in order, but try the enumerated types first.
		for _, t := range util.FlattenedTypes(args.yangType.Type) {
			snippet, convertedKind, err := s.yangDefaultValueToGo(value, resolveTypeArgs{yangType: t, contextEntry: args.contextEntry}, isSingletonUnion, compressOCPaths, skipEnumDedup, shortenEnumLeafNames, useDefiningModuleForTypedefEnumNames, enumOrgPrefixesToTrim)
			if err == nil {
				if !isSingletonUnion {
					if simpleName, ok := simpleUnionConversionsFromKind[convertedKind]; ok {
						snippet = fmt.Sprintf("%s(%s)", simpleName, snippet)
					}
				}
				return snippet, convertedKind, nil
			}
		}
		return "", yang.Ynone, fmt.Errorf("default value conversion: cannot convert default value %q to any union subtype, type name %q", value, args.yangType.Name)
	default:
		// Default values are not supported for unsupported types, so
		// just generate the zero value instead.
		// TODO(wenbli): support bit type.
		return "", yang.Ynone, fmt.Errorf("default value conversion: cannot create default value for unsupported type %v, type name: %q", ykind, args.yangType.Name)
	}
}

// goLeafDefaults returns the default value(s) of the leaf e if specified. If it
// is unspecified, the value specified by the type is returned if it is not nil,
// otherwise nil is returned to indicate no default was specified.
// TODO(wenbli): This doesn't handle unions. Deprecate this for v1 release.
func goLeafDefaults(e *yang.Entry, t *ygen.MappedType) []string {
	defaultValues := e.DefaultValues()
	if len(defaultValues) == 0 && t.DefaultValue != nil {
		defaultValues = []string{*t.DefaultValue}
	}

	for i, defVal := range defaultValues {
		if t.IsEnumeratedValue {
			defaultValues[i] = enumDefaultValue(t.NativeType, defVal, goEnumPrefix)
		} else {
			defaultValues[i] = quoteDefault(defVal, t.NativeType)
		}
	}

	return defaultValues
}

// quoteDefault adds quotation marks to the value string if the goType specified
// is a string, and hence requires quoting.
func quoteDefault(value string, goType string) string {
	if goType == "string" {
		return fmt.Sprintf("%q", value)
	}

	return value
}
