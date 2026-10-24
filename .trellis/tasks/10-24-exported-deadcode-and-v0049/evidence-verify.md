# 59 个导出死符号删除 + 死注释清理：独立验证证据（verifier）

- 任务：共享任务板 `task-48`（Trellis 任务 `10-24-exported-deadcode-and-v0049` 的独立验证）
- 验证者：verifier（只读验证；未改任何仓库文件，未使用 `-mod=mod`）
- 验证时间：2026-10-24（UTC 05:58 起）
- 基线：`HEAD = 5d27364`；最终 `git status --porcelain` 行数 = **43**（40 个已跟踪修改 + 2 个已跟踪删除 + 1 个任务目录）；
  `git diff` = **42 文件、3 增、806 删**；最终 diff 指纹 `bd21ed889fbc5fa8b61965b8447ba819be699e44b901e7134cca76d4f47a1688`
- `go.mod` / `go.sum` 无改动：blob 均等于 `HEAD` 版本（`go.mod = b0642f16…`、`go.sum = 357374c3…`）
- 工具环境：Go 1.26.6；`~/go/bin/deadcode`（RTA 静态分析）

## 结论

**接受（Accept）**。59 个被删符号经**全量**（非抽样）核对：删除前在根模块（非 relaykit）的全树引用**全部位于被删除行内**，
删除后根模块零残留；非 Go 文档/脚本/前端位置对已删符号零引用；两个整文件删除的符号无任何路由/中间件/测试引用；
保留 3 个的理由成立（其一为防御性保留，已注明）；门禁全绿、`unused` 恰 1、`deadcode` 85 条与声明一致；
级联分析显示**新增不可达 = 0**；边界与依赖文件干净。
另有 2 条精确性说明（不改判）：实现者「59 个导出符号」的口径不精确（实际为 57 个导出函数 + 2 个局部结构体导出字段，
另有 2 个 relaykit 非导出重复副本一并删除）；`Message.String` 的保留理由属「接口可反射可达但当前无调用点」的防御性保留。

## 一、删除安全（最重要）

### 1.1 符号提取（独立，不照抄）

从 `git diff -U0 -- '*.go'` 独立解析删除行，提取出 **59 个导出标识符**：

| 类别 | 数量 | 说明 |
|---|---|---|
| 导出函数 | 57 | 覆盖 controller/middleware/model/relay/service/relaykit 等 |
| 导出字段 | 2 | 匿名局部结构体 `result` 的 `Type` / `Count`（随 `CountChannelsGroupByType` 一并删除，不构成独立 API） |

另有 **2 个非导出函数**在同一批被删：`relaykit/.../internal/oai_responses/` 下的
`convertOpenAIResponsesRequestToClaudeMessages` / `convertOpenAIResponsesRequestToGeminiChat`
（内部重复副本；同名且被使用的副本仍在 `relaykit/relayconvert/request_registry.go:467/475`，由
`text_converter_registry.go:224/240` 作为 Convert 回调使用，未受影响）。

### 1.2 全量引用核对（59/59）

方法：`git archive HEAD` 解出 HEAD 全树，对每个符号做标识符边界全文件扫描；命中若落在该文件的「删除行区间」内即随本次删除消失，
剩余才算残留。结果：

| 核对维度 | 结果 |
|---|---|
| 根模块（非 relaykit）Go 引用，删除后残留 | **0**（59/59 符号） |
| 根模块 Go 引用，删除前落在删除行之外 | **0**（即所有旧引用点都已被删除覆盖） |
| 非 Go 位置（README / MAINTENANCE / AGENTS.md / CLAUDE.md / `.trellis/spec/**` / scripts / .github / web，排除构建产物） | **0 命中**（全仓扫描，排除 .git / node_modules / web/dist / .trellis/tasks） |
| relaykit 内同名命中 | 仅 5 个同名包装函数（ChatCompletions 与 Responses 转换包装）存在于 relaykit 自身；relaykit 是独立模块，**不能**引用根模块符号，属同名不同物 |
| 构建标签文件 | 仓库仅 2 个 `//go:build` 文件（windows / !windows），已纳入 HEAD 全树扫描，无被删符号引用 |

**整文件删除专项**（`controller/image.go` 9 行、`middleware/recover.go` 29 行）：

