# GoStruct Shadow Paths

## Introduction

GoStruct shadow paths refer to `shadow-path` annotations currently generated via
the flag `-ignore_shadow_schema_paths`:

```go
// Interface represents the /openconfig-interfaces/interfaces/interface YANG schema element.
type Interface struct {
    Description      *string                            `path:"state/description" module:"openconfig-interfaces/openconfig-interfaces" shadow-path:"config/description" shadow-module:"openconfig-interfaces/openconfig-interfaces"`
    Enabled          *bool                              `path:"state/enabled" module:"openconfig-interfaces/openconfig-interfaces" shadow-path:"config/enabled" shadow-module:"openconfig-interfaces/openconfig-interfaces"`
    Mtu              *uint16                            `path:"state/mtu" module:"openconfig-interfaces/openconfig-interfaces" shadow-path:"config/mtu" shadow-module:"openconfig-interfaces/openconfig-interfaces"`
    Name             *string                            `path:"state/name|name" module:"openconfig-interfaces/openconfig-interfaces|openconfig-interfaces" shadow-path:"config/name|name" shadow-module:"openconfig-interfaces/openconfig-interfaces|openconfig-interfaces"`
    OperStatus       E_Interface_OperStatus             `path:"state/oper-status" module:"openconfig-interfaces/openconfig-interfaces"`
}
```

The aim of this document is to clarify what shadow paths are and why they exist
as a generation option.

## Shadow paths: compressed-out "config" or "state" YANG leaves

https://datatracker.ietf.org/doc/html/draft-openconfig-netmod-opstate-01#section-2
contains a diagram of the relationship between intended config (`/config`) and
applied config (`/state`) leaves:

```
          +---------+
          |         |    transition intended
          |intended |    to applied
          | config  +---------+
          |         |         |
          +---------+         |
              ^               |         config: true
   +----------|------------------------------------+
              |               |         config: false
              |               |
              |               |
              |       +-----------------------------+
              |       |       | operational state   |
              |       |  +----v----+ +-----------+  |
              |       |  |         | |           |  |
              +       |  | applied | |  derived  |  |   operational:true
            same +------>| config  | |   state   |<-------+
            leaves    |  |         | |           |  |
                      |  |         | |           |  |
                      |  +---------+ +-----------+  |
                      +-----------------------------+
```

In this diagram, applied config are "config false", or `/state` leaves that
mirror intended config leaves, which are "config true", or `/config` leaves. Per
[design.md](design.md#openconfig-path-compression), one of these are compressed
out depending on the value of the `-prefer_operational_state` generation flag.

Shadow paths are relevant only for compressed GoStructs, and indicate the
compressed-out `/config` or `/state` YANG `leaf` nodes.

## Problems with Path Compression and how Shadow Paths Help

Path compression leads to the GoStruct that ygot generates not being able to
represent both intended config and applied config at the same time. This leads
to two problems:

1.  When subscribing to a non-leaf path, some gNMI clients want to silently
    ignore the compressed-out paths rather than erroring out due to an
    unrecognized path.
2.  Some use cases (e.g. [ygnmi](https://github.com/openconfig/ygnmi#queries))
    use the same compressed GoStruct for representing either the "config view"
    (intended config+derived state) combination, or the "state view" (applied
    config+derived state) combination of leaves. We want to allow switching
    between "config views" and "state views" for certain ygot utilities (e.g.
    marshalling/unmarshalling).

ygot address these issues by,

1.  Always ignoring paths that match a `shadow-path` tag when doing
    unmarshalling. For example, if a gNMI update for
    `/interfaces/interface[name="foo"]/config/mtu` is unmarshalled into the
    `Interface` GoStruct at the beginning of this documentation, then the field
    will not be populated since it is a shadow path.
2.  Supporting a `PreferShadowPath` option for some utilities (see section
    below). `PreferShadowPath` means that the "shadow" path will be used in
    preference to the "primary" path annotation.

## Preferring Shadow Paths

`PreferShadowPath` is behavioural option used to describe utilities preferring
the `shadow-path` tag instead of the `path` tag in the generated GoStructs when
they both exist on a field, and is therefore used to switch the meaning of the
GoStruct from a "config view" (intended config+derived state) to a "state view"
(applied config+derived state) or vice-versa (depending on the value of the
`-prefer_operational_state` generation flag).

For example, say we're using the utility `ytypes.SetNode` to unmarshal a gNMI
update for `/interfaces/interface[name="foo"]/config/mtu`. Recall that this
update is ignored when unmarshalled into the `Interface` GoStruct at the
beginning of this documentation. This is because ygot sees that it is a shadow
path (or alternatively, it does not exist in the "state view" of the YANG
subtree corresponding to the GoStruct). However, if the `ygot.PreferShadowPath`
option is used, then `ytypes.SetNode` will now populate the field using this
update since it prefers the shadow path (or alternatively, it exists in the
"config view" which `ygot.PreferShadowPath` indicated).
