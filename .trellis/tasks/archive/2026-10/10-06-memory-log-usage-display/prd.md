# 前端：内存日志占用/上限只读展示

## Goal

后端已在 `GET /api/performance/logs` 增加只读字段 `memory_log_bytes` / `memory_log_max_bytes` / `memory_log_rows`，
前端在「系统设置 → 系统维护 → 日志设置」区块补一行**只读**展示，让管理员看到 DB/内存日志的**载荷占用与预算**。

## Requirements

### R1 展示内容
- 形如：`内存日志占用 12.3 MB / 上限 200 MB（176,000 行）`。
- 复用区块内既有的 `formatBytes()`；行数按当前语言做千分位格式化。

### R2 四态
| 状态 | 条件 | 展示 |
|---|---|---|
| 未设上限 | `memory_log_max_bytes = 0` | `内存日志占用 X（未设置上限）` —— **不显示 0/0** |
| 正常 | `0 < usage < 90% × limit` | `内存日志占用 X / 上限 Y（N 行）` |
| 接近上限 | `usage ≥ 90% × limit` | 同上 + 轻微提示（琥珀色）+ `接近上限` 文案 |
| 字段缺失/接口失败 | 字段不存在（旧后端）或响应缺失 | **不渲染该行**（沿用磁盘区块既有行为，不新增错误 UI） |

### R3 硬约束
- **不新增任何外呼**：只用既有 `GET /api/performance/logs`（该端点已回填字段，且 `log_dir` 为空时也回填）。
- **不得出现"自动清理"字样**：后端在字节超限时自行裁剪，面板只做展示。
- 新文案必须走临时 `scripts/add-missing-keys.mjs` + `node scripts/sync-i18n.mjs`，**禁止手改 locale JSON**；七语言齐全。

### R4 回归测试
- 为四态补组件级回归测试（happy-dom + `node:test`，与既有组件测试同风格），断言可见文案与提示态。

### R5 质量门
- `typecheck / lint / test / build / knip / i18n:check` 全绿，另跑 `format:check`；`knip` 基线红照旧说明。

## Constraints

- 只改 `web/**`；不 commit / push；不碰用户 3000 实例；不新增依赖。
- 展示位置：磁盘日志文件那块**附近**；因后端在 `log_dir` 为空时同样回填，该行应与 `enabled` **无关**地展示（数据存在即显示）。

## Acceptance Criteria

- [ ] AC1：四态文案均可复现，且"未设上限"不显示 0/0。
- [ ] AC2：全仓 `web/src` 无新增外呼（`grep api.github.com` 等仍为 0；未新增任何 `fetch`/端点）。
- [ ] AC3：新增文案七语言齐备，`i18n:check` 通过；无"自动清理/automatic cleanup"字样。
- [ ] AC4：四态回归测试通过。
- [ ] AC5：六条命令 + `format:check` 输出齐备；改动仅 `web/**`；未 commit/push。

## Notes

- 结论与命令输出留 `findings.md`。
- 上游：`.trellis/tasks/10-06-log-byte-budget-write-trigger/prd.md`（后端机制）。
