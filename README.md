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

## Quick Start

```bash
# Clone the project
git clone https://github.com/zhemed/new-api-own.git
cd new-api-own

# Edit docker-compose.yml (uses ghcr.io/zhemed/new-api-own:latest by default)
nano docker-compose.yml

# Start the service
docker-compose up -d
```

After deployment, visit `http://localhost:3000` to start using it.

## Data

- SQLite by default, data stored in `./data`
- The data directory can be mounted to any absolute path for backup and migration

## Maintained by

[zhemed](https://github.com/zhemed)
