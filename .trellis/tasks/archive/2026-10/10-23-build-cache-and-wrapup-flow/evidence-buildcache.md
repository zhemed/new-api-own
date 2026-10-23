# 清理证据：Docker Build Cache（task-12）

- 任务：Trellis `10-23-build-cache-and-wrapup-flow`；共享任务板 `task-12`；执行者 worker-containers
- 执行时间：2026-10-23T21:49:46Z ～ 21:52:16Z（UTC），全程只访问本机
- 事前依据：PRD `prd.md:19-22`（R1 Build Cache 清理）；用户已点名授权
- 红线遵守：**只清构建缓存**；未删除任何镜像、容器、卷或 `data/`；未使用 `docker system prune`；
  未提交未推送；未升级镜像
- 结论摘要：`docker builder prune -f` 一次执行即回收 **24.13GB**（精确 **24121233408 B**，为 24.22GB 目标的 99.6%），
  因此**未**再执行 `--all`；8 容器、实例、镜像、卷、`data/` 全部核验不变

## 1. 清前取证

### 1.1 容量与构建缓存归属

```console
### 2026-10-23T21:49:46Z PRE-CLEANUP evidence
--- df -h / ---
/dev/sda2       251G  114G  125G  48% /
--- df -B1 / ---
/dev/sda2      269425360896 121850494976 133814304768  48% /
--- docker system df ---
TYPE            TOTAL     ACTIVE    SIZE      RECLAIMABLE
Images          110       8         17.96GB   12.47GB (69%)
Containers      8         8         262.1kB   0B (0%)
Local Volumes   3         1         996.9MB   814.6MB (81%)
Build Cache     636       2         49.88GB   24.22GB

--- docker system df -v : Build cache 段（节选） ---
Build cache usage: 49.88GB
CACHE ID        CACHE TYPE     SIZE      CREATED        LAST USED      USAGE     SHARED
b5kaljt9ntl5*   regular        4.1kB     15 hours ago                  0         false
numrc64cuhzq    regular        14.5MB    5 weeks ago    5 weeks ago    1         true
ivrkh1wni9ks    regular        113MB     5 weeks ago    5 weeks ago    1         true
kw4lcuus4z2c    regular        24.7kB    5 weeks ago    5 weeks ago    2         true
（共 636 条缓存记录，多数 SHARED=true、USAGE 1-3）
```

### 1.2 归属确认：24.22GB 记在哪个 builder 名下

```console
--- buildx plugin ---
github.com/docker/buildx v0.37.1 0b265a9f62db554fa9aba6dd19e1bd5704bc7d8a

--- docker buildx ls ---
NAME/NODE     DRIVER/ENDPOINT   STATUS    BUILDKIT   PLATFORMS
default*      docker
 \_ default    \_ default       running   v0.32.2    linux/amd64 (+3), linux/arm64

--- docker buildx du（清前，耗时约 1 分钟） ---
nfrjxuqfy3j8vop7o6fuut6au    true          843.8MB    7 days ago
Shared:		25.66GB
Private:	24.22GB
Reclaimable:	49.88GB
Total:		49.88GB
```

归属结论：

- 全机只有一个 builder：`default`（buildx 默认实例，driver `docker`，BuildKit `v0.32.2`），
  **不存在**「docker 引擎缓存 + 独立 buildx builder 各存一份」的情况。
- `docker system df` 的 Build Cache 49.88GB 与 `docker buildx du` 的 Total 49.88GB 完全一致。
- 其中 **Private 24.22GB** 与 `docker system df` 的 **RECLAIMABLE 24.22GB** 逐位吻合 →
  「可回收 24.22GB」= default builder 的**私有（未被镜像引用）缓存记录**；
  另 25.66GB 标为 Shared（与在用镜像共享层），system df 不计入可回收。

### 1.3 旗标语义确认（本机 Docker 29.7.2）

```console
$ docker builder prune --help
Usage:  docker buildx prune
Remove build cache

Options:
  -a, --all                    Include internal/frontend images
      --builder string         Override the configured builder instance (default "default")
  -D, --debug                  Enable debug logging
      --filter filter          Provide filter values
  -f, --force                  Do not prompt for confirmation
      --max-used-space bytes   Maximum amount of disk space allowed to keep for cache
      --min-free-space bytes   Target amount of free disk space after pruning
      --reserved-space bytes   Amount of disk space always allowed to keep for cache
      --timeout duration       Override the default timeout for loading builder status (default 20s)
      --verbose                Provide a more verbose output
```

