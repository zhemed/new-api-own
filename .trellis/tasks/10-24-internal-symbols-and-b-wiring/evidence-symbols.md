# 29 个内部面导出符号清理证据（task-53 / PRD R1）

- Trellis 任务：`10-24-internal-symbols-and-b-wiring`（PRD 见同目录 `prd.md`，R1）
- 共享任务：task-53「清 29 个内部面导出符号（pkg/oauth/types/setting）」
- 执行者：worker-script2（接替前一位中断成员；B 堆 task-54 完成后接手本项）
- 日期：2026-10-24
- 边界：未提交、未推送、未发版、未联网、未装工具、未动 `web/`、`data/`、实例、`relaykit`

## 0. 结论

| 指标 | 起点（B 堆收尾后） | 本轮后 | 变化 |
|---|---|---|---|
| `deadcode -test ./...` 行数 | 81 | **53** | -28 |
| 原始 85 条清单剩余 | 81 | **53** | 本轮 -28（加上 B 堆 -4，原始 85 → 53） |
| 本轮处置符号 | - | **28 删 + 1 保留** | 29 = pkg 13 + oauth 2 + types 2 + setting 12 |
| `golangci-lint unused` | 1（`logKeyCol`） | **1（`logKeyCol`）** | 0 |
| 门禁 | 绿 | **绿**（build/vet/test/relaykit/gofmt） | - |

保留的 1 个：`setting/ratio_setting.GetModelRatioOrPrice` —— 全仓 Go 0 引用，但 `web/` 的 TSX 注释点名它（四类核对 ③ 不满足，见 §3 第 29 行与 §6.3）。删除其余 28 个后无失败、无回退。

## 1. 对象与计数

清单来源：`.trellis/tasks/archive/2026-10/10-24-panic-recovery-and-cleanup/evidence-symbols.md`（85 条表，A 76 / B 4 / C 5）。

| 包 | 清单数 | 已删 | 保留 |
|---|---|---|---|
| `pkg/*`（billingexpr 1、cachex 3、ionet 9） | 13 | 13 | 0 |
| `oauth/*` | 2 | 2 | 0 |
| `types/*` | 2 | 2 | 0 |
| `setting/*`（含 4 个子包） | 12 | 11 | 1 |
| **合计** | **29** | **28** | **1** |

## 2. 四类核对规则与执行命令

每个符号必须四类全满足才删（① 不可达、② 全仓 Go 无引用、③ 非 Go 位置无引用、④ 非反射/注册表/键名）：

```bash
# ① 可达性：本轮起点/终点的完整 deadcode 输出
/root/go/bin/deadcode -test ./...            # 起点 81 行、终点 53 行（见 §5）

# ② 全仓 Go 引用（含 _test.go 与构建标签；Go 扫描天然覆盖 build tag 文件）
grep -rn --include='*.go' -w '<Symbol>' .

# ③ 非 Go 位置：README/MAINTENANCE/AGENTS.md/CLAUDE.md/.trellis/spec/scripts/.github/web
grep -rn -w '<Symbol>' . --exclude-dir=.git --exclude-dir=node_modules --exclude-dir=dist \
  --exclude='*.go' --exclude-dir=tasks --exclude-dir=archive

# ④ 反射/linkname/注册表字符串
grep -rn 'go:linkname' --include='*.go' .    # 0 命中
grep -rn -w '<Symbol>' --include='*.go' .    # 无引号字符串键命中（除定义/注释）
```

**`.trellis/tasks/**` 的排除说明**：任务 PRD/证据/归档清单**按设计**逐条列出全部 85 个符号名（本清单就是从那里来的），若把它们当作“非 Go 引用”，任何符号都无法删除。③ 只针对**契约性文档**（spec/README/CI/脚本/前端代码），不针对任务记录；这一点在取舍上与上一轮一致。

## 3. 逐符号四类取证表（29 行）

`删前位置` = 本轮起点 `deadcode -test ./...` 输出的精确行号。① 列全部为「不可达」（均来自实际 deadcode 输出，见 §5 的起点清单）。

