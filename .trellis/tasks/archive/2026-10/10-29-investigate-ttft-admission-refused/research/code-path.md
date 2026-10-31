# 代码链路分析：ttft_admission_refused 与 429 / RATE_LIMIT 的来源

- 任务：task-2，`.trellis/tasks/10-29-investigate-ttft-admission-refused`
- 仓库：`/root/new-api-own`，HEAD `1c96601`
- 方式：只读；未改代码与配置；未访问工作区外路径与凭据；联网仅公开只读检索

## 0. 结论速览

1. 截图中 `ttft_admission_refused`、`the cell is at capacity`、`Upstream request failed`、已重试模型请求、重试延迟、本轮运行失败、RATE_LIMIT 在本仓库代码中**零命中**，截图的错误条目界面**不是本项目 Web 前端**，判定为 DeepSeek Harness 客户端界面，依据见第 1 节与第 7 节。
2. `ttft_admission_refused` 是**上游服务返回的错误体内容**，本网关既没有产生也没有改写该字符串；new-api 只在 429 响应体解析时保留上游的 message、type、code 字段，并以 `{"error":{...}}` 包装回客户端。
3. `RATE_LIMIT` 是客户端对 HTTP 429 的**分类标签**，不是本网关的限流枚举；本网关限流默认关闭，且其 429 文案为中文、带 request id、type 为 `new_api_error`，与截图不符。
4. `已重试模型请求 5/5` 与 `重试延迟 7-9 秒` 是**客户端重试策略**，不是本网关行为：本仓库 `RetryTimes` 默认 0，且重试循环内没有任何 sleep，不可能产生 5 次重试和秒级延迟。
5. 业务含义：上游容量准入拒绝；本网关无自适应熔断，429 默认**不会**触发渠道自动禁用，因此故障期间会持续把请求发往同一饱和上游。

## 1. 字符串溯源

在仓库根目录做全量 grep，结果如下。

| 关键词 | 命中 | 位置与判断 |
|---|---|---|
| `ttft_admission_refused` | 0 | 不在本仓库，非本网关文案 |
| `the cell is at capacity` | 0 | 不在本仓库 |
| `Upstream request failed` | 0 | 不在本仓库；本网关不加此统一前缀 |
| `已重试模型请求` | 0 | 不在本仓库 |
| `重试延迟` | 0 | 不在本仓库 |
| `本轮运行失败` | 0 | 不在本仓库 |
| `处理失败` | 仅 `批量处理失败` | `web/src/i18n/locales/zh.json:542`，与本截图无关 |
| `失败原因` | 有 | `web/src/i18n/locales/zh.json:1683` 的 `Fail Reason`，用于后台日志详情，周边文案与截图不符 |
| `RATE_LIMIT` | 3 | 全部是前端表单常量 `RATE_LIMIT_FORM_ID`，见 `web/src/features/system-settings/request-limits/rate-limit-dialog.tsx:52`；与限流标签无关 |
| `API error` | Go 仅注释、web 仅语言偏好文案 | `pkg/ionet/types.go:250`、`pkg/ionet/client.go:127`、`web/src/features/profile/components/language-preferences-card.tsx:118`；不存在 `xx API error (429)` 这种对外错误前缀 |
| `upstream_error` | 有，作为内部错误类别 | `relaykit/types/error.go:35`、`:327`、`:351`、`relaykit/dto/claude.go:544`；不是本仓库对外固定文案 |
| `server_error` | 有，仅视频代理端点 | `controller/video_proxy.go:44` 等 13 处 `videoProxyError`；聊天中继链路不产生该 code |

结论：截图的展示层不是本仓库；本仓库只负责把上游错误原样上报，错误内容体来自更上游。

## 2. 传递链文字图

