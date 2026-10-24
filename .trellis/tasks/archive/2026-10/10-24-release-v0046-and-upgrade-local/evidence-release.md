# 发布证据：v0.0.46 发布 + 本机实例升级（task-22）

- 任务：Trellis `10-24-release-v0046-and-upgrade-local`；共享任务板 `task-22`；执行者 worker-containers
- 执行时间：2026-10-24T02:10:25Z ～ 02:19:19Z（UTC）
- 事前依据：用户点名授权发版与升级本机实例（task-22 描述）；Lead 已完成 `VERSION` 提升提交与附注 tag 创建
- 红线遵守：未 force push；未动其它容器 / 镜像 / 卷；未删 `data/`；未改部署形态（仍是 docker run + 公开镜像单容器）；
  除 GitHub（推送、CI、Release）、ghcr.io（拉镜像）与本机实例外未访问其它主机

## 0. 结论摘要

| 环节 | 结果 |
|---|---|
| 推送 | `main` fast-forward `690c425..fabe59d`；tag `v0.0.46` 新建推送，退出码均 0；`origin/main...main` 变为 `0 0` |
| CI | 两个 workflow **双绿**：`Publish Docker image (Multi-arch)` run **40053075548** success；`Publish release (image + Linux updater binaries)` run **40053075528** success（轮询 11 次、约 5 分钟） |
| Release 资产 | 4/4 齐：`new-api-linux-amd64`(118513848) / `new-api-linux-arm64`(115474616) / `SHA256SUMS`(172) / **`SHA256SUMS.sig`(89)** |
| 镜像 manifest | `ghcr.io/zhemed/new-api-own:v0.0.46` 为 OCI index，含 linux/amd64 + linux/arm64（另 2 个 attestation） |
| 实例升级 | 新容器 `c36343c31c9f`，镜像 `v0.0.46`；2 秒就绪；`X-New-Api-Version: v0.0.46` |
| 修复落地 | `signature_asset_missing` 在运行二进制中出现 **2 次**（要求 ≥1）；镜像内二进制与 Release 资产体积逐字节同长、仅 432 字节差异 |
| 服务状态 | 8 容器全 running；new-api running / restarts=0 / 挂载 `/root/new-api-own/data:/data` / 200 + v0.0.46；卷 3 个；`data/` 核心资产完好 |

## 1. 推送（02:10:36Z）

```console
$ git push origin main
To https://github.com/zhemed/new-api-own.git
   690c425..fabe59d  main -> main
push_main_exit=0

$ git push origin v0.0.46
To https://github.com/zhemed/new-api-own.git
 * [new tag]         v0.0.46 -> v0.0.46
push_tag_exit=0

ahead/behind(origin/main...main)=0	0
origin tag: 2f727abca1b9e87ccbf58f0d21f7ec6e8f5cbd4c	refs/tags/v0.0.46
```

均为普通推送（**无** `--force`）；`main` 为 fast-forward，tag 为新建。

## 2. CI 双绿

推 tag 后 15 秒内两个 workflow 均已 in_progress：

```console
  40053075548  Publish Docker image (Multi-arch)                    in_progress   branch=v0.0.46   event=push
  40053075528  Publish release (image + Linux updater binaries)     in_progress   branch=v0.0.46   event=push
  40053074241  trellis-gate                                         completed success  branch=main
```

轮询（每 25 秒，最多 50 次；实际第 11 次命中，02:15:59Z）：

```console
=== poll 11 at 02:15:59Z ===
  40053075548  Publish Docker image (Multi-arch)                  completed    success    branch=v0.0.46
  40053075528  Publish release (image + Linux updater binaries)   completed    success    branch=v0.0.46
  -> GREEN Publish Docker image (Multi-=success(40053075548) | Publish release (image + Lin=success(40053075528)
watch_exit=0
```

无失败项，未出现需要停止的情况。

## 3. Release 资产核验（关键：签名资产）

