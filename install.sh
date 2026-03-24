#!/bin/sh
# near-intents + portfolio CLI Installer
# Usage: curl -fsSL https://raw.githubusercontent.com/FlipsideCrypto/near-intents-cli/main/install.sh | sh
# Or with specific version: VERSION=v0.1.0 curl -fsSL ... | sh

set -e

REPO="FlipsideCrypto/near-intents-cli"
GITHUB_API="https://api.github.com/repos/${REPO}/releases/latest"
GITHUB_DOWNLOAD="https://github.com/${REPO}/releases/download"

# Colors (only if terminal supports them)
if [ -t 1 ]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[1;33m'
    BLUE='\033[0;34m'
    NC='\033[0m'
else
    RED=''
    GREEN=''
    YELLOW=''
    BLUE=''
    NC=''
fi

info()    { printf "${BLUE}==>${NC} %s\n" "$1"; }
success() { printf "${GREEN}==>${NC} %s\n" "$1"; }
warn()    { printf "${YELLOW}Warning:${NC} %s\n" "$1"; }
error()   { printf "${RED}Error:${NC} %s\n" "$1" >&2; exit 1; }

detect_os() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    case "$OS" in
        darwin)          echo "darwin" ;;
        linux)           echo "linux" ;;
        mingw*|msys*|cygwin*) echo "windows" ;;
        *)               error "Unsupported operating system: $OS" ;;
    esac
}

detect_arch() {
    ARCH=$(uname -m)
    case "$ARCH" in
        x86_64|amd64)  echo "amd64" ;;
        arm64|aarch64) echo "arm64" ;;
        *)             error "Unsupported architecture: $ARCH" ;;
    esac
}

check_dependencies() {
    for cmd in curl; do
        if ! command -v "$cmd" >/dev/null 2>&1; then
            error "Required command not found: $cmd"
        fi
    done
    OS_CHECK=$(detect_os)
    if [ "$OS_CHECK" = "windows" ]; then
        if ! command -v unzip >/dev/null 2>&1; then
            error "Required command not found: unzip (install via 'pacman -S unzip' in MSYS2 or Git Bash)"
        fi
    else
        if ! command -v tar >/dev/null 2>&1; then
            error "Required command not found: tar"
        fi
    fi
}

get_latest_version() {
    info "Fetching latest version..." >&2
    LATEST=$(curl -fsSL "$GITHUB_API" 2>/dev/null | grep '"tag_name"' | head -1 | sed -E 's/.*"([^"]+)".*/\1/')
    if [ -z "$LATEST" ]; then
        error "Failed to fetch latest version from GitHub. Set VERSION explicitly: VERSION=v0.1.0 curl -fsSL ... | sh"
    fi
    echo "$LATEST"
}

determine_install_dir() {
    if [ -n "$INSTALL_DIR" ]; then
        echo "$INSTALL_DIR"
        return
    fi

    # Default to ~/.local/bin — no sudo required
    LOCAL_BIN="${HOME}/.local/bin"
    mkdir -p "$LOCAL_BIN"
    echo "$LOCAL_BIN"
}

check_path() {
    INSTALL_DIR="$1"
    case ":$PATH:" in
        *":${INSTALL_DIR}:"*) return 0 ;;
    esac

    echo ""
    warn "${INSTALL_DIR} is not in your PATH"
    echo ""
    echo "Add it to your shell configuration:"
    echo ""
    SHELL_NAME=$(basename "$SHELL")
    case "$SHELL_NAME" in
        bash) echo "  echo 'export PATH=\"\$PATH:${INSTALL_DIR}\"' >> ~/.bashrc && source ~/.bashrc" ;;
        zsh)  echo "  echo 'export PATH=\"\$PATH:${INSTALL_DIR}\"' >> ~/.zshrc && source ~/.zshrc" ;;
        fish) echo "  fish_add_path ${INSTALL_DIR}" ;;
        *)    echo "  export PATH=\"\$PATH:${INSTALL_DIR}\"" ;;
    esac
    echo ""
}

