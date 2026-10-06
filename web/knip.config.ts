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
    'src/components/ui/**',
    // 成套 AI 组件库（22 文件）：当前无引用，但删除会连带清掉一批依赖，
    // 属产品决策范围，先登记为 ignore 而非静默删除（2026-10-06）。
    'src/components/ai-elements/**',
    'src/routeTree.gen.ts',
  ],
  ignoreDependencies: ['tailwindcss', 'tw-animate-css'],
}

export default config
