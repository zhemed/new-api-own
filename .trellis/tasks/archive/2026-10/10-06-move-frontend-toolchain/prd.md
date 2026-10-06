# MAINTENANCE：前端工具链移出「已知不一致」

- 任务：`10-06-move-frontend-toolchain`
- 执行：`review-ops`（`MAINTENANCE.md` 是写范围）
- 共享任务：`task-18`（Lead 指派）

## 目标

1. 把 `### 前端工具链（2026-10-06 起本机可用）` 从 `## 已知不一致` 移到 `## 构建与测试` 之后。
2. 自查「已知不一致」该节：移出后应**只剩"决定不改"的条目**。
3. 顺手用实测刷新工具链块的基线数字（避免文档写着过期数字误导人）。

## 执行记录（2026-10-06）

### 1. 移动

`### 前端工具链` 现位于 **131 行**，紧跟 `## 构建与测试`（107 行）的 `> 注意：… embed web/dist` 之后，
作为构建与测试的小节；`## 本地部署验证` 顺延到 142 行。

### 2. 「已知不一致」自查结果：原表混入 2 条"已处理"行 → 已拆分

原表 9 行里有 2 行不是"决定不改"（与节标题「刻意未改」矛盾）：

- `3 个 api-key group 表格测试失败`（已修复）
- `footer XSS 测试曾经永远失败`（已修复）

→ 拆出为独立小节 `## 团队审查已处理项（2026-10-06 留痕）`（246 行），并把这两行的"处理结论"
写清；「已知不一致」表**现剩 7 行，全部是"决定不改"**（multipart 校验 / DTO 指针 / DELETE LIMIT /
count_token_failed 500 / get_channel_failed 500 / ClickHouse 行数上限 / 前端孤儿模块 knip ignore）。

### 3. 工具链基线数字刷新（**全部本机实测**，仅本地命令，无网络）

| 命令 | 实测 |
|---|---|
| `bun --version` | `1.4.2` |
| `bun run typecheck` | 退出码 0（0 错误） |
| `bun run lint` | `Found 21 warnings and 0 errors.`（退出码 0，939 文件 / 186 规则） |
| `bun test` | **167 pass / 0 fail**（34 个文件） |
| `bun run knip --include files` | 退出码 0（**unused files = 0**） |
| `bun run knip`（全量） | 退出码 1：311 unused exports / 99 unused exported types / 6 unused deps + 2 devDeps / 1 duplicate export / 2 configuration hints |

原文的 `151 pass / 3 fail`（及"3 个为上述既有失败"的跨节引用）已过期 —— 改成实测值，
并去掉那句跨节引用（移动后"上述"会指向别处）。

## 校验

- 目录结构：`### 前端工具链` 不再出现在「已知不一致」下（见下方结构 grep 输出）。
- 「已知不一致」表：表头 1 + 分隔行 1 + 数据行 7，全部为"决定不改"；「已处理项」表：表头 + 2 行。
- 代码围栏 38 个配平；`grep -rn 已知不一致` 在仓库内只有本节与其自身的相对引用。
- 未 commit/push；只跑了本地 `bun` 命令（无 `bun install`、无网络）。
