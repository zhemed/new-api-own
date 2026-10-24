# 79 个「需确认」导出符号：三堆分类清单（只读）

对象：`deadcode -test ./...` 当前输出中的 85 个不可达导出符号（来源：`.trellis/tasks/archive/2026-10/10-24-deadcode-docs-ci/evidence-deadcode.md` §5.2 的「需确认」清单）
性质：只读分析——**未删除任何代码、未修改任何文件**（唯一写入为本文件）

---

## 一、结论与计数对账（先解决 79 / 82 / 85 三个数字）

【已证实】重跑 `deadcode -test ./...` 得到 **85 行 / 85 个唯一 (文件, 符号)**，按顶层目录：common 31、relaykit 23、pkg 13、setting 12、oauth 2、types 2、relay 1、service 1。与上一轮证据记录的「删除后 85 行 / 85 个唯一符号」完全一致。

三个数字的来源与关系（已逐条核对仓库内证据）：

| 数字 | 来源 | 与 85 的关系 |
|---|---|---|
| 85 | 当前 deadcode 输出行数 | 总数 |
| 82 | 上一轮「需确认」实际条目（79 + 3 个 relaykit 同名条目） | 85 − 3 个已裁定保留 |
| 79 | PRD / journal 口径（.trellis/workspace/new-api-own/journal-2.md:1967） | 82 − 3 个「同名但不同符号」的 relaykit 条目（上一轮按另一类单列） |

- 上一轮证据把 85 拆成「79 需确认 + 3 保留 + 3 同名」（evidence-exported.md:132-134）：3 个保留 = `Message.String`、`resolveTextConverterID`、`ClaudeErrorWrapper`；3 个同名 = relaykit 的 `ChatCompletionsRequestToResponsesRequest`、`ResponsesRequestToChatCompletionsRequest`、`ResponsesFinishReasonFromStatus`（被删的是 `service/` 下同名薄包装，relaykit 的仍在名单里）。
- 为不丢符号，本清单覆盖**全部 85 个**：其中 82 个属待决策面（含 3 个同名条目，表中标「⚠同名」），另 3 个沿用上一轮「保留」裁定（C 堆）。
- 【异常/风险】源文件 §5.2 的正文是**空的**（`evidence-deadcode.md:255` 只有标题，无清单），无法直接抄录；本清单因此按同一口径**重跑 deadcode 重建**，并已用「85 行 + 包分布」与上一轮记录交叉验证一致。建议 Lead 后续把本表回填为 §5.2 的正文，避免清单再次丢失。

归堆口径（本报告采用，避免"公共包里的死符号只能算保留"的悖论）：

- **A 可删**：全仓 0 引用、无文档约定、无接口/反射绑定；属应用内部工具面或冗余副本。
- **B 需补调用**：功能上「本该被调用」却零调用，或与既有实现重复造成两套逻辑；给出具体应调用处。
- **C 保留**：被文档/规范点名、实现接口或反射面、独立模块已发布 API 契约、或上一轮已明确裁定保留。
- 「对外 API 面」一栏只标**属性与风险**（relaykit 独立模块 / common 公共工具 / types+pkg 基础设施 / oauth+setting 配置面），**不自动决定归堆**；relaykit 的 A 项全部另列 owner 决策（见第五节）。

归堆结果：**A 可删 76 个 / B 需补调用 4 个 / C 保留 5 个 = 85**。

---

## 二、重跑证据（命令与输出）

```console
cd /root/new-api-own && /root/go/bin/deadcode -test ./...
85 行，全部形如 `<file>:<line>:<col>: unreachable func: <Name>`
包分布：common 31 / relaykit 23 / pkg 13 / setting 12 / oauth 2 / types 2 / relay 1 / service 1
```

- deadcode 判定列在表中统一写「不可达」（全部来自本次实际输出，无一条是推断）。
- 交叉核对：本清单与上一轮 §7 记录的 3 个保留符号、§3 记录的 85 行数字、journal-2.md:1967 的 79 个口径均可对上（见第一节）。
- 说明：`-test` 模式已把「仅测试引用」的函数纳入可达，因此表中无「只在测试里活」的误报；反射/接口类风险在表内单独标注（`Message.String`、`StringValue`、`Set` 等）。

---

## 三、逐符号清单（85 行，按包分组）

### 3.1 common（31 个：A 29 / B 2）

