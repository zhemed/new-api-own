# 清理证据（第二轮）：Build Cache 13.26GB（task-17）

- 任务：Trellis `10-24-prune-buildcache-after-image-cleanup`；共享任务板 `task-17`；执行者 worker-containers
- 执行时间：2026-10-24T01:03:33Z ～ 01:04:35Z（UTC），全程只访问本机
- 事前依据：用户点名授权清 Build Cache；task-17 描述（承接 task-16 删除 84 镜像后缓存可回收量升至 13.26GB）
- 红线遵守：**只清构建缓存**；命令为 `docker builder prune -f`（**未加** `--all`、未用任何 `docker system prune` 系列）；
  未删镜像 / 容器 / 卷 / 网络 / `data/`；未提交未推送；未升级镜像

## 0. 结论摘要

| 项目 | 结果 |
|---|---|
| 命令 | `docker builder prune -f`（默认语义，无额外旗标） |
| 执行 | rc=0，耗时 43 秒，输出 259 行，末行 `Total: 13.17GB` |
| Build Cache | 429 条 / 25.75GB（可回收 13.26GB）→ **172 条 / 12.58GB（可回收 85.79MB）** |
| `df -B1 /` 已用 | 88320507904 → 74900303872，**实测回收 13420204032 B ≈ 12.50 GiB / 13.42 GB** |
| 实测 vs 预期 | 预期 13.26GB，prune 自报 13.17GB（差 0.7%）、df 实测 13.42GB（差 +1.2%）—— 见第 4 节 |
| 镜像清单 | **完全不变**：唯一 26、标签行 35、悬空 0 |
| 服务 | **零影响**：8 容器全 running、new-api 200 + v0.0.45、卷 3 个、`data/` 55 文件 |

## 1. 清前基线（01:03:33Z）

```console
### 2026-10-24T01:03:33Z PRE-CLEANUP baseline
--- docker system df ---
TYPE            TOTAL     ACTIVE    SIZE      RECLAIMABLE
Images          26        8         5.276GB   1.754GB (33%)
Containers      8         8         262.1kB   0B (0%)
Local Volumes   3         1         996.9MB   814.6MB (81%)
Build Cache     429       2         25.75GB   13.26GB
--- df -h / ---
/dev/sda2       251G   83G  156G  35% /
--- df -B1 / ---
/dev/sda2      269425360896 88320507904 167344291840  35% /
--- image counts ---
unique=26  tag_rows=35  dangling=0
--- containers --- running=8  all=8（逐个 State.Status 全为 running）
--- new-api ---
running restarts=0 mount=/root/new-api-own/data:/data
HTTP/1.1 200 OK
X-New-Api-Version: v0.0.45
--- volumes --- total=3  dangling=2
--- data dir --- files=55  exists=yes
--- buildx du (pre) ---
Shared:		12.49GB
Private:	13.26GB
Reclaimable:	25.75GB
Total:		25.75GB
```

`buildx du` 的 Private 13.26GB 与 `docker system df` 的 RECLAIMABLE 13.26GB 完全吻合 ——
即「可回收」= 已被镜像删除释放引用、但仍留在缓存里的私有记录（上轮 task-16 删除 84 个镜像的连带效果）。

## 2. 执行清理（01:03:42Z）

```console
$ docker builder prune -f
（输出 259 行，首行为表头）
ID						RECLAIMABLE	SIZE		LAST ACCESSED
p7f14vwft39ius7j5ufn7zzrj               	true 		4.207kB   	5 weeks ago
8squs072447hpoj4y87fula11               	true 	4.207kB   	4 weeks ago
…
fmg9rjz7n2otw18ayxj2mumwn               	true 	135.9MB   	Less than a second ago
hnzxvkyr8bg6vx1znxqsdgx4q               	true 	135.9MB   	Less than a second ago
Total:	13.17GB
prune_exit=0  elapsed=43s  raw_lines=259
```

- 未加 `--all`、未加 `--filter`、未使用 `docker system prune`（该系列会连带清镜像/容器/网络/卷，属红线）。
- 原始输出全文留在 `/tmp/bc2/prune.raw`（未入库，摘要如上）。

## 3. 清后核验（01:04:35Z）

