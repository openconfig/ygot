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

// Package ypathgen contains a library to generate gNMI paths from a YANG model.
// The ygen library is used to parse YANG and obtain intermediate and some final
// information. The output always assumes the OpenConfig-specific conventions
// for a compressed schema.
//
// TODO(wenbli): This package is written with only compressed schemas in mind.
// If needed, can write tests, verify, and enhance it to support uncompressed
// ygen structs.
package ypathgen

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"text/template"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/genutil"
	"github.com/openconfig/ygot/util"
	"github.com/openconfig/ygot/ygen"
	"github.com/openconfig/ygot/ygot"
)

// Static default configuration values that differ from the zero value for their types.
const (
	// defaultSchemaStructPkgAlias is the package alias for the imported ygen-generated file.
	defaultSchemaStructPkgAlias = "oc"
	// defaultPathPackageName specifies the default name that should be
	// used for the generated Go package.
	defaultPathPackageName = "ocpathstructs"
	// defaultFakeRootName is the default name for the root structure.
	defaultFakeRootName = "device"
	// WildcardSuffix is the suffix given to the wildcard versions of each
	// node as well as a list's wildcard child constructor methods that
	// distinguishes each from its non-wildcard counterpart.
	WildcardSuffix = "Any"
)

// NewDefaultConfig creates a GenConfig with default configuration. schemaStructPkgPath is a
// required configuration parameter.
func NewDefaultConfig(schemaStructPkgPath string) *GenConfig {
	return &GenConfig{
		PackageName: defaultPathPackageName,
		GoImports: GoImports{
			SchemaStructPkgPath: schemaStructPkgPath,
			GNMIProtoPath:       genutil.GoDefaultGNMIImportPath,
			YgotImportPath:      genutil.GoDefaultYgotImportPath,
		},
		FakeRootName:         defaultFakeRootName,
		SchemaStructPkgAlias: defaultSchemaStructPkgAlias,
		GeneratingBinary:     genutil.CallerName(),
	}
}

// GenConfig stores code generation configuration.
type GenConfig struct {
	// PackageName is the name that should be used for the generating package.
	PackageName string
	// GoImports contains package import options.
	GoImports GoImports
	// FakeRootName specifies the name of the struct that should be generated
	// representing the root.
	FakeRootName string
	// ExcludeModules specifies any modules that are included within the set of
	// modules that should have code generated for them that should be ignored during
	// code generation. This is due to the fact that some schemas (e.g., OpenConfig
	// interfaces) currently result in overlapping entities (e.g., /interfaces).
	ExcludeModules []string
	// SchemaStructPkgAlias is the package alias of the schema struct package.
	SchemaStructPkgAlias string
	// YANGParseOptions provides the options that should be handed to the
	// github.com/openconfig/goyang/pkg/yang library. These specify how the
	// input YANG files should be parsed.
	YANGParseOptions yang.Options
	// GeneratingBinary is the name of the binary calling the generator library, it is
	// included in the header of output files for debugging purposes. If a
	// string is not specified, the location of the library is utilised.
	GeneratingBinary string
}

// GoImports contains package import options.
type GoImports struct {
	// SchemaStructPkgPath specifies the path to the ygen-generated structs, which
	// is used to get the enum and union type names used as the list key
	// for calling a list path accessor.
	SchemaStructPkgPath string
	// GNMIProtoPath specifies the path to the generated gNMI protobuf, which
	// is used to store the catalogue entries for generated modules.
	GNMIProtoPath string
	// YgotImportPath specifies the path to the ygot library that should be used
	// in the generated code.
	YgotImportPath string
}

