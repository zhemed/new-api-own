# 清旧回滚镜像证据：只留 v0.0.49 + v0.0.48（task-51）

- 任务：Trellis `10-24-panic-recovery-and-cleanup`（PRD R3）；共享任务板 `task-51`；执行者 worker-containers
- 执行时间：2026-10-24T06:19:27Z ～ 06:20:01Z（UTC），全程只访问本机
- 事前依据：用户点名「只保留 v0.0.49 与 v0.0.48」；删除 v0.0.47 / v0.0.46 / v0.0.45
- 红线遵守：**未**用 `docker system prune` / `docker image prune -a` / 名字通配删除；未加 `-f` 强删；
  未删卷、未删容器、未动 `data/`；未访问外部主机；未提交未推送

## 0. 结论摘要

| 项目 | 结果 |
|---|---|
| 删除范围 | 仅 3 个点名 tag 及其独占镜像：`v0.0.47`、`v0.0.46`、`v0.0.45` |
| 执行 | 逐条 `docker rmi repo:tag`，**3/3 rc=0**，无失败无跳过 |
| 保留 | `v0.0.49`（= `latest`，完整 ID `sha256:c029c132…90674`）、`v0.0.48`（完整 ID `sha256:e485361d…6234c`） |
| allow-list | 保留集合 **27 个完整 image ID**（点名保留 3 + 容器在用 8 + 范围外镜像），删除候选 3 个，**交集为空** |
| 二次核对 | `new-api` 的完整 image ID **不在**删除清单（SAFE） |
| 镜像计数 | 唯一 30 → **27**；标签行 39 → **36**；悬空镜像仍 **0** |
| 实例 | `v0.0.49` / running / restarts=0 / 挂载 `/root/new-api-own/data` / 200 + `X-New-Api-Version: v0.0.49` |
| 邻接面 | 8 容器全 running；卷 3（dangling 2）；`data/` 59 文件 —— 与删除前一致 |
| **磁盘释放** | `df -B1 /` 已用 83794219008 → 83321532416，**实测释放 472686592 B ≈ 450.79 MiB / 0.47 GB** |
| docker system df | Images 30 个 / 5.907GB（可回收 2.385GB）→ **27 个 / 5.434GB（可回收 1.912GB）** |

## 1. 删除前基线（06:19:27Z）

```console
--- df -B1 / ---
/dev/sda2      269425360896 83794219008 171870580736  33% /
--- docker system df ---
Images          30        8         5.907GB   2.385GB (40%)
Containers      8         8         262.1kB   0B (0%)
Local Volumes   3         1         996.9MB   814.6MB (81%)
Build Cache     172       2         12.58GB   85.79MB

--- images with FULL ids (new-api-own) ---
ghcr.io/zhemed/new-api-own:latest | sha256:c029c13215aaf3ff8a467b2a0f9e930d24455d01d65ecd7166da8a23cfe90674
ghcr.io/zhemed/new-api-own:v0.0.49 | sha256:c029c13215aaf3ff8a467b2a0f9e930d24455d01d65ecd7166da8a23cfe90674
ghcr.io/zhemed/new-api-own:v0.0.48 | sha256:e485361d15d42ce3804fa92a1fcde544b3c4a5d7010c5862fd5f8a7e2876234c
ghcr.io/zhemed/new-api-own:v0.0.47 | sha256:bccc31d54ad7f37d0e33a0f0cfb913a72dce99ce6681a579161f51ba7c59fe55
ghcr.io/zhemed/new-api-own:v0.0.46 | sha256:95e3c1b97e327d56eb2131b8d2a306d63ed2d3a82cdcdc29e336143cc48d5126
ghcr.io/zhemed/new-api-own:v0.0.45 | sha256:e114b17532f179760f87410595c0821d8b116955dc1d68cf82c0aaffe25c5c9f

--- image counts ---
unique=30  tags=39  dangling=0
--- instance ---
Image=ghcr.io/zhemed/new-api-own:v0.0.49
ImageID=sha256:c029c13215aaf3ff8a467b2a0f9e930d24455d01d65ecd7166da8a23cfe90674
Status=running Restarts=0 Mount=/root/new-api-own/data
HTTP/1.1 200 OK / X-New-Api-Version: v0.0.49
--- misc ---
containers=8/8  volumes=3  dangling_volumes=2  data_files=59
```

