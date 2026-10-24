# 对齐本机 latest 并修 model 包 flaky 测试隔离

## Goal

用户点名（2026-10-24）：
1. 本机 docker `latest` 对齐到 v0.0.48（registry 的 `latest` 已是它）；
2. 修 `model` 包里两个日志预算用例的**测试隔离缺陷**——全仓测试在并发负载下会偶发翻红，
   与业务逻辑无关，但会持续污染以后的验收门禁。

## 已知事实（来自上一轮独立验证）

- 偶发翻红的用例：`TestLogPayloadTrimIsSingleFlight`（失败信息：单飞标志被占用，`Should be true`）与
  `TestLogPayloadBudgetDisabledByDefault`（失败信息：字节计数器残留，例如 `8526`）。
- 相关生产代码：`model/log_budget.go`
  - `logPayloadBytes atomic.Int64`（:50）、`logByteTrimRunning atomic.Bool`（:51）；
  - 写日志时累加（:177）；超过预算时尝试 `CompareAndSwap(false,true)` 启动裁剪（:224），结束后 `Store(false)`（:228）；
    裁剪完成后 `logPayloadBytes.Store(total)`（:334）。
- 机制推断：前面用例写日志触发了 `gopool` 后台裁剪/校准 goroutine，该 goroutine 可能**活过**本用例边界，
  于是：① 它仍持有 `logByteTrimRunning`，使后续用例的 CAS 失败；② 它在断言之后更新 `logPayloadBytes`，使计数器不为 0。
- 证据：单跑该文件/整包 `-count=10`/`-shuffle=on` 均绿；全仓并发负载下 6 次运行有 2 次翻红；本次改动与 `model` 包无关。

## Requirements

- R1 本机 `latest`：`docker tag ghcr.io/zhemed/new-api-own:v0.0.48 ghcr.io/zhemed/new-api-own:latest`；
  核验 `latest == v0.0.48`，`v0.0.47`/`v0.0.46`/`v0.0.45` 保留，实例零影响。
- R2 **先复现**：写一个稳定的复现方式（例如在同包内先执行触发裁剪的用例、再执行目标用例，配合 `-count` 循环或
  `-shuffle`，必要时构造"后台 goroutine 未结束"的时序），证明修复前会翻红；把复现命令与失败输出留证。
- R3 修复**测试隔离**（优先只改测试）：
  - 在受影响用例的 setup/cleanup 里等待后台裁剪收敛（有界轮询，例如等 `logByteTrimRunning` 为 false 且
    `logPayloadBytes` 稳定，带超时与失败信息），不要用固定 `time.Sleep`；
  - 在每个用例结束处复位全局状态（计数器、单飞标志、日志表数据），确保用例可单独运行、不受执行顺序影响；
  - 若必须改动生产代码才能让状态可等待（例如暴露一个仅供测试的等待钩子），需最小化、说明理由，且不得改变线上行为。
- R4 禁止通过"放宽断言/跳过用例/加 sleep 掩盖"的方式让测试变绿；断言要保留原有的契约强度
  （单飞语义、预算关闭时不计字节）。
- R5 验证：修复后至少跑
  - `go test ./model/ -count=20 -shuffle=on`；
  - 全仓 `go test ./... -count=3`；
  - 并尽量复现上一轮的并发负载场景（例如多包并发跑）确认不再翻红。
- R6 红线：不动无关代码；不提交不推送；不发版（本轮为测试隔离修复，是否发版由用户另行决定）。

## 追加需求（2026-10-24，修 model 用例过程中发现）

- R7 **第二个既有 flaky**：`middleware/TestRedisFailurePolicies` 在 `-count>1` 时**必然**失败
  （`rate_limit_test.go:224` 期望 204 实得 429）。根因：用例先 `redisClient.Close()`，
  使 `EmailVerificationRateLimit()` 回落包级全局 `inMemoryRateLimiter`，而
  `common.InMemoryRateLimiter.Init` 只做一次性初始化、**不清空已有 store**；
  `-count=3` 重复执行时同一固定 IP（`192.0.2.62:12345`）第二次即被判 429。
  已在干净 HEAD（stash 掉本任务改动）复现同样失败 → 与本次改动无关。
  处置：修复测试隔离（给内存限流器加测试可见的 reset，或让用例每次使用唯一 key），
  **不得降低断言强度**（失败策略契约必须保留）。
- R8 修复后 `go test ./... -count=3` 应全绿（这是本任务的最终验收命令）。

## Acceptance Criteria

- [ ] AC1 本机 `latest` 与 `v0.0.48` 同 ID；回滚镜像保留；实例 200 + `v0.0.48` 零影响。
- [ ] AC2 有**修复前稳定复现**的命令与失败输出（不是"偶发见过"）。
- [ ] AC3 修复后目标用例在 `-count=20 -shuffle=on` 与全仓 `-count=3` 下全绿；断言强度未被降低。
- [ ] AC4 用例可单独运行、任意顺序运行；`t.Cleanup` 复位全局状态有代码证据。
- [ ] AC5 门禁：`go build ./...`、`go vet ./...`、`go test ./model/... ./controller/... -count=1`。
- [ ] AC6 独立验证（非实现者自证），并记录 flaky 是否真的消失（给出多次运行统计）。
- [ ] AC7 `middleware/TestRedisFailurePolicies` 在 `-count=3` 下通过，且该用例仍验证原有的失败策略契约。
- [ ] AC8 `go test ./... -count=3` 全绿（有输出证据）。

## Notes

- 这是"测试可靠性"修复，不发版：只影响 CI/本地门禁可信度。
- 若发现根因其实是生产代码的 goroutine 泄漏（而非单纯测试隔离），需在证据里如实升级问题并给出建议，不要只做表面等待。
