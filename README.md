Command-line tools that fetch a user's listening history from last.fm and
perform analysis on it (e.g. what was I listening to last October?).

# Usage

First, run `update` to fetch data from last.fm. This data is stored in a local SQLite database. Then, you can run the analysis commands `top-artists` and `top-albums` to get the top artists and albums for a given time period. Usage for these commands looks like:

```bash
$ last-fm-tools top-artists 2020-01 --user=foo
$ last-fm-tools top-albums 2020-01-01 2020-02-01 --user=foo --number=20
```

# Building

This project uses [bazel](https://bazel.build/) for building. It's the only required dependency. To build and run directly using Bazel:

```bash
$ bazel run //:last-fm-tools -- update --user=foo --database=$HOME/lastfm.db
```

To run tests:
```bash
$ bazel test //...
```
