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
	"bytes"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/genutil"
	"github.com/openconfig/ygot/util"
	"github.com/openconfig/ygot/ygot"
)

const (
	// goEnumPrefix is the prefix that is used for type names in the output
	// Go code, such that an enumeration's name is of the form
	//   <goEnumPrefix><EnumName>
	goEnumPrefix string = "E_"
)

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
)

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

	if IsFakeRoot(e) {
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

// buildDirectoryDefinitions extracts the yang.Entry instances from a map of
// entries that need struct definitions built for them. It resolves each
// non-leaf yang.Entry to a Directory which contains the elements that are
// needed for subsequent code generation, with the relationships between the
// elements being determined by the compress behaviour and genFakeRoot (whether
// a fake root element is generated). The skipEnumDedup argument specifies to
// the code generation whether to try to output a single type for an
// enumeration that is logically defined once in the output code, but
// instantiated in multiple places in the schema tree.  The skipEnumDedup
// argument specifies whether leaves of type 'enumeration' which are used more
// than once in the schema should use a common output type in the generated Go
// code. By default a type is shared.
func (s *goGenState) buildDirectoryDefinitions(entries map[string]*yang.Entry, compBehaviour genutil.CompressBehaviour, genFakeRoot, skipEnumDedup, shortenEnumLeafNames, useDefiningModuleForTypedefEnumNames bool) (map[string]*Directory, []error) {
	return buildDirectoryDefinitions(entries, compBehaviour,
		// For Go, we map the name of the struct to the path elements
		// in CamelCase separated by underscores.
		func(e *yang.Entry) string {
			return s.goStructName(e, compBehaviour.CompressEnabled(), genFakeRoot)
		},
		func(keyleaf *yang.Entry) (*MappedType, error) {
			return s.yangTypeToGoType(resolveTypeArgs{yangType: keyleaf.Type, contextEntry: keyleaf}, compBehaviour.CompressEnabled(), skipEnumDedup, shortenEnumLeafNames, useDefiningModuleForTypedefEnumNames)
		})
}
