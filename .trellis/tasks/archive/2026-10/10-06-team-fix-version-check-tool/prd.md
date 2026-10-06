# 版本对齐核查脚本（复用编制）

## Goal

把"版本对齐核查"从文档里的一句话变成**机械可执行**的一条命令：比对
①仓库 `VERSION` ②最新 git tag ③ghcr registry 上 `latest` / `<版本>` / `v<版本>` 的摘要
④（可选）运行实例 `/api/status` 的 version，并给出**明确、不可能误报**的结论与退出码。

背景：此前只把核查步骤写进了文档，结果没人执行；本任务负责脚本与 make 目标。

## Requirements

1. 新增 `scripts/check-version-drift.sh`：
   - 用 `set -euo pipefail`；纯 bash；不依赖 jq（用 grep/sed 解析 `imagetools inspect` 文本输出），
     若 jq 存在可用于 `/api/status` 的 JSON 解析（或同样用 grep）。
   - 输入：
     - `--instance <url>`（可选）读取 `<url>/api/status` 的 `version`；未给则跳过该处并标注 SKIP；
     - `--registry <ref>`（可选，默认 `ghcr.io/zhemed/new-api-own`）；
     - `--no-network`（可选，强制跳过 registry/实例检查，用于离线自测）。
   - 检查项：
     a. 仓库 `VERSION` vs 最新 git tag（`git tag --sort=-v:refname | head -1`，容忍 `v` 前缀）；
     b. registry 上 `latest`、`<VERSION>`、`v<VERSION>` 三个标签的摘要/是否存在——三者应指向同一 digest；
     c. 可选实例 `/api/status` 的 version 与 `<VERSION>` 比较。
   - **退出码语义**（必须严格）：
     - `0` = 全部一致
     - `1` = 发现不一致（drift）
     - `2` = 无法判定（registry 不可达/标签缺失/命令缺失/实例不可达）
     - 关键约束：**任何"查不到"都不能算"一致"**。若 registry 与实例都查不到，
       整体判定必须是 `2`（无法判定），绝不能返回 0。
   - 输出：人话结论（`OK` / `DRIFT` / `UNKNOWN`）+ 逐项明细（checked/skipped + 原因）。
   - 需要处理 `docker` 命令不存在或 Docker 环境未达标的情况（提示后按 UNKNOWN 处理）。
2. `makefile`：新增 `check-version` 目标调用脚本（透传参数，例如 `make check-version INSTANCE=http://127.0.0.1:3000`）。
3. `MAINTENANCE.md` 的"版本对齐核查"段**只加一行**命令引用（其余内容归 review-ops，不重写该段）。
4. 自测：
   - 正常路径（本机当前状态）应一致或明确 SKIP；
   - 故意构造不一致（如 `VERSION` 与 tag 不符 / 传入不存在的版本 tag）应报 DRIFT 且退出码 1；
   - 离线场景（`--no-network` 或 registry 不可达）必须报 UNKNOWN 且退出码 2。

## Acceptance Criteria

- [ ] `scripts/check-version-drift.sh` 存在、可执行、`bash -n` 通过。
- [ ] 退出码三态严格（0/1/2），"查不到"绝不报一致。
- [ ] `make check-version` 可运行并透传参数。
- [ ] `MAINTENANCE.md` 只增加一行引用。
- [ ] 两类实测输出已贴（正常 + 故意不一致），另附离线 UNKNOWN 实测。
- [ ] `git diff` 只涉及 `scripts/`、`makefile`、`MAINTENANCE.md`（+1 行）；不改 Go/web。
- [ ] 汇报 ≤20 行。

## 硬约束

- 无外部网络访问：**仅** ghcr.io 只读查询（`docker buildx imagetools inspect`）；
  不做任何 push/写操作；不 commit/push。
- 不改 Go/web 代码；不动 `.local-instance/`。
- 不 `git add -A`。

## Notes

- 共享任务：`task-15`（claim → complete）。
- 本脚本的"实例"检查只读 `/api/status`，不写任何状态。
- 若本机 Docker 环境不达标准（Docker Engine 29.7.2 + Compose v5.4.0），
  脚本仍必须可用（输出 UNKNOWN + 明确提示），自测以离线路径为主。

## 交付与实测结果

### 交付

- `scripts/check-version-drift.sh`（755，`set -euo pipefail`；只读：`docker buildx imagetools inspect` + `curl /api/status`）。
- `makefile`：`.PHONY` 增加 `check-version`，目标透传 `INSTANCE=` / `REGISTRY=` / `VERSION=`。
- `MAINTENANCE.md`：「版本对齐核查」段**只加一行**引用（该段正由 review-ops 改写，我只加这一行）。

### 实测（2026-10-06，本机 Docker 29.7.2 / Compose v5.4.0）

| 场景 | 命令 | 结果 | 退出码 |
|---|---|---|---|
| 正常（联网） | `./scripts/check-version-drift.sh` | VERSION=0.0.6 == tag v0.0.6；latest/0.0.6/v0.0.6 三标签同一摘要 `sha256:a74d0a6c…`，判定 OK | **0** |
| make 目标（正常） | `make check-version` | 同上 | **0** |
| 构造不一致（离线） | `./scripts/check-version-drift.sh --version 9.9.9 --no-network` | `❌ VERSION=9.9.9 != 最新 tag v0.0.6`，判定 DRIFT | **1** |
| 构造不一致（联网，含不存在的 tag） | `./scripts/check-version-drift.sh --version 9.9.9` | 另报 `ghcr.io/...:9.9.9 / :v9.9.9 not found`，判定 DRIFT | **1** |
| make 目标（不一致） | `make check-version VERSION=9.9.9` | 脚本判定 DRIFT 可见；make 自身报 Error 1 | 2（make 语义） |
| 离线 | `./scripts/check-version-drift.sh --no-network` | 仓库内一致，但 registry/实例跳过 → 判定 UNKNOWN（**不报一致**） | **2** |
| 实例不可达 | `./scripts/check-version-drift.sh --instance http://127.0.0.1:9` | `curl: (7) Connection refused` → UNKNOWN | **2** |
| 参数错误 | `./scripts/check-version-drift.sh --bogus` | usage + 提示 | **2** |

静态检查：`bash -n` 通过；`shellcheck -S warning` 无告警；从任意 cwd 调用均能定位仓库根。

### 说明

- 退出码语义：`0`=一致、`1`=drift、`2`=无法判定；**任何查不到/被跳过一律 2，绝不报 0**。
- 实例检查为可选（`--instance`），未给时该项记 SKIP 但不影响整体判定。
- `--version` 用于自测/CI 覆盖仓库版本值，方便构造不一致场景。
- 网络动作仅限 ghcr.io 只读查询（本任务授权范围）；实例自测用本机回环 `127.0.0.1:9`（拒绝连接），未触碰 `.local-instance/`。
