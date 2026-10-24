# 发布与升级证据：v0.0.50（task-62）

- 任务：Trellis `10-24-release-v0050`；共享任务板 `task-62`；执行者 worker-containers
- 执行时间：2026-10-24T08:07:03Z ～ 08:13Z（UTC），全程只访问本机（除 GitHub / ghcr 的发布与拉取）
- 事前依据：用户已点头发版 v0.0.50；Lead 已完成 `VERSION`=0.0.50 提交 `0920671`、附注 tag `v0.0.50`
- 红线遵守：未 force push；未动其它容器 / 镜像 / 卷；未删 `data/`；未提交未推送

## 0. 结论摘要

| 环节 | 结果 |
|---|---|
| A 推送 | `main` `d0d7377..0920671`（fast-forward）+ tag `v0.0.50` 新建，exit 均 0；`origin/main...main` = `0 0` |
| A CI（**三个**） | 全绿：`Publish Docker image (Multi-arch)` **40076787622**；`Publish release` **40076787630**；`lint (gofmt + shellcheck)` **40076785799** |
| A 资产 | 4/4 齐：`new-api-linux-amd64`(118522040) / `new-api-linux-arm64`(115474616) / `SHA256SUMS`(172) / **`SHA256SUMS.sig`(89)** |
| A 镜像 | `v0.0.50` OCI index 存在（linux/amd64 `0935a7ae…` + linux/arm64 + 2 attestation） |
| B 升级 | 实例升到 **v0.0.50**（新容器 `be9a4778f9c0`），2 秒就绪 |
| B 核验 | running / restarts=0 / 挂载 `/root/new-api-own/data` / 200 + `X-New-Api-Version: v0.0.50` / 8 容器全 Up / 卷 3 / `data/` 59 → 60（新启动日志，不减） |
| C① panic recovery | 实现已在运行二进制中（`panic recovered` 命中 **5**、`panic_recovery` **1**）；日志中 `panic recovered` **0 次**（无 panic 触发，按任务口径 0 也算通过） |
| C② 档位名 | 运行实例实际服务的 JS 含 `Peak hours`×8 / `Off-peak hours`×8 / `高峰`×2 / `低谷`×2；**被删除的旧键 `"Peak":`、`"Off-peak":` 命中 0**；二进制内嵌同名字符串 |
| C② 计费配置 | 两行 `billing_mode` 仍 `tiered_expr`；两行表达式 sha256 **相同**且与历史一致（`5705272e190aacb6…`）——升级未改计费 |
| C③ 回滚点 | `v0.0.49`(c029c13215aa)、`v0.0.48`(e485361d15d4) **均保留** |
| C④ latest | `latest` 已对齐 v0.0.50，同完整 ID `sha256:1a909a53…58b38` |

## 1. A：发布 v0.0.50

### 1.1 推送（无 force）

```console
### 2026-10-24T08:07:03Z STEP A1 push
$ git push origin main
To https://github.com/zhemed/new-api-own.git
   d0d7377..0920671  main -> main
push_main_exit=0

$ git push origin v0.0.50
To https://github.com/zhemed/new-api-own.git
 * [new tag]         v0.0.50 -> v0.0.50
push_tag_exit=0

ahead/behind=0	0
origin tag: 213ec0a98c997788ce48760ad2abe749775bdb6c	refs/tags/v0.0.50
```

### 1.2 CI：三个 workflow 全绿

```console
  -> GREEN Publish Docker image (Mult=success(40076787622) | Publish release (image + L=success(40076787630) | lint (gofmt + shellcheck)=success(40076785799)
watch_exit=0
```

| workflow | run id | 结论 |
|---|---|---|
| `Publish Docker image (Multi-arch)` | 40076787622 | success |
| `Publish release (image + Linux updater binaries)` | 40076787630 | success |
| `lint (gofmt + shellcheck)` | 40076785799 | success |

（`lint` 由 main 推送触发，branch=main；两个 `Publish` 由 tag 推送触发，branch=v0.0.50。）

### 1.3 Release 资产

