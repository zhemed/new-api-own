# 发布与升级证据：v0.0.49（task-49）

- 任务：Trellis `10-24-exported-deadcode-and-v0049`；共享任务板 `task-49`；执行者 worker-containers
- 执行时间：2026-10-24T06:08:24Z ～ 06:14:41Z（UTC），全程只访问本机（除 GitHub / ghcr 的发布与拉取）
- 事前依据：用户点名授权发版与升级；Lead 已完成死代码删除提交 `f78a6c9`、`VERSION`=0.0.49 提交 `eddee29`、附注 tag `v0.0.49`
- 红线遵守：未 force push；未动其它容器 / 镜像 / 卷；未删 `data/`；未提交未推送

## 0. 结论摘要

| 环节 | 结果 |
|---|---|
| A 推送 | `main` `5d27364..eddee29`（fast-forward）+ tag `v0.0.49` 新建，exit 均 0；`origin/main...main` = `0 0` |
| A CI（**三个**） | 全绿：`Publish Docker image (Multi-arch)` **40069015095** success；`Publish release` **40069015023** success；**`lint (gofmt + shellcheck)` 40069013081 success（新增 lint CI 首次真实运行即绿）** |
| A 资产 | 4/4 齐：`new-api-linux-amd64`(118522040) / `new-api-linux-arm64`(115474616) / `SHA256SUMS`(172) / **`SHA256SUMS.sig`(89)** |
| A 镜像 | `v0.0.49` OCI index 存在（linux/amd64 `7f44714e…` + linux/arm64 + 2 attestation） |
| B 升级 | 实例升到 **v0.0.49**（新容器 `a2658511c79a`），2 秒就绪 |
| B 核验 | running / restarts=0 / 挂载 `/root/new-api-own/data` / 200 + `X-New-Api-Version: v0.0.49` / 8 容器全 Up / 卷 3 / `data/` 58 → 59（新启动日志） |
| B 回滚点 | `v0.0.48`(e485361d15d4)、`v0.0.47`(bccc31d54ad7)、`v0.0.46`(95e3c1b97e32) 镜像**全部保留** |
| C 定价复核 | 两行 `billing_mode` 均 `tiered_expr`；两行表达式 sha256 **相同**且与历史基线一致（`5705272e190aacb6…`）；`GroupRatio`/`TopupGroupRatio` 仍未被改动 |
| C 接口抽查 | 5 个真实公开 GET 端点**全部 200**，**0 个 5xx**（大删代码后无回归） |
| C latest | `latest` 已对齐 `v0.0.49`，同 ID `c029c13215aa` |

## 1. A：发布 v0.0.49

### 1.1 推送（无 force）

```console
### 2026-10-24T06:08:24Z STEP A1 push
$ git push origin main
To https://github.com/zhemed/new-api-own.git
   5d27364..eddee29  main -> main
push_main_exit=0

$ git push origin v0.0.49
To https://github.com/zhemed/new-api-own.git
 * [new tag]         v0.0.49 -> v0.0.49
push_tag_exit=0

ahead/behind=0	0
origin tag: 67673895de47ebe013fd7a13e0d685298eadefe7	refs/tags/v0.0.49
```

### 1.2 CI：三个 workflow 全绿

```console
=== 触发后首轮观察 ===
  40069015095  Publish Docker image (Multi-arch)                in_progress           branch=v0.0.49  event=push
  40069015023  Publish release (image + Linux updater binaries) in_progress           branch=v0.0.49  event=push
  40069013081  lint (gofmt + shellcheck)                        completed   success   branch=main     event=push
  40069013177  trellis-gate                                     completed   success   branch=main     event=push

=== 轮询收敛（/tmp/rel49/ci-watch.log）===
  -> GREEN Publish Docker image (Mult=success(40069015095) | Publish release (image + L=success(40069015023) | lint (gofmt + shellcheck)=success(40069013081)
watch_exit=0
```

| workflow | run id | 结论 |
|---|---|---|
| `Publish Docker image (Multi-arch)` | 40069015095 | success |
| `Publish release (image + Linux updater binaries)` | 40069015023 | success |
| **`lint (gofmt + shellcheck)`** | **40069013081** | **success（该 workflow 于上一轮新增，这是它第一次在真实推送中运行）** |

说明：`lint` 的触发条件是 `push: branches: [main]` + `pull_request`，所以它由 **main 推送**触发（branch=main）；
两个 `Publish` 由 **tag 推送**触发（branch=v0.0.49）。三条路径互不干扰。

### 1.3 Release 资产

```console
tag=v0.0.49 draft=false prerelease=false published=2026-10-24T06:11:28Z
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
  platform=linux/amd64 digest=sha256:7f44714e46e459783
  platform=unknown/unknown digest=sha256:1d083de0608543c6d
  platform=linux/arm64 digest=sha256:bc1e5b9cc11d6f3e6
  platform=unknown/unknown digest=sha256:b5b57a01d7a1faa6c
```

## 2. B：升级本机实例到 v0.0.49

