# 更新确认弹窗保持打开并显示重启进度 — 交付

## 一、弹窗状态机改法

`handleConfirmUpdate()` 的 `ok` 分支**不再** `setConfirmOpen(false)`：

```ts
if (result.ok) {
  if (!shouldWatchAfterApply(result)) return
  setPhase('restarting')          // 弹窗保持打开，进度就在弹窗里
  const outcome = await (postUpdateWatch ?? watchForRestartAndReload)({...})
  if (outcome.status === 'timeout') setPhase('timed-out')
  return
}
```

弹窗按 `phase` 切换（文案用 if/else 预先算好，避免嵌套三元 —— 本仓 lint 禁止）：

| phase | 标题 | 正文 | 主按钮 | 次按钮 |
|---|---|---|---|---|
| `idle` | Update to {{version}}? | 原有两条重启/镜像说明 | Update now（可点） | Cancel |
| **`restarting`** | **Restarting the service…** | 进度说明（`data-testid=update-restart-progress`，含"通常几秒、最长约 90 秒"） | **Restarting…（disabled）** | **disabled** |
| **`timed-out`** | **Update may be complete** | "请手动刷新页面"（`data-testid=update-restart-timeout`） | **Close（可点 → 只关弹窗、不刷新）** | Cancel |
| 失败 | 原确认标题 | 原说明 | Update now | Cancel（错误仍走 toast，弹窗不收） |

- `isLoading={applying || phase === 'restarting'}` → `ConfirmDialog` 内部用它对**两个**按钮加 `disabled`
  （`AlertDialogCancel disabled={isLoading}`、确认按钮 `disabled={disabled || isLoading}`），
  所以重启中用户点不动、也关不掉；
- `disabled={phase === 'restarting'}` 再兜一层；
- `onOpenChange` 在 `restarting` 时**拒绝关闭**（防 Esc / 点遮罩把进度关掉）；
- 成功路径由 watcher 调 `window.location.reload()`，弹窗随整页消失（天然收尾）。

## 二、页面常驻提示：只保留「超时」一种

原先 `phase !== 'idle'` 就渲染 section 提示 → 与弹窗重复。现改为 **`phase === 'timed-out'` 才渲染**
（文案即"更新可能已完成，请手动刷新…"）。理由：重启中弹窗里已有进度，页面再提示是冗余；
而超时后用户关掉弹窗，页面上这条提示还能兜住"接下来该干嘛"。

## 三、新文案（7 语言，走脚本）
- `Restarting the service…` / `Restarting…` / `Update may be complete` /
  `The update was applied. The service is restarting and this page reloads automatically once the new version answers — usually within a few seconds, at most about 90.`
- `Close` 复用既有键；`The update may already be finished…` 复用上一轮的键。
- `i18n:check` 通过（4170 键 × 7 语言）。

## 四、测试（新增 2，另扩展 1）

`__tests__/update-checker-section.test.tsx`（现 18 个）：
1. **成功路径弹窗不关**：watcher 用可控 Promise 挂起 → 断言 `[data-slot=alert-dialog-content]` 仍在、
   `data-testid=update-restart-progress` 存在、watcher 收到 `previousVersion='0.0.6'`、
   主按钮文案为 `Restarting…` 且 **disabled**；随后放行 Promise 收尾；
2. **超时路径**：watcher 返回 `{status:'timeout'}` → 弹窗仍在、`update-restart-timeout` 存在；
   点 `Close` → 弹窗消失（且该路径不触发 reload，reload 守卫由 `restart-watcher` 的用例保证"恰一次"）；
3. **失败路径**（`not_newer`）：弹窗仍在且仍显示 `Update now`，`postUpdateWatch` 零调用。

补充说明：`reload` 恰好一次的保证仍由 `restart-watcher.test.ts` 覆盖（守卫按版本 + 调用计数断言）——
组件测试注入的是 `postUpdateWatch` 桩，不触发真实 reload。

## 五、七条门
typecheck **0 错误** ｜ lint **0 warnings / 0 errors** ｜ `bun test` **341 pass / 0 fail**（339 → +2）｜
format:check exit 0 ｜ build exit 0 ｜ **knip exit 0** ｜ i18n:check exit 0。

## 六、边界
只改 `system-settings/maintenance/**` + 7 locale。未 commit。
