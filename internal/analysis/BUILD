load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "analysis.go",
        "forgotten.go",
        "types.go",
    ],
    importpath = "github.com/ademuri/last-fm-tools/internal/analysis",
    visibility = ["//visibility:public"],
    deps = [
        "//internal/store:go_default_library",
    ],
)

go_test(
    name = "go_default_test",
    srcs = [
        "analysis_test.go",
        "forgotten_test.go",
        "top_albums_format_test.go",
    ],
    embed = [":go_default_library"],
    deps = [
        "//internal/migration:go_default_library",
        "//internal/store:go_default_library",
        "@com_github_mattn_go_sqlite3//:go_default_library",
    ],
)