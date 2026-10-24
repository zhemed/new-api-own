# 孤儿类型 + locale 键 + relaykit 收敛：独立验证证据（verifier）

- 任务：共享任务板 `task-61`（Trellis 任务 `10-24-orphan-types-and-relaykit-policy` 的独立验证）
- 验证者：verifier（只读验证；未改仓库文件；未用 `-mod=mod`，golangci 用 `-mod=readonly`；`i18n:sync` 只在 /tmp 副本运行）
- 验证时间：2026-10-24（UTC 07:52 起）
- 基线：`HEAD = 47b2321`；最终 `git status --porcelain` 行数 = **18**（17 已跟踪修改 + 1 任务目录）；
  `git diff` = **17 文件、32 增、149 删**；diff 指纹 `70d6b2c6dd4fcb4efc63f6dcb04a23a12b223460e3ddd45fdfe4e9081a9de2f4`
- `go.mod` / `go.sum` 无改动：blob 均等于 HEAD（`b0642f16…` / `357374c3…`）
- 工具环境：Go 1.26.6、`~/go/bin/deadcode`（RTA）、Bun + Node（web 目录）

## 结论

**接受（Accept），另附 1 处建议修正的文档表述。** 5 个孤儿导出类型在全仓（Go 含测试与构建标签、非 Go）零引用且无序列化/GORM/反射依赖；
「deadcode 仍 53、unused 仍 1」的口径经独立复核成立并给出机制解释。locale 两个裸键确实无引用，14 处删除、零重排、
键数 5330 到 5328 与引用数 4197 不变均复现，`i18n:sync` 在 /tmp 副本运行后混淆键完好且文件逐字节不变。
relaykit 收敛的 16 个导出函数与 `dto.StringValue` 类型零残留（含注册表内部面核对），测试改写只换调用目标、断言与输入等价；
两套 deadcode 53 到 37 与 109 到 96 均按符号名比对为「无新增不可达」。门禁与边界全绿。
**建议修正**：README「公开 API 政策」中 4 个保留项那行的理由写「主仓库与模块测试依赖」，实测主仓库对这 4 项零调用、
且 `NewResponseStreamStateByID` 全仓零引用（仅定义）；建议改写为「模块测试依赖（3 项）；`NewResponseStreamStateByID` 当前无调用，保留为公开面内省入口」。
属文档表述问题，不影响代码正确性与门禁。

## 一、孤儿类型（A）

### 1.1 删除与引用核对

5 个类型：`pkg/cachex.StringCodec`、`pkg/ionet.ContainerLogs`、`pkg/ionet.LogEntry`、
`pkg/ionet.LocationAvailability`、`pkg/ionet.HardwareAvailability`（2 文件、35 行删除）。

| 核对维度 | 结果 |
|---|---|
| 工作区 Go 引用（含 `_test.go`、含构建标签文件的全量扫描） | **0 / 5** |
| HEAD 树 Go 引用（确认删除前即孤儿） | 仅定义处与「死类型内部互引」：`ContainerLogs` 的字段引用 `LogEntry`、`LocationAvailability` 的字段引用 `HardwareAvailability`，两者同批删除 |
| 工作区非 Go 引用 | 仅本任务目录的 PRD/证据（任务记录，非项目文档）；README / MAINTENANCE / AGENTS.md / CLAUDE.md / `.trellis/spec` / scripts / .github / web 均为 0 |
| HEAD 树非 Go 引用 | 仅 `.trellis/tasks/archive/**` 的历史证据与 journal（历史记录，不构成引用） |
| 序列化依赖 | 无 `json.Marshal` / `Unmarshal` 目标；类型上的 JSON 标签无消费者；`StringValue` 之外无自定义 JSON 方法 |
| GORM 依赖 | `AutoMigrate` 调用点（model/main.go:314/363/374）均不涉及这 5 个类型 |
| 反射 / linkname | 这 5 个类型所在文件与相关文件中无 `reflect.TypeOf/ValueOf`、无 `%T` 目标、无 `//go:linkname` |

### 1.2 「deadcode 仍 53、unused 仍 1」的独立复核与解释

