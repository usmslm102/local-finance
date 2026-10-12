#!/bin/sh
set -eu
# Invoke from repository root. Native toolchains are required for macOS/Linux.
desktop_version=${VERSION:-dev}
desktop_output=${DESKTOP_OUTPUT:-../../dist/desktop/local-finance-desktop}
cd cmd/desktop
mkdir -p "$(dirname "$desktop_output")"
desktop_flags="-s -w -X local-finance/internal/updater.CurrentVersion=$desktop_version"
case "$(go env GOOS)" in
    darwin)
        MACOSX_DEPLOYMENT_TARGET=13.0 CGO_ENABLED=1 \
            CGO_CFLAGS='-O2 -g -mmacosx-version-min=13.0' \
            CGO_CXXFLAGS='-O2 -g -mmacosx-version-min=13.0' \
            go build -trimpath -tags production \
            -ldflags "$desktop_flags -extldflags=-mmacosx-version-min=13.0" \
            -o "$desktop_output" . ;;
    windows)
        desktop_arch=$(go env GOARCH)
        desktop_resource="rsrc_windows_$desktop_arch.syso"
        trap 'rm -f "$desktop_resource"' EXIT
        mkdir -p ../../dist/desktop
        GOOS=$(go env GOHOSTOS) GOARCH=$(go env GOHOSTARCH) CGO_ENABLED=0 \
            go run ./tools/resources -arch "$desktop_arch" -version "$desktop_version"
        CGO_ENABLED=0 go build -trimpath -tags production \
            -ldflags "$desktop_flags -H windowsgui" -o "$desktop_output.exe" . ;;
    linux)
        CGO_ENABLED=1 go build -trimpath -tags production \
            -ldflags "$desktop_flags" -o "$desktop_output" . ;;
    *) echo 'Unsupported desktop platform' >&2; exit 1 ;;
esac
