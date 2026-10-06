# 复查前端改动（团队）

## Goal

作为独立复核方（默认怀疑），核验另一位 agent 本轮前端改动（commit 208483c / 97fd7d6 / 5ad97b7）的四件事：
删除 33 个「knip 判定无引用」的 `web/src/**` 文件是否安全、`web/knip.config.ts` 是否过宽、
`footer.test.tsx` 由 happy-dom 换 jsdom 是否弱化测试、`footer.tsx` 消毒失效安全分支是否有缺陷。

## Requirements

- 复查以「能否被真实引用」为准，不轻信 knip 结论；除常规 import 外必须排查：动态 `import(`、`require(`、
  路由约定（`src/routeTree.gen.ts` 等生成路由树）、组件 registry、`components.json`、Tailwind `content` 配置、
  CSS/HTML 路径引用、字符串常量里的模块路径、测试夹具、`src/**/index.ts` barrel 之间的互相引用。
- 重点复核组：`src/features/home/**`、`src/components/layout/components/**`、`src/hooks/*`、`src/features/*/lib/index.ts`。
- 若发现被删文件仍被使用：立即 `git show 5ad97b7^:<path> > <path>` 恢复，并在汇报中写清依据（文件:行）。
- knip 配置复核：`ignore` 是否掩盖真实死码；`entry` 是否正确、有无漏申报导致误报（误报会诱导后续错误删除）。
- jsdom 复核：与 `git show 0941a4f:web/src/components/layout/components/__tests__/footer.test.tsx`
  逐行对比，确认断言未被放宽、覆盖未减少。
- footer 消毒复核：`DOMPurify.isSupported ? DOMPurify.sanitize(h) : ''` 在真实浏览器（打包后默认导出为已初始化实例）
  与 Node（导出为工厂函数）两种语义下的正确性；是否存在「本应显示 footer 却不显示」的场景。
- 必须实测并保留原始输出：`bun run typecheck`、`bun run lint`、`bun test`、`bun run build`、
  `bun run knip --include files`（均在 `web/` 下执行）。

## Acceptance Criteria

- [ ] 四个问题各给出明确结论（安全/不安全）+ 证据（文件:行），并区分「已验证」与「无法验证」。
- [ ] 33 个删除逐个有「有引用 / 无引用」判定；发现仍被使用的文件已恢复（或给出未恢复的原因）。
- [ ] knip `entry` / `ignore` 逐条点评，指出任何掩盖死码或有误报风险的条目。
- [ ] footer.test.tsx 与 0941a4f 版本逐行 diff 结论（断言是否放宽）。
- [ ] footer.tsx 消毒分支给出浏览器/Node 双语义分析结论。
- [ ] 五条命令的实测结果（exit code + 关键输出）已记录。
- [ ] 向 Lead 提交 ≤25 行中文汇报：四个结论、已修清单、需 Lead 处置清单、命令实测结果、无法验证项。

## Constraints

- 禁止网络动作（禁止 `bun install` / `bunx` 拉新包）；禁止 `git commit` / `push` / `tag`；
  禁止 `git add -A`；禁止在仓库内建 git worktree。
- 写入范围仅 `web/**` 与本任务目录 `.trellis/tasks/10-06-team-audit-frontend/**`。
- 不得触碰 `.local-instance/`，不得重启任何服务（3020 端口演示实例）。

## Notes

- 本轮定位为复核 + 必要的最小恢复；不顺手重构、不扩大范围。
- 共享任务：`task-5`（Team 任务板），领取后在完成时置为 completed。
