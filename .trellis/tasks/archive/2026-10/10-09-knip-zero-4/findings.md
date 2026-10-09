# knip 清仓 round4（AST + 授权删文件）— 交付

## 结果

| 指标 | 本轮开始 | 现在 |
|---|---|---|
| Unused exports | 157 | **53** |
| Unused exported types | 19 | **13** |
| Unused files | 17 | **0** |
| （round3 开始时）exports+types | 301+97=398 | **66**（-83%） |

**五条硬门全绿**：`tsgo -b --force` **0 错误**｜`lint` **0 warnings / 0 errors**｜`bun test` **247 pass / 0 fail**｜`format:check` 干净｜`build` **exit 0**｜`knip --include files` **0 项**。
`bun run knip` **仍未 exit 0**（剩 66 条）。

## 一、删除了什么

**17 个文件（逐个核对绝对路径、逐个 `rm`，无通配/递归）**：
- 13 个 `src/assets/brand-icons/icon-*.tsx`（docker/facebook/figma/gitlab/gmail/medium/notion/skype/slack/stripe/trello/whatsapp/zoom）
  —— Lead 已复核外部提及 0 次，我此前也查证**无动态取用**（无 `Icons[name]`/`iconMap`/注册表/`.map()` 拼名）；
- 3 个 layout 组件：`mobile-drawer.tsx`、`nav-link-item.tsx`、`public-navigation.tsx`（0 外部提及）；
- `src/components/layout/constants.ts` —— 按 Lead 要求用**精确 import 路径**复核：
  `grep -rn "layout/constants" src` **0 命中**；其唯一引用者是 `mobile-drawer.tsx`（已删，且用的是相对 `'../constants'`）。
  删掉 mobile-drawer 后再查 **0 importer** → 确认后删除。

**符号**：本轮（AST）删除 **110** 个（176 → 66）；连同 round3 的 222 个，两轮共清 **332** 个符号。

## 二、AST 方案（Lead 授权的 `@babel/parser`）

`node_modules/@babel/parser` 已存在 ✓（`oxc-parser`/`acorn` 也在）。
脚本（临时文件，**已移出 `web/` 到 /tmp**，否则 knip 会把它报成 unused file）：
- `parse(code, {sourceType:'module', plugins:['typescript','jsx','decorators-legacy'], errorRecovery:true})`
- 遍历 `program.body`，按导出名匹配 `ExportNamedDeclaration`（声明式 / `export {}` 说明符两种）；
- 删除范围 = `node.start..node.end`，并**向前吸收 `leadingComments`**（JSDoc）；
- 若该名字在文件内仍被使用（出现次数 > 1）→ **只去掉 `export`**，不删声明；
- **级联收敛**：删完后若强制 typecheck 报出**同文件内**的 `TS6133`/`TS6192`，
  用 AST 按错误行定位并再次删除（未用 import 走 `dropAtLine` 的 import 分支，缩到实际使用的说明符），最多 3 轮；
- 仍失败 → `git checkout -- <该文件>`（**只回退这一个文件**）并记入保留。

**行级删除器做不到的复杂声明形状（含 JSX/正则/多行签名）在 AST 下全部通过** —— 这就是本轮突破点。

## 三、保留清单（66 条，逐条原因）

全部集中在 12 个文件，**原因分三类**：

**A. 类型/接口被"同名重复导出"或跨文件类型合并引用，删除会破坏契约**
- `src/lib/auth-session.ts`：`AuthTokenRotation`（interface）、`AuthRefreshHTTPResponse`、`getCommonHeaders`、`AuthRotationError`；
- `src/lib/api.ts`：`AuthTokenRotation`（type）、`bootstrapAuthentication`、`getCommonHeaders`、`AuthRotationError` ——
  与 `auth-session.ts` **同名成对**，属认证旋转契约的两侧定义，删任一侧都可能让另一侧失去对照；
- `src/features/auth/types.ts`：`OAuthProvider`；`channels/types.ts`：`ChannelOtherSettings`；
  `models/lib/model-form.ts`：`ModelFormValues` / `VendorFormValues`；`system-settings/utils/json-parser.ts`：`JsonParseResult`；
  `hooks/use-dialog.ts`：`DialogHandlers` / `DialogStateHandlers`；`chat/lib/chat-links.ts`：`ChatLinkType` / `ActiveApiKey`；
  `pricing/components/model-details.tsx`：`ModelDetailsContentProps`。

**判定依据**：这些符号在 AST 删除后会产生**跨文件**新错误签名（说明被其它模块以类型/契约方式引用），
按规程"**不确定 → 保留（宁留条目不改错）**"，脚本已自动回退。

**B. 主动保留（非 src 文本引用）**：`getDashboardSectionNavItems`、`getModelsSectionNavItems`、
`getUsageLogsSectionNavItems`（出现在 `web/.oxlintrc.json`）。

**C. 依赖/重复导出/配置提示**：均为 **0**（round3 已解决）。

## 四、过程中修掉的自身瑕疵（都属"删除的连带效应"）

- `X as X` 无用重命名（我的 import 重写器产生）→ 5 个文件、约 20 处，已正则修正；
- `consistent-type-imports` 报错 1 处（`model-details-uptime-sparkline.tsx`）→ `import type`；
- 空 `export {}` / `export type {}`、`newline-after-import` 等 → 已用 `bun run format` + `oxlint --fix` + 定点脚本清干净。

## 五、离 exit 0 还差什么

剩 **53 exports + 13 types**，全部是第三节 A 类：**删除会产生跨文件新错误签名**。
它们要么是**真实的跨模块类型契约**（删除需要同步改调用方，属**改行为**，超出"只删死代码"的授权），
要么需要人工判断。**按规程我没有硬删**。

**建议**：这 66 条留给人工评审（每条约 1 分钟即可判断"是否真是死类型"），或授权我"删除 + 同步修调用方"的更大范围改动。
