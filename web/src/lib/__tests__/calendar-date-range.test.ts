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

const { getCalendarDateRange } = await import('../../lib/time')

/** Local-time constructor so the assertions stay timezone independent. */
function local(y: number, m: number, d: number, h = 0, min = 0) {
  return new Date(y, m - 1, d, h, min, 0, 0)
}

describe('getCalendarDateRange', () => {
  test('one day spans from local midnight today until now', () => {
    const now = local(2026, 1, 15, 13, 45)
    const { start, end } = getCalendarDateRange(1, now)

    assert.equal(start.getTime(), local(2026, 1, 15).getTime())
    assert.equal(start.getHours(), 0)
    assert.equal(start.getMinutes(), 0)
    assert.equal(end.getTime(), now.getTime())
  })

  test('thirty days covers thirty whole calendar days across a year boundary', () => {
    const now = local(2026, 1, 15, 13, 45)
    const { start, end } = getCalendarDateRange(30, now)

    // 15 Jan minus 29 days = 17 Dec 2025
    assert.equal(start.getTime(), local(2025, 12, 17).getTime())
    assert.equal(end.getTime(), now.getTime())
    assert.equal(
      (end.getTime() - start.getTime()) / 86_400_000 > 29,
      true,
      'range must cover 29 full days plus the current partial day'
    )
  })

  test('crosses a month boundary using calendar days, not 24h steps', () => {
    const now = local(2026, 3, 3, 10, 0)
    const { start } = getCalendarDateRange(7, now)

    // 3 Mar minus 6 days = 25 Feb (2026 is not a leap year)
    assert.equal(start.getTime(), local(2026, 2, 25).getTime())
  })

  test('a range starting at the first minute of a month keeps that month start', () => {
    const now = local(2026, 3, 1, 0, 5)
    const { start } = getCalendarDateRange(1, now)

    assert.equal(start.getTime(), local(2026, 3, 1).getTime())
  })

  test('treats a non-positive or invalid day count as one day', () => {
    const now = local(2026, 1, 15, 9, 0)
    const today = local(2026, 1, 15).getTime()

    assert.equal(getCalendarDateRange(0, now).start.getTime(), today)
    assert.equal(getCalendarDateRange(-3, now).start.getTime(), today)
    assert.equal(getCalendarDateRange(Number.NaN, now).start.getTime(), today)
  })
})
