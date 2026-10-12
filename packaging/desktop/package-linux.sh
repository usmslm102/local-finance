#!/bin/sh
set -eu
desktop_arch=$(go env GOARCH)
desktop_version=${VERSION:-0.0.0}
desktop_version=${desktop_version#v}
desktop_version=$(printf '%s' "$desktop_version" | sed 's/-/~/')
# Debian permits semver prerelease suffixes; dev builds use a numeric baseline.
case "$desktop_version" in [0-9]*) ;; *) desktop_version=0.0.0 ;; esac
desktop_stage=$(mktemp -d)
trap 'rm -rf "$desktop_stage"' EXIT
mkdir -p "$desktop_stage/usr/share/icons/hicolor/256x256/apps" "$desktop_stage/usr/bin" "$desktop_stage/usr/share/applications" "$desktop_stage/DEBIAN"
cp dist/desktop/local-finance-desktop "$desktop_stage/usr/bin/local-finance-desktop"
chmod 755 "$desktop_stage/usr/bin/local-finance-desktop"
cp packaging/desktop/AppIcon.png "$desktop_stage/usr/share/icons/hicolor/256x256/apps/localfinance.png"
cp packaging/desktop/localfinance.desktop "$desktop_stage/usr/share/applications/"
cat > "$desktop_stage/DEBIAN/control" <<CONTROL
Package: local-finance-desktop
Version: $desktop_version
Architecture: $desktop_arch
Maintainer: LocalFinance maintainers
Depends: libc6 (>= 2.39), libgtk-4-1, libwebkitgtk-6.0-4
Section: utils
Priority: optional
Description: Offline personal finance with a native desktop window
CONTROL
dpkg-deb --root-owner-group --build "$desktop_stage" "dist/desktop/local-finance-desktop-linux-$desktop_arch.deb"
tar -C dist/desktop -czf "dist/desktop/local-finance-desktop-linux-$desktop_arch.tar.gz" local-finance-desktop
