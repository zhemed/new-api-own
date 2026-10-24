# Journal - new-api-own (Part 3)

> Continuation from `journal-2.md` (archived at ~2000 lines)
> Started: 2026-10-24

---



## Session 114: 加固 panic recovery（不回显原文/堆栈日志/已写响应保护）；清旧回滚镜像；分类 85 个导出符号
<!-- trellis-session: v=2 fp=267138390197404e -->

**Date**: 2026-10-24
**Task**: 加固 panic recovery（不回显原文/堆栈日志/已写响应保护）；清旧回滚镜像；分类 85 个导出符号
**Branch**: `main`

### Summary

更正上轮错误结论：panic recovery 一直存在（main.go:174 的 gin.CustomRecovery），不是缺失。真实问题是三个：把 panic 原文回显给客户端（信息泄露）、只记 SysLog 无堆栈无 request id、响应已写时仍追加 JSON。已抽成 middleware.PanicRecovery 并修掉三点（5 个测试先红后绿）。另清掉 v0.0.45-47 三个旧镜像（allow-list 释放约 451MB），并把 85 个导出符号分成 A76/B4/C5 三堆（含 owner 决策项）。

### Main Changes

- 新增 middleware/panic_recovery.go：CustomRecoveryWithWriter(nil) 关闭 gin 自带日志（其 debug 模式会明文 dump Cookie），客户端只拿通用文案，服务端 error 级日志含完整堆栈+方法/路径+request id，c.Writer.Written() 为真时只 Abort
- main.go：内联 recovery 换成 server.Use(middleware.PanicRecovery())，仍在最外层先于 RequestId
- 新增 middleware/panic_recovery_test.go 五用例：不泄露敏感串、日志含堆栈与上下文、panic 后路由可用、已写响应不追加 JSON、broken pipe 不写 body
- 镜像清理：删除 v0.0.47/46/45（allow-list + 完整 ID 保留集合 + 运行中镜像二次核对 + 无 prune），保留 v0.0.49(=latest) 与 v0.0.48，释放约 450.79 MiB
- 符号分类：85 个 = A 可删 76 / B 需补调用 4 / C 保留 5；按包 common31、relaykit23、pkg13、setting12、oauth2、types2、relay1、service1

### Git Commits

| Hash | Message |
|------|---------|
| `193a409` | fix(security): panic recovery 不回显原文、error 级堆栈日志、已写响应不追加 JSON [task:panic-recovery-and-cleanup] |
| `f3a2353` | chore(task): recovery 加固、镜像清理与符号分类的证据 [task:panic-recovery-and-cleanup] |

### Testing

- [OK] [OK] 门禁：build/vet/全仓 36 包测试/gofmt 空；recovery 5 用例全 PASS（修复前 3 条真实失败：响应含 sk-live-DEADBEEF、无 ERR/堆栈、已写响应被追加 JSON）
- [OK] [OK] 镜像清理后：实例 v0.0.49 running/restarts=0/200、8 容器、卷 3、data 59 文件不变；v0.0.48 回滚点仍在

### Status

[OK] **Completed**

### Next Steps

- 本 recovery 加固是行为变更，需发 v0.0.50 才到实例（用户未点头，暂未发）
- 归档证据 evidence-deadcode.md §5.2 正文为空（上一轮写丢），本轮已在 evidence-symbols.md 重建 85 条；建议后续把清单回填并保留 79/82/85 口径说明
- A 堆 76 个可删（建议顺序：pkg+oauth+types+setting 29 → common 29 → relaykit 18 需 owner 批 → B 4 项接线+单测）；relaykit 独立模块 API 面需 owner 决策
