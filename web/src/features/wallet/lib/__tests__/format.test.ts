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

import { formatCreemPrice } from '../format'

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

describe('Creem price formatting', () => {
  test('the system currency follows the configured symbol', () => {
    // Base-currency mode shows ¥ as well (product decision, see lib/currency).
    setCurrency(DEFAULT_CURRENCY_CONFIG)
    assert.equal(formatCreemPrice(10, 'USD'), '¥10.00')

    setCurrency({
      quotaDisplayType: 'CUSTOM',
      customCurrencySymbol: '¥',
      customCurrencyExchangeRate: 1,
    })
    assert.equal(formatCreemPrice(10, 'USD'), '¥10.00')
  })

  test('the amount itself is never converted', () => {
    setCurrency({
      quotaDisplayType: 'CUSTOM',
      customCurrencySymbol: '¥',
      customCurrencyExchangeRate: 7.3,
    })

    assert.equal(formatCreemPrice(19.99, 'USD'), '¥19.99')
  })

  test('EUR keeps its own symbol because the gateway charges in euro', () => {
    setCurrency({
      quotaDisplayType: 'CUSTOM',
      customCurrencySymbol: '¥',
      customCurrencyExchangeRate: 1,
    })

    assert.equal(formatCreemPrice(10, 'EUR'), '€10.00')
  })
})
