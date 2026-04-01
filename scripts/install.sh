#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
PREFIX="${PREFIX:-$HOME/.local}"
BIN_DIR="$PREFIX/bin"
APP_DIR="$PREFIX/share/applications"
PIXMAPS_DIR="$PREFIX/share/pixmaps"
ICON_DIR_64="$PREFIX/share/icons/hicolor/64x64/apps"
ICON_DIR_256="$PREFIX/share/icons/hicolor/256x256/apps"
AUTOSTART_DIR="$HOME/.config/autostart"
BIN_PATH="$BIN_DIR/codex-switch"
DESKTOP_PATH="$APP_DIR/codex-switch.desktop"
PIXMAPS_ICON_PATH="$PIXMAPS_DIR/codex-switch.png"
ICON_PATH_64="$ICON_DIR_64/codex-switch.png"
ICON_PATH_256="$ICON_DIR_256/codex-switch.png"
AUTOSTART_PATH="$AUTOSTART_DIR/codex-switch.desktop"

enable_autostart=0
disable_autostart=0

for arg in "$@"; do
  case "$arg" in
    --autostart)
      enable_autostart=1
      ;;
    --no-autostart)
      disable_autostart=1
      ;;
    *)
      echo "error: unknown argument: $arg" >&2
      echo "usage: $0 [--autostart] [--no-autostart]" >&2
      exit 1
      ;;
  esac
done

if ((enable_autostart == 1 && disable_autostart == 1)); then
  echo "error: --autostart and --no-autostart cannot be used together" >&2
  exit 1
fi

"$ROOT_DIR/scripts/build.sh"

mkdir -p "$BIN_DIR" "$APP_DIR" "$PIXMAPS_DIR" "$ICON_DIR_64" "$ICON_DIR_256"
install -m 755 "$ROOT_DIR/dist/codex-switch" "$BIN_PATH"
install -m 644 "$ROOT_DIR/assets/codex-switch-icon.png" "$PIXMAPS_ICON_PATH"
install -m 644 "$ROOT_DIR/assets/codex-switch-tray.png" "$ICON_PATH_64"
install -m 644 "$ROOT_DIR/assets/codex-switch-icon.png" "$ICON_PATH_256"

cat >"$DESKTOP_PATH" <<EOF
[Desktop Entry]
Type=Application
Version=1.0
Name=Codex Switch
Comment=Switch Codex accounts from a classic tray menu
Exec=$BIN_PATH
Icon=codex-switch
Terminal=false
Categories=Utility;
NoDisplay=false
StartupNotify=false
EOF

if command -v gtk-update-icon-cache >/dev/null 2>&1; then
  gtk-update-icon-cache -q -t "$PREFIX/share/icons/hicolor" || true
fi

if command -v update-desktop-database >/dev/null 2>&1; then
  update-desktop-database "$APP_DIR" || true
fi

if ((enable_autostart == 1)); then
  mkdir -p "$AUTOSTART_DIR"
  install -m 644 "$DESKTOP_PATH" "$AUTOSTART_PATH"
  echo "==> Installed autostart entry: $AUTOSTART_PATH"
elif ((disable_autostart == 1)); then
  rm -f "$AUTOSTART_PATH"
  echo "==> Removed autostart entry: $AUTOSTART_PATH"
fi

echo "==> Installed binary: $BIN_PATH"
echo "==> Installed icon: $PIXMAPS_ICON_PATH"
echo "==> Installed desktop entry: $DESKTOP_PATH"
