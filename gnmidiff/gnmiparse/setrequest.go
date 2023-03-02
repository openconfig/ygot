// Copyright 2023 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// gnmiparse contains utilities for parsing the textproto of gNMI messages.
package gnmiparse

import (
	"os"

	"google.golang.org/protobuf/encoding/prototext"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
)

// SetRequestFromFile parses a SetRequest from a textproto file.
func SetRequestFromFile(file string) (*gpb.SetRequest, error) {
	sr := &gpb.SetRequest{}
	bs, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	if err := prototext.Unmarshal(bs, sr); err != nil {
		return nil, err
	}

	return sr, nil
}