```console
### 2026-10-24T01:04:35Z POST-CLEANUP verification
--- docker system df ---
TYPE            TOTAL     ACTIVE    SIZE      RECLAIMABLE
Images          26        8         5.276GB   1.754GB (33%)
Containers      8         8         262.1kB   0B (0%)
Local Volumes   3         1         996.9MB   814.6MB (81%)
Build Cache     172       2         12.58GB   85.79MB
--- buildx du (post) ---
Shared:		12.49GB
Private:	85.8MB
Reclaimable:	12.58GB
Total:		12.58GB
--- df -h / ---
/dev/sda2       251G   70G  169G  30% /
--- df -B1 / ---
/dev/sda2      269425360896 74900303872 180764495872  30% /
--- image counts (must be unchanged: 26/35/0) ---
unique=26  tag_rows=35  dangling=0
```

### 3.1 镜像清单（26 个唯一 / 35 标签，全部保留）

```console
alpine:3.21 | ee16d92f96db              alpine:3.23 | 85fe1e81d675
alpine:latest | 294b683cb724            curlimages/curl:latest | a134550be58a
debian:bookworm-slim | 7ff6ba31527b     debian:trixie-slim | 62bd423aeee9
eclipse-temurin:21-jdk | 4d0603880065   ghcr.io/zhemed/halo:v2.26.1-custom.1 | 8b4d729b7727
ghcr.io/zhemed/komari-agent:0.0.23 | d261e841e122
ghcr.io/zhemed/komari-agent:0.0.24 | cb4b4cec1be6
ghcr.io/zhemed/komari-agent:latest | cb4b4cec1be6
ghcr.io/zhemed/komari:0.0.23 | 4817dbd21193
ghcr.io/zhemed/komari:0.0.24 | fc0fbf2ce0f6
ghcr.io/zhemed/komari:latest | fc0fbf2ce0f6
ghcr.io/zhemed/litepan:0.0.49 | 6ddd6e7973da      ghcr.io/zhemed/litepan:0.0.50 | eeda8ce7d49a
ghcr.io/zhemed/litepan:latest | eeda8ce7d49a     ghcr.io/zhemed/litepan:v0.0.49 | 6ddd6e7973da
ghcr.io/zhemed/litepan:v0.0.50 | eeda8ce7d49a    ghcr.io/zhemed/new-api-own:latest | e114b17532f1
ghcr.io/zhemed/new-api-own:v0.0.45 | e114b17532f1
ghcr.io/zhemed/picsur:0.10 | 248755520402        ghcr.io/zhemed/picsur:0.10.6 | 248755520402
ghcr.io/zhemed/picsur:allinone | 4dd8bd40598b    ghcr.io/zhemed/picsur:latest | 248755520402
ghcr.io/zhemed/picsur:v0.10.6 | 248755520402     halohub/halo:2.26 | bfe89bf70c44
hello-world:latest | 5e2309035332            node:20-alpine | fb4cd12c85ee
postgres:15.4 | d1168ad5397b              postgres:17-alpine | b0f9560a2de0
redis:7 | 71da9275c5f3                    redis:latest | 298e5b3bc566
registry:2 | a3d8aaa63ed8                tonistiigi/binfmt:latest | 400a4873b838
```

### 3.2 保留清单抽样（12 项全部命中）

```console
  KEEP OK ghcr.io/zhemed/new-api-own:v0.0.45     e114b17532f1
  KEEP OK ghcr.io/zhemed/new-api-own:latest      e114b17532f1
  KEEP OK ghcr.io/zhemed/komari:0.0.24           fc0fbf2ce0f6
  KEEP OK ghcr.io/zhemed/komari:0.0.23           4817dbd21193
  KEEP OK ghcr.io/zhemed/komari-agent:0.0.24     cb4b4cec1be6
  KEEP OK ghcr.io/zhemed/komari-agent:0.0.23     d261e841e122
  KEEP OK ghcr.io/zhemed/picsur:allinone         4dd8bd40598b
  KEEP OK ghcr.io/zhemed/picsur:latest           248755520402
  KEEP OK ghcr.io/zhemed/litepan:0.0.49          6ddd6e7973da
  KEEP OK ghcr.io/zhemed/litepan:0.0.50          eeda8ce7d49a
  KEEP OK postgres:15.4                          d1168ad5397b
  KEEP OK postgres:17-alpine                     b0f9560a2de0
```

