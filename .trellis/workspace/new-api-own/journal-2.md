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


## Session 64: 按用户要求升级实例到 v0.0.27，并 logout 撤销本次会话
<!-- trellis-session: v=2 fp=bb69a0a4cde49256 -->

**Date**: 2026-10-18
**Task**: 按用户要求升级实例到 v0.0.27，并 logout 撤销本次会话
**Branch**: `main`

### Summary

用户明确要求升级。执行：单次登录 → POST /api/status/update/apply → 约 10 秒 v0.0.26→v0.0.27（env 与 --log-opt 未变、容器 running 未重建、前端包 index.a20a140f8e.js、releases/latest=v0.0.27）。调完接口后 POST /api/user/auth/logout 返回 200，本次会话被撤销。注意：user_sessions 表行数仍会 +1（logout 保留历史行），判断是否堆积应看 status='active' 的计数而非总行数。

### Main Changes

- 无代码改动（实例升级）；会话卫生做法落地：每轮登录用完即 logout

### Git Commits

| Hash | Message |
|------|---------|
| `85e494b` | chore(task): archive 10-18-evaluate-pref-range-cleanup |

### Testing

- [OK] [OK] v0.0.27 / env 与日志上限未变 / logout HTTP 200

### Status

[OK] **Completed**

### Next Steps

- 用户在偏好设置确认默认范围只有上月/本月，并保存一次完成旧值迁移


## Session 65: 查清 logout 200 未撤销的根因（重启换签名密钥）并清理我的 7 条会话（12→5 active）
<!-- trellis-session: v=2 fp=6656dc2bc43c9cef -->

**Date**: 2026-10-19
**Task**: 查清 logout 200 未撤销的根因（重启换签名密钥）并清理我的 7 条会话（12→5 active）
**Branch**: `main`

### Summary

v0.0.27 升级后 logout 返回 200 但会话未撤销。调查：AuthLogout 的撤销分支要求 ParseAccessToken 成功，而实例未设 SESSION_SECRET，每次重启会换签名密钥 → 升级后旧 token 验签失败 → 走兜底只清 cookie 仍返回 200。危险度：上限按 status='active' 计（当初 50 撞顶），当时 11 条。按用户授权清理：先用新登录 token 的 logout 验证理论（返回 revoked_sid ✓），会话列表接口结构不符预期故改走 DB（已备份）：删除 active 且 UA 为 curl/% 或 %X11; Linux% 的 7 条，保留 5 条 Windows 浏览器会话 → active 12→5，服务与实例版本正常。沉淀升级轮会话卫生做法：A 登录→升级→B 登录→用 B 删 A（解 JWT 取 sid）→logout B，净增 0。

### Main Changes

- 记录两条结论：logout 的 200 不代表撤销（需 token 可解析）；判断会话堆积要看 status='active' 且可用 user_agent 区分自己的 curl 会话与用户浏览器会话

### Git Commits

| Hash | Message |
|------|---------|
| `29d5cf9` | chore(task): archive 10-18-upgrade-instance-v0027 |

### Testing

- [OK] [OK] active 12→5；/api/status 200；实例 v0.0.27；new-token logout 返回 revoked_sid

### Status

[OK] **Completed**

### Next Steps

- 下次升级按新的会话卫生流程执行；如需彻底避免换密钥可另行设置 SESSION_SECRET（需用户自行操作）


## Session 66: 修复立即升级弹窗提前关闭（重启中在弹窗内显示进度），发 v0.0.28
<!-- trellis-session: v=2 fp=025f4409fe6a9683 -->

**Date**: 2026-10-19
**Task**: 修复立即升级弹窗提前关闭（重启中在弹窗内显示进度），发 v0.0.28
**Branch**: `main`

### Summary

用户先报'卡住'，随后更正为：能升级、但确认弹窗一点就关、进度不可见、随后页面突然刷新。定位：update-checker-section 的 ok 分支立刻 setConfirmOpen(false) 再把 phase 设为 restarting，进度渲染在页面里而弹窗已关。修复：ok 后不关弹窗，改为弹窗三态（idle/restarting/timed-out），restarting 时弹窗内显示进度、两按钮禁用并拒绝 Esc/遮罩关闭；超时态提示手动刷新且 Close 不刷新；成功由 watcher reload、弹窗随页面消失；页面常驻提示只保留超时一种。新增 4 个 i18n 键 7 语言、组件测试 +2，test 339→341，七条门全绿。发 v0.0.28（latest=v0.0.28、三资产齐）。真机需在下一版本发布时验证。

### Main Changes

- 把'进度可见性'放回用户看得见的位置（弹窗内），并顺手补上'重启中不可关闭'的约束

### Git Commits

| Hash | Message |
|------|---------|
| `8759472` | fix(web): 立即升级的确认弹窗不再提前关闭——重启中在弹窗内显示进度并禁用按钮，超时改为手动刷新提示 [task:fix-apply-update-hang] |

### Testing

- [OK] [OK] 341 pass / knip exit 0；成功不关弹窗、超时关闭不刷新、失败不 watch 均有断言

### Status

[OK] **Completed**

### Next Steps

- 待用户决定是否升级到 v0.0.28（下一个版本发布时可端到端看到新弹窗行为）


## Session 67: 升级到 v0.0.28 并盯测：RestartCount 0（原地重执行）、日志无异常、会话净增 0
<!-- trellis-session: v=2 fp=28c41e2411b36e99 -->

**Date**: 2026-10-19
**Task**: 升级到 v0.0.28 并盯测：RestartCount 0（原地重执行）、日志无异常、会话净增 0
**Branch**: `main`

### Summary

用户要求升级并盯测。执行 apply → 首次采样即 v0.0.28。盯测证据：容器 running 且 RestartCount 0、StartedAt 未变（证明原地重执行而非容器重启）；docker logs 无 panic/error，重启后正常 200；四条 LOG_* 与 --log-opt 未变；前端包 index.fc5a125e62.js；latest=v0.0.28。会话收尾：DELETE /api/user/self/sessions/<JWT-sid> 返回 404（接口 id 语义与 JWT sid 不一致），改按 UA 精确删除本轮 curl 会话（保留 5 条浏览器会话）→ active 6→5，净增 0；logout（新 token）返回 revoked_sid，再次验证 token 有效才真正撤销。过程失误自报：监控脚本 case 语法错误导致采样循环未执行（apply 已成功，未重复触发）。

### Main Changes

- 沉淀两条可复用结论：RestartCount + StartedAt 可证明'in-place exec'；会话接口的 sid 与 JWT sid 不同源，清理应走 UA 精确匹配

### Git Commits

| Hash | Message |
|------|---------|
| `474945c` | chore(task): archive 10-19-update-dialog-progress |

### Testing

- [OK] [OK] v0.0.28 / RestartCount 0 / 日志无异常 / active 6→5

### Status

[OK] **Completed**

### Next Steps

- 下一版本发布时，用户从面板点『立即升级』可验收新弹窗（不消失+进度+自动刷新）


## Session 68: 图表标题『总计』数值提亮加粗（两处），发 v0.0.29
<!-- trellis-session: v=2 fp=b8a734d5bd34abcc -->

**Date**: 2026-10-19
**Task**: 图表标题『总计』数值提亮加粗（两处），发 v0.0.29
**Branch**: `main`

### Summary

用户反馈消耗分布标题里 总计:1.68M 的数字颜色太浅。定位：两处图表标题把标签与数值放在同一个 text-muted-foreground text-xs span 里，主指标被当辅助信息弱化。按用户选择两处都改：数值单独包 text-foreground + font-medium + tabular-nums，标签保持 muted、字号与布局不变；未动 tooltip/排行榜的汇总（辅助值保留弱化）。新增 5 条结构契约断言（成员说明因 vchart 需要真实 canvas 且禁止联网安装 canvas 包，断言止于源码结构层）。test 341→346，七条门全绿。发 v0.0.29（latest=v0.0.29、三资产齐）。

### Main Changes

- 确立视觉层级：主指标（标题旁总计）用正文色+加粗+等宽数字，辅助汇总保持弱化色

### Git Commits

| Hash | Message |
|------|---------|
| `1822599` | style(web): 图表标题的总计数值改用正文色+加粗+等宽数字（标签保持弱化）[task:investigate-chart-total-color] |

### Testing

- [OK] [OK] 346 pass / knip exit 0；两处类名与标签弱化均有断言

### Status

[OK] **Completed**

### Next Steps

- 待用户决定是否升级实例到 v0.0.29 查看效果