| # | 符号 | 签名摘要 | deadcode | 文档点名 | 对外面 | 归堆·风险·理由 |
|---|---|---|---|---|---|---|
| 1 | common.ReadDiskCacheFile | `ReadDiskCacheFile(filePath string) ([]byte, error)` | 不可达 | 无 | common 公共工具 | A·低·磁盘缓存写入侧在用（WriteDiskCacheFileString），读取侧由 relaykit 自行处理，这三个读取/删除辅助全仓 0 引用 |
| 2 | common.ReadDiskCacheFileString | `ReadDiskCacheFileString(filePath string) (string, error)` | 不可达 | 无 | common 公共工具 | A·低·同上 |
| 3 | common.RemoveDiskCacheFile | `RemoveDiskCacheFile(filePath string) error` | 不可达 | 无 | common 公共工具 | A·低·清理走 CleanupOldDiskCacheFiles，单文件删除无调用 |
| 4 | common.GetDiskCacheInfo | `GetDiskCacheInfo() (fileCount int, totalSize int64, err error)` | 不可达 | 无 | common 公共工具 | A·低·统计已由 GetDiskCacheStats 提供（controller/performance.go:87） |
| 5 | common.ResetDiskCacheUsage | `ResetDiskCacheUsage()` | 不可达 | 无 | common 公共工具 | A·低·管理端用 ResetDiskCacheStats（controller/performance.go:161） |
| 6 | common.SyncDiskCacheStats | `SyncDiskCacheStats()` | 不可达 | 无 | common 公共工具 | A·低·无调用，统计由增量函数维护 |
| 7 | common.LoginAuth | `LoginAuth(username, password string) smtp.Auth` | 不可达 | 无 | common 公共工具 | A·低·旧 Outlook AUTH 实现，已被 AutoSMTPAuth 取代（common/email_ntlm_auth.go:17） |
| 8 | common.outlookAuth.Start | `(a *outlookAuth) Start(_ *smtp.ServerInfo) (string, []byte, error)` | 不可达 | 无 | common 公共工具 | A·低·随 LoginAuth 一起死；注意保留 isOutlookServer（仍被 common/email.go:26 使用） |
| 9 | common.outlookAuth.Next | `(a *outlookAuth) Next(fromServer []byte, more bool) ([]byte, error)` | 不可达 | 无 | common 公共工具 | A·低·同上 |
| 10 | common.GetContextKeyStringSlice | `GetContextKeyStringSlice(c *gin.Context, key constant.ContextKey) []string` | 不可达 | 无 | common 公共工具 | B·低·service/group.go:98-103 手工 GetContextKey 加类型断言，应改用该 helper |
| 11 | common.SafeSendString | `SafeSendString(ch chan string, value string) (closed bool)` | 不可达 | 无 | common 公共工具 | A·低·同文件仅 SafeSendBool 在用（common/go-channel.go:7） |
| 12 | common.SafeSendStringTimeout | `SafeSendStringTimeout(ch chan string, value string, timeout int) (closed bool)` | 不可达 | 无 | common 公共工具 | A·低·同上 |
| 13 | common.Sha256Raw | `Sha256Raw(data []byte) []byte` | 不可达 | 无 | common 公共工具 | A·低·同文件 Sha1/Sha1Raw 在用，Sha256Raw 0 引用 |
| 14 | common.HmacSha256Raw | `HmacSha256Raw(message, key []byte) []byte` | 不可达 | 无 | common 公共工具 | A·低·webhook/auth_token 直接 hmac.New，无调用 |
| 15 | common.HmacSha256 | `HmacSha256(message, key string) string` | 不可达 | 无 | common 公共工具 | A·低·同上 |
| 16 | common.IsIP | `IsIP(s string) bool` | 不可达 | 无 | common 公共工具 | A·低·SSRF 校验直接 net.ParseIP，无调用 |
| 17 | common.ParseIP | `ParseIP(s string) net.IP` | 不可达 | 无 | common 公共工具 | A·低·同上 |
| 18 | common.IsPrivateIP | `IsPrivateIP(ip net.IP) bool` | 不可达 | 无 | common 公共工具 | B·中·common/ssrf_protection.go:108/132 内联了等价判定，建议复用消除两套私有网段逻辑（安全语义相关） |
| 19 | common.ParseRedisOption | `ParseRedisOption() *redis.Options` | 不可达 | 无 | common 公共工具 | A·低·redis.go:35 初始化直接 redis.ParseURL |
| 20 | common.RedisDel | `RedisDel(key string) error` | 不可达 | 无 | common 公共工具 | A·低·实际在用 RedisDelKey（common/redis.go:99） |
| 21 | common.StrToJsonArray | `StrToJsonArray(str string) ([]interface{}, error)` | 不可达 | 无 | common 公共工具 | A·低·同文件 StrToMap/IsJsonObject 在用 |
| 22 | common.IsJsonArray | `IsJsonArray(str string) bool` | 不可达 | 无 | common 公共工具 | A·低·同上 |
| 23 | common.EncodeBase64 | `EncodeBase64(str string) string` | 不可达 | 无 | common 公共工具 | A·低·base64 编码各处直调，无调用 |
| 24 | common.OpenBrowser | `OpenBrowser(url string)` | 不可达 | 无 | common 公共工具 | A·低·上游遗留的本地开浏览器工具，本仓库无调用 |
| 25 | common.Bytes2Size | `Bytes2Size(num int64) string` | 不可达 | 无 | common 公共工具 | A·低·格式化辅助，0 引用（前端自己有格式化） |
| 26 | common.Seconds2Time | `Seconds2Time(num int) (time string)` | 不可达 | 无 | common 公共工具 | A·低·同上 |
| 27 | common.UnescapeHTML | `UnescapeHTML(x string) interface{}` | 不可达 | 无 | common 公共工具 | A·低·0 引用 |
| 28 | common.IntMax | `IntMax(a int, b int) int` | 不可达 | 无 | common 公共工具 | A·低·0 引用（IntMin 也不存在） |
| 29 | common.Any2Type | `Any2Type[T any](data any) (T, error)` | 不可达 | 无 | common 公共工具 | A·低·活跃副本是 relaykit/relayconvert/kitutil.Any2Type（被 dto 多处使用），根模块副本无人用 |
| 30 | common.SaveTmpFile | `SaveTmpFile(filename string, data io.Reader) (string, error)` | 不可达 | 无 | common 公共工具 | A·低·落盘走 disk_cache 或 service 自有实现 |
| 31 | common.BuildURL | `BuildURL(base string, endpoint string) string` | 不可达 | 无 | common 公共工具 | A·低·URL 拼接各处自写（更新侧另有 sanitizePublicURL） |

