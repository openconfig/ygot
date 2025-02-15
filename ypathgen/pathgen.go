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
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"text/template"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/genutil"
	"github.com/openconfig/ygot/gogen"
	"github.com/openconfig/ygot/util"
	"github.com/openconfig/ygot/ygen"
	"github.com/openconfig/ygot/ygot"
)

// Static default configuration values that differ from the zero value for their types.
const (
	// defaultPathPackageName specifies the default name that should be
	// used for the generated Go package.
	defaultPathPackageName = "ocpathstructs"
	// defaultFakeRootName is the default name for the root structure.
	defaultFakeRootName = "root"
	// defaultPathStructSuffix is the default suffix for generated
	// PathStructs to distinguish them from the generated GoStructs
	defaultPathStructSuffix = "Path"
	// defaultPackageSuffix is the default suffix for generated packages.
	defaultPackageSuffix = "path"
	// schemaStructPkgAlias is the package alias of the schema struct
	// package when the path struct package is to be generated in a
	// separate package.
	schemaStructPkgAlias = "oc"
	// WildcardSuffix is the suffix given to the wildcard versions of each
	// node as well as a list's wildcard child constructor methods that
	// distinguishes each from its non-wildcard counterpart.
	WildcardSuffix = "Any"
	// BuilderCtorSuffix is the suffix applied to the list builder
	// constructor method's name in order to indicate itself to the user.
	BuilderCtorSuffix = "Any"
	// BuilderKeyPrefix is the prefix applied to the key-modifying builder
	// method for a list PathStruct that uses the builder API.
	// NOTE: This cannot be "", as the builder method name would conflict
	// with the child constructor method for the keys.
	BuilderKeyPrefix = "With"

	// yangTypeNameFlagKey is a custom flag for storing the YANG type's
	// name for a YANG node.
	yangTypeNameFlagKey = "YANG:typename"
)

// NewDefaultConfig creates a GenConfig with default configuration.
// schemaStructPkgPath is a required configuration parameter. It should be set
// to "" when the generated PathStruct package is to be the same package as the
// GoStructs package.
func NewDefaultConfig(schemaStructPkgPath string) *GenConfig {
	return &GenConfig{
		PackageName:   defaultPathPackageName,
		PackageSuffix: defaultPackageSuffix,
		GoImports: GoImports{
			SchemaStructPkgPath: schemaStructPkgPath,
			YgotImportPath:      genutil.GoDefaultYgotImportPath,
		},
		FakeRootName:     defaultFakeRootName,
		PathStructSuffix: defaultPathStructSuffix,
		GeneratingBinary: genutil.CallerName(),
	}
}

// GenConfig stores code generation configuration.
type GenConfig struct {
	// PackageName is the name that should be used for the generating package.
	PackageName string
	// GoImports contains package import options.
	GoImports GoImports
	// PreferOperationalState generates path-build methods for only the
	// "state" version of a field when it exists under both "config" and
	// "state" containers of its parent YANG model. If it is false, then
	// the reverse is true. There are no omissions if a conflict does not
	// exist, e.g. if a leaf exists only under a "state" container, then
	// its path-building method will always be generated, and use "state".
	PreferOperationalState bool
	// ExcludeState determines whether derived state leaves are excluded
	// from the path-building methods.
	ExcludeState bool
	// FakeRootName specifies the name of the struct that should be generated
	// representing the root.
	FakeRootName string
	// PathStructSuffix is the suffix to be appended to generated
	// PathStructs to distinguish them from the generated GoStructs, which
	// assume a similar name.
	PathStructSuffix string
	// SkipEnumDeduplication specifies whether leaves of type 'enumeration' that
	// are used in multiple places in the schema should share a common type within
	// the generated code that is output by ygen. By default (false), a common type
	// is used.
	// This is the same flag used by ygen: they must match for pathgen's
	// generated code to be compatible with it.
	SkipEnumDeduplication bool
	// ShortenEnumLeafNames removes the module name from the name of
	// enumeration leaves.
	// This is the same flag used by ygen: they must match for pathgen's
	// generated code to be compatible with it.
	ShortenEnumLeafNames bool
	// EnumOrgPrefixesToTrim trims the organization name from the module
	// part of the name of enumeration leaves if there is a match.
	EnumOrgPrefixesToTrim []string
	// UseDefiningModuleForTypedefEnumNames uses the defining module name
	// to prefix typedef enumerated types instead of the module where the
	// typedef enumerated value is used.
	// This is the same flag used by ygen: they must match for pathgen's
	// generated code to be compatible with it.
	UseDefiningModuleForTypedefEnumNames bool
	// AppendEnumSuffixForSimpleUnionEnums appends an "Enum" suffix to the
	// enumeration name for simple (i.e. non-typedef) leaves which are
	// unions with an enumeration inside. This makes all inlined
	// enumerations within unions, whether typedef or not, have this
	// suffix, achieving consistency.  Since this flag is planned to be a
	// v1 compatibility flag along with
	// UseDefiningModuleForTypedefEnumNames, and will be removed in v1, it
	// only applies when useDefiningModuleForTypedefEnumNames is also set
	// to true.
	AppendEnumSuffixForSimpleUnionEnums bool
	// IgnoreUnsupportedStatements ignores unsupported YANG statements when
	// parsing, such that they do not show up errors during IR generation.
	IgnoreUnsupportedStatements bool
	// ExcludeModules specifies any modules that are included within the set of
	// modules that should have code generated for them that should be ignored during
	// code generation. This is due to the fact that some schemas (e.g., OpenConfig
	// interfaces) currently result in overlapping entities (e.g., /interfaces).
	ExcludeModules []string
	// YANGParseOptions provides the options that should be handed to the
	// github.com/openconfig/goyang/pkg/yang library. These specify how the
	// input YANG files should be parsed.
	YANGParseOptions yang.Options
	// GeneratingBinary is the name of the binary calling the generator library, it is
	// included in the header of output files for debugging purposes. If a
	// string is not specified, the location of the library is utilised.
	GeneratingBinary string
	// ListBuilderKeyThreshold means to use the builder API format instead
	// of the key-combination API format for constructing list keys when
	// the number of keys is at least the threshold value.
	// 0 (default) means no threshold, i.e. always use the key-combination
	// API format.
	ListBuilderKeyThreshold uint
	// GenerateWildcardPaths means to generate wildcard nodes and paths.
	GenerateWildcardPaths bool
	// SimplifyWildcardPaths causes non-builder-style generated wildcard
	// nodes, where all key values are wildcards, to omit the [key="*"] in
	// the generated path.
	//
	// e.g. For the following path node,
	//
	// list foo {
	//  key "one two three";
	// }
	//
	// "foo[one=*][two=*][three=*]" would be the string representation for
	// all keys being wildcards when this flag is false, whereas simply
	// "foo" when the flag is true. These two representations are
	// equivalent per the gNMI specification.
	// If any key is not a wildcard, then this flag doesn't apply, since
	// all key values must now be specified in the path.
	SimplifyWildcardPaths bool
	// SplitByModule controls whether to generate a go package for each yang module.
	SplitByModule bool
	// TrimPackagePrefix is the prefix to trim from generated go package names.
	TrimPackagePrefix string
	// BaseImportPath is used to create to full import path of the generated go packages.
	BaseImportPath string
	// PackageString is the string to apppend to the generated Go package names.
	PackageSuffix string
}

