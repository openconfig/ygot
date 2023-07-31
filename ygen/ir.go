// Copyright 2020 Google Inc.
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

	gpb "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/genutil"
	"github.com/openconfig/ygot/yangschema"
	"github.com/openconfig/ygot/ygot"
)

// This file describes the intermediate representation that is produced by
// the ygen generator. The design of this IR is described in detail in the
// docs/code-generation-design.md directory in further detail.
//
// In addition to the IR, it describes the "LangMapper" interface which is
// to be implemented by a language specific code package. This interface
// allows the code output parts of the toolchain to be simplified by
// encapsulating the naming and designation of language-specific types for
// the output language.

// LangMapper is the interface to be implemented by a language-specific
// library and provided as an input to the IR production phase of ygen.
//
// Note: though the output names are meant to be usable within the output
// language, it may not be the final name used in the generated code, for
// example due to naming conflicts, which are better resolved in a later
// pass prior to code generation (see note below).
//
// NB: LangMapper's methods should be idempotent, such that the order in
// which they're called and the number of times each is called per input
// parameter does not affect the output. Do not depend on the same order of
// method calls on langMapper by GenerateIR.
type LangMapper interface {
	// LangMapperBaseSetup defines setup methods that are required for all
	// LangMapper instances.
	LangMapperBaseSetup

	// FieldName maps an input yang.Entry to the name that should be used
	// in the intermediate representation. It is called for each field of
	// a defined directory.
	FieldName(e *yang.Entry) (string, error)

	// DirectoryName maps an input yang.Entry to the name that should be used in the
	// intermediate representation (IR). It is called for any directory entity that
	// is to be output in the generated code.
	DirectoryName(*yang.Entry, genutil.CompressBehaviour) (string, error)

	// KeyLeafType maps an input yang.Entry which must represent a leaf to the
	// type that should be used when the leaf is used in the context of a
	// list key within the output IR.
	KeyLeafType(*yang.Entry, IROptions) (*MappedType, error)

	// LeafType maps an input yang.Entry which must represent a leaf to the
	// type that should be used when the leaf is used in the context of a
	// field within a directory within the output IR.
	LeafType(*yang.Entry, IROptions) (*MappedType, error)

	// PackageName maps an input yang.Entry, which must correspond to a
	// directory type (container or list), to the package name to which it
	// belongs. The bool parameter specifies whether the generated
	// directories will be nested or not since some languages allow nested
	// structs.
	PackageName(*yang.Entry, genutil.CompressBehaviour, bool) (string, error)

	// LangMapperExt contains extensions that the LangMapper instance
	// should implement if extra information from the IR is required.
	// When implementing this, UnimplementedLangMapperExt should be
	// embedded in the implementation type in order to ensure forward
	// compatibility.
	LangMapperExt
}

// LangMapperBaseSetup defines setup methods that are required for all
// LangMapper instances.
type LangMapperBaseSetup interface {
	// setEnumSet is used to supply a set of enumerated values to the
	// mapper such that leaves that have enumerated types can be looked up.
	// An enumSet provides lookup methods that allow:
	//  - simple enumerated types
	//  - identityrefs
	//  - enumerations within typedefs
	//  - identityrefs within typedefs
	// to be resolved to the corresponding type that is to be used in
	// the IR.
	setEnumSet(*enumSet)

	// setSchemaTree is used to supply a copy of the YANG schema tree to
	// the mapped such that leaves of type leafref can be resolved to
	// their target leaves.
	setSchemaTree(*yangschema.Tree)

	// InjectEnumSet is intended to be called by unit tests in order to set up the
	// LangMapperBase such that generated enumeration/identity names can be looked
	// up. The input parameters correspond to fields in IROptions.
	// It returns an error if there is a failure to generate the enumerated
	// values' names.
	InjectEnumSet(entries map[string]*yang.Entry, compressPaths, noUnderscores, skipEnumDedup, shortenEnumLeafNames, useDefiningModuleForTypedefEnumNames, appendEnumSuffixForSimpleUnionEnums bool, enumOrgPrefixesToTrim []string) error

	// InjectSchemaTree is intended to be called by unit tests in order to set up
	// the LangMapperBase such that leafrefs targets may be looked up.
	// It returns an error if there is duplication within the set of entries.
	InjectSchemaTree(entries []*yang.Entry) error
}

