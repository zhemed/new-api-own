# knip 清仓 round3（冻结期单线程）— 交付

**结果**：`Unused exports` **301 → 157**、`Unused exported types` **97 → 19**（两类合计 **398 → 176**，降 56%）。
四条硬门全绿（typecheck 0 / lint 0-0 / test 247 pass / format 干净 / build exit 0）。
**knip 未 exit 0**：剩 176 条，且新出现 `Unused files (17)`（原因见第四节）。

## 一、执行方式（严格遵守安全规程）

- **单线程、唯一写入者**；未叫醒任何成员。
- **每批只改一个文件**，改完立刻 `bunx tsgo -b --force`，**按错误签名**（文件, 代码, 符号）比较；
  有**任何**新签名 → **只 `git checkout -- <该文件>`**，记入保留清单。
- **未使用** `git stash` / `git clean` / 目录级或全仓 checkout / `--no-verify`。
- 每 20 批跑 `bun test`：checkpoint 20 / 40 / … 均为 **247 pass / 0 fail**（一次报 `[]` 是我脚本没读 stderr 的假警报，已修）。

## 二、删除统计

| 项 | 数 |
|---|---|
| 处理文件（两轮合计） | 125 个（第二轮 knip 重新解析后） |
| **成功修改文件** | **86** |
| **删除符号数** | **≈216** |
| **因复核失败回退（保留）的文件** | **45** |

## 三、保留清单

### A. 回退保留的 45 个文件（原因统一：我的行级删除器对它们的**声明形状**判断不安全，
### 首次尝试即产生新错误签名（多为 `TS1161` 未终止正则 / `TS1005`），按"宁留不改错"回退）
`components/layout/components/{mobile-drawer,nav-link-item,public-navigation}.tsx`、`components/status-badge.tsx`、
`context/font-provider.tsx`、`features/auth/lib/{oauth,validation}.ts`、`features/channels/{api.ts,lib/channel-form.ts,lib/channel-type-config.ts,lib/channel-utils.ts,lib/model-mapping-validation.ts}`、
`features/chat/lib/chat-links.ts`、`features/dashboard/lib/index.ts`、`features/keys/lib/{api-key-form.ts,index.ts}`、
`features/models/{api.ts,lib/model-form.ts,lib/model-utils.ts,lib/vendor-actions.ts,types.ts}`、
`features/playground/lib/{message/message-utils.ts,storage/storage.ts}`、`features/pricing/components/{index.ts,model-details.tsx}`、
`features/profile/lib/format.ts`、`features/subscriptions/api.ts`、`features/system-settings/utils/{json-parser.ts,json-validators.ts}`、
`features/usage-logs/{api.ts,components/columns/column-helpers.tsx,constants.ts,lib/filter.ts,lib/mappers.ts,lib/utils.ts}`、
`hooks/index.ts`、`hooks/use-dialog.ts`、`i18n/languages.ts`、`lib/{api.ts,auth-session.ts,colors.ts,format.ts,passkey.ts,time.ts,utils.ts}`

### B. 按规则**主动保留**的 3 个符号（名字出现在 `web/.oxlintrc.json`，属"非 src 文本引用"）
`getDashboardSectionNavItems`、`getModelsSectionNavItems`、`getUsageLogsSectionNavItems`

### C. 风险区查证结论
- `src/assets/brand-icons/**`：**已查无动态用法**（无 `Icons[name]` / `iconMap` / 注册表 / `.map()` 拼名）；
  本轮**已删**其未引用导出（13 个图标文件）。
- 全仓扫描 361 个符号在 `.json/.md/.mjs/.sh/.yml` 中的出现 → 仅上述 3 个命中（`.oxlintrc.json`），其余无字符串引用。

## 四、未归零的两块（如实）

1. **剩 157 exports + 19 types**：集中在第三节 A 的 45 个文件。它们不是我判断"不该删"，
   而是**我的行级删除器对复杂声明形状会生成错误代码** —— 安全网每次都成功拦下（无错签名才采纳），
   代价是这些文件保持原样。要继续清需要 **TS AST**（`typescript` 包）或人工逐条。
2. **新出现 `Unused files (17)`**（13 个 brand-icon + 4 个 layout 文件）：删掉 barrel 的 re-export 后这些文件**失去可达性**。
   按你的规程"**不要删文件**"，我没有删它们 → `knip --include files` 由 0 变 17。
   **取舍**：要么删这些文件（违反你本轮规程），要么恢复那些 barrel re-export（会把 exports 条数加回去）。
   **我选择如实报告，不擅自决定**。

## 五、收工判定

| 门 | 结果 |
|---|---|
| `bunx tsgo -b --force` | **0 错误** ✓ |
| `bun run lint` | **0 warnings / 0 errors** ✓ |
| `bun test` | **247 pass / 0 fail** ✓ |
| `bun run format:check` | 干净 ✓ |
| `bun run build` | exit 0（57331.4 kB / gzip 16541.4 kB）✓ |
| `bun run knip` | **未 exit 0**：157 exports + 19 types + 17 files |
| `bun run knip --include files` | **未 0**（同 17 个文件） |

## 六、过程中的修复（非删除类）

- 清掉 5 处 `export {}` / `export type {}` 空导出（我删除后留下的 lint error）；
- 修复 35 + 4 处 `newline-after-import` warning（我删除声明后留下的空行）；
- 清掉 `system-config-store.ts` 末尾 1 处 `no-unused-expressions` 残留表达式。
