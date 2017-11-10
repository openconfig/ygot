package openconfig_simple

//go:generate sh -c "cd $GOPATH/src && protoc --proto_path=. --go_out=plugins=grpc:. github.com/openconfig/ygot/experimental/protounmarshal/testdata/simple/proto/simple.proto"
