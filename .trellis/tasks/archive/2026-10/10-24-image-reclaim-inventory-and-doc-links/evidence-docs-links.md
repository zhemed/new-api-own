# evidence-docs-links：补 wrapup 互链 + 自证清单加 --dry-run

- 任务板：`task-15`（Trellis 任务 `10-24-image-reclaim-inventory-and-doc-links`，PRD 见同目录 `prd.md`）
- 执行者：`worker-docs`；基线 HEAD `e6ec2d8`（2026-10-24）
- 写作用域内改动：`MAINTENANCE.md`、`.trellis/spec/guides/trellis-gate-guide.md`（+ 本证据文件）
- 未改代码/脚本/配置；未提交、未推送

## 一、结论

1. `MAINTENANCE.md`「流程闸门」小节新增一行指向 `./scripts/trellis-wrapup.sh`，
   写明它把 `finish → archive → push → 校验` 固化为一条命令，并指出动机（归档后本会话已无任务、补推会被闸门拦；
   也不必再建临时任务），细节指向闸门指南第 9 节。风格与该节一致（纯文字 + 行内代码路径，该文件全文不使用 Markdown 链接）。
2. `trellis-gate-guide.md` 第 6 节自证清单新增第 4 项：收尾前先跑
   `./scripts/trellis-wrapup.sh <任务名> --dry-run` 看将要执行的动作（零改动、跳过网络预检），再真跑。
   编号/缩进/句尾分号与既有条目一致，且插在「check-trellis-gate.sh 通过」与「任务已 finish + archive」之间，
   保持"先自检、再收尾"的顺序。
3. **第 9 节 vs 脚本行为核对**：逐项比对用法、阶段、守卫、`--dry-run` 语义、失败恢复，**发现 1 处不一致并已按脚本修正**——
   第 9 节守卫行漏写了脚本实际存在的「任务名匹配到多个目录」拒绝分支（`scripts/trellis-wrapup.sh:112-114`），
   已补进守卫单元格；其余各项一致（详见第二节核对表）。

## 二、改动清单

| 文件 | 改动 |
|---|---|
| `MAINTENANCE.md` | 「流程闸门」小节末尾新增 1 行：收尾用 `./scripts/trellis-wrapup.sh <任务名>` 一条命令完成四步、动机、指向指南第 9 节 |
| `.trellis/spec/guides/trellis-gate-guide.md` | 第 6 节加 1 条自证项（`--dry-run`）；第 9 节守卫行补「名字匹配到多个目录」 |

第 9 节核对表（左＝文档原描述，右＝脚本实际行为，行号为脚本基线）：

| 核对项 | 文档 | 脚本 | 结论 |
|---|---|---|---|
| 用法 | `<任务名或slug> [--dry-run]` | 同（另接受 `-h/--help`；未知参数退出码 2） | 一致 |
| 阶段 | finish → archive(带 `--skip-branch-validation`) → push → 校验 | `1/4`…`4/4` 同名四步 | 一致 |
| 守卫 | 7 项（原缺"名字歧义"） | 8 项：git 仓库 / origin / detached HEAD / 跟踪文件脏 / 任务不存在或已归档 / **名字匹配多个目录** / status 非 in_progress / 远端预检 | **不一致 → 已补文档** |
| `--dry-run` 语义 | 零改动、零网络动作；预检只打印不执行 | 打印 4 条命令后 `exit 0`，预检打印"跳过（--dry-run 不做网络动作）" | 一致 |
| 归档提交 | 移入 `tasks/archive/YYYY-MM/`，自动提交 `chore(task): archive …` | 同上（HEAD 变化时报告 hash 与主题） | 一致 |
| 校验 | 两个字段都是 0 + 无未提交跟踪文件 | `rev-list --left-right --count` 拆两字段判 0；`git status --porcelain` 去 `??` 后须空 | 一致 |
| 失败恢复 | 第 3 步失败打印本地提交 hash 与重推命令 | 打印归档提交变量（ARCHIVE_COMMIT）与 `git push origin <分支>`，并提示会话内可能被拦 | 一致 |
| 退出码 | 守卫拒绝＝1 | 守卫/步骤失败＝1，用法错误＝2（文档未提 2，属补充信息，未改） | 一致（不冲突） |

## 三、验证

| 检查 | 命令 | 结果 |
|---|---|---|
| 改动范围 | `git diff --stat` | 仅 `MAINTENANCE.md`(+2)、`.trellis/spec/guides/trellis-gate-guide.md`(+2/-1) |
| 成对美元符号 | `git diff -U0 \| grep '^+' \| grep -c` | 0 |
| 单星号斜体 | 同上，正则扫描孤立星号并排除代码块 | 0 处 |
| 围栏配平 | `grep -c '^```'` | `MAINTENANCE.md` 40、指南 8（均偶） |
| 表格列数 | awk 统计各行分隔符数 | 指南两张表分别为 4 列 / 3 列，行内一致；被改的守卫行仍 3 列 |
| 脚本行为复跑 | `bash /tmp/trellis-wrapup-e2e/run-tests.sh` | `PASS=46 FAIL=0`，退出码 0；其中 T8「名字歧义」PASS（对应本次补进文档的守卫） |
| 脚本静态检查 | `bash -n` / `shellcheck -S warning`（未改脚本，仅确认基线） | 均退出码 0 |

## 四、风险与未完成

- 「会话层闸门是否在单次调用内放行」仍未由真实收尾验证（上一条任务遗留），与本轮文档改动无关。
- 第 9 节仍以"会话层闸门在工具调用边界检查"为前提叙述；若真实收尾证明判定点不同，该节需要一并修正（届时属新任务）。
- `MAINTENANCE.md` 全文不使用 Markdown 链接，因此新行用行内代码路径而非超链接；若后续希望可点击，需要整体统一风格（本轮未做）。

## 五、建议下一步

1. 真实收尾时按自证清单先跑 `--dry-run`，把输出与结果回填证据/journal（同时验证会话层闸门）。
2. 若确认闸门判定点与第 9 节描述不同，另开任务修正第 9 节的机制说明。
3. 可选：在 `MAINTENANCE.md`「流程闸门」小节把脚本用法也放进 `./scripts/check-trellis-gate.sh` 附近的自检命令块（本轮未做，避免重复内容）。
