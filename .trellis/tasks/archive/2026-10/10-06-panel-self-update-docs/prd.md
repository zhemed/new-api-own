# 面板内更新（self-update）文档

- 任务：`10-06-panel-self-update-docs`
- 执行：`review-ops`（写范围：`README.md`、`README.en.md`、`MAINTENANCE.md`）
- 共享任务：`task-20`（Lead 指派）

## 背景

用户要求「面板能直接更新」。Lead 给的范围变更：在三个文档补「面板内更新（self-update）」一节，
内容须包含机制、代价、开关与网络、排障、与「版本与升级」互引，且不得写成第二种部署方式。

## 写前核实（只读，2026-10-06）

| 事实 | 证据 | 写进文档的措辞 |
|---|---|---|
| 服务端更新检查已实现：外呼 GitHub Releases，浏览器不直连，带超时/缓存/失败退避 | `service/update_check.go`、`setting/update_check.go` | 按已实现写 |
| 开关 `UPDATE_CHECK_ENABLED` **默认关闭**（未显式开启时服务端不外呼） | `setting/update_check.go`：`GetEnvOrDefaultBool("UPDATE_CHECK_ENABLED", false)` | 按代码写"默认关闭"，**不写"默认允许"** |
| 更新源仓库可换：`UPDATE_CHECK_REPOSITORY`（默认本项目） | 同上 | 写"可换仓库" |
| 代理：HTTP 客户端用默认 transport → 遵循标准 `HTTPS_PROXY`/`HTTP_PROXY`/`NO_PROXY` | `service/update_check.go`（`http.Client{Timeout: …}`） | 写标准代理环境变量；**自定义镜像 base 目前不存在，不写** |
| 二进制交付（`new-api-linux-amd64` / `arm64` + `SHA256SUMS`）是**计划**：`release.yml` 当前为 image-only，未附资产 | `.github/workflows/release.yml`（name: Publish release (image-only)）、`10-06-release-linux-binaries/prd.md` | 写"资产固定名"作为契约，不断言"已发布" |
| **自更新的下载/校验/替换/重执行代码尚未落地**：全仓库无 `SHA256SUMS` 解析、无下载替换、无 update 端点、面板无「立即更新」按钮 | `grep`（Go/前端/路由均为空） | **不写成"现成可用"**：以"前置条件 + 契约"措辞（面板出现该按钮才说明该构建内置更新器；否则按镜像三步） |

## 要求（Lead 给定，全部覆盖）

1. **它是什么**：检查新版本 → 管理员点「立即更新」→ 后端下载**本机架构** Linux 二进制 →
   校验 `SHA256SUMS` → 原子替换 → 原地重执行；校验失败**绝不替换**。
2. **代价（显眼）**：容器内替换的二进制在**下一次 `docker rm/run` 会退回镜像内版本**；
   适合"临时跟上版本"，**正式升级仍走镜像三步**。
3. **开关与网络**：可整体关闭（`UPDATE_CHECK_ENABLED`）；GitHub 不通时用标准代理
   （`HTTPS_PROXY`）/ 换更新源仓库（`UPDATE_CHECK_REPOSITORY`）。
4. **排障**：更新后 `/api/status` 的 version 是否变化、如何确认二进制摘要、失败后如何回退。
5. 与「版本与升级（每次维护必做）」**互相引用**，不重复内容，不写成第二套部署方式。

## 硬约束

- 只改这三个文件；不 commit/push；无网络动作；不写主机名/IP/凭据。

## 验收

- 改动清单 + `grep -n '^## \|^### '` 目录结构；
- 文中不出现"已有但现在没有"的能力（自更新按钮/资产以"前置条件/契约"表述）。

## 待 Lead 决策（写完后一并汇报）

1. `UPDATE_CHECK_ENABLED` 代码默认是 **false**，与 Lead 说的"默认允许"不一致 → 文档按代码写，
   如需"默认允许"要改代码（不在我范围）。
