# UpdateOption 错误传播：独立验证证据（verifier）

- 任务：共享任务板 `task-32`（Trellis 任务 `10-24-fix-updateoption-error-propagation` 的独立验证）
- 验证者：verifier（只读验证；未改任何仓库文件）
- 验证时间：2026-10-24（UTC 04:00 起）
- 基线：`HEAD = a7108d9`；已跟踪改动 2 文件（19 增 18 删，其中 option.go 为底座修复）+ 1 个新增未跟踪测试
- 基线指纹：`git diff | sha256sum = 0f66cdc7f7f9b5d3a265938ed219d623f9a615b1172a808f3eca881e19f9a1ce`
- 被验证文件 blob：`model/option.go = edf1b36458245ec5224fd7f8e945d2980cbe565d`、
  `controller/pricing_test.go = a6eb3376fa069cf63d3b6d4cb26aa3dd85150185`、
  新增 `model/option_test.go = fcdbc9d1be86f9c6e9b074833a74a68f0f4adc99`

## 结论

**接受（Accept）**。修复行为正确且回归有效：新增用例在修复前**失败**（旧代码吞错，接口谎报 `success:true`）、修复后全绿；
错误传播、失败不调 `updateOptionMap`、错误只格式化 key 三项均核实；`UpdateOptionsBulk` 与无关函数一行未改；
门禁与全仓测试全绿。实现者「4 个生产调用点」的**表述不精确**（实际 5 处调用表达式 / 4 个位置），
但其证据表本身列全了 5 处，且 5 处全部处理返回值 —— 属计数措辞问题，不构成打回理由。

## 一、修复前后对照（关键证据，无需 shim）

方法：`git clone /root/new-api-own /tmp/verify-option`（改动未提交 → 克隆即修复前代码），
拷入工作区的 `model/option_test.go` 与新版 `controller/pricing_test.go` 后运行。
本轮新旧测试引用的符号在 HEAD 全部存在，**没有使用任何 shim**；**全程未在真实工作区做 checkout/stash/restore**。

| 用例 | 修复前（克隆） | 修复后（真实工作区） |
|---|---|---|
| TestUpdateOptionPropagatesSaveErrorAndKeepsMemory | **FAIL**：option_test.go:80 期望 error，实际 nil | PASS |
| TestUpdateOptionPropagatesCreateError | **FAIL**：option_test.go:100 期望 error，实际 nil | PASS |
| TestUpdateOptionSuccessKeepsDBAndMemoryInSync | PASS（正常路径本来一致） | PASS |
| TestResetModelRatioWriteFailureReturnsFailure | **FAIL**：响应体为 `{"message":"重置模型倍率成功","success":true}`，不含 `"success":false` | PASS |

修复前的控制器用例失败信息直接复现了历史症状：写库失败（触发器 `RAISE(ABORT)`）时接口仍返回成功，
这正是上一轮「重置定价静默成功」的来源；修复后同一场景返回 `success:false` 且带 message，同时内存倍率保持旧值。

## 二、`model/option.go` 修改逐项核对

变更仅 3 个 hunk（`git diff -U0` 计数 = 3）：新增 `fmt` 导入 + `FirstOrCreate` 错误检查 + `Save` 错误检查。

| 核对项 | 代码事实 | 判定 |
|---|---|---|
| 错误真的传播 | `if err := DB.FirstOrCreate(...).Error; err != nil { return fmt.Errorf("cannot create option %q: %w", key, err) }`；`if err := DB.Save(&option).Error; err != nil { return fmt.Errorf("cannot save option %q: %w", key, err) }`；两处均直接 return，不再落到 `updateOptionMap` | 通过 |
| 失败路径不调 `updateOptionMap`（内存不变） | 两个 `return` 都在 `return updateOptionMap(key, value)` 之前；`common.OptionMap` / 各 setting 内存表只由 `updateOptionMap` 写入。用例断言：Save 失败后内存仍为 `old-value`、Create 失败后内存键不存在，DB 也保持旧值 | 通过 |
| 错误只格式化 key，从不格式化 value | 全文件仅 2 处 `fmt.Errorf`（:230、:237），格式串分别是 `"cannot create option %q: %w"` 与 `"cannot save option %q: %w"`，唯一参数是 `key` 与包裹错误；`value` 从未进入格式串。用例另有 `NotContains(err, "new-value")` 断言 | 通过 |
| 成功后仍同步内存 | 成功路径 `return updateOptionMap(key, value)` 未变；正常路径用例断言两次写入后 DB 与内存一致 | 通过 |
| 代码风格 | `gofmt -l` 对 `model/option.go`、`model/option_test.go`、`controller/pricing_test.go` 均无输出 | 通过 |

