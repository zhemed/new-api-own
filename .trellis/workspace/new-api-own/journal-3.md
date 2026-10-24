# Journal - new-api-own (Part 3)

> Continuation from `journal-2.md` (archived at ~2000 lines)
> Started: 2026-10-24

---



## Session 114: 加固 panic recovery（不回显原文/堆栈日志/已写响应保护）；清旧回滚镜像；分类 85 个导出符号
<!-- trellis-session: v=2 fp=267138390197404e -->

**Date**: 2026-10-24
**Task**: 加固 panic recovery（不回显原文/堆栈日志/已写响应保护）；清旧回滚镜像；分类 85 个导出符号
**Branch**: `main`

### Summary

更正上轮错误结论：panic recovery 一直存在（main.go:174 的 gin.CustomRecovery），不是缺失。真实问题是三个：把 panic 原文回显给客户端（信息泄露）、只记 SysLog 无堆栈无 request id、响应已写时仍追加 JSON。已抽成 middleware.PanicRecovery 并修掉三点（5 个测试先红后绿）。另清掉 v0.0.45-47 三个旧镜像（allow-list 释放约 451MB），并把 85 个导出符号分成 A76/B4/C5 三堆（含 owner 决策项）。

### Main Changes

- 新增 middleware/panic_recovery.go：CustomRecoveryWithWriter(nil) 关闭 gin 自带日志（其 debug 模式会明文 dump Cookie），客户端只拿通用文案，服务端 error 级日志含完整堆栈+方法/路径+request id，c.Writer.Written() 为真时只 Abort
- main.go：内联 recovery 换成 server.Use(middleware.PanicRecovery())，仍在最外层先于 RequestId
- 新增 middleware/panic_recovery_test.go 五用例：不泄露敏感串、日志含堆栈与上下文、panic 后路由可用、已写响应不追加 JSON、broken pipe 不写 body
- 镜像清理：删除 v0.0.47/46/45（allow-list + 完整 ID 保留集合 + 运行中镜像二次核对 + 无 prune），保留 v0.0.49(=latest) 与 v0.0.48，释放约 450.79 MiB
- 符号分类：85 个 = A 可删 76 / B 需补调用 4 / C 保留 5；按包 common31、relaykit23、pkg13、setting12、oauth2、types2、relay1、service1

### Git Commits

| Hash | Message |
|------|---------|
| `193a409` | fix(security): panic recovery 不回显原文、error 级堆栈日志、已写响应不追加 JSON [task:panic-recovery-and-cleanup] |
| `f3a2353` | chore(task): recovery 加固、镜像清理与符号分类的证据 [task:panic-recovery-and-cleanup] |

### Testing

- [OK] [OK] 门禁：build/vet/全仓 36 包测试/gofmt 空；recovery 5 用例全 PASS（修复前 3 条真实失败：响应含 sk-live-DEADBEEF、无 ERR/堆栈、已写响应被追加 JSON）
- [OK] [OK] 镜像清理后：实例 v0.0.49 running/restarts=0/200、8 容器、卷 3、data 59 文件不变；v0.0.48 回滚点仍在

### Status

[OK] **Completed**

### Next Steps

- 本 recovery 加固是行为变更，需发 v0.0.50 才到实例（用户未点头，暂未发）
- 归档证据 evidence-deadcode.md §5.2 正文为空（上一轮写丢），本轮已在 evidence-symbols.md 重建 85 条；建议后续把清单回填并保留 79/82/85 口径说明
- A 堆 76 个可删（建议顺序：pkg+oauth+types+setting 29 → common 29 → relaykit 18 需 owner 批 → B 4 项接线+单测）；relaykit 独立模块 API 面需 owner 决策


## Session 115: 档位名显示层翻译（peak/offpeak→高峰/低谷）；清 28 个内部面导出符号 + B 堆接线
<!-- trellis-session: v=2 fp=994cce1af5862494 -->

**Date**: 2026-10-24
**Task**: 档位名显示层翻译（peak/offpeak→高峰/低谷）；清 28 个内部面导出符号 + B 堆接线
**Branch**: `main`

### Summary

两件事：① 用户截图发现「动态计费」档位名显示英文 peak/offpeak——根因是档位名是表达式里的字面量、前端原样渲染且无翻译映射；改为展示层映射（12 个内置标签），中文 高峰/低谷，未知标签原样、匹配高亮仍用原始标签。② 技术债继续：删 28 个内部面导出死符号（pkg/oauth/types/setting），并把两处「本该被用却没人调」的符号接上线（common.IsPrivateIP 合并为唯一判定源、GetContextKeyStringSlice 用于 service/group.go），relaykit 的 JSON 包装删除并在 README 记录模块例外。用户明确本轮不发版。

