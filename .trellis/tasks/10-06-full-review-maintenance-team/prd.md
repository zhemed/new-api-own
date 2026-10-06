# 全面维护与审查（Agent Teams）

## 用户指示

「调用团体开始全面维护和审查」——明确授权使用 Agent Teams。

## 组织（Lead = 本会话）

| 成员 | 写范围（互不重叠） | 交付 |
|---|---|---|
| `review-backend` | Go 代码（model/ relay/ relaykit/ common/ service/ controller/ middleware/ setting/ types/ router/ dto/ constant/ i18n/）| 发现清单（附证据）+ 低风险修复 + vet/build/relaykit/make test 实测 |
| `review-frontend` | `web/**` | 发现清单 + 可证明安全的修复（i18n 必须走脚本；本机无 bun 需如实标注未验证项）|
| `review-ops` | `.github/ docs/ README* MAINTENANCE AGENTS CLAUDE .env.example .gitignore scripts/ .githooks/` | 文档/配置与现状一致性、仓库卫生、敏感信息扫描（只报文件不复述）|

Lead 本人：共享任务看板、最终 diff 复核、全套门禁（guards + vet/build + make test）、提交与推送、汇总汇报。

## 硬约束（对全员生效）

- 禁止网络动作、禁止访问生产、禁止读工作区外文件；禁止 git commit/push/tag（由 Lead 统一提交）；
- 不许动 `.local-instance/`（演示实例在跑）；不许改彼此范围；
- 发现必须附证据（文件:行），不确定的标"待确认"，禁止臆断。

## Acceptance Criteria

- [ ] 三条线的发现清单与修复清单齐备（附证据）
- [ ] 各自范围内验证命令实测并贴结果（无法验证的如实标注）
- [ ] Lead 复核 diff、跑全套门禁、提交推送、给出汇总
