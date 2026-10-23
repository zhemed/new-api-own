# P4 / P5 独立验证证据（verifier）

- 任务：共享任务板 `task-6`（独立验证 P4 文档/前端与 P5 脚本改动）
- 验证者：verifier（只读验证，未改任何仓库文件）
- 验证时间：2026-10-23（UTC 16:05–16:20）
- 基线：`HEAD = e6ec2d8`，工作区未提交改动
- 被验证文件的 blob 哈希（验证基线，若之后变化即说明被再次改动）：
  - `MAINTENANCE.md` → `5e230e8c6709a9baa8bea9923463899dbba7d736`
  - `scripts/check-version-drift.sh` → `b9aa26d31317d396aa15927ab0207c186d90cc57`
  - `web/src/features/channels/lib/channel-type-config.ts` → `4dab0d4297b17f092f6f5a53843260203b5177fd`

## 结论

**接受（Accept）**。三条改动经独立复现与代码核对，事实与行为均成立；未发现夹带改动；
`bash -n` / shellcheck / 前端 typecheck / Go 构建 / 相关单测 / i18n 检查全部通过。
有 2 条非阻断观察（见「风险与未完成」第 1、2 条），不构成打回理由。

## 一、改动最小性核对（有无夹带）

`git status --porcelain` 共 5 个已跟踪改动文件 + 2 个未跟踪目录：

| 文件 | 归属 | 是否在验证范围 | 判定 |
|---|---|---|---|
| `MAINTENANCE.md` | P4 | 是 | 仅 A3 条目 1 行改 5 行，无其他改动 |
| `web/src/features/channels/lib/channel-type-config.ts` | P4 | 是 | 仅 :113 一行，`hints.models` 值替换 |
| `scripts/check-version-drift.sh` | P5 | 是 | 注释 + `normalize_version()` + 3 处调用点，无其他逻辑变动 |
| `.trellis/workspace/new-api-own/index.md` | Lead 记账 | 否 | Trellis 会话记账（总会话数 / 文档行数 / 会话历史表） |
| `.trellis/workspace/new-api-own/journal-2.md` | Lead 记账 | 否 | 新增 Session 94 journal 段 |
| `.trellis/tasks/10-23-maintain-clean-pending-items/` | Lead/成员留痕 | 否 | 未跟踪任务目录（含实现者 evidence-P*.md） |
| `.trellis/tasks/archive/2026-10/10-23-workspace-task-audit/` | 上次盘点归档 | 否 | 未跟踪归档目录 |

三个目标文件的 diff 逐行读过：无调试残留、无顺手重构、无行尾/格式噪音，改动量最小。

## 二、P4-A：MAINTENANCE.md A3 分路径陈述的事实核对

代码事实（自行 read/grep，非照抄实现者）：

| 路径 | 代码事实 | 日志结果 | 文档新写法 | 判定 |
|---|---|---|---|---|
| OpenAI 兼容 chat | `ApplyDeepSeekV4OpenAIThinkingSuffix` 在 `thinkingType == "disabled"` 时写 `info.ReasoningEffort = "none"`（`relay/common/deepseek_v4_thinking.go:74-78`） | 写 `"none"` | 记 `"none"` | 一致 |
| Responses | `ApplyDeepSeekV4ResponsesThinkingSuffix` 在 disabled 时把 effort 兜底为 `"none"`（`:93-95`），并在 `:113-115` 写入 `info.ReasoningEffort` | 写 `"none"` | 记 `"none"` | 一致 |
| Claude | `ApplyDeepSeekV4ClaudeThinkingSuffix`（`:41`）与 `relay/claude_handler.go:110-134` 对 `-none` 只写 `thinking.type=disabled`、清空 `output_config`，`info.ReasoningEffort` 保持空串 | 不写该键 | 字段缺失 | 一致 |

补充核对（文档未写但影响结论的链路，均已读代码确认）：