| # | 符号 | 删前位置（deadcode 行） | ① 可达性 | ② 全仓 Go 引用（含测试/构建标签） | ③ 非 Go 引用 | ④ 反射/注册表/键名 | 处置 |
|---|---|---|---|---|---|---|---|
| 1 | `pkg/billingexpr.RunExprByHash` | `pkg/billingexpr/run.go:39` | 不可达 | 0（仅自身定义+注释）；同文件 `RunExprByHashWithRequest` 是活跃入口（`pkg/billingexpr/settle.go:22`） | 0 | 无注册表/linkname | 已删 |
| 2 | `pkg/cachex.StringCodec.Encode` | `pkg/cachex/codec.go:31` | 不可达 | 0；`StringCodec` 全仓从未实例化（删前仅 3 行：类型+两方法） | 0 | codec 均为静态实例化（IntCodec/JSONCodec），无名字注册表 | 已删 |
| 3 | `pkg/cachex.StringCodec.Decode` | `pkg/cachex/codec.go:32` | 不可达 | 同 #2 | 0 | 同 #2 | 已删 |
| 4 | `pkg/cachex.HybridCache.FullKey` | `pkg/cachex/hybrid_cache.go:55` | 不可达 | 0（同文件 `Namespace.FullKey` 是另一个方法，仍被 hybrid_cache 内部使用，保留） | 0 | 无 | 已删 |
| 5 | `pkg/ionet.Client.GetContainerJobs` | `pkg/ionet/container.go:59` | 不可达 | 0 | 0 | ionet 方法与路由/反射无关；controller/deployment.go 的 client 调用清单不含它（§6.1） | 已删 |
| 6 | `pkg/ionet.Client.GetContainerLogs` | `pkg/ionet/container.go:125` | 不可达 | 0 | 0 | 同 #5 | 已删 |
| 7 | `pkg/ionet.Client.StreamContainerLogs` | `pkg/ionet/container.go:168` | 不可达 | 0 | 0 | 同 #5 | 已删 |
| 8 | `pkg/ionet.Client.RestartContainer` | `pkg/ionet/container.go:232` | 不可达 | 0 | 0 | 同 #5；路由表无 restart 动作 | 已删 |
| 9 | `pkg/ionet.Client.StopContainer` | `pkg/ionet/container.go:251` | 不可达 | 0 | 0 | 同 #5；路由表无 stop 动作 | 已删 |
| 10 | `pkg/ionet.Client.ExecuteInContainer` | `pkg/ionet/container.go:270` | 不可达 | 0 | 0 | 同 #5；无 exec 路由 | 已删 |
| 11 | `pkg/ionet.Client.GetHardwareType` | `pkg/ionet/hardware.go:139` | 不可达 | 0（硬件列表走 `ListHardwareTypes`，保留） | 0 | 同 #5 | 已删 |
| 12 | `pkg/ionet.Client.GetLocation` | `pkg/ionet/hardware.go:161` | 不可达 | 0（位置列表走 `ListLocations`，保留） | 0 | 同 #5 | 已删 |
| 13 | `pkg/ionet.Client.GetLocationAvailability` | `pkg/ionet/hardware.go:183` | 不可达 | 0（可用性走 `GetAvailableReplicas`，保留） | 0 | 同 #5 | 已删 |
| 14 | `oauth.GetAllProviders` | `oauth/registry.go:49` | 不可达 | 0 | 0 | `providers` map 存的是 `Provider` 实例，不以函数名为键；无反射 | 已删 |
| 15 | `oauth.ReloadCustomProviders` | `oauth/registry.go:118` | 不可达 | 0（启动用 `LoadCustomProviders`，`main.go:354`，保留） | 0 | 无 | 已删 |
| 16 | `types.Set.Remove` | `types/set.go:17` | 不可达 | 0 调用；`Set` 实例仅用 `Add`/`Len`/`Items`（`model/log.go:527/531/538/551`、`controller/task.go:69/71`、`model/pricing.go:269/361`） | 0 | 泛型方法，无反射/接口 | 已删 |
| 17 | `types.Set.Contains` | `types/set.go:21` | 不可达 | 同 #16 | 0 | 同 #16 | 已删 |
| 18 | `setting.ContainsAutoGroup` | `setting/auto_group.go:25` | 不可达 | 0（auto group 判定在 `service/group.go` 内联） | 0 | 不在配置注册表；OptionMap 键为 `AutoGroups` 等字符串，与函数名无关 | 已删 |
| 19 | `setting.SetWaffoPayMethods` | `setting/payment_waffo.go:43` | 不可达 | 0；`WaffoPayMethods` 的 option 更新走 `model/option.go:606` 的 `case`（直接写 OptionMap，注释明确“读取走 GetWaffoPayMethods”），`GetWaffoPayMethods` 保留且在用 | 0 | OptionMap 键名字符串与函数名无关 | 已删 |
| 20 | `setting/model_setting.GetQwenSettings` | `setting/model_setting/qwen.go:39` | 不可达 | 0 | 0 | `qwen` 配置经 `config.GlobalConfig.Register("qwen", &qwenSettings)` 注册**结构体**，manager 反射结构体字段（`setting/config/config.go:27/92/164`），不解析 getter | 已删 |
| 21 | `setting/operation_setting.IsCheckinEnabled` | `setting/operation_setting/checkin_setting.go:30` | 不可达 | 0 | 0 | `checkin_setting` 注册结构体；`GetCheckinSetting` 保留且在 `model/checkin.go:56`、`controller/checkin.go:17/48`、`controller/misc.go:124` 使用 | 已删 |
| 22 | `setting/operation_setting.GetCheckinQuotaRange` | `setting/operation_setting/checkin_setting.go:35` | 不可达 | 0 | 0 | 同 #21 | 已删 |
| 23 | `setting/operation_setting.IsCNYDisplay` | `setting/operation_setting/general_setting.go:53` | 不可达 | 0 | 0 | `general_setting` 注册结构体；同文件 `IsCurrencyDisplay`（`controller/misc.go:77`）、`GetQuotaDisplayType`（logger/controller 多处）保留且在用 | 已删 |
| 24 | `setting/performance_setting.GetPerformanceSetting` | `setting/performance_setting/config.go:67` | 不可达 | 0 | 0 | `performance_setting` 注册结构体；`UpdateAndSync` 保留（`model/option.go:643` 调用） | 已删 |
| 25 | `setting/performance_setting.GetCacheStats` | `setting/performance_setting/config.go:78` | 不可达 | 0 | 0 | 纯代理 `common.GetDiskCacheStats`；实际调用方 `controller/performance.go:87` 直接用 common | 已删 |
| 26 | `setting/performance_setting.ResetStats` | `setting/performance_setting/config.go:83` | 不可达 | 0 | 0 | 纯代理；实际调用方 `controller/performance.go:161` 直接用 common | 已删 |
| 27 | `setting/ratio_setting.GetCacheRatioMap` | `setting/ratio_setting/cache_ratio.go:21` | 不可达 | 0 | 0 | 配置保存走 `CacheRatio2JSONString`；`GetCacheRatio`/`GetCacheRatioCopy`/`GetDefaultCacheRatioMap` 保留 | 已删 |
| 28 | `setting/ratio_setting.GetModelPriceMap` | `setting/ratio_setting/model_ratio.go:75` | 不可达 | 0 | 0 | 配置保存走 `ModelPrice2JSONString`；`GetModelPrice`/`GetDefaultModelPriceMap` 保留 | 已删 |
| 29 | `setting/ratio_setting.GetModelRatioOrPrice` | `setting/ratio_setting/model_ratio.go:556` | 不可达 | **0 Go 引用**（仅定义） | **1：`web/src/features/models/components/drawers/model-mutate-drawer.tsx:185` 注释点名** | 无 | **保留**（③ 不满足，见 §6.3） |

