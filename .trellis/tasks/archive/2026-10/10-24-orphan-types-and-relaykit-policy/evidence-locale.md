# evidence-locale：清 2 个失效 locale 键（Peak / Off-peak）

- 任务板：`task-58`（Trellis 任务 `10-24-orphan-types-and-relaykit-policy`，PRD 见同目录 `prd.md` 的 R2）
- 执行者：`worker-docs`；基线 HEAD `43bc237`（2026-10-24）
- 改动：`web/src/i18n/locales/` 7 个文件，各 **-2 行**（纯删除，无重排）
- 未提交、未推送；未动 Go 文件、`data/`、实例

## 一、结论

两个键**确认已无任何引用**（含源码字面量、`STATIC_I18N_KEYS`、动态映射、测试断言、其它语义用法），
已从 7 个语言文件删除；`i18n:sync` 后混淆键完好，`i18n:check` 通过，门禁全绿。

## 二、引用核实（删除前，逐项）

| # | 检查 | 命令 | 结果 |
|---|---|---|---|
| 1 | 源码精确字面量 | `grep -rn "'Peak'\|\"Peak\"\|'Off-peak'\|\"Off-peak\"" web/src --include=*.ts --include=*.tsx` | **0 命中** |
| 2 | 静态键登记表 | `grep -n "'Peak'\|'Off-peak'\|Peak hours\|Off-peak hours" web/src/i18n/static-keys.ts` | 只有 `'Peak hours'` / `'Off-peak hours'`（上一轮已替换） |
| 3 | 任何 `peak` 相关出现（排除 locales） | `grep -rn '\bPeak\b\|\bpeak\b' web/src --include=*.ts --include=*.tsx` | 全部是：档位标签字面量 `tier("peak", …)`、新键 `Peak hours`/`Off-peak hours`、以及用新键的测试——**无**对旧键的引用 |
| 4 | 动态 key 映射（可能产出旧键的常量表） | 检查 `TIER_LABEL_I18N_KEYS` 与其它 `Record<string, string>` 标签表 | 映射值为 `Peak hours` / `Off-peak hours`，**不会**产出 `Peak` / `Off-peak` |
| 5 | 测试断言 | `grep -rn "Peak" web/src/features/pricing/components/__tests__/` | 断言用的是 `'高峰'`、`'Peak hours'`、`'Off-peak hours'` |
| 6 | 其它语义（性能/吞吐等） | 全仓 `grep -rn --exclude-dir=node_modules --exclude-dir=dist --exclude-dir=.git --exclude-dir=data --exclude-dir=locales "['\"]Peak['\"]\|['\"]Off-peak['\"]" .` | 唯一命中是**本任务自己的 PRD 文本**（描述用词，不是引用）；后端 `i18n/` 目录无 `Peak` |
| 7 | 文档/spec 引用 | `grep -rn "Peak hours\|Off-peak hours" README.md MAINTENANCE.md .trellis/spec/` | 无（文档不引用具体 i18n 键） |

结论：**无任何真实引用**，可以删除。

## 三、删除方式与结果

- 临时脚本 `web/scripts/remove-dead-keys.mjs`：只 `delete` 这两个键，**不排序、不重排**其它键；跑完已删除
  （`web/scripts/` 只剩 4 个常驻脚本）。
- `bun run i18n:sync` 已跑：混淆键 `footer.new\u0061pi.projectAttributionSuffix` 转义形式仍在（`grep -c` = 1）。

```text
 web/src/i18n/locales/en.json    | 2 --
 web/src/i18n/locales/fr.json    | 2 --
 web/src/i18n/locales/ja.json    | 2 --
 web/src/i18n/locales/ru.json    | 2 --
 web/src/i18n/locales/vi.json    | 2 --
 web/src/i18n/locales/zh-TW.json | 2 --
 web/src/i18n/locales/zh.json    | 2 --
 7 files changed, 14 deletions(-)
```

删除后复核（每语言）：`Peak` = 不存在、`Off-peak` = 不存在、`Peak hours` = 存在。

## 四、计数前后对照

| 指标 | 删除前 | 删除后 |
|---|---|---|
| 每个 locale 的键数 | 5330 | **5328**（各 -2） |
| `i18n:check`：literal t() 键 | 4036 | 4036 |
| `i18n:check`：static 键 | 467 | 467 |
| `i18n:check`：引用键总数 | 4197 | 4197 |
| `i18n:sync` 报告（7 语言） | missing / extras / untranslated 全 0 | 全 0 |

说明：引用类计数**不变**正是本次删除安全的证据——这两个键本来就不在任何引用集合里
（literal + static 都没变），删掉的只是语言文件里的冗余条目。

## 五、门禁输出（删除后）

| 命令 | 结果 |
|---|---|
| `cd web && bun run typecheck` | `TYPECHECK_EXIT=0` |
| `cd web && bun run i18n:check` | `I18N_EXIT=0`：`4036 literal t() keys + 467 static keys = 4197 referenced keys across 7 locales. All referenced keys exist in every locale.` |
| `bunx oxlint -c .oxlintrc.json .` | `Found 0 warnings and 0 errors.`（962 文件；改动文件是 JSON，oxlint 不含 JSON，故跑全仓） |
| `cd web && bun run format:check` | `FORMAT_EXIT=0`（1047 文件） |
| `bun test src/features/pricing/` | **15 pass / 0 fail**（3 文件） |
| `cd web && bun test` | **420 pass / 0 fail**（63 文件） |

## 六、风险与未完成

- 若将来有代码重新引用 `Peak` / `Off-peak`，`i18n:check` 会立刻报缺失（因为两者已不在语言文件中）——
  这是预期保护；届时应改用语义正确的 `Peak hours` / `Off-peak hours`。
- 其它语言文件里仍存在**历史上未引用的键**（本轮只删清单里的 2 个，未做全量死键扫描）；
  如需彻底清理，建议单独一轮用"引用集合差集"的方式批量处理并在证据里给清单。

## 七、建议下一步

1. 如需做**全量死键清理**，我可以先出一份"语言文件有、引用集合无"的差集清单（只报不改），供你圈定范围。
2. 后续新增档位类标签时，继续走 `TIER_LABEL_I18N_KEYS` + `STATIC_I18N_KEYS` 双登记，避免再产生孤儿键。
