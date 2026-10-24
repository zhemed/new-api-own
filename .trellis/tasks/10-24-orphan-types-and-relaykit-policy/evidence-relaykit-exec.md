# relaykit 导出面收敛执行证据（task-60 / PRD R4）

- Trellis 任务：`10-24-orphan-types-and-relaykit-policy`（PRD 见同目录 `prd.md` 的 **R4**）
- 共享任务：task-60「relaykit 收敛：删 18 项 + dto.StringValue + README API 政策」
- 执行者：worker-script2
- 日期：2026-10-24
- 前提：**用户已拍板**——relaykit **不计划独立发布**，定位＝主仓库的内部拆分模块，因此可以收敛导出 API。
- 边界：未提交、未推送、未发版、未联网、未装工具、未动 `web/`、`data/`、实例；未把 deadcode 写进 `go.mod`
- 基线：HEAD = `47b2321`（task-57 的孤儿类型改动仍在工作区未提交，本任务改动叠加其上）
- 依据材料：`evidence-relaykit-policy.md`（scout 的 20 符号逐个清单与三方案对比）

## 0. 结论

| 指标 | 本轮前 | 本轮后 | 说明 |
|---|---|---|---|
| relaykit 删除 | - | **16 个导出函数 + 1 个孤儿类型**（17 项） | 真死 14 项（13 函数 + `dto.StringValue` 类型本体）+ 仅模块自测引用 3 项函数 |
| 根模块 `deadcode -test ./...` | 53 | **37**（-16） | 与删除的 16 个函数一一对应；无级联新增 |
| relaykit 模块 `GOWORK=off deadcode -test ./...` | 109 | **96**（-13） | 按符号名 diff：移除 13 / 新增 0 |
| `golangci-lint unused`（根模块） | 1（`logKeyCol`） | **1（`logKeyCol`）** | 不变 |
| 测试调用点改写 | - | **2 个测试文件、3 处调用** | 材料/PRD 写的“3 个文件 4 处”与实际不符，以本证据 §3 实测为准 |
| 门禁 | 绿 | **绿**（根模块 + relaykit 独立 build/test） | 见 §6 |

- 保留的公开面 4 项（已写进 README 公开面清单）：`ConvertResponseByID`、`NewResponseStreamStateByID`、`LookupTextConverter`、`ChannelTypeOf`。
- README 新增「公开 API 政策」，并把「可 `go get ...@latest` 独立嵌入」的表述与「当前无发布流水线、无 tag、不计划独立发布」的现状对齐。

## 1. 分批执行记录

| 批次 | 内容 | 涉及文件 | 验证 |
|---|---|---|---|
| B1 | `dto.StringValue` 类型本体 + 2 个 JSON 方法；`kitutil.StringToByteSlice`；`types.NewFileMeta` / `NewImageFileMeta`（5 项） | `relaykit/dto/values.go`、`relaykit/relayconvert/kitutil/json.go`、`relaykit/types/request_meta.go` | 根模块 build+test exit=0；relaykit build+test exit=0 |
| B2 | 8 个真死兼容门面（4 个 request + 4 个 response） | `relaykit/relayconvert/request_compat.go`、`response_compat.go` | 同上，全绿 |
| B3 | 3 个仅模块自测引用的兼容门面 + 改写 3 处测试调用 | `request_compat.go`、`safety_settings_test.go`、`claude_default_max_tokens_test.go` | 同上，全绿 |
| B4 | `relaykit/README.md`：新增「公开 API 政策」、对齐安装/定位/版本表述 | `relaykit/README.md` | 文档改动，见 §4 |

每批后都实际执行了：根模块 `go build ./...` + `go test ./... -count=1`，以及 `cd relaykit && GOWORK=off go build ./...`（B1/B2/B3 还额外跑了 `GOWORK=off go test ./... -count=1`）。全程零失败，**未发生回退**。

最终 diff（`git diff --numstat -- relaykit`，+32/-100）：

```console
$ git diff --numstat -- relaykit
26	4	relaykit/README.md
0	22	relaykit/dto/values.go
2	1	relaykit/relayconvert/claude_default_max_tokens_test.go
0	7	relaykit/relayconvert/kitutil/json.go
0	31	relaykit/relayconvert/request_compat.go
0	16	relaykit/relayconvert/response_compat.go
4	2	relaykit/relayconvert/safety_settings_test.go
0	17	relaykit/types/request_meta.go
```

## 2. 删除清单与四类取证（16 函数 + 1 类型）

