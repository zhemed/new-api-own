# relaykit 20 个导出 API 面：决策材料（只读）

对象：`.trellis/tasks/archive/2026-10/10-24-panic-recovery-and-cleanup/evidence-symbols.md` 列出的 relaykit 导出面，按**当前** `deadcode -test ./...` 重新取数
性质：只读决策支持——**未删改任何代码**（唯一写入为本文件）；不发版、不提交、不联网

---

## 一、结论速览（给决策者）

| 项 | 事实 |
|---|---|
| 当前 relaykit 死符号 | 根模块视角 `deadcode -test ./...` 共 53 条，其中 **relaykit 21 条 = 20 个导出符号 + 1 个非导出实现**（`resolveTextConverterID`） |
| 其中「模块自身测试也不覆盖」 | **13 个 deadcode 条目**（含 `StringValue` 的 2 个方法）+ **孤儿类型本体 1 个** = 14 项真死（删了对本仓库编译、测试、功能零影响） |
| 其中「仅 relaykit 自身测试在用」 | **7 个导出符号**（删需改写 3 个测试文件的调用点；非运行时影响） |
| 文档点名 | 只有 2 个：`ConvertResponseByID`、`NewResponseStreamStateByID`（relaykit/README.md:227） |
| 模块定位现状 | 独立 `go.mod` + 根模块 `replace`；README 宣称可外部嵌入并给出 `go get ...@latest`；**但没有任何发布流程，也没有 `relaykit/vX.Y.Z` tag** → 当前"对外发布"是意图而非事实 |
| 使用者 | 根模块 11 个顶层目录、约 214 个文件 import 它（relay 146 文件最多）；relaykit 内部 68 个文件自引用 |
| 附加发现 | `relaykit/dto.StringValue` 是**第 5 个孤儿导出类型**（连同其 2 个 JSON 方法），与 PRD R1 的 4 个孤儿类型同性质 |

**推荐：方案 C（分批收敛 + 补 API 政策）**——先删 14 项真死（零影响），把 7 项"有测试覆盖或有文档点名"的逐个定策，并在 `relaykit/README.md` 增加一节「公开 API 政策」。理由见第六节。

---

## 二、模块定位事实（可复核）

| 事实 | 证据 |
|---|---|
| 模块 path | `github.com/QuantumNous/new-api/relaykit` | relaykit/go.mod:1 |
| Go 版本 | `go 1.25.1` | relaykit/go.mod:3 |
| 直接依赖 | uuid、samber/lo、testify、tidwall/gjson、tidwall/sjson（5 个；无 Gin/DB/根模块依赖） | relaykit/go.mod:5-11 |
| 根模块如何引用 | `github.com/QuantumNous/new-api/relaykit v0.0.0` + `replace ... => ./relaykit` | go.mod:167、go.mod:170 |
| 是否从 registry 解析 | 否：replace 指向本地目录；`grep -c relaykit go.sum` = **0**；无 `vendor/`、无 `go.work` | go.sum、目录列表 |
| 独立构建要求 | `cd relaykit && GOWORK=off go build ./...`（AGENTS.md 约束 + README:229-237） | AGENTS.md「relaykit module independence」、relaykit/README.md:231 |
| CI 是否构建/测试 relaykit | **没有**任何 workflow 步骤编译或测试该模块；`.github/workflows/` 中 relaykit 只出现在 2 行注释里（说明根构建为何用 `GOWORK=off`） | .github/workflows/release.yml:145、.github/workflows/lint.yml:45 |
| CI 是否发布 relaykit | **没有**发布步骤：无 goreleaser、无 `git tag`/`git push --tags`、无 module proxy 动作；release.yml 只发布主程序二进制与镜像（且有签名流程） | .github/workflows/release.yml:145-165、:243、:264 |
| 是否存在模块 tag | 本地 `git tag -l 'relaykit/*'` **为空**；现有 tag 全是 `v0.0.x` 主程序 tag | git tag 输出 |
| README 自述定位 | 「本项目拆分出的独立 Go 模块」；「只负责协议层…**可以脱离 new-api 主模块，嵌入其他 Go 网关或代理服务**」 | relaykit/README.md:3、:5 |
| README 是否教外部安装 | 是：`go get github.com/QuantumNous/new-api/relaykit@latest` | relaykit/README.md:40 |
| 版本与兼容承诺 | 当前 `v0.x`，「公开 API 和 DTO 仍可能在小版本中调整」 | relaykit/README.md:260 |
| 既有的"模块例外"先例 | 2026-10-24 已删 `kitutil.UnmarshalJsonStr` / `kitutil.DecodeJson` 两个无人使用的同名包装，并写入 README 说明 | relaykit/README.md:244-256 |
| 其它仓库内引用 | `docs/FILE_INVENTORY.md:13` 把 relaykit(132 文件) 记为保留的独立模块；`scripts/forbid-encoding-json.sh:4-8` 给 relaykit/ 开了 JSON 直调白名单 | docs/FILE_INVENTORY.md:13、scripts/forbid-encoding-json.sh:4-8 |

