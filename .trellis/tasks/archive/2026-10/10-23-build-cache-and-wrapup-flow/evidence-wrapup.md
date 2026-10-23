# evidence-wrapup：Trellis 收尾标准流程（脚本 + 指南 + /tmp 端到端测试）

- 任务板：`task-13`（Trellis 任务 `10-23-build-cache-and-wrapup-flow`，PRD 见同目录 `prd.md`）
- 执行者：`worker-docs`；基线 HEAD `e6ec2d8`（2026-10-23）
- 写作用域内改动：`scripts/trellis-wrapup.sh`（新增）、`.trellis/spec/guides/trellis-gate-guide.md`（追加第 9 节）、本证据文件
- 未触碰：真实仓库的 `.trellis/tasks/`（未执行 finish/archive）、`data/`、任何远端与主机

## 一、结论

1. 新增 `scripts/trellis-wrapup.sh`：一条命令完成 `finish → archive → push → 校验`，
   带 7 类守卫（见下）、`--dry-run`（零改动、零网络动作）、清晰的四阶段输出（归档路径、归档提交 hash、
   推送范围、最终同步状态）。`bash -n` 与 `shellcheck -S warning` 均通过（退出码 0）。
2. `/tmp` 端到端测试：在带 **bare remote 作 origin**、含**真实 `.githooks`（`core.hooksPath=.githooks`）** 与
   `.trellis/scripts/` 副本的临时仓库里跑，**9 个场景 / 46 条断言全部 PASS**（`PASS=46 FAIL=0`）。
3. **关于"同一次调用内 finish+archive+push 是否不被闸门拦"（实验结论）**：
   - 已验证的部分：整条链可以在**单次 bash 调用**内完成，中途不产生任何工具调用；
     仓库层闸门（`pre-commit` / `commit-msg`）对归档提交**放行**，因为归档提交只改 `.trellis/`（第 4 节"刻意放行"）；
     同一临时仓库里的对照实验证明 hook 确实生效：纯 `.trellis/` 提交在无进行中任务时放行，
     非 `.trellis/` 提交在同样状态下被拒（T9a / T9b）。
   - 未能验证的部分（诚实边界）：**会话层（harness）闸门**无法在临时仓库里复现——它按会话状态判定，
     只在真实仓库的真实收尾时才触发。本任务的证据只能证明"脚本把整条链压进一次调用"，
     "这次调用不被会话层闸门拦"需要 Lead 用本脚本做真实收尾来确认（PRD 已把该步列为最终验证）。
4. 文档：`.trellis/spec/guides/trellis-gate-guide.md` 追加第 9 节，写明标准序列、为什么必须同一次调用、
   脚本用法与 `--dry-run`、远端预检的理由，并把"建临时任务过闸门、推完删掉"记为**不推荐的历史做法**
   与三条代价（审计链断裂为主）。

## 二、改动清单

| 文件 | 改动 |
|---|---|
| `scripts/trellis-wrapup.sh` | 新增（可执行位已设）：`<任务名或slug> [--dry-run]`；守卫 + 四阶段 + 校验 |
| `.trellis/spec/guides/trellis-gate-guide.md` | 追加 `## 9. 收尾标准序列（一条命令做完，2026-10-23 定调）`（含"不推荐的历史做法"与"失败恢复"两小节） |
| `.trellis/tasks/10-23-build-cache-and-wrapup-flow/evidence-wrapup.md` | 本证据 |

脚本守卫一览（全部为"拒绝即退出、零改动"）：

1. 不在 git 仓库；
2. 没有 `origin` 远端；
3. detached HEAD；
4. 跟踪文件有未提交改动（`git status --porcelain` 中的非 `??` 行）；
5. 任务目录不存在，或已归档（会指出归档路径）；
6. `task.json` 的 `status` 不是 `in_progress`（打印实际值）；
7. 远端预检失败（`git push --dry-run origin <分支>`），避免"归档成功、推送失败"后无任务可用。

命令形态（与既有 `scripts/*.sh` 一致：`#!/usr/bin/env bash` + `set -uo pipefail` + 彩色 ✓/✗ + 中文说明）：

```bash
scripts/trellis-wrapup.sh 10-23-my-task            # 真跑
scripts/trellis-wrapup.sh my-task --dry-run        # 只打印
```

## 三、验证

### 1. 静态检查

| 命令 | 结果 |
|---|---|
| `bash -n scripts/trellis-wrapup.sh` | 退出码 0（无语法错误） |
| `shellcheck -S warning scripts/trellis-wrapup.sh` | 退出码 0（无 warning 及以上） |
| `git status --short` | 仅 `M .trellis/spec/guides/trellis-gate-guide.md`、`?? scripts/trellis-wrapup.sh`、`?? .trellis/tasks/10-23-build-cache-and-wrapup-flow/` |

### 2. /tmp 端到端测试

测试环境（`/tmp/trellis-wrapup-e2e/run-tests.sh`，每次重新搭建）：

- `origin.git`：`git init --bare`；`repo`：`git init -b main`，`core.hooksPath=.githooks`（复制真实 hook）；
- 复制真实 `.trellis/scripts/`、`.trellis/config.yaml`、`.trellis/.developer`；
- 复制真实 `scripts/forbid-extra-deploy-methods.sh`（pre-commit 会调用它）与被测脚本；
- 假任务目录 + `task.json`（可指定 status）；初始提交用 `[task:<slug>]` 锚点，并 `push -u origin main`。

