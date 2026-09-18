# v0.0.3 维护轮次

## 结果摘要

**已完成并发布**：`VERSION 0.0.2 → 0.0.3`，tag `v0.0.3`，GitHub Release 与 GHCR 双标签镜像产出，四个工作流全绿。
维护复查中后端门禁全绿；前端检查因本机无 bun 跳过（原因已记录）；顺带校正 3 处与实际不符的文档。

## 清单与结果

### A. 事实复查（先看再改）

- [x] `git status` 干净、`origin/main..HEAD` 无未推送提交（发版前）
- [x] `git remote -v` 只有 `origin` —— 未引入 upstream，本轮无 fetch/merge/rebase
- [x] 四组限流开关仍为有意关闭：`common/init.go:125/129/133/137` 默认 `false` +
      `docker-compose.yml:46-49` 四个 `*=false`
- [x] 部署安全基线：**部分复核** —— 从本机（（本机地址已脱敏））可无鉴权访问
      `http://（内网地址已脱敏）:3000/api/status`（面板直连可达、/api/status 泄露配置），
      与基线第 1 条一致；防火墙/公网视角需在 `（内网地址已脱敏）` 上跑 `nft list ruleset`，
      **本机无该机 SSH 私钥（`Permission denied`），未做**
- [~] 数据/日志权限（`chmod 700 data data/logs`）：需 SSH 到 `（内网地址已脱敏）`，同上未做

### B. 依赖与工具链

- [x] 后端：`GOWORK=off go vet ./...`、`GOWORK=off go build ./...`（exit 0）；
      `cd relaykit && GOWORK=off go vet/build ./...`（exit 0）；`make test`（exit 0，root + relaykit）
- [~] 前端：`cd web && bun install --frozen-lockfile && bun run typecheck && bun test` ——
      **本机没有 bun（`~/.bun` 不存在），未执行**；需在有 bun 的机器或交给 CI（CI 会跑 typecheck + test）
- [x] 版本记录：Go `go1.26.6`（文档原写 1.26.4，已校正）；Docker `29.7.2` + Compose `v5.4.0`（符合强制标准）；
      bun 未安装（已在文档标明）
- [~] 依赖安全扫描：`govulncheck`、`bun audit` 本机均未安装，未执行

### C. 文档与流程

- [x] `./scripts/check-trellis-gate.sh` 通过（闸门已装 + 提交可追溯）
- [x] `.trellis/spec/` 内 61 处 `file:line` 引用全部有效（0 处失效）
- [x] 文档修正 3 处（本轮改动）：
      1. Go 版本 1.26.4 → 实测 1.26.6
      2. 标明本机没有 bun，前端检查需换机器
      3. 推送方式改为 `git -c credential.helper='!gh auth git-credential'`（旧记录的
         `（本机凭据文件）` 在本机不存在，且 AGENTS.md 禁止改 git config）

### D. 发版 0.0.3

- [x] `VERSION` → `0.0.3`，提交 `3471112`，带 `[task:maintenance-0.0.3]` 锚点
- [x] `git tag -a v0.0.3`，推送 `main` 与 tag
- [x] CI 全绿：`trellis-gate` ×2、`Release (Linux, macOS, Windows)`、`Build Electron App`、
      `Publish Docker image (Multi-arch)`（GitCode 同步按设计 skip）
- [x] Release `v0.0.3`（Latest）附件：`new-api-v0.0.3`、`new-api-arm64-v0.0.3`、
      `new-api-macos-v0.0.3`、`new-api-v0.0.3.exe` + 3 个 checksums
- [x] 下载的 Release 二进制 `--version` → `v0.0.3`
- [x] GHCR 同时存在 `v0.0.3` 与 `0.0.3`（及 `latest`），多架构清单 + cosign 签名
- [ ] 本任务 `finish` + `archive` + journal（本次收尾动作）

## Acceptance Criteria

- [x] 清单全部处理：完成项已勾选，无法执行项写明原因（A 的安全基线/权限需 SSH、B 的前端与漏洞扫描缺工具）
- [x] `main` 与 `origin/main` 同步，tag `v0.0.3` 已推送并在 Releases 页可见
- [x] GHCR 同时存在 `v0.0.3` 与 `0.0.3`，镜像内版本号为 `v0.0.3`

## 后续可选（另开任务，不在本轮）

- 线上实例（（内网地址已脱敏））仍跑着 version 为空的旧镜像（早于 0.0.2），升级到 `ghcr.io/zhemed/new-api-own:0.0.3`
  需要在那台机器上 `docker compose pull && docker compose up -d`（本机无 SSH 私钥）
- 在有 bun 的机器上补跑前端 `typecheck` + `bun test`
- 部署安全基线中依赖 SSH 的项目（防火墙规则、data/logs 权限）补做
