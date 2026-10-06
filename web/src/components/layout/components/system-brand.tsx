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
import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { useStatus } from '@/hooks/use-status'
import { useSystemConfig } from '@/hooks/use-system-config'
import { cn } from '@/lib/utils'

/**
 * System brand shown in the top app bar: logo + system name + running version.
 *
 * The version is surfaced here on purpose — it used to live only in the
 * admin-only "System maintenance" settings section, so an operator had no way
 * to tell which build a panel was running while using the console.
 */
export function SystemBrand() {
  const { t } = useTranslation()
  const { status } = useStatus()
  const { logo } = useSystemConfig()

  const name = status?.system_name || 'New API'
  const version = status?.version?.trim()

  return (
    <Link
      to='/'
      aria-label={t('Go to home')}
      className={cn(
        'text-foreground inline-flex h-7 items-center gap-1.5 rounded-md px-1.5 text-sm font-medium transition-colors outline-none select-none',
        'hover:bg-accent focus-visible:ring-ring/40 focus-visible:ring-2'
      )}
    >
      <div className='flex size-5 items-center justify-center overflow-hidden rounded-md'>
        <img
          src={logo}
          alt={t('Logo')}
          className='size-full rounded-md object-cover'
        />
      </div>
      <span className='max-w-[12rem] truncate'>{name}</span>
      {version ? (
        <span
          data-testid='system-brand-version'
          className='text-muted-foreground shrink-0 text-xs font-normal tabular-nums'
        >
          {version}
        </span>
      ) : null}
    </Link>
  )
}
