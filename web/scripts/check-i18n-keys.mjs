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
/*
 * i18n key check — verifies that every key the app can actually request at
 * runtime exists in every locale file.
 *
 * "Referenced" keys come from two sources:
 *   A. literal `t('...')` call sites in src/**  (what the regex can see)
 *   B. `STATIC_I18N_KEYS` in src/i18n/static-keys.ts — dynamic labels
 *      (constants/configs) that are passed to `t()` at runtime and therefore
 *      never appear as a `t('literal')` call site.
 *
 * `scripts/sync-i18n.mjs` only aligns locale files with each other; it never
 * reads source code, so without this check a key that is deleted from a locale
 * file — or used in source but never translated — goes unnoticed.
 *
 * Run from the web/ package root: `bun run i18n:check`
 * (uses bun so the .ts registry can be imported directly, no text parsing).
 */
import fs from 'node:fs/promises'
import path from 'node:path'

import { STATIC_I18N_KEYS } from '../src/i18n/static-keys.ts'

const LOCALES_DIR = path.resolve('src/i18n/locales')
const SRC_DIR = path.resolve('src')
const REFERENCE_LOCALE = 'en'

/** Literal `t('key')` / `i18next.t('key')`, single- or multi-line. */
const T_CALL_PATTERNS = [
  /\bt\(\s*'([^'\n]+?)'\s*[,)]/g,
  /\bt\(\s*"([^"\n]+?)"\s*[,)]/g,
  /\bt\(\s*`([^`\n]+?)`\s*[,)]/g,
]

/**
 * Strip comments so commented-out `t('...')` snippets are not treated as real
 * call sites. The `[^:]` guard keeps `https://…` inside string literals intact.
 */
function stripComments(code) {
  return code
    .replaceAll(/\/\*[\s\S]*?\*\//g, ' ')
    .replaceAll(/(^|[^:])\/\/.*$/gm, '$1')
}

async function collectSourceFiles(dir) {
  const files = []
  for (const entry of await fs.readdir(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name)
    if (entry.isDirectory()) {
      if (entry.name === 'node_modules' || entry.name === 'locales') continue
      files.push(...(await collectSourceFiles(full)))
    } else if (/\.tsx?$/.test(entry.name)) {
      files.push(full)
    }
  }
  return files
}

async function main() {
  // ── Build the referenced-key set ─────────────────────────────────────────
  const literalKeys = new Set()
  for (const file of await collectSourceFiles(SRC_DIR)) {
    const code = stripComments(await fs.readFile(file, 'utf8'))
    for (const pattern of T_CALL_PATTERNS) {
      pattern.lastIndex = 0
      let match
      while ((match = pattern.exec(code)) !== null) {
        const key = match[1]
        // Template interpolation is not a literal key (e.g. `preset.${v}`).
        if (key === '' || key.includes('${')) continue
        literalKeys.add(key)
      }
    }
  }

  const staticKeys = new Set(STATIC_I18N_KEYS)
  const referenced = new Set([...literalKeys, ...staticKeys])

  // ── Load locales ─────────────────────────────────────────────────────────
  const localeNames = (await fs.readdir(LOCALES_DIR, { withFileTypes: true }))
    .filter((e) => e.isFile() && e.name.endsWith('.json'))
    .map((e) => e.name.replace(/\.json$/i, ''))
    .sort((a, b) => a.localeCompare(b))

  if (!localeNames.includes(REFERENCE_LOCALE)) {
    throw new Error(`Reference locale ${REFERENCE_LOCALE}.json not found.`)
  }

  const translationsByLocale = {}
  for (const locale of localeNames) {
    const raw = await fs.readFile(
      path.join(LOCALES_DIR, `${locale}.json`),
      'utf8'
    )
    translationsByLocale[locale] = JSON.parse(raw)?.translation ?? {}
  }

  // ── Verify every referenced key exists in every locale ───────────────────
  const missingByLocale = {}
  let totalMissing = 0
  for (const locale of localeNames) {
    const translation = translationsByLocale[locale]
    const missing = [...referenced].filter(
      (key) => !Object.hasOwn(translation, key)
    )
    if (missing.length > 0) {
      missingByLocale[locale] = missing.sort((a, b) => a.localeCompare(b))
      totalMissing += missing.length
    }
  }

  console.log(
    `i18n check: ${literalKeys.size} literal t() keys + ${staticKeys.size} static keys ` +
      `= ${referenced.size} referenced keys across ${localeNames.length} locales.`
  )

  if (totalMissing === 0) {
    console.log('All referenced keys exist in every locale.')
    return
  }

  console.error(`\nMissing translations (${totalMissing} total):`)
  for (const [locale, missing] of Object.entries(missingByLocale)) {
    console.error(`\n  ${locale} (${missing.length}):`)
    for (const key of missing) console.error(`    ${JSON.stringify(key)}`)
  }
  console.error(
    '\nFix via scripts/add-missing-keys.mjs + `bun run i18n:sync`; ' +
      'never edit locale JSON by hand.'
  )
  process.exitCode = 1
}

main().catch((err) => {
  console.error(err)
  process.exitCode = 1
})