// GoImports contains package import options.
type GoImports struct {
	// SchemaStructPkgPath specifies the path to the ygen-generated structs, which
	// is used to get the enum and union type names used as the list key
	// for calling a list path accessor.
	SchemaStructPkgPath string
	// YgotImportPath specifies the path to the ygot library that should be used
	// in the generated code.
	YgotImportPath string
}

type goLangMapper struct {
	*gogen.GoLangMapper
}

// PopulateFieldFlags populates extra field information for pathgen.
func (goLangMapper) PopulateFieldFlags(nd ygen.NodeDetails, field *yang.Entry) map[string]string {
	flags := map[string]string{}
	if nd.Type == ygen.LeafNode || nd.Type == ygen.LeafListNode {
		// Only leaf or leaf-list nodes can have type statements.
		flags[yangTypeNameFlagKey] = field.Type.Name
	}
	return flags
}

// GeneratePathCode takes a slice of strings containing the path to a set of YANG
// files which contain YANG modules, and a second slice of strings which
// specifies the set of paths that are to be searched for associated models (e.g.,
// modules that are included by the specified set of modules, or submodules of those
// modules). It extracts the set of modules that are to be generated, and returns
// a map of package names to GeneratedPathCode structs. Each struct contains
// all the generated code of that package needed support the path-creation API.
// The important components of the generated code are listed below:
//  1. Struct definitions for each container, list, or leaf schema node,
//     as well as the fakeroot.
//  2. Next-level methods for the fakeroot and each non-leaf schema node,
//     which instantiate and return the next-level structs corresponding to
//     its child schema nodes.
//
// With these components, the generated API is able to support absolute path
// creation of any node of the input schema.
// Also returned is the NodeDataMap of the schema, i.e. information about each
// node in the generated code, which may help callers add customized
// augmentations to the basic generated path code.
// If errors are encountered during code generation, they are returned.
func (cg *GenConfig) GeneratePathCode(yangFiles, includePaths []string) (map[string]*GeneratedPathCode, NodeDataMap, util.Errors) {
	// Note: The input configuration may cause the code to not compile.
	// While it's possible to write checks for better error messages, the
	// many ways in which compilation may fail, coupled with the plethora
	// of configurations, means there is an argument to force the user to
	// debug instead of making ypathgen having to catch every error.
	compressBehaviour, err := genutil.TranslateToCompressBehaviour(true, cg.ExcludeState, cg.PreferOperationalState)
	if err != nil {
		return nil, nil, util.NewErrs(fmt.Errorf("ypathgen: unable to translate compress behaviour: %v", err))
	}

	opts := ygen.IROptions{
		ParseOptions: ygen.ParseOpts{
			IgnoreUnsupportedStatements: cg.IgnoreUnsupportedStatements,
			YANGParseOptions:            cg.YANGParseOptions,
			ExcludeModules:              cg.ExcludeModules,
		},
		TransformationOptions: ygen.TransformationOpts{
			CompressBehaviour:                    compressBehaviour,
			GenerateFakeRoot:                     true,
			FakeRootName:                         cg.FakeRootName,
			SkipEnumDeduplication:                cg.SkipEnumDeduplication,
			ShortenEnumLeafNames:                 cg.ShortenEnumLeafNames,
			EnumOrgPrefixesToTrim:                cg.EnumOrgPrefixesToTrim,
			UseDefiningModuleForTypedefEnumNames: cg.UseDefiningModuleForTypedefEnumNames,
			EnumerationsUseUnderscores:           true,
		},
		NestedDirectories:                   false,
		AbsoluteMapPaths:                    false,
		AppendEnumSuffixForSimpleUnionEnums: cg.AppendEnumSuffixForSimpleUnionEnums,
	}

	var errs util.Errors
	ir, err := ygen.GenerateIR(yangFiles, includePaths, goLangMapper{GoLangMapper: gogen.NewGoLangMapper(true)}, opts)
	if err != nil {
		return nil, nil, util.AppendErr(errs, err)
	}

	var schemaStructPkgAccessor string
	if cg.GoImports.SchemaStructPkgPath != "" {
		schemaStructPkgAccessor = schemaStructPkgAlias + "."
	}

	// Get NodeDataMap for the schema.
	nodeDataMap, es := getNodeDataMap(ir, cg.FakeRootName, schemaStructPkgAccessor, cg.PathStructSuffix, cg.PackageName, cg.PackageSuffix, cg.TrimPackagePrefix, cg.SplitByModule)
	if es != nil {
		errs = util.AppendErrs(errs, es)
	}

	// Generate struct code.
	var structSnippets []GoPathStructCodeSnippet
	for _, directoryPath := range ir.OrderedDirectoryPathsByName() {
		directory := ir.Directories[directoryPath]

		var listBuilderKeyThreshold uint
		if cg.GenerateWildcardPaths {
			listBuilderKeyThreshold = cg.ListBuilderKeyThreshold
		}

		structSnippet, es := generateDirectorySnippet(directory, ir.Directories, schemaStructPkgAccessor, cg.PathStructSuffix, listBuilderKeyThreshold, cg.GenerateWildcardPaths, cg.SimplifyWildcardPaths, cg.SplitByModule, cg.PackageName, cg.PackageSuffix, cg.TrimPackagePrefix)
		if es != nil {
			errs = util.AppendErrs(errs, es)
		}
		structSnippets = append(structSnippets, structSnippet...)
	}

	// Aggregate snippets by package and compute their deps.
	packages := map[string]*GeneratedPathCode{}
	for _, snippet := range structSnippets {
		if _, ok := packages[snippet.Package]; !ok {
			packages[snippet.Package] = &GeneratedPathCode{
				Deps: make(map[string]bool),
			}
		}
		packages[snippet.Package].Structs = append(packages[snippet.Package].Structs, snippet)
		for _, d := range snippet.Deps {
			packages[snippet.Package].Deps[d] = true
		}
	}
	for name, p := range packages {
		err := writeHeader(yangFiles, includePaths, name, cg, p)
		util.AppendErr(errs, err)
	}

	if len(errs) == 0 {
		errs = nil
	}
	return packages, nodeDataMap, errs
}