| 场景 | 断言要点 | 结果 |
|---|---|---|
| T1 happy path（slug 形式） | 归档目录 `.trellis/tasks/archive/2026-10/10-23-fake-task` 存在；HEAD 提交为 `chore(task): archive 10-23-fake-task`；本地 HEAD 与 bare remote 的 `main` 相同；`rev-list --left-right --count` 为 `0 0`（制表符分隔两个 0）；工作区干净 | PASS |
| T2 脏树 | 退出码非 0；输出含"拒绝执行/未提交"；HEAD、提交数、`git status` 与执行前逐项相同；任务目录仍在 `tasks/` | PASS |
| T3 任务不存在 | 退出码非 0；输出"找不到任务"；HEAD 与工作区未变 | PASS |
| T4 非 in_progress | 退出码非 0；输出 `status=pending`；HEAD 未变；任务未被归档 | PASS |
| T5 `--dry-run` | 退出码 0；输出含"不会被执行"；HEAD、提交数、`git status` 与执行前相同；`task.json` 仍 in_progress | PASS |
| T5b dry-run 后真跑（完整目录名） | 退出码 0；归档提交 `chore(task): archive 10-23-fake-task`（证明 dry-run 没有消耗任务） | PASS |
| T6 无 origin | 退出码非 0；输出指向 origin；HEAD 未变；未归档 | PASS |
| T7 远端预检失败（origin 指向不存在的路径） | 退出码非 0；输出"远端预检失败"；HEAD 未变；**任务仍在 `tasks/` 且仍 in_progress**（可修好后重跑） | PASS |
| T8 名字歧义（`10-23-dup-task` + `10-24-dup-task`） | 退出码非 0；输出"匹配到多个目录"；HEAD 未变 | PASS |
| T9a 对照：纯 `.trellis/` 提交在无进行中任务时 | 提交成功（放行）——解释归档提交为何不被拦 | PASS |
| T9b 对照：非 `.trellis/` 提交在无进行中任务时 | 被 `pre-commit` 拒绝（输出含"拒绝提交"）——证明临时仓库里闸门 hook 真实生效 | PASS |

汇总：`PASS=46 FAIL=0`，测试脚本退出码 0（脚本最终版复跑结果；临时仓库每次重建，提交 hash 每次不同）。

关键原始输出（T1，真实跑，脚本最终版）：

```
== 收尾计划 ==
== 1/4 finish（清除当前任务指针） ==
  ✓ 已 finish
== 2/4 archive（移动到 archive/YYYY-MM/ 并自动提交） ==
  ✓ 归档到 .trellis/tasks/archive/2026-10/10-23-fake-task
  ✓ 归档提交 1b117cf chore(task): archive 10-23-fake-task
== 3/4 push origin main ==
  ✓ 待推送 1 个提交：
  ✓ 已推送
== 4/4 校验（0 0 + 工作区干净） ==
  ✓ origin/main 与 HEAD 同步（0 0）
  ✓ 工作区干净（跟踪文件无未提交改动）
== 收尾完成 ==
```

预检失败（T7）时的输出要点：

```
== 拒绝执行 ==
  ✗ 远端预检失败（git push --dry-run origin main），先修好推送通道再收尾：
  未做任何改动（任务仍是 in_progress）。
```

## 四、风险与未完成

- **会话层闸门未在本任务中验证**：见"结论 3"的边界说明。若真实收尾仍被拦，说明该闸门的判定不只在调用边界
  （可能按步/按事件检查），脚本需改成"先 push、后 finish/archive"的顺序变体——但那会破坏"归档提交必须被推送"的
  因果（push 之后再归档就还要再推一次）。请把真实收尾结果回填到本文件或 journal。
- **push 失败后的恢复**：脚本已用预检把风险前置；但若真实 push 在预检通过后失败，任务已归档、会话内无任务，
  恢复只能在会话外推送（指南第 9 节"失败恢复"已写）。
- 脚本不处理"远端已有人推送"的冲突：`git push` 会失败并原样打印 git 错误；不做 force / rebase 等危险动作。
- 测试在 `/tmp` 用副本运行，未覆盖真实仓库的 CI（`.github/workflows/trellis-gate.yml`）与 `check-trellis-gate.sh` 审计——
  这两层只校验历史提交，与本次新增流程无直接耦合。
- `/tmp/trellis-wrapup-e2e/` 保留（含测试脚本与日志 `test-run.log`），便于 Lead 复跑；确认后可删。

## 五、建议下一步

1. Lead 用 `scripts/trellis-wrapup.sh` 做真实收尾（建议先 `--dry-run` 看一遍输出），把结果回填证据/journal；
2. 若真实收尾仍需两次调用，说明会话层闸门另有判定点，届时按上一条风险给脚本加顺序变体；
3. 是否在 `MAINTENANCE.md`「流程闸门」一节加一行指向脚本（本轮未改，超出写作用域）；
4. 可考虑把 `scripts/trellis-wrapup.sh --dry-run` 纳入收尾自证清单（指南第 6 节）。
