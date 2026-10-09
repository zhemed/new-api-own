# 实施计划

## 步骤

1. **基线固化**：`bun run knip > /tmp/knip-now.txt`，按"类别 × 目录"归组计数（已得 307/97/6/2/1/2）。
2. **依赖实查**：对 8 个依赖逐条 `grep -rn "<dep>" web/src web/*.ts web/*.json`（含 CSS `@import`），
   记录命中位置 → 归入"由被忽略树引用"或"其它"。
3. **沙箱预演**：复制 `web/`（node_modules 用软链）到 `/tmp`，在同一份基线跑一遍拟采用的机械删除，
   比对结果与真实树差异，确认策略可行后再动真树。
4. **执行删除**（真树，逐文件）：
   - 先处理 barrel 与独立死文件；
   - 再处理 `features/**` 与 `src/lib/**` 的未使用导出；
   - 删除方式见 design §2.2（去 `export` / 删声明）。
5. **R4 必删项**：`formatTokenVolume`（mock-stats.ts）、`ApiTabIcon`（model-details-api.tsx）——先 `grep` 复核 0 引用。
6. **真修项**：`use-dialog.ts` 重复导出；`knip.config.ts` 移除两个失效 `ignoreDependencies`。
7. **配置补全**：为 2.1 类依赖逐条加 `ignoreDependencies` + 理由注释；对"半库"路径（如 `assets/brand-icons/**`）
   按需加路径级 `ignore` **并写明为什么它是刻意保留的产品面**。
8. **验证**：
   - `bun run knip` → 目标 exit 0（若仍有条目，逐条分析是"漏删"还是"应配置"，不得粗暴 ignore）；
   - `bun run knip --include files` → 0；
   - `typecheck` / `lint` / `bun test`（≥224 pass）/ `build` / `i18n:check` / `format:check`。
9. **留痕**：`findings.md` 写分类结果、删除清单、config 变更与每条例外理由、knip 前后对比、命令输出。
10. **回报 Lead**：含"未能归零"的剩余条目（若有）与原因。

## 验证门（每条都必须看到预期结果）

| 门 | 期望 |
|---|---|
| `bun run knip` | exit 0 |
| `bun run knip --include files` | exit 0（无未使用文件） |
| `bun run typecheck` | 0 错误（删除导出的直接证据） |
| `bun run lint` | 0 error |
| `bun test` | ≥ 224 pass / 0 fail |
| `bun run build` | exit 0 |
| `bun run i18n:check` | 全部键七语言齐备 |
| `bun run format:check` | exit 0 |
| 排除文件 | `git diff` 零改动 |

## 回滚点

每完成一个类别（barrel / features / src-lib / config）即跑一次 `typecheck` + `bun test`；
若出现红，先定位是本类改动导致还是既有问题，再决定继续或回退该类。
