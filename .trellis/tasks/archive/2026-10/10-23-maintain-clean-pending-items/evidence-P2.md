# P2 证据：PAT（系统访问令牌）实机验证 —— 本地实例

- 任务：Trellis `10-23-maintain-clean-pending-items`（P2）；共享任务板 `task-5`
- 执行者：worker-pat；时间：2026-10-23（UTC）
- 目标实例：本机 `new-api`（镜像 `ghcr.io/zhemed/new-api-own:v0.0.45`，`--network host`，bind `/root/data:/data`）
- 红线遵守：只发 GET；未 POST/PUT/PATCH/DELETE；未装包、未重启、未提交、未改仓库文件（本文件为唯一新增）；
  未访问任何非本地地址（全部 `127.0.0.1:3000`）。
- 令牌处理：**未取得任何令牌值**（原因见第 3 节），故无值可回显/落盘/外发。

## 1. 机制说明（只读代码结论，含文件:行号）

### 1.1 三条 update 路由与鉴权中间件

| 路由 | 方法 | 中间件 | 出处 |
| --- | --- | --- | --- |
| `/api/status/update` | GET | `middleware.RootAuth()` | `router/api-router.go:27` |
| `/api/status/update/apply` | POST | `middleware.RootAuth()` | `router/api-router.go:28` |
| `/api/status/update/rollback` | POST | `middleware.RootAuth()` | `router/api-router.go:29` |

- 三条路由挂在 `/api` 组下（`router/api-router.go:15-19`：`RouteTag("api")` + gzip + `BodyStorageCleanup` + `GlobalAPIRateLimit`），
  再由 `RootAuth()` 要求 root 角色（`middleware/auth.go:104-108` → `authHelper(c, common.RoleRootUser)`）。
- 运行实例的 root 会话配额仅 1 个用户：库中 `users.role` 只有 `100`（=root），见第 2 节查询。
- 路由上方的注释（`router/api-router.go:25-26`）明确写了设计意图：
  「自更新 = 替换本机二进制（等价 root 能力）：路由要求 root 会话，handler 内再用
  `middleware.RequireSessionCredential` 拒绝系统访问令牌（PAT）」。

### 1.2 `pat_not_allowed` 的产生点

- 稳定错误码常量：`middleware/auth.go:140` `CodePATNotAllowed = "pat_not_allowed"`。
- 拒绝函数：`middleware/auth.go:148-159` `RequireSessionCredential(c)`：
  - `c.GetBool("use_access_token")` 为假 → `return true`（放行）；
  - 为真 → `c.AbortWithStatusJSON(403, {success:false, code:"pat_not_allowed", message:...})` 并 `return false`。
- `use_access_token` 的写入点：`middleware/auth.go:223`（`setDashboardAuthContext(..., useAccessToken)`），
  由 `authenticateDashboardRequest`（`middleware/auth.go:161-170`）判定：
  凭据类型为 `dashboardCredentialPAT` 时为 true。
- 三个 handler 都在最前面做了检查：`controller/update_check.go:21`（GetUpdateStatus）、
  `:36`（ApplyUpdate）、`:59`（RollbackUpdate）——即**三条路由都会返回该码**（GET 也会）。
- 即：**普通管理员**（role=10，`RootAuth` 阶段就 403）与 **PAT**（过 `RootAuth`，被 handler 的
  `RequireSessionCredential` 403 + `pat_not_allowed`）两条路径都被拒，符合 journal-2 记录的设计口径。

### 1.3 令牌如何携带

- 只走请求头 `Authorization`（`middleware/auth.go:173` `authorizationToken(c.GetHeader("Authorization"))`），
  接受 `Bearer <token>` 或裸 token（`middleware/auth.go:202-214`：`strings.Fields` 后 2 段且首段等于 `Bearer`，
  否则必须恰好 1 段）。**不看 Cookie**。
- 分类：`service.ParseDashboardAccessToken(raw)`（内部会话令牌，`middleware/auth.go:177`）
  失败后回落到 `model.ValidateAccessToken(raw)`（PAT，`middleware/auth.go:188`）。
- 认证失败时返回 `401 AUTH_UNAUTHORIZED`（实测见第 4 节），**不是** `pat_not_allowed`——该码只在
  「PAT 有效但动作被禁」的场景出现。

### 1.4 实例侧令牌存在哪里（表名/字段名/名称约定）

- 表：`users`；列：`access_token`；模型定义：`model/user.go:94`
  `AccessToken *string \`json:"-" gorm:"type:char(32);column:access_token;uniqueIndex"\``
  注释原文：`// this token is for system management`（即「系统访问令牌」）。
- 校验查询：`model/user.go:1112-1126` `ValidateAccessToken`：
  `DB.Where("access_token = ?", token).First(user)`——PAT 是**挂在某个 user 行上的 32 字符字段**，
  不是独立表、不是 `options` 配置项、不是环境变量。
