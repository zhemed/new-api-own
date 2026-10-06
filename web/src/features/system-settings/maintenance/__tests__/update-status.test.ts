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
import { describe, test } from 'node:test'

import { resolveUpdateAvailability } from '../update-status'

describe('resolveUpdateAvailability', () => {
  test('reports the server verdict when the check is on', () => {
    assert.equal(
      resolveUpdateAvailability({
        checkEnabled: true,
        serverState: 'update_available',
        comparison: 'newer',
      }),
      'update-available'
    )
    assert.equal(
      resolveUpdateAvailability({
        checkEnabled: true,
        serverState: 'up_to_date',
        comparison: 'equal',
      }),
      'up-to-date'
    )
  })

  test('treats a turned-off check as its own state, never as unknown', () => {
    assert.equal(
      resolveUpdateAvailability({
        checkEnabled: false,
        serverState: 'disabled',
        comparison: 'newer',
      }),
      'check-disabled'
    )
    // The switch wins even if a stale payload still reports a verdict.
    assert.equal(
      resolveUpdateAvailability({
        checkEnabled: false,
        serverState: 'update_available',
        comparison: 'newer',
      }),
      'check-disabled'
    )
  })

  test('honours the server state even when the switch field is missing', () => {
    assert.equal(
      resolveUpdateAvailability({
        checkEnabled: true,
        serverState: 'disabled',
        comparison: 'newer',
      }),
      'check-disabled'
    )
  })

  test('keeps a failed check unknown instead of calling it up to date', () => {
    for (const serverState of ['unknown', 'unavailable']) {
      assert.equal(
        resolveUpdateAvailability({
          checkEnabled: true,
          serverState,
          comparison: 'unknown',
        }),
        'unknown'
      )
    }
  })

  test('stays unknown when the status endpoint could not be reached', () => {
    assert.equal(
      resolveUpdateAvailability({
        checkEnabled: null,
        comparison: 'newer',
      }),
      'unknown'
    )
  })

  test('falls back to the local comparison when the server sent no verdict', () => {
    assert.equal(
      resolveUpdateAvailability({
        checkEnabled: true,
        serverState: null,
        comparison: 'newer',
      }),
      'update-available'
    )
    assert.equal(
      resolveUpdateAvailability({
        checkEnabled: true,
        serverState: undefined,
        comparison: 'equal',
      }),
      'up-to-date'
    )
    assert.equal(
      resolveUpdateAvailability({
        checkEnabled: true,
        comparison: 'unknown',
      }),
      'unknown'
    )
  })

  test('reports up to date when the published version is behind the build', () => {
    // Regression: a stale release feed must never be advertised as an update.
    assert.equal(
      resolveUpdateAvailability({
        checkEnabled: true,
        serverState: null,
        comparison: 'older',
      }),
      'up-to-date'
    )
  })
})
