#!/bin/bash

clean() {
  rm -rf public
  rm -rf deps
}

# Ensure that the .pb.go has been generated for the extensions
# that are required.
(cd ../../proto/yext && go generate)
(cd ../../proto/ywrapper && go generate)

clean

go run ../../proto_generator/protogenerator.go \
  -generate_fakeroot \
  -base_import_path="github.com/openconfig/ygot/demo/protobuf_getting_started/ribproto" \
  -path=yang -output_dir=ribproto \
  -enum_package_name=enums -package_name=openconfig \
  -exclude_modules=ietf-interfaces \
  yang/rib/openconfig-rib-bgp.yang

go get -u github.com/google/protobuf
proto_imports=".:${GOPATH}/src/github.com/google/protobuf/src:${GOPATH}/src"
find ribproto -name "*.proto" | while read l; do
  protoc -I=$proto_imports --go_out=. $l
done

clean
