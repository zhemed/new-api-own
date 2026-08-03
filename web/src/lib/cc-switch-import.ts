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
import { getSecureServerOrigin } from './channel-connection-info'

export const CC_SWITCH_USAGE_SCRIPT = `({
  request: {
    url: "{{baseUrl}}/api/usage/account/",
    method: "GET",
    headers: {
      "Content-Type": "application/json",
      "Authorization": "Bearer {{apiKey}}"
    }
  },
  extractor: function (response) {
    if (response && response.success && response.data) {
      return {
        planName: response.data.group || "Default",
        remaining: response.data.quota / 500000,
        used: response.data.used_quota / 500000,
        total: response.data.total_quota
          ? response.data.total_quota / 500000
          : (response.data.quota + response.data.used_quota) / 500000,
        unit: response.data.unit || "USD"
      };
    }
    return {
      isValid: false,
      invalidMessage: (response && response.message) || "Query failed"
    };
  }
})`

function encodeBase64Url(value: string): string {
  let binary = ''
  for (const byte of new TextEncoder().encode(value)) {
    binary += String.fromCharCode(byte)
  }
  return btoa(binary)
    .replaceAll('+', '-')
    .replaceAll('/', '_')
    .replace(/=+$/, '')
}

export function buildCCSwitchURL(
  app: string,
  name: string,
  models: Record<string, string>,
  apiKey: string,
  serverAddress = getSecureServerOrigin()
): string {
  const origin = serverAddress.replace(/\/+$/, '')
  const endpoint = app === 'codex' ? `${origin}/v1` : origin
  const params = new URLSearchParams()
  params.set('resource', 'provider')
  params.set('app', app)
  params.set('name', name)
  params.set('endpoint', endpoint)
  params.set('apiKey', apiKey)
  for (const [k, v] of Object.entries(models)) {
    if (v) params.set(k, v)
  }
  params.set('homepage', origin)
  if (app === 'codex') {
    params.set('usageEnabled', 'true')
    params.set('usageScript', encodeBase64Url(CC_SWITCH_USAGE_SCRIPT))
    params.set('usageApiKey', apiKey)
    params.set('usageBaseUrl', origin)
    params.set('usageAutoInterval', '30')
  }
  params.set('enabled', 'true')
  return `ccswitch://v1/import?${params.toString()}`
}
