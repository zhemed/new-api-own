# 删除证据：C 桶全部 + B 桶五项目旧镜像（task-16）

- 任务：Trellis `10-24-prune-reclaimable-images`；共享任务板 `task-16`；执行者 worker-containers
- 执行时间：2026-10-24T00:46:00Z ～ 00:48:05Z（UTC），全程只访问本机
- 事前依据：用户点名授权（C 桶全部 + B 桶 komari / komari-agent / picsur / litepan / postgres 旧标签）；
  task-16 描述 + 上轮清单 `archive/2026-10/10-24-image-reclaim-inventory-and-doc-links/evidence-image-inventory.md`
- 红线遵守：**逐条按「仓库:标签」`docker rmi`**，未按镜像 ID 删、未用 `prune`、未用 `-f`、失败不重试；
  未删容器 / 卷 / 网络 / `data/`；未提交未推送

## 0. 结论摘要

| 项目 | 结果 |
|---|---|
| 待删清单 | **84 个镜像 / 111 个标签**（C 桶 22 个镜像 + B 桶五项目 62 个镜像） |
| 执行结果 | **111/111 条 rc=0，失败 0，跳过 0** |
| 唯一镜像数 | 110 → **26** |
| 标签行数 | 146 → **35** |
| `df -B1 /` 已用 | 97761890304 → 88328695808，**实测回收 9433194496 B ≈ 8.79 GiB / 9.43 GB** |
| `docker system df` Images | 110 个 / 17.96GB（可回收 12.47GB）→ **26 个 / 5.276GB（可回收 1.754GB）** |
| 服务影响 | **零影响**：8 容器全 running、new-api 200 + v0.0.45、卷 3 个、`data/` 55 文件 |
| 保留清单 | 全部命中保留（18 项逐条核验通过），无一误删 |

## 1. 授权范围与保留名单

授权范围（用户点名）：

- **C 桶全部**：上轮盘点中「当前无容器项目的镜像 + 本地一次性测试标签」共 22 个镜像。
- **B 桶五项目**：`ghcr.io/zhemed/komari`、`ghcr.io/zhemed/komari-agent`、`ghcr.io/zhemed/picsur`、
  `ghcr.io/zhemed/litepan`、`postgres` —— 仅删保留名单之外的旧标签。
- **D 桶（基础/工具镜像）不动**。

保留名单解析结果（动态 + 点名）：

```console
### 动态在用镜像（8 个容器，逐个 docker inspect 现取）###
   IN-USE 4dd8bd40598b <- picsur (ghcr.io/zhemed/picsur:allinone)
   IN-USE 6ddd6e7973da <- litepan (ghcr.io/zhemed/litepan:v0.0.49)
   IN-USE 8b4d729b7727 <- halo (ghcr.io/zhemed/halo:v2.26.1-custom.1)
   IN-USE b0f9560a2de0 <- support-devdb-1 (postgres:17-alpine)
   IN-USE cb4b4cec1be6 <- komari-agent-test (ghcr.io/zhemed/komari-agent:latest)
   IN-USE d1168ad5397b <- halodb (postgres:15.4)
   IN-USE e114b17532f1 <- new-api (ghcr.io/zhemed/new-api-own:v0.0.45)
   IN-USE fc0fbf2ce0f6 <- komari-test (ghcr.io/zhemed/komari:latest)

### 点名保留标签 → 镜像 ID ###
  ghcr.io/zhemed/new-api-own:v0.0.45 / :latest      -> e114b17532f1
  ghcr.io/zhemed/komari:0.0.24 / :latest            -> fc0fbf2ce0f6
  ghcr.io/zhemed/komari:0.0.23                      -> 4817dbd21193
  ghcr.io/zhemed/komari-agent:0.0.24 / :latest      -> cb4b4cec1be6
  ghcr.io/zhemed/komari-agent:0.0.23                -> d261e841e122
  ghcr.io/zhemed/picsur:allinone                    -> 4dd8bd40598b
  ghcr.io/zhemed/picsur:latest                      -> 248755520402
  ghcr.io/zhemed/litepan:0.0.49 / :v0.0.49          -> 6ddd6e7973da
  ghcr.io/zhemed/litepan:0.0.50 / :v0.0.50 / :latest-> eeda8ce7d49a
  postgres:15.4                                     -> d1168ad5397b
  postgres:17-alpine                                -> b0f9560a2de0
  debian:bookworm-slim                              -> 7ff6ba31527b   （Dockerfile FROM）
```

Dockerfile FROM 基线镜像（读取仓库根 `Dockerfile` 的 FROM 行）：

