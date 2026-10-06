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
const { LogSettingsSection } = await import('../log-settings-section')
const { api } = await import('@/lib/api')

// Expectation copy comes from the real locale file, so the assertions pin the
// i18n key the panel picked rather than the English wording here.
const en = (
  JSON.parse(
    readFileSync(
      new URL('../../../../i18n/locales/en.json', import.meta.url),
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

const realGet = api.get
const roots: Array<{ unmount: () => void }> = []

after(async () => {
  i18n.removeResourceBundle('en', 'translation')
})

afterEach(async () => {
  await act(async () => {
    for (const root of roots.splice(0)) root.unmount()
  })
  api.get = realGet
})

const MB = 1024 * 1024

/** Serve the performance/logs payload, keeping every other read empty. */
function stubLogFiles(logFilesData: Record<string, unknown>): void {
  api.get = (async (url: string) => {
    if (url === '/api/performance/logs') {
      return { data: { success: true, data: logFilesData } }
    }
    return { data: { success: true, data: null } }
  }) as unknown as typeof api.get
}

async function renderSection(i18nInstance: typeof i18n = i18n) {
  const container = document.createElement('div')
  document.body.append(container)
  const root = createRoot(container)
  roots.push(root)

  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })

  await act(async () => {
    root.render(
      <QueryClientProvider client={queryClient}>
        <I18nextProvider i18n={i18nInstance}>
          <LogSettingsSection defaultEnabled={false} />
        </I18nextProvider>
      </QueryClientProvider>
    )
    for (let tick = 0; tick < 12; tick++) await Promise.resolve()
  })

  return container
}

function usageLine(container: HTMLElement): string | null {
  return (
    container.querySelector('[data-testid="memory-log-usage"]')?.textContent ??
    null
  )
}

describe('memory log usage line', () => {
  test('shows usage and the configured limit with the row count', async () => {
    stubLogFiles({
      enabled: true,
      log_dir: '/tmp/logs',
      file_count: 1,
      total_size: 2048,
      memory_log_bytes: 12 * MB,
      memory_log_max_bytes: 200 * MB,
      memory_log_rows: 176000,
    })

    const container = await renderSection()

    assert.equal(
      usageLine(container),
      'Memory log usage 12 MB / limit 200 MB (176,000 rows)'
    )
    assert.equal(
      container.querySelector('[data-testid="memory-log-near-limit"]'),
      null
    )
  })

  test('says the limit is unset instead of showing zero out of zero', async () => {
    stubLogFiles({
      enabled: true,
      log_dir: '/tmp/logs',
      file_count: 0,
      total_size: 0,
      memory_log_bytes: 12 * MB,
      memory_log_max_bytes: 0,
      memory_log_rows: 176000,
    })

    const container = await renderSection()

    assert.equal(
      usageLine(container),
      'Memory log usage 12 MB (no limit configured)'
    )
    assert.equal(usageLine(container)?.includes('0 Bytes'), false)
  })

  test('flags the soft warning once usage reaches the budget threshold', async () => {
    stubLogFiles({
      enabled: true,
      log_dir: '/tmp/logs',
      file_count: 0,
      total_size: 0,
      memory_log_bytes: 190 * MB,
      memory_log_max_bytes: 200 * MB,
      memory_log_rows: 900000,
    })

    const container = await renderSection()

    // The line carries the numbers; the badge is a separate, non-numeric hint.
    assert.equal(
      usageLine(container)?.includes(
        'Memory log usage 190 MB / limit 200 MB (900,000 rows)'
      ),
      true
    )
    assert.equal(
      container.querySelector('[data-testid="memory-log-near-limit"]')
        ?.textContent,
      en['Approaching the limit']
    )
    // The panel only reports — it must never claim anything is cleaned up.
    assert.equal(
      (container.textContent ?? '').toLowerCase().includes('automatically'),
      false
    )
  })

  test('renders nothing when the server does not report the fields yet', async () => {
    stubLogFiles({
      enabled: true,
      log_dir: '/tmp/logs',
      file_count: 1,
      total_size: 2048,
    })

    const container = await renderSection()

    assert.equal(usageLine(container), null)
  })

  test('reports the payload footprint even without a disk log directory', async () => {
    stubLogFiles({
      enabled: false,
      log_dir: '',
      file_count: 0,
      total_size: 0,
      memory_log_bytes: 5 * MB,
      memory_log_max_bytes: 200 * MB,
      memory_log_rows: 1000,
    })

    const container = await renderSection()

    assert.equal(
      usageLine(container),
      'Memory log usage 5 MB / limit 200 MB (1,000 rows)'
    )
  })

  test('renders the row count when the interface language tag needs normalization', async () => {
    // The app reports its own language tags ("zhCN" without a hyphen), which
    // Intl rejects with a RangeError; the row count must go through the
    // project's normalizer instead of the raw tag.
    const zhCn = createInstance()
    await zhCn.use(initReactI18next).init({
      lng: 'zhCN',
      resources: { zhCN: { translation: en } },
    })
    stubLogFiles({
      enabled: true,
      log_dir: '/tmp/logs',
      file_count: 1,
      total_size: 2048,
      memory_log_bytes: 12 * MB,
      memory_log_max_bytes: 200 * MB,
      memory_log_rows: 176000,
    })

    const container = await renderSection(zhCn)

    assert.equal(zhCn.language, 'zhCN')
    assert.equal(
      usageLine(container),
      'Memory log usage 12 MB / limit 200 MB (176,000 rows)'
    )
  })
})
