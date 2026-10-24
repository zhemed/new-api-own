# 修 setup.go 内存赋值顺序 + 发 v0.0.48 并升级实例

## Goal

用户点名（2026-10-24）：
1. 修 `controller/setup.go` 在**落库前**就改内存值的问题（DB 写失败时接口现在会如实报错，但内存已被改，DB 与内存分叉）；
2. 发 **v0.0.48**（含上一轮的底座 `UpdateOption` 错误传播修复 + 本次顺序修复）并镜像升级本机实例。

## 现状（代码事实）

`controller/setup.go:124-126`：

```go
// Set operation modes
operation_setting.SelfUseModeEnabled = req.SelfUseModeEnabled   // ← 先改内存
operation_setting.DemoSiteEnabled = req.DemoSiteEnabled         // ← 先改内存

err = model.UpdateOption("SelfUseModeEnabled", ...)  // 再落库；失败则这里 return
...
err = model.UpdateOption("DemoSiteEnabled", ...)
```

而 `model/option.go:376-379` 的 `updateOptionMap` **本来就会**在这两个键落库成功后同步
`operation_setting.SelfUseModeEnabled` / `DemoSiteEnabled`。也就是说 setup.go 里那两行手工赋值
**既冗余、又制造了顺序问题**（DB 失败时内存已变）。

## Requirements

- R1 主修复：删除 `controller/setup.go` 中落库前的两处手工内存赋值，改由 `model.UpdateOption` 成功后的
  `updateOptionMap` 同步（需先确认该路径确实覆盖这两个键，现有代码行号 `model/option.go:376-379`）。
- R2 建议一并做（若改动可控）：把两次 `model.UpdateOption` 换成 `model.UpdateOptionsBulk`，
  让 "SelfUseModeEnabled + DemoSiteEnabled" 两个键**原子写入**（避免第一个成功、第二个失败导致的半应用状态）；
  错误文案保持语义（区分写库失败），不要吞错。若判断风险大于收益，可只做 R1 并在证据里说明理由。
- R3 测试（先写失败用例）：模拟 DB 写失败（沿用底座测试里的 SQLite 触发器手法）→ 调用 setup 写入路径 →
  断言：接口返回失败、**内存值保持旧值**（这是本次修复的核心断言）、DB 未被写入。
- R4 门禁：`go build ./...`、`go vet ./...`、`go test ./controller/... ./model/... -count=1`，并跑一次 `go test ./... -count=1`。
- R5 发版：`VERSION` → `0.0.48`，附注 tag `v0.0.48` 写 changelog（含上一轮底座修复 + 本次顺序修复），
  推 main 与 tag，双 workflow 必须双绿，Release 四资产齐全（含 `SHA256SUMS.sig`），镜像多架构就位。
- R6 升级本机实例：按原参数重建到 v0.0.48，核验 running / `restarts=0` / 工作区挂载 / 200 + `v0.0.48` /
  8 容器全 Up / 卷 3 / `data/` 文件数不减；`v0.0.47`/`v0.0.46` 保留作回滚点。
- R7 红线：不动其它容器/镜像/卷；不删 `data/`；不 force push；不改部署形态。

## Acceptance Criteria

- [ ] AC1 setup.go 不再在落库前改内存；DB 失败时内存保持旧值（有测试断言）。
- [ ] AC2 成功路径行为不变（内存与 DB 一致，setup 仍能完成初始化）。
- [ ] AC3 若采纳 R2：两个键原子写入；未采纳则证据里写明理由。
- [ ] AC4 门禁全过，有命令与输出。
- [ ] AC5 独立验证通过（非实现者自证）。
- [ ] AC6 v0.0.48 发布完成：CI 双绿、Release 四资产（含 `.sig`）、镜像就位。
- [ ] AC7 本机实例升到 v0.0.48 且健康；上一版本镜像保留作回滚点。

## Notes

- 本任务同时承载"上一轮底座修复发版"与"本次顺序修复"，两者合并为一次发版，避免连发两版。
- 发版属外部副作用动作，用户本轮已点名授权。