```
用户
  │ 在 DeepSeek Harness 客户端里发模型请求，模型路由 provider 指向 new-api
  ▼
DSH 客户端  ── HTTP POST /v1/chat/completions 或 /v1/responses ──►
  new-api 入口：middleware.TokenAuth → ModelRequestRateLimit → Distribute 选渠道
  （router/relay-router.go:69-98）
  ▼
relay 通用处理器转发到上游渠道
  （relay/compatible_handler.go:203-220）
  ▼
上游服务 / 上游中转网关
  ── 返回 429，body 内层错误 message 为
     "Upstream request failed: [ttft_admission_refused] the cell is at capacity; retry shortly (after 4 s)"
  ▼
service.RelayErrorHandler 解析上游错误体
  （service/error.go:81-132）
  嵌套 error 对象路径：保留 message / type=upstream_error / code=server_error
  （service/error.go:110-119 + relaykit/types/error.go:317-347）
  ▼
controller.Relay 统一写回客户端：HTTP 429 + {"error":{message,type,param,code}}
  （controller/relay.go:92-110）
  ▼
DSH 客户端把 429 归类为 RATE_LIMIT，按客户端 retryPolicy 重试 5 次并退避
  ▼
5 次均失败 → 界面显示 已重试模型请求 5/5、重试延迟、失败原因、本轮运行失败 RATE_LIMIT
```

截图中的 `xx` 位于 `xx API error (429)`，是 DSH 侧模型路由或 provider 名称，不是 new-api 的渠道名；本仓库无对应文案，无法从代码把它映射到具体渠道，需要 DSH 配置与生产库的渠道记录对照，属 prod-ops 取证范围。

## 3. 429 传递链关键代码

### 3.1 上游非 200 的统一入口

- `relay/compatible_handler.go:211-219`：`httpResp.StatusCode != http.StatusOK` 时调用 `service.RelayErrorHandler`，再按渠道配置做 `ResetStatusCode` 状态码映射。
- 其余格式同理：`relay/claude_handler.go:239`、`relay/responses_handler.go:144`、`relay/gemini_handler.go:170,272`、`relay/embedding_handler.go:79`、`relay/rerank_handler.go:91`、`relay/audio_handler.go:57`、`relay/image_handler.go:106`、`relay/chat_completions_via_responses.go:156`。

### 3.2 错误体解析：是否原样透传

`service/error.go:81-132` 的 `RelayErrorHandler`：

- `:82` 先以自定义错误码与上游状态码初始化 `NewAPIError`，状态码默认原样保留，429 仍是 429。
- `:84-97` 读取整个响应体；解析失败时只保留状态码。
- `:110-119` 若 body 的 `error` 是对象，按通用 OpenAI 错误结构解析，`types.WithOpenAIError` 会**保留上游 message、type、param、code**。
- `:121-131` 若 body 是扁平结构，则只取 message，type 与 code 会被改成本地错误码 `bad_response_status_code`。
- 反证：截图里 type 为 `upstream_error`、code 为 `server_error`，不是 `bad_response_status_code`，因此上游返回的必须是**嵌套的 OpenAI 风格 error 对象**，new-api 只是原样保留了它的字段。这一点也被回归测试锁定：`service/error_test.go:111-123` 用 `{"error":{"message":...,"type":"server_error","code":"server_error"}}` 断言 message 被保留。

`relaykit/dto/error.go:23-50` 定义通用错误体，`TryToOpenAIError` 只接受嵌套 error 对象；`:52-93` 的 `ToMessage` 支持顶层 message、msg、err、error_msg、detail 等扁平写法。

`relaykit/types/error.go` 关键点：

- `:13-19` 对外错误结构字段顺序为 message、type、param、code，与截图 JSON 的字段顺序一致。
- `:180-211` `ToOpenAIError`：上游错误类型的 message、type、code 原样输出，只做敏感信息遮蔽。
- `:317-347` `WithOpenAIError`：type 为空时**本网关会补 `upstream_error`**，code 取上游 code 字符串并转成内部错误码；若上游 body 显式给了 type，则原样保留。
- `:326-328` 因此截图中的 `upstream_error` 可能来自上游，也可能是 new-api 补的默认类型；两种情况下 message 都来自上游。

### 3.3 客户端收到什么

`controller/relay.go:92-110`：统一 `SetMessage` 追加 request id 后，按协议写回。OpenAI 格式为 `{"error": newAPIError.ToOpenAIError()}`，HTTP 状态码取上游状态码。`common/utils.go:292-294` 的 `MessageWithRequestId` 会生成 `... (request id: xxx)`，但该后缀只作用于 `Err` 字段，OpenAI 类型错误走 `RelayError` 原样输出，所以截图中 message 没有 request id 后缀，与“上游 OpenAI 错误对象被保留”的判断一致。

### 3.4 重试在哪一层、上限与延迟

