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

const { processChartData } = await import('../charts')

function local(y: number, m: number, d: number, h = 0) {
  return new Date(y, m - 1, d, h, 0, 0, 0)
}

const SEC = (date: Date) => Math.floor(date.getTime() / 1000)

type Value = { Time: string; Model: string; Usage: number; rawTokens: number }
type Spec = {
  data: Array<{ values: Value[] }>
  yField: string
  axes: Array<{
    orient: string
    label?: { formatMethod: (value: number) => string }
  }>
  tooltip: {
    mark: { content: Array<{ value: (d: Record<string, unknown>) => unknown }> }
  }
}

/** Two models in one hour, deliberately different token counts. */
function buildChart() {
  const hour = local(2026, 1, 15, 10)
  return processChartData(
    [
      {
        created_at: SEC(hour),
        model_name: 'gpt-5',
        quota: 500,
        count: 2,
        token_used: 1_682_465,
      },
      {
        created_at: SEC(hour),
        model_name: 'claude',
        quota: 900,
        count: 1,
        token_used: 1_000,
      },
    ],
    'hour',
    undefined,
    undefined,
    {
      start_timestamp: SEC(hour),
      end_timestamp: SEC(local(2026, 1, 15, 10)),
    }
  )
}

describe('consumption distribution reports tokens', () => {
  test('bar and area values come from token_used, not quota', () => {
    const processed = buildChart()
    const bar = processed.spec_line as unknown as Spec
    const area = processed.spec_area as unknown as Spec

    assert.equal(bar.yField, 'Usage')
    const barValues = bar.data[0].values
    const byModel = new Map(barValues.map((v) => [v.Model, v]))
    // quota was 900 vs 500, but tokens are 1_682_465 vs 1_000 — tokens win
    assert.equal(byModel.get('gpt-5')?.Usage, 1_682_465)
    assert.equal(byModel.get('claude')?.Usage, 1_000)

    const areaTotals = new Map<string, number>()
    for (const v of area.data[0].values) {
      areaTotals.set(v.Model, (areaTotals.get(v.Model) ?? 0) + v.Usage)
    }
    assert.equal(areaTotals.get('gpt-5'), 1_682_465)
    assert.equal(areaTotals.get('claude'), 1_000)
  })

  test('total, axis and tooltips are token formatted and carry no currency symbol', () => {
    const processed = buildChart()
    const bar = processed.spec_line as unknown as Spec

    // 1_682_465 + 1_000 = 1_683_465 -> 1.68M
    assert.equal(processed.totalQuotaDisplay, '1.68M')
    assert.equal(processed.totalQuotaDisplay.includes('$'), false)

    const axisFormatter = bar.axes.find((axis) => axis.orient === 'left')?.label
      ?.formatMethod
    assert.notEqual(axisFormatter, undefined)
    assert.equal(axisFormatter?.(1_682_465), '1.68M')
    assert.equal(axisFormatter?.(42), '42')
    assert.equal(axisFormatter?.(0), '0')

    const tooltipValue = bar.tooltip.mark.content[0].value
    assert.equal(tooltipValue({ rawTokens: 1_682_465 }), '1.68M')
    assert.equal(
      String(tooltipValue({ rawTokens: 1_682_465 })).includes('$'),
      false
    )
  })

  test('a row without token_used counts as zero instead of breaking', () => {
    const hour = local(2026, 1, 15, 11)
    const processed = processChartData(
      [
        {
          created_at: SEC(hour),
          model_name: 'gpt-5',
          quota: 500,
          count: 2,
        },
      ],
      'hour',
      undefined,
      undefined,
      {
        start_timestamp: SEC(hour),
        end_timestamp: SEC(local(2026, 1, 15, 11)),
      }
    )
    const bar = processed.spec_line as unknown as Spec
    const values = bar.data[0].values

    assert.equal(values.length > 0, true)
    assert.equal(
      values.every((v) => v.Usage === 0),
      true
    )
    assert.equal(processed.totalQuotaDisplay, '0')
    assert.equal(bar.tooltip.mark.content[0].value({}), '0')
  })
})