### 3.2 relaykit（23 个：A 18 / B 2 / C 3；全部标 owner 决策）

| # | 符号 | 签名摘要 | deadcode | 文档点名 | 对外面 | 归堆·风险·理由 |
|---|---|---|---|---|---|---|
| 32 | relaykit/dto.StringValue.UnmarshalJSON | `(s *StringValue) UnmarshalJSON(data []byte) error` | 不可达 | 无 | relaykit 独立模块 | A⚠·低·StringValue 类型全仓未使用；json 接口方法只因类型被实例化才可达 |
| 33 | relaykit/dto.StringValue.MarshalJSON | `(s StringValue) MarshalJSON() ([]byte, error)` | 不可达 | 无 | relaykit 独立模块 | A⚠·低·同上 |
| 34 | relaykit/relayconvert.ClaudeMessagesRequestToOpenAIChat | `ClaudeMessagesRequestToOpenAIChat(claudeRequest dto.ClaudeRequest, info convmeta.Meta) (*dto.GeneralOpenAIRequest, error)` | 不可达 | 无 | relaykit 独立模块 | A⚠·低·兼容门面层；活跃路径是 request_registry.go:425 直调 internal 包 |
| 35 | relaykit/relayconvert.GeminiGenerateContentRequestToOpenAIChat | `GeminiGenerateContentRequestToOpenAIChat(geminiRequest *dto.GeminiChatRequest, info convmeta.Meta) (*dto.GeneralOpenAIRequest, error)` | 不可达 | 无 | relaykit 独立模块 | A⚠·低·同上（request_registry.go:451） |
| 36 | relaykit/relayconvert.OpenAIChatRequestToGeminiGenerateContent | `OpenAIChatRequestToGeminiGenerateContent(c context.Context, textRequest dto.GeneralOpenAIRequest, info convmeta.Meta) (*dto.GeminiChatRequest, error)` | 不可达 | 无 | relaykit 独立模块 | A⚠·低·同上（request_registry.go:464） |
| 37 | relaykit/relayconvert.ChatCompletionsRequestToResponsesRequest | `ChatCompletionsRequestToResponsesRequest(req *dto.GeneralOpenAIRequest) (*dto.OpenAIResponsesRequest, error)` | 不可达 | 无 | relaykit 独立模块 | A⚠·低·同上（request_registry.go:412）；⚠同名条目（上一轮按同名另计） |
| 38 | relaykit/relayconvert.ResponsesRequestToChatCompletionsRequest | `ResponsesRequestToChatCompletionsRequest(req *dto.OpenAIResponsesRequest) (*dto.GeneralOpenAIRequest, error)` | 不可达 | 无 | relaykit 独立模块 | A⚠·低·同上（request_registry.go:498）；⚠同名条目 |
| 39 | relaykit/relayconvert.OpenAIResponsesRequestToClaudeMessages | `OpenAIResponsesRequestToClaudeMessages(c context.Context, info convmeta.Meta, req *dto.OpenAIResponsesRequest) (*dto.ClaudeRequest, error)` | 不可达 | 无 | relaykit 独立模块 | A⚠·低·同上（request_registry.go:472） |
| 40 | relaykit/relayconvert.OpenAIResponsesRequestToGeminiChat | `OpenAIResponsesRequestToGeminiChat(c context.Context, req *dto.OpenAIResponsesRequest, info convmeta.Meta) (*dto.GeminiChatRequest, error)` | 不可达 | 无 | relaykit 独立模块 | A⚠·低·同上（request_registry.go:485） |
| 41 | relaykit/relayconvert.StopReasonClaudeToOpenAI | `StopReasonClaudeToOpenAI(reason string) string` | 不可达 | 无 | relaykit 独立模块 | A⚠·低·兼容门面；活跃实现在 internal/claude_messages |
| 42 | relaykit/relayconvert.ResponsesStatusFromChatFinishReason | `ResponsesStatusFromChatFinishReason(finishReason string) (string, *dto.IncompleteDetails)` | 不可达 | 无 | relaykit 独立模块 | A⚠·低·兼容门面；活跃实现在 internal/oai_chat |
| 43 | relaykit/relayconvert.ResponsesFinishReasonFromStatus | `ResponsesFinishReasonFromStatus(resp *dto.OpenAIResponsesResponse) (string, bool)` | 不可达 | 无 | relaykit 独立模块 | A⚠·低·兼容门面；活跃实现在 internal/oai_responses；⚠同名条目 |
| 44 | relaykit/relayconvert.ExtractReasoningTextFromResponses | `ExtractReasoningTextFromResponses(resp *dto.OpenAIResponsesResponse) string` | 不可达 | 无 | relaykit 独立模块 | A⚠·低·兼容门面；活跃实现在 internal/oai_responses |
| 45 | relaykit/relayconvert.ConvertResponseByID | `ConvertResponseByID(c context.Context, info convmeta.Meta, converter string, response any) (*ResponseResult, error)` | 不可达 | relaykit/README.md:227 | relaykit 独立模块 | C·低·模块 README 点名的对外 API，保留 |
| 46 | relaykit/relayconvert.NewResponseStreamStateByID | `NewResponseStreamStateByID(converter string, options ResponseStreamOptions) (*ResponseStreamState, error)` | 不可达 | relaykit/README.md:227 | relaykit 独立模块 | C·低·同上 |
| 47 | relaykit/relayconvert.LookupTextConverter | `LookupTextConverter(converter string) (TextConverterSpec, bool)` | 不可达 | 无 | relaykit 独立模块 | A⚠·低·仅 text_converter_registry_test.go:120/132 引用 |
| 48 | relaykit/relayconvert.resolveTextConverterID | `resolveTextConverterID(converter string) string` | 不可达 | 无 | relaykit 独立模块 | C·低·上一轮已裁定保留（LookupTextConverter 内部调用它） |
| 49 | relaykit/relayconvert/convmeta.ChannelTypeOf | `ChannelTypeOf(m Meta) int` | 不可达 | 无 | relaykit 独立模块 | A⚠·低·仅 convmeta/meta_test.go:37 使用；同文件另有 UpstreamModelName 等访问器 |
| 50 | relaykit/relayconvert/kitutil.UnmarshalJsonStr | `UnmarshalJsonStr(data string, v any) error` | 不可达 | 同名约定见 AGENTS.md:76；.trellis/spec/backend/quality-guidelines.md:41 | relaykit 独立模块 | B⚠·中·文档点名的是根模块 common.UnmarshalJsonStr；relaykit 内 14 处直调 std json，owner 需定「统一走 kitutil」或「删副本 + 记录模块例外」 |
| 51 | relaykit/relayconvert/kitutil.DecodeJson | `DecodeJson(reader io.Reader, v any) error` | 不可达 | 同名约定见 AGENTS.md:77；quality-guidelines.md:41 | relaykit 独立模块 | B⚠·中·同上 |
| 52 | relaykit/relayconvert/kitutil.StringToByteSlice | `StringToByteSlice(s string) []byte` | 不可达 | 无 | relaykit 独立模块 | A⚠·低·仅被上两项使用，而上两项自身也无人调 |
| 53 | relaykit/types.NewFileMeta | `NewFileMeta(fileType FileType, source FileSource) *FileMeta` | 不可达 | 无 | relaykit 独立模块 | A⚠·低·类型构造器 0 调用 |
| 54 | relaykit/types.NewImageFileMeta | `NewImageFileMeta(source FileSource, detail string) *FileMeta` | 不可达 | 无 | relaykit 独立模块 | A⚠·低·同上 |

