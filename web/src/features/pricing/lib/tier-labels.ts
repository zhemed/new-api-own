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
 * Display labels for the tier names used by the built-in billing expression
 * templates (`tier("peak", …)` / `tier("offpeak", …)`).
 *
 * The tier name is user data inside the expression — an operator may write any
 * label they like — so only the rendering layer translates here: an unmapped
 * label is shown exactly as written, and matching/highlighting keeps comparing
 * the raw labels (see `normalizeTierLabel` in `lib/billing-expr.ts`).
 */
export const TIER_LABEL_I18N_KEYS: Record<string, string> = {
  peak: 'Peak hours',
  offpeak: 'Off-peak hours',
  base: 'Base',
  standard: 'Standard',
  short: 'Short',
  mid: 'Mid',
  long: 'Long',
  short_output: 'Short output',
  long_output: 'Long output',
  mid_context: 'Mid context',
  long_context: 'Long context',
  discount: 'Discount',
}

/** `Short Output` / `short-output` all resolve to the `short_output` entry. */
function tierLabelLookupKey(label: string): string {
  return label.trim().toLowerCase().replaceAll(/\s+/g, '_')
}

/**
 * Translate a tier label for display.
 *
 * - a known template label renders in the active language;
 * - an unknown label is returned unchanged (never dropped, never blanked);
 * - an empty/absent label falls back to the translated `Default`.
 */
export function getTierDisplayLabel(
  label: string | null | undefined,
  t: (key: string) => string
): string {
  if (label == null) return t('Default')
  const trimmed = label.trim()
  if (trimmed === '') return t('Default')
  const i18nKey = TIER_LABEL_I18N_KEYS[tierLabelLookupKey(trimmed)]
  return i18nKey ? t(i18nKey) : label
}
