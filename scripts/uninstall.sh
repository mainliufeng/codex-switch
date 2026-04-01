#!/usr/bin/env bash

set -euo pipefail

PREFIX="${PREFIX:-$HOME/.local}"
BIN_PATH="$PREFIX/bin/codex-switch"
DESKTOP_PATH="$PREFIX/share/applications/codex-switch.desktop"
PIXMAPS_ICON_PATH="$PREFIX/share/pixmaps/codex-switch.png"
ICON_PATH_64="$PREFIX/share/icons/hicolor/64x64/apps/codex-switch.png"
ICON_PATH_256="$PREFIX/share/icons/hicolor/256x256/apps/codex-switch.png"
AUTOSTART_PATH="$HOME/.config/autostart/codex-switch.desktop"
DATA_DIR="${XDG_DATA_HOME:-$HOME/.local/share}/codex-switch"

purge_data=0

for arg in "$@"; do
  case "$arg" in
    --purge-data)
      purge_data=1
      ;;
    *)
      echo "error: unknown argument: $arg" >&2
      echo "usage: $0 [--purge-data]" >&2
      exit 1
      ;;
  esac
done

rm -f "$BIN_PATH"
rm -f "$DESKTOP_PATH"
rm -f "$PIXMAPS_ICON_PATH"
rm -f "$ICON_PATH_64"
rm -f "$ICON_PATH_256"
rm -f "$AUTOSTART_PATH"

if command -v gtk-update-icon-cache >/dev/null 2>&1; then
  gtk-update-icon-cache -q -t "$PREFIX/share/icons/hicolor" || true
fi

if command -v update-desktop-database >/dev/null 2>&1; then
  update-desktop-database "$PREFIX/share/applications" || true
fi

if ((purge_data == 1)); then
  rm -rf "$DATA_DIR"
  echo "==> Removed data directory: $DATA_DIR"
else
  echo "==> Kept data directory: $DATA_DIR"
fi

echo "==> Removed binary: $BIN_PATH"
echo "==> Removed desktop entry: $DESKTOP_PATH"
echo "==> Removed icons for codex-switch"
echo "==> Removed autostart entry: $AUTOSTART_PATH"