- 网关内部重试：`controller/relay.go:188-248` 的 `for ; retryParam.GetRetry() <= common.RetryTimes; ...` 循环；每次重试重新选渠道后重放请求。
- 上限：`common.RetryTimes`，默认 0，见 `common/constants.go:137`，可由后台配置项 `RetryTimes` 修改，见 `model/option.go:159` 与 `model/option.go:556-557`。
- 延迟：该循环体内**没有任何 sleep 或退避**；全仓库与 relay 重试相关的 `time.Sleep` 只有渠道测试、模型同步等旁路，`controller/relay.go` 内无等待。因此本网关重试是“立即连续重放”，不会产生 7-9 秒延迟。
- 哪些状态码会重试：`controller/relay.go:335-365` 的 `shouldRetry`；默认重试区间 `setting/operation_setting/status_code_ranges.go:21-29`，其中 409-499 覆盖 429，504 与 524 永不重试，见 `status_code_ranges.go:31-34` 与 `:80-85`。
- 渠道亲和时跳过重试：`controller/relay.go:339` 与 `service/channel_affinity.go:626-642`。
- 指定渠道时跳过重试：`controller/relay.go:351-353`。
- 任务类请求的 429 重试独立实现：`controller/relay.go:626-665`，429 直接判定可重试。
- 结论：截图中 5/5 与 7-9 秒延迟不可能来自默认配置的 new-api；与 DSH 默认重试策略吻合，见第 7 节。

### 3.5 重试是否会切换渠道

- 入口选择渠道：`middleware/distributor.go:33-171`，先看渠道亲和命中，见 `:105-133`；未命中则 `service.CacheGetRandomSatisfiedChannel` 按分组、优先级、权重选渠道，见 `:135-161`。
- 渠道亲和默认开启：`setting/operation_setting/channel_affinity_setting.go:112-149`，`Enabled: true`，内置 codex cli 与 claude cli 两条规则，匹配 `/v1/responses` 与 `/v1/messages`，`SkipRetryOnFailure: true`，默认 TTL 3600 秒。
- 亲和只对成功响应记录：`middleware/distributor.go:167-169`，仅状态码小于 400 时写亲和缓存；因此失败请求本身不会重新绑定渠道，但此前成功过的请求可能把后续请求固定在同一渠道，命中后本网关内部重试也会被跳过。
- 网关内部重试的选路：`service/channel_select.go:83-162`，按 retry 序号逐级走优先级与分组；`model/channel_cache.go:114-209` 在同优先级内按权重随机。
- 客户端每次重试是新的 HTTP 请求：会重新经过 `Distribute`，若同模型存在多优先级或多渠道，有概率切到其他渠道；若只有一个渠道、或亲和命中，则仍打同一上游。

### 3.6 前端文案如何封装

本仓库 Web 前端没有该错误条目的封装逻辑；错误文案由客户端直接展示上游返回的 message。new-api 只保证 message 字段与状态码透传，不做“处理失败、重试延迟、本轮运行失败”这类客户端状态文案。

## 4. 错误映射表

| 截图元素 | 产生层 | 判定依据 |
|---|---|---|
| `ttft_admission_refused` | 上游服务，本仓库零命中 | 全仓库 grep 0 命中；new-api 不产生该 token |
| `the cell is at capacity; retry shortly (after 4 s)` | 上游服务的容量准入调度器 | 同上；是上游 429 错误体的 message 内容 |
| `Upstream request failed: [...]` 前缀 | 上游或上游中转网关的错误包装 | 本仓库无该串；公开检索发现同类网关对上游失败使用相同前缀格式，见第 7 节 URL |
| `{"message":...,"type":"upstream_error","param":"","code":"server_error"}` | 上游 OpenAI 风格错误对象，经 new-api 解析保留字段 | `service/error.go:110-119`、`relaykit/types/error.go:180-211`、`:317-347`；扁平 body 会被改写成 `bad_response_status_code`，与截图不符 |
| HTTP 429 | 上游状态码原样透传 | `service/error.go:82`、`:127`、`relay/compatible_handler.go:214-218` |
| `RATE_LIMIT` | 客户端对 429 的分类标签 | 本仓库无该枚举，仅前端表单常量同名；客户端公开讨论显示 429 映射为 RATE_LIMIT，见第 7 节 |
| `已重试模型请求 5/5` | 客户端重试 UI | 本仓库零命中；本网关默认不重试 |
| `重试延迟 8638/7356 毫秒` | 客户端指数退避 | 本网关重试循环无 sleep；7-9 秒量级与客户端默认退避上限吻合，见第 7 节 |
| `本轮运行失败` | 客户端会话状态文案 | 本仓库零命中 |
| `xx API error` 前缀 | 客户端 provider 级错误前缀 | 本仓库零命中；`xx` 取自客户端模型路由配置，非渠道名 |
| 右侧 `RATE_LIMIT` 徽标 | 客户端错误分类徽标 | 同 RATE_LIMIT |

