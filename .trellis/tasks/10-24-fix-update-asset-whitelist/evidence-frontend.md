# evidence-frontend：前端错误码 `signature_asset_missing` 映射 + 七语言专用文案（C2）

- 任务板：`task-20`（Trellis 任务 `10-24-fix-update-asset-whitelist`）
- 执行者：`worker-docs`；基线 HEAD `e6ec2d8`（2026-10-24）
- 未动 Go 文件（后端由 `worker-script` 负责）；未提交、未推送、未访问外部主机

## 一、结论

1. 前端已把「Release 缺少签名资产」与「签名不匹配」区分开：新错误码 `signature_asset_missing`
   映射到**独立**失败原因与新文案，不再落到 `signature_mismatch` 分支。
2. 七语言（en / zh / zh-TW / fr / ru / ja / vi）均已补齐新 key，英文源串为任务指定文案；
   `bun run i18n:check` 报 4036 个 t() 字面量键在 7 个语言文件中全部存在；
   `i18n:sync` 报告每个语言的 missing / extras / untranslated 均为 `0`。
3. 门禁：`bun run typecheck` 退出码 0；oxlint 对 4 个改动文件 0 warning / 0 error；
   `bun test`（maintenance 目录）**70 pass / 0 fail**；`bun run build` 退出码 0（额外跑的）。
4. 全程 diff 为 **33 行纯新增、0 删除**，分散在 11 个文件（2 源码 + 2 测试 + 7 locale）。

## 二、改动清单

| 文件 | 改动 |
|---|---|
| `web/src/features/system-settings/maintenance/update-api.ts` | `UpdateFailureReason` 联合类型新增 `signature_asset_missing`（含 JSDoc）；`FAILURE_CODES` 新增 `signature_asset_missing: 'signature_asset_missing'`（放在 signature 组末尾） |
| `web/src/features/system-settings/maintenance/update-checker-section.tsx` | 在 `signature_mismatch` 分支后新增 `signature_asset_missing` 展示分支，文案取新 key；其它分支未改 |
| `web/src/i18n/locales/{en,zh,zh-TW,fr,ru,ja,vi}.json` | 各新增 1 行新 key（按既有顺序插入，未重排任何既有键） |
| `web/src/features/system-settings/maintenance/__tests__/update-api.test.ts` | 既有 code→reason 表新增 1 行 `['signature_asset_missing', 'signature_asset_missing']` |
| `web/src/features/system-settings/maintenance/__tests__/update-checker-section.test.tsx` | 新增 1 个用例「explains a missing signature asset instead of a signature mismatch」（镜像既有 mismatch 用例，断言 toast 文案为新 key 的 en 值） |

测试是否超出写作用域：任务列出的写作用域只含 2 个源码文件与 locales，但 `web/AGENTS.md` 3.14 要求
「修改现有行为必须同步新增或更新测试」，且这两个测试文件**本就覆盖**同一映射表与同一展示分支
（`update-api.test.ts:203` 的 code→reason 表、`update-checker-section.test.tsx:542` 的 mismatch 用例）。
本次只做**增量**（+1 行表项、+18 行用例），未改动既有断言；如不需要可单独回退这两个文件。

放弃的替代实现：没有把新码并入 `signature_mismatch` 映射（那样前端仍会显示"签名被篡改"文案，
与后端拆分该错误码的意图相悖）；也没有引入第三个 UI 文案分支到 `rollbackUpdate` 路径
（回滚接口不产生该错误码，后端只在 apply 路径返回它）。

## 三、七语言对照

| 语言 | 文案 |
|---|---|
| en | The release is missing its signature asset, so the replacement was refused. |
| zh | 发布版本缺少签名资产，因此已拒绝替换。 |
| zh-TW | 發布版本缺少簽章資產，因此已拒絕替換。 |
| fr | L'actif de signature est absent de la publication ; le remplacement a été refusé. |
| ja | リリースに署名アセットが含まれていないため、置き換えを拒否しました。 |
| ru | В релизе отсутствует файл подписи, поэтому замена отклонена. |
| vi | Bản phát hành thiếu tệp chữ ký nên đã từ chối thay thế. |

