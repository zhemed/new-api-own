# 日志设置区块 vs 新默认值 — 核对结论

**结论：`log-settings-section.tsx` 是纯手动入口，全仓没有任何"自动每 5 分钟清理"的文案或状态展示 → 本次未做任何改动。**
（工作区对 `web/` 无未提交变更；改动仅在确认"需要改"时才做，此处不满足条件。）

## 一、区块现状（证据到文件:行）

### 1. 手动按时间点清理 DB 日志（就是 Lead 说的那条）
- 调用：`POST /api/system-task/log-cleanup?target_timestamp=<ts>` — `src/features/system-settings/api.ts:52-59`
- UI：`log-settings-section.tsx:372-400`（标题 **Clean history logs** + 日期选择器 + 快捷选项 + 红色「Clean logs」按钮）
  + 二次确认框 `:584-613`（"This will permanently remove all log entries created before {{date}}."）
- 这是**清楚的手动动作**：不选时间点会报错（`:267-274`），确认后才发起。

### 2. 只展示"当前手动任务的进度"，不是自动调度
- 挂载时查一次：`GET /api/system-task/current?type=log_cleanup` — `api.ts:63-71`，用于 `log-settings-section.tsx:184-203`
- 任务运行中每 **1 秒**轮询 `GET /api/system-task/{taskId}` — `api.ts:73-78`，用于 `:225-257`
- ⚠️ 全文件唯一的 `setInterval` 就是这个 **1 秒进度轮询**（`:229`），**与清理周期无关**；不会被读成"每 5 分钟自动清理"。

### 3. 磁盘日志文件（另一件事，与字节预算不同对象）
- 只读展示 `log_dir / file_count / total_size / 日期范围`：`GET /api/performance/logs` — `log-settings-section.tsx:167-174`、渲染 `:442-477`
- 手动清理：`DELETE /api/performance/logs?mode=<by_count|by_days>&value=<n>` — `log-settings-section.tsx:315-317`，二次确认 `:524-570`

### 4. 与清理无关的开关
- `LogConsumeEnabled`（记录配额用量）— schema `:81-83`、提交 `:259-265`、UI `:346-368`

## 二、为什么判定"不需要改"

| 检查 | 结果 |
|---|---|
| 任何"自动清理"状态展示 | **无** |
| `grep -iE "automatically\|auto clean\|every [0-9]+ minute\|cleanup interval\|LOG_CLEANUP\|定时"`（限日志上下文） | **0 条**（命中的都是 OAuth/worker/缓存等无关项） |
| i18n 中 `automatic\|interval` × `clean\|log` 的组合键 | **0 个** |
| `grep -iE "LOG_MEMORY_MAX_BYTES\|LOG_MEMORY_MAX_ROWS\|LOG_CLEANUP_INTERVAL"` | 前端**完全未出现**（env-only，所以没有"旧默认值"需要同步） |
| 唯一擦边句 `:437`「Log files accumulate over time; regular cleanup is recommended to free disk space.」 | 是**建议人工**定期清理**磁盘日志文件**，不是声称系统定时清理；且它讲的是 log 目录文件，而新机制约束的是 DB 日志**载荷字节** → 在新默认下**依然成立**，不需改 |

补充：`system-info/components/system-tasks-panel.tsx:83` 的 `log_cleanup: 'Log cleanup'` 只是系统任务列表里的**类型标签**（展示手动任务的执行记录），同样不构成调度声明。

## 三、"内存日志占用 / 上限"只读展示 — 可行性（本轮未实现）

- **现状可用的数据不含该项**：`/api/performance/logs` 只返回**磁盘日志文件**信息
  （`controller/performance.go:54-56` 的 `file_count` / `total_size`）。
- 后端**没有**暴露 DB/内存日志载荷字节数与预算：`grep -iE "MaxBytes|memoryLogBytes|logBytes|CurrentBytes"` 与 log 相关 = **0**。
- 因此要做必须先加后端字段，最小侵入方案：在既有 `/api/performance/logs`（或 `/api/status`）响应里增加
  `memory_log_bytes`（当前载荷）与 `memory_log_max_bytes`（预算，未配置时 0/缺省 → 界面显示"未启用"）。
- **不需要任何新的浏览器外呼**（复用现有管理端接口，页面上已有调用点）。
- 按指示**本轮不实现**；待后端字段落地后，可在 `Server Log Management` 区块旁加一行只读文案（约 10 行改动）。

## 四、六条命令（未改代码，全量复跑）

| # | 命令 | 结果 |
|---|---|---|
| 1 | `bun run typecheck` | ✅ 0 错误 |
| 2 | `bun run lint` | ✅ 0 error（21 warning 既有基线） |
| 3 | `bun test` | ✅ 202 pass / 0 fail（37 files） |
| 4 | `bun run build` | ✅ exit 0（Total 57326.1 kB / gzip 16539.2 kB） |
| 5 | `bun run knip` | ⚠️ exit 1 —— **基线即红**（此前用 `git archive HEAD` 纯净对照证实）；本次 0 处提及本 feature |
| 6 | `bun run i18n:check` | ✅ 4025 literal + 455 static = 4176 键，七语言齐备 |
| 附 | `format:check` | ✅ exit 0（1030 files） |

## 五、边界

- 本轮**未触碰任何文件**；`git status --porcelain -- web/` 为空（工作区已由 Lead 提交，我的更新面板文件已在 HEAD 中）。
- 未 commit/push；未碰用户实例（3000）。
