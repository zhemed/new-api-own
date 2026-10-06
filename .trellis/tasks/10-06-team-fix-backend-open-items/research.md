# 后端开放项评估结论（证据留痕）

范围：`relay/helper/valid_request.go` multipart 图片编辑缺 model 校验；`relay/common/relay_info.go:849`（`TaskSubmitReq.Duration`）
与 `dto/video.go:10-12`（`Fps/Seed/N`）非指针标量 + `omitempty`。

---

## ① multipart 图片编辑缺 `model` 校验 → **结论：已补校验（option a）**

### 改了什么

- `relay/helper/valid_request.go:195-203`：multipart 分支读入 `model` 后立即 `return nil, errors.New("model is required")`，
  与 JSON 分支 `relay/helper/valid_request.go:245-248` 的报错文案一致。

### 证据链（为什么安全）

1. **空 model 本来就不会走到上游（常规路径）**：`middleware/distributor.go:370-378` 只在 `/v1/images/edits` 且
   `model` 非空时才写回 `modelRequest.Model`；若为空，`middleware/distributor.go:81-84` 直接 400
   （i18n `distributor.model_name_required`）。该中间件在同一路由组上先于 handler 执行
   （`router/relay-router.go:85,120`）。
2. **唯一能穿透的路径是"指定渠道"的密钥**：`middleware/auth.go:515-522` 对 admin 的 `sk-xxx-<channelId>` 设置
   `specific_channel_id`，此时 `middleware/distributor.go:36,42-56` 走 `ok` 分支，**跳过**第 81-84 行的空 model 检查。
   空 model 于是进入 handler → `relay/helper/model_mapped.go:77-79`（把空 model 当作 upstream model）→
   `relay/channel/openai/adaptor.go:457` 会把 `model=` 空字段写进上游 form；
   若渠道是 Replicate，`relay/channel/replicate/adaptor.go:79-85` 还会静默回落成 `ModelFlux11Pro`。
   即：要么白白打一次上游、要么按渠道隐式默认模型计费。
3. **合法"不带 model"的用法不存在**：JSON 分支自始就要求 model（`valid_request.go:245-248`），distributor 对
   `/v1/images/edits` 的 JSON / 表单 / multipart 三种 body 一律要求 model（`distributor.go:81-84,370-378`）；
   `distributor.go:371` 注释掉的 `gpt-image-1` 默认值说明"缺省补默认模型"是被有意取消的。
4. 补校验后唯一被拒的输入是：指定渠道 + Replicate + multipart 且不带 model（admin-only、渠道类型限定、无文档）。
   该形状在 JSON body 下本来就会失败，不是稳定契约。

### 测试

- 新增 `relay/helper/openai_image_request_test.go:TestGetAndValidOpenAIImageRequestMultipartModelRequired`
  （表驱动：缺 model → `model is required`；带 model → `req.Model` 正确解析）。实测通过。

### 附带发现（未修，不在本次两项范围）

handler 层校验错误经 `controller/relay.go:112` → `types.NewError(err, ErrorCodeInvalidRequest)`，而
`relaykit/types/error.go:257` 默认 `StatusCode = http.StatusInternalServerError`。
实测：`types.NewError(errors.New("model is required"), ErrorCodeInvalidRequest).StatusCode == 500`。
因此"缺 model 的 400"实际来自 distributor（`middleware/distributor.go:81-84`）；走指定渠道时补校验返回的是 500。
建议由 Lead 决定是否把 `controller/relay.go:112` 改成 `NewErrorWithStatusCode(..., http.StatusBadRequest)`
（`controller/` 不在本任务写范围内）。

---

## ② 客户端 DTO 非指针标量 + `omitempty` → **结论：不改（留痕）**

### 影响面清点

| 目标 | 读写点 | 是否会被 marshal |
|---|---|---|
| `dto/video.go:3-16` `VideoRequest{Fps,Seed,N}` | **0**（`grep "dto.VideoRequest"` = 0 命中） | 全仓库无人使用该类型 |
| `relay/common/relay_info.go:849` `TaskSubmitReq.Duration int` | relay/common 8 处（`relay_utils.go:149,177,216`；`relay_info.go:877-885`）+ 9 个 task 适配器 13 处（jimeng/ali/vidu/hailuo/sora/gemini/vertex/doubao/kling），`grep "\.Duration"` 共 29 行 | **从未被 marshal** |

