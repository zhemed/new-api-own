# 按用户要求升级到 v0.0.29 并盯测

## 用户指示（2026-10-19）

「要，帮我升级一下试试」

## 内容

v0.0.29 含：图表标题「总计」数值提亮加粗（两处）。
执行：登录 A → apply → 采样至 v0.0.29 → 复核（容器 RestartCount/StartedAt、日志无 panic、env、日志上限、前端包）→ 登录 B → 按 UA 精确清理本轮 curl 会话 + logout B → active 净增 0。

## Acceptance Criteria

- [ ] 实例 v0.0.29、前端包更新
- [ ] 容器未重建、日志无异常、env/日志上限未变
- [ ] active 会话数升级前后一致
- [ ] 留痕归档

## 执行结果（2026-10-19）

### 升级（全程采样）

- `apply` → **+24s** 仍是 v0.0.28 → **+26s** 变为 **v0.0.29** ✓；
- **累计中断采样 0 次**（每次采样都是 HTTP 200 —— 原地重执行的切换窗口小于 2s 采样间隔）；
- 容器：running / **RestartCount 0** / StartedAt 仍是 2026-10-16 ✓（未容器重启）；
- 日志：重启后正常打印访问地址 + `GET /api/status 200`，**无 panic/error** ✓；
- env 四条 `LOG_*` 与 `--log-opt max-size=10m max-file=3` 未变 ✓；前端包 `index.b907d24f59.js` ✓；`releases/latest = v0.0.29` ✓。

### 会话

- 我本轮产生的 curl 会话已清空（`curl/7.81.0` 现在 active **0** 条，revoked 2 条为历史）；
- 当前 active **6 条全部是 `Mozilla/5.0 (Windows ...)`** —— 即用户自己的浏览器会话（期间用户又登录了一次），**不是**我造成的 ✓。

### 过程备注（自报）

查询 latest 时我先写错了 API 路径（`release/latest` → 404），已用 `releases/latest` 重新确认 ✓。
