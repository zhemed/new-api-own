# evidence-frontend：技术债清扫（前端 W1/W2/W3 + 仓库 C1/C2）

- 任务板：`task-40`（Trellis 任务 `10-24-tech-debt-sweep`，PRD 见同目录 `prd.md` 的前端与一致性部分）
- 执行者：`worker-docs`；基线 HEAD `43bc237`（2026-10-24）；`VERSION=0.0.48`
- 写作用域内改动：`web/src/i18n/locales/*.json`（7 个文件，各 -2 行）+ 本证据文件
- 未改 Go 文件（后端由其他 worker 负责）；未提交、未推送；未动 `data/`；未改运行时行为

## 一、结论

| 项 | 结论 |
|---|---|
| W1 死 i18n key | 已核实两个 key 在源码 0 引用，从 7 个 locale 删除（各 -2 行，**未重排**），`i18n:sync` 已跑、混淆键完好，`i18n:check` 通过 |
| W2 oxlint 全仓 | **基线即 0 error / 0 warning**（960 文件 / 186 规则），无需修；无 warning 清单可分类 |
| W3 门禁 | `typecheck` 0、`i18n:check` 0、`bun run lint` 0；全量测试 **410 pass / 0 fail** |
| C1 版本引用 | 无"把 0.0.x 当作当前版本"的硬错误；但 README/MAINTENANCE 里大量**示例用 0.0.6 / 0.0.5** 相对 `VERSION=0.0.48` 已陈旧，建议后续统一（**只报不改**，因写作用域限定 web/） |
| C2 残留文件 | 无 `*.orig` / `*.rej` / `.DS_Store` / 备份文件；唯一命中是归档任务里**已提交**的证据日志 `verify.log`（非误入）；`web/dist`、`web/node_modules`、`.verify-scratch/` 等均已被 gitignore |

## 二、W1 死 i18n key 清理

### 核实（删除前）

```bash
grep -rn 'Model prices reset successfully' web/src --include=*.ts --include=*.tsx | grep -v 'i18n/locales'   # 0 行
grep -rn 'This will clear custom pricing ratios and revert to upstream defaults.' web/src --include=*.ts --include=*.tsx | grep -v 'i18n/locales'   # 0 行
# 7 个 locale 各命中 2 行（两个 key 各 1 行）：
for f in en zh zh-TW fr ru ja vi; do grep -c 'Model prices reset successfully\|This will clear custom pricing ratios' web/src/i18n/locales/<lang>.json; done   # 全部 2
```

上一轮（task-27）已把重置文案换成新 key，因此这两个键确为死键。

### 删除方式（遵守项目 i18n 流程）

- 用临时 `web/scripts/remove-dead-keys.mjs`：只 `delete` 这两个键，**不排序、不重排**任何既有键；
  脚本跑完已删除（`web/scripts/` 现只剩 4 个常驻脚本）。
- 随后 `bun run i18n:sync`：恢复 JSON 往返解码掉的混淆键 `footer.new\u0061pi.projectAttributionSuffix`。
  校验：`grep -c 'footer.new\\u0061pi' web/src/i18n/locales/en.json` → `1`；明文形式 0 处。

### 删除后 diff（纯删除、无重排）

```text
 web/src/i18n/locales/en.json    | 2 --
 web/src/i18n/locales/fr.json    | 2 --
 web/src/i18n/locales/ja.json    | 2 --
 web/src/i18n/locales/ru.json    | 2 --
 web/src/i18n/locales/vi.json    | 2 --
 web/src/i18n/locales/zh-TW.json | 2 --
 web/src/i18n/locales/zh.json    | 2 --
 7 files changed, 14 deletions(-)
```

删除后复核：两个 key 在 7 个 locale 中的命中数均为 `0`。

## 三、W2 oxlint 全仓体检

```bash
cd web && bunx oxlint -c .oxlintrc.json .          # 删除前
# Found 0 warnings and 0 errors.
# Finished in 461ms on 960 files with 186 rules using 16 threads.   (exit 0)

cd web && bun run lint                              # 删除后（package.json 的 lint 脚本，同一命令）
# Found 0 warnings and 0 errors.
# Finished in 218ms on 960 files with 186 rules using 16 threads.   (exit 0)
```

- **error 清零**：基线就是 0，无需修；本轮改动后复跑仍为 0。
- **warning 分类清单**：无 warning，故清单为空。若要长期守住，建议把 `bun run lint` 纳入本地收尾自检（已在 README/MAINTENANCE 的质量门禁段落中有提及）。

## 四、W3 门禁 + 回归

| 命令 | 结果 |
|---|---|
| `cd web && bun run typecheck` | `TYPECHECK_EXIT=0` |
| `cd web && bun run i18n:check` | `I18N_EXIT=0`；`4036 literal t() keys + 455 static keys = 4187 referenced keys across 7 locales. All referenced keys exist in every locale.` |
| `cd web && bunx oxlint -c .oxlintrc.json .` | `Found 0 warnings and 0 errors.`（960 文件） |
| `cd web && bun test` | **410 pass / 0 fail**（62 文件），退出码 0 |

## 五、C1 版本引用一致性（只报不改）

`VERSION` 现为 **0.0.48**。命中分三类：

