#!/usr/bin/env bash
set -euo pipefail

BINARY="autocap"
INSTALL_DIR="/usr/local/bin"
PLIST_NAME="com.sharzil.autocap.plist"
PLIST_SRC="$(dirname "$0")/${PLIST_NAME}"
PLIST_DST="$HOME/Library/LaunchAgents/${PLIST_NAME}"

echo "Installing AutoCap..."

# Build if binary doesn't exist
if [ ! -f "${BINARY}" ]; then
    echo "Building..."
    go build -o "${BINARY}" ./cmd/autocap
fi

# Install binary
echo "Copying binary to ${INSTALL_DIR}..."
sudo cp "${BINARY}" "${INSTALL_DIR}/${BINARY}"
sudo chmod 755 "${INSTALL_DIR}/${BINARY}"

# Install LaunchAgent
echo "Installing LaunchAgent..."
mkdir -p "$HOME/Library/LaunchAgents"
cp "${PLIST_SRC}" "${PLIST_DST}"

# Unload if already loaded
launchctl bootout "gui/$(id -u)" "${PLIST_DST}" 2>/dev/null || true

# Load
launchctl bootstrap "gui/$(id -u)" "${PLIST_DST}"

echo "Done! AutoCap is now running in the background."
echo "Logs: /tmp/autocap.log"
echo "To uninstall: autocap uninstall"
