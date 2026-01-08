load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "doc.go",
        "migration.go",
    ],
    embedsrcs = ["create-tables.sql"],
    importpath = "github.com/ademuri/last-fm-tools/internal/migration",
    visibility = ["//visibility:public"],
)
