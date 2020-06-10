// Copyright 2019 Google Inc.
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
	"io"
	"os"
	"path/filepath"
	"strings"

	log "github.com/golang/glog"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/genutil"
	"github.com/openconfig/ygot/ypathgen"
)

var (
	yangPaths               = flag.String("path", "", "Comma separated list of paths to be recursively searched for included modules or submodules within the defined YANG modules.")
	excludeModules          = flag.String("exclude_modules", "", "Comma separated set of module names that should be excluded from code generation this can be used to ensure overlapping namespaces can be ignored.")
	packageName             = flag.String("package_name", "telemetry", "The name of the Go package that should be generated.")
	outputFile              = flag.String("output_file", "", "The single file that the Go package should be written to.")
	ignoreCircDeps          = flag.Bool("ignore_circdeps", false, "If set to true, circular dependencies between submodules are ignored.")
	preferOperationalState  = flag.Bool("prefer_operational_state", false, "If set to true, state (config false) fields in the YANG schema are preferred over intended config leaves when building paths. This flag is only valid when exclude_state=false.")
	fakeRootName            = flag.String("fakeroot_name", "device", "The name of the fake root entity. This name will be capitalized for exporting.")
	schemaStructPkgAlias    = flag.String("schema_struct_pkg_alias", "", "The package alias of the schema struct package.")
	schemaStructPath        = flag.String("schema_struct_path", "", "The import path to use for ygen-generated schema structs.")
	gnmiProtoPath           = flag.String("gnmi_proto_path", genutil.GoDefaultGNMIImportPath, "The import path to use for gNMI's proto package.")
	ygotImportPath          = flag.String("ygot_path", genutil.GoDefaultYgotImportPath, "The import path to use for ygot.")
	listBuilderKeyThreshold = flag.Uint("list_builder_key_threshold", 0, "The threshold equal or over which the builder API is used for key population. 0 means infinity.")
	skipEnumDedup           = flag.Bool("skip_enum_deduplication", false, "If set to true, all leaves of type enumeration will have a unique enum output for them, rather than sharing a common type (default behaviour).")
	shortenEnumLeafNames    = flag.Bool("shorten_enum_leaf_names", false, "If also set to true when compression is on, all leaves of type enumeration will by default not be prefixed with the name of its residing module.")
)

// writeGoCodeSingleFile takes a ypathgen.GeneratedPathCode struct and writes
// it to a single file to the io.Writer, w, provided as an argument.
// The output includes a package header which is generated.
func writeGoCodeSingleFile(w io.Writer, pathCode *ypathgen.GeneratedPathCode) error {
	_, err := io.WriteString(w, pathCode.String())
	return err
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

	if *schemaStructPath == "" {
		log.Exitln("Error: schemaStructPath unspecified")
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

	if *outputFile == "" {
		log.Exitln("Error: outputFile unspecified")
	}

	// Perform the code generation.
	cg := &ypathgen.GenConfig{
		PackageName: *packageName,
		GoImports: ypathgen.GoImports{
			SchemaStructPkgPath: *schemaStructPath,
			GNMIProtoPath:       *gnmiProtoPath,
			YgotImportPath:      *ygotImportPath,
		},
		PreferOperationalState: *preferOperationalState,
		SkipEnumDeduplication:  *skipEnumDedup,
		ShortenEnumLeafNames:   *shortenEnumLeafNames,
		FakeRootName:           *fakeRootName,
		ExcludeModules:         modsExcluded,
		SchemaStructPkgAlias:   "oc",
		YANGParseOptions: yang.Options{
			IgnoreSubmoduleCircularDependencies: *ignoreCircDeps,
		},
		GeneratingBinary:        genutil.CallerName(),
		ListBuilderKeyThreshold: *listBuilderKeyThreshold,
	}

	pathCode, _, errs := cg.GeneratePathCode(generateModules, includePaths)
	if errs != nil {
		log.Exitf("ERROR Generating Code: %s\n", errs)
	}

	// If no output file is specified, we output to os.Stdout, otherwise
	// we write to the specified file.
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

	writeGoCodeSingleFile(outfh, pathCode)
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
