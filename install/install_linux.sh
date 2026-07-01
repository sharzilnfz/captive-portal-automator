#!/usr/bin/env bash
set -euo pipefail

BINARY="autocap"
INSTALL_DIR="/usr/local/bin"
SERVICE_DIR="$HOME/.config/systemd/user"
DISPATCHER_DIR="/etc/NetworkManager/dispatcher.d"

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

# Install systemd unit
echo "Installing systemd user service..."
mkdir -p "${SERVICE_DIR}"
cp "$(dirname "$0")/autocap.service" "${SERVICE_DIR}/autocap.service"
systemctl --user daemon-reload
systemctl --user enable autocap.service

# Install NetworkManager dispatcher (optional)
if [ -d "${DISPATCHER_DIR}" ]; then
    echo "Installing NetworkManager dispatcher..."
    sudo tee "${DISPATCHER_DIR}/99-autocap" > /dev/null << 'EOF'
#!/bin/bash
if [ "$2" = "up" ]; then
    /usr/local/bin/autocap &
fi
EOF
    sudo chmod 755 "${DISPATCHER_DIR}/99-autocap"
fi

echo "Done! AutoCap is now installed."
echo "To run manually: autocap"
echo "To uninstall: autocap uninstall"
