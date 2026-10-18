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
import { describe, test } from 'node:test'

const { resolveRangePreset, granularityForRangeDays } =
  await import('../filters')
const { timeAxisLabelConfig, MAX_AXIS_LABELS } = await import('../charts')

function local(y: number, m: number, d: number, h = 0, min = 0) {
  return new Date(y, m - 1, d, h, min, 0, 0)
}

describe('natural month presets', () => {
  test('this month starts at the first day 00:00 and ends now', () => {
    const now = local(2026, 10, 18, 13, 45)
    const range = resolveRangePreset({ kind: 'month-to-date' }, now)

    assert.equal(range.start.getTime(), local(2026, 10, 1).getTime())
    assert.equal(range.start.getHours(), 0)
    assert.equal(range.end.getTime(), now.getTime())
    assert.equal(range.days, 18)
  })

  test('this month on the first day keeps that first day', () => {
    const now = local(2026, 11, 1, 0, 5)
    const range = resolveRangePreset({ kind: 'month-to-date' }, now)

    assert.equal(range.start.getTime(), local(2026, 11, 1).getTime())
    assert.equal(range.days, 1)
  })

  test('this month across a year boundary starts on January 1st', () => {
    const now = local(2026, 1, 3, 9, 0)
    const range = resolveRangePreset({ kind: 'month-to-date' }, now)

    assert.equal(range.start.getTime(), local(2026, 1, 1).getTime())
    assert.equal(range.days, 3)
  })

  test('last month covers the whole previous month (30-day month)', () => {
    const now = local(2026, 10, 18, 13, 45)
    const range = resolveRangePreset({ kind: 'last-month' }, now)

    assert.equal(range.start.getTime(), local(2026, 9, 1).getTime())
    // last instant of September
    assert.equal(range.end.getTime(), local(2026, 10, 1).getTime() - 1)
    assert.equal(new Date(range.end).getDate(), 30)
    assert.equal(new Date(range.end).getHours(), 23)
    assert.equal(range.days, 30)
  })

  test('last month across a year boundary is December', () => {
    const now = local(2026, 1, 10, 8, 0)
    const range = resolveRangePreset({ kind: 'last-month' }, now)

    assert.equal(range.start.getTime(), local(2025, 12, 1).getTime())
    assert.equal(new Date(range.end).getMonth(), 11)
    assert.equal(range.days, 31)
  })

  test('last month handles February in a non-leap year', () => {
    const now = local(2026, 3, 5, 12, 0)
    const range = resolveRangePreset({ kind: 'last-month' }, now)

    assert.equal(range.start.getTime(), local(2026, 2, 1).getTime())
    assert.equal(range.days, 28)
    assert.equal(new Date(range.end).getDate(), 28)
  })
})

describe('day presets keep the calendar-range semantics', () => {
  test('a 30-day preset still starts at 00:00 of the first day', () => {
    const now = local(2026, 10, 18, 13, 45)
    const range = resolveRangePreset({ kind: 'days', days: 30 }, now)

    assert.equal(range.start.getTime(), local(2026, 9, 19).getTime())
    assert.equal(range.end.getTime(), now.getTime())
    assert.equal(range.days, 30)
  })

  test('granularity follows the resolved span', () => {
    const monthToDate = resolveRangePreset(
      { kind: 'month-to-date' },
      local(2026, 10, 18, 13, 45)
    )
    const lastMonth = resolveRangePreset(
      { kind: 'last-month' },
      local(2026, 10, 18, 13, 45)
    )

    assert.equal(granularityForRangeDays(monthToDate.days), 'day')
    assert.equal(granularityForRangeDays(lastMonth.days), 'day')
  })
})

describe('time axis keeps its last label', () => {
  for (const count of [24, 30, 31]) {
    test(`${count} buckets keep the first and the last label`, () => {
      const config = timeAxisLabelConfig(count) as {
        dataFilter: (data: unknown[]) => unknown[]
      }
      const labels = Array.from({ length: count }, (_, i) => `d${i}`)
      const visible = config.dataFilter(labels) as string[]

      assert.equal(visible[0], 'd0')
      assert.equal(visible.at(-1), `d${count - 1}`)
      assert.equal(visible.length <= MAX_AXIS_LABELS, true)
      assert.equal(visible.length >= 2, true)
    })
  }

  test('an evenly divisible bucket count does not duplicate the end', () => {
    const config = timeAxisLabelConfig(20) as {
      dataFilter: (data: unknown[]) => unknown[]
    }
    // stride 2 -> 0,2,...,18 already ends on the last index
    const visible = config.dataFilter(
      Array.from({ length: 20 }, (_, i) => `d${i}`)
    ) as string[]

    assert.equal(visible.at(-1), 'd19')
    assert.equal(new Set(visible).size, visible.length)
  })
})
