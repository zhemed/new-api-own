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
  DASHBOARD_CHART_PREFERENCES_STORAGE_KEY,
  DEFAULT_DASHBOARD_CHART_PREFERENCES,
  DEFAULT_TIME_GRANULARITY,
  EMPTY_DASHBOARD_FILTERS,
  TIME_GRANULARITY_STORAGE_KEY,
  TIME_RANGE_PRESETS,
  TIME_RANGE_BY_GRANULARITY,
  type DashboardRangePresetKind,
} from '@/features/dashboard/constants'
import type {
  ConsumptionDistributionChartType,
  DashboardChartPreferences,
  DashboardFilters,
  ModelAnalyticsChartTab,
} from '@/features/dashboard/types'
import { getCalendarDateRange, type TimeGranularity } from '@/lib/time'

import { emitChartPreferencesChange } from './chart-preferences-events'

function isTimeGranularity(value: unknown): value is TimeGranularity {
  return value === 'hour' || value === 'day' || value === 'week'
}

function getLegacySavedGranularity(): TimeGranularity {
  if (typeof window === 'undefined') return DEFAULT_TIME_GRANULARITY
  const saved = localStorage.getItem(TIME_GRANULARITY_STORAGE_KEY)
  return isTimeGranularity(saved) ? saved : DEFAULT_TIME_GRANULARITY
}

function isConsumptionDistributionChartType(
  value: unknown
): value is ConsumptionDistributionChartType {
  return value === 'bar' || value === 'area'
}

function isModelAnalyticsChartTab(
  value: unknown
): value is ModelAnalyticsChartTab {
  return value === 'trend' || value === 'proportion' || value === 'top'
}

function isTimeRangePresetDays(value: unknown): value is number {
  return TIME_RANGE_PRESETS.some((preset) => preset.days === value)
}

const RANGE_PRESET_KINDS = new Set<DashboardRangePresetKind>([
  'days',
  'month-to-date',
  'last-month',
])

function isRangePresetKind(value: unknown): value is DashboardRangePresetKind {
  return RANGE_PRESET_KINDS.has(value as DashboardRangePresetKind)
}

/**
 * Map a stored range length onto a currently offered preset.
 *
 * The longest preset used to be 29 days and is now 30, so preferences saved
 * before that change must keep working instead of silently falling back to the
 * default. Only that exact legacy value is translated — anything else is left
 * for `isTimeRangePresetDays` to accept or reject.
 */
export function migrateLegacyTimeRangeDays(value: unknown): unknown {
  return value === 29 ? 30 : value
}

export function cleanFilters<T extends Record<string, unknown>>(
  filters: T
): Partial<T> {
  const cleaned: Partial<T> = {}
  for (const [key, value] of Object.entries(filters)) {
    if (value === undefined || value === null) continue
    if (typeof value === 'string') {
      const trimmed = value.trim()
      if (trimmed) cleaned[key as keyof T] = trimmed as T[keyof T]
      continue
    }
    cleaned[key as keyof T] = value as T[keyof T]
  }
  return cleaned
}

/**
 * The only place that decides a chart's bucket granularity.
 *
 * Granularity is a function of the selected range, never an independent
 * setting: up to one day is bucketed per hour, up to 31 days per day, and
 * anything longer per week. Every entry point (default preferences, quick
 * ranges, custom ranges, the query's `time_granularity`) must use this.
 */
/** A resolved date range plus the calendar days it spans. */
export interface ResolvedRange {
  start: Date
  end: Date
  /** Inclusive calendar-day span, used to pick the bucket granularity. */
  days: number
}

function startOfMonth(now: Date, offsetMonths = 0): Date {
  return new Date(
    now.getFullYear(),
    now.getMonth() + offsetMonths,
    1,
    0,
    0,
    0,
    0
  )
}

function daysInMonth(now: Date, offsetMonths = 0): number {
  return new Date(
    now.getFullYear(),
    now.getMonth() + offsetMonths + 1,
    0
  ).getDate()
}

/**
 * Resolve a range preset to dates. The only place presets become a range, so
 * rolling windows and natural months can never disagree between surfaces.
 */
export function resolveRangePreset(
  preset: { kind: DashboardRangePresetKind; days?: number },
  now: Date = new Date()
): ResolvedRange {
  if (preset.kind === 'month-to-date') {
    return {
      start: startOfMonth(now),
      end: new Date(now),
      days: now.getDate(),
    }
  }

  if (preset.kind === 'last-month') {
    const start = startOfMonth(now, -1)
    // Last instant of the previous month, i.e. one millisecond before this one.
    const end = new Date(startOfMonth(now).getTime() - 1)
    return { start, end, days: daysInMonth(now, -1) }
  }

  const days = preset.days ?? 1
  const { start, end } = getCalendarDateRange(days, now)
  return { start, end, days }
}

