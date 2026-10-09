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

## 执行结果（2026-10-09）

用户三项全选，按"只清一次日志"的建议**合并为一次升级**：

| 项 | 结果 |
|---|---|
| 死代码清理 | ✓ 4 个生成器 + 级联 8 个符号删除（`mock-stats.ts` 854→445 行、`seed.ts` 63→43 行）；真实路径仍用到的（`PROFILE_BY_NAME`/`LatencyTimePoint`/`UptimeDayPoint`/`aggregateUptime`/`hashStringToSeed`/`seededRandom`）与两张已标注的 mock 表**保留**；**knip 收敛 6 条** |
| 最后 2 条 lint 警告 | ✓ 采用"保留实现 + 带三条依据的就地 disable"（不强行替换为语义不同的 `structuredClone`）；`lint` 达 **0 warnings / 0 errors** |
| 实例升级 | ✓ 面板路径 v0.0.11 → **v0.0.13**（约 10 秒，容器未重建，四条 `LOG_*` 与 `--log-opt` 未变）|
| 浏览器复核 | ✓ 顶栏 `New API v0.0.13`、无错误边界；控制台 KPI 渲染 `243.1 t/s` 且为**绿色**（新上色实时生效）|

### 过程中的一次自伤（成员如实报告，已修复）

清理脚本按"行首 `}`"判断**数组字面量**边界，误吞了紧随其后的真实路径符号 `PROFILE_BY_NAME`；
`typecheck` 当场报错、3 个测试变红 → 已从 HEAD **逐字节还原**并恢复全绿（最终 diff 中表现为"移动"）。
教训：块语句才用 `}` 配对，数组/对象字面量须按各自收尾符判断。

### 发布

**v0.0.13**：镜像 `0.0.13`/`latest` 同一摘要 `sha256:291b8c3a94445`；Release 成功；`releases/latest = v0.0.13`。
终验：无 Go 改动；typecheck / lint(0/0) / 224 pass / build / i18n(4181 键) / format / knip(files) 全绿；门禁通过。

### 仍未处理（都不急）

1. knip 全量基线仍红（本次 404→398 条），含既有 `ApiTabIcon` 未使用导出；
2. `formatTokenVolume`（`mock-stats.ts:65`）0 引用但属**改动前**即存在的死代码，按指令未删（1 行可清）；
3. 2 条 `prefer-structured-clone` 以带依据的 disable 收尾（若日后收窄类型契约可真正移除）。
