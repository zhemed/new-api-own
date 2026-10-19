# 排行榜跟随主页偏好 + 面积图默认排首位

## 用户报告（2026-10-19，三张截图）

1. **排行榜页**（今天/本周/本月/今年）：今天、本周正常 ✓；但
   - **本月** 不是从 1 号开始 ✗（轴上首条为 `Sep 26`）；
   - 日期是**英文** ✗（`Sep 26` / `Oct 4`），与整体中文界面不一致；
   - 只有**柱状图**，没有面积图 ✗；
2. **偏好设置**：默认范围=本月 ✓、默认消耗分布图=**面积图** ✓；
3. **模型调用分析**：消耗分布为面积图 ✓，右上角「偏好设置」按钮被用户框出。

## 用户要求

- 「把这个面积改成默认的，顺序对调变成第一个」→ **面积图设为默认**，切换顺序改为 **面积图 → 柱状图**；
- 「排行榜按照主页的偏好设置走」→ 排行榜的**范围**（本月=1 号→现在）、**图表类型**（面积图）都跟随已保存偏好；
- 顺带修 **日期本地化**（英文 → 当前语言）。

## 调查项（只读）

1. 排行榜 `今天/本周/本月/今年` 的时间范围如何计算（本月是否被当成 30 天）；
2. 排行榜日期格式化为何是英文（硬编码 locale / format 串）；
3. 排行榜图表类型是否写死柱状；是否读取 `getSavedChartPreferences`；
4. 消耗分布/模型调用图的视图切换顺序常量在哪。

## Acceptance Criteria

- [ ] 四条定位到文件:行号
- [ ] 本月=自然月 1 号 00:00 → 现在；日期随语言；图表类型跟随偏好；面积图默认且排首位
- [ ] 七条门全绿；发版并真机核对

## 执行结果（2026-10-19，v0.0.30）

### 后端（只动 month）

- `service/rankings.go:176`：`month` 档加 **`calendarMonth: true`**；`:227`/`:241` 在 calendar 模式下
  `start = 本机时区本月 1 号 00:00`、`end = now`；上一周期 = **上一个自然月整月**；
- `today` / `week` / `year` **未动**（用户认可现状）；API 的 `label` 字段保留（前端不再用它的英文格式）。

### 前端

- `dashboard/constants.ts:39` 默认 `consumptionDistributionChart` → **`'area'`**；
- `constants.ts:86-87` 选项顺序 → **`[area, bar]`**（面积图在首位）；
- 新增 `rankings/lib/format.ts: formatRankingBucketLabel(ts, locale, hourly)` —— `Intl.DateTimeFormat` 按界面语言出标签
  （zh: `9月26日` / en: `Sep 26`；today 档出时分；非法值回退原值）；
- 新增 `rankings/lib/chart-preference.ts`：`rankingChartType(pref)` / `defaultRankingPeriod(rangeKind)`；
- `models-section.tsx`：图表 `type` 取自偏好且保留 `stack: true`；标签改由 `point.ts` 生成本地化文本；
  `market-share-section.tsx` 同法；`rankings/index.tsx` 默认周期按偏好推导（自然月 → 本月）。

### 验证

- **Go**：`go build ./...` ✓、`go vet ./...` ✓（无输出）、`go test ./...` **exit 0，33 包全过**；
  另查明仓库的测试入口是小写 **`makefile`**（`make test` 可用 ✓，成员用大小写敏感的 find 才没找到 ✗）；
- **前端七条门**：typecheck 0 / lint 0-0 / **357 pass**（346→+11）/ build / knip exit 0 / format / i18n 全绿；
- **发布**：v0.0.30 镜像与 Release 成功；**新 Release 正文即 changelog**（新的 release.yml 首次生效 ✓）。

### 已知取舍

`last-month` 偏好暂映射到排行榜的 `month` 档（排行榜只有今天/本周/本月/今年四档）；需要第 5 档可另做。
