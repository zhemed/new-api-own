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
import { after, describe, test } from 'node:test'

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
const { QueryClient, QueryClientProvider } =
  await import('@tanstack/react-query')
const { RouterProvider, createMemoryHistory, createRootRoute, createRouter } =
  await import('@tanstack/react-router')
const { SystemBrand } = await import('../system-brand')

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: {
    en: {
      translation: {
        'Go to home': 'Go to home',
        Logo: 'Logo',
        'Unknown version': 'Unknown version',
      },
    },
  },
})

const reactTestGlobals = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean
}
reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = true

async function renderBrand(statusData: Record<string, unknown> | null) {
  const container = document.createElement('div')
  document.body.append(container)
  const root = createRoot(container)
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: Infinity } },
  })
  if (statusData) {
    // `getStatus()` returns the flattened `/api/status` payload, so seed the
    // cache with that exact shape. A fresh `updatedAt` keeps react-query from
    // background-refetching, so the component renders the payload under test.
    queryClient.setQueryData(['status'], statusData, { updatedAt: Date.now() })
  }

  // `SystemBrand` renders a router `Link`, so it needs router context.
  const rootRoute = createRootRoute({
    component: () => (
      <QueryClientProvider client={queryClient}>
        <I18nextProvider i18n={i18n}>
          <SystemBrand />
        </I18nextProvider>
      </QueryClientProvider>
    ),
  })
  const router = createRouter({
    routeTree: rootRoute,
    history: createMemoryHistory({ initialEntries: ['/'] }),
  })

  await act(async () => root.render(<RouterProvider router={router} />))
  await act(async () => {
    await router.load()
  })

  return { container, root }
}

describe('system brand version visibility', () => {
  after(() => {
    domWindow.close()
  })

  test('shows the running version reported by the backend', async () => {
    const { container, root } = await renderBrand({
      system_name: 'My Gateway',
      version: '0.0.6',
    })

    const version = container.querySelector(
      '[data-testid="system-brand-version"]'
    )
    assert.ok(version, 'the header must surface the running version')
    assert.equal(version.textContent, '0.0.6')
    assert.equal(container.textContent?.includes('My Gateway'), true)

    await act(async () => root.unmount())
    container.remove()
  })

  test('renders no version chip when the backend reports none', async () => {
    const { container, root } = await renderBrand({ system_name: 'My Gateway' })

    assert.equal(
      container.querySelector('[data-testid="system-brand-version"]'),
      null
    )
    assert.equal(container.textContent?.includes('My Gateway'), true)

    await act(async () => root.unmount())
    container.remove()
  })
})
