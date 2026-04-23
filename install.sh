#!/bin/bash
set -eo pipefail

# Default variables
ENV_FILE_PATH=""
INSTALL_BIN_PATH="/usr/local/bin/aurora-controlplane"
CONFIG_DIR="/etc/aurora-controlplane"
ENV_DEST_PATH="${CONFIG_DIR}/.env"
TLS_SRC_DIR=".local/tls"
TLS_DEST_DIR="${CONFIG_DIR}/tls"
SYSTEMD_SERVICE_DEST="/etc/systemd/system/aurora-controlplane.service"
ADMIN_API_TOKEN_PATH="/var/lib/aurora-controlplane/admin-api-token"
ADMIN_API_TOKEN_FALLBACK_PATH="/tmp/aurora-controlplane-admin-api-token"

generate_master_key() {
    if command -v openssl >/dev/null 2>&1; then
        openssl rand -base64 32 | tr -d '\n'
        return
    fi

    head -c 32 /dev/urandom | base64 | tr -d '\n'
}

ensure_core_secret_master_key() {
    local current_key
    current_key="$(grep -E '^CORE_SECRET_MASTER_KEY=' "$ENV_FILE_PATH" | head -n1 | cut -d'=' -f2- | tr -d '"')"
    if [ -n "$current_key" ]; then
        return
    fi

    local generated_key
    generated_key="$(generate_master_key)"

    local tmp_file
    tmp_file="$(mktemp)"

    awk -v key="$generated_key" '
        BEGIN { replaced = 0 }
        /^CORE_SECRET_MASTER_KEY=/ {
            print "CORE_SECRET_MASTER_KEY=\"" key "\""
            replaced = 1
            next
        }
        { print }
        END {
            if (!replaced) {
                print "CORE_SECRET_MASTER_KEY=\"" key "\""
            }
        }
    ' "$ENV_FILE_PATH" > "$tmp_file"

    mv "$tmp_file" "$ENV_FILE_PATH"
    echo " Generated CORE_SECRET_MASTER_KEY in $ENV_FILE_PATH"
}

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

ensure_core_secret_master_key

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
# Clear previous bootstrap token outputs to avoid stale key print.
sudo rm -f "$ADMIN_API_TOKEN_PATH" "$ADMIN_API_TOKEN_FALLBACK_PATH" >/dev/null 2>&1 || true

# Install the new binary atomically to avoid "Text file busy" errors.
sudo install -m 755 bin/aurora-controlplane "${INSTALL_BIN_PATH}.new"
sudo mv -f "${INSTALL_BIN_PATH}.new" "$INSTALL_BIN_PATH"

# Create config directory and copy env file
sudo mkdir -p "$CONFIG_DIR"
sudo cp "$ENV_FILE_PATH" "$ENV_DEST_PATH"
sudo chmod 600 "$ENV_DEST_PATH"

if [ -d "$TLS_SRC_DIR" ]; then
    sudo rm -rf "$TLS_DEST_DIR"
    sudo mkdir -p "$TLS_DEST_DIR"
    sudo cp -R "$TLS_SRC_DIR"/. "$TLS_DEST_DIR"/
    sudo chmod 700 "$TLS_DEST_DIR"
    sudo find "$TLS_DEST_DIR" -type d -exec chmod 700 {} +
    sudo find "$TLS_DEST_DIR" -type f -name '*.crt' -exec chmod 644 {} +
    sudo find "$TLS_DEST_DIR" -type f -name '*.key' -exec chmod 600 {} +
fi

echo "[4/4] Installing and starting systemd service..."
sudo cp package/aurora-controlplane.service "$SYSTEMD_SERVICE_DEST"
sudo systemctl daemon-reload
sudo systemctl enable aurora-controlplane.service
sudo systemctl restart aurora-controlplane.service

ADMIN_API_KEY=""
if sudo test -s "$ADMIN_API_TOKEN_PATH"; then
    ADMIN_API_KEY="$(sudo cat "$ADMIN_API_TOKEN_PATH" | tr -d '\r\n')"
    sudo rm -f "$ADMIN_API_TOKEN_PATH"
elif sudo test -s "$ADMIN_API_TOKEN_FALLBACK_PATH"; then
    ADMIN_API_KEY="$(sudo cat "$ADMIN_API_TOKEN_FALLBACK_PATH" | tr -d '\r\n')"
    sudo rm -f "$ADMIN_API_TOKEN_FALLBACK_PATH"
fi

echo "=========================================="
echo " Service Status"
sudo systemctl status aurora-controlplane --no-pager -l
echo " Access URL: http://localhost:${APP_HTTP_PORT}"
if [ -n "$ADMIN_API_KEY" ]; then
    echo " Admin API key: ${ADMIN_API_KEY}"
fi
echo "=========================================="
