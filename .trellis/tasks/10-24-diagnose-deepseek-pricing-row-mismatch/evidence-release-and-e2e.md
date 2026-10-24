# 发布与真机演练证据：v0.0.47（task-29）

- 任务：Trellis `10-24-diagnose-deepseek-pricing-row-mismatch`；共享任务板 `task-29`；执行者 worker-containers
- 执行时间：2026-10-24T03:42:56Z ～ 03:51:48Z（UTC）
- 事前依据：用户点名授权发版与升级本机实例；修复提交 `19ff3b8` + 附注 tag `v0.0.47`（Lead 已完成）；演练凭据读写经用户授权
- 红线遵守：未 force push；未动其它容器 / 镜像 / 卷；未删 `data/`；**令牌未回显、未落盘、未写入本文件**；
  除 GitHub / ghcr 与本地实例外未访问其它主机；未提交未推送

## 0. 结论摘要

| 环节 | 结果 |
|---|---|
| A 推送 | `main` `fe8cba5..19ff3b8`（fast-forward）+ tag `v0.0.47` 新建，exit 均 0；`origin/main...main` = `0 0` |
| A CI | **双绿**：`Publish Docker image (Multi-arch)` run **40059436370** success；`Publish release (image + Linux updater binaries)` run **40059436414** success |
| A 资产 | 4/4 齐：`new-api-linux-amd64`(118526136) / `new-api-linux-arm64`(115474616) / `SHA256SUMS`(172) / **`SHA256SUMS.sig`(89)** |
| A 镜像 | `v0.0.47` OCI index 存在（linux/amd64 + linux/arm64 + 2 attestation） |
| B 升级 | 实例升到 **v0.0.47**（新容器 `8f0a7624e016`），2 秒就绪；配置除 tag 外逐项不变 |
| C 演练 | **全部关键断言 PASS**：重置后两行 `billing_mode` 均为 `tiered_expr`；两行表达式 **sha256 完全相同**（`5705272e190aacb6…`）；`GroupRatio`/`TopupGroupRatio` 未被改动 |
| C 凭据 | 临时 32 位令牌仅存在于内存与 DB 事务中，演练后 `users.access_token` 已还原为 **NULL**（复核通过） |
| 服务 | 8 容器全 running；`new-api` running / restarts=0 / 挂载工作区 / 200 + v0.0.47；卷 3；`data/` 57 文件（较演练前 56 增加 1 个启动日志） |

## 1. A：发布 v0.0.47

### 1.1 推送（无 force）

```console
### 2026-10-24T03:42:56Z PRE-PUSH baseline
ahead/behind(origin/main...main)=0	1
HEAD=19ff3b8 fix(pricing): 重置恢复整份定价默认（含计费模式与表达式）+ 前端文案与回归 [task:diagnose-deepseek-pricing-row-mismatch]
VERSION=0.0.47
instance: ghcr.io/zhemed/new-api-own:v0.0.46 | running | restarts=0
api: X-New-Api-Version: v0.0.46

### STEP A1 push
$ git push origin main
To https://github.com/zhemed/new-api-own.git
   fe8cba5..19ff3b8  main -> main
push_main_exit=0

$ git push origin v0.0.47
To https://github.com/zhemed/new-api-own.git
 * [new tag]         v0.0.47 -> v0.0.47
push_tag_exit=0

ahead/behind=0	0
origin tag: b513291146e7d4080205384744a96594f110a124	refs/tags/v0.0.47
```

### 1.2 CI 双绿（轮询每 25 秒）

```console
### 2026-10-24T03:48:51Z A2 CI result
  40059436370  Publish Docker image (Multi-arch)                  completed    success    branch=v0.0.47
  -> GREEN Publish Docker image (Multi-=success(40059436370) | Publish release (image + Lin=success(40059436414)
watch_exit=0
```

### 1.3 Release 资产（关键：签名资产）

```console
### A3 release assets
tag=v0.0.47 draft=false prerelease=false published=2026-10-24T03:46:22Z
  new-api-linux-amd64 | 118526136 | uploaded
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
### A4 manifest inspect
image-ok
  mediaType: application/vnd.oci.image.index.v1+json
  platform=linux/amd64 digest=sha256:7ceaee38c39193091
  platform=unknown/unknown digest=sha256:aec3e0a8e008446a2
  platform=linux/arm64 digest=sha256:b7aa8597619b9ae51
  platform=unknown/unknown digest=sha256:d9fb9c5e0b1532f10
```

## 2. B：升级本机实例到 v0.0.47

