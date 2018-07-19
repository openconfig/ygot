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

// Package cc is a ygot integration test for choice/cases.
package cc

//go:generate go run ../../generator/generator.go -path=yang -output_file=pkg/cctest/gencc.go -package_name=cctest -generate_fakeroot -fakeroot_name=root yang/cctest.yang
