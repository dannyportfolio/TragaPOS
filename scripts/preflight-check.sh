#!/usr/bin/env bash
# ==============================================================================
# Pre-flight Check Script for WSL Kali Linux & Docker Environment
# Project: Headless Multi-Industry POS & Commerce Engine (Ticket-Prod)
# ==============================================================================

set -euo pipefail

BOLD="\033[1m"
GREEN="\033[0;32m"
YELLOW="\033[0;33m"
RED="\033[0;31m"
RESET="\033[0m"

echo -e "${BOLD}======================================================${RESET}"
echo -e "${BOLD}   WSL Kali Linux & Docker Pre-flight Diagnostic      ${RESET}"
echo -e "${BOLD}======================================================${RESET}"

ERRORS=0
WARNINGS=0

# 1. Check OS / Kernel / WSL detection
echo -n "[*] Detecting OS and WSL environment... "
if grep -qi "microsoft" /proc/version 2>/dev/null; then
    DISTRO_NAME="Unknown"
    if [ -f /etc/os-release ]; then
        DISTRO_NAME=$(grep -E '^PRETTY_NAME=' /etc/os-release | cut -d= -f2 | tr -d '"')
    fi
    echo -e "${GREEN}OK${RESET} (WSL Detected: ${DISTRO_NAME})"
else
    echo -e "${YELLOW}Notice${RESET} (Running on native Linux / container environment)"
fi

# 2. Check Docker CLI
echo -n "[*] Checking Docker CLI... "
if command -v docker >/dev/null 2>&1; then
    DOCKER_VER=$(docker --version)
    echo -e "${GREEN}OK${RESET} (${DOCKER_VER})"
else
    echo -e "${RED}FAILED${RESET} (Docker binary not found in PATH)"
    ERRORS=$((ERRORS + 1))
fi

# 3. Check Docker Daemon Connectivity
echo -n "[*] Checking Docker Daemon status... "
if docker info >/dev/null 2>&1; then
    echo -e "${GREEN}OK${RESET} (Docker daemon is running and reachable)"
else
    echo -e "${RED}FAILED${RESET} (Cannot connect to Docker daemon)"
    echo -e "    ${YELLOW}Suggestion:${RESET} If running Docker Desktop, ensure 'WSL Integration' is enabled for Kali."
    echo -e "    ${YELLOW}Or start service:${RESET} sudo service docker start (or sudo systemctl start docker)"
    ERRORS=$((ERRORS + 1))
fi

# 4. Check Docker Compose v2
echo -n "[*] Checking Docker Compose v2... "
if docker compose version >/dev/null 2>&1; then
    COMPOSE_VER=$(docker compose version)
    echo -e "${GREEN}OK${RESET} (${COMPOSE_VER})"
elif command -v docker-compose >/dev/null 2>&1; then
    COMPOSE_VER=$(docker-compose --version)
    echo -e "${YELLOW}Notice${RESET} (${COMPOSE_VER} - standalone binary found)"
else
    echo -e "${RED}FAILED${RESET} (Docker Compose plugin or binary not found)"
    ERRORS=$((ERRORS + 1))
fi

# 5. Check System Memory & Resources
echo -n "[*] Checking available RAM... "
MEM_TOTAL_KB=$(grep MemTotal /proc/meminfo | awk '{print $2}')
MEM_TOTAL_MB=$((MEM_TOTAL_KB / 1024))
if [ "$MEM_TOTAL_MB" -ge 2048 ]; then
    echo -e "${GREEN}OK${RESET} (${MEM_TOTAL_MB} MB allocated to WSL)"
else
    echo -e "${YELLOW}WARNING${RESET} (${MEM_TOTAL_MB} MB is under recommended 2048 MB for Postgres + 2 Go Engines + Vite)"
    WARNINGS=$((WARNINGS + 1))
fi

# 6. Check Port Availability
echo "[*] Checking required port bindings:"
PORTS=(5432 8080 8081 3000)
PORT_DESC=("PostgreSQL" "POS Cloud Engine" "POS Edge Engine" "Web Cashier UI")

for i in "${!PORTS[@]}"; do
    PORT="${PORTS[$i]}"
    DESC="${PORT_DESC[$i]}"
    echo -n "    - Port ${PORT} (${DESC}): "
    if command -v ss >/dev/null 2>&1; then
        if ss -tuln | grep -q ":${PORT} "; then
            echo -e "${YELLOW}BUSY${RESET} (Port is currently in use!)"
            WARNINGS=$((WARNINGS + 1))
        else
            echo -e "${GREEN}FREE${RESET}"
        fi
    elif command -v netstat >/dev/null 2>&1; then
        if netstat -tuln | grep -q ":${PORT} "; then
            echo -e "${YELLOW}BUSY${RESET} (Port is currently in use!)"
            WARNINGS=$((WARNINGS + 1))
        else
            echo -e "${GREEN}FREE${RESET}"
        fi
    else
        echo -e "${YELLOW}SKIPPED${RESET} (ss/netstat not installed)"
    fi
done

echo -e "\n${BOLD}======================================================${RESET}"
if [ "$ERRORS" -eq 0 ]; then
    echo -e "${GREEN}${BOLD}Pre-flight check PASSED!${RESET} (${WARNINGS} warnings)"
    exit 0
else
    echo -e "${RED}${BOLD}Pre-flight check FAILED with ${ERRORS} error(s).${RESET}"
    echo -e "Please address the errors above before launching containers."
    exit 1
fi