```console
### 2026-10-24T03:49:19Z B upgrade to v0.0.47
--- pre-upgrade config ---
Image=ghcr.io/zhemed/new-api-own:v0.0.46 Binds=["/root/new-api-own/data:/data"] Net=host Restart={"Name":"always","MaximumRetryCount":0} Log={"Type":"json-file","Config":{"max-file":"3","max-size":"10m"}}
$ docker pull ghcr.io/zhemed/new-api-own:v0.0.47
Digest: sha256:bccc31d54ad7f37d0e33a0f0cfb913a72dce99ce6681a579161f51ba7c59fe55
Status: Downloaded newer image for ghcr.io/zhemed/new-api-own:v0.0.47
pull_exit=0
$ docker rm -f new-api   (no -v)
new-api
rm_exit=0
$ docker run -d --name new-api --restart always --network host --log-opt max-size=10m --log-opt max-file=3 -v /root/new-api-own/data:/data -e LOG_SQL_DSN=memory -e LOG_MEMORY_MAX_BYTES=200MB -e LOG_MEMORY_MAX_ROWS=200000 -e LOG_CLEANUP_RETENTION_DAYS=7 ghcr.io/zhemed/new-api-own:v0.0.47
8f0a7624e01628a785133549f39cc939c1dd2e7dd0d25f3177075756b8d66b6e
run_exit=0
--- readiness poll (max 60s) ---
  t+1s http_code=000
  t+2s http_code=200
final_http_code=200
--- verify ---
Status=running RestartCount=0 Image=ghcr.io/zhemed/new-api-own:v0.0.47 Mounts=/root/new-api-own/data
HTTP/1.1 200 OK
X-New-Api-Version: v0.0.47
containers: running=8 all=8
volumes=3  data_files=57
images: ghcr.io/zhemed/new-api-own:v0.0.47 ghcr.io/zhemed/new-api-own:latest ghcr.io/zhemed/new-api-own:v0.0.46 ghcr.io/zhemed/new-api-own:v0.0.45
```

升级前后除镜像 tag 外逐项一致（`host` 网络 / `always` / `json-file 10m×3` / 工作区挂载 / 四个 LOG_ 前缀环境变量）；
停机窗口约 2 秒（03:49:19 停旧 → 03:49:21 就绪）。`v0.0.46` 与 `v0.0.45` 镜像均保留作回滚点。

## 3. C：真机端到端演练（核心验收）

### 3.0 方法与安全措施

- 演练把用户报的「坏状态」用**管理接口**写回实例，再点「重置」，最后用**读库**断言两行是否一致。
- 临时凭据：随机 32 位字母数字令牌写入 `users.access_token`（原值为 **NULL**）；
  令牌只在同一个 bash 会话的环境变量中存在，**不打印、不写盘、不进 curl argv**（管理调用统一用 python urllib 发请求）；
  脚本用 `trap ... EXIT` 保证任何失败退出路径都会把 `access_token` 还原为 NULL。
- 演练脚本与原始日志位于 `/tmp/rel47/drill.sh`、`/tmp/rel47/C-drill.log`（均未入库，日志内无令牌）。
- 事后自查：日志中不存在非 hex 的 32 位随机串，`Bearer` 出现次数为 0。

关于动词：task-29 写的是 `POST /api/option/`，但路由表实际注册的是 **`PUT /api/option/`**
（`router/api-router.go:200 optionRoute.PUT("/", controller.UpdateOption)`），POST 会 404；
本次按路由实际用 **PUT**，语义等价（更新某个 option）。

### 3.1 完整原始日志（verbatim，不含令牌）

