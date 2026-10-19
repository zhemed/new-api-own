# 紧急：500 爆炸 + 「今年」改自然年

## 用户报告（2026-10-19）

升级 v0.0.30 后页面出现 **500 错误页**（截图：糟糕！出错了），用户：「你好像没有测试，爆炸了！」
另：「顺便把年挡位看看能不能也改成从 1 月开始」。

## 定位优先级

1. **500 根因**（最高优先）：查后端日志（panic/error）、`/api/rankings` 等接口实际返回、前端错误边界；
   - 怀疑与本轮改动（`service/rankings.go` 的 calendarMonth / 前端标签与图表类型）相关；
2. **今年档位**：改为自然年（今年 1 月 1 日 00:00 → 现在），上一周期=上一年整年（如适用）；
   - 注意与 month 的实现保持一致（`calendarMonth` 模式或扩展为 `calendarYear`）。

## Acceptance Criteria

- [ ] 给出 500 的确切根因与堆栈/日志证据
- [ ] 修复 500，并补上能覆盖它的测试（含真实接口路径的验证）
- [ ] 「今年」= 自然年（跨年边界有测试）
- [ ] Go 与前端门禁全绿；发版并在实例真机核对（不允许再"没测出来"）

## 500 根因与修复（2026-10-19，v0.0.31）

### 根因

上一版把 **`i18n.language` 原值**传给 `formatRankingBucketLabel()`，而应用语言码是 **`zhCN`/`zhTW`/`enUS`**（非合法 BCP-47）：

```
new Intl.DateTimeFormat('zhCN') → RangeError: invalid language tag: zhCN
```

→ React 渲染抛错 → 全局错误页（用户看到的 500）。仓库已有 `toIntlLocale()`，上一版新代码没用它 ✗。

### 为什么没测出来（教训）

成员单测用的是 `'zh'` / `'en'` **字面量**，而应用实际传的是 **`zhCN`** ✗ → 测试全绿但线上崩 ✗。

### 修复（三层）

1. 两处调用点改 `toIntlLocale(i18n.resolvedLanguage || i18n.language)`（与既有 4 处一致）；
2. `formatRankingBucketLabel` 内部候选链 `[toIntlLocale, locale, 'en']` 逐个 try/catch，最后手工 `MM-DD`/`HH:mm` 兜底 —— **任何 locale 都不再抛错**；
3. 回归测试改用**真实语言码**（`zhCN`/`enUS`/`zhTW`/`xx-invalid`/`undefined`）+ i18next 端到端链路。

### 年挡位

`service/rankings.go` 加 `calendarYear`：`year` = **今年 1 月 1 日 00:00 → 现在**，上一周期 = **上一年整年**；桶粒度不变；today/week/month 未动。

### Lead 亲自真机验证（v0.0.31）

用无头浏览器 fetch 登录（带 cookie）→ 导航排行榜：
- `/rankings`（本周）：**无 500** ✓、日期中文 `10月18日` ✓；
- `/rankings?period=month`：**本月高亮** ✓、说明「按模型统计本月的每日 Token 用量」✓、日期中文 ✓、**面积图** ✓、总计 1.68M ✓；
- `/rankings?period=year`：无 500 ✓。
- 截图：`.verify-scratch/v0031-rankings.png`、`-month.png`、`-year.png`。

### 流程改进（重要）

上一版的漏测有两条原因：①测试用假语言码；②我无法登录面板、没做页面级验证 ✗。
本次才发现 **fetch 登录 + cookie** 可以让无头浏览器进入管理页 → 以后**涉及 UI 的改动一律加这一道真机页面校验**。