## Session 69: 升级到 v0.0.29：+26s 完成、0 次中断采样、RestartCount 0；我的会话清零（active 6 全是用户浏览器）
<!-- trellis-session: v=2 fp=c5d15594a323fccc -->

**Date**: 2026-10-19
**Task**: 升级到 v0.0.29：+26s 完成、0 次中断采样、RestartCount 0；我的会话清零（active 6 全是用户浏览器）
**Branch**: `main`

### Summary

用户要求升级试用。执行 apply → +24s 仍 v0.0.28、+26s 变 v0.0.29，累计中断采样 0 次（切换窗口小于 2s 采样间隔），容器 RestartCount 0 / StartedAt 未变证明原地重执行，日志无 panic，env 与 --log-opt 未变，前端包 index.b907d24f59.js，latest=v0.0.29。会话：清理后 curl 类 active 会话为 0，当前 active 6 条全部是 Windows 浏览器会话（用户期间又登录一次），我未造成堆积。自报：查 latest 时先写错路径（release/latest→404），已用正确路径复核。

### Main Changes

- 无代码改动（实例升级）

### Git Commits

| Hash | Message |
|------|---------|
| `f018560` | chore(task): archive 10-19-total-value-emphasis |

### Testing

- [OK] [OK] v0.0.29 / +26s / 0 中断采样 / RestartCount 0 / 日志无异常 / 我的会话 0 条

### Status

[OK] **Completed**

### Next Steps

- 请用户刷新浏览器查看『总计』数值的新视觉；下一版本发布时可用面板点更新验收新弹窗


## Session 70: Release 正文改为 tag 注解 changelog，并回填全部 28 个历史 Release
<!-- trellis-session: v=2 fp=78fe2bbfcebea421 -->

**Date**: 2026-10-19
**Task**: Release 正文改为 tag 注解 changelog，并回填全部 28 个历史 Release
**Branch**: `main`

### Summary

用户指出 Release 正文全是部署说明（README 已有），要求改成更新日志。定位 release.yml:210-221 硬编码 BODY；发现 28/28 个 tag 注解正文本身就是 changelog。改造：BODY 取 git tag -l --format='%(contents:body)'，空时回退提交标题；保留附件上传与契约断言、latest/prerelease 逻辑；bash -n 校验 run 块通过、两个仓库门禁通过。回填：对 28 个已有 Release 执行 gh release edit --notes <tag 注解正文>，28 成功 0 失败；抽验三版正文均为 changelog；latest 仍 v0.0.29、附件三件套完好。工作流对新 tag 生效。

### Main Changes

- 确立 Release 正文的唯一来源 = 注解 tag 正文，部署说明只留在 README（避免两处重复且各自过期）

### Git Commits

| Hash | Message |
|------|---------|
| `4e22794` | chore(task): archive 10-19-upgrade-instance-v0029 |

### Testing

- [OK] [OK] bash -n 全部 run 块 exit 0；两道门禁通过；回填 28/28；latest 与附件未变

### Status

[OK] **Completed**

### Next Steps

- 下一个 tag 发布时核对 Release 正文是否为 changelog


## Session 71: 排行榜本月改自然月+日期本地化+跟随偏好；消耗分布默认面积图，发 v0.0.30
<!-- trellis-session: v=2 fp=e30e4b3ac3548a58 -->

**Date**: 2026-10-19
**Task**: 排行榜本月改自然月+日期本地化+跟随偏好；消耗分布默认面积图，发 v0.0.30
**Branch**: `main`

### Summary

用户报排行榜：本月不是1号、日期英文、只有柱状图，并要求面积图设为默认排首位、排行榜跟随主页偏好。实现：后端 rankings.go 给 month 加 calendarMonth（本月1号00:00→now，上一周期=上月整月，today/week/year 不动，保留 label 字段）；前端新增 formatRankingBucketLabel 用 Intl 按界面语言出标签（zh 9月26日 / en Sep 26），新增 rankingChartType/defaultRankingPeriod 纯函数，models-section 图表类型取自偏好且保留 stack，market-share 同法，rankings/index 默认周期按偏好；消耗分布默认改 area 且选项顺序 [area, bar]。测试：Go 4 个（本月/跨年/上一周期/其它档仍滚动，go test ./... exit 0、33 包全过）+ 前端 11 个，bun test 346→357，七条门全绿。顺带核实仓库测试入口是小写 makefile（make test 可用），并确认新 Release 正文已是 changelog。发 v0.0.30。

### Main Changes

- 统一口径：排行榜也遵循'自然月=1号起'与界面语言；图表类型/默认周期以已保存偏好为单一来源

### Git Commits

| Hash | Message |
|------|---------|
| `16f6b71` | feat(rankings): 本月改自然月、日期随界面语言、图表类型与默认周期跟随偏好；消耗分布默认面积图并排首位 [task:rankings-follow-preferences] |

### Testing

- [OK] [OK] go build/vet/test 全绿；前端 357 pass / knip exit 0；v0.0.30 镜像与 Release 成功且正文为 changelog

### Status

[OK] **Completed**

### Next Steps

- 待用户决定是否升级实例到 v0.0.30 并核对排行榜本月从 1 号起、日期为中文、图表为面积图


## Session 72: 升级到 v0.0.30：+20s 完成、0 次中断采样、会话净增 0
<!-- trellis-session: v=2 fp=0dd0d490e79341d6 -->

**Date**: 2026-10-19
**Task**: 升级到 v0.0.30：+20s 完成、0 次中断采样、会话净增 0
**Branch**: `main`

### Summary

用户要求升级。执行 apply → +18s 仍 v0.0.29、+20s 变 v0.0.30，累计中断采样 0 次；容器 RestartCount 0、StartedAt 未变（原地重执行），日志无 panic，四条 LOG_* 与 --log-opt 未变，前端包 index.7110c132c6.js。会话：清理本轮 curl 会话后 active 7→7（全部是用户浏览器会话），净增 0。

### Main Changes

- 无代码改动（实例升级）

### Git Commits

| Hash | Message |
|------|---------|
| `ee2174a` | chore(task): archive 10-19-rankings-natural-month-i18n |

### Testing

- [OK] [OK] v0.0.30 / +20s / 0 中断 / RestartCount 0 / 会话净增 0

### Status

[OK] **Completed**

### Next Steps

- 请用户刷新浏览器核对：排行榜本月从 1 号起、日期为中文、图形为面积图；消耗分布默认面积图且切换顺序为 面积图→柱状图


## Session 73: 修复 zhCN 语言码导致的 500（三层防御+真实语言码回归）；年挡位改自然年；首次用无头浏览器真机验证页面
<!-- trellis-session: v=2 fp=0d0c8fa76e1d12cc -->

**Date**: 2026-10-19
**Task**: 修复 zhCN 语言码导致的 500（三层防御+真实语言码回归）；年挡位改自然年；首次用无头浏览器真机验证页面
**Branch**: `main`

### Summary

用户报 v0.0.30 页面 500。定位：排行榜把 i18n.language 原值（zhCN/zhTW/enUS）传给 Intl.DateTimeFormat → RangeError invalid language tag → 渲染抛错到全局错误页；后端日志无 panic、/api/rankings 四档均 200。修复三层：调用点改 toIntlLocale、函数内候选链 try/catch + 手工格式兜底、回归测试改用真实语言码（zhCN/enUS/zhTW/xx-invalid/undefined）+ i18next 端到端。另按要求把 year 改成自然年（今年1/1→now，上一周期=上一年整年，桶粒度不变）。Go make test exit 0（39 包）、前端 359 pass、七条门全绿；发 v0.0.31。Lead 首次用无头浏览器 fetch 登录（cookie 会话）真机验证：/rankings 无 500、日期中文；?period=month 本月高亮/说明正确/中文日期/面积图/总计1.68M；?period=year 无 500。沉淀：UI 改动必须加页面级真机校验，测试不得用假 locale。

### Main Changes

- 两条硬经验：①引入 Intl 必须用 toIntlLocale 且不得假设 locale 合法；②任何 UI 改动都要有'真机页面'验证，不能只靠单测

### Git Commits

| Hash | Message |
|------|---------|
| `404d72d` | fix(rankings): 修复 zhCN 语言码导致 Intl 抛错引发 500（三层防御 + 真实语言码回归）；年挡位改自然年 [task:fix-rankings-500-and-year-range] |

### Testing

- [OK] [OK] Go make test 39 包 ok；前端 359 pass；真机三档页面均无 500 且 label 中文、面积图生效

### Status

[OK] **Completed**

### Next Steps

- 如需，可把无头浏览器 fetch 登录封装成标准验证步骤（当前为临时脚本）