// packageNameReplacePattern matches all characters allowed in yang modules, but not go packages.
var packageNameReplacePattern = regexp.MustCompile("[._-]")

// goPackageName returns the go package to use when generating code given the
// input root module, which is the module in which the YANG tree the node is
// attached to was instantiated (rather than the module that has the same
// namespace as the node).
//
// If splitByModule is false, the pkgName is always returned. Otherwise,
// a transformed version of the module that the directory belongs to is returned.
// If trimPrefix is non-empty, trimPrefix is removed from the package name.
// fakeRootPkgName is the name of the package that contains just the fake root path struct.
func goPackageName(rootModuleName string, splitByModule, isFakeRoot bool, pkgName, pkgSuffix, trimPrefix string) string {
	if !splitByModule || isFakeRoot {
		return pkgName
	}
	name := rootModuleName
	if trimPrefix != "" {
		name = strings.TrimPrefix(name, trimPrefix)
	}

	name = packageNameReplacePattern.ReplaceAllString(name, "")
	return strings.ToLower(name) + pkgSuffix
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
	Deps         map[string]bool           // Deps is the list of packages that this package depends on.
}

// String method for GeneratedPathCode, which can be used to write all the
// generated code into a single file.
func (genCode GeneratedPathCode) String() string {
	var gotCode strings.Builder
	gotCode.WriteString(genCode.CommonHeader)
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
	if fileN < 1 {
		return nil, fmt.Errorf("requested %d files, but must be between 1 and %d (number of structs)", fileN, structN)
	}

	files := []string{}
	structsPerFile := int(math.Ceil(float64(structN) / float64(fileN)))

	fileContent := [][]string{}
	structs := []string{}
	for i, gotStruct := range genCode.Structs {
		if i != 0 && i%structsPerFile == 0 {
			fileContent = append(fileContent, structs)
			structs = []string{}
		}
		structs = append(structs, gotStruct.String())
	}
	fileContent = append(fileContent, structs)
	var gotCode strings.Builder
	for _, f := range fileContent {
		gotCode.WriteString(genCode.CommonHeader)
		for _, s := range f {
			gotCode.WriteString(s)
		}
		files = append(files, gotCode.String())
		gotCode.Reset()
	}

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
	// Package is the name of the package that this snippet belongs to.
	Package string
	// Deps are any packages that this snippet depends on.
	Deps []string
}

// String returns the contents of a GoPathStructCodeSnippet as a string by
// simply writing out all of its generated code.
func (g GoPathStructCodeSnippet) String() string {
	var b strings.Builder
	for _, method := range []string{g.StructBase, g.ChildConstructors} {
		genutil.WriteIfNotEmpty(&b, method)
	}
	return b.String()
}

// NodeDataMap is a map from the path struct type name of a schema node to its NodeData.
type NodeDataMap map[string]*NodeData

// NodeData contains information about the ygen-generated code of a YANG schema node.
type NodeData struct {
	// GoTypeName is the generated Go type name of a schema node. It is
	// qualified by the SchemaStructPkgAlias if necessary. It could be a
	// GoStruct or a leaf type.
	GoTypeName string
	// LocalGoTypeName is the generated Go type name of a schema node, but
	// always with the SchemaStructPkgAlias stripped. It could be a
	// GoStruct or a leaf type.
	LocalGoTypeName string
	// GoFieldName is the field name of the node under its parent struct.
	GoFieldName string
	// SubsumingGoStructName is the GoStruct type name corresponding to the node. If
	// the node is a leaf, then it is the parent GoStruct's name.
	SubsumingGoStructName string
	// IsLeaf indicates whether this child is a leaf node.
	IsLeaf bool
	// IsScalarField indicates a leaf that is stored as a pointer in its
	// parent struct.
	IsScalarField bool
	// HasDefault indicates whether this node has a default value
	// associated with it. This is only relevant to leaf or leaf-list
	// nodes.
	HasDefault bool
	// YANGTypeName is the type of the leaf given in the YANG file (without
	// the module prefix, if any, per goyang behaviour). If the node is not
	// a leaf this will be empty. Note that the current purpose for this is
	// to allow callers to handle certain types as special cases, but since
	// the name of the node is a very basic piece of information which
	// excludes the defining module, this is somewhat hacky, so it may be
	// removed or modified in the future.
	YANGTypeName string
	// YANGPath is the schema path of the YANG node.
	YANGPath string
	// GoPathPackageName is the Go package name containing the generated PathStruct for the schema node.
	GoPathPackageName string
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
	goPathCommonHeaderTemplate = mustTemplate("commonHeader", `
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
	{{- if .SchemaStructPkgPath }}
	{{ .SchemaStructPkgAlias }} "{{ .SchemaStructPkgPath }}"
	{{- end }}
	"{{ .YgotImportPath }}"
{{- range $import := .ExtraImports }}
	"{{ $import }}"
{{- end }}
)
`)

	// goPathFakeRootTemplate defines a template for the type definition and
	// basic methods of the fakeroot object. The fakeroot object adheres to
	// the methods of PathStructInterfaceName and FakeRootBaseTypeName in
	// order to allow its path struct descendents to use the ygot.Resolve()
	// helper function for obtaining their absolute paths.
	goPathFakeRootTemplate = mustTemplate("fakeroot", `
// {{ .TypeName }} represents the {{ .YANGPath }} YANG schema element.
type {{ .TypeName }} struct {
	*ygot.{{ .FakeRootBaseTypeName }}
}

// DeviceRoot returns a new path object from which YANG paths can be constructed.
func DeviceRoot(id string) *{{ .TypeName }} {
	return &{{ .TypeName }}{ygot.New{{- .FakeRootBaseTypeName }}(id)}
}
`)

	// goPathStructTemplate defines the template for the type definition of
	// a path node as well as its core method(s). A path struct/node is
	// either a container, list, or a leaf node in the openconfig schema
	// where the tree formed by the nodes mirrors the compressed YANG
	// schema tree. The defined type stores the relative path to the
	// current node, as well as its parent node for obtaining its absolute
	// path. There are two versions of these, non-wildcard and wildcard.
	// The wildcard version is simply a type to indicate that the path it
	// holds contains a wildcard, but is otherwise the exact same.
	goPathStructTemplate = mustTemplate("struct", `
// {{ .TypeName }} represents the {{ .YANGPath }} YANG schema element.
type {{ .TypeName }} struct {
	*ygot.{{ .PathBaseTypeName }}
}

{{- if .GenerateWildcardPaths }}

// {{ .TypeName }}{{ .WildcardSuffix }} represents the wildcard version of the {{ .YANGPath }} YANG schema element.
type {{ .TypeName }}{{ .WildcardSuffix }} struct {
	*ygot.{{ .PathBaseTypeName }}
}
{{- end }}
`)

	// goPathChildConstructorTemplate generates the child constructor method
	// for a generated struct by returning an instantiation of the child's
	// path struct object.
	goPathChildConstructorTemplate = mustTemplate("childConstructor", `
// {{ .MethodName }} ({{ .YANGNodeType }}): {{ .YANGDescription }}
// ----------------------------------------
// Defining module: "{{ .DefiningModuleName }}"
// Instantiating module: "{{ .InstantiatingModuleName }}"
// Path from parent: "{{ .RelPath }}"
// Path from root: "{{ .AbsPath }}"
{{- range $paramDocStr := .KeyParamDocStrs }}
// {{ $paramDocStr }}
{{- end }}
func (n *{{ .Struct.TypeName }}) {{ .MethodName -}} ({{ .KeyParamListStr }}) *{{ .ChildPkgAccessor }}{{ .TypeName }} {
	return &{{ .ChildPkgAccessor }}{{ .TypeName }}{
		{{ .Struct.PathBaseTypeName }}: ygot.New{{ .Struct.PathBaseTypeName }}(
			[]string{ {{- .RelPathList -}} },
			map[string]interface{}{ {{- .KeyEntriesStr -}} },
			n,
		),
	}
}
`)

	// goKeyBuilderTemplate generates a setter for a list key. This is used in the
	// builder style for the list API.
	goKeyBuilderTemplate = mustTemplate("goKeyBuilder", `
// {{ .MethodName }} sets {{ .TypeName }}'s key "{{ .KeySchemaName }}" to the specified value.
// {{ .KeyParamDocStr }}
func (n *{{ .TypeName }}) {{ .MethodName }}({{ .KeyParamName }} {{ .KeyParamType }}) *{{ .TypeName }} {
	ygot.ModifyKey(n.NodePath, "{{ .KeySchemaName }}", {{ .KeyParamName }})
	return n
}
`)
)

