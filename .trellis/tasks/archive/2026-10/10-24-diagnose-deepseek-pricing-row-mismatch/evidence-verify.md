# 重置定价恢复整份默认：独立验证证据（verifier）

- 任务：共享任务板 `task-28`（Trellis 任务 `10-24-diagnose-deepseek-pricing-row-mismatch` 的独立验证）
- 验证者：verifier（只读验证；未改任何仓库文件）
- 验证时间：2026-10-24（UTC 03:34 起）
- 基线：`HEAD = fe8cba5`；已跟踪改动 12 文件（149 增 17 删）+ 2 个新增未跟踪文件
- 基线指纹：`git diff | sha256sum = 691024135caffb3d43496942aeb137b2bef13f97ba8d14cb4a29cbc0133d09d1`
- 新增文件哈希：`controller/pricing_test.go = 68d0a0c4eca0a7970f56d05bd6d237b96ffa912b`、
  `ratio-settings-card-reset.test.tsx = 3b71ef4ef7af3664ce7b1b8c304aa8a0ef46724c`
- 被验证已跟踪文件 blob：

| 文件 | blob |
|---|---|
| controller/pricing.go | 952cc26255fbcd4a5086b07b1c01fbe5aa739414 |
| setting/billing_setting/tiered_billing.go | 04cf6ffa9584ea66da9a98ef78be0dc336b8bf98 |
| setting/ratio_setting/cache_ratio.go | e5ea58a277241cec4a65f37a51509d1b7482df12 |
| setting/ratio_setting/model_ratio.go | 5b1edcd062fc915e13a2321b72475504241266e4 |
| web .../models/ratio-settings-card.tsx | 0b950aea49a4b8d3d4876779c55acbc55f4ed888 |
| web/src/i18n/locales/en.json | 1057f4d423fc3edea8cbbdbef5f92d868d5c3414 |
| web/src/i18n/locales/zh.json | 7cf1f43ac8d6c37a39f075893879b1a6ef766c66 |
| web/src/i18n/locales/zh-TW.json | 369b756f132b7ae4a33841f46ce79717de32d85f |
| web/src/i18n/locales/fr.json | 0b353697437dd1d4e42cd24d1dcdec28e6ad9753 |
| web/src/i18n/locales/ja.json | 6bf986671f7010f3bb24bfec162b87670230cceb |
| web/src/i18n/locales/ru.json | 78343da31de851782189ba6491ab9425c10d3485 |
| web/src/i18n/locales/vi.json | 4b27a1186ac478e2c4ee23c604b44afb38761f49 |

## 结论

**接受（Accept）**。重置范围与 PRD R1 逐键一致、不越界；回归测试在修复前**行为性失败**（失败值与根因症状一一对应）、修复后全绿；
前端文案与后端实际范围一致；七语言各 +2 键无重排；门禁全绿。实现者自曝的两点均**独立确认属实**（按要求不要求改动）。

## 一、修复前后对照（关键证据）

方法：`git clone /root/new-api-own /tmp/verify-reset`（改动未提交 → 克隆即修复前代码），拷入工作区新增测试后运行。
**全程未在真实工作区做任何 checkout/stash/restore**。

### 1.1 Go 侧

原样拷入 `controller/pricing_test.go` 后先得到**编译失败**（只能证明测试引用了新符号）：

```
controller/pricing_test.go:134:34: undefined: billing_setting.DefaultBillingMode2JSONString
controller/pricing_test.go:135:34: undefined: billing_setting.DefaultBillingExpr2JSONString
controller/pricing_test.go:152:37: undefined: ratio_setting.GetDefaultCompletionRatioMap
controller/pricing_test.go:153:32: undefined: ratio_setting.GetDefaultCacheRatioMap
```

因此我在克隆内补了**两个只提供这 4 个只读访问器、不含任何重置逻辑**的 shim（取值来自同包 internal 默认变量，
与正式实现同源；仅存在于 /tmp 克隆），取得真正的行为对照：

| 用例 | 修复前（克隆 + shim） | 修复后（真实工作区） |
|---|---|---|
| TestResetModelRatioRestoresFullPricingDefaults | FAIL（pricing_test.go:138/139/141） | PASS |
| TestResetModelRatioClearsEmptyDefaultTables | FAIL（pricing_test.go:198/200-204） | PASS |
| TestResetModelRatioWriteFailureIsSilentlySuccessfulInModel | PASS（记录上游吞错行为，符合预期） | PASS |

