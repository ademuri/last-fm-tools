load("@bazel_gazelle//:def.bzl", "gazelle")
load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library")

# gazelle:prefix github.com/ademuri/last-fm-tools
# gazelle:build_file_name BUILD
gazelle(name = "gazelle")

go_library(
    name = "go_default_library",
    srcs = ["main.go"],
    importpath = "github.com/ademuri/last-fm-tools",
    visibility = ["//visibility:private"],
    deps = ["//cmd:go_default_library"],
)

go_binary(
    name = "last-fm-tools",
    embed = [":go_default_library"],
    visibility = ["//visibility:public"],
)