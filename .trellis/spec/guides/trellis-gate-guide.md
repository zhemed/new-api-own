# Trellis 闸门指南（强制规则怎么落地、怎么自证）

> 用户 2026-09-18 定调：**调用 Trellis 不是口头承诺，而是强制规则**。
> 本文写"规则是什么、机器怎么拦、我该怎么自证没跳过"。规则原文见 `AGENTS.md` 末尾的
> `TRELLIS-GATE` 段，维护者视角见 `MAINTENANCE.md`「流程闸门」。

---

## 1. 规则（最严范围）

**任何会话工作——包括只读调查——开工第一步必须先建 Trellis 任务。**
"只读调查"指：看代码、查日志、读数据库定位原因、跑诊断命令，全都算。

唯一例外：用户明确说"这次跳过 Trellis"。没有第二条例外；不要自己判断"这次很小所以不用"。

标准开场（先建任务，之后才允许读代码/改文件）：

```bash
python3 .trellis/scripts/task.py create "<标题>" --slug <slug> -d "<描述>"
python3 .trellis/scripts/task.py start <MM-DD-slug>
```

收尾：`task.py finish` → `task.py archive <name> --skip-branch-validation` → 写 journal
（`python3 .trellis/scripts/add_session.py --title ... --commit <sha>`）。

## 2. 机器怎么拦（三层）

| 层 | 文件 | 触发点 | 违规后果 |
|---|---|---|---|
| ① 当场拦 | `.githooks/pre-commit` | `git commit` 且暂存区含非 `.trellis/` 改动、却无进行中任务 | 提交被拒，打印建任务与提交命令 |
| ① 当场拦 | `.githooks/commit-msg` | 同上条件、消息里没有 `[task:<slug>]` 或 slug 不存在 | 提交被拒，打印正确格式与现有任务 |
| ② 事后审计 | `scripts/check-trellis-gate.sh` | 手工运行（发版前必跑） | ✗ 列出违规提交，退出码 1 |
| ③ 远程兜底 | `.github/workflows/trellis-gate.yml` | push 到 main / 开 PR | GitHub 上 workflow 红叉 |

新克隆必须跑一次 `./scripts/install-git-hooks.sh`（`core.hooksPath` 是本地配置，不随仓库分发）。

## 3. 消息格式

```
<type>(<scope>): <一句话说明> [task:<slug>]
```

slug 是 `.trellis/tasks/<MM-DD>-<slug>` 去掉日期前缀的部分。例：

```
feat(relay): 新增渠道适配 [task:add-provider-x]
```

## 4. 刻意放行

- **纯 `.trellis/` 改动**：journal、任务归档、闸门自身 —— 流程产物不该被拦；
- **merge 提交**：被合并的提交各自有审计，合并动作不产生新改动。

## 5. 绕过与代价（诚实写）

`git commit --no-verify` 是 git 内建能力，**封不死**。闸门的设计不是"禁止绕过"，而是
**绕过必留痕**：第 ② / ③ 层审计逐个提交核对，任何漏锚点的提交都会在下一次自检或下一次
push 时暴露。**不许把 `--no-verify` 当常规手段。**

## 6. 自证清单（每个任务收尾前自问）

- [ ] 任务是在**动手之前**建的（看 `task.json` 的 `createdAt` 与首个改动提交的时间关系）；
- [ ] 每个改动提交都带 `[task:<slug>]`（`git log --format='%h %s' <起点>..HEAD` 自查）；
- [ ] `./scripts/check-trellis-gate.sh` 通过；
- [ ] 任务已 `finish` + `archive`，journal 有本次会话记录；
- [ ] 如果为了排查"只读调查"而没建任务——那是违规，别在 journal 里粉饰成"顺手看的"。

## 7. 已知边界（不许假装全自动）

机器只能强制"**提交时**必须有任务"。只读调查不产生提交，因此"只读也要建任务"这条
**没有机械保障**，靠本指南 + `AGENTS.md` + journal 留痕。写文档或向用户汇报时，
必须照实说，不要声称闸门覆盖了一切。

审计起点记录在 `.trellis/gates/enforce-from`（启用闸门那一刻的提交 SHA），此前的仓库历史
（上游与自维护基线）不受审计，避免把历史提交全部标红。

## 8. 其它项目

