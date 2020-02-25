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
	"fmt"
	"strings"

	"github.com/openconfig/ygot/ygot"
)

// safeGoEnumeratedValueName takes an input string, which is the name of an
// enumerated value from a YANG schema, and ensures that it is safe to be
// output as part of the name of the enumerated value in the Go code. The
// sanitised value is returned.  Per RFC6020 Section 9.6.4,
// "The enum Statement [...] takes as an argument a string which is the
// assigned name. The string MUST NOT be empty and MUST NOT have any
// leading or trailing whitespace characters. The use of Unicode control
// codes SHOULD be avoided."
// Note: this rule is distinct and looser than the rule for YANG identifiers.
// The implementation used here replaces some (not all) characters allowed
// in a YANG enum assigned name but not in Go code. Current support is based
// on real-world feedback e.g. in OpenConfig schemas, there are currently
// a small number of identity values that contain "." and hence
// must be specifically handled.
func safeGoEnumeratedValueName(name string) string {
	// NewReplacer takes pairs of strings to be replaced in the form
	// old, new.
	replacer := strings.NewReplacer(
		".", "_",
		"-", "_",
		"/", "_",
		"+", "_PLUS",
		",", "_COMMA",
		"@", "_AT",
		"$", "_DOLLAR",
		"*", "_ASTERISK",
		":", "_COLON",
		" ", "_")
	return replacer.Replace(name)
}

// addNewKeys appends entries from the newKeys string slice to the
// existing map if the entry is not an existing key. The existing
// map is modified in place.
func addNewKeys(existing map[string]interface{}, newKeys []string) {
	for _, n := range newKeys {
		if _, ok := existing[n]; !ok {
			existing[n] = true
		}
	}
}

// stringKeys returns the keys of the supplied map as a slice of strings.
func stringKeys(m map[string]interface{}) []string {
	var ss []string
	for k := range m {
		ss = append(ss, k)
	}
	return ss
}

// enumDefaultValue sanitises a default value specified for an enumeration
// which can be specified as prefix:value in the YANG schema. The baseName
// is used as the generated enumeration name stripping any prefix specified,
// (allowing removal of the enumeration type prefix if required). The default
// value in the form <sanitised_baseName>_<sanitised_defVal> is returned as
// a pointer.
func enumDefaultValue(baseName, defVal, prefix string) *string {
	if strings.Contains(defVal, ":") {
		defVal = strings.Split(defVal, ":")[1]
	}

	if prefix != "" {
		baseName = strings.TrimPrefix(baseName, prefix)
	}

	return ygot.String(fmt.Sprintf("%s_%s", baseName, defVal))
}

// resolveRootName resolves the name of the fakeroot by taking configuration
// and the default values, along with a boolean indicating whether the fake
// root is to be generated. It returns an empty string if the root is not
// to be generated.
func resolveRootName(name, defName string, generateRoot bool) string {
	if !generateRoot {
		return ""
	}

	if name == "" {
		return defName
	}

	return name
}
