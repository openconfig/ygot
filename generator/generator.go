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

// Binary generator generates Go code corresponding to an input YANG schema.
// The input set of YANG modules are read, parsed using Goyang, and handed as
// input to the codegen package which generates the corresponding Go code.
package main

import (
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"

	log "github.com/golang/glog"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/genutil"
	"github.com/openconfig/ygot/ygen"
	"github.com/openconfig/ygot/ypathgen"
)

const (
	// enumMapFn is the filename to be used for the enum map when Go code is output to a directory.
	enumMapFn = "enum_map.go"
	// enumFn is the filename to be used for the enum code when Go code is output to a directory.
	enumFn = "enum.go"
	// schemaFn is the filename to be used for the schema code when outputting to a directory.
	schemaFn = "schema.go"
	// interfaceFn is the filename to be used for interface code when outputting to a directory.
	interfaceFn = "union.go"
	// structsFileFmt is the format string filename (missing index) to be
	// used for files containing structs when outputting to a directory.
	structsFileFmt = "structs-%d.go"
	// pathStructsFileFmt is the format string filename (missing index) to
	// be used for the path structs when path struct code is output to a directory.
	pathStructsFileFmt = "path_structs-%d.go"
)

var (
	generateGoStructs       = flag.Bool("generate_structs", true, "If true, then Go code for YANG path construction (schema/Go structs) will be generated.")
	generatePathStructs     = flag.Bool("generate_path_structs", false, "If true, then Go code for YANG path construction (path structs) will be generated.")
	ocStructsOutputFile     = flag.String("output_file", "", "The file that the generated Go code for manipulating YANG data (schema/Go structs) should be written to. Specify \"-\" for stdout.")
	structsFileN            = flag.Int("structs_split_files_count", 0, "The number of files to split the generated schema structs into when output_file is specified.")
	ocPathStructsOutputFile = flag.String("path_structs_output_file", "", "The file that the generated Go code for YANG path construction (path structs) will be generated. Specify \"-\" for stdout.")
	pathStructsFileN        = flag.Int("path_structs_split_files_count", 0, "The number of files to split the generated path structs into when output_file is specified for generating path structs")
	outputDir               = flag.String("output_dir", "", "The directory that the generated Go code should be written to. This is common between schema structs and path structs.")
	compressPaths           = flag.Bool("compress_paths", false, "If set to true, the schema's paths are compressed, according to OpenConfig YANG module conventions. Path structs generation currently only supports compressed paths.")

	// Common flags used for GoStruct and PathStruct generation.
	yangPaths                            = flag.String("path", "", "Comma separated list of paths to be recursively searched for included modules or submodules within the defined YANG modules.")
	excludeModules                       = flag.String("exclude_modules", "", "Comma separated set of module names that should be excluded from code generation this can be used to ensure overlapping namespaces can be ignored.")
	packageName                          = flag.String("package_name", "ocstructs", "The name of the Go package that should be generated.")
	ignoreCircDeps                       = flag.Bool("ignore_circdeps", false, "If set to true, circular dependencies between submodules are ignored.")
	fakeRootName                         = flag.String("fakeroot_name", "", "The name of the fake root entity.")
	skipEnumDedup                        = flag.Bool("skip_enum_deduplication", false, "If set to true, all leaves of type enumeration will have a unique enum output for them, rather than sharing a common type (default behaviour).")
	preferOperationalState               = flag.Bool("prefer_operational_state", false, "If set to true, state (config false) fields in the YANG schema are preferred over intended config leaves in the generated Go code with compressed schema paths. This flag is only valid for compress_paths=true and exclude_state=false.")
	shortenEnumLeafNames                 = flag.Bool("shorten_enum_leaf_names", false, "If also set to true when compress_paths=true, all leaves of type enumeration will by default not be prefixed with the name of its residing module.")
	useDefiningModuleForTypedefEnumNames = flag.Bool("typedef_enum_with_defmod", false, "If set to true, all typedefs of type enumeration or identity will be prefixed with the name of its module of definition instead of its residing module.")
	ygotImportPath                       = flag.String("ygot_path", genutil.GoDefaultYgotImportPath, "The import path to use for ygot.")

	// Flags used for GoStruct generation only.
	generateFakeRoot     = flag.Bool("generate_fakeroot", false, "If set to true, a fake element at the root of the data tree is generated. By default the fake root entity is named Device, its name can be controlled with the fakeroot_name flag.")
	generateSchema       = flag.Bool("include_schema", true, "If set to true, the YANG schema will be encoded as JSON and stored in the generated code artefact.")
	ytypesImportPath     = flag.String("ytypes_path", genutil.GoDefaultYtypesImportPath, "The import path to use for ytypes.")
	goyangImportPath     = flag.String("goyang_path", genutil.GoDefaultGoyangImportPath, "The import path to use for goyang's yang package.")
	generateRename       = flag.Bool("generate_rename", false, "If set to true, rename methods are generated for lists within the Go code.")
	addAnnotations       = flag.Bool("annotations", false, "If set to true, metadata annotations are added within the generated structs.")
	annotationPrefix     = flag.String("annotation_prefix", ygen.DefaultAnnotationPrefix, "String to be appended to each metadata field within the generated structs if annoations is set to true.")
	excludeState         = flag.Bool("exclude_state", false, "If set to true, state (config false) fields in the YANG schema are not included in the generated Go code.")
	generateAppend       = flag.Bool("generate_append", false, "If set to true, append methods are generated for YANG lists (Go maps) within the Go code.")
	generateGetters      = flag.Bool("generate_getters", false, "If set to true, getter methdos that retrieve or create an element are generated for YANG container (Go struct pointer) or list (Go map) fields within the generated code.")
	generateDelete       = flag.Bool("generate_delete", false, "If set to true, delete methods are generated for YANG lists (Go maps) within the Go code.")
	generateLeafGetters  = flag.Bool("generate_leaf_getters", false, "If set to true, getters for YANG leaves are generated within the Go code. Caution should be exercised when using leaf getters, since values that are explicitly set to the Go default/zero value are not distinguishable from those that are unset when retrieved via the GetXXX method.")
	generateSimpleUnions = flag.Bool("generate_simple_unions", false, "If set to true, then generated typedefs will be used to represent union subtypes within Go code instead of wrapper struct types.")
	includeModelData     = flag.Bool("include_model_data", false, "If set to true, a slice of gNMI ModelData messages are included in the generated Go code containing the details of the input schemas from which the code was generated.")

	// Flags used for PathStruct generation only.
	schemaStructPath        = flag.String("schema_struct_path", "", "The Go import path for the schema structs package. This should be specified if and only if schema structs are not being generated at the same time as path structs.")
	listBuilderKeyThreshold = flag.Uint("list_builder_key_threshold", 0, "The threshold equal or over which the path structs' builder API is used for key population. 0 means infinity.")
	pathStructSuffix        = flag.String("path_struct_suffix", "Path", "The suffix string appended to each generated path struct in order to differentiate their names from their corresponding schema struct names.")
)

