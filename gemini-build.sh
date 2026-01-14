#!/bin/bash
set -e

VERB=$1
shift

if [[ "$VERB" != "build" && "$VERB" != "test" && "$VERB" != "run" ]]; then
  echo "Usage: $0 [build|test|run] [targets/args...]"
  exit 1
fi

CACHE_BASE="/home/adam/.gemini/tmp/fb109f76a946ce4b91c8829dd30073ddd175711580991c1acf35b7a400a0f905/cache"
mkdir -p "$CACHE_BASE/npm" "$CACHE_BASE/xdg" "$CACHE_BASE/bazel_root" "$CACHE_BASE/bin" "$CACHE_BASE/go" "$CACHE_BASE/go-cache"
ln -sf $(which python3) "$CACHE_BASE/bin/python"
export npm_config_cache="$CACHE_BASE/npm"
export XDG_CACHE_HOME="$CACHE_BASE/xdg"
export GOPATH="$CACHE_BASE/go"
export GOCACHE="$CACHE_BASE/go-cache"
export PATH="$CACHE_BASE/bin:$PATH"

ARGS=("$@")
if [[ "$VERB" == "run" ]]; then
  if [[ ${#ARGS[@]} -eq 0 || ( "${ARGS[0]}" != //* && "${ARGS[0]}" != @* && "${ARGS[0]}" != :* ) ]]; then
     ARGS=("//:last-fm-tools" "${ARGS[@]}")
  fi
elif [[ ${#ARGS[@]} -eq 0 ]]; then
  ARGS=("//...")
fi

USE_BAZEL_VERSION=7.1.0 npx --yes @bazel/bazelisk --output_user_root="$CACHE_BASE/bazel_root" "$VERB" --noshow_loading_progress --noshow_progress "${ARGS[@]}"
