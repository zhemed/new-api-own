# 3020 真机验证：初始化 + 部署新前端 + 截图取证

## Goal

在 Lead 授权的范围内，把本轮前端修复**真正部署到本机 3020 演示实例**并做真机取证：
完成初始化向导 → 重建并重启实例以带上新前端 → 截"认证后顶栏显示版本"的图，
并与 `/api/status` 的版本对照。

## 现状（已侦察）

- 进程：pid `476111`，`./new-api -port 3020 -log-dir=`，cwd/exe 均在 `/root/new-api-own/.local-instance/`。
- 前端**内嵌**：`main.go:42 //go:embed web/dist` ⇒ 要让新前端生效**必须重建 Go 二进制**。
- 旧进程环境含 `LOG_SQL_DSN` / `LOG_CLEANUP_INTERVAL` / `LOG_MEMORY_MAX_ROWS`，重启时**必须原样带过去**。
- 工具链：`go1.26.6`；前端 `bun build` 产物在 `web/dist`。

## Requirements

1. **初始化**：在 3020 走完初始化向导，创建本地管理员账号。
   - 凭据**不得**写入任何文件、不得出现在汇报里；生成后仅存活于浏览器会话内（sessionStorage）。
2. **重启前记录**：pid + 启动方式；从 `/proc/<pid>/environ` 读环境变量进当前 shell（**不打印值**）。
3. **部署新前端**：`web/**` 改动需先 `bun run build`，再重建 Go 二进制（产物先落临时路径，
   再 `mv` 覆盖 `.local-instance/new-api`，避免对运行中二进制的 ETXTBSY），最后按记录方式重启。
4. **取证**：
   - `bw` 截图：认证后顶栏显示版本（图落工作区内）；
   - 同时给出 `/api/status` 的 `version` 与构建标识（应为 `rv.0.0.6.*`，不再是 `rv.0000.*`）；
   - 两者必须一致。
5. 按 Lead 要求：把"HTTP 下复制按钮失效"与"setup 守卫 localStorage 永久绕过"的复现步骤各写 ≤3 行。

## Constraints

- 只改 `web/**`（Go 源码不动；重建二进制属授权范围）；除 3020 外不得重启其它服务；
- 不动用户远程实例；禁止 `git commit/push/tag`、`git add -A`、worktree；
- 禁止网络动作（本机 3020 / `bw` 除外）；凭据零落盘、零上报；
- 收工前 `bun run typecheck | lint | test | build` 必须全绿。

## Acceptance Criteria

- [ ] 3020 完成初始化并可登录。
- [ ] 新前端已部署（构建标识不再是 `rv.0000.*`）。
- [ ] 顶栏版本截图 + `/api/status` 版本，二者一致，证据路径已给出。
- [ ] 四条命令实测输出（含退出码）。
- [ ] 两条缺陷的复现步骤各 ≤3 行。
