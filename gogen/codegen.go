// Package gogen is a library for generating Go structs from a YANG schema.
package gogen

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/openconfig/ygot/internal/igenutil"
	"github.com/openconfig/ygot/util"
	"github.com/openconfig/ygot/ygen"
	"github.com/openconfig/ygot/ygot"
)

// CodeGenerator is a structure that is used to pass arguments as to
// how the output Go code should be generated.
type CodeGenerator struct {
	// Caller is the name of the binary calling the generator library, it is
	// included in the header of output files for debugging purposes. If a
	// string is not specified, the location of the library is utilised.
	Caller string
	// IROptions stores the configuration parameters used for IR generation.
	IROptions ygen.IROptions
	// GoOptions stores a struct which stores Go code generation specific
	// options for code generaton post IR generation.
	GoOptions GoOpts
}

// GoOpts stores Go specific options for the code generation library.
type GoOpts struct {
	// PackageName is the name that should be used for the generating package.
	PackageName string
	// GenerateJSONSchema stores a boolean which defines whether to generate
	// the JSON corresponding to the YANG schema parsed to generate the
	// output code.
	GenerateJSONSchema bool
	// IncludeDescriptions specifies that YANG entry descriptions are added
	// to the JSON schema. Is false by default, to reduce the size of generated schema
	IncludeDescriptions bool
	// SchemaVarName is the name for the variable which stores the compressed
	// JSON schema in the generated Go code. JSON schema output is only
	// produced if the GenerateJSONSchema field is set to true.
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
	// GenerateRenameMethod specifies whether methods for renaming list entries
	// should be generated in the output Go code.
	GenerateRenameMethod bool
	// AddAnnotationFields specifies whether annotation fields should be added to
	// the generated structs. When set to true, a metadata field is added for each
	// struct, and for each field of each struct. Metadata field's names are
	// prefixed by the string specified in the AnnotationPrefix argument.
	AddAnnotationFields bool
	// AnnotationPrefix specifies the string which is prefixed to the name of
	// annotation fields. It defaults to Λ.
	AnnotationPrefix string
	// AddYangPresence specifies whether tags should be added to the generated
	// fields of a struct. When set to true, a struct tag will be added to the field
	// when a YANG container is a presence container
	// https://datatracker.ietf.org/doc/html/rfc6020#section-7.5.1
	// a field tag of `yangPresence="true"` will only be added if the container is
	// a YANG presence container, and will be omitted if this is not the case.
	AddYangPresence bool
	// GenerateGetters specifies whether GetOrCreate* methods should be created
	// for struct pointer (YANG container) and map (YANG list) fields of generated
	// structs.
	GenerateGetters bool
	// GenerateDeleteMethod specifies whether Delete* methods should be created for
	// map (YANG list) fields of generated structs.
	GenerateDeleteMethod bool
	// GenerateAppendList specifies whether Append* methods should be created for
	// list fields of a struct. These methods take an input list member type, extract
	// the key and append the supplied value to the list.
	GenerateAppendMethod bool
	// GenerateSimpleUnions specifies whether simple typedefs are used to
	// represent union subtypes in the generated code instead of using
	// wrapper types.
	GenerateSimpleUnions bool
	// GenerateLeafGetters specifies whether Get* methods should be created for
	// leaf fields of a struct. Care should be taken with this option since a Get
	// method returns the *Go* zero value for a particular entity if the field is
	// unset. This means that it is not possible for a caller of method to know
	// whether a field has been explicitly set to the zero value (i.e., an integer
	// field is set to 0), or whether the field was actually unset.
	GenerateLeafGetters bool
	// GenerateLeafSetters specifies whether Set* methods should be created for
	// leaf fields of a struct.
	GenerateLeafSetters bool
	// GeneratePopulateDefault specifies whether a PopulateDefaults method
	// should be generated for every GoStruct that recursively populates
	// default values within the subtree.
	GeneratePopulateDefault bool
	// GNMIProtoPath specifies the path to the generated gNMI protobuf, which
	// is used to store the catalogue entries for generated modules.
	GNMIProtoPath string
	// ValidateFunctionName specifies the name of a function that proxies ΛValidate.
	ValidateFunctionName string
	// IncludeModelData specifies whether gNMI ModelData messages should be generated
	// in the output code.
	IncludeModelData bool
	// AppendEnumSuffixForSimpleUnionEnums appends an "Enum" suffix to the
	// enumeration name for simple (i.e. non-typedef) leaves which are
	// unions with an enumeration inside. This makes all inlined
	// enumerations within unions, whether typedef or not, have this
	// suffix, achieving consistency. Since this flag is planned to be a
	// v1 compatibility flag along with
	// UseDefiningModuleForTypedefEnumNames, and will be removed in v1, it
	// only applies when useDefiningModuleForTypedefEnumNames is also set
	// to true.
	AppendEnumSuffixForSimpleUnionEnums bool
	// IgnoreShadowSchemaPaths indicates whether when OpenConfig path
	// compression is enabled, that the shadowed paths are to be ignored
	// while while unmarshalling.
	IgnoreShadowSchemaPaths bool
	// GenerateOrderedListsAsUnorderedMaps indicates that lists that are
	// marked `ordered-by user` will be represented using built-in Go maps
	// instead of an ordered map Go structure.
	GenerateOrderedListsAsUnorderedMaps bool
}