### 3.3 服务无损

```console
--- containers --- running=8  all=8
  逐个 State.Status：komari-agent-test=running / komari-test=running / picsur=running /
  support-devdb-1=running / litepan=running / halo=running / halodb=running / new-api=running
--- new-api ---
running restarts=0 mount=/root/new-api-own/data:/data
HTTP/1.1 200 OK
X-New-Api-Version: v0.0.45
--- volumes --- volumes=3  dangling_volumes=2（halo-gradle-cache / picsur_picsur-data / support_db-data）
--- data dir --- data_files=55  data_exists=yes
```

## 4. 前后对比与回收量核算

| 指标 | 清前（01:03:33Z） | 清后（01:04:35Z） | 变化 |
|---|---|---|---|
| Build Cache 体积 | 25.75GB（429 条） | 12.58GB（172 条） | -13.17GB / -257 条 |
| Build Cache 可回收 | 13.26GB | 85.79MB | -13.18GB |
| buildx du Private | 13.26GB | 85.8MB | -13.18GB |
| buildx du Shared | 12.49GB | 12.49GB | 不变 |
| df 已用 | 83G | 70G | -13G |
| df 可用 | 156G | 169G | +13G |
| df 已用（精确字节） | 88320507904 | 74900303872 | **-13420204032（12.50 GiB / 13.42 GB）** |
| 唯一镜像 / 标签行 / 悬空镜像 | 26 / 35 / 0 | 26 / 35 / 0 | 不变 |
| 容器 / 卷 / `data/` 文件 | 8 / 3 / 55 | 8 / 3 / 55 | 不变 |

回收量三个口径的差异说明（不凑数字，如实并列）：

1. **预期 13.26GB**：清前 `docker system df` 与 `buildx du` 的「可回收 / Private」。
2. **prune 自报 13.17GB**：它统计的是被删除缓存记录的大小之和，与实际磁盘块释放略有口径差。
3. **df 实测 13.42GB**：真实文件系统释放量；比 prune 自报多约 0.25GB（约 2%），
   属缓存记录统计口径与文件系统实际块释放（含元数据/对齐）之间的正常差异，也含命令执行期间其它写入的抵消。

结论：**本次回收达到预期的约 99%**（13.17/13.26），无需也不应为凑数字加 `--all` 等旗标。

## 5. 残留的 12.58GB 为什么不能再清

- 清后 Build Cache 仍有 12.58GB / 172 条，其中 **Shared 12.49GB、Private 仅 85.8MB**。
- Shared 意味着这些缓存记录与**现存 26 个镜像共享同一批层**；只要镜像还在，这些层就仍被引用，
  删除缓存记录不会释放磁盘（`docker system df` 也只把它们记为不可回收 85.79MB）。
- 要再压这部分，只能删镜像（本轮红线明确禁止，且镜像侧已按上轮授权清到只剩保留项），
  因此**当前状态下构建缓存已清无可清**。

## 6. 副作用与风险

- **其它项目下次构建会失去命中缓存、明显变慢**（需重新拉取/重算构建层）；**没有任何数据丢失** ——
  属用户已知并接受的预期副作用。
- 本轮动作不可逆（缓存需重新构建生成）；对运行中的 8 个容器与本项目实例**零影响**（第 3.3 节已核验）。
- 未触碰任何镜像 / 容器 / 卷 / 网络 / `data/`；镜像与卷计数前后逐项一致，可交叉验证。
- 未提交：本文件留工作区，待 Lead 按闸门收口（锚点 `[task:prune-buildcache-after-image-cleanup]`）。

## 7. 建议下一步

1. Lead 复核可抽查：`docker system df`（Build Cache 行应 172 条 / 12.58GB / 85.79MB）与 `df -B1 /`（应 ≈ 74.9GB 已用）。
2. 磁盘治理到此已到本项目授权边界内的尽头：镜像仅剩保留项（可回收 1.754GB 属 D 桶基础镜像，
   需新授权）、缓存残留 12.49GB 与镜像共享（不清镜像就动不了）、卷侧 814.6MB 为两个命名卷（按此前裁定保留）。
3. 若后续还要压缩，建议方向是「删除不再使用的 D 桶基础镜像」或「处理两个命名悬空卷」，
   两者都需用户对相应对象单独点头。
