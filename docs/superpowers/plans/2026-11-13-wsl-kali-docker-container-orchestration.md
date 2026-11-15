# WSL Kali Linux & Docker Container Orchestration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Establish, containerize, and orchestrate the full Headless POS & Commerce Engine (`pos-cloud`, `pos-edge`, `pos-postgres`, and `pos-web`) on the user's WSL Kali Linux and Docker environment, supporting both Docker Compose multi-container runtime and Kubernetes (K3s/Minikube) orchestration.

**Architecture:**
- **Central Cloud Hub (`pos-cloud`):** Go binary compiled via distroless zero-CGO container (`deploy/Dockerfile`), connected to PostgreSQL 16, running Ed25519 cryptographic ticket signer and sync ingestor on port `8080`.
- **Local Edge Hub (`pos-edge`):** Go binary (`deploy/Dockerfile`) in `APP_MODE=EDGE`, using persistent SQLite WAL on `/data/edge.db`, 100% offline Ed25519 ticket verifier, and background sync worker on port `8081`.
- **Database Engine (`pos-postgres`):** PostgreSQL 16 Alpine container with healthchecks and persistent volume.
- **Frontend POS & Dashboard (`pos-web`):** Multi-stage production container (`web/Dockerfile`) with Node 22 builder and Nginx Alpine server routing requests to Cloud, Edge, and WebSocket Hub on port `3000`.
- **Orchestration Tiers:**
  1. **Tier 1 (Docker Compose):** Instant local stack via `compose.yaml` with health-gated startup ordering (`depends_on.condition: service_healthy`), isolated bridge network, and persistent named volumes.
  2. **Tier 2 (Kubernetes / K3s in WSL Kali):** Full orchestration using `deploy/k8s/cloud-deployment.yaml` and `deploy/k8s/edge-k3s.yaml`, testing Pod lifecycle, PVC claims, Recreate strategy for SQLite single-writer safety, and HPA.

**Tech Stack:** WSL2 (Kali Linux), Docker Engine / Docker Compose v2, K3s / Kubernetes v1.28+, Go 1.24, Distroless Debian 12, PostgreSQL 16, SQLite (WAL mode), Nginx Alpine.

---

## Global Constraints & Security Boundaries
- **Zero-CGO & Distroless:** Backend builds must strictly use `CGO_ENABLED=0` and target `gcr.io/distroless/static-debian12:nonroot` with UID `65532`.
- **Single-Writer SQLite Safety:** Edge deployments must enforce `strategy: type: Recreate` with `replicas: 1` and dedicated volume mounts to prevent SQLite database corruption.
- **Secret Isolation:** Cryptographic Ed25519 private keys must ONLY be provided to `pos-cloud`. `pos-edge` must receive ONLY the public key. Never hardcode keys in git; load via `.env` or Kubernetes Secrets.
- **Sandboxed Execution Awareness:** WSL and Docker commands run on the host or inside WSL Kali Linux. Terminal commands interacting with Docker/WSL require user sandbox bypass or direct terminal execution.

---

## Proposed Changes & File Structure

```text
Ticket-Prod/
├── compose.yaml                             # Complete multi-service Docker Compose definition
├── .env.example                             # Environment variable template with key generation guide
├── web/
│   ├── Dockerfile                           # Multi-stage production build (Node 22 + Nginx)
│   └── nginx.conf                           # Nginx reverse proxy configuration for web + APIs + WS
├── scripts/
│   ├── generate-keys.sh                     # Ed25519 keypair generation helper (OpenSSL / Go)
│   ├── run-compose.sh                       # Quickstart orchestration script for Docker Compose
│   └── deploy-k3s-kali.sh                   # Deployment script for K3s inside WSL Kali Linux
└── deploy/
    └── k8s/
        ├── cloud-deployment.yaml            # Cloud Kubernetes manifest (Namespace, ConfigMap, Deployment, Svc, HPA)
        └── edge-k3s.yaml                    # Edge K3s manifest (PVC, Recreate Deployment, Svc)
```

---

## Tasks

### Task 1: Environment Verification & WSL Kali Docker Pre-flight

**Files:**
- Create: `scripts/preflight-check.sh`

- [x] **Step 1: Check WSL Kali Linux and Docker availability**
  Verify Docker daemon is active in WSL Kali Linux (`docker info` or `systemctl status docker` / `service docker status`).
- [x] **Step 2: Ensure Docker user group permissions**
  Validate current user can run docker without root: `sudo usermod -aG docker $USER`.
- [x] **Step 3: Create executable pre-flight script**
  Create `scripts/preflight-check.sh` verifying Docker engine version, Docker Compose plugin, and memory allocation in WSL.
- [x] **Step 4: Execute preflight check**
  Run `bash scripts/preflight-check.sh` and ensure all prerequisites pass.

---

### Task 2: Web Client Containerization (`web/Dockerfile` & `nginx.conf`)

**Files:**
- Create: `web/Dockerfile`
- Create: `web/nginx.conf`
- Create: `web/.dockerignore`

- [x] **Step 1: Create `web/.dockerignore`**
  Exclude `node_modules`, `dist`, `.git`, and build logs from Docker context.
- [x] **Step 2: Create `web/nginx.conf`**
  Configure Nginx to:
  - Serve SPA index.html on fallback (`try_files $uri $uri/ /index.html;`)
  - Proxy `/api/cloud/` to `http://pos-cloud:8080/`
  - Proxy `/api/edge/` to `http://pos-edge:8081/`
  - Proxy `/ws` with WebSocket upgrade headers (`Upgrade $http_upgrade`, `Connection "upgrade"`)
