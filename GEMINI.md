# last-fm-tools for Gemini

This project provides command-line tools to fetch a user's listening history from last.fm and perform analysis on it.

## Description

The tool is written in Go and uses a local SQLite database to store fetched data. Key features include:

- **Fetching Data:** The `update` command retrieves listening history from last.fm.
- **Analysis:**
  - `top-artists`: Calculates top artists for a given time period (e.g., `2020-01`).
  - `top-albums`: Calculates top albums for a given time period.
- **Reporting:** Can send email reports via SendGrid.

## Build Instructions (Gemini Environment)

For safety, we run the Gemini CLI in a Docker container. In this environment, standard `bazel` commands may fail due to permission issues or incompatible tooling. Use the provided `gemini-build.sh` script to build and test the project. It sets up a persistent cache in the project's temporary directory to avoid re-downloading dependencies and bypasses root-owned file permission errors.

The script takes one argument: `build` or `test`.

```bash
$ bash gemini-build.sh build
# OR
$ bash gemini-build.sh test
```

## General Instructions

Prefer to create (Git) commits for small units of work, e.g. a single bugfix.
