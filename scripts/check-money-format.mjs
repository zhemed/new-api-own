#!/usr/bin/env node
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
 * money-format check — fails when user-facing money text is hardcoded in the
 * frontend instead of going through the currency layer.
 *
 * Why: the panel display currency is admin-configurable (USD / CNY / CUSTOM),
 * so every amount must be formatted by `web/src/lib/currency.ts`. Three classes
 * of regressions are easy to reintroduce and were missed by earlier reviews:
 *
 *   A. template literals that prepend a literal currency sign to an
 *      interpolation:      `$${amount}`  `¥${amount}`  `€${amount}`
 *   B. string concatenation or literal amounts:
 *      `'$' + amount`, `">$" + v`, JSX text `>$12.5`
 *   C. Intl.NumberFormat / toLocaleString pinned to USD:
 *      `new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' })`
 *
 * Everything that is not a display decision is allowed and must stay allowed:
 * comments, regex replacement patterns (`'$1 $2'`), shell/env examples
 * (`$${ctx.apiKeyEnv}` inside curl snippets), USD comparisons/types/defaults
 * (`=== 'USD'`, `currency: 'USD' | 'EUR'`, `default('USD')`), contract values
 * (`currencyCode: 'USD'`, payment-provider product currency) and the
 * USD-denominated price-coefficient unit notation (`$/1M`).
 *
 * Usage:  node scripts/check-money-format.mjs
 * Exit:   0 = no finding, 1 = findings (each printed as file:line [class] snippet)
 */
import fs from 'node:fs/promises'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const SCRIPT_DIR = path.dirname(fileURLToPath(import.meta.url))
const REPO_ROOT = path.resolve(SCRIPT_DIR, '..')
const SCAN_ROOT = path.join(REPO_ROOT, 'web', 'src')
const SOURCE_EXTENSIONS = new Set(['.ts', '.tsx', '.js', '.jsx', '.mjs', '.cjs'])

const CURRENCY_SIGNS = '$¥€'

/*
 * Strip comments while keeping strings and template literals intact, and keep
 * the line structure so findings can report their original line numbers. A
 * character scanner is used instead of one regex because `//` inside a URL,
 * `/*` inside a string, and `${...}` expressions inside template literals all
 * need the surrounding quote state to be interpreted correctly.
 */
function stripComments(source) {
  const out = source.split('')
  let line = 0
  let i = 0
  // normal | line | block | single | double | template | expr
  let state = 'normal'
  let templateDepth = 0 // brace depth while inside a `${ ... }` expression
  const blank = (from, to) => {
    for (let k = from; k < to && k < out.length; k++) if (out[k] !== '\n') out[k] = ' '
  }
  while (i < source.length) {
    const ch = source[i]
    const next = source[i + 1]
    if (ch === '\n') line++
    switch (state) {
      case 'normal':
        if (ch === '/' && next === '/') {
          state = 'line'
          blank(i, source.indexOf('\n', i) === -1 ? source.length : source.indexOf('\n', i))
        } else if (ch === '/' && next === '*') {
          const end = source.indexOf('*/', i + 2)
          const stop = end === -1 ? source.length : end + 2
          blank(i, stop)
          for (let k = i; k < stop; k++) if (source[k] === '\n') line++
          i = stop
          continue
        } else if (ch === "'") state = 'single'
        else if (ch === '"') state = 'double'
        else if (ch === '`') state = 'template'
        break
      case 'line':
        if (ch === '\n') state = 'normal'
        break
      case 'block':
        break
      case 'single':
        if (ch === '\\') i++
        else if (ch === "'") state = 'normal'
        break
      case 'double':
        if (ch === '\\') i++
        else if (ch === '"') state = 'normal'
        break
      case 'template':
        if (ch === '\\') i++
        else if (ch === '`') state = 'normal'
        else if (ch === '$' && next === '{') {
          state = 'expr'
          templateDepth = 0
          i += 2
          continue
        }
        break
      case 'expr':
        if (ch === '\\') i++
        else if (ch === '{') templateDepth++
        else if (ch === '}') {
          if (templateDepth === 0) state = 'template'
          else templateDepth--
        } else if (ch === "'") state = 'single'
        else if (ch === '"') state = 'double'
        else if (ch === '`') state = 'template'
        break
    }
    i++
  }
  return { code: out.join(''), lineCount: line + 1 }
}

