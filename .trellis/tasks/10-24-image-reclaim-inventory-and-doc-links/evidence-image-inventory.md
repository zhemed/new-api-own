# 只读盘点：镜像侧可回收量（12.47GB）清单、归属与安全边界

- 任务：Trellis `10-24-image-reclaim-inventory-and-doc-links`；共享任务板 `task-14`；执行者 worker-containers
- 执行时间：2026-10-24T00:29:33Z ～ 00:33Z（UTC），全程只访问本机
- 事前依据：PRD `prd.md:19-29`（R1-R4，只读盘点）；用户明确要求**先盘点、不删除**
- 约束遵守：**未执行任何删除类命令**（无 `docker rmi` / `image prune` / `system prune` / `volume prune`），
  只用 `docker images` / `docker ps -a` / `docker inspect` / `docker system df` / `docker system df -v`；
  未碰 `/root/new-api-own/data`；未提交未推送

## 0. 结论摘要

| 结论 | 数值 |
|---|---|
| 镜像总数（唯一 ID，= `docker system df` 的 Images） | 110 |
| 标签行数（`docker images` 行数，同一镜像多标签会占多行） | 146 |
| 被容器引用的镜像 | 8 个（8 个容器全部在跑） |
| **候选（有标签但无任何容器在用）** | **102 个** |
| 候选 unique 层合计（独占层，估算回收量） | **12.52GB** |
| `docker system df` 报告的 RECLAIMABLE | **12.47GB**（与上面 12.52GB 相差 0.05GB / 0.4%，互相印证） |
| 候选磁盘占用合计（含共享层重复计，仅作上界参考） | 30.72GB |
| **本项目 `ghcr.io/zhemed/new-api-own` 的候选数** | **0**（唯一镜像正在被 `new-api` 使用） |

一句话给用户：**本项目在镜像侧已无可回收量**；12.47GB 全部属于**其它项目**
（komari / komari-agent / picsur / litepan / 本地临时标签与基础镜像），
删除需要用户对相应项目单独点头，且真实回收量只能在删除后实测。

## 1. 口径与方法

1. 唯一镜像清单：`docker images -q | sort -u` → 110 个 ID；逐 ID
   `docker inspect --format '{{.Id}}|{{.Size}}|{{.Created}}|{{.RepoTags}}'` 取标签与创建时间。
2. 引用集合：`docker ps -aq`（含**已停止**容器，本机现在 0 个已停）逐个
   `docker inspect --format '{{.Name}}|{{.Config.Image}}|{{.Image}}|{{.State.Status}}'`，
   取 `.Image`（镜像 ID）构成 8 个被引用 ID。
3. 差集：镜像 ID 不在引用集合中 → 候选；本机 `dangling=true` 为 **0**，所以候选都是
   「**有标签**但无容器」的镜像，不是无标签悬空镜像。
4. 体积口径：`docker inspect` 的 `.Size` 是**压缩后**体积（例如本项目 v0.0.45 只有 72.1MB），
   不等于磁盘占用；因此改用 `docker system df -v` 的 Images 段：
   SIZE（磁盘占用）、SHARED SIZE（与其它镜像共享的层）、UNIQUE SIZE（该镜像独占层）、CONTAINERS（引用计数）。
5. 回收量口径：`docker system df` 的 Build/Images RECLAIMABLE 是 Docker 的**估算**；
   与候选镜像的 UNIQUE 层合计互相印证（12.52GB 对 12.47GB）。

## 2. 只读自证（盘点前后计数完全一致）

```console
### 2026-10-24T00:29:33Z PRE-COUNTS (read-only)
TYPE            TOTAL     ACTIVE    SIZE      RECLAIMABLE
Images          110       8         17.96GB   12.47GB (69%)
Containers      8         8         262.1kB   0B (0%)
Local Volumes   3         1         996.9MB   814.6MB (81%)
Build Cache     429       2         25.75GB   85.79MB
image_tag_rows=146  image_unique_ids=110
containers_all=8 running=8
volumes=3 dangling_volumes=2
dangling_images=0

### POST-COUNTS（盘点结束后重取，逐项与上面一致）
Images          110       8         17.96GB   12.47GB (69%)
Containers      8         8         262.1kB   0B (0%)
Local Volumes   3         1         996.9MB   814.6MB (81%)
Build Cache     429       2         25.75GB   85.79MB
image_tag_rows=146  image_unique_ids=110
containers_all=8 running=8
volumes=3 dangling_volumes=2
dangling_images=0
new-api: running restarts=0 mount=/root/new-api-own/data
http=200 version=v0.0.45
data_files=55
```

