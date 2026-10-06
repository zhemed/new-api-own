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
import { readFileSync } from 'node:fs'
import { after, afterEach, describe, test } from 'node:test'

import { Window } from 'happy-dom'

const domWindow = new Window()
const domGlobals = [
  'window',
  'document',
  'navigator',
  'HTMLElement',
  'HTMLButtonElement',
  'SVGElement',
  'Node',
  'Element',
  'Event',
  'CustomEvent',
  'MutationObserver',
  'ResizeObserver',
  'requestAnimationFrame',
  'cancelAnimationFrame',
  'getComputedStyle',
] as const

for (const key of domGlobals) {
  Object.defineProperty(globalThis, key, {
    configurable: true,
    value: domWindow[key],
  })
}

const { act } = await import('react')
const { createRoot } = await import('react-dom/client')
const { createInstance } = await import('i18next')
const { I18nextProvider, initReactI18next } = await import('react-i18next')
const { UpdateCheckerSection } = await import('../update-checker-section')
const { useAuthStore } = await import('@/stores/auth-store')
const { ROLE } = await import('@/lib/roles')
const { api } = await import('@/lib/api')
const { toast } = await import('sonner')

// Expectation copy comes from the real locale file, so the assertions pin the
// i18n key the panel picked rather than the English wording here.
const en = (
  JSON.parse(
    readFileSync(
      new URL('../../../../i18n/locales/en.json', import.meta.url),
      'utf8'
    )
  ) as { translation: Record<string, string> }
).translation

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: { en: { translation: en } },
})

const reactTestGlobals = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean
}
reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = true

const realGet = api.get
const realPost = api.post
const calls: Array<{ method: string; url: string }> = []
const roots: Array<{ unmount: () => void }> = []

after(async () => {
  i18n.removeResourceBundle('en', 'translation')
})

afterEach(async () => {
  // Unmount and reset shared state inside act: both re-render the tree.
  await act(async () => {
    for (const root of roots.splice(0)) root.unmount()
    useAuthStore.getState().auth.setUser(null)
    toast.dismiss()
  })
  api.get = realGet
  api.post = realPost
  calls.length = 0
})

const UNKNOWN_STATUS = en['Unable to determine the latest version.']
const DISABLED_STATUS =
  en[
    'Update checks are turned off. Set UPDATE_CHECK_ENABLED=true on the server to turn them back on.'
  ]
const APPLY_DISABLED_NOTE =
  en[
    'In-panel updates are turned off by the administrator. Use the image upgrade steps in the README instead.'
  ]

function signInAsAdmin(): void {
  useAuthStore.getState().auth.setUser({
    id: 1,
    username: 'operator',
    role: ROLE.ADMIN,
  })
}

/** Canned answer for `GET /api/status/update`. */
function stubUpdateStatus(statusData: Record<string, unknown> | null): void {
  api.get = (async (url: string) => {
    calls.push({ method: 'GET', url })
    if (statusData === null) throw new Error('status endpoint unavailable')
    return { data: { success: true, data: statusData } }
  }) as unknown as typeof api.get
}

function stubApplyUpdate(handler: () => unknown): void {
  api.post = (async (url: string) => {
    calls.push({ method: 'POST', url })
    return handler()
  }) as unknown as typeof api.post
}

type RenderInput = {
  /** Payload for `GET /api/status/update`; `null` = the endpoint fails. */
  status?: Record<string, unknown> | null
}

async function renderUpdateChecker(input: RenderInput = {}) {
  stubUpdateStatus(input.status === undefined ? AVAILABLE : input.status)

  const container = document.createElement('div')
  document.body.append(container)
  const root = createRoot(container)
  roots.push(root)

  await act(async () => {
    root.render(
      <I18nextProvider i18n={i18n}>
        <UpdateCheckerSection currentVersion='0.0.6' startTime={null} />
      </I18nextProvider>
    )
    // Drain the mount-time status request so the panel has settled.
    for (let tick = 0; tick < 12; tick++) await Promise.resolve()
  })

  return container
}

function textOf(container: HTMLElement, testId: string): string {
  return container.querySelector(`[data-testid="${testId}"]`)?.textContent ?? ''
}

function applyButton(container: HTMLElement): HTMLElement | null {
  return container.querySelector<HTMLElement>('[data-testid="apply-update"]')
}

const AVAILABLE = {
  check_enabled: true,
  apply_enabled: true,
  state: 'update_available',
  latest_version: 'v0.0.7',
}