// GeneratePathCode takes a slice of strings containing the path to a set of YANG
// files which contain YANG modules, and a second slice of strings which
// specifies the set of paths that are to be searched for associated models (e.g.,
// modules that are included by the specified set of modules, or submodules of those
// modules). It extracts the set of modules that are to be generated, and returns
// a pointer to a GeneratedPathCode struct containing all the generated code to
// support the path-creation API. The important components of the generated
// code are listed below:
//	1. Struct definitions for each container, list, or leaf schema node,
//	as well as the fakeroot.
//	2. A Resolve() helper function, which can return the absolute path of
//	any struct.
//	3. Next-level methods for the fakeroot and each non-leaf schema node,
//	which instantiate and return the next-level structs corresponding to
//	its child schema nodes.
// With these components, the generated API is able to support absolute path
// creation of any node of the input schema.
// Also returned is the NodeDataMap of the schema, i.e. information about each
// node in the generated code, which may help callers add customized
// augmentations to the basic generated path code.
// If errors are encountered during code generation, they are returned.
func (cg *GenConfig) GeneratePathCode(yangFiles, includePaths []string) (*GeneratedPathCode, NodeDataMap, util.Errors) {
	if cg.GoImports.SchemaStructPkgPath == "" {
		return nil, nil, util.NewErrs(fmt.Errorf("GeneratePathCode: Must specify SchemaStructPkgPath"))
	}

	dcg := &ygen.DirectoryGenConfig{
		ParseOptions: ygen.ParseOpts{
			YANGParseOptions: cg.YANGParseOptions,
			ExcludeModules:   cg.ExcludeModules,
		},
		TransformationOptions: ygen.TransformationOpts{
			CompressBehaviour: genutil.PreferOperationalState,
			GenerateFakeRoot:  true,
		},
	}
	directories, leafTypeMap, errs := dcg.GetDirectoriesAndLeafTypes(yangFiles, includePaths)
	if errs != nil {
		return nil, nil, errs
	}

	genCode := &GeneratedPathCode{}
	errs = util.Errors{}
	if err := writeHeader(yangFiles, includePaths, cg, genCode); err != nil {
		return nil, nil, util.AppendErr(errs, err)
	}

	// Alphabetically order directories to produce deterministic output.
	orderedDirNames, dirNameMap, err := ygen.GetOrderedDirectories(directories)
	if err != nil {
		return nil, nil, util.AppendErr(errs, err)
	}

	// Generate struct code.
	var structSnippets []GoPathStructCodeSnippet
	for _, directoryName := range orderedDirNames {
		directory, ok := dirNameMap[directoryName]
		if !ok {
			return nil, nil, util.AppendErr(errs,
				util.NewErrs(fmt.Errorf("GeneratePathCode: Implementation bug -- node %s not found in dirNameMap", directoryName)))
		}

		structSnippet, es := generateDirectorySnippet(directory, directories, cg.SchemaStructPkgAlias)
		if es != nil {
			errs = util.AppendErrs(errs, es)
		}
		structSnippets = append(structSnippets, structSnippet)
	}
	genCode.Structs = structSnippets

	// Get NodeDataMap for the schema.
	nodeDataMap, es := getNodeDataMap(directories, leafTypeMap, cg.SchemaStructPkgAlias)
	if es != nil {
		util.AppendErrs(errs, es)
	}

	if len(errs) == 0 {
		errs = nil
	}
	return genCode, nodeDataMap, errs
}

// GeneratedPathCode contains generated code snippets that can be processed by the calling
// application. The generated code is divided into two types of objects - both represented
// as a slice of strings: Structs contains a set of Go structures that have been generated,
// and Enums contains the code for generated enumerated types (corresponding to identities,
// or enumerated values within the YANG models for which code is being generated). Additionally
// the header with package comment of the generated code is returned in Header, along with the
// a slice of strings containing the packages that are required for the generated Go code to
// be compiled is returned.
//
// For schemas that contain enumerated types (identities, or enumerations), a code snippet is
// returned as the EnumMap field that allows the string values from the YANG schema to be resolved.
// The keys of the map are strings corresponding to the name of the generated type, with the
// map values being maps of the int64 identifier for each value of the enumeration to the name of
// the element, as used in the YANG schema.
type GeneratedPathCode struct {
	Structs      []GoPathStructCodeSnippet // Structs is the generated set of structs representing containers or lists in the input YANG models.
	CommonHeader string                    // CommonHeader is the header that should be used for all output Go files.
	OneOffHeader string                    // OneOffHeader defines the header that should be included in only one output Go file - such as package init statements.
}

// String method for GeneratedPathCode, which can be used to write all the
// generated code into a single file.
func (genCode GeneratedPathCode) String() string {
	var gotCode strings.Builder
	gotCode.WriteString(genCode.CommonHeader)
	gotCode.WriteString(genCode.OneOffHeader)
	for _, gotStruct := range genCode.Structs {
		gotCode.WriteString(gotStruct.String())
	}
	return gotCode.String()
}

// SplitFiles returns a slice of strings, each representing a file that
// together contains the entire generated code. fileN specifies the number of
// files to split the code into, and has to be between 1 and the total number
// of directory entries in the input schema. By splitting, the size of the
// output files can be roughly controlled.
func (genCode GeneratedPathCode) SplitFiles(fileN int) ([]string, error) {
	structN := len(genCode.Structs)
	if fileN < 1 || fileN > structN {
		return nil, fmt.Errorf("requested %d files, but must be between 1 and %d (number of structs)", fileN, structN)
	}

	files := make([]string, 0, fileN)
	structsPerFile := structN / fileN
	var gotCode strings.Builder
	gotCode.WriteString(genCode.CommonHeader)
	gotCode.WriteString(genCode.OneOffHeader)

	for i, gotStruct := range genCode.Structs {
		// The last file contains the remainder of the structs.
		if i%structsPerFile == 0 && i/structsPerFile > 0 && i/structsPerFile < fileN {
			files = append(files, gotCode.String())
			gotCode.Reset()
			gotCode.WriteString(genCode.CommonHeader)
		}
		gotCode.WriteString(gotStruct.String())
	}
	files = append(files, gotCode.String())

	return files, nil
}

// GoPathStructCodeSnippet is used to store the generated code snippets associated with
// a particular Go struct entity (corresponding to a container, list, or leaf in the schema).
type GoPathStructCodeSnippet struct {
	// PathStructName is the name of the struct that is contained within the snippet.
	// It is stored such that callers can identify the struct to control where it
	// is output.
	PathStructName string
	// StructBase stores the basic code snippet that represents the struct that is
	// the input when code generation is performed, which includes its definition.
	StructBase string
	// ChildConstructors contains the method code snippets with the input struct as a
	// receiver, that is used to get the child path struct.
	ChildConstructors string
}