关键推论（供决策）：

- 【已证实】"对外可发布"目前**没有落地机制**：无 tag、无发布流水线、根模块用 replace + `v0.0.0` 占位。`go get ...@latest` 在无模块 tag 时只能解析到伪版本（需仓库公开且可被 proxy 抓取；**这条未联网验证**）。
- 【已证实】只要有 `replace`，主仓库构建与 CI 都不会从远端拉 relaykit；因此删除模块内死符号对本仓库毫无风险（唯一影响是模块自身测试，见第四节）。
- 【未验证项】是否有外部仓库/用户 import 了该模块、GitHub 上是否有下游使用：**本轮不联网，无法核实**。

---

## 三、使用者盘点

### 3.1 根模块（relaykit 之外）按顶层目录

| 顶层目录 | import 语句数 | 文件数 |
|---|---|---|
| relay | 256 | 146 |
| service | 44 | 31 |
| controller | 18 | 14 |
| model | 11 | 10 |
| middleware | 6 | 5 |
| constant | 2 | 2 |
| setting | 2 | 2 |
| router | 1 | 1 |
| main.go | 1 | 1 |
| i18n | 1 | 1 |
| common | 1 | 1 |
| 合计 | 343 | 约 214 |

### 3.2 按 relaykit 内部包（根模块引用次数）

| relaykit 包 | 根模块引用次数 | 说明 |
|---|---|---|
| relaykit/dto | 185 | 主用面：DTO |
| relaykit/types | 135 | 主用面：共享类型/文件来源/错误 |
| relaykit/relayconvert | 16 | 转换入口 |
| relaykit/relayconvert/convmeta | 3 | 转换上下文 |
| relaykit/relayconvert/kitutil | 2 | 计数与 JSON 包装 |
| relaykit/relayconvert/reasoning | 2 | 推理档位 |

（relaykit 模块内部自引用另有 68 个文件；上表只统计根模块。）

### 3.3 对外使用迹象

- 支持"可能对外"的证据：README:3/:5 自述可嵌入其它网关；README:40 教 `go get`；README:260 给出 v0.x 兼容政策雏形；模块刻意不依赖根模块/Gin/DB（relaykit/go.mod 依赖表）。
- 反对"已经对外"的证据：无发布流水线、无 `relaykit/v*` tag、根模块 replace 到本地、go.sum 无条目。
- **未验证项**：外部仓库是否 import、是否有下游反馈——本轮不联网，无法核实，需用户判断或后续联网核实。

---

## 四、20 个导出符号逐个清单（+1 个非导出附项）

判定列说明：
- **根模块视角**：`deadcode -test ./...`（仓库标准命令）是否判不可达 —— 21 条全部"不可达"。
- **模块自测视角**：`cd relaykit && GOWORK=off deadcode -test ./...`（把 relaykit 自己的测试二进制作为根；该跑法输出 109 条）——"仍报"= 连模块自身测试都不覆盖（真死）；"未报"= 被模块自身测试调用（删需改测试）。
- 【口径更正】根模块视角的 `-test` **只把根模块自己的测试当根**，不含依赖模块（relaykit）的测试；因此上一轮"清单内无仅测试存活的误报"这句话对 relaykit 项不成立，本表用两套视角分别标注。

