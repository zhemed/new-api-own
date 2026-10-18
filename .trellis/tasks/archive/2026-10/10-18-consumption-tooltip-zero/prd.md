# 消耗分布 tooltip 分项为 0

## 现象（用户截图）
`10-09 / 总计: 1.16M / deepseek-flash 0` —— 总计对（Token），**每模型那行为 0**；用户当时是**面积图**视图。

## 要求
1. **先写能失败的测试**：用真实形状数据构造 `processChartData`，取**面积图** `spec_area`，
   构造 dimension 行项目（含 `datum`），调用 `dimension.updateContent` / `mark.content` 的 `value`，
   断言**每模型值 === 该模型该桶 token 数**（现在应红），并断言同桶合计 === 总计；**同时覆盖柱状图**。
2. 修到绿：bar/area 两套 `Usage`/`rawTokens`/`TimeSum` 语义一致；**Other 合并桶也按 token**。
3. 七条门全绿（基线 **286**，只能增）。

## 回报
失败输出、根因（文件:行号）、修法、两套 spec 字段对照表、七条门。
