# 清理前端 lint 警告（零行为变更）

## Goal

把 `bun run lint` 的 **21 warnings / 0 errors 清零**，全部为**语义等价**的机械改写。
任何无法证明等价的改动**跳过**并说明原因（宁留 warning，不改行为）。

## 现状（已实测）

`bun run lint` = 21 warnings / 0 errors，分布：

| 规则 | 条数 | 位置 |
| --- | --- | --- |
| `unicorn(prefer-number-properties)` | 6 | `system-settings/integrations/amount-options-visual-editor.tsx:52,59`、`amount-discount-visual-editor.tsx:62`(×2)、`dashboard/lib/stats.ts:30`(×2) |
| `unicorn(prefer-add-event-listener)` | 7 | `hooks/use-system-config.ts:123,124,128,129`、`components/turnstile.tsx:71`、`ai-elements/prompt-input.tsx:694,1164` |
| `unicorn(prefer-string-slice)` | 4 | `system-settings/utils/json-parser.ts:76`、`system-settings/integrations/utils.ts:56`、`system-settings/models/utils.ts:72`、`dashboard/lib/text.ts:32` |
| `unicorn(prefer-structured-clone)` | 2 | `system-settings/general/channel-affinity/constants.ts:133`、`channels/lib/advanced-custom.ts:343` |
| `unicorn(prefer-array-index-of)` | 1 | `components/data-table/toolbar/bulk-actions.tsx:87` |
| `react(no-danger)` | 1 | `components/layout/components/footer.tsx:246`（**非本轮 5 类**） |

## Requirements

1. **逐条语义等价**。特别注意：
   - `isNaN(x)` ≠ `Number.isNaN(x)`（前者会强制转型）；`isFinite(x)` ≠ `Number.isFinite(x)`（同理）。
     仅当实参**已是 number 类型**时才等价；否则跳过。
   - `substring(a,b)` ≠ `slice(a,b)`：负数与 `a>b` 时行为不同；仅当参数为非负常量/已知非负时才等价。
   - `structuredClone` 与原浅拷贝**只在原意确实是深拷贝**时才等价。
   - `findIndex(pred)` → `indexOf(x)` 仅当谓词是**简单相等比较**时才等价。
   - `onload/onerror` → `addEventListener` 必须同步处理**清理函数**里的移除。
2. 每个改动文件跑 `bun run typecheck` + 相关测试。
3. 收工跑 `bun test`（基线 **213 pass / 0 fail**，不得下降）与 `format:check`。
4. 逐条记录：改了哪些文件、跳过哪些及原因。

## Constraints

- 写范围：`web/src/hooks/use-system-config.ts`、`web/src/features/system-settings/**`、
  `web/src/features/dashboard/lib/*`、`web/src/components/ai-elements/prompt-input.tsx`
  以及**其余 lint 命中文件**；
- **禁止**改 `web/src/features/pricing/components/**`（review-frontend 正在并行修改，避免冲突）；
- 不 commit / push；不碰用户实例；只改 `web/**`。

## Acceptance Criteria

- [ ] `bun run lint` 达到 **0 warnings / 0 errors**，或每一项残留都有明文理由。
- [ ] 所有改动均可证明语义等价；不确定者已跳过并列出原因。
- [ ] `bun test` ≥ 213 pass / 0 fail（不下降）。
- [ ] `bun run typecheck`、`format:check` 通过。
- [ ] 回报含改动文件清单、跳过清单及原因、lint 前后对比、测试结果。
