# 确定 v0.0.12 升级与剩余三项是否处理

## 背景

v0.0.12 已发布（镜像三方同摘要 `sha256:3a963ac94c520`、Release 成功、`releases/latest=v0.0.12`）。
实例仍在 v0.0.11。本轮四项已全部完成并终验通过。

## 待用户决定（多选）

1. **升级实例到 v0.0.12**（面板路径，约 10 秒，容器不重建；内存日志历史会被清空，属已接受代价）；
2. **清理 4 个死代码 mock 生成器**（`buildLatencyTimeSeries`/`buildUptimeSeries`/`buildGroupPerformance`/`buildAppRankings`，全仓 0 调用），顺带看能否收敛 knip 基线；
3. **收尾最后 2 条 lint 警告**（`prefer-structured-clone`，需先收窄 `cloneTemplate<T>` 的类型契约，属签名变更）。

## Acceptance Criteria

- [ ] 记录用户选择
- [ ] 按选择执行（升级需复核版本/环境不变；清理需六条命令全绿）
- [ ] 未选中的项明确记录为"暂不处理"
