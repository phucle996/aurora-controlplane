#!/bin/bash
set -eo pipefail

# Default variables
ENV_FILE_PATH=""
INSTALL_BIN_PATH="/usr/local/bin/aurora-controlplane"
CONFIG_DIR="/etc/aurora-controlplane"
ENV_DEST_PATH="${CONFIG_DIR}/.env"
TLS_SRC_DIR=".local/tls"
TLS_DEST_DIR="${CONFIG_DIR}/tls"
MIGRATIONS_SRC_DIR="internal/iam/migrations"
MIGRATIONS_DEST_DIR="${CONFIG_DIR}/migrations/iam"
SYSTEMD_SERVICE_DEST="/etc/systemd/system/aurora-controlplane.service"

# Print usage function
usage() {
    echo "Usage: $0 -e <path-to-env-file>"
    echo "  -e    Path to the environment configuration file (required)"
    exit 1
}

# Parse arguments
while getopts "e:" opt; do
    case "$opt" in
        e)
            ENV_FILE_PATH="$OPTARG"
            ;;
        *)
            usage
            ;;
    esac
done

if [ -z "$ENV_FILE_PATH" ]; then
    echo "Error: Environment file is required."
    usage
fi

if [ ! -f "$ENV_FILE_PATH" ]; then
    echo "Error: Environment file not found at $ENV_FILE_PATH"
    exit 1
fi

APP_HTTP_PORT="$(grep -E '^APP_HTTP_PORT=' "$ENV_FILE_PATH" | head -n1 | cut -d'=' -f2 | tr -d '"')"
APP_HTTP_PORT="${APP_HTTP_PORT:-8080}"

echo "=========================================="
echo " Starting Aurora Controlplane Installer"
echo "=========================================="

echo "[1/4] Building UI..."
cd ui || exit
npm install
npm run build
# Fix permission denied by using sudo to remove existing root-owned dist
sudo rm -rf ../internal/http/dist
# Move the newly built out folder to dist
mv out ../internal/http/dist
cd ..

echo "[2/4] Building Go Binary..."
go build -o bin/aurora-controlplane cmd/server/main.go

echo "[3/4] Installing Binary and Configuration..."
# Stop the running service first so the binary can be replaced safely.
sudo systemctl stop aurora-controlplane.service >/dev/null 2>&1 || true

# Install the new binary atomically to avoid "Text file busy" errors.
sudo install -m 755 bin/aurora-controlplane "${INSTALL_BIN_PATH}.new"
sudo mv -f "${INSTALL_BIN_PATH}.new" "$INSTALL_BIN_PATH"

# Create config directory and copy env file
sudo mkdir -p "$CONFIG_DIR"
sudo cp "$ENV_FILE_PATH" "$ENV_DEST_PATH"
sudo chmod 600 "$ENV_DEST_PATH"

if [ -d "$MIGRATIONS_SRC_DIR" ]; then
    sudo rm -rf "$MIGRATIONS_DEST_DIR"
    sudo mkdir -p "$MIGRATIONS_DEST_DIR"
    sudo cp "$MIGRATIONS_SRC_DIR"/*.sql "$MIGRATIONS_DEST_DIR"/
    sudo chmod 755 "$(dirname "$MIGRATIONS_DEST_DIR")" "$MIGRATIONS_DEST_DIR"
    sudo chmod 644 "$MIGRATIONS_DEST_DIR"/*.sql
fi

if [ -d "$TLS_SRC_DIR" ]; then
    sudo rm -rf "$TLS_DEST_DIR"
    sudo mkdir -p "$TLS_DEST_DIR"
    sudo cp -R "$TLS_SRC_DIR"/. "$TLS_DEST_DIR"/
    sudo chmod 700 "$TLS_DEST_DIR"
    sudo chmod 700 "$TLS_DEST_DIR/ca" "$TLS_DEST_DIR/postgres" "$TLS_DEST_DIR/redis" 2>/dev/null || true
    sudo chmod 644 "$TLS_DEST_DIR/ca/ca.crt" "$TLS_DEST_DIR/postgres/server.crt" "$TLS_DEST_DIR/redis/server.crt" 2>/dev/null || true
    sudo chmod 600 "$TLS_DEST_DIR/postgres/server.key" "$TLS_DEST_DIR/redis/server.key" 2>/dev/null || true
fi

echo "[4/4] Installing and starting systemd service..."
sudo cp package/aurora-controlplane.service "$SYSTEMD_SERVICE_DEST"
sudo systemctl daemon-reload
sudo systemctl enable aurora-controlplane.service
sudo systemctl restart aurora-controlplane.service

echo "=========================================="
echo " Service Status"
sudo systemctl status aurora-controlplane --no-pager -l
echo " Access URL: http://localhost:${APP_HTTP_PORT}"
echo "=========================================="
