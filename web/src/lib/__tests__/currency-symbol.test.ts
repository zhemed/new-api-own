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
  formatBillingCurrencyFromUSD,
  formatCurrencyFromUSD,
  formatLocalCurrencyAmount,
  formatQuotaWithCurrency,
  getCurrencyDisplay,
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

// Product decision: the panel is RMB-only in every display mode. The
// base-currency mode (API value 'USD') keeps its accounting semantics — code
// and rate untouched — but shows the ¥ sign, so no dollar sign reaches the UI.
describe('base currency display mode', () => {
  test('shows the RMB sign without touching the accounting unit', () => {
    setCurrency(DEFAULT_CURRENCY_CONFIG)

    const { config, meta } = getCurrencyDisplay()
    assert.equal(config.quotaDisplayType, 'USD')
    assert.equal(meta.kind, 'currency')
    assert.equal('currencyCode' in meta ? meta.currencyCode : '', 'USD')
    assert.equal('exchangeRate' in meta ? meta.exchangeRate : 0, 1)
  })

  test('reports the RMB symbol and label to the UI', () => {
    setCurrency(DEFAULT_CURRENCY_CONFIG)

    assert.equal(getCurrencySymbol(), '¥')
    assert.equal(getCurrencyLabel(), '¥')
  })

  test('never converts the amount, because the rate stays 1', () => {
    setCurrency(DEFAULT_CURRENCY_CONFIG)

    assert.equal(formatCurrencyFromUSD(10), '¥ 10')
    assert.equal(formatLocalCurrencyAmount(12.5), '¥ 12.5')
    assert.equal(formatQuotaWithCurrency(6_250_000), '¥ 12.5')
  })

  test('renders exactly like CUSTOM(¥, 1)', () => {
    setCurrency(DEFAULT_CURRENCY_CONFIG)
    const baseCurrencyMode = [
      formatCurrencyFromUSD(12.5),
      formatLocalCurrencyAmount(12.5),
      formatQuotaWithCurrency(6_250_000),
      getCurrencyLabel(),
    ]

    setCurrency({
      quotaDisplayType: 'CUSTOM',
      customCurrencySymbol: '¥',
      customCurrencyExchangeRate: 1,
    })
    const customMode = [
      formatCurrencyFromUSD(12.5),
      formatLocalCurrencyAmount(12.5),
      formatQuotaWithCurrency(6_250_000),
      getCurrencyLabel(),
    ]

    assert.deepEqual(baseCurrencyMode, customMode)
  })
})

describe('other display modes', () => {
  test('CNY mode still converts with the configured rate', () => {
    setCurrency({ quotaDisplayType: 'CNY', usdExchangeRate: 7 })

    assert.equal(getCurrencySymbol(), '¥')
    assert.equal(getCurrencyLabel(), 'CNY')
    assert.equal(formatCurrencyFromUSD(10), '¥70')
  })

  test('token mode keeps the RMB sign for input prefixes', () => {
    setCurrency({ quotaDisplayType: 'TOKENS' })

    assert.equal(getCurrencySymbol(), '¥')
    assert.equal(getCurrencyLabel(), 'Tokens')
  })

  test('billing amounts use the same sign in base-currency mode', () => {
    setCurrency(DEFAULT_CURRENCY_CONFIG)

    assert.equal(formatBillingCurrencyFromUSD(10), '¥ 10')
  })

  test('token mode still prices billing amounts in ¥, never as tokens', () => {
    setCurrency({ quotaDisplayType: 'TOKENS' })

    // The billing path never shows tokens, and now signs them in ¥ as well.
    assert.equal(formatBillingCurrencyFromUSD(10), '¥ 10')
    assert.equal(formatLocalCurrencyAmount(10), '¥ 10')
  })

  test('hiding the symbol leaves a bare number', () => {
    setCurrency(DEFAULT_CURRENCY_CONFIG)

    assert.equal(formatLocalCurrencyAmount(10, { showSymbol: false }), '10')
  })

  test('a custom non-RMB currency is still honoured', () => {
    setCurrency({
      quotaDisplayType: 'CUSTOM',
      customCurrencySymbol: '€',
      customCurrencyExchangeRate: 0.9,
    })

    assert.equal(getCurrencySymbol(), '€')
    assert.equal(formatLocalCurrencyAmount(10), '€ 10')
  })
})
