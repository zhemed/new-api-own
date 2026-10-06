# 执行 compose 清理 + 强制唯一部署方式

## Goal

按用户要求：删除 docker-compose.yml / docker-compose.dev.yml 及其引用；并把“本项目只有一种部署方式（docker run + 公开镜像）”做成强制约束——文档铁律 + 机械闸门（提交/CI 均拦截 compose/K8s 等第二种编排形态），并加 do-not-restore 清单与“回滚后必须核验”规则。

## Requirements

- TBD

## Acceptance Criteria

- [ ] TBD

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
