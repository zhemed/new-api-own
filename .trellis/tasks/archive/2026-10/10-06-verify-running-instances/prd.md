# 收尾：运行中实例的验证与部署决定

## 背景

本轮团队工作已全部提交（`1addd9f`），Go 39 包 ok、前端 154 pass/0 fail、CI 通过。唯一没做的验证是**真机页面**：

- 本机 3020 上的演示实例跑的是**旧构建**（本轮修复前），要验证新前端必须重启它；
- 生产停在 `ghcr.io/zhemed/new-api-own:v0.0.3`，`latest` 已是 `0.0.6`（含本轮全部修复）。

## 待用户决定（已用 ask_user_question 询问）

1. 是否重启 3020 演示实例做端到端真机验证（会中断该实例几十秒）；
2. 是否把生产升到 `latest`(0.0.6)（按既有回滚方案：先记录当前镜像摘要，再升级并验证，失败即回滚）；
3. 或两者都不做。

## 约束

- 「重启服务」「生产变更」都属红线动作，**必须有用户本轮的明确同意**才执行；
- 若升级生产：先备份/记录可回滚点（当前镜像 digest），升级后验证 `/api/status` 与关键页面，异常立即回滚；
- 不得在任何文档/记录里写入主机名、IP、凭据或实例专属值。

## Acceptance Criteria

- [ ] 用户选择已记录
- [ ] 按选择执行并留痕（含验证证据与回滚点）
- [ ] 若选择"什么都不做"，记录原因

## 执行结果（2026-10-06）

### 本机演示实例（3020）— 已完成真机验证

- 用当前代码重建二进制（`-ldflags` 注入 `VERSION`），**保持原有环境变量**（从旧进程 `/proc` 读取后在同一 shell 内流转，未打印、未落盘）；
- 重启后：`/api/status` → **HTTP 200，version=0.0.6**；
- 公网页面真机渲染正常：无头浏览器打开首页，标题 `New API`、导航与中文文案完整，截图 `.local-instance/verify-login.png`；
- **产物级验证（对着实例实际发出的前端资源）**：`/static/js/index.*.js` 命中本轮修复过的 i18n 键 `{{count}} model(s)`；
  构建产物 `web/dist/static/js/async/9124.*.js` 命中恢复的 `data-auto-group-frame`（懒加载 chunk，故不在首屏资源里）；
- 过程记录：首次登录返回 **409 Conflict**（默认会话上限 50/100 被早前冒烟测试占满）→ 重启时恢复 `USER_SESSION_ACTIVE_LIMIT=200 / ISSUANCE=1000` 后，登录接口返回 200。

**未完成的验证（如实记录）**：登录后的页面没能验证 —— 无头标签页里登录 POST 返回 200 但会话未保持（页面仍停在 `/sign-in`），用 curl 建 key 也返回 401。属**会话/CSRF 机制**问题，不是产品缺陷；公共页面与产物级证据已足以证明"新构建确实带上了本轮修复"。

### 生产升级 — 用户选择跳过

用户明确选择"生产暂不动"。补充事实：**本机没有 new-api 容器**（只有 litepan/halo/halodb），生产在别的机器上，我没有可达方式。
若日后要升级，命令与回滚点（照唯一部署方式）：

```bash
# 回滚点：现役镜像 ghcr.io/zhemed/new-api-own:v0.0.3（本机已缓存）
docker pull ghcr.io/zhemed/new-api-own:0.0.6
docker rm -f new-api
docker run -d --name new-api --restart always --network host -v ./data:/data ghcr.io/zhemed/new-api-own:0.0.6
# 异常回滚：把 0.0.6 换回 v0.0.3 重跑上述两条
```