## 4. 分批记录与每批验证

| 批次 | 范围（符号数） | 验证 | 结果 |
|---|---|---|---|
| A1 | `pkg/billingexpr` 1 + `pkg/cachex` 3 = 4 | 与 A2 合并做一次门禁 | 见 A2 |
| A2 | `pkg/ionet` 9（container 6 + hardware 3） | `go build ./...` + `go test ./... -count=1` | build exit=0、test exit=0（36 包 ok / FAIL 0） |
| B | `oauth` 2 + `types` 2 = 4 | `go build ./...` + `go test ./... -count=1` | build exit=0、test exit=0 |
| C1 | `setting`：ContainsAutoGroup、SetWaffoPayMethods、GetQwenSettings、IsCheckinEnabled、GetCheckinQuotaRange、IsCNYDisplay = 6 | 与 C2 合并做最终门禁 V3 | 见 C2 |
| C2 | `setting`：performance_setting 3 + ratio_setting 2 = 5 | 最终门禁 V3（§7） | 全部绿 |

说明（如实记录）：
- A1/A2 是两次连续机械删除、共用一个门禁点，合计 13 个，略高于建议的“每批 ≤10”；因 13 个全部是叶子包 0 引用方法，且改完立刻全量 build+test 通过，未回退。
- B 与 C 的删除在时间上有重叠，单批门禁结论以最终门禁 V3（全量 28 删完成后的整树）为准：**全绿**。
- 全程零失败，**未发生任何回退**。