1. 后缀解析：`relaykit/relayconvert/reasoning/suffix.go:45-57`，`-none` → `thinkingType="disabled"`、`effort=""`、`ok=true`。
2. Claude 路径不会被「默认 high」补回：`relay/claude_handler.go:131-134` 的守卫要求 `request.Thinking == nil || Type != "disabled"`，`-none` 下为 disabled，跳过。
3. Claude 路径不会被元数据同步补回：`syncClaudeRequestMetadata`（`relay/claude_handler.go:257-264`）取 `request.GetEfforts()`，而 `GetEfforts`（`relaykit/dto/claude.go:414-421`）只读 `output_config.effort`；`-none` 时 `OutputConfig=nil` → 返回空串，不写入。
4. 日志落盘：`service/log_info_generate.go:83-84` 仅在 `relayInfo.ReasoningEffort != ""` 时写 `other["reasoning_effort"]` → 空值即字段缺失。
5. chat 路径不会被适配器覆盖：`relay/channel/openai/adaptor.go:366-368` 的覆盖有 `request.ReasoningEffort != ""` 守卫，`-none` 时 `request.ReasoningEffort` 为空，故 `info` 保持 `"none"`。
6. 行号引用逐条核对：`:74-78`、`:93-95`、`:113-115`、`:41`、`relay/claude_handler.go:110-134`、`service/log_info_generate.go:83-84` 全部命中所述代码。
7. 历史论断核对：`git show fd5d5c8 -- relay/common/deepseek_v4_thinking.go relay/claude_handler.go` 显示该提交**只**改了 chat 分支（新增 disabled → `"none"`），Claude 分支未动；提交信息却写「record -none uniformly as 'none' on Claude/chat paths」→ 文档「提交信息写的是 Claude/chat，实际未含 Claude」成立。
8. 现状固化引用核对：`relay/common/deepseek_v4_thinking_test.go:161-166` 确为 Claude 用例表，其中 `:165` 断言 `-none` 时 `wantInfoEffort: ""`；chat 用例 `:114` 断言 `infoEffort: "none"`、Responses 用例 `:217` 断言 `wantInfoEffort: "none"` —— 三条路径的测试断言与文档新写法完全一致。
9. 无残留旧说法：全仓库 grep `统一记录 none` 无其他命中；`MAINTENANCE.md` 中 A3 只有这一处。

## 三、P4-B：前端 `hints.models` 核对

| 核对项 | 证据 | 判定 |
|---|---|---|
| 渠道身份 | `channel-type-config.ts:102-115`，key `43` = DeepSeek，`defaultBaseUrl=https://api.deepseek.com` | 对 |
| 与默认模型表一致 | `setting/ratio_setting/model_ratio.go:37-38` 默认模板仅 `deepseek-flash`、`deepseek-v4.1-flash`（注释 :34-36 声明与渠道 ModelList 同源） | 一致 |
| 与渠道 ModelList 一致 | `relay/channel/deepseek/constants.go:4-5` 含 `deepseek-flash`、`deepseek-v4.1-flash` 及其 `-none/-max` 变体 | 一致 |
| 旧值确已过时 | `setting/ratio_setting/model_defaults_test.go:100-102` 把 `deepseek-coder`、`deepseek-chat` 列入「不应再有默认倍率」清单 | 改得对 |
| i18n 缺口 | 全 `web/src` grep `hints`：除定义文件外无任何消费点；`getChannelTypeConfig` 仅被单测 `new-api-channel.test.ts` 使用（断言 icon）→ 该字符串不进 `t()`，因此**不引入也不需要**新 locale 键 | 无缺口 |
| 门禁 | `cd web && bun run typecheck`（tsgo -b --force）退出码 0；`bun run i18n:check` 报告 4186 个引用键在 7 个 locale 全部存在，退出码 0；`bunx oxlint` 该文件 0 warning / 0 error | 通过 |

## 四、P5：`scripts/check-version-drift.sh` 行为核对

### 4.1 静态检查

