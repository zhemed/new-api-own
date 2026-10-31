# 排查 429 ttft_admission_refused 无法使用

## Goal

用户调用模型时持续失败，前端显示：xx API error (429): {"message":"Upstream request failed: [ttft_admission_refused] the cell is at capacity; retry shortly (after 4 s)","type":"upstream_error","code":"server_error"}，标注 RATE_LIMIT，已重试模型请求 5/5 后本轮运行失败。本次任务以只读方式查清该错误的来源与用户无法使用的原因，并给出可执行处置建议。

## Requirements

- R1 全链路定位：梳理 客户端 → new-api 前端 → relay 层 → 渠道 → 上游服务商 的错误传递路径，确认 `ttft_admission_refused` 与 `the cell is at capacity` 的原始产生方，以及 429 是否被本网关注入或改写。
- R2 渠道归属取证：查明报错中 `xx API error` 的 `xx` 在代码/UI 中的真实含义，以及用户当前请求实际命中的渠道、模型与上游 base URL，证据脱敏。
- R3 网关行为核对：核对重试策略、5/5 上限、`RATE_LIMIT` 标记、渠道自动禁用与限流配置，判断本网关是否放大或掩盖了上游故障。
- R4 运行态取证：用户当轮指定生产主机并授权 SSH 访问，取证以生产机为准，本机开发环境仅作代码参考；只读检查容器/进程状态、服务日志、数据库中的渠道与近期错误日志，判断是单渠道故障、上游容量问题，还是全局不可用。
- R6 凭据纪律：生产主机名、账号、密码不写入任何持久文件，仅在会话内使用；本轮取证范围严格只读，发现问题另开任务并取得当轮明确授权后再修复。
- R5 处置建议：分别给出临时可用手段与根治手段，明确风险、影响面与需用户授权后才能执行的动作。
- 约束：本次仅只读，不修改代码、配置、数据库，不重启服务，不发起写网络请求，不读取或外发任何凭据。

## Acceptance Criteria

- [x] 给出错误链路说明，指出 429 与 `ttft_admission_refused` 的产生位置与传递过程。
- [x] 给出实际命中的渠道、模型、上游地址证据，敏感字段脱敏。
- [x] 给出运行态证据，包括容器状态、近期相关日志、渠道与错误统计。
- [x] 明确回答"为什么无法使用"，并在 上游容量拒绝 / 渠道配置问题 / 网关侧限流 之间给出有证据支撑的判断。
- [x] 给出可执行处置建议，含临时规避方案与需授权动作清单。
- [x] 调查证据落盘到任务目录 research/ 下，最终答复引用对应文件路径。

## Result

- 结论：生产唯一渠道 id=1 的上游 OpenCode Zen 在 2026-10-29 02:14-02:26Z 做容量准入拒绝，返回 429 ttft_admission_refused；网关原样透传，客户端 DSH 归类 RATE_LIMIT、重试 5/5 后本轮运行失败。网关、账号、额度、渠道状态均正常，非网关限流或封禁。
- 放大因素：单渠道单模型、无备用渠道、网关 RetryTimes=0 且 429 不触发自动禁用或熔断。
- 现状：02:27Z 上游恢复；04:09-04:10Z 主机与容器重启后至今 0 次 429，请求正常。
- 证据文件：research/prod-raw.md、research/prod-runtime.md、research/code-path.md、research/prod-collect.sh。
- 未实施任何生产改动；后续处置需用户授权并另开任务。

## Notes

- 任务类型：只读调查，轻量任务，PRD-only 即可，不产生代码改动。
- 前端报错截图见本轮用户消息附件，关键字段：429、ttft_admission_refused、the cell is at capacity、RATE_LIMIT、已重试 5/5、重试延迟 8 秒级。
- 若确认需要改代码或改渠道配置，另开实现任务，不在本任务内实施。
