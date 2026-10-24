# evidence-frontend-reset：重置确认框与成功提示写清范围（含七语言）

- 任务板：`task-27`（Trellis 任务 `10-24-diagnose-deepseek-pricing-row-mismatch`，方案 C）
- 执行者：`worker-docs`；基线 HEAD `43bc237`（2026-10-24）
- 写作用域内改动：`ratio-settings-card.tsx`、`models/__tests__/`（新增 1 个测试文件）、7 个 locale 文件
- 未动 Go 文件（后端由 `worker-script` 负责）；未提交、未推送

## 一、结论

「重置」的两处文案已改成与实际后端行为一致：

- **确认框正文**列出重置范围（模型倍率、补全倍率、缓存倍率、缓存写入倍率、按次价格、图片与音频倍率，
  以及**计费模式与计费表达式**），并明确**分组倍率与用户可用分组不受影响**；
- **成功提示**与正文口径一致（不再说含混的「模型价格重置成功」）。

七语言已补齐（en / zh / zh-TW / fr / ru / ja / vi）；门禁全绿：`typecheck` 0、`i18n:check` 0、
oxlint 0/0、`system-settings` 测试 **85 pass / 0 fail**（含新增用例）。locale diff 为每文件 **+2 行**（不重排）。

## 二、改动清单

| 文件 | 改动 |
|---|---|
| `web/src/features/system-settings/models/ratio-settings-card.tsx` | ① `:172` 成功提示改用新 key `Model pricing was restored to the default templates. Group ratios were not changed.`；② `:517` 确认框 `desc` 改用新 key（范围 + 分组倍率不受影响）。标题 `Reset all model prices?` 保持不动（语义仍准确，避免无谓 key 变动） |
| `web/src/i18n/locales/{en,zh,zh-TW,fr,ru,ja,vi}.json` | 各 +2 行（两个新 key，按既有顺序插入） |
| `web/src/features/system-settings/models/__tests__/ratio-settings-card-reset.test.tsx` | 新增 1 个用例：点击「Reset prices」后确认框必须说明完整范围，且点击前不得出现该文案 |

未使用的旧 key（保留在 locale 文件中，未删）：`Model prices reset successfully`、
`This will clear custom pricing ratios and revert to upstream defaults.`——两键在 `src/` 内引用数已为 0，
属可选清理项（本轮不动 locale 之外的东西，避免扩大改动面）。

## 三、七语言对照

| 位置 | 语言 | 文案 |
|---|---|---|
| 确认框 | en | This restores model pricing to the default templates: model, completion, cache and cache-write ratios, per-call prices, image and audio ratios, and the billing mode with its billing expression. Group ratios and usable groups are not affected. |
| 确认框 | zh | 将模型定价恢复为默认模板：模型倍率、补全倍率、缓存倍率、缓存写入倍率、按次价格、图片倍率与音频倍率，以及计费模式与计费表达式。分组倍率与用户可用分组不受影响。 |
| 确认框 | zh-TW | 將模型定價恢復為預設範本：模型倍率、補全倍率、緩存倍率、緩存寫入倍率、按次價格、圖片倍率與音頻倍率，以及收費模式與收費表達式。分組倍率與使用者可用分組不受影響。 |
| 确认框 | fr | Cela restaure la tarification des modèles selon les modèles par défaut : ratios modèle, complétion, cache et écriture de cache, prix par appel, ratios image et audio, ainsi que le mode de facturation et son expression. Les ratios de groupe et les groupes utilisables ne sont pas affectés. |
| 确认框 | ru | Тарифы моделей восстанавливаются по шаблонам по умолчанию: коэффициенты модели, завершения, кэша и записи кэша, цены за вызов, коэффициенты изображений и аудио, а также режим биллинга и его выражение. Групповые коэффициенты и доступные группы не затрагиваются. |
| 确认框 | ja | モデル料金を既定のテンプレートに戻します：モデル・補完・キャッシュ・キャッシュ書き込みの各倍率、呼び出しごとの価格、画像・音声の倍率、および課金モードと課金式が対象です。グループ倍率と利用可能なグループは変更されません。 |
| 确认框 | vi | Thao tác này khôi phục giá mô hình về mẫu mặc định: tỷ lệ mô hình, hoàn thành, bộ nhớ đệm và ghi bộ nhớ đệm, giá mỗi lần gọi, tỷ lệ hình ảnh và âm thanh, cùng chế độ thanh toán và biểu thức tính phí. Tỷ lệ nhóm và các nhóm khả dụng không bị ảnh hưởng. |
| 成功提示 | en | Model pricing was restored to the default templates. Group ratios were not changed. |
| 成功提示 | zh | 模型定价已恢复为默认模板。分组倍率未改动。 |
| 成功提示 | zh-TW | 模型定價已恢復為預設範本。分組倍率未改動。 |
| 成功提示 | fr | La tarification des modèles a été restaurée selon les modèles par défaut. Les ratios de groupe n'ont pas été modifiés. |
| 成功提示 | ru | Тарифы моделей восстановлены по шаблонам по умолчанию. Групповые коэффициенты не изменялись. |
| 成功提示 | ja | モデル料金を既定のテンプレートに戻しました。グループ倍率は変更されていません。 |
| 成功提示 | vi | Giá mô hình đã được khôi phục về mẫu mặc định. Tỷ lệ nhóm không thay đổi. |

