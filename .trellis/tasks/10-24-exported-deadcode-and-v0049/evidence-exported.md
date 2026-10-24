# 证据：清 59 个「可删」导出死符号（四类核对 + 分批 + 级联分析）

- 任务：Trellis `10-24-exported-deadcode-and-v0049`；共享任务板 `task-47`
- 执行者：worker-script；时间：2026-10-24
- 对象：上轮 `deadcode -test ./...` 报告 §5.1 的 **62 个「可删」导出符号**
- 结果：**59 个删除、3 个保留并说明**；41 文件变更（39 改 + 2 整文件删除），**805 行删除 / 3 行新增**
- 未提交、未推送、**未发版**（由 Lead 验证后发 v0.0.49）；未动 `web/`、`data/`、实例；`go.mod`/`go.sum` 零改动

## 1. 四类核对方法（每个符号逐一执行）

| 类别 | 做法 |
| --- | --- |
| ① deadcode | `~/go/bin/deadcode -test ./...` 判不可达（`-test` 把测试也当根，避免"仅测试引用"误报） |
| ② 全仓 Go 引用 | 包内 `grep -rEn "\b<sym>\b" --include=*.go <pkgdir>`（排除跨包同名干扰）＋全仓 Go 计数；`--include=*.go` 覆盖所有构建标签文件 |
| ③ 非 Go 引用 | `README(.en).md`、`MAINTENANCE.md`、`AGENTS.md`、`CLAUDE.md`、`NOTICE`、`.trellis/spec/**`、`scripts/`、`.github/`、`web/src`、`web/scripts` |
| ④ 反射/接口/注册表 | 名字钩子表（GORM `After*/Before*`、`TableName`、`Scan/Value`、`Marshal*/Unmarshal*`、`String/Error`、`ServeHTTP`、`Len/Less/Swap`）+ 同文件 `go:linkname`/反射字符串排查 |

判定：**四类全满足才删**；任一不满足 → 保留 + 说明。最终 **59 删 / 3 留**。

### 1.1 逐符号取证与处置（62 个）

