# 评估：模型分析「消耗分布」改为按 Token 统计

## 用户指示（2026-10-18）

「验收成功，不过这个消耗发布，能不能换成统计 Token，评估一下」

## 本轮范围（只读评估，不改代码）

现状：模型分析页「消耗分布」按**费用（配额换算 USD）**统计，柱状/面积两种视图，数据来自 dashboard 模型分析接口。

需要回答：

1. **数据面**：接口返回的行里是否已有 token 字段（如 `token_used` / `prompt_tokens` / `completion_tokens`）？
   - 有 → 纯前端改动（换值字段 + 文案/i18n + tooltip + 总计）；
   - 没有 → 需改后端（`controller`/`model` 聚合口径），评估影响面；
2. **语义取舍**：
   - 费用口径 = 价值归一（能横向比较不同模型的成本）；
   - Token 口径 = 体量口径（缓存命中/推理 token 混算，不同模型不可比价值）；
   - 是否要区分 prompt/completion 或总量；
3. **方案**：替换 / 增加"口径切换（费用｜Token）"/ 两个图表并存，各自改动量与风险。

## Acceptance Criteria

- [ ] 给出数据字段证据（文件行号 + 接口样例）
- [ ] 给出前端/后端改动量与风险分级
- [ ] 给出 2–3 个方案与推荐项（提问组件让用户选）

## 评估结论（2026-10-18，只读）

### 数据面：后端**已经**提供 token 聚合 ✓

- `model/usedata.go:145/156/166`：`Select(... sum(count) as count, sum(quota) as quota, sum(token_used) as token_used)`；
- 前端类型已有字段：`web/src/features/dashboard/types.ts` 的 `QuotaDataItem.token_used?: number`；
- **实测**（`/api/data/?start..&default_time=day`，30 天）：4 行样本形如
  `{model_name:"deepseek-flash", created_at:1791331200, token_used:47, count:1, quota:125}`；
  合计 `quota=575064`、`token_used=1682465`、`count=48`。
- ⇒ **纯前端改动，无需动 Go**。

### 改动点（都在 `web/src/features/dashboard/lib/charts.ts` + 文案）

- `:323 quotaPerUnit`、`:438-446`（`rawQuota → usd → Usage`）是费用换算处 → 改为按口径取值；
- `renderQuotaCompat()` 已有 `meta.kind === 'tokens'` 分支（`:58`，K/M 千分位格式）→ token 格式化能力已现成；
- 标题「消耗分布」+ 总计 + tooltip 文案 → i18n 7 语言；
- **先例**：flow 图已有 `FlowMetric = 'quota' | 'tokens' | 'requests'` 的口径切换，交互可复用。

### 语义取舍

| 口径 | 含义 | 优点 | 局限 |
|---|---|---|---|
| 费用（现状）| 配额换算 USD | 跨模型可比成本 ✓ | 看不到体量 |
| Token | `sum(token_used)` | 直观反映吞吐/缓存命中 ✓ | 输入+输出+缓存混算，不可比价值 ✗ |

### 方案

- **A（推荐）**：加口径切换「费用 ｜ Tokens」（复用 flow 图先例，非破坏性）；
- **B**：直接替换为 Token 口径（用户原话"换成统计Token"，费用视图消失）；
- **C**：两图并存（占空间，不建议）。

风险均低：不改后端，`token_used` 缺失时回落 0。

## 执行结果（2026-10-18，v0.0.21）

用户裁定：**标题「消耗分布」不动**，只把数值口径（总计 / y 轴 / tooltip）从金额换成 Token。

### 改动（仅 `dashboard/lib/charts.ts` + 1 个新测试）

- 柱状/面积图的 `Usage` 改取 **`token_used`**（不再 `/quotaPerUnit` 转 USD）；
- **Top-Model 选取改按 token 排名**（原按 quota，改成 token 口径后会出现"小 token 模型占大色块"）；
- 新增 `formatTokenValue`（复用共享 `formatTokens`，`0 → '0'`，不读货币设置），应用到 **y 轴刻度 + tooltip（单行/合计/溢出桶）+ 总计**；
- 总计 = `token_used` 求和；「总计：」前缀与标题、视图切换、饼图/趋势/排行/用户图表**均未动**；
- `token_used` 缺失 → 0（图表不变空不报错）；新增 3 个测试，`bun test` 283 → **286**。

### 真机验证（v0.0.21）

`/dashboard/models` 截图 `.verify-scratch/v0021-token-dist.png`：
标题「消耗分布」不变 ✓；**总计：526.9K** ✓；y 轴 `0 / 200.0K / 400.0K / 600.0K` ✓；**全图无 `$`** ✓；
x 轴 `10-18 00:00 … 14:00`（00:00→now）✓；无报错、版本 v0.0.21 ✓。

### 备注

- 数据面本来就有 `token_used`（后端 `sum(token_used)`），**全程零 Go 改动**；
- 口径差异（费用=价值归一 vs Token=体量，含输入/输出/缓存混算）已在评估中提示用户，用户选择换口径。
