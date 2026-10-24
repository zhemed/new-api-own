# 残留任务审计证据（2026-10-24）

只读审计，命令与关键输出如下。

## 1. 非归档任务目录

```bash
ls -1 .trellis/tasks/
# 10-24-leftover-task-audit
# archive
```

审计前非归档任务为 0；`10-24-leftover-task-audit` 是本轮为回答该问题而新建的任务。

```bash
python3 .trellis/scripts/task.py list
# All active tasks:
#   - 10-24-leftover-task-audit/ (in_progress) <- current
# Total: 1 task(s)
```

## 2. 归档仓完整性

脚本遍历 `.trellis/tasks/archive/**/task.json`：

- 文件数：188
- status 分布：`{'completed': 188}`
- status != completed：0
- completed 但 completedAt 为空：0
- 记录 worktree_path 的任务：0
- branch 字段：`{'main': 177, None: 11}`
- 归档目录数 188，与 task.json 数一致，无缺 task.json 的目录

## 3. git 残留

```bash
git worktree list      # /root/new-api-own d67382d [main]（仅主工作区）
git branch -a          # * main；remotes/origin/main；无本地 feature 分支
git stash list         # 空
git status --porcelain # 仅 ?? .trellis/tasks/10-24-leftover-task-audit/
```

## 4. 运行态 / 非任务残留（gitignored）

```bash
ls -la .trellis/.runtime/sessions/   # 仅当前会话 dsh_session-66ad...json，current_task 指向本任务
find .trellis/.runtime -name '*.marker'  # 9 个 update-check-*.marker（2026-09-18 ~ 2026-10-23）
find .verify-scratch -type f | wc -l     # 23 个文件，du -sh = 2.6M（2026-10-09 ~ 10-19 验证截图）
```

上述均为 gitignored 生成物，不属 Trellis 任务，不影响流程；如需清理应另建任务执行。
