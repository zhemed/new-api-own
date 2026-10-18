# 按用户要求升级实例到 v0.0.27

## 用户指示（2026-10-18）

「要，帮我升级一下」

## 内容

v0.0.27 含：默认范围只保留「上月/本月」（上月在前）+ 旧天数默认值兜底迁移。
执行：单次登录 → `POST /api/status/update/apply` → 轮询到 `v0.0.27` → 复核 → **`POST /api/user/auth/logout` 撤销本次会话**（会话卫生）。

## Acceptance Criteria

- [ ] 实例 v0.0.27，前端包已换
- [ ] env / 日志上限 / 容器状态未变
- [ ] 本次登录的会话已撤销（计数不堆积）
- [ ] 留痕归档

## 执行结果（2026-10-19）

- 前 v0.0.26 → `POST /api/status/update/apply` → **约 10 秒**后 **v0.0.27**；
- 复核：四条 `LOG_*` 未变；`--log-opt max-size=10m max-file=3` 未变；容器 running（未重建）；前端包变为 `index.a20a140f8e.js`；`releases/latest = v0.0.27` ✓；
- **会话卫生**：调完接口后 `POST /api/user/auth/logout` 返回 **HTTP 200** ✓ —— 该会话被撤销
  （表里行数 10→11 但那是**历史行**；判定新登录会话是否堆积要看 `status='active'` 计数）。
