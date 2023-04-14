// Copyright 2023 Google Inc.
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

package gogen

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/openconfig/ygot/ygen"
	"github.com/openconfig/ygot/ygot"
)

// goEnumeratedType contains the intermediate representation of an enumerated
// type (identityref or enumeration) suitable for Go code generation.
type goEnumeratedType struct {
	Name       string
	CodeValues map[int64]string
	YANGValues map[int64]ygot.EnumDefinition
}

// enumGeneratedCode contains generated Go code for enumerated types.
type enumGeneratedCode struct {
	enums  []string
	valMap string
}

// genGoEnumeratedTypes converts the input map of EnumeratedYANGType objects to
// another intermediate representation suitable for Go code generation.
func genGoEnumeratedTypes(enums map[string]*ygen.EnumeratedYANGType) (map[string]*goEnumeratedType, error) {
	et := map[string]*goEnumeratedType{}
	for _, e := range enums {
		// initialised to be UNSET, such that it is possible to determine that the enumerated value
		// was not modified.
		values := map[int64]string{
			0: "UNSET",
		}

		// origValues stores the original set of value names, these are not maintained to be
		// Go-safe, and are rather used to map back to the original schema values if required.
		// 0 is not populated within this map, such that the values can be used to check whether
		// there was a valid entry in the original schema. The value is stored as a ygot
		// EnumDefinition, which stores the name, and in the case of identity values, the
		// module within which the identity was defined.
		origValues := map[int64]ygot.EnumDefinition{}

		switch e.Kind {
		case ygen.IdentityType, ygen.SimpleEnumerationType, ygen.DerivedEnumerationType, ygen.UnionEnumerationType, ygen.DerivedUnionEnumerationType:
			for _, v := range e.ValToYANGDetails {
				values[int64(v.Value)+1] = safeGoEnumeratedValueName(v.Name)
				origValues[int64(v.Value)+1] = v
			}
		default:
			return nil, fmt.Errorf("unknown enumerated type %v", e.Kind)
		}

		et[e.Name] = &goEnumeratedType{
			Name:       e.Name,
			CodeValues: values,
			YANGValues: origValues,
		}
	}
	return et, nil
}

// writeGoEnumeratedTypes generates Go code for the input enumerations if they
// are present in the usedEnums map.
func writeGoEnumeratedTypes(enums map[string]*goEnumeratedType, usedEnums map[string]bool) (*enumGeneratedCode, error) {
	orderedEnumNames := []string{}
	for _, e := range enums {
		orderedEnumNames = append(orderedEnumNames, e.Name)
	}
	sort.Strings(orderedEnumNames)

	enumValMap := map[string]map[int64]ygot.EnumDefinition{}
	enumSnippets := []string{}

	for _, en := range orderedEnumNames {
		e := enums[en]
		if _, ok := usedEnums[fmt.Sprintf("%s%s", goEnumPrefix, e.Name)]; !ok {
			// Don't output enumerated types that are not used in the code that we have
			// such that we don't create generated code for a large array of types that
			// just happen to be in modules that were included by other modules.
			continue
		}
		enumOut, err := writeGoEnum(e)
		if err != nil {
			return nil, err
		}
		enumSnippets = append(enumSnippets, enumOut)
		enumValMap[e.Name] = e.YANGValues
	}

	// Write the map of string -> int -> YANG enum name string out.
	vmap, err := writeGoEnumMap(enumValMap)
	if err != nil {
		return nil, err
	}

	return &enumGeneratedCode{
		enums:  enumSnippets,
		valMap: vmap,
	}, nil
}

// writeGoEnum takes an input goEnumeratedType, and generates the code corresponding
// to it. If errors are encountered whilst mapping the enumeration to
// code, they are returned. The enumDefinition template is used to convert a
// constructed generatedGoEnumeration struct to code within the function.
func writeGoEnum(inputEnum *goEnumeratedType) (string, error) {
	var buf strings.Builder
	if err := goEnumDefinitionTemplate.Execute(&buf, generatedGoEnumeration{
		EnumerationPrefix: inputEnum.Name,
		Values:            inputEnum.CodeValues,
	}); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// writeGoEnumMap takes in a enumerated value map firstly keyed by the name of
// the enumerated type, then by the enumerated type value. It outputs a piece
// of generated Go code from which this information can be accessed
// programmatically.
func writeGoEnumMap(enums map[string]map[int64]ygot.EnumDefinition) (string, error) {
	if len(enums) == 0 {
		return "", nil
	}

	var buf bytes.Buffer
	if err := goEnumMapTemplate.Execute(&buf, enums); err != nil {
		return "", err
	}
	return buf.String(), nil
}
