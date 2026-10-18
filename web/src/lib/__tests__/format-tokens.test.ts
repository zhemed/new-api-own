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

const { formatTokens } = await import('../format')

describe('formatTokens units', () => {
  const cases: Array<[number, string, string]> = [
    [0, '-', 'zero has no magnitude'],
    [999, '999', 'below one thousand stays literal'],
    [1000, '1.0K', 'K keeps one decimal'],
    [999_999, '1.0M', 'K rounding carries into M (not 1000.0K)'],
    [1_000_000, '1.00M', 'M keeps two decimals'],
    [1_500_000, '1.50M', 'M keeps trailing zeros'],
    [999_999_999, '1.00B', 'M rounding carries into B (not 1000.00M)'],
    [1_000_000_000, '1.00B', 'billions use B'],
    [1_234_567_890, '1.23B', 'B rounds to two decimals'],
    [999_999_999_999, '1.00T', 'B rounding carries into T (not 1000.00B)'],
    [1e12, '1.00T', 'trillions use T'],
    [1.5e12, '1.50T', 'T keeps two decimals'],
  ]

  for (const [value, expected, why] of cases) {
    test(`${value} renders ${expected} — ${why}`, () => {
      assert.equal(formatTokens(value), expected)
    })
  }
})

describe('formatTokens edge inputs', () => {
  test('non-finite values render as a dash instead of NaN text', () => {
    assert.equal(formatTokens(Number.NaN), '-')
    assert.equal(formatTokens(Number.POSITIVE_INFINITY), '-')
    assert.equal(formatTokens(Number.NEGATIVE_INFINITY), '-')
  })

  test('negative values keep their sign and magnitude', () => {
    assert.equal(formatTokens(-5), '-5')
    assert.equal(formatTokens(-5_000), '-5.0K')
    assert.equal(formatTokens(-1_234_567_890), '-1.23B')
  })

  test('values just under a unit boundary round up, not down', () => {
    assert.equal(formatTokens(999_949), '999.9K')
    assert.equal(formatTokens(999_950), '1.0M')
    assert.equal(formatTokens(999_999_999_999), '1.00T')
  })
})