本次盘点执行的命令只有：`docker images` / `docker images -q` / `docker ps -a` / `docker inspect`
/ `docker system df` / `docker system df -v`。**没有** `rmi`、`prune`、`rm` 等删除类命令。

## 3. 在用镜像（8 个，任何方案都必须保留）

| 镜像 ID | 仓库:标签 | 磁盘(MB) | 使用容器 |
|---|---|---|---|
| 8b4d729b7727 | ghcr.io/zhemed/halo:v2.26.1-custom.1 | 1040.0 | halo |
| 4dd8bd40598b | ghcr.io/zhemed/picsur:allinone | 799.0 | picsur |
| d1168ad5397b | postgres:15.4 | 590.0 | halodb |
| b0f9560a2de0 | postgres:17-alpine | 424.0 | support-devdb-1 |
| e114b17532f1 | ghcr.io/zhemed/new-api-own:latest + :v0.0.45 | 302.0 | new-api（本项目） |
| 6ddd6e7973da | ghcr.io/zhemed/litepan:0.0.49 + :v0.0.49 | 158.0 | litepan |
| fc0fbf2ce0f6 | ghcr.io/zhemed/komari:0.0.24 + :latest | 143.0 | komari-test |
| cb4b4cec1be6 | ghcr.io/zhemed/komari-agent:0.0.24 + :latest | 90.0 | komari-agent-test |

## 4. 候选清单（102 个）

### 4.1 按仓库分组汇总

| 仓库 | 镜像数 | 标签行数 | 磁盘合计 | unique 合计 | 桶 | 归属判定 |
|---|---|---|---|---|---|---|
| ghcr.io/zhemed/picsur | 18 | 38 | 13.62 GB | 5.64 GB | B | 在用: allinone（该仓库有容器） |
| zhemed/picsur | 4 | 4 | 2.54 GB | 1.39 GB | C | 无容器在用；标签示例: allinone, local, pre-detach |
| ghcr.io/zhemed/komari | 21 | 21 | 1.94 GB | 0.96 GB | B | 在用: 0.0.24, latest（该仓库有容器） |
| 127.0.0.1:5001/picsur-nonroot | 3 | 6 | 1.58 GB | 0.00 GB | C | 无容器在用；标签示例: 0.8.0, 0.8.1, broken |
| v2.26.1-custom.local | 1 | 1 | 1.04 GB | 0.21 GB | C | 无容器在用；标签示例: latest |
| ghcr.io/zhemed/litepan | 6 | 12 | 0.96 GB | 0.32 GB | B | 在用: 0.0.49, v0.0.49（该仓库有容器） |
| halohub/halo | 1 | 1 | 0.83 GB | 0.00 GB | D | 无容器在用；标签示例: 2.26 |
| 127.0.0.1:5001/picsur-root | 1 | 2 | 0.82 GB | 0.62 GB | C | 无容器在用；标签示例: 0.8.0 |
| ghcr.io/zhemed/komari-agent | 20 | 20 | 0.81 GB | 0.15 GB | B | 在用: 0.0.24, latest（该仓库有容器） |
| picsur-oldroot | 1 | 1 | 0.78 GB | 0.40 GB | C | 无容器在用；标签示例: 0.7.9fix |
| 127.0.0.1:5001/picsur-rootreg | 1 | 2 | 0.77 GB | 0.00 GB | C | 无容器在用；标签示例: 0.8.0 |
| picsur-reltest | 1 | 1 | 0.76 GB | 0.56 GB | C | 无容器在用；标签示例: diag |
| eclipse-temurin | 1 | 1 | 0.73 GB | 0.73 GB | D | 无容器在用；标签示例: 21-jdk |
| 127.0.0.1:5001/picsur | 1 | 2 | 0.69 GB | 0.31 GB | C | 无容器在用；标签示例: allinone |
| komari-own | 4 | 4 | 0.66 GB | 0.16 GB | C | 无容器在用；标签示例: 0.0.24-own.1, 0.0.24-own.2, 0.0.24-own.3 |
| postgres | 1 | 1 | 0.63 GB | 0.52 GB | B | 在用: 17-alpine, 15.4（该仓库有容器） |
| redis | 2 | 2 | 0.38 GB | 0.15 GB | D | 无容器在用；标签示例: 7, latest |
| <none> | 3 | 3 | 0.27 GB | 0.09 GB | C | 无容器在用；标签示例: <none> |
| debian | 2 | 2 | 0.23 GB | 0.00 GB | D | 无容器在用；标签示例: bookworm-slim, trixie-slim |
| node | 1 | 1 | 0.19 GB | 0.00 GB | D | 无容器在用；标签示例: 20-alpine |
| komari-local | 1 | 1 | 0.16 GB | 0.05 GB | C | 无容器在用；标签示例: 0.0.24-4d6cc4f |
| tonistiigi/binfmt | 1 | 1 | 0.12 GB | 0.12 GB | D | 无容器在用；标签示例: latest |
| komari-test | 1 | 1 | 0.09 GB | 0.07 GB | C | 无容器在用；标签示例: 0.0.1 |
| alpine | 3 | 3 | 0.05 GB | 0.03 GB | D | 无容器在用；标签示例: 3.21, 3.23, latest |
| registry | 1 | 1 | 0.04 GB | 0.00 GB | D | 无容器在用；标签示例: 2 |
| curlimages/curl | 1 | 1 | 0.04 GB | 0.04 GB | D | 无容器在用；标签示例: latest |
| hello-world | 1 | 1 | 0.00 GB | 0.00 GB | D | 无容器在用；标签示例: latest |

