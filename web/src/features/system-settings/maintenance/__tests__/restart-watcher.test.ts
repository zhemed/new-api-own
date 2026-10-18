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

import { Window } from 'happy-dom'

// The one-shot guard reads sessionStorage, so give the module a DOM.
const domWindow = new Window()
Object.defineProperty(globalThis, 'window', {
  configurable: true,
  value: domWindow,
})
Object.defineProperty(globalThis, 'sessionStorage', {
  configurable: true,
  value: domWindow.sessionStorage,
})

const {
  waitForRestartedService,
  watchForRestartAndReload,
  shouldWatchAfterApply,
  hasReloadedFor,
  markReloadedFor,
  RESTART_WATCH_TIMEOUT_MS,
} = await import('../restart-watcher')

/** Virtual clock: `sleep` advances time instead of really waiting. */
function makeClock() {
  let elapsed = 0
  return {
    now: () => elapsed,
    sleep: async (ms: number) => {
      elapsed += ms
    },
    elapsed: () => elapsed,
  }
}

/** Probe that yields the given snapshots in order, then throws forever. */
function probeSequence(snapshots: Array<{ version?: string } | 'fail'>) {
  let index = 0
  return async () => {
    const next = snapshots[index++]
    if (next === undefined || next === 'fail') throw new Error('unreachable')
    return next
  }
}

afterEach(() => {
  sessionStorage.clear()
})

describe('waitForRestartedService', () => {
  test('reports the new version as soon as /api/status answers with it', async () => {
    const clock = makeClock()
    const outcome = await waitForRestartedService({
      previousVersion: '0.0.19',
      now: clock.now,
      sleep: clock.sleep,
      readStatus: probeSequence([
        'fail', // the server is still restarting
        'fail',
        { version: '0.0.20' },
      ]),
    })

    assert.deepEqual(outcome, { status: 'version-changed', version: '0.0.20' })
  })

  test('treats two consecutive same-version successes as recovered', async () => {
    const clock = makeClock()
    const outcome = await waitForRestartedService({
      previousVersion: '0.0.19',
      now: clock.now,
      sleep: clock.sleep,
      readStatus: probeSequence([
        { version: '0.0.19' }, // single success is not enough
        'fail',
        { version: '0.0.19' },
        { version: '0.0.19' },
      ]),
    })

    assert.deepEqual(outcome, { status: 'recovered', version: '0.0.19' })
  })

  test('a single success during shutdown does not count as recovery', async () => {
    const clock = makeClock()
    const outcome = await waitForRestartedService({
      previousVersion: '0.0.19',
      now: clock.now,
      sleep: clock.sleep,
      timeoutMs: 10_000,
      readStatus: probeSequence([
        { version: '0.0.19' },
        'fail',
        'fail',
        'fail',
      ]),
    })

    assert.deepEqual(outcome, { status: 'timeout' })
  })

  test('times out when the service never comes back', async () => {
    const clock = makeClock()
    const outcome = await waitForRestartedService({
      previousVersion: '0.0.19',
      now: clock.now,
      sleep: clock.sleep,
      readStatus: async () => {
        throw new Error('connection refused')
      },
    })

    assert.deepEqual(outcome, { status: 'timeout' })
    // backoff must stay bounded and stop at the timeout
    assert.equal(clock.elapsed() >= RESTART_WATCH_TIMEOUT_MS, true)
    assert.equal(clock.elapsed() < RESTART_WATCH_TIMEOUT_MS + 20_000, true)
  })
})

describe('watchForRestartAndReload', () => {
  test('reloads exactly once when the version changes', async () => {
    const clock = makeClock()
    const reloads: string[] = []

    const outcome = await watchForRestartAndReload({
      previousVersion: '0.0.19',
      now: clock.now,
      sleep: clock.sleep,
      readStatus: probeSequence(['fail', { version: '0.0.20' }]),
      reload: (version) => reloads.push(version),
    })

    assert.equal(outcome.status, 'version-changed')
    assert.deepEqual(reloads, ['0.0.20'])
  })

  test('never reloads when the watch times out', async () => {
    const clock = makeClock()
    const reloads: string[] = []

    const outcome = await watchForRestartAndReload({
      previousVersion: '0.0.19',
      now: clock.now,
      sleep: clock.sleep,
      readStatus: async () => {
        throw new Error('connection refused')
      },
      reload: (version) => reloads.push(version),
    })

    assert.equal(outcome.status, 'timeout')
    assert.deepEqual(reloads, [])
  })

  test('a repeated trigger for the same version does not reload twice', async () => {
    const clock = makeClock()
    const reloads: string[] = []
    const deps = {
      previousVersion: '0.0.19',
      now: clock.now,
      sleep: clock.sleep,
      readStatus: probeSequence([{ version: '0.0.20' }, { version: '0.0.20' }]),
      reload: (version: string) => reloads.push(version),
    }

    await watchForRestartAndReload(deps)
    await watchForRestartAndReload(deps)

    assert.deepEqual(reloads, ['0.0.20'])
    assert.equal(hasReloadedFor('0.0.20'), true)
  })

  test('a later update to another version is allowed to reload again', async () => {
    const clock = makeClock()
    const reloads: string[] = []

    await watchForRestartAndReload({
      previousVersion: '0.0.19',
      now: clock.now,
      sleep: clock.sleep,
      readStatus: probeSequence([{ version: '0.0.20' }]),
      reload: (version) => reloads.push(version),
    })
    await watchForRestartAndReload({
      previousVersion: '0.0.20',
      now: clock.now,
      sleep: clock.sleep,
      readStatus: probeSequence([{ version: '0.0.21' }]),
      reload: (version) => reloads.push(version),
    })

    assert.deepEqual(reloads, ['0.0.20', '0.0.21'])
  })

  test('the rebuilt page is not reloaded again for the version it already runs', async () => {
    markReloadedFor('0.0.20')
    const clock = makeClock()
    const reloads: string[] = []

    const outcome = await watchForRestartAndReload({
      previousVersion: '0.0.20',
      now: clock.now,
      sleep: clock.sleep,
      readStatus: probeSequence([{ version: '0.0.20' }, { version: '0.0.20' }]),
      reload: (version) => reloads.push(version),
    })

    assert.equal(outcome.status, 'recovered')
    assert.deepEqual(reloads, [])
  })
})

describe('shouldWatchAfterApply', () => {
  test('only a confirmed ok answer starts the watch', () => {
    assert.equal(shouldWatchAfterApply({ ok: true }), true)
    for (const reason of [
      'disabled',
      'busy',
      'not_newer',
      'download_failed',
      'checksum_failed',
      'unsupported_platform',
      'too_large',
      'replace_failed',
      'http_error',
    ]) {
      assert.equal(
        shouldWatchAfterApply({ ok: false, reason } as { ok: boolean }),
        false,
        `${reason} must not trigger a reload`
      )
    }
  })
})
