# 服务端更新检查 + 立即更新（后端）：设计与证据留痕

## 1. 接口设计

选择**独立端点**而不是扩展 `/api/status`：

- `/api/status` 每次面板启动都会拉取，其 handler 还持有 `common.OptionMapRWMutex` 读锁
  （`controller/misc.go:46-47`）；把可能触发外呼的更新检查放进去会让页面加载被网络 I/O 拖住。
- 独立端点可单独鉴权、单独限流、单独降级，且不改变既有 `/api/status` 契约。

| 方法 | 路径 | 鉴权 | 说明 |
|---|---|---|---|
| GET | `/api/status/update` | `middleware.AdminAuth()` | 三态更新状态（默认关闭时返回 disabled，不外呼） |
| POST | `/api/status/update/apply` | `middleware.AdminAuth()` | 下载 → 校验 → 原子替换 → 原地重执行 |

鉴权理由：`POST .../apply` 会下载并替换本机二进制，仅管理员可触发；查询端点与它成对，
且消费者只有管理员面板（`web/src/features/system-settings/maintenance/update-checker-section.tsx`），
与相邻的 `/api/status/test`（同样 AdminAuth）保持一致。

响应字段（`GET /api/status/update`，`success/data` 包装沿用仓库约定）：

| 字段 | 语义 |
|---|---|
| `check_enabled` | **更新检查**开关（= `update_check_setting.enabled`）；关闭时 `state=disabled`、结果字段中性且不外呼 |
| `apply_enabled` | **立即更新**开关（= `update_check_setting.apply_enabled`）；与检查开关相互独立：`check_enabled=false` 时管理员仍可触发更新 |
| `enabled` | `check_enabled` 的兼容别名（旧调用方使用），新代码请用 `check_enabled` |
| `state` | `disabled` / `unknown` / `up_to_date` / `update_available`（**失败/超时/不可比一律 unknown**） |
| `current_version` | 当前构建版本（`common.Version`） |
| `latest_version` | 最新正式版本；未知时为空串 |
| `has_update` | **三态**：true / false / null（null=未知，绝不谎报） |
| `release_url` / `published_at` / `checked_at` | 面板展示与排障用；失败时保留上次成功值。`release_url` 会经 `sanitizePublicURL()` 剥掉 userinfo（非 http(s)/解析失败一律置空），**响应不回显任何凭据/令牌** |

`POST .../apply` 成功返回 `{applied, version, previous_version, restart_mode}`；
失败返回 `{success:false, code, message}`，状态码：403 关闭、409 非更新/并发、400 平台不支持、
502 源不可用/下载失败/校验不匹配、500 临时文件或替换失败。

## 2. 配置项

环境变量（启动默认值）与运行期开关（注册进 `config.GlobalConfig`，选项接口改完**即时生效**、无需重启）：

| 项 | 默认 | 说明 |
|---|---|---|
| `UPDATE_CHECK_ENABLED` → `update_check_setting.enabled` | **`true`** | 更新检查总开关；默认开启使面板开箱可用（含「立即更新」按钮），`=false` 彻底关闭且零外呼 |
| `UPDATE_APPLY_ENABLED` → `update_check_setting.apply_enabled` | `true` | "立即更新"总开关（仅管理员可触发；置 false 整体关闭） |
| `UPDATE_CHECK_REPOSITORY` | `zhemed/new-api-own` | 更新源仓库 |
| `UPDATE_CHECK_API_BASE_URL` | `https://api.github.com` | 可指向镜像/自建代理（大陆网络出口） |
| `UPDATE_CHECK_PROXY_URL` | 空 | 可选 HTTP(S) 代理；为空时沿用 Go 默认（`HTTPS_PROXY` 等） |

常量（`setting/update_check.go`）：检查超时 10s、缓存 TTL 1h、失败退避 30m、下载超时 10min、
二进制上限 256MB、校验和文件上限 64KB。

## 3. 缓存 / 退避 / 降级

- 成功 → 结果缓存 1h（`nextAttempt = now + TTL`），TTL 内不再外呼。
- 失败/超时 → 结果**不写入**成功缓存，状态一律 `unknown`、`has_update` 保持 `null`
  （绝不把"查不到"当"已是最新"，也不对外报错），退避 30m 内不再外呼；
  若此前有成功结果则继续展示上次已知版本（`state` 仍按上次比较结果给出）。
- 外呼只发生在 `GET /api/status/update` 这次管理员请求内或启动预热的后台 goroutine：
  不阻塞启动、不持有全局锁、不影响任何其它请求路径。
- 版本无法比较（tag 非版本号）→ `unknown`，不谎报有更新。
- 日志只记脱敏后的失败原因：`common.MaskSensitiveInfo`（`service/update_check.go:159`），
  且请求不带任何凭据（不设置 `Authorization`）。

## 4. 与前端/komari 的语义对齐

- 后端 `common.CompareVersions`（`common/version_compare.go`）与前端
  `web/src/features/system-settings/utils/version-compare.ts` 逐条对齐：去一个 `v`/`V`、去 `+` 构建元数据、
  缺段补零、逐段数值比较（去前导零 + 比长度，避免溢出）、同核预发布低于正式版、缺输入为 unknown；
  单测用例直接对照前端的 `__tests__/version-compare.test.ts`。
