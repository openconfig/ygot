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
	"reflect"
	"sort"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/genutil"
	"github.com/openconfig/ygot/util"
	"github.com/openconfig/ygot/yangschema"
	"github.com/openconfig/ygot/ygot"
)

// Directory stores information needed for outputting a data node of the
// generated code. When viewed as a collection of entries that is generated
// from an entire YANG schema, they serve the purpose of mapping the YANG
// schema tree to a directory tree (connected through implicit yang.Entry
// edges) where each directory corresponds to a data node of the Go version of
// the schema, and where digested data is stored that is friendly to the code
// generation algorithm.
type Directory struct {
	Name           string                 // Name is the name of the struct to be generated.
	Entry          *yang.Entry            // Entry is the yang.Entry that corresponds to the schema element being converted to a struct.
	Fields         map[string]*yang.Entry // Fields is a map, keyed by the YANG node identifier, of the entries that are the struct fields.
	ShadowedFields map[string]*yang.Entry // ShadowedFields is a map, keyed by the YANG node identifier, of the field entries duplicated via compression.
	Path           []string               // Path is a slice of strings indicating the element's path.
	ListAttr       *YangListAttr          // ListAttr is used to store characteristics of structs that represent YANG lists.
	IsFakeRoot     bool                   // IsFakeRoot indicates that the struct is a fake root struct, so specific mapping rules should be implemented.
}

