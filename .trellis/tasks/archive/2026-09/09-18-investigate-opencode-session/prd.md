# 调查 x-opencode-session / 实例专属兜底值 渠道配置

## 结论（TL;DR）

这不是可疑改动，而是 **2026-09-12 有意配置**、并已写进 `MAINTENANCE.md` 的机制：线上唯一渠道（`上游渠道` →
`https://opencode.ai/zen/go`，即 OpenCode Go / Zen）上有一条 `param_override`，作用是**透传客户端自带的会话头，
客户端没带时兜底填 `实例专属兜底值`**。上游因此看到我们的出口 IP + 一个非 OpenCode 客户端自带的会话标识。

## 证据（可复现）

### 1. 线上配置原文（取自非公开运维备份，只读）

来源：非公开运维备份 → `channels` 表。

```
id=1  name=上游渠道  type=60(ChannelTypeNewAPI)  status=1  base_url=https://opencode.ai/zen/go
header_override = (空)
param_override  = {"operations":[
  {"mode":"pass_headers","value":["x-opencode-session","Session-Id","X-Session-Id"],"keep_origin":true},
  {"mode":"set_header","path":"x-opencode-session","value":"实例专属兜底值","keep_origin":true}
]}
```

- 全库扫描该兜底值字符串：**只出现 1 次**，就在 `channels.param_override`；没有出现在任何其它表/字段。
- 该库只有 1 个渠道（启用 1 个）。
- 顺序正确（`pass_headers` 在前、`set_header` 在后且 `keep_origin:true`），符合 `MAINTENANCE.md:253-255` 的警告。

### 2. 时间线

| 时间 | 事件 |
|---|---|
| 2026-09-05 | 上游 OpenCode Go 开始要求会话头，缺失返回 `400 MissingSessionID` |
| 2026-09-12 00:41 备份 | `param_override` 仍为**空** |
| 2026-09-12 10:41 备份（planB-working） | 已带上 `实例专属兜底值` 兜底配置 → **配置在这一夜之间加上** |
| 2026-09-12 11:05 | 提交 `25c9aa1` `feat(channel): OpenCode Go 会话头透传与占位符兜底`：新增 `{client_header:NAME|DEFAULT}` 占位符、面板预设「OpenCode Go Session Header」、7 语言词条、`MAINTENANCE.md` 新章节 |
| 2026-09-14 / 09-16 备份 | 配置未变 |

### 3. 仓库侧机制

- `relay/channel/api_request.go:177-209`：`{client_header:NAME}` 透传、`{client_header:NAME|DEFAULT}` 缺失时兜底。
- 前端预设：`web/src/features/channels/components/dialogs/param-override-editor-dialog.tsx:355+`（示例兜底值是
  `opencode-go-fallback`；线上实际用 `实例专属兜底值`）。
- 文档：`MAINTENANCE.md:212-269`（含上游实测结论与探针诊断手法）。

## 上游能看到什么

- 我们的出口 IP；
- `x-opencode-session: 实例专属兜底值`（**仅当客户端未自带**该头；自带则原样透传）；
- `User-Agent: Go-http-client/1.1`、`Authorization`、`Content-Type`（`MAINTENANCE.md:267` 探针实测）。

因此严格说不是"完整伪装成 OpenCode 客户端"——UA 仍是 Go 的；实际做的是"补上上游强制要求的会话路由标识"。

## 风险与代价（诚实写）

1. **共用一个缓存桶**：所有不自带会话头的客户端都用 `实例专属兜底值`，对话级 prompt-cache 亲和丢失；反过来，不同对话/用户可能命中同一份上游缓存条目（成本更低，但存在跨对话缓存共享）。`MAINTENANCE.md:240` 已写明这个代价。
2. **`-01` 的编号没用起来**：命名暗示可按客户端/用户分桶，目前只有单一兜底值，没有分桶。
3. 该头不是鉴权，伪造它不构成对上游的鉴权绕过；但它是上游做路由/缓存决策的依据。

## 盲区（不许假装查全）

- 证据来自 **2026-09-16 的数据备份**；本机没有到 `线上主机` 的 SSH 私钥（`Permission denied (publickey,password)`），无法直连线上库核对"此刻"的配置。
- 线上 `/api/status` 的 `version` 为空字符串 → 跑的是 2026-09-12 12:30（`e1fcb53` 填版本号）之前构建的镜像；要带上版本号需要更新镜像（`v0.0.2` 已发布）。

## Acceptance Criteria

- [x] 定位配置载体（`channels.param_override`，非 `header_override`）与作用渠道
- [x] 定位引入时间（09-12 凌晨到上午）与对应提交（`25c9aa1`）
- [x] 说明上游可见内容与共享缓存桶的代价
- [x] 写明证据时间点与无法直连线上的盲区
- [x] 未改动任何线上配置与代码
