# 前端审查与修复（团队）— PRD

## Goal

对 `web/` 前端做一轮只读为主的审查（i18n 七语言一致性、类型安全、死代码、a11y/交互缺陷、已删除功能的残留引用），
并把**能证明安全**的低风险问题直接修掉（补 i18n 键、修明显字段名错误、补 aria-label、清理确凿的死代码引用）。

本任务属于团队协作任务：Lead 负责总体调度与最终验收，本审查员（review-frontend）负责前端维度。

## Requirements

### R1 i18n 一致性（最高优先）
- 写一个**只读** node 脚本扫描 `web/src` 中所有 `t('...')`（含 `t("...")`、`i18nKey`、`titleKey`、
  `t(\`...\`)` 模板字面量等静态可判定的键），与 `web/src/i18n/locales/{en,zh,zh-TW,fr,ja,ru,vi}.json` 求差集。
- 报告：缺失键（按语言）、locale 多余键（标注"待确认"，因为可能被静态字符串引用）。
- 面向用户的硬编码中文/英文未走 i18n。
- **所有 locale 改动必须走脚本**：按 `.agents/skills/i18n-translate/SKILL.md` 用临时 `add-missing-keys.mjs` + `node scripts/sync-i18n.mjs`。**禁止手改 locale JSON。**

### R2 类型安全
- 扫 `any` / `as any` / 不安全断言 / 可能 undefined 的解引用。
- 与后端 Go 结构体 json tag 对照，找前端字段名不匹配（例如把 `channel_id` 写成 `channelId`）。
- 只修**能证明安全**的（字段名错误、明显拼写），类型系统相关的大改动只报不改。

### R3 死代码 / 孤儿模块
- 保守判断：只有确凿无引用的模块才报；删除需二次确认（可能被动态 import / 路由表引用）。

### R4 交互 / 可访问性
- 纯图标按钮缺 `aria-label`；键盘不可操作的自定义交互。只报确凿的。

### R5 已删除功能的残留引用
- 重点核查 `web/src/features/setup/components/database-step.tsx:74-80` 的 electron 探测残留。
- 全仓搜索其它已删除功能的引用（electron、已下线渠道类型等）。

## Constraints（硬约束）

- **禁止网络动作与安装依赖**；禁止 `git commit` / `git push`。
- **只改 `web/**` 内文件**；不动 Go 代码、`docs/`、根 `*.md`、`.github/`、`scripts/`、`.local-instance/`。
- **禁止手改 locale JSON**，必须走脚本链路。
- 环境事实：本机无 `bun`，`web/node_modules` 不存在 → `bun run typecheck` / `bun test` / `bun run build`
  **全部无法运行**，且不得安装。可用的只有 `node`（v22）与仓库自带 node 脚本（如 `node web/scripts/sync-i18n.mjs`）。
- 类型相关改动**无从验证** → 只做能证明安全的改动，并在汇报里明确标"未验证（无工具链）"。

## Acceptance Criteria

- [ ] AC1：产出 i18n 差集扫描脚本与**实际执行输出**（七语言），输出可复核（键名 + 计数）。
- [ ] AC2：所有确认缺失的键已通过 `add-missing-keys.mjs` + `node scripts/sync-i18n.mjs` 补齐；
      补完后重新扫描，七语言差集为空（或明确列出仍未修项与原因）。
- [ ] AC3：locale 多余键清单产出，标注"待确认"，不做删除。
- [ ] AC4：类型安全 / 死代码 / a11y 各出一份发现清单，每条含**严重度 + 文件:行 + 依据 + 是否已修**。
- [ ] AC5：所有已修改文件均在 `web/**` 内，且 `git status --porcelain` 可复核。
- [ ] AC6：无任何网络动作、无依赖安装、无 locale JSON 手改（脚本生成可验证）。
- [ ] AC7：给出"未验证"清单（因无 bun/node_modules 无法验证的全部项目）。
- [ ] AC8：向 Lead 汇报（中文、结论先行、≤40 行），细节留在本任务目录。

## Notes

- 审查结论以文件为准确来源；不确定的一律标"待确认"，不猜。
- 低风险项直接修；有疑问的一律只报不改（宁可漏修，不可误改）。