| FROM 行 | 本机是否命中 | 处置 |
|---|---|---|
| `oven/bun:1@sha256:0733e503...` | 未命中（不在本机） | 无需处置 |
| `golang:1.25.1-alpine` | 未命中（不在本机） | 无需处置 |
| `debian:bookworm-slim@sha256:f0653765...` | **命中** `7ff6ba31527b` | 已列入保留，未删（属 D 桶，本就不动） |

注：仓库只有根 `Dockerfile` 一个（`relaykit/Dockerfile` 不存在）。

## 2. 待删清单生成过程（含一次误判的发现与修正）

**执行前发现并修正了一处严重误判**（保留审计痕迹，不做静默处理）：

- 第一版脚本用 `镜像引用[7:12]` 解析在用镜像 ID，**少截了 7 位**（实际应为 12 位十六进制），
  导致「在用集合」全部不匹配；`ghcr.io/zhemed/halo:v2.26.1-custom.1`（**正在被 halo 容器使用**）
  因此被误归入 C 桶并出现在待删清单里（该镜像不在点名保留标签列表内，缺少第二道保护）。
- 修正为 `sha256:` 后取 12 位十六进制后，在用集合 8 个全部正确匹配，halo 回到保留侧。
- 同时把范围判定改为**防御性默认**：只有落在 C 桶仓库名单或 B 桶五项目内的镜像才可删，
  其余（A/D/未知仓库）一律视为**不在授权范围**并保留 —— 这正是 halo 之前被误判的根因（未知仓库被默认当成 C）。
- 修正后复核：待删 **84 个镜像 / 111 个标签**；与点名保留标签冲突 **0**；与在用集合冲突 **0**。

在用的 8 个镜像与点名保留标签在生成阶段就已被排除，最终清单中不含它们。

## 3. 待删清单（84 个镜像 / 111 个标签）

按桶：C 桶 22 个镜像、B 桶五项目 62 个镜像（komari 20、komari-agent 19、picsur 17、litepan 5、postgres 1）。

