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

type NewLangMapperFn func() LangMapper

// LangMapper is the interface to be implemented by a language-specific
// library and provided as an input to the IR production phase of ygen.
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
	KeyLeafType(*yang.Entry, genutil.CompressBehaviour) (*MappedType, error)

	// LeafType maps an input yang.Entry which must represent a leaf to the
	// type that should be used when the leaf is used in the context of a
	// field within a directory within the output IR.
	LeafType(*yang.Entry, genutil.CompressBehaviour) (*MappedType, error)

	// EnumeratedValueName maps an input string representing an enumerated
	// value to a language-safe name for the enumerated value. This function
	// should ensure that the returned string is sanitised to ensure that
	// it can be directly output in the generated code.
	EnumeratedValueName(string) (string, error)

	// EnumeratedTypePrefix specifies a prefix that should be used as a
	// prefix to types that are mapped from the YANG schema. The prefix
	// is applied only to the type name - and not to the values within
	// the enumeration.
	EnumeratedTypePrefix() string

	// EnumerationsUseUnderscores specifies whether enumeration names
	// should use underscores between path segments.
	EnumerationsUseUnderscores() bool

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

// IR represents the returned intermediate representation produced by ygen.
type IR struct {
	// Directories is the set of 'directory' entries that are to be produced
	// in the generated code.
	Directories map[string]*ParsedDirectory

	// Enums is the set of enumerated entries that are to be output in the
	// generated language code.
	Enums map[string]*EnumeratedYANGType
}

// ParsedDirectory describes an internal node within the generated
// code. Such a 'directory' may represent a struct, or a message,
// in the generated code. It represents a YANG 'container' or 'list'.
type ParsedDirectory struct {
	// Name is the language-specific name of the directory to be
	// output.
	Name string
	// Type describes the type of directory that is being produced -
	// such that YANG 'list' entries can have special handling.
	Type DirType
	// Fields is the set of direct children of the node that are
	// to be output. It is keyed by the name of the child field
	// using a language-specific format.
	Fields map[string]*NodeDetails
	// ListAttr describes the attributes of a YANG list that
	// are required in the output code (e.g., the characteristics
	// of the list's keys).
	ListAttr *YangListAttr
	// IsFakeRoot indicates whether the directory being described
	// is the root entity and has been synthetically generated by
	// ygen.
	IsFakeRoot bool
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
	// MapPaths describes the paths that the output node should
	// be mapped to in the output code - these annotations can be
	// used to annotation the output code with the field(s) that it
	// corresponds to in the YANG schema.
	MapPaths [][]string
}

// NodeType describes the different types of node that can
// be output within the IR.
type NodeType int64

const (
	// InvalidNode represents a node that has not been correctly
	// set up.
	InvalidNode NodeType = iota
	// DirectoryNode indicates a YANG 'container'.
	DirectoryNode
	// ListNode indicates a YANG 'list'.
	ListNode
	// LeafNode represents a YANG 'leaf'.
	LeafNode
	// LeafListNode represents a YANG 'leaf-list'.
	LeafListNode
)

// YANGNodeDetails stores the YANG-specific details of a node
// within the schema.
type YANGNodeDetails struct {
	// Name is the name of the node from the YANG schema.
	Name string
	// Default represents the 'default' value directly
	// specified in the YANG schema.
	Default string
	// Module stores the name of the module that instantiates
	// the node.
	Module string
	// Path specifies the complete YANG schema node path.
	Path []string
}

// EnumeratedValueType is used to indicate the source YANG type
// that an enumeration was generated based on.
type EnumeratedValueType int64

const (
	_ EnumeratedValueType = iota
	// SimpleEnumerationType represents 'enumeration' leaves within
	// the YANG schema.
	SimpleEnumerationType
	// DerivedEnumerationType represents enumerations that are within
	// a YANG 'typedef'
	DerivedEnumerationType
	// UnionEnumerationType represents a 'type enumeration' within
	// a union.
	UnionEnumerationType
	// DerivedUnionEnumerationType represents a 'enumeration' within
	// a union that is itself within a typedef.
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
	// that specifies what prefix should be appended to value names within the type.
	ValuePrefix []string
	// TypeName stores the original YANG type name for the enumeration.
	TypeName string

	// ValToCodeName stores the mapping between the int64
	// value for the enumeration, and its language-specific
	// name.
	ValToCodeName map[int64]string
	// ValToYANGDetails stores the mapping between the
	// int64 identifier for the enumeration value and its
	// YANG-specific details (as defined by the ygot.EnumDefinition).
	ValToYANGDetails map[int64]*ygot.EnumDefinition
}
