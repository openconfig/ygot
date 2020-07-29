# Multi-Stage Code Generation within `ygot`
**Authors**: robjs<sup>†</sup>, wenbli<sup>†</sup>    <small>(<sup>†</sup>@google.com)</small>  
**July 2020**

## Overview
Originally, the `ygen` package was developed to solely generate a single format of Go code from a YANG schema. Following this initial implementation, it has subsequently been extended to generate additional code artifacts, including a library for generating path-based helpers (`ypathgen`) and generating protobuf IDL files from a YANG schema. As each of these languages were added, some incremental changes were made (e.g., generating a set of leaf types and directory definitions for `ypathgen`, and refactoring of common state generation for protobuf-generation), but as new options for generation of code have been added to the `ygen` package, its complexity has increased. In order to reduce the maintenance effort for ygen, as well as allow simpler extension of the ygot-suite into generating new code from YANG modules (e.g., C++, or Python), some refactoring is required.

Particularly, we intend to restructure ygen to enforce a strict multi-stage code generation process. The target end-to-end pipeline is targeted to be:

 1. An input set of YANG schemas is to be parsed by `goyang` and resolved through `yang.Node` representations into a set of `yang.Entry` structs. The set of `yang.Entry` structures that exist for a schema will be implemented to be lossless, such that there is no information that is available in the input YANG schema that cannot be extracted from them, and the original format of the structure is maintained.
 1. A refactored `ygen` library will take the input set of `yang.Entry` structures for a schema, and create an intermediate representation (IR) that:
   * Has undergone transformation of the schema that is required by the code generation process. The current schema transformations are described by `genutil.CompressBehaviour` and are implemented solely for OpenConfig modules. Their purpose is to simplify the generated code for particular users.
   * Has resolved the types and names of entities that are to be output in the generated code. This should include the identification of directories, their fields and types, and any derived types that are needed by the generated code (typically enumerated types).

   The IR produced by ygen will be lossy compared to the input YANG schema, and should expose the minimum required fields to the subsequent code generation stages. The IR should not include `yang.Entry` fields, such that there is an explicitly defined API that is available to code generation - and transformations that require full schema knowledge are applied within the `ygen` library itself.
   
   Further to this, the IR itself must be language-agnostic to the greatest extent possible -- with any language-specific requirements being through well-known interfaces that are implemented by downstream code generation.
1. A set of language-specific, and potentially use-case specific, code generation libraries that convert the IR into a specific set of output code. These language libraries may consume only the `ygen` IR, and convert this from the pre-transformed schema into generated code artifacts. Different binaries may be utilised to call these libraries which provide the user interface.

This structure will have the following benefits:

* Knowledge of schema transformations will be clearly encapsulated into the `ygen` library -- today, numerous methods that are outputting code must be aware of the different transformations that are being applied to the schema - resulting in complex state tracking being required through the entire generation process. This causes significant additional effort to understand all the hooks required in the code generation libraries to add new output formats.
* Generation of names for output code entities is moved from being a on-the-fly process, which requires careful consideration of order, or can potentially have non-deterministic output - to being an up-front process. Previous changes have shown the benefits of moving enumeration naming to an up-front process where there can be clear understanding of how names are to be resolved, ensuring that there is a strictly defined IR ensures that this pattern can be applied across the code base.
* The requirement for expert knowledge of YANG for adding code generation can be reduced - for example, rather than requiring a developer adding to `ygen` to understand the structure of a `yang.Entry` and all the possible fields that can be used in these cases, `ygen` itself can clearly document the IR, and keep this to being a subset of the available information - for example, this allows abstraction of properties that depend on a characteristic of the YANG schema that can only be described in a `yang.Node` (e.g., how a element is defined in the actual YANG structure) away from code generation libraries.
* In the future, the IR may form a more compact means to express the YANG schema that is being used by `ytypes` for validation and unmarshalling -- since there is a reasonable binary size, and memory overhead for storing the existing `yang.Entry` structure. This aim is not part of the initial goal of implementing this design.

