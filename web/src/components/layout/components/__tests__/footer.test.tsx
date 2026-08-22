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

import DOMPurify from 'dompurify'
import { Window } from 'happy-dom'

// Setup DOMPurify with happy-dom window
const domWindow = new Window() as unknown as Window & typeof globalThis
const purify = DOMPurify(domWindow as unknown as Window)

describe('footer XSS sanitization', () => {
  test('sanitizes script tag injection', () => {
    const malicious = '<img src=x onerror=alert(1)><script>alert(1)</script><p>safe</p>'
    const sanitized = purify.sanitize(malicious)
    assert.equal(sanitized.includes('<script'), false)
    assert.equal(sanitized.includes('onerror'), false)
    assert.equal(sanitized.includes('<p>safe</p>'), true)
  })

  test('sanitizes svg onload injection', () => {
    const malicious = '<svg onload=alert(1)><p>safe</p></svg>'
    const sanitized = purify.sanitize(malicious)
    assert.equal(sanitized.includes('onload'), false)
    assert.equal(sanitized.includes('<p>safe</p>'), true)
  })

  test('preserves safe html attributes', () => {
    const safe = '<a href="https://example.com" target="_blank" rel="noopener noreferrer">link</a><p>safe</p>'
    const sanitized = purify.sanitize(safe)
    assert.equal(sanitized.includes('href="https://example.com"'), true)
    assert.equal(sanitized.includes('<p>safe</p>'), true)
  })
})