### 3.3 pkg（13 个：A 13）

| # | 符号 | 签名摘要 | deadcode | 文档点名 | 对外面 | 归堆·风险·理由 |
|---|---|---|---|---|---|---|
| 55 | pkg/billingexpr.RunExprByHash | `RunExprByHash(exprStr, hash string, params TokenParams) (float64, TraceResult, error)` | 不可达 | 无 | pkg 基础设施 | A·低·活跃入口是 RunExprByHashWithRequest（pkg/billingexpr/settle.go:22） |
| 56 | pkg/cachex.StringCodec.Encode | `(c StringCodec) Encode(v string) (string, error)` | 不可达 | 无 | pkg 基础设施 | A·低·活跃 codec 是 IntCodec；StringCodec 从未被实例化 |
| 57 | pkg/cachex.StringCodec.Decode | `(c StringCodec) Decode(s string) (string, error)` | 不可达 | 无 | pkg 基础设施 | A·低·同上 |
| 58 | pkg/cachex.HybridCache.FullKey | `(c *HybridCache[V]) FullKey(key string) string` | 不可达 | 无 | pkg 基础设施 | A·低·HybridCache 在用（service/channel_affinity.go），该方法无调用 |
| 59 | pkg/ionet.Client.GetContainerJobs | `(c *Client) GetContainerJobs(deploymentID, containerID string) (*ContainerList, error)` | 不可达 | 无 | pkg 基础设施 | A·低·容器列表用 ListContainers（controller/deployment.go） |
| 60 | pkg/ionet.Client.GetContainerLogs | `(c *Client) GetContainerLogs(deploymentID, containerID string, opts *GetLogsOptions) (*ContainerLogs, error)` | 不可达 | 无 | pkg 基础设施 | A·低·日志用 GetContainerLogsRaw（controller/deployment.go:645 附近） |
| 61 | pkg/ionet.Client.StreamContainerLogs | `(c *Client) StreamContainerLogs(deploymentID, containerID string, opts *GetLogsOptions, callback func(*LogEntry) error) error` | 不可达 | 无 | pkg 基础设施 | A·低·无流式日志调用 |
| 62 | pkg/ionet.Client.RestartContainer | `(c *Client) RestartContainer(deploymentID, containerID string) error` | 不可达 | 无 | pkg 基础设施 | A·低·前后端均无容器重启动作（路由表无 restart） |
| 63 | pkg/ionet.Client.StopContainer | `(c *Client) StopContainer(deploymentID, containerID string) error` | 不可达 | 无 | pkg 基础设施 | A·低·无停止动作 |
| 64 | pkg/ionet.Client.ExecuteInContainer | `(c *Client) ExecuteInContainer(deploymentID, containerID string, command []string) (string, error)` | 不可达 | 无 | pkg 基础设施 | A·低·无容器 exec 调用 |
| 65 | pkg/ionet.Client.GetHardwareType | `(c *Client) GetHardwareType(hardwareID int) (*HardwareType, error)` | 不可达 | 无 | pkg 基础设施 | A·低·硬件列表用 ListHardwareTypes |
| 66 | pkg/ionet.Client.GetLocation | `(c *Client) GetLocation(locationID int) (*Location, error)` | 不可达 | 无 | pkg 基础设施 | A·低·位置列表用 ListLocations |
| 67 | pkg/ionet.Client.GetLocationAvailability | `(c *Client) GetLocationAvailability(locationID int) (*LocationAvailability, error)` | 不可达 | 无 | pkg 基础设施 | A·低·可用性用 GetAvailableReplicas |

