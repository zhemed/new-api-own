# 证据：`model.UpdateOption` 传播 DB 错误 + 调用点审查 + 回归

- 任务：Trellis `10-24-fix-updateoption-error-propagation`；共享任务板 `task-31`
- 执行者：worker-script；时间：2026-10-24
- 写作用域：`model/option.go`、`model/option_test.go`（新增）、`controller/pricing_test.go`、本文件。
- 未改 `UpdateOptionsBulk`；未动 `web/`；未提交、未推送、未发版；无外部主机访问。
- 说明：PRD 的 **R6（本机 docker `latest` 对齐 v0.0.47）不在本次 task-31 范围**（task-31 只含代码侧
  R1–R5，写作用域也不含 docker），由其它成员按 PRD 执行。

## 1. 根因与修复

修复前（`model/option.go:218-235`）：

```go
DB.FirstOrCreate(&option, Option{Key: key})   // 返回值丢弃
option.Value = value
DB.Save(&option)                              // 返回值丢弃 → 写库失败也返回 nil
return updateOptionMap(key, value)            // 内存照样更新 → DB 与内存分叉
```

后果：任何写库失败都被当成成功。直接受害者是本轮的「重置定价」接口——它会回
`{"success":true}` 而实例状态没变（上一任务已实测复现）。

修复后：

```go
if err := DB.FirstOrCreate(&option, Option{Key: key}).Error; err != nil {
	return fmt.Errorf("cannot create option %q: %w", key, err)
}
option.Value = value
if err := DB.Save(&option).Error; err != nil {
	return fmt.Errorf("cannot save option %q: %w", key, err)
}
// 只有 DB 落盘成功才同步内存
return updateOptionMap(key, value)
```

关键点：
- **两个写库动作的错误都传播**；
- **写库失败时不再调用 `updateOptionMap`** → 内存保持旧值，DB 与内存不分叉；
- 错误信息只格式化 **key**（`%q`），**从不格式化 value** → 不泄露配置值（可能含密钥/支付配置）；
- `%w` 保留底层错误便于排障（见 §5 残留风险评估）；
- 风格与同文件既有的正确写法 `UpdateOptionsBulk`（事务 + `.Error` 检查）一致。
- `UpdateOptionsBulk` **一行未改**（`git diff -U0 -- model/option.go | grep -c UpdateOptionsBulk` = 0）。

## 2. 全仓调用点审查（R2）

**结论：4 个生产调用点全部已处理返回值，无需改动**；不存在 `_ =` 忽略或裸调用。

| # | 调用点 | 返回值处理 | 修复后行为 | 可接受性 |
| --- | --- | --- | --- | --- |
| 1 | `controller/option.go:366`（管理员改配置 `POST /api/option/`） | `if err != nil { common.ApiError(c, err); return }` | 原先 DB 失败仍返回 `success:true` 并写审计日志；现在返回 `success:false` + 错误信息，且**不写** `option.update` 审计（因提前 return） | ✅ 更正确：不再产生"配置已更新"的假审计 |
| 2 | `controller/pricing.go:105`（重置定价，上一任务新增） | 循环内 `if err := ...; err != nil { success:false; return }` | 写库失败如实返回 `success:false` | ✅ 本缺陷的直接修复目标 |
| 3 | `controller/setup.go:129`（初始化：SelfUseModeEnabled） | `err = ...; if err != nil { success:false; return }` | 初始化不再"带病继续"：失败时提前返回，**不会**置 `constant.Setup = true`、不会写入 Setup 记录，管理员可重试 | ✅ 更正确（避免半初始化实例被标记为已初始化） |
| 4 | `controller/setup.go:138`（初始化：DemoSiteEnabled） | 同上 | 同上 | ✅ |
| 5 | `controller/payment_compliance.go:62`（支付合规确认，5 键循环） | `if err := ...; err != nil { common.ApiError(c, err); return }` | 原先 DB 失败仍返回 `confirmed:true`；现在如实失败 | ✅ 更正确；注意多键循环非事务，中途失败时前面的键可能已写入（见 §5） |

测试侧调用点（不影响生产语义）：
- `model/frontend_option_migration_test.go:176`、`controller/pricing_test.go` 多处：均 `require.NoError` 包裹；
- `controller/theme_compat_test.go:24`、`option_gemini_test.go:24`、`option_claude_test.go:24` 里的
  `UpdateOption(context)` 是 **gin handler**（`controller.UpdateOption`），与 `model.UpdateOption` 同名不同物，不受影响。

