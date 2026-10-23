# evidence-P4：订正 MAINTENANCE.md:318 与渠道模型提示

- 任务板：`task-2`（P4），执行者 `worker-docs`
- 基线 HEAD：`e6ec2d8f6ec19e9911fcc7f1bdb5a636b1c8278a`（2026-10-23）
- 写作用域内改动：`MAINTENANCE.md`、`web/src/features/channels/lib/channel-type-config.ts`（未新增 i18n 键）

## 一、代码事实核对（先实读代码，不照抄 journal）

### 1. `-none` 在三条路径上分别记录什么

`ParseDeepSeekV4ThinkingSuffix`（`relaykit/relayconvert/reasoning/suffix.go:45-58`）对 `-none` 返回
`thinkingType="disabled"`、`effort=""`；三条路径对 `info.ReasoningEffort` 的写法不同：

| 路径 | 代码位置 | `-none` 时 `info.ReasoningEffort` | 日志 `other.reasoning_effort` |
| --- | --- | --- | --- |
| OpenAI 兼容 chat | `relay/common/deepseek_v4_thinking.go:74-78`（`ApplyDeepSeekV4OpenAIThinkingSuffix`） | `"none"`（disabled 分支显式赋值） | `"none"` |
| Responses | 同文件 `:93-95` + `:113-115`（`ApplyDeepSeekV4ResponsesThinkingSuffix`） | `"none"`（`effort = "none"` 后同步到 info） | `"none"` |
| Claude | 同文件 `:41`（`ApplyDeepSeekV4ClaudeThinkingSuffix`）+ `relay/claude_handler.go:110-134` | `""`（`effort` 本身就是空串，直接赋值） | 键缺失 |

关键片段：

- chat 分支（唯一被提交 `fd5d5c8` 补上的 disabled 分支）：

  ```go
  // -none means thinking explicitly disabled; record it uniformly as
  // "none" (matches the Responses path) instead of an empty value.
  if thinkingType == "disabled" {
      info.ReasoningEffort = "none"
  } else {
      info.ReasoningEffort = effort
  }
  ```

- Claude 分支（未同步，仍是裸赋值）：

  ```go
  info.ReasoningEffort = effort   // -none 时 effort == ""
  ```

- 落库条件（空串不写键，所以 Claude 路径是「字段缺失」而不是「记 empty」）：

  ```go
  // service/log_info_generate.go:83-84
  if relayInfo.ReasoningEffort != "" {
      other["reasoning_effort"] = relayInfo.ReasoningEffort
  }
  ```

- Claude handler 的默认档补位守卫（`relay/claude_handler.go:131-134`）明确排除 `Thinking.Type == "disabled"`，
  因此 `-none` 不会被兜底成 `high`，仍留空。

结论：原文档「`-none` 三路径统一记录 "none"」**只对 chat 与 Responses 成立，Claude 不成立**；
其括号注解「Claude/chat 路径补 disabled 分支」也不成立——`fd5d5c8` 的 diff 只改了
`ApplyDeepSeekV4OpenAIThinkingSuffix`（`git show fd5d5c8 -- relay/common/deepseek_v4_thinking.go` 可复核），
提交信息里写的 "Claude/chat paths" 与实际改动不符。

### 2. 现状已被测试固化

`relay/common/deepseek_v4_thinking_test.go:161-166`（Claude 路径）显式断言 `-none` 时
`info.ReasoningEffort == ""`，并注明「OpenAI/Responses 路径记 "none"…本测试固化现状，不做顺手统一」；
`:114`、`:217` 分别断言 chat / Responses 记 `"none"`。

### 3. 默认模型表

- `setting/ratio_setting/model_ratio.go:38-39`：`defaultModelRatio` 只有 `deepseek-flash`、`deepseek-v4.1-flash`。
- `relay/channel/deepseek/constants.go:3-6`：渠道默认 `ModelList` = 两个基名 + 各自 `-none`/`-max`（共 6 条）。
- `MAINTENANCE.md:345`（「模型定价换算速查」段，本次订正后行号）已描述同一口径。
- 因此渠道提示里的 `deepseek-chat,deepseek-coder` 已失效，改为 `deepseek-flash,deepseek-v4.1-flash`。

## 二、改动清单

