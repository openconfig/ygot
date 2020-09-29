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

// UnionInt8 is an int8 type assignable to unions of which it is a subtype.
type UnionInt8 int8

// UnionInt16 is an int16 type assignable to unions of which it is a subtype.
type UnionInt16 int16

// UnionInt32 is an int32 type assignable to unions of which it is a subtype.
type UnionInt32 int32

// UnionInt64 is an int64 type assignable to unions of which it is a subtype.
type UnionInt64 int64

// UnionUint8 is a uint8 type assignable to unions of which it is a subtype.
type UnionUint8 uint8

// UnionUint16 is a uint16 type assignable to unions of which it is a subtype.
type UnionUint16 uint16

// UnionUint32 is a uint32 type assignable to unions of which it is a subtype.
type UnionUint32 uint32

// UnionUint64 is a uint64 type assignable to unions of which it is a subtype.
type UnionUint64 uint64

// UnionFloat64 is a float64 type assignable to unions of which it is a subtype.
type UnionFloat64 float64

// UnionString is a string type assignable to unions of which it is a subtype.
type UnionString string

// UnionBool is a bool type assignable to unions of which it is a subtype.
type UnionBool bool

// UnionUnsupported is an interface{} wrapper type for unsupported types. It is
// assignable to unions of which it is a subtype.
type UnionUnsupported struct {
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

type TestUnion2 interface {
	IsTestUnion2()
}

func (UnionString) IsTestUnion() {}
func (UnionInt16) IsTestUnion()  {}
func (UnionInt64) IsTestUnion()  {}
func (Binary) IsTestUnion()      {}

func (UnionString) IsUnion() {}
func (UnionInt64) IsUnion()  {}
func (Binary) IsUnion()      {}

func (UnionInt16) IsTestUnion2() {}
func (UnionInt64) IsTestUnion2() {}
func (Binary) IsTestUnion2()     {}
func (UnionBool) IsTestUnion2()  {}

func (UnionString) Is_UnionLeafTypeSimple() {}
func (UnionUint32) Is_UnionLeafTypeSimple() {}
func (Binary) Is_UnionLeafTypeSimple()      {}

func (UnionString) IsExampleUnion()       {}
func (UnionFloat64) IsExampleUnion()      {}
func (UnionInt64) IsExampleUnion()        {}
func (UnionBool) IsExampleUnion()         {}
func (YANGEmpty) IsExampleUnion()         {}
func (Binary) IsExampleUnion()            {}
func (*UnionUnsupported) IsExampleUnion() {}

func (*UnionUnsupported) IsU() {}