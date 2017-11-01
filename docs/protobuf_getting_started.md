# Using ygot To Generate a Protobuf Representation of a YANG Schema
Author: robjs  
Date: October 2017

There are a number of cases in which a protobuf representation of a YANG schema is useful. Particularly:

* Where there is no native language binding generator for the language. Whilst [ygot](https://github.com/openconfig/ygot) and [pyangbind](https://github.com/robshakir/pyangbind) provide solutions for Go and Python respectively, there are limited toolkits for other languages. Use of a protobuf schema allows usable code artefacts to be generated through use of `protoc`. Protobuf supports a number of languages both [natively](https://developers.google.com/protocol-buffers/docs/reference/overview) and via [third-party plugins](https://github.com/google/protobuf/blob/master/docs/third_party.md).
* Where efficient on-the-wire encoding is required. Protobuf can be serialised efficiently to an [binary format](https://developers.google.com/protocol-buffers/docs/encoding) resulting in significantly lower data volumes than other encodings (e.g., XML, JSON).

To allow these two use cases to be met, [ygot](https://github.com/openconfig/ygot) implements transformation of a YANG schema to a set of protobuf messages. The design choices made for this transformation are described [in this document](https://github.com/openconfig/ygot/blob/master/docs/yang-to-protobuf-transformations-spec.md).

This document provides some background as to how to generate a Protobuf definition from a YANG schema. Using the [OpenConfig BGP RIB model](https://github.com/openconfig/public/tree/master/release/models/rib) as an example.

# 