#!/usr/bin/env bash
# ==============================================================================
# Kubernetes / K3s Orchestration Deployment inside WSL Kali Linux
# ==============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

cd "${ROOT_DIR}"

echo "=========================================================="
echo " Deploying Headless POS on K3s Kubernetes (WSL Kali Linux)"
echo "=========================================================="

# 1. Check or install K3s
if ! command -v k3s >/dev/null 2>&1; then
    echo "[*] K3s not detected. Installing lightweight K3s cluster..."
    curl -sfL https://get.k3s.io | sh -s - --write-kubeconfig-mode 644
    echo "[+] K3s installed successfully."
else
    echo "[+] K3s is already installed."
fi

# Ensure kubeconfig is accessible
export KUBECONFIG="${KUBECONFIG:-/etc/rancher/k3s/k3s.yaml}"
if [ ! -r "${KUBECONFIG}" ]; then
    echo "[!] /etc/rancher/k3s/k3s.yaml not readable without root. Using sudo..."
    sudo chmod 644 /etc/rancher/k3s/k3s.yaml || true
fi

# 2. Check cluster readiness
echo "[*] Checking cluster node readiness..."
kubectl wait --for=condition=Ready node --all --timeout=60s

# 3. Build container images with Docker
echo "[*] Building pos-engine and pos-web container images..."
docker build -t pos-engine:latest -f deploy/Dockerfile .
docker build -t pos-web:latest -f web/Dockerfile web/

# 4. Import images into K3s containerd
echo "[*] Importing Docker images into K3s containerd runtime..."
docker save pos-engine:latest | sudo k3s ctr images import -
docker save pos-web:latest | sudo k3s ctr images import -

# 5. Load or generate keys
if [ ! -f ".env" ]; then
    bash "${SCRIPT_DIR}/generate-keys.sh"
fi

PUB_KEY=$(grep '^ED25519_PUBLIC_KEY=' .env | cut -d= -f2)
PRIV_KEY=$(grep '^ED25519_PRIVATE_KEY=' .env | cut -d= -f2)
AUTH_TOKEN=$(grep '^EDGE_AUTH_TOKEN=' .env | cut -d= -f2)

# 6. Apply Cloud Namespace & Manifests
echo "[*] Deploying Cloud Central Hub (pos-cloud)..."
kubectl apply -f deploy/k8s/cloud-deployment.yaml

# Patch Cloud secrets with active keys
if [ -n "${PRIV_KEY}" ] && [ -n "${PUB_KEY}" ]; then
    kubectl -n pos-cloud create secret generic pos-cloud-secrets \
        --from-literal=DATABASE_URL="postgres://postgres:securepassword@pos-postgres.pos-cloud.svc.cluster.local:5432/pos_central?sslmode=disable" \
        --from-literal=ED25519_PRIVATE_KEY="${PRIV_KEY}" \
        --from-literal=ED25519_PUBLIC_KEY="${PUB_KEY}" \
        --from-literal=EDGE_AUTH_TOKEN="${AUTH_TOKEN}" \
        --dry-run=client -o yaml | kubectl apply -f -
fi

# 7. Apply Edge Namespace & Manifests
echo "[*] Deploying Edge Local Outlet Hub (pos-edge)..."
kubectl apply -f deploy/k8s/edge-k3s.yaml

# Patch Edge secrets with active public key
if [ -n "${PUB_KEY}" ]; then
    kubectl -n pos-edge create secret generic pos-edge-secrets \
        --from-literal=ED25519_PUBLIC_KEY="${PUB_KEY}" \
        --from-literal=EDGE_AUTH_TOKEN="${AUTH_TOKEN}" \
        --dry-run=client -o yaml | kubectl apply -f -
fi

# 8. Display Cluster State
echo "[*] Waiting for deployment rollout..."
kubectl -n pos-cloud rollout status deployment/pos-postgres --timeout=90s
kubectl -n pos-cloud rollout status deployment/pos-cloud-engine --timeout=90s
kubectl -n pos-edge rollout status deployment/pos-edge-engine --timeout=90s

echo "=========================================================="
echo " K3s Kubernetes Cluster Status:"
echo "=========================================================="
kubectl get namespaces
echo ""
kubectl get pods -A -l 'app in (pos-cloud-engine, pos-edge-engine, pos-postgres)'
echo ""
kubectl get pvc -n pos-edge
echo ""
kubectl get svc -A -l 'app in (pos-cloud-engine, pos-edge-engine, pos-postgres)'
echo "=========================================================="
echo " Kubernetes Orchestration successfully deployed!"
