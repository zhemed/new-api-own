# 处理前端开放项（团队）

## Goal

把前端遗留的三项开放项**处理掉**（不是再写报告）：① 3 个既有失败测试的取向；
② knip 剩余 8 项未使用文件收尾；③ 被 `ignore` 掩盖的 14 个 `src/components/ui/**` 取舍。
收尾时 `web/` 的五条验证命令必须全绿或每项有明文结论。

## Requirements

### R1 三个失败测试的取向（先查史实，再二选一）

- 文件：`web/src/features/keys/components/__tests__/api-key-group-cell.test.tsx`
- 失败断言：`data-auto-group-frame` 计数 `1≠2`、`0≠1`。
- 已知事实：组件 `web/src/features/keys/components/api-key-group-cell.tsx:31,70` 把
  `AutoGroupBadge` 与动效环**注释掉了**；测试仍断言旧设计；该失败在基线 `0941a4f` 同样存在。
- 必须先用 `git log -p --follow` 等查清"为什么被注释"，判断属于：
  - **(a) 临时禁用 → 应恢复**：恢复被注释功能，恢复后 typecheck/build/test 必须全绿，不引入新 lint error；
  - **(b) 设计已简化 → 测试过时**：让测试对齐现状，断言仍须检验**真实用户可见行为**，
    不得放宽成空断言；逐条记录"旧断言断什么 / 现在实际是什么 / 为何这样改"。

### R2 knip 收尾（当前 8 项未使用文件）

- 逐项确证：真死码删除；有约定用途（barrel/index、CLI 约定、registry 引用）的登记 `ignore` 并写明理由。
- 目标：`bun run knip --include files` 无未使用文件，或每一项都有明文结论。
- **必须保留** `web/knip.config.ts` 中那条警告注释：忽略整棵树会导致未使用依赖误报
  （recharts / tokenlens / @xyflow/react / embla-carousel-react / react-resizable-panels）。

### R3 被 ignore 掩盖的 14 个 `src/components/ui/**`

- 先确证是否存在动态/registry 引用（`components.json`、shadcn CLI 约定、样式引用、`@/components/ui/*` 字符串引用）。
- 给出逐文件结论：**保留**（写明理由）或**删除**（确证无引用）。

## Constraints

- 禁止网络动作（不得 `bun install` / `bunx`）；禁止 `git commit` / `push` / `tag`。
- 禁止 `git add -A`；禁止创建 git worktree。
- 只改 `web/**`；不得改动 `.local-instance/`；不得重启任何服务。
- 不得用"改测试消除失败"掩盖真实缺陷；不确定的结论标"待确认"。

## Acceptance Criteria

- [ ] R1：`bun test` 全绿（3 个失败测试全部通过或逐条说明无法修的理由），取向有史实证据支撑。
- [ ] R2：`bun run knip --include files` 无未使用文件，或 8 项各有明文结论；警告注释保留。
- [ ] R3：14 个 `ui/**` 文件逐个给出保留/删除结论及依据。
- [ ] 五条命令实测并记录退出码：`bun run typecheck`、`bun run lint`、`bun test`、
      `bun run build`、`bun run knip --include files`。
- [ ] 改动文件清单明确，且全部位于 `web/**`。

## Notes

- 范围仅前端；后端由其他 teammate 负责（`task-6`），互不覆盖写域。
