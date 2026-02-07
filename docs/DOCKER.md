# Docker: Local Development & Production

This guide covers running the app with **Docker Compose** (local development) and **Docker** (production image), including how static assets (Tailwind CSS and htmx) are built and where they are stored.

---

## Static assets overview

| Asset | Source | Output | Purpose |
|-------|--------|--------|---------|
| **Tailwind CSS** | `web/css/input.css` | `web/static/css/styles.css` | Production styles (minified) |
| **htmx** | `node_modules/htmx.org/dist/htmx.min.js` | `web/static/js/htmx.min.js` | Client-side interactivity |

The Go app serves everything under `web/` (templates and `web/static/`). So after building, the app uses:

- `web/static/css/styles.css`
- `web/static/js/htmx.min.js`

---

## 1. Local development (Docker Compose)

Uses `docker/Dockerfile.dev`: Go + Air for hot reload. **CSS and htmx are not built inside the container**; they come from your host (or you build them on the host and the container sees them via the volume mount).

### Prerequisites

- Docker and Docker Compose
- On your **host**: Node.js 18+ and npm (for building Tailwind and copying htmx)

### Step 1: Go dependencies (vendor)

The Dockerfile copies `vendor/`. Create it on the host:

```bash
go mod tidy && go mod vendor
# or
make deps
```

### Step 2: Environment

```bash
cp .env.example .env
# Edit .env (Cloudinary, JWT_SECRET, WHATSAPP_NUMBER, etc.)
```

### Step 3: Build CSS and htmx (static assets)

Run on your **host** so that `web/static/` is filled before starting the app:

```bash
npm install
make assets
```

This:

1. Builds Tailwind: `web/css/input.css` → `web/static/css/styles.css` (minified).
2. Copies htmx: `node_modules/htmx.org/dist/htmx.min.js` → `web/static/js/htmx.min.js`.

Optional: for live CSS updates while developing, in a **separate terminal**:

```bash
make css-watch
```

Then `web/static/css/styles.css` is updated on change; the app container sees it via the mounted volume.

### Step 4: Start services

```bash
docker-compose up -d
# or
make docker-up
```

This:

- Builds the app image from `docker/Dockerfile.dev`.
- Mounts the project directory into the container (`.:/app`), so `web/static/` (including the CSS and JS you built) is available inside the app.
- Runs `air -c .air.toml` (Go hot reload).
- Starts Postgres and (optionally) Adminer.
- **Database migrations** run via the **db-migration** service (dbmate) before the api starts (see [Database migrations](#database-migrations)).

### Step 5: Use the app

- App: http://localhost:3000  
- Admin: http://localhost:3000/admin  
- Adminer (DB): http://localhost:8888  

### Summary: local flow

```text
Host: npm install → make assets [→ make css-watch]
Host: go mod vendor
Host: make server/start (or docker-compose up -d)
→ db-migration runs (dbmate up), then api starts (make build && ./bin/aslam-flower)
```

### Useful commands

| Command | Description |
|--------|-------------|
| `make docker-logs` | Follow app logs |
| `make docker-ps` | List running containers |
| `make docker-down` | Stop and remove containers |

### Database migrations

Migrations use [dbmate](https://github.com/amacneil/dbmate). SQL files live in **`db/migrations/`** with the format `YYYYMMDDHHMMSS_name.sql`; each file has `-- migrate:up` and `-- migrate:down` sections.

- **With Docker:** The **db-migration** service runs `dbmate up` before the api starts (api depends on `db-migration` with `condition: service_completed_successfully`). No manual step needed for normal `make server/start`.
- **Without Docker:** Install dbmate (`brew install dbmate` or [see docs](https://github.com/amacneil/dbmate#installation)), set `DATABASE_URL` in `.env`, then run `make migrate-up` or `dbmate up`.
- **New migration:** `make migrate-new name=description` or `dbmate new description`.
- **Rollback:** `make migrate-rollback` or `dbmate rollback`.

---

## 2. Production (Dockerfile)

Uses `docker/Dockerfile`: multi-stage build. **CSS and htmx are built inside the image**; you do not need to run `make assets` on the host before building the image.

### Build stage (inside Docker)

1. **Go deps**: `go.mod` and `go.sum` are copied; then `go mod tidy` and `go mod vendor` run inside the image to create `vendor/` (no host `vendor/` needed). Build uses `-mod=vendor`.
2. **Static assets** (in the same stage):
   - `npm install`
   - `mkdir -p web/static/css web/static/js`
   - Tailwind: `npx tailwindcss -i ./web/css/input.css -o ./web/static/css/styles.css --minify`
   - htmx: `cp node_modules/htmx.org/dist/htmx.min.js web/static/js/htmx.min.js`
3. **Go binary**: `go build -mod=vendor -o server ./cmd/server`

### Final stage

- Copies `server`, the whole `web/` tree (templates + `web/static/` with the built CSS and JS), and the **root `.env`** into the image. Use the root `.env` for local/default config; in production you can override or set variables via the Railway dashboard (or any orchestrator env).
- Runs as non-root, exposes 3000, includes a health check.

### Prerequisites on host

- Docker only. You do **not** need Go or Node/npm on the host; the image runs `go mod tidy && go mod vendor` and builds CSS/htmx inside the builder stage.

### Step 1: Build the image

```bash
docker build -f docker/Dockerfile -t aslam-flower:latest .
# or
make docker-build
```

The image includes the root `.env` when present at build time (otherwise an empty `.env`). For production, set or override env in the Railway dashboard; the app still loads `.env` and env vars take precedence.

### Step 2: Run the container

Example (replace with your real env and DB):

```bash
docker run -d --name aslam-flower \
  -p 3000:3000 \
  -e DATABASE_URL="postgresql://user:pass@host:5432/db?sslmode=require" \
  -e CLOUDINARY_CLOUD_NAME="..." \
  -e CLOUDINARY_API_KEY="..." \
  -e CLOUDINARY_API_SECRET="..." \
  -e JWT_SECRET="..." \
  -e ENV=production \
  -e WHATSAPP_NUMBER="628xxxxxxxxxx" \
  aslam-flower:latest
```

Or use Docker Compose / your orchestrator with the same image and env.

### Summary: production flow

```text
Host: docker build -f docker/Dockerfile -t aslam-flower:latest .
  → Inside image: go mod tidy && go mod vendor → npm install → build CSS → copy htmx → go build
  → Image contains binary + web/ (templates + web/static with CSS & JS)
Run: docker run ... aslam-flower:latest
```

---

## Quick reference

| Context | Build CSS & htmx | Where | Run Go app |
|--------|-------------------|--------|------------|
| **Local (Docker Compose)** | On **host**: `npm install && make assets` | `web/static/` (mount into container) | `docker-compose up -d` (Air inside container) |
| **Production (Dockerfile)** | **Inside** builder stage of Dockerfile | `web/static/` copied into image | `docker run ... aslam-flower:latest` |

---

## Makefile targets

| Target | Description |
|--------|-------------|
| `make assets` | Build Tailwind CSS + copy htmx to `web/static/` |
| `make css` | Build only Tailwind to `web/static/css/styles.css` |
| `make css-watch` | Watch and rebuild Tailwind (dev) |
| `make docker-up` | Start Docker Compose (local dev) |
| `make docker-down` | Stop Docker Compose |
| `make docker-build` | Build production image (`docker/Dockerfile`) |
