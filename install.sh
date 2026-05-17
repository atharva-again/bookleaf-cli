#!/usr/bin/env bash
set -euo pipefail

APP="bookleaf"
REPO="atharva-again/bookleaf-cli"

Color_Off=''
Green=''
Red=''
Yellow=''
Dim=''
Bold=''

if [[ -t 1 ]]; then
  Color_Off='\033[0m'
  Green='\033[0;32m'
  Red='\033[0;31m'
  Yellow='\033[0;33m'
  Dim='\033[0;2m'
  Bold='\033[1m'
fi

info()  { echo -e "${Dim}$* ${Color_Off}"; }
info_bold() { echo -e "${Bold}$* ${Color_Off}"; }
success() { echo -e "${Green}$* ${Color_Off}"; }
error() { echo -e "${Red}error${Color_Off}: $*" >&2; exit 1; }

usage() {
  cat <<EOF
BookLeaf CLI Installer

Install the BookLeaf CLI to manage the BookLeaf support portal.

Usage: install.sh [options]

Options:
    -h, --help              Display this help message
    -v, --version <ver>     Install a specific version (e.g., v0.1.5)
        --no-modify-path    Don't modify shell config files (.zshrc, .bashrc, etc.)

Examples:
    curl -fsSL https://bookleaf-assignment-atharva.vercel.app/cli/install.sh | bash
    curl -fsSL https://bookleaf-assignment-atharva.vercel.app/cli/install.sh | bash -s -- --version v0.1.5
EOF
  exit 0
}

requested_version=""
no_modify_path=false

while [[ $# -gt 0 ]]; do
  case "$1" in
    -h|--help) usage ;;
    -v|--version)
      if [[ -n "${2:-}" ]]; then
        requested_version="$2"
        shift 2
      else
        error "--version requires a version argument"
      fi
      ;;
    --no-modify-path)
      no_modify_path=true
      shift
      ;;
    *)
      if [[ -z "$requested_version" ]]; then
        requested_version="$1"
        shift
      else
        error "Unknown option: $1"
      fi
      ;;
  esac
done

INSTALL_DIR="${BOOKLEAF_INSTALL:-$HOME/.bookleaf}"
BIN_DIR="$INSTALL_DIR/bin"

# ---- OS/Arch detection ----
raw_os=$(uname -s)
case "$raw_os" in
  Linux*)  os="Linux" ;;
  Darwin*) os="Darwin" ;;
  MINGW*|MSYS*|CYGWIN*) os="Windows" ;;
  *) error "Unsupported OS: $raw_os" ;;
esac

arch=$(uname -m)
case "$arch" in
  x86_64|amd64) arch="x86_64" ;;
  aarch64|arm64) arch="arm64" ;;
  *) error "Unsupported architecture: $arch" ;;
esac

# Rosetta detection: if on macOS x64 but running under Rosetta, use arm64
if [[ "$os" == "Darwin" && "$arch" == "x86_64" ]]; then
  if [[ $(sysctl -n sysctl.proc_translated 2>/dev/null) == 1 ]]; then
    info "Running under Rosetta 2, downloading arm64 binary instead"
    arch="arm64"
  fi
fi

archive_ext=".tar.gz"
if [[ "$os" == "Windows" ]]; then
  archive_ext=".zip"
fi

FILENAME="${APP}_${os}_${arch}${archive_ext}"

# ---- Version resolution ----
if [[ -z "$requested_version" ]]; then
  info "Fetching latest version..."
  RAW_TAG=$(curl -fsSLI -o /dev/null -w '%{url_effective}' "https://github.com/$REPO/releases/latest" 2>/dev/null)
  TAG=$(echo "$RAW_TAG" | sed 's|.*/tag/v||')
  if [[ -z "$TAG" || "$TAG" == "$RAW_TAG" ]]; then
    error "Could not determine latest version. Try: curl -fsSL ... | bash -s -- --version v0.1.5"
  fi
  VERSION="$TAG"
else
  VERSION="${requested_version#v}"
fi

DOWNLOAD_URL="https://github.com/$REPO/releases/download/v${VERSION}/${FILENAME}"

mkdir -p "$BIN_DIR"

