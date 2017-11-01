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
rm -rf ribproto

git clone https://github.com/openconfig/public.git
mkdir deps
cp ../getting_started/yang/{ietf,iana}* deps
go run ../../proto_generator/protogenerator.go \
  -generate_fakeroot \
   -base_import_path="github.com/openconfig/ygot/demo/protobuf_getting_started/ribproto" \
  -path=public,deps -output_dir=ribproto \
  -enum_package_name=enums -package_name=openconfig \
  -exclude_modules=ietf-interfaces \
  public/release/models/rib/openconfig-rib-bgp.yang

proto_imports=".:${GOPATH}/src/github.com/google/protobuf/src:${GOPATH}/src"
find ribproto -name "*.proto" | while read l; do
  protoc -I=$proto_imports --go_out=. $l
done

clean
