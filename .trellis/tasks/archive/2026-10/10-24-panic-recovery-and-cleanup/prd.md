# 加固 panic recovery + 分类 79 个待确认符号 + 清旧回滚镜像

## 重要更正（Lead 复核后）

上一轮独立验证报告称「全仓不存在 panic recovery 中间件」，**该结论有误**：
`main.go:174-188` 实际装配了 `gin.New()` + `server.Use(gin.CustomRecovery(...))`，panic 是有引擎级兜底的。
验证者只 grep 了 `router/` 与 `middleware/`，漏看了 `main.go`。

因此本任务 R1 从"补中间件"改为**加固既有 recovery**。

## Goal

用户点名三件（2026-10-24）：
1. 加固 panic recovery（既有实现有信息泄露与可观测性缺陷）；
2. 把 79 个「需确认」导出 API 分三堆出清单（**只读，不删代码**）；
3. 清旧回滚镜像，只留 v0.0.49 与 v0.0.48。

## Requirements

### R1 加固 panic recovery（代码 + 测试）

现状（`main.go:179-187`）：
```go
server.Use(gin.CustomRecovery(func(c *gin.Context, err any) {
    common.SysLog(fmt.Sprintf("panic detected: %v", err))
    c.JSON(http.StatusInternalServerError, gin.H{
        "error": gin.H{
            "message": fmt.Sprintf("Panic detected, error: %v. Please submit a issue here: https://github.com/zhemed/new-api-own", err),
            "type":    "new_api_panic",
        },
    })
}))
```
问题：
- **P1 信息泄露**：把 panic 原文（`%v`）直接回给客户端。panic 信息可能含数据库错误、SQL 片段、token/密钥等敏感内容。
- **P2 可观测性不足**：只 `SysLog`（非 error 级）、**不打堆栈**、不带 request id，线上定位困难。
- **P3 响应已写时的安全性**：若响应已部分写出（SSE/流式），仍会追加 JSON 造成脏响应。

要求：
- R1.1 客户端只返回**通用文案**（如 `内部错误，请稍后重试`，i18n 交由前端；后端固定英文或中文择一，与既有错误结构保持一致：
  `{"error":{"message":...,"type":"new_api_panic"}}`），**绝不回显 panic 原文**。
- R1.2 服务端记录**error 级**日志：panic 值 + **完整堆栈**（`debug.Stack()`）+ 请求方法/路径 + `request_id`（若 `middleware.RequestId` 已设置到 context）。
  日志仍需脱敏（不打印 Authorization/Cookie 等请求头）。
- R1.3 若 `c.Writer.Written()` 为真（响应已开始写），**不要再写 JSON**，只 `c.Abort()` 并记录日志，避免脏响应。
- R1.4 保持既有行为：panic 后该请求返回 500、**进程不退出**、后续请求仍可正常处理；保留 `broken pipe` 的既有语义（gin CustomRecovery 内部已处理，不要改坏）。
- R1.5 测试（先写失败用例，用 gin test mode）：
  ① panic 的 handler → 500 + 响应体**不含** panic 原文（用敏感假串验证）；
  ② 日志被记录且包含堆栈标记（可用可注入 logger 或对 `common.SysError` 打桩/捕获）；
  ③ 连续两次请求：第一次 panic、第二次正常 → 证明进程与路由未被破坏；
  ④ 已写响应的场景（先 WriteHeader/Write 再 panic）→ 不追加 JSON、不 panic 二次。
- R1.6 若发现 `common.SysError` 无堆栈能力，允许新增一个专用 helper（如 `common.SysErrorWithStack`），不改变既有函数签名。

### R2 分类 79 个「需确认」导出符号（**只读报告**）

- 对象：`.trellis/tasks/archive/2026-10/10-24-deadcode-docs-ci/evidence-deadcode.md` §5.2 的 79 个导出符号。
- 逐符号给出：包、符号、deadcode 判定、是否被文档/规范点名（AGENTS.md / `.trellis/spec/**`）、
  是否属于对外 API 面（`relaykit` 独立发布模块、`common` 公共工具、`types/pkg` 基础设施、`oauth/setting` 配置面），
  并**归入三堆**：① 可删（无引用且非公开面）② 需补调用（本该被用到却没人调） ③ 保留（公开面/文档约定/反射）。
- **禁止删除任何代码**；产出清单表（按包分组）+ 建议执行顺序 + 每堆风险等级。

### R3 清旧回滚镜像（本机 docker）

- 现状镜像：`v0.0.49`（当前运行）、`v0.0.48`、`v0.0.47`、`v0.0.46`、`v0.0.45`、`latest`(=v0.0.49)。
- 目标：**只保留 `v0.0.49` 与 `v0.0.48`**（含 `latest` 指向的同一 image ID），删除 `v0.0.47`、`v0.0.46`、`v0.0.45` 的 tag 与其独占镜像层。
- 红线：
  - **必须用 allow-list 方式**（先算出「要保留的 image ID 集合」，再从「可删候选」里排除），禁止按名字前缀/通配盲删；
    历史上曾出现把运行中镜像误列入删除名单的险情。
  - 删除前打印「将删除 / 将保留」两份清单并**二次核对运行中容器的 image ID 不在删除清单内**（`docker inspect` 取真实 ID，不截断）。
  - 不使用 `docker system prune` / `docker image prune -a`（会波及无关镜像）；不删卷、不删容器。
  - 删除后复核：实例仍 running、200 + v0.0.49、`restarts=0`、`v0.0.48` 仍在、`latest` 仍指向 v0.0.49。

## Acceptance Criteria

- [ ] AC1 recovery 加固落地：客户端不回显 panic 原文、服务端有堆栈级日志、响应已写时不追加 JSON（均有测试）。
- [ ] AC2 `go build/vet`、相关包测试、全仓 `go test ./... -count=1` 全绿；`gofmt -l` 空。
- [ ] AC3 79 个符号三堆清单齐备（只读，无代码改动）。
- [ ] AC4 镜像清理后仅剩 v0.0.49 / v0.0.48（+latest），实例健康、回滚点 v0.0.48 可用。
- [ ] AC5 独立验证通过（非实现者自证）。

## Notes

- R1 是**行为变更**（安全加固），需要发版才到实例；发版与否留待用户决定（本轮不自动发）。
- 本轮先更正上一轮的错误结论并写入任务记录，避免"假问题"变成后续工作依据。