`删前位置` 取自本轮前根模块 `deadcode -test ./...` 的 53 行清单（`evidence-symbols.md` 同源）。四类标准：

① 可达性（根模块 deadcode + relaykit 模块自测视角两套）；② 全仓 Go 引用（含 `_test.go` 与 build tag）；③ 非 Go 契约位置引用（README/MAINTENANCE/AGENTS.md/CLAUDE.md/`.trellis/spec`/scripts/.github/web）；④ 反射/序列化/注册表依赖。

| # | 符号 | 删前位置 | ① 可达性 | ② Go 引用（删前） | ③ 非 Go 契约位置 | ④ 反射/序列化 | 处置 |
|---|---|---|---|---|---|---|---|
| 1 | `dto.StringValue.UnmarshalJSON` | `relaykit/dto/values.go:10` | 根不可达；模块自测视角「仍报」（真死） | 仅方法定义 + 类型定义；根模块 0 引用 | 0 | 无注册表/反射；类型本身 0 实例化 | **已删** |
| 2 | `dto.StringValue.MarshalJSON` | `relaykit/dto/values.go:26` | 同上 | 同上 | 0 | 同上 | **已删** |
| 3 | `dto.StringValue`（类型本体） | `relaykit/dto/values.go:8-28` | deadcode 不报类型；人工核：`var _`/断言/复合字面量/切片/指针/映射探针 0 命中；删掉 2 方法后仅剩定义 | 删除方法后 0 引用；同族 `IntValue`（根模块用 9 次）、`BoolValue`（用 5 次）保留 | 0 | 不再有 `MarshalJSON/UnmarshalJSON`；无序列化目标 | **已删** |
| 4 | `kitutil.StringToByteSlice` | `relaykit/relayconvert/kitutil/json.go:59` | 根不可达；模块「仍报」（真死） | 仅定义（其原消费者 `UnmarshalJsonStr` 已于 10-24 删除） | 0 | 无 | **已删**（连带 `unsafe` import） |
| 5 | `types.NewFileMeta` | `relaykit/types/request_meta.go:41` | 根不可达；模块「仍报」（真死） | 0（模块内统一用 `&FileMeta{...}` 字面量） | 0 | 构造器，无反射/注册表 | **已删** |
| 6 | `types.NewImageFileMeta` | `relaykit/types/request_meta.go:49` | 根不可达；模块「仍报」（真死） | 0 | 0 | 同上 | **已删** |
| 7 | `relayconvert.ClaudeMessagesRequestToOpenAIChat` | `relayconvert/request_compat.go:14` | 根不可达；模块「仍报」（真死） | relaykit 内仅定义；根模块 0 引用（`git grep HEAD` 排除 relaykit 后为空） | 0 | 兼容门面，无反射 | **已删** |
| 8 | `relayconvert.GeminiGenerateContentRequestToOpenAIChat` | `request_compat.go:22` | 同上 | 同上 | 0 | 同上 | **已删** |
| 9 | `relayconvert.ChatCompletionsRequestToResponsesRequest` | `request_compat.go:34` | 同上 | 同上 | 0 | 同上 | **已删** |
| 10 | `relayconvert.ResponsesRequestToChatCompletionsRequest` | `request_compat.go:38` | 同上 | 同上 | 0 | 同上 | **已删** |
| 11 | `relayconvert.StopReasonClaudeToOpenAI` | `response_compat.go:31` | 同上 | 同上 | 0 | 同上 | **已删** |
| 12 | `relayconvert.ResponsesStatusFromChatFinishReason` | `response_compat.go:87` | 同上 | 同上 | 0 | 同上 | **已删** |
| 13 | `relayconvert.ResponsesFinishReasonFromStatus` | `response_compat.go:107` | 同上 | 同上 | 0 | 同上 | **已删** |
| 14 | `relayconvert.ExtractReasoningTextFromResponses` | `response_compat.go:123` | 同上 | 同上 | 0 | 同上 | **已删** |
| 15 | `relayconvert.OpenAIChatRequestToGeminiGenerateContent` | `request_compat.go:26` | 根不可达；模块自测视角「未报」= **仅 relaykit 自身测试在用** | 仅 `safety_settings_test.go:23` | 0 | 无 | **已删 + 测试改写**（§3） |
| 16 | `relayconvert.OpenAIResponsesRequestToClaudeMessages` | `request_compat.go:42` | 同上 | 仅 `claude_default_max_tokens_test.go:36` | 0 | 无 | **已删 + 测试改写** |
| 17 | `relayconvert.OpenAIResponsesRequestToGeminiChat` | `request_compat.go:46` | 同上 | 仅 `safety_settings_test.go:37` | 0 | 无 | **已删 + 测试改写** |