export function granularityForRangeDays(days: number): TimeGranularity {
  if (!Number.isFinite(days)) return 'day'
  if (days <= 1) return 'hour'
  if (days <= 31) return 'day'
  return 'week'
}

/** Inclusive calendar-day span of a filter range, floored to at least one. */
export function rangeDaysOf(start?: Date, end?: Date): number | undefined {
  if (!start || !end) return undefined
  const startDay = new Date(start)
  startDay.setHours(0, 0, 0, 0)
  const endDay = new Date(end)
  endDay.setHours(0, 0, 0, 0)
  const days =
    Math.round((endDay.getTime() - startDay.getTime()) / 86_400_000) + 1
  return Number.isFinite(days) && days > 0 ? days : undefined
}

export function getSavedChartPreferences(): DashboardChartPreferences {
  if (typeof window === 'undefined') return DEFAULT_DASHBOARD_CHART_PREFERENCES

  const fallbackPreferences = {
    ...DEFAULT_DASHBOARD_CHART_PREFERENCES,
    defaultTimeGranularity: getLegacySavedGranularity(),
  }

  try {
    const raw = localStorage.getItem(DASHBOARD_CHART_PREFERENCES_STORAGE_KEY)
    if (!raw) return fallbackPreferences

    const parsed = JSON.parse(raw) as Partial<DashboardChartPreferences>
    // `29` was the long preset before it became `30`; migrate it before the
    // preset check so an existing preference is not reset to the default.
    const migratedDays = migrateLegacyTimeRangeDays(parsed.defaultTimeRangeDays)
    return {
      consumptionDistributionChart: isConsumptionDistributionChartType(
        parsed.consumptionDistributionChart
      )
        ? parsed.consumptionDistributionChart
        : fallbackPreferences.consumptionDistributionChart,
      modelAnalyticsChart: isModelAnalyticsChartTab(parsed.modelAnalyticsChart)
        ? parsed.modelAnalyticsChart
        : fallbackPreferences.modelAnalyticsChart,
      defaultTimeRangeDays: isTimeRangePresetDays(migratedDays)
        ? migratedDays
        : fallbackPreferences.defaultTimeRangeDays,
      defaultTimeGranularity: isTimeGranularity(parsed.defaultTimeGranularity)
        ? parsed.defaultTimeGranularity
        : fallbackPreferences.defaultTimeGranularity,
      // Missing on payloads saved before natural months existed: read as days.
      defaultRangeKind: isRangePresetKind(parsed.defaultRangeKind)
        ? parsed.defaultRangeKind
        : 'days',
    }
  } catch {
    return fallbackPreferences
  }
}

export function saveChartPreferences(
  preferences: DashboardChartPreferences
): void {
  if (typeof window === 'undefined') return
  const stored: DashboardChartPreferences = {
    ...preferences,
    defaultRangeKind: isRangePresetKind(preferences.defaultRangeKind)
      ? preferences.defaultRangeKind
      : 'days',
  }
  localStorage.setItem(
    DASHBOARD_CHART_PREFERENCES_STORAGE_KEY,
    JSON.stringify(stored)
  )
  // Tell every mounted consumer (rankings cards, chart types) to re-read.
  emitChartPreferencesChange()
}

export function getDefaultDays(granularity?: TimeGranularity): number {
  // Granularity is derived from the range, so the only direction left is
  // granularity -> its paired range length (used by the flow charts).
  if (granularity) return TIME_RANGE_BY_GRANULARITY[granularity]
  return getSavedChartPreferences().defaultTimeRangeDays
}

export function buildDefaultDashboardFilters(
  preferences: DashboardChartPreferences = getSavedChartPreferences(),
  /** Injectable clock so callers/tests can pin "now". */
  now: Date = new Date()
): DashboardFilters {
  // One resolver for every surface: a natural-month preference opens on the
  // 1st of that month, a day preference stays a rolling calendar window.
  const range = resolveRangePreset(
    {
      kind: isRangePresetKind(preferences.defaultRangeKind)
        ? preferences.defaultRangeKind
        : 'days',
      days: preferences.defaultTimeRangeDays,
    },
    now
  )
  return {
    ...EMPTY_DASHBOARD_FILTERS,
    start_timestamp: range.start,
    end_timestamp: range.end,
    time_granularity: granularityForRangeDays(range.days),
  }
}

export function buildQueryParams(
  timeRange: { start_timestamp: number; end_timestamp: number },
  filters?: { time_granularity?: TimeGranularity; username?: string }
): {
  start_timestamp: number
  end_timestamp: number
  default_time: string
  username?: string
} {
  return {
    ...timeRange,
    default_time:
      filters?.time_granularity ??
      granularityForRangeDays(getSavedChartPreferences().defaultTimeRangeDays),
    ...(filters?.username && { username: filters.username }),
  }
}