同一条规则也写在全局 `~/.dsh/AGENTS.md` 的 `TRELLIS-MANDATORY` 段，适用范围是本机所有
根目录存在 `.trellis/` 的项目；本项目（`zhemed/new-api-own`）与 `zhemed/komari` 都是
"三层闸门"的完整实例，其它项目可以照 `.githooks/` + `scripts/check-trellis-gate.sh` + CI 落地。

---

## 9. 收尾标准序列（一条命令做完，2026-10-23 定调）

**为什么必须一条命令做完**：会话层闸门在**工具调用边界**检查"有没有进行中任务"。
`task.py archive` 会当场把任务移出 `.trellis/tasks/` 并自动提交（纯 `.trellis/`），
此时本会话**已经没有进行中任务**——之后哪怕只是单独调用一次 `git push`，也会被闸门拦下。
所以收尾必须放进同一次调用：`finish → archive → push → 校验`。

> 边界说明：本节说的"调用边界"是**会话层（harness）闸门**；仓库层的三层（hook / 审计 / CI）见第 2 节。
> 两者互不替代。`/tmp` 端到端测试能验证的是脚本行为与仓库层 hook，会话层闸门由真实收尾来验证。

标准用法：

```bash
scripts/trellis-wrapup.sh <任务名或slug>             # 真跑：finish → archive → push → 校验
scripts/trellis-wrapup.sh <任务名或slug> --dry-run   # 只打印将要执行的命令，零改动、零网络动作
```

脚本阶段与守卫：

| 阶段 | 动作 | 失败时 |
|---|---|---|
| 守卫 | 不在 git 仓库 / 无 `origin` / detached HEAD / 跟踪文件有未提交改动 / 任务不存在或已归档 / `status` 不是 in_progress / 远端预检失败 | 拒绝执行，**零改动**，退出码 1 |
| 1 | `task.py finish`（清除当前任务指针） | 任务目录仍在，可修正后重跑 |
| 2 | `task.py archive <name> --skip-branch-validation`（移入 `tasks/archive/YYYY-MM/` 并自动提交 `chore(task): archive …`） | 同上（重复执行无害） |
| 3 | `git push origin <当前分支>` | 归档已完成，本地提交仍在；按下方"失败恢复"处理 |
| 4 | 校验 `git rev-list --left-right --count origin/<分支>...HEAD` 的两个字段都是 `0`，且工作区无未提交的跟踪文件 | 非 0 退出并打印差异 |

**归档提交为什么不会被第一层闸门拦**：它只改 `.trellis/`（任务目录从 `tasks/` 移到 `tasks/archive/`），
属于第 4 节"刻意放行"的第一类，`pre-commit` 与 `commit-msg` 都直接放行。
`/tmp` 端到端测试里有对照实验：纯 `.trellis/` 提交在无进行中任务时放行，
非 `.trellis/` 提交在同样状态下被 `pre-commit` 拒绝。

**为什么先做远端预检**：脚本在 archive 之前先跑 `git push --dry-run origin <分支>`。
一旦"归档成功、推送失败"，会话内已无任务可用（后续调用会被拦），修凭据/网络只能在会话外做；
预检把这类失败提前到任务还在的时刻。`--dry-run` 模式下预检只打印、不执行。

### 不推荐的历史做法：建临时任务过闸门、推完删掉

本脚本出现前，归档后的补推靠"建一个临时任务 → 推送 → 把临时任务删掉"（已用过三次）：

- **代价 1（主要）**：临时任务不进 `archive/`、被直接删除，**审计链断在收尾这一步**——
  `.trellis/tasks/` 里找不到它，事后无法回答"谁在什么时候为了什么做了这次补推"；
- **代价 2**：删除任务目录本身是一次纯 `.trellis/` 改动，容易漏提交或顺手提交，制造幽灵删除；
- **代价 3**：每次都要靠人记住顺序，漏一步就得再建一次临时任务，且每次都留下一个"任务创建又消失"的痕迹。

**该做法自本指南起不再使用**；需要补推时用 `scripts/trellis-wrapup.sh`。

### 失败恢复（诚实写）

脚本在第 3 步失败时会打印本地归档提交 hash 与重推命令。此时任务已归档、会话内无进行中任务，
恢复顺序是：

1. 先在会话外（或由用户）执行 `git push origin <分支>`；
2. 若必须回到会话内继续，按第 1 节建一个**真实任务**（记录补推原因与处理结果），不要建了又删；
3. 把这次失败与处理写进 journal，**不要**用临时任务掩盖。