| # | 符号（包.名） | 签名摘要 | 根模块 deadcode | 模块自测视角 | 文档点名 | 语义归属 | 删了会影响谁 |
|---|---|---|---|---|---|---|---|
| 1 | relaykit/dto.StringValue.UnmarshalJSON | `(s *StringValue) UnmarshalJSON(data []byte) error` | 不可达 | 仍报（真死） | 无 | 孤儿类型的方法 | 无人引用（类型本身也无人用） |
| 2 | relaykit/dto.StringValue.MarshalJSON | `(s StringValue) MarshalJSON() ([]byte, error)` | 不可达 | 仍报（真死） | 无 | 孤儿类型的方法 | 无人引用 |
| 3 | relayconvert.ClaudeMessagesRequestToOpenAIChat | `ClaudeMessagesRequestToOpenAIChat(claudeRequest dto.ClaudeRequest, info convmeta.Meta) (*dto.GeneralOpenAIRequest, error)` | 不可达 | 仍报（真死） | 无 | 兼容层门面（转调 internal/claude_messages） | 无人引用（注册表走 internal 直调） |
| 4 | relayconvert.GeminiGenerateContentRequestToOpenAIChat | `GeminiGenerateContentRequestToOpenAIChat(geminiRequest *dto.GeminiChatRequest, info convmeta.Meta) (*dto.GeneralOpenAIRequest, error)` | 不可达 | 仍报（真死） | 无 | 兼容层门面（internal/gemini_chat） | 无人引用 |
| 5 | relayconvert.OpenAIChatRequestToGeminiGenerateContent | `OpenAIChatRequestToGeminiGenerateContent(c context.Context, textRequest dto.GeneralOpenAIRequest, info convmeta.Meta) (*dto.GeminiChatRequest, error)` | 不可达 | **未报（自测覆盖）** | 无 | 兼容层门面（internal/oai_chat） | relaykit/relayconvert/safety_settings_test.go:23 |
| 6 | relayconvert.ChatCompletionsRequestToResponsesRequest | `ChatCompletionsRequestToResponsesRequest(req *dto.GeneralOpenAIRequest) (*dto.OpenAIResponsesRequest, error)` | 不可达 | 仍报（真死） | 无 | 兼容层门面 | 无人引用（同名 internal 函数另有测试） |
| 7 | relayconvert.ResponsesRequestToChatCompletionsRequest | `ResponsesRequestToChatCompletionsRequest(req *dto.OpenAIResponsesRequest) (*dto.GeneralOpenAIRequest, error)` | 不可达 | 仍报（真死） | 无 | 兼容层门面 | 无人引用 |
| 8 | relayconvert.OpenAIResponsesRequestToClaudeMessages | `OpenAIResponsesRequestToClaudeMessages(c context.Context, info convmeta.Meta, req *dto.OpenAIResponsesRequest) (*dto.ClaudeRequest, error)` | 不可达 | **未报（自测覆盖）** | 无 | 兼容层门面 | relaykit/relayconvert/claude_default_max_tokens_test.go:36 |
| 9 | relayconvert.OpenAIResponsesRequestToGeminiChat | `OpenAIResponsesRequestToGeminiChat(c context.Context, req *dto.OpenAIResponsesRequest, info convmeta.Meta) (*dto.GeminiChatRequest, error)` | 不可达 | **未报（自测覆盖）** | 无 | 兼容层门面 | relaykit/relayconvert/safety_settings_test.go:37 |
| 10 | relayconvert.StopReasonClaudeToOpenAI | `StopReasonClaudeToOpenAI(reason string) string` | 不可达 | 仍报（真死） | 无 | 兼容层门面（internal/claude_messages） | 无人引用 |
| 11 | relayconvert.ResponsesStatusFromChatFinishReason | `ResponsesStatusFromChatFinishReason(finishReason string) (string, *dto.IncompleteDetails)` | 不可达 | 仍报（真死） | 无 | 兼容层门面（internal/oai_chat） | 无人引用 |
| 12 | relayconvert.ResponsesFinishReasonFromStatus | `ResponsesFinishReasonFromStatus(resp *dto.OpenAIResponsesResponse) (string, bool)` | 不可达 | 仍报（真死） | 无 | 兼容层门面（internal/oai_responses） | 无人引用 |
| 13 | relayconvert.ExtractReasoningTextFromResponses | `ExtractReasoningTextFromResponses(resp *dto.OpenAIResponsesResponse) string` | 不可达 | 仍报（真死） | 无 | 兼容层门面（internal/oai_responses） | 无人引用 |
| 14 | relayconvert.ConvertResponseByID | `ConvertResponseByID(c context.Context, info convmeta.Meta, converter string, response any) (*ResponseResult, error)` | 不可达 | **未报（自测覆盖）** | relaykit/README.md:227 | 公共入口（按转换器 ID 执行） | relaykit/relayconvert/response_registry_test.go:238、:246；外部使用者（未验证） |
| 15 | relayconvert.NewResponseStreamStateByID | `NewResponseStreamStateByID(converter string, options ResponseStreamOptions) (*ResponseStreamState, error)` | 不可达 | 仍报（真死） | relaykit/README.md:227 | 公共入口（流式状态） | 当前无人（文档承诺的入口却无实现方调用，属"文档面"） |
| 16 | relayconvert.LookupTextConverter | `LookupTextConverter(converter string) (TextConverterSpec, bool)` | 不可达 | **未报（自测覆盖）** | 无 | 注册表内省 API | relaykit/relayconvert/text_converter_registry_test.go:120、:132 |
| 17 | relayconvert/convmeta.ChannelTypeOf | `ChannelTypeOf(m Meta) int` | 不可达 | **未报（自测覆盖）** | 无 | 公共包内省访问器（nil-safe） | relaykit/relayconvert/convmeta/meta_test.go:37 |
| 18 | relayconvert/kitutil.StringToByteSlice | `StringToByteSlice(s string) []byte` | 不可达 | 仍报（真死） | 无 | 内部工具泄漏（原为已删两个包装服务） | 无人引用 |
| 19 | relaykit/types.NewFileMeta | `NewFileMeta(fileType FileType, source FileSource) *FileMeta` | 不可达 | 仍报（真死） | 无 | 公开类型构造器 | 无人引用（模块内统一用 `&FileMeta{...}` 字面量，见 relaykit/dto/claude.go:271 等） |
| 20 | relaykit/types.NewImageFileMeta | `NewImageFileMeta(source FileSource, detail string) *FileMeta` | 不可达 | 仍报（真死） | 无 | 公开类型构造器 | 无人引用 |
| 附 | relayconvert.resolveTextConverterID（**非导出**） | `resolveTextConverterID(converter string) string` | 不可达 | **未报（自测覆盖）** | 无 | 内部实现（LookupTextConverter 调用） | 随第 16 项一起决定 |

