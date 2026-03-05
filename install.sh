#!/bin/sh
set -e

REPO="dropseed/git-stats"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
    linux)  ;;
    darwin) ;;
    *)      echo "Unsupported OS: $OS"; exit 1 ;;
esac

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
    x86_64|amd64)  ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *)             echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

# Get latest version
if [ -z "$VERSION" ]; then
    VERSION=$(curl -sI "https://github.com/$REPO/releases/latest" | grep -i "^location:" | sed 's/.*tag\///' | tr -d '\r\n')
    if [ -z "$VERSION" ]; then
        echo "Failed to detect latest version"
        exit 1
    fi
fi

FILENAME="git-stats_${OS}_${ARCH}.tar.gz"
URL="https://github.com/$REPO/releases/download/$VERSION/$FILENAME"

echo "Downloading git-stats $VERSION for ${OS}/${ARCH}..."

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

curl -sL "$URL" -o "$TMPDIR/$FILENAME"
tar -xzf "$TMPDIR/$FILENAME" -C "$TMPDIR"

if [ -w "$INSTALL_DIR" ]; then
    mv "$TMPDIR/git-stats" "$INSTALL_DIR/git-stats"
else
    echo "Installing to $INSTALL_DIR (requires sudo)..."
    sudo mv "$TMPDIR/git-stats" "$INSTALL_DIR/git-stats"
fi

echo "git-stats $VERSION installed to $INSTALL_DIR/git-stats"
