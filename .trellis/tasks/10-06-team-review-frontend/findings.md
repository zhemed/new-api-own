# 前端审查发现清单（review-frontend / task-2）

范围：`web/**`。工作目录 `/root/new-api-own`。基线 HEAD `0941a4f`。
只读扫描脚本：`/tmp/i18n-audit.mjs`（仓库外，只读，不落库）。

---

## A. i18n

### A1 七个键在七语言全部缺失（严重度：中 · 已修）
`t()` 引用的键在 `en/zh/zh-TW/fr/ja/ru/vi` 全部不存在 → i18next 直接回显键原文。

| 键 | 调用点 |
|---|---|
| `Added {{count}} models from "{{name}}"` | `features/channels/components/drawers/channel-mutate-drawer.tsx:1530` |
| `Are you sure you want to delete "{{name}}"? Users who authenticated with this provider will no longer be able to log in.` | `features/system-settings/auth/custom-oauth/components/provider-table.tsx:142` |
| `Auto Detect` | `features/system-settings/auth/custom-oauth/types.ts:221`（经 `provider-form-dialog.tsx:387,404` 的 `t(option.labelKey)` 渲染） |
| `Params (in body)` | 同上 `types.ts:222` |
| `Header (Basic Auth)` | 同上 `types.ts:223` |
| `Configure a custom ratio for "{{userGroup}}" users when using a specific token group."` | `features/system-settings/models/group-ratio-visual-editor.tsx:1064` |
| `Delete model "{{name}}"? This cannot be undone.` | `features/channels/components/dialogs/ollama-models-dialog.tsx:583` |

### A2 硬编码英文无障碍名称（严重度：中 · 已修）
见 B2/B3。这些字符串对中文用户会以英文被屏幕阅读器朗读，既违反 a11y 也违反 i18n 规范。

### A3 locale 多余键
**0**。七语言键集与 en 完全一致（`extra=0`、`missing=0`），无需"待确认"清单。

### A4 面向用户的硬编码中文
**无**。`web/src` 内 132 行含 CJK，逐行去注释后仅剩 28 行，全部属以下合法类别：
- 注释 / JSDoc（如 `top-nav.tsx:39-40`、`lib/utils.ts:31-34`）
- 测试 fixture（`__tests__/*.test.tsx` 中的 `ratio='自动'`）
- 语言自名（`i18n/languages.ts` 的 `简体中文`/`日本語`，应当如此）
- 与后端常量/默认配置对齐的中文值（`payment-methods-visual-editor.tsx:71,79,105` 的 `支付宝`/`微信`/`自定义1`、`models/constants.ts:35,41` 的 `官方倍率预设`/`models.dev 价格预设`）→ 保留，若改动会破坏与后端默认值匹配
- 嵌在英文键内的关键词（`wechat-bind-dialog.tsx:79` 等 `reply with "验证码"`）→ 该词是用户必须发送的字面量，保留

技术占位符（`gpt`、`400`、`bash -lc`、`{"KEY":"VALUE"}`、`https://provider.com/...`、`0.01`、`price_...`、`session_id`）按规范**正确保留英文，不应翻译**。

---

## B. 可访问性（严重度：中 · 已修 17 处）

### B1 纯图标按钮完全无无障碍名称（确凿缺陷）
| 位置 | 控件 | 修复 |
|---|---|---|
| `features/profile/components/checkin-calendar-card.tsx:380,388` | 上一月 / 下一月 | 新增键 `Previous month` / `Next month` |
| `features/wallet/components/subscription-plans-card.tsx:368` | 刷新 | 复用既有键 `Refresh` |
| `features/channels/components/dialogs/param-override-editor-dialog.tsx:1946` | 新增规则（Plus） | 复用既有键 `Add Rule` |

### B2 硬编码英文 aria-label（已改为 t()，新增 6 键）
| 位置 | 原文 |
|---|---|
| `components/config-drawer.tsx:152` | `'Reset'`（复用既有键；`SectionTitle` 原先无 i18n，已补 `useTranslation`） |
| `components/datetime-picker.tsx:158` | `'Clear'`（复用既有键） |
| `components/json-editor.tsx:259` | `'Delete row'`（新键） |
| `components/tag-input.tsx:93` | `'Remove tag'`（新键） |
| `components/password-input.tsx:57` | `'Toggle password visibility'`（新键；文件原先无 i18n，已补 hook） |
| `features/channels/components/numeric-spinner-input.tsx:154,197` | `'Decrement'` / `'Increment'`（新键；同上补 hook） |
| `features/models/components/dialogs/update-config-dialog.tsx:341` | placeholder `'Optional'`（新键） |

