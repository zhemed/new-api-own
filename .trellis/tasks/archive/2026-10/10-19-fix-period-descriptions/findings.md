# 修正排行榜周期描述文案 — 交付

## 一、新旧键对照（5 删 5 增，键数净变化 0）

### `models-section.tsx`（按你给的定义）
| 档 | 旧键 | 新键 |
|---|---|---|
| week | `Weekly token usage by model across the past few weeks` | `Daily token usage by model across the past 7 days` |
| year | `Weekly token usage by model across the past year` | `Monthly token usage by model across this year` |
| today / month | 不变（`Hourly … last 24 hours` / `Daily … this month`） | — |

### `market-share-section.tsx`（自有一份 "Token share by model author" 文案，同样失真）
| 档 | 旧键 | 新键 |
|---|---|---|
| week | `Token share by model author across the past few weeks` | `Daily token share by model author across the past 7 days` |
| month | `Token share by model author across the past month` | `Daily token share by model author across this month`（已改自然月） |
| year | `Token share by model author across the past year` | `Monthly token share by model author across this year` |
| today | 不变 | — |

> ⚠️ 比你列的多了 market-share 的 3 个：它那份文案不是在 `models-section` 里、
> 但同样写着 "past few weeks / past month / past year"，不一起改就会继续骗人。
> 净变化仍为 **0**（5 删 5 增）。

## 二、7 语言译文（键始终是英文源串，只有值本地化）

| 键 | zh | zh-TW | fr | ja | ru | vi |
|---|---|---|---|---|---|---|
| Daily token usage by model across the past 7 days | 最近 7 天按模型统计的每日 Token 用量 | 最近 7 天依模型統計的每日 Token 用量 | Utilisation quotidienne de tokens par modèle sur les 7 derniers jours | 過去 7 日間のモデル別の日次トークン使用量 | Ежедневное использование токенов по моделям за последние 7 дней | Mức dùng token hằng ngày theo mô hình trong 7 ngày qua |
| Monthly token usage by model across this year | 今年按模型统计的每月 Token 用量 | 今年依模型統計的每月 Token 用量 | Utilisation mensuelle de tokens par modèle sur cette année | 今年のモデル別の月次トークン使用量 | Ежемесячное использование токенов по моделям за этот год | Mức dùng token hằng tháng theo mô hình trong năm nay |
| Daily token share by model author across the past 7 days | 最近 7 天按模型作者统计的每日 Token 占比 | 最近 7 天依模型作者統計的每日 Token 佔比 | Part quotidienne de tokens par auteur de modèle sur les 7 derniers jours | 過去 7 日間のモデル提供元別の日次トークン比率 | Ежедневная доля токенов по авторам моделей за последние 7 дней | Tỷ lệ token hằng ngày theo tác giả mô hình trong 7 ngày qua |
| Daily token share by model author across this month | 本月按模型作者统计的每日 Token 占比 | 本月依模型作者統計的每日 Token 佔比 | Part quotidienne de tokens par auteur de modèle sur ce mois-ci | 今月のモデル提供元別の日次トークン比率 | Ежедневная доля токенов по авторам моделей за этот месяц | Tỷ lệ token hằng ngày theo tác giả mô hình trong tháng này |
| Monthly token share by model author across this year | 今年按模型作者统计的每月 Token 占比 | 今年依模型作者統計的每月 Token 佔比 | Part mensuelle de tokens par auteur de modèle sur cette année | 今年のモデル提供元別の月次トークン比率 | Ежемесячная доля токенов по авторам моделей за этот год | Tỷ lệ token hằng tháng theo tác giả mô hình trong năm nay |

全部 6 个非英文语言**已确认无英文残留**（逐个比对 value ≠ key）。

## 三、i18n 校验
```
i18n check: 4019 literal t() keys + 455 static keys = 4170 referenced keys across 7 locales.
All referenced keys exist in every locale.
```
键数与上一轮**完全一致（4170）** → 净变化 0 ✓；旧键已从 7 个 locale 全部移除。

**过程中的一次自我纠错**：第一版脚本把「本地化后的文案」当成了 JSON **键**（只有 en 才对），
`sync-i18n` 随即将英文源串作为兜底键补回、并删掉那些中文键 → 表现为"键在但值是英文"。
已重写脚本（键恒为英文源串、只本地化值）并逐语言复核。

## 四、七条门
typecheck **0 错误** ｜ lint **0 warnings / 0 errors** ｜ `bun test` **366 pass / 0 fail**（与基线一致）｜
format:check exit 0 ｜ build exit 0 ｜ **knip exit 0** ｜ i18n:check exit 0。

## 五、边界
只改两个组件 + 7 个 locale。未 commit。
