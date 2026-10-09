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
/**
 * Deep clone a value with **exactly** the semantics of
 * `JSON.parse(JSON.stringify(value))`.
 *
 * This exists so callers can express "I want the JSON view of this value"
 * without a round-trip through a string, and without having to suppress
 * `unicorn(prefer-structured-clone)` — `structuredClone` is *not* equivalent
 * (it preserves `Date`/`NaN`/`undefined`-valued keys instead of normalizing
 * or dropping them).
 *
 * Contract (identical to `JSON.parse(JSON.stringify(value))`):
 *
 * | Input                                        | Result                                              |
 * | -------------------------------------------- | --------------------------------------------------- |
 * | own property whose value is `undefined`      | the property is dropped                             |
 * | `undefined` array element / sparse array hole| `null`                                              |
 * | `Date`                                       | ISO string (via `toJSON`)                           |
 * | `NaN` / `Infinity` / `-Infinity`             | `null`                                              |
 * | `-0`                                         | `0`                                                 |
 * | function / `Symbol` as a property value      | the property is dropped                             |
 * | function / `Symbol` as an array element      | `null`                                              |
 * | `BigInt`                                     | throws `TypeError`                                  |
 * | `Map` / `Set` / class instance               | expanded by own enumerable string keys (`{}` for `Map`/`Set`) |
 * | `null` / `string` / `boolean` / finite number| copied as-is                                        |
 * | nested object / array                        | cloned recursively; `toJSON()` is called when present |
 * | non-enumerable property / `Symbol` key       | ignored                                             |
 * | circular reference                           | throws `TypeError`                                  |
 * | the input value                              | never mutated (pure)                                |
 *
 * Deliberate deviation: a **root** value that JSON cannot represent at all
 * (`undefined`, a function or a `Symbol`) throws `TypeError` here, where
 * `JSON.parse(JSON.stringify(x))` would throw a `SyntaxError` from parsing
 * `"undefined"`. Both throw; this one just says why.
 */
export function cloneJsonValue<T>(value: T): T {
  const cloned = convertToJsonValue(value, '', [])
  if (cloned === OMIT) {
    throw new TypeError(
      'cloneJsonValue: the value has no JSON representation (undefined, function or symbol)'
    )
  }
  return cloned as T
}

/** Internal marker: `JSON.stringify` omits this value entirely. */
const OMIT = Symbol('cloneJsonValue.omit')

function convertToJsonValue(
  value: unknown,
  key: string,
  ancestors: object[]
): unknown {
  // `JSON.stringify` runs `toJSON` before any other branch — this is what turns
  // a `Date` into its ISO string.
  if (
    value !== null &&
    (typeof value === 'object' || typeof value === 'bigint')
  ) {
    const { toJSON } = value as { toJSON?: unknown }
    if (typeof toJSON === 'function') {
      value = (toJSON as (k: string) => unknown).call(value, key)
    }
  }

  // `JSON.stringify` unwraps boxed primitives (`new Number(5)` serializes as 5).
  if (
    value !== null &&
    (value instanceof Number ||
      value instanceof String ||
      value instanceof Boolean)
  ) {
    value = value.valueOf()
  }

  if (value === null) return null

  switch (typeof value) {
    case 'string':
    case 'boolean':
      return value
    case 'number':
      if (!Number.isFinite(value)) return null
      return Object.is(value, -0) ? 0 : value
    case 'bigint':
      throw new TypeError('Do not know how to serialize a BigInt')
    case 'undefined':
    case 'function':
    case 'symbol':
      return OMIT
    default:
      break
  }

  const container = value as object
  if (ancestors.includes(container)) {
    throw new TypeError('Converting circular structure to JSON')
  }

  ancestors.push(container)
  try {
    if (Array.isArray(container)) {
      const result: unknown[] = []
      for (let index = 0; index < container.length; index++) {
        const item = convertToJsonValue(
          container[index],
          String(index),
          ancestors
        )
        result.push(item === OMIT ? null : item)
      }
      return result
    }

    const source = container as Record<string, unknown>
    const result: Record<string, unknown> = {}
    for (const ownKey of Object.keys(source)) {
      const item = convertToJsonValue(source[ownKey], ownKey, ancestors)
      if (item !== OMIT) result[ownKey] = item
    }
    return result
  } finally {
    ancestors.pop()
  }
}