### 3.4 setting（12 个：A 12）

| # | 符号 | 签名摘要 | deadcode | 文档点名 | 对外面 | 归堆·风险·理由 |
|---|---|---|---|---|---|---|
| 68 | setting.ContainsAutoGroup | `ContainsAutoGroup(group string) bool` | 不可达 | 无 | setting 配置面 | A·低·auto group 判定在 service/group.go 内联，0 调用 |
| 69 | setting.SetWaffoPayMethods | `SetWaffoPayMethods(methods []constant.WaffoPayMethod) error` | 不可达 | 无 | setting 配置面 | A·低·选项保存直写 OptionMap（model/option.go:112），仅 Get 被用 |
| 70 | setting/model_setting.GetQwenSettings | `GetQwenSettings() *QwenSettings` | 不可达 | 无 | setting 配置面 | A·低·qwenSettings 已由 config 注册，IsSyncImageModel 直接用变量，取结构体无调用 |
| 71 | setting/operation_setting.IsCheckinEnabled | `IsCheckinEnabled() bool` | 不可达 | 无 | setting 配置面 | A·低·功能代码用 GetCheckinSetting（model/checkin.go:56、controller/checkin.go:17/48） |
| 72 | setting/operation_setting.GetCheckinQuotaRange | `GetCheckinQuotaRange() (min, max int)` | 不可达 | 无 | setting 配置面 | A·低·同上 |
| 73 | setting/operation_setting.IsCNYDisplay | `IsCNYDisplay() bool` | 不可达 | 无 | setting 配置面 | A·低·同文件 IsCurrencyDisplay 在用，本函数 0 调用 |
| 74 | setting/performance_setting.GetPerformanceSetting | `GetPerformanceSetting() *PerformanceSetting` | 不可达 | 无 | setting 配置面 | A·低·UpdateAndSync 在用，取结构体无调用 |
| 75 | setting/performance_setting.GetCacheStats | `GetCacheStats() common.DiskCacheStats` | 不可达 | 无 | setting 配置面 | A·低·controller/performance.go:87 直接用 common.GetDiskCacheStats |
| 76 | setting/performance_setting.ResetStats | `ResetStats()` | 不可达 | 无 | setting 配置面 | A·低·controller/performance.go:161 直接用 common.ResetDiskCacheStats |
| 77 | setting/ratio_setting.GetCacheRatioMap | `GetCacheRatioMap() map[string]float64` | 不可达 | 无 | setting 配置面 | A·低·单模型 GetCacheRatio 在用（cache_ratio.go:69） |
| 78 | setting/ratio_setting.GetModelPriceMap | `GetModelPriceMap() map[string]float64` | 不可达 | 无 | setting 配置面 | A·低·有效价走 GetModelPrice / GetDefaultModelPriceMap（relay/helper/price.go:195） |
| 79 | setting/ratio_setting.GetModelRatioOrPrice | `GetModelRatioOrPrice(model string) (float64, bool, bool)` | 不可达 | 无 | setting 配置面 | A·低·组合读无调用；调用方分别用 GetModelPrice 与 GetModelRatio |

