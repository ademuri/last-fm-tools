# Update with:
# bazel run //:gazelle -- update-repos -from_file=go.mod -to_macro=repositories.bzl%go_repositories

load("@bazel_gazelle//:deps.bzl", "go_repository")

def go_repositories():
    go_repository(
        name = "com_github_ademuri_lastfm_go",
        importpath = "github.com/ademuri/lastfm-go",
        sum = "h1:H28NRPPrRABhj/6c/GnuJX63oaySrC+FTq4emwF5TEA=",
        version = "v0.0.0-20200704180735-3661b99c9dec",
    )
    go_repository(
        name = "com_github_andybalholm_cascadia",
        importpath = "github.com/andybalholm/cascadia",
        sum = "h1:vuRCkM5Ozh/BfmsaTm26kbjm0mIOM3yS5Ek/F5h18aE=",
        version = "v1.2.0",
    )
    go_repository(
        name = "com_github_avast_retry_go",
        importpath = "github.com/avast/retry-go",
        sum = "h1:4SOWQ7Qs+oroOTQOYnAHqelpCO0biHSxpiH9JdtuBj0=",
        version = "v3.0.0+incompatible",
    )
    go_repository(
        name = "com_github_clipperhouse_displaywidth",
        importpath = "github.com/clipperhouse/displaywidth",
        sum = "h1:ZDpTkFfpHOKte4RG5O/BOyf3ysnvFswpyYrV7z2uAKo=",
        version = "v0.6.2",
    )
    go_repository(
        name = "com_github_clipperhouse_stringish",
        importpath = "github.com/clipperhouse/stringish",
        sum = "h1:+NSqMOr3GR6k1FdRhhnXrLfztGzuG+VuFDfatpWHKCs=",
        version = "v0.1.1",
    )
    go_repository(
        name = "com_github_clipperhouse_uax29_v2",
        importpath = "github.com/clipperhouse/uax29/v2",
        sum = "h1:SNdx9DVUqMoBuBoW3iLOj4FQv3dN5mDtuqwuhIGpJy4=",
        version = "v2.3.0",
    )
    go_repository(
        name = "com_github_frankban_quicktest",
        importpath = "github.com/frankban/quicktest",
        sum = "h1:7Xjx+VpznH+oBnejlPUj8oUpdxnVs4f8XU8WnHkI4W8=",
        version = "v1.14.6",
    )
    go_repository(
        name = "com_github_go_viper_mapstructure_v2",
        importpath = "github.com/go-viper/mapstructure/v2",
        sum = "h1:EBsztssimR/CONLSZZ04E8qAkxNYq4Qp9LvH92wZUgs=",
        version = "v2.4.0",
    )
    go_repository(
        name = "com_github_mattn_go_sqlite3",
        importpath = "github.com/mattn/go-sqlite3",
        sum = "h1:A5blZ5ulQo2AtayQ9/limgHEkFreKj1Dv226a1K73s0=",
        version = "v1.14.33",
    )
    go_repository(
        name = "com_github_olekukonko_cat",
        importpath = "github.com/olekukonko/cat",
        sum = "h1:zrbMGy9YXpIeTnGj4EljqMiZsIcE09mmF8XsD5AYOJc=",
        version = "v0.0.0-20250911104152-50322a0618f6",
    )
    go_repository(
        name = "com_github_olekukonko_errors",
        importpath = "github.com/olekukonko/errors",
        sum = "h1:RNuGIh15QdDenh+hNvKrJkmxxjV4hcS50Db478Ou5sM=",
        version = "v1.1.0",
    )
    go_repository(
        name = "com_github_olekukonko_ll",
        importpath = "github.com/olekukonko/ll",
        sum = "h1:sV2jrhQGq5B3W0nENUISCR6azIPf7UBUpVq0x/y70Fg=",
        version = "v0.1.3",
    )
    go_repository(
        name = "com_github_olekukonko_ts",
        importpath = "github.com/olekukonko/ts",
        sum = "h1:LiZB1h0GIcudcDci2bxbqI6DXV8bF8POAnArqvRrIyw=",
        version = "v0.0.0-20171002115256-78ecb04241c0",
    )
    go_repository(
        name = "com_github_pelletier_go_toml_v2",
        importpath = "github.com/pelletier/go-toml/v2",
        sum = "h1:mye9XuhQ6gvn5h28+VilKrrPoQVanw5PMw/TB0t5Ec4=",
        version = "v2.2.4",
    )

    go_repository(
        name = "com_github_puerkitobio_goquery",
        importpath = "github.com/PuerkitoBio/goquery",
        sum = "h1:PSPBGne8NIUWw+/7vFBV+kG2J/5MOjbzc7154OaKCSE=",
        version = "v1.5.1",
    )
    go_repository(
        name = "com_github_sagikazarmark_locafero",
        importpath = "github.com/sagikazarmark/locafero",
        sum = "h1:/NQhBAkUb4+fH1jivKHWusDYFjMOOKU88eegjfxfHb4=",
        version = "v0.12.0",
    )
    go_repository(
        name = "com_github_sendgrid_rest",
        importpath = "github.com/sendgrid/rest",
        sum = "h1:1EyIcsNdn9KIisLW50MKwmSRSK+ekueiEMJ7NEoxJo0=",
        version = "v2.6.9+incompatible",
    )
    go_repository(
        name = "com_github_sendgrid_sendgrid_go",
        importpath = "github.com/sendgrid/sendgrid-go",
        sum = "h1:zWhTmB0Y8XCDzeWIm2/BIt1GjJohAA0p6hVEaDtHWWs=",
        version = "v3.16.1+incompatible",
    )
    go_repository(
        name = "com_github_shkh_lastfm_go",
        importpath = "github.com/shkh/lastfm-go",
        sum = "h1:cgqwZtnR+IQfUYDLJ3Kiy4aE+O/wExTzEIg8xwC4Qfs=",
        version = "v0.0.0-20191215035245-89a801c244e0",
    )
    go_repository(
        name = "com_github_sourcegraph_conc",
        importpath = "github.com/sourcegraph/conc",
        sum = "h1:+jumHNA0Wrelhe64i8F6HNlS8pkoyMv5sreGx2Ry5Rw=",
        version = "v0.3.1-0.20240121214520-5f936abd7ae8",
    )

    go_repository(
        name = "com_github_yuin_goldmark",
        importpath = "github.com/yuin/goldmark",
        sum = "h1:fVcFKWvrslecOb/tg+Cc05dkeYx540o0FuFt3nUVDoE=",
        version = "v1.4.13",
    )
    go_repository(
        name = "in_gopkg_yaml_v3",
        importpath = "gopkg.in/yaml.v3",
        sum = "h1:fxVm/GzAzEWqLHuvctI91KS9hhNmmWOoWu0XTYJS7CA=",
        version = "v3.0.1",
    )
    go_repository(
        name = "in_yaml_go_yaml_v3",
        importpath = "go.yaml.in/yaml/v3",
        sum = "h1:tfq32ie2Jv2UxXFdLJdh3jXuOzWiL1fo0bu/FbuKpbc=",
        version = "v3.0.4",
    )
    go_repository(
        name = "org_golang_x_crypto",
        importpath = "golang.org/x/crypto",
        sum = "h1:cKRW/pmt1pKAfetfu+RCEvjvZkA9RimPbh7bhFjGVBU=",
        version = "v0.46.0",
    )
    go_repository(
        name = "org_golang_x_mod",
        importpath = "golang.org/x/mod",
        sum = "h1:fDEXFVZ/fmCKProc/yAXXUijritrDzahmwwefnjoPFk=",
        version = "v0.30.0",
    )
    go_repository(
        name = "org_golang_x_net",
        importpath = "golang.org/x/net",
        sum = "h1:zyQRTTrjc33Lhh0fBgT/H3oZq9WuvRR5gPC70xpDiQU=",
        version = "v0.48.0",
    )
    go_repository(
        name = "org_golang_x_sync",
        importpath = "golang.org/x/sync",
        sum = "h1:vV+1eWNmZ5geRlYjzm2adRgW2/mcpevXNg50YZtPCE4=",
        version = "v0.19.0",
    )
    go_repository(
        name = "org_golang_x_sys",
        importpath = "golang.org/x/sys",
        sum = "h1:CvCKL8MeisomCi6qNZ+wbb0DN9E5AATixKsvNtMoMFk=",
        version = "v0.39.0",
    )
    go_repository(
        name = "org_golang_x_term",
        importpath = "golang.org/x/term",
        sum = "h1:PQ5pkm/rLO6HnxFR7N2lJHOZX6Kez5Y1gDSJla6jo7Q=",
        version = "v0.38.0",
    )
    go_repository(
        name = "org_golang_x_text",
        importpath = "golang.org/x/text",
        sum = "h1:ZD01bjUt1FQ9WJ0ClOL5vxgxOI/sVCNgX1YtKwcY0mU=",
        version = "v0.32.0",
    )
    go_repository(
        name = "org_golang_x_tools",
        importpath = "golang.org/x/tools",
        sum = "h1:ik4ho21kwuQln40uelmciQPp9SipgNDdrafrYA4TmQQ=",
        version = "v0.39.0",
    )
    go_repository(
        name = "org_golang_x_xerrors",
        importpath = "golang.org/x/xerrors",
        sum = "h1:E7g+9GITq07hpfrRu66IVDexMakfv52eLZ2CXBWiKr4=",
        version = "v0.0.0-20191204190536-9bdfabe68543",
    )
