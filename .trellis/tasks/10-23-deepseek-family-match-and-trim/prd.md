# deepseek 判定放宽到家族前缀 + 默认模板裁到两个模型，发 v0.0.45

## 背景（用户当轮裁定）

1. `deepseek-v4-` 前缀判定对用户的实际用法不可靠：其渠道里挂的是普通 `deepseek-flash`，`deepseek-flash-max/-none` 完全不生效。
   用户裁定：**直接按 `deepseek` 匹配**，并且「后缀识别」与「未指定档位时自动补 `reasoning_effort=high`」**两条一起放宽**。
2. 代码默认价表是 5 个键（上一版加了 `deepseek-v4.1-flash`）；用户在实例面板上把模板定为
   **`deepseek-flash` + `deepseek-v4.1-flash` 两个模型**（都是「阶梯 / 表达式」），要求把代码默认值清理成这一份。
3. 用户裁定：`deepseek` 渠道默认 ModelList **一起裁到 6 条**（两个基名 + 各自 `-none`/`-max`），
   避免「清单里有名字但没定价」。

## Requirements

### R1 判定放宽为家族前缀

- 谓词从「`deepseek-v4-` 前缀，或 `deepseek-v4.<数字>` 前缀」改为**单一 `deepseek` 前缀**，
  并改名为 `IsDeepSeekModel`（旧名 `IsDeepSeekV4Model` 已名不副实），宿主侧仍走 `setting/reasoning` 垫片。
- 两条行为一起放宽：
  1. `-max` / `-none` 档位识别（OpenAI Chat / Claude / Responses 三条路径）；
  2. 未显式指定档位时自动补 `reasoning_effort=high`（OpenAI Chat 与 Responses 写进上游请求；Claude 路径记入 info）。
- 覆盖用户真实用法：`deepseek-flash`、`deepseek-flash-max`、`deepseek-flash-none`。
- 已知副作用（用户已知情并接受）：将来挂 `deepseek-chat` 这类非思考模型时，请求会带 `reasoning_effort=high`，
  上游可能拒绝。

### R2 代码默认模板裁到两个模型

下列默认值只保留 `deepseek-flash` 与 `deepseek-v4.1-flash`（与实例面板一致，两者数值相同）：

| 表 | 文件 | 保留键 |
|---|---|---|
| `defaultModelRatio` | `setting/ratio_setting/model_ratio.go` | 2 个，均 0.5 |
| `defaultCompletionRatio` | 同上 | 2 个，均 4 |
| `defaultCacheRatio` | `setting/ratio_setting/cache_ratio.go` | 2 个，均 0.02 |
| `defaultTieredBillingModels` | `setting/billing_setting/tiered_billing.go` | 2 个，共用同一条峰谷表达式 |

被移除的：`deepseek-v4-flash`、`deepseek-chat`、`deepseek-reasoner`（`deepseek-chat`/`deepseek-reasoner` 是官方 API 名，
本轮按用户「清理干净」的口径一并移除）。

### R3 deepseek 渠道 ModelList 裁到 6 条

`relay/channel/deepseek/constants.go` 的 `ModelList` =
`deepseek-flash`、`deepseek-flash-none`、`deepseek-flash-max`、
`deepseek-v4.1-flash`、`deepseek-v4.1-flash-none`、`deepseek-v4.1-flash-max`。

### R4 测试同步（不留旧语义的锚点）

- `setting/ratio_setting/model_defaults_test.go`：目录锁 5 → 2；家族列表改为两个模型；
  「同价」断言改为这两个模型。
- `setting/billing_setting/tiered_billing_defaults_test.go`：家族列表改为两个模型。
- `relaykit/relayconvert/reasoning/suffix_test.go`：谓词真值表按新语义重写
  （`deepseek-flash` / `deepseek-flash-max` / `deepseek-v4.1-flash` / `deepseek-v3-flash` = true；
  `gpt-4o` / 空串 / 大小写不同的 `DeepSeek-flash` = false），并保留 `ParseDeepSeekV4ThinkingSuffix` 契约用例。
- `relay/common/deepseek_v4_thinking_test.go`：新增 `deepseek-flash`（无后缀 / `-max` / `-none`）三档用例，
  断言与 `deepseek-v4-flash` 同后缀结果一致。

### R5 发版 v0.0.45

`VERSION` → `0.0.45`，带锚点提交，注释 tag `v0.0.45`，推送 `main` 与 tag；CI 产出镜像与 Release 资产；
校验三标签同摘要、Release 四件套齐全。

## Acceptance Criteria

- [ ] AC1 `IsDeepSeekModel` 对 `deepseek` 前缀返回 true（含 `deepseek-flash`、`deepseek-v3-flash`），
      对 `gpt-4o`、空串、非小写开头的 `DeepSeek-flash` 返回 false（表驱动断言）。
- [ ] AC2 `deepseek-flash-max` / `deepseek-flash-none` 在三条路径上都生效，且与 `deepseek-v4-flash` 同后缀结果一致。
- [ ] AC3 四张默认表各只剩 2 个键（0.5 / 4 / 0.02 / tiered），`GetBillingMode` 对两个模型都是 `tiered_expr`
      且表达式逐字符相同；目录锁断言 = 2。
- [ ] AC4 `deepseek` 渠道 `ModelList` 恰好 6 条，集合与 R3 相同。
- [ ] AC5 `GOWORK=off go vet ./...`、`GOWORK=off go build ./...`、`make test`（用 `-count=1`）、
      `cd relaykit && GOWORK=off go build ./...`、改动文件 `gofmt -l` 无输出。
- [ ] AC6 回归：`gpt-4o` 等非 deepseek 模型行为不变；`-none/-max` 之外的后缀（如 `-high`）不被误识别。
- [ ] AC7 tag `v0.0.45` 已推送；镜像 `latest`/`v0.0.45`/`0.0.45` 同摘要；Release 资产含
      `SHA256SUMS`、`SHA256SUMS.sig` 与两个 Linux 二进制。
- [ ] AC8 改动只落在声明写作用域内；提交消息带 `[task:deepseek-family-match-and-trim]`。

## Non-goals

- 不改实例（实例面板已经是"两个模型 + 阶梯表达式"的目标模板）。
- 不改 `ApplyDeepSeekV4*ThinkingSuffix` 三个导出函数名与文件名（只重命名语义已变的谓词），
  避免无谓的跨适配器改名；函数注释同步说明适用范围。
- 不做 deepseek 之外的任何模型/价格调整；不动峰谷表达式字面量。

## 风险

- R1 放宽后，任何 `deepseek` 开头的模型名都会获得档位识别与默认 high（用户已知情并接受）。
- R2/R3 会让 `deepseek-chat`、`deepseek-reasoner`、`deepseek-v4-flash`、`deepseek-v4-pro` 等既无默认价、
  也不再出现在 deepseek 渠道默认清单里；已装实例若仍挂着这些名字，会走"未配置计费"分支（实例侧不受代码默认影响）。
