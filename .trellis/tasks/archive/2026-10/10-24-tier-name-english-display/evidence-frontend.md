# evidence-frontend：档位名显示层翻译（peak / offpeak → 中文及六语言）

- 任务板：`task-55`（Trellis 任务 `10-24-tier-name-english-display`，PRD 见同目录 `prd.md`）
- 执行者：`worker-docs`；基线 HEAD `43bc237`（2026-10-24）
- 未改后端表达式（`setting/billing_setting/tiered_billing.go` 未动）、未改 Go 文件、未提交、未推送、未动 `data/` 与实例

## 一、结论

档位名现在**只在展示层**翻译：新增 `features/pricing/lib/tier-labels.ts`（映射 + `getTierDisplayLabel`），
两处渲染点（移动端卡片、分档价格表）改用它；**匹配 / 高亮 / 条件摘要仍按原始标签**（有测试反向证明）。
未知标签原样显示、空标签回退 `Default`。9 个新键 + 2 个语义修正键补齐七语言，`i18n:sync` 后混淆键完好，门禁全绿。

> **2026-10-24 修正**：峰谷计费的档位名改用独立键 `Peak hours` / `Off-peak hours`（中文「高峰 / 低谷」），
> 不再复用语义不符的 `Peak`（「峰值」）——详见第十节。

## 二、改动清单

| 文件 | 改动 |
|---|---|
| `web/src/features/pricing/lib/tier-labels.ts` | **新增**：`TIER_LABEL_I18N_KEYS`（12 个内置档位名 → i18n 键）+ `getTierDisplayLabel(label, t)`；空/缺省 → `t('Default')`，未知 → 原样返回，查找时 trim + 小写 + 空白折叠为 `_` |
| `web/src/features/pricing/components/dynamic-pricing-breakdown.tsx` | 仅 3 行：新增 1 行 import；两处 `{tier.label \|\| t('Default')}` → `{getTierDisplayLabel(tier.label, t)}`（:302 移动端、:385 表格）。`normalizeTierLabel` / `matchedTierLabel` / `formatConditionSummary` 一行未动 |
| `web/src/i18n/static-keys.ts` | 登记 12 个映射键（该文件是"不被 `t('...')` 正则捕获的动态键"登记表，见 web/AGENTS.md 3.1）；`i18n:check` 的 static 计数由 455 → 467 |
| `web/src/i18n/locales/{en,zh,zh-TW,fr,ru,ja,vi}.json` | 各 **+9 行**（只插入、不重排）；`Peak` / `Standard` / `Discount` / `Default` 复用既有键，未重复新增 |
| `web/src/features/pricing/components/__tests__/tier-labels.test.ts` | **新增**：映射单测 6 项 |
| `web/src/features/pricing/components/__tests__/dynamic-pricing-breakdown.test.tsx` | **追加** 4 项渲染测试（原 2 项未改） |

## 三、映射与七语言对照

中文 / 繁体：

| 表达式标签 | i18n 键 | zh | zh-TW |
|---|---|---|---|
| `peak` | Peak hours（独立键，2026-10-24 修正） | 高峰 | 高峰 |
| `offpeak` | Off-peak hours（独立键，2026-10-24 修正） | 低谷 | 低谷 |
| `base` | Base（新） | 基础 | 基礎 |
| `standard` | Standard（既有键） | 标准 | 標準 |
| `short` | Short（新） | 短 | 短 |
| `mid` | Mid（新） | 中 | 中 |
| `long` | Long（新） | 长 | 長 |
| `short_output` | Short output（新） | 短输出 | 短輸出 |
| `long_output` | Long output（新） | 长输出 | 長輸出 |
| `mid_context` | Mid context（新） | 中上下文 | 中上下文 |
| `long_context` | Long context（新） | 长上下文 | 長上下文 |
| `discount` | Discount（既有键） | 优惠 | 優惠 |

其余四语言：