分组小结（20 个导出 + 1 个非导出）：

| 语义归属 | 数量 | 成员 |
|---|---|---|
| 兼容层门面 | 11 | #3-#13（8 个真死 + 3 个仅自测覆盖） |
| 公共入口（README 点名） | 2 | #14、#15 |
| 注册表/上下文内省 API | 2 | #16、#17 |
| 孤儿类型的方法 | 2 | #1、#2 |
| 内部工具泄漏 | 1 | #18 |
| 公开类型构造器 | 2 | #19、#20 |
| 非导出实现（附） | 1 | resolveTextConverterID |

附加发现（建议一并决策）：`relaykit/dto.StringValue` **类型本身是孤儿导出类型**——全仓（含测试）除定义处外 0 引用；同文件的 `IntValue`（根模块用 9 次）、`BoolValue`（用 5 次）都在用，`StringValue` 是家族里唯一孤儿。它与 PRD R1 的 4 个孤儿类型（`pkg/cachex.StringCodec`、`pkg/ionet.{ContainerLogs,LogEntry,LocationAvailability}`）同性质，可作为**第 5 个孤儿类型**同批处理（但属模块 API，需用户点头）。

---

## 五、三种方案对比

| 维度 | A 全删收敛 | B 冻结保留 + 写 API 政策 | C 分批收敛 + 写 API 政策（推荐） |
|---|---|---|---|
| 动作 | 删 20 个导出符号 + 1 个非导出实现与孤儿类型 `StringValue`；改写 3 个测试文件的 4 处调用；删 README:227 的两处点名 | 代码不动；在 relaykit/README.md 增加「公开 API 政策」，把 20 个列为公开面并声明兼容/不保证 | 第 1 批删 14 项真死（含孤儿类型 StringValue）；第 2 批对 7 项（3 个自测覆盖门面、README 2 项、内省 2 项）逐项定策；同时补 API 政策文档 |
| 规模 | 21 个函数体合计约 115 行 + 类型声明 1 行；改写 3 个测试文件 4 处调用；README 1 行改写 | 0 行代码；README 新增约 20-30 行政策段 | 第 1 批约 62 行、涉及 5 个文件、零测试改动；第 2 批视裁定：保留则 0 行，删除则约 49 行 + 3 个测试文件 4 处调用 |
| 收益 | 死符号归零，API 面最小，deadcode 基线最干净 | 零风险，保住"未来要发布"的可能性 | 死代码大幅下降且零行为风险；公开面被显式写清，后续清理有判据 |
| 风险 | 若未来发布/外部已 import（未验证），属破坏性变更；删掉 README 推荐入口会削弱模块对外可用性 | 20 个死符号常驻；deadcode 基线永久带噪；每轮清理都要重新争论同一问题 | 需要两轮小步执行与一次文档决策；第 2 批若删需改写 3 处测试 |
| 回滚难度 | 低（逐文件 revert；但一旦发布过则外部不可逆） | 极低（纯文档） | 低（分批提交，可逐批 revert；第 1 批不碰测试与公开入口语义） |
| 与既有先例一致性 | 中（此前删的是"无人用 + 无文档"的 kitutil 包装，本次 A 会连文档点名的入口一起删） | 中（与"删死代码"取向相反） | 高（完全沿用 2026-10-24 kitutil 的处理模式：删无人用 + 写模块例外/政策） |