// String returns the contents of a GoPathStructCodeSnippet as a string by
// simply writing out all of its generated code.
func (g GoPathStructCodeSnippet) String() string {
	var b bytes.Buffer
	for _, method := range []string{g.StructBase, g.ChildConstructors} {
		genutil.WriteIfNotEmpty(&b, method)
	}
	return b.String()
}

// NodeDataMap is a map from the path struct type name of a schema node to its NodeData.
type NodeDataMap map[string]*NodeData

// NodeData contains information about the ygen-generated code of a YANG schema node.
type NodeData struct {
	// GoTypeName is the ygen type name of the node, which is qualified by
	// the SchemaStructPkgAlias if necessary.
	GoTypeName string
	// GoFieldName is the field name of the node under its parent struct.
	GoFieldName string
	// ParentGoTypeName is the parent struct's type name.
	ParentGoTypeName string
	// IsLeaf indicates whether this child is a leaf node.
	IsLeaf bool
	// IsScalarField indicates a leaf that is stored as a pointer in its
	// parent struct.
	IsScalarField bool
	// YANGTypeName is the type of the leaf given in the YANG file (without
	// the module prefix, if any, per goyang behaviour). If the node is not
	// a leaf this will be empty. Note that the current purpose for this is
	// to allow callers to handle certain types as special cases, but since
	// the name of the node is a very basic piece of information which
	// excludes the defining module, this is somewhat hacky, so it may be
	// removed or modified in the future.
	YANGTypeName string
}

// GetOrderedNodeDataNames returns the alphabetically-sorted slice of keys
// (path struct names) for a given NodeDataMap.
func GetOrderedNodeDataNames(nodeDataMap NodeDataMap) []string {
	nodeDataNames := make([]string, 0, len(nodeDataMap))
	for name := range nodeDataMap {
		nodeDataNames = append(nodeDataNames, name)
	}
	sort.Slice(nodeDataNames, func(i, j int) bool {
		return nodeDataNames[i] < nodeDataNames[j]
	})
	return nodeDataNames
}

var (
	// goPathCommonHeaderTemplate is populated and output at the top of the generated code package
	goPathCommonHeaderTemplate = `
{{- /**/ -}}
/*
Package {{ .PackageName }} is a generated package which contains definitions
of structs which generate gNMI paths for a YANG schema. The generated paths are
based on a compressed form of the schema.

This package was generated by {{ .GeneratingBinary }}
using the following YANG input files:
{{- range $inputFile := .YANGFiles }}
	- {{ $inputFile }}
{{- end }}
Imported modules were sourced from:
{{- range $importPath := .IncludePaths }}
	- {{ $importPath }}
{{- end }}
*/
package {{ .PackageName }}

import (
	"fmt"

	gpb "{{ .GNMIProtoPath }}"
	{{ .SchemaStructPkgAlias }} "{{ .SchemaStructPkgPath }}"
	"{{ .YgotImportPath }}"
)
`

	// goPathOneOffHeaderTemplate defines the template for package code that should
	// be output in only one file.
	goPathOneOffHeaderTemplate = `
// Resolve is a helper which returns the resolved *gpb.Path of a PathStruct node.
func Resolve(n ygot.{{ .PathStructInterfaceName }}) (*gpb.Path, []error) {
	n, p, errs := ygot.ResolvePath(n)
	root, ok := n.(*{{ .FakeRootTypeName }})
	if !ok {
		errs = append(errs, fmt.Errorf("Resolve(n ygot.{{ .PathStructInterfaceName }}): got unexpected root of (type, value) (%T, %v)", n, n))
	}

	if errs != nil {
		return nil, errs
	}
	return &gpb.Path{Target: root.id, Elem: p}, nil
}
`

	// goFakerootTemplate defines a template for the type definition and
	// basic methods of the fakeroot object. The fakeroot object adheres to
	// the methods of PathStructInterfaceName in order to allow its path
	// struct descendents to use the Resolve() helper function for
	// obtaining their absolute paths.
	goFakeRootTemplate = `
// {{ .TypeName }} represents the {{ .YANGPath }} YANG schema element.
type {{ .TypeName }} struct {
	ygot.{{ .PathBaseTypeName }}
	id string
}

func For{{ .TypeName }}(id string) *{{ .TypeName }} {
	return &{{ .TypeName }}{id: id}
}
`

	// goPathStructTemplate defines the template for the type definition of
	// a path node as well as its core method(s). A path struct/node is
	// either a container, list, or a leaf node in the openconfig schema
	// where the tree formed by the nodes mirrors the compressed YANG
	// schema tree. The defined type stores the relative path to the
	// current node, as well as its parent node for obtaining its absolute
	// path. There are two versions of these, non-wildcard and wildcard.
	// The wildcard version is simply a type to indicate that the path it
	// holds contains a wildcard, but is otherwise the exact same.
	goPathStructTemplate = `
// {{ .TypeName }} represents the {{ .YANGPath }} YANG schema element.
type {{ .TypeName }} struct {
	ygot.{{ .PathBaseTypeName }}
}

// {{ .TypeName }}{{ .WildcardSuffix }} represents the wildcard version of the {{ .YANGPath }} YANG schema element.
type {{ .TypeName }}{{ .WildcardSuffix }} struct {
	ygot.{{ .PathBaseTypeName }}
}
`

	// goChildConstructorTemplate generates the child constructor method
	// for a generated struct by returning an instantiation of the child's
	// path struct object.
	goChildConstructorTemplate = `
// {{ .MethodName }} returns from {{ .Struct.TypeName }} the path struct for its child "{{ .SchemaName }}".
func (n *{{ .Struct.TypeName }}) {{ .MethodName -}} ({{ .KeyParamListStr }}) *{{ .TypeName }} {
	return &{{ .TypeName }}{
		{{ .Struct.PathBaseTypeName }}: ygot.New{{ .Struct.PathBaseTypeName }}(
			[]string{ {{- .RelPathList -}} },
			map[string]interface{}{ {{- .KeyEntriesStr -}} },
			n,
		),
	}
}
`

	// The set of built templates that are to be referenced during code generation.
	goPathTemplates = map[string]*template.Template{
		"commonHeader":     makePathTemplate("commonHeader", goPathCommonHeaderTemplate),
		"oneoffHeader":     makePathTemplate("oneoffHeader", goPathOneOffHeaderTemplate),
		"fakeroot":         makePathTemplate("fakeroot", goFakeRootTemplate),
		"struct":           makePathTemplate("struct", goPathStructTemplate),
		"childConstructor": makePathTemplate("childConstructor", goChildConstructorTemplate),
	}
)

