# 清理证据：旧 new-api 镜像标签 + 悬空镜像（卷部分按 gate 主动停止）

- 任务：Trellis `10-23-cleanup-volumes-images-and-docs`；共享任务板 `task-10`；执行者 worker-containers
- 执行时间：2026-10-23T17:11:41Z ～ 17:13:32Z（UTC），全程只访问本机
- 事前依据：PRD `prd.md:21-26`（R1-R4）、`prd.md:25`（R3 出现命名卷即停止并报告）；用户已点名授权
- 红线遵守：只按「仓库:标签」删镜像（**未**按镜像 ID 删）；未碰其它仓库镜像；未删运行中容器使用的镜像/卷；
  未删 `/root/new-api-own/data`；未提交未推送；未升级镜像
- 结论摘要：**镜像部分完成并通过全部验收；卷部分按 R3 停手且未删任何卷**（原因见第 0 节）

## 0. 关键发现：侦察结论与实际不符（卷部分停手的原因）

PRD/任务描述写的是「`dangling=true` 39 个，**全为 64 位十六进制匿名卷，无命名卷**」。
实测**不符**：39 个悬空卷里包含 **2 个命名卷**。

```console
### 2026-10-23T17:13:32Z dangling volume breakdown
dangling_total=39  anonymous_64hex=37  named=2
--- NAMED dangling volumes (NOT authorized -> preserved) ---
  PRESERVE: halo-gradle-cache
  PRESERVE: picsur_picsur-data
--- container usage of the 2 named dangling volumes ---
  halo-gradle-cache -> containers: （无容器使用，确为 dangling）
  picsur_picsur-data -> containers: （无容器使用，确为 dangling）
```

判定：

- 授权对象是「悬空**匿名**卷」；实际存在的匿名悬空卷是 **37 个**，不是 39 个。
- 若照原计划执行 `docker volume prune -f`，它会连带删除 `halo-gradle-cache` 与 `picsur_picsur-data`
  这两个**其它应用的命名卷**（`picsur_picsur-data` 65.63MB，命名形态明显是应用数据卷），
  属超出点名范围、且可能造成其它应用数据丢失。
- 因此按 R3「出现命名卷即停止并报告」**停止卷清理，未执行 prune，未删除任何卷**。

追加核查（对决策有直接影响，见下方原始输出）：本机 Docker 29.7.2 的 `docker volume prune`
**默认只删匿名未用卷**，命名卷需显式 `-a/--all` 才会被删。也就是说，原计划的
`docker volume prune -f` 实际**不会**碰到那两个命名卷——R3 停手条件想防的风险在默认语义下并不存在。
但因为「出现命名卷即停手」是明确指令，我仍按要求停手，把选择权交回 Lead（第 6 节选项）。

```console
$ docker volume prune --help
Usage:  docker volume prune [OPTIONS]
Remove unused local volumes
Options:
  -a, --all             Remove all unused volumes, not just anonymous ones
      --filter filter   Provide filter values (e.g. "label=<label>")
  -f, --force           Do not prompt for confirmation

$ docker version --format '{{.Server.Version}}'
29.7.2   (API 1.55)
```

## 1. 清理前基线