隐含假设检查：**未发现**任何调用方依赖"`UpdateOption` 永不返回 DB 错误"。
没有任何生产调用点在 `main.go` / `router/` 启动路径上（启动期不写 option），因此不存在
"DB 尚未初始化就调用 → 因 nil DB panic" 的新增风险（`DB` 为 nil 时 `FirstOrCreate` 本身会 panic，
这是修复前后一致的既有前提，调用点都发生在 DB 就绪之后）。

## 3. 测试（先写失败用例，再修复）

新增 `model/option_test.go`（沿用同包 `frontend_option_migration_test.go` 的内存 SQLite 隔离手法：
临时替换 `DB` 与 `common.OptionMap`，测试结束恢复）：

1. `TestUpdateOptionPropagatesSaveErrorAndKeepsMemory`：先正常写入建行，再挂
   `BEFORE UPDATE` 触发器强制 `Save` 失败 → 断言返回 error、错误含 key 且**不含 value**、
   **内存保持旧值**、DB 保持旧值。
2. `TestUpdateOptionPropagatesCreateError`：挂 `BEFORE INSERT` 触发器强制 `FirstOrCreate` 失败 →
   断言返回 error、错误含 key 不含 value、内存未写入。
3. `TestUpdateOptionSuccessKeepsDBAndMemoryInSync`：正常 INSERT 与 UPDATE 两条路径，DB 与内存一致。

控制器层（`controller/pricing_test.go`）：把上一任务用于"钉住旧行为"的用例改写为正向断言
`TestResetModelRatioWriteFailureReturnsFailure`：触发器让写库失败 → 接口必须返回 `success:false`
且内存倍率保持旧值（不得半途改成默认）。

### 修复前后对照（实测）

修复前（`model/option.go` 原版 + 新测试）：

```console
$ go test ./model/ -count=1 -run 'TestUpdateOption' -v
=== RUN   TestUpdateOptionPropagatesSaveErrorAndKeepsMemory
    Error: An error is expected but got nil.
    Messages: Save 失败必须传播为 error（修复前这里返回 nil）
--- FAIL: TestUpdateOptionPropagatesSaveErrorAndKeepsMemory (0.00s)
=== RUN   TestUpdateOptionPropagatesCreateError
    Error: An error is expected but got nil.
    Messages: FirstOrCreate 失败必须传播为 error（修复前这里返回 nil）
--- FAIL: TestUpdateOptionPropagatesCreateError (0.00s)
=== RUN   TestUpdateOptionSuccessKeepsDBAndMemoryInSync
--- PASS: TestUpdateOptionSuccessKeepsDBAndMemoryInSync (0.00s)
FAIL

$ go test ./controller/ -count=1 -run 'TestResetModelRatioWriteFailureReturnsFailure' -v
--- FAIL: TestResetModelRatioWriteFailureReturnsFailure (0.02s)
    Error: "{\"message\":\"重置模型倍率成功\",\"success\":true}" does not contain "\"success\":false"
    Messages: 写库失败必须如实返回失败：{"message":"重置模型倍率成功","success":true}
```

修复后：

```console
$ go test ./model/ -count=1 -run 'TestUpdateOption' -v
--- PASS: TestUpdateOptionPropagatesSaveErrorAndKeepsMemory (0.00s)
--- PASS: TestUpdateOptionPropagatesCreateError (0.00s)
--- PASS: TestUpdateOptionSuccessKeepsDBAndMemoryInSync (0.00s)
ok  	github.com/QuantumNous/new-api/model	0.024s

$ go test ./controller/ -count=1 -run 'TestResetModelRatio' -v
--- PASS: TestResetModelRatioRestoresFullPricingDefaults (0.02s)
--- PASS: TestResetModelRatioClearsEmptyDefaultTables (0.02s)
--- PASS: TestResetModelRatioWriteFailureReturnsFailure (0.02s)
ok  	github.com/QuantumNous/new-api/controller	0.059s
```

## 4. 门禁输出（全部实际执行）