// makePathTemplate generates a template.Template for a particular named source template
func makePathTemplate(name, src string) *template.Template {
	return template.Must(template.New(name).Parse(src))
}

// getNodeDataMap returns the NodeDataMap for the provided schema given its
// parsed information. The directories map is keyed by the path of the
// directory entries. leafTypeMap stores type information for all nodes, and is
// keyed first also by the path of the directory entries, and second by the
// schema field names of that directory entry (i.e. the same keys as the
// "Fields" map of the Directory entry). Since ygen provides a *MappedType for
// every leaf node only, leafTypeMap's value is nil for non-leaf nodes.
// If a directory or field doesn't exist in the leafTypeMap, then an error is returned.
// Note: Top-level nodes, but *not* the fake root, are part of the output.
func getNodeDataMap(directories map[string]*ygen.Directory, leafTypeMap map[string]map[string]*ygen.MappedType, schemaStructPkgAlias string) (NodeDataMap, util.Errors) {
	nodeDataMap := NodeDataMap{}
	var errs util.Errors
	for path, dir := range directories {
		goFieldNameMap := ygen.GoFieldNameMap(dir)
		fieldTypeMap, ok := leafTypeMap[path]
		if !ok {
			errs = util.AppendErr(errs, fmt.Errorf("getChildDataList: directory path %q does not exist in leafTypeMap's keys", path))
			continue
		}
		for fieldName, field := range dir.Fields {
			pathStructName, err := getFieldTypeName(dir, fieldName, goFieldNameMap[fieldName], directories)
			if err != nil {
				errs = util.AppendErr(errs, err)
				continue
			}
			mType, ok := fieldTypeMap[fieldName]
			if !ok {
				errs = util.AppendErr(errs, fmt.Errorf("getChildDataList: field name %q does not exist for directory %q in the map of field names to their MappedType values: %v", fieldName, path, fieldTypeMap))
				continue
			}

			isLeaf := mType != nil
			var goTypeName string
			switch {
			case !isLeaf:
				goTypeName = "*" + schemaStructPkgAlias + "." + pathStructName
			case field.ListAttr != nil && ygen.IsYgenDefinedGoType(mType):
				goTypeName = "[]" + schemaStructPkgAlias + "." + mType.NativeType
			case ygen.IsYgenDefinedGoType(mType):
				goTypeName = schemaStructPkgAlias + "." + mType.NativeType
			case field.ListAttr != nil:
				goTypeName = "[]" + mType.NativeType
			default:
				goTypeName = mType.NativeType
			}

			var yangTypeName string
			if isLeaf {
				yangTypeName = field.Type.Name
			}
			nodeDataMap[pathStructName] = &NodeData{
				GoTypeName:       goTypeName,
				GoFieldName:      goFieldNameMap[fieldName],
				ParentGoTypeName: dir.Name,
				IsLeaf:           isLeaf,
				IsScalarField:    ygen.IsScalarField(field, mType),
				YANGTypeName:     yangTypeName,
			}
		}
	}

	if len(errs) != 0 {
		return nil, errs
	}
	return nodeDataMap, nil
}