2. 自更新执行路径尚未落地 → 文档以契约+前置条件写；若实现马上合并，我把措辞改成"现成可用"。

---

# 执行记录（2026-10-06）

## 写作过程中实现落地了 → 已按**实际代码**对齐

写文档期间 `service/update_apply.go` + `release.yml`（附二进制资产）落地，因此把措辞从"契约/计划"
改成与实现一致，并逐项核对：

| 文档内容 | 代码依据 |
|---|---|
| 开关 `UPDATE_CHECK_ENABLED` 默认 `false`；`UPDATE_APPLY_ENABLED` 默认 `true`（仅管理员可触发，可整体关闭）| `setting/update_check.go:20-23`（注册进全局配置管理器，面板改即时生效）|
| 更新源：`UPDATE_CHECK_REPOSITORY` / `UPDATE_CHECK_API_BASE_URL`（默认 `https://api.github.com`）/ `UPDATE_CHECK_PROXY_URL`（空则走 Go 默认代理 env）| `setting/update_check.go:36-45` |
| 资产名 `new-api-linux-amd64` / `new-api-linux-arm64` + `SHA256SUMS` | `service/update_check.go:37-39`、`.github/workflows/release.yml:162-172` |
| 校验通过才 `os.Rename` 替换；失败一律丢弃临时文件 | `service/update_apply.go:160-166, 244-276` |
| 生效方式：优先 `syscall.Exec` 原地重执行，exec 失败则退出进程交给容器 restart 策略 | `service/update_apply.go:174-184` |
| 护栏：不回退（`not_newer`）、仅 amd64/arm64、并发互斥（`busy`）、下载 256MiB 上限、10 分钟超时 | `service/update_apply.go:92-160` + `setting/update_check.go` 常量 |
| 面板入口文案：系统设置 → 系统维护（`System maintenance`）→ 按钮「立即更新」（i18n `Update now`）| `web/.../section-registry.tsx:132`、`zh.json` `"Update now": "立即更新"` |
| 排障里的 `docker exec new-api sha256sum /new-api` | Dockerfile: `COPY … /new-api` + `ENTRYPOINT ["/new-api"]`；**实测**该镜像内命令可跑（输出 0.0.6 二进制摘要） |

## 顺带纠正的两处（相对既有的过时写法）

- 更新检查**已不再是"有更新就说"**：三态（有更新 / 已最新 / 无法确定），且"更新源版本比本机旧"按已最新处理
  （避免历史上"跑 0.0.6 却提示 v0.0.5 有更新"的误报）；
- 代理/镜像 base **现在真的有**（`UPDATE_CHECK_API_BASE_URL`），不是只能用标准代理环境变量。

## 交付

| 文件 | 改动 |
|---|---|
| `MAINTENANCE.md` | 新增 `## 面板内更新（self-update）`（493 行起），下辖「它是什么 / 代价 / 开关与网络 / 排障 / 与版本与升级的关系」五节，并与上节双向互引 |
| `README.md` | 部署段新增 `### 面板内更新（临时跟上版本，不是第二种部署方式）`（92 行） |
| `README.en.md` | 同步 `### In-panel update (catch up temporarily, not a second deployment method)` |

校验：围栏 README 14 / README.en 14 / MAINTENANCE 40（全部配平）；五个环境变量名全部在代码中存在；
`127.0.0.1` 之外无 IP/主机名/凭据；未 commit/push；除只读 `docker run`（验证 sha256sum 与镜像路径）外无网络动作。

## 仍需 Lead 注意

- 自更新要端到端可用，需要一个**已发布且带三个资产**的 Release：现有 `v0.0.6` 的 Release 情况需发布线确认
  （缺资产时点「立即更新」会明确失败，不会替换）。
- 镜像内的 0.0.6 **不含**更新器（`service/update_apply.go` 刚落地、尚未发版）→ 文档已用"面板有按钮才可用"的前置条件表述。

