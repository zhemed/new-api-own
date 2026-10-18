# 修复：消耗分布 tooltip 模型值为 0

## 用户报告（2026-10-18）

截图：tooltip 显示 `10-09 / 总计: 1.16M / deepseek-flash 0`
→ **总计正确（Token 口径 ✓）**，但**每模型那行是 0** ✗。

## 排查方向

- 「总计」与「每模型」在 tooltip 里由不同分支取值：
  - 总计来自维度合计（`dimension.content` 的合计/`TimeSum` 等）；
  - 每模型来自 `mark.content` 的 `value(datum)`。
- 重点看是否仍在读**已删除的旧字段**（如 `rawQuota`）或读错字段名（`Usage` vs `rawTokens`），
  导致 `Number(undefined) || 0` → 0。
- 同时核对柱状图（`spec_bar`）与面积图（`spec_area`）两套 spec 是否都改到了；以及「Other」合并桶分支。

## Acceptance Criteria

- [ ] 定位到具体行（文件:行号）与错误字段
- [ ] 修复后 tooltip 每模型值 = 该模型该时间桶的 Token 数，与总计自洽
- [ ] 补回归测试：构造两模型两时段数据，断言每个 (time, model) 的 tooltip 值等于其 token，且同桶合计 = 总计
- [ ] 七条门全绿；发版并真机截图验证

## 修复与验证（2026-10-18，v0.0.22）

### 根因（成员复现，先红灯后修）

`charts.ts` 的 `spec_line.dimension.content[0].value`（:631）与 `spec_area.dimension.content[0].value`（:676）
在上一轮被写成了 `formatTokenValue(...)` → 返回**字符串** `'1.68M'`；
但 vchart 的 dimension 流程要求 `content[].value` 给**原始值**，由 `updateContent` 负责格式化，
而后者内部是 `Number(array[i].value) || 0` → `Number('1.68M')` = **NaN → 0** ✗。
总计走独立数字字段 `datum.TimeSum`，所以出现"**总计对、分项 0**"。

**柱状与面积两套 spec 都错**，面积图只是用户当时在看的视图。

### 修法

两处恢复为 `Number(datum?.rawTokens) || 0`（加注释"不可返回格式化字符串"）；`updateContent` 继续格式化；
`mark.content` 保持字符串。**先写红灯复现测试**（`'0' !== '1.68M'`）再修 → 新增 3 个测试，`bun test` 286 → **289**。

### 另一件必须记录的事：我造成了一次会话额度耗尽 ✗

- 排查过程中发现浏览器登录返回 **409**：`service/auth_session.go:404` → `AUTH_SESSION_LIMIT`；
- 查明 `user_sessions` 表有 **50 条 active 会话**，全部是我这些轮次用 curl 登录产生的（每人 30 天有效）→ **占满并发会话额度**，
  会**连带阻止用户在新设备登录**；
- 用户选择"全部清掉" → 已备份 `/root/data/one-api.db` 后就地清空 `user_sessions`（50 → 0）→ 登录恢复 **HTTP 200** ✓；
- **副作用**：用户自己的浏览器会话也已失效，需要重新登录一次（用户已知情同意）。
- **改进**：以后不再"每轮 curl 登录一次"，改为在临时文件里复用同一 token，用完不重复登录，避免再次堆积会话。

### 验证状态

- 单元/契约测试：已锁定"每模型 tooltip 值 = 该模型该桶 token，且同桶合计自洽"（bar/area 两套）；七条门全绿（289 pass、knip exit 0）；
- **真机截图未完成** ✗：headless 浏览器的登录表单流程在本实例上不落会话（接口 200 ✓ 但前端不完成登录），
  故本次无法自动悬停取 tooltip；**请用户在真实浏览器把鼠标悬停到柱子上确认**（总计与模型行都应显示 Token 数）。
