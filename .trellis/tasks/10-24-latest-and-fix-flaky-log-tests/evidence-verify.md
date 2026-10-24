# 两处测试隔离 flaky 修复：独立验证证据（verifier）

- 任务：共享任务板 `task-39`（Trellis 任务 `10-24-latest-and-fix-flaky-log-tests` 的独立验证）
- 验证者：verifier（只读验证；未改任何仓库文件）
- 验证时间：2026-10-24（UTC 05:01 起）
- 基线：`HEAD = c7ffc5f`；改动 3 文件（`common/rate-limit.go` +18、`middleware/rate_limit_test.go` +16、`model/log_budget_test.go` +70/-12）
- 基线指纹：`git diff | sha256sum = 36679a55bac5980578f30406d75eb4c8ffa9c840b9b82b72518ad25cdf88338c`
- 被验证文件 blob：`common/rate-limit.go = af32aaba062998c2fa664bbf75a8f25de35946d4`、
  `middleware/rate_limit_test.go = 576e3c852bc526f0a267f71bca2985c5e4d14a46`、
  `model/log_budget_test.go = d05e52174f19cd17ae0d98d46302a0bf103e433a`
- 说明：本任务 R1/R2 中的「本机 latest 对齐 v0.0.48」属实例/镜像操作，不在代码只读验证范围（由 Lead/成员另行留证）。

## 结论

**接受（Accept）**。两处 flaky 均有**确定性复现**且修复后同一场景稳定转绿；断言强度未降低（无断言删除/放宽、无 Skip、无新增 sleep，
等待 helper 超时会真的 fail）；`Reset()` 无生产调用且持锁、未初始化语义正确；断言强度与门禁全部通过，
其中 R8 的 `go test ./... -count=3` 在**并发负载下**也全绿（两次全绿，一次单跑、一次与 model -count=20 并发）。

## 一、修复前后对照（关键证据）

### 1.1 model 包日志预算（确定性复现，非靠概率）

方法：`git clone /root/new-api-own /tmp/verify-flaky`（改动未提交 → 克隆即旧测试），在克隆内新增一个临时测试文件，
用实现者的确定性手法复现「上个用例遗留的后台裁剪」：
占住 model 测试唯一 DB 连接（`TestMain` 已 `SetMaxOpenConns(1)`）→ 调生产函数 `triggerLogPayloadTrim()`
（同步占单飞 + 异步起裁剪）→ 150ms 后 `tx.Rollback()` 放行 → 直接以子测试方式调用**真实目标用例**。
**全程未在真实工作区做 checkout/stash/restore**。

| 场景 | 旧测试文件（克隆） | 新测试文件（同一克隆） |
|---|---|---|
| repro run 1 | FAIL：`Should be true`（log_budget_test.go:80，单飞标志被遗留裁剪占用） | PASS（singleflight 0.15s / budget_disabled 0.04s） |
| repro run 2 | FAIL：同上 | PASS |
| repro run 3 | FAIL：同上 | PASS |

- 失败原文与上一轮全仓翻红时**逐字一致**（`Should be true`），对应「单飞被上一个用例的后台裁剪占着」。
- 我这一版时序下只有「单飞被占」稳定复现；「计数器残留」分支在我这 3 次中未触发（budget_disabled PASS），
  属同一根因的另一种时序表现；新 helper 同时等待「单飞空闲 + 计数器连续稳定」，覆盖两条路径。

### 1.2 middleware 包 `TestRedisFailurePolicies`（-count 必现失败）

| 场景 | 修复前（克隆旧测试 + 旧 common） | 修复后（真实工作区） |
|---|---|---|
| `go test ./middleware/ -run TestRedisFailurePolicies -count=3 -v` | 第 1、2 次 PASS；**第 3 次 FAIL**：rate_limit_test.go:224 `expected: 204 actual: 429` | `-count=5` 全 PASS；`-count=3 -race` 全 PASS |

失败原文与 PRD R7 记录一致；原因是用例让 Redis 不可用后回落包级内存限流器，固定 IP 的计数跨 repetition 残留。

## 二、断言强度核对（R4）

| 核对项 | 证据 | 判定 |
|---|---|---|
| 无断言被改写/删除/放宽 | `git diff -U0` 的删除行中**没有**任何 `assert.` / `require.`（model 删除行仅为 8 处 setup 行与 1 处函数签名行；middleware diff 为纯新增 +16） | 通过 |
| 契约断言原样保留 | 「并发触发只允许一路赢得单飞」（`assert.Equal(int32(1), winners)`）与「未配置预算时不记账」（`assert.Zero(logPayloadBytes.Load())`）均未改动 | 通过 |
| 无 `t.Skip` | 两个测试文件 grep `t.Skip` 无命中 | 通过 |
| 无新增 `time.Sleep` 掩盖 | diff 新增行 grep `time.Sleep` 无命中；两文件现存全文亦无 `time.Sleep`；等待使用 `require.Eventually`（5ms 轮询 / 5s 上限） | 通过 |
| 超时真的会 fail | 在克隆内加临时探针：强制 `logByteTrimRunning.Store(true)` 后调 `waitLogBudgetIdle` → **5.00s 后 FAIL**，输出 `Condition never satisfied` + `日志预算后台任务未在 5s 内收敛（…）：logByteTrimRunning=true, logPayloadBytes=0` | 通过 |
| 8 个用例都改用统一隔离 | `beginLogBudgetTest(t)` 调用 8 处，覆盖文件内全部日志预算用例（含单飞用例与磁盘库用例） | 通过 |
| helper 的 cleanup 顺序 | `beginLogBudgetTest` 先注册 `truncateTables` 的删表 cleanup、后注册收敛等待 → LIFO 下先收敛后删表（与注释一致），避免后台任务读表时被清空 | 通过 |

