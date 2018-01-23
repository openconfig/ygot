// Copyright 2018 Google Inc.
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

// Package diff provides functions that are use to diff two ygot.GoStructs.
package diff

import (
	"fmt"

	"github.com/openconfig/ygot/util"
	"github.com/openconfig/ygot/ygot"

	gnmipb "github.com/openconfig/gnmi/proto/gnmi"
)

// Diff returns the difference between the original and modified ygot.GoStruct,
// which must be of the same type, as a gNMI Notification message.
func Diff(original, modified ygot.GoStruct) (*gnmipb.Notification, error) {

	findSetIterFunc := func(ni *util.NodeInfo, in, out interface{}) util.Errors {
		fmt.Printf("%v -> %v\n", ni.PathFromParent, ni.StructField)
		return nil
	}

	return nil, nil
}