// writeHeader parses the yangFiles from the includePaths, and fills the given
// *GeneratedPathCode with the header of the generated Go path code.
func writeHeader(yangFiles, includePaths []string, cg *GenConfig, genCode *GeneratedPathCode) error {
	// Build input to the header template which stores parameters which are included
	// in the header of generated code.
	s := struct {
		GoImports                        // GoImports contains package import options.
		PackageName             string   // PackageName is the name that should be used for the generating package.
		GeneratingBinary        string   // GeneratingBinary is the name of the binary calling the generator library.
		YANGFiles               []string // YANGFiles contains the list of input YANG source files for code generation.
		IncludePaths            []string // IncludePaths contains the list of paths that included modules were searched for in.
		SchemaStructPkgAlias    string   // SchemaStructPkgAlias is the package alias for the imported ygen-generated file.
		PathBaseTypeName        string   // PathBaseTypeName is the type name of the common embedded path struct.
		PathStructInterfaceName string   // PathStructInterfaceName is the name of the interface which all path structs implement.
		FakeRootTypeName        string   // FakeRootTypeName is the type name of the fakeroot node in the generated code.
	}{
		GoImports:               cg.GoImports,
		PackageName:             cg.PackageName,
		GeneratingBinary:        cg.GeneratingBinary,
		YANGFiles:               yangFiles,
		IncludePaths:            includePaths,
		SchemaStructPkgAlias:    cg.SchemaStructPkgAlias,
		PathBaseTypeName:        ygot.PathBaseTypeName,
		PathStructInterfaceName: ygot.PathStructInterfaceName,
		FakeRootTypeName:        yang.CamelCase(cg.FakeRootName),
	}

	var common bytes.Buffer
	if err := goPathTemplates["commonHeader"].Execute(&common, s); err != nil {
		return err
	}

	var oneoff bytes.Buffer
	if err := goPathTemplates["oneoffHeader"].Execute(&oneoff, s); err != nil {
		return err
	}

	genCode.CommonHeader = common.String()
	genCode.OneOffHeader = oneoff.String()
	return nil
}

// goPathStructData stores template information needed to generate a struct
// field's type definition.
type goPathStructData struct {
	// TypeName is the type name of the struct being output.
	TypeName string
	// YANGPath is the schema path of the struct being output.
	YANGPath string
	// PathBaseTypeName is the type name of the common embedded path struct.
	PathBaseTypeName string
	// PathStructInterfaceName is the name of the interface which all path structs implement.
	PathStructInterfaceName string
	// WildcardSuffix is the suffix given to the wildcard versions of
	// each node that distinguishes each from its non-wildcard counterpart.
	WildcardSuffix string
}

// getStructData returns the goPathStructData corresponding to a Directory,
// which is used to store the attributes of the template for which code is
// being generated.
func getStructData(directory *ygen.Directory) goPathStructData {
	return goPathStructData{
		TypeName:                directory.Name,
		YANGPath:                util.SlicePathToString(directory.Path),
		PathBaseTypeName:        ygot.PathBaseTypeName,
		PathStructInterfaceName: ygot.PathStructInterfaceName,
		WildcardSuffix:          WildcardSuffix,
	}
}

// goPathFieldData stores template information needed to generate a struct
// field's child constructor method.
type goPathFieldData struct {
	MethodName      string           // MethodName is the name of the method that can be called to get to this field.
	SchemaName      string           // SchemaName is the field's original name in the schema.
	TypeName        string           // TypeName is the type name of the returned struct.
	RelPathList     string           // RelPathList is the list of strings that form the relative path from its containing struct.
	Struct          goPathStructData // Struct stores template information for the field's containing struct.
	KeyParamListStr string           // KeyParamListStr is the parameter list of the field's accessor method.
	KeyEntriesStr   string           // KeyEntriesStr is an ordered list of comma-separated ("schemaName": unique camel-case name) for a list's keys.
}

// generateDirectorySnippet generates all Go code associated with a schema node
// (container, list, leaf, or fakeroot), all of which have a corresponding
// struct onto which to attach the necessary methods for path generation. The
// code comprises of the type definition for the struct, and all accessors to
// the fields of the struct. directory is the parsed information of a schema
// node, and directories is a map from path to a parsed schema node for all
// nodes in the schema.
func generateDirectorySnippet(directory *ygen.Directory, directories map[string]*ygen.Directory, schemaStructPkgAlias string) (GoPathStructCodeSnippet, util.Errors) {
	var errs util.Errors
	// structBuf is used to store the code associated with the struct defined for
	// the target YANG entity.
	var structBuf bytes.Buffer
	var methodBuf bytes.Buffer

	// Output struct snippets.
	structData := getStructData(directory)
	if ygen.IsFakeRoot(directory.Entry) {
		// Fakeroot has its unique output.
		if err := goPathTemplates["fakeroot"].Execute(&structBuf, structData); err != nil {
			return GoPathStructCodeSnippet{}, util.AppendErr(errs, err)
		}
	} else if err := goPathTemplates["struct"].Execute(&structBuf, structData); err != nil {
		return GoPathStructCodeSnippet{}, util.AppendErr(errs, err)
	}

	goFieldNameMap := ygen.GoFieldNameMap(directory)

	// Generate child constructor snippets for all fields of the node.
	// Alphabetically order fields to produce deterministic output.
	for _, fieldName := range ygen.GetOrderedFieldNames(directory) {
		field, ok := directory.Fields[fieldName]
		if !ok {
			errs = util.AppendErr(errs, fmt.Errorf("generateDirectorySnippet: field %s not found in directory %v", fieldName, directory))
			continue
		}
		goFieldName := goFieldNameMap[fieldName]

		if es := generateChildConstructors(&methodBuf, directory, fieldName, goFieldName, directories, schemaStructPkgAlias); es != nil {
			errs = util.AppendErrs(errs, es)
		}

		// Since leaves don't have their own Directory entries, we need
		// to output their struct snippets somewhere, and here is
		// convenient.
		if field.IsLeaf() || field.IsLeafList() {
			leafTypeName, err := getFieldTypeName(directory, fieldName, goFieldName, directories)
			if err != nil {
				errs = util.AppendErr(errs, err)
			} else {
				structData := goPathStructData{
					TypeName:                leafTypeName,
					YANGPath:                field.Path(),
					PathBaseTypeName:        ygot.PathBaseTypeName,
					PathStructInterfaceName: ygot.PathStructInterfaceName,
					WildcardSuffix:          WildcardSuffix,
				}
				if err := goPathTemplates["struct"].Execute(&structBuf, structData); err != nil {
					errs = util.AppendErr(errs, err)
				}
			}
		}
	}

	if len(errs) == 0 {
		errs = nil
	}
	return GoPathStructCodeSnippet{
		PathStructName:    structData.TypeName,
		StructBase:        structBuf.String(),
		ChildConstructors: methodBuf.String(),
	}, errs
}