## Session 74: 核实并加固月/年视图刻度密度：确定性上限≤10、年档按月、修正桶起点与过期文案（v0.0.32/v0.0.33）
<!-- trellis-session: v=2 fp=2bcffcdcd869c346 -->

**Date**: 2026-10-19
**Task**: 核实并加固月/年视图刻度密度：确定性上限≤10、年档按月、修正桶起点与过期文案（v0.0.32/v0.0.33）
**Branch**: `main`

### Summary

用户担心本月从1号起30个点会密密麻麻。核实：月档按天最多30桶、年档原为按周(52点)、标签只有 autoHide/autoLimit 不可靠。按用户选择：①年档改30天桶(12-13点)；②x 轴标签改确定性抽稀(复用 axisLabelStride，stride=ceil(n/10)，首尾必留，dataFilter 实现，1-60 桶全枚举测试)；③顺手修 epoch 对齐导致首桶早于范围起点的问题(clampRankingBucketsToRange)。真机验证 v0.0.32：本月无500、中文日期、面积图，接口实测仅3个桶有数据故显示3个标签，满月30桶的≤10由枚举测试保证。另发现并修正 4+3 条过期周期描述（week/year/market-share），v0.0.33 发布。Go make test 39包ok、前端 366 pass、七条门全绿。

### Main Changes

- 确立'标签密度必须确定性可证'的原则：不依赖第三方库 autoLimit 的未知上限，用自己的 stride + 首尾必留 + 枚举测试

### Git Commits

| Hash | Message |
|------|---------|
| `4c0c6a4` | docs(i18n): 修正排行榜四个周期的描述文案（与按天/按 30 天/自然月/自然年分桶一致）[task:verify-rankings-density] |

### Testing

- [OK] [OK] 30桶→10标签、12桶→7标签、1-60桶全枚举≤10；两版均真机无500

### Status

[OK] **Completed**

### Next Steps

- 无需后续；若真实流量覆盖整月，可再截图确认观感


## Session 75: 交付刻度密度结论并升级实例到 v0.0.33（真机核对周期文案）
<!-- trellis-session: v=2 fp=9cad89431ead41e0 -->

**Date**: 2026-10-19
**Task**: 交付刻度密度结论并升级实例到 v0.0.33（真机核对周期文案）
**Branch**: `main`

### Summary

回答用户'本月30天会不会密密麻麻'：v0.0.32 已把 x 轴标签改为确定性抽稀（stride=ceil(n/10)、首尾必留、1-60 桶全枚举测试），30 桶只显示 10 个；真机当前仅 3 个桶有数据故图上 3 标签。同版本把年档改 30 天桶并修正桶起点早于范围的问题。v0.0.33 修正四个周期描述与 market-share 三条文案。按用户选择把实例升级到 v0.0.33：apply 后 6 秒内切换、RestartCount 0、日志正常、前端包 index.3266cd733c.js；真机核对 week/year 文案已正确。清理本轮 curl 与无头浏览器会话，active 8 条均为用户浏览器会话。

### Main Changes

- 无新增代码（本轮为交付与升级）

### Git Commits

| Hash | Message |
|------|---------|
| `f8d0291` | chore(task): archive 10-19-verify-rankings-density |

### Testing

- [OK] [OK] 真机 week/year 文案正确；升级零中断、容器未重建

### Status

[OK] **Completed**

### Next Steps

- 无


## Session 76: 市场份额卡片跟随偏好改面积图（100% 堆叠不变），发 v0.0.34 并真机验证
<!-- trellis-session: v=2 fp=84bf4201aa160cd1 -->

**Date**: 2026-10-19
**Task**: 市场份额卡片跟随偏好改面积图（100% 堆叠不变），发 v0.0.34 并真机验证
**Branch**: `main`

### Summary

用户要求市场份额卡片与热门模型一致也变面积图。实现：新增 rankingChartShape(pref) 统一返回 {type, stack:true, bar 时带 paddingInner:0.12}，market-share-section 用它替换写死 type:'bar'；因该图表在数据层已把 share 归一化到 0–1（y 轴 min0/max1、标签*100%），换 type 不影响 100% 语义；标签抽稀共用 rankingLabelFilter。测试 +3（area/bar 形态与 100% 契约），bun test 366→369，七条门全绿；发 v0.0.34 并升级实例。真机验证：week 单桶为点、month 多桶为填充面积@100%、中文日期，截图留证。已知边界：同页切换偏好不热更新。

### Main Changes

- 两处排行榜图表统一走 rankingChartShape，偏好成为单一来源

### Git Commits

| Hash | Message |
|------|---------|
| `c4f61de` | feat(rankings): 市场份额卡片也跟随偏好（面积/柱状 100% 堆叠），共用 rankingChartShape [task:market-share-area-chart] |

### Testing

- [OK] [OK] 369 pass / knip 0；真机 month 面积图@100% 与 week 单点均正常

### Status

[OK] **Completed**

### Next Steps

- 无


## Session 77: 保存偏好后图表即时生效（useSyncExternalStore + 稳定快照），发 v0.0.35 并真机点保存验证
<!-- trellis-session: v=2 fp=207882cf25f10ff1 -->

**Date**: 2026-10-19
**Task**: 保存偏好后图表即时生效（useSyncExternalStore + 稳定快照），发 v0.0.35 并真机点保存验证
**Branch**: `main`

### Summary

用户要求把偏好热更新绑到『保存偏好设置』按钮。实现：新增 chart-preferences-events（listener+emit，含 storage 跨标签）与 use-chart-preferences（useSyncExternalStore，用 localStorage 原始串做快照身份避免无限重渲染）；saveChartPreferences 写盘后广播；5 处读取点（dashboard、排行榜默认周期-仅挂载读、热门模型、市场份额、barrel）改为订阅。测试 +4（通知恰一次、快照引用稳定、保存后类型切换、周期不被重置），bun test 369→373，七条门全绿；发 v0.0.35 并升级实例。Lead 真机验证：打开 /dashboard/models → 选柱状图 → 点保存偏好设置 → 页面不刷新即见 localStorage 变 bar 且切换按钮立刻选中柱状图；随后改回面积图恢复原设置。清理会话。

### Main Changes

- 偏好成为可订阅的单一来源：保存即全局生效，默认周期仍只在挂载时读取，不覆盖手动切换

### Git Commits

| Hash | Message |
|------|---------|
| `e19547e` | feat(web): 偏好保存后图表即时生效（useSyncExternalStore + 稳定快照），覆盖排行榜与模型分析 [task:prefs-live-update] |

### Testing

- [OK] [OK] 373 pass / knip 0；真机点保存后无刷新即时生效（bar→area 双向验证）

### Status

[OK] **Completed**

### Next Steps

- 无


## Session 78: 全站美元改人民币：实例切 CUSTOM ¥/1.0、默认值与硬编码、i18n 两轮去美元（v0.0.36/v0.0.37）
<!-- trellis-session: v=2 fp=f92a5cae76a2ace9 -->

**Date**: 2026-10-22
**Task**: 全站美元改人民币：实例切 CUSTOM ¥/1.0、默认值与硬编码、i18n 两轮去美元（v0.0.36/v0.0.37）
**Branch**: `main`

### Summary

用户要求把美元改成人民币（￥），选定口径=只换符号、数值不变。调查（两路只读）：前端 43 文件/156 处走统一货币层，8 文件 13 处硬编码绕过；后端内置 quota_display_type(USD/CNY/TOKENS/CUSTOM)+自定义符号+汇率，经 /api/status 下发且 logger 的日志/错误/通知跟随；CNY 模式会乘 USDExchangeRate=7.3 与口径不符，唯一可行=CUSTOM+¥+1.0。执行：①实例 PUT /api/option/ 设三键并落库复核，真机确认 dashboard ¥1.15、pricing ¥0.3/¥15 且无 $；旧键 DisplayInCurrencyEnabled 不在库中（否则会强制改回 USD）。②v0.0.36：Go 默认值改 CUSTOM/¥/1.0、logger 统一复用 GetCurrencySymbol/GetUsdToCurrencyRate 真源（USD/TOKENS 输出字节级不变）、前端 8 处硬编码改走货币层并新增 getCurrencySymbol()、i18n 22 键×7 语言。③v0.0.37：10 个机制类键去美元字样改“基准货币”口径，API 值不变。门禁：Go vet/build/relaykit/make test(40 包) 全绿；前端七条门全绿 383 pass。成员曾因裸替换一度改坏 32 文件 228 处 USD 子串，已完全还原并经 Lead 独立复核（工作区仅 9 文件、关键标识符完好）。

### Main Changes

