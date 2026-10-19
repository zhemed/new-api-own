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
import { readFileSync } from 'node:fs'
import { describe, test } from 'node:test'

/**
 * Both chart headers print "Total: <value>". The value is a headline number,
 * so it must not inherit the muted colour of its label.
 *
 * These are source-contract assertions rather than a rendered component test:
 * both panels mount VChart, which needs a real canvas 2D context that
 * happy-dom does not provide (and installing `canvas` is off-limits here).
 */
const HEADERS = [
  new URL('../consumption-distribution-chart.tsx', import.meta.url).pathname,
  new URL('../model-charts.tsx', import.meta.url).pathname,
]

describe('chart header total emphasis', () => {
  for (const file of HEADERS) {
    const source = readFileSync(file, 'utf8')
    const name = file.split('/').pop()

    test(`${name}: the total value is wrapped in its own emphasised span`, () => {
      assert.equal(
        source.includes("data-testid='chart-total-value'"),
        true,
        'the value needs its own element'
      )
      assert.equal(source.includes('text-foreground'), true)
      assert.equal(source.includes('tabular-nums'), true)
      assert.equal(source.includes('font-medium'), true)
    })

    test(`${name}: the label wrapper keeps its muted styling`, () => {
      // the muted class must still be on the outer span that renders "Total:"
      const outer =
        /<span className='text-muted-foreground text-xs'>[\s\S]*?Total:/.test(
          source
        )
      assert.equal(outer, true, 'the "Total:" label should stay muted')
      // font size stays text-xs everywhere in the header
      assert.equal(
        source.includes('text-foreground font-medium tabular-nums'),
        true
      )
      assert.equal(
        /text-foreground[^']*text-(sm|base|lg)/.test(source),
        false,
        'the value must not change the header font size'
      )
    })
  }

  test('tooltip and ranking totals keep their muted treatment', () => {
    const charts = readFileSync(
      new URL('../../../lib/charts.ts', import.meta.url).pathname,
      'utf8'
    )
    // tooltip rows are plain values in the tooltip body, never the header span
    assert.equal(charts.includes("data-testid='chart-total-value'"), false)
  })
})
