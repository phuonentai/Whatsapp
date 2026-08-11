# 🛠️ Local Development Setup

Complete guide to running the Production SaaS Starter Kit locally.

---

## 📋 Prerequisites

Install these tools before starting:

| Tool | Version | Purpose |
|------|---------|---------|
| **Docker Desktop** | Latest | Runs PostgreSQL + Redis containers |
| **Go** | 1.25+ | Backend server |
| **Node.js** | 20+ | Frontend build |
| **pnpm** | 9+ | Frontend package manager |

> **Note:** You do NOT need to install PostgreSQL or Redis directly — Docker handles them.

---

### 🐳 Install Docker Desktop

Docker runs PostgreSQL and Redis in containers so you don't need to install them directly.

**macOS:**
```bash
# Option 1: Homebrew
brew install --cask docker

# Option 2: Direct download
# https://www.docker.com/products/docker-desktop/
```

**Windows:**
1. Download from https://www.docker.com/products/docker-desktop/
2. Run installer
3. Enable WSL 2 when prompted

**Linux (Ubuntu/Debian):**
```bash
# Add Docker's official GPG key
sudo apt-get update
sudo apt-get install ca-certificates curl
sudo install -m 0755 -d /etc/apt/keyrings
sudo curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
sudo chmod a+r /etc/apt/keyrings/docker.asc

# Add the repository
echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu \
  $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | \
  sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

# Install Docker
sudo apt-get update
sudo apt-get install docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

# Add your user to docker group (logout/login required)
sudo usermod -aG docker $USER
```

After installing, **open Docker Desktop** and wait for it to start before proceeding.

---

### 🐹 Install Go

The backend requires Go 1.25 or higher.

**macOS:**
```bash
# Option 1: Homebrew (recommended)
brew install go

# Option 2: Direct download
# https://go.dev/dl/
```

**Windows:**
1. Download from https://go.dev/dl/
2. Run the MSI installer
3. Restart terminal after install

**Linux:**
```bash
# Download latest (check https://go.dev/dl/ for current version)
wget https://go.dev/dl/go1.25.0.linux-amd64.tar.gz

# Remove old version and extract new
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.25.0.linux-amd64.tar.gz

# Add to PATH (add to ~/.bashrc or ~/.zshrc)
export PATH=$PATH:/usr/local/go/bin
```

---

### 📦 Install Node.js

The frontend requires Node.js 20 or higher. We recommend using **nvm** (Node Version Manager) to manage Node versions.

**macOS/Linux — Install nvm:**
```bash
# Install nvm
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.1/install.sh | bash

# Restart terminal, then install Node 20
nvm install 20
nvm use 20
nvm alias default 20
```

**Windows — Install nvm-windows:**
1. Download from https://github.com/coreybutler/nvm-windows/releases
2. Run the installer
3. Open new terminal:
```bash
nvm install 20
nvm use 20
```

**Alternative — Direct install (without nvm):**
```bash
# macOS
brew install node@20

# Or download from https://nodejs.org/
```

---

### 📦 Install pnpm

pnpm is our package manager for the frontend (faster than npm, saves disk space).

```bash
# After Node is installed
npm install -g pnpm
```

Or use Corepack (built into Node 16.13+):
```bash
corepack enable
corepack prepare pnpm@latest --activate
```

---

### ✅ Verify Installation

Run these commands to confirm everything is installed correctly:

```bash
docker --version    # Docker version 24+
go version          # go1.25+
node --version      # v20+
pnpm --version      # 9+
```

All four should return version numbers. If any command fails, revisit the install steps above.

---

## 📁 Project Structure

```
production-saas-starter/
├── go-b2b-starter/       # Go backend
├── next_b2b_starter/     # Next.js frontend
├── deps/                 # Docker Compose files
└── setup.sh              # Automated setup script
```

---

## 🚀 First-Time Setup

### 1. Clone the Repository

```bash
git clone <repo-url>
cd production-saas-starter
```

### 2. Start Backend Services

```bash
cd go-b2b-starter

# Start PostgreSQL + Redis containers
make run-deps

# Wait for containers to be healthy, then run migrations
make migrateup
```

### 3. Configure Frontend Environment

```bash
cd next_b2b_starter

# Copy environment template
cp .env.example .env.local

# Edit .env.local and fill in:
# - STYTCH_PROJECT_ID
# - STYTCH_SECRET
# - NEXT_PUBLIC_STYTCH_PUBLIC_TOKEN
# - POLAR_ACCESS_TOKEN (if using billing)
```

### 4. Install Frontend Dependencies

```bash
pnpm install
```

### 5. Start Development Servers

**Terminal 1 — Backend:**
```bash
cd go-b2b-starter
make dev
```

**Terminal 2 — Frontend:**
```bash
cd next_b2b_starter
pnpm dev
```

### 6. Access the App

