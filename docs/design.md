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

To improve human usability, the `ygen` library provides a `CompressBehaviour`
option (specified in the `YANGCodeGenerator` struct's `Config` field). When
`CompressBehaviour` is set to one of the compressed options, the following
schema transformations are made:

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

With `CompressBehaviour` set to a compressed value, the modified forms of the
paths are used whenever the path of an entity is required (e.g., in YANG name
generation).

The logic to extract which entities are valid to have code
generation performed for them (skipping `config`/`state` containers, and
surrounding containers for lists) is found in
`FindAllChildren` in `genutil` package.

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
The path may be modified using the `CompressBehaviour` as described above.

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
the type of the enumerated leaf in YANG. Each name element is in camelcase, and
when Go is generated, they are delimited by underscores in the same style as
struct names.

* `leaf` nodes with a type of `enumeration` are mapped to an enumeration named
  according to the path of the `leaf`. The path specified is
  `ModuleName_<PathElement1>_<PathElement2>_..._<PathElementN>_LeafName` (index
  starting from 1), or for compressed paths, `LeafGrandParentName_LeafName`,
  such that a path of `/interfaces/interface/state/enumerated-value` defined
  within the `openconfig-interfaces` module is represented by an enumerated type
  named `OpenconfigInterfaces_Interfaces_Interface_State_EnumeratedValue`
  (assuming path compression is disabled), or `Interface_EnumeratedValue` when
  it is enabled. Here, `ModuleName` refers to the defining module of the
  `enumeration` type.
  * This mapping is handled by `enumgen.go`:`resolveEnumName`.
* Defined `identity` statements are generated only when they are referenced by a
  `leaf` in the schema (i.e., an `identityref`). They are named according to the
  module that they are defined in, and the `identity` name - i.e., `identity
  foo` in module `bar-module` is named `BarModule_Foo`. The naming of such
  identities is not modified when compression is enabled.
  * This mapping is handled by `enumgen.go`:`resolveIdentityRefBaseType`.
* Non-builtin types created via a `typedef` statement that contain an
  enumeration are identified according to the module that they are defined in,
  and the `typedef` name - i.e., `typedef bar { type enumeration { ... }}` in
  module `baz` is represented by an enumerated type named `Bar_Baz`.
  * This mapping is handled by `enumgen.go`:`resolveTypedefEnumeratedName`.
* Where an `enumeration` is defined within a `typedef` that contains a `union`,
  the enumerated language type that is generated is named according to the name 
  of the `typedef` with `_Enum` appended to the name.
  * This mapping is handled by `enumgen.go`:`resolveEnumeratedUnionEntry`.

  For example:
```
module bar {
  ...
  typedef baz {
     type union {
        type enumeration { ... }
        type string;
     }
  }
}
```

  would result in a type named `Bar_Baz_Enum` being generated in the output
  code.

Only a single enumeration is generated for a `typedef` or `identity` -
regardless of the number of times that is referenced throughout the code. This
ensures that the user of the library does not have to be aware of the
enumeration's context when referencing the Go enumerated type. Since `typedef`
and `identity` nodes do not have a path within the YANG schematree, the library
uses the synthesised name `defined-module-name/statement-name` as a pseudo-path
to reference each `typedef` and `identity` such that the name it is mapped to
in Go code can be re-used throughout code generation.

There are occasions where an `enumeration` leaf is used in multiple places due
to re-use of a grouping. In such cases, the leaf whose path is lexicographically
earlier will by default determine the name of the enumeration in the generated
code. While this may work well for some YANG schemas, it essentially requires
knowledge of the other parts of the schema and may not suit others. The
`-skip_enum_deduplication` flag within `generator.go` overrides this behaviour
and generates different enumerations in the generated code as if there was no
re-use of these `enumeration` leaves (unless their generated names were to be
the same anyways).

A conflict in enumerated type names may occur due to the way they are defined,
and for most enumerated types, such a collision will cause an error to be
returned when attempting to generate code. Since compressed enumeration leaves
have a high probability of name collision, it has a conflict resolution
mechanism that works in the following manner when multiple distinct enumerations
whose default names in the format `LeafGrandParentName_LeafName` collide:
* If prepending the module name disambiguates all conflicting enumerations, then
  `ModuleName_LeafGrandParentName_LeafName` is the name format for all
  conflicting enumeration leaves.
* If the module name fails to disambiguate, then equidistant non-module ancestor
  names relative to each enumeration, starting from
  `LeafGreatGrandParentName_LeafGrandParentName_LeafName` as the format for all
  enumerations, is checked one by one for disambiguation until success. If an
  equidistant ancestor does not exist for a single enumeration, disambiguation
  is still possible, provided the ancestor exists for all others in the conflict
  set. If more than one enumeration runs out of ancestors to try for
  disambiguation, however, an error is returned stating that the names cannot be
  resolved.

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

If there is already a Go structure named `C_Bar_Key` due to the [camel case rules](#output-go-structures-and-their-fields), then the back-up name of `C_Bar_YANGListKey` will be used instead.

Each YANG list that exists within a container has a helper-method generated for it. For a list named `foo`, the parent container (`C`) has a `NewFoo(fookey string)` method generated, taking a key value as an argument, and returning a new member of the map within the `foo` list.

##### Note on using binary as a list key type
Because `Binary`'s underlying `[]byte` type is not hashable, YANG models
containing lists with `binary` as a key value, or a `union` type containing a
`binary` type is not supported. An error is returned by the Go code generation
process for such cases, this is a known limitation.

### YANG Union Leaves

In order to preserve strict type validation at compile time, `union` leaves within the YANG schema are mapped to an Go `interface` which is subsequently implemented for each type that is defined within the YANG union.

For the following YANG module:

```yang
container foo {
	container bar {
		leaf union-leaf {
			type union {
				type int8;
				type enumeration {
					enum ONE;
					enum TWO;
				}
			}
		}
	}
}
```

The `bar` container can be translated to Go code according to one of the
following strategies:

#### Simplified Union Leaves (Recommended)
In this representation, generated defined types are used to represent all concrete union types.
```go
type Binary []byte
type YANGEmpty bool
type Int8 int8
type Int16 int16
// ... etc.
type String string
type Bool bool
```

```go
type Bar struct {
	UnionLeaf		Foo_Bar_UnionLeaf_Union		`path:"union-leaf"`
}

type Foo_Bar_UnionLeaf_Union interface {
	// Union type can be one of [Int8, E_Foo_Bar_UnionLeaf]
	Documentation_for_Foo_Bar_UnionLeaf_Union()
}

func (Int8) Documentation_for_Foo_Bar_UnionLeaf_Union() {}

func (E_Foo_Bar_UnionLeaf) Documentation_for_Foo_Bar_UnionLeaf_Union() {}
```

The `UnionLeaf` field can be set to any defined type (including enumeration
typedefs) that implements the `Foo_Bar_UnionLeaf_Union` interface. These
typedefs are re-used for different union types; so, it's possible to assign an
`Int8` value to any union which has `int8` in its definition.

#### Wrapper Union Leaves

```go
type Bar struct {
	UnionLeaf		Foo_Bar_UnionLeaf_Union		`path:"union-leaf"`
}

type Foo_Bar_UnionLeaf_Union interface {
	Is_Foo_Bar_UnionLeaf_Union()
}

type Foo_Bar_UnionLeaf_Union_Int8 struct {
	Int8 int8
}

func (Foo_Bar_UnionLeaf_Union_Int8) Is_Foo_Bar_UnionLeaf_Union() {}

type Foo_Bar_UnionLeaf_Union_E_Foo_Bar_UnionLeaf struct {
	E_Foo_Bar_UnionLeaf E_Foo_Bar_UnionLeaf
}

func (Foo_Bar_UnionLeaf_Union_E_Foo_Bar_UnionLeaf) Is_Foo_Bar_UnionLeaf_Union() {}
```

The `UnionLeaf` field can be set to any of the structs that implement the
`Foo_Bar_UnionLeaf_Union` interface. Since these structs are single-field
entities, a struct initialiser that does not specify the field name can be used
(e.g., `Foo_Bar_UnionLeaf_Union_Int8{42}`), similarly to the generate Go code
for a Protobuf `oneof`.
