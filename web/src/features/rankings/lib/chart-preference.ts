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
import type { RankingPeriod } from '../types'

/**
 * The rankings page follows the dashboard's saved chart preferences instead of
 * hard-coding its own: picking an area chart there should render one here too.
 */
export function rankingChartType(
  preference: string | undefined
): 'area' | 'bar' {
  return preference === 'area' ? 'area' : 'bar'
}

/**
 * Period used when the URL does not name one. A natural-month default range
 * opens the rankings on "This month" so both pages agree; otherwise the
 * historical weekly default is kept.
 */
export function defaultRankingPeriod(
  rangeKind: string | undefined
): RankingPeriod {
  return rangeKind === 'month-to-date' || rangeKind === 'last-month'
    ? 'month'
    : 'week'
}
