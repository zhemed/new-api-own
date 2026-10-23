# deepseek-v4.1-flash 定价接入 + 发 v0.0.44（含回滚演练）

## Goal

新增模型 `deepseek-v4.1-flash`，定价与 `deepseek-flash` 完全一致，并把它接进 deepseek 家族的既有
默认目录与渠道清单；随后发 v0.0.44（镜像 + Release），并在本地实例补做 v0.0.43 遗留、约定"下次发布
自动补做"的回滚演练。

## 背景（事实，来自仓库与任务记录）

- 默认目录（2026-10-22 起）只保留 deepseek 家族 4 个键：`deepseek-flash` / `deepseek-v4-flash` /
  `deepseek-chat` / `deepseek-reasoner`，三项价表（倍率 0.5 / 输出 4 / 缓存 0.02）加一条官方峰谷表达式。
- `.trellis/tasks/archive/2026-10/10-23-decide-update-scope-and-drill/prd.md` 用户裁定第 2 条：
  回滚演练**下次发布（v0.0.44）自动补做**，四步清单已写明。
- 本地实例当前 v0.0.43；容器内 `/new-api.prev` 不存在（v0.0.43 由 v0.0.42 升上来，旧版不写回滚点）。

## Requirements

### R1 定价与 deepseek-flash 逐项一致

`deepseek-v4.1-flash` 必须同时进入下列默认表，取值与 `deepseek-flash` 完全相同：

| 表 | 文件 | 取值 |
|---|---|---|
| `defaultModelRatio` | `setting/ratio_setting/model_ratio.go` | 0.5 |
| `defaultCompletionRatio` | `setting/ratio_setting/model_ratio.go` | 4 |
| `defaultCacheRatio` | `setting/ratio_setting/cache_ratio.go` | 0.02 |
| `defaultTieredBillingModels` | `setting/billing_setting/tiered_billing.go` | 与家族同一条峰谷表达式 |

显示口径（显示汇率 1.0 时）：低谷 未命中 1.00 元 / 输出 4.00 元 / 缓存命中 0.02 元（每 1M tokens），
高峰时段同一表达式整体 ×2。

### R2 接入面与 `deepseek-v4-flash` 完全对齐

1. deepseek 渠道 `ModelList` 增 `deepseek-v4.1-flash`，以及思考档位后缀 `-none` / `-max` 两个条目
   （与 `deepseek-v4-flash` 的现有形态一致）。
2. 思考档位后缀归一覆盖新名字：`deepseek-v4.1-flash-max` / `-none` 必须被识别为 V4 系思考档位，
   计费按归一后的基名 `deepseek-v4.1-flash` 命中（与 `deepseek-v4-flash-max` 行为一致）。
3. V4 家族前缀判定收敛为**单一实现**，不再在各处硬编码 `deepseek-v4-` 字面量。

### R3 回归面不扩大

除新模型外，任何既有模型的定价、计费模式、渠道清单与思考档位行为都不得改变；被显式锁定的目录规模
测试随之更新为新的事实。

### R4 发版 v0.0.44

1. `VERSION` 递增到 `0.0.44`（不带 v），提交进 `main`，打注释 tag `v0.0.44`，推送 `main` 与 tag。
2. CI 产出多架构镜像 `v0.0.44` / `0.0.44` / `latest`，以及带 `SHA256SUMS.sig` 的 Release 资产。
3. `scripts/check-version-drift.sh` 判定一致（仓库 VERSION / 最新 tag / registry 三个标签摘要）。

### R5 回滚演练（本地实例，四步）

按 `10-23-decide-update-scope-and-drill/prd.md` 的清单执行：

1. 面板更新到 v0.0.44 → 检查容器内 `/new-api.prev` 生成、状态接口 `rollback_available=true`；
2. `POST /api/status/update/rollback` → 断言 `rolled_back:true`、版本回到 v0.0.43、二进制哈希与
   回滚前的 v0.0.43 一致；
3. 再更新一次回到 v0.0.44（重建回滚点）；
4. 用**系统访问令牌**（PAT）调三个 update 端点，实测返回 `pat_not_allowed`。

## 验收结果（2026-10-23 结项）

- [x] AC1 `deepseek-v4.1-flash` 三项数值 0.5 / 4 / 0.02，`GetBillingMode` = `tiered_expr`，表达式与
      `deepseek-flash` 逐字符相同 —— Lead 抽查 + verify-pricing 独立复跑（含金额级求值：P=1M→1e6、
      C=1M→4e6、CR=1M→20000，与官方 ¥1.00/¥4.00/¥0.02 吻合）。
