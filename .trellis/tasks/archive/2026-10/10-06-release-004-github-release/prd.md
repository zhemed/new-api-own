# 补发 v0.0.4 的 GitHub Release

## 事故（自查）

发 `v0.0.4` 时：VERSION 递增、tag 推送、镜像构建都做了，**但 GitHub Release 没有产生** ——
因为 `release.yml` 的 `on:` 只保留 `workflow_dispatch`（触发面收敛，用户先前点名要的），
我既没跑它、也没在发版汇报里点明"Release 尚未生成"。结果 Releases 页停在 v0.0.3（Latest），
看起来像什么都没发。文档里我甚至写了"必须在 tag ref 上手动跑"，却没执行。

## 处置

1. 在 tag ref 上手动触发：`gh workflow run release.yml --ref v0.0.4 -f name="v0.0.4"`；
2. 等构建完成，校验 `gh release view v0.0.4` 有 Linux amd64/arm64 + macOS + Windows 与 checksums；
3. 确认 Releases 页的 Latest 变成 v0.0.4；
4. 决策：是否把 `Release` 恢复为 tag 自动触发（只恢复它，Electron/GitCode 仍手动），
   避免"发版漏 Release"再来一次。

## Acceptance Criteria

- [ ] v0.0.4 Release 存在且带三平台产物 + checksums
- [ ] Latest 指向 v0.0.4
- [ ] 文档/流程与执行一致（含防再犯措施）

## 用户新指示（2026-10-06，暂停）

「暂停，我们只维护 Linux 的电脑版本，其它的跟我们又没关系。」

- 已**取消**正在运行的多平台 Release（run 37459587394，含 macOS/Windows job）；
- 待定：产物范围收敛为 **Linux（amd64/arm64）**；Electron 目前只构建 **Windows**，与"只维护 Linux"冲突；
- 未动文件：等用户确认收敛方式（改 release.yml 为 Linux-only / 或整体去掉 GitHub Release 只用镜像）。
