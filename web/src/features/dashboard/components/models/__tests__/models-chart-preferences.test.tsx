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
const { ModelsChartPreferences } = await import('../models-chart-preferences')

const en = (
  JSON.parse(
    readFileSync(
      new URL('../../../../../i18n/locales/en.json', import.meta.url),
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
  document.body.innerHTML = ''
})

function findButton(label: string): HTMLButtonElement {
  for (const button of document.querySelectorAll('button')) {
    if ((button.textContent ?? '').includes(label)) {
      return button as HTMLButtonElement
    }
  }
  throw new Error(`button not found: ${label}`)
}

async function openPreferences(): Promise<HTMLElement> {
  const container = document.createElement('div')
  document.body.append(container)
  const root = createRoot(container)
  roots.push(root)

  await act(async () => {
    root.render(
      <I18nextProvider i18n={i18n}>
        <ModelsChartPreferences
          preferences={{
            consumptionDistributionChart: 'bar',
            modelAnalyticsChart: 'trend',
            defaultTimeRangeDays: 30,
            defaultTimeGranularity: 'day',
          }}
          onPreferencesChange={() => {}}
        />
      </I18nextProvider>
    )
    for (let tick = 0; tick < 8; tick++) await Promise.resolve()
  })

  await act(async () => {
    findButton(en['Preferences']).click()
    for (let tick = 0; tick < 8; tick++) await Promise.resolve()
  })

  return container
}

/** The range <select> only renders its options once opened. */
async function openRangeSelect(): Promise<void> {
  const trigger = document.querySelector('#default-time-range')
  assert.notEqual(trigger, null)
  await act(async () => {
    ;(trigger as HTMLElement).click()
    for (let tick = 0; tick < 8; tick++) await Promise.resolve()
  })
}

describe('models chart preferences', () => {
  test('the default range offers natural months next to day windows', async () => {
    await openPreferences()
    await openRangeSelect()

    const text = document.body.textContent ?? ''
    for (const label of [
      '1 Day',
      '7 Days',
      '14 Days',
      '30 Days',
      en['This Month'],
      en['Last Month'],
    ]) {
      assert.equal(text.includes(label), true, `missing range option: ${label}`)
    }
  })

  test('a natural-month preference is shown as the selected range', async () => {
    const container = document.createElement('div')
    document.body.append(container)
    const root = createRoot(container)
    roots.push(root)

    await act(async () => {
      root.render(
        <I18nextProvider i18n={i18n}>
          <ModelsChartPreferences
            preferences={{
              consumptionDistributionChart: 'bar',
              modelAnalyticsChart: 'trend',
              defaultTimeRangeDays: 30,
              defaultTimeGranularity: 'day',
              defaultRangeKind: 'last-month',
            }}
            onPreferencesChange={() => {}}
          />
        </I18nextProvider>
      )
      for (let tick = 0; tick < 8; tick++) await Promise.resolve()
    })

    await act(async () => {
      findButton(en['Preferences']).click()
      for (let tick = 0; tick < 8; tick++) await Promise.resolve()
    })

    const trigger = document.querySelector('#default-time-range')
    assert.notEqual(trigger, null)
    assert.equal(
      (trigger?.textContent ?? '').includes(en['Last Month']),
      true,
      `trigger shows "${trigger?.textContent}" instead of the saved kind`
    )
  })
})
