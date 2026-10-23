# P2b 证据：PAT 实机验证补做（本地实例，用户已授权凭据读写）

- 承接：Trellis 归档任务 `10-23-maintain-clean-pending-items` 的 P2（P2 首次执行见同目录 `evidence-P2.md`：当时实例侧未配置令牌 → 阻塞）
- 共享任务板：`task-7`；执行者：worker-pat；时间：2026-10-23T16:39Z 起
- 授权：用户当轮明确授权「为验证目的读取与写入本地实例的凭据字段」、「工具缺少直接补和更新」
  （task-7 描述记为 2026-10-24；本机当前 UTC 日期为 2026-10-23，此处按任务书原文标注，不擅自改写日期）
- 目标实例：**仅本地** `127.0.0.1:3000`（容器 `new-api`，`ghcr.io/zhemed/new-api-own:v0.0.45`，`--network host`，bind `/root/data:/data`）
- 请求方法：**全程只发 GET**，共 5 次（首版探针 1 + 修正版探针 2 + `/api/status` 健康检查 2）；**未发任何 POST/PUT/PATCH/DELETE**
- 未装包（无需：宿主自带 `python3` + `sqlite3` 模块即可读写/查询，未触发授权中的"补工具"分支）
- 未重启实例（`RestartCount=0`，容器 `StartedAt=2026-10-23T14:33:36Z` 与本轮全程一致）

## 1. 机制确认：鉴权链读取 `access_token` **不经过任何缓存** → 写入即时生效

问题（任务硬约束）：若 `users.access_token` 的读取走内存/Redis 缓存，则新令牌不会即时生效，必须记「需重启才生效」并停止。

### 1.1 代码证据（逐跳，无反例）

| 跳 | 代码位置 | 是否查库 | 是否可缓存 |
| --- | --- | --- | --- |
| 取令牌 | `middleware/auth.go:173` `authorizationToken(c.GetHeader("Authorization"))` | 否 | 否（纯请求头解析） |
| 解析内部会话令牌 | `middleware/auth.go:177` `service.ParseDashboardAccessToken(raw)` → 不匹配则回落 | 否 | 否 |
| **校验 PAT** | `middleware/auth.go:188` → `model.ValidateAccessToken(raw)` → `model/user.go:1118` `DB.Where("access_token = ?", token).First(user)` | **是（直查 DB）** | **否——无任何缓存封装** |
| 取用户基信息 | `middleware/auth.go:195` `model.GetUserCache(patUser.Id)` | `model/user_cache.go:96` 先 Redis，`:104` 回落 DB | 有缓存层，但见 1.2 |
| 角色判定 | `middleware/auth.go:55` `user.Role < minRole` → root 需 `RoleRootUser` | 同上 | — |

关键点：**「令牌 → 用户」这一步（`ValidateAccessToken`）是裸 DB 查询，没有缓存**；
被缓存的只有用户基信息（Role/Status 等），而 `GetUserCache` 在 Redis 不可用时直接回落 DB。

### 1.2 本实例 Redis 未启用（缓存层实际不参与）

```console
# 容器 env（只列 key 名，非敏感）
$ docker inspect new-api --format '{{range .Config.Env}}{{println .}}{{end}}' | cut -d= -f1 | sort
LOG_CLEANUP_RETENTION_DAYS LOG_MEMORY_MAX_BYTES LOG_MEMORY_MAX_ROWS LOG_SQL_DSN PATH
                          # 无 REDIS_CONN_STRING

# 实例启动日志
$ docker exec new-api sh -c 'grep -h "Redis" /data/logs/*.log | tail -3'
[SYS] 2026/10/23 - 01:48:37 | REDIS_CONN_STRING not set, Redis is not enabled
[SYS] 2026/10/23 - 13:04:33 | REDIS_CONN_STRING not set, Redis is not enabled
[SYS] 2026/10/23 - 14:33:36 | REDIS_CONN_STRING not set, Redis is not enabled
```

`common/redis.go:17,25-28`：`REDIS_CONN_STRING` 为空即 `RedisEnabled = false`；
`model/user_cache.go:123-125` `cacheGetUserBase` 在 Redis 未启用时直接返回错误 → `GetUserCache` 走 DB 回落分支（`model/user_cache.go:101-107`）。

**结论：写入 `users.access_token` 后无需重启，下一次请求即可生效**——并由 §3 的实测结果反向确证（写入后第一次请求即返回 `pat_not_allowed`，而非 401）。

