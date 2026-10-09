# 修正「已知不一致」过期条目 + 复核剩余 6 条

- 任务：`10-09-fix-stale-known-inconsistency`
- 执行：`review-ops`（写范围：`MAINTENANCE.md` 一处区块）
- 共享任务：`task-25`（Lead 指派）

## 目标

1. 把 `MAINTENANCE.md`「已知不一致」表里**已过期**的 multipart `model` 校验行移除，
   改写进「团队审查已处理项」（带文件:行与验证方式）。
2. **只核对不改**：逐条复核「已知不一致」剩余 6 条是否仍成立，给证据行号与判定。

## 硬约束

- 只改 `MAINTENANCE.md`；不 commit/push；不碰代码与实例。

## 验收

- diff 摘要；6 条核对结论（仍成立/已失效 + 证据行号）。

---

# 执行记录（2026-10-09）

## 1. 过期条目处置（已改）

| 动作 | 内容 |
|---|---|
| 从「已知不一致」删除 | `\| multipart 图片编辑缺 model 必填校验（JSON 分支有）\| relay/helper/valid_request.go:195-231 vs :240-243 \| …刻意未改… \|` |
| 写入「团队审查已处理项」 | 新行（现 293 行）：**已修复（`7e017cd`）**；`:196` 取表单 `model`、`:200-202` 空值即 `errors.New("model is required")`、注释 `:197-199`；JSON 分支同校验 `:246-249`；验证方式 `grep -n 'model is required' relay/helper/valid_request.go`（两条分支各一处）+ 回归测试 `relay/helper/openai_image_request_test.go:80`（断言 `:120`）|
| 顺带修的格式问题 | 「已知不一致」表最后一行与 `## 团队审查已处理项` 之间**缺一个空行**（此前编辑遗留），已补 |

**代码证据**（只读核对）：
- `relay/helper/valid_request.go:196` `imageRequest.Model = formData.Get("model")`；`:200-202` 空值返回错误；注释 `:197-199`
- JSON 分支：`:246-249`
- 回归测试：`relay/helper/openai_image_request_test.go:80` `TestGetAndVallidOpenAIImageRequestMultipartModelRequired`，断言 `:120`
- 修复提交：`7e017cd fix(relay): 堵住 sora 时长上界绕过、multipart 补 model 校验、客户端错误改 400`

## 2. 剩余 6 条复核结论（**未改内容**）

| # | 条目 | 证据行号 | 判定 |
|---|---|---|---|
| 1 | 客户端请求 DTO 用非指针标量 + `omitempty` | `relay/common/relay_info.go:849`（`Duration int json:"duration,omitempty"`，行号吻合）；`dto/video.go:7`（`Duration float64 json:"duration"`）；钳制 `relay/relay_task.go:124-126`、`relay/common/relay_utils.go:148-160` | **仍成立**（非指针标量、与 AGENTS.md 不符、改动波及大）。**但描述需精确化**：`dto/video.go` 的 Duration 已不在 `:10-12` 且**没有** `omitempty`（"omitempty" 只对 `relay_info.go` 那处成立） |
| 2 | 日志裁剪的 `DELETE ... LIMIT` 仅 MySQL 生效 | `model/log.go:753` `TrimLogToMaxRows`；注释 `:749-752` 明写 "only MySQL renders DELETE ... LIMIT; GORM's SQLite and PostgreSQL dialects drop the clause" | **仍成立**（表内用函数名引用，无行号漂移） |
| 3 | `count_token_failed` 走 500 | `controller/relay.go:154` 仍是 `types.NewError(err, types.ErrorCodeCountTokenFailed)`；默认状态码 `relaykit/types/error.go:257` = 500；`relaykit/types/error.go:156/204/233` 只影响脱敏，不改状态码 | **仍成立**（同提交改 400 的是 `sensitive_words_detected` 与 `invalid_request`，不是它） |
| 4 | `get_channel_failed`（重试取渠道）走 500 | `controller/relay.go:320`、`:323`（行号不变，仍是 `ErrorCodeGetChannelFailed` + `ErrOptionWithSkipRetry`，该选项只跳过重试、不改码）；默认 500 同上 | **仍成立** |
| 5 | ClickHouse 日志库不支持行数上限 | 策略已上移：`model/log_budget.go:60-81` `LogRowCap()`（CH 返回 0 + 一次性 WARN）；`service/system_task.go:151-155` 仅委托；字节预算对 CH 同样关闭 `model/log_budget.go:85-99` | **仍成立**，但表内位置 `service/system_task.go:logCleanupMaxRows` **已不是事实来源** → 建议更新为 `model.LogRowCap()` |
| 6 | 前端孤儿模块（knip ignore 决策） | `web/knip.config.ts`：`routeTree.gen.ts` 的 ignore **已移除**（注释：由 `src/main.tsx` 可达）；`i18n/static-keys.ts` **已不再是** "未 ignore 且被列为 Unused"——它已接进 `scripts/check-i18n-keys.mjs`/`i18n:check` 入口，注释明确"不要再加回来"；实测 `bun run knip --include files` → **无未使用文件输出** | **部分失效**：`ui/`、`ai-elements/` 仍登记 ignore（产品决策）+ "不要据 ignore 删依赖" 的警告仍成立；但 `routeTree.gen.ts`/`static-keys.ts` 两句已过期 |

## 3. 校验

- `MAINTENANCE.md` 围栏 40 个（配平）；「已知不一致」表现为 **表头 + 分隔 + 6 行**（全部为"决定不改"）；
  「团队审查已处理项」表为 **表头 + 分隔 + 3 行**。
- 引用行号逐条 `sed -n` 复核（含修正 `test:79 → test:80` 一处笔误）。
- 未 commit/push；未改代码、未碰实例；除只读 `grep`/`sed`/`bun run knip --include files`（本地）外无动作。
