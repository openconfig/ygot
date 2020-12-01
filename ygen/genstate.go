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
	"sort"
	"strings"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/genutil"
	"github.com/openconfig/ygot/util"
	"github.com/openconfig/ygot/ygot"
)

// MappedType is used to store the generated language type that a leaf entity
// in YANG is mapped to. The NativeType is always populated for any leaf.
// UnionTypes is populated when the type may have subtypes (i.e., is a union).
// enumValues is populated when the type is an enumerated type.
//
// The code generation explicitly maps YANG types to corresponding generated
// language types. In the case that an explicit mapping is not specified, a
// type will be mapped to an empty interface (interface{}). For an explicit
// list of types that are supported, see the yangTypeTo*Type functions in this
// package.
type MappedType struct {
	// NativeType is the type which is to be used for the mapped entity.
	NativeType string
	// UnionTypes is a map, keyed by the generated type name, of the types
	// specified as valid for a union. The value of the map indicates the
	// order of the type, since order is important for unions in YANG.
	// Where two types are mapped to the same generated language type
	// (e.g., string) then only the order of the first is maintained. Since
	// the generated code from the structs maintains only type validation,
	// this is not currently a limitation.
	UnionTypes map[string]int
	// IsEnumeratedValue specifies whether the NativeType that is returned
	// is a generated enumerated value. Such entities are reflected as
	// derived types with constant values, and are hence not represented
	// as pointers in the output code.
	IsEnumeratedValue bool
	// ZeroValue stores the value that should be used for the type if
	// it is unset. This is used only in contexts where the nil pointer
	// cannot be used, such as leaf getters.
	ZeroValue string
	// DefaultValue stores the default value for the type if is specified.
	// It is represented as a string pointer to ensure that default values
	// of the empty string can be distinguished from unset defaults.
	DefaultValue *string
}

// IsYgenDefinedGoType returns true if the native type of a MappedType is a Go
// type that's defined by ygen's generated code.
func IsYgenDefinedGoType(t *MappedType) bool {
	return t.IsEnumeratedValue || len(t.UnionTypes) >= 2 || t.NativeType == ygot.BinaryTypeName || t.NativeType == ygot.EmptyTypeName
}

// unionType is an internal type used to sort the UnionTypes map field of
// MappedType. It satisfies sort.Interface.
type unionType struct {
	name  string
	index int
}

type unionTypeList []unionType

func (u unionTypeList) Len() int {
	return len(u)
}

func (u unionTypeList) Swap(i, j int) {
	u[i], u[j] = u[j], u[i]
}

func (u unionTypeList) Less(i, j int) bool {
	return u[i].index < u[j].index
}

// OrderedUnionTypes returns a slice of union type names of the given
// MappedType in YANG order. If the type is not a union (i.e. UnionTypes is
// empty), then a nil slice is returned.
func (t *MappedType) OrderedUnionTypes() []string {
	var unionTypes unionTypeList
	for name, index := range t.UnionTypes {
		unionTypes = append(unionTypes, unionType{name: name, index: index})
	}
	sort.Sort(unionTypes)

	var orderedUnionTypes []string
	for _, unionType := range unionTypes {
		orderedUnionTypes = append(orderedUnionTypes, unionType.name)
	}
	return orderedUnionTypes
}

// buildDirectoryDefinitions extracts the yang.Entry instances from a map of
// entries that need struct or message definitions built for them. It resolves
// each non-leaf yang.Entry to a Directory which contains the elements that are
// needed for subsequent code generation. The name of the directory entry that
// is returned is based on the input genDirectoryName function. The type name
// of list keys stored as part of the ListAttr attribute of the returned
// Directories is calculated using the input resolveKeyTypeName function.
// compBehaviour determines how to set the direct children of a Directory,
// including whether those elements within the YANG schema that are marked
// config false (i.e. are read only) are excluded from the returned
// directories.
func buildDirectoryDefinitions(entries map[string]*yang.Entry, compBehaviour genutil.CompressBehaviour,
	genDirectoryName func(*yang.Entry) string, resolveKeyTypeName func(keyleaf *yang.Entry) (*MappedType, error)) (map[string]*Directory, []error) {

	var errs []error
	mappedStructs := make(map[string]*Directory)

	for _, entryKey := range genutil.GetOrderedEntryKeys(entries) {
		e := entries[entryKey]
		// If we are excluding config false (state entries) then skip processing
		// this element.
		if compBehaviour.StateExcluded() && !util.IsConfig(e) {
			continue
		}
		if e.IsList() || e.IsDir() || util.IsRoot(e) {
			// This should be mapped to a struct in the generated code since it has
			// child elements in the YANG schema.
			elem := &Directory{
				Entry: e,
			}

			// Encode the name of the struct according to the input function.
			elem.Name = genDirectoryName(e)

			// Find the elements that should be rooted on this particular entity.
			var fieldErr []error
			elem.Fields, elem.ShadowedFields, fieldErr = genutil.FindAllChildren(e, compBehaviour)
			if fieldErr != nil {
				errs = append(errs, fieldErr...)
				continue
			}

			// Determine the path of the element from the schema.
			elem.Path = strings.Split(util.SchemaTreePath(e), "/")

			// Mark this struct as the fake root if it is specified to be.
			if IsFakeRoot(e) {
				elem.IsFakeRoot = true
			}

			// Handle structures that will represent the container which is duplicated
			// inside a list. This involves extracting the key elements of the list
			// and returning a YangListAttr structure that describes how they should
			// be represented.
			if e.IsList() {
				// Resolve the type name of the key according to the input function.
				lattr, listErr := buildListKey(e, compBehaviour.CompressEnabled(), resolveKeyTypeName)
				if listErr != nil {
					errs = append(errs, listErr...)
					continue
				}
				elem.ListAttr = lattr
			}
			mappedStructs[e.Path()] = elem
		} else {
			errs = append(errs, fmt.Errorf("%s was not an element mapped to a struct", e.Path()))
		}
	}

	return mappedStructs, errs
}

