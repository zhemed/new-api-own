# 补齐前端工具链并完成验证 + 死码清理

## 用户指示

「本机无 knip 和本机无工具链像这种直接补上就行了」——授权安装缺失工具链，把验证缺口堵住。

## 计划

1. 安装 bun（npm 全局或官方脚本；网络动作已获本轮授权）；
2. `cd web && bun install --frozen-lockfile`（拉取依赖，含 devDeps：tsgo/oxlint/vitest/knip）；
3. 跑 `bun run typecheck` / `bun run lint` / `bun test`（必要时 `bun run build`）；
4. 修掉**真实**类型/lint 错误（尤其团队审查改过的 17 个 TSX 与 7 个 locale）；
5. 跑 knip，据结论决定前端孤儿模块候选的删留（删要能证明无引用）；
6. 复验 + 提交推送 + CI；更新 MAINTENANCE 的「已知不一致」段（若因此消解）。

## Acceptance Criteria

- [ ] bun 与依赖装好，`bun --version`、`web/node_modules` 存在
- [ ] typecheck / lint 结果贴出（错误全修或逐条说明为何不修）
- [ ] knip 结论落地（删除清单或保留理由）
- [ ] 复验通过、提交推送、CI 绿、记录归档

## 执行结果（2026-10-06）

### 补齐的工具链

- `bun 1.4.2`（npm 全局装）；`cd web && bun install --frozen-lockfile` → 1118 包；
- 新增 devDeps：`jsdom`、`@types/jsdom`（为了 XSS 测试真正生效，非随手加）；
- 本机自此可跑：`typecheck` / `lint` / `test` / `build` / `knip`（已写进 MAINTENANCE）。

### 工具链一上来就抓到的真问题

| # | 发现 | 处置 |
|---|---|---|
| 1 | **我们上一轮改出来的 lint error**：`upstream-conflict-dialog.tsx` 的 `useMemo` 用了 `t` 却没进依赖数组 | 已修（`[isMobile, t]`）→ lint 0 error |
| 2 | **footer 的 XSS 测试从未真正验证消毒**：happy-dom 下 `DOMPurify.isSupported=false`，`sanitize()` 原样返回（实测 `<script>` 存活）；两个用例长期失败=零保护 | 改用 jsdom（**断言一行未改**）+ 加 `isSupported` 前置断言 → 3/3 通过；并给 `footer.tsx` 加**失效安全**：不支持时不注入原始 HTML |
| 3 | **knip 配置无效**：没声明应用入口（`index.html`/`main.tsx`）与测试入口 → 把 `main.tsx`、`i18n/static-keys.ts`、`styles/*.css` 全报成"未使用"（65 项里大量误报） | 修 `knip.config.ts`（入口 + `ai-elements` 登记 ignore）→ 误报消失，据可信清单删码 |
| 4 | **33 个确证无引用的前端文件** | 删除（外加两轮引用保险检查 + `bun run build` 兜底验证）；knip 65 → **9** |
| 5 | 3 个 api-key group 表格测试失败 | **既有**（基线同样失败）：组件里 `AutoGroupBadge`/动效环被注释掉，测试仍断言旧设计 → 属产品决策，**不改测试掩盖**，写进 MAINTENANCE 留痕 |

### 我自己的失误（如实记录）

用 `.verify-baseline` 工作树做基线对照后，`git add -A` 把它当成 gitlink **误提交并推送**（`208483c`）。
已 `git rm --cached` + `git worktree remove` + `.gitignore` 加 `.verify-*/`，`97fd7d6` 修复并确认远端干净（该 gitlink 不含文件内容）。

### 验证证据

`bun run typecheck` exit=0；`bun run lint` **0 error**（21 warning 为既有基线）；`bun run build` exit=0；
`bun test` 151 pass / 3 fail（均为上述既有失败）；`bun run knip --include files` 由 65 → 9；
Go 侧不受影响（未改 Go 代码）；CI 通过。
