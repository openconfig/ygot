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
	"sort"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/genutil"
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

	// TODO(wenbli): Add this.
	// EnumeratedValueName maps an input string representing an enumerated
	// value to a language-safe name for the enumerated value. This function
	// should ensure that the returned string is sanitised to ensure that
	// it can be directly output in the generated code.
	//EnumeratedValueName(string) (string, error)

	// TODO(wenbli): Consider removing this from the IR since the prefix
	// can depend on the type of the enumeration, so it might make sense to
	// have the code generation stage do this instead.
	// EnumeratedTypePrefix specifies a prefix that should be used as a
	// prefix to types that are mapped from the YANG schema. The prefix
	// is applied only to the type name - and not to the values within
	// the enumeration.
	//EnumeratedTypePrefix(EnumeratedValueType) string

	// SetEnumSet is used to supply a set of enumerated values to the
	// mapper such that leaves that have enumerated types can be looked up.
	// An enumSet provides lookup methods that allow:
	//  - simple enumerated types
	//  - identityrefs
	//  - enumerations within typedefs
	//  - identityrefs within typedefs
	// to be resolved to the corresponding type that is to be used in
	// the IR.
	SetEnumSet(*enumSet)

	// SetSchemaTree is used to supply a copy of the YANG schema tree to
	// the mapped such that leaves of type leafref can be resolved to
	// their target leaves.
	SetSchemaTree(*schemaTree)
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
	// List represents a YANG 'list'.
	List
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
	// Type is the YANG type which represents the node. It is only
	// applicable for leaf or leaf-list nodes because only these nodes can
	// have type statements.
	Type *YANGType
}

// YANGType represents a YANG type.
type YANGType struct {
	// Name is the YANG type name of the type.
	Name string
	// TODO(wenbli): Add this.
	// Module is the name of the module which defined the type. This is
	// only applicable if the type were a typedef.
	//Module string
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
	// ValuePrefix stores any prefix that has been annotated by the IR generation
	// that specifies what prefix should be prepended to value names within the type.
	ValuePrefix []string
	// TypeName stores the original YANG type name for the enumeration.
	TypeName string
	// ValToYANGDetails stores the YANG-ordered set of enumeration value
	// and its YANG-specific details (as defined by the
	// ygot.EnumDefinition).
	ValToYANGDetails []ygot.EnumDefinition
}