// generateChildConstructors generates and writes to methodBuf the Go methods
// that returns an instantiation of the child node's path struct object. It
// takes as input the buffer to store the method, a directory, the field name
// of the directory identifying the child yang.Entry, a directory-level unique
// field name to be used as the generated method's name and the incremental
// type name of of the child path struct, and a map of all directories of the
// whole schema keyed by their schema paths.
func generateChildConstructors(methodBuf *bytes.Buffer, directory *ygen.Directory, directoryFieldName string, goFieldName string, directories map[string]*ygen.Directory, schemaStructPkgAlias string) []error {
	field, ok := directory.Fields[directoryFieldName]
	if !ok {
		return []error{fmt.Errorf("generateChildConstructors: field %s not found in directory %v", directoryFieldName, directory)}
	}
	fieldTypeName, err := getFieldTypeName(directory, directoryFieldName, goFieldName, directories)
	if err != nil {
		return []error{err}
	}

	structData := getStructData(directory)
	relPath, err := ygen.FindSchemaPath(directory, directoryFieldName, false)
	if err != nil {
		return []error{err}
	}
	fieldData := goPathFieldData{
		MethodName:  goFieldName,
		TypeName:    fieldTypeName,
		SchemaName:  field.Name,
		Struct:      structData,
		RelPathList: `"` + strings.Join(relPath, `", "`) + `"`,
	}

	isUnderFakeRoot := ygen.IsFakeRoot(directory.Entry)

	// This is expected to be nil for leaf fields.
	fieldDirectory := directories[field.Path()]

	if field.IsList() { // else, the field is a container or leaf.
		if fieldDirectory.ListAttr == nil {
			// TODO(wenbli): keyless lists as a path are not supported by gNMI, but this
			// library is currently intended for gNMI, so need to decide on a long-term solution.

			// As a short-term solution, we just need to prevent the user from accessing any node in the keyless list's subtree.
			// Here, we simply skip generating the child constructor, such that its subtree is unreachable.
			return nil
			// Erroring out, on the other hand, is impractical due to their existence in the current OpenConfig models.
			// return fmt.Errorf("generateChildConstructors: schemas containing keyless lists are unsupported, path: %s", field.Path())
		}

		return generateChildConstructorsForList(methodBuf, fieldDirectory.ListAttr, fieldData, isUnderFakeRoot, schemaStructPkgAlias)
	}

	return generateChildConstructorsForLeafOrContainer(methodBuf, fieldData, isUnderFakeRoot)
}

// generateChildConstructorsForLeafOrContainer writes into methodBuf the child
// constructor snippets for the container or leaf template output information
// contained in fieldData.
func generateChildConstructorsForLeafOrContainer(methodBuf *bytes.Buffer, fieldData goPathFieldData, isUnderFakeRoot bool) []error {
	// Generate child constructor for the non-wildcard version of the parent struct.
	var errors []error
	if err := goPathTemplates["childConstructor"].Execute(methodBuf, fieldData); err != nil {
		errors = append(errors, err)
	}

	// The root node doesn't have a wildcard version of itself.
	if isUnderFakeRoot {
		return errors
	}

	// Generate child constructor for the wildcard version of the parent struct.
	fieldData.TypeName += WildcardSuffix
	fieldData.Struct.TypeName += WildcardSuffix
	if err := goPathTemplates["childConstructor"].Execute(methodBuf, fieldData); err != nil {
		errors = append(errors, err)
	}
	return errors
}

