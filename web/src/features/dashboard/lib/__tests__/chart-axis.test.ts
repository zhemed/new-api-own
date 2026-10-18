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
const { granularityForRangeDays, rangeDaysOf } = await import('../filters')
const { axisLabelStride, MAX_AXIS_LABELS, timeAxisLabelConfig } =
  await import('../charts')

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

describe('granularity is derived from the range', () => {
  const cases: Array<[number, string]> = [
    [1, 'hour'],
    [2, 'day'],
    [7, 'day'],
    [14, 'day'],
    [30, 'day'],
    [31, 'day'],
    [60, 'week'],
  ]

  for (const [days, expected] of cases) {
    test(`${days} days buckets per ${expected}`, () => {
      assert.equal(granularityForRangeDays(days), expected)
    })
  }

  test('an unknown span falls back to day buckets', () => {
    assert.equal(granularityForRangeDays(Number.NaN), 'day')
  })
})

describe('bucket counts follow the range', () => {
  test('one day at hour granularity yields 24 buckets', () => {
    const start = local(2026, 1, 15)
    const now = local(2026, 1, 15, 23, 30)
    const buckets = buildTimeBuckets(
      { start_timestamp: SEC(start), end_timestamp: SEC(now) },
      granularityForRangeDays(1)
    )

    assert.equal(buckets.length, 24)
  })

  test('seven days at day granularity yields 7 buckets', () => {
    const start = local(2026, 1, 9)
    const now = local(2026, 1, 15, 13, 45)
    const buckets = buildTimeBuckets(
      { start_timestamp: SEC(start), end_timestamp: SEC(now) },
      granularityForRangeDays(7)
    )

    assert.equal(buckets.length, 7)
  })

  test('rangeDaysOf counts inclusive calendar days', () => {
    assert.equal(
      rangeDaysOf(local(2025, 12, 17), local(2026, 1, 15, 13, 45)),
      30
    )
    assert.equal(rangeDaysOf(local(2026, 1, 15), local(2026, 1, 15, 9)), 1)
    assert.equal(rangeDaysOf(undefined, local(2026, 1, 15)), undefined)
  })
})

describe('axis label thinning', () => {
  test('keeps the visible labels at or below the cap', () => {
    for (const count of [7, 24, 30, 31]) {
      const stride = axisLabelStride(count)
      const visible = Math.ceil(count / stride)
      assert.equal(
        visible <= MAX_AXIS_LABELS,
        true,
        `${count} buckets produced ${visible} labels`
      )
    }
    assert.equal(axisLabelStride(30), 3)
    assert.equal(axisLabelStride(24), 3)
    assert.equal(axisLabelStride(7), 1)
  })

  test('never blanks labels: thinning goes through dataFilter, not formatMethod', () => {
    for (const count of [1, 7, 24, 30, 31]) {
      const config = timeAxisLabelConfig(count) as {
        formatMethod?: unknown
        dataFilter: (data: unknown[]) => unknown[]
      }

      // A formatMethod would receive (value, datum, formatter) from vchart, so
      // index-based logic there silently blanks every label. Guard the contract.
      assert.equal(config.formatMethod, undefined)

      const items = Array.from({ length: count }, (_, i) => `label-${i}`)
      const visible = config.dataFilter(items)
      // a single bucket legitimately keeps its only label; anything wider must
      // still show more than one, otherwise the axis reads as empty
      assert.equal(
        visible.length >= Math.min(2, count),
        true,
        `${count} buckets rendered only ${visible.length} labels`
      )
      assert.equal(visible.length <= MAX_AXIS_LABELS, true)
      assert.equal(visible[0], 'label-0')
      // every surviving item is a real label, never an empty string
      assert.equal(
        visible.every((label) => typeof label === 'string' && label !== ''),
        true
      )
    }
  })

  test('keeps labels evenly spaced', () => {
    const config = timeAxisLabelConfig(30) as {
      dataFilter: (data: unknown[]) => unknown[]
    }
    const visible = config.dataFilter(
      Array.from({ length: 30 }, (_, i) => `d${i}`)
    ) as string[]

    assert.deepEqual(visible, [
      'd0',
      'd3',
      'd6',
      'd9',
      'd12',
      'd15',
      'd18',
      'd21',
      'd24',
      'd27',
    ])
  })

  test('the chart spec carries the thinned time axis', () => {
    const range = {
      start_timestamp: SEC(local(2026, 1, 15)),
      end_timestamp: SEC(local(2026, 1, 15, 23, 30)),
    }
    const processed = processChartData(
      [
        {
          created_at: SEC(local(2026, 1, 15, 10)),
          model_name: 'gpt-5',
          quota: 1000,
          count: 3,
          token_used: 20,
        },
      ],
      'hour',
      undefined,
      undefined,
      range
    )
    const axes = processed.spec_model_line.axes as Array<{
      orient: string
      label?: {
        formatMethod?: unknown
        dataFilter?: (data: unknown[]) => unknown[]
      }
    }>

    assert.equal(axes[0].orient, 'bottom')
    // the axis must never blank labels via formatMethod (vchart passes the
    // formatter, not the index, as the third argument)
    assert.equal(axes[0].label?.formatMethod, undefined)
    const labels = Array.from({ length: 24 }, (_, i) => `01-15 ${i}:00`)
    const visible = axes[0].label?.dataFilter?.(labels) ?? []
    assert.equal(visible.length <= MAX_AXIS_LABELS, true)
    assert.equal(visible[0], '01-15 0:00')
  })
})