| 文件:行 | 符号 | ② Go 引用(包内/全仓) | ③ 非 Go | ④ 钩子 | 处置 |
| --- | --- | --- | --- | --- | --- |
| `controller/image.go:7` | `GetImage` | 1/1 | 0 | — | 删除 |
| `controller/misc.go:214` | `GetMidjourney` | 1/2 | 0 | — | 删除 |
| `controller/token.go:194` | `GetTokenStatus` | 1/1 | 0 | — | 删除 |
| `middleware/auth.go:264` | `WssAuth` | 1/1 | 0 | — | 删除 |
| `middleware/i18n.go:45` | `GetLanguage` | 2/2 | 0 | — | 删除 |
| `middleware/rate-limit.go:181` | `DownloadRateLimit` | 1/1 | 0 | — | 删除 |
| `middleware/rate-limit.go:185` | `UploadRateLimit` | 1/1 | 0 | — | 删除 |
| `middleware/recover.go:12` | `RelayPanicRecover` | 1/1 | 0 | — | 删除 |
| `model/ability.go:57` | `GetAllEnableAbilities` | 1/1 | 0 | — | 删除 |
| `model/channel.go:645` | `CleanupChannelPollingLocks` | 2/2 | 0 | — | 删除 |
| `model/channel.go:892` | `DeleteChannelByStatus` | 1/1 | 0 | — | 删除 |
| `model/channel.go:902` | `GetPaginatedTags` | 1/1 | 0 | — | 删除 |
| `model/channel.go:1107` | `CountAllChannels` | 2/2 | 0 | — | 删除 |
| `model/channel.go:1114` | `CountAllTags` | 2/2 | 0 | — | 删除 |
| `model/channel.go:1125` | `GetChannelsByType` | 1/1 | 0 | — | 删除 |
| `model/channel.go:1136` | `CountChannelsByType` | 1/1 | 0 | — | 删除 |
| `model/channel.go:1143` | `CountChannelsGroupByType` | 1/1 | 0 | — | 删除 |
| `model/channel_satisfy.go:33` | `IsChannelEnabledForAnyGroupModel` | 1/1 | 0 | — | 删除 |
| `model/custom_oauth_provider.go:81` | `GetEnabledCustomOAuthProviders` | 2/2 | 0 | — | 删除 |
| `model/custom_oauth_provider.go:98` | `GetCustomOAuthProviderBySlug` | 2/2 | 0 | — | 删除 |
| `model/log.go:679` | `SumUsedToken` | 1/3 | 0 | — | 删除 |
| `model/midjourney.go:147` | `GetMjByuId` | 1/1 | 0 | — | 删除 |
| `model/midjourney.go:157` | `UpdateProgress` | 1/1 | 0 | — | 删除 |
| `model/model_meta.go:107` | `GetAllModels` | 1/1 | 0 | — | 删除 |
| `model/perf_metric.go:81` | `GetPerfMetricsSummaryAll` | 1/1 | 0 | — | 删除 |
| `model/perf_metric.go:125` | `PerfMetricStartTime` | 1/1 | 0 | — | 删除 |
| `model/system_task.go:146` | `FindPendingSystemTasks` | 1/1 | 0 | — | 删除 |
| `model/token.go:382` | `DisableModelLimits` | 1/1 | 0 | — | 删除 |
| `model/topup.go:62` | `GetTopUpById` | 1/1 | 0 | — | 删除 |
| `model/user.go:1068` | `IsTelegramIdAlreadyTaken` | 1/1 | 0 | — | 删除 |
| `model/user.go:1161` | `GetUserEmail` | 1/1 | 0 | — | 删除 |
| `model/user.go:1282` | `DeltaUpdateUserQuota` | 1/1 | 0 | — | 删除 |
| `model/user_cache.go:72` | `InvalidateUserCache` | 3/3 | 0 | — | 删除 |
| `model/user_oauth_binding.go:31` | `GetUserOAuthBinding` | 2/2 | 0 | — | 删除 |
| `relay/channel/api_request.go:120` | `IsHeaderPassthroughRuleKey` | 1/1 | 0 | — | 删除 |
| `relay/channel/jimeng/sign.go:43` | `SetPayloadHash` | 2/2 | 0 | — | 删除 |
| `relay/channel/volcengine/protocols.go:256` | `Message.String` | 0/0 | 0 | ⚠️ | **保留**（hook） |
| `relay/common/relay_info.go:1044` | `RemoveGeminiDisabledFields` | 4/4 | 0 | — | 删除 |
| `relaykit/relayconvert/text_converter_registry.go:356` | `resolveTextConverterID` | 2/2 | 0 | — | **保留**（pkg_refs=2） |
| `relaykit/relayconvert/internal/oai_responses/to_claude_messages_req.go:15` | `convertOpenAIResponsesRequestToClaudeMessages` | 1/3 | 0 | — | 删除 |
| `relaykit/relayconvert/internal/oai_responses/to_gemini_chat_req.go:15` | `convertOpenAIResponsesRequestToGeminiChat` | 1/3 | 0 | — | 删除 |
| `service/audio.go:34` | `DecodeBase64AudioData` | 1/1 | 0 | — | 删除 |
| `service/auth_session.go:426` | `FormatAuthError` | 1/1 | 0 | — | 删除 |
| `service/codex_oauth.go:29` | `RefreshCodexOAuthToken` | 1/1 | 0 | — | 删除 |
| `service/error.go:62` | `ClaudeErrorWrapper` | 2/2 | 1 | — | **保留**（pkg_refs=2；nongo=1: .trellis/spec/backend/error-handling.md:56:  Protocol-specific wrappers live next to it (`ClaudeErro） |
| `service/error.go:81` | `ClaudeErrorWrapperLocal` | 1/1 | 0 | — | 删除 |
| `service/file_decoder.go:23` | `GetFileTypeFromUrl` | 4/4 | 0 | — | 删除 |
| `service/file_decoder.go:139` | `GetFileBase64FromUrl` | 2/2 | 0 | — | 删除 |
| `service/file_service.go:431` | `GetMimeType` | 4/4 | 0 | — | 删除 |
| `service/http_client.go:181` | `ValidateProxyURL` | 2/2 | 0 | — | 删除 |
| `service/image.go:119` | `DecodeUrlImageData` | 1/1 | 0 | — | 删除 |
| `service/openai_chat_responses_compat.go:8` | `ChatCompletionsRequestToResponsesRequest` | 2/9 | 0 | — | 删除 |
| `service/openai_chat_responses_compat.go:12` | `ResponsesRequestToChatCompletionsRequest` | 2/14 | 0 | — | 删除 |
| `service/openai_chat_responses_compat.go:16` | `ChatCompletionsResponseToResponsesResponse` | 2/8 | 0 | — | 删除 |
| `service/openai_chat_responses_compat.go:20` | `ResponsesResponseToChatCompletionsResponse` | 2/10 | 0 | — | 删除 |
| `service/openai_chat_responses_compat.go:24` | `ResponsesFinishReasonFromStatus` | 2/8 | 0 | — | 删除 |
| `service/request_converter.go:38` | `ClaudeToOpenAIRequest` | 1/1 | 0 | — | 删除 |
| `service/request_converter.go:50` | `GeminiToOpenAIRequest` | 1/1 | 0 | — | 删除 |
| `service/sensitive.go:11` | `CheckSensitiveMessages` | 1/1 | 0 | — | 删除 |
| `service/sensitive.go:52` | `SensitiveWordReplace` | 2/2 | 0 | — | 删除 |
| `service/str.go:51` | `RemoveDuplicate` | 1/1 | 0 | — | 删除 |
| `service/authz/registry.go:40` | `AllPermissions` | 2/2 | 0 | — | 删除 |

> 说明：`②` 列为「包内命中数 / 全仓 Go 命中数」。包内=1 即只有定义处；部分符号全仓 >1 是因为**其它包存在同名符号**
> （例如 `service/openai_chat_responses_compat.go` 的包装函数与 `relaykit/relayconvert` 的同名函数是不同符号）。
> 少数包内 >1 的条目经逐行核对，其"额外引用"要么是**文档注释/自身日志**（如 `RemoveGeminiDisabledFields`
> 在自己的 `SysError` 文案里），要么来自**同批删除的死代码簇**（如 `GetMimeType`→`GetFileTypeFromUrl`）。

### 1.2 保留的 3 个及理由

| 符号 | 位置 | 触发类别 | 理由 |
| --- | --- | --- | --- |
| `Message.String` | `relay/channel/volcengine/protocols.go:256` | ④ 反射/接口 | 实现 `fmt.Stringer`；`Message` 类型**可达**，未来任何 `%v`/日志格式化都会经接口反射调用它，属典型"删了可能悄悄改变调试输出"的方法，保守保留 |
| `resolveTextConverterID` | `relaykit/relayconvert/text_converter_registry.go:356` | ② Go 引用 | 被 `LookupTextConverter`（**本轮保留**，属 relaykit 独立模块 API）实际调用；删除会直接编译失败 |
| `ClaudeErrorWrapper` | `service/error.go:62` | ③ 非 Go + ② | `.trellis/spec/backend/error-handling.md:56` 明确记录「Protocol-specific wrappers live next to it (`ClaudeErrorWrapper`…)」；且包内仍有引用（死代码簇）。删它会**污染 spec**，保留 |

## 2. 分批删除记录（每批后 build + 全量测试）

| 批次 | 范围 | 符号数 | `go build ./...` | `go test ./... -count=1` |
| --- | --- | --- | --- | --- |
| B1 controller+middleware | controller + middleware | 8 | ✅（合并验收） | ✅ exit=0 FAIL=0（B1-B3 合并跑） |
| B2 model-1 | model（第 1/3 组） | 10 | ✅（合并验收） | ✅ exit=0 FAIL=0（B1-B3 合并跑） |
| B3 model-2 | model（第 2/3 组） | 10 | ✅（合并验收） | ✅ exit=0 FAIL=0（B1-B3 合并跑） |
| B4 model-3+relay | model 剩余 + relay | 9 | ✅ | ✅ exit=0 FAIL=0 |
| B5 service-1 | service（第 1/2 组） | 10 | ✅ | ✅ exit=0 FAIL=0 |
| B6 service-2+authz | service 剩余 + service/authz | 10 | ✅ | ✅ exit=0 FAIL=0 |
| B7 relaykit | relaykit 内部工具函数 | 2 | ✅ | ✅ exit=0 FAIL=0 |

**B1–B3 说明**：B1 删 `middleware.RelayPanicRecover` 后 `middleware/recover.go` 只剩 `package`+`import`；
B2/B3 删 `model/perf_metric.go` 的两个函数后 `time` 变成未用 import；B1 的 `GetImage` 删完后
`controller/image.go` 同样只剩空壳。处理方式：**整文件删除 2 个空壳**（`middleware/recover.go`、`controller/image.go`）
＋清理未用 import，随后 B1–B3 合并跑 `go build`/`go test` **全绿**（exit=0，36 ok）。

**后续批次的 import 连带清理**（删除函数后其独占 import 失效，属预期，非语义改动）：
- B4：`relay/channel/jimeng/sign.go` 清理 `common`、`logger` 两个 import；
- B5：`service/file_decoder.go` 清理 9 个未用 import（`fmt/image/io/net/http/logger/types/gin/common` 等，
  文件仍保留活代码 `GetMimeTypeByExtension`）；
- B6：`service/request_converter.go`（`fmt`、`relaykit/dto`）、`service/sensitive.go`（`errors`、`relaykit/dto`）。

## 3. 级联分析（删除后重跑 deadcode）

```console
$ ~/go/bin/deadcode -test ./...
删除前: 144 行 / 141 个唯一符号
删除后:  85 行 /  85 个唯一符号
```

- **级联新增不可达：0 个**（`new - old = ∅`）。原因：本轮删的是"叶子"函数，它们不构成其它符号的唯一调用链。
- 剩余 85 = `需确认` 的 79 个 + 保留的 3 个 + 3 个**同名但不同符号**（`relaykit/relayconvert` 里的
  `ChatCompletionsRequestToResponsesRequest` / `ResponsesRequestToChatCompletionsRequest` /
  `ResponsesFinishReasonFromStatus`——被删的是 `service/` 下同名的薄包装，relaykit 的仍在名单里属"需确认"）。
- 与 `unused` 交叉：仍为 **恰 1 issue**（`logKeyCol`），无级联。

## 4. 残留引用清理（注释类）

删除后有 4 处 **Go 注释/注释掉的代码**仍点着已删符号（不构成编译引用，但会误导）：

| 位置 | 内容 | 处理 |
| --- | --- | --- |
| `service/token_counter.go:261-263` | 注释解释"为何不用 GetMimeType"（点名 `GetMimeType`/`GetFileTypeFromUrl`） | 改写为不点名已删函数的等价说明 |
| `model/token.go:505` | 「配合 InvalidateUserCache 使用」 | 去掉该半句 |
| `controller/log.go:112,139` | 两行注释掉的 `//tokenNum := model.SumUsedToken(...)` | 直接删除（死注释） |
| `router/api-router.go:37` | 注释掉的 `//apiRouter.GET("/midjourney", controller.GetMidjourney)` | **未改**：`router/` 不在本任务写作用域；仅注释、不影响编译，留给 Lead 决定 |

清理后：`grep` 已删的 59 个符号 → **Go 代码零引用**（只剩上述 router 注释与 `relaykit` 同名不同符号）。

## 5. 门禁输出（全部实际执行）

```console
$ go build ./...                              # OK
$ go vet ./...                                # OK
$ cd relaykit && GOWORK=off go build ./...    # OK
$ gofmt -l .
（空，0 行）
$ go test ./... -count=1
exit=0  FAIL=0  ok=36
$ ~/go/bin/deadcode -test ./... | wc -l
85
$ golangci-lint run --no-config --default=none --enable=unused ./...
model/main.go:27:5: var logKeyCol is unused (unused)
1 issues
$ git status --porcelain go.mod go.sum     # 工具未进依赖
（无输出）
```

## 6. 改动规模与回滚

- 41 文件变更：**39 修改 + 2 整文件删除**（`controller/image.go`、`middleware/recover.go` 均只剩空壳）；**805 行删除 / 3 行新增**。
- 回滚：`git checkout -- common controller middleware model relay relaykit service setting`
  ＋ `git checkout -- middleware/recover.go controller/image.go`（恢复被整文件删除的两个空壳）。
  未改数据库/配置/运行逻辑，回滚无副作用。

## 7. 风险与未完成

- **风险很低**：所有删除项经四类核对；每批 build+全量测试；级联 0；`unused` 仍恰 1（`logKeyCol`，属文档约定辅助，见上轮证据 §10）。
- 保守保留的 3 个：`Message.String`（fmt.Stringer 反射面）、`resolveTextConverterID`（被保留符号调用）、
  `ClaudeErrorWrapper`（spec 文档点名）。
- 未删的 `需确认` 79 个（`common/`、`relaykit/`、`types/`、`pkg/`、`oauth/`、`setting/` 的导出 API）需 owner 决策，
  清单见上轮证据 §5.2。
- `router/api-router.go:37` 的一行注释掉的旧路由（点名已删 `controller.GetMidjourney`）因写作用域不含 `router/` 未处理。
- 未提交、未推送、未发版；未动 `web/`、`data/`、实例。
