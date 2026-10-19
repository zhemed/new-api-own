# 修复 zhCN 语言码导致排行榜 500 + 年挡位改自然年

## 根因（已验证）
`models-section.tsx` / `market-share-section.tsx` 把 **`i18n.language` 原值**传给
`formatRankingBucketLabel()`，该函数直接 `new Intl.DateTimeFormat(locale, …)`。

实测：`new Intl.DateTimeFormat('zhCN')` → **`RangeError: invalid language tag: zhCN`**
（`zhTW` / `enUS` 同样抛错）→ React 渲染抛错 → 全局错误页（用户看到的 500）。
仓库已有 `toIntlLocale()`（`web/src/i18n/languages.ts`，channels/system-info 已在用）。

单元测试漏掉的原因：测试用 `'zh'`/`'en'` 字面量，**不是应用真实语言码**。

## 要修（三层防御）
1. **调用点**：两处改为 `toIntlLocale(i18n.resolvedLanguage || i18n.language)`；
2. **函数内兜底**：`formatRankingBucketLabel` 内 try/catch，失败回退 `'en'`，再失败用手工 `MM-DD`；
   **任何非法 locale 都不得抛错**；
3. **回归测试**：`zhCN` / `enUS` / `zh-TW` / `undefined` / `xx-invalid` 都不抛错且输出合理
   （`zhCN` → 中文、`enUS` → 英文）。

## 同时：年挡位改自然年
- `service/rankings.go`：`year` 改为**自然年**（今年 1 月 1 日 00:00 → 现在，本机时区）；
  上一周期 = **上一个自然年整年**；`today`/`week`/`month` 不动；桶粒度不变。
- Go 测试：今年 1 月 1 日 → now、跨年边界、上一周期 = 上一年整年。

## 门
Go：`go build ./...` + `go test ./...`（仓库测试入口是小写 `makefile`，`make test` 可用）；
前端七条门（基线 **357**，只能增）。

## 写范围
`web/src/features/rankings/**`、`service/rankings.go`、必要测试。不 commit/push。
