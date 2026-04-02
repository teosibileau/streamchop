#!/bin/sh
set -e

REPO="teosibileau/streamchop"
BINARY="streamchop"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64) ARCH="arm64" ;;
  arm64)   ARCH="arm64" ;;
  *)
    echo "Error: unsupported architecture: $ARCH" >&2
    exit 1
    ;;
esac

case "$OS" in
  linux) ;;
  *)
    echo "Error: unsupported OS: $OS" >&2
    exit 1
    ;;
esac

BASE_URL="${STREAMCHOP_BASE_URL:-https://github.com/${REPO}/releases/latest/download}"
URL="${BASE_URL}/${BINARY}-${OS}-${ARCH}"

echo "Downloading ${BINARY} for ${OS}/${ARCH}..."
curl -fsSL -o "$BINARY" "$URL"
chmod +x "$BINARY"

echo ""
echo "Installed ./${BINARY}"
echo ""
echo "Run the setup wizard to discover cameras and generate deployment config:"
echo "  ./${BINARY} setup"
