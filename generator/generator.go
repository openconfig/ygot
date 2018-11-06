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
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	log "github.com/golang/glog"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygen"
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
	// methodFn is the filename to be used for methods related to the structs when outputting
	// to a directory.
	methodFn = "methods.go"
	// structBaseFn is the base filename to be used for files containing structs when outputting
	// to a directory. Structs are divided alphabetically and the first character appended to the
	// base specified in this value - e.g., structs beginning with "A" are stored in {structBaseFnA}.go.
	structBaseFn = "structs_"
)

var (
	yangPaths           = flag.String("path", "", "Comma separated list of paths to be recursively searched for included modules or submodules within the defined YANG modules.")
	compressPaths       = flag.Bool("compress_paths", false, "If set to true, the schema's paths are compressed, according to OpenConfig YANG module conventions.")
	excludeModules      = flag.String("exclude_modules", "", "Comma separated set of module names that should be excluded from code generation this can be used to ensure overlapping namespaces can be ignored.")
	packageName         = flag.String("package_name", "ocstructs", "The name of the Go package that should be generated.")
	outputFile          = flag.String("output_file", "", "The file that the generated Go code should be written to.")
	outputDir           = flag.String("output_dir", "", "The directory that the Go package should be written to.")
	ignoreCircDeps      = flag.Bool("ignore_circdeps", false, "If set to true, circular dependencies between submodules are ignored.")
	generateFakeRoot    = flag.Bool("generate_fakeroot", false, "If set to true, a fake element at the root of the data tree is generated. By default the fake root entity is named Device, its name can be controlled with the fakeroot_name flag.")
	fakeRootName        = flag.String("fakeroot_name", "", "The name of the fake root entity.")
	generateSchema      = flag.Bool("include_schema", true, "If set to true, the YANG schema will be encoded as JSON and stored in the generated code artefact.")
	ygotImportPath      = flag.String("ygot_path", ygen.DefaultYgotImportPath, "The import path to use for ygot.")
	ytypesImportPath    = flag.String("ytypes_path", ygen.DefaultYtypesImportPath, "The import path to use for ytypes.")
	goyangImportPath    = flag.String("goyang_path", ygen.DefaultGoyangImportPath, "The import path to use for goyang's yang package.")
	generateRename      = flag.Bool("generate_rename", false, "If set to true, rename methods are generated for lists within the Go code.")
	addAnnotations      = flag.Bool("annotations", false, "If set to true, metadata annotations are added within the generated structs.")
	annotationPrefix    = flag.String("annotation_prefix", ygen.DefaultAnnotationPrefix, "String to be appended to each metadata field within the generated structs if annoations is set to true.")
	excludeState        = flag.Bool("exclude_state", false, "If set to true, state (config false) fields in the YANG schema are not included in the generated Go code.")
	generateAppend      = flag.Bool("generate_append", false, "If set to true, append methods are generated for YANG lists (Go maps) within the Go code.")
	generateGetters     = flag.Bool("generate_getters", false, "If set to true, getter methdos that retrieve or create an element are generated for YANG container (Go struct pointer) or list (Go map) fields within the generated code.")
	generateDelete      = flag.Bool("generate_delete", false, "If set to true, delete methods are generated for YANG lists (Go maps) within the Go code.")
	generateLeafGetters = flag.Bool("generate_leaf_getters", false, "If set to true, getters for YANG leaves are generated within the Go code. Caution should be exercised when using leaf getters, since values that are explicitly set to the Go default/zero value are not distinguishable from those that are unset when retrieved via the GetXXX method.")
	includeModelData    = flag.Bool("include_model_data", false, "If set to true, a slice of gNMI ModelData messages are included in the generated Go code containing the details of the input schemas from which the code was generated.")
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

// writeIfNotEmpty writes the string s to b if it has a non-zero length.
func writeIfNotEmpty(b *bytes.Buffer, s string) {
	if len(s) != 0 {
		b.WriteString(s)
	}
}

// codeOut describes an output file for Go code.
type codeOut struct {
	// contents is the code that is contained in the output file.
	contents string
	// oneoffHeader indicates whether the one-off header should be included in this
	// file.
	oneoffHeader bool
}

// makeOutputSpec generates a map, keyed by filename, to a codeOut struct containing
// the code to be output to that filename. It allows division of a ygen.GeneratedGoCode
// struct into a set of source files. It divides the methods, interfaces, and enumeration
// code snippets into their own files. Structs are output into files dependent on the
// first letter of their name within the code.
func makeOutputSpec(goCode *ygen.GeneratedGoCode) map[string]codeOut {
	var methodCode, interfaceCode bytes.Buffer
	structCode := map[byte]*bytes.Buffer{}
	for _, s := range goCode.Structs {
		// Index by the first character of the struct.
		fc := s.StructName[0]
		if _, ok := structCode[fc]; !ok {
			structCode[fc] = &bytes.Buffer{}
		}
		cs := structCode[fc]
		writeIfNotEmpty(cs, s.StructDef)
		writeIfNotEmpty(cs, fmt.Sprintf("%s\n", s.ListKeys))
		writeIfNotEmpty(&methodCode, fmt.Sprintf("%s\n", s.Methods))
		writeIfNotEmpty(&interfaceCode, fmt.Sprintf("%s\n", s.Interfaces))
	}

	emap := &bytes.Buffer{}
	writeIfNotEmpty(emap, goCode.EnumMap)
	if emap.Len() != 0 {
		emap.WriteString("\n")
	}
	writeIfNotEmpty(emap, goCode.EnumTypeMap)

	out := map[string]codeOut{
		enumMapFn:   {contents: emap.String()},
		schemaFn:    {contents: goCode.JSONSchemaCode},
		interfaceFn: {contents: interfaceCode.String()},
		methodFn:    {contents: methodCode.String(), oneoffHeader: true},
		enumFn:      {contents: strings.Join(goCode.Enums, "\n")},
	}

	for fn, code := range structCode {
		out[fmt.Sprintf("%s%c.go", structBaseFn, fn)] = codeOut{
			contents: code.String(),
		}
	}

	return out
}

// writeGoCodeMultipleFiles writes the input goCode to a set of files as specified
// by specification returned by output spec.
func writeGoCodeMultipleFiles(dir string, goCode *ygen.GeneratedGoCode) error {
	out := makeOutputSpec(goCode)

	for fn, f := range out {
		if len(f.contents) == 0 {
			continue
		}
		fh := openFile(filepath.Join(dir, fn))
		defer syncFile(fh)
		fmt.Fprintln(fh, goCode.CommonHeader)
		if f.oneoffHeader {
			fmt.Fprintln(fh, goCode.OneOffHeader)
		}
		fmt.Fprintln(fh, f.contents)
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

	if *outputFile != "" && *outputDir != "" {
		log.Exitf("Error: cannot specify both outputFile (%s) and outputDir (%s)", *outputFile, *outputDir)
	}

	// Perform the code generation.
	cg := ygen.NewYANGCodeGenerator(&ygen.GeneratorConfig{
		CompressOCPaths:    *compressPaths,
		ExcludeModules:     modsExcluded,
		PackageName:        *packageName,
		GenerateFakeRoot:   *generateFakeRoot,
		FakeRootName:       *fakeRootName,
		GenerateJSONSchema: *generateSchema,
		YANGParseOptions: yang.Options{
			IgnoreSubmoduleCircularDependencies: *ignoreCircDeps,
		},
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
			IncludeModelData:     *includeModelData,
		},
		ExcludeState: *excludeState,
	})

	generatedGoCode, err := cg.GenerateGoCode(generateModules, includePaths)
	if err != nil {
		log.Exitf("ERROR Generating Code: %s\n", err)
	}

	// If no output file is specified, we output to os.Stdout, otherwise
	// we write to the specified file.
	if *outputFile != "" {
		var outfh *os.File
		switch *outputFile {
		case "":
			outfh = os.Stdout
		default:
			// Assign the newly created filehandle to the outfh, and ensure
			// that it is synced and closed before exit of main.
			outfh = openFile(*outputFile)
			defer syncFile(outfh)
		}

		writeGoCodeSingleFile(outfh, generatedGoCode)
		return
	}

	// Write the Go code to a series of output files.
	writeGoCodeMultipleFiles(*outputDir, generatedGoCode)
}

// openFile opens a file with the supplied name, logging and exiting if it cannot
// be opened.
func openFile(fn string) *os.File {
	fileOut, err := os.Create(fn)
	if err != nil {
		log.Exitf("Error: could not open output file: %v\n", err)
	}
	return fileOut
}

// syncFile synchronises the supplied os.File and closes it.
func syncFile(fh *os.File) {
	if err := fh.Sync(); err != nil {
		log.Exitf("Error: could not sync file output: %v\n", err)
	}

	if err := fh.Close(); err != nil {
		log.Exitf("Error: could not close output file: %v\n", err)
	}
}
