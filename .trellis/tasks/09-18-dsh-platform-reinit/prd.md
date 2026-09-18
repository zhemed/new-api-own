# 恢复 dsh 平台注册并提交 Trellis 初始化产物

## Goal

把这次 `trellis init`（22:33）留下的两处未收尾收干净：让 dsh 平台重新被 Trellis 登记为「已配置平台」，
并把本次初始化产生的文件提交入库；不改任何业务代码、不推送远端。

## Context（已核事实）

- HEAD 版 `.trellis/.template-hashes.json` 有 47 个非 `.trellis/` key（`.dsh/**` 4 个、`.agents/**` 43 个）。
- 工作区版只剩 39 个 key，**全部**在 `.trellis/**` 下；`.dsh/**` 与 `.agents/**` 各 0 个
  （另外多出 6 个 `__pycache__/*.pyc` key，属噪声）。
- `getConfiguredPlatforms(cwd)`（trellis 0.6.17 `dist/configurators/index.js:103`）靠 manifest 里是否存在
  该平台 `configDir` 下的 key 来反推「已配置」。dsh 的 `configDir` 是 `.dsh`，所以现在 dsh 等于未注册。
- 影响：后续 `trellis update` 不再刷新 `.dsh/skills/**` 与 `.agents/skills/trellis-*`；磁盘文件与本次会话
  功能不受影响。`.agents/skills` 里的项目自有技能（i18n-translate / shadcn-ui / vercel-react-best-practices）
  本来就不在 manifest 里，保持原状。

## Requirements

1. 用官方 reinit 快路径尝试恢复登记：`trellis init --dsh -u new-api-own`。
   - **实测该路径无法恢复**：`utils/file-writer.js` 对已存在且内容完全一致的文件不记录写盘事件，
     而 manifest 平台段只收"本次真的写过"的路径 → 平台 key 数为 0 不变（详见 Result）。
   - 该路径不触发模板/registry 网络请求；本任务全程不做网络动作。
2. 核验恢复结果：manifest 中 `.dsh/` 与 `.agents/` key 数量回到非零，且 `.trellis/**` key 数量不减少。
3. 核验零副作用：`git status` 只多出预期文件；`.dsh/**`、`.agents/**` 文件内容与 HEAD 一致（无内容改动）。
4. 提交初始化产物（仅 `.trellis/**`，不推送）：
   - `.trellis/.template-hashes.json`
   - `.trellis/tasks/00-join-new-api-own/`（init 生成的 join 引导任务）
   - 本任务目录
   - 提交消息带 `[task:dsh-platform-reinit]`。
5. 不动 `00-join-new-api-own` 的状态（是否 finish/archive 由用户另行决定）。

## Non-goals

- 不修改 trellis CLI 源码 / 不手工编辑 manifest 之外的 trellis 内部文件。
- 不清理 `.trellis/scripts/common/__pycache__/*.pyc`（gitignored 的构建产物，交给工具下次重建）。
- 不改业务代码、不跑 docker、不推送远端、不删任何平台文件。

## Acceptance Criteria

- [x] `trellis init --dsh -u new-api-own` 已执行（exit 0，无网络动作、未进入 Full re-initialize 分支），
      但**未能**恢复平台 key —— 已改走 manifest 数据修复，并把这个坑写进 spec。
- [x] 修复后 manifest：`.dsh/` key = 4、`.agents/` key = 43、`.trellis/` key = 33（合计 80）。
- [x] `git diff --stat` 无 `.dsh/**`、`.agents/**` 内容改动（修复前后平台文件 sha256 汇总一致：
      `925233d1b1885de298bf6eaf8c0a7e35f67acd2d850610fb8a680befa3d311e1`）。
- [x] `trellis platforms` 输出 `DeepSeek Harness (dsh) (dsh) — .dsh`；`getConfiguredPlatforms` = `['dsh']`。
- [ ] 提交完成，消息含 `[task:dsh-platform-reinit]`，且只含 `.trellis/**` 文件。
- [x] `./scripts/check-trellis-gate.sh` 通过（`3b658688..HEAD` 两个改动提交全部带锚点）。

## Result（实际执行记录）

1. **复现基线**：修复前 manifest 39 key，全在 `.trellis/**`，`.dsh` / `.agents` 各 0；平台文件 sha256 汇总
   `925233d1…`。
2. **官方路径尝试**：`trellis init --dsh -u new-api-own` → `📝 Configuring DeepSeek Harness (dsh)...`
   + `📋 Tracking 56 template files`，exit 0，但 `.dsh` / `.agents` key 仍为 0，新增的 17 条全在 `.trellis/**`
   （pyc 19 条、`.runtime/**` 4 条属噪声）。**根因**：`writeFile` 对内容一致的既有文件不记录写盘事件。
3. **等价性证明**：用 CLI 自己的 `collectDshTemplates()` 比对 —— 模板 47 个文件，磁盘 47 个全部逐字节一致
   （`diff=0`），且这 47 个的 sha256 与 HEAD manifest 里的历史值完全相等（`headMismatch=0`）。
   因此把 HEAD 的 47 条平台 key 并回 manifest，与一次强制重写所记录的哈希**等价**。
4. **修复**：并回 47 条平台 key（HEAD 原值），剔除 23 条 gitignored 运行态噪声
   （`__pycache__` 19 + `.runtime` 4），保留 `.trellis/**` 既有 key 并补录 HEAD 缺失的
   `.trellis/gates/enforce-from`。最终 diff vs HEAD = **+1 行**（就是补录的 enforce-from）。
5. **验收**：`trellis platforms` 列出 dsh；平台文件 sha256 汇总与基线一致；闸门自检通过。
6. **知识沉淀**：新增 `.trellis/spec/guides/trellis-manifest-guide.md` 并登记到 guides 索引。

## Rollback

`git checkout -- .trellis/.template-hashes.json` 可退回本次改动前的 manifest（工作区版本由 git 持有）。
join 任务目录、本任务目录与新增 spec 文档为新增文件，删除即可回退。
