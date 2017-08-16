# ygot Makefile
#
# This makefile is used by Travis CI to run tests against the ygot library.
#
ROOT_DIR:=$(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))

test:
	go test ./...
generate:
	cd ${ROOT_DIR}/demo/getting_started && go generate
clean:
	rm -f ${ROOT_DIR}/demo/getting_started/pkg/ocdemo/oc.go
deps:
	go get -t -d ./...
install: deps generate
all:
	clean deps generate test
