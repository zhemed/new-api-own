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
/**
 * Local version comparison for the update checker.
 *
 * GitHub release tags carry a `v` prefix (`v0.0.6`) while the panel reports the
 * build version without it (`0.0.6`), so a strict `===` comparison always
 * reported a new release even when the running build was current.
 *
 * Everything here is pure string/number handling on values already received —
 * no network access, no new dependencies.
 */

export type VersionComparison =
  /** Both sides resolve to the same version. */
  | 'equal'
  /** `latest` is greater than `current` — a real update is available. */
  | 'newer'
  /** `latest` is lower than `current` — the running build is ahead. */
  | 'older'
  /** Cannot be decided reliably (missing or non-comparable input). */
  | 'unknown'

/** Strip surrounding whitespace, a leading `v`/`V`, and any build metadata. */
export function normalizeVersion(raw: string | null | undefined): string {
  if (typeof raw !== 'string') return ''
  const trimmed = raw.trim().replace(/^[vV]/, '')
  return trimmed.split('+')[0].trim()
}

/** Split `1.2.3-beta.1` into its numeric core and its pre-release part. */
function splitVersion(value: string): { core: string[]; preRelease: string } {
  const [core = '', preRelease = ''] = value.split('-', 2)
  return { core: core.split('.'), preRelease }
}

function isNumeric(segment: string): boolean {
  return /^\d+$/.test(segment)
}

/**
 * Compare two normalized version cores segment by segment.
 * Returns a negative number, 0, or a positive number.
 * Falls back to a plain string comparison when a segment is not numeric
 * (e.g. `1.0.0-rc` style cores that survived normalization).
 */
function compareCores(a: string[], b: string[]): number {
  const length = Math.max(a.length, b.length)
  for (let index = 0; index < length; index++) {
    const left = a[index] ?? '0'
    const right = b[index] ?? '0'
    if (left === right) continue
    if (isNumeric(left) && isNumeric(right)) {
      return Number(left) - Number(right)
    }
    return left.localeCompare(right)
  }
  return 0
}

/**
 * Compare the running version against the latest published version.
 *
 * A pre-release (`0.0.6-beta`) ranks below its release (`0.0.6`), matching
 * semver precedence, so a beta build still sees the final release as an update.
 */
export function compareVersions(
  current: string | null | undefined,
  latest: string | null | undefined
): VersionComparison {
  const currentNormalized = normalizeVersion(current)
  const latestNormalized = normalizeVersion(latest)

  if (currentNormalized === '' || latestNormalized === '') return 'unknown'

  const currentParts = splitVersion(currentNormalized)
  const latestParts = splitVersion(latestNormalized)

  const coreResult = compareCores(currentParts.core, latestParts.core)
  // `coreResult > 0` means the running version is ahead, so `latest` is older.
  if (coreResult !== 0) return coreResult > 0 ? 'older' : 'newer'

  if (currentParts.preRelease === latestParts.preRelease) return 'equal'
  // Same core: the one WITHOUT a pre-release suffix is the newer release.
  if (currentParts.preRelease === '') return 'older'
  if (latestParts.preRelease === '') return 'newer'
  return latestParts.preRelease.localeCompare(currentParts.preRelease) > 0
    ? 'newer'
    : 'older'
}
