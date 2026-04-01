#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="${DIST_DIR:-$ROOT_DIR/dist}"
APP_BIN="$DIST_DIR/codex-switch"

if ! command -v go >/dev/null 2>&1; then
  echo "error: go is not installed" >&2
  exit 1
fi

if ! command -v pkg-config >/dev/null 2>&1; then
  echo "error: pkg-config is required to build codex-switch" >&2
  exit 1
fi

if ! pkg-config --exists gtk+-3.0; then
  echo "error: missing gtk+-3.0 development headers" >&2
  exit 1
fi

if ! pkg-config --exists ayatana-appindicator3-0.1 && ! pkg-config --exists appindicator3-0.1; then
  echo "error: missing appindicator development headers" >&2
  exit 1
fi

mkdir -p "$DIST_DIR"
echo "==> Building $APP_BIN"
CGO_ENABLED=1 go build -o "$APP_BIN" ./cmd/codex-switch
echo "==> Done"
