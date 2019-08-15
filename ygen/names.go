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
	"github.com/openconfig/goyang/pkg/yang"
)

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

// parentModuleName returns the name of the module or submodule that defined
// the supplied node.
func parentModuleName(node yang.Node) string {
	return definingModule(node).NName()
}

// parentModulePrettyName returns the name of the module that defined the yang.Node
// supplied as the node argument. If the discovered root node of the node is found
// to be a submodule, the name of the parent module is returned. If the root has
// a camel case extension, this is returned rather than the actual module name.
func parentModulePrettyName(node yang.Node) string {
	definingMod := definingModule(node)
	if name, ok := camelCaseNameExt(definingMod.Exts()); ok {
		return name
	}

	return definingMod.NName()
}