- 确立口径：显示货币由设置驱动（CUSTOM+符号+汇率），代码不得写死符号；日志/错误/通知与前端同源

### Git Commits

| Hash | Message |
|------|---------|
| `3e9825c` | i18n(currency): 机制类文案去美元字样（改用基准货币口径），保留 API 值不变 [task:switch-currency-to-cny] |

### Testing

- [OK] [OK] 真机 ¥ 且无 $；Go 40 包 ok；前端 383 pass；i18n 零美元命中

### Status

[OK] **Completed**

### Next Steps

- 待用户裁定：显示模式 USD 分支的 $/USD 标签、Waffo 网关币种字段；并确认是否升级实例到 v0.0.37


## Session 79: 基准货币模式也显示 ¥（useSymbol 纯显示开关）+ 补 15 处模板字面量美元，发 v0.0.38
<!-- trellis-session: v=2 fp=4bcf9957cbbb74f7 -->

**Date**: 2026-10-22
**Task**: 基准货币模式也显示 ¥（useSymbol 纯显示开关）+ 补 15 处模板字面量美元，发 v0.0.38
**Branch**: `main`

### Summary

用户裁定基准货币（API 值 USD）模式也要显示人民币符号，Waffo 网关币种按高风险不改。实现：发现符号由 Intl.NumberFormat 的 currencyCode 生成、改 symbol 无效，故新增纯显示开关 useSymbol（currency.ts:120），置位时与 CUSTOM 共用 formatWithSymbol；USD 分支 symbol 为人民币符号 + useSymbol:true，kind/currencyCode:'USD'/exchangeRate:1 未动（永不换算、API 契约不变）；getCurrencyLabel 在基准货币与 Token 模式返回人民币符号。成员同时补掉上一轮盘点盲区：15 处模板字面量形式硬编码美元（管理端计价快照/单价核心）。测试 +3（386 pass）；七条门与契约复核全绿；发 v0.0.38（latest=v0.0.38）。遗留待裁定：CNY 模式空格不一致、Creem 美元商品显示、是否加防回归脚本。

### Main Changes

- 确立显示符号与计量币种解耦：useSymbol 只影响渲染，不动 currencyCode/exchangeRate，避免为改显示而影响换算与 API 契约

### Git Commits

| Hash | Message |
|------|---------|
| `5a57ee5` | feat(web): 基准货币模式也显示 ¥（纯显示开关 useSymbol），并补掉 15 处模板字面量美元 [task:deliver-cny-and-decide] |

### Testing

- [OK] [OK] 386 pass；currencyCode 仍为 USD、exchangeRate 仍为 1 有断言锁定；七条门全绿；v0.0.38 已发布

### Status

[OK] **Completed**

### Next Steps

- 待用户裁定三项遗留；并按选择升级实例到 v0.0.38


## Session 80: 统一金额格式（三模式一致+两处自研拼接收口）并新增货币硬编码守则，发 v0.0.39
<!-- trellis-session: v=2 fp=69105dfe43201149 -->

**Date**: 2026-10-22
**Task**: 统一金额格式（三模式一致+两处自研拼接收口）并新增货币硬编码守则，发 v0.0.39
**Branch**: `main`

### Summary

用户裁定只统一空格、Creem 不动、加防回归自检。执行：①CNY 分支改走 formatWithSymbol（¥70 到 ¥ 70），与基准货币/CUSTOM 三模式格式一致，换算率与位数语义未动；②收口两处自研拼接——计费拆解组件删掉自研 symbol/rate switch 改用 formatBillingCurrencyFromUSD（保留 4 位有效数字，不再补零、大额带千分位），图表 renderQuotaCompat 只加空格；③新增 scripts/check-money-format.mjs 守则（去注释状态机 + 三分类扫描 + 白名单），接 make 目标与 CI，当前 955 文件 OK、7 个人造反例全抓到、白名单零误报。Lead 实测澄清争议：守则确实跳过注释，命中的是字符串字面量（正确行为）。测试 386 到 393；七条门 + 守则 + 两道仓库门禁全绿；发 v0.0.39。

### Main Changes

- 全站金额格式与符号来源收敛为单一货币层；新增机械守则防止硬编码货币回归

### Git Commits

| Hash | Message |
|------|---------|
| `4859a62` | feat(currency): 统一金额格式（CNY 与图表/拆解均改为符号+空格）并新增货币硬编码防回归守则 [task:deliver-base-currency-and-decide] |

### Testing

- [OK] [OK] 393 pass；make check-money-format exit 0；守则反例 7/7 命中且白名单零误报

### Status

[OK] **Completed**

### Next Steps

- 待用户决定是否升级实例到 v0.0.39


## Session 81: 升级实例到 v0.0.39 并真机复核三个页面无美元
<!-- trellis-session: v=2 fp=6548dc61c90bd505 -->

**Date**: 2026-10-22
**Task**: 升级实例到 v0.0.39 并真机复核三个页面无美元
**Branch**: `main`

### Summary

用户选择升到 v0.0.39 且计费拆解小数位保持现状。执行：apply 后约 6 秒切到 v0.0.39，容器 RestartCount 0、日志无异常、币种设置 CUSTOM/¥/1 重启后仍在。真机复核 /pricing（¥ 0.3、¥ 15，无美元符号与 USD 字样）、/dashboard/models（¥ 1.15）、/keys（无金额，无美元）均正常。清理本轮 curl 与无头浏览器会话。

### Main Changes

- 无代码改动（实例升级与复核）

### Git Commits

| Hash | Message |
|------|---------|
| `3c5b5f9` | chore(task): archive 10-22-deliver-base-currency-and-decide |

### Testing

- [OK] [OK] v0.0.39；三页面零美元命中；设置保持

### Status

[OK] **Completed**

### Next Steps

- 无；如需固定 4 位小数或收口 Creem 显示可另开任务


## Session 82: 默认模型目录裁到只剩 deepseek 家族 + 官方人民币峰谷价（实例+代码），发 v0.0.40
<!-- trellis-session: v=2 fp=38982a6e37fd4d23 -->

**Date**: 2026-10-22
**Task**: 默认模型目录裁到只剩 deepseek 家族 + 官方人民币峰谷价（实例+代码），发 v0.0.40
**Branch**: `main`

### Summary

用户要求：①284 个默认模型除 deepseek-flash 外全部彻底移除；②查 DeepSeek-Flash 官方人民币定价（含低谷/高峰）并默认写入。三路只读调查结论：284 不是数据库模型表（models 0 行、abilities 只有 deepseek-flash、渠道只绑它、/api/pricing 只返回 1 个），而是代码内置定价目录（defaultModelRatio 239 + defaultModelPrice 31 等）+ 实例 option 覆盖；前端无写死清单、i18n 无关、删除不级联。官方价（2026-10-22）：低谷 缓存命中 ¥0.02/未命中 ¥1.00/输出 ¥4.00 每 1M tokens，高峰全部 ×2，高峰时段=北京时间周一至周五 09:00-12:00/14:00-18:00 不含节假日；比率体系不支持分时，唯一内建时段感知是 billingexpr。执行：①实例备份 + 写入 5 个 option（ModelRatio 240 到 1、CompletionRatio/CacheRatio、billing_mode/expr 峰谷表达式）；②真实请求测试打到上游但上游失败（与本次无关），顺带发现渠道上游是第三方网关而非官方；临时令牌用完即删；③代码 v0.0.40 把默认目录裁到 deepseek 家族（保留引擎注入音频默认与 MJ/Suno 计费键，均有硬依赖依据）、deepseek-flash 默认改官方低谷价、修 MAINTENANCE 低估 7.3 倍的旧公式、新增 4 测试；go vet/build/relaykit/make test(41 包)全绿。成员自报一次工作区外误写（已删、无外发）并已核实。

### Main Changes

- 确立：模型目录的唯一来源是代码默认表 + 实例 option，渠道能力表与定价目录是两件事；分时定价用 billingexpr

### Git Commits

| Hash | Message |
|------|---------|
| `5ce9703` | feat(pricing): 默认模型目录裁到只剩 deepseek 家族 + 按官方人民币低谷价配置；修文档换算公式 [task:deepseek-only-and-pricing] |

### Testing

- [OK] [OK] make test 41 包 ok；实例 option 落库复核；临时令牌与备份状态可查

### Status

[OK] **Completed**

### Next Steps

- 待用户决定：是否升级到 v0.0.40、是否再删 5 个音频默认与 3 个别名、渠道上游失败需排查


## Session 83: 升级到 v0.0.40 并复核（目录裁剪与峰谷定价生效，option 未被重启覆盖）
<!-- trellis-session: v=2 fp=5cfcf027711a09de -->

