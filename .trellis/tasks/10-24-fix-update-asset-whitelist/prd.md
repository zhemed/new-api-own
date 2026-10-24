# 修复自更新资产白名单缺陷（A+B+C）

## Goal

修复「面板自更新自 v0.0.43 起 100% 失败」的缺陷（诊断见已归档任务
`archive/2026-10/10-24-diagnose-update-signature-mismatch`），让面板内「立即更新」重新可用，
并消除同类漂移与误诊风险。

三项全做（用户 2026-10-24 点名）：
- **A** 白名单补回 `SHA256SUMS.sig`；
- **B** 把资产裁剪移出**共用 fetcher**，只在状态响应组装处裁剪；
- **C** 缺失资产补日志 + 独立错误码与专用文案（七语言）+ fetcher→applier 集成回归测试。

## 根因（已证实的证据链）

| 环节 | 事实 | 位置 |
|---|---|---|
| 裁剪 | 共用 fetcher 用白名单裁剪资产（容量 3），漏掉 `SHA256SUMS.sig` | `service/update_check.go:276`、`:289-299` |
| 取用 | applier 在下载任何东西之前就要求签名资产存在 | `service/update_apply.go:207-212` |
| 结果 | 条件恒假 → 稳定 `signature_mismatch`（502），前端映射为「签名与公布的校验值不匹配」 | `service/update_apply.go:30`、`web/.../update-api.ts:123`、`zh.json:4358` |

引入点：提交 `813d5c4`（v0.0.43）；v0.0.43/44/45 同一份实现。发布侧资产齐全且自洽（已核）。

## Requirements

- R1 **A**：`filterUpdateAssets`（或等价裁剪函数）必须保留 `SHA256SUMS.sig`；保留项用既有常量
  `UpdateAssetNameSHA256SumsSig`，不硬编码字符串。
- R2 **B**：裁剪**不得**发生在 applier 会读到的路径上。fetcher 返回完整资产列表；裁剪只在
  「状态响应组装」处进行，且**响应体形状与今天保持一致**（前端无需改字段）。
- R3 **C1 可观测**：签名资产缺失时，用**独立错误码**（建议 `signature_asset_missing`）替代当前的
  `signature_mismatch`，并用 `common.SysError` 记录缺失的资产名；其余 T2–T5 仍按 fail-closed 归 `signature_mismatch`。
- R4 **C2 前端**：新错误码映射到**专用文案**（英文源串建议
  `The release is missing its signature asset, so the replacement was refused.`），
  并补齐 `web/src/i18n/locales/` 下 en、zh、zh-TW、fr、ru、ja、vi 七个文件；
  同时确认 `update-checker-section.tsx` 的分支能展示该文案。
- R5 **C3 回归测试**：新增「fetcher → applier」直通测试——用 httptest 伪造 Release（含 sig 资产），
  断言 applier 能走到**验签阶段**而不是在 T1 资产缺失处失败；另加一条「真的没有 sig 资产」用例，
  断言返回新错误码。现有测试都绕开 fetcher，这是缺陷漏网的原因，必须在断言层面堵住。
- R6 安全语义不变：**fail closed**——未签名、验签失败、公钥不可用等一律拒绝替换，不得降级放行。
- R7 兼容性：老 Release（无 `.sig`）仍按设计拒绝，但错误码与文案要能说明「资产缺失」而非「签名不匹配」。
- R8 门禁：改动后跑 Go 侧 `go build ./...`、`go vet ./...`、相关包测试（`service/...` 等）；
  前端跑 `bun run typecheck`、`bun run i18n:check`；不得引入 lint 错误。
- R9 红线：不提交不推送由成员执行；不访问外部主机；不改部署形态；本轮**不发版**（发版由用户点名后另行执行）。

## Acceptance Criteria（收尾回填 2026-10-24）

- [x] AC1 A 修复：fetcher 不再裁剪资产、`filterUpdateAssets` 已删除（全仓零残留）；
  `TestGithubReleaseFetcherParsesStableRelease` 由「期望 2 资产」改为断言**完整列表**（原断言正是锁死 bug 处）。
- [x] AC2 B 修复：applier 路径拿到的 Release 含 `.sig`（直通用例实测 applier 真的请求了 `.../SHA256SUMS.sig`，修复前是 `[]string(nil)`）；
  `UpdateCheckStatus` 响应结构与 HEAD **逐字节相同**（本就不含 assets 字段）。
  **实现偏差已接受**：代码里不存在「状态响应组装处」可挂裁剪，故按 B 的意图改为「fetcher 全量 + 删除裁剪函数」，
  效果等价且更简单。
- [x] AC3 C1：缺 `.sig` → 新码 `signature_asset_missing`（502）+ `common.SysError` 记缺失资产名；
  T2–T5 仍 `signature_mismatch`；fail-closed 语义不变。
  **实现偏差已接受**：该日志未套 `MaskSensitiveInfo` —— Lead 与 verifier 双方实测
  `MaskSensitiveInfo("SHA256SUMS.sig") == "***.sig"`，包装会抹掉唯一诊断价值；写入的是编译期常量/公开文件名，同文件其它日志仍脱敏。
- [x] AC4 C2：前端新码映射 + 展示分支 + 七语言文案齐备；`i18n:check` rc=0（引用键 4186 → 4187），
  每 locale `+1/-0`、无重排；前后端字面量一致（大小写敏感）。
- [x] AC5 C3：新增 fetcher→applier 直通回归与缺 sig 用例，**修复前后对照有据**：
  Go 侧修复前 4 条失败 → 修复后 11 条全 PASS；前端修复前恰好 2 条失败 → 修复后 47 pass / 0 fail
  （verifier 在 `/tmp` 克隆内独立复现，未触碰真实工作区）。
- [x] AC6 安全语义不变：forge / checksum 不符 / 公钥不可用等 fail-closed 用例全绿，无降级放行。
- [x] AC7 门禁全过：`go build ./...`、`go vet ./...`、`go test ./service/... -count=1`（另跑全仓 `go test ./...`）、
  `bun run typecheck`、`bun run i18n:check`、`bun test .../maintenance/__tests__/`（70 pass）。
- [x] AC8 独立验证：verifier **接受**（`evidence-verify.md`），并额外补登记稳定 code 清单
  （`controller/update_check_endpoint_test.go:66` 新增一行，属测试清单补全）。

## 交付后的运维事实（必须让用户知道）

- 受影响实例（v0.0.43 / 44 / 45，含本机与用户提到的另一实例）**无法靠面板自更新到修复版**——
  bug 在实例体内。必须**先用镜像方式升级一次**（拉新版 tag → 按原参数重建容器），之后面板自更新恢复正常。
- 本轮**未发版**（PRD R9）：发版（tag / Release / 镜像）与升级实例均待用户点名后另行执行。

## Notes

- 本缺陷的**实际后果**：受影响实例（含本机与另一实例）面板内更新自 v0.0.43 起不可用；
  它们**无法靠面板自更新到修复版**（bug 在实例体内），必须先用镜像方式升级一次。
- 发版不在本任务内；修复合并后由用户点名再走发版流程。
