# RelayKit

RelayKit 是本项目拆分出的独立 Go 模块，提供常用大模型文本协议的 DTO、请求转换、响应转换和流式事件转换。

它只负责协议层的数据建模与语义转换，不包含 HTTP 服务、上游请求发送、渠道调度、鉴权、计费或数据库逻辑。技术上它可以脱离 new-api 主模块独立构建；但**当前定位是主仓库的内部拆分模块，不计划独立发布**（没有 `relaykit/v*` tag、没有独立发布流水线，主仓库通过 `replace` 指向本地目录），公开面与兼容性口径见「公开 API 政策」。

## 能力

- 在 OpenAI Chat Completions、OpenAI Responses、Anthropic Messages 和 Gemini `generateContent` 之间转换
- 同时支持请求、非流式响应和增量流式响应
- 自动根据 DTO 类型识别源协议，并选择内置的直接或多跳转换路径
- 返回转换器 ID、质量等级、实际转换步骤和统一 usage，方便审计与调试
- 支持常用的文本、多模态内容、工具调用、推理内容和 usage 映射
- 作为独立 Go module 构建，不依赖 new-api 主模块、Gin、数据库或全局设置

## 支持矩阵

以下四种文本协议支持任意两种格式之间的转换：

| 源格式 \ 目标格式 | OpenAI Chat | OpenAI Responses | Claude Messages | Gemini |
|---|---:|---:|---:|---:|
| OpenAI Chat | — | Good | Fair | Fair |
| OpenAI Responses | Good | — | Fair | Fair |
| Claude Messages | Fair | Fair | — | Discouraged |
| Gemini | Fair | Fair | Discouraged | — |

质量等级表示协议之间的语义匹配程度：

- `Good`：两种协议的核心结构较接近
- `Fair`：主要能力可转换，但部分协议特性可能需要适配或无法完整保留
- `Discouraged`：目前需要经过中间协议转换，语义损失风险更高

请求、非流式响应和流式响应均覆盖上述矩阵。实际采用的路径可从转换结果的 `Steps` 和 `Quality` 字段中读取。

## 使用与安装

RelayKit 要求 Go 1.25.1 或更高版本。

**当前状态：未独立发布。** RelayKit 作为 new-api 的内部拆分模块随主仓库一起开发：没有 `relaykit/v*` tag，也没有独立发布流水线，主仓库的 `go.mod` 通过 `replace ... => ./relaykit` 引用本地目录。因此**现在不能**用 `go get github.com/QuantumNous/new-api/relaykit@latest` 安装（无可用版本 tag）；在仓库内按下面的方式构建与测试：

```bash
cd relaykit
GOWORK=off go build ./...
GOWORK=off go test ./...
```

主要包：

| 包 | 用途 |
|---|---|
| `relaykit/dto` | 各协议的请求、响应、流式事件和 usage DTO |
| `relaykit/types` | 协议格式、错误、文件来源及共享类型 |
| `relaykit/relayconvert` | 请求、响应和流式转换入口 |
| `relaykit/relayconvert/convmeta` | 与宿主实现解耦的转换上下文和选项 |
| `relaykit/reasonmap` | 不同协议之间的结束原因映射 |

## 快速开始

下面将 OpenAI Chat Completions 请求转换为 Claude Messages 请求：

```go
package main

import (
	"context"
	"fmt"

	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/relayconvert"
	"github.com/QuantumNous/new-api/relaykit/relayconvert/convmeta"
	"github.com/QuantumNous/new-api/relaykit/types"
)

func main() {
	maxTokens := uint(1024)
	request := &dto.GeneralOpenAIRequest{
		Model: "claude-sonnet-4-5",
		Messages: []dto.Message{
			{Role: "user", Content: "Hello!"},
		},
		MaxTokens: &maxTokens,
	}

	meta := &convmeta.Values{
		OriginModelName:     "client-model",
		UpstreamModelName:   request.Model,
		ChannelMetaAttached: true,
	}

	result, err := relayconvert.ConvertRequest(
		context.Background(),
		meta,
		types.RelayFormatClaude,
		request,
	)
	if err != nil {
		panic(err)
	}

	claudeRequest, ok := result.Value.(*dto.ClaudeRequest)
	if !ok {
		panic(fmt.Sprintf("unexpected result type %T", result.Value))
	}

	fmt.Printf("model=%s messages=%d\n", claudeRequest.Model, len(claudeRequest.Messages))
}
```

`ConvertRequest` 根据请求的具体 DTO 类型推断源格式。传入原始 JSON、`map[string]any` 或不受支持的 DTO 会返回错误。

### 非流式响应

响应转换使用相同的目标格式模型：

