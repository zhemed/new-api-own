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
import { useEffect, useState } from 'react'

import { compareVersions } from '../utils/version-compare'
import { fetchUpdateStatus } from './update-api'
import {
  resolveUpdateAvailability,
  type UpdateAvailability,
} from './update-status'

export type UpdateStatusState = {
  /** Newest published version the server reported, or `null` when it has none. */
  latestVersion: string | null
  /** The single state the panel renders. */
  availability: UpdateAvailability
  /** Whether the server allows applying updates from the panel. */
  applyEnabled: boolean
  /** A status request is in flight. */
  checking: boolean
  /** Re-read the server's update status. */
  refresh: () => Promise<void>
}

/**
 * Single access point for "is there a newer version, and may I install it?".
 *
 * Reads our own backend only. The check runs once on mount and again on demand;
 * the server owns the release lookup, its cache and its failure backoff, so the
 * panel never talks to a release host itself. Anything unusable degrades to
 * `unknown` instead of an error or a guessed update.
 */
export function useUpdateStatus(
  currentVersion?: string | null
): UpdateStatusState {
  const [status, setStatus] = useState<{
    checkEnabled: boolean
    serverState: string | null
    latestVersion: string | null
    applyEnabled: boolean | null
  } | null>(null)
  const [checking, setChecking] = useState(false)

  const refresh = async () => {
    setChecking(true)
    try {
      setStatus(await fetchUpdateStatus())
    } finally {
      setChecking(false)
    }
  }

  useEffect(() => {
    void refresh()
  }, [])

  return {
    latestVersion: status?.latestVersion ?? null,
    availability: resolveUpdateAvailability({
      checkEnabled: status?.checkEnabled ?? null,
      serverState: status?.serverState,
      comparison: compareVersions(
        currentVersion,
        status?.latestVersion ?? null
      ),
    }),
    // `apply_enabled` comes from the same payload; the server default is
    // "enabled" and the apply endpoint re-checks the switch on every call, so
    // a payload that omits the field must not hide the feature.
    applyEnabled: status?.applyEnabled ?? true,
    checking,
    refresh,
  }
}
