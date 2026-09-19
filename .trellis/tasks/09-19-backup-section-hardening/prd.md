# 修正备份段两处真缺陷 + 加强三处（含 postgres 就绪判定）

## Goal

评审指出 `MAINTENANCE.md`「数据备份与恢复」段有两处**真会出事**、三处值得加强，并对
`compose.yaml` 的 postgres 就绪判定提出一条机制性异议。逐条核实后全部落地。

## 判定与证据

| # | 判定 | 证据 / 我的补充 |
|---|---|---|
| **必须 1** 物理备份只停应用不停数据库 | ✅ **成立，是我文档里的真缺陷** | 卷里是 Postgres 数据目录，停掉应用只停客户端写入；checkpoint / WAL / autovacuum / bgwriter 照跑，`tar` 出来不是一致快照。最坑的是**它可能起得来**，用一阵子才崩 |
| **必须 2** 卷名用 `${PWD##*/}_pg_data` 推导 | ✅ **成立，且与自己的设计矛盾** | 卷名 = `<项目名>_pg_data`，项目名由 `.env` 的 `COMPOSE_PROJECT_NAME` 决定。本仓库安装脚本**刻意**把项目名固定成 `new-api-own`（正是为了"换目录不换卷"）。本轮 E2E 就是反例：目录 `/tmp/e2e-deploy`、卷 `e2e-napi-test_pg_data` |
| **加强 1** 粒度/保留/演练 | ✅ 成立 | 文件名只到 `%F`，同日重跑互相覆盖；无保留策略；无演练要求 |
| **加强 2** 验证偏弱 | ✅ 成立 | `head -20` 看到 `CREATE TABLE` 只证明"文件像有内容" |
| **加强 3** 逻辑备份用 `-Fc` | ✅ 成立 | 可 `pg_restore -l` 验 TOC、可选择性恢复；纯文本做不到 |
| **附带** `pg_isready` 会对初始化临时实例误报 | ✅ **成立，已实测坐实** | 见下 |

### `pg_isready` 误报的实测（本轮新增证据）

给官方镜像塞一个慢 init 脚本（`select pg_sleep(8)`）拉长初始化窗口后轮询两种探针：

```
第 2 次轮询起：pg_isready(默认/unix socket) = 就绪(0)
                pg_isready(-h 127.0.0.1)     = 未就绪(2)   ← 误报窗口
第 16 次轮询：TCP 探针也通了（真实实例起来了），窗口结束
窗口长度 ≈ 7 秒（14 次 × 0.5s）
```

机制：官方 entrypoint 在数据目录为空时先起一个**只监听 unix socket** 的临时实例跑 init 脚本，
跑完关掉再起真实例。socket 探针无法区分两者。**这正是 B6 加的那个健康检查踩的坑** ——
`depends_on: condition: service_healthy` 会被它提前放行。

## Requirements（已确认全量落地）

- R1 **必须 1**：物理备份改为 `docker compose stop`（停整栈，PG 收到 SIGTERM 干净关闭）→ tar → `up -d`；
  并补充不停机的 `pg_basebackup` 作为一致物理备份的替代。
- R2 **必须 2**：卷名改为从 `.env` 的 `COMPOSE_PROJECT_NAME` 推导（回退目录名）并 `docker volume inspect` 验证；
  附容器实际挂载的权威读法；显式警告"别用目录名推"，并强调用 `docker container inspect` 而非 `docker inspect`。
- R3 **加强 1**：文件名带秒级时间戳；7 份日备 + 4 份周备；写明"没演练过的备份不算备份"+ 每季度真还原一次。
- R4 **加强 2**：验证升级为"还原到一次性容器 + 逐表比对行数"，并给出可直接复制的完整配方。
- R5 **加强 3**：`pg_dump -Fc` + `pg_restore`；说明为何不用 `pg_dump | gzip`。
- R6 **就绪判定**：`compose.yaml` 的 postgres 健康检查强制走 TCP（`-h 127.0.0.1`），并写明原因。
- R7 步骤重排为 0（定卷名）→1（逻辑）→2（物理）→3（文件）→4（验证）→5（保留/演练）→6（恢复）。
- R8 守闸门：提交带锚点、journal、归档。

## Acceptance Criteria

- [x] 备份段两处真缺陷已改写；替换后的配方**真跑过一遍**：`pg_dump -Fc` 产出 116 KB / 34 条 TABLE DATA；
      还原到一次性容器后 **34 张表逐表行数完全一致**
- [x] 卷名推导有反例佐证（本轮 E2E：目录 `/tmp/e2e-deploy` → 卷 `e2e-napi-test_pg_data`）
- [x] `pg_isready` 误报有实测证据（上表），`compose.yaml` 已改为强制 TCP
- [x] `docker compose config` 在有 `.env` 时通过，健康检查命令确为 `pg_isready -h 127.0.0.1 …`
- [x] 本机 `/opt/docker/new-api-own` 应用新健康检查后 postgres 仍 healthy，`komari`/`litepan` 未受影响
- [x] journal + 归档完成

## 范围外

- 不引入备份调度器/cron（文档给的是可复制的配方，调度是运维选择）。
- 不改 `data/`、`logs/` 的备份方式（它们本就不需要停库）。
