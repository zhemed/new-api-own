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

const { buildTimeBuckets, toBucketStart, processChartData } =
  await import('../charts')
const { TIME_RANGE_PRESETS, TIME_RANGE_BY_GRANULARITY } =
  await import('../../constants')

function local(y: number, m: number, d: number, h = 0, min = 0) {
  return new Date(y, m - 1, d, h, min, 0, 0)
}

const SEC = (date: Date) => Math.floor(date.getTime() / 1000)

describe('toBucketStart', () => {
  test('floors hours to the top of the hour', () => {
    const bucket = toBucketStart(local(2026, 1, 15, 13, 45).getTime(), 'hour')
    assert.equal(bucket, local(2026, 1, 15, 13).getTime())
  })

  test('floors days to local midnight', () => {
    const bucket = toBucketStart(local(2026, 1, 15, 13, 45).getTime(), 'day')
    assert.equal(bucket, local(2026, 1, 15).getTime())
  })

  test('floors weeks to Monday midnight', () => {
    // 2026-01-15 is a Thursday -> its week starts on Monday 2026-01-12
    const bucket = toBucketStart(local(2026, 1, 15, 13, 45).getTime(), 'week')
    assert.equal(bucket, local(2026, 1, 12).getTime())
    assert.equal(new Date(bucket).getDay(), 1)
  })
})

describe('buildTimeBuckets', () => {
  test('starts at the range start bucket and ends at the bucket holding now', () => {
    const start = local(2026, 1, 15, 0, 0)
    const now = local(2026, 1, 15, 3, 20)
    const buckets = buildTimeBuckets(
      { start_timestamp: SEC(start), end_timestamp: SEC(now) },
      'hour'
    )

    assert.equal(buckets.length, 4)
    assert.equal(buckets[0], local(2026, 1, 15, 0).getTime())
    assert.equal(buckets.at(-1), local(2026, 1, 15, 3).getTime())
  })

  test('covers thirty calendar days one bucket per day', () => {
    const start = local(2025, 12, 17)
    const now = local(2026, 1, 15, 13, 45)
    const buckets = buildTimeBuckets(
      { start_timestamp: SEC(start), end_timestamp: SEC(now) },
      'day'
    )

    assert.equal(buckets.length, 30)
    assert.equal(buckets[0], start.getTime())
    assert.equal(buckets.at(-1), local(2026, 1, 15).getTime())
  })

  test('the last bucket follows now instead of the newest data point', () => {
    const start = local(2026, 1, 15, 0, 0)
    const earlier = buildTimeBuckets(
      {
        start_timestamp: SEC(start),
        end_timestamp: SEC(local(2026, 1, 15, 9)),
      },
      'hour'
    )
    const later = buildTimeBuckets(
      {
        start_timestamp: SEC(start),
        end_timestamp: SEC(local(2026, 1, 15, 11)),
      },
      'hour'
    )

    assert.equal(earlier.at(-1), local(2026, 1, 15, 9).getTime())
    assert.equal(later.at(-1), local(2026, 1, 15, 11).getTime())
  })

  test('steps weeks on Monday boundaries', () => {
    const buckets = buildTimeBuckets(
      {
        start_timestamp: SEC(local(2026, 1, 12)),
        end_timestamp: SEC(local(2026, 2, 2, 12)),
      },
      'week'
    )

    assert.equal(buckets.length, 4)
    assert.equal(buckets[0], local(2026, 1, 12).getTime())
    assert.equal(buckets.at(-1), local(2026, 2, 2).getTime())
  })
})

describe('processChartData axis', () => {
  const data = [
    {
      // 2026-01-15 10:00 local, one hour of usage
      created_at: SEC(local(2026, 1, 15, 10)),
      model_name: 'gpt-5',
      quota: 1000,
      count: 3,
      token_used: 20,
    },
  ]

  test('axis spans the whole range and pads empty buckets with zero', () => {
    const range = {
      start_timestamp: SEC(local(2026, 1, 15, 8)),
      end_timestamp: SEC(local(2026, 1, 15, 12, 30)),
    }
    const processed = processChartData(
      data,
      'hour',
      undefined,
      undefined,
      range
    )
    const values = processed.spec_model_line.data[0].values as Array<{
      Time: string
      Count: number
    }>
    const times = [...new Set(values.map((v) => v.Time))]

    assert.deepEqual(times, [
      '01-15 08:00',
      '01-15 09:00',
      '01-15 10:00',
      '01-15 11:00',
      '01-15 12:00',
    ])
    // the single row lands in its own bucket, the rest are zero-filled
    const byTime = new Map(values.map((v) => [v.Time, v.Count]))
    assert.equal(byTime.get('01-15 10:00'), 3)
    assert.equal(byTime.get('01-15 08:00'), 0)
    assert.equal(byTime.get('01-15 12:00'), 0)
  })

  test('without a range the axis still ends at the newest data point', () => {
    const processed = processChartData(data, 'hour')
    const values = processed.spec_model_line.data[0].values as Array<{
      Time: string
    }>
    const times = [...new Set(values.map((v) => v.Time))]

    assert.equal(times.at(-1), '01-15 10:00')
  })
})

describe('time range presets', () => {
  test('the long preset is 30 days, not 29', () => {
    const longest = TIME_RANGE_PRESETS.at(-1)
    assert.deepEqual(longest, { label: '30 Days', days: 30 })
    assert.deepEqual(
      TIME_RANGE_PRESETS.map((preset) => preset.days),
      [1, 7, 14, 30]
    )
  })

  test('every granularity pairs with a range that fits the axis cap', () => {
    const widest = Math.max(...Object.values(TIME_RANGE_BY_GRANULARITY))
    const hourBuckets = widest * 24
    assert.equal(hourBuckets < 1000, true)
  })
})
