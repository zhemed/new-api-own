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

> **One deployment method only (enforced)**: `docker run` with the published image — a single
> container, host networking, `./data` mounted. Compose / Helm / K8s manifests are rejected by
> both the commit hook and CI (`scripts/forbid-extra-deploy-methods.sh`); see
> [.trellis/spec/guides/deployment-single-method.md](./.trellis/spec/guides/deployment-single-method.md).

### Requirements

- Docker **29.7.2**

One-click install:

```bash
curl -fsSL https://raw.githubusercontent.com/zhemed/new-api-own/main/install-docker.sh | bash
```

### Deploy (no source needed)

```bash
docker run -d --name new-api --restart always \
  --network host \
  -v ./data:/data \
  ghcr.io/zhemed/new-api-own:latest
```

- SQLite by default; data is stored in `./data`
- After deployment, visit `http://localhost:3000`
- Building your own image (`docker build -t new-api-own .`) then running it is the **same**
  `docker run` form, not a second deployment method

Optional environment variables (append `-e ...`): see [`.env.example`](./.env.example) and
[MAINTENANCE.md](./MAINTENANCE.md).

## Maintained by

[zhemed](https://github.com/zhemed)