## 5. deadcode 变化与级联

起点（B 堆收尾后）**81 行**，终点 **53 行**，减少 **28**，与本次删除的 28 个符号一一对应（删前 28 条 `unreachable func` 行见下表；`GetModelRatioOrPrice` 为保留项）。

```console
$ /root/go/bin/deadcode -test ./... | wc -l
53
```

被移除的 28 条（起点 → 终点都消失）：

```text
pkg/billingexpr/run.go:39            RunExprByHash
pkg/cachex/codec.go:31               StringCodec.Encode
pkg/cachex/codec.go:32               StringCodec.Decode
pkg/cachex/hybrid_cache.go:55        HybridCache.FullKey
pkg/ionet/container.go:59            Client.GetContainerJobs
pkg/ionet/container.go:125           Client.GetContainerLogs
pkg/ionet/container.go:168           Client.StreamContainerLogs
pkg/ionet/container.go:232           Client.RestartContainer
pkg/ionet/container.go:251           Client.StopContainer
pkg/ionet/container.go:270           Client.ExecuteInContainer
pkg/ionet/hardware.go:139            Client.GetHardwareType
pkg/ionet/hardware.go:161            Client.GetLocation
pkg/ionet/hardware.go:183            Client.GetLocationAvailability
oauth/registry.go:49                 GetAllProviders
oauth/registry.go:118                ReloadCustomProviders
types/set.go:17                      Set.Remove
types/set.go:21                      Set.Contains
setting/auto_group.go:25             ContainsAutoGroup
setting/payment_waffo.go:43          SetWaffoPayMethods
setting/model_setting/qwen.go:39     GetQwenSettings
setting/operation_setting/checkin_setting.go:30   IsCheckinEnabled
setting/operation_setting/checkin_setting.go:35   GetCheckinQuotaRange
setting/operation_setting/general_setting.go:53   IsCNYDisplay
setting/performance_setting/config.go:67          GetPerformanceSetting
setting/performance_setting/config.go:78          GetCacheStats
setting/performance_setting/config.go:83          ResetStats
setting/ratio_setting/cache_ratio.go:21           GetCacheRatioMap
setting/ratio_setting/model_ratio.go:75           GetModelPriceMap
```

级联核对（对比起点/终点 deadcode 集合）：

```console
$ comm -13 <(sort before-81.txt) <(sort after-53.txt)
setting/ratio_setting/model_ratio.go:552:6: unreachable func: GetModelRatioOrPrice
```

唯一“新增”行是**保留符号的行号前移**（删掉上方 `GetModelPriceMap` 后由 `:556` 变 `:552`），不是新不可达符号 → **级联新增 = 0**。

终点 53 行的分布：`common` 29、`relaykit` 21、`relay` 1、`service` 1、`setting` 1（即保留的 `GetModelRatioOrPrice`）。`pkg`/`oauth`/`types` 已归零。

对账：原始 85 → 53 = 85 − 4（B 堆：`IsPrivateIP`、`GetContextKeyStringSlice` 可达 + kitutil 两函数删除）− 28（本轮）。PRD 的“85 → 约 56”是未计 B 堆的口径；以本表实测为准。

## 6. setting/ 特殊规则与保留项说明

### 6.1 配置注册表机制（为什么删 getter 安全）

`setting/config/config.go` 的机制：

- `ConfigManager.Register(name string, config interface{})`（`:27`）把**配置结构体指针**放进 map；
- `LoadFromDB` / `SaveToDB` 通过 `configToMap`（`:92`）与 `updateConfigFromMap`（`:164`）**反射结构体字段 + `json` tag** 读写键值；
- 全过程不解析、不查找任何 getter 函数。

因此删掉 11 个 setting getter 不改变任何配置键名、注册项或 DB 读写行为；它们的结构体实例（`qwenSettings`、`checkinSetting`、`generalSetting`、`performanceSetting`）与在用的 accessor（`GetCheckinSetting`、`IsCurrencyDisplay`、`GetQuotaDisplayType`、`GetWaffoPayMethods`、`CacheRatio2JSONString`、`ModelPrice2JSONString` 等）全部保留。删后 `setting/*` 四个子包的测试均通过（§7）。