| Service | URL |
|---------|-----|
| Frontend | http://localhost:3000 |
| Backend API | http://localhost:8080 |
| API Docs (Swagger) | http://localhost:8080/swagger/index.html |

---

## 📅 Daily Workflow

### Starting Your Day

```bash
# Terminal 1: Start database containers (if not running)
cd go-b2b-starter
make run-deps

# Terminal 2: Start backend with hot reload
cd go-b2b-starter
make dev

# Terminal 3: Start frontend
cd next_b2b_starter
pnpm dev
```

### Ending Your Day

```bash
# Stop the Go server: Ctrl+C in Terminal 2
# Stop the Next.js server: Ctrl+C in Terminal 3

# Optional: Stop database containers (data persists)
cd go-b2b-starter
make stop-deps
```

---

## 📋 Commands Cheat Sheet

### Backend (Go)

Run from `go-b2b-starter/`:

| Command | Description |
|---------|-------------|
| `make run-deps` | Start PostgreSQL + Redis containers |
| `make stop-deps` | Stop and remove containers |
| `make dev` | Start server with hot reload (Air) |
| `make server` | Start server without hot reload |
| `make migrateup` | Apply database migrations |
| `make migratedown` | Rollback migrations |
| `make create-migration MIGRATION_NAME=add_users` | Create new migration |
| `make sqlc` | Generate Go code from SQL queries |
| `make test` | Run tests |
| `make swagger` | Regenerate Swagger docs |
| `make build` | Build production binary |

### Frontend (Next.js)

Run from `next_b2b_starter/`:

| Command | Description |
|---------|-------------|
| `pnpm dev` | Start development server |
| `pnpm build` | Build for production |
| `pnpm start` | Start production server |
| `pnpm lint` | Run ESLint |

---

## 🔌 Service Ports

| Service | Port | Notes |
|---------|------|-------|
| Next.js Frontend | 3000 | Turbopack hot reload |
| Go Backend | 8080 | Air hot reload |
| PostgreSQL | 5432 | pgvector enabled |
| Redis | 6379 | For caching/sessions |

---

## 🗄️ Database Access

### Connect via psql

```bash
psql -h localhost -p 5432 -U user -d mydatabase
# Password: password
```

### Connect via GUI (TablePlus, DBeaver, etc.)

| Field | Value |
|-------|-------|
| Host | localhost |
| Port | 5432 |
| User | user |
| Password | password |
| Database | mydatabase |

### Database Credentials

Defined in `go-b2b-starter/deps/docker-compose.yml`:

```yaml
POSTGRES_DB: mydatabase
POSTGRES_USER: user
POSTGRES_PASSWORD: password
```

---

## 🔐 Environment Variables

### Backend

The backend uses Docker environment variables defined in `deps/docker-compose.yml`. No `.env` file needed for local dev.

### Frontend

Required variables in `.env.local`:

```bash
# App URLs
APP_BASE_URL=http://localhost:3000
NEXT_PUBLIC_APP_BASE_URL=http://localhost:3000

# Stytch B2B Auth (required)
STYTCH_PROJECT_ID=project-test-xxx
STYTCH_SECRET=secret-test-xxx
STYTCH_PROJECT_ENV=test
NEXT_PUBLIC_STYTCH_PROJECT_ENV=test
NEXT_PUBLIC_STYTCH_PUBLIC_TOKEN=public-token-test-xxx

# Polar Billing (optional for dev)
POLAR_ACCESS_TOKEN=
POLAR_WEBHOOK_SECRET=
```

Get Stytch credentials from: https://stytch.com/dashboard

---

## 🧪 E2E Tests

The Playwright suite (`next_b2b_starter/e2e/`, 61 tests across 13 specs) runs fully offline using mock auth. No Stytch, Meta, or billing credentials are required.

### Canonical e2e environment

| Component | Value |
|-----------|-------|
| Go backend | `:8080` (`SERVER_ADDRESS=:8080`) |
| Next.js frontend | `:3001` (`pnpm dev -p 3001`) |
| Mock Siigo provider | `:8090` (`cmd/mock-siigo`) — backend points at it via `SIIGO_BASE_URL`/`SIIGO_TOKEN_URL`/`SIIGO_WEBHOOK_SECRET`; no real Siigo traffic |
| Test database | `saas_db_test` |
| Auth | `AUTH_MOCK_ENABLED=true` (`X-Test-Org-ID: <slug>:<email>`) |
| Mock orgs | `test-org-free`, `test-org-pro`, `test-org-enterprise`, `test-org-rbac`, `test-org-siigo` (Siigo onboarding suite: `admin-siigo@test.com` / `member-siigo@test.com`) |

