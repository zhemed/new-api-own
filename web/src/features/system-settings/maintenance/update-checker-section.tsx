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

import { SettingsSection } from '../components/settings-section'
import { applyUpdate } from './update-api'
import { useUpdateStatus } from './use-update-status'

type UpdateCheckerSectionProps = {
  currentVersion?: string | null
  startTime?: number | null
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
}: UpdateCheckerSectionProps) {
  const { t } = useTranslation()
  const isAdmin = useIsAdmin()
  const { latestVersion, availability, applyEnabled, checking, refresh } =
    useUpdateStatus(currentVersion)
  const [confirmOpen, setConfirmOpen] = useState(false)
  const [applying, setApplying] = useState(false)

  const uptime = startTime ? formatTimestamp(startTime) : t('Unknown')
  const version = currentVersion || t('Unknown')
  const updateConfirmed = availability === 'update-available'
  const canApplyUpdate = isAdmin && updateConfirmed && applyEnabled

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
        setConfirmOpen(false)
        toast.success(
          t(
            'Update started. The service will restart shortly, then reload this page.'
          )
        )
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
                  disabled={applying}
                >
                  {applying ? t('Updating...') : t('Update now')}
                </Button>
              )}
            </div>
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
        open={confirmOpen}
        onOpenChange={setConfirmOpen}
        title={t('Update to {{version}}?', { version: latestVersion ?? '' })}
        desc={
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
        }
        confirmText={t('Update now')}
        destructive
        isLoading={applying}
        handleConfirm={handleConfirmUpdate}
      />
    </>
  )
}