### 4.2 逐条明细（102 行）

列含义：ID = 镜像短 ID；标签 = 该镜像的标签（多标签时给出首个，`...` 表示还有更多）；
磁盘 = `docker system df -v` 的 SIZE；创建 = df -v 的 CREATED；桶 = 第 5 节的建议分桶。

| 镜像 ID | 仓库:标签 | 磁盘(MB) | 创建 | 桶 | 是否被容器引用 |
|---|---|---|---|---|---|
| 82de87364bce | v2.26.1-custom.local:latest | 1040.0 | 3 weeks ago | C | 否（containers=0） |
| bfe89bf70c44 | halohub/halo:2.26 | 831.0 | 7 weeks ago | D | 否（containers=0） |
| 32dee8361017 | 127.0.0.1:5001/picsur-root:0.8.0, picsur-root:0.8.0 | 816.0 | 7 days ago | C | 否（containers=0） |
| 7b823687ad1b | ghcr.io/zhemed/picsur:allinone-test | 803.0 | 7 days ago | B | 否（containers=0） |
| 9404ba22c815 | picsur-oldroot:0.7.9fix | 780.0 | 7 days ago | C | 否（containers=0） |
| 2c6a7775edc5 | 127.0.0.1:5001/picsur-nonroot:0.8.1, picsur-nonroot:0.8.1 | 769.0 | 7 days ago | C | 否（containers=0） |
| 20647bd7fe3e | 127.0.0.1:5001/picsur-rootreg:0.8.0, picsur-rootreg:0.8.0 | 769.0 | 7 days ago | C | 否（containers=0） |
| 244fa53cf794 | 127.0.0.1:5001/picsur-nonroot:0.8.0, picsur-nonroot:0.8.0 | 769.0 | 7 days ago | C | 否（containers=0） |
| f850a06a6a3e | ghcr.io/zhemed/picsur:0.8.9, ghcr.io/zhemed/picsur:0.8 ...(+1) | 758.0 | 7 days ago | B | 否（containers=0） |
| dc9424941141 | ghcr.io/zhemed/picsur:0.8.8, ghcr.io/zhemed/picsur:v0.8.8 | 756.0 | 7 days ago | B | 否（containers=0） |
| 56eafdb9851a | ghcr.io/zhemed/picsur:0.8.7, ghcr.io/zhemed/picsur:v0.8.7 | 756.0 | 7 days ago | B | 否（containers=0） |
| dad2c5c00886 | ghcr.io/zhemed/picsur:0.8.6, ghcr.io/zhemed/picsur:v0.8.6 | 756.0 | 7 days ago | B | 否（containers=0） |
| f81e17bc9226 | ghcr.io/zhemed/picsur:0.8.5, ghcr.io/zhemed/picsur:v0.8.5 | 756.0 | 7 days ago | B | 否（containers=0） |
| 4263c3c5285f | ghcr.io/zhemed/picsur:0.8.4, ghcr.io/zhemed/picsur:v0.8.4 | 756.0 | 7 days ago | B | 否（containers=0） |
| da9d34cf8169 | ghcr.io/zhemed/picsur:0.8.3, ghcr.io/zhemed/picsur:v0.8.3 | 756.0 | 7 days ago | B | 否（containers=0） |
| 5f7ac86cb6a4 | ghcr.io/zhemed/picsur:0.8.2, ghcr.io/zhemed/picsur:v0.8.2 | 756.0 | 7 days ago | B | 否（containers=0） |
| 0c439afe625b | ghcr.io/zhemed/picsur:localtest | 756.0 | 7 days ago | B | 否（containers=0） |
| 3cb25b91c887 | picsur-reltest:diag | 756.0 | 8 days ago | C | 否（containers=0） |
| 248755520402 | ghcr.io/zhemed/picsur:0.10.6, ghcr.io/zhemed/picsur:0.10 ...(+2) | 752.0 | 7 days ago | B | 否（containers=0） |
| e51e209a35c3 | ghcr.io/zhemed/picsur:0.10.5, ghcr.io/zhemed/picsur:v0.10.5 | 752.0 | 7 days ago | B | 否（containers=0） |
| bfee5bf2a402 | ghcr.io/zhemed/picsur:0.10.4, ghcr.io/zhemed/picsur:v0.10.4 | 752.0 | 7 days ago | B | 否（containers=0） |
| 3ff778618ebc | ghcr.io/zhemed/picsur:0.10.3, ghcr.io/zhemed/picsur:v0.10.3 | 752.0 | 7 days ago | B | 否（containers=0） |
| 76886a977484 | ghcr.io/zhemed/picsur:0.10.2, ghcr.io/zhemed/picsur:v0.10.2 | 752.0 | 7 days ago | B | 否（containers=0） |
| 929a04667692 | ghcr.io/zhemed/picsur:0.10.1, ghcr.io/zhemed/picsur:v0.10.1 | 752.0 | 7 days ago | B | 否（containers=0） |
| ede2042d265b | ghcr.io/zhemed/picsur:0.10.0, ghcr.io/zhemed/picsur:v0.10.0 | 752.0 | 7 days ago | B | 否（containers=0） |
| 9fb040253452 | ghcr.io/zhemed/picsur:0.9.0, ghcr.io/zhemed/picsur:0.9 ...(+1) | 752.0 | 7 days ago | B | 否（containers=0） |
| 4d0603880065 | eclipse-temurin:21-jdk | 726.0 | 4 weeks ago | D | 否（containers=0） |
| e6750e6cbe15 | 127.0.0.1:5001/picsur:allinone, picsur-fix:allinone | 695.0 | 8 days ago | C | 否（containers=0） |
| 75939faa28ed | zhemed/picsur:allinone | 684.0 | 10 days ago | C | 否（containers=0） |
| c07aac29d86d | zhemed/picsur:local | 637.0 | 10 days ago | C | 否（containers=0） |
| 9b1d34adbce1 | postgres:15 | 633.0 | 2 months ago | B | 否（containers=0） |
| 3774af509bb8 | zhemed/picsur:pre-detach | 611.0 | 10 days ago | C | 否（containers=0） |
| 13aac61fae2c | zhemed/picsur:pre-t0 | 609.0 | 10 days ago | C | 否（containers=0） |
| 298e5b3bc566 | redis:latest | 212.0 | 2 months ago | D | 否（containers=0） |
| fb4cd12c85ee | node:20-alpine | 194.0 | 6 months ago | D | 否（containers=0） |
| 71da9275c5f3 | redis:7 | 170.0 | 2 months ago | D | 否（containers=0） |
| 63036cd80752 | komari-own:0.0.24-own.3 | 164.0 | 10 hours ago | C | 否（containers=0） |
| 7ffd6e6ecb8f | komari-own:0.0.24-own.2 | 164.0 | 10 hours ago | C | 否（containers=0） |
| c4598a89e5c0 | komari-own:0.0.24-own.1 | 164.0 | 11 hours ago | C | 否（containers=0） |
| 9ce5104ec0ee | komari-local:0.0.24-4d6cc4f | 164.0 | 12 hours ago | C | 否（containers=0） |
| feccf2f40534 | komari-own:0.0.24-own.4 | 163.0 | 10 hours ago | C | 否（containers=0） |
| 236392d85380 | ghcr.io/zhemed/litepan:0.0.45, ghcr.io/zhemed/litepan:v0.0.45 | 161.0 | 5 weeks ago | B | 否（containers=0） |
| a864057d46ac | ghcr.io/zhemed/litepan:v0.0.44 | 161.0 | 5 weeks ago | B | 否（containers=0） |
| f43b75d491f6 | ghcr.io/zhemed/litepan:0.0.48, ghcr.io/zhemed/litepan:v0.0.48 | 159.0 | 4 weeks ago | B | 否（containers=0） |
| 2ae2b2265243 | ghcr.io/zhemed/litepan:0.0.47, ghcr.io/zhemed/litepan:v0.0.47 | 159.0 | 5 weeks ago | B | 否（containers=0） |
| 31b122a678d5 | ghcr.io/zhemed/litepan:0.0.46, ghcr.io/zhemed/litepan:v0.0.46 | 159.0 | 5 weeks ago | B | 否（containers=0） |
| eeda8ce7d49a | ghcr.io/zhemed/litepan:0.0.50, ghcr.io/zhemed/litepan:latest ...(+1) | 158.0 | 7 days ago | B | 否（containers=0） |
| 0db27f562499 | ghcr.io/zhemed/komari:<none> | 145.0 | 18 hours ago | C | 否（containers=0） |
| c10dbc6b2b71 | ghcr.io/zhemed/komari:0.0.21 | 144.0 | 4 weeks ago | B | 否（containers=0） |
| 4817dbd21193 | ghcr.io/zhemed/komari:0.0.23 | 143.0 | 17 hours ago | B | 否（containers=0） |
| daba981889af | ghcr.io/zhemed/komari:0.0.22 | 143.0 | 18 hours ago | B | 否（containers=0） |
| 62bd423aeee9 | debian:trixie-slim | 119.0 | 3 days ago | D | 否（containers=0） |
| 400a4873b838 | tonistiigi/binfmt:latest | 118.0 | 4 months ago | D | 否（containers=0） |
| 7ff6ba31527b | debian:bookworm-slim | 116.0 | 3 days ago | D | 否（containers=0） |
| 03bd8acdcd67 | ghcr.io/zhemed/komari-agent:<none> | 92.5 | 21 hours ago | C | 否（containers=0） |
| 9fe38e74650f | ghcr.io/zhemed/komari:0.0.16 | 91.9 | 5 weeks ago | B | 否（containers=0） |
| 631adfe719d8 | ghcr.io/zhemed/komari:0.0.15 | 91.9 | 5 weeks ago | B | 否（containers=0） |
| b0bf44fae784 | ghcr.io/zhemed/komari:0.0.14 | 91.9 | 5 weeks ago | B | 否（containers=0） |
| 4763431298e4 | ghcr.io/zhemed/komari:0.0.13 | 91.9 | 5 weeks ago | B | 否（containers=0） |
| 4b348bc36fc2 | ghcr.io/zhemed/komari:0.0.18 | 91.8 | 4 weeks ago | B | 否（containers=0） |
| 360e1f33cda4 | ghcr.io/zhemed/komari:0.0.17 | 91.8 | 5 weeks ago | B | 否（containers=0） |
| f181c709a9bc | ghcr.io/zhemed/komari:0.0.12 | 91.8 | 5 weeks ago | B | 否（containers=0） |
| f10f9766af63 | ghcr.io/zhemed/komari:0.0.10 | 91.7 | 5 weeks ago | B | 否（containers=0） |
| 696e109b450b | ghcr.io/zhemed/komari:0.0.9 | 91.7 | 5 weeks ago | B | 否（containers=0） |
| 01ebfeb6cea4 | ghcr.io/zhemed/komari:0.0.20 | 91.4 | 4 weeks ago | B | 否（containers=0） |
| cc395d2af893 | ghcr.io/zhemed/komari:0.0.19 | 91.4 | 4 weeks ago | B | 否（containers=0） |
| 9c52428a52b6 | ghcr.io/zhemed/komari:0.0.7 | 91.3 | 5 weeks ago | B | 否（containers=0） |
| d133494942a6 | ghcr.io/zhemed/komari:0.0.5 | 91.3 | 5 weeks ago | B | 否（containers=0） |
| 391ebbce81e1 | ghcr.io/zhemed/komari:0.0.4 | 91.3 | 5 weeks ago | B | 否（containers=0） |
| d261e841e122 | ghcr.io/zhemed/komari-agent:0.0.23 | 90.0 | 17 hours ago | B | 否（containers=0） |
| 4284c6541d62 | ghcr.io/zhemed/komari-agent:0.0.22 | 90.0 | 21 hours ago | B | 否（containers=0） |
| 78d36db1f073 | komari-test:0.0.1 | 89.1 | 5 weeks ago | C | 否（containers=0） |
| 5200a5d94418 | ghcr.io/zhemed/komari:0.0.18-test | 71.8 | 5 weeks ago | B | 否（containers=0） |
| a4f1c9ae7b6e | ghcr.io/zhemed/komari:0.0.11 | 71.8 | 5 weeks ago | B | 否（containers=0） |
| 49bfb911370c | ghcr.io/zhemed/komari-agent:0.0.4 | 53.4 | 5 weeks ago | B | 否（containers=0） |
| 6b3ba3d70733 | ghcr.io/zhemed/komari-agent:0.0.21 | 53.0 | 4 weeks ago | B | 否（containers=0） |
| 3b19b517f53f | ghcr.io/zhemed/komari-agent:0.0.18 | 52.7 | 5 weeks ago | B | 否（containers=0） |
| 81932147afc8 | ghcr.io/zhemed/komari:0.0.8 | 39.7 | 5 weeks ago | B | 否（containers=0） |
| eaecedc35370 | ghcr.io/zhemed/komari:0.0.6 | 39.5 | 5 weeks ago | B | 否（containers=0） |
| a3d8aaa63ed8 | registry:2 | 37.4 | 3 years ago | D | 否（containers=0） |
| 46faa9a1ae68 | 127.0.0.1:5001/picsur-nonroot:broken, 127.0.0.1:5001/picsur:broken | 37.2 | 3 years ago | C | 否（containers=0） |
| a134550be58a | curlimages/curl:latest | 35.9 | 9 days ago | D | 否（containers=0） |
| 48b0309ca019 | alpine:<none> | 34.0 | 6 months ago | C | 否（containers=0） |
| 38581f91ed32 | ghcr.io/zhemed/komari-agent:0.0.20 | 31.6 | 4 weeks ago | B | 否（containers=0） |
| 58343b957158 | ghcr.io/zhemed/komari-agent:0.0.19 | 31.6 | 4 weeks ago | B | 否（containers=0） |
| 0545390e474b | ghcr.io/zhemed/komari-agent:0.0.17 | 31.6 | 5 weeks ago | B | 否（containers=0） |
| 605cc13505c2 | ghcr.io/zhemed/komari-agent:0.0.16 | 31.6 | 5 weeks ago | B | 否（containers=0） |
| 8f3b9a5aced8 | ghcr.io/zhemed/komari-agent:0.0.15 | 31.6 | 5 weeks ago | B | 否（containers=0） |
| 5ce2f1c0e615 | ghcr.io/zhemed/komari-agent:0.0.14 | 31.6 | 5 weeks ago | B | 否（containers=0） |
| 154d8ed78b8b | ghcr.io/zhemed/komari-agent:0.0.13 | 31.6 | 5 weeks ago | B | 否（containers=0） |
| 39dbe8cbc31c | ghcr.io/zhemed/komari-agent:0.0.12 | 31.6 | 5 weeks ago | B | 否（containers=0） |
| f57f682a3d5c | ghcr.io/zhemed/komari-agent:0.0.11 | 31.6 | 5 weeks ago | B | 否（containers=0） |
| d53811843b03 | ghcr.io/zhemed/komari-agent:0.0.10 | 31.6 | 5 weeks ago | B | 否（containers=0） |
| 97b003554408 | ghcr.io/zhemed/komari-agent:0.0.9 | 31.6 | 5 weeks ago | B | 否（containers=0） |
| 0f9a6687f7a9 | ghcr.io/zhemed/komari-agent:0.0.8 | 31.6 | 5 weeks ago | B | 否（containers=0） |
| 200ff8496b75 | ghcr.io/zhemed/komari-agent:0.0.7 | 31.6 | 5 weeks ago | B | 否（containers=0） |
| 6cca51ed6bf6 | ghcr.io/zhemed/komari-agent:0.0.6 | 31.6 | 5 weeks ago | B | 否（containers=0） |
| 4726fbeb541d | ghcr.io/zhemed/komari-agent:0.0.5 | 31.6 | 5 weeks ago | B | 否（containers=0） |
| ee16d92f96db | alpine:3.21 | 21.8 | 56 years ago | D | 否（containers=0） |
| 85fe1e81d675 | alpine:3.23 | 13.0 | 5 weeks ago | D | 否（containers=0） |
| 294b683cb724 | alpine:latest | 13.0 | 5 weeks ago | D | 否（containers=0） |
| 5e2309035332 | hello-world:latest | 0.0 | 7 months ago | D | 否（containers=0） |

