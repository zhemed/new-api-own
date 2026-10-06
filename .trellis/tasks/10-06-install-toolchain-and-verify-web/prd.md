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