- 未抄 komari 的两个坑：**不在首个 `-`/`+` 处截断**（保留预发布后缀参与比较）、
  **版本不可比时不判"有新版"**（返回 unknown）。
- 预发布过滤以 Release 的 `prerelease`（以及 `draft`）标志为准，不靠版本号推导：
  请求 `/releases/latest` 并再次校验标志（`service/update_check.go:214-219`）。

## 5. "立即更新"流程与安全边界

1. 解析最新正式 Release（同数据源，带 base/proxy 配置）；
2. 只在**当前版本更旧**时继续（`not_newer` 拒绝降级/重装）；
3. 取 `SHA256SUMS` 资产 → 解析出目标资产（`new-api-linux-amd64` / `new-api-linux-arm64`）的期望哈希；
   取不到/格式不合法 → 终止，绝不替换；
4. 下载二进制到**可执行文件同目录**的临时文件（`os.CreateTemp`，`0600`→`0755`），边写边算 SHA256，
   带体积上限与超时；
5. 校验通过才 `os.Rename` 覆盖（同目录原子替换；进程在运行时旧 inode 继续服务，避开 ETXTBSY）；
   任何一步失败都会删除临时文件，磁盘上仍是旧二进制（可自愈）；
6. 响应写回后 `syscall.Exec` 原地重执行；exec 失败则 `os.Exit(0)` 交给容器 restart 策略。
7. 并发保护：同一进程内同时只允许一个 apply（`busy`）。下载不携带任何凭据。

**容器部署警示（已写入代码注释）**：替换的是运行中容器内的二进制，下一次
`docker rm` / `docker run`（或重建容器）会回到镜像内的版本；长期升级仍应以发布新镜像 + 重建容器为准。
代码注释位置：`service/update_apply.go`（`UpdateApplier` 文档注释）与 `controller/update_check.go`（`ApplyUpdate`）。

## 5.5 启动行为与日志

`router.SetRouter` 启动时调用 `service.StartUpdateCheck()`（放在 SetRouter 而不是 SetApiRouter，
后者被路由单测直接调用，启动期外呼不应在测试里发生）：
- 打一行 info：`update check: enabled=… (default true), apply_enabled=… (default true), source=…, egress=direct|proxy, cache_ttl=…, failure_backoff=…`
  —— 目标源由 `describeUpdateSource()` 生成，会剥掉 URL 里的 userinfo，**日志不含任何凭据/令牌**；
  代理只打印"是否启用"，不打印代理地址本身。
- 检查开启时后台预热一次（`gopool.Go` + `recover`，绝不阻塞启动、panic 不拖垮进程），
  这样面板首次打开通常已是缓存命中，不会卡在外呼上。

## 6. 测试（全部离线：假数据源 / 假 HTTP 客户端 / 临时目录，无真实网络、无 dialector）

- `common/version_compare_test.go`：归一化 + 比较边界（v 前缀、补零、数值、预发布、非版本号、超长数字段）。
- `service/update_check_test.go`：默认关闭不外呼、开关即时生效、缓存命中、TTL 过期重查、
  失败转 unknown + 退避、退避后恢复、超时降级、失败保留上次结果、非版本 tag 中性；
  数据源请求 URL/UA/Accept、base 覆盖、prerelease/draft 过滤、非 200 与传输错误。
- `service/update_apply_test.go`：正常替换（内容/权限/无残留临时文件/生效由 Activate 触发）、
  校验和不匹配不替换、缺少校验条目不替换、缺资产不替换、下载失败不替换、体积超限不替换、
  平台不支持不外呼、开关关闭不外呼、非更新目标拒绝、并发拒绝、exec 失败回退 exit、SHA256SUMS 解析边界。
- `controller/update_check_endpoint_test.go`：关闭时三态中性且不外呼；apply 关闭时 403 `disabled`。
- `service` 另有 `TestDescribeUpdateSourceStripsCredentials`（日志描述无 userinfo/令牌）、
  `TestGetUpdateStatusReportsBothSwitches`（三态组合：检查关 / 检查开+应用关 / 两者都开）、
  `TestSanitizePublicURL` + `TestUpdateCheckerStripsCredentialsFromReleaseURL`（响应链接不带凭据）。

## 7. 实测（四条命令 + gofmt）

见下节结果。

## 8. 与其它线对接 / 待办

- 数据源结论（`hunt-deploy-chain`）：版本源已定为"补 Release 条目"，面板读 `/releases/latest`；
  若后续换成自建版本清单，只需实现 `service.ReleaseFetcher` 并替换 `defaultUpdateSource()`。
- Agent 自更新（二进制/容器替换那套）**不在范围**：本接口只做"当前实例立即更新"，不做容器重建。
- 面板前端（另一条线）消费：`GET /api/status/update`（管理员）、`POST /api/status/update/apply`（管理员）。
- 未做：回滚备份（`.old`）、下载断点续传、`force` 重装同版本——如需再议。
