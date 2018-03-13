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
package getterdemo

// Generate rule to create the example structs:
//go:generate go run ../../generator/generator.go -path=yang -output_file=pkg/getteroc/getteroc.go -package_name=getteroc -generate_fakeroot -fakeroot_name=root -generate_getters yang/getters-demo.yang
