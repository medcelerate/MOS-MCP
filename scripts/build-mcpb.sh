#!/bin/sh
# Build a cross-platform Claude Desktop Extension (.mcpb bundle) for mos-mcp.
#
# A .mcpb bundle is a zip containing manifest.json plus self-contained binaries
# under server/. This produces ONE bundle that runs on:
#   - macOS  (universal: Intel + Apple Silicon, via lipo)
#   - Windows (amd64)
#   - Linux  (amd64)
# The manifest's platform_overrides select the right binary per OS.
#
#   sh scripts/build-mcpb.sh
#
# macOS is required to build the universal Mac binary (needs `lipo`). Without it
# the script falls back to a single-arch Mac binary and warns.
set -eu

VERSION="${VERSION:-$(git describe --tags --always 2>/dev/null || echo dev)}"
BIN="mos-mcp"
STAGE="build/mcpb"
OUT_DIR="dist"
OUT="${OUT_DIR}/${BIN}-${VERSION}-universal.mcpb"
LDFLAGS="-s -w -X github.com/medcelerate/MOS-MCP/internal/version.Version=${VERSION}"

build() { # GOOS GOARCH OUTFILE
  echo "  building $1/$2"
  CGO_ENABLED=0 GOOS="$1" GOARCH="$2" go build -ldflags "$LDFLAGS" -o "$3" ./cmd/mos-mcp
}

rm -rf "$STAGE"
mkdir -p "$STAGE/server" "$OUT_DIR"

echo "Building binaries..."
# macOS universal (Intel + Apple Silicon).
if command -v lipo >/dev/null 2>&1; then
  build darwin amd64 "$STAGE/server/${BIN}-darwin-amd64"
  build darwin arm64 "$STAGE/server/${BIN}-darwin-arm64"
  lipo -create -output "$STAGE/server/${BIN}-darwin" \
    "$STAGE/server/${BIN}-darwin-amd64" "$STAGE/server/${BIN}-darwin-arm64"
  rm -f "$STAGE/server/${BIN}-darwin-amd64" "$STAGE/server/${BIN}-darwin-arm64"
else
  echo "  warning: lipo not found; shipping a single-arch macOS binary ($(go env GOARCH))"
  build darwin "$(go env GOARCH)" "$STAGE/server/${BIN}-darwin"
fi
# Windows and Linux (amd64).
build windows amd64 "$STAGE/server/${BIN}-win32.exe"
build linux amd64 "$STAGE/server/${BIN}-linux"

cp mcpb/manifest.json "$STAGE/manifest.json"
[ -f mcpb/icon.png ] && cp mcpb/icon.png "$STAGE/icon.png"

# Prefer the official mcpb CLI (validates the manifest); fall back to zip.
if command -v mcpb >/dev/null 2>&1; then
  mcpb pack "$STAGE" "$OUT"
elif command -v npx >/dev/null 2>&1; then
  npx --yes @anthropic-ai/mcpb pack "$STAGE" "$OUT"
elif command -v zip >/dev/null 2>&1; then
  echo "mcpb CLI not found; packing with zip (manifest not validated)."
  (cd "$STAGE" && zip -qr - .) > "$OUT"
else
  echo "error: need the mcpb CLI, npx, or zip to pack the bundle" >&2
  exit 1
fi

echo "Wrote ${OUT}"
