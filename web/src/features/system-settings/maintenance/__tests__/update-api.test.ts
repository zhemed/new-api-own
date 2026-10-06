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
import assert from 'node:assert/strict'
import { afterEach, describe, test } from 'node:test'

import { api } from '@/lib/api'

import { applyUpdate, fetchUpdateStatus } from '../update-api'

type ApiGet = typeof api.get
type ApiPost = typeof api.post

const realGet = api.get
const realPost = api.post
const calls: Array<{ method: string; url: string }> = []

function stubGet(handler: () => unknown): void {
  api.get = (async (url: string) => {
    calls.push({ method: 'GET', url })
    return handler()
  }) as unknown as ApiGet
}

function stubPost(handler: () => unknown): void {
  api.post = (async (url: string) => {
    calls.push({ method: 'POST', url })
    return handler()
  }) as unknown as ApiPost
}

afterEach(() => {
  api.get = realGet
  api.post = realPost
  calls.length = 0
})

describe('fetchUpdateStatus', () => {
  test('reads the switch, the verdict and the published version', async () => {
    stubGet(() => ({
      data: {
        success: true,
        data: {
          enabled: true,
          state: 'update_available',
          latest_version: 'v0.0.7',
          has_update: true,
        },
      },
    }))

    const status = await fetchUpdateStatus()

    assert.deepEqual(status, {
      checkEnabled: true,
      serverState: 'update_available',
      latestVersion: 'v0.0.7',
      applyEnabled: null,
    })
    assert.deepEqual(calls, [{ method: 'GET', url: '/api/status/update' }])
  })

  test('prefers the new check_enabled/apply_enabled fields when present', async () => {
    stubGet(() => ({
      data: {
        data: {
          enabled: true,
          check_enabled: false,
          apply_enabled: false,
          state: 'update_available',
          latest_version: 'v0.0.7',
        },
      },
    }))

    const status = await fetchUpdateStatus()

    assert.equal(status?.checkEnabled, false)
    assert.equal(status?.applyEnabled, false)
  })

  test('keeps applyEnabled null when the server does not report the switch', async () => {
    stubGet(() => ({ data: { data: { enabled: true, state: 'unknown' } } }))

    const status = await fetchUpdateStatus()

    assert.equal(status?.applyEnabled, null)
  })

  test('reports the check as off when the server says so', async () => {
    stubGet(() => ({ data: { data: { enabled: false, state: 'disabled' } } }))

    const status = await fetchUpdateStatus()

    assert.equal(status?.checkEnabled, false)
    assert.equal(status?.serverState, 'disabled')
    assert.equal(status?.latestVersion, null)
  })

  test('treats a missing switch as off rather than guessing', async () => {
    stubGet(() => ({ data: { data: { state: 'unknown' } } }))

    const status = await fetchUpdateStatus()

    assert.equal(status?.checkEnabled, false)
  })

  test('resolves null when the endpoint answers without a payload', async () => {
    stubGet(() => ({ data: {} }))

    assert.equal(await fetchUpdateStatus(), null)
  })

  test('resolves null, without throwing, when the request fails', async () => {
    stubGet(() => {
      throw new Error('offline')
    })

    assert.equal(await fetchUpdateStatus(), null)
  })
})

describe('applyUpdate', () => {
  test('posts to the apply endpoint and reports the installed version', async () => {
    stubPost(() => ({
      data: { success: true, data: { applied: true, version: '0.0.7' } },
    }))

    const result = await applyUpdate()

    assert.deepEqual(result, { ok: true, version: '0.0.7' })
    assert.deepEqual(calls, [
      { method: 'POST', url: '/api/status/update/apply' },
    ])
  })

  test('maps the server error code onto a readable reason', async () => {
    stubPost(() => {
      throw {
        response: {
          data: {
            success: false,
            code: 'checksum_mismatch',
            message: 'bad sum',
          },
        },
      }
    })

    const result = await applyUpdate()

    assert.deepEqual(result, {
      ok: false,
      reason: 'checksum_failed',
      message: 'bad sum',
    })
  })

  test('keeps the apply switch failure distinguishable from a broken build', async () => {
    stubPost(() => {
      throw {
        response: {
          status: 403,
          data: { success: false, code: 'disabled', message: 'nope' },
        },
      }
    })

    const result = await applyUpdate()

    assert.equal(result.ok, false)
    assert.equal(result.ok === false && result.reason, 'disabled')
  })

  test('collapses several download codes into one reason', async () => {
    for (const code of [
      'download_failed',
      'source_unavailable',
      'asset_missing',
      'checksum_download_failed',
    ]) {
      stubPost(() => ({ data: { success: false, code } }))
      const result = await applyUpdate()
      assert.equal(result.ok === false && result.reason, 'download_failed')
    }
  })

  test('falls back to the server message for an unrecognised code', async () => {
    stubPost(() => ({
      data: {
        success: false,
        code: 'brand_new_code',
        message: 'kettle is busy',
      },
    }))

    const result = await applyUpdate()

    assert.deepEqual(result, {
      ok: false,
      reason: null,
      message: 'kettle is busy',
    })
  })

  test('never throws when the failure carries no detail at all', async () => {
    stubPost(() => {
      throw undefined
    })

    const result = await applyUpdate()

    assert.deepEqual(result, { ok: false, reason: null, message: '' })
  })
})
