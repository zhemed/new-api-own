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
  TIER_LABEL_I18N_KEYS,
  getTierDisplayLabel,
} from '../../lib/tier-labels'

const LOCALES = ['en', 'zh', 'zh-TW', 'fr', 'ja', 'ru', 'vi'] as const

function loadLocale(locale: string): Record<string, string> {
  return (
    JSON.parse(
      readFileSync(
        new URL(`../../../../i18n/locales/${locale}.json`, import.meta.url),
        'utf8'
      )
    ) as { translation: Record<string, string> }
  ).translation
}

/** Stand-in for i18next's `t`: unknown keys render themselves. */
function translatorFor(locale: string): (key: string) => string {
  const dict = loadLocale(locale)
  return (key: string) => dict[key] ?? key
}

const BUILT_IN_TIERS: Array<[string, string]> = [
  ['peak', '高峰时段'],
  ['offpeak', '空闲时段'],
  ['base', '基础'],
  ['standard', '标准'],
  ['short', '短'],
  ['mid', '中'],
  ['long', '长'],
  ['short_output', '短输出'],
  ['long_output', '长输出'],
  ['mid_context', '中上下文'],
  ['long_context', '长上下文'],
  ['discount', '优惠'],
]

describe('tier label display mapping', () => {
  test('every built-in tier name renders translated in Chinese', () => {
    const t = translatorFor('zh')
    for (const [label, expected] of BUILT_IN_TIERS) {
      assert.equal(
        getTierDisplayLabel(label, t),
        expected,
        `${label} should render as ${expected}`
      )
    }
  })

  test('the same tier names render their English source strings in English', () => {
    const t = translatorFor('en')
    for (const [label] of BUILT_IN_TIERS) {
      assert.equal(
        getTierDisplayLabel(label, t),
        t(TIER_LABEL_I18N_KEYS[label])
      )
    }
    assert.equal(getTierDisplayLabel('peak', t), 'Peak hours')
    assert.equal(getTierDisplayLabel('offpeak', t), 'Off-peak hours')
  })

  test('an unknown tier name is shown exactly as written', () => {
    const t = translatorFor('zh')
    assert.equal(getTierDisplayLabel('my_custom_tier', t), 'my_custom_tier')
    assert.equal(getTierDisplayLabel('My Custom Tier', t), 'My Custom Tier')
    assert.equal(getTierDisplayLabel('off_peak', t), 'off_peak')
  })

  test('an empty tier name falls back to Default', () => {
    const zh = translatorFor('zh')
    const en = translatorFor('en')
    assert.equal(getTierDisplayLabel('', zh), '默认')
    assert.equal(getTierDisplayLabel('   ', zh), '默认')
    assert.equal(getTierDisplayLabel(null, zh), '默认')
    assert.equal(getTierDisplayLabel(undefined, zh), '默认')
    assert.equal(getTierDisplayLabel('', en), 'Default')
  })

  test('case and separators do not defeat the mapping', () => {
    const t = translatorFor('zh')
    assert.equal(getTierDisplayLabel('Short Output', t), '短输出')
    assert.equal(getTierDisplayLabel('  PEAK  ', t), '高峰时段')
  })

  test('every mapped key exists in all seven locales', () => {
    for (const locale of LOCALES) {
      const dict = loadLocale(locale)
      for (const key of Object.values(TIER_LABEL_I18N_KEYS)) {
        assert.ok(
          typeof dict[key] === 'string' && dict[key].length > 0,
          `${locale}.json is missing "${key}"`
        )
      }
    }
  })
})
