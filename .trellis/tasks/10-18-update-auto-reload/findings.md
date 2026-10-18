# 面板内更新成功后自动刷新 — 交付

## 一、改动文件与流程

| 文件 | 作用 |
|---|---|
| **新增** `maintenance/restart-watcher.ts` | 纯逻辑 + 可注入依赖：`waitForRestartedService`（轮询）、`watchForRestartAndReload`（轮询→一次性 reload）、`shouldWatchAfterApply`（只认 ok）、`hasReloadedFor/markReloadedFor`（sessionStorage 守卫）、`readFreshStatus`（绕缓存探针） |
| `maintenance/update-checker-section.tsx` | `ok` 后：关弹窗 → `phase='restarting'`（**按钮禁用 + 常驻提示**，不再用 toast）→ 调 `watchForRestartAndReload` → 超时则 `phase='timed-out'`（提示改手动刷新、按钮恢复）。新增测试接缝 `postUpdateWatch`（生产恒为默认实现） |
| 7 个 locale | 新增 2 条文案（重启中 / 请手动刷新）；删除失真的旧文案 `Update started. The service will restart shortly, then reload this page.`（行为已自动，且不再用 toast） |
| `__tests__/restart-watcher.test.ts`（新增） | 10 个用例，全部注入假时钟 |
| `__tests__/update-checker-section.test.tsx` | 更新 apply 用例 + 新增"失败绝不启动 watch"用例 |

## 二、轮询参数与判据

| 参数 | 值 | 说明 |
|---|---|---|
| 首次延迟 | **2s** | 服务需要片刻才开始重启 |
| 退避 | **×2，上限 15s** | 失败（网络/非 2xx/抛错）一律重试，不会狂刷 |
| 总超时 | **90s** | 到点停止并转手动提示 |
| 成功判据① | 200 且 `version !== 更新前版本` | 立刻 reload |
| 成功判据② | 200 且 `version === 更新前版本`，**连续 2 次** | 同版本重装兜底，随后 reload（单次成功不算，避免关闭过程中的偶发响应触发刷新） |
| 超时判据 | 90s 内未达成 | **不刷新** + 常驻提示改为"请手动刷新" |
| 探针 | `api.get('/api/status', { params: { _ts: Date.now() }, disableDuplicate: true })` | 唯一时间戳绕开缓存 + `disableDuplicate` 跳过客户端 in-flight GET 去重；实例本身已带 `Cache-Control: no-store` |

**只在 `result.ok === true` 路径**进入 watch（`shouldWatchAfterApply`），失败 reason 全部不触发

## 三、"reload 只发生一次"如何保证 / 验证

- **守卫键**：`sessionStorage['update-reload-completed-for'] = <目标版本>`；
  `watchForRestartAndReload` 在 reload **之前**先写入，重复触发发现同版本已写 → 直接返回、不再 reload。
- **键是"目标版本"而非布尔值**：所以同一版本不会二次刷新，而**以后升级到别的版本仍可自动刷新**（不会把功能永久锁死）。
- **测试验证**：
  - `reloads` 数组断言 `['0.0.20']`（恰好一次）；
  - 同一 deps 连续调用两次 → 仍 `['0.0.20']`，且 `hasReloadedFor('0.0.20') === true`；
  - 升级到 `0.0.21` → `['0.0.20','0.0.21']`（不误伤后续更新）；
  - 已运行新版本的页面再次进入 → 不 reload。

## 四、测试（新增 11 个，全部假时钟/注入依赖，不真等 90s）

1. 版本变化 → `version-changed`（`'fail'` 两次后成功）；
2. 同版本连续 2 次成功 → `recovered`；
3. 单次成功 + 随后失败 → **不**判恢复（timeout）；
4. 一直失败 → `timeout`，且耗时落在 90s 附近（退避有界）；
5. 版本变化 → reload **恰好一次**；
6. 超时 → reload **零次**；
7. 重复触发同版本 → 不二次 reload；
8. 后续版本 → 允许再次 reload；
9. 已运行新版本 → 不 reload；
10. `shouldWatchAfterApply`：`ok:true` → true；`disabled/busy/not_newer/download_failed/checksum_failed/unsupported_platform/too_large/replace_failed/http_error` 全部 false；
11. 组件级：apply 失败（`not_newer`）→ `postUpdateWatch` **从未被调用**，且页面上没有 restart 提示。

## 五、七条门

typecheck **0 错误** ｜ lint **0 warnings / 0 errors** ｜ `bun test` **300 pass / 0 fail**（289 → +11）｜
format:check exit 0 ｜ build exit 0 ｜ **knip exit 0** ｜ i18n:check exit 0。

## 六、边界
只改 `system-settings/maintenance/**` + 7 个 locale；无后端改动；未 commit。