要点：`docker builder prune` 在本版本是 `docker buildx prune` 的别名，作用于 default builder；
默认语义为「清理未在用（可回收）的缓存记录」，`-a/--all` 额外包含 **internal/frontend images**
（BuildKit 前端内部镜像）；两者都**不会**删除用户镜像/容器/卷。

## 2. 清理执行（单次 `docker builder prune -f`）

```console
### 2026-10-23T21:50:00Z STEP1 docker builder prune -f (default semantics)
prune_exit=0  elapsed=70s
raw_output_lines=209
--- 输出前 3 行（被清理记录清单） ---
ID						RECLAIMABLE	SIZE		LAST ACCESSED
w4s5ghtzr6iahhshtjhkkh4m6*              	true 		34.58MB   	15 hours ago
3hxl8zn9v7y8ihxj8ib86677i*              	true 	2.056MB   	7 days ago
--- 输出末 5 行（汇总） ---
nfrjxuqfy3j8vop7o6fuut6au               	true 	843.8MB   	Less than a second ago
qvkeys73n0g28987z8y8ysn1y               	true 	258.1MB   	Less than a second ago
bkvcmm9cyh9gg2yl7fgrdqizm               	true 	76.27MB   	Less than a second ago
6v3fyfaoy6dz0d6vnvk35yv57               	true 	181.3MB   	Less than a second ago
Total:	24.13GB
```

- 命令：`docker builder prune -f`（未加 `--all`，未加任何 `--filter`）
- 结果：rc=0，耗时 70 秒，输出 209 行（208 条记录 + Total 行），**回收 24.13GB**
- 原始输出全文留在 `/tmp/buildcache-evidence/prune-default.raw`（未入库，摘要如上）

## 3. 为什么没有再跑 `--all`（决策与数据）

任务给出的触发条件是「默认只清 dangling 缓存、回收量**远低于** 24GB」。实测：

| 指标 | 数值 |
|---|---|
| 清前 system df 报告的可回收 | 24.22GB |
| 本次实际回收（prune 自报） | 24.13GB |
| 达到目标比例 | 99.6% |

→ 触发条件**不成立**，故未执行 `--all`。清后复核进一步支持这个决定：

```console
--- docker buildx du（清后） ---
Shared:		25.66GB
Private:	85.8MB
Reclaimable:	25.75GB
Total:		25.75GB

--- docker system df（清后） ---
Build Cache     429       2         25.75GB   85.79MB
```

- 剩余 25.75GB 中，25.66GB 是 **Shared**（与在用镜像共享的层），
  `docker system df` 只把 **85.79MB** 记为可回收 —— 即默认语义下已基本榨干。
- `--all` 会额外移除 BuildKit 的 internal/frontend images；其带来的额外磁盘收益在此状态上界约为
  85.79MB 量级，却把动作从「清缓存」扩到「删内部前端镜像」，与「只清构建缓存、不删镜像」的红线边界贴近。
- 因此按「最小必要动作」停在默认语义；若 Lead 仍要压掉剩余 25.66GB 的**缓存记录**，
  需要明确点头（并接受它主要是删记录、不释放等量磁盘）。

## 4. 清后核验（21:52:16Z）

