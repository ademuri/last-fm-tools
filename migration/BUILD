load("@io_bazel_rules_go//go:def.bzl", "go_library")
load("@io_bazel_rules_go//extras:embed_data.bzl", "go_embed_data")

go_embed_data(
    name = "sql",
    src = "create-tables.sql",
    string = True,
    package = "migration",
    var = "Create",
)

go_library(
    name = "go_default_library",
    srcs = [
        "doc.go",
        "sql",  #keep
    ],
    importpath = "github.com/ademuri/last-fm-tools/migration",
    visibility = ["//visibility:public"],
)