## 5. 建议分桶

| 桶 | 含义 | 镜像数 | 磁盘合计 | unique 合计 | 处置建议 |
|---|---|---|---|---|---|
| A | 本项目 `ghcr.io/zhemed/new-api-own` | 0 | 0 | 0 | 无可删项（唯一镜像在用） |
| B | 在用项目的**旧版本**（其它项目） | 66 | 17.96GB | 7.59GB | 需用户对相应项目单独授权 |
| C | 当前无容器项目的镜像 / 本地一次性测试标签 | 22 | 10.15GB | 3.86GB | 需对相应项目单独授权（多为本地测试产物） |
| D | 通用基础 / 工具镜像 | 14 | 2.61GB | 1.07GB | 建议保留（构建流程复用，且共享层多、实收远低于 unique） |

各桶组成（仓库 × 数量）：

- B 桶：`ghcr.io/zhemed/komari` ×21、`ghcr.io/zhemed/komari-agent` ×20、`ghcr.io/zhemed/picsur` ×18、
  `ghcr.io/zhemed/litepan` ×6、`postgres` ×1（`postgres:15`，容器实际用 15.4）。
- C 桶：`komari-own` ×4、`zhemed/picsur` ×4、`<none>`（digest 引用形态）×3、
  `127.0.0.1:5001/picsur-nonroot` ×3、`komari-local` ×1、`komari-test` ×1、
  `picsur-oldroot` ×1、`picsur-reltest` ×1、`127.0.0.1:5001/picsur` ×1、`127.0.0.1:5001/picsur-root` ×1、
  `127.0.0.1:5001/picsur-rootreg` ×1、`v2.26.1-custom.local` ×1。
