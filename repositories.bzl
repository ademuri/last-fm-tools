# Update with:
# bazel run //:gazelle -- update-repos -from_file=go.mod -to_macro=repositories.bzl%go_repositories

load("@bazel_gazelle//:deps.bzl", "go_repository")

def go_repositories():
    go_repository(
        name = "com_github_andybalholm_cascadia",
        importpath = "github.com/andybalholm/cascadia",
        sum = "h1:vuRCkM5Ozh/BfmsaTm26kbjm0mIOM3yS5Ek/F5h18aE=",
        version = "v1.2.0",
    )
    go_repository(
        name = "com_github_puerkitobio_goquery",
        importpath = "github.com/PuerkitoBio/goquery",
        sum = "h1:PSPBGne8NIUWw+/7vFBV+kG2J/5MOjbzc7154OaKCSE=",
        version = "v1.5.1",
    )
    go_repository(
        name = "com_github_yuin_goldmark",
        importpath = "github.com/yuin/goldmark",
        sum = "h1:5tjfNdR2ki3yYQ842+eX2sQHeiwpKJ0RnHO4IYOc4V8=",
        version = "v1.1.32",
    )
    go_repository(
        name = "org_golang_x_crypto",
        importpath = "golang.org/x/crypto",
        sum = "h1:vEg9joUBmeBcK9iSJftGNf3coIG4HqZElCPehJsfAYM=",
        version = "v0.0.0-20200604202706-70a84ac30bf9",
    )
    go_repository(
        name = "org_golang_x_mod",
        importpath = "golang.org/x/mod",
        sum = "h1:RM4zey1++hCTbCVQfnWeKs9/IEsaBLA8vTkd0WVtmH4=",
        version = "v0.3.0",
    )
    go_repository(
        name = "org_golang_x_net",
        importpath = "golang.org/x/net",
        sum = "h1:pNX+40auqi2JqRfOP1akLGtYcn15TUbkhwuCO3foqqM=",
        version = "v0.0.0-20200602114024-627f9648deb9",
    )
    go_repository(
        name = "org_golang_x_sync",
        importpath = "golang.org/x/sync",
        sum = "h1:WXEvlFVvvGxCJLG6REjsT03iWnKLEWinaScsxF2Vm2o=",
        version = "v0.0.0-20200317015054-43a5402ce75a",
    )
    go_repository(
        name = "org_golang_x_sys",
        importpath = "golang.org/x/sys",
        sum = "h1:bGb80FudwxpeucJUjPYJXuJ8Hk91vNtfvrymzwiei38=",
        version = "v0.0.0-20200610111108-226ff32320da",
    )
    go_repository(
        name = "org_golang_x_text",
        importpath = "golang.org/x/text",
        sum = "h1:tW2bmiBqwgJj/UpqtC8EpXEZVYOwU0yG4iWbprSVAcs=",
        version = "v0.3.2",
    )
    go_repository(
        name = "org_golang_x_tools",
        importpath = "golang.org/x/tools",
        sum = "h1:MI14dOfl3OG6Zd32w3ugsrvcUO810fDZdWakTq39dH4=",
        version = "v0.0.0-20200608174601-1b747fd94509",
    )
    go_repository(
        name = "org_golang_x_xerrors",
        importpath = "golang.org/x/xerrors",
        sum = "h1:E7g+9GITq07hpfrRu66IVDexMakfv52eLZ2CXBWiKr4=",
        version = "v0.0.0-20191204190536-9bdfabe68543",
    )
    go_repository(
        name = "com_github_mattn_go_sqlite3",
        importpath = "github.com/mattn/go-sqlite3",
        sum = "h1:gXHsfypPkaMZrKbD5209QV9jbUTJKjyR5WD3HYQSd+U=",
        version = "v2.0.3+incompatible",
    )
    go_repository(
        name = "com_github_shkh_lastfm_go",
        importpath = "github.com/shkh/lastfm-go",
        sum = "h1:cgqwZtnR+IQfUYDLJ3Kiy4aE+O/wExTzEIg8xwC4Qfs=",
        version = "v0.0.0-20191215035245-89a801c244e0",
    )
    go_repository(
        name = "com_github_avast_retry_go",
        importpath = "github.com/avast/retry-go",
        sum = "h1:FelcMrm7Bxacr1/RM8+/eqkDkmVN7tjlsy51dOzB3LI=",
        version = "v2.6.0+incompatible",
    )
