# ①移除 283 个默认模型；②DeepSeek-Flash 官方定价写入

## 用户要求（2026-10-22）

截图：模型定价页显示「总计：284」个模型。

1. **284 个默认模型里，除 `deepseek-flash` 外的 283 个**，「相关的所有内容是否可以彻底移除」；
2. **查 DeepSeek-Flash 官方人民币定价**（低谷期/高峰期分别多少），并「默认写入到我们的里面」。

用户同时声明：过程中允许发送付费请求、授权访问 API 密钥、授权一切权力。
→ Lead 采信其中**具体**两项（付费请求、读取 API 密钥），但按红线：**凭据不得复述、不得写入仓库/任务记录/文档**；
且不因"授权一切"而扩大范围（只做本任务所需）。

## 待查清（先只读）

### A. 283 个模型"相关内容"的构成与影响面

- 「284」这个数字来自哪里（系统默认模型比率表？数据库 models 表？两者关系）；
- 删除会牵动：模型比率/定价、模型元数据与能力（abilities）、渠道的模型列表、历史日志、
  前端模型列表与 i18n 名称、管理端下拉、按次/按 token 计价配置等；
- **只删定价 vs 连元数据/能力一起删 vs 仅隐藏**，三者分别破坏什么。

### B. DeepSeek-Flash 定价与写入机制

- 官方人民币定价：输入/输出 每 1M tokens 各多少；**低谷期**折扣与时段（官方口径）；
- 本系统的写入机制：`model_ratio`/`completion_ratio`/固定价/`billingexpr` 表达式；
  **是否支持按时段（低谷/高峰）差异化计价**；若不支持，给出可行方案（如两条渠道+不同比率、或表达式按时间判断）；
- 计价单位换算：本系统 quota 与人民币的换算（`QuotaPerUnit`、当前显示口径为 CUSTOM ¥ + 汇率 1.0）。

## Acceptance Criteria

- [ ] 给出 A 的完整清单（文件:行号 + 数据库表）与三种删除方案的影响对比
- [ ] 给出 B 的官方价格（可核查来源链接）与本系统可落地的写入方案（含低谷/高峰如何表达）
- [ ] 用户确认范围后执行，并留痕（含删除前备份）
- [ ] 门禁：Go 与前端七条门；实例侧改动需用户授权后执行

## 调查结论（2026-10-22，三路只读）

### A. 「284」的真实来源（关键）

不是数据库模型表：

| 环节 | 实例现状 |
|---|---|
| `models`（模型元数据）| **0 行** |
| `abilities`（渠道能力）| **1 行**，只有 deepseek-flash |
| `channels` | 1 个，模型列表只含 deepseek-flash |
| 对外 `GET /api/pricing` | 只返回 **1** 个模型 |

「284」= **定价目录**：代码内置 `defaultModelRatio` 239 条 + `defaultModelPrice` 31 条等（外加数据库 option 覆盖项）。
→ 因此"彻底移除 283 个"的落点是**默认目录**（代码 + 实例 option），渠道/元数据环节**本来就是干净的**。

- 前端**无**写死模型清单（6 个接口全数据驱动）；i18n 与该批模型无关（删了 `i18n:check` 不受影响）；
- 删除不会级联（`logs`/`quota_data` 里模型名是普通字符串）；手工删 abilities 会被渠道保存"复活"，但本实例渠道本就只绑一个。

### B. 官方定价（2026-10-22 查询）

DeepSeek-Flash 官方人民币价（每 1M tokens，来源官方中文定价页）：低谷 缓存命中 ¥0.02 / 未命中 ¥1.00 / 输出 ¥4.00；高峰全部 ×2；
高峰时段 = 北京时间周一至周五 09:00–12:00、14:00–18:00（不含法定节假日）。
换算：显示值（每 1M）= 2 × model_ratio × 显示汇率；本实例汇率 1.0 → ratio = 元/1M ÷ 2。

### C. 本系统是否支持分时计价

比率体系**不支持**；唯一内建时段感知机制是 **billingexpr 表达式**（`hour/weekday/tier`）。

## 执行（实例侧，Lead）

1. **备份**：`/root/data/one-api.db.bak-before-model-trim-*`（1.1MB）+ option 原值导出（含旧 240/76/7 条）；
2. **写入 5 个 option**（`PUT /api/option/`，落库已复核）：
   - `ModelRatio` = `{deepseek-flash: 0.5}`（原 240 条 → 1 条）；
   - `CompletionRatio` = `{deepseek-flash: 4}`；`CacheRatio` = `{deepseek-flash: 0.02}`；
   - `billing_setting.billing_mode` = `{deepseek-flash: tiered_expr}`；
   - `billing_setting.billing_expr` = 官方峰谷表达式（weekday/hour 判北京时间，peak ×2，tier 标记）；
3. **真实请求测试**：临时令牌（用完即删，密钥未回显）→ 请求**打到上游但上游失败**（`upstream error: do request failed`）——
   与本次计费改动无关；顺带发现**渠道上游不是 DeepSeek 官方而是第三方网关**，故官方挂牌价与该渠道实际结算价可能不同（用户要的是官方价，已按官方价配置）；
4. **临时令牌清理**：测试期间创建的 2 个临时令牌已全部删除（仅保留用户自己的 1 个）。

## 执行（代码侧，v0.0.40）

- `defaultModelRatio` 239 → **10**（deepseek-flash + 3 个官方别名 + 5 个引擎注入的音频默认 + …）、`defaultModelPrice` 31 → 20（保留 MJ/Suno/换脸等功能动作计费键）、`defaultCacheRatio` 75 → 4、`defaultCreateCacheRatio` 42 → 0、`defaultAudioRatio` → 空、`defaultImageRatio` → 空；
- 保留项都有硬依赖依据：音频默认（`middleware/distributor.go:384,390,397` 注入，删了语音/转写未带 model 会 400）、MJ/Suno 计费键（`constant/midjourney.go`、`constant/task.go`）；
- deepseek-flash 默认价改为官方低谷价（0.5 / 4 / 0.02）；
- 修正 `MAINTENANCE.md` 里会低估 7.3 倍的旧换算公式；
- 新增 4 个测试；`go vet`/`go build`/`relaykit` 构建/`make test`（41 包）全绿；发 v0.0.40。

## 风险与未完成

1. **保留的 8 个非 deepseek 键**：5 个音频默认 + 3 个 deepseek 别名（删别名会让 `-none/-max` 归一失去价格）；
2. **渠道上游失败**：真实请求报 upstream 错，需用户侧排查（与本次改动无关）；
3. **渠道预设 ModelList 未动**（新装渠道的下拉预设仍含旧模型名，但不再带默认价）；
4. **成员自报越界**：代码线成员在一条命令模板里误把文件拷到工作区外（`/tmp`），发现后立即删除；Lead 已核实工作区外无残留、未外发、无凭据。
