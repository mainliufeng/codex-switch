pkgname=codex-switch-git
pkgver=r1.f599ab6
pkgrel=1
pkgdesc="Tray app for switching Codex auth.json profiles"
arch=("x86_64")
url="https://github.com/mainliufeng/codex-switch"
license=("custom")
depends=("gtk3" "libayatana-appindicator" "xdg-utils")
makedepends=("git" "go" "gcc" "pkgconf")
provides=("codex-switch")
conflicts=("codex-switch")
source=("git+$url.git")
sha256sums=("SKIP")

pkgver() {
  cd "$srcdir/codex-switch"
  printf "r%s.%s" "$(git rev-list --count HEAD)" "$(git rev-parse --short HEAD)"
}

build() {
  cd "$srcdir/codex-switch"
  ./scripts/build.sh
}

package() {
  cd "$srcdir/codex-switch"

  install -Dm755 "dist/codex-switch" "$pkgdir/usr/bin/codex-switch"
  install -Dm644 "assets/codex-switch-icon.png" "$pkgdir/usr/share/pixmaps/codex-switch.png"

  install -Dm644 /dev/stdin "$pkgdir/usr/share/applications/codex-switch.desktop" <<'EOF'
[Desktop Entry]
Type=Application
Version=1.0
Name=Codex Switch
Comment=Switch Codex auth.json profiles from the system tray
Exec=/usr/bin/codex-switch
Icon=codex-switch
Terminal=false
Categories=Utility;
NoDisplay=false
StartupNotify=false
EOF
}
