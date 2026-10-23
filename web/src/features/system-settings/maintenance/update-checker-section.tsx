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
import { RefreshCcwIcon } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { Button } from '@/components/ui/button'
import { useIsAdmin } from '@/hooks/use-admin'
import { getBuildRevision } from '@/lib/build-metadata'
import { formatTimestamp } from '@/lib/format'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

import { SettingsSection } from '../components/settings-section'
import {
  readFreshStatus,
  shouldWatchAfterApply,
  watchForRestartAndReload,
} from './restart-watcher'
import { applyUpdate, rollbackUpdate } from './update-api'
import { useUpdateStatus } from './use-update-status'

type UpdateCheckerSectionProps = {
  currentVersion?: string | null
  startTime?: number | null
  /**
   * Test seam: the real watcher polls for up to 90s, which no test should wait
   * for. Production always uses the default.
   */
  postUpdateWatch?: typeof watchForRestartAndReload
}

/**
 * Version / update panel.
 *
 * Reports which build is running, which version the server last saw published,
 * and what that means — in four distinct states: an update is available, the
 * build is current, the server cannot tell, or an administrator turned the
 * check off. The last two are deliberately different sentences: "we were told
 * not to look" is not "we looked and could not tell".
 *
 * Admins get a confirmed one-click apply when an update is confirmed and the
 * server allows it. The panel never downloads or replaces anything itself — it
 * asks our own backend, which owns the download, the checksum and the restart.
 */