### 3.5 oauth / types / relay / service（7 个）

| # | 符号 | 签名摘要 | deadcode | 文档点名 | 对外面 | 归堆·风险·理由 |
|---|---|---|---|---|---|---|
| 80 | oauth.GetAllProviders | `GetAllProviders() map[string]Provider` | 不可达 | 无 | oauth 配置面 | A·低·对外列表用 GetEnabledCustomProviders（controller/misc.go:139） |
| 81 | oauth.ReloadCustomProviders | `ReloadCustomProviders() error` | 不可达 | 无 | oauth 配置面 | A·低·启动用 LoadCustomProviders（main.go:354）；管理端直接 RegisterOrUpdateCustomProvider |
| 82 | types.Set.Remove | `(s *Set[T]) Remove(item T)` | 不可达 | 无 | types 基础设施 | A·低·类型在 model/pricing.go:261 使用（Add/Items/Len），Remove 无调用 |
| 83 | types.Set.Contains | `(s *Set[T]) Contains(item T) bool` | 不可达 | 无 | types 基础设施 | A·低·同上 |
| 84 | relay/channel/volcengine.Message.String | `(m *Message) String() string` | 不可达 | 无 | 应用层 | C·低·fmt.Stringer 反射/格式化面，上一轮已裁定保留 |
| 85 | service.ClaudeErrorWrapper | `ClaudeErrorWrapper(err error, code string, statusCode int) *dto.ClaudeErrorWithStatusCode` | 不可达 | .trellis/spec/backend/error-handling.md:56 | 应用层 | C·低·规范点名的协议错误包装器，上一轮已裁定保留 |

---

## 四、按包分组统计

| 包 | A 可删 | B 需补调用 | C 保留 | 合计 |
|---|---|---|---|---|
| common | 29 | 2 | 0 | 31 |
| relaykit | 18 | 2 | 3 | 23 |
| pkg | 13 | 0 | 0 | 13 |
| setting | 12 | 0 | 0 | 12 |
| oauth | 2 | 0 | 0 | 2 |
| types | 2 | 0 | 0 | 2 |
| relay | 0 | 0 | 1 | 1 |
| service | 0 | 0 | 1 | 1 |
| **合计** | **76** | **4** | **5** | **85** |

风险分布：A 堆 76 个全部低风险；B 堆 4 个中 3 个中风险（`common.IsPrivateIP`、`relaykit/kitutil` 两项）+ 1 个低（`common.GetContextKeyStringSlice`）；C 堆 5 个全部低风险。

---

## 五、建议执行顺序（本任务不执行，仅建议）

原则：先动「零契约、纯内部」的，再动「公共包导出面」，最后动「独立模块 API」；每批都要能独立回滚。

