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
export const CHANNEL_CONNECTION_INFO_TYPE = 'newapi_channel_conn'

export function getSecureServerOrigin(): string {
  if (
    typeof window !== 'undefined' &&
    window.location.protocol === 'https:'
  ) {
    return window.location.origin
  }

  try {
    if (typeof window !== 'undefined') {
      const raw = window.localStorage.getItem('status')
      if (raw) {
        const status = JSON.parse(raw)
        if (typeof status.server_address === 'string') {
          const configured = new URL(status.server_address)
          if (configured.protocol === 'https:') return configured.origin
        }
      }
    }
  } catch {
    /* empty */
  }

  // Fallback: use the origin the user is currently viewing
  // (covers HTTP / IP-based deployments instead of a hardcoded domain).
  if (typeof window !== 'undefined') {
    return window.location.origin
  }
  return ''
}

export type ChannelConnectionInfo = {
  key: string
  url: string
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}

export function encodeChannelConnectionInfo(key: string, url: string): string {
  return JSON.stringify({
    _type: CHANNEL_CONNECTION_INFO_TYPE,
    key,
    url,
  })
}

export function parseChannelConnectionInfo(
  text: string | null | undefined
): ChannelConnectionInfo | null {
  if (!text || typeof text !== 'string') return null

  try {
    const parsed: unknown = JSON.parse(text.trim())
    if (
      isRecord(parsed) &&
      parsed._type === CHANNEL_CONNECTION_INFO_TYPE &&
      typeof parsed.key === 'string' &&
      typeof parsed.url === 'string'
    ) {
      return { key: parsed.key, url: parsed.url }
    }
  } catch {
    /* not valid connection info JSON */
  }

  return null
}