| 结论 | 证据 |
|---|---|
| 5 个类型在两套 deadcode 输出中**从未出现**（删除前后都不在） | 对 4 份 deadcode 输出 grep 这 5 个名字均为 0 命中 |
| 删除类型不影响 deadcode 计数 | deadcode 报告格式只含 `unreachable func:`（函数级 RTA）；这 5 个类型在 HEAD 上**没有任何方法**（`StringCodec` 的方法已在上一任务删除），因此无法进入函数名单 |
| 删除类型不影响 unused 计数 | `unused` 只报未使用的**未导出**标识符：删除前后均恰 1 条 `model/main.go:27 var logKeyCol is unused`；这 5 个导出类型在 HEAD 就存在时 unused 也一直是 1，说明导出类型不进该名单 |
| 数字变化来自 C 而非 A | 根模块 deadcode 由 53 降到 37，减少的 16 条全部是 relaykit 被删函数（见第三节），与 5 个类型无关 |

## 二、locale 键（B）

### 2.1 删除与无引用核对

| 核对项 | 结果 |
|---|---|
| 改动形态 | 7 个 locale 各删 2 行（`"Peak"`、`"Off-peak"`），**共 14 deletions / 0 insertions**；numstat 逐文件为 0 增 2 删 |
| 无重排 | diff 只有删除行，无任何新增/移动行；删除位置保持原有字母序上下文 |
| 键数 | 每个 locale 从 **5330 降到 5328**（逐语言独立计数），7 个语言一致 |
| 源码字面量引用 | `web/src`、`web/scripts` 中无 `'Peak'` / `"Peak"` / `'Off-peak'` / `"Off-peak"` 键引用（`Peak hours` / `Off-peak hours` 不受影响） |
| `STATIC_I18N_KEYS` | 未登记这两个裸键：文件内相关项只有 `Peak hours`（:553）与 `Off-peak hours`（:554），裸 `Peak` / `Off-peak` 从未登记 |
| 测试断言 | 前端测试无对这两个键的断言（pricing 测试全部通过，见 4.3） |
| `i18n:check` | rc=0，且引用键总数为 **4197（4036 字面 + 467 静态）——与改动前一致**，说明没有任何引用指向被删键 |

### 2.2 `i18n:sync` 与混淆键（在 /tmp 副本运行，真实工作区零写入）

| 核对项 | 结果 |
|---|---|
| 运行方式 | 复制 `web/package.json`、`web/scripts`、`web/src` 到 `/tmp/t61-web`，`node_modules` 软链到工作区；`bun run i18n:sync` → rc=0 |
| 混淆键 | 7 个 locale 同步后均保留 `footer.new\u0061pi.projectAttributionSuffix`（各 1 处） |
| 是否重排 / 重写 | 7 个 locale 同步前后 sha256 **逐个相同**（幂等，无重排、无回填被删键） |
| 与真实工作区一致性 | /tmp 副本同步后的 7 个文件与真实工作区对应文件 `cmp` 全部一致（既证明真实文件是 sync 的稳定态，也证明未额外改动） |

## 三、relaykit 收敛（C）

### 3.1 删除清单与引用核对

删除 16 个导出函数/方法 + 1 个孤儿类型 `dto.StringValue`：

| 文件 | 删除项 | 数量 |
|---|---|---|
| relayconvert/request_compat.go | ClaudeMessagesRequestToOpenAIChat、GeminiGenerateContentRequestToOpenAIChat、OpenAIChatRequestToGeminiGenerateContent、ChatCompletionsRequestToResponsesRequest、ResponsesRequestToChatCompletionsRequest、OpenAIResponsesRequestToClaudeMessages、OpenAIResponsesRequestToGeminiChat | 7 |
| relayconvert/response_compat.go | StopReasonClaudeToOpenAI、ResponsesStatusFromChatFinishReason、ResponsesFinishReasonFromStatus、ExtractReasoningTextFromResponses | 4 |
| relayconvert/kitutil/json.go | StringToByteSlice | 1 |
| types/request_meta.go | NewFileMeta、NewImageFileMeta | 2 |
| dto/values.go | StringValue.UnmarshalJSON、StringValue.MarshalJSON | 2 |
| dto/values.go | 类型 `StringValue` 本体 | 1 |

