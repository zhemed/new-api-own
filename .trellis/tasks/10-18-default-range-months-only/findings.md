# 默认范围只留自然月（上月在前）— 交付

## 一、下拉选项最终形态

**偏好设置 →「默认范围」**（`models-chart-preferences.tsx`）：只剩 2 项，**顺序：Last Month → This Month**。

- `constants.ts` 的 `NATURAL_MONTH_PRESETS` 改为 `[Last Month, This Month]`
  → 偏好下拉与**筛选对话框的月按钮**同时换序（同一常量，一致性 ✓）；
- 偏好下拉**不再生成任何天数项**（`TIME_RANGE_PRESETS` 已从该组件移除）；
- **筛选对话框的 1/7/14/30 天快捷按钮保留**（未改动）。

## 二、旧值兜底实现

```ts
// 只表示"选中值"；天数值不被静默改写
const selectedRangeValue =
  draft.defaultRangeKind === 'last-month' ? 'last-month' : 'month-to-date'

const handleSave = () => {
  // 保存时才落盘为所见的自然月（旧 days 值由此完成迁移）
  props.onPreferencesChange({
    ...draft,
    defaultRangeKind: selectedRangeValue as DashboardRangePresetKind,
  })
}
```

- 旧存档 `{kind:'days', days:30}` → 打开对话框**显示为「本月」**，但 `draft` 与 localStorage **都不变**
  （测试断言：仅打开对话框时 `onPreferencesChange` 一次都没被调用）；
- **点保存**才落盘 `defaultRangeKind: 'month-to-date'`（`defaultTimeRangeDays: 30` 同时保留，作为兼容字段）；
- 保存前图表仍按 30 天滚动窗口渲染（解析层未改）。

**解析层保持向后兼容**：`getSavedChartPreferences` / `buildDefaultDashboardFilters` 继续接受 `days`
（`isRangePresetKind` 允许三种 kind），`{kind:'days',days:30}` 仍解析为 **09-19 00:00 → now**；
**29→30 迁移测试原样通过**。

## 三、测试（净增 2 个；3 个偏好用例替换了原 2 个）

`components/models/__tests__/models-chart-preferences.test.tsx`（3）：
1. 打开对话框并展开下拉 → `[role="option"]` 文本**恰好等于** `['Last Month','This Month']`（顺序断言，
   且不含 `1 Day / 7 Days / 14 Days / 30 Days`）；
   *实现细节*：不能用整页 `textContent.indexOf` 判顺序——触发器本身已显示其中一个标签，会误判；
2. 旧值 `{kind:'days', days:30}` → 触发器显示 **This Month**；**仅打开时不落盘**（spy 为空）；
   点 Save 后 `onPreferencesChange` 收到 `defaultRangeKind: 'month-to-date'`；
3. `defaultRangeKind:'last-month'` → 触发器显示 **Last Month**。

`components/models/__tests__/models-filter-dialog.test.tsx`（1，新增）：
筛选对话框快捷范围仍含 `1 Day/7 Days/14 Days/30 Days` **且**含上月/本月，且**上月在前**。
（Base UI 的 ScrollArea 需要 `Element.prototype.getAnimations`，happy-dom 没有 → 测试里补了个空实现。）

解析层既有测试（`default-range-preference.test.ts` 8 个 + `chart-preferences-migration.test.ts` 5 个）全部保持通过。

## 四、七条门
typecheck **0 错误** ｜ lint **0 warnings / 0 errors** ｜ `bun test` **339 pass / 0 fail**（337 → +2）｜
format:check exit 0 ｜ build exit 0 ｜ **knip exit 0** ｜ i18n:check exit 0（无新增/删除键）。

## 五、边界
只改 `dashboard/**`（2 源文件 + 1 测试文件修改 + 1 测试文件新增）。未 commit。
