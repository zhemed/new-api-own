# 面板内更新成功后自动刷新页面

## 现状
`update-checker-section.tsx` 的 `handleConfirmUpdate()` 在 `result.ok` 后只关弹窗 + toast，
**无轮询/刷新**；提示语写着 "…then reload this page"（用户以为自动）。

## 要求
1. `ok` 后进入**更新中**状态：按钮禁用 + **常驻提示**（不用会自动消失的 toast）。
2. **轮询 `GET /api/status`**（绕开缓存：时间戳查询参数 + `cache:'no-store'`）：
   - 重启期间失败 → **退避重试**（2s 起，上限约 60–90s）；
   - **成功判据**：200 且 `version` ≠ 更新前版本 → 立刻 `window.location.reload()`；
   - 90s 内版本不变但服务连续 2 次成功（同版本重装兜底）→ 也刷新。
3. **超时兜底**：90s 未达成 → 常驻提示改"更新可能已完成，请手动刷新页面"，按钮恢复可点，**不刷新**。
4. **只在 `ok === true`** 路径刷新；失败路径（disabled/busy/not_newer/下载/校验/替换失败等）**绝不刷新**。
5. **只刷一次**：一次性守卫（模块级标志 + sessionStorage），避免 reload 循环。
6. 文案与实际行为一致；新文案走 i18n 脚本（7 语言，禁止手改 locale 顺序）。

## 测试（全部假定时器/注入依赖，不真等）
- 轮询到版本变化 → `reload` **恰好一次**；
- 更新失败（各 reason）→ `reload` **绝不**调用；
- 轮询持续失败 → 超时后**不刷新**且提示切换为"请手动刷新"；
- 同版本重装（200 但 version 不变）→ 连续 2 次成功后仍刷新；
- 一次性守卫：成功路径重复触发不会二次 reload。

## 门
七条门全绿（基线 **289**，只能增）：`tsgo -b --force`、`lint`、`bun test`、`format:check`、`build`、`knip`(exit 0)、`i18n:check`。

## 写范围
`web/src/features/system-settings/maintenance/**` + 必要 i18n。不 commit/push。