| i18n 键 | en | fr | ru | ja | vi |
|---|---|---|---|---|---|
| Peak hours（新） | Peak hours | Heures pleines | Часы пик | ピーク時間 | Giờ cao điểm |
| Off-peak hours（新） | Off-peak hours | Heures creuses | Непиковые часы | オフピーク時間 | Giờ thấp điểm |
| Base（新） | Base | Base | Базовый | ベース | Cơ bản |
| Standard（既有） | Standard | Standard | Стандартный | 標準 | Tiêu chuẩn |
| Short（新） | Short | Court | Короткий | 短 | Ngắn |
| Mid（新） | Mid | Moyen | Средний | 中 | Trung bình |
| Long（新） | Long | Long | Длинный | 長 | Dài |
| Short output（新） | Short output | Sortie courte | Короткий вывод | 短い出力 | Đầu ra ngắn |
| Long output（新） | Long output | Sortie longue | Длинный вывод | 長い出力 | Đầu ra dài |
| Mid context（新） | Mid context | Contexte moyen | Средний контекст | 中コンテキスト | Ngữ cảnh trung bình |
| Long context（新） | Long context | Contexte long | Длинный контекст | 長コンテキスト | Ngữ cảnh dài |
| Discount（既有） | Discount | Remise | Скидка | 特典 | Giảm giá |

注：`Standard` / `Discount` 的既有译文直接复用；`peak` / `offpeak` 起初复用 `Peak` / 新增 `Off-peak`，
复核后改为**独立键** `Peak hours` / `Off-peak hours`（原因见第十节）。既有键 `Peak` 未删除、未改动（可能用在别处），
只是不再被档位映射引用。

## 四、「只改展示」的证据

1. 组件 diff 只有 3 行（1 行 import + 2 处渲染表达式），`normalizeTierLabel(tier.label)`、
   `normalizedMatchedTierLabel`、`formatConditionSummary(tier.conditions, t)` 均未改动。
2. 测试反向证明：同一表达式下传 `matchedTierLabel='base'`（原始标签）时有"已命中"徽标，
   传中文译文 `'基础'` 时**没有**徽标——说明比较用的仍是原始标签，没有被翻译串污染。
3. 未知标签 `my_custom_tier` 渲染结果仍为 `my_custom_tier`（原样，不吞、不空）。

## 五、i18n 流程与结果

- 写入只走临时 `web/scripts/add-missing-keys.mjs`（跑完已删除）：**只插入、不重排**既有键，
  7 个语言各 **+9 行**（`git diff --stat` 每个文件 9 insertions）。
- `bun run i18n:sync` 已跑：混淆键 `footer.new\u0061pi.projectAttributionSuffix` 转义形式仍在（命中 1 处），
  sync 报告 7 个语言的 missing / extras / untranslated 全为 0。
- 12 个映射键登记进 `STATIC_I18N_KEYS`：`i18n:check` 输出 static 计数由 **455 → 467**，
  引用键总数 4187 → **4197**，7 语言齐备。

## 六、测试

```text
bun test src/features/pricing/components/__tests__/tier-labels.test.ts
  (pass) every built-in tier name renders translated in Chinese
  (pass) the same tier names render their English source strings in English
  (pass) an unknown tier name is shown exactly as written
  (pass) an empty tier name falls back to Default
  (pass) case and separators do not defeat the mapping
  (pass) every mapped key exists in all seven locales
  6 pass / 0 fail

bun test src/features/pricing/components/__tests__/dynamic-pricing-breakdown.test.tsx
  (pass) tier prices use the shared symbol + space format          ← 原有用例
  (pass) CNY mode converts the same cells with the same spacing    ← 原有用例
  (pass) built-in tier names render in the active language         ← 徽标级断言：基础 / 峰值，且不含原始 base
  (pass) the same tier names stay English on an English UI
  (pass) a custom tier name is shown exactly as written
  (pass) the matched row is still resolved by the raw tier label   ← 原始标签命中；译文不命中
  6 pass / 0 fail
```

## 七、门禁输出

| 命令 | 结果 |
|---|---|
| `cd web && bun run typecheck` | `TYPECHECK_EXIT=0` |
| `cd web && bun run i18n:check` | `I18N_EXIT=0`：`4036 literal t() keys + 467 static keys = 4197 referenced keys across 7 locales. All referenced keys exist in every locale.` |
| `bunx oxlint`（5 个改动文件） | `Found 0 warnings and 0 errors.`（首轮有 1 条 `unicorn(prefer-set-has)`，已改为 Set 用法） |
| `bun run format:check` | `FORMAT_CHECK_EXIT=0`（1047 文件；`tier-labels.test.ts` 首轮被指出格式，已用 oxfmt 修正） |
| `bun test src/features/pricing/` | **15 pass / 0 fail**（3 文件） |
| `cd web && bun test`（全量） | **420 pass / 0 fail**（63 文件），退出码 0 |

## 八、风险与未完成

