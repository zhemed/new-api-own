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

/**
 * Backend contract (router/api-router.go, both AdminAuth):
 *   GET  /api/status/update        → `{ success, data: UpdateCheckStatus }`
 *   POST /api/status/update/apply  → `{ success, data: UpdateApplyResult }`
 *
 * The panel never talks to a release host itself: the server owns the check,
 * the download, the checksum verification and the restart. `GET` is a normal
 * status read (the server caches and backs off), so re-asking is cheap.
 */
const UPDATE_STATUS_PATH = '/api/status/update'
const UPDATE_APPLY_PATH = '/api/status/update/apply'
const UPDATE_ROLLBACK_PATH = '/api/status/update/rollback'

/** What the server reports about the running build vs the published release. */
export type ServerUpdateStatus = {
  /** `check_enabled` (or the older `enabled`) — false means the panel must not imply a result. */
  checkEnabled: boolean
  /** Server verdict: `disabled|unknown|up_to_date|update_available|unavailable`. */
  serverState: string | null
  /** Newest published version (`v0.0.6`), empty when the server has none. */
  latestVersion: string | null
  /** `apply_enabled`; `null` when a server omits the field. */
  applyEnabled: boolean | null
  /**
   * `persistent === false` means the new binary lives only inside the running
   * container. `null` keeps "the server did not say" distinct from "it said yes".
   */
  persistent: boolean | null
  /** Server-authored persistence warning; preferred over our local copy. */
  hint: string | null
  /**
   * Whether a local rollback point exists. `null` when the server does not
   * report it — the panel then lets the operator try and explains the refusal.
   */
  rollbackAvailable: boolean | null
}

/** Failure categories the panel can explain in translated copy. */
type UpdateFailureReason =
  /** `update_check_setting.apply_enabled` is false. */
  | 'disabled'
  /** Another apply is already running. */
  | 'busy'
  /** The published version is not newer than the running build. */
  | 'not_newer'
  /** The release asset could not be fetched. */
  | 'download_failed'
  /** The downloaded binary failed checksum verification. */
  | 'checksum_failed'
  /** The release has no asset for this platform. */
  | 'unsupported_platform'
  /** The asset exceeds the server's size limit. */
  | 'too_large'
  /** Replacing the running binary failed. */
  | 'replace_failed'
  /** The downloaded artifact failed signature verification, so it was refused. */
  | 'signature_mismatch'
  /** The caller authenticated with a system access token instead of root. */
  | 'pat_not_allowed'
  /** No `.prev` artifact exists, so there is nothing to roll back to. */
  | 'rollback_unavailable'
  /** The rollback itself failed. */
  | 'rollback_failed'
  /** The rollback point exists but is not a usable executable. */
  | 'rollback_point_invalid'
  /** The tree could not be backed up before replacing. */
  | 'backup_failed'

export type ApplyUpdateResult =
  | {
      ok: true
      version: string | null
      /** Server view of whether the new binary survives a container recreate. */
      persistent: boolean | null
      /** Server-authored persistence warning, when it sent one. */
      hint: string | null
    }
  | {
      ok: false
      /** Classified failure, or `null` when only `message` is available. */
      reason: UpdateFailureReason | null
      /** Server detail; empty when the request never reached a reason. */
      message: string
    }

const FAILURE_CODES: Record<string, UpdateFailureReason> = {
  disabled: 'disabled',
  busy: 'busy',
  not_newer: 'not_newer',
  download_failed: 'download_failed',
  source_unavailable: 'download_failed',
  asset_missing: 'download_failed',
  checksum_download_failed: 'download_failed',
  checksum_mismatch: 'checksum_failed',
  checksum_missing: 'checksum_failed',
  checksum_asset_missing: 'checksum_failed',
  unsupported_platform: 'unsupported_platform',
  binary_too_large: 'too_large',
  replace_failed: 'replace_failed',
  temp_file_failed: 'replace_failed',
  executable_unknown: 'replace_failed',
  config_invalid: 'replace_failed',
  signature_mismatch: 'signature_mismatch',
  signature_invalid: 'signature_mismatch',
  signature_verify_failed: 'signature_mismatch',
  pat_not_allowed: 'pat_not_allowed',
  token_not_allowed: 'pat_not_allowed',
  rollback_unavailable: 'rollback_unavailable',
  no_previous_version: 'rollback_unavailable',
  no_rollback_point: 'rollback_unavailable',
  rollback_point_invalid: 'rollback_point_invalid',
  rollback_failed: 'rollback_failed',
  backup_failed: 'backup_failed',
}

export type RollbackUpdateResult =
  | { ok: true; persistent: boolean | null; hint: string | null }
  | {
      ok: false
      reason: UpdateFailureReason | null
      message: string
    }

type RollbackEnvelope = {
  success?: unknown
  message?: unknown
  code?: unknown
  data?: { rolled_back?: unknown; persistent?: unknown; hint?: unknown }
}