describe('update checker states', () => {
  test('announces the new version when the server confirmed one', async () => {
    const container = await renderUpdateChecker()

    assert.equal(textOf(container, 'latest-version-value'), 'v0.0.7')
    assert.equal(
      textOf(container, 'update-status-text'),
      en['New version available: {{version}}'].replace('{{version}}', 'v0.0.7')
    )
  })

  test('reports up to date when the server says the build is current', async () => {
    const container = await renderUpdateChecker({
      status: { enabled: true, state: 'up_to_date', latest_version: 'v0.0.6' },
    })

    assert.equal(textOf(container, 'update-status-text'), en['Up to date'])
  })

  test('stays neutral when the server could not tell', async () => {
    const container = await renderUpdateChecker({
      status: { enabled: true, state: 'unavailable', latest_version: '' },
    })

    assert.equal(textOf(container, 'update-status-text'), UNKNOWN_STATUS)
  })

  test('says the check is turned off instead of saying it cannot tell', async () => {
    const container = await renderUpdateChecker({
      status: { enabled: false, state: 'disabled' },
    })

    assert.equal(textOf(container, 'update-status-text'), DISABLED_STATUS)
    assert.notEqual(textOf(container, 'update-status-text'), UNKNOWN_STATUS)
    assert.equal(applyButton(container), null)
  })

  test('stays neutral when the status endpoint itself is unreachable', async () => {
    const container = await renderUpdateChecker({ status: null })

    assert.equal(textOf(container, 'update-status-text'), UNKNOWN_STATUS)
    assert.equal(applyButton(container), null)
  })

  test('does not claim an update for a published version that is behind', async () => {
    // Regression for the production report "new version available: v0.0.5"
    // on a panel that was running 0.0.6: the server compares, not the panel.
    const container = await renderUpdateChecker({
      status: { enabled: true, state: 'up_to_date', latest_version: 'v0.0.5' },
    })

    assert.equal(textOf(container, 'update-status-text'), en['Up to date'])
    assert.equal(
      textOf(container, 'update-status-text').includes('v0.0.5'),
      false
    )
  })

  test('checks our own backend once on mount, and never a release host', async () => {
    await renderUpdateChecker()

    assert.deepEqual(calls, [{ method: 'GET', url: '/api/status/update' }])
  })
})

describe('update checker apply flow', () => {
  test('offers the apply button to an admin when the server confirmed an update', async () => {
    signInAsAdmin()
    const container = await renderUpdateChecker()

    assert.notEqual(applyButton(container), null)
  })

  test('hides the apply button from an operator without admin rights', async () => {
    const container = await renderUpdateChecker()

    assert.equal(applyButton(container), null)
  })

  test('replaces the button with an explanation when the administrator turned apply off', async () => {
    signInAsAdmin()
    const container = await renderUpdateChecker({
      status: { ...AVAILABLE, apply_enabled: false },
    })

    assert.equal(applyButton(container), null)
    assert.equal(textOf(container, 'apply-disabled-note'), APPLY_DISABLED_NOTE)
  })

  test('keeps the apply button hidden while the state is unknown', async () => {
    signInAsAdmin()
    const container = await renderUpdateChecker({
      status: { enabled: true, state: 'unknown', latest_version: '' },
    })

    assert.equal(applyButton(container), null)
  })

  test('asks for confirmation, warning about the restart and image fallback', async () => {
    signInAsAdmin()
    const container = await renderUpdateChecker()

    await clickButton(applyButton(container))

    const dialog = document.body.querySelector(
      '[data-slot="alert-dialog-content"]'
    )
    const text = dialog?.textContent ?? ''
    assert.equal(
      text.includes(
        en[
          'The service restarts during the update, so the panel is briefly unavailable.'
        ]
      ),
      true
    )
    assert.equal(
      text.includes(
        en[
          'If the container is recreated, the panel falls back to the image version.'
        ]
      ),
      true
    )
    assert.equal(
      calls.some((call) => call.method === 'POST'),
      false
    )
  })

  test('applies the update on our own backend after confirmation', async () => {
    signInAsAdmin()
    stubApplyUpdate(() => ({
      data: { success: true, data: { applied: true, version: '0.0.7' } },
    }))
    const container = await renderUpdateChecker()

    await clickButton(applyButton(container))
    await clickButton(confirmButton())

    assert.deepEqual(calls.at(-1), {
      method: 'POST',
      url: '/api/status/update/apply',
    })
    assert.equal(
      latestToastTitle(),
      en[
        'Update started. The service will restart shortly, then reload this page.'
      ]
    )
  })

  test('explains a refused apply with the reason the server classified', async () => {
    signInAsAdmin()
    stubApplyUpdate(() => {
      throw {
        response: {
          status: 403,
          data: { success: false, code: 'disabled', message: 'nope' },
        },
      }
    })
    const container = await renderUpdateChecker()

    await clickButton(applyButton(container))
    await clickButton(confirmButton())

    assert.equal(latestToastTitle(), APPLY_DISABLED_NOTE)
  })

  test('falls back to the server message for an unclassified failure', async () => {
    signInAsAdmin()
    stubApplyUpdate(() => ({
      data: { success: false, message: 'checksum mismatch' },
    }))
    const container = await renderUpdateChecker()

    await clickButton(applyButton(container))
    await clickButton(confirmButton())

    assert.equal(latestToastTitle(), 'checksum mismatch')
  })
})

/** The confirm button lives in a portal, outside the component's container. */
function confirmButton(): HTMLElement {
  const content = document.body.querySelector(
    '[data-slot="alert-dialog-content"]'
  )
  if (!content) throw new Error('confirm dialog did not open')
  for (const button of content.querySelectorAll('button')) {
    if (button.textContent?.trim() === en['Update now']) return button
  }
  throw new Error('confirm button was not rendered')
}

function latestToastTitle(): string {
  const last = toast.getHistory().at(-1)
  if (!last || !('title' in last)) return ''
  return String(last.title)
}

async function clickButton(button: HTMLElement | null) {
  if (!button) throw new Error('button was not rendered')
  await act(async () => {
    button.click()
    for (let tick = 0; tick < 12; tick++) await Promise.resolve()
  })
}