---

## 六、推荐与理由

**推荐方案 C：分批收敛 + 补 API 政策。** 具体建议：

1. **第 1 批（真死 14 项，零风险）**：8 个真死门面（#3、#4、#6、#7、#10、#11、#12、#13）+ `StringToByteSlice`（#18）+ `NewFileMeta`/`NewImageFileMeta`（#19、#20）+ `StringValue` 孤儿类型及其 2 个方法（#1、#2 与类型本体）。
   验证：`cd relaykit && GOWORK=off go build ./... && GOWORK=off go test ./...`；根模块 `go build ./... && go test ./... -count=1`；根模块 `deadcode -test ./...` 应从 53 降到 40（-13 条；类型本身不计入 deadcode 输出）。
2. **第 2 批（7 项，需用户先拍板定位）**：
   - README 点名的 2 个（`ConvertResponseByID`、`NewResponseStreamStateByID`）+ 内省 2 个（`LookupTextConverter`、`ChannelTypeOf`）→ 若确定模块对外：写进 README 的「公开 API」表并注明兼容承诺，保留（归 C 堆）；
   - 3 个仅自测覆盖的门面（#5、#8、#9）→ 若确定"注册表入口是唯一推荐路径"：删除并把这 3 处测试调用改到 `relayconvert` 注册表或 internal 包（改动仅 3 个测试文件、4 行调用）。
3. **API 政策文档**：在 `relaykit/README.md` 增加一节，明确：
   - 公开面（承诺兼容）：`relaykit/dto`、`relaykit/types`、`relaykit/relayconvert` 的注册表入口（`ConvertRequest*` / `ConvertResponseByID` / `NewResponseStreamStateByID` 等）、`relaykit/relayconvert/convmeta`、`relaykit/reasonmap`；
   - 内部面（不承诺兼容）：`relayconvert/internal/**`、`relayconvert/kitutil`、`request_compat.go` / `response_compat.go` 的门面别名；
   - 变更流程：删/改公开面需在 README 与发布说明记录（当前 v0.x 允许小版本调整）。

**理由（按证据权重）**：

- "破坏性变更"风险目前缺乏事实支撑：无 tag、无发布流水线、根模块 replace 本地（第二节全部可复核）。因此把 20 个死符号一律按"对外契约"冻结，成本（永久死代码 + 每轮争论）高于收益。
- 但也不能一次全删：README 明确把模块定位为可嵌入外部（README:3/:5/:40），且已把这 2 个入口写进文档（README:227）——文档承诺过的东西不该被代码清理顺手删掉，应先补齐政策再动。
- 13 项真死在模块自身测试视角下依然不可达（第四节"仍报"列），删除对编译、测试、行为**零影响**；这是本轮唯一"无争议"的收益，应立刻落袋。
- 与既有先例一致：kitutil 两个包装就是按"无人用→删→README 记例外"处理的（README:244-256），沿用同一模式可避免每轮重新论证。

---

## 七、需要用户拍板的问题