`ionet` 侧同样核对：`controller/deployment.go` 里实际调用的 client 方法为 `CheckClusterNameAvailability / DeleteDeployment / DeployContainer / ExtendDeployment / GetAvailableReplicas / GetContainerDetails / GetContainerLogsRaw / GetDeployment / GetMaxGPUsPerContainer / GetPriceEstimation / ListContainers / ListDeployments / ListHardwareTypes / ListLocations / UpdateClusterName / UpdateDeployment`——与删除的 9 个方法零交集。

### 6.2 成组扩展点的判定

任务规则要求“若某 getter 属于同类配置的成组扩展点 → 保留并说明”。逐个核对后判定**本批 11 个 setting 符号都不属于成组扩展点**，理由分三类：

| 类别 | 符号 | 判定依据 |
|---|---|---|
| 冗余便捷 getter（同文件/同配置文件另有在用入口） | `GetQwenSettings`（`IsSyncImageModel` 直接用结构体）、`GetPerformanceSetting`（`UpdateAndSync` 保留）、`IsCheckinEnabled`/`GetCheckinQuotaRange`（`GetCheckinSetting` 保留）、`IsCNYDisplay`（`IsCurrencyDisplay`/`GetQuotaDisplayType` 在用） | 配置文件仍完整暴露注册结构体；删的只是“再多包一层”的未用入口 |
| 纯代理 common 的包装 | `GetCacheStats`、`ResetStats` | 真实调用方直接调 `common.GetDiskCacheStats` / `common.ResetDiskCacheStats`（`controller/performance.go:87/161`），包装无人使用 |
| 未被保存路径使用的读/写辅助 | `ContainsAutoGroup`（判定已内联）、`SetWaffoPayMethods`（option 更新走 `model/option.go:606` 直接写 OptionMap）、`GetCacheRatioMap`/`GetModelPriceMap`（配置保存走 `*2JSONString`；单模型读取走 `GetCacheRatio`/`GetModelPrice`） | 都不在注册表/反射/字符串键上 |

### 6.3 保留项：`GetModelRatioOrPrice`

- 全仓 Go 0 引用（① deadcode 不可达、② 无 Go 引用、④ 无反射/键名）均满足；
- 但 ③ 不满足：`web/src/features/models/components/drawers/model-mutate-drawer.tsx:185` 的注释写着
  `// GetModelRatioOrPrice), so a name that has one is shown, and saved back, as`——它用这个 Go 函数名解释前端“固定价优先”的展示逻辑。删掉 Go 符号会让该前端注释悬空（正是 ③ 要防的 spec 腐化）。
- 本任务写作用域不含 `web/`，故**保留符号**并在本文件记录。若 Lead 决定删除，需同步把该 TSX 注释改成不点名已删函数（或指向 `GetModelPrice`），再把本行改为“已删”。

## 7. 门禁（最终树实测）

执行环境：`GOPROXY=off`（离线）；工具在位（`/usr/local/bin/golangci-lint`、`/root/go/bin/deadcode`）。

```console
### GOPROXY=off go build ./...
exit=0
### GOPROXY=off go vet ./...
exit=0
### GOPROXY=off go test ./... -count=1
ok  	github.com/QuantumNous/new-api/common	1.148s
ok  	github.com/QuantumNous/new-api/controller	0.622s
ok  	github.com/QuantumNous/new-api/logger	0.013s
ok  	github.com/QuantumNous/new-api/middleware	0.111s
ok  	github.com/QuantumNous/new-api/model	4.517s
ok  	github.com/QuantumNous/new-api/oauth	0.029s
ok  	github.com/QuantumNous/new-api/pkg/billingexpr	0.049s
...（共 36 个包 ok，FAIL 行 0）
exit=0
### relaykit: GOWORK=off go build ./...
exit=0
### relaykit: GOWORK=off go test ./... -count=1
ok  	.../relaykit/dto 0.004s
ok  	.../relaykit/relayconvert 0.013s
ok  	.../relaykit/relayconvert/convmeta 0.003s
ok  	.../relaykit/relayconvert/internal/oai_chat 0.003s
ok  	.../relaykit/relayconvert/internal/oai_responses 0.004s
ok  	.../relaykit/relayconvert/kitutil 0.003s
ok  	.../relaykit/relayconvert/reasoning 0.003s
（其余 8 个包无测试文件）
exit=0
### gofmt -l .
(无输出)
exit=0
### golangci-lint run --default=none --enable=unused ./...
model/main.go:27:5: var logKeyCol is unused (unused)
1 issues:
* unused: 1
```

