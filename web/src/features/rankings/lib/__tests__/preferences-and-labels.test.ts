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

import {
  CONSUMPTION_DISTRIBUTION_CHART_OPTIONS,
  DEFAULT_DASHBOARD_CHART_PREFERENCES,
} from '@/features/dashboard/constants'
import { toIntlLocale } from '@/i18n/languages'

import {
  defaultRankingPeriod,
  rankingChartShape,
  rankingChartType,
} from '../chart-preference'
import { formatRankingBucketLabel } from '../format'

/** Local noon on 26 September 2026, so the label is stable in any timezone. */
const TS = new Date(2026, 8, 26, 12, 0, 0).toISOString()

describe('dashboard chart preference defaults', () => {
  test('the consumption distribution opens as an area chart', () => {
    assert.equal(
      DEFAULT_DASHBOARD_CHART_PREFERENCES.consumptionDistributionChart,
      'area'
    )
  })

  test('area comes before bar in the view switcher', () => {
    assert.deepEqual(
      CONSUMPTION_DISTRIBUTION_CHART_OPTIONS.map((option) => option.value),
      ['area', 'bar']
    )
  })
})

describe('rankings follow the saved preference', () => {
  test('an area preference renders a stacked area chart', () => {
    assert.equal(rankingChartType('area'), 'area')
  })

  test('any other preference keeps the bar chart', () => {
    assert.equal(rankingChartType('bar'), 'bar')
    assert.equal(rankingChartType(undefined), 'bar')
  })

  test('the models chart spec stays stacked and uses the preference type', () => {
    // VChart cannot render without a canvas, so the spec wiring is asserted at
    // the source level: the type comes from the preference and stacking is on.
    const source = readFileSync(
      new URL('../../components/models-section.tsx', import.meta.url).pathname,
      'utf8'
    )

    assert.match(source, /type: chartType,/)
    assert.match(source, /stack: true,/)
    assert.match(
      source,
      /rankingChartType\(\s*preferences\.consumptionDistributionChart\s*\)/
    )
  })

  test('an area preference produces a stacked area shape without bar padding', () => {
    const shape = rankingChartShape('area')

    assert.equal(shape.type, 'area')
    assert.equal(shape.stack, true)
    assert.equal(
      Object.hasOwn(shape, 'paddingInner'),
      false,
      'paddingInner only affects bar geometry'
    )
  })

  test('a bar preference produces a stacked bar shape with padding', () => {
    const shape = rankingChartShape('bar')

    assert.equal(shape.type, 'bar')
    assert.equal(shape.stack, true)
    assert.equal(shape.paddingInner, 0.12)
  })

  test('the vendor share chart keeps its 100% semantics and label thinning', () => {
    // VChart cannot render without a canvas, so the spec contract is asserted
    // at the source level: preference-driven type, pre-normalised share field,
    // a fixed 0–1 axis and the shared label filter.
    const source = readFileSync(
      new URL('../../components/market-share-section.tsx', import.meta.url)
        .pathname,
      'utf8'
    )

    assert.match(source, /\.\.\.rankingChartShape\(/)
    assert.match(source, /yField: 'share',/)
    assert.match(source, /max: 1,/)
    assert.match(source, /min: 0,/)
    assert.match(source, /dataFilter: rankingLabelFilter\(bucketCount\)/)
  })

  test('a natural-month default range selects the month period', () => {
    assert.equal(defaultRankingPeriod('month-to-date'), 'month')
    assert.equal(defaultRankingPeriod('last-month'), 'month')
  })

  test('a day default range keeps the weekly period', () => {
    assert.equal(defaultRankingPeriod('days'), 'week')
    assert.equal(defaultRankingPeriod(undefined), 'week')
  })
})

describe('ranking bucket labels follow the interface language', () => {
  test('English renders a short month and day', () => {
    assert.equal(formatRankingBucketLabel(TS, 'en'), 'Sep 26')
  })

  test('Chinese renders the localised month and day', () => {
    assert.equal(formatRankingBucketLabel(TS, 'zh'), '9月26日')
  })

  test('hourly buckets show a clock time instead of a date', () => {
    const label = formatRankingBucketLabel(TS, 'zh', true)

    assert.match(label, /\d{1,2}:\d{2}/)
    assert.equal(label.includes('月'), false, `unexpected date label: ${label}`)
  })

  // Regression: the app feeds i18next codes such as `zhCN` / `enUS` here, and
  // `new Intl.DateTimeFormat('zhCN')` throws RangeError — which crashed the
  // rankings page in production. Every one of these must stay throw-free.
  test("the app's real language codes never throw", () => {
    const cases: Array<[string | undefined, string]> = [
      ['zhCN', '月'],
      ['enUS', 'Sep'],
      ['zhTW', '月'],
      ['zh-TW', '月'],
      ['en', 'Sep'],
      ['xx-invalid', ''],
      [undefined, ''],
    ]

    for (const [code, expectedFragment] of cases) {
      let label = ''
      assert.doesNotThrow(
        () => {
          label = formatRankingBucketLabel(TS, code)
        },
        `locale ${String(code)} must not throw`
      )

      assert.notEqual(
        label,
        '',
        `locale ${String(code)} produced an empty label`
      )
      if (expectedFragment) {
        assert.equal(
          label.includes(expectedFragment),
          true,
          `locale ${String(code)} produced "${label}"`
        )
      }
    }
  })

  // The production path: an i18next instance configured with the app's own
  // `zhCN` code feeds the chart label builder. This is the exact chain that
  // crashed the rankings page, so it is asserted end to end here.
  test('a zhCN i18next instance still yields Chinese chart labels', async () => {
    const { createInstance } = await import('i18next')
    const i18n = createInstance()
    await i18n.init({
      lng: 'zhCN',
      fallbackLng: 'zhCN',
      resources: { zhCN: { translation: {} } },
    })

    const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
    assert.equal(locale, 'zh-CN')

    const points = [{ ts: TS, label: 'Sep 26' }].map((point) => ({
      ...point,
      label: formatRankingBucketLabel(point.ts, locale),
    }))

    assert.equal(points[0].label, '9月26日')
    await i18n.removeResourceBundle('zhCN', 'translation')
  })

  test('an unparsable timestamp falls back to the raw value', () => {
    assert.equal(formatRankingBucketLabel('not-a-date', 'en'), 'not-a-date')
  })
})
