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
import { after, describe, test } from 'node:test'

import { Window } from 'happy-dom'

const domWindow = new Window()
const domGlobals = [
  'window',
  'document',
  'navigator',
  'localStorage',
  'sessionStorage',
  'HTMLElement',
  'HTMLInputElement',
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
const { QueryClient, QueryClientProvider } =
  await import('@tanstack/react-query')
const { createInstance } = await import('i18next')
const { I18nextProvider, initReactI18next } = await import('react-i18next')
const { RatioSettingsCard } = await import('../ratio-settings-card')

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

const RESET_SCOPE_KEY =
  'This restores model pricing to the default templates: model, completion, cache and cache-write ratios, per-call prices, image and audio ratios, and the billing mode with its billing expression. Group ratios and usable groups are not affected.'

const modelDefaults = {
  ModelPrice: '{}',
  ModelRatio: '{}',
  CacheRatio: '{}',
  CreateCacheRatio: '{}',
  CompletionRatio: '{}',
  ImageRatio: '{}',
  AudioRatio: '{}',
  AudioCompletionRatio: '{}',
  ExposeRatioEnabled: false,
  BillingMode: '{}',
  BillingExpr: '{}',
}

const groupDefaults = {
  GroupRatio: '{}',
  TopupGroupRatio: '{}',
  UserUsableGroups: '{}',
  GroupGroupRatio: '{}',
  AutoGroups: '[]',
  MaxTokenAutoGroups: 1,
  DefaultUseAutoGroup: false,
  GroupSpecialUsableGroup: '{}',
}

describe('ratio settings reset scope', () => {
  after(() => {
    domWindow.close()
  })

  test('the confirm dialog spells out the reset scope and what stays untouched', async () => {
    const container = document.createElement('div')
    document.body.append(container)
    const root = createRoot(container)
    const queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
        mutations: { retry: false },
      },
    })

    await act(async () => {
      root.render(
        <QueryClientProvider client={queryClient}>
          <I18nextProvider i18n={i18n}>
            <RatioSettingsCard
              modelDefaults={modelDefaults}
              groupDefaults={groupDefaults}
              toolPricesDefault='{}'
              visibleTabs={['models']}
            />
          </I18nextProvider>
        </QueryClientProvider>
      )
    })

    const resetButton = [...container.querySelectorAll('button')].find(
      (button) => button.textContent?.includes('Reset prices')
    )
    assert.ok(resetButton, 'the reset button should be rendered')
    assert.ok(
      !(document.body.textContent ?? '').includes(en[RESET_SCOPE_KEY]),
      'the scope copy must not be shown until the operator asks to reset'
    )

    await act(async () => {
      resetButton.click()
      for (let tick = 0; tick < 12; tick++) await Promise.resolve()
    })

    const dialogText = document.body.textContent ?? ''
    assert.ok(
      dialogText.includes(en[RESET_SCOPE_KEY]),
      'the confirm dialog should state the full reset scope'
    )
    assert.ok(
      dialogText.includes('billing mode with its billing expression'),
      'the confirm dialog should mention the billing mode and expression'
    )
    assert.ok(
      dialogText.includes('Group ratios and usable groups are not affected.'),
      'the confirm dialog should say group ratios are untouched'
    )

    await act(async () => root.unmount())
    container.remove()
    queryClient.clear()
  })
})
