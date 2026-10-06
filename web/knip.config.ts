import type { KnipConfig } from 'knip'

const config: KnipConfig = {
  // 测试通过 `bun test` / node:test 执行，knip 不会自动识别入口 → 显式声明，
  // 否则会把测试文件当成"未使用文件"（2026-10-06 实测：95 项里有 50+ 是测试）。
  entry: [
    // 应用入口：rsbuild 从 index.html 引到这里；不声明的话 knip 会把整条
    // 依赖链（main.tsx、路由生成文件、样式、i18n 配置…）都当成未使用（实测误报）。
    'index.html',
    'src/main.tsx',
    'src/**/*.test.{ts,tsx}',
    'src/**/__tests__/**/*.{ts,tsx}',
  ],
  ignore: [
    // shadcn 底座组件：按需引入、相当一部分当前无引用，属成套件而非散乱死码。
    // 注意（2026-10-06 复核）：被 ignore 的树不可见会导致"未使用依赖"误报
    // （recharts / tokenlens / @xyflow/react / embla-carousel-react / react-resizable-panels），
    // 不要据此删除这些依赖。
    'src/components/ui/**',
    // 成套 AI 组件库（22 文件）：当前无引用，但删除会连带清掉一批依赖，
    // 属产品决策范围，先登记为 ignore 而非静默删除（2026-10-06）。
    'src/components/ai-elements/**',
    // 注：`src/i18n/static-keys.ts`（`STATIC_I18N_KEYS` 登记表）曾登记在此。
    // 2026-10-06 已把它接进 i18n 校验链路：`scripts/check-i18n-keys.mjs` 直接 import 它，
    // 而该脚本经 package.json 的 `i18n:check` 成为 knip 入口 ⇒ 文件已可达，ignore 冗余。
    // 不要再加回来；若哪天 knip 又报它未使用，说明校验脚本与 package.json 的接线断了，
    // 应先修接线而不是 ignore。
    // 注：`src/routeTree.gen.ts`（TanStack Router 生成文件，已入库）曾登记在此。
    // 2026-10-06 实测：它由 `src/main.tsx` 可达，knip 会提示"Remove from ignore"；
    // 移除该 ignore 后 `knip` 全量输出与保留时**逐节一致**（仅是提示 3→2），
    // 故按提示移除，避免留下无效配置。不要再加回来。
  ],
  ignoreDependencies: ['tailwindcss', 'tw-animate-css'],
}

export default config
