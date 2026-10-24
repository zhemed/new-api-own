# 对齐证据：本机 `latest` 标签指向 v0.0.46（task-23）

- 任务：Trellis `10-24-align-latest-tag-and-upgrade-notes`；共享任务板 `task-23`；执行者 worker-containers
- 执行时间：2026-10-24T02:34:37Z ～ 02:34:59Z（UTC），全程只访问本机
- 事前依据：task-23 描述（Lead 承接上一轮 task-22 的遗留项，用户裁定对齐标签）
- 操作性质：**只重打一个标签**（`docker tag`）+ 只读核验；**未** pull、**未**删除任何镜像、**未**动容器、
  未动卷、未删 `data/`、未提交未推送

## 0. 结论摘要

| 项目 | 结果 |
|---|---|
| 执行 | `docker tag ghcr.io/zhemed/new-api-own:v0.0.46 ghcr.io/zhemed/new-api-own:latest`，exit=0 |
| 对齐结果 | 本机 `latest` 与 `v0.0.46` **同 ID `95e3c1b97e32`**（此前 `latest` 指 `e114b17532f1` = v0.0.45） |
| 回滚点 | `ghcr.io/zhemed/new-api-own:v0.0.45`（`e114b17532f1`，2026-10-23 14:24:51）**仍在** |
| 镜像计数 | 27 唯一 / 36 标签 / 悬空 0 —— 与操作前**逐项相同**（只换标签指向，不增不减镜像） |
| 实例 | `ghcr.io/zhemed/new-api-own:v0.0.46` / running / restarts=0 / 挂载 `/root/new-api-own/data` |
| 接口 | HTTP 200 + `X-New-Api-Version: v0.0.46` |
| 其它容器 | 8 个全 running |
| 卷 / 数据 | 卷 3 个（dangling 2）；`data/` 文件 56 / 核心 4 —— 与操作前一致 |

## 1. 操作前（02:34:37Z）

```console
$ docker images --format '{{.Repository}}:{{.Tag}} | {{.ID}} | {{.CreatedAt}}' | grep new-api-own
ghcr.io/zhemed/new-api-own:v0.0.46 | 95e3c1b97e32 | 2026-10-24 02:12:57 +0000 UTC
ghcr.io/zhemed/new-api-own:latest | e114b17532f1 | 2026-10-23 14:24:51 +0000 UTC   <- 旧指向（= v0.0.45）
ghcr.io/zhemed/new-api-own:v0.0.45 | e114b17532f1 | 2026-10-23 14:24:51 +0000 UTC

unique=27  tags=36  dangling=0
container: ghcr.io/zhemed/new-api-own:v0.0.46 | running | restarts=0 | /root/new-api-own/data
api: HTTP/1.1 200 OK / X-New-Api-Version: v0.0.46
containers: running=8  all=8
volumes: total=3  dangling=2
data: files=56  core=4
```

## 2. 执行（02:34:59Z）

```console
$ docker tag ghcr.io/zhemed/new-api-own:v0.0.46 ghcr.io/zhemed/new-api-own:latest
tag_exit=0
```

该命令只把 `latest` 这个引用指向已存在的 `v0.0.46` 镜像；没有拉取、没有构建、没有删除。

## 3. 操作后核验

```console
--- new-api-own images (after) ---
ghcr.io/zhemed/new-api-own:latest | 95e3c1b97e32 | 2026-10-24 02:12:57 +0000 UTC
ghcr.io/zhemed/new-api-own:v0.0.46 | 95e3c1b97e32 | 2026-10-24 02:12:57 +0000 UTC
ghcr.io/zhemed/new-api-own:v0.0.45 | e114b17532f1 | 2026-10-23 14:24:51 +0000 UTC

--- latest & v0.0.46 same ID? ---
  latest=95e3c1b97e32  v0.0.46=95e3c1b97e32  v0.0.45=e114b17532f1
  OK latest == v0.0.46
  OK v0.0.45 仍在（回滚点）

--- image counts (unchanged: 27/36/0) ---
unique=27  tags=36  dangling=0

--- container ---
ghcr.io/zhemed/new-api-own:v0.0.46 running 0 /root/new-api-own/data

--- api ---
HTTP/1.1 200 OK
X-New-Api-Version: v0.0.46

--- containers / volumes / data ---
running=8  all=8
new-api | Up 18 minutes            komari-agent-test | Up 16 hours
komari-test | Up 15 hours          picsur | Up 4 days (healthy)
support-devdb-1 | Up 4 days        litepan | Up 4 days
halo | Up 4 days (healthy)         halodb | Up 4 days (healthy)
volumes=3  dangling_volumes=2
data_files=56  core=4
```