## 2. 前置状态与还原目标（P2 审计基线）

```console
$ docker inspect new-api --format '{{.Name}} | image={{.Config.Image}} | status={{.State.Status}} | net={{.HostConfig.NetworkMode}}'
/new-api | image=ghcr.io/zhemed/new-api-own:v0.0.45 | status=running | net=host

$ curl -s -o /dev/null -D - --max-time 5 http://127.0.0.1:3000/api/status | grep -iE '^(HTTP/|X-New-Api-Version)'
HTTP/1.1 200 OK
X-New-Api-Version: v0.0.45
```

- 写入前基线：`users` 共 1 行，`id=1 username=z role=100(root)`，`access_token IS NULL`，非空行数 **0**。
- 还原目标 = **SQL NULL**（与 `evidence-P2.md` 记录的基线完全一致）。

## 3. 执行与结果（命令不含令牌字面值）

### 3.1 命令（探针脚本 `/tmp/p2b_pat_probe2.py`，仅本地写 DB + 仅 GET）

脚本要点（完整源码见 §6）：
1. 只读快照原值（`SELECT access_token FROM users WHERE id=1`）；
2. `uuid.uuid4().hex[:32]` 生成 32 位十六进制随机令牌，`UPDATE users SET access_token=? WHERE id=1` 并 commit；
3. **一次** `GET http://127.0.0.1:3000/api/status/update`，头 `Authorization: Bearer <该令牌>`；
4. **一次**对照探针：另一个无关的 32 位随机令牌（证明路由不是"见令牌就放行"）；
5. `finally` 中还原原值（**NULL 安全**：原值为 NULL 时写回 `SET access_token=NULL`），并复核。
令牌只存在于进程内存与 DB 字段，**未打印、未落盘到任何文件、未写进消息**；脚本输出的掩码只用前 2 位。

```console
$ python3 /tmp/p2b_pat_probe2.py
```

### 3.2 输出摘要（原样，含掩码与状态码）

```text
[step0] baseline row=(1, 'z', 100, 1, 0) users=1 nonempty=0
[step1] original token shape: <empty> (is_null=True)
[step2] wrote token: len=32 exact_match=True
[step3] GET with written PAT      -> http=403 code=pat_not_allowed
[step4] GET with unrelated token  -> http=401 code=AUTH_UNAUTHORIZED
[step3.body] {"success": false, "code": "pat_not_allowed", "message": "自更新只能由管理员登录会话触发，系统访问令牌（PAT）不被允许。 Self-update requires an admin login session; personal access tokens are not allowed."}
[step4.body] {"success": false, "code": "AUTH_UNAUTHORIZED", "message": "Unauthorized, invalid access token"}
[step5] restore: exact_equal_to_original=True shape=<empty>
[step6] final row=(1, 'z', 100, 1, 0) users=1 nonempty=0
[step6] row_identity_unchanged=True token_field_restored=True
```

### 3.3 判定表

| 场景 | 请求 | HTTP | `code` | 期望 | 符合 |
| --- | --- | --- | --- | --- | --- |
| 有效 PAT（root，role=100）访问 `GET /api/status/update` | 1 次 GET | **403** | **`pat_not_allowed`** | 403 + `pat_not_allowed` | **是** |
| 无关随机令牌（对照） | 1 次 GET | 401 | `AUTH_UNAUTHORIZED` | 401（无效凭据） | 是 |

- 403 证明：令牌**通过了** `RootAuth`（否则会是 401），随后在 handler 首行被
  `middleware.RequireSessionCredential`（`middleware/auth.go:148-159`）以 `pat_not_allowed` 拒绝
  ——与 P2 的目标行为一致。
- 对照 401 证明：这不是"路由不校验"或"任何 token 都 403"，而是**有效 PAT 被专门拒绝**。
- **未发生放行（无 200）**：update 路由的全部 3 次 GET 命中结果为 403（有效 PAT）与 401（无/无效凭据）；
  另 2 次 `GET /api/status`（健康检查，非 update 路由）为 200，与鉴权无关。

### 3.4 三条路由的覆盖说明（重要：只有 1 条被实测，其余 2 条为代码同源论证）

| 路由 | 方法 | 本轮是否实测 | 依据 |
| --- | --- | --- | --- |
| `/api/status/update` | GET | **是（403 `pat_not_allowed`）** | 实机请求 |
| `/api/status/update/apply` | POST | 否（红线禁 POST） | `controller/update_check.go:36` 同一 `RequireSessionCredential` 首行守卫，与 GET 同源 |
| `/api/status/update/rollback` | POST | 否（红线禁 POST） | `controller/update_check.go:59` 同上 |