| 批次 | 范围 | 规模 | 为什么这个顺序 | 每批验证方式 |
|---|---|---|---|---|
| 第 1 批 | pkg（13）+ oauth（2）+ types（2）+ setting（12） | 29 | 全是内部基础设施/配置 getter，仓库内 0 引用，不涉及独立模块 | `go build ./...`、`go vet ./...`、`go test ./... -count=1`、`gofmt -l .` 为空、重跑 deadcode 确认 85→56 |
| 第 2 批 | common A 堆（29），拆两小批：2a 存储/网络/编解码/协程/旧邮件认证（disk_cache 6、ip 2、redis 2、hash 3、str 3、go-channel 2、email 旧 AUTH 3 = 21）；2b utils 遗留（8） | 29 | common 是公共工具面，建议小步删 + 每小批全测试 | 同第 1 批；2a 后 deadcode 预期 85→56 或更少；`go build ./...` 必须全绿（common 被全仓引用最多） |
| 第 3 批 | relaykit A 堆（18，需 owner 批准） | 18 | 独立模块导出面，先拿 owner 结论再动；建议先删 11 个兼容门面 + 5 个明确零引用项，分两次提交 | `cd relaykit && GOWORK=off go build ./...` 与 GOWORK=off go test；根模块 build/test（replace 指向本地）；`gofmt -l` 空 |
| 第 4 批 | B 堆（4）——不是删除而是补调用/统一 | 4 | 前 3 批删完后，再决定这些是「接线」还是「删+改文档」 | 每项单独 PR/提交 + 对应单测：`common.IsPrivateIP` 改造后在 ssrf 测试里断言私网/回环/链路本地；`GetContextKeyStringSlice` 改造后跑 service 包测试；kitutil 两项需 relaykit 测试 |
| 保持 | C 堆（5） | 5 | 不动（文档/接口/裁定保留） | 仅需在 deadcode 复跑时确认仍为不可达但有意保留，记录原因 |

补充建议：

- 每批提交信息带 `[task:<slug>]`；每批完成后重跑 `deadcode -test ./...` 并记录数字（85 → 56 → 27 → 9），作为进度证据。
- 第 1、2 批可合并为一个「内部工具清理」任务但分两个提交；第 3 批必须独立成任务（独立模块 API 变更）。
- 由于 §5.2 原文缺失，建议本轮结束时把本清单回填进归档证据文件，避免下一轮再重建。

---

## 六、需要 owner（用户/Lead）决策的符号清单

### 6.1 relaykit 独立模块导出 API（18 个 A + 2 个 B = 20）

理由：`relaykit/` 是独立 Go 模块（relaykit/go.mod；根 go.mod:170 用 replace 指向本地），AGENTS.md 要求它保持可独立构建。删除导出符号等于改模块 API 面，需 owner 明确「本仓库自用、允许删」还是「保留对外兼容面」。

- A 堆（18）：`StringValue.MarshalJSON`、`StringValue.UnmarshalJSON`、11 个 request/response 兼容门面（request_compat.go 7 个 + response_compat.go 4 个，清单见 3.2 第 34-44 行）、`LookupTextConverter`、`ChannelTypeOf`、`NewFileMeta`、`NewImageFileMeta`。
- B 堆（2）：`kitutil.UnmarshalJsonStr`、`kitutil.DecodeJson`——名字与 AGENTS.md:76-77 的 `common.*` 约定撞名，但位于独立模块内（无法 import 根模块）。需要 owner 定：① relaykit 内部统一改走 kitutil（现有 14 处直调 std json）；或 ② 删除这两个副本并在 AGENTS.md / spec 注明「relaykit 模块例外」。

### 6.2 与文档/规范点名相关的符号

- `service.ClaudeErrorWrapper`：被 `.trellis/spec/backend/error-handling.md:56` 点名 → 本报告归 C（保留）；若 owner 要删，必须同步改 spec。
- `relaykit/.../kitutil.UnmarshalJsonStr` / `DecodeJson`：见 6.1。
- 说明：根模块 `common.UnmarshalJsonStr` / `common.DecodeJson` 本身**在 live 代码里可达**（不在本次 85 名单中），文档约定未失效；风险仅在 relaykit 侧同名副本。

### 6.3 语义/安全相关，建议 owner 过一眼（4 个）