术语依据（取自既有译文）：`Model ratio` 模型倍率 / Ratio modèle / Коэффициент модели / モデル倍率 / Tỷ lệ mô hình；
`Completion ratio` 补全倍率 / Ratio de complétion / Коэффициент завершения / 補完倍率 / Tỷ lệ hoàn thành；
`Cache ratio` 缓存倍率（zh-TW 緩存倍率）/ Tỷ lệ bộ nhớ đệm；`Cache write price` 缓存写入价格；
`Image ratio` 图片倍率 / Ratio d'image；`Audio ratio` 音频倍率 / Ratio audio / 音声倍率 / Tỷ lệ âm thanh；
`Billing Mode` 计费模式（zh-TW 收費模式）/ Mode de facturation / Режим биллинга / 課金モード / Chế độ thanh toán；
`Special billing expression` 计费表达式族；`Group Ratio` 分组倍率 / Ratio de groupe / Групповой коэффициент / グループ倍率 / Tỷ lệ nhóm。

## 四、i18n 流程（沿用上一轮教训）

1. 写入只走临时 `web/scripts/add-missing-keys.mjs` + `bun run i18n:sync`（禁止手改 locale JSON）；
   临时脚本已在完成后删除。
2. 脚本**只插入新键、不重排既有键**（按 `localeCompare` 找首个更大者定位插入点），
   因此 diff 为每个语言文件 **+2 行**，没有上一轮那种 2000+ 行重排。
3. `bun run i18n:sync` 已跑：混淆键 `footer.new\u0061pi.projectAttributionSuffix` 保持转义形式（校验命中 1 处），
   sync 报告 7 个语言的 missing / extras / untranslated 均为 0。

## 五、测试（新用例）

`__tests__/ratio-settings-card-reset.test.tsx`（happy-dom + react-dom/client + QueryClientProvider + 真实 `en.json`）：

- 渲染 `RatioSettingsCard`（`visibleTabs={['models']}`，model/group 默认值给最小合法 JSON）；
- 断言 ① 重置按钮存在；② **点击前**页面不含该范围文案；③ 点击后确认框文本包含完整范围文案
  （含 `billing mode with its billing expression`）；④ 含 `Group ratios and usable groups are not affected.`。

运行结果：该文件 `1 pass / 0 fail`；`src/features/system-settings/` 全量 **85 pass / 0 fail**（9 文件）。

## 六、门禁输出

| 命令 | 结果 |
|---|---|
| `cd web && bun run typecheck` | `TYPECHECK_EXIT=0` |
| `cd web && bun run i18n:check` | `I18N_EXIT=0`；`4036 literal t() keys + 455 static keys = 4187 referenced keys across 7 locales. All referenced keys exist in every locale.` |
| `bunx oxlint -c .oxlintrc.json`（组件 + 新测试） | `Found 0 warnings and 0 errors.` |
| `bun test src/features/system-settings/` | `85 pass / 0 fail` |
| `git diff --stat -- web/` | 8 个已跟踪文件（组件 + 7 locale）共 +20/-2；另有 1 个新测试文件 |

## 七、风险与未完成

- 文案只说「与后端扩展后的行为一致」，前端未按后端字段逐一核对；若 `rest_model_ratio` 实际范围与本卡片文案有差异，
  以后端为准再改文案（本轮按任务书给定的范围书写）。
- 旧的两个 key 已无引用但仍在 7 个语言文件中（各 2 行）；建议后续随 i18n 清理一并删除（本轮未动）。
- 标题仍是 `Reset all model prices?`，未改；如希望标题也体现「含计费表达式」，需再加一个新 key。
- 未做端到端点击（真实调用后端接口）验证；本轮覆盖的是"用户看到什么"。

## 八、建议下一步

1. 与 `worker-script` 的后端改动合并后，跑一次真实的「重置 → 看面板」回归，确认倍率、计费模式与表达式都回到默认、分组倍率不变。
2. 若确认无误，可把两个旧 key 从 7 个语言文件清理掉（需一次受控的 locale 改动）。
3. 若产品希望标题也点明"含计费表达式"，我给标题补一个新 key 并同步七语言。
