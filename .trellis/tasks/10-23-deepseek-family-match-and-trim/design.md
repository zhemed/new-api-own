# Design — deepseek 家族判定与默认模板（v0.0.45）

## 1. 变更边界（写作用域）

| 文件 | 变更 |
|---|---|
| `relaykit/relayconvert/reasoning/suffix.go` | `IsDeepSeekV4Model` → **`IsDeepSeekModel`**，判定简化为单一 `strings.HasPrefix(modelName, "deepseek")`；`ParseDeepSeekV4ThinkingSuffix` 改调用新名 |
| `setting/reasoning/suffix.go` | 垫片 re-export 改名（`IsDeepSeekModel = kitreasoning.IsDeepSeekModel`） |
| `relay/common/deepseek_v4_thinking.go` | 2 处调用点改用新名；三个导出函数的注释改为"deepseek 家族（任何 deepseek 前缀模型）" |
| `setting/ratio_setting/model_ratio.go` | `FormatMatchingModelName` 改调用新名；两张默认表裁到 2 键；注释同步 |
| `setting/ratio_setting/cache_ratio.go` | 默认缓存表裁到 2 键 |
| `setting/billing_setting/tiered_billing.go` | `defaultTieredBillingModels` 裁到 2 个模型 |
| `relay/channel/deepseek/constants.go` | `ModelList` 裁到 6 条 |
| `relay/claude_handler.go` | 1 处调用点改用新名 |
| `setting/ratio_setting/model_defaults_test.go` | 目录锁 5→2、家族列表与同价断言改 2 模型 |
| `setting/billing_setting/tiered_billing_defaults_test.go` | 家族列表改 2 模型 |
| `relaykit/relayconvert/reasoning/suffix_test.go` | 谓词真值表按新语义重写 |
| `relay/common/deepseek_v4_thinking_test.go` | 新增 `deepseek-flash` 三档用例（与 v4-flash 对照） |
| `MAINTENANCE.md` | 默认目录描述同步为两个模型 |
| `VERSION` | 0.0.44 → 0.0.45 |

## 2. 关键设计决定

### 2.1 谓词语义：一条前缀，两条行为

```go
// IsDeepSeekModel reports whether modelName belongs to the DeepSeek family
// (any model name starting with "deepseek"): these accept the -none/-max
// thinking-effort suffixes and think by default.
func IsDeepSeekModel(modelName string) bool {
	return strings.HasPrefix(modelName, "deepseek")
}
```

- 判定**大小写敏感**（与仓库既有 `HasPrefix(name, "deepseek")` 一致）；`DeepSeek-V3.1` 这类大写开头不命中，
  这是刻意选择：宁可不命中，也不要把任意大小写变体都当思考模型。
- 两条行为一起放宽（用户裁定）：① `-max`/`-none` 识别；② 未指定档位时补 `reasoning_effort=high`。
- 已知副作用：`deepseek-chat`、`deepseek-coder`、`deepseek-ai/...` 等同前缀名字也会获得这两条行为。
  实例当前只挂 `deepseek-flash`，无即时影响；将来挂非思考模型需重新评估（用户已知情并接受）。

### 2.2 命名：只改语义已变的那一个符号

- 重命名 `IsDeepSeekV4Model`（语义已从"V4 家族"变为"deepseek 家族"，名字必须跟着改，否则是误导）。
- **不**重命名 `ApplyDeepSeekV4{OpenAI,Claude,Responses}ThinkingSuffix` 与文件名
  `relay/common/deepseek_v4_thinking.go`：这三个函数被 5 个适配器/处理器引用，改名属纯机械 churn，
  本轮以注释说明适用范围代替；若日后要正名，单独一次改名提交。

### 2.3 默认模板与渠道清单的一致性

- 四张默认表（倍率/输出/缓存/tiered）保留的键**完全一致**：`deepseek-flash`、`deepseek-v4.1-flash`。
  三者不一致会导致"有倍率没输出倍率"这类半配置状态（v0.0.44 前实例上真实出现过）。
- `ModelList` 只列这两个基名与各自 `-none`/`-max`（6 条），保证清单里的每个名字都有默认定价。
- 代价（用户裁定接受）：`deepseek-chat`、`deepseek-reasoner`、`deepseek-v4-flash`、`deepseek-v4-pro`
  既不在默认定价里，也不在 deepseek 渠道默认清单里；已在实例上挂着它们的部署需自行配置定价。

### 2.4 测试锚点的改写

旧测试把 `deepseek-v4.txt`、`deepseek-v4.1-flash-high` 之类的"点号/噪音"边界当作契约；
新语义下 `deepseek-v4.txt` 命中（它确实以 `deepseek` 开头），旧断言必须删掉而不是保留。
新真值表以"是否 `deepseek` 前缀"为唯一契约，且显式锁定 `gpt-4o`、空串、`DeepSeek-flash`（大写）为 false。

## 3. 发布设计

- `VERSION` = 0.0.45，与代码同一次提交，锚点 `[task:deepseek-family-match-and-trim]`。
- 注释 tag `v0.0.45` → 推 `main` 与 tag → `docker-build.yml` + `release.yml` 并行。
- 校验：三标签同摘要、Release 四件套（含 `.sig`）、`check-version-drift.sh`（不带 `--instance`，
  该脚本的实例比对有既有 v 前缀缺陷，见上一任务记录）。

## 4. 回滚

- 代码 revert 单个提交即回到 v0.0.44 语义（5 键默认表 + `deepseek-v4-` 判定）。
- 实例侧不需要动作：实例面板不读代码默认表。
