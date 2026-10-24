# 发布与升级证据：v0.0.48（task-35）

- 任务：Trellis `10-24-setup-order-and-v0048`；共享任务板 `task-35`；执行者 worker-containers
- 执行时间：2026-10-24T04:17:18Z ～ 04:23:13Z（UTC），全程只访问本机（除 GitHub / ghcr 的发布与拉取）
- 事前依据：用户点名授权发版与升级本机实例；Lead 已完成修复提交 `4f9c0d5`、`VERSION`=0.0.48 提交 `819dbb5`、附注 tag `v0.0.48`
- 红线遵守：未 force push；未动其它容器 / 镜像 / 卷；未删 `data/`；未升级镜像外的任何对象；未提交未推送

## 0. 结论摘要

| 环节 | 结果 |
|---|---|
| A 推送 | `main` `2867700..819dbb5`（fast-forward）+ tag `v0.0.48` 新建，exit 均 0；`origin/main...main` = `0 0` |
| A CI | **双绿**：`Publish Docker image (Multi-arch)` run **40061752745** success；`Publish release (image + Linux updater binaries)` run **40061752807** success |
| A 资产 | 4/4 齐：`new-api-linux-amd64`(118526136) / `new-api-linux-arm64`(115474616) / `SHA256SUMS`(172) / **`SHA256SUMS.sig`(89)** |
| A 镜像 | `v0.0.48` OCI index 存在（linux/amd64 `dc856bf4…` + linux/arm64 + 2 attestation） |
| B 升级 | 实例升到 **v0.0.48**（新容器 `c75150b479c5`），2 秒就绪 |
| B 核验 | running / restarts=0 / 挂载 `/root/new-api-own/data` / 200 + `X-New-Api-Version: v0.0.48` / 8 容器全 Up / 卷 3 / `data/` 57 → 58（新启动日志） |
| B 回滚点 | `v0.0.47`(bccc31d54ad7)、`v0.0.46`(95e3c1b97e32)、`v0.0.45`(e114b17532f1) 镜像**全部保留** |
| C 上轮修复 | **仍在**：两行 `billing_mode` 均 `tiered_expr`；两行表达式 sha256 **相同**且与上轮演练后记录值一致（`5705272e190aacb6…`）；`GroupRatio`/`TopupGroupRatio` 仍未被改动 |

## 1. A：发布 v0.0.48

```console
### 2026-10-24T04:17:18Z STEP A1 push
$ git push origin main
To https://github.com/zhemed/new-api-own.git
   2867700..819dbb5  main -> main
push_main_exit=0

$ git push origin v0.0.48
To https://github.com/zhemed/new-api-own.git
 * [new tag]         v0.0.48 -> v0.0.48
push_tag_exit=0

ahead/behind=0	0
origin tag: 9b707bcda85b448c44306163115cca59057740eb	refs/tags/v0.0.48
```

均为普通推送（**无** `--force`）。

### 1.1 CI 双绿

```console
### 2026-10-24T04:22:30Z A2 CI result
  -> GREEN Publish Docker image (Multi-=success(40061752745) | Publish release (image + Lin=success(40061752807)
```

- `Publish Docker image (Multi-arch)`：run **40061752745** → `success`
- `Publish release (image + Linux updater binaries)`：run **40061752807** → `success`

### 1.2 Release 资产（关键：签名资产）

```console
### A3 release assets
tag=v0.0.48 draft=false prerelease=false published=2026-10-24T04:20:44Z
  new-api-linux-amd64 | 118526136 | uploaded
  new-api-linux-arm64 | 115474616 | uploaded
  SHA256SUMS | 172 | uploaded
  SHA256SUMS.sig | 89 | uploaded
  OK   new-api-linux-amd64
  OK   new-api-linux-arm64
  OK   SHA256SUMS
  OK   SHA256SUMS.sig
```

### 1.3 镜像 manifest

```console
### A4 manifest inspect
image-ok
  mediaType: application/vnd.oci.image.index.v1+json
  platform=linux/amd64 digest=sha256:dc856bf4d0e218c9d
  platform=unknown/unknown digest=sha256:132b272700ee7d66b
  platform=linux/arm64 digest=sha256:77fddb2fa42f3ab21
  platform=unknown/unknown digest=sha256:6c29063a047f432e7
```

registry `v0.0.48` index digest（只读）：`sha256:dc856bf4d0e218c9d6c35be751ec0c977f0ffb77ac4908e7cee741d20f6272f1`

## 2. B：升级本机实例到 v0.0.48