## 2. allow-list（完整 ID，未截断）

### 2.1 将保留：27 个完整 image ID

来源三类：① 点名保留（v0.0.49 / v0.0.48 / latest）；② 容器在用（8 个容器逐个 `docker inspect -f '{{.Image}}'`）；
③ 范围外镜像（非 `new-api-own` 仓库，全部不动）。

```console
### 将保留：27 个 image ID（完整，未截断）###
  KEEP sha256:248755520402e0d1e639ba11eefa68e9e6b3698d47edc91fe7172773cc90a58a  <- 范围外: picsur:0.10/0.10.6/latest/v0.10.6
  KEEP sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6  <- 范围外: alpine:latest
  KEEP sha256:298e5b3bc566bade82f46ad5511777a4a07a294097ce16ada2f6a42be5239df5  <- 范围外: redis:latest
  KEEP sha256:400a4873b838d1b89194d982c45e5fb3cda4593fbfd7e08a02e76b03b21166f0  <- 范围外: tonistiigi/binfmt:latest
  KEEP sha256:4817dbd211934a0c3d78a5acd487fc5307ae1a80b6388da2bf387e73b9d12334 <- 范围外: komari:0.0.23
  KEEP sha256:4d06038800655fe1211760cd561de70ef2ed7a47f5d69255e9834414602b7026  <- 范围外: eclipse-temurin:21-jdk
  KEEP sha256:4dd8bd40598b1657e467c1ae2792d42571695ac16db77b19299ba285fc9a3f3f  <- 容器在用: picsur
  KEEP sha256:5e23090353324d887c48ad5e5c56d294eab81588df9605b07d1afe895f9cc8f8  <- 范围外: hello-world:latest
  KEEP sha256:62bd423aeee912639ca8ee681c2b762cb3e25137e0a92b20be455898a9f034ad  <- 范围外: debian:trixie-slim
  KEEP sha256:6ddd6e7973da665109eedb3b2588df1a032b4d14407470f04e25a1b27c7d14a4  <- 容器在用: litepan
  KEEP sha256:71da9275c5f3fcb97d0fa0c8c5b36cc995327265420f17a04bfd544f458059f7  <- 范围外: redis:7
  KEEP sha256:7ff6ba31527b9e336162fa061746c366c4b2032f4d2cff2078bea43f78efe3f8  <- 范围外: debian:bookworm-slim
  KEEP sha256:85fe1e81d6758c208f3e1eed4338a1997e19d4be002d4dd32d3100c9a8c010a0  <- 范围外: alpine:3.23
  KEEP sha256:8b4d729b7727a37d2f1e2a84167216ab1b4caaf1bfb36b6f8360e2273f10db20  <- 容器在用: halo
  KEEP sha256:a134550be58ab783810e613ae96fd253c38f7bdd16d9bf9be350700c7a29ac06  <- 范围外: curlimages/curl:latest
  KEEP sha256:a3d8aaa63ed8681a604f1dea0aa03f100d5895b6a58ace528858a7b332415373  <- 范围外: registry:2
  KEEP sha256:b0f9560a2de083e2cc7382e75f808c7381a32852a7ec49117deedb300e552b24  <- 容器在用: support-devdb-1
  KEEP sha256:bfe89bf70c44ea94a7dd33f8ec98e8e59088adfb7a9fb7b6f013f82391c33bee  <- 范围外: halohub/halo:2.26
  KEEP sha256:c029c13215aaf3ff8a467b2a0f9e930d24455d01d65ecd7166da8a23cfe90674  <- 点名保留: v0.0.49 / latest；容器在用: new-api
  KEEP sha256:cb4b4cec1be680bdc147eadf192eca54fe0b0b86ccf71bdee49b14ee71b34157  <- 容器在用: komari-agent-test
  KEEP sha256:d1168ad5397bf9c94cc417c530469805a8767ed585481d93f174461ac8557e3a  <- 容器在用: halodb
  KEEP sha256:d261e841e1228798110dbb9a2a31ab13c208bfef992313a7ac26cb79af546bca  <- 范围外: komari-agent:0.0.23
  KEEP sha256:e485361d15d42ce3804fa92a1fcde544b3c4a5d7010c5862fd5f8a7e2876234c  <- 点名保留: v0.0.48
  KEEP sha256:ee16d92f96db08d9d2d07da48555a27f3a69d8f0759021dbc601195ed0d2b5bd  <- 范围外: alpine:3.21
  KEEP sha256:eeda8ce7d49acb2e50b0ef931951de2eb9f23a92df46dc4756319153cc9fc63e  <- 范围外: litepan:0.0.50/latest/v0.0.50
  KEEP sha256:fb4cd12c85ee03686f6af5362a0b0d56d50c58a04632e6c0fb8363f609372293  <- 范围外: node:20-alpine
  KEEP sha256:fc0fbf2ce0f6a313e51b68b5815a96d0c4f43e3a41e0f82c9653902a74304aa9  <- 容器在用: komari-test
```

