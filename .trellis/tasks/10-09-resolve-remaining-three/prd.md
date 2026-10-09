# 彻底解决剩余三项

## 用户指示（2026-10-09）

「仍未处理全部解决掉」——指上一轮明确列出的三项：

1. **knip 全量基线红**（404→398 条；含既有 `ApiTabIcon` 未使用导出）；
2. **`formatTokenVolume`（`mock-stats.ts:65`）0 引用死代码**（改动前即存在，当时按规矩未顺手删）；
3. **2 条 `prefer-structured-clone` 以 disable 收尾**（当时为避免静默改行为而保留）。

## 目标与判定

| 项 | 完成标准 |
|---|---|
| knip | `bun run knip`（全量）**exit 0**；`knip --include files` 仍 0；**不得**用粗暴全局 ignore 换绿——刻意例外必须在 `knip.config.ts` 里按路径 + 写明依据 |
| 死代码 | 删除 `formatTokenVolume`、`ApiTabIcon` 及其它经复核 0 引用的符号；删前逐个 `grep`（含动态 import / 字符串引用） |
| structuredClone | 写一个**语义显式**的 JSON 克隆 helper（丢弃 `undefined` 值键、`Date`→ISO 串、`NaN/Infinity`→`null`，函数/`Map`/`Set` 按 JSON 语义处理或明确拒绝），替换两处 `JSON.parse(JSON.stringify())`，**移除两条 oxlint disable**；helper 补单测覆盖上述边界 |

## 约束

- 不得降低检查强度（例如把 `knip.config.ts` 改成 ignore 全部 `src/**`）；
- 删导出符号前必须确认无动态/字符串引用；不确定者**保留并在 config 写明理由**；
- 六条命令全绿（`bun test` 基线 **224 pass / 0 fail**，不得下降）+ `format:check`；
- 完成后由 Lead 终验、发版、更新实例。

## Acceptance Criteria

- [ ] `bun run knip` exit 0 且有说明（哪些是删除、哪些是按路径例外）
- [ ] `formatTokenVolume` / `ApiTabIcon` 等死代码已删，0 引用复核记录
- [ ] JSON 克隆 helper + 单测；两处 disable 已移除；lint 仍 0/0
- [ ] 六条命令全绿；Lead 终验通过
