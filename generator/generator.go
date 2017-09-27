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
	"os"
	"path/filepath"
	"strings"

	log "github.com/golang/glog"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygen"
)

var (
	yangPaths        = flag.String("path", "", "Comma separated list of paths to be recursively searched for included modules or submodules within the defined YANG modules.")
	compressPaths    = flag.Bool("compress_paths", false, "If set to true, the schema's paths are compressed, according to OpenConfig YANG module conventions.")
	excludeModules   = flag.String("exclude_modules", "", "Comma separated set of module names that should be excluded from code generation this can be used to ensure overlapping namespaces can be ignored.")
	packageName      = flag.String("package_name", "ocstructs", "The name of the Go package that should be generated.")
	outputFile       = flag.String("output_file", "", "The file that the generated Go code should be written to.")
	ignoreCircDeps   = flag.Bool("ignore_circdeps", false, "If set to true, circular dependencies between submodules are ignored.")
	generateFakeRoot = flag.Bool("generate_fakeroot", false, "If set to true, a fake element at the root of the data tree is generated. By default the fake root entity is named Device, its name can be controlled with the fakeroot_name flag.")
	fakeRootName     = flag.String("fakeroot_name", "", "The name of the fake root entity.")
	generateSchema   = flag.Bool("include_schema", true, "If set to true, the YANG schema will be encoded as JSON and stored in the generated code artefact.")
	ygotImportPath   = flag.String("ygot_path", ygen.DefaultYgotImportPath, "The import path to use for ygot.")
	ytypesImportPath = flag.String("ytypes_path", ygen.DefaultYtypesImportPath, "The import path to use for ytypes.")
	goyangImportPath = flag.String("goyang_path", ygen.DefaultGoyangImportPath, "The import path to use for goyang's yang package.")
)

// writeGoCode takes a ygen.GeneratedGoCode struct and writes the Go code
// snippets contained within it to the io.Writer, w, provided as an argument.
// The output includes a package header which is generated.
func writeGoCode(w io.Writer, goCode *ygen.GeneratedGoCode) error {
	// Write the package header to the supplier writer.
	fmt.Fprint(w, goCode.Header)

	// Write the returned Go code out. First the Structs - which is the struct
	// definitions for the generated YANG entity, followed by the enumerations.
	for _, codeSnippets := range [][]string{goCode.Structs, goCode.Enums} {
		for _, snippet := range codeSnippets {
			fmt.Fprintln(w, snippet)
		}
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

	// If no output file is specified, we output to os.Stdout, otherwise
	// we write to the specified file.
	var outfh *os.File
	switch *outputFile {
	case "":
		outfh = os.Stdout
	default:
		fileOut, err := os.Create(*outputFile)
		if err != nil {
			log.Exitf("Error: could not open output file: %v\n", err)
		}

		// Assign the newly created filehandle to the outfh, and ensure
		// that it is synced and closed before exit of main.
		outfh = fileOut
		defer func() {
			if err := outfh.Sync(); err != nil {
				log.Exitf("Error: could not sync file output: %v\n", err)
			}

			if err := outfh.Close(); err != nil {
				log.Exitf("Error: could not close output file: %v\n", err)
			}
		}()
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
			YgotImportPath:   *ygotImportPath,
			YtypesImportPath: *ytypesImportPath,
			GoyangImportPath: *goyangImportPath,
		},
	})

	generatedGoCode, err := cg.GenerateGoCode(generateModules, includePaths)
	if err != nil {
		log.Exitf("ERROR Generating Code: %s\n", err)
	}

	// Write out the Go code to the specified file handle.
	writeGoCode(outfh, generatedGoCode)
}
