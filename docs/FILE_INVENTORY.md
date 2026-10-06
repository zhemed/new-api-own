# 文件有用性分级清单

> 生成于 2026-08-21 · 跟踪文件 1986 · `web/src` 1045 · 结论：**无垃圾文件，均为有用或历史兼容**，仅 2 处小归档建议。
>
> **2026-10-06 复核**（大瘦身后，含当日第二轮删掉的 33 个前端死文件 + 复核阶段追加删除的 `web/src/hooks/use-mobile.tsx`）：
> 跟踪文件 **2085**（其中 `.trellis/` 105——该计数随任务归档浮动、`.agents/` 57）、`web/src` **1012**、`web/` 全目录 **1039**（本行计数为实测快照，仍会随当日清理浮动）。
> 本清单的"遗留兼容"一类已清空——见下文各条与第 4 节的更新。

## 1. 汇总

| 分级 | 目录/文件 | 数量 | 用途 | 处理 |
|------|-----------|------|------|------|
| **核心运行时** | `common(59)/constant(14)/controller(87)/dto(4)/i18n(5)/logger(1)/middleware(33)/model(74)/oauth(9)/relay(238)/relaykit(132)/router(10)/service(90)/setting(53)/types(3)/main.go` | ~813 | Go 后端分层 + 40+ 渠道适配 + 独立 `relaykit` 模块 | **保留**，仅死码标记 |
| **前端** | `web/src/features(628)/components(208)/routes(59)/lib(41)/hooks(19)/...` + `web/*` 配置 | 1040 | React 19 + Rsbuild + Base UI 23 功能域 | **保留** |
| **构建部署** | `Dockerfile, install-docker.sh, scripts/forbid-extra-deploy-methods.sh, makefile, go.mod/sum, VERSION` | 7 | 29.7.2 标准 + 构建注入；部署方式唯一（禁止 compose，见 `.trellis/spec/guides/deployment-single-method.md`）| **保留** |
| **文档配置** | `README*, AGENTS.md, CLAUDE.md, MAINTENANCE.md, LICENSE/NOTICE/THIRD-PARTY, .git*, .dockerignore, .env.example` | 12 | 项目规范与合规 | **保留**（受保护标识） |
| **AI 协作** | `.agents/skills/*`（i18n-translate / shadcn-ui / vercel-react-best-practices / trellis-*） | 57 | Agent 能力 | **保留**（已跟踪） |
| **遗留兼容** | （无）| 0 | 历史迁移/词表/审计文档已于 2026-10-06 全部移除 | **已删除** |

## 2. 关键疑问解答

**为什么看起来多？** `web` 占 50%（1040/2086，2026-10-06 复核；2026-08-21 生成时为 1045/1986 ≈ 53%），`relay/channel` 40 provider 各含 `adapter.go+test+dto`，属多模型聚合网关本质；顶层 30 条目是 Go 分层必需，非膨胀。

**`.agents` 是垃圾吗？** 否。`git ls-files .agents | wc -l` 显示 57 文件被跟踪（2026-10-06 复核），`.gitignore` 未忽略它（仅忽略 `.claude/.cursor`），为 Agent 技能库（Trellis 全套 + 项目技能）。

**`VERSION` 有用吗？** 有用。当前内容是版本号 `0.0.6`（2026-10-06 递增；历史上在自维护基线提交 `1092e46` 中被留空，`f3fbc3e` 起写入版本号）；`Dockerfile` 用 `-X github.com/QuantumNous/new-api/common.Version=$(cat VERSION)` 注入，`docker-build.yml` 在推 tag 时用 tag 覆盖该文件后重建镜像（`Dockerfile:9`、`Dockerfile:29`），勿删。

**`bin/` 那俩 `.sql` 能删吗？** 已删。`bin/`（`migration_*.sql` + `time_test.sh`）于 2026-10-06 随历史迁移脚本一并移除，无需再归档；历史仍可从 git 历史取回（见本节第 2 条）。

**`electron/` 呢？** 已于 2026-10-06 整体移除：交付只有镜像（`docker run` + 公开镜像），桌面壳与二进制 Release 均不在维护范围。

**`docs/translation-glossary.*` 呢？** 已于 2026-10-06 随历史文档一并删除。若后续要做多语言术语统一，届时按需重建（`web/src/i18n/locales/{7}.json` 仍是唯一运行时文案来源）。

**`web/src/components/ui` 60+ 感觉多？** `knip.config.ts` 已 `ignore: ['src/components/ui/**']`，为 shadcn 按需底座，非死码。

## 3. 验证

- `git ls-files --others --exclude-standard | wc -l` = 0（无未跟踪残留；`.trellis/tasks/` 下**进行中**的任务目录在归档提交前会短暂出现，属正常）
- `find -name "*.tmp|*.bak|*.swp|.DS_Store|tiktoken_cache"` = 0
- `find -perm 755 -name "*.md"` 已修为 644（`SKILL.md`）
- `.env` 真文件不存在，仅 `.env.example`（已查无密钥泄露）

## 4. 建议

1. **不动**：核心/前端/构建/文档/AI 协作 5 类
2. **已完成**：`bin/migration_*.sql`、`docs/translation-glossary.*`、`docs/AUDIT_REPORT.md`、`docs/installation/BT.md` 均于 2026-10-06 删除
3. **后续**：`bun run knip` 跑一次复核 `web` 未用导出，`GOWORK=off go vet` 查 Go 死码，仅打 `// Deprecated` 不直接删

> 结论：当前 **2086** 文件（2026-10-06 复核，实测 `git ls-files | wc -l`）均为有用或兼容保留，**无需精简**；如觉 GitHub 列表视觉冗余，仅因 `web` 与 `relay` 天然文件多，可通过 GitHub 折叠或本地 `ls` 过滤查看。
