# 清 29 个内部面导出符号 + B 堆 4 项接线

## Goal

用户点名（2026-10-24）：
1. 清 **A 堆内部面 29 个导出符号**（`pkg` 13 + `oauth` 2 + `types` 2 + `setting` 12，**不动 `relaykit`**，其 20 个需 owner 决策）；
2. 修 **B 堆 4 项**：`common.IsPrivateIP`、`common.GetContextKeyStringSlice` 接线；`relaykit/kitutil` 两个 JSON 函数定策略。
3. **不发版**（用户明确：目前没必要那么频繁发版，先攒着解决问题）。

## 参考清单

`.trellis/tasks/archive/2026-10/10-24-panic-recovery-and-cleanup/evidence-symbols.md`
（85 条逐符号表：A 76 / B 4 / C 5；按包统计与建议执行顺序）。注意该清单是上一轮**重跑 deadcode 重建**的
（更早那份归档证据 §5.2 正文为空、内容丢失），因此以重建版为准。

## Requirements

### R1 清 29 个内部面导出符号

- 对象：A 堆中的 `pkg/*` 13 个、`oauth/*` 2 个、`types/*` 2 个、`setting/*` 12 个。
- 每个符号删前**四类核对**（同前两轮标准）：① `deadcode -test` 判不可达；② 全仓 Go（含 `_test.go`/构建标签）无引用；
  ③ 非 Go 位置无引用（README/MAINTENANCE/AGENTS.md/CLAUDE.md/`.trellis/spec/**`/scripts/.github/web）；
  ④ 非反射/linkname/注册表字符串（`setting/` 尤其注意：可能有配置键名字符串引用）。
- **`setting/` 特殊规则**：若某 getter 是同类配置的成组扩展点（清单已提示「可整体归 C」），**保留**并说明；
  只删确实孤立无用者。
- 分批（每批 ≤10），每批后 `go build ./...` + `go test ./... -count=1`；失败回退该批。
- 完成后重跑 `deadcode -test`：记录数字变化与级联（预期 85 → 约 56）。

### R2 B 堆接线（4 项）

- **B1 `common.IsPrivateIP`**：`common/ssrf_protection.go` 内联了等价私网/回环判定
  （约 :100-137 的 `IsUnspecified`/`IsLoopback`/`IsLinkLocal*`/`privateIPv4Nets`/`privateIPv6Nets`/`IsPrivate` 组合）。
  抽成 `IsPrivateIP(ip net.IP) bool` 并让调用点复用它，消除两套等价逻辑；**行为必须逐条等价**（补测试覆盖各分支，
  含 IPv4/IPv6/未指定/回环/链路本地/私网/公网）。
- **B2 `common.GetContextKeyStringSlice`**：`service/group.go:96-107` 的 `GetRequestAutoGroups` 手工
  `common.GetContextKey` + `value.([]string)` + 类型断言失败返回空切片。改用该 helper，
  保持三分支语义完全一致（缺失 → 继承全局；存在但类型不符 → 空切片；存在且合法 → 过滤），补/复用测试。
- **B3 `relaykit/kitutil` 的 `UnmarshalJsonStr` 与 `DecodeJson`（2 项）**：二选一并说明理由：
  ① **接线**：relaykit 内 14 处直调 `encoding/json` 改为走 kitutil 包装（范围较大，需评估）；
  ② **删除**：若 relaykit 有意直连标准库、且这两函数无使用者，删掉并在 `relaykit/README.md`
     或 `.trellis/spec/backend/quality-guidelines.md` 注明「relaykit 独立模块的 JSON 例外」。
  倾向 ②（更小、更诚实），但需先核对 AGENTS.md:76-77 的约定是否要求 relaykit 也走包装；结论写进证据。
  注意：`relaykit` 是独立模块，任何改动都要 `cd relaykit && GOWORK=off go build ./...` 与 `go test ./...`。

### R3 统一约束

- 不改运行时行为（除 R2 的等价重构）；不改部署形态；不动 `data/`、实例；**不发版**。
- 门禁：`go build ./...`、`go vet ./...`、`go test ./... -count=1`、`cd relaykit && GOWORK=off go build ./...`、
  `gofmt -l .`、`golangci-lint run --default=none --enable=unused ./...`（应仍恰 1，`logKeyCol`）。
- 不提交不推送（Lead 收口）。

## Acceptance Criteria

- [ ] AC1 29 个符号处置完毕（已删 / 保留并说明），逐符号四类取证表齐备。
- [ ] AC2 每批后全量测试绿；`deadcode` 数字变化与级联有记录；`unused` 仍为 1；`gofmt` 空。
- [ ] AC3 B1/B2 接线完成且行为等价（有测试）；B3 有明确策略与理由，relaykit 独立构建/测试通过。
- [ ] AC4 门禁全过（含 relaykit 独立模块），有命令与输出。
- [ ] AC5 独立验证通过（非实现者自证）。

## Notes

- 本轮**不发版**：改动攒在 `main`，等用户决定发版时机（会与下一批一起发）。
- `setting/` 的 getter 若成组保留，需在证据里写明「按扩展点保留」而非遗漏。