// generateChildConstructorsForList writes into methodBuf the child constructor
// method snippets for the list represented by listAttr. fieldData contains the
// childConstructor template output information for if the node were a
// container (which contains a subset of the basic information required for
// the list constructor methods).
func generateChildConstructorsForList(methodBuf *bytes.Buffer, listAttr *ygen.YangListAttr, fieldData goPathFieldData, isUnderFakeRoot bool, schemaStructPkgAlias string) []error {
	var errors []error
	// List of function parameters as would appear in the method definition.
	keyParamListStrs, err := makeParamListStrs(listAttr, schemaStructPkgAlias)
	if err != nil {
		return append(errors, err)
	}
	// List of key parameters as would appear in the key attribute of a
	// ygot.NodePath definition.
	keyNames, keyVarNames := makeKeyMapStrs(listAttr)
	keyN := len(keyParamListStrs)
	combos := combinations(keyN)

	// Names that are subject to change depending on which keys are
	// wildcarded and whether the parent struct is a wildcard node.
	baseMethodName := fieldData.MethodName
	parentTypeName := fieldData.Struct.TypeName
	wildcardParentTypeName := parentTypeName + WildcardSuffix
	fieldTypeName := fieldData.TypeName
	wildcardFieldTypeName := fieldTypeName + WildcardSuffix

	// For each combination of parameter indices to be part of the method
	// parameter list (i.e. NOT wildcarded).
	for comboIndex, combo := range combos {
		var paramListStrs, keyEntryStrs []string
		var anySuffixes []string

		i := 0 // Loop through each parameter
		for _, paramIndex := range combo {
			// Add unselected parameters as a wildcard.
			for ; i != paramIndex; i++ {
				keyEntryStrs = append(keyEntryStrs, fmt.Sprintf(`"%s": "*"`, keyNames[i]))
				anySuffixes = append(anySuffixes, WildcardSuffix+keyVarNames[i])
			}
			// Add selected parameters to the parameter list.
			paramListStrs = append(paramListStrs, keyParamListStrs[paramIndex])
			keyEntryStrs = append(keyEntryStrs, fmt.Sprintf(`"%s": %s`, keyNames[paramIndex], keyVarNames[paramIndex]))
			i++
		}
		for ; i != keyN; i++ { // Handle edge case
			keyEntryStrs = append(keyEntryStrs, fmt.Sprintf(`"%s": "*"`, keyNames[i]))
			anySuffixes = append(anySuffixes, WildcardSuffix+keyVarNames[i])
		}
		// Create the string for the method parameter list and ygot.NodePath's key list.
		fieldData.KeyParamListStr = strings.Join(paramListStrs, ", ")
		fieldData.KeyEntriesStr = strings.Join(keyEntryStrs, ", ")

		// Add wildcard description suffixes to the base method name
		// for wildcarded parameters.
		fieldData.MethodName = baseMethodName + strings.Join(anySuffixes, "")
		// By default, set the child type to be the wildcard version.
		fieldData.TypeName = wildcardFieldTypeName

		// Corner cases
		switch {
		case comboIndex == 0:
			// When all keys are wildcarded, just use
			// WildcardSuffix alone as the suffix.
			fieldData.MethodName = baseMethodName + WildcardSuffix
		case comboIndex == len(combos)-1:
			// When all keys are not wildcarded, then the child
			// type should be the non-wildcard version.
			fieldData.TypeName = fieldTypeName
		}

		// Generate child constructor method for non-wildcard version of parent struct.
		fieldData.Struct.TypeName = parentTypeName
		if err := goPathTemplates["childConstructor"].Execute(methodBuf, fieldData); err != nil {
			errors = append(errors, err)
		}

		// The root node doesn't have a wildcard version of itself.
		if isUnderFakeRoot {
			continue
		}

		// Generate child constructor method for wildcard version of parent struct.
		fieldData.Struct.TypeName = wildcardParentTypeName
		// Override the corner case for generating the non-wildcard child.
		fieldData.TypeName = wildcardFieldTypeName
		if err := goPathTemplates["childConstructor"].Execute(methodBuf, fieldData); err != nil {
			errors = append(errors, err)
		}
	}
	return errors
}

// getFieldTypeName returns the type name for a field node of a directory -
// handling the case where the field supplied is a leaf or directory. The input
// directories is a map from paths to directory entries, and goFieldName is the
// incremental type name to be used for the case that the directory field is a
// leaf. For non-leaves, their corresponding directories' "Name"s, which are the
// same names as their corresponding ygen Go struct type names, are re-used as
// their type names; for leaves, type names are synthesized.
func getFieldTypeName(directory *ygen.Directory, directoryFieldName string, goFieldName string, directories map[string]*ygen.Directory) (string, error) {
	field, ok := directory.Fields[directoryFieldName]
	if !ok {
		return "", fmt.Errorf("getFieldTypeName: field %s not found in directory %v", directoryFieldName, directory)
	}

	if !field.IsLeaf() && !field.IsLeafList() {
		fieldDirectory, ok := directories[field.Path()]
		if !ok {
			return "", fmt.Errorf("getFieldTypeName: unexpected - field %s not found in parsed yang structs map: %v", field.Path(), directories)
		}
		return fieldDirectory.Name, nil
	}

	// Leaves do not have corresponding Directory entries, so their names need to be constructed.
	if isTopLevelLeaf := directory.Entry.Parent == nil; isTopLevelLeaf {
		// When a leaf resides at the root, its type name is its whole name -- we never want fakeroot's name as a prefix.
		return goFieldName, nil
	}
	return directory.Name + "_" + goFieldName, nil
}

