# 清死代码 + 文档版本号 + CI 格式检查 + deadcode 分析

## Goal

用户点名四项（2026-10-24）：
1. 清 **39 个未导出死符号**（上一轮 golangci-lint `unused` 的精确清单）；
2. 统一 README / MAINTENANCE 里的**陈旧示例版本号**（示例 tag 仍写 0.0.5/0.0.6，当前 VERSION=0.0.48）；
3. 把 **gofmt 与 shellcheck 检查接进 CI**，防止格式/脚本问题回流；
4. 安装 **`deadcode`** 工具做导出符号全程序分析（用户已点头，含联网下载）。

## Requirements

### R1 清死代码（后端）

- 对象 = 上一轮 `golangci-lint run --default=none --enable=unused ./...` 的 39 个**未导出**符号。
- **逐符号取证**：删除前确认该符号在仓库内**无任何引用**（含 `_test.go`、构建标签、反射字符串、模板/脚本引用）；
  证据留表（符号 → 包 → 引用数 0 的取证命令与输出）。
- **分批删除**：按包分批（建议每批 ≤ 10 个符号），每批后跑 `go build ./...` + `go test ./... -count=1`，
  任一批失败即回退该批并报告。
- 删完后 `golangci-lint ... unused` 应为 0 issues（未导出）；`gofmt -l .` 仍空。
- 若某符号看似死代码但实际是**接口实现/反射/构建标签**所需 → **保留并在报告说明**（不得为清数字误删）。

### R2 文档示例版本号（文档）

- 只改**示例值**：README / MAINTENANCE 中「示例 tag / 示例命令」里的 `0.0.5`、`0.0.6` 等陈旧版本，
  统一到 `0.0.48` 或改成明确的占位符（如 `X.Y.Z`），**同一文档内保持统一风格**。
- **不改**：功能门槛（"某能力自 v0.0.x 起"）、历史事故记录、Release 说明里的历史版本。
- 每处改动在证据里列出「原文 → 新文 + 分类（示例/门槛/历史）」；渲染安全检查（无成对美元符号、围栏配平）。

### R3 CI 格式与脚本检查（CI）

- 在 `.github/workflows/` 增加或扩展现有工作流，加入：
  - `gofmt -l .`（排除 vendor/生成物；有输出即 fail）；
  - `shellcheck -S warning scripts/*.sh`（CI runner 自带 shellcheck；若没有则显式安装并说明固定版本）；
  - 可选：`bash -n scripts/*.sh`（廉价双保险）。
- 要求：**不破坏现有 workflow**（`trellis-gate.yml` 等职责不变）；新增检查必须**在本地先验证等价命令通过**；
  workflow YAML 语法校验（`python3 -c "import yaml,sys;yaml.safe_load(open(...))"` 或 actionlint 若已装）。
- 说明预期失败行为：故意造一个 gofmt 不干净的文件 → 本地同一命令确实返回非零（证明检查有效，验证后清理）。

### R4 deadcode 分析（工具 + 报告）

- 安装 `golang.org/x/tools/cmd/deadcode`（用户已授权联网）。安装方式与版本记录在证据里
  （`go install ...@latest` 或固定版本；记录版本号）。
- 运行方式注意 `deadcode` 需要入口（`-test`、或指定 main 包）；把命令、输出、以及**误报甄别**写进报告。
- 产出：导出符号死代码清单（按包分组，标注「可删 / 需确认 / 误报」三类），**本轮不删导出符号**。
- 若 39 个未导出符号在 deadcode 输出中重复出现，交叉核对两套工具结论一致性。

### R5 统一约束

- 不改运行时行为（死代码删除除外，那是无引用代码）；不改部署形态；不动 `data/`；不动实例。
- 不提交不推送、不发版（由 Lead 收口）。
- 门禁：`go build ./...`、`go vet ./...`、`go test ./... -count=1`、`cd relaykit && GOWORK=off go build ./...`；
  前端 `bun run typecheck`、`bun run i18n:check`；`sh scripts/forbid-extra-deploy-methods.sh`。

## Acceptance Criteria

- [ ] AC1 39 个未导出死符号处置完毕（已删 / 保留并说明），有逐符号取证表。
- [ ] AC2 每批删除后全量测试绿；最终 `unused` 为 0、`gofmt -l` 空。
- [ ] AC3 文档示例版本号统一，分类清单齐备，未误改功能门槛与历史记录。
- [ ] AC4 CI 新增检查落地且本地等价命令验证通过；故意违规能 fail；YAML 合法；现有 workflow 未破坏。
- [ ] AC5 `deadcode` 安装成功并产出导出符号清单（分三类），与 unused 结论交叉核对。
- [ ] AC6 门禁全过（含 relaykit 独立构建与前端检查），有命令与输出。
- [ ] AC7 独立验证通过（非实现者自证）。

## Notes

- 删除类改动是本项目里风险最高的类型：宁可少删、不可误删。取证表是硬要求。
- R3 的 CI 改动会进 `main`，必须先在本地把等价命令跑通再合并。