## 三、调用点审查（独立复核）

`grep -rn "UpdateOption(" --include=*.go .` 并排除测试后，**生产调用表达式共 5 处、分布 4 个位置**：

| # | 位置 | 返回值处理 | 新错误对用户的影响 |
|---|---|---|---|
| 1 | `controller/option.go:366`（管理员改配置） | `if err != nil { common.ApiError(c, err); return }` | 返回 `success:false`；且因提前 return **不写** `option.update` 审计（比原先「失败也记审计」更正确） |
| 2 | `controller/pricing.go:105`（重置定价） | 循环内 `if err := ...; err != nil { success:false; return }` | 写库失败如实报错（本修复的直接目标） |
| 3 | `controller/setup.go:129`（初始化 SelfUseModeEnabled） | `err = ...; if err != nil { success:false; return }` | 失败提前返回，不会置 `constant.Setup = true`、不写 Setup 记录，可重试 |
| 4 | `controller/setup.go:138`（初始化 DemoSiteEnabled） | 同上 | 同上 |
| 5 | `controller/payment_compliance.go:62`（支付合规 5 键循环） | `if err := ...; err != nil { common.ApiError(c, err); return }` | 原先 DB 失败仍报 confirmed；现在如实失败 |

- **无** `_ =` 忽略调用、**无**裸调用（grep 全量标识符核对：仅函数定义、上述 5 处、以及 `controller.UpdateOption` gin handler
  与路由 `router/api-router.go:200` 注册——后者与 `model.UpdateOption` 同名不同物）。
- 无任何生产调用点位于 `main.go` / `router/` 启动路径，故不存在「DB 未就绪即调用」的新增风险。
- 实现者证据表（`evidence-updateoption.md` §2）实际列了 5 行（#1–#5），但正文写「4 个生产调用点」——
  按位置数是 4（option / pricing / setup / payment_compliance），按调用表达式是 5。**审计内容完整，仅计数措辞不精确。**

## 四、范围控制核对

| 核对项 | 证据 | 判定 |
|---|---|---|
| `UpdateOptionsBulk` 一行未改 | `git diff -U0 -- model/option.go` 仅 3 个 hunk，均不在该函数内；HEAD 与工作区该函数体 `diff` 为空（BULK_IDENTICAL） | 通过 |
| 无顺手改无关函数 | `git diff --name-only` 仅 `controller/pricing_test.go`、`model/option.go`；工作区未跟踪仅 `model/option_test.go` 与任务目录 | 通过 |
| 新增测试文件本身无越界 | `model/option_test.go` 只新增 3 个用例 + 2 个 helper，用内存 SQLite 并保存/恢复 `DB`、`common.OptionMap`、数据库类型（`t.Cleanup`），不写外部资源 | 通过 |

## 五、门禁复跑（真实工作区，Go 全部 `GOPROXY=off`）

| 门禁 | 命令 | 结果 |
|---|---|---|
| Go 构建 | `go build ./...` | rc=0 |
| Go 静态检查 | `go vet ./...` | rc=0，无输出 |
| 目标包测试 | `go test ./model/... ./controller/... -count=1` | ok model 6.450s / ok controller 0.441s |
| 全仓测试 | `go test ./... -count=1` | rc=0，无任何 `FAIL` 行（跑两遍结果一致） |
| 目标用例明细 | `go test ./model/ -run TestUpdateOption -v`、`go test ./controller/ -run TestResetModelRatioWriteFailureReturnsFailure -v` | 3 + 1 全 PASS |
| 格式 | `gofmt -l` 三个相关文件 | 无输出 |

## 六、残留风险评估

### 6.1 `%w` 透传驱动原文是否可能带出配置值/绑定参数

