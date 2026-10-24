# 后端证据：重置接口恢复整份定价默认（含计费模式与表达式）

- 任务：Trellis `10-24-diagnose-deepseek-pricing-row-mismatch`（方案 C）；共享任务板 `task-26`
- 执行者：worker-script；时间：2026-10-24
- 写作用域：`controller/pricing.go`、`setting/ratio_setting/*.go`（仅新增导出 getter）、
  `setting/billing_setting/tiered_billing.go`（仅新增导出 getter）、`controller/pricing_test.go` + 本文件。
- 未改 `web/`；未提交、未推送、未发版；无任何外部主机访问（测试全用内存 SQLite）。

## 1. 根因与语义

`ResetModelRatio`（原 `controller/pricing.go:79-101`）只推进 `ModelRatio` 一个键，
但面板按钮的语义是「模型价格重置」，而定价由多张表共同决定。于是
`billing_setting.billing_mode` / `billing_setting.billing_expr` 永远不进重置范围：
实例里 `deepseek-v4.1-flash` 一旦缺表达式，重置后仍是「按 Token / 仅基础」，
与 `deepseek-flash` 的阶梯表达式不一致（用户生产实例即此状态）。

修复后该接口恢复**整份定价默认**：三张倍率表 + 价格表 + 图像/音频表 + 计费模式 + 计费表达式。

## 2. 改动清单

1. **`controller/pricing.go`**：`ResetModelRatio` 由"只写 ModelRatio"改为遍历 10 个键恢复默认。
   - 默认值全部取自各包的**导出 getter**，未复制任何默认值字面量：
     `DefaultModelRatio2JSONString` / `DefaultCompletionRatio2JSONString` / `DefaultCacheRatio2JSONString` /
     `DefaultCreateCacheRatio2JSONString` / `DefaultModelPrice2JSONString` / `DefaultImageRatio2JSONString` /
     `DefaultAudioRatio2JSONString` / `DefaultAudioCompletionRatio2JSONString` /
     `DefaultBillingMode2JSONString` / `DefaultBillingExpr2JSONString`。
   - 每个键都走 `model.UpdateOption`：它写库（`options` 表）**并**经 `updateOptionMap`
     同步内存（ratio 类走 `ratio_setting.Update*ByJSONString`；`billing_setting.*` 走
     `handleConfigUpdate` → `config.UpdateConfigFromMap` 全量替换 map，并 `InvalidatePricingCache`）。
   - 任一步失败立即返回 `{"success":false,"message":<err>}` 并停止，不继续写后续键。
   - **接口路径、请求体、响应结构与成功文案全部保持兼容**（仍 `POST /api/option/rest_model_ratio`，
     仍 `success` + `message`，成功文案仍是「重置模型倍率成功」）。
   - **不动** `GroupRatio` / `TopupGroupRatio` / `GroupGroupRatio` / `UserUsableGroups` / `AutoGroups`
     与 `ExposeRatioEnabled`（不在按钮语义内）——它们根本不在上面 10 个键里。
2. **`setting/ratio_setting/model_ratio.go`**：新增 `DefaultCompletionRatio2JSONString`、
   `DefaultModelPrice2JSONString`、`DefaultImageRatio2JSONString`、`DefaultAudioRatio2JSONString`、
   `DefaultAudioCompletionRatio2JSONString`、`GetDefaultCompletionRatioMap`。
3. **`setting/ratio_setting/cache_ratio.go`**：新增 `DefaultCacheRatio2JSONString`、
   `DefaultCreateCacheRatio2JSONString`、`GetDefaultCacheRatioMap`。
4. **`setting/billing_setting/tiered_billing.go`**：新增 `DefaultBillingMode2JSONString` /
   `DefaultBillingExpr2JSONString`（内部共用一个 `marshalDefault` helper，避免重复 marshalling 分支）。
   两者都从 `defaultBillingSetting()` 取默认，**不复制表达式字面量**。
5. **`controller/pricing_test.go`**（新增）：三个用例，见下。

关键 diff 摘要：