```console
$ gh release view v0.0.46 --json tagName,name,isDraft,isPrerelease,publishedAt,assets --jq ...
tag=v0.0.46 name=v0.0.46 draft=false prerelease=false published=2026-10-24T02:14:35Z
  new-api-linux-amd64 | 118513848 | uploaded
  new-api-linux-arm64 | 115474616 | uploaded
  SHA256SUMS | 172 | uploaded
  SHA256SUMS.sig | 89 | uploaded

required assets check:
  OK   new-api-linux-amd64
  OK   new-api-linux-arm64
  OK   SHA256SUMS
  OK   SHA256SUMS.sig
```

`SHA256SUMS` 内容（下载后核对）：

```console
0f21014f163fe711334a2998af826607a4b8a9951b6010ffddc451a5bc3ae639  new-api-linux-amd64
6029ba43fae3a3b9b8c1fbcbd694387e1cd5abc1a80399380597d918d7659cc9  new-api-linux-arm64
```

下载 `new-api-linux-amd64` 后 `sha256sum -c` 校验：`new-api-linux-amd64: OK`
（arm64 未下载，提示 `No such file or directory` 属预期，非失败）。

**`SHA256SUMS.sig` 存在（89 字节）→ 本次修复的核心条件满足**，无需停止。

## 4. 镜像 manifest 核验

```console
$ docker manifest inspect ghcr.io/zhemed/new-api-own:v0.0.46 >/dev/null && echo image-ok
image-ok
schemaVersion: 2 mediaType: application/vnd.oci.image.index.v1+json
  platform=linux/amd64   digest=sha256:5e8d4e84fe192c1dd...
  platform=unknown/unknown digest=sha256:5da2e331074fb6965...（attestation）
  platform=linux/arm64   digest=sha256:83f9b2b1276fba3ef...
  platform=unknown/unknown digest=sha256:e5bf0aae8df81ec49...（attestation）
```

registry 侧标签（只读）：`v0.0.46` 与 `latest` 均指向 manifest `sha256:5e8d4e84fe192c1dde6c2a22bef2626ce6366cb3186746c9ad30d0f6eeecb357`。

## 5. 升级执行（02:16:41Z ～ 02:16:57Z）

```console
$ docker pull ghcr.io/zhemed/new-api-own:v0.0.46
Digest: sha256:95e3c1b97e327d56eb2131b8d2a306d63ed2d3a82cdcdc29e336143cc48d5126
Status: Downloaded newer image for ghcr.io/zhemed/new-api-own:v0.0.46
pull_exit=0

$ docker rm -f new-api          # 未加 -v，数据卷不动
new-api
rm_exit=0

$ docker run -d --name new-api --restart always --network host --log-opt max-size=10m --log-opt max-file=3 \
    -v /root/new-api-own/data:/data -e LOG_SQL_DSN=memory -e LOG_MEMORY_MAX_BYTES=200MB \
    -e LOG_MEMORY_MAX_ROWS=200000 -e LOG_CLEANUP_RETENTION_DAYS=7 ghcr.io/zhemed/new-api-own:v0.0.46
c36343c31c9f2ea27acd6f47ba67cff31994032470f8d99e4866a12c5486a5a6
run_exit=0

--- readiness poll (max 60s) ---
  t+1s http_code=000
  t+2s http_code=200
final_http_code=200
```

停机窗口约 14 秒（02:16:41 停旧 → 02:16:55 新容器 StartedAt），就绪耗时 2 秒。

## 6. 升级前后配置对照（逐项）

| 参数项 | 升级前（v0.0.45） | 升级后（v0.0.46） | 判定 |
|---|---|---|---|
| 镜像 | ghcr.io/zhemed/new-api-own:v0.0.45 | ghcr.io/zhemed/new-api-own:v0.0.46 | **预期变化（本次目标）** |
| 网络 | host | host | 一致 |
| 重启策略 | always / MaximumRetryCount=0 | always / MaximumRetryCount=0 | 一致 |
| 日志 | json-file max-size=10m max-file=3 | 同左 | 一致 |
| 挂载 | /root/new-api-own/data:/data | /root/new-api-own/data:/data | 一致 |
| 环境变量 | LOG_SQL_DSN / LOG_MEMORY_MAX_BYTES / LOG_MEMORY_MAX_ROWS / LOG_CLEANUP_RETENTION_DAYS | 同左（四项逐字相同） | 一致 |
| 容器名 | new-api | new-api | 一致 |
| RestartCount | 0 | 0 | 一致 |

