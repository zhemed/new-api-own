# 发布与升级证据：v0.0.51（task-64）

- 任务：Trellis `10-24-release-v0051`；共享任务板 `task-64`；执行者 worker-containers
- 执行时间：2026-10-24T09:03:12Z ～ 09:09Z（UTC），全程只访问本机（除 GitHub / ghcr 的发布与拉取）
- 事前依据：用户点头发版 v0.0.51；Lead 已完成 `VERSION`=0.0.51 提交 `ab0dfa3`、附注 tag `v0.0.51` 已打、main 已推送
- 红线遵守：未 force push；未动其它容器 / 镜像 / 卷；未删 `data/`；未提交未推送

## 0. 结论摘要

| 环节 | 结果 |
|---|---|
| A 推送 | 只推 tag：`v0.0.51` 新建（`8ccf0fef…`），exit=0；main 已由 Lead 推送，`origin/main...main` = `0 0` |
| A CI（**三个**） | 全绿：`Publish Docker image (Multi-arch)` **40080432515**；`Publish release` **40080432597**；`lint (gofmt + shellcheck)` **40080383189** |
| A 资产 | 4/4 齐：`new-api-linux-amd64`(118522040) / `new-api-linux-arm64`(115474616) / `SHA256SUMS`(172) / **`SHA256SUMS.sig`(89)** |
| A 镜像 | `v0.0.51` OCI index 存在（linux/amd64 `4cec59dc…` + linux/arm64 + 2 attestation） |
| B 升级 | 实例升到 **v0.0.51**（新容器 `56c85d316f2a`），2 秒就绪 |
| B 核验 | running / restarts=0 / 挂载 `/root/new-api-own/data` / 200 + `X-New-Api-Version: v0.0.51` / 8 容器全 Up / 卷 3 / `data/` 60 → 61（新启动日志，不减） |
| **C 新措辞在线** | 运行实例服务出的 `index.b47b92b672.js`（**新构建哈希**）中：`高峰时段` **1**、`空闲时段` **1**、`高峰時段` **1**、`空閒時段` **1**（四项全部 PASS＞0） |
| C 旧措辞/原始标签 | `"Peak hours":"高峰"` 0 次、`"Off-peak hours":"低谷"` 0 次、`offpeak · ` 0 次、`peak · ` 0 次 |
| C 入口覆盖 | 4 个入口均使用同一映射 `getTierDisplayLabel`（定价面板 / 日志列表详情列 / 日志详情弹窗 / 模型详情价格表） |
| C 回滚点 | `v0.0.50`(1a909a53589d)、`v0.0.49`(c029c13215aa) 均保留（`v0.0.48` 也在） |
| C latest | `latest` 已对齐 v0.0.51，同完整 ID `sha256:c966b8b9…bb0e8` |

## 1. A：发布 v0.0.51

```console
### STEP A1 push
预检: main 与 origin 差异=0	0  本地 tag=8ccf0fef597f6c589bac6d73af7028eab9a9347a  远端 tag=
$ git push origin v0.0.51
To https://github.com/zhemed/new-api-own.git
 * [new tag]         v0.0.51 -> v0.0.51
push_tag_exit=0
ahead/behind=0	0
origin tag: 8ccf0fef597f6c589bac6d73af7028eab9a9347a	refs/tags/v0.0.51
```

### 1.2 CI：三个 workflow 全绿

```console
  -> GREEN Publish Docker image (Mult=success(40080432515) | Publish release (image + L=success(40080432597) | lint (gofmt + shellcheck)=success(40080383189)
watch_exit=0
```

| workflow | run id | 结论 |
|---|---|---|
| `Publish Docker image (Multi-arch)` | 40080432515 | success |
| `Publish release (image + Linux updater binaries)` | 40080432597 | success |
| `lint (gofmt + shellcheck)` | 40080383189 | success（Lead 的 main 推送触发，早于 tag 推送即已绿） |

### 1.3 Release 资产

```console
tag=v0.0.51 draft=false prerelease=false published=2026-10-24T09:06:15Z
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
  platform=linux/amd64 digest=sha256:4cec59dca688b113f
  platform=unknown/unknown digest=sha256:25090ea3b04f08b05
  platform=linux/arm64 digest=sha256:e791adc11df00c89b
  platform=unknown/unknown digest=sha256:81bf7b4fd8d17d8b1
```

## 2. B：升级本机实例到 v0.0.51