| 镜像 ID | 桶 | 仓库 | 标签 |
|---|---|---|---|
| e6750e6cbe15 | C | 127.0.0.1:5001/picsur | 127.0.0.1:5001/picsur:allinone, picsur-fix:allinone |
| 244fa53cf794 | C | 127.0.0.1:5001/picsur-nonroot | 127.0.0.1:5001/picsur-nonroot:0.8.0, picsur-nonroot:0.8.0 |
| 2c6a7775edc5 | C | 127.0.0.1:5001/picsur-nonroot | 127.0.0.1:5001/picsur-nonroot:0.8.1, picsur-nonroot:0.8.1 |
| 46faa9a1ae68 | C | 127.0.0.1:5001/picsur-nonroot | 127.0.0.1:5001/picsur-nonroot:broken, 127.0.0.1:5001/picsur:broken |
| 32dee8361017 | C | 127.0.0.1:5001/picsur-root | 127.0.0.1:5001/picsur-root:0.8.0, picsur-root:0.8.0 |
| 20647bd7fe3e | C | 127.0.0.1:5001/picsur-rootreg | 127.0.0.1:5001/picsur-rootreg:0.8.0, picsur-rootreg:0.8.0 |
| 03bd8acdcd67 | C | ghcr.io/zhemed/komari-agent | ghcr.io/zhemed/komari-agent@sha256:03bd8acdcd677ca66c1730df2b8d5a2d60891bfec0ca774d140667969c5eb14c |
| 0db27f562499 | C | ghcr.io/zhemed/komari | ghcr.io/zhemed/komari@sha256:0db27f5624994c160160b82530f960b5043724643d6916448023e6d403c71123 |
| 48b0309ca019 | C | alpine | alpine@sha256:48b0309ca019d89d40f670aa1bc06e426dc0931948452e8491e3d65087abc07d |
| 01ebfeb6cea4 | B | ghcr.io/zhemed/komari | ghcr.io/zhemed/komari:0.0.20 |
| 360e1f33cda4 | B | ghcr.io/zhemed/komari | ghcr.io/zhemed/komari:0.0.17 |
| 391ebbce81e1 | B | ghcr.io/zhemed/komari | ghcr.io/zhemed/komari:0.0.4 |
| 4763431298e4 | B | ghcr.io/zhemed/komari | ghcr.io/zhemed/komari:0.0.13 |
| 4b348bc36fc2 | B | ghcr.io/zhemed/komari | ghcr.io/zhemed/komari:0.0.18 |
| 5200a5d94418 | B | ghcr.io/zhemed/komari | ghcr.io/zhemed/komari:0.0.18-test |
| 631adfe719d8 | B | ghcr.io/zhemed/komari | ghcr.io/zhemed/komari:0.0.15 |
| 696e109b450b | B | ghcr.io/zhemed/komari | ghcr.io/zhemed/komari:0.0.9 |
| 81932147afc8 | B | ghcr.io/zhemed/komari | ghcr.io/zhemed/komari:0.0.8 |
| 9c52428a52b6 | B | ghcr.io/zhemed/komari | ghcr.io/zhemed/komari:0.0.7 |
| 9fe38e74650f | B | ghcr.io/zhemed/komari | ghcr.io/zhemed/komari:0.0.16 |
| a4f1c9ae7b6e | B | ghcr.io/zhemed/komari | ghcr.io/zhemed/komari:0.0.11 |
| b0bf44fae784 | B | ghcr.io/zhemed/komari | ghcr.io/zhemed/komari:0.0.14 |
| c10dbc6b2b71 | B | ghcr.io/zhemed/komari | ghcr.io/zhemed/komari:0.0.21 |
| cc395d2af893 | B | ghcr.io/zhemed/komari | ghcr.io/zhemed/komari:0.0.19 |
| d133494942a6 | B | ghcr.io/zhemed/komari | ghcr.io/zhemed/komari:0.0.5 |
| daba981889af | B | ghcr.io/zhemed/komari | ghcr.io/zhemed/komari:0.0.22 |
| eaecedc35370 | B | ghcr.io/zhemed/komari | ghcr.io/zhemed/komari:0.0.6 |
| f10f9766af63 | B | ghcr.io/zhemed/komari | ghcr.io/zhemed/komari:0.0.10 |
| f181c709a9bc | B | ghcr.io/zhemed/komari | ghcr.io/zhemed/komari:0.0.12 |
| 0545390e474b | B | ghcr.io/zhemed/komari-agent | ghcr.io/zhemed/komari-agent:0.0.17 |
| 0f9a6687f7a9 | B | ghcr.io/zhemed/komari-agent | ghcr.io/zhemed/komari-agent:0.0.8 |
| 154d8ed78b8b | B | ghcr.io/zhemed/komari-agent | ghcr.io/zhemed/komari-agent:0.0.13 |
| 200ff8496b75 | B | ghcr.io/zhemed/komari-agent | ghcr.io/zhemed/komari-agent:0.0.7 |
| 38581f91ed32 | B | ghcr.io/zhemed/komari-agent | ghcr.io/zhemed/komari-agent:0.0.20 |
| 39dbe8cbc31c | B | ghcr.io/zhemed/komari-agent | ghcr.io/zhemed/komari-agent:0.0.12 |
| 3b19b517f53f | B | ghcr.io/zhemed/komari-agent | ghcr.io/zhemed/komari-agent:0.0.18 |
| 4284c6541d62 | B | ghcr.io/zhemed/komari-agent | ghcr.io/zhemed/komari-agent:0.0.22 |
| 4726fbeb541d | B | ghcr.io/zhemed/komari-agent | ghcr.io/zhemed/komari-agent:0.0.5 |
| 49bfb911370c | B | ghcr.io/zhemed/komari-agent | ghcr.io/zhemed/komari-agent:0.0.4 |
| 58343b957158 | B | ghcr.io/zhemed/komari-agent | ghcr.io/zhemed/komari-agent:0.0.19 |
| 5ce2f1c0e615 | B | ghcr.io/zhemed/komari-agent | ghcr.io/zhemed/komari-agent:0.0.14 |
| 605cc13505c2 | B | ghcr.io/zhemed/komari-agent | ghcr.io/zhemed/komari-agent:0.0.16 |
| 6b3ba3d70733 | B | ghcr.io/zhemed/komari-agent | ghcr.io/zhemed/komari-agent:0.0.21 |
| 6cca51ed6bf6 | B | ghcr.io/zhemed/komari-agent | ghcr.io/zhemed/komari-agent:0.0.6 |
| 8f3b9a5aced8 | B | ghcr.io/zhemed/komari-agent | ghcr.io/zhemed/komari-agent:0.0.15 |
| 97b003554408 | B | ghcr.io/zhemed/komari-agent | ghcr.io/zhemed/komari-agent:0.0.9 |
| d53811843b03 | B | ghcr.io/zhemed/komari-agent | ghcr.io/zhemed/komari-agent:0.0.10 |
| f57f682a3d5c | B | ghcr.io/zhemed/komari-agent | ghcr.io/zhemed/komari-agent:0.0.11 |
| 236392d85380 | B | ghcr.io/zhemed/litepan | ghcr.io/zhemed/litepan:0.0.45, ghcr.io/zhemed/litepan:v0.0.45 |
| 2ae2b2265243 | B | ghcr.io/zhemed/litepan | ghcr.io/zhemed/litepan:0.0.47, ghcr.io/zhemed/litepan:v0.0.47 |
| 31b122a678d5 | B | ghcr.io/zhemed/litepan | ghcr.io/zhemed/litepan:0.0.46, ghcr.io/zhemed/litepan:v0.0.46 |
| a864057d46ac | B | ghcr.io/zhemed/litepan | ghcr.io/zhemed/litepan:v0.0.44 |
| f43b75d491f6 | B | ghcr.io/zhemed/litepan | ghcr.io/zhemed/litepan:0.0.48, ghcr.io/zhemed/litepan:v0.0.48 |
| 0c439afe625b | B | ghcr.io/zhemed/picsur | ghcr.io/zhemed/picsur:localtest |
| 3ff778618ebc | B | ghcr.io/zhemed/picsur | ghcr.io/zhemed/picsur:0.10.3, ghcr.io/zhemed/picsur:v0.10.3 |
| 4263c3c5285f | B | ghcr.io/zhemed/picsur | ghcr.io/zhemed/picsur:0.8.4, ghcr.io/zhemed/picsur:v0.8.4 |
| 56eafdb9851a | B | ghcr.io/zhemed/picsur | ghcr.io/zhemed/picsur:0.8.7, ghcr.io/zhemed/picsur:v0.8.7 |
| 5f7ac86cb6a4 | B | ghcr.io/zhemed/picsur | ghcr.io/zhemed/picsur:0.8.2, ghcr.io/zhemed/picsur:v0.8.2 |
| 76886a977484 | B | ghcr.io/zhemed/picsur | ghcr.io/zhemed/picsur:0.10.2, ghcr.io/zhemed/picsur:v0.10.2 |
| 7b823687ad1b | B | ghcr.io/zhemed/picsur | ghcr.io/zhemed/picsur:allinone-test |
| 929a04667692 | B | ghcr.io/zhemed/picsur | ghcr.io/zhemed/picsur:0.10.1, ghcr.io/zhemed/picsur:v0.10.1 |
| 9fb040253452 | B | ghcr.io/zhemed/picsur | ghcr.io/zhemed/picsur:0.9, ghcr.io/zhemed/picsur:0.9.0, ghcr.io/zhemed/picsur:v0.9.0 |
| bfee5bf2a402 | B | ghcr.io/zhemed/picsur | ghcr.io/zhemed/picsur:0.10.4, ghcr.io/zhemed/picsur:v0.10.4 |
| da9d34cf8169 | B | ghcr.io/zhemed/picsur | ghcr.io/zhemed/picsur:0.8.3, ghcr.io/zhemed/picsur:v0.8.3 |
| dad2c5c00886 | B | ghcr.io/zhemed/picsur | ghcr.io/zhemed/picsur:0.8.6, ghcr.io/zhemed/picsur:v0.8.6 |
| dc9424941141 | B | ghcr.io/zhemed/picsur | ghcr.io/zhemed/picsur:0.8.8, ghcr.io/zhemed/picsur:v0.8.8 |
| e51e209a35c3 | B | ghcr.io/zhemed/picsur | ghcr.io/zhemed/picsur:0.10.5, ghcr.io/zhemed/picsur:v0.10.5 |
| ede2042d265b | B | ghcr.io/zhemed/picsur | ghcr.io/zhemed/picsur:0.10.0, ghcr.io/zhemed/picsur:v0.10.0 |
| f81e17bc9226 | B | ghcr.io/zhemed/picsur | ghcr.io/zhemed/picsur:0.8.5, ghcr.io/zhemed/picsur:v0.8.5 |
| f850a06a6a3e | B | ghcr.io/zhemed/picsur | ghcr.io/zhemed/picsur:0.8, ghcr.io/zhemed/picsur:0.8.9, ghcr.io/zhemed/picsur:v0.8.9 |
| 9ce5104ec0ee | C | komari-local | komari-local:0.0.24-4d6cc4f |
| 63036cd80752 | C | komari-own | komari-own:0.0.24-own.3 |
| 7ffd6e6ecb8f | C | komari-own | komari-own:0.0.24-own.2 |
| c4598a89e5c0 | C | komari-own | komari-own:0.0.24-own.1 |
| feccf2f40534 | C | komari-own | komari-own:0.0.24-own.4 |
| 78d36db1f073 | C | komari-test | komari-test:0.0.1 |
| 9404ba22c815 | C | picsur-oldroot | picsur-oldroot:0.7.9fix |
| 3cb25b91c887 | C | picsur-reltest | picsur-reltest:diag |
| 9b1d34adbce1 | B | postgres | postgres:15 |
| 82de87364bce | C | v2.26.1-custom.local | v2.26.1-custom.local:latest |
| 13aac61fae2c | C | zhemed/picsur | zhemed/picsur:pre-t0 |
| 3774af509bb8 | C | zhemed/picsur | zhemed/picsur:pre-detach |
| 75939faa28ed | C | zhemed/picsur | zhemed/picsur:allinone |
| c07aac29d86d | C | zhemed/picsur | zhemed/picsur:local |

