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

// This file contains type definitions and functions associated with the
// Directory type, which is the basic data element constructed from further
// processing goyang Entry elements that helps in the code generation process.

import (
	"fmt"
	"sort"
	"strings"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/genutil"
	"github.com/openconfig/ygot/util"
)

// Directory stores information needed for outputting a data node of the
// generated code. When viewed as a collection of entries that is generated
// from an entire YANG schema, they serve the purpose of mapping the YANG
// schema tree to a directory tree (connected through implicit yang.Entry
// edges) where each directory corresponds to a data node of the Go version of
// the schema, and where digested data is stored that is friendly to the code
// generation algorithm.
type Directory struct {
	Name       string                 // Name is the name of the struct to be generated.
	Entry      *yang.Entry            // Entry is the yang.Entry that corresponds to the schema element being converted to a struct.
	Fields     map[string]*yang.Entry // Fields is a map, keyed by the YANG node identifier, of the entries that are the struct fields.
	Path       []string               // Path is a slice of strings indicating the element's path.
	ListAttr   *YangListAttr          // ListAttr is used to store characteristics of structs that represent YANG lists.
	IsFakeRoot bool                   // IsFakeRoot indicates that the struct is a fake root struct, so specific mapping rules should be implemented.
}

// isList returns true if the Directory describes a list.
func (y *Directory) isList() bool {
	return y.ListAttr != nil
}

// isChildOfModule determines whether the Directory represents a container
// or list member that is the direct child of a module entry.
func (y *ParsedDirectory) isChildOfModule() bool {
	if y.IsFakeRoot || len(y.Path) == 3 {
		// If the message has a path length of 3, then it is a top-level entity
		// within a module, since the  path is in the format []{"", <module>, <element>}.
		return true
	}
	return false
}

// YangListAttr is used to store the additional elements for a Go struct that
// are required if the struct represents a YANG list. It stores the name of
// the key elements, and their associated types, along with pointers to those
// elements.
type YangListAttr struct {
	// keys is a map, keyed by the name of the key leaf, with values of the type
	// of the key of a YANG list.
	Keys map[string]*MappedType
	// keyElems is a slice containing the pointers to yang.Entry structs that
	// make up the list key.
	KeyElems        []*yang.Entry
	OrderedKeyNames []string
	ListKeyNames    map[string]string
}

// GetOrderedFieldNames returns the field names of a Directory in alphabetical order.
func GetOrderedFieldNames(directory *Directory) []string {
	if directory == nil {
		return nil
	}

	orderedFieldNames := make([]string, 0, len(directory.Fields))
	for fieldName := range directory.Fields {
		orderedFieldNames = append(orderedFieldNames, fieldName)
	}
	sort.Strings(orderedFieldNames)
	return orderedFieldNames
}

func GetOrderedNodeFieldNames(directory *ParsedDirectory) []string {
	if directory == nil {
		return nil
	}

	orderedFieldNames := make([]string, 0, len(directory.Fields))
	for fieldName := range directory.Fields {
		orderedFieldNames = append(orderedFieldNames, fieldName)
	}
	sort.Strings(orderedFieldNames)
	return orderedFieldNames
}

// GoFieldNameMap returns a map containing the Go name for a field (key
// is the field schema name). Camelcase and uniquification is done to ensure
// compilation. Naming uniquification is done deterministically.
func GoFieldNameMap(directory *Directory) map[string]string {
	if directory == nil {
		return nil
	}
	// Order by schema name; then, uniquify in order of schema name.
	orderedFieldNames := GetOrderedFieldNames(directory)

	uniqueGenFieldNames := map[string]bool{}
	uniqueNameMap := make(map[string]string, len(directory.Fields))
	for _, fieldName := range orderedFieldNames {
		uniqueNameMap[fieldName] = genutil.MakeNameUnique(directory.Fields[fieldName].Name, uniqueGenFieldNames)
	}

	return uniqueNameMap
}

func goParsedDirectoryFieldNameMap(d *ParsedDirectory) map[string]string {
	if d == nil {
		return nil
	}
	// Order by schema name; then, uniquify in order of schema name.
	orderedFieldNames := GetOrderedNodeFieldNames(d)

	uniqueGenFieldNames := map[string]bool{}
	uniqueNameMap := make(map[string]string, len(d.Fields))
	for _, fieldName := range orderedFieldNames {
		uniqueNameMap[fieldName] = genutil.MakeNameUnique(d.Fields[fieldName].Name, uniqueGenFieldNames)
	}

	return uniqueNameMap
}

// GetOrderedPathDirectories returns an alphabetically-ordered slice of
// Directory names and a map of Directories keyed by their paths, so that each
// directory can be processed in path-alphabetical order. This helps produce
// deterministic generated code, and minimize diffs when compared with expected
// output (i.e., diffs don't appear simply due to reordering of the Directory
// maps).
func GetOrderedPathDirectories(directory map[string]*Directory) []string {
	orderedDirPaths := make([]string, 0, len(directory))

	for path := range directory {
		orderedDirPaths = append(orderedDirPaths, path)
	}
	sort.Strings(orderedDirPaths)

	return orderedDirPaths
}