修复前失败值与根因**逐条对应**（即「重置只写 ModelRatio」）：

```
billing_setting.billing_mode  期望 {"deepseek-flash":"tiered_expr","deepseek-v4.1-flash":"tiered_expr"}
                              实得 {"deepseek-v4.1-flash":"ratio"}
billing_setting.billing_expr  期望 两个模型同一条 266 字符表达式
                              实得 {"deepseek-v4.1-flash":""}
内存 billing mode              期望 tiered_expr，实得 ratio
ModelPrice/ImageRatio/AudioRatio/AudioCompletionRatio/CreateCacheRatio
                              期望 "{}"，实得 {"legacy-model":1.5} / {"legacy-image":2.5} / …
                              内存同样残留 map[legacy-model:1.5] 等
```

即：修复前点击重置后，两个默认模型仍然一行是表达式、一行是按 Token —— 与用户报的症状一致。

### 1.2 前端侧

`/tmp/verify-reset/web` 指回工作区 `node_modules`（符号链接，只读依赖），拷入新测试后运行：

| 用例 | 修复前 | 修复后 |
|---|---|---|
| ratio settings reset scope > the confirm dialog spells out the reset scope and what stays untouched | FAIL（第 156 行断言，确认框仍是旧文案） | PASS |

## 二、门禁复跑（真实工作区）

| 门禁 | 命令 | 结果 |
|---|---|---|
| Go 构建 | `GOPROXY=off go build ./...` | rc=0 |
| Go 静态检查 | `GOPROXY=off go vet ./...` | rc=0，无输出 |
| Go 测试 | `GOPROXY=off go test ./service/... ./controller/... ./setting/... -count=1` | 全部 ok（service、authz、controller、setting、billing_setting、config、model_setting、operation_setting、ratio_setting、system_setting；无 FAIL） |
| 目标用例 | `go test ./controller/ -run TestResetModelRatio -count=1 -v` | 3 用例全 PASS |
| 前端类型 | `cd web && bun run typecheck` | rc=0 |
| 前端 i18n | `bun run i18n:check` | rc=0；4187 引用键 × 7 locale 全存在 |
| 前端测试 | `bun test src/features/system-settings/` | 85 pass / 0 fail（9 文件）；新增文件单跑 1 pass / 0 fail |

Go 全部以 `GOPROXY=off` 运行，测试用内存 SQLite（`file:...?mode=memory&cache=shared`），不触外网。

## 三、重置范围逐键核对（PRD R1）

`controller/pricing.go:88-99` 的 `pricingDefaults` 恰好 10 项，与 PRD R1 逐项对齐：

| R1 要求 | 实现键 | 默认值来源 |
|---|---|---|
| ModelRatio | ModelRatio | `ratio_setting.DefaultModelRatio2JSONString()` |
| CompletionRatio | CompletionRatio | `ratio_setting.DefaultCompletionRatio2JSONString()`（新增） |
| CacheRatio | CacheRatio | `ratio_setting.DefaultCacheRatio2JSONString()`（新增） |
| CreateCacheRatio | CreateCacheRatio | `ratio_setting.DefaultCreateCacheRatio2JSONString()`（新增） |
| ModelPrice | ModelPrice | `ratio_setting.DefaultModelPrice2JSONString()`（新增） |
| ImageRatio | ImageRatio | `ratio_setting.DefaultImageRatio2JSONString()`（新增） |
| AudioRatio | AudioRatio | `ratio_setting.DefaultAudioRatio2JSONString()`（新增） |
| AudioCompletionRatio | AudioCompletionRatio | `ratio_setting.DefaultAudioCompletionRatio2JSONString()`（新增） |
| billing_setting.billing_mode | `"billing_setting." + BillingModeField` | `billing_setting.DefaultBillingMode2JSONString()`（新增） |
| billing_setting.billing_expr | `"billing_setting." + BillingExprField` | `billing_setting.DefaultBillingExpr2JSONString()`（新增） |

不越界核对：

- `GroupRatio` / `TopupGroupRatio` / `GroupGroupRatio` / `UserUsableGroups` / `AutoGroups` / `ExposeRatioEnabled`
  在 `controller/pricing.go` 中**只**出现在 `GetPricing` 的只读查询（:42/:51/:59/:62）与注释（:85-86），重置路径无任何写入。
- 回归用例另有哨兵：重置前写入 `GroupRatio={"default":1,"vip":7.5,"sentinel_group":3.25}`、
  `TopupGroupRatio={"default":1,"sentinel_topup":9.5}`，重置后 DB 与内存（`vip=7.5`）均原样保留。