`next_b2b_starter/.env.local` already carries the e2e values (`API_REWRITE_TARGET=http://localhost:8080`, `APP_BASE_URL=http://localhost:3001`). E2E helper defaults in `e2e/helpers/` point at `:8080`. The mock Siigo provider (`go-b2b-starter/cmd/mock-siigo`) implements the spike-verified adapter surface — token grant, absent `/v1/company`, paginated customers, invoice creation with Idempotency-Key dedupe and consecutive numbering — so the Siigo onboarding suite runs fully offline.

### One-command run

```bash
# From go-b2b-starter/ — boots infra, migrates, seeds, starts both servers, runs suite, cleans up
make test-e2e
```

Prerequisites: Docker (root `docker-compose.yml` postgres + redis), Go 1.25, pnpm, and a Playwright chromium install:

```bash
pnpm --dir next_b2b_starter exec playwright install chromium
```

`make test-e2e` is the local equivalent of the CI `e2e` job in `.github/workflows/ci.yml`, so the two cannot drift.

### Running the pieces manually

```bash
# 1. Infra + test DB (postgres/redis from docker-compose.yml)
docker compose -f docker-compose.yml up -d postgres redis
docker compose -f docker-compose.yml exec -T postgres createdb -U postgres saas_db_test 2>/dev/null || true

# 2. Migrate + seed
POSTGRES_DB=saas_db_test docker compose -f docker-compose.yml run --rm migrate
POSTGRES_HOST=localhost POSTGRES_DB=saas_db_test AUTH_MOCK_ENABLED=true go run ./cmd/seed-e2e

# 3. Backend (mock auth, :8080)
AUTH_MOCK_ENABLED=true SERVER_ADDRESS=:8080 go run ./cmd/api/main.go &

# 4. Frontend (:3001, reads .env.local)
pnpm --dir next_b2b_starter dev -p 3001 &

# 5. Suite
pnpm --dir next_b2b_starter test:e2e
```

### CI

Push/pull requests run `.github/workflows/ci.yml` (backend unit tests, frontend lint+build, offline e2e). The workflow also fails if `AUTH_MOCK_ENABLED=true` ever appears in production configuration.

---

## 🔧 Troubleshooting

### Docker containers won't start

```bash
# Check if Docker is running
docker info

# Check container status
docker ps -a

# View container logs
docker compose -f deps/docker-compose.yml logs postgres
docker compose -f deps/docker-compose.yml logs redis

# Nuclear option: remove everything and start fresh
make stop-deps
docker volume rm deps_postgres_data deps_redis_data
make run-deps
```

### Port already in use

```bash
# Find what's using the port
lsof -i :5432  # PostgreSQL
lsof -i :8080  # Backend
lsof -i :3000  # Frontend

# Kill the process
kill -9 <PID>
```

### Migration errors

```bash
# Check migration status
docker compose -f deps/docker-compose.yml run --rm cli migrate -path ./internal/db/postgres/sqlc/migrations -database "postgresql://user:password@postgres:5432/mydatabase?sslmode=disable" version

# Force to specific version if stuck
docker compose -f deps/docker-compose.yml run --rm cli migrate -path ./internal/db/postgres/sqlc/migrations -database "postgresql://user:password@postgres:5432/mydatabase?sslmode=disable" force <VERSION>
```

### SQLC generation fails

```bash
# Make sure containers are running
make run-deps

# Check sqlc.yaml configuration
cat internal/db/postgres/sqlc/sqlc.yaml

# Run sqlc with verbose output
docker compose -f deps/docker-compose.yml run --rm -w /workspace/internal/db/postgres/sqlc cli sqlc generate
```

### Hot reload not working (Backend)

Air watches for file changes. If it's not working:

```bash
# Check Air is installed in container
docker compose -f deps/docker-compose.yml run --rm cli which air

# Restart with fresh build
make stop-deps
make run-deps
make dev
```

### Hot reload not working (Frontend)

```bash
# Clear Next.js cache
rm -rf .next

# Restart
pnpm dev
```

### Can't connect to backend from frontend

1. Check backend is running: http://localhost:8080/health
2. Check CORS settings in backend
3. Verify API URL in frontend matches backend port

---

## 💡 Tips

### VS Code Extensions

Recommended for this project:

- **Go** — Go language support
- **ESLint** — JavaScript/TypeScript linting
- **Tailwind CSS IntelliSense** — Tailwind autocomplete
- **Docker** — Docker file support
- **PostgreSQL** — SQL syntax highlighting

### Multiple Terminals

Use a terminal multiplexer or VS Code's integrated terminals:

```
┌─────────────────────────────────────────┐
│ Terminal 1: make run-deps (background)  │
├─────────────────────────────────────────┤
│ Terminal 2: make dev (backend)          │
├─────────────────────────────────────────┤
│ Terminal 3: pnpm dev (frontend)         │
└─────────────────────────────────────────┘
```

### Fast Iteration Cycle

1. Make code changes
2. Save file (hot reload triggers)
3. Test in browser
4. Repeat

No manual restart needed thanks to Air (Go) and Turbopack (Next.js).
