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

## Compressed-out "config" or "state" YANG leaves

Shadow paths are relevant only for compressed GoStructs, and indicate the
compressed-out `/config` or `/state` YANG `leaf` nodes.

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

## Problem with Path Compression

Path compression, however, leads to the GoStruct not being able to represent
both intended config and applied config at the same time, but for certain use
cases (e.g. [ygnmi](https://github.com/openconfig/ygnmi#queries)), it is
desirable to use the same GoStruct for representing either the (intended
config+derived state) combination, as well as the (applied config+derived state)
combination of leaves. This is where shadow paths can help.

## Preferring Shadow Paths

This term is used to describe utilities preferring the `shadow-path` tag instead
of the `path` tag in the generated GoStructs when they both exist on a field,
and is therefore used to switch the meaning of the GoStruct from a "config view"
(intended config+derived state) to a "state view" (applied config+derived
state).

For example, if a gNMI update for `/interfaces/interface[name="foo"]/state/mtu`
is unmarshalled into the `Interface` GoStruct at the beginning of this
documentation with `ygot.SetNode`/`ygot.PreferShadowPath`, then the update would
be ignored, and the field will not be populated. This is because it doesn't
exist in the "config view" which `ygot.PreferShadowPath` indicated.