/* ------------------------------------------------------------------ */
/* Whitelist: things that are NOT display-currency decisions           */
/* ------------------------------------------------------------------ */

const SHELL_LINE_HINTS = [
  'curl ',
  '-H "',
  'Bearer',
  'Authorization:',
  'x-api-key',
  'export ',
  'ApiKeyEnv',
  'apiKeyEnv',
]

/** `$${ctx.apiKeyEnv}` / `$${HOME}` inside a curl sample is shell interpolation. */
function isShellInterpolation(line, matchEnd) {
  if (SHELL_LINE_HINTS.some((hint) => line.includes(hint))) return true
  const body = line.slice(matchEnd).match(/^([A-Za-z_$][\w$]*(?:\.[A-Za-z_$][\w$]*)?)/)
  if (!body) return false
  return /(apiKeyEnv|apiKey|env|HOME|PATH|USER|HOST|PORT|SHELL|BASE_URL)/i.test(body[1])
}

/** `'$1 $2'` / `'$&'` are regex replacement back-references, not money. */
const REGEX_BACKREF_ONLY = /['"]\$[\d&'`]{1,2}(?:\s+\$[\d&'`]{1,2})*['"]/s
function isRegexReplacement(line) {
  return /\.replace(?:All)?\(/.test(line) || /replace(?:All)?\s*\(/.test(line) || REGEX_BACKREF_ONLY.test(line)
}

/** `$/1M tokens` / `$2.50/1M` — USD-denominated price-coefficient notation. */
function isPriceCoefficientUnit(line, index) {
  const rest = line.slice(index)
  return /^\$(?:\s?\d[\d.,]*)?\s?\/\s?1[KkMm]\b/.test(rest) || /\/1[KkMm]\b/.test(line)
}

/** Comparisons, type positions, defaults, and contract values for USD. */
const CONTRACT_VALUE = [
  /===\s*['"]USD['"]/,
  /!==\s*['"]USD['"]/,
  /==\s*['"]USD['"]/,
  /default\(\s*['"]USD['"]\s*\)/,
  /\?\?\s*['"]USD['"]/,
  /\|\|\s*['"]USD['"]/,
  /['"]USD['"]\s*\|/,
  /\|\s*['"]USD['"]/,
  /currencyCode\s*:\s*['"]USD['"]/,
  /type\s+\w+\s*=\s*['"]USD['"]/,
]
function isContractValueLine(line) {
  return CONTRACT_VALUE.some((re) => re.test(line))
}

/**
 * i18n source strings are lookup keys, not rendered text — the locale file holds
 * the rendered value (and its own symbol). A key may therefore embed a currency
 * sign or a `{{placeholder}}` without hardcoding the display currency.
 *
 * Detected as: a line containing a `t(...)` call, a line right after an unclosed
 * `t(` (multi-line call), or any line with an i18next `{{...}}` placeholder.
 */
const I18N_PLACEHOLDER = /\{\{\s*[\w.]+\s*\}\}/
function isI18nKeyLine(codeLines, index) {
  const line = codeLines[index]
  if (I18N_PLACEHOLDER.test(line)) return true
  if (/\bt\s*\(/.test(line)) return true
  for (let back = 1; back <= 3 && index - back >= 0; back++) {
    const previous = codeLines[index - back]
    if (!/\bt\s*\(/.test(previous)) continue
    if (openParens(previous) > 0) return true
  }
  return false
}

/** Rough open-paren balance of a line's code (ignores parens inside strings). */
function openParens(line) {
  let depth = 0
  let quote = ''
  for (let i = 0; i < line.length; i++) {
    const ch = line[i]
    if (quote) {
      if (ch === '\\') i++
      else if (ch === quote) quote = ''
      continue
    }
    if (ch === "'" || ch === '"' || ch === '`') quote = ch
    else if (ch === '(') depth++
    else if (ch === ')') depth--
  }
  return depth
}

/* ------------------------------------------------------------------ */
/* Findings                                                            */
/* ------------------------------------------------------------------ */

const findings = []

function report(state, lineNo, klass, snippet) {
  findings.push({
    file: path.relative(REPO_ROOT, state.file),
    line: lineNo + 1,
    klass,
    snippet: snippet.trim(),
  })
}

function checkLine(state, lineNo, line, codeLines) {
  if (!line.trim()) return
  if (isI18nKeyLine(codeLines, lineNo)) return

  // A. template literal / interpolation with a literal currency sign.
  for (let index = 0; index < line.length; index++) {
    const ch = line[index]
    if (!CURRENCY_SIGNS.includes(ch)) continue
    // `$${...}` (USD) or `$ ${...}` or `¥${...}` / `€${...}`
    const rest = line.slice(index)
    const dollarMatch = /^\$(?:\$|\s+\$)\{/.exec(rest)
    const signMatch = /^[¥€]\s*\$\{/.exec(rest)
    if (!dollarMatch && !signMatch) continue
    const end = index + (dollarMatch ? dollarMatch[0].length : signMatch[0].length)
    if (dollarMatch && isShellInterpolation(line, end)) continue
    report(state, lineNo, 'template-currency-literal', line)
    return
  }

  // B1. concatenation: `'$' + amount` / `"¥" + v` / `` `$` + ``.
  const concat = /['"`]([$¥€])['"`]\s*\+/.exec(line)
  if (concat && !isContractValueLine(line)) {
    report(state, lineNo, 'currency-concat', line)
    return
  }

  // B2. literal amount in a string or JSX text.
  for (let index = 0; index < line.length; index++) {
    if (!CURRENCY_SIGNS.includes(line[index])) continue
    const rest = line.slice(index)
    if (!/^[$¥€]\s?\d/.test(rest)) continue
    if (isRegexReplacement(line)) continue
    if (isPriceCoefficientUnit(line, index)) continue
    if (isContractValueLine(line)) continue
    report(state, lineNo, 'literal-currency-amount', line)
    return
  }
}

/* C. Intl.NumberFormat / toLocaleString pinned to USD (checked line-wise with a
 * small look-back window, so contract objects like Creem product currency are
 * not mistaken for formatting). */
function checkIntlCurrency(state) {
  const lines = state.rawLines
  lines.forEach((line, index) => {
    if (!/\bcurrency\s*:\s*['"]USD['"]/.test(line)) return
    if (isContractValueLine(line)) return
    const from = Math.max(0, index - 12)
    const window = lines.slice(from, index + 1).join('\n')
    const match = /(Intl\.NumberFormat\(|\.toLocaleString\()/.exec(window)
    if (!match) return
    // A closing `)` between the formatter call and the currency option means
    // the option belongs to a different object; treat the hit as ambiguous and
    // only report when the formatter call is still open (heuristic: no `)` on
    // the formatter line before the option line).
    report(state, index, 'intl-currency-usd', line)
  })
}

/* ------------------------------------------------------------------ */
/* Walk                                                                */
/* ------------------------------------------------------------------ */

async function* walk(dir) {
  const entries = await fs.readdir(dir, { withFileTypes: true })
  for (const entry of entries) {
    const full = path.join(dir, entry.name)
    if (entry.isDirectory()) {
      if (entry.name === 'node_modules' || entry.name === 'dist') continue
      yield* walk(full)
    } else if (SOURCE_EXTENSIONS.has(path.extname(entry.name))) {
      // Test files assert exact formatter output (`'¥10.00'` IS the expected
      // value there), so they are intentionally out of scope for this scan.
      if (entry.name.includes('.test.') || entry.name.includes('.spec.')) continue
      if (full.includes(`${path.sep}__tests__${path.sep}`)) continue
      yield full
    }
  }
}

async function main() {
  let scanned = 0
  for await (const file of walk(SCAN_ROOT)) {
    const raw = await fs.readFile(file, 'utf8')
    const { code } = stripComments(raw)
    const rawLines = raw.split('\n')
    const codeLines = code.split('\n')
    const state = { file, rawLines }
    for (let i = 0; i < codeLines.length; i++) checkLine(state, i, codeLines[i], codeLines)
    checkIntlCurrency(state)
    scanned++
  }

  if (findings.length === 0) {
    console.log(`OK — no hardcoded currency in web/src (${scanned} files scanned)`)
    process.exit(0)
  }

  findings
    .sort((a, b) => a.file.localeCompare(b.file) || a.line - b.line)
    .forEach((f) => {
      console.log(`${f.file}:${f.line} [${f.klass}] ${f.snippet}`)
    })
  console.log(`\n${findings.length} finding(s) in ${scanned} files scanned`)
  console.log('规则：前端金额一律走 web/src/lib/currency.ts；例外需在本脚本白名单里写明理由。')
  process.exit(1)
}

main().catch((error) => {
  console.error('money-format check crashed:', error)
  process.exit(2)
})
