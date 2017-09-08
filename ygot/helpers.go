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
	"reflect"
)

// String takes a string argument and returns a pointer to it.
func String(s string) *string { return &s }

// Uint8 takes an uint8 argument and returns a pointer to it.
func Uint8(u uint8) *uint8 { return &u }

// Uint16 takes an uint16 argument and returns a pointer to it.
func Uint16(u uint16) *uint16 { return &u }

// Uint32 takes an uint32 argument and returns a pointer to it.
func Uint32(u uint32) *uint32 { return &u }

// Uint64 takes an uint64 argument and returns a pointer to it.
func Uint64(u uint64) *uint64 { return &u }

// Int8 takes an int8 argument and returns a pointer to it.
func Int8(i int8) *int8 { return &i }

// Int16 takes an int16 argument and returns a pointer to it.
func Int16(i int16) *int16 { return &i }

// Int32 takes an int32 argument and returns a pointer to it.
func Int32(i int32) *int32 { return &i }

// Int64 takes an int64 argument and returns a pointer to it.
func Int64(i int64) *int64 { return &i }

// Bool takes a boolean argument and returns a pointer to it.
func Bool(b bool) *bool { return &b }

// Float32 takes a float32 argument and returns a pointer to it.
func Float32(f float32) *float32 { return &f }

// Float64 takes a float64 argument and returns a pointer to it.
func Float64(f float64) *float64 { return &f }

// ToPtr returns a pointer to v.
func ToPtr(v interface{}) interface{} {
	n := reflect.New(reflect.TypeOf(v))
	n.Elem().Set(reflect.ValueOf(v))
	return n.Interface()
}
