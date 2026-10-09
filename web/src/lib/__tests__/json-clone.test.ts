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

import { cloneJsonValue } from '../json-clone'

describe('cloneJsonValue — undefined handling', () => {
  test('drops an own property whose value is undefined', () => {
    const cloned = cloneJsonValue({ keep: 1, drop: undefined })

    assert.deepStrictEqual(cloned, { keep: 1 })
    assert.equal(Object.hasOwn(cloned, 'drop'), false)
  })

  test('turns an undefined array element into null', () => {
    assert.deepStrictEqual(cloneJsonValue([1, undefined, 3]), [1, null, 3])
  })

  test('turns a sparse array hole into null', () => {
    const sparse: unknown[] = []
    sparse[0] = 'a'
    sparse[2] = 'c'

    assert.deepStrictEqual(cloneJsonValue(sparse), ['a', null, 'c'])
  })
})

describe('cloneJsonValue — number normalization', () => {
  test('turns NaN, Infinity and -Infinity into null', () => {
    assert.deepStrictEqual(
      cloneJsonValue({ nan: Number.NaN, inf: Infinity, ninf: -Infinity }),
      { nan: null, inf: null, ninf: null }
    )
  })

  test('normalizes negative zero to zero', () => {
    assert.equal(Object.is(cloneJsonValue(-0), 0), true)
  })

  test('copies finite numbers as-is', () => {
    assert.deepStrictEqual(cloneJsonValue({ int: 7, float: 1.5 }), {
      int: 7,
      float: 1.5,
    })
  })
})

describe('cloneJsonValue — values JSON cannot represent', () => {
  test('drops a function-valued property but nulls a function array element', () => {
    const cloned = cloneJsonValue({ keep: 1, fn: () => 'x' })
    assert.deepStrictEqual(cloned, { keep: 1 })
    assert.deepStrictEqual(cloneJsonValue([() => 'x']), [null])
  })

  test('drops a symbol-valued property but nulls a symbol array element', () => {
    const symbol = Symbol('s')
    const cloned = cloneJsonValue({ keep: 1, sym: symbol })
    assert.deepStrictEqual(cloned, { keep: 1 })
    assert.deepStrictEqual(cloneJsonValue([symbol]), [null])
  })

  test('ignores symbol-keyed and non-enumerable properties', () => {
    const source: Record<string | symbol, unknown> = { visible: 1 }
    source[Symbol('hidden')] = 2
    Object.defineProperty(source, 'nonEnumerable', {
      value: 3,
      enumerable: false,
    })

    assert.deepStrictEqual(cloneJsonValue(source), { visible: 1 })
  })

  test('throws TypeError for BigInt', () => {
    assert.throws(() => cloneJsonValue({ big: BigInt(1) }), TypeError)
  })

  test('throws TypeError for an unrepresentable root value', () => {
    assert.throws(() => cloneJsonValue(undefined), TypeError)
    assert.throws(() => cloneJsonValue(() => 'x'), TypeError)
    assert.throws(() => cloneJsonValue(Symbol('s')), TypeError)
  })
})

describe('cloneJsonValue — dates and toJSON', () => {
  test('converts a Date to its ISO string', () => {
    const cloned = cloneJsonValue({ at: new Date('2024-01-02T03:04:05.000Z') })

    assert.deepStrictEqual(cloned, { at: '2024-01-02T03:04:05.000Z' })
  })

  test('honours a custom toJSON', () => {
    const source = {
      plain: 1,
      custom: { toJSON: () => ({ projected: true }) },
    }

    assert.deepStrictEqual(cloneJsonValue(source), {
      plain: 1,
      custom: { projected: true },
    })
  })

  test('turns an invalid Date into null', () => {
    assert.deepStrictEqual(cloneJsonValue({ at: new Date('nope') }), {
      at: null,
    })
  })
})

describe('cloneJsonValue — object kinds', () => {
  test('expands a Map and a Set into empty objects', () => {
    assert.deepStrictEqual(
      cloneJsonValue({
        map: new Map([['a', 1]]),
        set: new Set([1, 2]),
      }),
      { map: {}, set: {} }
    )
  })

  test('expands a class instance into its own enumerable properties', () => {
    class Route {
      incoming = '/v1/messages'
      upstream = '/v1/messages'
    }

    assert.deepStrictEqual(cloneJsonValue(new Route()), {
      incoming: '/v1/messages',
      upstream: '/v1/messages',
    })
  })

  test('keeps null as null and passes primitives through', () => {
    assert.deepStrictEqual(
      cloneJsonValue({ nil: null, str: 'a', bool: false }),
      { nil: null, str: 'a', bool: false }
    )
  })
})

describe('cloneJsonValue — structure', () => {
  test('clones nested objects and arrays deeply', () => {
    const source = {
      routes: [
        { path: '/a', tags: ['x', 'y'], meta: { depth: 2 } },
        { path: '/b', tags: [], meta: {} },
      ],
    }

    assert.deepStrictEqual(cloneJsonValue(source), {
      routes: [
        { path: '/a', tags: ['x', 'y'], meta: { depth: 2 } },
        { path: '/b', tags: [], meta: {} },
      ],
    })
  })

  test('detaches the clone from the source', () => {
    const nested = { value: 1 }
    const source = { nested, list: [nested] }
    const cloned = cloneJsonValue(source)

    cloned.nested.value = 99
    assert.equal(source.nested.value, 1)
    assert.notEqual(cloned.nested, source.nested)
    assert.notEqual(cloned.list, source.list)
  })

  test('keeps a shared (non-circular) reference duplicated, not rejected', () => {
    const shared = { id: 1 }
    const cloned = cloneJsonValue({ first: shared, second: shared })

    assert.deepStrictEqual(cloned, { first: { id: 1 }, second: { id: 1 } })
    assert.notEqual(cloned.first, cloned.second)
  })

  test('does not mutate the source, including undefined-valued keys', () => {
    const source: Record<string, unknown> = { keep: 1, drop: undefined }
    const list: unknown[] = [1, undefined]

    cloneJsonValue(source)
    cloneJsonValue(list)

    assert.equal(Object.hasOwn(source, 'drop'), true)
    assert.equal(source.drop, undefined)
    assert.equal(list.length, 2)
    assert.equal(list[1], undefined)
  })

  test('throws TypeError on a circular reference', () => {
    const circular: Record<string, unknown> = {}
    circular.self = circular

    assert.throws(() => cloneJsonValue(circular), TypeError)
  })

  test('throws TypeError on a circular reference nested in an array', () => {
    const list: unknown[] = []
    list.push({ backTo: list })

    assert.throws(() => cloneJsonValue(list), TypeError)
  })
})
