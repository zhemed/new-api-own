# 消耗分布 tooltip 分项为 0 — 复现与修复

## 一、复现（先红后绿）

新增复现测试 `consumption tooltip dimension rows`（`lib/__tests__/consumption-tokens.test.ts`）：
用真实形状数据（同桶两模型：token `1_682_465` / `1_000`）构造 `processChartData`，
按 vchart 的方式把 `dimension.content[0].value(row)` 的返回值塞进 items，再调 `dimension.updateContent`。

**修复前（红）**：
```
(fail) consumption tooltip dimension rows > barSpec: each model row shows its own token count
(fail) consumption tooltip dimension rows > areaSpec: each model row shows its own token count
  '0' !== '1.68M'   actual: "0"   expected: "1.68M"
```
→ 与线上截图完全一致：**总计对、分项 0**（bar/area 两边都错，不止面积图）。

## 二、根因（Lead 的定位方向正确，具体落点如下）

`src/features/dashboard/lib/charts.ts`
- `spec_line.dimension.content[0].value`（原 **:631-640** 区块）
- `spec_area.dimension.content[0].value`（原 **:676-685** 区块）

上一轮我把这两处也改成了 `formatTokenValue(...)` → **返回的是字符串** `'1.68M'`。
但 vchart 的 dimension 流程是：`content[].value` 产出每 series 的原始值 → 交给 `updateContent` 再格式化。
`makeTooltipDimensionUpdateContent()` 内是 `Number(array[i].value) || 0` →
`Number('1.68M')` = **NaN → 0** ✗；而总计读的是 `datum.TimeSum`（独立字段，纯数字）→ 所以总计正确。
**这正是"总计对、分项 0"的机制**，与柱/面积无关（两边都错）。

## 三、修法

`dimension.content[].value` 恢复为**原始数字** `Number(datum?.rawTokens) || 0`（并加注释说明原因），
格式化交给 `updateContent`（它本来就会调用 `formatTokenValue`）。`mark.content` 仍返回格式化字符串（单 series 提示，无聚合）。

## 四、两套 spec 字段对照（修复后完全一致）

| 字段/行为 | `spec_line`（柱状） | `spec_area`（面积） |
|---|---|---|
| data id | `barData` ← `lineValues` | `areaData` ← `areaValues` |
| `yField` | `'Usage'`（= rawTokens） | `'Usage'`（= rawTokens） |
| `rawTokens` | 该模型该桶 `token_used` | 同 |
| `TimeSum` | 该桶 Σ rawTokens | 同 |
| mark tooltip | `formatTokenValue(rawTokens)`（字符串） | 同 |
| **dimension tooltip content** | **`Number(rawTokens)`（原始数字）** | **同** |
| `updateContent` | 每行 `formatTokenValue`；Total 取 `TimeSum` | 同（`collapseOverflow: false`） |
| Other 合并桶 | `rawTokens` 累加后再格式化 | 同 |

## 五、七条门

typecheck **0 错误** ｜ lint **0 warnings / 0 errors** ｜ `bun test` **289 pass / 0 fail**（286 → +3）｜
format:check exit 0 ｜ build exit 0 ｜ **knip exit 0** ｜ i18n:check exit 0。

## 六、仍然建议截图复核
本机无 `canvas`，无法无头渲染 vchart。本次复现测试**忠实模拟了 vchart 的 dimension 取值/格式化链路**
（content → updateContent），比上一轮只看 `mark.content` 更贴近真实路径。
