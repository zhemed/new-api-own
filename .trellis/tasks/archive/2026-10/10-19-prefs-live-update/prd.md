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

## 实现与真机验证（2026-10-19，v0.0.35）

### 实现

- `chart-preferences-events.ts`：listener Set + `subscribe/emit`（+ 跨标签页 storage 事件）；
- `use-chart-preferences.ts`：`useSyncExternalStore`，**用 localStorage 原始串做快照身份** ——
  原始串不变就复用同一对象引用（`getSavedChartPreferences()` 每次解析都造新对象，直接用会无限重渲染）；
- `saveChartPreferences()` 写盘后 `emitChartPreferencesChange()`；
- 5 处读取点改为订阅：dashboard 主页面、排行榜默认周期（**仅挂载读**）、热门模型、市场份额、barrel。

### Lead 真机验证（实例 v0.0.35，无刷新）

1. 打开 `/dashboard/models`：图表切换按钮为 **面积图（默认，选中）/ 柱状图** ✓；
2. 偏好弹窗实际内容：默认范围 `上月/本月`、默认消耗分布图 `面积图/柱状图`、默认模型调用图 ✓；
3. 选「柱状图」→ 点「**保存偏好设置**」→ **不刷新页面**：
   - `localStorage.dashboard_models_chart_preferences = {"consumptionDistributionChart":"bar", ...}` ✓；
   - 图表切换按钮变为 **柱状图选中、面积图未选中** ✓；
4. 再改回「面积图」保存 → 存储回到 `"area"`、按钮回到面积图选中 ✓（用户偏好已恢复原状）。

截图：`.verify-scratch/v0035-prefs-open.png`、`v0035-prefs-restored.png`。