**① 示例值（陈旧，建议更新；本轮不改，因写作用域限定 `web/`）**

| 位置 | 内容 |
|---|---|
| `README.md:92` | 升级命令示例「如 `0.0.6`」 |
| `README.md:111`、`README.md:121` | 回滚示例 `ghcr.io/...:0.0.5` |
| `README.md:136` | 「`0.0.6` 与 `v0.0.6` 指向同一份多架构清单」 |
| `README.md:140` | 反例命令 `docker images \| grep 0.0.5` |
| `README.md:147` | `# 预期："version":"v0.0.6"` |
| `README.md:187`、`:190-191` | 发版示例「`VERSION` 递增（0.0.3、0.0.4 …）」与 `git tag -a v0.0.3` |
| `MAINTENANCE.md:425` | 发版流程「0.0.2 → 0.0.3 → 0.0.4 …」 |
| `MAINTENANCE.md:428`、`:475` | 升级命令示例「如 `0.0.6`」 |
| `MAINTENANCE.md:491`、`:494`、`:504` | 回滚示例 `0.0.5` / `0.0.4` |
| `MAINTENANCE.md:527-529` | 版本核对表「期望」列：`"version":"v0.0.6"`、`0.0.6`、`v0.0.6` |
| `MAINTENANCE.md:553-554` | 反例 `grep 0.0.5` |
| `MAINTENANCE.md:560-561` | `docker image inspect ...:0.0.6` 示例 |

说明：这些是**命令示例里的示例 tag**，不是"当前版本"的断言；但读者容易把 `0.0.6` 当成现状。
建议统一改成 `0.0.48` 或显式占位（如 `<当前版本>`），并把"为什么用固定 tag"的说明保留。

**② 功能事实门槛（不该改）**：`README.md:124-125`（`LOG_SQL_DSN=memory` 自 v0.0.4、`LOG_MEMORY_MAX_BYTES` 自 v0.0.8）、
`README.md:127-130` 与 `:167`（受影响的 v0.0.43–v0.0.45、修复版本 v0.0.46）、
`MAINTENANCE.md:454`（`-ldflags` 缺失时版本停在 `v0.0.0`）、`MAINTENANCE.md:507-508`（同 README 的两个版本门槛）。

**③ 历史记录（不该改）**：`README.md:135`（2026-10-06 `latest` 停在 `0.0.3`）、
`MAINTENANCE.md:71`（v0.0.2 推送成功的历史）、`:466-467`（事故记录）、`:513`、`:543`。

**其它包内的版本号**：`web/package.json` 的 `"version": "1.0.0"` 是**前端包自身版本**，与产品 `VERSION` 无关，不算不一致；
`web/src` 内硬编码 `0.0.x` 只出现在 `version-compare.test.ts` 的**测试夹具**（用于验证 semver 比较），属刻意保留；
`web/rsbuild.config.ts:114` 的 `0.0.0.0` 是监听地址，不是版本号；`.env.example` 内无产品版本号（`0.0.x` 命中全是 IP/DSN）。

## 六、C2 残留文件（只报不改）

```bash
find . \( -name '*.orig' -o -name '*.rej' -o -name '.DS_Store' -o -name '*~' -o -name '*.bak' \
  -o -name '*.tmp' -o -name '*.swp' -o -name 'npm-debug.log*' -o -name '*.log' \) \
  -not -path './web/node_modules/*' -not -path './.git/*' -not -path './data/*'
```

- 命中 **1 个**：`.trellis/tasks/archive/2026-10/10-06-team-review-backend/verify.log`（944 字节）。
  经 `git ls-files --error-unmatch` 确认**已提交**，是那次任务归档时留下的验证日志证据，**不是**误入的临时文件；
  建议保留（若要清理需另行决定，属证据文件）。
- **无** `*.orig` / `*.rej` / `.DS_Store` / `*~` / `*.bak` / `*.swp` / 调试日志。
- 被 git 忽略的本地产物（非债务）：`web/dist/`（56M 构建产物）、`web/node_modules/`、`data/`（实例数据）、
  `.trellis/.runtime/`、`.trellis/.developer`、`.trellis/scripts/common/__pycache__/`、
  `.verify-scratch/`（历史截图，`.gitignore:46` 用 `.verify-*/` 规则忽略）。

## 七、风险与未完成

- 本轮仅删死键，未改任何运行时文案；`bun test` 410 项全过，说明无测试依赖被删的 key。
- C1 的示例值陈旧问题**未处理**（写作用域只到 `web/`）；若 Lead 需要，我可另开一次文档改动统一更新。
- `verify.log` 与 `.verify-scratch/` 属历史证据/截图，未处理（只报不改是任务要求）。
- `i18n:sync` 会重写文件，已确认混淆键转义完好；若后续有脚本绕过 sync 手改 locale，需重新跑 sync。

## 八、建议下一步

1. C1：把 README/MAINTENANCE 里**示例值**统一到 0.0.48 或占位符（保留功能门槛与事故记录不动），一次文档改动即可。
2. 决定 `verify.log` 与 `.verify-scratch/` 的去留（建议保留前者作为归档证据，后者可清）。
3. 把 `bun run lint` 与 `bun run i18n:check` 纳入固定收尾自检（当前都是干净的，适合设门禁）。
