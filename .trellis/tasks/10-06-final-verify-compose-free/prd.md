# 收尾核验：compose 残留与闸门状态

## Goal

清理与闸门落地后做一次只读收尾：确认 `git ls-files | grep -i compose` 命中的两个路径是什么、
闸门是否仍通过、生产与本机演示实例当前状态。

## Acceptance Criteria

- [ ] 两个命中路径定性（是残留还是误报）
- [ ] `scripts/forbid-extra-deploy-methods.sh` 通过
- [ ] 生产与本机实例状态一句话交代

## 核验结果（2026-10-06）

- `git ls-files | grep -i compose` 命中 2 个路径：`.trellis/tasks/archive/2026-10/10-06-audit-compose-cleanup-history/{prd.md,task.json}`
  —— **只是这份核查记录的文件名含 "compose"**，不是部署产物；仓库里已无 compose / Helm / K8s 任何形态。
- `scripts/forbid-extra-deploy-methods.sh` → ✅ 通过；CI `trellis-gate` 含该步骤，同样 success。
- 生产：容器 `ghcr.io/zhemed/new-api-own:v0.0.3`，与"唯一部署方式（docker run + 公开镜像）"一致。
- 本机演示实例：仍在 3020 运行（`LOG_SQL_DSN=memory`，30 秒清理 / 上限 50）。

## 未决项

`new-api.service`（systemd unit）是否也视为第二种部署方式 —— 等用户定夺。
