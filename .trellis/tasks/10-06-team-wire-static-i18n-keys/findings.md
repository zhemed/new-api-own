# static-keys 接入 i18n 校验链路（2026-10-06）

## 结论

`STATIC_I18N_KEYS` 已**真正接入**校验链路；接线过程暴露并修复了 **5 个七语言全缺的真实键**；
登记表内 **21 个废弃项 + 6 个重复项**已清理。所有校验命令 exit 0。

## 1) 接入前"哪些键算被引用"= 无人定义

`web/scripts/sync-i18n.mjs` **只做 locale 之间对齐**（挑键最多的当 base、按 base 顺序重排、
缺失键用 en 值回填、统计与 en 相同的未翻译项），**不读源码、也不读 `static-keys.ts`**。
`web/scripts/` 原本只有 3 个脚本，没有任何源码扫描 → 登记表确实"白登记"。

## 2) 新增 `web/scripts/check-i18n-keys.mjs`（+ `bun run i18n:check`）

引用键 = **A ∪ B**：

- **A**：扫描 `src/**/*.{ts,tsx}` 的字面量 `t('…')`（含多行；跳过含 `${` 的模板串；
  先剥注释，避免把注释里的示例 `t('…')` 当成真实调用）。
- **B**：直接 `import` `STATIC_I18N_KEYS`。
- 逐个校验是否存在于 **7 个 locale**；缺失打印分组报告并 **exit 1**，齐备则 exit 0。

用 bun 运行是为了能直接 import `.ts`，不做脆弱的文本解析（项目本就以 bun 为准）。

**非空洞性证据（关键）**：接入后首次运行即 **exit 1**，报出 5 个键 × 7 语言全缺：

```
i18n check: 4012 literal t() keys + 476 static keys = 4184 referenced keys across 7 locales.
Missing translations (35 total):
  en (5): "{{count}} announcements deleted. Click \"Save Settings\" to apply."
          "{{count}} API entries deleted. Click \"Save Settings\" to apply."
          "{{count}} FAQs deleted. Click \"Save Settings\" to apply."
          "{{count}} groups deleted. Click \"Save Settings\" to apply."
          "{{count}} model(s)"
  (fr/ja/ru/vi/zh/zh-TW 同为这 5 个)
```

## 3) 这 5 个键是**真实用户可见缺陷**

`src/i18n/config.ts` 没有 `returnNull` / `parseMissingKeyHandler`，i18next 缺键时**返回键本身**，
即用户看到的是字面量 `{{count}} model(s)`（插值也不会发生）。调用点：

| 键 | 调用点 | 场景 |
| --- | --- | --- |
| `{{count}} model(s)` | `src/features/keys/components/api-keys-cells.tsx:199` | API Key 列表「模型数量」徽章 |
| `{{count}} announcements deleted.…` | `src/features/system-settings/content/announcements-section.tsx:242` | 批量删除公告的 toast |
| `{{count}} API entries deleted.…` | `src/features/system-settings/content/api-info-section.tsx:214` | 批量删除 API 信息的 toast |
| `{{count}} FAQs deleted.…` | `src/features/system-settings/content/faq-section.tsx:182` | 批量删除 FAQ 的 toast |
| `{{count}} groups deleted.…` | `src/features/system-settings/content/uptime-kuma-section.tsx:191` | 批量删除分组的 toast |

**补法（合规）**：临时 `scripts/add-missing-keys.mjs` 一次写全 7 语言 → `node scripts/sync-i18n.mjs`
→ 删除临时脚本。**未手改任何 locale JSON。** 译法对齐各语言已有的单数同族键
（如 zh `公告已删除。点击 "保存设置" 以应用。`、ru 沿用 `{{count}} models → моделей: {{count}}` 的
冒号式以避免数词一致问题）。`_sync-report.json` 复跑后七语言 missing/untranslated 全 0。

## 4) 登记表清理：删 27 行（21 废弃 + 6 重复）

**判据**：该键**整串文本**在 `web/` 全部 1024 个文件（排除 `node_modules/dist/locales/static-keys.ts`，
不限扩展名）中出现 **0 次** ⇒ 没有任何运行路径能请求到它。并用**特征片段**二次搜索以排除拼接构造。

- **废弃 21**：`Request Limits`(系统设置侧栏旧名)、`Model Access`/`Guardrails`/`Observability`/`Budgets`/
  `Token Mgmt`/`Prompt Caching`/`Pass-Through`（旧首页卡片）、`requests served`/`AI models supported`/
  `active users`（旧首页统计）、`Complete API documentation with multi-language SDK support`/
  `Technical Support`/`Professional team providing 24/7 technical support`（旧首页文案）、
  `Enter model name`、`Actual Amount`、`Available Models`、`View all currently available models`、
  `No available models`、`No models available in this category`、
  `Batch detection complete: {{channels}} channels, …`。
  **结构佐证**：`system-settings/*/section-registry.tsx` 的现行 `titleKey` 已无这批侧栏名；
  `home/components/sections/stats.tsx:101-104` 现行统计文案是 `upstream services integrated` 等，
  与登记的旧串完全不同 ⇒ 属改版遗留，非动态数据驱动。
- **重复 6**：`User`、`Disabled`、`Expired`、`Deleted`、`User updated successfully`（曾相邻重复两行）、
  `No Reset` —— 保留首次出现，删后者。登记表是扁平集合，重复纯属噪声。

登记表 476 → **455** 个唯一键；`check` 仍 exit 0（4163 个引用键七语言齐备）。

> 风险说明：删除登记项**不会**删除 locale 里的翻译，只影响校验覆盖面，可随时按需加回。

## 5) 接线的"自证"：knip ignore 已被迫移除

接线前 `static-keys.ts` 因无人引用被 knip 报为未使用文件（曾登记 ignore）。
现在 `check-i18n-keys.mjs` 直接 import 它、而该脚本经 `package.json` 的 `i18n:check` 成为 knip 入口
⇒ **文件已可达**，knip 提示 `Remove from ignore`。已按提示移除该 ignore 并在 `knip.config.ts`
写明"若 knip 又报它未使用，说明接线断了，应修接线而不是 ignore"。

## 6) 实测（全部 exit 0）

| 命令 | exit | 结果 |
| --- | --- | --- |
| `bun run i18n:check` | 0 | 4163 referenced keys，七语言齐备 |
| `bun run typecheck` | 0 | 无输出 |
| `bun run lint` | 0 | 21 warnings / **0 errors** |
| `bun test` | 0 | **154 pass / 0 fail** |
| `bun run build` | 0 | 产物正常 |
| `bun run knip --include files` | 0 | **无任何输出**（0 unused files、0 hints） |
| `bun run copyright:check` | 0 | 新脚本带 AGPL 头，0 added |

## 7) 遗留（不在本任务范围，交 Lead）

- `bun run format:check` **exit 1**，但失败文件是
  `src/components/layout/components/__tests__/footer.test.tsx` —— 该文件与 HEAD **逐字节相同**
  （`git diff` 为空），本任务的改动**未触碰**它，故属**既有**问题，非本轮引入。
- 本任务无对应共享任务（Lead 直接指派），Trellis 任务为 `10-06-team-wire-static-i18n-keys`。