结论：**本仓库自身代码不会格式化 value，透传的驱动原文在 SQLite/PostgreSQL 下不含绑定值；MySQL 有一个理论例外，风险低且暴露面仅管理员/setup。**

- 实测（/tmp 副本临时探针，不改仓库）：
  - 触发器打断 UPDATE：`cannot save option "ProbeKey": constraint failed: forced option write failure (1811)`
  - 表缺失：`cannot create option "ProbeKey": SQL logic error: no such table: options (1)`
  - 两种文本都含 key、**不含** value。
- 错误最终去向：`controller/option.go` 与 `payment_compliance.go` 经 `common.ApiError` 原样写进 `message`；
  `pricing.go` / `setup.go` 同样把 `err.Error()` 拼进 `message`。这些接口都是管理员/setup 面，不是终端用户面。
- 唯一值得记录的理论例外：MySQL 错误 1366（`Incorrect string value: ... for column ...`）会把**非法字符处的值片段**写进错误文本。
  触发条件苛刻（value 含当前连接字符集不可编码的字节），且配置值多为 ASCII；概率很低。
- 可选的进一步加固（非本轮要求）：服务端记录原始错误、对外返回通用文案 —— 但当前实现的方向（只带 key）已满足 R2「不泄露值」的诉求。

### 6.2 「DB 成功但 `updateOptionMap` 失败」的分叉是否属本轮范围外

结论：**确实属本轮范围外，且是修复前后一致的既有行为。**

- 现实现的保证是单向的：DB 失败 → 不同步内存（本轮修复）。反方向（DB 已写入、随后 `updateOptionMap` 解析失败，例如
  倍率类键收到非法 JSON）仍会让 DB 新值 + 内存旧值分叉，同时 `UpdateOption` 返回 error。
- 触发条件：仅当 `updateOptionMap` 内部返回解析错误（`ratio_setting.UpdateXxxByJSONString` 对非法 JSON 报错）；
  本任务涉及的两个调用场景（重置定价写的都是内部 `Marshal` 产物；管理员改配置有前置校验）实际很难触发。
- 若要彻底消除，需要在写库前先校验（或写库失败回滚 + 内存回滚），属另一项设计改动（PRD 未列入），建议单开任务。

### 6.3 额外观察（不阻塞）

1. `controller/setup.go:125-127` 在调用 `model.UpdateOption` **之前**就把 `operation_setting.SelfUseModeEnabled/DemoSiteEnabled`
   写进内存；若 DB 写入失败，接口现在会如实返回失败，但内存已被调用方改成新值（DB 仍旧值）。
   这是调用方既有的「先改内存后落库」写法，被本次错误传播暴露出来，不属底座修复范围；建议在 setup 流程里调整顺序（后置内存赋值）。
2. `controller/payment_compliance.go` 的 5 键循环非事务，中途失败时前面的键可能已写入（修复前也存在）；现在至少会如实报失败。
   若要原子化可改用已有的 `model.UpdateOptionsBulk`。

## 七、不确定项

1. 未在真实 MySQL/PostgreSQL 下复现写库失败（禁外部主机、本机只有测试用 SQLite）；结论基于驱动常见错误文本与代码路径推理。
2. 未验证「DB 成功后 `updateOptionMap` 失败」的实际触发（属范围外，且需要构造非法 JSON 的值）。

## 八、复跑命令清单

```bash
# 修复前后对照（Go）：clone 即修复前代码
rm -rf /tmp/verify-option && git clone -q /root/new-api-own /tmp/verify-option
cp /root/new-api-own/model/option_test.go /tmp/verify-option/model/
cp /root/new-api-own/controller/pricing_test.go /tmp/verify-option/controller/
cd /tmp/verify-option && GOPROXY=off go test ./model/ -run TestUpdateOption -count=1 -v
GOPROXY=off go test ./controller/ -run TestResetModelRatioWriteFailureReturnsFailure -count=1 -v

# 真实工作区
cd /root/new-api-own && GOPROXY=off go build ./... && GOPROXY=off go vet ./...
GOPROXY=off go test ./model/... ./controller/... -count=1
GOPROXY=off go test ./... -count=1
go test ./model/ -run TestUpdateOption -count=1 -v
go test ./controller/ -run TestResetModelRatioWriteFailureReturnsFailure -count=1 -v
```
