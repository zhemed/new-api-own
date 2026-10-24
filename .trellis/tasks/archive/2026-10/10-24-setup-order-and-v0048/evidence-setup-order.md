# 证据：`controller/setup.go` 落库前改内存的顺序问题 + 原子写入

- 任务：Trellis `10-24-setup-order-and-v0048`；共享任务板 `task-33`
- 执行者：worker-script；时间：2026-10-24
- 写作用域：`controller/setup.go`、`controller/setup_test.go`（新增）、本文件。
- **未发版、未升级实例、未提交、未推送**：PRD 的 R5（v0.0.48 发布）与 R6（实例升级）不在本任务范围，
  由 Lead 在验证通过后统一执行（task-33 只承载代码侧 R1–R4）。
- 未动 `web/`；无外部主机访问。

## 1. 根因

修复前 `controller/setup.go:124-145`：

```go
// Set operation modes
operation_setting.SelfUseModeEnabled = req.SelfUseModeEnabled   // ← 先改内存
operation_setting.DemoSiteEnabled = req.DemoSiteEnabled         // ← 先改内存

err = model.UpdateOption("SelfUseModeEnabled", ...)   // 再落库；失败才 return
...
err = model.UpdateOption("DemoSiteEnabled", ...)
```

上一轮底座修复后 `model.UpdateOption` 会如实返回 DB 错误，于是失败路径变成：
**接口报失败、但内存两个开关已经被改成请求值** → DB 与内存分叉。

而这两个键的内存同步**本来就该由底座负责**：已核对 `model/option.go:376-379`
的 `updateOptionMap` 确实包含

```go
case "DemoSiteEnabled":
	operation_setting.DemoSiteEnabled = boolValue
case "SelfUseModeEnabled":
	operation_setting.SelfUseModeEnabled = boolValue
```

且该 switch 位于 `strings.HasSuffix(key, "Enabled")` 分支内，两个键名都以 `Enabled` 结尾，
因此**落库成功后必经此分支**。setup.go 里的手工赋值既冗余、又制造了顺序问题。

## 2. 改动（diff）

```diff
-	// Set operation modes
-	operation_setting.SelfUseModeEnabled = req.SelfUseModeEnabled
-	operation_setting.DemoSiteEnabled = req.DemoSiteEnabled
-
-	// Save operation modes to database for persistence
-	err = model.UpdateOption("SelfUseModeEnabled", boolToString(req.SelfUseModeEnabled))
-	if err != nil { ... "保存自用模式设置失败: " ... }
-	err = model.UpdateOption("DemoSiteEnabled", boolToString(req.DemoSiteEnabled))
-	if err != nil { ... "保存演示站点模式设置失败: " ... }
+	// 1) 不得在落库前手工改内存：内存同步由 UpdateOptionsBulk 成功后的 updateOptionMap 负责
+	// 2) 两个键原子写入：避免第一个成功、第二个失败的半应用状态
+	err = model.UpdateOptionsBulk(map[string]string{
+		"SelfUseModeEnabled": boolToString(req.SelfUseModeEnabled),
+		"DemoSiteEnabled":    boolToString(req.DemoSiteEnabled),
+	})
+	if err != nil {
+		c.JSON(200, gin.H{
+			"success": false,
+			"message": "保存运营模式设置失败: " + err.Error(),
+		})
+		return
+	}
```

同时删除因此不再使用的 `setting/operation_setting` import。

- **R1（主修复）**：删掉落库前的两处手工内存赋值。
- **R2（采纳）**：两次 `model.UpdateOption` 换成 `model.UpdateOptionsBulk`，两键原子写入。
- 错误文案：由两条"分键"文案合并为「保存运营模式设置失败: <err>」。语义不变（仍明确是写库失败、
  仍透出底层错误、不吞错）；两键现在是一起提交的，再区分"自用/演示"已无意义。
  前端不匹配该文案（只展示 `data.message`），无兼容风险（已 grep `web/src/features/setup/`）。

### R2 取舍理由（采纳而非只做 R1）

采纳，因为收益明显大于风险：

- **收益**：`UpdateOptionsBulk` 内部是 `DB.Transaction`，任一键写失败会整体回滚。否则旧实现会出现
  "SelfUseModeEnabled 已落库、DemoSiteEnabled 失败"的半应用状态（已实测复现，见 §3.2）。
- **风险低**：`UpdateOptionsBulk` 是同文件既有的正确写法（事务 + 校验 + 成功后统一 `updateOptionMap`），
  本次不修改它；这两个键的校验函数对它们返回 nil；`updateOptionMap` 的 bool 分支不会失败。
- **可观测变化**：失败文案合并（见上），属可接受的最小降级。

