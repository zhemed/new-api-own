# 回滚到迁移前的部署方式

## Goal

用户判定：迁移后出现大量问题，要求还原之前的部署方式。回滚范围：删除 install-compose.sh、compose.yaml 恢复为 docker-compose.yml（默认口令、restart: always、redis:latest、无额外健康检查与 depends_on 条件）、README/README.en/MAINTENANCE/BT.md 恢复迁移前内容。不触碰任何部署的数据与卷。

## Requirements

- TBD

## Acceptance Criteria

- [ ] TBD

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
