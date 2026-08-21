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

### Option 1: Docker image (recommended, no source needed)

## 环境要求

- Docker **29.7.2** + Docker Compose **v5.4.0**（项目标准版本）
- 部署前运行 `./check-docker-env.sh` 校验环境

```bash
# 私有镜像，需要先登录 GHCR（使用 GitHub 凭据）
docker login ghcr.io -u zhemed

docker run -d --name new-api --restart always \
  -p 3000:3000 \
  -v ./data:/data \
  ghcr.io/zhemed/new-api-own:latest
```

- SQLite by default; data is stored in `./data`
- After deployment, visit `http://localhost:3000`

### Option 2: Build from source (requires repository access)

```bash
# Private repository: cloning requires GitHub credentials
git clone https://github.com/zhemed/new-api-own.git
cd new-api-own

# Single-container build and run
docker build -t new-api-own .
docker run -d --name new-api --restart always \
  -p 3000:3000 -v ./data:/data new-api-own
```

### Production deployment reference

The `docker-compose.yml` in this repository is a host-network production setup with PostgreSQL/Redis. Adjust the default passwords before use:

```bash
docker-compose up -d
```

## Maintained by

[zhemed](https://github.com/zhemed)
