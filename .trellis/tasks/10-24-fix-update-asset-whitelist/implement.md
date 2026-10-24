# 执行计划：自更新资产白名单缺陷修复（A+B+C）

## 顺序与责任

| # | 步骤 | 负责 | 产出 | 验证 |
|---|---|---|---|---|
| 1 | 后端 A+B：裁剪移出 fetcher、响应处裁剪、白名单含 `.sig` | worker-script | `service/update_check.go` 等 | `go build ./...`、`go vet ./...`、`go test ./service/...` |
| 2 | 后端 C1+C3：新错误码 + 日志 + 回归测试 | worker-script | `service/update_apply.go`、`*_test.go` | 新用例修复前失败/修复后通过（附对照输出） |
| 3 | 前端 C2：错误码映射 + 七语言文案 | worker-docs | `update-api.ts`、`update-checker-section.tsx`、7 个 locale | `bun run typecheck`、`bun run i18n:check` |
| 4 | 独立验证 | verifier | 复核报告 | 见下方检查清单 |
| 5 | Lead 复核 + 提交（锚点 `[task:fix-update-asset-whitelist]`）+ 收尾 | Lead | 提交与 journal | `scripts/trellis-wrapup.sh` |

## 验证命令（必须实际执行）

```bash
# Go
go build ./...
go vet ./...
go test ./service/... -count=1
# 前端
cd web && bun run typecheck && bun run i18n:check
```

## 审查门（Lead 与 verifier 共同）

1. applier 路径确实能看到 `SHA256SUMS.sig`（不是靠 applier 绕过，而是 fetcher 不再裁剪）。
2. 状态响应体形状与修复前一致（前端零字段改动）。
3. 新错误码只在「资产缺失」出现；T2–T5 仍为 `signature_mismatch`。
4. 日志不含签名/公钥内容，但含缺失资产名。
5. 七语言文案齐备且 key 一致。
6. 回归测试能证明「修复前会失败」——必须有修复前对照证据，否则视为未验证。
7. fail-closed 语义无变化（未签名资产仍被拒）。

## 回滚点

- 后端与前端改动各自独立，必要时可分别 revert；回滚后行为等价于 v0.0.45（面板更新仍不可用）。

## 不做（避免范围蔓延）

- 不发版、不动发布工作流、不改部署形态、不加新的更新源、不改 UI 布局。
