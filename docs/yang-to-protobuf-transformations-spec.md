## YANG to Protobuf: Transformation Specification

**Revision:**: 1.1.0  
**Published**: September 2017   
**Contributors**: robjs<sup>†</sup> (ed), aashaikh<sup>†</sup>, kevingrant<sup>†</sup>, hines<sup>†</sup>, csl<sup>†</sup>, wmohsin<sup>†</sup>, aghaffar<sup>†</sup>, tmadejski<sup>†</sup>  
<small><sup>†</sup> @google.com</small>


## Introduction

This document defines the method for transforming a YANG schema to a Protobuf
IDL file. This document concisely defines the rules for transformation.


## YANG to Protobuf Types Mapping

To allow values to be distinguished as explicitly null versus unset (required to
allow for defaults in a YANG schema in `proto3`), all types are mapped to a
wrapper message similar to the open source
[`wrappers.proto`](https://github.com/google/protobuf/blob/master/src/google/protobuf/wrappers.proto)).
Since protobuf does not have the same fidelity of types as YANG, the most
permissive numeric type is used to store each value. See notes on protobuf field
options below for information as to how original YANG schema information can be
retrieved.

| YANG Type               | Protobuf Type                       | Notes         | 
| ----------------------- | ----------------------------------- | ------------- |
| `binary`                | `bytes` as `ywrapper.BytesValue`    | Length restrictions encoded as a field option.  |
| `bits`                  | `enum`                              | Each value within the `enum` utilises a name of the `bit` argument to the `bits` type and the value of the bit `position`.              |
| `boolean`               | `bool` as `ywrapper.BoolValue`      |               |
| `decimal64`             | `ywrapper.Decimal64Value`           |  The `Decimal64` message contains an integer value of the `digits` and an unsigned integer `precision` indicating the number of digits following the decimal point. |
| `empty`                 | `bool` as `ywrapper.BoolValue`      |               |
| `enumeration`           | `enum`                              | Embedded within a message where an `enumeration` field exists, globally defined `enum` if corresponding to a typedef.              |
| `identityref`           | `enum`                              | A global `enum` is generated for the `identityref` base `identity`. |
| `instance-identifier`   | `gnmi.Path`                         | gNMI’s paths are used to reference a node within the tree. Refer to the [path specification](https://github.com/openconfig/reference/blob/master/rpc/gnmi/gnmi-path-conventions.md). |
| `int{8,16,32,64}`       | `sint64` as `ywrapper.IntValue`     | Range restrictions encoded as a field option. |
| `leafref`               | Type of leafref `path` target node  |               |
| `string`                | `string` as `ywrapper.StringValue`  | Length and pattern restrictions encoded as a field option.  |
| `uint{8,16,32,64}`      | `uint64` as `ywrapper.UintValue`    | Range restrictions encoded as a field option. |
| `union`                 | `oneof` containing included types   | See note below concerning repeated `union` fields. |


Types that are not built-in types are flattened to their underlying type(s). For
example, a `typedef` specifying a `string` is marked solely as a `string` in
protobuf.

YANG `leaf-list` entities are mapped to a `repeated` field containing the
relevant wrapper message for the included type. In the case that a `leaf-list`
of union values exists, it is mapped to a `repeated` field containing a message
generated with the `oneof` representing the union as the only field.


## Field and Message Naming

Each directory entry within the YANG schema (i.e., `list` or `container` node)
maps to a protobuf message. Such messages are contained within their own package
dependent upon the schema path to the directory. For instance,
`/interfaces/interface/subinterfaces/subinterface/config` is contained within a
`openconfig_interfaces.interfaces.interface.subinterfaces.subinterface package`
package. The [ygot](github.com/openconfig/ygot) package writes these messages
out in a hierarchical file structure.

Messages are named by translating the name of the message into `CamelCase`
optionally using the `openconfig-codegen-extensions` field `camelcase-name`
annotation to learn the supplied camelcase-ified name if it is present.

Fields are named in the form `foo` or `foo_bar`. All characters that cannot be
used within a protobuf field name (e.g., `-`) are translated to underscores.


## Enumeration Naming

YANG `enumeration` leaves are embedded within the message that represents the
YANG directory entry within which they are defined. In the case of enumeration
that can be referenced by more than one leaf (e.g., `typedef`, or identities
that are referenced by `identityref `leaves) these are output to a single global
enumerations file, with the naming corresponding to their location within the
schema, e.g., `ModuleNameIdentityName` and `ModuleNameTypedefName`. Values
within enumerations are represented in the format of
`UPPERCASE_WITH_UNDERSCORES`. 


## Mapping of YANG Lists

YANG lists are represented in the output protobuf as a `repeated` field which in
turn contains:

*   Each field that is specified within the YANG list's `key` statement.
*   A field containing a message which contains all other entities within the
    YANG list (not including the list keys).

The motivation for this choice such that we simply support cases whereby the
type of the key is not possible to use within a YANG map. 

For example:


```
container parent {
  list foo-list {
    key "k1 k2";
  
    leaf k1 { type string; } 
    leaf k2 { type string; }
    
    leaf bar { type string; } 
  }
}
```


Is translated to:


```
message Parent {
  repeated FooList foo_list = 1;
}

message FooList {
  string k1 = 1;
  string k2 = 2;
  FooList value = 3;
}

message FooList {
  string bar = 3;
}
```

## Field Numbering

By default, all protobuf fields have a tag number generated for them by
consistently hashing the path of the field using the Fowler-Noll-Vo hash of the
string. Two sets of values are reserved:

*   19,000 - 19,999 which are reserved for Protobuf internal usage.
*   1-1,000 such that there is a possibility for explicit annotations to be
    utilised (improving efficiency, or making protobufs appear more consistent
    than those that were generated).

In order to explicitly specify a field tag, the `field-number` extension
specifying the field number that is expected for the entity. This extension is
defined within the OpenConfig code generation extensions module. In addition, to
avoid conflicts when multiple logical groupings are imported into the same
schema tree location, the `uses` statement can be annotated with the
`field-number-offset` when more than one grouping is utilised within the same
`container` or `list` within the YANG schema. That is to say:


```
grouping a {
        leaf one {
                type string;
                occodegenext:field-number 1;
        }
}

grouping b {
        leaf two {
                type string;
                occodegenext:field-number 1;
        }
}

container foo {
        uses grouping-a;
        uses grouping-b {
                occodegenext:field-number-offset 100;
        }
}
```


Both leaves `one` and `two` have been tagged with `field-number` equal to `1`.
Code generating protobuf messages would attempt to utilise this field number as
a duplicate, thus creating an error. However, reading the `field-number-offset`
extension specifies that the fields within `grouping-b` should utilise an offset
of 100, and hence `field-b` is given field number 101.

## Annotation of Schema Paths

Transformed protobuf messages have a different structure to the input YANG
schema. For example, additional layers of hierarchy are introduced to support
lists, and identiifers are transformed from YANG-compatible to
protobuf-compatible. In order that the original schema path of an entity can be
determined, the schema is annotated with the original schema path.

This annotation uses a protobuf `FieldOption` defined in
[yext.proto](https://github.com/openconfig/ygot/blob/master/proto/yext/yext.proto).
The format of the option is a string specifying the complete YANG schema tree
path (e.g., `/interfaces/interface/config/name`). In the case that a particular
single protobuf field maps to more than one leaf in the YANG schema (possible
where certain kinds of transformations, or compressions of the schema are used)
then multiple schema tree paths are separated by the `|` character.

## Annotation of Enum Values

When YANG enumerated types (`enumeration`, `identityref` or `union` or `typedef`
nodes referencing these types) are output to Protobuf, their names are
transformed to comply with the Protobuf style guide. Particularly, enum names
are transformed to `CamelCase`, and their values are transformed to be
`UPPERCASE_WITH_UNDERSCORES`. This results in the name of the output enumeration
differing from that which is used in the YANG model from which the Protobuf was
generated.

To allow the original YANG enumeration value label to be retrieved from the
Protobuf that is output, the `EnumValueOptions` field of the the Protobuf
descriptor is extended to add a `yang_name` string value. Each enumeration value
that has a name within the YANG schema is annotated with the string name of the
original value, if requested by code generation.

## Encoding of Anydata

Anydata nodes in a YANG schema can be used to embed arbitrary, opaque data into
a schema. In the mapping to Protobuf, these are represented as
`google.protobuf.Any` messages. Such messages can be used to embed the contents
of any other protobuf message into the schema, and are defined in [the Proto3
documentation](https://developers.google.com/protocol-buffers/docs/proto3#any).
