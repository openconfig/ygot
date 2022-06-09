package main

import (
	"bytes"
	"fmt"
	"go/parser"
	"go/token"
	"path/filepath"
	"testing"

	"github.com/openconfig/ygot/genutil"
	"github.com/openconfig/ygot/gogen"
	"github.com/openconfig/ygot/ygen"
)

// Simple test case that ensures that the end-to-end ygot pipeline works
// correctly. This is a smoke-test for the ygot package.

// The path to this directory within the test package.
var TestRoot string

func TestGenerateCode(t *testing.T) {
	tests := []struct {
		name     string
		inIROpts ygen.IROptions
		inGoOpts gogen.GoOpts
		inFiles  []string
		inPaths  []string
	}{{
		name: "openconfig interfaces",
		inIROpts: ygen.IROptions{
			ParseOptions: ygen.ParseOpts{
				ExcludeModules: []string{"ietf-interfaces"},
			},
			TransformationOptions: ygen.TransformationOpts{
				CompressBehaviour: genutil.PreferIntendedConfig,
				GenerateFakeRoot:  true,
			},
		},
		inGoOpts: gogen.GoOpts{
			GenerateJSONSchema:   true,
			GenerateSimpleUnions: true,
		},
		inFiles: []string{
			filepath.Join(TestRoot, "yang", "openconfig-interfaces.yang"),
			filepath.Join(TestRoot, "yang", "openconfig-if-ip.yang"),
		},
		inPaths: []string{filepath.Join(TestRoot, "yang")},
	}, {
		name: "openconfig interfaces with no compression",
		inIROpts: ygen.IROptions{
			ParseOptions: ygen.ParseOpts{
				ExcludeModules: []string{"ietf-interfaces"},
			},
			TransformationOptions: ygen.TransformationOpts{
				GenerateFakeRoot: true,
			},
		},
		inGoOpts: gogen.GoOpts{
			GenerateJSONSchema:   true,
			GenerateSimpleUnions: true,
		},
		inFiles: []string{
			filepath.Join(TestRoot, "yang", "openconfig-interfaces.yang"),
			filepath.Join(TestRoot, "yang", "openconfig-if-ip.yang"),
		},
		inPaths: []string{filepath.Join(TestRoot, "yang")},
	}}

	for _, tt := range tests {
		cg := gogen.New("", tt.inIROpts, tt.inGoOpts)
		got, err := cg.Generate(tt.inFiles, tt.inPaths)
		if err != nil {
			t.Errorf("%s: Generate(%v, %v): Config: %v, got unexpected error: %v", tt.name, tt.inFiles, tt.inPaths, tt.inIROpts, err)
			continue
		}

		var b bytes.Buffer
		fmt.Fprintf(&b, got.CommonHeader)
		fmt.Fprintf(&b, got.OneOffHeader)
		for _, s := range got.Structs {
			fmt.Fprintf(&b, s.String())
		}

		for _, e := range got.Enums {
			fmt.Fprintf(&b, e)
		}

		fmt.Fprintf(&b, got.EnumMap)
		fmt.Fprintf(&b, got.JSONSchemaCode)

		// Parse the generated code using the Go parser and check whether any errors
		// are returned.
		fset := token.NewFileSet()
		if _, err := parser.ParseFile(fset, "", b.String(), parser.AllErrors); err != nil {
			t.Errorf("%s: could not parse generated Go code: %v\n\n%s", tt.name, err, b.String())
		}
	}
}
