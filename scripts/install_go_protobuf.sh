#!/bin/bash

GIT_TAG=v1.3.4 
GP=$(go env GOPATH)
go get -u google.golang.org/protobuf/cmd/protoc-gen-go 
git -C $GOPATH/src/google.golang.org/protobuf checkout $GIT_TAG 
go install google.golang.org/protobuf/cmd/protoc-gen-go
