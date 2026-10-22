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
