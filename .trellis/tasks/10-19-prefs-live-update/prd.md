# 偏好订阅：保存后图表立即刷新

## 用户要求（2026-10-19）

上一轮我说明「停留在页面时切换偏好不会热更新，需离开再回来」，用户回复截图（偏好设置弹窗 + 「保存偏好设置」按钮）：
**「和这个按钮绑定怎么样」** → 点保存后，图表应**立即**按新偏好渲染，不需要刷新/离开页面。

## 只读调查

1. `saveChartPreferences()` 的写入路径与调用点（`models-chart-preferences.tsx` 的 handleSave）；
2. 各图表读取偏好的方式（`getSavedChartPreferences()` 在 `useMemo` 里读 localStorage → 无订阅）；
3. 需要订阅的位置：模型分析 3 个图表（消耗分布 / 模型调用 / ？）+ 排行榜 2 个图表。

## 设计方向

- 新增 `useChartPreferences()`：`useSyncExternalStore` + 自定义事件（`saveChartPreferences` 内 dispatch）或小 store，
  保证**同页**保存后立即重渲染；
- 兼容 localStorage 跨标签页（storage 事件可选，非必须）；
- 不改变偏好 JSON 结构与既有测试语义；订阅失败不影响渲染（回退到当前值）。

## Acceptance Criteria

- [ ] 保存后当前页图表立即按新范围/图形渲染（同页、无刷新）
- [ ] 排行榜页同法生效
- [ ] 有测试覆盖"保存 → 订阅者收到新值"，且旧偏好读取接口仍可用
- [ ] 七条门全绿；发版；Lead 真机点保存验证
