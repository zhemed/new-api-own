# 内存日志占用/上限只读展示 — 交付

## 展示文案（四态）

| 状态 | 触发条件 | 展示 |
|---|---|---|
| **未设上限** | `memory_log_max_bytes = 0` | `内存日志占用 12 MB（未设置上限）` —— 明确**不显示 0/0** |
| **正常** | `0 < usage < 90% × limit` | `内存日志占用 12 MB / 上限 200 MB（176,000 行）` |
| **接近上限** | `usage ≥ 90% × limit` | 同"正常" + 琥珀色提示 `接近上限`（`text-amber-600`，浅提示，非报错） |
| **字段缺失 / 接口失败** | 字段不存在（旧后端）或响应缺失 | **该行不渲染**（沿用磁盘区块既有行为，不新增错误 UI） |

英文原文（en 即 key）：
- `Memory log usage {{usage}} / limit {{limit}} ({{rows}} rows)`
- `Memory log usage {{usage}} (no limit configured)`
- `Approaching the limit`

**全篇无"自动清理 / automatic"字样**（测试里也断言了 `'automatically'` 不出现）——后端在字节超限时自行裁剪，面板只展示。

## 实现

`web/src/features/system-settings/maintenance/log-settings-section.tsx`
- `ServerLogInfo` 增加可选字段 `memory_log_bytes` / `memory_log_max_bytes` / `memory_log_rows`。
- 新增常量 `MEMORY_LOG_NEAR_LIMIT_RATIO = 0.9`。
- 组件内计算文案（if/else，无嵌套三元），复用区块里既有的 `formatBytes()`；行数用
  `memory_log_rows.toLocaleString(i18n.language)` 按当前语言格式化。
- 渲染一行 `data-testid='memory-log-usage'`（接近上限时再加 `data-testid='memory-log-near-limit'`），
  位置在「Server Log Management」标题与磁盘信息块之间。
- **与 `enabled` 无关**：后端在 `log_dir` 为空时同样回填这三个字段（`controller/performance.go:242-249`），
  所以该行只要**有数据就显示**，不依赖是否配置了磁盘日志目录（有测试覆盖）。

**未新增任何外呼**：继续用页面已有的 `GET /api/performance/logs`（`log-settings-section.tsx:169` 的
`fetchServerLogInfo`，挂载时一次）。`web/src` 内 `api.github.com` = 0，本文件无 `fetch(` 调用。

## 回归测试（新增 5 个）

`__tests__/log-settings-memory-usage.test.tsx`（happy-dom + `node:test`，与既有组件测试同风格）：
1. 正常态：`Memory log usage 12 MB / limit 200 MB (176,000 rows)`，且无接近上限徽标；
2. 未设上限：`Memory log usage 12 MB (no limit configured)`，且断言**不含** `0 Bytes`；
3. 接近上限（190/200 MB）：数字正确 + 徽标文案 = `Approaching the limit` + 全页不含 `automatically`；
4. 字段缺失（旧后端）：该行**不渲染**（`data-testid` 为 null）；
5. `enabled=false`（未配磁盘日志目录）：仍展示载荷占用（证明与磁盘块解耦）。

`bun test`：**207 pass / 0 fail**（基线 202，本次 +5）；act 警告 0。

## i18n

3 个新键 × 7 语言 = 21 条，经临时 `scripts/add-missing-keys.mjs`（自建、**已删除**）+ `node scripts/sync-i18n.mjs`，
**未手改 locale JSON**。

```
i18n check: 4028 literal t() keys + 455 static keys = 4179 referenced keys across 7 locales.
All referenced keys exist in every locale.
```

## 六条命令 + format:check

| # | 命令 | 结果 |
|---|---|---|
| 1 | `bun run typecheck` | ✅ 0 错误 |
| 2 | `bun run lint` | ✅ 0 error（21 warning 既有基线） |
| 3 | `bun test` | ✅ **207 pass / 0 fail**（38 files） |
| 4 | `bun run build` | ✅ exit 0（Total 57329.0 kB / gzip 16540.3 kB） |
| 5 | `bun run knip` | ⚠️ exit 1 —— **基线即红**（此前 `git archive HEAD` 纯净对照已证）；本次 0 处提及本文件 |
| 6 | `bun run i18n:check` | ✅ 4179 键七语言齐备 |
| 附 | `bun run format:check` | ✅ exit 0（1031 files） |

## 改动清单（仅 `web/**`）

- `M web/src/features/system-settings/maintenance/log-settings-section.tsx`
- `?? web/src/features/system-settings/maintenance/__tests__/log-settings-memory-usage.test.tsx`（新增）
- `M web/src/i18n/locales/{en,fr,ja,ru,vi,zh-TW,zh}.json`（脚本生成）

未 commit/push；未碰用户 3000 实例。
（工作区里 `.env.example` / `MAINTENANCE.md` / `README.md` / `common/env.go` / `controller/performance.go`
的改动属其他成员，非本次所为。）
