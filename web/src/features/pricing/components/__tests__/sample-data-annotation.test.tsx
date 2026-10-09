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
const { ModelDetailsApi } = await import('../model-details-api')

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

const roots: Array<{ unmount: () => void }> = []

after(async () => {
  i18n.removeResourceBundle('en', 'translation')
})

afterEach(async () => {
  await act(async () => {
    for (const root of roots.splice(0)) root.unmount()
  })
})

async function renderApiTab(): Promise<HTMLElement> {
  const container = document.createElement('div')
  document.body.append(container)
  const root = createRoot(container)
  roots.push(root)

  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: Infinity } },
  })
  // `useStatus()` reads this cache entry; seeding keeps the tab offline.
  queryClient.setQueryData(
    ['status'],
    { version: '0.0.7' },
    {
      updatedAt: Date.now(),
    }
  )

  await act(async () => {
    root.render(
      <QueryClientProvider client={queryClient}>
        <I18nextProvider i18n={i18n}>
          <ModelDetailsApi
            model={{
              id: 1,
              model_name: 'gpt-5',
              quota_type: 0,
              model_ratio: 1,
              completion_ratio: 1,
              enable_groups: ['default'],
              supported_endpoint_types: ['openai'],
            }}
            endpointMap={{ openai: { path: '/v1/chat/completions' } }}
          />
        </I18nextProvider>
      </QueryClientProvider>
    )
    for (let tick = 0; tick < 12; tick++) await Promise.resolve()
  })

  return container
}

describe('API tab sample-data annotation', () => {
  test('marks exactly the two locally generated tables', async () => {
    const container = await renderApiTab()
    const notes = [
      ...container.querySelectorAll('[data-testid="sample-data-note"]'),
    ]

    // Four sections render here and only two are mock-built, so the count is
    // the contract: a badge leaking into the code-sample or auth block (or
    // disappearing from a generated one) changes this number.
    assert.equal(notes.length, 2)

    const marked = notes.map(
      (note) => note.closest('section')?.textContent ?? ''
    )
    assert.equal(
      marked.some((text) => text.includes(en['Supported parameters'])),
      true
    )
    assert.equal(
      marked.some((text) => text.includes(en['Rate limits'])),
      true
    )
  })

  test('says the values were not measured instead of implying live data', async () => {
    const container = await renderApiTab()
    const note = container.querySelector('[data-testid="sample-data-note"]')

    assert.notEqual(note, null)
    assert.equal(note?.textContent?.includes(en['Sample data']), true)
    assert.equal(
      note?.textContent?.includes(
        en['Not measured on this deployment — shown for reference only.']
      ),
      true
    )
  })

  test('leaves blocks backed by the API unmarked', async () => {
    const container = await renderApiTab()

    for (const section of container.querySelectorAll('section')) {
      const text = section.textContent ?? ''
      if (
        text.includes(en['Supported parameters']) ||
        text.includes(en['Rate limits'])
      ) {
        continue
      }
      assert.equal(
        section.querySelector('[data-testid="sample-data-note"]'),
        null,
        `unexpected sample-data note in: ${text.slice(0, 40)}`
      )
    }
  })
})