### 为什么"不改"

1. **`omitempty` 对这两处是死规则**：`TaskSubmitReq` 只作为内部中间结构在 gin context 中传递
   （`relay/common/relay_utils.go:120-134`）、被适配器读成各自的 payload 结构
   （如 `relay/channel/task/hailuo/adaptor.go:146`）；全仓库没有任何 `common.Marshal(TaskSubmitReq…)`
   （task 适配器内 23 处 `Marshal` 全部是适配器自己的 `requestPayload`/`openAIVideo`/`body`）。
   AGENTS.md 该条针对的是"解析自客户端 JSON 并**再 marshal 给上游**"的结构，此处不适用。
2. **`dto/video.go` 是"文档型"死代码**：Go 侧 0 引用，只对应手写的 `docs/openapi/relay.json:4168-4230`
   （schema 把 fps/seed/n 描述为 integer）。把它改成指针会让 Go 类型与已发布 schema 静默漂移，
   且不改变任何运行时行为；真要修应连同 `docs/openapi/relay.json`（本次写范围外）一起处理或删除该死类型。
3. **显式 0 与缺省在语义上等价、且各消费点都有显式兜底**：`relay/channel/task/hailuo/adaptor.go:149`（`> 0` 才用）、
   `relay/channel/task/vidu/adaptor.go:233`、`relay/channel/task/kling/adaptor.go:272`（`DefaultInt(req.Duration, 5)`）、
   `relay/channel/task/sora/adaptor.go:110-113`（`<= 0 → 4`）、`relay/channel/task/ali/adaptor.go:410-421`（`> 0` 才用 + 默认 5 秒）。
   因此"显式 0 被丢"没有真实后果；改成指针只会给 13+ 处读取点加 nil 判断，行为零变化。

### 时长/计费钳制复核

- `relay/common/relay_utils.go:146-157`：`MaxTaskDurationSeconds = 3600`，`validateTaskDurationBounds` 生效（未改动）。
- `relay/relay_task.go:121-127`：remix 旧数据 `seconds` 钳到 3600 生效（未改动）。
- `relay/channel/task/gemini/billing.go:53-68`（`min(..., MaxTaskDurationSeconds)`）、
  `relay/channel/task/ali/adaptor.go:463`（同）均仍生效。

### 附带发现（未修，需 Lead 决定）

`duration` 与 `seconds` 同时出现时上界可被绕过：校验先取 `Duration`（`relay/common/relay_utils.go:149-152`），
而 sora 的计费先取 `Seconds`（`relay/channel/task/sora/adaptor.go:108-112`，**未做 min 钳制**）。
实测（临时探针，已删除）：`TaskSubmitReq{Duration:4, Seconds:"100000"}` → `EstimateBilling` 返回
`map[seconds:100000 size:1]`，即计费乘数 100000 > 上限 3600。入口为 `POST /v1/videos`（`router/video-router.go:30`
→ `controller.RelayTask` → `relay.RelayTaskSubmit` → `ValidateMultipartDirect`，`relay/common/relay_utils.go:206-239`）。
后果：配额经 `common.QuotaFromFloatChecked` 饱和（不会出现负扣费），但要么误报"额度不足"、要么按远超上限的价格预扣。
建议改法（约 4 行 + 1 个用例，写点在 `relay/common/relay_utils.go`）：上界同时约束两个字段，例如
`if req.Seconds != "" { if v, err := strconv.Atoi(req.Seconds); err == nil && v > seconds { seconds = v } }`。

---

## 实测命令与退出码（全部通过）

| 命令 | 退出码 | 备注 |
|---|---|---|
| `GOWORK=off go vet ./...` | 0 | 无输出 |
| `GOWORK=off go build ./...` | 0 | 无输出（`web/dist` 已存在） |
| `cd relaykit && GOWORK=off go build ./...` | 0 | 无输出 |
| `make test` | 0 | 根模块（除 main 包）+ relaykit 全绿；`relay/helper` 2.622s（含新增用例） |

