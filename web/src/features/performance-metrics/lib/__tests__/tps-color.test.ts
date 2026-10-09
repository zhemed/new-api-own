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

const { getTpsLevel, getTpsTextClass } = await import('../format')

describe('throughput grading', () => {
  test('grades the interval boundaries inclusively at the lower bound', () => {
    // < 50 → slow; 50 itself already counts as medium (the "50 and above" case)
    assert.equal(getTpsLevel(49.9), 'slow')
    assert.equal(getTpsLevel(50), 'medium')
    assert.equal(getTpsLevel(99.9), 'medium')
    assert.equal(getTpsLevel(100), 'fast')
    assert.equal(getTpsLevel(173), 'fast')
  })

  test('treats a missing measurement as unknown instead of the slowest grade', () => {
    for (const value of [0, -1, Number.NaN, Number.POSITIVE_INFINITY]) {
      assert.equal(getTpsLevel(value), 'unknown')
      assert.equal(getTpsTextClass(value), 'text-muted-foreground')
    }
  })

  test('maps each grade to a distinct semantic colour class', () => {
    assert.equal(getTpsTextClass(20), 'text-red-600 dark:text-red-400')
    assert.equal(getTpsTextClass(75), 'text-amber-600 dark:text-amber-400')
    assert.equal(getTpsTextClass(150), 'text-emerald-600 dark:text-emerald-400')
  })
})
