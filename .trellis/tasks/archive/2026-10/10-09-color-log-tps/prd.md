# 使用日志表：流式 t/s 按阈值上色

## 用户指示（2026-10-09）

贴出「通用日志」表格截图：列为 时间/渠道/用户/令牌/模型/**推理强度**/**流**/Tokens/费用/耗时/详情，
其中「流」列显示 `流` + `24 t/s`（另一行 `12 t/s`）。结合上一轮「173 t/s 我就想让它有颜色」，
目标是**这张表的 t/s 数值**按阈值上色。

## 现状差距

上一轮已把颜色加到：模型详情页性能表 TPS 列、详情页 TPS 概览卡/头部指标、模型广场卡片紧凑吞吐。
**使用日志表（本截图）未覆盖** —— 需补齐，复用同一 `getTpsTextClass()`（阈值 50/100，左闭右开，无数据中性灰）。

## 要做

1. 定位使用日志表的「流」列定义与数值来源（每请求的 tokens/耗时换算），确认 t/s 的计算口径；
2. 复用 `getTpsTextClass()` 上色（不新增阈值、不复制配色）；
3. 补回归测试（边界与无数据行为）；
4. 六条命令全绿；随后按既有流程发版（v0.0.11）并更新用户实例。

## Acceptance Criteria

- [ ] 使用日志表的 t/s 按 <50 红 / 50–100 黄 / ≥100 绿 显示，无数据不染色
- [ ] 不引入第二套阈值或配色
- [ ] 测试覆盖边界；六条命令全绿
- [ ] 发版并在实例上肉眼验证

## 执行结果（2026-10-09）

### 改动

| 文件 | 内容 |
|---|---|
| `web/src/features/usage-logs/components/timing-metrics-cell.tsx` | `StreamTpsCell` 的 t/s 文本改为 `getTpsTextClass()` 上色；**无数据时保留原 `text-muted-foreground/60`**（不显示颜色）|
| `web/src/features/pricing/components/model-perf-badge.tsx` | **修复 v0.0.10 引入的重复 import**（`no-duplicate-imports` ×2）——上一轮我误判为"既有文件问题"，实为我引入 |
| `web/src/features/usage-logs/components/__tests__/stream-tps-cell.test.tsx`（新） | 2 例：20/75/150 分别红/黄/绿；`null` 保持中性灰且文案为 `—` |

上色点合计 **5 处**，全部走同一个 `getTpsTextClass()`（阈值 50/100，左闭右开）：使用日志表「流」列、详情页性能表 TPS 列、
详情页 TPS 概览卡与头部指标、模型广场卡片紧凑吞吐。

### 门禁

`bun test` **213 pass / 0 fail**；typecheck 0；**lint 21 warnings / 0 errors**（修复后）；
build / i18n:check / format:check / knip(files) 全绿；`format` 后测试文件因版本变更按流程重读再改（FS_STALE_VERSION 流程）。

### 发版

**v0.0.11** 发布成功：镜像 `0.0.11`/`v0.0.11`/`latest` 同一摘要 `sha256:abcb2404805da`；
Release 三资产齐备；`releases/latest = v0.0.11`。

### 自我更正（留痕）

- 上一轮我把 lint 的 2 个 error 归因于既有文件 `use-system-config.ts` —— **错误**，实际是我在 `model-perf-badge.tsx`
  写了同一模块的两行 import；本轮已修并复核为 0 error。
- 上一轮上色范围漏了用户真正指的**使用日志表**（截图列），已在本轮补齐。
