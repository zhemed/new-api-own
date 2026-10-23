# 把本地实例数据目录迁入工作区

## Goal

用户指令（2026-10-24）：「工作区外？那就是我们负责的容器位置不对，迁移到我们的工作区，到时候维护也方便」。

把本地实例（容器 `new-api`）的数据目录从工作区外的 `/root/data` 迁到工作区内的
`/root/new-api-own/data`，并按 README 记载的唯一部署形态 `-v ./data:/data` 重建容器，
使实例数据与维护动作都落在工作区内（此前的越界读取隐患随之消失）。

## 现状（只读侦察，2026-10-24）

| 项 | 值 |
|---|---|
| 镜像 | `ghcr.io/zhemed/new-api-own:v0.0.45`（保持同版本，不借机升级） |
| entrypoint / workdir | `/new-api` / `/data` |
| 网络 / 重启 | `host` / `always` |
| 日志选项 | `json-file`，`max-size=10m`，`max-file=3` |
| 环境变量 | `LOG_SQL_DSN=memory`、`LOG_MEMORY_MAX_BYTES=200MB`、`LOG_MEMORY_MAX_ROWS=200000`、`LOG_CLEANUP_RETENTION_DAYS=7` |
| 卷 | `/root/data:/data`（待改为 `/root/new-api-own/data:/data`） |
| 数据体积 | 7.0M（`one-api.db` 1.1M + 3 个历史备份约 3.1M + `logs/` 2.8M） |
| git 忽略 | `.gitignore:27` 已有 `/data/` → 迁入后不会被跟踪 |
| 磁盘 | `/` 剩余 123G，充足 |

## Requirements

- R1 **零数据丢失**：迁移前先停容器保证 SQLite 一致；用 `cp -a` 整目录复制（保留属主/权限/时间戳）；
  `/root/data` 原目录**原样保留**作为回滚点，验证通过前不得删除。
- R2 **配置逐项保留**：镜像 tag（`v0.0.45`）、`--network host`、`--restart always`、
  两条 `--log-opt`、四个 `LOG_*` 环境变量、容器名 `new-api` 全部与现状一致；不得引入 compose 等第二种形态。
- R3 **回滚路径**：若新容器异常，`docker rm -f new-api` 后用原挂载 `/root/data:/data` 重建即可恢复；
  回滚命令写进证据文件。
- R4 **验证（缺一不可）**：
  - 容器 `Up`、`restarts=0`、`docker inspect` 的挂载确为 `/root/new-api-own/data`；
  - `GET /api/status` → 200 且 `X-New-Api-Version: v0.0.45`；
  - 业务数据完好：`users` 行数、`tokens` 行数、`options` 中定价相关键的数量与迁移前一致（迁移前后各取一次）；
  - `logs/` 目录与文件仍在；
  - 数据文件属主/权限与迁移前一致。
- R5 **不入库**：确认 `git status` 不出现 `data/`；若出现，补 `.gitignore` 后再继续。
- R6 停机窗口需用户确认（本机实例在此期间不可用），Lead 负责取得确认后再动手。
- R7 全程只访问本机；不推送、不改业务配置、不升级镜像。

## Acceptance Criteria（收尾回填 2026-10-24）

- [x] AC1 迁移前后业务数据计数一致：users `1 → 1`、tokens `1 → 1`、options 总 `14 → 14`、
      定价相关 `5 → 5`（key 列表逐项相同）；复制 gate 另证 53 文件逐个 sha256 清单一致、
      `one-api.db` 两侧 sha256 = `551941…b5283`。
- [x] AC2 容器 running、`RestartCount=0`、Mounts = `/root/new-api-own/data:/data`；
      镜像/entrypoint/workdir/host 网络/restart always/log-opt/四个 `LOG_*` 逐项与迁移前一致，唯一差异是 bind 宿主路径。
- [x] AC3 `GET /api/status` = 200 + `X-New-Api-Version: v0.0.45`；**重启存活验证**后仍为 200 与同版本
      （StartedAt 由 16:59:25 变 17:03:45，证明真实重启）。
- [x] AC4 旧目录处置：用户裁定「验证通过后立即删除」→ 四条删前安全校验（新库健康 / 新目录非空 /
      `realpath` = `/root/data` / 旧库 sha256 与冻结值逐位相同）全 PASS 后删除；删后 `/root/data` 不存在，
      工作区数据 55 文件完好，实例仍 200。**代价：本地回滚点已消失**（见下方 Notes）。
- [x] AC5 `git status` 只有本任务目录未跟踪；`git check-ignore -v data/` 命中 `.gitignore:27:/data/`，实例数据不入库。
- [x] AC6 证据 `.trellis/tasks/10-23-move-instance-data-into-workspace/evidence-migration.md`
      （21.5KB，含前后对照、完整重建命令、原回滚命令、阶段 A/B 全部原始输出与四条删前校验）。

## 收尾记录（2026-10-24）

- 停机窗口约 **19 秒**（16:59:06 停 → 16:59:25 起）；重启验证带来的第二次中断约 2 秒。
- 迁移后写入落点已证：容器启动后旧目录无任何文件被改动，新路径持续产生日志与 db 增长。
- Lead 独立复核：`mount=/root/new-api-own/data:/data`、`restarts=0`、200 + `v0.0.45`、
  `/root/data` 已不存在、工作区 `data/` 6.9M 完好。

## 后续建议（需用户另行授权，本轮未做）

- **备份策略缺失**：回滚点已按裁定删除，当前 `data/` 是唯一数据源（目录内仅 3 个历史 `.bak`，无异地/定时备份）。
  建议开一个小任务做定时快照（例如每日 sqlite `.backup` + 保留 N 份）。
- 可选：清理悬空卷（当前 40 个）、旧镜像标签；对齐 README 部署段落与本次绝对路径表述。

## Notes

- 这是**有停机的服务变更**：需要用户确认窗口；迁移动作由成员执行、Lead 复核证据。
- 迁移后 `/root/data` 与工作区内的 `data/` 会出现两份数据；在用户决定删除旧目录前，
  **运行实例只以工作区内那份为准**，避免出现「双写/双份不一致」的误解。
- 与已归档任务 `10-23-maintain-clean-pending-items` 的关系：那次是为验证临时读了一次工作区外的库；
  本次迁移正是消除该结构性隐患。
