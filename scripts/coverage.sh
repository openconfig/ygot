#!/bin/bash

go get github.com/go-playground/overalls && go get github.com/mattn/goveralls

overalls -project=github.com/openconfig/ygot -covermode=count -ignore=".git,vendor,integration_tests,ygot/schema_tests,ygen/schema_tests,demo,experimental/ygotutils,generator,ytypes/schema_tests"
goveralls -coverprofile=overalls.coverprofile -service travis-ci


