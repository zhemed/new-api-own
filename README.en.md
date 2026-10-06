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

### Upgrade / rollback (never left on an old version)

> **Key point: restarting a container does not change its image.** `docker restart` (or an automatic
> restart from `--restart always`) just runs the **same image** again, so the version cannot change.
> Upgrading always means **removing the container and recreating it from the new image**.

Three steps (`<version>` is the target tag, e.g. `0.0.6`; **run this in your original deploy directory** —
`./data` is a relative path):

```bash
docker pull ghcr.io/zhemed/new-api-own:<version>   # 1. pull the new image first
docker rm -f new-api                               # 2. remove the old container (data stays in ./data)
docker run -d --name new-api --restart always \
  --network host -v ./data:/data \
  ghcr.io/zhemed/new-api-own:<version>             # 3. recreate from the new image
```

Rollback uses the same three steps with the previous tag (every historical tag stays in the registry):

```bash
docker pull ghcr.io/zhemed/new-api-own:0.0.5
docker rm -f new-api
docker run -d --name new-api --restart always --network host -v ./data:/data \
  ghcr.io/zhemed/new-api-own:0.0.5
```

**Why an explicit `docker pull`, and why not `:latest`**: `:latest` is a **local tag** — `docker run …:latest`
does not check the registry for updates, it just uses whatever the local tag points at. That is how you
"upgrade" and still end up on an old version: on 2026-10-06 this machine's `latest` was still stuck on `0.0.3`
(refreshed the same day), and **any machine's local tag can go stale the same way**. So pin the tag (`0.0.6`
and `v0.0.6` are the same multi-arch manifest); if you must use `:latest`, run
`docker pull ghcr.io/zhemed/new-api-own:latest` first, then recreate the container.

> To check whether the local image is stale: `docker images ghcr.io/zhemed/new-api-own --digests`.
> **Always filter by the full repository name** — `docker images | grep 0.0.5` also matches unrelated
> projects' tags and, because `.` is a regex wildcard in `grep`, it can give you the wrong answer.

Verify the running version after upgrading (the instance itself is the only source of truth):

```bash
curl -s http://127.0.0.1:3000/api/status | grep -o '"version":"[^"]*"'
# expect: "version":"v0.0.6"; if it differs, walk the checks in MAINTENANCE.md
# with a source checkout, one command covers all four checks (repo VERSION / latest tag /
# registry digests / running instance):
#   make check-version INSTANCE=http://127.0.0.1:3000
```

> Upgrades and rollbacks **still have exactly one form: `docker run`**. Do not introduce compose / Helm /
> K8s / systemd for them — the commit/CI gate (`scripts/forbid-extra-deploy-methods.sh`) rejects that.

### In-panel update (catch up temporarily, not a second deployment method)

**System Settings → System maintenance → Check for updates** shows the running and latest versions. When a
newer version exists, an admin can click **Update now**: the backend downloads the Linux binary for **this
machine's architecture**, verifies it against the published `SHA256SUMS`, and only then atomically replaces
the binary and re-executes in place (**a failed checksum never replaces anything**). Checking and downloading
both happen server-side; your browser never talks to GitHub.

> ⚠️ **This is a temporary measure**: a binary replaced inside the container is **not part of the image**, so
> the next `docker rm` + `docker run` falls back to the image's version. To stay on a version long-term, use
> the three-step image upgrade above. Switches: the update **check** is off by default
> (`UPDATE_CHECK_ENABLED=true` enables it) while **Update now** is allowed by default
> (`UPDATE_APPLY_ENABLED=false` disables it entirely); if GitHub is unreachable, use `HTTPS_PROXY` or point
> `UPDATE_CHECK_API_BASE_URL` at a mirror/proxy. Details, troubleshooting and rollback:
> [MAINTENANCE.md](./MAINTENANCE.md).

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
