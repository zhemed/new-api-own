# 复核结论（audit-frontend-writes，2026-10-06）

被复查：commit 208483c / 97fd7d6 / 5ad97b7。方式：静态全量引用解析 + 逐条实测 + 真机浏览器验证。
所有临时探针文件已删除；`git status -- web` 干净。

## Q1 33 个删除是否安全 → 安全（高置信）

1. 33 文件清单 = `git show --name-status --diff-filter=D 208483c | grep web/src`（33 行），全部来自 1092e46 自维护基线，是真源码而非工作树产物。
2. 引用形式穷举：
   - 路径 stem 全仓库 grep（排除 `.trellis/**`、`web/src/i18n/**`、`bun.lock`）→ **0 命中**。
   - 导出的符号（Logo/Glow/Section/StatItem/Route/…）逐个 grep → 仅命中注释、i18n 字面量、同名局部接口（`features/home/components/sections/stats.tsx:90` 的局部 `StatItem`）。
   - `import(`/`require(` 全部为字面量（无变量式动态导入），`React.lazy` 命中的只有 `@/components/ui/markdown`、`@visactor/vchart`。
   - **1006 个文件的所有 import specifier 可解析（`@/` 与相对路径，0 unresolved）**：任何文件（含 barrel）若还引用被删模块都会在此暴露。
   - `src/routeTree.gen.ts` 只引用 `./routes/**`；rsbuild 中 `tanstackRouter()` 使用默认 `routesDirectory=src/routes`，33 个文件无一在 `src/routes/`。
   - `components.json`（shadcn，tailwind 配置为 CSS 而非 content globs）、`index.html`、`src/styles/*.css`、`web/scripts/*.mjs`、`netlify.toml` 均无引用。
   - 重点组：`features/home/components/index.ts` 只导出 `sections/*`（存活实现）；`features/*/lib/index.ts` 等 barrel 均未涉及被删文件；`hooks/*` 无引用。
3. 交叉验证：`tsgo -b` 与 `rsbuild build` 均 exit 0（缺模块必然失败）。
4. 遗留（非破坏）：删除后 `src/features/home/constants.ts`、`src/components/empty-state.tsx`、`features/auth/index.ts`、`features/{pricing,rankings,usage-logs}/…index.ts`、`src/i18n/static-keys.ts` 等 9 项仍无引用（knip 报告），清理未收尾。

## Q2 knip 配置 → 部分过宽，另有 3 条冗余（knip 自己提示）

- 无 `ignore` 时 45 项 src 未使用文件；现有配置只报 9 项 → **ignore 掩盖 36 个真实无引用文件**（22 个 `ai-elements/**` + 14 个 `ui/**`）。ai-elements 有注释登记为产品决策；`ui/**` 无任何说明。
- ignore 的副作用：只被 ignore 树使用的依赖被误报为「未使用依赖」——`recharts`(ui/chart.tsx)、`tokenlens`(ai-elements/context.tsx)、`@xyflow/react`(ai-elements/*)、`embla-carousel-react`(ui/carousel.tsx)、`react-resizable-panels`(ui/resizable.tsx)。**不得据此 `bun remove`**。
- `entry` 判定正确、非过宽：`src/main.tsx` 与 rsbuild `source.entry` 一致；实测去掉 entry 后测试文件被误报为未使用（41+ 项）。`index.html` 实际是 no-op（该文件内无 `<script>`，knip 无法据此发现 main.tsx），无害但可删。
- knip 自报 3 条冗余：`src/routeTree.gen.ts`、`tailwindcss`、`tw-animate-css` 均应 Remove from ignore/ignoreDependencies。
- 现状：`bun run knip --include files` **exit 1**（9 项），「knip 干净」不成立。

## Q3 jsdom 改动是否弱化 → 未弱化，实为修好失败测试；但注释机制说错

- 与 `0941a4f` 逐行 diff：三组断言**完全未变**，唯一新增 `assert.equal(purify.isSupported, true)`（合理且通过）。
- 实测把 `0941a4f` 原文件原样跑一遍：**1 pass / 2 fail**（happy-dom 下 DOMPurify 未剥离 `<script>`、并改写 href 序列化）。换 jsdom 后 3 pass。
- 注释事实错误：happy-dom 下实测 `isSupported === true`（注释称 false）；真实故障是 happy-dom 消毒不完整（输出保留 `<script>alert(1)</script>`）。
- 覆盖缺口（既有）：该测试**从未 import `footer.tsx`**，只测 DOMPurify 本身，Footer 的 isSupported 分支无测试保护。

## Q4 footer 消毒加固是否有缺陷 → 语义正确，不引入「该显示不显示」

- `dist/purify.es.mjs:1984`：`isSupported===false` 时 `sanitize()` **原样返回 dirty** → 加固方向正确（原写法确实会注入未消毒 HTML）。
- 真机 Chromium 153 注入 `dompurify@3.4.11` 实测：`isSupported === true`，`sanitize` 剥离 `<script>`/`onerror` → 浏览器走消毒分支，footer 正常渲染。
- 兜底只在 DOMPurify 报告不支持的运行环境生效，此时仅隐藏「自定义 HTML 区块」，法务链接与版权署名仍渲染 → 不存在整块 footer 消失。
- 残留（可选）：环境不支持时会静默显示空自定义区块，建议加日志/占位。

## 五条命令实测

| 命令 | exit | 结果 |
| --- | --- | --- |
| `bun run typecheck` | 0 | 无输出 |
| `bun run lint` | 0 | 21 warnings / 0 errors（含 footer.tsx:245 no-danger，设计使然） |
| `bun test` | 1 | 151 pass / 3 fail，全部在 `features/keys/components/__tests__/api-key-group-cell.test.tsx`（`[data-auto-group-frame]` 计数 1≠2、0≠1） |
| `bun run build` | 0 | 产物正常 |
| `bun run knip --include files` | 1 | 9 unused files + 1 hint |

3 个失败与本轮改动无关：测试与组件自 9f0cdf9 起未被触碰，happy-dom/react 版本未被 208483c 改动（lock 仅新增 jsdom 树），单独运行同样 3 fail。

## 无法验证

- 3020 演示实例的真实页面渲染（未获授权访问/重启服务）。
- `208483c^` 的整树构建（禁止在仓库建 worktree；以依赖版本比对 + 基线文件原样复跑替代）。
