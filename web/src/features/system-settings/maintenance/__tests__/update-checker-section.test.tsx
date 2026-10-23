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

/** Applying an update is root-only on the server. */
function signInAsRoot(): void {
  useAuthStore.getState().auth.setUser({
    id: 2,
    username: 'root',
    role: ROLE.SUPER_ADMIN,
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
  /** Replaces the 90s restart watcher so the test stays instantaneous. */
  postUpdateWatch?: (deps: {
    previousVersion: string
  }) => Promise<{ status: 'timeout' }>
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
        <UpdateCheckerSection
          currentVersion='0.0.6'
          startTime={null}
          postUpdateWatch={
            input.postUpdateWatch as unknown as
              | typeof import('../restart-watcher').watchForRestartAndReload
              | undefined
          }
        />
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

function rollbackButton(container: HTMLElement): HTMLButtonElement | null {
  return container.querySelector<HTMLButtonElement>(
    '[data-testid="rollback-update"]'
  )
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
  // Applying is root-only on the server (RootAuth, and system access tokens are
  // refused), so a plain admin must not be offered a button that only 403s.
  test('offers the apply button to the root account', async () => {
    signInAsRoot()
    const container = await renderUpdateChecker()

    assert.notEqual(applyButton(container), null)
    assert.equal(textOf(container, 'apply-root-only-note'), '')
  })

  test('hides the apply button from a non-root admin and explains why', async () => {
    signInAsAdmin()
    const container = await renderUpdateChecker()

    assert.equal(applyButton(container), null)
    assert.equal(
      textOf(container, 'apply-root-only-note'),
      en[
        'Only the root account can install updates from this panel. Use the image upgrade steps in the README instead.'
      ]
    )
  })

  test('a status-provided warning also wins over the local copy', async () => {
    signInAsRoot()
    const container = await renderUpdateChecker({
      status: { ...AVAILABLE, persistent: false, hint: 'STATUS HINT' },
    })

    assert.equal(textOf(container, 'update-not-persistent-hint'), 'STATUS HINT')
  })

  test('the non-persistence warning sits next to the button before applying', async () => {
    signInAsRoot()
    const container = await renderUpdateChecker()

    assert.equal(
      textOf(container, 'update-not-persistent-hint'),
      en[
        'This replaces the binary inside the running container. Recreating the container falls back to the version baked into the image.'
      ]
    )
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
    signInAsRoot()
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
    signInAsRoot()
    stubApplyUpdate(() => ({
      data: { success: true, data: { applied: true, version: '0.0.7' } },
    }))
    const watched: string[] = []
    const container = await renderUpdateChecker({
      postUpdateWatch: async (deps) => {
        watched.push(deps.previousVersion)
        return { status: 'timeout' }
      },
    })

    await clickButton(applyButton(container))
    await clickButton(confirmButton())

    assert.deepEqual(calls.at(-1), {
      method: 'POST',
      url: '/api/status/update/apply',
    })
    // the watch starts from the version the page was running
    assert.deepEqual(watched, ['0.0.6'])
    // a timeout hands the reload back to the operator instead of reloading
    assert.equal(
      container.querySelector('[data-testid="update-restart-note"]')
        ?.textContent,
      en[
        'The update may already be finished. Reload this page manually to check the new version.'
      ]
    )
  })

  test('a successful apply keeps the dialog open showing restart progress', async () => {
    signInAsRoot()
    stubApplyUpdate(() => ({
      data: { success: true, data: { applied: true, version: '0.0.7' } },
    }))

    let release: (() => void) | undefined
    const watched: string[] = []
    const container = await renderUpdateChecker({
      postUpdateWatch: async (deps) => {
        watched.push(deps.previousVersion)
        // hold the "restart in progress" state so the test can observe it
        await new Promise<void>((resolve) => {
          release = resolve
        })
        return { status: 'timeout' }
      },
    })

    await clickButton(applyButton(container))
    await clickButton(confirmButton())

    // the dialog must stay open, in the restarting state
    assert.notEqual(dialogContent(), null, 'confirm dialog was closed')
    assert.notEqual(
      document.body.querySelector('[data-testid="update-restart-progress"]'),
      null
    )
    assert.deepEqual(watched, ['0.0.6'])

    const restarting = dialogButton(en['Restarting…'])
    assert.equal(
      (restarting as HTMLButtonElement).disabled,
      true,
      'the primary action must be disabled while restarting'
    )

    // settle the pending watch so the test can finish
    await act(async () => {
      release?.()
      for (let tick = 0; tick < 8; tick++) await Promise.resolve()
    })
  })

  test('after the restart timeout the dialog asks for a manual refresh and closes cleanly', async () => {
    signInAsRoot()
    stubApplyUpdate(() => ({
      data: { success: true, data: { applied: true, version: '0.0.7' } },
    }))
    const container = await renderUpdateChecker({
      postUpdateWatch: async () => ({ status: 'timeout' }),
    })

    await clickButton(applyButton(container))
    await clickButton(confirmButton())

    assert.notEqual(dialogContent(), null, 'confirm dialog should stay open')
    assert.notEqual(
      document.body.querySelector('[data-testid="update-restart-timeout"]'),
      null
    )

    // closing the dialog ends the flow; it never reloads the page
    await clickButton(dialogButton(en['Close']))
    assert.equal(dialogContent(), null, 'dialog should be closed now')
  })

  test('a failed apply never starts the restart watch', async () => {
    signInAsRoot()
    stubApplyUpdate(() => ({
      data: { success: false, code: 'not_newer', message: 'same build' },
    }))
    const watched: string[] = []
    const container = await renderUpdateChecker({
      postUpdateWatch: async (deps) => {
        watched.push(deps.previousVersion)
        return { status: 'timeout' }
      },
    })

    await clickButton(applyButton(container))
    await clickButton(confirmButton())

    assert.deepEqual(watched, [])
    assert.equal(
      container.querySelector('[data-testid="update-restart-note"]'),
      null
    )
    // the dialog stays open so the operator can retry after fixing the cause
    assert.notEqual(dialogContent(), null, 'dialog should stay open on failure')
    assert.equal(
      (dialogContent()?.textContent ?? '').includes(en['Update now']),
      true
    )
  })

  test('explains a refused apply with the reason the server classified', async () => {
    signInAsRoot()
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

  test('explains a refused system-token call', async () => {
    signInAsRoot()
    stubApplyUpdate(() => {
      throw {
        response: {
          status: 403,
          data: { success: false, code: 'pat_not_allowed', message: '' },
        },
      }
    })
    const container = await renderUpdateChecker()

    await clickButton(applyButton(container))
    await clickButton(confirmButton())

    assert.equal(
      latestToastTitle(),
      en[
        'System access tokens cannot install updates. Sign in as the root account instead.'
      ]
    )
  })

  test('explains a signature mismatch instead of a generic failure', async () => {
    signInAsRoot()
    stubApplyUpdate(() => ({
      data: { success: false, code: 'signature_mismatch', message: '' },
    }))
    const container = await renderUpdateChecker()

    await clickButton(applyButton(container))
    await clickButton(confirmButton())

    assert.equal(
      latestToastTitle(),
      en[
        'The update signature did not match the published checksum, so the replacement was refused.'
      ]
    )
  })

  test('ends the dialog with the persistence warning after a successful apply', async () => {
    signInAsRoot()
    stubApplyUpdate(() => ({
      data: { success: true, data: { applied: true, version: '0.0.7' } },
    }))
    let release: (() => void) | undefined
    await renderUpdateChecker({
      postUpdateWatch: async () => {
        await new Promise<void>((resolve) => {
          release = resolve
        })
        return { status: 'timeout' }
      },
    })

    const container = document.body as unknown as HTMLElement
    await clickButton(applyButton(container))
    await clickButton(confirmButton())

    assert.equal(
      textOf(container, 'update-restart-persistence'),
      en[
        'This replaces the binary inside the running container. Recreating the container falls back to the version baked into the image.'
      ]
    )

    await act(async () => {
      release?.()
      for (let tick = 0; tick < 8; tick++) await Promise.resolve()
    })
  })

  test('prefers the server-authored warning over the local copy', async () => {
    signInAsRoot()
    stubApplyUpdate(() => ({
      data: {
        success: true,
        data: {
          applied: true,
          version: '0.0.7',
          persistent: false,
          hint: 'SERVER-AUTHORED HINT',
        },
      },
    }))
    let release: (() => void) | undefined
    await renderUpdateChecker({
      postUpdateWatch: async () => {
        await new Promise<void>((resolve) => {
          release = resolve
        })
        return { status: 'timeout' }
      },
    })

    const container = document.body as unknown as HTMLElement
    await clickButton(applyButton(container))
    await clickButton(confirmButton())

    assert.equal(
      textOf(container, 'update-restart-persistence'),
      'SERVER-AUTHORED HINT'
    )

    await act(async () => {
      release?.()
      for (let tick = 0; tick < 8; tick++) await Promise.resolve()
    })
  })

  test('falls back to the server message for an unclassified failure', async () => {
    signInAsRoot()
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
function confirmButton(label: string = en['Update now']): HTMLElement {
  const content = document.body.querySelector(
    '[data-slot="alert-dialog-content"]'
  )
  if (!content) throw new Error('confirm dialog did not open')
  for (const button of content.querySelectorAll('button')) {
    if (button.textContent?.trim() === label) return button
  }
  throw new Error(`confirm button was not rendered: ${label}`)
}

function dialogContent(): HTMLElement | null {
  return document.body.querySelector('[data-slot="alert-dialog-content"]')
}

/** Confirm-dialog button by its visible label. */
function dialogButton(label: string): HTMLElement {
  const content = dialogContent()
  if (!content) throw new Error('confirm dialog did not open')
  for (const button of content.querySelectorAll('button')) {
    if (button.textContent?.trim() === label) return button as HTMLElement
  }
  throw new Error(`dialog button not found: ${label}`)
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

describe('update checker rollback entry', () => {
  test('offers rollback to the root account only', async () => {
    signInAsRoot()
    const rootContainer = await renderUpdateChecker()
    assert.notEqual(rollbackButton(rootContainer), null)
    assert.equal(rollbackButton(rootContainer)?.disabled, false)

    signInAsAdmin()
    const adminContainer = await renderUpdateChecker()
    assert.equal(rollbackButton(adminContainer), null)
  })

  test('disables rollback and explains when the server reports no rollback point', async () => {
    signInAsRoot()
    const container = await renderUpdateChecker({
      status: { ...AVAILABLE, rollback_available: false },
    })

    assert.equal(rollbackButton(container)?.disabled, true)
    assert.equal(
      textOf(container, 'rollback-unavailable-note'),
      en[
        'No rollback point exists on this server yet — it is created by an in-panel update.'
      ]
    )
  })

  test('hides rollback when in-panel updates are turned off', async () => {
    signInAsRoot()
    const container = await renderUpdateChecker({
      status: { ...AVAILABLE, apply_enabled: false },
    })

    assert.equal(rollbackButton(container), null)
  })

  test('rolls back after a second confirmation and keeps the persistence warning', async () => {
    signInAsRoot()
    stubApplyUpdate(() => ({
      data: {
        success: true,
        data: {
          rolled_back: true,
          restart_mode: 'exec',
          persistent: false,
          hint: 'ROLLBACK HINT',
        },
      },
    }))
    let release: (() => void) | undefined
    await renderUpdateChecker({
      postUpdateWatch: async () => {
        await new Promise<void>((resolve) => {
          release = resolve
        })
        return { status: 'timeout' }
      },
    })

    const container = document.body as unknown as HTMLElement
    await clickButton(rollbackButton(container))
    assert.equal(
      textOf(container, 'rollback-persistence'),
      en[
        'This replaces the binary inside the running container. Recreating the container falls back to the version baked into the image.'
      ]
    )
    await clickButton(confirmButton(en['Roll back now']))

    assert.deepEqual(calls.at(-1), {
      method: 'POST',
      url: '/api/status/update/rollback',
    })
    assert.equal(textOf(container, 'rollback-persistence'), 'ROLLBACK HINT')

    await act(async () => {
      release?.()
      for (let tick = 0; tick < 8; tick++) await Promise.resolve()
    })
  })

  test('explains a refused rollback when there is nothing to roll back to', async () => {
    signInAsRoot()
    stubApplyUpdate(() => ({
      data: { success: false, code: 'no_rollback_point', message: '' },
    }))
    await renderUpdateChecker()
    const container = document.body as unknown as HTMLElement

    await clickButton(rollbackButton(container))
    await clickButton(confirmButton(en['Roll back now']))

    assert.equal(
      latestToastTitle(),
      en['No previous version is available to roll back to.']
    )
  })
})
