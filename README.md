# omniroute-cli

Docker Compose stack for running **OmniRoute** + **Open WebUI** locally.

OmniRoute is an AI model router that aggregates 100+ LLM providers behind a single OpenAI-compatible `/v1` API. Open WebUI provides a browser-based chat interface wired directly to that endpoint.

---

## Architecture

```
Browser / any OpenAI-compatible client
        │
        ▼
  Open WebUI  :20000          ← chat UI
        │ (OpenAI API)
        ▼
   OmniRoute  :20128          ← model router / proxy
        │
        ├── Anthropic (Claude)
        ├── OpenAI (GPT)
        ├── Google (Gemini)
        ├── AWS Bedrock
        ├── Azure AI
        ├── Mistral
        ├── Groq
        ├── Cohere
        ├── Ollama (local)
        └── 100+ more providers
```

Both services run in Docker containers on a shared internal network. Persistent data lives in named Docker volumes so it survives container restarts and updates.

---

## Screenshots

> Add real screenshots to `docs/screenshots/` and update the paths below.

| OmniRoute — provider routing | Open WebUI — chat interface |
|---|---|
| ![OmniRoute dashboard](docs/screenshots/omniroute-dashboard.png) | ![Open WebUI chat](docs/screenshots/openwebui-chat.png) |

| OmniRoute — model selection | OmniRoute — tier flow |
|---|---|
| ![Model selection](docs/screenshots/omniroute-models.png) | ![Tier flow](docs/screenshots/omniroute-tier-flow.png) |

---

## Requirements

- **Docker** (with Compose v2 plugin) — `docker compose version`
- **just** (optional, recommended) — `brew install just` / `cargo install just`

Without `just`, every command is a plain `docker compose` call.

---

## Install

### 1 — Clone or download

```bash
git clone <repo-url> omniroute-cli
cd omniroute-cli
```

### 2 — Create `.env`

```bash
just init
# or: cp .env.example .env
```

### 3 — Set required secrets

Open `.env` and replace the sample values with real secrets:

```bash
# Generate recommended values:
openssl rand -base64 48   # → OMNIROUTE_JWT_SECRET
openssl rand -hex 32      # → OMNIROUTE_API_KEY_SECRET
openssl rand -base64 24   # → OMNIROUTE_INITIAL_PASSWORD
openssl rand -base64 32   # → OMNIROUTE_WS_BRIDGE_SECRET
openssl rand -hex 32      # → OPENWEBUI_SECRET_KEY
```

**Do not expose the stack publicly with the sample secrets from `.env.example`.**

### 4 — Start the stack

```bash
just up
# or: docker compose pull && docker compose up -d
```

---

## Quickstart

```bash
# 1. Clone
git clone <repo-url> omniroute-cli && cd omniroute-cli

# 2. Create .env with sample secrets
just init

# 3. Pull images and start
just up

# 4. Open in browser
open http://localhost:20000   # Open WebUI
open http://localhost:20128   # OmniRoute admin
```

Default credentials for OmniRoute admin:

| Field | Value |
|-------|-------|
| Username | `admin` |
| Password | value of `OMNIROUTE_INITIAL_PASSWORD` in `.env` |

Add your provider API keys inside OmniRoute's admin UI, then select any model in Open WebUI and start chatting.

---

## Configuration

All configuration lives in `.env`. Copy from `.env.example`:

### Ports

| Variable | Default | Description |
|----------|---------|-------------|
| `OMNIROUTE_PUBLIC_PORT` | `20128` | Host port for OmniRoute |
| `OPENWEBUI_PUBLIC_PORT` | `20000` | Host port for Open WebUI |

### Required secrets

| Variable | Description |
|----------|-------------|
| `OMNIROUTE_JWT_SECRET` | Signs user session tokens |
| `OMNIROUTE_API_KEY_SECRET` | Encrypts stored API keys |
| `OMNIROUTE_INITIAL_PASSWORD` | First-run admin password |
| `OMNIROUTE_WS_BRIDGE_SECRET` | WebSocket bridge auth |
| `OPENWEBUI_SECRET_KEY` | Open WebUI session signing |

### Optional settings

