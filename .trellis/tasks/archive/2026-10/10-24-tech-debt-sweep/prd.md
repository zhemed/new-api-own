# 技术债清扫（一轮、低风险优先）

## Goal

用户点名（2026-10-24）：继续清扫技术债。本轮**限定为一轮、低风险优先**：
能安全修的当场修，可疑或有影响面的**只出清单**，不擅自动手。

## 范围（本轮做）

### 前端（`web/`）

- W1 **死 i18n key**：上一轮改动后 `Model prices reset successfully`、`This will clear custom pricing ratios and revert to upstream defaults.`
  已无引用但仍留在 7 个 locale 文件（各 2 行）。清理方式必须遵守项目既有流程
  （只删无引用键、不重排既有键；改完跑 `bun run i18n:sync` 恢复混淆键；`bun run i18n:check` 必须通过）。
- W2 **静态检查**：全仓 `bunx oxlint` 收集 warning/error 清单；error 必须修，warning 分类列出
  （低风险且明确的顺手修，其余进报告）。
- W3 `bun run typecheck` 与 `bun run i18n:check` 必须通过（基线）。

### 后端（Go）

- B1 `gofmt -l .`、`go vet ./...`：必须干净（当前基线应已是），有输出则修。
- B2 **TODO/FIXME 清单**：`grep -rn "TODO\|FIXME\|XXX\|HACK"` 汇总为报告（文件、行、内容、是否本仓库自己写的、
  建议处置），**不擅自改**。
- B3 **死代码候选**：至少检查「只有定义没有调用」的导出符号（轻量手段即可，如 grep 调用点），
  列出清单与证据；**不擅自删**（影响面可能大）。
- B4 **脚本**：`scripts/*.sh` 语法检查（`bash -n`，若 shellcheck 已装则一并跑），报告问题；
  低风险项（引号/未定义变量）可修。

### 仓库一致性

- C1 版本引用一致性：`VERSION`（0.0.48）与 README / MAINTENANCE / 其它文档中的版本号是否过时。
- C2 明显无用文件：是否残留临时调试文件、备份文件、`*.orig` / `*.rej` / `.DS_Store` 之类（只报不改，除非明确是误提交）。
- C3 部署形态红线复查：`sh scripts/forbid-extra-deploy-methods.sh` 通过。

## 不在本轮范围

- 重构类改动（拆函数、改接口、改数据模型）——只进报告。
- 依赖升级（`bun update`、Go 依赖升级）。
- 发布/升级实例。
- 任何需要用户新授权的外部动作。

## Requirements

- R1 前端 W1/W2/W3、后端 B1/B4、仓库 C3 必须完成；W2/B2/B3/C1/C2 产出清单。
- R2 所有改动必须**可回滚**且**不改变运行时行为**（死键、格式、脚本语法修复属安全类）。
- R3 门禁：`cd web && bun run typecheck && bun run i18n:check && bunx oxlint`；Go `go build ./...`、`go vet ./...`、
  `go test ./... -count=1`；`sh scripts/forbid-extra-deploy-methods.sh`。
- R4 红线：不动 `data/`、不动实例、不发版、不提交不推送（由 Lead 收口）。

## Acceptance Criteria

- [ ] AC1 死 i18n key 清理完成，7 语言一致，`i18n:check` 通过。
- [ ] AC2 `oxlint` error 清零，warning 有分类清单。
- [ ] AC3 Go 格式/vet 干净；TODO/FIXME 与死代码候选有清单（含证据与建议）。
- [ ] AC4 脚本问题有清单，低风险项已修。
- [ ] AC5 版本引用与仓库残留文件有结论。
- [ ] AC6 门禁全过（前端 + Go + 部署形态红线），有命令与输出。
- [ ] AC7 独立验证通过（非实现者自证）。

## Notes

- 目标不是"清零一切"，而是**让门禁与清单可信**：能修的修、不能修的写清楚，避免下一轮重复发现。