```console
tag=v0.0.50 draft=false prerelease=false published=2026-10-24T08:10:44Z
  new-api-linux-amd64 | 118522040 | uploaded
  new-api-linux-arm64 | 115474616 | uploaded
  SHA256SUMS | 172 | uploaded
  SHA256SUMS.sig | 89 | uploaded
  OK   new-api-linux-amd64
  OK   new-api-linux-arm64
  OK   SHA256SUMS
  OK   SHA256SUMS.sig
```

### 1.4 镜像 manifest

```console
image-ok
  mediaType: application/vnd.oci.image.index.v1+json
  platform=linux/amd64 digest=sha256:0935a7aeb8e49ac10
  platform=unknown/unknown digest=sha256:3a37ea4ae0fa09e3f
  platform=linux/arm64 digest=sha256:173b99afcc60f82e9
  platform=unknown/unknown digest=sha256:45a143ac5699598f3
```

## 2. B：升级本机实例到 v0.0.50

```console
### 2026-10-24T08:12:40Z B upgrade to v0.0.50
--- pre ---
Image=ghcr.io/zhemed/new-api-own:v0.0.49 Binds=["/root/new-api-own/data:/data"] Net=host Restart={"Name":"always","MaximumRetryCount":0} Log={"Type":"json-file","Config":{"max-file":"3","max-size":"10m"}}
data_files_before=59
$ docker pull ghcr.io/zhemed/new-api-own:v0.0.50
Digest: sha256:1a909a53589da212b57080375326b1b17282e61ec483ab7e27b23dfd89458b38
Status: Downloaded newer image for ghcr.io/zhemed/new-api-own:v0.0.50
pull_exit=0
$ docker rm -f new-api   (no -v)
new-api
rm_exit=0
$ docker run -d ... v0.0.50
be9a4778f9c081f8caeb9db1a2450addef27b5f9c8decaad4fd1b00d5e130a05
run_exit=0
--- readiness poll (max 60s) ---
  t+1s http_code=000
  t+2s http_code=200
final_http_code=200

### B6 verification
Status=running RestartCount=0 Image=ghcr.io/zhemed/new-api-own:v0.0.50 Mounts=/root/new-api-own/data
Binds=["/root/new-api-own/data:/data"] Net=host Restart={"Name":"always","MaximumRetryCount":0} Log={"Type":"json-file","Config":{"max-file":"3","max-size":"10m"}}
HTTP/1.1 200 OK
X-New-Api-Version: v0.0.50
containers_running=8 all=8
volumes=3 dangling=2 data_files=60
```

升级前后配置逐项一致（`host` / `always` / `json-file 10m×3` / 工作区挂载 / 四个 LOG_ 前缀环境变量），
**仅镜像 tag 从 v0.0.49 变为 v0.0.50**；停机窗口约 2 秒；`data/` 59 → 60 为新容器启动日志。

## 3. C：本轮特有复核

### 3.1 panic recovery（实现已上线，本轮无触发）

```console
=== 从二进制里核对实现已上线（容器内 grep -a）===
  panic recovered          hits=5
  panic_recovery           hits=1
=== 日志里 panic recovered 计数（0 也算通过）===
  当前日志 grep -c: 0
  命中文件数: 0
```

实现要点（源码 `middleware/panic_recovery.go`）：进程不退出、当前请求返回 500 + 通用文案（**不含 panic 原文**）、
服务端以 error 级记录 `panic recovered: <值>` + method/path + request id + 完整堆栈（**不含请求头**）。
日志 0 命中 = 升级后没有触发 panic（正常路径），实现已随二进制上线（命中 5 次）。

### 3.2 档位名按语言显示（前端已随镜像更新）

从**运行中的实例**实际服务出的 JS 抓取（不是本地源码）：

```console
  static/js/index.3e479afbed.js  http=200 bytes=3485675
  （另有 4 个 vendor/chunk 同样 200）

--- 档位名相关字符串命中（新键与中文）---
  Off-peak hours     命中=8
  Peak hours         命中=8
  高峰             命中=2
  低谷             命中=2
--- 被删除的旧键（应为 0 命中）---
  "Off-peak":        命中=0
  "Peak":            命中=0

--- 二进制内嵌（embedded 前端）---
  binary "offpeak" hits=2
  binary "Peak hours" hits=2
  binary "高峰" hits=1
```

