load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "date.go",
        "root.go",
        "topAlbums.go",
        "topArtists.go",
        "update.go",
    ],
    importpath = "github.com/ademuri/last-fm-tools/cmd",
    visibility = ["//visibility:public"],
    deps = [
        "//internal/migration:go_default_library",
        "//internal/secrets:go_default_library",
        "@com_github_avast_retry_go//:go_default_library",
        "@com_github_mattn_go_sqlite3//:go_default_library",
        "@com_github_mitchellh_go_homedir//:go_default_library",
        "@com_github_olekukonko_tablewriter//:go_default_library",
        "@com_github_shkh_lastfm_go//lastfm:go_default_library",
        "@com_github_spf13_cobra//:go_default_library",
        "@com_github_spf13_pflag//:go_default_library",
        "@com_github_spf13_viper//:go_default_library",
        "@org_golang_x_time//rate:go_default_library",
    ],
)

go_test(
    name = "go_default_test",
    srcs = [
        "date_test.go",
        "topAlbums_test.go",
        "update_test.go",
    ],
    embed = [":go_default_library"],
)
