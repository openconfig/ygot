#!/bin/bash

# Copyright 2023 Google Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# To remove the dependency on GOPATH, we locally cache the protobufs that
# we need as dependencies during build time with the intended paths.

if [ -z $SRCDIR ]; then
	THIS_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
	SRC_DIR=${THIS_DIR}/../../..
fi

cd ${SRC_DIR}
protoc -I${SRC_DIR} -I ${SRC_DIR}/../../.. --go_out=. --go_opt=paths=source_relative ${SRC_DIR}/protomap/integration_tests/testdata/gribi_aft/gribi_aft.proto
protoc -I${SRC_DIR} -I ${SRC_DIR}/../../.. --go_out=. --go_opt=paths=source_relative ${SRC_DIR}/protomap/integration_tests/testdata/gribi_aft/enums/enums.proto