除镜像 tag 外逐项一致，部署形态未变（仍为单容器 docker run + host 网络）。

## 7. 升级后核验

```console
--- instance ---
Image=ghcr.io/zhemed/new-api-own:v0.0.46
Status=running  RestartCount=0
Mounts=/root/new-api-own/data->/data
StartedAt=2026-10-24T02:16:55.495377723Z
container_id=c36343c31c9f2ea27acd6f47ba67cff31994032470f8d99e4866a12c5486a5a6
--- api ---
HTTP/1.1 200 OK
X-New-Api-Version: v0.0.46
--- containers ---
running=8  all=8
new-api | ghcr.io/zhemed/new-api-own:v0.0.46 | Up 2 minutes
komari-agent-test | ghcr.io/zhemed/komari-agent:latest | Up 16 hours
komari-test | ghcr.io/zhemed/komari:latest | Up 14 hours
picsur | ghcr.io/zhemed/picsur:allinone | Up 4 days (healthy)
support-devdb-1 | postgres:17-alpine | Up 4 days
litepan | ghcr.io/zhemed/litepan:v0.0.49 | Up 4 days
halo | ghcr.io/zhemed/halo:v2.26.1-custom.1 | Up 4 days (healthy)
halodb | postgres:15.4 | Up 4 days (healthy)
  （逐个 State.Status 均为 running）
--- volumes --- total=3  dangling=2（halo-gradle-cache / picsur_picsur-data / support_db-data）
--- data dir --- files_total=56  logs=52  core=4
--- images (new-api-own) ---
ghcr.io/zhemed/new-api-own:v0.0.46 | 95e3c1b97e32 | 6 minutes ago  | 302MB
ghcr.io/zhemed/new-api-own:latest | e114b17532f1 | 12 hours ago    | 302MB
ghcr.io/zhemed/new-api-own:v0.0.45 | e114b17532f1 | 12 hours ago    | 302MB
--- images total --- unique=27  tags=36  dangling=0
```

### 7.1 修复在运行二进制里

容器内**没有 `strings`（binutils 未安装）**，原命令 `strings /new-api | grep -c ...` 会因
`sh: 1: strings: not found` 得到 0 命中（假阴性）。改用等价只读命令：

```console
$ docker exec new-api sh -c "grep -a -c signature_asset_missing /new-api"
2
```

**命中 2 次（要求 ≥1）→ 修复确实在运行中的二进制里。** 旁证：

```console
$ docker exec new-api sh -c "grep -a -c signature_mismatch /new-api"
2
$ docker exec new-api stat -c '%s bytes' /new-api
118513848 bytes        # 与 Release 的 new-api-linux-amd64 资产同长
```

### 7.2 `data/` 文件数 55 → 56 的说明（如实解释，非数据异常）

- 核心资产 4 个全部在：`one-api.db`(1163264) + 3 个历史备份（1126400 / 1044480 / 1044480）。
- 增量来自**新容器启动时新建的日志文件** `logs/oneapi-20261024021655.log`（02:17:10 生成），
  以及旧日志 `logs/oneapi-20261023170345.log` 在切换时被续写；日志文件数 51 → 52。
- 即：55 → 56 是运行实例正常写日志的结果，不是数据损坏或丢失。

### 7.3 卷清单

```console
halo-gradle-cache / picsur_picsur-data / support_db-data
total=3  dangling=2
```

## 8. 额外强校验（超出任务要求，用于确认「发布物 == 镜像内二进制」）

