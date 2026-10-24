# 对齐证据：本机 `latest` 指向 v0.0.48（task-37）

- 任务：Trellis `10-24-latest-and-fix-flaky-log-tests`；共享任务板 `task-37`；执行者 worker-containers
- 执行时间：2026-10-24T04:26:02Z ～ 04:26:09Z（UTC），全程只访问本机
- 事前依据：task-37 描述（registry 的 `latest` 已由 CI 指到 v0.0.48；本机仍指 v0.0.47）
- 操作性质：**只重打一个标签**（`docker tag`）+ 只读核验；**未** pull、**未**删除任何镜像、**未**动容器/卷/`data/`、
  未提交未推送

## 0. 结论摘要

| 项目 | 结果 |
|---|---|
| 执行 | `docker tag ghcr.io/zhemed/new-api-own:v0.0.48 ghcr.io/zhemed/new-api-own:latest`，exit=0 |
| 对齐结果 | 本机 `latest` 现与 `v0.0.48` **同 ID `e485361d15d4`**（此前指 `bccc31d54ad7` = v0.0.47） |
| 回滚点 | `v0.0.47`(bccc31d54ad7)、`v0.0.46`(95e3c1b97e32)、`v0.0.45`(e114b17532f1) **全部仍在** |
| 镜像计数 | 29 唯一 / 38 标签 / 悬空 0 —— 与操作前**逐项相同**（只换引用，不增删镜像） |
| 实例 | `ghcr.io/zhemed/new-api-own:v0.0.48` / running / restarts=0 / 挂载 `/root/new-api-own/data` |
| 接口 | HTTP 200 + `X-New-Api-Version: v0.0.48` |
| 其它容器 | 8 个全 running |
| 卷 / 数据 | 卷 3（dangling 2）；`data/` 58 文件 —— 与操作前一致 |
| 与 registry 一致 | registry `latest` index digest = `sha256:dc856bf4d0e218c9d6c35be751ec0c977f0ffb77ac4908e7cee741d20f6272f1`（与 v0.0.48 index 相同） |

## 1. 操作前（04:26:02Z）

```console
### 2026-10-24T04:26:02Z PRE baseline
--- new-api-own images ---
ghcr.io/zhemed/new-api-own:v0.0.48 | e485361d15d4 | 2026-10-24 04:19:32 +0000 UTC
ghcr.io/zhemed/new-api-own:latest | bccc31d54ad7 | 2026-10-24 03:45:18 +0000 UTC   <- 旧指向（= v0.0.47）
ghcr.io/zhemed/new-api-own:v0.0.47 | bccc31d54ad7 | 2026-10-24 03:45:18 +0000 UTC
ghcr.io/zhemed/new-api-own:v0.0.46 | 95e3c1b97e32 | 2026-10-24 02:12:57 +0000 UTC
ghcr.io/zhemed/new-api-own:v0.0.45 | e114b17532f1 | 2026-10-23 14:24:51 +0000 UTC
--- counts ---
unique=29  tags=38  dangling=0
--- instance ---
ghcr.io/zhemed/new-api-own:v0.0.48 | running | restarts=0 | /root/new-api-own/data
--- api ---
HTTP/1.1 200 OK
X-New-Api-Version: v0.0.48
--- containers/volumes/data ---
running=8  all=8  volumes=3  dangling_volumes=2  data_files=58
```

## 2. 执行（04:26:09Z）

```console
$ docker tag ghcr.io/zhemed/new-api-own:v0.0.48 ghcr.io/zhemed/new-api-own:latest
tag_exit=0
```

只把 `latest` 引用指向已存在的 `v0.0.48` 镜像；未拉取、未构建、未删除。

## 3. 操作后核验