## 4. 删除执行（逐条返回码）

执行方式：按 `delete-refs.txt` 逐行读取「仓库:标签」引用，对每条执行 `docker rmi 引用`，每条前先
`docker inspect` 解析其镜像 ID 并与保留集合比对（命中即 SKIP 并记录）；**未使用** `-f`、**未使用** `prune`、
失败不重试。完整原始输出（含每条 rmi 的 stdout）留在 `/tmp/prune-plan/delete-run.log`。

```console
TOTAL=111  ok=111  failed=0  skipped=0
```

| # | rc | 引用 |
|---|---|---|
| 1 | 0 | 127.0.0.1:5001/picsur:allinone |
| 2 | 0 | picsur-fix:allinone |
| 3 | 0 | 127.0.0.1:5001/picsur-nonroot:0.8.0 |
| 4 | 0 | picsur-nonroot:0.8.0 |
| 5 | 0 | 127.0.0.1:5001/picsur-nonroot:0.8.1 |
| 6 | 0 | picsur-nonroot:0.8.1 |
| 7 | 0 | 127.0.0.1:5001/picsur-nonroot:broken |
| 8 | 0 | 127.0.0.1:5001/picsur:broken |
| 9 | 0 | 127.0.0.1:5001/picsur-root:0.8.0 |
| 10 | 0 | picsur-root:0.8.0 |
| 11 | 0 | 127.0.0.1:5001/picsur-rootreg:0.8.0 |
| 12 | 0 | picsur-rootreg:0.8.0 |
| 13 | 0 | ghcr.io/zhemed/komari-agent@sha256:03bd8acdcd677ca66c1730df2b8d5a2d60891bfec0ca774d140667969c5eb14c |
| 14 | 0 | ghcr.io/zhemed/komari@sha256:0db27f5624994c160160b82530f960b5043724643d6916448023e6d403c71123 |
| 15 | 0 | alpine@sha256:48b0309ca019d89d40f670aa1bc06e426dc0931948452e8491e3d65087abc07d |
| 16 | 0 | ghcr.io/zhemed/komari:0.0.20 |
| 17 | 0 | ghcr.io/zhemed/komari:0.0.17 |
| 18 | 0 | ghcr.io/zhemed/komari:0.0.4 |
| 19 | 0 | ghcr.io/zhemed/komari:0.0.13 |
| 20 | 0 | ghcr.io/zhemed/komari:0.0.18 |
| 21 | 0 | ghcr.io/zhemed/komari:0.0.18-test |
| 22 | 0 | ghcr.io/zhemed/komari:0.0.15 |
| 23 | 0 | ghcr.io/zhemed/komari:0.0.9 |
| 24 | 0 | ghcr.io/zhemed/komari:0.0.8 |
| 25 | 0 | ghcr.io/zhemed/komari:0.0.7 |
| 26 | 0 | ghcr.io/zhemed/komari:0.0.16 |
| 27 | 0 | ghcr.io/zhemed/komari:0.0.11 |
| 28 | 0 | ghcr.io/zhemed/komari:0.0.14 |
| 29 | 0 | ghcr.io/zhemed/komari:0.0.21 |
| 30 | 0 | ghcr.io/zhemed/komari:0.0.19 |
| 31 | 0 | ghcr.io/zhemed/komari:0.0.5 |
| 32 | 0 | ghcr.io/zhemed/komari:0.0.22 |
| 33 | 0 | ghcr.io/zhemed/komari:0.0.6 |
| 34 | 0 | ghcr.io/zhemed/komari:0.0.10 |
| 35 | 0 | ghcr.io/zhemed/komari:0.0.12 |
| 36 | 0 | ghcr.io/zhemed/komari-agent:0.0.17 |
| 37 | 0 | ghcr.io/zhemed/komari-agent:0.0.8 |
| 38 | 0 | ghcr.io/zhemed/komari-agent:0.0.13 |
| 39 | 0 | ghcr.io/zhemed/komari-agent:0.0.7 |
| 40 | 0 | ghcr.io/zhemed/komari-agent:0.0.20 |
| 41 | 0 | ghcr.io/zhemed/komari-agent:0.0.12 |
| 42 | 0 | ghcr.io/zhemed/komari-agent:0.0.18 |
| 43 | 0 | ghcr.io/zhemed/komari-agent:0.0.22 |
| 44 | 0 | ghcr.io/zhemed/komari-agent:0.0.5 |
| 45 | 0 | ghcr.io/zhemed/komari-agent:0.0.4 |
| 46 | 0 | ghcr.io/zhemed/komari-agent:0.0.19 |
| 47 | 0 | ghcr.io/zhemed/komari-agent:0.0.14 |
| 48 | 0 | ghcr.io/zhemed/komari-agent:0.0.16 |
| 49 | 0 | ghcr.io/zhemed/komari-agent:0.0.21 |
| 50 | 0 | ghcr.io/zhemed/komari-agent:0.0.6 |
| 51 | 0 | ghcr.io/zhemed/komari-agent:0.0.15 |
| 52 | 0 | ghcr.io/zhemed/komari-agent:0.0.9 |
| 53 | 0 | ghcr.io/zhemed/komari-agent:0.0.10 |
| 54 | 0 | ghcr.io/zhemed/komari-agent:0.0.11 |
| 55 | 0 | ghcr.io/zhemed/litepan:0.0.45 |
| 56 | 0 | ghcr.io/zhemed/litepan:v0.0.45 |
| 57 | 0 | ghcr.io/zhemed/litepan:0.0.47 |
| 58 | 0 | ghcr.io/zhemed/litepan:v0.0.47 |
| 59 | 0 | ghcr.io/zhemed/litepan:0.0.46 |
| 60 | 0 | ghcr.io/zhemed/litepan:v0.0.46 |
| 61 | 0 | ghcr.io/zhemed/litepan:v0.0.44 |
| 62 | 0 | ghcr.io/zhemed/litepan:0.0.48 |
| 63 | 0 | ghcr.io/zhemed/litepan:v0.0.48 |
| 64 | 0 | ghcr.io/zhemed/picsur:localtest |
| 65 | 0 | ghcr.io/zhemed/picsur:0.10.3 |
| 66 | 0 | ghcr.io/zhemed/picsur:v0.10.3 |
| 67 | 0 | ghcr.io/zhemed/picsur:0.8.4 |
| 68 | 0 | ghcr.io/zhemed/picsur:v0.8.4 |
| 69 | 0 | ghcr.io/zhemed/picsur:0.8.7 |
| 70 | 0 | ghcr.io/zhemed/picsur:v0.8.7 |
| 71 | 0 | ghcr.io/zhemed/picsur:0.8.2 |
| 72 | 0 | ghcr.io/zhemed/picsur:v0.8.2 |
| 73 | 0 | ghcr.io/zhemed/picsur:0.10.2 |
| 74 | 0 | ghcr.io/zhemed/picsur:v0.10.2 |
| 75 | 0 | ghcr.io/zhemed/picsur:allinone-test |
| 76 | 0 | ghcr.io/zhemed/picsur:0.10.1 |
| 77 | 0 | ghcr.io/zhemed/picsur:v0.10.1 |
| 78 | 0 | ghcr.io/zhemed/picsur:0.9 |
| 79 | 0 | ghcr.io/zhemed/picsur:0.9.0 |
| 80 | 0 | ghcr.io/zhemed/picsur:v0.9.0 |
| 81 | 0 | ghcr.io/zhemed/picsur:0.10.4 |
| 82 | 0 | ghcr.io/zhemed/picsur:v0.10.4 |
| 83 | 0 | ghcr.io/zhemed/picsur:0.8.3 |
| 84 | 0 | ghcr.io/zhemed/picsur:v0.8.3 |
| 85 | 0 | ghcr.io/zhemed/picsur:0.8.6 |
| 86 | 0 | ghcr.io/zhemed/picsur:v0.8.6 |
| 87 | 0 | ghcr.io/zhemed/picsur:0.8.8 |
| 88 | 0 | ghcr.io/zhemed/picsur:v0.8.8 |
| 89 | 0 | ghcr.io/zhemed/picsur:0.10.5 |
| 90 | 0 | ghcr.io/zhemed/picsur:v0.10.5 |
| 91 | 0 | ghcr.io/zhemed/picsur:0.10.0 |
| 92 | 0 | ghcr.io/zhemed/picsur:v0.10.0 |
| 93 | 0 | ghcr.io/zhemed/picsur:0.8.5 |
| 94 | 0 | ghcr.io/zhemed/picsur:v0.8.5 |
| 95 | 0 | ghcr.io/zhemed/picsur:0.8 |
| 96 | 0 | ghcr.io/zhemed/picsur:0.8.9 |
| 97 | 0 | ghcr.io/zhemed/picsur:v0.8.9 |
| 98 | 0 | komari-local:0.0.24-4d6cc4f |
| 99 | 0 | komari-own:0.0.24-own.3 |
| 100 | 0 | komari-own:0.0.24-own.2 |
| 101 | 0 | komari-own:0.0.24-own.1 |
| 102 | 0 | komari-own:0.0.24-own.4 |
| 103 | 0 | komari-test:0.0.1 |
| 104 | 0 | picsur-oldroot:0.7.9fix |
| 105 | 0 | picsur-reltest:diag |
| 106 | 0 | postgres:15 |
| 107 | 0 | v2.26.1-custom.local:latest |
| 108 | 0 | zhemed/picsur:pre-t0 |
| 109 | 0 | zhemed/picsur:pre-detach |
| 110 | 0 | zhemed/picsur:allinone |
| 111 | 0 | zhemed/picsur:local |

