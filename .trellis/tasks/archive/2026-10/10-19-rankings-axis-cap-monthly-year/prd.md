# 排行榜轴标签确定性上限 + 年挡位按月分桶

## ① x 轴标签：确定性上限 ≤10
- 现状：两处排行榜 bottom 轴只有 `autoHide/autoLimit`，30 桶全有数据时是否稀疏**无法保证**；
- 改为**按索引 stride 抽稀**（复用 `dashboard/lib/charts.ts` 的 `axisLabelStride`，`stride = ceil(桶数/10)`），
  **首桶与末桶必留**，`autoHide` 保留作窄屏第二道防线；
- 抽成**纯函数** `rankingLabelFilter(bucketCount)`（底层 `rankingLabelIndexes`）便于单测；
- 用 vchart 官方 `label.dataFilter`（入参是 AxisItem 数组；**不用** `formatMethod`，其第三参是 formatter）。

## ② 「今年」改为按月分桶
- `service/rankings.go` year 档 `bucketSize: 7*24*3600` → **`30*24*3600`**（≈12–13 点）；
- today/week/month 档不动；
- Go 测试：自然年范围内应得 **12–13 个桶**，且首桶从 **1 月 1 日** 开始。

## 测试
- 前端：30 桶（全有数据）→ 标签 ≤10、首尾都在、除末位外等距；12 桶 → ≤10 且首尾都在；
- 纯函数级断言（不只测 vchart 配置对象）。
- 七条门（前端基线 **359**，只能增）＋ Go `make test`（小写 `makefile`）。

## 写范围
`service/rankings.go`、`web/src/features/rankings/**` + 测试。不 commit/push。