// GeneratedCode contains generated code snippets that can be processed by the calling
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
type GeneratedCode struct {
	Structs      []GoStructCodeSnippet // Structs is the generated set of structs representing containers or lists in the input YANG models.
	Enums        []string              // Enums is the generated set of enum definitions corresponding to identities and enumerations in the input YANG models.
	CommonHeader string                // CommonHeader is the header that should be used for all output Go files.
	OneOffHeader string                // OneOffHeader defines the header that should be included in only one output Go file - such as package init statements.
	EnumMap      string                // EnumMap is a Go map that allows the YANG string values of enumerated types to be resolved.
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
	RawJSONSchema []byte
	// EnumTypeMap is a Go map that allows YANG schemapaths to be mapped to reflect.Type values.
	EnumTypeMap string
}

// New returns a new instance of the CodeGenerator
// struct to the calling function.
func New(callerName string, opts ygen.IROptions, goOpts GoOpts) *CodeGenerator {
	return &CodeGenerator{
		Caller:    callerName,
		IROptions: opts,
		GoOptions: goOpts,
	}
}

// checkForBinaryKeys returns a non-empty list of errors if the input directory
// has one or more binary types (including union types containing binary types)
// as a list key.
func checkForBinaryKeys(dir *ygen.ParsedDirectory) []error {
	var errs []error
	for _, k := range dir.ListKeys {
		if k.LangType.NativeType == ygot.BinaryTypeName {
			errs = append(errs, fmt.Errorf("list %s has a binary key -- this is unsupported", dir.Path))
			continue
		}
		for typeName := range k.LangType.UnionTypes {
			if typeName == ygot.BinaryTypeName {
				errs = append(errs, fmt.Errorf("list %s has a union key containing a binary -- this is unsupported", dir.Path))
			}
		}
	}
	return errs
}

