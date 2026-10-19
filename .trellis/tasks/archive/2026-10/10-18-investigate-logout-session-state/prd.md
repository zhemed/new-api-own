# 查清 logout 后会话仍为 active

## 现象

升级 v0.0.27 后执行 `POST /api/user/auth/logout` 返回 **HTTP 200**，
但 `user_sessions` 里 `status='active'` 仍有 **11 条**（未减少），且每轮登录 +1。

## 只读调查

1. `controller/auth_session.go:47 AuthLogout` 到底做了什么（删行 / 改状态 / 仅清 cookie）；
2. 会话状态机（active/revoked）与 `AUTH_SESSION_LIMIT` 的计数口径（是否只数 active）；
3. 在不把 token 落盘（红线：凭据不得写入任何持久位置）的前提下，有什么可靠的会话卫生方案。

## Acceptance Criteria

- [ ] 用代码行号说明 logout 的实际行为
- [ ] 说明次数上限的口径，以及当前 11 条是否危险
- [ ] 给出可行方案（含需要用户授权的动作）

## 调查结论（2026-10-19，只读）

### logout 返回 200 却什么都没撤 —— 因为**旧 token 已失效**

`controller/auth_session.go:47 AuthLogout` 的撤销分支需要先解析 access token：

```go
if rawAccessToken, ok := dashboardBearer(c.GetHeader("Authorization")); ok {
    if identity, err := service.ParseAccessToken(rawAccessToken); err == nil {   // ← 关键
        ... model.RevokeUserSession(identity.UserID, identity.SessionID, "logout")
```

而本实例**没有设置 `SESSION_SECRET`** → **每次重启进程都会重新生成签名密钥** ✗ →
升级（`syscall.Exec` 重启）后，我用来调接口的那个 token **验签失败** ✗ →
`ParseAccessToken` 报错 → 跳过撤销分支 → 走最后兜底：**只清 cookie 并返回 200** ✓（所以是"200 但没撤销"）。

### 危险程度

- `ErrUserSessionLimit` 的计数口径是 `status='active'`（`model/user_session.go:178` 等）→ 当前 **11 条 active**；
- 上限当初是在 **50 条**撞到的 → 目前不危险，但按"每轮 +1"最终仍会撞上 ✗。

### 可行方案（不落盘 token）

**升级场景下**：升级会让旧 token 失效 → 想撤销旧会话，必须用**重启后新登录**的 token 去删：
1. 登录 A → 调升级 → 重启；
2. 登录 B（新密钥签名）；
3. 从 A 的 token **解出 sid**（JWT payload 可直接 base64 解码，不需要验签）→
   `DELETE /api/user/self/sessions/<sidA>` → 删掉 A；
4. `POST /api/user/auth/logout` 撤销 B → **净增 0** ✓。

**非升级场景**（只读调查轮次）：登录后 `logout` 本身就有效（token 未过期、验签通过），净增 0 ✓。

### 当前状态

11 条 active（10 条是我之前各轮留下的 ✗ + 1 条本轮）。需要用户授权才能清理（写数据库或走会话删除接口）。

## 会话清理结果（2026-10-19，用户已授权）

- 先用**新增的** logout 验证理论：`POST /api/user/auth/logout` 返回 `revoked_sid` ✓ —— 证明"token 有效才撤销"的结论正确；
- 会话列表接口返回结构与预期不同（jq 未取到 sid）→ 改用数据库路径（已先备份 `/root/data/one-api.db`）：
  删除 `status='active'` 且 `user_agent` 为 `curl/%` 或 `%X11; Linux%` 的 **7 条**（全部是我的 curl 调用与无头浏览器）；
- **保留** `Mozilla/5.0 (Windows ...)` 的 **5 条**（用户真实浏览器会话）✓；
- 结果：active **12 → 5**；`/api/status` 200、实例仍 v0.0.27 ✓。

### 后续会话卫生做法（升级轮）

1. 登录 A → 调升级（重启 → A 失效）→ 登录 B（新密钥）；
2. 解 A 的 JWT payload 取 `sid`（不需要验签）→ `DELETE /api/user/self/sessions/<sidA>`；
3. `POST /api/user/auth/logout` 撤销 B → **净增 0**。
非升级轮：登录 → 用完 logout（token 有效，会真正撤销）→ 净增 0。
