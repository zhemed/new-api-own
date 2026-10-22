# 把美元显示全部换成人民币（￥）

## 用户要求（2026-10-22）

「新任务，把我们的美元都换成人民币和￥」

## 待查清（先只读调查）

1. 前端所有出现美元的位置：`$` 前缀、`USD` 字样、`formatQuota`/金额格式化函数、价格页（按 1M tokens 计价）、
   钱包/充值、模型分析「总额度」、日志页消费额、订阅等；
2. 是否已有货币相关系统设置（如 `QuotaPerUnit`、汇率、货币符号）；
3. **换算口径**：只把符号 `$`→`¥`（数值不变）✗，还是按汇率把数值也换算成人民币 ✓ —— **必须问用户**；
4. 后端是否有美元相关文案（i18n、邮件、错误信息）；
5. 涉及范围与回归风险（金额语义、对账、与上游计费的一致性）。

## Acceptance Criteria

- [ ] 列出所有美元显示点（文件:行号）与是否有系统设置可用
- [ ] 向用户确认口径（仅符号 / 按汇率换算、汇率取值来源）
- [ ] 实现后：界面不再出现 `$`/`USD`；数值口径与用户确认一致
- [ ] 七条门全绿；发版；真机截图核对

## 调查结论（2026-10-22，两路只读盘点）

- **前端**：货币显示由设置驱动，`web/src/lib/currency.ts getDisplayMeta()` 按 `quota_display_type` 返回符号；
  43 文件 / 156 处走统一货币层；**8 文件 13 处硬编码美元**绕过它；i18n 每语言 9 个含 `$` 键 + 24–26 个含 `USD` 键。
- **后端**：quota 是无量纲整数（`QuotaPerUnit=500000` = 1 货币单位，默认语义 USD）；
  已内置 `quota_display_type`（USD/CNY/TOKENS/CUSTOM）+ 自定义符号 + 汇率，经 `/api/status` 下发；
  `logger.LogQuota`/`FormatQuota` 输出的日志/错误/通知金额字符串跟随同一开关。

## 口径与实现（用户选择：只换符号、数值不变）

- **CNY 模式会乘 `USDExchangeRate`(7.3)** → 与"数值不变"不符 ✗；
  唯一可行口径 = **CUSTOM + 符号 ¥ + 汇率 1.0** ✓；
- **实例已改**（Lead 执行，`PUT /api/option/` 三键 + 落库复核）：
  `general_setting.quota_display_type=CUSTOM`、`custom_currency_symbol=¥`、`custom_currency_exchange_rate=1.0`；
  旧键 `DisplayInCurrencyEnabled` **不在库中**（否则启动会强制改回 USD/TOKENS）✓；
- **真机验证**：`/dashboard/models` 金额显示 `¥ 1.15`、整页无 `$`；`/pricing` 显示 `¥ 0.3` / `¥ 15`、无 `$` 金额、无 "USD" 字样；
- **代码（v0.0.36）**：Go 默认值改 CUSTOM/¥/1.0 且 logger 统一走 `GetCurrencySymbol`/`GetUsdToCurrencyRate` 真源
  （USD/TOKENS 输出字节级不变）；前端 8 处硬编码改走货币层并新增 `getCurrencySymbol()`；
  i18n 22 键 × 7 语言同步（插值保留、键数净变化 0）；
- **门禁**：Go `go vet/build` + `make test`（40 包 ok）；前端七条门全绿（**383 pass**，373→+10）。

## 保留未改（需用户裁定）

i18n 中 10 个"机制/模式"键保留美元字样，例如 `USD`（显示模式名）、`USD Exchange Rate`、`CNY per USD`、
`Units per USD`、`Conversion rate from USD to your custom currency`——它们描述的是**内部换算机制**，改了会与实现/DB 键不符。