### B3 表格选择复选框 aria-label 硬编码英文（键已存在，仅需 t() 包裹）
`api-keys-columns.tsx:91,99`、`models-columns.tsx:78,85`、`upstream-conflict-dialog.tsx:207,290,297`、
`users-columns.tsx:55,63`、`channel-selector-dialog.tsx:131,138`、`tiered-pricing-editor.tsx:500`（`'remove'`→`t('Remove')`）。

### B4 未改（判定非缺陷 / 待确认）
- `features/dashboard/components/overview/uptime-panel.tsx:114`：`aria-label` 挂在子 `<RotateCcw>` 上而非 `Button`。
  按 AccName 计算，后代节点的 `aria-label` 会贡献给按钮的 name-from-content，故**按钮有名称**；且同仓同族面板
  （`system-tasks-panel.tsx:359`、`system-instances-panel.tsx:686`）把 aria-label 放在 `Button` 上。
  结论：**一致性建议，非缺陷，未改**。
- `features/pricing/components/model-card.tsx:228` 原生 `<button>` 只有 Copy 图标，但有 `title={t('Copy')}` → 有可访问名称，合格。
- `components/ui/*`、`components/ai-elements/*` 内 7 处英文 aria-label 属 vendored 组件默认值 → 未改（只报）。

---

## C. 死代码

### C1 electron 死分支（**已删**）
`web/src/features/setup/components/database-step.tsx:74-81`（探测）+ `:121-132`（两个不可达 JSX 分支）。

删除依据（逐条可复核）：
1. `electron/` 整目录与 `.github/workflows/electron-build.yml` 已于 2026-10-06 随任务
   `10-06-strip-non-deploy-extras` 移除；`docs/FILE_INVENTORY.md:26` 明文记录"已于 2026-10-06 整体移除"。
2. 全 `web/` 对 `window.electron` **只有读取、无任何写入方**（grep 生产者 = 0）→ 被注入方已不存在。
3. `web/src/env.d.ts` 无 `window.electron` 声明，代码只能用 `as unknown as Record<string, unknown>` 绕过类型
   → 反证它是被外部壳注入的全局量，而非本仓类型。
4. `AGENTS.md`「唯一部署方式（强制，违反即事故）」规定唯一交付为 `docker run` + 公开镜像，
   该运行时不具备注入该全局的能力。

因此 `isElectron` 恒为 `false`，两个分支不可达 → 删除是**行为等价**的（不改变任何可达渲染结果）。

副作用：`Data directory:` 与 `Data is stored locally on this device. Use system backups to keep a safe copy.`
两个键变为无引用。**保留不删**（locale 允许无引用键；删键需再跑脚本，收益为零且增加风险）。

### C2 孤儿模块候选 62 个（**只报不改**）
判定方式：文件 basename 不出现在任何 `import/from/require/export ... from` 说明符中（已排除 `__tests__`/`*.test.*`/路由文件/index）。
主要簇：
- `components/ai-elements/` 23 个（actions/artifact/branch/canvas/chain-of-thought/confirmation/connection/context/edge/image/inline-citation/node/open-in-chat/panel/plan/queue/suggestion/task/tool/web-preview …）
- `components/ui/` 9 个（aspect-ratio、breadcrumb、button-group、chart、direction、item、kbd、native-select、resizable）
- `features/home/components/` 5 个（connection-line、feature-item、gateway-card、hero-buttons、scrolling-icons）
- `features/system-settings/hooks/` 3 个、`features/system-settings/utils/route-config.ts`
- 其它单点：`components/auto-skeleton.tsx`、`components/coming-soon.tsx`、`components/date-picker.tsx`、
  `components/learn-more.tsx`、`components/theme-quick-switcher.tsx`、`lib/show-submitted-data.tsx`、
  `hooks/use-minimum-loading-time.ts`、`hooks/use-table-compact-mode.ts`、`i18n/static-keys.ts` 等

保守结论：**不动**。`static-keys.ts` 实为扫描用的键登记表（非运行时依赖）；`components/ui/*` 与
`ai-elements/*` 属引入的组件库全集，删除会破坏后续复用与上游同步。需要真实瘦身应先接 `bun run knip`（当前无工具链）。

---

## D. 类型安全（只报不改：无工具链无法验证）

