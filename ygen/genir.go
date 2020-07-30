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

type IROptions struct {
	// ParseOptions specifies the options for how the YANG schema is
	// produced.
	ParseOptions ParseOpts

	// Transformation options specifies any transformations that should
	// be applied to the input YANG schema when producing the IR.
	TransformationOptions TransformationOpts
}

// GenerateIR creates the ygen intermediate representation for a set of
// YANG modules. The YANG files to be parsed are read from the yangFiles
// argument, with any includes that they use searched for in the string
// slice of paths specified by includePaths. The supplier LangMapper interface
// is used to perform mapping of language-specific naming whilst creating
// the IR -- the full details of the implementation of LangMapper can be found
// in ygen/ir.go and docs/code-generation-design.md.
//
// The supplied IROptions controls the parsing and transformations that are
// applied to the schema whilst generating the IR.
//
// GenerateIR returns the complete ygen intermediate representation.
func GenerateIR(yangFiles, includePaths []string, newLangMapper NewLangMapperFn, opts IROptions) (*IR, error) {
	// TODO(robjs): Implementation of GenerateIR.
	return nil, nil
}
