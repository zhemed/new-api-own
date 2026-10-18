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

const { getSavedChartPreferences, migrateLegacyTimeRangeDays } =
  await import('../filters')
const { DASHBOARD_CHART_PREFERENCES_STORAGE_KEY, TIME_RANGE_PRESETS } =
  await import('../../constants')

const defaultDays = TIME_RANGE_PRESETS[0].days

function storePreferences(value: unknown): void {
  localStorage.setItem(
    DASHBOARD_CHART_PREFERENCES_STORAGE_KEY,
    JSON.stringify({ defaultTimeRangeDays: value })
  )
}

afterEach(() => {
  localStorage.clear()
})

describe('legacy time range migration', () => {
  test('a preference saved as 29 days now reads back as 30', () => {
    storePreferences(29)

    assert.equal(getSavedChartPreferences().defaultTimeRangeDays, 30)
  })

  test('a preference already saved as 30 stays 30', () => {
    storePreferences(30)

    assert.equal(getSavedChartPreferences().defaultTimeRangeDays, 30)
  })

  test('an unknown value still falls back to the default range', () => {
    storePreferences(5)

    assert.equal(getSavedChartPreferences().defaultTimeRangeDays, defaultDays)
  })

  test('with nothing saved the default range is used', () => {
    assert.equal(getSavedChartPreferences().defaultTimeRangeDays, defaultDays)
  })

  test('only the exact legacy value is translated', () => {
    assert.equal(migrateLegacyTimeRangeDays(29), 30)
    assert.equal(migrateLegacyTimeRangeDays(30), 30)
    assert.equal(migrateLegacyTimeRangeDays(7), 7)
    assert.equal(migrateLegacyTimeRangeDays(undefined), undefined)
    assert.equal(migrateLegacyTimeRangeDays('29'), '29')
  })
})