```console
### 2026-10-23T17:11:41Z BASELINE
--- df -h / ---
Filesystem      Size  Used Avail Use% Mounted on
/dev/sda2       251G  116G  123G  49% /
--- df -B1 / (bytes) ---
/dev/sda2      269425360896 124383657984 131281141760  49% /
--- docker system df ---
TYPE            TOTAL     ACTIVE    SIZE      RECLAIMABLE
Images          122       8         19.64GB   14.1GB (71%)
Containers      8         8         262.1kB   0B (0%)
Local Volumes   40        1         1.86GB    1.678GB (90%)
Build Cache     636       2         49.88GB   24.17GB

--- images: new-api-own (14 tags) ---
ghcr.io/zhemed/new-api-own:v0.0.45 | e114b17532f1
ghcr.io/zhemed/new-api-own:v0.0.44 | 92d066b505ae
ghcr.io/zhemed/new-api-own:0.0.9  | f326deaef922
ghcr.io/zhemed/new-api-own:0.0.8  | fdb3883105ce
ghcr.io/zhemed/new-api-own:0.0.7  | 68c9842d16fb
ghcr.io/zhemed/new-api-own:latest | 68c9842d16fb   <- 过期 latest
ghcr.io/zhemed/new-api-own:v0.0.7 | 68c9842d16fb
ghcr.io/zhemed/new-api-own:0.0.6  | a74d0a6c0940
ghcr.io/zhemed/new-api-own:0.0.5  | 79f4bdcb712f
ghcr.io/zhemed/new-api-own:v0.0.4 | e850cb5dc8b0
ghcr.io/zhemed/new-api-own:0.0.3  | 3d04916fe29a
ghcr.io/zhemed/new-api-own:v0.0.3 | 3d04916fe29a
ghcr.io/zhemed/new-api-own:0.0.2  | e16437fdc39f
ghcr.io/zhemed/new-api-own:v0.0.2 | e16437fdc39f
tag_count=14

--- dangling images (untagged) before ---
53cdcb06feb5 | <none>:<none> | 7 days ago  | 756MB
224c8c749624 | <none>:<none> | 8 days ago  | 803MB
a3d8aaa63ed8 | <none>:<none> | 3 years ago | 37.4MB
bbfcd601d7cd | <none>:<none> | 56 years ago | 21.8MB

--- volumes ---
volume_total=40  volume_dangling=39
--- containers: 8 total, 8 running ---
new-api | ghcr.io/zhemed/new-api-own:v0.0.45 | Up
komari-agent-test | ghcr.io/zhemed/komari-agent:latest | Up
komari-test | ghcr.io/zhemed/komari:latest | Up
picsur | ghcr.io/zhemed/picsur:allinone | Up 3 days (healthy)
support-devdb-1 | postgres:17-alpine | Up 3 days
litepan | ghcr.io/zhemed/litepan:v0.0.49 | Up 3 days
halo | ghcr.io/zhemed/halo:v2.26.1-custom.1 | Up 3 days (healthy)
halodb | postgres:15.4 | Up 3 days (healthy)

--- new-api health before ---
state=running restarts=0 mount=/root/new-api-own/data:/data
http=200 version=v0.0.45
```

## 2. 镜像清理（R1）：逐个按「仓库:标签」删除

删前守护：先证明这 12 个待删标签对应的镜像 ID **没有任何容器在用**（含已停容器；实际 0 个已停容器）。

```console
--- pre-check: target image IDs vs container-referenced IDs ---
container-referenced: 4dd8bd40598b... 6ddd6e7973da... 8b4d729b7727... b0f9560a2de0...
                      cb4b4cec1be6... d1168ad5397b... e114b17532f1... fc0fbf2ce0f6...
  pre-check PASS (none of the 12 tags is used by a container)
```

