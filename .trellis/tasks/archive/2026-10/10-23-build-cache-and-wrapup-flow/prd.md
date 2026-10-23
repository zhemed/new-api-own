# 清理 Build Cache 并固化归档后补推流程

## Goal

用户指令（2026-10-24）：① 清理 Docker Build Cache（约 24.22GB 可回收）；② 把「Trellis 归档后补推」
做成固定流程，免去每次建临时任务。**备份策略本轮仍不做**（用户未选，风险已在上一任务与 journal 记录）。

## 背景与关键事实

- 上一轮清理后 `docker system df` 显示 Build Cache 约 **24.22GB** 可回收，是本机磁盘收益最大的目标；
  它可能分属默认 builder 与 buildx 两处，需要实测确认。
- 「归档后补推」的痛点：`task.py archive` 会在归档瞬间自动产生一个纯 `.trellis` 提交，
  而此时本会话已无进行中任务 → 闸门拦掉后续所有工具调用 → 只能建临时任务过闸门。
- **推断（需实验证实）**：闸门在**工具调用边界**检查任务状态，因此把 `finish` → `archive` → `git push`
  放进**同一次调用**即可整体通过，无需临时任务。

## Requirements

- R1 Build Cache 清理：
  - 清前取证（`docker system df` / 必要时 `-v`、builder prune 的旗标语义、buildx 缓存统计），写清 24.22GB 归属；
  - 只清构建缓存，**不得**使用会连带删除镜像/容器/卷的命令；
  - 清后核验：容器 8 个全 Up、`new-api` 200 + `v0.0.45`、镜像与卷清单不变、`data/` 完好、`df` 前后对比。
- R2 固化补推流程：
  - 新增 `scripts/trellis-wrapup.sh`：`finish` → `archive --skip-branch-validation` → `push origin <branch>` → 校验 `0 0` 与工作区干净；
    带守卫（已跟踪文件有未提交改动 / 任务不存在 / 非 in_progress 时拒绝）与 `--dry-run`；
  - 在 `.trellis/spec/guides/trellis-gate-guide.md` 记录标准序列与「为什么必须在同一次调用内」，
    并把「建临时任务过闸门再删」记为不推荐的历史做法（不留审计记录）；
  - 在 /tmp 建临时仓库 + bare remote 做端到端测试（happy path / 脏树拒绝 / 任务不存在 / 非 in_progress / dry-run）。
- R3 本轮收尾**用新脚本真实执行一次**（Lead），同时充当对「同一次调用不被拦」的实证。
- R4 红线：不访问外部主机；不碰 `data/`；不改部署形态；不提交不推送由成员执行的部分。

## Acceptance Criteria

- [ ] AC1 Build Cache 清理完成，`docker system df` 前后对比与 `df -h /` 精确字节回收量有证据。
- [ ] AC2 清理后 8 个容器全 Up、`new-api` running/`restarts=0`/200 + `v0.0.45`、镜像（2 标签同 ID）与卷（3 个）清单不变、`data/` 55 文件完好。
- [ ] AC3 `scripts/trellis-wrapup.sh` 落地，`bash -n` + `shellcheck -S warning` 通过，守卫与 `--dry-run` 行为有测试证据。
- [ ] AC4 闸门指南新增一节，说明标准收尾序列与临时任务做法的代价；文档渲染安全。
- [ ] AC5 /tmp 端到端测试四类用例全部有命令与输出证据。
- [ ] AC6 本轮真实收尾用新脚本执行成功：归档提交被推送、`git rev-list --left-right --count` 为 `0 0`、工作区干净。

## Notes

- 构建缓存清理会让其它项目下次构建变慢（无数据丢失），属预期副作用，需在证据与交付说明中写明。
- 若实验证伪「同一次调用可以通过」，脚本与文档需按实测结论改写（例如要求 push 前先完成归档提交，
  并在文档里写明真实限制）——以实测为准，不要照抄本 PRD 的推断。
