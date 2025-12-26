# How to Run the Server

This guide explains how to run and deploy the **DevOps-Valgfag (WhoKnows)** Go server in **development (local)** and **production (VM)** using Docker and GitHub Container Registry (GHCR).

---

## Development (local setup)

### 1. Prerequisites

- Install Docker (includes the Docker Compose plugin): https://docs.docker.com/get-docker/

---

### 2. Clone the repository

```bash
git clone https://github.com/GitDenGas123456/DevOps-Valgfag.git
cd DevOps-Valgfag
```

---

### 3. Create a `.env` file (required for Docker Compose)

Docker Compose **requires** a `.env` file because `docker-compose.yml` uses `:?`-guarded variables (Compose will exit immediately if they’re missing).

Start by copying the example:

```bash
cp .env.example .env
```

Fill in **at minimum** (values below are examples):

```env
# Which image tag compose should run (or build locally)
APP_IMAGE_TAG=latest

# Postgres (compose db service)
POSTGRES_USER=devops
POSTGRES_PASSWORD=devops
POSTGRES_DB=whoknows
POSTGRES_PORT=5432

# App
SESSION_KEY=replace-with-32+byte-random-secret
DMI_API_KEY=replace-with-your-dmi-api-key

# Grafana (monitoring)
GF_SECURITY_ADMIN_USER=admin
GF_SECURITY_ADMIN_PASSWORD=admin

# Local grafana URL settings (compose serves grafana under a subpath)
GF_SERVER_DOMAIN=localhost
```

Notes:

- `docker-compose.yml` runs the app with `APP_ENV=prod` by default (so “prod” behaviour/logging applies locally too).
- Grafana is served from a **subpath**: `http://localhost:3000/grafana/` (that’s why `GF_SERVER_DOMAIN=localhost` helps locally).

---

### 4. (Optional) Build the image locally

```bash
sudo docker build -t devops-valgfag:latest .
```

⚠️ This image is only used by Docker Compose if you set:

```env
APP_IMAGE_TAG=latest
```

Otherwise, Compose will pull `ghcr.io/gitdengas123456/devops-valgfag:${APP_IMAGE_TAG}`.

---

### 5. Run the server locally

#### Option A (recommended): Run with Docker Compose

Requires the `.env` from step 3.

```bash
sudo docker compose up --build
```

Access:

- App: http://localhost:8080
- Grafana: http://localhost:3000/grafana/

Stop:

```bash
sudo docker compose down
```

---

#### Option B: Run only the app container (requires an existing Postgres DB)

If you run the app container without Compose, it must be able to reach Postgres **from inside the container**.

```bash
sudo docker run --rm -p 8080:8080   -e APP_ENV=dev   -e PORT=8080   -e SESSION_KEY="dev-session-key-0123456789-abcdefghijklmnopqrstuvwxyz"   -e DMI_API_KEY="replace-with-your-dmi-api-key"   -e DATABASE_URL="postgres://devops:devops@host.docker.internal:5432/whoknows?sslmode=disable"   devops-valgfag:latest
```

Linux note: `host.docker.internal` may not exist. Use one of:

- `--network host` (quick local workaround), or
- run Postgres in Compose and use the Compose network alias (e.g. `postgres_db`) from the app container.

---

## Production (VM or cloud deployment)

This setup uses the image published to **GitHub Container Registry (GHCR)**.

---

### 1. Log in to GHCR

You need a GitHub token with `read:packages` permission.

```bash
echo YOUR_GITHUB_TOKEN | sudo docker login ghcr.io -u YOUR_GITHUB_USERNAME --password-stdin
```

---

### 2. Pull the image

```bash
sudo docker pull ghcr.io/gitdengas123456/devops-valgfag:latest
```

---

### 3. Run the container (standalone)

```bash
sudo docker run -d --name devops-valgfag   -p 8080:8080   -e APP_ENV=prod   -e PORT=8080   -e SESSION_KEY="replace-with-a-strong-random-32+byte-secret"   -e DMI_API_KEY="replace-with-your-dmi-api-key"   -e DATABASE_URL="postgres://USER:PASSWORD@DB_HOST:5432/DB_NAME?sslmode=disable"   ghcr.io/gitdengas123456/devops-valgfag:latest
```

Access the app at:

- http://<your-vm-ip>:8080

---

### 4. View logs

```bash
sudo docker logs -f devops-valgfag
```

---

### 5. Stop / restart

```bash
sudo docker stop devops-valgfag
sudo docker start devops-valgfag
```

---

### 6. Cleanup

```bash
sudo docker rm -f devops-valgfag
sudo docker image prune -f
```

---

## Server Overview

| Component | Description |
| --- | --- |
| Language | Go |
| HTTP stack | `net/http` + `gorilla/mux` |
| Port | `8080` (default; configurable via `PORT`) |
| Database | PostgreSQL (via `DATABASE_URL` / `DB_HOST` + vars) |
| Sessions | `gorilla/sessions` (cookie store) |
| OpenAPI/Swagger | `swaggo/swag` + `http-swagger` |
| Container registry | `ghcr.io/gitdengas123456/devops-valgfag` |
| Purpose | Lightweight web backend for the DevOps-Valgfag project |

---

## Troubleshooting

| Issue | Cause | Fix |
| --- | --- | --- |
| `docker compose up` exits immediately | Missing `.env` values | Fill required env vars and retry |
| Failed to connect to PostgreSQL | Wrong DSN / DB down | Verify `DATABASE_URL` or `POSTGRES_*` settings and DB availability |
| Weather endpoints return 503 | Missing `DMI_API_KEY` | Provide a valid DMI API key (`.env` for compose, `-e DMI_API_KEY=...` for docker run) |
| `SESSION_KEY is required` | Missing env var | Set `SESSION_KEY` (32+ bytes recommended) |
| GHCR permission denied | Token scope issue | Ensure token has `read:packages` |

---

## Notes

- Prefer `.env` + `docker compose up -d` in both dev and prod for parity with CI/CD.
- Make sure your VM firewall allows inbound traffic on port **8080**.
- Never commit real secrets to git.