```console
### 2026-10-23T17:12:36Z STEP1 delete old new-api-own tags (repo:tag only, never by ID)
$ docker rmi ghcr.io/zhemed/new-api-own:v0.0.2
Untagged: ghcr.io/zhemed/new-api-own:v0.0.2
$ docker rmi ghcr.io/zhemed/new-api-own:0.0.2
Untagged: ghcr.io/zhemed/new-api-own:0.0.2
Deleted: sha256:e16437fdc39f5e84da53057876c48929034a129dab6ac3ecb65e50da81009b2d
$ docker rmi ghcr.io/zhemed/new-api-own:v0.0.3
Untagged: ghcr.io/zhemed/new-api-own:v0.0.3
$ docker rmi ghcr.io/zhemed/new-api-own:0.0.3
Untagged: ghcr.io/zhemed/new-api-own:0.0.3
Deleted: sha256:3d04916fe29a32f89dbf887792498d8866c3e7720dc9282cf9a272c40c740352
$ docker rmi ghcr.io/zhemed/new-api-own:v0.0.4
Untagged: ghcr.io/zhemed/new-api-own:v0.0.4
Deleted: sha256:e850cb5dc8b035fdee2b56e166d333077b3f9b444ab23ceaf06fafef24ab9ec8
$ docker rmi ghcr.io/zhemed/new-api-own:0.0.5
Untagged: ghcr.io/zhemed/new-api-own:0.0.5
Deleted: sha256:79f4bdcb712f9a3e5b838a2e7ca52159755c77bfd92e0c7b1d17fd8c8900a3a3
$ docker rmi ghcr.io/zhemed/new-api-own:0.0.6
Untagged: ghcr.io/zhemed/new-api-own:0.0.6
Deleted: sha256:a74d0a6c09401d69a0181ca60df955e497fa8547c9991e2a70713af958971b2c
$ docker rmi ghcr.io/zhemed/new-api-own:v0.0.7
Untagged: ghcr.io/zhemed/new-api-own:v0.0.7
$ docker rmi ghcr.io/zhemed/new-api-own:0.0.7
Untagged: ghcr.io/zhemed/new-api-own:0.0.7
$ docker rmi ghcr.io/zhemed/new-api-own:0.0.8
Untagged: ghcr.io/zhemed/new-api-own:0.0.8
Deleted: sha256:fdb3883105ce1c7375af38d74bb26028c1561af76882a33906ab992a57b9d547
$ docker rmi ghcr.io/zhemed/new-api-own:0.0.9
Untagged: ghcr.io/zhemed/new-api-own:0.0.9
Deleted: sha256:f326deaef9228133b55f28d6289c7186c678b86e516404c091881fd93950115a
$ docker rmi ghcr.io/zhemed/new-api-own:v0.0.44
Untagged: ghcr.io/zhemed/new-api-own:v0.0.44
Deleted: sha256:92d066b505ae98ecb0515ed7ae64008a1d62ef6f575d57f2da1e2b82fbe6b83f
```

共删 12 个标签；其中 8 个镜像因最后一个标签被移除而整镜像删除；
`68c9842d16fb`（v0.0.7/0.0.7/latest 共用）在删掉 v0.0.7、0.0.7 后仍被 `latest` 引用，
由第 3 步移走 `latest` 后转为悬空镜像，由第 4 步清理。

## 3. `latest` 对齐（R2）

```console
### 2026-10-23T17:12:37Z STEP2 point latest at v0.0.45
$ docker tag ghcr.io/zhemed/new-api-own:v0.0.45 ghcr.io/zhemed/new-api-own:latest
tag_exit=0
--- images after step1+2 ---
ghcr.io/zhemed/new-api-own:latest | e114b17532f1 | 3 hours ago
ghcr.io/zhemed/new-api-own:v0.0.45 | e114b17532f1 | 3 hours ago
tag_count=2
```

`latest` 由旧镜像 `68c9842d16fb`（2026-10-06）改为与在跑版本同一镜像 `e114b17532f1`。

## 4. 悬空镜像清理（R4）

