#!/bin/bash -e

if [ -z ${SRCDIR} ]; then
   DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
   SRCDIR=${DIR}/../..
fi

GO_PATH="$(go env GOPATH)"

if [ -z ${GO_PATH} ]; then
  echo "no GOPATH defined!!!"

  exit 1
fi

GO111MODULE=off go get -u github.com/openconfig/ygot || :
GO111MODULE=off go get -u github.com/google/protobuf || :

go run ${SRCDIR}/proto_generator/protogenerator.go \
  -generate_fakeroot \
  -base_import_path="github.com/openconfig/ygot/demo/protobuf_getting_started/ribproto" \
  -path=yang -output_dir=ribproto \
  -typedef_enum_with_defmod \
  -consistent_union_enum_names \
  -enum_package_name=enums -package_name=openconfig \
  -exclude_modules=ietf-interfaces \
  yang/rib/openconfig-rib-bgp.yang

PROTO_IMPORTS=".:${GO_PATH}/src/github.com/google/protobuf/src:${GO_PATH}/src"
find ribproto -name "*.proto" | while read l; do
  protoc -I$PROTO_IMPORTS --go_out=. $l --go_opt=paths=source_relative
done
