#!/bin/bash

set -euo pipefail

cp -n secrets/secrets.go.sample secrets/secrets.go
bazel build :last-fm-tools