```console
### 2026-10-23T17:12:53Z STEP4 dangling images: list before prune
dangling_count_before=5
53cdcb06feb5 | <none>:<none> | 7 days ago    | 756MB
224c8c749624 | <none>:<none> | 8 days ago    | 803MB
68c9842d16fb | <none>:<none> | 2 weeks ago   | 302MB   (旧 new-api 镜像，第 3 步后失去全部标签)
a3d8aaa63ed8 | <none>:<none> | 3 years ago   | 37.4MB
bbfcd601d7cd | <none>:<none> | 56 years ago  | 21.8MB
--- verify: none of these IDs is referenced by any container ---
  OK not-used-by-container: 53cdcb06feb5
  OK not-used-by-container: 224c8c749624
  OK not-used-by-container: 68c9842d16fb
  OK not-used-by-container: a3d8aaa63ed8
  OK not-used-by-container: bbfcd601d7cd
--- docker image prune -f ---
Deleted Images:
untagged: sha256:bbfcd601d7cd... deleted: sha256:bbfcd601d7cd...
untagged: sha256:224c8c749624... deleted: sha256:224c8c749624...
deleted: sha256:5b19964679d0...
deleted: sha256:64c0209f07a4...
untagged: sha256:53cdcb06feb5...
untagged: sha256:68c9842d16fb...
deleted: sha256:68c9842d16fb...
deleted: sha256:b9fe428b2d72...
deleted: sha256:a80ff84f954f...
untagged: sha256:a3d8aaa63ed8...

Total reclaimed space: 222.9MB
dangling_count_after=0
```

5 个悬空镜像全部为 `<none>:<none>` 无标签形态、且无任何容器引用；`docker image prune -f`
只清悬空镜像（未用 `-a`，不碰有标签的镜像）。prune 自身报告回收 222.9MB
（其余层与在用镜像共享，故小于各镜像显示体积之和）。

## 5. 验收（镜像部分）

```console
### 2026-10-23T17:13:16Z VERIFY
--- images: new-api-own (expect only v0.0.45 + latest, same ID) ---
ghcr.io/zhemed/new-api-own:latest | e114b17532f1 | 3 hours ago | 302MB
ghcr.io/zhemed/new-api-own:v0.0.45 | e114b17532f1 | 3 hours ago | 302MB
tag_count=2
--- images total ---
images_total=146  dangling=0

--- containers: all 8 must be Up ---
new-api | ghcr.io/zhemed/new-api-own:v0.0.45 | Up 9 minutes
komari-agent-test | ghcr.io/zhemed/komari-agent:latest | Up 7 hours
komari-test | ghcr.io/zhemed/komari:latest | Up 5 hours
picsur | ghcr.io/zhemed/picsur:allinone | Up 3 days (healthy)
support-devdb-1 | postgres:17-alpine | Up 3 days
litepan | ghcr.io/zhemed/litepan:v0.0.49 | Up 3 days
halo | ghcr.io/zhemed/halo:v2.26.1-custom.1 | Up 3 days (healthy)
halodb | postgres:15.4 | Up 3 days (healthy)
running_count=8  all_count=8
  逐个 State.Status：komari-agent-test=running / komari-test=running / picsur=running /
  support-devdb-1=running / litepan=running / halo=running / halodb=running / new-api=running

--- new-api health ---
state=running restarts=0 mount=/root/new-api-own/data:/data
HTTP/1.1 200 OK
X-New-Api-Version: v0.0.45

--- volumes (UNTOUCHED by this task) ---
volume_total=40  volume_dangling=39
named volumes still present:
  halo-gradle-cache
  picsur_picsur-data
  support_db-data

--- /root/new-api-own/data intact ---
files=55  dir_exists=yes
```

### 5.1 验收对照（PRD AC）

| AC | 内容 | 状态 | 证据 |
|---|---|---|---|
| AC1 | 只剩 v0.0.45 与 latest 且同 ID e114b17532f1 | 达标 | 第 3、5 节 |
| AC2 | 8 容器全 Up；new-api restarts=0、200+v0.0.45、挂载工作区 | 达标 | 第 5 节 |
| AC3 | 匿名悬空卷清零，命名悬空卷按裁定保留 | 达标（17:17:42Z：总数 40 → 3、悬空 39 → 2 且均为命名卷、匿名卷残留 0） | 第 9-11 节 |
| AC4 | 磁盘回收前后对比 | 达标 | 第 5.2 节 |
| AC5 | README/spec 文档改动 | **不属于本 task-10** | PRD R6（另有人负责） |
| AC6 | 证据文件齐备 | 本文件即 docker 清理证据 | 全文 |

