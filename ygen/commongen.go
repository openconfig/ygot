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
	"runtime"
)

// commonCodeHeaderParams stores common parameters which are included
// in the header of generated code.
type commonCodeHeaderParams struct {
	PackageName      string   // PackgeName is the name of the package to be generated.
	YANGFiles        []string // YANGFiles contains the list of input YANG source files for code generation.
	IncludePaths     []string // IncludePaths contains the list of paths that included modules were searched for in.
	CompressEnabled  bool     // CompressEnabled indicates whether CompressOCPaths was set.
	GeneratingBinary string   // GeneratingBinary is the name of the binary generating the code.
	GenerateSchema   bool     // GenerateSchema stores whether the generator requested that the schema was to be stored with the output code.
}

// buildCommonHeader constructs the commonCodeHeaderParams struct that a caller can use
// in a template to output a package header. The package name, compress settings, and caller
// are gleaned from the supplied YANGCodeGenerator struct if they are defined - with the input files,
// and paths within which includes are found learnt from the yangFiles and includePaths
// arguments. Returns a commonCodeHeaderParams struct.
func buildCommonHeader(packageName, caller string, compressPaths bool, yangFiles, includePaths []string, generateSchema bool) *commonCodeHeaderParams {
	// Find out the name of this binary so that it can be included in the
	// generated code for debug reasons. It is dynamically learnt based on
	// review suggestions that this code may move in the future.
	_, currentCodeFile, _, ok := runtime.Caller(0)
	switch {
	case caller != "":
		// If the caller was specifically overridden, then use the specified
		// value rather than the code name.
		currentCodeFile = caller
	case !ok:
		// This is a non-fatal error, since it simply means we can't
		// find the current file. At this point, we do not want to abandon
		// what otherwise would be successful code generation, so give
		// an identifiable string.
		currentCodeFile = "codegen"
	}

	return &commonCodeHeaderParams{
		PackageName:      packageName,
		YANGFiles:        yangFiles,
		IncludePaths:     includePaths,
		CompressEnabled:  compressPaths,
		GeneratingBinary: currentCodeFile,
		GenerateSchema:   generateSchema,
	}
}
