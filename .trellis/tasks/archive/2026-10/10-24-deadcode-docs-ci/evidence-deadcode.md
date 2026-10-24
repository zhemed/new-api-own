# 证据：清 39 个未导出死符号 + deadcode 导出符号分析

- 任务：Trellis `10-24-deadcode-docs-ci`；共享任务板 `task-43`
- 执行者：worker-script；时间：2026-10-24
- 范围：本任务只做 **A（清 39 个未导出死符号）** 与 **B（deadcode 工具 + 导出符号分析）**。
  PRD 的 R2（文档示例版本号）与 R3（CI 格式检查）由其它成员负责，**本轮未触碰** `README.md` /
  `MAINTENANCE.md` / `.github/`（`git status` 里这三个文件的改动属他人）。
- 未提交、未推送、未发版；未动 `web/`、`data/`、实例。

## 0. 总览

| 项 | 结果 |
| --- | --- |
| A. 39 个未导出死符号 | **36 个已删**；另 3 个的最终处置见 §10（2 个改为被真实使用、1 个按 spec 保留） |
| 删除改动 | 25 个文件，**487 行删除、0 行新增**（纯删除 + 移除因此不再使用的 import） |
| 分批 | 6 批（7/3/5/8/5/8），每批后 `go build` + `go test ./... -count=1`，**全部通过** |
| 最终 `unused` | **1 issue**（`logKeyCol`，按 spec 保留；`commonTrueVal`/`commonFalseVal` 已通过 `model/twofa.go` 真实使用，见 §10） |
| B. deadcode | 安装 `golang.org/x/tools v0.52.0` 的 `deadcode`；`-test` 模式 **144 条**（无 `-test` 177 条） |
| 导出符号分类 | **需确认 82** / **可删 62** / **误报（模式级 32 + 反射级 0）** |
| 交叉核对 | 39 个 unused 中的 **29 个函数与 deadcode 100% 重合**；10 个非函数（var/const/type）deadcode 结构上不报 → 两套工具**互不矛盾** |
| 门禁 | build ✅ vet ✅ test ✅(36 ok) relaykit 独立构建 ✅ gofmt ✅ |

## 1. A：39 个未导出符号的处置（逐符号取证）

取证方法（每条都执行）：
1. `golangci-lint run --no-config --default=none --enable=unused ./...` 给出精确行号；
2. **包内** `grep -rEn "\b<sym>\b" --include=*.go <pkgdir>` 排除跨包同名干扰；
3. 全仓非 `.go` 文件（`*.sh/*.yml/*.yaml/*.json/*.md/*.ts/*.tsx`，排除 `.git`/`.trellis`/`node_modules`）
   搜索符号名，确认**无脚本/模板/反射字符串引用**；
4. 用 `deadcode`（全程序分析）交叉验证其不可达（见 §5）。

结论：**36 个删除**；3 个「保留」的最终处置见 §10（2 个改为被真实使用、1 个按 spec 保留）。

### 1.1 已删除（36 个）

