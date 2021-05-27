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

package genutil

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/openconfig/goyang/pkg/yang"
)

// CallerName returns the name of the Go binary that is currently running.
func CallerName() string {
	// Find out the name of this binary so that it can be used for debug
	// reasons.
	_, currentCodeFile, _, ok := runtime.Caller(0)
	if !ok {
		// In the case that we cannot determine the current running binary's name
		// this is non-fatal, so return a default string.
		return "unknown - unable to determine calling binary name"
	}
	return currentCodeFile
}

// definingModule returns the name of the module that defined the yang.Node
// supplied. If node is within a submodule, the parent module name is returned.
func definingModule(node yang.Node) yang.Node {
	var definingMod yang.Node
	definingMod = yang.RootNode(node)
	if definingMod.Kind() == "submodule" {
		// A submodule must always be a *yang.Module.
		mod := definingMod.(*yang.Module)
		definingMod = mod.BelongsTo
	}
	return definingMod
}

// ParentModuleName returns the name of the module or submodule that defined
// the supplied node.
func ParentModuleName(node yang.Node) string {
	return definingModule(node).NName()
}

// ParentModulePrettyName returns the name of the module that defined the yang.Node
// supplied as the node argument. If the discovered root node of the node is found
// to be a submodule, the name of the parent module is returned. If the root has
// a camel case extension, this is returned rather than the actual module name.
// If organization prefixes (e.g. "openconfig", "ietf") are given, they are
// trimmed from the module name if a match is found.
func ParentModulePrettyName(node yang.Node, orgPrefixesToTrim ...string) string {
	definingMod := definingModule(node)
	if name, ok := CamelCaseNameExt(definingMod.Exts()); ok {
		return name
	}

	return yang.CamelCase(TrimOrgPrefix(definingMod.NName(), orgPrefixesToTrim...))
}

// TrimOrgPrefix checks each input organization prefix (e.g. "openconfig", "ietf")
// (https://tools.ietf.org/html/rfc8407#section-4.1), and if matching the input
// module name, trims it and returns it. If none is matching, the original
// module name is returned.
// E.g. If "openconfig" is provided as a prefix to trim, then
// "openconfig-interfaces" becomes simply "interfaces".
func TrimOrgPrefix(modName string, orgPrefixesToTrim ...string) string {
	for _, pfx := range orgPrefixesToTrim {
		if trimmedModName := strings.TrimPrefix(modName, pfx+"-"); trimmedModName != modName {
			return trimmedModName
		}
	}
	return modName
}

// MakeNameUnique makes the name specified as an argument unique based on the names
// already defined within a particular context which are specified within the
// definedNames map. If the name has already been defined, an underscore is appended
// to the name until it is unique.
func MakeNameUnique(name string, definedNames map[string]bool) string {
	for {
		if _, nameUsed := definedNames[name]; !nameUsed {
			definedNames[name] = true
			return name
		}
		name = fmt.Sprintf("%s_", name)
	}
}

// EntryCamelCaseName returns the camel case version of the Entry Name field, or
// the CamelCase name that is specified by a "camelcase-name" extension on the
// field. The returned name is not guaranteed to be unique within any context.
func EntryCamelCaseName(e *yang.Entry) string {
	if name, ok := CamelCaseNameExt(e.Exts); ok {
		return name
	}
	return yang.CamelCase(e.Name)
}

// CamelCaseNameExt returns the CamelCase name from the slice of extensions, if
// one of the extensions is named "camelcase-name". It returns the a string
// containing the name if the bool return argument is set to true; otherwise no
// such extension was specified.
// TODO(wenbli): add unit test
func CamelCaseNameExt(exts []*yang.Statement) (string, bool) {
	// Check the extensions to determine whethere an extension
	// exists that specifies the camelcase name of the entity. If so
	// use this as the name in the structs.
	// TODO(robjs): Add more robust parsing into goyang such that rather
	// than having a Statement here, we have some more concrete type to
	// parse within the Extras field. This would allow robust validation
	// of the module in which the extension is defined.
	var name string
	var ok bool
	r := strings.NewReplacer(`\n`, ``, `"`, ``)
	for _, s := range exts {
		if p := strings.Split(s.Keyword, ":"); len(p) < 2 || p[1] != "camelcase-name" || !s.HasArgument {
			continue
		}
		name = r.Replace(s.Argument)
		ok = true
		break
	}
	return name, ok
}