```console
### 2026-10-24T04:22:52Z B upgrade to v0.0.48
--- pre-upgrade ---
Image=ghcr.io/zhemed/new-api-own:v0.0.47 Binds=["/root/new-api-own/data:/data"] Net=host Restart={"Name":"always","MaximumRetryCount":0} Log={"Type":"json-file","Config":{"max-file":"3","max-size":"10m"}}
data_files_before=57
$ docker pull ghcr.io/zhemed/new-api-own:v0.0.48
Digest: sha256:e485361d15d42ce3804fa92a1fcde544b3c4a5d7010c5862fd5f8a7e2876234c
Status: Downloaded newer image for ghcr.io/zhemed/new-api-own:v0.0.48
pull_exit=0
$ docker rm -f new-api   (no -v)
new-api
rm_exit=0
$ docker run -d ... ghcr.io/zhemed/new-api-own:v0.0.48
c75150b479c5c98249fe79b86de1a0190164abd3c62b8ec72cfd450c113ea621
run_exit=0
--- readiness poll (max 60s) ---
  t+1s http_code=000
  t+2s http_code=200
final_http_code=200

### B6 verification
Status=running RestartCount=0 Image=ghcr.io/zhemed/new-api-own:v0.0.48 Mounts=/root/new-api-own/data
Binds=["/root/new-api-own/data:/data"] Net=host Restart={"Name":"always","MaximumRetryCount":0} Log={"Type":"json-file","Config":{"max-file":"3","max-size":"10m"}}
Env: LOG_SQL_DSN=memory / LOG_MEMORY_MAX_BYTES=200MB / LOG_MEMORY_MAX_ROWS=200000 / LOG_CLEANUP_RETENTION_DAYS=7
HTTP/1.1 200 OK
X-New-Api-Version: v0.0.48
running=8 all=8
volumes=3 dangling=2 data_files=58
```

升级前后配置逐项一致（`host` 网络 / `always` / `json-file 10m×3` / 工作区挂载 / 四个 LOG_ 前缀环境变量），
**仅镜像 tag 从 v0.0.47 变为 v0.0.48**；停机窗口约 1 秒（04:22:52 停旧 → 04:22:53 就绪）。

`data/` 文件数 57 → 58：增量来自新容器启动时新建的日志文件（正常运行产物），核心资产未受影响。

## 3. C：确认上一轮「重置恢复整份默认」修复仍在

```console
### 2026-10-24T04:23:13Z C 确认上一轮修复仍在（升级后读库）
  billing_mode 全文 : {"deepseek-flash": "tiered_expr", "deepseek-v4.1-flash": "tiered_expr"}
  mode 两行 tiered_expr : PASS
  deepseek-flash     expr sha256 = 5705272e190aacb672a4a6bafe8b606ff006f0d6ef7b94cf5d067fae2025471f
  deepseek-v4.1-flash expr sha256 = 5705272e190aacb672a4a6bafe8b606ff006f0d6ef7b94cf5d067fae2025471f
  两行表达式 sha256 相同 : PASS
  与上一轮演练后记录值(5705272e190aacb6...) 一致 : PASS
  GroupRatio / TopupGroupRatio : None / None
```

说明：本次升级没有触碰 `options` 表的定价行，两行仍为默认的 tiered_expr 且表达式逐字符相同；
与 task-29 真机演练后的记录值逐位一致（同一 sha256），说明该修复随升级保持不变。

## 4. 镜像清单（回滚点齐全）

```console
ghcr.io/zhemed/new-api-own:v0.0.48 | e485361d15d4 | 3 minutes ago
ghcr.io/zhemed/new-api-own:latest   | bccc31d54ad7 | 37 minutes ago   <- 仍指 v0.0.47
ghcr.io/zhemed/new-api-own:v0.0.47 | bccc31d54ad7 | 37 minutes ago
ghcr.io/zhemed/new-api-own:v0.0.46 | 95e3c1b97e32 | 2 hours ago
ghcr.io/zhemed/new-api-own:v0.0.45 | e114b17532f1 | 14 hours ago
```

## 5. 回滚路径（未触发）

```console
$ docker rm -f new-api
$ docker run -d --name new-api --restart always --network host --log-opt max-size=10m --log-opt max-file=3 \
    -v /root/new-api-own/data:/data -e LOG_SQL_DSN=memory -e LOG_MEMORY_MAX_BYTES=200MB \
    -e LOG_MEMORY_MAX_ROWS=200000 -e LOG_CLEANUP_RETENTION_DAYS=7 ghcr.io/zhemed/new-api-own:v0.0.47
$ curl -s -o /dev/null -w '%{http_code}\n' http://127.0.0.1:3000/api/status   # 期望 200
```

`v0.0.47` / `v0.0.46` / `v0.0.45` 镜像均在本地，`/root/new-api-own/data` 未受影响。

## 6. 风险与未完成

- 无功能风险：升级参数与现状逐项一致，部署形态未变（仍为单容器 docker run + host 网络）。
- 本地 `latest` 仍指 **v0.0.47**（bccc31d54ad7），registry 的 `latest` 已由 CI 指到 v0.0.48；
  是否再对齐一次由 Lead / 用户决定（未在本次任务范围）。
- `data/` 文件数 57 → 58（新启动日志），核心资产完好。
- `data/` 仍无异地备份（历史挂账项，本轮未涉及）。
- 未提交：本文件留工作区，待 Lead 按闸门收口（锚点 `[task:setup-order-and-v0048]`）。

## 7. 建议下一步

1. Lead 复核可抽查：`gh release view v0.0.48 --json assets`、`docker inspect -f '{{.Config.Image}}' new-api`、
   `curl -sI http://127.0.0.1:3000/api/status | grep -i x-new-api-version`。
2. 观察一个运行周期（重启后仍 v0.0.48、restarts=0），再考虑清理更早的回滚镜像（需用户单独点头）。
3. 若要保持本机 `latest` 与 registry 一致，可再 `docker tag ...:v0.0.48 ...:latest`（一条命令，需授权）。
4. 下一个版本发布后，补做「面板内立即更新」真机演练，闭环用户原始缺陷。