- 只覆盖**内置模板**出现的 12 个标签；运营自定义的标签（例如 `"my_peak"`）仍按原文显示——这是刻意行为，
  但意味着"自定义档位名不会被翻译"，如果将来希望可翻译，需要另设机制（例如允许在模板里写 i18n 键）。
- `peak` 复用了既有键 `Peak`（中文为"峰值"而非 PRD 建议的"高峰"）；如需"高峰"需新增独立键并迁移，
  本轮按"优先复用既有键"处理。
- 未知标签的**空白**保留原样（只有已知标签会被 trim 后查找）；若运营写了带空格的标签，展示会保持原样。
- 未做端到端截图验证（需要实例与真实表达式）；本轮为组件级渲染测试 + 单测。

## 九、建议下一步

1. 在实例（v0.0.48 之后）打开「动态计费 → 分档价格表」确认中文显示为"峰值 / 低谷"；
   若产品希望显示"高峰"，我补一个独立键替换 `peak` 的复用。
2. 若希望自定义档位名也可翻译，可讨论"表达式里写 i18n 键 + 前缀约定"的方案（需产品决策）。
3. 后续新增内置模板标签时，记得同步 `TIER_LABEL_I18N_KEYS` 与 `STATIC_I18N_KEYS`（两处一起改）。

## 十、语义修正：`peak` / `offpeak` 改用独立键（2026-10-24 复核后）

**问题**：首版把 `peak` 映射到既有键 `Peak`，中文显示「峰值」。「峰谷计费」的 peak / offpeak 是**时间段**
（peak hours / off-peak hours），中文应为「高峰 / 低谷」——「峰值」是数值峰值，语义不符。

**改动**：

| 项 | 首版 | 修正后 |
|---|---|---|
| 映射 | `peak → Peak`、`offpeak → Off-peak` | `peak → Peak hours`、`offpeak → Off-peak hours` |
| 新增键 | `Off-peak` | `Peak hours`、`Off-peak hours` |
| `STATIC_I18N_KEYS` | `'Peak'`、`'Off-peak'` | `'Peak hours'`、`'Off-peak hours'` |
| 既有键 `Peak` | 复用 | **不动**（locale 条目保留，可能用在别处；只是不再被档位映射引用） |
| 其它标签 | `base/standard/short/mid/long/*_output/*_context/discount` | **保持不变**（`standard` 仍复用 `Standard`、`discount` 仍复用 `Discount`） |

**新键译文（七语言）**：

| 键 | en | zh | zh-TW | fr | ru | ja | vi |
|---|---|---|---|---|---|---|---|
| Peak hours | Peak hours | 高峰 | 高峰 | Heures pleines | Часы пик | ピーク時間 | Giờ cao điểm |
| Off-peak hours | Off-peak hours | 低谷 | 低谷 | Heures creuses | Непиковые часы | オフピーク時間 | Giờ thấp điểm |

**i18n 落盘**：临时脚本只插入、不重排；7 个 locale 现各 **+11 行**（首版 9 + 本次 2）；`bun run i18n:sync` 已跑，
混淆键 `footer.new\u0061pi.projectAttributionSuffix` 转义形式仍在（1 处）。

**测试同步**：`tier-labels.test.ts` 的 zh 期望 `peak → 高峰`、en 期望 `Peak hours` / `Off-peak hours`，
大小写/空白用例同步改为「高峰」；`dynamic-pricing-breakdown.test.tsx` 的徽标断言改为「高峰」，
英文界面断言改为 `Peak hours`。

**修正后门禁（重跑）**：

| 命令 | 结果 |
|---|---|
| `cd web && bun run typecheck` | `TYPECHECK_EXIT=0` |
| `cd web && bun run i18n:check` | `I18N_EXIT=0`：`4036 literal + 467 static = 4197 referenced keys`，7 语言齐备 |
| `bunx oxlint`（改动文件） | `Found 0 warnings and 0 errors.` |
| `bunx oxfmt --check`（改动文件） | `All matched files use the correct format.` |
| `bun test src/features/pricing/` | **15 pass / 0 fail** |
| `cd web && bun test`（全量） | **420 pass / 0 fail**（63 文件） |

**遗留**：首版新增的 `Off-peak` 键现已无引用（仍留在 7 个 locale 中，各 1 行）；与既有 `Peak` 同属
"未被引用的既有条目"，建议后续随 i18n 清理一并处理（本轮按"不动既有键"的要求保留）。
