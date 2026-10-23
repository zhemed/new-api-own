# P5 证据：`scripts/check-version-drift.sh` 未剥 `v` 前缀缺陷修复

- 任务：Trellis `10-23-maintain-clean-pending-items`（P5 代码侧）；共享任务板 `task-3`
- 执行者：worker-script；时间：2026-10-23
- 写作用域：`scripts/check-version-drift.sh`（唯一被改的仓库文件）+ 本文件
- 红线遵守：未提交、未推送、未碰 `MAINTENANCE.md` 与 `web/`；所有验证在 `/tmp` fixture 内完成，
  `curl` / `docker` 均为 `/tmp` 下的 stub，**没有任何真实网络动作，也没有连任何实例**。

## 1. 缺陷定位（修复前）

脚本共 2 个版本比对点，另有 1 处 `v` 前缀处理：

| 行（原文件） | 比对 | 修复前是否有归一化 |
| --- | --- | --- |
| L137 `[[ "${TAG_VERSION}" == "${REPO_VERSION}" ]]` | 最新 git tag vs 仓库 VERSION | 有，但用的是 `LATEST_TAG#v`（L135） |
| L249 `[[ "${instance_version}" == "${REPO_VERSION}" ]]` | 实例 `/api/status` version vs 仓库 VERSION | **无** → `v0.0.45` vs `0.0.45` 误判漂移 |

调用点：`makefile:48` 的 `check-version` 目标（`make check-version INSTANCE=...`）；
`MAINTENANCE.md:454` 有文档引用（未改动）。实例数据只来自 `--instance <url>` 的 `/api/status`。

## 2. 改动清单

`scripts/check-version-drift.sh`，4 处改动、无重构：

1. 新增 `normalize_version()`（第 97-104 行附近）：只剥一个可选 `v` 前缀
   （`printf '%s' "${v#v}"`），版本号本体不动。
2. `REPO_VERSION` 在 VERSION 文件读取 / `--version` 覆盖之后统一归一化一次
   （放在原有空值检查之前，检查语义不变）。
3. `TAG_VERSION="$(normalize_version "${LATEST_TAG}")"` 取代 `"${LATEST_TAG#v}"`（等价，语义不变）。
4. 实例比对前加一行 `instance_version="$(normalize_version "${instance_version}")"`。

退出码语义未动：`0=一致 / 1=drift / 2=无法判定`；`❌/✅/⚠️/⏭` 输出格式与计数（CHECKED/SKIPPED）未动；
禁用网络的 `--no-network`、registry 三标签摘要比对、`extract_version` 回落链均未触碰。
文件头注释补充说明"所有版本比对先归一化"（文档同步）。

## 3. 验证（全部实际执行）

### 3.1 语法检查

```console
$ bash -n scripts/check-version-drift.sh
bash -n OK (exit 0)
```

### 3.2 修复前后对照（/tmp fixture）

fixture：`/tmp/version-drift-fixture/sandbox`（假仓库根：`VERSION` + 假 `.git`）+
`/tmp/version-drift-fixture/sandbox/bin/{docker,curl}` stub（零网络）。
"修复前"= `git show HEAD:scripts/check-version-drift.sh` 写进 `sandbox/scripts/before.sh`；
"修复后"= 工作区脚本写进 `sandbox/scripts/after.sh`——两者同目录，`REPO_ROOT` 解析一致，
比对条件完全相同。脚本：`/tmp/version-drift-fixture/run-matrix.sh`、`run-more.sh`、`run-edge.sh`。

矩阵结果（fixture 内 `VERSION=0.0.45`，tag `v0.0.45`）：

| 用例 | 场景 | 修复前 exit / 判定 | 修复后 exit / 判定 | 期望 |
| --- | --- | --- | --- | --- |
| A | 实例报 `v0.0.45`（tag/registry 全一致） | 1 / DRIFT（误报） | **0 / OK** | 修复后不报 ✅ |
| B | 实例报 `v0.0.46`（真漂移） | 1 / DRIFT | 1 / DRIFT | 真漂移仍报 ✅ |
| C | registry `v0.0.45` 标签摘要不一致 | 1 / DRIFT | 1 / DRIFT（且实例项由 ❌ 转 ✅） | 真漂移仍报 ✅ |
| D | 实例报 `0.0.45`（无前缀，旧行为） | 0 / OK | 0 / OK | 不回归 ✅ |
| E | registry 查不到 + 实例 `v0.0.45` | 1 / DRIFT | **2 / UNKNOWN** | 归因正确 ✅ |
| F | tag 无 `v`：`0.0.45` vs `VERSION=0.0.45` | 2 / UNKNOWN（tag 项 ✅） | 2 / UNKNOWN（tag 项 ✅） | tag 路径不回归 ✅ |
| G | `--version v0.0.45` vs tag `0.0.45` | 1 / DRIFT（误报） | 2 / UNKNOWN（tag 项 ✅） | 覆盖值也归一化 ✅ |
| H | VERSION 自带 `v`：`v0.0.46` vs 实例 `0.0.46` | 1 / DRIFT（误报） | 2 / UNKNOWN（两处 ✅） | 两侧都归一化 ✅ |
| J | 实例 version 恰为 `"v"`（归一化后为空） | 1 / DRIFT | 1 / DRIFT（真 dif） | 不误判为一致 ✅ |
| K | 实例 `0.0.46` vs VERSION `0.0.45`（无前缀真漂移） | 1 / DRIFT | 1 / DRIFT | 不漏报 ✅ |

