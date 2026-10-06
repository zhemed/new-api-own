# 弱盘日志改动：提交与发布决策

## 背景

路线 B 已实现并验证完毕（记录见 `.trellis/tasks/archive/2026-10/10-06-weak-disk-log-route/`）：

- `LOG_SQL_DSN` 新增 `memory` / `:memory:` / `sqlite:<path>`（仅日志库生效）
- 内存模式钉连接（`MaxOpen=1 / MaxIdle=1 / ConnMaxLifetime=0`）
- 日志清理接入调度器：内存模式默认 5m + 20 万行上限
- `make test` 38 包 ok；启动冒烟两种形态均通过

## 当前状态（工作区，未提交）

```
 M MAINTENANCE.md
 M common/env.go
 M model/log.go
 M model/main.go
 M service/system_task.go
?? model/log_retention_test.go
?? service/system_task_log_cleanup_test.go
```

分支为 `main`，HEAD = `c00d538`（journal 记录），版本号仍是 `0.0.3`，
远端 tag 只有 `v0.0.2` / `v0.0.3`。

## 待用户决定

| 选项 | 内容 | 影响 |
|---|---|---|
| 提交并推送 | 代码+测试+文档+记录一起提交并 push（版本号不动）| 不触发镜像构建；改动进入仓库历史 |
| 提交 + 发版 | 再抬版本并推 tag | 触发 1 个 workflow（镜像构建），GHCR 出现新版本 |
| 先不提交 | 保持工作区 | 无副作用，但改动未纳入历史 |

## Requirements

- 待选定后填写（提交信息需带 `[task:weak-disk-log-ship]` 锚点；发版需按项目发布流程抬 VERSION）。

## Acceptance Criteria

- [ ] 用户选定范围并执行
- [ ] 若提交：提交信息带任务锚点，`git status` 干净
- [ ] 若发版：镜像构建成功、版本自报一致（未获授权则不上生产）
