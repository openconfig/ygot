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
	"fmt"
	"sort"
	"strings"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygot"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
)

// children returns all child elements of a directory element e that are not
// RPC entries.
func children(e *yang.Entry) []*yang.Entry {
	var entries []*yang.Entry

	for _, e := range e.Dir {
		if e.RPC == nil {
			entries = append(entries, e)
		}
	}
	return entries
}

// hasOnlyChild returns true if the directory passed to it only has a single
// element below it.
func hasOnlyChild(e *yang.Entry) bool {
	return e.Dir != nil && len(children(e)) == 1
}

// isRoot returns true if the entry is an entity at the root of the tree.
func isRoot(e *yang.Entry) bool {
	return e.Parent == nil
}

// isConfigState returns true if the entry is an entity that represents a
// container called config or state.
func isConfigState(e *yang.Entry) bool {
	return e.IsDir() && (e.Name == "config" || e.Name == "state")
}

// isChoiceOrCase returns true if the entry is either a 'case' or a 'choice'
// node within the schema. These are schema nodes only, and the code generation
// operates on data tree paths.
func isChoiceOrCase(e *yang.Entry) bool {
	return e.Kind == yang.CaseEntry || e.Kind == yang.ChoiceEntry
}

// isUnionType returns true if the entry is a union within the YANG schema,
// checked by determining the length of the Type slice within the YangType.
func isUnionType(t *yang.YangType) bool {
	return len(t.Type) > 0
}

// isEnumType returns true if the entry is an enumerated type within the
// YANG schema - i.e., an enumeration or identityref leaf.
func isEnumType(t *yang.YangType) bool {
	return t.Kind == yang.Yenum || t.Kind == yang.Yidentityref
}

// isAnydata returns true if the entry is an Anydata node.
func isAnydata(e *yang.Entry) bool {
	return e.Kind == yang.AnyDataEntry
}

// isOCCompressedValidElement returns true if the element would be output in the
// compressed YANG code.
func isOCCompressedValidElement(e *yang.Entry) bool {
	switch {
	case hasOnlyChild(e) && children(e)[0].IsList():
		// This is a surrounding container for a list which is removed from the
		// structure.
		return false
	case isRoot(e):
		// This is a top-level module within the goyang structure, so is not output.
		return false
	case isConfigState(e):
		// This is a container that is called config or state, which is removed from
		// a compressed OpenConfig schema.
		return false
	case isChoiceOrCase(e):
		// This is a choice or case node that is removed from the overall schema
		// so code generation does not occur for it.
		return false
	}
	return true
}

// joinPath concatenates a slice of strings into a / separated path. i.e.,
// []string{"", "foo", "bar"} becomes "/foo/bar". Paths in YANG are generally
// represented in this return format, but the []string format is more flexible
// for internal use.
func joinPath(parts []string) string {
	var buf bytes.Buffer
	for i, p := range parts {
		buf.WriteString(p)
		if i != len(parts)-1 {
			buf.WriteRune('/')
		}
	}
	return buf.String()
}

// findFirstNonChoice traverses the data tree and determines the first directory
// nodes from a root e that are neither case nor choice nodes. The map, m, is
// updated in place to append new entries that are found when recursively
// traversing the set of choice/case nodes.
func findFirstNonChoice(e *yang.Entry, m map[string]*yang.Entry) {
	switch {
	case !isChoiceOrCase(e):
		m[e.Path()] = e
	case e.IsDir():
		for _, ch := range e.Dir {
			findFirstNonChoice(ch, m)
		}
	}
}

