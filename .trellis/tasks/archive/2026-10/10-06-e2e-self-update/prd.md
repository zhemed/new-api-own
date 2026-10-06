# 真机端到端自更新验证（3020 → v0.0.7）

## Goal

在 3020 试验机上跑通「面板直接更新」的完整闭环：面板识别有新版本 → 二次确认 →
服务端下载 / 校验 / 原子替换二进制 → 原地重执行（`syscall.Exec`）→ 面板与接口确认已是最新。

## Requirements

### R1 更新前
- 面板显示「最新版本 v0.0.7」「更新状态 有新版本可用：v0.0.7」+「立即更新」按钮；截图留证。
- 记录基线：`/api/status` 版本、进程 pid、二进制 sha256。

### R2 执行
- 点「立即更新」→ 二次确认（重启 / 容器重建退回镜像版本两条代价）→ 截图后确认。
- 只点一次；失败不反复重试，保留现场。

### R3 更新后断言
- `GET /api/status` 的 `data.version` 变为 0.0.7 系列。
- 进程 pid 变化情况需按**实际**路径解释：走 `syscall.Exec` 时 pid **不变**（同进程换程序映像），
  走 `exit` 交给重启策略时 pid 才变——以 `UpdateApplyResult.restart_mode` 与 `/proc` 实测为准，不预设结论。
- 磁盘二进制 sha256 == Release `SHA256SUMS` 中 `new-api-linux-amd64` 那一行。
- 面板重新打开显示「已是最新」。

### R4 证据
截图（更新前 / 确认框 / 更新后）+ `/api/status` 输出 + 更新前后二进制 sha256 + 进程实测。

### R5 环境还原
- 管理员密码 hash 还原（测试需要临时改，测完写回原值）。
- 不打印任何凭据；不动用户远程生产；不 commit / push。

## Constraints

- 3020 是试验机（Lead 已授权）；它是 **bare 进程**（非容器），更新器优先 `syscall.Exec` 原地重执行。
- 只在 `web/**` 与 `.local-instance/` 内动作；不新增依赖。

## Acceptance Criteria

- [ ] AC1：更新前面板显示「有新版本可用：v0.0.7」+ 按钮（截图）。
- [ ] AC2：二次确认含重启与镜像两条代价（截图）。
- [ ] AC3：更新后 `data.version` 变为 v0.0.7。
- [ ] AC4：磁盘二进制 sha256 与 Release `SHA256SUMS` 的 amd64 行一致。
- [ ] AC5：进程事实与 `restart_mode` 相符（exec → 同 pid；exit → 新 pid），并说明实际观测。
- [ ] AC6：面板更新后显示「已是最新」（截图）。
- [ ] AC7：管理员密码 hash 已还原、临时凭据文件已删、未 commit/push。
- [ ] AC8：失败情形若发生，贴出接口返回体与 `code` 且不重试。

## Notes

- 本任务是 `10-06-update-check-ui` 的真机验收补充；该任务线代码已交付，此处只做运行验证与留证。
- 细节留 `findings.md`。