### Main Changes

- 前端：新增 web/src/features/pricing/lib/tier-labels.ts（12 标签映射 + getTierDisplayLabel），dynamic-pricing-breakdown.tsx 两处渲染点接入；只影响展示，normalizeTierLabel/matchedTierLabel/formatConditionSummary 仍用原始标签（有反证测试）
- i18n：7 个 locale 各 +11 行（新增 Peak hours/Off-peak hours 等键，只插入不重排）；STATIC_I18N_KEYS 登记 12 键；i18n:check 4197 键 × 7 语言全存在
- 死代码：删 28 个导出符号（billingexpr 1、cachex 3、ionet 9、oauth 2、types 2、setting 11，共 15 文件 -368/+1）；保留 setting/ratio_setting.GetModelRatioOrPrice（被 web 侧 TSX 注释点名，删了会让注释悬空）
- B 堆接线：common/ip.go 删旧 IsPrivateIP（HEAD 全仓 0 调用），ssrf_protection.go 内部实现导出为唯一判定源；service/group.go 改用 common.GetContextKeyStringSlice 且保持三分支语义；relaykit/kitutil 两个无人使用的 JSON 包装删除 + relaykit/README.md 记「模块例外」

### Git Commits

| Hash | Message |
|------|---------|
| `4714e01` | chore: 删 28 个内部面导出死符号；IsPrivateIP/GetContextKeyStringSlice 接线；relaykit JSON 例外入文档 [task:internal-symbols-and-b-wiring] |
| `9752a5e` | chore(task): 内部面符号清理与 B 堆接线证据 [task:internal-symbols-and-b-wiring] |
| `5bbedbb` | fix(web): 档位名按当前语言显示（peak/offpeak → 高峰/低谷），仅展示层翻译 [task:tier-name-english-display] |
| `8ad544b` | chore(task): 档位名翻译的证据 [task:tier-name-english-display] |
| `4db0809` | chore(task): archive 10-24-internal-symbols-and-b-wiring |

### Testing

- [OK] [OK] Go 门禁：build/vet/全仓 36 包测试/relaykit GOWORK=off build+test 全绿；gofmt 空；unused 恰 1（logKeyCol）；deadcode 85 → 53 且新增不可达 0
- [OK] [OK] 前端门禁：typecheck、i18n:check（4197 键 × 7 语言）、bun test src/features/pricing 15 pass、全量 bun test 420 pass、oxlint 0 warning
- [OK] [OK] 独立验证（task-56）：接受；31 个导出声明删除全量核对零残留引用、非 Go 位置零引用；IsPrivateIP 合并经 git grep 证明旧导出 0 调用、函数体仅改函数名 → 无行为倒退；三分支语义等价

### Status

[OK] **Completed**

### Next Steps

- 本轮未发版（用户明确不要频繁发版）：这些改动攒在 main，等下次发版一起带出去
- 可选收尾：4 个孤儿导出类型（pkg/cachex.StringCodec 空壳、pkg/ionet 的 ContainerLogs/LogEntry/LocationAvailability）；GetModelRatioOrPrice（需连 web 注释一起改）；locale 中现已无引用的 Peak/Off-peak 两键
- relaykit 20 项导出 API 面仍等 owner 决策


## Session 116: 删 5 个孤儿导出类型、2 个失效 locale 键；relaykit 收敛 17 项并补公开 API 政策
<!-- trellis-session: v=2 fp=81df1526cc65b5c2 -->

**Date**: 2026-10-24
**Task**: 删 5 个孤儿导出类型、2 个失效 locale 键；relaykit 收敛 17 项并补公开 API 政策
**Branch**: `main`

### Summary

技术债收尾一轮（不发版）：① 删 5 个孤儿导出类型（cachex.StringCodec、ionet 的 ContainerLogs/LogEntry/LocationAvailability/HardwareAvailability）；② 删 2 个失效 locale 键（Peak/Off-peak，档位名翻译已改用 Peak hours/Off-peak hours）；③ 用户拍板 relaykit 定位为「内部拆分、不独立发布」后，收敛 16 个导出函数 + dto.StringValue 类型，同步改写 2 个测试文件 3 处调用（断言零改动），并在 relaykit/README.md 增加「公开 API 政策」并纠正 go get 表述。

### Main Changes

- pkg：cachex/codec.go -2、ionet/types.go -33（5 个类型；其中 HardwareAvailability 是本轮新发现的第 5 个孤儿）
- locale：7 个语言文件各 -2 行（无重排）；键数 5330→5328、引用数 4197 不变（反证原本无引用）
- relaykit：删 8 个兼容门面 + kitutil.StringToByteSlice + types.NewFileMeta/NewImageFileMeta + dto.StringValue（类型与 2 方法）+ 3 个仅模块自测用的门面；两条测试改写为直调 internal 实现（用例名/输入/断言未动）
- relaykit/README.md：新增「公开 API 政策」（公开面/内部面/变更流程），定位改为「当前无 tag、无发布流水线、不可 go get」；保留 4 个注册表与内省入口并列入公开面，依赖实况按验证意见写精确

