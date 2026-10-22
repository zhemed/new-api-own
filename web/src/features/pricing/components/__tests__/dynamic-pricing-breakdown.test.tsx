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
const { DynamicPricingBreakdown } = await import('../dynamic-pricing-breakdown')

const { DEFAULT_CURRENCY_CONFIG, useSystemConfigStore } =
  await import('@/stores/system-config-store')
type CurrencyConfig = import('@/stores/system-config-store').CurrencyConfig

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: { en: { translation: {} } },
})

const reactTestGlobals = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean
}
reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = true

function setCurrency(currency: Partial<CurrencyConfig>): void {
  useSystemConfigStore.setState((state) => ({
    config: {
      ...state.config,
      currency: { ...DEFAULT_CURRENCY_CONFIG, ...currency },
    },
  }))
}

const roots: Array<{ unmount: () => void }> = []

after(async () => {
  i18n.removeResourceBundle('en', 'translation')
})

afterEach(async () => {
  await act(async () => {
    for (const root of roots.splice(0)) root.unmount()
  })
  document.body.innerHTML = ''
  setCurrency(DEFAULT_CURRENCY_CONFIG)
})

async function renderBreakdown(): Promise<string> {
  const container = document.createElement('div')
  document.body.append(container)
  const root = createRoot(container)
  roots.push(root)

  await act(async () => {
    root.render(
      <I18nextProvider i18n={i18n}>
        <DynamicPricingBreakdown billingExpr='tier("base", p * 2 + c * 4)' />
      </I18nextProvider>
    )
    for (let tick = 0; tick < 8; tick++) await Promise.resolve()
  })

  return container.textContent ?? ''
}

describe('dynamic pricing breakdown money cells', () => {
  test('tier prices use the shared symbol + space format', async () => {
    setCurrency(DEFAULT_CURRENCY_CONFIG)

    const text = await renderBreakdown()

    // the tier table renders its unit prices through the currency layer
    assert.match(text, /¥ \d/)
    assert.equal(text.includes('¥1'), false, 'missing space before the number')
  })

  test('CNY mode converts the same cells with the same spacing', async () => {
    setCurrency({ quotaDisplayType: 'CNY', usdExchangeRate: 7 })

    const text = await renderBreakdown()

    assert.match(text, /¥ \d/)
  })
})
