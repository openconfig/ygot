package gogen

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/openconfig/gnmi/errdiff"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/genutil"
	"github.com/openconfig/ygot/testutil"
	"github.com/openconfig/ygot/ygen"
)

const (
	// TestRoot is the root of the test directory such that this is not
	// repeated when referencing files.
	TestRoot string = ""
	// deflakeRuns specifies the number of runs of code generation that
	// should be performed to check for flakes.
	deflakeRuns int = 10
)

var updateGolden = flag.Bool("update_golden", false, "Update golden files")

//go:generate go test -run TestSimpleStructs -args -update_golden

// yangTestCase describs a test case for which code generation is performed
// through Goyang's API, it provides the input set of parameters in a way that
// can be reused across tests.
type yangTestCase struct {
	name                string        // Name is the identifier for the test.
	inFiles             []string      // inFiles is the set of inputFiles for the test.
	inIncludePaths      []string      // inIncludePaths is the set of paths that should be searched for imports.
	inExcludeModules    []string      // inExcludeModules is the set of modules that should be excluded from code generation.
	inConfig            CodeGenerator // inConfig specifies the configuration that should be used for the generator test case.
	wantStructsCodeFile string        // wantsStructsCodeFile is the path of the generated Go code that the output of the test should be compared to.
	wantErrSubstring    string        // wantErrSubstring specifies whether the test should expect an error.
	wantSchemaFile      string        // wantSchemaFile is the path to the schema JSON that the output of the test should be compared to.
}