**Date**: 2026-10-22
**Task**: 升级到 v0.0.40 并复核（目录裁剪与峰谷定价生效，option 未被重启覆盖）
**Branch**: `main`

### Summary

用户选择：升到 v0.0.40、保留目录里 8 个非 deepseek 键（5 音频默认+3 别名）、上游报错系其自身忘记更新 API 故归档不排查。执行：apply 后约 6 秒切到 v0.0.40，RestartCount 0；重启后复查 option 未被覆盖——ModelRatio/CompletionRatio/CacheRatio 各 1 条且为官方低谷价（0.5/4/0.02），billing_setting.billing_mode 为 deepseek-flash:tiered_expr，币种仍 CUSTOM+人民币符号。清理本轮 curl 与无头浏览器会话。

### Main Changes

- 无代码改动（实例升级与复核）

### Git Commits

| Hash | Message |
|------|---------|
| `f5a210b` | chore(task): archive 10-22-deepseek-only-and-pricing |

### Testing

- [OK] [OK] v0.0.40；option 重启后保持；币种与峰谷表达式生效

### Status

[OK] **Completed**

### Next Steps

- 用户可自行在面板核对模型定价表条目；如需进一步裁剪音频默认可另开任务


## Session 84: 查明模型定价表 271 的来源（option 并集+进程内存）并修正自己的诊断命令错误
<!-- trellis-session: v=2 fp=fdbb5cb4bd3f9183 -->

**Date**: 2026-10-22
**Task**: 查明模型定价表 271 的来源（option 并集+进程内存）并修正自己的诊断命令错误
**Branch**: `main`

### Summary

用户报告 v0.0.40 模型定价表仍显示 271。调查（成员只读链路 + Lead 接口实测）：表行=8 个 JSON option 键的键名并集，通道为 RatioSettingsCard 到 GET /api/option/，而后端读的是进程内存 OptionMap（仅启动或走 API 保存时刷新），前端另有 5 分钟 staleTime；实证排除前端内置目录（下载 100 chunk/49MB，旧模型名 0 命中）、代码默认（9+20 条）与 ratio_config（403 因 ExposeRatioEnabled=false）。Lead 自报错误：上一轮把 /api/option/ 的 data 数组当对象用 to_entries 过滤，误判'没有 ModelRatio'；改用正确 jq 后实测内存并集=25（ModelPrice 20 + AudioCompletionRatio 4 + 三个 1 条），与后端 CompletionRatioMeta=25 一致，说明裁剪已生效、截图为改配置前的旧页面。用户裁定 20 个功能计费键保留。

### Main Changes

- 沉淀两条经验：/api/option/ 返回 data 为数组且读进程内存（改库不重启不生效）；诊断前先打印 data 的类型，别假设结构

### Git Commits

| Hash | Message |
|------|---------|
| `131bf03` | chore(task): archive 10-22-deliver-model-trim-and-decide |

### Testing

- [OK] [OK] 接口并集=25 与 CompletionRatioMeta=25 一致；旧模型名在前端产物 0 命中

### Status

[OK] **Completed**

### Next Steps

- 用户强刷页面确认总计变为 25；如仍为 271 再查镜像 digest 是否与 v0.0.40 一致


## Session 85: 澄清定价表 25 行来源（产品默认非测试残留）并按裁定清空功能键与音频键，表格只剩 deepseek-flash
<!-- trellis-session: v=2 fp=f440747832370ef4 -->

**Date**: 2026-10-22
**Task**: 澄清定价表 25 行来源（产品默认非测试残留）并按裁定清空功能键与音频键，表格只剩 deepseek-flash
**Branch**: `main`

### Summary

用户看到表格 25 行、怀疑是测试残留。核实：DB 里只有我写的 3 个键（ModelRatio/CompletionRatio/CacheRatio 各 1 条），其余 24 行来自产品默认表（defaultModelPrice 20 条 MJ/Suno/换脸功能键 + defaultAudioRatio/AudioCompletionRatio 各 4 条音频键），不是测试残留；但如实认领副作用：DB ModelRatio 覆盖为 1 条后代码默认 9 条被整体替换，音频模型丢价显示未设。用户裁定两批都清，已写入 ModelPrice={}、AudioRatio={}、AudioCompletionRatio={} 并复核：接口内存 8 键并集=1，唯一模型名 deepseek-flash，DB 落库一致。代价（MJ/Suno 无法预扣费、未带 model 的语音请求报未配置计费）用户已知情。

### Main Changes

- 沉淀：/api/option/ 的定价键是『实例写入』与『代码默认播种』两来源的并集；清空某键即整体替换该键的运行时 map

### Git Commits

| Hash | Message |
|------|---------|
| `68d77ef` | chore(task): archive 10-22-investigate-pricing-table-271 |

### Testing

- [OK] [OK] 接口并集=1 且只剩 deepseek-flash；DB 六键落库复核一致

### Status

[OK] **Completed**

### Next Steps

- 可选：把代码默认表里的 ModelPrice 20 条与音频 4 条也清掉（影响新部署/重置比率），需用户确认


## Session 86: 代码默认表只保留 deepseek 家族（v0.0.41）并核实音频注入链路的最终行为
<!-- trellis-session: v=2 fp=184ba9dda82a2156 -->

**Date**: 2026-10-22
**Task**: 代码默认表只保留 deepseek 家族（v0.0.41）并核实音频注入链路的最终行为
**Branch**: `main`

### Summary

用户裁定把代码默认表的功能计费键与音频键也清掉。实现：defaultModelRatio 9 到 4（deepseek-flash+3 别名，移除 whisper-1 与 4 个 tts）、defaultModelPrice 20 到空、defaultAudioRatio/defaultAudioCompletionRatio 到空，变量保留并加产品口径注释；测试规模锁定改 4/0/4/4 并新增只含 deepseek 家族断言；make test 41 包 ok；发 v0.0.41。成员只读核实注入链路最终行为：未带 model 的音频请求在无渠道时 503、有渠道未配价时 400，不会静默走 37.5（仅 self-use 或显式接受未配置模型时兜底）。Lead 自报一次抽查脚本缺陷（非贪婪正则跨读下一个 map，误判五张表都含 deepseek），改用直接读文件核实代码正确。

### Main Changes

- 代码默认目录只保留 deepseek 家族：新部署与重置比率后同样只剩它；音频与功能动作计费改由运营在实例侧配置

### Git Commits

| Hash | Message |
|------|---------|
| `a600c08` | feat(pricing): 代码默认表只保留 deepseek 家族（清空功能计费键与音频默认）[task:deliver-pricing-cleanup-and-decide] |

### Testing

- [OK] [OK] make test 41 包 ok；直接读文件确认三张表为空 map；v0.0.41 已发布

### Status

[OK] **Completed**

### Next Steps

- 待用户决定是否升级实例到 v0.0.41；可选：把 distributor 的默认注入改为直接报错（需用户确认）


## Session 87: 升级到 v0.0.41 并复核：模型定价表只剩 deepseek-flash（数据库与接口双证）
<!-- trellis-session: v=2 fp=588c426d02684ef6 -->

**Date**: 2026-10-22
**Task**: 升级到 v0.0.41 并复核：模型定价表只剩 deepseek-flash（数据库与接口双证）
**Branch**: `main`

### Summary

用户选择升到 v0.0.41、音频注入语义保持现状。执行：apply 后约 6 秒切到 v0.0.41，容器 RestartCount 0。升级后双证复核——数据库：ModelRatio/CompletionRatio/CacheRatio 各 1 条（deepseek-flash）、ModelPrice/AudioRatio/AudioCompletionRatio 为 0 条，模型名并集=deepseek-flash；接口：8 个定价键并集=1、唯一名称 deepseek-flash；币种仍 CUSTOM+人民币符号。清理会话。

### Main Changes

- 无代码改动（实例升级与复核）

### Git Commits

| Hash | Message |
|------|---------|
| `f1b6aea` | chore(task): archive 10-22-deliver-pricing-cleanup-and-decide |

### Testing

- [OK] [OK] v0.0.41；数据库与接口双向确认表格只剩 1 行

### Status

[OK] **Completed**

### Next Steps

- 无；若日后要启用 TTS/MJ/Suno，在实例侧给对应键配价即可


## Session 88: 生产实例（x.miio.cc/zzds.de）定价目录对齐到只剩 deepseek-flash，并澄清两边差异根因
<!-- trellis-session: v=2 fp=c47a33a0c3ec6a3e -->

**Date**: 2026-10-22
**Task**: 生产实例（x.miio.cc/zzds.de）定价目录对齐到只剩 deepseek-flash，并澄清两边差异根因
**Branch**: `main`

