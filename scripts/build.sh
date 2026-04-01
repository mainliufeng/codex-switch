#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="${DIST_DIR:-$ROOT_DIR/dist}"
BIN_PATH="$DIST_DIR/codex-switch"

if ! command -v go >/dev/null 2>&1; then
  echo "error: go is not installed" >&2
  exit 1
fi

if ! command -v pkg-config >/dev/null 2>&1; then
  echo "error: pkg-config is required to build the tray binary" >&2
  exit 1
fi

if ! pkg-config --exists gtk+-3.0; then
  echo "error: missing gtk+-3.0 development headers" >&2
  echo "hint: install libgtk-3-dev (Debian/Ubuntu) or gtk3 (Arch)" >&2
  exit 1
fi

build_tags=()
if pkg-config --exists ayatana-appindicator3-0.1; then
  :
elif pkg-config --exists appindicator3-0.1; then
  build_tags+=("legacy_appindicator")
else
  echo "error: missing appindicator development headers" >&2
  echo "hint: install libayatana-appindicator3-dev or libappindicator3-dev" >&2
  exit 1
fi

mkdir -p "$DIST_DIR"

build_cmd=(go build -o "$BIN_PATH")
if ((${#build_tags[@]} > 0)); then
  build_cmd+=(-tags "${build_tags[*]}")
fi
build_cmd+=(./cmd/codex-switch)

echo "==> Building $BIN_PATH"
CGO_ENABLED=1 "${build_cmd[@]}"
echo "==> Done"