## 3. 测试（先写失败用例）

新增 `controller/setup_test.go`（内存 SQLite：options/users/setups 三表；预建 root 用户以跳过建号分支；
保存/恢复 `model.DB`、`common.OptionMap`、`operation_setting.*`、`constant.Setup` 全部全局状态）。

1. `TestPostSetupWriteFailureKeepsMemoryAndDBUntouched`（核心）：`BEFORE INSERT ON options` 触发器让写库必失败
   → 调用 `PostSetup` → 断言 ① 响应 `success:false`；② **内存两个开关保持旧值 `false`**；③ DB 无这两行；
   ④ `constant.Setup` 仍为 `false`（未把实例标记为已初始化）。
2. `TestPostSetupWritesOperationModesAtomically`（R2 证据）：触发器只让 `DemoSiteEnabled` 的 INSERT 失败 →
   断言响应失败，且 **`SelfUseModeEnabled` 也不得落库**（原子回滚），内存两键均不变。
3. `TestPostSetupSuccessSyncsMemoryAndDB`（AC2）：成功路径行为不变——响应 `success:true`、内存为请求值、
   DB 为 `"true"`、`constant.Setup=true`、`setups` 表有记录。

### 3.1 修复前后对照（R1）

修复前：

```console
$ go test ./controller/ -count=1 -run 'TestPostSetup' -v
=== RUN   TestPostSetupWriteFailureKeepsMemoryAndDBUntouched
    Error: Should be false
    Messages: 写库失败后 SelfUseModeEnabled 必须保持旧值，不得在落库前就被改
    Error: Should be false
    Messages: 写库失败后 DemoSiteEnabled 必须保持旧值，不得在落库前就被改
--- FAIL: TestPostSetupWriteFailureKeepsMemoryAndDBUntouched (0.00s)
=== RUN   TestPostSetupSuccessSyncsMemoryAndDB
--- PASS: TestPostSetupSuccessSyncsMemoryAndDB (0.00s)
FAIL
```

修复后：

```console
=== RUN   TestPostSetupWriteFailureKeepsMemoryAndDBUntouched
--- PASS: TestPostSetupWriteFailureKeepsMemoryAndDBUntouched (0.00s)
=== RUN   TestPostSetupSuccessSyncsMemoryAndDB
--- PASS: TestPostSetupSuccessSyncsMemoryAndDB (0.00s)
=== RUN   TestPostSetupWritesOperationModesAtomically
--- PASS: TestPostSetupWritesOperationModesAtomically (0.00s)
ok  	github.com/QuantumNous/new-api/controller	0.023s
```

### 3.2 修复前后对照（R2 原子性）

把 `controller/setup.go` 临时还原为 git 原版（两次独立 `UpdateOption` + 手工改内存）后跑原子性用例：

```console
--- FAIL: TestPostSetupWritesOperationModesAtomically (0.00s)
    Error: Should be zero, but was 1
    Messages: 原子写入：另一个键失败时 SelfUseModeEnabled 也不得落库（不得半应用）
    Error: Should be false
    Error: Should be false
```

`SelfUseModeEnabled` 行数为 1 = 旧实现确实留下半应用状态；修复后该用例通过（两键都回滚为 0 行）。

## 4. 门禁输出（全部实际执行）

```console
$ go build ./...
（无输出，exit 0）

$ go vet ./...
（无输出，exit 0）

$ go test ./controller/... ./model/... -count=1
ok  	github.com/QuantumNous/new-api/controller	0.441s
ok  	github.com/QuantumNous/new-api/model	7.449s

$ go test ./... -count=1
（除 ok/? 外无输出 = 全仓全绿）
```

## 5. 影响面 / 回滚 / 残留

### 影响面

- 仅初始化接口 `POST /api/setup` 的失败路径与写入方式：
  - **失败路径**：以前会留下"内存已改 / 半应用"的脏状态，现在两键要么全成、要么全不成；
  - **成功路径行为不变**（AC2 有断言）。
- 对外可见差异仅两点：失败文案由两条分键文案合并为「保存运营模式设置失败」；两键改为原子写入。
- 无数据库结构/迁移变化，无 API 路径或请求/响应结构变化。

### 回滚

单文件 revert `controller/setup.go`（恢复两处手工赋值 + 两次 `UpdateOption`）即可；不牵动数据与其它模块。

### 残留（同函数内同类问题）→ 复核后已修，见 §7

首次提交时发现 `PostSetup` 末尾也有同类的"先改内存后落库"（`constant.Setup = true` 在
`model.DB.Create(&setup)` 之前），当时未擅自改动、只做了标记。**Lead 复核后要求一并修复，已在 §6 完成。**

