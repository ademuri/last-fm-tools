load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "analysis.go",
        "forgotten.go",
        "read.go",
        "store.go",
        "top.go",
        "write.go",
    ],
    importpath = "github.com/ademuri/last-fm-tools/internal/store",
    visibility = ["//visibility:public"],
    deps = [
        "//internal/migration:go_default_library",
        "@com_github_mattn_go_sqlite3//:go_default_library",
    ],
)

go_test(
    name = "go_default_test",
    srcs = ["store_test.go"],
    embed = [":go_default_library"],
)