```console
### 2026-10-24T09:08:12Z B upgrade to v0.0.51
--- pre ---
Image=ghcr.io/zhemed/new-api-own:v0.0.50 Binds=["/root/new-api-own/data:/data"] Net=host Restart={"Name":"always","MaximumRetryCount":0} Log={"Type":"json-file","Config":{"max-file":"3","max-size":"10m"}}
data_files_before=60
$ docker pull ghcr.io/zhemed/new-api-own:v0.0.51
Digest: sha256:c966b8b9327455b9309463d7d69f9353b212cfebb90597c1bc5fdbfdf1bbb0e8
pull_exit=0
$ docker rm -f new-api   (no -v)
new-api
rm_exit=0
$ docker run -d ... v0.0.51
56c85d316f2a9d4c16e75c8af2e000ddd1285b0f2b9f5f31686851125215b432
run_exit=0
--- readiness poll ---
  t+1s http_code=000
  t+2s http_code=200
final_http_code=200

### B6 verification
Status=running RestartCount=0 Image=ghcr.io/zhemed/new-api-own:v0.0.51 Mounts=/root/new-api-own/data
Binds=["/root/new-api-own/data:/data"] Net=host Restart={"Name":"always","MaximumRetryCount":0} Log={"Type":"json-file","Config":{"max-file":"3","max-size":"10m"}}
HTTP/1.1 200 OK
X-New-Api-Version: v0.0.51
containers_running=8 all=8
volumes=3 dangling=2 data_files=61
```

升级前后配置逐项一致，仅镜像 tag 从 v0.0.50 变为 v0.0.51；停机约 2 秒；`data/` 60 → 61 为新启动日志。

## 3. C：本轮特有核对（用户就是为此发版）

### 3.1 从运行实例**实际服务出的前端资源**核对新措辞

```console
### 2026-10-24T09:08:35Z C7 从运行实例实际服务出的前端资源核对新措辞
  首页 http=200 bytes=1094
  index.html 引用的 JS:
    src="/static/js/vendor-ui-primitives.b17b3046c5.js"
    src="/static/js/vendor-tanstack.74c48f25c5.js"
    src="/static/js/lib-react.064bab1680.js"
    src="/static/js/2507.3b5d1e1d80.js"
    src="/static/js/index.b47b92b672.js"          <- 注意：v0.0.50 时是 index.3e479afbed.js（新构建）

  抓取 /static/js/vendor-ui-primitives.b17b3046c5.js http=200 bytes=339927
  抓取 /static/js/vendor-tanstack.74c48f25c5.js   http=200 bytes=178193
  抓取 /static/js/lib-react.064bab1680.js         http=200 bytes=189031
  抓取 /static/js/2507.3b5d1e1d80.js              http=200 bytes=471209
  抓取 /static/js/index.b47b92b672.js             http=200 bytes=3485699

--- C7 断言：新措辞在 served JS 中的命中次数 ---
  高峰时段 命中=1  PASS(>0)
  空闲时段 命中=1  PASS(>0)
  高峰時段 命中=1  PASS(>0)
  空閒時段 命中=1  PASS(>0)

--- 旧措辞/原始标签残留检查（应为 0 命中）---
  "Peak hours":"高峰"          命中=0
  "Off-peak hours":"低谷"      命中=0
  offpeak ·                     命中=0
  peak ·                        命中=0

--- 档位名映射键仍在（英文 i18n 键） ---
  Peak hours       命中=8
  Off-peak hours   命中=8
```

结论：新措辞（zh 高峰时段/空闲时段、zh-TW 高峰時段/空閒時段）**已在运行实例实际服务的前端产物中**；
旧译法（高峰/低谷作独立值）与「原始标签 + · 」拼接路径均无命中。

### 3.2 四个入口共用同一映射（源码取证，只读）

```console
web/src/features/usage-logs/components/columns/common-logs-columns.tsx:171:
  const tierLabel = getTierDisplayLabel(tieredSummary.tier.label, t)
web/src/features/usage-logs/components/columns/common-logs-columns.tsx:173:
  text: `${tierLabel} · ${formatPriceList(baseEntries, true)}`
web/src/features/usage-logs/components/dialogs/details-dialog.tsx:61:  import { getTierDisplayLabel } from '@/features/pricing/lib/tier-labels'
web/src/features/pricing/components/dynamic-pricing-breakdown.tsx:47:  import { getTierDisplayLabel } from '../lib/tier-labels'
web/src/features/pricing/components/model-details.tsx:73:  import { getTierDisplayLabel } from '../lib/tier-labels'
```

