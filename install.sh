#!/usr/bin/env bash
set -e

REPO="traiproject/yaml-schema-router"
PROJECT_NAME="yaml-schema-router"

# Detect OS
OS="$(uname -s)"
case "${OS}" in
    Linux*)     OS_NAME="linux";;
    Darwin*)    OS_NAME="macOS";;
    *)          echo "Unsupported OS: ${OS}"; exit 1;;
esac

# Detect Architecture
ARCH="$(uname -m)"
case "${ARCH}" in
    x86_64)     ARCH_NAME="x86_64" ;;
    amd64)      ARCH_NAME="x86_64" ;;
    arm64)      ARCH_NAME="arm64" ;;
    aarch64)    ARCH_NAME="arm64" ;;
    *)          echo "Unsupported Architecture: ${ARCH}"; exit 1;;
esac

echo "Detecting latest version for ${PROJECT_NAME}..."
# Fetch the latest release tag from GitHub API
VERSION=$(curl -s "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$VERSION" ]; then
    echo "Error: Could not fetch the latest version."
    exit 1
fi

# Construct file name based on your .goreleaser.yaml template
# e.g., yaml-schema-router_v1.0.0_linux_x86_64.tar.gz
TAR_FILE="${PROJECT_NAME}_${VERSION#v}_${OS_NAME}_${ARCH_NAME}.tar.gz"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${TAR_FILE}"

TMP_DIR=$(mktemp -d)
echo "Downloading ${PROJECT_NAME} ${VERSION} for ${OS_NAME} ${ARCH_NAME}..."
curl -sL "${DOWNLOAD_URL}" -o "${TMP_DIR}/${TAR_FILE}"

echo "Extracting..."
tar -xzf "${TMP_DIR}/${TAR_FILE}" -C "${TMP_DIR}"

# Determine installation directory
INSTALL_DIR="/usr/local/bin"
if [ "$(id -u)" -ne 0 ]; then
    INSTALL_DIR="${HOME}/.local/bin"
    mkdir -p "${INSTALL_DIR}"
fi

echo "Installing to ${INSTALL_DIR}..."
mv "${TMP_DIR}/${PROJECT_NAME}" "${INSTALL_DIR}/${PROJECT_NAME}"
chmod +x "${INSTALL_DIR}/${PROJECT_NAME}"

# Clean up
rm -rf "${TMP_DIR}"

echo ""
echo "Successfully installed ${PROJECT_NAME} ${VERSION} to ${INSTALL_DIR}!"
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo "Note: ${INSTALL_DIR} is not in your PATH."
    echo "Please add 'export PATH=\$PATH:${INSTALL_DIR}' to your ~/.bashrc or ~/.zshrc."
fi
