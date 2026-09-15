#!/bin/sh
# MOS-MCP installer for macOS and Linux.
#
#   curl -fsSL https://raw.githubusercontent.com/medcelerate/MOS-MCP/main/scripts/install.sh | sh
#
# Environment overrides:
#   MOSMCP_VERSION   version tag to install (default: latest release)
#   MOSMCP_INSTALL_DIR   install directory (default: /usr/local/bin, else ~/.local/bin)
set -eu

REPO="medcelerate/MOS-MCP"
BIN="mos-mcp"

info() { printf '\033[1;34m==>\033[0m %s\n' "$1"; }
err()  { printf '\033[1;31merror:\033[0m %s\n' "$1" >&2; exit 1; }

command -v curl >/dev/null 2>&1 || err "curl is required"
command -v tar  >/dev/null 2>&1 || err "tar is required"

# Detect OS.
os=$(uname -s)
case "$os" in
  Linux)  OS="linux" ;;
  Darwin) OS="darwin" ;;
  *) err "unsupported OS: $os" ;;
esac

# Detect architecture.
arch=$(uname -m)
case "$arch" in
  x86_64|amd64) ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *) err "unsupported architecture: $arch" ;;
esac

# Resolve version.
VERSION="${MOSMCP_VERSION:-}"
if [ -z "$VERSION" ]; then
  info "Resolving latest release"
  VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep '"tag_name"' | head -1 | sed -E 's/.*"tag_name" *: *"([^"]+)".*/\1/')
  [ -n "$VERSION" ] || err "could not determine latest version; set MOSMCP_VERSION"
fi
VNUM="${VERSION#v}"

ARCHIVE="${BIN}_${VNUM}_${OS}_${ARCH}.tar.gz"
BASE="https://github.com/${REPO}/releases/download/${VERSION}"

TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

info "Downloading ${ARCHIVE}"
curl -fsSL "${BASE}/${ARCHIVE}" -o "${TMP}/${ARCHIVE}" || err "download failed"

# Verify checksum when available.
if curl -fsSL "${BASE}/checksums.txt" -o "${TMP}/checksums.txt" 2>/dev/null; then
  info "Verifying checksum"
  expected=$(grep "  ${ARCHIVE}\$" "${TMP}/checksums.txt" | awk '{print $1}')
  if [ -n "$expected" ]; then
    if command -v sha256sum >/dev/null 2>&1; then
      actual=$(sha256sum "${TMP}/${ARCHIVE}" | awk '{print $1}')
    else
      actual=$(shasum -a 256 "${TMP}/${ARCHIVE}" | awk '{print $1}')
    fi
    [ "$expected" = "$actual" ] || err "checksum mismatch for ${ARCHIVE}"
  fi
fi

info "Extracting"
tar -xzf "${TMP}/${ARCHIVE}" -C "${TMP}"
[ -f "${TMP}/${BIN}" ] || err "binary not found in archive"
chmod +x "${TMP}/${BIN}"

# Choose an install directory.
if [ -n "${MOSMCP_INSTALL_DIR:-}" ]; then
  DIR="$MOSMCP_INSTALL_DIR"
elif [ -w /usr/local/bin ] 2>/dev/null; then
  DIR="/usr/local/bin"
elif [ "$(id -u)" = "0" ]; then
  DIR="/usr/local/bin"
else
  DIR="$HOME/.local/bin"
fi
mkdir -p "$DIR"

if mv "${TMP}/${BIN}" "${DIR}/${BIN}" 2>/dev/null; then
  :
elif command -v sudo >/dev/null 2>&1; then
  info "Installing to ${DIR} (requires sudo)"
  sudo mv "${TMP}/${BIN}" "${DIR}/${BIN}"
else
  err "cannot write to ${DIR}; set MOSMCP_INSTALL_DIR to a writable directory"
fi

info "Installed ${BIN} ${VERSION} to ${DIR}/${BIN}"
case ":$PATH:" in
  *":$DIR:"*) ;;
  *) printf '\033[1;33mnote:\033[0m %s is not on your PATH. Add it with:\n  export PATH="%s:$PATH"\n' "$DIR" "$DIR" ;;
esac
"${DIR}/${BIN}" --version || true