改动文件（`git diff --stat`）：

```
 relay/helper/openai_image_request_test.go | 54 +++++++++++++++++++++++++++++++
 relay/helper/valid_request.go             |  6 ++++
 2 files changed, 60 insertions(+)
```

工作区其余改动（`MAINTENANCE.md`、`web/**`、`web/scripts/check-i18n-keys.mjs` 等）来自其他队友，本任务未触碰。

## 不确定项

1. multipart 补校验后，走"指定渠道"的请求会拿到 500（`controller/relay.go:112` + `relaykit/types/error.go:257`），
   不是 400；是否改 `controller/` 的映射由 Lead 决定。
2. sora `seconds` 上界绕过（见上）未修，需 Lead 决定是否单开一项。
3. `dto/video.go` 的 `VideoRequest` 是否为对外承诺的 OpenAPI 契约（`docs/openapi/relay.json:4168`）无法从代码判定，
   故按"不改"处理；若要删/改需同时处理该 schema。

> 上表两处"不确定项"已由 Lead 指派（task-9）处理，见下节。

---

# 追加（Lead 指派，task-9）：sora 时长上界绕过修复 + handler 校验状态码定性

## ① sora 计费上界绕过 → 已修（入口上界同时约束两个字段 + 适配器本地钳制）

**改前**：`validateTaskDurationBounds`（`relay/common/relay_utils.go:148-157`）只取 `Duration`（仅当 `Duration == 0` 才回退 `Seconds`），
而 sora 的 `EstimateBilling`（`relay/channel/task/sora/adaptor.go:109-115`）优先取 `Seconds` 且不钳制 →
`{"duration":4,"seconds":"100000"}` 通过校验，计费乘数 100000 > 上限 3600。

**改后**
- `relay/common/relay_utils.go:148-166`：Seconds 自身越界（<0 或 >3600）时由它决定校验结果，否则沿用原逻辑。
  只收紧、不放松：`{duration:-4,seconds:"8"}` 仍拒（Duration 负）、`{duration:4,seconds:"0"}` 仍放行、`Seconds` 解析失败不新增拒绝。
- `relay/channel/task/sora/adaptor.go:116-121`：本地钳到 `MaxTaskDurationSeconds`（与 `ali/adaptor.go:463`、`gemini/billing.go:53-68` 同款防御）。

| 输入（sora） | 改前 | 改后 |
|---|---|---|
| `{duration:4, seconds:"100000"}` | 通过校验，seconds ratio=100000，配额饱和 → 误报额度不足或按上限预扣 | 入口 400 `invalid_seconds`；即使到达适配器也被钳到 3600 |
| `{seconds:"100000"}`（无 duration） | 400 | 400（不变） |
| `{duration:4, seconds:"8"}` | 通过 | 通过（不变） |
| `{duration:4, seconds:"0"}` | 通过 | 通过（不变） |
| `{duration:4, seconds:"-5"}` | 通过（Duration 胜出；sora 回落 4 秒） | 400（Seconds 自身越界，收紧） |
| `{seconds:"-5"}` | 400 | 400（不变） |
| `{duration:-4, seconds:"8"}` | 400 | 400（不变） |

测试：
- `relay/common/relay_utils_test.go` `TestTaskDurationBounds` 新增 6 个用例（超界 Seconds 被拒、负 Seconds 被拒（含与合法 duration 并存）、负 duration 与合法 seconds 并存仍拒、边界 3600 放行、两字段均合法放行、seconds=0 放行）。
- 新增 `relay/channel/task/sora/adaptor_test.go` `TestEstimateBillingSecondsBound`：超界输入 → ratio 恰为 3600（断言不会出现 100000）。

**同形排查（"第二个时长字段绕过上界"）结论：无第二处需要修**

