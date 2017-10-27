// Copyright 2017 Google Inc.
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

package ygot

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io/ioutil"

	"github.com/openconfig/goyang/pkg/yang"
)

// GzipToSchema takes an input byte slice, and returns it as
// a map of yang.Entry nodes, keyed by the name of the struct that
// the yang.Entry describes the schema for.
func GzipToSchema(gzj []byte) (map[string]*yang.Entry, error) {
	gzr, err := gzip.NewReader(bytes.NewReader(gzj))
	if err != nil {
		return nil, err
	}
	defer gzr.Close()

	s, err := ioutil.ReadAll(gzr)
	if err != nil {
		return nil, err
	}

	root := &yang.Entry{}
	if err := json.Unmarshal(s, &root); err != nil {
		return nil, err
	}

	schema := map[string]*yang.Entry{}
	rebuildSchemaMap(root, nil, schema)
	return schema, nil
}

// rebuildSchemaMap takes an input yang.Entry and appends it to the
// schema map. The key of the map is the stored name of the generated
// struct which is stored in the Annotation field of the yang.Entry when
// serialised.
func rebuildSchemaMap(e, parent *yang.Entry, schema map[string]*yang.Entry) {
	if n, ok := e.Annotation["structname"]; ok {
		if s, ok := n.(string); ok {
			schema[s] = e
		}
	}
	e.Parent = parent

	for _, ch := range e.Dir {
		rebuildSchemaMap(ch, e, schema)
	}
}