// buildListKey takes a yang.Entry, e, corresponding to a list and extracts the definition
// of the list key, returning a YangListAttr struct describing the key element(s). If
// errors are encountered during the extraction, they are returned as a slice of errors.
// The YangListAttr that is returned consists of a map, keyed by the key leaf's YANG
// identifier, with a value of a MappedType struct generated using resolveKeyTypeName which
// indicates how that key leaf is to be represented in the generated language. The key
// elements themselves are returned in the keyElems slice.
func buildListKey(e *yang.Entry, compressOCPaths bool, resolveKeyTypeName func(keyleaf *yang.Entry) (*MappedType, error)) (*YangListAttr, []error) {
	if !e.IsList() {
		return nil, []error{fmt.Errorf("%s is not a list", e.Name)}
	}

	if e.Key == "" {
		// A null key is not valid if we have a config true list, so return an error
		if util.IsConfig(e) {
			return nil, []error{fmt.Errorf("no key specified for a config true list: %s", e.Name)}
		}
		// This is a keyless list so return an empty YangListAttr but no error, downstream
		// mapping code should consider this to mean that this should be mapped into a
		// keyless structure (i.e., a slice).
		return nil, nil
	}

	listattr := &YangListAttr{
		Keys: make(map[string]*MappedType),
	}

	var errs []error
	keys := strings.Fields(e.Key)
	for _, k := range keys {
		// Extract the key leaf itself from the Dir of the list element. Dir is populated
		// by goyang, and is a map keyed by leaf identifier with values of a *yang.Entry
		// corresponding to the leaf.
		keyleaf, ok := e.Dir[k]
		if !ok {
			return nil, []error{fmt.Errorf("key %s did not exist for %s", k, e.Name)}
		}

		if keyleaf.Type != nil {
			switch keyleaf.Type.Kind {
			case yang.Yleafref:
				// In the case that the key leaf is a YANG leafref, then in OpenConfig
				// this means that the key is a pointer to an element under 'config' or
				// 'state' under the list itself. In the case that this is not an OpenConfig
				// compliant schema, then it may be a leafref to some other element in the
				// schema. Therefore, when the key is a leafref for the OC case, then
				// find the actual leaf that it points to, for other schemas, then ignore
				// this lookup.
				if compressOCPaths {
					// keyleaf.Type.Path specifies the (goyang validated) path to the
					// leaf that is the target of the reference when the keyleaf is a
					// leafref.
					refparts := strings.Split(keyleaf.Type.Path, "/")
					if len(refparts) < 2 {
						return nil, []error{fmt.Errorf("key %s had an invalid path %s", k, keyleaf.Path())}
					}
					// In the case of OpenConfig, the list key is specified to be under
					// the 'config' or 'state' container of the list element (e). To this
					// end, we extract the name of the config/state container. However, in
					// some cases, it can be prefixed, so we need to remove the prefixes
					// from the path.
					dir := util.StripModulePrefix(refparts[len(refparts)-2])
					d, ok := e.Dir[dir]
					if !ok {
						return nil, []error{
							fmt.Errorf("key %s had a leafref key (%s) in dir %s that did not exist (%v)",
								k, keyleaf.Path(), dir, refparts),
						}
					}
					targetLeaf := util.StripModulePrefix(refparts[len(refparts)-1])
					if _, ok := d.Dir[targetLeaf]; !ok {
						return nil, []error{
							fmt.Errorf("key %s had leafref key (%s) that did not exist at (%v)", k, keyleaf.Path(), refparts),
						}
					}
					keyleaf = d.Dir[targetLeaf]
				}
			}
		}

		listattr.KeyElems = append(listattr.KeyElems, keyleaf)
		if resolveKeyTypeName != nil {
			keyType, err := resolveKeyTypeName(keyleaf)
			if err != nil {
				errs = append(errs, err)
			}
			listattr.Keys[keyleaf.Name] = keyType
		}
	}

	return listattr, errs
}
