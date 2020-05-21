# ygot Makefile
#
# This makefile is used by Travis CI to run tests against the ygot library.
#
ROOT_DIR:=$(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))

test:
	go test ./...
generate:
	cd ${ROOT_DIR}/demo/getting_started && SRCDIR=${ROOT_DIR} go generate
	cd ${ROOT_DIR}/proto/ywrapper && SRCDIR=${ROOT_DIR} go generate
	cd $(ROOT_DIR)/proto/yext && SRCDIR=${ROOT_DIR} go generate
	cd $(ROOT_DIR)/demo/uncompressed && SRCDIR=${ROOT_DIR} go generate
	cd $(ROOT_DIR)/demo/protobuf_getting_started && SRCDIR=${ROOT_DIR} ./update.sh
	cd $(ROOT_DIR)/integration_tests/uncompressed && SRCDIR=${ROOT_DIR} go generate
	cd $(ROOT_DIR)/integration_tests/annotations/apb && SRCDIR=${ROOT_DIR} go generate
	cd $(ROOT_DIR)/integration_tests/annotations/proto2apb && SRCDIR=${ROOT_DIR} go generate
clean:
	rm -f ${ROOT_DIR}/demo/getting_started/pkg/ocdemo/oc.go
	rm -f ${ROOT_DIR}/demo/uncompressed/pkg/demo/uncompressed.go
deps:
	go get -t -d ./ygot
	go get -t -d ./ygen
	go get -t -d ./generator
	go get -t -d ./proto_generator
	go get -t -d ./exampleoc
	go get -t -d ./ytypes
	go get -t -d ./demo/gnmi_telemetry
install: deps generate
all: clean deps generate test