```go
result, err := relayconvert.ConvertResponse(
	ctx,
	meta,
	types.RelayFormatOpenAI,
	claudeResponse,
)
if err != nil {
	return err
}

openAIResponse := result.Value.(*dto.OpenAITextResponse)
usage := result.Usage
```

支持的响应 DTO：

| 格式 | 非流式响应 | 流式事件 |
|---|---|---|
| OpenAI Chat | `dto.OpenAITextResponse` | `dto.ChatCompletionsStreamResponse` |
| OpenAI Responses | `dto.OpenAIResponsesResponse` | `dto.ResponsesStreamResponse` |
| Claude Messages | `dto.ClaudeResponse` | `dto.ClaudeResponse` |
| Gemini | `dto.GeminiChatResponse` | `dto.GeminiChatResponse` |

### 流式响应

流式转换可能需要跨事件保存工具调用、usage 和结束状态。每条上游流应创建独立的 `ResponseStreamState`，并在上游结束后调用 `FinalizeStreamResponse`：

```go
state, err := relayconvert.NewResponseStreamState(
	types.RelayFormatOpenAI,
	types.RelayFormatOpenAIResponses,
	relayconvert.ResponseStreamOptions{
		ID:           "resp_123",
		Model:        "gpt-4.1",
		IncludeUsage: true,
	},
)
if err != nil {
	return err
}

for _, chunk := range upstreamChunks {
	results, err := relayconvert.ConvertStreamResponseChunk(ctx, meta, state, chunk)
	if err != nil {
		return err
	}
	for _, result := range results {
		emit(result.Value)
	}
}

finalResults, err := relayconvert.FinalizeStreamResponse(ctx, meta, state)
if err != nil {
	return err
}
for _, result := range finalResults {
	emit(result.Value)
}

usage := state.Usage()
```

RelayKit 不负责 SSE 的读取和写入。宿主需要将每个 SSE 事件解析为对应 DTO，并将转换结果重新编码后发送给下游。不要省略 `FinalizeStreamResponse`，部分转换器会在该阶段补发终止事件或最终 usage。

## 转换上下文

大多数基础转换可以传入 `nil` 作为 `convmeta.Meta`。需要模型映射、推理适配、安全设置或流式状态时，应使用 `convmeta.Values`，或在宿主中实现 `convmeta.Meta`。

常用选项通过 `convmeta.Options` 按请求传入：

```go
meta := &convmeta.Values{
	Options: &convmeta.Options{
		Claude: convmeta.ClaudeOptions{
			DefaultMaxTokens: func(model string) int {
				return 4096
			},
		},
		Gemini: convmeta.GeminiOptions{
			ThinkingAdapterEnabled: true,
		},
	},
}
```

需要注意：

- OpenAI Chat 或 OpenAI Responses 转 Claude 时，Claude 请求必须具有 `max_tokens`。源请求未提供时，需要配置 `Claude.DefaultMaxTokens`，否则转换会返回错误。
- RelayKit 不负责选择渠道或映射模型名。调用转换前，应将请求中的 `Model` 设置为目标上游使用的模型名。
- 自定义 `convmeta.Meta` 的指针实现必须保证所有方法对 nil receiver 安全，完整约束见 `convmeta.Meta` 的接口注释。

## 多模态内容

某些跨协议的图片转换需要下载 URL 内容或解析 data URL。宿主应在启动时配置媒体解析器：

```go
relayconvert.SetMediaResolver(relayconvert.MediaResolver{
	GetBase64Data:        getBase64Data,
	DecodeBase64FileData: decodeBase64FileData,
})
```

两个回调的签名由 `relayconvert.MediaResolver` 定义。需要媒体解析而未配置对应回调时，转换会明确返回错误；RelayKit 本身不会发起网络请求。

## 转换结果

请求转换返回 `relayconvert.RequestResult`，响应转换返回 `relayconvert.ResponseResult`。除 `Value` 外，建议关注：

- `From` / `To`：源格式和目标格式
- `Converter`：所选转换器 ID
- `Quality`：转换质量等级
- `Steps`：直接转换或多跳转换的实际路径
- `Usage`：响应转换后的统一 token usage
- `Stream`：结果是否来自流式转换

如果需要固定转换路径，可使用 `ConvertRequestVia`；如果需要按转换器 ID 执行，可使用 `ConvertRequestByID`、`ConvertResponseByID` 和 `NewResponseStreamStateByID`。

## 公开 API 政策

**模块定位（2026-10-24 起）**：RelayKit 是 new-api 主仓库的**内部拆分模块**，不计划独立发布（无 `relaykit/v*` tag、无独立发布流水线、主仓库用 `replace` 本地引用）。下表的「公开面」是**主仓库自用契约**，不是对外兼容承诺。