// YangListAttr is used to store the additional elements for a Go struct that
// are required if the struct represents a YANG list. It stores the name of
// the key elements, and their associated types, along with pointers to those
// elements.
type YangListAttr struct {
	// keys is a map, keyed by the name of the key leaf, with values of the type
	// of the key of a YANG list.
	Keys map[string]*ListKey
	// keyElems is a slice containing the pointers to yang.Entry structs that
	// make up the list key.
	KeyElems []*yang.Entry
	// ListKeyYANGNames is the ordered list of YANG names specified in the
	// YANG list per Section 7.8.2 of RFC6020. ygot relies on this ordering
	// for deterministic ordering in output code and rendering.
	ListKeyYANGNames []string
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

// GoFieldNameMap returns a map containing the Go name for a field (key
// is the field schema name). Name Uniquification is done to ensure
// compilation -- this is done deterministically.
func GoFieldNameMap(directory *ParsedDirectory) map[string]string {
	if directory == nil {
		return nil
	}
	uniqueGenFieldNames := make(map[string]bool, len(directory.Fields))
	uniqueNameMap := make(map[string]string, len(directory.Fields))
	// Alphabetically order fields to produce deterministic output.
	for _, fName := range directory.OrderedFieldNames() {
		uniqueNameMap[fName] = genutil.MakeNameUnique(directory.Fields[fName].Name, uniqueGenFieldNames)
	}
	return uniqueNameMap
}

// GetOrderedDirectories returns an alphabetically-ordered slice of Directory
// names and a map of Directories keyed by their names instead of their paths,
// so that each directory can be processed in alphabetical order. This helps
// produce deterministic generated code, and minimize diffs when compared with
// expected output (i.e., diffs don't appear simply due to reordering of the
// Directory maps). If the names of the directories are not unique, which is
// unexpected, an error is returned.
// TODO(wenbli): Deprecate this after ygot uses the IR for code generation.
// This function's purpose is to check for name conflicts. This functionality
// doesn't belong during IR processing but rather with downstream
// language-specific processing since it's possible that conflicts are allowed
// (e.g. nested struct definitions).
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

// getOrderedDirDetails takes in a language-specific LangMapper, a map of
// Directory objects containing the raw AST information, a SchemaTree, and IR
// generation options, and returns a map of ParsedDirectory objects that form
// the primary component of ygen's IR output.
func getOrderedDirDetails(langMapper LangMapper, directory map[string]*Directory, schematree *yangschema.Tree, opts IROptions) (map[string]*ParsedDirectory, error) {
	dirDets := map[string]*ParsedDirectory{}
	for _, dirPath := range GetOrderedPathDirectories(directory) {
		dir := directory[dirPath]
		packageName, err := langMapper.PackageName(dir.Entry, opts.TransformationOptions.CompressBehaviour, opts.NestedDirectories)
		if err != nil {
			return nil, err
		}

		var belongingModule string
		var rootModule string
		if !dir.IsFakeRoot {
			var err error
			if belongingModule, err = dir.Entry.InstantiatingModule(); err != nil {
				return nil, fmt.Errorf("ygen: cannot find instantiating module for Directory %s: %v", dir.Path, err)
			}
			rootModule = util.TopLevelModule(dir.Entry).Name
		}
		var definingModuleName string
		if definingModule := yang.RootNode(dir.Entry.Node); definingModule != nil {
			definingModuleName = definingModule.Name
		}

		pd := &ParsedDirectory{
			Name:              dir.Name,
			Path:              util.SlicePathToString(dir.Path),
			SchemaPath:        util.SchemaTreePathNoModule(dir.Entry),
			PackageName:       packageName,
			IsFakeRoot:        dir.IsFakeRoot,
			BelongingModule:   belongingModule,
			DefiningModule:    definingModuleName,
			RootElementModule: rootModule,
			ConfigFalse:       !util.IsConfig(dir.Entry),
		}
		switch {
		case dir.Entry.IsList():
			pd.Type = List
			if dir.Entry.ListAttr.OrderedByUser {
				pd.Type = OrderedList
			}
			if !util.IsUnkeyedList(dir.Entry) {
				pd.ListKeys = dir.ListAttr.Keys
				pd.ListKeyYANGNames = dir.ListAttr.ListKeyYANGNames
			}
		default:
			pd.Type = Container
		}

		for i, entry := 0, dir.Entry; ; i++ {
			exts, err := yang.MatchingEntryExtensions(entry, "openconfig-extensions", "telemetry-atomic")
			if err != nil {
				return nil, fmt.Errorf("cannot retrieve OpenConfig extensions: %v", err)
			}
			if len(exts) > 0 {
				if i == 0 {
					pd.TelemetryAtomic = true
				} else {
					pd.CompressedTelemetryAtomic = true
				}
			}

			if entry.Parent == nil {
				// The very first element is the module -- skip it.
				break
			}
			entry = entry.Parent
			if _, ok := directory[entry.Path()]; ok {
				break
			}
		}

		pd.Fields = make(map[string]*NodeDetails, len(dir.Fields))
		for _, fn := range GetOrderedFieldNames(dir) {
			field := dir.Fields[fn]
			shadowField, hasShadowField := dir.ShadowedFields[fn]

			mp, mm, err := findMapPaths(dir, fn, opts.TransformationOptions.CompressBehaviour.CompressEnabled(), false, opts.AbsoluteMapPaths)
			if err != nil {
				return nil, err
			}

			smp, smm, err := findMapPaths(dir, fn, opts.TransformationOptions.CompressBehaviour.CompressEnabled(), true, opts.AbsoluteMapPaths)
			if err != nil {
				return nil, err
			}

			mod, err := field.InstantiatingModule()
			if err != nil {
				return nil, err
			}

			var target *yang.Entry
			if field.Type != nil && field.Type.Kind == yang.Yleafref {
				if target, err = schematree.ResolveLeafrefTarget(field.Type.Path, field); err != nil {
					return nil, fmt.Errorf("unable to resolve leafref field: %v", err)
				}
			}

			name, err := langMapper.FieldName(field)
			if err != nil {
				return nil, err
			}

			var definingModuleName string
			if definingModule := yang.RootNode(field.Node); definingModule != nil {
				definingModuleName = definingModule.Name
			}
			nd := &NodeDetails{
				Name: name,
				YANGDetails: YANGNodeDetails{
					Name:              field.Name,
					Defaults:          field.DefaultValues(),
					BelongingModule:   mod,
					RootElementModule: util.TopLevelModule(field).Name,
					DefiningModule:    definingModuleName,
					Path:              field.Path(),
					SchemaPath:        util.SchemaTreePathNoModule(field),
					LeafrefTargetPath: target.Path(),
					Description:       field.Description,
					ConfigFalse:       !util.IsConfig(field),
				},
				MappedPaths:             mp,
				MappedPathModules:       mm,
				ShadowMappedPaths:       smp,
				ShadowMappedPathModules: smm,
			}
			if hasShadowField {
				nd.YANGDetails.ShadowSchemaPath = util.SchemaTreePathNoModule(shadowField)
			}

			switch {
			case field.IsLeaf(), field.IsLeafList():
				mtype, err := langMapper.LeafType(field, opts)
				if err != nil {
					return nil, err
				}
				t := LeafNode
				if field.IsLeafList() {
					t = LeafListNode
					nd.YANGDetails.OrderedByUser = field.ListAttr.OrderedByUser
				}

				nd.Type = t
				nd.LangType = mtype
			case field.IsList():
				nd.Type = ListNode
				nd.YANGDetails.OrderedByUser = field.ListAttr.OrderedByUser
			case util.IsAnydata(field):
				nd.Type = AnyDataNode
			default:
				nd.Type = ContainerNode
				// TODO(wenovus):
				// a presence container is an unimplemented keyword in goyang.
				// if and when this changes, the field lookup below would need to change as well.
				if len(field.Extra["presence"]) > 0 {
					if v := field.Extra["presence"][0].(*yang.Value); v != nil {
						nd.YANGDetails.PresenceStatement = ygot.String(v.Name)
					} else {
						return nil, fmt.Errorf("unable to retrieve presence statement, expected non-nil *yang.Value, got %v", dir.Entry.Extra["presence"][0])
					}
				}
			}

			nd.Flags = langMapper.PopulateFieldFlags(*nd, field)

			pd.Fields[fn] = nd
		}
		dirDets[dir.Entry.Path()] = pd
	}

	return dirDets, nil
}

// FindSchemaPath finds the relative or absolute schema path of a given field
// of a Directory. The Field is specified as a name in order to guarantee its
// existence before processing.
func FindSchemaPath(parent *Directory, fieldName string, absolutePaths bool) ([]string, error) {
	schemaPaths, _, err := findSchemaPath(parent, fieldName, false, absolutePaths)
	return schemaPaths, err
}

// FindShadowSchemaPath finds the relative or absolute schema path of a given field
// of a Directory with preference to the shadow path. The Field is specified as a name
// in order to guarantee its  existence before processing.
// NOTE: No error is returned if fieldName is not found.
func FindShadowSchemaPath(parent *Directory, fieldName string, absolutePaths bool) ([]string, error) {
	schemaPaths, _, err := findSchemaPath(parent, fieldName, true, absolutePaths)
	return schemaPaths, err
}

// findSchemaPath finds the relative or absolute schema path of a given field
// of a Directory, or the shadowed field path (i.e. field duplicated and
// deprioritized via compression) of a Directory. The first returned slice
// contains the names of the path elements, and the second contains the
// corresponding module names for each path element's resident namespace. The
// Field is specified as a name in order to guarantee its existence before
// processing.
// NOTE: If shadowSchemaPaths is true, no error is returned if fieldName is not found.
func findSchemaPath(parent *Directory, fieldName string, shadowSchemaPaths, absolutePaths bool) ([]string, []string, error) {
	field, ok := parent.Fields[fieldName]
	if shadowSchemaPaths {
		if field, ok = parent.ShadowedFields[fieldName]; !ok {
			return nil, nil, nil
		}
	}
	if !ok {
		return nil, nil, fmt.Errorf("FindSchemaPath(shadowSchemaPaths:%v): field name %q does not exist in Directory %s", shadowSchemaPaths, fieldName, parent.Path)
	}
	fieldSlicePath := util.SchemaPathNoChoiceCase(field)
	var fieldSliceModules []string
	for _, e := range util.SchemaEntryPathNoChoiceCase(field) {
		im, err := e.InstantiatingModule()
		if err != nil {
			return nil, nil, fmt.Errorf("FindSchemaPath(shadowSchemaPaths:%v): cannot find instantiating module for field %q in Directory %s: %v", shadowSchemaPaths, fieldName, parent.Path, err)
		}
		fieldSliceModules = append(fieldSliceModules, im)
	}

	if absolutePaths {
		return append([]string{""}, fieldSlicePath[1:]...), append([]string{""}, fieldSliceModules[1:]...), nil
	}
	// Return the elements that are not common between the two paths.
	// Since the field is necessarily a child of the parent, then to
	// determine those elements of the field's path that are not contained
	// in the parent's, we walk from index X of the field's path (where X
	// is the number of elements in the path of the parent).
	if len(fieldSlicePath) < len(parent.Path) {
		return nil, nil, fmt.Errorf("FindSchemaPath(shadowSchemaPaths:%v): field %v is not a valid child of %v", shadowSchemaPaths, fieldSlicePath, parent.Path)
	}
	return fieldSlicePath[len(parent.Path)-1:], fieldSliceModules[len(parent.Path)-1:], nil
}

// findMapPaths takes an input field name for a parent Directory and calculates
// the set of schema paths it represents, as well as the corresponding module
// names for each schema path element's resident namespace.
// If absolutePaths is set, the paths are absolute; otherwise, they are relative to the parent. If
// the input entry is a key to a list, and is of type leafref, then the corresponding target leaf's
// path is also returned. If shadowSchemaPaths is set, then the path of the
// field deprioritized via compression is returned instead of the prioritized paths.
// The first returned path is the path of the direct child, followed by the key
// leafref path, if any.
func findMapPaths(parent *Directory, fieldName string, compressPaths, shadowSchemaPaths, absolutePaths bool) ([][]string, [][]string, error) {
	childPath, childModulePath, err := findSchemaPath(parent, fieldName, shadowSchemaPaths, absolutePaths)
	if err != nil {
		return nil, nil, err
	}
	var mapPaths, mapModulePaths [][]string
	if childPath != nil {
		mapPaths = append(mapPaths, childPath)
	}
	if childModulePath != nil {
		mapModulePaths = append(mapModulePaths, childModulePath)
	}
	// Only for compressed data schema paths for list fields do we have the
	// possibility for a direct leafref path as a second path for the field.
	if !compressPaths || parent.ListAttr == nil {
		return mapPaths, mapModulePaths, nil
	}

	field, ok := parent.Fields[fieldName]
	if !ok {
		return nil, nil, fmt.Errorf("field name %s does not exist in Directory %s", fieldName, parent.Path)
	}
	fieldSlicePath := util.SchemaPathNoChoiceCase(field)
	var fieldSliceModules []string
	for _, e := range util.SchemaEntryPathNoChoiceCase(field) {
		im, err := e.InstantiatingModule()
		if err != nil {
			return nil, nil, fmt.Errorf("FindSchemaPath(shadowSchemaPaths:%v): cannot find instantiating module for field %q in Directory %s: %v", shadowSchemaPaths, fieldName, parent.Path, err)
		}
		fieldSliceModules = append(fieldSliceModules, im)
	}

	// Handle specific issue of compressed path schemas, where a key of the
	// parent list is a leafref to this leaf.
	for _, k := range parent.ListAttr.KeyElems {
		// If the key element has the same path as this element, and the
		// corresponding element that is within the parent's container is of
		// type leafref, then within an OpenConfig schema this means that
		// the key leaf was a pointer to this leaf. To this end, we set
		// isKey to true so that the struct field can be mapped to the
		// leafref leaf within the schema as well as the target of the
		// leafref.
		if k.Parent == nil || k.Parent.Parent == nil || k.Parent.Parent.Dir[k.Name] == nil || k.Parent.Parent.Dir[k.Name].Type == nil {
			return nil, nil, fmt.Errorf("invalid compressed schema, could not find the key %s or the grandparent of %s", k.Name, k.Path())
		}

		// If a key of the list is a leafref that points to the field,
		// then add this as an alternative path.
		// Note: if k is a leafref, buildListKey() would have already
		// resolved it the field that the leafref points to. So, we
		// compare their absolute paths for equality.
		if k.Parent.Parent.Dir[k.Name].Type.Kind == yang.Yleafref && reflect.DeepEqual(util.SchemaPathNoChoiceCase(k), fieldSlicePath) {
			// The path of the key element is simply the name of the leaf under the
			// list, since the YANG specification enforces that keys are direct
			// children of the list.
			keyPath := []string{fieldSlicePath[len(fieldSlicePath)-1]}
			keyModulePath := []string{fieldSliceModules[len(fieldSliceModules)-1]}
			if absolutePaths {
				// If absolute paths are required, then the 'config' or 'state' container needs to be omitted from
				// the complete path for the secondary mapping.
				keyPath = append([]string{""}, fieldSlicePath[1:len(fieldSlicePath)-2]...)
				keyPath = append(keyPath, fieldSlicePath[len(fieldSlicePath)-1])
				keyModulePath = append([]string{""}, fieldSliceModules[1:len(fieldSliceModules)-2]...)
				keyModulePath = append(keyModulePath, fieldSliceModules[len(fieldSliceModules)-1])
			}
			mapPaths = append(mapPaths, keyPath)
			mapModulePaths = append(mapModulePaths, keyModulePath)
			break
		}
	}
	return mapPaths, mapModulePaths, nil
}