### 5.2 磁盘回收前后对比（`df -h /` 与精确字节）

| 指标 | 清理前（17:11:41Z） | 清理后（17:13:32Z） | 变化 |
|---|---|---|---|
| df 已用 | 116G | 115G | -1G |
| df 可用 | 123G | 124G | +1G |
| df 已用（字节） | 124383657984 | 122713034752 | -1670623232（约 1.56 GiB） |
| docker 镜像体积 | 19.64GB（可回收 14.1GB） | 17.96GB（可回收 12.47GB） | -1.68GB |
| docker 镜像数量 | 122 | 110 | -12 |
| 悬空镜像 | 4 | 0 | -4 |
| 本地卷 | 40（悬空 39） | 40（悬空 39） | 未动 |

## 6. 卷部分：停手报告与待决策选项

- 事实：悬空 39 = 匿名 64 位十六进制 **37** 个 + 命名 **2** 个（`halo-gradle-cache`、`picsur_picsur-data`）。
- 两个命名卷当前**无任何容器挂载**（确为 dangling），其中 `picsur_picsur-data` 65.63MB，
  命名形态指向其它应用（picsur / halo）的数据或缓存。
- 卷总量 1.86GB，`docker system df` 报告可回收 1.678GB；扣除在用 `support_db-data`(182.3MB) 与
  `picsur_picsur-data`(65.63MB)，匿名悬空卷合计约 1.61GB。
- 本任务**未执行** `docker volume prune -f`，**未删除任何卷**，两个命名卷原样保留。

待 Lead 决策的选项：

| 选项 | 动作 | 影响 |
|---|---|---|
| A（推荐） | `docker volume prune -f`（不传 `-a`） | 按本机 Docker 29.7.2 默认语义只删 37 个匿名未用卷，命名卷不受影响；回收约 1.61GB；达到「悬空匿名卷 = 0」 |
| B | 逐个 `docker volume rm` 已核验的 37 个匿名悬空卷（显式名单） | 与 A 等效，且完全不依赖 prune 默认语义（更保守、更啰嗦） |
| C | 连 2 个命名卷一起清（`prune -a` 或显式 rm） | 多回收约 66MB；需用户对其它应用明确点头，有删掉其它应用数据/缓存的风险 |
| D | 全部保留不动 | 0 风险；1.678GB 空间不回收，AC3 保持未达标 |

注：A/B 都不会删除 `support_db-data`（正在被 `support-devdb-1` 使用，非 dangling）。

## 7. 风险与未完成

- **AC3 未达标**：悬空卷仍 39（37 匿名 + 2 命名）；这是按 PRD R3 的停手条件刻意造成的，不是执行失败。
- 侦察基线有误：PRD/任务描述称「39 个全为匿名卷、无命名卷」，实测含 2 个命名卷；
  后续任务引用该基线时请以本文件第 0 节为准。
- 未做（范围外）：README/spec 文档改动（PRD R6，属 Trellis 同一任务的另一部分）、
  Build Cache 清理（49.88GB，其中 24.22GB 可回收，**未授权未动**）、备份策略（用户本轮未选）。
- 磁盘回收量包含镜像标签删除与悬空镜像清理两部分，`docker image prune` 自报 222.9MB，
  而整体镜像体积下降 1.68GB（差额来自被删标签对应的镜像层）。
- 未提交：本文件留工作区，待 Lead 按闸门提交。

## 8. 建议下一步

1. Lead 复核后对第 6 节选项拍板；选 A（`docker volume prune -f`）或 B（逐个 `rm` 37 个匿名卷）
   我可立即执行并补齐本文件，随后 complete task-10。
2. 文档部分（README/spec）与本 task-10 并行推进，提交时一并带上本证据。
3. 后续如需回收磁盘，优先考虑 Build Cache（24.22GB 可回收，收益远大于卷），但属独立授权项。

---

# 追加：裁定 B 执行（删除 37 个匿名悬空卷，显式名单，未使用 prune）