### Git Commits

| Hash | Message |
|------|---------|
| `7f334e2` | chore: 删除 5 个孤儿导出类型（cachex/ionet） [task:orphan-types-and-relaykit-policy] |
| `26749d3` | chore(i18n): 删除失效键 Peak / Off-peak（档位名已改用 Peak hours / Off-peak hours） [task:orphan-types-and-relaykit-policy] |
| `6cb7eb6` | chore(relaykit): 收敛 16 个导出函数 + dto.StringValue，补公开 API 政策（定位为内部拆分） [task:orphan-types-and-relaykit-policy] |
| `714e8f7` | chore(task): 孤儿类型/locale/relaykit 决策与执行证据 [task:orphan-types-and-relaykit-policy] |

### Testing

- [OK] [OK] 门禁：根模块 build/vet/全仓测试/gofmt 空/unused 恰 1；relaykit GOWORK=off build + 7 包测试全绿；前端 typecheck/i18n:check（4197 键）/bun test src/features/pricing 15 pass
- [OK] [OK] deadcode：根模块 53 → 37（-16，按符号名比对无新增不可达）；relaykit 模块 109 → 96（-13）
- [OK] [OK] 独立验证（task-61）：接受；5 类型零引用且无序列化/GORM/反射依赖；locale 键引用核实 7 项全 0；测试改写逐字等价（被删门面原本就是转发同一 internal 函数）；README 表述与 replace/无 tag/无 workflow 结构一致

### Status

[OK] **Completed**

### Next Steps

- 用户已裁定 relaykit 不独立发布；NewResponseStreamStateByID 当前零引用但按公开面内省入口保留，后续可再评估
- 仍未发版（用户要求不频繁发版）：main 已累积 panic recovery 加固、档位名翻译、多轮死代码清理，等下次发版一起带出


## Session 117: 发布 v0.0.50：panic recovery 加固、档位名中文显示、死代码清理、lint CI 一并带出
<!-- trellis-session: v=2 fp=4f49e9666e9ab0ae -->

**Date**: 2026-10-24
**Task**: 发布 v0.0.50：panic recovery 加固、档位名中文显示、死代码清理、lint CI 一并带出
**Branch**: `main`

### Summary

用户点名发一版带上全部累积改动。三个 workflow 全绿（Publish image 40076787622 / Publish release 40076787630 / lint 40076785799），四资产含 SHA256SUMS.sig；实例升级到 v0.0.50（2 秒就绪，restarts=0，200），v0.0.49/v0.0.48 保留作回滚点，latest 已对齐。真机复核：运行实例服务出的 JS 含 Peak hours/高峰/低谷（旧键 Peak/Off-peak 0 命中）；计费配置未变（两行 tiered_expr、表达式同 sha）；二进制含 panic_recovery 实现。

### Main Changes

- VERSION 0.0.50 + 附注 tag（注解即 Release 正文，列明四类改动）
- 实例：docker pull → rm -f → 原参数重建（host/always/json-file 10m×3/四个 LOG_*/工作区挂载），仅镜像 tag 变化

### Git Commits

| Hash | Message |
|------|---------|
| `0920671` | chore(release): 版本提升到 0.0.50 [task:release-v0050] |
| `11d0678` | chore(task): v0.0.50 发版与升级证据 [task:release-v0050] |

### Testing

- [OK] [OK] 三 CI 全绿 + Release 四资产 + manifest 多架构就位
- [OK] [OK] 真机复核：/api/status 200 + v0.0.50；8 容器全 Up；卷 3；data 59→60（新日志）；运行实例 JS 内 Peak hours/高峰/低谷 命中、旧键 0；billing_setting 两行 tiered_expr 且表达式 sha 与历史基线一致

### Status

[OK] **Completed**

### Next Steps

- 观察一个运行周期后可按「当前+上一版」清理 v0.0.48（需单独点头）；data/ 仍无异地备份
- 面板「立即更新」真机演练可在下一版本窗口闭环（v0.0.50 是目前最新，点击只会提示已是最新）


## Session 118: 档位名：措辞改为高峰时段/空闲时段，并把翻译补齐到日志与模型详情（发 v0.0.51）
<!-- trellis-session: v=2 fp=ab062af4fdc9a742 -->

**Date**: 2026-10-24
**Task**: 档位名：措辞改为高峰时段/空闲时段，并把翻译补齐到日志与模型详情（发 v0.0.51）
**Branch**: `main`

