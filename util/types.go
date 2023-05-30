// Copyright 2021 Google Inc.
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

package util

import (
	"fmt"

	"github.com/openconfig/goyang/pkg/yang"
)

// YangIntTypeBits returns the number of bits for a YANG int type.
// It returns an error if the type is not an int type.
func YangIntTypeBits(t yang.TypeKind) (int, error) {
	switch t {
	case yang.Yint8, yang.Yuint8:
		return 8, nil
	case yang.Yint16, yang.Yuint16:
		return 16, nil
	case yang.Yint32, yang.Yuint32:
		return 32, nil
	case yang.Yint64, yang.Yuint64:
		return 64, nil
	}
	return 0, fmt.Errorf("type is not an int")
}
