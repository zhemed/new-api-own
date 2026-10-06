# 执行 compose 清理 + 强制唯一部署方式

## 用户定调（2026-10-06）

「我们只有一种部署方式，也是唯一一种，必须强制约束了。」

## 做了什么

| 层面 | 内容 |
|---|---|
| 删除 | `docker-compose.yml`、`docker-compose.dev.yml` |
| 机械闸门 | 新增 `scripts/forbid-extra-deploy-methods.sh`：按**文件名**（`(docker-)?compose*.y*ml`）、**目录名**（`charts/`、`helm/`、`k8s/`、`kubernetes/`、`manifests/`）、**内容特征**（`services:` / `network_mode:`，排除 `.github/`）三重判定 |
| 提交当场 | `.githooks/pre-commit` 调用该脚本 → 暂存区出现 compose 形态直接拒绝提交 |
| 远程兜底 | `.github/workflows/trellis-gate.yml` 新增步骤「检查部署方式唯一」→ CI 红叉 |
| 规则源 | 新增 `.trellis/spec/guides/deployment-single-method.md`：唯一命令、禁止清单、三层闸门、**do-not-restore 清单**、**回滚后必须核验**、事故记录 |
| 文档同步 | `README.md` / `README.en.md`（唯一部署方式，删掉"方式二/生产部署参考"）、`AGENTS.md`（新增强制段）、`MAINTENANCE.md`（铁律第 5 条 + 安全基线三处改写）、`docs/installation/BT.md`（**删掉"方法二：Docker Compose"整节**，更新步骤改回 docker run）、`docs/FILE_INVENTORY.md`、`.trellis/spec/backend/quality-guidelines.md` |
| 附带修好 | `.trellis/gates/enforce-from` 审计起点指向的提交已随强推回滚消失，CI 因此**一直红**；起点改到当前基线 `8a99114`（v0.0.3） |

## 为什么做成闸门而不是只写文档

`docker-compose.yml` **两删三回**（`843f988` 删除 → `770e26c`/`a29cd82`/今天的 `rollback-to-0-0-3` 三次回滚带回）。
结论写进规则源：**"文档写了规则"挡不住回滚，"提交/CI 拦得住"才挡得住。**

## 验证

- 闸门双向验证：删除前跑 → 拦住两个 compose 文件；删除后跑 → 通过；
- `pre-commit` 自测：本次提交经新钩子放行（无 compose）；
- CI：`trellis-gate` 全绿，其中「检查部署方式唯一（禁止 compose/集群编排）」步骤 **success**；
- 全仓库复查：除规则文本与桥接说明外，**再没有教 compose 部署的内容**。

## 未处理（需你定夺）

`new-api.service`（systemd unit，514B）也是"第二种部署方式"的一种形态；本次**没动**，
待你确认是删还是保留（保留则闸门不拦它）。

## Acceptance Criteria

- [x] 两个 compose 文件删除，所有引用同步改写
- [x] 唯一部署方式写入 AGENTS.md / MAINTENANCE.md 铁律 / 规则源
- [x] 提交当场 + CI 双闸门（已实测双向）
- [x] do-not-restore 清单 + 回滚后核验规则
- [x] CI 恢复全绿（顺带修好被回滚搞坏的审计起点）
- [ ] `new-api.service` 的处置（等用户确认）