- 用户当轮授权与 lead 指令均为「发**一次** GET `/api/status/update`」；两条 POST 若被意外放行会**真的触发升级/回滚**，
  故按红线**不探**。其 `pat_not_allowed` 行为由**同一守卫函数在三处首行调用**这一代码事实支撑，不冒充实测。
- 选择 GET 而非 POST 的证明力说明：PAT 通过 `RootAuth` 后落到 handler，handler 的第一条语句就是该守卫，
  与路由 method 无关——GET 已足以触达判定点。

## 4. 完整时间线（含一次脚本缺陷，如实记录）

| 顺序 | 动作 | GET 次数 | 结果 |
| --- | --- | --- | --- |
| 1 | 首版探针 `/tmp/p2b_pat_probe.py`：写令牌 → 1 次 GET → 还原 | +1 | GET **403 `pat_not_allowed`**（成功）；但**还原语句抛异常**（见下），令牌仍留在 DB |
| 2 | 立即用 `UPDATE users SET access_token=NULL WHERE id=1` 还原并复核 | 0 | `nonempty=0`、`token_is_null=1` ✅ |
| 3 | 修正脚本（NULL 安全绑定 + 对照探针 + 异常也还原），重跑 `/tmp/p2b_pat_probe2.py` | +2 | 403 `pat_not_allowed` + 401 对照；还原 `exact_equal_to_original=True` ✅ |
| 4 | 健康检查 `GET /api/status`（确认实例未被影响） | +1 | 200 + `X-New-Api-Version: v0.0.45` |

**缺陷与处置（不隐瞒）**：首版脚本的还原语句写作
`UPDATE users SET access_token=? WHERE id=?` 并传入 `original=None`，Python `sqlite3` 拒绝绑定 `None`
→ `ValueError: parameters are of unsupported type`，还原未执行。**当轮立即用 `IS NULL` 语义的显式
`SET access_token=NULL` 还原成功**（§3 前置基线 = NULL，故"还原为 NULL"即等价于原值），
并在修正版脚本中以 `if original is None: SET access_token=NULL` 分支修好、复跑验证。
期间令牌值始终只存在于 DB 字段与进程内存，未外泄。

## 5. 还原证据与残留检查

```text
# 还原后只读复核
users 非空 access_token 行数 = 0
users 行 = (id=1, username='z', role=100, access_token IS NULL)
脚本自证：restore exact_equal_to_original=True；row_identity_unchanged=True；token_field_restored=True

# 无审计/日志残留污染
logs 表共 1 行（id=1, type=7, content='Logged in successfully via password', 非本轮产生）
  —— 403 响应在 handler 首行即中止，未触发管理审计写入

# 实例未受影响、未重启
curl -s -o /dev/null -D - http://127.0.0.1:3000/api/status → HTTP/1.1 200 OK, X-New-Api-Version: v0.0.45
docker inspect new-api → status=running started=2026-10-23T14:33:36Z restarts=0
```

- 令牌掩码（仅供记录形态，无值）：`7e...(32 chars, masked)`（首次探针，前 2 位 + 长度）；
  第二次探测的令牌未在任何输出中出现过前 2 位（脚本只打印长度与 `exact_match`）。**两者均已在还原步骤中从库中清除。**

## 6. 探针脚本（可复核；不含任何令牌值）