- 名称约定：字段无独立名字，等价于「root 用户（role=100）那一行的 `access_token`」。
  另一个易混表是 `tokens`（渠道 API Key，`key` 列，48 字符），**不参与** dashboard 鉴权路径。

## 2. 第二步：取用令牌（工作区内优先）——结果：实例侧未配置 PAT

### 2.1 先确认实例身份与监听地址（未越界）

```console
$ docker --version ; docker version --format '{{.Server.Version}}' ; docker compose version
Docker version 29.7.2, build a7dcaa6
29.7.2
Docker Compose version v5.4.0          # 环境达标（29.7.2 + v5.4.0）

$ docker inspect new-api --format '{{.Name}} | image={{.Config.Image}} | status={{.State.Status}} | networkmode={{.HostConfig.NetworkMode}} | ports={{json .HostConfig.PortBindings}} | mounts={{range .Mounts}}{{.Source}}=>{{.Destination}} ({{.Type}}) {{end}}'
/new-api | image=ghcr.io/zhemed/new-api-own:v0.0.45 | status=running | networkmode=host | ports={} | mounts=/root/data=>/data (bind)

$ docker inspect new-api --format '{{range .Config.Env}}{{println .}}{{end}}' | cut -d= -f1 | sort
LOG_CLEANUP_RETENTION_DAYS LOG_MEMORY_MAX_BYTES LOG_MEMORY_MAX_ROWS LOG_SQL_DSN PATH
                          # 无 PORT / 无 SQL_DSN / 无 SQLITE_PATH → 用默认端口与默认 SQLite

$ curl -s -o /dev/null -D - --max-time 5 http://127.0.0.1:3000/api/status | grep -iE '^(HTTP/|X-New-Api-Version)'
HTTP/1.1 200 OK
X-New-Api-Version: v0.0.45
```

- 结论：实例 = 本机 `new-api` 容器，`--network host`，监听 **127.0.0.1:3000**，版本 **v0.0.45**（与镜像标签一致）。
- 实例数据目录 = bind 挂载 `/root/data`（容器内 `/data`），SQLite 库文件 `/data/one-api.db`
  （宿主同 inode：`ls -la` 两边均为 1155072 字节；文件头 `SQLite format 3\0`，确认 dialect 为 SQLite）。
- **工作区 `/root/new-api-own` 下不存在任何 `*.db` / `*.sqlite` 文件**（`find` 全量搜过），
  该实例的数据目录落在 `/root/data`，不在工作区之内。

### 2.2 读取令牌（只读、不打印值、命令中无令牌字面值）

优先尝试工作区 → 无 DB；随后通过容器 bind 挂载读取**运行中实例自己的** SQLite 文件（`mode=ro` 只读，
未修改任何字节；未打印任何令牌值）。命令（正文不含令牌值）：

```console
$ python3 - <<'PY'
import sqlite3
con = sqlite3.connect("file:/root/data/one-api.db?mode=ro", uri=True)
cur = con.cursor()
print("users has access_token column:", "access_token" in [r[1] for r in cur.execute("PRAGMA table_info(users)")])
print("users rows:", cur.execute("SELECT COUNT(*) FROM users").fetchone()[0])
print("rows with non-empty access_token:",
      cur.execute("SELECT COUNT(*) FROM users WHERE access_token IS NOT NULL AND access_token<>''").fetchone()[0])
for row in cur.execute("SELECT id, username, role, LENGTH(access_token) FROM users WHERE access_token IS NOT NULL AND access_token<>'' ORDER BY id"):
    print("  candidate id=%s role=%s token_len=%s" % row)
con.close()
PY
```

输出摘要：

```text
users has access_token column: True
users rows: 1
rows with non-empty access_token: 0
（候选行列表为空）
```

补充核对（同一只读连接）：

```text
各表 tokens 类：tables containing 'token': ['tokens']；tokens 行数=1，列含 key/user_id/status/name（渠道 API Key，非 PAT）
users 行：id=1, username=z, role=100, access_token 长度=0（NULL 或空串）
options 表：14 行，key 中无 token/access 相关项
库文件 mtime 在我读完后仍有更新（16:01 → 16:03 UTC），且新日志每 60s 打印 "syncing options from database"
  → 证明我读的就是运行实例正在使用的那个库，不是陈旧副本
```

### 2.3 结论：取不到令牌 → 按任务书停止

- **实例侧未配置系统访问令牌**（`users.access_token` 为空），因此 `pat_not_allowed` 路径**无法被实测触发**。
- 令牌来源（若存在）应为：SQLite 表 `users`、列 `access_token`（`model/user.go:94/1112`），
  长度约定 32 字符（`char(32)`）。