## 三、`InMemoryRateLimiter.Reset()` 核对（R3/R7）

| 核对项 | 证据 | 判定 |
|---|---|---|
| 无生产调用 | 全仓 `grep '.Reset()'` 非测试命中只有 `relay/.../openai/relay_responses.go` 的 `imageCounter.Reset()` 与 relaykit 的 `tool.Arguments.Reset()`（不同物）；`inMemoryRateLimiter.Reset()` 只出现在 `middleware/rate_limit_test.go:57-58`（用例首尾各一次，经 `t.Cleanup`） | 通过 |
| 并发安全 | `Reset` 全程持 `l.mutex`（与 `Request` / `clearExpiredItems` 同一把锁）；`-race` 下并发探针（8 goroutine 持续 `Request` + 200 次 `Reset`）PASS | 通过 |
| 未初始化保持未初始化 | 探针：零值 `InMemoryRateLimiter{}.Reset()` 不 panic，`store` 仍为 nil；随后 `Init(time.Minute)` 仍能正常初始化 | 通过 |
| 保留初始化属性 | 探针：已初始化实例 Reset 后窗口清空、`expirationDuration` 不变、复位后可继续 `Request` 放行；清理 goroutine 未受影响（Reset 不触碰其启动逻辑） | 通过 |
| 生产行为不变 | `common/rate-limit.go` 只有 1 个 hunk（新增 Reset，+18/-0），`Init/Request/clearExpiredItems` 一行未改 | 通过 |

## 四、门禁复跑（真实工作区，Go 全部 `GOPROXY=off`）

| 门禁 | 命令 | 结果 |
|---|---|---|
| Go 构建 | `go build ./...` | rc=0 |
| Go 静态检查 | `go vet ./...` | rc=0 |
| 目标包测试 | `go test ./model/... ./controller/... ./middleware/... -count=1` | 全 ok（model 6.416s / controller 0.447s / middleware 0.040s） |
| model 压力 | `go test ./model/ -count=20 -shuffle=on` | ok（两轮：126.8s、130.4s） |
| middleware 回归 | `go test ./middleware/ -run TestRedisFailurePolicies -count=5` | ok |
| 全仓 R8 | `go test ./... -count=3` | **rc=0，无 FAIL 行**（单跑一次） |
| 全仓 R8（负载下） | 同一命令与 `model -count=20 -shuffle=on` **并发**执行 | **rc=0，无 FAIL 行**（复现上一轮「并发负载下翻红」的原始条件仍绿） |
| 竞态 | `go test -race ./middleware/ ./common/`、`-race ./middleware/ -run TestRedisFailurePolicies -count=3` | 均 ok |

## 五、边界

- `git diff --name-only` 恰为 3 个文件：`common/rate-limit.go`、`middleware/rate_limit_test.go`、`model/log_budget_test.go`；无顺手改动。
- hunk 数：rate-limit.go 1 个（纯新增）、middleware 测试 2 个（纯新增）、log_budget 测试 9 个；`gofmt -l` 三个文件均无输出。
- 未跟踪仅任务目录；真实工作区未被本次验证修改（基线指纹验证前后一致）。
- 临时探针与复现文件只存在于 `/tmp/verify-flaky`，未写入仓库。

## 六、风险与不确定项（均非阻断）

1. `waitLogBudgetIdle` 观测的是「单飞标志 + 计数器稳定」，无法感知**尚未启动**的 fire-and-forget goroutine。
   实际风险很低：一次性校准是 `sync.Once` 且在进程早期触发；R8 负载测试与 model -count=20 两轮均绿。若要更稳可后续加测试专用等待钩子。
2. 磁盘库用例（`TestLogPayloadBudgetWorksOnDiskBackedLogs`）的 cleanup 顺序是：先恢复/关闭磁盘库（用例自身 cleanup）、
   再等后台收敛（helper cleanup）——若残留裁剪仍在使用已关闭的磁盘库，只会记一条日志，不影响断言。压力运行未出现该路径问题。
3. `Reset()` 是 `common` 包的导出方法，目前仅测试使用；未来若无意识地在生产调用，会清空全局限流窗口。
   方法注释已写明「Production code has no reason to call it」，风险可接受。
4. R1/R2（本机 latest 对齐 v0.0.48、实例 200 与版本核对）属镜像/实例操作，不在本只读验证范围。

## 七、复跑命令清单

```bash
# model flaky 修复前后对照（确定性复现）
rm -rf /tmp/verify-flaky && git clone -q /root/new-api-own /tmp/verify-flaky
# 旧测试 + /tmp 临时复现文件（占唯一连接 + triggerLogPayloadTrim + 150ms 后放行，再调用两个真实用例）
cd /tmp/verify-flaky && GOPROXY=off go test ./model/ -run 'TestZZReproLeftoverTrim' -count=1 -v
cp /root/new-api-own/model/log_budget_test.go model/log_budget_test.go   # 换新测试文件后重跑 → 应为 PASS

# middleware flaky 修复前后对照
cd /tmp/verify-flaky && GOPROXY=off go test ./middleware/ -run 'TestRedisFailurePolicies' -count=3 -v  # 旧：第 3 轮 204->429

# 真实工作区门禁
cd /root/new-api-own && GOPROXY=off go build ./... && GOPROXY=off go vet ./...
GOPROXY=off go test ./model/... ./controller/... ./middleware/... -count=1
GOPROXY=off go test ./model/ -count=20 -shuffle=on
GOPROXY=off go test ./middleware/ -run 'TestRedisFailurePolicies' -count=5
GOPROXY=off go test ./... -count=3
CGO_ENABLED=1 GOPROXY=off go test -race ./middleware/ ./common/
```
