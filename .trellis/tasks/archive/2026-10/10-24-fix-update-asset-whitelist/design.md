# 技术设计：自更新资产裁剪与错误码

## 1. 边界

| 层 | 职责（修复后） | 不改的东西 |
|---|---|---|
| fetcher（`githubReleaseFetcher.fetch`） | 拉取 Release、校验 draft/prerelease/tag，**返回完整资产列表**（不再裁剪） | Release 解析结构、超时与代理逻辑 |
| 状态响应组装（`controller/update_check.go` / `service/update_check.go` 的响应构建处） | 只在这里裁剪成前端需要的形状（**保持今天的响应字段与顺序**） | 响应 JSON 形状（前端零改动） |
| applier（`service/update_apply.go`） | 按既有顺序要求 binary / SHA256SUMS / SHA256SUMS.sig 三类资产；缺失走**新错误码** | fail-closed 语义、验签在下载二进制之前 |

## 2. 契约

- **错误码**：新增 `UpdateApplyCodeSignatureAssetMissing = "signature_asset_missing"`（HTTP 502）；
  仅在「签名资产缺失」时返回。其余签名类失败（公钥为空、签名长度非法、验签不通过、公钥不可解析）
  继续返回 `signature_mismatch`。
- **日志**：缺失分支调用 `common.SysError`，消息含缺失资产名（用 `common.MaskSensitiveInfo` 包裹动态信息，
  与既有风格一致）。**不得**把签名字节或公钥内容写进日志。
- **前端映射**：`web/src/features/system-settings/maintenance/update-api.ts` 的错误码联合类型与映射表新增该项，
  指向新的 i18n key；`update-checker-section.tsx` 增加对应分支。
- **i18n**：英文源串 `The release is missing its signature asset, so the replacement was refused.`，
  七语言（en、zh、zh-TW、fr、ru、ja、vi）全部补齐。

## 3. 测试策略（补上漏网的那一层）

1. **单测（裁剪）**：`filterUpdateAssets`（若保留）或响应组装裁剪函数——断言保留 4 类资产（含 `.sig`），
   且不误留无关资产。
2. **直通回归（关键）**：用 `httptest` 起一个假 GitHub Release（资产含 binary + SHA256SUMS + SHA256SUMS.sig），
   经由 **fetcher** 得到 Release 后交给 applier 的校验入口，断言：
   - 修复前：在 T1 处失败（`signature_mismatch`）；
   - 修复后：越过资产检查、进入验签阶段（用故意错误的签名让它停在验签失败，以此证明「已不是资产缺失」）。
3. **缺失用例**：假 Release 不含 `.sig` → 断言 `signature_asset_missing` + 日志含资产名。
4. **fail-closed 回归**：既有 T2–T5 用例保持全绿，确认没有降级放行。

## 4. 权衡

- 为什么不在 applier 里自己带上 sig 资产名？——那只是绕过症状；裁剪与取用的**职责分离**（B）才是根因修复。
- 为什么不把状态响应也带上 sig 资产？——前端不需要，扩大响应体没有必要；B 的做法是「fetcher 全量、响应裁剪」。
- 为什么不把 T1 也归到 `signature_mismatch`？——文案误导是本次误诊的直接原因，独立错误码成本极低。

## 5. 兼容与回滚

- 兼容：老 Release 无 `.sig` → 仍拒绝（安全语义不变），但错误码/文案准确。
- 回滚：改动集中在 3 个 Go 文件与 2 个前端文件；若发现问题可整体 revert，不影响发布侧资产与其它功能。
