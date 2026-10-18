# 默认范围支持自然月 — 交付

## 一、字段与解析改动

| 文件 | 改动 |
|---|---|
| `types.ts` | 新增 `DashboardRangePresetKind = 'days' \| 'month-to-date' \| 'last-month'`；`DashboardChartPreferences` **保留** `defaultTimeRangeDays: number`，**新增可选** `defaultRangeKind?`（缺省 = `'days'`） |
| `constants.ts` | 不再重复声明 kind 联合，改为从 `types.ts` 导入并 `export type` 转发；`TIME_RANGE_PRESETS` / `NATURAL_MONTH_PRESETS` 形状不变 |
| `lib/filters.ts` | ①`isRangePresetKind`（`Set` 查表）；②`getSavedChartPreferences` 读 `defaultRangeKind`，**非法/缺失 → `'days'`**，天数仍走 `isTimeRangePresetDays` + `migrateLegacyTimeRangeDays`（29→30）；③`saveChartPreferences` **归一化写入**显式 kind；④`buildDefaultDashboardFilters(preferences, now?)` 统一走 **`resolveRangePreset({kind, days}, now)`**，粒度用**实际跨度** `granularityForRangeDays(range.days)` |
| `models-chart-preferences.tsx` | 默认范围下拉改成**两组** `SelectGroup`：天数（1/7/14/30）+ 自然月（This Month / Last Month，复用 `NATURAL_MONTH_PRESETS` 与已有 i18n）；draft 同时持有 kind |

**下拉取值编码**：`days:<n>`（天数档）或 `<kind>`（自然月）——单个 `<Select>` 承载两种形态：
- `value` 由 draft 推导（自然月 → 对应 kind；否则 `days:<days>`）；
- `onValueChange` 解析后写回 `{ defaultTimeRangeDays, defaultRangeKind }`；
- **无需新增 i18n 键**（复用上一轮的 `This Month` / `Last Month`）。

## 二、向后兼容怎么保证
- 存量 payload `{defaultTimeRangeDays: 30}`（无 kind）→ `defaultRangeKind` 读作 `'days'` → **仍解析成 09-19 00:00 → now**（有测试逐字节断言）；
- 29→30 迁移逻辑未改，且在新解析路径下仍生效（有测试）；
- 保存时总是写入显式 kind，所以"旧 payload 被再保存"会升级为 `{defaultRangeKind:'days'}`（有测试）。

## 三、测试（新增 10）

`lib/__tests__/default-range-preference.test.ts`（8，happy-dom + localStorage）：
1. 旧数据无 kind → 09-19 00:00 → now（行为不变）；
2. `month-to-date` → 本月 1 号 00:00 → now；
3. `month-to-date` 跨年（01-03）→ 01-01 00:00；
4. `last-month` → 上月整月（09-01 00:00 ~ 09-30 23:59:59.999）；
5. 非法 kind（`last-year`）→ 回落 `days`，并按 7 天窗口解析；
6. 保存→读取**往返**一致（last-month 与 days 各一次）；
7. 无 kind 的旧 payload 再保存 → 落盘为 `defaultRangeKind: 'days'`；
8. 29→30 迁移仍生效，且解析为 09-19 起。

`components/models/__tests__/models-chart-preferences.test.tsx`（2，组件级）：
1. 打开偏好对话框 + 展开范围下拉 → 选项包含 `1/7/14/30 Days` **以及** `This Month` / `Last Month`；
2. `defaultRangeKind: 'last-month'` 时，下拉触发器显示 `Last Month`（说明读取的是 kind 而非天数）。

## 四、七条门
typecheck **0 错误** ｜ lint **0 warnings / 0 errors** ｜ `bun test` **337 pass / 0 fail**（327 → +10）｜
format:check exit 0 ｜ build exit 0 ｜ **knip exit 0** ｜ i18n:check exit 0（无新增键）。

## 五、边界
只改 `dashboard/**`（4 个源文件 + 2 个测试目录）；`detectQuickRangeDays` 未改（自然月范围不匹配天数档，属预期）。未 commit。