- D 桶：`alpine` ×3、`debian` ×2、`redis` ×2、`curlimages/curl` ×1、`eclipse-temurin` ×1、
  `halohub/halo` ×1（上游 2.26，halo 实际跑的是自建 `ghcr.io/zhemed/halo`）、`tonistiigi/binfmt` ×1、
  `node` ×1、`hello-world` ×1、`registry` ×1。

## 6. 归属与判定依据（谁在用 / 为什么没人用）

| 仓库 | 当前有容器在用？ | 在用标签 | 候选说明与判定依据 |
|---|---|---|---|
| ghcr.io/zhemed/new-api-own | 是（本项目） | latest、v0.0.45（同一 ID e114b17532f1） | 无候选：上轮已把标签收敛到 2 个，均指向在用镜像 |
| ghcr.io/zhemed/komari | 是 | 0.0.24、latest（fc0fbf2ce0f6） | 21 个候选全是 0.0.4-0.0.23 历史版本；0.0.23 是当前版本的上一版（回滚点） |
| ghcr.io/zhemed/komari-agent | 是 | 0.0.24、latest（cb4b4cec1be6） | 20 个候选为 0.0.4-0.0.23 历史版本；0.0.23 为上一版（回滚点） |
| ghcr.io/zhemed/picsur | 是 | allinone（4dd8bd40598b） | 18 个候选为 0.8.2-0.10.6 与 `latest`/本地测试标签；注意容器用的是 `allinone`，**`latest` 并未被引用** |
| ghcr.io/zhemed/litepan | 是 | 0.0.49（6ddd6e7973da） | 6 个候选：0.0.44-0.0.48 为旧版；**0.0.50（= `latest`）比在用版本更新，属待部署版本，建议保留** |
| postgres | 是 | 15.4、17-alpine | 候选 `postgres:15`（旧大版本），可能被其它项目 compose 引用 |
| halohub/halo | 否 | —（halo 跑的是 ghcr.io/zhemed/halo:v2.26.1-custom.1） | 上游镜像 `halohub/halo:2.26`，自建镜像的前身/参考 |
| zhemed/picsur | 否 | — | 4 个本地构建标签（allinone/pre-t0/local/pre-detach），项目已改用 ghcr.io 命名空间 |
| 127.0.0.1:5001/… | 否 | — | 4 个本地 registry（含 broken 标签）测试镜像，unique 为 0（层已被其它镜像引用） |
| komari-own / komari-local / komari-test | 否 | — | 4/1/1 个本地构建与测试标签（0.0.24-own.1-4 等） |
| picsur-oldroot / picsur-reltest / v2.26.1-custom.local | 否 | — | 一次性诊断/发版测试标签 |
| 基础镜像（alpine/debian/node/redis/registry/curl/temurin/binfmt/hello-world） | 否 | — | 供构建流程复用；unique 合计仅 1.07GB，且多与其它镜像共享层 |