- `bash -n scripts/check-version-drift.sh` → 通过（SYNTAX_OK）。
- `shellcheck -S warning scripts/check-version-drift.sh` → 无输出、退出码 0。

### 4.2 自造 fixture 行为矩阵（全部在 /tmp/vdrift，禁止联网：docker/curl 均为 stub）

harness：`/tmp/vdrift/run.sh`（34 项断言，结果 `34 passed, 0 failed`）。fixture 仓库 = 临时 git 仓库（自带 VERSION 文件与 tag）+ 脚本副本（新/旧两版各一份）。

| 用例 | 场景 | 期望 | 实际 |
|---|---|---|---|
| A0 | **旧脚本** + 实例返回 `v0.0.45`、VERSION `0.0.45`、tag `v0.0.45` | rc=1（缺陷复现） | rc=1，输出 `❌ 实例` |
| A1 | 新脚本同场景 | rc=0，无 `❌` | rc=0，无 `❌` |
| A2 | 新脚本 + 实例返回 `0.0.45` | rc=0 | rc=0 |
| A3 | 新脚本 + `--version v0.0.45` 对 tag `v0.0.45` | rc=0 | rc=0 |
| B1 | VERSION `0.0.46` vs tag `v0.0.45` | rc=1，tag 漂移 | rc=1，`!= 最新 tag v0.0.45` |
| B2 | VERSION `0.0.46` vs 实例 `v0.0.45` | rc=1（归一化不得掩盖真漂移） | rc=1，`❌ 实例` |
| B3 | VERSION `0.0.45` vs 实例 `v0.0.44` | rc=1 | rc=1 |
| B4 | tag `va0.0.45` vs VERSION `0.0.45` | rc=1（只剥一个前导 v） | rc=1 |
| C1 | `--no-network` 且仓库内部一致 | rc=2（UNKNOWN） | rc=2，`判定: UNKNOWN` |
| C2 | PATH 沙箱内**无 docker** | rc=2 | rc=2，`未安装 docker` |
| C3 | VERSION 文件内容为单个 `v` → 归一化为空 | rc=2 | rc=2，`VERSION 文件为空` |
| C4 | 无 tag | rc=2 | rc=2 |
| E1-E3 | registry 查询标签集合 | 仍是 `latest` / `0.0.45` / `v0.0.45` | 三条均命中 stub 调用日志 |
| E4 | 不得出现 `vv0.0.45` | 无 | 无 |

结论：v 前缀假漂移修复、真漂移仍报、退出码 0/1/2 语义未变，四条路径（tag、实例、registry 标签构造、`--version` 覆盖）行为均可解释且与脚本头部注释一致。

### 4.3 `normalize_version` 边界（直接从新脚本正文抽取函数体后逐值调用）

| 输入 | 输出 | 判定 |
|---|---|---|
| 空串 | 空串 | 符合预期 |
| `v` | 空串 | 符合预期（后续空值检查 → rc=2） |
| `0.0.45` | `0.0.45` | 不变 |
| `v0.0.45` | `0.0.45` | 剥一个 |
| `1.0.0-rc1` | `1.0.0-rc1` | 不变 |
| `v1.0.0-rc1` | `1.0.0-rc1` | 剥一个 |
| `vv0.0.45` | `v0.0.45` | 只剥一个（与注释一致） |
| `V0.0.45` | `V0.0.45` | 大写 V 不剥（大小写敏感） |
| `latest` | `latest` | 非版本串原样 |

## 五、门禁复跑

| 门禁 | 命令 | 结果 |
|---|---|---|
| 前端类型 | `cd web && bun run typecheck` | 退出码 0，无诊断输出 |
| 前端 i18n | `cd web && bun run i18n:check` | 退出码 0 |
| 前端 lint（改动的单文件） | `bunx oxlint -c .oxlintrc.json src/features/channels/lib/channel-type-config.ts` | 0 warning / 0 error |
| Go 构建 | `go build ./...` | 退出码 0（本轮 Go 源码未改，构建通过即排除误伤） |
| Go 单测（三条路径锚点） | `go test ./relay/common/ -run 'TestApplyDeepSeekV4' -count=1` | ok |
| Go 单测（默认模型目录） | `go test ./setting/ratio_setting/ -run 'TestDefaultCatalog' -count=1`（另跑 `TestDeepSeek` 前缀组合同样 ok） | ok |

