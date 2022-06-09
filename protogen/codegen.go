package protogen

import (
	"fmt"
	"sort"
	"strings"

	"github.com/openconfig/ygot/util"
	"github.com/openconfig/ygot/ygen"
)

// CodeGenerator is a structure that is used to pass arguments as to
// how the output protobuf code should be generated.
type CodeGenerator struct {
	// Caller is the name of the binary calling the generator library, it is
	// included in the header of output files for debugging purposes. If a
	// string is not specified, the location of the library is utilised.
	Caller string
	// IROptions stores the configuration parameters used for IR generation.
	IROptions ygen.IROptions
	// ProtoOptions stores a struct which contains Protobuf specific
	// options for code generation post IR generation.
	ProtoOptions ProtoOpts
}

// ProtoOpts stores Protobuf specific options for the code generation library.
type ProtoOpts struct {
	// PackageName is the name that should be used for the generating package.
	PackageName string
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

// New returns a new instance of the CodeGenerator
// struct to the calling function.
func New(callerName string, opts ygen.IROptions, protoOpts ProtoOpts) *CodeGenerator {
	return &CodeGenerator{
		Caller:       callerName,
		IROptions:    opts,
		ProtoOptions: protoOpts,
	}
}

// GeneratedCode stores a set of generated Protobuf packages.
type GeneratedCode struct {
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

// Generate generates Protobuf 3 code for the input set of YANG files.
// The YANG schemas for which protobufs are to be created is supplied as the
// yangFiles argument, with included modules being searched for in includePaths.
// It returns a GeneratedCode struct containing the messages that are to be
// output, along with any associated values (e.g., enumerations).
func (cg *CodeGenerator) Generate(yangFiles, includePaths []string) (*GeneratedCode, util.Errors) {
	basePackageName := cg.ProtoOptions.PackageName
	if basePackageName == "" {
		basePackageName = DefaultBasePackageName
	}
	enumPackageName := cg.ProtoOptions.EnumPackageName
	if enumPackageName == "" {
		enumPackageName = DefaultEnumPackageName
	}
	ywrapperPath := cg.ProtoOptions.YwrapperPath
	if ywrapperPath == "" {
		ywrapperPath = DefaultYwrapperPath
	}
	yextPath := cg.ProtoOptions.YextPath
	if yextPath == "" {
		yextPath = DefaultYextPath
	}

	// This flag is always true for proto generation.
	cg.IROptions.TransformationOptions.UseDefiningModuleForTypedefEnumNames = true
	opts := ygen.IROptions{
		ParseOptions:                        cg.IROptions.ParseOptions,
		TransformationOptions:               cg.IROptions.TransformationOptions,
		NestedDirectories:                   cg.ProtoOptions.NestedMessages,
		AbsoluteMapPaths:                    true,
		AppendEnumSuffixForSimpleUnionEnums: true,
	}

	ir, err := ygen.GenerateIR(yangFiles, includePaths, NewProtoLangMapper(basePackageName, enumPackageName), opts)
	if err != nil {
		return nil, util.NewErrs(err)
	}

	protoEnums, err := writeProtoEnums(ir.Enums, cg.ProtoOptions.AnnotateEnumNames)
	if err != nil {
		return nil, util.NewErrs(err)
	}

	genProto := &GeneratedCode{
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
			UsesYextImport: cg.ProtoOptions.AnnotateEnumNames,
		}
	}

	// Ensure that the slice of messages returned is in a deterministic order by
	// sorting the message paths. We use the path rather than the name as the
	// proto message name may not be unique.
	for _, directoryPath := range ir.OrderedDirectoryPaths() {
		m := ir.Directories[directoryPath]

		genMsg, errs := writeProto3Msg(m, ir, &protoMsgConfig{
			compressPaths:       cg.IROptions.TransformationOptions.CompressBehaviour.CompressEnabled(),
			basePackageName:     basePackageName,
			enumPackageName:     enumPackageName,
			baseImportPath:      cg.ProtoOptions.BaseImportPath,
			annotateSchemaPaths: cg.ProtoOptions.AnnotateSchemaPaths,
			annotateEnumNames:   cg.ProtoOptions.AnnotateEnumNames,
			nestedMessages:      cg.ProtoOptions.NestedMessages,
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
		if cg.ProtoOptions.GoPackageBase != "" {
			gpn = fmt.Sprintf("%s/%s", cg.ProtoOptions.GoPackageBase, strings.ReplaceAll(n, ".", "/"))
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
			CompressPaths:          cg.IROptions.TransformationOptions.CompressBehaviour.CompressEnabled(),
			CallerName:             cg.Caller,
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
