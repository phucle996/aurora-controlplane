#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TLS_ROOT="${ROOT_DIR}/.local/tls"
CA_DIR="${TLS_ROOT}/ca"
PSQL_DIR="${TLS_ROOT}/postgres"
REDIS_DIR="${TLS_ROOT}/redis"
COMPOSE_FILE="${ROOT_DIR}/docker-compose.local.yml"
ENV_FILE="${ROOT_DIR}/package/env.prod"

mkdir -p "$CA_DIR" "$PSQL_DIR" "$REDIS_DIR"

if [[ ! -f "${CA_DIR}/ca.crt" || ! -f "${CA_DIR}/ca.key" ]]; then
  openssl req -x509 -new -nodes -newkey rsa:4096 \
    -days 3650 \
    -keyout "${CA_DIR}/ca.key" \
    -out "${CA_DIR}/ca.crt" \
    -subj "/CN=Aurora Controlplane Local CA"
fi

make_server_cert() {
  local name="$1"
  local dir="$2"
  local san="$3"

  if [[ -f "${dir}/server.crt" && -f "${dir}/server.key" ]]; then
    return
  fi

  local csr_file="${dir}/server.csr"
  local cnf_file="${dir}/openssl.cnf"

  openssl req -new -nodes -newkey rsa:2048 \
    -keyout "${dir}/server.key" \
    -out "${csr_file}" \
    -subj "/CN=${name}"

  cat > "${cnf_file}" <<EOF
[v3_req]
subjectAltName = ${san}
extendedKeyUsage = serverAuth
EOF

  openssl x509 -req \
    -in "${csr_file}" \
    -CA "${CA_DIR}/ca.crt" \
    -CAkey "${CA_DIR}/ca.key" \
    -CAcreateserial \
    -out "${dir}/server.crt" \
    -days 825 \
    -sha256 \
    -extfile "${cnf_file}" \
    -extensions v3_req

  rm -f "${csr_file}" "${cnf_file}"
  chmod 600 "${dir}/server.key"
}

make_server_cert "psql" "$PSQL_DIR" "DNS:psql,DNS:localhost,IP:127.0.0.1"
make_server_cert "redis" "$REDIS_DIR" "DNS:redis,DNS:localhost,IP:127.0.0.1"

chown 70:70 "${PSQL_DIR}/server.key" 2>/dev/null || true
chmod 600 "${PSQL_DIR}/server.key"
chown 999:1000 "${REDIS_DIR}/server.key" 2>/dev/null || true
chmod 600 "${REDIS_DIR}/server.key"
chmod 644 "${CA_DIR}/ca.crt" "${PSQL_DIR}/server.crt" "${REDIS_DIR}/server.crt"

if command -v systemctl >/dev/null 2>&1; then
  systemctl enable --now docker >/dev/null 2>&1 || true
fi

docker compose \
  --env-file "$ENV_FILE" \
  -f "$COMPOSE_FILE" \
  up -d
