#!/bin/sh
set -e

# Install near-intents and portfolio CLI tools
# Usage: curl -fsSL https://raw.githubusercontent.com/FlipsideCrypto/near-intents-cli/main/install.sh | sh

REPO="FlipsideCrypto/near-intents-cli"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
VERSION="${VERSION:-latest}"

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64) ARCH="arm64" ;;
  arm64)   ARCH="arm64" ;;
  *)       echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

case "$OS" in
  linux)  ;;
  darwin) ;;
  *)      echo "Unsupported OS: $OS"; exit 1 ;;
esac

# Resolve version
if [ "$VERSION" = "latest" ]; then
  VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
  if [ -z "$VERSION" ]; then
    echo "Failed to fetch latest version. Set VERSION explicitly:"
    echo "  VERSION=v0.1.0 curl -fsSL ... | sh"
    exit 1
  fi
fi

# Strip leading v for archive names
VERSION_NUM="${VERSION#v}"

echo "Installing near-intents and portfolio ${VERSION} for ${OS}/${ARCH}..."

TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

download_and_install() {
  TOOL=$1
  ARCHIVE="${TOOL}_${VERSION_NUM}_${OS}_${ARCH}.tar.gz"
  URL="https://github.com/${REPO}/releases/download/${VERSION}/${ARCHIVE}"

  echo "  Downloading ${TOOL}..."
  if ! curl -fsSL "$URL" -o "${TMP}/${ARCHIVE}"; then
    echo "  Failed to download ${TOOL} from ${URL}"
    echo "  Check that version ${VERSION} exists at https://github.com/${REPO}/releases"
    return 1
  fi

  tar -xzf "${TMP}/${ARCHIVE}" -C "$TMP"

  if [ -w "$INSTALL_DIR" ]; then
    mv "${TMP}/${TOOL}" "${INSTALL_DIR}/${TOOL}"
  else
    echo "  Need sudo to install to ${INSTALL_DIR}"
    sudo mv "${TMP}/${TOOL}" "${INSTALL_DIR}/${TOOL}"
  fi

  chmod +x "${INSTALL_DIR}/${TOOL}"
  echo "  Installed ${TOOL} to ${INSTALL_DIR}/${TOOL}"
}

download_and_install "near-intents"
download_and_install "portfolio"

echo ""
echo "Done! Verify with:"
echo "  near-intents --version"
echo "  portfolio --version"
echo ""
echo "Get started:"
echo "  near-intents llm onboard    # learn the swap CLI"
echo "  portfolio llm onboard       # learn the portfolio CLI"
