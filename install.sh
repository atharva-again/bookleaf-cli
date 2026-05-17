#!/bin/sh
# BookLeaf CLI installer
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/atharva-again/bookleaf-cli/main/install.sh | sh -s -- v0.1.4

set -e

REPO="atharva-again/bookleaf-cli"
BINARY="bookleaf"
INSTALL_DIR="${BOOKLEAF_INSTALL:-/usr/local/bin}"

# ---- OS/Arch detection ----
if [ "$OS" = "Windows_NT" ]; then
    echo "Error: Windows is not supported by this installer." >&2
    echo "Please download the latest release from: https://github.com/$REPO/releases" >&2
    exit 1
fi

ARCH=$(uname -m)
case "$ARCH" in
    x86_64|amd64) ARCH="x86_64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *)
        echo "Error: unsupported architecture: $ARCH" >&2
        exit 1
        ;;
esac

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
    linux) OS="Linux" ;;
    darwin) OS="Darwin" ;;
    *)
        echo "Error: unsupported OS: $OS" >&2
        exit 1
        ;;
esac

# ---- Version resolution ----
if [ -z "$1" ]; then
    echo "Fetching latest version..."
    RAW_TAG=$(curl -fsSLI -o /dev/null -w '%{url_effective}' "https://github.com/$REPO/releases/latest" 2>/dev/null)
    TAG=$(printf '%s' "$RAW_TAG" | sed 's|.*/tag/v||')
    if [ -z "$TAG" ] || [ "$TAG" = "$RAW_TAG" ]; then
        echo "Error: could not determine latest version." >&2
        echo "Pass the version explicitly:" >&2
        echo "  curl -fsSL https://raw.githubusercontent.com/atharva-again/bookleaf-cli/main/install.sh | sh -s -- v0.1.4" >&2
        exit 1
    fi
    VERSION="$TAG"
else
    VERSION="${1#v}"
fi

echo "Installing $BINARY v$VERSION ($OS/$ARCH)..."

# ---- Build asset names ----
ARCHIVE="${BINARY}_${OS}_${ARCH}.tar.gz"
DOWNLOAD_URL="https://github.com/$REPO/releases/download/v${VERSION}/${ARCHIVE}"
CHECKSUMS_URL="https://github.com/$REPO/releases/download/v${VERSION}/checksums.txt"

# ---- Download ----
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

echo "Downloading $ARCHIVE..."
curl -fsSL --retry 3 "$DOWNLOAD_URL" -o "$TMP/$ARCHIVE"
curl -fsSL --retry 3 "$CHECKSUMS_URL" -o "$TMP/checksums.txt"

# ---- Checksum verification ----
if command -v sha256sum >/dev/null 2>&1; then
    SHA_CMD="sha256sum"
elif command -v shasum >/dev/null 2>&1; then
    SHA_CMD="shasum -a 256"
else
    echo "Warning: no sha256sum or shasum found; skipping checksum verification" >&2
    SKIP_CHECKSUM=1
fi

if [ -z "${SKIP_CHECKSUM:-}" ]; then
    echo "Verifying checksum..."
    EXPECTED=$(grep -E "  ${ARCHIVE}$" "$TMP/checksums.txt" | awk '{print $1}')
    if [ -z "$EXPECTED" ]; then
        echo "Error: checksum for $ARCHIVE not found in checksums.txt" >&2
        exit 1
    fi
    ACTUAL=$($SHA_CMD "$TMP/$ARCHIVE" | awk '{print $1}')
    if [ "$EXPECTED" != "$ACTUAL" ]; then
        echo "Error: checksum mismatch" >&2
        echo "  Expected: $EXPECTED" >&2
        echo "  Actual:   $ACTUAL" >&2
        exit 1
    fi
    echo "Checksum verified."
fi

# ---- Extract ----
echo "Extracting..."
tar -xzf "$TMP/$ARCHIVE" -C "$TMP"

# Archives are wrapped: bookleaf_Linux_x86_64/bookleaf
BINARY_PATH=$(find "$TMP" -name "$BINARY" -type f)
if [ -z "$BINARY_PATH" ]; then
    echo "Error: binary not found in archive" >&2
    exit 1
fi

# ---- Install ----
if [ ! -d "$INSTALL_DIR" ]; then
    mkdir -p "$INSTALL_DIR"
fi

if [ -w "$INSTALL_DIR" ]; then
    mv "$BINARY_PATH" "$INSTALL_DIR/$BINARY"
else
    echo "Cannot write to $INSTALL_DIR. Trying with sudo..." >&2
    sudo mv "$BINARY_PATH" "$INSTALL_DIR/$BINARY"
fi

chmod +x "$INSTALL_DIR/$BINARY"

echo ""
echo "$BINARY v$VERSION has been installed to $INSTALL_DIR/$BINARY"
echo ""
"$INSTALL_DIR/$BINARY" --help
