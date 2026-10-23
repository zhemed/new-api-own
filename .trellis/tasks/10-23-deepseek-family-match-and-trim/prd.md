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

## 验收结果（2026-10-23 结项）

- [x] AC1 `IsDeepSeekModel` 契约：true = `deepseek` / `deepseek-flash` / `deepseek-flash-max` / `deepseek-v3-flash` /
      `deepseek-v4.1-flash` / `deepseek-chat` / `deepseek-ai/deepseek-v3` / `deepseekx`；false = `gpt-4o` / 空串 /
      `DeepSeek-flash` / `DEEPSEEK` / `" deepseek-flash"` / `xdeepseek-flash` / `deep-seek-flash`
      （实现者 10 例 + verify-pricing 独立 17 例真值表 + 11 例解析契约，全 PASS）。
- [x] AC2 `deepseek-flash` / `-max` / `-none` 在 OpenAI Chat / Claude / Responses 三条路径上**端到端实测**通过，
      且与 `deepseek-v4-flash` 同后缀逐项一致（verify-pricing 独立推导逐格比对一致；另有 18 个子测试固化进仓库）。
- [x] AC3 四张默认表键集完全相同且恰好 2 个（`deepseek-flash`、`deepseek-v4.1-flash`）：0.5 / 4 / 0.02 /
      `tiered_expr`，表达式逐字符相同；目录锁 2；被裁掉的 `deepseek-v4-flash` / `deepseek-chat` /
      `deepseek-reasoner` 三查皆无默认价。
- [x] AC4 `ModelList` 恰好 6 条，集合与 R3 相同。
- [x] AC5 `go vet` / `go build` / `go test -count=1 ./setting/... ./relay/...`（26 ok）/ relaykit 测试与独立构建
      全通过；Lead 另跑根模块全量无缓存（36 ok）+ relaykit 全量；改动文件 `gofmt -l` 无输出。
- [x] AC6 回归：`gpt-4o` 行为不变；`deepseek-flash-high` 不被识别为档位（但按新语义补 high，见风险）；
      mutation 两条（谓词 `return false`、默认表多一键）都能让断言变红并已还原。
- [x] AC7 tag `v0.0.45` 已推送；镜像 `latest`/`v0.0.45`/`0.0.45` 同摘要
      `sha256:e114b17532f179760f87410595c0821d8b116955dc1d68cf82c0aaffe25c5c9f`；Release 资产四件套齐全；
      `check-version-drift.sh`（不带 `--instance`）判定 OK。
- [x] AC8 改动 14 个文件（12 代码/测试 + `MAINTENANCE.md` + `VERSION`）+ 任务目录；提交 `b5f9388` 带锚点
      `[task:deepseek-family-match-and-trim]`，闸门放行；随后 `09b1af1` 为 parity 单测（仅测试文件）。

## 追加交付（用户当轮追加确认）

- [x] 固化 parity 单测：`relay/common/deepseek_v4_thinking_test.go` 新增 Claude / Responses 两条路径的
      3 档 × 2 模型用例（+108/-7），每格同时有绝对期望值与与 `deepseek-v4-flash` 的逐项对照；
      verify-pricing 复核：逐格与独立推导一致、2 格翻转活性验证通过、无 parity-only 断言。
- [x] 本地实例升到 v0.0.45（镜像方式，参数逐项保留）：`new-api` = v0.0.45 running，
      `new-api-prev-v0044` / `new-api-prev-v0043` 停着作回滚点；`/api/status` = v0.0.45，日志无 error；
      实例侧定价配置（2 键 + 同一表达式）随重建完整保留。
- 未随测试单独发版：测试不进镜像，CI 不跑测试，故 `09b1af1` 不提版本、不打 tag（VERSION 仍 0.0.45）。

## 遗留（本轮发现，未处理）

1. **Claude 路径 `-none` 记空**：`info.ReasoningEffort=""`，而 OpenAI/Responses 记 `"none"`；测试内已注明
   "既有行为，不做顺手统一"。`MAINTENANCE.md:318` 的旧条目「`-none` 三路径统一记录 none」与此不符（文档口径）。
2. **`deepseek-flash-high` 这类不受支持的后缀**：不被识别为档位、模型名原样透传，但会因前缀命中补 `high`
   （用户裁定"两条一起放宽"的既定副作用，已写入 Release notes）。
3. **前端提示未同步**：`web/src/features/channels/lib/channel-type-config.ts:113` 对渠道类型 43 的模型提示仍是
   `deepseek-chat,deepseek-coder`，与裁剪后的 6 条清单不一致（既有问题，本轮未动前端）。
4. **两个回滚容器占名**：`new-api-prev-v0044` / `new-api-prev-v0043` 均为 host 网络，真回滚须先停 `new-api`
   （会短暂中断 3000 端口）；保留期限待用户定。
5. **实例仍未暴露新模型**：`/api/pricing` 是能力驱动，渠道未挂 `deepseek-v4.1-flash` 时只出 1 条（用户自行处理）。
6. 既有缺陷：`scripts/check-version-drift.sh` 的实例比对未剥 `v` 前缀，带 `--instance` 必报 DRIFT。

## Non-goals

- 不改实例（实例面板已经是"两个模型 + 阶梯表达式"的目标模板）。
- 不改 `ApplyDeepSeekV4*ThinkingSuffix` 三个导出函数名与文件名（只重命名语义已变的谓词），
  避免无谓的跨适配器改名；函数注释同步说明适用范围。
- 不做 deepseek 之外的任何模型/价格调整；不动峰谷表达式字面量。

## 风险

- R1 放宽后，任何 `deepseek` 开头的模型名都会获得档位识别与默认 high（用户已知情并接受）。
- R2/R3 会让 `deepseek-chat`、`deepseek-reasoner`、`deepseek-v4-flash`、`deepseek-v4-pro` 等既无默认价、
  也不再出现在 deepseek 渠道默认清单里；已装实例若仍挂着这些名字，会走"未配置计费"分支（实例侧不受代码默认影响）。
