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

package ytypes

import (
	"fmt"
	"reflect"
	"regexp"
	"sync"
	"unicode/utf8"

	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/util"
)

// Refer to: https://tools.ietf.org/html/rfc6020#section-9.4.

var (
	// posixRegexCache stores previously-compiled Regexp objects.
	// This helps the performance of validation of, say, a large prefix
	// list that have the same pattern specification.
	//
	// Concurrency Requirements
	//
	// Only the regexp cache has to be protected by mutexes, since
	// a Regexp is safe for concurrent use by multiple goroutines:
	// https://golang.org/src/regexp/regexp.go
	posixRegexCache = map[string]*regexp.Regexp{}
	posixRegexMutex = &sync.RWMutex{}
	re2RegexCache   = map[string]*regexp.Regexp{}
	re2RegexMutex   = &sync.RWMutex{}
)

// compilePattern returns the compiled regex for the given regex
// pattern. It caches previous look-ups for faster performance.
// Go's regexp implementation might be relatively slow compared to other
// languages: https://github.com/golang/go/issues/11646
func compilePattern(pattern string, isPOSIX bool) (*regexp.Regexp, error) {
	regexCache := re2RegexCache
	regexMutex := re2RegexMutex
	regexCompile := regexp.Compile
	if isPOSIX {
		regexCache = posixRegexCache
		regexCompile = regexp.CompilePOSIX
		regexMutex = posixRegexMutex
	}

	regexMutex.RLock()
	r, ok := regexCache[pattern]
	regexMutex.RUnlock()
	if !ok {
		var err error
		r, err = regexCompile(pattern)
		if err != nil {
			return nil, err
		}
		// It's true that there may be multiple writers into the map at
		// the same time. This, however, doesn't impact correctness,
		// since any compiled Regexp objects are acceptable, and
		// displaced objects are simply dropped at the end of this
		// function.
		regexMutex.Lock()
		regexCache[pattern] = r
		regexMutex.Unlock()
	}
	return r, nil
}

// ValidateStringRestrictions checks that the given string matches the string
// schema's length and pattern restrictions (if any). It returns an error if
// the validation fails.
func ValidateStringRestrictions(schemaType *yang.YangType, stringVal string) error {
	// Check that the length is within the allowed range.
	allowedRanges := schemaType.Length
	strLen := uint64(utf8.RuneCountInString(stringVal))
	if !lengthOk(allowedRanges, strLen) {
		return fmt.Errorf("length %d is outside range %v", strLen, allowedRanges)
	}

	// Check that the value satisfies any regex patterns.
	patterns, isPOSIX := util.SanitizedPattern(schemaType)
	for _, p := range patterns {
		r, err := compilePattern(p, isPOSIX)
		if err != nil {
			return err
		}
		if !r.MatchString(stringVal) {
			return fmt.Errorf("%q does not match regular expression pattern %q", stringVal, r)
		}
	}
	return nil
}

// validateString validates value, which must be a Go string type, against the
// given schema.
func validateString(schema *yang.Entry, value interface{}) error {
	// Check that the schema itself is valid.
	if err := validateStringSchema(schema); err != nil {
		return err
	}

	vv := reflect.ValueOf(value)

	// Check that type of value is the type expected from the schema.
	if vv.Kind() != reflect.String {
		return fmt.Errorf("non string type %T with value %v for schema %s", value, value, schema.Name)
	}

	// This value could be a union typedef string, so convert it to make
	// sure it's the primitive string type.
	stringVal := vv.Convert(reflect.TypeOf("")).Interface().(string)

	if err := ValidateStringRestrictions(schema.Type, stringVal); err != nil {
		return fmt.Errorf("schema %q: %v", schema.Name, err)
	}
	return nil
}

// validateStringSlice validates value, which must be a Go string slice type,
// against the given schema.
func validateStringSlice(schema *yang.Entry, value interface{}) error {
	// Check that the schema itself is valid.
	if err := validateStringSchema(schema); err != nil {
		return err
	}

	// Check that type of value is the type expected from the schema.
	slice, ok := value.([]string)
	if !ok {
		return fmt.Errorf("non []string type %T with value %v for schema %s", value, value, schema.Name)
	}

	// Each slice element must be valid and unique.
	tbl := make(map[string]bool, len(slice))
	for i, val := range slice {
		if err := validateString(schema, val); err != nil {
			return fmt.Errorf("invalid element at index %d: %v for schema %s", i, err, schema.Name)
		}
		if tbl[val] {
			return fmt.Errorf("duplicate string: %q for schema %s", val, schema.Name)
		}
		tbl[val] = true
	}
	return nil
}

// validateStringSchema validates the given string type schema. This is a quick
// check rather than a comprehensive validation against the RFC.
// It is assumed that such a validation is done when the schema is parsed from
// source YANG.
func validateStringSchema(schema *yang.Entry) error {
	if schema == nil {
		return fmt.Errorf("string schema is nil")
	}
	if schema.Type == nil {
		return fmt.Errorf("string schema %s Type is nil", schema.Name)
	}
	if schema.Type.Kind != yang.Ystring {
		return fmt.Errorf("string schema %s has wrong type %v", schema.Name, schema.Type.Kind)
	}

	patterns, isPOSIX := util.SanitizedPattern(schema.Type)
	for _, p := range patterns {
		if _, err := compilePattern(p, isPOSIX); err != nil {
			return fmt.Errorf("error generating regexp %s %v for schema %s", p, err, schema.Name)
		}
	}

	return validateLengthSchema(schema)
}