### Summary

用户报告生产与本地模型定价/数量不一致。调查：x.miio.cc 解析到另一台机器（非本机），两边都是 v0.0.41；生产库在 /root/new-api/data/one-api.db，改前 ModelRatio 4 条、CacheRatio 76 条、CompletionRatio 7 条（表格多出行数主因是 CacheRatio 旧模型），币种走代码默认。用户授权生产也对齐：先备份（bak-align-20261022-175404），再通过生产本机 PUT /api/option/ 写入 9 个键（3 个 deepseek 定价 + 3 个空表 + CUSTOM/¥/1.0），复核 /api/status 与库并集=1（deepseek-flash）、abilities 只有它，清理本次 curl 会话保留用户 19 条浏览器会话。SSH 首连报主机指纹变更，经用户确认重装后才 ssh-keygen -R 重连；已知本机/.ssh 操作与私有地址不入记录。如实记录两处 Lead 失误：探测角色时误打印 access_token（已注销该会话）、字符串型 option 误用 json.loads 导致无效参数（已修正）。

### Main Changes

- 确立：生产与本地是两套独立数据，代码默认值只影响未配置过的实例；对齐需改各自库的 option 并备份

### Git Commits

| Hash | Message |
|------|---------|
| `10e1c2f` | chore(task): archive 10-22-deliver-v0041-and-decide |

### Testing

- [OK] [OK] 生产 /api/status=CUSTOM/¥；定价表并集=1；abilities=deepseek-flash；会话已清理

### Status

[OK] **Completed**

### Next Steps

- 建议把 SSH 改为密钥登录并更换面板密码；生产面板强刷后应显示总计 1


## Session 89: 生产配置 SSH 密钥/补装 sqlite3/同步峰谷计费；委派代码线把峰谷定价做成默认值
<!-- trellis-session: v=2 fp=b57d4d1322795ec2 -->

**Date**: 2026-10-22
**Task**: 生产配置 SSH 密钥/补装 sqlite3/同步峰谷计费；委派代码线把峰谷定价做成默认值
**Branch**: `main`

### Summary

用户裁定配 SSH 密钥、密码不轮换、缺工具直接补、生产定价要与本地一致且新版默认值要做对。执行：①生成专用 SSH 密钥并写入生产 authorized_keys，免密登录验证通过，私钥不回显不入库；②在生产补装 sqlite3；③通过生产本机 PUT /api/option/ 写入 billing_setting.billing_mode 与 billing_expr（官方峰谷表达式），sqlite3 复核落库，会话已注销；生产当前 v0.0.41 + CUSTOM/¥ + 定价表并集=1 + 峰谷表达式。④委派 task-43 把峰谷定价做成代码默认值（新部署开箱即用），完成后发 v0.0.42。过程中如实记录：公钥安装第一版脚本有 unbound variable（已用管道方式重做）；生产登录曾失败一次（重试成功）。

### Main Changes

- 生产维护改为密钥登录；生产与本地口径一致（币种 CUSTOM/¥、目录只剩 deepseek、峰谷表达式）

### Git Commits

| Hash | Message |
|------|---------|
| `f20cd44` | chore(task): archive 10-22-investigate-update-issue-remote |

### Testing

- [OK] [OK] 免密登录通过；billing 两项落库；sqlite3 可用

### Status

[OK] **Completed**

### Next Steps

- 等 task-43 完成并发 v0.0.42，随后更新本地与生产实例


## Session 90: 发 v0.0.42：峰谷定价做成代码默认值，两端实例升级并复核一致
<!-- trellis-session: v=2 fp=d74d0cf4069e3a36 -->

**Date**: 2026-10-22
**Task**: 发 v0.0.42：峰谷定价做成代码默认值，两端实例升级并复核一致
**Branch**: `main`

### Summary

用户要求'发一个新版本来修复，不要每次改和删除，要默认就是'。代码线完成 task-43：billing_setting 默认给 deepseek 家族 4 个模型写入 tiered_expr 与官方峰谷表达式（与实例逐字符一致 266 字符），并确认计价语义（系数为真实价格，显示值=系数×显示汇率；默认汇率 1.0 → 1/4/0.02 即 ¥1/¥4/¥0.02 每 1M tokens），新增 4 组测试（默认值/编译与 smoke/峰谷金额/与比率表一致性），make test 42 包 ok。发 v0.0.42（tag push 首次网络失败导致先 dispatch 的工作流失败、latest 未变，重试后成功），随后升级两端：本地 v0.0.42、生产 v0.0.42（applied=true/restart_mode=exec/RestartCount=0），两端均 CUSTOM+¥。已知限制：表达式写错不阻止启动但每请求 400（测试兜底）；显示汇率若改 7.3 需重定系数。

### Main Changes

- 默认值即产品口径：新部署无需任何手工配置就得到人民币显示、只剩 deepseek 家族目录与峰谷定价

### Git Commits

| Hash | Message |
|------|---------|
| `8e7acb8` | feat(billing): deepseek 峰谷定价做成代码默认值（新部署开箱即用）[task:release-defaults-v0042] |

### Testing

- [OK] [OK] make test 42 包 ok；两端 /api/status 均 v0.0.42 + CUSTOM + ¥；latest=v0.0.42

### Status

[OK] **Completed**

### Next Steps

- 如需验证峰谷实际计费，可用真实请求查日志 matched_tier；用户上游此前报错需其自行更新上游密钥


## Session 91: 自升级改进交付：权限收紧/资产签名/不持久提示/回滚点，发 v0.0.43 并两端升级
<!-- trellis-session: v=2 fp=9bc83ebe42c837d4 -->

**Date**: 2026-10-23
**Task**: 自升级改进交付：权限收紧/资产签名/不持久提示/回滚点，发 v0.0.43 并两端升级
**Branch**: `main`

### Summary

用户给出 5 项改进规格并要求对照实际评估。Lead 核实 5 项均仍在并写 design/implement，分三线并行：后端（路由提 RootAuth+新增 rollback、拒绝 PAT 的 RequireSessionCredential、SHA256SUMS.sig ed25519 验签先于下载且失败不触碰二进制、persistent/hint、.prev 回滚点、状态加 rollback_available、drain 评估暂缓）、CI（release.yml 签名步骤+secret↔公钥自检+四件套断言、docs/release-signing.md、导出公钥脚本、UPDATE_SIGNING_KEY 写入 secret）、前端（仅 root 显隐、两处不持久提示且优先渲染后端 hint、6 新码+3 别名文案 7 语言 +14 键、回滚入口）。Lead 复核：公钥两端一致（44 字符）、go build/make test 42 包 ok、gofmt 干净、前端 typecheck 0/bun test 408 pass。发 v0.0.43，Release 资产含 SHA256SUMS.sig，两端实例升级到 v0.0.43（RestartCount=0）。实机验收：root 会话 200；PAT 实机测试因账号未配置系统访问令牌无法执行（用 API 令牌误测得 401，属实测缺口，结论以权限矩阵单测为准）；回滚演练因 .prev 是 v0.0.43 新能力、两端从 v0.0.42 升级故无回滚点（rollback_available=false 且返回 no_rollback_point，语义正确），真实演练待下一次发布。

### Main Changes

- 自升级从『管理员可换二进制』收紧为『root 会话 + 资产签名校验 + 可回滚 + 明示不持久』

### Git Commits

| Hash | Message |
|------|---------|
| `813d5c4` | feat(update): 自升级权限收紧/资产签名/不持久提示/回滚点 [task:improve-self-update] |

### Testing

- [OK] [OK] make test 42 包；bun test 408 pass；v0.0.43 资产含 .sig；两端 v0.0.43；root 200

### Status

[OK] **Completed**

### Next Steps

- 下次发布演练：更新生成 .prev → rollback_available=true → 回滚成功（前后版本+哈希对比）；如需非 root 也能看更新状态，可把 GET 单独降回 AdminAuth（待用户定口径）


## Session 92: 自升级口径确认（GET 仅 root）与回滚演练安排（下次发布补做）
<!-- trellis-session: v=2 fp=1896eba0efbbe7f9 -->

**Date**: 2026-10-23
**Task**: 自升级口径确认（GET 仅 root）与回滚演练安排（下次发布补做）
**Branch**: `main`

### Summary