- `GetImage`（空 handler）与 `RelayPanicRecover`（gin 中间件）：HEAD 全树（含 router、middleware、全部 _test.go）
  在定义文件之外**零引用**；工作区同样零引用；无路由挂载、无中间件链路、无测试引用。
- 附注（非本次问题）：全仓不存在任何 panic recovery 中间件挂载（`main.go` 用 `gin.New()`，无 `gin.Recovery()`），
  说明 `RelayPanicRecover` 自诞生起就从未接线；删除它**不改变运行时行为**，但也意味着「无 panic 兜底」属既有问题，不在本任务范围。

### 1.3 抽样 38 个符号独立取证（要求 ≥15）

对 38 个符号逐个数「HEAD 全树 .go 命中数 / 其中在删除行内 / 残留」：

| 指标 | 结果 |
|---|---|
| 抽样符号数 | 38 |
| HEAD Go 命中总数 | 全部命中均落在删除行内 |
| 残留（删除行之外） | **0 / 38** |

代表项（完整计算见脚本）：`GetImage`、`GetMidjourney`、`GetTokenStatus`、`WssAuth`、`GetLanguage`、
`DownloadRateLimit`、`UploadRateLimit`、`RelayPanicRecover`、`GetAllEnableAbilities`、`CleanupChannelPollingLocks`、
`DeleteChannelByStatus`、`GetPaginatedTags`、`CountAllChannels`、`SumUsedToken`、`GetMjByuId`、`UpdateProgress`、
`GetAllModels`、`GetPerfMetricsSummaryAll`、`FindPendingSystemTasks`、`DisableModelLimits`、`GetTopUpById`、
`IsTelegramIdAlreadyTaken`、`InvalidateUserCache`、`GetUserOAuthBinding`、`IsHeaderPassthroughRuleKey`、`SetPayloadHash`、
`RemoveGeminiDisabledFields`、`DecodeBase64AudioData`、`FormatAuthError`、`AllPermissions`、`RefreshCodexOAuthToken`、
`ClaudeErrorWrapperLocal`、`GetFileTypeFromUrl`、`GetMimeType`、`ValidateProxyURL`、`RemoveDuplicate`、
`SensitiveWordReplace`、`CheckSensitiveMessages`。

### 1.4 注释/导入清理核对

- 3 处点名已删符号的注释已同步改写（且改写后不引用旧名）：
  `router/api-router.go`（删掉 `//apiRouter.GET("/midjourney", controller.GetMidjourney)` 一行，纯注释）、
  `model/token.go`（`InvalidateUserTokensCache` 文档注释去掉对 `InvalidateUserCache` 的引用）、
  `service/token_counter.go`（注释去掉对 `GetMimeType` / `GetFileTypeFromUrl` 的引用）。
- `git diff` 的 3 行新增**全部是注释**（无任何逻辑新增）：`model/token.go` 1 行、`service/token_counter.go` 2 行，
  为上述注释改写；无行为变化。
- 失效 import 随删除清理（`go build` / `go vet` / `gofmt` 全绿为证）。

## 二、保留 3 个候选的理由核对

| 保留项 | 独立核对 | 判定 |
|---|---|---|
| `resolveTextConverterID`（relaykit/relayconvert/text_converter_registry.go:356） | 被**保留的导出符号** `LookupTextConverter`（:258）在 :262 直接调用；删它会导致该保留符号编译失败 → 结构性必需 | 成立 |
| `ClaudeErrorWrapper`（service/error.go:62） | `.trellis/spec/backend/error-handling.md:56` 明确点名「Protocol-specific wrappers live next to it (`ClaudeErrorWrapper`, `MidjourneyErrorWrapper`, ...)」；其 HEAD 唯一调用者是已删除的 `ClaudeErrorWrapperLocal`，故静态不可达但按 spec 保留 | 成立（spec 点名） |
| `Message.String`（relay/channel/volcengine/protocols.go:256） | 方法签名满足 `fmt.Stringer`；`*Message` 由仍在使用的 `ReceiveMessage` → `tts.go:251` 产生，任何 fmt 动词（%v/%s）作用于它都会**反射调用** String；但当前代码只读字段、**没有任何格式化 Message 的调用点** → 可达性属「潜在/防御性」，删了不会编译失败，保留与否是取舍 | 成立但属防御性（如实注明） |

三者均仍存在于工作区（未删除）；且按预期继续出现在 deadcode 85 条中（保留项计入该数字）。

