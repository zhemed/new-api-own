# 修复 zhCN 导致排行榜 500 + 年挡位改自然年 — 交付

## 一、根因确认（已复现）
```
$ bun -e "new Intl.DateTimeFormat('zhCN')"
RangeError: invalid language tag: zhCN        ← 应用真实语言码
RangeError: invalid language tag: zhTW / enUS ← 同类
```
应用的语言码是 `zhCN` / `en` / `fr` / `ru` / `ja` / `vi` / `zhTW`（`web/src/i18n/languages.ts`），
其中 `zhCN` / `zhTW` **不是合法 BCP-47**。上一轮我把 `i18n.language` 原值直接传进
`formatRankingBucketLabel()` → 渲染期抛 `RangeError` → 全局错误页（用户看到的 500）。

单测漏掉的原因：只用了 `'zh'` / `'en'` 字面量。**本轮已用真实语言码覆盖。**

## 二、三层防御

### 1) 调用点（2 处）
```diff
-          i18n.language,
+          toIntlLocale(i18n.resolvedLanguage || i18n.language),
```
`models-section.tsx:84`、`market-share-section.tsx:136`；依赖数组补 `i18n.resolvedLanguage`
（与仓库既有 4 处用法完全一致：`system-tasks-panel` / `system-instances-panel` / `api-keys-columns` 等）。

### 2) 函数内兜底（`rankings/lib/format.ts`）
```ts
for (const tag of [toIntlLocale(locale), locale, 'en']) {
  if (!tag) continue
  try {
    return new Intl.DateTimeFormat(tag, options).format(date)
  } catch { /* invalid tag — try the next candidate */ }
}
// 最后兜底：手工 MM-DD / HH:mm，绝不抛错
```
候选顺序 = 归一化后的 tag → 原值 → `'en'` → 手工格式。**任何非法 locale 都不再抛错**。

### 3) 回归测试（`rankings/lib/__tests__/preferences-and-labels.test.ts`）
```
zhCN → 含「月」   enUS → 含 Sep   zhTW → 含「月」   zh-TW → 含「月」
en   → 含 Sep     xx-invalid → 不抛错    undefined → 不抛错
```
外加 **端到端链路**用例：`createInstance().init({ lng: 'zhCN' })` →
`toIntlLocale(resolvedLanguage||language)` === `'zh-CN'` → 标签 `9月26日`（复刻线上真实路径）。

**先写失败用例**：加完测试后先跑，输出正是线上的 `RangeError: invalid language tag: zhCN`，再落修复。

### 顺带核查（未改，均安全）
全仓 `new Intl.*` 变量 locale 的调用点都已是 `toIntlLocale(...)`（system-info 2 处、api-keys 1 处），
`channel-utils.formatRelativeTime` 外层有 try/catch，其余为 `'en-US'` / `undefined` 常量。

## 三、「今年」改自然年（`service/rankings.go`）
- `rankingPeriodConfig` 新增 `calendarYear bool`，`year` 档置 `true`；
- `rankingTimeRange`：`start = time.Date(now.Year(), 1, 1, 0,0,0,0, now.Location())`，`end = now`；
- `previousRankingTimeRange`：上一周期 = **上一个自然年整年**（`1月1日 00:00` ~ `今年1月1日 - 1s`）；
- **桶粒度不变**（仍 `7*24*3600`，测试专门断言 `bucketSize`），不会退化成上千个日点；
- `today` / `week` / `month` 未动。

## 四、测试与门禁

### Go（`service/rankings_test.go`，现 8 个）
- 新增：`year` = 今年 1 月 1 日 00:00 → now；1 月 1 日 00:30 边界；上一周期 = 上一年整年；
  桶粒度仍按周；
- 既有 month（自然月/跨年/上一月）与 `week`/`today` 滚动断言保持通过。

```
$ go build ./...     → OK
$ make test          → exit 0，无 FAIL
```
> 更正上一轮的说法：仓库里**有**测试入口，文件名是小写 `makefile`（`makefile:36 test:`），
> 上一轮用 `-name "Makefile*"` 大小写敏感地找漏了。`make test` 现在确认可用且 exit 0。

### 前端七条门
typecheck **0 错误** ｜ lint **0 warnings / 0 errors** ｜ `bun test` **359 pass / 0 fail**（357 → +2）｜
format:check exit 0 ｜ build exit 0 ｜ **knip exit 0** ｜ i18n:check exit 0。

## 五、边界
只改 `web/src/features/rankings/**` + `service/rankings.go` + 测试。未 commit。
