# 团队评估并处理剩余开放项

## 开放项来源

- 上一轮团队审查 + 本轮复核留下的：3 个既有失败测试、knip 剩余项与 ignore 掩盖、后端两处"刻意未改"的不一致。
- 用户指示：「你调用团队评估并处理吧」——要**结论 + 落地**，不是再写一份报告。

## 分工（写范围互不重叠）

| 成员 | 共享任务 | 写范围 | 目标 |
|---|---|---|---|
| `fix-frontend-open-items` | task-7 | `web/**` | 处理 3 个失败测试的取向、knip 剩余 8 项、被 ignore 掩盖的 14 个 `ui/**`，收尾后 typecheck/lint/test/build/knip 全绿 |
| `fix-backend-open-items` | task-8 | Go（relay/ dto/ model/ service/ common/）| 评估并处理 multipart 缺 `model` 校验、DTO 非指针标量；能安全改就改+补测试，波及面过大则留痕 |

Lead：复核 diff、全套门禁、提交推送、汇总与遗留。

## 硬约束（全员，沿用前几轮教训）

- 禁止网络动作；禁止访问生产；禁止 git commit/push/tag；**禁止 `git add -A`、禁止在仓库内建 git worktree**；
- 不要动 `.local-instance/`、不要重启任何服务；
- 不许改测试来"消除失败"——除非结论是"测试断言的是已废弃设计"，且必须在汇报里逐条说明依据；
- 禁用 `--no-verify`；发现不确定就标"待确认"，不许猜。

## Acceptance Criteria

- [ ] 3 个失败测试有明确取向（恢复功能或对齐测试）且最终 `bun test` 全绿或逐条说明为何仍失败
- [ ] knip 收尾：剩余项有结论（删除/ignore/留待）、`bun run knip` 的 exit 状态与文档描述一致
- [ ] 后端两处各自给出"已改+测试"或"不改+证据"，并附命令实测
- [ ] Lead 复核、提交、CI 绿、记录归档