// TestSimpleStructs tests the processModules, Generate and writeGoCode
// functions. It takes the set of YANG modules described in the slice of
// yangTestCases and generates the struct code for them, comparing the output
// to the wantStructsCodeFile.  In order to simplify the files that are used,
// the Generate structs are concatenated before comparison with the
// expected output. If the generated code matches the expected output, it is
// run against the Go parser to ensure that the code is valid Go - this is
// expected, but it ensures that the input file does not contain Go which is
// invalid.
func TestSimpleStructs(t *testing.T) {
	tests := []yangTestCase{{
		name:    "simple openconfig test, with compression, with (useless) enum org name trimming",
		inFiles: []string{filepath.Join(datapath, "openconfig-simple.yang")},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					CompressBehaviour:                    genutil.PreferIntendedConfig,
					ShortenEnumLeafNames:                 true,
					EnumOrgPrefixesToTrim:                []string{"openconfig"},
					UseDefiningModuleForTypedefEnumNames: true,
					EnumerationsUseUnderscores:           true,
				},
			},
			GoOptions: GoOpts{
				GenerateSimpleUnions:    true,
				GenerateLeafGetters:     true,
				GenerateLeafSetters:     true,
				GeneratePopulateDefault: true,
			},
		},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata/structs/openconfig-simple.formatted-txt"),
	}, {
		name:    "simple openconfig test, with excluded state, with compression, with enum org name trimming",
		inFiles: []string{filepath.Join(datapath, "openconfig-simple.yang")},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					CompressBehaviour:                    genutil.ExcludeDerivedState,
					ShortenEnumLeafNames:                 true,
					EnumOrgPrefixesToTrim:                []string{"openconfig"},
					UseDefiningModuleForTypedefEnumNames: true,
					EnumerationsUseUnderscores:           true,
				},
			},
			GoOptions: GoOpts{
				GenerateSimpleUnions: true,
				GenerateLeafGetters:  true,
			},
		},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata/structs/openconfig-simple-excludestate.formatted-txt"),
	}, {
		name:    "simple openconfig test, with no compression",
		inFiles: []string{filepath.Join(datapath, "openconfig-simple.yang")},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					ShortenEnumLeafNames:                 true,
					UseDefiningModuleForTypedefEnumNames: true,
					EnumerationsUseUnderscores:           true,
				},
			},
			GoOptions: GoOpts{
				GenerateSimpleUnions:    true,
				GenerateLeafGetters:     true,
				GeneratePopulateDefault: true,
			},
		},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata/structs/openconfig-simple-no-compress.formatted-txt"),
	}, {
		name:    "simple openconfig test, with compression, without shortened enum leaf names, with enum org name trimming",
		inFiles: []string{filepath.Join(datapath, "openconfig-simple.yang")},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					CompressBehaviour:                    genutil.PreferIntendedConfig,
					EnumOrgPrefixesToTrim:                []string{"openconfig"},
					UseDefiningModuleForTypedefEnumNames: true,
					EnumerationsUseUnderscores:           true,
				},
			},
			GoOptions: GoOpts{
				GenerateSimpleUnions: true,
			},
		},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata/structs/openconfig-simple.long-enum-names.trimmed-enum.formatted-txt"),
	}, {
		name:    "simple openconfig test, with no compression, with enum org name trimming",
		inFiles: []string{filepath.Join(datapath, "openconfig-simple.yang")},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					ShortenEnumLeafNames:                 true,
					EnumOrgPrefixesToTrim:                []string{"openconfig"},
					UseDefiningModuleForTypedefEnumNames: true,
					EnumerationsUseUnderscores:           true,
				},
			},
			GoOptions: GoOpts{
				GenerateSimpleUnions: true,
			},
		},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata/structs/openconfig-simple-no-compress.trimmed-enum.formatted-txt"),
	}, {
		name:    "simple openconfig test with unsupported statements, don't tolerate",
		inFiles: []string{filepath.Join(datapath, "openconfig-simple-with-unsupported.yang")},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					CompressBehaviour:                    genutil.PreferIntendedConfig,
					ShortenEnumLeafNames:                 true,
					EnumOrgPrefixesToTrim:                []string{"openconfig"},
					UseDefiningModuleForTypedefEnumNames: true,
					EnumerationsUseUnderscores:           true,
				},
			},
			GoOptions: GoOpts{
				GenerateSimpleUnions:    true,
				GenerateLeafGetters:     true,
				GeneratePopulateDefault: true,
			},
		},
		wantErrSubstring: "unsupported statement type (Notification)",
	}, {
		name:    "simple openconfig test with unsupported statements, tolerate",
		inFiles: []string{filepath.Join(datapath, "openconfig-simple-with-unsupported.yang")},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				ParseOptions: ygen.ParseOpts{
					IgnoreUnsupportedStatements: true,
				},
				TransformationOptions: ygen.TransformationOpts{
					CompressBehaviour:                    genutil.PreferIntendedConfig,
					ShortenEnumLeafNames:                 true,
					EnumOrgPrefixesToTrim:                []string{"openconfig"},
					UseDefiningModuleForTypedefEnumNames: true,
					EnumerationsUseUnderscores:           true,
				},
			},
			GoOptions: GoOpts{
				GenerateSimpleUnions:    true,
				GenerateLeafGetters:     true,
				GeneratePopulateDefault: true,
			},
		},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata/structs/openconfig-simple-with-unsupported.formatted-txt"),
	}, {
		name:    "simple openconfig test with deviate not-supported",
		inFiles: []string{filepath.Join(datapath, "deviate-not-supported.yang")},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					CompressBehaviour:                    genutil.PreferIntendedConfig,
					ShortenEnumLeafNames:                 true,
					EnumOrgPrefixesToTrim:                []string{"openconfig"},
					UseDefiningModuleForTypedefEnumNames: true,
					EnumerationsUseUnderscores:           true,
				},
			},
			GoOptions: GoOpts{
				GenerateSimpleUnions:    true,
				GenerateLeafGetters:     true,
				GenerateLeafSetters:     true,
				GeneratePopulateDefault: true,
			},
		},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata/structs/deviate-not-supported.formatted-txt"),
	}, {
		name:    "simple openconfig test with deviate not-supported ignored",
		inFiles: []string{filepath.Join(datapath, "deviate-not-supported.yang")},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				ParseOptions: ygen.ParseOpts{
					YANGParseOptions: yang.Options{
						DeviateOptions: yang.DeviateOptions{
							IgnoreDeviateNotSupported: true,
						},
					},
				},
				TransformationOptions: ygen.TransformationOpts{
					CompressBehaviour:                    genutil.PreferIntendedConfig,
					ShortenEnumLeafNames:                 true,
					EnumOrgPrefixesToTrim:                []string{"openconfig"},
					UseDefiningModuleForTypedefEnumNames: true,
					EnumerationsUseUnderscores:           true,
				},
			},
			GoOptions: GoOpts{
				GenerateSimpleUnions:    true,
				GenerateLeafGetters:     true,
				GenerateLeafSetters:     true,
				GeneratePopulateDefault: true,
			},
		},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata/structs/deviate-not-supported-keep.formatted-txt"),
	}, {
		name:    "OpenConfig leaf-list defaults test, with compression",
		inFiles: []string{filepath.Join(datapath, "openconfig-leaflist-default.yang")},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					CompressBehaviour:                    genutil.PreferIntendedConfig,
					ShortenEnumLeafNames:                 true,
					UseDefiningModuleForTypedefEnumNames: true,
					EnumerationsUseUnderscores:           true,
				},
			},
			GoOptions: GoOpts{
				GenerateSimpleUnions:    true,
				GenerateLeafGetters:     true,
				GenerateLeafSetters:     true,
				GeneratePopulateDefault: true,
			},
		},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata/structs/openconfig-leaflist-default.formatted-txt"),
	}, {
		name:    "OpenConfig schema test - with annotations",
		inFiles: []string{filepath.Join(datapath, "openconfig-simple.yang")},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					ShortenEnumLeafNames:                 true,
					UseDefiningModuleForTypedefEnumNames: true,
					EnumerationsUseUnderscores:           true,
				},
			},
			GoOptions: GoOpts{
				AddAnnotationFields:  true,
				AnnotationPrefix:     "ᗩ",
				GenerateSimpleUnions: true,
			},
		},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata", "structs", "openconfig-simple-annotations.formatted-txt"),
	}, {
		name:    "OpenConfig schema test - list and associated method (rename, new)",
		inFiles: []string{filepath.Join(datapath, "openconfig-withlist.yang")},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					CompressBehaviour:                    genutil.PreferIntendedConfig,
					ShortenEnumLeafNames:                 true,
					UseDefiningModuleForTypedefEnumNames: true,
					EnumerationsUseUnderscores:           true,
				},
			},
			GoOptions: GoOpts{
				GenerateRenameMethod: true,
				GenerateSimpleUnions: true,
			},
		},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata/structs/openconfig-withlist.formatted-txt"),
	}, {
		name:    "OpenConfig schema test - list and associated method (rename, new) - using operational state",
		inFiles: []string{filepath.Join(datapath, "openconfig-withlist.yang")},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					CompressBehaviour:                    genutil.PreferOperationalState,
					ShortenEnumLeafNames:                 true,
					UseDefiningModuleForTypedefEnumNames: true,
					EnumerationsUseUnderscores:           true,
				},
			},
			GoOptions: GoOpts{
				GenerateRenameMethod: true,
				GenerateSimpleUnions: true,
			},
		},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata/structs/openconfig-withlist-opstate.formatted-txt"),
	}, {
		name:    "OpenConfig schema test - list and associated method (rename, new) - using unordered list",
		inFiles: []string{filepath.Join(datapath, "openconfig-withlist.yang")},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					CompressBehaviour:                    genutil.PreferIntendedConfig,
					ShortenEnumLeafNames:                 true,
					UseDefiningModuleForTypedefEnumNames: true,
					EnumerationsUseUnderscores:           true,
				},
			},
			GoOptions: GoOpts{
				GenerateRenameMethod:                true,
				GenerateSimpleUnions:                true,
				GenerateOrderedListsAsUnorderedMaps: true,
			},
		},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata/structs/openconfig-withlist-unordered.formatted-txt"),
	}, {
		name:    "OpenConfig schema test - multi-keyed list key struct name conflict and associated method (rename, new)",
		inFiles: []string{filepath.Join(datapath, "openconfig-multikey-list-name-conflict.yang")},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					CompressBehaviour:                    genutil.PreferIntendedConfig,
					ShortenEnumLeafNames:                 true,
					UseDefiningModuleForTypedefEnumNames: true,
					EnumerationsUseUnderscores:           true,
				},
			},
			GoOptions: GoOpts{
				GenerateRenameMethod: true,
				GenerateSimpleUnions: true,
			},
		},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata/structs/openconfig-multikey-list-name-conflict.formatted-txt"),
	}, {
		name:    "simple openconfig test, with a list that has an enumeration key",
		inFiles: []string{filepath.Join(datapath, "openconfig-list-enum-key.yang")},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					CompressBehaviour:                    genutil.PreferIntendedConfig,
					ShortenEnumLeafNames:                 true,
					UseDefiningModuleForTypedefEnumNames: true,
					EnumerationsUseUnderscores:           true,
				},
			},
			GoOptions: GoOpts{
				GenerateSimpleUnions:    true,
				IgnoreShadowSchemaPaths: true,
			},
		},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata/structs/openconfig-list-enum-key.formatted-txt"),
	}, {
		name:    "simple openconfig test, with a list that has an enumeration key, with enum org name trimming",
		inFiles: []string{filepath.Join(datapath, "openconfig-list-enum-key.yang")},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					CompressBehaviour:                    genutil.PreferIntendedConfig,
					ShortenEnumLeafNames:                 true,
					EnumOrgPrefixesToTrim:                []string{"openconfig"},
					UseDefiningModuleForTypedefEnumNames: true,
					EnumerationsUseUnderscores:           true,
				},
			},
			GoOptions: GoOpts{
				GenerateSimpleUnions: true,
			},
		},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata/structs/openconfig-list-enum-key.trimmed-enum.formatted-txt"),
	}, {
		name:    "openconfig test with a identityref union",
		inFiles: []string{filepath.Join(datapath, "openconfig-unione.yang")},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					CompressBehaviour:                    genutil.PreferIntendedConfig,
					ShortenEnumLeafNames:                 true,
					UseDefiningModuleForTypedefEnumNames: true,
					EnumerationsUseUnderscores:           true,
				},
			},
			GoOptions: GoOpts{
				GenerateSimpleUnions: true,
			},
		},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata/structs/openconfig-unione.formatted-txt"),
	}, {
		name:    "openconfig test with a identityref union (wrapper unions)",
		inFiles: []string{filepath.Join(datapath, "openconfig-unione.yang")},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					CompressBehaviour:                    genutil.PreferIntendedConfig,
					ShortenEnumLeafNames:                 true,
					UseDefiningModuleForTypedefEnumNames: true,
					EnumerationsUseUnderscores:           true,
				},
			},
		},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata/structs/openconfig-unione.wrapper-unions.formatted-txt"),
	}, {
		name:    "openconfig tests with fakeroot",
		inFiles: []string{filepath.Join(datapath, "openconfig-fakeroot.yang")},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					CompressBehaviour:                    genutil.PreferIntendedConfig,
					GenerateFakeRoot:                     true,
					ShortenEnumLeafNames:                 true,
					UseDefiningModuleForTypedefEnumNames: true,
					EnumerationsUseUnderscores:           true,
				},
			},
			GoOptions: GoOpts{
				GenerateSimpleUnions: true,
			},
		},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata/structs/openconfig-fakeroot.formatted-txt"),
	}, {
		name:    "openconfig noncompressed tests with fakeroot",
		inFiles: []string{filepath.Join(datapath, "openconfig-fakeroot.yang")},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					GenerateFakeRoot:                     true,
					ShortenEnumLeafNames:                 true,
					UseDefiningModuleForTypedefEnumNames: true,
					EnumerationsUseUnderscores:           true,
				},
			},
			GoOptions: GoOpts{
				GenerateSimpleUnions: true,
			},
		},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata/structs/openconfig-fakeroot-nc.formatted-txt"),
	}, {
		name:    "schema test with compression",
		inFiles: []string{filepath.Join(TestRoot, "testdata/schema/openconfig-options.yang")},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					CompressBehaviour:                    genutil.PreferIntendedConfig,
					ShortenEnumLeafNames:                 true,
					UseDefiningModuleForTypedefEnumNames: true,
					EnumerationsUseUnderscores:           true,
				},
			},
			GoOptions: GoOpts{
				GenerateJSONSchema:   true,
				GenerateSimpleUnions: true,
			},
		},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata/schema/openconfig-options-compress.formatted-txt"),
		wantSchemaFile:      filepath.Join(TestRoot, "testdata/schema/openconfig-options-compress-schema.json"),
	}, {
		name:    "schema test without compression",
		inFiles: []string{filepath.Join(TestRoot, "testdata/schema/openconfig-options.yang")},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					EnumerationsUseUnderscores: true,
				},
			},
			GoOptions: GoOpts{
				GenerateJSONSchema:   true,
				GenerateSimpleUnions: true,
			},
		},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata/schema/openconfig-options-nocompress.formatted-txt"),
		wantSchemaFile:      filepath.Join(TestRoot, "testdata/schema/openconfig-options-nocompress-schema.json"),
	}, {
		name:    "schema test with fakeroot",
		inFiles: []string{filepath.Join(TestRoot, "testdata/schema/openconfig-options.yang")},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					CompressBehaviour:                    genutil.PreferIntendedConfig,
					GenerateFakeRoot:                     true,
					ShortenEnumLeafNames:                 true,
					UseDefiningModuleForTypedefEnumNames: true,
					EnumerationsUseUnderscores:           true,
				},
			},
			GoOptions: GoOpts{
				GenerateJSONSchema:   true,
				GenerateSimpleUnions: true,
			},
		},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata/schema/openconfig-options-compress-fakeroot.formatted-txt"),
		wantSchemaFile:      filepath.Join(TestRoot, "testdata/schema/openconfig-options-compress-fakeroot-schema.json"),
	}, {
		name:    "schema test with fakeroot and no compression",
		inFiles: []string{filepath.Join(TestRoot, "testdata/schema/openconfig-options.yang")},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					GenerateFakeRoot:                     true,
					UseDefiningModuleForTypedefEnumNames: true,
					EnumerationsUseUnderscores:           true,
				},
			},
			GoOptions: GoOpts{
				GenerateJSONSchema:   true,
				GenerateSimpleUnions: true,
			},
		},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata/schema/openconfig-options-nocompress-fakeroot.formatted-txt"),
		wantSchemaFile:      filepath.Join(TestRoot, "testdata/schema/openconfig-options-nocompress-fakeroot-schema.json"),
	}, {
		name:    "schema test with camelcase annotations",
		inFiles: []string{filepath.Join(datapath, "openconfig-camelcase.yang")},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					CompressBehaviour:                    genutil.PreferIntendedConfig,
					GenerateFakeRoot:                     true,
					ShortenEnumLeafNames:                 true,
					UseDefiningModuleForTypedefEnumNames: true,
					EnumerationsUseUnderscores:           true,
				},
			},
			GoOptions: GoOpts{
				GenerateSimpleUnions: true,
			},
		},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata/structs/openconfig-camelcase-compress.formatted-txt"),
	}, {
		name:    "structs test with camelcase annotations",
		inFiles: []string{filepath.Join(datapath, "openconfig-enumcamelcase.yang")},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					CompressBehaviour:                    genutil.PreferIntendedConfig,
					ShortenEnumLeafNames:                 true,
					UseDefiningModuleForTypedefEnumNames: true,
					EnumerationsUseUnderscores:           true,
				},
			},
			GoOptions: GoOpts{
				GenerateSimpleUnions: true,
			},
		},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata/structs/openconfig-enumcamelcase-compress.formatted-txt"),
	}, {
		name:                "structs test with choices and cases",
		inFiles:             []string{filepath.Join(datapath, "choice-case-example.yang")},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata/structs/choice-case-example.formatted-txt"),
	}, {
		name: "module with augments",
		inFiles: []string{
			filepath.Join(datapath, "openconfig-simple-target.yang"),
			filepath.Join(datapath, "openconfig-simple-augment.yang"),
		},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					CompressBehaviour:          genutil.PreferIntendedConfig,
					GenerateFakeRoot:           true,
					EnumerationsUseUnderscores: true,
				},
			},
			GoOptions: GoOpts{
				GenerateSimpleUnions: true,
			},
		},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata/structs/openconfig-augmented.formatted-txt"),
	}, {
		name: "module with conflicting augments",
		inFiles: []string{
			filepath.Join(datapath, "openconfig-simple-target.yang"),
			filepath.Join(datapath, "openconfig-simple-augment.yang"),
			filepath.Join(datapath, "openconfig-simple-augment-conflict.yang"),
		},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					CompressBehaviour:          genutil.PreferIntendedConfig,
					GenerateFakeRoot:           true,
					EnumerationsUseUnderscores: true,
				},
			},
			GoOptions: GoOpts{
				GenerateSimpleUnions: true,
			},
		},
		wantErrSubstring: `Duplicate node "foo" in "target"`,
	}, {
		name:    "variable and import explicitly specified",
		inFiles: []string{filepath.Join(datapath, "openconfig-simple.yang")},
		inConfig: CodeGenerator{
			Caller: "testcase",
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					CompressBehaviour:                    genutil.PreferIntendedConfig,
					GenerateFakeRoot:                     true,
					ShortenEnumLeafNames:                 true,
					UseDefiningModuleForTypedefEnumNames: true,
					FakeRootName:                         "fakeroot",
					EnumerationsUseUnderscores:           true,
				},
			},
			GoOptions: GoOpts{
				GenerateJSONSchema:   true,
				SchemaVarName:        "YANGSchema",
				GoyangImportPath:     "foo/goyang",
				YgotImportPath:       "bar/ygot",
				YtypesImportPath:     "baz/ytypes",
				GenerateSimpleUnions: true,
			},
		},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata/schema/openconfig-options-explicit.formatted-txt"),
		wantSchemaFile:      filepath.Join(TestRoot, "testdata/schema/openconfig-options-explicit-schema.json"),
	}, {
		name:    "module with entities at the root",
		inFiles: []string{filepath.Join(datapath, "root-entities.yang")},
		inConfig: CodeGenerator{
			Caller: "testcase",
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					FakeRootName:               "fakeroot",
					GenerateFakeRoot:           true,
					EnumerationsUseUnderscores: true,
				},
			},
			GoOptions: GoOpts{
				GenerateSimpleUnions: true,
			},
		},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata/structs/root-entities.formatted-txt"),
	}, {
		name:                "module with empty leaf",
		inFiles:             []string{filepath.Join(datapath, "empty.yang")},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata/structs/empty.formatted-txt"),
	}, {
		name:             "module with excluded modules",
		inFiles:          []string{filepath.Join(datapath, "excluded-module.yang")},
		inExcludeModules: []string{"excluded-module-two"},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					GenerateFakeRoot:           true,
					FakeRootName:               "office",
					EnumerationsUseUnderscores: true,
				},
			},
			GoOptions: GoOpts{
				GenerateSimpleUnions: true,
			},
		},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata/structs/excluded-module.formatted-txt"),
	}, {
		name:    "module with excluded config false",
		inFiles: []string{filepath.Join(datapath, "", "openconfig-config-false.yang")},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					CompressBehaviour:          genutil.UncompressedExcludeDerivedState,
					GenerateFakeRoot:           true,
					EnumerationsUseUnderscores: true,
				},
			},
			GoOptions: GoOpts{
				GenerateSimpleUnions: true,
			},
		},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata", "structs", "openconfig-config-false-uncompressed.formatted-txt"),
	}, {
		name:    "module with excluded config false - with compression",
		inFiles: []string{filepath.Join(datapath, "", "openconfig-config-false.yang")},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					GenerateFakeRoot:           true,
					CompressBehaviour:          genutil.ExcludeDerivedState,
					EnumerationsUseUnderscores: true,
				},
			},
			GoOptions: GoOpts{
				GenerateSimpleUnions: true,
			},
		},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata", "structs", "openconfig-config-false-compressed.formatted-txt"),
	}, {
		name:    "module with getters, delete and append methods",
		inFiles: []string{filepath.Join(datapath, "", "openconfig-list-enum-key.yang")},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					GenerateFakeRoot:           true,
					EnumerationsUseUnderscores: true,
				},
			},
			GoOptions: GoOpts{
				GenerateAppendMethod: true,
				GenerateGetters:      true,
				GenerateDeleteMethod: true,
				GenerateSimpleUnions: true,
			},
		},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata", "structs", "openconfig-list-enum-key.getters-append.formatted-txt"),
	}, {
		name:    "module with excluded state, with RO list, path compression on",
		inFiles: []string{filepath.Join(datapath, "", "exclude-state-ro-list.yang")},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					GenerateFakeRoot:           true,
					CompressBehaviour:          genutil.ExcludeDerivedState,
					EnumerationsUseUnderscores: true,
				},
			},
			GoOptions: GoOpts{
				GenerateSimpleUnions: true,
			},
		},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata", "structs", "exclude-state-ro-list.formatted-txt"),
	}, {
		name:           "different union enumeration types",
		inFiles:        []string{filepath.Join(datapath, "", "enum-union.yang")},
		inIncludePaths: []string{filepath.Join(datapath, "modules")},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					CompressBehaviour:                    genutil.PreferIntendedConfig,
					ShortenEnumLeafNames:                 true,
					UseDefiningModuleForTypedefEnumNames: true,
					EnumerationsUseUnderscores:           true,
				},
			},
			GoOptions: GoOpts{
				GenerateSimpleUnions: true,
				GenerateLeafGetters:  true,
				GenerateLeafSetters:  true,
			},
		},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata", "structs", "enum-union.formatted-txt"),
	}, {
		name:           "different union enumeration types with consistent naming for union-inlined enums",
		inFiles:        []string{filepath.Join(datapath, "", "enum-union.yang")},
		inIncludePaths: []string{filepath.Join(datapath, "modules")},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					CompressBehaviour:                    genutil.PreferIntendedConfig,
					ShortenEnumLeafNames:                 true,
					UseDefiningModuleForTypedefEnumNames: true,
					EnumerationsUseUnderscores:           true,
				},
			},
			GoOptions: GoOpts{
				GenerateSimpleUnions:                true,
				GenerateLeafGetters:                 true,
				AppendEnumSuffixForSimpleUnionEnums: true,
			},
		},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata", "structs", "enum-union.consistent.formatted-txt"),
	}, {
		name:           "different union enumeration types with default enum values",
		inFiles:        []string{filepath.Join(datapath, "", "enum-union-with-enum-defaults.yang")},
		inIncludePaths: []string{filepath.Join(datapath, "modules")},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					CompressBehaviour:                    genutil.PreferIntendedConfig,
					ShortenEnumLeafNames:                 true,
					UseDefiningModuleForTypedefEnumNames: true,
					EnumerationsUseUnderscores:           true,
				},
			},
			GoOptions: GoOpts{
				GenerateSimpleUnions:                true,
				GenerateLeafGetters:                 true,
				GeneratePopulateDefault:             true,
				AppendEnumSuffixForSimpleUnionEnums: true,
			},
		},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata", "structs", "enum-union-with-enum-defaults.formatted-txt"),
	}, {
		name:           "different union enumeration types with default enum values (wrapper union)",
		inFiles:        []string{filepath.Join(datapath, "", "enum-union-with-enum-defaults.yang")},
		inIncludePaths: []string{filepath.Join(datapath, "modules")},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					CompressBehaviour:                    genutil.PreferIntendedConfig,
					ShortenEnumLeafNames:                 true,
					UseDefiningModuleForTypedefEnumNames: true,
					EnumerationsUseUnderscores:           true,
				},
			},
			GoOptions: GoOpts{
				GenerateLeafGetters:                 true,
				GenerateLeafSetters:                 true,
				GeneratePopulateDefault:             true,
				AppendEnumSuffixForSimpleUnionEnums: true,
			},
		},
		wantErrSubstring: "default value not supported for wrapper union values, please generate using simplified union leaves",
	}, {
		name:           "enumeration behaviour - resolution across submodules and grouping re-use within union",
		inFiles:        []string{filepath.Join(datapath, "", "enum-module.yang")},
		inIncludePaths: []string{filepath.Join(datapath, "modules")},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					CompressBehaviour:                    genutil.PreferIntendedConfig,
					ShortenEnumLeafNames:                 true,
					UseDefiningModuleForTypedefEnumNames: true,
					EnumerationsUseUnderscores:           true,
				},
			},
			GoOptions: GoOpts{
				GenerateSimpleUnions: true,
				GenerateLeafGetters:  true,
				GenerateLeafSetters:  true,
			},
		},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata", "structs", "enum-module.formatted-txt"),
	}, {
		name:           "enumeration behaviour (wrapper unions) - resolution across submodules and grouping re-use within union",
		inFiles:        []string{filepath.Join(datapath, "", "enum-module.yang")},
		inIncludePaths: []string{filepath.Join(datapath, "modules")},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					CompressBehaviour:                    genutil.PreferIntendedConfig,
					ShortenEnumLeafNames:                 true,
					UseDefiningModuleForTypedefEnumNames: true,
					EnumerationsUseUnderscores:           true,
				},
			},
			GoOptions: GoOpts{
				GenerateLeafGetters: true,
			},
		},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata", "structs", "enum-module.wrapper-unions.formatted-txt"),
	}, {
		name:           "enumeration behaviour - resolution across submodules and grouping re-use within union, with enumeration leaf names not shortened",
		inFiles:        []string{filepath.Join(datapath, "", "enum-module.yang")},
		inIncludePaths: []string{filepath.Join(datapath, "modules")},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					CompressBehaviour:                    genutil.PreferIntendedConfig,
					UseDefiningModuleForTypedefEnumNames: true,
					EnumerationsUseUnderscores:           true,
				},
			},
			GoOptions: GoOpts{
				GenerateSimpleUnions: true,
			},
		},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata", "structs", "enum-module.long-enum-names.formatted-txt"),
	}, {
		name:           "enumeration behaviour - resolution across submodules and grouping re-use within union, with typedef enum names being prefixed by the module of their use/residence rather than of their definition",
		inFiles:        []string{filepath.Join(datapath, "", "enum-module.yang")},
		inIncludePaths: []string{filepath.Join(datapath, "modules")},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					CompressBehaviour:          genutil.PreferIntendedConfig,
					ShortenEnumLeafNames:       true,
					EnumerationsUseUnderscores: true,
				},
			},
			GoOptions: GoOpts{
				GenerateSimpleUnions: true,
			},
		},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata", "structs", "enum-module.residing-module-typedef-enum-name.formatted-txt"),
	}, {
		name:           "enumeration behaviour - resolution across submodules and grouping re-use within union, with typedef enum names being prefixed by the module of their use/residence rather than of their definition, and enumeration leaf names not shortened",
		inFiles:        []string{filepath.Join(datapath, "", "enum-module.yang")},
		inIncludePaths: []string{filepath.Join(datapath, "modules")},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					CompressBehaviour:          genutil.PreferIntendedConfig,
					EnumerationsUseUnderscores: true,
				},
			},
			GoOptions: GoOpts{
				GenerateSimpleUnions: true,
			},
		},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata", "structs", "enum-module.long-enum-names.residing-module-typedef-enum-name.formatted-txt"),
	}, {
		name:           "enumeration behaviour - resolution across submodules and grouping re-use within union, with typedef enum names being prefixed by the module of their use/residence rather than of their definition, and enumeration leaf names not shortened",
		inFiles:        []string{filepath.Join(datapath, "", "enum-module.yang")},
		inIncludePaths: []string{filepath.Join(datapath, "modules")},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					CompressBehaviour:          genutil.PreferIntendedConfig,
					EnumerationsUseUnderscores: true,
				},
			},
		},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata", "structs", "enum-module.long-enum-names.residing-module-typedef-enum-name.wrapper-unions.formatted-txt"),
	}, {
		name:           "enumeration behaviour - multiple enumerations within a union",
		inFiles:        []string{filepath.Join(datapath, "", "enum-multi-module.yang")},
		inIncludePaths: []string{filepath.Join(datapath, "modules")},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					CompressBehaviour:                    genutil.PreferIntendedConfig,
					ShortenEnumLeafNames:                 true,
					UseDefiningModuleForTypedefEnumNames: true,
					EnumerationsUseUnderscores:           true,
				},
			},
			GoOptions: GoOpts{
				GenerateJSONSchema:   true,
				GenerateSimpleUnions: true,
			},
		},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata", "structs", "enum-multi-module.formatted-txt"),
	}, {
		name:    "module with leaf getters",
		inFiles: []string{filepath.Join(datapath, "", "openconfig-list-enum-key.yang")},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					GenerateFakeRoot:                     true,
					ShortenEnumLeafNames:                 true,
					UseDefiningModuleForTypedefEnumNames: true,
					CompressBehaviour:                    genutil.PreferIntendedConfig,
					EnumerationsUseUnderscores:           true,
				},
			},
			GoOptions: GoOpts{
				GenerateLeafGetters:     true,
				GeneratePopulateDefault: true,
				GenerateSimpleUnions:    true,
			},
		},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata", "structs", "openconfig-list-enum-key.leaf-getters.formatted-txt"),
	}, {
		name:    "module with leaf setters",
		inFiles: []string{filepath.Join(datapath, "", "openconfig-list-enum-key.yang")},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					GenerateFakeRoot:                     true,
					ShortenEnumLeafNames:                 true,
					UseDefiningModuleForTypedefEnumNames: true,
					CompressBehaviour:                    genutil.PreferIntendedConfig,
					EnumerationsUseUnderscores:           true,
				},
			},
			GoOptions: GoOpts{
				GenerateLeafSetters:     true,
				GeneratePopulateDefault: true,
				GenerateSimpleUnions:    true,
			},
		},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata", "structs", "openconfig-list-enum-key.leaf-setters.formatted-txt"),
	}, {
		name:    "uncompressed module with two different enums",
		inFiles: []string{filepath.Join(datapath, "", "enum-list-uncompressed.yang")},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					GenerateFakeRoot:           true,
					EnumerationsUseUnderscores: true,
				},
			},
			GoOptions: GoOpts{
				GenerateSimpleUnions: true,
			},
		},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata", "structs", "enum-list-uncompressed.formatted-txt"),
	}, {
		name:    "uncompressed module with two different enums (wrapper unions)",
		inFiles: []string{filepath.Join(datapath, "", "enum-list-uncompressed.yang")},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					GenerateFakeRoot:           true,
					EnumerationsUseUnderscores: true,
				},
			},
		},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata", "structs", "enum-list-uncompressed.wrapper-unions.formatted-txt"),
	}, {
		name:    "with model data",
		inFiles: []string{filepath.Join(datapath, "", "openconfig-versioned-mod.yang")},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					GenerateFakeRoot:           true,
					CompressBehaviour:          genutil.PreferIntendedConfig,
					EnumerationsUseUnderscores: true,
				},
			},
			GoOptions: GoOpts{
				IncludeModelData:     true,
				GenerateSimpleUnions: true,
			},
		},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata", "structs", "openconfig-versioned-mod.formatted-txt"),
	}, {
		name:    "model with deduplicated enums",
		inFiles: []string{filepath.Join(datapath, "enum-duplication.yang")},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					GenerateFakeRoot:           true,
					EnumerationsUseUnderscores: true,
				},
			},
			GoOptions: GoOpts{
				GenerateSimpleUnions: true,
			},
		},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata", "structs", "enum-duplication-dedup.formatted-txt"),
	}, {
		name:    "model with enums that are in the same grouping duplicated",
		inFiles: []string{filepath.Join(datapath, "enum-duplication.yang")},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					GenerateFakeRoot:           true,
					EnumerationsUseUnderscores: true,
					SkipEnumDeduplication:      true,
				},
				ParseOptions: ygen.ParseOpts{},
			},
			GoOptions: GoOpts{
				GenerateSimpleUnions: true,
			},
		},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata", "structs", "enum-duplication-dup.formatted-txt"),
	}, {
		name:    "OpenConfig schema test - list with binary key",
		inFiles: []string{filepath.Join(datapath, "openconfig-binary-list.yang")},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					CompressBehaviour:                    genutil.PreferIntendedConfig,
					ShortenEnumLeafNames:                 true,
					UseDefiningModuleForTypedefEnumNames: true,
					EnumerationsUseUnderscores:           true,
				},
			},
			GoOptions: GoOpts{
				GenerateRenameMethod: true,
				GenerateSimpleUnions: true,
			},
		},
		wantErrSubstring: "has a binary key",
	}, {
		name:    "OpenConfig schema test - multi-keyed list with binary key",
		inFiles: []string{filepath.Join(datapath, "openconfig-binary-multi-list.yang")},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					CompressBehaviour:                    genutil.PreferIntendedConfig,
					ShortenEnumLeafNames:                 true,
					UseDefiningModuleForTypedefEnumNames: true,
					EnumerationsUseUnderscores:           true,
				},
			},
			GoOptions: GoOpts{
				GenerateRenameMethod: true,
				GenerateSimpleUnions: true,
			},
		},
		wantErrSubstring: "has a binary key",
	}, {
		name:    "OpenConfig schema test - list with union key containing binary",
		inFiles: []string{filepath.Join(datapath, "openconfig-union-binary-list.yang")},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					CompressBehaviour:                    genutil.PreferIntendedConfig,
					ShortenEnumLeafNames:                 true,
					UseDefiningModuleForTypedefEnumNames: true,
					EnumerationsUseUnderscores:           true,
				},
			},
			GoOptions: GoOpts{
				GenerateRenameMethod: true,
				GenerateSimpleUnions: true,
			},
		},
		wantErrSubstring: "has a union key containing a binary",
	}, {
		name:    "module with presence containers",
		inFiles: []string{filepath.Join(datapath, "presence-container-example.yang")},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					GenerateFakeRoot:           true,
					FakeRootName:               "device",
					EnumerationsUseUnderscores: true,
				},
			},
			GoOptions: GoOpts{
				GenerateSimpleUnions:    true,
				GenerateLeafGetters:     true,
				GeneratePopulateDefault: true,
				AddYangPresence:         true,
			},
		},
		wantStructsCodeFile: filepath.Join(TestRoot, "testdata/structs/presence-container-example.formatted-txt"),
	}, {
		name:    "skip obsolete list keys test",
		inFiles: []string{filepath.Join(datapath, "skip-obsolete-test.yang")},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					SkipObsolete: true,
				},
			},
			GoOptions: GoOpts{
				GenerateSimpleUnions: true,
			},
		},
		wantErrSubstring: "did not find type for key obsolete-key",
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			genCode := func() (*GeneratedCode, string, map[string]interface{}, error) {
				// Set defaults within the supplied configuration for these tests.
				if tt.inConfig.Caller == "" {
					// Set the name of the caller explicitly to avoid issues when
					// the unit tests are called by external test entities.
					tt.inConfig.Caller = "codegen-tests"
				}
				tt.inConfig.IROptions.ParseOptions.ExcludeModules = tt.inExcludeModules

				cg := New(tt.inConfig.Caller, tt.inConfig.IROptions, tt.inConfig.GoOptions)

				gotGeneratedCode, errs := cg.Generate(tt.inFiles, tt.inIncludePaths)
				var err error
				if len(errs) > 0 {
					err = fmt.Errorf("%w", errs)
				}
				if diff := errdiff.Substring(err, tt.wantErrSubstring); diff != "" {
					t.Fatalf("%s: cg.GenerateCode(%v, %v): Config: %+v, Did not get expected error: %s", tt.name, tt.inFiles, tt.inIncludePaths, tt.inConfig, diff)
				}
				if err != nil {
					return nil, "", nil, err
				}

				// Write all the received structs into a single file such that
				// it can be compared to the received file.
				var gotCode bytes.Buffer
				fmt.Fprint(&gotCode, gotGeneratedCode.CommonHeader)
				fmt.Fprint(&gotCode, gotGeneratedCode.OneOffHeader)
				for _, gotStruct := range gotGeneratedCode.Structs {
					fmt.Fprint(&gotCode, gotStruct.String())
				}

				for _, gotEnum := range gotGeneratedCode.Enums {
					fmt.Fprint(&gotCode, gotEnum)
				}

				// Write generated enumeration map out.
				fmt.Fprint(&gotCode, gotGeneratedCode.EnumMap)

				var gotJSON map[string]interface{}
				if tt.inConfig.GoOptions.GenerateJSONSchema {
					// Write the schema byte array out.
					fmt.Fprint(&gotCode, gotGeneratedCode.JSONSchemaCode)
					fmt.Fprint(&gotCode, gotGeneratedCode.EnumTypeMap)

					if err := json.Unmarshal(gotGeneratedCode.RawJSONSchema, &gotJSON); err != nil {
						t.Fatalf("%s: json.Unmarshal(..., %v), could not unmarshal received JSON: %v", tt.name, gotGeneratedCode.RawJSONSchema, err)
					}
				}
				return gotGeneratedCode, gotCode.String(), gotJSON, nil
			}

			gotGeneratedCode, gotCode, gotJSON, err := genCode()
			if err != nil {
				return
			}

			if tt.wantSchemaFile != "" {
				if *updateGolden {
					if err := os.WriteFile(tt.wantSchemaFile, []byte(string(gotGeneratedCode.RawJSONSchema)), 0644); err != nil {
						panic(err)
					}
				}

				wantSchema, rferr := os.ReadFile(tt.wantSchemaFile)
				if rferr != nil {
					t.Fatalf("%s: os.ReadFile(%q) error: %v", tt.name, tt.wantSchemaFile, rferr)
				}

				var wantJSON map[string]interface{}
				if err := json.Unmarshal(wantSchema, &wantJSON); err != nil {
					t.Fatalf("%s: json.Unmarshal(..., [contents of %s]), could not unmarshal golden JSON file: %v", tt.name, tt.wantSchemaFile, err)
				}

				if !cmp.Equal(gotJSON, wantJSON) {
					diff, _ := testutil.GenerateUnifiedDiff(string(wantSchema), string(gotGeneratedCode.RawJSONSchema))
					t.Fatalf("%s: Generate(%v, %v), Config: %+v, did not return correct JSON (file: %v), diff: \n%s", tt.name, tt.inFiles, tt.inIncludePaths, tt.inConfig, tt.wantSchemaFile, diff)
				}
			}

			if *updateGolden {
				if err := os.WriteFile(tt.wantStructsCodeFile, []byte(gotCode), 0644); err != nil {
					t.Fatal(err)
				}
			}

			wantCodeBytes, rferr := os.ReadFile(tt.wantStructsCodeFile)
			if rferr != nil {
				t.Fatalf("%s: os.ReadFile(%q) error: %v", tt.name, tt.wantStructsCodeFile, rferr)
			}

			wantCode := string(wantCodeBytes)

			if gotCode != wantCode {
				// Use difflib to generate a unified diff between the
				// two code snippets such that this is simpler to debug
				// in the test output.
				diff, _ := testutil.GenerateUnifiedDiff(wantCode, gotCode)
				t.Errorf("%s: Generate(%v, %v), Config: %+v, did not return correct code (file: %v), diff:\n%s",
					tt.name, tt.inFiles, tt.inIncludePaths, tt.inConfig, tt.wantStructsCodeFile, diff)
			}

			for i := 0; i < deflakeRuns; i++ {
				_, gotAttempt, _, _ := genCode()
				if gotAttempt != gotCode {
					diff, _ := testutil.GenerateUnifiedDiff(gotAttempt, gotCode)
					t.Fatalf("flaky code generation, diff:\n%s", diff)
				}
			}
		})
	}
}