补充：本网关自身也可能返回 429，但与截图不符，可作为 prod 排查对照。

- 模型请求限流：`middleware/model-rate-limit.go:83`、`:105`，仅在后台开启 `ModelRequestRateLimitEnabled` 时生效，默认关闭，见 `setting/rate_limit.go:11`；文案为中文“您已达到请求数限制…”，外层经 `middleware/utils.go:12-27` 包装为 `type: new_api_error` 且 message 带 request id。
- 全局接口限流：`middleware/rate-limit.go:135-146`，默认关闭，见 `common/init.go:125-139`；响应带 `Retry-After` 头。
- 渠道负载饱和类文案：`controller/relay.go:444`、`:621` 的中文“当前分组负载已饱和”，属于 Midjourney 与任务类，不在聊天链路。

## 5. RATE_LIMIT 标记判定

- 本仓库 Go 代码没有 `RATE_LIMIT` 枚举或标签；`RATE_LIMIT` 只作为前端表单常量出现在 `web/src/features/system-settings/request-limits/rate-limit-dialog.tsx:52`。
- 本网关的错误码体系是另一个集合，见 `relaykit/types/error.go:38-88`，例如 `bad_response_status_code`、`channel:invalid_key` 等，没有 RATE_LIMIT。
- 因此 RATE_LIMIT 是**截图客户端对 HTTP 429 的分类结果**，既不是本网关限流命中，也不是本网关对上游错误做的映射。

## 6. 业务含义与网关侧行为

### 6.1 对用户的含义

上游容量准入拒绝，不是本网关的并发或速率限制。上游用 429 表达“当前容量单元已满，稍后重试”，其中 after 4 s 是上游建议的等待时间。用户侧表现为该模型持续不可用，直到上游恢复或请求被路由到其他渠道。

### 6.2 渠道自动禁用与熔断

- 自动禁用总开关默认关闭：`common/constants.go:132`，`AutomaticDisableChannelEnabled = false`；开关实现见 `service/channel.go:45-65`。
- 默认仅 401 触发自动禁用：`setting/operation_setting/status_code_ranges.go:17`；429 不在默认禁用范围，除非生产后台把 429 加进禁用状态码或关键词。
- 即使开启自动禁用，还要渠道自身 `AutoBan` 为真，见 `controller/relay.go:371-375` 与 `service/channel.go:23-26`。
- 本网关没有基于错误率的自适应熔断或降级；失败只记录错误日志，见 `controller/relay.go:367-410`，日志字段含 `error_type`、`error_code`、`status_code`、`channel_id`、`channel_name`、`use_channel`，可供 prod-ops 直接查库取证。
- 结论：默认配置下，上游持续 429 不会让渠道被自动摘除，网关会继续把新请求路由到该渠道。

### 6.3 点击重试与自动重试的差别

- 客户端重试：每次是新的 HTTP 请求，重新经过路由与渠道选择，有可能换到其他渠道；是否换取决于渠道数量、优先级、权重与亲和缓存。
- 网关自动重试：默认关闭；若生产开启 `RetryTimes` 大于 0，429 属默认可重试区间，会立即重放并逐级切换优先级或分组，没有退避延迟，客户端只看到最终一次结果。
- 亲和规则命中时，本网关会跳过内部重试，见 `controller/relay.go:339`，所以开启亲和且客户端走 `/v1/messages` 或 `/v1/responses` 时，更容易反复命中同一渠道。

## 7. 外部公开检索结论

1. `ttft_admission_refused` 与 `the cell is at capacity` 没有检索到官方文档、公开代码或服务商说明，无法确认具体发布方。按字面含义为：上游网关在首 token 时延指标上做容量准入，`cell` 是容量单元，当前已满，返回 429 并建议 4 秒后重试。该错误属于上游临时容量问题，不是鉴权、余额或参数错误。
2. 同类错误前缀 `Upstream request failed: [<code>] <message>` 在 opencode 的 Console Go 提供商问题中出现，说明这是某类网关对上游失败的统一包装格式，而不是 new-api 的行为：
   - https://github.com/anomalyco/opencode/issues/37231
   - https://github.com/anomalyco/opencode/issues/37815