> F/G/H 的 exit=2 是 `--no-network` / 未指定实例导致的"跳过"项，属脚本既有语义，非本次改动引入。

关键用例 A 的原始输出摘录：

```console
########## A. v 前缀假漂移：tag=v0.0.45 / VERSION=0.0.45 / 实例 v0.0.45 ##########
CASE A | impl=before
    ✅ 仓库: VERSION=0.0.45 == 最新 tag v0.0.45
    ❌ 实例: http://127.0.0.1:30000/api/status version=v0.0.45 != VERSION=0.0.45
    已核对 5 项，跳过/无法判定 0 项
    ==> 判定: DRIFT（三处版本不一致，按事故处理）
    [exit=1]
CASE A | impl=after
    ✅ 仓库: VERSION=0.0.45 == 最新 tag v0.0.45
    ✅ 实例: http://127.0.0.1:30000/api/status version=0.0.45 == VERSION=0.0.45
    已核对 5 项，跳过/无法判定 0 项
    ==> 判定: OK（仓库 VERSION、最新 tag、registry 标签摘要、实例版本一致）
    [exit=0]
```

真漂移用例 B 的原始输出摘录：

```console
CASE B | impl=after
    ✅ 仓库: VERSION=0.0.45 == 最新 tag v0.0.45
    ❌ 实例: http://127.0.0.1:30000/api/status version=0.0.46 != VERSION=0.0.45
    已核对 5 项，跳过/无法判定 0 项
    ==> 判定: DRIFT（三处版本不一致，按事故处理）
    [exit=1]
```

### 3.3 `normalize_version` 边界

```console
    normalize_version("0.0.45") = "0.0.45"
    normalize_version("v0.0.45") = "0.0.45"
    normalize_version("v") = ""
    normalize_version("0.0.4") = "0.0.4"
    normalize_version("5") = "5"
    normalize_version("") = ""
    normalize_version("1.0.0-rc1") = "1.0.0-rc1"
    normalize_version("v1.0.0-rc1") = "1.0.0-rc1"
```

### 3.4 真实仓库离线 smoke（无网络、无实例）

```console
$ bash scripts/check-version-drift.sh --no-network
≫ ✅ 仓库: VERSION=0.0.45 == 最新 tag v0.0.45
≫ ⏭  registry: 跳过 ghcr.io/zhemed/new-api-own:{latest,0.0.45,v0.0.45}（--no-network）
≫ ⏭  实例: 未指定
==> 判定: UNKNOWN（有检查项查不到/被跳过，不能当作一致；联网或修正参数后重跑）
[exit=2]        # 与修复前一致：离线即 UNKNOWN，未因本次改动变化
```

## 4. 风险与未完成

- **未联网验证**：`--instance` 路径只用 stub 覆盖，未对任何真实实例跑过（红线要求，且需授权）；
  registry 三标签摘要比对同样只用 stub 覆盖。
- 归一化只剥**一个**前导 `v`（`vv0.0.45` → `v0.0.45`，仍会判漂移）。这是保守选择：
  宁可多报不清洗，避免把畸形输入"洗白"成一致。
- 实例比较改成对归一化后的值比较，成功/失败行的回显因此显示归一化值
  （`version=0.0.45` 而非 `version=v0.0.45`）。判定正确，只是原始串不再回显；
  如希望回显原值需再加一个变量，当前按"最小改动"未加。
- 未执行提交（红线）：改动留在工作区，`git diff -- scripts/check-version-drift.sh` 可复核。

## 5. 建议下一步

1. Lead 复核 diff 后按 Trellis 闸门提交（提交消息需带 `[task:maintain-clean-pending-items]`）。
2. 若要在 CI 里防回归，可把 fixture 矩阵（stub curl/docker）固化为离线测试用例；
   当前未做，需先确认 CI 是否接受 `/tmp` fixture 形态。
3. 实例真实核对（`make check-version INSTANCE=...`）需实例访问授权后再补，属 P1/P2 授权范围。
