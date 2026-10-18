# 按用户要求升级实例到 v0.0.25

## 用户指示（2026-10-18）

「要，帮我升级一下」

## 内容

v0.0.25 含：自然月快捷范围（本月/上月）+ x 轴末标签对齐 + （前一版）Token 单位 K/M/B/T。
执行：单次登录复用 token → `POST /api/status/update/apply` → 轮询 `/api/status` 到 `v0.0.25`；
复核版本、前端包 hash、四条 `LOG_*`、`--log-opt`、容器状态、会话数不堆积。

## Acceptance Criteria

- [ ] 实例 v0.0.25，前端包已换
- [ ] env / 日志上限 / 容器状态未变
- [ ] 会话数无堆积；留痕归档

## 执行结果（2026-10-18）

- 升级前 v0.0.23 / 会话数 7 → `POST /api/status/update/apply` → **约 10 秒**后 **v0.0.25**；
- 复核：四条 `LOG_*` 未变；`--log-opt max-size=10m max-file=3` 未变；容器 running（未重建）；
  前端包变为 `index.4ce8275950.js`；会话数仍 7（单次登录复用 ✓）；`releases/latest = v0.0.25` ✓。
- 用户可在筛选对话框看到新增的「本月 / 上月」快捷范围；x 轴末标签应为当天。