补充事实：

- `docker images -qf dangling=true` = 0：候选**全部有标签**，不属于「无标签悬空镜像」。
- 3 个 `<none>` 行是 digest 形式引用（`仓库@sha256:...`）在 `docker system df -v` 中的显示形态
  （`ghcr.io/zhemed/komari@sha256:0db27f...`、`ghcr.io/zhemed/komari-agent@sha256:03bd8a...`、
  `alpine@sha256:48b030...`），不是悬空镜像。
- 特殊条目 `ee16d92f96db` = `alpine:3.21`：其 inspect 缺少 `Created`/`Config` 字段
  （时间戳为零值，`docker images` 显示「56 years ago」），已在明细表中单列，不影响差集判断
  （容器引用数为 0）。

## 7. 局限说明（回收量只能估算，真实值需实测）

1. `docker system df` 的 RECLAIMABLE 是 Docker 自己的**估算**；镜像 SIZE 含共享层，
   直接把候选 SIZE 相加会重复计数（本例得 30.72GB，明显高于实际可回收）。
2. 单个镜像删除后实际释放的字节数取决于「有多少共享层因同时删除而变为可释放」，
   因此只能给出区间：下界 ≈ 该镜像 UNIQUE 层，上界 ≈ 其 SIZE；**精确值只能在删除后实测 `df`**。
