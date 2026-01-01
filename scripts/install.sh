#!/bin/bash
# APIGate installer script
# Usage: curl -fsSL https://raw.githubusercontent.com/artpar/apigate/main/scripts/install.sh | bash

set -e

REPO="artpar/apigate"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Detect OS and architecture
detect_platform() {
    local os arch

    os="$(uname -s | tr '[:upper:]' '[:lower:]')"
    arch="$(uname -m)"

    case "$os" in
        linux) os="linux" ;;
        darwin) os="darwin" ;;
        mingw*|msys*|cygwin*) os="windows" ;;
        *) echo "Unsupported OS: $os" >&2; exit 1 ;;
    esac

    case "$arch" in
        x86_64|amd64) arch="amd64" ;;
        aarch64|arm64) arch="arm64" ;;
        *) echo "Unsupported architecture: $arch" >&2; exit 1 ;;
    esac

    echo "${os}-${arch}"
}

# Get latest release version
get_latest_version() {
    curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" |
        grep '"tag_name":' |
        sed -E 's/.*"([^"]+)".*/\1/'
}

main() {
    local platform version download_url tmp_dir

    echo "Detecting platform..."
    platform="$(detect_platform)"
    echo "Platform: $platform"

    echo "Fetching latest version..."
    version="$(get_latest_version)"
    echo "Latest version: $version"

    # Determine file extension
    local ext="tar.gz"
    if [[ "$platform" == windows-* ]]; then
        ext="zip"
    fi

    download_url="https://github.com/${REPO}/releases/download/${version}/apigate-${platform}.${ext}"
    echo "Downloading from: $download_url"

    tmp_dir="$(mktemp -d)"
    trap "rm -rf '$tmp_dir'" EXIT

    cd "$tmp_dir"

    if [[ "$ext" == "zip" ]]; then
        curl -fsSL "$download_url" -o apigate.zip
        unzip -q apigate.zip
    else
        curl -fsSL "$download_url" | tar xz
    fi

    # Find the binary
    local binary
    if [[ "$platform" == windows-* ]]; then
        binary="apigate-${platform}.exe"
    else
        binary="apigate-${platform}"
    fi

    if [[ ! -f "$binary" ]]; then
        echo "Error: Binary not found in archive" >&2
        exit 1
    fi

    chmod +x "$binary"

    # Install
    echo "Installing to $INSTALL_DIR..."
    if [[ -w "$INSTALL_DIR" ]]; then
        mv "$binary" "$INSTALL_DIR/apigate"
    else
        sudo mv "$binary" "$INSTALL_DIR/apigate"
    fi

    echo ""
    echo "APIGate ${version} installed successfully!"
    echo "Run 'apigate --help' to get started."
}

main "$@"
