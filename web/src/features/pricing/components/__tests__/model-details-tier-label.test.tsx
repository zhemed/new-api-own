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
  'localStorage',
  'sessionStorage',
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
  'matchMedia',
  'customElements',
  'CSS',
  'getSelection',
  'IntersectionObserver',
  'PointerEvent',
  'MouseEvent',
  'KeyboardEvent',
  'DOMParser',
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
const { ModelDetailsContent } = await import('../model-details')
type PricingModel = import('../../types').PricingModel

const zh = (
  JSON.parse(
    readFileSync(
      new URL('../../../../i18n/locales/zh.json', import.meta.url),
      'utf8'
    )
  ) as { translation: Record<string, string> }
).translation

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'zh',
  resources: { zh: { translation: zh } },
})

const reactTestGlobals = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean
}
reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = true

const roots: Array<{ unmount: () => void }> = []

after(() => {
  domWindow.close()
})

afterEach(async () => {
  await act(async () => {
    for (const root of roots.splice(0)) root.unmount()
  })
  document.body.innerHTML = ''
})

function modelWith(expr: string): PricingModel {
  return {
    id: 1,
    model_name: 'deepseek-flash',
    quota_type: 0,
    model_ratio: 0.5,
    completion_ratio: 4,
    enable_groups: ['default'],
    group_ratio: { default: 1 },
    billing_mode: 'tiered_expr',
    billing_expr: expr,
  } as unknown as PricingModel
}

async function renderModelDetails(expr: string): Promise<string> {
  const container = document.createElement('div')
  document.body.append(container)
  const root = createRoot(container)
  roots.push(root)
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
          <ModelDetailsContent
            model={modelWith(expr)}
            groupRatio={{ default: 1 }}
            usableGroup={{ default: { desc: '', ratio: 1 } }}
            endpointMap={{}}
            autoGroups={[]}
            priceRate={1}
            usdExchangeRate={1}
            tokenUnit='M'
          />
        </I18nextProvider>
      </QueryClientProvider>
    )
    for (let tick = 0; tick < 12; tick++) await Promise.resolve()
  })

  return document.body.textContent ?? ''
}

describe('model details tier label display', () => {
  test('the tier column renders the translated tier names', async () => {
    const text = await renderModelDetails(
      'tier("offpeak", p * 1 + c * 4) + tier("peak", p * 2 + c * 8)'
    )

    assert.ok(
      text.includes(zh['Off-peak hours']),
      'expected 空闲时段 in the tier column'
    )
    assert.ok(
      text.includes(zh['Peak hours']),
      'expected 高峰时段 in the tier column'
    )
    assert.equal(
      text.includes('offpeak'),
      false,
      'the raw tier label must not leak into the price table'
    )
  })

  test('a custom tier name is shown exactly as written', async () => {
    const text = await renderModelDetails(
      'tier("my_custom_tier", p * 1 + c * 4) + tier("peak", p * 2 + c * 8)'
    )

    assert.ok(text.includes('my_custom_tier'))
  })

  test('an empty tier name falls back to Default', async () => {
    const text = await renderModelDetails(
      'tier("", p * 1 + c * 4) + tier("peak", p * 2 + c * 8)'
    )

    assert.ok(
      text.includes(zh['Default']),
      'expected 默认 for the empty tier name'
    )
  })
})
