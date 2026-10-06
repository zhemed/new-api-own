# 运维文档卫生审查（团队）

- 任务：`10-06-team-review-ops`
- 归属：团队评审中 `review-ops`（运维/文档/卫生审查员）
- 共享任务：`task-3`

## 背景

仓库刚完成一次大幅瘦身，交付形态收敛为**唯一一种**部署方式：`docker run` + 公开镜像
`ghcr.io/zhemed/new-api-own`（单容器、`--network host`、挂 `./data:/data`）。

已删除：`electron/`、`.github/workflows/{release,electron-build,sync-release-to-gitcode,docker-image-branch,ci,pr-check}.yml`、
`bin/`、`docs/AUDIT_REPORT.md`、`docs/translation-glossary*.md`、`docs/installation/`、`new-api.service`、`docker-compose*.yml`。

现存工作流只有两个：`docker-build.yml`（推 tag 出镜像）与 `trellis-gate.yml`（提交闸门）。
新增机制：`scripts/forbid-extra-deploy-methods.sh` + 规则源 `.trellis/spec/guides/deployment-single-method.md`。
新增环境变量：`LOG_SQL_DSN`（`memory`/`:memory:`/`sqlite:<path>`）、`LOG_CLEANUP_INTERVAL`、
`LOG_CLEANUP_RETENTION_DAYS`、`LOG_MEMORY_MAX_ROWS`、`SQLITE_PATH`（可加 `&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)`）。

## 目标

核对"大瘦身之后"的文档/配置/工作流与仓库现状是否一致，修掉低风险不一致，产出可执行的结论清单。

## 审查范围与验收标准

