# ygot Makefile
#
# This makefile is used by GitHub Actions CI to run tests against the ygot library.
#
test:
	go test ./...
generate:
	go generate ./demo/getting_started
	go generate ./proto/ywrapper
	go generate ./proto/yext
	go generate ./demo/uncompressed
	go generate ./demo/protobuf_getting_started
	go generate ./integration_tests/uncompressed
	go generate ./integration_tests/annotations/apb
	go generate ./integration_tests/annotations/proto2apb
clean:
	rm -f demo/getting_started/pkg/ocdemo/oc.go
	rm -f demo/uncompressed/pkg/demo/uncompressed.go
install: deps generate
all: clean deps generate test
