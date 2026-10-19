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
import { DASHBOARD_CHART_PREFERENCES_STORAGE_KEY } from '../constants'

/**
 * Tiny pub/sub for the saved chart preferences.
 *
 * Kept separate from `filters.ts` so that `filters.ts` can emit without the
 * store having to import it back (that would be a cycle).
 */
const listeners = new Set<() => void>()
let storageBound = false

function bindCrossTabListener(): void {
  if (storageBound || typeof window === 'undefined') return
  storageBound = true
  window.addEventListener('storage', (event) => {
    if (event.key && event.key !== DASHBOARD_CHART_PREFERENCES_STORAGE_KEY) {
      return
    }
    emitChartPreferencesChange()
  })
}

/** Subscribe to preference writes. Returns the unsubscribe function. */
export function subscribeChartPreferences(listener: () => void): () => void {
  listeners.add(listener)
  bindCrossTabListener()
  return () => {
    listeners.delete(listener)
  }
}

/** Notify every subscriber that the stored preferences changed. */
export function emitChartPreferencesChange(): void {
  for (const listener of listeners) {
    listener()
  }
}
