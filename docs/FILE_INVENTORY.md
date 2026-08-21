# 文件有用性分级清单

> 生成于 2026-08-21 · 跟踪文件 1986 · `web/src` 1045 · 结论：**无垃圾文件，均为有用或历史兼容**，仅 2 处小归档建议。

## 1. 汇总

| 分级 | 目录/文件 | 数量 | 用途 | 处理 |
|------|-----------|------|------|------|
| **核心运行时** | `common(59)/constant(14)/controller(87)/dto(4)/i18n(5)/logger(1)/middleware(33)/model(73)/oauth(9)/relay(238)/relaykit(132)/router(10)/service(89)/setting(53)/types(3)/main.go` | ~811 | Go 后端分层 + 40+ 渠道适配 + 独立 `relaykit` 模块 | **保留**，仅死码标记 |
| **前端** | `web/src/features(645)/components(217)/routes(59)/lib(42)/hooks(21)/...` + `web/*` 配置 | 1072 | React 19 + Rsbuild + Base UI 23 功能域 | **保留** |
| **构建部署** | `Dockerfile/.dev, docker-compose*.yml, install-docker.sh, new-api.service, makefile, go.mod/sum, VERSION` | 9 | 29.7.2+v5.4.0 标准 + 构建注入 | **保留** |
| **文档配置** | `README*, AGENTS.md, CLAUDE.md, MAINTENANCE.md, LICENSE/NOTICE/THIRD-PARTY, .git*, .dockerignore, .env.example` | 13 | 项目规范与合规 | **保留**（受保护标识） |
| **AI 协作** | `.agents/skills/{i18n-translate,shadcn-ui,vercel-react-best-practices}` | 14 | Agent 能力 | **保留**（已跟踪） |
| **遗留兼容** | `bin/migration_*.sql(2)+time_test.sh`, `electron(12)`, `docs/translation-glossary.*`, `docs/AUDIT_REPORT.md` | ~20 | 历史迁移/桌面壳/词表/本次审计 | **保留或归档** |

## 2. 关键疑问解答

**为什么看起来多？** `web` 占 54%（1045/1986），`relay/channel` 40 provider 各含 `adapter.go+test+dto`，属多模型聚合网关本质；顶层 30 条目是 Go 分层必需，非膨胀。

**`.agents` 是垃圾吗？** 否。`git ls-files --cached | grep .agents` 显示 14 文件被跟踪，`.gitignore` 未忽略它（仅忽略 `.claude/.cursor`），为计划中的 Agent 技能库。

**`VERSION` 空文件有用吗？** 有用。历史 `f4450040` 故意留空，`go build -ldflags -X common.Version=$(cat VERSION)` 注入，勿删勿填。

**`bin/` 那俩 `.sql` 能删吗？** 建议不动或移 `bin/archive/` 并注释。`git log --follow -- bin/` 显示历史迁移，留作可追溯，体积 <1K。

**`electron/` 12 文件必须吗？** 可选。若不发桌面版，可标记 `optional` 并保持 `electron/dist` 忽略；删则影响 `electron/build.sh` 与 `release.yml` 桌面产物。

**`docs/translation-glossary.fr/ru` 等有用吗？** 有用——i18n 术语基线，`web/src/i18n/locales/{7}.json` 依赖；未引用时可归档非删除。

**`web/src/components/ui` 60+ 感觉多？** `knip.config.ts` 已 `ignore: ['src/components/ui/**']`，为 shadcn 按需底座，非死码。

## 3. 验证

- `git ls-files --others --exclude-standard | wc -l` = 0（无未跟踪残留）
- `find -name "*.tmp|*.bak|*.swp|.DS_Store|tiktoken_cache"` = 0
- `find -perm 755 -name "*.md"` 已修为 644（`SKILL.md`）
- `.env` 真文件不存在，仅 `.env.example`（已查无密钥泄露）

## 4. 建议

1. **不动**：核心/前端/构建/文档/AI 协作 5 类
2. **可选归档**：`bin/migration_*.sql → bin/archive/` + README 注记；`electron` 若不发布桌面版则文档标注 `optional`
3. **后续**：`bun run knip` 跑一次复核 `web` 未用导出，`GOWORK=off go vet` 查 Go 死码，仅打 `// Deprecated` 不直接删

> 结论：1986 文件均为有用或兼容保留，**无需精简**；如觉 GitHub 列表视觉冗余，仅因 `web` 与 `relay` 天然文件多，可通过 GitHub 折叠或本地 `ls` 过滤查看。