```console
### 2026-10-23T21:52:16Z POST-CLEANUP verification
--- containers: all 8 must be Up ---
new-api | ghcr.io/zhemed/new-api-own:v0.0.45 | Up 5 hours
komari-agent-test | ghcr.io/zhemed/komari-agent:latest | Up 12 hours
komari-test | ghcr.io/zhemed/komari:latest | Up 10 hours
picsur | ghcr.io/zhemed/picsur:allinone | Up 4 days (healthy)
support-devdb-1 | postgres:17-alpine | Up 4 days
litepan | ghcr.io/zhemed/litepan:v0.0.49 | Up 4 days
halo | ghcr.io/zhemed/halo:v2.26.1-custom.1 | Up 4 days (healthy)
halodb | postgres:15.4 | Up 4 days (healthy)
running_count=8  all_count=8
  逐个 State.Status：komari-agent-test=running / komari-test=running / picsur=running /
  support-devdb-1=running / litepan=running / halo=running / halodb=running / new-api=running

--- new-api health ---
state=running restarts=0 mount=/root/new-api-own/data:/data
HTTP/1.1 200 OK
X-New-Api-Version: v0.0.45

--- images (must be unchanged: 2 tags, same ID) ---
ghcr.io/zhemed/new-api-own:latest | e114b17532f1
ghcr.io/zhemed/new-api-own:v0.0.45 | e114b17532f1
new_api_own_tags=2
dangling_images=0

--- volumes (must be 3, dangling=2) ---
DRIVER    VOLUME NAME
local     halo-gradle-cache
local     picsur_picsur-data
local     support_db-data
volume_total=3  volume_dangling=2

--- data dir intact ---
dir_exists=yes  files=55

--- df -h / (after) ---
/dev/sda2       251G   92G  148G  39% /
/dev/sda2      269425360896 97729261568 157935538176  39% /
```

## 5. 前后对比

| 指标 | 清理前（21:49:46Z） | 清理后（21:52:16Z） | 变化 |
|---|---|---|---|
| df 已用 | 114G | 92G | -22G |
| df 可用 | 125G | 148G | +23G |
| df 已用（精确字节） | 121850494976 | 97729261568 | **-24121233408（22.465 GiB / 24.121 GB）** |
| Build Cache 体积 | 49.88GB（636 条） | 25.75GB（429 条） | -24.13GB / -207 条 |
| Build Cache 可回收 | 24.22GB | 85.79MB | -24.22GB |
| 镜像 | 110 个 / 17.96GB | 110 个 / 17.96GB | 不变 |
| new-api-own 标签 | 2（v0.0.45、latest，同 ID） | 2（同 ID e114b17532f1） | 不变 |
| 容器 | 8 个（全运行） | 8 个（全运行） | 不变 |
| 本地卷 | 3 个（dangling 2） | 3 个（dangling 2） | 不变 |
| `data/` 文件数 | 55 | 55 | 不变 |

验收对照（PRD AC，仅本任务范围）：

| AC | 内容 | 状态 | 证据 |
|---|---|---|---|
| AC1 | Build Cache 清理完成，system df 前后对比与 df 精确字节回收量 | 达标 | 第 1、2、5 节 |
| AC2 | 8 容器全 Up、new-api running/restarts=0/200+v0.0.45、镜像 2 标签同 ID、卷 3 个、data 55 文件 | 达标 | 第 4 节 |
| AC3-AC6 | wrapup 脚本 / 闸门指南 / 端到端测试 / 真实收尾 | 不属于 task-12（PRD R2-R3，由 Lead 侧执行） | PRD `prd.md:23-29` |

## 6. 副作用与风险

- **其它项目下次构建会失去缓存、明显变慢**（需重新拉取/重算构建层），**无任何数据丢失** —— 属预期副作用。
- 剩余 25.75GB 缓存记录仍在（Shared 与在用镜像共享层）；它们不占额外磁盘，
  因为对应层同时被镜像引用；`docker system df` 记为不可回收。
- 本次动作不可逆（缓存需重新构建生成）；对运行中的 8 个容器与实例零影响（第 4 节已核验）。
- 未使用 `docker system prune` 系列命令；镜像/容器/卷的 `TOTAL` 与 `ACTIVE` 计数前后一致，可交叉验证。

## 7. 建议下一步

1. Lead 复核（可抽查：`docker system df` 的 Build Cache 行与 `df -B1 /`）。
2. 若接受「只删缓存记录、不释放等量磁盘」，可再授权 `docker builder prune -a -f` 清剩余 Shared 记录；
   预期额外回收约 85MB 量级，收益低——建议维持现状。
3. 后续磁盘治理方向：镜像可回收 12.47GB（需删旧镜像，属新授权）、本地卷 814.6MB（两个命名卷，
   按上一轮裁定保留）；两者都要新的用户授权。
4. 提交收口由 Lead 负责（锚点 `[task:build-cache-and-wrapup-flow]`）；本文件与 wrapup 流程改动可一并提交。
