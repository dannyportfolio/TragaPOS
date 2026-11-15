#!/usr/bin/env bash
# ==============================================================================
# Docker Compose Orchestration Quickstart
# ==============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

cd "${ROOT_DIR}"

echo "======================================================"
echo " Starting Headless POS & Commerce Engine (Docker Compose)"
echo "======================================================"

# 1. Run Pre-flight Checks
if [ -f "${SCRIPT_DIR}/preflight-check.sh" ]; then
    bash "${SCRIPT_DIR}/preflight-check.sh" || {
        echo "Pre-flight checks failed. Please check errors above."
        exit 1
    }
fi

# 2. Check or generate .env file
if [ ! -f ".env" ]; then
    echo "[!] .env file not found. Generating fresh configuration with Ed25519 keys..."
    bash "${SCRIPT_DIR}/generate-keys.sh"
fi

# 3. Build & start containers
echo "[*] Building and starting containers via Docker Compose..."
docker compose up -d --build

# 4. Wait for health status
echo "[*] Waiting for services to become healthy..."
MAX_WAIT=60
ELAPSED=0

while [ $ELAPSED -lt $MAX_WAIT ]; do
    UNHEALTHY=$(docker compose ps --format json 2>/dev/null | grep -i '"Health":' | grep -iv '"healthy"' || true)
    if [ -z "${UNHEALTHY}" ]; then
        echo "[+] All containers are running and healthy!"
        break
    fi
    sleep 3
    ELAPSED=$((ELAPSED + 3))
    echo -n "."
done
echo ""

docker compose ps

echo "======================================================"
echo " System Endpoints Ready:"
echo "   - Web Cashier & KDS UI: http://localhost:3000"
echo "   - Cloud Central Hub:    http://localhost:8080 (Health: /healthz)"
echo "   - Edge Local Outlet:    http://localhost:8081 (Health: /healthz)"
echo "   - PostgreSQL Database:  localhost:5432"
echo "======================================================"
