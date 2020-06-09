load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library")

go_binary(
    name = "last-fm-tools",
    data = [
        "sql/create-tables.sql",
    ],
    embed = [":go_default_library"],
    visibility = ["//visibility:public"],
)

load("@bazel_gazelle//:def.bzl", "gazelle")

# gazelle:prefix github.com/ademuri/last-fm-tools
gazelle(name = "gazelle")

go_library(
    name = "go_default_library",
    srcs = ["tools.go"],
    importpath = "github.com/ademuri/last-fm-tools",
    visibility = ["//visibility:private"],
    deps = [
        "//secrets:go_default_library",
        "@com_github_mattn_go_sqlite3//:go_default_library",
        "@com_github_shkh_lastfm_go//lastfm:go_default_library",
    ],
)
