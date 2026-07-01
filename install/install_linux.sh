#!/usr/bin/env bash
set -euo pipefail

BINARY="capauto"
INSTALL_DIR="/usr/local/bin"
SERVICE_DIR="$HOME/.config/systemd/user"
DISPATCHER_DIR="/etc/NetworkManager/dispatcher.d"

echo "Installing CapAuto..."

# Build if binary doesn't exist
if [ ! -f "${BINARY}" ]; then
    echo "Building..."
    go build -o "${BINARY}" ./cmd/capauto
fi

# Install binary
echo "Copying binary to ${INSTALL_DIR}..."
sudo cp "${BINARY}" "${INSTALL_DIR}/${BINARY}"
sudo chmod 755 "${INSTALL_DIR}/${BINARY}"

# Install systemd unit
echo "Installing systemd user service..."
mkdir -p "${SERVICE_DIR}"
cp "$(dirname "$0")/capauto.service" "${SERVICE_DIR}/capauto.service"
systemctl --user daemon-reload
systemctl --user enable capauto.service

# Install NetworkManager dispatcher (optional)
if [ -d "${DISPATCHER_DIR}" ]; then
    echo "Installing NetworkManager dispatcher..."
    sudo tee "${DISPATCHER_DIR}/99-capauto" > /dev/null << 'EOF'
#!/bin/bash
if [ "$2" = "up" ]; then
    /usr/local/bin/capauto &
fi
EOF
    sudo chmod 755 "${DISPATCHER_DIR}/99-capauto"
fi

echo "Done! CapAuto is now installed."
echo "To run manually: capauto"
echo "To uninstall: capauto uninstall"
