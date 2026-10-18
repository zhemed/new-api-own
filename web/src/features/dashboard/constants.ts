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
import type {
  DashboardChartPreferences,
  DashboardFilters,
  DashboardRangePresetKind,
} from './types'

export const TIME_GRANULARITY_STORAGE_KEY = 'data_export_default_time'
export const DASHBOARD_CHART_PREFERENCES_STORAGE_KEY =
  'dashboard_models_chart_preferences'
export const DEFAULT_TIME_GRANULARITY = 'hour' as const
export const MAX_CHART_TREND_POINTS = 7

/**
 * Safety cap for the range-driven chart axis. The widest combination the UI
 * offers is 30 days at hour granularity (720 buckets), so this only exists to
 * keep a pathological range from producing thousands of points.
 */
export const MAX_CHART_AXIS_BUCKETS = 1000

export const DEFAULT_DASHBOARD_CHART_PREFERENCES: DashboardChartPreferences = {
  consumptionDistributionChart: 'bar',
  modelAnalyticsChart: 'trend',
  defaultTimeRangeDays: 1,
  defaultTimeGranularity: DEFAULT_TIME_GRANULARITY,
}

export const TIME_RANGE_BY_GRANULARITY = {
  hour: 1,
  day: 7,
  week: 30,
} as const

/**
 * How a range preset resolves to actual dates.
 *
 * `days` is the classic rolling window; the natural-month kinds exist because
 * "this month" cannot be expressed as a day count (28–31 days, month length
 * varies). Kept as a discriminant so every consumer resolves it the same way.
 */
export type { DashboardRangePresetKind }

export interface DashboardRangePreset {
  label: string
  kind: DashboardRangePresetKind
  /** Only meaningful when `kind === 'days'`. */
  days?: number
}

/** Rolling windows, safe to store in preferences as a plain day count. */
export const TIME_RANGE_PRESETS: readonly DashboardRangePreset[] = [
  { label: '1 Day', kind: 'days', days: 1 },
  { label: '7 Days', kind: 'days', days: 7 },
  { label: '14 Days', kind: 'days', days: 14 },
  { label: '30 Days', kind: 'days', days: 30 },
]

/**
 * Natural-month shortcuts. Offered in the filter dialog only: preferences keep
 * storing `defaultTimeRangeDays` (a number), so an existing saved value — and
 * the 29 → 30 migration — keeps working untouched.
 */
export const NATURAL_MONTH_PRESETS: readonly DashboardRangePreset[] = [
  { label: 'This Month', kind: 'month-to-date' },
  { label: 'Last Month', kind: 'last-month' },
]

export const CONSUMPTION_DISTRIBUTION_CHART_OPTIONS = [
  { value: 'bar', labelKey: 'Bar Chart' },
  { value: 'area', labelKey: 'Area Chart' },
] as const

export const MODEL_ANALYTICS_CHART_OPTIONS = [
  { value: 'trend', labelKey: 'Call Trend' },
  { value: 'proportion', labelKey: 'Call Count Distribution' },
  { value: 'top', labelKey: 'Call Count Ranking' },
] as const

export const EMPTY_DASHBOARD_FILTERS: DashboardFilters = {
  start_timestamp: undefined,
  end_timestamp: undefined,
  time_granularity: 'hour',
  username: '',
}