// Generate takes a slice of strings containing the path to a set of YANG
// files which contain YANG modules, and a second slice of strings which
// specifies the set of paths that are to be searched for associated models (e.g.,
// modules that are included by the specified set of modules, or submodules of those
// modules). It extracts the set of modules that are to be generated, and returns
// a GeneratedCode struct which contains:
//  1. A struct definition for each container or list that is within the specified
//     set of models.
//  2. Enumerated values which correspond to the set of enumerated entities (leaves
//     of type enumeration, identities, typedefs that reference an enumeration)
//     within the specified models.
//
// If errors are encountered during code generation, an error is returned.
func (cg *CodeGenerator) Generate(yangFiles, includePaths []string) (*GeneratedCode, util.Errors) {
	opts := ygen.IROptions{
		ParseOptions:                        cg.IROptions.ParseOptions,
		TransformationOptions:               cg.IROptions.TransformationOptions,
		NestedDirectories:                   false,
		AbsoluteMapPaths:                    false,
		AppendEnumSuffixForSimpleUnionEnums: cg.GoOptions.AppendEnumSuffixForSimpleUnionEnums,
	}

	var codegenErr util.Errors
	ir, err := ygen.GenerateIR(yangFiles, includePaths, NewGoLangMapper(cg.GoOptions.GenerateSimpleUnions), opts)
	if err != nil {
		return nil, util.AppendErr(codegenErr, err)
	}

	var rootName string
	if cg.IROptions.TransformationOptions.GenerateFakeRoot {
		rootName = cg.IROptions.TransformationOptions.FakeRootName
		if rootName == "" {
			rootName = igenutil.DefaultRootName
		}
		if r, ok := ir.Directories[fmt.Sprintf("/%s", rootName)]; ok {
			rootName = r.Name
		}
	}
	commonHeader, oneoffHeader, err := writeGoHeader(yangFiles, includePaths, cg, rootName, ir.ModelData)
	if err != nil {
		return nil, util.AppendErr(codegenErr, err)
	}

	usedEnumeratedTypes := map[string]bool{}
	// generatedUnions stores a map, keyed by the output name for a union,
	// that has already been output in the generated code. This ensures that
	// where two entities re-use a union that has already been created (e.g.,
	// a leafref to a union) then it is output only once in the generated code.
	generatedUnions := map[string]bool{}
	enumTypeMap := map[string][]string{}
	structSnippets := []GoStructCodeSnippet{}

	isBuiltInType := func(fType string) bool {
		_, ok := validGoBuiltinTypes[fType]
		return ok
	}

	// Range through the directories to find the enumerated and union types that we
	// need. We have to do this without writing the code out, since we require some
	// knowledge of these types to do code generation along with the values.
	for _, directoryPath := range ir.OrderedDirectoryPathsByName() {
		dir := ir.Directories[directoryPath]

		// Generate structs.
		if errs := checkForBinaryKeys(dir); len(errs) != 0 {
			codegenErr = util.AppendErrs(codegenErr, errs)
			continue
		}
		structOut, errs := writeGoStruct(dir, ir.Directories, generatedUnions, cg.GoOptions)
		if errs != nil {
			codegenErr = util.AppendErrs(codegenErr, errs)
			continue
		}
		structSnippets = append(structSnippets, structOut)

		// Record down all the enum types we encounter in each field.

		// definedUnionTypes keeps track of which unions we have
		// already processed to avoid processing the same one twice.
		definedUnionTypes := map[string]bool{}
		for _, fn := range dir.OrderedFieldNames() {
			field := dir.Fields[fn]

			schemaPath := field.YANGDetails.SchemaPath
			switch {
			case field.LangType == nil:
				// This is a directory, so we continue.
				continue
			case field.LangType.IsEnumeratedValue:
				usedEnumeratedTypes[field.LangType.NativeType] = true
				enumTypeMap[schemaPath] = []string{field.LangType.NativeType}
			case len(field.LangType.UnionTypes) > 1:
				if definedUnionTypes[field.LangType.NativeType] {
					continue
				}
				definedUnionTypes[field.LangType.NativeType] = true

				for ut := range field.LangType.UnionTypes {
					if !isBuiltInType(ut) {
						// non-builtin union types are always enumerated types.
						usedEnumeratedTypes[ut] = true
						if enumTypeMap[schemaPath] == nil {
							enumTypeMap[schemaPath] = []string{}
						}
						enumTypeMap[schemaPath] = append(enumTypeMap[schemaPath], ut)
					}
				}
				// Sort the enumerated types into schema order.
				sort.Slice(enumTypeMap[schemaPath], func(i, j int) bool {
					return field.LangType.UnionTypes[enumTypeMap[schemaPath][i]].Index < field.LangType.UnionTypes[enumTypeMap[schemaPath][j]].Index
				})
			}
			if v, ok := enumTypeMap[schemaPath]; ok {
				if shadowPath := field.YANGDetails.ShadowSchemaPath; shadowPath != "" {
					enumTypeMap[shadowPath] = v
				}
			}
		}
	}

	processedEnums, err := genGoEnumeratedTypes(ir.Enums)
	if err != nil {
		return nil, append(codegenErr, err)
	}

	genum, err := writeGoEnumeratedTypes(processedEnums, usedEnumeratedTypes)
	if err != nil {
		return nil, append(codegenErr, err)
	}

	var rawSchema []byte
	var jsonSchema string
	var enumTypeMapCode string
	if cg.GoOptions.GenerateJSONSchema {
		var err error
		rawSchema, err = ir.SchemaTree(cg.GoOptions.IncludeDescriptions)
		if err != nil {
			codegenErr = util.AppendErr(codegenErr, fmt.Errorf("error marshalling JSON schema: %v", err))
		}

		if rawSchema != nil {
			if jsonSchema, err = writeGoSchema(rawSchema, cg.GoOptions.SchemaVarName); err != nil {
				codegenErr = util.AppendErr(codegenErr, err)
			}
		}

		if enumTypeMapCode, err = generateEnumTypeMap(enumTypeMap); err != nil {
			codegenErr = util.AppendErr(codegenErr, err)
		}
	}

	// Return any errors that were encountered during code generation.
	if len(codegenErr) != 0 {
		return nil, codegenErr
	}

	return &GeneratedCode{
		CommonHeader:   commonHeader,
		OneOffHeader:   oneoffHeader,
		Structs:        structSnippets,
		Enums:          genum.enums,
		EnumMap:        genum.valMap,
		JSONSchemaCode: jsonSchema,
		RawJSONSchema:  rawSchema,
		EnumTypeMap:    enumTypeMapCode,
	}, nil
}

// generateEnumTypeMap outputs a map using the enumTypeMap template. It takes an
// input of a map, keyed by schema path, to the string names of the enumerated
// types that can correspond to the schema path. The map generated allows a
// schemapath to be mapped into the reflect.Type representing the enum value.
func generateEnumTypeMap(enumTypeMap map[string][]string) (string, error) {
	var buf bytes.Buffer
	if err := goEnumTypeMapTemplate.Execute(&buf, enumTypeMap); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// writeGoSchema generates Go code which serialises the rawSchema byte slice
// provided and stores it in a variable which can be written out to the generated
// Go code file.
func writeGoSchema(js []byte, schemaVarName string) (string, error) {
	jbyte, err := ygen.WriteGzippedByteSlice(js)
	if err != nil {
		return "", fmt.Errorf("could not write Byte slice: %v", err)
	}

	vn := defaultSchemaVarName
	if schemaVarName != "" {
		vn = schemaVarName
	}

	in := struct {
		VarName string
		Schema  []string
	}{
		VarName: vn,
		Schema:  ygen.BytesToGoByteSlice(jbyte),
	}

	var buf bytes.Buffer
	if err := schemaVarTemplate.Execute(&buf, in); err != nil {
		return "", err
	}

	return buf.String(), nil
}
