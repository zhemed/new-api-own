import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

import { defineConfig, loadEnv } from '@rsbuild/core'
import { pluginReact } from '@rsbuild/plugin-react'
import { pluginTailwindcss } from '@rsbuild/plugin-tailwindcss'
import { tanstackRouter } from '@tanstack/router-plugin/rspack'

const __dirname = path.dirname(fileURLToPath(import.meta.url))

/**
 * Resolve the version stamped into the client bundle.
 *
 * Precedence mirrors the Docker build (`Dockerfile` exports
 * `VITE_REACT_APP_VERSION=$(cat VERSION)`), with the repository `VERSION`
 * file as a fallback so that a plain local `bun run build` also carries the
 * real version instead of the `0000` placeholder.
 */
function resolveAppVersion(rawEnvVersion: string | undefined): string {
  if (rawEnvVersion && rawEnvVersion.trim().length > 0) {
    return rawEnvVersion.trim()
  }
  try {
    const fromRepo = fs
      .readFileSync(path.resolve(__dirname, '../VERSION'), 'utf8')
      .trim()
    if (fromRepo.length > 0) return fromRepo
  } catch {
    // VERSION file is optional outside the repository build context.
  }
  return ''
}

export default defineConfig(({ envMode }) => {
  // `loadEnv` only surfaces variables that actually exist, so make sure the
  // resolved version is present in the environment *before* collecting it.
  // The Docker build already exports VITE_REACT_APP_VERSION; a plain local
  // `bun run build` falls back to the repository VERSION file.
  const appVersion = resolveAppVersion(process.env.VITE_REACT_APP_VERSION)
  if (appVersion.length > 0) {
    process.env.VITE_REACT_APP_VERSION = appVersion
  }

  const env = loadEnv({ mode: envMode, prefixes: ['VITE_'] })
  const serverUrl =
    process.env.VITE_REACT_APP_SERVER_URL ||
    env.rawPublicVars.VITE_REACT_APP_SERVER_URL ||
    'http://localhost:3000'

  const isProd = envMode === 'production'
  const devProxy = Object.fromEntries(
    (['/api', '/mj', '/pg'] as const).map((key) => [
      key,
      { target: serverUrl, changeOrigin: true },
    ])
  ) as Record<string, { target: string; changeOrigin: boolean }>

  return {
    plugins: [pluginReact(), pluginTailwindcss({ optimize: false })],
    // Rsbuild 2: replaces deprecated `performance.chunkSplit` (RSPack 2 aligned)
    splitChunks: {
      preset: 'default',
      cacheGroups: {
        'vendor-react': {
          test: /node_modules[\\/](react|react-dom)[\\/]/,
          name: 'vendor-react',
          chunks: 'all',
          priority: 0,
          enforce: true,
        },
        'vendor-ui-primitives': {
          test: /node_modules[\\/](@base-ui|@radix-ui)[\\/]/,
          name: 'vendor-ui-primitives',
          chunks: 'all',
          priority: 0,
          enforce: true,
        },
        'vendor-tanstack': {
          test: /node_modules[\\/]@tanstack[\\/]/,
          name: 'vendor-tanstack',
          chunks: 'all',
          priority: 0,
          enforce: true,
        },
      },
    },
    source: {
      // `loadEnv` only *collects* VITE_* values; without `define` they never
      // reach the client bundle (build-metadata.ts then fell back to `0000`,
      // so the build-id stayed constant regardless of the real version).
      //
      // `__APP_VERSION__` is set explicitly because Rsbuild owns the
      // `import.meta.env` object itself and it wins over a dotted
      // `import.meta.env.VITE_*` define when the variable is absent from the
      // process environment (e.g. a plain local `bun run build`).
      define: {
        ...env.publicVars,
        __APP_VERSION__: JSON.stringify(appVersion),
      },
      entry: {
        index: './src/main.tsx',
      },
    },
    resolve: {
      alias: {
        '@': path.resolve(__dirname, './src'),
      },
    },
    html: {
      template: './index.html',
    },
    server: {
      host: '0.0.0.0',
      strictPort: false,
      proxy: devProxy,
    },
    output: {
      // Production optimizations
      minify: isProd,
      target: 'web',
      distPath: {
        root: 'dist',
      },
      // Rely on Rsbuild default legalComments ("linked" → per-chunk *.LICENSE.txt) in all modes.
      // Do not set "none" in production: that strips minifier-preserved third-party notices and
      // extracted license files, which some distributions require for open-source compliance.
    },
    performance: {
      // Remove console in production
      removeConsole: isProd ? ['log'] : false,
      buildCache: false,
    },
    tools: {
      rspack: {
        plugins: [
          tanstackRouter({
            target: 'react',
            // Dev: avoid per-route async chunks (reduces white flash on navigation + faster HMR feedback).
            // Prod: keep route-based code splitting.
            autoCodeSplitting: isProd,
          }),
        ],
      },
    },
  }
})