- 裁定来源：Lead 2026-10-23 转述——走 **B（最保守）**：只删已核验的 37 个匿名悬空卷，
  显式名单逐个 `docker volume rm`，不用 `docker volume prune`；两个命名卷
  （`halo-gradle-cache`、`picsur_picsur-data`）属其它应用、不在授权范围，保留不动；
  Build Cache 本轮不清。
- 执行时间：2026-10-23T17:17:19Z ～ 17:17:42Z（UTC）
- 结果：**37 个全部删除成功（failed=0）**；卷总数 40 → 3；匿名悬空卷残留 0；其余验收全绿

## 9. 删前复核（17:17:19Z）

```console
### 2026-10-23T17:17:19Z STEP-B1 pre-delete recheck (option B)
--- all volumes (expect 40) ---
  （输出 40 行：37 个 64 位十六进制匿名名 + halo-gradle-cache、picsur_picsur-data、support_db-data）
volume_total=40
--- dangling volumes (expect 39) ---
  （输出 39 行：37 个匿名名 + halo-gradle-cache、picsur_picsur-data）
volume_dangling=39
--- named volumes (must be PRESERVED) ---
  PRESERVE: halo-gradle-cache
  PRESERVE: picsur_picsur-data
  PRESERVE: support_db-data

--- build explicit delete list = dangling AND anonymous 64-hex ---
delete_list_count=37
--- guards ---
volumes mounted by any container: support_db-data
  no candidate is mounted by any container: OK
GUARDS=PASS (37 candidates verified: dangling + anonymous + unmounted + not in preserve set)
```

守护逐项含义：37 个候选逐个通过「名字为 64 位十六进制」「存在于卷列表」「确为 dangling」
「不在保留清单」「不被任何容器挂载」五项断言；`support_db-data` 是唯一被容器挂载的卷（非候选）。
完整 37 个名单落在 `/tmp/migration-evidence/vol-delete-list.txt`，本文档第 10 节逐行回显。

## 10. 逐个删除（17:17:29Z，显式名单、单条执行、失败不重试）

删除前磁盘基线：`df -h /` 已用 115G；`docker system df` 本地卷 40 个 / 1.86GB / 可回收 1.678GB。