译法依据：同族 key「The update signature did not match the published checksum, so the replacement was refused.」
在各语言的既有译法（zh「…因此已拒绝替换。」、zh-TW「…因此已拒絕替換。」、fr 分号句式、
ja「…ため、置き換えを拒否しました。」、ru「…замена отклонена.」、vi「…nên đã từ chối thay thế.」），
新文案沿用同一句式，只把「签名不匹配」换成「缺少签名资产」；术语取 zh「签名资产」、zh-TW「簽章資產」、
fr「actif de signature」、ja「署名アセット」、ru「файл подписи」、vi「tệp chữ ký」。

## 四、i18n 流程与一次返工（重要，供后续会话复用）

- 按项目规则（`web/AGENTS.md` 3.1 与 i18n-translate 技能），locale 写入**只能**走
  临时 `web/scripts/add-missing-keys.mjs` + `bun run i18n:sync`，禁止手改 JSON。
  临时脚本已按要求在完成后删除（`web/scripts/` 现只剩 4 个常驻脚本）。
- **返工点**：技能模板里的脚本会 `localeCompare` 全量重排序；本仓库的语言文件当前顺序**并非**
  该 collation 的结果，照抄模板会产出 **每个文件约 2235 行的重排 diff**（已实际发生）。
  处置：`git checkout -- web/src/i18n/locales/` 全量回退，改为"**只插入新键、不重排既有键**"的写法
  （新键插入位置＝对既有键序列做 `localeCompare` 找首个更大者），随后 `bun run i18n:sync`
  只做 base 顺序对齐 → 最终 diff 为 7 个文件各 **+1 行**。
- **另一个坑**：脚本 `JSON.parse`/`stringify` 往返会把混淆键
  `footer.new\u0061pi.projectAttributionSuffix` 解码成明文 `footer.newapi...`。
  `bun run i18n:sync` 会按 `OBFUSCATED_KEYS` 重新写回转义形式，因此**必须**跑 sync（残留检查已确认：
  转义形式存在 1 处、明文形式 0 处）。
- 建议：技能模板与 `web/AGENTS.md` 的示例脚本应补一句"插入而非重排"的告警（本轮未改技能文件，超写作用域）。

## 五、门禁输出

| 命令 | 结果 |
|---|---|
| `cd web && bun run typecheck` | `TYPECHECK_EXIT=0` |
| `cd web && bun run i18n:check` | `I18N_EXIT=0`；`4036 literal t() keys + 455 static keys = 4187 referenced keys across 7 locales. All referenced keys exist in every locale.` |
| `cd web && bunx oxlint -c .oxlintrc.json`（4 个改动文件） | `Found 0 warnings and 0 errors.`（2 files 后再跑 4 files，均 0） |
| `cd web && bun test src/features/system-settings/maintenance/__tests__/` | `70 pass / 0 fail`（基线 69 pass；新增用例后 70），含 `(pass) … explains a missing signature asset instead of a signature mismatch` |
| `cd web && bun run build` | `BUILD_EXIT=0`（Total 57364.0 kB / gzip 16552.5 kB） |
| `bun run i18n:sync` 报告 | 7 个语言 missing / extras / untranslated 全为 `0` |
| `git diff --stat -- web/` | 11 文件、33 行新增、0 删除（locale 各 +1 行） |

## 六、风险与未完成

- 前端只做了「新码 → 新文案」；**旧码** `signature_mismatch` 的映射与文案保持不变
  （若后端改用新码表示资产缺失，旧码仍表示签名不匹配，符合预期）。
- 展示分支只加在 apply（升级）路径：`rollbackUpdate` 目前不返回该码；若后端后续在回滚路径也返回，
  需要同步补分支（届时另开改动）。
- 未做端到端点击验证（需要真实 502 响应）；文案正确性由单测（toast 文案断言）与 i18n 检查覆盖。
- 任务书未列出测试文件，本次为满足 `web/AGENTS.md` 3.14 做了增量测试，是否保留由 Lead 决定。

## 七、建议下一步

1. 与 `worker-script` 的后端改动合并验证：确认后端 `signature_asset_missing` 的 code 字符串与
   HTTP 502 语义一致（前端只按 `code` 字符串映射，大小写敏感）。
2. 真实环境回归一次「Release 缺 `SHA256SUMS.sig`」场景，确认 toast 显示新文案而非通用失败文案。
3. 建议把「locale 只插入不重排 + 必须跑 i18n:sync 恢复混淆键」写进 i18n 技能或 `web/AGENTS.md`。
