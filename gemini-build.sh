#!/bin/bash
set -e

VERB=$1
if [[ "$VERB" != "build" && "$VERB" != "test" ]]; then
  echo "Usage: $0 [build|test]"
  exit 1
fi

CACHE_BASE="/home/adam/.gemini/tmp/fb109f76a946ce4b91c8829dd30073ddd175711580991c1acf35b7a400a0f905/cache"
mkdir -p "$CACHE_BASE/npm" "$CACHE_BASE/xdg" "$CACHE_BASE/bazel_root" "$CACHE_BASE/bin"
ln -sf $(which python3) "$CACHE_BASE/bin/python"
export npm_config_cache="$CACHE_BASE/npm"
export XDG_CACHE_HOME="$CACHE_BASE/xdg"
export PATH="$CACHE_BASE/bin:$PATH"

USE_BAZEL_VERSION=3.6.0 npx --yes @bazel/bazelisk --output_user_root="$CACHE_BASE/bazel_root" "$VERB" //... --noshow_loading_progress --noshow_progress