- [x] **Step 3: Create multi-stage `web/Dockerfile`**
  - Stage 1: `node:22-alpine` with `corepack enable` and `pnpm build`
  - Stage 2: `nginx:alpine-slim` unprivileged, copying dist to `/usr/share/nginx/html` and custom `nginx.conf`
- [x] **Step 4: Test build web Docker image locally**
  Run `docker build -t pos-web:latest -f web/Dockerfile web/` to verify clean build.

---

### Task 3: Keypair Generation & Environment Scaffolding

**Files:**
- Create: `scripts/generate-keys.sh`
- Create: `.env.example`
- Create: `.env` (gitignored)

- [x] **Step 1: Create `scripts/generate-keys.sh`**
  Use a short Go one-liner or `openssl` to generate a valid Ed25519 keypair and output base64-encoded strings:
  - `ED25519_PRIVATE_KEY` (64-byte base64)
  - `ED25519_PUBLIC_KEY` (32-byte base64)
- [x] **Step 2: Create `.env.example`**
  Document all environment variables:
  - `POSTGRES_USER=postgres`, `POSTGRES_PASSWORD=securepassword`, `POSTGRES_DB=pos_central`
  - `EDGE_AUTH_TOKEN=pos-edge-shared-secret-token`
  - `TENANT_ID=tenant-park-01`, `OUTLET_ID=outlet-gate-north`
- [x] **Step 3: Generate `.env` from template**
  Run `bash scripts/generate-keys.sh` to populate active `.env` with freshly generated cryptographic keys.

---

### Task 4: Complete Docker Compose Orchestration Setup

**Files:**
- Create: `compose.yaml`
- Modify: `.gitignore` (ensure SQLite volumes and `.env` are ignored)

- [x] **Step 1: Author `compose.yaml`**
  Define services:
  1. `pos-postgres`: image `postgres:16-alpine`, port 5432, persistent volume `postgres_data`, health check `pg_isready -U postgres`.
  2. `pos-cloud`: build from `deploy/Dockerfile`, depends_on `pos-postgres: condition: service_healthy`, port 8080, health check `/healthz`.
  3. `pos-edge`: build from `deploy/Dockerfile`, persistent volume `edge_sqlite_data` mounted to `/data`, `APP_MODE=EDGE`, `CLOUD_SYNC_URL=http://pos-cloud:8080`, port 8081, health check `/healthz`.
  4. `pos-web`: build from `web/Dockerfile`, port 3000, depends_on `pos-cloud` and `pos-edge`.
- [x] **Step 2: Validate compose configuration syntax**
  Run `docker compose config` to verify zero YAML or variable interpolation errors.
- [x] **Step 3: Launch containers via Docker Compose**
  Run `docker compose up -d --build`.
- [x] **Step 4: Inspect container runtime health**
  Run `docker compose ps` to ensure all 4 containers are in `healthy` / `running` status.
- [x] **Step 5: Verify health endpoints**
  - `curl -f http://localhost:8080/healthz` (Cloud Hub)
  - `curl -f http://localhost:8081/healthz` (Edge Hub)
  - `curl -f http://localhost:3000/` (Web UI)

---

### Task 5: Kubernetes / K3s Orchestration in WSL Kali Linux

**Files:**
- Create: `scripts/deploy-k3s-kali.sh`
- Update: `deploy/k8s/cloud-deployment.yaml` (inject populated secrets template)
- Update: `deploy/k8s/edge-k3s.yaml` (inject populated secrets template)

- [x] **Step 1: Create K3s WSL launch & cluster init script `scripts/deploy-k3s-kali.sh`**
  Automate K3s installation or startup inside WSL Kali:
  `curl -sfL https://get.k3s.io | sh -s - --write-kubeconfig-mode 644`
  Export `KUBECONFIG=/etc/rancher/k3s/k3s.yaml`.
- [x] **Step 2: Import local Docker images into K3s containerd**
  `docker save pos-engine:latest | sudo k3s ctr images import -`
  `docker save pos-web:latest | sudo k3s ctr images import -`
- [x] **Step 3: Create Postgres manifest for K8s Cloud Hub**
  Ensure PostgreSQL deployment and service exist in `pos-cloud` namespace.
- [x] **Step 4: Apply Cloud and Edge K8s manifests**
  - `kubectl apply -f deploy/k8s/cloud-deployment.yaml`
  - `kubectl apply -f deploy/k8s/edge-k3s.yaml`
- [x] **Step 5: Verify Pods, Services, and Volume Mounts**
  - Run `kubectl get pods -A`
  - Verify `pos-edge-sqlite-pvc` bound status and `Recreate` deployment strategy.
  - Verify HorizontalPodAutoscaler on `pos-cloud-engine`.

---

### Task 6: End-to-End Functional Sanity & Sync Verification

**Files:**
- Create: `scripts/test-e2e-orchestration.sh`
- Create: `docs/orchestration_guide.md`

- [x] **Step 1: Test Cloud Ticket Generation (Ed25519 Signer)**
  Post an order checkout to Cloud on `http://localhost:8080/v1/checkout` and verify cryptographically signed QR ticket payload is returned.
- [x] **Step 2: Test Edge Offline Ticket Verification (Ed25519 Verifier)**
  Send the signed QR payload to Edge on `http://localhost:8081/v1/tickets/verify` and verify signature passes offline against local SQLite.
- [x] **Step 3: Test Background Sync Ingestion**
  Trigger checkout on Edge (`:8081`), inspect Outbox table, wait for background sync worker tick (5s), and verify record arrives at Cloud (`:8080/v1/sync/push`).
- [x] **Step 4: Document Orchestration Operations Guide**
  Document complete operational commands (start, stop, logs, teardown) in `docs/orchestration_guide.md`.

  Document complete operational commands (start, stop, logs, teardown) in `docs/orchestration_guide.md`.
