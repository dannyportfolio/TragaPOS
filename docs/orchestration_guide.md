# Container & Orchestration Operational Guide

This document describes how to build, run, and orchestrate the **Headless Multi-Industry POS & Commerce Engine** on **WSL Kali Linux** and **Docker**.

---

## 1. Prerequisites in WSL Kali Linux

### 1.1 Docker Installation & Daemon Startup
If using **Docker Desktop**:
- Open Docker Desktop Settings -> **Resources** -> **WSL Integration** -> Enable for `kali-linux`.

If using **Docker CE (native inside Kali)**:
```bash
# Update and install Docker
sudo apt update
sudo apt install -y docker.io docker-compose-v2

# Start Docker service
sudo service docker start

# Allow current user to run docker without sudo
sudo usermod -aG docker $USER
newgrp docker
```

### 1.2 Run Pre-flight Diagnostic
Run the automated pre-flight check to verify Docker daemon, memory, and port availability:
```bash
bash scripts/preflight-check.sh
```

---

## 2. Docker Compose Orchestration (Fast Local Multi-Container)

Docker Compose manages the entire stack (`pos-postgres`, `pos-cloud`, `pos-edge`, and `pos-web`) with automated healthchecks, volume persistence, and isolated networking.

### 2.1 Launch the Full Stack
```bash
# Automated quickstart (checks env, generates keys, builds, and launches)
bash scripts/run-compose.sh

# Or run directly via docker compose:
docker compose up -d --build
```

### 2.2 Inspect Container Status & Logs
```bash
# Check container status and health checks
docker compose ps

# Follow logs across all services
docker compose logs -f

# Follow logs for a specific service
docker compose logs -f pos-cloud
docker compose logs -f pos-edge
docker compose logs -f pos-web
```

### 2.3 Exposed Ports & Endpoints
| Service | Internal Port | Host Port | Purpose | Health Endpoint |
| :--- | :--- | :--- | :--- | :--- |
| **`pos-web`** | `80` | `3000` | Web Cashier & KDS Dashboard | `http://localhost:3000/healthz` |
| **`pos-cloud`** | `8080` | `8080` | Cloud Central Hub (Postgres) | `http://localhost:8080/healthz` |
| **`pos-edge`** | `8081` | `8081` | Edge Local Outlet (SQLite WAL)| `http://localhost:8081/healthz` |
| **`pos-postgres`**| `5432` | `5432` | PostgreSQL Database | `pg_isready -U postgres` |

### 2.4 Teardown
```bash
# Stop containers (preserves database volumes)
docker compose down

# Stop and wipe persistent volumes (fresh reset)
docker compose down -v
```

---

## 3. Kubernetes / K3s Orchestration (Cluster Deployment)

Designed for staging and production testing inside WSL Kali Linux using K3s.

### 3.1 Automated Cluster Deployment
```bash
# Executes automated K3s installation, image builds, import, and manifest rollout
bash scripts/deploy-k3s-kali.sh
```

### 3.2 Manual Manifest Application
```bash
# Ensure K3s kubeconfig is exported
export KUBECONFIG=/etc/rancher/k3s/k3s.yaml

# 1. Deploy Central Cloud Hub (Namespace: pos-cloud)
kubectl apply -f deploy/k8s/cloud-deployment.yaml

# 2. Deploy Local Edge Outlet (Namespace: pos-edge)
kubectl apply -f deploy/k8s/edge-k3s.yaml

# 3. Check Pod and PVC status
kubectl get pods -A
kubectl get pvc -n pos-edge
```

### 3.3 Key Kubernetes Safety Features
1. **Single-Writer SQLite Recreate Strategy:** In `deploy/k8s/edge-k3s.yaml`, the deployment specifies `strategy: type: Recreate` and `replicas: 1`. This guarantees that old edge pods terminate before new pods mount the persistent SQLite volume `/data/edge.db`, preventing lock contention and file corruption.
2. **Distroless Non-Root Security:** All application pods run as non-root UID `65532` (`runAsNonRoot: true`).
3. **Horizontal Pod Autoscaling (HPA):** `pos-cloud-hpa` automatically scales central cloud pods between 2 and 10 replicas based on CPU/Memory load.

---

## 4. End-to-End Verification & Sanity Testing

To test the entire integrated workflow (cryptographic ticket issuing, offline verification, and transactional sync):

```bash
bash scripts/test-e2e-orchestration.sh
```

This verifies:
1. Cloud and Edge healthcheck status (`/healthz`, `/readyz`).
2. Central catalog retrieval.
3. Ed25519 cryptographic ticket generation on Cloud (`POST /v1/tickets/issue`).
4. Offline Ed25519 ticket mathematical verification on Edge without network access (`POST /v1/tickets/redeem`).
5. Outbox transactional checkout on Edge.

---

## 5. Troubleshooting & FAQ

- **Docker daemon error in WSL Kali (`Cannot connect to the Docker daemon`)**:
  - Run `sudo service docker start` or verify Docker Desktop WSL Integration is toggled on.
- **Port conflicts (`address already in use`)**:
  - Change `CLOUD_PORT`, `EDGE_PORT`, or `WEB_PORT` in `.env`.
- **Permission denied on `.db` file**:
  - Ensure the `/data` directory in the Edge container is writable by user `65532` or mounted via a named Docker volume (`edge_sqlite_data`).
