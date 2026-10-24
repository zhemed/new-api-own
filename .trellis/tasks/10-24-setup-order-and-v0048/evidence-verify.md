# setup.go 顺序与原子写入修复：独立验证证据（verifier）

- 任务：共享任务板 `task-34`（Trellis 任务 `10-24-setup-order-and-v0048` 的独立验证）
- 验证者：verifier（只读验证；未改任何仓库文件）
- 验证时间：2026-10-24（UTC 04:10 起）
- 基线：`HEAD = 2867700`；已跟踪改动仅 1 文件（15 增 19 删）+ 1 个新增未跟踪测试
- 基线指纹：`git diff | sha256sum = 9ddcad6b3e9d1289f6d4069ffb624fa3280ccdc84f14ef30e40ebe8e7e5b10f2`
- 被验证文件 blob：`controller/setup.go = 1dd20bab1f9914e9b07cbac3548e63b75d9018c0`、
  新增 `controller/setup_test.go = 60cf20d0a94fc1ef08166c3d7a4dd99f2e606303`

## 结论

**接受（Accept）**。两处顺序缺陷与原子写入均修复到位：新增用例在修复前**失败且失败信息逐条对应两处缺陷**、修复后 4/4 全绿；
`updateOptionMap` 确实覆盖两个键（删除手工赋值不会造成内存失同步）；`UpdateOptionsBulk` 真事务；
`constant.Setup` 只在 setups 记录落库成功后置位；范围仅 `controller/setup.go`；测试全局状态复位完整。
**唯一需要 Lead 知悉的异常**：全仓 `go test ./...` 存在**既有 flaky**（model 包日志预算用例，与本次改动无关），
6 次全仓运行中 2 次在并发负载下翻红，单独运行与复跑均绿（详见第五节）。

## 一、修复前后对照（关键证据）

方法：`git clone /root/new-api-own /tmp/verify-setup`（改动未提交 → 克隆即修复前代码），
拷入工作区新增的 `controller/setup_test.go` 后运行；本轮无需任何 shim。
**全程未在真实工作区做 checkout/stash/restore。**

| 用例 | 修复前（克隆） | 修复后（真实工作区） |
|---|---|---|
| TestPostSetupWriteFailureKeepsMemoryAndDBUntouched | **FAIL**（对应缺陷 ①） | PASS |
| TestPostSetupWritesOperationModesAtomically | **FAIL**（两键非原子，半应用） | PASS |
| TestPostSetupCreateFailureDoesNotMarkInitialized | **FAIL**（对应缺陷 ②） | PASS |
| TestPostSetupSuccessSyncsMemoryAndDB | PASS（正常路径行为不变） | PASS |

修复前失败信息与两处缺陷逐条对应：

```
setup_test.go:115  Should be false  写库失败后 SelfUseModeEnabled 必须保持旧值，不得在落库前就被改
setup_test.go:117  Should be false  写库失败后 DemoSiteEnabled 必须保持旧值，不得在落库前就被改
                 → 缺陷 ①：旧代码在落库前手工赋值 operation_setting，写库失败后内存已被污染
setup_test.go:170  Should be zero, but was 1  原子写入：另一个键失败时 SelfUseModeEnabled 也不得落库
setup_test.go:174/175 两个内存开关同样被改
                 → 两次独立 UpdateOption 造成半应用（第一个键已落库）
setup_test.go:194  Should be false  Create 失败时不得把实例标记为已初始化
setup_test.go:200  retry response = {"message":"系统已经初始化完成","success":false}
                 → 缺陷 ②：constant.Setup 在 DB.Create 之前置位，失败后重试被永久挡住
```

## 二、代码核对

| 核对项 | 证据 | 判定 |
|---|---|---|
| `updateOptionMap` 覆盖两个键 | 两键均以 `Enabled` 结尾，命中 `model/option.go:313` 的分支条件 `strings.HasSuffix(key, "Enabled")`；分支内 `:376-378` `DemoSiteEnabled`、`:378-379` `SelfUseModeEnabled` 把 bool 写回 `operation_setting` | 通过 |
| 删除手工赋值后成功路径仍有内存同步 | 全仓非测试代码中这两个全局变量的**唯一赋值点**就是 `model/option.go:377/379`（`grep` 全仓确认，`setting/operation_setting/operation_setting.go:5-6` 仅为声明）；`UpdateOptionsBulk` 成功后逐键调用 `updateOptionMap`，故成功路径由它同步；用例 `TestPostSetupSuccessSyncsMemoryAndDB` 断言内存与 DB 均为请求值 | 通过 |
| `UpdateOptionsBulk` 真事务 | 函数体：先统一 `validateOptionValue`，再 `DB.Transaction` 内逐键 `FirstOrCreate` + `Save` 且都检查 `.Error`（任一失败整体回滚），事务提交后才循环 `updateOptionMap`；失败直接 return，不碰内存 | 通过 |
| 原子性有行为证据 | 用例只让第二个键（`DemoSiteEnabled`）INSERT 失败 → 修复前第一个键已落库（`Should be zero, but was 1`），修复后两键均不落库 | 通过 |
| `constant.Setup` 在 Create 成功之后 | 代码顺序：`UpdateOptionsBulk` → `DB.Create(&setup)` 失败则 return → 成功后才 `constant.Setup = true`；失败分支 `constant.Setup` 保持 false | 通过 |
| 失败后允许重试 / 成功后重复 setup 仍被拒 | `PostSetup` 开头的 `if constant.Setup { 系统已经初始化完成 }` 未改动：成功置位后重复请求仍被拒；失败未置位 → 重试可达；用例先制造 Create 失败再撤掉触发器重试，断言重试成功且只有 1 条 setups 记录 | 通过 |
| 启动期标记不受影响 | `model/main.go:CheckSetup` 依据 DB 中 setups 记录设置 `constant.Setup`，本次未改；只覆盖「初始化请求内」的顺序 | 通过 |
| 旧失败文案无残留依赖 | 全仓 grep「保存自用模式设置失败 / 保存演示站点模式设置失败」：仅被替换掉的位置，无其他引用；新文案为「保存运营模式设置失败」 | 通过 |