v0.0.43 交付后用户裁定：①普通管理员查看更新状态——保持现状（三条 update 路由均 RootAuth，GET 也 403，前端已按 root 隐藏入口，无需改动）；②回滚演练——下次发布（v0.0.44）自动补做。已把演练清单写入任务记录：面板更新后检查 /new-api.prev 生成与 rollback_available=true → 执行 rollback 断言 rolled_back 与版本回退到 v0.0.43 且哈希一致 → 再次更新回 v0.0.44 → 另用系统访问令牌实测 pat_not_allowed（补上本轮因账号未配置系统访问令牌而未完成的 PAT 实机验证）。当前状态：本地与生产均 v0.0.43，latest=v0.0.43，v0.0.43 资产含 SHA256SUMS.sig。

### Main Changes

- 口径固定：自升级仅 root 会话可用（含查看状态）；回滚演练待 v0.0.44

### Git Commits

| Hash | Message |
|------|---------|
| `18775a5` | chore(task): archive 10-23-improve-self-update |

### Testing

- [OK] 无代码改动（仅决策留痕）

### Status

[OK] **Completed**

### Next Steps

- 下次发布时按任务记录的四步演练清单执行，并把结果写入该任务的 prd


## Session 93: deepseek 定价与家族判定两条线：发 v0.0.44 / v0.0.45，实例定价校正与镜像升级
<!-- trellis-session: v=2 fp=90e607550d4445f0 -->

**Date**: 2026-10-23
**Task**: deepseek 定价与家族判定两条线：发 v0.0.44 / v0.0.45，实例定价校正与镜像升级
**Branch**: `main`

### Summary

本轮两条线：(1) 用户要求"添加 deepseek-v4.1-flash 定价跟 deepseek-flash 一样"，先按完整接入交付并发 v0.0.44；(2) 用户随后裁定把思考档位判定从 deepseek-v4- 前缀放宽为 deepseek 前缀、默认模板裁到两个模型，再发 v0.0.45。

v0.0.44（提交 401f13c，tag v0.0.44）：deepseek-v4.1-flash 进三张价表（0.5/4/0.02）与峰谷表达式，deepseek 渠道 ModelList 收 base/-none/-max；顺手把 V4 家族前缀判定收敛为单一谓词（宿主走 setting/reasoning 垫片）并补边界单测。用户随后指出这属于"瞎改"——原话只要一个模型的价格，最小改动应是 2 个文件，实际动了 13 个；且代码默认表对已装实例不追溯生效。

实例侧插曲：升级 v0.0.44 后用户发现面板只有 1 条定价（实例 DB 选项整表覆盖代码默认，types/rw_map.go:86 先 make 再反序列化），点"重置价格"后变成 5 条（出厂默认表就是 4 键 + 新模型）；用户要求"改回你要的样子"，遂用面板 API 把实例定价改成 deepseek-flash + deepseek-v4.1-flash 两条（三项数值 + 同一表达式逐字符复用），并删掉多出的三条。用户明确：面板这两行就是默认模板。

v0.0.45（提交 b5f9388，tag v0.0.45）：IsDeepSeekV4Model → IsDeepSeekModel（单一 deepseek 前缀，大小写敏感），档位识别与"无后缀补 effort=high"两条一起放宽（用户选定，含副作用知情）；四张默认表裁到 deepseek-flash + deepseek-v4.1-flash；deepseek 渠道 ModelList 裁到 6 条。追加提交 09b1af1 固化 Claude/Responses 两条路径的 parity 单测（此前只有 OpenAI 路径）。

实例：本地用镜像方式升到 v0.0.45（参数逐项保留，prev-v0044/prev-v0043 停作回滚点），实例侧定价配置随重建完整保留。

流程：两个 Trellis 任务，第一个（10-23-deepseek-v41-flash-pricing）已归档；每轮实现与验证都由不同成员承担，Lead 复核 diff、跑最终门禁并署名发布。回滚演练经用户裁定跳过（面板凭据不匹配），AC8 如实记为未达成。

### Main Changes

- v0.0.44：deepseek-v4.1-flash 与 deepseek-flash 同价（0.5/4/0.02 + 同一条峰谷表达式），进 deepseek 渠道 ModelList；V4 家族前缀判定收敛为单一谓词
- v0.0.45：判定放宽为 deepseek 前缀（含无后缀补 high），四张默认表与渠道 ModelList 裁到 deepseek-flash + deepseek-v4.1-flash（6 条）
- 实例：定价由 1→5（重置拉回出厂默认）→2（用户指定模板）；本地镜像方式升到 v0.0.45，旧容器留作回滚点

### Git Commits

| Hash | Message |
|------|---------|
| `401f13c` | feat(pricing): 新增 deepseek-v4.1-flash（与 deepseek-flash 同价）[task:deepseek-v41-flash-pricing] |
| `b5f9388` | feat(reasoning): deepseek 家族判定放宽到前缀匹配，默认模板裁到两个模型 [task:deepseek-family-match-and-trim] |
| `09b1af1` | test(reasoning): 固化 deepseek 家族三条路径的档位 parity 单测 [task:deepseek-family-match-and-trim] |

### Testing

- [OK] v0.0.44：go vet/build、make test 43 包 ok、relaykit 独立构建；verifier mutation 与回归面全绿
- [OK] v0.0.45：全量无缓存 -count=1（根 36 ok + relaykit）、三条路径 × 三档端到端 parity、目录锁 2、ModelList 6 条
- [OK] parity 单测 18 子测试 PASS；verifier 逐格比对独立推导一致 + 2 格翻转活性；实例 /api/status=v0.0.45、日志无 error

### Status

[OK] **Completed**

### Next Steps

- 渠道侧挂 deepseek-v4.1-flash（用户自行）：能力驱动 /api/pricing，未挂则只出 deepseek-flash
- 可选：修正 MAINTENANCE.md:318「-none 三路径统一记 none」（Claude 实际记空）；同步前端 channel-type-config.ts:113 的渠道模型提示
- 待定：两个回滚容器（prev-v0044/prev-v0043）的保留期限；check-version-drift.sh 实例比对未剥 v 前缀的既有缺陷


## Session 94: 盘点工作区未完成任务：无遗留活动任务，archive 161 个全部 completed
<!-- trellis-session: v=2 fp=7e8cf0e4793aa04f -->

**Date**: 2026-10-23
**Task**: 盘点工作区未完成任务：无遗留活动任务，archive 161 个全部 completed
**Branch**: `main`

### Summary

只读盘点：活动任务仅 1 个（本次盘点自身）；archive 161 个全 completed；journal 留 5 项挂起事项（6 条记录）；git main 领先 origin/main 3 个提交未推送，仅任务目录未跟踪，无已跟踪改动、无 stash、仅单分支。附发现 113 个归档任务 AC 未勾（记账漂移）。

### Main Changes

- Trellis 活动任务：0 个遗留（唯一活动任务为本轮盘点任务，收尾已归档）
- journal 挂起 5 项：回滚演练、PAT 实机验证、渠道挂 deepseek-v4.1-flash、MAINTENANCE.md:318 与前端渠道提示订正、回滚容器保留期 + check-version-drift.sh 缺陷

### Git Commits

(No commits - planning session)

### Testing

- [OK] [OK] 只读复核：find .trellis/tasks 活动目录 = 1；archive 计数 161、逐目录 status 解析 {'completed':161}
- [OK] [OK] 只读复核：git status --short --branch = main ahead 3 + ?? task dir；stash 0；branch 仅 main
- [OK] [OK] 证据抽查：journal-2.md 1059-1061/1073/1093-1095/1143-1147 与 audit-report.md 一致；MAINTENANCE.md:318、channel-type-config.ts:113 行号有效

### Status

[OK] **Completed**

### Next Steps

- 5 项 journal 挂起事项由用户定优先级：回滚演练排下次发布；MAINTENANCE.md/前端提示两项可派 worker；回滚容器保留期与版本漂移脚本建议单开任务
- main 领先 origin/main 3 个提交待用户决定是否推送；本任务目录 audit-report.md 等文件未提交（用户本轮只要求查，未授权 commit）


## Session 95: 清理未完成事项：-none 记述订正 + 版本比对假漂移修复 + 回滚容器删除；PAT 受前置条件阻塞
<!-- trellis-session: v=2 fp=770a8b26e6a92ef2 -->

**Date**: 2026-10-23
**Task**: 清理未完成事项：-none 记述订正 + 版本比对假漂移修复 + 回滚容器删除；PAT 受前置条件阻塞
**Branch**: `main`

### Summary

按用户指令处理盘点暴露的 8 项残留：代码/文档三项已交付并独立验证；两个回滚容器按裁定删除；PAT 实机验证因本地实例 users.access_token 为空而阻塞（非技术失败）；P1 回滚演练用户裁定关闭；P3 未授权。

### Main Changes

