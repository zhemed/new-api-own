# 修复「重置定价」只重置倍率导致的表达式不齐（两行不一致）

## Goal

用户报告：面板「模型定价」里 `deepseek-flash` 显示「阶梯计 / 基于表达式」，而 `deepseek-v4.1-flash` 显示
「按 Token / 仅基础输入价格」；在另一台已升级 v0.0.46 的实例上点过「重置」后仍然不一致。

**已定位为真实缺陷**（不是配置玄学、也不是版本没更新），修复方案 C：
让「重置」真正重置**整份定价默认**（含计费模式与表达式），确认框与提示写清范围，并补回归测试
（重置后 `deepseek-flash` 与 `deepseek-v4.1-flash` 的定价配置必须逐字符一致）。修完发版。

## 根因（证据链）

| 环节 | 事实 | 位置 |
|---|---|---|
| 按钮 | 前端 `resetModelRatios()` → `POST /api/option/rest_model_ratio` | `web/src/features/system-settings/api.ts:87-93` |
| 后端 | **只**写 `ModelRatio` 一个键：`model.UpdateOption("ModelRatio", DefaultModelRatio2JSONString())` | `controller/pricing.go:79-101` |
| 未重置 | `CompletionRatio`、`CacheRatio`、`CreateCacheRatio`、`ModelPrice`、`billing_setting.billing_mode`、`billing_setting.billing_expr` 全部保持实例原值 | 同上 |
| 文案 | 前端提示「模型价格重置成功」，让人以为整份定价恢复默认 | `ratio-settings-card.tsx:168-176` |
| 默认模板 | **本身是对的**：两个模型都在默认阶梯计费清单里，表达式按测试逐字符相同 | `setting/billing_setting/tiered_billing.go:38`、`tiered_billing_defaults_test.go:21`、`ratio_setting/model_defaults_test.go:55-82` |

结论：缺陷是**重置范围与按钮语义不符**。受影响实例的 `billing_setting.billing_expr` 里只有
`deepseek-flash` 的键（`deepseek-v4.1-flash` 缺失），重置又不碰这个键，于是那一行长期显示为按 Token/仅基础。
本机之所以看起来正常，是因为早期会话手工把两行都设成了同一表达式（DB 中两键 sha256 相同，已核）。

## Requirements

- R1 后端：重置接口改为恢复**整份定价默认值**，至少覆盖：
  `ModelRatio`、`CompletionRatio`、`CacheRatio`、`CreateCacheRatio`、`ModelPrice`、
  `ImageRatio`、`AudioRatio`、`AudioCompletionRatio`、`billing_setting.billing_mode`、`billing_setting.billing_expr`。
  - 每项都写其**代码默认**（与前端 `modelDefaults` 同源），写入 DB（`model.UpdateOption`）并同步内存
    （对应 `setting/...UpdateXxxByJSONString`）；任一失败要如实返回错误，不得半途而废静默成功。
  - **不**动 `GroupRatio` / `TopupGroupRatio`（分组倍率不在这颗按钮语义内）；不动 `ExposeRatioEnabled` 等开关。
  - 接口路径与响应结构保持兼容（前端已有调用），错误语义沿用现有风格。
- R2 前端：确认框与成功提示写清范围（例如「将把模型定价恢复为默认模板：倍率、完成倍率、缓存倍率、按次价格、
  计费模式与计费表达式；分组倍率不受影响」），七语言齐备。
- R3 回归测试（关键）：后端新增测试断言「重置后 `deepseek-flash` 与 `deepseek-v4.1-flash` 的
  倍率/完成/缓存/计费模式/计费表达式逐字符相同」，且重置前若人为把 `deepseek-v4.1-flash` 的表达式删掉，
  重置后必须恢复（**先写失败用例再修**的既有约定）。
- R4 门禁：`go build ./...`、`go vet ./...`、`go test ./service/... ./controller/... ./setting/... -count=1`；
  前端 `bun run typecheck`、`bun run i18n:check`、相关测试。
- R5 安全/兼容：不改变计费计算逻辑本身（只改重置写入哪些键）；老实例点一次重置即可对齐两行。
- R6 发版：用户已授权「修完发版」（走既有 tag → CI → Release → 镜像流程），发版后本机实例按镜像升级。

## Acceptance Criteria

- [ ] AC1 重置接口写入 R1 列出的全部键，且都取代码默认值（有测试断言）。
- [ ] AC2 回归：人为破坏 `deepseek-v4.1-flash` 的表达式/模式后执行重置 → 两行配置逐字符一致。
- [ ] AC3 确认框与提示文案更新，七语言齐备，`i18n:check` 通过。
- [ ] AC4 门禁全过（Go + 前端），有命令与输出证据。
- [ ] AC5 独立验证通过（非实现者自证）。
- [ ] AC6 发版并升级本机实例；在面板上实测「重置」后两行一致（含截图/接口比对证据）。

## Notes

- 立即绕过（无需等发版）：在受影响实例的面板里编辑 `deepseek-v4.1-flash` 行 → 计费模式选「表达式」→
  粘贴与 `deepseek-flash` 逐字符相同的表达式。
- 另一实例（生产）本轮不动，需点名主机；它会因为这次修复在下次升级后受益。