# ---- Download with progress ----
info "Downloading $APP v${VERSION}..."
TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

TMP_FILE="$TMP_DIR/$FILENAME"

if [[ -t 2 ]]; then
  curl -#fSL --retry 3 -o "$TMP_FILE" "$DOWNLOAD_URL" 2>&1
else
  curl -fsSL --retry 3 -o "$TMP_FILE" "$DOWNLOAD_URL"
fi

[[ -f "$TMP_FILE" ]] || error "Download failed"

# ---- Checksum verification ----
CHECKSUMS_URL="https://github.com/$REPO/releases/download/v${VERSION}/checksums.txt"
TMP_CHECKSUMS="$TMP_DIR/checksums.txt"
curl -fsSL --retry 3 -o "$TMP_CHECKSUMS" "$CHECKSUMS_URL" 2>/dev/null || true

if command -v sha256sum &>/dev/null; then
  SHA_CMD="sha256sum"
elif command -v shasum &>/dev/null; then
  SHA_CMD="shasum -a 256"
else
  SHA_CMD=""
fi

if [[ -n "$SHA_CMD" && -f "$TMP_CHECKSUMS" ]]; then
  EXPECTED=$(grep -E "  ${FILENAME}$" "$TMP_CHECKSUMS" | awk '{print $1}')
  if [[ -n "$EXPECTED" ]]; then
    ACTUAL=$($SHA_CMD "$TMP_FILE" | awk '{print $1}')
    if [[ "$EXPECTED" != "$ACTUAL" ]]; then
      error "Checksum mismatch"
    fi
    success "Checksum verified."
  fi
fi

# ---- Extract ----
info "Extracting..."
if [[ "$archive_ext" == ".zip" ]]; then
  if command -v unzip &>/dev/null; then
    unzip -oq "$TMP_FILE" -d "$TMP_DIR"
  elif command -v 7z &>/dev/null; then
    7z x -o"$TMP_DIR" -y "$TMP_FILE" &>/dev/null
  else
    error "Need unzip or 7z to extract"
  fi
else
  tar -xzf "$TMP_FILE" -C "$TMP_DIR"
fi

BINARY_PATH=$(find "$TMP_DIR" -name "$APP" -type f)
[[ -n "$BINARY_PATH" ]] || error "Binary not found in archive"

# ---- Install ----
mv "$BINARY_PATH" "$BIN_DIR/$APP"
chmod 755 "$BIN_DIR/$APP"

success "$APP v${VERSION} installed successfully to $BIN_DIR/$APP"

# ---- PATH setup ----
add_to_path() {
  local config_file="$1"
  local line="$2"
  if ! grep -Fxq "$line" "$config_file" 2>/dev/null; then
    echo -e "\n# $APP" >> "$config_file"
    echo "$line" >> "$config_file"
    info "Added to PATH in $config_file"
  fi
}

if [[ "$no_modify_path" == "false" && "$os" != "Windows" ]]; then
  PATH_LINE="export PATH=\"\$HOME/.bookleaf/bin:\$PATH\""
  current_shell=$(basename "$SHELL")

  case "$current_shell" in
    fish)
      config_file="$HOME/.config/fish/config.fish"
      if [[ ! -f "$config_file" ]]; then
        mkdir -p "$(dirname "$config_file")"
        touch "$config_file"
      fi
      add_to_path "$config_file" "fish_add_path \$HOME/.bookleaf/bin"
      ;;
    zsh)
      for f in "${ZDOTDIR:-$HOME}/.zshrc" "$HOME/.zshenv"; do
        if [[ -f "$f" ]]; then
          add_to_path "$f" "$PATH_LINE"
          break
        fi
      done
      ;;
    bash)
      for f in "$HOME/.bashrc" "$HOME/.bash_profile"; do
        if [[ -f "$f" ]]; then
          add_to_path "$f" "$PATH_LINE"
          break
        fi
      done
      ;;
  esac
fi

if [[ -n "${GITHUB_ACTIONS:-}" ]]; then
  echo "$BIN_DIR" >> "$GITHUB_PATH"
fi

echo ""
info_bold "To get started, run:"
echo ""
info_bold "  $APP --help"
echo ""
