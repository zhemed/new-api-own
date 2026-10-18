# 调查：面板更新成功后是否自动刷新界面

## 用户报告（2026-10-18）

「在 ui 更新的时候，我们好像没有加上更新成功后自动刷新界面，你去调查一下」

## 背景（Lead 已知）

- 面板更新走 `POST /api/status/update/apply`，后端替换二进制后 `syscall.Exec` 原地重执行（进程重启）；
- 前端静态包随新版本替换（hash 变化），但**用户浏览器里已加载的旧 SPA 不会自动换成新包** —— 必须刷新页面才会加载新 bundle。
- 因此"更新成功后自动刷新"是必要的收尾动作（否则用户看到的仍是旧界面，甚至旧 API 契约）。

## 只读调查项

1. `update/apply` 的调用点（文件:行号），成功后都做了什么（toast / 轮询版本 / 跳转 / `location.reload()`）；
2. 是否存在"等待服务恢复再刷新"的逻辑（进程重启期间请求会失败，需轮询 `/api/status` 直到新版本可用）；
3. 版本检查 UI（有新版本提示/对话框）里是否有同样的收尾；
4. 若缺失，给出最小补法（含：刷新前等待服务可用、超时兜底、失败提示、避免刷新循环）与风险。

## Acceptance Criteria

- [ ] 给出调用点与现状行为（含成功/失败两条路径）
- [ ] 明确"是否自动刷新"，缺失则给出补法与风险
- [ ] 不写代码（调查结论交用户决定）

## 调查结论（2026-10-18，只读）

### 现状：**没有自动刷新** ✗（提示语却在说会刷新）

`web/src/features/system-settings/maintenance/update-checker-section.tsx` 的 `handleConfirmUpdate()`（约 :86-101）：

```ts
const result = await applyUpdate()
if (result.ok) {
  setConfirmOpen(false)
  toast.success(t('Update started. The service will restart shortly, then reload this page.'))
  return                       // ← 到此为止，没有任何轮询/跳转/reload
}
```

- `applyUpdate()`（`update-api.ts:177`）只负责发 `POST /api/status/update/apply` 并把结果翻译成 `{ok, reason, message}`，**不轮询**；
- 全仓唯一的 `window.location.reload()` 在 `routes/__root.tsx:83`，与更新流程无关；
- ⇒ 用户看到的是"**提示说会刷新，实际要手动刷**"，而且旧 SPA 仍在跑旧 bundle（更新后前端包 hash 已变），
  不刷新就会继续用旧界面（极端情况下新旧 API 契约不一致）✗。

### 建议补法（最小改动，含兜底）

1. `result.ok` 后进入"更新中"状态：禁用按钮 + 常驻提示（不要只弹 3 秒 toast）；
2. **轮询** `GET /api/status`（进程重启期间会失败 → 网络错误按退避重试，2s 起、上限约 60–90s）；
3. 判定成功：状态接口 **200 且返回的 `version` 与更新前不同**（或至少服务已恢复）→ `window.location.reload()`；
4. **超时兜底**：超时后 toast「更新可能已完成，请手动刷新」，并恢复按钮可点（避免死等）；
5. **防循环**：只刷新一次（本地一次性标志），且仅在 `ok === true` 时刷新；失败路径（`busy`/`not_newer`/下载/校验失败等）**绝不刷新**；
6. 注意 `/api/status` 可能被前端缓存（有 `newapi:default:cache-version` 之类键）→ 轮询时加时间戳/`cache: 'no-store'`。

### 风险

- 低：改动只在前端更新组件内；刷新发生在服务已恢复之后；
- 需注意：若更新后版本号未变（同版本重装），第 3 步的"version 变化"判据不成立 → 可用"服务恢复 + 连续两次成功"作为替代判据，或直接以"状态接口恢复"为准。