即：日志列表「详情」列、日志详情弹窗、定价面板、模型详情价格表 4 个入口都走
`features/pricing/lib/tier-labels.ts` 的 `getTierDisplayLabel`（内部映射 `TIER_LABEL_I18N_KEYS`），
不存在各自拼原始标签的分叉路径。

### 3.3 tiered_expr 历史记录（条件不满足，如实说明）

```console
  logs 表中含 tiered_expr 的记录数: 0
```

本实例的用量日志表里没有 tiered_expr 计费记录（未产生过 tiered 计费用量），因此无法用「历史记录渲染」做二次核对；
前端渲染规则已通过 3.1 的产物核对与 3.2 的源码取证确认就位。未造新请求（按任务要求）。

### 3.4 回滚点与 latest 对齐

```console
ghcr.io/zhemed/new-api-own:v0.0.51 | c966b8b93274
ghcr.io/zhemed/new-api-own:latest | 1a909a53589d   <- 对齐前
ghcr.io/zhemed/new-api-own:v0.0.50 | 1a909a53589d   <- 回滚点
ghcr.io/zhemed/new-api-own:v0.0.49 | c029c13215aa   <- 回滚点
ghcr.io/zhemed/new-api-own:v0.0.48 | e485361d15d4
$ docker tag ghcr.io/zhemed/new-api-own:v0.0.51 ghcr.io/zhemed/new-api-own:latest
  tag_exit=0
  latest=sha256:c966b8b9327455b9309463d7d69f9353b212cfebb90597c1bc5fdbfdf1bbb0e8
  v0.0.51=sha256:c966b8b9327455b9309463d7d69f9353b212cfebb90597c1bc5fdbfdf1bbb0e8
  OK latest == v0.0.51
```

### 3.5 最终服务状态

```console
  Image=ghcr.io/zhemed/new-api-own:v0.0.51 Status=running Restarts=0 Mount=/root/new-api-own/data
  HTTP/1.1 200 OK / X-New-Api-Version: v0.0.51
  containers=8/8  volumes=3  data_files=61
  GET /api/pricing http=200
```

## 4. 回滚命令（未触发）

```console
$ docker rm -f new-api
$ docker run -d --name new-api --restart always --network host --log-opt max-size=10m --log-opt max-file=3 \
    -v /root/new-api-own/data:/data -e LOG_SQL_DSN=memory -e LOG_MEMORY_MAX_BYTES=200MB \
    -e LOG_MEMORY_MAX_ROWS=200000 -e LOG_CLEANUP_RETENTION_DAYS=7 ghcr.io/zhemed/new-api-own:v0.0.50
$ curl -s -o /dev/null -w '%{http_code}\n' http://127.0.0.1:3000/api/status   # 期望 200
```

`v0.0.50` 与 `v0.0.49` 镜像均在本地；`/root/new-api-own/data` 未受影响。
标签回退：`docker tag ghcr.io/zhemed/new-api-own:v0.0.50 ghcr.io/zhemed/new-api-own:latest`。

## 5. 风险与未完成

- 本轮改动集中在前端展示层（措辞 + 4 入口覆盖），已用**运行实例实际服务出的 JS** 逐项验证，且旧措辞/原始标签拼接均无残留；
  未做浏览器内的视觉点击验证（属 UI 手工核对，如需要可在 `bw` 浏览器里点一遍，本轮未做）。
- `logs` 表无 tiered_expr 历史记录，故「历史日志渲染」一项条件不满足（如实记录，未造数据）。
- `data/` 仍无异地备份（历史挂账项）。
- 未提交：本文件留工作区，待 Lead 按闸门收口（锚点 `[task:release-v0051]`）。

## 6. 建议下一步

1. Lead 复核可抽查：`curl -s http://127.0.0.1:3000/static/js/index.b47b92b672.js | grep -c 高峰时段`（应 ≥1）。
2. 若想更彻底验证 4 个入口的**视觉**效果，可打开面板「定价 / 用量日志」页人工看一眼（或让我用常驻浏览器 `bw` 截图，需授权）。
3. 观察一个运行周期后，可按「当前 + 上一版」策略清理更早回滚镜像（`v0.0.49` 及更早，需用户单独点头）。
