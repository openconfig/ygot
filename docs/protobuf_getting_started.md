# Using ygot To Generate a Protobuf Representation of a YANG Schema
**Author**: robjs  
**Date**: October 2017

There are a number of cases in which a protobuf representation of a YANG schema is useful. Particularly:

* Where there is no native language binding generator for the language. Whilst [ygot](https://github.com/openconfig/ygot) and [pyangbind](https://github.com/robshakir/pyangbind) provide solutions for Go and Python respectively, there are limited toolkits for other languages. Use of a protobuf schema allows usable code artefacts to be generated through use of `protoc`. Protobuf supports a number of languages both [natively](https://developers.google.com/protocol-buffers/docs/reference/overview) and via [third-party plugins](https://github.com/google/protobuf/blob/master/docs/third_party.md).
* Where efficient on-the-wire encoding is required. Protobuf can be serialised efficiently to an [binary format](https://developers.google.com/protocol-buffers/docs/encoding) resulting in significantly lower data volumes than other encodings (e.g., XML, JSON).

To allow these two use cases to be met, [ygot](https://github.com/openconfig/ygot) implements transformation of a YANG schema to a set of protobuf messages. The design choices made for this transformation are described [in this document](https://github.com/openconfig/ygot/blob/master/docs/yang-to-protobuf-transformations-spec.md).

This document provides some background as to how to generate a Protobuf definition from a YANG schema. Using the [OpenConfig BGP RIB model](https://github.com/openconfig/public/tree/master/release/models/rib) as an example.

## Understanding the Output Protobuf Files

### Filesystem Hierarchy

The `ygot` package contains a `proto_generator` binary ([source](https://github.com/openconfig/ygot/tree/master/proto_generator)). This binary is used to generate the protocol buffer definitions. Rather than outputting an individual protobuf file with all message definitions within it, the generate outputs a hierarchy of files following the schema of the YANG module that is supplied.

For example,  with the following YANG module:

```yang
module simple {
  namespace "urn:s";
  prefix "s";

  container a {
    container b {
      container c {
        leaf d { type string; }
      }
    }

    container e {
      leaf f { type string; }
    }
  }
}
```

The following file hierarchy is output:

```
<output_dir>/<package_name>
<output_dir>/<package_name>/simple
<output_dir>/<package_name>/simple/a
<output_dir>/<package_name>/simple/a/a.proto
<output_dir>/<package_name>/simple/a/b
<output_dir>/<package_name>/simple/a/b/b.proto
<output_dir>/<package_name>/simple/simple.proto
```

The three generated protobuf files (`simple.proto`, `a.proto` and `b.proto`) contain the children of the element that they are named after. Therefore `a.proto` contains definitions for the schema element `/a/b`, `b.proto` contains definitions for `/a/b/c` (and `d` since it is contained within `c` and does not generate a protobuf `message`).

In the filesystem hierarchy `<output_dir>` is a directory name specified using the `output_dir` flag of the `proto_generator` binary. The `<package_name>` is specified using the `package_name` flag, and indicates the base package name that should be used for the generated protobuf file definitions.

In order to comply with constraints of some build systems (particularly `go build`), one protobuf `package` is output per filesystem directory. This allows generated code for each package to be within its own directory, and hence build systems that require packages to have a one-to-one mapping between packages and filesystem directories to be used.

### Protobuf File Contents

If we examine an individual `proto`, for example,  `simple.proto`, trimming the header the contents are as follows:

```protobuf
syntax = "proto3";

package <package_name>.simple;

import "github.com/openconfig/ygot/proto/ywrapper/ywrapper.proto";
import "github.com/openconfig/ygot/proto/yext/yext.proto";
import "<base_import_path>/<output_dir>/<package_name>/simple/a/a.proto";

// A represents the /simple/a YANG schema element.
message A {
  a.B b = 367480893 [(yext.schemapath) = "/a/b"];
  a.E e = 367480890 [(yext.schemapath) = "/a/e"];
}
```

Each `package` is named according to the base package name specified as the `package_name` flag. By default, this name is specified to be `openconfig`. There are two dependent protobuf files imported:

 * `ywrapper.proto` provides a set of wrapper messages for basic protobuf types, allowing a caller to be able to distinguish whether a field was set. The path to `ywrapper.proto` can be modified using the `ywrapper_path` command-line flag.
 * `yext.proto` provides a set of extensions to the base protobuf descriptors to be able to add YANG-specific annotations. This is used to annotate schema paths (e.g., the `/a/b` annotation in the example above), and information relating to YANG identifiers (e.g., enumerated value names) to the output protobuf.

Both `yext.proto` and `ywrapper.proto` default to being imported from the `ygot` GitHub repository.

The package then imports any child packages which define its children. In this example, since the YANG container `a` has children `b` and `e`, the `a.proto` which defines the *children* of `a` is imported. The path to this import consists of a number of parts:

* `<base_import_path>` -- this path is specified through the `base_import_path` flag. It specifies the path that should be used to search for imports. This can be used to set the output directory to some relative path to a particular import path that is supplied to `protoc`.
* `<output_dir>` and `<package_name` are specified as per the filesystem descriptions above.

## Generating `openconfig-bgp-rib` Protobufs

This example walks through the generation of Protobuf files for the `openconfig-rib-bgp` model. Example code can be found in [`demo/protobuf_getting_started`](https://github.com/openconfig/ygot/tree/master/demo/protobuf_getting_started).

`ygot` has some external dependencies for the full Protobuf generation toolchain. Starting from a new Go environment, the following dependencies are required. 

* A copy of the `protoc` compiler (available from [github.com/google/protobuf](https://github.com/google/protobuf)) is required to build generated code for protobuf. `protoc` should be installed and available on the current environment's `PATH`.
* If generating Go code, as per this example, the `proto-gen-go` plugin is required. This can be installed using `go get -u github.com/golang/protobuf/protoc-gen-go`.
* A copy of the `ygot` `proto_generator` binary is required. Most simply, this can be installed through:
  * Retrieving ygot: `go get -u github.com/openconfig/ygot`
  * Installing ygot dependencies: `cd $GOPATH/src/github.com/openconfig/ygot && go get -t -d ./...`
 
After these dependencies are met, the following command generates the example protobufs from the vendored OpenConfig BGP RIB model: 

```
go run $GOPATH/src/github.com/openconfig/ygot/proto_generator/protogenerator.go \
  -generate_fakeroot \
  -base_import_path="github.com/openconfig/ygot/demo/protobuf_getting_started/ribproto" \
  -path=yang -output_dir=ribproto \
  -package_name=openconfig -enum_package_name=enums \
  yang/rib/openconfig-rib-bgp.yang
```

In this command:

 * `-generate_fakeroot` creates a root level container `message` which contains all elements at the root. By default, this message is called `Device`. It can be renamed using the `fakeroot_name` command-line flag.
 * `-base_import_path` (as described above) specifies the import path that should be used in the generated protobufs. The path used in this example specifies the entire path from `$GOPATH/src`, since this will be the include path supplied to `protoc`.
 * `-path` specifies the search path(s) that should be used to find dependencies of the input YANG modules. Multiple directories can be separated with a comma.
 * `-output_dir` specifies the directory into which the output files for the schema should be written.
 * `-package_name` (as described above) specifies the name of the top-level package that should be created for the output schema.
 * `-enum_package` specifies the name that should be used for the package which stores global enumerated values. Such values are generated for:
 	* `identity` statements in the YANG schema.
 	* `typedef` statements which contain an `enumeration`.
 * Finally, the `yang/rib/openconfig-rib-bgp.yang` argument specifies the YANG schema for which the protobufs should be generated.

Running this command outputs the generated set of protobufs to the specified directory, in this case `demo/protobuf_getting_started/ribproto`.

In order to generate code for these protobufs, we run the `protoc` compiler. When running `protoc` we must specify import paths that allow the include paths specified in the protobufs generated to be resolved. Since `-base_import_path` was specified to be the path to the GitHub `ygot` repo, we can simply set the import path to `$GOPATH/src`. 

Additionally, since the generated protobufs depend upon the `yext.proto` and `ywrapper.proto` files, generated code is required for these files. The code for these files can be generated using:

```
go get -u github.com/google/protobuf  # Ensure referenced protobufs are downloaded
cd $GOPATH/src/github.com/openconfig/ygot/proto/yext && go generate
cd $GOPATH/src/github.com/openconfig/ygot/proto/ywrapper && go generate
```

Finally, to generate the code for the the protobufs, generated we can simply loop:

```
proto_imports=".:${GOPATH}/src/github.com/google/protobuf/src:${GOPATH}/src"
find $GOPATH/src/github.com/openconfig/ygot/demo/protobuf_getting_started/ribproto -name "*.proto" | while read l; do
  protoc -I=$proto_imports --go_out=. $l
done
```

This leaves us with a generated `.pb.go` file for each `.proto` that was generated by `ygot`.

## Using the Generated Protobufs in a Go program




