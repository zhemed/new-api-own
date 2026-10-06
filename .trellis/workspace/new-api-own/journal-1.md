# Journal - new-api-own (Part 1)

> AI development session journal
> Started: 2026-09-18

---



## Session 1: Trellis bootstrap: backend spec filled from real code
<!-- trellis-session: v=2 fp=da614e3bf7bf0b45 -->

**Date**: 2026-09-18
**Task**: Trellis bootstrap: backend spec filled from real code
**Branch**: `main`

### Summary

Cloned zhemed/new-api-own into /root/new-api-own, initialized Trellis (dev: new-api-own, dsh skills), filled the five .trellis/spec/backend guidelines from AGENTS.md plus real file:line evidence, added the Trellis managed block to the project AGENTS.md, and archived the 00-bootstrap-guidelines task.

### Git Commits

| Hash | Message |
|------|---------|
| `8e62ba2` | docs(trellis): 填充后端规范并接入 Trellis 托管块 [task:00-bootstrap-guidelines] |

### Status

[OK] **Completed**


## Session 2: Disable global web/API/critical/search rate limiters
<!-- trellis-session: v=2 fp=ff0d3ab684fe1acb -->

**Date**: 2026-09-18
**Task**: Disable global web/API/critical/search rate limiters
**Branch**: `main`

### Summary

Flipped the four *_ENABLE rate-limit defaults to false in common/init.go and common/constants.go, declared them explicitly off in docker-compose.yml and .env.example, updated MAINTENANCE.md self-hosted notes, and recorded the policy in .trellis/spec/backend/quality-guidelines.md. Counts and durations kept so any limiter can be restored with a single env var. go vet/build and make test all green.

### Git Commits

| Hash | Message |
|------|---------|
| `a4e74df` | chore(config): 默认关闭全局 web/API/敏感操作与搜索限流 [task:09-18-disable-global-rate-limits] |

### Status

[OK] **Completed**


## Session 3: Audit: VERSION 0.0.1 origin and the （内网地址已脱敏） reference
<!-- trellis-session: v=2 fp=f0e0f973f72eb5e2 -->

**Date**: 2026-09-18
**Task**: Audit: VERSION 0.0.1 origin and the （内网地址已脱敏） reference
**Branch**: `main`

### Summary

Read-only audit. VERSION was an empty file from the self-maintained baseline (1092e46) until e1fcb53 (2026-09-12) set it to 0.0.1, which is why the built image reported an empty version. No git tags exist, so the tag-driven image workflow (which overwrites VERSION from the tag) has never run. The （内网地址已脱敏） comment in docker-compose.yml:4 came from 4a24792 (2026-08-13, host-network switch) and is the only real internal address in the repo.

### Git Commits

(No commits - planning session)

### Status

[OK] **Completed**


## Session 4: Release v0.0.2: dual image tags, GitHub Release, version-injection fix
<!-- trellis-session: v=2 fp=52f6fd9dfaf24603 -->

**Date**: 2026-09-18
**Task**: Release v0.0.2: dual image tags, GitHub Release, version-injection fix
**Branch**: `main`

### Summary

Fixed the abbreviated -X ldflags path in release.yml/electron-build.yml that silently left every release binary at v0.0.0; bumped VERSION to 0.0.2; made docker-build.yml publish both v0.0.2 and 0.0.2 plus latest with multi-arch manifests and cosign signatures; replaced the internal IP in the compose comment; documented the release runbook in MAINTENANCE.md. Tagged and pushed v0.0.2: the GitHub Release carries Linux/macOS/Windows binaries and the GHCR image answers on both tags with version v0.0.2.

### Git Commits

| Hash | Message |
|------|---------|
| `6829e8e` | chore(release): v0.0.2 发版准备（双标签镜像 + 修复版本注入 + 中性化注释） [task:09-18-release-v0.0.2] |

### Status

[OK] **Completed**


## Session 5: Maintainer onboarding flow documented and verified
<!-- trellis-session: v=2 fp=82fd5bf554cfc413 -->

**Date**: 2026-09-18
**Task**: Maintainer onboarding flow documented and verified
**Branch**: `main`

### Summary

Verified push state (origin/main = f86cb24, tag v0.0.2 on remote), verified the task workflow artifacts ship with the repo, and reproduced a fresh clone: without a developer identity get_context.py errors, while trellis init --dsh -u <name> -s -y writes the local identity, creates a 00-join-<name> onboarding task and leaves the tree clean except .template-hashes.json. Documented the whole path in MAINTENANCE.md (new 维护者接入 section) and README 维护.

### Git Commits

| Hash | Message |
|------|---------|
| `2bc0e1a` | docs(maintenance): 维护者接入流程（clone → trellis init → 任务与发版） [task:09-18-maintainer-onboarding] |

### Status

[OK] **Completed**