| 核对维度 | 结果 |
|---|---|
| relaykit 之外的引用 | **0 处真实引用**。`git grep HEAD` 命中里：生产代码仅同名不同物（根模块自己的 `common.StringToByteSlice`、AWS `eventstream.StringValue`、relay/common 的 `trimStringValue` 等辅助函数）；其余是 `.trellis/tasks/archive/**` 历史证据与 `.agents/skills/**` 示例文档 |
| 当前工作区残留 | 仅 relaykit 内部同名实现与改写后的测试（见 3.2）；被删的公开门面在两个 compat 文件中已无残留 |
| 注册表依赖 | 注册表**直接调用 internal 实现**，不经过被删门面：request_registry.go:412/425/451/464/472/485 分别以 `oaichat.` / `claudemessages.` / `geminichat.` / `oairesponses.` 前缀调用；text_converter_registry.go:224/240 的 Convert 回调指向模块内 `convertOpenAIResponsesRequestTo*` |
| 序列化 / 反射依赖 | 无 `reflect.`、无 `//go:linkname`、无 JSON 标签消费者；`dto.StringValue` 未被任何 DTO 字段使用（HEAD 中除 values.go 自身外零引用） |
| `StringToByteSlice` | HEAD 中唯一使用者是上一任务已删的 `kitutil.UnmarshalJsonStr`；删除后无残留 |

### 3.2 测试改写未放宽断言（2 文件 3 处）

diff 只含「新增 internal 导入 + 3 处调用目标替换」，用例名、输入构造、断言全部未动：

| 文件 | 处 | 改写 | 等价性 |
|---|---|---|---|
| claude_default_max_tokens_test.go | 1 | `OpenAIResponsesRequestToClaudeMessages(...)` 改为 `oairesponses.OpenAIResponsesRequestToClaudeMessages(...)` | HEAD 中的门面函数体就是 `return oairesponses.OpenAIResponsesRequestToClaudeMessages(c, info, req)`，参数与顺序逐字相同 |
| safety_settings_test.go | 2 | `OpenAIChatRequestToGeminiGenerateContent(...)` 改为 `oaichat.…`；`OpenAIResponsesRequestToGeminiChat(...)` 改为 `oairesponses.…` | 对应门面函数体分别是 `return oaichat.…(c, textRequest, info)` 与 `return oairesponses.…(c, req, info)`，签名与实参一致 |

核对结论：3 处改写为「直接调用原先被门面透传的同一 internal 函数」，**断言强度不变**；`GOWORK=off go test ./... -count=1` 全绿。

### 3.3 README 与真实结构一致性

| README 声明 | 独立核对 |
|---|---|
| 不计划独立发布；没有 `relaykit/v*` tag | `git tag -l` 无任何 relaykit 相关 tag |
| 主仓库用 `replace` 指向本地目录 | 根 go.mod:167 `require github.com/QuantumNous/new-api/relaykit v0.0.0` + :170 `replace … => ./relaykit` |
| 没有独立发布流水线 | 工作流中 relaykit 仅出现在注释里：lint.yml:46（gofmt 覆盖说明）、release.yml:145（测试用 `GOWORK=off` 说明）；release.yml 只做主仓库 release，无模块发布链路 |
| 现在不能用 go get 安装 | 与「无 tag + 仅本地 replace」一致（无远端版本可解析） |
| 4 个保留项并入公开面清单 | README:240 的表格行明确列出 `ConvertResponseByID`、`NewResponseStreamStateByID`、`LookupTextConverter`、`ChannelTypeOf` |
| 「最近一次收敛」段的数量 | 16 个函数/方法（8 真死门面 + 3 自测门面 + kitutil 1 + types 2 + dto 2）与孤儿类型 1 个，与实际删除清单逐项吻合 |

**建议修正（唯一发现）**：README:240 该行理由写的「主仓库与模块测试依赖」不准确——独立检索显示
主仓库对这 4 项**零调用**（无 `relayconvert.<名字>` 调用点），`NewResponseStreamStateByID` 全仓仅定义、零引用（含模块测试）；
`ConvertResponseByID`、`LookupTextConverter`、`ChannelTypeOf` 仅被模块测试调用。建议改为
「模块测试依赖（3 项）；`NewResponseStreamStateByID` 当前无调用，保留为公开面内省入口」。

### 3.4 两套 deadcode（按符号名比对）

| 运行 | 起点 | 终点 | 集合关系 | 新增不可达 | 消失 |
|---|---|---|---|---|---|
| 根模块（`deadcode -test ./...`） | 53 | **37** | after 是 before 子集 | **0** | 16 |
| relaykit 模块（`cd relaykit && GOWORK=off deadcode -test ./...`） | 109 | **96** | after 是 before 子集 | **0** | 13 |

