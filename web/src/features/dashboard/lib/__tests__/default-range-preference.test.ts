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
Object.defineProperty(globalThis, 'window', {
  configurable: true,
  value: domWindow,
})
Object.defineProperty(globalThis, 'localStorage', {
  configurable: true,
  value: domWindow.localStorage,
})

const {
  buildDefaultDashboardFilters,
  getSavedChartPreferences,
  saveChartPreferences,
} = await import('../filters')
const { DASHBOARD_CHART_PREFERENCES_STORAGE_KEY } =
  await import('../../constants')

function local(y: number, m: number, d: number, h = 0, min = 0) {
  return new Date(y, m - 1, d, h, min, 0, 0)
}

function storePayload(payload: Record<string, unknown>): void {
  localStorage.setItem(
    DASHBOARD_CHART_PREFERENCES_STORAGE_KEY,
    JSON.stringify(payload)
  )
}

afterEach(() => {
  localStorage.clear()
})

describe('default range preference', () => {
  test('legacy payload without a kind still resolves the rolling window', () => {
    storePayload({ defaultTimeRangeDays: 30 })
    const now = local(2026, 10, 18, 13, 45)

    const filters = buildDefaultDashboardFilters(
      getSavedChartPreferences(),
      now
    )

    assert.equal(
      filters.start_timestamp?.getTime(),
      local(2026, 9, 19).getTime()
    )
    assert.equal(filters.end_timestamp?.getTime(), now.getTime())
    assert.equal(filters.time_granularity, 'day')
  })

  test('month-to-date opens on the 1st of the current month', () => {
    storePayload({
      defaultTimeRangeDays: 30,
      defaultRangeKind: 'month-to-date',
    })
    const now = local(2026, 10, 18, 13, 45)

    const filters = buildDefaultDashboardFilters(
      getSavedChartPreferences(),
      now
    )

    assert.equal(
      filters.start_timestamp?.getTime(),
      local(2026, 10, 1).getTime()
    )
    assert.equal(filters.start_timestamp?.getHours(), 0)
    assert.equal(filters.end_timestamp?.getTime(), now.getTime())
  })

  test('month-to-date across a year boundary starts on January 1st', () => {
    storePayload({ defaultTimeRangeDays: 1, defaultRangeKind: 'month-to-date' })
    const now = local(2026, 1, 3, 9, 0)

    const filters = buildDefaultDashboardFilters(
      getSavedChartPreferences(),
      now
    )

    assert.equal(
      filters.start_timestamp?.getTime(),
      local(2026, 1, 1).getTime()
    )
  })

  test('last-month covers the whole previous month', () => {
    storePayload({ defaultTimeRangeDays: 30, defaultRangeKind: 'last-month' })
    const now = local(2026, 10, 18, 13, 45)

    const filters = buildDefaultDashboardFilters(
      getSavedChartPreferences(),
      now
    )

    assert.equal(
      filters.start_timestamp?.getTime(),
      local(2026, 9, 1).getTime()
    )
    assert.equal(
      filters.end_timestamp?.getTime(),
      local(2026, 10, 1).getTime() - 1
    )
  })

  test('an unknown kind falls back to the day window', () => {
    storePayload({ defaultTimeRangeDays: 7, defaultRangeKind: 'last-year' })
    const now = local(2026, 10, 18, 13, 45)

    assert.equal(getSavedChartPreferences().defaultRangeKind, 'days')
    const filters = buildDefaultDashboardFilters(
      getSavedChartPreferences(),
      now
    )
    assert.equal(
      filters.start_timestamp?.getTime(),
      local(2026, 10, 12).getTime()
    )
  })

  test('saving and reading round-trips the kind', () => {
    const base = getSavedChartPreferences()

    saveChartPreferences({
      ...base,
      defaultTimeRangeDays: 14,
      defaultRangeKind: 'last-month',
    })
    assert.equal(getSavedChartPreferences().defaultRangeKind, 'last-month')
    assert.equal(getSavedChartPreferences().defaultTimeRangeDays, 14)

    saveChartPreferences({
      ...base,
      defaultTimeRangeDays: 7,
      defaultRangeKind: 'days',
    })
    assert.equal(getSavedChartPreferences().defaultRangeKind, 'days')
    assert.equal(getSavedChartPreferences().defaultTimeRangeDays, 7)
  })

  test('a payload saved without a kind is upgraded to days on save', () => {
    storePayload({ defaultTimeRangeDays: 30 })
    saveChartPreferences(getSavedChartPreferences())

    const raw = JSON.parse(
      localStorage.getItem(DASHBOARD_CHART_PREFERENCES_STORAGE_KEY) ?? '{}'
    ) as { defaultRangeKind?: string }

    assert.equal(raw.defaultRangeKind, 'days')
  })

  test('the 29 to 30 migration still applies to the day window', () => {
    storePayload({ defaultTimeRangeDays: 29 })
    const now = local(2026, 10, 18, 13, 45)

    assert.equal(getSavedChartPreferences().defaultTimeRangeDays, 30)
    const filters = buildDefaultDashboardFilters(
      getSavedChartPreferences(),
      now
    )
    assert.equal(
      filters.start_timestamp?.getTime(),
      local(2026, 9, 19).getTime()
    )
  })
})