```console
--- new-api-own images (after) ---
ghcr.io/zhemed/new-api-own:latest | e485361d15d4 | 2026-10-24 04:19:32 +0000 UTC
ghcr.io/zhemed/new-api-own:v0.0.48 | e485361d15d4 | 2026-10-24 04:19:32 +0000 UTC
ghcr.io/zhemed/new-api-own:v0.0.47 | bccc31d54ad7 | 2026-10-24 03:45:18 +0000 UTC
ghcr.io/zhemed/new-api-own:v0.0.46 | 95e3c1b97e32 | 2026-10-24 02:12:57 +0000 UTC
ghcr.io/zhemed/new-api-own:v0.0.45 | e114b17532f1 | 2026-10-23 14:24:51 +0000 UTC

--- ID comparison ---
  latest=e485361d15d4  v0.0.48=e485361d15d4  v0.0.47=bccc31d54ad7  v0.0.46=95e3c1b97e32  v0.0.45=e114b17532f1
  OK latest == v0.0.48
  OK v0.0.47 仍在（回滚点）
  OK v0.0.46 仍在（回滚点）
  OK v0.0.45 仍在（回滚点）

--- counts (unchanged: 29/38/0) ---
unique=29  tags=38  dangling=0

--- instance ---
ghcr.io/zhemed/new-api-own:v0.0.48 | running | restarts=0 | mount=/root/new-api-own/data
HTTP/1.1 200 OK
X-New-Api-Version: v0.0.48

--- containers/volumes/data ---
running=8  all=8  volumes=3  dangling_volumes=2  data_files=58
new-api | Up 3 minutes            komari-agent-test | Up 18 hours
komari-test | Up 16 hours         picsur | Up 4 days (healthy)
support-devdb-1 | Up 4 days       litepan | Up 4 days
halo | Up 4 days (healthy)        halodb | Up 4 days (healthy)

--- registry latest digest (read-only) ---
registry latest index: sha256:dc856bf4d0e218c9d6c35be751ec0c977f0ffb77ac4908e7cee741d20f6272f1
```

## 4. 前后对照

| 指标 | 操作前（04:26:02Z） | 操作后（04:26:09Z） | 判定 |
|---|---|---|---|
| `latest` 指向 | bccc31d54ad7（= v0.0.47） | **e485361d15d4（= v0.0.48）** | 本次目标，达标 |
| `v0.0.48` | e485361d15d4 | e485361d15d4 | 不变 |
| `v0.0.47`（回滚点） | bccc31d54ad7 | bccc31d54ad7 | 仍在 |
| `v0.0.46`（回滚点） | 95e3c1b97e32 | 95e3c1b97e32 | 仍在 |
| `v0.0.45`（回滚点） | e114b17532f1 | e114b17532f1 | 仍在 |
| 唯一镜像 / 标签行 / 悬空 | 29 / 38 / 0 | 29 / 38 / 0 | 不变（无增删） |
| 容器 `new-api` | v0.0.48 / running / restarts=0 | v0.0.48 / running / restarts=0 | 不变 |
| `/api/status` | 200 + v0.0.48 | 200 + v0.0.48 | 不变 |
| 其它 7 容器 | 全 running | 全 running | 不变 |
| 卷 | 3（dangling 2） | 3（dangling 2） | 不变 |
| `data/` | 58 文件 | 58 文件 | 不变 |

## 5. 与 registry 的一致性

registry `latest` 的 OCI index digest = `sha256:dc856bf4d0e218c9d6c35be751ec0c977f0ffb77ac4908e7cee741d20f6272f1`，
与 task-35 记录的 registry `v0.0.48` index digest 相同 → **本机 `latest` 现在与 registry `latest` 同版本**。
（本地 `e485361d15d4` 是本平台 amd64 镜像 config digest，registry 的 `dc856bf4…` 是多架构 index digest，
两者属不同层级，不矛盾。）

## 6. 复原路径

若要把 `latest` 退回上一版：
`docker tag ghcr.io/zhemed/new-api-own:v0.0.47 ghcr.io/zhemed/new-api-own:latest`（同样只改引用）。
实例回滚（若需要）见 task-35 证据：`docker rm -f new-api` + 原参数、镜像换 `v0.0.47` 重建。

## 7. 风险与未完成

- 无功能性风险：实例运行镜像未变（仍是 v0.0.48），本次只改本地标签引用。
- 语义提醒：本机 `latest` 为 amd64 单平台镜像，registry `latest` 为多架构 index；多架构主机重新 pull 时会取对应平台（正常）。
- 未做（按约束）：未 pull、未删镜像、未动容器/卷/`data/`、未提交未推送。
- 本文件留工作区，待 Lead 按闸门收口（锚点 `[task:latest-and-fix-flaky-log-tests]`）。

## 8. 建议下一步

1. Lead 复核可抽查：`docker images | grep new-api-own`（五行，`latest` 与 `v0.0.48` 同 ID）。
2. 观察运行周期后，再决定是否清理 `v0.0.47`/`v0.0.46`/`v0.0.45` 回滚镜像（需用户单独点头）。
3. 本机现在按 README 用 `:latest` 起容器会得到 v0.0.48（本次对齐的实际收益）。