```console
### 2026-10-23T17:17:29Z STEP-B2 delete 37 anonymous dangling volumes (explicit names, one by one, no prune)
$ docker volume rm 04a0fa68b0e811122f4d6690e748e7cd5ef88d2ddce1d7d6234345fe8f3f83f5  -> rc=0
$ docker volume rm 0a4c2ff361112bcc29d30aac809fd8ce979cd8618593838041826ac60fd7d2e3  -> rc=0
$ docker volume rm 2c295bef0989e55eff5aa938f156f7be9796e955fa31da15ac97d03bb2470532  -> rc=0
$ docker volume rm 33f18fe5d4e6cc66d0cc5ff451671c6a0c1719c0a6d6eaae82cfff2453ff92d9  -> rc=0
$ docker volume rm 3654f60dfe034aae532463c6e610f04996594656afd51ff988810650c984c0f4  -> rc=0
$ docker volume rm 367921b6635aaeff224fd3a80e3f798305f2b766c72d9164dd100335f43062ae  -> rc=0
$ docker volume rm 37e76095bae82780a24e593f58359ccea1120f77d50fb1ce73129c3934d640f4  -> rc=0
$ docker volume rm 38b256b7e544e38a84c4fa8a38face0764516a90766d42781d8227001b6620ad  -> rc=0
$ docker volume rm 41bf84273ab31da643270170850d268ca4a29140448b3ec2cfd7025b60274ad3  -> rc=0
$ docker volume rm 41e48fa3642b0b2d574419166633ba729dc1d6971cb49b7c9ebc47da9942f081  -> rc=0
$ docker volume rm 42cadfe38599f8857875273e66b529579d0c0533dbb9111c6900629557de58e6  -> rc=0
$ docker volume rm 42dd57a0c1209cad196b73533e7fd27599a24fa6e2e6683bd72c5e277359cf8f  -> rc=0
$ docker volume rm 47799a06441293ec265c9ad9f2e377b7daffab5a4367376d5179aaa37e12d35d  -> rc=0
$ docker volume rm 480ae09ec23396a646a25f5f7b4acb3e556e00477a96f5c56ecfe7bf348631c9  -> rc=0
$ docker volume rm 50b4e938f87a6b578fd5854a58a49dad8bfedcd13359fba032b222b60b5c4337  -> rc=0
$ docker volume rm 512da3017f506496fd4119cf5d9da64884d23467d6766a8755332bdaf08c92f9  -> rc=0
$ docker volume rm 5c25819338598f538d67b41b5b6ebe3bb462ea07213f0312211c4b4bf0d5b4dc  -> rc=0
$ docker volume rm 6783a099dc94fdab842c4f2d08b349a84b7c825e8a0da982d234c5de243f87a9  -> rc=0
$ docker volume rm 710db365c7f7059c3171e57b106c362f874eb7adfb58398697c51623ca925c0f  -> rc=0
$ docker volume rm 77f93c6e6f24375c8c707eff520e56416c9cf1aff88a5872dad286315b84e3d8  -> rc=0
$ docker volume rm 7887065c0ab2bae92f1bffa84a35e9821ed9a2680d0076622fd19df5a6fc8833  -> rc=0
$ docker volume rm 79597b3cd83ee59c177a75bf3506d737bf256db52cc06625f0e30e26dd22ffcc  -> rc=0
$ docker volume rm 8f5f1860e389983f29bf4e2bb6618e5d1a2ac11b840151cbff45941437ceea1b  -> rc=0
$ docker volume rm 9822318e15a3282958a02336d5c54611f6e464df78fe38f19f333da8f5d78103  -> rc=0
$ docker volume rm a326e6ea19e1b5375ab32e5411e2583a63a5c1b44df0683dea7d79656def244c  -> rc=0
$ docker volume rm a3b7323dbe47548909aa55ffd0d2980887e0aa6eed82e6f8039cadf4f188990c  -> rc=0
$ docker volume rm aaa6285ab6fd9347049a1d42cb5ea5037a3afc5442c9cec02f7ce34e9c60f9e3  -> rc=0
$ docker volume rm b120c1587497a01aa0af9ccf078eca0bf302fad37f1b5a6663461d44a04f18a1  -> rc=0
$ docker volume rm b252b49bfef446794c081336cab71874c3cb442a7ea93688681c834e3156605a  -> rc=0
$ docker volume rm b8c40d2a9c6c358afad1864a1c10370c762b996f7b4e4cc06b5eeff1c840c69c  -> rc=0
$ docker volume rm c0b06a23c4b66f1e1455aeb3dd345848ba96fcac0b248fd3fc05e92a07683df3  -> rc=0
$ docker volume rm d3fe6e9feaf6499dafb757e31aa6925a22300a7edc041088cb6c9884fbefa205  -> rc=0
$ docker volume rm d79f14c2ad704bf4c11930620dca8b1b7ec084f72b37a13374c62eccfa83a6f9  -> rc=0
$ docker volume rm d8b34a8d4efa4fbe3391f84c8b9a44f814e63608c9035bd8836d92bf1d42f238  -> rc=0
$ docker volume rm de473d3c8b6d2ce331bb7353555ff001c88085f294da0942e90f4fe547ca641c  -> rc=0
$ docker volume rm ea5b25aed45573401020222432b1446f143c2df81dac070af0b87a7281e22246  -> rc=0
$ docker volume rm ee86b51e890520252bc04cf05d5afdda46c0ffc5333ad3c0d2695c8b22c76131  -> rc=0
attempted=37  failed=0
```

