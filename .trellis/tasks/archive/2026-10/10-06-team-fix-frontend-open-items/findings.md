# 前端开放项处理结果（2026-10-06）

范围：`web/**`。基线：`bun test` 151 pass / 3 fail；`knip --include files` 8 unused files。

## 1) 三个失败测试 → 取向：**恢复**（不是改测试）

**史实**：`web/src/features/keys/components/api-key-group-cell.tsx` 在仓库里只有 2 个提交
（`1092e46` 根提交 + `9f0cdf9` 仅格式化），根提交 `1092e46` 就是自维护基线 ⇒
**注释掉的成因在 squash 历史里不可考**；`web/MAINTENANCE.md:233` 曾记为"既有失败，保持失败并留痕"。

**判据（为何是恢复而非改测试）**：

| 证据 | 位置 | 含义 |
| --- | --- | --- |
| 同款动效仍在使用中 | `web/src/features/keys/components/api-key-group-combobox.tsx:111,119,173,179` | `AUTO_GROUP_FRAME_CLASS_NAME` / `AutoGroupFlowBorder` 在姊妹组件里活着且测试通过 ⇒ 设计未废弃 |
| 动画 CSS 完整保留 | `web/src/styles/index.css:655-700` | 若设计退役，`@keyframes` + `@media (prefers-reduced-motion)` 应先被删 |
| 被注释组件成死码 | `web/src/features/keys/components/auto-group-visuals.tsx:131` | `AutoGroupBadge` 唯一引用就是那行注释 ⇒ 不恢复即真死码 |
| 降级语义正好对上 | `auto-group-visuals.tsx:36` + `index.css:695` | 测试的 reduced-motion 契约 = `shouldReduceMotion → null` + CSS `display:none` |

**改动**（2 行，未放宽任何断言）：`api-key-group-cell.tsx:31` 取消 import 注释、`:70` 取消 JSX 注释。
结果：该文件 4/4 通过，全量 **154 pass / 0 fail**。

## 2) knip：8 → 0

逐项确证后**删除 7 个真死码**（全量 `typecheck` exit 0 反证无人引用）：

| 文件 | 结论依据 |
| --- | --- |
| `src/components/empty-state.tsx` | 零引用；`src/components/ui/empty` 被 10+ 处直接使用 |
| `src/features/auth/index.ts` | 纯 barrel，零引用（`@/features/auth/api` 有 8 处直连） |
| `src/features/home/constants.ts` | `AI_APPLICATIONS`/`AI_MODELS` 全仓仅定义处出现 |
| `src/features/pricing/hooks/index.ts` | barrel 零引用；两个 hook 均被相对路径直连 |
| `src/features/pricing/lib/index.ts` | barrel 零引用；7 个源模块全被**相对路径**直连 ⇒ 无级联 |
| `src/features/rankings/lib/index.ts` | barrel 零引用；`format.ts` 被 3 处相对路径使用 |
| `src/features/usage-logs/lib/index.ts` | barrel 零引用；`format.ts`/`filter.ts` 被相对路径使用 |

**保留并登记 ignore 1 项**：`src/i18n/static-keys.ts`（577 行 `STATIC_I18N_KEYS`）——
`web/AGENTS.md:71` 明文约定它是动态文案的登记位置；但当前**无任何代码消费者**
（`scripts/sync-i18n.mjs` 只按正则提取）。结论写进 `web/knip.config.ts:23-29`：
保留 + 标注"需接进 i18n 提取脚本才真正生效"（属工具改动，未在本轮做）。

另：按 knip 提示移除冗余的 `src/routeTree.gen.ts` ignore —— 实测移除前后**逐节输出一致**（仅提示 3→2），
已把"不要再加回来"的原因写在 `knip.config.ts:30-33`。
**保留**了"被 ignore 的树会导致未使用依赖误报"的警告注释（`knip.config.ts:15-18`）。

## 3) 14 个 `ui/**` → 结论：**全部保留**

`components.json` 的 `aliases.ui = "@/components/ui"` ⇒ 该目录是 **shadcn CLI 安装目标**（registry 约定），
非手写死码。13 个生产文件全部符合本项目配置的 `style: base-nova` + `iconLibrary: hugeicons`
（均引 `@base-ui/react/*` 与 `@hugeicons/*`）。关键：**删除会连带孤立依赖**——

- `src/components/ui/chart.tsx` 是 `recharts` 的**唯一**消费者
- `src/components/ui/resizable.tsx` 是 `react-resizable-panels` 的**唯一**消费者

删掉即复现 `knip.config.ts:16-18` 警告的"未使用依赖误报"陷阱。第 14 个
`src/components/ui/dropdown-menu.test.tsx` 是测试文件（`bun test` entry，非死码）。

## 五条命令实测（全部 exit 0）

| 命令 | exit | 结果 |
| --- | --- | --- |
| `bun run typecheck` | 0 | 无输出 |
| `bun run lint` | 0 | 21 warnings / **0 errors**（与基线同，含 footer no-danger 设计使然） |
| `bun test` | 0 | **154 pass / 0 fail**（基线 151/3） |
| `bun run build` | 0 | 产物正常 |
| `bun run knip --include files` | 0 | **无任何输出**（0 unused files，0 hints） |

## 遗留 / 超越本轮范围

- `web/MAINTENANCE.md:233` 那行"保持失败"已过期，但该文件在 `web/**` 之外，本轮**未改**，交 Lead 处理。
- `src/i18n/static-keys.ts` 目前不生效，建议后续接进 `scripts/sync-i18n.mjs`（可暴露 7 语言的缺 key）。
