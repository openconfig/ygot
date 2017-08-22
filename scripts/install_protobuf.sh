#!/bin/bash

# Install protoc to ensure that we can build protobuf examples.
# Based on the example at
# https://github.com/travis-ci/container-example/blob/master/install-protobuf.sh.

PROTO_URL=https://github.com/google/protobuf/releases/download/v3.4.0/protoc-3.4.0-linux-x86_64.zip
PROTO_FILE=protoc-3.4.0-linux-x86_64.zip

if [ ! -d "$HOME/protobuf" ]; then
  cd $HOME
  wget $PROTO_URL
  mkdir $HOME/protobuf
  cd $HOME/protobuf
  unzip $HOME/$PROTO_FILE
fi
