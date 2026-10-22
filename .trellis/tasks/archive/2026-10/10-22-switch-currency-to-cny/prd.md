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

## 第二轮（i18n 收尾，v0.0.37）

用户裁定"界面上零美元字样" → 10 个机制/模式类键改中性口径（`基准货币 / 每 1 单位基准货币`）：

| 旧键 | 新键（en） | zh |
|---|---|---|
| `USD` | `Base currency` | 基准货币 |
| `USD Exchange Rate` | `Base Currency Exchange Rate` | 基准货币汇率 |
| `CNY per USD` | `CNY per base currency unit` | 每 1 单位基准货币折合人民币 |
| `Units per USD` | `Units per base currency unit` | 每 1 单位基准货币的额度单位数 |
| `Conversion rate from USD to your custom currency` | `Conversion rate from the base currency to your custom currency` | 从基准货币到自定义货币的换算率 |
| `Real exchange rate between USD and your payment gateway currency` | `…between the base currency and your payment gateway currency` | 基准货币与支付网关货币之间的实际汇率 |
| `e.g. 8 means 1 USD = 8 units` | `…1 base currency unit = 8 units` | 例如 8 表示 1 单位基准货币 = 8 个单位 |
| `e.g., 8 means 8 local currency per USD` | `…per base currency unit` | 例如 8 表示每 1 单位基准货币兑 8 单位本地货币 |
| `How much to charge for each US dollar of balance (Epay)` | `…each base currency unit of balance (Epay)` | 每 1 单位基准货币余额收取多少 |
| `Configure Waffo Pancake … for USD-priced top-ups` | `…for top-ups priced in the base currency` | 以基准货币计价的充值 |

- **API 值未动**（`value: 'USD'`、`z.enum(['USD',...])`、`default('USD')` 均原样）；
- 7 语言同步，键数净变化 0（各 5304 键），零美元命中；
- **Lead 独立复核**：工作区仅 9 文件改动（成员曾因裸替换一度改坏 32 文件 228 处，**已完全还原** ✓）；
  关键标识符完好；`formatCurrencyFromUSD` 等未受影响；七条门 Lead 复跑全绿（383 pass）。

### 仍存在的两处"美元"（待用户裁定）

1. **显示模式选 `Base currency`（API 值 USD）时**：`getCurrencyLabel()` 返回 `'USD'`、符号走 `$` 分支 —— 属该模式自身语义，消除需代码级改动；
2. **Waffo 网关币种输入框**显示网关真实结算币种（默认 USD）—— 改了会误导管理员（网关按美元扣款），故未改。
