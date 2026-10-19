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
import { useSyncExternalStore } from 'react'

import { DASHBOARD_CHART_PREFERENCES_STORAGE_KEY } from '../constants'
import type { DashboardChartPreferences } from '../types'
import { subscribeChartPreferences } from './chart-preferences-events'
import { getSavedChartPreferences } from './filters'

/**
 * Cached snapshot so `useSyncExternalStore` sees a stable reference.
 *
 * `getSavedChartPreferences()` parses and validates on every call, so calling it
 * from `getSnapshot` would return a new object each time and React would loop.
 * The raw localStorage string is the cheap identity: as long as it is unchanged
 * the cached object is returned as-is.
 */
let cachedRaw: string | null | undefined
let cachedValue: DashboardChartPreferences | undefined

function getSnapshot(): DashboardChartPreferences {
  if (typeof window === 'undefined') return getSavedChartPreferences()

  const raw = window.localStorage.getItem(
    DASHBOARD_CHART_PREFERENCES_STORAGE_KEY
  )
  if (cachedValue !== undefined && raw === cachedRaw) return cachedValue

  cachedRaw = raw
  cachedValue = getSavedChartPreferences()
  return cachedValue
}

/** Test seam: forget the cached snapshot. */
export function resetChartPreferencesSnapshot(): void {
  cachedRaw = undefined
  cachedValue = undefined
}

/**
 * The saved chart preferences as reactive state.
 *
 * Every place that renders something derived from the preference (chart type,
 * default range) must use this instead of calling `getSavedChartPreferences()`
 * directly, otherwise saving the preferences only takes effect after a reload.
 */
export function useChartPreferences(): DashboardChartPreferences {
  return useSyncExternalStore(
    subscribeChartPreferences,
    getSnapshot,
    getSnapshot
  )
}
