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
import type { VersionComparison } from '../utils/version-compare'

/** What the panel tells the operator about the running build. */
export type UpdateAvailability =
  /** An administrator turned the update check off — not the same as "unknown". */
  | 'check-disabled'
  /** The server reported a version newer than the running build. */
  | 'update-available'
  /** The running build is current, or ahead of the published version. */
  | 'up-to-date'
  /** Nothing comparable to report — stay neutral instead of guessing. */
  | 'unknown'

/** Local fallback for payloads without a usable server state. */
function fromComparison(comparison: VersionComparison): UpdateAvailability {
  if (comparison === 'newer') return 'update-available'
  if (comparison === 'equal' || comparison === 'older') return 'up-to-date'
  return 'unknown'
}

/**
 * Decide the single state the panel shows.
 *
 * Precedence matters:
 * 1. A turned-off check is reported as `check-disabled`, never as `unknown` —
 *    "we were told not to look" and "we looked and could not tell" are
 *    different facts, and conflating them misleads the operator.
 * 2. Otherwise the server's own verdict wins (`state`, one of `update_available`,
 *    `up_to_date`, `disabled`, `unknown`, `unavailable`); a check that failed
 *    (`unavailable`) stays `unknown` rather than becoming "up to date".
 * 3. Only when the server sent no verdict at all do we fall back to the local
 *    comparison — which itself returns `unknown` for `older`-vs-`newer`
 *    inputs it cannot decide.
 */
export function resolveUpdateAvailability(input: {
  /** `null` means the panel could not reach the server's status endpoint. */
  checkEnabled: boolean | null
  serverState?: string | null
  comparison: VersionComparison
}): UpdateAvailability {
  if (input.checkEnabled === false) return 'check-disabled'
  if (input.checkEnabled === null) return 'unknown'

  switch (input.serverState) {
    case 'disabled':
      return 'check-disabled'
    case 'update_available':
      return 'update-available'
    case 'up_to_date':
      return 'up-to-date'
    case 'unknown':
    case 'unavailable':
      return 'unknown'
    default:
      return fromComparison(input.comparison)
  }
}
