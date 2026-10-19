# 排行榜轴标签确定性上限 + 年挡位按月分桶 — 交付

## 一、抽稀规则（`rankings/lib/axis-labels.ts`，新增）

```ts
export function rankingLabelIndexes(bucketCount: number): number[] {
  if (!Number.isFinite(bucketCount) || bucketCount <= 0) return []
  const count = Math.trunc(bucketCount)
  if (count === 1) return [0]

  const stride = axisLabelStride(count)        // 复用 dashboard/lib/charts.ts：ceil(n / 10)
  const indexes: number[] = []
  for (let index = 0; index < count; index += stride) indexes.push(index)

  if (indexes.at(-1) !== count - 1) {
    indexes.push(count - 1)                    // 末桶（今天）必留
    if (indexes.length > MAX_AXIS_LABELS) indexes.splice(indexes.length - 2, 1)  // 超限则丢倒数第二个
  }
  return indexes
}

export function rankingLabelFilter(bucketCount: number) {
  const keep = new Set(rankingLabelIndexes(bucketCount))
  return (data: unknown[]) => data.filter((_, index) => keep.has(index))
}
```
- **复用** `axisLabelStride` / `MAX_AXIS_LABELS`（模型分析已验证的同一套规则，不另起一套）；
- 两处图表 bottom 轴：`label: { style, dataFilter: rankingLabelFilter(bucketCount), autoHide: true }`
  （`autoHide` 保留作窄屏第二道防线；**没用** `formatMethod` —— 其第三参是 formatter 不是 index）；
- `bucketCount` = **不同 `ts` 的数量**（堆叠图每个桶有多条点）。

## 二、实测标签数（`rankingLabelIndexes`）

| 桶数 | 标签数 | 保留索引 |
|---|---|---|
| **30**（满月） | **10** | `0,3,6,9,12,15,18,21,24,29` |
| 31 | 9 | `0,4,8,12,16,20,24,28,30` |
| 24 | 9 | `0,3,6,9,12,15,18,21,23` |
| **12**（新 year 档） | **7** | `0,2,4,6,8,10,11` |
| 13 | 7 | `0,2,4,6,8,10,12` |
| 7 | 7（全部） | `0…6` |

契约：**≤10**、**首/尾必在**、除末位外**等距**（末位间隔 1 ~ 2×stride）。

## 三、年挡位按月分桶（`service/rankings.go`）
- `year` 的 `bucketSize`：`7*24*3600` → **`30*24*3600`**（满年 `ceil(365/30) = 13` 个桶，落在 12–13）；
- `today` / `week` / `month` **未动**；
- **额外修了一个头尾问题**：桶是按 epoch 对齐的（`FLOOR(created_at/size)*size`），
  自然月/自然年的第一个桶可能**早于范围起点**（"今年"轴会从去年 12 月的日期开始）。
  新增 `clampRankingBucketsToRange(buckets, startTime)` 把首个越界桶改标为范围起点 ——
  无损：该桶内的行本就被 `created_at >= startTime` 过滤，都在范围内。

## 四、测试

### Go（`service/rankings_test.go`，现 10 个）
新增/更新：
1. `TestYearBucketsAreMonthly`：`bucketSize === 30*24*3600`，且满年桶数 **12–13**；
2. `TestClampRankingBucketsToRangeStartsAtTheRangeStart`：首个越界桶 → 范围起点，后续桶不动、tokens 保留；
3. 既有 year/month 自然期间与 week/today 滚动断言的 8 个用例保持通过。

```
$ go build ./...   → OK
$ make test        → exit 0，无 FAIL
```

### 前端（`rankings/lib/__tests__/axis-labels.test.ts`，新增 7 个）
- 30 桶：≤10、首=0、末=29、除末位等距（末位间隔 1~2×stride）；
- 12 桶：≤10、首=0、末=11；
- 1–60 桶**全枚举**：一律 ≤10 且首尾都在；
- 小轴（1、7 桶）全显示；桶数 0 / NaN → 空；
- `rankingLabelFilter(30)` 作用在 30 个轴项上，保留项与索引完全对应。

## 五、门禁
Go：`go build ./...` OK ｜ **`make test` exit 0**
前端七条门：typecheck **0** ｜ lint **0 warnings / 0 errors** ｜ `bun test` **366 pass / 0 fail**（359 → +7）｜
format:check exit 0 ｜ build exit 0 ｜ **knip exit 0** ｜ i18n:check exit 0。

## 六、边界
只改 `service/rankings.go`（+测试）、`rankings/**`（2 组件 + 1 新模块 + 1 新测试）。未 commit。