## 4. 前后对照

| 指标 | 操作前（02:34:37Z） | 操作后（02:34:59Z） | 判定 |
|---|---|---|---|
| `latest` 指向 | e114b17532f1（= v0.0.45） | **95e3c1b97e32（= v0.0.46）** | 本次目标，达标 |
| `v0.0.46` | 95e3c1b97e32 | 95e3c1b97e32 | 不变 |
| `v0.0.45`（回滚点） | e114b17532f1 | e114b17532f1 | 仍在 |
| 唯一镜像 / 标签行 / 悬空 | 27 / 36 / 0 | 27 / 36 / 0 | 不变（无增删） |
| 容器 `new-api` | v0.0.46 / running / restarts=0 | v0.0.46 / running / restarts=0 | 不变 |
| `/api/status` | 200 + v0.0.46 | 200 + v0.0.46 | 不变 |
| 其它 7 容器 | 全 running | 全 running | 不变 |
| 卷 | 3（dangling 2） | 3（dangling 2） | 不变 |
| `data/` | 56 文件 / 核心 4 | 56 文件 / 核心 4 | 不变 |

## 5. 与 registry 的一致性

只读核对（`docker manifest inspect -v`）：

```console
registry latest manifest: sha256:5e8d4e84fe192c1dde6c2a22bef2626ce6366cb3186746c9ad30d0f6eeecb357
```

该 index digest 与上一轮 task-22 记录的 registry `v0.0.46` index digest **完全相同** ——
即 registry 侧 `latest` 与 `v0.0.46` 本就指向同一多架构 index；本机现在把 `latest` 对齐到了
同一版本的本地镜像（本地 ID `95e3c1b97e32` 是本平台 amd64 镜像的 config digest，
与 registry 的多架构 index digest `5e8d4e84…` 属不同层级，二者不矛盾）。

## 6. 回滚 / 复原路径

- 实例回滚（若 v0.0.46 异常）：`docker rm -f new-api` 后按原参数、镜像换 `ghcr.io/zhemed/new-api-own:v0.0.45` 重建。
- 标签复原（若要把 `latest` 退回 v0.0.45）：`docker tag ghcr.io/zhemed/new-api-own:v0.0.45 ghcr.io/zhemed/new-api-own:latest`
  （同样只改引用，不动镜像本体）。

## 7. 风险与未完成

- 无功能性风险：本次只改本地标签引用，实例运行镜像未变（仍是 v0.0.46，来自上一轮升级）。
- 注意：本机 `latest` 与 registry `latest` 现在**版本一致**，但本地镜像是 amd64 单平台产物，
  而 registry 是多架构 index；在多架构主机上重新 pull 才会取到对应平台镜像（属正常行为）。
- 未做（按约束）：未 pull、未删镜像、未动容器 / 卷 / `data/`、未提交未推送。
- 本文件留工作区，待 Lead 按闸门收口（锚点 `[task:align-latest-tag-and-upgrade-notes]`）。

## 8. 建议下一步

1. Lead 复核可抽查：`docker images | grep new-api-own`（三行、`latest` 与 `v0.0.46` 同 ID）与
   `docker inspect -f '{{.Config.Image}} {{.State.Status}}' new-api`。
2. 观察一个运行周期后，再决定是否清理 `v0.0.45` 回滚镜像（需用户单独点头）。
3. 若要在本机按 README 演练 `docker run ...:latest`，现在会正确起 v0.0.46（本次对齐的实际收益）。