type UpdateStatusEnvelope = {
  data?: {
    enabled?: unknown
    check_enabled?: unknown
    apply_enabled?: unknown
    state?: unknown
    latest_version?: unknown
    persistent?: unknown
    hint?: unknown
    rollback_available?: unknown
  }
}

type ApplyEnvelope = {
  success?: unknown
  message?: unknown
  code?: unknown
  data?: { version?: unknown; persistent?: unknown; hint?: unknown }
}

function readString(value: unknown): string | null {
  if (typeof value !== 'string') return null
  const trimmed = value.trim()
  return trimmed.length > 0 ? trimmed : null
}

/** First boolean among the candidates, or `null` when none is a boolean. */
function readBoolean(...values: unknown[]): boolean | null {
  for (const value of values) {
    if (typeof value === 'boolean') return value
  }
  return null
}

/** Pull the server's message/code out of whatever the request layer threw. */
function readThrownDetail(error: unknown): {
  message: string
  code: string | null
} {
  if (error && typeof error === 'object') {
    const data = (error as { response?: { data?: unknown } }).response?.data
    if (data && typeof data === 'object') {
      const payload = data as { message?: unknown; code?: unknown }
      return {
        message: readString(payload.message) ?? '',
        code: readString(payload.code),
      }
    }
  }
  return { message: error instanceof Error ? error.message : '', code: null }
}

/**
 * Ask the server to swap the running binary back to its local rollback point.
 *
 * Never throws, for the same reason as `applyUpdate`: the panel must always be
 * able to describe what happened, including "there was nothing to roll back to".
 */
export async function rollbackUpdate(): Promise<RollbackUpdateResult> {
  try {
    const res = await api.post<RollbackEnvelope>(
      UPDATE_ROLLBACK_PATH,
      {},
      { skipBusinessError: true, skipErrorHandler: true }
    )
    const payload = res.data ?? {}
    if (payload.success !== true || payload.data?.rolled_back !== true) {
      const code = readString(payload.code)
      return {
        ok: false,
        reason: code ? (FAILURE_CODES[code] ?? null) : null,
        message: readString(payload.message) ?? '',
      }
    }
    return {
      ok: true,
      persistent: readBoolean(payload.data.persistent),
      hint: readString(payload.data.hint),
    }
  } catch (error) {
    const detail = readThrownDetail(error)
    return {
      ok: false,
      reason: detail.code ? (FAILURE_CODES[detail.code] ?? null) : null,
      message: detail.message,
    }
  }
}

/**
 * Read the server-side update status.
 *
 * Resolves `null` when the endpoint cannot be reached or answers unusably: the
 * caller then shows the neutral "cannot tell" state instead of an error.
 */
export async function fetchUpdateStatus(): Promise<ServerUpdateStatus | null> {
  try {
    const res = await api.get<UpdateStatusEnvelope>(UPDATE_STATUS_PATH, {
      skipBusinessError: true,
      skipErrorHandler: true,
    })
    const data = res.data?.data
    if (!data || typeof data !== 'object') return null

    return {
      // `check_enabled` is the newer name for the same switch; a payload with
      // neither is treated as "not enabled" rather than guessed at.
      checkEnabled: readBoolean(data.check_enabled, data.enabled) === true,
      serverState: readString(data.state),
      latestVersion: readString(data.latest_version),
      // `null` keeps "the server did not say" distinct from "the server said no".
      applyEnabled: readBoolean(data.apply_enabled),
      persistent: readBoolean(data.persistent),
      hint: readString(data.hint),
      rollbackAvailable: readBoolean(data.rollback_available),
    }
  } catch {
    return null
  }
}

/**
 * Ask the server to install the newest release.
 *
 * Never throws: transport errors and rejected responses both come back as a
 * failed result so the caller can always show a readable outcome.
 */
export async function applyUpdate(): Promise<ApplyUpdateResult> {
  try {
    const res = await api.post<ApplyEnvelope>(
      UPDATE_APPLY_PATH,
      {},
      { skipBusinessError: true, skipErrorHandler: true }
    )
    const payload = res.data ?? {}
    // `common.ApiSuccess` always answers `{success: true, data}`, so anything
    // else is not a confirmation that the update was applied. Reporting a
    // success the server never sent would be the worst possible lie here.
    if (payload.success !== true) {
      const code = readString(payload.code)
      return {
        ok: false,
        reason: code ? (FAILURE_CODES[code] ?? null) : null,
        message: readString(payload.message) ?? '',
      }
    }
    return {
      ok: true,
      version: readString(payload.data?.version),
      persistent: readBoolean(payload.data?.persistent),
      hint: readString(payload.data?.hint),
    }
  } catch (error) {
    const detail = readThrownDetail(error)
    return {
      ok: false,
      reason: detail.code ? (FAILURE_CODES[detail.code] ?? null) : null,
      message: detail.message,
    }
  }
}