| 项 | 数量 | 说明 |
|---|---|---|
| 非生成代码 `as any` | **5** | `channels-columns.tsx:1175`（`row={row as any}`）、`settings-page.tsx:116`（`useParams({from: routePath as any})`）、`sidebar-modules-section.tsx:212,242,257`（`name=... as any`） |
| `routeTree.gen.ts` 的 `as any` | 58 | 生成文件，`@ts-nocheck`，不算问题 |
| `@ts-ignore` / `@ts-expect-error` | **0** | 仅生成文件有 `@ts-nocheck` |
| 非空断言 `!.` | **0** | 表现很好 |
| `as unknown as` | 14 | 多在测试与边界（`chat-links.ts`、`ollama-utils.ts`、`channel-utils.ts:647-656`、`edit-tag-dialog.tsx:194`、`subscriptions-mutate-drawer.tsx:110`） |
| `dangerouslySetInnerHTML` | 4 | `ui/chart.tsx:111`、`ui/markdown.tsx:786`、`layout/components/footer.tsx:243`（经 sanitize）、`html-content.tsx:204` |

契约抽查：`status.database_type`（`features/setup/types.ts:24`）与后端 `controller/setup.go:16` 的
`json:"database_type"` 一致 ✓。`web/src/features/setup/components/database-step.tsx` 内 `sqlite/mysql/postgres`
分支与 `common.MainDatabaseType()` 取值域一致 ✓。

---

## E. i18n 一致性校验实际输出（最终态）

```
scanned 1034 files; distinct keys referenced in code: 4245
en.json key count: 5268; STATIC_I18N_KEYS: 476

===== MISSING KEYS (0) =====

===== STATIC_I18N_KEYS absent from en.json (0) =====

===== LOCALE KEY-SET DRIFT vs en =====
en: keys=5268 extra=0 missing=0
zh: keys=5268 extra=0 missing=0
zh-TW: keys=5268 extra=0 missing=0
fr: keys=5268 extra=0 missing=0
ja: keys=5268 extra=0 missing=0
ru: keys=5268 extra=0 missing=0
vi: keys=5268 extra=0 missing=0
```
（en key 数 5253 → 5268 = 新增 **15** 键：A1 缺失键 7 + B1 月份导航 2 + B2 无障碍名称 6。）

locale 写入路径：`web/scripts/add-missing-keys.mjs`（自建、**已删除**）→ `node scripts/sync-i18n.mjs`。
两批合计 15 键 × 7 语言 = 105 条，**全程未手改 locale JSON**。
`sync-i18n.mjs` 的 `_reports/_sync-report.json` 为运行痕迹，七语言 `missingCount/extrasCount/untranslatedCount` 全 0。

可复核：`footer.new\u0061pi.projectAttributionSuffix` 混淆键完好（未被脚本降级为明文）。

---

## F. 未验证清单（本机无 bun / 无 web/node_modules）

1. `bun run typecheck`（`tsgo -b`）——无法运行；本机亦无全局 `tsc`/`tsgo` → **所有 TS 改动均未经类型检查**。
2. `bun run lint`（`oxlint`）——无法运行 → 新增 import 顺序、unused 变量等未经 lint。
3. `bun run build`（`rsbuild`）/ `bun run format:check` ——无法运行。
4. `bun test` / vitest——无法运行；受影响组件的回归测试（如 `aria-label` 断言、日历翻月、刷新按钮）**未执行**。
5. 浏览器实测——未起 dev server，a11y 名称的实际计算结果、页面渲染无报错均**未经运行时验证**。
6. 因此本轮改动限定为「字面量 → `t('已存在键')`」与「补 `useTranslation()` + `aria-label`」两类**无类型/无逻辑影响**的编辑；
   新增 6 键已逐语言核对齐备（`E` 节可复核）。

---

## G. 改动文件清单（全部位于 `web/**`）

**代码 17 个**：`components/config-drawer.tsx`、`components/datetime-picker.tsx`、`components/json-editor.tsx`、
`components/password-input.tsx`、`components/tag-input.tsx`、
`features/channels/components/dialogs/param-override-editor-dialog.tsx`、
`features/channels/components/numeric-spinner-input.tsx`、`features/keys/components/api-keys-columns.tsx`、
`features/models/components/dialogs/update-config-dialog.tsx`、
`features/models/components/dialogs/upstream-conflict-dialog.tsx`、`features/models/components/models-columns.tsx`、
`features/profile/components/checkin-calendar-card.tsx`、`features/setup/components/database-step.tsx`、
`features/system-settings/models/channel-selector-dialog.tsx`、
`features/system-settings/models/tiered-pricing-editor.tsx`、`features/users/components/users-columns.tsx`、
`features/wallet/components/subscription-plans-card.tsx`

**locale 7 个**：`en/zh/zh-TW/fr/ja/ru/vi.json`（脚本生成）

**无新增文件**（临时脚本已删）；未触碰 Go / docs / 根 md / .github / scripts / .local-instance。
