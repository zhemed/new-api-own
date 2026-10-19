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

import { MAX_AXIS_LABELS } from '@/features/dashboard/lib/charts'

import { rankingLabelFilter, rankingLabelIndexes } from '../axis-labels'

describe('ranking axis label indexes', () => {
  test('a full 30-day month keeps at most ten labels, both ends included', () => {
    const indexes = rankingLabelIndexes(30)

    assert.equal(indexes.length <= MAX_AXIS_LABELS, true)
    assert.equal(indexes[0], 0, 'the first bucket must be labelled')
    assert.equal(indexes.at(-1), 29, 'the last bucket (today) must be labelled')
  })

  test('a 30-bucket axis is evenly strided apart from the final gap', () => {
    const indexes = rankingLabelIndexes(30)
    const gaps = indexes
      .slice(1)
      .map((value, position) => value - indexes[position])

    // every gap but the last equals the stride
    const stride = gaps[0] ?? 0
    assert.equal(
      gaps.slice(0, -1).every((gap) => gap === stride),
      true,
      `uneven gaps: ${gaps.join(',')}`
    )
    // appending the last bucket may stretch or shrink the final gap
    const finalGap = gaps.at(-1) ?? 0
    assert.equal(finalGap >= 1 && finalGap <= stride * 2, true)
  })

  test('a 12-bucket year axis stays under the cap and keeps both ends', () => {
    const indexes = rankingLabelIndexes(12)

    assert.equal(indexes.length <= MAX_AXIS_LABELS, true)
    assert.equal(indexes[0], 0)
    assert.equal(indexes.at(-1), 11)
  })

  test('every bucket count up to 60 respects the cap and both ends', () => {
    for (let count = 2; count <= 60; count++) {
      const indexes = rankingLabelIndexes(count)

      assert.equal(
        indexes.length <= MAX_AXIS_LABELS,
        true,
        `${count} buckets produced ${indexes.length} labels`
      )
      assert.equal(indexes[0], 0, `${count} buckets lost the first label`)
      assert.equal(
        indexes.at(-1),
        count - 1,
        `${count} buckets lost the last label`
      )
    }
  })

  test('small axes keep every bucket', () => {
    assert.deepEqual(rankingLabelIndexes(1), [0])
    assert.deepEqual(rankingLabelIndexes(7), [0, 1, 2, 3, 4, 5, 6])
  })

  test('a degenerate bucket count yields no labels', () => {
    assert.deepEqual(rankingLabelIndexes(0), [])
    assert.deepEqual(rankingLabelIndexes(Number.NaN), [])
  })
})

describe('ranking axis label filter', () => {
  test('the dataFilter keeps exactly the strided axis items', () => {
    const items = Array.from({ length: 30 }, (_, index) => `label-${index}`)
    const kept = rankingLabelFilter(30)(items) as string[]
    const indexes = rankingLabelIndexes(30)

    assert.deepEqual(
      kept,
      indexes.map((index) => `label-${index}`)
    )
    assert.equal(kept[0], 'label-0')
    assert.equal(kept.at(-1), 'label-29')
  })
})