// writeGoCodeSingleFile takes a ygen.GeneratedGoCode struct and writes the Go code
// snippets contained within it to the io.Writer, w, provided as an argument.
// The output includes a package header which is generated.
func writeGoCodeSingleFile(w io.Writer, goCode *ygen.GeneratedGoCode) error {
	// Write the package header to the supplier writer.
	fmt.Fprint(w, goCode.CommonHeader)
	fmt.Fprint(w, goCode.OneOffHeader)

	// Write the returned Go code out. First the Structs - which is the struct
	// definitions for the generated YANG entity, followed by the enumerations.
	for _, snippet := range goCode.Structs {
		fmt.Fprintln(w, snippet.String())
	}

	for _, snippet := range goCode.Enums {
		fmt.Fprintln(w, snippet)
	}

	// Write the generated enumeration map out.
	fmt.Fprintln(w, goCode.EnumMap)

	// Write the schema out if it was received.
	if len(goCode.JSONSchemaCode) > 0 {
		fmt.Fprintln(w, goCode.JSONSchemaCode)
	}

	if len(goCode.EnumTypeMap) > 0 {
		fmt.Fprintln(w, goCode.EnumTypeMap)
	}

	return nil
}

// writeGoPathCodeSingleFile takes a ypathgen.GeneratedPathCode struct and writes
// it to a single file to the io.Writer, w, provided as an argument.
// The output includes a package header which is generated.
func writeGoPathCodeSingleFile(w io.Writer, pathCode *ypathgen.GeneratedPathCode) error {
	_, err := io.WriteString(w, pathCode.String())
	return err
}