3. 截图界面判定为 DeepSeek Harness 客户端，证据是相同界面文案与重试机制在 DSH 公开讨论中被完整描述：
   - https://github.com/deepseek-ai/deepseek-harness/discussions/6818 ：界面显示 `已重试模型请求（5/5） · 9s`，展开为 `重试延迟：8124 毫秒` 与 `失败原因：...`，随后 `本轮运行失败 TRANSPORT`。与截图的“已重试模型请求 5/5、重试延迟 8638/7356 毫秒、失败原因、本轮运行失败”结构一致。
   - https://github.com/deepseek-ai/deepseek-harness/discussions/8840 ：默认 `retryPolicy = normal`，5 次尝试，可重试码包含 `EMPTY_RESPONSE / RATE_LIMIT / SERVER / TIMEOUT / TRANSPORT`，退避 500 毫秒起倍增封顶 10 秒并带抖动。
   - https://github.com/deepseek-ai/deepseek-harness/discussions/4657 ：429 会被归为 `RATE_LIMIT` 并进入重试白名单。
   - https://github.com/deepseek-ai/deepseek-harness/discussions/4431 ：客户端把 429 一类错误统一按 `RATE_LIMIT` 处理。
   - https://github.com/deepseek-ai/deepseek-harness/discussions/6987 ：非 2xx 的错误码映射包含 `RATE_LIMIT`，并说明重试上限与退避细节。
4. 推断：截图中 8638 与 7356 毫秒接近 8000 毫秒的 ±10% 抖动区间，即退避序列 500 / 1000 / 2000 / 4000 / 8000 毫秒的最后一级，与 DSH 默认退避一致。此为推断，不是官方断言。
5. 没有检索到 `ttft_admission_refused` 与具体服务商的对应关系；上游身份必须由生产取证确认。

## 8. 风险与不确定点

1. 客户端 provider 指向未知：如果 DSH 的该模型 provider 直连上游，而没有经过 new-api，则 new-api 不在链路中；需用户或生产配置确认 baseURL。这是本分析最大的不确定点。
2. 截图 JSON 是客户端的展示形态，客户端可能只展示内层 error 对象；不能 100% 断言 new-api 的原始 body 就是扁平结构。但字段集合与顺序、以及 type/code 保留逻辑与 new-api 代码一致。
3. 上游身份未知：`ttft_admission_refused` 既可能来自模型服务商，也可能来自上游中转网关；new-api 渠道类型、base_url、模型映射决定该判断，需 prod-ops 证据。
4. 生产配置未知：`RetryTimes`、`AutomaticRetryStatusCodes`、`AutomaticDisableChannelEnabled`、`AutomaticDisableStatusCodeRanges`、渠道 `status_code_mapping`、亲和规则是否被改过，均会改变链路表现；默认值分析与生产实际可能不同。
5. 若生产把 429 加入自动禁用状态码，渠道可能已被禁用，用户看到的现象会随渠道状态变化；需与渠道表 status 对照。
6. 本分析未执行任何请求，未验证线上返回体；结论基于代码与公开资料。

## 9. 需 Lead 决策项

1. 是否让 prod-ops 在只读取证中加入 DSH 配置核对：确认 DSH 中该模型 provider 的 baseURL 是否指向 new-api，以排除“客户端直连上游”的可能。此项需要用户授权或用户直接提供配置。
2. 是否让 prod-ops 只读查询生产库中 `options` 表的 `RetryTimes`、`AutomaticDisableChannelEnabled`、`AutomaticRetryStatusCodes`、`AutomaticDisableStatusCodeRanges`，以及目标渠道的 `auto_ban`、`priority`、`weight`、`status_code_mapping`、`status`。
3. 处置方向需在取得授权后决定：等待上游恢复；临时把该模型路由到其他渠道或调整权重；临时关闭该渠道；或调整网关重试与禁用策略。本任务不实施任何修改。
4. 是否另开实现任务做网关侧增强：把 429 的 `Retry-After` 透传给客户端、对上游容量拒绝做渠道转移或熔断、错误日志补齐上游身份。属代码改动，需新任务与明确授权。
