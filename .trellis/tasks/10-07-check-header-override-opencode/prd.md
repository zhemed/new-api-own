# 排查：渠道测试 400（上游要求 x-opencode-session）与请求头覆盖

## 用户提供

- 截图 1：渠道 `deepseek-flash` 测试失败，上游返回 400
  `{"type":"error","error":{"type":"MissingSessionID","message":"Request is missing x-opencode-session …"}}`
  —— **错误来自上游 opencode，不是网关**。
- 截图 2：渠道编辑 → 高级设置 → 覆盖规则 → 请求头覆盖，内容
  `{"x-opencode-session": "{client_header:x-opencode-session|opencode-go-fallback}"}`
- 用户判断：「好像必须要设置一下这东西，要不然无法使用，你去看一下」

## 排查结论

### 1) 占位符语义（代码 + 既有测试为准）

`relay/channel/api_request.go:225-320`：

- `{client_header:NAME}`：**渠道测试时被跳过**（测试没有客户端请求可读）；
- `{client_header:NAME|DEFAULT}`：**带默认值，测试也照常解析**并发出 DEFAULT。

既有权威用例：`TestProcessHeaderOverride_ChannelTestSkipsClientHeaderPlaceholder`（无默认值→跳过）、
`TestProcessHeaderOverride_NonTestKeepsClientHeaderPlaceholder`（正式请求→保留）、
`TestProcessHeaderOverride_ChannelTestSkipsPassthroughRules`（测试跳过 `*` 透传）。

→ 用户写法**恰是正确形态**：正式请求透传客户端 session，测试用默认值兜底，两条路径都不会缺头。

### 2) 真正的症结：配置**没有保存**

用管理接口读用户实例的渠道 1：

```
{ "id": 1, "name": "deepseek-flash", "base_url": "https://opencode.ai/zen/go", "type": 60,
  "header_override": "（空）",
  "setting": {"force_format":false,…,"pass_through_body_enabled":false,…} }   ← setting 里也没有
```

前端保存路径：`web/src/features/channels/lib/channel-form.ts:870` → `payload.header_override`
（渠道**顶层字段**，不是 `setting` 内）。后端消费路径完整：
`model/channel.go:51`（字段）→ `GetHeaderOverride()`（JSON 解析）→
`relay/channel/api_request.go:366/398/422`（三处请求构造注入）。

→ **结论**：写法正确、链路完整，只差"保存"这一步；测试失败发生在保存之前。

### 3) 附带发现（可改进项，未改）

渠道测试**永远无法验证"无默认值"的透传占位符**（设计如此：测试无客户端请求），
因此面板上"测试失败"可能被误读为"渠道不可用"。可在测试结果里提示"该头由客户端提供，测试未携带"，
但这属于产品改进，需用户点头再做。

## 用户决定

（待定：是否由 Lead 代保存并重跑测试）