3. 候选 UNIQUE 合计 12.52GB 与 system df 的 12.47GB 高度一致（差 0.4%），
   说明这批候选基本就是那 12.47GB 的来源，但**逐桶拆分后的回收量仍需实测复核**。
4. 本机 8 个容器全部在运行、且没有已停止容器，因此「被引用集合」稳定；
   若将来有容器停止，其镜像仍算被引用（本盘点已用 `docker ps -a` 覆盖该情形）。

## 8. 建议（给用户决策用）

**可以安全删除的**：本项目范围内 **0 个**。`ghcr.io/zhemed/new-api-own` 只剩一个镜像
（`latest` 与 `v0.0.45` 同 ID）且正被运行中的 `new-api` 使用，删它等于删实例本体，不可行。

**需用户对其它项目单独授权后才可考虑**（合计 unique ≈ 11.45GB）：

1. B 桶（7.59GB unique）：komari / komari-agent / picsur / litepan / postgres 的旧版本历史标签。
   这些项目**当前都有容器在跑**，删旧版本会失去回滚点——建议至少保留各自「当前版本的上一版」
   （komari 0.0.23、komari-agent 0.0.23、picsur 0.10.6/latest、litepan 0.0.50）与 postgres:15。
2. C 桶（3.86GB unique）：已无容器项目的本地构建/测试标签（komari-own、komari-local、komari-test、
   zhemed/picsur、picsur-oldroot、picsur-reltest、v2.26.1-custom.local）与本地 registry 测试镜像
   （127.0.0.1:5001/…，unique 为 0，删了几乎不回收空间）。这些多为一次性产物，删除风险最低，
   但仍属其它项目，需点名授权。

