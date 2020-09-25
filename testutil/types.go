// Copyright 2020 Google Inc.
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

package testutil

// Binary is a type that is used for fields that have a YANG type of
// binary. It is used such that binary fields can be distinguished from
// leaf-lists of uint8s (which are mapped to []uint8, equivalent to
// []byte in reflection).
type Binary []byte

// YANGEmpty is a type that is used for fields that have a YANG type of
// empty. It is used such that empty fields can be distinguished from boolean fields
// in the generated code.
type YANGEmpty bool

// Int8 is an int8 type assignable to unions of which it is a subtype.
type Int8 int8

// Int16 is an int16 type assignable to unions of which it is a subtype.
type Int16 int16

// Int32 is an int32 type assignable to unions of which it is a subtype.
type Int32 int32

// Int64 is an int64 type assignable to unions of which it is a subtype.
type Int64 int64

// Uint8 is a uint8 type assignable to unions of which it is a subtype.
type Uint8 uint8

// Uint16 is a uint16 type assignable to unions of which it is a subtype.
type Uint16 uint16

// Uint32 is a uint32 type assignable to unions of which it is a subtype.
type Uint32 uint32

// Uint64 is a uint64 type assignable to unions of which it is a subtype.
type Uint64 uint64

// Float64 is a float64 type assignable to unions of which it is a subtype.
type Float64 float64

// String is a string type assignable to unions of which it is a subtype.
type String string

// Bool is a bool type assignable to unions of which it is a subtype.
type Bool bool

// Unsupported is an interface{} wrapper type for unsupported types. It is
// assignable to unions of which it is a subtype.
type Unsupported struct {
	Value interface{}
}

// TestUnion is an interface defined within *this* (testutil) package that is
// satisfied by a subset of the above union types to aid testing within other
// packages.
// Enumerations defined within other test packages can still satisfy this
// interface by defining an IsTestUnion() method.
type TestUnion interface {
	IsTestUnion()
}

func (String) IsTestUnion() {}
func (Int16) IsTestUnion()  {}
func (Int64) IsTestUnion()  {}
func (Binary) IsTestUnion() {}

func (String) IsUnion() {}
func (Int64) IsUnion()  {}
func (Binary) IsUnion() {}

func (String) Is_UnionLeafTypeSimple() {}
func (Uint32) Is_UnionLeafTypeSimple() {}
func (Binary) Is_UnionLeafTypeSimple() {}
