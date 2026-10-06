# 把 static-keys 接进 i18n 校验链路

## Goal

`web/src/i18n/static-keys.ts` 的 `STATIC_I18N_KEYS` 目前是**无人消费的登记表**（没有任何脚本读它），
等于白登记。本任务把它真正接入 i18n 校验链路，并清理登记表本身的废弃/重复项。

## 现状（已查证）

- `web/package.json` 只有 `i18n:sync → node scripts/sync-i18n.mjs`；`web/scripts/` 仅 3 个脚本，
  **没有任何脚本扫描源码里的 `t()` 键**。
- `sync-i18n.mjs` 只做"locale 之间互相对齐"：挑键最多的 locale 当 base、按 base 顺序重排、
  缺失键用 en 值回填、统计"与 en 相同"的未翻译项。**它不读源码，也不读 static-keys.ts**。
- `_reports/_sync-report.json` 显示七语言当前 missing/untranslated 全为 0。

## Requirements

### R1 纳入扫描源

新增 `web/scripts/check-i18n-keys.mjs`，定义"哪些键算被引用"：

- **来源 A**：扫描 `src/**/*.{ts,tsx}` 里的 `t('...')` / `i18next.t('...')` **字面量**键
  （跳过含 `${` 的模板串等非字面量，避免误报）。
- **来源 B**：从 `src/i18n/static-keys.ts` 读取 `STATIC_I18N_KEYS`（登记表本身）。
- **引用集合 = A ∪ B**，逐个校验是否存在于全部 7 个 locale（en/zh/zh-TW/fr/ja/ru/vi）。
- 缺失时打印分组报告并 **exit 1**；全部齐备时输出通过信息并 exit 0。
- 在 `package.json` 注册 `i18n:check`，使其成为可重复执行的校验入口。

### R2 跑校验并补缺

- 校验暴露出的缺失键 → 严格按 `.agents/skills/i18n-translate/SKILL.md`：
  **禁止手改 `locales/*.json`**，必须用临时 `scripts/add-missing-keys.mjs`（七语言一次写全）
  + `node scripts/sync-i18n.mjs`，临时脚本用完删除。

### R3 清理登记表

- **确认废弃**的登记键（判据：整串文本在 `web/` 全部源码中出现 0 次）→ 从 `static-keys.ts` 删除，
  汇报里逐条列出。
- 登记表内的**重复项** → 去重。
- 注意：删除登记项**不会**删除 locale 里的翻译，只影响校验覆盖面，风险可控。
- 若出现"登记键疑似由后端/动态数据驱动"而无法确证，**保守保留**并标注待确认。

### R4 回归

复跑 `bun run typecheck`、`bun test`、`bun run knip`（含退出码）；
不得引入新的 lint error / typecheck 失败。

## Constraints

- 禁止网络动作；禁止 `git commit/push/tag`；禁止 `git add -A`；禁止建 worktree。
- 只改 `web/**`；不动 `.local-instance/`；不重启服务。
- **不得为了跑通放宽校验**：若缺失键数量巨大（数十个量级），停下来说明现状，不批量灌水翻译。
- 翻译质量遵循 i18n skill：长度/布局意识，品牌与技术标识保留英文。

## Acceptance Criteria

- [ ] `web/scripts/check-i18n-keys.mjs` 存在，且 `STATIC_I18N_KEYS` 是其引用键来源之一。
- [ ] `bun run i18n:check` 可执行，键齐备时 exit 0。
- [ ] 校验暴露的真实缺失键已按规范补齐（七语言）。
- [ ] `static-keys.ts` 的废弃项与重复项已清理，逐条有依据。
- [ ] `bun run typecheck` / `bun test` / `bun run knip` 复跑结果（含退出码）已记录。