// mustTemplate generates a template.Template for a particular named source template
func mustTemplate(name, src string) *template.Template {
	return template.Must(template.New(name).Parse(src))
}

// getNodeDataMap returns the NodeDataMap for the provided schema given its
// parsed information.
// packageName, trimPackagePrefix, and splitByModule, are used to determine
// the generated Go package name for the generated PathStructs.
func getNodeDataMap(ir *ygen.IR, fakeRootName, schemaStructPkgAccessor, pathStructSuffix, packageName, packageSuffix, trimPackagePrefix string, splitByModule bool) (NodeDataMap, util.Errors) {
	nodeDataMap := NodeDataMap{}
	var errs util.Errors
	for _, dir := range ir.Directories {
		if dir.IsFakeRoot {
			// Since we always generate the fake root, we add the
			// fake root GoStruct to the data map as well.
			nodeDataMap[dir.Name+pathStructSuffix] = &NodeData{
				GoTypeName:            "*" + schemaStructPkgAccessor + yang.CamelCase(fakeRootName),
				LocalGoTypeName:       "*" + yang.CamelCase(fakeRootName),
				GoFieldName:           "",
				SubsumingGoStructName: yang.CamelCase(fakeRootName),
				IsLeaf:                false,
				IsScalarField:         false,
				HasDefault:            false,
				YANGTypeName:          "",
				YANGPath:              "/",
				GoPathPackageName:     goPackageName(dir.RootElementModule, splitByModule, true, packageName, packageSuffix, trimPackagePrefix),
			}
		}

		goFieldNameMap := ygen.GoFieldNameMap(dir)
		for fieldName, field := range dir.Fields {
			pathStructName, err := getFieldTypeName(dir, fieldName, goFieldNameMap[fieldName], ir.Directories, pathStructSuffix)
			if err != nil {
				errs = util.AppendErr(errs, err)
				continue
			}

			mType := field.LangType
			isLeaf := mType != nil

			subsumingGoStructName := dir.Name
			if !isLeaf {
				subsumingGoStructName = ir.Directories[field.YANGDetails.Path].Name
			}

			var goTypeName, localGoTypeName string
			switch {
			case !isLeaf:
				goTypeName = "*" + schemaStructPkgAccessor + subsumingGoStructName
				localGoTypeName = "*" + subsumingGoStructName
			case field.Type == ygen.LeafListNode && ygen.IsYgenDefinedGoType(mType):
				goTypeName = "[]" + schemaStructPkgAccessor + mType.NativeType
				localGoTypeName = "[]" + mType.NativeType
			case ygen.IsYgenDefinedGoType(mType):
				goTypeName = schemaStructPkgAccessor + mType.NativeType
				localGoTypeName = mType.NativeType
			case field.Type == ygen.LeafListNode:
				goTypeName = "[]" + mType.NativeType
			default:
				goTypeName = mType.NativeType
			}
			if localGoTypeName == "" {
				localGoTypeName = goTypeName
			}

			var yangTypeName string
			if field.Flags != nil {
				yangTypeName = field.Flags[yangTypeNameFlagKey]
			}
			nodeDataMap[pathStructName] = &NodeData{
				GoTypeName:            goTypeName,
				LocalGoTypeName:       localGoTypeName,
				GoFieldName:           goFieldNameMap[fieldName],
				SubsumingGoStructName: subsumingGoStructName,
				IsLeaf:                isLeaf,
				IsScalarField:         gogen.IsScalarField(field),
				HasDefault:            isLeaf && (len(field.YANGDetails.Defaults) > 0 || mType.DefaultValue != nil),
				YANGTypeName:          yangTypeName,
				YANGPath:              field.YANGDetails.Path,
				GoPathPackageName:     goPackageName(field.YANGDetails.RootElementModule, splitByModule, false, packageName, packageSuffix, trimPackagePrefix),
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
func writeHeader(yangFiles, includePaths []string, packageName string, cg *GenConfig, genCode *GeneratedPathCode) error {
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
		ExtraImports            []string // ExtraImports for path structs that are in a different package.
	}{
		GoImports:               cg.GoImports,
		PackageName:             packageName,
		GeneratingBinary:        cg.GeneratingBinary,
		YANGFiles:               yangFiles,
		IncludePaths:            includePaths,
		SchemaStructPkgAlias:    schemaStructPkgAlias,
		PathBaseTypeName:        ygot.PathBaseTypeName,
		PathStructInterfaceName: ygot.PathStructInterfaceName,
		FakeRootTypeName:        yang.CamelCase(cg.FakeRootName),
	}
	// Create an ordered list of imports to include in the header.
	for dep := range genCode.Deps {
		s.ExtraImports = append(s.ExtraImports, fmt.Sprintf("%s/%s", cg.BaseImportPath, dep))
	}
	sort.Slice(s.ExtraImports, func(i, j int) bool { return s.ExtraImports[i] < s.ExtraImports[j] })

	var common strings.Builder
	if err := goPathCommonHeaderTemplate.Execute(&common, s); err != nil {
		return err
	}

	genCode.CommonHeader = common.String()
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
	// FakeRootBaseTypeName is the type name of the fake root struct which
	// should be embedded within the fake root path struct.
	FakeRootBaseTypeName string
	// WildcardSuffix is the suffix given to the wildcard versions of
	// each node that distinguishes each from its non-wildcard counterpart.
	WildcardSuffix string
	// GenerateWildcardPaths means to generate wildcard nodes and paths.
	GenerateWildcardPaths bool
}

// getStructData returns the goPathStructData corresponding to a
// ParsedDirectory, which is used to store the attributes of the template for
// which code is being generated.
func getStructData(directory *ygen.ParsedDirectory, pathStructSuffix string, generateWildcardPaths bool) goPathStructData {
	return goPathStructData{
		TypeName:                directory.Name + pathStructSuffix,
		YANGPath:                directory.Path,
		PathBaseTypeName:        ygot.PathBaseTypeName,
		FakeRootBaseTypeName:    ygot.FakeRootBaseTypeName,
		PathStructInterfaceName: ygot.PathStructInterfaceName,
		WildcardSuffix:          WildcardSuffix,
		GenerateWildcardPaths:   generateWildcardPaths,
	}
}

// goPathFieldData stores template information needed to generate a struct
// field's child constructor method.
type goPathFieldData struct {
	MethodName              string           // MethodName is the name of the method that can be called to get to this field.
	SchemaName              string           // SchemaName is the field's original name in the schema.
	TypeName                string           // TypeName is the type name of the returned struct.
	YANGNodeType            string           // YANGNodeType is the type of YANG node for the node (e.g. "list", "container", "leaf").
	YANGDescription         string           // YANGDescription is the description for the node from its YANG definition.
	DefiningModuleName      string           // DefiningModuleName is the defining module for the node.
	InstantiatingModuleName string           // InstantiatingModuleName is the module that instantiated the node.
	AbsPath                 string           // AbsPath is the full path from root (not including keys).
	RelPath                 string           // RelPath is the relative path from its containing struct.
	RelPathList             string           // RelPathList is the list of strings that form the relative path from its containing struct.
	Struct                  goPathStructData // Struct stores template information for the field's containing struct.
	KeyParamListStr         string           // KeyParamListStr is the parameter list of the field's accessor method.
	KeyEntriesStr           string           // KeyEntriesStr is an ordered list of comma-separated ("schemaName": unique camel-case name) for a list's keys.
	KeyParamDocStrs         []string         // KeyParamDocStrs is an ordered slice of docstrings documenting the types of each list key parameter.
	ChildPkgAccessor        string           // ChildPkgAccessor is used if the child path struct exists in another package.
}

// generateDirectorySnippet generates all Go code associated with a schema node
// (container, list, leaf, or fakeroot), all of which have a corresponding
// struct onto which to attach the necessary methods for path generation.
// When generating code for the fakeroot, several structs may be returned,
// one package containing the fake root struct and one package for each of the
// fake root's child lists that uses the builder API methods,
// since they must be defined in their respective child packages. Otherwise,
// the returned slice will only have a single struct containing the all the code.
// The code comprises of the type definition for the struct, and all accessors to
// the fields of the struct. directory is the parsed information of a schema
// node, and directories is a map from path to a parsed schema node for all
// directory nodes in the schema.
func generateDirectorySnippet(directory *ygen.ParsedDirectory, directories map[string]*ygen.ParsedDirectory, schemaStructPkgAccessor, pathStructSuffix string, listBuilderKeyThreshold uint,
	generateWildcardPaths, simplifyWildcardPaths, splitByModule bool, pkgName, pkgSuffix, trimPkgPrefix string) ([]GoPathStructCodeSnippet, util.Errors) {

	var errs util.Errors
	// structBuf is used to store the code associated with the struct defined for
	// the target YANG entity.
	var structBuf strings.Builder
	var methodBuf strings.Builder

	// Output struct snippets.
	structData := getStructData(directory, pathStructSuffix, generateWildcardPaths)
	if directory.IsFakeRoot {
		// Fakeroot has its unique output.
		if err := goPathFakeRootTemplate.Execute(&structBuf, structData); err != nil {
			return nil, util.AppendErr(errs, err)
		}
	} else if err := goPathStructTemplate.Execute(&structBuf, structData); err != nil {
		return nil, util.AppendErr(errs, err)
	}

	deps := map[string]bool{}
	listBuilderAPIBufs := map[string]*strings.Builder{}

	goFieldNameMap := ygen.GoFieldNameMap(directory)
	// Generate child constructor snippets for all fields of the node.
	// Alphabetically order fields to produce deterministic output.
	for _, fName := range directory.OrderedFieldNames() {
		field, ok := directory.Fields[fName]
		if !ok {
			errs = util.AppendErr(errs, fmt.Errorf("generateDirectorySnippet: field %s not found in directory %v", fName, directory))
			continue
		}
		goFieldName := goFieldNameMap[fName]
		var childPkgAccessor string

		// The most common case is that list builder API is written to same package as the rest of the child methods.
		buildBuf := &methodBuf

		// Only the fake root could be importing a child path struct from another package.
		// If it is, add that package as a dependency and set the accessor.
		if directory.IsFakeRoot && (field.Type == ygen.ContainerNode || field.Type == ygen.ListNode) {
			parentPackge := goPackageName(directory.RootElementModule, splitByModule, directory.IsFakeRoot, pkgName, pkgSuffix, trimPkgPrefix)
			childPackage := goPackageName(field.YANGDetails.RootElementModule, splitByModule, false, pkgName, pkgSuffix, trimPkgPrefix)
			if parentPackge != childPackage {
				deps[childPackage] = true
				childPkgAccessor = childPackage + "."
				// The fake root could be generating a list builder API for one of its children which is in another package.
				// Write any list builders into the map, keyed by the child package name.
				if _, ok := listBuilderAPIBufs[childPackage]; !ok {
					listBuilderAPIBufs[childPackage] = &strings.Builder{}
				}
				buildBuf = listBuilderAPIBufs[childPackage]
			}
		}

		if es := generateChildConstructors(&methodBuf, buildBuf, directory, fName, goFieldName, directories, schemaStructPkgAccessor, pathStructSuffix, listBuilderKeyThreshold, generateWildcardPaths, simplifyWildcardPaths, childPkgAccessor); es != nil {
			errs = util.AppendErrs(errs, es)
		}

		// Since leaves don't have their own Directory entries, we need
		// to output their struct snippets somewhere, and here is
		// convenient.
		if field.Type == ygen.LeafNode || field.Type == ygen.LeafListNode {
			leafTypeName, err := getFieldTypeName(directory, fName, goFieldName, directories, pathStructSuffix)
			if err != nil {
				errs = util.AppendErr(errs, err)
			} else {
				structData := goPathStructData{
					TypeName:                leafTypeName,
					YANGPath:                field.YANGDetails.Path,
					PathBaseTypeName:        ygot.PathBaseTypeName,
					PathStructInterfaceName: ygot.PathStructInterfaceName,
					WildcardSuffix:          WildcardSuffix,
					GenerateWildcardPaths:   generateWildcardPaths,
				}
				if err := goPathStructTemplate.Execute(&structBuf, structData); err != nil {
					errs = util.AppendErr(errs, err)
				}
			}
		}
	}

	if len(errs) == 0 {
		errs = nil
	}
	snippet := GoPathStructCodeSnippet{
		PathStructName:    structData.TypeName,
		StructBase:        structBuf.String(),
		ChildConstructors: methodBuf.String(),
		Package:           goPackageName(directory.RootElementModule, splitByModule, directory.IsFakeRoot, pkgName, pkgSuffix, trimPkgPrefix),
	}
	for dep := range deps {
		snippet.Deps = append(snippet.Deps, dep)
	}
	snippets := []GoPathStructCodeSnippet{snippet}

	for pkg, build := range listBuilderAPIBufs {
		if code := build.String(); code != "" {
			snippets = append(snippets, GoPathStructCodeSnippet{
				PathStructName:    structData.TypeName,
				Package:           pkg,
				ChildConstructors: code,
			})
		}
	}
	return snippets, errs
}

// generateChildConstructors generates and writes to methodBuf the Go methods
// that returns an instantiation of the child node's path struct object.
// When this is called on the fakeroot, the list builder API's methods
// need to be in the child package as opposed to the fakeroot's package.
// In all other cases, methodBuf and builderBuf can point to the same buffer.
// The func takes as input the buffers to store the method, a directory, the field name
// of the directory identifying the child yang.Entry, a directory-level unique
// field name to be used as the generated method's name and the incremental
// type name of of the child path struct, and a map of all directories of the
// whole schema keyed by their schema paths.
func generateChildConstructors(methodBuf *strings.Builder, builderBuf *strings.Builder, directory *ygen.ParsedDirectory, directoryFieldName string, goFieldName string, directories map[string]*ygen.ParsedDirectory, schemaStructPkgAccessor, pathStructSuffix string, listBuilderKeyThreshold uint, generateWildcardPaths, simplifyWildcardPaths bool, childPkgAccessor string) []error {
	field, ok := directory.Fields[directoryFieldName]
	if !ok {
		return []error{fmt.Errorf("generateChildConstructors: field %s not found in directory %v", directoryFieldName, directory)}
	}
	fieldTypeName, err := getFieldTypeName(directory, directoryFieldName, goFieldName, directories, pathStructSuffix)
	if err != nil {
		return []error{err}
	}

	longestPath := func(ss [][]string) []string {
		var longest []string
		for _, s := range ss {
			if longest == nil {
				longest = s
				continue
			}
			if len(s) > len(longest) {
				longest = s
			}
		}
		return longest
	}

	structData := getStructData(directory, pathStructSuffix, generateWildcardPaths)
	// The longest path is the non-key path. This is the one we want to use
	// since the key is "compressed out".
	relPath := longestPath(field.MappedPaths)
	// Be nil-tolerant for these two attributes. In real deployments (i.e.
	// not tests), these should be populated. Since these are just use for
	// documentation, it is not critical that they are populated.
	fieldData := goPathFieldData{
		MethodName:              goFieldName,
		TypeName:                fieldTypeName,
		SchemaName:              field.Name,
		YANGNodeType:            field.Type.String(),
		YANGDescription:         strings.ReplaceAll(field.YANGDetails.Description, "\n", "\n// "),
		DefiningModuleName:      field.YANGDetails.DefiningModule,
		InstantiatingModuleName: field.YANGDetails.RootElementModule,
		AbsPath:                 field.YANGDetails.SchemaPath,
		Struct:                  structData,
		RelPath:                 strings.Join(relPath, `/`),
		RelPathList:             `"` + strings.Join(relPath, `", "`) + `"`,
		ChildPkgAccessor:        childPkgAccessor,
	}

	isUnderFakeRoot := directory.IsFakeRoot

	// This is expected to be nil for leaf fields.
	fieldDirectory := directories[field.YANGDetails.Path]

	switch {
	case field.Type != ygen.ListNode:
		return generateChildConstructorsForLeafOrContainer(methodBuf, fieldData, isUnderFakeRoot, generateWildcardPaths)
	case len(fieldDirectory.ListKeys) == 0:
		// TODO(wenbli): keyless lists as a path are not supported by gNMI, but this
		// library is currently intended for gNMI, so need to decide on a long-term solution.

		// As a short-term solution, we just need to prevent the user from accessing any node in the keyless list's subtree.
		// Here, we simply skip generating the child constructor, such that its subtree is unreachable.
		return nil
		// Erroring out, on the other hand, is impractical due to their existence in the current OpenConfig models.
		// return fmt.Errorf("generateChildConstructors: schemas containing keyless lists are unsupported, path: %s", field.Path())
	case listBuilderKeyThreshold != 0 && uint(len(fieldDirectory.ListKeys)) >= listBuilderKeyThreshold:
		// If the number of keys is equal to or over the builder API threshold,
		// then use the builder API format to make the list path API less
		// confusing for the user.
		// The generated const
		return generateChildConstructorsForListBuilderFormat(methodBuf, builderBuf, fieldDirectory.ListKeys, fieldDirectory.ListKeyYANGNames, fieldData, isUnderFakeRoot, schemaStructPkgAccessor)
	default:
		return generateChildConstructorsForList(methodBuf, fieldDirectory.ListKeys, fieldDirectory.ListKeyYANGNames, fieldData, isUnderFakeRoot, generateWildcardPaths, simplifyWildcardPaths, schemaStructPkgAccessor)
	}
}

// generateChildConstructorsForLeafOrContainer writes into methodBuf the child
// constructor snippets for the container or leaf template output information
// contained in fieldData.
func generateChildConstructorsForLeafOrContainer(methodBuf *strings.Builder, fieldData goPathFieldData, isUnderFakeRoot, generateWildcardPaths bool) []error {
	// Generate child constructor for the non-wildcard version of the parent struct.
	var errors []error
	if err := goPathChildConstructorTemplate.Execute(methodBuf, fieldData); err != nil {
		errors = append(errors, err)
	}

	// The root node doesn't have a wildcard version of itself.
	if isUnderFakeRoot {
		return errors
	}

	if generateWildcardPaths {
		// Generate child constructor for the wildcard version of the parent struct.
		fieldData.TypeName += WildcardSuffix
		fieldData.Struct.TypeName += WildcardSuffix
		if err := goPathChildConstructorTemplate.Execute(methodBuf, fieldData); err != nil {
			errors = append(errors, err)
		}
	}
	return errors
}

// generateChildConstructorsForListBuilderFormat writes into methodBuf the
// child constructor method snippets for the list keys. The builder API helpers
// are written into the builderBuf, this allows the helpers to be written to
// package the child belongs to, not the parent if the child belongs in its own
// package. fieldData contains the childConstructor template output information
// for if the node were a container (which contains a subset of the basic
// information required for the list constructor methods).
func generateChildConstructorsForListBuilderFormat(methodBuf *strings.Builder, builderBuf *strings.Builder, keys map[string]*ygen.ListKey, keyNames []string, fieldData goPathFieldData, isUnderFakeRoot bool, schemaStructPkgAccessor string) []error {
	var errors []error
	// List of function parameters as would appear in the method definition.
	keyParams, err := makeKeyParams(keys, keyNames, schemaStructPkgAccessor)
	if err != nil {
		return append(errors, err)
	}
	keyN := len(keyParams)

	// Initialize ygot.NodePath's key list with wildcard values.
	var keyEntryStrs []string
	for i := 0; i != keyN; i++ {
		keyEntryStrs = append(keyEntryStrs, fmt.Sprintf(`"%s": "*"`, keyParams[i].name))
	}
	fieldData.KeyEntriesStr = strings.Join(keyEntryStrs, ", ")

	// There are no initial key parameters for the builder API.
	fieldData.KeyParamListStr = ""

	// Set the child type to be the wildcard version.
	fieldData.TypeName += WildcardSuffix

	// Add Builder suffix to the child constructor method name.
	fieldData.MethodName += BuilderCtorSuffix

	// Generate builder constructor method for non-wildcard version of parent struct.
	if err := goPathChildConstructorTemplate.Execute(methodBuf, fieldData); err != nil {
		errors = append(errors, err)
	}

	// The root node doesn't have a wildcard version of itself.
	if !isUnderFakeRoot {
		// Generate builder constructor method for wildcard version of parent struct.
		fieldData.Struct.TypeName += WildcardSuffix
		if err := goPathChildConstructorTemplate.Execute(methodBuf, fieldData); err != nil {
			errors = append(errors, err)
		}
	}

	// Generate key-builder methods for the wildcard version of the PathStruct.
	// Although non-wildcard PathStruct is unnecessary, it is kept for generation simplicity.
	for i := 0; i != keyN; i++ {
		if err := goKeyBuilderTemplate.Execute(builderBuf,
			struct {
				MethodName     string
				TypeName       string
				KeySchemaName  string
				KeyParamType   string
				KeyParamName   string
				KeyParamDocStr string
			}{
				MethodName:     BuilderKeyPrefix + keyParams[i].varName,
				TypeName:       fieldData.TypeName,
				KeySchemaName:  keyParams[i].name,
				KeyParamName:   keyParams[i].varName,
				KeyParamType:   keyParams[i].typeName,
				KeyParamDocStr: keyParams[i].varName + ": " + keyParams[i].typeDocString,
			}); err != nil {
			errors = append(errors, err)
		}
	}

	return errors
}

// generateChildConstructorsForList writes into methodBuf the child constructor
// method snippets for the list represented by the list keys. fieldData contains the
// childConstructor template output information for if the node were a
// container (which contains a subset of the basic information required for
// the list constructor methods).
func generateChildConstructorsForList(methodBuf *strings.Builder, keys map[string]*ygen.ListKey, keyNames []string, fieldData goPathFieldData, isUnderFakeRoot, generateWildcardPaths, simplifyWildcardPaths bool, schemaStructPkgAccessor string) []error {
	var errors []error
	// List of function parameters as would appear in the method definition.
	keyParams, err := makeKeyParams(keys, keyNames, schemaStructPkgAccessor)
	if err != nil {
		return append(errors, err)
	}
	keyN := len(keyParams)
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
		if !generateWildcardPaths && comboIndex != len(combos)-1 {
			// All but the last combo contain wildcard paths.
			continue
		}
		var paramListStrs, paramDocStrs, keyEntryStrs []string
		var anySuffixes []string

		i := 0 // Loop through each parameter
		for _, paramIndex := range combo {
			// Add unselected parameters as a wildcard.
			for ; i != paramIndex; i++ {
				param := keyParams[i]
				paramDocStrs = append(paramDocStrs, param.varName+" (wildcarded): "+param.typeDocString)
				keyEntryStrs = append(keyEntryStrs, fmt.Sprintf(`"%s": "*"`, param.name))
				anySuffixes = append(anySuffixes, WildcardSuffix+param.varName)
			}
			// Add selected parameters to the parameter list.
			param := keyParams[paramIndex]
			paramDocStrs = append(paramDocStrs, param.varName+": "+param.typeDocString)
			paramListStrs = append(paramListStrs, fmt.Sprintf("%s %s", param.varName, param.typeName))
			keyEntryStrs = append(keyEntryStrs, fmt.Sprintf(`"%s": %s`, param.name, param.varName))
			i++
		}
		for ; i != keyN; i++ { // Handle edge case
			param := keyParams[i]
			paramDocStrs = append(paramDocStrs, param.varName+" (wildcarded): "+param.typeDocString)
			keyEntryStrs = append(keyEntryStrs, fmt.Sprintf(`"%s": "*"`, param.name))
			anySuffixes = append(anySuffixes, WildcardSuffix+param.varName)
		}
		// Create the string for the method parameter list, docstrings, and ygot.NodePath's key list.
		fieldData.KeyParamListStr = strings.Join(paramListStrs, ", ")
		fieldData.KeyParamDocStrs = paramDocStrs
		fieldData.KeyEntriesStr = strings.Join(keyEntryStrs, ", ")
		if simplifyWildcardPaths && comboIndex == 0 {
			// The zeroth index has every key as a wildcard, so
			// we can equivalently omit specifying any key values
			// per the gNMI spec if the user prefers this
			// alternative simplified format.
			fieldData.KeyEntriesStr = ""
		}

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
		if err := goPathChildConstructorTemplate.Execute(methodBuf, fieldData); err != nil {
			errors = append(errors, err)
		}

		// The root node doesn't have a wildcard version of itself.
		if isUnderFakeRoot {
			continue
		}

		if generateWildcardPaths {
			// Generate child constructor method for wildcard version of parent struct.
			fieldData.Struct.TypeName = wildcardParentTypeName
			// Override the corner case for generating the non-wildcard child.
			fieldData.TypeName = wildcardFieldTypeName
			if err := goPathChildConstructorTemplate.Execute(methodBuf, fieldData); err != nil {
				errors = append(errors, err)
			}
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
func getFieldTypeName(directory *ygen.ParsedDirectory, directoryFieldName string, goFieldName string, directories map[string]*ygen.ParsedDirectory, pathStructSuffix string) (string, error) {
	field, ok := directory.Fields[directoryFieldName]
	if !ok {
		return "", fmt.Errorf("getFieldTypeName: field %s not found in directory %v", directoryFieldName, directory)
	}

	switch field.Type {
	case ygen.ContainerNode, ygen.ListNode:
		fieldDirectory, ok := directories[field.YANGDetails.Path]
		if !ok {
			return "", fmt.Errorf("getFieldTypeName: unexpected - field with path %q not found in parsed yang structs map: %v", field.YANGDetails.Path, directories)
		}
		return fieldDirectory.Name + pathStructSuffix, nil
	// Leaves do not have corresponding Directory entries, so their names need to be constructed.
	default:
		if directory.IsFakeRoot {
			// When a leaf resides at the root, its type name is its whole name -- we never want fakeroot's name as a prefix.
			return goFieldName + pathStructSuffix, nil
		}
		return directory.Name + "_" + goFieldName + pathStructSuffix, nil
	}
}

type keyParam struct {
	name          string
	varName       string
	typeName      string
	typeDocString string
}

// makeKeyParams generates the list of go parameter list components for a child
// list's constructor method given the list's key spec, as well as a
// list of each parameter's types as a comment string.
// It outputs the parameters in the same order as in the given keyNames.
// e.g.
//
//	  in: &map[string]*ygen.ListKey{
//			"fluorine": &ygen.ListKey{
//				Name: "Fluorine", LangType: &ygen.MappedType{NativeType: "string"}
//			},
//			"iodine-liquid": &ygen.ListKey{
//				Name: "IodineLiquid", LangType: &ygen.MappedType{NativeType: "A_Union", UnionTypes: {"Binary": 0, "uint64": 1}}
//			},
//	      }
//	      KeyNames: []string{"fluorine", "iodine-liquid"},
//
//	  {name, varName, typeName} out: [{"fluroine", "Fluorine", "string"}, {"iodine-liquid", "IodineLiquid", "oc.A_Union"}]
//	  docstring out: ["string", "[oc.Binary, oc.UnionUint64]"]
func makeKeyParams(keys map[string]*ygen.ListKey, keyNames []string, schemaStructPkgAccessor string) ([]keyParam, error) {
	if len(keys) == 0 {
		return nil, fmt.Errorf("makeKeyParams: invalid list - has no key; cannot process param list string")
	}

	// Create parameter list *in order* of keys, which should be in schema order.
	var keyParams []keyParam
	// NOTE: Although the generated key names might not match their
	// corresponding ygen field names in case of a camelcase name
	// collision, we expect that the user is aware of the schema to know
	// the order of the keys, and not rely on the naming in that case.
	goKeyNameMap, err := getGoKeyNameMap(keys, keyNames)
	if err != nil {
		return nil, err
	}
	for _, keyName := range keyNames {
		listKey, ok := keys[keyName]
		switch {
		case !ok:
			return nil, fmt.Errorf("makeKeyParams: inconsistent IR: key doesn't exist: %s", keyName)
		case listKey.LangType == nil:
			return nil, fmt.Errorf("makeKeyParams: mappedType for key is nil: %s", keyName)
		}

		var typeName string
		switch {
		case listKey.LangType.NativeType == "interface{}": // ygen-unsupported types
			typeName = "string"
		case ygen.IsYgenDefinedGoType(listKey.LangType):
			typeName = schemaStructPkgAccessor + listKey.LangType.NativeType
		default:
			typeName = listKey.LangType.NativeType
		}
		varName := goKeyNameMap[keyName]

		typeDocString := typeName
		if len(listKey.LangType.UnionTypes) > 1 {
			var genTypes []string
			for _, name := range listKey.LangType.OrderedUnionTypes() {
				unionTypeName := name
				if simpleName, ok := ygot.SimpleUnionBuiltinGoTypes[name]; ok {
					unionTypeName = simpleName
				}
				// Add schemaStructPkgAccessor.
				if strings.HasPrefix(unionTypeName, "*") {
					unionTypeName = "*" + schemaStructPkgAccessor + unionTypeName[1:]
				} else {
					unionTypeName = schemaStructPkgAccessor + unionTypeName
				}
				genTypes = append(genTypes, unionTypeName)
			}
			// Create the subtype documentation string.
			typeDocString = "[" + strings.Join(genTypes, ", ") + "]"
		}

		keyParams = append(keyParams, keyParam{
			name:          keyName,
			varName:       varName,
			typeName:      typeName,
			typeDocString: typeDocString,
		})
	}
	return keyParams, nil
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
// given a list of key entries and their order. Names are camelcased and
// uniquified to ensure compilation. Uniqification is done deterministically.
func getGoKeyNameMap(keys map[string]*ygen.ListKey, keyNames []string) (map[string]string, error) {
	goKeyNameMap := make(map[string]string, len(keyNames))

	usedKeyNames := map[string]bool{}
	for _, name := range keyNames {
		key, ok := keys[name]
		if !ok {
			return nil, fmt.Errorf("key %q doesn't exist in key map %v", name, keys)
		}
		goKeyNameMap[name] = genutil.MakeNameUnique(key.Name, usedKeyNames)
	}
	return goKeyNameMap, nil
}