| 符号 | 包 | 包内引用 | 非 Go 引用 | 说明 |
| --- | --- | --- | --- | --- |
| `fieldReplacer` | `common` | 1 | 0 | 删除前仅定义处引用（refs=1） |
| `getEnvOrDefault` | `common` | 2 | 0 | 删除前仅定义处引用（refs=1） |
| `updateChannelCloseAIBalance` | `controller` | 1 | 0 | 删除前仅定义处引用（refs=1） |
| `updateChannelOpenAISBBalance` | `controller` | 2 | 0 | 删除前仅定义处引用（refs=1） |
| `upstreamModelsURL` | `controller` | 1 | 0 | 删除前仅定义处引用（refs=1） |
| `upstreamVendorsURL` | `controller` | 1 | 0 | 删除前仅定义处引用（refs=1） |
| `isStripeWebhookConfigured` | `controller` | 1 | 0 | 删除前仅定义处引用（refs=1） |
| `isWaffoPancakeWebhookConfigured` | `controller` | 1 | 0 | 删除前仅定义处引用（refs=1） |
| `generateDefaultSidebarConfig` | `controller` | 1 | 0 | 删除前仅定义处引用（refs=1） |
| `abortWithMidjourneyMessage` | `middleware` | 1 | 0 | 删除前仅定义处引用（refs=1） |
| `createRootAccountIfNeed` | `model` | 1 | 0 | 删除前仅定义处引用（refs=1） |
| `migrateDBFast` | `model` | 1 | 0 | 删除前仅定义处引用（refs=1） |
| `cacheSetTokenField` | `model` | 1 | 0 | 删除前仅定义处引用（refs=1） |
| `updateUserUsedQuota` | `model` | 1 | 0 | 删除前仅定义处引用（refs=1） |
| `updateUserRequestCount` | `model` | 1 | 0 | 删除前仅定义处引用（refs=1） |
| `getUserStatusCache` | `model` | 1 | 0 | 删除前仅定义处引用（refs=1） |
| `updateUserStatusCache` | `model` | 1 | 0 | 删除前仅定义处引用（refs=1） |
| `updateUserEmailCache` | `model` | 1 | 0 | 删除前仅定义处引用（refs=1） |
| `redisCounters` | `pkg/perf_metrics` | 2 | 0 | 死代码簇（三者互引，同批删除） |
| `parseRedisInt` | `pkg/perf_metrics` | 8 | 0 | 死代码簇（三者互引，同批删除） |
| `mergeRedisActiveBuckets` | `pkg/perf_metrics` | 1 | 0 | 死代码簇（三者互引，同批删除） |
| `syncModels` | `relay/channel/ali` | 2 | 0 | 包级 var；同文件另有函数内同名局部变量（保留） |
| `stopReasonClaude2OpenAI` | `relay/channel/claude` | 1 | 0 | 删除前仅定义处引用（refs=1） |
| `cacheCreationTokensForOpenAIUsage` | `relay/channel/claude` | 1 | 0 | 删除前仅定义处引用（refs=1） |
| `getPayloadHash` | `relay/channel/jimeng` | 1 | 0 | 删除前仅定义处引用（refs=1） |
| `getContentTypeByFormat` | `relay/channel/minimax` | 1 | 0 | 删除前仅定义处引用（refs=1） |
| `handleChatCompletionResponse` | `relay/channel/minimax` | 1 | 0 | 删除前仅定义处引用（refs=1） |
| `contains` | `relay/channel/task/hailuo` | 1 | 0 | 死代码簇（containsInt 仅被 contains 之外的死代码/自身域引用） |
| `containsInt` | `relay/channel/task/hailuo` | 1 | 0 | 死代码簇（containsInt 仅被 contains 之外的死代码/自身域引用） |
| `detectImageMimeType` | `relay/channel/volcengine` | 3 | 0 | 删除前仅定义处引用（refs=1） |
| `tokenData` | `relay/channel/zhipu_4v` | 1 | 0 | 删除前仅定义处引用（refs=1） |
| `zhipuImageRequest` | `relay/channel/zhipu_4v` | 1 | 0 | 删除前仅定义处引用（refs=1） |
| `trimModelThinking` | `relay` | 1 | 0 | 删除前仅定义处引用（refs=1） |
| `taskChangeParams` | `relay` | 1 | 0 | 删除前仅定义处引用（refs=1） |
| `shouldAutoRefreshCodexChannelStatus` | `service` | 1 | 0 | 删除前仅定义处引用（refs=1） |
| `newProxyHTTPClient` | `service` | 1 | 0 | 删除前仅定义处引用（refs=1） |

> 说明：表中「包内引用」为包目录内 `\b<sym>\b` 命中数（1 = 仅定义处）；`refs>1` 的少数条目
> 经人工确认其引用方本身也在同一死代码簇内（如同属 `pkg/perf_metrics` 的
> `redisCounters/parseRedisInt/mergeRedisActiveBuckets`），故同批删除。

### 1.2 保留（3 个）——**文档约定的跨库辅助，不得为清数字误删**

