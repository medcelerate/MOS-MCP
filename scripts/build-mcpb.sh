#!/bin/sh
# Build a Claude Desktop Extension (.mcpb bundle) for mos-mcp.
#
# A .mcpb bundle is a zip containing manifest.json plus a self-contained binary
# under server/. This script builds the binary for one platform, stages the
# bundle, and packs it. Run once per platform you want to ship.
#
#   sh scripts/build-mcpb.sh                 # host platform
#   GOOS=windows GOARCH=amd64 sh scripts/build-mcpb.sh
#   GOOS=linux   GOARCH=arm64 sh scripts/build-mcpb.sh
set -eu

VERSION="${VERSION:-$(git describe --tags --always 2>/dev/null || echo dev)}"
GOOS="${GOOS:-$(go env GOOS)}"
GOARCH="${GOARCH:-$(go env GOARCH)}"

BIN="mos-mcp"
EXT=""
[ "$GOOS" = "windows" ] && EXT=".exe"

STAGE="build/mcpb"
OUT_DIR="dist"
OUT="${OUT_DIR}/${BIN}-${VERSION}-${GOOS}-${GOARCH}.mcpb"

rm -rf "$STAGE"
mkdir -p "$STAGE/server" "$OUT_DIR"

echo "Building ${BIN} for ${GOOS}/${GOARCH}..."
CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" go build \
  -ldflags "-s -w -X github.com/medcelerate/MOS-MCP/internal/version.Version=${VERSION}" \
  -o "$STAGE/server/${BIN}${EXT}" ./cmd/mos-mcp

cp mcpb/manifest.json "$STAGE/manifest.json"

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