// LangMapperBase contains unexported base types and exported built-in methods
// that all LangMapper implementation instances should embed. These built-in
// methods are available for use anywhere in the LangMapper implementation
// instance.
type LangMapperBase struct {
	// enumSet contains the generated enum names which can be queried.
	enumSet *enumSet

	// schematree is a copy of the YANG schema tree, containing only leaf
	// entries, such that schema paths can be referenced.
	schematree *yangschema.Tree
}

// setEnumSet is used to supply a set of enumerated values to the
// mapper such that leaves that have enumerated types can be looked up.
//
// NB: This method is a set-up method that GenerateIR automatically invokes.
// In testing contexts outside of GenerateIR, however, the corresponding
// exported Inject method needs to be called in order for certain built-in
// methods of LangMapperBase to be available for use.
func (s *LangMapperBase) setEnumSet(e *enumSet) {
	s.enumSet = e
}

// setSchemaTree is used to supply a copy of the YANG schema tree to
// the mapped such that leaves of type leafref can be resolved to
// their target leaves.
//
// NB: This method is a set-up method that GenerateIR automatically invokes.
// In testing contexts outside of GenerateIR, however, the corresponding
// exported Inject method needs to be called in order for certain built-in
// methods of LangMapperBase to be available for use.
func (s *LangMapperBase) setSchemaTree(st *yangschema.Tree) {
	s.schematree = st
}

// InjectEnumSet is intended to be called by unit tests in order to set up the
// LangMapperBase such that generated enumeration/identity names can be looked
// up. It walks the input map of enumerated value leaves keyed by path and
// creates generates names for them. The input parameters correspond to fields
// in IROptions.
// It returns an error if there is a failure to generate the enumerated values'
// names.
func (s *LangMapperBase) InjectEnumSet(entries map[string]*yang.Entry, compressPaths, noUnderscores, skipEnumDedup, shortenEnumLeafNames, useDefiningModuleForTypedefEnumNames, appendEnumSuffixForSimpleUnionEnums bool, enumOrgPrefixesToTrim []string) error {
	enumSet, _, errs := findEnumSet(entries, compressPaths, noUnderscores, skipEnumDedup, shortenEnumLeafNames, useDefiningModuleForTypedefEnumNames, appendEnumSuffixForSimpleUnionEnums, enumOrgPrefixesToTrim)
	if errs != nil {
		return fmt.Errorf("%v", errs)
	}
	s.setEnumSet(enumSet)
	return nil
}

// InjectSchemaTree is intended to be called by unit tests in order to set up
// the LangMapperBase such that leafrefs targets may be looked up. It maps a
// set of yang.Entry pointers into a ctree structure.
// It returns an error if there is duplication within the set of entries.
func (s *LangMapperBase) InjectSchemaTree(entries []*yang.Entry) error {
	schematree, err := yangschema.BuildTree(entries)
	if err != nil {
		return err
	}
	s.setSchemaTree(schematree)
	return nil
}

// ResolveLeafrefTarget takes an input path and context entry and
// determines the type of the leaf that is referred to by the path, such that
// it can be mapped to a native language type. It returns the yang.YangType that
// is associated with the target, and the target yang.Entry, such that the
// caller can map this to the relevant language type.
//
// In testing contexts, this function requires InjectSchemaTree to be called
// prior to being usable.
func (b *LangMapperBase) ResolveLeafrefTarget(path string, contextEntry *yang.Entry) (*yang.Entry, error) {
	return b.schematree.ResolveLeafrefTarget(path, contextEntry)
}

// EnumeratedTypedefTypeName retrieves the name of an enumerated typedef (i.e.,
// a typedef which is either an identityref or an enumeration). The resolved
// name is prefixed with the prefix supplied. If the type that was supplied
// within the resolveTypeArgs struct is not a type definition which includes an
// enumerated type, the third returned value (boolean) will be false.
// The second value returned is a string key that uniquely identifies this
// enumerated value among all possible enumerated values in the input set of
// YANG files.
//
// In testing contexts, this function requires InjectEnumSet to be called prior
// to being usable.
func (b *LangMapperBase) EnumeratedTypedefTypeName(yangType *yang.YangType, contextEntry *yang.Entry, prefix string, noUnderscores, useDefiningModuleForTypedefEnumNames bool) (string, string, bool, error) {
	return b.enumSet.enumeratedTypedefTypeName(resolveTypeArgs{yangType: yangType, contextEntry: contextEntry}, prefix, noUnderscores, useDefiningModuleForTypedefEnumNames)
}

