# 按桶删除可回收镜像（C 桶 + B 桶五项目）

## Goal

用户点名授权（2026-10-24）：删除 C 桶全部 + B 桶中 komari / komari-agent / picsur / litepan / postgres
五个项目的旧标签（各保留在用版与上一版）；D 桶保留不动；本项目镜像侧无可删项。

## 事前依据（授权点名，逐条）

- 用户在盘点交付后的选择题中勾选：**「删 C 桶（推荐）」+「删 B 桶（需逐项目点名）」**；
  随后的点名题勾选：**komari、komari-agent、picsur、litepan、postgres 旧标签**。
- 盘点结论（已归档任务 `10-24-image-reclaim-inventory-and-doc-links`）：
  候选 102 个 / 独占层 12.52GB（与 `docker system df` 的 RECLAIMABLE 12.47GB 互相印证）；
  A 本项目 0 个；B 66 个 / 7.59GB；C 22 个 / 3.86GB；D 14 个 / 1.07GB。

## 硬性保留名单（执行时不得删除）

| # | 保留对象 | 理由 |
|---|---|---|
| 1 | 8 个运行中容器当前引用的镜像（动态现取） | 删了会破坏在跑服务 |
| 2 | `ghcr.io/zhemed/new-api-own:v0.0.45` 与 `:latest` | 本项目实例在用 |
| 3 | `ghcr.io/zhemed/komari` 在用版 + 上一版（0.0.24 / 0.0.23） | 回滚点 |
| 4 | `ghcr.io/zhemed/komari-agent` 同上（0.0.24 / 0.0.23） | 回滚点 |
| 5 | `ghcr.io/zhemed/picsur:allinone`（在用）与 `:latest` | 在用 + 回滚标签 |
| 6 | `ghcr.io/zhemed/litepan:0.0.49`（在用）与 `:0.0.50`（待部署） | 在用 + 待部署 |
| 7 | `postgres:15.4` 与 `postgres:17-alpine`（在用） | 在用；只删旧大版本标签如 `postgres:15` |
| 8 | 仓库 Dockerfile 的 FROM 基础镜像 | 构建依赖 |

## Requirements

- R1 删除方式：**逐个按「仓库:标签」`docker rmi`**；禁止按镜像 ID 删、禁止 `docker image prune` /
  `docker system prune` / `-f` 强删；不使用通配。
- R2 待删清单生成后必须先打印（数量 + 桶归属 + 保留判定），与保留名单冲突的一律剔除并说明。
- R3 删除前取基线（`df -B1 /`、`docker system df`、唯一镜像数、逐容器状态），删除后重取并给出**实测回收字节**，
  与估算对比说明偏差原因（共享层）。
- R4 服务无损核验：8 容器全 Up；`new-api` running / `restarts=0` / 挂载 `/root/new-api-own/data:/data` /
  200 + `v0.0.45`；卷仍 3 个；`data/` 55 文件。
- R5 失败项如实记录（返回码 + stderr），不做重试变通；被其它镜像共享层阻挡导致无法删除的，属正常结果。
- R6 红线：只动本机、只删点名对象、不删容器/卷/网络/`data/`、不提交不推送、不访问外部主机。

## Acceptance Criteria（收尾回填 2026-10-24）

- [x] AC1 待删清单执行前打印并校验：C 桶 22 个 + B 桶五项目旧标签 62 个 = **84 个镜像 / 111 个标签**；
      与保留名单零冲突。**执行前拦下一次误判**：首版脚本解析在用镜像 ID 时截断长度写成 5 位（应 12 位），
      导致 `ghcr.io/zhemed/halo:v2.26.1-custom.1`（halo 容器在用）被误列入 C 桶；修正后 halo 回到保留侧，
      范围判定改为防御性默认（只认 C 桶仓库名单与 B 桶五项目，未知仓库一律视为不在授权内）。**未造成任何误删**。
- [x] AC2 逐条 `docker rmi` 返回码全部记录：**111/111 rc=0，失败 0、跳过 0**；未按镜像 ID 删、未用 `prune`、未用 `-f`。
- [x] AC3 实测回收：`df -B1 /` 97761890304 → 88328695808 = **9433194496 B ≈ 8.79 GiB（9.43GB）**；
      `docker system df` Images 17.96GB → **5.276GB**；镜像 110 → **26**（标签行 146 → 35，悬空 0）。
      与估算 11.45GB 差约 2GB，归因共享层与 Build Cache 继续引用。
- [x] AC4 服务无损：8 容器全 running；`new-api` running / `restarts=0` / 挂载 `/root/new-api-own/data:/data` /
      200 + `v0.0.45`；卷 3 个；`data/` 55 文件。Lead 独立复核以上全部一致。
- [x] AC5 保留清单最终状态：18 项逐条命中（new-api-own v0.0.45+latest、komari 0.0.24+0.0.23+latest、
      komari-agent 0.0.24+0.0.23+latest、picsur allinone+latest+0.10/0.10.6、litepan 0.0.49+0.0.50+latest、
      postgres 15.4+17-alpine、halo 在用镜像、Dockerfile 基础镜像）。
- [x] AC6 收尾由 `scripts/trellis-wrapup.sh` 一条命令完成并推送。

## 连带效果与新授权项（需用户决定）

- **Build Cache 可回收量从 85.79MB 跳到 13.26GB**：被删镜像原先共享的缓存层现已无人引用，属镜像删除的连带结果，
  本轮未动。一条 `docker builder prune -f`（只清缓存、不动镜像/容器/卷）即可再释放约 13GB。
- 镜像侧剩余可回收 1.754GB 主要来自 D 桶基础镜像（用户本轮未选删除）。
- 回滚点保留策略：每个 B 桶项目保留「在用版 + 上一版」；更早版本（komari 0.0.4-0.0.22、picsur 0.8.x 等）
  已永久删除，如需回退到更早版本要重新拉取或重建。
- 备份策略仍缺位（`data/` 单点、无异地副本），用户连续四轮未选。

## Notes

- 保留「上一版」的用意是可回滚；如果某项目的历史标签不够整洁（例如同一 ID 多标签），按 ID 去重后保留即可。
- D 桶虽然也无人使用，但属构建/工具镜像，回收收益低且会拖慢后续构建——按用户选择不动。