**建议保留**：

1. 第 3 节的 8 个在用镜像（含本项目实例镜像）。
2. D 桶基础/工具镜像（`alpine`/`debian`/`node`/`redis`/`registry`/`curlimages/curl`/`eclipse-temurin`/
   `tonistiigi/binfmt`）：`alpine` 被 `tonistiigi/binfmt` 等复用、`registry` 与本地 5001 registry 工作流相关，
   删除后其它项目构建需重新拉取；且其 unique 合计仅 1.07GB，收益有限。
3. `postgres:15`：虽无容器在用，但 halodb（15.4）与其它项目的 compose 可能引用同系列，
   建议与 halodb 的升级计划一起决定。

## 9. 未完成与下一步

- 本轮**不删除任何镜像**（用户要求先盘点）；删除动作需按第 8 节分桶点名授权后另开任务。
- 回收量精确值只能在删除后实测（第 7 节）；建议届时按桶分批删除、每批删完立即 `df -B1 /` 对照，
  以便把估算与实际对齐。
- 本文件只覆盖 PRD 的 R1-R4（AC1-AC4）；AC5（`MAINTENANCE.md` / 闸门指南文档改动）与本文件同任务目录下的
  `evidence-docs-links.md` 对应，AC6（用 `scripts/trellis-wrapup.sh` 收尾）属 Lead 侧。
- 未提交：本文件留工作区，待 Lead 按闸门收口（锚点 `[task:image-reclaim-inventory-and-doc-links]`）。