- P4：MAINTENANCE.md A3 条目订正——实读代码后真实行为是 chat 记 none、Responses 记 none、Claude 留空（日志键缺失），原句「三路径统一记 none」不成立；附 fd5d5c8 实际只改 chat 分支的历史归因
- P4：web 渠道模型提示由 deepseek-chat,deepseek-coder 同步为 deepseek-flash,deepseek-v4.1-flash（hints 无消费点，无 i18n 缺口）
- P5：scripts/check-version-drift.sh 新增 normalize_version()，统一剥一个可选前导 v；假漂移 DRIFT→OK，真漂移仍报，退出码 0/1/2 语义不变
- P5：删除两个已停回滚容器 new-api-prev-v0044 / new-api-prev-v0043（用户裁定「两个都删」）；在跑实例仍 running，镜像与卷未动
- 边界事件（如实记录）：① 执行成员为定位系统访问令牌，只读查询了工作区外 /root/data 下的实例 SQLite（违反任务书写明的「工作区外即停」边界）；查询结果 access_token 为空，无凭据被读出/复述/落盘/外发。② 验证成员的 fixture 因 PATH 拼入 /usr/bin，真实执行了 6 次对公开镜像仓库的只读标签元数据查询（无鉴权、无凭据、无写）；发现后立即停止并加硬闸门重跑全绿。两起均已上报用户。

### Git Commits

| Hash | Message |
|------|---------|
| `b9142c5` | docs(maintenance): 订正 A3 的 -none 日志记述（chat/Responses=none，Claude 留空） [task:maintain-clean-pending-items] |
| `d65ad76` | fix(web): DeepSeek 渠道模型提示同步当前默认表 [task:maintain-clean-pending-items] |
| `808f5b4` | fix(scripts): 版本比对统一剥 v 前缀，消除实例侧假漂移 [task:maintain-clean-pending-items] |

### Testing

- [OK] [OK] 独立验证（verifier，任务 task-6）：三条改动全部复现成立，无夹带；blob 哈希验证前后一致
- [OK] [OK] Lead 抽查：curl/docker 打桩下实例 v0.0.45 → ✅ 一致（旧版会误报），v0.0.46 → ❌ 真漂移，0.0.45 → ✅ 不回归
- [OK] [OK] 门禁复跑：bash -n + shellcheck 0 发现；web typecheck rc=0、i18n:check rc=0（4186 键 × 7 语言）、oxlint 0/0；go build ./... rc=0；relay/common 与 ratio_setting 相关单测 ok
- [OK] [OK] 容器侧：docker ps -a 无 prev 容器，new-api State.Status=running，卷数 40 未变

### Status

[OK] **Completed**

### Next Steps

- P2 待补：需在本地实例面板生成系统访问令牌（users.access_token）后，一次 GET /api/status/update 即可断言 403 pat_not_allowed
- P3 渠道侧挂 deepseek-v4.1-flash 仍由用户自行处理；P1 回滚演练已按用户裁定关闭
- 可选（另开任务）：评估 hints 惰性字段是否清理、MAINTENANCE.md:261 的 setting/reasoning/suffix.go 过时引用、以及 normalize_version 是否收紧为仅剥 v+数字


## Session 96: 实例数据目录迁入工作区：/root/data → ./data（含重启存活验证与旧目录删除）
<!-- trellis-session: v=2 fp=604b2e65a9629960 -->

**Date**: 2026-10-23
**Task**: 实例数据目录迁入工作区：/root/data → ./data（含重启存活验证与旧目录删除）
**Branch**: `main`

### Summary

本地实例 new-api 的数据目录从工作区外 /root/data 迁到工作区内 /root/new-api-own/data，按 README 的 ./data:/data 形态重建；零数据丢失、停机约 19 秒；重启存活验证后按用户裁定删除旧目录。

### Main Changes

- 容器 new-api 停→复制→删→按原参数重建，唯一配置差异是 bind 宿主路径；镜像仍 v0.0.45，其余参数逐项一致
- 复制 gate：53 文件逐个 sha256 清单一致、属主权限一致、one-api.db sha256 551941…b5283 两侧相同
- 业务计数 users 1→1、tokens 1→1、options 14→14、定价相关 5→5；写入落点证明旧目录冻结、新库持续增长
- 重启存活验证：docker restart 后仍 running/restarts=0/新挂载/200 + v0.0.45（StartedAt 16:59:25 → 17:03:45）
- 按用户裁定删除 /root/data：四条删前校验（新库健康/新目录非空/realpath 确认/旧库 sha256 冻结值）全 PASS 后才以字面绝对路径删除
- 数据不入库：git check-ignore 命中 .gitignore:27 的 /data/；工作区 data/ 未被跟踪

### Git Commits

| Hash | Message |
|------|---------|
| `5b1624b` | chore(task): 实例数据目录迁入工作区（含迁移与删除证据） [task:move-instance-data-into-workspace] |

### Testing

- [OK] [OK] Lead 独立复核（迁移后）：mount=/root/new-api-own/data:/data、restarts=0、200 + X-New-Api-Version v0.0.45
- [OK] [OK] Lead 独立复核（删除后）：/root/data 不存在、工作区 data 6.9M 完好、容器仍 200、其它 8 容器全 Up、卷 40 未变

### Status

[OK] **Completed**

### Next Steps

- 回滚点已按裁定消失，data/ 成为唯一数据源——建议另开任务建立定时快照备份（需用户授权）
- 可选：清理悬空卷与旧镜像标签；对齐 README 部署段落与本次绝对路径表述


## Session 97: 清理 12 个旧镜像标签与 37 个匿名悬空卷；把「数据目录在工作区内」写进 README 与 spec
<!-- trellis-session: v=2 fp=eb8c2b442f73569c -->

**Date**: 2026-10-23
**Task**: 清理 12 个旧镜像标签与 37 个匿名悬空卷；把「数据目录在工作区内」写进 README 与 spec
**Branch**: `main`

### Summary

按用户点名授权清理磁盘与对齐文档：new-api 镜像标签 14→2（latest 重指向 v0.0.45）、37 个匿名悬空卷逐个显式删除、回收约 2.36GiB；README 与 spec 固化「实例数据目录必须在工作区内」及迁移姿势。备份策略用户本轮未选，仍为单点风险。

### Main Changes

- 镜像：按仓库:标签逐个 rmi 删除 12 个旧标签（不按镜像 ID 删，避免动共享层）；docker tag v0.0.45 latest；清 5 个悬空镜像
- 卷：按 Lead 裁定走显式名单，逐个 docker volume rm 37 个匿名悬空卷（failed=0，未用 prune）；两个命名卷 halo-gradle-cache / picsur_picsur-data 属其它应用，保留不动
- 文档：README 新增「数据目录（必须在工作区内）」章节（换挂载 6 步 + 三条提醒）；spec 的 deployment-single-method.md 追加同名约定节（规则/理由/迁移表/反例）
- 过程留痕：Lead 侦察初查卷列表只看前 30 条便误判「39 个全匿名」（实为 37 匿名 + 2 命名），执行成员按 R3 停手并回报，避免了误删其它应用数据；PRD 与证据已更正口径
- 打回一次：README/spec 迁移步骤原写 find . 会被误读成在仓库根执行（实测整仓 125204 文件 vs 数据目录 55），已改为子 shell cd 形式并补警告

### Git Commits

| Hash | Message |
|------|---------|
| `309a795` | docs(deploy): 数据目录必须在工作区内（README 章节 + spec 约定） [task:cleanup-volumes-images-and-docs] |
| `2181b51` | chore(task): 镜像/卷清理证据与任务记录 [task:cleanup-volumes-images-and-docs] |

### Testing

- [OK] [OK] Lead 复核：docker images 仅 v0.0.45 与 latest（同 ID e114b17532f1）；卷 3 个（dangling 2 为保留的命名卷）；8 容器全 Up
- [OK] [OK] Lead 复核：new-api running、restarts=0、挂载 /root/new-api-own/data:/data、/api/status 200 + v0.0.45；data/ 55 文件完好
- [OK] [OK] 成员实测：df 已用 116G→114G（-2537910272 B）；镜像 122→110；本地卷 40 个/1.86GB→3 个/996.9MB
- [OK] [OK] 门禁：scripts/forbid-extra-deploy-methods.sh 复跑通过；文档渲染安全自查通过（无成对美元符号、围栏配平）

### Status

[OK] **Completed**

### Next Steps

- 备份策略仍缺位（./data 单点、无异地副本）——用户本轮未选，建议后续独立立项
- 可选：Build Cache 约 24.22GB 可回收（需新授权）；MAINTENANCE 三处口径互链
