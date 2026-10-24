# 28 符号删除 + B 堆接线 + 档位名翻译：独立验证证据（verifier）

- 任务：共享任务板 `task-56`（Go 侧 Trellis 任务 `10-24-internal-symbols-and-b-wiring` + 前端 `10-24-tier-name-english-display` 的独立验证）
- 验证者：verifier（只读验证；未改仓库文件；未使用 `-mod=mod`，golangci 用 `-mod=readonly`）
- 验证时间：2026-10-24（UTC 07:04 起）
- 基线：`HEAD = 8a4b05e`；最终 `git status --porcelain` 行数 = **36**（32 已跟踪修改 + 4 未跟踪条目）；
  `git diff` = **32 文件、320 增、410 删**；diff 指纹 `9d7a8b815e0e77f04a68ac124cb3f9256501ec09423ec22d886a15ad5a27bd0e`
- `go.mod` / `go.sum` 无改动：blob 均等于 HEAD（`b0642f16…` / `357374c3…`）
- 工具环境：Go 1.26.6、`~/go/bin/deadcode`（RTA）、Bun（web 目录）

## 结论

**接受（Accept）**。Go 侧：31 个导出声明删除（= 任务点名的 28 个死符号 + kitutil 2 个包装 + 旧 `common.IsPrivateIP`），
经全量与编译器双重核对无任何残留引用与文档引用；B 堆接线语义逐条核对无行为倒退；门禁全绿、`unused` 恰 1、deadcode
53 且**新增不可达 0**。前端侧：翻译严格限于展示层（比较/高亮/条件摘要仍用原始标签，且有测试反证），映射与
7 locale 齐备正确、`i18n:check` 与全部前端门禁通过。边界干净。
两条非阻断说明：实现者「起点 81」是 B 堆收尾后的中间态基线（HEAD 实测 85，终点 53 与「无新增不可达」均独立复现）；
`GetContextKeyStringSlice` 因 group.go 重构由不可达变为可达，是 gone 为 32 而非 31 的原因。

## 一、Go：删除安全（28 + 3 个导出声明）

### 1.1 独立提取与计数口径

从 `git diff -U0 -- '*.go'` 独立解析删除行，得到 **31 个导出声明**（25 函数 + 6 方法）：

| 分组 | 数量 | 明细 |
|---|---|---|
| pkg/billingexpr | 1 | `RunExprByHash` |
| pkg/cachex | 3 | `StringCodec.Encode`、`StringCodec.Decode`、`HybridCache.FullKey` |
| pkg/ionet | 9 | `Client.GetContainerJobs`、`GetContainerLogs`、`StreamContainerLogs`、`RestartContainer`、`StopContainer`、`ExecuteInContainer`、`GetHardwareType`、`GetLocation`、`GetLocationAvailability` |
| oauth | 2 | `GetAllProviders`、`ReloadCustomProviders` |
| types/set.go | 2 | `Set.Remove`、`Set.Contains` |
| setting（8 文件） | 11 | `ContainsAutoGroup`、`GetQwenSettings`、`IsCheckinEnabled`、`GetCheckinQuotaRange`、`IsCNYDisplay`、`SetWaffoPayMethods`、`GetPerformanceSetting`、`GetCacheStats`、`ResetStats`、`GetCacheRatioMap`、`GetModelPriceMap` |
| 小计（= 任务点名的 28 个死符号） | **28** | 与前六行一致 |
| B 堆附带 | 3 | `relaykit/relayconvert/kitutil.UnmarshalJsonStr`、`kitutil.DecodeJson`、`common/ip.go` 旧 `IsPrivateIP` |

结论：「28 个导出死符号」的口径准确（不含 B 堆的 3 个），本证据按 31 个全量核对。

### 1.2 引用核对（全量）

