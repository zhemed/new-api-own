# 设计：UI 自升级二进制的改进取舍

## 对照实际情况的评估（Lead）

| 项 | 现状（实测） | 我们这里的实际风险 | 结论 |
|---|---|---|---|
| P0 权限 | 两条路由 `middleware.AdminAuth()`（`router/api-router.go:25-26`），且鉴权接受 PAT | 自维护 fork，管理员基本只有站点所有者；但"替换二进制"等价于 root 级能力，管理员/PAT 泄露即可拿下整个容器 | **做**：提 `RootAuth` + **拒绝 PAT**，前端入口按 root 显隐 |
| P1 签名 | 只比对同源 `SHA256SUMS`（`service/update_apply.go:202+`），二进制无 cosign | 镜像侧有 cosign，但二进制的"签名校验"若引入 cosign keyless 需要 fulcio/sigstore 依赖；我们只需要"防篡改 + 离线可验" | **做（改方案）**：用 **ed25519 分离签名** 签 `SHA256SUMS`，CI 用 GitHub Secret 私钥，二进制内嵌公钥、用 Go 标准库验证（零新依赖）；新增稳定错误码 `signature_mismatch` |
| P1 不持久提示 | 仅代码注释说明（`update_apply.go:52-53`），API/UI 无提示 | 用户会误以为升级后长期生效 | **做**：响应加 `persistent:false` + `hint`；UI 按钮旁与结果提示各一次 |
| P2 回滚点 | 无 `.bak`/prev；`not_newer` 拒绝降级；唯一回退是重建容器 | 生产实例在远端，回滚靠重建容器成本高 | **做**：替换前把旧二进制保留为 `<exe>.prev`，新增 `POST /status/update/rollback`（**仅从本地 `.prev` 回滚，不新增下载旧版本能力**）|
| P2 优雅退出 | `syscall.Exec` 直接替换进程镜像（`update_apply.go:363+`）| 升级是管理员手动触发、低频；实测新进程 ~1s 就绪，影响是"触发那一刻的在途请求" | **评估后暂缓**：需要把 `http.Server` 的 drain 钩子注入 update 包（跨 main/包边界），收益低而回归面大；写出残余影响与后续做法 |

## 关键设计

### 1. 权限（P0）
- 路由改 `middleware.RootAuth()`；**新增**一个中间件（或复用 RootAuth 后加判定）**拒绝系统访问令牌（PAT）** 调用 update 能力；
- 理由：PAT 常被用于脚本/第三方集成，权限边界模糊；二进制替换应由"人 + 最高权限会话"执行；
- **并存问题**：RootAuth 与会话/PAT 的判定在 `middleware/auth.go`，需要在该文件里区分"本轮用的凭据类型"，把结论写进上下文供 handler 判定；不改其它路由行为。

### 2. 签名（P1，ed25519 方案）
- CI（`release.yml`）在生成 `SHA256SUMS` 后，用 **ed25519 私钥**（GitHub Secret，例如 `UPDATE_SIGNING_KEY`）对 `SHA256SUMS` 内容签名，产出 **`SHA256SUMS.sig`**（base64 或 hex 文本，一行）；
- 二进制内嵌 **公钥**（Go `const`），更新时下载 `SHA256SUMS` + `SHA256SUMS.sig`，用 `crypto/ed25519` 验签；**验签失败/缺签名 → 拒绝替换**，返回新错误码 `signature_mismatch`；
- 保留原 SHA256 校验（验签通过后仍逐个比对），失败码不变；
- 公钥轮换：预留"多个公钥"结构（列表），避免换钥即断更；
- 兼容：若线上资产暂无 `.sig`（老版本），**必须拒绝**而不是降级放行（安全优先），需在文档写明"从本版本起必须带签名"。

### 3. 不持久提示（P1）
- 响应结构加：`persistent: false`、`hint`: "本次为容器内就地替换；重建容器会回退到镜像内版本"，`restart_mode`（保留）；
- UI：按钮下方常驻一行小字 + 成功/超时结果里再出现一次。

### 4. 回滚（P2）
- 替换前：把当前可执行文件（`os.Executable()`）复制/改名到 `<exe>.prev`（同目录、保持权限）；替换失败则不动 `.prev`；
- 新增 `POST /status/update/rollback`（RootAuth + 非 PAT）：校验 `.prev` 存在 + 体积/权限合理 → 原子换回 → 走与 apply 相同的激活路径；
- **不做**"下载旧版本"能力；`not_newer` 语义不变（apply 仍不允许降级）；
- 回滚失败必须原样保留当前二进制（与 apply 同一套临时文件语义）。

### 5. 优雅退出（P2，暂缓的理由）
- 现状：`Activate` 在响应 Flush + 500ms 后 `syscall.Exec`；进程镜像被替换，在途请求断开；
- 评估：若要 drain，需要 `main` 把 `*http.Server` 注入 update 包，`Shutdown(ctx)` 后 exec；跨边界、影响启动流程、回归面大；
- 残余影响：**触发更新的那一刻**在途请求会失败（实测新进程约 1s 就绪，服务中断窗口很短，且升级由管理员手动触发）；
- 后续做法（不在本轮）：在 `main` 里注册 `drainBeforeExec` 钩子（`Shutdown` 带 5–10s 超时），update 包在 exec 前调用；需专项测试与演练。
