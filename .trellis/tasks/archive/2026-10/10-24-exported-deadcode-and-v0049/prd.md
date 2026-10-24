# 盯 lint CI + 清 62 个可删导出符号 + 发 v0.0.49

## Goal

用户点名三件（2026-10-24）：
1. 确认新加的 `lint` workflow 首次真实运行是否绿；
2. 分批清 **62 个 deadcode「可删」导出符号**（每批全测试）；
3. 发 **v0.0.49**（含本轮死代码删除、CI 工作流、文档版本号）并镜像升级本机实例。

## 进展（Lead 已做）

- **R1 已完成**：gh run `40067689123`「lint (gofmt + shellcheck)」**completed success**，
  同批 `trellis-gate` 也 success → 新 CI 在真实 runner 上工作正常（含 runner 的 shellcheck 版本无额外告警）。

## Requirements

### R2 清 62 个「可删」导出符号（后端）

- 对象 = 上一轮 `deadcode -test ./...` 报告里标记为「可删」的 62 个**导出**符号（证据见
  `.trellis/tasks/archive/2026-10/10-24-deadcode-docs-ci/evidence-deadcode.md` §5.1）。
- **先复核再删**：每个符号必须同时满足
  ① `deadcode` 判定不可达；② 全仓 Go（含 `_test.go`、构建标签）无引用；
  ③ **非 Go 位置也无引用**（README / MAINTENANCE / `AGENTS.md` / `.trellis/spec/**` / `scripts/` / `.github/` / `web/`）；
  ④ 不是反射/`go:linkname`/注册表字符串（如 GORM 钩子、casbin、webauthn、插件名）。
  任一不满足 → **保留并写入报告**（文档/规范里出现的尤其要留，避免 spec 腐化）。
- **分批删除**（每批 ≤ 10），每批后 `go build ./...` + `go test ./... -count=1`；失败即回退该批。
- 完成后重跑 `deadcode -test ./...`：删除可能产生**级联**新不可达符号，需说明本轮是否已清零、剩余多少。
- `unused`（未导出）应仍为 1（`logKeyCol`），不得因本轮新增。
- `gofmt -l .` 必须空。

### R3 发版 v0.0.49

- `VERSION` → `0.0.49`；附注 tag `v0.0.49`（注解即 Release 正文），内容覆盖：
  重置定价修复（v0.0.47 已发，可只做历史提及）、`UpdateOption` 错误传播、setup 顺序修复、
  测试隔离修复、死代码清理（未导出 36 + 本轮导出若干）、新增 lint CI、文档示例版本号。
- 推送 `main` + tag，两个发布 workflow 必须双绿；Release 四资产齐全（含 `SHA256SUMS.sig`）；
  镜像多架构就位；**必须确认新的 lint workflow 也绿**。

### R4 升级本机实例

- 按原参数重建到 v0.0.49，核验：running / `restarts=0` / 工作区挂载 / 200 + `X-New-Api-Version: v0.0.49` /
  8 容器全 Up / 卷 3 / `data/` 文件数不减；保留 v0.0.47 / v0.0.48 与 v0.0.46 作回滚点。
- 升级后复核上一轮修复仍在（`billing_setting.*` 两行 `tiered_expr` + 表达式同 sha）。

### R5 统一约束

- 不改运行时行为（死代码删除除外）；不改部署形态；不动 `data/`；不 force push。
- 门禁：`go build ./...`、`go vet ./...`、`go test ./... -count=1`、`cd relaykit && GOWORK=off go build ./...`、
  `gofmt -l .`、`unused` = 1、前端 `bun run typecheck` / `bun run i18n:check`、`sh scripts/forbid-extra-deploy-methods.sh`。
- 独立验证（非实现者自证）后再发版。

## Acceptance Criteria

- [ ] AC1 lint CI 首次运行结论已记录（run id + 结论）。
- [ ] AC2 62 个符号处置完毕（已删 / 保留并说明），逐符号取证表齐备（含非 Go 位置核对）。
- [ ] AC3 每批删除后全量测试绿；`deadcode -test` 重跑结论（含级联说明）；`unused` 仍为 1；`gofmt` 空。
- [ ] AC4 独立验证通过。
- [ ] AC5 v0.0.49 发布完成：CI（发布 + lint）双绿/三绿、Release 四资产含 `.sig`、镜像就位。
- [ ] AC6 本机实例升到 v0.0.49 且健康；回滚镜像保留；上一轮定价修复仍在。

## Notes

- 导出符号比未导出更危险：删错会破坏下游/文档约定；**宁可少删**，取证表是硬要求。
- 本轮同时承载「发版」与「删代码」，务必先验证通过再打 tag。