| 维度 | 结果 |
|---|---|
| HEAD 全树中落在删除行之外的引用 | 0（唯一命中是 `common.DecodeJson` / `common.UnmarshalJsonStr` 的同名**保留**函数，与 kitutil 被删包装无关） |
| 工作区对被删 kitutil 包装的引用（含 .go/.md） | 0（仅任务归档证据与 README 的说明性提及） |
| 工作区对 `IsPrivateIP` 的引用 | 仅新位置定义 + 3 处 SSRF 内部调用 + 新增测试（即 B 堆的搬迁目标） |
| 方法类删除（StringCodec 两个方法、HybridCache.FullKey、Client 的 9 个方法、Set.Remove 与 Set.Contains） | 编译器级证据：`go build ./...`、`go vet ./...`、`go test ./... -count=1`、relaykit `GOWORK=off build+test` 全绿；另经针对性检索确认 `StringCodec` 仅剩类型定义无使用者、`types.Set` 使用者只调 `Add/Len/Items`、`FullKey` 命中均为 `Namespace.FullKey`（保留方法）而非被删的 `HybridCache.FullKey` |
| 非 Go 位置（README/MAINTENANCE/AGENTS.md/CLAUDE.md/.trellis/spec/scripts/.github/web） | 对 31 个名字**无一处真实引用**：命中仅通用英文词（Contains/Remove/Decode）与**保留**的 `common.UnmarshalJsonStr` / `common.DecodeJson`（AGENTS.md:76-77、quality-guidelines.md:41）；`relaykit/README.md:248/255` 是本次新增的「模块例外」说明（刻意点名被删副本） |
| 抽样 | ≥10 覆盖各分组（见 1.1 明细逐条核对；另 `pkg/ionet` 唯一消费者为 controller/deployment.go，删除的 9 个方法在其中无调用） |

### 1.3 保留项 `setting/ratio_setting.GetModelRatioOrPrice` 复核

| 核对项 | 结果 |
|---|---|
| Go 引用 | `git grep` HEAD：仅定义处（0 调用）；工作区函数仍在（model_ratio.go:552） |
| 前端点名 | `web/src/features/models/components/drawers/model-mutate-drawer.tsx:185` 注释原文含 `GetModelRatioOrPrice`，用它解释「固定价优先」的展示逻辑 |
| 判定 | 理由**成立**：删除 Go 符号会让该 TSX 注释悬空（跨模块文档腐化）。它是 deadcode 53 中 setting 的唯一条目，与声明一致 |

## 二、Go：B 堆接线语义核对

### 2.1 `IsPrivateIP` 搬迁

| 核对项 | 结果 |
|---|---|
| 「旧导出 `IsPrivateIP` 在 HEAD 全仓 0 调用」 | **成立**：`git grep 'IsPrivateIP(' HEAD -- '*.go'` 只有 `common/ip.go:14` 定义一行 |
| 新实现来源 | `common/ssrf_protection.go` 的内部 `isPrivateIP` 改名导出；逐行比对函数体（去掉注释后 diff）**仅函数名一行不同**，逻辑零改动 |
| 调用点影响 | 工作区调用点只有 SSRF 内部 3 处（`IsIPAccessAllowed`、`ipAccessError` 两处）+ 新增测试；这些点**原本就在用**同一实现 → 更严格的语义（nil→true、ULA、未指定、接口本地组播等）**没有**新落到任何此前行为不同的调用路径上，无行为倒退 |
| SSRF 行为 | 因函数体未变，SSRF 校验行为与改动前逐字一致 |
| 新测试 | `common/ssrf_protection_test.go` 覆盖 IPv4/IPv6 私网与特殊用途（含 `fc00::/7` 上下边界、`2001:db8::/32`、多播、nil→true、IPv4-mapped），并显式注释「空但非 nil 的 IP 按非私网处理，与抽取前逐字一致」；3 个测试全部 PASS |
| 文档引用 | 全仓非 Go 位置无 `IsPrivateIP` 引用（旧名与新名均无腐化） |

### 2.2 `service/group.go` 三分支语义

| 分支 | 改动前（HEAD） | 改动后 | 等价性 |
|---|---|---|---|
| ① context 缺 key | 返回 `GetUserAutoGroup` | 同 | 一致 |
| ② 存在但类型不符 | `value.([]string)` 失败 → 返回 `[]string{}` | `GetStringSlice` 对非 `[]string` 返回 nil → `FilterUserTokenAutoGroups(nil)` 返回 `make([]string, 0, 0)` | 均为非 nil 空切片，语义一致（不会回退全局） |
| ③ 存在且合法 | 直接传入 `groups` 过滤 | 取 `GetStringSlice` 后过滤 | 一致（gin 对 `[]string` 原样返回） |

- `common.GetContextKeyStringSlice` 的实现是 `c.GetStringSlice`（common/gin.go:177-179），gin 对类型不符返回 nil，故分支②不会变成 `[]string{}` 以外的结果。
- 新增 `service/group_auto_groups_test.go` 5 个用例覆盖三分支 + limit 顺序 + 权限变更后不回退；`go test ./service/ -run TestGetRequestAutoGroups -count=1 -v` → 5 PASS。

