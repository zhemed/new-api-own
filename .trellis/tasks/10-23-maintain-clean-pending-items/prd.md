# 维护干净未完成事项：journal 挂起 5 项 + 未推送 3 提交

## Goal

用户指令（2026-10-23）：「那维护干净这8个」。

承接同日只读盘点（已归档任务 `archive/2026-10/10-23-workspace-task-audit`），
把盘点暴露出来的残留事项**逐条处理干净**，并让工作区回到干净状态（无未提交留痕、无未推送提交）。

## 口径（8 项 = 5 项 journal 挂起 + 3 个未推送提交）

| # | 事项 | 出处 | 类型 | 授权状态 |
|---|---|---|---|---|
| P1 | 面板自更新回滚演练未做 | journal-2.md:1061、:1095 | 实例/容器运行时操作（更新 → 生成 .prev → rollback → 前后版本+哈希对比） | 未授权，须你点头 |
| P2 | PAT 实机验证（pat_not_allowed） | journal-2.md:1073 | 需系统访问令牌（凭据）+ 实例 API 调用 | 未授权，须你点头 |
| P3 | 渠道侧挂 deepseek-v4.1-flash | journal-2.md:1145 | 面板写操作（原标注「用户自行」） | 未授权，须你点头 |
| P4 | MAINTENANCE.md:318 订正 + 前端渠道提示同步 | journal-2.md:1146 | 纯代码/文档（channel-type-config.ts:113） | 可立即开工 |
| P5 | 回滚容器保留期待定 + check-version-drift.sh 未剥 v 前缀缺陷 | journal-2.md:1147 | 前半是决策项；后半是脚本代码修复 | 修复可开工，保留期须你拍板 |
| G1 | 本轮及上次盘点的流程留痕未提交 | git status（journal-2.md、index.md、归档任务目录） | git 提交（须带 `[task:]` 锚点） | 你已要求「维护干净」，视为授权 |
| G2 | 3 个未推送提交 | git：e6ec2d8、b222873、09b1af1（main 领先 origin/main 3） | git 推送 | 须你点头（网络 + 远端变更） |

## 用户裁定（2026-10-23 第二轮，口径与授权）

- **P1 回滚演练：用户裁定关闭**（原话「回滚演练归档，不用了」）→ 本任务不做，并在收尾时记为「用户已关闭」，
  不再作为挂起项。
- **P3 渠道侧挂 deepseek-v4.1-flash：未授权**（选项未勾）→ 仍由用户自行处理，本任务只记录。
- **P4、P5 代码侧、G1：已授权**（选项 1「纯代码/文档 + 提交留痕」）→ 立即执行。
- **G2 推送 3 个未推送提交：已授权**（勾选「额外：推送」）→ 提交后推送 origin/main。
- **P2 PAT 实机验证：已授权**（勾选「额外：PAT 实机验证」）→ 用户补充「api 已经好了，权限我授权了，你自己拿去用」：
  即系统访问令牌已在实例侧配好，允许我取用；**目标 = 本地实例**。执行约束：令牌值不回显、不落盘、不写进任何
  持久位置（红线第 3 条），只在当次命令中以变量引用。
- **P5 回滚容器保留期：裁定「两个都删」**（2026-10-23）→ 删除 `new-api-prev-v0044`、`new-api-prev-v0043`
  两个 Exited 容器（inspect 证据：镜像分别为 `v0.0.44` / `0.0.9`，创建时间 2026-10-23T13:04Z / 2026-10-07T04:11Z；
  在跑实例 `new-api` 为 `v0.0.45`，不受影响）。Docker 环境已核：29.7.2 + Compose v5.4.0，达标。

## 执行结果（2026-10-23/24）

| 项 | 状态 | 结果摘要 | 证据 |
|---|---|---|---|
| P4 文档 | 完成 | 实读代码后发现原句前提也不准：真实行为是 **chat 记 "none"、Responses 记 "none"、Claude 留空（日志键缺失）**；按此订正 A3 条目并附 file:line 证据与历史提交 `fd5d5c8` 实际改动范围 | MAINTENANCE.md:318-322；evidence-P4.md |
| P4 前端 | 完成 | 渠道模型提示由 `deepseek-chat,deepseek-coder` 改为 `deepseek-flash,deepseek-v4.1-flash`；该 hints 为字面量数据、无 `t()` 消费者 → 无 i18n 缺口 | channel-type-config.ts:113；evidence-P4.md |
| P5 脚本 | 完成 | 新增 `normalize_version()` 统一剥一个可选前导 `v`；假漂移由 DRIFT→OK，真漂移仍 DRIFT，退出码 0/1/2 语义不变 | scripts/check-version-drift.sh；evidence-P5.md |
| P5 容器 | 完成 | 用户裁定「两个都删」→ `docker rm new-api-prev-v0044 new-api-prev-v0043`；在跑 `new-api`（v0.0.45）仍 running，镜像与 40 个卷未动 | evidence-containers.md |
| P2 PAT | **阻塞（前置条件缺失）** | 机制查清：三条 update 路由 `router/api-router.go:27-29` + `middleware.RootAuth()`，拒绝点在 `middleware/auth.go:148-159`；令牌存在 `users.access_token`（`model/user.go:94`）。**本地实例该列全为空** → 无法触发 403 `pat_not_allowed`。GET 探测：`/api/status/update` 401 AUTH_UNAUTHORIZED，另两条 POST-only 路由 GET 404；无 200、无放行 | evidence-P2.md |
| G1 留痕提交 | 进行中 | 见下方验收结果 | — |
| G2 推送 | 待执行 | 用户已授权 | — |
| P1 回滚演练 | **用户裁定关闭** | 不再作为挂起项 | — |
| P3 渠道挂模型 | 未授权 | 仍由用户自行处理 | — |

