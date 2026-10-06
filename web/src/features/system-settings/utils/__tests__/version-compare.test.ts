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

import { compareVersions, normalizeVersion } from '../version-compare'

describe('normalizeVersion', () => {
  test('strips a leading v prefix in either case', () => {
    assert.equal(normalizeVersion('v0.0.6'), '0.0.6')
    assert.equal(normalizeVersion('V0.0.6'), '0.0.6')
  })

  test('keeps an unprefixed version untouched and trims whitespace', () => {
    assert.equal(normalizeVersion(' 0.0.6 '), '0.0.6')
  })

  test('drops build metadata but keeps the pre-release suffix', () => {
    assert.equal(normalizeVersion('v0.0.6+build.9'), '0.0.6')
    assert.equal(normalizeVersion('v0.0.6-beta.1'), '0.0.6-beta.1')
  })
})

describe('compareVersions', () => {
  test('reports equal when the release tag only differs by the v prefix', () => {
    assert.equal(compareVersions('0.0.6', 'v0.0.6'), 'equal')
  })

  test('reports equal for padding differences such as 0.0 versus 0.0.0', () => {
    assert.equal(compareVersions('0.0.6', 'v0.0.6.0'), 'equal')
  })

  test('reports newer when the published release is actually ahead', () => {
    assert.equal(compareVersions('0.0.6', 'v0.0.7'), 'newer')
    assert.equal(compareVersions('0.0.6', 'v0.1.0'), 'newer')
    assert.equal(compareVersions('0.0.6', 'v1.0.0'), 'newer')
  })

  test('compares numerically rather than lexicographically', () => {
    // "0.0.10" > "0.0.9" holds numerically but not as a raw string compare.
    assert.equal(compareVersions('0.0.9', 'v0.0.10'), 'newer')
    assert.equal(compareVersions('0.0.10', 'v0.0.9'), 'older')
  })

  test('reports older when the running build is ahead of the release', () => {
    assert.equal(compareVersions('0.0.7', 'v0.0.6'), 'older')
  })

  test('treats the historical v0.0.x tag format as an equal match', () => {
    assert.equal(compareVersions('v0.0.4', 'v0.0.4'), 'equal')
    assert.equal(compareVersions('0.0.4', 'v0.0.4'), 'equal')
  })

  test('ranks a pre-release below its final release', () => {
    assert.equal(compareVersions('0.0.6-beta.1', 'v0.0.6'), 'newer')
    assert.equal(compareVersions('0.0.6', 'v0.0.6-beta.1'), 'older')
  })

  test('reports unknown when either side is missing', () => {
    assert.equal(compareVersions('0.0.6', undefined), 'unknown')
    assert.equal(compareVersions(null, 'v0.0.6'), 'unknown')
    assert.equal(compareVersions('', 'v0.0.6'), 'unknown')
  })
})
