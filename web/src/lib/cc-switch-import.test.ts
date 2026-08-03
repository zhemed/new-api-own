/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import { buildCCSwitchURL } from './cc-switch-import'

function decodeBase64Url(value: string): string {
  const base64 = value.replaceAll('-', '+').replaceAll('_', '/')
  const padded = base64 + '='.repeat((4 - (base64.length % 4)) % 4)
  const binary = atob(padded)
  const bytes = Uint8Array.from(binary, (char) => char.charCodeAt(0))
  return new TextDecoder().decode(bytes)
}

describe('CC Switch import contract', () => {
  test('includes the New API account usage contract for Codex', () => {
    const url = new URL(
      buildCCSwitchURL(
        'codex',
        'My Codex',
        { model: 'deepseek-v4-flash' },
        'sk-test-key',
        'https://cs.shemedhb.eu.org/'
      )
    )
    const params = url.searchParams
    const script = decodeBase64Url(params.get('usageScript') ?? '')

    assert.equal(url.protocol, 'ccswitch:')
    assert.equal(url.hostname, 'v1')
    assert.equal(url.pathname, '/import')
    assert.equal(params.get('endpoint'), 'https://cs.shemedhb.eu.org/v1')
    assert.equal(params.get('homepage'), 'https://cs.shemedhb.eu.org')
    assert.equal(params.get('usageEnabled'), 'true')
    assert.equal(params.get('usageApiKey'), 'sk-test-key')
    assert.equal(params.get('usageBaseUrl'), 'https://cs.shemedhb.eu.org')
    assert.equal(params.get('usageAutoInterval'), '30')
    assert.match(script, /\{\{baseUrl\}\}\/api\/usage\/account\//)
    assert.match(script, /Bearer \{\{apiKey\}\}/)
    assert.match(script, /quota/)
    assert.match(script, /used_quota/)
    assert.match(script, /total_quota/)
  })

  test('does not add Codex usage metadata to non-Codex imports', () => {
    const params = new URL(
      buildCCSwitchURL(
        'claude',
        'My Claude',
        { model: 'claude-sonnet' },
        'sk-test-key',
        'https://cs.shemedhb.eu.org'
      )
    ).searchParams

    assert.equal(params.get('endpoint'), 'https://cs.shemedhb.eu.org')
    assert.equal(params.get('usageEnabled'), null)
    assert.equal(params.get('usageScript'), null)
  })
})