// makeKeyMapStrs returns the components of the literal instantiations of the
// "key" attribute of a ygot.NodePath, i.e. the literal key and value strings
// to be output within the "{}" of its map[string]interface{} field. The first
// returned slice contains the YANG key names, and the second slice the
// camel-cased and uniquified versions of them, done in an identical manner as
// makeParamListStrs to ensure compilation.
// e.g.
// in: &ygen.YangListAttr{
// 	Keys: map[string]*ygen.MappedType{
// 		"fluorine":        &ygen.MappedType{NativeType: "string"},
// 		"iodine-liquid":   &ygen.MappedType{NativeType: "Binary"},
// 	},
// 	KeyElems: []*yang.Entry{{Name: "fluorine"}, {Name: "iodine-liquid"}},
// }
// out: {"fluorine", "iodine-liquid"}, {"Fluorine", "IodineLiquid"}
func makeKeyMapStrs(listAttr *ygen.YangListAttr) ([]string, []string) {
	var keyNames, keyVarNames []string
	goKeyNameMap := getGoKeyNameMap(listAttr.KeyElems)
	for _, key := range listAttr.KeyElems { // NOTE: loop on list for deterministic output.
		keyNames = append(keyNames, key.Name)
		keyVarNames = append(keyVarNames, goKeyNameMap[key.Name])
	}
	return keyNames, keyVarNames
}

// makeParamListStrs generates the list of go parameter list components for a
// child list's constructor method given the list's ygen.YangListAttr.
// It outputs the parameters in the same order as in the YangListAttr.
// e.g.
// in: &ygen.YangListAttr{
// 	Keys: map[string]*ygen.MappedType{
// 		"fluorine": &ygen.MappedType{NativeType: "string"},
// 		"iodine-liquid":   &ygen.MappedType{NativeType: "Binary"},
// 	},
// 	KeyElems: []*yang.Entry{{Name: "fluorine"}, {Name: "iodine-liquid"}},
// }
// out: {"Fluorine string", "IodineLiquid oc.Binary"}
func makeParamListStrs(listAttr *ygen.YangListAttr, schemaStructPkgAlias string) ([]string, error) {
	if len(listAttr.KeyElems) == 0 {
		return nil, fmt.Errorf("makeParamListStrs: invalid list - has no key; cannot process param list string")
	}

	// Create parameter list *in order* of keys, which should be in schema order.
	var entries []string
	// NOTE: Although the generated key names might not match their
	// corresponding ygen field names in case of a camelcase name
	// collision, we expect that the user is aware of the schema to know
	// the order of the keys, and not rely on the naming in that case.
	goKeyNameMap := getGoKeyNameMap(listAttr.KeyElems)
	for _, keyElem := range listAttr.KeyElems {
		mappedType, ok := listAttr.Keys[keyElem.Name]
		switch {
		case !ok:
			return nil, fmt.Errorf("makeParamListStrs: key doesn't have a mappedType: %s", keyElem.Name)
		case mappedType == nil:
			return nil, fmt.Errorf("makeParamListStrs: mappedType for key is nil: %s", keyElem.Name)
		}

		var typeName string
		switch {
		case mappedType.NativeType == "interface{}": // ygen-unsupported types
			typeName = "string"
		case ygen.IsYgenDefinedGoType(mappedType):
			typeName = schemaStructPkgAlias + "." + mappedType.NativeType
		default:
			typeName = mappedType.NativeType
		}

		entries = append(entries, fmt.Sprintf("%s %s", goKeyNameMap[keyElem.Name], typeName))
	}
	return entries, nil
}

// combinations returns the mathematical combinations of the numbers from 0 to n-1.
// e.g. n = 2 -> []int{{}, {0}, {1}, {0, 1}}
// It outputs combination(0) if n < 0.
// Guarantees:
// - Deterministic output.
// - All numbers within a combination are in order.
// - The first combination is the shortest (i.e. containing no numbers).
// - The last combination is the longest (i.e. containing all numbers from 0 to n-1).
func combinations(n int) [][]int {
	cs := [][]int{{}}
	for i := 0; i < n; i++ {
		size := len(cs)
		for j := 0; j != size; j++ {
			cs = append(cs, append(append([]int{}, cs[j]...), i))
		}
	}
	return cs
}

// getGoKeyNameMap returns a map of Go key names keyed by their schema names
// given a list of key entries. Names are camelcased and uniquified to ensure
// compilation. Uniqification is done deterministically.
func getGoKeyNameMap(keyElems []*yang.Entry) map[string]string {
	goKeyNameMap := make(map[string]string, len(keyElems))

	usedKeyNames := map[string]bool{}
	for _, keyElem := range keyElems {
		goKeyNameMap[keyElem.Name] = genutil.MakeNameUnique(genutil.EntryCamelCaseName(keyElem), usedKeyNames)
	}
	return goKeyNameMap
}
