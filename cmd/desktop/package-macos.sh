#!/bin/sh
set -eu
[ "$(uname -s)" = Darwin ] || exit 0
desktop_bundle='dist/desktop/LocalFinance.app'
mkdir -p "$desktop_bundle/Contents/MacOS" "$desktop_bundle/Contents/Resources"
cp dist/desktop/local-finance-desktop "$desktop_bundle/Contents/MacOS/LocalFinance"
cp cmd/desktop/Info.plist "$desktop_bundle/Contents/Info.plist"
cp packaging/macos/AppIcon.icns "$desktop_bundle/Contents/Resources/AppIcon.icns"
desktop_version=${VERSION:-0.0.0}
desktop_version=${desktop_version#v}
desktop_version=${desktop_version%%-*}
case "$desktop_version" in *[!0-9.]*|'') desktop_version=0.0.0 ;; esac
/usr/libexec/PlistBuddy -c "Set :CFBundleShortVersionString $desktop_version" "$desktop_bundle/Contents/Info.plist"
/usr/libexec/PlistBuddy -c "Set :CFBundleVersion ${BUILD_NUMBER:-1}" "$desktop_bundle/Contents/Info.plist"
# Ad-hoc signing works without an Apple account. Developer ID is optional.
if [ -n "${APPLE_SIGNING_IDENTITY:-}" ]; then
    codesign --force --options runtime --timestamp --sign "$APPLE_SIGNING_IDENTITY" "$desktop_bundle"
else
    codesign --force --sign - "$desktop_bundle"
fi
codesign --verify --deep --strict --all-architectures "$desktop_bundle"
if [ "${PACKAGE_DMG:-0}" = 1 ]; then
    desktop_stage=$(mktemp -d)
    trap 'rm -rf "$desktop_stage"' EXIT
    cp -R "$desktop_bundle" "$desktop_stage/"
    ln -s /Applications "$desktop_stage/Applications"
    hdiutil create -volname LocalFinance -srcfolder "$desktop_stage" -ov -format UDZO dist/desktop/LocalFinance-Desktop.dmg
    if [ -n "${APPLE_SIGNING_IDENTITY:-}" ]; then
        codesign --force --timestamp --sign "$APPLE_SIGNING_IDENTITY" dist/desktop/LocalFinance-Desktop.dmg
        codesign --verify --strict dist/desktop/LocalFinance-Desktop.dmg
    fi
fi