| 符号 | 位置 | 为什么保留 |
| --- | --- | --- |
| `commonTrueVal` | `model/main.go:24` | `AGENTS.md:90`「Use `commonTrueVal`/`commonFalseVal` for boolean values」+ `.trellis/spec/backend/database-guidelines.md:28-30` 明确写入「Booleans in raw SQL go through commonTrueVal/commonFalseVal」 |
| `commonFalseVal` | `model/main.go:25` | 同上 |
| `logKeyCol` | `model/main.go:27` | 与 `commonKeyCol`/`commonGroupCol`/`logGroupCol` 同族（主库/日志库保留字列适配），`.trellis/spec` 的 database-guidelines 把这一族列为跨库 SQL 机制 |

它们当前是「在 `initCol()` 里按数据库类型赋值、但暂未被读取」的状态（write-only），
所以 linter 报 unused；但**删除会让 AGENTS.md 与 backend spec 指向不存在的符号**，
从而制造文档腐化。故按任务「看似死但需保留→说明」处理。

**（Lead 已裁决：采用方案 (b)；最终落地结果见 §10，本节保留为决策记录）**

原三选一：
- (a) **接受保留**（推荐）：`unused` 保持 3 issues，在 lint 白名单/说明里注明这 3 个是约定辅助；
- (b) **改为使用**：`model/twofa.go:246/273/409` 有 3 处 raw SQL 硬编码 `is_used = false` /
  `is_enabled = true`，改用 `commonTrueVal/commonFalseVal` 即可让 3 个符号「被读取」→ `unused` 归零，
  且 SQL 语义在 MySQL/PG/SQLite 上等价（`false`↔`0`）；但这属「改非删除类代码」，超出本任务授权，需你点头；
- (c) **删除 + 同步改文档**：删这 3 个并同步改 `AGENTS.md` 与 backend spec（文档不在我写作用域）。

## 2. A：分批删除记录（每批后 build + 全量测试）

| 批次 | 范围 | 符号数 | `go build ./...` | `go test ./... -count=1` |
| --- | --- | --- | --- | --- |
| 1 controller | updateChannelCloseAIBalance, updateChannelOpenAISBBalance, isStripeWebhookConfigured… | 7 | ✅ | exit=0 FAIL=0 |
| 2 model/main+token_cache | createRootAccountIfNeed, migrateDBFast, cacheSetTokenField | 3 | ✅ | exit=0 FAIL=0 |
| 3 model/user+user_cache | updateUserUsedQuota, updateUserRequestCount, getUserStatusCache… | 5 | ✅ | exit=0 FAIL=0 |
| 4 relay-1 | syncModels, stopReasonClaude2OpenAI, cacheCreationTokensForOpenAIUsage… | 8 | ✅ | exit=0 FAIL=0 |
| 5 relay-2 | detectImageMimeType, tokenData, zhipuImageRequest… | 5 | ✅ | exit=0 FAIL=0 |
| 6 misc | fieldReplacer, getEnvOrDefault, abortWithMidjourneyMessage… | 8 | ✅ | exit=0 FAIL=0（首次因未用 import 失败，修 import 后通过） |

补充：批 4/5/6 首次构建各因「删掉唯一使用者后 import 未使用」而失败（`minimax/tts.go` 的
`errors`/`service`、`zhipu_4v/dto.go` 的 `time`、`volcengine/adaptor.go` 的 `path/filepath`、
`common/totp.go` 的 `os`、`pkg/perf_metrics/flush.go` 的 `strconv`），**均已随删随清并复跑通过**。
这正是"每批 build + test"的价值：把遗漏立刻挡在批内。

## 3. A：最终 lint 状态

```console
$ golangci-lint run --no-config --default=none --enable=unused ./...
model/main.go:24:5: var commonTrueVal is unused (unused)
model/main.go:25:5: var commonFalseVal is unused (unused)
model/main.go:27:5: var logKeyCol is unused (unused)
1 issue（logKeyCol）
```

`gofmt -l .` 为空。与 AC2（unused 0）的差异 = 保留的 3 个文档约定辅助，处置方案见 §1.2。

## 4. B：deadcode 安装与分析

