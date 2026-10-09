# 评估：模型速度数值按阈值上色

## 用户诉求（2026-10-09）

截图里速度列显示 `流 167 t/s`；旁边等级标签 `max`（绿）/`high`（黄）已带颜色。
用户希望 **速度数字本身也带颜色**：`<50` 红、`50–100` 黄、`>100` 绿，并强调**这只是大概的阈值**。
明确要求：**先评估，不实现**。

## 评估要点

1. **定位**：该数值由哪个组件渲染、数据来自哪个接口/统计；
2. **配色一致性**：复用 `max`/`high` 标签现有的颜色令牌（不硬编码 hex），暗色模式可读；
3. **可访问性**：颜色不能是唯一信息载体（色觉障碍），需考虑提示/图标或保留原位数文本；
4. **阈值方案**：固定阈值（50/100）vs 相对分位（同表内 min–max / 四分位）——t/s 数值与模型大小强相关，
   固定阈值可能大面积落在红/黄；
5. **数据依赖**：该统计是否依赖用量日志（本实例已切内存日志 → 重启后历史归零会否影响显示/排序）；
6. **工作量**：改动面（组件数、是否只影响一个页面）、测试点、是否需要 i18n 文案。

## 边界

- 本轮**只评估**：不改代码、不改数据、不改实例；
- 结论需给出可选方案与推荐，由用户决定后再实现。

## 评估结论（2026-10-09，只读调查）

### 关键事实：**该数值目前是 mock 假数据**

`web/src/features/pricing/lib/mock-stats.ts` 文件头自述：
> The backend has not yet implemented latency / uptime / app-ranking data. These helpers generate
> plausible, **deterministic mock values seeded from the model name** (and optionally the group name)…
> When the backend ships real metrics, callers should switch to the real API and these helpers can be deleted.

- 数据结构：`GroupPerformance { group, ttft_p50_ms, ttft_p95_ms, ttft_p99_ms, throughput_tps, uptime_30d_pct, request_volume_24h }`；
- 渲染处：`web/src/features/pricing/components/model-details-charts.tsx:382`（`${tps.toFixed(1)} t/s`）；
- 后端确认**无**吞吐/延迟接口（`controller/ service/ model/` 无 throughput/tps 实现）。

→ **给这些数字上色不是"给指标上色"，而是"给假数据上色"**：颜色会传达"经过测量、可信"的语义，
反而比现在更危险。

### 技术上能不能做？能，而且很简单

- 复用项目既有语义色即可：`text-emerald-600 dark:text-emerald-400`（好）、`text-amber-600 dark:text-amber-400`（注意）、
  `text-red-600 dark:text-red-400`（差）——与 `max`(绿)/`high`(黄) 同一套，暗色模式已适配；
- 工作量：一个纯函数（阈值判定）+ 一处渲染改动 + 单测 ≈ 很小。

### 阈值方案（用户给的 50/100 只是示意）

- **固定阈值不推荐**：t/s 与模型规模强相关（大模型 20 t/s 属正常、小模型 200 t/s 也正常），
  固定线会让绝大多数模型常年落在红/黄，失去区分度；
- **推荐相对分位**：同表内 min–max 归一或四分位（自校准、无需维护阈值）；
- 若坚持固定线：把它做成**命名常量集中在一处**（便于调参）并注明"粗略"；颜色必须与数字同现（避免只靠颜色表意）。

### 真正该先做的一步（推荐）

本项目的 `logs` 表**已经有真实可算的吞吐原料**：`use_time`(秒)、`completion_tokens`、`is_stream`、
`model_name`、`channel_id` → 可按模型/分组聚合出 `sum(completion_tokens)/sum(use_time)`，
正好填进 mock 里已经定义好的 `throughput_tps` 契约（该文件注释也说明"形状按预期真实接口设计"）。
链路：后端聚合接口 → 前端把 mock 换成真实数据 → **再上色**。
注意：本实例已切内存日志，统计窗口受"重启即丢 + 200MB/7 天"限制，需在 UI 上标注统计口径。

### 工作量

| 方案 | 内容 | 量级 |
|---|---|---|
| 只上色（仍用 mock） | 阈值函数 + 渲染 + 测试 | 小，但**不建议** |
| 真实吞吐 + 上色（推荐） | 后端聚合接口 + 前端替换 + 上色 + 测试 | 中（约 1–2 轮） |
| 先不做 | 保持现状 | 0 |

## 重要更正（Lead 自我修正，2026-10-09）

先前结论「那列速度是 mock 假数据」**不完整、对截图那一列是错的**。实测澄清：

### 截图那一列是**真实测量**

- 真实指标子系统：`pkg/perf_metrics`（`metrics.go` / `flush.go` / `types.go`）；
- 采集点：`service/quota.go:382`、`service/text_quota.go:541` → `perfmetrics.RecordRelaySample(relayInfo, ok, completionTokens)`
  —— 每条真实请求采集 **延迟 / TTFT / 输出 tokens / 成功与否**；
- 聚合与**落库**：时间桶 + Redis 计数 + `perf_metrics` 表（`model/perf_metric.go` 的 upsert）→ **重启不丢**；
- 配置项：`setting/perf_metrics_setting`（Enabled / FlushIntervalMinutes / RetentionDays）；
- 接口：`GET /api/perf-metrics`、`GET /api/perf-metrics/summary`（`controller/perf_metrics.go`）；
- 使用方：模型详情页**性能区块**（截图那列 `avg_tps`）、模型广场卡片（`getPerfMetricsSummary`）、控制台性能总览/健康面板。

### mock 只覆盖同一页面的**其他区块**

`web/src/features/pricing/lib/mock-stats.ts`（按**模型名**哈希生成，`PROFILE_BY_NAME` + `seed.ts` 的 djb2+LCG）：

- 用到它的组件：`model-details-charts.tsx`（延迟/吞吐图表）、`model-details-uptime-sparkline.tsx`（30 天可用率）、
  `model-details-api.tsx`；
- 实跑证据（真实代码）：`deepseek-v3` → default 85.7 t/s、vip 72.1 t/s（重复调用完全相同）；
  把名字改成 `deepseek v3`（多一个空格）→ 81.7 / 122.3 —— 真实测速不可能因改名而变，故为生成值；
- 该文件头部自述「后端尚未提供 latency/uptime/app-ranking 数据，等真实接口上线后应切换并删除这些 helper」；
- 来历：`1092e46 chore: 自维护基线（基于 2026-08-01 代码基）`——来自代码基线，不是本轮引入。

### 对上色的影响

- 对**真实的 `avg_tps`** 上色**有意义**（用户原始想法成立）；阈值仍建议相对分位；
- 对 mock 的图表/可用率**不要**上色——先接真实数据，或至少显式标注"示例数据"（当前界面无任何标注）。