// splitCodeByFileN generates a map, keyed by filename, to a string containing
// the code to be output to that filename. It allows division of a
// ygen.GeneratedGoCode struct into a set of source files. It divides the
// methods, interfaces, and enumeration code snippets into their own files.
// Structs are output into files by splitting them evenly among the input split
// number.
func splitCodeByFileN(goCode *ygen.GeneratedGoCode, fileN int) (map[string]string, error) {
	structN := len(goCode.Structs)
	if fileN < 1 || fileN > structN {
		return nil, fmt.Errorf("requested %d files, but must be between 1 and %d (number of schema structs)", fileN, structN)
	}

	out := map[string]string{
		schemaFn: goCode.JSONSchemaCode,
		enumFn:   strings.Join(goCode.Enums, "\n"),
	}

	var structFiles []string
	var code, interfaceCode strings.Builder
	structsPerFile := int(math.Ceil(float64(structN) / float64(fileN)))
	// Empty files could appear with certain structN/fileN combinations due
	// to the ceiling numbers being used for structsPerFile.
	// e.g. 4/3 gives two files of two structs.
	// This is a little more complex, but spreads out the structs more evenly.
	// If we instead use the floor number, and put all remainder structs in
	// the last file, we might double the last file's number of structs if we get unlucky.
	// e.g. 99/10 assigns 18 structs to the last file.
	emptyFiles := fileN - int(math.Ceil(float64(structN)/float64(structsPerFile)))
	code.WriteString(goCode.OneOffHeader)
	for i, s := range goCode.Structs {
		code.WriteString(s.StructDef)
		code.WriteString(s.ListKeys)
		code.WriteString("\n")
		code.WriteString(s.Methods)
		if s.Methods != "" {
			code.WriteString("\n")
		}
		interfaceCode.WriteString(s.Interfaces)
		if s.Interfaces != "" {
			interfaceCode.WriteString("\n")
		}
		// The last file contains the remainder of the structs.
		if i == structN-1 || (i+1)%structsPerFile == 0 {
			structFiles = append(structFiles, code.String())
			code.Reset()
		}
	}
	for i := 0; i != emptyFiles; i++ {
		structFiles = append(structFiles, "")
	}

	for i, structFile := range structFiles {
		out[fmt.Sprintf(structsFileFmt, i)] = structFile
	}

	code.Reset()
	code.WriteString(goCode.EnumMap)
	if code.Len() != 0 {
		code.WriteString("\n")
	}
	code.WriteString(goCode.EnumTypeMap)

	out[enumMapFn] = code.String()
	out[interfaceFn] = interfaceCode.String()

	for name, code := range out {
		out[name] = goCode.CommonHeader + code
	}

	return out, nil
}

// writeFiles creates or truncates files in a given base directory and writes
// to them. Keys of the contents map are file names, and values are the
// contents to be written. An error is returned if the base directory does not
// exist. If a file cannot be written, the function aborts with the error,
// leaving an unspecified set of the other input files written with their given
// contents.
func writeFiles(dir string, out map[string]string) error {
	for filename, contents := range out {
		if len(contents) == 0 {
			continue
		}
		fh := genutil.OpenFile(filepath.Join(dir, filename))
		if fh == nil {
			return fmt.Errorf("could not open file %q", filename)
		}
		if _, err := fh.WriteString(contents); err != nil {
			return err
		}
		// flush & close written files before function finishes.
		defer genutil.SyncFile(fh)
	}

	return nil
}