| 编号 | 问题 | 为什么必须先答 | 影响范围 |
|---|---|---|---|
| Q1 | relaykit 未来是否需要**独立发布**（打 `relaykit/vX.Y.Z` tag、供外部 `go get`/import）？ | 这是 A/B/C 的分水岭：需要发布 → 走 B 或 C 的第 2 批保守路线；纯内部拆分 → 可直接走 A | 全部 20 个符号 |
| Q2 | 若保留对外定位：`ConvertResponseByID`、`NewResponseStreamStateByID`、`LookupTextConverter`、`ChannelTypeOf` 是否纳入**正式公开 API**（写进 README API 表并承诺兼容）？ | README 已点名前 2 个；后 2 个只有测试在用，定位不清会反复争论 | 4 个符号（#14-#17） |
| Q3 | `request_compat.go` / `response_compat.go` 这层**门面别名**的定位？保留（文档化）还是收敛（删死门面 + 改写 3 处测试）？ | 该层有 8-10 个 live 包装在用，说明是刻意设计；但 11 个死门面是否继续维护需明确 | 11 个符号（#3-#13） |
| Q4 | 是否同意把 `dto.StringValue` 作为**第 5 个孤儿导出类型**，与 R1 的 4 个孤儿类型同批删除？ | 属模块 API，超出 R1 原定对象；但性质完全相同 | 1 类型 + 2 方法（#1、#2） |
| Q5 | 是否接受在 relaykit/README.md 增加「公开 API 政策」段落（公开面 / 内部面 / 变更流程）？ | 后续 deadcode 清理与模块演进都需要这个判据；也是方案 C 的收尾动作 | 文档（约 20-30 行） |
| Q6 | 是否需要在有网络时核实外部是否已 import 该模块（GitHub 下游/搜索结果）？ | 本轮无法联网，"外部消费者"始终是未验证项；若 Q1 答"要发布"，建议先核实 | 全局风险判断 |

---

## 八、关键数据与复核命令

关键数据：

- 根模块视角：`deadcode -test ./...` = **53** 条；包分布 common 29 / relaykit 21 / relay 1 / service 1 / setting 1。
- relaykit 自身测试视角：`cd relaykit && GOWORK=off deadcode -test ./...` = **109** 条；上表 21 项中有 7 项**不再出现**（即被模块自身测试覆盖）。
- 根模块 import relaykit：343 条语句 / 约 214 个文件；relaykit 自引用 68 个文件。
- 发布/tag 事实：workflows 中 relaykit 仅 2 行注释；`git tag -l 'relaykit/*'` 为空；go.sum 无 relaykit 条目。

复核命令（本轮实际执行，全部只读）：

```bash
# 死符号两套视角
/root/go/bin/deadcode -test ./...
cd relaykit && GOWORK=off /root/go/bin/deadcode -test ./...

# 模块定位与发布事实
sed -n '1,20p' relaykit/go.mod
sed -n '160,175p' go.mod
grep -rn relaykit .github/workflows/
git tag -l 'relaykit/*'
grep -c relaykit go.sum
ls go.work 2>/dev/null ; ls -d vendor 2>/dev/null

# 文档点名与自述定位
sed -n '1,14p;38,42p;227,262p' relaykit/README.md
grep -rn relaykit docs/ scripts/ | head

# 使用者盘点（脚本统计 import 语句，见第三节）
# 测试覆盖点
grep -rn -w LookupTextConverter relaykit/relayconvert/text_converter_registry_test.go
grep -rn -w ChannelTypeOf relaykit/relayconvert/convmeta/meta_test.go
grep -rn -w ConvertResponseByID relaykit/relayconvert/response_registry_test.go
grep -rn -w OpenAIChatRequestToGeminiGenerateContent relaykit/relayconvert/safety_settings_test.go
grep -rn -w OpenAIResponsesRequestToClaudeMessages relaykit/relayconvert/claude_default_max_tokens_test.go
grep -rn -w OpenAIResponsesRequestToGeminiChat relaykit/relayconvert/safety_settings_test.go

# 孤儿类型与家族使用
grep -rn --include=*.go '\bStringValue\b' . | grep -v 'relaykit/dto/values.go'
grep -rn --include=*.go 'dto\.IntValue\|dto\.BoolValue' . | wc -l
grep -rn --include=*.go 'FileMeta{' . | head
```

自查：0 个成对美元符号；星号仅出现在反引号代码片段内（函数签名指针类型），不构成 Markdown 斜体；各表列数一致。
越界自述：未删除/修改任何代码或配置，未提交、未推送、未发版、未联网、未装工具；唯一写入为本文件。