### 2.3 `relaykit` 两个包装删除 + README 模块例外

- `kitutil.UnmarshalJsonStr` / `kitutil.DecodeJson` 删除后，全仓 .go/.md 引用为 0（relaykit 内也无人调用）。
- `relaykit/README.md` 新增「JSON 编码约定（模块例外）」：说明 relaykit 不能导入根模块 `common`，因此 AGENTS.md 的
  `common.*` JSON 约定在模块内不适用，并记录这两个副本因此删除——与 AGENTS.md「relaykit module independence」一致。
- `cd relaykit && GOWORK=off go build ./...` 与 `GOWORK=off go test ./... -count=1` 均 rc=0。

## 三、Go 门禁与级联

| 门禁 | 命令 | 结果 |
|---|---|---|
| 构建 | `GOPROXY=off go build ./...` | rc=0 |
| 静态检查 | `GOPROXY=off go vet ./...` | rc=0 |
| 全仓测试 | `GOPROXY=off go test ./... -count=1` | rc=0，无 FAIL |
| relaykit 构建 + 测试 | `cd relaykit && GOWORK=off GOPROXY=off go build ./... && GOWORK=off go test ./... -count=1` | 均 rc=0 |
| 格式 | `gofmt -l .` | 输出为空 |
| unused | `GOPROXY=off GOFLAGS=-mod=readonly golangci-lint run --default=none --enable=unused ./...` | **恰 1 issue**：`model/main.go:27 var logKeyCol is unused` |

**deadcode 级联（独立复现）**：

| 指标 | 数值 |
|---|---|
| HEAD 全树（/tmp 副本，含 web/dist 以支持 embed） | **85** |
| 工作区（终态） | **53** |
| 集合关系 | after 是 before 的**子集** |
| 新增不可达 | **0** |
| 消失条目 | **32** |

消失的 32 条 = **31 个被删导出声明** + **1 个「由不可达变为可达」的 `common.GetContextKeyStringSlice`**
（group.go 重构后开始调用它，故从名单消失，并非被删除）。终态 53 的分布（独立统计）：common 29、relaykit 各子包 21、
relay 1、service 1、setting/ratio_setting 1（即保留的 `GetModelRatioOrPrice`）。

**与实现者「起点 81 → 终点 53、减少 28」的对账**：81 是「B 堆收尾后、删 28 个死符号前」的中间态基线：
85 − 3（B 堆删除：旧 IsPrivateIP、2 个 kitutil 包装）− 1（GetContextKeyStringSlice 转为可达）= 81；再 −28 = 53。
两个口径的**终点与「无新增不可达」结论一致**，仅基线定义需在报告里写明（见观察 1）。

## 四、前端：档位名翻译

### 4.1 仅在展示层（有测试反证）

| 核对项 | 结果 |
|---|---|
| 组件改动面 | 仅 1 行 import + 2 处 Badge 文案（原来的原始标签加 Default 回退写法 → `getTierDisplayLabel(tier.label, t)`）；`git diff` 无其它行 |
| 比较/高亮仍用原始标签 | 代码：行 287 `tier.label === matchedTierLabel`、行 357/376 `normalizeTierLabel(tier.label) === normalizedMatchedTierLabel`；`formatConditionSummary` 未被改动 |
| 测试反证 | 新增组件测试「the matched row is still resolved by the raw tier label」：传原始标签 `base` 时高亮出现，传译文 `Base` 时高亮为 0 —— 证明高亮比较的是原始标签 |
| 展示测试 | 中文 UI 下 Badge 出现「基础」「高峰」且**不出现**原始 `base`；英文 UI 下出现 `Base`/`Peak hours`；自定义档位 `my_custom_tier` 原样显示 |

### 4.2 映射与边界行为

| 输入 | 输出 | 核对 |
|---|---|---|
| `peak` | 英文 `Peak hours`，中文「高峰」 | 与要求一致 |
| `offpeak` | 英文 `Off-peak hours`，中文「低谷」 | 与要求一致 |
| 未知标签（`my_custom_tier`、`My Custom Tier`、`off_peak`） | 原样返回 | 单测断言 |
| 空串 / 纯空格 / null / undefined | `t('Default')`（中文「默认」） | 单测断言 |
| 大小写与分隔符（`  PEAK  `、`Short Output`） | 归一化后命中映射（「高峰」「短输出」） | 单测断言 |