## Session 6: Trellis commit gate enabled (3 layers)
<!-- trellis-session: v=2 fp=a327a658fce5f880 -->

**Date**: 2026-09-18
**Task**: Trellis commit gate enabled (3 layers)
**Branch**: `main`

### Summary

Ported the komari-style three-layer gate: .githooks/pre-commit + commit-msg reject commits without an in-progress task or without a valid [task:<slug>] anchor; scripts/check-trellis-gate.sh audits every commit after .trellis/gates/enforce-from; .github/workflows/trellis-gate.yml runs the same audit on push to main and PRs. Documented in MAINTENANCE.md 流程闸门, AGENTS.md TRELLIS-GATE block and .trellis/spec/guides/trellis-gate-guide.md. Verified all four paths: missing anchor rejected, unknown slug rejected, no in-progress task rejected, and a --no-verify bypass commit caught by the audit. Also opened the v0.0.3 maintenance round task.

### Git Commits

| Hash | Message |
|------|---------|
| `47c61db` | feat(process): Trellis 三层提交闸门（hooks + 审计脚本 + CI 兜底） [task:09-18-commit-gate] |

### Status

[OK] **Completed**


## Session 7: Investigation: x-opencode-session / （实例兜底值已脱敏） channel override
<!-- trellis-session: v=2 fp=1041b2c7f8a1658f -->

**Date**: 2026-09-18
**Task**: Investigation: x-opencode-session / （实例兜底值已脱敏） channel override
**Branch**: `main`

### Summary

Read-only investigation. The live （实例域名已脱敏） gateway runs a single channel (id=1 （渠道名已脱敏）, type=NewAPI, base_url https://opencode.ai/zen/go) whose param_override passes through x-opencode-session/Session-Id/X-Session-Id and falls back to the literal （实例兜底值已脱敏） when the client sends none. Evidence from the 2026-09-16 DB backup: the value appears exactly once in channels.param_override; the 09-12 00:41 backup is empty while the 10:41 one already has it, and commit 25c9aa1 (09-12 11:05) turned it into the reusable {client_header:NAME|DEFAULT} placeholder plus panel preset and the MAINTENANCE.md chapter. Upstream therefore sees our egress IP plus a session id, with the documented cost that every client without its own header shares one prompt-cache bucket. Blind spot: the value could not be re-read from the live DB (no SSH key to （内网地址已脱敏）); the live /api/status reports an empty version, i.e. an image built before the 09-12 version fill.

### Git Commits

(No commits - planning session)

### Status

[OK] **Completed**


## Session 8: Status check: v0.0.3 not released yet
<!-- trellis-session: v=2 fp=c3fccc598a0b0ba2 -->

**Date**: 2026-09-18
**Task**: Status check: v0.0.3 not released yet
**Branch**: `main`

### Summary

Verified across every release outlet: VERSION is still 0.0.2, the only remote tag is v0.0.2, the only GitHub Release is v0.0.2 (Latest), GHCR carries only v0.0.2/0.0.2/latest, and the 09-18-maintenance-0.0.3 task is still in planning. Also noted that the live instance at （内网地址已脱敏） reports an empty version, i.e. a build older than 0.0.2, and that bun is absent on this machine so frontend checks cannot run here.

### Git Commits

(No commits - planning session)

### Status

[OK] **Completed**


## Session 9: 评估：弱盘机器把日志放内存
<!-- trellis-session: v=2 fp=93442baa6bedabd1 -->

**Date**: 2026-10-06
**Task**: 评估：弱盘机器把日志放内存
**Branch**: `main`

### Summary

评估结论：两类日志（应用日志追加写 vs 用量日志每行一事务+fsync）；弱盘痛点在 fsync。零代码可做：应用日志 -log-dir 关闭或走 tmpfs；日志库 DSN 加 _pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)（本机实测 delete/FULL(2) → wal/NORMAL(1)）。日志整体进内存：主库非 SQLite 时纯配置可达（LOG_SQL_DSN=local + SQLITE_PATH=/dev/shm/...），主库即 SQLite 时需小改支持 SQLite 文件/内存库；内存方案需先解决自动裁剪（清理任务当前是手动触发）。不建议整库 tmpfs 或 MySQL datadir 上 tmpfs。

### Main Changes

- 本机探针实测 PRAGMA 可经 DSN 配置，并发现默认 DSN 的 _busy_timeout=30000 未生效（读回 5000）
- 记录内存方案风险：重启即失、无自动上限、多节点各自为政、日志含计费 quota 时不可丢

### Git Commits

(No commits - planning session)

### Testing

- [OK] [OK] 只读评估：未改代码/配置，未重启服务，未访问外部主机
- [OK] [OK] 探针目录已删除，工作树仅剩 .trellis 记录

### Status

[OK] **Completed**

### Next Steps

- 等用户选定路线（零代码两步 / 加内存库支持 / 先不动）