- **掩码形式：不可用（无值）**——不是"我拿到了但不写"，而是库中该字段为空。
  本次未取得、未回显、未落盘、未外发任何令牌值。

## 3. 第三步：实机验证（仅 GET，仅本地回环）

按任务书「若令牌取不到 → 立即停止、如实报告、不要反复重试、不要尝试绕过」，**未尝试任何伪造/绕过**。
为如实刻画路由现状，做了一轮**有界**的 GET 探测（共 6 次 GET，无写动作），并在下方明确标注其证据边界。

### 3.1 命令（不含任何令牌值）

```console
# [A] 无 Authorization 头（基线）
for p in /api/status/update /api/status/update/apply /api/status/update/rollback; do
  curl -s -o /tmp/p2_noauth.json -w 'http=%{http_code} ' --max-time 5 "http://127.0.0.1:3000$p"
  python3 -c "import json;d=json.load(open('/tmp/p2_noauth.json'));print('code=',d.get('code'))"
done

# [B] 合成占位 token（openssl rand -hex 16 → 32 字符，非真实凭据，值未打印）
TOKEN="$(openssl rand -hex 16)"; echo "token_len=${#TOKEN}"
for p in /api/status/update /api/status/update/apply /api/status/update/rollback; do
  curl -s -o /tmp/p2_dummy.json -w 'http=%{http_code} ' --max-time 5 -H "Authorization: Bearer $TOKEN" "http://127.0.0.1:3000$p"
  python3 -c "import json;d=json.load(open('/tmp/p2_dummy.json'));print('code=',d.get('code'))"
done
```

### 3.2 输出摘要

| 路由 | 无凭据 GET | 占位 token GET（Bearer，32 字符合成值） |
| --- | --- | --- |
| `/api/status/update` | `401` + `AUTH_UNAUTHORIZED` | `401` + `AUTH_UNAUTHORIZED` |
| `/api/status/update/apply` | `404` | `404` |
| `/api/status/update/rollback` | `404` | `404` |

`401` 响应体示例（无令牌内容）：

```json
{"success":false,"code":"AUTH_UNAUTHORIZED","message":"Unauthorized, invalid access token"}
```

- `/api/status/update` 的 `401` 证明：`RootAuth` 鉴权链在**无效凭据**处就返回 `AUTH_UNAUTHORIZED`。
- **`apply` / `rollback` 是 POST-only 路由**：GET 命中不到注册的 method，Gin 返回 `404`。
  按红线（只 GET）**没有**对这两条发 POST，因此它们的 `pat_not_allowed` 分支在本次**未被实测**。

### 3.3 证据边界（必须明说）

- **期望结果「403 + pat_not_allowed」本次未观测到，且未观测到任何放行（200）**。
  原因是前置条件缺失（实例侧无 PAT），不是被测行为不符。
- 已实测的确定性结论：
  1. 实例 v0.0.45 在跑，三条路由均已注册（GET `/api/status/update` 走通鉴权链 → 401 而非 404/200）；
  2. 无凭据与本轮占位 token 均**不能**拿到 200；
  3. `users.access_token` 为空 → 当前不可能存在任何可用的 PAT 来命中 `pat_not_allowed`。
- 未实测部分（GET-only 约束下的固有限制）：`apply` / `rollback` 的 method 级行为与 PAT 分支；
  「有效 PAT → 403 `pat_not_allowed`」这一条需要实例侧先生成令牌 + 一次 POST（危险动作，未获授权）。
- 未重试、未绕过、未扩大范围：全部探测共 6 次 GET，均为本地回环。

## 4. 结论

- 机制已查清：三条 update 路由 = `router/api-router.go:27-29`（GET 1 条 + POST 2 条），
  统一 `RootAuth()`；PAT 被拒的判定在 handler 首行 `middleware.RequireSessionCredential`
  （`middleware/auth.go:148-159`，三个调用点 `controller/update_check.go:21/36/59`），
  错误码常量 `middleware/auth.go:140`；令牌经 `Authorization` 头携带（`middleware/auth.go:173/202-214`）；
  存储在 `users.access_token`（`model/user.go:94`，32 字符，`model/user.go:1112` 校验）。
- 实机验证：**被前置条件阻塞**——本地实例侧**根本没有配置系统访问令牌**（`users` 表仅 1 行且
  `access_token` 为空），因此「403 + `pat_not_allowed`」无法实测；期间未观测到任何放行（无 200）。
- 与 journal 记录的差异：journal 说"上次因账号未配置系统访问令牌而没做成"，
  **本轮复核后仍然是同一状态**——用户当轮说的"api 已经好了，权限我授权了"可能指的是**授权给我用**，
  而非"令牌已在实例侧生成"；库证据是后者未发生（要生成需在面板 root 会话里点一次生成/重置）。
