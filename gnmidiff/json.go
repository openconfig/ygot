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

package gnmidiff

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/openconfig/ygot/ygot"
)

// flattenOCJSON outputs all leaf path-value pairs in the root per RFC7951.
//
// It assumes the input JSON is with respect to OpenConfig-compliant YANG.
// See the following for checking compliance:
// * https://github.com/openconfig/oc-pyang
// * https://github.com/openconfig/public/blob/master/doc/openconfig_style_guide.md
//
// Output path format is per gNMI's path conventions.
//
// e.g.
// {
//   "openconfig-network-instance:config": {
//     "description": "VRF RED",
//     "enabled": true,
//     "enabled-address-families": [
//       "openconfig-types:IPV4",
//       "openconfig-types:IPV6"
//     ],
//     "name": "RED",
//     "type": "openconfig-network-instance-types:L3VRF"
//     },
//   "openconfig-network-instance:name": "RED"
// }
//
// returns
// {
//   "openconfig-network-instance:config/description": "VRF RED",
//   "openconfig-network-instance:config/enabled": true,
//   "openconfig-network-instance:config/enabled-address-families": ["openconfig-types:IPV4", "openconfig-types:IPV6"],
//   "openconfig-network-instance:config/name": "RED",
//   "openconfig-network-instance:config/type": "openconfig-network-instance-types:L3VRF",
//   "openconfig-network-instance:name": "RED",
// }
func flattenOCJSON(json7951 []byte) (map[string]interface{}, error) {
	// TODO: Add option to remove the namespace on paths of returned updates.
	var root interface{}
	if err := json.Unmarshal(json7951, &root); err != nil {
		return nil, fmt.Errorf("gnmidiff: %v", err)
	}
	leaves := map[string]interface{}{}
	if err := flattenOCJSONAux(root, "", leaves); err != nil {
		return nil, err
	}
	return leaves, nil
}

func flattenOCJSONAux(root interface{}, path string, leaves map[string]interface{}) error {
	switch v := root.(type) {
	case bool, float64, string:
		leaves[path] = root
	case []interface{}:
		if len(v) == 0 {
			// These must be leaf-lists since you can't set a
			// list to nothing, only delete it or update descendant leaves.
			leaves[path] = root
			// If this assumption is wrong, and the list is later updated, an
			// error prefix matching can detect this invalid operation.
		} else {
			switch v[0].(type) {
			case bool, float64, string:
				leaves[path] = root
			case []interface{}:
				return fmt.Errorf("invalid RFC7951 JSON: list within a list: %v contains %v", v, v[0])
			case map[string]interface{}:
				// v is a list.
				for _, ele := range v {
					listele, ok := ele.(map[string]interface{})
					if !ok {
						return fmt.Errorf("invalid RFC7951 JSON: array has different element types: %v", v)
					}
					keyVals := map[string]string{}
					var keyNames []string
					for name, subv := range listele {
						// Here we assume that the JSON follows OpenConfig YANG style guidelines
						// and so the direct leafs MUST exactly be the list keys.
						// To keep consistent, we write them in order in the path.
						switch subsubv := subv.(type) {
						case bool, float64, string:
							var err error
							if keyVals[name], err = ygot.KeyValueAsString(subsubv); err != nil {
								return fmt.Errorf("gnmidiff cannot convert key value to string: %v", err)
							}
							keyNames = append(keyNames, name)
						}
					}
					sort.Strings(keyNames)
					var listelepath string
					for _, name := range keyNames {
						listelepath += fmt.Sprintf("[%s=%s]", name, keyVals[name])
					}
					if err := flattenOCJSONAux(listele, path+listelepath, leaves); err != nil {
						return err
					}
				}
			default:
				return fmt.Errorf("unrecognized JSON type: (%T, %v)", v[0], v[0])
			}
		}
	case map[string]interface{}:
		// This is a container or a list element.
		for subpath, subv := range v {
			if err := flattenOCJSONAux(subv, path+"/"+subpath, leaves); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unrecognized JSON type: (%T, %v)", root, root)
	}
	return nil
}
