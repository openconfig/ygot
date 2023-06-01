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

package ytestutil

import (
	"fmt"

	"github.com/google/go-cmp/cmp"
	"github.com/kr/pretty"
	"github.com/openconfig/ygot/integration_tests/schemaops/ctestschema"
	"github.com/openconfig/ygot/util"
)

var (
	PrintFieldsIterFunc = func(ni *util.NodeInfo, in, out interface{}) (errs util.Errors) {
		// Only print basic scalar values, skip everything else.
		if !util.IsValueScalar(ni.FieldValue) || util.IsValueNil(ni.FieldKey) {
			return
		}
		outs := out.(*string)
		*outs += fmt.Sprintf("%v: %v, ", ni.PathFromParent, pretty.Sprint(ni.FieldValue.Interface()))
		return
	}

	OrderedMapCmpOptions = []cmp.Option{
		cmp.AllowUnexported(
			ctestschema.OrderedList_OrderedMap{},
			ctestschema.OrderedList_OrderedList_OrderedMap{},
			ctestschema.OrderedMultikeyedList_OrderedMap{},
		),
	}
)