- [x] AC2 五个模型三项数值两两相等 —— 由 `TestDeepSeekFamilySharesOnePricing` 断言；mutation（0.5→0.9）
      实测能让断言变红（verify-pricing 复现并还原，md5 一致）。
- [x] AC3 `-max` / `-none` 归一到基名并命中同一价，且与 `deepseek-v4-flash-*` 逐项一致；
      `deepseek-v4-pro*`、`gpt-4o` 行为不变（回归对照 HEAD 与当前源码抽取，差异只有新增键）。
- [x] AC4 deepseek 渠道 `ModelList` 含新模型三件套（11 条 = HEAD 8 + 新 3）。
- [x] AC5 `go vet` / `go build` / `make test` / relaykit 独立构建全部通过；抽查包用 `-count=1`
      复跑通过（verify-pricing 另做根模块 86 包 `-count=1 -p 1` 全绿）。
- [x] AC6 目录锁 4→5，家族断言更新后通过；其余默认表仍为空。
- [x] AC7 tag v0.0.44 已推送；镜像 `latest` / `v0.0.44` / `0.0.44` 同摘要
      `sha256:92d066b505ae98ecb0515ed7ae64008a1d62ef6f575d57f2da1e2b82fbe6b83f`；
      Release 资产含 `new-api-linux-amd64` / `new-api-linux-arm64` / `SHA256SUMS` / `SHA256SUMS.sig`。
- [ ] **AC8 未达成**：回滚演练经用户当轮裁定跳过（面板登录凭据不匹配 → 用户选"算了，跳过演练"），
      `.prev` / `rollback_available` / rollback 哈希比对 / PAT 实测**均未执行**；作为替代，本地实例改用
      镜像方式升到 v0.0.44（容器参数逐项保留，旧容器 `new-api-prev-v0043` 留作回滚路径）。
- [x] AC9 改动落在 13 个文件内（11 代码/测试 + `VERSION` + `MAINTENANCE.md`；后者是 verify-pricing 报告的
      文档漂移，由 Lead 裁定纳入并同步更新 design 之外的声明），提交消息带
      `[task:deepseek-v41-flash-pricing]` 锚点，闸门放行。

## 遗留与后续（本轮发现，未处理）

1. **实例侧定价遮蔽**：本地实例的定价来自 DB 选项（2026-10-22 清成只剩 `deepseek-flash`），
   `/api/pricing` 只返回 1 条 → `deepseek-v4.1-flash` 在该实例上**尚未定价**；需在面板加模型或
   "重置比率"才生效。代码默认值只对新部署/重置后生效。
2. **check-version-drift 既有缺陷**：`scripts/check-version-drift.sh` 的实例比对是字面比较
   （实例报 `v0.0.44`，VERSION 是 `0.0.44`），未剥 `v` 前缀 → 带 `--instance` 必然报 DRIFT
   （blame 8be8ba3f，2026-10-06 起既有，非本轮引入）。
3. **测试门禁口径**：`make test` 会命中缓存（verifier 实测 43 条 ok 全是 `(cached)`），
   建议门禁改用 `-count=1`（必要时 `-p 1`）。
4. **model 包既有 flake**：`TestLogPayloadTrimIsSingleFlight` 在无缓存并行全量跑时偶发失败，
   `-p 1` 串行全绿；与本次 diff 无关（本轮未动 `model/`）。
5. **go fmt 历史债**：9 个 `relay/channel/**` 文件在 HEAD 即未格式化，本轮按裁定不清理。

## Non-goals

- 不升级生产实例（用户本轮选择：只升级本地）。
- 不改前端（web 无 deepseek 硬编码，定价从接口取）。
- 不调整其它模型价格、不改峰谷表达式的字面量、不动 `.prev` 只保留一份的既有约束。
- 不做 graceful drain 钩子（v0.0.43 已裁定暂缓）。

## 用户裁定（本轮）

| 问题 | 裁定 |
|---|---|
| 接入范围 | **完整接入**：三张价表 + 峰谷表达式 + 渠道 ModelList + `-none`/`-max` 后缀 |
| 未发版的"维护" | 就是 v0.0.43 遗留的**回滚演练**，本次随 v0.0.44 补做 |
| 实例升级范围 | **只升级本地实例** |
| 提交/发版 | 改完直接提交并发版 v0.0.44 |

## 已知风险与残余

- 更新期间面板会 exec 重启，`.prev` 只保留最近一次；演练会把本地实例短暂回退到 v0.0.43 再升回，
  期间若中断，恢复手段是重跑面板更新（镜像 tag 都在）。
- PAT 实机验证需要面板内**系统访问令牌**；若账号未配置，需现场创建，否则该步按"实测缺口"如实记录。
