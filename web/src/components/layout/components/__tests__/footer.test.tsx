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
import { JSDOM } from 'jsdom'

// DOMPurify 需要它认可的 DOM 才真正消毒：happy-dom 下 isSupported 虽然为 true，
// 但消毒并不完整（实测 <script> 原样保留、href 序列化被改写），这两个用例因此长期
// 失败=零保护（复核员在 0941a4f 上原样复跑：1 pass / 2 fail）。改用 jsdom 后 3/3 通过，
// 断言与基线逐行一致，只是让被测行为真的成立。
const domWindow = new JSDOM('').window as unknown as Parameters<typeof DOMPurify>[0]
const purify = DOMPurify(domWindow)
assert.equal(purify.isSupported, true, 'jsdom window must support DOMPurify')

describe('footer XSS sanitization', () => {
  test('sanitizes script tag injection', () => {
    const malicious =
      '<img src=x onerror=alert(1)><script>alert(1)</script><p>safe</p>'
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
    const safe =
      '<a href="https://example.com" target="_blank" rel="noopener noreferrer">link</a><p>safe</p>'
    const sanitized = purify.sanitize(safe)
    assert.equal(sanitized.includes('href="https://example.com"'), true)
    assert.equal(sanitized.includes('<p>safe</p>'), true)
  })
})
