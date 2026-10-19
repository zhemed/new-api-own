# 偏好保存后当前页图表即时生效

## 要求
点「保存偏好设置」后，图表**立即**按新偏好渲染，不需刷新/离开页面；排行榜页同样生效。

## 要实现
1. 新增 `useChartPreferences()`（`dashboard/lib/`，barrel 导出）：
   - `useSyncExternalStore(subscribe, getSnapshot)`；**快照必须稳定**（同一引用直到真正变化）；
   - `saveChartPreferences()` 写盘后 **notify** 订阅者（模块内 listener 集合）；
   - 监听 `storage` 事件做跨标签页同步（幂等）。
2. 替换读取点：排行榜 3 处 + 模型分析图表类型读取；`rankings/index.tsx` 的默认周期**只在挂载时**读偏好
   （用户手切到「今年」后不能被偏好重置）。
3. **保留** `getSavedChartPreferences()`（非 React 场景/测试使用）。
4. 不改偏好 JSON 结构、不新增存储键。

## 测试
- `saveChartPreferences()` 后订阅者收到**一次**通知，快照为新值；未保存时快照引用**稳定**（`===`）；
- 组件级（无需 vchart）：保存后图表 `type` 由 bar→area 变化；
- 默认周期只在挂载时读偏好（先切"今年"再保存偏好不会被重置）。

## 门
七条门全绿（前端基线 **369**，只能增）。

## 写范围
`dashboard/lib/filters.ts`、新增 hook、`dashboard/index.tsx`、`rankings/**`、测试。不 commit/push。