// removePrefix verifies whether there is a ":" character in a path element
// and returns the unprefixed path if so. This is to handle cases where YANG
// XPath expressions may specify paths in the form prefix:pathelement.
func removePrefix(s string) string {
	if !strings.ContainsRune(s, ':') {
		return s
	}
	return strings.Split(s, ":")[1]
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

// traverseElementSchemaPath takes an input yang.Entry and walks up the tree to find
// its path, expressed as a slice of strings, which is returned.
func traverseElementSchemaPath(elem *yang.Entry) []string {
	var pp []string
	e := elem
	for ; e.Parent != nil; e = e.Parent {
		if !isChoiceOrCase(e) {
			pp = append(pp, e.Name)
		}
	}
	pp = append(pp, e.Name)

	// Reverse the slice that was specified to us as it was appended to
	// from the leaf to the root.
	for i := len(pp)/2 - 1; i >= 0; i-- {
		o := len(pp) - 1 - i
		pp[i], pp[o] = pp[o], pp[i]
	}
	return pp
}

// isConfig takes a yang.Entry and traverses up the tree to find the config
// state of that element. In YANG, if the config parameter is unset, then it is
// is inherited from the parent of the element - hence we must walk up the tree to find
// the state. If the element at the top of the tree does not have config set, then config
// is true. See https://tools.ietf.org/html/rfc6020#section-7.19.1.
func isConfig(e *yang.Entry) bool {
	for ; e.Parent != nil; e = e.Parent {
		switch e.Config {
		case yang.TSTrue:
			return true
		case yang.TSFalse:
			return false
		}
	}

	// Reached the last element in the tree without explicit configuration
	// being set.
	if e.Config == yang.TSFalse {
		return false
	}
	return true
}

// isKeyedList returns true if the supplied yang.Entry represents a keyed list.
func isKeyedList(e *yang.Entry) bool {
	return e.IsList() && e.Key != ""
}

// isSimpleEnumerationType returns true wen the type supplied is a simple
// enumeration (i.e., a leaf that is defined as type enumeration { ... },
// and is not a typedef that contains an enumeration, or a union that
// contains an enumeration which may have enum values specified. The type
// name enumeration is used in these cases by goyang.
func isSimpleEnumerationType(t *yang.YangType) bool {
	return t.Kind == yang.Yenum && t.Name == "enumeration"
}

// isIdentityrefLeaf returns true if the supplied yang.Entry represents an
// identityref.
func isIdentityrefLeaf(e *yang.Entry) bool {
	return e.Type.IdentityBase != nil
}

// slicePathToString takes a path represented as a slice of strings, and outputs
// it as a single string, with path elements separated by a forward slash.
func slicePathToString(path []string) string {
	var buf bytes.Buffer
	for i, elem := range path {
		buf.WriteString(elem)
		if i != len(path)-1 {
			buf.WriteRune('/')
		}
	}
	return buf.String()
}

// safeGoEnumeratedValueName takes an input string, which is the name of an
// enumerated value from a YANG schema, and ensures that it is safe to be
// output as part of the name of the enumerated value in the Go code. The
// sanitised value is returned.  Per RFC6020 Section 6.2, a YANG identifier is
// of the form [_a-zA-Z][a-zA-Z0-9\-\.]+ - such that we must replace "." and
// "-" characters.  The implementation used here replaces [\.\-] with "_"
// characters.  In OpenConfig schemas, there are currently a small number of
// identity values that contain "." and hence must be specifically handled.
func safeGoEnumeratedValueName(name string) string {
	// NewReplacer takes pairs of strings to be replaced in the form
	// old, new.
	replacer := strings.NewReplacer(
		".", "_",
		"-", "_",
		"/", "_",
		"+", "_PLUS",
		"*", "_ASTERISK",
		" ", "_")
	return replacer.Replace(name)
}

// enumeratedUnionTypes recursively searches the set of yang.YangTypes supplied to
// extract the enumerated types that are within a union.
func enumeratedUnionTypes(types []*yang.YangType) []*yang.YangType {
	var eTypes []*yang.YangType
	for _, t := range types {
		switch {
		case isEnumType(t):
			eTypes = append(eTypes, t)
		case isUnionType(t):
			eTypes = append(eTypes, enumeratedUnionTypes(t.Type)...)
		}
	}
	return eTypes
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

// listKeyFieldsMap returns a map[string]bool where the keys of the map
// are the fields that are the keys of the list described by the supplied
// yang.Entry. In the case the yang.Entry does not described a keyed list,
// an empty map is returned.
func listKeyFieldsMap(e *yang.Entry) map[string]bool {
	r := map[string]bool{}
	for _, k := range strings.Split(e.Key, " ") {
		r[k] = true
	}
	return r
}

// entrySchemaPath takes an input yang.Entry, and returns its YANG schema
// path.
func entrySchemaPath(e *yang.Entry) string {
	return slicePathToString(append([]string{""}, traverseElementSchemaPath(e)[1:]...))
}

// isDirectEntryChild determines whether the entry c is a direct child of the
// entry p within the output code. If compressPaths is set, a check to determine
// whether c would be a direct child after schema compression is performed.
func isDirectEntryChild(p, c *yang.Entry, compressPaths bool) bool {
	ppp := strings.Split(p.Path(), "/")
	cpp := strings.Split(c.Path(), "/")
	dc := isPathChild(ppp, cpp)

	// If we are not compressing paths, then directly return whether the child
	// is a path of the parent.
	if !compressPaths {
		return dc
	}

	// If the length of the child path is greater than two larger than the
	// parent path, then this means that it cannot be a direct child, since all
	// path compression will remove only one level of hierarchy (config/state or
	// a surrounding container at maximum). We also check that the length of
	// the child path is more specific than or equal to the length of the parent
	// path in which case this cannot be a child.
	if len(cpp) > len(ppp)+2 || len(cpp) <= len(ppp) {
		return false
	}

	if isConfigState(c.Parent) {
		// If the parent of this entity was the config/state container, then this
		// level of the hierarchy will have been removed so we check whether the
		// parent of both are equal and return this.
		return p.Path() == c.Parent.Parent.Path()
	}

	// If the child is a list, then we check whether the parent has only one
	// child (i.e., is a surrounding container) and then check whether the
	// single child is the child we were provided.
	if c.IsList() {
		ppe, ok := p.Dir[c.Parent.Name]
		if !ok {
			// Can't be a valid child because the parent of the entity doesn't exist
			// within this container.
			return false
		}
		if !hasOnlyChild(ppe) {
			return false
		}

		// We are guaranteed to have 1 child (and not zero) since hasOnlyChild will
		// return false for directories with 0 children.
		return children(ppe)[0].Path() == c.Path()
	}

	return dc
}

// isPathChild takes an input slice of strings representing a path and determines
// whether b is a child of a within the YANG schema.
func isPathChild(a, b []string) bool {
	// If b does not have a greater path length than a, it cannot be a child. If
	// b has more than one element than a, it must be at least a grandchild.
	if len(b) <= len(a) || len(b) > len(a)+1 {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// isChildOfModule determines whether the yangDirectory represents a container
// or list member that is the direct child of a module entry.
func isChildOfModule(msg *yangDirectory) bool {
	if msg.isFakeRoot || len(msg.path) == 3 {
		// If the message has a path length of 3, then it is a top-level entity
		// within a module, since the  path is in the format []{"", <module>, <element>}.
		return true
	}
	return false
}

// isYANGBaseType determines whether the supplied YangType is a built-in type
// in YANG, or a derived type (i.e., typedef).
func isYANGBaseType(t *yang.YangType) bool {
	_, builtin := yang.TypeKindFromName[t.Name]
	return builtin
}

// typeDefaultValue returns the default value of the type t if it is specified.
// nil is returned if no default is specified.
func typeDefaultValue(t *yang.YangType) *string {
	if t.Default == "" {
		return nil
	}
	return ygot.String(t.Default)
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

type modelDataProto []*gpb.ModelData

func (m modelDataProto) Less(a, b int) bool { return m[a].Name < m[b].Name }
func (m modelDataProto) Len() int           { return len(m) }
func (m modelDataProto) Swap(a, b int)      { m[a], m[b] = m[b], m[a] }

// findModelData takes an input slice of yang.Entry pointers, which are assumed to
// represent YANG modules, and returns the gNMI ModelData that corresponds with each
// of the input modules.
func findModelData(mods []*yang.Entry) ([]*gpb.ModelData, error) {
	modelData := modelDataProto{}
	for _, mod := range mods {
		mNode, ok := mod.Node.(*yang.Module)
		if !ok || mNode == nil {
			return nil, fmt.Errorf("nil node, or not a module for node %s", mod.Name)
		}
		md := &gpb.ModelData{
			Name: mod.Name,
		}

		if mNode.Organization != nil {
			md.Organization = mNode.Organization.Statement().Argument
		}

		for _, e := range mNode.Exts() {
			if p := strings.Split(e.Keyword, ":"); len(p) == 2 && p[1] == "openconfig-version" {
				md.Version = e.Argument
				break
			}
		}

		modelData = append(modelData, md)
	}

	sort.Sort(modelData)

	return modelData, nil
}
