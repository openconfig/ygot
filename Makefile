# ygot Makefile
#
# This makefile is used to build and test the ygot library.
#
export GO111MODULE := on

ROOT_DIR:=$(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))

test: generate
	go test -v ./...

testrace: generate
	go test -race -v ./...

generate: deps
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

tools:
	go get -u google.golang.org/protobuf/cmd/protoc-gen-go
	go get -u golang.org/x/tools/cmd/goimports
	go get -u honnef.co/go/tools/cmd/staticcheck

deps: tools
	go mod download
	go mod tidy

install: deps generate

all: clean deps generate test

build: generate
	go build -v ./...