## 6. constant.Setup 顺序（复核后追加）

### 6.1 问题

首次提交时已标记：`PostSetup` 末尾把内存标记放在落库之前。

```go
constant.Setup = true                                  // ← 先改内存
setup := model.Setup{...}
err = model.DB.Create(&setup).Error                     // ← 再落库；失败则 return
```

`DB.Create(&setup)` 失败时：`constant.Setup` 已为 `true` → 用户重试初始化会得到
「系统已经初始化完成」，但 `setups` 表里没有记录（只有重启后才会因无记录回到未初始化）。
这是与运营模式写入**同一类**的"先改内存后落库"顺序问题。

### 6.2 修复

按 Lead 指定，把 `constant.Setup = true` 移到 `model.DB.Create(&setup)` **成功之后**；
失败分支保持原样（返回「系统初始化失败」且不标记已初始化）。

```diff
 	// Update setup status
-	constant.Setup = true
-
 	setup := model.Setup{
 		Version:       common.Version,
 		InitializedAt: time.Now().Unix(),
@@
 		})
 		return
 	}
+	// 只有 setups 记录落库成功后才标记内存"已初始化"：否则 Create 失败会让内存谎报已初始化，
+	// 用户重试被「系统已经初始化完成」挡住，而 DB 里并没有记录（与运营模式写入同类的顺序问题）。
+	constant.Setup = true
```

### 6.3 测试（先写失败用例）

新增 `TestPostSetupCreateFailureDoesNotMarkInitialized`：用
`BEFORE INSERT ON setups` 触发器让 `Create(&setup)` 必失败（options 写入正常成功，精确定位到该步），断言：

1. 接口返回 `success:false` 且文案含「系统初始化失败」；
2. `constant.Setup` 仍为 `false`；
3. `setups` 表零记录；
4. **重试不被挡**：删掉触发器后再次调用 `PostSetup`，不得出现「系统已经初始化完成」，
   重试应 `success:true`，且 `constant.Setup == true`、`setups` 恰有 1 条记录。

修复前（`constant.Setup` 仍在 Create 之前）：

```console
$ go test ./controller/ -count=1 -run 'TestPostSetupCreateFailureDoesNotMarkInitialized' -v
=== RUN   TestPostSetupCreateFailureDoesNotMarkInitialized
    Error: Should be false
    Messages: Create 失败时不得把实例标记为已初始化
    Error: "{\"message\":\"系统已经初始化完成\",\"success\":false}" should not contain "系统已经初始化完成"
    Messages: Create 失败后必须允许重试，不得因内存已标记而永久卡住：{"message":"系统已经初始化完成","success":false}
--- FAIL: TestPostSetupCreateFailureDoesNotMarkInitialized (0.00s)
```

> 两条失败正好对应两个症状：内存谎报已初始化 + 重试被永久挡住。

修复后（四个 setup 用例全部通过）：

```console
$ go test ./controller/ -count=1 -run 'TestPostSetup' -v
--- PASS: TestPostSetupWriteFailureKeepsMemoryAndDBUntouched (0.00s)
--- PASS: TestPostSetupSuccessSyncsMemoryAndDB (0.00s)
--- PASS: TestPostSetupWritesOperationModesAtomically (0.00s)
--- PASS: TestPostSetupCreateFailureDoesNotMarkInitialized (0.00s)
ok  	github.com/QuantumNous/new-api/controller	0.028s
```

### 6.4 门禁（追加轮次实际执行）

```console
$ go build ./...
（无输出，exit 0）

$ go vet ./...
（无输出，exit 0）

$ go test ./controller/... -count=1
ok  	github.com/QuantumNous/new-api/controller	0.444s

$ go test ./... -count=1
（除 ok/? 外无输出 = 全仓全绿）
```

### 6.5 影响面与回滚

- 成功路径行为不变（仍是成功后 `constant.Setup = true`）；变化只发生在 `Create(&setup)` 失败时：
  不再把实例标记为已初始化，用户可以正常重试。
- 回滚：单文件 revert `controller/setup.go` 即回到"先标记后落库"的旧行为。
- 本轮只改这一处（+ 对应测试与本节证据），未顺带动其它逻辑。

## 7. 建议下一步

1. Lead/verifier 复核 `git diff -- controller/setup.go`（两处顺序修复：运营模式写入 + `constant.Setup`）。
2. 代码侧接受后，由 Lead 执行 PRD 的 R5（`VERSION` → 0.0.48、tag、双 workflow、Release 四资产含 `.sig`）
   与 R6（实例升级到 v0.0.48 并核验），本任务未触碰发版与实例。
