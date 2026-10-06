# 团队评估并处理剩余开放项

## 开放项来源

- 上一轮团队审查 + 本轮复核留下的：3 个既有失败测试、knip 剩余项与 ignore 掩盖、后端两处"刻意未改"的不一致。
- 用户指示：「你调用团队评估并处理吧」——要**结论 + 落地**，不是再写一份报告。

## 分工（写范围互不重叠）

| 成员 | 共享任务 | 写范围 | 目标 |
|---|---|---|---|
| `fix-frontend-open-items` | task-7 | `web/**` | 处理 3 个失败测试的取向、knip 剩余 8 项、被 ignore 掩盖的 14 个 `ui/**`，收尾后 typecheck/lint/test/build/knip 全绿 |
| `fix-backend-open-items` | task-8 | Go（relay/ dto/ model/ service/ common/）| 评估并处理 multipart 缺 `model` 校验、DTO 非指针标量；能安全改就改+补测试，波及面过大则留痕 |

Lead：复核 diff、全套门禁、提交推送、汇总与遗留。

## 硬约束（全员，沿用前几轮教训）

- 禁止网络动作；禁止访问生产；禁止 git commit/push/tag；**禁止 `git add -A`、禁止在仓库内建 git worktree**；
- 不要动 `.local-instance/`、不要重启任何服务；
- 不许改测试来"消除失败"——除非结论是"测试断言的是已废弃设计"，且必须在汇报里逐条说明依据；
- 禁用 `--no-verify`；发现不确定就标"待确认"，不许猜。

## Acceptance Criteria

- [ ] 3 个失败测试有明确取向（恢复功能或对齐测试）且最终 `bun test` 全绿或逐条说明为何仍失败
- [ ] knip 收尾：剩余项有结论（删除/ignore/留待）、`bun run knip` 的 exit 状态与文档描述一致
- [ ] 后端两处各自给出"已改+测试"或"不改+证据"，并附命令实测
- [ ] Lead 复核、提交、CI 绿、记录归档

## 执行结果（2026-10-06，两条线并行）

### 前端线（task-7 → 完成后追加 static-keys 接线）

| 开放项 | 结论 |
|---|---|
| 3 个 api-key 表格测试失败 | **恢复被注释的功能**（不是改测试）：同款动效在姊妹组件活跃使用且测试通过、动效 CSS（含 `prefers-reduced-motion`）完整保留、`AutoGroupBadge` 唯一引用就是那行注释 → 取消注释 2 行，**未放宽任何断言**；`bun test` 151 pass/3 fail → **154 pass/0 fail** |
| knip 8 项 | 删 7 个真死码（barrel 的源模块都用相对路径直连，删除无级联）+ 1 项（static-keys）写明理由 → **knip 无任何输出** |
| 14 个 `ui/**` 被 ignore 掩盖 | **全部保留**：`components.json` 的 `aliases.ui` 是 shadcn CLI 约定；决定性依据是 `ui/chart.tsx` 是 `recharts` 唯一消费者、`ui/resizable.tsx` 是 `react-resizable-panels` 唯一消费者（删了就复现"依赖误报"坑）|
| static-keys 白登记（追加）| 新建 `web/scripts/check-i18n-keys.mjs` + `i18n:check`，引用键 = 源码 `t('…')` ∪ `STATIC_I18N_KEYS`；**一接线就抓出 5 个七语言全缺的用户可见键**（i18next 缺键返回键本身 → 用户看到字面量 `{{count}} model(s)`），按 i18n 规范补齐；登记表清掉 27 行（21 废弃 + 6 重复），476 → 455 |

### 后端线（task-8 → task-9 → task-10）

| 开放项 | 结论 |
|---|---|
| multipart 图像编辑缺 `model` 校验 | **已补**（与 JSON 分支同文案）：常规路径根本走不到（distributor 已 400），唯一穿透口是 admin 指定渠道密钥；补边界测试 |
| 客户端 DTO 非指针标量 | **不改**：`Duration` 29 处读写点但**全仓库从不 marshal**（omitempty 是死规则）；`dto/video.go` 零 Go 引用、只对应已发布 openapi schema |
| **新发现：sora 时长上界绕过** | **已修**（入口 + 适配器双层）：`{duration:4,seconds:"100000"}` 改前 ratio=100000 → 配额饱和后误报额度不足；改后入口 400，适配器再钳到 3600。同形排查无第二处 |
| handler 校验错误 500 → 400 | **已改**（证据：openapi 契约、同函数既有 400、relaykit 无固定 5xx 逻辑）；新增 controller 测试 |
| 敏感词命中 500 → 400 | **已改**（证据：同类先例 gemini prompt_blocked 用 400、middleware 的 403 全是身份类、发布契约无 403、5xx 会被重试）|
| `count_token_failed` / `get_channel_failed` 5xx | **刻意不改**（混装客户端与服务端成因 / 容量类故障重试是期望行为）→ 记入 MAINTENANCE「已知不一致」|

### Lead 终验（冻结后）

Go：`gofmt` 干净、vet/build/relaykit exit=0、`make test` exit=0（**39 包 ok**）；
前端：typecheck 0、lint 0 error、`bun test` **154 pass/0 fail**、build 0、knip 0、`i18n:check` 0、`format:check` 0（顺手修掉我上一轮引入的格式问题）；
门禁三件套通过；CI 通过。提交分两次（`759261b` 前端 / `7e017cd` 后端+留痕）便于复审。