| 取值点 | 来源 | 是否计费乘数 | 上界 |
|---|---|---|---|
| `sora/adaptor.go:109` `req.Seconds` | 客户端 JSON `seconds` | 是 | **本次修复**（入口 + 本地） |
| `ali/adaptor.go:412` `req.Seconds`（仅 Duration≤0 时） | 同上 | 是 | 已有（`ali/adaptor.go:463` `min(…,Max)`） |
| gemini/vertex `ResolveVeoDuration`（`gemini/billing.go:53-68`） | metadata `durationSeconds` / Duration / Seconds | 是 | 已有 `min(…,Max)` |
| `doubao/adaptor.go:297` `req.Seconds` | 同上 | 否（doubao 按 resolution/video_input 查表计费，`doubao/constants.go:43-56`） | 不适用 |
| `relay/relay_task.go:121-133`（remix 旧数据） | DB 历史 task data | 是 | 已有钳制 |
| kling/vidu/hailuo/jimeng/suno | — | 无 `EstimateBilling`（`taskcommon.BaseBilling` 默认 nil） | 不适用 |
| `AdjustBillingOnSubmit` | — | 无任何适配器覆写（`taskcommon/helpers.go:90` 返回 nil） | 不适用 |

## ② handler 层校验错误 500 → 定性为"人手漏改"，已改为 400

判据：
1. **对外契约写明 400**：`docs/openapi/relay.json` 给 `POST /v1/chat/completions` 的 400 响应描述为"请求参数错误"。
2. **同函数同款写法**：`controller/relay.go:158`（模型价格错误）用 `ErrOptionWithStatusCode(http.StatusBadRequest)`；
   `relay/responses_handler.go:167`、`controller/channel-test.go:300` 同样。
3. **成对的 400/500 反例**：`relay/image_handler.go:28`（请求类型不符 → 显式 400）与 `:33`（DeepCopy 失败 → 默认 500），
   即"默认 500 留给内部故障、入参问题显式 400"；第 112 行漏了。
4. `relaykit/types/error.go:257` 的 500 只是通用兜底，relaykit 内没有把 `invalid_request` 固定映射成 5xx 的逻辑。

改法（最小面）：`controller/relay.go:112` →
`types.NewError(err, types.ErrorCodeInvalidRequest, types.ErrOptionWithStatusCode(http.StatusBadRequest))`。
- 只影响 `helper.GetAndValidateRequest` 出错这一个分支；`NewError` 的 `errors.As` 分支保留内部 typed error（含 skipRetry），仅改状态码。
- 不改 relaykit（`NewError` 默认仍 500，`service/billing_session.go:348` 等内部故障路径不受影响）；413 分支与其它错误码映射不变。

测试：新增 `controller/relay_validation_status_test.go` `TestRelayValidationErrorStatus`
（`/v1/chat/completions` 缺 model、`/v1/images/edits` multipart 缺 model → 均 HTTP 400 且消息含 `model is required`）。

观察（未改，另行报备）：`controller/relay.go:145` 敏感词命中仍走默认 500（`ErrorCodeSensitiveWordsDetected`），
同属"客户端触发、应 4xx"的类别，但改动触及内容安全语义，未纳入本次范围。

## 追加改动文件与实测（第二轮）

```
 controller/relay.go                         |  3 ++-
 controller/relay_validation_status_test.go  | new
 relay/common/relay_utils.go                 |  7 +++++
 relay/common/relay_utils_test.go            | 16 ++++++++++
 relay/channel/task/sora/adaptor.go          |  5 +++
 relay/channel/task/sora/adaptor_test.go     | new
```

| 命令 | 退出码 |
|---|---|
| `GOWORK=off go vet ./...` | 0 |
| `GOWORK=off go build ./...` | 0 |
| `cd relaykit && GOWORK=off go build ./...` | 0 |
| `make test` | 0（`relay/common` 0.018s、`relay/channel/task/sora` ok、`controller` 0.831s 含新增用例、`relay/helper` ok） |

`gofmt -l` 对全部改动文件无输出。

---

# 追加（Lead 指派，task-10）：敏感词命中 500 → 400

## 选码依据（为什么 400 而不是 403）

1. **同仓库同类先例**：上游内容策略拦截在 gemini 适配器里用 `ErrorCodePromptBlocked` + **400**——
   `relay/channel/gemini/relay-gemini.go:334-338`、`relay/channel/gemini/relay_responses.go:41-45`
   （`request blocked by Gemini API: …` → `http.StatusBadRequest`）。本地敏感词过滤属同一类"请求内容被策略拒绝"。