## 5. 删后核验（00:48:05Z）

```console
### 2026-10-24T00:48:05Z POST-DELETE verification
--- df -B1 / (after) ---
/dev/sda2      269425360896 88328695808 167336103936  35% /
--- df -h / (after) ---
/dev/sda2       251G   83G  156G  35% /
--- docker system df (after) ---
TYPE            TOTAL     ACTIVE    SIZE      RECLAIMABLE
Images          26        8         5.276GB   1.754GB (33%)
Containers      8         8         262.1kB   0B (0%)
Local Volumes   3         1         996.9MB   814.6MB (81%)
Build Cache     429       2         25.75GB   13.26GB
--- image counts (after) ---
unique=26  tag_rows=35  dangling=0
--- containers (all must be Up) ---
new-api | ghcr.io/zhemed/new-api-own:v0.0.45 | Up 8 hours
komari-agent-test | ghcr.io/zhemed/komari-agent:latest | Up 15 hours
komari-test | ghcr.io/zhemed/komari:latest | Up 13 hours
picsur | ghcr.io/zhemed/picsur:allinone | Up 4 days (healthy)
support-devdb-1 | postgres:17-alpine | Up 4 days
litepan | ghcr.io/zhemed/litepan:v0.0.49 | Up 4 days
halo | ghcr.io/zhemed/halo:v2.26.1-custom.1 | Up 4 days (healthy)
halodb | postgres:15.4 | Up 4 days (healthy)
running=8  all=8
  逐个 State.Status：komari-agent-test=running / komari-test=running / picsur=running /
  support-devdb-1=running / litepan=running / halo=running / halodb=running / new-api=running
--- new-api ---
running restarts=0 mount=/root/new-api-own/data:/data
HTTP/1.1 200 OK
X-New-Api-Version: v0.0.45
--- volumes ---
local  halo-gradle-cache / picsur_picsur-data / support_db-data   (total=3 dangling=2)
--- data dir ---
exists=yes  files=55
```

