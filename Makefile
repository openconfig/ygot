# ygot Makefile
#
# This makefile is used by Travis CI to run tests against the ygot library.
#
ROOT_DIR:=$(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))

all:
	go test ./...
generate:
	cd ${ROOT_DIR}/demo/getting_started && go generate