2. **middleware 的 403 全是身份/授权类**：用户封禁 `middleware/auth.go:452`、IP 白名单 `:437`、分组准入 `:463`、
   "普通用户不支持指定渠道" `:520`、`middleware/distributor.go:54,65,75,97`——没有任何"内容策略"用 403。
   403 语义是"你无权"，而这里客户端改内容就能通过。
3. **发布契约**：`docs/openapi/relay.json` 对 relay 端点只声明 200/400/401/404/429/501（**无 403**），
   `/v1/chat/completions` 的 400 描述即"请求参数错误"。
4. **重试语义**：5xx 会被中继/SDK 当成服务端故障重试，而该请求重试必然再被拦。
   （旁证：OpenAI 自身内容策略拒绝也是 400 `content_policy_violation`；本机无网络，未做在线核验。）

## 改法（仅一处映射）

`controller/relay.go:147`：
`types.NewError(err, types.ErrorCodeSensitiveWordsDetected, types.ErrOptionWithStatusCode(http.StatusBadRequest))`

- 保留原错误码 `sensitive_words_detected`（客户端仍可按码分支）。
- **relaykit 未改**：`relaykit/types/error.go:42` 只定义该码，没有把它固定成 5xx 的逻辑；控制器这一处是唯一使用点。
- 兼容性核对：web 侧与文档均未依赖该状态码（`grep -rn 敏感/sensitive web/src`、`grep sensitive_words_detected docs` 无相关命中）。
- 行为差异：`500 {"error":{"message":"sensitive_words_detected","type":"new_api_error","code":"sensitive_words_detected"}}`
  → `400 {"error":{…同上…}}`（消息体不变，仅状态码；注意此处 `err` 为 nil，消息即错误码字符串，本次未改）。

## 测试

`controller/relay_error_status_test.go`（由 `relay_validation_status_test.go` 改名，两类映射同文件）：
- `TestRelayValidationErrorStatus`：`/v1/chat/completions` 缺 model、`/v1/images/edits` multipart 缺 model → 400。
- `TestRelaySensitiveWordsStatus`：fixture 内显式设 `setting.SensitiveWords = []string{"blocked-phrase"}` 与两个开关（`t.Cleanup` 还原），
  正文含该词 → 断言 400 且响应含 `sensitive_words_detected`；无需 DB/上游。

## 同类分支盘点（`controller/relay.go`，**只列不改**）

| 行 | 错误码/来源 | 触发方 | 现状 | 备注 |
|---|---|---|---|---|
| 116 | `read_request_body_failed` | 客户端 | 413（显式） | OK |
| 120 | `invalid_request` | 客户端 | 400（上一轮已改） | OK |
| 127 | `gen_relay_info_failed` | 服务端（relayFormat/request 类型不匹配，路由固定） | 500 | 可接受 |
| 147 | `sensitive_words_detected` | 客户端 | 400（本轮已改） | OK |
| 154 | `count_token_failed` | **客户端可控**：multipart 解析失败、音频文件读不出时长（`service/token_counter.go:193-209`） | 500 | **候选，未改，请 Lead 定** |
| 162 | `model_price_error` | 客户端 | 400（显式） | OK |
| 171 | 预扣费（`service.PreConsumeBilling`） | 客户端 | 403（显式，`service/billing_session.go:201,219,280,362`） | OK |
| 216/218 | `read_request_body_failed` | 客户端 | 413 / 400（显式） | OK |
| 320/323 | `get_channel_failed`（重试取渠道） | 服务端容量/配置 | 500 | 与首轮选择的 `middleware/distributor.go:154,158`（503/404）语义不一致，报备 |

## 第三轮实测

| 命令 | 退出码 | 备注 |
|---|---|---|
| `GOWORK=off go vet ./...` | 0 | |
| `GOWORK=off go build ./...` | 0 | |
| `cd relaykit && GOWORK=off go build ./...` | 0 | relaykit 未改动，独立构建仍通过 |
| `make test` | 0 | `controller` 0.447s（含新增敏感词用例） |
| `gofmt -l`（8 个改动文件） | 0 | 无输出 |
