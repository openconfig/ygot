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

// Package pathtranslate exports api to transform given string slice into
// different forms in a schema aware manner.
package pathtranslate

import (
	"fmt"
	"strings"

	"github.com/openconfig/goyang/pkg/yang"

	gnmipb "github.com/openconfig/gnmi/proto/gnmi"
)

const separator = "/"

// PathTranslator stores the rules required to rewrite a given path as gNMI PathElem.
type PathTranslator struct {
	rules map[string][]string
}

// NewPathTranslator instantiates a PathTranslator with the given slice of schemas.
// It returns an error if any of the keyed list schemas have the similar full path.
func NewPathTranslator(schemaTree []*yang.Entry) (*PathTranslator, error) {
	r := &PathTranslator{
		rules: map[string][]string{},
	}
	for _, v := range schemaTree {
		if v.Key == "" {
			continue
		}
		fullPath := resolveUntilRoot(v)
		if _, ok := r.rules[fullPath]; ok {
			return nil, fmt.Errorf("got %v path multiple times", fullPath)
		}
		r.rules[fullPath] = strings.Split(v.Key, " ")
	}
	return r, nil
}

// resolveUntilRoot concatenates the schema names of the given schema and its
// ancestors. '/' is used as separator. The root schema in the tree is ignored
// as it is an artificially inserted schema.
func resolveUntilRoot(schema *yang.Entry) string {
	path := []string{}
	// The schema with nil Parent is assumed to be root schema. Root schema is't
	// appended into string slice.
	for e := schema; e.Parent != nil; e = e.Parent {
		path = append(path, e.Name)
	}
	// Append an empty string to get a concatenated string starting with "/".
	path = append(path, "")

	// Reverse the slice.
	for i := len(path)/2 - 1; i >= 0; i-- {
		o := len(path) - 1 - i
		path[i], path[o] = path[o], path[i]
	}
	return strings.Join(path, separator)
}

// PathElem receives a path as string slice and generates slice of gNMI PathElem
// based on stored rewrite rules. It returns an error if there are less elements
// following the element's name in the path than the number of keys of the list.
func (r *PathTranslator) PathElem(p []string) ([]*gnmipb.PathElem, error) {
	// Keeps track of whether element in the p slice is consumed or not.
	// When keys are consumed, they are set as true in "used" slice.
	used := make([]bool, len(p))

	// Keeps the path elements seeen so far by appending with a separator.
	var pathSoFar string

	var res []*gnmipb.PathElem
	for i := 0; i < len(p); i++ {
		// this must be a key element which was considered in prior iterations.
		if used[i] {
			continue
		}
		pathSoFar = pathSoFar + separator + p[i]

		keyNames, ok := r.rules[pathSoFar]
		// If pathSoFar isn't in rule list, this can be an arbitrary element or
		// part of the path that constitues the full path of keyed list.
		// Note that this isn't a check to decide whether arbitrary element is
		// schema compliant.
		if !ok {
			res = append(res, &gnmipb.PathElem{Name: p[i]})
			continue
		}
		keysStartPos := i + 1
		if len(keyNames) > len(p)-keysStartPos {
			return nil, fmt.Errorf("got %d, want %d keys for %s", len(p)-keysStartPos, len(keyNames), pathSoFar)
		}
		keys := map[string]string{}
		for j, k := range keyNames {
			used[keysStartPos+j] = true
			keys[k] = p[keysStartPos+j]
		}
		res = append(res, &gnmipb.PathElem{Name: p[i], Key: keys})
	}

	return res, nil
}
