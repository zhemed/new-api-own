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

## 执行结果（用户选定：提交 + 发版，且先恢复 CI 收敛）

| 提交 | 内容 |
|---|---|
| `c9aa0e3` | feat(log)：DSN 形态 + 钉连接 + 清理可调度 + 测试 + 文档 |
| `34da3aa` | ci：Release / Electron / GitCode 同步改回**仅手动触发**（收敛恢复）|
| `cb2ca0f` | chore(release)：VERSION → `0.0.4` |

- `main` 已推送到 `cb2ca0f`，工作树干净；
- tag `v0.0.4` 推送后**只触发 1 个 workflow**（`Publish Docker image (Multi-arch)`），构建 **success**；
- GHCR：`0.0.4` / `v0.0.4` / `latest` 同一 digest（`sha256:d159a4df…`），`latest-amd64` 同步更新；
- **生产未改动**：仍运行 `v0.0.3`（上线属另行授权，本轮只发版不上线）。

## 本机演示实例（用户选定：本机试跑）

- 二进制：工作区内 `.local-instance/`（已加入 `.git/info/exclude`，不进仓库），用 v0.0.4 源码 +
  `-ldflags` 注入版本号构建，`/api/status` 自报 `0.0.4`；
- 配置：`LOG_SQL_DSN=memory`、`LOG_CLEANUP_INTERVAL=30s`、`LOG_MEMORY_MAX_ROWS=50`、
  `-log-dir=`（应用日志不落盘）、监听 3020；
- 为便于造演示数据，实例另放宽 `USER_SESSION_ACTIVE_LIMIT / USER_SESSION_ISSUANCE_LIMIT`
  （默认值会把重复登录挡在 `AUTH_SESSION_LIMIT`）；
- 演示证据：启动日志出现三条内存模式提示（含"keeping at most 50 rows, cleaned every 30s"）；
  写 60 次登录审计后日志接口 `total=61`，等一轮 30 秒清理后 `total=50`（上限生效）；进程 RSS ≈ 62 MB；
- 凭据与访问地址**不写入本文件**（只在对话里告知，避免落到持久位置）。

## Acceptance Criteria

- [x] 用户选定范围并执行
- [x] 提交信息带任务锚点，工作树干净
- [x] 镜像构建成功、版本自报一致（未获授权则不上生产）
- [x] 推 tag 的扇出收敛为 1 个 workflow（符合用户先前要求）
- [x] 本机演示实例跑通并给出可复现的裁剪证据
