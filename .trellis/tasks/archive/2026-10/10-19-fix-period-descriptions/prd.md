# 修正排行榜周期描述文案

## 问题
两处组件的 `PERIOD_DESCRIPTIONS` 与实际分桶/范围不符：

| 档 | 现有文案 | 实际行为 |
|---|---|---|
| today | Hourly … last 24 hours | ✓ 正确 |
| week | Weekly … past few weeks ✗ | **按天**、**最近 7 天** |
| month | Daily … this month | ✓（models-section 正确；market-share 仍写 "past month" ✗） |
| year | Weekly … past year ✗ | **30 天桶**、**今年（自然年）** |

## 要改
1. `models-section.tsx`：
   - week → `Daily token usage by model across the past 7 days`
   - year → `Monthly token usage by model across this year`
   - today / month 不变
2. `market-share-section.tsx`（自有一份 "Token share by model author" 文案，同样失真）：
   - week → `Daily token share by model author across the past 7 days`
   - month → `Daily token share by model author across this month`（自然月）
   - year → `Monthly token share by model author across this year`
3. 旧键用脚本从 7 个 locale 删除，新键补齐 7 语言；**键数净变化 0**。

## 门
`i18n:check` 通过；前端七条门（基线 **366**，只能增）。

## 写范围
两个组件 + 7 locale。不 commit/push。
