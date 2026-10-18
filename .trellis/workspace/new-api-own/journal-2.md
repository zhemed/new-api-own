# Journal - new-api-own (Part 2)

> Continuation from `journal-1.md` (archived at ~2000 lines)
> Started: 2026-10-18

---



## Session 61: 默认范围支持本月/上月（加法式 kind），发 v0.0.26
<!-- trellis-session: v=2 fp=bb5a5fb6f22ac25f -->

**Date**: 2026-10-18
**Task**: 默认范围支持本月/上月（加法式 kind），发 v0.0.26
**Branch**: `main`

### Summary

用户发现默认范围只能是天数导致图表永远从 09-19 起。调查确认：偏好设置下拉只列天数预设、自然月只在筛选对话框，buildDefaultDashboardFilters 直接用 defaultTimeRangeDays 算滚动窗口。按用户选择实现：DashboardChartPreferences 加法式新增可选 defaultRangeKind（days/month-to-date/last-month，缺省 days），非法回落；保存归一化写 kind；buildDefaultDashboardFilters 统一走 resolveRangePreset 并按实际跨度推导粒度；偏好下拉改单控件两分组（天数 + 本月/上月，无需新 i18n）。新增 10 测试（含跨年本月、上月整月、往返、非法回落、29→30 迁移仍生效、组件级下拉与触发器显示），test 327→337，七条门全绿。发 v0.0.26（latest=v0.0.26、三资产齐）。

### Main Changes

- 承认并纠正上一轮取舍：默认范围必须能表达自然月，才符合用户'打开就从 1 号起'的诉求

### Git Commits

| Hash | Message |
|------|---------|
| `66e00ee` | feat(web): 默认范围支持「本月/上月」（加法式 kind 字段，老数据零迁移）[task:investigate-default-range-month] |

### Testing

- [OK] [OK] 337 pass / knip exit 0；本月跨年、上月整月、旧数据不变均有断言

### Status

[OK] **Completed**

### Next Steps

- 待用户决定是否升级实例到 v0.0.26 并在偏好里选「本月」


## Session 62: 按用户要求升级实例到 v0.0.26；并记下会话卫生做法
<!-- trellis-session: v=2 fp=0d02b28d0341d52d -->

**Date**: 2026-10-18
**Task**: 按用户要求升级实例到 v0.0.26；并记下会话卫生做法
**Branch**: `main`

### Summary

用户明确要求升级。执行：单次登录复用 token → POST /api/status/update/apply → 约 10 秒 v0.0.25→v0.0.26。复核：env 与 --log-opt 未变、容器 running 未重建、前端包 index.839a7ff985.js、releases/latest=v0.0.26、会话数 9（本轮 +1）。记录运维改进：以后每轮调完接口用 POST /api/auth/logout 撤销会话，避免 user_sessions 累积触碰上限；不把 token 落盘（凭据不持久化）。用户下一步在偏好设置把默认范围改为「本月」。

### Main Changes

- 无代码改动（仅实例升级）；补充会话卫生做法

### Git Commits

| Hash | Message |
|------|---------|
| `f908d69` | chore(task): archive 10-18-investigate-default-range-month |

### Testing

- [OK] [OK] 版本 v0.0.26 / env 与日志上限未变 / 会话数可控

### Status

[OK] **Completed**

### Next Steps

- 用户在偏好设置选「本月」并保存，验证图表从 1 号起


## Session 63: 默认范围只留上月/本月（上月在前）并兜底旧天数默认值，发 v0.0.27
<!-- trellis-session: v=2 fp=4707f7b09bf263c3 -->

**Date**: 2026-10-18
**Task**: 默认范围只留上月/本月（上月在前）并兜底旧天数默认值，发 v0.0.27
**Branch**: `main`

### Summary

用户要求默认范围去掉 1/7/14/30 天、并把上月排到上面。实现：NATURAL_MONTH_PRESETS 改为 [Last Month, This Month]（一处改动同时作用于偏好下拉与筛选月按钮）；偏好下拉不再生成天数项，筛选对话框天数按钮保留；旧值兜底——存档为 days 时对话框显示「本月」但不静默写盘，点保存才落盘 month-to-date，保存前图表仍按旧值渲染；解析层继续接受 days，29→30 迁移不回归。测试净增 2（选项顺序与无天数项、旧值展示与保存迁移、筛选月按钮顺序），test 337→339，七条门全绿。发 v0.0.27（latest=v0.0.27、三资产齐）。

### Main Changes

- 把'默认范围'限定为周期口径（自然月），天数档只保留在临时筛选里，从界面层面消除'30天滚动'误设的可能

### Git Commits

| Hash | Message |
|------|---------|
| `32ecf90` | feat(web): 默认范围只保留上月/本月（上月在前），旧天数默认值显示为本月并在保存时迁移 [task:evaluate-pref-range-cleanup] |

### Testing

- [OK] [OK] 339 pass / knip exit 0；解析层 days 兼容与迁移仍通过

### Status

[OK] **Completed**

### Next Steps

- 待用户决定是否升级实例到 v0.0.27；升级脚本顺带在调完接口后 logout 撤销会话（会话卫生）