export function UpdateCheckerSection({
  currentVersion,
  startTime,
  postUpdateWatch,
}: UpdateCheckerSectionProps) {
  const { t } = useTranslation()
  const isAdmin = useIsAdmin()
  // Applying an update is a root-only action on the server (RootAuth, and the
  // system access token is refused), so a plain admin must never see a button
  // that could only answer 403.
  const isRoot = useAuthStore(
    (state) => state.auth.user?.role === ROLE.SUPER_ADMIN
  )
  const {
    latestVersion,
    availability,
    applyEnabled,
    persistenceHint: statusHint,
    rollbackAvailable,
    checking,
    refresh,
  } = useUpdateStatus(currentVersion)
  const [confirmOpen, setConfirmOpen] = useState(false)
  const [applying, setApplying] = useState(false)
  const [rollbackConfirmOpen, setRollbackConfirmOpen] = useState(false)
  const [rollingBack, setRollingBack] = useState(false)
  // Which action owns the shared restart dialog: the copy differs.
  const [flow, setFlow] = useState<'apply' | 'rollback'>('apply')
  // `restarting` keeps a persistent hint on screen (a toast would vanish before
  // the restart even starts); `timed-out` hands the reload back to the operator.
  const [phase, setPhase] = useState<'idle' | 'restarting' | 'timed-out'>(
    'idle'
  )
  // Server-authored persistence warning from the last apply; when present it
  // replaces our local wording everywhere the warning is shown.
  const [serverHint, setServerHint] = useState<string | null>(null)

  const uptime = startTime ? formatTimestamp(startTime) : t('Unknown')
  const version = currentVersion || t('Unknown')
  const updateConfirmed = availability === 'update-available'
  const canApplyUpdate = isRoot && updateConfirmed && applyEnabled
  const applyIsRootOnly = isAdmin && updateConfirmed && applyEnabled && !isRoot
  // Rollback replaces the same binary through the same root-only route, so it
  // follows the same gate. `rollbackAvailable === false` means the server told
  // us there is no rollback point; `null` means it did not say.
  const canRollback = isRoot && applyEnabled
  const rollbackKnownMissing = canRollback && rollbackAvailable === false
  // Prefer whatever the server said — first from the apply response, then from
  // the status payload — and only fall back to our own sentence.
  const persistenceHint =
    serverHint ??
    statusHint ??
    t(
      'This replaces the binary inside the running container. Recreating the container falls back to the version baked into the image.'
    )

  let statusText: string
  if (availability === 'check-disabled') {
    statusText = t(
      'Update checks are turned off. Set UPDATE_CHECK_ENABLED=true on the server to turn them back on.'
    )
  } else if (updateConfirmed && latestVersion) {
    statusText = t('New version available: {{version}}', {
      version: latestVersion,
    })
  } else if (availability === 'up-to-date') {
    statusText = t('Up to date')
  } else {
    statusText = t('Unable to determine the latest version.')
  }

  const handleCheckUpdates = async () => {
    await refresh()
  }

  const handleConfirmUpdate = async () => {
    setApplying(true)
    try {
      const result = await applyUpdate()
      if (result.ok) {
        // Belt and braces: only a confirmed success may ever reload the page.
        if (!shouldWatchAfterApply(result)) return
        // The dialog stays open: the restart progress belongs in it, otherwise
        // the operator sees the dialog vanish and the page reload "by itself".
        setServerHint(result.hint)
        setPhase('restarting')
        const outcome = await (postUpdateWatch ?? watchForRestartAndReload)({
          previousVersion: currentVersion ?? '',
          readStatus: readFreshStatus,
          sleep: (ms) => new Promise((resolve) => setTimeout(resolve, ms)),
          now: () => Date.now(),
          reload: () => window.location.reload(),
        })
        if (outcome.status === 'timeout') {
          setPhase('timed-out')
        }
        return
      }

      let detail: string
      if (result.reason === 'disabled') {
        detail = t(
          'In-panel updates are turned off by the administrator. Use the image upgrade steps in the README instead.'
        )
      } else if (result.reason === 'busy') {
        detail = t('Another update is already in progress.')
      } else if (result.reason === 'not_newer') {
        detail = t('This build already matches that version.')
      } else if (result.reason === 'download_failed') {
        detail = t('The new version could not be downloaded.')
      } else if (result.reason === 'checksum_failed') {
        detail = t('The downloaded update failed its checksum check.')
      } else if (result.reason === 'unsupported_platform') {
        detail = t('This platform does not support in-panel updates.')
      } else if (result.reason === 'too_large') {
        detail = t('The update package is larger than the server allows.')
      } else if (result.reason === 'signature_mismatch') {
        detail = t(
          'The update signature did not match the published checksum, so the replacement was refused.'
        )
      } else if (result.reason === 'pat_not_allowed') {
        detail = t(
          'System access tokens cannot install updates. Sign in as the root account instead.'
        )
      } else if (result.reason === 'rollback_unavailable') {
        detail = t('No previous version is available to roll back to.')
      } else if (result.reason === 'rollback_failed') {
        detail = t('Rolling back failed. Check the server logs for details.')
      } else if (result.reason === 'replace_failed') {
        detail = t(
          'The server could not replace its own files. Check the container permissions.'
        )
      } else if (result.message) {
        detail = result.message
      } else {
        detail = t('Update failed. Check the server logs for details.')
      }
      // Keep the dialog open so the operator can retry after fixing the cause.
      toast.error(detail)
    } finally {
      setApplying(false)
    }
  }

  const handleConfirmRollback = async () => {
    setRollingBack(true)
    try {
      const result = await rollbackUpdate()
      if (result.ok) {
        setFlow('rollback')
        setServerHint(result.hint)
        setPhase('restarting')
        const outcome = await (postUpdateWatch ?? watchForRestartAndReload)({
          previousVersion: currentVersion ?? '',
          readStatus: readFreshStatus,
          sleep: (ms) => new Promise((resolve) => setTimeout(resolve, ms)),
          now: () => Date.now(),
          reload: () => window.location.reload(),
        })
        if (outcome.status === 'timeout') {
          setPhase('timed-out')
        }
        return
      }
      let detail = result.message
      if (result.reason === 'rollback_unavailable') {
        detail = t('No previous version is available to roll back to.')
      } else if (result.reason === 'rollback_point_invalid') {
        detail = t('The rollback point is unusable, so nothing was replaced.')
      } else if (result.reason === 'rollback_failed') {
        detail = t('Rolling back failed. Check the server logs for details.')
      } else if (result.reason === 'pat_not_allowed') {
        detail = t(
          'System access tokens cannot install updates. Sign in as the root account instead.'
        )
      } else if (!detail) {
        detail = t('Update failed. Check the server logs for details.')
      }
      toast.error(detail)
    } finally {
      setRollingBack(false)
    }
  }

  let flowLabel = ''
  let dialogTitle = t('Update to {{version}}?', {
    version: latestVersion ?? '',
  })
  let confirmText = t('Update now')
  let dialogDesc = (
    <div className='space-y-2'>
      <p>
        {t(
          'The service restarts during the update, so the panel is briefly unavailable.'
        )}
      </p>
      <p>
        {t(
          'If the container is recreated, the panel falls back to the image version.'
        )}
      </p>
    </div>
  )

  if (phase === 'restarting') {
    dialogTitle = t('Restarting the service…')
    flowLabel =
      flow === 'rollback'
        ? t(
            'The rollback was applied. The service is restarting and this page reloads automatically once it answers.'
          )
        : t(
            'The update was applied. The service is restarting and this page reloads automatically once the new version answers — usually within a few seconds, at most about 90.'
          )
    confirmText = t('Restarting…')
    dialogDesc = (
      <div className='space-y-2' data-testid='update-restart-progress'>
        <p>{flowLabel}</p>
        <p data-testid='update-restart-persistence'>{persistenceHint}</p>
      </div>
    )
  } else if (phase === 'timed-out') {
    dialogTitle = t('Update may be complete')
    confirmText = t('Close')
    dialogDesc = (
      <div className='space-y-2' data-testid='update-restart-timeout'>
        <p>
          {t(
            'The update may already be finished. Reload this page manually to check the new version.'
          )}
        </p>
        <p data-testid='update-timeout-persistence'>{persistenceHint}</p>
      </div>
    )
  }

  return (
    <>
      <SettingsSection title={t('System maintenance')}>
        <div className='space-y-6'>
          <div className='grid gap-4 md:grid-cols-2'>
            <div className='rounded-lg border p-4'>
              <div className='text-muted-foreground text-sm'>
                {t('Current version')}
              </div>
              <div className='text-lg font-semibold'>{version}</div>
              <div
                data-testid='app-build-revision'
                className='text-muted-foreground mt-1 font-mono text-xs break-all'
              >
                {t('Build ID')}: {getBuildRevision()}
              </div>
            </div>
            <div className='rounded-lg border p-4'>
              <div className='text-muted-foreground text-sm'>
                {t('Latest version')}
              </div>
              <div
                data-testid='latest-version-value'
                className='text-lg font-semibold'
              >
                {latestVersion ?? t('Unknown')}
              </div>
            </div>
            <div className='rounded-lg border p-4'>
              <div className='text-muted-foreground text-sm'>
                {t('Update status')}
              </div>
              <div
                data-testid='update-status-text'
                className='text-lg font-semibold'
              >
                {statusText}
              </div>
            </div>
            <div className='rounded-lg border p-4'>
              <div className='text-muted-foreground text-sm'>
                {t('Uptime since')}
              </div>
              <div className='text-lg font-semibold'>{uptime}</div>
            </div>
          </div>

          <div className='space-y-3'>
            <div className='flex flex-wrap items-center gap-2'>
              <Button onClick={handleCheckUpdates} disabled={checking}>
                {checking ? (
                  t('Checking updates...')
                ) : (
                  <>
                    <RefreshCcwIcon className='me-2 h-4 w-4' />
                    {t('Check for updates')}
                  </>
                )}
              </Button>
              {canApplyUpdate && (
                <Button
                  variant='destructive'
                  data-testid='apply-update'
                  onClick={() => setConfirmOpen(true)}
                  disabled={applying || phase === 'restarting'}
                >
                  {applying || phase === 'restarting'
                    ? t('Updating...')
                    : t('Update now')}
                </Button>
              )}
            </div>
            {canRollback && (
              <Button
                variant='outline'
                data-testid='rollback-update'
                disabled={
                  rollingBack ||
                  rollbackKnownMissing ||
                  applying ||
                  phase === 'restarting'
                }
                onClick={() => setRollbackConfirmOpen(true)}
              >
                {rollingBack
                  ? t('Rolling back...')
                  : t('Roll back to the previous version')}
              </Button>
            )}
            {rollbackKnownMissing && (
              <p
                data-testid='rollback-unavailable-note'
                className='text-muted-foreground text-xs'
              >
                {t(
                  'No rollback point exists on this server yet — it is created by an in-panel update.'
                )}
              </p>
            )}
            {canApplyUpdate && (
              <p
                data-testid='update-not-persistent-hint'
                className='text-muted-foreground text-xs'
              >
                {persistenceHint}
              </p>
            )}
            {applyIsRootOnly && (
              <p
                data-testid='apply-root-only-note'
                className='text-muted-foreground text-xs'
              >
                {t(
                  'Only the root account can install updates from this panel. Use the image upgrade steps in the README instead.'
                )}
              </p>
            )}
            {updateConfirmed && !applyEnabled && (
              <p
                data-testid='apply-disabled-note'
                className='text-muted-foreground text-xs'
              >
                {t(
                  'In-panel updates are turned off by the administrator. Use the image upgrade steps in the README instead.'
                )}
              </p>
            )}
            {phase === 'timed-out' && (
              <p
                data-testid='update-restart-note'
                className='text-muted-foreground text-xs'
              >
                {t(
                  'The update may already be finished. Reload this page manually to check the new version.'
                )}
              </p>
            )}
            <p
              data-testid='upgrade-hint'
              className='text-muted-foreground text-xs'
            >
              <span className='text-foreground font-medium'>
                {t('How to upgrade')}
              </span>{' '}
              {t(
                'Pull the new image, remove the old container, then recreate it from the new image. See the upgrade steps in the README.'
              )}
            </p>
          </div>
        </div>
      </SettingsSection>

      <ConfirmDialog
        open={rollbackConfirmOpen}
        onOpenChange={(nextOpen) => {
          if (!nextOpen && phase === 'restarting') return
          setRollbackConfirmOpen(nextOpen)
        }}
        title={
          phase === 'restarting'
            ? t('Restarting the service…')
            : t('Roll back to the previous version?')
        }
        desc={
          <div className='space-y-2'>
            <p>
              {t(
                'The running binary is swapped back to the local rollback point. No old version is downloaded.'
              )}
            </p>
            <p data-testid='rollback-persistence'>{persistenceHint}</p>
          </div>
        }
        confirmText={
          phase === 'restarting' ? t('Restarting…') : t('Roll back now')
        }
        destructive
        isLoading={rollingBack || phase === 'restarting'}
        disabled={phase === 'restarting'}
        handleConfirm={handleConfirmRollback}
      />

      <ConfirmDialog
        open={confirmOpen}
        onOpenChange={(nextOpen) => {
          // While restarting the dialog is the progress indicator; dismissing it
          // would hide the only feedback the operator has.
          if (!nextOpen && phase === 'restarting') return
          setConfirmOpen(nextOpen)
        }}
        title={dialogTitle}
        desc={dialogDesc}
        confirmText={confirmText}
        destructive={phase !== 'timed-out'}
        isLoading={applying || phase === 'restarting'}
        disabled={phase === 'restarting'}
        handleConfirm={async () => {
          if (phase === 'timed-out') {
            // Manual-refresh path: closing is the end of it, never a reload.
            setConfirmOpen(false)
            return
          }
          await handleConfirmUpdate()
        }}
      />
    </>
  )
}
