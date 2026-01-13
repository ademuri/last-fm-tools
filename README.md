Command-line tools that fetch a user's listening history from last.fm and
perform analysis on it (e.g. what was I listening to last October?).

# Commands

## update

Fetches data from last.fm and stores it in a local SQLite database. You must run this before running analysis commands.

```bash
$ last-fm-tools update --user=foo
```

## top-artists

Calculates the top artists for a given time period.

```bash
$ last-fm-tools top-artists 2020-01 --user=foo
```

## top-albums

Calculates the top albums for a given time period.

```bash
$ last-fm-tools top-albums 2020-01-01 2020-02-01 --user=foo --number=20
```

## taste-report

Generates a comprehensive music taste report in YAML format. This report includes metadata, current taste (artists, albums, tags), historical baseline, taste drift, and listening patterns.

```bash
$ last-fm-tools taste-report --user=foo
```

## forgotten

Surfaces artists and albums that were heavily listened to in the past but haven't been played recently. This helps in rediscovering music that has fallen out of rotation.

```bash
$ last-fm-tools forgotten --user=foo
```

Options:
- `--min-artist`: Minimum scrobbles for artist inclusion (default: 10)
- `--min-album`: Minimum scrobbles for album inclusion (default: 5)
- `--results`: Max results shown per interest band (default: 10)
- `--sort`: Sort order: 'dormancy' or 'listens' (default: 'dormancy')
- `--last_listen_after`: Only include entities with last listen after this date. Supports absolute dates (YYYY, YYYY-MM, YYYY-MM-DD) or relative durations (e.g., 30d, 12w, 6m, 1y).
- `--last_listen_before`: Only include entities with last listen before this date (default: 90d). Supports absolute dates or relative durations.
- `--first_listen_after`: Only include entities with first listen after this date. Supports absolute dates or relative durations.
- `--first_listen_before`: Only include entities with first listen before this date. Supports absolute dates or relative durations.

## check-sources

Analyzes recent scrobbling activity to detect potential failures in your listening setup. It checks for:
- **Work Scrobbler Failure:** Zero listens during work hours (Mon-Fri, 09:00-17:00).
- **Weekend Scrobbler Failure:** Zero listens during weekends.
- **General Failure:** Zero listens during off-hours.

```bash
$ last-fm-tools check-sources --user=foo --days=14
```

Options:
- `--days`: Number of days to check back (default: 14).
- `--history`: Simulate the check for the past N days to see if alerts would have triggered.
- `--work-streak`: Threshold for consecutive silent working days to trigger a work scrobbler alert (default: 3).
- `--other-streak`: Threshold for consecutive silent days (off-hours) to trigger a general/mobile scrobbler alert (default: 3).
- `--weekend-streak`: Threshold for consecutive silent weekend days to trigger a weekend scrobbler alert (default: 4).
- `--cooldown-days`: Days to wait before repeating a warning (default: 0).
- `--timezone`: Timezone for determining work hours (e.g., "America/Los_Angeles").
- `--work-hours`: Work hours interval in start-end format (default: "09-17").

## email

Sends an email report to the specified address. Supports multiple analysis types.

```bash
$ last-fm-tools email user@example.com top-artists top-albums 2023-01
```

Analysis types: `top-artists`, `top-albums`, `new-artists`, `new-albums`, `forgotten`, `top-n`, `taste-report`, `check-sources`.

You can pass parameters to specific reports using the `--params` flag. Parameters are matched to reports by their order:
```bash
$ last-fm-tools email user@example.com top-n forgotten --params "artists=20,albums=10" --params "min-artist=20"
```
To skip parameters for a report in the middle, use an empty string:
```bash
$ last-fm-tools email user@example.com top-n top-artists forgotten --params "artists=20" --params "" --params "min-artist=20"
```

## add-report

Adds a report configuration to the database to be sent periodically via `send-reports`.

```bash
$ last-fm-tools add-report --name="Monthly Summary" --dest=user@example.com --run_day=1 top-n taste-report --params "artists=20"
```

To add a daily report (e.g., for `check-sources`), use `--run_day=0`. You can configure timezones and cool-off periods using `--params`.

```bash
$ last-fm-tools add-report check-sources --name="Daily Check" --dest=user@example.com --run_day=0 --params "days=14,timezone=America/Los_Angeles,cool_off_days=1,work_hours=09-17"
```

Parameters for `check-sources`:
- `days`: Lookback window size.
- `timezone`: Timezone for determining work hours (e.g., "America/Los_Angeles").
- `work_hours`: Work hours interval (e.g., "09-17").
- `cool_off_days`: Minimum days to wait before sending another alert (default: 1).
- `work_streak`, `other_streak`, `weekend_streak`: Sensitivity thresholds for alerts.

## list-reports

Lists all configured periodic reports.

```bash
$ last-fm-tools list-reports
```

## send-reports

Checks the database for reports that need to be sent (based on `run_day`) and emails them. It also updates the database with the latest scrobbles before sending.

```bash
$ last-fm-tools send-reports
```

## Configuration

Configuration options

- `api_key` and `secret` come from [https://www.last.fm/api/account/create].
  Note that last.fm doesn't save these values, so you'll need to put them
  somewhere safe (e.g. the config file mentioned below, or a password manager).
- `user` is the last.fm username.
- `database` is the path to the sqlite database file.
- `smtp_username` (optional) is the SMTP username (e.g. your Gmail address), used for sending email reports.
- `smtp_password` (optional) is the SMTP password. For Gmail, this must be a [Google App Password](https://support.google.com/accounts/answer/185833).
- `from` (optional) is the email address to send reports from

These may be specified either as normal flags, or as configuration options in
`$HOME/.last-fm-tools.yaml`, forex:

```yaml
database: "$HOME/lastfm.db"
api_key: ""
secret: ""
smtp_username: "me@gmail.com"
smtp_password: "abcd1234efgh5678"
from: "me@me.com"
```

# Building

This project uses [bazel](https://bazel.build/) for building. It's the only
required dependency. To build and run directly using Bazel:

```bash
$ USE_BAZEL_VERSION=7.1.0 npx @bazel/bazelisk run //:last-fm-tools -- update --user=foo --database=$HOME/lastfm.db
```

To run tests:

```bash
$ USE_BAZEL_VERSION=7.1.0 npx @bazel/bazelisk test //...
```

## Updating dependencies

To update dependencies edit [go.mod], and then run Gazelle:

```bash
USE_BAZEL_VERSION=7.1.0 npx @bazel/bazelisk run //:gazelle -- update-repos -from_file=go.mod -to_macro=repositories.bzl%go_repositories
USE_BAZEL_VERSION=7.1.0 npx @bazel/bazelisk run //:gazelle
```

# Gemini

To use this project with the Gemini CLI in a consistent environment, you can use the provided `Dockerfile` as a sandbox.

1. **Build the sandbox image:**

   ```bash
   docker build -t last-fm-tools-sandbox .
   ```

2. **Run Gemini with the sandbox:**
   ```bash
   GEMINI_SANDBOX_IMAGE=last-fm-tools-sandbox gemini
   ```

This ensures that Gemini has access to the correct version of Go (1.25.0) and Bazel without modifying your local system.
