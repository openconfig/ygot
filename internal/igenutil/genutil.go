// Copyright 2022 Google Inc.
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

// Package igenutil contains internal generation utilities.
package igenutil

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/util"

	log "github.com/golang/glog"
)

const (
	// RootElementNodeName is the synthesised node name that is used for an
	// element that represents the root. Such an element is generated only
	// when the GenerateFakeRoot bool is set to true within the
	// YANGCodeGenerator instance used as a receiver.
	RootElementNodeName = "!fakeroot!"
	// DefaultRootName is the default name for the root structure if GenerateFakeRoot is
	// set to true.
	DefaultRootName = "device"
)

var (
	// TemplateHelperFunctions specifies a set of functions that are supplied as
	// helpers to the templates that are used within this file.
	TemplateHelperFunctions = template.FuncMap{
		// inc provides a means to add 1 to a number, and is used within templates
		// to check whether the index of an element within a loop is the last one,
		// such that special handling can be provided for it (e.g., not following
		// it with a comma in a list of arguments).
		"inc": func(i int) int {
			return i + 1
		},
		"toUpper": strings.ToUpper,
		"indentLines": func(s string) string {
			var b bytes.Buffer
			p := strings.Split(s, "\n")
			b.WriteRune('\n')
			for i, l := range p {
				if l == "" {
					continue
				}
				b.WriteString(fmt.Sprintf("  %s", l))
				if i != len(p)-1 {
					b.WriteRune('\n')
				}
			}
			return b.String()
		},
		// stripAsteriskPrefix provides a template helper that removes an asterisk
		// from the start of a string. It is used to remove "*" from the start of
		// pointer types.
		"stripAsteriskPrefix": func(s string) string { return strings.TrimPrefix(s, "*") },
	}
)

// IsFakeRoot checks whether a given entry is the generated fake root.
func IsFakeRoot(e *yang.Entry) bool {
	return e != nil && e.Node != nil && e.Node.NName() == RootElementNodeName
}

// MappableLeaf determines whether the yang.Entry e is leaf with an
// enumerated value, such that the referenced enumerated type (enumeration or
// identity) should have code generated for it. If it is an enumerated type
// the leaf is returned.
func MappableLeaf(e *yang.Entry) *yang.Entry {
	if e.Type == nil {
		// If the type of the leaf is nil, then this is not a valid
		// leaf within the schema - since goyang must populate the
		// entry Type.
		// TODO(robjs): Add this as an error case that can be handled by
		// the caller directly.
		log.Warningf("got unexpected nil value type for leaf %s (%s), entry: %v", e.Name, e.Path(), e)
		return nil
	}

	var types []*yang.YangType
	switch {
	case util.IsEnumeratedType(e.Type):
		// Handle the case that this leaf is an enumeration or identityref itself.
		// This also handles cases where the leaf is a typedef that is an enumeration
		// or identityref, since the util.IsEnumeratedType check does not use the name of the
		// type.
		types = append(types, e.Type)
	case util.IsUnionType(e.Type):
		// Check for leaves that include a union that itself
		// includes an identityref or enumerated value.
		types = append(types, util.EnumeratedUnionTypes(e.Type.Type)...)
	}

	if types != nil {
		return e
	}
	return nil
}
