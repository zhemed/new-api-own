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
import { afterEach, describe, test } from 'node:test'

import { Window } from 'happy-dom'

const domWindow = new Window()
const domGlobals = [
  'window',
  'document',
  'localStorage',
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

const { act, useState } = await import('react')
const { createRoot } = await import('react-dom/client')

const {
  resetChartPreferencesSnapshot,
  saveChartPreferences,
  useChartPreferences,
} = await import('../index')
const { subscribeChartPreferences } =
  await import('../chart-preferences-events')
const { rankingChartType } =
  await import('@/features/rankings/lib/chart-preference')
const { useInitialRankingPeriod } =
  await import('@/features/rankings/lib/use-initial-ranking-period')

const reactTestGlobals = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean
}
reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = true

const roots: Array<{ unmount: () => void }> = []

afterEach(async () => {
  await act(async () => {
    for (const root of roots.splice(0)) root.unmount()
  })
  document.body.innerHTML = ''
  localStorage.clear()
  resetChartPreferencesSnapshot()
})

interface Probe {
  renders: Array<Record<string, unknown>>
  bump: () => void
}

/** Renders a component that observes the hook, and can be re-rendered on cue. */
async function mountProbe(): Promise<Probe> {
  const seen: Array<Record<string, unknown>> = []
  const container = document.createElement('div')
  document.body.append(container)

  let forceRender: (() => void) | undefined

  function Probe() {
    const preferences = useChartPreferences()
    const period = useInitialRankingPeriod()
    seen.push({ preferences, period })
    return null
  }

  function Host() {
    const [, setTick] = useState(0)
    forceRender = () => setTick((value) => value + 1)
    return <Probe />
  }

  const root = createRoot(container)
  roots.push(root)

  await act(async () => {
    root.render(<Host />)
    for (let tick = 0; tick < 8; tick++) await Promise.resolve()
  })

  return {
    renders: seen,
    bump: () => forceRender?.(),
  }
}

describe('chart preferences store', () => {
  test('saving notifies subscribers exactly once per write', () => {
    let notified = 0
    const unsubscribe = subscribeChartPreferences(() => {
      notified += 1
    })

    saveChartPreferences({
      consumptionDistributionChart: 'area',
      modelAnalyticsChart: 'trend',
      defaultTimeRangeDays: 30,
      defaultTimeGranularity: 'day',
      defaultRangeKind: 'month-to-date',
    })

    assert.equal(notified, 1)
    unsubscribe()
  })

  test('the snapshot reference stays stable until a write happens', async () => {
    const probe = await mountProbe()

    await act(async () => {
      probe.bump()
      probe.bump()
      for (let tick = 0; tick < 8; tick++) await Promise.resolve()
    })

    const renders = probe.renders
    assert.equal(renders.length >= 2, true)
    const first = renders[0].preferences
    assert.equal(
      renders.at(-1)?.preferences,
      first,
      're-rendering must reuse the same snapshot object'
    )
  })

  test('saving re-renders consumers with the new preference', async () => {
    const probe = await mountProbe()

    await act(async () => {
      saveChartPreferences({
        consumptionDistributionChart: 'area',
        modelAnalyticsChart: 'trend',
        defaultTimeRangeDays: 30,
        defaultTimeGranularity: 'day',
        defaultRangeKind: 'month-to-date',
      })
      for (let tick = 0; tick < 8; tick++) await Promise.resolve()
    })

    const latest = probe.renders.at(-1)?.preferences as {
      consumptionDistributionChart: string
    }

    assert.equal(latest.consumptionDistributionChart, 'area')
    assert.equal(rankingChartType(latest.consumptionDistributionChart), 'area')
  })

  test('the rankings period is decided on mount and never reset by a save', async () => {
    const probe = await mountProbe()

    await act(async () => {
      saveChartPreferences({
        consumptionDistributionChart: 'bar',
        modelAnalyticsChart: 'trend',
        defaultTimeRangeDays: 30,
        defaultTimeGranularity: 'day',
        defaultRangeKind: 'month-to-date',
      })
      for (let tick = 0; tick < 8; tick++) await Promise.resolve()
    })

    // mounted while the preference was still "1 day" -> weekly default
    assert.equal(probe.renders[0].period, 'week')
    assert.equal(
      probe.renders.at(-1)?.period,
      'week',
      'a later save must not change the already-chosen period'
    )
  })
})