1. `MAINTENANCE.md:318-322`：把 A3 条目由「三路径统一记录 none」改为分路径陈述
   （chat → `"none"`、Responses → `"none"`、Claude → 字段缺失），补上 file:line 证据、
   提交 `fd5d5c8` 的实际改动范围，以及固化该行为的测试位置。
2. `web/src/features/channels/lib/channel-type-config.ts:113`：
   `models: 'deepseek-chat,deepseek-coder'` → `models: 'deepseek-flash,deepseek-v4.1-flash'`。

未改 i18n locale 文件，理由：`hints.models` 是**字面量数据字段**，不是 `t(...)` key；
且 `hints` 在 `web/src` 内已无消费者（`grep -rn "hints" web/src --include=*.ts --include=*.tsx`
只命中 3 处无关注释/字段；`getChannelTypeConfig` 仅被 `web/src/features/channels/lib/__tests__/new-api-channel.test.ts:64` 使用），
locale 文件里也不存在 `deepseek-chat` 相关键，故无键可补、无需七个语言文件同步。

## 三、验证（实际执行）

| 命令 | 结果 |
| --- | --- |
| `cd web && bun run build` | `BUILD_EXIT=0`，产物正常输出（Total 57362.7 kB / gzip 16551.3 kB）；`web/dist` 被 `web/.gitignore:11` 忽略 |
| `cd web && bun run typecheck`（tsgo -b --force） | `TYPECHECK_EXIT=0` |
| `cd web && bunx oxlint -c .oxlintrc.json src/features/channels/lib/channel-type-config.ts` | `Found 0 warnings and 0 errors`（1 file / 186 rules） |
| `cd web && bun run i18n:check` | `I18N_CHECK_EXIT=0`：4035 literal t() keys + 455 static keys = 4186 keys，7 语言全部存在（未改 i18n，跑一遍确认无回归） |
| `go test ./relay/common/ -run 'TestApplyDeepSeekV4.*CoversDeepSeekFamily' -count=1 -v` | `ok github.com/QuantumNous/new-api/relay/common 0.009s`；三条路径的 `-none` 子用例（`deepseek-flash-none` / `deepseek-v4.1-flash-none`）全部 PASS |

## 四、人工一致性核对说明（文档 ↔ 代码，可按下列行号逐条复查）

1. `relay/common/deepseek_v4_thinking.go:74-78`：`thinkingType == "disabled"` → `info.ReasoningEffort = "none"`（chat 记 `"none"`）。
2. `relay/common/deepseek_v4_thinking.go:93-95` + `:113-115`：Responses 把 `-none` 兜底为 `"none"` 并写回 info。
3. `relay/common/deepseek_v4_thinking.go:41`：Claude applier 裸赋值 `info.ReasoningEffort = effort`，`-none` 时为空。
4. `relay/claude_handler.go:110-134`：Claude relay 路径同样留空，且默认 high 守卫排除 `disabled`。
5. `service/log_info_generate.go:83-84`：空串不写键 → Claude 路径日志中该字段缺失。
6. `relay/common/deepseek_v4_thinking_test.go:161-166`：现状断言（Claude `-none` → 空）。

## 五、风险 / 未完成

- 未做端到端实跑（需要起服务、发请求、查日志表）；结论来自源码路径 + 现有单元测试，属静态与单测级证据。
- `hints.models` 目前无渲染方（可能在上游 knip 清理后成为惰性数据），本次仅做数据订正，未接入 UI 或 i18n。
- MAINTENANCE.md 只在 A3 条目订正；未顺带修改其他条目（如 `:261` 指 `setting/reasoning/suffix.go`，
  该文件仍存在但只有 20 行、`ParseOpenAIReasoningEffortFromModelSuffix` 等是转出口，
  实现在 `relaykit/relayconvert/reasoning/suffix.go`；是否补注留给 Lead 决定）。
- 行号会随后续改动漂移；本次以基线 `e6ec2d8` 为准。

## 六、建议下一步

1. 若产品口径要「三路径统一」：给 `ApplyDeepSeekV4ClaudeThinkingSuffix` 补 disabled → `"none"` 分支，
   并同步 `deepseek_v4_thinking_test.go:161-166` 的现状断言 + 更新 MAINTENANCE.md；若维持现状，
   本次订正即终态（无需改代码）。
2. 视需要订正 `MAINTENANCE.md:261` 的过时路径引用。
3. 若确认 `hints` 已无消费者，可评估删除整块惰性配置（本任务范围外）。
