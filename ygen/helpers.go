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

import "github.com/openconfig/goyang/pkg/yang"

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