## 三、边界与测试卫生

| 核对项 | 结果 |
|---|---|
| 改动范围 | `git diff --name-only` 只有 `controller/setup.go`；`-U0` 共 5 个 hunk（删 import / 删手工赋值+两次 UpdateOption / 改失败文案 / 删提前置位 / Create 后置位），无顺手改动 |
| 未跟踪文件 | 仅 `controller/setup_test.go` 与任务目录 |
| 新测试全局状态复位 | `t.Cleanup` 复位 6 项：`model.DB`、`common.MainDatabaseType`、`common.OptionMap`、`operation_setting.SelfUseModeEnabled`、`operation_setting.DemoSiteEnabled`、`constant.Setup`；内存 SQLite 隔离，root 用户预置在隔离库内 |
| 无跨用例泄漏 | `go test ./controller/ -run TestPostSetup -count=3` 通过；`go test ./controller/ -count=2` 通过 |
| 格式 | `gofmt -l controller/setup.go controller/setup_test.go` 无输出 |

## 四、门禁复跑（真实工作区，Go 全部 `GOPROXY=off`）

| 门禁 | 命令 | 结果 |
|---|---|---|
| Go 构建 | `go build ./...` | rc=0 |
| Go 静态检查 | `go vet ./...` | rc=0，无输出 |
| 目标包测试 | `go test ./controller/... ./model/... -count=1` | ok controller 0.449s / ok model 5.076s |
| 全仓测试 | `go test ./... -count=1` | 最终一次 rc=0；但存在既有 flaky（见下） |
| 目标用例明细 | `go test ./controller/ -run TestPostSetup -count=1 -v` | 4/4 PASS |

### 全仓测试的既有 flaky（与本次改动无关，必须如实记录）

- 现象：多次运行全仓测试时，`model` 包偶发两个用例失败 ——
  `TestLogPayloadTrimIsSingleFlight`（`Should be true`，即单飞标志被占用）与
  `TestLogPayloadBudgetDisabledByDefault`（`Should be zero, but was 8526`，即字节计数器残留）。
- 频率：6 次全仓运行中 2 次翻红，且**都发生在我同时跑其它测试任务的并发负载下**；负载消失后单跑 rc=0。
- 隔离验证：`go test ./model/ -run TestLogPayloadTrimIsSingleFlight -count=5` 全绿；
  `go test ./model/ -count=10` 全绿；`go test ./model/ -shuffle=on -count=5` 全绿。
- 机制（读代码确认）：`TestLogPayloadBudgetTrimsOldestRowsToBudget` 等用例写日志会触发
  `noteLogPayloadBytes` → `triggerLogPayloadTrim` → `gopool.Go` 后台裁剪/校准 goroutine；
  这些 goroutine 可能活过本用例，持有 `logByteTrimRunning` 单飞标志或延后更新 `logPayloadBytes`，
  让紧随其后的用例看到脏全局状态。
- 与本次改动的关联：**无**。`git diff --name-only` 只有 `controller/setup.go`，model 包与 HEAD 逐字节相同；
  这些用例由既有提交 `4c1977b feat(logs): 日志字节预算` 引入，早于本任务。
- 结论：属 model 包既有的测试隔离缺陷（时间敏感 flaky），不是本次修复引入；建议单开任务让后台裁剪在用例结束时收敛。

## 五、风险与不确定项

1. （非阻断，环境）`/tmp/verify-setup` 克隆里根包 `go test` 会 `[setup failed]`：`main.go` 用 `go:embed web/dist`，
   而 `web/dist` 是 gitignored 构建产物，克隆里不存在。这只影响克隆的环境，不影响 controller/model 包验证
   （两者在克隆里正常构建运行，前置失败用例已取得）。
2. （非阻断）`UpdateOptionsBulk` 的保证是「DB 事务失败 → 不同步内存」；反方向（事务提交成功、随后 `updateOptionMap` 报错）
   仍是部分内存同步的既有分叉。但对本次两个 boolean 键，`updateOptionMap` 的对应分支不产生 error，
   因此本场景不触发；与上一轮 `UpdateOption` 修复同属既有边界。
3. （非阻断）setup 失败文案由两条合并为「保存运营模式设置失败」，与原子写入语义一致；全仓无按旧文案匹配的代码或测试。
4. （不确定项）未在真实 MySQL/PostgreSQL 上验证事务回滚行为（本机只有内存 SQLite），失败路径用触发器构造；
   事务语义本身由 `UpdateOptionsBulk` 既有实现与 GORM 保证。

## 六、复跑命令清单

```bash
# 修复前后对照（Go）：clone 即修复前代码
rm -rf /tmp/verify-setup && git clone -q /root/new-api-own /tmp/verify-setup
cp /root/new-api-own/controller/setup_test.go /tmp/verify-setup/controller/
cd /tmp/verify-setup && GOPROXY=off go test ./controller/ -run TestPostSetup -count=1 -v

# 真实工作区
cd /root/new-api-own && GOPROXY=off go build ./... && GOPROXY=off go vet ./...
GOPROXY=off go test ./controller/... ./model/... -count=1
GOPROXY=off go test ./... -count=1
go test ./controller/ -run TestPostSetup -count=1 -v
go test ./controller/ -run TestPostSetup -count=3
```