每条命令只用 64 位十六进制字面量卷名；未使用通配符，未使用 `docker volume prune`；
`docker volume rm` 成功时原样回显卷名，逐条 rc=0。

## 11. 删后验收（17:17:42Z）

```console
### 2026-10-23T17:17:42Z STEP-B3 post-delete verification
--- volumes remaining (expect 3) ---
DRIVER    VOLUME NAME
local     halo-gradle-cache
local     picsur_picsur-data
local     support_db-data
volume_total=3
volume_dangling=2
--- required named volumes present? ---
  OK present: halo-gradle-cache
  OK present: picsur_picsur-data
  OK present: support_db-data
--- dangling list (expect exactly the 2 named ones) ---
  halo-gradle-cache
  picsur_picsur-data
--- removed list check: any leftover anonymous volume? (expect 0) ---
0

--- containers (expect 8 running) ---
new-api | Up 13 minutes ; komari-agent-test | Up 7 hours ; komari-test | Up 5 hours ;
picsur | Up 3 days (healthy) ; support-devdb-1 | Up 3 days ; litepan | Up 3 days ;
halo | Up 3 days (healthy) ; halodb | Up 3 days (healthy)
running_count=8  all_count=8

--- new-api health ---
state=running restarts=0 mount=/root/new-api-own/data:/data
HTTP/1.1 200 OK
X-New-Api-Version: v0.0.45

--- images (unchanged) ---
ghcr.io/zhemed/new-api-own:latest | e114b17532f1
ghcr.io/zhemed/new-api-own:v0.0.45 | e114b17532f1

--- df -h / (after) ---
/dev/sda2       251G  114G  125G  48% /
/dev/sda2      269425360896 121845747712 133819052032  48% /
--- docker system df (after) ---
Images          110       8         17.96GB   12.47GB (69%)
Containers      8         8         262.1kB   0B (0%)
Local Volumes   3         1         996.9MB   814.6MB (81%)
Build Cache     636       2         49.88GB   24.22GB

--- /root/new-api-own/data intact ---
dir_exists=yes files=55
```

保留的 814.6MB「可回收」全部来自两个命名悬空卷（按裁定保留），不含任何匿名卷。

## 12. task-10 全阶段空间回收对比

| 指标 | 起点（17:11:41Z） | 镜像清理后（17:13:32Z） | 删卷后（17:17:42Z） | 合计变化 |
|---|---|---|---|---|
| df 已用 | 116G | 115G | 114G | -2G |
| df 可用 | 123G | 124G | 125G | +2G |
| df 已用（字节） | 124383657984 | 122713034752 | 121845747712 | -2537910272（约 2.36 GiB） |
| docker 镜像 | 122 个 / 19.64GB | 110 个 / 17.96GB | 110 个 / 17.96GB | -12 个 / -1.68GB |
| 悬空镜像 | 4 | 0 | 0 | -4 |
| 本地卷 | 40 个 / 1.86GB | 40 个 / 1.86GB | 3 个 / 996.9MB | -37 个 / -约 890MB |
| 悬空卷 | 39（37 匿名 + 2 命名） | 39（37 匿名 + 2 命名） | 2（均为命名，按裁定保留） | 匿名 -37 |

## 13. 追加阶段的结论与风险

- 裁定 B 执行完毕：匿名悬空卷清零（残留 0），两个命名卷与其数据未被触碰，
  `support_db-data`（在用）未被触碰，8 个容器与实例健康不受影响。
- 未使用 `docker volume prune`（含默认语义），完全按显式名单执行——不依赖 Docker 默认行为，
  可逐条复核（第 10 节 37 行命令与返回码）。
- 剩余可回收空间提示：本地卷 814.6MB（两个命名卷，按裁定保留）、Build Cache 24.22GB（本轮不清）；
  两者均需新的授权才能动。
- 未提交：本文件留工作区，待 Lead 按闸门提交（锚点 `[task:cleanup-volumes-images-and-docs]`）。