`tier-labels.ts` 的 `getTierDisplayLabel` 只做「查表 + 回退」，不改写调用方数据；映射表 `TIER_LABEL_I18N_KEYS` 12 项。

### 4.3 i18n 与前端门禁

| 核对项 | 结果 |
|---|---|
| 7 locale 新增键 | 每份 **11 行新增、0 行删除**（en/zh/zh-TW/fr/ja/ru/vi）→ **只插入、无重排** |
| 为何 11 而非 12 | 映射表第 12 项 `Discount` 在 7 份 locale 中**本就存在**（各 1 处），无需新增 |
| 中文翻译 | `Peak hours` = 高峰、`Off-peak hours` = 低谷（另有 `Off-peak` = 低谷 兼容旧写法） |
| `STATIC_I18N_KEYS` | 在数组末尾追加 12 个键（附注释），未重排既有项 |
| `i18n:check` | rc=0：4036 个字面 t() 键 + 467 个静态键 = 4197 个引用键在 7 个 locale 全部存在 |
| typecheck | `bun run typecheck`（tsgo）rc=0 |
| 单测 | `bun test src/features/pricing/` → **15 pass / 0 fail**（3 个文件，含新增 tier-labels 6 例与组件 4 例） |
| oxlint | 4 个改动文件：0 warnings / 0 errors |

## 五、边界

| 核对项 | 结果 |
|---|---|
| `git status --porcelain` | 36 行 = 32 M + 4 ??；32 个已跟踪修改与任务描述一致（Go 22 + 前端 10） |
| 未跟踪 | 仅 2 个任务目录 + 新增 `web/src/features/pricing/lib/tier-labels.ts`、`web/src/features/pricing/components/__tests__/tier-labels.test.ts` |
| 依赖文件 | `go.mod` / `go.sum` blob 均等于 HEAD，无改动 |
| 意外文件 | 无（无临时文件、无演示残留） |

## 六、观察与不确定项（均非阻断）

1. **基线口径**：实现者「起点 81」是 B 堆收尾后的中间态（HEAD 实测 85）。对账完全吻合（85−3−1=81，81−28=53），
   建议在其证据里标注「起点 = B 堆收尾后」，避免与仓库 HEAD 的数字混淆。
2. **额外收益**：`common.GetContextKeyStringSlice` 因 group.go 重构从不可达转为可达，是 gone=32 的唯一非删除项，
   也是「接线」正向效果的证据。
3. **保留项**：`GetModelRatioOrPrice` 的保留理由成立但依赖一条前端注释；若后续要压 deadcode 数字，可改为删除
   函数 + 同步改写该 TSX 注释（属 owner 取舍，本轮保留无风险）。
4. **不确定项**：B 堆更严格的 IP 语义未在真实攻击/边界流量下验证（仅有单测覆盖）；因该实现原本就是 SSRF 的唯一判定源，
   路径未变化，判断为无行为差异。
5. **不确定项**：前端 `bun test` 在 node:test + jsdom 环境下运行；未做浏览器端人工验证（门禁与组件测试已覆盖渲染文案）。

## 七、复跑命令清单

```bash
cd /root/new-api-own
# A. 删除安全（提取 + 引用核对脚本见本证据正文）
git diff -U0 -- '*.go'                       # 31 个导出声明（28 + kitutil 2 + 旧 IsPrivateIP）
git grep -n 'IsPrivateIP(' HEAD -- '*.go'    # 仅定义处 = 0 调用
# B. 门禁
GOPROXY=off go build ./... && GOPROXY=off go vet ./...
GOPROXY=off go test ./... -count=1
(cd relaykit && GOWORK=off GOPROXY=off go build ./... && GOWORK=off go test ./... -count=1)
gofmt -l .
GOPROXY=off GOFLAGS=-mod=readonly golangci-lint run --default=none --enable=unused ./...
GOPROXY=off GOFLAGS=-mod=readonly ~/go/bin/deadcode -test ./...   # 53 行
GOPROXY=off go test ./service/ -run TestGetRequestAutoGroups -count=1 -v
GOPROXY=off go test ./common/ -run TestSSRFProtection -count=1 -v
# C. 前端
cd web
bun run typecheck && bun run i18n:check
bun test src/features/pricing/
./node_modules/.bin/oxlint -c .oxlintrc.json src/features/pricing/lib/tier-labels.ts src/features/pricing/components/dynamic-pricing-breakdown.tsx
# D. 边界
git status --porcelain go.mod go.sum   # 应为空
```