| Variable | Default | Description |
|----------|---------|-------------|
| `OMNIROUTE_STORAGE_ENCRYPTION_KEY` | — | AES key for encrypted storage (recommended for production) |
| `OMNIROUTE_REQUIRE_API_KEY` | `false` | Require API key on every request |
| `OMNIROUTE_AUTH_COOKIE_SECURE` | `false` | Set `true` when served over HTTPS |
| `OMNIROUTE_APP_LOG_TO_FILE` | `true` | Write logs to file inside container |
| `OMNIROUTE_MEMORY_MB` | `512` | OmniRoute Node.js heap limit |
| `OMNIROUTE_OPENAI_API_KEY` | `omniroute-local` | Key Open WebUI sends to OmniRoute |

### Docker images

| Variable | Default |
|----------|---------|
| `OMNIROUTE_IMAGE` | `diegosouzapw/omniroute:latest` |
| `OPENWEBUI_IMAGE` | `openwebui/open-webui:latest` |

Pin to a specific digest for reproducible deployments.

---

## Commands

All commands use `just`. Equivalent `docker compose` calls are shown where helpful.

### Lifecycle

| Command | Description |
|---------|-------------|
| `just up` | Pull images, create and start all containers |
| `just start` | Start already-created containers |
| `just stop` | Stop containers (keep them) |
| `just restart` | Restart all services |
| `just down` | Stop and remove containers; **keep volumes** |

### Monitoring

| Command | Description |
|---------|-------------|
| `just status` | Show container state |
| `just logs` | Follow all logs (last 200 lines) |
| `just logs ai-tools-omniroute` | Follow OmniRoute logs only |
| `just top` | Show processes inside containers |
| `just urls` | Print configured public URLs |

### Maintenance

| Command | Description |
|---------|-------------|
| `just update` | Pull latest images and recreate containers |
| `just pull` | Pull images without restarting |
| `just recreate` | Force-recreate containers from current images |
| `just doctor` | Validate config and show resolved services/images |

### Debugging

| Command | Description |
|---------|-------------|
| `just shell-omniroute` | Shell into OmniRoute container |
| `just shell-openwebui` | Shell into Open WebUI container |

### Backup

| Command | Description |
|---------|-------------|
| `just backup` | Backup both volumes to `./backups/` |
| `just backup-omniroute` | Backup OmniRoute volume only |
| `just backup-openwebui` | Backup Open WebUI volume only |

### Cleanup

| Command | Description |
|---------|-------------|
| `just clean` | Remove containers + prune unused images; **keep volumes** |
| `just clean-all` | Remove containers **and volumes** — deletes all data |

---

## Data persistence

| Volume | Service | Contents |
|--------|---------|----------|
| `ai-tools-omniroute-data` | OmniRoute | Provider keys, user accounts, routing config, logs |
| `ai-tools-omniroute-openwebui-data` | Open WebUI | Chat history, user accounts, settings |

Volumes survive `just down` and `just update`. Only `just clean-all` deletes them (requires typing `DELETE` to confirm).

---

## Updating

```bash
just update
```

Pulls latest images, stops the stack, recreates containers in place. Volumes are preserved.

---

## Connecting other clients

OmniRoute exposes an OpenAI-compatible endpoint at:

```
http://localhost:20128/v1
```

Point any OpenAI SDK or tool at that URL. Use an OmniRoute-generated API key as the bearer token (or the placeholder `omniroute-local` when `OMNIROUTE_REQUIRE_API_KEY=false`).

```python
from openai import OpenAI

client = OpenAI(
    base_url="http://localhost:20128/v1",
    api_key="omniroute-local",
)

response = client.chat.completions.create(
    model="claude-sonnet-4-5",   # any model configured in OmniRoute
    messages=[{"role": "user", "content": "Hello"}],
)
```

---

## Troubleshooting

**Stack won't start — missing secret**

```
Error: OMNIROUTE_JWT_SECRET is required
```

Run `just init` then fill in secrets in `.env`.

**Port already in use**

Change `OMNIROUTE_PUBLIC_PORT` or `OPENWEBUI_PUBLIC_PORT` in `.env`, then `just recreate`.

**Forgot admin password**

Set a new value for `OMNIROUTE_INITIAL_PASSWORD` in `.env` and run `just recreate`. The initial password only applies on first run; if an admin account already exists, reset it from the OmniRoute admin UI instead.

**View logs for a specific service**

```bash
just logs ai-tools-omniroute
just logs ai-tools-omniroute-openwebui
```