- 重置函数只经由 `model.UpdateOption` 写这 10 个键，无其它写路径。

无字面量复制核对：

- 10 项默认值全部来自导出 getter，控制器内无任何内联默认表/默认表达式字面量。
- 新增的 9 个 getter 全部直接 `common.Marshal` 同包 internal 默认变量（`defaultModelRatio` / `defaultCompletionRatio` /
  `defaultCacheRatio` / `defaultCreateCacheRatio` / `defaultModelPrice` / `defaultImageRatio` / `defaultAudioRatio` /
  `defaultAudioCompletionRatio` / `defaultBillingSetting()`），未复制任何表内容。
- 默认表达式字面量在全仓库生产代码中**只出现 1 次**（`setting/billing_setting/tiered_billing.go:36`
  的 `defaultDeepSeekPeakOffPeakExpr`），另有 2 次出现在既有测试常量里（预期）。
- 「代码默认」与「启动种子」同源：`InitRatioSettings()`（`setting/ratio_setting/model_ratio.go:64-73`）
  用同一批 `default*` 变量初始化 8 张活表，故重置写回的正是启动时的出厂值。

## 四、一致性核对（计费默认 / 两模型逐字符相同 / DB 与内存同步）

| 核对项 | 证据 | 判定 |
|---|---|---|
| 两个默认模型都得到 tiered_expr | `defaultBillingSetting()` 遍历 `defaultTieredBillingModels`（恰为 `deepseek-flash`、`deepseek-v4.1-flash`），逐个赋 `BillingModeTieredExpr` 与同一个 `defaultDeepSeekPeakOffPeakExpr` | 通过 |
| 表达式逐字符相同 | 同一次循环引用同一个常量 → 构造上相同；既有 `tiered_billing_defaults_test.go:22-38` 断言两模型都等于固定字面量、且默认表恰好 2 个键；重置后 `pricing_test.go` 又断言 `GetBillingExpr(ref) == GetBillingExpr(drift)`（逐字符，加 `NotEmpty`） | 通过 |
| 重置后 DB 落库 | `pricing_test.go` 用 `storedOption()` 直接查 options 表，断言 `billing_setting.billing_mode` / `billing_expr` / `ModelRatio` / `CompletionRatio` / `CacheRatio` 的 DB 值等于代码默认，且陈旧条目（`stale-model`）被清掉 | 通过 |
| 重置后内存同步 | 8 个倍率键在 `model.updateOptionMap` 有对应 case（`model/option.go:554-575`，例如 `ModelRatio → ratio_setting.UpdateModelRatioByJSONString`）；两个 billing 键走 `handleConfigUpdate` 的注册表路径（`model/option.go:607-641`，`billing_setting` 已注册），并触发 `InvalidatePricingCache()` + `InvalidateExposedDataCache()` | 通过 |
| 用例同时断言内存 | `pricing_test.go` 对三张倍率表断言「内存值 == 重置后 DB 值」，对 billing 断言 `GetBillingMode`/`GetBillingExpr`，对空表断言 `GetXxxCopy()` 为空 | 通过 |
| 接口兼容 | 路由未改（`router/api-router.go:204` → `controller.ResetModelRatio`；前端仍 POST `/api/option/rest_model_ratio`），响应仍 HTTP 200 + `success`/`message` | 通过 |

## 五、前端文案与七语言

确认框新文案（`ratio-settings-card.tsx:516-519`）与后端 10 键逐项对应，无夸大、无遗漏：

| 文案片段 | 对应后端键 |
|---|---|
| model ... ratios | ModelRatio |
| completion ... ratios | CompletionRatio |
| cache and cache-write ratios | CacheRatio、CreateCacheRatio |
| per-call prices | ModelPrice |
| image and audio ratios | ImageRatio、AudioRatio、AudioCompletionRatio |
| the billing mode with its billing expression | billing_setting.billing_mode、billing_setting.billing_expr |
| Group ratios and usable groups are not affected. | 反向声明：GroupRatio、TopupGroupRatio、GroupGroupRatio、UserUsableGroups 未被写入 |

成功提示（同文件 `:172-176`）改为「Model pricing was restored to the default templates. Group ratios were not changed.」，
与后端范围一致（且 `queryClient.invalidateQueries(['system-options'])` 未变，UI 会刷新）。

七语言核对：