`unused` 仍恰 1（`logKeyCol`），与 PRD 预期一致。

改动边界自证：

```console
$ git status --short
 M common/ip.go                                  # B 堆（task-54）
 M common/ssrf_protection.go                     # B 堆
 M common/ssrf_protection_test.go                # B 堆
 M oauth/registry.go                             # 本任务
 M pkg/billingexpr/run.go                        # 本任务
 M pkg/cachex/codec.go                           # 本任务
 M pkg/cachex/hybrid_cache.go                    # 本任务
 M pkg/ionet/container.go                        # 本任务
 M pkg/ionet/hardware.go                         # 本任务
 M relaykit/README.md                            # B 堆（task-54）
 M relaykit/relayconvert/kitutil/json.go         # B 堆
 M service/group.go                              # B 堆
 M service/group_auto_groups_test.go             # B 堆
 M setting/auto_group.go                         # 本任务
 M setting/model_setting/qwen.go                 # 本任务
 M setting/operation_setting/checkin_setting.go  # 本任务
 M setting/operation_setting/general_setting.go  # 本任务
 M setting/payment_waffo.go                      # 本任务
 M setting/performance_setting/config.go         # 本任务
 M setting/ratio_setting/cache_ratio.go          # 本任务
 M setting/ratio_setting/model_ratio.go          # 本任务
 M types/set.go                                  # 本任务
?? .trellis/tasks/10-24-internal-symbols-and-b-wiring/   # 任务目录（证据文件）
?? .trellis/tasks/10-24-tier-name-english-display/       # 其它会话新建的任务目录，与本任务无关
```

本任务代码改动共 **15 个文件、-368/+1 行**（`git diff --numstat -- pkg oauth types setting`；+1 是 `pkg/ionet/container.go` 删到只剩 `fmt` 后 import 改为单行）。无 `go.mod`/`go.sum` 改动、无临时文件残留、未触碰 `web/` 与 `relaykit/` 代码。

## 8. 回滚方式

删除均为纯删除（+ 1 处 import 收缩），未改逻辑；逐批回滚命令（未跟踪任务目录不受影响）：

```bash
# A 批：pkg
git checkout -- pkg/billingexpr/run.go pkg/cachex/codec.go pkg/cachex/hybrid_cache.go \
  pkg/ionet/container.go pkg/ionet/hardware.go
# B 批：oauth + types
git checkout -- oauth/registry.go types/set.go
# C 批：setting
git checkout -- setting/auto_group.go setting/model_setting/qwen.go \
  setting/operation_setting/checkin_setting.go setting/operation_setting/general_setting.go \
  setting/payment_waffo.go setting/performance_setting/config.go \
  setting/ratio_setting/cache_ratio.go setting/ratio_setting/model_ratio.go
# 或一次性回滚本任务
git checkout -- pkg oauth types setting
```

回滚后重跑 §7 门禁；注意 `git checkout -- pkg` 不会影响 B 堆的 `common/`、`service/`、`relaykit/` 改动。

## 9. 残留风险与建议下一步

1. **孤儿导出类型（本任务范围外，deadcode 不报告类型，故清单未列）**：删除方法/函数后以下类型在仓内已无引用，建议下一轮（或 Lead 决定）作为级联清理：
   - `pkg/cachex.StringCodec`（两方法删后成空壳类型）；
   - `pkg/ionet.ContainerLogs`、`pkg/ionet.LogEntry`（唯一使用者是已删的 `GetContainerLogs` / `StreamContainerLogs`）；
   - `pkg/ionet.LocationAvailability`（唯一使用者是已删的 `GetLocationAvailability`）。
   本轮严格按 29 条清单执行，未顺手删除，避免范围外 API 面变更。
2. **`GetModelRatioOrPrice` 保留**（§6.3）：若 Lead 决定删，需要 `web/` 的 TSX 注释同步改（写作用域外）。
3. **relaykit 20 项导出面**（A 18 + B 2）仍需 owner 决策，本轮未动。
4. 全部改动未提交、未推送、未发版；由 Lead 收口提交与后续独立验证。

渲染安全自查：正文无成对美元符号（美元符号只出现在 console 代码块的命令提示符与代码片段里）；不使用单星号斜体；各表格列数一致；反引号代码片段已配对。
