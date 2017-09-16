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

// Binary proto_generator generates Protobuf3 code corresponding to an input
// YANG schema. The input set of modules are read, parsed using goyang, and
// handled as input to the ygen package which generates the corresponding
// set of Protobuf3 messages.
package main

import (
	"flag"
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
	packageName      = flag.String("package_name", "openconfig", "The name of the Proto package that generated messages should belong to as their parent.")
	enumPackageName  = flag.String("enum_package_name", "enums", "The name of the package within the generated package that should contain global enum definitions.")
	outputDir        = flag.String("output_dir", "", "The path to which files should be output, hierarchical folders are created for the generated messages.")
	ignoreCircDeps   = flag.Bool("ignore_circdeps", false, "If set to true, circular dependencies between submodules are ignored.")
	generateFakeRoot = flag.Bool("generate_fakeroot", false, "If set to true, a fake element at the root of the data tree is generated. By default the fake root entity is named Device, its name can be controlled with the fakeroot_name flag.")
	fakeRootName     = flag.String("fakeroot_name", "", "The name of the fake root entity.")
)

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

	if *outputDir == "" {
		log.Exitln("Error: an output directory must be specified")
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

	// Perform the code generation.
	cg := ygen.NewYANGCodeGenerator(&ygen.GeneratorConfig{
		CompressOCPaths:  *compressPaths,
		ExcludeModules:   modsExcluded,
		PackageName:      *packageName,
		GenerateFakeRoot: *generateFakeRoot,
		FakeRootName:     *fakeRootName,
		YANGParseOptions: yang.Options{
			IgnoreSubmoduleCircularDependencies: *ignoreCircDeps,
		},
	})

	generatedProtoCode, err := cg.GenerateProto3(generateModules, includePaths)
	if err != nil {
		log.Exitf("%v\n", err)
	}

	for _, p := range generatedProtoCode.Packages {
		fp := filepath.Join(append([]string{*outputDir}, p.FilePath[:len(p.FilePath)-1]...)...)
		if err := os.MkdirAll(fp, 0755); err != nil {
			log.Exitf("could not create directory %v, got error: %v", fp, err)
		}

		f, err := os.Create(filepath.Join(fp, p.FilePath[len(p.FilePath)-1]))
		if err != nil {
			log.Exitf("could not create file %v, got error: %v", fp, err)
		}
		defer f.Close()

		f.WriteString(p.Header)
		for _, m := range p.Messages {
			f.WriteString(m)
		}
		for _, e := range p.Enums {
			f.WriteString(e)
		}
		f.Sync()
	}
}