保留清单 18 项逐条复核（`docker images -q <tag>`，全部命中）：

```console
  KEEP OK ghcr.io/zhemed/new-api-own:v0.0.45     e114b17532f1
  KEEP OK ghcr.io/zhemed/new-api-own:latest      e114b17532f1
  KEEP OK ghcr.io/zhemed/komari:0.0.24           fc0fbf2ce0f6
  KEEP OK ghcr.io/zhemed/komari:0.0.23           4817dbd21193
  KEEP OK ghcr.io/zhemed/komari:latest           fc0fbf2ce0f6
  KEEP OK ghcr.io/zhemed/komari-agent:0.0.24     cb4b4cec1be6
  KEEP OK ghcr.io/zhemed/komari-agent:0.0.23     d261e841e122
  KEEP OK ghcr.io/zhemed/komari-agent:latest     cb4b4cec1be6
  KEEP OK ghcr.io/zhemed/picsur:allinone         4dd8bd40598b
  KEEP OK ghcr.io/zhemed/picsur:latest           248755520402
  KEEP OK ghcr.io/zhemed/litepan:0.0.49          6ddd6e7973da
  KEEP OK ghcr.io/zhemed/litepan:v0.0.49         6ddd6e7973da
  KEEP OK ghcr.io/zhemed/litepan:0.0.50          eeda8ce7d49a
  KEEP OK ghcr.io/zhemed/litepan:latest          eeda8ce7d49a
  KEEP OK postgres:15.4                          d1168ad5397b
  KEEP OK postgres:17-alpine                     b0f9560a2de0
  KEEP OK debian:bookworm-slim                   7ff6ba31527b
  KEEP OK ghcr.io/zhemed/halo:v2.26.1-custom.1   8b4d729b7727
```

