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

## Maintenance

Maintained by [zhemed](https://github.com/zhemed) with the [Trellis](https://github.com/mindfold-ai/trellis)
task workflow and GitHub Actions release automation. Full runbook: [MAINTENANCE.md](./MAINTENANCE.md).

Onboarding (workflow artifacts and skills ship with the repo; each machine only initializes its identity once):

```bash
git clone https://github.com/zhemed/new-api-own.git
cd new-api-own
trellis init --dsh -u <your-developer-name> -s -y   # creates a 00-join-<name> task; never overwrites repo files
```

Release: bump `VERSION` (0.0.3, 0.0.4, …), commit, then tag:

```bash
git tag -a v0.0.3 -m "v0.0.3"
git push origin main v0.0.3
```

Pushing a tag **only triggers the image build**: it publishes `v<version>`, `<version>`, and `:latest` to
`ghcr.io/zhemed/new-api-own` (multi-arch manifest + cosign signature).

**The image is the only delivery artifact**: a tag run triggers `Publish Docker image (Multi-arch)`, producing
`v<version>` / `<version>` / `:latest`. Binary releases, the Electron desktop shell, and GitCode sync have all been removed.

### Before you push

PR quality gates were removed on 2026-10-06, so **local self-checks are the only gate**: run backend
`go vet` / `go build` / `make test` (plus `cd relaykit && GOWORK=off go build ./...`) and frontend
`bun run typecheck` / `bun run lint` / `bun test`. The tag build only builds and pushes images — it does not run tests.

Commits must carry a Trellis task anchor (`[task:<slug>]`); the local commit hook and the `trellis-gate`
workflow reject anything else.
