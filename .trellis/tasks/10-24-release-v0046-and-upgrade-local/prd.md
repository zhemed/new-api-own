# 发 v0.0.46 并升级本机实例

## Goal

用户点名授权（2026-10-24）：① 发新版——把自更新资产白名单修复（`f843c3f` / `6820971`）发出去；
② 发版后用**镜像方式**升级本机实例，使其面板自更新恢复可用。

## 背景事实（已核实）

- 发布链路：推送版本 tag（`v0.0.46`）会**同时触发**两个 workflow：
  - `docker-build.yml` → 构建并推送多架构镜像（`v0.0.46` / `0.0.46` / `latest`）；
  - `release.yml` → 建 GitHub Release，附 `new-api-linux-amd64`、`new-api-linux-arm64`、`SHA256SUMS`、
    **`SHA256SUMS.sig`**（ed25519 分离签名；缺 `UPDATE_SIGNING_KEY` 时**拒绝发布未签名 Release**）。
- 两个 workflow 并行、互不等待（workflow 注释写明这是刻意取舍），因此**必须分别确认成功**。
- tag 注解即 Release 正文（惯例：一条条 changelog）。
- 本机 `gh` CLI 已安装且已登录（账号 `zhemed`），可用于监控 run 与查看 release 资产。
- 本机实例当前 v0.0.45（旧代码，面板更新不可用）；数据在 `./data`。

## Requirements

- R1 版本提升：`VERSION` 由 `0.0.45` → `0.0.46`，提交带锚点 `[task:release-v0046-and-upgrade-local]`。
- R2 tag：**附注 tag** `v0.0.46`，注解正文写 changelog（本次修复 + 运维提示）。
- R3 推送：`main` 与 tag 都推 `origin`（外部副作用动作，用户已授权）。
- R4 监控：等两个 workflow 都结束；**必须双绿**。任一失败 → 停止，不升级实例，如实报告。
- R5 资产核验：`gh release view v0.0.46` 必须含四个资产，且 `SHA256SUMS.sig` 存在（这正是本次修复的关键资产）；
  镜像 tag 必须可见。
- R6 升级本机实例（停机几十秒）：拉 `v0.0.46` → `docker rm -f new-api` → 按**原参数**重建
  （`--network host`、`--restart always`、`--log-opt max-size=10m --log-opt max-file=3`、
  `-v /root/new-api-own/data:/data`、四个 `LOG_*`）。
- R7 升级后核验：容器 running、`restarts=0`、挂载仍为工作区路径、`/api/status` 200 且
  `X-New-Api-Version: v0.0.46`；8 个容器全 Up；卷仍 3 个；`data/` 55 文件；其它容器不受影响。
- R8 修复确实在运行的二进制里：容器内 `strings /new-api | grep -c signature_asset_missing`（只读检查）有命中；
  **不得**用该证据冒充「面板更新端到端已验证」。
- R9 回滚点：`v0.0.45` 镜像仍在本地；若新版异常，按同参数用 `v0.0.45` 重建即可。
- R10 红线：不动其它容器/镜像/卷、不删 `data/`、不 force push、不改部署形态。

## Acceptance Criteria

- [ ] AC1 `VERSION` = `0.0.46`，提交带锚点；`v0.0.46` 附注 tag 存在且注解含 changelog。
- [ ] AC2 `origin/main` 与 tag 均已推送；`git rev-list --left-right --count` 为 `0 0`。
- [ ] AC3 两个 workflow 双绿（附 run id 与结论）。
- [ ] AC4 Release 含四个资产（含 `SHA256SUMS.sig`）；镜像 tag 可见。
- [ ] AC5 本机实例升到 `v0.0.46` 且健康（R7 各项）；`data/` 与卷无损。
- [ ] AC6 运行中的二进制含修复（R8 的只读证据）。
- [ ] AC7 明确记录：**面板更新的端到端验证需要下一个版本**（本次只能证明「修复已在运行的二进制里 + 发布侧资产齐全」）。
- [ ] AC8 收尾由 `scripts/trellis-wrapup.sh` 一条命令完成并推送。

## Notes

- 另一实例（生产）未获点名主机，**本轮不动**；其升级路径与命令已在上一轮交付里给出。
- 面板内更新的完整闭环演示留待下一次发版（v0.0.46 → v0.0.47 时可在面板里实测）。