// EnumName retrieves the type name of the input enum *yang.Entry that will be
// used in the generated code, which is the first returned value. The second
// value returned is a string key that uniquely identifies this enumerated
// value among all possible enumerated values in the input set of YANG files.
//
// In testing contexts, this function requires InjectEnumSet to be called prior
// to being usable.
func (b *LangMapperBase) EnumName(e *yang.Entry, compressPaths, noUnderscores, skipDedup, shortenEnumLeafNames, addEnumeratedUnionSuffix bool, enumOrgPrefixesToTrim []string) (string, string, error) {
	return b.enumSet.enumName(e, compressPaths, noUnderscores, skipDedup, shortenEnumLeafNames, addEnumeratedUnionSuffix, enumOrgPrefixesToTrim)
}

// IdentityrefBaseTypeFromIdentity retrieves the generated type name of the
// input *yang.Identity. The first value returned is the defining module
// followed by the CamelCase-ified version of the identity's name. The second
// value returned is a string key that uniquely identifies this enumerated
// value among all possible enumerated values in the input set of YANG files.
//
// In testing contexts, this function requires InjectEnumSet to be called prior
// to being usable.
func (b *LangMapperBase) IdentityrefBaseTypeFromIdentity(i *yang.Identity) (string, string, error) {
	return b.enumSet.identityrefBaseTypeFromIdentity(i)
}

// IdentityrefBaseTypeFromLeaf retrieves the mapped name of an identityref's
// base such that it can be used in generated code. The first value returned is
// the defining module name followed by the CamelCase-ified version of the
// base's name. The second value returned is a string key that uniquely
// identifies this enumerated value among all possible enumerated values in the
// input set of YANG files.
// This function wraps the identityrefBaseTypeFromIdentity function since it
// covers the common case that the caller is interested in determining the name
// from an identityref leaf, rather than directly from the identity.
//
// In testing contexts, this function requires InjectEnumSet to be called prior
// to being usable.
func (b *LangMapperBase) IdentityrefBaseTypeFromLeaf(idr *yang.Entry) (string, string, error) {
	return b.enumSet.identityrefBaseTypeFromIdentity(idr.Type.IdentityBase)
}

// LangMapperExt contains extensions that the LangMapper instance should
// implement if extra information from the IR is required. These flag values
// are expected to contain information rarely used but needed from goyang's
// AST. Values that are expected to be used more often should be placed in the
// IR itself so that other users can get access to the same information without
// implementing it themselves.
type LangMapperExt interface {
	// PopulateFieldFlags populates extra information given a particular
	// field of a ParsedDirectory and the corresponding AST node.
	// Fields of a ParsedDirectory can be any non-choice/case node (e.g.
	// YANG leafs, containers, lists).
	PopulateFieldFlags(NodeDetails, *yang.Entry) map[string]string
	// PopulateEnumFlags populates extra information given a particular
	// enumerated type its corresponding AST representation.
	PopulateEnumFlags(EnumeratedYANGType, *yang.YangType) map[string]string
}

// UnimplementedLangMapperExt should be embedded to have forward compatible
// implementations.
type UnimplementedLangMapperExt struct {
}

// PopulateFieldFlags populates extra information given a particular
// field of a ParsedDirectory and the corresponding AST node.
func (UnimplementedLangMapperExt) PopulateFieldFlags(NodeDetails, *yang.Entry) map[string]string {
	return nil
}

// PopulateEnumFlags populates extra information given a particular
// enumerated type its corresponding AST representation.
func (UnimplementedLangMapperExt) PopulateEnumFlags(EnumeratedYANGType, *yang.YangType) map[string]string {
	return nil
}

// IR represents the returned intermediate representation produced by ygen to
// be consumed by language-specific passes prior to code generation.
type IR struct {
	// Directories is the set of 'directory', or non-leaf entries that are
	// to be produced in the generated code. They are keyed by the absolute
	// YANG path of their locations.
	Directories map[string]*ParsedDirectory

	// Enums is the set of enumerated entries that are to be output in the
	// generated language code. They are each keyed by a name that
	// uniquely identifies the enumeration. Note that this name may not be
	// the same type name that would be used in the generated code due to
	// inner definitions.
	Enums map[string]*EnumeratedYANGType

	// ModelData stores the metadata extracted from the input YANG modules.
	ModelData []*gpb.ModelData

	// opts stores the IROptions that were used to generate the IR.
	opts IROptions

	// parsedModules stores the list of YANG entries for creating a
	// serialized version of the AST if needed.
	parsedModules []*yang.Entry

	// fakeroot stores the fake root's AST node for creating a serialized
	// version of the AST if needed.
	fakeroot *yang.Entry
}

