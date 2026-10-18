# formatTokens 扩档 K/M/B/T + 合并实现 — 交付

## 一、最终实现（`src/lib/format.ts`）

```ts
/** Token magnitudes, largest first, with the digits each one shows. */
const TOKEN_UNITS = [
  { size: 1_000_000_000_000, suffix: 'T', digits: 2 },
  { size: 1_000_000_000, suffix: 'B', digits: 2 },
  { size: 1_000_000, suffix: 'M', digits: 2 },
  { size: 1_000, suffix: 'K', digits: 1 },
] as const

export function formatTokens(tokens: number): string {
  if (tokens === 0) return '-'
  if (!Number.isFinite(tokens)) return '-'

  const abs = Math.abs(tokens)
  if (abs < 1_000) return tokens.toString()

  let unitIndex = TOKEN_UNITS.findIndex((unit) => abs >= unit.size)
  if (unitIndex < 0) unitIndex = TOKEN_UNITS.length - 1

  const digits = TOKEN_UNITS[unitIndex].digits
  let rounded = Number((tokens / TOKEN_UNITS[unitIndex].size).toFixed(digits))

  // Rounding can reach the next magnitude (999.999999 -> "1000.00"). Promote a
  // unit and keep the same decimals, so 999_999 is `1.0M` and 999_999_999 is
  // `1.00B` instead of `1000.0K` / `1000.00M`.
  while (Math.abs(rounded) >= 1_000 && unitIndex > 0) {
    unitIndex -= 1
    rounded = Number((tokens / TOKEN_UNITS[unitIndex].size).toFixed(digits))
  }

  return `${rounded.toFixed(digits)}${TOKEN_UNITS[unitIndex].suffix}`
}
```

## 二、进位处理方式
1. 先按**原档**的位数 `toFixed`；
2. 若结果的绝对值 ≥ 1000 → **升一档**并**沿用原档的小数位**重算，最多升到 T；
3. 这样：`999_999`（K 档 1 位）→ `999.9`→进位→ **`1.0M`**；
   `999_999_999`（M 档 2 位）→ `1000.00`→进位→ **`1.00B`**；
   `999_999_999_999` → **`1.00T`**。
   （这两个例子与你给的 `1.0M` / `1.00B` 完全一致 —— 注意：若沿用"目标档"的位数，`999_999` 会得到 `1.00M`，与你给的示例不符，故此处按"沿用原档位数"实现。）

## 三、去重结果

```
$ grep -rn "function formatTokens" src
src/lib/format.ts:205:export function formatTokens(tokens: number): string {
```
**全仓只剩一处定义** ✓。`features/rankings/lib/format.ts` 里的同名实现已删除；
3 个调用方（`model-leaderboard.tsx`、`models-section.tsx`、`market-share-section.tsx`）
改为 `import { formatTokens } from '@/lib/format'`（`formatShare` 仍从原文件导入）。

## 四、边界输出（实测）

| 输入 | 输出 |
|---|---|
| `0` | `-` |
| `999` | `999` |
| `1000` | `1.0K` |
| `999_999` | **`1.0M`**（进位，不是 `1000.0K`） |
| `1_000_000` | `1.00M` |
| `1_500_000` | `1.50M` |
| `999_999_999` | **`1.00B`**（进位，不是 `1000.00M`） |
| `1_000_000_000` | `1.00B` |
| `1_234_567_890` | `1.23B` |
| `999_999_999_999` | **`1.00T`** |
| `1e12` | `1.00T` |
| `1.5e12` | `1.50T` |
| `999_949` | `999.9K` |
| `999_950` | `1.0M`（四舍五入进位） |
| `-5000` | `-5.0K` |
| `NaN` / `±Infinity` | `-` |

## 五、测试（新增 15）
`src/lib/__tests__/format-tokens.test.ts`：12 条档位/进位表驱动 +
非有限值（NaN/±Infinity → `-`，替代旧实现会输出 `NaNM` 的行为）+
负值（保留符号与量级）+ 边界进位（`999_949 → 999.9K`、`999_950 → 1.0M`）。

## 六、调用方行为变化（需知会）
rankings 页原先用的是**另一套**规则，合并后统一为共享实现，因此：
- `0` 由 `'0'` 变为 `'-'`（`src/lib/format.ts` 的既有约定；`charts.ts` 那层自己把 0 转 `'0'`，未受影响）；
- `<1000` 由 `toLocaleString()`（`"1,234"`）变为 `toString()`（`"1234"`）；
- 大盘数字的小数位由"可变"（如 `≥1e7` 用 1 位）变为**固定**（M/B/T 两位、K 一位）。
若产品希望 rankings 保留千分位，可单独加一个 `formatTokensGrouped`，本次未做。

## 七、七条门
typecheck **0 错误** ｜ lint **0 warnings / 0 errors** ｜ `bun test` **315 pass / 0 fail**（300 → +15）｜
format:check exit 0 ｜ build exit 0 ｜ **knip exit 0** ｜ i18n:check exit 0。