## 6. 实测回收量 vs 估算（差额归因）

| 指标 | 数值 |
|---|---|
| 估算（上轮清单：B 桶 unique 7.59GB + C 桶 unique 3.86GB） | 约 11.45GB |
| **实测（`df -B1 /` 前后差）** | **9433194496 B = 8.79 GiB = 9.43 GB** |
| `docker system df` 口径的镜像体积下降 | 17.96GB → 5.276GB（-12.68GB） |
| 差额与原因 | 实测比估算少约 2.0GB：① 被删镜像之间、以及与保留下来的 26 个镜像之间**仍有共享层**，这些层不会随删除释放；② 部分层被 **Build Cache** 记录继续引用，因而留在磁盘上（见下）；③ `system df` 的估算口径与文件系统实际释放不对等 |

**重要副产物（本轮未处理，需新授权）**：Build Cache 的「可回收」从 **85.79MB 跳到 13.26GB**
（总量仍是 25.75GB / 429 条）——因为原先与镜像共享的缓存记录，在镜像删除后变成可回收。
也就是说，本轮没释放的那约 2GB 加上原本共享的部分，现在压在 Build Cache 上；
若后续再跑一次 `docker builder prune -f`（task-12 曾按裁定保留），预计还能释放约 13GB。

## 7. 保留清单最终状态（为什么还在）

