# 排行榜自然月/本地化日期/图表类型与周期跟随偏好 + 消耗分布默认面积图

## 用户要求（三张截图）
1. 消耗分布/模型调用的视图切换：**默认面积图**，顺序对调为 **面积图 → 柱状图**。
2. 排行榜页：
   - 「本月」必须是**自然月**（本月 1 号 00:00 → 现在），当前是滚动 30 天 ✗；
   - 日期**跟随界面语言**（现在固定英文 `Sep 26` ✗）；
   - 图表类型**跟随已保存偏好**（偏好=面积图，现在只有柱状图 ✗）；
   - 默认周期也跟随偏好（偏好=本月 → 打开即选中「本月」）。

## 改动
### 后端（`service/rankings.go`）
- `month` 周期改为**自然月**：`start = 本月 1 号 00:00（服务器本地时区）`、`end = now`；
  上一周期（`hasPrevious`）相应改为**上一个自然月**整月。
- `today` / `week` / `year` 语义**保持不变**；`label` 字段**保留**（API 兼容），前端不再使用其英文格式。

### 前端
- `dashboard/constants.ts`：`consumptionDistributionChart` 默认 `'area'`；`CONSUMPTION_DISTRIBUTION_CHART_OPTIONS` 顺序 `[area, bar]`。
- `rankings/components/models-section.tsx`：spec `type` 取已存偏好（area → `'area'` 且 `stack: true`；否则 `'bar'`）。
- 排行榜标签改为**前端用 `points[].ts` 本地化格式化**（`Intl.DateTimeFormat(i18n.language, …)`）：
  `today` 用时分、其余用 `month:'short', day:'numeric'`；`market-share-section` 用同一套。
- `rankings/index.tsx`：URL 未指定 `period` 时，默认周期跟随偏好（自然月偏好 → `month`，否则沿用 `week`）。

## 测试
- Go：`month` 范围 = 本月 1 号 00:00 → now；含跨年 1 月边界；上一周期 = 上月整月。
- 前端：①偏好 area → spec `type==='area'` 且 `stack===true`，偏好 bar → `'bar'`；②默认偏好 `'area'` 且选项顺序 `[area, bar]`；
  ③标签格式化 zh/en 分别产出中文/英文（固定 ts）；④默认周期来自偏好（本月 → `period==='month'`）。

## 门
Go：`go build ./...` + `make test`；前端七条门（基线 **346**，只能增）。

## 写范围
`service/rankings.go`、`web/src/features/rankings/**`、`web/src/features/dashboard/constants.ts`（及必要测试）。不 commit/push。