### 2.2 将删除：3 个点名 tag（各自独占一个完整 image ID，与保留集合无交集）

```console
  DELETE ghcr.io/zhemed/new-api-own:v0.0.47  sha256:bccc31d54ad7f37d0e33a0f0cfb913a72dce99ce6681a579161f51ba7c59fe55  tags=["ghcr.io/zhemed/new-api-own:v0.0.47"]
  DELETE ghcr.io/zhemed/new-api-own:v0.0.46  sha256:95e3c1b97e327d56eb2131b8d2a306d63ed2d3a82cdcdc29e336143cc48d5126  tags=["ghcr.io/zhemed/new-api-own:v0.0.46"]
  DELETE ghcr.io/zhemed/new-api-own:v0.0.45  sha256:e114b17532f179760f87410595c0821d8b116955dc1d68cf82c0aaffe25c5c9f  tags=["ghcr.io/zhemed/new-api-own:v0.0.45"]
```

- 三个 image 的 `RepoTags` 都只有各自一个 tag → 删除 tag 即整镜像删除，不会影响保留镜像的层。
- 无「tag 与保留 image 共享 ID」的情况，故本次无跳过项（skip=0）。

### 2.3 删除前二次核对

```console
  new-api Image（完整） = sha256:c029c13215aaf3ff8a467b2a0f9e930d24455d01d65ecd7166da8a23cfe90674
  in delete list        = False
  RESULT: SAFE
```

## 3. 删除执行（逐条，无 `-f` / 无通配 / 无 prune）

```console
$ docker rmi ghcr.io/zhemed/new-api-own:v0.0.47
  rc=0 :: Untagged: ghcr.io/zhemed/new-api-own:v0.0.47 Deleted: sha256:bccc31d54ad7f37d0e33a0f0cfb913a72dce99ce6681a579161f51ba7c59fe55
$ docker rmi ghcr.io/zhemed/new-api-own:v0.0.46
  rc=0 :: Untagged: ghcr.io/zhemed/new-api-own:v0.0.46 Deleted: sha256:95e3c1b97e327d56eb2131b8d2a306d63ed2d3a82cdcdc29e336143cc48d5126
$ docker rmi ghcr.io/zhemed/new-api-own:v0.0.45
  rc=0 :: Untagged: ghcr.io/zhemed/new-api-own:v0.0.45 Deleted: sha256:e114b17532f179760f87410595c0821d8b116955dc1d68cf82c0aaffe25c5c9f
TOTAL=3 failed=0
```

## 4. 删除后复核（06:20:01Z）