// GetOrderedDirectories returns an alphabetically-ordered slice of Directory
// names and a map of Directories keyed by their names instead of their paths,
// so that each directory can be processed in alphabetical order. This helps
// produce deterministic generated code, and minimize diffs when compared with
// expected output (i.e., diffs don't appear simply due to reordering of the
// Directory maps). If the names of the directories are not unique, which is
// unexpected, an error is returned.
func GetOrderedDirectories(directory map[string]*Directory) ([]string, map[string]*Directory, error) {
	if directory == nil {
		return nil, nil, fmt.Errorf("directory map null")
	}
	orderedDirNames := make([]string, 0, len(directory))
	dirNameMap := make(map[string]*Directory, len(directory))

	for _, dir := range directory {
		orderedDirNames = append(orderedDirNames, dir.Name)
		dirNameMap[dir.Name] = dir
	}
	if len(dirNameMap) != len(directory) {
		return nil, nil, fmt.Errorf("directory name conflict(s) exist")
	}
	sort.Strings(orderedDirNames)

	return orderedDirNames, dirNameMap, nil
}

func getOrderedDirDetails(langMapper LangMapper, directory map[string]*Directory, compBehaviour genutil.CompressBehaviour, absolutePaths, nestedStructs bool) ([]string, map[string]*ParsedDirectory, error) {
	orderedPaths := []string{}
	dirDets := map[string]*ParsedDirectory{}
	for _, dirPath := range GetOrderedPathDirectories(directory) {
		dir := directory[dirPath]
		dirDets[dir.Entry.Path()] = parseDir(dir)
		packageName, err := langMapper.PackageName(dir.Entry, compBehaviour, nestedStructs)
		if err != nil {
			return nil, nil, err
		}
		dirDets[dir.Entry.Path()].PackageName = packageName
		dirDets[dir.Entry.Path()].Type = DirectoryNode
		if dir.Entry.IsList() {
			dirDets[dir.Entry.Path()].Type = ListNode
		}

		dirDets[dir.Entry.Path()].Fields = make(map[string]*NodeDetails, len(dir.Fields))
		for _, fn := range GetOrderedFieldNames(dir) {
			field := dir.Fields[fn]

			mp, err := findMapPaths(dir, fn, compBehaviour.CompressEnabled(), absolutePaths)
			if err != nil {
				return nil, nil, err
			}

			mod, err := field.InstantiatingModule()
			if err != nil {
				return nil, nil, err
			}

			target, err := langMapper.ResolveLeafref(field)
			if err != nil {
				return nil, nil, err
			}

			nd := &NodeDetails{
				YANGDetails: YANGNodeDetails{
					Default:         field.DefaultValue(), // FIXME(wenbli): Test this
					Path:            strings.Split(field.Path(), "/"),
					PathStr:         field.Path(),
					ResolvedPathStr: target.Path(),
					Module:          mod,
					Name:            field.Name,
				},
				MapPaths: mp,
				Type:     DirectoryNode,
			}

			name, err := langMapper.LeafName(field)
			if err != nil {
				return nil, nil, err
			}
			nd.Name = name

			// FIXME(wenbli): This should be a switch statement instead of if statements that could overwrite each other.
			if isLeaf := field.IsLeaf() || field.IsLeafList(); isLeaf {
				mtype, err := langMapper.LeafType(field, compBehaviour)
				if err != nil {
					return nil, nil, err
				}
				t := LeafNode
				if field.IsLeafList() {
					t = LeafListNode
				}

				nd.Type = t
				nd.LangType = mtype
			}

			if field.IsList() {
				nd.Type = ListNode
			}

			if util.IsAnydata(field) {
				nd.Type = AnyDataNode
			}

			dirDets[dir.Entry.Path()].Fields[fn] = nd
		}
		orderedPaths = append(orderedPaths, dir.Entry.Path())
	}

	return orderedPaths, dirDets, nil

}

// FindSchemaPath finds the relative or absolute schema path of a given field
// of a Directory. The Field is specified as a name in order to guarantee its
// existence before processing.
func FindSchemaPath(parent *Directory, fieldName string, absolutePaths bool) ([]string, error) {
	field, ok := parent.Fields[fieldName]
	if !ok {
		return nil, fmt.Errorf("FindSchemaPath: field name %q does not exist in Directory %s", fieldName, parent.Path)
	}
	fieldSlicePath := util.SchemaPathNoChoiceCase(field)

	if absolutePaths {
		return append([]string{""}, fieldSlicePath[1:]...), nil
	}
	// Return the elements that are not common between the two paths.
	// Since the field is necessarily a child of the parent, then to
	// determine those elements of the field's path that are not contained
	// in the parent's, we walk from index X of the field's path (where X
	// is the number of elements in the path of the parent).
	if len(fieldSlicePath) < len(parent.Path) {
		return nil, fmt.Errorf("FindSchemaPath: field %v is not a valid child of %v", fieldSlicePath, parent.Path)
	}
	return fieldSlicePath[len(parent.Path)-1:], nil
}
