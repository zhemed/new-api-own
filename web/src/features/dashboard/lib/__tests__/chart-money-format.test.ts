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

import {
  DEFAULT_CURRENCY_CONFIG,
  useSystemConfigStore,
  type CurrencyConfig,
} from '@/stores/system-config-store'

const { processUserChartData } = await import('../charts')
const { formatCurrencyFromUSD } = await import('@/lib/currency')

function setCurrency(currency: Partial<CurrencyConfig>): void {
  useSystemConfigStore.setState((state) => ({
    config: {
      ...state.config,
      currency: { ...DEFAULT_CURRENCY_CONFIG, ...currency },
    },
  }))
}

afterEach(() => {
  setCurrency(DEFAULT_CURRENCY_CONFIG)
})

interface ChartLabelSpec {
  label?: { formatMethod?: (value: number) => string }
  title?: { subtext?: string }
  tooltip?: {
    mark?: {
      content?: Array<{ value?: (datum: Record<string, unknown>) => string }>
    }
  }
}

function buildChartLabels() {
  const nowSeconds = Math.floor(Date.now() / 1000)
  const processed = processUserChartData(
    [
      { created_at: nowSeconds, username: 'alice', quota: 500_000, count: 1 },
      { created_at: nowSeconds, username: 'bob', quota: 250_000, count: 1 },
    ],
    'day',
    undefined,
    10
  )

  return processed.spec_user_rank as unknown as ChartLabelSpec
}

describe('user chart money labels', () => {
  test('bar labels use the shared symbol + space format', () => {
    setCurrency(DEFAULT_CURRENCY_CONFIG)
    const spec = buildChartLabels()

    const label = spec.label?.formatMethod?.(500_000)

    assert.match(String(label), /^¥ \d/)
  })

  test('the tooltip and total agree with the currency layer', () => {
    setCurrency(DEFAULT_CURRENCY_CONFIG)
    const spec = buildChartLabels()

    const tooltipValue = spec.tooltip?.mark?.content?.[0]?.value?.({
      rawQuota: 500_000,
      User: 'alice',
    })

    // 500,000 quota at the default quotaPerUnit is 1 base unit.
    assert.equal(formatCurrencyFromUSD(1), '¥ 1')
    assert.match(String(tooltipValue), /^¥ \d/)
    assert.equal(
      Number(String(tooltipValue).replace('¥ ', '')),
      Number(formatCurrencyFromUSD(1).replace('¥ ', ''))
    )
    assert.match(String(spec.title?.subtext), /¥ \d/)
  })

  test('CNY mode keeps converting while using the same spacing', () => {
    setCurrency({ quotaDisplayType: 'CNY', usdExchangeRate: 7 })
    const spec = buildChartLabels()

    const label = String(spec.label?.formatMethod?.(500_000))

    assert.match(label, /^¥ \d/)
    // 1 unit × 7 = 7, and the chart keeps two decimals for its labels.
    assert.equal(Number(label.replace('¥ ', '')), 7)
  })
})