```python
#!/usr/bin/env python3
"""P2b PAT live verification (local instance 127.0.0.1:3000 only; GET only)."""
import json, sqlite3, urllib.error, urllib.request, uuid

DB = "/root/data/one-api.db"
URL = "http://127.0.0.1:3000/api/status/update"
USER_ID = 1
STATE_SQL = "SELECT id, username, role, access_token IS NULL, COALESCE(LENGTH(access_token),0) FROM users WHERE id=?"

def snapshot():
    con = sqlite3.connect("file:%s?mode=ro" % DB, uri=True)
    try:
        row = con.execute(STATE_SQL, (USER_ID,)).fetchone()
        total = con.execute("SELECT COUNT(*) FROM users").fetchone()[0]
        nonempty = con.execute("SELECT COUNT(*) FROM users WHERE access_token IS NOT NULL AND access_token<>''").fetchone()[0]
        return row, total, nonempty
    finally:
        con.close()

def mask(tok):
    return "<empty>" if not tok else "%s...(%d chars, masked)" % (tok[:2], len(tok))

def get_with(token):
    req = urllib.request.Request(URL, method="GET")
    if token:
        req.add_header("Authorization", "Bearer " + token)
    try:
        with urllib.request.urlopen(req, timeout=10) as r:
            status, body = r.status, r.read().decode("utf-8", "replace")
    except urllib.error.HTTPError as e:
        status, body = e.code, e.read().decode("utf-8", "replace")
    safe = body.replace(token, "<TOKEN-REDACTED>") if token else body
    try:
        parsed = json.loads(safe)
        safe = json.dumps({k: parsed[k] for k in ("success", "code", "message") if k in parsed}, ensure_ascii=False)
    except Exception:
        pass
    return status, safe[:300]

before, total0, nonempty0 = snapshot()
print("[step0] baseline row=%s users=%d nonempty=%d" % (before, total0, nonempty0))

original_set = False
original = None
con = sqlite3.connect(DB, timeout=10.0)
try:
    original = con.execute("SELECT access_token FROM users WHERE id=?", (USER_ID,)).fetchone()[0]
    original_set = True
    print("[step1] original token shape: %s (is_null=%s)" % (mask(original), original is None))

    token = uuid.uuid4().hex[:32]
    con.execute("UPDATE users SET access_token=? WHERE id=?", (token, USER_ID))
    con.commit()
    stored = con.execute("SELECT access_token FROM users WHERE id=?", (USER_ID,)).fetchone()[0]
    print("[step2] wrote token: len=%s exact_match=%s" % (len(stored or ""), stored == token))

    status_a, body_a = get_with(token)
    code_a = json.loads(body_a).get("code") if body_a.startswith("{") else "?"
    print("[step3] GET with written PAT      -> http=%s code=%s" % (status_a, code_a))

    other = uuid.uuid4().hex[:32]
    status_b, body_b = get_with(other)
    code_b = json.loads(body_b).get("code") if body_b.startswith("{") else "?"
    print("[step4] GET with unrelated token  -> http=%s code=%s" % (status_b, code_b))
    print("[step3.body] %s" % body_a)
    print("[step4.body] %s" % body_b)
finally:
    if original_set:
        try:
            if original is None:
                con.execute("UPDATE users SET access_token=NULL WHERE id=?", (USER_ID,))
            else:
                con.execute("UPDATE users SET access_token=? WHERE id=?", (original, USER_ID))
            con.commit()
            after_raw = con.execute("SELECT access_token FROM users WHERE id=?", (USER_ID,)).fetchone()[0]
            print("[step5] restore: exact_equal_to_original=%s shape=%s" % (after_raw == original, mask(after_raw)))
        except Exception as e:
            print("[step5] RESTORE FAILED: %s: %s" % (type(e).__name__, e))
    con.close()

after, total1, nonempty1 = snapshot()
print("[step6] final row=%s users=%d nonempty=%d" % (after, total1, nonempty1))
print("[step6] row_identity_unchanged=%s token_field_restored=%s" % (after[:3] == before[:3], after[3:] == before[3:]))
```

## 7. 结论

1. **机制**：鉴权链中「令牌 → 用户」的解析（`model.ValidateAccessToken` → `model/user.go:1118`）是裸 DB 查询、
   **不经过缓存**；用户基信息的缓存层（`GetUserCache`）在本实例因 `REDIS_CONN_STRING` 未设置而不启用。
   因此写入 `access_token` **即时生效，无需重启**——脚本未重启、`RestartCount=0`，实测亦即时生效。
2. **实测**：有效 root PAT 访问 `GET /api/status/update` → **403 + `code: "pat_not_allowed"`**，
   与期望完全一致；对照的无关令牌 → 401 `AUTH_UNAUTHORIZED`；**无任何放行（update 路由无 200）**。
3. **还原**：原值为 SQL NULL，已还原为 NULL 并三方复核（脚本自证 + 只读复查 + 基线对比），
   `users` 表无非空 `access_token` 残留，实例健康、未重启、无审计残留。
4. **未实测项**：两条 POST 路由（apply/rollback）按红线未探，其 `pat_not_allowed` 由同一守卫在
   `controller/update_check.go:36/59` 首行同源调用支撑（代码论证，非实机证据）。
5. **过程缺陷（已修正并记录）**：首版脚本还原语句在 `None` 绑定上抛异常，导致令牌短暂残留；
   当轮显式 NULL 还原 + 修正脚本复跑，最终状态干净（详见 §4）。
