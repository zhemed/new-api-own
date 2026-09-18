# 清除仓库内的部署细节污染（内网 IP/主机/实例信息）

## Goal

仓库是公开的，不应包含部署环境信息：扫描并清理内网 IP（10.x/192.168.x/fc00::）、内网域名（线上实例）、主机名、实例专属值（实例专属兜底值）、以及 /root 本机路径；同时定下『部署细节只放非公开位置』的规则；评估是否需要重写 git 历史

## Requirements

- TBD

## Acceptance Criteria

- [ ] TBD

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
