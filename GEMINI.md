# last-fm-tools for Gemini

This project provides command-line tools to fetch a user's listening history from last.fm and perform analysis on it.

## Description

The tool is written in Go and uses a local SQLite database to store fetched data. Key features include:

- **Fetching Data:** The `update` command retrieves listening history from last.fm.
- **Analysis:**
  - `forgotten`: Finds artists and albums that were listened to in the past but haven't been played recently.
  - `taste-report`: Generates a comprehensive music taste report in YAML format.
- **Reporting:** Can send email reports via SMTP.

## Build Instructions (Gemini Environment)

For safety, we run the Gemini CLI in a Docker container. In this environment, standard `bazel` commands may fail due to permission issues or incompatible tooling. Use the provided `gemini-build.sh` script to build, test, and run the project. It sets up a persistent cache in the project's temporary directory to avoid re-downloading dependencies and bypasses root-owned file permission errors.

### Commands

The script accepts `build`, `test`, or `run` as the first argument.

```bash
# Build the project
$ bash gemini-build.sh build

# Run all tests
$ bash gemini-build.sh test

# Run the application (pass arguments after '--')
$ bash gemini-build.sh run -- --help
$ bash gemini-build.sh run -- update
```

## Configuration

The application uses `viper` for configuration. You can provide settings via command-line flags or a YAML configuration file.

**Recommended:** Create a config file at `~/.last-fm-tools.yaml` (or pass `--config /path/to/config.yaml`).

```yaml
api_key: "YOUR_LAST_FM_API_KEY"
secret: "YOUR_LAST_FM_SECRET"
user: "target_username"
database: "./lastfm.db"
# Optional: Email reporting
smtp_username: "..."
smtp_password: "..."
from: "..."
```

Alternatively, pass flags directly:
```bash
$ bash gemini-build.sh run -- update --api_key=... --user=...
```

## Project Structure

- **`cmd/`**: Contains the CLI command definitions (Cobra). Entry point for `update`, `top-artists`, etc.
- **`internal/`**: Core application logic.
  - **`internal/store`**: SQLite database interactions.
  - **`internal/analysis`**: Logic for calculating statistics.
- **`gemini-build.sh`**: Wrapper script for Bazel operations in restricted environments.

## General Instructions

- Prefer to create (Git) commits for small units of work, e.g. a single bugfix.
- Strive for total test coverage whenever practical. Add test coverage by default.
- Update the documentation in the README when making changes. If what you're working on isn't already in the README, add it if it is significant.