```console
$ go build ./...
exit=0

$ go vet ./...
exit=0

$ go test ./service/... ./controller/... ./setting/... ./model/... -count=1
ok  	github.com/QuantumNous/new-api/service
ok  	github.com/QuantumNous/new-api/service/authz
?   	github.com/QuantumNous/new-api/service/passkey	[no test files]
ok  	github.com/QuantumNous/new-api/controller
ok  	github.com/QuantumNous/new-api/setting
ok  	github.com/QuantumNous/new-api/setting/billing_setting
ok  	github.com/QuantumNous/new-api/setting/config
?   	github.com/QuantumNous/new-api/setting/console_setting	[no test files]
ok  	github.com/QuantumNous/new-api/setting/model_setting
ok  	github.com/QuantumNous/new-api/setting/operation_setting
?   	github.com/QuantumNous/new-api/setting/perf_metrics_setting	[no test files]
?   	github.com/QuantumNous/new-api/setting/performance_setting	[no test files]
ok  	github.com/QuantumNous/new-api/setting/ratio_setting
?   	github.com/QuantumNous/new-api/setting/reasoning	[no test files]
ok  	github.com/QuantumNous/new-api/setting/system_setting
ok  	github.com/QuantumNous/new-api/model

$ go test ./... -count=1          # 额外：全仓回归（底座改动，范围外也确认）
（除 ok/? 外无输出 = 全绿）
```

## 5. 影响面、残留风险与回滚

### 影响面（行为变更：更正确，但确有可观察变化）

- 凡 DB 写 `options` 失败，**以前静默成功、现在会返回错误**。受影响的对外接口：
  `POST /api/option/`、`POST /api/option/rest_model_ratio`、初始化 `POST /api/setup`、
  支付合规确认接口。正常路径（DB 可用）**行为完全不变**。
- 副作用是正向的：不再出现"接口说成功但实例没变"，也不再写与事实不符的审计日志。
- 立即回滚方式：**单函数 revert** `model.UpdateOption`（把两处 `if err := ...; err != nil`
  改回裸调用即可），不牵动数据、迁移或其它函数。

### 残留风险（均已记录，不在本次范围）

1. **`%w` 包装底层错误**：错误里带 gorm/驱动原文。已确认测试中驱动错误不含绑定参数（值未泄露），
   且我从不格式化 `value`；但理论上某些驱动可能把参数带进错误串。若要绝对保险，可改为不透传 `%w`
   （代价是排障信息变少）——当前按"能定位 key + 便于排障"取向保留。
2. **DB 成功但 `updateOptionMap` 失败**（例如 ratio 键的 JSON 非法）：此时 DB 已更新、内存未更新，
   仍会短暂分叉。属修复前既有行为（R1 只要求"写库失败不同步内存"，未要求反向补偿），
   且 `SyncOptions` 周期任务 / 重启会从 DB 回灌，可自愈。**未改**，避免扩大到事务化重构。
3. **多键循环非事务**（`payment_compliance.go` 5 键、重置定价 10 键）：中途失败时前面的键已写入。
   现在至少会如实报错（以前是静默半成功 + 报成功）。做成整体事务需改 `model` 层接口，超出本任务。
4. **`controller/setup.go` 的直改内存**：`operation_setting.SelfUseModeEnabled/DemoSiteEnabled`
   在 `UpdateOption` **之前**就被直接赋值。DB 失败时该直改不会回滚（`OptionMap` 不会被改）。
   属该 handler 既有顺序问题，与 `UpdateOption` 无关；且失败时 setup 会提前返回、不会标记已初始化。
5. `UpdateOptionsBulk` 语义**未改**（仍是"事务 + 成功后才同步内存"）。

## 6. 改动清单

| 文件 | 改动 |
| --- | --- |
| `model/option.go` | `UpdateOption` 检查 `FirstOrCreate`/`Save` 的 `.Error` 并返回（错误带 key，不带 value）；写库失败时提前 return，不调 `updateOptionMap`；新增 `fmt` import |
| `model/option_test.go`（新增） | 3 个用例：Save 失败传播 + 内存不变、Create 失败传播、正常路径 DB/内存一致 |
| `controller/pricing_test.go` | 旧"钉住静默成功"用例改写为 `TestResetModelRatioWriteFailureReturnsFailure`（写库失败 → `success:false` + 内存不变） |
| `.trellis/tasks/10-24-fix-updateoption-error-propagation/evidence-updateoption.md` | 本证据 |

## 7. 建议下一步

1. Lead/verifier 复核 `git diff -- model/option.go controller/pricing_test.go` 与 §2 调用点结论。
2. 接受后按闸门提交（锚点 `[task:fix-updateoption-error-propagation]`）。
3. 若要彻底消除 §5.2/§5.3 的分叉与半写，建议单开任务做"option 写入事务化"（改 `model` 层接口），
   本轮刻意不做以避免底座大改。
4. PRD 的 R6（docker `latest` 对齐）由对应成员执行；本轮未触碰 docker。
