#!/bin/sh
# Build internal/web/static/tailwind.css from internal/web/input.css using the
# standalone Tailwind CLI (no Node.js required). The generated file is committed
# and embedded via go:embed, so this only needs to run when the console's markup
# or the input CSS changes.
#
#   TAILWIND_VERSION=v4.1.0 sh scripts/build-css.sh   # pin a version
set -eu

VERSION="${TAILWIND_VERSION:-latest}"
BASE="https://github.com/tailwindlabs/tailwindcss/releases"

err() { printf 'error: %s\n' "$1" >&2; exit 1; }
command -v curl >/dev/null 2>&1 || err "curl is required"

os=$(uname -s)
case "$os" in
  Linux)  OS="linux" ;;
  Darwin) OS="macos" ;;
  MINGW*|MSYS*|CYGWIN*) OS="windows" ;;
  *) err "unsupported OS: $os" ;;
esac

arch=$(uname -m)
case "$arch" in
  x86_64|amd64) A="x64" ;;
  arm64|aarch64) A="arm64" ;;
  *) err "unsupported architecture: $arch" ;;
esac

ASSET="tailwindcss-${OS}-${A}"
[ "$OS" = "windows" ] && ASSET="${ASSET}.exe"

mkdir -p .tools
BIN=".tools/${ASSET}"
if [ ! -x "$BIN" ]; then
  if [ "$VERSION" = "latest" ]; then
    URL="${BASE}/latest/download/${ASSET}"
  else
    URL="${BASE}/download/${VERSION}/${ASSET}"
  fi
  printf 'Downloading Tailwind CLI (%s)...\n' "$ASSET"
  curl -fsSL "$URL" -o "$BIN"
  chmod +x "$BIN"
fi

"$BIN" -i internal/web/input.css -o internal/web/static/tailwind.css --minify
printf 'Wrote internal/web/static/tailwind.css\n'