```console
### 2026-10-24T03:51:25Z C drill start
=== C1 演练前基线（只读） ===
  billing_setting.billing_mode       sha256=41db12c3042a188b len=68
  billing_setting.billing_expr       sha256=971c21f80fa06643 len=610
  ModelRatio                         sha256=466b1cc44e72c28f len=48
  CompletionRatio                    sha256=3f8060dac798fc33 len=44
  CacheRatio                         sha256=148c4a8c671b6d71 len=50
  CreateCacheRatio                   ABSENT
  ModelPrice                         sha256=44136fa355b3678a len=2
  ImageRatio                         ABSENT
  AudioRatio                         sha256=44136fa355b3678a len=2
  AudioCompletionRatio               sha256=44136fa355b3678a len=2
  GroupRatio                         ABSENT
  TopupGroupRatio                    ABSENT
  billing_mode 全文: {"deepseek-flash":"tiered_expr","deepseek-v4.1-flash":"tiered_expr"}
  billing_expr keys: ['deepseek-flash', 'deepseek-v4.1-flash']
  access_token 原值: NULL

=== C2 生成临时令牌并写入 users.access_token（令牌值不打印） ===
  token_len = 32 (值不打印)
  写入完成，库中长度 = 32

=== C3 鉴权 + 用管理接口制造坏状态 ===
  [auth check] GET /api/option/ -> http=200 success=True
  坏 mode 值 = {"deepseek-flash":"tiered_expr","deepseek-v4.1-flash":"ratio"}
  坏 expr 值 keys = ['deepseek-flash']  len=303
  PUT /api/option/ billing_setting.billing_mode     -> http=200 {"message":"","success":true}
  PUT /api/option/ billing_setting.billing_expr     -> http=200 {"message":"","success":true}

=== C3b 确认坏状态已生效 ===
  -- 读库 --
  billing_setting.billing_mode = {"deepseek-flash":"tiered_expr","deepseek-v4.1-flash":"ratio"}
     sha256=124ee8d715961fcc9153b94c03c9c2ee4c08f9ceb17dc8996d75690bebd77282
  billing_setting.billing_expr = {"deepseek-flash":"(weekday(\"Asia/Shanghai\") >= 1 && weekday(\"Asia/Shanghai\") <= 5 && ((hour(\"Asia/Shanghai\") >= 9 && hour(\"Asia/Shanghai\") < 12) || (hour(\"Asia/Shanghai\") >= 14 && hour(\"Asia/Shanghai\") < 18))) ? tier(\"peak\", p*2 + c*8 + cr*0.04) : tier(\"offpeak\", p*1 + c*4 + cr*0.02)"}
     sha256=3ce6c624eb707a3c2a8714db0babe8871f162fc23a247f6f314d2f94320ebda4
  -- GET /api/pricing（公开接口，只取 deepseek 条目）--
  http=200
  pricing: deepseek-flash         billing_mode=tiered_expr  billing_expr=(weekday("Asia/Shanghai") >= 1 && weekday("Asia/Shanghai") <

=== C4 调用重置 POST /api/option/rest_model_ratio ===
  http=200 body={"message":"重置模型倍率成功","success":true}

=== C5 验证修复（读库断言） ===
  重置后 billing_mode = {"deepseek-flash": "tiered_expr", "deepseek-v4.1-flash": "tiered_expr"}
  重置后 billing_expr keys = ['deepseek-flash', 'deepseek-v4.1-flash']
  deepseek-flash     expr sha256 = 5705272e190aacb672a4a6bafe8b606ff006f0d6ef7b94cf5d067fae2025471f
  deepseek-v4.1-flash expr sha256 = 5705272e190aacb672a4a6bafe8b606ff006f0d6ef7b94cf5d067fae2025471f
  ASSERT mode 两行都是 tiered_expr      : PASS
  ASSERT 两行表达式逐字符相同            : PASS
  ASSERT 与演练前默认值逐字节一致        : FAIL
  ASSERT GroupRatio         未被改动 : PASS (演练前=None 现在=None)
  ASSERT TopupGroupRatio    未被改动 : PASS (演练前=None 现在=None)
  --- 其它定价键与基线一致性 ---
    ModelRatio               与基线一致
    CompletionRatio          与基线一致
    CacheRatio               与基线一致
    CreateCacheRatio         与基线不同 (was=None now={})
    ModelPrice               与基线一致
    ImageRatio               与基线不同 (was=None now={})
    AudioRatio               与基线一致
    AudioCompletionRatio     与基线一致
  -- GET /api/pricing（重置后）--
  http=200
  pricing: deepseek-flash         billing_mode=tiered_expr  billing_expr=(weekday("Asia/Shanghai") >= 1 && weekday("Asia/Shanghai") <

=== C6 还原凭据 ===
  --- restore: users.access_token -> NULL ---
  access_token 现为: NULL
  复核 users.access_token: NULL
### 2026-10-24T03:51:26Z C drill end
```

### 3.2 环境提示

- `deepseek-v4.1-flash` **不在 `/api/pricing` 的返回列表**里（该接口只暴露目录内模型；本实例只列 `deepseek-flash`），
  所以「坏状态已生效」的确认以**读库**为主、`/api/pricing` 为辅；两处证据都在上面的日志里。
- 演练用的两个模型都是默认模板模型，默认目录就是 `deepseek-flash` 与 `deepseek-v4.1-flash`
  （`setting/ratio_setting/model_ratio.go`、`setting/billing_setting/tiered_billing.go`），
  因此「重置后两行相同」正是默认状态应有的样子。

## 4. 关键对照：坏状态 → 重置后（「用户实例会被治好」的直接证据）