// OrderedDirectoryPaths returns the absolute YANG paths of all ParsedDirectory
// entries in the IR in lexicographical order.
func (ir *IR) OrderedDirectoryPaths() []string {
	if ir == nil {
		return nil
	}

	paths := make([]string, 0, len(ir.Directories))
	for path := range ir.Directories {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

// OrderedDirectoryPathsByName returns the absolute YANG paths of all ParsedDirectory
// entries in the IR in the lexicographical order of their candidate generated
// names. Where there are duplicate names the path is used to tie-break.
func (ir *IR) OrderedDirectoryPathsByName() []string {
	if ir == nil {
		return nil
	}

	paths := make([]string, 0, len(ir.Directories))
	for path := range ir.Directories {
		paths = append(paths, path)
	}
	sort.Slice(paths, func(i, j int) bool {
		switch {
		case ir.Directories[paths[i]].Name == ir.Directories[paths[j]].Name:
			return paths[i] < paths[j]
		default:
			return ir.Directories[paths[i]].Name < ir.Directories[paths[j]].Name
		}
	})

	return paths
}

// SchemaTree returns a JSON serialised tree of the schema for the set of
// modules used to generate the IR. The JSON document that is returned is
// always rooted on a yang.Entry which corresponds to the root item, and stores
// all root-level enties (and their subtrees) within the input module set. All
// YANG directories are annotated in the output JSON with the name of the type
// they correspond to in the generated code, and the absolute schema path that
// the entry corresponds to. In the case that there is not a fake root struct,
// a synthetic root entry is used to store the schema tree.
func (ir *IR) SchemaTree(inclDescriptions bool) ([]byte, error) {
	dirNames := make(map[string]string, len(ir.Directories))
	for p, d := range ir.Directories {
		dirNames[p] = d.Name
	}
	rawSchema, err := buildJSONTree(ir.parsedModules, dirNames, ir.fakeroot, ir.opts.TransformationOptions.CompressBehaviour.CompressEnabled(), inclDescriptions)
	if err != nil {
		return nil, err
	}
	return rawSchema, nil
}

// ParsedDirectory describes an internal node within the generated
// code. Such a 'directory' may represent a struct, or a message,
// in the generated code. It represents a YANG 'container' or 'list'.
type ParsedDirectory struct {
	// Name is the candidate language-specific name of the directory.
	Name string
	// Type describes the type of directory that is being produced -
	// such that YANG 'list' entries can have special handling.
	Type DirType
	// Path specifies the absolute YANG schema path of the node.
	Path string
	// SchemaPath specifies the absolute YANG schema node path. It does not
	// include the module name nor choice/case elements in the YANG file.
	SchemaPath string
	// Fields is the set of direct children of the node that are to be
	// output. It is keyed by the YANG node identifier of the child field
	// since there could be name conflicts at this processing stage.
	Fields map[string]*NodeDetails
	// ListKeys describes the leaves of a YANG list that
	// are required in the output code (e.g., the characteristics
	// of the list's keys). It is keyed by the YANG name of the list key.
	ListKeys map[string]*ListKey
	// ListKeyYANGNames is the ordered list of YANG names specified in the
	// YANG list per Section 7.8.2 of RFC6020. The consumer of the IR can
	// rely on this ordering for deterministic ordering in output code and
	// rendering.
	ListKeyYANGNames []string
	// PackageName is the package in which this directory node's generated
	// code should reside.
	PackageName string
	// IsFakeRoot indicates whether the directory being described
	// is the root entity and has been synthetically generated by
	// ygen.
	IsFakeRoot bool
	// BelongingModule is the name of the module having the same XML
	// namespace as this directory node.
	// For more information on YANG's XML namespaces see
	// https://datatracker.ietf.org/doc/html/rfc7950#section-5.3
	BelongingModule string
	// RootElementModule is the module in which the root of the YANG tree that the
	// node is attached to was instantiated (rather than the module that
	// has the same namespace as the node).
	//
	// In this example, container 'con' has
	// RootElementModule: "openconfig-simple"
	// BelongingModule:   "openconfig-augment"
	// DefiningModule:    "openconfig-grouping"
	//
	//   module openconfig-augment {
	//     import openconfig-simple { prefix "s"; }
	//     import openconfig-grouping { prefix "g"; }
	//
	//     augment "/s:parent/child/state" {
	//       uses g:group;
	//     }
	//   }
	//
	//   module openconfig-grouping {
	//     grouping group {
	//       container con {
	//         leaf zero { type string; }
	//       }
	//     }
	//   }
	RootElementModule string
	// DefiningModule is the module that contains the text definition of
	// the field.
	DefiningModule string
	// ConfigFalse represents whether the node is state data as opposed to
	// configuration data.
	// The meaning of "config" is exactly the same as the "config"
	// statement in YANG:
	// https://datatracker.ietf.org/doc/html/rfc7950#section-7.21.1
	ConfigFalse bool
	// TelemetryAtomic indicates that the node has been modified with the
	// OpenConfig extension "telemetry-atomic".
	// https://github.com/openconfig/public/blob/master/release/models/openconfig-extensions.yang#L154
	//
	// For example in the relative path /subinterfaces/subinterface, this
	// field be true if and only if the second element, /interface, is
	// marked "telemetry-atomic" in the YANG schema.
	TelemetryAtomic bool
	// CompressedTelemetryAtomic indicates that a parent of the node which
	// has been compressed out has been modified with the OpenConfig
	// extension "telemetry-atomic".
	//
	// For example, /interfaces/interface/subinterfaces/subinterface may be
	// a path where the /subinterfaces element within the relative path
	// /subinterfaces/subinterface is marked "telemetry-atomic". In this
	// case, this field will be marked true since the relative path from
	// the parent ParsedDirectory contains a compressed-out element that's
	// marked "telemetry-atomic".
	//
	// https://github.com/openconfig/public/blob/master/release/models/openconfig-extensions.yang#L154
	CompressedTelemetryAtomic bool
}

// OrderedFieldNames returns the YANG name of all fields belonging to the
// ParsedDirectory in lexicographical order.
func (d *ParsedDirectory) OrderedFieldNames() []string {
	if d == nil {
		return nil
	}

	fieldNames := make([]string, 0, len(d.Fields))
	for fieldName := range d.Fields {
		fieldNames = append(fieldNames, fieldName)
	}
	sort.Strings(fieldNames)
	return fieldNames
}

// OrderedFieldNames returns the YANG name of all child directories for
// ParsedDirectory in lexicographical order.
// It returns an error if any child directory field doesn't exist in the input IR.
func (d *ParsedDirectory) ChildDirectories(ir *IR) ([]*ParsedDirectory, error) {
	var childDirs []*ParsedDirectory
	for _, fieldName := range d.OrderedFieldNames() {
		if field := d.Fields[fieldName]; field.Type == ContainerNode || field.Type == ListNode {
			childDir, ok := ir.Directories[field.YANGDetails.Path]
			if !ok {
				return nil, fmt.Errorf("%s field %q with path %q not found in input IR", field.Type, fieldName, field.YANGDetails.Path)
			}
			childDirs = append(childDirs, childDir)
		}
	}
	return childDirs, nil
}

// OrderedFieldNames returns the YANG name of all key fields belonging to the
// ParsedDirectory in lexicographical order.
func (d *ParsedDirectory) OrderedListKeyNames() []string {
	keyNames := []string{}
	for name := range d.ListKeys {
		keyNames = append(keyNames, name)
	}
	sort.Strings(keyNames)
	return keyNames
}

type ListKey struct {
	// Name is the candidate language-specific name of the list key leaf.
	Name string
	// LangType describes the type that the node should be given in
	// the output code, using the output of the language-specific
	// type mapping provided by calling the LangMapper interface.
	LangType *MappedType
}

// DirType describes the different types of Directory that
// can be output within the IR such that 'list' directories
// can have special handling applied.
type DirType int64

const (
	_ DirType = iota
	// Container represents a YANG 'container'.
	Container
	// List represents a YANG 'list' that is 'ordered-by system'.
	List
	// OrderedList represents a YANG 'list' that is 'ordered-by user'.
	OrderedList
)

// NodeDetails describes an individual field of the generated
// code tree. The Node may correspond to another Directory
// entry in the output code, or a individual leaf node.
type NodeDetails struct {
	// Name is the language-specific name that should be used for
	// the node.
	Name string
	// YANGDetails stores details of the node from the original
	// YANG schema, such that some characteristics can be accessed
	// by the code generation process. Only details that are
	// directly required are provided.
	YANGDetails YANGNodeDetails
	// Type describes the type of node that the leaf represents,
	// allowing for container, list, leaf and leaf-list entries
	// to be distinguished.
	// In the future it can be used to store other node types that
	// form a direct child of a subtree node.
	Type NodeType
	// LangType describes the type that the node should be given in
	// the output code, using the output of the language-specific
	// type mapping provided by calling the LangMapper interface.
	LangType *MappedType
	// MappedPaths describes the paths that the output node should
	// be mapped to in the output code - these annotations can be
	// used to annotation the output code with the field(s) that it
	// corresponds to in the YANG schema.
	MappedPaths [][]string
	// MappedPathModules describes the path elements' belonging modules that
	// the output node should be mapped to in the output code - these
	// annotations can be used to annotation the output code with the
	// field(s) that it corresponds to in the YANG schema.
	MappedPathModules [][]string
	// ShadowMappedPaths describes the shadow paths (if any) that the output
	// node should be mapped to in the output code - these annotations can
	// be used to annotation the output code with the field(s) that it
	// corresponds to in the YANG schema.
	// Shadow paths are paths that have sibling config/state values
	// that have been compressed out due to path compression.
	ShadowMappedPaths [][]string
	// ShadowMappedPathModules describes the shadow path elements' belonging
	// modules (if any) that the output node should be mapped to in the
	// output code - these annotations can be used to annotation the output
	// code with the field(s) that it corresponds to in the YANG schema.
	// Shadow paths are paths that have sibling config/state values
	// that have been compressed out due to path compression.
	ShadowMappedPathModules [][]string
	// Flags contains extra information that can be populated by the
	// LangMapper during IR generation to assist the code generation stage.
	// Specifically, this field is set by the
	// LangMapperExt.PopulateFieldFlags function.
	Flags map[string]string
}

// NodeType describes the different types of node that can
// be output within the IR.
type NodeType int64

const (
	// InvalidNode represents a node that has not been correctly
	// set up.
	InvalidNode NodeType = iota
	// ContainerNode indicates a YANG 'container'.
	ContainerNode
	// ListNode indicates a YANG 'list'.
	ListNode
	// LeafNode represents a YANG 'leaf'.
	LeafNode
	// LeafListNode represents a YANG 'leaf-list'.
	LeafListNode
	// AnyDataNode represents a YANG 'anydata'.
	AnyDataNode
)

func (n NodeType) String() string {
	switch n {
	case InvalidNode:
		return "invalid"
	case ContainerNode:
		return "container"
	case ListNode:
		return "list"
	case LeafNode:
		return "leaf"
	case LeafListNode:
		return "leaf-list"
	case AnyDataNode:
		return "anydata"
	default:
		return "unknown"
	}
}

// YANGNodeDetails stores the YANG-specific details of a node
// within the schema.
// TODO(wenbli): Split this out so that parts can be re-used by
// ParsedDirectory.
type YANGNodeDetails struct {
	// Name is the name of the node from the YANG schema.
	Name string
	// Defaults represents the 'default' value(s) directly
	// specified in the YANG schema.
	Defaults []string
	// BelongingModule is the name of the module having the same XML
	// namespace as this node.
	// For more information on YANG's XML namespaces see
	// https://datatracker.ietf.org/doc/html/rfc7950#section-5.3
	BelongingModule string
	// RootElementModule is the module in which the root of the YANG tree that the
	// node is attached to was instantiated (rather than the module that
	// has the same namespace as the node).
	//
	// In this example, leaf 'zero' has
	// RootElementModule: "openconfig-simple"
	// BelongingModule:   "openconfig-augment"
	// DefiningModule:    "openconfig-grouping"
	//
	//   module openconfig-augment {
	//     import openconfig-simple { prefix "s"; }
	//     import openconfig-grouping { prefix "g"; }
	//
	//     augment "/s:parent/child/state" {
	//       uses g:group;
	//     }
	//   }
	//
	//   module openconfig-grouping {
	//     grouping group {
	//       leaf zero { type string; }
	//     }
	//   }
	RootElementModule string
	// DefiningModule is the module that contains the text definition of
	// the field.
	DefiningModule string
	// Path specifies the absolute YANG schema node path that can be used
	// to index into the ParsedDirectory map in the IR. It includes the
	// module name as well as choice/case elements.
	Path string
	// SchemaPath specifies the absolute YANG schema node path. It does not
	// include the module name nor choice/case elements in the YANG file.
	SchemaPath string
	// ShadowSchemaPath, which specifies the absolute YANG schema node path
	// of the "shadowed" sibling node, is included when a leaf exists in
	// both 'intended' and 'applied' state of an OpenConfig schema (see
	// https://datatracker.ietf.org/doc/html/draft-openconfig-netmod-opstate-01)
	// and hence is within the 'config' and 'state' containers of the
	// schema. ShadowSchemaPath is populated only when the -compress
	// generator flag is used, and indicates the path of the node not
	// represented in the generated IR based on the preference to prefer
	// intended or applied leaves.
	// Similar to SchemaPath, it does not include the module name nor
	// choice/case elements.
	ShadowSchemaPath string
	// LeafrefTargetPath is the absolute YANG schema node path of the
	// target node to which the leafref points via its path statement. Note
	// that this is *not* the recursively-resolved path.
	// This is populated only if the YANG node was a leafref.
	LeafrefTargetPath string
	// PresenceStatement, if non-nil, indicates that this directory is a
	// presence container. It contains the value of the presence statement.
	PresenceStatement *string
	// Description contains the description of the node.
	Description string
	// OrderedByUser indicates whether the node has the modifier
	// "ordered-by user".
	OrderedByUser bool
	// ConfigFalse represents whether the node is state data as opposed to
	// configuration data.
	// The meaning of "config" is exactly the same as the "config"
	// statement in YANG:
	// https://datatracker.ietf.org/doc/html/rfc7950#section-7.21.1
	ConfigFalse bool
}

// EnumeratedValueType is used to indicate the source YANG type
// that an enumeration was generated based on.
type EnumeratedValueType int64

const (
	UnknownEnumerationType EnumeratedValueType = iota
	// SimpleEnumerationType represents 'enumeration' leaves within
	// the YANG schema that are defined inline.
	SimpleEnumerationType
	// DerivedEnumerationType represents enumerations that are defined
	// within a YANG 'typedef'.
	DerivedEnumerationType
	// UnionEnumerationType represents a 'type enumeration' defined within
	// a union.
	UnionEnumerationType
	// DerivedUnionEnumerationType represents a 'enumeration' defined
	// within a union that is itself within a typedef.
	DerivedUnionEnumerationType
	// IdentityType represents an enumeration that is an 'identity'
	// within the YANG schema.
	IdentityType
)

func (n EnumeratedValueType) String() string {
	switch n {
	case UnknownEnumerationType:
		return "unknown enumeration type"
	case SimpleEnumerationType:
		return "simple enumeration"
	case DerivedEnumerationType:
		return "derived enumeration"
	case UnionEnumerationType:
		return "union enumeration"
	case DerivedUnionEnumerationType:
		return "derived union enumeration"
	case IdentityType:
		return "identity"
	default:
		return "unspecified enumeration type"
	}
}

// EnumeratedYANGType is an abstract representation of an enumerated
// type to be produced in the output code.
type EnumeratedYANGType struct {
	// Name is the name of the generated enumeration to be
	// used in the generated code.
	Name string
	// Kind indicates the type of enumerated value that the
	// EnumeratedYANGType represents - allowing for a code
	// generation mechanism to select how different enumerated
	// value types are output.
	Kind EnumeratedValueType
	// IdentityBaseName, which is present only when the enumerated type is
	// an IdentityType, is the name of the base identity from which all
	// valid identity values are derived.
	IdentityBaseName string
	// TypeName stores the original YANG type name for the enumeration.
	TypeName string
	// TypeDefaultValue stores the default value of the enum type's default
	// statement (note: this is different from the default statement of the
	// leaf type).
	TypeDefaultValue string
	// ValToYANGDetails stores the YANG-ordered set of enumeration value
	// and its YANG-specific details (as defined by the
	// ygot.EnumDefinition).
	ValToYANGDetails []ygot.EnumDefinition
	// Flags contains extra information that can be populated by the
	// LangMapper during IR generation to assist the code generation stage.
	// Specifically, this field is set by the
	// LangMapperExt.PopulateEnumFlags function.
	Flags map[string]string
}