download_and_install() {
    TOOL="$1"
    VERSION="$2"
    OS="$3"
    ARCH="$4"
    TMPDIR="$5"
    INSTALL_DIR="$6"

    VERSION_NUM="${VERSION#v}"

    # Windows uses .zip archives and .exe binaries
    if [ "$OS" = "windows" ]; then
        ARCHIVE="${TOOL}_${VERSION_NUM}_${OS}_${ARCH}.zip"
        BINARY="${TOOL}.exe"
    else
        ARCHIVE="${TOOL}_${VERSION_NUM}_${OS}_${ARCH}.tar.gz"
        BINARY="${TOOL}"
    fi

    URL="${GITHUB_DOWNLOAD}/${VERSION}/${ARCHIVE}"
    CHECKSUM_URL="${GITHUB_DOWNLOAD}/${VERSION}/checksums.txt"

    info "Downloading ${TOOL} ${VERSION}..."
    if ! curl -fsSL -o "${TMPDIR}/${ARCHIVE}" "$URL"; then
        error "Failed to download ${TOOL} from ${URL}\nCheck that version ${VERSION} exists at https://github.com/${REPO}/releases"
    fi

    # Checksum verification
    if curl -fsSL -o "${TMPDIR}/checksums.txt" "$CHECKSUM_URL" 2>/dev/null; then
        EXPECTED=$(grep "${ARCHIVE}" "${TMPDIR}/checksums.txt" | awk '{print $1}')
        if [ -n "$EXPECTED" ]; then
            if command -v sha256sum >/dev/null 2>&1; then
                ACTUAL=$(sha256sum "${TMPDIR}/${ARCHIVE}" | awk '{print $1}')
            elif command -v shasum >/dev/null 2>&1; then
                ACTUAL=$(shasum -a 256 "${TMPDIR}/${ARCHIVE}" | awk '{print $1}')
            fi
            if [ -n "$ACTUAL" ] && [ "$EXPECTED" != "$ACTUAL" ]; then
                error "Checksum verification failed for ${TOOL}. The download may be corrupted."
            fi
            success "Checksum verified"
        fi
    else
        warn "Could not fetch checksums — skipping verification"
    fi

    # Extract archive
    if [ "$OS" = "windows" ]; then
        unzip -qo "${TMPDIR}/${ARCHIVE}" -d "${TMPDIR}"
    else
        tar -xzf "${TMPDIR}/${ARCHIVE}" -C "${TMPDIR}"
    fi

    if [ ! -f "${TMPDIR}/${BINARY}" ]; then
        error "Binary not found after extraction"
    fi

    chmod +x "${TMPDIR}/${BINARY}"

    if [ ! -w "$INSTALL_DIR" ]; then
        error "Cannot write to ${INSTALL_DIR}. Set INSTALL_DIR to a writable path."
    fi

    mv "${TMPDIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
    success "Installed ${TOOL} to ${INSTALL_DIR}/${BINARY}"
}

main() {
    echo ""
    echo "near-intents + portfolio Installer"
    echo "==================================="
    echo ""

    check_dependencies

    OS=$(detect_os)
    ARCH=$(detect_arch)
    info "Detected platform: ${OS}/${ARCH}"

    if [ -n "$VERSION" ]; then
        info "Using specified version: ${VERSION}"
    else
        VERSION=$(get_latest_version)
        info "Latest version: ${VERSION}"
    fi

    TMPDIR=$(mktemp -d)
    trap 'rm -rf "$TMPDIR"' EXIT

    INSTALL_DIR=$(determine_install_dir)
    info "Installing to: ${INSTALL_DIR}"

    download_and_install "near-intents" "$VERSION" "$OS" "$ARCH" "$TMPDIR" "$INSTALL_DIR"
    download_and_install "portfolio"    "$VERSION" "$OS" "$ARCH" "$TMPDIR" "$INSTALL_DIR"

    check_path "$INSTALL_DIR"

    echo ""
    success "Installation complete!"
    echo ""
    echo "Get started:"
    echo "  near-intents llm onboard    # learn the swap CLI"
    echo "  portfolio llm onboard       # learn the portfolio CLI"
    echo ""
}

main
