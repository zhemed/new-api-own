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

import {
  formatCurrencyFromUSD,
  formatLocalCurrencyAmount,
  getCurrencyLabel,
  getCurrencySymbol,
} from '../currency'

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

// The product decision is "swap the symbol, keep the number": the RMB display
// is configured as a CUSTOM currency with symbol ¥ and rate 1, so every
// symbol-only surface must render the same amount with a different sign.
describe('currency symbol source', () => {
  test('the default configuration shows the dollar symbol', () => {
    setCurrency(DEFAULT_CURRENCY_CONFIG)

    assert.equal(getCurrencySymbol(), '$')
    assert.equal(getCurrencyLabel(), 'USD')
  })

  test('a custom ¥ currency replaces the hardcoded dollar sign', () => {
    setCurrency({
      quotaDisplayType: 'CUSTOM',
      customCurrencySymbol: '¥',
      customCurrencyExchangeRate: 1,
    })

    assert.equal(getCurrencySymbol(), '¥')
  })

  test('CNY mode reports both the label and the symbol', () => {
    setCurrency({ quotaDisplayType: 'CNY', usdExchangeRate: 7 })

    assert.equal(getCurrencySymbol(), '¥')
    assert.equal(getCurrencyLabel(), 'CNY')
  })

  test('token mode falls back to the dollar symbol for input prefixes', () => {
    setCurrency({ quotaDisplayType: 'TOKENS' })

    assert.equal(getCurrencySymbol(), '$')
    assert.equal(getCurrencyLabel(), 'Tokens')
  })
})

describe('symbol-only formatting keeps the amount untouched', () => {
  test('formatLocalCurrencyAmount never applies the exchange rate', () => {
    setCurrency({
      quotaDisplayType: 'CUSTOM',
      customCurrencySymbol: '¥',
      customCurrencyExchangeRate: 7.3,
    })

    assert.equal(formatLocalCurrencyAmount(10), '¥ 10')
  })

  test('formatCurrencyFromUSD still converts, so the two are not interchangeable', () => {
    setCurrency({
      quotaDisplayType: 'CUSTOM',
      customCurrencySymbol: '¥',
      customCurrencyExchangeRate: 7.3,
    })

    assert.equal(formatCurrencyFromUSD(10), '¥ 73')
  })

  test('CNY mode formats the localised symbol without conversion', () => {
    setCurrency({ quotaDisplayType: 'CNY', usdExchangeRate: 7 })

    assert.equal(formatLocalCurrencyAmount(10).includes('10'), true)
    assert.equal(formatLocalCurrencyAmount(10).includes('¥'), true)
  })
})