| 保留对象 | 最终状态 | 原因 |
|---|---|---|
| 8 个运行中容器引用的镜像 | 全部在用（第 5 节逐容器核验） | 硬约束 #1：删了会直接打断服务 |
| `new-api-own:v0.0.45` / `:latest` | 同一 ID `e114b17532f1` 在用 | 硬约束 #2（本项目实例本体） |
| komari `0.0.24` + `0.0.23` | 均在（`fc0fbf2ce0f6` / `4817dbd21193`） | 硬约束 #3：在用版 + 上一版回滚点 |
| komari-agent `0.0.24` + `0.0.23` | 均在（`cb4b4cec1be6` / `d261e841e122`） | 硬约束 #4：同上 |
| picsur `allinone` + `latest` | 均在（`4dd8bd40598b` / `248755520402`） | 硬约束 #5：在用 + 回滚标签；`latest` 同镜像还带 `0.10`/`0.10.6`/`v0.10.6`，一并保留（同镜像删了不回收空间） |
| litepan `0.0.49` + `0.0.50` | 均在（`6ddd6e7973da` / `eeda8ce7d49a`） | 硬约束 #6：在用 + 待部署 |
| `postgres:15.4` + `17-alpine` | 均在（在用） | 硬约束 #7；只删了旧大版本 `postgres:15` |
| Dockerfile FROM `debian:bookworm-slim` | 在（`7ff6ba31527b`） | 硬约束 #8；其余两个 FROM 不在本机 |
| D 桶基础/工具镜像 14 个（alpine×3、debian×2、redis×2、node、registry、curl、temurin、binfmt、hello-world、halohub/halo） | 全部未动 | D 桶不在授权范围；其中 `halo`/`new-api-own` 等未知或 A 桶仓库按防御性默认保留 |

## 8. 风险与未完成

- 删除不可逆：84 个镜像（含各项目的多个历史版本）已永久删除，只能重新构建/拉取恢复。
  已按硬约束为每个项目保留「在用版 + 上一版」回滚点。
- 其它项目构建/回滚若依赖更早的版本标签（例如 picsur 0.8.x、komari 0.0.4-0.0.22），
  现已不可本地回退，需要重新 pull 或重建。
- 本地 registry 镜像（`127.0.0.1:5001/…` 系列）与 `zhemed/picsur`、`komari-own`、`komari-local`、
  `komari-test`、`picsur-oldroot`、`picsur-reltest`、`v2.26.1-custom.local` 等一次性/测试标签已删除。
- 3 个 digest 形态的构建产物（`ghcr.io/zhemed/komari@sha256:0db27f...`、
  `ghcr.io/zhemed/komari-agent@sha256:03bd8acd...`、`alpine@sha256:48b0309c...`）也在 C 桶授权内并已删除
  （均为无容器引用的 manifest 产物；`alpine:3.21`、`alpine:3.23`、`alpine:latest` 三个正常 alpine 镜像
  属 D 桶，未动）。
- **未做**：Build Cache 现在多出 13.26GB 可回收，本轮**未清**（不在本任务授权内）；容器/卷/网络/`data/` 未动。
- 未提交：本文件留工作区，待 Lead 按闸门收口（锚点 `[task:prune-reclaimable-images]`）。

## 9. 建议下一步

1. Lead 复核可抽查：`docker images --format '{{.Repository}}:{{.Tag}}' | grep -E 'new-api-own|komari|picsur|litepan|postgres'`
   （应只剩第 7 节保留项）与 `df -B1 /`。
2. 若要继续回收，建议单列一次 `docker builder prune -f`：预计释放约 13GB（本轮实测已把它从 85.79MB 抬到 13.26GB），
   收益远大于卷侧 814.6MB；需用户对「清构建缓存」再次点头（task-12 当时的裁定是保留）。
3. 若其它项目需要更早版本回滚，请在其仓库重新构建或从 registry 重新拉取；必要时把这些版本固定到发布流程里。