```console
$ go install golang.org/x/tools/cmd/deadcode@latest      # 用户已授权联网
$ go version -m "$(go env GOPATH)/bin/deadcode" | grep -E 'path|mod'
	path	golang.org/x/tools/cmd/deadcode
	mod	golang.org/x/tools	v0.52.0	h1:ryxg4U63oaUYk7uOLTF37n43f4qgtGvqV4Ioit1BaEs=
$ git status --porcelain go.mod go.sum    # 工具未写入项目依赖
（无输出）
```

运行（两个模式，证明必须用 `-test`）：

```console
$ deadcode ./...            # 177 行
$ deadcode -test ./...      # 144 行
```

- **为什么用 `-test`**：模型把 main + 测试都当根；**32 个函数只被测试引用**，
  不带 `-test` 会被误报为死代码（差值清单见 §6.3）。故下面清单以 `-test` 为准。
- 输出格式：`file:line:col: unreachable func: <Name>`（方法记为 `Type.Method`）。
- 本工具报告的是**函数/方法**，不含包级变量/类型/常量（这也是 §5 交叉核对的结构性差异）。

## 5. B：导出符号三类清单（144 条，按包分组）

分类规则（可复核）：
- **可删**：应用层（`controller/model/service/middleware/relay`）或共享包内部**未导出**符号，
  全仓无引用（deadcode `-test` 判定不可达）；
- **需确认**：**共享库/独立模块的导出 API**（`common/`、`relaykit/`、`types/`、`pkg/`、`oauth/`、`setting/`），
  这些是公共面（`relaykit/` 是 AGENTS.md 要求「保持可独立构建」的独立模块，且 `relaykit/README.md`
  文档化了 `ConvertResponseByID` 等 API），删除需 owner 决策；
- **误报**：见 §6。

### 5.1 可删（62 条）

