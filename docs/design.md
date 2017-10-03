# `ygen` Library Design

## Introduction
This document describes how YANG modeled data is mapped into Go code entities,
by the ygen library.

YANG ([RFC6020](https://tools.ietf.org/html/rfc6020)) is a data modelling
language, that is used to describe a schema. ygen is primarily developed to meet the use case of allowing Gophers to interact with the
[OpenConfig](http://www.openconfig.net) data models.

In order to make a YANG schema useful, some means to instantiate data trees
which correspond to the schema is required. The `ygen` library uses the
[`goyang`](https://github.com/openconfig/goyang) to parse a YANG model, and
extract the AST corresponding to the model; `ygen` takes such a parsed tree
and outputs a set of Go structs and enumerations that correspond to nodes in the
schema.

## OpenConfig Path Compression

OpenConfig YANG models correspond to a specific hierarchy; which is designed to
allow machine-to-machine interaction as well as for human consumption. This
leads to additional levels of hierarchy being introduced to the model,
particularly:

* Data values (`leaf` nodes) are mirrored in a `config` and `state` container -
  such that it is possible for a system to determine the *intended*
  configuration for a `leaf`, along with the *applied* configuration. The former
  is stored as a read-write `leaf` within a `container` which is named `config`,
  whilst the latter is reflected by a `leaf` of the same name in a `container`
  named `state` at the same level of the tree as the `config` container.
* YANG `list` nodes are enclosed in a `container`, which they are the sole child
  of. This allows for some systems to provide means to retrieve the
  entire list, rather than solely the keys when a particular leaf path is
  queried.

To improve human usability, the `ygen` library provides a `CompressOCPaths`
option (specified in the `YANGCodeGenerator` struct's `Config` field). When `CompressOCPaths` is
set to `true`, the following schema transformations are made:

* The `config` and `state` containers are "compressed" out of the schema.
* The surrounding `container` entities are removed from `list` nodes.

This results in a model such as the OpenConfig interfaces model having paths
that are shorter, and more human-usable:

* `/interfaces/interface/subinterfaces/subinterface/ipv4/addresses/address`
  becomes `/interface/subinterface/ipv4/address`.
* `/interfaces/interface/subinterfaces/subinterface/config/enabled` becomes
  `/interface/subinterface/enabled`.
* `/interfaces/interface/subinterfaces/subinterface/state/oper-status` becomes
  `/interface/subinterface/oper-status`.

With `CompressOCPaths` set to `true`, the modified forms of the paths are used
whenever the path of an entity is required (e.g., in YANG name generation).

The logic to extract which entities are valid to have code
generation performed for them (skipping `config`/`state` containers, and
surrounding containers for lists) is found in
`go_elements.go`:`findAllChildren`.

## YANG Entities Mapped to Go Entities

`ygen` creates two types of Go output, a set of structs - corresponding to
containers or list nodes within the schema; and a set of enumerations which
correspond to nodes that have a restricted set of values in the schema. The set
of enumerated values are:

* `leaf` nodes which have a YANG `type` of enumeration, whether directly or
  within a `union`.
* `identity` statements in the schema.
* `typedef` statements within the YANG schema which have a `type` of
  `enumeration`, as their sole type, or within an `enumeration`.

Each entity is named in the output Go code according to its path in the schema.
The path may be modified using the `CompressOCPaths` as described above.

### Output Go structures and their fields

`struct` entities are named according to their path, with each path element
being converted to CamelCase and concatenated in the form
`PathElementOne_PathElementTwo` - such that `/interfaces/interfaces/config`
becomes `Interfaces_Interface_Config` (if path compression is disabled). In the
case that path compression is enabled, the `interface` list becomes `Interface`.

Each leaf that is contained under a particular `container` is represented by a
member of the struct, with the leaf's name converted to CamelCase.

If an entity has an extension with the name `camelcase-name`, this can be used to specify the CamelCase name of the entity explicitly, rather than relying on the the goyang `yang.CamelCase` function for naming.

Pointers are used for all scalar field types (non-slice, or map) such that unset (`nil`) fields can be distiguished from those that are set to their null value. The `ygot` package provides a set of helper methods to return an input value as a pointer - for example, `ygot.String("foo")` will return a string pointer suitable for setting a YANG string field.

For example, the following YANG module:

```yang
container test {
	leaf a { type string; }
	leaf b { type uint8; }
	leaf-list c { type string; }
}
```

Will be output as the following Go struct:

```go
type Test struct {
	A	*string	`path:"a"`
	B	*uint8		`path:"b"`
	C	[]string	`path:"c"`
}
```

All structs that are produced by the `ygen` library implement the `ygot.GoStruct` interface, such that handling code can determine the provenance of such structures.

### Naming of Enumerated Entities

For each enumerated entity (described above), an enumerated type in Go is
generated, in a similar fashion to the `proto` library. Naming is according to
the type of the enumerated leaf in YANG.

* `leaf` nodes with a type of `enumeration` are mapped to an enumeration named
  according to the path of the `leaf`. The path specified is
  `ModuleName_LeafParentName_LeafName` such that a path of
  `/interfaces/interface/state/enumerated-value` defined within the
  `openconfig-interfaces` module is represented by an enumerated type named
  `OpenconfigInterfaces_State_EnumeratedValue` (assuming path compression is
  disabled), or `OpenconfigInterfaces_Interface_EnumeratedValue` when it is
  enabled.
  * This mapping is handled by `yang_helpers.go`:`resolveEnumName`.
* Defined `identity` statements are generated only when they are referenced by a
  `leaf` in the schema (i.e., an `identityref`). They are named according to the
  module that they are defined in, and the `identity` name - i.e., `identity
  foo` in module `bar-module` is named `BarModule_Foo`. The naming of such
  identities is not modified when `CompressOCPaths` is enabled.
  * This mapping is handled by `yang_helpers.go`:`resolveIdentityRefBaseType`.
* Non-builtin types created via a `typedef` statement that contain an
  enumeration are identified according to the module that they are defined in,
  and the `typedef` name - i.e., `typedef bar { type enumeration { ... }}` in
  module `baz` is represented by an enumerated type named `Bar_Baz`.
  * This mapping is handled by `yang_helpers.go`:`resolveTypedefEnumeratedName`.

Only a single enumeration is generated for a `typedef` or `identity` -
regardless of the number of times that is referenced throughout the code. This
ensures that the user of the library does not have to be aware of the
enumeration's context when referencing the Go enumerated type. Since `typedef`
and `identity` nodes do not have a path within the YANG schematree, the library
uses the synthesised name `module-name/statement-name` as a pseudo-path to
reference each `typedef` and `identity` such that the name it is mapped to in Go
code can be re-used throughout code generation.

#### Handling Name Collisions

Since in YANG `leaf-one` and `leaf-One` are considered unique names, during the
process of converting a name to CamelCase, it is possible that two entities are
mapped to the same CamelCase name (`LeafOne` in this case). Such cases are
handled by appending underscores to the name of an entity as its name is
converting to CamelCase until such time as the name is unique. i.e., in the case
that `leaf-one` and `leaf-One` exist within the same container then the first
mapped entity will be named `LeafOne` and the second `LeafOne_`. A similar
de-duplication technique is utilised for the names of enumerated types
(following the process described above).

It is not expected that with OpenConfig schemas, such name collisions are
encountered, although at the time of writing, no OpenConfig linter rule exists
to ensure that this is the case.

### Mapping of YANG Types to Go Types

The following mapping between YANG and Go types are used by the `ygen`
library:

YANG Type | Go Type  | Notes
--------- | -------- | -------
`int{8,16,32,64}` | `int{8,16,32,64}` |
`uint{8,16,32,64}` | `uint{8,16,32,64}` |
`bool` | `bool` |
`empty` | `bool` (derived) |
`string` | `string` |
`union` | `interface{}` | A `union` is represented as an empty interface, with validation intending to be done whilst mapping into the `//ops/openconfig/lib/go` library.
`enumeration` | `int64` | Each enumeration is generated as a new type based on Go's int64, names are assigned to each value of the enumeration akin to the `proto` library.
`identityref` | `int64` | The identityref's "base" is mapped using the same process as the an enumeration leaf.
`decimal64` | `float64` |
`binary` | `[]byte` (derived) |
`bits` | `interface{}` | TODO(robjs): Add support for `bits`, this is low priority as it is not used in any OpenConfig schema.

### YANG Lists

YANG Lists are output as `map` fields within the Go structures, with a key type that is derived from the YANG schema, for example:

```
container c {
	list foo {
		key "fookey";
	
		leaf fookey { type string; }
	}
	
	list bar {
		key "barkey1 barkey2";
		
		leaf barkey1 { type string; }
		leaf barkey2 { type string; }
		leaf barmember { type string; }
	}
}
```

Is output as:

```
type C struct {
	Foo		map[string]*C_Foo	`path:"foo"`
	Bar		map[C_Bar_Key]*C_Bar	`path:"bar"`
}

type C_Foo struct {
	FooKey		*string	`path:"fookey"`
}

type C_Bar_Key struct {
	Barkey1	string
	Barkey2	string
}

type C_Bar struct {
	Barkey1	*string	`path:"barkey1"`
	Barkey2	*string	`path:"barkey2"`
	Barmember	*string	`path:"barmmember"`
} 
```

Such that the `Foo` field is a map, keyed on the type of the key leaf (`fookey`). For lists with multiple keys, a specific key `struct` is generated (`C_Bar_Key` in the above example), with fields that correspond to the key fields of the YANG list.

Each YANG list that exists within a container has a helper-method generated for it. For a list named `foo`, the parent container (`C`) has a `NewFoo(fookey string)` method generated, taking a key value as an argument, and returning a new member of the map within the `foo` list.

### YANG Union Leaves

In order to preserve strict type validation at compile time, `union` leaves within the YANG schema are mapped to an Go `interface` which is subsequently implemented for each type that is defined within the YANG union.

For the following YANG module:

```yang
container foo {
	container bar {
		leaf union-leaf {
			type union {
				type string;
				type int8;
			}
		}
	}
}
```

the `bar` container is mapped to:

```go
type Bar struct {
	UnionLeaf		Foo_Bar_UnionLeaf_Union		`path:"union-leaf"`
}

type Foo_Bar_UnionLeaf_Union interface {
	Is_Foo_Bar_UnionLeaf_Union()
}

type Foo_Bar_UnionLeaf_Union_String struct {
	String string
}

func (Foo_Bar_UnionLeaf_Union_String) Is_Foo_Bar_UnionLeaf_Union() {}

type Foo_Bar_UnionLeaf_Union_Int8 struct {
	Int8 int8
}

func (Foo_Bar_UnionLeaf_Union_Int8) Is_Foo_Bar_UnionLeaf_Union() {}
```

The `UnionLeaf` field can be set to any of the structs that implement the `Foo_Bar_UnionLeaf_Union` interface. Since these structs are single-field entities, a struct initialiser that does not specify the field name can be used (e.g., `Foo_Bar_UnionLeaf_Union_String{"baz"}`), similarly to the generate Go code for a Protobuf `oneof`.