1. **引用一致性**：全仓库 grep 不得再出现指向已删文件/工作流/功能的**有效引用**
   （README.md、README.en.md、MAINTENANCE.md、AGENTS.md、CLAUDE.md、docs/**、.env.example、
   scripts/**、.githooks/**）。历史性措辞（如"已删除"说明、do-not-restore 清单）不算不一致。
   - 验收：逐条给出 `文件:行` 证据；能直接改的（死链、过时命令）直接改。
2. **文档与现状一致**：
   - 部署/发版说明只剩镜像路径 `ghcr.io/zhemed/new-api-own`，不再出现 compose/Helm/systemd 作为可选方案；
   - `MAINTENANCE.md` 的铁律、流程闸门、部署安全基线仍准确；**PR 质量门禁已删 → 本地自查是唯一手段**，
     文档必须写明这一点。
   - 验收：相关段落可被逐句对照现状，无"已删功能仍被描述为可用"的表述。
3. **`.env.example` 覆盖度**：`LOG_SQL_DSN`、`LOG_CLEANUP_INTERVAL`、`LOG_CLEANUP_RETENTION_DAYS`、
   `LOG_MEMORY_MAX_ROWS`、`SQLITE_PATH` 均有条目且注释与代码默认值一致。
   - 验收：变量名可 grep 到；注释中的取值说明与 `common/`/`model/` 代码一致（只读核对代码，不改代码）。
4. **仓库卫生**：`.gitignore` 覆盖 `.local-instance/` 与构建产物；无空目录/孤儿文件。
   - 验收：列出 `.gitignore` 缺失项并补齐（仅限允许改动的文件）。
5. **敏感信息扫描**：内网地址/域名/主机名/密钥残留。
   - 验收：**只报告文件数与类别，绝不复述内容**。
6. **`docs/FILE_INVENTORY.md` 准确性**：与当前实际文件树对照，不准确的直接改。
   - 验收：清单中不再列出已删文件；新增文件按需补齐。

## 硬约束

- 禁止网络动作（不 gh / curl / 触发工作流 / docker push）；禁止访问生产；禁止 git commit / push / tag。
- 只改：`.github/**`、`docs/**`、`README.md`、`README.en.md`、`MAINTENANCE.md`、`AGENTS.md`、
  `CLAUDE.md`、`.env.example`、`.gitignore`、`scripts/**`、`.githooks/**`。
- 不动 Go 代码、`web/**`、`.local-instance/`。
- 校验手段限本地：`python3 -c "import yaml"`、`bash -n`、grep、diff。

## 交付物

1. 不一致清单（证据 `文件:行`）+ 已修清单（含 diff 摘要）；
2. 敏感信息扫描结论（文件数/类别，不复述内容）；
3. 工作流与文档一致性的一句话结论；
4. 待确认项。
5. 给 Lead 的中文汇报（结论先行，40 行以内）。

## Notes

- 细节留在本 PRD / 审查记录中，汇报只给结论。

---

# 审查记录（2026-10-06）

## A. 不一致清单（证据）

| # | 位置 | 问题 | 处置 |
|---|---|---|---|
| A1 | `MAINTENANCE.md:355`（原） | 发版校验用 `gh release view v0.0.3`——Release 工作流已删，该产物不存在 | 已改为 `docker run --rm <image> --version` + `docker buildx imagetools inspect` |
| A2 | `MAINTENANCE.md:358`（原） | "只改文件不推 tag 不会产生 Release"——现状只产镜像 | 已改为"不会触发镜像构建…不再有 GitHub Release" |
| A3 | `MAINTENANCE.md:70`（原） | "Release 需要 `contents: write`"——Release 工作流已删 | 已改为两个现存工作流的真实 `permissions:`（`packages/id-token: write`、`contents: read`） |
| A4 | `MAINTENANCE.md:27`（原） | "要换到有 bun 的机器或交给 CI"——PR 质量门禁已删，无 CI 可交 | 已明确"没有 CI 可兜底" |
| A5 | `MAINTENANCE.md:27/28/434` | 脱敏占位残留成嵌套括号（`（\`ubuntu\` / （本机地址已脱敏））`、`（\`（本机凭据文件）\`）`） | 已改为干净措辞，并顺带移除残留主机名 |
| A6 | `MAINTENANCE.md:217` | 历史条目里仍写着实例域名（敏感信息） | 已改为"实例域名（已脱敏）" |
| A7 | `MAINTENANCE.md:222` 质量门禁节 | 基线是 2026-08-18，未标注 PR 门禁已删 | 已加 2026-10-06 变更提示：无 CI 兜底、本地自查唯一、`docker-build.yml` 不跑测试 |
| A8 | `MAINTENANCE.md:431` 工作流节 | 同样缺"本地自查是唯一手段" | 已补 |
| A9 | `.gitignore:27-28`（原） | `electron/node_modules`、`electron/dist` 指向已删目录 | 已删这两行 |
| A10 | `docs/FILE_INVENTORY.md:24`（原） | `bin/` Q&A 说"建议不动或移 bin/archive/"——`bin/` 已删，与第 4 节自相矛盾 | 已改为"已删（2026-10-06）" |
| A11 | `docs/FILE_INVENTORY.md:14`（原） | 行标签仍列 `bin/migration_*.sql(2)+time_test.sh` 而数量写 0 | 已改为"（无）" |
| A12 | `docs/FILE_INVENTORY.md:3/9/10/11/12/13/20/45` | 计数过期：总文件 1986、`web/src` 1045、`components` 217、`model` 73、`service` 89、`.agents` 14 | 已按实测更新：2106 / 1046 / 218 / 74 / 90 / 57；行内合计 9→8、13→12 |
| A13 | `docs/FILE_INVENTORY.md:34`（原） | "未跟踪残留 = 0"不可复现（进行中任务目录会短暂出现） | 已限定范围 |
| A14 | `.env.example:30`（原） | 只写了 `LOG_SQL_DSN=memory`，代码同时支持 `:memory:` | 已补 `# LOG_SQL_DSN=:memory:` 与等价说明 |

**核对通过、无需改动**：`README.md` / `README.en.md`（部署只剩镜像路径、发版只讲 tag → 镜像）、
`AGENTS.md`（唯一部署方式段、do-not-restore 引用）、`.githooks/pre-commit`、`scripts/forbid-extra-deploy-methods.sh`、
`.github/workflows/docker-build.yml`、`.github/workflows/trellis-gate.yml`、`.github/ISSUE_TEMPLATE/*`、
`docs/authentication.md`、`docs/channel/other_setting.md`、`docs/ionet-client.md`（无已删文件/工作流引用）。
`.trellis/spec/guides/deployment-single-method.md` 的 do-not-restore 清单与工单一致（`docker-compose.yml` /
`docker-compose.dev.yml` / `new-api.service` 均已移除且未被带回）。

## B. 范围外待修（不在我的写入白名单，需 Lead 指派）

| # | 位置 | 问题 | 建议 |
|---|---|---|---|
| B1 | `makefile:4,28,32,53,55,61` | `DEV_COMPOSE_FILE = docker-compose.dev.yml`（文件已删）→ `make dev` / `dev-api` / `dev-api-rebuild` / `reset-setup` **必然报错**；且是残留的 compose 形态 | 删除这几个 compose 目标与变量，改为直连 `go run` + `bun run dev`（属 `makefile`，不在我的白名单） |
| B2 | `Dockerfile:1`、`Dockerfile.dev:1` | 注释引用 `./check-docker-env.sh`，仓库里已无此脚本 | 删掉该半句或恢复脚本 |
| B3 | `.gitattributes:40` | `electron/** linguist-vendored` 指向已删目录 | 删该行 |
| B4 | `web/src/features/setup/components/database-step.tsx:74-126` | 仍保留 Electron 检测分支（桌面壳已删） | 交由前端审查员判断是否清理（死分支，非故障） |

## C. 敏感信息扫描结论

- 跟踪文件内**无**可用凭据：API Key / 私钥 / PAT / OAuth token 形态的模式串均为**掩码正则或 PEM 拼装代码**
  （`relay/channel/vertex/service_account.go`、`relaykit/relayconvert/kitutil/mask.go`），非真实密钥。
- 私网地址（10/172.16-31/192.168）只出现在**代码默认值、测试与 i18n 文案**中（RFC1918 语义），无实例专属地址。
- 实例专属域名 1 处、主机名 1 处（均在 `MAINTENANCE.md`）**已脱敏/移除**。
- `.local-instance/` 是运行时目录，`.gitignore` 已覆盖且**未被跟踪**（含实例数据，不进仓库）。
- 结论：**0 个文件含真实密钥；0 个文件含可用的内网地址/实例域名（修复后）**。

## D. 工作流与文档一致性（一句话）

现存两个工作流（`docker-build.yml` / `trellis-gate.yml`）的触发条件、产出标签、`permissions` 与
`README.md`+`MAINTENANCE.md` 的发版说明完全一致；唯一不一致是历史 Release 产物残留（A1–A3，已修），
以及 `makefile` 内残留的 compose 目标（B1，范围外）。

## E. 本地校验（全部通过）

- `bash -n`：`scripts/*.sh`、`.githooks/pre-commit`、`.githooks/commit-msg`、`install-docker.sh` → OK
- `python3 -c "import yaml"`：两个工作流 + ISSUE_TEMPLATE → OK
- `sh scripts/forbid-extra-deploy-methods.sh` → ✅ 部署方式唯一
- `./scripts/check-trellis-gate.sh` → ✅ 提交可追溯（8a991149..HEAD 共 11 个提交）
- 未执行任何网络动作、未 docker/compose、未 commit/push

---

# 补充轮（Lead 指派，共享任务 `task-4`）

| # | 位置 | 问题 | 处置 |
|---|---|---|---|
| F1 | `README.en.md:50`（原） | 只有 `## Maintained by` 一行，缺中文 README 的「维护/发版」段，英文读者看不到唯一交付路径与本地自查要求 | 已补 `## Maintenance`：Trellis 接入、`VERSION`+tag 发版、推 tag 只触发 `Publish Docker image (Multi-arch)`（`v<版本>`/`<版本>`/`:latest` + 多架构 + cosign）、镜像是唯一交付产物、PR 质量门禁已移除（本地自查唯一）、提交需 `[task:<slug>]` 锚点 |
| F2 | `.gitattributes:40` | `electron/** linguist-vendored` 指向已删目录 | 已删除 |
| F3 | `.gitattributes` 全量复核 | 逐条核对剩余规则：`*`、各语言 eol、二进制、`web/**/*.tsx`、`web/src/routeTree.gen.ts`、`.trellis/workspace/*/journal-*.md` | **无其它死规则**（`web/src/routeTree.gen.ts` 存在；无 `bin/`、`docs/`、`new-api.service` 相关条目） |

校验：`git diff` 只含 `README.en.md` 与 `.gitattributes`；两份 README 代码围栏各 8 个（配平）；未提交、无网络动作。


