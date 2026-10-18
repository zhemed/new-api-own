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
import { api } from '@/lib/api'

/** First probe delay: the server needs a moment to begin restarting. */
const RESTART_POLL_INITIAL_DELAY_MS = 2_000
/** Backoff ceiling so a slow restart is still probed often enough. */
const RESTART_POLL_MAX_DELAY_MS = 15_000
/** Give up after this long and ask the operator to reload manually. */
export const RESTART_WATCH_TIMEOUT_MS = 90_000
/** Same-version probes needed to accept "the service came back" as success. */
const RECOVERED_OK_STREAK = 2

const RELOAD_STORAGE_KEY = 'update-reload-completed-for'

/** A `/api/status` payload, or `null` when the probe failed. */
export type StatusSnapshot = { version?: string } | null

export type RestartWatchOutcome =
  | { status: 'version-changed'; version: string }
  | { status: 'recovered'; version: string }
  | { status: 'timeout' }

export interface RestartWatchDeps {
  /** Version the page was running before the update was applied. */
  previousVersion: string
  readStatus: () => Promise<StatusSnapshot>
  /** Injected so tests can advance a virtual clock instead of sleeping. */
  sleep: (ms: number) => Promise<void>
  now: () => number
  timeoutMs?: number
  initialDelayMs?: number
  maxDelayMs?: number
}

/**
 * Poll `/api/status` until the restarted service reports a new version.
 *
 * Failures during the restart are expected, so every failed probe is retried
 * with exponential backoff. A same-version success streak also counts as
 * "back up" — that is the reinstall-without-version-bump case — but only after
 * two consecutive successes so a single stray response during shutdown cannot
 * trigger a reload.
 */
export async function waitForRestartedService(
  deps: RestartWatchDeps
): Promise<RestartWatchOutcome> {
  const timeoutMs = deps.timeoutMs ?? RESTART_WATCH_TIMEOUT_MS
  const maxDelayMs = deps.maxDelayMs ?? RESTART_POLL_MAX_DELAY_MS
  const startedAt = deps.now()
  let delay = deps.initialDelayMs ?? RESTART_POLL_INITIAL_DELAY_MS
  let okStreak = 0

  while (deps.now() - startedAt < timeoutMs) {
    await deps.sleep(delay)

    let snapshot: StatusSnapshot = null
    try {
      snapshot = await deps.readStatus()
    } catch {
      snapshot = null
    }

    const version = snapshot?.version
    if (typeof version === 'string' && version.length > 0) {
      if (version !== deps.previousVersion) {
        return { status: 'version-changed', version }
      }
      okStreak += 1
      if (okStreak >= RECOVERED_OK_STREAK) {
        return { status: 'recovered', version }
      }
    } else {
      okStreak = 0
    }

    delay = Math.min(delay * 2, maxDelayMs)
  }

  return { status: 'timeout' }
}

/** Only a confirmed `ok` answer may start the watch-and-reload flow. */
export function shouldWatchAfterApply(result: { ok: boolean }): boolean {
  return result.ok === true
}

/** Storage shim: sessionStorage is absent in some test/SSR environments. */
function reloadStore(): Storage | null {
  try {
    return typeof sessionStorage === 'undefined' ? null : sessionStorage
  } catch {
    return null
  }
}

export function hasReloadedFor(
  version: string,
  store: Storage | null = reloadStore()
): boolean {
  if (!store) return false
  try {
    return store.getItem(RELOAD_STORAGE_KEY) === version
  } catch {
    return false
  }
}

export function markReloadedFor(
  version: string,
  store: Storage | null = reloadStore()
): void {
  if (!store) return
  try {
    store.setItem(RELOAD_STORAGE_KEY, version)
  } catch {
    // A blocked storage must not stop the reload itself.
  }
}

export interface WatchAndReloadDeps extends RestartWatchDeps {
  reload: (version: string) => void
  hasReloaded?: (version: string) => boolean
  markReloaded?: (version: string) => void
}

/**
 * Wait for the restart, then reload exactly once per target version.
 *
 * The guard is keyed by the version we are about to run: after the reload the
 * page boots on that version, and a duplicate trigger for the same version is
 * ignored. A later update targets a different version and is allowed again.
 */
export async function watchForRestartAndReload(
  deps: WatchAndReloadDeps
): Promise<RestartWatchOutcome> {
  const outcome = await waitForRestartedService(deps)
  if (outcome.status === 'timeout') return outcome

  const targetVersion = outcome.version || deps.previousVersion
  const hasReloaded = deps.hasReloaded ?? hasReloadedFor
  const markReloaded = deps.markReloaded ?? markReloadedFor

  if (hasReloaded(targetVersion)) return outcome
  markReloaded(targetVersion)
  deps.reload(targetVersion)
  return outcome
}

/**
 * Default probe: reads `/api/status` bypassing every local cache.
 *
 * A reload must never be decided from a stale payload, so the request carries a
 * unique query parameter (`_ts`), `cache: 'no-store'`, and `disableDuplicate`
 * to skip the client's in-flight GET de-duplication.
 */
export async function readFreshStatus(): Promise<StatusSnapshot> {
  const res = await api.get('/api/status', {
    params: { _ts: Date.now() },
    disableDuplicate: true,
  })
  const payload = res.data?.data as Record<string, unknown> | undefined
  if (!payload || typeof payload !== 'object') return null
  return {
    version: typeof payload.version === 'string' ? payload.version : undefined,
  }
}