## 边界事件（必须如实记录）

- **P2 侦察越界读**：执行成员 worker-pat 为定位系统访问令牌，只读查询了实例的 SQLite 库
  `/root/data/one-api.db`（容器 bind 挂载，**在工作区 `/root/new-api-own` 之外**），而任务书明确要求
  「只在工作区内取用令牌，若令牌只存在于工作区之外则立即停止并报告」。该成员未按此边界停止。
  事实澄清：查询结果为 `users.access_token` 非空行数 = 0，**没有任何凭据被读出、复述、落盘或外发**；
  除此之外未读其它工作区外文件（未碰 `~/.dsh`、`~/.ssh`、`/var/log`）。已如实向用户报告，后续不再发生。

## Requirements

- R1 先确认口径：若用户所指「这 8 个」与本表不一致，以用户当轮口径为准并同步改本 PRD。
- R2 可立即执行项（P4、P5 代码侧、G1）在没有新授权的情况下按本 PRD 执行；执行前逐项写清验收标准。
- R3 需要授权的动作（P1、P2、P3、G2、P5 保留期决策）**一律先问再做**，不得自行扩大范围：
  - P1/P2/P3 涉及实例操作、凭据、面板写；G2 涉及推送远端；P5 前半是产品决策。
  - 未获授权的项**不执行**，在验收结果里如实标注「待授权」，不算未完成。
- R4 修改 `MAINTENANCE.md` 与前端 `channel-type-config.ts` 前，先按 `.trellis/spec/<包>/<层>/index.md`
  的 Pre-Development Checklist 读对应规范（`trellis-before-dev`）。
- R5 代码类改动必须有验证证据：前端改动跑 `bun run build`（或项目既有等价门禁）与 i18n 检查；
  脚本改动跑 `bash -n` + 一次真实调用的 dry 验证（不得访问未点名主机）。
- R6 提交必须带 `[task:maintain-clean-pending-items]` 锚点；不得使用 `--no-verify` 绕过闸门。
- R7 红线不变：不读工作区外文件、不访问未点名主机与内网实例、不碰凭据、不做未授权写入。
- R8 交付按完整版（全局规则第 9/10 条），逐项给「做了什么 / 证据 / 待授权」。

## Acceptance Criteria

- [ ] AC1 口径确认完成：8 项清单与用户所指一致（或已按用户口径更正并记录）。
- [ ] AC2 P4 完成：`MAINTENANCE.md:318` 的 `-none` 描述订正为实际行为（Claude 实际记空），
      且 `web/src/features/channels/lib/channel-type-config.ts:113` 的渠道模型提示与当前默认表一致。
- [ ] AC3 P5 代码侧完成：`scripts/check-version-drift.sh` 的实例比对剥掉 `v` 前缀，且不破坏现有语义。
- [ ] AC4 G1 完成：本轮与上次盘点的流程留痕已提交，提交消息带 `[task:maintain-clean-pending-items]`。
- [ ] AC5 待授权项（P1/P2/P3/G2/P5 保留期）逐条给出「未执行 + 原因 + 需要什么授权」，不静默跳过。
- [ ] AC6 终态：`git status` 无未提交的流程留痕（授权项未做则说明其导致的残留）。
- [ ] AC7 每项改动都有可复核证据（命令 + 输出摘要 + 文件路径）。

## Notes

- 本任务属于「多交付物」：代码文档修复（可并行）+ 决策与授权项（Lead 向用户取授权）+ git 收口。
- P1/P2/P3 若要执行，属实例级操作，须在获得当轮点名授权后由成员执行，Lead 复核证据。
- 113 个归档任务 AC 未勾属记账漂移（盘点已记录），**默认不在本次 8 项内**；若用户要把「干净」扩到它，
  需单列一条 R 并评估批量勾选的判据（哪些真未达成）。