## Refactored `ygen` Design

### Language-specific Implementation Characteristics

In order to fully resolve the schema into an IR that includes all directories, and types, `ygen` must understand how to name these types. There are two approaches that could be taken to allow this.

* Produce an abstract naming that is language agnostic that can be mapped by the language-specific libraries at output time. Adopting this approach keeps the `ygen` logic itself as simple as possible, but comes at the cost of needing to make assumptions as to the uniqueness required of names -- for example, for languages that use a single namespace for all generated directories (that is `struct` or `message` entities) then there would need to be a globally-unique identifier, for languages that have scoped naming (e.g., use different packages, or scopes for the output code) this global uniqueness may need to be undone. 
* Provide an `interface` via which a 'naming' entity can be provided to the generated code. This interface allows for a language-generating library to pick a specific means by which to name entities, and use this as the means by which they are referenced in the output code. Using this design, each generating library can provide individual "safe naming" methods, e.g., choosing `CamelCase` over `names_with_underscores`, without needing each of these naming behaviours to be translated from an abstract format, or implemented within `ygen` itself.

Based on prototyping, the language specific naming interface is to be adopted. Particularly, we define a `LangMapper` interface with the following characteristics:

```golang
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
```

Within this interface:

* There is some leakage of the schema transformations, and `yang.Entry` outside of the `ygen` library itself (since the package defining the interface must be aware of these). However, these are limited - and encapsulated cleanly within methods that return simple types to the `ygen` generator process.
* Separate methods for naming leaves and directories are provided - it is expected that the `LeafName` can be used to name any field within the generated code, and requires only the input `yang.Entry`, where `DirectoryName` can be used to determine the name of a directory.
* Since we do not want the generating package to need to be aware of the logic behing generating an enumerated type, or the entire schema tree, `ygen` supplies the `LangMapper` with a copy of the fully-resolved set of enumerated types, and the schema tree via which references can be looked up.
* Two methods for generating leaf types are included - `LeafType` is used when the leaf being mapped to a type is a field of a `directory`, and `KeyLeafType` is used when a leaf is being mapped as part of a list key. The two methods are defined separately to ensure that where there are different types that are used for optional vs. mandatory contexts the mapping language can return different types. Most notably, this is the case in the `protobuf` output, where a wrapper type is used for standard leaves, whereas the simple built-in type is used when the field is mandatory as a list key.
* A language-specific method to map enumerated value names - this function should handle the means by which a enumeration or identity name (as specified by the RFC7950 grammar) can be safely translated to a value which can be used in the target language.

An implementation of a `LangMapper` keeps its own internal state that is not made accessible to the `ygen` library. Since a `LangMapper` implementation has access to the `yang.Entry`, it is able to use the entire context of a particular YANG entry to be able to choose how to map it to its name in the generated language.

Other than the `LangMapper` interface, no other language-specific mapping code is implemented in the production of the IR.

### Defining the ygen IR

Currently, the `ypathgen` library defines a `GetDirectoriesAndLeafTypes` method that can be called to return:

 * A map of the directories that are used in the generated code, keyed by the YANG path (represented as a string).
 * A map of maps, keyed by directory name, with the inner map keyed by field name, returning the language-type to be used for a particular leaf.

These types are sufficient for `ypathgen` to generate the code that it currently generates, but insufficient for all languages to be produced (or other formats of Go). Equally, there is some duplication between these types that requires cross-referencing between them which could be simplified.

We propose to modify this return format - and define the IR to be broken down into two subsets - encapsulated by a common type:

```
type Definitions struct {
	// ParsedTree is the set of parsed directory entries, keyed by the YANG path
	// to the directory.
	ParsedTree map[string]*ParsedDirectory
	
	// Enums is the set of enumerated entries that are extracted from the generated
	// code. The key of the map is the global name of the enumeration.
	Enums map[string]*EnumeratedYANGType

	// ModelData stores the metadata extracted from the input YANG modules.
	ModelData     []*gpb.ModelData
}
```