| 分类 | 范围 | 变更要求 |
|---|---|---|
| 公开面 | `relaykit/dto`、`relaykit/types`、`relaykit/relayconvert` 的注册表入口（`ConvertRequest` / `ConvertRequestVia` / `ConvertRequestByID` / `ConvertResponse` / `ConvertStreamResponseChunk` / `FinalizeStreamResponse` / `NewResponseStreamState` / `SetMediaResolver` 等）、`relaykit/relayconvert/convmeta`、`relaykit/reasonmap` | 删除/改名/改签名必须同步主仓库全部调用点，并更新本清单与主仓库发布说明 |
| 注册表与内省入口（公开面，保留） | `ConvertResponseByID`、`NewResponseStreamStateByID`、`LookupTextConverter`、`ChannelTypeOf` | 同上；这 4 个是转换器注册表与上下文内省 API。依赖实况：`ConvertResponseByID` / `LookupTextConverter` / `ChannelTypeOf` 由**模块测试**使用（主仓库零调用）；`NewResponseStreamStateByID` 当前无调用，按公开面内省入口保留 |
| 内部面（不承诺兼容） | `relayconvert/internal/**`、`relayconvert/kitutil`、`request_compat.go` / `response_compat.go` 的历史兼容门面 | 可自由删改，但必须保持主仓库可编译、两套门禁全绿 |

变更流程：

1. 改公开面 → 同步主仓库调用点 → 跑主仓库 `go build ./...`、`go test ./... -count=1`，以及 `cd relaykit && GOWORK=off go build ./... && GOWORK=off go test ./...`；
2. 在本节「公开面」清单与主仓库发布说明（`MAINTENANCE.md` 的发布记录）中记录删改；
3. 内部面清理不需要发布说明，但需保持本节与「JSON 编码约定（模块例外）」同步，避免文档腐化。

最近一次收敛（2026-10-24）：删除 16 个无人使用的导出函数（8 个真死兼容门面、3 个仅模块自测引用的兼容门面、`kitutil.StringToByteSlice`、`types.NewFileMeta` / `NewImageFileMeta`、`dto.StringValue` 的 2 个 JSON 方法）与孤儿类型 `dto.StringValue` 本体；3 个仅自测引用的门面删除后，测试改为直接调用 `internal` 实现；保留上表 4 个注册表/内省入口。

## 开发

RelayKit 必须始终保持独立可构建。修改模块后，在 `relaykit` 目录运行：

```bash
GOWORK=off go test ./...
GOWORK=off go build ./...
```

转换矩阵由 golden tests 覆盖。确认协议输出变化是预期行为后，可更新快照：

```bash
GOWORK=off go test ./relayconvert -run TestGolden -update
```

### JSON 编码约定（模块例外）

`relaykit` 是独立 Go 模块，按项目 `AGENTS.md` 的「relaykit module independence」约束**不得**导入根模块的
`common` 包；因此根模块的 JSON 包装约定（`common.Marshal` / `common.Unmarshal` /
`common.UnmarshalJsonStr` / `common.DecodeJson` / `common.GetJsonType`）在本模块内不适用。模块内的 JSON 操作约定为：

- 普通序列化/反序列化优先使用 `relayconvert/kitutil` 的本地包装（`kitutil.Marshal` / `kitutil.Unmarshal` /
  `kitutil.GetJsonType` / `kitutil.JsonRawMessageToString`）；
- 需要流式解码、或实现类型自身的 `MarshalJSON` / `UnmarshalJSON` 方法时，允许直接使用标准库 `encoding/json`。

这是对 `AGENTS.md`「JSON package」约定的**模块内例外**，不是遗漏：`relaykit` 无法引用 `common`，而为了统一包装
再在模块里维护一套无人使用的同名辅助函数只会留下死代码（`kitutil.UnmarshalJsonStr` / `kitutil.DecodeJson`
即因此于 2026-10-24 删除）。根模块业务代码仍必须走 `common.*` 包装。

## 版本与兼容性

RelayKit 当前使用 `v0.x` 版本（版本号随主仓库演进，当前没有独立 tag）。公开 API 和 DTO 仍可能随主仓库小版本调整；由于当前不计划独立发布，RelayKit 不承诺独立的模块级兼容性，升级或改动前请检查主仓库发布说明和实际序列化结果。协议之间并非完全同构，建议对业务实际使用的工具调用、多模态、推理和流式场景增加端到端测试。

## 许可证

RelayKit 是 new-api 项目的一部分，遵循项目根目录中的 [GNU Affero General Public License v3.0](../LICENSE)。