补充核对：

- **根模块零调用点**：对 16 个函数执行 `git grep -n -w <Symbol> HEAD -- '*.go' | grep -v '^HEAD:relaykit/'`，除同名但无关的根模块自有符号（`common.StringToByteSlice`、aws 的 `eventstream.StringValue`）外为空——这 16 个函数从未被主仓库业务代码调用。
- **兼容门面的实际用户是注册表**：`request_registry.go` 调用的是 `internal/*` 的直接实现（如 `oaichat.OpenAIChatRequestToGeminiGenerateContent`），与被删的 `relayconvert` 同名门面是两个符号；删门面不改变注册表路径。
- **③ 契约位置扫描**：四个可检索名字（`StringValue`/`StringToByteSlice`/`NewFileMeta`/`NewImageFileMeta`）在 README/MAINTENANCE/AGENTS.md/CLAUDE.md/`.trellis/spec`/scripts/.github/web/src 全部 0 命中（`web/` 里唯一命中来自第三方打包产物中的同名无关标识，已按契约路径排除）。README 新政策段**有意**提及被删函数名（作为删除记录，非悬空引用）。
- **④ 序列化**：被删的 `dto.StringValue` 的 2 个 JSON 方法在删除类型后不再存在；其余带 `json` tag 的类型不受影响。GORM/reflect/`%T` 依赖均无。

## 3. 测试调用点改写（2 文件 / 3 处，断言零放宽）

材料/PRD 写的是“3 个测试文件、4 处调用”，实际 grep（HEAD）为 **2 个文件、3 处调用**——已按实际执行并记录：

```console
$ git grep -n -E "OpenAIChatRequestToGeminiGenerateContent|OpenAIResponsesRequestToClaudeMessages|OpenAIResponsesRequestToGeminiChat" HEAD -- 'relaykit/relayconvert/*_test.go'
HEAD:relaykit/relayconvert/claude_default_max_tokens_test.go:36:  return OpenAIResponsesRequestToClaudeMessages(...)
HEAD:relaykit/relayconvert/safety_settings_test.go:23:         got, err := OpenAIChatRequestToGeminiGenerateContent(...)
HEAD:relaykit/relayconvert/safety_settings_test.go:37:         got, err := OpenAIResponsesRequestToGeminiChat(...)
```

改写方式：直接调用被门面转发的 `internal` 实现（`relaykit/relayconvert/internal/oai_chat`、`internal/oai_responses`），**测试用例名、输入与断言全部不变**；`internal` 导入在本包测试中已有先例（`claude_default_max_tokens_test.go` 原本就 import `internal/shared/claude`，`request_registry_test.go` import `internal/shared/gemini`）。

```diff
--- relaykit/relayconvert/safety_settings_test.go
-				got, err := OpenAIChatRequestToGeminiGenerateContent(context.Background(), dto.GeneralOpenAIRequest{
+				got, err := oaichat.OpenAIChatRequestToGeminiGenerateContent(context.Background(), dto.GeneralOpenAIRequest{
...
-				got, err := OpenAIResponsesRequestToGeminiChat(context.Background(), &dto.OpenAIResponsesRequest{
+				got, err := oairesponses.OpenAIResponsesRequestToGeminiChat(context.Background(), &dto.OpenAIResponsesRequest{

--- relaykit/relayconvert/claude_default_max_tokens_test.go
-				return OpenAIResponsesRequestToClaudeMessages(context.Background(), meta, &dto.OpenAIResponsesRequest{
+				return oairesponses.OpenAIResponsesRequestToClaudeMessages(context.Background(), meta, &dto.OpenAIResponsesRequest{
```

（`OpenAIChatRequestToClaudeMessages` 是**保留**的 live 门面，主仓库 `relay/gemini_handler.go` 等路径不涉及，但它的 2 处测试调用未改。）

## 4. README 变更（公开 API 政策 + 现状对齐）

### 4.1 定位与安装表述对齐

