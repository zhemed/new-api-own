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
import type React from 'react'

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
const { useCommonLogsColumns } = await import('../columns/common-logs-columns')
const { DetailsDialog } = await import('../dialogs/details-dialog')
type UsageLog = import('../../data/schema').UsageLog

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

async function resetDom(): Promise<void> {
  await act(async () => {
    for (const root of roots.splice(0)) root.unmount()
  })
  document.body.innerHTML = ''
}

async function renderNode(node: React.ReactNode): Promise<string> {
  await resetDom()
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
        <I18nextProvider i18n={i18n}>{node}</I18nextProvider>
      </QueryClientProvider>
    )
    for (let tick = 0; tick < 8; tick++) await Promise.resolve()
  })

  return document.body.textContent ?? ''
}

function exprB64(expr: string): string {
  return Buffer.from(expr, 'utf8').toString('base64')
}

const TWO_TIERS = 'tier("offpeak", p * 1 + c * 4) + tier("peak", p * 2 + c * 8)'

function tieredLog(expr: string, matchedTier: string): UsageLog {
  return {
    id: 1,
    type: 2,
    content: '',
    quota: 1000,
    other: JSON.stringify({
      billing_mode: 'tiered_expr',
      expr_b64: exprB64(expr),
      matched_tier: matchedTier,
      model_ratio: 0.5,
      completion_ratio: 4,
    }),
  } as unknown as UsageLog
}

let capturedColumns: Array<Record<string, unknown>> = []

function ColumnsProbe() {
  capturedColumns = useCommonLogsColumns(false) as unknown as Array<
    Record<string, unknown>
  >
  return null
}

async function renderDetailsCell(log: UsageLog): Promise<string> {
  await renderNode(<ColumnsProbe />)
  const detailsColumn = capturedColumns.find(
    (column) => column.accessorKey === 'content'
  )
  assert.ok(detailsColumn, 'the details column should exist')
  const Cell = detailsColumn.cell as unknown as React.ComponentType<{
    row: { original: UsageLog }
  }>
  return await renderNode(<Cell row={{ original: log }} />)
}

function countOccurrences(haystack: string, needle: string): number {
  return needle === '' ? 0 : haystack.split(needle).length - 1
}

describe('usage log tier label display', () => {
  test('the log list details segment shows the translated tier name', async () => {
    const text = await renderDetailsCell(tieredLog(TWO_TIERS, 'offpeak'))

    assert.ok(
      text.includes(zh['Off-peak hours']),
      'expected 空闲时段 in the segment'
    )
    assert.equal(
      text.includes('offpeak'),
      false,
      'the raw tier label must not leak into the log list'
    )
  })

  test('a custom tier name is shown exactly as written in the log list', async () => {
    const custom = 'tier("my_custom_tier", p * 1 + c * 4)'
    const text = await renderDetailsCell(tieredLog(custom, 'my_custom_tier'))

    assert.ok(text.includes('my_custom_tier'))
  })

  test('the details dialog shows the translated tier name for the matched tier', async () => {
    const text = await renderNode(
      <DetailsDialog
        log={tieredLog(TWO_TIERS, 'offpeak')}
        isAdmin={false}
        open
        onOpenChange={() => {}}
      />
    )

    assert.ok(
      text.includes(zh['Matched Tier']),
      'expected the tier field label'
    )
    assert.ok(
      text.includes(zh['Off-peak hours']),
      'expected 空闲时段 as the field value'
    )
    assert.equal(
      text.includes('offpeak'),
      false,
      'the raw tier label must not leak into the details dialog'
    )
  })

  test('the matched highlight still compares the raw tier labels', async () => {
    const rawMatch = await renderNode(
      <DetailsDialog
        log={tieredLog(TWO_TIERS, 'offpeak')}
        isAdmin={false}
        open
        onOpenChange={() => {}}
      />
    )
    const translatedMatch = await renderNode(
      <DetailsDialog
        log={tieredLog(TWO_TIERS, zh['Off-peak hours'])}
        isAdmin={false}
        open
        onOpenChange={() => {}}
      />
    )

    assert.ok(
      countOccurrences(rawMatch, zh['Matched']) > 0,
      'the raw label offpeak should highlight its tier'
    )
    assert.equal(
      countOccurrences(translatedMatch, zh['Matched']),
      0,
      'a translated label must never match a tier'
    )
  })
})
