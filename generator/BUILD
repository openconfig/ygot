go_binary(
    name = "generator",
    srcs = ["generator.go"],
)

go_test(
    name = "generator_test",
    size = "small",
    srcs = ["generator_test.go"],
    library = ":generator",
)
