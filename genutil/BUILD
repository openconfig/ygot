package(default_visibility = ["//visibility:public"])

licenses(["notice"])  # Apache 2.0

go_library(
    name = "genutil",
    srcs = [
        "common.go",
        "names.go",
    ],
    deps = [
        "//third_party/golang/goyang/pkg/yang",
        "//third_party/golang/ygot/util",
        "//third_party/golang/ygot/ygot",
    ],
)

go_test(
    name = "genutil_test",
    size = "small",
    srcs = [
        "common_test.go",
        "names_test.go",
    ],
    library = ":genutil",
    deps = ["//third_party/golang/goyang/pkg/yang"],
)