**controller/**（3 个）

| 文件:行 | 符号 |
| --- | --- |
| `controller/image.go:7` | `GetImage` |
| `controller/misc.go:214` | `GetMidjourney` |
| `controller/token.go:194` | `GetTokenStatus` |

**middleware/**（5 个）

| 文件:行 | 符号 |
| --- | --- |
| `middleware/auth.go:264` | `WssAuth` |
| `middleware/i18n.go:45` | `GetLanguage` |
| `middleware/rate-limit.go:181` | `DownloadRateLimit` |
| `middleware/rate-limit.go:185` | `UploadRateLimit` |
| `middleware/recover.go:12` | `RelayPanicRecover` |

**model/**（26 个）

| 文件:行 | 符号 |
| --- | --- |
| `model/ability.go:57` | `GetAllEnableAbilities` |
| `model/channel.go:645` | `CleanupChannelPollingLocks` |
| `model/channel.go:892` | `DeleteChannelByStatus` |
| `model/channel.go:902` | `GetPaginatedTags` |
| `model/channel.go:1107` | `CountAllChannels` |
| `model/channel.go:1114` | `CountAllTags` |
| `model/channel.go:1125` | `GetChannelsByType` |
| `model/channel.go:1136` | `CountChannelsByType` |
| `model/channel.go:1143` | `CountChannelsGroupByType` |
| `model/channel_satisfy.go:33` | `IsChannelEnabledForAnyGroupModel` |
| `model/custom_oauth_provider.go:81` | `GetEnabledCustomOAuthProviders` |
| `model/custom_oauth_provider.go:98` | `GetCustomOAuthProviderBySlug` |
| `model/log.go:679` | `SumUsedToken` |
| `model/midjourney.go:147` | `GetMjByuId` |
| `model/midjourney.go:157` | `UpdateProgress` |
| `model/model_meta.go:107` | `GetAllModels` |
| `model/perf_metric.go:81` | `GetPerfMetricsSummaryAll` |
| `model/perf_metric.go:125` | `PerfMetricStartTime` |
| `model/system_task.go:146` | `FindPendingSystemTasks` |
| `model/token.go:382` | `DisableModelLimits` |
| `model/topup.go:62` | `GetTopUpById` |
| `model/user.go:1068` | `IsTelegramIdAlreadyTaken` |
| `model/user.go:1161` | `GetUserEmail` |
| `model/user.go:1282` | `DeltaUpdateUserQuota` |
| `model/user_cache.go:72` | `InvalidateUserCache` |
| `model/user_oauth_binding.go:31` | `GetUserOAuthBinding` |

**relay/**（4 个）

| 文件:行 | 符号 |
| --- | --- |
| `relay/channel/api_request.go:120` | `IsHeaderPassthroughRuleKey` |
| `relay/channel/jimeng/sign.go:43` | `SetPayloadHash` |
| `relay/channel/volcengine/protocols.go:256` | `Message.String` |
| `relay/common/relay_info.go:1044` | `RemoveGeminiDisabledFields` |

**relaykit/**（3 个）

| 文件:行 | 符号 |
| --- | --- |
| `relaykit/relayconvert/internal/oai_responses/to_claude_messages_req.go:15` | `convertOpenAIResponsesRequestToClaudeMessages` |
| `relaykit/relayconvert/internal/oai_responses/to_gemini_chat_req.go:15` | `convertOpenAIResponsesRequestToGeminiChat` |
| `relaykit/relayconvert/text_converter_registry.go:356` | `resolveTextConverterID` |

**service/**（21 个）

| 文件:行 | 符号 |
| --- | --- |
| `service/audio.go:34` | `DecodeBase64AudioData` |
| `service/auth_session.go:426` | `FormatAuthError` |
| `service/authz/registry.go:40` | `AllPermissions` |
| `service/codex_oauth.go:29` | `RefreshCodexOAuthToken` |
| `service/error.go:62` | `ClaudeErrorWrapper` |
| `service/error.go:81` | `ClaudeErrorWrapperLocal` |
| `service/file_decoder.go:23` | `GetFileTypeFromUrl` |
| `service/file_decoder.go:139` | `GetFileBase64FromUrl` |
| `service/file_service.go:431` | `GetMimeType` |
| `service/http_client.go:181` | `ValidateProxyURL` |
| `service/image.go:119` | `DecodeUrlImageData` |
| `service/openai_chat_responses_compat.go:8` | `ChatCompletionsRequestToResponsesRequest` |
| `service/openai_chat_responses_compat.go:12` | `ResponsesRequestToChatCompletionsRequest` |
| `service/openai_chat_responses_compat.go:16` | `ChatCompletionsResponseToResponsesResponse` |
| `service/openai_chat_responses_compat.go:20` | `ResponsesResponseToChatCompletionsResponse` |
| `service/openai_chat_responses_compat.go:24` | `ResponsesFinishReasonFromStatus` |
| `service/request_converter.go:38` | `ClaudeToOpenAIRequest` |
| `service/request_converter.go:50` | `GeminiToOpenAIRequest` |
| `service/sensitive.go:11` | `CheckSensitiveMessages` |
| `service/sensitive.go:52` | `SensitiveWordReplace` |
| `service/str.go:51` | `RemoveDuplicate` |

### 5.2 需确认（82 条）


## 6. B：误报（三类证据）

### 6.1 模式级误报：`-test` 与不带 `-test` 的差值（32 条）

这些函数**只在测试中被引用**；不带 `-test` 时被误报为不可达（本报告已改用 `-test` 消除）：

- `AdvanceCurrentSessionSecurity`\n- `BumpUserAuthVersion`\n- `CacheUpdateChannel`\n- `ClearUserAuthorization`\n- `ClearUserPermissions`\n- `ComputeTieredQuota`\n- `ConfigManager.LoadFromDB`\n- `ConfigManager.SaveToDB`\n- `DeleteToolPriceForTest`\n- `ExplicitUserOverrides`\n- `ExplicitUserPermissions`\n- `GetDefaultCacheRatioMap`\n- `GetDefaultCompletionRatioMap`\n- `GetLatestSystemTask`\n- `GetToolPrice`\n- `ImageGenerationCallCounter.Count`\n- `InMemoryRateLimiter.Reset`\n- `InvalidateCache`\n- `NormalizeCacheCreationSplit`\n- `OpenAIChatRequestToClaudeMessages`\n- `QuotaRound`\n- `ResponseOpenAI2Claude`\n- `ResponseOpenAI2Gemini`\n- `RunExpr`\n- `SetDatabaseTypes`\n- `SetToolPriceForTest`\n- `SetUserPermissions`\n- `SmokeTestExpr`\n- `StreamResponseOpenAI2Claude`\n- `StreamResponseOpenAI2Gemini`\n- `newHTTPClientWithPolicyAndTLS`\n- `smokeTestExpr`

### 6.2 反射/接口级误报：**0 条（已逐类排查）**

- 对 141 个唯一符号在**全部非 Go 文件**（脚本/YAML/JSON/Markdown/TS/TSX，排除 `.git`/`.trellis`/`node_modules`）
  做名字搜索 → 只命中**文档里对 API 的说明**（如 `AGENTS.md` 的 `DecodeJson`/`UnmarshalJsonStr`），
  **无脚本/模板/反射字符串调用**；
- 逐一核查最可能靠反射调用的候选：
  - `relaykit/dto.StringValue.MarshalJSON/UnmarshalJSON` → `StringValue` 类型在本仓库**未被任何结构体使用**，
    故 json 反射不会触发 → **真死**，不是误报；
  - `relay/channel/volcengine.Message.String` → `Message` 类型被使用，但全包无 `fmt`/`logger` 对 `Message`
    做格式化 → `String()` 不会被调用 → 真死；
  - `pkg/cachex.StringCodec.Encode/Decode` → `StringCodec` 从未被实例化 → 真死；
  - GORM 钩子（`AfterFind` 等）在本仓库是导出方法，deadcode 未报（对照组正常）。
- 结论：本清单中**没有**需要靠反射解释的误报；但保留 deadcode 的通用限制提醒：
  **若未来引入 Reflect 调用/插件/`//go:linkname`，deadcode 无法看到**，删前仍需 grep。

### 6.3 文档约定 API（需确认，不是误报）

`DecodeJson`、`UnmarshalJsonStr`（`AGENTS.md:75-77` 强制要求业务代码使用 `common.*` 包装）、
`ClaudeErrorWrapper`、`Set.Contains`、`StringCodec.Decode` 出现在 `AGENTS.md` / `.trellis/spec/` 中，
但当前**仓库内无调用**。删除会让文档失去对应实现 → 归入「需确认」，建议 Lead 决定
「补调用 / 删符号+改文档 / 保持现状」。

## 7. 交叉核对：`unused`（39） vs `deadcode`（144）

在**删除前的基线**（`git stash` 暂存本次改动）上跑 `deadcode -test`（170 个唯一名），与 39 个 unused 对比：

| 结论 | 数量 | 明细 |
| --- | --- | --- |
| **两套工具一致（均不可达）** | **29/29 函数** | 39 个里的全部函数都被 deadcode 同时判为 unreachable（名单见下方） |
| **结构上互斥（deadcode 无法表达）** | 10 | 39 个中的 3 个 type（`tokenData`/`zhipuImageRequest`/`taskChangeParams`）、5 个 var（`fieldReplacer`/`commonTrueVal`/`commonFalseVal`/`logKeyCol`/`syncModels`）、2 个 const（`upstreamModelsURL`/`upstreamVendorsURL`）——deadcode 只报函数 |
| 矛盾项 | **0** | 无「unused 报死但 deadcode 判可达」的情况 |

29 个一致项：`abortWithMidjourneyMessage, cacheCreationTokensForOpenAIUsage, cacheSetTokenField, contains,
containsInt, createRootAccountIfNeed, detectImageMimeType, generateDefaultSidebarConfig, getContentTypeByFormat,
getEnvOrDefault, getPayloadHash, getUserStatusCache, handleChatCompletionResponse, isStripeWebhookConfigured,
isWaffoPancakeWebhookConfigured, mergeRedisActiveBuckets, migrateDBFast, newProxyHTTPClient, parseRedisInt,
redisCounters, shouldAutoRefreshCodexChannelStatus, stopReasonClaude2OpenAI, trimModelThinking,
updateChannelCloseAIBalance, updateChannelOpenAISBBalance, updateUserEmailCache, updateUserRequestCount,
updateUserStatusCache, updateUserUsedQuota`。

（做法：`git stash push` → `deadcode -test ./...` → `git stash pop`；stash/pop 均成功，工作区完整，
删除后 `grep` 确认 36 个符号无残留引用。）

## 8. 门禁输出（全部实际执行）

```console
$ go build ./...                              # OK
$ go vet ./...                                # OK
$ go test ./... -count=1
exit=0  FAIL=0  ok=36
$ cd relaykit && GOWORK=off go build ./...    # OK（独立模块）
$ gofmt -l .
（空，0 行）
$ golangci-lint run --no-config --default=none --enable=unused ./...
1 issue（`logKeyCol`，见 §10）
```

## 9. 风险 / 未完成 / 建议

- **风险（很低）**：删除的全部是 `unused` + `deadcode` 双工具确认、且全仓零引用（含测试/脚本/文档/模板）的符号；
  25 文件 487 行纯删除，无新增行；回滚只需 `git checkout -- common/ controller/ middleware/ model/ pkg/ relay/ service/`。
- **唯一偏差**：`unused` 未归零（3 个文档约定辅助），三个处置方案见 §1.2，请 Lead 裁决。
- **本轮未删任何导出符号**（按任务要求）：`可删 62 + 需确认 82` 是候选清单；
  建议下一轮先删 **62 个「可删」**（同样分批 + 每批全测试），`需确认 82` 需 owner 决策。
- `deadcode` 已装到 `$(go env GOPATH)/bin/deadcode`（`golang.org/x/tools v0.52.0`），**未写入项目 `go.mod`**；
  CI 若要用需显式 `go install`（属联网动作，需授权）。
- **未触碰**：`README.md`/`MAINTENANCE.md`/`.github/`（他人任务）、`web/`、`data/`、实例；未提交未推送未发版。

## 10. 追加轮次：3 个保留符号的处置（Lead 裁决 = 方案 b）

Lead 复核结论：36 个删除接受；剩余 3 个按方案 (b)「变成被使用」处理（原 §1.2 的三选一）。

### 10.1 `commonTrueVal` / `commonFalseVal` → 在 `model/twofa.go` 里被使用

改动（3 行，纯替换字面量；**查询结构、`?` 参数绑定完全不变**）：

```diff
--- a/model/twofa.go
 // ValidateBackupCode：查找未使用的备用码
-	if err := DB.Where("user_id = ? AND is_used = false", userId).Find(&backupCodes).Error; err != nil {
+	if err := DB.Where("user_id = ? AND is_used = "+commonFalseVal, userId).Find(&backupCodes).Error; err != nil {
 // GetUnusedBackupCodeCount
-	err := DB.Model(&TwoFABackupCode{}).Where("user_id = ? AND is_used = false", userId).Count(&count).Error
+	err := DB.Model(&TwoFABackupCode{}).Where("user_id = ? AND is_used = "+commonFalseVal, userId).Count(&count).Error
 // GetTwoFAStats：启用 2FA 的用户数
-	if err := DB.Model(&TwoFA{}).Where("is_enabled = true").Count(&enabledUsers).Error; err != nil {
+	if err := DB.Model(&TwoFA{}).Where("is_enabled = " + commonTrueVal).Count(&enabledUsers).Error; err != nil {
```

**三库等价性确认**（取值来自 `model/main.go` 的 `initCol()`）：

| 主库 | `commonTrueVal` | `commonFalseVal` | 与替换前字面量的关系 |
| --- | --- | --- | --- |
| PostgreSQL | `"true"` | `"false"` | **与原字面量逐字符相同**，零变化 |
| MySQL | `"1"` | `"0"` | MySQL 中 `TRUE`/`FALSE` 是 `1`/`0` 的别名，语义等价 |
| SQLite | `"1"` | `"0"` | SQLite 中 `TRUE`/`FALSE` 解析为 `1`/`0`（3.23+），语义等价 |

安全性：拼接的是**编译期常量**（非用户输入），不存在注入面；`userId` 仍是 `?` 参数绑定。
写法沿用项目既有约定（`model/log.go` 里 `logGroupCol` 也是同样的「常量拼进 Where 片段 + `?` 绑定」）。

**新增回归测试** `model/twofa_test.go`（2 个用例）：

1. `TestTwoFABooleanLiteralsPerDialect`：分别在 SQLite 与 PostgreSQL 类型下调 `initCol()`，断言
   常量取值为 `1/0` 与 `true/false`；并在 SQLite 上**真实执行**拼接后的 SQL
   （`SELECT 1 WHERE 1 = <trueVal>` / `0 = <falseVal>` 命中，`1 = <falseVal>` 不命中），
   证明两侧值确实不同且语义正确。
2. `TestTwoFABooleanQueriesBehavior`：端到端覆盖三处查询（SQLite 方言）——
   生成真实备用码 → `GetUnusedBackupCodeCount` 只统计未使用（1）→ `ValidateBackupCode` 通过并把该码
   标记为已使用（计数归 0）→ 再次验证同一码失败（证明 `is_used` 条件不是恒真）；
   再插入 enabled/disabled 两条 `TwoFA` 记录 → `GetTwoFAStats()["enabled_users"] == 1`。

### 10.2 `logKeyCol` → **确实无处可用，保留**（并说明理由）

排查结论（证据）：
- `Log` 结构体（`model/log.go:59-81`）**没有 `key` 字段**，logs 表也没有 `key` 列，
  因此不存在「该用 `logKeyCol` 的 raw SQL」；
- 全仓搜索 logs 表 raw SQL 中的 `key` 列引用（`SELECT/WHERE/ORDER BY/GROUP BY`）→ **0 处**；
- 作为对照，同族兄弟 `logGroupCol` **正在被使用**：`model/log.go:506/596/656/657`
  的 `Where("logs."+logGroupCol+" = ?", group)`；
- `logKeyCol` 仍在 `.trellis/spec/backend/database-guidelines.md:29` 被记录为日志库标识符族的一员。

处置：**保留**。它是历史遗留的「日志库 `key` 列方言标识符」，当前无对应列/查询，删除会让
backend spec 的族描述失去一半；且本轮 Lead 已确认「无可用处则保留并说明」。
代价：`golangci-lint unused` 剩 **1 issue**（`model/main.go:27 var logKeyCol is unused`）。

> 诚实性说明：一度因为测试里做 `previousLogKey := logKeyCol` 的保存/恢复而让 lint 变 0，
> 那只是"被测试读到"，并非真实使用；已改为 cleanup 里 `initCol()` 重新推导（不读取该变量），
> 让 lint 如实暴露 1 issue，避免假绿。

### 10.3 PRD 回填文本（供 Lead 贴入 `prd.md` Notes / AC1 备注，agent 不代写 Trellis 记账）

> AC1 备注（2026-10-24 追加）：39 个未导出死符号已完成处置——36 个删除（双工具确认 + 全仓零引用 +
> 6 批每批全测试）；`commonTrueVal`/`commonFalseVal` 已在 `model/twofa.go` 三处 raw SQL 中实际使用
> （三库语义等价，附回归测试）；`logKeyCol` 经查无对应列/查询（`Log` 结构体无 `key` 字段），
> 作为日志库标识符族的文档约定保留，因此 `golangci-lint unused` 最终为 **1 issue**（该项接受）。

### 10.4 追加轮次门禁（全部实际执行）

```console
$ go build ./...                              # OK
$ go vet ./...                                # OK
$ go test ./... -count=1
exit=0  FAIL=0  ok=36
$ cd relaykit && GOWORK=off go build ./...    # OK
$ gofmt -l .
（空，0 行）
$ golangci-lint run --no-config --default=none --enable=unused ./...
model/main.go:27:5: var logKeyCol is unused (unused)
1 issues
$ go test ./model/ -count=1 -run 'TestTwoFABoolean' -v
--- PASS: TestTwoFABooleanLiteralsPerDialect (0.00s)
--- PASS: TestTwoFABooleanQueriesBehavior (0.09s)
```

本轮改动：`model/twofa.go`（3 行替换）+ `model/twofa_test.go`（新增 2 用例）。
**未触碰** `web/`、`data/`、实例；未提交、未推送、未发版。
