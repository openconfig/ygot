package protogen

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
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

func TestGenerateProto3(t *testing.T) {
	tests := []struct {
		name           string
		inFiles        []string
		inIncludePaths []string
		inConfig       CodeGenerator
		// wantOutputFiles is a map keyed on protobuf package name with a path
		// to the file that is expected for each package.
		wantOutputFiles map[string]string
		wantErr         bool
	}{{
		name:    "simple protobuf test with compression",
		inFiles: []string{filepath.Join(TestRoot, "testdata", "proto", "proto-test-a.yang")},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					CompressBehaviour: genutil.PreferIntendedConfig,
				},
			},
		},
		wantOutputFiles: map[string]string{
			"openconfig":        filepath.Join(TestRoot, "testdata", "proto", "proto-test-a.compress.parent.formatted-txt"),
			"openconfig.parent": filepath.Join(TestRoot, "testdata", "proto", "proto-test-a.compress.parent.child.formatted-txt"),
		},
	}, {
		name:    "simple protobuf test without compression",
		inFiles: []string{filepath.Join(TestRoot, "testdata", "proto", "proto-test-a.yang")},
		wantOutputFiles: map[string]string{
			"openconfig.proto_test_a":              filepath.Join(TestRoot, "testdata", "proto", "proto-test-a.nocompress.formatted-txt"),
			"openconfig.proto_test_a.parent":       filepath.Join(TestRoot, "testdata", "proto", "proto-test-a.nocompress.parent.formatted-txt"),
			"openconfig.proto_test_a.parent.child": filepath.Join(TestRoot, "testdata", "proto", "proto-test-a.nocompress.parent.child.formatted-txt"),
		},
	}, {
		name:    "enumeration under unions test with compression",
		inFiles: []string{filepath.Join(datapath, "enum-union.yang")},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					CompressBehaviour:                    genutil.PreferIntendedConfig,
					GenerateFakeRoot:                     true,
					UseDefiningModuleForTypedefEnumNames: true,
				},
			},
			ProtoOptions: ProtoOpts{
				AnnotateEnumNames: true,
				NestedMessages:    true,
				GoPackageBase:     "github.com/foo/bar",
			},
		},
		wantOutputFiles: map[string]string{
			"openconfig":       filepath.Join(TestRoot, "testdata", "proto", "enum-union.compress.formatted-txt"),
			"openconfig.enums": filepath.Join(TestRoot, "testdata", "proto", "enum-union.compress.enums.formatted-txt"),
		},
	}, {
		name:    "yang schema with a list",
		inFiles: []string{filepath.Join(TestRoot, "testdata", "proto", "proto-test-b.yang")},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					CompressBehaviour: genutil.PreferIntendedConfig,
				},
			},
		},
		wantOutputFiles: map[string]string{
			"openconfig":        filepath.Join(TestRoot, "testdata", "proto", "proto-test-b.compress.formatted-txt"),
			"openconfig.device": filepath.Join(TestRoot, "testdata", "proto", "proto-test-b.compress.device.formatted-txt"),
		},
	}, {
		name:    "yang schema with simple enumerations",
		inFiles: []string{filepath.Join(TestRoot, "testdata", "proto", "proto-test-c.yang")},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{},
			ProtoOptions: ProtoOpts{
				GoPackageBase: "github.com/foo/baz",
			},
		},
		wantOutputFiles: map[string]string{
			"openconfig.proto_test_c":              filepath.Join(TestRoot, "testdata", "proto", "proto-test-c.proto-test-c.formatted-txt"),
			"openconfig.proto_test_c.entity":       filepath.Join(TestRoot, "testdata", "proto", "proto-test-c.proto-test-c.entity.formatted-txt"),
			"openconfig.proto_test_c.elists":       filepath.Join(TestRoot, "testdata", "proto", "proto-test-c.proto-test-c.elists.formatted-txt"),
			"openconfig.proto_test_c.elists.elist": filepath.Join(TestRoot, "testdata", "proto", "proto-test-c.proto-test-c.elists.elist.formatted-txt"),
			"openconfig.enums":                     filepath.Join(TestRoot, "testdata", "proto", "proto-test-c.enums.formatted-txt"),
		},
	}, {
		name:    "yang schema with identityref and enumerated typedef, compression off",
		inFiles: []string{filepath.Join(TestRoot, "testdata", "proto", "proto-test-d.yang")},
		wantOutputFiles: map[string]string{
			"openconfig.proto_test_d":      filepath.Join(TestRoot, "testdata", "proto", "proto-test-d.uncompressed.proto-test-d.formatted-txt"),
			"openconfig.proto_test_d.test": filepath.Join(TestRoot, "testdata", "proto", "proto-test-d.uncompressed.proto-test-d.test.formatted-txt"),
			"openconfig.enums":             filepath.Join(TestRoot, "testdata", "proto", "proto-test-d.uncompressed.enums.formatted-txt"),
		},
	}, {
		name:    "yang schema with unions",
		inFiles: []string{filepath.Join(TestRoot, "testdata", "proto", "proto-test-e.yang")},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					UseDefiningModuleForTypedefEnumNames: true,
				},
			},
		},
		wantOutputFiles: map[string]string{
			"openconfig.proto_test_e":                filepath.Join(TestRoot, "testdata", "proto", "proto-test-e.uncompressed.proto-test-e.formatted-txt"),
			"openconfig.proto_test_e.test":           filepath.Join(TestRoot, "testdata", "proto", "proto-test-e.uncompressed.proto-test-e.test.formatted-txt"),
			"openconfig.proto_test_e.foos":           filepath.Join(TestRoot, "testdata", "proto", "proto-test-e.uncompressed.proto-test-e.foos.formatted-txt"),
			"openconfig.proto_test_e.foos.foo":       filepath.Join(TestRoot, "testdata", "proto", "proto-test-e.uncompressed.proto-test-e.foos.foo.formatted-txt"),
			"openconfig.proto_test_e.bars":           filepath.Join(TestRoot, "testdata", "proto", "proto-test-e.uncompressed.proto-test-e.bars.formatted-txt"),
			"openconfig.enums":                       filepath.Join(TestRoot, "testdata", "proto", "proto-test-e.uncompressed.enums.formatted-txt"),
			"openconfig.proto_test_e.animals":        filepath.Join(TestRoot, "testdata", "proto", "proto-test-e.uncompressed.proto-test-e.animals.formatted-txt"),
			"openconfig.proto_test_e.animals.animal": filepath.Join(TestRoot, "testdata", "proto", "proto-test-e.uncompressed.proto-test-e.animals.animal.formatted-txt"),
		},
	}, {
		name:    "yang schema with anydata",
		inFiles: []string{filepath.Join(TestRoot, "testdata", "proto", "proto-anydata-test.yang")},
		wantOutputFiles: map[string]string{
			"openconfig.proto_anydata_test":   filepath.Join(TestRoot, "testdata", "proto", "proto_anydata_test.formatted-txt"),
			"openconfig.proto_anydata_test.e": filepath.Join(TestRoot, "testdata", "proto", "proto_anydata_test.e.formatted-txt"),
		},
	}, {
		name:    "yang schema with path annotations",
		inFiles: []string{filepath.Join(TestRoot, "testdata", "proto", "proto-test-f.yang")},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{},
			ProtoOptions: ProtoOpts{
				AnnotateSchemaPaths: true,
			},
		},
		wantOutputFiles: map[string]string{
			"openconfig.proto_test_f":     filepath.Join(TestRoot, "testdata", "proto", "proto_test_f.uncompressed.proto_test_f.formatted-txt"),
			"openconfig.proto_test_f.a":   filepath.Join(TestRoot, "testdata", "proto", "proto_test_f.uncompressed.proto_test_f.a.formatted-txt"),
			"openconfig.proto_test_f.a.c": filepath.Join(TestRoot, "testdata", "proto", "proto_test_f.uncompressed.proto_test_f.a.c.formatted-txt"),
		},
	}, {
		name:    "yang schema with leafrefs that point to the same path",
		inFiles: []string{filepath.Join(TestRoot, "testdata", "proto", "proto-test-g.yang")},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{},
			ProtoOptions: ProtoOpts{
				GoPackageBase:  "github.com/foo/baz",
				NestedMessages: true,
			},
		},
		wantOutputFiles: map[string]string{
			"openconfig.proto_test_g": filepath.Join(TestRoot, "testdata", "proto", "proto-test-g.proto-test-g.formatted-txt"),
		},
	}, {
		name:    "yang schema with fake root, path compression and union list key",
		inFiles: []string{filepath.Join(TestRoot, "testdata", "proto", "proto-union-list-key.yang")},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					CompressBehaviour: genutil.PreferIntendedConfig,
					GenerateFakeRoot:  true,
				},
			},
			ProtoOptions: ProtoOpts{
				AnnotateSchemaPaths: true,
			},
		},
		wantOutputFiles: map[string]string{
			"openconfig":                filepath.Join(TestRoot, "testdata", "proto", "proto-union-list-key.compressed.openconfig.formatted-txt"),
			"openconfig.routing_policy": filepath.Join(TestRoot, "testdata", "proto", "proto-union-list-key.compressed.openconfig.routing_policy.formatted-txt"),
		},
	}, {
		name:    "yang schema with fakeroot, and union list key",
		inFiles: []string{filepath.Join(TestRoot, "testdata", "proto", "proto-union-list-key.yang")},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					GenerateFakeRoot: true,
				},
			},
			ProtoOptions: ProtoOpts{
				AnnotateSchemaPaths: true,
			},
		},
		wantOutputFiles: map[string]string{
			"openconfig":                                                     filepath.Join(TestRoot, "testdata", "proto", "proto-union-list_key.uncompressed.openconfig.formatted-txt"),
			"openconfig.proto_union_list_key":                                filepath.Join(TestRoot, "testdata", "proto", "proto-union-list-key.uncompressed.openconfig.proto_union_list_key.formatted-txt"),
			"openconfig.proto_union_list_key.routing_policy":                 filepath.Join(TestRoot, "testdata", "proto", "proto-union-list-key.uncompressed.openconfig.proto_union_list_key.routing_policy.formatted-txt"),
			"openconfig.proto_union_list_key.routing_policy.policies":        filepath.Join(TestRoot, "testdata", "proto", "proto-union-list-key.uncompressed.openconfig.proto_union_list_key.routing_policy.policies.formatted-txt"),
			"openconfig.proto_union_list_key.routing_policy.policies.policy": filepath.Join(TestRoot, "testdata", "proto", "proto-union-list-key.uncompressed.openconfig.proto_union_list_key.routing_policy.policies.policy.formatted-txt"),
			"openconfig.proto_union_list_key.routing_policy.sets":            filepath.Join(TestRoot, "testdata", "proto", "proto-union-list-key.uncompressed.openconfig.proto_union_list_key.routing_policy.sets.formatted-txt"),
		},
	}, {
		name:    "enums: yang schema with various types of enums with underscores",
		inFiles: []string{filepath.Join(TestRoot, "testdata", "proto", "proto-enums.yang")},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					UseDefiningModuleForTypedefEnumNames: true,
				},
			},
		},
		wantOutputFiles: map[string]string{
			"openconfig.enums":       filepath.Join(TestRoot, "testdata", "proto", "proto-enums.enums.formatted-txt"),
			"openconfig.proto_enums": filepath.Join(TestRoot, "testdata", "proto", "proto-enums.formatted-txt"),
		},
	}, {
		name: "enums: yang schema with identity that adds to previous module",
		inFiles: []string{
			filepath.Join(TestRoot, "testdata", "proto", "proto-enums.yang"),
			filepath.Join(TestRoot, "testdata", "proto", "proto-enums-addid.yang"),
		},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					UseDefiningModuleForTypedefEnumNames: true,
				},
			},
			ProtoOptions: ProtoOpts{
				AnnotateEnumNames: true,
			},
		},
		wantOutputFiles: map[string]string{
			"openconfig.enums":       filepath.Join(TestRoot, "testdata", "proto", "proto-enums-addid.enums.formatted-txt"),
			"openconfig.proto_enums": filepath.Join(TestRoot, "testdata", "proto", "proto-enums-addid.formatted-txt"),
		},
	}, {
		name: "yang schema with nested messages requested - uncompressed with fakeroot",
		inFiles: []string{
			filepath.Join(TestRoot, "testdata", "proto", "nested-messages.yang"),
		},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					GenerateFakeRoot:                     true,
					UseDefiningModuleForTypedefEnumNames: true,
				},
			},
			ProtoOptions: ProtoOpts{
				AnnotateEnumNames:   true,
				AnnotateSchemaPaths: true,
				NestedMessages:      true,
			},
		},
		wantOutputFiles: map[string]string{
			"openconfig":                 filepath.Join(TestRoot, "testdata", "proto", "nested-messages.openconfig.formatted-txt"),
			"openconfig.enums":           filepath.Join(TestRoot, "testdata", "proto", "nested-messages.enums.formatted-txt"),
			"openconfig.nested_messages": filepath.Join(TestRoot, "testdata", "proto", "nested-messages.nested_messages.formatted-txt"),
		},
	}, {
		name: "yang schema with nested messages - compressed with fakeroot",
		inFiles: []string{
			filepath.Join(TestRoot, "testdata", "proto", "nested-messages.yang"),
		},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					CompressBehaviour:                    genutil.PreferIntendedConfig,
					GenerateFakeRoot:                     true,
					UseDefiningModuleForTypedefEnumNames: true,
				},
			},
			ProtoOptions: ProtoOpts{
				AnnotateEnumNames:   true,
				AnnotateSchemaPaths: true,
				NestedMessages:      true,
			},
		},
		wantOutputFiles: map[string]string{
			"openconfig.enums": filepath.Join(TestRoot, "testdata", "proto", "nested-messages.compressed.enums.formatted-txt"),
			"openconfig":       filepath.Join(TestRoot, "testdata", "proto", "nested-messages.compressed.nested_messages.formatted-txt"),
		},
	}, {
		name: "yang schema with a leafref key to a union with enumeration",
		inFiles: []string{
			filepath.Join(TestRoot, "testdata", "proto", "union-list-key.yang"),
		},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					GenerateFakeRoot:                     true,
					UseDefiningModuleForTypedefEnumNames: true,
				},
			},
			ProtoOptions: ProtoOpts{
				AnnotateEnumNames:   true,
				AnnotateSchemaPaths: true,
				NestedMessages:      true,
			},
		},
		wantOutputFiles: map[string]string{
			"openconfig.enums":          filepath.Join(TestRoot, "testdata", "proto", "union-list-key.enums.formatted-txt"),
			"openconfig.union_list_key": filepath.Join(TestRoot, "testdata", "proto", "union-list-key.union_list_key.formatted-txt"),
			"openconfig":                filepath.Join(TestRoot, "testdata", "proto", "union-list-key.formatted-txt"),
		},
	}, {
		name: "protobuf generation with excluded read only fields - compressed",
		inFiles: []string{
			filepath.Join(datapath, "openconfig-config-false.yang"),
		},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					GenerateFakeRoot:  true,
					CompressBehaviour: genutil.UncompressedExcludeDerivedState,
				},
			},
			ProtoOptions: ProtoOpts{
				AnnotateEnumNames:   true,
				AnnotateSchemaPaths: true,
				NestedMessages:      true,
			},
		},
		wantOutputFiles: map[string]string{
			"openconfig":                         filepath.Join(TestRoot, "testdata", "proto", "excluded-config-false.compressed.formatted-txt"),
			"openconfig.openconfig_config_false": filepath.Join(TestRoot, "testdata", "proto", "excluded-config-false.config_false.compressed.formatted-txt"),
		},
	}, {
		name: "protobuf generation with excluded read only fields - compressed",
		inFiles: []string{
			filepath.Join(datapath, "openconfig-config-false.yang"),
		},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					GenerateFakeRoot:  true,
					CompressBehaviour: genutil.ExcludeDerivedState,
				},
			},
			ProtoOptions: ProtoOpts{
				AnnotateEnumNames:   true,
				AnnotateSchemaPaths: true,
				NestedMessages:      true,
				GoPackageBase:       "github.com/openconfig/a/package",
			},
		},
		wantOutputFiles: map[string]string{
			"openconfig": filepath.Join(TestRoot, "testdata", "proto", "excluded-config-false.uncompressed.formatted-txt"),
		},
	}, {
		name: "protobuf generation with leafref to a module excluded by the test",
		inFiles: []string{
			filepath.Join(TestRoot, "testdata", "proto", "cross-ref-target.yang"),
			filepath.Join(TestRoot, "testdata", "proto", "cross-ref-src.yang"),
		},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				ParseOptions: ygen.ParseOpts{
					ExcludeModules: []string{"cross-ref-target"},
				},
			},
			ProtoOptions: ProtoOpts{
				NestedMessages: true,
			},
		},
		wantOutputFiles: map[string]string{
			"openconfig.cross_ref_src": filepath.Join(TestRoot, "testdata", "proto", "cross-ref-src.formatted-txt"),
		},
	}, {
		name: "multimod with fakeroot and nested",
		inFiles: []string{
			filepath.Join(TestRoot, "testdata", "proto", "fakeroot-multimod-one.yang"),
			filepath.Join(TestRoot, "testdata", "proto", "fakeroot-multimod-two.yang"),
		},
		inConfig: CodeGenerator{
			IROptions: ygen.IROptions{
				TransformationOptions: ygen.TransformationOpts{
					GenerateFakeRoot:  true,
					CompressBehaviour: genutil.PreferIntendedConfig,
				},
			},
			ProtoOptions: ProtoOpts{
				NestedMessages:      true,
				AnnotateEnumNames:   true,
				AnnotateSchemaPaths: true,
			},
		},
		wantOutputFiles: map[string]string{
			"openconfig": filepath.Join(TestRoot, "testdata", "proto", "fakeroot-multimod.formatted-txt"),
		},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			sortedPkgNames := func(pkgs map[string]string) []string {
				wantPkgs := []string{}
				for k := range tt.wantOutputFiles {
					wantPkgs = append(wantPkgs, k)
				}
				sort.Strings(wantPkgs)
				return wantPkgs
			}

			genCode := func() *GeneratedCode {
				if tt.inConfig.Caller == "" {
					// Override the caller if it is not set, to ensure that test
					// output is deterministic.
					tt.inConfig.Caller = "codegen-tests"
				}

				cg := New(tt.inConfig.Caller, tt.inConfig.IROptions, tt.inConfig.ProtoOptions)
				gotProto, err := cg.Generate(tt.inFiles, tt.inIncludePaths)
				if (err != nil) != tt.wantErr {
					t.Fatalf("cg.Generate(%v, %v), config: %v: got unexpected error: %v", tt.inFiles, tt.inIncludePaths, tt.inConfig, err)
				}

				if tt.wantErr || err != nil {
					return nil
				}

				return gotProto
			}

			gotProto := genCode()

			allCode := bytes.Buffer{}

			seenPkg := map[string]bool{}
			for n := range gotProto.Packages {
				seenPkg[n] = false
			}

			protoPkgs := func(m map[string]Proto3Package) []string {
				a := []string{}
				for k := range m {
					a = append(a, k)
				}
				return a
			}

			wantPkgs := sortedPkgNames(tt.wantOutputFiles)
			for _, pkg := range wantPkgs {
				wantFile := tt.wantOutputFiles[pkg]
				wantCodeBytes, err := ioutil.ReadFile(wantFile)
				if err != nil {
					t.Errorf("%s: ioutil.ReadFile(%v): could not read file for package %s", tt.name, wantFile, pkg)
					return
				}

				gotPkg, ok := gotProto.Packages[pkg]
				if !ok {
					t.Fatalf("%s: cg.Generate(%v, %v): did not find expected package %s in output, got: %#v, want key: %v", tt.name, tt.inFiles, tt.inIncludePaths, pkg, protoPkgs(gotProto.Packages), pkg)
				}

				// Mark this package as having been seen.
				seenPkg[pkg] = true

				// Write the returned struct out to a buffer to compare with the
				// testdata file.
				var gotCodeBuf bytes.Buffer
				fmt.Fprintf(&gotCodeBuf, gotPkg.Header)

				for _, gotMsg := range gotPkg.Messages {
					fmt.Fprintf(&gotCodeBuf, "%s\n", gotMsg)
				}

				for _, gotEnum := range gotPkg.Enums {
					fmt.Fprintf(&gotCodeBuf, "%s", gotEnum)
				}

				wantCode := string(wantCodeBytes)

				allCode.WriteString(gotCodeBuf.String())

				if diff := cmp.Diff(gotCodeBuf.String(), wantCode); diff != "" {
					if diffl, _ := testutil.GenerateUnifiedDiff(wantCode, gotCodeBuf.String()); diffl != "" {
						diff = diffl
					}
					t.Errorf("%s: cg.Generate(%v, %v) for package %s, did not get expected code (code file: %v), diff(-want, +got):\n%s", tt.name, tt.inFiles, tt.inIncludePaths, pkg, wantFile, diff)
				}
			}

			for pkg, seen := range seenPkg {
				if !seen {
					t.Errorf("%s: cg.Generate(%v, %v) did not test received package %v", tt.name, tt.inFiles, tt.inIncludePaths, pkg)
				}
			}

			for i := 0; i < deflakeRuns; i++ {
				got := genCode()
				var gotCodeBuf bytes.Buffer

				wantPkgs := sortedPkgNames(tt.wantOutputFiles)
				for _, pkg := range wantPkgs {
					gotPkg, ok := got.Packages[pkg]
					if !ok {
						t.Fatalf("%s: cg.Generate(%v, %v): did not find expected package %s in output, got: %#v, want key: %v", tt.name, tt.inFiles, tt.inIncludePaths, pkg, protoPkgs(gotProto.Packages), pkg)
					}
					fmt.Fprintf(&gotCodeBuf, gotPkg.Header)
					for _, gotMsg := range gotPkg.Messages {
						fmt.Fprintf(&gotCodeBuf, "%s\n", gotMsg)
					}
					for _, gotEnum := range gotPkg.Enums {
						fmt.Fprintf(&gotCodeBuf, "%s", gotEnum)
					}
				}

				if diff := cmp.Diff(gotCodeBuf.String(), allCode.String()); diff != "" {
					diff, _ = testutil.GenerateUnifiedDiff(allCode.String(), gotCodeBuf.String())
					t.Fatalf("flaky code generation iter: %d, diff(-want, +got):\n%s", i, diff)
				}
			}
		})
	}
}
