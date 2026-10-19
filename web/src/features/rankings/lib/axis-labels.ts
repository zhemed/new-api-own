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
import {
  axisLabelStride,
  MAX_AXIS_LABELS,
} from '@/features/dashboard/lib/charts'

/**
 * Which bucket indexes keep their label.
 *
 * A full 30-day month has 30 buckets; printing all of them is unreadable. The
 * rule is the same one the model-analytics charts use (`axisLabelStride`), plus
 * the guarantee that the *last* bucket keeps a label so the axis ends on today.
 *
 * The kept indexes are spaced by `stride`; the appended final index can make
 * the last gap shorter or up to one stride longer.
 */
export function rankingLabelIndexes(bucketCount: number): number[] {
  if (!Number.isFinite(bucketCount) || bucketCount <= 0) return []

  const count = Math.trunc(bucketCount)
  if (count === 1) return [0]

  const stride = axisLabelStride(count)
  const indexes: number[] = []
  for (let index = 0; index < count; index += stride) {
    indexes.push(index)
  }

  if (indexes.at(-1) !== count - 1) {
    indexes.push(count - 1)
    if (indexes.length > MAX_AXIS_LABELS) {
      // Drop the stride item before the end so both ends stay labelled.
      indexes.splice(indexes.length - 2, 1)
    }
  }

  return indexes
}

/**
 * vchart `label.dataFilter` for the rankings axes.
 *
 * `dataFilter` receives the array of axis items (NOT a single datum), so the
 * decision can only be index-based. Deliberately not `formatMethod`: vchart
 * calls it as `formatFunc(value, datum, formatter)`, and the third argument is
 * the formatter, not the index.
 */
export function rankingLabelFilter(
  bucketCount: number
): (data: unknown[]) => unknown[] {
  const keep = new Set(rankingLabelIndexes(bucketCount))
  return (data: unknown[]) => data.filter((_, index) => keep.has(index))
}
