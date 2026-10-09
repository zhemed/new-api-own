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
import { after, describe, test } from 'node:test'

import { Window } from 'happy-dom'
import type React from 'react'

const domWindow = new Window()
const domGlobals = [
  'window',
  'document',
  'navigator',
  'HTMLElement',
  'SVGElement',
  'Node',
  'Element',
  'Event',
  'CustomEvent',
  'MutationObserver',
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

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: {
    en: {
      translation: {
        Stream: 'Stream',
        'Non-stream': 'Non-stream',
      },
    },
  },
})

const { StreamTpsCell } = await import('../timing-metrics-cell')
const reactTestGlobals = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean
}
reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = true

const roots: Array<{ unmount: () => void }> = []

after(async () => {
  await act(async () => {
    for (const root of roots.splice(0)) root.unmount()
  })
})

async function renderCell(
  props: React.ComponentProps<typeof StreamTpsCell>
): Promise<HTMLElement> {
  const container = document.createElement('div')
  document.body.append(container)
  const root = createRoot(container)
  roots.push(root)

  await act(async () => {
    root.render(
      <I18nextProvider i18n={i18n}>
        <StreamTpsCell {...props} />
      </I18nextProvider>
    )
    await Promise.resolve()
  })

  return container
}

/** The t/s value element is the last span rendered by the cell. */
function tpsElement(container: HTMLElement): HTMLElement | null {
  const spans = [...container.querySelectorAll('span')]
  const last = spans.at(-1)
  return last instanceof HTMLElement ? last : null
}

describe('usage log stream throughput cell', () => {
  test('colours the per-request t/s with the shared throughput grading', async () => {
    const slow = tpsElement(
      await renderCell({ isStream: true, tokensPerSecond: 20 })
    )
    assert.equal(slow?.textContent, '20 t/s')
    assert.equal(slow?.className.includes('text-red-600'), true)

    const medium = tpsElement(
      await renderCell({ isStream: true, tokensPerSecond: 75 })
    )
    assert.equal(medium?.className.includes('text-amber-600'), true)

    const fast = tpsElement(
      await renderCell({ isStream: true, tokensPerSecond: 150 })
    )
    assert.equal(fast?.className.includes('text-emerald-600'), true)
  })

  test('keeps the neutral style when a request carries no throughput', async () => {
    const missing = tpsElement(
      await renderCell({ isStream: true, tokensPerSecond: null })
    )
    assert.equal(missing?.textContent, '—')
    assert.equal(missing?.className.includes('text-red-600'), false)
    assert.equal(missing?.className.includes('text-muted-foreground/60'), true)
  })
})
