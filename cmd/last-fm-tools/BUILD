load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = ["tools.go"],
    importpath = "github.com/ademuri/last-fm-tools/cmd/last-fm-tools",
    visibility = ["//visibility:private"],
    deps = [
        "//internal/migration:go_default_library",
        "//internal/secrets:go_default_library",
        "@com_github_mattn_go_sqlite3//:go_default_library",
        "@com_github_shkh_lastfm_go//lastfm:go_default_library",
    ],
)

go_binary(
    name = "last-fm-tools",
    embed = [":go_default_library"],
    visibility = ["//visibility:public"],
)

go_test(
    name = "go_default_test",
    srcs = ["tools_test.go"],
    embed = [":go_default_library"],
)
