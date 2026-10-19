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
// ----------------------------------------------------------------------------
import { toIntlLocale } from '@/i18n/languages'

// Rankings formatting helpers
// ----------------------------------------------------------------------------
/**
 * Localised x-axis label for a ranking bucket.
 *
 * The backend still sends an English `label` ("Sep 26"), but the axis must
 * follow the interface language, so the label is rebuilt from the bucket's
 * ISO timestamp. Hourly buckets (the "today" period) show a time instead of a
 * date.
 *
 * i18next language codes (`zhCN`, `zhTW`, …) are NOT valid BCP-47 tags and make
 * `new Intl.DateTimeFormat` throw `RangeError`, which used to crash the whole
 * rankings page. The tag is normalised first and every construction attempt is
 * guarded, so an unknown locale degrades to English and never throws.
 */
export function formatRankingBucketLabel(
  ts: string,
  locale?: string | null,
  hourly = false
): string {
  const date = new Date(ts)
  if (Number.isNaN(date.getTime())) return ts

  const options: Intl.DateTimeFormatOptions = hourly
    ? { hour: '2-digit', minute: '2-digit' }
    : { month: 'short', day: 'numeric' }

  for (const tag of [toIntlLocale(locale), locale, 'en']) {
    if (!tag) continue
    try {
      return new Intl.DateTimeFormat(tag, options).format(date)
    } catch {
      // invalid tag — try the next candidate
    }
  }

  // Last resort: a locale-independent label is better than a crashed page.
  const pad = (value: number) => String(value).padStart(2, '0')
  const time = `${pad(date.getHours())}:${pad(date.getMinutes())}`
  return hourly ? time : `${pad(date.getMonth() + 1)}-${pad(date.getDate())}`
}

/** Format a 0..1 share as a percentage with two decimals. */
export function formatShare(share: number): string {
  if (!Number.isFinite(share) || share <= 0) return '0%'
  if (share < 0.001) return '<0.1%'
  return `${(share * 100).toFixed(share < 0.01 ? 2 : 1)}%`
}