### Summary

用户指出两点：① 档位名的正确中文措辞是「高峰时段/空闲时段」（我先前用了高峰/低谷、更早还用过峰值）；② 用量日志里仍显示 offpeak，说明 v0.0.50 只翻译了定价面板一处。本版按 4 个展示入口全部接入同一 helper（定价面板/日志列表/日志详情/模型详情），措辞改为高峰时段与空闲时段（zh/zh-TW），表达式与历史数据不动、翻译在渲染时发生。用户点名要求立即发版生效。

### Main Changes

- web/src/i18n/locales/zh.json、zh-TW.json：Peak hours→高峰时段/高峰時段，Off-peak hours→空闲时段/空閒時段
- 日志列表 common-logs-columns.tsx、日志详情 details-dialog.tsx、模型详情 model-details.tsx 改用 getTierDisplayLabel
- 新增/更新测试：两处入口渲染断言、未知标签原样、空标签默认、反向测试（高亮仍按原始标签）；旧措辞断言同步更新

### Git Commits

| Hash | Message |
|------|---------|
| `61f9e2e` | fix(web): 档位名中文措辞改为高峰时段/空闲时段，并补齐日志列表/日志详情/模型详情三处翻译 [task:tier-label-display-other-surfaces] |
| `d84f45a` | chore(task): 4 个展示入口翻译与措辞修正的证据 [task:tier-label-display-other-surfaces] |
| `ab0dfa3` | chore(release): 版本提升到 0.0.51 [task:tier-label-display-other-surfaces] |

### Testing

- [OK] [OK] 前端门禁：typecheck、i18n:check（4197 键 ×7 语言）、oxlint 0/0、format:check；相关 31 项测试与全量 427 项测试全绿

### Status

[OK] **Completed**

### Next Steps

- v0.0.51 发布并升级本机实例后，用户在日志与面板都能看到「空闲时段/高峰时段」


## Session 119: 发布 v0.0.51：档位名措辞改为高峰时段/空闲时段，翻译补齐 4 个展示入口
<!-- trellis-session: v=2 fp=c30fe3881a7a0374 -->

**Date**: 2026-10-24
**Task**: 发布 v0.0.51：档位名措辞改为高峰时段/空闲时段，翻译补齐 4 个展示入口
**Branch**: `main`

### Summary

按用户给的措辞修正：中文档位名改为「高峰时段/空闲时段」（原为高峰/低谷、更早误用峰值），并把展示层翻译补齐到全部 4 个入口（定价面板、用量日志列表详情列、日志详情弹窗、模型详情价格表）。三 CI 全绿、四资产齐；实例升级到 v0.0.51 并用运行实例实际服务出的 JS 验证新措辞在线（高峰时段/空闲时段各命中 1，旧措辞与 offpeak 原始拼接 0 残留）。

### Main Changes

- i18n：zh 高峰时段/空闲时段，zh-TW 高峰時段/空閒時段；其余语言保持原义
- 日志列表/日志详情/模型详情三处接入 getTierDisplayLabel（定价面板上版已接），逻辑路径仍用原始标签
- 发版：VERSION 0.0.51 + 附注 tag；Publish image 40080432515 / Publish release 40080432597 / lint 40080383189 三绿；实例 2 秒就绪、restarts=0、200 + v0.0.51；v0.0.50/49 保留回滚点；latest 对齐

### Git Commits

| Hash | Message |
|------|---------|
| `61f9e2e` | fix(web): 档位名中文措辞改为高峰时段/空闲时段，并补齐日志列表/日志详情/模型详情三处翻译 [task:tier-label-display-other-surfaces] |
| `d84f45a` | chore(task): 4 个展示入口翻译与措辞修正的证据 [task:tier-label-display-other-surfaces] |
| `ab0dfa3` | chore(release): 版本提升到 0.0.51 [task:tier-label-display-other-surfaces] |
| `d047189` | chore(task): v0.0.51 发版与措辞核对证据 [task:release-v0051] |

### Testing

- [OK] [OK] 前端：typecheck/i18n:check(4197 键)/oxlint 0-0/format:check；相关 31 项与全量 427 项测试全绿
- [OK] [OK] 真机：服务出的 index.b47b92b672.js 中 高峰时段=1、空闲时段=1、旧措辞与 'offpeak ·' 均 0；四个入口源码全部走同一 helper

### Status

[OK] **Completed**

### Next Steps

- 本机 logs 表无 tiered_expr 记录（本实例未产生该计费用量），用户截图那条日志可能来自另一台实例——那台需同样升级到 v0.0.51 才会显示中文
- 浏览器可能缓存旧 JS，需硬刷新（Ctrl+Shift+R）才能看到新措辞