```diff
 func ResetModelRatio(c *gin.Context) {
-	defaultStr := ratio_setting.DefaultModelRatio2JSONString()
-	err := model.UpdateOption("ModelRatio", defaultStr)
-	... 只写一个键 ...
+	pricingDefaults := map[string]string{
+		"ModelRatio":           ratio_setting.DefaultModelRatio2JSONString(),
+		"CompletionRatio":      ratio_setting.DefaultCompletionRatio2JSONString(),
+		"CacheRatio":           ratio_setting.DefaultCacheRatio2JSONString(),
+		"CreateCacheRatio":     ratio_setting.DefaultCreateCacheRatio2JSONString(),
+		"ModelPrice":           ratio_setting.DefaultModelPrice2JSONString(),
+		"ImageRatio":           ratio_setting.DefaultImageRatio2JSONString(),
+		"AudioRatio":           ratio_setting.DefaultAudioRatio2JSONString(),
+		"AudioCompletionRatio": ratio_setting.DefaultAudioCompletionRatio2JSONString(),
+		"billing_setting." + billing_setting.BillingModeField: billing_setting.DefaultBillingMode2JSONString(),
+		"billing_setting." + billing_setting.BillingExprField: billing_setting.DefaultBillingExpr2JSONString(),
+	}
+	for key, defaultStr := range pricingDefaults {
+		if err := model.UpdateOption(key, defaultStr); err != nil {
+			c.JSON(200, gin.H{"success": false, "message": err.Error()})
+			return
+		}
+	}
```

## 3. 测试与前后对照（先写能失败的用例）

新用例（`controller/pricing_test.go`，内存 SQLite + `IsMasterNode=true` 跑迁移建 `options` 表）：

1. `TestResetModelRatioRestoresFullPricingDefaults`（核心回归）：
   先制造实例漂移——`billing_mode` 只留 `deepseek-v4.1-flash: ratio`、`billing_expr` 该模型为空串、
   `ModelRatio` 掺入 `stale-model`；再调用重置。断言：
   - 计费模式与表达式**等于代码默认**，且 `deepseek-flash` 与 `deepseek-v4.1-flash` **逐字符相同**；
   - 三张倍率表两模型一致、等于代码默认、内存值与 DB 值一致、陈旧条目被清掉；
   - 分组哨兵 `GroupRatio`（含 `vip:7.5`、`sentinel_group`）与 `TopupGroupRatio`
     在 DB 与内存里**都未被改动**。
2. `TestResetModelRatioClearsEmptyDefaultTables`：默认是空表的 5 个键
   （`ModelPrice`/`ImageRatio`/`AudioRatio`/`AudioCompletionRatio`/`CreateCacheRatio`）
   重置后必须真的变回 `{}`，而不是把实例残留条目留在表内。
3. `TestResetModelRatioWriteFailureIsSilentlySuccessfulInModel`：记录已知上游限制（见第 5 节）。

### 修复前后对照

方法：把 `controller/pricing.go` 临时还原为 git 原版（`git checkout --`），测试文件不变，跑同一组用例，
再恢复修复版本并重跑。

修复前（原版只写 `ModelRatio`）：

```console
$ go test ./controller/ -count=1 -run 'TestResetModelRatioRestoresFullPricingDefaults'
--- FAIL: TestResetModelRatioRestoresFullPricingDefaults (0.02s)
    pricing_test.go:138: Error: Not equal:
        expected: map[string]interface {}{"deepseek-flash":"tiered_expr", "deepseek-v4.1-flash":"tiered_expr"}
        actual  : map[string]interface {}{"deepseek-v4.1-flash":"ratio"}
        Messages: 计费模式必须回到代码默认
    pricing_test.go:139: Error: Not equal:
        expected: map[..."deepseek-v4.1-flash":"(weekday(\"Asia/Shanghai\") >= 1 && ... ? tier(\"peak\", p*2 + c*8 + cr*0.04) : tier(\"offpeak\", p*1 + c*4 + cr*0.02)"]
        actual  : map[string]interface {}{"deepseek-v4.1-flash":""}
        Messages: 计费表达式必须回到代码默认
    pricing_test.go:141: Error: Not equal:
        expected: "tiered_expr"
        actual  : "ratio"
        Messages: drift 模型必须恢复为表达式计费（这正是本缺陷的症状）
FAIL
```

> 这三条失败**逐字对应用户实例的症状**：模式 `ratio`、表达式空、与 `deepseek-flash` 不一致。
> 另 `TestResetModelRatioClearsEmptyDefaultTables` 修复前亦失败（`ModelPrice` 等 5 个键保持实例残留值，
> 且「选项 CompletionRatio 应已落库」直接报不存在——证明旧实现根本没写这些键）。

修复后：

```console
$ go test ./controller/ -count=1 -run 'TestResetModelRatio' -v
=== RUN   TestResetModelRatioRestoresFullPricingDefaults
--- PASS: TestResetModelRatioRestoresFullPricingDefaults (0.02s)
=== RUN   TestResetModelRatioClearsEmptyDefaultTables
--- PASS: TestResetModelRatioClearsEmptyDefaultTables (0.01s)
=== RUN   TestResetModelRatioWriteFailureIsSilentlySuccessfulInModel
--- PASS: TestResetModelRatioWriteFailureIsSilentlySuccessfulInModel (0.01s)
ok  	github.com/QuantumNous/new-api/controller	0.059s
```

## 4. 门禁输出（全部实际执行）