| 观察点 | 演练前基线 | 坏状态（模拟用户实例） | 调用重置后 | 判定 |
|---|---|---|---|---|
| `billing_mode`（两模型） | 都是 tiered_expr | `{"deepseek-flash":"tiered_expr","deepseek-v4.1-flash":"ratio"}`（sha256 `124ee8d7…`） | `{"deepseek-flash": "tiered_expr", "deepseek-v4.1-flash": "tiered_expr"}` | **PASS** |
| `billing_expr` 键 | 两个键 | **只剩 `deepseek-flash`**（len=303，sha256 `3ce6c624…`） | 两个键都回来了 | **PASS** |
| 两行表达式 sha256 | — | 一行缺失（无法比较） | 两行均为 **`5705272e190aacb672a4a6bafe8b606ff006f0d6ef7b94cf5d067fae2025471f`**（len=266，逐字符相同） | **PASS** |
| `GroupRatio` | 行不存在 | 不存在 | **仍不存在（未被改动）** | **PASS** |
| `TopupGroupRatio` | 行不存在 | 不存在 | **仍不存在（未被改动）** | **PASS** |
| 重置接口响应 | — | — | `http=200 {"message":"重置模型倍率成功","success":true}` | 成功 |

语义级前后对比（把 JSON 解析后再比较，避免被格式差异误导）：

| key | 语义一致 | 原文一致 | 说明 |
|---|---|---|---|
| billing_setting.billing_mode | 是 | 是 | 完全一致 |
| billing_setting.billing_expr | 是 | 否 | **仅 JSON 空格格式不同，内容一致**（默认序列化带空格） |
| ModelRatio / CompletionRatio / CacheRatio / ModelPrice / AudioRatio / AudioCompletionRatio | 是 | 是 | 一致 |
| CreateCacheRatio / ImageRatio | 否 | 否 | 基线**缺失**该行 → 重置后写入默认空表 `{}`（修复的预期副作用：重置会补齐这 10 个键） |
| GroupRatio / TopupGroupRatio | 是 | 是 | 未被触碰 |

## 5. 回滚路径（未触发）

```console
$ docker rm -f new-api
$ docker run -d --name new-api --restart always --network host --log-opt max-size=10m --log-opt max-file=3 \
    -v /root/new-api-own/data:/data -e LOG_SQL_DSN=memory -e LOG_MEMORY_MAX_BYTES=200MB \
    -e LOG_MEMORY_MAX_ROWS=200000 -e LOG_CLEANUP_RETENTION_DAYS=7 ghcr.io/zhemed/new-api-own:v0.0.46
$ curl -s -o /dev/null -w '%{http_code}\n' http://127.0.0.1:3000/api/status   # 期望 200
```

`v0.0.46`、`v0.0.45` 镜像均在本地；`/root/new-api-own/data` 未受影响（演练只改了 options 表与用户令牌字段，
且令牌已还原）。

## 6. 风险与未完成

- 演练改动了实例的两处数据：`options`（billing_mode/billing_expr 等 10 个定价键，已回到默认）与
  `users.access_token`（已还原 NULL）。除此外未动任何数据。
- **`CreateCacheRatio`/`ImageRatio` 从「无行」变成「`{}` 空表」**：属修复的预期行为（重置会写满 10 个键），
  语义与默认一致；用户实例重置后同样会出现这两行，不影响计费。
- `billing_expr` 的 JSON 空格格式在重置前后不同（内容相同）：面板与计费按解析后的值使用，不影响行为；
  若用户对比原始字符串会看到差异，已在第 4 节注明。
- 本地 `latest` 仍指向 **v0.0.46**（上一轮对齐的结果），registry 的 `latest` 已由 CI 更新为 v0.0.47；
  是否再对齐一次由 Lead/用户决定（未在本次任务范围）。
- 面板内「立即更新」的真机演练仍未做（按 PRD 留待下一版本窗口；当前已是最新，点击只会提示「已是最新」）。
- 演练期间 `data/` 文件数 56 → 57（新容器启动日志），属正常运行产物。
- 未提交：本文件留工作区，待 Lead 按闸门收口（锚点 `[task:diagnose-deepseek-pricing-row-mismatch]`）。

## 7. 建议下一步

1. Lead 复核可抽查：`curl -s http://127.0.0.1:3000/api/status -I | grep -i x-new-api-version`（应 v0.0.47）与
   读库确认 `users.access_token` 为 NULL、两个模型 `billing_mode` 均为 `tiered_expr`。
2. 用户侧可在自己的实例上点一次「重置」验证同样效果（本演练已用管理接口复现其坏状态并验证修复）。
3. 若要保持本机 `latest` 与 registry 一致，可再执行一次 `docker tag ...:v0.0.47 ...:latest`（一条命令，需授权）。
4. 下一个版本发布后，补做「面板内立即更新」的真机演练，形成完整闭环。