说明：`web/src/features/pricing/lib/tier-labels.ts` 把 `peak`/`offpeak` 映射到 i18n 键
`Peak hours` / `Off-peak hours`（zh 为「高峰」/「低谷」）；上述实测表明 v0.0.50 镜像里**新映射与中文文案已上线**，
且上一轮清理掉的旧键 `Peak` / `Off-peak` 已不存在。

### 3.3 计费配置未被升级改动

```console
  billing_mode: {"deepseek-flash": "tiered_expr", "deepseek-v4.1-flash": "tiered_expr"}
  mode 两行 tiered_expr : PASS
  两行 expr sha256      : 5705272e190aacb672a4a6bafe8b606ff006f0d6ef7b94cf5d067fae2025471f
  两行 sha256 相同      : PASS
  与历史基线(5705272e...) 一致 : PASS
```

### 3.4 回滚点与 latest 对齐

```console
ghcr.io/zhemed/new-api-own:latest | 1a909a53589d | 4 minutes ago
ghcr.io/zhemed/new-api-own:v0.0.50 | 1a909a53589d | 4 minutes ago
ghcr.io/zhemed/new-api-own:v0.0.49 | c029c13215aa | 2 hours ago   <- 回滚点
ghcr.io/zhemed/new-api-own:v0.0.48 | e485361d15d4 | 4 hours ago   <- 回滚点
$ docker tag ghcr.io/zhemed/new-api-own:v0.0.50 ghcr.io/zhemed/new-api-own:latest
latest=sha256:1a909a53589da212b57080375326b1b17282e61ec483ab7e27b23dfd89458b38
v0.0.50=sha256:1a909a53589da212b57080375326b1b17282e61ec483ab7e27b23dfd89458b38
  OK latest == v0.0.50
```

### 3.5 最终服务状态

```console
Image=ghcr.io/zhemed/new-api-own:v0.0.50 Status=running Restarts=0 Mount=/root/new-api-own/data
HTTP/1.1 200 OK / X-New-Api-Version: v0.0.50
containers=8/8  volumes=3  data_files=60
GET /api/status  http=200
GET /api/pricing http=200
```

## 4. 回滚命令（未触发）

```console
$ docker rm -f new-api
$ docker run -d --name new-api --restart always --network host --log-opt max-size=10m --log-opt max-file=3 \
    -v /root/new-api-own/data:/data -e LOG_SQL_DSN=memory -e LOG_MEMORY_MAX_BYTES=200MB \
    -e LOG_MEMORY_MAX_ROWS=200000 -e LOG_CLEANUP_RETENTION_DAYS=7 ghcr.io/zhemed/new-api-own:v0.0.49
$ curl -s -o /dev/null -w '%{http_code}\n' http://127.0.0.1:3000/api/status   # 期望 200
```

`v0.0.49` 与 `v0.0.48` 镜像均在本地；`/root/new-api-own/data` 未受影响。
标签回退：`docker tag ghcr.io/zhemed/new-api-own:v0.0.49 ghcr.io/zhemed/new-api-own:latest`。

## 5. 风险与未完成

- 本轮带了行为变更（panic recovery、档位名、多轮死代码清理、relaykit 收敛），已覆盖：三 CI 绿、实例 200、
  8 容器存活、GET 端点 200、定价配置不变、前端新文案实测在线；更深的业务链路（真实计费请求、面板交互）未逐项点测。
- 本地 `latest` 已对齐 v0.0.50；registry 侧 `latest` 由 CI 更新（同 tag 触发链，本轮 release 已发布）。
- `data/` 仍无异地备份（历史挂账项）。
- 未提交：本文件留工作区，待 Lead 按闸门收口（锚点 `[task:release-v0050]`）。

## 6. 建议下一步

1. Lead 复核可抽查：`gh release view v0.0.50 --json assets`、`docker inspect -f '{{.Config.Image}}' new-api`、
   `curl -sI http://127.0.0.1:3000/api/status | grep -i x-new-api-version`。
2. 观察一个运行周期后，可按「当前 + 上一版」策略清理更早回滚镜像（`v0.0.48` 及更早，需用户单独点头）。
3. 面板真机「立即更新」演练仍可留待下一版本窗口闭环（需用户当轮授权）。