## 三、门禁复跑（真实工作区，Go 全部 `GOPROXY=off`，无 `-mod=mod`）

| 门禁 | 命令 | 结果 |
|---|---|---|
| Go 构建 | `go build ./...` | rc=0 |
| Go 静态检查 | `go vet ./...` | rc=0 |
| relaykit 独立构建 | `cd relaykit && GOWORK=off go build ./...` | rc=0 |
| 全仓测试 | `go test ./... -count=1` | rc=0，无 FAIL 行 |
| 格式 | `gofmt -l .` | 输出为空 |
| unused | `GOPROXY=off GOFLAGS=-mod=readonly golangci-lint run --default=none --enable=unused ./...` | **恰 1 issue**：`model/main.go:27:5 var logKeyCol is unused`（与上一轮一致） |
| deadcode 行数 | `~/go/bin/deadcode -test ./...`，对输出行数计数 | **85**（与实现者声明一致） |

`-mod=readonly` 运行前后 `go.mod` / `go.sum` blob 均未变化（`b0642f16…` / `357374c3…` = HEAD）。

## 四、级联分析（新增不可达）

| 指标 | 删除前（HEAD 全树，含 web/dist 以便 embed） | 删除后（工作区） |
|---|---|---|
| deadcode 条目数 | 144 | 85 |
| 集合关系 | — | **after 是 before 的子集**（无新增） |
| 消失条目 | — | **59** |
| **新增不可达** | — | **0** |

- 144 - 59 = 85，与逐条集合差一致；
- 消失的 59 条 = 57 个被删导出函数 + 2 个被删非导出 relaykit 重复副本（`CountChannelsGroupByType` 内部匿名结构体的
  `Type`/`Count` 字段不参与 RTA 函数级统计，故不在 59 条内）；
- 未出现「只被已删符号引用的下游辅助函数」变成新孤儿（这是级联风险的主要形式）。

## 五、边界

| 核对项 | 结果 |
|---|---|
| `git status --porcelain` | 43 行 = 40 M + 2 D（`controller/image.go`、`middleware/recover.go`）+ 1 个任务目录；与任务描述一致 |
| 依赖文件 | `go.mod` / `go.sum` 均无改动（blob = HEAD） |
| 未跟踪文件 | `-uall` 下仅任务目录内文件（prd / task.json / 实现者证据）；无临时/演示残留 |
| 改动规模 | 42 文件、3 增 806 删（任务描述称「约 -805/+3」，实测多 1 行删除，属计数差异，不影响结论） |

## 六、不确定项与观察（均非阻断）

1. 实现者「59 个导出符号」的**口径不精确**：独立提取为 59 个导出标识符（57 函数 + 2 个局部结构体导出字段），
   而 deadcode 前后差为 59 个**函数**（57 导出 + 2 个 relaykit 非导出重复副本）。两个「59」来源不同，建议报告里写清，
   避免后续误以为删了 59 个导出函数。
2. 全仓不存在 panic recovery 中间件（`RelayPanicRecover` 从未挂载）；删除行为中立，但「无 panic 兜底」是既有问题，
   若需要可单独立项（不属死代码清扫范围）。
3. `Message.String` 的保留是防御性选择（当前无格式化调用点），如后续要压 deadcode 数字可再评估；
   本轮保留不影响正确性。
4. 跨 OS 构建未执行（离线模块缓存缺平台依赖，与上一轮同因）；删除符号的引用风险已由 HEAD 全树（含 build tag 文件）静态扫描覆盖。

## 七、复跑命令清单

```bash
cd /root/new-api-own
# 删除安全（提取 + 引用核对脚本见本证据正文）
git diff -U0 -- '*.go'                                    # 59 个导出标识符 + 2 个 relaykit 非导出重复副本
git status --porcelain | wc -l                            # 43
# 门禁
GOPROXY=off go build ./... && GOPROXY=off go vet ./...
(cd relaykit && GOWORK=off GOPROXY=off go build ./...)
GOPROXY=off go test ./... -count=1
gofmt -l .
GOPROXY=off GOFLAGS=-mod=readonly golangci-lint run --default=none --enable=unused ./...
# deadcode 级联
GOPROXY=off GOFLAGS=-mod=readonly ~/go/bin/deadcode -test ./... | wc -l   # 85
# 依赖文件
git status --porcelain go.mod go.sum                      # 应为空
```
