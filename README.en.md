# New API

A self-hosted LLM gateway and AI asset management platform: multi-model aggregation, key management, billing, and distribution.

## Features

- **Multi-model aggregation**: unified access to OpenAI, Claude, Gemini, DeepSeek, Qwen, and more
- **API format conversion**: OpenAI-compatible ⇄ Claude Messages, OpenAI → Gemini, and more
- **Key management**: multi-key pooling, grouping, model restrictions, usage statistics, and dashboards
- **Billing and quotas**: per-request, usage-based, and cache-hit billing with flexible policies
- **Authorization logins**: Discord, Telegram, LinuxDO, and OIDC unified authentication
- **Intelligent routing**: weighted random, automatic retry, and per-user model rate limiting
- **Reasoning effort support**: OpenAI o-series, Claude thinking, Gemini thinking
- **Modern UI**: clean interface with multi-language support

## Deployment

### Requirements

- Docker **29.7.2** + Docker Compose **v5.4.0** (standard)

One-click install:

```bash
curl -fsSL https://raw.githubusercontent.com/zhemed/new-api-own/main/install-docker.sh | bash
```

### Option 1: One-command deployment (recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/zhemed/new-api-own/main/install-compose.sh | sudo bash
```

Installs to `/opt/docker/new-api-own`. The script generates random database passwords into `.env`
(mode 600; re-runs **reuse** them and never regenerate), fetches `compose.yaml` from the same ref,
starts new-api + PostgreSQL + Redis, waits for the health check, then tightens `data/` and `logs/`
permissions.

**Re-running is not an upgrade**: when `.env` already exists its image tag is reused, so re-running
just to fix configuration does not change versions. Use `--upgrade` (follows `VERSION` on the ref)
or `--tag v0.0.3` to pin a version.

An existing deployment is never overwritten (`--force` rewrites after backing up), and `./data`,
`./logs` plus the `pg_data` volume are never deleted. The script **exits non-zero** when the health
check fails, leaving the containers in place for inspection. See `--help` for all options.

> Read the deployment security baseline in [MAINTENANCE.md](./MAINTENANCE.md) first: host networking
> exposes the panel on every interface, and `/api/status` answers without authentication.

### Option 2: Docker image (single container, no source needed)

```bash
docker run -d --name new-api --restart always \
  --network host \
  -v ./data:/data \
  ghcr.io/zhemed/new-api-own:latest
```

- SQLite by default; data is stored in `./data`
- After deployment, visit `http://localhost:3000`

### Option 3: Build from source (requires repository access)

```bash
git clone https://github.com/zhemed/new-api-own.git
cd new-api-own

# Single-container build and run
docker build -t new-api-own .
docker run -d --name new-api --restart always \
  --network host \
  -v ./data:/data new-api-own
```

### Production deployment reference

[compose.yaml](./compose.yaml) is a host-network production setup with PostgreSQL/Redis.
`POSTGRES_PASSWORD` and `REDIS_PASSWORD` are **required** — there is deliberately no fallback, so a
missing `.env` makes `docker compose config` / `up` fail loudly instead of booting a database
superuser with a well-known password. Create a `.env` first:

```bash
cat > .env <<'EOF'
POSTGRES_PASSWORD=<random, e.g. openssl rand -hex 16>
REDIS_PASSWORD=<random>
EOF
docker compose up -d
```

The file name is `compose.yaml`, Compose v2's preferred name (measured precedence:
`compose.yaml` > `compose.yml` > `docker-compose.yml` > `docker-compose.yaml`; several at once
produce a warning).

## Maintained by

[zhemed](https://github.com/zhemed)