// main parses command-line flags to determine the set of YANG modules for
// which code generation should be performed, and calls the codegen library
// to generate Go code corresponding to their schema. The output is written
// to the specified file.
func main() {
	flag.Parse()
	// Extract the set of modules that code is to be generated for,
	// throwing an error if the set is empty.
	generateModules := flag.Args()
	if len(generateModules) == 0 {
		log.Exitln("Error: no input modules specified")
	}

	if !*generateGoStructs && !*generatePathStructs {
		log.Exitf("Error: Neither schema structs nor path structs generation is enabled.")
	}

	if *generatePathStructs {
		if *generateGoStructs && *schemaStructPath != "" {
			log.Exitf("Error: provided non-empty schema_struct_path for import by path structs file(s), but schema structs are also to be generated within the same package.")
		}
		if !*generateGoStructs && *schemaStructPath == "" {
			log.Exitf("Error: need to provide schema_struct_path for import by path structs file(s) when schema structs are not being generated at the same time.")
		}
	}

	// Determine the set of paths that should be searched for included
	// modules. This is supplied by the user as a set of comma-separated
	// paths, so we split the string. Additionally, for each path
	// specified, we append "..." to ensure that the directory is
	// recursively searched.
	includePaths := []string{}
	if len(*yangPaths) > 0 {
		pathParts := strings.Split(*yangPaths, ",")
		for _, path := range pathParts {
			includePaths = append(includePaths, filepath.Join(path, "..."))
		}
	}

	// Determine which modules the user has requested to be excluded from
	// code generation.
	modsExcluded := []string{}
	if len(*excludeModules) > 0 {
		modParts := strings.Split(*excludeModules, ",")
		for _, mod := range modParts {
			modsExcluded = append(modsExcluded, mod)
		}
	}

	if *generateGoStructs {
		generateGoStructsSingleFile := *ocStructsOutputFile != ""
		generateGoStructsMultipleFiles := *outputDir != ""
		if generateGoStructsSingleFile && generateGoStructsMultipleFiles {
			log.Exitf("Error: cannot specify both output_file (%s) and output_dir (%s)", *ocStructsOutputFile, *outputDir)
		}
		if !generateGoStructsSingleFile && !generateGoStructsMultipleFiles {
			log.Exitf("Error: Go struct generation requires a specified output file or output directory.")
		}

		compressBehaviour, err := genutil.TranslateToCompressBehaviour(*compressPaths, *excludeState, *preferOperationalState)
		if err != nil {
			log.Exitf("ERROR Generating Code: %v\n", err)
		}

		// Perform the code generation.
		cg := ygen.NewYANGCodeGenerator(&ygen.GeneratorConfig{
			ParseOptions: ygen.ParseOpts{
				ExcludeModules:        modsExcluded,
				SkipEnumDeduplication: *skipEnumDedup,
				YANGParseOptions: yang.Options{
					IgnoreSubmoduleCircularDependencies: *ignoreCircDeps,
				},
			},
			TransformationOptions: ygen.TransformationOpts{
				CompressBehaviour:                    compressBehaviour,
				GenerateFakeRoot:                     *generateFakeRoot,
				FakeRootName:                         *fakeRootName,
				ShortenEnumLeafNames:                 *shortenEnumLeafNames,
				UseDefiningModuleForTypedefEnumNames: *useDefiningModuleForTypedefEnumNames,
			},
			PackageName:        *packageName,
			GenerateJSONSchema: *generateSchema,
			GoOptions: ygen.GoOpts{
				YgotImportPath:       *ygotImportPath,
				YtypesImportPath:     *ytypesImportPath,
				GoyangImportPath:     *goyangImportPath,
				GenerateRenameMethod: *generateRename,
				AddAnnotationFields:  *addAnnotations,
				AnnotationPrefix:     *annotationPrefix,
				GenerateGetters:      *generateGetters,
				GenerateDeleteMethod: *generateDelete,
				GenerateAppendMethod: *generateAppend,
				GenerateLeafGetters:  *generateLeafGetters,
				GenerateSimpleUnions: *generateSimpleUnions,
				IncludeModelData:     *includeModelData,
			},
		})

		generatedGoCode, errs := cg.GenerateGoCode(generateModules, includePaths)
		if errs != nil {
			log.Exitf("ERROR Generating GoStruct Code: %v\n", errs)
		}

		switch {
		case generateGoStructsSingleFile:
			var outfh *os.File
			switch *ocStructsOutputFile {
			case "-":
				// If "-" is the output file name, we output to os.Stdout, otherwise
				// we write to the specified file.
				outfh = os.Stdout
			default:
				// Assign the newly created filehandle to the outfh, and ensure
				// that it is synced and closed before exit of main.
				outfh = genutil.OpenFile(*ocStructsOutputFile)
				defer genutil.SyncFile(outfh)
			}

			writeGoCodeSingleFile(outfh, generatedGoCode)
		case generateGoStructsMultipleFiles:
			// Write the Go code to a series of output files.
			out, err := splitCodeByFileN(generatedGoCode, *structsFileN)
			if err != nil {
				log.Exitf("ERROR writing split GoStruct Code: %v\n", err)
			}
			if err := writeFiles(*outputDir, out); err != nil {
				log.Exitf("Error while writing schema struct files: %v", err)
			}
		}
	}

	// Generate PathStructs.
	if !*generatePathStructs {
		return
	}
	if !*compressPaths {
		log.Exitf("Error: path struct generation not supported for uncompressed paths. Please use compressed paths or remove output file flag for path struct generation.")
	}

	generatePathStructsSingleFile := *ocPathStructsOutputFile != ""
	generatePathStructsMultipleFiles := *outputDir != ""
	if generatePathStructsSingleFile && generatePathStructsMultipleFiles {
		log.Exitf("Error: cannot specify both path_structs_output_file (%s) and output_dir (%s)", *ocPathStructsOutputFile, *outputDir)
	}
	if !generatePathStructsSingleFile && !generatePathStructsMultipleFiles {
		log.Exitf("Error: path struct generation requires a specified output file or directory.")
	}

	// Perform the code generation.
	pcg := &ypathgen.GenConfig{
		PackageName: *packageName,
		GoImports: ypathgen.GoImports{
			SchemaStructPkgPath: *schemaStructPath,
			YgotImportPath:      *ygotImportPath,
		},
		PreferOperationalState:               *preferOperationalState,
		SkipEnumDeduplication:                *skipEnumDedup,
		ShortenEnumLeafNames:                 *shortenEnumLeafNames,
		UseDefiningModuleForTypedefEnumNames: *useDefiningModuleForTypedefEnumNames,
		FakeRootName:                         *fakeRootName,
		PathStructSuffix:                     *pathStructSuffix,
		ExcludeModules:                       modsExcluded,
		YANGParseOptions: yang.Options{
			IgnoreSubmoduleCircularDependencies: *ignoreCircDeps,
		},
		GeneratingBinary:        genutil.CallerName(),
		ListBuilderKeyThreshold: *listBuilderKeyThreshold,
	}

	pathCode, _, errs := pcg.GeneratePathCode(generateModules, includePaths)
	if errs != nil {
		log.Exitf("ERROR Generating PathStruct Code: %s\n", errs)
	}

	switch {
	case generatePathStructsSingleFile:
		var outfh *os.File
		switch *ocPathStructsOutputFile {
		case "-":
			// If "-" is the output file name, we output to os.Stdout, otherwise
			// we write to the specified file.
			outfh = os.Stdout
		default:
			// Assign the newly created filehandle to the outfh, and ensure
			// that it is synced and closed before exit of main.
			outfh = genutil.OpenFile(*ocPathStructsOutputFile)
			defer genutil.SyncFile(outfh)
		}

		writeGoPathCodeSingleFile(outfh, pathCode)
	case generatePathStructsMultipleFiles:
		out := map[string]string{}
		// Split the path struct code into files.
		files, err := pathCode.SplitFiles(*pathStructsFileN)
		if err != nil {
			log.Exitf("Error while splitting path structs code into %d files: %v\n", pathStructsFileN, err)
		}
		for i, file := range files {
			out[fmt.Sprintf(pathStructsFileFmt, i)] = file
		}
		if err := writeFiles(*outputDir, out); err != nil {
			log.Exitf("Error while writing path struct files: %v", err)
		}
	}
}