| 核对项 | 结果 |
|---|---|
| 每文件 diff | en / zh / zh-TW / fr / ja / ru / vi 均 `2 insertions, 0 deletions`（纯插入，无重排） |
| 键存在且非空 | 7 个文件都含两个新键，译文非空，索引一致（desc=4237、toast=2525） |
| 插入位置 | 7 个文件邻居一致：desc 介于「The release is missing its signature asset…」与「TPM」之间；toast 介于「Model Pricing」与「Model Regex」之间；`localeCompare` 判定前后序均正确 |
| JSON 合法性 | 7 个文件键数 5319 → 5321（+2），`jq` 解析全部 OK |
| i18n 门禁 | `bun run i18n:check` 通过（4187 引用键 × 7 locale） |
| 陈旧键观察 | 被替换掉的旧键「This will clear custom pricing ratios and revert to upstream defaults.」与「Model prices reset successfully」仍留在 7 个 locale 文件里（本次为纯新增，未删除）；不影响 `i18n:check`，属可选清理项 |

## 六、实现者自曝两点的独立确认（按要求不改）

1. **`model.UpdateOption` 吞掉 `DB.Save` 错误 —— 属实。** `model/option.go:227` 的
   `DB.FirstOrCreate(&option, Option{Key: key})` 与 `:232` 的 `DB.Save(&option)` 均**未接收返回值**，
   函数只把 `updateOptionMap` 的错误返回给调用方。因此「写库失败」不会变成 error，控制器
   `controller/pricing.go:101-107` 的错误分支不会因 DB 失败触发。
   补充精确边界：`updateOptionMap` 仍可能返回解析类错误（`ratio_setting.UpdateXxxByJSONString` 对非法 JSON 报错），
   所以该分支不是严格死代码，只是对本接口的入参（内部 `Marshal` 产物）实际不可达。
2. **后端成功文案未改 —— 属实。** `controller/pricing.go:118` 仍为 `"message": "重置模型倍率成功"`（与修复前一致，
   注释 :114-116 说明是为兼容按文本匹配的调用方而刻意不改）。前端成功提示走自己的 i18n key，不受影响。

## 七、风险与不确定项

1. （行为放大，已披露）重置现在会清掉**所有非默认条目**：除倍率表外，还包括实例自定义的按次价格、
   图片/音频倍率、缓存写入倍率，以及**目录外模型的计费模式与表达式**（默认表达式表只含两个 deepseek 模型）。
   这是 PRD R1 的既定语义，确认框文案已写明「恢复到默认模板」，但点击一次的破坏面比修复前明显更大，
   建议 Lead 在发版说明里提醒操作者。
2. （非阻断）`pricingDefaults` 是 Go map，遍历顺序随机；某个键失败时已写入的键不会回滚（非原子）。
   由于失败只可能来自非法 JSON 而本接口写入的都是内部 `Marshal` 产物，实际风险极低；
   若将来要原子化，`model.UpdateOptionsBulk` 已具备事务语义。
3. （非阻断）7 个 locale 里各残留 2 个已不再被引用的旧键，可在下次 `i18n:sync` 时清理。
4. （非阻断）旧文案「重置模型倍率成功」与新语义不完全匹配（自曝点 ②），本轮按要求不改；
   若前端/调用方有按 message 匹配的逻辑，改动需单独评估。
5. （不确定项）本轮未对真实运行实例点重置（无实例操作授权、红线禁外部主机）；DB/内存行为由内存 SQLite 用例覆盖，
   真实 MySQL/PostgreSQL 下的 `Save` 语义未在本轮实测（该路径未改动）。

## 八、复跑命令清单

```bash
# 修复前后对照（Go）：clone 即修复前代码
rm -rf /tmp/verify-reset && git clone -q /root/new-api-own /tmp/verify-reset
cp /root/new-api-own/controller/pricing_test.go /tmp/verify-reset/controller/
# 仅在 /tmp 克隆内补 4 个只读访问器 shim（见 evidence 正文），取得行为对照
cd /tmp/verify-reset && GOPROXY=off go test ./controller/ -run TestResetModelRatio -count=1 -v

# 真实工作区
cd /root/new-api-own && GOPROXY=off go build ./... && GOPROXY=off go vet ./...
GOPROXY=off go test ./service/... ./controller/... ./setting/... -count=1
go test ./controller/ -run TestResetModelRatio -count=1 -v
cd web && bun run typecheck && bun run i18n:check
bun test src/features/system-settings/
```
