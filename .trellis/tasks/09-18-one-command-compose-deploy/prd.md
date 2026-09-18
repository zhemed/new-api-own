# 适配 komari 的一条命令 compose 部署形态（new-api-own 自用）

## Goal

用户认可 `zhemed/komari` 的 `install-compose.sh` 部署形态（`curl | bash` 一条命令、幂等、拒绝覆盖、
亚秒唯一备份、等健康检查、容器名冲突预检、打印升级/回滚方式），要求**适配到 new-api-own**。
本仓库维护对象是 new-api-own，不是 komari —— 目标是长出一套自己的、与本仓库安全基线不冲突的
一条命令部署。

## 现状（本仓库事实，已核对）

| 项 | 现状 |
|---|---|
| 部署形态 | 仓库根 `docker-compose.yml`：**3 个服务**（new-api / redis / postgres），全部 `network_mode: host` |
| 数据 | `./data:/data`、`./logs:/app/logs` 绑定挂载 + `pg_data` **命名卷**（Postgres 数据） |
| 镜像 | `ghcr.io/zhemed/new-api-own:latest` + `pull_policy: always`；发版由 tag 触发，产出 `vX.Y.Z` / `X.Y.Z` / `latest`（多架构 + cosign） |
| 版本 | `VERSION` = `0.0.3`（不带 `v`），tag = `v0.0.3` |
| 既有脚本 | `install-docker.sh` 只装 Docker 环境标准（29.7.2 + Compose v5.4.0），**不做部署** |
| 缺口 | 没有一条命令的部署脚本；口令硬编码 `123456`；数据目录权限未自动收紧 |
| 安全基线 | `MAINTENANCE.md`「部署安全基线」已写明：面板端口默认暴露、`data/` 含明文上游 Key、容器 root、默认口令必须替换 |
| 仓库边界 | 仓库**公开**：真实口令 / 内网地址 / 运维机路径**不得进仓库**（含 `.trellis/` 的 PRD 与 journal） |
| 闸门 | `.githooks` 已装；提交须带 `[task:<slug>]`；`./scripts/check-trellis-gate.sh` 可自检 |

## Requirements

- **R1 一条命令**：`curl -fsSL https://raw.githubusercontent.com/zhemed/new-api-own/<ref>/install-compose.sh | sudo bash`
  形态，默认装到 `/opt/docker/new-api-own`，`--dir` 可改。
- **R2 幂等 + 保护既有部署**：目标目录已有部署时**拒绝覆盖**，`--force` 才重写；重写前把旧 compose
  备份成 `.bak-<时间戳>-<微秒>`（同一秒内连跑不互相覆盖），只保留最近 3 份；**`data/`、`logs/` 与
  `pg_data` 卷永不被脚本删除或覆盖**。
- **R3 口令自生成且可复用**：首次安装生成强随机 Postgres / Redis 口令写入目标目录 `.env`（600）；
  **重跑必须复用既有 `.env`，绝不重新生成**（库已用旧口令初始化，重生成等于把部署弄挂）。
- **R4 启动前预检**：① Docker 版本 `29.7.2` + Compose `v5.4.0`（不满足则指向 `install-docker.sh`）；
  ② 三个容器名（`new-api` / `redis` / `postgres`）若已被**别的**项目目录占用，在 `compose up` **之前**
  拦下并给出解法（换 `--name-prefix` 或先处理既有容器），不允许起一半才报 Conflict。
- **R5 等健康**：`up -d` 后轮询 new-api 健康检查至 `healthy`（超时上限明确），未通过时给出
  `docker compose logs` / `docker inspect` 的下一步命令，而不是静默返回成功。
- **R6 安全基线落地**：`data`/`logs` 目录 700、`.env` 600；三个服务加 docker 日志轮转上限；
  结束时打印端口暴露面与「部署安全基线」的两条加固选项（不改部署形态，只提示）。
- **R7 打印运维信息**：面板地址、项目目录、数据目录、常用命令、升级方式、回滚方式。
- **R8 参数与环境变量**：`--dir` `--name-prefix` `--tag` `--ref` `--tz` `--force` `--no-start` `--help`
  （同名环境变量可覆盖），`--help` 直接取自脚本头部注释块。
- **R9 干跑**：`--no-start` 只写文件、不碰容器，也不做容器名预检（便于生成文件做对比）。
- **R10 文档同步**：`README.md` / `MAINTENANCE.md` 增加「一条命令部署」入口；发版流程补上
  「安装脚本按 ref 自动取版本」的说明，避免版本字面量漂移。
- **R11 守闸门**：建任务 → start → 提交带 `[task:one-command-compose-deploy]` → journal → 归档。

## Out of Scope

- 不改服务端业务代码、不改 `relaykit`、不改发版流水线。
- **不引入面板内一键升级**：komari 那条线依赖把 `/var/run/docker.sock` 交给容器（≈ 宿主 root），
  与本仓库「部署安全基线」的取向冲突；本仓库的升级路径仍走 `docker compose pull && up -d`。
- **v1 不改网络形态**：沿用 `network_mode: host`（与现有 compose 和线上部署一致）。
  「只绑本地 + 反代」需要同时把 redis/postgres 迁到容器网络并改内部地址（`MAINTENANCE.md` 方案 B），
  留作后续独立任务，v1 只在输出里提示。
- 不在本任务里对任何线上主机执行安装（需用户另行授权）。

## Acceptance Criteria

- [x] **E2E**：临时目录 + 非默认项目名跑通脚本 → 三个容器起来、new-api 到达 `healthy`、
      `GET /api/status` 返回 200
- [x] **幂等**：重跑被拒（退出码 1）并给出解法；`--force` 后目录里只有一个 compose 文件名
- [x] **备份唯一（判别性）**：顺序 6 次 `--force` 留 6 份唯一备份；把备份名退化到秒级**并**去掉
      占位重试循环后只剩 2 份。**本项初稿的断言被实测证伪**：单独去掉亚秒精度并不会丢备份——
      重试循环自己就能防住；真正的缺陷形态是两者同时缺席。断言已按证据改写
- [x] **口令复用**（判别性）：连续 `--force` 后两个口令逐字节不变；用 `.env` 里的口令
      `psql -U root` 与 `redis-cli ping` 均认证通过（口令若被重生成必然失败）
- [x] **撞名预检**：非 compose 的同名容器存在时在启动前被拦下，既有容器未被删改，输出给两条解法
- [x] **口令不外泄**：仓库内 grep 不到生成的口令；脚本输出不打印口令值
- [x] **`bash -n` 与 `shellcheck` 均通过**（shellcheck 退出码 0）
- [x] **不变量**：`--force` 前后 `data/`、`logs/` 文件哈希集合不变；`pg_data` 卷内容与存在性不变
- [x] `README.md` / `MAINTENANCE.md` 有入口；E2E 环境已清理；journal + 归档完成

## 决策记录

- **Q1 目标目录**：用户选 `/opt/docker/new-api-own`（即脚本默认值）。
- **Q2 执行范围**：用户在**本机临时目录**授权 E2E（含 docker 操作与从 ghcr.io 拉镜像）。
  **未对任何真实部署执行安装**，`/opt/docker/new-api-own` 未被创建。