```console
$ go build ./...
（无输出，exit 0）

$ go vet ./...
（无输出，exit 0）

$ go test ./service/... ./controller/... ./setting/... -count=1
ok  	github.com/QuantumNous/new-api/service	0.468s
ok  	github.com/QuantumNous/new-api/service/authz	0.032s
?   	github.com/QuantumNous/new-api/service/passkey	[no test files]
ok  	github.com/QuantumNous/new-api/controller	0.442s
ok  	github.com/QuantumNous/new-api/setting	0.012s
ok  	github.com/QuantumNous/new-api/setting/billing_setting	0.015s
ok  	github.com/QuantumNous/new-api/setting/config	0.014s
?   	github.com/QuantumNous/new-api/setting/console_setting	[no test files]
ok  	github.com/QuantumNous/new-api/setting/model_setting	0.014s
ok  	github.com/QuantumNous/new-api/setting/operation_setting	0.017s
?   	github.com/QuantumNous/new-api/setting/perf_metrics_setting	[no test files]
?   	github.com/QuantumNous/new-api/setting/performance_setting	[no test files]
ok  	github.com/QuantumNous/new-api/setting/ratio_setting	0.013s
?   	github.com/QuantumNous/new-api/setting/reasoning	[no test files]
ok  	github.com/QuantumNous/new-api/setting/system_setting	0.012s

$ go test ./... -count=1        # 全仓回归
（除 ok/? 外无输出 = 全绿）
```

## 5. 必须报给 Lead 的两点发现

### 发现 1：`model.UpdateOption` 吞掉 `DB.Save` 错误（上游静默失败，未修）

任务要求"任一步失败要如实报错"。控制器侧的错误分支已就位（任一步返回 error 即返回 `success:false`
并停止），但实测**无法从 `model.UpdateOption` 触发**——它在 `model/option.go:232` 调 `DB.Save(&option)`
却丢弃了返回值，只 `return updateOptionMap(key, value)`。实测证据：

```console
direct DB.Save err = constraint failed: forced option write failure (1811)   # 触发器让 UPDATE 必失败
UpdateOption err   = <nil>                                                    # 错误被吞掉
```

影响：真实写库失败（磁盘只读、DB 故障）时，重置接口仍会回 `{"success":true}`，而实例可能半新半旧。
**我没有改 `model/option.go`**（不在本任务写作用域，且它是全站共用底座，改动面大）。
已用 `TestResetModelRatioWriteFailureIsSilentlySuccessfulInModel` 把该行为钉住：
一旦有人修好上游静默失败，该用例会失败并提醒同步复核控制器语义。是否要顺手修上游，请 Lead 定。

### 发现 2：成功文案未改（有意保持兼容）

改后接口做的是「整份定价重置」，文案仍是「重置模型倍率成功」（`controller/pricing.go:118`），
与 `i18n/locales/*.yaml` 的 `model.reset_success` 一致。
我**刻意没改**：该文案属对外响应一部分，任务要求"响应结构保持兼容"，且前端成功提示走自己的 i18n key
（`ratio-settings-card.tsx` 的 `Model pricing was restored to the default templates...`），
不读后端 message；后端 message 只在失败分支被展示。
若要同步改成「重置模型价格成功」，是一行改动，请 Lead 定（改的话建议同步 `i18n/locales/*.yaml` 的 key 文案）。

## 6. 风险与未完成

- **未做真实实例验证**：红线未授权访问外部主机/实例；本任务全程仅内存 SQLite。生产上建议在实例面板点一次
  重置后核对 `deepseek-v4.1-flash` 的计费模式是否变为表达式（属需授权的实例操作）。
- 重置是**逐键**写入（非单事务）：若中途失败，前面的键已生效。这是"如实报错 + 不静默"的取舍；
  做成单事务需改 `model` 层（超出写作用域）。
- 新增 getter 只读默认变量，返回的是包内变量本身（与既有 `GetDefaultModelRatioMap` 同风格），未做深拷贝。
- 新测试用 `IsMasterNode=true` 跑迁移建表，测试结束后恢复原值；与同包既有测试（`model_list_test.go`）
  的隔离手法一致。
- 未提交：改动留在工作区（`controller/pricing.go`、`setting/` 两个文件、新增 `controller/pricing_test.go`）。

## 7. 建议下一步

1. Lead/verifier 复核 `git diff -- controller/pricing.go setting/` 与第 5 节两个待决项。
2. 若接受：按闸门提交（锚点 `[task:diagnose-deepseek-pricing-row-mismatch]`）。
3. 是否顺手修 `model.UpdateOption` 的静默失败（发现 1）——建议单开任务，避免把全站底座改动混进本次修复。
4. 授权后可在实例上点一次重置做端到端确认（记录重置前后 `billing_setting.*`）。