```console
### 2026-10-24T06:14:20Z B upgrade to v0.0.49
--- pre-upgrade ---
Image=ghcr.io/zhemed/new-api-own:v0.0.48 Binds=["/root/new-api-own/data:/data"] Net=host Restart={"Name":"always","MaximumRetryCount":0} Log={"Type":"json-file","Config":{"max-file":"3","max-size":"10m"}}
data_files_before=58
$ docker pull ghcr.io/zhemed/new-api-own:v0.0.49
Digest: sha256:c029c13215aaf3ff8a467b2a0f9e930d24455d01d65ecd7166da8a23cfe90674
Status: Downloaded newer image for ghcr.io/zhemed/new-api-own:v0.0.49
pull_exit=0
$ docker rm -f new-api   (no -v)
new-api
rm_exit=0
$ docker run -d ... ghcr.io/zhemed/new-api-own:v0.0.49
a2658511c79a8e279c15dab7b84f341d466159b3b01ae9a9899ba98b287a8cb1
run_exit=0
--- readiness poll (max 60s) ---
  t+1s http_code=000
  t+2s http_code=200
final_http_code=200

### B6 verification
Status=running RestartCount=0 Image=ghcr.io/zhemed/new-api-own:v0.0.49 Mounts=/root/new-api-own/data
Binds=["/root/new-api-own/data:/data"] Net=host Restart={"Name":"always","MaximumRetryCount":0} Log={"Type":"json-file","Config":{"max-file":"3","max-size":"10m"}}
HTTP/1.1 200 OK
X-New-Api-Version: v0.0.49
containers_running=8 all=8
volumes=3 dangling=2 data_files=59
```

升级前后配置逐项一致（`host` / `always` / `json-file 10m×3` / 工作区挂载 / 四个 LOG_ 前缀环境变量），
**仅镜像 tag 从 v0.0.48 变为 v0.0.49**；停机窗口约 2 秒。`data/` 58 → 59 是新容器启动日志（正常运行产物）。

## 3. C：latest 对齐 + 定价复核 + 接口抽查

### 3.1 latest 对齐

```console
$ docker tag ghcr.io/zhemed/new-api-own:v0.0.49 ghcr.io/zhemed/new-api-own:latest
tag_exit=0
latest=c029c13215aa  v0.0.49=c029c13215aa
  OK latest == v0.0.49
--- images after tag ---
  ghcr.io/zhemed/new-api-own:latest | c029c13215aa
  ghcr.io/zhemed/new-api-own:v0.0.49 | c029c13215aa
  ghcr.io/zhemed/new-api-own:v0.0.48 | e485361d15d4
  ghcr.io/zhemed/new-api-own:v0.0.47 | bccc31d54ad7
  ghcr.io/zhemed/new-api-own:v0.0.46 | 95e3c1b97e32
  ghcr.io/zhemed/new-api-own:v0.0.45 | e114b17532f1
counts: unique=30 tags=39 dangling=0
```

### 3.2 定价复核（升级后读库，确认上一轮修复未被本次大删除影响）

```console
  billing_mode: {"deepseek-flash": "tiered_expr", "deepseek-v4.1-flash": "tiered_expr"}
  mode 两行 tiered_expr : PASS
  两行 expr sha256      : 5705272e190aacb672a4a6bafe8b606ff006f0d6ef7b94cf5d067fae2025471f
  两行 sha256 相同      : PASS
  与历史基线(5705272e...) 一致 : PASS
  GroupRatio/TopupGroupRatio : None / None
```

### 3.3 GET 接口抽查（大删代码后无 5xx）

| 端点 | HTTP | 响应体摘要 |
|---|---|---|
| `/api/status` | 200 | 完整 status JSON（2485 字节） |
| `/api/pricing` | 200 | 模型定价 JSON（1456 字节，含 deepseek-flash） |
| `/api/notice` | 200 | `{"data":"","message":"","success":true}` |
| `/api/about` | 200 | `{"data":"","message":"","success":true}` |
| `/api/uptime/status` | 200 | `{"data":[],"message":"","success":true}` |

**5xx 计数 = 0**（上述 5 个真实公开 GET 端点；均来自路由表 `router/api-router.go`，非 404 占位）。

## 4. 回滚路径（未触发）

```console
$ docker rm -f new-api
$ docker run -d --name new-api --restart always --network host --log-opt max-size=10m --log-opt max-file=3 \
    -v /root/new-api-own/data:/data -e LOG_SQL_DSN=memory -e LOG_MEMORY_MAX_BYTES=200MB \
    -e LOG_MEMORY_MAX_ROWS=200000 -e LOG_CLEANUP_RETENTION_DAYS=7 ghcr.io/zhemed/new-api-own:v0.0.48
$ curl -s -o /dev/null -w '%{http_code}\n' http://127.0.0.1:3000/api/status   # 期望 200
```

`v0.0.48` / `v0.0.47` / `v0.0.46` / `v0.0.45` 镜像均在本地；`/root/new-api-own/data` 未受影响。
若需把本地 `latest` 退回：`docker tag ...:v0.0.48 ...:latest`。

## 5. 风险与未完成

- **lint CI 首次真实运行即绿**：说明新增的 gofmt + shellcheck 守卫在当前代码基线（含本次大删除）下通过；
  后续任何格式/脚本退化会在这条 workflow 上直接红。
- Go 侧：本次删了 59 个导出符号 + 2 个空壳文件，升级后服务 8 容器全 Up、5 个 GET 端点全 200、定价复核 PASS，
  暂未发现运行时回归；更深路径（面板交互、计费链路）本轮未逐项点测。
- `data/` 仍无异地备份（历史挂账项）；`data/` 文件数 58 → 59 属新启动日志。
- 本地 `latest` 已对齐 v0.0.49；registry `latest` 由 CI 同步（同一 tag 触发链）。
- 未提交：本文件留工作区，待 Lead 按闸门收口（锚点 `[task:exported-deadcode-and-v0049]`）。

## 6. 建议下一步

1. Lead 复核可抽查：`gh release view v0.0.49 --json assets`、`docker inspect -f '{{.Config.Image}}' new-api`、
   `curl -sI http://127.0.0.1:3000/api/status | grep -i x-new-api-version`。
2. 观察一个运行周期后，再决定是否清理更早的回滚镜像（`v0.0.45`~`v0.0.48`，需用户单独点头）。
3. 若后续要继续压缩回滚点，建议保留「当前版本 + 上一版」两档即可。