- 根模块消失的 16 条 = 本次删除的全部 16 个函数/方法（含 `StringValue.MarshalJSON/UnmarshalJSON` 两个方法），逐名吻合；
- relaykit 模块消失 13 条 = 16 减去 3 个「仅模块自测引用」的门面（`OpenAIChatRequestToGeminiGenerateContent`、
  `OpenAIResponsesRequestToClaudeMessages`、`OpenAIResponsesRequestToGeminiChat`）。原因：relaykit 单独运行时 `-test` 会把模块测试
  作为可达根，这 3 个函数在起点 109 中根本不在名单里；根模块运行时依赖模块的测试文件不参与分析，故它们出现在 53 中。两个数字自洽；
- 两套均确认**无新增不可达**（不是只看行数，而是按 `文件:符号名` 集合做差）。

## 四、门禁（D）

| 门禁 | 命令 | 结果 |
|---|---|---|
| 根构建 | `GOPROXY=off go build ./...` | rc=0 |
| 根静态检查 | `GOPROXY=off go vet ./...` | rc=0 |
| 根全仓测试 | `GOPROXY=off go test ./... -count=1` | rc=0，无 FAIL |
| 格式 | `gofmt -l .` | 输出为空 |
| unused | `GOPROXY=off GOFLAGS=-mod=readonly golangci-lint run --default=none --enable=unused ./...` | **恰 1**：`model/main.go:27 var logKeyCol is unused` |
| relaykit | `cd relaykit && GOWORK=off GOPROXY=off go build ./... && GOWORK=off go test ./... -count=1` | 均 rc=0 |
| 前端 typecheck | `cd web && bun run typecheck` | rc=0 |
| 前端 i18n:check | `bun run i18n:check` | rc=0，4197 引用键在 7 locale 全存在 |
| 前端单测 | `bun test src/features/pricing/` | 15 pass / 0 fail（3 文件） |

## 五、边界

| 核对项 | 结果 |
|---|---|
| `git status --porcelain` | 18 行 = 17 M + 1 任务目录；17 个改动文件与任务预期完全一致（pkg 2 + relaykit 8 + locale 7） |
| 未跟踪 | `-uall` 下仅任务目录内 6 个文件（prd、task.json、4 份实现者证据）；无临时/演示残留 |
| 依赖文件 | `go.mod` / `go.sum` blob 均等于 HEAD，无改动 |
| 只读纪律 | `i18n:sync` 只在 /tmp 副本运行；真实工作区未做任何写入 |

## 六、不确定项

1. README 4 个保留项的理由表述不准确（见 3.3 建议修正）；不影响代码与门禁。
2. 保留 4 项是否属于「真实需要的公开面」属产品判断：其中 `NewResponseStreamStateByID` 当前零引用，若后续确认无用可再评估；
   本轮按「并入公开面清单」的要求核对，结论是已并入。
3. 未做真实 `go get` 网络验证（红线禁止联网）；「不可 go get」的结论基于「无 tag + 仅本地 replace + 无发布流水线」的结构事实。

## 七、复跑命令清单

```bash
cd /root/new-api-own
# A. 孤儿类型
grep -rnE 'StringCodec|ContainerLogs|LogEntry|LocationAvailability|HardwareAvailability' --include='*.go' .
GOPROXY=off GOFLAGS=-mod=readonly ~/go/bin/deadcode -test ./... | wc -l        # 37
GOPROXY=off GOFLAGS=-mod=readonly golangci-lint run --default=none --enable=unused ./...   # 恰 1
# B. locale
git diff --numstat -- web/src/i18n/locales/                                   # 每个 0 增 2 删
cd web && bun run i18n:check && bun test src/features/pricing/
#（i18n:sync 请在 /tmp 副本运行，避免写真实工作区）
# C. relaykit
cd relaykit && GOWORK=off GOPROXY=off go build ./... && GOWORK=off go test ./... -count=1
GOWORK=off GOPROXY=off GOFLAGS=-mod=readonly ~/go/bin/deadcode -test ./... | wc -l          # 96
# D. 边界
git status --porcelain go.mod go.sum                                          # 应为空
```
