# 排行榜自然月/本地化日期/偏好跟随 + 消耗分布默认面积图 — 交付

## 一、改动清单

### 后端 `service/rankings.go`
| 改动 | 说明 |
|---|---|
| `rankingPeriodConfig` 新增 `calendarMonth bool` | `month` 档置 `true`，其余档不动 |
| `rankingTimeRange` | `calendarMonth` 时 `start = time.Date(now.Year(), now.Month(), 1, 0,0,0,0, now.Location())`，`end = now` |
| `previousRankingTimeRange` | `calendarMonth` 时上一周期 = **上一个自然月整月**（`start=上月1号00:00`，`end=本月1号-1s`） |
| `today` / `week` / `year` | **语义完全未动**（仍 `now - duration`） |
| API `label` 字段 | **保留**（前端不再使用其英文格式） |

### 前端
| 文件 | 改动 |
|---|---|
| `dashboard/constants.ts` | `consumptionDistributionChart` 默认 `'bar'` → **`'area'`**；选项顺序 → **`[area, bar]`** |
| `rankings/lib/format.ts` | 新增 `formatRankingBucketLabel(ts, locale, hourly)`：`Intl.DateTimeFormat`，按界面语言输出（`today` 档用 `hour/minute`，其余 `month:'short'+day:'numeric'`），无法解析时回退原值 |
| `rankings/lib/chart-preference.ts`（新增） | `rankingChartType(pref)` → `'area' \| 'bar'`；`defaultRankingPeriod(rangeKind)` → 自然月偏好映射 `'month'`，否则 `'week'` |
| `rankings/components/models-section.tsx` | `type: chartType`（来自已存偏好，`stack: true` 保留）；`orderedPoints` 改用 `point.ts` 生成标签（覆盖后端英文 label）；month 档描述文案改为 "across this month" |
| `rankings/components/market-share-section.tsx` | 同样用 `ts` 本地化标签（保持两卡一致） |
| `rankings/index.tsx` | URL 未指定 `period` 时用 `preferredRankingPeriod()`（= 偏好推导） |

### i18n
替换 1 个键：`Daily token usage by model across the past month` → `Daily token usage by model across this month`
（7 语言，走临时脚本 + `sync-i18n`；`i18n:check` 4170 键全通过）。

## 二、测试

### Go（`service/rankings_test.go`，4 个，testify）
1. `month` 范围 = 2026-10-01 00:00 → now（本机时区）；
2. 跨年：now=2026-01-03 → start=2026-01-01 00:00；
3. 上一周期 = 2026-09-01 00:00 ~ 2026-09-30 23:59:59（`currentStart-1`）；
4. `week`/`today`/`year` 仍是滚动窗口（`now - duration`）。

### 前端（`rankings/lib/__tests__/preferences-and-labels.test.ts`，11 个）
- 默认偏好 `'area'`；选项顺序 `['area','bar']`；
- `rankingChartType('area')==='area'`、`'bar'`/`undefined` → `'bar'`；
- 自然月偏好 → `'month'`，`days`/`undefined` → `'week'`；
- 标签：`en` → `Sep 26`、`zh` → `9月26日`、hourly → 时钟格式（不含"月"）、非法 ts → 原值；
- **spec 契约**：`type: chartType` + `stack: true` + `rankingChartType(getSavedChartPreferences()…`（VChart 需 canvas，无法渲染，见备注）。

## 三、门禁输出

**Go**
```
go build ./...        → OK
go test ./...         → exit 0（无 FAIL）
```
⚠️ 仓库里**没有 Makefile**（`find . -maxdepth 2 -name 'Makefile*'` 为空），`make test` 无法执行；
以上用等价的 `go test ./...` 全量跑完，**exit 0**。

**前端七条门**
typecheck **0 错误** ｜ lint **0 warnings / 0 errors** ｜ `bun test` **357 pass / 0 fail**（346 → +11）｜
format:check exit 0 ｜ build exit 0 ｜ **knip exit 0** ｜ i18n:check exit 0。

## 四、备注与边界
- **VChart 组件无法在 happy-dom 渲染**（需 canvas 2D，本机无 `canvas` 包且禁联网安装），
  因此"area + stack"用**源码契约断言**；标签/偏好映射是纯函数，做了真实单测。
- 后端 `label` 字段保留给旧客户端；前端自行格式化，两卡一致。
- `last-month` 偏好目前映射到 `month`（排行榜无"上月"档）—— 记录为已知取舍。
- 未 commit。
