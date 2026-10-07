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

## 执行结果（2026-10-06，用户授权"帮我保存并重跑测试"）

### 保存

- 先确认安全边界：管理接口**不返回渠道密钥**（`key` 为空），故采用**最小补丁**：`{"id":1,"header_override":"{…}"}`
  （后端绑定的 `PatchChannel` 是补丁式，只发要改的字段最安全）；
- 过程记录：前两次尝试（整对象回写 / 前端全字段形状）均被拒 `Invalid parameters` —— 原因是
  `controller/channel.go:961` 对**包含 `status` 的更新请求**一律拒绝（状态改动必须走
  `POST /api/channel/:id/status`）。此为**上游既有行为**（`v0.0.3` 同一逻辑），不影响面板；
- 最小补丁提交成功：`{"success":true}`；
- **密钥完整性**：保存前后 `channels.key` 长度 67、sha256 前 12 位 `dd7dd290a541` **完全一致**；
  其它字段（name/models/group/status/base_url/type）未被破坏。

### 测试结果

```
GET /api/channel/test/1?model=deepseek-flash
→ {"success":true,"message":"","time":1.595}
```

上游 opencode **接受了** `x-opencode-session: opencode-go-fallback`（通道默认值兜底），400 MissingSessionID 消失。

### 更正一条 Lead 的错误结论（留痕）

排查中我一度推断"面板保存渠道一直失败"——**错误**。证据反证：
`transformFormDataToUpdatePayload()`（编辑保存专用构造器）**不含 `status`**，
`use-channel-mutate-form.ts` 还会在密钥未改动时删除 `key`；最小补丁（无 `status`）能成功即证明面板路径正常。
真实情况是：该条 header override **此前从未保存过**。教训：涉及"面板是否可用"的判断必须在读到
**实际构造 payload 的代码**之后再下结论，不能从一次手工请求被拒反推面板行为。
