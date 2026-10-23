# 清理悬空卷与旧镜像 + 对齐 README/spec

## Goal

用户指令（2026-10-24）：① 清理悬空卷与不再需要的旧 new-api 镜像标签；② 把 README 部署段与实例实际形态对齐，
并在 spec 里固化「实例数据目录必须在工作区内」这条约定。**备份策略本轮不做**（用户未选）。

## 侦察基线（Lead 只读，2026-10-24）

| 项 | 事实 |
|---|---|
| new-api 镜像标签 | 14 个：`v0.0.2`/`0.0.2`、`v0.0.3`/`0.0.3`、`v0.0.4`、`0.0.5`、`0.0.6`、`v0.0.7`/`0.0.7`/`latest`、`0.0.8`、`0.0.9`、`v0.0.44`、`v0.0.45`（每个显示 302MB，含大量共享层） |
| 在用镜像 | 仅 `ghcr.io/zhemed/new-api-own:v0.0.45`（容器 `new-api`） |
| `latest` 现状 | 指向 `68c9842d16fb`（2026-10-06 的旧构建），已过期 |
| 卷 | 总数 40，其中 `dangling=true` **39 个** ⚠️ **基线更正**：实际为 **37 个匿名卷 + 2 个命名卷**（`halo-gradle-cache`、`picsur_picsur-data`）；Lead 初查只看了列表前 30 条便误判为「全匿名」，执行成员发现后按 R3 停手，未删任何卷 |
| 文档残留 | `README.md` / `MAINTENANCE.md` / `.trellis/spec/` 中无 `/root/data` 引用 |
| Docker | 29.7.2 + Compose v5.4.0（达标）；另注：29.7.2 的 `docker volume prune` 默认只删匿名未用卷，命名卷需显式 `-a` |

## 决策记录（2026-10-24，Lead 裁定）

- **卷清理范围 = 仅 37 个已验证的匿名悬空卷**，采用**显式名单**方式逐个 `docker volume rm`（不依赖 prune 语义）；
  两个命名卷 `halo-gradle-cache`、`picsur_picsur-data` 属其它应用的数据，**保留不动**（红线：不自行扩大范围）。
- **Build Cache（约 24.22GB 可回收）本轮不清**：属新授权项，用户未点名。
- PRD 侦察基线错误由 Lead 负责，已在上表更正并在证据/收尾记录中留痕。

## Requirements

- R1 镜像清理：删除除 `v0.0.45` 以外的所有 `ghcr.io/zhemed/new-api-own` 标签；
  逐个按「仓库:标签」删除，**不得**按镜像 ID 删除（避免连带移除 v0.0.45 仍在用的共享层）；
  不得触碰其它仓库镜像。
- R2 `latest` 对齐：`docker tag ...:v0.0.45 ...:latest`，使 README 的 `:latest` 命令在本机指向当前版本。
- R3 悬空卷清理：先列全并确认全部为匿名卷（出现命名卷即停止并报告），再 `docker volume prune -f`。
- R4 可选：清理未打标签的悬空镜像（先打印清单与数量）。
- R5 保护在跑服务：不得删除任何运行中容器使用的镜像/卷；不得删除 `/root/new-api-own/data`。
- R6 文档：README 部署段补齐「数据目录必须挂工作区内 `./data`」与迁移姿势；
  `.trellis/spec/guides/deployment-single-method.md` 追加「数据目录必须位于工作区内」一节
  （约定 + 理由 + 迁移步骤 + 反例/风险）。不得改动部署形态本身（仍只有 docker run + 公开镜像一种）。
- R7 红线：只动本机、不提交不推送（提交由 Lead 收口）、不升级镜像、不访问外部主机。

## Acceptance Criteria（收尾回填 2026-10-24）

- [x] AC1 `docker images | grep new-api-own` 只剩 `v0.0.45` 与 `latest`，同 ID `e114b17532f1`
      （14 个标签 → 2；12 个旧标签按「仓库:标签」逐个删除，未按镜像 ID 删；另清 5 个悬空镜像）。
- [x] AC2 8 个容器全部 running；`new-api` running、`restarts=0`、挂载 `/root/new-api-own/data:/data`、
      `/api/status` 200 + `X-New-Api-Version: v0.0.45`。
- [x] AC3 悬空卷按裁定收口：37 个**匿名**悬空卷逐个显式 `docker volume rm`，`failed=0`；
      卷总数 40 → 3、匿名残留 0、`dangling=2`（即按裁定保留的两个命名卷 `halo-gradle-cache`、`picsur_picsur-data`）。
      **口径修正**：原 AC 写的「dangling = 0」不成立，因为那 2 个命名卷属其它应用的数据，按红线不自行扩大范围。
- [x] AC4 空间回收有前后对比：`df -h /` 已用 116G → 114G（精确 -2537910272 B ≈ 2.36 GiB）；
      镜像 122 → 110 个（19.64GB → 17.96GB）；本地卷 40 个/1.86GB → 3 个/996.9MB。
- [x] AC5 README（+33/-1）与 `.trellis/spec/guides/deployment-single-method.md`（+68）落地；
      `git diff --stat` 仅含这两处 + 任务目录；`scripts/forbid-extra-deploy-methods.sh` 复跑通过；
      文档内无 `/root/data` 残留、无与实测矛盾的表述。
- [x] AC6 证据齐备：`evidence-docker-cleanup.md`（24KB，含删前复核、37 行显式删除与返回码、删后验收、空间对比）、
      `evidence-docs.md`（8.6KB，含打回点修复与自查）。

## 执行记录

- 镜像与卷的清理由 worker-containers 执行；README/spec 由 worker-docs 执行；Lead 独立复核：
  镜像 2 行同 ID、卷 3 行、8 容器 Up、实例 200、`data/` 完好、README 命令块已改为子 shell 形式。
- **打回一次**：README/spec 迁移步骤原写 `find . -type f`，读者在仓库根执行会把整仓算进去
  （实测 125204 个文件 vs 数据目录 55 个）；已改为 `( cd <目录> && find . … )` 并补警告，spec 同缺陷同修。
- **Lead 侦察偏差留痕**：初查卷列表只看了前 30 条便断言「39 个全为匿名」，实际是 37 匿名 + 2 命名；
  执行成员按 R3 停手并回报，避免了误删其它应用的数据。口径已在上文与证据中更正。
- 未做（需新授权）：Build Cache 清理（约 24.22GB 可回收）；备份策略（用户本轮未选）；MAINTENANCE 三处互链。

## Notes

- 备份策略未做：`data/` 仍是唯一数据源且无异地备份，属已知单点风险，已记入上一任务与 journal。
- 清理动作由成员执行、Lead 复核关键点（至少自己跑一次镜像清单与悬空卷计数）。
