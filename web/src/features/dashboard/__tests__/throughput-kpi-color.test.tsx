/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { after, afterEach, describe, test } from 'node:test'

import { Window } from 'happy-dom'

const domWindow = new Window()
const domGlobals = [
  'window',
  'document',
  'navigator',
  'HTMLElement',
  'HTMLButtonElement',
  'SVGElement',
  'Node',
  'Element',
  'Event',
  'CustomEvent',
  'MutationObserver',
  'ResizeObserver',
  'requestAnimationFrame',
  'cancelAnimationFrame',
  'getComputedStyle',
] as const

for (const key of domGlobals) {
  Object.defineProperty(globalThis, key, {
    configurable: true,
    value: domWindow[key],
  })
}

const { act } = await import('react')
const { createRoot } = await import('react-dom/client')
const { createInstance } = await import('i18next')
const { I18nextProvider, initReactI18next } = await import('react-i18next')
const { QueryClient, QueryClientProvider } =
  await import('@tanstack/react-query')
const { PerformanceOverview } =
  await import('../components/models/performance-overview')
const { PerformanceHealthPanel } =
  await import('../components/overview/performance-health-panel')

const en = (
  JSON.parse(
    readFileSync(
      new URL('../../../i18n/locales/en.json', import.meta.url),
      'utf8'
    )
  ) as { translation: Record<string, string> }
).translation

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: { en: { translation: en } },
})

const reactTestGlobals = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean
}
reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = true

const roots: Array<{ unmount: () => void }> = []

after(async () => {
  i18n.removeResourceBundle('en', 'translation')
})

afterEach(async () => {
  await act(async () => {
    for (const root of roots.splice(0)) root.unmount()
  })
})

/** Class of the leaf element rendering exactly `text`, or '' when absent. */
function classOfValue(container: HTMLElement, text: string): string {
  for (const element of container.querySelectorAll('span,div')) {
    if (element.children.length === 0 && element.textContent?.trim() === text) {
      return element.className
    }
  }
  return ''
}

async function renderWithSummary(
  Component: React.ComponentType,
  avgTps: number
): Promise<HTMLElement> {
  const container = document.createElement('div')
  document.body.append(container)
  const root = createRoot(container)
  roots.push(root)

  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: Infinity } },
  })
  queryClient.setQueryData(
    ['perf-metrics-summary', 24],
    {
      data: {
        models: [
          {
            model_name: 'seed-model',
            avg_latency_ms: 500,
            avg_tps: avgTps,
            success_rate: 100,
            request_count: 10,
          },
        ],
      },
    },
    { updatedAt: Date.now() }
  )

  await act(async () => {
    root.render(
      <QueryClientProvider client={queryClient}>
        <I18nextProvider i18n={i18n}>
          <Component />
        </I18nextProvider>
      </QueryClientProvider>
    )
    for (let tick = 0; tick < 8; tick++) await Promise.resolve()
  })

  return container
}

const CASES: Array<{ avgTps: number; shown: string; expected: string }> = [
  // Same thresholds as the rest of the console: [0,50) slow, [50,100) medium,
  // [100,∞) fast, and no measurement stays neutral.
  { avgTps: 20, shown: '20.0 t/s', expected: 'text-red-600' },
  { avgTps: 75, shown: '75.0 t/s', expected: 'text-amber-600' },
  { avgTps: 120, shown: '120.0 t/s', expected: 'text-emerald-600' },
  { avgTps: 0, shown: '—', expected: 'text-muted-foreground' },
]

for (const [name, Component] of [
  ['PerformanceOverview', PerformanceOverview],
  ['PerformanceHealthPanel', PerformanceHealthPanel],
] as const) {
  describe(`${name} throughput KPI colour`, () => {
    for (const testCase of CASES) {
      test(`renders ${testCase.shown} with ${testCase.expected}`, async () => {
        const container = await renderWithSummary(Component, testCase.avgTps)

        const className = classOfValue(container, testCase.shown)
        assert.notEqual(className, '')
        assert.equal(className.includes(testCase.expected), true)
      })
    }
  })
}