```console
--- images (new-api-own) ---
ghcr.io/zhemed/new-api-own:latest | sha256:c029c13215aaf3ff8a467b2a0f9e930d24455d01d65ecd7166da8a23cfe90674
ghcr.io/zhemed/new-api-own:v0.0.49 | sha256:c029c13215aaf3ff8a467b2a0f9e930d24455d01d65ecd7166da8a23cfe90674
ghcr.io/zhemed/new-api-own:v0.0.48 | sha256:e485361d15d42ce3804fa92a1fcde544b3c4a5d7010c5862fd5f8a7e2876234c
--- counts ---
unique=27  tags=36  dangling=0
--- latest == v0.0.49 ? ---
  OK latest == v0.0.49
  OK v0.0.48 仍在
--- 三个旧 tag 应已不存在 ---
  OK 已删除 v0.0.47
  OK 已删除 v0.0.46
  OK 已删除 v0.0.45
--- instance ---
Image=ghcr.io/zhemed/new-api-own:v0.0.49 Status=running Restarts=0 Mount=/root/new-api-own/data
HTTP/1.1 200 OK / X-New-Api-Version: v0.0.49
--- containers / volumes / data ---
running=8  all=8  volumes=3  dangling_volumes=2  data_files=59
new-api | ghcr.io/zhemed/new-api-own:v0.0.49 | Up 5 minutes
komari-agent-test | ghcr.io/zhemed/komari-agent:latest | Up 20 hours
komari-test | ghcr.io/zhemed/komari:latest | Up 18 hours
picsur | ghcr.io/zhemed/picsur:allinone | Up 4 days (healthy)
support-devdb-1 | postgres:17-alpine | Up 4 days
litepan | ghcr.io/zhemed/litepan:v0.0.49 | Up 4 days
halo | ghcr.io/zhemed/halo:v2.26.1-custom.1 | Up 4 days (healthy)
halodb | postgres:15.4 | Up 4 days (healthy)
```

## 5. 磁盘释放量与计数对比

| 指标 | 删除前 | 删除后 | 变化 |
|---|---|---|---|
| df 已用（字节） | 83794219008 | 83321532416 | **-472686592（≈450.79 MiB / 0.47 GB）** |
| df 可用 | 171870580736 | 172343267328 | +472686592 |
| df -h | 79G 已用 / 161G 可用 / 33% | 78G / 161G / 33% | -1G（取整） |
| 唯一镜像 / 标签行 / 悬空 | 30 / 39 / 0 | 27 / 36 / 0 | -3 / -3 / 0 |
| docker Images 体积 | 5.907GB（可回收 2.385GB） | 5.434GB（可回收 1.912GB） | -0.473GB（与 df 实测一致） |
| 容器 / 卷 / `data/` 文件 | 8 / 3 / 59 | 8 / 3 / 59 | 不变 |

## 6. 回滚方式说明

- 本次删除的是**旧版本回滚镜像**，当前保留 `v0.0.48` 作为上一版回滚点：
  实例回滚 → `docker rm -f new-api` + 原参数、镜像换 `ghcr.io/zhemed/new-api-own:v0.0.48` 重建 + 验 200。
- 若要恢复被删的 v0.0.47/46/45 镜像：从 registry 重新拉取即可（例如
  `docker pull ghcr.io/zhemed/new-api-own:v0.0.47`）；tag 与镜像仍存在于 ghcr，只是本机不再保留。
- 本次未触碰任何容器、卷、构建缓存与 `data/`，因此不存在数据侧回滚需求。

## 7. 风险与未完成

- 删除不可逆（本机）：若将来需要回滚到 v0.0.45~v0.0.47，需要联网重新 `docker pull`（几秒到几十秒，取决于网络）。
- 保留下来的可回收量仍有：镜像侧 1.912GB（主要来自范围外基础镜像，需另授权）、卷侧 814.6MB（两个命名卷，按此前裁定保留）、
  构建缓存 85.79MB —— 均不在本任务范围。
- 未提交：本文件留工作区，待 Lead 按闸门收口（锚点 `[task:panic-recovery-and-cleanup]`）。

## 8. 建议下一步

1. Lead 复核可抽查：`docker images | grep new-api-own`（应 3 行：latest / v0.0.49 同 ID / v0.0.48）。
2. 按「当前 + 上一版」两档保留策略，下次发版后可继续用同一 allow-list 流程清更早版本。
3. 若要进一步压盘，镜像侧可考虑清理范围外基础镜像（需逐项确认用途），属新授权项。
