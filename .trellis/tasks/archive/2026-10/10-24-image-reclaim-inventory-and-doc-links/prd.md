# 盘点镜像侧可回收量并补文档互链

## Goal

用户指令（2026-10-24）：① **只读**盘点镜像侧「12.47GB 可回收量」的清单与归属，**不删除**；
② 在 `MAINTENANCE.md` 的流程闸门段补一行指向 `scripts/trellis-wrapup.sh`，并把 `--dry-run`
写进闸门指南的自证清单。**备份策略本轮仍不做**（用户连续三轮未选，风险继续挂账）。

## 背景事实

- 上轮清理后 `docker system df` 显示：Images 110 个 / 17.96GB，其中 **RECLAIMABLE 12.47GB（69%）**；
  悬空镜像为 **0**，所以这 12.47GB 是「有标签但没有任何容器在用」的镜像。
- 本机除 `new-api` 外还跑着 komari-agent-test、komari-test、picsur、support-devdb-1、litepan、halo、halodb，
  **这些镜像很可能属于其它项目**——删除前必须分清归属，不能顺手扩大范围（红线第 4 条）。
- 镜像 SIZE 列含共享层，`RECLAIMABLE` 是估算；真实回收量只有在删除后才能精确测量。

## Requirements

- R1 盘点清单（只读命令）：列出所有**未被任何容器（含已停止容器）使用**的镜像：
  `仓库:标签 | 镜像 ID | SIZE | 创建时间 | 是否被容器引用`；按仓库分组，标注归属判断
  （`ghcr.io/zhemed/new-api-own` 属本项目；其它仓库属其它项目）。
- R2 归属与判定依据：对每个候选给出「谁在用/为什么没人用」的证据（例如该仓库当前运行容器用的是哪个标签、
  是否存在同仓库的更新标签）。
- R3 回收量估算必须说明方法与其局限：
  - `docker system df` 的 RECLAIMABLE 口径；
  - 共享层导致的偏差（删除后真实回收量只能实测）；
  - 给出分组小计（本项目 / 其它项目），让用户能分步授权。
- R4 **只读**：本轮**不得**删除任何镜像/容器/卷，不得 `docker rmi`、不得 `prune`；只允许 `docker images`、
  `docker inspect`、`docker ps -a`、`docker system df` 等只读命令。
- R5 文档：`MAINTENANCE.md` 的「流程闸门」一节补一行指向 `scripts/trellis-wrapup.sh`（说明它把
  finish→archive→push 固化为一条命令，避免归档后被闸门拦）；`.trellis/spec/guides/trellis-gate-guide.md`
  的自证清单（第 6 节）加入 `--dry-run` 一项。两处都要与该文件既有风格一致，不重复大段内容。
- R6 红线：只动本机、不访问外部主机、不碰 `data/`；成员不提交不推送（由 Lead 收口）。

## Acceptance Criteria（收尾回填 2026-10-24）

- [x] AC1 候选清单产出：102 个候选（有标签但无任何容器引用；`dangling=0`），逐条含 ID/标签/磁盘/创建时间/桶/引用状态；
      数字与 `docker system df` 对齐（唯一镜像 110、标签行 146、被引用 8）。
- [x] AC2 分组小计与归属：**A 本项目 0 个**；B 在用项目的旧版本 66 个 / 7.59GB unique；
      C 当前无容器项目的镜像与一次性测试标签 22 个 / 3.86GB unique；D 基础与工具镜像 14 个 / 1.07GB unique。
      并给逐桶授权建议（C 风险最低；B 需对 komari/komari-agent/picsur/litepan/postgres 分别点头；
      D 建议保留）。归属细节已核实（picsur 用 `allinone` 而非 `latest`；litepan 候选 `0.0.50` 比在用 `0.0.49` 新）。
- [x] AC3 回收量口径与局限写明：候选 UNIQUE 合计 12.52GB vs `docker system df` RECLAIMABLE 12.47GB（差 0.4%）；
      单个镜像真实释放量介于 UNIQUE 层与 SIZE 之间，精确值只能删除后实测。
- [x] AC4 全程只读已自证：盘点前后 Images/Containers/Volumes/Build Cache/标签行/悬空镜像计数逐项一致；
      Lead 独立复核 `docker system df`、`docker images | grep new-api-own`、各项计数，与证据完全一致。
- [x] AC5 `MAINTENANCE.md`（+2）与 `.trellis/spec/guides/trellis-gate-guide.md`（+2/-1）两处落地；
      并顺带修正指南第 9 节守卫行漏写的「名字匹配到多个目录」分支（以脚本实际行为为准）。
- [x] AC6 收尾用 `scripts/trellis-wrapup.sh` 一条命令完成（含 push），实证「同一次调用可整体通过闸门」。

## 执行记录

- 盘点由 worker-containers 执行（只读，零删除）；文档互链由 worker-docs 执行；Lead 独立抽查数字一致。
- **「会话层闸门在调用边界检查」已被上一任务的真实收尾证实**（`10-23-build-cache-and-wrapup-flow` 用脚本
  finish→archive→push 一次通过、exit 0、推送成功），worker-docs 在总结里标注的「仍未验证」随即作废。
- 删除决策不在本任务内：待用户按桶点名授权后另开动作。
- 备份策略仍缺位（`data/` 单点、无异地副本），用户连续三轮未选，继续挂账。

## Notes

- 删除决策不在本任务内：盘点完成后由用户针对分组点名授权，再另开动作。
- 若发现「本项目镜像」已无可删项（上轮已把 new-api 收敛到 2 个标签），也要如实说明，不要为凑数扩大范围。