| 校验 | 结果 |
|---|---|
| Release 资产完整性 | 下载 `new-api-linux-amd64` 后按 `SHA256SUMS` 校验：`OK`（sha256 = 0f21014f163f…） |
| 容器内二进制 sha256 | 8cd525470185f4369e06505ac0b12a0ea3c062701cbd17a552fe6c96e23feda7 |
| 两者 sha256 是否相同 | **不同**（但体积逐字节同长：均 118513848 B） |
| 逐字节差异 | 差异 **432 字节 / 118513848（0.0004%）**，分布在 **10 段短区**（最长 141 字节） |
| 结论 | 镜像构建 job 与 updater 二进制构建 job 是**两次独立构建**，Go 产物非字节可复现（内嵌 build id/路径等定长标识不同）；代码同源 → 体积同长、差异仅 0.0004%、两者都各含 `signature_asset_missing` 2 次与 `signature_mismatch` 2 次 |

即：Release 资产与镜像内二进制**不是同一字节流**（不能拿资产的 sha256 去校验镜像里的文件，反之亦然），
但两者代码同源、功能一致；本次验收要求的「运行时版本 = v0.0.46」与「修复字符串在运行二进制中」两条均已直接验证。

## 9. 回滚路径（未触发）

若 v0.0.46 出现异常，按下述命令回滚到 v0.0.45（本地镜像与数据均在）：

```console
$ docker rm -f new-api
$ docker run -d --name new-api --restart always --network host --log-opt max-size=10m --log-opt max-file=3 \
    -v /root/new-api-own/data:/data -e LOG_SQL_DSN=memory -e LOG_MEMORY_MAX_BYTES=200MB \
    -e LOG_MEMORY_MAX_ROWS=200000 -e LOG_CLEANUP_RETENTION_DAYS=7 ghcr.io/zhemed/new-api-own:v0.0.45
$ curl -s -o /dev/null -w '%{http_code}\n' http://127.0.0.1:3000/api/status   # 期望 200
```

回滚前提：`ghcr.io/zhemed/new-api-own:v0.0.45`（e114b17532f1）仍在本地（已确认），
`/root/new-api-own/data` 未受影响（已确认）。

## 10. 风险与未完成

- **本地 `latest` 标签仍指向 v0.0.45**（e114b17532f1）：registry 侧 `latest` 已由 CI 更新为 v0.0.46
  （manifest `sha256:5e8d4e84…`），但本机 `latest` 未同步。任务未要求改动，我**未**改；
  影响：若在本机用 README 的 `:latest` 命令且不 pull，会起 v0.0.45。建议 Lead 决定是否本地
  `docker tag ...:v0.0.46 ...:latest`（一条只读性很低的操作，但属未授权变更，故留给 Lead）。
- `SHA256SUMS.sig` 的**签名验证**未做（需要公钥与 cosign/gpg 环境，超出本任务范围）；
  本次仅确认资产存在（89 字节）且 `SHA256SUMS` 与下载物一致。
- 面板「立即更新」端到端演练未做（属用户侧/后续任务；本任务只保证发布物与实例版本正确）。
- `data/` 文件数由 55 变 56（新启动日志，见 7.2），核心资产完好。
- 未提交：本文件留在工作区，待 Lead 按闸门收口（锚点 `[task:release-v0046-and-upgrade-local]`）。

## 11. 建议下一步

1. 复核可抽查：`gh release view v0.0.46 --json assets`、`docker inspect -f '{{.Config.Image}}' new-api`、
   `curl -sI http://127.0.0.1:3000/api/status | grep -i x-new-api-version`。
2. 决定是否把本地 `latest` 同步到 v0.0.46（保持与 registry 一致；v0.0.45 仍留作回滚点）。
3. 观察一个运行周期（重启一次仍为 v0.0.46、`restarts=0`），再考虑清理 v0.0.45 回滚镜像。
4. 若要闭环用户那条原始缺陷，可在面板上真机演练一次「立即更新」（属实例操作，需用户当轮授权）。
