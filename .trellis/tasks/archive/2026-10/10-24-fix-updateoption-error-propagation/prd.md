# 修 model.UpdateOption 吞异常 + 对齐本机 latest

## Goal

用户点名（2026-10-24）：
1. 修全站底座 `model.UpdateOption` 吞掉 `DB.Save` 错误的缺陷——写库失败时接口仍可能返回成功；
2. 本机 docker `latest` 对齐到 v0.0.47（registry 的 `latest` 已是它）。

修完需测试与独立验证；**发版与否待定**（不自动发）。

## 现状（代码事实）

`model/option.go` 的 `UpdateOption`（约 :218-235）当前实现：

```go
DB.FirstOrCreate(&option, Option{Key: key})   // 返回值被丢弃
option.Value = value
DB.Save(&option)                              // 返回值被丢弃
return updateOptionMap(key, value)            // 只有这里的错误会返回
```

实测（上一任务证据）：用 SQLite 触发器强制 `DB.Save` 失败时，`DB.Save` 报错但 `UpdateOption` 返回 `nil`，
调用方（如重置定价接口）会把失败当成功返回。同文件的 `UpdateOptionsBulk` 已是正确写法（事务 + 校验错误），
可作为风格参照。

## Requirements

- R1 `UpdateOption` 必须传播数据库错误：
  - `FirstOrCreate` 的错误要返回；
  - `Save` 的错误要返回；
  - **写库失败时不得更新内存**（`updateOptionMap` 不得被调用），避免 DB 与内存不一致；
  - 错误信息要能定位到 key（但不泄露值内容；值可能是配置，按项目既有风格处理）。
- R2 调用方兼容性：审查全仓 `UpdateOption(` 调用点，确认返回值被处理或按既有习惯记录日志；
  若有调用方依赖「永远不返回 DB 错误」的隐含假设，逐一说明并保持行为可预期（默认应是如实返回错误）。
- R3 测试（先写失败用例）：
  - 强制 DB 写失败（例如 SQLite 触发器，参考上一任务做法）→ `UpdateOption` 必须返回 error，且内存值保持旧值；
  - 正常路径仍成功且内存与 DB 一致；
  - 重置定价接口在写库失败时返回 `success:false`（控制器层用例，可复用/扩展现有 `controller/pricing_test.go`）。
- R4 不得改变 `UpdateOptionsBulk` 语义；不得顺手重构无关函数。
- R5 门禁：`go build ./...`、`go vet ./...`、`go test ./service/... ./controller/... ./setting/... ./model/... -count=1`（附输出）。
- R6 latest 对齐：`docker tag ...:v0.0.47 ...:latest`，核验 `latest == v0.0.47`，`v0.0.46`/`v0.0.45` 保留，实例零影响。
- R7 红线：不动其它容器/镜像/卷；不删 `data/`；不提交不推送、不发版。

## Acceptance Criteria

- [ ] AC1 `UpdateOption` 在 `FirstOrCreate`/`Save` 失败时返回 error，且内存未被更新（有测试）。
- [ ] AC2 正常路径行为不变（DB 与内存一致）。
- [ ] AC3 重置定价接口在写库失败时如实返回失败（有控制器层用例）。
- [ ] AC4 全仓调用点审查结论落地（哪些调用方会看到新错误、为何可接受）。
- [ ] AC5 门禁全过，有命令与输出。
- [ ] AC6 本机 `latest` 与 `v0.0.47` 同 ID；`v0.0.46`/`v0.0.45` 仍在；实例 200 + v0.0.47 零影响。
- [ ] AC7 独立验证通过（非实现者自证）。

## Notes

- 这是**全站共用底座**：写库失败现在会冒泡，属行为变更（更正确），需在证据里写清影响面与回滚方式（单函数 revert）。
- 发版决策留待用户；本轮只做代码 + 测试 + 验证。
