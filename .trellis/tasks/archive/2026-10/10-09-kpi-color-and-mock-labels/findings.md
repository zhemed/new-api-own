# KPI 阈值上色 + mock 区块标注 — 交付

## 一、数据来源核对（先核对再动手，这是本次最关键的结论）

| 位置 | 实际数据来源 | 判定 |
|---|---|---|
| `model-details-charts.tsx`（延迟 / 吞吐趋势图） | 只 import **类型**；数据由 `model-details-performance.tsx:90,112` 的 `toLatencySeries` / `toUptimeSeries` 从 `PerformanceGroup[]` 变换而来 | **真实数据** |
| `model-details-uptime-sparkline.tsx`（30 天可用率） | 只 import `aggregateUptime`（对**传入序列**的真实聚合）+ 类型 | **真实数据** |
| `model-details-api.tsx` → `SupportedParametersSection` | `buildSupportedParameters(model)`（`mock-stats.ts:789`）——按模态硬编码的参数表 | **MOCK** |
| `model-details-api.tsx` → `RateLimitsSection` | `buildRateLimits(model)`（`mock-stats.ts:808`）——按 `model_name` 哈希生成 | **MOCK** |
| `model-details-api.tsx` 其余（Code samples / Auth） | `props.model`（真实定价数据）+ `useStatus()` | **真实数据** |

`PerformanceGroup[]` 来自 `getPerfMetrics()` → `GET /api/perf-metrics`（`src/features/performance-metrics/api.ts:36`），**没有任何 mock 回落**。

### 额外发现：四个 mock 生成器是死代码

`grep buildLatencyTimeSeries|buildUptimeSeries|buildGroupPerformance|buildAppRankings`（排除 `mock-stats.ts` 自身）= **0 处调用**。
所以任务描述里"图表 / 可用率区块消费 mock"的前提**不成立**——这两个组件拿到的是真实指标。
按"只标注真正用 mock 的块、真实字段不要误标"的要求，**未给它们加标注**。

真正的 mock 使用点只有两处（都在 API 页签内）：`buildSupportedParameters` → 参数表、`buildRateLimits` → 速率限制表。

## 二、改动清单

**① KPI 上色（每文件 +2 行）**
- `src/features/dashboard/components/models/performance-overview.tsx`
  - import 增加 `getTpsTextClass`
  - Throughput 的 `InlineMetric` 增加 `valueClassName={getTpsTextClass(summary.avgTps)}`
- `src/features/dashboard/components/overview/performance-health-panel.tsx`
  - 同上（`MetricCell`）

两个组件**本来就支持 `valueClassName`**（Success rate 已用同样方式），所以不需要新增 prop 或包 span。
未新增第二套阈值/配色：仓库内 TPS 阈值只有 `performance-metrics/lib/format.ts` 一处（50/100，左闭右开，`unknown` 中性灰）。

**② mock 标注（只标真 mock 的两张表）**
- `src/features/pricing/components/model-details-api.tsx`
  - 新增局部组件 `SampleDataNote()`：小徽标 `Sample data` + 小字说明，`data-testid='sample-data-note'`
  - 仅在 `SupportedParametersSection` 与 `RateLimitsSection` 的标题下各插入一行
  - 组件上方写了注释说明"为什么只标这两块"（避免后续被误加到真实区块）

**③ i18n**：2 个新键 × 7 语言 = 14 条，走临时 `scripts/add-missing-keys.mjs`（已删除）+ `node scripts/sync-i18n.mjs`
- `Sample data` → 示例数据 / 範例資料 / Données d'exemple / サンプルデータ / Примерные данные / Dữ liệu mẫu
- `Not measured on this deployment — shown for reference only.` → 本机未实测，仅供参考。/ …

**④ 测试（新增 11 个）**
- `src/features/dashboard/__tests__/throughput-kpi-color.test.tsx`（8）
  两个组件 × 4 档：`20 → text-red-600`、`75 → text-amber-600`、`120 → text-emerald-600`、
  无测量（`avg_tps=0` → 显示 `—`）→ `text-muted-foreground`。断言的是**渲染出来的类名**。
  （阈值函数自身的边界 49.9/50/99.9/100/0/-1/NaN/+Inf 已由既有 `performance-metrics/lib/__tests__/tps-color.test.ts` 覆盖，未重复造。）
- `src/features/pricing/components/__tests__/sample-data-annotation.test.tsx`（3）
  1. 标注**恰好 2 处**（该页签共 4 个 section，计数即契约：漏标/误标都会改变它）；
  2. 两处分别落在 `Supported parameters` 与 `Rate limits` 两个 section 内；
  3. 文案是"示例数据 + 本机未实测"，且**遍历全部 section 断言真实区块无该标注**。

## 三、六条命令 + format:check

| # | 命令 | 结果 |
|---|---|---|
| 1 | `bun run typecheck` | ✅ 0 错误 |
| 2 | `bun run lint` | ✅ 0 error（2 warnings） |
| 3 | `bun test` | ✅ **224 pass / 0 fail**（42 files）——基线 213，本次 +11，无下降 |
| 4 | `bun run build` | ✅ exit 0（Total 57331.5 kB / gzip 16541.2 kB） |
| 5 | `bun run knip` | ⚠️ exit 1 —— **基线即红**；唯一条目涉及我文件的是 **既有的** `ApiTabIcon` 未使用导出（`HEAD` 中已存在、我未触碰该行） |
| 6 | `bun run i18n:check` | ✅ 4030 literal + 455 static = 4181 键，七语言齐备 |
| 附 | `bun run format:check` | ✅ exit 0（修复前只列出我这 3 个文件，格式化后通过） |

## 四、边界

- 本次仅改：`performance-overview.tsx`、`performance-health-panel.tsx`（①点名）、`model-details-api.tsx`（②范围）、7 个 locale、2 个新测试目录。
- 工作区里 `prompt-input.tsx` / `footer.tsx` / `dashboard/lib/*` / `amount-*.tsx` / `integrations/*` 等改动**属于 `fix-frontend-open-items`**，我未触碰。
- 这两个 KPI 文件**不在 lint 命中清单**内（已核对），与 teammate 无写冲突。
- 未 commit/push；未碰用户实例。
