Command-line tools that fetch a user's listening history from last.fm and
perform analysis on it (e.g. what was I listening to last October?).

# Usage

First, run `update` to fetch data from last.fm. This data is stored in a local
SQLite database. Then, you can run the analysis commands `top-artists` and
`top-albums` to get the top artists and albums for a given time period. Usage
for these commands looks like:

```bash
$ last-fm-tools top-artists 2020-01 --user=foo
$ last-fm-tools top-albums 2020-01-01 2020-02-01 --user=foo --number=20
```

## Configuration

Configuration options

- `api_key` and `secret` come from [https://www.last.fm/api/account/create].
  Note that last.fm doesn't save these values, so you'll need to put them
  somewhere safe (e.g. the config file mentioned below, or a password manager).
- `user` is the last.fm username.
- `database` is the path to the sqlite database file. 
- `sendgrid_api_key` (optional) is the API key for SendGrid, used for sending
  email reports
- `from` (optional) is the email address to send reports from

These may be specified either as normal flags, or as configuration options in
`$HOME/.last-fm-tools.yaml`, forex:

```yaml
database: "$HOME/lastfm.db"
api_key: ""
secret: ""
sendgrid_api_key: ""
from: "me@me.com"
```

# Building

This project uses [bazel](https://bazel.build/) for building. It's the only
required dependency. To build and run directly using Bazel:

```bash
$ bazel run //:last-fm-tools -- update --user=foo --database=$HOME/lastfm.db
```

To run tests:
```bash
$ bazel test //...
```

## Updating dependencies

To update dependencies edit [go.mod], and then run Gazelle:

```bash
bazel run //:gazelle -- update-repos -from_file=go.mod -to_macro=repositories.bzl%go_repositories
bazel run //:gazelle
```

