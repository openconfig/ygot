#!/bin/bash

clean() {
  rm -rf public
  rm -rf deps
}

if [ -z ${SRCDIR} ]; then
   SRCDIR=$GOPATH/src/github.com/openconfig/ygot
fi

# Ensure that the .pb.go has been generated for the extensions
# that are required.
(cd ${SRCDIR}/proto/yext && SRCDIR=${SRCDIR} go generate)
(cd ${SRCDIR}/proto/ywrapper && SRCDIR=${SRCDIR} go generate)

clean

go run ${SRCDIR}/proto_generator/protogenerator.go \
  -generate_fakeroot \
  -base_import_path="github.com/openconfig/ygot/demo/protobuf_getting_started/ribproto" \
  -path=yang -output_dir=ribproto \
  -enum_package_name=enums -package_name=openconfig \
  -exclude_modules=ietf-interfaces \
  yang/rib/openconfig-rib-bgp.yang

go get -u github.com/google/protobuf
proto_imports=".:${SRCDIR}/../../../../src/github.com/google/protobuf/src:${SRCDIR}/../../../../src"
find ribproto -name "*.proto" | while read l; do
  protoc -I=$proto_imports --go_out=. $l
done

clean
