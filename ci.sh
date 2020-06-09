#!/bin/bash

set -euo pipefail

cp -n internal/secrets/secrets.go.sample internal/secrets/secrets.go
bazel build //...
bazel test //...
