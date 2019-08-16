go_library(
    name = "ygen",
    srcs = [
        "codegen.go",
        "directory.go",
        "genstate.go",
        "goelements.go",
        "gogen.go",
        "helpers.go",
        "protoelements.go",
        "protogen.go",
        "schemaparse.go",
        "schematree.go",
    ],
    deps = ["//third_party/golang/goyang/pkg/yang"],
)

go_test(
    name = "ygen_test",
    size = "small",
    srcs = [
        "codegen_test.go",
        "directory_test.go",
        "genstate_test.go",
        "goelements_test.go",
        "gogen_test.go",
        "helpers_test.go",
        "protoelements_test.go",
        "protogen_test.go",
        "schemaparse_test.go",
        "schematree_test.go",
    ],
    library = ":ygen",
    deps = ["//third_party/golang/goyang/pkg/yang"],
)