- `common.IsPrivateIP`（B）：是否统一到 ssrf_protection，涉及安全语义（回环/链路本地/私网判定范围），建议由 owner 确认判定口径后改。
- `common.LoginAuth` / `outlookAuth.Start` / `outlookAuth.Next`（A）：旧 Outlook SMTP AUTH 实现已被 AutoSMTPAuth 取代；若还有企业 Outlook SMTP 场景依赖它，应改为接线（B），否则删。
- `types.Set.Contains` / `Remove`（A）：Set 类型仍在用，删两个方法不影响编译；若希望保留 `Set` 作为完整公共容器 API，可归 C。
- `pkg/ionet` 容器运维方法（A，6 个）：`RestartContainer` / `StopContainer` / `ExecuteInContainer` / `StreamContainerLogs` / `GetContainerJobs` / `GetContainerLogs`——当前前后端均无对应动作；若产品计划补「容器重启/停止/流式日志」，应保留并接线（B），否则可删。

### 6.4 配置面 getter（12 个 setting）

全部 A 低风险，但属"配置与扩展面"：若 owner 希望 setting 子包保留成组 getter 作为管理端扩展点，可整体改为 C；当前实现里功能代码统一用结构体 getter，删除不会改变行为。

---

## 七、不确定项与风险

1. **源清单缺失**：归档证据 §5.2 正文为空（evidence-deadcode.md:255），本清单由重跑 deadcode 重建（85 行与上一轮记录一致），但无法逐字比对上一轮原始 82 条 → 若上一轮曾人工剔除某条，本表会多出对应行。缓解：本表已覆盖全部 85 行，宁多不少。
2. **计数口径**：79 = 82 − 3 个同名条目；若 owner 坚持「同名条目不算独立符号」，则本表的 37、38、43 三行可并入对应 79 口径，结论不变（仍是 A⚠）。
3. **deadcode 固有盲区**：反射/插件/`//go:linkname` 不可见。本清单中的接口/反射嫌疑项（`Message.String`、`StringValue`、`StringCodec`、`Set`）已逐个用「类型是否被实例化」核对，未发现真反射调用；删除前仍建议对每批做一次 grep 复核。
4. **relaykit 模块版本策略未知**：本仓库内用 replace 指向本地，若该模块被其它仓库/下游引用，删导出符号会是破坏性变更——这是第 6.1 节需要 owner 拍板的根本原因。
5. **A 堆中的公共包导出面**（common 29 个）严格说会改变 `common` 包导出 API；本仓库是应用仓库、未发现外部导入方，故按低风险评估；若 owner 认为 common 也算对外契约，可把这 29 个整体降为 C。
6. 本轮**未做**任何删除、未改任何文件、未跑构建（只读约束）；第 1-3 批的验证方式均为建议，实际执行需另开任务并授权。

---

## 八、复核命令与越界自述

```bash
# 重跑清单（只读）
/root/go/bin/deadcode -test ./...

# 引用/文档核对（示例，逐符号执行过）
grep -rn --include=*.go 'common\.ReadDiskCacheFile' .          # 0
grep -rn --include=*.go 'common\.IsPrivateIP' .                # 0
grep -rn --include=*.go 'setting\.SetWaffoPayMethods' .        # 0
grep -rn --include=*.go 'relayconvert\.ConvertResponseByID' .  # 仅测试
grep -rn -w 'ClaudeErrorWrapper' .trellis/spec                 # error-handling.md:56
grep -rn -w 'ConvertResponseByID\|NewResponseStreamStateByID' relaykit/README.md   # :227
grep -rn -w 'UnmarshalJsonStr\|DecodeJson' AGENTS.md .trellis/spec/backend/quality-guidelines.md
grep -rn --include=*.go 'json.Unmarshal\|json.NewDecoder' relaykit/ | grep -v _test.go | grep -v kitutil/json.go | wc -l   # 14

# 关键上下文（逐步核过）
sed -n '15,60p' common/email.go                    # SMTP 认证新实现取代旧 Outlook AUTH
sed -n '90,110p' service/group.go                  # 手工 GetContextKey + 类型断言
sed -n '100,140p' common/ssrf_protection.go        # 内联私网/回环判定
sed -n '360,395p' router/api-router.go             # deployment 路由无 restart/stop
grep -n 'client\.' controller/deployment.go        # 实际调用的 ionet 方法
sed -n '550,570p' setting/ratio_setting/model_ratio.go   # GetModelRatioOrPrice 实现
```

外部证据：无（本轮不联网）。工具：`/root/go/bin/deadcode`（golang.org/x/tools，上一轮已安装，本轮仅执行，未安装未联网）。

自查：0 个成对美元符号；星号仅出现在反引号代码片段内（函数签名中的指针类型），不构成 Markdown 斜体；四类表格各自列数一致（7 列主表 / 5 列统计表 / 3 列对账表 / 5 列执行顺序表）。

越界自述：未删除/修改任何代码或配置，未提交、未推送、未构建、未联网、未装工具；唯一写入为本文件。唯一一次临时写入是 `/tmp/deadcode_out.txt`（管道重定向产物），已立即删除，未包含任何凭据或实例信息。
