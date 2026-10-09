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

## 执行结果（2026-10-09，v0.0.14 + v0.0.15）

### 已彻底解决

| 项 | 证据 |
|---|---|
| 3 个真无引用依赖 | 用户授权一次联网 `bun remove`：`@tanstack/react-virtual` / `auto-skeleton-react` / `next-themes` 从 `package.json` + `bun.lock` 清零（无残留）|
| 最后 2 条 `prefer-structured-clone` 抑制 | **彻底移除**：新增 `web/src/lib/json-clone.ts` 的 `cloneJsonValue<T>()`（**显式递归**、零抑制），JSDoc 写死 14 行 JSON 语义契约；两处调用点改用之；23 例单测 + 一次性差分校验（34 个刁钻输入 vs 旧实现）**0 偏差**；全仓抑制与 `JSON.parse(JSON.stringify(...))` 均归零 |
| `formatTokenVolume` / `ApiTabIcon` | ✓ 删除（含连带的未用 `Zap` 导入）；0 引用复核 |
| knip 四类 | `Unused dependencies 6→0`、`devDependencies 2→0`、`Duplicate exports 1→0`、`Configuration hints 2→0`（5 条依赖按"仅被已忽略组件库树引用"逐条登记理由；失效的两条 ignore 删除；`use-dialog.ts` 重复导出真修）|
| 真实隐患（成员发现） | `tsgo -b` **增量构建会漏报**（会给"假绿"）→ `typecheck` / `build:check` 改为 **`--force`** |

### 未解决（按风险判断主动停手，如实记录）

剩 `Unused exports 301` + `Unused exported types 97`（= 398）。原因：knip 只能去掉 `export` 关键字，
其中约 140 条去掉后**文件内也无人用**，必须跨 ~152 文件整段删声明 —— 在并发开发期风险不成比例，
且期间已发生一次"跨文件批量回滚误伤他人文件"（成员如实报告并已修复）。

**成员交的三条经验（值得进 spec）**：
1. `tsgo -b` 是增量构建，逐批校验会漏报未改动文件的错误 → 必须 `--force`；
2. 批量校验应按**错误签名**（文件, 代码, 符号）比对，而非总数（删声明会致行号位移）；
3. 多人并发写时**禁止跨文件批量回滚**（只允许对"本轮刚改过的单个文件" `git checkout --`）。