#### Definitions for Directory / Leaf Nodes

The proposed IR for the directory types in the returned code is as follows. The base philosophy of whether fields are included within this representation is to keep to the set of fields made available to the code output library to their minimum, to ensure that there is clear encapsulation of the complexities of the YANG hierarchy within `ygen`.

```golang
// ParsedDirectory describes an internal node within the generated
// code. Such a 'directory' may represent a struct, or a message,
// in the generated code. It represents a YANG 'container' or 'list'.
type ParsedDirectory struct {
   // Name is the language-specific name of the directory to be
   // output.
	Name       string
	// Type describes the type of directory that is being produced -
	// such that YANG 'list' entries can have special handling. 
	Type		 DirType
	// Fields is the set of direct children of the node that are
	// to be output. It is keyed by the name of the child field
	// using a language-specific format.
	Fields     map[string]*NodeDetails
	// ListAttr describes the attributes of a YANG list that
	// are required in the output code (e.g., the characteristics
	// of the list's keys).
	ListAttr   *YangListAttr
	// IsFakeRoot indicates whether the directory being described 
	// is the root entity and has been synthetically generated by
	// ygen.
	IsFakeRoot bool
}

// NodeDetails describes an individual field of the generated
// code tree. The Node may correspond to another Directory
// entry in the output code, or a individual leaf node.
type NodeDetails struct {
   // Name is the language-specific name that should be used for
   // the node.
	Name        string
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
	Type        NodeType
	// LangType describes the type that the node should be given in
	// the output code, using the output of the language-specific
	// type mapping provided by calling the LangMapper interface.
	LangType    *MappedType
	// MapPaths describes the paths that the output node should
	// be mapped to in the output code - these annotations can be
	// used to annotation the output code with the field(s) that it
	// corresponds to in the YANG schema.
	MapPaths    [][]string
}

// YANGNodeDetails stores the YANG-specific details of a node
// within the schema.
type YANGNodeDetails struct {
	// Name is the name of the node from the YANG schema.
	Name    string
	// Default represents the 'default' value directly
	// specified in the YANG schema.
	Default string
	// Module stores the name of the module that instantiates
	// the node.
	Module  string
	// Path specifies the complete YANG schema node path.
	Path    []string
}
```

#### Definitions for Enumerated Types

YANG has a number of different types of enumerated values - particularly, `identity` and `enumeration` statements. Generated code may choose to treat these differently. For example, for an `enumeration` leaf - some languages may choose to use an embedded enumeration (e.g., protobuf can use a scoped `Enum` within a message). For this reason, some of the provenance of a particular value needs to be exposed to the downstream code generation libraries.

Despite this, a significant amount of pre-parsing can be done by the `ygen` library to pre-process enumerated values prior to being output to code. Particularly:

* Handling of an enumerated value's integer ID to an language-specific name can be created (using the relevant features of the `LangMapper` interface).
* A reverse mapping between the integer ID and defining module and corresponding YANG identifier can be created.

By performing this pre-processing, the functionality of the code generation library is constrained to determine how each type of enumeration should be output, and creating the language-specific constructs required for mapping (e.g., the enumeration map created in `ygen`'s current Go output).

The IR format for enumerations is defined to be as follows:

```golang
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
	// DerivedUnionEnumeration type represents a 'enumeration' within
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
```

Some additional information - such as the `ValuePrefix` can be optionally generated when processing the enumerated types by the `LangMapper`, and is stored within the relevant `yang.Entry` as an `Annotation` field. 

## Proposed Next Steps

This design has been arrived at through significant prototyping work in the `ygen` code-base. Following agreement of this design, we will begin to refactor the existing ygen code to meet this pattern. This work will be considered a pre-requisite for a 1.0.0 release of ygot as a package.