| 位置 | 改前 | 改后 |
|---|---|---|
| 开头定位段（原 :5） | 「因此可以脱离 new-api 主模块，嵌入其他 Go 网关或代理服务。」 | 保留“技术上可独立构建”，明确「**当前定位是主仓库的内部拆分模块，不计划独立发布**（无 `relaykit/v*` tag、无独立发布流水线、主仓库 `replace` 本地目录）」 |
| 「## 安装」段（原 :35-41） | `go get github.com/QuantumNous/new-api/relaykit@latest` | 段名改为「## 使用与安装」，写明**当前未独立发布、不能 `go get`**；给出仓库内 `cd relaykit && GOWORK=off go build/test` 方式 |
| 「## 版本与兼容性」 | 「升级前请检查发布说明…」 | 补充“版本号随主仓库演进、当前没有独立 tag、不承诺模块级兼容性” |

### 4.2 新增「## 公开 API 政策」（全文）

```markdown
## 公开 API 政策

**模块定位（2026-10-24 起）**：RelayKit 是 new-api 主仓库的**内部拆分模块**，不计划独立发布（无 `relaykit/v*` tag、无独立发布流水线、主仓库用 `replace` 本地引用）。下表的「公开面」是**主仓库自用契约**，不是对外兼容承诺。

| 分类 | 范围 | 变更要求 |
|---|---|---|
| 公开面 | `relaykit/dto`、`relaykit/types`、`relaykit/relayconvert` 的注册表入口（`ConvertRequest` / `ConvertRequestVia` / `ConvertRequestByID` / `ConvertResponse` / `ConvertStreamResponseChunk` / `FinalizeStreamResponse` / `NewResponseStreamState` / `SetMediaResolver` 等）、`relaykit/relayconvert/convmeta`、`relaykit/reasonmap` | 删除/改名/改签名必须同步主仓库全部调用点，并更新本清单与主仓库发布说明 |
| 注册表与内省入口（公开面，保留） | `ConvertResponseByID`、`NewResponseStreamStateByID`、`LookupTextConverter`、`ChannelTypeOf` | 同上；这 4 个是转换器注册表与上下文内省 API，主仓库与模块测试依赖 |
| 内部面（不承诺兼容） | `relayconvert/internal/**`、`relayconvert/kitutil`、`request_compat.go` / `response_compat.go` 的历史兼容门面 | 可自由删改，但必须保持主仓库可编译、两套门禁全绿 |

变更流程：

1. 改公开面 → 同步主仓库调用点 → 跑主仓库 `go build ./...`、`go test ./... -count=1`，以及 `cd relaykit && GOWORK=off go build ./... && GOWORK=off go test ./...`；
2. 在本节「公开面」清单与主仓库发布说明（`MAINTENANCE.md` 的发布记录）中记录删改；
3. 内部面清理不需要发布说明，但需保持本节与「JSON 编码约定（模块例外）」同步，避免文档腐化。

最近一次收敛（2026-10-24）：删除 16 个无人使用的导出函数（8 个真死兼容门面、3 个仅模块自测引用的兼容门面、`kitutil.StringToByteSlice`、`types.NewFileMeta` / `NewImageFileMeta`、`dto.StringValue` 的 2 个 JSON 方法）与孤儿类型 `dto.StringValue` 本体；3 个仅自测引用的门面删除后，测试改为直接调用 `internal` 实现；保留上表 4 个注册表/内省入口。
```

文档位置：新增段在「## 转换结果」之后、「## 开发」之前；保留项在原有 README:231（现 :231）的叙述与政策表两处都有体现。

## 5. 两套 deadcode 数字与级联分析

### 5.1 根模块视角（标准命令）

```console
# 本轮前
$ /root/go/bin/deadcode -test ./... | wc -l
53
# 本轮后
$ /root/go/bin/deadcode -test ./... | wc -l
37
```

- 减少 **16 行**，与删除的 16 个函数严格对应（`StringValue` 类型本体不参与 deadcode 统计，故不单独计数）。
- 后 37 行分布：`common` 29 / `relay` 1 / `relaykit` 5 / `service` 1 / `setting` 1。
- 剩余 relaykit 5 项 = 保留的 4 个公开面入口 + 非导出的 `resolveTextConverterID`（被保留的 `LookupTextConverter` 内部调用，随公开面保留）：

```text
relaykit/relayconvert/response_registry.go:231   ConvertResponseByID
relaykit/relayconvert/response_registry.go:293   NewResponseStreamStateByID
relaykit/relayconvert/text_converter_registry.go:258  LookupTextConverter
relaykit/relayconvert/text_converter_registry.go:356  resolveTextConverterID
relaykit/relayconvert/convmeta/meta.go:202       ChannelTypeOf
```

### 5.2 relaykit 模块自测视角

```console
# 本轮前（GOWORK=off，在 relaykit/ 下）
$ GOWORK=off /root/go/bin/deadcode -test ./... | wc -l
109
# 本轮后
$ GOWORK=off /root/go/bin/deadcode -test ./... | wc -l
96
```

按**符号名**（忽略行号漂移）做集合 diff：

```console
$ comm -23 <(sort before-names) <(sort after-names)   # 移除 13 个
StringValue.UnmarshalJSON / StringValue.MarshalJSON / StringToByteSlice /
ClaudeMessagesRequestToOpenAIChat / GeminiGenerateContentRequestToOpenAIChat /
ChatCompletionsRequestToResponsesRequest / ResponsesRequestToChatCompletionsRequest /
StopReasonClaudeToOpenAI / ResponsesStatusFromChatFinishReason /
ResponsesFinishReasonFromStatus / ExtractReasoningTextFromResponses /
NewFileMeta / NewImageFileMeta
$ comm -13 <(sort before-names) <(sort after-names)   # 新增 0 个
(无输出)
```

- **-13 = 真死的 13 个函数**（8 门面 + `StringToByteSlice` + 2 构造器 + 2 JSON 方法，`StringValue` 类型本体不计数）。
- 3 个“仅自测引用”的门面在模块视角本来就不在不可达清单里（被自身测试调用），删除后测试改为直接调 `internal`，因此也**不产生新的不可达项** → 新增 0，**无级联**。
- 行号漂移（`request_compat.go` / `response_compat.go` / `values.go` 内位置前移）是唯一“看起来变化”的项，按符号名已排除。

## 6. 门禁（最终树实测，`GOPROXY=off`）

```console
### root: go build ./...
exit=0
### root: go vet ./...
exit=0
### root: go test ./... -count=1
...（36 个包 ok，FAIL 行 0）
exit=0
### relaykit: GOWORK=off go build ./...
exit=0
### relaykit: GOWORK=off go test ./... -count=1
...（7 个包 ok，FAIL 行 0）
exit=0
### gofmt -l . (root)
(无输出)  exit=0
### gofmt -l relaykit
(无输出)  exit=0
### golangci-lint run --default=none --enable=unused ./...
model/main.go:27:5: var logKeyCol is unused (unused)
1 issues:
* unused: 1
### root deadcode
37
### relaykit module deadcode
96
```

## 7. 回滚方式

纯删除 + 测试改写 + 文档；未提交，按文件回滚即可（不触碰 `pkg/`、`web/` 等其它未提交改动）：

```bash
git checkout -- relaykit/dto/values.go \
  relaykit/relayconvert/kitutil/json.go \
  relaykit/relayconvert/request_compat.go \
  relaykit/relayconvert/response_compat.go \
  relaykit/relayconvert/safety_settings_test.go \
  relaykit/relayconvert/claude_default_max_tokens_test.go \
  relaykit/types/request_meta.go \
  relaykit/README.md
```

回滚后重跑 §6 门禁；期望恢复到根模块 deadcode 53 / relaykit 模块 109。

## 8. 边界自证与风险

```console
$ git status --short -- relaykit
 M relaykit/README.md
 M relaykit/dto/values.go
 M relaykit/relayconvert/claude_default_max_tokens_test.go
 M relaykit/relayconvert/kitutil/json.go
 M relaykit/relayconvert/request_compat.go
 M relaykit/relayconvert/response_compat.go
 M relaykit/relayconvert/safety_settings_test.go
 M relaykit/types/request_meta.go
```

- 本任务只改上述 8 个 relaykit 文件与本证据文件；未碰 `web/`（locale 改动属 worker-docs）、`data/`、实例；未提交、未推送、未发版；`go.mod`/`go.sum` 无改动；deadcode 未写入依赖。
- 风险：删除的都是主仓库零调用点符号（`git grep` 证明根模块从未调用），对编译、测试、运行时行为零影响；若未来改变“不独立发布”的定位，需要重新评估这次收敛（当前用户已裁定不发布，README 政策已记录）。
- 未验证项（沿用 scout 材料）：无法联网核实是否存在 GitHub 外部下游 import；但既然定位为内部拆分且无 tag，该风险由用户定位裁定覆盖。

渲染安全自查：正文无成对美元符号（美元符号只出现在 console 代码块的命令提示符中）；不使用单星号斜体；表格列数一致；反引号代码片段已配对。