运行门禁未产生新的仓库改动：复跑后 `git status --porcelain` 与验证前完全一致（无新增未跟踪产物、无 tsbuildinfo 落盘）。

## 六、风险与未完成

1. （非阻断，P5）`normalize_version` 是「剥字符串首字母 v」，对非版本串同样生效（例如 tag 若叫 `vnext` → 归一化为 `next`）。当前三处输入分别来自 VERSION 文件、git tag、实例 `/api/status`，均为版本串，实际风险低；如要更严可加 `^v[0-9]` 判定，但会改变 `v` 单独输入时的现有分支（现为 rc=2），需一并评估。
2. （非阻断，P4）C3 用例暴露一处文案不精确：VERSION 文件内容为 `v` 时报「VERSION 文件为空」，实际是「归一化后为空」。仅提示语措辞，不影响退出码语义。
3. （非阻断，P4）`hints` 字段当前在 `web/src` 内无任何消费点（`getChannelTypeConfig` 仅被单测引用），即本次前端改动属「元数据订正」，运行时不产生 UI 变化；这与「无 i18n 缺口」是同一事实的两面，供 Lead 判断是否需要另立任务处理该死字段。
4. （未完成项，非本次范围）PRD 的 AC4/AC5/AC6 涉及提交、推送与待授权项，不在本次只读验证范围，需 Lead 收口。
5. （验证基线时效）本次结论绑定上文三个 blob 哈希；若之后任一对被验证文件再改，需重跑第 4、5 节。

### 越界事故披露（如实报告，不辩解）

验证过程中我自己的 harness 出现一次**非预期网络访问**：前两轮 fixture 运行把 `/usr/bin` 拼进了 PATH（`PATH=<stub 目录>:/usr/bin:/bin`），导致「无 docker」用例里 `command -v docker` 穿透到系统 `/usr/bin/docker`，脚本据此真实调用了
`docker buildx imagetools inspect ghcr.io/zhemed/new-api-own:{latest,0.0.45,v0.0.45}`。

- 次数与时间：2 轮 × 3 次 = 6 次，2026-10-23T16:08:05Z 与 16:09:11Z。
- 性质：公开镜像仓库的**只读标签元数据查询**（等价 GET），无鉴权、无凭据、无推送、无写操作、无任何状态变更，未接触内网或未点名主机；未把任何现场信息写入持久位置（仅本文件的这一句披露）。
- 处置：发现后立即停止该用例，改用 `PATH=/tmp/vdrift/bin-min`（仅符号链接必要 coreutils、**不含 docker**）并给 harness 加了硬闸门（`command -v docker`/`curl` 只要解析到 `/tmp/vdrift/*` 之外的二进制就 `exit 99` 拒绝运行）；此后全部用例在闸门下重跑，34 项全绿，未再发生真实调用。
- 影响：不影响本次验证结论（该分支是改动前既有逻辑，与 normalize_version 无关），但按红线必须上报，由 Lead 决定是否需要向用户说明。

## 七、复核命令清单（可复跑）

```bash
git --no-pager diff -- MAINTENANCE.md scripts/check-version-drift.sh web/src/features/channels/lib/channel-type-config.ts
bash -n scripts/check-version-drift.sh && shellcheck -S warning scripts/check-version-drift.sh
bash /tmp/vdrift/run.sh                 # 自造 fixture 矩阵（34 项，需保留 /tmp/vdrift）
cd web && bun run typecheck && bun run i18n:check
go build ./... && go test ./relay/common/ -run 'TestApplyDeepSeekV4' -count=1
go test ./setting/ratio_setting/ -run 'TestDefaultCatalog|TestDeepSeek' -count=1
```
